package systems

import (
	"image"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ShovelInteractionSystem 铲子交互系统
// Story 19.2: 铲子交互系统增强
//
// 此系统负责：
//   - 检测铲子槽位点击，切换铲子选中状态
//   - 检测鼠标悬停的植物，更新高亮状态
//   - 检测左键点击事件，播放音效并移除植物
//   - 渲染铲子光标和植物高亮效果
type ShovelInteractionSystem struct {
	entityManager        *ecs.EntityManager
	gameState            *game.GameState
	resourceManager      *game.ResourceManager
	cursorImage          *ebiten.Image         // 铲子光标图片
	cursorAnchorX        float64               // 光标锚点X偏移
	cursorAnchorY        float64               // 光标锚点Y偏移
	shovelEntity         ecs.EntityID          // 铲子交互实体ID
	lastCursorMode       ebiten.CursorModeType // 上一次的光标模式
	justActivated        bool                  // 刚刚被激活，跳过本帧的取消检测
	lastHighlightedPlant ecs.EntityID          // 上一帧高亮的植物实体ID（用于移除闪烁效果）
}

// ShovelStateProvider 铲子状态提供者接口
// Story 19.2: 用于 GameScene 向系统提供铲子选中状态
// 遵循零耦合原则，系统不直接依赖 GameScene
type ShovelStateProvider interface {
	// IsShovelSelected 返回铲子是否被选中
	IsShovelSelected() bool
	// SetShovelSelected 设置铲子选中状态
	SetShovelSelected(selected bool)
	// GetShovelSlotBounds 获取铲子槽位边界（屏幕坐标）
	GetShovelSlotBounds() image.Rectangle
}

// shovelStateProvider 铲子状态提供者引用
var shovelStateProvider ShovelStateProvider

// NewShovelInteractionSystem 创建铲子交互系统
//
// 参数：
//   - em: 实体管理器
//   - gs: 游戏状态
//   - rm: 资源管理器
//
// 返回：
//   - 铲子交互系统实例
func NewShovelInteractionSystem(em *ecs.EntityManager, gs *game.GameState, rm *game.ResourceManager) *ShovelInteractionSystem {
	system := &ShovelInteractionSystem{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
		cursorAnchorX:   10.0, // 铲子尖端X偏移（左下角附近）
		cursorAnchorY:   70.0, // 铲子尖端Y偏移（左下角附近，图片高度80）
		lastCursorMode:  ebiten.CursorModeVisible,
	}

	// 音效统一由 AudioManager 管理（Story 10.9）

	// 加载铲子光标图片（使用 Shovel.png，与卡槽中的铲子一致）
	cursorImg, err := rm.LoadImage("assets/images/Shovel.png")
	if err != nil {
		log.Printf("[ShovelInteractionSystem] Warning: Failed to load shovel cursor image: %v", err)
	}
	system.cursorImage = cursorImg

	// 创建铲子交互实体
	system.shovelEntity = em.CreateEntity()
	shovelComp := &components.ShovelInteractionComponent{
		IsSelected:             false,
		CursorImage:            cursorImg,
		HighlightedPlantEntity: 0,
		CursorAnchorX:          system.cursorAnchorX,
		CursorAnchorY:          system.cursorAnchorY,
	}
	em.AddComponent(system.shovelEntity, shovelComp)

	// 验证光标图片是否加载成功
	if system.cursorImage != nil {
		bounds := system.cursorImage.Bounds()
		log.Printf("[ShovelInteractionSystem] Cursor image loaded: %dx%d pixels", bounds.Dx(), bounds.Dy())
	} else {
		log.Printf("[ShovelInteractionSystem] WARNING: Cursor image is nil, shovel cursor will not display!")
	}

	log.Printf("[ShovelInteractionSystem] Initialized (Entity ID: %d)", system.shovelEntity)

	return system
}

// SetShovelStateProvider 设置铲子状态提供者
// Story 19.2: GameScene 在初始化后调用此方法，提供铲子状态访问接口
func SetShovelStateProvider(provider ShovelStateProvider) {
	shovelStateProvider = provider
}

// Update 更新铲子交互状态
//
// 参数：
//   - deltaTime: 时间增量（秒）
//   - cameraX: 摄像机X坐标
func (s *ShovelInteractionSystem) Update(deltaTime float64, cameraX float64) {
	// 获取铲子组件
	shovelComp, ok := ecs.GetComponent[*components.ShovelInteractionComponent](s.entityManager, s.shovelEntity)
	if !ok {
		return
	}

	// 记录上一帧的选中状态
	wasSelected := shovelComp.IsSelected

	// 从状态提供者同步选中状态
	if shovelStateProvider != nil {
		shovelComp.IsSelected = shovelStateProvider.IsShovelSelected()
	}

	// 检测是否刚刚被激活（从未选中变为选中）
	if shovelComp.IsSelected && !wasSelected {
		s.justActivated = true
	}

	// 如果铲子未选中，跳过处理
	if !shovelComp.IsSelected {
		// 恢复系统光标
		if s.lastCursorMode != ebiten.CursorModeVisible {
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
			s.lastCursorMode = ebiten.CursorModeVisible
		}
		// 移除上一帧高亮植物的闪烁效果
		s.clearPlantHighlight()
		shovelComp.HighlightedPlantEntity = 0
		s.justActivated = false
		return
	}

	// 隐藏系统光标
	if s.lastCursorMode != ebiten.CursorModeHidden {
		ebiten.SetCursorMode(ebiten.CursorModeHidden)
		s.lastCursorMode = ebiten.CursorModeHidden
	}

	// 获取鼠标位置
	mouseScreenX, mouseScreenY := ebiten.CursorPosition()
	mouseWorldX := float64(mouseScreenX) + cameraX
	mouseWorldY := float64(mouseScreenY)

	// 检测鼠标悬停的植物
	highlightedPlant := s.detectPlantUnderMouse(mouseWorldX, mouseWorldY)

	// 更新植物高亮效果（使用 FlashEffectComponent）
	s.updatePlantHighlight(highlightedPlant)
	shovelComp.HighlightedPlantEntity = highlightedPlant

	// 检测左键点击事件
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// 如果刚刚被激活，跳过本次点击检测（避免同一次点击被处理两次）
		if s.justActivated {
			s.justActivated = false
			return
		}

		// 如果有高亮植物，移除它
		if highlightedPlant != 0 {
			s.removePlant(highlightedPlant)
			shovelComp.HighlightedPlantEntity = 0
		}
	}

	// 重置激活标记（如果没有点击事件，下一帧重置）
	s.justActivated = false
}

// detectPlantUnderMouse 检测鼠标悬停的植物
//
// 参数：
//   - worldX: 鼠标世界坐标X
//   - worldY: 鼠标世界坐标Y
//
// 返回：
//   - 检测到的植物实体ID，如果没有则返回 0
func (s *ShovelInteractionSystem) detectPlantUnderMouse(worldX, worldY float64) ecs.EntityID {
	// 查询所有植物实体
	plantEntities := ecs.GetEntitiesWith1[*components.PlantComponent](s.entityManager)

	for _, entity := range plantEntities {
		// 获取植物位置
		posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entity)
		if !ok {
			continue
		}

		// 计算植物边界
		// 使用一个通用的植物检测尺寸
		plantWidth := 60.0
		plantHeight := 80.0

		plantLeft := posComp.X - plantWidth/2
		plantRight := posComp.X + plantWidth/2
		plantTop := posComp.Y - plantHeight/2
		plantBottom := posComp.Y + plantHeight/2

		// 检测鼠标是否在植物边界内
		if worldX >= plantLeft && worldX <= plantRight &&
			worldY >= plantTop && worldY <= plantBottom {
			return entity
		}
	}

	return 0
}

// removePlant 移除植物
//
// 参数：
//   - entityID: 要移除的植物实体ID
//
// Story 19.3: 强引导模式下通知系统植物被移除
// Level 1-5: 铲子教学阶段，每次铲除一个植物后自动释放铲子
func (s *ShovelInteractionSystem) removePlant(entityID ecs.EntityID) {
	// Story 19.3: 通知强引导系统发生了植物点击操作
	NotifyGuidedTutorialOperation("click_plant")
	// 获取植物信息（用于日志和网格更新）
	plantComp, hasPlant := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID)
	posComp, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)

	if hasPlant && hasPos {
		log.Printf("[ShovelInteractionSystem] 移除植物: 类型=%v, 位置=(%.1f, %.1f), 网格=(%d, %d)",
			plantComp.PlantType, posComp.X, posComp.Y, plantComp.GridRow, plantComp.GridCol)

		// 更新草坪网格，释放该格子
		lawnGridEntities := ecs.GetEntitiesWith1[*components.LawnGridComponent](s.entityManager)
		if len(lawnGridEntities) > 0 {
			gridComp, ok := ecs.GetComponent[*components.LawnGridComponent](s.entityManager, lawnGridEntities[0])
			if ok && plantComp.GridRow >= 0 && plantComp.GridRow < 5 &&
				plantComp.GridCol >= 0 && plantComp.GridCol < 9 {
				gridComp.Occupancy[plantComp.GridRow][plantComp.GridCol] = 0 // 0 表示空格子
				log.Printf("[ShovelInteractionSystem] 释放网格 (%d, %d)", plantComp.GridRow, plantComp.GridCol)
			}
		}
	}

	// 播放铲除植物音效（使用 AudioManager 统一管理 - Story 10.9）
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_PLANT")
	}

	// 移除植物实体（不返还阳光）
	s.entityManager.DestroyEntity(entityID)

	log.Printf("[ShovelInteractionSystem] 植物已移除 (Entity ID: %d)", entityID)

	// Level 1-5: 强引导模式（铲子教学阶段）下，每次铲除一个植物后自动释放铲子
	// 玩家需要再次点击铲子才能继续铲除下一个植物
	if guidedTutorialStateProvider != nil && guidedTutorialStateProvider.IsGuidedTutorialActive() {
		// 释放铲子选中状态
		if shovelStateProvider != nil {
			shovelStateProvider.SetShovelSelected(false)
			log.Printf("[ShovelInteractionSystem] 强引导模式: 铲除植物后自动释放铲子")
		}
		// 清除高亮效果
		s.clearPlantHighlight()
		// 立即恢复系统光标（不等待下一帧 Update）
		ebiten.SetCursorMode(ebiten.CursorModeVisible)
		s.lastCursorMode = ebiten.CursorModeVisible
	}
}

// Draw 渲染铲子光标和植物高亮效果
//
// 参数：
//   - screen: 目标屏幕
//   - cameraX: 摄像机X坐标
func (s *ShovelInteractionSystem) Draw(screen *ebiten.Image, cameraX float64) {
	// 获取铲子组件
	shovelComp, ok := ecs.GetComponent[*components.ShovelInteractionComponent](s.entityManager, s.shovelEntity)
	if !ok || !shovelComp.IsSelected {
		return
	}

	// 渲染植物高亮效果
	if shovelComp.HighlightedPlantEntity != 0 {
		s.drawPlantHighlight(screen, shovelComp.HighlightedPlantEntity, cameraX)
	}

	// 渲染铲子光标
	s.drawShovelCursor(screen)
}

// updatePlantHighlight 更新植物高亮效果
// 使用 HoverHighlightComponent 实现持续高亮（不闪烁）
//
// 参数：
//   - newHighlightedPlant: 新的高亮植物实体ID（0 表示无高亮）
func (s *ShovelInteractionSystem) updatePlantHighlight(newHighlightedPlant ecs.EntityID) {
	// 如果高亮植物没有变化，无需操作
	if newHighlightedPlant == s.lastHighlightedPlant {
		return
	}

	// 移除旧植物的高亮效果
	if s.lastHighlightedPlant != 0 {
		if _, hasHighlight := ecs.GetComponent[*components.HoverHighlightComponent](s.entityManager, s.lastHighlightedPlant); hasHighlight {
			ecs.RemoveComponent[*components.HoverHighlightComponent](s.entityManager, s.lastHighlightedPlant)
		}
	}

	// 添加新植物的高亮效果
	if newHighlightedPlant != 0 {
		// 检查植物是否已有 HoverHighlightComponent
		if _, hasHighlight := ecs.GetComponent[*components.HoverHighlightComponent](s.entityManager, newHighlightedPlant); !hasHighlight {
			// 添加悬停高亮组件
			highlightComp := &components.HoverHighlightComponent{
				Intensity: 0.3, // 中等强度的高亮效果
				IsActive:  true,
			}
			ecs.AddComponent(s.entityManager, newHighlightedPlant, highlightComp)
		}
	}

	// 更新上一帧高亮的植物
	s.lastHighlightedPlant = newHighlightedPlant
}

// clearPlantHighlight 清除所有植物高亮效果
func (s *ShovelInteractionSystem) clearPlantHighlight() {
	if s.lastHighlightedPlant != 0 {
		if _, hasHighlight := ecs.GetComponent[*components.HoverHighlightComponent](s.entityManager, s.lastHighlightedPlant); hasHighlight {
			ecs.RemoveComponent[*components.HoverHighlightComponent](s.entityManager, s.lastHighlightedPlant)
		}
		s.lastHighlightedPlant = 0
	}
}

// ClearHighlight 公开方法：清除植物高亮效果
// 供 GameScene 在取消铲子模式时调用
func (s *ShovelInteractionSystem) ClearHighlight() {
	s.clearPlantHighlight()
}

// drawPlantHighlight 渲染植物高亮效果
// 注意：高亮效果现在通过 FlashEffectComponent 由 RenderSystem 自动处理
// 此方法保留但不再绘制任何内容
//
// 参数：
//   - screen: 目标屏幕
//   - entityID: 高亮植物实体ID
//   - cameraX: 摄像机X坐标
func (s *ShovelInteractionSystem) drawPlantHighlight(screen *ebiten.Image, entityID ecs.EntityID, cameraX float64) {
	// 高亮效果通过 FlashEffectComponent 实现，RenderSystem 会自动渲染
	// 此方法不再需要绘制任何内容
}

// drawShovelCursor 渲染铲子光标
//
// 参数：
//   - screen: 目标屏幕
func (s *ShovelInteractionSystem) drawShovelCursor(screen *ebiten.Image) {
	if s.cursorImage == nil {
		log.Printf("[ShovelInteractionSystem] drawShovelCursor: cursorImage is nil!")
		return
	}

	// 获取鼠标位置
	mouseX, mouseY := ebiten.CursorPosition()

	// 绘制铲子图标
	op := &ebiten.DrawImageOptions{}
	// 应用锚点偏移，使铲子尖端对准鼠标位置
	op.GeoM.Translate(-s.cursorAnchorX, -s.cursorAnchorY)
	op.GeoM.Translate(float64(mouseX), float64(mouseY))
	screen.DrawImage(s.cursorImage, op)
}

// GetShovelEntity 获取铲子交互实体ID
func (s *ShovelInteractionSystem) GetShovelEntity() ecs.EntityID {
	return s.shovelEntity
}

// IsShovelMode 检查是否处于铲子模式
func (s *ShovelInteractionSystem) IsShovelMode() bool {
	shovelComp, ok := ecs.GetComponent[*components.ShovelInteractionComponent](s.entityManager, s.shovelEntity)
	if !ok {
		return false
	}
	return shovelComp.IsSelected
}

// Cleanup 清理系统资源
func (s *ShovelInteractionSystem) Cleanup() {
	// 恢复系统光标
	ebiten.SetCursorMode(ebiten.CursorModeVisible)
	// 清除状态提供者
	shovelStateProvider = nil
}
