package systems

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// InputSystem 处理所有用户输入，包括鼠标点击和键盘输入
type InputSystem struct {
	entityManager      *ecs.EntityManager
	resourceManager    *game.ResourceManager
	gameState          *game.GameState
	reanimSystem       entities.ReanimSystemInterface // Reanim 系统（用于初始化植物动画）
	sunCounterX        float64                        // 阳光计数器X坐标（收集动画目标）
	sunCounterY        float64                        // 阳光计数器Y坐标（收集动画目标）
	lawnGridSystem     *LawnGridSystem                // 草坪网格管理系统
	lawnGridEntityID   ecs.EntityID                   // 草坪网格实体ID
	lastBuzzerPlayTime float64                        // 上次播放无效操作音效的时间 (Story 10.8)
	buzzerCooldownTime float64                        // 无效操作音效冷却时间（秒）(Story 10.8)
	gameTime           float64                        // 游戏时间累计（秒）(Story 10.8)
	tooltipEntity      ecs.EntityID                   // Tooltip 实体ID (Story 10.8)

	// 拖拽种植模式状态（移动端触摸拖拽支持）
	isDragPlanting      bool                 // 是否处于拖拽种植模式
	dragPlantType       components.PlantType // 拖拽中的植物类型
	dragStartCardEntity ecs.EntityID         // 拖拽开始时的卡片实体ID

	// 调试/验证模式选项
	ignoreCooldown bool // 忽略植物卡片冷却（用于 verify_gameplay 等验证工具）
}

// NewInputSystem 创建一个新的输入系统
// 添加 reanimSystem 参数用于初始化植物动画
func NewInputSystem(em *ecs.EntityManager, rm *game.ResourceManager, gs *game.GameState, rs entities.ReanimSystemInterface, sunCounterX, sunCounterY float64, lawnGridSystem *LawnGridSystem, lawnGridEntityID ecs.EntityID) *InputSystem {
	system := &InputSystem{
		entityManager:      em,
		resourceManager:    rm,
		gameState:          gs,
		reanimSystem:       rs,
		sunCounterX:        sunCounterX,
		sunCounterY:        sunCounterY,
		lawnGridSystem:     lawnGridSystem,
		lawnGridEntityID:   lawnGridEntityID,
		buzzerCooldownTime: 0.5, // Story 10.8: 0.5秒冷却时间，防止连续点击播放多次
	}

	// 音效统一由 AudioManager 管理（Story 10.9）

	return system
}

// SetIgnoreCooldown 设置是否忽略植物卡片冷却
// 用于验证工具（如 verify_gameplay）可以随意种植植物
func (s *InputSystem) SetIgnoreCooldown(ignore bool) {
	s.ignoreCooldown = ignore
}

// Update 处理用户输入
// 参数:
//   - deltaTime: 时间增量（秒）
//   - cameraX: 摄像机的世界坐标X位置（用于屏幕坐标到世界坐标的转换）
func (s *InputSystem) Update(deltaTime float64, cameraX float64) {
	// Story 10.8: 更新游戏时间（用于音效冷却）
	s.gameTime += deltaTime

	// 更新拖拽管理器状态（支持移动端拖拽种植）
	dragManager := utils.GetDragManager()
	dragManager.Update()

	// ESC 键切换暂停/恢复
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.gameState.TogglePause()
		if s.gameState.IsPaused {
			log.Printf("[InputSystem] 游戏暂停 (ESC)")
			// 播放暂停音效
			if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
				audioManager.PlaySound("SOUND_PAUSE")
			}
		} else {
			log.Printf("[InputSystem] 游戏恢复 (ESC)")
		}
		return // 处理暂停切换后立即返回，避免响应其他输入
	}

	// 暂停时屏蔽游戏世界交互
	if s.gameState.IsPaused {
		return // 暂停时不处理任何游戏输入
	}

	// 注意：植物预览位置现在由 PlantPreviewSystem 统一管理，无需在这里更新

	// Story 8.2.1: 更新植物卡片悬停状态（每帧检测）
	s.updatePlantCardHover()

	// 更新阳光悬停状态（每帧检测，用于手形光标）
	s.updateSunHover(cameraX)

	// 处理植物选择快捷键（数字键 1-9）
	s.handlePlantCardHotkeys(cameraX)

	// 检测鼠标右键取消种植模式
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		if s.gameState.IsPlantingMode {
			log.Printf("[InputSystem] 右键取消种植模式")
			s.gameState.ExitPlantingMode()
			s.destroyPlantPreview()
			s.cancelDragPlanting()
		}
	}

	// ========================================================================
	// 拖拽种植处理（仅移动端触摸拖拽支持）
	// 桌面端使用传统点击模式，不触发拖拽逻辑
	// ========================================================================
	if dragManager.IsTouchDrag() || s.isDragPlanting {
		// 只有触摸输入或已经处于拖拽种植模式才处理拖拽逻辑
		if s.handleDragPlanting(cameraX) {
			return // 拖拽种植已处理，跳过传统点击逻辑
		}
	}

	// ========================================================================
	// 传统点击处理（桌面端鼠标点击 + 移动端触摸支持）
	// ========================================================================
	// 检测指针是否刚被按下（支持触摸和鼠标）
	justPressed, mouseScreenX, mouseScreenY := utils.IsPointerJustPressed()
	if justPressed {
		// 将鼠标屏幕坐标转换为世界坐标
		// worldX = screenX + cameraX (摄像机向右移动时，世界坐标增大)
		mouseWorldX := float64(mouseScreenX) + cameraX
		mouseWorldY := float64(mouseScreenY) // Y轴不受摄像机水平移动影响

		// DEBUG: 鼠标点击日志（每次点击都打印会刷屏，已禁用）
		// log.Printf("[InputSystem] 鼠标点击: 屏幕(%d, %d) -> 世界(%.1f, %.1f)",
		// 	mouseScreenX, mouseScreenY, mouseWorldX, mouseWorldY)

		// 优先检查阳光点击（阳光可能飘到UI区域，优先级高于植物卡片）
		sunHandled := s.handleSunClickAt(mouseWorldX, mouseWorldY)
		if sunHandled {
			return // 已处理阳光点击，不继续处理其他点击
		}

		// 检查植物卡片点击（UI元素不受摄像机影响，使用屏幕坐标）
		cardHandled := s.handlePlantCardClick(mouseScreenX, mouseScreenY, cameraX)
		if cardHandled {
			return // 已处理卡片点击，不继续处理其他点击
		}

		// 检查是否在种植模式下点击草坪
		// 注意：handleLawnClick 内部会调用 MouseToGridCoords 进行坐标转换，所以传递屏幕坐标
		lawnHandled := s.handleLawnClick(mouseScreenX, mouseScreenY)
		if lawnHandled {
			return // 已处理草坪种植
		}
	}
}
