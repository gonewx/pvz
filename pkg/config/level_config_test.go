package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadLevelConfig 测试关卡配置文件加载
func TestLoadLevelConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		// 创建临时测试文件
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test-level.yaml")

		validYAML := `id: "1-1"
name: "Test Level"
description: "A test level"
waves:
  - time: 10
    zombies:
      - type: basic
        lane: 3
        count: 1
  - time: 30
    zombies:
      - type: basic
        lane: 1
        count: 2
      - type: conehead
        lane: 5
        count: 1
`
		if err := os.WriteFile(testFile, []byte(validYAML), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// 加载配置
		config, err := LoadLevelConfig(testFile)
		if err != nil {
			t.Fatalf("LoadLevelConfig() failed: %v", err)
		}

		// 验证基本字段
		if config.ID != "1-1" {
			t.Errorf("Expected ID '1-1', got '%s'", config.ID)
		}
		if config.Name != "Test Level" {
			t.Errorf("Expected Name 'Test Level', got '%s'", config.Name)
		}
		if config.Description != "A test level" {
			t.Errorf("Expected Description 'A test level', got '%s'", config.Description)
		}

		// 验证波次数量
		if len(config.Waves) != 2 {
			t.Fatalf("Expected 2 waves, got %d", len(config.Waves))
		}

		// 验证第一波
		wave1 := config.Waves[0]
		if wave1.Time != 10 {
			t.Errorf("Wave 1: Expected time 10, got %f", wave1.Time)
		}
		if len(wave1.Zombies) != 1 {
			t.Fatalf("Wave 1: Expected 1 zombie spawn, got %d", len(wave1.Zombies))
		}
		if wave1.Zombies[0].Type != "basic" {
			t.Errorf("Wave 1 Zombie 0: Expected type 'basic', got '%s'", wave1.Zombies[0].Type)
		}
		if wave1.Zombies[0].Lane != 3 {
			t.Errorf("Wave 1 Zombie 0: Expected lane 3, got %d", wave1.Zombies[0].Lane)
		}
		if wave1.Zombies[0].Count != 1 {
			t.Errorf("Wave 1 Zombie 0: Expected count 1, got %d", wave1.Zombies[0].Count)
		}

		// 验证第二波
		wave2 := config.Waves[1]
		if wave2.Time != 30 {
			t.Errorf("Wave 2: Expected time 30, got %f", wave2.Time)
		}
		if len(wave2.Zombies) != 2 {
			t.Fatalf("Wave 2: Expected 2 zombie spawns, got %d", len(wave2.Zombies))
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadLevelConfig("nonexistent-file.yaml")
		if err == nil {
			t.Error("Expected error for nonexistent file, got nil")
		}
	})

	t.Run("invalid YAML", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "invalid.yaml")

		invalidYAML := `id: "1-1"
name: [this is not a string]
invalid yaml structure
`
		if err := os.WriteFile(testFile, []byte(invalidYAML), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		_, err := LoadLevelConfig(testFile)
		if err == nil {
			t.Error("Expected error for invalid YAML, got nil")
		}
	})
}

// TestLevelConfigValidation 测试关卡配置验证逻辑
func TestLevelConfigValidation(t *testing.T) {
	t.Run("missing ID", func(t *testing.T) {
		config := &LevelConfig{
			Name: "Test Level",
			Waves: []WaveConfig{
				{Time: 10, Zombies: []ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
			},
		}
		err := validateLevelConfig(config)
		if err == nil {
			t.Error("Expected error for missing ID, got nil")
		}
	})

	t.Run("missing name", func(t *testing.T) {
		config := &LevelConfig{
			ID: "1-1",
			Waves: []WaveConfig{
				{Time: 10, Zombies: []ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
			},
		}
		err := validateLevelConfig(config)
		if err == nil {
			t.Error("Expected error for missing name, got nil")
		}
	})

	t.Run("no waves", func(t *testing.T) {
		config := &LevelConfig{
			ID:    "1-1",
			Name:  "Test Level",
			Waves: []WaveConfig{},
		}
		err := validateLevelConfig(config)
		if err == nil {
			t.Error("Expected error for no waves, got nil")
		}
	})

	t.Run("negative wave time", func(t *testing.T) {
		config := &LevelConfig{
			ID:   "1-1",
			Name: "Test Level",
			Waves: []WaveConfig{
				{Time: -5, Zombies: []ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
			},
		}
		err := validateLevelConfig(config)
		if err == nil {
			t.Error("Expected error for negative wave time, got nil")
		}
	})

	t.Run("wave with no zombies", func(t *testing.T) {
		config := &LevelConfig{
			ID:   "1-1",
			Name: "Test Level",
			Waves: []WaveConfig{
				{Time: 10, Zombies: []ZombieSpawn{}},
			},
		}
		err := validateLevelConfig(config)
		if err == nil {
			t.Error("Expected error for wave with no zombies, got nil")
		}
	})

	t.Run("zombie missing type", func(t *testing.T) {
		config := &LevelConfig{
			ID:   "1-1",
			Name: "Test Level",
			Waves: []WaveConfig{
				{Time: 10, Zombies: []ZombieSpawn{{Type: "", Lane: 1, Count: 1}}},
			},
		}
		err := validateLevelConfig(config)
		if err == nil {
			t.Error("Expected error for zombie missing type, got nil")
		}
	})

	t.Run("invalid lane number", func(t *testing.T) {
		testCases := []struct {
			lane int
		}{
			{lane: 0},
			{lane: 6},
			{lane: -1},
		}

		for _, tc := range testCases {
			config := &LevelConfig{
				ID:   "1-1",
				Name: "Test Level",
				Waves: []WaveConfig{
					{Time: 10, Zombies: []ZombieSpawn{{Type: "basic", Lane: tc.lane, Count: 1}}},
				},
			}
			err := validateLevelConfig(config)
			if err == nil {
				t.Errorf("Expected error for lane %d, got nil", tc.lane)
			}
		}
	})

	t.Run("invalid zombie count", func(t *testing.T) {
		config := &LevelConfig{
			ID:   "1-1",
			Name: "Test Level",
			Waves: []WaveConfig{
				{Time: 10, Zombies: []ZombieSpawn{{Type: "basic", Lane: 1, Count: 0}}},
			},
		}
		err := validateLevelConfig(config)
		if err == nil {
			t.Error("Expected error for zombie count 0, got nil")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		config := &LevelConfig{
			ID:   "1-1",
			Name: "Test Level",
			Waves: []WaveConfig{
				{Time: 10, Zombies: []ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
				{Time: 30, Zombies: []ZombieSpawn{{Type: "conehead", Lane: 3, Count: 2}}},
			},
		}
		err := validateLevelConfig(config)
		if err != nil {
			t.Errorf("Expected no error for valid config, got: %v", err)
		}
	})
}
