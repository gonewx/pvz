package components

import "testing"

// TestLawnmowerComponent 测试除草车组件的基本属性
func TestLawnmowerComponent(t *testing.T) {
	lawnmower := &LawnmowerComponent{
		Lane:        3,
		IsTriggered: false,
		IsMoving:    false,
		Speed:       300.0,
	}

	// 验证初始状态
	if lawnmower.Lane != 3 {
		t.Errorf("Expected Lane to be 3, got %d", lawnmower.Lane)
	}
	if lawnmower.IsTriggered {
		t.Error("Expected IsTriggered to be false initially")
	}
	if lawnmower.IsMoving {
		t.Error("Expected IsMoving to be false initially")
	}
	if lawnmower.Speed != 300.0 {
		t.Errorf("Expected Speed to be 300.0, got %.1f", lawnmower.Speed)
	}

	// 模拟触发
	lawnmower.IsTriggered = true
	lawnmower.IsMoving = true

	if !lawnmower.IsTriggered {
		t.Error("Expected IsTriggered to be true after trigger")
	}
	if !lawnmower.IsMoving {
		t.Error("Expected IsMoving to be true after trigger")
	}
}

// TestLawnmowerStateComponent 测试全局除草车状态组件
func TestLawnmowerStateComponent(t *testing.T) {
	state := &LawnmowerStateComponent{
		UsedLanes: make(map[int]bool),
	}

	// 验证初始状态为空
	if len(state.UsedLanes) != 0 {
		t.Errorf("Expected UsedLanes to be empty initially, got %d entries", len(state.UsedLanes))
	}

	// 标记某些行为已使用
	state.UsedLanes[1] = true
	state.UsedLanes[3] = true

	// 验证标记的行
	if !state.UsedLanes[1] {
		t.Error("Expected lane 1 to be marked as used")
	}
	if !state.UsedLanes[3] {
		t.Error("Expected lane 3 to be marked as used")
	}
	if state.UsedLanes[2] {
		t.Error("Expected lane 2 to be not used")
	}
	if state.UsedLanes[4] {
		t.Error("Expected lane 4 to be not used")
	}
	if state.UsedLanes[5] {
		t.Error("Expected lane 5 to be not used")
	}

	// 验证总数
	usedCount := 0
	for _, used := range state.UsedLanes {
		if used {
			usedCount++
		}
	}
	if usedCount != 2 {
		t.Errorf("Expected 2 lanes to be marked as used, got %d", usedCount)
	}
}
