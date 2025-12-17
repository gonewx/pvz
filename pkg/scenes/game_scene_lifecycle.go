package scenes

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
)

// game_scene_lifecycle.go
// 游戏生命周期管理模块 - 将 Update() 的复杂逻辑拆分为职责清晰的阶段处理函数
//
// 架构说明:
//   - 本文件包含所有游戏生命周期阶段的更新逻辑
//   - 按优先级组织: 对话框 → 暂停 → 开场动画 → 游戏结束 → 正常游戏循环
//   - 每个阶段函数返回 bool 表示是否应继续后续更新
//   - Update() 调用这些函数以减少复杂度

// updateBattleSaveDialog 处理战斗存档对话框逻辑
// 返回 true 表示应继续后续更新，false 表示应停止
func (s *GameScene) updateBattleSaveDialog(deltaTime float64) bool {
	// Story 18.3: 战斗存档对话框优先于所有其他逻辑
	// 如果有战斗存档且对话框未显示，先显示对话框
	if s.hasBattleSave && !s.battleSaveDialogShown {
		s.showBattleSaveDialog()
		s.battleSaveDialogShown = true
		s.updateMouseCursor()
		return false // 等待对话框显示
	}

	// 如果对话框正在显示，只更新对话框系统（所有元素保持静止）
	if s.battleSaveDialogID != 0 {
		if s.dialogInputSystem != nil {
			s.dialogInputSystem.Update(deltaTime)
			s.entityManager.RemoveMarkedEntities()
		}
		// 检查对话框是否已关闭
		if _, ok := ecs.GetComponent[*components.DialogComponent](s.entityManager, s.battleSaveDialogID); !ok {
			s.battleSaveDialogID = 0 // 对话框已关闭
		}
		s.updateMouseCursor()
		return false // 对话框打开时阻止其他更新（类似暂停效果）
	}

	return true // 继续后续更新
}

// updatePauseMenu 处理暂停菜单逻辑
// 返回 true 表示应继续后续更新，false 表示应停止
func (s *GameScene) updatePauseMenu(deltaTime float64) bool {
	// DEBUG: 输出暂停状态（仅当 IsPaused 为 true 时）
	if s.gameState.IsPaused {
		log.Printf("[GameScene] updatePauseMenu: IsPaused=%v, pauseMenuModule=%v",
			s.gameState.IsPaused, s.pauseMenuModule != nil)
	}

	// DEBUG: Check for GameFreezeComponent on every frame to debug freeze issue
	freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](s.entityManager)
	if len(freezeEntities) > 0 && s.zombiesWonPhaseSystem == nil {
		log.Printf("[GameScene] ⚠️ WARNING: GameFreezeComponent found but ZombiesWonPhaseSystem is nil! Count: %d", len(freezeEntities))
	}

	// Story 10.1: 更新暂停菜单模块
	if s.pauseMenuModule != nil {
		s.pauseMenuModule.Update(deltaTime)
	}

	// Story 10.1: Check if game is paused
	if s.gameState.IsPaused {
		// 暂停时只更新 UI 系统（按钮交互、暂停菜单、对话框、滑块、复选框）
		if s.buttonSystem != nil {
			s.buttonSystem.Update(deltaTime)
		}
		// Story 20.5: 暂停菜单中的滑块和复选框交互
		if s.sliderSystem != nil {
			s.sliderSystem.Update(deltaTime)
		}
		if s.checkboxSystem != nil {
			s.checkboxSystem.Update(deltaTime)
		}
		// ✅ ECS 架构修复: 更新对话框输入系统（暂停菜单可能包含对话框）
		if s.dialogInputSystem != nil {
			s.dialogInputSystem.Update(deltaTime)
			s.entityManager.RemoveMarkedEntities()
		}
		// ✅ 暂停时也需要更新鼠标光标（按钮悬停效果）
		s.updateMouseCursor()
		return false // 跳过所有游戏逻辑系统
	}

	return true // 继续后续更新
}

// updateOpeningPhase 处理开场动画阶段的逻辑
// 返回 true 表示应继续后续更新，false 表示应停止
func (s *GameScene) updateOpeningPhase(deltaTime float64) bool {
	// Story 8.2 QA改进：铺草皮动画系统更新（必须在开场动画之前）
	if s.soddingSystem != nil {
		s.soddingSystem.Update(deltaTime)
	}

	// Story 8.3: Check if opening animation is playing
	if s.openingSystem != nil && !s.openingSystem.IsCompleted() {
		// 开场动画期间，只更新镜头系统、开场动画系统和 Reanim 系统（僵尸动画需要）
		s.cameraSystem.Update(deltaTime)
		s.openingSystem.Update(deltaTime)
		s.reanimSystem.Update(deltaTime) // 更新僵尸 idle 动画

		// Story 8.12: 选卡阶段时额外更新选卡系统和按钮系统
		if s.gameState.IsSeedSelectionActive() && s.seedChooserInputSystem != nil {
			s.seedChooserInputSystem.Update(deltaTime)
			// Story 8.12 QA: 选卡阶段也需要更新按钮系统（菜单按钮点击）
			if s.buttonSystem != nil {
				s.buttonSystem.Update(deltaTime)
			}
			// 选卡阶段需要更新鼠标光标（手形光标）
			s.updateMouseCursor()
		}

		// Story 8.12 修复: 选卡确认后、镜头返回期间就创建植物卡片
		// 这样在镜头返回期间，植物选择栏和菜单按钮可以立即显示
		isSeedSelectionLevel := s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.EnableSeedSelection
		if isSeedSelectionLevel && s.gameState.IsSeedSelectionConfirmed() && s.plantSelectionModule == nil {
			selectedPlants := s.gameState.GetSelectedPlants()
			if len(selectedPlants) > 0 {
				log.Printf("[GameScene] 选卡确认后创建植物卡片（镜头返回期间）: %v", selectedPlants)
				s.initPlantCardsAfterSelection()
			}
		}

		// Story 8.12: 选卡确认后更新水平滑动动画、植物选择栏和按钮系统
		if isSeedSelectionLevel && s.gameState.IsSeedSelectionConfirmed() {
			s.updateSeedBankHorizontalSlide(deltaTime)
			// 更新植物选择栏模块（卡片冷却、状态等）
			if s.plantSelectionModule != nil {
				s.plantSelectionModule.Update(deltaTime)
			}
			// Story 8.12 修复: 选卡确认后也需要更新按钮系统（菜单按钮点击）
			if s.buttonSystem != nil {
				s.buttonSystem.Update(deltaTime)
			}
			// 更新鼠标光标
			s.updateMouseCursor()
		}

		// 同步镜头位置到本地 cameraX（用于渲染）
		s.cameraX = s.gameState.CameraX
		return false // 暂停其他游戏系统
	}

	return true // 继续后续更新
}

// updatePostOpeningSetup 处理开场动画完成后的一次性设置
// 返回 skipInputThisFrame 表示当前帧是否应跳过输入处理
func (s *GameScene) updatePostOpeningSetup() bool {
	skipInputThisFrame := false

	// 检查开场动画是否刚被跳过（ESC/Space）
	// 如果是，跳过当前帧的输入处理，防止同一按键触发暂停菜单
	if s.openingSystem != nil && s.openingSystem.WasSkipKeyConsumed() {
		log.Printf("[GameScene] 开场动画跳过按键已消费，当前帧跳过输入处理")
		skipInputThisFrame = true
	}

	// Story 8.12: 选卡完成后创建植物卡片（必须在开场动画完成后检查）
	// 条件：开场动画已完成、启用了选卡系统、植物选择栏模块还未创建、用户已选择植物
	if s.openingSystem != nil && s.openingSystem.IsCompleted() && s.plantSelectionModule == nil {
		if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.EnableSeedSelection {
			selectedPlants := s.gameState.GetSelectedPlants()
			log.Printf("[GameScene] 检查选卡完成条件: EnableSeedSelection=%v, SelectedPlants=%v",
				s.gameState.CurrentLevel.EnableSeedSelection, selectedPlants)
			if len(selectedPlants) > 0 {
				s.initPlantCardsAfterSelection()
			}
		}
	}

	// 开场动画刚完成，触发铺草皮动画（如果配置了且还未启动）
	s.triggerSoddingAnimation()

	return skipInputThisFrame
}

// triggerSoddingAnimation 触发铺草皮动画及相关回调
func (s *GameScene) triggerSoddingAnimation() {
	// 检查条件：开场动画完成、铺草皮动画未启动、系统存在、关卡已加载
	if s.openingSystem == nil || !s.openingSystem.IsCompleted() {
		return
	}
	if s.soddingAnimStarted || s.soddingSystem == nil || s.gameState.CurrentLevel == nil {
		return
	}

	// 实时生成模式：不再预生成僵尸，由 WaveTimingSystem 触发时实时生成

	// 检查是否应该播放铺草皮动画
	shouldPlayAnim := s.gameState.CurrentLevel.ShowSoddingAnim || s.gameState.CurrentLevel.SodRollAnimation

	// Story 8.12 修复：即使不需要铺草皮动画，也要标记为已启动
	// 否则 isWaitingForSodding 会永远为 true，导致 UI 被隐藏
	if !shouldPlayAnim {
		s.soddingAnimStarted = true
		log.Printf("[GameScene] 无需铺草皮动画，直接执行后续步骤")

		// 通知教学系统
		if s.tutorialSystem != nil {
			s.tutorialSystem.OnSoddingComplete()
		}

		// 启动 ReadySetPlant 动画（如果需要）
		s.startReadySetPlantAnimation()

		// 创建除草车
		s.createLawnmowers()
		return
	}

	log.Printf("[GameScene] 开场动画完成，启动铺草皮动画")

	// 启动动画，传递启用的行列表、草皮位置和图片高度
	enabledLanes := s.gameState.CurrentLevel.EnabledLanes
	// Story 8.6 QA修正: 获取需要播放动画的行列表
	animLanes := s.gameState.CurrentLevel.SoddingAnimLanes
	// Story 11.4: 读取粒子特效配置
	enableParticles := s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.SodRollParticles
	s.soddingSystem.StartAnimation(func() {
		// 动画完成回调：通知教学系统可以开始了
		log.Printf("[GameScene] 铺草皮动画完成")

		// Story 11.5 修复：动画完成后处理草皮叠加层
		s.handleSoddingCompletion()

		// 通知教学系统
		if s.tutorialSystem != nil {
			s.tutorialSystem.OnSoddingComplete()
		}

		// Story 8.3: 铺草皮完成后启动 ReadySetPlant 动画（仅 Story 8.3 测试关卡）
		s.startReadySetPlantAnimation()

		// Story 10.6.1: 创建除草车（Story 8.2 修复：在铺草皮完成后创建）
		s.createLawnmowers()
	}, enabledLanes, animLanes, s.sodOverlayX, float64(s.sodHeight), enableParticles)

	// 标记铺草皮动画已启动（防止重复触发）
	s.soddingAnimStarted = true
	log.Printf("[GameScene] 铺草皮动画已触发，标记为已启动")
}

// handleSoddingCompletion 处理铺草皮动画完成后的草皮合并逻辑
func (s *GameScene) handleSoddingCompletion() {
	// Story 11.5 修复：动画完成后处理草皮叠加层
	// 原理：动画期间使用叠加层裁剪显示，完成后将草皮合并到底层背景
	// Story 8.2.1 修复：保留 sodRowImage 用于草坪闪烁效果
	if s.soddedBackground != nil {
		// Level 1-4: 有完整的已铺草皮背景，直接替换
		log.Printf("[GameScene] 替换底层背景: IMAGE_BACKGROUND1UNSODDED → IMAGE_BACKGROUND1")
		s.background = s.soddedBackground
		s.soddedBackground = nil
		s.preSoddedImage = nil
		// Story 8.2.1: 保留 sodRowImage 用于草坪闪烁
		log.Printf("[GameScene] 保留 sodRowImage 用于草坪闪烁效果")
	} else if s.preSoddedImage != nil || s.sodRowImage != nil {
		// Level 1-1, 1-2: 需要将草皮叠加层合并到底层背景
		log.Printf("[GameScene] 合并草皮叠加层到底层背景")
		mergedBg := s.createMergedBackground()
		if mergedBg != nil {
			// 原子操作：先替换背景，再清空preSoddedImage
			s.background = mergedBg
			s.preSoddedImage = nil
			// Story 8.2.1: 保留 sodRowImage 用于草坪闪烁，不清空
			log.Printf("[GameScene] 背景合并完成，保留 sodRowImage 用于草坪闪烁")
		} else {
			// 合并失败，保持叠加层不清空，避免草皮消失
			log.Printf("[GameScene] 警告：合并背景失败，保持叠加层")
		}
	}
}

// startReadySetPlantAnimation 启动 ReadySetPlant 动画
func (s *GameScene) startReadySetPlantAnimation() {
	if s.readySetPlantSystem == nil || s.gameState.CurrentLevel == nil {
		return
	}

	// 检查是否启用 ReadySetPlant 动画
	if !s.gameState.CurrentLevel.ShowReadySetPlant {
		return
	}

	log.Printf("[GameScene] 启动 ReadySetPlant 动画")
	s.readySetPlantSystem.Start()
}

// createLawnmowers 创建除草车
func (s *GameScene) createLawnmowers() {
	if s.lawnmowerSystem == nil || s.gameState.CurrentLevel == nil {
		return
	}

	// 使用原有的初始化方法

	// 调用原有的除草车初始化方法
	s.initLawnmowers()
}

// checkIntroAnimation 检查是否正在播放 intro 动画（遗留代码）
// 返回 true 表示应继续后续更新，false 表示应停止
func (s *GameScene) checkIntroAnimation(deltaTime float64) bool {
	// Handle intro animation
	if s.isIntroAnimPlaying {
		s.updateIntroAnimation(deltaTime)
		// 同步摄像机位置到全局状态（即使在动画期间也保持同步）
		s.gameState.CameraX = s.cameraX
		return false // Don't update game systems during intro animation
	}

	return true // 继续后续更新
}

// syncCameraPosition 同步摄像机位置
func (s *GameScene) syncCameraPosition() {
	// Story 8.8: 如果游戏结束且 ZombiesWonPhaseSystem 激活，则反向同步（让系统控制摄像机）
	if s.gameState.IsGameOver {
		// 游戏结束时，ZombiesWonPhaseSystem 控制摄像机移动
		// 所以需要反向同步：从 GameState.CameraX 更新到 s.cameraX
		s.cameraX = s.gameState.CameraX
	} else {
		// 正常游戏时，GameScene 控制摄像机
		s.gameState.CameraX = s.cameraX
	}
}

// updateGameOverPhase 处理游戏结束阶段的逻辑
// 返回 true 表示应继续后续更新，false 表示应停止
func (s *GameScene) updateGameOverPhase(deltaTime float64) bool {
	// Story 5.5: Check if game is over (win or lose)
	// If game is over, stop updating game systems but allow reward animation to play
	if !s.gameState.IsGameOver {
		return true // 游戏未结束，继续后续更新
	}

	// 游戏结束时仍然更新奖励系统和必要的动画系统
	// 这样玩家可以看到完整的奖励动画流程
	s.rewardSystem.Update(deltaTime)   // 奖励动画系统（卡片包动画）
	s.reanimSystem.Update(deltaTime)   // Reanim 系统（植物卡片动画）
	s.particleSystem.Update(deltaTime) // 粒子系统（光晕效果）

	// Story 10.6: 除草车系统（压扁动画需要继续播放）
	if s.lawnmowerSystem != nil {
		s.lawnmowerSystem.Update(deltaTime)
	}

	// Story 8.8: 僵尸获胜流程需要继续更新
	if s.zombiesWonPhaseSystem != nil {
		s.zombiesWonPhaseSystem.Update(deltaTime)
	}
	// Story 8.8: 触发僵尸需要继续移动（BehaviorSystem 会检测冻结状态）
	s.behaviorSystem.Update(deltaTime)

	// Story 8.8: 游戏结束时也需要更新对话框输入系统（处理按钮点击）
	if s.dialogInputSystem != nil {
		s.dialogInputSystem.Update(deltaTime)
		s.entityManager.RemoveMarkedEntities()
	}
	// 更新鼠标光标（按钮悬停效果）
	s.updateMouseCursor()

	return false // 停止其他游戏系统（僵尸移动、植物攻击等）
}

// updateBackgroundMusic 处理背景音乐播放控制
func (s *GameScene) updateBackgroundMusic() {
	if s.bgmStarted {
		return // 背景音乐已启动
	}

	// 背景音乐播放：在所有开场动画完成后开始播放
	// 检测条件：开场动画完成 + 铺草皮动画完成 + ReadySetPlant动画完成（如果有的话）
	isOpeningComplete := s.openingSystem == nil || s.openingSystem.IsCompleted()
	isSoddingComplete := s.soddingSystem == nil || !s.soddingSystem.IsPlaying()
	isReadySetPlantComplete := s.readySetPlantSystem == nil || !s.readySetPlantSystem.IsPlaying()

	// 所有开场动画完成后开始播放背景音乐
	if isOpeningComplete && isSoddingComplete && isReadySetPlantComplete && s.seedBankSlideInCompleted {
		if audioManager := s.gameState.GetAudioManager(); audioManager != nil {
			audioManager.PlayMusic("SOUND_MAINMUSIC")
			log.Printf("[GameScene] 开始播放背景音乐")
		}
		s.bgmStarted = true
	}
}

// updateGameplaySystems 更新所有核心游戏系统
func (s *GameScene) updateGameplaySystems(deltaTime, cameraX float64, skipInputThisFrame bool) {
	// Update all ECS systems in order (order matters for correct game logic)
	s.levelSystem.Update(deltaTime)                       // 0. Update level system (Story 5.5: wave spawning, victory/defeat)
	s.waveSpawnSystem.UpdatePendingActivations(deltaTime) // 0.05. Update pending zombie activations (散落入场效果)
	s.rewardSystem.Update(deltaTime)                      // 0.1. Update reward animation system (Story 8.3: 卡片包动画)
	s.finalWaveWarningSystem.Update(deltaTime)            // 0.2. Update final wave warning (Story 11.3: 自动清理提示动画)
	if s.zombiesWonPhaseSystem != nil {
		s.zombiesWonPhaseSystem.Update(deltaTime) // 0.3. Update zombies won flow (Story 8.8: 僵尸获胜四阶段流程)
	}
	s.zombieLaneTransitionSystem.Update(deltaTime) // 0.5. Update zombie lane transitions (move to target lane before attacking)

	// 植物选择栏滑入动画更新
	// 在开场动画、铺草皮动画、除草车入场动画、ReadySetPlant 动画全部完成后启动
	s.updateSeedBankSlideIn(deltaTime)

	// Story 8.12: 选卡关卡水平滑动动画更新
	// 选卡确认后，植物选择栏从选卡位置水平滑动到正常游戏位置
	s.updateSeedBankHorizontalSlide(deltaTime)

	// Story 3.1 架构优化：使用模块化方式更新植物选择栏
	if s.plantSelectionModule != nil {
		s.plantSelectionModule.Update(deltaTime) // 1. Update plant card states (before input)
	}

	// Story 19.2: 铲子槽位点击检测（在输入系统之前，优先处理铲子模式切换）
	s.updateShovelSlotClick()

	// Story 19.5: 传送带卡片点击检测
	s.updateConveyorBeltClick()

	// Story 19.2: 如果处于铲子模式，更新铲子交互系统
	if s.shovelInteractionSystem != nil && s.shovelSelected {
		s.shovelInteractionSystem.Update(deltaTime, cameraX)
	}

	// 2. Process player input (highest priority, 传递摄像机位置)
	// 如果开场动画跳过按键刚被消费，跳过输入处理（防止 ESC 同时跳过动画和触发暂停）
	if !skipInputThisFrame {
		s.inputSystem.Update(deltaTime, cameraX)
	}

	// 3. Generate new suns
	// 教学关卡：在第一次收集阳光后启用自动生成（由 TutorialSystem 控制）
	// 非教学关卡：始终启用自动生成
	s.sunSpawnSystem.Update(deltaTime)

	s.sunMovementSystem.Update(deltaTime)   // 4. Move suns (includes collection animation)
	s.sunCollectionSystem.Update(deltaTime) // 5. Check if collection is complete

	// Story 8.9: 撑杆僵尸跳跃系统（必须在 BehaviorSystem 之前，检测植物触发跳跃）
	if s.poleVaultSystem != nil {
		s.poleVaultSystem.Update(deltaTime) // 5.5. Check pole vault jumps
	}

	s.behaviorSystem.Update(deltaTime) // 6. Update plant behaviors (Story 3.4)
	// Story 10.2: Update lawnmower system (除草车系统)
	if s.lawnmowerSystem != nil {
		s.lawnmowerSystem.Update(deltaTime) // 6.5. Check lawnmower triggers and move lawnmowers
	}
	s.physicsSystem.Update(deltaTime) // 7. Check collisions (Story 4.3)

	// Story 8.9: 减速效果系统（处理冰豌豆减速效果持续时间）
	if s.slowEffectSystem != nil {
		s.slowEffectSystem.Update(deltaTime) // 7.5. Update slow effects
	}

	// Story 8.11: 大嘴花系统（处理吞噬/撕咬/消化逻辑）
	if s.chomperSystem != nil {
		s.chomperSystem.Update(deltaTime) // 7.6. Update chomper states
	}

	// Story 6.3: Reanim 动画系统（替代旧的 AnimationSystem）
	s.reanimSystem.Update(deltaTime) // 8. Update Reanim animation frames

	// Story 7.3: 粒子系统（种植效果、爆炸等）
	if s.particleSystem != nil {
		s.particleSystem.Update(deltaTime) // 8.1. Update particle emitters and particles
	}
	// 闪烁效果系统（僵尸受击闪烁）
	if s.flashEffectSystem != nil {
		s.flashEffectSystem.Update(deltaTime) // 8.2. Update flash effect duration
	}
	// Story 8.3: ReadySetPlant 动画系统（铺草皮完成后播放）
	if s.readySetPlantSystem != nil {
		s.readySetPlantSystem.Update(deltaTime) // 8.5. Update ReadySetPlant animation duration
	}
	// Story 11.6: 更新 Dave 对话系统
	if s.daveDialogueSystem != nil {
		s.daveDialogueSystem.Update(deltaTime) // 9.10. Update Dave dialogue
	}
	// Story 8.2: 更新教学系统（Level 1-1, 1-2 的教学引导）
	if s.tutorialSystem != nil {
		s.tutorialSystem.Update(deltaTime) // 9.10.1. Update tutorial (text, arrows, card highlight)
	}
	// Story 19.3: 更新强引导教学系统（监控植物数量变化、空闲提示箭头）
	if s.guidedTutorialSystem != nil {
		s.guidedTutorialSystem.Update(deltaTime) // 9.11. Update guided tutorial (plant count monitoring)
	}
	// Story 19.4: 更新阶段转场系统（铲子教学 → 保龄球阶段转场）
	if s.levelPhaseSystem != nil {
		s.levelPhaseSystem.Update(deltaTime) // 9.12. Update level phase transitions
	}
	// Story 19.5: 更新传送带系统（生成坚果卡片、移动卡片）
	if s.conveyorBeltSystem != nil {
		s.conveyorBeltSystem.Update(deltaTime) // 9.13. Update conveyor belt card generation and movement
	}
	// Story 19.6: 更新保龄球坚果系统（坚果滚动、碰撞检测）
	if s.bowlingNutSystem != nil {
		s.bowlingNutSystem.Update(deltaTime) // 9.14. Update bowling nut rolling and collision
	}
	// 僵尸呻吟音效系统 - 环境音效
	if s.zombieGroanSystem != nil {
		s.zombieGroanSystem.Update(deltaTime)
	}
	// Story 3.2: 植物预览系统 - 更新预览位置（双图像支持）
	s.plantPreviewSystem.Update(deltaTime) // 10. Update plant preview position (dual-image support)
	s.lawnGridSystem.Update(deltaTime)     // 10.5. Update lawn flash animation (Story 8.2)
	// ECS 按钮系统更新（交互检测）
	if s.buttonSystem != nil {
		s.buttonSystem.Update(deltaTime) // 10.7. Update button interactions (hover, click)
	}
	s.lifetimeSystem.Update(deltaTime)     // 11. Check for expired entities
	s.entityManager.RemoveMarkedEntities() // 12. Clean up deleted entities (always last)

	// 13. Update mouse cursor based on component states
	s.updateMouseCursor()
}
