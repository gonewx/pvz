package systems

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
)

// 关卡流程常量
const (
	// DefeatBoundaryX 失败边界X坐标（僵尸到达此位置视为游戏失败）
	DefeatBoundaryX = 100.0

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
//
// 架构说明：
//   - 通过 GameState 单例管理关卡状态
//   - 依赖 WaveSpawnSystem 进行僵尸生成（通过构造函数注入）
//   - 遵循零耦合原则：不直接修改其他系统的状态
type LevelSystem struct {
	entityManager        *ecs.EntityManager
	gameState            *game.GameState
	waveSpawnSystem      *WaveSpawnSystem
	resourceManager      *game.ResourceManager   // 用于加载 FinalWave 音效
	reanimSystem         *ReanimSystem           // 用于创建 FinalWave 动画实体
	rewardSystem         *RewardAnimationSystem  // 用于触发奖励动画（Story 8.3）
	lastWaveWarningShown bool                    // 是否已显示最后一波提示
}

// NewLevelSystem 创建关卡管理系统
//
// 参数：
//
//	em - 实体管理器
//	gs - 游戏状态单例
//	waveSpawnSystem - 波次生成系统（依赖注入）
//	rm - 资源管理器（用于加载音效）
//	rs - Reanim系统（用于创建动画实体）
//	rewardSystem - 奖励动画系统（可选，Story 8.3）
func NewLevelSystem(em *ecs.EntityManager, gs *game.GameState, waveSpawnSystem *WaveSpawnSystem, rm *game.ResourceManager, rs *ReanimSystem, rewardSystem *RewardAnimationSystem) *LevelSystem {
	return &LevelSystem{
		entityManager:        em,
		gameState:            gs,
		waveSpawnSystem:      waveSpawnSystem,
		resourceManager:      rm,
		reanimSystem:         rs,
		rewardSystem:         rewardSystem,
		lastWaveWarningShown: false,
	}
}

// Update 更新关卡系统
//
// 执行流程：
//  1. 检查游戏是否结束，如果是则不处理
//  2. 更新关卡时间
//  3. 检查并生成到期的僵尸波次
//  4. 检查最后一波提示
//  5. 检查失败条件（僵尸到达左边界）
//  6. 检查胜利条件（所有僵尸消灭）
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

	// 检查并生成僵尸波次
	s.checkAndSpawnWaves()

	// 检查是否需要显示最后一波提示
	s.checkLastWaveWarning()

	// 检查失败条件（必须在胜利条件之前，优先级更高）
	s.checkDefeatCondition()

	// 检查胜利条件
	s.checkVictoryCondition()
}

// checkAndSpawnWaves 检查并激活到期的僵尸波次
//
// 遍历所有波次，找到时间已到且未激活的波次，调用 WaveSpawnSystem.ActivateWave() 激活僵尸
// 教学关卡由 TutorialSystem 控制僵尸激活，不使用此方法
func (s *LevelSystem) checkAndSpawnWaves() {
	// 教学关卡：僵尸由 TutorialSystem 控制激活
	if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.OpeningType == "tutorial" {
		return
	}

	waveIndex := s.gameState.GetCurrentWave()
	if waveIndex == -1 {
		// 没有到期的波次
		return
	}

	// 调用 WaveSpawnSystem 激活僵尸（而不是生成）
	zombieCount := s.waveSpawnSystem.ActivateWave(waveIndex)

	// 标记波次已激活
	s.gameState.MarkWaveSpawned(waveIndex)

	log.Printf("[LevelSystem] Wave %d activated: %d zombies", waveIndex+1, zombieCount)
}

// checkVictoryCondition 检查胜利条件
//
// 胜利条件：所有波次已生成 且 所有僵尸已消灭
// 如果达成胜利条件，设置游戏结果为 "win"
func (s *LevelSystem) checkVictoryCondition() {
	if s.gameState.CheckVictory() {
		s.gameState.SetGameResult("win")
		log.Println("[LevelSystem] Victory! All zombies defeated!")

		// Story 8.2: 关卡 1-1 完成后解锁向日葵
		if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.ID == "1-1" {
			s.gameState.GetPlantUnlockManager().UnlockPlant("sunflower")
			log.Println("[LevelSystem] Unlocked plant: sunflower (completed level 1-1)")
		}

		// Story 8.3: 检查是否有新植物解锁，触发奖励动画
		s.triggerRewardIfNeeded()
	}
}

// triggerRewardIfNeeded 检查是否有新植物解锁，如果有则触发奖励动画
func (s *LevelSystem) triggerRewardIfNeeded() {
	if s.rewardSystem == nil {
		return
	}

	// 获取最后解锁的植物
	lastUnlocked := s.gameState.GetPlantUnlockManager().GetLastUnlocked()
	if lastUnlocked == "" {
		log.Println("[LevelSystem] No new plant unlocked, skipping reward animation")
		return
	}

	// 触发奖励动画
	log.Printf("[LevelSystem] Triggering reward animation for plant: %s", lastUnlocked)
	s.rewardSystem.TriggerReward(lastUnlocked)

	// 清除最后解锁标记（避免重复触发）
	s.gameState.GetPlantUnlockManager().ClearLastUnlocked()
}

// checkDefeatCondition 检查失败条件
//
// 失败条件：任意僵尸的X坐标 < DefeatBoundaryX（到达屏幕左侧边界）
// 如果检测到失败，设置游戏结果为 "lose"
func (s *LevelSystem) checkDefeatCondition() {
	// 查询所有拥有 BehaviorComponent 和 PositionComponent 的实体
	// 然后通过 BehaviorComponent 的 Type 字段筛选僵尸
	zombieEntities := ecs.GetEntitiesWith2[
		*components.BehaviorComponent,
		*components.PositionComponent,
	](s.entityManager)

	// 检查是否有僵尸到达左边界
	for _, entityID := range zombieEntities {
		// 获取 BehaviorComponent
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 只检查僵尸类型的实体
		if !isZombieType(behavior.Type) {
			continue
		}

		// 获取位置组件
		pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 僵尸到达左边界，游戏失败
		if pos.X < DefeatBoundaryX {
			s.gameState.SetGameResult("lose")
			log.Printf("[LevelSystem] Defeat! Zombie (ID:%d) reached the left boundary at X=%.0f", entityID, pos.X)
			return // 只需检测到一个即可
		}
	}
}

// isZombieType 判断行为类型是否是僵尸
func isZombieType(behaviorType components.BehaviorType) bool {
	return behaviorType == components.BehaviorZombieBasic ||
		behaviorType == components.BehaviorZombieConehead ||
		behaviorType == components.BehaviorZombieBuckethead
}

// checkLastWaveWarning 检查是否需要显示最后一波提示
//
// 在最后一波即将生成时显示提示（倒数第二波消灭完毕后）
// 提示只显示一次
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

// showLastWaveWarning 显示最后一波提示
//
// 创建 FinalWave.reanim 动画实体，播放最后一波警告动画和音效
// 动画在屏幕中心显示，从大到小缩放并淡出
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
	finalWaveEntityID, err := entities.CreateFinalWaveEntity(
		s.entityManager,
		s.resourceManager,
		s.reanimSystem,
		400.0, // X坐标（屏幕中心，世界坐标）
		300.0, // Y坐标（屏幕中心）
	)

	if err != nil {
		log.Printf("[LevelSystem] ERROR: Failed to create FinalWave entity: %v", err)
	} else {
		log.Printf("[LevelSystem] Created FinalWave warning entity (ID: %d)", finalWaveEntityID)
	}
}
