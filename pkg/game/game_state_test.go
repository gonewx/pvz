package game

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
)

// TestGameStateSingleton 测试单例模式是否正确实现
// 验证多次调用 GetGameState() 返回同一个实例
func TestGameStateSingleton(t *testing.T) {
	gs1 := GetGameState()
	gs2 := GetGameState()

	if gs1 != gs2 {
		t.Error("GetGameState() should return the same instance")
	}
}

// TestGameStateInitialValue 测试初始阳光值
// 注意：当前为测试设置为500，原版游戏为50
func TestGameStateInitialValue(t *testing.T) {
	// 重置全局状态以测试初始化
	globalGameState = nil
	gs := GetGameState()

	if gs.Sun != 500 {
		t.Errorf("Expected initial sun to be 500, got %d", gs.Sun)
	}
}

// TestGetSun 测试 GetSun 方法是否正确返回阳光值
func TestGetSun(t *testing.T) {
	gs := GetGameState()
	gs.Sun = 100

	if gs.GetSun() != 100 {
		t.Errorf("Expected GetSun() to return 100, got %d", gs.GetSun())
	}
}

// TestAddSun 测试 AddSun 方法是否正确增加阳光
func TestAddSun(t *testing.T) {
	gs := GetGameState()
	gs.Sun = 100 // 设置初始值

	gs.AddSun(50)
	if gs.Sun != 150 {
		t.Errorf("Expected 150, got %d", gs.Sun)
	}
}

// TestAddSunCap 测试 AddSun 是否正确限制阳光上限为9990
func TestAddSunCap(t *testing.T) {
	gs := GetGameState()
	gs.Sun = 9980

	gs.AddSun(50)
	if gs.Sun != 9990 {
		t.Errorf("Expected 9990 (capped), got %d", gs.Sun)
	}
}

// TestAddSunExceedsCap 测试超过上限的情况
func TestAddSunExceedsCap(t *testing.T) {
	gs := GetGameState()
	gs.Sun = 9990

	gs.AddSun(100) // 尝试超过上限
	if gs.Sun != 9990 {
		t.Errorf("Expected 9990 (capped), got %d", gs.Sun)
	}
}

// TestSpendSunSuccess 测试阳光充足时 SpendSun 成功扣除
func TestSpendSunSuccess(t *testing.T) {
	gs := GetGameState()
	gs.Sun = 100

	success := gs.SpendSun(50)
	if !success {
		t.Error("Expected SpendSun to succeed")
	}
	if gs.Sun != 50 {
		t.Errorf("Expected 50, got %d", gs.Sun)
	}
}

// TestSpendSunFailure 测试阳光不足时 SpendSun 失败且阳光不变
func TestSpendSunFailure(t *testing.T) {
	gs := GetGameState()
	gs.Sun = 30

	success := gs.SpendSun(50)
	if success {
		t.Error("Expected SpendSun to fail")
	}
	if gs.Sun != 30 {
		t.Errorf("Expected sun to remain 30, got %d", gs.Sun)
	}
}

// TestSpendSunExactAmount 测试恰好花费全部阳光
func TestSpendSunExactAmount(t *testing.T) {
	gs := GetGameState()
	gs.Sun = 50

	success := gs.SpendSun(50)
	if !success {
		t.Error("Expected SpendSun to succeed")
	}
	if gs.Sun != 0 {
		t.Errorf("Expected 0, got %d", gs.Sun)
	}
}

// TestSpendSunZeroSun 测试阳光为0时无法扣除
func TestSpendSunZeroSun(t *testing.T) {
	gs := GetGameState()
	gs.Sun = 0

	success := gs.SpendSun(25)
	if success {
		t.Error("Expected SpendSun to fail when sun is 0")
	}
	if gs.Sun != 0 {
		t.Errorf("Expected sun to remain 0, got %d", gs.Sun)
	}
}

// TestEnterPlantingMode 测试进入种植模式
// 验证 IsPlantingMode 设置为 true，SelectedPlantType 正确设置
func TestEnterPlantingMode(t *testing.T) {
	gs := GetGameState()
	gs.IsPlantingMode = false // 初始状态

	gs.EnterPlantingMode(components.PlantSunflower)

	if !gs.IsPlantingMode {
		t.Error("Expected IsPlantingMode to be true")
	}
	if gs.SelectedPlantType != components.PlantSunflower {
		t.Errorf("Expected SelectedPlantType to be PlantSunflower, got %v", gs.SelectedPlantType)
	}
}

// TestExitPlantingMode 测试退出种植模式
// 验证 IsPlantingMode 设置为 false
func TestExitPlantingMode(t *testing.T) {
	gs := GetGameState()
	gs.IsPlantingMode = true // 先进入种植模式
	gs.SelectedPlantType = components.PlantPeashooter

	gs.ExitPlantingMode()

	if gs.IsPlantingMode {
		t.Error("Expected IsPlantingMode to be false")
	}
	// SelectedPlantType 保持不变（可选行为）
	if gs.SelectedPlantType != components.PlantPeashooter {
		t.Errorf("Expected SelectedPlantType to remain PlantPeashooter, got %v", gs.SelectedPlantType)
	}
}

// TestGetPlantingMode 测试获取种植模式状态
// 验证正确返回当前状态和选择的植物类型
func TestGetPlantingMode(t *testing.T) {
	gs := GetGameState()
	gs.IsPlantingMode = true
	gs.SelectedPlantType = components.PlantSunflower

	isPlanting, plantType := gs.GetPlantingMode()

	if !isPlanting {
		t.Error("Expected isPlanting to be true")
	}
	if plantType != components.PlantSunflower {
		t.Errorf("Expected plantType to be PlantSunflower, got %v", plantType)
	}
}

// TestPlantingModeToggle 测试种植模式切换
// 验证可以正确进入和退出种植模式多次
func TestPlantingModeToggle(t *testing.T) {
	gs := GetGameState()

	// 第一次进入
	gs.EnterPlantingMode(components.PlantSunflower)
	if !gs.IsPlantingMode {
		t.Error("Expected IsPlantingMode to be true after first enter")
	}

	// 退出
	gs.ExitPlantingMode()
	if gs.IsPlantingMode {
		t.Error("Expected IsPlantingMode to be false after exit")
	}

	// 第二次进入（不同植物类型）
	gs.EnterPlantingMode(components.PlantPeashooter)
	if !gs.IsPlantingMode {
		t.Error("Expected IsPlantingMode to be true after second enter")
	}
	if gs.SelectedPlantType != components.PlantPeashooter {
		t.Errorf("Expected SelectedPlantType to be PlantPeashooter, got %v", gs.SelectedPlantType)
	}
}

// TestLoadLevel 测试加载关卡配置
func TestLoadLevel(t *testing.T) {
	gs := GetGameState()

	// 创建测试关卡配置
	levelConfig := &config.LevelConfig{
		ID:   "test-1",
		Name: "Test Level",
		Waves: []config.WaveConfig{
			{Time: 10, Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
			{Time: 30, Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 2, Count: 2}}},
		},
	}

	// 加载关卡
	gs.LoadLevel(levelConfig)

	// 验证状态
	if gs.CurrentLevel != levelConfig {
		t.Error("CurrentLevel not set correctly")
	}
	if gs.LevelTime != 0 {
		t.Errorf("Expected LevelTime 0, got %f", gs.LevelTime)
	}
	if gs.CurrentWaveIndex != 0 {
		t.Errorf("Expected CurrentWaveIndex 0, got %d", gs.CurrentWaveIndex)
	}
	if len(gs.SpawnedWaves) != 2 {
		t.Errorf("Expected SpawnedWaves length 2, got %d", len(gs.SpawnedWaves))
	}
	if gs.TotalZombiesSpawned != 0 {
		t.Errorf("Expected TotalZombiesSpawned 0, got %d", gs.TotalZombiesSpawned)
	}
	if gs.ZombiesKilled != 0 {
		t.Errorf("Expected ZombiesKilled 0, got %d", gs.ZombiesKilled)
	}
	if gs.IsLevelComplete {
		t.Error("Expected IsLevelComplete false")
	}
	if gs.IsGameOver {
		t.Error("Expected IsGameOver false")
	}
	if gs.GameResult != "" {
		t.Errorf("Expected GameResult empty, got '%s'", gs.GameResult)
	}
}

// TestUpdateLevelTime 测试时间更新
func TestUpdateLevelTime(t *testing.T) {
	gs := GetGameState()
	levelConfig := &config.LevelConfig{
		ID:    "test-1",
		Name:  "Test Level",
		Waves: []config.WaveConfig{{Time: 10, Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}}},
	}
	gs.LoadLevel(levelConfig)

	// 更新时间
	gs.UpdateLevelTime(1.5)
	if gs.LevelTime != 1.5 {
		t.Errorf("Expected LevelTime 1.5, got %f", gs.LevelTime)
	}

	gs.UpdateLevelTime(2.3)
	if gs.LevelTime != 3.8 {
		t.Errorf("Expected LevelTime 3.8, got %f", gs.LevelTime)
	}
}

// TestGetCurrentWave 测试获取当前波次
func TestGetCurrentWave(t *testing.T) {
	gs := GetGameState()
	levelConfig := &config.LevelConfig{
		ID:   "test-1",
		Name: "Test Level",
		Waves: []config.WaveConfig{
			{Time: 10, Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
			{Time: 30, Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 2, Count: 2}}},
			{Time: 50, Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 3, Count: 1}}},
		},
	}
	gs.LoadLevel(levelConfig)

	// 时间未到任何波次
	gs.LevelTime = 5
	wave := gs.GetCurrentWave()
	if wave != -1 {
		t.Errorf("Expected wave -1 at time 5, got %d", wave)
	}

	// 第一波时间到
	gs.LevelTime = 10
	wave = gs.GetCurrentWave()
	if wave != 0 {
		t.Errorf("Expected wave 0 at time 10, got %d", wave)
	}

	// 标记第一波已生成
	gs.MarkWaveSpawned(0)

	// 第一波已生成，应该不再返回
	gs.LevelTime = 15
	wave = gs.GetCurrentWave()
	if wave != -1 {
		t.Errorf("Expected wave -1 after first wave spawned, got %d", wave)
	}

	// 第二波时间到
	gs.LevelTime = 30
	wave = gs.GetCurrentWave()
	if wave != 1 {
		t.Errorf("Expected wave 1 at time 30, got %d", wave)
	}

	// 标记第二波已生成
	gs.MarkWaveSpawned(1)

	// 第三波时间到
	gs.LevelTime = 50
	wave = gs.GetCurrentWave()
	if wave != 2 {
		t.Errorf("Expected wave 2 at time 50, got %d", wave)
	}
}

// TestMarkWaveSpawned 测试标记波次已生成
func TestMarkWaveSpawned(t *testing.T) {
	gs := GetGameState()
	levelConfig := &config.LevelConfig{
		ID:   "test-1",
		Name: "Test Level",
		Waves: []config.WaveConfig{
			{Time: 10, Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
			{Time: 30, Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 2, Count: 2}}},
		},
	}
	gs.LoadLevel(levelConfig)

	// 标记第一波
	gs.MarkWaveSpawned(0)
	if !gs.SpawnedWaves[0] {
		t.Error("Expected wave 0 to be marked as spawned")
	}
	if gs.CurrentWaveIndex != 1 {
		t.Errorf("Expected CurrentWaveIndex 1, got %d", gs.CurrentWaveIndex)
	}

	// 标记第二波
	gs.MarkWaveSpawned(1)
	if !gs.SpawnedWaves[1] {
		t.Error("Expected wave 1 to be marked as spawned")
	}
	if gs.CurrentWaveIndex != 2 {
		t.Errorf("Expected CurrentWaveIndex 2, got %d", gs.CurrentWaveIndex)
	}
}

// TestIsWaveSpawned 测试检查波次是否已生成
func TestIsWaveSpawned(t *testing.T) {
	gs := GetGameState()
	levelConfig := &config.LevelConfig{
		ID:   "test-1",
		Name: "Test Level",
		Waves: []config.WaveConfig{
			{Time: 10, Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
			{Time: 30, Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 2, Count: 2}}},
		},
	}
	gs.LoadLevel(levelConfig)

	// 初始状态
	if gs.IsWaveSpawned(0) {
		t.Error("Expected wave 0 not spawned initially")
	}
	if gs.IsWaveSpawned(1) {
		t.Error("Expected wave 1 not spawned initially")
	}

	// 标记第一波
	gs.MarkWaveSpawned(0)
	if !gs.IsWaveSpawned(0) {
		t.Error("Expected wave 0 spawned after marking")
	}
	if gs.IsWaveSpawned(1) {
		t.Error("Expected wave 1 not spawned yet")
	}

	// 边界测试
	if gs.IsWaveSpawned(-1) {
		t.Error("Expected negative index to return false")
	}
	if gs.IsWaveSpawned(10) {
		t.Error("Expected out-of-bounds index to return false")
	}
}

// TestIncrementZombiesSpawned 测试增加已生成僵尸计数
func TestIncrementZombiesSpawned(t *testing.T) {
	gs := GetGameState()
	levelConfig := &config.LevelConfig{
		ID:    "test-1",
		Name:  "Test Level",
		Waves: []config.WaveConfig{{Time: 10, Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}}},
	}
	gs.LoadLevel(levelConfig)

	// 初始状态
	if gs.TotalZombiesSpawned != 0 {
		t.Errorf("Expected TotalZombiesSpawned 0, got %d", gs.TotalZombiesSpawned)
	}

	// 增加计数
	gs.IncrementZombiesSpawned(1)
	if gs.TotalZombiesSpawned != 1 {
		t.Errorf("Expected TotalZombiesSpawned 1, got %d", gs.TotalZombiesSpawned)
	}

	gs.IncrementZombiesSpawned(3)
	if gs.TotalZombiesSpawned != 4 {
		t.Errorf("Expected TotalZombiesSpawned 4, got %d", gs.TotalZombiesSpawned)
	}
}

// TestIncrementZombiesKilled 测试增加已消灭僵尸计数
func TestIncrementZombiesKilled(t *testing.T) {
	gs := GetGameState()
	levelConfig := &config.LevelConfig{
		ID:    "test-1",
		Name:  "Test Level",
		Waves: []config.WaveConfig{{Time: 10, Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}}},
	}
	gs.LoadLevel(levelConfig)

	// 初始状态
	if gs.ZombiesKilled != 0 {
		t.Errorf("Expected ZombiesKilled 0, got %d", gs.ZombiesKilled)
	}

	// 增加计数
	gs.IncrementZombiesKilled()
	if gs.ZombiesKilled != 1 {
		t.Errorf("Expected ZombiesKilled 1, got %d", gs.ZombiesKilled)
	}

	gs.IncrementZombiesKilled()
	gs.IncrementZombiesKilled()
	if gs.ZombiesKilled != 3 {
		t.Errorf("Expected ZombiesKilled 3, got %d", gs.ZombiesKilled)
	}
}

// TestCheckVictory 测试胜利条件检测
func TestCheckVictory(t *testing.T) {
	gs := GetGameState()
	levelConfig := &config.LevelConfig{
		ID:   "test-1",
		Name: "Test Level",
		Waves: []config.WaveConfig{
			{Time: 10, Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 2}}},
			{Time: 30, Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 2, Count: 3}}},
		},
	}
	gs.LoadLevel(levelConfig)

	// 初始状态，未胜利
	if gs.CheckVictory() {
		t.Error("Expected no victory initially")
	}

	// 标记所有波次已生成
	gs.MarkWaveSpawned(0)
	gs.MarkWaveSpawned(1)

	// 生成了5个僵尸
	gs.IncrementZombiesSpawned(5)

	// 还未消灭所有僵尸
	if gs.CheckVictory() {
		t.Error("Expected no victory with zombies remaining")
	}

	// 消灭3个僵尸
	gs.IncrementZombiesKilled()
	gs.IncrementZombiesKilled()
	gs.IncrementZombiesKilled()
	if gs.CheckVictory() {
		t.Error("Expected no victory with 2 zombies remaining")
	}

	// 消灭剩余2个僵尸
	gs.IncrementZombiesKilled()
	gs.IncrementZombiesKilled()
	if !gs.CheckVictory() {
		t.Error("Expected victory after killing all zombies")
	}
}

// TestSetGameResult 测试设置游戏结果
func TestSetGameResult(t *testing.T) {
	gs := GetGameState()
	levelConfig := &config.LevelConfig{
		ID:    "test-1",
		Name:  "Test Level",
		Waves: []config.WaveConfig{{Time: 10, Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}}},
	}

	t.Run("win result", func(t *testing.T) {
		gs.LoadLevel(levelConfig)
		gs.SetGameResult("win")
		if gs.GameResult != "win" {
			t.Errorf("Expected GameResult 'win', got '%s'", gs.GameResult)
		}
		if !gs.IsGameOver {
			t.Error("Expected IsGameOver true")
		}
		if !gs.IsLevelComplete {
			t.Error("Expected IsLevelComplete true for win")
		}
	})

	t.Run("lose result", func(t *testing.T) {
		gs.LoadLevel(levelConfig)
		gs.SetGameResult("lose")
		if gs.GameResult != "lose" {
			t.Errorf("Expected GameResult 'lose', got '%s'", gs.GameResult)
		}
		if !gs.IsGameOver {
			t.Error("Expected IsGameOver true")
		}
		if gs.IsLevelComplete {
			t.Error("Expected IsLevelComplete false for lose")
		}
	})
}

// TestGetLevelProgress 测试获取关卡进度
func TestGetLevelProgress(t *testing.T) {
	gs := GetGameState()
	levelConfig := &config.LevelConfig{
		ID:   "test-1",
		Name: "Test Level",
		Waves: []config.WaveConfig{
			{Time: 10, Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
			{Time: 30, Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 2, Count: 2}}},
			{Time: 50, Zombies: []config.ZombieSpawn{{Type: "basic", Lane: 3, Count: 1}}},
		},
	}
	gs.LoadLevel(levelConfig)

	// 初始进度
	current, total := gs.GetLevelProgress()
	if current != 0 || total != 3 {
		t.Errorf("Expected progress 0/3, got %d/%d", current, total)
	}

	// 标记第一波
	gs.MarkWaveSpawned(0)
	current, total = gs.GetLevelProgress()
	if current != 1 || total != 3 {
		t.Errorf("Expected progress 1/3, got %d/%d", current, total)
	}

	// 标记第二波
	gs.MarkWaveSpawned(1)
	current, total = gs.GetLevelProgress()
	if current != 2 || total != 3 {
		t.Errorf("Expected progress 2/3, got %d/%d", current, total)
	}

	// 标记第三波
	gs.MarkWaveSpawned(2)
	current, total = gs.GetLevelProgress()
	if current != 3 || total != 3 {
		t.Errorf("Expected progress 3/3, got %d/%d", current, total)
	}
}
