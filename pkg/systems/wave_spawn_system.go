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
//
// 架构说明：
//   - 作为 LevelSystem 的依赖，由 LevelSystem 调用
//   - 使用僵尸工厂函数创建实体（entities 包）
//   - 遵循数据驱动原则：根据配置文件生成僵尸
//
// 预生成机制：
//   1. PreSpawnAllWaves() 在关卡开始时调用，预生成所有僵尸
//   2. ActivateWave(waveIndex) 在波次时间到达时调用，激活指定波次的僵尸
type WaveSpawnSystem struct {
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager
	reanimSystem    *ReanimSystem       // 用于初始化僵尸动画
	levelConfig     *config.LevelConfig // Story 8.1: 关卡配置（用于验证行数限制）
}

// NewWaveSpawnSystem 创建波次生成系统
//
// 参数：
//
//	em - 实体管理器
//	rm - 资源管理器
//	rs - Reanim系统（用于初始化僵尸动画）
//	lc - 关卡配置（Story 8.1: 用于验证行数限制）
func NewWaveSpawnSystem(em *ecs.EntityManager, rm *game.ResourceManager, rs *ReanimSystem, lc *config.LevelConfig) *WaveSpawnSystem {
	return &WaveSpawnSystem{
		entityManager:   em,
		resourceManager: rm,
		reanimSystem:    rs,
		levelConfig:     lc,
	}
}

// SpawnWave 生成一波僵尸
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

	// 遍历波次中的所有僵尸生成配置
	for _, zombieSpawn := range waveConfig.Zombies {
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

	return totalSpawned
}

// PreSpawnAllWaves 预生成所有波次的僵尸
//
// 在关卡开始时调用，一次性生成所有僵尸并放置在屏幕右侧站位
// 僵尸初始状态为"待命"（不移动），等待 ActivateWave() 激活
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
		// 遍历本波的所有僵尸配置
		for _, zombieSpawn := range waveConfig.Zombies {
			// 生成多个僵尸
			for i := 0; i < zombieSpawn.Count; i++ {
				entityID := s.spawnZombieForWave(zombieSpawn.Type, zombieSpawn.Lane, waveIndex, i)
				if entityID != 0 {
					totalSpawned++
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
					vel.VX = -23.0 // 僵尸标准移动速度
					log.Printf("[WaveSpawnSystem] Activated zombie %d (wave %d, index %d), VX=%.1f",
						entityID, waveIndex, waveState.IndexInWave, vel.VX)
				}
			}

			// Story 8.3: 切换到 walk 动画（僵尸开始移动）
			if behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID); ok {
				if behavior.ZombieAnimState == components.ZombieAnimIdle {
					behavior.ZombieAnimState = components.ZombieAnimWalking
					if err := s.reanimSystem.PlayAnimation(entityID, "anim_walk"); err != nil {
						log.Printf("[WaveSpawnSystem] Warning: Failed to play walk animation for zombie %d: %v", entityID, err)
					} else {
						log.Printf("[WaveSpawnSystem] Zombie %d switched to walk animation (activated)", entityID)
					}
				}
			}

			activated++
		}
	}

	log.Printf("[WaveSpawnSystem] Activated wave %d: %d zombies", waveIndex, activated)
	return activated
}

// spawnZombieForWave 为指定波次生成僵尸（预生成模式）
//
// 生成的僵尸初始状态为"待命"（不移动），需要调用 ActivateWave() 激活
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
	// Story 8.3: 开场预览期间，僵尸随机分布在5行上
	// 激活后，僵尸会移动到随机选择的有效行

	// 预览行：随机选择一行（0-4）用于开场预览展示
	previewRow := rand.Intn(5)

	// 计算预览位置（僵尸初始站位）
	// X坐标：在配置的范围内随机生成，不依赖波次索引
	spawnX := s.getZombieSpawnX()
	spawnY := s.getZombieSpawnY(previewRow) // 使用随机行的Y坐标

	// 根据类型创建僵尸
	var entityID ecs.EntityID
	var err error

	switch zombieType {
	case "basic":
		entityID, err = entities.NewZombieEntity(
			s.entityManager,
			s.resourceManager,
			s.reanimSystem,
			previewRow,
			spawnX,
		)
	case "conehead":
		entityID, err = entities.NewConeheadZombieEntity(
			s.entityManager,
			s.resourceManager,
			s.reanimSystem,
			previewRow,
			spawnX,
		)
	case "buckethead":
		entityID, err = entities.NewBucketheadZombieEntity(
			s.entityManager,
			s.resourceManager,
			s.reanimSystem,
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

	// Story 8.3: 切换到 idle 动画（预览期间僵尸静止）
	// 激活时会切换回 walk 动画
	if behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID); ok {
		behavior.ZombieAnimState = components.ZombieAnimIdle

		// 读取当前动画状态（调试用）
		if reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
			log.Printf("[WaveSpawnSystem] Zombie %d 切换前动画: %s, 帧: %d", entityID, reanimComp.CurrentAnim, reanimComp.CurrentFrame)
		}

		if err := s.reanimSystem.PlayAnimation(entityID, "anim_idle"); err != nil {
			log.Printf("[WaveSpawnSystem] Warning: Failed to play idle animation for zombie %d: %v", entityID, err)
		} else {
			// 验证切换后的状态
			if reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
				log.Printf("[WaveSpawnSystem] Zombie %d 切换后动画: %s, 帧: %d (preview mode)", entityID, reanimComp.CurrentAnim, reanimComp.CurrentFrame)
			}
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
	// X坐标：在配置的范围内随机生成，不依赖索引
	spawnX := s.getZombieSpawnX()
	spawnY := s.getZombieSpawnY(row)

	// 查找目标有效行（如果当前行无效）
	targetLane := s.findNearestEnabledLane(lane)
	targetRow := targetLane - 1

	// 根据僵尸类型调用对应的工厂函数
	var entityID ecs.EntityID
	var err error

	switch zombieType {
	case "basic":
		entityID, err = entities.NewZombieEntity(
			s.entityManager,
			s.resourceManager,
			s.reanimSystem,
			row,
			spawnX,
		)
	case "conehead":
		entityID, err = entities.NewConeheadZombieEntity(
			s.entityManager,
			s.resourceManager,
			s.reanimSystem,
			row,
			spawnX,
		)
	case "buckethead":
		entityID, err = entities.NewBucketheadZombieEntity(
			s.entityManager,
			s.resourceManager,
			s.reanimSystem,
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
	// 添加目标行组件
	ecs.AddComponent(s.entityManager, entityID, &components.ZombieTargetLaneComponent{
		TargetRow:            targetRow,
		HasReachedTargetLane: false,
	})

	// 添加或更新速度组件，添加Y轴速度以移动到目标行
	// 必须与僵尸工厂函数的Y坐标计算公式保持一致
	targetY := config.GridWorldStartY + float64(targetRow)*config.CellHeight + config.CellHeight/2.0 + config.ZombieVerticalOffset
	deltaY := targetY - currentY

	// 计算Y轴速度（每秒移动距离）
	// 假设僵尸需要在3秒内到达目标行
	vySpeed := deltaY / 3.0

	// 获取或创建速度组件
	if vel, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID); ok {
		// 已有速度组件，添加Y轴速度
		vel.VY = vySpeed
	} else {
		// 创建新的速度组件
		ecs.AddComponent(s.entityManager, entityID, &components.VelocityComponent{
			VX: 0,  // X轴速度暂时为0，到达目标行后才开始向左移动
			VY: vySpeed,
		})
	}

	log.Printf("[WaveSpawnSystem] Added target lane component: targetRow=%d, deltaY=%.2f, vySpeed=%.2f",
		targetRow, deltaY, vySpeed)
}

// getZombieSpawnX 获取僵尸生成X坐标
//
// 在配置的范围内随机生成，不依赖波次索引
// 范围：config.ZombieSpawnMinX ~ config.ZombieSpawnMaxX
func (s *WaveSpawnSystem) getZombieSpawnX() float64 {
	// 在配置范围内均匀随机分布
	spawnRange := config.ZombieSpawnMaxX - config.ZombieSpawnMinX
	return config.ZombieSpawnMinX + rand.Float64()*spawnRange
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
	return selectedLane - 1                                  // 转换为 0-4
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
