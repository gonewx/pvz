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

// TestLoadLevelConfig_WithNewFields 测试解析包含新字段的配置文件 (Story 8.1)
func TestLoadLevelConfig_WithNewFields(t *testing.T) {
	// 创建临时测试配置文件
	yamlContent := `id: "test-1"
name: "Test Level"
description: "Test level with new fields"
openingType: "tutorial"
enabledLanes: [1, 2, 3]
availablePlants: ["peashooter", "sunflower"]
skipOpening: true
specialRules: "bowling"
waves:
  - time: 10
    zombies:
      - type: basic
        lane: 2
        count: 1
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-level.yaml")
	if err := os.WriteFile(testFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 加载配置
	config, err := LoadLevelConfig(testFile)
	if err != nil {
		t.Fatalf("LoadLevelConfig failed: %v", err)
	}

	// 验证基础字段
	if config.ID != "test-1" {
		t.Errorf("Expected ID 'test-1', got %q", config.ID)
	}
	if config.Name != "Test Level" {
		t.Errorf("Expected Name 'Test Level', got %q", config.Name)
	}

	// 验证新字段
	if config.OpeningType != "tutorial" {
		t.Errorf("Expected OpeningType 'tutorial', got %q", config.OpeningType)
	}

	expectedLanes := []int{1, 2, 3}
	if len(config.EnabledLanes) != len(expectedLanes) {
		t.Errorf("Expected EnabledLanes length %d, got %d", len(expectedLanes), len(config.EnabledLanes))
	} else {
		for i, lane := range expectedLanes {
			if config.EnabledLanes[i] != lane {
				t.Errorf("EnabledLanes[%d]: expected %d, got %d", i, lane, config.EnabledLanes[i])
			}
		}
	}

	expectedPlants := []string{"peashooter", "sunflower"}
	if len(config.AvailablePlants) != len(expectedPlants) {
		t.Errorf("Expected AvailablePlants length %d, got %d", len(expectedPlants), len(config.AvailablePlants))
	} else {
		for i, plant := range expectedPlants {
			if config.AvailablePlants[i] != plant {
				t.Errorf("AvailablePlants[%d]: expected %q, got %q", i, plant, config.AvailablePlants[i])
			}
		}
	}

	if !config.SkipOpening {
		t.Errorf("Expected SkipOpening true, got false")
	}

	if config.SpecialRules != "bowling" {
		t.Errorf("Expected SpecialRules 'bowling', got %q", config.SpecialRules)
	}
}

// TestLoadLevelConfig_Defaults 测试默认值处理（新字段缺失时）(Story 8.1)
func TestLoadLevelConfig_Defaults(t *testing.T) {
	// 创建不包含新字段的旧版配置文件
	yamlContent := `id: "old-1"
name: "Old Level"
waves:
  - time: 10
    zombies:
      - type: basic
        lane: 3
        count: 1
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "old-level.yaml")
	if err := os.WriteFile(testFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 加载配置
	config, err := LoadLevelConfig(testFile)
	if err != nil {
		t.Fatalf("LoadLevelConfig failed: %v", err)
	}

	// 验证默认值
	if config.OpeningType != "standard" {
		t.Errorf("Expected default OpeningType 'standard', got %q", config.OpeningType)
	}

	expectedDefaultLanes := []int{1, 2, 3, 4, 5}
	if len(config.EnabledLanes) != len(expectedDefaultLanes) {
		t.Errorf("Expected default EnabledLanes length %d, got %d", len(expectedDefaultLanes), len(config.EnabledLanes))
	} else {
		for i, lane := range expectedDefaultLanes {
			if config.EnabledLanes[i] != lane {
				t.Errorf("EnabledLanes[%d]: expected %d, got %d", i, lane, config.EnabledLanes[i])
			}
		}
	}

	if len(config.AvailablePlants) != 0 {
		t.Errorf("Expected empty AvailablePlants, got %v", config.AvailablePlants)
	}

	if config.SkipOpening {
		t.Errorf("Expected default SkipOpening false, got true")
	}

	if config.SpecialRules != "" {
		t.Errorf("Expected empty SpecialRules, got %q", config.SpecialRules)
	}

	if len(config.TutorialSteps) != 0 {
		t.Errorf("Expected empty TutorialSteps, got %v", config.TutorialSteps)
	}
}

// TestValidateLevelConfig_InvalidEnabledLanes 测试 EnabledLanes 验证 (Story 8.1)
func TestValidateLevelConfig_InvalidEnabledLanes(t *testing.T) {
	tests := []struct {
		name         string
		enabledLanes []int
		expectError  bool
	}{
		{"Valid lanes", []int{1, 2, 3}, false},
		{"Valid single lane", []int{3}, false},
		{"Valid all lanes", []int{1, 2, 3, 4, 5}, false},
		{"Invalid lane 0", []int{0, 1, 2}, true},
		{"Invalid lane 6", []int{1, 6}, true},
		{"Invalid negative lane", []int{-1, 2}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LevelConfig{
				ID:           "test",
				Name:         "Test",
				EnabledLanes: tt.enabledLanes,
				Waves: []WaveConfig{
					{
						Time: 10,
						Zombies: []ZombieSpawn{
							{Type: "basic", Lane: 2, Count: 1},
						},
					},
				},
			}

			applyDefaults(config) // 应用默认值
			err := validateLevelConfig(config)
			if tt.expectError && err == nil {
				t.Errorf("Expected validation error for lanes %v, but got none", tt.enabledLanes)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error for lanes %v: %v", tt.enabledLanes, err)
			}
		})
	}
}

// TestValidateLevelConfig_InvalidOpeningType 测试 OpeningType 验证 (Story 8.1)
func TestValidateLevelConfig_InvalidOpeningType(t *testing.T) {
	tests := []struct {
		name        string
		openingType string
		expectError bool
	}{
		{"Valid tutorial", "tutorial", false},
		{"Valid standard", "standard", false},
		{"Valid special", "special", false},
		{"Invalid type", "invalid", true},
		{"Empty (will use default)", "", false}, // 空值会被默认值填充
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LevelConfig{
				ID:          "test",
				Name:        "Test",
				OpeningType: tt.openingType,
				Waves: []WaveConfig{
					{
						Time: 10,
						Zombies: []ZombieSpawn{
							{Type: "basic", Lane: 2, Count: 1},
						},
					},
				},
			}

			applyDefaults(config) // 应用默认值
			err := validateLevelConfig(config)
			if tt.expectError && err == nil {
				t.Errorf("Expected validation error for openingType %q, but got none", tt.openingType)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error for openingType %q: %v", tt.openingType, err)
			}
		})
	}
}

// TestValidateLevelConfig_InvalidSpecialRules 测试 SpecialRules 验证 (Story 8.1)
func TestValidateLevelConfig_InvalidSpecialRules(t *testing.T) {
	tests := []struct {
		name         string
		specialRules string
		expectError  bool
	}{
		{"Valid bowling", "bowling", false},
		{"Valid conveyor", "conveyor", false},
		{"Invalid rule", "invalid", true},
		{"Empty (valid)", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LevelConfig{
				ID:           "test",
				Name:         "Test",
				SpecialRules: tt.specialRules,
				Waves: []WaveConfig{
					{
						Time: 10,
						Zombies: []ZombieSpawn{
							{Type: "basic", Lane: 2, Count: 1},
						},
					},
				},
			}

			applyDefaults(config)
			err := validateLevelConfig(config)
			if tt.expectError && err == nil {
				t.Errorf("Expected validation error for specialRules %q, but got none", tt.specialRules)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error for specialRules %q: %v", tt.specialRules, err)
			}
		})
	}
}

// TestLoadLevelConfig_BackwardCompatibility 测试向后兼容性 (Story 8.1)
// 验证旧版配置文件（无新字段）仍能正常加载
func TestLoadLevelConfig_BackwardCompatibility(t *testing.T) {
	// 使用项目中现有的关卡配置文件（如果存在）
	// 这里我们创建一个与旧版完全一致的配置
	yamlContent := `id: "1-1"
name: "前院白天 1-1"
description: "教学关卡"
waves:
  - time: 10
    zombies:
      - type: basic
        lane: 3
        count: 1
  - time: 30
    zombies:
      - type: basic
        lane: 2
        count: 2
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "level-1-1.yaml")
	if err := os.WriteFile(testFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 加载配置（应该成功）
	config, err := LoadLevelConfig(testFile)
	if err != nil {
		t.Fatalf("Failed to load backward-compatible config: %v", err)
	}

	// 验证基础功能仍然正常
	if config.ID != "1-1" {
		t.Errorf("Expected ID '1-1', got %q", config.ID)
	}

	if len(config.Waves) != 2 {
		t.Errorf("Expected 2 waves, got %d", len(config.Waves))
	}

	// 验证默认值已应用
	if config.OpeningType != "standard" {
		t.Errorf("Expected default OpeningType 'standard', got %q", config.OpeningType)
	}

	if len(config.EnabledLanes) != 5 {
		t.Errorf("Expected 5 enabled lanes by default, got %d", len(config.EnabledLanes))
	}
}

// TestTutorialSteps_Parsing 测试 TutorialSteps 解析 (Story 8.1, updated in Story 8.2)
func TestTutorialSteps_Parsing(t *testing.T) {
	yamlContent := `id: "tutorial-1"
name: "Tutorial Level"
tutorialSteps:
  - trigger: "gameStart"
    textKey: "TUTORIAL_WELCOME"
    action: "waitForSunCollect"
  - trigger: "sunCollected"
    textKey: "TUTORIAL_PLANT_PEASHOOTER"
    action: "waitForPlantPlaced"
waves:
  - time: 10
    zombies:
      - type: basic
        lane: 3
        count: 1
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "tutorial.yaml")
	if err := os.WriteFile(testFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config, err := LoadLevelConfig(testFile)
	if err != nil {
		t.Fatalf("LoadLevelConfig failed: %v", err)
	}

	if len(config.TutorialSteps) != 2 {
		t.Errorf("Expected 2 tutorial steps, got %d", len(config.TutorialSteps))
	}

	// 验证第一个步骤
	if config.TutorialSteps[0].Trigger != "gameStart" {
		t.Errorf("Expected first step trigger 'gameStart', got %q", config.TutorialSteps[0].Trigger)
	}
	if config.TutorialSteps[0].TextKey != "TUTORIAL_WELCOME" {
		t.Errorf("Expected first step textKey 'TUTORIAL_WELCOME', got %q", config.TutorialSteps[0].TextKey)
	}
	if config.TutorialSteps[0].Action != "waitForSunCollect" {
		t.Errorf("Expected first step action 'waitForSunCollect', got %q", config.TutorialSteps[0].Action)
	}

	// 验证第二个步骤
	if config.TutorialSteps[1].Trigger != "sunCollected" {
		t.Errorf("Expected second step trigger 'sunCollected', got %q", config.TutorialSteps[1].Trigger)
	}
}
