package systems

import (
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
)

// TestIsWaterLane 测试水路行判定
func TestIsWaterLane(t *testing.T) {
	waterLaneConfig := map[string][]int{
		"pool": {3, 4},
		"fog":  {3, 4},
	}

	tests := []struct {
		name      string
		laneIndex int
		sceneType string
		expected  bool
	}{
		{"泳池场景第3行是水路", 3, "pool", true},
		{"泳池场景第4行是水路", 4, "pool", true},
		{"泳池场景第1行不是水路", 1, "pool", false},
		{"雾夜场景第3行是水路", 3, "fog", true},
		{"白天场景无水路", 3, "day", false},
		{"未知场景类型", 3, "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWaterLane(tt.laneIndex, tt.sceneType, waterLaneConfig)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestIsWaterZombie 测试水路僵尸判定
func TestIsWaterZombie(t *testing.T) {
	waterZombies := []string{"snorkel", "dolphinrider", "ducky"}

	tests := []struct {
		name       string
		zombieType string
		expected   bool
	}{
		{"潜水僵尸是水路僵尸", "snorkel", true},
		{"海豚骑士是水路僵尸", "dolphinrider", true},
		{"鸭子救生圈是水路僵尸", "ducky", true},
		{"普通僵尸不是水路僵尸", "basic", false},
		{"路障僵尸不是水路僵尸", "conehead", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWaterZombie(tt.zombieType, waterZombies)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestIsAdjacentLaneValid 测试相邻行有效性判定
func TestIsAdjacentLaneValid(t *testing.T) {
	tests := []struct {
		name         string
		laneIndex    int
		enabledLanes []int
		expected     bool
	}{
		{"上下都有效", 3, []int{1, 2, 3, 4, 5}, true},
		{"仅上有效", 3, []int{1, 2, 3}, true},
		{"仅下有效", 3, []int{3, 4, 5}, true},
		{"上下都无效", 3, []int{1, 3, 5}, false},
		{"第一行（仅下有效）", 1, []int{1, 2, 3}, true},
		{"第一行（下无效）", 1, []int{1}, false},
		{"最后一行（仅上有效）", 5, []int{3, 4, 5}, true},
		{"空启用列表", 3, []int{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAdjacentLaneValid(tt.laneIndex, tt.enabledLanes)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestFilterLegalLanes_WaterRestrictions 测试水路限制
func TestFilterLegalLanes_WaterRestrictions(t *testing.T) {
	spawnRules := &config.SpawnRulesConfig{
		SceneTypeRestrictions: config.SceneRestrictions{
			WaterZombies: []string{"snorkel", "dolphinrider", "ducky"},
			WaterLaneConfig: map[string][]int{
				"pool": {3, 4},
				"fog":  {3, 4},
			},
		},
	}

	laneStates := []*components.LaneStateComponent{
		{LaneIndex: 1, Weight: 1.0},
		{LaneIndex: 2, Weight: 1.0},
		{LaneIndex: 3, Weight: 1.0}, // 水路
		{LaneIndex: 4, Weight: 1.0}, // 水路
		{LaneIndex: 5, Weight: 1.0},
	}

	tests := []struct {
		name       string
		zombieType string
		sceneType  string
		expected   []int
	}{
		{"水路僵尸只能在水路行（泳池场景）", "snorkel", "pool", []int{2, 3}},
		{"非水路僵尸不能在水路行（泳池场景）", "basic", "pool", []int{0, 1, 4}},
		{"非泳池场景无水路限制", "snorkel", "day", []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterLegalLanes(laneStates, tt.zombieType, tt.sceneType, spawnRules, []int{1, 2, 3, 4, 5}, nil)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("Index %d: expected %d, got %d", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

// TestFilterLegalLanes_DancingRestrictions 测试舞王限制
func TestFilterLegalLanes_DancingRestrictions(t *testing.T) {
	spawnRules := &config.SpawnRulesConfig{
		SceneTypeRestrictions: config.SceneRestrictions{
			DancingRestrictions: config.DancingRestrictions{
				ProhibitedScenes:      []string{"roof"},
				RequiresAdjacentLanes: true,
			},
		},
	}

	laneStates := []*components.LaneStateComponent{
		{LaneIndex: 1, Weight: 1.0},
		{LaneIndex: 2, Weight: 1.0},
		{LaneIndex: 3, Weight: 1.0},
		{LaneIndex: 4, Weight: 1.0},
		{LaneIndex: 5, Weight: 1.0},
	}

	tests := []struct {
		name         string
		sceneType    string
		enabledLanes []int
		expected     []int
	}{
		{"屋顶场景舞王被完全过滤", "roof", []int{1, 2, 3, 4, 5}, []int{}},
		{"白天场景舞王需要相邻行", "day", []int{1, 2, 3, 4, 5}, []int{0, 1, 2, 3, 4}},
		// 白天场景 enabledLanes=[1,3,5]，舞王需要相邻行在 enabledLanes 中
		// 第1行: 相邻行2不在enabledLanes，无效
		// 第3行: 相邻行2,4不在enabledLanes，无效
		// 第5行: 相邻行4不在enabledLanes，无效
		// 所以没有合法行
		{"白天场景部分行无相邻行被过滤", "day", []int{1, 3, 5}, []int{}},
		// 泳池场景舞王无相邻限制，但 enabledLanes 仍然过滤行
		// 只有第1、3、5行（索引0、2、4）在 enabledLanes 中
		{"泳池场景舞王无相邻限制", "pool", []int{1, 3, 5}, []int{0, 2, 4}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterLegalLanes(laneStates, "dancing", tt.sceneType, spawnRules, tt.enabledLanes, nil)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("Index %d: expected %d, got %d", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

// TestFilterLegalLanes_EnabledLanes 测试关卡级行限制（enabledLanes）
// 这是修复实时生成僵尸不按可用网格行生成的关键测试
func TestFilterLegalLanes_EnabledLanes(t *testing.T) {
	laneStates := []*components.LaneStateComponent{
		{LaneIndex: 1, Weight: 1.0},
		{LaneIndex: 2, Weight: 1.0},
		{LaneIndex: 3, Weight: 1.0},
		{LaneIndex: 4, Weight: 1.0},
		{LaneIndex: 5, Weight: 1.0},
	}

	tests := []struct {
		name         string
		enabledLanes []int
		expected     []int // 期望的合法行索引
	}{
		// 当 enabledLanes 为空时，所有行合法
		{"空 enabledLanes 时所有行合法", []int{}, []int{0, 1, 2, 3, 4}},
		// 只有中间三行启用（模拟 level-1-2）
		{"只启用中间三行 [2,3,4]", []int{2, 3, 4}, []int{1, 2, 3}},
		// 只有单行启用（模拟 level-1-1）
		{"只启用第三行 [3]", []int{3}, []int{2}},
		// 所有行启用
		{"所有行启用 [1,2,3,4,5]", []int{1, 2, 3, 4, 5}, []int{0, 1, 2, 3, 4}},
		// 只启用边缘行
		{"只启用边缘行 [1,5]", []int{1, 5}, []int{0, 4}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterLegalLanes(laneStates, "basic", "day", nil, tt.enabledLanes, nil)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("Index %d: expected %d, got %d", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

// TestFilterLegalLanes_Scene6thRow 测试第6行场景限制
func TestFilterLegalLanes_Scene6thRow(t *testing.T) {
	laneStates := []*components.LaneStateComponent{
		{LaneIndex: 5, Weight: 1.0},
		{LaneIndex: 6, Weight: 1.0},
	}

	tests := []struct {
		name      string
		sceneType string
		expected  []int
	}{
		{"泳池场景第6行合法", "pool", []int{0, 1}},
		{"雾夜场景第6行合法", "fog", []int{0, 1}},
		{"白天场景第6行被过滤", "day", []int{0}},
		{"屋顶场景第6行被过滤", "roof", []int{0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterLegalLanes(laneStates, "basic", tt.sceneType, nil, []int{5, 6}, nil)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("Index %d: expected %d, got %d", i, tt.expected[i], result[i])
				}
			}
		})
	}
}
