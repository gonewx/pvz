package systems

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// ============================================================================
// 拖拽种植处理（移动端触摸拖拽支持）
// ============================================================================

// handleDragPlanting 处理拖拽种植逻辑
// 移动端触摸拖拽流程：
//  1. 触摸植物卡片 → 开始拖拽，显示预览
//  2. 拖拽过程中 → 预览跟随手指移动
//  3. 释放手指 → 如果在有效草坪格子上则放置，否则取消
//
// 返回 true 表示正在处理拖拽种植，应跳过传统点击逻辑
func (s *InputSystem) handleDragPlanting(cameraX float64) bool {
	dragManager := utils.GetDragManager()
	dragInfo := dragManager.GetInfo()

	switch dragInfo.State {
	case utils.DragStateStarted:
		// 拖拽刚开始，检测是否从植物卡片开始
		return s.handleDragStart(dragInfo, cameraX)

	case utils.DragStateDragging:
		// 拖拽进行中，更新预览位置
		if s.isDragPlanting {
			s.updateDragPreview(dragInfo, cameraX)
			return true
		}

	case utils.DragStateEnded:
		// 拖拽结束，尝试放置或取消
		if s.isDragPlanting {
			s.handleDragEnd(dragInfo, cameraX)
			return true
		}
	}

	return false
}

// handleDragStart 处理拖拽开始
// 检测拖拽是否从有效的植物卡片开始（仅触摸输入）
func (s *InputSystem) handleDragStart(dragInfo utils.DragInfo, cameraX float64) bool {
	// 只处理触摸输入的拖拽，桌面端鼠标使用传统点击模式
	if !dragInfo.IsTouchInput {
		return false
	}

	// 检测触摸位置是否在植物卡片上
	cardEntity, plantCard := s.getPlantCardAtPosition(dragInfo.StartX, dragInfo.StartY)
	if cardEntity == 0 || plantCard == nil {
		return false // 不是从植物卡片开始的拖拽
	}

	// Story 19.3: 检查强引导模式是否阻止卡片点击
	if IsGuidedTutorialBlocking("click_plant_card") {
		return false
	}

	// 检查卡片状态
	currentSun := s.gameState.GetSun()
	plantCost := plantCard.SunCost

	// 1. 冷却中 - 不做任何反应（除非启用忽略冷却模式）
	if plantCard.CurrentCooldown > 0 && !s.ignoreCooldown {
		log.Printf("[InputSystem] 拖拽开始: 卡片冷却中，忽略")
		return false
	}

	// 2. 阳光不足 - 触发闪烁反馈
	if currentSun < plantCost {
		log.Printf("[InputSystem] 拖拽开始: 阳光不足 (需要 %d, 当前 %d)", plantCost, currentSun)
		s.gameState.TriggerSunFlash()

		// 播放无效操作音效（带冷却）
		timeSinceLastBuzzer := s.gameTime - s.lastBuzzerPlayTime
		if timeSinceLastBuzzer >= s.buzzerCooldownTime {
			if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
				audioManager.PlaySound("SOUND_BUZZER")
				s.lastBuzzerPlayTime = s.gameTime
			}
		}
		return false
	}

	// 3. 卡片不可用 - 跳过
	if !plantCard.IsAvailable {
		log.Printf("[InputSystem] 拖拽开始: 卡片不可用，忽略")
		return false
	}

	// 开始拖拽种植模式
	s.isDragPlanting = true
	s.dragPlantType = plantCard.PlantType
	s.dragStartCardEntity = cardEntity

	// Bug修复: 如果当前处于铲子模式，先取消铲子模式
	if shovelStateProvider != nil && shovelStateProvider.IsShovelSelected() {
		shovelStateProvider.SetShovelSelected(false)
		// 恢复系统光标
		ebiten.SetCursorMode(ebiten.CursorModeVisible)
		log.Printf("[InputSystem] 拖拽取消铲子模式（选择植物卡片）")
	}

	// 进入种植模式
	s.gameState.EnterPlantingMode(plantCard.PlantType)

	// 播放选中植物卡片音效
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_SEEDLIFT")
	}

	// 创建植物预览实体
	mouseWorldX := float64(dragInfo.StartX) + cameraX
	mouseWorldY := float64(dragInfo.StartY)
	s.createPlantPreview(plantCard.PlantType, mouseWorldX, mouseWorldY)

	// 设置卡片状态为 Clicked（视觉反馈）
	if ui, ok := ecs.GetComponent[*components.UIComponent](s.entityManager, cardEntity); ok {
		ui.State = components.UIClicked
	}

	log.Printf("[InputSystem] 拖拽种植开始: PlantType=%v, 起点=(%d, %d)", plantCard.PlantType, dragInfo.StartX, dragInfo.StartY)

	return true
}

// updateDragPreview 更新拖拽预览位置
func (s *InputSystem) updateDragPreview(dragInfo utils.DragInfo, cameraX float64) {
	// 更新预览实体的位置
	previewEntities := ecs.GetEntitiesWith1[*components.PlantPreviewComponent](s.entityManager)
	for _, entityID := range previewEntities {
		pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		if ok {
			pos.X = float64(dragInfo.CurrentX) + cameraX
			pos.Y = float64(dragInfo.CurrentY)
		}
	}
}

// handleDragEnd 处理拖拽结束
func (s *InputSystem) handleDragEnd(dragInfo utils.DragInfo, cameraX float64) {
	defer s.cancelDragPlanting()

	// 检测释放位置是否在有效的草坪格子上
	col, row, isValid := utils.MouseToGridCoords(
		dragInfo.CurrentX, dragInfo.CurrentY,
		cameraX,
		config.GridWorldStartX, config.GridWorldStartY,
		config.GridColumns, config.GridRows,
		config.CellWidth, config.CellHeight,
	)

	if !isValid {
		log.Printf("[InputSystem] 拖拽结束: 位置在网格外，取消种植")
		return
	}

	// 检查该行是否启用
	lane := row + 1
	if !s.lawnGridSystem.IsLaneEnabled(lane) {
		log.Printf("[InputSystem] 拖拽结束: 行 %d 已被禁用，取消种植", lane)
		return
	}

	// 检查格子是否已被占用
	if s.lawnGridSystem.IsOccupied(s.lawnGridEntityID, col, row) {
		log.Printf("[InputSystem] 拖拽结束: 格子 (%d, %d) 已被占用，取消种植", col, row)
		return
	}

	// 获取植物消耗
	sunCost := s.getPlantCost(s.dragPlantType)

	// 尝试扣除阳光
	if !s.gameState.SpendSun(sunCost) {
		log.Printf("[InputSystem] 拖拽结束: 阳光不足，需要 %d，当前 %d", sunCost, s.gameState.GetSun())
		return
	}

	log.Printf("[InputSystem] 拖拽结束: 扣除阳光 %d，剩余 %d", sunCost, s.gameState.GetSun())

	// 创建植物实体
	plantID, err := s.createPlantEntity(s.dragPlantType, col, row)
	if err != nil {
		log.Printf("[InputSystem] 拖拽结束: 创建植物实体失败: %v", err)
		s.gameState.AddSun(sunCost) // 创建失败，返还阳光
		return
	}

	log.Printf("[InputSystem] 拖拽结束: 成功创建植物实体 (ID: %d, Type: %v) 在 (%d, %d)", plantID, s.dragPlantType, col, row)

	// 触发种植粒子效果
	worldX, worldY := utils.GridToWorldCoords(
		col, row,
		config.GridWorldStartX, config.GridWorldStartY,
		config.CellWidth, config.CellHeight,
	)
	_, err = entities.NewPlantingParticleEffect(s.entityManager, s.resourceManager, worldX, worldY)
	if err != nil {
		log.Printf("[InputSystem] 警告：创建种植粒子效果失败: %v", err)
	}

	// 标记格子为占用
	err = s.lawnGridSystem.OccupyCell(s.lawnGridEntityID, col, row, plantID)
	if err != nil {
		log.Printf("[InputSystem] 标记格子占用失败: %v", err)
		s.entityManager.DestroyEntity(plantID)
		s.gameState.AddSun(sunCost)
		return
	}

	// 播放种植音效
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_PLANT")
	}

	// 触发植物卡片冷却
	s.triggerPlantCardCooldown(s.dragPlantType)

	log.Printf("[InputSystem] 拖拽种植完成: PlantType=%v, 位置=(%d, %d)", s.dragPlantType, col, row)
}

// cancelDragPlanting 取消拖拽种植模式
func (s *InputSystem) cancelDragPlanting() {
	if !s.isDragPlanting {
		return
	}

	// 删除预览实体
	s.destroyPlantPreview()

	// 重置植物卡片的选择状态
	if s.dragStartCardEntity != 0 {
		s.resetPlantCardSelection(s.dragPlantType)
	}

	// 退出种植模式
	s.gameState.ExitPlantingMode()

	// 重置拖拽状态
	s.isDragPlanting = false
	s.dragPlantType = 0
	s.dragStartCardEntity = 0

	log.Printf("[InputSystem] 拖拽种植取消")
}
