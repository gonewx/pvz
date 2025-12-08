package utils

import (
	"testing"
)

func TestDragManagerInitialState(t *testing.T) {
	dm := GetDragManager()

	// 验证初始状态
	if dm.GetState() != DragStateNone {
		t.Errorf("Expected initial state to be DragStateNone, got %v", dm.GetState())
	}

	if dm.IsDragging() {
		t.Error("Expected IsDragging to be false initially")
	}

	if dm.JustStarted() {
		t.Error("Expected JustStarted to be false initially")
	}

	if dm.JustEnded() {
		t.Error("Expected JustEnded to be false initially")
	}
}

func TestDragManagerReset(t *testing.T) {
	dm := GetDragManager()

	// 模拟一些状态
	dm.info.State = DragStateDragging
	dm.info.StartX = 100
	dm.info.StartY = 200
	dm.info.CurrentX = 150
	dm.info.CurrentY = 250

	// 重置
	dm.Reset()

	// 验证重置后的状态
	if dm.GetState() != DragStateNone {
		t.Errorf("Expected state to be DragStateNone after reset, got %v", dm.GetState())
	}

	info := dm.GetInfo()
	if info.StartX != 0 || info.StartY != 0 {
		t.Errorf("Expected start position to be (0, 0) after reset, got (%d, %d)", info.StartX, info.StartY)
	}

	if info.CurrentX != 0 || info.CurrentY != 0 {
		t.Errorf("Expected current position to be (0, 0) after reset, got (%d, %d)", info.CurrentX, info.CurrentY)
	}

	if info.TouchID != -1 {
		t.Errorf("Expected TouchID to be -1 after reset, got %d", info.TouchID)
	}
}

func TestDragManagerGetDragDistance(t *testing.T) {
	dm := GetDragManager()

	// 设置起始和当前位置
	dm.info.StartX = 100
	dm.info.StartY = 200
	dm.info.CurrentX = 150
	dm.info.CurrentY = 280

	dx, dy := dm.GetDragDistance()

	if dx != 50 {
		t.Errorf("Expected dx to be 50, got %d", dx)
	}

	if dy != 80 {
		t.Errorf("Expected dy to be 80, got %d", dy)
	}
}

func TestDragManagerStateTransitions(t *testing.T) {
	dm := GetDragManager()
	dm.Reset()

	// 测试状态转换方法
	dm.info.State = DragStateStarted
	if !dm.JustStarted() {
		t.Error("Expected JustStarted to be true when state is DragStateStarted")
	}

	dm.info.State = DragStateDragging
	if !dm.IsDragging() {
		t.Error("Expected IsDragging to be true when state is DragStateDragging")
	}

	dm.info.State = DragStateEnded
	if !dm.JustEnded() {
		t.Error("Expected JustEnded to be true when state is DragStateEnded")
	}

	// 清理
	dm.Reset()
}

func TestDragManagerIsTouchDrag(t *testing.T) {
	dm := GetDragManager()
	dm.Reset()

	// 默认不是触摸拖拽
	if dm.IsTouchDrag() {
		t.Error("Expected IsTouchDrag to be false initially")
	}

	// 设置为触摸输入
	dm.info.IsTouchInput = true
	if !dm.IsTouchDrag() {
		t.Error("Expected IsTouchDrag to be true when IsTouchInput is true")
	}

	// 清理
	dm.Reset()
}

func TestDragManagerGetInfo(t *testing.T) {
	dm := GetDragManager()
	dm.Reset()

	// 设置一些值
	dm.info.State = DragStateDragging
	dm.info.StartX = 10
	dm.info.StartY = 20
	dm.info.CurrentX = 30
	dm.info.CurrentY = 40
	dm.info.IsTouchInput = true

	info := dm.GetInfo()

	if info.State != DragStateDragging {
		t.Errorf("Expected State to be DragStateDragging, got %v", info.State)
	}

	if info.StartX != 10 || info.StartY != 20 {
		t.Errorf("Expected start position to be (10, 20), got (%d, %d)", info.StartX, info.StartY)
	}

	if info.CurrentX != 30 || info.CurrentY != 40 {
		t.Errorf("Expected current position to be (30, 40), got (%d, %d)", info.CurrentX, info.CurrentY)
	}

	if !info.IsTouchInput {
		t.Error("Expected IsTouchInput to be true")
	}

	// 清理
	dm.Reset()
}
