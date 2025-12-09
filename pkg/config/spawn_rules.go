package config

import (
	"fmt"

	"github.com/gonewx/pvz/pkg/embedded"
	"gopkg.in/yaml.v3"
)

// SpawnRulesConfig 僵尸生成规则配置
type SpawnRulesConfig struct {
	ZombieTiers           map[string]int           `yaml:"zombieTiers"`           // 僵尸类型 -> 阶数
	TierWaveRestrictions  map[int]int              `yaml:"tierWaveRestrictions"`  // 阶数 -> 最早波次
	RedEyeRules           RedEyeRulesConfig        `yaml:"redEyeRules"`           // 红眼规则
	SceneTypeRestrictions SceneRestrictions        `yaml:"sceneTypeRestrictions"` // 场景限制
	HealthAcceleration    HealthAccelerationConfig `yaml:"healthAcceleration"`    // Story 17.8: 血量加速配置
}

// RedEyeRulesConfig 红眼巨人生成规则
type RedEyeRulesConfig struct {
	StartRound       int `yaml:"startRound"`       // 开始允许红眼的轮数
	CapacityPerRound int `yaml:"capacityPerRound"` // 每轮增加的红眼上限
}

// SceneRestrictions 场景类型限制规则
type SceneRestrictions struct {
	WaterZombies        []string            `yaml:"waterZombies"`        // 水路专属僵尸列表
	DancingRestrictions DancingRestrictions `yaml:"dancingRestrictions"` // 舞王限制
	WaterLaneConfig     map[string][]int    `yaml:"waterLaneConfig"`     // 场景 -> 水路行号列表
}

// DancingRestrictions 舞王僵尸限制规则
type DancingRestrictions struct {
	ProhibitedScenes      []string `yaml:"prohibitedScenes"`      // 禁止舞王的场景列表
	RequiresAdjacentLanes bool     `yaml:"requiresAdjacentLanes"` // 是否需要上下相邻行都是草地
}

// HealthAccelerationConfig 血量触发加速刷新配置
// Story 17.8: 配置血量加速刷新的参数
type HealthAccelerationConfig struct {
	MinTriggerTimeCs  int     `yaml:"minTriggerTimeCs"`  // 最小刷出时间（厘秒），默认 401
	TargetCountdownCs int     `yaml:"targetCountdownCs"` // 触发后倒计时（厘秒），默认 200
	ThresholdMin      float64 `yaml:"thresholdMin"`      // 血量阈值下限，默认 0.50
	ThresholdMax      float64 `yaml:"thresholdMax"`      // 血量阈值上限，默认 0.65
}

// LoadSpawnRules 从 YAML 文件加载僵尸生成规则配置
func LoadSpawnRules(filePath string) (*SpawnRulesConfig, error) {
	data, err := embedded.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read spawn rules file: %w", err)
	}

	var config SpawnRulesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse spawn rules YAML: %w", err)
	}

	// Story 17.8: 应用血量加速默认值
	applyHealthAccelerationDefaults(&config)

	if err := validateSpawnRules(&config); err != nil {
		return nil, fmt.Errorf("invalid spawn rules config: %w", err)
	}

	return &config, nil
}

// applyHealthAccelerationDefaults 应用血量加速配置默认值
// Story 17.8: 为未配置的字段设置默认值
func applyHealthAccelerationDefaults(config *SpawnRulesConfig) {
	if config.HealthAcceleration.MinTriggerTimeCs == 0 {
		config.HealthAcceleration.MinTriggerTimeCs = 401
	}
	if config.HealthAcceleration.TargetCountdownCs == 0 {
		config.HealthAcceleration.TargetCountdownCs = 200
	}
	if config.HealthAcceleration.ThresholdMin == 0 {
		config.HealthAcceleration.ThresholdMin = 0.50
	}
	if config.HealthAcceleration.ThresholdMax == 0 {
		config.HealthAcceleration.ThresholdMax = 0.65
	}
}

// validateSpawnRules 验证配置的有效性
func validateSpawnRules(config *SpawnRulesConfig) error {
	// 验证僵尸阶数映射表
	if len(config.ZombieTiers) == 0 {
		return fmt.Errorf("zombieTiers cannot be empty")
	}

	for zombieType, tier := range config.ZombieTiers {
		if zombieType == "" {
			return fmt.Errorf("zombie type cannot be empty")
		}
		if tier < 1 || tier > 4 {
			return fmt.Errorf("zombie tier must be between 1 and 4, got %d for %s", tier, zombieType)
		}
	}

	// 验证阶数波次限制
	if len(config.TierWaveRestrictions) == 0 {
		return fmt.Errorf("tierWaveRestrictions cannot be empty")
	}

	for tier, minWave := range config.TierWaveRestrictions {
		if tier < 1 || tier > 4 {
			return fmt.Errorf("tier in tierWaveRestrictions must be between 1 and 4, got %d", tier)
		}
		if minWave < 1 {
			return fmt.Errorf("minimum wave for tier %d must be >= 1, got %d", tier, minWave)
		}
	}

	// 验证红眼规则
	if config.RedEyeRules.StartRound < 0 {
		return fmt.Errorf("redEyeRules.startRound must be >= 0, got %d", config.RedEyeRules.StartRound)
	}
	if config.RedEyeRules.CapacityPerRound < 0 {
		return fmt.Errorf("redEyeRules.capacityPerRound must be >= 0, got %d", config.RedEyeRules.CapacityPerRound)
	}

	// 验证场景限制（可选字段，无需强制验证）

	// Story 17.8: 验证血量加速配置
	if config.HealthAcceleration.MinTriggerTimeCs < 0 {
		return fmt.Errorf("healthAcceleration.minTriggerTimeCs must be >= 0, got %d", config.HealthAcceleration.MinTriggerTimeCs)
	}
	if config.HealthAcceleration.TargetCountdownCs < 0 {
		return fmt.Errorf("healthAcceleration.targetCountdownCs must be >= 0, got %d", config.HealthAcceleration.TargetCountdownCs)
	}
	if config.HealthAcceleration.ThresholdMin < 0 || config.HealthAcceleration.ThresholdMin > 1 {
		return fmt.Errorf("healthAcceleration.thresholdMin must be between 0 and 1, got %.2f", config.HealthAcceleration.ThresholdMin)
	}
	if config.HealthAcceleration.ThresholdMax < 0 || config.HealthAcceleration.ThresholdMax > 1 {
		return fmt.Errorf("healthAcceleration.thresholdMax must be between 0 and 1, got %.2f", config.HealthAcceleration.ThresholdMax)
	}
	if config.HealthAcceleration.ThresholdMin > config.HealthAcceleration.ThresholdMax {
		return fmt.Errorf("healthAcceleration.thresholdMin (%.2f) must be <= thresholdMax (%.2f)",
			config.HealthAcceleration.ThresholdMin, config.HealthAcceleration.ThresholdMax)
	}

	return nil
}
