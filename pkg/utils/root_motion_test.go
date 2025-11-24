package utils

import (
	"math"
	"testing"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
)

// TestCalculateRootMotionDelta_NormalMovement 测试正常帧间位移计算（含插值）
func TestCalculateRootMotionDelta_NormalMovement(t *testing.T) {
	// 准备测试数据：模拟 _ground 轨道
	x1, y1 := 10.0, 0.0
	x2, y2 := 25.0, 0.0

	groundFrames := []reanim.Frame{
		{X: &x1, Y: &y1}, // 帧 0
		{X: &x2, Y: &y2}, // 帧 1
	}

	comp := &components.ReanimComponent{
		MergedTracks: map[string][]reanim.Frame{
			"_ground": groundFrames,
		},
		CurrentFrame:      1,
		LastGroundX:       10.0, // 上一帧 X 坐标
		LastGroundY:       0.0,
		LastAnimFrame:     -1, // 初始化为 -1，表示尚未开始
		AccumulatedDeltaX: 0.0,
		AccumulatedDeltaY: 0.0,
	}

	// 执行第一次调用（动画帧变化）
	deltaX, deltaY, err := CalculateRootMotionDelta(comp, "_ground")

	// 验证
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// 期望值（插值后）：-(25.0 - 10.0) / 5 = -15.0 / 5 = -3.0
	// 因为假设动画 12 FPS，游戏 60 FPS，每个动画帧持续 5 个游戏帧
	expectedDeltaX := -3.0
	expectedDeltaY := 0.0

	if math.Abs(deltaX-expectedDeltaX) > 0.001 {
		t.Errorf("expected deltaX=%.2f, got %.2f", expectedDeltaX, deltaX)
	}
	if math.Abs(deltaY-expectedDeltaY) > 0.001 {
		t.Errorf("expected deltaY=%.2f, got %.2f", expectedDeltaY, deltaY)
	}

	// 验证 LastGroundX/Y 已更新
	if math.Abs(comp.LastGroundX-25.0) > 0.001 {
		t.Errorf("expected LastGroundX=25.0, got %.2f", comp.LastGroundX)
	}
	if math.Abs(comp.LastGroundY-0.0) > 0.001 {
		t.Errorf("expected LastGroundY=0.0, got %.2f", comp.LastGroundY)
	}

	// 验证累积位移（现在存储的是每帧固定值，而不是剩余值）
	// 期望值：-(25.0 - 10.0) / 5 = -15.0 / 5 = -3.0
	expectedAccumulated := -3.0
	if math.Abs(comp.AccumulatedDeltaX-expectedAccumulated) > 0.001 {
		t.Errorf("expected AccumulatedDeltaX=%.2f, got %.2f", expectedAccumulated, comp.AccumulatedDeltaX)
	}
}

// TestCalculateRootMotionDelta_LoopReset 测试动画循环重置（防瞬移）
func TestCalculateRootMotionDelta_LoopReset(t *testing.T) {
	// 准备测试数据：模拟动画循环（最后一帧 X=200 跳回第一帧 X=0）
	x1, y1 := 0.0, 0.0

	groundFrames := []reanim.Frame{
		{X: &x1, Y: &y1}, // 帧 0：循环开始
	}

	comp := &components.ReanimComponent{
		MergedTracks: map[string][]reanim.Frame{
			"_ground": groundFrames,
		},
		CurrentFrame: 0,
		LastGroundX:  200.0, // 上一帧 X=200（循环结束时的值）
		LastGroundY:  0.0,
	}

	// 执行
	deltaX, deltaY, err := CalculateRootMotionDelta(comp, "_ground")

	// 验证：应该检测到瞬移并返回 0
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if deltaX != 0 {
		t.Errorf("expected deltaX=0 (loop reset detected), got %.2f", deltaX)
	}
	if deltaY != 0 {
		t.Errorf("expected deltaY=0 (loop reset detected), got %.2f", deltaY)
	}
}

// TestCalculateRootMotionDelta_MissingTrack 测试 _ground 轨道不存在
func TestCalculateRootMotionDelta_MissingTrack(t *testing.T) {
	// 准备测试数据：没有 _ground 轨道
	comp := &components.ReanimComponent{
		MergedTracks: map[string][]reanim.Frame{
			"other_track": {},
		},
		CurrentFrame: 0,
		LastGroundX:  0.0,
		LastGroundY:  0.0,
	}

	// 执行
	deltaX, deltaY, err := CalculateRootMotionDelta(comp, "_ground")

	// 验证：应该返回错误
	if err == nil {
		t.Error("expected error for missing track, got nil")
	}

	if deltaX != 0 || deltaY != 0 {
		t.Errorf("expected deltaX=0, deltaY=0 on error, got (%.2f, %.2f)", deltaX, deltaY)
	}
}

// TestCalculateRootMotionDelta_NilComponent 测试 nil 组件
func TestCalculateRootMotionDelta_NilComponent(t *testing.T) {
	// 执行
	deltaX, deltaY, err := CalculateRootMotionDelta(nil, "_ground")

	// 验证：应该返回错误
	if err == nil {
		t.Error("expected error for nil component, got nil")
	}

	if deltaX != 0 || deltaY != 0 {
		t.Errorf("expected deltaX=0, deltaY=0 on error, got (%.2f, %.2f)", deltaX, deltaY)
	}
}

// TestCalculateRootMotionDelta_NilMergedTracks 测试 MergedTracks 为 nil
func TestCalculateRootMotionDelta_NilMergedTracks(t *testing.T) {
	comp := &components.ReanimComponent{
		MergedTracks: nil,
		CurrentFrame: 0,
	}

	// 执行
	deltaX, deltaY, err := CalculateRootMotionDelta(comp, "_ground")

	// 验证：应该返回错误
	if err == nil {
		t.Error("expected error for nil MergedTracks, got nil")
	}

	if deltaX != 0 || deltaY != 0 {
		t.Errorf("expected deltaX=0, deltaY=0 on error, got (%.2f, %.2f)", deltaX, deltaY)
	}
}

// TestCalculateRootMotionDelta_FrameIndexOutOfRange 测试帧索引越界
func TestCalculateRootMotionDelta_FrameIndexOutOfRange(t *testing.T) {
	x1, y1 := 10.0, 0.0

	groundFrames := []reanim.Frame{
		{X: &x1, Y: &y1}, // 只有帧 0
	}

	comp := &components.ReanimComponent{
		MergedTracks: map[string][]reanim.Frame{
			"_ground": groundFrames,
		},
		CurrentFrame: 5, // 越界
		LastGroundX:  0.0,
		LastGroundY:  0.0,
	}

	// 执行
	deltaX, deltaY, err := CalculateRootMotionDelta(comp, "_ground")

	// 验证：应该返回错误
	if err == nil {
		t.Error("expected error for out of range frame index, got nil")
	}

	if deltaX != 0 || deltaY != 0 {
		t.Errorf("expected deltaX=0, deltaY=0 on error, got (%.2f, %.2f)", deltaX, deltaY)
	}
}

// TestGetGroundPosition_EmptyFrameInheritance 测试空帧继承处理
// MergedTracks 已经处理了帧继承，所以 nil 值表示该帧没有设置值（使用默认值 0）
func TestGetGroundPosition_EmptyFrameInheritance(t *testing.T) {
	// 准备测试数据：帧 1 没有 X 值（nil）
	x1, y1 := 50.0, 0.0

	groundFrames := []reanim.Frame{
		{X: &x1, Y: &y1}, // 帧 0：X=50
		{X: nil, Y: nil}, // 帧 1：空帧（MergedTracks 处理后 nil 表示默认值）
	}

	// 执行
	x, y := getGroundPosition(groundFrames, 1)

	// 验证：nil 值应返回默认值 0
	if x != 0 {
		t.Errorf("expected x=0 for nil X, got %.2f", x)
	}
	if y != 0 {
		t.Errorf("expected y=0 for nil Y, got %.2f", y)
	}
}

// TestGetGroundPosition_ValidFrame 测试有效帧读取
func TestGetGroundPosition_ValidFrame(t *testing.T) {
	x1, y1 := 30.0, 40.0

	groundFrames := []reanim.Frame{
		{X: &x1, Y: &y1},
	}

	// 执行
	x, y := getGroundPosition(groundFrames, 0)

	// 验证
	if math.Abs(x-30.0) > 0.001 {
		t.Errorf("expected x=30.0, got %.2f", x)
	}
	if math.Abs(y-40.0) > 0.001 {
		t.Errorf("expected y=40.0, got %.2f", y)
	}
}

// TestGetGroundPosition_OutOfRange 测试越界帧索引
func TestGetGroundPosition_OutOfRange(t *testing.T) {
	x1, y1 := 10.0, 20.0

	groundFrames := []reanim.Frame{
		{X: &x1, Y: &y1},
	}

	// 执行：负索引
	x, y := getGroundPosition(groundFrames, -1)
	if x != 0 || y != 0 {
		t.Errorf("expected (0, 0) for negative index, got (%.2f, %.2f)", x, y)
	}

	// 执行：超出范围
	x, y = getGroundPosition(groundFrames, 10)
	if x != 0 || y != 0 {
		t.Errorf("expected (0, 0) for out of range index, got (%.2f, %.2f)", x, y)
	}
}

// TestGetGroundTrackFrameCount 测试获取帧数
func TestGetGroundTrackFrameCount(t *testing.T) {
	x1, y1 := 10.0, 0.0
	x2, y2 := 20.0, 0.0

	groundFrames := []reanim.Frame{
		{X: &x1, Y: &y1},
		{X: &x2, Y: &y2},
	}

	comp := &components.ReanimComponent{
		MergedTracks: map[string][]reanim.Frame{
			"_ground": groundFrames,
		},
	}

	// 执行
	count := GetGroundTrackFrameCount(comp, "_ground")

	// 验证
	if count != 2 {
		t.Errorf("expected frame count=2, got %d", count)
	}

	// 测试不存在的轨道
	count = GetGroundTrackFrameCount(comp, "_nonexistent")
	if count != 0 {
		t.Errorf("expected frame count=0 for nonexistent track, got %d", count)
	}

	// 测试 nil 组件
	count = GetGroundTrackFrameCount(nil, "_ground")
	if count != 0 {
		t.Errorf("expected frame count=0 for nil component, got %d", count)
	}
}

// TestCalculateRootMotionDelta_SmallMovement 测试小幅度移动（不触发瞬移检测，含插值）
func TestCalculateRootMotionDelta_SmallMovement(t *testing.T) {
	// 准备测试数据：小幅度移动（5 像素）
	x1, y1 := 10.0, 5.0
	x2, y2 := 15.0, 5.0

	groundFrames := []reanim.Frame{
		{X: &x1, Y: &y1},
		{X: &x2, Y: &y2},
	}

	comp := &components.ReanimComponent{
		MergedTracks: map[string][]reanim.Frame{
			"_ground": groundFrames,
		},
		CurrentFrame:      1,
		LastGroundX:       10.0,
		LastGroundY:       5.0,
		LastAnimFrame:     -1, // 初始化为 -1
		AccumulatedDeltaX: 0.0,
		AccumulatedDeltaY: 0.0,
	}

	// 执行
	deltaX, deltaY, err := CalculateRootMotionDelta(comp, "_ground")

	// 验证：小幅度移动不应触发瞬移检测
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	expectedDeltaX := -1.0 // -(15 - 10) / 5 = -5.0 / 5
	expectedDeltaY := 0.0  // -(5 - 5) / 5

	if math.Abs(deltaX-expectedDeltaX) > 0.001 {
		t.Errorf("expected deltaX=%.2f, got %.2f", expectedDeltaX, deltaX)
	}
	if math.Abs(deltaY-expectedDeltaY) > 0.001 {
		t.Errorf("expected deltaY=%.2f, got %.2f", expectedDeltaY, deltaY)
	}
}
