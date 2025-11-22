package systems

import (
	"log"
	"math"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// 僵尸获胜流程常量
const (
	// Phase 1: 游戏冻结阶段持续时间（秒）
	Phase1FreezeDuration = 1.5

	// Phase 2: 摄像机移动目标位置（世界坐标）
	Phase2CameraTargetX = 0.0

	// Phase 2: 摄像机移动速度（像素/秒）
	// 假设从标准游戏位置（~400）移动到 0，约 2-3 秒完成
	Phase2CameraMoveSpeed = 200.0

	// Phase 2: 僵尸目标位置（房子门口）
	Phase2ZombieTargetX = 100.0
	Phase2ZombieTargetY = 350.0

	// Phase 3: 咀嚼音效延迟时间（相对于惨叫音效）
	Phase3ChompDelay = 0.5

	// Phase 3: 惨叫动画总持续时间（秒）
	Phase3AnimationDuration = 3.5

	// Phase 3: 屏幕抖动参数
	ScreenShakeAmplitude = 5.0  // 振幅（像素）
	ScreenShakeFrequency = 50.0 // 频率（Hz）
	ScreenShakeDuration  = 2.5  // 抖动持续时间（秒）

	// Phase 4: 等待玩家点击的超时时间（秒）
	Phase4WaitTimeout = 5.0

	// 僵尸触发失败的边界X坐标（进入 Phase 1）
	DefeatBoundaryX = 250.0
)

// ZombiesWonPhaseSystem 僵尸获胜流程系统
//
// 管理四阶段僵尸获胜流程：
// - Phase 1: 游戏冻结（1.5秒）
// - Phase 2: 僵尸入侵动画（僵尸走出屏幕）
// - Phase 3: 惨叫与"吃脑子"动画（3-4秒）
// - Phase 4: 游戏结束对话框
//
// 架构说明：
// - 通过组件驱动（检测 ZombiesWonPhaseComponent）
// - 使用 GameFreezeComponent 标记游戏冻结状态
// - 遵循 ECS 零耦合原则
type ZombiesWonPhaseSystem struct {
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager
	gameState       *game.GameState

	// 窗口尺寸（用于对话框居中）
	windowWidth  int
	windowHeight int

	// 用于 Phase 4 的回调（由 GameScene 提供）
	onRetryCallback     func() // "再次尝试"回调
	onMainMenuCallback  func() // "返回主菜单"回调（可选，验证程序不需要）

	// 保存原始摄像机位置（用于屏幕抖动后恢复）
	originalCameraX float64

	// ZombiesWon 动画实体 ID（用于抖动效果）
	zombiesWonEntityID ecs.EntityID
}

// NewZombiesWonPhaseSystem 创建僵尸获胜流程系统
func NewZombiesWonPhaseSystem(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	gs *game.GameState,
	windowWidth, windowHeight int,
) *ZombiesWonPhaseSystem {
	return &ZombiesWonPhaseSystem{
		entityManager:   em,
		resourceManager: rm,
		gameState:       gs,
		windowWidth:     windowWidth,
		windowHeight:    windowHeight,
		originalCameraX: 0,
	}
}

// SetRetryCallback 设置"再次尝试"回调（由 GameScene 调用）
func (s *ZombiesWonPhaseSystem) SetRetryCallback(callback func()) {
	s.onRetryCallback = callback
}

// SetMainMenuCallback 设置"返回主菜单"回调（由 GameScene 调用，验证程序可选）
func (s *ZombiesWonPhaseSystem) SetMainMenuCallback(callback func()) {
	s.onMainMenuCallback = callback
}

// Update 更新僵尸获胜流程
func (s *ZombiesWonPhaseSystem) Update(deltaTime float64) {
	// 查询所有 ZombiesWonPhaseComponent 实体
	phaseEntities := ecs.GetEntitiesWith1[*components.ZombiesWonPhaseComponent](s.entityManager)
	if len(phaseEntities) == 0 {
		return // 没有激活的僵尸获胜流程
	}

	// DEBUG: 确认 Update 被调用
	log.Printf("[ZombiesWonPhaseSystem] Update called, %d phase entities", len(phaseEntities))

	for _, entityID := range phaseEntities {
		phaseComp, ok := ecs.GetComponent[*components.ZombiesWonPhaseComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// DEBUG: 输出当前状态
		log.Printf("[ZombiesWonPhaseSystem] Entity %d: Phase %d, Timer %.2f", entityID, phaseComp.CurrentPhase, phaseComp.PhaseTimer)

		// 更新阶段计时器
		phaseComp.PhaseTimer += deltaTime

		switch phaseComp.CurrentPhase {
		case 1: // Phase 1: 游戏冻结
			s.updatePhase1Freeze(entityID, phaseComp, deltaTime)
		case 2: // Phase 2: 僵尸入侵
			s.updatePhase2ZombieEntry(entityID, phaseComp, deltaTime)
		case 3: // Phase 3: 惨叫与动画
			s.updatePhase3ScreamAnimation(entityID, phaseComp, deltaTime)
		case 4: // Phase 4: 游戏结束对话框
			s.updatePhase4Dialog(entityID, phaseComp, deltaTime)
		}
	}
}

// updatePhase1Freeze Phase 1: 游戏冻结
func (s *ZombiesWonPhaseSystem) updatePhase1Freeze(
	entityID ecs.EntityID,
	phaseComp *components.ZombiesWonPhaseComponent,
	deltaTime float64,
) {
	// 检测是否已经添加了 GameFreezeComponent
	freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](s.entityManager)
	if len(freezeEntities) == 0 {
		// 这种情况不应该发生（LevelSystem 应该已经添加了）
		log.Printf("[ZombiesWonPhaseSystem] Warning: GameFreezeComponent not found in Phase 1")
	}

	// 淡出背景音乐（仅第一次执行）
	if phaseComp.PhaseTimer == deltaTime { // 刚进入 Phase 1
		// TODO: Task 7 - 实现背景音乐淡出
		// s.resourceManager.FadeOutMusic(0.2)
		log.Printf("[ZombiesWonPhaseSystem] Phase 1: Game frozen, music fade out (TODO)")
	}

	// 延迟 1.5 秒后进入 Phase 2
	if phaseComp.PhaseTimer >= Phase1FreezeDuration {
		log.Printf("[ZombiesWonPhaseSystem] Phase 1 -> Phase 2 (zombie entry)")
		phaseComp.CurrentPhase = 2
		phaseComp.PhaseTimer = 0.0

		// 保存原始摄像机位置（用于 Phase 3 屏幕抖动后恢复）
		s.originalCameraX = s.gameState.CameraX
	}
}

// updatePhase2ZombieEntry Phase 2: 僵尸入侵动画（两步执行）
// 第一步：摄像机移动到目标位置
// 第二步：僵尸行走到目标位置
func (s *ZombiesWonPhaseSystem) updatePhase2ZombieEntry(
	entityID ecs.EntityID,
	phaseComp *components.ZombiesWonPhaseComponent,
	deltaTime float64,
) {
	// 第一次进入 Phase 2：保存初始摄像机位置，冻结僵尸移动
	if phaseComp.PhaseTimer == deltaTime { // 刚进入 Phase 2
		phaseComp.InitialCameraX = s.gameState.CameraX
		log.Printf("[ZombiesWonPhaseSystem] Phase 2 started: Initial camera X=%.2f", phaseComp.InitialCameraX)

		// 冻结触发僵尸的移动（保存速度，然后设为 0）
		if velComp, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, phaseComp.TriggerZombieID); ok {
			// 注意：不需要保存速度，因为僵尸的标准移动速度是常量 -150.0
			velComp.VX = 0
			velComp.VY = 0
			log.Printf("[ZombiesWonPhaseSystem] Phase 2: Frozen zombie movement (zombie ID=%d)", phaseComp.TriggerZombieID)
		}
	}

	// 第一步：摄像机平滑移动到目标位置（世界坐标 0）
	if !phaseComp.CameraMovedToTarget {
		currentCameraX := s.gameState.CameraX

		log.Printf("[ZombiesWonPhaseSystem] Phase 2 Step 1: Camera X=%.2f, Target=%.2f",
			currentCameraX, Phase2CameraTargetX)

		// 计算移动方向（向左移动，即减小 CameraX）
		if currentCameraX > Phase2CameraTargetX {
			// 计算本帧移动距离
			moveDistance := Phase2CameraMoveSpeed * deltaTime

			// 计算新位置
			newCameraX := currentCameraX - moveDistance

			log.Printf("[ZombiesWonPhaseSystem] Phase 2: Moving camera from %.2f to %.2f (distance=%.2f)",
				currentCameraX, newCameraX, moveDistance)

			// 防止超调：如果移动后小于目标位置，则直接设为目标位置
			if newCameraX <= Phase2CameraTargetX {
				newCameraX = Phase2CameraTargetX
				phaseComp.CameraMovedToTarget = true
				log.Printf("[ZombiesWonPhaseSystem] Camera reached target position X=%.2f", newCameraX)
			}

			// 更新摄像机位置
			s.gameState.CameraX = newCameraX
		} else {
			// 已经在目标位置或更左侧
			phaseComp.CameraMovedToTarget = true
			log.Printf("[ZombiesWonPhaseSystem] Camera already at or past target")
		}
		return // 继续等待摄像机到位
	}

	// 第二步：摄像机到位后，僵尸开始行走到目标位置
	if !phaseComp.ZombieStartedWalking {
		phaseComp.ZombieStartedWalking = true
		log.Printf("[ZombiesWonPhaseSystem] Phase 2 Step 2: Zombie starts walking to target position (%.2f, %.2f)",
			Phase2ZombieTargetX, Phase2ZombieTargetY)

		// 获取僵尸当前位置
		posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, phaseComp.TriggerZombieID)
		if !ok {
			return
		}

		// 恢复触发僵尸的移动速度（X 和 Y 同时移动）
		if velComp, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, phaseComp.TriggerZombieID); ok {
			velComp.VX = -150.0 // 僵尸标准移动速度（向左）

			// 根据当前 Y 和目标 Y 决定 Y 方向速度
			deltaY := Phase2ZombieTargetY - posComp.Y
			if deltaY > 0 {
				velComp.VY = 50.0 // 向下移动（加速到 50.0，原来是 5.0）
				log.Printf("[ZombiesWonPhaseSystem] Phase 2: Zombie moving (VX=-150.0, VY=50.0 downward)")
			} else if deltaY < 0 {
				velComp.VY = -50.0 // 向上移动（加速到 -50.0）
				log.Printf("[ZombiesWonPhaseSystem] Phase 2: Zombie moving (VX=-150.0, VY=-50.0 upward)")
			} else {
				velComp.VY = 0 // 已经在目标 Y
				log.Printf("[ZombiesWonPhaseSystem] Phase 2: Zombie moving (VX=-150.0, VY=0)")
			}
		}
	}

	// 检查僵尸是否到达目标位置
	posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, phaseComp.TriggerZombieID)
	if !ok {
		// 僵尸实体已被删除（不应该发生）
		log.Printf("[ZombiesWonPhaseSystem] Warning: Trigger zombie entity %d not found", phaseComp.TriggerZombieID)
		// 强制进入 Phase 3
		phaseComp.CurrentPhase = 3
		phaseComp.PhaseTimer = 0.0
		phaseComp.ZombieReachedTarget = true
		return
	}

	log.Printf("[ZombiesWonPhaseSystem] Phase 2: Zombie position (%.2f, %.2f), Target (%.2f, %.2f)",
		posComp.X, posComp.Y, Phase2ZombieTargetX, Phase2ZombieTargetY)

	// 判断是否到达目标位置（使用容差）
	const xTolerance = 5.0
	const yTolerance = 5.0

	// 检查每个方向是否到达，并停止已到达方向的移动
	velComp, hasVel := ecs.GetComponent[*components.VelocityComponent](s.entityManager, phaseComp.TriggerZombieID)

	xReached := posComp.X <= Phase2ZombieTargetX+xTolerance && posComp.X >= Phase2ZombieTargetX-xTolerance
	yReached := posComp.Y <= Phase2ZombieTargetY+yTolerance && posComp.Y >= Phase2ZombieTargetY-yTolerance

	// X 方向到达，停止 X 移动
	if xReached && hasVel && velComp.VX != 0 {
		velComp.VX = 0
		log.Printf("[ZombiesWonPhaseSystem] Phase 2: X reached target (%.2f), stopped X movement", posComp.X)
	}

	// Y 方向到达，停止 Y 移动
	if yReached && hasVel && velComp.VY != 0 {
		velComp.VY = 0
		log.Printf("[ZombiesWonPhaseSystem] Phase 2: Y reached target (%.2f), stopped Y movement", posComp.Y)
	}

	// 两个方向都到达目标，进入 Phase 3
	if xReached && yReached {
		if !phaseComp.ZombieReachedTarget {
			log.Printf("[ZombiesWonPhaseSystem] Phase 2: Zombie reached target position (%.2f, %.2f)",
				posComp.X, posComp.Y)

			// 冻结僵尸移动
			if velComp, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, phaseComp.TriggerZombieID); ok {
				velComp.VX = 0
				velComp.VY = 0
				log.Printf("[ZombiesWonPhaseSystem] Phase 2: Frozen zombie movement")
			}

			// 冻结僵尸动画（暂停 Reanim 动画）
			if reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, phaseComp.TriggerZombieID); ok {
				if reanimComp.AnimationPausedStates == nil {
					reanimComp.AnimationPausedStates = make(map[string]bool)
				}
				// 暂停所有当前播放的动画
				for _, animName := range reanimComp.CurrentAnimations {
					reanimComp.AnimationPausedStates[animName] = true
				}
				log.Printf("[ZombiesWonPhaseSystem] Phase 2: Frozen zombie animation (paused %d animations)", len(reanimComp.CurrentAnimations))
			}
		}
		phaseComp.ZombieReachedTarget = true

		// 僵尸到达目标位置，进入 Phase 3
		log.Printf("[ZombiesWonPhaseSystem] Phase 2 complete, Phase 2 -> Phase 3")
		phaseComp.CurrentPhase = 3
		phaseComp.PhaseTimer = 0.0
	}
}

// updatePhase3ScreamAnimation Phase 3: 惨叫与"吃脑子"动画
func (s *ZombiesWonPhaseSystem) updatePhase3ScreamAnimation(
	entityID ecs.EntityID,
	phaseComp *components.ZombiesWonPhaseComponent,
	deltaTime float64,
) {
	// 播放惨叫音效（只播放一次）
	if !phaseComp.ScreamPlayed {
		s.playScreamSound()
		phaseComp.ScreamPlayed = true
		log.Printf("[ZombiesWonPhaseSystem] Phase 3: Scream sound played")
	}

	// 延迟 0.5 秒后播放咀嚼音效
	if !phaseComp.ChompPlayed && phaseComp.PhaseTimer >= Phase3ChompDelay {
		s.playChompSound()
		phaseComp.ChompPlayed = true
		log.Printf("[ZombiesWonPhaseSystem] Phase 3: Chomp sound played")
	}

	// 创建 ZombiesWon.reanim 动画实体（只创建一次）
	if !phaseComp.AnimationReady {
		s.createZombiesWonAnimation()
		phaseComp.AnimationReady = true
		log.Printf("[ZombiesWonPhaseSystem] Phase 3: ZombiesWon animation created")
	}

	// "吃脑子"动画播放期间要有抖动效果
	// 抖动从动画创建后立即开始，持续 ScreenShakeDuration 秒
	// 只抖动动画实体本身，不影响背景
	if phaseComp.AnimationReady && s.zombiesWonEntityID != 0 {
		if phaseComp.ScreenShakeTime < ScreenShakeDuration {
			s.applyAnimationShake(phaseComp.ScreenShakeTime)
			phaseComp.ScreenShakeTime += deltaTime
		} else {
			// 抖动结束，恢复动画位置
			s.resetAnimationPosition()
		}
	}

	// 延迟 3-4 秒后进入 Phase 4
	if phaseComp.PhaseTimer >= Phase3AnimationDuration {
		log.Printf("[ZombiesWonPhaseSystem] Phase 3 -> Phase 4 (dialog)")
		phaseComp.CurrentPhase = 4
		phaseComp.PhaseTimer = 0.0

		// Story 8.8: 摄像机应该停留在目标位置（世界坐标 0），不要恢复到原始位置
		// 原版 PVZ 在失败流程中摄像机会停留在显示房子的位置
		s.gameState.CameraX = Phase2CameraTargetX // 停留在 0
	}
}

// updatePhase4Dialog Phase 4: 游戏结束对话框
func (s *ZombiesWonPhaseSystem) updatePhase4Dialog(
	entityID ecs.EntityID,
	phaseComp *components.ZombiesWonPhaseComponent,
	deltaTime float64,
) {
	// 检测鼠标点击或超时
	clicked := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	timeout := phaseComp.WaitTimer >= Phase4WaitTimeout

	phaseComp.WaitTimer += deltaTime

	if (clicked || timeout) && !phaseComp.DialogShown {
		log.Printf("[ZombiesWonPhaseSystem] Phase 4: Showing game over dialog (clicked=%v, timeout=%v)", clicked, timeout)

		// 直接创建游戏结束对话框（只有"再次尝试"按钮）
		_, err := entities.NewGameOverDialogEntity(
			s.entityManager,
			s.resourceManager,
			s.windowWidth,
			s.windowHeight,
			s.onRetryCallback,    // "再次尝试"回调
			nil,                   // 不需要"返回主菜单"按钮
		)
		if err != nil {
			log.Printf("[ZombiesWonPhaseSystem] Error creating game over dialog: %v", err)
		} else {
			log.Printf("[ZombiesWonPhaseSystem] Game over dialog created successfully")
		}

		phaseComp.DialogShown = true

		// 删除流程控制实体（流程结束）
		s.entityManager.DestroyEntity(entityID)
		log.Printf("[ZombiesWonPhaseSystem] Flow completed, entity destroyed")
	}
}

// playScreamSound 播放惨叫音效
func (s *ZombiesWonPhaseSystem) playScreamSound() {
	player := s.resourceManager.GetAudioPlayer("SOUND_SCREAM")
	if player != nil {
		player.Rewind()
		player.Play()
	} else {
		log.Printf("[ZombiesWonPhaseSystem] Warning: SOUND_SCREAM not loaded")
	}
}

// playChompSound 播放咀嚼音效
func (s *ZombiesWonPhaseSystem) playChompSound() {
	player := s.resourceManager.GetAudioPlayer("SOUND_CHOMP")
	if player != nil {
		player.Rewind()
		player.Play()
	} else {
		log.Printf("[ZombiesWonPhaseSystem] Warning: SOUND_CHOMP not loaded")
	}
}

// createZombiesWonAnimation 创建 ZombiesWon 动画实体
func (s *ZombiesWonPhaseSystem) createZombiesWonAnimation() {
	// 计算屏幕中央位置（使用配置常量）
	screenWidth := config.ScreenWidth
	screenHeight := config.ScreenHeight
	centerX := screenWidth / 2
	centerY := screenHeight / 2

	// 使用实体工厂创建 ZombiesWon 实体
	zombiesWonEntity, err := entities.NewZombiesWonEntity(
		s.entityManager,
		s.resourceManager,
		centerX,
		centerY,
	)
	if err != nil {
		log.Printf("[ZombiesWonPhaseSystem] Error creating ZombiesWon entity: %v", err)
		return
	}

	// 保存实体 ID 用于抖动效果
	s.zombiesWonEntityID = zombiesWonEntity

	// Story 8.8: ZombiesWon.reanim 是单动画文件（无命名动画）
	// 不需要通过 AnimationCommandComponent 播放动画
	// NewZombiesWonEntity 会自动设置 CurrentAnimations = ["_root"]
	// 注意：绝对不要播放 "anim_screen"（那是控制轨道），会导致：
	//   - anim_screen 只有 frame 24 可见
	//   - 而 ZombiesWon 轨道在 frame 24 是隐藏的（f=-1）
	//   - 结果：文字图片永远不显示

	log.Printf("[ZombiesWonPhaseSystem] ZombiesWon animation entity created (ID: %d) at (%.2f, %.2f)", zombiesWonEntity, centerX, centerY)
}

// applyAnimationShake 应用动画实体抖动效果（只影响文字轨道）
func (s *ZombiesWonPhaseSystem) applyAnimationShake(shakeTime float64) {
	if s.zombiesWonEntityID == 0 {
		return
	}

	// 获取动画实体的 Reanim 组件
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, s.zombiesWonEntityID)
	if !ok {
		return
	}

	// 使用正弦波计算抖动偏移
	offset := ScreenShakeAmplitude * math.Sin(2*math.Pi*ScreenShakeFrequency*shakeTime)

	// 只抖动文字轨道 "ZombiesWon"，背景轨道 "fullscreen"、"fullscreen2" 保持静止
	if reanimComp.TrackOffsets == nil {
		reanimComp.TrackOffsets = make(map[string][2]float64)
	}
	reanimComp.TrackOffsets["ZombiesWon"] = [2]float64{offset, 0} // 水平抖动

	// 标记缓存失效，下一帧会重新构建缓存并应用偏移
	reanimComp.LastRenderFrame = -1
}

// resetAnimationPosition 重置动画轨道偏移
func (s *ZombiesWonPhaseSystem) resetAnimationPosition() {
	if s.zombiesWonEntityID == 0 {
		return
	}

	// 获取动画实体的 Reanim 组件
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, s.zombiesWonEntityID)
	if !ok {
		return
	}

	// 清除轨道偏移
	if reanimComp.TrackOffsets != nil {
		delete(reanimComp.TrackOffsets, "ZombiesWon")
	}

	// 标记缓存失效
	reanimComp.LastRenderFrame = -1
}
