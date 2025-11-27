package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SpawnRulesConfig 僵尸生成规则配置
type SpawnRulesConfig struct {
	ZombieTiers           map[string]int      `yaml:"zombieTiers"`           // 僵尸类型 -> 阶数
	TierWaveRestrictions  map[int]int         `yaml:"tierWaveRestrictions"`  // 阶数 -> 最早波次
	RedEyeRules           RedEyeRulesConfig   `yaml:"redEyeRules"`           // 红眼规则
	SceneTypeRestrictions SceneRestrictions   `yaml:"sceneTypeRestrictions"` // 场景限制
}

// RedEyeRulesConfig 红眼巨人生成规则
type RedEyeRulesConfig struct {
	StartRound       int `yaml:"startRound"`       // 开始允许红眼的轮数
	CapacityPerRound int `yaml:"capacityPerRound"` // 每轮增加的红眼上限
}

// SceneRestrictions 场景类型限制规则
type SceneRestrictions struct {
	WaterZombies        []string              `yaml:"waterZombies"`        // 水路专属僵尸列表
	DancingRestrictions DancingRestrictions   `yaml:"dancingRestrictions"` // 舞王限制
	WaterLaneConfig     map[string][]int      `yaml:"waterLaneConfig"`     // 场景 -> 水路行号列表
}

// DancingRestrictions 舞王僵尸限制规则
type DancingRestrictions struct {
	ProhibitedScenes       []string `yaml:"prohibitedScenes"`       // 禁止舞王的场景列表
	RequiresAdjacentLanes  bool     `yaml:"requiresAdjacentLanes"`  // 是否需要上下相邻行都是草地
}

// LoadSpawnRules 从 YAML 文件加载僵尸生成规则配置
func LoadSpawnRules(filePath string) (*SpawnRulesConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read spawn rules file: %w", err)
	}

	var config SpawnRulesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse spawn rules YAML: %w", err)
	}

	if err := validateSpawnRules(&config); err != nil {
		return nil, fmt.Errorf("invalid spawn rules config: %w", err)
	}

	return &config, nil
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

	return nil
}
