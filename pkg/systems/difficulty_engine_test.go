package systems

import (
	"testing"

	"github.com/gonewx/pvz/pkg/config"
)

// 创建测试用的 ZombieStatsConfig
func newTestZombieStats() *config.ZombieStatsConfig {
	return &config.ZombieStatsConfig{
		Zombies: map[string]config.ZombieStats{
			"basic":      {Level: 1, Weight: 4000, BaseHealth: 270},
			"conehead":   {Level: 2, Weight: 4000, BaseHealth: 270},
			"buckethead": {Level: 4, Weight: 3000, BaseHealth: 270},
			"gargantuar": {Level: 10, Weight: 1500, BaseHealth: 3000},
		},
	}
}

func TestNewDifficultyEngine(t *testing.T) {
	stats := newTestZombieStats()
	engine := NewDifficultyEngine(stats)

	if engine == nil {
		t.Fatal("NewDifficultyEngine returned nil")
	}

	if engine.zombieStats != stats {
		t.Error("DifficultyEngine.zombieStats not set correctly")
	}
}

func TestCalculateRoundNumber(t *testing.T) {
	engine := NewDifficultyEngine(newTestZombieStats())

	tests := []struct {
		name                string
		totalCompletedFlags int
		expectedRound       int
	}{
		// 一周目早期关卡（负数轮数）
		{"一周目1-1（0旗）", 0, -1},
		{"一周目1-2（1旗）", 1, -1},
		{"一周目1-3（2旗）", 2, 0},
		{"一周目1-4（3旗）", 3, 0},
		{"一周目1-5（4旗）", 4, 1},
		{"一周目1-6（5旗）", 5, 1},

		// 中期关卡
		{"10旗", 10, 4},
		{"20旗", 20, 9},
		{"30旗", 30, 14},

		// 一周目完成
		{"一周目完成（50旗）", 50, 24},

		// 二周目
		{"二周目开始", 52, 25},
		{"二周目进行中", 100, 49},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.CalculateRoundNumber(tt.totalCompletedFlags)
			if result != tt.expectedRound {
				t.Errorf("CalculateRoundNumber(%d): expected %d, got %d",
					tt.totalCompletedFlags, tt.expectedRound, result)
			}
		})
	}
}

func TestCalculateLevelCapacity(t *testing.T) {
	engine := NewDifficultyEngine(newTestZombieStats())

	tests := []struct {
		name          string
		waveNum       int
		roundNumber   int
		wavesPerRound int
		isFlagWave    bool
		expected      int
	}{
		// 基础测试：轮数0，每轮20波
		{"第1波，轮数0，非旗帜波", 1, 0, 20, false, 1},
		{"第2波，轮数0，非旗帜波", 2, 0, 20, false, 1},
		{"第5波，轮数0，非旗帜波", 5, 0, 20, false, 3},
		{"第10波，轮数0，非旗帜波", 10, 0, 20, false, 5},
		{"第20波，轮数0，非旗帜波", 20, 0, 20, false, 9},

		// 旗帜波（× 2.5）
		{"第10波，轮数0，旗帜波", 10, 0, 20, true, 12}, // 5 * 2.5 = 12.5 → 12
		{"第20波，轮数0，旗帜波", 20, 0, 20, true, 22}, // 9 * 2.5 = 22.5 → 22

		// 不同轮数
		{"第1波，轮数1，非旗帜波", 1, 1, 20, false, 9},    // (1 + 1*20) * 0.8 / 2 + 1 = 9.4 → 9
		{"第1波，轮数5，非旗帜波", 1, 5, 20, false, 41},   // (1 + 5*20) * 0.8 / 2 + 1 = 41.4 → 41
		{"第10波，轮数1，非旗帜波", 10, 1, 20, false, 13}, // (10 + 1*20) * 0.8 / 2 + 1 = 13

		// 负数轮数（一周目早期）
		// 公式: int(int((waveNum + roundNumber * wavesPerRound) * 0.8) / 2) + 1
		// 第1波，轮数-1: (1 + -1*20) * 0.8 / 2 + 1 = (-19 * 0.8) / 2 + 1 = -15.2 / 2 + 1 = -7.6 + 1 = -6.6 → -6
		{"第1波，轮数-1，非旗帜波", 1, -1, 20, false, -6},
		// 第10波，轮数-1: (10 + -1*20) * 0.8 / 2 + 1 = (-10 * 0.8) / 2 + 1 = -8 / 2 + 1 = -4 + 1 = -3
		{"第10波，轮数-1，非旗帜波", 10, -1, 20, false, -3},

		// 二周目高轮数
		// 第20波，轮数25: (20 + 25*20) * 0.8 / 2 + 1 = (520 * 0.8) / 2 + 1 = 416 / 2 + 1 = 208 + 1 = 209
		// 旗帜波: 209 * 2.5 = 522.5 → 522
		{"第20波，轮数25，旗帜波", 20, 25, 20, true, 522},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.CalculateLevelCapacity(tt.waveNum, tt.roundNumber, tt.wavesPerRound, tt.isFlagWave)
			if result != tt.expected {
				t.Errorf("CalculateLevelCapacity(%d, %d, %d, %v): expected %d, got %d",
					tt.waveNum, tt.roundNumber, tt.wavesPerRound, tt.isFlagWave, tt.expected, result)
			}
		})
	}
}

func TestGetZombieLevel(t *testing.T) {
	engine := NewDifficultyEngine(newTestZombieStats())

	tests := []struct {
		name       string
		zombieType string
		expected   int
	}{
		{"普通僵尸", "basic", 1},
		{"路障僵尸", "conehead", 2},
		{"铁桶僵尸", "buckethead", 4},
		{"巨人僵尸", "gargantuar", 10},
		{"未知僵尸（默认级别1）", "unknown", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.GetZombieLevel(tt.zombieType)
			if result != tt.expected {
				t.Errorf("GetZombieLevel(%s): expected %d, got %d",
					tt.zombieType, tt.expected, result)
			}
		})
	}
}

func TestGetZombieLevel_NilStats(t *testing.T) {
	engine := NewDifficultyEngine(nil)

	// 当 zombieStats 为 nil 时，应返回默认级别 1
	result := engine.GetZombieLevel("basic")
	if result != 1 {
		t.Errorf("GetZombieLevel with nil stats: expected 1, got %d", result)
	}
}

func TestCalculateTotalLevel(t *testing.T) {
	engine := NewDifficultyEngine(newTestZombieStats())

	tests := []struct {
		name        string
		zombieTypes []string
		expected    int
	}{
		{"空列表", []string{}, 0},
		{"单个普通僵尸", []string{"basic"}, 1},
		{"两个普通僵尸", []string{"basic", "basic"}, 2},
		{"一个路障僵尸", []string{"conehead"}, 2},
		{"混合僵尸", []string{"basic", "conehead", "buckethead"}, 7}, // 1 + 2 + 4 = 7
		{"包含巨人僵尸", []string{"basic", "gargantuar"}, 11},          // 1 + 10 = 11
		{"多个铁桶僵尸", []string{"buckethead", "buckethead"}, 8},      // 4 + 4 = 8
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.CalculateTotalLevel(tt.zombieTypes)
			if result != tt.expected {
				t.Errorf("CalculateTotalLevel(%v): expected %d, got %d",
					tt.zombieTypes, tt.expected, result)
			}
		})
	}
}

func TestValidateWaveCapacity(t *testing.T) {
	engine := NewDifficultyEngine(newTestZombieStats())

	tests := []struct {
		name        string
		zombieTypes []string
		capacityCap int
		expected    bool
	}{
		// 容量上限为 5
		{"5个普通僵尸（5×1=5），上限5：通过", []string{"basic", "basic", "basic", "basic", "basic"}, 5, true},
		{"6个普通僵尸（6×1=6），上限5：失败", []string{"basic", "basic", "basic", "basic", "basic", "basic"}, 5, false},
		{"2个路障僵尸（2×2=4），上限5：通过", []string{"conehead", "conehead"}, 5, true},
		{"3个路障僵尸（3×2=6），上限5：失败", []string{"conehead", "conehead", "conehead"}, 5, false},
		{"1个铁桶僵尸（1×4=4），上限5：通过", []string{"buckethead"}, 5, true},
		{"2个铁桶僵尸（2×4=8），上限5：失败", []string{"buckethead", "buckethead"}, 5, false},

		// 容量上限为 10
		{"混合僵尸（1+2+4=7），上限10：通过", []string{"basic", "conehead", "buckethead"}, 10, true},
		{"1个巨人僵尸（10），上限10：通过", []string{"gargantuar"}, 10, true},
		{"1个巨人+1个普通（11），上限10：失败", []string{"gargantuar", "basic"}, 10, false},

		// 边界情况
		{"空列表，上限5：通过", []string{}, 5, true},
		{"刚好等于上限", []string{"basic", "basic", "basic"}, 3, true},
		{"超过上限1点", []string{"basic", "basic", "basic", "basic"}, 3, false},

		// 高容量上限
		{"大量僵尸，高上限：通过", []string{"buckethead", "buckethead", "buckethead", "conehead", "basic"}, 20, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.ValidateWaveCapacity(tt.zombieTypes, tt.capacityCap)
			if result != tt.expected {
				t.Errorf("ValidateWaveCapacity(%v, %d): expected %v, got %v",
					tt.zombieTypes, tt.capacityCap, tt.expected, result)
			}
		})
	}
}

func TestGetZombieStats(t *testing.T) {
	stats := newTestZombieStats()
	engine := NewDifficultyEngine(stats)

	if engine.GetZombieStats() != stats {
		t.Error("GetZombieStats() did not return the expected stats")
	}
}

func TestGetZombieStats_Nil(t *testing.T) {
	engine := NewDifficultyEngine(nil)

	if engine.GetZombieStats() != nil {
		t.Error("GetZombieStats() should return nil when stats is nil")
	}
}

// 集成测试：验证完整的难度计算流程
func TestDifficultyEngine_Integration(t *testing.T) {
	engine := NewDifficultyEngine(newTestZombieStats())

	// 场景：一周目 1-3 关（2旗完成），第5波
	totalCompletedFlags := 2
	roundNumber := engine.CalculateRoundNumber(totalCompletedFlags) // 应该是 0
	waveNum := 5
	isFlagWave := false

	if roundNumber != 0 {
		t.Errorf("Round number: expected 0, got %d", roundNumber)
	}

	capacity := engine.CalculateLevelCapacity(waveNum, roundNumber, 20, isFlagWave)
	// capacity = int(int((5 + 0*20) * 0.8) / 2) + 1 = int(int(4) / 2) + 1 = 2 + 1 = 3
	if capacity != 3 {
		t.Errorf("Capacity: expected 3, got %d", capacity)
	}

	// 验证一些僵尸组合
	// 3个普通僵尸（3×1=3）：应该通过
	if !engine.ValidateWaveCapacity([]string{"basic", "basic", "basic"}, capacity) {
		t.Error("3 basic zombies should pass capacity 3")
	}

	// 1个路障+1个普通（2+1=3）：应该通过
	if !engine.ValidateWaveCapacity([]string{"conehead", "basic"}, capacity) {
		t.Error("1 conehead + 1 basic should pass capacity 3")
	}

	// 1个铁桶（4）：应该失败
	if engine.ValidateWaveCapacity([]string{"buckethead"}, capacity) {
		t.Error("1 buckethead (level 4) should fail capacity 3")
	}
}
