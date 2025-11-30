package config

import (
	"testing"
)

// TestLevel1_3_Configuration 测试关卡 1-3 配置（Sprint Change 2025-10-28）
func TestLevel1_3_Configuration(t *testing.T) {
	config, err := LoadLevelConfig("../../data/levels/level-1-3.yaml")
	if err != nil {
		t.Fatalf("Failed to load level-1-3.yaml: %v", err)
	}

	// 验证基本信息
	if config.ID != "1-3" {
		t.Errorf("Expected ID '1-3', got '%s'", config.ID)
	}

	// 验证场地布局：中间3行草地（2, 3, 4）
	t.Run("场地布局", func(t *testing.T) {
		if len(config.EnabledLanes) != 3 {
			t.Errorf("Expected 3 enabled lanes, got %d", len(config.EnabledLanes))
		}
		expectedLanes := []int{2, 3, 4}
		for i, lane := range expectedLanes {
			if i >= len(config.EnabledLanes) || config.EnabledLanes[i] != lane {
				t.Errorf("Expected lanes [2,3,4], got %v", config.EnabledLanes)
				break
			}
		}
	})

	// 验证波次配置：1面旗帜（8波：7个小波次 + 1个旗帜波）
	t.Run("波次配置", func(t *testing.T) {
		if len(config.Waves) != 8 {
			t.Errorf("Expected 8 waves (7 small + 1 flag), got %d", len(config.Waves))
		}

		// 统计旗帜数量
		flagCount := 0
		for _, wave := range config.Waves {
			if wave.IsFlag {
				flagCount++
			}
		}

		if flagCount != 1 {
			t.Errorf("Expected 1 flag wave, got %d", flagCount)
		}

		// 验证旗帜波是最后一波
		if len(config.Waves) > 0 {
			lastWave := config.Waves[len(config.Waves)-1]
			if !lastWave.IsFlag {
				t.Error("Expected last wave to be a flag wave")
			}
			if lastWave.FlagIndex != 1 {
				t.Errorf("Expected flag index 1, got %d", lastWave.FlagIndex)
			}
		}
	})

	// 验证僵尸只在启用的行中生成
	t.Run("僵尸行验证", func(t *testing.T) {
		validLanes := map[int]bool{2: true, 3: true, 4: true}
		for i, wave := range config.Waves {
			for j, zombieGroup := range wave.Zombies {
				for _, lane := range zombieGroup.Lanes {
					if !validLanes[lane] {
						t.Errorf("Wave %d, zombie group %d: Invalid lane %d (expected 2,3,4)",
							i, j, lane)
					}
				}
			}
		}
	})

	// 验证草皮配置
	// 设计规范 (chapter1.md:92): "使用图片 IMAGE_SOD3ROW 预先渲染草皮，没有铺草皮的动画"
	t.Run("草皮配置", func(t *testing.T) {
		// 验证预先渲染的行：中间3行全部预先显示
		expectedPreSoddedLanes := []int{2, 3, 4}
		if len(config.PreSoddedLanes) != 3 {
			t.Errorf("Expected 3 pre-sodded lanes, got %d", len(config.PreSoddedLanes))
		}
		for i, lane := range expectedPreSoddedLanes {
			if i >= len(config.PreSoddedLanes) || config.PreSoddedLanes[i] != lane {
				t.Errorf("Expected preSoddedLanes [2,3,4], got %v", config.PreSoddedLanes)
				break
			}
		}

		// 验证无铺草皮动画
		if config.ShowSoddingAnim {
			t.Error("Expected showSoddingAnim=false (no sodding animation)")
		}
		if config.SodRollAnimation {
			t.Error("Expected sodRollAnimation=false (no sodding animation)")
		}
	})
}

// TestLevel1_4_Configuration 测试关卡 1-4 配置（Sprint Change 2025-10-28）
func TestLevel1_4_Configuration(t *testing.T) {
	config, err := LoadLevelConfig("../../data/levels/level-1-4.yaml")
	if err != nil {
		t.Fatalf("Failed to load level-1-4.yaml: %v", err)
	}

	// 验证基本信息
	if config.ID != "1-4" {
		t.Errorf("Expected ID '1-4', got '%s'", config.ID)
	}

	// 验证场地布局：全部5行草地
	t.Run("场地布局", func(t *testing.T) {
		if len(config.EnabledLanes) != 5 {
			t.Errorf("Expected 5 enabled lanes, got %d", len(config.EnabledLanes))
		}
		expectedLanes := []int{1, 2, 3, 4, 5}
		for i, lane := range expectedLanes {
			if i >= len(config.EnabledLanes) || config.EnabledLanes[i] != lane {
				t.Errorf("Expected lanes [1,2,3,4,5], got %v", config.EnabledLanes)
				break
			}
		}
	})

	// 验证波次配置：1面旗帜（10波：9个小波次 + 1个旗帜波）
	t.Run("波次配置", func(t *testing.T) {
		if len(config.Waves) != 10 {
			t.Errorf("Expected 10 waves (9 small + 1 flag), got %d", len(config.Waves))
		}

		// 统计旗帜数量
		flagCount := 0
		for _, wave := range config.Waves {
			if wave.IsFlag {
				flagCount++
			}
		}

		if flagCount != 1 {
			t.Errorf("Expected 1 flag wave, got %d", flagCount)
		}

		// 验证旗帜波是最后一波
		if len(config.Waves) > 0 {
			lastWave := config.Waves[len(config.Waves)-1]
			if !lastWave.IsFlag {
				t.Error("Expected last wave to be a flag wave")
			}
			if lastWave.FlagIndex != 1 {
				t.Errorf("Expected flag index 1, got %d", lastWave.FlagIndex)
			}
		}
	})

	// 验证僵尸只在启用的行中生成
	t.Run("僵尸行验证", func(t *testing.T) {
		validLanes := map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true}
		for i, wave := range config.Waves {
			for j, zombieGroup := range wave.Zombies {
				for _, lane := range zombieGroup.Lanes {
					if !validLanes[lane] {
						t.Errorf("Wave %d, zombie group %d: Invalid lane %d (expected 1-5)",
							i, j, lane)
					}
				}
			}
		}
	})

	// 验证僵尸类型包含路障僵尸（conehead）
	t.Run("僵尸类型验证", func(t *testing.T) {
		hasConehead := false
		for _, wave := range config.Waves {
			for _, zombieGroup := range wave.Zombies {
				if zombieGroup.Type == "conehead" {
					hasConehead = true
					break
				}
			}
			if hasConehead {
				break
			}
		}
		if !hasConehead {
			t.Error("Expected level 1-4 to include conehead zombies")
		}
	})
}
