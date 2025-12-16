package scenes

import (
	"image"
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/systems"
	"github.com/gonewx/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ============================================================================
// 铲子系统相关方法
// Story 15.5: 从 game_scene.go 拆分出铲子系统逻辑（约400行）
// ============================================================================

// IsShovelSelected 返回铲子是否被选中
// 实现 systems.ShovelStateProvider 接口
func (s *GameScene) IsShovelSelected() bool {
	return s.shovelSelected
}

// SetShovelSelected 设置铲子选中状态
// 实现 systems.ShovelStateProvider 接口
// Bug修复: 选中铲子时需要清理植物预览实体，避免铲子光标与植物预览同时显示
func (s *GameScene) SetShovelSelected(selected bool) {
	s.shovelSelected = selected
	// 如果取消铲子模式，同时取消种植模式（避免状态冲突）
	if !selected && s.gameState.IsPlantingMode {
		// 不需要操作，铲子模式和种植模式互斥
	}
	// 如果选中铲子，取消种植模式并清理预览
	if selected && s.gameState.IsPlantingMode {
		s.gameState.ExitPlantingMode()
		// 销毁植物预览实体，避免铲子光标与植物预览同时显示
		s.destroyAllPlantPreviews()
		log.Printf("[GameScene] 铲子模式激活，取消种植模式并清理植物预览")
	}
}

// GetShovelSlotBounds 获取铲子槽位边界（屏幕坐标）
// 实现 systems.ShovelStateProvider 接口
// Story 19.5: 保龄球模式使用相对于菜单按钮的位置
// Bug修复: 铲子Y位置固定在顶部，与菜单按钮显示时机一致，不跟随植物选择栏滑入动画
func (s *GameScene) GetShovelSlotBounds() image.Rectangle {
	// 计算铲子位置
	var shovelX int
	// Story 19.5: 保龄球模式（initialSun == 0）使用相对于菜单按钮的位置
	if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.InitialSun == 0 {
		// 菜单按钮 X 位置
		menuButtonX := WindowWidth - int(config.MenuButtonOffsetFromRight)
		// 铲子右边缘到菜单按钮左边缘的距离为 BowlingShovelGapFromMenuButton
		// 铲子 X = 菜单按钮 X - 间距 - 铲子宽度
		shovelX = menuButtonX - config.BowlingShovelGapFromMenuButton - config.ShovelWidth
	} else if s.seedBank != nil {
		// 普通模式根据选择栏图片宽度动态计算
		seedBankWidth := s.seedBank.Bounds().Dx()
		shovelX = config.SeedBankX + seedBankWidth + config.ShovelGapFromSeedBank
	} else {
		shovelX = config.ShovelX // 默认值
	}

	// 铲子 Y 位置固定在顶部（与菜单按钮一致，不跟随植物选择栏滑入动画）
	shovelY := int(config.MenuButtonOffsetFromTop)

	return image.Rect(
		shovelX,
		shovelY,
		shovelX+config.ShovelWidth,
		shovelY+config.ShovelHeight,
	)
}

// GetShovelIconBounds 获取铲子图标边界（屏幕坐标）
// 实现 systems.ShovelSlotBoundsProvider 接口
// Story 19.x QA: 铲子图标在卡槽内居中显示，此方法返回图标的实际位置
func (s *GameScene) GetShovelIconBounds() image.Rectangle {
	slotBounds := s.GetShovelSlotBounds()

	// 如果铲子图片不存在，返回卡槽边界
	if s.shovel == nil {
		return slotBounds
	}

	// 计算铲子图标在卡槽内的居中位置
	shovelImgBounds := s.shovel.Bounds()
	shovelImgW := shovelImgBounds.Dx()
	shovelImgH := shovelImgBounds.Dy()

	// 居中偏移：(卡槽尺寸 - 图片尺寸) / 2
	slotW := config.ShovelWidth
	slotH := config.ShovelHeight
	offsetX := (slotW - shovelImgW) / 2
	offsetY := (slotH - shovelImgH) / 2

	iconX := slotBounds.Min.X + offsetX
	iconY := slotBounds.Min.Y + offsetY

	return image.Rect(
		iconX,
		iconY,
		iconX+shovelImgW,
		iconY+shovelImgH,
	)
}

// updateShovelSlotClick 检测铲子槽位点击
// Story 19.2: 点击铲子槽位切换铲子模式
// Story 19.3: 强引导模式下检查操作限制
// Story 19.x QA: 铲子教学关卡（有预设植物）强制启用铲子点击
// 移动端拖拽：支持从铲子槽位拖拽到植物直接铲除
func (s *GameScene) updateShovelSlotClick() {
	// Bug Fix: Dave 对话期间禁止铲子槽位点击
	// 对话期间点击屏幕是推进对话，不应触发其他交互
	if s.isDaveDialogueActive() {
		return
	}

	// 检查铲子是否可用
	// Story 19.x QA: 铲子教学关卡（有预设植物）强制启用铲子
	isShovelTutorialLevel := s.gameState.CurrentLevel != nil && len(s.gameState.CurrentLevel.PresetPlants) > 0

	// 教学关卡不显示铲子（玩家还不需要学习移除植物）
	// 但是：铲子教学关卡或强引导模式激活时需要启用铲子
	if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.OpeningType == "tutorial" {
		if !isShovelTutorialLevel && !s.IsGuidedTutorialActive() {
			return
		}
	}

	// Story 8.6: 检查铲子是否已解锁
	// 例外：铲子教学关卡强制启用铲子
	if !s.gameState.IsToolUnlocked("shovel") && !isShovelTutorialLevel {
		return
	}

	// Q 键快捷键切换铲子模式
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		s.toggleShovelMode()
		return
	}

	// ========================================================================
	// 移动端拖拽处理：从铲子槽位拖拽到植物直接铲除
	// ========================================================================
	dragManager := utils.GetDragManager()
	if dragManager.IsTouchDrag() || s.isDragShovel {
		if s.handleShovelDrag() {
			return // 拖拽处理中，跳过传统点击逻辑
		}
	}

	// ========================================================================
	// 传统点击处理（桌面端鼠标点击）
	// ========================================================================
	// 检测左键点击或触摸
	justPressed, mouseX, mouseY := utils.IsJustTouchedOrClicked()
	if justPressed {
		// Bug Fix: 触摸输入由拖拽逻辑处理，这里只处理鼠标点击
		// 避免触摸开始时 toggleShovelMode() 和 handleShovelDragStart() 都被调用导致重复音效
		touchIDs := inpututil.AppendJustPressedTouchIDs(nil)
		if len(touchIDs) > 0 {
			// 触摸输入会在下一帧由 handleShovelDrag() 处理（DragManager 状态更新后）
			// 这里跳过，避免重复播放音效
			return
		}

		bounds := s.GetShovelSlotBounds()

		// 检查是否点击了铲子槽位
		if mouseX >= bounds.Min.X && mouseX <= bounds.Max.X &&
			mouseY >= bounds.Min.Y && mouseY <= bounds.Max.Y {
			s.toggleShovelMode()
		}
	}

	// 检测右键取消铲子模式
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		if s.shovelSelected {
			s.SetShovelSelected(false)
			// 立即恢复系统光标并清除高亮
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
			if s.shovelInteractionSystem != nil {
				s.shovelInteractionSystem.ClearHighlight()
			}
			log.Printf("[GameScene] 右键取消铲子模式")
		}
	}
}

// toggleShovelMode 切换铲子模式
func (s *GameScene) toggleShovelMode() {
	// Story 19.3: 检查操作是否被允许
	if !s.IsOperationAllowed("click_shovel") {
		return // 静默忽略
	}

	// Story 19.3: 通知系统操作发生
	s.NotifyOperation("click_shovel")

	// 播放铲子点击音效（使用 AudioManager 统一管理 - Story 10.9）
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_SHOVEL")
	}

	// 切换铲子选中状态
	s.SetShovelSelected(!s.shovelSelected)
	if s.shovelSelected {
		// 激活时隐藏系统光标（铲子图标会作为光标显示）
		ebiten.SetCursorMode(ebiten.CursorModeHidden)
		log.Printf("[GameScene] 铲子模式激活")
	} else {
		// 取消时恢复系统光标并清除高亮
		ebiten.SetCursorMode(ebiten.CursorModeVisible)
		if s.shovelInteractionSystem != nil {
			s.shovelInteractionSystem.ClearHighlight()
		}
		log.Printf("[GameScene] 铲子模式取消")
	}
}

// handleShovelDrag 处理铲子的拖拽交互（移动端触摸拖拽支持）
// 流程：
//  1. 触摸铲子槽位 → 开始拖拽，激活铲子模式
//  2. 拖拽过程中 → 铲子图标跟随手指，检测悬停植物
//  3. 释放手指 → 如果在植物上则铲除，否则取消铲子模式
//
// 返回 true 表示正在处理拖拽，应跳过传统点击逻辑
func (s *GameScene) handleShovelDrag() bool {
	dragManager := utils.GetDragManager()
	dragInfo := dragManager.GetInfo()

	switch dragInfo.State {
	case utils.DragStateStarted:
		// 拖拽刚开始，检测是否从铲子槽位开始
		return s.handleShovelDragStart(dragInfo)

	case utils.DragStateDragging:
		// 拖拽进行中，更新铲子位置和植物高亮
		if s.isDragShovel {
			s.updateShovelDragPreview(dragInfo)
			return true
		}

	case utils.DragStateEnded:
		// 拖拽结束，尝试铲除植物或取消
		if s.isDragShovel {
			s.handleShovelDragEnd(dragInfo)
			return true
		}
	}

	return false
}

// handleShovelDragStart 处理铲子拖拽开始
// 检测拖拽是否从铲子槽位开始（仅触摸输入）
func (s *GameScene) handleShovelDragStart(dragInfo utils.DragInfo) bool {
	// 只处理触摸输入的拖拽，桌面端鼠标使用传统点击模式
	if !dragInfo.IsTouchInput {
		return false
	}

	// 检测触摸位置是否在铲子槽位上
	bounds := s.GetShovelSlotBounds()
	if dragInfo.StartX < bounds.Min.X || dragInfo.StartX > bounds.Max.X ||
		dragInfo.StartY < bounds.Min.Y || dragInfo.StartY > bounds.Max.Y {
		return false // 不是从铲子槽位开始的拖拽
	}

	// Story 19.3: 检查操作是否被允许
	if !s.IsOperationAllowed("click_shovel") {
		return false
	}

	// Story 19.3: 通知系统操作发生
	s.NotifyOperation("click_shovel")

	// 开始铲子拖拽模式
	s.isDragShovel = true
	s.SetShovelSelected(true)

	// 播放铲子点击音效
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_SHOVEL")
	}

	log.Printf("[GameScene] 铲子拖拽开始: 起点=(%d, %d)", dragInfo.StartX, dragInfo.StartY)

	return true
}

// updateShovelDragPreview 更新铲子拖拽预览
// 铲子图标跟随手指移动，并检测悬停的植物
func (s *GameScene) updateShovelDragPreview(dragInfo utils.DragInfo) {
	// 铲子光标的渲染由 ShovelInteractionSystem 处理
	// 这里只需要确保铲子模式保持激活状态
	// ShovelInteractionSystem.Update() 会自动检测鼠标/触摸位置下的植物并高亮
}

// handleShovelDragEnd 处理铲子拖拽结束
func (s *GameScene) handleShovelDragEnd(dragInfo utils.DragInfo) {
	defer s.cancelShovelDrag()

	// 转换为世界坐标
	worldX := float64(dragInfo.CurrentX) + s.cameraX
	worldY := float64(dragInfo.CurrentY)

	// 检测释放位置是否有植物
	plantEntity := s.detectPlantAtPosition(worldX, worldY)
	if plantEntity != 0 {
		// 找到植物，执行铲除操作
		s.removePlantWithShovel(plantEntity)
		log.Printf("[GameScene] 铲子拖拽结束: 铲除植物 EntityID=%d", plantEntity)
	} else {
		log.Printf("[GameScene] 铲子拖拽结束: 未找到植物，取消铲子模式")
	}
}

// cancelShovelDrag 取消铲子拖拽模式
func (s *GameScene) cancelShovelDrag() {
	if !s.isDragShovel {
		return
	}

	// 取消铲子选中状态
	s.SetShovelSelected(false)

	// 清除高亮效果
	if s.shovelInteractionSystem != nil {
		s.shovelInteractionSystem.ClearHighlight()
	}

	// 恢复系统光标
	ebiten.SetCursorMode(ebiten.CursorModeVisible)

	// 重置拖拽状态
	s.isDragShovel = false

	log.Printf("[GameScene] 铲子拖拽取消")
}

// detectPlantAtPosition 检测指定世界坐标位置的植物
// 返回植物实体ID，如果没有则返回 0
func (s *GameScene) detectPlantAtPosition(worldX, worldY float64) ecs.EntityID {
	// 查询所有植物实体
	plantEntities := ecs.GetEntitiesWith1[*components.PlantComponent](s.entityManager)

	for _, entity := range plantEntities {
		// 获取植物位置
		posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entity)
		if !ok {
			continue
		}

		// 计算植物边界（与 ShovelInteractionSystem 保持一致）
		plantWidth := 60.0
		plantHeight := 80.0

		plantLeft := posComp.X - plantWidth/2
		plantRight := posComp.X + plantWidth/2
		plantTop := posComp.Y - plantHeight/2
		plantBottom := posComp.Y + plantHeight/2

		// 检测坐标是否在植物边界内
		if worldX >= plantLeft && worldX <= plantRight &&
			worldY >= plantTop && worldY <= plantBottom {
			return entity
		}
	}

	return 0
}

// removePlantWithShovel 使用铲子移除植物
// 复用 ShovelInteractionSystem 的逻辑
func (s *GameScene) removePlantWithShovel(entityID ecs.EntityID) {
	// Story 19.3: 通知强引导系统发生了植物点击操作
	systems.NotifyGuidedTutorialOperation("click_plant")

	// 获取植物信息（用于日志和网格更新）
	plantComp, hasPlant := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID)
	posComp, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)

	if hasPlant && hasPos {
		log.Printf("[GameScene] 拖拽铲除植物: 类型=%v, 位置=(%.1f, %.1f), 网格=(%d, %d)",
			plantComp.PlantType, posComp.X, posComp.Y, plantComp.GridRow, plantComp.GridCol)

		// 更新草坪网格，释放该格子
		lawnGridEntities := ecs.GetEntitiesWith1[*components.LawnGridComponent](s.entityManager)
		if len(lawnGridEntities) > 0 {
			gridComp, ok := ecs.GetComponent[*components.LawnGridComponent](s.entityManager, lawnGridEntities[0])
			if ok && plantComp.GridRow >= 0 && plantComp.GridRow < 5 &&
				plantComp.GridCol >= 0 && plantComp.GridCol < 9 {
				gridComp.Occupancy[plantComp.GridRow][plantComp.GridCol] = 0 // 0 表示空格子
				log.Printf("[GameScene] 释放网格 (%d, %d)", plantComp.GridRow, plantComp.GridCol)
			}
		}
	}

	// 播放铲除植物音效
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_PLANT")
	}

	// 移除植物实体（不返还阳光）
	s.entityManager.DestroyEntity(entityID)

	log.Printf("[GameScene] 植物已移除 (Entity ID: %d)", entityID)
}
