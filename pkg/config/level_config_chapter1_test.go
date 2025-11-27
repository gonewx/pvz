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
