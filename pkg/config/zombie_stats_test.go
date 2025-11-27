package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadZombieStats(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	t.Run("加载有效配置文件", func(t *testing.T) {
		configContent := `
zombies:
  basic:
    level: 1
    weight: 4000
    baseHealth: 270
    tier1AccessoryHealth: 0
    tier2AccessoryHealth: 0
  conehead:
    level: 2
    weight: 4000
    baseHealth: 270
    tier1AccessoryHealth: 370
    tier2AccessoryHealth: 0
  buckethead:
    level: 4
    weight: 3000
    baseHealth: 270
    tier1AccessoryHealth: 1100
    tier2AccessoryHealth: 0
`
		configPath := filepath.Join(tempDir, "valid_config.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		config, err := LoadZombieStats(configPath)
		if err != nil {
			t.Fatalf("LoadZombieStats failed: %v", err)
		}

		if len(config.Zombies) != 3 {
			t.Errorf("Expected 3 zombie types, got %d", len(config.Zombies))
		}

		// 验证 basic 僵尸
		basic, ok := config.Zombies["basic"]
		if !ok {
			t.Fatal("basic zombie not found")
		}
		if basic.Level != 1 {
			t.Errorf("basic level: expected 1, got %d", basic.Level)
		}
		if basic.Weight != 4000 {
			t.Errorf("basic weight: expected 4000, got %d", basic.Weight)
		}
		if basic.BaseHealth != 270 {
			t.Errorf("basic baseHealth: expected 270, got %d", basic.BaseHealth)
		}

		// 验证 conehead 僵尸
		conehead, ok := config.Zombies["conehead"]
		if !ok {
			t.Fatal("conehead zombie not found")
		}
		if conehead.Level != 2 {
			t.Errorf("conehead level: expected 2, got %d", conehead.Level)
		}
		if conehead.Tier1AccessoryHealth != 370 {
			t.Errorf("conehead tier1AccessoryHealth: expected 370, got %d", conehead.Tier1AccessoryHealth)
		}

		// 验证 buckethead 僵尸
		buckethead, ok := config.Zombies["buckethead"]
		if !ok {
			t.Fatal("buckethead zombie not found")
		}
		if buckethead.Level != 4 {
			t.Errorf("buckethead level: expected 4, got %d", buckethead.Level)
		}
		if buckethead.Tier1AccessoryHealth != 1100 {
			t.Errorf("buckethead tier1AccessoryHealth: expected 1100, got %d", buckethead.Tier1AccessoryHealth)
		}
	})

	t.Run("文件不存在", func(t *testing.T) {
		_, err := LoadZombieStats(filepath.Join(tempDir, "nonexistent.yaml"))
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})

	t.Run("无效 YAML 格式", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "invalid_yaml.yaml")
		if err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		_, err := LoadZombieStats(configPath)
		if err == nil {
			t.Error("Expected error for invalid YAML")
		}
	})

	t.Run("空僵尸列表", func(t *testing.T) {
		configContent := `zombies: {}`
		configPath := filepath.Join(tempDir, "empty_zombies.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		_, err := LoadZombieStats(configPath)
		if err == nil {
			t.Error("Expected error for empty zombies list")
		}
	})

	t.Run("无效级别（小于1）", func(t *testing.T) {
		configContent := `
zombies:
  basic:
    level: 0
    weight: 4000
    baseHealth: 270
`
		configPath := filepath.Join(tempDir, "invalid_level.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		_, err := LoadZombieStats(configPath)
		if err == nil {
			t.Error("Expected error for invalid level")
		}
	})

	t.Run("负数权重", func(t *testing.T) {
		configContent := `
zombies:
  basic:
    level: 1
    weight: -100
    baseHealth: 270
`
		configPath := filepath.Join(tempDir, "negative_weight.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		_, err := LoadZombieStats(configPath)
		if err == nil {
			t.Error("Expected error for negative weight")
		}
	})

	t.Run("负数血量", func(t *testing.T) {
		configContent := `
zombies:
  basic:
    level: 1
    weight: 4000
    baseHealth: -10
`
		configPath := filepath.Join(tempDir, "negative_health.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		_, err := LoadZombieStats(configPath)
		if err == nil {
			t.Error("Expected error for negative health")
		}
	})
}

func TestZombieStatsConfig_GetZombieLevel(t *testing.T) {
	config := &ZombieStatsConfig{
		Zombies: map[string]ZombieStats{
			"basic":      {Level: 1, Weight: 4000, BaseHealth: 270},
			"conehead":   {Level: 2, Weight: 4000, BaseHealth: 270},
			"buckethead": {Level: 4, Weight: 3000, BaseHealth: 270},
		},
	}

	tests := []struct {
		name       string
		zombieType string
		expected   int
	}{
		{"普通僵尸", "basic", 1},
		{"路障僵尸", "conehead", 2},
		{"铁桶僵尸", "buckethead", 4},
		{"不存在的僵尸（返回默认值1）", "unknown", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := config.GetZombieLevel(tt.zombieType)
			if level != tt.expected {
				t.Errorf("GetZombieLevel(%s): expected %d, got %d", tt.zombieType, tt.expected, level)
			}
		})
	}
}

func TestZombieStatsConfig_GetZombieWeight(t *testing.T) {
	config := &ZombieStatsConfig{
		Zombies: map[string]ZombieStats{
			"basic":      {Level: 1, Weight: 4000, BaseHealth: 270},
			"buckethead": {Level: 4, Weight: 3000, BaseHealth: 270},
		},
	}

	tests := []struct {
		name       string
		zombieType string
		expected   int
	}{
		{"普通僵尸", "basic", 4000},
		{"铁桶僵尸", "buckethead", 3000},
		{"不存在的僵尸（返回默认值0）", "unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weight := config.GetZombieWeight(tt.zombieType)
			if weight != tt.expected {
				t.Errorf("GetZombieWeight(%s): expected %d, got %d", tt.zombieType, tt.expected, weight)
			}
		})
	}
}

func TestZombieStatsConfig_GetZombieStats(t *testing.T) {
	config := &ZombieStatsConfig{
		Zombies: map[string]ZombieStats{
			"basic": {Level: 1, Weight: 4000, BaseHealth: 270, Tier1AccessoryHealth: 0},
		},
	}

	t.Run("获取存在的僵尸属性", func(t *testing.T) {
		stats, ok := config.GetZombieStats("basic")
		if !ok {
			t.Fatal("Expected to find basic zombie stats")
		}
		if stats.Level != 1 {
			t.Errorf("Expected level 1, got %d", stats.Level)
		}
		if stats.Weight != 4000 {
			t.Errorf("Expected weight 4000, got %d", stats.Weight)
		}
	})

	t.Run("获取不存在的僵尸属性", func(t *testing.T) {
		stats, ok := config.GetZombieStats("unknown")
		if ok {
			t.Error("Expected not to find unknown zombie stats")
		}
		if stats != nil {
			t.Error("Expected nil stats for unknown zombie")
		}
	})
}

func TestLoadZombieStats_Integration(t *testing.T) {
	// 测试加载实际配置文件（从项目根目录运行测试）
	config, err := LoadZombieStats("../../data/zombie_stats.yaml")
	if err != nil {
		t.Fatalf("Failed to load actual zombie stats: %v", err)
	}

	// 验证必要的僵尸类型存在
	requiredTypes := []string{"basic", "conehead", "buckethead", "gargantuar"}
	for _, zombieType := range requiredTypes {
		if _, ok := config.Zombies[zombieType]; !ok {
			t.Errorf("Required zombie type %s not found in config", zombieType)
		}
	}

	// 验证级别符合预期
	expectedLevels := map[string]int{
		"basic":      1,
		"conehead":   2,
		"buckethead": 4,
		"gargantuar": 10,
	}

	for zombieType, expectedLevel := range expectedLevels {
		level := config.GetZombieLevel(zombieType)
		if level != expectedLevel {
			t.Errorf("%s level: expected %d, got %d", zombieType, expectedLevel, level)
		}
	}
}
