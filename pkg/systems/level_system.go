package systems

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
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

// checkVictoryCondition 检查胜利条件
//
// 胜利条件：
// 1. 所有波次已生成 且 所有僵尸已消灭（GameState.CheckVictory()）
// 2. 没有活跃的（移动中的）除草车（Story 10.2）- 车完全消失后再显示胜利动画
//
// 如果达成胜利条件，设置游戏结果为 "win"
func (s *LevelSystem) checkVictoryCondition() {
	// 检查是否有活跃的除草车
	// 原版行为：除草车完全消失后，才显示胜利动画
	hasActiveLawnmowers := false
	if s.lawnmowerSystem != nil {
		hasActiveLawnmowers = s.lawnmowerSystem.HasActiveLawnmowers()
	}

	// Story 18.3 调试：计算场上僵尸数量
	zombiesOnField := s.gameState.TotalZombiesSpawned - s.gameState.ZombiesKilled
	if zombiesOnField <= 0 && s.gameState.TotalZombiesSpawned > 0 {
		// 场上没有僵尸了，输出详细状态
		log.Printf("[LevelSystem] checkVictoryCondition: ZombiesOnField=%d, Killed=%d/%d, SpawnedWaves=%v, hasActiveLawnmowers=%v",
			zombiesOnField, s.gameState.ZombiesKilled, s.gameState.TotalZombiesInLevel,
			s.gameState.SpawnedWaves, hasActiveLawnmowers)
	}

	// 只有在没有活跃除草车的情况下才能胜利
	if s.gameState.CheckVictory() && !hasActiveLawnmowers {
		s.gameState.SetGameResult("win")
		log.Println("[LevelSystem] Victory! All zombies defeated!")

		// 播放胜利音乐
		if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
			audioManager.PlayMusic("SOUND_WINMUSIC")
		}

		// 清理场上所有阳光实体（避免在奖励动画阶段继续显示）
		s.cleanupAllSunEntities()

		// 保存关卡进度
		if s.gameState.CurrentLevel != nil {
			levelID := s.gameState.CurrentLevel.ID
			rewardPlant := s.gameState.CurrentLevel.RewardPlant
			unlockTools := s.gameState.CurrentLevel.UnlockTools

			// 保存进度（包括关卡完成、植物解锁、工具解锁）
			if err := s.gameState.CompleteLevel(levelID, rewardPlant, unlockTools); err != nil {
				log.Printf("[LevelSystem] Warning: Failed to save progress: %v", err)
			} else {
				log.Printf("[LevelSystem] Progress saved: level=%s, plant=%s, tools=%v", levelID, rewardPlant, unlockTools)
			}
		}

		// 检查是否有新植物解锁，触发奖励动画
		s.triggerRewardIfNeeded()
	}
}

// triggerRewardIfNeeded 检查是否有新植物或工具解锁，如果有则触发奖励动画
func (s *LevelSystem) triggerRewardIfNeeded() {
	if s.rewardSystem == nil {
		return
	}

	hasReward := false

	// 优先检查工具奖励（工具比植物更稀有，优先显示）
	if s.gameState.CurrentLevel != nil && len(s.gameState.CurrentLevel.UnlockTools) > 0 {
		toolID := s.gameState.CurrentLevel.UnlockTools[0] // 通常只解锁一个工具
		log.Printf("[LevelSystem] Triggering reward animation for tool: %s", toolID)
		s.rewardSystem.TriggerToolReward(toolID)
		hasReward = true
	}

	// 如果没有工具奖励，检查植物奖励
	if !hasReward {
		lastUnlocked := s.gameState.GetPlantUnlockManager().GetLastUnlocked()
		if lastUnlocked != "" {
			log.Printf("[LevelSystem] Triggering reward animation for plant: %s", lastUnlocked)
			s.rewardSystem.TriggerPlantReward(lastUnlocked)
			hasReward = true
			// 清除最后解锁标记（避免重复触发）
			s.gameState.GetPlantUnlockManager().ClearLastUnlocked()
		}
	}

	if !hasReward {
		log.Println("[LevelSystem] No new reward unlocked, skipping reward animation")
	}
}

// checkDefeatCondition 检查失败条件
//
// 失败条件（Story 10.2 增强）：
// 1. 如果启用了除草车系统：僵尸到达左侧边界 && 该行除草车已使用 → 游戏失败
// 2. 如果未启用除草车系统：僵尸到达左侧边界 → 游戏失败（原逻辑）
//
// 这样设计的原因：
// - 除草车是每行的最后防线，只在僵尸到达左侧时触发一次
// - 如果除���车未使用，僵尸到达左侧会触发除草车（不触发失败）
// - 如果除草车已使用，僵尸再次到达左侧直接失败（无最后防线）
func (s *LevelSystem) checkDefeatCondition() {
	// 如果启用了除草车系统，检查除草车状态
	if s.lawnmowerSystem != nil {
		s.checkDefeatWithLawnmower()
		return
	}

	// 原逻辑：未启用除草车系统，僵尸到达左侧直接失败
	s.checkDefeatWithoutLawnmower()
}

// checkDefeatWithLawnmower 检查失败条件（有除草车）
// Story 17.9: 使用类型化进家边界
func (s *LevelSystem) checkDefeatWithLawnmower() {
	// 获取除草车状态组件
	stateEntityID := s.lawnmowerSystem.GetStateEntityID()
	state, ok := ecs.GetComponent[*components.LawnmowerStateComponent](s.entityManager, stateEntityID)
	if !ok {
		log.Printf("[LevelSystem] Warning: LawnmowerStateComponent not found, falling back to original defeat logic")
		s.checkDefeatWithoutLawnmower()
		return
	}

	// 查询所有僵尸实体
	zombieEntities := ecs.GetEntitiesWith2[
		*components.BehaviorComponent,
		*components.PositionComponent,
	](s.entityManager)

	// 检查是否有僵尸到达左边界
	for _, entityID := range zombieEntities {
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 只检查僵尸类型的实体
		if !isZombieType(behavior.Type) {
			continue
		}

		pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// Story 17.9: 获取类型化的进家边界
		zombieTypeStr := s.behaviorTypeToString(behavior.Type)
		defeatBoundary := s.getDefeatBoundary(zombieTypeStr)

		// 僵尸到达左边界
		if pos.X < defeatBoundary {
			// 计算僵尸所在行
			lane := s.getEntityLane(pos.Y)

			// 检查该行除草车是否已使用
			if state.UsedLanes[lane] {
				// 除草车已使用，游戏失败
				s.gameState.SetGameResult("lose")
				log.Printf("[LevelSystem] Defeat! Zombie (ID:%d, type:%s) reached the left boundary on lane %d (lawnmower used, boundary=%.1f)", entityID, zombieTypeStr, lane, defeatBoundary)

				// 触发完整的僵尸获胜流程（Story 8.8）
				s.triggerZombiesWonFlow(entityID)
				return
			} else {
				// 除草车未使用，不触发失败（让除草车触发）
				log.Printf("[LevelSystem] Zombie (ID:%d, type:%s) reached left boundary on lane %d, waiting for lawnmower to trigger", entityID, zombieTypeStr, lane)
				// 注意：不 return，继续检查其他行是否有除草车用完的情况
			}
		}
	}
}

// checkDefeatWithoutLawnmower 检查失败条件（无除草车，原逻辑）
// Story 17.9: 使用类型化进家边界
func (s *LevelSystem) checkDefeatWithoutLawnmower() {
	zombieEntities := ecs.GetEntitiesWith2[
		*components.BehaviorComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, entityID := range zombieEntities {
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		if !isZombieType(behavior.Type) {
			continue
		}

		pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// Story 17.9: 获取类型化的进家边界
		zombieTypeStr := s.behaviorTypeToString(behavior.Type)
		defeatBoundary := s.getDefeatBoundary(zombieTypeStr)

		// 僵尸到达左边界，游戏失败
		if pos.X < defeatBoundary {
			s.gameState.SetGameResult("lose")
			log.Printf("[LevelSystem] Defeat! Zombie (ID:%d, type:%s) reached the left boundary at X=%.0f (boundary=%.1f)", entityID, zombieTypeStr, pos.X, defeatBoundary)

			// 触发完整的僵尸获胜流程（Story 8.8）
			s.triggerZombiesWonFlow(entityID)
			return
		}
	}
}

// getDefeatBoundary 获取僵尸类型对应的进家边界（世界坐标）
//
// Story 17.9: 使用类型化进家边界配置
// 如果未加载物理配置，使用现有 DefeatBoundaryX 常量（向后兼容）
//
// 参数:
//   - zombieType: 僵尸类型字符串（如 "basic", "polevaulter"）
//
// 返回:
//   - float64: 进家判定边界（世界坐标）
func (s *LevelSystem) getDefeatBoundary(zombieType string) float64 {
	if s.zombiePhysics != nil {
		boundary := s.zombiePhysics.GetDefeatBoundary(zombieType)
		return config.GridToWorldX(boundary)
	}
	// 兼容现有常量
	return DefeatBoundaryX
}

// behaviorTypeToString 将 BehaviorType 转换为僵尸类型字符串
//
// Story 17.9: 用于查找进家边界配置
//
// 参数:
//   - behaviorType: 行为类型枚举
//
// 返回:
//   - string: 僵尸类型字符串
func (s *LevelSystem) behaviorTypeToString(behaviorType components.BehaviorType) string {
	switch behaviorType {
	case components.BehaviorZombieBasic, components.BehaviorZombieEating, components.BehaviorZombieDying:
		return "basic"
	case components.BehaviorZombieConehead:
		return "conehead"
	case components.BehaviorZombieBuckethead:
		return "buckethead"
	// 以下为未来扩展的僵尸类型，当前返回默认值
	default:
		return "default"
	}
}

// getEntityLane 根据实体的Y坐标计算所在行（1-5）
func (s *LevelSystem) getEntityLane(y float64) int {
	// 使用与 LawnmowerSystem 相同的计算方法
	offsetY := y - config.GridWorldStartY
	row := int(offsetY / config.CellHeight)
	lane := row + 1

	// 限制范围
	if lane < 1 {
		lane = 1
	}
	if lane > 5 {
		lane = 5
	}

	return lane
}

// isZombieType 判断行为类型是否是僵尸（包括各种活跃状态）
// 返回 true 的状态会被除草车消灭
func isZombieType(behaviorType components.BehaviorType) bool {
	return behaviorType == components.BehaviorZombieBasic ||
		behaviorType == components.BehaviorZombieEating ||
		behaviorType == components.BehaviorZombieConehead ||
		behaviorType == components.BehaviorZombieBuckethead
}

// triggerFinalWaveWarning 已废弃：统一由 FlagWaveWarningSystem 处理
//
// @deprecated Story 17.6+统一后，此方法不再被调用
func (s *LevelSystem) triggerFinalWaveWarning() {
	log.Printf("[LevelSystem] [DEPRECATED] triggerFinalWaveWarning called - should use FlagWaveWarningSystem")

	s.gameState.ShowingFinalWave = true

	if audioPlayer := s.resourceManager.GetAudioPlayer("SOUND_AWOOGA"); audioPlayer != nil {
		audioPlayer.Rewind()
		audioPlayer.Play()
	}

	centerX := float64(config.ScreenWidth) / 2
	centerY := float64(config.ScreenHeight) / 2

	warningEntity, err := entities.NewFinalWaveWarningEntity(
		s.entityManager,
		s.resourceManager,
		centerX,
		centerY,
	)

	if err != nil {
		log.Printf("[LevelSystem] ERROR: Failed to create FinalWave warning entity: %v", err)
		return
	}

	ecs.AddComponent(s.entityManager, warningEntity, &components.AnimationCommandComponent{
		UnitID:    "finalwave",
		ComboName: "warning",
		Processed: false,
	})
}

// ========================================
// 关卡进度条支持
// ========================================

// SetProgressBarEntity 设置关卡进度条实体ID（由 GameScene 初始化时调用）
func (s *LevelSystem) SetProgressBarEntity(entityID ecs.EntityID) {
	s.progressBarEntityID = entityID
	s.initializeProgressBar()
}

// initializeProgressBar 初始化进度条（计算总僵尸数、旗帜位置）
//
// Story 11.5: 增加原版进度条机制的初始化
func (s *LevelSystem) initializeProgressBar() {
	if s.progressBarEntityID == 0 {
		return
	}

	progressBar, ok := ecs.GetComponent[*components.LevelProgressBarComponent](s.entityManager, s.progressBarEntityID)
	if !ok {
		log.Printf("[LevelSystem] WARNING: LevelProgressBarComponent not found for entity %d", s.progressBarEntityID)
		return
	}

	// 1. 计算总僵尸数（废弃字段，保留向后兼容）
	totalZombies := s.calculateTotalZombies()
	progressBar.TotalZombies = totalZombies
	progressBar.KilledZombies = 0
	progressBar.ProgressPercent = 0.0

	// 2. 计算旗帜位置百分比
	if s.gameState.CurrentLevel != nil && len(s.gameState.CurrentLevel.FlagWaves) > 0 {
		progressBar.FlagPositions = s.calculateFlagPositions()
	} else {
		progressBar.FlagPositions = []float64{} // 无旗帜
	}

	// 3. 设置关卡文本
	if s.gameState.CurrentLevel != nil {
		progressBar.LevelText = "关卡 " + s.gameState.CurrentLevel.ID
	}

	// 4. 默认只显示文本（第一波生成前）
	progressBar.ShowLevelTextOnly = true

	// === Story 11.5: 初始化原版进度条机制 ===
	s.initProgressBarStructure(progressBar)

	log.Printf("[LevelSystem] Progress bar initialized: total zombies=%d, flags=%v, totalWaves=%d",
		totalZombies, progressBar.FlagPositions, progressBar.TotalWaves)
}

// calculateTotalZombies 计算关卡总僵尸数
func (s *LevelSystem) calculateTotalZombies() int {
	if s.gameState.CurrentLevel == nil {
		return 0
	}

	total := 0
	for _, wave := range s.gameState.CurrentLevel.Waves {
		// 新格式：Zombies 字段
		for _, zombie := range wave.Zombies {
			total += zombie.Count
		}
		// 旧格式：OldZombies 字段（向后兼容）
		for _, zombie := range wave.OldZombies {
			total += zombie.Count
		}
	}

	return total
}

// calculateFlagPositions 计算旗帜在进度条上的位置百分比
//
// Story 11.5 修复：使用双段式结构计算旗帜位置
// 原版机制：
//   - 进度条总长 = 150 单位
//   - 红字波段 = 旗帜数 × 12（每个旗帜波刷新时立即 +12）
//   - 普通波段 = 150 - 红字波段（平均分配给普通波）
//   - 旗帜位置 = 该旗帜波刷新前的进度位置
//
// 计算公式：
//
//	旗帜位置 = (已完成红字波数 × 12 + 已完成普通波数 × 每波普通进度) / 150
func (s *LevelSystem) calculateFlagPositions() []float64 {
	if s.gameState.CurrentLevel == nil {
		return []float64{}
	}

	levelConfig := s.gameState.CurrentLevel
	flagWaves := levelConfig.FlagWaves
	totalWaves := len(levelConfig.Waves)
	flagCount := len(flagWaves)

	if flagCount == 0 || totalWaves == 0 {
		return []float64{}
	}

	// 双段式结构计算
	totalLength := float64(config.ProgressBarTotalLength)       // 150
	flagSegment := float64(config.ProgressBarFlagSegmentLength) // 12
	normalSegment := totalLength - float64(flagCount)*flagSegment

	// 普通波数 = 总波数 - 旗帜波数
	normalWaveCount := totalWaves - flagCount

	// 每波普通进度
	progressPerNormalWave := 0.0
	if normalWaveCount > 0 {
		progressPerNormalWave = normalSegment / float64(normalWaveCount)
	}

	positions := make([]float64, 0, flagCount)

	// 构建旗帜波索引集合（用于快速查找）
	flagWaveSet := make(map[int]bool)
	for _, idx := range flagWaves {
		flagWaveSet[idx] = true
	}

	// 遍历所有波次，计算每个旗帜波的位置
	completedFlags := 0
	completedNormalWaves := 0

	for waveIdx := 0; waveIdx < totalWaves; waveIdx++ {
		if flagWaveSet[waveIdx] {
			// 这是旗帜波，计算其位置（刷新前的进度）
			flagProgress := float64(completedFlags) * flagSegment
			normalProgress := float64(completedNormalWaves) * progressPerNormalWave
			position := (flagProgress + normalProgress) / totalLength

			// 进度条从右到左显示，旗帜段在左边，需要反转位置
			// 原始位置 92% -> 显示位置 8%（1 - 0.92 = 0.08）
			position = 1.0 - position

			// 限制在 [0, 1] 范围内
			if position < 0 {
				position = 0
			}
			positions = append(positions, position)

			completedFlags++
		} else {
			completedNormalWaves++
		}
	}

	return positions
}

// UpdateProgressBar 更新进度条进度
//
// Story 11.5: 实现原版进度条的双重进度计算和虚拟/现实追踪
// 每帧调用，更新游戏时钟、虚拟进度和现实进度
func (s *LevelSystem) UpdateProgressBar() {
	if s.progressBarEntityID == 0 {
		return
	}

	progressBar, ok := ecs.GetComponent[*components.LevelProgressBarComponent](s.entityManager, s.progressBarEntityID)
	if !ok {
		log.Printf("[ProgressBar] ERROR: failed to get progress bar component for entity %d", s.progressBarEntityID)
		return
	}

	// === 废弃逻辑（向后兼容） ===
	// 统计当前击杀的僵尸数（通过 GameState.ZombiesKilled）
	killedZombies := s.gameState.ZombiesKilled
	progressBar.KilledZombies = killedZombies

	// === Story 11.5: 原版进度条机制 ===

	// 1. 更新游戏时钟（厘秒）
	s.updateGameTickCS(progressBar)

	// 2. 更新血量追踪
	s.updateWaveHealthTracking(progressBar)

	// 记录旧的虚拟进度用于调试
	oldVirtual := progressBar.VirtualProgress
	oldReal := progressBar.RealProgress

	// 3. 计算虚拟进度
	s.calculateVirtualProgress(progressBar)

	// 4. 更新现实进度（平滑追踪）
	s.updateRealProgress(progressBar)

	// 调试日志：每秒输出一次（检查整秒边界）
	currentSecond := progressBar.GameTickCS / 100
	if currentSecond > 0 && progressBar.GameTickCS%100 < 10 {
		log.Printf("[ProgressBar] wave=%d, gameTickCS=%d, virtual=%.4f->%.4f, real=%.4f->%.4f, timeProgress=%.4f, dmgProgress=%.4f, WaveInitialDelay=%.2f",
			progressBar.CurrentWaveNum, progressBar.GameTickCS,
			oldVirtual, progressBar.VirtualProgress,
			oldReal, progressBar.RealProgress,
			s.calculateTimeProgress(progressBar),
			s.calculateDamageProgress(progressBar),
			progressBar.WaveInitialDelay)
	}

	// 5. 同步到废弃字段（向后兼容）
	progressBar.ProgressPercent = progressBar.RealProgress
	if progressBar.ProgressPercent > 1.0 {
		progressBar.ProgressPercent = 1.0
	}
}

// ShowProgressBar 显示完整进度条（第一波僵尸生成后调用）
func (s *LevelSystem) ShowProgressBar() {
	if s.progressBarEntityID == 0 {
		return
	}

	progressBar, ok := ecs.GetComponent[*components.LevelProgressBarComponent](s.entityManager, s.progressBarEntityID)
	if !ok {
		return
	}

	// 切换到完整显示模式
	progressBar.ShowLevelTextOnly = false
	log.Println("[LevelSystem] Progress bar now showing full display (background + progress + flags)")
}

// triggerZombiesWonFlow 触发完整的四阶段僵尸获胜流程（Story 8.8）
//
// 相比旧版 triggerZombiesWon()，新流程包含：
// - Phase 1: 游戏冻结（1.5秒）- 植物停止攻击、子弹消失、UI隐藏、音乐淡出
// - Phase 2: 僵尸入侵动画 - 触发僵尸继续行走至屏幕外
// - Phase 3: 惨叫与"吃脑子"动画（3-4秒）- 惨叫音效、咀嚼音效、屏幕抖动
// - Phase 4: 游戏结束对话框 - 显示"再次尝试"/"返回主菜单"按钮
//
// 参数：
//   - triggerZombieID: 触发失败的僵尸实体ID（在 Phase 2 继续移动）
//
// 执行步骤：
//  1. 创建流程控制实体
//  2. 添加 ZombiesWonPhaseComponent（阶段状态机）
//  3. 添加 GameFreezeComponent（冻结标记）
//  4. 交由 ZombiesWonPhaseSystem 管理流程推进
//
// 注意：
//   - 流程由 ZombiesWonPhaseSystem 自动推进，无需手动管理
//   - 遵循 ECS 零耦合原则，通过组件通信
func (s *LevelSystem) triggerZombiesWonFlow(triggerZombieID ecs.EntityID) {
	log.Printf("[LevelSystem] Triggering zombies won flow (4-phase), zombie ID:%d", triggerZombieID)

	// 使用 ZombiesWonPhaseSystem 提供的标准接口启动流程
	// 遵循 DRY 原则，复用业务逻辑
	flowEntityID := StartZombiesWonFlow(s.entityManager, triggerZombieID)

	log.Printf("[LevelSystem] Flow entity created (ID:%d), zombie ID:%d", flowEntityID, triggerZombieID)
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
func (s *LevelSystem) ResumeWaveTiming() {
	if s.waveTimingSystem == nil {
		return
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

// ========== Story 17.8: 血量触发加速刷新 ==========

// SetZombieStatsConfig 设置僵尸属性配置
//
// Story 17.8: 用于血量计算
// 在关卡初始化时调用
//
// 参数:
//   - cfg: 僵尸属性配置
func (s *LevelSystem) SetZombieStatsConfig(cfg *config.ZombieStatsConfig) {
	s.zombieStatsConfig = cfg
}

// SetZombiePhysicsConfig 设置僵尸物理配置
//
// Story 17.9: 用于类型化进家判定
// 在关卡初始化时调用
//
// 参数:
//   - cfg: 僵尸物理配置
func (s *LevelSystem) SetZombiePhysicsConfig(cfg *config.ZombiePhysicsConfig) {
	s.zombiePhysics = cfg
}

// initializeWaveHealth 初始化波次血量追踪
//
// Story 17.8: 在波次激活后调用，计算并设置本波僵尸总血量
//
// 参数:
//   - waveIndex: 波次索引（0-based）
func (s *LevelSystem) initializeWaveHealth(waveIndex int) {
	// 不使用 WaveTimingSystem 时跳过
	if !s.useWaveTimingSystem || s.waveTimingSystem == nil {
		return
	}

	// 获取关卡配置
	levelConfig := s.gameState.CurrentLevel
	if levelConfig == nil {
		return
	}

	// 确保波次索引有效
	if waveIndex < 0 || waveIndex >= len(levelConfig.Waves) {
		return
	}

	// 从关卡配置中获取波次僵尸信息
	waveConfig := levelConfig.Waves[waveIndex]
	zombieList := s.extractZombieSpawnInfo(&waveConfig)

	// 调用 WaveTimingSystem 初始化血量
	s.waveTimingSystem.InitializeWaveHealth(zombieList, s.zombieStatsConfig)
}

// extractZombieSpawnInfo 从波次配置中提取僵尸生成信息
//
// Story 17.8: 支持新格式 ZombieGroup 和旧格式 OldZombies
//
// 参数:
//   - waveConfig: 波次配置
//
// 返回:
//   - []ZombieSpawnInfo: 僵尸生成信息列表
func (s *LevelSystem) extractZombieSpawnInfo(waveConfig *config.WaveConfig) []ZombieSpawnInfo {
	var result []ZombieSpawnInfo

	// 处理新格式 ZombieGroup
	for _, group := range waveConfig.Zombies {
		result = append(result, ZombieSpawnInfo{
			Type:  group.Type,
			Count: group.Count,
		})
	}

	// 处理旧格式 OldZombies（向后兼容）
	for _, spawn := range waveConfig.OldZombies {
		count := spawn.Count
		if count == 0 {
			count = 1 // 默认生成 1 只
		}
		result = append(result, ZombieSpawnInfo{
			Type:  spawn.Type,
			Count: count,
		})
	}

	return result
}

// ========================================
// Story 11.5: 原版进度条机制实现
// ========================================

// initProgressBarStructure 初始化进度条双段式结构
//
// Story 11.5 Task 2: 实现双段式进度条分配逻辑
// - 计算总波次数
// - 计算红字波段和普通波段长度
// - 初始化波次追踪状态
//
// 参数:
//   - pb: 进度条组件
func (s *LevelSystem) initProgressBarStructure(pb *components.LevelProgressBarComponent) {
	if s.gameState.CurrentLevel == nil {
		return
	}

	levelConfig := s.gameState.CurrentLevel

	// 1. 设置进度条总长度
	pb.TotalProgressLength = config.ProgressBarTotalLength
	pb.FlagSegmentLength = config.ProgressBarFlagSegmentLength

	// 2. 计算总波次数
	pb.TotalWaves = len(levelConfig.Waves)
	if pb.TotalWaves == 0 {
		pb.TotalWaves = 1 // 防止除零
	}

	// 3. 计算红字波数量和普通波段长度
	flagCount := len(levelConfig.FlagWaves)
	pb.NormalSegmentBase = pb.TotalProgressLength - (flagCount * pb.FlagSegmentLength)
	if pb.NormalSegmentBase < 0 {
		pb.NormalSegmentBase = 0
	}

	// 4. 初始化波次追踪状态
	pb.CurrentWaveNum = 0 // 游戏开始前为 0，第一波激活后为 1
	pb.FlagWaveCount = 0

	// 5. 初始化时间和血量追踪
	pb.WaveStartTime = 0
	pb.WaveInitialDelay = 0
	pb.WaveInitialHealth = 0
	pb.WaveCurrentHealth = 0
	pb.WaveRequiredDamage = 0

	// 6. 初始化虚拟/现实进度
	pb.VirtualProgress = 0
	pb.RealProgress = 0

	// 7. 初始化游戏时钟
	pb.GameTickCS = 0
	pb.LastTrackUpdateCS = 0
}

// updateGameTickCS 更新游戏时钟（厘秒）
//
// Story 11.5 Task 7: 游戏时钟计数器
// 将游戏时间（秒）转换为厘秒累加
//
// 参数:
//   - pb: 进度条组件
func (s *LevelSystem) updateGameTickCS(pb *components.LevelProgressBarComponent) {
	// 从 GameState 获取当前关卡时间（秒）
	levelTime := s.gameState.LevelTime

	// 转换为厘秒
	pb.GameTickCS = int(levelTime * config.CentisecondsPerSecond)
}

// updateWaveHealthTracking 更新血量追踪
//
// Story 11.5 Task 4: 实现血量追踪系统
// 每帧查询所有僵尸的当前血量总和
//
// 参数:
//   - pb: 进度条组件
func (s *LevelSystem) updateWaveHealthTracking(pb *components.LevelProgressBarComponent) {
	// 查询所有僵尸实体的血量
	totalHealth := 0.0

	zombieEntities := ecs.GetEntitiesWith2[
		*components.BehaviorComponent,
		*components.HealthComponent,
	](s.entityManager)

	for _, entityID := range zombieEntities {
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 只统计僵尸类型
		if !isZombieType(behavior.Type) {
			continue
		}

		health, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		if health.CurrentHealth > 0 {
			totalHealth += float64(health.CurrentHealth)
		}
	}

	pb.WaveCurrentHealth = totalHealth
}

// calculateVirtualProgress 计算虚拟进度
//
// Story 11.5 Task 3: 实现双重进度计算逻辑
// - 红字波段：每个红字波贡献 12 格
// - 普通波段：根据左端点/右端点和波进度计算
// - 波进度 = max(时间进度, 血量削减进度)
//
// Story 11.5 Task 5: 进度允许超过右端点（不限制在 1.0 内）
//
// 参数:
//   - pb: 进度条组件
func (s *LevelSystem) calculateVirtualProgress(pb *components.LevelProgressBarComponent) {
	if pb.TotalProgressLength == 0 {
		return
	}

	// 1. 红字波段进度
	flagProgress := float64(pb.FlagWaveCount * pb.FlagSegmentLength)

	// 2. 普通波段进度
	normalProgress := 0.0

	if pb.TotalWaves > 1 && pb.CurrentWaveNum > 0 {
		// 计算左端点和右端点（整数除法向下取整）
		leftEndpoint := float64((pb.CurrentWaveNum - 1) * pb.NormalSegmentBase / (pb.TotalWaves - 1))
		rightEndpoint := float64(pb.CurrentWaveNum * pb.NormalSegmentBase / (pb.TotalWaves - 1))

		// 计算波进度（取时间进度和血量削减进度的最大值）
		waveProgress := s.calculateWaveProgress(pb)

		// 限制波进度不超过 1.0，防止进度条溢出
		// 只有在波次真正切换时（通过 OnWaveActivated），才会进入下一段
		if waveProgress > 1.0 {
			waveProgress = 1.0
		}

		// 计算普通波段内的进度
		// Story 11.5 Task 5: 不限制 waveProgress，允许超过 1.0
		normalProgress = (rightEndpoint-leftEndpoint)*waveProgress + leftEndpoint
	} else if pb.TotalWaves == 1 && pb.CurrentWaveNum > 0 {
		// 单波次关卡：普通波段进度直接等于波进度
		waveProgress := s.calculateWaveProgress(pb)
		if waveProgress > 1.0 {
			waveProgress = 1.0
		}
		normalProgress = float64(pb.NormalSegmentBase) * waveProgress
	}

	// 3. 计算总虚拟进度（允许超过 1.0）
	pb.VirtualProgress = (flagProgress + normalProgress) / float64(pb.TotalProgressLength)
}

// calculateWaveProgress 计算当前波进度
//
// Story 11.5 Task 3: 双重进度计算
// 波进度 = max(时间进度, 血量削减进度)
//
// 参数:
//   - pb: 进度条组件
//
// 返回:
//   - float64: 波进度（0.0 到 1.0+，可超过 1.0）
func (s *LevelSystem) calculateWaveProgress(pb *components.LevelProgressBarComponent) float64 {
	timeProgress := s.calculateTimeProgress(pb)
	damageProgress := s.calculateDamageProgress(pb)

	// 取两者最大值
	if timeProgress > damageProgress {
		return timeProgress
	}
	return damageProgress
}

// calculateTimeProgress 计算时间进度
//
// Story 11.5 Task 3: 时间进度 = 本波已存在时间 / 本波初始刷新倒计时
//
// 参数:
//   - pb: 进度条组件
//
// 返回:
//   - float64: 时间进度（0.0 到 1.0+）
func (s *LevelSystem) calculateTimeProgress(pb *components.LevelProgressBarComponent) float64 {
	if pb.WaveInitialDelay <= 0 {
		return 0
	}

	// 计算本波已存在时间
	elapsed := s.gameState.LevelTime - pb.WaveStartTime
	if elapsed < 0 {
		elapsed = 0
	}

	// 时间进度 = 已存在时间 / 初始倒计时
	return elapsed / pb.WaveInitialDelay
}

// calculateDamageProgress 计算血量削减进度
//
// Story 11.5 Task 3: 血量削减进度 = 已削减血量 / 所需削减血量
//
// 参数:
//   - pb: 进度条组件
//
// 返回:
//   - float64: 血量削减进度（0.0 到 1.0+）
func (s *LevelSystem) calculateDamageProgress(pb *components.LevelProgressBarComponent) float64 {
	if pb.WaveRequiredDamage <= 0 {
		return 0
	}

	// 计算已削减血量
	damageDealt := pb.WaveInitialHealth - pb.WaveCurrentHealth
	if damageDealt < 0 {
		damageDealt = 0
	}

	// 血量削减进度 = 已削减血量 / 所需削减血量
	return damageDealt / pb.WaveRequiredDamage
}

// updateRealProgress 更新现实进度（平滑追踪）
//
// Story 11.5 Task 6: 虚拟/现实双层追踪机制
// - 落后 1-6 格：每 20cs (0.2秒) 前进一格
// - 落后 7+ 格：每 5cs (0.05秒) 前进一格
// - 不落后时：不更新
//
// 参数:
//   - pb: 进度条组件
func (s *LevelSystem) updateRealProgress(pb *components.LevelProgressBarComponent) {
	// 计算虚拟与现实的差距（格数）
	diff := pb.VirtualProgress - pb.RealProgress
	if diff <= 0 {
		// 不落后或现实超前（回退情况），直接同步
		if pb.RealProgress > pb.VirtualProgress {
			pb.RealProgress = pb.VirtualProgress
		}
		return
	}

	// 转换为格数
	diffInUnits := int(diff * float64(pb.TotalProgressLength))
	if diffInUnits < 1 {
		return // 差距小于 1 格，不更新
	}

	// 计算每格对应的进度增量
	stepSize := 1.0 / float64(pb.TotalProgressLength)

	// 根据差距选择追踪速度
	if diffInUnits >= config.FastTrackThreshold {
		// 落后 7+ 格：快速追踪（每 5cs 前进一格）
		// 检查是否跨过了时间间隔倍数（避免因帧率波动跳过精确的倍数帧）
		if pb.GameTickCS/config.FastTrackIntervalCS > pb.LastTrackUpdateCS/config.FastTrackIntervalCS {
			pb.RealProgress += stepSize
			pb.LastTrackUpdateCS = pb.GameTickCS
		}
	} else {
		// 落后 1-6 格：慢速追踪（每 20cs 前进一格）
		// 检查是否跨过了时间间隔倍数
		if pb.GameTickCS/config.SlowTrackIntervalCS > pb.LastTrackUpdateCS/config.SlowTrackIntervalCS {
			pb.RealProgress += stepSize
			pb.LastTrackUpdateCS = pb.GameTickCS
		}
	}

	// 确保现实进度不超过虚拟进度
	if pb.RealProgress > pb.VirtualProgress {
		pb.RealProgress = pb.VirtualProgress
	}
}

// OnWaveActivated 波次激活时的回调
//
// Story 11.5: 更新进度条的波次追踪状态
// - 更新当前波次号
// - 记录本波开始时间和初始倒计时
// - 如果是红字波，增加红字波计数并立即推进虚拟进度
// - 初始化本波血量追踪
//
// 参数:
//   - waveIndex: 激活的波次索引（0-based）
//   - initialDelay: 下一波的初始倒计时（秒）
func (s *LevelSystem) OnWaveActivated(waveIndex int, initialDelay float64) {
	log.Printf("[LevelSystem] OnWaveActivated called: waveIndex=%d, initialDelay=%.2f, progressBarEntityID=%d",
		waveIndex, initialDelay, s.progressBarEntityID)

	if s.progressBarEntityID == 0 {
		log.Printf("[LevelSystem] OnWaveActivated: progressBarEntityID is 0, skipping")
		return
	}

	progressBar, ok := ecs.GetComponent[*components.LevelProgressBarComponent](s.entityManager, s.progressBarEntityID)
	if !ok {
		log.Printf("[LevelSystem] OnWaveActivated: failed to get progress bar component")
		return
	}

	// 更新当前波次号（从 1 开始）
	oldWaveNum := progressBar.CurrentWaveNum
	progressBar.CurrentWaveNum = waveIndex + 1

	// 记录本波开始时间
	progressBar.WaveStartTime = s.gameState.LevelTime

	// 记录下一波的初始倒计时（用于时间进度计算）
	progressBar.WaveInitialDelay = initialDelay

	log.Printf("[LevelSystem] OnWaveActivated: updated CurrentWaveNum %d->%d, WaveStartTime=%.2f, WaveInitialDelay=%.2f",
		oldWaveNum, progressBar.CurrentWaveNum, progressBar.WaveStartTime, progressBar.WaveInitialDelay)

	// 检查是否是红字波
	if s.gameState.CurrentLevel != nil {
		for _, flagWaveIndex := range s.gameState.CurrentLevel.FlagWaves {
			if flagWaveIndex == waveIndex {
				// 是红字波，增加红字波计数
				progressBar.FlagWaveCount++
				log.Printf("[LevelSystem] Flag wave %d activated, flagWaveCount=%d", waveIndex+1, progressBar.FlagWaveCount)
				break
			}
		}
	}

	// 初始化本波血量追踪
	s.initializeProgressBarWaveHealth(progressBar, waveIndex)
}

// initializeProgressBarWaveHealth 初始化进度条的波次血量追踪
//
// Story 11.5 Task 4: 在波次激活时记录本波僵尸总血量
//
// 参数:
//   - pb: 进度条组件
//   - waveIndex: 波次索引
func (s *LevelSystem) initializeProgressBarWaveHealth(pb *components.LevelProgressBarComponent, waveIndex int) {
	if s.gameState.CurrentLevel == nil {
		return
	}

	levelConfig := s.gameState.CurrentLevel
	if waveIndex < 0 || waveIndex >= len(levelConfig.Waves) {
		return
	}

	// 获取本波僵尸配置
	waveConfig := levelConfig.Waves[waveIndex]
	zombieList := s.extractZombieSpawnInfo(&waveConfig)

	// 计算本波僵尸总血量
	totalHealth := 0.0
	for _, zombie := range zombieList {
		// 获取僵尸类型的血量
		zombieHealth := s.getZombieTypeHealth(zombie.Type)
		totalHealth += float64(zombie.Count) * zombieHealth
	}

	pb.WaveInitialHealth = totalHealth
	pb.WaveCurrentHealth = totalHealth

	// 设置所需削减血量（用于血量进度计算）
	// 通常等于总血量，但可以根据关卡配置调整
	pb.WaveRequiredDamage = totalHealth

	log.Printf("[LevelSystem] Wave %d health initialized: initial=%.0f, required=%.0f",
		waveIndex+1, pb.WaveInitialHealth, pb.WaveRequiredDamage)
}

// getZombieTypeHealth 获取僵尸类型的血量
//
// 参数:
//   - zombieType: 僵尸类型字符串
//
// 返回:
//   - float64: 僵尸总血量（本体 + 饰品）
func (s *LevelSystem) getZombieTypeHealth(zombieType string) float64 {
	// 如果有僵尸配置，从配置中获取
	if s.zombieStatsConfig != nil {
		stats, ok := s.zombieStatsConfig.GetZombieStats(zombieType)
		if ok && stats != nil {
			// 总血量 = 本体血量 + I类饰品血量 + II类饰品血量
			return float64(stats.BaseHealth + stats.Tier1AccessoryHealth + stats.Tier2AccessoryHealth)
		}
	}

	// 默认血量（基于类型）
	switch zombieType {
	case "basic":
		return 200
	case "conehead":
		return 560
	case "buckethead":
		return 1300
	case "polevaulter":
		return 335
	case "newspaperzombie":
		return 331
	case "screendoor":
		return 1400
	case "footballzombie":
		return 1670
	case "dancingzombie":
		return 335
	case "backupdancer":
		return 200
	case "snorkel":
		return 200
	case "zomboni":
		return 1350
	case "bobsled":
		return 200
	case "dolphinrider":
		return 200
	case "jackinthebox":
		return 500
	case "balloon":
		return 200
	case "digger":
		return 200
	case "pogo":
		return 200
	case "yeti":
		return 1350
	case "bungee":
		return 450
	case "ladder":
		return 500
	case "catapult":
		return 850
	case "gargantuar":
		return 3000
	case "imp":
		return 70
	case "drzomboss":
		return 40000
	default:
		return 200 // 默认普通僵尸血量
	}
}

// cleanupAllSunEntities 清理场上所有阳光实体
//
// 在游戏胜利时调用，清理所有天空掉落和向日葵生成的阳光
// 确保奖励动画阶段场上不会残留阳光
func (s *LevelSystem) cleanupAllSunEntities() {
	// 查询所有阳光实体
	sunEntities := ecs.GetEntitiesWith1[*components.SunComponent](s.entityManager)

	if len(sunEntities) == 0 {
		return
	}

	// 销毁所有阳光实体
	for _, entityID := range sunEntities {
		s.entityManager.DestroyEntity(entityID)
	}

	// 立即清理标记的实体
	s.entityManager.RemoveMarkedEntities()

	log.Printf("[LevelSystem] Cleaned up %d sun entities on victory", len(sunEntities))
}
