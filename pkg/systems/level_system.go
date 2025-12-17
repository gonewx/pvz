package systems

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
)

// 关卡流程常量
const (
	// LastWaveWarningTime 最后一波提示提前时间（秒）
	LastWaveWarningTime = 5.0
)

// LevelSystem 关卡管理系统
//
// 职责：
//   - 管理关卡时间推进
//   - 检测应该触发的僵尸波次
//   - 调用 WaveSpawnSystem 生成僵尸
//   - 检测胜利/失败条件
//   - 触发最后一波提示
//   - 触发关卡完成奖励动画（Story 8.3）
//   - Story 17.7: 管理旗帜波警告和最终波白字显示
//   - Story 17.8: 血量触发加速刷新
//   - Story 17.9: 类型化进家判定
//
// 架构说明：
//   - 通过 GameState 单例管理关卡状态
//   - 依赖 WaveSpawnSystem 进行僵尸生成（通过构造函数注入）
//   - 依赖 WaveTimingSystem 管理波次计时（Story 17.6）
//   - 遵循零耦合原则：不直接修改其他系统的状态
type LevelSystem struct {
	entityManager        *ecs.EntityManager
	gameState            *game.GameState
	waveSpawnSystem      *WaveSpawnSystem
	waveTimingSystem     *WaveTimingSystem      // Story 17.6: 波次计时系统
	resourceManager      *game.ResourceManager  // 用于加载 FinalWave 音效
	rewardSystem         *RewardAnimationSystem // 用于触发奖励动画（Story 8.3）
	lawnmowerSystem      *LawnmowerSystem       // 用于检查除草车状态（Story 10.2）
	lastWaveWarningShown bool                   // 已废弃：使用 finalWaveWarningTriggered 代替

	// Story 17.6: 是否使用新的波次计时系统
	useWaveTimingSystem bool

	// Story 17.8: 僵尸属性配置（用于血量计算）
	zombieStatsConfig *config.ZombieStatsConfig

	// Story 17.9: 僵尸物理配置（用于类型化进家判定）
	zombiePhysics *config.ZombiePhysicsConfig

	// 关卡进度条支持
	progressBarEntityID ecs.EntityID // 进度条实体ID（如果存在）

	// 最后一波提示相关
	finalWaveWarningTriggered bool    // 是否已触发提示（防止重复）
	finalWaveWarningLeadTime  float64 // 提前触发时间（秒，默认 3.0）

	// Story 17.7: 旗帜波警告系统
	flagWaveWarningSystem *FlagWaveWarningSystem // 红字警告系统
}

// NewLevelSystem 创建关卡管理系统
//
// 参数：
//
//	em - 实体管理器
//	gs - 游戏状态单例
//	waveSpawnSystem - 波次生成系统（依赖注入）
//	rm - 资源管理器（用于加载音效）
//	rewardSystem - 奖励动画系统（可选，Story 8.3）
//	lawnmowerSystem - 除草车系统（可选，Story 10.2）
//
// Removed ReanimSystem dependency, using AnimationCommand component
func NewLevelSystem(em *ecs.EntityManager, gs *game.GameState, waveSpawnSystem *WaveSpawnSystem, rm *game.ResourceManager, rewardSystem *RewardAnimationSystem, lawnmowerSystem *LawnmowerSystem) *LevelSystem {
	isTutorialLevel := gs.CurrentLevel != nil && gs.CurrentLevel.OpeningType == "tutorial"
	// Story 19.9: 特殊开场关卡（如保龄球 Level 1-5）也需要暂停波次，等待阶段转场完成
	isSpecialLevel := gs.CurrentLevel != nil && gs.CurrentLevel.OpeningType == "special"

	ls := &LevelSystem{
		entityManager:             em,
		gameState:                 gs,
		waveSpawnSystem:           waveSpawnSystem,
		resourceManager:           rm,
		rewardSystem:              rewardSystem,
		lawnmowerSystem:           lawnmowerSystem,
		lastWaveWarningShown:      false, // 已废弃，保留向后兼容
		finalWaveWarningTriggered: false, // 已废弃，统一由 FlagWaveWarningSystem 处理
		finalWaveWarningLeadTime:  3.0,   // 已废弃
		useWaveTimingSystem:       true,  // 统一使用 WaveTimingSystem
	}

	// Story 17.6+统一: 所有关卡都创建 WaveTimingSystem
	// 教学关卡：初始暂停，等待 TutorialSystem 触发第一波后恢复
	// 特殊开场关卡（如保龄球）：初始暂停，等待阶段转场完成后恢复
	// 普通关卡：自动开始计时
	if gs.CurrentLevel != nil {
		ls.waveTimingSystem = NewWaveTimingSystem(em, gs, gs.CurrentLevel)

		// Story 17.7: 创建旗帜波警告系统
		ls.flagWaveWarningSystem = NewFlagWaveWarningSystem(em, ls.waveTimingSystem, rm)

		if isTutorialLevel {
			// 教学关卡：暂停计时器，等待 TutorialSystem 触发第一波
			ls.waveTimingSystem.Pause()
			log.Printf("[LevelSystem] Tutorial level: WaveTimingSystem created in paused state")
		} else if isSpecialLevel {
			// Story 19.9: 特殊开场关卡（如保龄球）：暂停计时器，等待 LevelPhaseSystem 触发
			ls.waveTimingSystem.Pause()
			log.Printf("[LevelSystem] Special level: WaveTimingSystem created in paused state, waiting for phase transition")
		} else {
			// 普通关卡：自动初始化计时器
			ls.waveTimingSystem.InitializeTimerWithDelay(true, gs.CurrentLevel)
		}
	}

	return ls
}

// Update 更新关卡系统
//
// 执行流程：
//  1. 检查游戏是否结束，如果是则不处理
//  2. 更新关卡时间
//  3. 更新波次计时器（Story 17.6）
//  4. Story 17.7: 更新旗帜波警告和最终波白字系统
//  5. Story 17.7: 检查加速刷新条件
//  6. 检查并生成到期的僵尸波次
//  7. 检查最后一波提示
//  8. 检查失败条件（僵尸到达左边界）
//  9. 检查胜利条件（所有僵尸消灭）
//
// 参数：
//
//	deltaTime - 自上一帧以来经过的时间（秒）
func (s *LevelSystem) Update(deltaTime float64) {
	// 如果游戏已结束，不再处理逻辑
	if s.gameState.IsGameOver {
		return
	}

	// 如果未加载关卡，不处理
	if s.gameState.CurrentLevel == nil {
		return
	}

	// 更新关卡时间
	s.gameState.UpdateLevelTime(deltaTime)

	// 每秒输出一次调试日志
	levelTimeInt := int(s.gameState.LevelTime)
	if levelTimeInt > 0 && int(s.gameState.LevelTime*10)%10 == 0 {
		log.Printf("[LevelSystem] Update: LevelTime=%.2f, progressBarEntityID=%d", s.gameState.LevelTime, s.progressBarEntityID)
	}

	// Story 17.6: 更新波次计时器（如果启用）
	if s.useWaveTimingSystem && s.waveTimingSystem != nil {
		s.waveTimingSystem.Update(deltaTime)

		// Story 17.7: 更新旗帜波警告系统
		if s.flagWaveWarningSystem != nil {
			s.flagWaveWarningSystem.Update(deltaTime)
		}

		// Story 17.7: 检查加速刷新条件
		s.checkAcceleratedRefresh()
	}

	// 检查并生成僵尸波次
	s.checkAndSpawnWaves()

	// 最后一波警告现在由 FlagWaveWarningSystem 统一处理（所有关卡类型）

	// 检查失败条件（必须在胜利条件之前，优先级更高）
	s.checkDefeatCondition()

	// 检查胜利条件
	s.checkVictoryCondition()

	// 检测第一波是否已激活（教学关卡）
	// 教学关卡的第一波由 TutorialSystem 激活，需要手动检测并显示进度条
	if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.OpeningType == "tutorial" {
		if s.gameState.IsWaveSpawned(0) && s.progressBarEntityID != 0 {
			progressBar, ok := ecs.GetComponent[*components.LevelProgressBarComponent](s.entityManager, s.progressBarEntityID)
			if ok && progressBar.ShowLevelTextOnly {
				// 第一波已激活，但进度条还在文本模式，切换到完整模式
				s.ShowProgressBar()
			}
		}
	}

	// 更新进度条
	s.UpdateProgressBar()
}

// checkAcceleratedRefresh 检查加速刷新条件
//
// Story 17.7: 旗帜波前一波的加速刷新逻辑（消灭触发）
// Story 17.8: 常规波次的血量触发加速刷新
//
// 当满足以下条件时触发加速刷新：
//   - 消灭触发（旗帜波前）: 僵尸全部消灭
//   - 血量触发（常规波）: 当前血量 <= 初始血量 × 阈值
func (s *LevelSystem) checkAcceleratedRefresh() {
	if s.waveTimingSystem == nil {
		return
	}

	// Story 17.7: 旗帜波前消灭触发
	if s.waveTimingSystem.IsFlagWaveApproaching() {
		// 检查本波僵尸是否全部消灭
		allZombiesCleared := s.areCurrentWaveZombiesCleared()
		// 调用 WaveTimingSystem 执行消灭触发加速刷新检查
		s.waveTimingSystem.CheckAcceleratedRefresh(allZombiesCleared)
		return
	}

	// Story 17.8: 常规波次血量触发
	// 获取当前波次索引（用于计算该波僵尸血量）
	currentWaveIndex := s.waveTimingSystem.GetCurrentWaveIndex()
	if currentWaveIndex > 0 {
		// 计算上一波（刚激活的波次）的当前血量
		// 注意：currentWaveIndex 是下一个要触发的波次，所以刚激活的是 currentWaveIndex-1
		lastActivatedWaveIndex := currentWaveIndex - 1
		currentHealth := CalculateCurrentWaveHealth(s.entityManager, lastActivatedWaveIndex)
		s.waveTimingSystem.CheckHealthAcceleratedRefresh(currentHealth)
	}
}

// areCurrentWaveZombiesCleared 检查当前波僵尸是否全部消灭
//
// Story 17.7: 用于加速刷新判定
// 注意：不计算伴舞僵尸
//
// 返回：
//   - bool: true 表示本波僵尸已全部消灭
func (s *LevelSystem) areCurrentWaveZombiesCleared() bool {
	// 查询所有活跃僵尸
	zombieEntities := ecs.GetEntitiesWith2[
		*components.BehaviorComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, entityID := range zombieEntities {
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 只检查僵尸类型
		if !isZombieType(behavior.Type) {
			continue
		}

		// 跳过伴舞僵尸（BackupDancer）
		// 注意：当前代码库可能没有 BehaviorZombieBackupDancer 类型
		// 如果有，需要在这里排除

		// 有活跃僵尸，返回 false
		// Story 17.8: 只统计已激活的僵尸
		if waveState, ok := ecs.GetComponent[*components.ZombieWaveStateComponent](s.entityManager, entityID); ok {
			if !waveState.IsActivated {
				continue
			}
		}

		return false
	}

	// 没有活跃僵尸，返回 true
	return true
}

// checkAndSpawnWaves 检查并激活到期的僵尸波次
//
// 遍历所有波次，找到时间已到且未激活的波次，调用 WaveSpawnSystem.ActivateWave() 激活僵尸
//
// Story 17.6: 如果启用了 WaveTimingSystem，从计时系统获取触发信号
// Story 17.8: 在激活波次后初始化血量追踪
// Story 17.6+统一: 所有关卡（包括教学关卡）的波次都由 WaveTimingSystem 统一管理
// Story 11.5: 激活波次后更新进度条状态
func (s *LevelSystem) checkAndSpawnWaves() {
	var waveIndex int
	var nextWaveDelay float64 = 0

	// Story 17.6: 优先使用 WaveTimingSystem 的触发信号
	if s.useWaveTimingSystem && s.waveTimingSystem != nil {
		triggered, idx := s.waveTimingSystem.IsWaveTriggered()
		if !triggered {
			// 没有到期的波次
			return
		}
		waveIndex = idx
		// 清除触发标志
		s.waveTimingSystem.ClearWaveTriggered()

		// Story 11.5: 获取下一波的初始倒计时（用于进度条时间进度计算）
		nextWaveDelay = s.waveTimingSystem.GetNextWaveDelay()
	} else {
		// 原有逻辑：从 GameState 获取
		waveIndex = s.gameState.GetCurrentWave()
		if waveIndex == -1 {
			// 没有到期的波次
			return
		}
	}

	// 调用 WaveSpawnSystem 实时生成并激活僵尸
	zombieCount := s.waveSpawnSystem.SpawnWaveRealtime(waveIndex)

	// 标记波次已激活
	s.gameState.MarkWaveSpawned(waveIndex)

	// Story 17.8: 初始化波次血量追踪
	s.initializeWaveHealth(waveIndex)

	// Story 11.5: 更新进度条波次状态
	s.OnWaveActivated(waveIndex, nextWaveDelay)

	// 第一波僵尸生成后显示完整进度条
	if waveIndex == 0 {
		s.ShowProgressBar()
	}

	log.Printf("[LevelSystem] Wave %d activated: %d zombies", waveIndex+1, zombieCount)
}

// ========================================
// Story 17.6: 波次计时系统集成
// ========================================

// EnableWaveTimingSystem 启用波次计时系统
//
// Story 17.6: 启用新的波次计时系统
// 调用此方法后，波次触发将由 WaveTimingSystem 控制
//
// 参数：
//   - isFirstPlaythrough: 是否为首次游戏（一周目首次）
func (s *LevelSystem) EnableWaveTimingSystem(isFirstPlaythrough bool) {
	if s.waveTimingSystem == nil {
		log.Printf("[LevelSystem] WARNING: WaveTimingSystem not initialized, cannot enable")
		return
	}

	s.useWaveTimingSystem = true
	s.waveTimingSystem.InitializeTimer(isFirstPlaythrough)
	log.Printf("[LevelSystem] WaveTimingSystem enabled (first playthrough: %v)", isFirstPlaythrough)
}

// DisableWaveTimingSystem 禁用波次计时系统
//
// Story 17.6: 禁用新的波次计时系统，恢复使用原有逻辑
func (s *LevelSystem) DisableWaveTimingSystem() {
	s.useWaveTimingSystem = false
	log.Printf("[LevelSystem] WaveTimingSystem disabled, using legacy timing")
}

// PauseWaveTiming 暂停波次计时
//
// Story 17.6: 在游戏暂停时调用
func (s *LevelSystem) PauseWaveTiming() {
	if s.waveTimingSystem != nil {
		s.waveTimingSystem.Pause()
	}
}

// ResumeWaveTiming 恢复波次计时
//
// Story 17.6: 在游戏恢复时调用
// Story 19.9: 特殊关卡（如保龄球）在阶段转场后调用，会自动初始化计时器
// Bug Fix: 在铲子教学阶段（Phase 1）暂停恢复时，不应该初始化计时器
//
//	只有在保龄球阶段（Phase 2）或转场完成后才应该初始化
func (s *LevelSystem) ResumeWaveTiming() {
	if s.waveTimingSystem == nil {
		return
	}

	// Bug Fix: 检查是否处于铲子教学阶段（Phase 1）
	// 如果存在 LevelPhaseComponent 且 CurrentPhase == 1，说明还在铲子教学阶段
	// 此时不应该自动初始化波次计时器，只恢复暂停状态（如果已初始化）
	phaseEntities := ecs.GetEntitiesWith1[*components.LevelPhaseComponent](s.entityManager)
	if len(phaseEntities) > 0 {
		phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](s.entityManager, phaseEntities[0])
		if ok && phaseComp.CurrentPhase == 1 {
			// 铲子教学阶段：只恢复暂停状态，不初始化计时器
			// 计时器会在阶段转场完成后由 onActivateBowling 回调初始化
			log.Printf("[LevelSystem] In shovel tutorial phase (Phase 1), skipping wave timer initialization on resume")
			// 只有当计时器已经初始化（CountdownCs > 0）时才恢复
			timer, timerOk := ecs.GetComponent[*components.WaveTimerComponent](s.entityManager, s.waveTimingSystem.GetTimerEntityID())
			if timerOk && timer.CountdownCs > 0 {
				s.waveTimingSystem.Resume()
			}
			return
		}
	}

	// Story 19.9: 检查计时器是否需要初始化
	// 特殊关卡在初始化时暂停了计时器但没有设置首波延迟
	timer, ok := ecs.GetComponent[*components.WaveTimerComponent](s.entityManager, s.waveTimingSystem.GetTimerEntityID())
	if ok && timer.CountdownCs == 0 && timer.CurrentWaveIndex == 0 && timer.IsFirstWave {
		// 计时器未初始化，需要先设置首波延迟
		log.Printf("[LevelSystem] Initializing wave timer before resume (special level)")
		s.waveTimingSystem.InitializeTimerWithDelay(true, s.gameState.CurrentLevel)
	}

	s.waveTimingSystem.Resume()
}

// GetWaveTimingSystem 获取波次计时系统（用于测试）
func (s *LevelSystem) GetWaveTimingSystem() *WaveTimingSystem {
	return s.waveTimingSystem
}

// IsWaveTimingSystemEnabled 检查波次计时系统是否启用
func (s *LevelSystem) IsWaveTimingSystemEnabled() bool {
	return s.useWaveTimingSystem
}

// InitializeWaveTimingSystem 初始化波次计时系统
//
// Story 17.6: 在关卡加载后调用，创建 WaveTimingSystem
// Story 17.7: 同时创建旗帜波警告和最终波白字系统
// 用于在 NewLevelSystem 之后关卡配置才加载的情况
func (s *LevelSystem) InitializeWaveTimingSystem() {
	if s.gameState.CurrentLevel == nil {
		log.Printf("[LevelSystem] WARNING: No level loaded, cannot initialize WaveTimingSystem")
		return
	}

	s.waveTimingSystem = NewWaveTimingSystem(s.entityManager, s.gameState, s.gameState.CurrentLevel)

	// Story 17.7: 创建旗帜波警告系统
	s.flagWaveWarningSystem = NewFlagWaveWarningSystem(s.entityManager, s.waveTimingSystem, s.resourceManager)

	log.Printf("[LevelSystem] WaveTimingSystem initialized with FlagWaveWarning system")
}

// GetFlagWaveWarningSystem 获取旗帜波警告系统（用于测试）
//
// Story 17.7: 供测试和调试使用
func (s *LevelSystem) GetFlagWaveWarningSystem() *FlagWaveWarningSystem {
	return s.flagWaveWarningSystem
}
