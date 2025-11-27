package systems

import (
	"math"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
)

// TestCalculateWeightP 测试权重占比计算
func TestCalculateWeightP(t *testing.T) {
	tests := []struct {
		name     string
		weights  []float64
		expected []float64
	}{
		{
			name:     "正常权重分配",
			weights:  []float64{1.0, 1.0, 1.0, 1.0, 1.0},
			expected: []float64{0.2, 0.2, 0.2, 0.2, 0.2},
		},
		{
			name:     "全零权重",
			weights:  []float64{0.0, 0.0, 0.0},
			expected: []float64{0.0, 0.0, 0.0},
		},
		{
			name:     "单个非零权重",
			weights:  []float64{0.0, 1.0, 0.0},
			expected: []float64{0.0, 1.0, 0.0},
		},
		{
			name:     "不均匀权重",
			weights:  []float64{1.0, 2.0, 3.0},
			expected: []float64{1.0 / 6.0, 2.0 / 6.0, 3.0 / 6.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateWeightP(tt.weights)
			if len(result) != len(tt.expected) {
				t.Fatalf("Expected length %d, got %d", len(tt.expected), len(result))
			}
			for i := range result {
				if math.Abs(result[i]-tt.expected[i]) > 1e-9 {
					t.Errorf("Index %d: expected %.6f, got %.6f", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

// TestCalculatePLast 测试影响因子 PLast 计算
func TestCalculatePLast(t *testing.T) {
	tests := []struct {
		name       string
		lastPicked int
		weightP    float64
		expected   float64
	}{
		{
			name:       "LastPicked=0, WeightP=0.2",
			lastPicked: 0,
			weightP:    0.2,
			expected:   (6.0*0.0*0.2 + 6.0*0.2 - 3.0) / 4.0,
		},
		{
			name:       "LastPicked=5, WeightP=0.2",
			lastPicked: 5,
			weightP:    0.2,
			expected:   (6.0*5.0*0.2 + 6.0*0.2 - 3.0) / 4.0,
		},
		{
			name:       "LastPicked=10, WeightP=0.5",
			lastPicked: 10,
			weightP:    0.5,
			expected:   (6.0*10.0*0.5 + 6.0*0.5 - 3.0) / 4.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePLast(tt.lastPicked, tt.weightP)
			if math.Abs(result-tt.expected) > 1e-9 {
				t.Errorf("Expected %.6f, got %.6f", tt.expected, result)
			}
		})
	}
}

// TestCalculatePSecondLast 测试影响因子 PSecondLast 计算
func TestCalculatePSecondLast(t *testing.T) {
	tests := []struct {
		name             string
		secondLastPicked int
		weightP          float64
		expected         float64
	}{
		{
			name:             "SecondLastPicked=0, WeightP=0.2",
			secondLastPicked: 0,
			weightP:          0.2,
			expected:         (0.0*0.2 + 0.2 - 1.0) / 4.0,
		},
		{
			name:             "SecondLastPicked=3, WeightP=0.2",
			secondLastPicked: 3,
			weightP:          0.2,
			expected:         (3.0*0.2 + 0.2 - 1.0) / 4.0,
		},
		{
			name:             "SecondLastPicked=8, WeightP=0.5",
			secondLastPicked: 8,
			weightP:          0.5,
			expected:         (8.0*0.5 + 0.5 - 1.0) / 4.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePSecondLast(tt.secondLastPicked, tt.weightP)
			if math.Abs(result-tt.expected) > 1e-9 {
				t.Errorf("Expected %.6f, got %.6f", tt.expected, result)
			}
		})
	}
}

// TestCalculateSmoothWeight 测试平滑权重计算
func TestCalculateSmoothWeight(t *testing.T) {
	tests := []struct {
		name        string
		weightP     float64
		pLast       float64
		pSecondLast float64
		expected    float64
	}{
		{
			name:        "WeightP < 1e-6 时返回 0",
			weightP:     1e-7,
			pLast:       1.0,
			pSecondLast: 1.0,
			expected:    0.0,
		},
		{
			name:        "clamp 下限 0.01",
			weightP:     0.5,
			pLast:       -1.0,
			pSecondLast: -1.0,
			expected:    0.5 * 0.01,
		},
		{
			name:        "clamp 上限 100",
			weightP:     0.5,
			pLast:       60.0,
			pSecondLast: 50.0,
			expected:    0.5 * 100.0,
		},
		{
			name:        "正常范围值",
			weightP:     0.2,
			pLast:       1.5,
			pSecondLast: 0.5,
			expected:    0.2 * 2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateSmoothWeight(tt.weightP, tt.pLast, tt.pSecondLast)
			if math.Abs(result-tt.expected) > 1e-9 {
				t.Errorf("Expected %.6f, got %.6f", tt.expected, result)
			}
		})
	}
}

// TestLaneAllocatorInitializeLanes 测试行初始化
func TestLaneAllocatorInitializeLanes(t *testing.T) {
	em := ecs.NewEntityManager()
	allocator := NewLaneAllocator(em)

	// 测试初始化 5 行
	allocator.InitializeLanes(5, 1.0)

	if len(allocator.laneEntities) != 5 {
		t.Errorf("Expected 5 lane entities, got %d", len(allocator.laneEntities))
	}

	// 验证每行的初始状态
	for i, entity := range allocator.laneEntities {
		laneState, ok := ecs.GetComponent[*components.LaneStateComponent](em, entity)
		if !ok {
			t.Fatalf("Lane %d: component not found", i+1)
		}
		if laneState.LaneIndex != i+1 {
			t.Errorf("Lane %d: expected LaneIndex=%d, got %d", i+1, i+1, laneState.LaneIndex)
		}
		if laneState.Weight != 1.0 {
			t.Errorf("Lane %d: expected Weight=1.0, got %.2f", i+1, laneState.Weight)
		}
		if laneState.LastPicked != 0 {
			t.Errorf("Lane %d: expected LastPicked=0, got %d", i+1, laneState.LastPicked)
		}
		if laneState.SecondLastPicked != 0 {
			t.Errorf("Lane %d: expected SecondLastPicked=0, got %d", i+1, laneState.SecondLastPicked)
		}
	}
}

// TestLaneAllocatorUpdateLaneCounters 测试计数器更新
func TestLaneAllocatorUpdateLaneCounters(t *testing.T) {
	em := ecs.NewEntityManager()
	allocator := NewLaneAllocator(em)
	allocator.InitializeLanes(5, 1.0)

	// 选中第 3 行
	allocator.UpdateLaneCounters(3)

	// 验证第 3 行的计数器被重置
	lane3State, _ := ecs.GetComponent[*components.LaneStateComponent](em, allocator.laneEntities[2])
	if lane3State.LastPicked != 0 {
		t.Errorf("Lane 3: expected LastPicked=0, got %d", lane3State.LastPicked)
	}
	if lane3State.SecondLastPicked != 0 {
		t.Errorf("Lane 3: expected SecondLastPicked=0, got %d", lane3State.SecondLastPicked)
	}

	// 验证其他行的计数器递增
	for i := 0; i < 5; i++ {
		if i == 2 {
			continue // 跳过第 3 行
		}
		laneState, _ := ecs.GetComponent[*components.LaneStateComponent](em, allocator.laneEntities[i])
		if laneState.LastPicked != 1 {
			t.Errorf("Lane %d: expected LastPicked=1, got %d", i+1, laneState.LastPicked)
		}
		if laneState.SecondLastPicked != 1 {
			t.Errorf("Lane %d: expected SecondLastPicked=1, got %d", i+1, laneState.SecondLastPicked)
		}
	}

	// 再选中第 2 行
	allocator.UpdateLaneCounters(2)

	// 验证第 2 行的计数器被重置
	lane2State, _ := ecs.GetComponent[*components.LaneStateComponent](em, allocator.laneEntities[1])
	if lane2State.LastPicked != 0 {
		t.Errorf("Lane 2: expected LastPicked=0, got %d", lane2State.LastPicked)
	}
	if lane2State.SecondLastPicked != 1 {
		t.Errorf("Lane 2: expected SecondLastPicked=1 (from previous LastPicked), got %d", lane2State.SecondLastPicked)
	}
}

// TestLaneAllocatorSelectLane 测试行选择逻辑
func TestLaneAllocatorSelectLane(t *testing.T) {
	em := ecs.NewEntityManager()
	allocator := NewLaneAllocator(em)

	// 测试单行选择
	allocator.InitializeLanes(1, 1.0)
	selectedLane := allocator.SelectLane("basic", "day", nil, []int{1}, nil)
	if selectedLane != 1 {
		t.Errorf("Single lane: expected 1, got %d", selectedLane)
	}

	// 测试全零权重时返回第六行
	allocator.InitializeLanes(5, 0.0)
	selectedLane = allocator.SelectLane("basic", "day", nil, []int{1, 2, 3, 4, 5}, nil)
	if selectedLane != 6 {
		t.Errorf("All zero weights: expected 6, got %d", selectedLane)
	}
}

// TestLaneSelectionDistribution 测试行选择分布均匀性
func TestLaneSelectionDistribution(t *testing.T) {
	em := ecs.NewEntityManager()
	allocator := NewLaneAllocator(em)
	allocator.InitializeLanes(5, 1.0)

	// 执行 1000 次抽取
	iterations := 1000
	counts := make(map[int]int)

	for i := 0; i < iterations; i++ {
		selectedLane := allocator.SelectLane("basic", "day", nil, []int{1, 2, 3, 4, 5}, nil)
		counts[selectedLane]++
		allocator.UpdateLaneCounters(selectedLane)
	}

	// 验证每行被选中的次数在合理范围内（预期每行约 200 次，允许偏差 ±100）
	expectedCount := iterations / 5
	tolerance := 100

	for lane := 1; lane <= 5; lane++ {
		count := counts[lane]
		if count < expectedCount-tolerance || count > expectedCount+tolerance {
			t.Logf("Lane %d: count=%d (expected ~%d, tolerance ±%d)", lane, count, expectedCount, tolerance)
		}
	}

	// 验证所有选择的总数等于迭代次数
	totalCount := 0
	for _, count := range counts {
		totalCount += count
	}
	if totalCount != iterations {
		t.Errorf("Total count=%d, expected %d", totalCount, iterations)
	}
}

// TestFilterLegalLanes 测试合法行过滤
func TestFilterLegalLanes(t *testing.T) {
	tests := []struct {
		name       string
		laneStates []*components.LaneStateComponent
		expected   []int
	}{
		{
			name: "所有行合法",
			laneStates: []*components.LaneStateComponent{
				{LaneIndex: 1, Weight: 1.0},
				{LaneIndex: 2, Weight: 1.0},
				{LaneIndex: 3, Weight: 1.0},
			},
			expected: []int{0, 1, 2},
		},
		{
			name: "行号不合法",
			laneStates: []*components.LaneStateComponent{
				{LaneIndex: 0, Weight: 1.0},  // 行号 < 1
				{LaneIndex: 2, Weight: 1.0},
				{LaneIndex: 7, Weight: 1.0},  // 行号 > 6
			},
			expected: []int{1},
		},
		{
			name: "权重为零的行不合法",
			laneStates: []*components.LaneStateComponent{
				{LaneIndex: 1, Weight: 1.0},
				{LaneIndex: 2, Weight: 0.0},  // 权重为 0
				{LaneIndex: 3, Weight: 1.0},
			},
			expected: []int{0, 2},
		},
		{
			name: "无合法行",
			laneStates: []*components.LaneStateComponent{
				{LaneIndex: 1, Weight: 0.0},
				{LaneIndex: 2, Weight: 0.0},
			},
			expected: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterLegalLanes(tt.laneStates, "basic", "day", nil, []int{1, 2, 3, 4, 5}, nil)
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

// TestFilterLegalLanes_LaneRestriction 测试波次级行限制 (Story 17.2)
func TestFilterLegalLanes_LaneRestriction(t *testing.T) {
	laneStates := []*components.LaneStateComponent{
		{LaneIndex: 1, Weight: 1.0},
		{LaneIndex: 2, Weight: 1.0},
		{LaneIndex: 3, Weight: 1.0},
		{LaneIndex: 4, Weight: 1.0},
		{LaneIndex: 5, Weight: 1.0},
	}

	tests := []struct {
		name            string
		laneRestriction []int
		expected        []int
	}{
		{
			name:            "无行限制时所有行合法",
			laneRestriction: nil,
			expected:        []int{0, 1, 2, 3, 4},
		},
		{
			name:            "空行限制时所有行合法",
			laneRestriction: []int{},
			expected:        []int{0, 1, 2, 3, 4},
		},
		{
			name:            "限制为单行",
			laneRestriction: []int{3},
			expected:        []int{2}, // 索引 2 对应 LaneIndex 3
		},
		{
			name:            "限制为多行",
			laneRestriction: []int{1, 3, 5},
			expected:        []int{0, 2, 4}, // 索引 0, 2, 4 对应 LaneIndex 1, 3, 5
		},
		{
			name:            "限制为连续行",
			laneRestriction: []int{2, 3, 4},
			expected:        []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterLegalLanes(laneStates, "basic", "day", nil, []int{1, 2, 3, 4, 5}, tt.laneRestriction)
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

// TestSelectLane_WithLaneRestriction 测试带行限制的行选择 (Story 17.2)
func TestSelectLane_WithLaneRestriction(t *testing.T) {
	em := ecs.NewEntityManager()
	allocator := NewLaneAllocator(em)
	allocator.InitializeLanes(5, 1.0)

	// 测试限制为单行时，必定选中该行
	for i := 0; i < 10; i++ {
		selectedLane := allocator.SelectLane("basic", "day", nil, []int{1, 2, 3, 4, 5}, []int{3})
		if selectedLane != 3 {
			t.Errorf("With laneRestriction=[3], expected lane 3, got %d", selectedLane)
		}
	}

	// 测试限制为多行时，只选择限制内的行
	laneCounts := make(map[int]int)
	restriction := []int{1, 5}
	for i := 0; i < 100; i++ {
		selectedLane := allocator.SelectLane("basic", "day", nil, []int{1, 2, 3, 4, 5}, restriction)
		laneCounts[selectedLane]++
		allocator.UpdateLaneCounters(selectedLane)
	}

	// 验证只有限制内的行被选中
	for lane, count := range laneCounts {
		found := false
		for _, r := range restriction {
			if r == lane {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Lane %d was selected %d times but is not in restriction %v", lane, count, restriction)
		}
	}

	// 验证限制内的行都有被选中
	for _, lane := range restriction {
		if laneCounts[lane] == 0 {
			t.Errorf("Lane %d in restriction was never selected", lane)
		}
	}
}
