package game

import (
	"testing"

	"github.com/decker502/pvz/pkg/config"
)

// TestGetCurrentWave_FirstWaveWithDelay 测试第一波带 delay 参数的情况
func TestGetCurrentWave_FirstWaveWithDelay(t *testing.T) {
	// 重置单例状态
	globalGameState = nil
	gs := GetGameState()

	// 创建测试关卡配置
	levelConfig := &config.LevelConfig{
		ID:   "test-delay",
		Name: "测试延迟",
		Waves: []config.WaveConfig{
			{
				Delay: 20.0, // 第一波延迟20秒
				Zombies: []config.ZombieGroup{
					{Type: "basic", Lanes: []int{3}, Count: 1, SpawnInterval: 0},
				},
			},
			{
				MinDelay: 5.0, // 第二波延迟5秒
				Zombies: []config.ZombieGroup{
					{Type: "basic", Lanes: []int{3}, Count: 1, SpawnInterval: 0},
				},
			},
		},
		EnabledLanes:    []int{3},
		AvailablePlants: []string{"peashooter"},
	}

	// 加载关卡
	gs.LoadLevel(levelConfig)

	// 测试1: 游戏开始时(LevelTime=0)，第一波不应触发
	gs.LevelTime = 0
	wave := gs.GetCurrentWave()
	if wave != -1 {
		t.Errorf("LevelTime=0: 期望 wave=-1 (等待中), 实际 wave=%d", wave)
	}

	// 测试2: 时间<20秒，第一波不应触发
	gs.LevelTime = 10.0
	wave = gs.GetCurrentWave()
	if wave != -1 {
		t.Errorf("LevelTime=10s: 期望 wave=-1 (等待中), 实际 wave=%d", wave)
	}

	// 测试3: 时间>=20秒，第一波应该触发
	gs.LevelTime = 20.0
	wave = gs.GetCurrentWave()
	if wave != 0 {
		t.Errorf("LevelTime=20s: 期望 wave=0 (第一波触发), 实际 wave=%d", wave)
	}

	// 测试4: 时间>20秒，第一波应该触发
	gs.LevelTime = 25.0
	wave = gs.GetCurrentWave()
	if wave != 0 {
		t.Errorf("LevelTime=25s: 期望 wave=0 (第一波仍可触发), 实际 wave=%d", wave)
	}

	// 标记第一波已生成
	gs.MarkWaveSpawned(0)

	// 测试5: 第一波已生成后，应等待僵尸消灭
	gs.LevelTime = 30.0
	wave = gs.GetCurrentWave()
	if wave != -1 {
		t.Errorf("第一波已生成，场上有僵尸: 期望 wave=-1 (等待中), 实际 wave=%d", wave)
	}
}

// TestGetCurrentWave_FirstWaveNoDelay 测试第一波无 delay 参数的情况（向后兼容）
func TestGetCurrentWave_FirstWaveNoDelay(t *testing.T) {
	// 重置单例状态
	globalGameState = nil
	gs := GetGameState()

	// 创建测试关卡配置（无 delay）
	levelConfig := &config.LevelConfig{
		ID:   "test-no-delay",
		Name: "测试无延迟",
		Waves: []config.WaveConfig{
			{
				Delay: 0, // 无延迟
				Zombies: []config.ZombieGroup{
					{Type: "basic", Lanes: []int{3}, Count: 1, SpawnInterval: 0},
				},
			},
		},
		EnabledLanes:    []int{3},
		AvailablePlants: []string{"peashooter"},
	}

	// 加载关卡
	gs.LoadLevel(levelConfig)

	// 测试: 游戏开始时(LevelTime=0)，第一波应该立即触发（向后兼容）
	gs.LevelTime = 0
	wave := gs.GetCurrentWave()
	if wave != 0 {
		t.Errorf("LevelTime=0, Delay=0: 期望 wave=0 (立即触发), 实际 wave=%d", wave)
	}
}

// TestGetCurrentWave_MultipleWavesWithDelay 测试多个波次的延迟机制
func TestGetCurrentWave_MultipleWavesWithDelay(t *testing.T) {
	// 重置单例状态
	globalGameState = nil
	gs := GetGameState()

	// 创建测试关卡配置
	levelConfig := &config.LevelConfig{
		ID:   "test-multi-delay",
		Name: "测试多波延迟",
		Waves: []config.WaveConfig{
			{
				Delay: 20.0, // 第一波延迟20秒
				Zombies: []config.ZombieGroup{
					{Type: "basic", Lanes: []int{3}, Count: 1, SpawnInterval: 0},
				},
			},
			{
				MinDelay: 5.0, // 第二波：上一波消灭后延迟5秒
				Zombies: []config.ZombieGroup{
					{Type: "basic", Lanes: []int{3}, Count: 1, SpawnInterval: 0},
				},
			},
			{
				MinDelay: 10.0, // 第三波：上一波消灭后延迟10秒
				Zombies: []config.ZombieGroup{
					{Type: "basic", Lanes: []int{3}, Count: 1, SpawnInterval: 0},
				},
			},
		},
		EnabledLanes:    []int{3},
		AvailablePlants: []string{"peashooter"},
	}

	// 加载关卡
	gs.LoadLevel(levelConfig)

	// === 第一波测试 ===
	gs.LevelTime = 20.0
	wave := gs.GetCurrentWave()
	if wave != 0 {
		t.Errorf("第一波: 期望 wave=0, 实际 wave=%d", wave)
	}
	gs.MarkWaveSpawned(0)
	gs.IncrementZombiesSpawned(1) // 模拟1个僵尸激活

	// === 第二波测试 ===
	// 场上有僵尸，应该等待
	gs.LevelTime = 25.0
	wave = gs.GetCurrentWave()
	if wave != -1 {
		t.Errorf("第二波等待(场上有僵尸): 期望 wave=-1, 实际 wave=%d", wave)
	}

	// 消灭僵尸
	gs.IncrementZombiesKilled()
	gs.LevelTime = 26.0
	wave = gs.GetCurrentWave()
	if wave != -1 {
		t.Errorf("第二波等待(刚消灭，进入MinDelay): 期望 wave=-1, 实际 wave=%d", wave)
	}

	// MinDelay=5秒后，应该触发第二波
	gs.LevelTime = 31.0 // 26 + 5 = 31
	wave = gs.GetCurrentWave()
	if wave != 1 {
		t.Errorf("第二波触发(MinDelay已过): 期望 wave=1, 实际 wave=%d", wave)
	}
	gs.MarkWaveSpawned(1)
	gs.IncrementZombiesSpawned(1)

	// === 第三波测试 ===
	gs.IncrementZombiesKilled()
	gs.LevelTime = 32.0
	wave = gs.GetCurrentWave()
	if wave != -1 {
		t.Errorf("第三波等待(刚消灭，进入MinDelay=10s): 期望 wave=-1, 实际 wave=%d", wave)
	}

	// MinDelay=10秒后，应该触发第三波
	gs.LevelTime = 42.0 // 32 + 10 = 42
	wave = gs.GetCurrentWave()
	if wave != 2 {
		t.Errorf("第三波触发(MinDelay已过): 期望 wave=2, 实际 wave=%d", wave)
	}
}
