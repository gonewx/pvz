package modules

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
)

// PauseMenuModule 暂停菜单模块
//
// 职责：
//   - 管理暂停菜单按钮（返回游戏、重新开始、主菜单）
//   - 控制暂停状态（GameState.IsPaused）
//   - 组合 SettingsPanelModule 复用设置 UI
//
// 设计原则：
//   - 组合优于继承：使用 SettingsPanelModule 组合模式
//   - 高内聚：暂停逻辑封装在单一模块中
//   - 低耦合：通过清晰的接口与外部系统交互
//
// 使用场景：
//   - GameScene: 游戏中的暂停菜单（当前）
//   - MiniGameScene: 小游戏中的暂停菜单（未来扩展）
//   - ChallengeScene: 挑战模式中的暂停菜单（未来扩展）
//
// Story 10.1: 暂停菜单系统
// 重构：使用组合模式消除代码重复
type PauseMenuModule struct {
	// ECS 框架
	entityManager *ecs.EntityManager

	// 组合：通用设置面板（复用）
	settingsPanelModule *SettingsPanelModule

	// 系统（引用，不拥有）
	buttonSystem       *systems.ButtonSystem       // 按钮交互
	buttonRenderSystem *systems.ButtonRenderSystem // 按钮渲染

	// 按钮实体列表（用于显示/隐藏控制）
	buttonEntities []ecs.EntityID

	// 外部依赖
	gameState *game.GameState

	// 回调函数（由外部场景提供）
	onContinue    func() // "继续"按钮回调
	onRestart     func() // "重新开始"按钮回调
	onMainMenu    func() // "返回主菜单"按钮回调
	onPauseMusic  func() // 暂停音乐回调
	onResumeMusic func() // 恢复音乐回调
	onSaveBattle  func() // Story 18.2: 保存战斗状态回调

	// 内部状态（用于检测状态变化）
	wasActive bool // 上一帧的激活状态

	// 屏幕尺寸（用于渲染）
	windowWidth  int
	windowHeight int
}

// PauseMenuCallbacks 暂停菜单回调函数集合
type PauseMenuCallbacks struct {
	OnContinue    func() // "继续"按钮回调
	OnRestart     func() // "重新开始"按钮回调
	OnMainMenu    func() // "返回主菜单"按钮回调
	OnPauseMusic  func() // 暂停音乐回调（可选）
	OnResumeMusic func() // 恢复音乐回调（可选）
	OnSaveBattle  func() // Story 18.2: 保存战斗状态回调（返回主菜单时触发）
}

// NewPauseMenuModule 创建暂停菜单模块
//
// 参数:
//   - em: EntityManager 实例
//   - gs: GameState 实例（用于控制暂停状态）
//   - rm: ResourceManager 实例（用于加载资源）
//   - buttonSystem: 按钮交互系统（引用，不拥有）
//   - buttonRenderSystem: 按钮渲染系统（引用，不拥有）
//   - windowWidth, windowHeight: 游戏窗口尺寸
//   - callbacks: 暂停菜单回调函数集合
//
// 返回:
//   - *PauseMenuModule: 新创建的模块实例
//   - error: 如果初始化失败
//
// Story 10.1: 暂停菜单系统
// 重构：使用 SettingsPanelModule 组合模式
func NewPauseMenuModule(
	em *ecs.EntityManager,
	gs *game.GameState,
	rm *game.ResourceManager,
	buttonSystem *systems.ButtonSystem,
	buttonRenderSystem *systems.ButtonRenderSystem,
	windowWidth, windowHeight int,
	callbacks PauseMenuCallbacks,
) (*PauseMenuModule, error) {
	module := &PauseMenuModule{
		entityManager:      em,
		gameState:          gs,
		buttonSystem:       buttonSystem,
		buttonRenderSystem: buttonRenderSystem,
		onContinue:         callbacks.OnContinue,
		onRestart:          callbacks.OnRestart,
		onMainMenu:         callbacks.OnMainMenu,
		onPauseMusic:       callbacks.OnPauseMusic,
		onResumeMusic:      callbacks.OnResumeMusic,
		onSaveBattle:       callbacks.OnSaveBattle, // Story 18.2
		windowWidth:        windowWidth,
		windowHeight:       windowHeight,
		buttonEntities:     make([]ecs.EntityID, 0, 3),
	}

	// 1. 创建通用设置面板模块（组合）
	// PauseMenuModule 不需要底部按钮，因为它自己管理 3 个按钮
	settingsPanelModule, err := NewSettingsPanelModule(
		em,
		rm,
		buttonRenderSystem, // 传递按钮渲染系统（虽然不需要底部按钮，但保持接口统一）
		windowWidth,
		windowHeight,
		SettingsPanelCallbacks{
			OnMusicVolumeChange: func(volume float64) {
				// TODO: 实际控制音乐音量
			},
			OnSoundVolumeChange: func(volume float64) {
				// TODO: 实际控制音效音量
			},
			On3DToggle: func(enabled bool) {
				// TODO: 实际控制3D加速
			},
			OnFullscreenToggle: func(enabled bool) {
				// TODO: 实际切换全屏
			},
		},
		nil, // 不需要底部按钮
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create settings panel module: %w", err)
	}
	module.settingsPanelModule = settingsPanelModule

	// 2. 创建暂停菜单按钮（返回游戏、重新开始、主菜单）
	if err := module.createPauseMenuButtons(rm); err != nil {
		return nil, fmt.Errorf("failed to create pause menu buttons: %w", err)
	}

	log.Printf("[PauseMenuModule] Initialized with %d buttons", len(module.buttonEntities))

	return module, nil
}

// createPauseMenuButtons 创建暂停菜单按钮
func (m *PauseMenuModule) createPauseMenuButtons(rm *game.ResourceManager) error {
	// 按钮初始位置：屏幕外（隐藏）
	hiddenX := -1000.0
	hiddenY := -1000.0

	// 加载"返回游戏"按钮图片（原版资源）
	backToGameNormal := rm.GetImageByID("IMAGE_OPTIONS_BACKTOGAMEBUTTON0")
	backToGamePressed := rm.GetImageByID("IMAGE_OPTIONS_BACKTOGAMEBUTTON2")

	if backToGameNormal == nil {
		return fmt.Errorf("failed to load back to game button images")
	}

	// 加载按钮文字字体
	buttonFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.PauseMenuBackToGameButtonFontSize)
	if err != nil {
		log.Printf("[PauseMenuModule] Warning: Failed to load button font: %v", err)
		buttonFont = nil
	}

	// 1. 创建"返回游戏"按钮
	backToGameEntity := m.entityManager.CreateEntity()
	ecs.AddComponent(m.entityManager, backToGameEntity, &components.PositionComponent{
		X: hiddenX,
		Y: hiddenY,
	})
	ecs.AddComponent(m.entityManager, backToGameEntity, &components.ButtonComponent{
		Type:         components.ButtonTypeSimple,
		NormalImage:  backToGameNormal,
		HoverImage:   backToGameNormal,  // ✅ 悬停时不换图（backtogamebutton 系列没有悬停状态）
		PressedImage: backToGamePressed, // ✅ 按下时使用 button2（下陷效果）
		Text:         "返回游戏",
		Font:         buttonFont,
		TextColor:    [4]uint8{0, 200, 0, 255},
		State:        components.UINormal,
		Enabled:      true,
		OnClick: func() {
			log.Printf("[PauseMenuModule] Back to game button clicked!")
			m.Hide()
			if m.onContinue != nil {
				m.onContinue()
			}
		},
	})
	m.buttonEntities = append(m.buttonEntities, backToGameEntity)

	// 2. 创建"重新开始"按钮
	restartEntity, err := m.createThreeSliceButton(rm, hiddenX, hiddenY, "重新开始", func() {
		log.Printf("[PauseMenuModule] Restart button clicked!")
		m.Hide()
		if m.onRestart != nil {
			m.onRestart()
		}
	})
	if err != nil {
		log.Printf("[PauseMenuModule] Warning: Failed to create restart button: %v", err)
	} else {
		m.buttonEntities = append(m.buttonEntities, restartEntity)
	}

	// 3. 创建"主菜单"按钮
	mainMenuEntity, err := m.createThreeSliceButton(rm, hiddenX, hiddenY, "主菜单", func() {
		log.Printf("[PauseMenuModule] Main menu button clicked!")
		// Story 18.2: 返回主菜单前先保存战斗状态
		if m.onSaveBattle != nil {
			log.Printf("[PauseMenuModule] Triggering battle save before returning to main menu...")
			m.onSaveBattle()
		}
		m.Hide()
		if m.onMainMenu != nil {
			m.onMainMenu()
		}
	})
	if err != nil {
		log.Printf("[PauseMenuModule] Warning: Failed to create main menu button: %v", err)
	} else {
		m.buttonEntities = append(m.buttonEntities, mainMenuEntity)
	}

	return nil
}

// createThreeSliceButton 创建三段式按钮（复用 entities.NewMenuButton）
func (m *PauseMenuModule) createThreeSliceButton(rm *game.ResourceManager, x, y float64, text string, onClick func()) (ecs.EntityID, error) {
	return entities.NewMenuButton(
		m.entityManager,
		rm,
		x,
		y,
		text,
		config.PauseMenuInnerButtonFontSize,
		[4]uint8{0, 200, 0, 255},
		config.PauseMenuInnerButtonWidth-40,
		onClick,
	)
}

// Update 更新暂停菜单状态
//
// 参数:
//   - deltaTime: 距离上一帧的时间间隔（秒）
//
// 职责:
//   - 检测 GameState.IsPaused 变化，触发音乐回调
//   - 更新设置面板和按钮位置
func (m *PauseMenuModule) Update(deltaTime float64) {
	// 从 GameState 同步暂停状态
	isPaused := m.gameState.IsPaused

	// 检测状态变化（处理 ESC 键触发的暂停/恢复）
	if isPaused != m.wasActive {
		if isPaused {
			// 刚刚暂停
			if m.onPauseMusic != nil {
				m.onPauseMusic()
			}
			log.Printf("[PauseMenuModule] Paused (triggered by external state change)")
		} else {
			// 刚刚恢复
			if m.onResumeMusic != nil {
				m.onResumeMusic()
			}
			log.Printf("[PauseMenuModule] Resumed (triggered by external state change)")
		}
		m.wasActive = isPaused
	}

	// 更新设置面板状态（委托给 SettingsPanelModule）
	if isPaused {
		if !m.settingsPanelModule.IsActive() {
			m.settingsPanelModule.Show()
		}
		m.showButtons()
	} else {
		if m.settingsPanelModule.IsActive() {
			m.settingsPanelModule.Hide()
		}
		m.hideButtons()
	}

	// 更新设置面板（UI 元素位置同步）
	m.settingsPanelModule.Update(deltaTime)
}

// showButtons 显示所有暂停菜单按钮（移动到正确位置）
func (m *PauseMenuModule) showButtons() {
	if len(m.buttonEntities) == 0 {
		return
	}

	screenCenterX := float64(m.windowWidth) / 2.0
	screenCenterY := float64(m.windowHeight) / 2.0

	// 1. "返回游戏"按钮（在墓碑底部外侧）
	if len(m.buttonEntities) > 0 {
		backToGameEntity := m.buttonEntities[0]
		if button, ok := ecs.GetComponent[*components.ButtonComponent](m.entityManager, backToGameEntity); ok {
			buttonWidth := float64(button.NormalImage.Bounds().Dx())
			buttonX := screenCenterX - buttonWidth/2.0
			buttonY := screenCenterY + config.PauseMenuBackToGameButtonOffsetY

			if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, backToGameEntity); ok {
				pos.X = buttonX
				pos.Y = buttonY
			}
		}
	}

	// 2. "重新开始"按钮（在墓碑内部，顶部）
	if len(m.buttonEntities) > 1 {
		restartEntity := m.buttonEntities[1]
		if button, ok := ecs.GetComponent[*components.ButtonComponent](m.entityManager, restartEntity); ok {
			var buttonWidth float64
			if button.Type == components.ButtonTypeNineSlice {
				leftWidth := float64(button.LeftImage.Bounds().Dx())
				rightWidth := float64(button.RightImage.Bounds().Dx())
				buttonWidth = leftWidth + button.MiddleWidth + rightWidth
			} else {
				buttonWidth = config.PauseMenuInnerButtonWidth
			}

			buttonX := screenCenterX - buttonWidth/2.0
			buttonY := screenCenterY + config.PauseMenuRestartButtonOffsetY

			if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, restartEntity); ok {
				pos.X = buttonX
				pos.Y = buttonY
			}
		}
	}

	// 3. "主菜单"按钮（在墓碑内部，中间偏下）
	if len(m.buttonEntities) > 2 {
		mainMenuEntity := m.buttonEntities[2]
		if button, ok := ecs.GetComponent[*components.ButtonComponent](m.entityManager, mainMenuEntity); ok {
			var buttonWidth float64
			if button.Type == components.ButtonTypeNineSlice {
				leftWidth := float64(button.LeftImage.Bounds().Dx())
				rightWidth := float64(button.RightImage.Bounds().Dx())
				buttonWidth = leftWidth + button.MiddleWidth + rightWidth
			} else {
				buttonWidth = config.PauseMenuInnerButtonWidth
			}

			buttonX := screenCenterX - buttonWidth/2.0
			buttonY := screenCenterY + config.PauseMenuMainMenuButtonOffsetY

			if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, mainMenuEntity); ok {
				pos.X = buttonX
				pos.Y = buttonY
			}
		}
	}
}

// hideButtons 隐藏所有暂停菜单按钮（移动到屏幕外）
func (m *PauseMenuModule) hideButtons() {
	hiddenX := -1000.0
	hiddenY := -1000.0

	for _, entityID := range m.buttonEntities {
		if pos, ok := ecs.GetComponent[*components.PositionComponent](m.entityManager, entityID); ok {
			pos.X = hiddenX
			pos.Y = hiddenY
		}
	}
}

// Draw 渲染暂停菜单到屏幕
//
// 参数:
//   - screen: 目标渲染屏幕
//
// 渲染顺序:
//  1. 设置面板（遮罩、墓碑背景、UI 元素） - 委托给 SettingsPanelModule
//  2. 暂停菜单按钮（在背景上方）
func (m *PauseMenuModule) Draw(screen *ebiten.Image) {
	if !m.gameState.IsPaused {
		return
	}

	// 1. 渲染设置面板（委托给 SettingsPanelModule）
	m.settingsPanelModule.Draw(screen)

	// 2. 渲染暂停菜单的按钮（在背景上方）
	if m.buttonRenderSystem != nil {
		for _, buttonEntity := range m.buttonEntities {
			m.buttonRenderSystem.DrawButton(screen, buttonEntity)
		}
	}
}

// Show 显示暂停菜单
//
// 效果:
//   - 设置 GameState.IsPaused = true
//   - 更新 wasActive 状态
//   - 暂停 BGM
func (m *PauseMenuModule) Show() {
	m.gameState.SetPaused(true)
	m.wasActive = true
	if m.onPauseMusic != nil {
		m.onPauseMusic()
	}
	log.Printf("[PauseMenuModule] Pause menu shown")
}

// Hide 隐藏暂停菜单
//
// 效果:
//   - 设置 GameState.IsPaused = false
//   - 更新 wasActive 状态
//   - 恢复 BGM
func (m *PauseMenuModule) Hide() {
	m.gameState.SetPaused(false)
	m.wasActive = false
	if m.onResumeMusic != nil {
		m.onResumeMusic()
	}
	log.Printf("[PauseMenuModule] Pause menu hidden")
}

// Toggle 切换暂停菜单显示/隐藏
func (m *PauseMenuModule) Toggle() {
	if m.gameState.IsPaused {
		m.Hide()
	} else {
		m.Show()
	}
}

// IsActive 检查暂停菜单是否激活
func (m *PauseMenuModule) IsActive() bool {
	return m.gameState.IsPaused
}

// GetButtonEntities 返回所有按钮实体ID（调试用）
func (m *PauseMenuModule) GetButtonEntities() []ecs.EntityID {
	return m.buttonEntities
}

// Cleanup 清理模块资源
//
// 用途:
//   - 场景切换时清理所有暂停菜单实体
//   - 避免内存泄漏
func (m *PauseMenuModule) Cleanup() {
	// 清理设置面板模块（委托）
	if m.settingsPanelModule != nil {
		m.settingsPanelModule.Cleanup()
	}

	// 清理所有按钮实体
	for _, entityID := range m.buttonEntities {
		m.entityManager.DestroyEntity(entityID)
	}

	log.Printf("[PauseMenuModule] Cleaned up")
}
