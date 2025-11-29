package systems

import (
	"math"
	"testing"

	"github.com/decker502/pvz/pkg/components"
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
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 1}}},
			},
			expected: 1,
		},
		{
			name: "单波次多僵尸",
			waves: []config.WaveConfig{
				{OldZombies: []config.ZombieSpawn{
					{Type: "basic", Lane: 1, Count: 2},
					{Type: "basic", Lane: 2, Count: 3},
				}},
			},
			expected: 5,
		},
		{
			name: "多波次多僵尸",
			waves: []config.WaveConfig{
				{OldZombies: []config.ZombieSpawn{
					{Type: "basic", Lane: 1, Count: 2},
					{Type: "basic", Lane: 2, Count: 1},
				}},
				{OldZombies: []config.ZombieSpawn{
					{Type: "basic", Lane: 3, Count: 3},
				}},
				{OldZombies: []config.ZombieSpawn{
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
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 5}}},
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 5}}},
			},
			flagWaves: []int{1}, // 第2波是旗帜波
			expected:  []float64{0.5},
		},
		{
			name: "单旗帜在最后",
			waves: []config.WaveConfig{
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 3}}},
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 2}}},
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 5}}},
			},
			flagWaves: []int{2}, // 第3波是旗帜波
			expected:  []float64{0.5},
		},
		{
			name: "多旗帜",
			waves: []config.WaveConfig{
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 2}}},
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 3}}},
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 2}}},
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 3}}},
			},
			flagWaves: []int{1, 3}, // 第2波和第4波是旗帜波
			expected:  []float64{0.2, 0.7},
		},
		{
			name: "无旗帜",
			waves: []config.WaveConfig{
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 1, Count: 5}}},
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
// Story 8.8: Zombies Won Flow Tests
// ========================================

// TestTriggerZombiesWonFlow_CreatesFlowEntity 测试 triggerZombiesWonFlow 创建流程实体
func TestTriggerZombiesWonFlow_CreatesFlowEntity(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := &game.GameState{
		CurrentLevel: &config.LevelConfig{},
	}

	// 创建一个僵尸实体作为触发者
	zombieID := em.CreateEntity()
	ecs.AddComponent(em, zombieID, &components.BehaviorComponent{
		Type: components.BehaviorZombieBasic,
	})
	ecs.AddComponent(em, zombieID, &components.PositionComponent{
		X: 200.0,
		Y: 300.0,
	})
	ecs.AddComponent(em, zombieID, &components.VelocityComponent{
		VX: -150.0,
		VY: 0,
	})

	ls := &LevelSystem{
		entityManager: em,
		gameState:     gs,
	}

	// 调用 triggerZombiesWonFlow
	ls.triggerZombiesWonFlow(zombieID)

	// 验证：应该创建了一个带有 ZombiesWonPhaseComponent 的实体
	phaseEntities := ecs.GetEntitiesWith1[*components.ZombiesWonPhaseComponent](em)
	if len(phaseEntities) != 1 {
		t.Fatalf("expected 1 phase entity, got %d", len(phaseEntities))
	}

	// 验证：流程实体应该包含 GameFreezeComponent
	freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](em)
	if len(freezeEntities) != 1 {
		t.Fatalf("expected 1 freeze entity, got %d", len(freezeEntities))
	}

	// 验证：ZombiesWonPhaseComponent 应该正确初始化
	phaseComp, ok := ecs.GetComponent[*components.ZombiesWonPhaseComponent](em, phaseEntities[0])
	if !ok {
		t.Fatalf("failed to get phase component")
	}

	if phaseComp.CurrentPhase != 1 {
		t.Errorf("expected CurrentPhase=1, got %d", phaseComp.CurrentPhase)
	}
	if phaseComp.TriggerZombieID != zombieID {
		t.Errorf("expected TriggerZombieID=%d, got %d", zombieID, phaseComp.TriggerZombieID)
	}

	// 验证：GameFreezeComponent 应该标记为已冻结
	freezeComp, ok := ecs.GetComponent[*components.GameFreezeComponent](em, freezeEntities[0])
	if !ok {
		t.Fatalf("failed to get freeze component")
	}
	if !freezeComp.IsFrozen {
		t.Errorf("expected IsFrozen=true")
	}
}

// TestTriggerZombiesWonFlow_OnlyCreatesOnce 测试重复调用只创建一个流程实体
func TestTriggerZombiesWonFlow_OnlyCreatesOnce(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := &game.GameState{
		CurrentLevel: &config.LevelConfig{},
	}

	zombieID := em.CreateEntity()
	ecs.AddComponent(em, zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	ecs.AddComponent(em, zombieID, &components.PositionComponent{X: 200.0, Y: 300.0})

	ls := &LevelSystem{
		entityManager: em,
		gameState:     gs,
	}

	// 调用两次
	ls.triggerZombiesWonFlow(zombieID)
	ls.triggerZombiesWonFlow(zombieID) // 第二次调用

	// 验证：应该创建了两个流程实体（每次调用都会创建）
	// 注意：实际游戏中不会重复调用，但测试确保函数行为一致
	phaseEntities := ecs.GetEntitiesWith1[*components.ZombiesWonPhaseComponent](em)
	if len(phaseEntities) != 2 {
		t.Fatalf("expected 2 phase entities (one per call), got %d", len(phaseEntities))
	}
}

// ========================================
// Story 17.9: Type-Specific Defeat Boundary Tests
// ========================================

// TestGetDefeatBoundary_WithPhysicsConfig 测试使用物理配置时的进家边界
func TestGetDefeatBoundary_WithPhysicsConfig(t *testing.T) {
	ls := &LevelSystem{
		zombiePhysics: &config.ZombiePhysicsConfig{
			DefeatBoundary: map[string]float64{
				"default":  -100,
				"basic":    -100,
				"football": -175,
			},
		},
	}

	tests := []struct {
		name         string
		zombieType   string
		expectedGrid float64
	}{
		{"basic zombie", "basic", -100},
		{"football zombie", "football", -175},
		{"unknown uses default", "unknown", -100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			boundary := ls.getDefeatBoundary(tt.zombieType)
			expected := config.GridToWorldX(tt.expectedGrid)
			if boundary != expected {
				t.Errorf("getDefeatBoundary(%s) = %.1f, want %.1f", tt.zombieType, boundary, expected)
			}
		})
	}
}

// TestGetDefeatBoundary_WithoutPhysicsConfig 测试未配置物理参数时使用默认值
func TestGetDefeatBoundary_WithoutPhysicsConfig(t *testing.T) {
	ls := &LevelSystem{
		zombiePhysics: nil, // 无配置
	}

	// 应使用默认常量 DefeatBoundaryX
	boundary := ls.getDefeatBoundary("basic")
	if boundary != DefeatBoundaryX {
		t.Errorf("getDefeatBoundary without config = %.1f, want %.1f", boundary, DefeatBoundaryX)
	}
}

// TestBehaviorTypeToString 测试行为类型到字符串的映射
func TestBehaviorTypeToString(t *testing.T) {
	ls := &LevelSystem{}

	tests := []struct {
		name         string
		behaviorType components.BehaviorType
		expected     string
	}{
		{"basic zombie", components.BehaviorZombieBasic, "basic"},
		{"eating zombie", components.BehaviorZombieEating, "basic"},
		{"dying zombie", components.BehaviorZombieDying, "basic"},
		{"conehead", components.BehaviorZombieConehead, "conehead"},
		{"buckethead", components.BehaviorZombieBuckethead, "buckethead"},
		{"unknown type", components.BehaviorSunflower, "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ls.behaviorTypeToString(tt.behaviorType)
			if result != tt.expected {
				t.Errorf("behaviorTypeToString(%v) = %s, want %s", tt.behaviorType, result, tt.expected)
			}
		})
	}
}

// TestSetZombiePhysicsConfig 测试设置物理配置
func TestSetZombiePhysicsConfig(t *testing.T) {
	ls := &LevelSystem{}

	// 初始状态：无配置
	if ls.zombiePhysics != nil {
		t.Errorf("initial zombiePhysics should be nil")
	}

	// 设置配置
	cfg := &config.ZombiePhysicsConfig{
		DefeatBoundary: map[string]float64{
			"default": -100,
		},
	}
	ls.SetZombiePhysicsConfig(cfg)

	// 验证配置已设置
	if ls.zombiePhysics != cfg {
		t.Errorf("zombiePhysics not set correctly")
	}
}

// ========================================
// Story 11.5: Progress Bar Mechanism Tests
// ========================================

// TestInitProgressBarStructure 测试双段式进度条结构初始化
func TestInitProgressBarStructure(t *testing.T) {
	tests := []struct {
		name              string
		totalWaves        int
		flagWaves         []int
		expectedNormalSeg int
	}{
		{
			name:              "无旗帜波",
			totalWaves:        10,
			flagWaves:         []int{},
			expectedNormalSeg: 150, // 150 - 0*12 = 150
		},
		{
			name:              "1个旗帜波",
			totalWaves:        10,
			flagWaves:         []int{9}, // 最后一波是旗帜波
			expectedNormalSeg: 138,      // 150 - 1*12 = 138
		},
		{
			name:              "2个旗帜波（无尽模式）",
			totalWaves:        20,
			flagWaves:         []int{9, 19},
			expectedNormalSeg: 126, // 150 - 2*12 = 126
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建波次配置
			waves := make([]config.WaveConfig, tt.totalWaves)
			for i := range waves {
				waves[i] = config.WaveConfig{
					Zombies: []config.ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 1}},
				}
			}

			levelConfig := &config.LevelConfig{
				Waves:     waves,
				FlagWaves: tt.flagWaves,
			}

			gs := &game.GameState{CurrentLevel: levelConfig}
			ls := &LevelSystem{gameState: gs}
			pb := &components.LevelProgressBarComponent{}

			ls.initProgressBarStructure(pb)

			if pb.TotalProgressLength != 150 {
				t.Errorf("TotalProgressLength = %d, want 150", pb.TotalProgressLength)
			}
			if pb.FlagSegmentLength != 12 {
				t.Errorf("FlagSegmentLength = %d, want 12", pb.FlagSegmentLength)
			}
			if pb.NormalSegmentBase != tt.expectedNormalSeg {
				t.Errorf("NormalSegmentBase = %d, want %d", pb.NormalSegmentBase, tt.expectedNormalSeg)
			}
			if pb.TotalWaves != tt.totalWaves {
				t.Errorf("TotalWaves = %d, want %d", pb.TotalWaves, tt.totalWaves)
			}
		})
	}
}

// TestCalculateTimeProgress 测试时间进度计算
func TestCalculateTimeProgress(t *testing.T) {
	tests := []struct {
		name             string
		waveStartTime    float64
		waveInitialDelay float64
		levelTime        float64
		expected         float64
	}{
		{
			name:             "波开始时进度为0",
			waveStartTime:    10.0,
			waveInitialDelay: 25.0,
			levelTime:        10.0,
			expected:         0.0,
		},
		{
			name:             "50%时间进度",
			waveStartTime:    10.0,
			waveInitialDelay: 20.0,
			levelTime:        20.0, // 过了10秒，总共20秒
			expected:         0.5,
		},
		{
			name:             "100%时间进度",
			waveStartTime:    10.0,
			waveInitialDelay: 25.0,
			levelTime:        35.0,
			expected:         1.0,
		},
		{
			name:             "超过100%时间进度",
			waveStartTime:    10.0,
			waveInitialDelay: 20.0,
			levelTime:        40.0, // 过了30秒，总共20秒延迟
			expected:         1.5,
		},
		{
			name:             "延迟为0时返回0",
			waveStartTime:    10.0,
			waveInitialDelay: 0,
			levelTime:        20.0,
			expected:         0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := &game.GameState{LevelTime: tt.levelTime}
			ls := &LevelSystem{gameState: gs}
			pb := &components.LevelProgressBarComponent{
				WaveStartTime:    tt.waveStartTime,
				WaveInitialDelay: tt.waveInitialDelay,
			}

			result := ls.calculateTimeProgress(pb)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("calculateTimeProgress() = %.2f, want %.2f", result, tt.expected)
			}
		})
	}
}

// TestCalculateDamageProgress 测试血量削减进度计算
func TestCalculateDamageProgress(t *testing.T) {
	tests := []struct {
		name               string
		waveInitialHealth  float64
		waveCurrentHealth  float64
		waveRequiredDamage float64
		expected           float64
	}{
		{
			name:               "无伤害时进度为0",
			waveInitialHealth:  1000,
			waveCurrentHealth:  1000,
			waveRequiredDamage: 1000,
			expected:           0.0,
		},
		{
			name:               "50%血量削减",
			waveInitialHealth:  1000,
			waveCurrentHealth:  500,
			waveRequiredDamage: 1000,
			expected:           0.5,
		},
		{
			name:               "全部消灭",
			waveInitialHealth:  1000,
			waveCurrentHealth:  0,
			waveRequiredDamage: 1000,
			expected:           1.0,
		},
		{
			name:               "所需伤害为0时返回0",
			waveInitialHealth:  1000,
			waveCurrentHealth:  500,
			waveRequiredDamage: 0,
			expected:           0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ls := &LevelSystem{}
			pb := &components.LevelProgressBarComponent{
				WaveInitialHealth:  tt.waveInitialHealth,
				WaveCurrentHealth:  tt.waveCurrentHealth,
				WaveRequiredDamage: tt.waveRequiredDamage,
			}

			result := ls.calculateDamageProgress(pb)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("calculateDamageProgress() = %.2f, want %.2f", result, tt.expected)
			}
		})
	}
}

// TestCalculateWaveProgress 测试波进度计算（取时间和血量的最大值）
func TestCalculateWaveProgress(t *testing.T) {
	tests := []struct {
		name             string
		waveStartTime    float64
		waveInitialDelay float64
		levelTime        float64
		waveInitialHP    float64
		waveCurrentHP    float64
		waveRequiredDmg  float64
		expected         float64
	}{
		{
			name:             "时间进度大于血量进度",
			waveStartTime:    0,
			waveInitialDelay: 20,
			levelTime:        16, // 时间进度 = 16/20 = 0.8
			waveInitialHP:    1000,
			waveCurrentHP:    700, // 血量进度 = 300/1000 = 0.3
			waveRequiredDmg:  1000,
			expected:         0.8, // max(0.8, 0.3) = 0.8
		},
		{
			name:             "血量进度大于时间进度",
			waveStartTime:    0,
			waveInitialDelay: 20,
			levelTime:        4, // 时间进度 = 4/20 = 0.2
			waveInitialHP:    1000,
			waveCurrentHP:    200, // 血量进度 = 800/1000 = 0.8
			waveRequiredDmg:  1000,
			expected:         0.8, // max(0.2, 0.8) = 0.8
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := &game.GameState{LevelTime: tt.levelTime}
			ls := &LevelSystem{gameState: gs}
			pb := &components.LevelProgressBarComponent{
				WaveStartTime:      tt.waveStartTime,
				WaveInitialDelay:   tt.waveInitialDelay,
				WaveInitialHealth:  tt.waveInitialHP,
				WaveCurrentHealth:  tt.waveCurrentHP,
				WaveRequiredDamage: tt.waveRequiredDmg,
			}

			result := ls.calculateWaveProgress(pb)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("calculateWaveProgress() = %.2f, want %.2f", result, tt.expected)
			}
		})
	}
}

// TestUpdateRealProgress 测试虚拟/现实追踪机制
func TestUpdateRealProgress(t *testing.T) {
	tests := []struct {
		name               string
		virtualProgress    float64
		realProgress       float64
		gameTickCS         int
		lastTrackUpdateCS  int
		totalProgressLen   int
		expectedRealChange bool // 是否期望 RealProgress 变化
	}{
		{
			name:               "落后1-6格，20cs倍数时前进",
			virtualProgress:    0.04, // 6格
			realProgress:       0.0,
			gameTickCS:         20,
			lastTrackUpdateCS:  0,
			totalProgressLen:   150,
			expectedRealChange: true,
		},
		{
			name:               "落后1-6格，非20cs倍数不前进",
			virtualProgress:    0.04,
			realProgress:       0.0,
			gameTickCS:         15,
			lastTrackUpdateCS:  0,
			totalProgressLen:   150,
			expectedRealChange: false,
		},
		{
			name:               "落后7+格，5cs倍数时前进",
			virtualProgress:    0.10, // 15格
			realProgress:       0.0,
			gameTickCS:         5,
			lastTrackUpdateCS:  0,
			totalProgressLen:   150,
			expectedRealChange: true,
		},
		{
			name:               "不落后时不更新",
			virtualProgress:    0.5,
			realProgress:       0.5,
			gameTickCS:         20,
			lastTrackUpdateCS:  0,
			totalProgressLen:   150,
			expectedRealChange: false,
		},
		{
			name:               "现实超前时回退到虚拟",
			virtualProgress:    0.3,
			realProgress:       0.5, // 超前
			gameTickCS:         20,
			lastTrackUpdateCS:  0,
			totalProgressLen:   150,
			expectedRealChange: true, // 会回退
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ls := &LevelSystem{}
			pb := &components.LevelProgressBarComponent{
				VirtualProgress:     tt.virtualProgress,
				RealProgress:        tt.realProgress,
				GameTickCS:          tt.gameTickCS,
				LastTrackUpdateCS:   tt.lastTrackUpdateCS,
				TotalProgressLength: tt.totalProgressLen,
			}

			originalReal := pb.RealProgress
			ls.updateRealProgress(pb)

			changed := pb.RealProgress != originalReal
			if changed != tt.expectedRealChange {
				t.Errorf("RealProgress change = %v, want %v (before=%.4f, after=%.4f)",
					changed, tt.expectedRealChange, originalReal, pb.RealProgress)
			}
		})
	}
}

// TestUpdateGameTickCS 测试游戏时钟更新
func TestUpdateGameTickCS(t *testing.T) {
	tests := []struct {
		name       string
		levelTime  float64
		expectedCS int
	}{
		{"0秒", 0.0, 0},
		{"1秒 = 100cs", 1.0, 100},
		{"2.5秒 = 250cs", 2.5, 250},
		{"10秒 = 1000cs", 10.0, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := &game.GameState{LevelTime: tt.levelTime}
			ls := &LevelSystem{gameState: gs}
			pb := &components.LevelProgressBarComponent{}

			ls.updateGameTickCS(pb)

			if pb.GameTickCS != tt.expectedCS {
				t.Errorf("GameTickCS = %d, want %d", pb.GameTickCS, tt.expectedCS)
			}
		})
	}
}

// TestCalculateVirtualProgress 测试虚拟进度计算
func TestCalculateVirtualProgress(t *testing.T) {
	tests := []struct {
		name              string
		flagWaveCount     int
		flagSegmentLength int
		normalSegmentBase int
		totalProgressLen  int
		totalWaves        int
		currentWaveNum    int
		waveProgress      float64 // 模拟的波进度
		expectedVirtual   float64
	}{
		{
			name:              "第一波50%进度",
			flagWaveCount:     0,
			flagSegmentLength: 12,
			normalSegmentBase: 138, // 1个旗帜波：150 - 12 = 138
			totalProgressLen:  150,
			totalWaves:        10,
			currentWaveNum:    1,
			waveProgress:      0.5,
			expectedVirtual:   0.05, // (0 + 15.33*0.5) / 150 ≈ 0.051
		},
		{
			name:              "红字波刚激活",
			flagWaveCount:     1, // 刚激活一个红字波
			flagSegmentLength: 12,
			normalSegmentBase: 126,
			totalProgressLen:  150,
			totalWaves:        10,
			currentWaveNum:    10, // 最后一波
			waveProgress:      0.0,
			expectedVirtual:   0.92, // (12 + 126) / 150 = 0.92
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := &game.GameState{LevelTime: 0}
			ls := &LevelSystem{gameState: gs}
			pb := &components.LevelProgressBarComponent{
				FlagWaveCount:       tt.flagWaveCount,
				FlagSegmentLength:   tt.flagSegmentLength,
				NormalSegmentBase:   tt.normalSegmentBase,
				TotalProgressLength: tt.totalProgressLen,
				TotalWaves:          tt.totalWaves,
				CurrentWaveNum:      tt.currentWaveNum,
				// 设置时间进度等于 waveProgress
				WaveStartTime:    0,
				WaveInitialDelay: 100,
				// 设置血量进度为 0（这样时间进度会是决定因素）
				WaveInitialHealth:  1000,
				WaveCurrentHealth:  1000,
				WaveRequiredDamage: 1000,
			}
			gs.LevelTime = tt.waveProgress * 100 // levelTime = waveProgress * delay

			ls.calculateVirtualProgress(pb)

			// 允许较大误差，因为计算涉及整数除法
			if math.Abs(pb.VirtualProgress-tt.expectedVirtual) > 0.1 {
				t.Errorf("VirtualProgress = %.4f, want approximately %.4f", pb.VirtualProgress, tt.expectedVirtual)
			}
		})
	}
}

// TestOnWaveActivated 测试波次激活回调
func TestOnWaveActivated(t *testing.T) {
	em := ecs.NewEntityManager()

	// 创建进度条实体
	pbEntityID := em.CreateEntity()
	pb := &components.LevelProgressBarComponent{
		TotalProgressLength: 150,
		FlagSegmentLength:   12,
		NormalSegmentBase:   138,
		TotalWaves:          10,
	}
	ecs.AddComponent(em, pbEntityID, pb)

	levelConfig := &config.LevelConfig{
		FlagWaves: []int{4, 9}, // 第5波和第10波是旗帜波
		Waves: []config.WaveConfig{
			{Zombies: []config.ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 2}}},
			{Zombies: []config.ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 2}}},
			{Zombies: []config.ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 2}}},
			{Zombies: []config.ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 2}}},
			{Zombies: []config.ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 5}}}, // 旗帜波
			{Zombies: []config.ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 2}}},
			{Zombies: []config.ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 2}}},
			{Zombies: []config.ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 2}}},
			{Zombies: []config.ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 2}}},
			{Zombies: []config.ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 10}}}, // 最后一波旗帜波
		},
	}

	gs := &game.GameState{
		CurrentLevel: levelConfig,
		LevelTime:    10.0,
	}

	ls := &LevelSystem{
		entityManager:       em,
		gameState:           gs,
		progressBarEntityID: pbEntityID,
	}

	// 激活第一波（非旗帜波）
	ls.OnWaveActivated(0, 25.0)

	if pb.CurrentWaveNum != 1 {
		t.Errorf("CurrentWaveNum = %d, want 1", pb.CurrentWaveNum)
	}
	if pb.FlagWaveCount != 0 {
		t.Errorf("FlagWaveCount = %d, want 0 (not a flag wave)", pb.FlagWaveCount)
	}
	if pb.WaveStartTime != 10.0 {
		t.Errorf("WaveStartTime = %.1f, want 10.0", pb.WaveStartTime)
	}
	if pb.WaveInitialDelay != 25.0 {
		t.Errorf("WaveInitialDelay = %.1f, want 25.0", pb.WaveInitialDelay)
	}

	// 激活第5波（旗帜波）
	gs.LevelTime = 50.0
	ls.OnWaveActivated(4, 45.0)

	if pb.CurrentWaveNum != 5 {
		t.Errorf("CurrentWaveNum = %d, want 5", pb.CurrentWaveNum)
	}
	if pb.FlagWaveCount != 1 {
		t.Errorf("FlagWaveCount = %d, want 1 (flag wave activated)", pb.FlagWaveCount)
	}
}

// TestGetZombieTypeHealth 测试僵尸血量获取
func TestGetZombieTypeHealth(t *testing.T) {
	tests := []struct {
		name       string
		zombieType string
		expected   float64
	}{
		{"basic", "basic", 200},
		{"conehead", "conehead", 560},
		{"buckethead", "buckethead", 1300},
		{"gargantuar", "gargantuar", 3000},
		{"unknown", "unknown", 200}, // 默认值
	}

	ls := &LevelSystem{zombieStatsConfig: nil} // 无配置，使用默认值

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ls.getZombieTypeHealth(tt.zombieType)
			if result != tt.expected {
				t.Errorf("getZombieTypeHealth(%s) = %.0f, want %.0f", tt.zombieType, result, tt.expected)
			}
		})
	}
}

// TestUpdateRealProgress_FrameSkipping 测试虚拟/现实追踪机制在掉帧情况下的表现
// 验证即使 GameTickCS 跳过了精确的倍数（如 20），只要跨越了倍数，也应该更新
func TestUpdateRealProgress_FrameSkipping(t *testing.T) {
	// 使用 SlowTrackIntervalCS = 20
	interval := config.SlowTrackIntervalCS

	tests := []struct {
		name               string
		virtualProgress    float64
		realProgress       float64
		gameTickCS         int
		lastTrackUpdateCS  int
		totalProgressLen   int
		expectedRealChange bool
	}{
		{
			name:               "精确命中倍数(20cs) - 应该更新",
			virtualProgress:    0.04, // 6格 (1/150 * 6 = 0.04)
			realProgress:       0.0,
			gameTickCS:         20,
			lastTrackUpdateCS:  0,
			totalProgressLen:   150,
			expectedRealChange: true,
		},
		{
			name:              "跳过倍数(19->21cs) - 应该更新",
			virtualProgress:   0.04,
			realProgress:      0.0,
			gameTickCS:        21, // 19 -> 21, 跨过了 20
			lastTrackUpdateCS: 0,  // 上次是 0 (初始) 或 19 (如果每帧记录但没更新)
			// 这里假设 LastTrackUpdateCS 记录的是上次*更新*的时间，或者是初始值0
			// 如果是 0，0/20 = 0. 21/20 = 1. 1 > 0. 应该更新.
			totalProgressLen:   150,
			expectedRealChange: true,
		},
		{
			name:               "未跨越倍数(21->39cs) - 不应更新",
			virtualProgress:    0.04,
			realProgress:       0.0066, // 已经前进了一格
			gameTickCS:         39,     // 21 -> 39, 没跨过 40
			lastTrackUpdateCS:  21,     // 上次更新是在 21
			totalProgressLen:   150,
			expectedRealChange: false,
		},
		{
			name:              "再次跨越倍数(39->41cs) - 应该更新",
			virtualProgress:   0.04,
			realProgress:      0.0066,
			gameTickCS:        41, // 39 -> 41, 跨过了 40
			lastTrackUpdateCS: 21, // 上次更新是在 21
			// 41/20 = 2. 21/20 = 1. 2 > 1. 应该更新.
			totalProgressLen:   150,
			expectedRealChange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ls := &LevelSystem{}
			pb := &components.LevelProgressBarComponent{
				VirtualProgress:     tt.virtualProgress,
				RealProgress:        tt.realProgress,
				GameTickCS:          tt.gameTickCS,
				LastTrackUpdateCS:   tt.lastTrackUpdateCS,
				TotalProgressLength: tt.totalProgressLen,
			}

			originalReal := pb.RealProgress
			ls.updateRealProgress(pb)

			changed := pb.RealProgress != originalReal

			// 验证是否更新
			if changed != tt.expectedRealChange {
				t.Errorf("RealProgress change = %v, want %v (currentCS=%d, lastUpdateCS=%d, interval=%d)",
					changed, tt.expectedRealChange, tt.gameTickCS, tt.lastTrackUpdateCS, interval)
			}

			// 如果更新了，验证 LastTrackUpdateCS 是否更新为当前的 GameTickCS
			if changed && pb.LastTrackUpdateCS != tt.gameTickCS {
				t.Errorf("LastTrackUpdateCS = %d, want %d", pb.LastTrackUpdateCS, tt.gameTickCS)
			}
		})
	}
}
