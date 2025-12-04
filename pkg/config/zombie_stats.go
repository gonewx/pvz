package config

import (
	"fmt"

	"github.com/decker502/pvz/pkg/embedded"
	"gopkg.in/yaml.v3"
)

// ZombieStats 单个僵尸类型的属性配置
type ZombieStats struct {
	Level                int `yaml:"level"`                // 僵尸级别，用于难度引擎计算级别容量
	Weight               int `yaml:"weight"`               // 权重，用于随机选择僵尸类型
	BaseHealth           int `yaml:"baseHealth"`           // 本体血量
	Tier1AccessoryHealth int `yaml:"tier1AccessoryHealth"` // I类饰品血量（如路障、铁桶）
	Tier2AccessoryHealth int `yaml:"tier2AccessoryHealth"` // II类饰品血量（预留）
}

// ZombieStatsConfig 僵尸属性配置文件结构
type ZombieStatsConfig struct {
	Zombies map[string]ZombieStats `yaml:"zombies"` // 僵尸类型到属性的映射
}

// LoadZombieStats 从 YAML 文件加载僵尸属性配置
// 参数：
//
//	filepath - 配置文件路径（相对或绝对路径）
//
// 返回：
//
//	*ZombieStatsConfig - 解析后的配置对象
//	error - 如果文件读取或解析失败，返回错误信息
func LoadZombieStats(filepath string) (*ZombieStatsConfig, error) {
	data, err := embedded.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read zombie stats file %s: %w", filepath, err)
	}

	var config ZombieStatsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse zombie stats YAML from %s: %w", filepath, err)
	}

	if err := validateZombieStats(&config); err != nil {
		return nil, fmt.Errorf("invalid zombie stats in %s: %w", filepath, err)
	}

	return &config, nil
}

// validateZombieStats 验证僵尸属性配置的完整性和合法性
func validateZombieStats(config *ZombieStatsConfig) error {
	if len(config.Zombies) == 0 {
		return fmt.Errorf("at least one zombie type is required")
	}

	for zombieType, stats := range config.Zombies {
		if stats.Level < 1 {
			return fmt.Errorf("zombie %s: level must be at least 1, got %d", zombieType, stats.Level)
		}

		if stats.Weight < 0 {
			return fmt.Errorf("zombie %s: weight cannot be negative, got %d", zombieType, stats.Weight)
		}

		if stats.BaseHealth < 0 {
			return fmt.Errorf("zombie %s: baseHealth cannot be negative, got %d", zombieType, stats.BaseHealth)
		}

		if stats.Tier1AccessoryHealth < 0 {
			return fmt.Errorf("zombie %s: tier1AccessoryHealth cannot be negative, got %d", zombieType, stats.Tier1AccessoryHealth)
		}

		if stats.Tier2AccessoryHealth < 0 {
			return fmt.Errorf("zombie %s: tier2AccessoryHealth cannot be negative, got %d", zombieType, stats.Tier2AccessoryHealth)
		}
	}

	return nil
}

// GetZombieLevel 获取指定僵尸类型的级别
// 如果僵尸类型不存在，返回默认级别 1
func (c *ZombieStatsConfig) GetZombieLevel(zombieType string) int {
	if stats, ok := c.Zombies[zombieType]; ok {
		return stats.Level
	}
	return 1
}

// GetZombieWeight 获取指定僵尸类型的权重
// 如果僵尸类型不存在，返回默认权重 0
func (c *ZombieStatsConfig) GetZombieWeight(zombieType string) int {
	if stats, ok := c.Zombies[zombieType]; ok {
		return stats.Weight
	}
	return 0
}

// GetZombieStats 获取指定僵尸类型的完整属性
// 如果僵尸类型不存在，返回 nil 和 false
func (c *ZombieStatsConfig) GetZombieStats(zombieType string) (*ZombieStats, bool) {
	stats, ok := c.Zombies[zombieType]
	if !ok {
		return nil, false
	}
	return &stats, true
}
