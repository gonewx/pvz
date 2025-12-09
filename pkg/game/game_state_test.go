package game

import (
	"math"
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
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
// 默认值为50，加载关卡后会被 levelConfig.InitialSun 覆盖
func TestGameStateInitialValue(t *testing.T) {
	// 重置全局状态以测试初始化
	globalGameState = nil
	gs := GetGameState()

	if gs.Sun != 50 {
		t.Errorf("Expected initial sun to be 50, got %d", gs.Sun)
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
			{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
			{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 2, Count: 2}}},
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
		Waves: []config.WaveConfig{{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}}},
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
// Story 17.6: 波次计时由 WaveTimingSystem 管理
// GetCurrentWave 简化为后备逻辑：场上无僵尸时立即触发下一波
func TestGetCurrentWave(t *testing.T) {
	gs := GetGameState()
	levelConfig := &config.LevelConfig{
		ID:   "test-1",
		Name: "Test Level",
		Waves: []config.WaveConfig{
			{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
			{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 2, Count: 2}}},
			{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 3, Count: 1}}},
		},
	}
	gs.LoadLevel(levelConfig)

	// 第一波：游戏开始时立即触发
	wave := gs.GetCurrentWave()
	if wave != 0 {
		t.Errorf("Expected wave 0 at game start, got %d", wave)
	}

	// 标记第一波已生成（1个僵尸）
	gs.MarkWaveSpawned(0)
	gs.IncrementZombiesSpawned(1)

	// 第一波生成后，场上还有僵尸（未消灭），不触发第二波
	wave = gs.GetCurrentWave()
	if wave != -1 {
		t.Errorf("Expected wave -1 (zombies still alive), got %d", wave)
	}

	// 消灭第一波所有僵尸
	gs.IncrementZombiesKilled()

	// 场上无僵尸，立即触发第二波（无延迟）
	wave = gs.GetCurrentWave()
	if wave != 1 {
		t.Errorf("Expected wave 1 (no zombies on field), got %d", wave)
	}

	// 标记第二波已生成（2个僵尸）
	gs.MarkWaveSpawned(1)
	gs.IncrementZombiesSpawned(2)

	// 场上有僵尸，不触发第三波
	wave = gs.GetCurrentWave()
	if wave != -1 {
		t.Errorf("Expected wave -1 (zombies on field), got %d", wave)
	}

	// 消灭第二波所有僵尸
	gs.IncrementZombiesKilled()
	gs.IncrementZombiesKilled()

	// 场上无僵尸，立即触发第三波
	wave = gs.GetCurrentWave()
	if wave != 2 {
		t.Errorf("Expected wave 2 (no zombies on field), got %d", wave)
	}

	// 标记第三波已生成
	gs.MarkWaveSpawned(2)
	gs.IncrementZombiesSpawned(1)

	// 所有波次已生成，返回 -1
	wave = gs.GetCurrentWave()
	if wave != -1 {
		t.Errorf("Expected wave -1 (all waves spawned), got %d", wave)
	}
}

// TestMarkWaveSpawned 测试标记波次已生成
func TestMarkWaveSpawned(t *testing.T) {
	gs := GetGameState()
	levelConfig := &config.LevelConfig{
		ID:   "test-1",
		Name: "Test Level",
		Waves: []config.WaveConfig{
			{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
			{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 2, Count: 2}}},
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
			{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
			{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 2, Count: 2}}},
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
		Waves: []config.WaveConfig{{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}}},
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
		Waves: []config.WaveConfig{{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}}},
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
			{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 2}}},
			{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 2, Count: 3}}},
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
		Waves: []config.WaveConfig{{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}}},
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
			{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
			{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 2, Count: 2}}},
			{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 3, Count: 1}}},
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

// TestLoadLevel_InitialSun 测试 LoadLevel 是否正确应用初始阳光值 (Story 8.2 QA改进)
func TestLoadLevel_InitialSun(t *testing.T) {
	gs := GetGameState()

	// 测试不同的初始阳光值
	testCases := []struct {
		name       string
		initialSun int
	}{
		{"教学关卡 150 阳光", 150},
		{"标准关卡 50 阳光", 50},
		{"挑战关卡 0 阳光", 0},
		{"特殊关卡 9999 阳光", 9999},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			levelConfig := &config.LevelConfig{
				ID:   "test-level",
				Name: tc.name,
				Waves: []config.WaveConfig{
					{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
				},
				InitialSun: tc.initialSun,
			}

			// 加载关卡（应该覆盖当前阳光值）
			gs.LoadLevel(levelConfig)

			// 验证阳光值是否被正确设置
			if gs.Sun != tc.initialSun {
				t.Errorf("%s: Expected sun %d, got %d", tc.name, tc.initialSun, gs.Sun)
			}
		})
	}
}

// TestLoadLevel1_1RealConfig 测试加载真实的 1-1 关卡配置
func TestLoadLevel1_1RealConfig(t *testing.T) {
	// 加载真实的 1-1 关卡配置
	levelConfig, err := config.LoadLevelConfig("../../data/levels/level-1-1.yaml")
	if err != nil {
		t.Fatalf("Failed to load level-1-1.yaml: %v", err)
	}

	// 重置 GameState
	globalGameState = nil
	gs := GetGameState()

	// 验证单例初始化的默认阳光值（应该是50）
	if gs.Sun != 50 {
		t.Errorf("Expected default sun 50 before loading level, got %d", gs.Sun)
	}

	// 加载关卡
	gs.LoadLevel(levelConfig)

	// 验证阳光值被关卡配置覆盖（应该是150）
	if gs.Sun != 150 {
		t.Errorf("Expected sun 150 from level config, got %d", gs.Sun)
	}

	// 验证其他关卡配置也正确加载
	if gs.CurrentLevel.ID != "1-1" {
		t.Errorf("Expected level ID '1-1', got '%s'", gs.CurrentLevel.ID)
	}

	if len(gs.CurrentLevel.EnabledLanes) != 1 || gs.CurrentLevel.EnabledLanes[0] != 3 {
		t.Errorf("Expected enabled lanes [3], got %v", gs.CurrentLevel.EnabledLanes)
	}

	if len(gs.CurrentLevel.AvailablePlants) != 1 || gs.CurrentLevel.AvailablePlants[0] != "peashooter" {
		t.Errorf("Expected available plants [peashooter], got %v", gs.CurrentLevel.AvailablePlants)
	}

	t.Logf("✓ Successfully loaded level 1-1 with initialSun=%d", gs.Sun)
}

// TestSetPaused 测试设置暂停状态 (Story 10.1)
func TestSetPaused(t *testing.T) {
	gs := GetGameState()

	// 初始状态应该是未暂停
	if gs.IsPaused {
		t.Error("Expected IsPaused to be false initially")
	}

	// 设置为暂停
	gs.SetPaused(true)
	if !gs.IsPaused {
		t.Error("Expected IsPaused to be true after SetPaused(true)")
	}

	// 设置为恢复
	gs.SetPaused(false)
	if gs.IsPaused {
		t.Error("Expected IsPaused to be false after SetPaused(false)")
	}
}

// TestTogglePause 测试切换暂停状态 (Story 10.1)
func TestTogglePause(t *testing.T) {
	gs := GetGameState()

	// 初始状态为未暂停
	gs.IsPaused = false

	// 第一次切换，应该变为暂停
	gs.TogglePause()
	if !gs.IsPaused {
		t.Error("Expected IsPaused to be true after first toggle")
	}

	// 第二次切换，应该变为恢复
	gs.TogglePause()
	if gs.IsPaused {
		t.Error("Expected IsPaused to be false after second toggle")
	}

	// 第三次切换，应该再次变为暂停
	gs.TogglePause()
	if !gs.IsPaused {
		t.Error("Expected IsPaused to be true after third toggle")
	}
}

// TestPauseStateInitial 测试暂停状态的初始值 (Story 10.1)
func TestPauseStateInitial(t *testing.T) {
	// 重置全局状态以测试初始化
	globalGameState = nil
	gs := GetGameState()

	// 初始状态应该是未暂停
	if gs.IsPaused {
		t.Error("Expected IsPaused to be false on initialization")
	}
}

// TestPauseStateIndependent 测试暂停状态独立于其他状态 (Story 10.1)
func TestPauseStateIndependent(t *testing.T) {
	gs := GetGameState()

	// 设置一些其他游戏状态
	gs.Sun = 500
	gs.IsPlantingMode = true
	gs.SelectedPlantType = components.PlantPeashooter

	// 暂停游戏
	gs.SetPaused(true)

	// 验证其他状态没有受到影响
	if gs.Sun != 500 {
		t.Errorf("Expected Sun 500, got %d", gs.Sun)
	}
	if !gs.IsPlantingMode {
		t.Error("Expected IsPlantingMode to remain true")
	}
	if gs.SelectedPlantType != components.PlantPeashooter {
		t.Errorf("Expected SelectedPlantType to remain PlantPeashooter, got %v", gs.SelectedPlantType)
	}
	if !gs.IsPaused {
		t.Error("Expected IsPaused to be true")
	}
}

// TestTriggerSunFlash 测试触发阳光闪烁 (Story 10.8)
func TestTriggerSunFlash(t *testing.T) {
	gs := GetGameState()

	// 初始状态：闪烁计时器应该为 0
	if gs.SunFlashTimer != 0 {
		t.Errorf("Expected SunFlashTimer 0 initially, got %f", gs.SunFlashTimer)
	}

	// 触发闪烁
	gs.TriggerSunFlash()

	// 验证闪烁计时器被设置为持续时间
	if gs.SunFlashTimer != gs.SunFlashDuration {
		t.Errorf("Expected SunFlashTimer %f, got %f", gs.SunFlashDuration, gs.SunFlashTimer)
	}
}

// TestUpdateSunFlash 测试更新闪烁计时器 (Story 10.8)
func TestUpdateSunFlash(t *testing.T) {
	gs := GetGameState()

	// 设置初始闪烁计时器
	gs.SunFlashTimer = 1.0

	// 更新 0.3 秒
	gs.UpdateSunFlash(0.3)
	expected := 0.7
	if math.Abs(gs.SunFlashTimer-expected) > 0.0001 {
		t.Errorf("Expected SunFlashTimer %.4f, got %.4f", expected, gs.SunFlashTimer)
	}

	// 更新 0.5 秒
	gs.UpdateSunFlash(0.5)
	expected = 0.2
	if math.Abs(gs.SunFlashTimer-expected) > 0.0001 {
		t.Errorf("Expected SunFlashTimer %.4f, got %.4f", expected, gs.SunFlashTimer)
	}

	// 更新超过剩余时间（应该停止在 0）
	gs.UpdateSunFlash(0.5)
	if gs.SunFlashTimer != 0.0 {
		t.Errorf("Expected SunFlashTimer 0.0, got %f", gs.SunFlashTimer)
	}
}

// TestUpdateSunFlash_NoNegative 测试闪烁计时器不会变为负数 (Story 10.8)
func TestUpdateSunFlash_NoNegative(t *testing.T) {
	gs := GetGameState()

	// 设置初始闪烁计时器
	gs.SunFlashTimer = 0.1

	// 更新超过剩余时间
	gs.UpdateSunFlash(0.5)

	// 验证计时器不会变为负数
	if gs.SunFlashTimer < 0 {
		t.Errorf("Expected SunFlashTimer >= 0, got %f", gs.SunFlashTimer)
	}
	if gs.SunFlashTimer != 0.0 {
		t.Errorf("Expected SunFlashTimer 0.0, got %f", gs.SunFlashTimer)
	}
}

// TestSunFlashInitialValues 测试闪烁相关字段的初始值 (Story 10.8)
func TestSunFlashInitialValues(t *testing.T) {
	// 重置全局状态以测试初始化
	globalGameState = nil
	gs := GetGameState()

	// 验证默认值
	if gs.SunFlashTimer != 0.0 {
		t.Errorf("Expected SunFlashTimer 0.0, got %f", gs.SunFlashTimer)
	}
	if gs.SunFlashCycle != 0.3 {
		t.Errorf("Expected SunFlashCycle 0.3, got %f", gs.SunFlashCycle)
	}
	if gs.SunFlashDuration != 1.0 {
		t.Errorf("Expected SunFlashDuration 1.0, got %f", gs.SunFlashDuration)
	}
}

// TestSunFlashMultipleTriggers 测试多次触发闪烁（重置计时器）(Story 10.8)
func TestSunFlashMultipleTriggers(t *testing.T) {
	gs := GetGameState()

	// 第一次触发
	gs.TriggerSunFlash()
	if gs.SunFlashTimer != 1.0 {
		t.Errorf("Expected SunFlashTimer 1.0 after first trigger, got %f", gs.SunFlashTimer)
	}

	// 更新一段时间
	gs.UpdateSunFlash(0.6)
	if gs.SunFlashTimer != 0.4 {
		t.Errorf("Expected SunFlashTimer 0.4 after update, got %f", gs.SunFlashTimer)
	}

	// 再次触发（应该重置为 1.0）
	gs.TriggerSunFlash()
	if gs.SunFlashTimer != 1.0 {
		t.Errorf("Expected SunFlashTimer 1.0 after second trigger, got %f", gs.SunFlashTimer)
	}
}
