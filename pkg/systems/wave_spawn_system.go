package systems

import (
	"log"

	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
)

// 僵尸生成位置常量
const (
	// ZombieSpawnX 僵尸生成的X坐标（屏幕右侧外）
	ZombieSpawnX = 1200.0
)

// WaveSpawnSystem 波次生成系统
//
// 职责：
//   - 根据 WaveConfig 生成僵尸实体
//   - 支持批量生成
//   - 处理不同僵尸类型的工厂调用
//   - Story 8.1: 验证僵尸生成行是否在 EnabledLanes 中
//
// 架构说明：
//   - 作为 LevelSystem 的依赖，由 LevelSystem 调用
//   - 使用僵尸工厂函数创建实体（entities 包）
//   - 遵循数据驱动原则：根据配置文件生成僵尸
type WaveSpawnSystem struct {
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager
	reanimSystem    *ReanimSystem      // 用于初始化僵尸动画
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
			// 生成单个僵尸
			entityID := s.spawnZombie(zombieSpawn.Type, zombieSpawn.Lane)
			if entityID != 0 {
				totalSpawned++
				log.Printf("[WaveSpawnSystem] Spawned zombie: type=%s, lane=%d, entityID=%d",
					zombieSpawn.Type, zombieSpawn.Lane, entityID)
			}
		}
	}

	return totalSpawned
}

// spawnZombie 生成单个僵尸
//
// 根据僵尸类型字符串调用对应的工厂函数
//
// 参数：
//
//	zombieType - 僵尸类型字符串："basic", "conehead", "buckethead"
//	lane - 行号（1-5，对应游戏界面的5行）
//
// 返回：
//
//	生成的僵尸实体ID，如果失败返回 0
func (s *WaveSpawnSystem) spawnZombie(zombieType string, lane int) ecs.EntityID {
	// Story 8.1: 验证行是否在 EnabledLanes 中
	if !s.validateLaneConfig(lane) {
		log.Printf("[WaveSpawnSystem] WARNING: Lane %d is not enabled in level config, skipping zombie spawn", lane)
		return 0
	}

	// 将行号从1-5转换为数组索引0-4
	row := lane - 1
	if row < 0 || row > 4 {
		log.Printf("[WaveSpawnSystem] ERROR: Invalid lane %d (must be 1-5)", lane)
		return 0
	}

	// 计算生成位置X坐标
	spawnX := s.getZombieSpawnX()

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

	return entityID
}

// getZombieSpawnX 获取僵尸生成X坐标
//
// 返回屏幕右侧外的生成坐标
// 可以在此方法中添加随机偏移，避免僵尸完全重叠
func (s *WaveSpawnSystem) getZombieSpawnX() float64 {
	// 当前版本返回固定坐标
	// 未来可以添加随机偏移：ZombieSpawnX + rand.Float64() * 50
	return ZombieSpawnX
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
