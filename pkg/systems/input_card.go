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

// ========================================
// 植物卡片点击、悬停和快捷键处理
// ========================================

// handlePlantCardClick 处理植物卡片点击逻辑
// 返回 true 表示处理了点击，false 表示未处理
// Story 19.3: 强引导模式下阻止卡片点击
func (s *InputSystem) handlePlantCardClick(mouseX, mouseY int, cameraX float64) bool {
	// Story 19.3: 检查强引导模式是否阻止卡片点击
	if IsGuidedTutorialBlocking("click_plant_card") {
		return false // 静默忽略，不处理卡片点击
	}
	// 查询所有植物卡片实体
	cardEntities := ecs.GetEntitiesWith4[
		*components.PlantCardComponent,
		*components.ClickableComponent,
		*components.PositionComponent,
		*components.UIComponent,
	](s.entityManager)

	// DEBUG: 卡片点击检查日志（每次点击都打印会刷屏，已禁用）
	// log.Printf("[InputSystem] 检查植物卡片点击: 鼠标(%d, %d), 找到 %d 个卡片", mouseX, mouseY, len(cardEntities))

	// 遍历卡片实体，检测点击
	for _, entityID := range cardEntities {
		// 跳过奖励卡片（奖励卡片不应被点击选择）
		if _, isRewardCard := ecs.GetComponent[*components.RewardCardComponent](s.entityManager, entityID); isRewardCard {
			continue
		}

		// 获取组件
		card, _ := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, entityID)
		clickable, _ := ecs.GetComponent[*components.ClickableComponent](s.entityManager, entityID)
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		ui, _ := ecs.GetComponent[*components.UIComponent](s.entityManager, entityID)

		log.Printf("[InputSystem] 卡片 %d: PlantType=%v, 位置=(%.1f, %.1f), 可点击区域=%.1fx%.1f, IsAvailable=%v, IsEnabled=%v",
			entityID, card.PlantType, pos.X, pos.Y, clickable.Width, clickable.Height, card.IsAvailable, clickable.IsEnabled)

		// AABB 碰撞检测（先检测点击，再判断卡片状态）
		if float64(mouseX) >= pos.X && float64(mouseX) <= pos.X+clickable.Width &&
			float64(mouseY) >= pos.Y && float64(mouseY) <= pos.Y+clickable.Height {
			// 点击命中卡片！
			log.Printf("[InputSystem] 点击植物卡片: PlantType=%v, IsPlantingMode=%v, IsAvailable=%v",
				card.PlantType, s.gameState.IsPlantingMode, card.IsAvailable)

			// Story 10.8: 检查卡片状态
			currentSun := s.gameState.GetSun()
			plantCost := card.SunCost

			// 1. 冷却中 - 不做任何反应（除非启用忽略冷却模式）
			if card.CurrentCooldown > 0 && !s.ignoreCooldown {
				log.Printf("[InputSystem] 卡片冷却中，跳过 (剩余 %.1f 秒)", card.CurrentCooldown)
				return true // 已处理点击，但卡片冷却中
			}

			// 2. 阳光不足 - 触发闪烁反馈
			if currentSun < plantCost {
				log.Printf("[InputSystem] 阳光不足: 需要 %d, 当前 %d", plantCost, currentSun)
				s.gameState.TriggerSunFlash()

				// Story 10.8: 播放无效操作音效（带冷却，使用 AudioManager - Story 10.9）
				timeSinceLastBuzzer := s.gameTime - s.lastBuzzerPlayTime
				if timeSinceLastBuzzer >= s.buzzerCooldownTime {
					if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
						audioManager.PlaySound("SOUND_BUZZER")
						s.lastBuzzerPlayTime = s.gameTime
						log.Printf("[InputSystem] 播放无效操作音效 (buzzer)")
					}
				} else {
					log.Printf("[InputSystem] 无效操作音效冷却中 (%.2f/%.2f)", timeSinceLastBuzzer, s.buzzerCooldownTime)
				}

				return true // 已处理点击，但阻止卡片选择
			}

			// 3. 其他不可用原因 - 跳过
			if !clickable.IsEnabled {
				log.Printf("[InputSystem] 卡片不可点击，跳过")
				return true
			}

			// 检查当前是否已在种植模式
			if s.gameState.IsPlantingMode {
				// 如果已在种植模式，点击卡片退出种植模式
				log.Printf("[InputSystem] 退出种植模式（点击卡片）")
				s.gameState.ExitPlantingMode()
				s.destroyPlantPreview()
				// 可选：设置卡片状态为 Normal
				ui.State = components.UINormal
			} else {
				// 如果不在种植模式，进入种植模式
				log.Printf("[InputSystem] 进入种植模式: PlantType=%v", card.PlantType)

				// Bug修复: 如果当前处于铲子模式，先取消铲子模式
				if shovelStateProvider != nil && shovelStateProvider.IsShovelSelected() {
					shovelStateProvider.SetShovelSelected(false)
					// 恢复系统光标
					ebiten.SetCursorMode(ebiten.CursorModeVisible)
					log.Printf("[InputSystem] 取消铲子模式（选择植物卡片）")
				}

				s.gameState.EnterPlantingMode(card.PlantType)

				// Story 10.9: 播放选中植物卡片音效 (seedlift.ogg)
				if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
					audioManager.PlaySound("SOUND_SEEDLIFT")
				}

				// 创建植物预览实体（转换为世界坐标）
				mouseWorldX := float64(mouseX) + cameraX
				mouseWorldY := float64(mouseY)
				s.createPlantPreview(card.PlantType, mouseWorldX, mouseWorldY)

				// 可选：设置卡片状态为 Clicked（视觉反馈）
				ui.State = components.UIClicked
			}

			return true // 已处理点击
		}
	}

	return false // 未处理任何卡片点击
}

// createPlantPreview 创建植物预览实体（复用植物卡片的渲染逻辑）
// 使用与植物卡片相同的 Reanim 离屏渲染技术,确保预览和卡片显示一致
func (s *InputSystem) createPlantPreview(plantType components.PlantType, x, y float64) {
	// 先删除现有预览
	s.destroyPlantPreview()

	// 使用 RenderPlantIcon 复用卡片渲染逻辑（直接传入 plantType）
	plantIcon, err := entities.RenderPlantIcon(s.entityManager, s.resourceManager, s.reanimSystem, plantType)
	if err != nil {
		log.Printf("[InputSystem] Failed to render plant icon for preview: %v", err)
		return
	}

	// 创建预览实体
	entityID := s.entityManager.CreateEntity()

	// 添加位置组件
	ecs.AddComponent(s.entityManager, entityID, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// 添加静态精灵组件（使用渲染好的图标）
	ecs.AddComponent(s.entityManager, entityID, &components.SpriteComponent{
		Image: plantIcon,
	})

	// 添加植物预览组件
	ecs.AddComponent(s.entityManager, entityID, &components.PlantPreviewComponent{
		PlantType: plantType,
		Alpha:     0.5, // 半透明效果
	})

	log.Printf("[InputSystem] Created plant preview (ID: %d, Type: %v) at (%.1f, %.1f)",
		entityID, plantType, x, y)
}

// destroyPlantPreview 删除所有植物预览实体
func (s *InputSystem) destroyPlantPreview() {
	// 查询所有拥有 PlantPreviewComponent 的实体
	previewEntities := ecs.GetEntitiesWith1[*components.PlantPreviewComponent](s.entityManager)

	// 删除所有预览实体
	for _, entityID := range previewEntities {
		s.entityManager.DestroyEntity(entityID)
		log.Printf("[InputSystem] Destroyed plant preview entity (ID: %d)", entityID)
	}

	// 立即清理标记删除的实体
	s.entityManager.RemoveMarkedEntities()
}

// triggerPlantCardCooldown 触发指定植物类型的卡片进入冷却状态
func (s *InputSystem) triggerPlantCardCooldown(plantType components.PlantType) {
	// 忽略冷却模式下不触发冷却
	if s.ignoreCooldown {
		return
	}

	// 查询所有植物卡片实体
	cardEntities := ecs.GetEntitiesWith1[*components.PlantCardComponent](s.entityManager)

	// 找到匹配的卡片并触发冷却
	for _, entityID := range cardEntities {
		card, _ := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, entityID)

		if card.PlantType == plantType {
			// 触发冷却
			card.CurrentCooldown = card.CooldownTime
			log.Printf("[InputSystem] 触发植物卡片冷却: PlantType=%v, CooldownTime=%.1f", plantType, card.CooldownTime)
			break // 只触发第一个匹配的卡片
		}
	}
}

// resetPlantCardSelection 重置指定植物类型的卡片选择状态
func (s *InputSystem) resetPlantCardSelection(plantType components.PlantType) {
	// 查询所有植物卡片实体
	cardEntities := ecs.GetEntitiesWith2[
		*components.PlantCardComponent,
		*components.UIComponent,
	](s.entityManager)

	// 找到匹配的卡片并重置UI状态
	for _, entityID := range cardEntities {
		card, _ := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, entityID)
		ui, _ := ecs.GetComponent[*components.UIComponent](s.entityManager, entityID)

		if card.PlantType == plantType {
			// 重置为正常状态
			ui.State = components.UINormal
			log.Printf("[InputSystem] 重置植物卡片选择状态: PlantType=%v", plantType)
			break // 只重置第一个匹配的卡片
		}
	}
}

// updatePlantCardHover 更新植物卡片的悬停状态
// Story 8.2.1: 每帧检测鼠标是否悬停在植物卡片上，设置 UIComponent.State 为 UIHovered
// Story 10.8: 添加 Tooltip 显示和鼠标光标切换
func (s *InputSystem) updatePlantCardHover() {
	mouseX, mouseY := utils.GetPointerPosition()

	// 查询所有植物卡片实体
	cardEntities := ecs.GetEntitiesWith4[
		*components.PlantCardComponent,
		*components.ClickableComponent,
		*components.PositionComponent,
		*components.UIComponent,
	](s.entityManager)

	// 记录是否有卡片被悬停
	hoveredEntityID := ecs.EntityID(0)
	var hoveredCard *components.PlantCardComponent

	// 遍历所有卡片，检测悬停
	for _, entityID := range cardEntities {
		// 跳过奖励卡片
		if _, isRewardCard := ecs.GetComponent[*components.RewardCardComponent](s.entityManager, entityID); isRewardCard {
			continue
		}

		// 获取组件
		card, _ := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, entityID)
		clickable, _ := ecs.GetComponent[*components.ClickableComponent](s.entityManager, entityID)
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		ui, _ := ecs.GetComponent[*components.UIComponent](s.entityManager, entityID)

		// AABB 碰撞检测
		isHovering := float64(mouseX) >= pos.X && float64(mouseX) <= pos.X+clickable.Width &&
			float64(mouseY) >= pos.Y && float64(mouseY) <= pos.Y+clickable.Height

		// Story 10.8: 悬停任何卡片都显示 Tooltip（不管是否可用）
		if isHovering {
			hoveredEntityID = entityID
			hoveredCard = card

			// 只有可用的卡片才显示悬停效果（UI状态变化）
			if card.IsAvailable && clickable.IsEnabled {
				// 只有当前状态不是 Clicked 时才设置为 Hovered（避免覆盖点击状态）
				if ui.State != components.UIClicked {
					ui.State = components.UIHovered
				}
			}
			break // 找到悬停卡片后停止遍历
		} else {
			// 如果不再悬停且当前是 Hovered 状态，恢复为 Normal
			if ui.State == components.UIHovered {
				if card.IsAvailable {
					ui.State = components.UINormal
				} else {
					ui.State = components.UIDisabled
				}
			}
		}
	}

	// Story 10.8: 更新 Tooltip（鼠标光标由 GameScene.updateMouseCursor() 统一管理）
	if hoveredEntityID != 0 {
		// 有卡片被悬停: 显示 Tooltip
		s.showTooltip(hoveredEntityID, hoveredCard)
	} else {
		// 没有卡片被悬停: 隐藏 Tooltip
		s.hideTooltip()
	}
}

// showTooltip 显示 Tooltip
// Story 10.8: 鼠标悬停植物卡片时显示提示信息
func (s *InputSystem) showTooltip(cardEntityID ecs.EntityID, card *components.PlantCardComponent) {
	// 获取或创建 Tooltip 实体
	if s.tooltipEntity == 0 {
		s.tooltipEntity = s.entityManager.CreateEntity()
		tooltip := components.NewTooltipComponent()
		s.entityManager.AddComponent(s.tooltipEntity, tooltip)
	}

	// 获取 Tooltip 组件
	tooltip, ok := ecs.GetComponent[*components.TooltipComponent](s.entityManager, s.tooltipEntity)
	if !ok {
		log.Printf("[InputSystem] 警告: Tooltip 组件不存在")
		return
	}

	// 获取卡片位置信息
	pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, cardEntityID)
	if !ok {
		return
	}
	clickable, ok := ecs.GetComponent[*components.ClickableComponent](s.entityManager, cardEntityID)
	if !ok {
		return
	}

	// 更新 Tooltip 内容
	tooltip.IsVisible = true
	tooltip.TargetEntity = cardEntityID
	tooltip.PlantName = s.getPlantName(card.PlantType)

	// 设置状态提示文本
	// 优先级：冷却中 > 阳光不足 > 可用
	currentSun := s.gameState.GetSun()
	if card.CurrentCooldown > 0 {
		tooltip.StatusText = "重新装填中..."
	} else if currentSun < card.SunCost {
		tooltip.StatusText = "没有足够的阳光"
	} else {
		tooltip.StatusText = "" // 可用状态不显示状态提示
	}

	// 计算 Tooltip 位置（卡片下方居中，因为卡片在屏幕顶部）
	// tooltip.X: 卡片中心 X 坐标
	// tooltip.Y: 卡片底部 Y 坐标
	tooltip.X = pos.X + clickable.Width/2
	tooltip.Y = pos.Y + clickable.Height // 卡片底部 Y 坐标
}

// hideTooltip 隐藏 Tooltip
// Story 10.8: 鼠标离开卡片时隐藏提示信息
func (s *InputSystem) hideTooltip() {
	if s.tooltipEntity == 0 {
		return
	}

	tooltip, ok := ecs.GetComponent[*components.TooltipComponent](s.entityManager, s.tooltipEntity)
	if ok {
		tooltip.IsVisible = false
	}
}

// isCardClickable 检测卡片是否可点击
// Story 10.8: 判断卡片是否处于可点击状态（决定鼠标光标样式）
func (s *InputSystem) isCardClickable(card *components.PlantCardComponent) bool {
	cooldownOk := card.CurrentCooldown <= 0 || s.ignoreCooldown
	return card.IsAvailable && cooldownOk && s.gameState.Sun >= card.SunCost
}

// getPlantName 获取植物名称
// Story 10.8: 根据植物类型返回中文名称
func (s *InputSystem) getPlantName(plantType components.PlantType) string {
	switch plantType {
	case components.PlantPeashooter:
		return "豌豆射手"
	case components.PlantSunflower:
		return "向日葵"
	case components.PlantCherryBomb:
		return "樱桃炸弹"
	case components.PlantWallnut:
		return "坚果墙"
	default:
		return "未知植物"
	}
}

// handlePlantCardHotkeys 处理植物卡片快捷键（数字键 1-9）
// 按卡片在卡槽中的位置从左到右对应数字键 1-9
func (s *InputSystem) handlePlantCardHotkeys(cameraX float64) {
	// 定义快捷键映射
	hotkeys := []ebiten.Key{
		ebiten.Key1, ebiten.Key2, ebiten.Key3,
		ebiten.Key4, ebiten.Key5, ebiten.Key6,
		ebiten.Key7, ebiten.Key8, ebiten.Key9,
	}

	// 检测按下的数字键
	pressedIndex := -1
	for i, key := range hotkeys {
		if inpututil.IsKeyJustPressed(key) {
			pressedIndex = i
			break
		}
	}

	if pressedIndex < 0 {
		return // 没有按下快捷键
	}

	// 查询所有植物卡片实体，按 X 坐标排序
	cardEntities := ecs.GetEntitiesWith4[
		*components.PlantCardComponent,
		*components.ClickableComponent,
		*components.PositionComponent,
		*components.UIComponent,
	](s.entityManager)

	// 过滤并收集非奖励卡片
	type cardInfo struct {
		entityID ecs.EntityID
		posX     float64
		card     *components.PlantCardComponent
		ui       *components.UIComponent
	}
	var cards []cardInfo

	for _, entityID := range cardEntities {
		// 跳过奖励卡片
		if _, isRewardCard := ecs.GetComponent[*components.RewardCardComponent](s.entityManager, entityID); isRewardCard {
			continue
		}

		card, _ := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, entityID)
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		ui, _ := ecs.GetComponent[*components.UIComponent](s.entityManager, entityID)

		cards = append(cards, cardInfo{
			entityID: entityID,
			posX:     pos.X,
			card:     card,
			ui:       ui,
		})
	}

	// 按 X 坐标排序（从左到右）
	for i := 0; i < len(cards)-1; i++ {
		for j := i + 1; j < len(cards); j++ {
			if cards[i].posX > cards[j].posX {
				cards[i], cards[j] = cards[j], cards[i]
			}
		}
	}

	// 检查索引是否有效
	if pressedIndex >= len(cards) {
		log.Printf("[InputSystem] 快捷键 %d 无效：只有 %d 张卡片", pressedIndex+1, len(cards))
		return
	}

	// 获取对应卡片
	targetCard := cards[pressedIndex]

	log.Printf("[InputSystem] 快捷键选择卡片: 按键=%d, 植物类型=%v", pressedIndex+1, targetCard.card.PlantType)

	// 检查卡片状态（与鼠标点击逻辑保持一致）
	// 1. 冷却中 - 不做任何反应（除非启用忽略冷却模式）
	if targetCard.card.CurrentCooldown > 0 && !s.ignoreCooldown {
		log.Printf("[InputSystem] 卡片冷却中，跳过 (剩余 %.1f 秒)", targetCard.card.CurrentCooldown)
		return
	}

	// 2. 阳光不足 - 触发闪烁反馈
	currentSun := s.gameState.GetSun()
	plantCost := targetCard.card.SunCost
	if currentSun < plantCost {
		log.Printf("[InputSystem] 阳光不足: 需要 %d, 当前 %d", plantCost, currentSun)
		s.gameState.TriggerSunFlash()

		// 播放无效操作音效（带冷却）
		timeSinceLastBuzzer := s.gameTime - s.lastBuzzerPlayTime
		if timeSinceLastBuzzer >= s.buzzerCooldownTime {
			if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
				audioManager.PlaySound("SOUND_BUZZER")
				s.lastBuzzerPlayTime = s.gameTime
			}
		}
		return
	}

	// 3. 卡片不可用 - 跳过
	if !targetCard.card.IsAvailable {
		log.Printf("[InputSystem] 卡片不可用，跳过")
		return
	}

	// 检查当前是否已在种植模式
	if s.gameState.IsPlantingMode {
		// 如果选择的是同一种植物，退出种植模式
		_, currentPlantType := s.gameState.GetPlantingMode()
		if currentPlantType == targetCard.card.PlantType {
			log.Printf("[InputSystem] 快捷键退出种植模式（重复选择同一卡片）")
			s.gameState.ExitPlantingMode()
			s.destroyPlantPreview()
			targetCard.ui.State = components.UINormal
			return
		}
		// 如果选择不同植物，切换到新植物
		log.Printf("[InputSystem] 快捷键切换种植模式: %v -> %v", currentPlantType, targetCard.card.PlantType)
		s.destroyPlantPreview()
	}

	// Bug修复: 如果当前处于铲子模式，先取消铲子模式
	if shovelStateProvider != nil && shovelStateProvider.IsShovelSelected() {
		shovelStateProvider.SetShovelSelected(false)
		// 恢复系统光标
		ebiten.SetCursorMode(ebiten.CursorModeVisible)
		log.Printf("[InputSystem] 快捷键取消铲子模式（选择植物卡片）")
	}

	// 进入种植模式
	log.Printf("[InputSystem] 快捷键进入种植模式: PlantType=%v", targetCard.card.PlantType)
	s.gameState.EnterPlantingMode(targetCard.card.PlantType)

	// 播放选中植物卡片音效
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_SEEDLIFT")
	}

	// 创建植物预览实体（使用鼠标位置）
	mouseX, mouseY := utils.GetPointerPosition()
	mouseWorldX := float64(mouseX) + cameraX
	mouseWorldY := float64(mouseY)
	s.createPlantPreview(targetCard.card.PlantType, mouseWorldX, mouseWorldY)

	// 设置卡片状态为 Clicked（视觉反馈）
	targetCard.ui.State = components.UIClicked
}

// getPlantCardAtPosition 获取指定位置的植物卡片
// 返回卡片实体ID和卡片组件，如果位置上没有卡片则返回 0 和 nil
func (s *InputSystem) getPlantCardAtPosition(screenX, screenY int) (ecs.EntityID, *components.PlantCardComponent) {
	// 查询所有植物卡片实体
	cardEntities := ecs.GetEntitiesWith4[
		*components.PlantCardComponent,
		*components.ClickableComponent,
		*components.PositionComponent,
		*components.UIComponent,
	](s.entityManager)

	for _, entityID := range cardEntities {
		// 跳过奖励卡片
		if _, isRewardCard := ecs.GetComponent[*components.RewardCardComponent](s.entityManager, entityID); isRewardCard {
			continue
		}

		// 获取组件
		card, _ := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, entityID)
		clickable, _ := ecs.GetComponent[*components.ClickableComponent](s.entityManager, entityID)
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)

		// AABB 碰撞检测
		if float64(screenX) >= pos.X && float64(screenX) <= pos.X+clickable.Width &&
			float64(screenY) >= pos.Y && float64(screenY) <= pos.Y+clickable.Height {
			return entityID, card
		}
	}

	return 0, nil
}
