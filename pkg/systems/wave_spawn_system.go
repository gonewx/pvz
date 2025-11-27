package systems

import (
	"log"
	"math/rand"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
)

// WaveSpawnSystem 波次生成系统
//
// 职责：
//   - 预生成所有僵尸实体（关卡开始时）
//   - 按波次激活僵尸（使其开始移动）
//   - 处理不同僵尸类型的工厂调用
//   - Story 8.1: 验证僵尸生成行是否在 EnabledLanes 中
//   - Story 17.3: 验证僵尸生成是否符合限制规则
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
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager
	levelConfig     *config.LevelConfig        // 关卡配置（用于验证行数限制）
	gameState       *game.GameState            // 用于更新僵尸生成计数
	spawnRules      *config.SpawnRulesConfig   // Story 17.3: 僵尸生成规则配置
	constraintID    ecs.EntityID               // Story 17.3: 生成限制组件实体ID
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
//
// Removed ReanimSystem dependency, using AnimationCommand component
func NewWaveSpawnSystem(em *ecs.EntityManager, rm *game.ResourceManager, lc *config.LevelConfig, gs *game.GameState, sr *config.SpawnRulesConfig) *WaveSpawnSystem {
	sys := &WaveSpawnSystem{
		entityManager:   em,
		resourceManager: rm,
		levelConfig:     lc,
		gameState:       gs,
		spawnRules:      sr,
	}

	// Story 17.3: 如果提供了生成规则，创建限制检查组件实体
	if sr != nil {
		sys.constraintID = sys.createConstraintEntity()
	}

	return sys
}

// createConstraintEntity 创建生成限制组件实体
// Story 17.3: 用于存储关卡级别的生成限制状态
func (s *WaveSpawnSystem) createConstraintEntity() ecs.EntityID {
	entityID := s.entityManager.CreateEntity()

	// 从关卡配置中提取允许的僵尸类型
	allowedTypes := s.extractAllowedZombieTypes()

	// 添加限制组件
	ecs.AddComponent(s.entityManager, entityID, &components.SpawnConstraintComponent{
		RedEyeCount:        0,
		CurrentWaveNum:     1, // 初始波次
		AllowedZombieTypes: allowedTypes,
		SceneType:          s.levelConfig.SceneType,
	})

	log.Printf("[WaveSpawnSystem] Created spawn constraint entity %d (scene: %s, allowed types: %d)",
		entityID, s.levelConfig.SceneType, len(allowedTypes))

	return entityID
}

// extractAllowedZombieTypes 从关卡配置中提取允许的僵尸类型
func (s *WaveSpawnSystem) extractAllowedZombieTypes() []string {
	typeSet := make(map[string]bool)

	// 从所有波次配置中提取僵尸类型
	for _, wave := range s.levelConfig.Waves {
		for _, zombie := range wave.Zombies {
			typeSet[zombie.Type] = true
		}
		// 兼容旧格式
		for _, zombie := range wave.OldZombies {
			typeSet[zombie.Type] = true
		}
	}

	// 转换为列表
	types := make([]string, 0, len(typeSet))
	for zombieType := range typeSet {
		types = append(types, zombieType)
	}

	return types
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
					// 从配置的 lanes 列表中随机选择一行
					randomLaneIndex := rand.Intn(len(zombieGroup.Lanes))
					selectedLane := zombieGroup.Lanes[randomLaneIndex] // 1-5

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
	}

	log.Printf("[WaveSpawnSystem] Pre-spawned %d zombies total", totalSpawned)
	return totalSpawned
}

// ActivateWave 激活指定波次的僵尸
//
// 使该波次的所有僵尸开始向左移动（进攻）
//
// 参数：
//
//	waveIndex - 波次索引（0-based）
//
// 返回：
//
//	激活的僵尸数量
func (s *WaveSpawnSystem) ActivateWave(waveIndex int) int {
	// 查询所有带 ZombieWaveStateComponent 的僵尸
	zombieEntities := ecs.GetEntitiesWith1[*components.ZombieWaveStateComponent](s.entityManager)

	activated := 0
	for _, entityID := range zombieEntities {
		waveState, ok := ecs.GetComponent[*components.ZombieWaveStateComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 只激活指定波次且未激活的僵尸
		if waveState.WaveIndex == waveIndex && !waveState.IsActivated {
			// 标记为已激活
			waveState.IsActivated = true

			// 获取僵尸当前位置
			pos, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
			if hasPos {
				// 计算当前所在行（0-4）
				currentRow := int((pos.Y - config.GridWorldStartY - config.ZombieVerticalOffset - config.CellHeight/2.0) / config.CellHeight)
				if currentRow < 0 {
					currentRow = 0
				}
				if currentRow > 4 {
					currentRow = 4
				}

				// 从 enabledLanes 中随机选择一个有效行作为目标行
				targetRow := s.randomEnabledLane()

				log.Printf("[WaveSpawnSystem] Zombie %d activating: currentRow=%d, targetRow=%d", entityID, currentRow, targetRow)

				// 如果当前行不是目标行，添加目标行组件
				if currentRow != targetRow {
					s.addTargetLaneComponent(entityID, targetRow, pos.Y)
					log.Printf("[WaveSpawnSystem] Zombie %d will transition from row %d to row %d", entityID, currentRow, targetRow)
				}
			}

			// 启动僵尸移动（设置X轴速度）
			if vel, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID); ok {
				// 如果僵尸还在行转换中（VY != 0），保持Y轴速度不变
				// 只设置X轴速度
				if vel.VX == 0 {
					vel.VX = -150.0 // 僵尸标准移动速度
					log.Printf("[WaveSpawnSystem] Activated zombie %d (wave %d, index %d), VX=%.1f",
						entityID, waveIndex, waveState.IndexInWave, vel.VX)
				}
			}

			// 使用组件通信替代直接调用
			// 僵尸使用配置驱动的动画组合（自动隐藏装备轨道）
			if behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID); ok {
				if behavior.ZombieAnimState == components.ZombieAnimIdle {
					behavior.ZombieAnimState = components.ZombieAnimWalking
					ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
						UnitID:    "zombie",
						ComboName: "walk",
						Processed: false,
					})
					log.Printf("[WaveSpawnSystem] Zombie %d 添加行走动画命令 (activated)", entityID)
				}
			}

			activated++

			// 增加已激活僵尸计数（用于计算场上僵尸数）
			// zombiesOnField = TotalZombiesSpawned - ZombiesKilled
			s.gameState.IncrementZombiesSpawned(1)
		}
	}

	log.Printf("[WaveSpawnSystem] Activated wave %d: %d zombies", waveIndex, activated)
	return activated
}

// spawnZombieForWave 为指定波次生成僵尸（预生成模式）
//
// 生成的僵尸初始状态为"待命"（不移动），需要调用 ActivateWave() 激活
//
// Story 17.3: 添加生成限制检查
//
// 参数：
//
//	zombieType - 僵尸类型
//	lane - 行号（1-5）
//	waveIndex - 所属波次索引（0-based）
//	indexInWave - 在本波中的索引（0, 1, 2...）
//
// 返回：
//
//	僵尸实体ID
func (s *WaveSpawnSystem) spawnZombieForWave(zombieType string, lane int, waveIndex int, indexInWave int) ecs.EntityID {
	// Story 17.3: 生成前检查限制规则
	if s.spawnRules != nil && s.constraintID != 0 {
		constraint, ok := ecs.GetComponent[*components.SpawnConstraintComponent](s.entityManager, s.constraintID)
		if ok {
			// 计算当前轮数
			// 公式: RoundNumber = TotalCompletedFlags / 2 - 1
			// 对于当前关卡，暂时假设 CompletedFlags = 0，即 RoundNumber = -1
			// TODO: 需要从 GameState 获取 TotalCompletedFlags
			roundNumber := -1 // 默认第一轮
			if s.gameState.TotalCompletedFlags > 0 {
				roundNumber = s.gameState.TotalCompletedFlags/2 - 1
			}

			// 转换 lane (1-5) 为 row (0-4)
			row := lane - 1
			if row < 0 {
				row = 0
			}
			if row > 4 {
				row = 4
			}

			// 使用纯函数验证
			valid, reason := ValidateZombieSpawn(zombieType, lane, constraint, roundNumber, s.spawnRules)
			if !valid {
				log.Printf("[WaveSpawnSystem] Zombie spawn rejected: type=%s, lane=%d, wave=%d, reason=%s",
					zombieType, lane, waveIndex+1, reason)
				return 0 // 跳过生成
			}

			// 如果是红眼巨人，增加计数
			if zombieType == "gargantuar_redeye" {
				constraint.RedEyeCount++
				log.Printf("[WaveSpawnSystem] Red eye count increased: %d (round %d)",
					constraint.RedEyeCount, roundNumber)
			}
		}
	}

	// 开场预览期间，僵尸随机分布在5行上
	// 激活后，僵尸会移动到随机选择的有效行

	// 预览行：随机选择一行（0-4）用于开场预览展示
	previewRow := rand.Intn(5)

	// 计算预览位置（僵尸初始站位）
	// X坐标：在配置的范围内随机生成，根据预览行的最大X值
	spawnX := s.getZombieSpawnX(previewRow)
	spawnY := s.getZombieSpawnY(previewRow) // 使用随机行的Y坐标

	// 根据类型创建僵尸
	var entityID ecs.EntityID
	var err error

	// 工厂函数不再接受 reanimSystem 参数
	switch zombieType {
	case "basic":
		entityID, err = entities.NewZombieEntity(
			s.entityManager,
			s.resourceManager,
			previewRow,
			spawnX,
		)
	case "conehead":
		entityID, err = entities.NewConeheadZombieEntity(
			s.entityManager,
			s.resourceManager,
			previewRow,
			spawnX,
		)
	case "buckethead":
		entityID, err = entities.NewBucketheadZombieEntity(
			s.entityManager,
			s.resourceManager,
			previewRow,
			spawnX,
		)
	default:
		log.Printf("[WaveSpawnSystem] ERROR: Unknown zombie type '%s'", zombieType)
		return 0
	}

	if err != nil {
		log.Printf("[WaveSpawnSystem] ERROR: Failed to spawn zombie: %v", err)
		return 0
	}

	// 添加波次状态组件（标记为待命状态）
	ecs.AddComponent(s.entityManager, entityID, &components.ZombieWaveStateComponent{
		WaveIndex:   waveIndex,
		IsActivated: false, // 初始状态：待命
		IndexInWave: indexInWave,
	})

	// 移除或清零速度组件（僵尸初始不移动）
	if vel, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID); ok {
		vel.VX = 0 // 待命状态：不向左移动
		vel.VY = 0 // 待命状态：不垂直移动
		log.Printf("[WaveSpawnSystem] Cleared zombie %d velocity: VX=%.2f, VY=%.2f", entityID, vel.VX, vel.VY)
	}

	// 切换到 idle 动画（预览期间僵尸静止）
	// 激活时会切换回 walk 动画
	if behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID); ok {
		behavior.ZombieAnimState = components.ZombieAnimIdle

		// 读取当前动画状态（调试用）
		if reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
			// 使用新的简化结构
			currentFrame := reanimComp.CurrentFrame
			currentAnim := ""
			if len(reanimComp.CurrentAnimations) > 0 {
				currentAnim = reanimComp.CurrentAnimations[0]
			}
			log.Printf("[WaveSpawnSystem] Zombie %d 切换前动画: %s, 帧: %d", entityID, currentAnim, currentFrame)
		}

		// 使用组件通信替代直接调用
		// 僵尸使用配置驱动的动画组合（自动隐藏装备轨道）
		ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
			UnitID:    "zombie",
			ComboName: "idle",
			Processed: false,
		})

		// 验证切换后的状态
		if reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
			// 使用新的简化结构
			currentFrame := reanimComp.CurrentFrame
			currentAnim := ""
			if len(reanimComp.CurrentAnimations) > 0 {
				currentAnim = reanimComp.CurrentAnimations[0]
			}
			log.Printf("[WaveSpawnSystem] Zombie %d 添加空闲动画命令: %s, 帧: %d (预生成阶段使用 idle)", entityID, currentAnim, currentFrame)
		}
	}

	// 注意：不在创建时添加目标行组件
	// 目标行将在激活时（ActivateWave）才随机选择并添加

	log.Printf("[WaveSpawnSystem] Pre-spawned zombie %d: wave=%d, index=%d, previewRow=%d, pos=(%.1f, %.1f)",
		entityID, waveIndex, indexInWave, previewRow, spawnX, spawnY)

	return entityID
}

// spawnZombieWithOffset 生成单个僵尸（带索引偏移避免重叠）
//
// 根据僵尸类型字符串调用对应的工厂函数
//
// 参数：
//
//	zombieType - 僵尸类型字符串："basic", "conehead", "buckethead"
//	lane - 行号（1-5，对应游戏界面的5行）
//	index - 僵尸在同一波中的索引（0, 1, 2...），用于计算额外的X偏移
//
// 返回：
//
//	生成的僵尸实体ID，如果失败返回 0
func (s *WaveSpawnSystem) spawnZombieWithOffset(zombieType string, lane int, index int) ecs.EntityID {
	// 将行号从1-5转换为数组索引0-4
	row := lane - 1
	if row < 0 || row > 4 {
		log.Printf("[WaveSpawnSystem] ERROR: Invalid lane %d (must be 1-5)", lane)
		return 0
	}

	// 计算生成位置
	// X坐标：在配置的范围内随机生成，根据行号的最大X值
	spawnX := s.getZombieSpawnX(row)
	spawnY := s.getZombieSpawnY(row)

	// 查找目标有效行（如果当前行无效）
	targetLane := s.findNearestEnabledLane(lane)
	targetRow := targetLane - 1

	// 根据僵尸类型调用对应的工厂函数
	var entityID ecs.EntityID
	var err error

	// 工厂函数不再接受 reanimSystem 参数
	switch zombieType {
	case "basic":
		entityID, err = entities.NewZombieEntity(
			s.entityManager,
			s.resourceManager,
			row,
			spawnX,
		)
	case "conehead":
		entityID, err = entities.NewConeheadZombieEntity(
			s.entityManager,
			s.resourceManager,
			row,
			spawnX,
		)
	case "buckethead":
		entityID, err = entities.NewBucketheadZombieEntity(
			s.entityManager,
			s.resourceManager,
			row,
			spawnX,
		)
	default:
		log.Printf("[WaveSpawnSystem] ERROR: Unknown zombie type '%s'", zombieType)
		return 0
	}

	// 检查是否创建成功
	if err != nil {
		log.Printf("[WaveSpawnSystem] ERROR: Failed to spawn zombie type '%s': %v", zombieType, err)
		return 0
	}

	// 如果生成行不是有效行，添加目标行组件
	if row != targetRow {
		s.addTargetLaneComponent(entityID, targetRow, spawnY)
		log.Printf("[WaveSpawnSystem] Zombie %d spawned at lane %d (disabled), will move to lane %d before attacking",
			entityID, lane, targetLane)
	}

	// 输出生成位置信息（用于调试重叠问题）
	log.Printf("[WaveSpawnSystem] Zombie %d spawned at position: X=%.1f (base+index*120+random), Y=%.1f, lane=%d, index=%d",
		entityID, spawnX, spawnY, lane, index)

	return entityID
}

// addTargetLaneComponent 添加目标行组件
//
// 参数：
//
//	entityID - 僵尸实体ID
//	targetRow - 目标行索引（0-4）
//	currentY - 当前Y坐标
func (s *WaveSpawnSystem) addTargetLaneComponent(entityID ecs.EntityID, targetRow int, currentY float64) {
	// 从关卡配置中读取行转换模式
	transitionMode := s.getLaneTransitionMode()

	// 添加目标行组件
	ecs.AddComponent(s.entityManager, entityID, &components.ZombieTargetLaneComponent{
		TargetRow:            targetRow,
		HasReachedTargetLane: false,
		TransitionMode:       transitionMode, // Story 8.7 新增字段
	})

	log.Printf("[WaveSpawnSystem] Added TargetLaneComponent to zombie %d: targetRow=%d, mode=%d",
		entityID, targetRow, transitionMode)

	// VY速度计算逻辑已移至 ZombieLaneTransitionSystem
	// 根据不同的转换模式（instant/gradual），系统会自动处理Y轴移动
	// 不再需要在这里设置VY速度
}

// getZombieSpawnX 获取僵尸生成X坐标
//
// 在配置的范围内随机生成，根据行号使用不同的最大X值
// 范围：config.ZombieSpawnMinX ~ getZombieSpawnMaxX(row)
//
// 参数：
//
//	row - 行索引（0-4）
//
// 返回：
//
//	随机生成的X坐标
func (s *WaveSpawnSystem) getZombieSpawnX(row int) float64 {
	// 根据行号获取最大X值
	maxX := s.getZombieSpawnMaxX(row)

	// 在配置范围内均匀随机分布
	spawnRange := maxX - config.ZombieSpawnMinX
	return config.ZombieSpawnMinX + rand.Float64()*spawnRange
}

// getZombieSpawnMaxX 根据行号获取僵尸生成的最大X坐标
//
// 第1行（row=0）使用 ZombieSpawnMaxX_Row1
// 第2行（row=1）使用 ZombieSpawnMaxX_Row2
// 其他行使用默认的 ZombieSpawnMaxX
//
// 参数：
//
//	row - 行索引（0-4）
//
// 返回：
//
//	该行的最大X坐标
func (s *WaveSpawnSystem) getZombieSpawnMaxX(row int) float64 {
	switch row {
	case 0: // 第1行
		return config.ZombieSpawnMaxX_Row1
	case 1: // 第2行
		return config.ZombieSpawnMaxX_Row2
	default: // 其他行（第3、4、5行）
		return config.ZombieSpawnMaxX
	}
}

// getZombieSpawnY 获取僵尸生成Y坐标
//
// 参数：
//
//	row - 目标行索引（0-4）
//
// 返回：
//
//	僵尸生成Y坐标（行中心 + 垂直偏移修正值）
func (s *WaveSpawnSystem) getZombieSpawnY(row int) float64 {
	// 计算行中心Y坐标
	rowCenterY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2.0

	// 应用僵尸垂直偏移修正值
	return rowCenterY + config.ZombieVerticalOffset
}

// findNearestEnabledLane 查找最近的有效行
//
// 参数：
//
//	targetLane - 目标行（1-5，1-based）
//
// 返回：
//
//	最近的有效行（1-5），如果没有有效行则返回 targetLane
func (s *WaveSpawnSystem) findNearestEnabledLane(targetLane int) int {
	// 如果没有关卡配置或无限制，返回原行
	if s.levelConfig == nil || len(s.levelConfig.EnabledLanes) == 0 {
		return targetLane
	}

	// 如果目标行本身就是有效行，直接返回
	for _, enabledLane := range s.levelConfig.EnabledLanes {
		if enabledLane == targetLane {
			return targetLane
		}
	}

	// 查找最近的有效行
	nearestLane := s.levelConfig.EnabledLanes[0]
	minDistance := abs(targetLane - nearestLane)

	for _, enabledLane := range s.levelConfig.EnabledLanes {
		distance := abs(targetLane - enabledLane)
		if distance < minDistance {
			nearestLane = enabledLane
			minDistance = distance
		}
	}

	return nearestLane
}

// abs 返回整数的绝对值
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// randomEnabledLane 从 enabledLanes 中随机选择一个有效行（返回0-4行索引）
//
// 返回：
//
//	随机选择的行索引（0-4），如果没有限制则从所有行中随机选择
func (s *WaveSpawnSystem) randomEnabledLane() int {
	// 如果没有关卡配置或无行限制，从所有行中随机选择
	if s.levelConfig == nil || len(s.levelConfig.EnabledLanes) == 0 {
		return rand.Intn(5) // 0-4
	}

	// 从 EnabledLanes 中随机选择一个（注意：EnabledLanes 是 1-based）
	randomIndex := rand.Intn(len(s.levelConfig.EnabledLanes))
	selectedLane := s.levelConfig.EnabledLanes[randomIndex] // 1-5
	return selectedLane - 1                                 // 转换为 0-4
}

// validateLaneConfig 验证行是否在关卡配置的 EnabledLanes 中 (Story 8.1)
//
// 参数：
//
//	lane - 行号（1-5，1-based）
//
// 返回：
//
//	true 表示行已启用或无限制，false 表示行被禁用
func (s *WaveSpawnSystem) validateLaneConfig(lane int) bool {
	// 如果没有关卡配置，默认允许所有行
	if s.levelConfig == nil {
		return true
	}

	// 如果 EnabledLanes 为空，默认允许所有行
	if len(s.levelConfig.EnabledLanes) == 0 {
		return true
	}

	// 检查 lane 是否在 EnabledLanes 列表中
	for _, enabledLane := range s.levelConfig.EnabledLanes {
		if enabledLane == lane {
			return true
		}
	}

	return false
}

// getLaneTransitionMode 从关卡配置中获取行转换模式
//
// 读取关卡配置的 laneTransitionMode 字段，
// 并将字符串解析为 LaneTransitionMode 枚举值
//
// 返回：
//
//	LaneTransitionMode - 行转换模式（渐变或瞬间）
//
// 规则：
//   - 如果关卡配置了 laneTransitionMode="gradual"，返回渐变模式
//   - 如果关卡配置了 laneTransitionMode="instant"，返回瞬间模式
//   - 默认返回瞬间模式（向后兼容，不影响现有关卡）
func (s *WaveSpawnSystem) getLaneTransitionMode() components.LaneTransitionMode {
	// 如果没有关卡配置，使用默认瞬间模式
	if s.levelConfig == nil {
		return components.TransitionModeInstant
	}

	// 从配置字符串解析为枚举值
	switch s.levelConfig.LaneTransitionMode {
	case "gradual":
		log.Printf("[WaveSpawnSystem] Lane transition mode: gradual (3-second smooth animation)")
		return components.TransitionModeGradual

	case "instant":
		log.Printf("[WaveSpawnSystem] Lane transition mode: instant (no animation)")
		return components.TransitionModeInstant

	default:
		// 默认瞬间模式（空字符串或未配置）
		log.Printf("[WaveSpawnSystem] Lane transition mode: instant (default)")
		return components.TransitionModeInstant
	}
}
