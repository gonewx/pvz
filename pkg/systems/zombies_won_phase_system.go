package systems

import (
	"log"
	"math"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/types"
)

// 僵尸获胜流程常量
const (
	// Phase 1: 游戏冻结阶段持续时间（秒）
	// Story 8.8: 游戏冻结1.5秒，期间所有物体静止
	Phase1FreezeDuration = 1.0

	// Phase 2: 摄像机移动目标位置（世界坐标）
	Phase2CameraTargetX = 0.0

	// Phase 2: 摄像机移动速度（像素/秒）
	// 假设从标准游戏位置（~400）移动到 0，约 2-3 秒完成
	Phase2CameraMoveSpeed = 200.0

	// Phase 2: 第1个目标位置偏移量（房门口，相对于 config.GameOverDoorMaskX/Y）
	// 实际目标位置 = GameOverDoorMaskX + OffsetX, GameOverDoorMaskY + OffsetY
	Phase2ZombieTarget1OffsetX = 50.0  // 让僵尸站在门口偏右
	Phase2ZombieTarget1OffsetY = 150.0 // 门口 Y 位置（手工调整）

	// Phase 2: 第2个目标位置 X 偏移量（即将进入房子，相对于第1个目标位置的 X）
	// 僵尸到达第1个目标后继续向左走，到达此位置时触发 Phase 3
	Phase2ZombieTarget2OffsetX = -120.0 // 从第1个目标位置向左偏移

	// Phase 2: 僵尸行走速度（满足 Story 8.8 视觉要求）
	Phase2ZombieHorizontalSpeed = 50.0
	Phase2ZombieVerticalSpeed   = 50.0

	// Phase 2: 判定到达目标的容差（±像素）
	Phase2ZombieReachThreshold = 5.0

	// Phase 3: 咀嚼音效延迟时间（相对于惨叫音效）
	Phase3ChompDelay = 0.5

	// Phase 3: 屏幕抖动参数
	ScreenShakeAmplitude = 3.0   // 振幅（像素）
	ScreenShakeFrequency = 500.0 // 频率（Hz）
	ScreenShakeDuration  = 2.5   // 抖动持续时间（秒）

	// 僵尸触发失败的边界X坐标（进入 Phase 1）
	// Story 17.9: 此常量作为兼容默认值，LevelSystem 会优先使用类型化的配置边界
	DefeatBoundaryX = 220.0
)

// StartZombiesWonFlow 启动僵尸获胜流程
// 创建流程控制实体并添加必要的组件（状态机、冻结标记）
// 返回流程实体 ID
func StartZombiesWonFlow(em *ecs.EntityManager, triggerZombieID ecs.EntityID) ecs.EntityID {
	// 创建流程控制实体
	flowEntityID := em.CreateEntity()

	// 添加阶段控制组件
	phaseComp := &components.ZombiesWonPhaseComponent{
		CurrentPhase:         1, // 默认为 Phase 1
		PhaseTimer:           0.0,
		TriggerZombieID:      triggerZombieID,
		CameraMovedToTarget:  false,
		InitialCameraX:       0.0, // 将在 System Update 中初始化
		ZombieStartedWalking: false,
		ZombieReachedTarget:  false,
		ScreamPlayed:         false,
		ChompPlayed:          false,
		AnimationReady:       false,
		ScreenShakeTime:      0.0,
		DialogShown:          false,
		WaitTimer:            0.0,
	}
	ecs.AddComponent(em, flowEntityID, phaseComp)

	// 添加游戏冻结标记
	freezeComp := &components.GameFreezeComponent{
		IsFrozen: true,
	}
	ecs.AddComponent(em, flowEntityID, freezeComp)

	log.Printf("[ZombiesWonPhaseSystem] Started flow (EntityID: %d, TriggerZombieID: %d)", flowEntityID, triggerZombieID)
	return flowEntityID
}

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
	onRetryCallback    func() // "再次尝试"回调
	onMainMenuCallback func() // "返回主菜单"回调（可选，验证程序不需要）

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
		// AC 1: 背景音乐在 ~0.2 秒内淡出
		if s.resourceManager != nil {
			s.resourceManager.FadeOutMusic(0.2)
			log.Printf("[ZombiesWonPhaseSystem] Phase 1: Music fade out started (0.2s)")
		}
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

// updatePhase2ZombieEntry Phase 2: 僵尸入侵动画（同时执行）
// 摄像机和僵尸同时移动（并行执行）
// 进入 Phase 3 条件：摄像机到位（CameraX ≤ 0）且 僵尸到达房门位置
func (s *ZombiesWonPhaseSystem) updatePhase2ZombieEntry(
	entityID ecs.EntityID,
	phaseComp *components.ZombiesWonPhaseComponent,
	deltaTime float64,
) {

	// 播放失败音乐（只播放一次，使用 AudioManager 统一管理 - Story 10.9）
	if !phaseComp.LoseMusicPlayed {
		if audioManager := s.gameState.GetAudioManager(); audioManager != nil {
			audioManager.PlaySound("SOUND_LOSEMUSIC")
			phaseComp.LoseMusicPlayed = true
			log.Printf("[ZombiesWonPhaseSystem] Phase 2: Lose music played")
		}
	}

	// 第一次进入 Phase 2：初始化状态
	if phaseComp.PhaseTimer == deltaTime { // 刚进入 Phase 2
		phaseComp.InitialCameraX = s.gameState.CameraX
		phaseComp.ZombieStartedWalking = true // 立即开始行走（与摄像机同时）
		log.Printf("[ZombiesWonPhaseSystem] Phase 2 started: Initial camera X=%.2f, zombie starts walking simultaneously",
			phaseComp.InitialCameraX)

		// Story 8.8: 懒加载房门图片（DelayLoad_Background1 资源组）
		// 确保门板图片在渲染前已加载到缓存
		// 注意：单元测试环境中 resourceManager 可能未初始化配置，需要容错处理
		if s.resourceManager != nil {
			if err := s.resourceManager.LoadResourceGroup("DelayLoad_Background1"); err != nil {
				log.Printf("[ZombiesWonPhaseSystem] 警告：房门图片加载失败: %v", err)
			} else {
				log.Printf("[ZombiesWonPhaseSystem] Phase 2: 成功加载房门图片资源")
			}
		}

	}

	// ========================================
	// Step 1: 摄像机平滑移动到目标位置（与僵尸同时）
	// ========================================
	if !phaseComp.CameraMovedToTarget {
		currentCameraX := s.gameState.CameraX

		if currentCameraX > Phase2CameraTargetX {
			moveDistance := Phase2CameraMoveSpeed * deltaTime
			newCameraX := currentCameraX - moveDistance

			// 防止超调
			if newCameraX <= Phase2CameraTargetX {
				newCameraX = Phase2CameraTargetX
				phaseComp.CameraMovedToTarget = true
				log.Printf("[ZombiesWonPhaseSystem] Phase 2: Camera reached target X=%.2f", newCameraX)
			}

			s.gameState.CameraX = newCameraX
		} else {
			phaseComp.CameraMovedToTarget = true
			log.Printf("[ZombiesWonPhaseSystem] Phase 2: Camera already at target")
		}
	}

	// ========================================
	// Step 2: 僵尸行走到房门目标位置（与摄像机同时）
	// ========================================
	posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, phaseComp.TriggerZombieID)
	if !ok {
		log.Printf("[ZombiesWonPhaseSystem] Warning: Trigger zombie entity %d not found", phaseComp.TriggerZombieID)
		phaseComp.CurrentPhase = 3
		phaseComp.PhaseTimer = 0.0
		phaseComp.ZombieReachedTarget = true
		return
	}

	// 获取第1个目标位置（房门口，基于 gameover_door_config.go + 偏移常量）
	target1X := config.GameOverDoorMaskX + Phase2ZombieTarget1OffsetX
	target1Y := config.GameOverDoorMaskY + Phase2ZombieTarget1OffsetY

	// 获取第2个目标位置（即将进入房子，相对于第1个目标位置的 X 偏移）
	target2X := target1X + Phase2ZombieTarget2OffsetX

	// 计算当前位置与第1个目标的距离
	currentX := posComp.X
	currentY := posComp.Y
	dx := target1X - currentX
	dy := target1Y - currentY
	distance := math.Sqrt(dx*dx + dy*dy)

	// DEBUG: 每秒输出一次僵尸位置和目标位置（避免日志过多）
	if int(phaseComp.PhaseTimer*10)%10 == 0 {
		log.Printf("[ZombiesWonPhaseSystem] DEBUG Phase 2: Zombie pos=(%.2f, %.2f), target1=(%.2f, %.2f), target2X=%.2f, distance=%.2f, reached1=%v, reached2=%v, cameraReached=%v",
			currentX, currentY, target1X, target1Y, target2X, distance, phaseComp.ZombieReachedTarget1, phaseComp.ZombieReachedTarget, phaseComp.CameraMovedToTarget)
	}

	// 僵尸移动逻辑
	if phaseComp.ZombieStartedWalking && !phaseComp.ZombieReachedTarget {
		if velComp, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, phaseComp.TriggerZombieID); ok {
			// 阶段1：僵尸还未到达第1个目标位置（房门口）
			if !phaseComp.ZombieReachedTarget1 {
				// 判断是否到达第1个目标位置（距离小于阈值）
				if distance <= Phase2ZombieReachThreshold {
					// 僵尸到达第1个目标位置
					// 注意：不强制 snap Y 坐标，因为僵尸可能从不同行走来
					// 只 snap X 坐标，保持当前 Y 位置（避免突然跳变）
					posComp.X = target1X
					// 切换为只向左移动（进入房门）
					velComp.VX = -Phase2ZombieHorizontalSpeed
					velComp.VY = 0
					phaseComp.ZombieReachedTarget1 = true
					// 僵尸到达房门口时播放一次啃食音效
					if !phaseComp.DoorChompPlayed {
						s.playChompSound()
						phaseComp.DoorChompPlayed = true
						log.Printf("[ZombiesWonPhaseSystem] Phase 2: Door chomp sound played")
					}
					log.Printf("[ZombiesWonPhaseSystem] Phase 2: Zombie reached target 1 (door position X=%.2f), keeping Y=%.2f, now walking into door", posComp.X, posComp.Y)
				} else {
					// 计算单位方向向量，沿直线走向第1个目标
					dirX := dx / distance
					dirY := dy / distance
					// 按固定速度移动
					velComp.VX = dirX * Phase2ZombieHorizontalSpeed
					velComp.VY = dirY * Phase2ZombieHorizontalSpeed // 使用相同速度保持匀速
				}
			} else {
				// 阶段2：僵尸已到达第1个目标，继续向左走到第2个目标位置
				// 判断是否到达第2个目标位置
				if currentX <= target2X {
					phaseComp.ZombieReachedTarget = true
					log.Printf("[ZombiesWonPhaseSystem] Phase 2: Zombie reached target 2 (entering house) X=%.2f (target2X=%.2f)", currentX, target2X)
				}
				// 继续向左移动
				velComp.VX = -Phase2ZombieHorizontalSpeed
				velComp.VY = 0
			}
		}
	}

	// ========================================
	// 进入 Phase 3 条件：摄像机到位 且 僵尸到达第2个目标位置
	// ========================================
	if phaseComp.CameraMovedToTarget && phaseComp.ZombieReachedTarget {
		log.Printf("[ZombiesWonPhaseSystem] Phase 2 complete -> Phase 3 (camera at target AND zombie at target 2)")
		phaseComp.CurrentPhase = 3
		phaseComp.PhaseTimer = 0.0

		// Story 8.8 AC: 僵尸到达第2个目标位置后继续在 X 轴前进（VY=0），模拟走进房门效果
		// 不冻结僵尸移动，让它继续向左走进门
		if velComp, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, phaseComp.TriggerZombieID); ok {
			velComp.VX = -Phase2ZombieHorizontalSpeed // 继续向左
			velComp.VY = 0
		}
		// 不冻结动画，让僵尸继续播放行走动画
	}
}

// updatePhase3ScreamAnimation Phase 3: 惨叫与"吃脑子"动画
func (s *ZombiesWonPhaseSystem) updatePhase3ScreamAnimation(
	entityID ecs.EntityID,
	phaseComp *components.ZombiesWonPhaseComponent,
	deltaTime float64,
) {
	// Story 8.8 AC Phase 2: 僵尸到达房门口后继续在 X 轴前进，模拟走进房门效果
	// Phase 3 开始时僵尸继续向左移动（已在 Phase 2 结束时设置 VX=-150）
	// 当僵尸完全走出画面（X < -100）后停止移动

	// 检查僵尸是否完全走出画面
	if posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, phaseComp.TriggerZombieID); ok {
		if posComp.X < -100 {
			// 僵尸已走出画面，冻结移动
			if velComp, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, phaseComp.TriggerZombieID); ok {
				velComp.VX = 0
				velComp.VY = 0
			}
		}
	}

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

	// 检查动画是否播放完成
	// 动画播放完成后停留在最后一帧（保持遮罩效果），然后进入 Phase 4
	if phaseComp.AnimationReady && s.zombiesWonEntityID != 0 {
		reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, s.zombiesWonEntityID)
		if ok && reanimComp.IsFinished {
			log.Printf("[ZombiesWonPhaseSystem] Phase 3 -> Phase 4 (animation finished, staying on last frame)")
			phaseComp.CurrentPhase = 4
			phaseComp.PhaseTimer = 0.0

			// Story 8.8: 摄像机应该停留在目标位置（世界坐标 0），不要恢复到原始位置
			// 原版 PVZ 在失败流程中摄像机会停留在显示房子的位置
			s.gameState.CameraX = Phase2CameraTargetX // 停留在 0
		}
	}
}

// updatePhase4Dialog Phase 4: 游戏结束对话框
func (s *ZombiesWonPhaseSystem) updatePhase4Dialog(
	entityID ecs.EntityID,
	phaseComp *components.ZombiesWonPhaseComponent,
	deltaTime float64,
) {
	// 确保僵尸已停止移动（此时僵尸应该已经走出画面 X < -100）
	if velComp, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, phaseComp.TriggerZombieID); ok {
		velComp.VX = 0
		velComp.VY = 0
	}

	// 动画播放完成后立即显示对话框
	if !phaseComp.DialogShown {
		log.Printf("[ZombiesWonPhaseSystem] Phase 4: Showing game over dialog immediately")

		// 直接创建游戏结束对话框（只有"再次尝试"按钮）
		_, err := entities.NewGameOverDialogEntity(
			s.entityManager,
			s.resourceManager,
			s.windowWidth,
			s.windowHeight,
			s.onRetryCallback, // "再次尝试"回调
			nil,               // 不需要"返回主菜单"按钮
		)
		if err != nil {
			log.Printf("[ZombiesWonPhaseSystem] Error creating game over dialog: %v", err)
		} else {
			log.Printf("[ZombiesWonPhaseSystem] Game over dialog created successfully")
		}

		phaseComp.DialogShown = true

		// Story 8.8: 保持流程实体存在，以维持游戏冻结状态（GameFreezeComponent）
		// 直到玩家点击按钮重置场景
		// s.entityManager.DestroyEntity(entityID)
		log.Printf("[ZombiesWonPhaseSystem] Phase 4: Dialog shown, keeping game frozen")
	}
}

// playScreamSound 播放惨叫音效
func (s *ZombiesWonPhaseSystem) playScreamSound() {
	// 使用 AudioManager 统一管理音效播放（Story 10.9）
	if audioManager := s.gameState.GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_SCREAM")
	} else {
		log.Printf("[ZombiesWonPhaseSystem] Warning: AudioManager not available for SOUND_SCREAM")
	}
}

// playChompSound 播放咀嚼音效
func (s *ZombiesWonPhaseSystem) playChompSound() {
	// 使用 AudioManager 统一管理音效播放（Story 10.9）
	if audioManager := s.gameState.GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_CHOMP")
	} else {
		log.Printf("[ZombiesWonPhaseSystem] Warning: AudioManager not available for SOUND_CHOMP")
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

	// ✅ 使用配置文件播放动画 (data/reanim_config/zombieswon.yaml)
	// combo "appear" 配置: loop=false, speed=1.0
	ecs.AddComponent(s.entityManager, zombiesWonEntity, &components.AnimationCommandComponent{
		UnitID:    types.UnitIDZombiesWon,
		ComboName: "appear",
		Processed: false,
	})

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
