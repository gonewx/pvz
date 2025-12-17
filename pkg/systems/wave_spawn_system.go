package systems

import (
	"log"
	"math/rand"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
)

// ============================================================================
// 波次生成系统核心（WaveSpawnSystem）
// ============================================================================

// WaveDispersionAllocator 波次分散分配器
//
// 用于跟踪同一波次中每行已分配的僵尸数量，
// 以便计算分散的激活延迟和X坐标偏移
type WaveDispersionAllocator struct {
	// rowZombieCount 每行已分配的僵尸数量 (key: row 0-4)
	rowZombieCount map[int]int

	// dispersionConfig 分散配置
	dispersionConfig *config.SpawnDispersionConfig
}

// NewWaveDispersionAllocator 创建波次分散分配器
func NewWaveDispersionAllocator(dispersionConfig *config.SpawnDispersionConfig) *WaveDispersionAllocator {
	return &WaveDispersionAllocator{
		rowZombieCount:   make(map[int]int),
		dispersionConfig: dispersionConfig,
	}
}

// Reset 重置分配器（每波开始时调用）
func (a *WaveDispersionAllocator) Reset() {
	a.rowZombieCount = make(map[int]int)
}

// AllocateZombie 为指定行分配一个僵尸，返回该僵尸的激活延迟和X偏移
//
// 参数:
//   - row: 行号（0-4）
//
// 返回:
//   - activationDelay: 激活延迟（秒）
//   - xOffset: X坐标偏移（像素，网格相对坐标）
func (a *WaveDispersionAllocator) AllocateZombie(row int) (activationDelay float64, xOffset float64) {
	// 获取该行当前已分配的僵尸数量
	countInRow := a.rowZombieCount[row]

	// 计算激活延迟：基础延迟 + 同行累加延迟 + 随机抖动
	baseDelay := config.ZombieActivationDelayMin
	if a.dispersionConfig != nil && a.dispersionConfig.Enabled {
		// 同行第N个僵尸的额外延迟 = (N-1) * step + random(0, jitter)
		sameRowDelay := float64(countInRow) * a.dispersionConfig.SameRowDelayStep
		jitter := rand.Float64() * a.dispersionConfig.SameRowDelayJitter
		activationDelay = baseDelay + sameRowDelay + jitter
	} else {
		// 未启用分散时，使用原来的随机延迟
		activationDelay = baseDelay + rand.Float64()*(config.ZombieActivationDelayMax-config.ZombieActivationDelayMin)
	}

	// 计算X坐标偏移：行交错偏移 + 随机抖动
	if a.dispersionConfig != nil && a.dispersionConfig.Enabled {
		// 使用交错模式：行0、3用偏移0，行1、4用偏移1，行2用偏移2
		rowPattern := row % 3
		baseOffset := float64(rowPattern) * a.dispersionConfig.RowOffsetBase
		// 随机抖动 [-jitter, +jitter]
		jitter := (rand.Float64()*2 - 1) * a.dispersionConfig.RowOffsetJitter
		xOffset = baseOffset + jitter
	}

	// 增加该行的僵尸计数
	a.rowZombieCount[row]++

	log.Printf("[WaveDispersionAllocator] Row %d: zombie #%d, delay=%.2fs, xOffset=%.1f",
		row, countInRow+1, activationDelay, xOffset)

	return activationDelay, xOffset
}

// WaveSpawnSystem 波次生成系统
//
// 职责：
//   - 预生成所有僵尸实体（关卡开始时）
//   - 按波次激活僵尸（使其开始移动）
//   - 处理不同僵尸类型的工厂调用
//   - Story 8.1: 验证僵尸生成行是否在 EnabledLanes 中
//   - Story 17.3: 验证僵尸生成是否符合限制规则
//   - Story 17.9: 使用精确的僵尸出生坐标
//   - Story 8.9: 支持 ExtraPoints 波次类型的动态僵尸分配
//   - 僵尸出场分散：同行时间错开 + 跨行空间错开
//
// 架构说明：
//   - 作为 LevelSystem 的依赖，由 LevelSystem 调用
//   - 使用僵尸工厂函数创建实体（entities 包）
//   - 遵循数据驱动原则：根据配置文件生成僵尸
//
// 预生成机制：
//  1. PreSpawnAllWaves() 在关卡开始时调用，预生成所有僵尸
//  2. ActivateWave(waveIndex) 在波次时间到达时调用，激活指定波次的僵尸
type WaveSpawnSystem struct {
	entityManager       *ecs.EntityManager
	resourceManager     *game.ResourceManager
	levelConfig         *config.LevelConfig         // 关卡配置（用于验证行数限制）
	gameState           *game.GameState             // 用于更新僵尸生成计数
	spawnRules          *config.SpawnRulesConfig    // Story 17.3: 僵尸生成规则配置
	constraintID        ecs.EntityID                // Story 17.3: 生成限制组件实体ID
	laneAllocator       *LaneAllocator              // Story 17.4: 行分配器系统
	zombiePhysics       *config.ZombiePhysicsConfig // Story 17.9: 僵尸物理配置（出生点、进家边界）
	dispersionAllocator *WaveDispersionAllocator    // 僵尸出场分散分配器
}

// NewWaveSpawnSystem 创建波次生成系统
//
// 参数：
//
//	em - 实体管理器
//	rm - 资源管理器
//	lc - 关卡配置（用于验证行数限制）
//	gs - 游戏状态（用于更新僵尸生成计数）
//	sr - Story 17.3: 僵尸生成规则配置（可选，nil 表示不启用限制检查）
//	zp - Story 17.9: 僵尸物理配置（可选，nil 表示使用默认坐标）
//
// Removed ReanimSystem dependency, using AnimationCommand component
func NewWaveSpawnSystem(em *ecs.EntityManager, rm *game.ResourceManager, lc *config.LevelConfig, gs *game.GameState, sr *config.SpawnRulesConfig, zp *config.ZombiePhysicsConfig) *WaveSpawnSystem {
	sys := &WaveSpawnSystem{
		entityManager:   em,
		resourceManager: rm,
		levelConfig:     lc,
		gameState:       gs,
		spawnRules:      sr,
		zombiePhysics:   zp,
	}

	// Story 17.3: 如果提供了生成规则，创建限制检查组件实体
	if sr != nil {
		sys.constraintID = sys.createConstraintEntity()
	}

	// Story 17.4: 创建并初始化行分配器
	sys.laneAllocator = NewLaneAllocator(em)
	// 冒险模式初始权重为 1，rowMax 根据场景类型确定（前院5行，后院6行）
	rowMax := 5 // 默认值
	if lc != nil && lc.RowMax > 0 {
		rowMax = lc.RowMax
	}
	sys.laneAllocator.InitializeLanes(rowMax, 1.0)

	// 初始化分散分配器
	var dispersionConfig *config.SpawnDispersionConfig
	if zp != nil {
		dispersionConfig = &zp.SpawnDispersion
	}
	sys.dispersionAllocator = NewWaveDispersionAllocator(dispersionConfig)

	return sys
}

// SpawnWave 生成一波僵尸（已废弃）
//
// 该方法已被 PreSpawnAllWaves + ActivateWave 取代
// 保留以向后兼容，但不推荐使用
//
// 根据波次配置，遍历所有僵尸生成配置，调用对应的僵尸工厂函数
//
// 参数：
//
//	waveConfig - 波次配置，包含僵尸类型、行数、数量等信息
//
// 返回：
//
//	生成的僵尸总数
func (s *WaveSpawnSystem) SpawnWave(waveConfig config.WaveConfig) int {
	totalSpawned := 0

	// 支持旧格式 OldZombies
	if len(waveConfig.OldZombies) > 0 {
		for _, zombieSpawn := range waveConfig.OldZombies {
			// 根据 Count 生成多个僵尸
			for i := 0; i < zombieSpawn.Count; i++ {
				// 生成单个僵尸，传递索引以计算额外的X偏移（避免重叠）
				entityID := s.spawnZombieWithOffset(zombieSpawn.Type, zombieSpawn.Lane, i)
				if entityID != 0 {
					totalSpawned++
					log.Printf("[WaveSpawnSystem] Spawned zombie: type=%s, lane=%d, index=%d, entityID=%d",
						zombieSpawn.Type, zombieSpawn.Lane, i, entityID)
				}
			}
		}
	}

	return totalSpawned
}

// PreSpawnAllWaves 预生成所有波次的僵尸
//
// 在关卡开始时调用，一次性生成所有僵尸并放置在屏幕右侧站位
// 僵尸初始状态为"待命"（不移动），等待 ActivateWave() 激活
//
// 支持新的 ZombieGroup 格式（随机行选择）
// Story 17.3: 在生成每波前更新限制组件的当前波次
//
// 返回：
//
//	生成的僵尸总数
func (s *WaveSpawnSystem) PreSpawnAllWaves() int {
	if s.levelConfig == nil {
		log.Printf("[WaveSpawnSystem] ERROR: No level config, cannot pre-spawn zombies")
		return 0
	}

	totalSpawned := 0
	log.Printf("[WaveSpawnSystem] Pre-spawning all zombies for %d waves", len(s.levelConfig.Waves))

	// 遍历所有波次
	for waveIndex, waveConfig := range s.levelConfig.Waves {
		// 每波开始时重置分散分配器（用于计算同行僵尸的时间错开和空间错开）
		s.dispersionAllocator.Reset()

		// Story 17.3: 更新限制组件的当前波次编号
		if s.spawnRules != nil && s.constraintID != 0 {
			if constraint, ok := ecs.GetComponent[*components.SpawnConstraintComponent](s.entityManager, s.constraintID); ok {
				constraint.CurrentWaveNum = waveIndex + 1 // 波次从 1 开始
			}
		}

		// 支持新格式 ZombieGroup
		if len(waveConfig.Zombies) > 0 {
			// 遍历本波的所有僵尸组配置
			for groupIndex, zombieGroup := range waveConfig.Zombies {
				// 为组内每个僵尸预选一个随机行（从 lanes 列表中选择）
				for i := 0; i < zombieGroup.Count; i++ {
					// Story 17.5: 使用 LaneAllocator 选择行（带合法行判定）
					// Story 17.2: 传入 laneRestriction 波次级行限制
					selectedLane := s.laneAllocator.SelectLane(
						zombieGroup.Type,
						s.levelConfig.SceneType,
						s.spawnRules,
						s.levelConfig.EnabledLanes,
						waveConfig.LaneRestriction,
					)
					s.laneAllocator.UpdateLaneCounters(selectedLane)

					entityID := s.spawnZombieForWave(zombieGroup.Type, selectedLane, waveIndex, groupIndex*100+i)
					if entityID != 0 {
						totalSpawned++
					}
				}
			}
		}

		// 向后兼容：支持旧格式 OldZombies
		if len(waveConfig.OldZombies) > 0 {
			for _, zombieSpawn := range waveConfig.OldZombies {
				// 生成多个僵尸
				for i := 0; i < zombieSpawn.Count; i++ {
					entityID := s.spawnZombieForWave(zombieSpawn.Type, zombieSpawn.Lane, waveIndex, i)
					if entityID != 0 {
						totalSpawned++
					}
				}
			}
		}

		// Story 8.9: 处理 ExtraPoints 类型波次的动态僵尸分配
		// 当波次有 extraPoints 点数时，从僵尸池中动态选择僵尸
		if waveConfig.ExtraPoints > 0 {
			extraZombies := s.allocateZombiesFromPoints(waveConfig.ExtraPoints, waveIndex)
			for i, zombieType := range extraZombies {
				selectedLane := s.laneAllocator.SelectLane(
					zombieType,
					s.levelConfig.SceneType,
					s.spawnRules,
					s.levelConfig.EnabledLanes,
					waveConfig.LaneRestriction,
				)
				s.laneAllocator.UpdateLaneCounters(selectedLane)

				// 使用 1000+ 作为额外点数僵尸的索引，避免与固定僵尸冲突
				entityID := s.spawnZombieForWave(zombieType, selectedLane, waveIndex, 1000+i)
				if entityID != 0 {
					totalSpawned++
					log.Printf("[WaveSpawnSystem] ExtraPoints zombie spawned: type=%s, wave=%d, index=%d",
						zombieType, waveIndex+1, i)
				}
			}
		}
	}

	log.Printf("[WaveSpawnSystem] Pre-spawned %d zombies total", totalSpawned)
	return totalSpawned
}

// ActivateWave 激活指定波次的僵尸
//
// 使该波次的所有僵尸开始等待激活（根据各自的延迟时间）
// 僵尸会在延迟时间后才真正开始移动，实现散落入场效果
//
// 参数：
//
//	waveIndex - 波次索引（0-based）
//
// 返回：
//
//	标记为等待激活的僵尸数量
func (s *WaveSpawnSystem) ActivateWave(waveIndex int) int {
	// 查询所有带 ZombieWaveStateComponent 的僵尸
	zombieEntities := ecs.GetEntitiesWith1[*components.ZombieWaveStateComponent](s.entityManager)

	// 播放僵尸呻吟音效（每波激活时播放一次，随机选择）
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		groanSounds := []string{"SOUND_GROAN", "SOUND_GROAN2", "SOUND_GROAN3", "SOUND_GROAN4", "SOUND_GROAN5", "SOUND_GROAN6"}
		randomIndex := rand.Intn(len(groanSounds))
		audioManager.PlaySound(groanSounds[randomIndex])
	}

	pendingCount := 0
	for _, entityID := range zombieEntities {
		waveState, ok := ecs.GetComponent[*components.ZombieWaveStateComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 只处理指定波次且未激活的僵尸
		if waveState.WaveIndex == waveIndex && !waveState.IsActivated && !waveState.IsPendingActivation {
			// 标记为等待激活状态
			waveState.IsPendingActivation = true
			waveState.ActivationTimer = 0
			pendingCount++

			log.Printf("[WaveSpawnSystem] Zombie %d marked for pending activation (wave %d, delay=%.2fs)",
				entityID, waveIndex, waveState.ActivationDelay)
		}
	}

	log.Printf("[WaveSpawnSystem] Wave %d: %d zombies marked for pending activation", waveIndex, pendingCount)
	return pendingCount
}

// SpawnWaveRealtime 实时生成并激活指定波次的僵尸
//
// 与 PreSpawnAllWaves + ActivateWave 的预生成模式不同，此方法在波次触发时
// 实时生成僵尸并立即激活它们（开始移动和播放行走动画）
//
// 参数：
//
//	waveIndex - 波次索引（0-based）
//
// 返回：
//
//	生成的僵尸总数
func (s *WaveSpawnSystem) SpawnWaveRealtime(waveIndex int) int {
	if s.levelConfig == nil {
		log.Printf("[WaveSpawnSystem] ERROR: No level config, cannot spawn wave")
		return 0
	}

	if waveIndex < 0 || waveIndex >= len(s.levelConfig.Waves) {
		log.Printf("[WaveSpawnSystem] ERROR: Invalid wave index %d (total waves: %d)", waveIndex, len(s.levelConfig.Waves))
		return 0
	}

	waveConfig := s.levelConfig.Waves[waveIndex]
	totalSpawned := 0

	// 每波开始时重置分散分配器（用于计算同行僵尸的时间错开和空间错开）
	s.dispersionAllocator.Reset()

	// Story 17.3: 更新限制组件的当前波次编号
	if s.spawnRules != nil && s.constraintID != 0 {
		if constraint, ok := ecs.GetComponent[*components.SpawnConstraintComponent](s.entityManager, s.constraintID); ok {
			constraint.CurrentWaveNum = waveIndex + 1 // 波次从 1 开始
		}
	}

	log.Printf("[WaveSpawnSystem] SpawnWaveRealtime: wave %d, zombies groups=%d", waveIndex+1, len(waveConfig.Zombies))

	// 支持新格式 ZombieGroup
	if len(waveConfig.Zombies) > 0 {
		for groupIndex, zombieGroup := range waveConfig.Zombies {
			for i := 0; i < zombieGroup.Count; i++ {
				// Story 17.5: 使用 LaneAllocator 选择行
				selectedLane := s.laneAllocator.SelectLane(
					zombieGroup.Type,
					s.levelConfig.SceneType,
					s.spawnRules,
					s.levelConfig.EnabledLanes,
					waveConfig.LaneRestriction,
				)
				s.laneAllocator.UpdateLaneCounters(selectedLane)

				entityID := s.spawnAndActivateZombie(zombieGroup.Type, selectedLane, waveIndex, groupIndex*100+i)
				if entityID != 0 {
					totalSpawned++
				}
			}
		}
	}

	// 向后兼容：支持旧格式 OldZombies
	if len(waveConfig.OldZombies) > 0 {
		for _, zombieSpawn := range waveConfig.OldZombies {
			for i := 0; i < zombieSpawn.Count; i++ {
				entityID := s.spawnAndActivateZombie(zombieSpawn.Type, zombieSpawn.Lane, waveIndex, i)
				if entityID != 0 {
					totalSpawned++
				}
			}
		}
	}

	// Story 8.9: 处理 ExtraPoints 类型波次的动态僵尸分配
	// 当波次有 extraPoints 点数时，从僵尸池中动态选择僵尸
	if waveConfig.ExtraPoints > 0 {
		extraZombies := s.allocateZombiesFromPoints(waveConfig.ExtraPoints, waveIndex)
		for i, zombieType := range extraZombies {
			selectedLane := s.laneAllocator.SelectLane(
				zombieType,
				s.levelConfig.SceneType,
				s.spawnRules,
				s.levelConfig.EnabledLanes,
				waveConfig.LaneRestriction,
			)
			s.laneAllocator.UpdateLaneCounters(selectedLane)

			// 使用 1000+ 作为额外点数僵尸的索引，避免与固定僵尸冲突
			entityID := s.spawnAndActivateZombie(zombieType, selectedLane, waveIndex, 1000+i)
			if entityID != 0 {
				totalSpawned++
				log.Printf("[WaveSpawnSystem] ExtraPoints zombie spawned (realtime): type=%s, wave=%d, index=%d",
					zombieType, waveIndex+1, i)
			}
		}
	}

	// 增加已激活僵尸计数
	s.gameState.IncrementZombiesSpawned(totalSpawned)

	log.Printf("[WaveSpawnSystem] SpawnWaveRealtime: wave %d completed, spawned %d zombies", waveIndex+1, totalSpawned)
	return totalSpawned
}
