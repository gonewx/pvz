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
	resourceManager      *game.ResourceManager  // 用于加载 FinalWave 音效
	reanimSystem         *ReanimSystem          // 用于创建 FinalWave 动画实体
	rewardSystem         *RewardAnimationSystem // 用于触发奖励动画（Story 8.3）
	lawnmowerSystem      *LawnmowerSystem       // 用于检查除草车状态（Story 10.2）
	lastWaveWarningShown bool                   // 是否已显示最后一波提示

	// Story 11.2: 关卡进度条支持
	progressBarEntityID ecs.EntityID // 进度条实体ID（如果存在）
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
//	lawnmowerSystem - 除草车系统（可选，Story 10.2）
func NewLevelSystem(em *ecs.EntityManager, gs *game.GameState, waveSpawnSystem *WaveSpawnSystem, rm *game.ResourceManager, rs *ReanimSystem, rewardSystem *RewardAnimationSystem, lawnmowerSystem *LawnmowerSystem) *LevelSystem {
	return &LevelSystem{
		entityManager:        em,
		gameState:            gs,
		waveSpawnSystem:      waveSpawnSystem,
		resourceManager:      rm,
		reanimSystem:         rs,
		rewardSystem:         rewardSystem,
		lawnmowerSystem:      lawnmowerSystem,
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

	// Story 11.2: 检测第一波是否已激活（教学关卡）
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

	// Story 11.2: 更新进度条
	s.UpdateProgressBar()
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

	// Story 11.2: 第一波僵尸生成后显示完整进度条
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
	// Story 10.2: 检查是否有活跃的除草车
	// 原版行为：除草车完全消失后，才显示胜利动画
	hasActiveLawnmowers := false
	if s.lawnmowerSystem != nil {
		hasActiveLawnmowers = s.lawnmowerSystem.HasActiveLawnmowers()
	}

	// 只有在没有活跃除草车的情况下才能胜利
	if s.gameState.CheckVictory() && !hasActiveLawnmowers {
		s.gameState.SetGameResult("win")
		log.Println("[LevelSystem] Victory! All zombies defeated!")

		// Story 8.3: 从关卡配置读取奖励植物（如果配置了）
		if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.RewardPlant != "" {
			rewardPlant := s.gameState.CurrentLevel.RewardPlant
			s.gameState.GetPlantUnlockManager().UnlockPlant(rewardPlant)
			log.Printf("[LevelSystem] Unlocked plant: %s (completed level %s)", rewardPlant, s.gameState.CurrentLevel.ID)
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
// 失败条件（Story 10.2 增强）：
// 1. 如果启用了除草车系统：僵尸到达左侧边界 && 该行除草车已使用 → 游戏失败
// 2. 如果未启用除草车系统：僵尸到达左侧边界 → 游戏失败（原逻辑）
//
// 这样设计的原因：
// - 除草车是每行的最后防线，只在僵尸到达左侧时触发一次
// - 如果除���车未使用，僵尸到达左侧会触发除草车（不触发失败）
// - 如果除草车已使用，僵尸再次到达左侧直接失败（无最后防线）
func (s *LevelSystem) checkDefeatCondition() {
	// Story 10.2: 如果启用了除草车系统，检查除草车状态
	if s.lawnmowerSystem != nil {
		s.checkDefeatWithLawnmower()
		return
	}

	// 原逻辑：未启用除草车系统，僵尸到达左侧直接失败
	s.checkDefeatWithoutLawnmower()
}

// checkDefeatWithLawnmower 检查失败条件（有除草车）
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

		// 僵尸到达左边界
		if pos.X < DefeatBoundaryX {
			// 计算僵尸所在行
			lane := s.getEntityLane(pos.Y)

			// 检查该行除草车是否已使用
			if state.UsedLanes[lane] {
				// 除草车已使用，游戏失败
				s.gameState.SetGameResult("lose")
				log.Printf("[LevelSystem] Defeat! Zombie (ID:%d) reached the left boundary on lane %d (lawnmower used)", entityID, lane)
				return
			} else {
				// 除草车未使用，不触发失败（让除草车触发）
				log.Printf("[LevelSystem] Zombie (ID:%d) reached left boundary on lane %d, waiting for lawnmower to trigger", entityID, lane)
				// 注意：不 return，继续检查其他行是否有除草车用完的情况
			}
		}
	}
}

// checkDefeatWithoutLawnmower 检查失败条件（无除草车，原逻辑）
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

		// 僵尸到达左边界，游戏失败
		if pos.X < DefeatBoundaryX {
			s.gameState.SetGameResult("lose")
			log.Printf("[LevelSystem] Defeat! Zombie (ID:%d) reached the left boundary at X=%.0f", entityID, pos.X)
			return
		}
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

// ========================================
// Story 11.2: 关卡进度条支持
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
		for _, zombie := range wave.Zombies {
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
			for _, zombie := range s.gameState.CurrentLevel.Waves[i].Zombies {
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

