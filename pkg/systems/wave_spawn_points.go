package systems

import (
	"log"
	"math/rand"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
)

// ============================================================================
// 点数分配和生成限制
// ============================================================================

// zombiePointCost 定义每种僵尸的点数成本
// Story 8.9: 用于 ExtraPoints 波次类型的动态僵尸分配
var zombiePointCost = map[string]int{
	"basic":       1, // 普通僵尸
	"zombie":      1, // 普通僵尸（别名）
	"conehead":    2, // 路障僵尸
	"buckethead":  3, // 铁桶僵尸
	"polevaulter": 2, // 撑杆僵尸
}

// createConstraintEntity 创建生成限制组件实体
// Story 17.3: 用于存储关卡级别的生成限制状态
func (s *WaveSpawnSystem) createConstraintEntity() ecs.EntityID {
	// 如果没有关卡配置，返回 0（不创建限制实体）
	if s.levelConfig == nil {
		log.Printf("[WaveSpawnSystem] No level config, skipping constraint entity creation")
		return 0
	}

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
	// 如果没有关卡配置或波次配置，返回空列表
	if s.levelConfig == nil || s.levelConfig.Waves == nil {
		return []string{}
	}

	typeSet := make(map[string]bool)

	// Story 8.9: 优先从 ZombiePool 中提取（ExtraPoints 波次使用）
	for _, zombieType := range s.levelConfig.ZombiePool {
		typeSet[zombieType] = true
	}

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

// allocateZombiesFromPoints 根据点数从僵尸池中动态分配僵尸
//
// Story 8.9: 用于 ExtraPoints 波次类型的动态僵尸分配
// 根据给定的点数从关卡配置的 zombiePool 中随机选择僵尸，直到点数用尽
//
// 参数：
//
//	points - 可用点数
//
// 返回：
//
//	选择的僵尸类型列表
//
// 僵尸点数成本：
//   - basic/zombie: 1 点
//   - conehead: 2 点
//   - buckethead: 3 点
//   - polevaulter: 2 点
func (s *WaveSpawnSystem) allocateZombiesFromPoints(points int, waveIndex int) []string {
	if points <= 0 {
		return nil
	}

	// 获取关卡配置的僵尸池
	zombiePool := s.getZombiePool()
	if len(zombiePool) == 0 {
		log.Printf("[WaveSpawnSystem] WARNING: No zombie pool configured, defaulting to basic zombie")
		// 默认使用普通僵尸
		zombiePool = []string{"basic"}
	}

	// 当前波次编号（1-based）
	currentWave := waveIndex + 1

	var result []string
	remainingPoints := points

	// 循环分配僵尸直到点数用尽
	for remainingPoints > 0 {
		// 过滤出当前点数可以负担且当前波次允许的僵尸类型
		affordableTypes := s.filterAffordableAndAllowedZombies(zombiePool, remainingPoints, currentWave)
		if len(affordableTypes) == 0 {
			// 没有可负担/可用的僵尸类型，退出循环
			log.Printf("[WaveSpawnSystem] No affordable/allowed zombie types for %d remaining points at wave %d", remainingPoints, currentWave)
			break
		}

		// 从可负担的类型中随机选择一个
		selectedType := affordableTypes[rand.Intn(len(affordableTypes))]
		cost := s.getZombieCost(selectedType)

		result = append(result, selectedType)
		remainingPoints -= cost

		log.Printf("[WaveSpawnSystem] ExtraPoints allocated: type=%s, cost=%d, remaining=%d",
			selectedType, cost, remainingPoints)
	}

	log.Printf("[WaveSpawnSystem] ExtraPoints allocation complete: %d points -> %d zombies", points, len(result))
	return result
}

// getZombiePool 获取关卡配置的僵尸池
//
// 返回：
//
//	僵尸类型列表
func (s *WaveSpawnSystem) getZombiePool() []string {
	if s.levelConfig == nil {
		return nil
	}

	// 优先使用关卡配置的 zombiePool
	if len(s.levelConfig.ZombiePool) > 0 {
		return s.levelConfig.ZombiePool
	}

	// 向后兼容：从波次配置中提取僵尸类型
	return s.extractAllowedZombieTypes()
}

// filterAffordableZombies 过滤出当前点数可以负担的僵尸类型
//
// 参数：
//
//	zombiePool - 僵尸池
//	availablePoints - 可用点数
//
// 返回：
//
//	可负担的僵尸类型列表
func (s *WaveSpawnSystem) filterAffordableZombies(zombiePool []string, availablePoints int) []string {
	var affordable []string
	for _, zombieType := range zombiePool {
		cost := s.getZombieCost(zombieType)
		if cost <= availablePoints {
			affordable = append(affordable, zombieType)
		}
	}
	return affordable
}

// filterAffordableAndAllowedZombies 过滤出当前点数可以负担的僵尸类型
//
// Story 8.9: 用于 ExtraPoints 波次类型的动态僵尸分配
//
// 参数：
//
//	zombiePool - 僵尸池
//	availablePoints - 可用点数
//	currentWave - 当前波次编号（1-based，保留用于后续扩展）
//
// 返回：
//
//	可负担的僵尸类型列表
func (s *WaveSpawnSystem) filterAffordableAndAllowedZombies(zombiePool []string, availablePoints int, currentWave int) []string {
	var result []string
	for _, zombieType := range zombiePool {
		// 检查点数是否足够
		cost := s.getZombieCost(zombieType)
		if cost > availablePoints {
			continue
		}

		result = append(result, zombieType)
	}
	return result
}

// getZombieCost 获取僵尸类型的点数成本
//
// 参数：
//
//	zombieType - 僵尸类型
//
// 返回：
//
//	点数成本（未知类型默认为 1）
func (s *WaveSpawnSystem) getZombieCost(zombieType string) int {
	if cost, ok := zombiePointCost[zombieType]; ok {
		return cost
	}
	// 未知类型默认为 1 点
	return 1
}
