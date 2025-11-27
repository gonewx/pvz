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

	// Story 17.7: 旗帜波警告和最终波白字系统
	flagWaveWarningSystem *FlagWaveWarningSystem // 红字警告系统
	finalWaveTextSystem   *FinalWaveTextSystem   // 白字显示系统
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
	// 教学关卡使用 TutorialSystem 控制僵尸生成，不使用 WaveTimingSystem
	isTutorialLevel := gs.CurrentLevel != nil && gs.CurrentLevel.OpeningType == "tutorial"

	ls := &LevelSystem{
		entityManager:             em,
		gameState:                 gs,
		waveSpawnSystem:           waveSpawnSystem,
		resourceManager:           rm,
		rewardSystem:              rewardSystem,
		lawnmowerSystem:           lawnmowerSystem,
		lastWaveWarningShown:      false,            // 已废弃，保留向后兼容
		finalWaveWarningTriggered: false,            // 新标志位
		finalWaveWarningLeadTime:  3.0,              // 提前 3 秒
		useWaveTimingSystem:       !isTutorialLevel, // 教学关卡禁用 WaveTimingSystem
	}

	// Story 17.6: 如果有关卡配置且非教学关卡，创建 WaveTimingSystem
	if gs.CurrentLevel != nil && !isTutorialLevel {
		ls.waveTimingSystem = NewWaveTimingSystem(em, gs, gs.CurrentLevel)

		// Story 17.7: 创建旗帜波警告和最终波白字系统
		ls.flagWaveWarningSystem = NewFlagWaveWarningSystem(em, ls.waveTimingSystem)
		ls.finalWaveTextSystem = NewFinalWaveTextSystem(em, ls.waveTimingSystem)

		// Story 17.6: 自动初始化计时器，使用关卡配置的首波延迟
		ls.waveTimingSystem.InitializeTimerWithDelay(true, gs.CurrentLevel)
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

	// Story 17.6: 更新波次计时器（如果启用）
	if s.useWaveTimingSystem && s.waveTimingSystem != nil {
		s.waveTimingSystem.Update(deltaTime)

		// Story 17.7: 更新旗帜波警告系统
		if s.flagWaveWarningSystem != nil {
			s.flagWaveWarningSystem.Update(deltaTime)
		}

		// Story 17.7: 更新最终波白字系统
		if s.finalWaveTextSystem != nil {
			s.finalWaveTextSystem.Update(deltaTime)
		}

		// Story 17.7: 检查加速刷新条件
		s.checkAcceleratedRefresh()
	}

	// 检查并生成僵尸波次
	s.checkAndSpawnWaves()

	// 检查是否需要显示最后一波提示（基于时间提前量）
	s.checkFinalWaveWarning(deltaTime)

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
		return false
	}

	// 没有活跃僵尸，返回 true
	return true
}

// checkAndSpawnWaves 检查并激活到期的僵尸波次
//
// 遍历所有波次，找到时间已到且未激活的波次，调用 WaveSpawnSystem.ActivateWave() 激活僵尸
// 教学关卡由 TutorialSystem 控制僵尸激活，不使用此方法
//
// Story 17.6: 如果启用了 WaveTimingSystem，从计时系统获取触发信号
// Story 17.8: 在激活波次后初始化血量追踪
func (s *LevelSystem) checkAndSpawnWaves() {
	// 教学关卡：僵尸由 TutorialSystem 控制激活
	if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.OpeningType == "tutorial" {
		return
	}

	var waveIndex int

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
	} else {
		// 原有逻辑：从 GameState 获取
		waveIndex = s.gameState.GetCurrentWave()
		if waveIndex == -1 {
			// 没有到期的波次
			return
		}
	}

	// 调用 WaveSpawnSystem 激活僵尸（而不是生成）
	zombieCount := s.waveSpawnSystem.ActivateWave(waveIndex)

	// 标记波次已激活
	s.gameState.MarkWaveSpawned(waveIndex)

	// Story 17.8: 初始化波次血量追踪
	s.initializeWaveHealth(waveIndex)

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

	// 只有在没有活跃除草车的情况下才能胜利
	if s.gameState.CheckVictory() && !hasActiveLawnmowers {
		s.gameState.SetGameResult("win")
		log.Println("[LevelSystem] Victory! All zombies defeated!")

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

// isZombieType 判断行为类型是否是僵尸
func isZombieType(behaviorType components.BehaviorType) bool {
	return behaviorType == components.BehaviorZombieBasic ||
		behaviorType == components.BehaviorZombieConehead ||
		behaviorType == components.BehaviorZombieBuckethead
}

// ========================================
// 最后一波提示系统
// ========================================

// checkFinalWaveWarning 检查是否需要触发最后一波提示
//
// 基于时间的精确提示：
//   - 在最后一波到来前 3 秒触发提示
//   - 只触发一次（通过 finalWaveWarningTriggered 标志）
//   - 教学关卡同样适用（不受特殊规则影响）
//
// 参数：
//
//	deltaTime - 未使用（保留用于未来扩展）
func (s *LevelSystem) checkFinalWaveWarning(deltaTime float64) {
	// 如果已触发，直接返回
	if s.finalWaveWarningTriggered {
		return
	}

	// 检测是否接近最后一波
	if s.isFinalWaveApproaching() {
		s.triggerFinalWaveWarning()
		s.finalWaveWarningTriggered = true
	}
}

// isFinalWaveApproaching 检测最后一波是否即将到来
//
// Story 17.6: 当 WaveTimingSystem 启用时，最后一波警告由 FlagWaveWarningSystem 处理
// 此方法仅在 WaveTimingSystem 未启用时作为后备逻辑
//
// 判断条件：
//  1. CurrentWaveIndex 指向最后一波（len-1）或者已经触发完所有波次（len）
//  2. 最后一波尚未被激活
//  3. 正在等待下一波（IsWaitingForNextWave = true）
//
// 返回：
//
//	true - 最后一波即将到来，应触发提示
//	false - 不应触发提示
func (s *LevelSystem) isFinalWaveApproaching() bool {
	// Story 17.6: WaveTimingSystem 启用时，由 FlagWaveWarningSystem 处理警告
	if s.useWaveTimingSystem {
		return false
	}

	// 检查关卡配置是否存在
	if s.gameState.CurrentLevel == nil || len(s.gameState.CurrentLevel.Waves) == 0 {
		return false
	}

	totalWaves := len(s.gameState.CurrentLevel.Waves)
	lastWaveIndex := totalWaves - 1

	// 检查当前是否正在等待最后一波
	isWaitingForFinalWave := s.gameState.CurrentWaveIndex == lastWaveIndex

	if !isWaitingForFinalWave {
		return false // 不是在等待最后一波
	}

	// 检查最后一波是否已经激活
	if s.gameState.IsWaveSpawned(lastWaveIndex) {
		return false // 最后一波已经激活，不需要提示
	}

	// 必须处于等待下一波的状态（上一波已消灭完毕）
	if !s.gameState.IsWaitingForNextWave {
		return false
	}

	// Story 17.6: 简化逻辑，立即触发警告（由 WaveTimingSystem 管理实际延迟）
	return true
}

// triggerFinalWaveWarning 触发最后一波提示
//
// 执行步骤：
//  1. 设置 GameState 标志 ShowingFinalWave = true
//  2. 播放音效 SOUND_AWOOGA
//  3. 创建 FinalWave.reanim 动画实体（调用工厂函数）
//
// 注意：
//   - 音效使用 SOUND_AWOOGA（原版 "僵尸来袭" 音效）
//   - 动画实体由 FinalWaveWarningSystem 自动管理生命周期
func (s *LevelSystem) triggerFinalWaveWarning() {
	log.Printf("[LevelSystem] Triggering final wave warning!")

	// 设置 GameState 标志
	s.gameState.ShowingFinalWave = true

	// 播放音效：SOUND_AWOOGA（僵尸来袭音效）
	if audioPlayer := s.resourceManager.GetAudioPlayer("SOUND_AWOOGA"); audioPlayer != nil {
		audioPlayer.Rewind()
		audioPlayer.Play()
		log.Printf("[LevelSystem] Playing SOUND_AWOOGA")
	} else {
		log.Printf("[LevelSystem] WARNING: SOUND_AWOOGA not loaded")
	}

	// 创建提示动画实体（屏幕中央）
	// 使用配置常量确保位置正确
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

	// 使用组件通信替代直接调用
	// 使用配置化的 combo 播放非循环动画
	ecs.AddComponent(s.entityManager, warningEntity, &components.AnimationCommandComponent{
		UnitID:    "finalwave",
		ComboName: "warning",
		Processed: false,
	})
	log.Printf("[LevelSystem] 添加 FinalWave 动画命令 (entity: %d, combo: warning, loop: false)", warningEntity)

	log.Printf("[LevelSystem] Created FinalWave warning entity (ID: %d)", warningEntity)
}

// ========================================
// 已废弃方法（保留向后兼容）
// ========================================

// checkLastWaveWarning 已废弃：请使用 checkFinalWaveWarning
//
// 在最后一波即将生成时显示提示（倒数第二波消灭完毕后）
// 提示只显示一次
//
// 废弃原因：
//   - Story 11.3 需要基于时间的精确提示（提前 3 秒）
//   - 新方法 checkFinalWaveWarning 提供更准确的时机控制
func (s *LevelSystem) checkLastWaveWarning() {
	// 如果已经显示过，不再显示
	if s.lastWaveWarningShown {
		return
	}

	// 获取总波次数
	totalWaves := len(s.gameState.CurrentLevel.Waves)
	if totalWaves == 0 {
		return
	}

	// 在倒数第二波消灭完毕后（等待最后一波触发时）显示提示
	// 条件：当前波次索引 == totalWaves-1（最后一波） 且最后一波尚未生成
	currentWaveIndex := s.gameState.CurrentWaveIndex
	lastWaveIndex := totalWaves - 1

	if currentWaveIndex == lastWaveIndex && !s.gameState.IsWaveSpawned(lastWaveIndex) {
		s.showLastWaveWarning()
		s.lastWaveWarningShown = true
		log.Println("[LevelSystem] Last wave warning displayed!")
	}
}

// showLastWaveWarning 已废弃：请使用 triggerFinalWaveWarning
//
// 创建 FinalWave.reanim 动画实体，播放最后一波警告动画和音效
// 动画在屏幕中心显示，从大到小缩放并淡出
//
// 废弃原因：
//   - Story 11.3 统一使用 triggerFinalWaveWarning 方法
//   - 新方法使用 SOUND_AWOOGA 而不是 SOUND_FINALWAVE
func (s *LevelSystem) showLastWaveWarning() {
	// 设置 GameState 标志
	s.gameState.ShowingFinalWave = true

	// 播放最后一波音效
	if s.resourceManager != nil {
		sfx, err := s.resourceManager.LoadSoundEffect("assets/sounds/finalwave.ogg")
		if err != nil {
			log.Printf("[LevelSystem] WARNING: Failed to load finalwave.ogg: %v", err)
		} else {
			sfx.Play()
			log.Printf("[LevelSystem] Playing finalwave.ogg sound effect")
		}
	}

	// 创建 FinalWave 动画实体（显示在屏幕中心）
	// 动画会自动播放一次后消失
	// 工厂函数不再接受 reanimSystem 参数
	finalWaveEntityID, err := entities.CreateFinalWaveEntity(
		s.entityManager,
		s.resourceManager,
		400.0, // X坐标（屏幕中心，世界坐标）
		300.0, // Y坐标（屏幕中心）
	)

	if err != nil {
		log.Printf("[LevelSystem] ERROR: Failed to create FinalWave entity: %v", err)
	} else {
		// 使用组件通信初始化动画
		ecs.AddComponent(s.entityManager, finalWaveEntityID, &components.AnimationCommandComponent{
			AnimationName: "anim",
			Processed:     false,
		})
		log.Printf("[LevelSystem] Created FinalWave warning entity (ID: %d)", finalWaveEntityID)
	}
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
func (s *LevelSystem) initializeProgressBar() {
	if s.progressBarEntityID == 0 {
		return
	}

	progressBar, ok := ecs.GetComponent[*components.LevelProgressBarComponent](s.entityManager, s.progressBarEntityID)
	if !ok {
		log.Printf("[LevelSystem] WARNING: LevelProgressBarComponent not found for entity %d", s.progressBarEntityID)
		return
	}

	// 1. 计算总僵尸数
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

	log.Printf("[LevelSystem] Progress bar initialized: total zombies=%d, flags=%v", totalZombies, progressBar.FlagPositions)
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
func (s *LevelSystem) calculateFlagPositions() []float64 {
	if s.gameState.CurrentLevel == nil {
		return []float64{}
	}

	totalZombies := s.calculateTotalZombies()
	if totalZombies == 0 {
		return []float64{}
	}

	flagWaves := s.gameState.CurrentLevel.FlagWaves
	positions := make([]float64, 0, len(flagWaves))

	// 计算每个旗帜波次前已经出现的僵尸数
	for _, flagWaveIndex := range flagWaves {
		if flagWaveIndex < 0 || flagWaveIndex >= len(s.gameState.CurrentLevel.Waves) {
			continue
		}

		// 计算旗帜波次前的僵尸总数
		zombiesBeforeFlag := 0
		for i := 0; i < flagWaveIndex; i++ {
			// 新格式：Zombies 字段
			for _, zombie := range s.gameState.CurrentLevel.Waves[i].Zombies {
				zombiesBeforeFlag += zombie.Count
			}
			// 旧格式：OldZombies 字段（向后兼容）
			for _, zombie := range s.gameState.CurrentLevel.Waves[i].OldZombies {
				zombiesBeforeFlag += zombie.Count
			}
		}

		// 旗帜位置 = 旗帜波次前的僵尸数 / 总僵尸数
		flagPercent := float64(zombiesBeforeFlag) / float64(totalZombies)
		positions = append(positions, flagPercent)
	}

	return positions
}

// UpdateProgressBar 更新进度条进度（僵尸死亡时调用）
func (s *LevelSystem) UpdateProgressBar() {
	if s.progressBarEntityID == 0 {
		return
	}

	progressBar, ok := ecs.GetComponent[*components.LevelProgressBarComponent](s.entityManager, s.progressBarEntityID)
	if !ok {
		return
	}

	// 统计当前击杀的僵尸数（通过 GameState.ZombiesKilled）
	killedZombies := s.gameState.ZombiesKilled
	progressBar.KilledZombies = killedZombies

	// 更新进度百分比
	if progressBar.TotalZombies > 0 {
		progressBar.ProgressPercent = float64(killedZombies) / float64(progressBar.TotalZombies)
		if progressBar.ProgressPercent > 1.0 {
			progressBar.ProgressPercent = 1.0
		}
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

// ========================================
// 僵尸胜利动画
// ========================================

// triggerZombiesWon 触发僵尸胜利动画（旧版本）
//
// @deprecated 请使用 triggerZombiesWonFlow() 来触发完整的四阶段僵尸获胜流程
//
// 当僵尸到达左边界且所有除草车已用完时调用。
// 显示 ZombiesWon.reanim 动画，覆盖整个屏幕。
//
// 执行步骤：
//  1. 创建僵尸胜利动画实体（屏幕中央）
//  2. 使用 AnimationCommandComponent 播放动画
//
// 注意：
//   - 动画会一直显示直到玩家退出游戏
//   - 使用单动画模式播放 "anim_screen"
//   - 新代码请使用 triggerZombiesWonFlow() 实现完整流程（Story 8.8）
func (s *LevelSystem) triggerZombiesWon() {
	log.Printf("[LevelSystem] Triggering zombies won animation!")

	// 创建僵尸胜利动画实体（屏幕中央）
	centerX := float64(config.ScreenWidth) / 2
	centerY := float64(config.ScreenHeight) / 2

	zombiesWonEntity, err := entities.NewZombiesWonEntity(
		s.entityManager,
		s.resourceManager,
		centerX,
		centerY,
	)

	if err != nil {
		log.Printf("[LevelSystem] ERROR: Failed to create ZombiesWon entity: %v", err)
		return
	}

	// 使用组件通信播放动画
	// 直接使用单动画模式播放 anim_screen
	ecs.AddComponent(s.entityManager, zombiesWonEntity, &components.AnimationCommandComponent{
		AnimationName: "anim_screen", // 直接播放单个动画
		Processed:     false,
	})

	log.Printf("[LevelSystem] Created ZombiesWon entity (ID: %d), playing anim_screen animation", zombiesWonEntity)
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
func (s *LevelSystem) ResumeWaveTiming() {
	if s.waveTimingSystem != nil {
		s.waveTimingSystem.Resume()
	}
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

	// Story 17.7: 创建旗帜波警告和最终波白字系统
	s.flagWaveWarningSystem = NewFlagWaveWarningSystem(s.entityManager, s.waveTimingSystem)
	s.finalWaveTextSystem = NewFinalWaveTextSystem(s.entityManager, s.waveTimingSystem)

	log.Printf("[LevelSystem] WaveTimingSystem initialized with FlagWaveWarning and FinalWaveText systems")
}

// GetFlagWaveWarningSystem 获取旗帜波警告系统（用于测试）
//
// Story 17.7: 供测试和调试使用
func (s *LevelSystem) GetFlagWaveWarningSystem() *FlagWaveWarningSystem {
	return s.flagWaveWarningSystem
}

// GetFinalWaveTextSystem 获取最终波白字系统（用于测试）
//
// Story 17.7: 供测试和调试使用
func (s *LevelSystem) GetFinalWaveTextSystem() *FinalWaveTextSystem {
	return s.finalWaveTextSystem
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
