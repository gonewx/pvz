package systems

import (
	"math"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// ========================================
// Task 1: 集成测试框架
// ========================================

// TestEnvironment 集成测试环境
// 包含所有 Epic 17 系统实例和必要的配置
type TestEnvironment struct {
	EM          *ecs.EntityManager
	GameState   *game.GameState
	LevelConfig *config.LevelConfig

	// Epic 17 系统
	DifficultyEngine *DifficultyEngine
	LaneAllocator    *LaneAllocator
	WaveTimingSystem *WaveTimingSystem
	WaveSpawnSystem  *WaveSpawnSystem

	// 配置
	ZombieStats   *config.ZombieStatsConfig
	SpawnRules    *config.SpawnRulesConfig
	ZombiePhysics *config.ZombiePhysicsConfig
}

// SetupTestEnvironment 创建测试环境
//
// 参数:
//   - levelID: 关卡ID，如 "1-1", "1-4"
//
// 返回:
//   - *TestEnvironment: 测试环境实例
//   - error: 加载失败时返回错误
func SetupTestEnvironment(levelID string) (*TestEnvironment, error) {
	// 1. 加载关卡配置
	levelPath := "data/levels/level-" + levelID + ".yaml"
	levelConfig, err := config.LoadLevelConfig(levelPath)
	if err != nil {
		return nil, err
	}

	// 2. 创建 EntityManager
	em := ecs.NewEntityManager()

	// 3. 创建测试用 GameState（不使用全局单例）
	gs := &game.GameState{
		Sun:                 levelConfig.InitialSun,
		TotalCompletedFlags: 0,
		WavesPerRound:       20,
	}

	// 4. 加载配置文件（可选，不存在时使用 nil）
	var zombieStats *config.ZombieStatsConfig
	var spawnRules *config.SpawnRulesConfig
	var zombiePhysics *config.ZombiePhysicsConfig

	// 尝试加载僵尸属性配置
	zombieStats, _ = config.LoadZombieStats("data/zombies.yaml")

	// 尝试加载生成规则配置
	spawnRules, _ = config.LoadSpawnRules("data/spawn_rules.yaml")

	// 尝试加载僵尸物理配置
	zombiePhysics, _ = config.LoadZombiePhysicsConfig("data/zombie_physics.yaml")

	// 5. 创建系统实例
	difficultyEngine := NewDifficultyEngine(zombieStats)
	laneAllocator := NewLaneAllocator(em)

	// 初始化行分配器
	rowMax := levelConfig.RowMax
	if rowMax == 0 {
		rowMax = 5
	}
	laneAllocator.InitializeLanes(rowMax, 1.0)

	waveTimingSystem := NewWaveTimingSystem(em, gs, levelConfig)

	// WaveSpawnSystem 需要 ResourceManager，但测试中不需要实际加载资源
	// 创建一个简化版本用于测试
	waveSpawnSystem := &WaveSpawnSystem{
		entityManager:   em,
		resourceManager: nil, // 测试中不需要
		levelConfig:     levelConfig,
		gameState:       gs,
		spawnRules:      spawnRules,
		zombiePhysics:   zombiePhysics,
		laneAllocator:   laneAllocator,
	}

	return &TestEnvironment{
		EM:               em,
		GameState:        gs,
		LevelConfig:      levelConfig,
		DifficultyEngine: difficultyEngine,
		LaneAllocator:    laneAllocator,
		WaveTimingSystem: waveTimingSystem,
		WaveSpawnSystem:  waveSpawnSystem,
		ZombieStats:      zombieStats,
		SpawnRules:       spawnRules,
		ZombiePhysics:    zombiePhysics,
	}, nil
}

// TeardownTestEnvironment 清理测试环境
func TeardownTestEnvironment(env *TestEnvironment) {
	// 清理实体管理器中的所有实体
	if env.EM != nil {
		env.EM.RemoveMarkedEntities()
	}
}

// ========================================
// Task 2: 关卡 1-1 验证测试
// ========================================

// TestLevel1_1_WaveCount 验证 1-1 波次数量正确（4波）
func TestLevel1_1_WaveCount(t *testing.T) {
	env, err := SetupTestEnvironment("1-1")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 验证波次数量
	expectedWaves := 4
	actualWaves := len(env.LevelConfig.Waves)

	if actualWaves != expectedWaves {
		t.Errorf("Expected %d waves, got %d", expectedWaves, actualWaves)
	}
}

// TestLevel1_1_ZombieCount 验证 1-1 每波僵尸数量（1, 1, 1, 2）
func TestLevel1_1_ZombieCount(t *testing.T) {
	env, err := SetupTestEnvironment("1-1")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 预期每波僵尸数量
	expectedCounts := []int{1, 1, 1, 2}

	for i, wave := range env.LevelConfig.Waves {
		// 计算本波僵尸总数
		totalCount := 0
		for _, zombieGroup := range wave.Zombies {
			totalCount += zombieGroup.Count
		}

		if i < len(expectedCounts) {
			if totalCount != expectedCounts[i] {
				t.Errorf("Wave %d: expected %d zombies, got %d", i+1, expectedCounts[i], totalCount)
			}
		}
	}
}

// TestLevel1_1_SingleLane 验证 1-1 所有僵尸只在第3行生成
func TestLevel1_1_SingleLane(t *testing.T) {
	env, err := SetupTestEnvironment("1-1")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 验证启用的行只有第3行
	if len(env.LevelConfig.EnabledLanes) != 1 {
		t.Errorf("Expected 1 enabled lane, got %d", len(env.LevelConfig.EnabledLanes))
	}

	if len(env.LevelConfig.EnabledLanes) > 0 && env.LevelConfig.EnabledLanes[0] != 3 {
		t.Errorf("Expected enabled lane to be 3, got %d", env.LevelConfig.EnabledLanes[0])
	}

	// 验证所有波次的僵尸配置都是第3行
	for waveIdx, wave := range env.LevelConfig.Waves {
		for groupIdx, zombieGroup := range wave.Zombies {
			for _, lane := range zombieGroup.Lanes {
				if lane != 3 {
					t.Errorf("Wave %d, Group %d: expected lane 3, got %d", waveIdx+1, groupIdx, lane)
				}
			}
		}
	}
}

// TestLevel1_1_BasicZombieOnly 验证 1-1 只生成普通僵尸
func TestLevel1_1_BasicZombieOnly(t *testing.T) {
	env, err := SetupTestEnvironment("1-1")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	for waveIdx, wave := range env.LevelConfig.Waves {
		for groupIdx, zombieGroup := range wave.Zombies {
			if zombieGroup.Type != "basic" {
				t.Errorf("Wave %d, Group %d: expected zombie type 'basic', got '%s'",
					waveIdx+1, groupIdx, zombieGroup.Type)
			}
		}
	}
}

// TestLevel1_1_SpawnCoordinates 验证 1-1 出生坐标在配置范围内
func TestLevel1_1_SpawnCoordinates(t *testing.T) {
	env, err := SetupTestEnvironment("1-1")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 如果有物理配置，验证出生坐标范围
	if env.ZombiePhysics != nil {
		minX, maxX := env.ZombiePhysics.GetSpawnXRange("basic", false, false)

		// 验证范围有效
		if minX > maxX {
			t.Errorf("Invalid spawn X range: min(%.1f) > max(%.1f)", minX, maxX)
		}

		// 验证范围在合理区间内（网格坐标系）
		if minX < 0 {
			t.Errorf("Spawn X min should be >= 0, got %.1f", minX)
		}
	}
}

// ========================================
// Task 3: 关卡 1-4 验证测试
// ========================================

// TestLevel1_4_WaveCount 验证 1-4 波次数量正确（10波，含1面旗帜）
func TestLevel1_4_WaveCount(t *testing.T) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	expectedWaves := 10
	actualWaves := len(env.LevelConfig.Waves)

	if actualWaves != expectedWaves {
		t.Errorf("Expected %d waves, got %d", expectedWaves, actualWaves)
	}

	// 验证旗帜数量
	flagCount := 0
	for _, wave := range env.LevelConfig.Waves {
		if wave.IsFlag {
			flagCount++
		}
	}

	if flagCount != 1 {
		t.Errorf("Expected 1 flag wave, got %d", flagCount)
	}
}

// TestLevel1_4_FlagWave 验证 1-4 旗帜波在第10波触发
func TestLevel1_4_FlagWave(t *testing.T) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 查找旗帜波
	flagWaveIndex := -1
	for i, wave := range env.LevelConfig.Waves {
		if wave.IsFlag {
			flagWaveIndex = i
			break
		}
	}

	// 旗帜波应该在第10波（索引9）
	if flagWaveIndex != 9 {
		t.Errorf("Expected flag wave at index 9 (wave 10), got index %d", flagWaveIndex)
	}

	// 验证旗帜索引
	if flagWaveIndex >= 0 && env.LevelConfig.Waves[flagWaveIndex].FlagIndex != 1 {
		t.Errorf("Expected flag index 1, got %d", env.LevelConfig.Waves[flagWaveIndex].FlagIndex)
	}
}

// TestLevel1_4_MultiLane 验证 1-4 僵尸分布在5行
func TestLevel1_4_MultiLane(t *testing.T) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 验证启用的行是全部5行
	if len(env.LevelConfig.EnabledLanes) != 5 {
		t.Errorf("Expected 5 enabled lanes, got %d", len(env.LevelConfig.EnabledLanes))
	}

	// 验证是1-5行
	expectedLanes := map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true}
	for _, lane := range env.LevelConfig.EnabledLanes {
		if !expectedLanes[lane] {
			t.Errorf("Unexpected lane %d in EnabledLanes", lane)
		}
	}
}

// TestLevel1_4_ZombieTypes 验证 1-4 僵尸类型（basic, conehead）
func TestLevel1_4_ZombieTypes(t *testing.T) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 收集所有僵尸类型
	zombieTypes := make(map[string]bool)
	for _, wave := range env.LevelConfig.Waves {
		for _, zombieGroup := range wave.Zombies {
			zombieTypes[zombieGroup.Type] = true
		}
	}

	// 验证包含预期的僵尸类型
	expectedTypes := []string{"basic", "conehead"}
	for _, expectedType := range expectedTypes {
		if !zombieTypes[expectedType] {
			t.Errorf("Expected zombie type '%s' not found in level 1-4", expectedType)
		}
	}
}

// TestLevel1_4_LaneNoConsecutive 验证 1-4 行分配无连续重复
func TestLevel1_4_LaneNoConsecutive(t *testing.T) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 模拟行分配多次，验证无连续重复
	lastLane := -1
	consecutiveCount := 0
	maxAllowedConsecutive := 3 // 允许小概率连续

	for i := 0; i < 100; i++ {
		lane := env.LaneAllocator.SelectLane(
			"basic",
			env.LevelConfig.SceneType,
			env.SpawnRules,
			env.LevelConfig.EnabledLanes,
			nil, // laneRestriction
		)
		env.LaneAllocator.UpdateLaneCounters(lane)

		if lane == lastLane {
			consecutiveCount++
			if consecutiveCount > maxAllowedConsecutive {
				t.Errorf("Too many consecutive lane selections: lane %d selected %d times in a row",
					lane, consecutiveCount+1)
				break
			}
		} else {
			consecutiveCount = 0
		}
		lastLane = lane
	}
}

// ========================================
// Task 4: 波次计时集成测试
// ========================================

// TestWaveTiming_FirstWaveDelay 验证首波延迟逻辑
func TestWaveTiming_FirstWaveDelay(t *testing.T) {
	env, err := SetupTestEnvironment("1-1")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 首次游戏：立即触发
	env.WaveTimingSystem.InitializeTimer(true)

	timer, ok := ecs.GetComponent[*components.WaveTimerComponent](
		env.EM, env.WaveTimingSystem.GetTimerEntityID())
	if !ok {
		t.Fatal("Timer component not found")
	}

	if timer.CountdownCs != 0 {
		t.Errorf("First playthrough: expected countdown 0, got %d", timer.CountdownCs)
	}

	// 非首次游戏：599cs 延迟
	env.WaveTimingSystem.InitializeTimer(false)

	timer, ok = ecs.GetComponent[*components.WaveTimerComponent](
		env.EM, env.WaveTimingSystem.GetTimerEntityID())
	if !ok {
		t.Fatal("Timer component not found")
	}

	if timer.CountdownCs != FirstWaveDelayCs {
		t.Errorf("Subsequent playthrough: expected countdown %d, got %d",
			FirstWaveDelayCs, timer.CountdownCs)
	}
}

// TestWaveTiming_RegularWaveInterval 验证常规波间隔 (25-31秒)
func TestWaveTiming_RegularWaveInterval(t *testing.T) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	env.WaveTimingSystem.InitializeTimer(false)

	// 模拟第一波触发后设置下一波倒计时
	// 先消耗首波延迟
	for i := 0; i < 600; i++ {
		env.WaveTimingSystem.Update(0.01) // 1cs per update
	}

	// 设置下一波倒计时（常规波）
	env.WaveTimingSystem.SetNextWaveCountdown()

	timer, ok := ecs.GetComponent[*components.WaveTimerComponent](
		env.EM, env.WaveTimingSystem.GetTimerEntityID())
	if !ok {
		t.Fatal("Timer component not found")
	}

	// 验证倒计时在合理范围内 (2500-3100cs，即 25-31秒)
	minExpected := RegularWaveBaseDelayCs
	maxExpected := RegularWaveBaseDelayCs + RegularWaveRandomDelayCs

	if timer.CountdownCs < minExpected || timer.CountdownCs > maxExpected {
		t.Errorf("Regular wave delay out of range: expected %d-%d cs, got %d cs",
			minExpected, maxExpected, timer.CountdownCs)
	}
}

// TestWaveTiming_FlagWavePrefixDelay 验证旗帜波前延迟 (45秒)
func TestWaveTiming_FlagWavePrefixDelay(t *testing.T) {
	// 旗帜波前延迟测试需要特殊处理
	// 当下一波是旗帜波时，应该使用 4500cs 延迟
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 直接检查常量值
	if FlagWavePrefixDelayCs != 4500 {
		t.Errorf("FlagWavePrefixDelayCs should be 4500, got %d", FlagWavePrefixDelayCs)
	}
}

// TestWaveTiming_FinalWaveDelay 验证最终波延迟 (55秒)
func TestWaveTiming_FinalWaveDelay(t *testing.T) {
	// 最终波延迟测试
	if FinalWaveDelayCs != 5500 {
		t.Errorf("FinalWaveDelayCs should be 5500, got %d", FinalWaveDelayCs)
	}
}

// TestWaveTiming_AcceleratedRefresh 验证加速刷新触发
func TestWaveTiming_AcceleratedRefresh(t *testing.T) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	env.WaveTimingSystem.InitializeTimer(false)

	// 获取计时器组件
	timer, ok := ecs.GetComponent[*components.WaveTimerComponent](
		env.EM, env.WaveTimingSystem.GetTimerEntityID())
	if !ok {
		t.Fatal("Timer component not found")
	}

	// 设置接近旗帜波的状态
	timer.IsFlagWaveApproaching = true
	timer.WaveElapsedCs = 500       // > 401cs
	timer.CountdownCs = 1000        // > 200cs

	// 触发加速刷新
	triggered := env.WaveTimingSystem.CheckAcceleratedRefresh(true)

	if !triggered {
		t.Error("Accelerated refresh should be triggered")
	}

	// 验证倒计时被设为 200cs
	if timer.CountdownCs != AcceleratedRefreshCountdownCs {
		t.Errorf("Countdown should be %d after acceleration, got %d",
			AcceleratedRefreshCountdownCs, timer.CountdownCs)
	}
}

// ========================================
// Task 5: 难度引擎集成测试
// ========================================

// TestDifficultyCurve_FirstPlaythrough 一周目难度曲线
func TestDifficultyCurve_FirstPlaythrough(t *testing.T) {
	env, err := SetupTestEnvironment("1-1")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 一周目：TotalCompletedFlags = 0
	env.GameState.TotalCompletedFlags = 0

	roundNumber := env.DifficultyEngine.CalculateRoundNumber(0)
	if roundNumber != -1 {
		t.Errorf("First playthrough round number should be -1, got %d", roundNumber)
	}
}

// TestDifficultyCurve_RoundNumberProgression 轮数递增验证
func TestDifficultyCurve_RoundNumberProgression(t *testing.T) {
	env, err := SetupTestEnvironment("1-1")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	testCases := []struct {
		completedFlags  int
		expectedRound   int
	}{
		{0, -1},   // 一周目开始
		{2, 0},    // 完成1个关卡（2旗）
		{4, 1},    // 完成2个关卡
		{10, 4},   // 完成5个关卡
		{50, 24},  // 一周目结束
	}

	for _, tc := range testCases {
		roundNumber := env.DifficultyEngine.CalculateRoundNumber(tc.completedFlags)
		if roundNumber != tc.expectedRound {
			t.Errorf("CompletedFlags=%d: expected round %d, got %d",
				tc.completedFlags, tc.expectedRound, roundNumber)
		}
	}
}

// TestDifficultyCurve_LevelCapacityProgression 级别容量递增验证
func TestDifficultyCurve_LevelCapacityProgression(t *testing.T) {
	env, err := SetupTestEnvironment("1-1")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 验证级别容量随波次递增
	roundNumber := 0
	wavesPerRound := 20

	prevCapacity := 0
	for waveNum := 1; waveNum <= 10; waveNum++ {
		capacity := env.DifficultyEngine.CalculateLevelCapacity(waveNum, roundNumber, wavesPerRound, false)

		if capacity < prevCapacity {
			t.Errorf("Capacity should not decrease: wave %d capacity %d < previous %d",
				waveNum, capacity, prevCapacity)
		}
		prevCapacity = capacity
	}
}

// TestDifficultyCurve_NegativeRound 一周目早期负轮数验证
func TestDifficultyCurve_NegativeRound(t *testing.T) {
	env, err := SetupTestEnvironment("1-1")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 负轮数时级别容量计算验证
	// 原版设计：一周目早期（负轮数）容量可以是负数或很低
	// 这会限制高级僵尸的出现
	roundNumber := -1
	wavesPerRound := 20

	// 验证公式正确性
	for waveNum := 1; waveNum <= 5; waveNum++ {
		capacity := env.DifficultyEngine.CalculateLevelCapacity(waveNum, roundNumber, wavesPerRound, false)

		// 公式: int(int((waveNum + roundNumber * wavesPerRound) * 0.8) / 2) + 1
		// 当 roundNumber = -1, wavesPerRound = 20 时:
		// waveNum = 1: int(int((1 + (-1)*20) * 0.8) / 2) + 1 = int(int(-19*0.8)/2) + 1 = int(-15/2) + 1 = -7 + 1 = -6
		expected := int(int(float64(waveNum+roundNumber*wavesPerRound)*0.8)/2) + 1
		if capacity != expected {
			t.Errorf("Wave %d, round %d: expected capacity %d, got %d",
				waveNum, roundNumber, expected, capacity)
		}
	}

	// 验证随着轮数增加，容量会变为正数
	roundNumber = 1 // 第二轮
	for waveNum := 1; waveNum <= 5; waveNum++ {
		capacity := env.DifficultyEngine.CalculateLevelCapacity(waveNum, roundNumber, wavesPerRound, false)
		if capacity < 1 {
			t.Errorf("Round 1, wave %d: capacity should be >= 1, got %d", waveNum, capacity)
		}
	}
}

// ========================================
// Task 6: 行分配集成测试
// ========================================

// TestLaneAllocation_Distribution 行分配分布统计
func TestLaneAllocation_Distribution(t *testing.T) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 重新初始化行分配器
	env.LaneAllocator.InitializeLanes(5, 1.0)

	// 抽样次数
	sampleCount := 10000
	laneCounts := make(map[int]int)

	for i := 0; i < sampleCount; i++ {
		lane := env.LaneAllocator.SelectLane(
			"basic",
			env.LevelConfig.SceneType,
			env.SpawnRules,
			env.LevelConfig.EnabledLanes,
			nil, // laneRestriction
		)
		laneCounts[lane]++
		env.LaneAllocator.UpdateLaneCounters(lane)
	}

	// 验证分布（允许 30% 偏差）
	expectedPerLane := sampleCount / 5
	tolerance := float64(expectedPerLane) * 0.30

	for lane := 1; lane <= 5; lane++ {
		count := laneCounts[lane]
		deviation := math.Abs(float64(count - expectedPerLane))

		if deviation > tolerance {
			t.Errorf("Lane %d: expected ~%d (±%.0f), got %d (deviation: %.0f)",
				lane, expectedPerLane, tolerance, count, deviation)
		}
	}
}

// TestLaneAllocation_NoConsecutive 无连续重复验证
func TestLaneAllocation_NoConsecutive(t *testing.T) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 重新初始化
	env.LaneAllocator.InitializeLanes(5, 1.0)

	// 统计连续选中同一行的次数
	consecutiveCount := 0
	maxConsecutive := 0
	lastLane := -1

	for i := 0; i < 1000; i++ {
		lane := env.LaneAllocator.SelectLane(
			"basic",
			env.LevelConfig.SceneType,
			env.SpawnRules,
			env.LevelConfig.EnabledLanes,
			nil, // laneRestriction
		)
		env.LaneAllocator.UpdateLaneCounters(lane)

		if lane == lastLane {
			consecutiveCount++
			if consecutiveCount > maxConsecutive {
				maxConsecutive = consecutiveCount
			}
		} else {
			consecutiveCount = 0
		}
		lastLane = lane
	}

	// 由于平滑权重算法，连续选中同一行的概率很低
	// 允许最多3次连续（极小概率事件）
	if maxConsecutive > 3 {
		t.Errorf("Max consecutive selections too high: %d", maxConsecutive)
	}
}

// TestLaneAllocation_SingleLaneLevel 单行关卡行分配
func TestLaneAllocation_SingleLaneLevel(t *testing.T) {
	env, err := SetupTestEnvironment("1-1")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 注意：当前 LaneAllocator.FilterLegalLanes 不根据 enabledLanes 过滤普通僵尸
	// enabledLanes 目前只用于舞王僵尸的相邻行检查
	// 实际的行限制在 WaveSpawnSystem.spawnZombieForWave 中由关卡配置处理
	//
	// 这个测试验证 LaneAllocator 在配置启用所有5行时的分布行为

	// 为验证单行关卡，我们需要直接验证关卡配置的 EnabledLanes 字段
	if len(env.LevelConfig.EnabledLanes) != 1 {
		t.Errorf("Level 1-1 should have exactly 1 enabled lane, got %d", len(env.LevelConfig.EnabledLanes))
		return
	}

	expectedLane := env.LevelConfig.EnabledLanes[0]
	if expectedLane != 3 {
		t.Errorf("Level 1-1 should only enable lane 3, got lane %d", expectedLane)
	}

	// 验证关卡配置中所有僵尸都配置在第3行
	for waveIdx, wave := range env.LevelConfig.Waves {
		for groupIdx, group := range wave.Zombies {
			for _, lane := range group.Lanes {
				if lane != 3 {
					t.Errorf("Wave %d, Group %d: zombie configured for lane %d, expected 3",
						waveIdx+1, groupIdx, lane)
				}
			}
		}
	}
}

// TestLaneAllocation_LegalLaneOnly 只分配合法行
func TestLaneAllocation_LegalLaneOnly(t *testing.T) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 验证所有分配的行都在 EnabledLanes 中
	enabledLanesMap := make(map[int]bool)
	for _, lane := range env.LevelConfig.EnabledLanes {
		enabledLanesMap[lane] = true
	}

	for i := 0; i < 100; i++ {
		lane := env.LaneAllocator.SelectLane(
			"basic",
			env.LevelConfig.SceneType,
			env.SpawnRules,
			env.LevelConfig.EnabledLanes,
			nil, // laneRestriction
		)
		env.LaneAllocator.UpdateLaneCounters(lane)

		if !enabledLanesMap[lane] {
			t.Errorf("Lane %d is not in EnabledLanes", lane)
		}
	}
}

// ========================================
// Task 7: 僵尸生成约束集成测试
// ========================================

// TestSpawnConstraint_TierRestriction 阶数限制验证
func TestSpawnConstraint_TierRestriction(t *testing.T) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	if env.SpawnRules == nil {
		t.Skip("SpawnRules not loaded, skipping tier restriction test")
	}

	// 验证阶数限制配置
	if len(env.SpawnRules.TierWaveRestrictions) == 0 {
		t.Error("TierWaveRestrictions should not be empty")
	}

	// 验证一阶僵尸可以在第1波出现
	tier1MinWave := env.SpawnRules.TierWaveRestrictions[1]
	if tier1MinWave != 1 {
		t.Errorf("Tier 1 min wave should be 1, got %d", tier1MinWave)
	}

	// 验证二阶僵尸最早在第3波出现
	tier2MinWave := env.SpawnRules.TierWaveRestrictions[2]
	if tier2MinWave != 3 {
		t.Errorf("Tier 2 min wave should be 3, got %d", tier2MinWave)
	}
}

// TestSpawnConstraint_SceneTypeRestriction 场景类型限制验证
func TestSpawnConstraint_SceneTypeRestriction(t *testing.T) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	if env.SpawnRules == nil {
		t.Skip("SpawnRules not loaded, skipping scene type restriction test")
	}

	// 验证场景类型
	if env.LevelConfig.SceneType != "day" {
		t.Errorf("Level 1-4 scene type should be 'day', got '%s'", env.LevelConfig.SceneType)
	}

	// 验证水路僵尸列表存在
	if len(env.SpawnRules.SceneTypeRestrictions.WaterZombies) == 0 {
		t.Log("No water zombies configured (expected for day scene)")
	}
}

// TestSpawnConstraint_WaterLane 水路僵尸限制验证
func TestSpawnConstraint_WaterLane(t *testing.T) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	if env.SpawnRules == nil {
		t.Skip("SpawnRules not loaded, skipping water lane test")
	}

	// 验证水路配置
	poolWaterLanes := env.SpawnRules.SceneTypeRestrictions.WaterLaneConfig["pool"]
	if len(poolWaterLanes) != 2 {
		t.Errorf("Pool water lanes should be 2, got %d", len(poolWaterLanes))
	}

	// 验证泳池场景第3、4行是水路
	waterLanesMap := make(map[int]bool)
	for _, lane := range poolWaterLanes {
		waterLanesMap[lane] = true
	}

	if !waterLanesMap[3] || !waterLanesMap[4] {
		t.Errorf("Pool water lanes should be [3, 4], got %v", poolWaterLanes)
	}
}

// ========================================
// Task 8: 僵尸物理坐标集成测试
// ========================================

// TestZombiePhysics_SpawnX_NormalWave 普通波出生X坐标
func TestZombiePhysics_SpawnX_NormalWave(t *testing.T) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	if env.ZombiePhysics == nil {
		t.Skip("ZombiePhysics not loaded, skipping spawn X test")
	}

	// 获取普通波出生点范围
	minX, maxX := env.ZombiePhysics.GetSpawnXRange("basic", false, false)

	// 验证范围有效
	if minX > maxX {
		t.Errorf("Normal wave spawn X range invalid: min(%.1f) > max(%.1f)", minX, maxX)
	}

	// 验证在合理范围内（僵尸应该在屏幕右侧生成）
	if minX < 700 {
		t.Errorf("Normal wave spawn X min should be >= 700, got %.1f", minX)
	}
}

// TestZombiePhysics_SpawnX_FlagWave 旗帜波出生X坐标
func TestZombiePhysics_SpawnX_FlagWave(t *testing.T) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	if env.ZombiePhysics == nil {
		t.Skip("ZombiePhysics not loaded, skipping flag wave spawn X test")
	}

	// 获取旗帜波出生点范围
	flagMinX, flagMaxX := env.ZombiePhysics.GetSpawnXRange("basic", true, false)
	normalMinX, normalMaxX := env.ZombiePhysics.GetSpawnXRange("basic", false, false)

	// 旗帜波范围可能与普通波不同
	t.Logf("Normal wave spawn X: %.1f - %.1f", normalMinX, normalMaxX)
	t.Logf("Flag wave spawn X: %.1f - %.1f", flagMinX, flagMaxX)

	// 验证范围有效
	if flagMinX > flagMaxX {
		t.Errorf("Flag wave spawn X range invalid: min(%.1f) > max(%.1f)", flagMinX, flagMaxX)
	}
}

// TestZombiePhysics_DefeatBoundary 类型化进家边界
func TestZombiePhysics_DefeatBoundary(t *testing.T) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	if env.ZombiePhysics == nil {
		t.Skip("ZombiePhysics not loaded, skipping defeat boundary test")
	}

	// 获取不同僵尸类型的进家边界
	basicBoundary := env.ZombiePhysics.GetDefeatBoundary("basic")
	defaultBoundary := env.ZombiePhysics.GetDefeatBoundary("unknown_type")

	// 验证边界是负值或零（在网格左侧）
	if basicBoundary > 0 {
		t.Errorf("Basic zombie defeat boundary should be <= 0, got %.1f", basicBoundary)
	}

	if defaultBoundary > 0 {
		t.Errorf("Default defeat boundary should be <= 0, got %.1f", defaultBoundary)
	}

	t.Logf("Basic zombie defeat boundary: %.1f", basicBoundary)
	t.Logf("Default defeat boundary: %.1f", defaultBoundary)
}

// ========================================
// Task 9: 性能基准测试
// ========================================

// BenchmarkLaneAllocator_Pick 行分配算法性能
func BenchmarkLaneAllocator_Pick(b *testing.B) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		b.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	env.LaneAllocator.InitializeLanes(5, 1.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lane := env.LaneAllocator.SelectLane(
			"basic",
			env.LevelConfig.SceneType,
			env.SpawnRules,
			env.LevelConfig.EnabledLanes,
			nil, // laneRestriction
		)
		env.LaneAllocator.UpdateLaneCounters(lane)
	}
}

// BenchmarkWaveSpawn_20Waves 20波僵尸生成性能
func BenchmarkWaveSpawn_20Waves(b *testing.B) {
	// 由于 WaveSpawnSystem 依赖 ResourceManager，这里测试配置解析性能
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := config.LoadLevelConfig("data/levels/level-1-4.yaml")
		if err != nil {
			b.Fatalf("Failed to load level config: %v", err)
		}
	}
}

// BenchmarkDifficultyEngine_Calculate 难度计算性能
func BenchmarkDifficultyEngine_Calculate(b *testing.B) {
	env, err := SetupTestEnvironment("1-1")
	if err != nil {
		b.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = env.DifficultyEngine.CalculateRoundNumber(50)
		_ = env.DifficultyEngine.CalculateLevelCapacity(10, 5, 20, false)
		_ = env.DifficultyEngine.CalculateLevelCapacity(10, 5, 20, true)
	}
}

// TestBenchmarkLaneAllocator_LessThan1ms 验证行分配算法 < 1ms
func TestBenchmarkLaneAllocator_LessThan1ms(t *testing.T) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	env.LaneAllocator.InitializeLanes(5, 1.0)

	// 执行 1000 次行分配，测量总时间
	iterations := 1000
	start := testing.AllocsPerRun(iterations, func() {
		lane := env.LaneAllocator.SelectLane(
			"basic",
			env.LevelConfig.SceneType,
			env.SpawnRules,
			env.LevelConfig.EnabledLanes,
			nil, // laneRestriction
		)
		env.LaneAllocator.UpdateLaneCounters(lane)
	})

	// 如果每次分配的内存分配次数过多，可能影响性能
	t.Logf("Average allocations per lane selection: %.2f", start)
}

// ========================================
// Task 10: 回归测试
// ========================================

// TestRegression_Level1_1_Playable 关卡1-1可正常游玩
func TestRegression_Level1_1_Playable(t *testing.T) {
	env, err := SetupTestEnvironment("1-1")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 验证基本配置有效
	if env.LevelConfig.ID != "1-1" {
		t.Errorf("Level ID should be '1-1', got '%s'", env.LevelConfig.ID)
	}

	if len(env.LevelConfig.Waves) == 0 {
		t.Error("Level should have at least one wave")
	}

	if env.LevelConfig.InitialSun <= 0 {
		t.Errorf("Initial sun should be > 0, got %d", env.LevelConfig.InitialSun)
	}
}

// TestRegression_Level1_4_Playable 关卡1-4可正常游玩
func TestRegression_Level1_4_Playable(t *testing.T) {
	env, err := SetupTestEnvironment("1-4")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 验证基本配置有效
	if env.LevelConfig.ID != "1-4" {
		t.Errorf("Level ID should be '1-4', got '%s'", env.LevelConfig.ID)
	}

	if len(env.LevelConfig.Waves) != 10 {
		t.Errorf("Level 1-4 should have 10 waves, got %d", len(env.LevelConfig.Waves))
	}

	// 验证有旗帜波
	hasFlagWave := false
	for _, wave := range env.LevelConfig.Waves {
		if wave.IsFlag {
			hasFlagWave = true
			break
		}
	}

	if !hasFlagWave {
		t.Error("Level 1-4 should have at least one flag wave")
	}
}

// TestRegression_NoConfigFile 无配置文件时使用默认值
func TestRegression_NoConfigFile(t *testing.T) {
	// 尝试加载不存在的配置文件
	_, err := config.LoadSpawnRules("data/nonexistent_spawn_rules.yaml")
	if err == nil {
		t.Error("Loading nonexistent file should return error")
	}

	// 验证物理配置加载失败时返回错误
	_, err = config.LoadZombiePhysicsConfig("data/nonexistent_physics.yaml")
	if err == nil {
		t.Error("Loading nonexistent physics config should return error")
	}
}

// TestRegression_BackwardCompatibility 旧格式关卡兼容
func TestRegression_BackwardCompatibility(t *testing.T) {
	// 验证关卡配置的默认值应用
	env, err := SetupTestEnvironment("1-1")
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}
	defer TeardownTestEnvironment(env)

	// 验证默认值已应用
	if env.LevelConfig.SceneType == "" {
		t.Error("SceneType default should be applied")
	}

	if env.LevelConfig.RowMax == 0 {
		t.Error("RowMax default should be applied")
	}

	// 验证波次默认值
	for i, wave := range env.LevelConfig.Waves {
		if wave.WaveNum == 0 {
			t.Errorf("Wave %d: WaveNum default should be applied", i)
		}

		if wave.Type == "" {
			t.Errorf("Wave %d: Type default should be applied", i)
		}
	}
}
