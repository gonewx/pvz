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
  - zombies:
      - type: basic
        lanes: [3]
        count: 1
        spawnInterval: 2.0
  - zombies:
      - type: basic
        lanes: [1]
        count: 2
        spawnInterval: 2.0
      - type: conehead
        lanes: [5]
        count: 1
        spawnInterval: 3.0
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
		if len(wave1.Zombies) != 1 {
			t.Fatalf("Wave 1: Expected 1 zombie spawn, got %d", len(wave1.Zombies))
		}
		if wave1.Zombies[0].Type != "basic" {
			t.Errorf("Wave 1 Zombie 0: Expected type 'basic', got '%s'", wave1.Zombies[0].Type)
		}
		if len(wave1.Zombies[0].Lanes) != 1 || wave1.Zombies[0].Lanes[0] != 3 {
			t.Errorf("Wave 1 Zombie 0: Expected lanes [3], got %v", wave1.Zombies[0].Lanes)
		}
		if wave1.Zombies[0].Count != 1 {
			t.Errorf("Wave 1 Zombie 0: Expected count 1, got %d", wave1.Zombies[0].Count)
		}

		// 验证第二波
		wave2 := config.Waves[1]
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
				{OldZombies: []ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
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
				{OldZombies: []ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
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

	t.Run("wave with no zombies", func(t *testing.T) {
		config := &LevelConfig{
			ID:   "1-1",
			Name: "Test Level",
			Waves: []WaveConfig{
				{OldZombies: []ZombieSpawn{}},
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
				{OldZombies: []ZombieSpawn{{Type: "", Lane: 1, Count: 1}}},
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
					{OldZombies: []ZombieSpawn{{Type: "basic", Lane: tc.lane, Count: 1}}},
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
				{OldZombies: []ZombieSpawn{{Type: "basic", Lane: 1, Count: 0}}},
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
				{OldZombies: []ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
				{OldZombies: []ZombieSpawn{{Type: "conehead", Lane: 3, Count: 2}}},
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
  - zombies:
      - type: basic
        lanes: [2]
        count: 1
        spawnInterval: 2.0
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
  - zombies:
      - type: basic
        lanes: [3]
        count: 1
        spawnInterval: 2.0
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
						OldZombies: []ZombieSpawn{
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
						OldZombies: []ZombieSpawn{
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
						OldZombies: []ZombieSpawn{
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
  - zombies:
      - type: basic
        lanes: [3]
        count: 1
        spawnInterval: 2.0
  - zombies:
      - type: basic
        lanes: [2]
        count: 2
        spawnInterval: 2.0
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
  - zombies:
      - type: basic
        lanes: [3]
        count: 1
        spawnInterval: 2.0
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

// ============================================================
// Story 17.2: 关卡脚本格式升级测试
// ============================================================

// TestLoadLevelConfig_NewFormatFields 测试新格式字段解析 (Story 17.2)
func TestLoadLevelConfig_NewFormatFields(t *testing.T) {
	// 创建包含所有新字段的配置文件
	yamlContent := `id: "test-17.2"
name: "Test Level New Format"
description: "Testing new format fields"
flags: 2
sceneType: "pool"
rowMax: 6
waves:
  - waveNum: 1
    type: "Fixed"
    extraPoints: 0
    laneRestriction: [1, 2, 3]
    zombies:
      - type: basic
        lanes: [1, 2, 3]
        count: 2
        spawnInterval: 1.5
  - waveNum: 2
    type: "ExtraPoints"
    extraPoints: 100
    zombies:
      - type: conehead
        lanes: [4, 5, 6]
        count: 3
  - waveNum: 3
    type: "Final"
    isFlag: true
    flagIndex: 1
    zombies:
      - type: buckethead
        lanes: [1, 2, 3, 4, 5, 6]
        count: 5
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-new-format.yaml")
	if err := os.WriteFile(testFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config, err := LoadLevelConfig(testFile)
	if err != nil {
		t.Fatalf("LoadLevelConfig failed: %v", err)
	}

	// 验证顶层新字段
	if config.Flags != 2 {
		t.Errorf("Expected Flags 2, got %d", config.Flags)
	}
	if config.SceneType != "pool" {
		t.Errorf("Expected SceneType 'pool', got %q", config.SceneType)
	}
	if config.RowMax != 6 {
		t.Errorf("Expected RowMax 6, got %d", config.RowMax)
	}

	// 验证波次数量
	if len(config.Waves) != 3 {
		t.Fatalf("Expected 3 waves, got %d", len(config.Waves))
	}

	// 验证第一波 (Fixed)
	wave1 := config.Waves[0]
	if wave1.WaveNum != 1 {
		t.Errorf("Wave 1: Expected WaveNum 1, got %d", wave1.WaveNum)
	}
	if wave1.Type != "Fixed" {
		t.Errorf("Wave 1: Expected Type 'Fixed', got %q", wave1.Type)
	}
	if wave1.ExtraPoints != 0 {
		t.Errorf("Wave 1: Expected ExtraPoints 0, got %d", wave1.ExtraPoints)
	}
	expectedLaneRestriction := []int{1, 2, 3}
	if len(wave1.LaneRestriction) != len(expectedLaneRestriction) {
		t.Errorf("Wave 1: Expected LaneRestriction length %d, got %d", len(expectedLaneRestriction), len(wave1.LaneRestriction))
	}

	// 验证第二波 (ExtraPoints)
	wave2 := config.Waves[1]
	if wave2.WaveNum != 2 {
		t.Errorf("Wave 2: Expected WaveNum 2, got %d", wave2.WaveNum)
	}
	if wave2.Type != "ExtraPoints" {
		t.Errorf("Wave 2: Expected Type 'ExtraPoints', got %q", wave2.Type)
	}
	if wave2.ExtraPoints != 100 {
		t.Errorf("Wave 2: Expected ExtraPoints 100, got %d", wave2.ExtraPoints)
	}

	// 验证第三波 (Final)
	wave3 := config.Waves[2]
	if wave3.WaveNum != 3 {
		t.Errorf("Wave 3: Expected WaveNum 3, got %d", wave3.WaveNum)
	}
	if wave3.Type != "Final" {
		t.Errorf("Wave 3: Expected Type 'Final', got %q", wave3.Type)
	}
	if !wave3.IsFlag {
		t.Errorf("Wave 3: Expected IsFlag true, got false")
	}
}

// TestLoadLevelConfig_NewFormatDefaults 测试新字段默认值 (Story 17.2)
func TestLoadLevelConfig_NewFormatDefaults(t *testing.T) {
	// 创建不包含新字段的旧格式配置
	yamlContent := `id: "test-defaults"
name: "Test Defaults"
waves:
  - zombies:
      - type: basic
        lanes: [3]
        count: 1
  - isFlag: true
    flagIndex: 1
    zombies:
      - type: basic
        lanes: [3]
        count: 2
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-defaults.yaml")
	if err := os.WriteFile(testFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config, err := LoadLevelConfig(testFile)
	if err != nil {
		t.Fatalf("LoadLevelConfig failed: %v", err)
	}

	// 验证默认值
	if config.SceneType != "day" {
		t.Errorf("Expected default SceneType 'day', got %q", config.SceneType)
	}
	if config.RowMax != 5 {
		t.Errorf("Expected default RowMax 5, got %d", config.RowMax)
	}
	// Flags 默认从 isFlag 数量推断
	if config.Flags != 1 {
		t.Errorf("Expected Flags 1 (inferred from isFlag), got %d", config.Flags)
	}

	// 验证波次默认值
	if len(config.Waves) != 2 {
		t.Fatalf("Expected 2 waves, got %d", len(config.Waves))
	}

	// 第一波：WaveNum 从索引推断，Type 从 isFlag 推断
	wave1 := config.Waves[0]
	if wave1.WaveNum != 1 {
		t.Errorf("Wave 1: Expected default WaveNum 1, got %d", wave1.WaveNum)
	}
	if wave1.Type != "Fixed" {
		t.Errorf("Wave 1: Expected default Type 'Fixed', got %q", wave1.Type)
	}

	// 第二波：isFlag=true，Type 应为 Final
	wave2 := config.Waves[1]
	if wave2.WaveNum != 2 {
		t.Errorf("Wave 2: Expected default WaveNum 2, got %d", wave2.WaveNum)
	}
	if wave2.Type != "Final" {
		t.Errorf("Wave 2: Expected Type 'Final' (inferred from isFlag), got %q", wave2.Type)
	}
}

// TestValidateLevelConfig_InvalidSceneType 测试无效 SceneType 验证 (Story 17.2)
func TestValidateLevelConfig_InvalidSceneType(t *testing.T) {
	config := &LevelConfig{
		ID:        "test",
		Name:      "Test",
		SceneType: "invalid_scene",
		Waves: []WaveConfig{
			{
				Zombies: []ZombieGroup{
					{Type: "basic", Lanes: []int{3}, Count: 1},
				},
			},
		},
	}

	applyDefaults(config)
	err := validateLevelConfig(config)
	if err == nil {
		t.Error("Expected validation error for invalid SceneType, got nil")
	}
	if err != nil && !containsString(err.Error(), "sceneType") {
		t.Errorf("Expected error message to mention 'sceneType', got: %v", err)
	}
}

// TestValidateLevelConfig_InvalidRowMax 测试无效 RowMax 验证 (Story 17.2)
func TestValidateLevelConfig_InvalidRowMax(t *testing.T) {
	tests := []struct {
		name        string
		rowMax      int
		expectError bool
	}{
		{"Valid 5", 5, false},
		{"Valid 6", 6, false},
		{"Valid 0 (default)", 0, false},
		{"Invalid 4", 4, true},
		{"Invalid 7", 7, true},
		{"Invalid -1", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LevelConfig{
				ID:     "test",
				Name:   "Test",
				RowMax: tt.rowMax,
				Waves: []WaveConfig{
					{
						Zombies: []ZombieGroup{
							{Type: "basic", Lanes: []int{3}, Count: 1},
						},
					},
				},
			}

			applyDefaults(config)
			err := validateLevelConfig(config)
			if tt.expectError && err == nil {
				t.Errorf("Expected validation error for RowMax %d, got nil", tt.rowMax)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error for RowMax %d: %v", tt.rowMax, err)
			}
		})
	}
}

// TestValidateLevelConfig_InvalidWaveType 测试无效波次类型验证 (Story 17.2)
func TestValidateLevelConfig_InvalidWaveType(t *testing.T) {
	tests := []struct {
		name        string
		waveType    string
		expectError bool
	}{
		{"Valid Fixed", "Fixed", false},
		{"Valid ExtraPoints", "ExtraPoints", false},
		{"Valid Final", "Final", false},
		{"Valid empty (default)", "", false},
		{"Invalid type", "Unknown", true},
		{"Invalid lowercase", "fixed", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LevelConfig{
				ID:   "test",
				Name: "Test",
				Waves: []WaveConfig{
					{
						Type: tt.waveType,
						Zombies: []ZombieGroup{
							{Type: "basic", Lanes: []int{3}, Count: 1},
						},
					},
				},
			}

			applyDefaults(config)
			err := validateLevelConfig(config)
			if tt.expectError && err == nil {
				t.Errorf("Expected validation error for wave type %q, got nil", tt.waveType)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error for wave type %q: %v", tt.waveType, err)
			}
		})
	}
}

// TestValidateLevelConfig_ExtraPointsValidation 测试 ExtraPoints 验证 (Story 17.2)
func TestValidateLevelConfig_ExtraPointsValidation(t *testing.T) {
	tests := []struct {
		name        string
		waveType    string
		extraPoints int
		expectError bool
	}{
		{"ExtraPoints type with points", "ExtraPoints", 100, false},
		{"ExtraPoints type with zero", "ExtraPoints", 0, false},
		{"Fixed type with zero", "Fixed", 0, false},
		{"Fixed type with points", "Fixed", 100, true},
		{"Final type with points", "Final", 50, true},
		{"Empty type with points", "", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LevelConfig{
				ID:   "test",
				Name: "Test",
				Waves: []WaveConfig{
					{
						Type:        tt.waveType,
						ExtraPoints: tt.extraPoints,
						Zombies: []ZombieGroup{
							{Type: "basic", Lanes: []int{3}, Count: 1},
						},
					},
				},
			}

			applyDefaults(config)
			err := validateLevelConfig(config)
			if tt.expectError && err == nil {
				t.Errorf("Expected validation error for type=%q extraPoints=%d, got nil", tt.waveType, tt.extraPoints)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error for type=%q extraPoints=%d: %v", tt.waveType, tt.extraPoints, err)
			}
		})
	}
}

// TestValidateLevelConfig_LaneRestriction 测试 LaneRestriction 验证 (Story 17.2)
func TestValidateLevelConfig_LaneRestriction(t *testing.T) {
	tests := []struct {
		name            string
		rowMax          int
		laneRestriction []int
		expectError     bool
	}{
		{"Valid lanes for RowMax 5", 5, []int{1, 3, 5}, false},
		{"Valid lanes for RowMax 6", 6, []int{1, 3, 6}, false},
		{"Invalid lane 6 for RowMax 5", 5, []int{1, 6}, true},
		{"Invalid lane 0", 5, []int{0, 1}, true},
		{"Invalid negative lane", 5, []int{-1, 1}, true},
		{"Empty restriction (valid)", 5, []int{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LevelConfig{
				ID:     "test",
				Name:   "Test",
				RowMax: tt.rowMax,
				Waves: []WaveConfig{
					{
						LaneRestriction: tt.laneRestriction,
						Zombies: []ZombieGroup{
							{Type: "basic", Lanes: []int{1}, Count: 1},
						},
					},
				},
			}

			applyDefaults(config)
			err := validateLevelConfig(config)
			if tt.expectError && err == nil {
				t.Errorf("Expected validation error for laneRestriction %v with RowMax %d, got nil", tt.laneRestriction, tt.rowMax)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error for laneRestriction %v with RowMax %d: %v", tt.laneRestriction, tt.rowMax, err)
			}
		})
	}
}

// TestLoadLevelConfig_V2File 测试加载 v2 格式示例文件 (Story 17.2)
// 现在 level-1-1.yaml 已经迁移到新格式，直接测试该文件
func TestLoadLevelConfig_V2File(t *testing.T) {
	config, err := LoadLevelConfig("../../data/levels/level-1-1.yaml")
	if err != nil {
		t.Fatalf("Failed to load level-1-1.yaml: %v", err)
	}

	// 验证基本字段
	if config.ID != "1-1" {
		t.Errorf("Expected ID '1-1', got %q", config.ID)
	}
	if config.SceneType != "day" {
		t.Errorf("Expected SceneType 'day', got %q", config.SceneType)
	}
	if config.RowMax != 5 {
		t.Errorf("Expected RowMax 5, got %d", config.RowMax)
	}
	if config.Flags != 0 {
		t.Errorf("Expected Flags 0, got %d", config.Flags)
	}

	// 验证波次
	if len(config.Waves) != 4 {
		t.Fatalf("Expected 4 waves, got %d", len(config.Waves))
	}

	// 验证波次类型
	for i, expectedType := range []string{"Fixed", "Fixed", "Fixed", "Final"} {
		if config.Waves[i].Type != expectedType {
			t.Errorf("Wave %d: Expected Type %q, got %q", i+1, expectedType, config.Waves[i].Type)
		}
		if config.Waves[i].WaveNum != i+1 {
			t.Errorf("Wave %d: Expected WaveNum %d, got %d", i+1, i+1, config.Waves[i].WaveNum)
		}
	}
}

// TestLoadLevelConfig_RealFilesBackwardCompat 测试现有关卡文件向后兼容 (Story 17.2)
func TestLoadLevelConfig_RealFilesBackwardCompat(t *testing.T) {
	files := []string{
		"../../data/levels/level-1-1.yaml",
		"../../data/levels/level-1-2.yaml",
		"../../data/levels/level-1-3.yaml",
		"../../data/levels/level-1-4.yaml",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			config, err := LoadLevelConfig(file)
			if err != nil {
				t.Fatalf("Failed to load %s: %v", file, err)
			}

			// 验证默认值已应用
			if config.SceneType != "day" {
				t.Errorf("Expected default SceneType 'day', got %q", config.SceneType)
			}
			if config.RowMax != 5 {
				t.Errorf("Expected default RowMax 5, got %d", config.RowMax)
			}

			// 验证波次有默认的 WaveNum 和 Type
			for i, wave := range config.Waves {
				if wave.WaveNum != i+1 {
					t.Errorf("Wave %d: Expected default WaveNum %d, got %d", i, i+1, wave.WaveNum)
				}
				if wave.Type == "" {
					t.Errorf("Wave %d: Expected non-empty Type (should have default)", i)
				}
			}
		})
	}
}

// TestValidateLevelConfig_NegativeFlags 测试负数 Flags 验证 (Story 17.2)
func TestValidateLevelConfig_NegativeFlags(t *testing.T) {
	config := &LevelConfig{
		ID:    "test",
		Name:  "Test",
		Flags: -1,
		Waves: []WaveConfig{
			{
				Zombies: []ZombieGroup{
					{Type: "basic", Lanes: []int{3}, Count: 1},
				},
			},
		},
	}

	applyDefaults(config)
	err := validateLevelConfig(config)
	if err == nil {
		t.Error("Expected validation error for negative Flags, got nil")
	}
}

// TestValidateLevelConfig_NegativeExtraPoints 测试负数 ExtraPoints 验证 (Story 17.2)
func TestValidateLevelConfig_NegativeExtraPoints(t *testing.T) {
	config := &LevelConfig{
		ID:   "test",
		Name: "Test",
		Waves: []WaveConfig{
			{
				Type:        "ExtraPoints",
				ExtraPoints: -10,
				Zombies: []ZombieGroup{
					{Type: "basic", Lanes: []int{3}, Count: 1},
				},
			},
		},
	}

	applyDefaults(config)
	err := validateLevelConfig(config)
	if err == nil {
		t.Error("Expected validation error for negative ExtraPoints, got nil")
	}
}

// TestValidateLevelConfig_ZombieLanesWithRowMax6 测试 RowMax=6 时的僵尸行验证 (Story 17.2)
func TestValidateLevelConfig_ZombieLanesWithRowMax6(t *testing.T) {
	tests := []struct {
		name        string
		rowMax      int
		lanes       []int
		expectError bool
	}{
		{"Lane 6 with RowMax 6", 6, []int{6}, false},
		{"Lane 6 with RowMax 5", 5, []int{6}, true},
		{"Lanes 1-6 with RowMax 6", 6, []int{1, 2, 3, 4, 5, 6}, false},
		{"Lanes 1-5 with RowMax 5", 5, []int{1, 2, 3, 4, 5}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LevelConfig{
				ID:     "test",
				Name:   "Test",
				RowMax: tt.rowMax,
				Waves: []WaveConfig{
					{
						Zombies: []ZombieGroup{
							{Type: "basic", Lanes: tt.lanes, Count: 1},
						},
					},
				},
			}

			applyDefaults(config)
			err := validateLevelConfig(config)
			if tt.expectError && err == nil {
				t.Errorf("Expected validation error for lanes %v with RowMax %d, got nil", tt.lanes, tt.rowMax)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error for lanes %v with RowMax %d: %v", tt.lanes, tt.rowMax, err)
			}
		})
	}
}

// containsString 辅助函数：检查字符串是否包含子串
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================
// Story 17.2: 额外边缘情况测试 - 提升覆盖率
// ============================================================

// TestValidateLevelConfig_ValidSceneTypes 测试所有有效场景类型 (Story 17.2)
func TestValidateLevelConfig_ValidSceneTypes(t *testing.T) {
	validSceneTypes := []string{"day", "night", "pool", "fog", "roof", "moon"}

	for _, sceneType := range validSceneTypes {
		t.Run("SceneType_"+sceneType, func(t *testing.T) {
			config := &LevelConfig{
				ID:        "test",
				Name:      "Test",
				SceneType: sceneType,
				Waves: []WaveConfig{
					{
						Zombies: []ZombieGroup{
							{Type: "basic", Lanes: []int{3}, Count: 1},
						},
					},
				},
			}

			applyDefaults(config)
			err := validateLevelConfig(config)
			if err != nil {
				t.Errorf("Expected no error for sceneType %q, got: %v", sceneType, err)
			}
		})
	}
}

// TestValidateLevelConfig_WaveNumEdgeCases 测试 WaveNum 边缘情况 (Story 17.2)
func TestValidateLevelConfig_WaveNumEdgeCases(t *testing.T) {
	// WaveNum 为 0 应该被 applyDefaults 修正为索引+1
	config := &LevelConfig{
		ID:   "test",
		Name: "Test",
		Waves: []WaveConfig{
			{
				WaveNum: 0, // 0 will be auto-corrected to 1
				Zombies: []ZombieGroup{
					{Type: "basic", Lanes: []int{3}, Count: 1},
				},
			},
		},
	}

	applyDefaults(config)
	err := validateLevelConfig(config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// 验证 WaveNum 被正确设置为 1
	if config.Waves[0].WaveNum != 1 {
		t.Errorf("Expected WaveNum 1 after applyDefaults, got %d", config.Waves[0].WaveNum)
	}
}

// TestValidateLevelConfig_ZeroFlags 测试 Flags=0 的有效情况 (Story 17.2)
func TestValidateLevelConfig_ZeroFlags(t *testing.T) {
	config := &LevelConfig{
		ID:    "test",
		Name:  "Test",
		Flags: 0, // 0 是有效的 (无旗帜关卡)
		Waves: []WaveConfig{
			{
				Zombies: []ZombieGroup{
					{Type: "basic", Lanes: []int{3}, Count: 1},
				},
			},
		},
	}

	applyDefaults(config)
	err := validateLevelConfig(config)
	if err != nil {
		t.Errorf("Expected no error for Flags=0, got: %v", err)
	}
}

// TestValidateLevelConfig_LaneRestrictionEmpty 测试空行限制 (Story 17.2)
func TestValidateLevelConfig_LaneRestrictionEmpty(t *testing.T) {
	config := &LevelConfig{
		ID:   "test",
		Name: "Test",
		Waves: []WaveConfig{
			{
				LaneRestriction: []int{}, // 空数组是有效的
				Zombies: []ZombieGroup{
					{Type: "basic", Lanes: []int{3}, Count: 1},
				},
			},
		},
	}

	applyDefaults(config)
	err := validateLevelConfig(config)
	if err != nil {
		t.Errorf("Expected no error for empty LaneRestriction, got: %v", err)
	}
}

// TestLoadLevelConfig_FileReadError 测试文件读取错误 (Story 17.2)
func TestLoadLevelConfig_FileReadError(t *testing.T) {
	_, err := LoadLevelConfig("/nonexistent/path/to/level.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

// TestApplyDefaults_EmptySceneType 测试空场景类型默认值 (Story 17.2)
func TestApplyDefaults_EmptySceneType(t *testing.T) {
	config := &LevelConfig{
		ID:        "test",
		Name:      "Test",
		SceneType: "", // 应该被设置为 "day"
		Waves: []WaveConfig{
			{
				Zombies: []ZombieGroup{
					{Type: "basic", Lanes: []int{3}, Count: 1},
				},
			},
		},
	}

	applyDefaults(config)

	if config.SceneType != "day" {
		t.Errorf("Expected default SceneType 'day', got %q", config.SceneType)
	}
}

// TestApplyDefaults_ZeroRowMax 测试 RowMax=0 的默认值 (Story 17.2)
func TestApplyDefaults_ZeroRowMax(t *testing.T) {
	config := &LevelConfig{
		ID:     "test",
		Name:   "Test",
		RowMax: 0, // 应该被设置为 5
		Waves: []WaveConfig{
			{
				Zombies: []ZombieGroup{
					{Type: "basic", Lanes: []int{3}, Count: 1},
				},
			},
		},
	}

	applyDefaults(config)

	if config.RowMax != 5 {
		t.Errorf("Expected default RowMax 5, got %d", config.RowMax)
	}
}

// ============================================================================
// Story 19.4: PresetPlant Tests
// ============================================================================

// TestPresetPlant_Validation 测试预设植物验证
func TestPresetPlant_Validation(t *testing.T) {
	t.Run("valid preset plant config", func(t *testing.T) {
		config := &LevelConfig{
			ID:   "1-5",
			Name: "Bowling Level",
			PresetPlants: []PresetPlant{
				{Type: "peashooter", Row: 2, Col: 6},
				{Type: "peashooter", Row: 3, Col: 8},
				{Type: "peashooter", Row: 4, Col: 7},
			},
			Waves: []WaveConfig{
				{Zombies: []ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 1}}},
			},
		}
		err := validateLevelConfig(config)
		if err != nil {
			t.Errorf("Expected no error for valid preset plants, got: %v", err)
		}
	})

	t.Run("invalid preset plant - missing type", func(t *testing.T) {
		config := &LevelConfig{
			ID:   "1-5",
			Name: "Test",
			PresetPlants: []PresetPlant{
				{Type: "", Row: 2, Col: 6}, // 缺少类型
			},
			Waves: []WaveConfig{
				{Zombies: []ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 1}}},
			},
		}
		err := validateLevelConfig(config)
		if err == nil {
			t.Error("Expected error for preset plant missing type")
		}
	})

	t.Run("invalid preset plant - row out of range", func(t *testing.T) {
		testCases := []struct {
			row int
		}{
			{row: 0},
			{row: 6},
			{row: -1},
		}

		for _, tc := range testCases {
			config := &LevelConfig{
				ID:   "1-5",
				Name: "Test",
				PresetPlants: []PresetPlant{
					{Type: "peashooter", Row: tc.row, Col: 5},
				},
				Waves: []WaveConfig{
					{Zombies: []ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 1}}},
				},
			}
			err := validateLevelConfig(config)
			if err == nil {
				t.Errorf("Expected error for preset plant row %d, got nil", tc.row)
			}
		}
	})

	t.Run("invalid preset plant - col out of range", func(t *testing.T) {
		testCases := []struct {
			col int
		}{
			{col: 0},
			{col: 10},
			{col: -1},
		}

		for _, tc := range testCases {
			config := &LevelConfig{
				ID:   "1-5",
				Name: "Test",
				PresetPlants: []PresetPlant{
					{Type: "peashooter", Row: 3, Col: tc.col},
				},
				Waves: []WaveConfig{
					{Zombies: []ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 1}}},
				},
			}
			err := validateLevelConfig(config)
			if err == nil {
				t.Errorf("Expected error for preset plant col %d, got nil", tc.col)
			}
		}
	})

	t.Run("empty preset plants is valid", func(t *testing.T) {
		config := &LevelConfig{
			ID:           "1-5",
			Name:         "Test",
			PresetPlants: []PresetPlant{},
			Waves: []WaveConfig{
				{Zombies: []ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 1}}},
			},
		}
		err := validateLevelConfig(config)
		if err != nil {
			t.Errorf("Expected no error for empty preset plants, got: %v", err)
		}
	})
}

// TestLoadLevelConfig_WithPresetPlants 测试加载包含预设植物的配置
func TestLoadLevelConfig_WithPresetPlants(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "level-1-5.yaml")

	yamlContent := `id: "1-5"
name: "Wall-nut Bowling"
description: "Shovel tutorial + bowling"
presetPlants:
  - type: "peashooter"
    row: 2
    col: 6
  - type: "peashooter"
    row: 3
    col: 8
  - type: "peashooter"
    row: 4
    col: 7
specialRules: "bowling"
waves:
  - zombies:
      - type: basic
        lanes: [3]
        count: 1
`
	if err := os.WriteFile(testFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config, err := LoadLevelConfig(testFile)
	if err != nil {
		t.Fatalf("LoadLevelConfig() failed: %v", err)
	}

	// 验证预设植物数量
	if len(config.PresetPlants) != 3 {
		t.Errorf("Expected 3 preset plants, got %d", len(config.PresetPlants))
	}

	// 验证第一个预设植物
	if config.PresetPlants[0].Type != "peashooter" {
		t.Errorf("Expected preset plant type 'peashooter', got '%s'", config.PresetPlants[0].Type)
	}
	if config.PresetPlants[0].Row != 2 {
		t.Errorf("Expected preset plant row 2, got %d", config.PresetPlants[0].Row)
	}
	if config.PresetPlants[0].Col != 6 {
		t.Errorf("Expected preset plant col 6, got %d", config.PresetPlants[0].Col)
	}

	// 验证特殊规则
	if config.SpecialRules != "bowling" {
		t.Errorf("Expected specialRules 'bowling', got '%s'", config.SpecialRules)
	}
}

// TestPresetPlant_Struct 测试 PresetPlant 结构体
func TestPresetPlant_Struct(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		plant := PresetPlant{}
		if plant.Type != "" {
			t.Errorf("Expected empty Type, got '%s'", plant.Type)
		}
		if plant.Row != 0 {
			t.Errorf("Expected Row 0, got %d", plant.Row)
		}
		if plant.Col != 0 {
			t.Errorf("Expected Col 0, got %d", plant.Col)
		}
	})

	t.Run("with values", func(t *testing.T) {
		plant := PresetPlant{
			Type: "sunflower",
			Row:  3,
			Col:  5,
		}
		if plant.Type != "sunflower" {
			t.Errorf("Expected Type 'sunflower', got '%s'", plant.Type)
		}
		if plant.Row != 3 {
			t.Errorf("Expected Row 3, got %d", plant.Row)
		}
		if plant.Col != 5 {
			t.Errorf("Expected Col 5, got %d", plant.Col)
		}
	})
}

// ========== Story 19.9: 保龄球关卡配置测试 ==========

// TestLoadLevelConfig_Bowling_RealFile 测试保龄球关卡配置解析逻辑
// 使用 testdata 目录的测试数据，不依赖真实配置文件
func TestLoadLevelConfig_Bowling_RealFile(t *testing.T) {
	config, err := LoadLevelConfig("testdata/level-1-5-bowling-test.yaml")
	if err != nil {
		t.Fatalf("LoadLevelConfig() failed: %v", err)
	}

	// 验证关卡 ID
	if config.ID != "1-5" {
		t.Errorf("Expected ID '1-5', got '%s'", config.ID)
	}

	// 验证无旗帜（flags: 0）
	if config.Flags != 0 {
		t.Errorf("Expected Flags 0 (bowling level has no flags), got %d", config.Flags)
	}

	// 验证 FlagWaves 为空（无旗帜波）
	if len(config.FlagWaves) != 0 {
		t.Errorf("Expected empty FlagWaves (bowling level has no flags), got %v", config.FlagWaves)
	}

	// 验证波次数量（应该有 16 波）
	if len(config.Waves) < 16 {
		t.Errorf("Expected at least 16 waves, got %d", len(config.Waves))
	}

	// 验证最终波类型为 "Final"
	if len(config.Waves) > 0 {
		lastWave := config.Waves[len(config.Waves)-1]
		if lastWave.Type != "Final" {
			t.Errorf("Expected last wave type 'Final', got '%s'", lastWave.Type)
		}
	}

	// 验证所有波次的 IsFlag 都为 false
	for i, wave := range config.Waves {
		if wave.IsFlag {
			t.Errorf("Wave %d should have IsFlag=false (bowling level has no flags), got true", i+1)
		}
	}

	// 验证僵尸类型限制（只有 basic 和 conehead）
	validTypes := map[string]bool{"basic": true, "conehead": true}
	for i, wave := range config.Waves {
		for j, zombie := range wave.Zombies {
			if !validTypes[zombie.Type] {
				t.Errorf("Wave %d zombie %d has invalid type '%s', expected 'basic' or 'conehead'",
					i+1, j+1, zombie.Type)
			}
		}
	}

	// 验证特殊规则为 bowling
	if config.SpecialRules != "bowling" {
		t.Errorf("Expected SpecialRules 'bowling', got '%s'", config.SpecialRules)
	}

	// 验证 openingType 为 special
	if config.OpeningType != "special" {
		t.Errorf("Expected OpeningType 'special', got '%s'", config.OpeningType)
	}

	// 验证奖励植物为 potatomine（Story 19.10 修正：使用无下划线命名约定）
	if config.RewardPlant != "potatomine" {
		t.Errorf("Expected RewardPlant 'potatomine', got '%s'", config.RewardPlant)
	}
}

// TestLoadLevelConfig_Bowling_WavePhases 测试保龄球关卡的四阶段波次结构
// 使用 testdata 目录的测试数据，不依赖真实配置文件
func TestLoadLevelConfig_Bowling_WavePhases(t *testing.T) {
	config, err := LoadLevelConfig("testdata/level-1-5-bowling-test.yaml")
	if err != nil {
		t.Fatalf("LoadLevelConfig() failed: %v", err)
	}

	// 验证阶段一（热身）：波次 1-3，仅普通僵尸
	for i := 0; i < 3 && i < len(config.Waves); i++ {
		wave := config.Waves[i]
		for j, zombie := range wave.Zombies {
			if zombie.Type != "basic" {
				t.Errorf("Phase 1 (Warm Up) wave %d zombie %d should be 'basic', got '%s'",
					i+1, j+1, zombie.Type)
			}
		}
	}

	// 验证阶段二开始引入路障僵尸（波次 4 及之后）
	hasConeheadInPhase2 := false
	for i := 3; i < 8 && i < len(config.Waves); i++ {
		for _, zombie := range config.Waves[i].Zombies {
			if zombie.Type == "conehead" {
				hasConeheadInPhase2 = true
				break
			}
		}
		if hasConeheadInPhase2 {
			break
		}
	}
	if !hasConeheadInPhase2 {
		t.Error("Phase 2 (Conehead Phase) should include conehead zombies")
	}

	// 验证最终波（波次 16）有大量僵尸
	if len(config.Waves) >= 16 {
		finalWave := config.Waves[15]
		totalZombies := 0
		for _, zombie := range finalWave.Zombies {
			totalZombies += zombie.Count
		}
		if totalZombies < 8 {
			t.Errorf("Final wave should have at least 8 zombies, got %d", totalZombies)
		}
	}
}
