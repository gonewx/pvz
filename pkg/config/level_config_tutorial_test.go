package config

import (
	"testing"
)

// TestLevel1_1_TutorialConfig 验证 1-1 关卡的教学配置 (Story 8.2)
func TestLevel1_1_TutorialConfig(t *testing.T) {
	// 加载实际的 level-1-1.yaml 文件
	config, err := LoadLevelConfig("../../data/levels/level-1-1.yaml")
	if err != nil {
		t.Fatalf("Failed to load level-1-1.yaml: %v", err)
	}

	// 验证关卡ID
	if config.ID != "1-1" {
		t.Errorf("Expected ID '1-1', got %q", config.ID)
	}

	// 验证教学关卡类型
	if config.OpeningType != "tutorial" {
		t.Errorf("Expected openingType 'tutorial', got %q", config.OpeningType)
	}

	// 验证启用的行数
	if len(config.EnabledLanes) != 1 {
		t.Errorf("Expected 1 enabled lane, got %d", len(config.EnabledLanes))
	}
	if len(config.EnabledLanes) > 0 && config.EnabledLanes[0] != 3 {
		t.Errorf("Expected enabled lane 3, got %d", config.EnabledLanes[0])
	}

	// 验证可用植物
	if len(config.AvailablePlants) != 1 {
		t.Errorf("Expected 1 available plant, got %d", len(config.AvailablePlants))
	}
	if len(config.AvailablePlants) > 0 && config.AvailablePlants[0] != "peashooter" {
		t.Errorf("Expected available plant 'peashooter', got %q", config.AvailablePlants[0])
	}

	// 验证跳过开场动画
	if !config.SkipOpening {
		t.Error("Expected skipOpening to be true for tutorial level")
	}

	// Story 8.2 方案A+：验证初始阳光值（教学关卡改为150）
	if config.InitialSun != 150 {
		t.Errorf("Expected initialSun 150, got %d", config.InitialSun)
	}

	// 验证教学步骤数量（方案A+：9步完整流程）
	expectedSteps := 9
	if len(config.TutorialSteps) != expectedSteps {
		t.Fatalf("Expected %d tutorial steps, got %d", expectedSteps, len(config.TutorialSteps))
	}

	// 验证每个教学步骤
	expectedStepsData := []struct {
		trigger string
		textKey string
		action  string
	}{
		{"gameStart", "ADVICE_CLICK_ON_SUN", "waitForSunClick"},
		{"sunClicked", "ADVICE_CLICKED_ON_SUN", "waitForEnoughSun"},
		{"enoughSun", "ADVICE_CLICK_SEED_PACKET", "waitForSeedClick"},
		{"seedClicked", "ADVICE_CLICK_ON_GRASS", "waitForPlantPlaced"},
		{"plantPlaced", "ADVICE_PLANTED_PEASHOOTER", "waitForCooldownFinished"},
		{"cooldownFinished", "ADVICE_CLICK_PEASHOOTER", "waitForSecondSeedClick"},
		{"enoughSunNotPlanting", "ADVICE_ENOUGH_SUN", "reminder"},
		{"secondSeedClicked", "ADVICE_CLICK_ON_GRASS", "waitForSecondPlantPlaced"},
		{"secondPlantPlaced", "ADVICE_ZOMBIE_ONSLAUGHT", "waitForLevelEnd"},
	}

	for i, expected := range expectedStepsData {
		step := config.TutorialSteps[i]
		if step.Trigger != expected.trigger {
			t.Errorf("Step %d: expected trigger %q, got %q", i, expected.trigger, step.Trigger)
		}
		if step.TextKey != expected.textKey {
			t.Errorf("Step %d: expected textKey %q, got %q", i, expected.textKey, step.TextKey)
		}
		if step.Action != expected.action {
			t.Errorf("Step %d: expected action %q, got %q", i, expected.action, step.Action)
		}
	}

	// 验证僵尸波次配置
	if len(config.Waves) != 3 {
		t.Errorf("Expected 3 waves, got %d", len(config.Waves))
	}

	// 验证所有僵尸都在第3行
	for i, wave := range config.Waves {
		for j, zombie := range wave.Zombies {
			if zombie.Lane != 3 {
				t.Errorf("Wave %d, zombie %d: expected lane 3, got %d", i, j, zombie.Lane)
			}
			if zombie.Type != "basic" {
				t.Errorf("Wave %d, zombie %d: expected type 'basic', got %q", i, j, zombie.Type)
			}
		}
	}

	t.Logf("Level 1-1 tutorial config validation passed")
}

// TestTutorialSteps_AllTextKeysExist 验证所有教学文本键在 LawnStrings.txt 中存在
func TestTutorialSteps_AllTextKeysExist(t *testing.T) {
	// 加载 1-1 关卡配置
	config, err := LoadLevelConfig("../../data/levels/level-1-1.yaml")
	if err != nil {
		t.Skipf("Skipping test: level-1-1.yaml not found: %v", err)
	}

	// 加载 LawnStrings.txt
	// 注意：这里需要从 pkg/game 导入，但为了避免循环依赖，我们只验证配置格式
	// 实际的文本键存在性由 TestLawnStrings_RealFile 测试

	expectedTextKeys := []string{
		"ADVICE_CLICK_ON_SUN",
		"ADVICE_CLICKED_ON_SUN",
		"ADVICE_CLICK_SEED_PACKET",
		"ADVICE_CLICK_ON_GRASS",
		"ADVICE_PLANTED_PEASHOOTER",
		"ADVICE_CLICK_PEASHOOTER",    // 方案A+ 新增
		"ADVICE_ENOUGH_SUN",          // 方案A+ 新增
		"ADVICE_ZOMBIE_ONSLAUGHT",
	}

	for i, step := range config.TutorialSteps {
		found := false
		for _, expectedKey := range expectedTextKeys {
			if step.TextKey == expectedKey {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Step %d: unexpected textKey %q (not in expected list)", i, step.TextKey)
		}
	}
}

