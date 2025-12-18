package systems

import (
	"log"
	"math"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ============================================================================
// 奖励动画阶段更新方法
// ============================================================================

// updateAppearingPhase 处理卡片包弹出阶段（0.4秒）。
// - 缩放动画：0.5 → 0.8
// - 微小抛物线：从草坪弹起，轻微上升后落回原位，无弹跳
func (ras *RewardAnimationSystem) updateAppearingPhase(dt float64, rewardComp *components.RewardAnimationComponent) {
	// 获取位置组件
	posComp, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.rewardEntity)
	if !ok {
		return
	}

	// 计算进度（0.0 - 1.0）
	progress := rewardComp.ElapsedTime / RewardAppearDuration
	if progress > 1.0 {
		progress = 1.0
	}

	// Phase 1 不需要缩放动画，保持标准卡片大小（config.PlantCardScale = 0.50）
	// Scale 已在 TriggerReward 中设置，这里无需修改

	// 计算Y轴位置（微小抛物线）
	// 使用 sin 曲线：从起点向上跳跃，到达最高点，然后平滑落回起点
	// sin(0) = 0（起点），sin(PI/2) = 1（最高点），sin(PI) = 0（落回起点）
	// 向上为负方向（Y轴向下为正）
	yOffset := -RewardAppearJumpHeight * math.Sin(progress*math.Pi)

	posComp.Y = rewardComp.StartY + yOffset

	// 检查完成
	if rewardComp.ElapsedTime >= RewardAppearDuration {
		log.Printf("[RewardAnimationSystem] Phase 1 (appearing) 完成，切换到 waiting")
		rewardComp.Phase = "waiting"
		rewardComp.ElapsedTime = 0

		// 根据奖励类型创建不同的 waiting 阶段粒子效果
		// - 植物奖励：SeedPacket（光晕 + 向下箭头）
		// - 工具奖励：AwardPickupArrow（向下箭头）
		// - 来信奖励：AwardPickupArrow（与工具奖励一致）
		if ras.glowEntity == 0 {
			var particleEffectName string
			if rewardComp.RewardType == "tool" || rewardComp.RewardType == "note" {
				particleEffectName = "AwardPickupArrow" // 工具/来信使用向下箭头
			} else {
				particleEffectName = "SeedPacket" // 植物使用光晕+箭头
			}

			// 计算粒子位置（卡片视觉中心，基于当前缩放）
			cardComp, _ := ecs.GetComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity)
			particleX := posComp.X
			particleY := posComp.Y
			if cardComp != nil && cardComp.BackgroundImage != nil {
				// 使用当前缩放计算视觉中心
				cardWidth := float64(cardComp.BackgroundImage.Bounds().Dx()) * cardComp.CardScale
				cardHeight := float64(cardComp.BackgroundImage.Bounds().Dy()) * cardComp.CardScale
				particleX += cardWidth / 2.0
				particleY += cardHeight / 2.0
			}

			glowID, err := entities.CreateParticleEffect(
				ras.entityManager,
				ras.resourceManager,
				particleEffectName,
				particleX,
				particleY,
				0.0,  // angleOffset = 0
				true, // isUIParticle = true
			)
			if err != nil {
				log.Printf("[RewardAnimationSystem] 创建 waiting 粒子失败（%s）: %v", particleEffectName, err)
			} else {
				ras.glowEntity = glowID
				log.Printf("[RewardAnimationSystem] 创建 waiting 粒子成功: %s, ID=%d", particleEffectName, glowID)
			}
		}
	}
}

// updateWaitingPhase 处理等待阶段。
// - 卡片包静止，显示光晕和箭头粒子效果
// - 等待玩家点击卡包本身或按空格键后进入展开阶段（Story 8.13）
// - 界面其他元素保持可操作状态
func (ras *RewardAnimationSystem) updateWaitingPhase(dt float64, rewardComp *components.RewardAnimationComponent) {
	// 检测玩家点击（只响应卡包点击或空格键 - Story 8.13）
	justPressed, mouseX, mouseY := utils.IsPointerJustPressed()
	spacePressed := inpututil.IsKeyJustPressed(ebiten.KeySpace)

	// 只有点击卡包本身或按空格键才进入下一阶段
	cardClicked := justPressed && ras.isCardPackClicked(float64(mouseX), float64(mouseY))

	if cardClicked || spacePressed {
		log.Printf("[RewardAnimationSystem] 玩家点击卡片包，切换到 expanding")

		// 使用 AudioManager 统一管理音效（Story 10.9）
		if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
			// 播放点击卡包音效
			audioManager.PlaySound("SOUND_TAP2")
			// 播放胜利音乐
			audioManager.PlaySound("SOUND_WINMUSIC")
		}

		// 清理 Phase 2 的光晕粒子（AwardPickupArrow）
		ras.cleanupGlowParticles()

		// Story 8.14: 来信奖励点击后直接消失，不需要移动动画
		if rewardComp.RewardType == "note" {
			log.Printf("[RewardAnimationSystem] 来信奖励：直接消失并播放 Starburst")

			// 获取位置用于粒子效果
			posComp, _ := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.rewardEntity)
			particleX, particleY := posComp.X, posComp.Y

			// 保存奖励信息
			ras.currentRewardType = rewardComp.RewardType
			ras.currentNoteID = rewardComp.NoteID

			// 立即清理来信实体（直接消失）
			if ras.rewardEntity != 0 {
				ras.entityManager.DestroyEntity(ras.rewardEntity)
				ras.entityManager.RemoveMarkedEntities()
				ras.rewardEntity = 0
			}

			// 创建 Starburst 粒子效果
			starburstID, err := entities.CreateParticleEffect(
				ras.entityManager,
				ras.resourceManager,
				"Starburst",
				particleX,
				particleY,
				0.0,  // angleOffset = 0
				true, // isUIParticle = true
			)
			if err != nil {
				log.Printf("[RewardAnimationSystem] 创建 Starburst 粒子失败: %v", err)
			} else {
				ras.glowEntity = starburstID
				log.Printf("[RewardAnimationSystem] 创建 Starburst 粒子成功, ID=%d", starburstID)
			}

			// 直接进入 fadingOut 阶段
			ras.currentPhase = "fadingOut"
			ras.phaseElapsed = 0
			return
		}

		// 植物/工具奖励：正常进入 expanding 阶段
		rewardComp.Phase = "expanding"
		rewardComp.ElapsedTime = 0

		// 立即创建 Award 粒子特效（点击后立即出现）
		ras.createAwardParticle(rewardComp)
	}
}

// updateExpandingPhase 处理卡片展开阶段（2秒）。
// - 植物/工具奖励：Award.xml 粒子特效播放（12个光芒发射器跟随卡片移动）
// - 来信奖励：Starburst 粒子特效播放（Story 8.14）
// - 卡片/工具放大到目标缩放
// - 移动到目标位置
func (ras *RewardAnimationSystem) updateExpandingPhase(dt float64, rewardComp *components.RewardAnimationComponent) {
	// 获取位置组件
	posComp, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.rewardEntity)
	if !ok {
		return
	}

	// 计算进度（0.0 - 1.0）
	progress := rewardComp.ElapsedTime / RewardExpandDuration
	if progress > 1.0 {
		progress = 1.0
	}

	// 应用缓动函数（easeOutQuad）
	easedProgress := easeOutQuad(progress)

	// 根据奖励类型确定缩放的起始值和目标值
	var startScale, endScale float64
	if rewardComp.RewardType == "tool" {
		startScale = config.ShovelRewardStartScale // 约 0.56
		endScale = config.ShovelRewardEndScale     // 1.0
	} else {
		startScale = config.PlantCardScale // 0.50
		endScale = RewardExpandScaleEnd    // 1.0
	}

	// 更新缩放（从起始缩放放大到目标缩放）
	rewardComp.Scale = startScale + (endScale-startScale)*easedProgress

	// 同步缩放到 PlantCardComponent（Story 8.4）
	if cardComp, ok := ecs.GetComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity); ok {
		cardComp.CardScale = rewardComp.Scale
	}

	// 更新位置（从当前位置移动到目标位置）
	// 注意：起始Y坐标已经包含了抛物线结束时的偏移
	posComp.X = rewardComp.StartX + (rewardComp.TargetX-rewardComp.StartX)*easedProgress
	posComp.Y = rewardComp.StartY + (rewardComp.TargetY-rewardComp.StartY)*easedProgress

	// 调试：每0.5秒输出一次当前位置
	if int(rewardComp.ElapsedTime*2)%1 == 0 && rewardComp.ElapsedTime > 0.01 {
		log.Printf("[Expanding] 进度: %.2f%%, 当前位置: (%.1f, %.1f), 当前缩放: %.2f",
			progress*100, posComp.X, posComp.Y, rewardComp.Scale)
	}

	// 检查完成
	if rewardComp.ElapsedTime >= RewardExpandDuration {
		log.Printf("[RewardAnimationSystem] Phase 3 (expanding) 移动完成")
		log.Printf("[RewardAnimationSystem] 最终位置: (%.1f, %.1f), 最终缩放: %.2f", posComp.X, posComp.Y, rewardComp.Scale)
		log.Printf("[RewardAnimationSystem] 目标位置: (%.1f, %.1f)", rewardComp.TargetX, rewardComp.TargetY)

		// Story 8.14: 来信奖励使用不同的流程
		if rewardComp.RewardType == "note" {
			// 来信奖励：清理卡包实体，进入 fadingOut 阶段
			log.Printf("[RewardAnimationSystem] 来信奖励：进入 fadingOut 阶段")

			// 清理粒子效果
			ras.cleanupGlowParticles()

			// 保存奖励信息
			ras.currentRewardType = rewardComp.RewardType
			ras.currentNoteID = rewardComp.PlantID // PlantID 字段存储了 noteID

			// 清理卡包实体
			if ras.rewardEntity != 0 {
				ras.entityManager.DestroyEntity(ras.rewardEntity)
				ras.entityManager.RemoveMarkedEntities()
				ras.rewardEntity = 0
			}

			// 进入 fadingOut 阶段
			ras.currentPhase = "fadingOut"
			ras.phaseElapsed = 0
			return
		}

		// Phase 3 完成，进入短暂停顿阶段（让玩家看清楚卡片包到达目标位置）
		rewardComp.Phase = "pausing"
		rewardComp.ElapsedTime = 0
		log.Printf("[RewardAnimationSystem] Phase 3 完成，进入 pausing 阶段（短暂停顿）")
	}
}

// updatePausingPhase 处理短暂停顿阶段。
// - 卡片包静止在目标位置，展示 Award 粒子特效
// - 持续时间由 config.RewardPausingDuration 配置
// - 然后自动进入消失阶段
func (ras *RewardAnimationSystem) updatePausingPhase(dt float64, rewardComp *components.RewardAnimationComponent) {
	// 检查完成
	if rewardComp.ElapsedTime >= config.RewardPausingDuration {
		log.Printf("[RewardAnimationSystem] 停顿完成，进入 disappearing 阶段")
		rewardComp.Phase = "disappearing"
		rewardComp.ElapsedTime = 0
	}
}

// updateDisappearingPhase 处理卡片包消失阶段。
// - 卡片包淡出（透明度 1.0 → 0.0）
// - 持续时间由 config.RewardDisappearDuration 配置
// - 消失完成后，立即进入 Phase 4 显示面板
// - 注意：Award 粒子在此阶段保持显示，直到面板完全显示后才清理
func (ras *RewardAnimationSystem) updateDisappearingPhase(dt float64, rewardComp *components.RewardAnimationComponent) {
	// 计算透明度（从 1.0 淡出到 0.0）
	progress := rewardComp.ElapsedTime / config.RewardDisappearDuration
	if progress > 1.0 {
		progress = 1.0
	}
	alpha := 1.0 - progress

	// 更新卡片透明度（通过 PlantCardComponent.Alpha）
	if cardComp, ok := ecs.GetComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity); ok {
		cardComp.Alpha = alpha
	}

	// 检查完成
	if rewardComp.ElapsedTime >= config.RewardDisappearDuration {
		log.Printf("[RewardAnimationSystem] 卡片包消失完成，进入面板显示阶段")

		// 保存奖励信息（在删除组件前）
		rewardType := rewardComp.RewardType
		plantID := rewardComp.PlantID
		toolID := rewardComp.ToolID

		// 重要：先更新组件状态，避免下一帧重复进入此阶段
		rewardComp.Phase = "showing"
		rewardComp.ElapsedTime = 0

		// 隐藏卡片包（移除 PlantCardComponent，使其不被渲染）
		if ras.rewardEntity != 0 {
			if _, hasCard := ecs.GetComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity); hasCard {
				ecs.RemoveComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity)
			}
		}

		// 注意：不在此处清理 Award 粒子！
		// 粒子将在 showing 阶段面板淡入完成后清理，避免背景闪烁

		// 立即移除所有标记的实体
		ras.entityManager.RemoveMarkedEntities()

		// 进入 Phase 4 (showing)
		// 注意：不再使用rewardComp，因为rewardEntity的PlantCardComponent已被移除
		// 直接设置系统内部状态
		ras.currentPhase = "showing"
		ras.phaseElapsed = 0
		ras.currentPlantID = plantID // 保存植物ID（向后兼容）

		// 创建奖励面板（传递完整奖励信息）
		// 面板初始 FadeAlpha=0，将逐渐淡入
		ras.createRewardPanel(rewardType, plantID, toolID)
		log.Printf("[RewardAnimationSystem] 进入 Phase 4 (showing)，显示奖励面板（粒子保持显示直到面板淡入完成）")
	}
}

// updateShowingPhaseInternal 处理显示奖励面板阶段（内部版本，不依赖rewardComp）。
// - 面板淡入动画（透明度 0.0 → 1.0，持续 config.RewardPanelFadeInDuration）
// - 淡入完成后清理粒子效果，然后等待玩家点击"下一关"或"主菜单"按钮
func (ras *RewardAnimationSystem) updateShowingPhaseInternal(dt float64) {
	ras.phaseElapsed += dt

	// 更新面板透明度（淡入动画）
	if ras.panelEntity != 0 {
		panelComp, ok := ecs.GetComponent[*components.RewardPanelComponent](ras.entityManager, ras.panelEntity)
		if ok {
			// 更新动画时间
			panelComp.AnimationTime += dt

			// 计算淡入进度（0.0 → 1.0）
			progress := panelComp.AnimationTime / config.RewardPanelFadeInDuration
			if progress > 1.0 {
				progress = 1.0
			}

			// 应用淡入效果
			panelComp.FadeAlpha = progress

			// 当面板完全显示后（淡入完成），清理粒子效果
			// 这样可以避免背景闪烁问题
			if progress >= 1.0 && ras.glowEntity != 0 {
				log.Printf("[RewardAnimationSystem] 面板淡入完成，清理 Award 粒子效果")
				ras.cleanupGlowParticles()
			}
		}
	}

	// 检测点击（支持触摸和鼠标）
	justPressed, _, _ := utils.IsPointerJustPressed()
	if justPressed {
		// 获取鼠标点击位置
		mouseX, mouseY := utils.GetPointerPosition()

		// 检查是否点击了"下一关"按钮
		if ras.isNextLevelButtonClicked(float64(mouseX), float64(mouseY)) {
			log.Printf("[RewardAnimationSystem] 玩家点击\"下一关\"按钮，准备切换场景")

			// 播放点击按钮音效（使用 AudioManager 统一管理 - Story 10.9）
			if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
				audioManager.PlaySound("SOUND_TAP2")
			}

			ras.currentPhase = "closing"
			ras.phaseElapsed = 0
			return // 避免同时处理多个按钮
		}

		// 检查是否点击了"主菜单"按钮（Story 8.13）
		if ras.isMainMenuButtonClicked(float64(mouseX), float64(mouseY)) {
			log.Printf("[RewardAnimationSystem] 玩家点击\"主菜单\"按钮，准备返回主菜单")

			// 播放点击按钮音效
			if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
				audioManager.PlaySound("SOUND_TAP2")
			}

			// 直接切换到主菜单（不需要进入 closing 阶段）
			ras.cleanupAndSwitchToMainMenu()
			return
		}
		// 注意：空格键不再触发关闭，只有点击按钮才能继续
	}
}

// isNextLevelButtonClicked 检查鼠标点击是否在"下一关"按钮范围内
func (ras *RewardAnimationSystem) isNextLevelButtonClicked(mouseX, mouseY float64) bool {
	// 计算按钮位置（与 reward_panel_render_system.go 中的 drawNextLevelButton 逻辑一致）
	bgWidth := config.RewardPanelBackgroundWidth
	bgHeight := config.RewardPanelBackgroundHeight
	offsetX := (ras.screenWidth - bgWidth) / 2
	offsetY := (ras.screenHeight - bgHeight) / 2

	// 按钮中心位置
	buttonX := offsetX + bgWidth/2
	buttonY := offsetY + bgHeight*config.RewardPanelButtonY

	// 按钮尺寸（根据 IMAGE_SEEDCHOOSER_BUTTON 的尺寸）
	// SeedChooser_Button.png 尺寸约为 155x38（需要验证）
	// 为了更容易点击，使用稍大的点击区域
	buttonWidth := 155.0
	buttonHeight := 50.0 // 增大高度以提高可点击性

	// AABB 碰撞检测
	halfWidth := buttonWidth / 2
	halfHeight := buttonHeight / 2

	if mouseX >= buttonX-halfWidth && mouseX <= buttonX+halfWidth &&
		mouseY >= buttonY-halfHeight && mouseY <= buttonY+halfHeight {
		log.Printf("[RewardAnimationSystem] 按钮点击命中! 鼠标=(%.1f, %.1f), 按钮中心=(%.1f, %.1f)", mouseX, mouseY, buttonX, buttonY)
		return true
	}

	return false
}

// isMainMenuButtonClicked 检查鼠标点击是否在"主菜单"按钮范围内（Story 8.13）
func (ras *RewardAnimationSystem) isMainMenuButtonClicked(mouseX, mouseY float64) bool {
	// 首先检查面板是否存在且配置显示主菜单按钮
	if ras.panelEntity == 0 {
		return false // 面板不存在
	}

	panelComp, ok := ecs.GetComponent[*components.RewardPanelComponent](ras.entityManager, ras.panelEntity)
	if !ok || !panelComp.ShowMainMenuButton {
		return false // 面板组件不存在或主菜单按钮未启用
	}

	// 计算按钮位置（与 reward_panel_render_system.go 中的 drawMainMenuButton 逻辑一致）
	bgWidth := config.RewardPanelBackgroundWidth
	bgHeight := config.RewardPanelBackgroundHeight
	offsetX := (ras.screenWidth - bgWidth) / 2
	offsetY := (ras.screenHeight - bgHeight) / 2

	// 按钮中心位置（右上角）
	buttonX := offsetX + bgWidth*config.RewardPanelMainMenuButtonX
	buttonY := offsetY + bgHeight*config.RewardPanelMainMenuButtonY

	// 按钮尺寸（根据 IMAGE_SEEDCHOOSER_BUTTON2 的尺寸，约 65x28）
	buttonWidth := 65.0
	buttonHeight := 40.0 // 增大高度以提高可点击性

	// AABB 碰撞检测
	halfWidth := buttonWidth / 2
	halfHeight := buttonHeight / 2

	if mouseX >= buttonX-halfWidth && mouseX <= buttonX+halfWidth &&
		mouseY >= buttonY-halfHeight && mouseY <= buttonY+halfHeight {
		log.Printf("[RewardAnimationSystem] 主菜单按钮点击命中! 鼠标=(%.1f, %.1f), 按钮中心=(%.1f, %.1f)", mouseX, mouseY, buttonX, buttonY)
		return true
	}

	return false
}

// cleanupAndSwitchToMainMenu 清理资源并切换到主菜单（Story 8.13）
func (ras *RewardAnimationSystem) cleanupAndSwitchToMainMenu() {
	log.Printf("[RewardAnimationSystem] 清理资源并切换到主菜单")

	// 清理奖励实体
	if ras.rewardEntity != 0 {
		ras.entityManager.DestroyEntity(ras.rewardEntity)
	}
	if ras.panelEntity != 0 {
		ras.entityManager.DestroyEntity(ras.panelEntity)
	}
	if ras.glowEntity != 0 {
		ras.entityManager.DestroyEntity(ras.glowEntity)
	}

	ras.rewardEntity = 0
	ras.panelEntity = 0
	ras.glowEntity = 0
	ras.isActive = false
	ras.currentPhase = ""

	// 切换到主菜单
	if ras.sceneManager != nil {
		ras.sceneManager.SwitchToMainMenu()
	} else {
		log.Printf("[RewardAnimationSystem] SceneManager 为 nil，跳过主菜单切换（可能在测试环境）")
	}

	log.Printf("[RewardAnimationSystem] 已返回主菜单")
}

// updateClosingPhaseInternal 处理关闭奖励面板阶段（0.5秒，内部版本）。
// - 淡出动画
// - 清理实体
// - 切换到下一关（或返回主菜单）
func (ras *RewardAnimationSystem) updateClosingPhaseInternal(dt float64) {
	ras.phaseElapsed += dt

	// 更新面板淡出效果（从 1.0 淡出到 0.0）
	// 这样在 closing 阶段渲染面板时会有淡出效果，避免菜单按钮闪烁
	if ras.panelEntity != 0 {
		if panelComp, ok := ecs.GetComponent[*components.RewardPanelComponent](ras.entityManager, ras.panelEntity); ok {
			fadeProgress := ras.phaseElapsed / RewardFadeOutDuration
			if fadeProgress > 1.0 {
				fadeProgress = 1.0
			}
			panelComp.FadeAlpha = 1.0 - fadeProgress
		}
	}

	// 检查完成
	if ras.phaseElapsed >= RewardFadeOutDuration {
		log.Printf("[RewardAnimationSystem] Phase 5 (closing) 完成，清理实体")

		// 清理奖励实体
		if ras.rewardEntity != 0 {
			ras.entityManager.DestroyEntity(ras.rewardEntity)
		}
		if ras.panelEntity != 0 {
			ras.entityManager.DestroyEntity(ras.panelEntity)
		}
		if ras.glowEntity != 0 {
			ras.entityManager.DestroyEntity(ras.glowEntity)
		}

		ras.rewardEntity = 0
		ras.panelEntity = 0
		ras.glowEntity = 0
		ras.isActive = false
		ras.currentPhase = ""

		// 切换到下一关
		nextLevelID := ras.gameState.GetNextLevelID()
		if nextLevelID != "" {
			log.Printf("[RewardAnimationSystem] 切换到下一关: %s", nextLevelID)
			// 检查 sceneManager 是否存在（测试程序可能传入 nil）
			if ras.sceneManager != nil {
				ras.sceneManager.LoadLevel(nextLevelID)
			} else {
				log.Printf("[RewardAnimationSystem] SceneManager 为 nil，跳过场景切换（可能在测试环境）")
			}
		} else {
			log.Printf("[RewardAnimationSystem] 没有下一关，返回主菜单")
			if ras.sceneManager != nil {
				ras.sceneManager.SwitchToMainMenu()
			} else {
				log.Printf("[RewardAnimationSystem] SceneManager 为 nil，跳过主菜单切换（可能在测试环境）")
			}
		}

		log.Printf("[RewardAnimationSystem] 奖励动画完成")
	}
}

// createRewardPanel 创建奖励面板实体。
func (ras *RewardAnimationSystem) createRewardPanel(rewardType string, plantID string, toolID string) {
	ras.panelEntity = ras.entityManager.CreateEntity()

	// 播放雷声音效（奖励面板显示时的震撼效果，使用 AudioManager - Story 10.9）
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_THUNDER")
	}

	var rewardName, rewardDesc string
	var sunCost int

	// 根据奖励类型加载不同信息
	if rewardType == "tool" {
		// 工具奖励信息 - 从 LawnStrings 加载
		if toolID == "shovel" {
			// 从 LawnStrings 加载铲子文本
			rewardName = "铁铲"                 // 默认值
			rewardDesc = "让你挖出一株植物，腾出空间给其他植物" // 默认值

			if ras.gameState.LawnStrings != nil {
				// 加载铲子名称
				if name := ras.gameState.LawnStrings.GetString("SHOVEL"); name != "" {
					rewardName = name
				}
				// 加载铲子描述
				if desc := ras.gameState.LawnStrings.GetString("SHOVEL_DESCRIPTION"); desc != "" {
					rewardDesc = desc
				}
			}
		} else {
			// 未知工具，使用默认文本
			rewardName = "新工具"
			rewardDesc = "一个实用的工具！"
		}
		sunCost = 0 // 工具无阳光消耗
	} else {
		// 植物奖励信息
		plantInfo := ras.gameState.GetPlantUnlockManager().GetPlantInfo(plantID)

		// 从 LawnStrings 加载实际文本
		rewardName = ras.gameState.LawnStrings.GetString(plantInfo.NameKey)
		rewardDesc = ras.gameState.LawnStrings.GetString(plantInfo.DescriptionKey)

		// 从植物注册表获取阳光值
		sunCost = game.GetPlantSunCost(plantID)
	}

	// 添加 RewardPanelComponent
	// Story 8.13: 从当前关卡配置获取 ShowMainMenuButton 设置
	showMainMenuButton := true // 默认显示
	if levelConfig := ras.gameState.CurrentLevel; levelConfig != nil {
		if levelConfig.ShowMainMenuButton != nil {
			showMainMenuButton = *levelConfig.ShowMainMenuButton
		}
	}

	ecs.AddComponent(ras.entityManager, ras.panelEntity, &components.RewardPanelComponent{
		RewardType:         rewardType,         // 新增：奖励类型
		PlantID:            plantID,            // 设置 PlantID，让渲染系统自动加载图标
		ToolID:             toolID,             // 新增：工具ID
		PlantName:          rewardName,         // 名称（植物或工具）
		PlantDescription:   rewardDesc,         // 描述（植物或工具）
		SunCost:            sunCost,            // 设置阳光值（工具为0）
		CardScale:          1.0,                // 卡片固定大小，不做动画
		FadeAlpha:          0.0,                // 初始完全透明，用于淡入动画
		ShowMainMenuButton: showMainMenuButton, // Story 8.13: 是否显示主菜单按钮
		// 卡片位置由 RewardPanelRenderSystem 自动计算（水平居中）
		IsVisible:     true,
		AnimationTime: 0.0, // 动画时间计数器，用于淡入效果
	})

	log.Printf("[RewardAnimationSystem] 奖励面板已创建：%s - %s", rewardName, rewardDesc)
}

// ============================================================================
// Story 8.14: 来信奖励专属阶段
// ============================================================================

// updateFadingOutPhaseInternal 处理淡出阶段（Story 8.14）
// - 画面渐暗到黑色（0.5秒）
// - 淡出完成后创建来信面板并进入 fadingIn 阶段
func (ras *RewardAnimationSystem) updateFadingOutPhaseInternal(dt float64) {
	ras.phaseElapsed += dt

	// 计算淡出进度（0.0 → 1.0）
	progress := ras.phaseElapsed / config.ZombieNoteFadeOutDuration
	if progress > 1.0 {
		progress = 1.0
	}

	// 保存淡出透明度（用于 Draw 方法绘制黑色遮罩）
	// 通过 GameState 临时存储，因为此阶段没有 rewardComp
	ras.gameState.NoteFadeAlpha = float32(progress)

	// 检查完成
	if ras.phaseElapsed >= config.ZombieNoteFadeOutDuration {
		log.Printf("[RewardAnimationSystem] fadingOut 完成，创建来信面板并进入 fadingIn 阶段")

		// 创建来信面板模块
		ras.createNotePanelModule()

		// 进入 fadingIn 阶段
		ras.currentPhase = "fadingIn"
		ras.phaseElapsed = 0
	}
}

// updateFadingInPhaseInternal 处理淡入阶段（Story 8.14）
// - 来信面板渐显（0.5秒）
// - 淡入完成后进入 showing 阶段
func (ras *RewardAnimationSystem) updateFadingInPhaseInternal(dt float64) {
	ras.phaseElapsed += dt

	// 计算淡入进度（0.0 → 1.0）
	progress := ras.phaseElapsed / config.ZombieNoteFadeInDuration
	if progress > 1.0 {
		progress = 1.0
	}

	// 更新来信面板透明度
	if ras.notePanelModule != nil {
		ras.notePanelModule.SetFadeAlpha(progress)
	}

	// 黑色遮罩逐渐消失（1.0 → 0.0）
	ras.gameState.NoteFadeAlpha = float32(1.0 - progress)

	// 检查完成
	if ras.phaseElapsed >= config.ZombieNoteFadeInDuration {
		log.Printf("[RewardAnimationSystem] fadingIn 完成，进入 showing 阶段（来信面板）")

		// 进入 showing 阶段
		ras.currentPhase = "showing"
		ras.phaseElapsed = 0

		// 显示面板
		if ras.notePanelModule != nil {
			ras.notePanelModule.Show()
		}
	}
}

// createNotePanelModule 创建来信面板模块（Story 8.14）
func (ras *RewardAnimationSystem) createNotePanelModule() {
	if ras.notePanelModule != nil {
		// 如果已存在，先清理
		ras.notePanelModule.Cleanup()
		ras.notePanelModule = nil
	}

	log.Printf("[RewardAnimationSystem] 创建来信面板模块，noteID: %s", ras.currentNoteID)

	// 检查工厂函数是否已注入
	if ras.notePanelFactory == nil {
		log.Printf("[RewardAnimationSystem] 来信面板工厂未注入，跳过面板创建")
		ras.cleanupNotePanelAndSwitchToNextLevel()
		return
	}

	// 使用工厂函数创建来信面板
	var err error
	ras.notePanelModule, err = ras.notePanelFactory(
		ras.currentNoteID,
		func() {
			// "下一关"按钮回调
			log.Printf("[RewardAnimationSystem] 来信面板：点击下一关")
			ras.cleanupNotePanelAndSwitchToNextLevel()
		},
		func() {
			// "主菜单"按钮回调
			log.Printf("[RewardAnimationSystem] 来信面板：点击主菜单")
			ras.cleanupNotePanelAndSwitchToMainMenu()
		},
	)

	if err != nil {
		log.Printf("[RewardAnimationSystem] 创建来信面板失败: %v", err)
		// 出错时直接切换到下一关
		ras.cleanupNotePanelAndSwitchToNextLevel()
		return
	}

	// 初始透明度为 0（将在 fadingIn 阶段逐渐增加）
	ras.notePanelModule.SetFadeAlpha(0.0)
}

// cleanupNotePanelAndSwitchToNextLevel 清理来信面板并切换到下一关（Story 8.14）
func (ras *RewardAnimationSystem) cleanupNotePanelAndSwitchToNextLevel() {
	log.Printf("[RewardAnimationSystem] 清理来信面板并切换到下一关")

	// 先切换场景，防止在切换前的渲染帧中游戏场景 UI 闪烁
	// 注意：场景切换后，整个 GameScene（包括 RewardAnimationSystem）会被销毁
	// 所以不需要手动清理面板和重置状态
	nextLevelID := ras.gameState.GetNextLevelID()
	if nextLevelID != "" {
		log.Printf("[RewardAnimationSystem] 切换到下一关: %s", nextLevelID)
		if ras.sceneManager != nil {
			ras.sceneManager.LoadLevel(nextLevelID)
		}
	} else {
		log.Printf("[RewardAnimationSystem] 没有下一关，返回主菜单")
		if ras.sceneManager != nil {
			ras.sceneManager.SwitchToMainMenu()
		}
	}
}

// cleanupNotePanelAndSwitchToMainMenu 清理来信面板并切换到主菜单（Story 8.14）
func (ras *RewardAnimationSystem) cleanupNotePanelAndSwitchToMainMenu() {
	log.Printf("[RewardAnimationSystem] 清理来信面板并切换到主菜单")

	// 先切换场景，防止在切换前的渲染帧中游戏场景 UI 闪烁
	// 注意：场景切换后，整个 GameScene（包括 RewardAnimationSystem）会被销毁
	// 所以不需要手动清理面板和重置状态
	if ras.sceneManager != nil {
		ras.sceneManager.SwitchToMainMenu()
	}
}
