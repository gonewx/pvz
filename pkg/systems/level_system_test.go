package systems

import (
	"math"
	"testing"

	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestCalculateTotalZombies 测试总僵尸数计算逻辑
func TestCalculateTotalZombies(t *testing.T) {
	tests := []struct {
		name     string
		waves    []config.WaveConfig
		expected int
	}{
		{
			name: "单波次单僵尸",
			waves: []config.WaveConfig{
				{Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
			},
			expected: 1,
		},
		{
			name: "单波次多僵尸",
			waves: []config.WaveConfig{
				{Zombies: []config.ZombieSpawn{
					{Type: "basic", Lane: 1, Count: 2},
					{Type: "basic", Lane: 2, Count: 3},
				}},
			},
			expected: 5,
		},
		{
			name: "多波次多僵尸",
			waves: []config.WaveConfig{
				{Zombies: []config.ZombieSpawn{
					{Type: "basic", Lane: 1, Count: 2},
					{Type: "basic", Lane: 2, Count: 1},
				}},
				{Zombies: []config.ZombieSpawn{
					{Type: "basic", Lane: 3, Count: 3},
				}},
				{Zombies: []config.ZombieSpawn{
					{Type: "basic", Lane: 1, Count: 1},
					{Type: "basic", Lane: 2, Count: 1},
					{Type: "basic", Lane: 3, Count: 1},
				}},
			},
			expected: 9,
		},
		{
			name:     "空波次",
			waves:    []config.WaveConfig{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			levelConfig := &config.LevelConfig{
				Waves: tt.waves,
			}

			// 创建临时的 LevelSystem 来测试（使用 nil 参数，因为我们只测试计算函数）
			ls := &LevelSystem{}
			ls.gameState = &game.GameState{CurrentLevel: levelConfig}

			actual := ls.calculateTotalZombies()
			if actual != tt.expected {
				t.Errorf("calculateTotalZombies() = %d, expected %d", actual, tt.expected)
			}
		})
	}
}

// TestCalculateFlagPositions 测试旗帜位置计算逻辑
func TestCalculateFlagPositions(t *testing.T) {
	tests := []struct {
		name      string
		waves     []config.WaveConfig
		flagWaves []int
		expected  []float64
	}{
		{
			name: "单旗帜在中间",
			waves: []config.WaveConfig{
				{Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 5}}},
				{Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 5}}},
			},
			flagWaves: []int{1}, // 第2波是旗帜波
			expected:  []float64{0.5},
		},
		{
			name: "单旗帜在最后",
			waves: []config.WaveConfig{
				{Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 3}}},
				{Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 2}}},
				{Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 5}}},
			},
			flagWaves: []int{2}, // 第3波是旗帜波
			expected:  []float64{0.5},
		},
		{
			name: "多旗帜",
			waves: []config.WaveConfig{
				{Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 2}}},
				{Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 3}}},
				{Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 2}}},
				{Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 3}}},
			},
			flagWaves: []int{1, 3}, // 第2波和第4波是旗帜波
			expected:  []float64{0.2, 0.7},
		},
		{
			name: "无旗帜",
			waves: []config.WaveConfig{
				{Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 5}}},
			},
			flagWaves: []int{},
			expected:  []float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			levelConfig := &config.LevelConfig{
				Waves:     tt.waves,
				FlagWaves: tt.flagWaves,
			}

			ls := &LevelSystem{}
			ls.gameState = &game.GameState{CurrentLevel: levelConfig}

			actual := ls.calculateFlagPositions()

			if len(actual) != len(tt.expected) {
				t.Fatalf("旗帜数量不匹配: got %d, expected %d", len(actual), len(tt.expected))
			}

			for i, pos := range actual {
				// 允许浮点误差
				if math.Abs(pos-tt.expected[i]) > 0.01 {
					t.Errorf("旗帜 %d 位置错误: got %.2f, expected %.2f", i, pos, tt.expected[i])
				}
			}
		})
	}
}

// ========================================
// Story 11.3: Final Wave Warning Tests
// ========================================

// TestIsFinalWaveApproaching 测试最后一波检测逻辑
func TestIsFinalWaveApproaching(t *testing.T) {
	tests := []struct {
		name                  string
		currentWaveIndex      int
		levelTime             float64
		lastWaveCompletedTime float64
		isWaitingForNextWave  bool
		waves                 []config.WaveConfig
		finalWaveLeadTime     float64
		expected              bool
	}{
		{
			name:                  "最后一波前 2 秒（应触发）",
			currentWaveIndex:      2,
			levelTime:             53.0,  // 上一波完成于 50 秒
			lastWaveCompletedTime: 50.0,
			isWaitingForNextWave:  true,
			waves: []config.WaveConfig{
				{MinDelay: 0},   // 第一波
				{MinDelay: 0},   // 第二波
				{MinDelay: 5.0}, // 第三波（最后一波），需要等待 5 秒
			},
			finalWaveLeadTime: 3.0,
			expected:          true, // 已等待 3 秒，还剩 2 秒，应触发
		},
		{
			name:                  "最后一波前 5 秒（不触发）",
			currentWaveIndex:      2,
			levelTime:             50.0, // 上一波刚完成
			lastWaveCompletedTime: 50.0,
			isWaitingForNextWave:  true,
			waves: []config.WaveConfig{
				{MinDelay: 0},
				{MinDelay: 0},
				{MinDelay: 5.0}, // 最后一波
			},
			finalWaveLeadTime: 3.0,
			expected:          false, // 还剩 5 秒，不应触发
		},
		{
			name:                  "不是最后一波（不触发）",
			currentWaveIndex:      0,
			levelTime:             3.0,
			lastWaveCompletedTime: 0.0,
			isWaitingForNextWave:  true,
			waves: []config.WaveConfig{
				{MinDelay: 0},
				{MinDelay: 5.0},
				{MinDelay: 5.0},
			},
			finalWaveLeadTime: 3.0,
			expected:          false, // 不是最后一波
		},
		{
			name:                  "未进入等待状态（不触发）",
			currentWaveIndex:      2,
			levelTime:             53.0,
			lastWaveCompletedTime: 50.0,
			isWaitingForNextWave:  false, // 上一波还有僵尸
			waves: []config.WaveConfig{
				{MinDelay: 0},
				{MinDelay: 0},
				{MinDelay: 5.0},
			},
			finalWaveLeadTime: 3.0,
			expected:          false, // 未进入等待状态
		},
		{
			name:                  "恰好剩余 3 秒（边界，应触发）",
			currentWaveIndex:      2,
			levelTime:             52.0,
			lastWaveCompletedTime: 50.0,
			isWaitingForNextWave:  true,
			waves: []config.WaveConfig{
				{MinDelay: 0},
				{MinDelay: 0},
				{MinDelay: 5.0},
			},
			finalWaveLeadTime: 3.0,
			expected:          true, // 恰好 3 秒，应触发
		},
		{
			name:                  "时间已过（不触发）",
			currentWaveIndex:      2,
			levelTime:             56.0, // 已超过触发时间
			lastWaveCompletedTime: 50.0,
			isWaitingForNextWave:  true,
			waves: []config.WaveConfig{
				{MinDelay: 0},
				{MinDelay: 0},
				{MinDelay: 5.0},
			},
			finalWaveLeadTime: 3.0,
			expected:          false, // 时间已过（> 0），不应触发
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 GameState
			gs := &game.GameState{
				CurrentWaveIndex:      tt.currentWaveIndex,
				LevelTime:             tt.levelTime,
				LastWaveCompletedTime: tt.lastWaveCompletedTime,
				IsWaitingForNextWave:  tt.isWaitingForNextWave,
				CurrentLevel: &config.LevelConfig{
					Waves: tt.waves,
				},
			}

			// 创建 LevelSystem
			ls := &LevelSystem{
				gameState:                gs,
				finalWaveWarningLeadTime: tt.finalWaveLeadTime,
			}

			// 测试
			actual := ls.isFinalWaveApproaching()
			if actual != tt.expected {
				t.Errorf("isFinalWaveApproaching() = %v, expected %v", actual, tt.expected)
				t.Logf("  CurrentWaveIndex: %d", tt.currentWaveIndex)
				t.Logf("  LevelTime: %.1f", tt.levelTime)
				t.Logf("  LastWaveCompletedTime: %.1f", tt.lastWaveCompletedTime)
				t.Logf("  IsWaitingForNextWave: %v", tt.isWaitingForNextWave)
				t.Logf("  MinDelay: %.1f", tt.waves[tt.currentWaveIndex].MinDelay)
			}
		})
	}
}

// TestCheckFinalWaveWarningTriggerOnce 测试只触发一次
func TestCheckFinalWaveWarningTriggerOnce(t *testing.T) {
	// 创建 GameState
	gs := &game.GameState{
		CurrentWaveIndex:      2,
		LevelTime:             53.0,  // 恰好满足触发条件
		LastWaveCompletedTime: 50.0,
		IsWaitingForNextWave:  true,
		CurrentLevel: &config.LevelConfig{
			Waves: []config.WaveConfig{
				{MinDelay: 0},
				{MinDelay: 0},
				{MinDelay: 5.0}, // 最后一波
			},
		},
	}

	// 创建 EntityManager（防止实体创建时的空指针）
	em := ecs.NewEntityManager()

	// 创建 ResourceManager（防止音效调用时的空指针）
	rm := game.NewResourceManager(nil)

	ls := &LevelSystem{
		entityManager:             em,
		gameState:                 gs,
		resourceManager:           rm,
		finalWaveWarningTriggered: false,
		finalWaveWarningLeadTime:  3.0,
		// reanimSystem 可以为 nil - triggerFinalWaveWarning 会处理错误
		reanimSystem: nil,
	}

	// 第一次检查：应触发
	if ls.finalWaveWarningTriggered {
		t.Errorf("第一次检查前，finalWaveWarningTriggered 应该为 false")
	}

	// 由于缺少实际资源，triggerFinalWaveWarning 会打印错误日志
	// 但标志位应该仍然会被设置为 true
	ls.checkFinalWaveWarning(0.1)

	if !ls.finalWaveWarningTriggered {
		t.Errorf("第一次检查后，finalWaveWarningTriggered 应该为 true")
	}

	// 第二次检查：不应再触发（通过标志位防止）
	triggerCountBefore := ls.finalWaveWarningTriggered
	ls.checkFinalWaveWarning(0.1)
	triggerCountAfter := ls.finalWaveWarningTriggered

	// 标志位应该仍然为 true（不会被重置）
	if !triggerCountAfter || triggerCountAfter != triggerCountBefore {
		t.Errorf("第二次检查后，finalWaveWarningTriggered 应该仍为 true 且不变")
	}
}

