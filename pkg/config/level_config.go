package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LevelConfig 关卡配置数据结构
// 定义了关卡的基本信息和僵尸波次配置
type LevelConfig struct {
	ID          string       `yaml:"id"`          // 关卡ID，如 "1-1"
	Name        string       `yaml:"name"`        // 关卡名称，如 "前院白天 1-1"
	Description string       `yaml:"description"` // 关卡描述（可选）
	Waves       []WaveConfig `yaml:"waves"`       // 僵尸波次配置列表
}

// WaveConfig 单个僵尸波次配置
// 定义了僵尸波次的触发时间和生成的僵尸列表
type WaveConfig struct {
	Time    float64       `yaml:"time"`    // 波次触发时间（秒，从关卡开始计时）
	Zombies []ZombieSpawn `yaml:"zombies"` // 本波次要生成的僵尸列表
}

// ZombieSpawn 单个僵尸生成配置
// 定义了僵尸的类型、出现行数和生成数量
type ZombieSpawn struct {
	Type  string `yaml:"type"`  // 僵尸类型："basic", "conehead", "buckethead"
	Lane  int    `yaml:"lane"`  // 僵尸出现的行（1-5，对应游戏界面的5行）
	Count int    `yaml:"count"` // 生成数量
}

// LoadLevelConfig 从YAML文件加载关卡配置
// 参数：
//
//	filepath - 关卡配置文件的路径（相对或绝对路径）
//
// 返回：
//
//	*LevelConfig - 解析后的关卡配置对象
//	error - 如果文件读取或解析失败，返回错误信息
func LoadLevelConfig(filepath string) (*LevelConfig, error) {
	// 读取文件内容
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read level config file %s: %w", filepath, err)
	}

	// 解析YAML数据
	var levelConfig LevelConfig
	if err := yaml.Unmarshal(data, &levelConfig); err != nil {
		return nil, fmt.Errorf("failed to parse level config YAML from %s: %w", filepath, err)
	}

	// 验证必填字段
	if err := validateLevelConfig(&levelConfig); err != nil {
		return nil, fmt.Errorf("invalid level config in %s: %w", filepath, err)
	}

	return &levelConfig, nil
}

// validateLevelConfig 验证关卡配置的完整性和合法性
func validateLevelConfig(config *LevelConfig) error {
	// 验证关卡ID
	if config.ID == "" {
		return fmt.Errorf("level ID is required")
	}

	// 验证关卡名称
	if config.Name == "" {
		return fmt.Errorf("level name is required")
	}

	// 验证波次配置
	if len(config.Waves) == 0 {
		return fmt.Errorf("at least one wave is required")
	}

	// 验证每个波次的配置
	for i, wave := range config.Waves {
		if wave.Time < 0 {
			return fmt.Errorf("wave %d: time cannot be negative", i)
		}

		if len(wave.Zombies) == 0 {
			return fmt.Errorf("wave %d: at least one zombie spawn is required", i)
		}

		// 验证每个僵尸生成配置
		for j, zombie := range wave.Zombies {
			if zombie.Type == "" {
				return fmt.Errorf("wave %d, zombie %d: type is required", i, j)
			}

			if zombie.Lane < 1 || zombie.Lane > 5 {
				return fmt.Errorf("wave %d, zombie %d: lane must be between 1 and 5, got %d", i, j, zombie.Lane)
			}

			if zombie.Count < 1 {
				return fmt.Errorf("wave %d, zombie %d: count must be at least 1, got %d", i, j, zombie.Count)
			}
		}
	}

	return nil
}
