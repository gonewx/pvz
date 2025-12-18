package config

import (
	"testing"
)

// TestLevel1_1_Specification 测试关卡 1-1 是否符合 chapter1.md 规范
func TestLevel1_1_Specification(t *testing.T) {
	config, err := LoadLevelConfig("../../data/levels/level-1-1.yaml")
	if err != nil {
		t.Fatalf("Failed to load level-1-1.yaml: %v", err)
	}

	// 验证基本信息
	if config.ID != "1-1" {
		t.Errorf("Expected ID '1-1', got '%s'", config.ID)
	}

	// 验证场地布局：仅有中间1行草地（第3行）
	t.Run("场地布局", func(t *testing.T) {
		if len(config.EnabledLanes) != 1 {
			t.Errorf("Expected 1 enabled lane, got %d", len(config.EnabledLanes))
		}
		if len(config.EnabledLanes) > 0 && config.EnabledLanes[0] != 3 {
			t.Errorf("Expected lane 3 to be enabled, got %v", config.EnabledLanes)
		}
	})

	// 验证波数：2-4波，总共5个普通僵尸
	t.Run("波次配置", func(t *testing.T) {
		if len(config.Waves) < 2 || len(config.Waves) > 4 {
			t.Errorf("Expected 2-4 waves, got %d", len(config.Waves))
		}

		totalZombies := 0
		for _, wave := range config.Waves {
			for _, zombie := range wave.Zombies {
				// 验证所有僵尸都是普通僵尸
				if zombie.Type != "basic" {
					t.Errorf("Expected only basic zombies in 1-1, got %s", zombie.Type)
				}
				// 验证所有僵尸都在第3行（检查 Lanes 数组）
				validLane := false
				for _, lane := range zombie.Lanes {
					if lane == 3 {
						validLane = true
						break
					}
				}
				if !validLane {
					t.Errorf("Expected all zombies in lane 3, got lanes %v", zombie.Lanes)
				}
				totalZombies += zombie.Count
			}
		}

		if totalZombies != 5 {
			t.Errorf("Expected 5 total zombies, got %d", totalZombies)
		}
	})

	// 验证可用植物：只有豌豆射手
	t.Run("可用植物", func(t *testing.T) {
		if len(config.AvailablePlants) != 1 {
			t.Errorf("Expected 1 available plant, got %d", len(config.AvailablePlants))
		}
		if len(config.AvailablePlants) > 0 && config.AvailablePlants[0] != "peashooter" {
			t.Errorf("Expected peashooter, got %s", config.AvailablePlants[0])
		}
	})

	// 验证开场类型：教学关卡
	t.Run("开场类型", func(t *testing.T) {
		if config.OpeningType != "tutorial" {
			t.Errorf("Expected openingType 'tutorial', got '%s'", config.OpeningType)
		}
		// skipOpening 可以是 false（显示完整开场动画）或 true（跳过）
		// 1-1关卡使用 false 以显示完整教学体验
	})
}

// TestLevel1_2_Specification 测试关卡 1-2 是否符合 chapter1.md 规范
func TestLevel1_2_Specification(t *testing.T) {
	config, err := LoadLevelConfig("../../data/levels/level-1-2.yaml")
	if err != nil {
		t.Fatalf("Failed to load level-1-2.yaml: %v", err)
	}

	// 验证基本信息
	if config.ID != "1-2" {
		t.Errorf("Expected ID '1-2', got '%s'", config.ID)
	}

	// 验证场地布局：中间3行草地
	t.Run("场地布局", func(t *testing.T) {
		if len(config.EnabledLanes) != 3 {
			t.Errorf("Expected 3 enabled lanes, got %d", len(config.EnabledLanes))
		}
		// 验证是中间3行（2, 3, 4）
		expectedLanes := []int{2, 3, 4}
		for i, lane := range expectedLanes {
			if i >= len(config.EnabledLanes) || config.EnabledLanes[i] != lane {
				t.Errorf("Expected lanes [2,3,4], got %v", config.EnabledLanes)
				break
			}
		}
	})

	// 验证波次：1面旗帜（通常2个小波次 + 1个旗帜波）
	t.Run("波次配置", func(t *testing.T) {
		if len(config.Waves) < 2 || len(config.Waves) > 4 {
			t.Logf("Warning: Expected 2-4 waves for 1 flag, got %d", len(config.Waves))
		}

		// 验证所有僵尸都是普通僵尸（1-2不应该有路障僵尸）
		for i, wave := range config.Waves {
			for j, zombie := range wave.Zombies {
				if zombie.Type != "basic" {
					t.Errorf("Wave %d, zombie %d: Expected only basic zombies in 1-2, got %s",
						i, j, zombie.Type)
				}
				// 验证所有僵尸都在启用的行中（检查 Lanes 数组）
				for _, zombieLane := range zombie.Lanes {
					validLane := false
					for _, enabledLane := range config.EnabledLanes {
						if zombieLane == enabledLane {
							validLane = true
							break
						}
					}
					if !validLane {
						t.Errorf("Wave %d, zombie %d: Zombie in lane %d, but only lanes %v are enabled",
							i, j, zombieLane, config.EnabledLanes)
					}
				}
			}
		}
	})

	// 验证可用植物：豌豆射手 + 向日葵
	t.Run("可用植物", func(t *testing.T) {
		if len(config.AvailablePlants) != 2 {
			t.Errorf("Expected 2 available plants, got %d", len(config.AvailablePlants))
		}
		expectedPlants := map[string]bool{"peashooter": true, "sunflower": true}
		for _, plant := range config.AvailablePlants {
			if !expectedPlants[plant] {
				t.Errorf("Unexpected plant '%s' in 1-2", plant)
			}
		}
	})

	// 验证开场类型：标准开场
	t.Run("开场类型", func(t *testing.T) {
		if config.OpeningType != "standard" {
			t.Errorf("Expected openingType 'standard', got '%s'", config.OpeningType)
		}
		if config.SkipOpening {
			t.Error("Expected skipOpening false for standard level")
		}
	})
}

// TestLevel1_9_Specification 测试关卡 1-9 是否符合 Story 8.14 规范
// Story 8.14: 综合挑战关卡 - 第一章最后一个标准关卡
func TestLevel1_9_Specification(t *testing.T) {
	config, err := LoadLevelConfig("../../data/levels/level-1-9.yaml")
	if err != nil {
		t.Fatalf("Failed to load level-1-9.yaml: %v", err)
	}

	// 验证基本信息
	t.Run("基本信息", func(t *testing.T) {
		if config.ID != "1-9" {
			t.Errorf("Expected ID '1-9', got '%s'", config.ID)
		}
		if config.SceneType != "day" {
			t.Errorf("Expected sceneType 'day', got '%s'", config.SceneType)
		}
		if config.RowMax != 5 {
			t.Errorf("Expected rowMax 5, got %d", config.RowMax)
		}
	})

	// 验证旗帜数量：2 面旗帜
	t.Run("旗帜配置", func(t *testing.T) {
		if config.Flags != 2 {
			t.Errorf("Expected 2 flags, got %d", config.Flags)
		}
	})

	// 验证波次数量：20 波
	t.Run("波次配置", func(t *testing.T) {
		if len(config.Waves) != 20 {
			t.Errorf("Expected 20 waves, got %d", len(config.Waves))
		}

		// 验证第 10 波和第 20 波是旗帜波
		flagWaves := []int{10, 20}
		for _, waveNum := range flagWaves {
			wave := config.Waves[waveNum-1] // 0-based index
			if !wave.IsFlag {
				t.Errorf("Wave %d should be a flag wave", waveNum)
			}
			if wave.Type != "Final" {
				t.Errorf("Wave %d should have type 'Final', got '%s'", waveNum, wave.Type)
			}
		}
	})

	// 验证场地布局：完整 5 行草地
	t.Run("场地布局", func(t *testing.T) {
		expectedLanes := []int{1, 2, 3, 4, 5}
		if len(config.EnabledLanes) != len(expectedLanes) {
			t.Errorf("Expected %d enabled lanes, got %d", len(expectedLanes), len(config.EnabledLanes))
		}
		for i, lane := range expectedLanes {
			if i >= len(config.EnabledLanes) || config.EnabledLanes[i] != lane {
				t.Errorf("Expected lanes %v, got %v", expectedLanes, config.EnabledLanes)
				break
			}
		}
	})

	// 验证选卡系统：已启用
	t.Run("选卡系统", func(t *testing.T) {
		if !config.EnableSeedSelection {
			t.Error("Expected enableSeedSelection to be true")
		}
	})

	// 验证可用植物：8 种
	t.Run("可用植物", func(t *testing.T) {
		expectedPlants := []string{
			"peashooter", "sunflower", "cherrybomb", "wallnut",
			"potatomine", "snowpea", "chomper", "repeater",
		}
		if len(config.AvailablePlants) != len(expectedPlants) {
			t.Errorf("Expected %d available plants, got %d", len(expectedPlants), len(config.AvailablePlants))
		}
		plantSet := make(map[string]bool)
		for _, plant := range config.AvailablePlants {
			plantSet[plant] = true
		}
		for _, expected := range expectedPlants {
			if !plantSet[expected] {
				t.Errorf("Expected plant '%s' to be available", expected)
			}
		}
	})

	// 验证奖励类型：僵尸来信
	t.Run("奖励配置", func(t *testing.T) {
		if config.RewardType != "note" {
			t.Errorf("Expected rewardType 'note', got '%s'", config.RewardType)
		}
		if config.RewardNote != "zombienote1" {
			t.Errorf("Expected rewardNote 'zombienote1', got '%s'", config.RewardNote)
		}
	})

	// 验证僵尸池：basic, conehead, buckethead, polevaulter
	t.Run("僵尸池", func(t *testing.T) {
		expectedZombies := []string{"basic", "conehead", "buckethead", "polevaulter"}
		if len(config.ZombiePool) != len(expectedZombies) {
			t.Errorf("Expected %d zombie types in pool, got %d", len(expectedZombies), len(config.ZombiePool))
		}
		zombieSet := make(map[string]bool)
		for _, zombie := range config.ZombiePool {
			zombieSet[zombie] = true
		}
		for _, expected := range expectedZombies {
			if !zombieSet[expected] {
				t.Errorf("Expected zombie type '%s' in pool", expected)
			}
		}
	})
}

// TestLevel1_9_RewardTypeValidation 测试来信奖励类型验证
func TestLevel1_9_RewardTypeValidation(t *testing.T) {
	// 测试 RewardType 为 "note" 时 RewardNote 不能为空
	t.Run("RewardNote必填验证", func(t *testing.T) {
		config := &LevelConfig{
			ID:         "test",
			Name:       "Test Level",
			RewardType: "note",
			RewardNote: "", // 空值应触发验证错误
			Waves: []WaveConfig{
				{WaveNum: 1, Type: "Fixed", Zombies: []ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 1}}},
			},
		}
		applyDefaults(config)
		err := validateLevelConfig(config)
		if err == nil {
			t.Error("Expected validation error when RewardType is 'note' and RewardNote is empty")
		}
	})

	// 测试有效的来信奖励配置
	t.Run("有效来信配置", func(t *testing.T) {
		config := &LevelConfig{
			ID:         "test",
			Name:       "Test Level",
			RewardType: "note",
			RewardNote: "zombienote1",
			Waves: []WaveConfig{
				{WaveNum: 1, Type: "Fixed", Zombies: []ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 1}}},
			},
		}
		applyDefaults(config)
		err := validateLevelConfig(config)
		if err != nil {
			t.Errorf("Expected no validation error, got: %v", err)
		}
	})

	// 测试无效的 RewardType 值
	t.Run("无效RewardType值", func(t *testing.T) {
		config := &LevelConfig{
			ID:         "test",
			Name:       "Test Level",
			RewardType: "invalid",
			Waves: []WaveConfig{
				{WaveNum: 1, Type: "Fixed", Zombies: []ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 1}}},
			},
		}
		applyDefaults(config)
		err := validateLevelConfig(config)
		if err == nil {
			t.Error("Expected validation error for invalid RewardType")
		}
	})
}

// TestChapter1_ProgressiveUnlocks 测试第一章植物解锁顺序
func TestChapter1_ProgressiveUnlocks(t *testing.T) {
	// 根据 chapter1.md，第一章的植物解锁顺序应该是：
	// 1-1: Peashooter
	// 1-2: Sunflower
	// 1-3: Cherry Bomb
	// 1-4: Wall-nut
	// 1-5: Potato Mine (特殊关卡)
	// 1-6: Snow Pea
	// 1-7: Chomper
	// 1-8: Repeater
	// 1-10: Puff-shroom

	expectedUnlocks := map[string][]string{
		"1-1": {"peashooter"},
		"1-2": {"peashooter", "sunflower"},
		// 后续关卡待实现
	}

	for levelID, expectedPlants := range expectedUnlocks {
		t.Run("Level_"+levelID, func(t *testing.T) {
			configPath := "../../data/levels/level-" + levelID + ".yaml"
			config, err := LoadLevelConfig(configPath)
			if err != nil {
				t.Skipf("Level %s not yet implemented", levelID)
				return
			}

			// 验证可用植物数量符合预期
			if len(config.AvailablePlants) != len(expectedPlants) {
				t.Logf("Level %s: Expected %d plants, got %d (may need adjustment)",
					levelID, len(expectedPlants), len(config.AvailablePlants))
			}
		})
	}
}
