package systems

import (
	"fmt"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
)

// SpawnConstraintSystem 僵尸生成限制检查系统
// 该系统提供独立的纯函数用于验证僵尸生成是否符合规则
// 遵循 ECS 零耦合原则，不依赖其他系统
type SpawnConstraintSystem struct {
	spawnRules *config.SpawnRulesConfig
}

// NewSpawnConstraintSystem 创建新的生成限制检查系统
func NewSpawnConstraintSystem(spawnRules *config.SpawnRulesConfig) *SpawnConstraintSystem {
	return &SpawnConstraintSystem{
		spawnRules: spawnRules,
	}
}

// CheckZombieTypeAllowed 检查僵尸类型是否在允许列表中
// 独立纯函数，无副作用
func CheckZombieTypeAllowed(zombieType string, allowedTypes []string) bool {
	if len(allowedTypes) == 0 {
		// 如果允许列表为空，则允许所有类型
		return true
	}

	for _, allowed := range allowedTypes {
		if allowed == zombieType {
			return true
		}
	}
	return false
}

// CheckTierRestriction 检查僵尸阶数是否符合波次限制
// 独立纯函数，无副作用
// 参数：
//   - zombieType: 僵尸类型（如 "basic", "gargantuar"）
//   - currentWave: 当前波次编号（从 1 开始）
//   - roundNumber: 当前轮数（影响四阶僵尸最早波次）
//   - spawnRules: 生成规则配置
func CheckTierRestriction(
	zombieType string,
	currentWave int,
	roundNumber int,
	spawnRules *config.SpawnRulesConfig,
) (bool, error) {
	// 获取僵尸阶数
	tier, ok := spawnRules.ZombieTiers[zombieType]
	if !ok {
		return false, fmt.Errorf("unknown zombie type: %s", zombieType)
	}

	// 获取该阶数的最早波次限制
	minWave, ok := spawnRules.TierWaveRestrictions[tier]
	if !ok {
		return false, fmt.Errorf("no wave restriction defined for tier %d", tier)
	}

	// 四阶僵尸：根据轮数调整最早波次
	// 公式: MinWave = max(15 - RoundNumber, 1)
	if tier == 4 {
		adjustedMinWave := 15 - roundNumber
		if adjustedMinWave < 1 {
			adjustedMinWave = 1
		}
		minWave = adjustedMinWave
	}

	// 检查当前波次是否满足限制
	if currentWave < minWave {
		return false, fmt.Errorf(
			"zombie %s (tier %d) cannot spawn before wave %d (current: %d, round: %d)",
			zombieType, tier, minWave, currentWave, roundNumber,
		)
	}

	return true, nil
}

// CheckRedEyeLimit 检查红眼数量是否超限
// 独立纯函数，无副作用
// 参数：
//   - zombieType: 僵尸类型
//   - redEyeCount: 已生成的红眼数量
//   - roundNumber: 当前轮数
//   - spawnRules: 生成规则配置
func CheckRedEyeLimit(
	zombieType string,
	redEyeCount int,
	roundNumber int,
	spawnRules *config.SpawnRulesConfig,
) (bool, error) {
	// 只检查红眼巨人
	if zombieType != "gargantuar_redeye" {
		return true, nil
	}

	// 计算红眼上限
	capacity := CalculateRedEyeCapacity(roundNumber, spawnRules)

	// 检查是否超限
	if redEyeCount >= capacity {
		return false, fmt.Errorf(
			"red eye limit reached: %d/%d (round: %d)",
			redEyeCount, capacity, roundNumber,
		)
	}

	return true, nil
}

// CalculateRedEyeCapacity 根据轮数计算红眼上限
// 独立纯函数，无副作用
// 公式:
//   - RoundNumber < StartRound: 0
//   - RoundNumber >= StartRound: (RoundNumber - StartRound + 1) * CapacityPerRound
func CalculateRedEyeCapacity(roundNumber int, spawnRules *config.SpawnRulesConfig) int {
	if roundNumber < spawnRules.RedEyeRules.StartRound {
		return 0
	}
	return (roundNumber - spawnRules.RedEyeRules.StartRound + 1) * spawnRules.RedEyeRules.CapacityPerRound
}

// CheckSceneTypeRestriction 检查场景类型限制
// 独立纯函数，无副作用
// 参数：
//   - zombieType: 僵尸类型
//   - sceneType: 场景类型（day/night/pool/fog/roof）
//   - lane: 生成行号（从 1 开始）
//   - spawnRules: 生成规则配置
func CheckSceneTypeRestriction(
	zombieType string,
	sceneType string,
	lane int,
	spawnRules *config.SpawnRulesConfig,
) (bool, error) {
	restrictions := spawnRules.SceneTypeRestrictions

	// 1. 检查水路僵尸限制
	isWaterZombie := false
	for _, waterType := range restrictions.WaterZombies {
		if waterType == zombieType {
			isWaterZombie = true
			break
		}
	}

	// 获取当前场景的水路行配置
	waterLanes, hasWaterLanes := restrictions.WaterLaneConfig[sceneType]

	if isWaterZombie {
		// 水路僵尸只能在水路行生成
		if !hasWaterLanes {
			return false, fmt.Errorf(
				"water zombie %s cannot spawn in scene %s (no water lanes)",
				zombieType, sceneType,
			)
		}

		isWaterLane := false
		for _, waterLane := range waterLanes {
			if waterLane == lane {
				isWaterLane = true
				break
			}
		}

		if !isWaterLane {
			return false, fmt.Errorf(
				"water zombie %s can only spawn in water lanes (lane %d is not water)",
				zombieType, lane,
			)
		}
	} else {
		// 非水路僵尸不能在水路行生成
		if hasWaterLanes {
			for _, waterLane := range waterLanes {
				if waterLane == lane {
					return false, fmt.Errorf(
						"non-water zombie %s cannot spawn in water lane %d",
						zombieType, lane,
					)
				}
			}
		}
	}

	// 2. 检查舞王限制
	if zombieType == "dancing" {
		// 检查舞王禁止的场景
		for _, prohibitedScene := range restrictions.DancingRestrictions.ProhibitedScenes {
			if prohibitedScene == sceneType {
				return false, fmt.Errorf(
					"dancing zombie is prohibited in scene %s",
					sceneType,
				)
			}
		}

		// TODO: 舞王需要上下相邻行都是草地的检查
		// 这个检查需要知道相邻行的状态，可能需要在调用时传入更多上下文
		// 当前实现暂时跳过这个检查，后续可以扩展
	}

	return true, nil
}

// ValidateZombieSpawn 综合验证僵尸是否可生成
// 独立纯函数，集成所有验证规则
// 参数：
//   - zombieType: 僵尸类型
//   - lane: 生成行号（用于场景限制检查）
//   - constraint: 生成限制组件（包含当前状态）
//   - roundNumber: 当前轮数
//   - spawnRules: 生成规则配置
// 返回：
//   - bool: 是否通过验证
//   - string: 失败原因（通过时为空字符串）
func ValidateZombieSpawn(
	zombieType string,
	lane int,
	constraint *components.SpawnConstraintComponent,
	roundNumber int,
	spawnRules *config.SpawnRulesConfig,
) (bool, string) {
	// 1. 检查僵尸类型是否在允许列表中
	if !CheckZombieTypeAllowed(zombieType, constraint.AllowedZombieTypes) {
		return false, fmt.Sprintf("zombie type %s not allowed in this level", zombieType)
	}

	// 2. 检查阶数限制
	ok, err := CheckTierRestriction(zombieType, constraint.CurrentWaveNum, roundNumber, spawnRules)
	if !ok {
		return false, err.Error()
	}

	// 3. 检查红眼数量上限
	ok, err = CheckRedEyeLimit(zombieType, constraint.RedEyeCount, roundNumber, spawnRules)
	if !ok {
		return false, err.Error()
	}

	// 4. 检查场景类型限制
	ok, err = CheckSceneTypeRestriction(zombieType, constraint.SceneType, lane, spawnRules)
	if !ok {
		return false, err.Error()
	}

	return true, ""
}
