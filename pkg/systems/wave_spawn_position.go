package systems

import (
	"log"
	"math/rand"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
)

// ============================================================================
// 僵尸生成位置和行相关计算
// ============================================================================

// getZombieSpawnX 获取僵尸生成X坐标
//
// Story 17.9: 优先使用物理配置的出生点范围，向后兼容旧逻辑
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
//	随机生成的X坐标（世界坐标）
func (s *WaveSpawnSystem) getZombieSpawnX(row int) float64 {
	// Story 17.9: 如果有物理配置，使用新坐标系
	if s.zombiePhysics != nil {
		// 使用默认的普通波配置（向后兼容调用）
		minGridX, maxGridX := s.zombiePhysics.GetSpawnXRange("basic", false, false)
		spawnRange := maxGridX - minGridX
		gridX := minGridX + rand.Float64()*spawnRange
		return config.GridToWorldX(gridX)
	}

	// 向后兼容：使用旧的硬编码逻辑
	maxX := s.getZombieSpawnMaxX(row)
	spawnRange := maxX - config.ZombieSpawnMinX
	return config.ZombieSpawnMinX + rand.Float64()*spawnRange
}

// getZombieSpawnXForWave 获取指定波次和僵尸类型的出生点X坐标
//
// Story 17.9: 根据波次类型和僵尸类型返回精确的出生点坐标
//
// 参数：
//
//	zombieType - 僵尸类型字符串（如 "basic", "gargantuar"）
//	isFlagWave - 是否为旗帜波
//	isFlagZombie - 是否为旗帜僵尸
//
// 返回：
//
//	随机生成的X坐标（世界坐标）
func (s *WaveSpawnSystem) getZombieSpawnXForWave(zombieType string, isFlagWave bool, isFlagZombie bool) float64 {
	// Story 17.9: 如果有物理配置，使用精确坐标
	if s.zombiePhysics != nil {
		minGridX, maxGridX := s.zombiePhysics.GetSpawnXRange(zombieType, isFlagWave, isFlagZombie)
		spawnRange := maxGridX - minGridX
		var gridX float64
		if spawnRange > 0 {
			gridX = minGridX + rand.Float64()*spawnRange
		} else {
			gridX = minGridX // 固定位置（如旗帜僵尸）
		}
		return config.GridToWorldX(gridX)
	}

	// 向后兼容：使用旧的硬编码逻辑
	return s.getZombieSpawnX(0)
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
