package components

import (
	"testing"

	"github.com/gonewx/pvz/internal/reanim"
	"github.com/hajimehoshi/ebiten/v2"
)

// TestRenderPartDataStructure 测试 RenderPartData 结构体定义（Story 13.4 Task 1）
func TestRenderPartDataStructure(t *testing.T) {
	// 创建测试图片
	img := ebiten.NewImage(100, 100)

	// 创建测试帧数据（使用指针）
	x := 10.0
	y := 20.0
	scaleX := 1.0
	scaleY := 1.0
	frame := reanim.Frame{
		X:      &x,
		Y:      &y,
		ScaleX: &scaleX,
		ScaleY: &scaleY,
	}

	// 创建 RenderPartData 实例
	data := RenderPartData{
		Img:     img,
		Frame:   frame,
		OffsetX: 5.0,
		OffsetY: 10.0,
	}

	// 验证字段能够正确访问
	if data.Img != img {
		t.Errorf("Expected Img to be %v, got %v", img, data.Img)
	}

	if data.Frame.X != frame.X {
		t.Errorf("Expected Frame.X to be %v, got %v", frame.X, data.Frame.X)
	}

	if *data.Frame.X != 10.0 {
		t.Errorf("Expected *Frame.X to be 10.0, got %f", *data.Frame.X)
	}

	if data.OffsetX != 5.0 {
		t.Errorf("Expected OffsetX to be 5.0, got %f", data.OffsetX)
	}

	if data.OffsetY != 10.0 {
		t.Errorf("Expected OffsetY to be 10.0, got %f", data.OffsetY)
	}
}

// TestReanimComponentCacheFields 测试 ReanimComponent 的缓存字段（Story 13.4 Task 1）
func TestReanimComponentCacheFields(t *testing.T) {
	// 创建 ReanimComponent 实例
	comp := &ReanimComponent{
		CachedRenderData: make([]RenderPartData, 0, 10),
		LastRenderFrame:  -1,
	}

	// 验证初始状态
	if comp.CachedRenderData == nil {
		t.Error("Expected CachedRenderData to be initialized, got nil")
	}

	if len(comp.CachedRenderData) != 0 {
		t.Errorf("Expected CachedRenderData length to be 0, got %d", len(comp.CachedRenderData))
	}

	if cap(comp.CachedRenderData) != 10 {
		t.Errorf("Expected CachedRenderData capacity to be 10, got %d", cap(comp.CachedRenderData))
	}

	if comp.LastRenderFrame != -1 {
		t.Errorf("Expected LastRenderFrame to be -1, got %d", comp.LastRenderFrame)
	}

	// 测试添加缓存数据
	img := ebiten.NewImage(50, 50)
	x := 1.0
	y := 2.0
	data := RenderPartData{
		Img:     img,
		Frame:   reanim.Frame{X: &x, Y: &y},
		OffsetX: 3.0,
		OffsetY: 4.0,
	}

	comp.CachedRenderData = append(comp.CachedRenderData, data)

	if len(comp.CachedRenderData) != 1 {
		t.Errorf("Expected CachedRenderData length to be 1 after append, got %d", len(comp.CachedRenderData))
	}

	// 测试清空缓存（重用切片）
	comp.CachedRenderData = comp.CachedRenderData[:0]

	if len(comp.CachedRenderData) != 0 {
		t.Errorf("Expected CachedRenderData length to be 0 after reset, got %d", len(comp.CachedRenderData))
	}

	if cap(comp.CachedRenderData) != 10 {
		t.Errorf("Expected CachedRenderData capacity to remain 10 after reset, got %d", cap(comp.CachedRenderData))
	}
}

// TestCacheSliceReuse 测试缓存切片重用，避免内存泄漏（Story 13.4 Task 1）
func TestCacheSliceReuse(t *testing.T) {
	comp := &ReanimComponent{
		CachedRenderData: make([]RenderPartData, 0, 10),
	}

	// 第一次填充
	for i := 0; i < 5; i++ {
		x := float64(i)
		comp.CachedRenderData = append(comp.CachedRenderData, RenderPartData{
			Img:     ebiten.NewImage(10, 10),
			Frame:   reanim.Frame{X: &x},
			OffsetX: float64(i),
			OffsetY: float64(i),
		})
	}

	if len(comp.CachedRenderData) != 5 {
		t.Errorf("Expected length 5, got %d", len(comp.CachedRenderData))
	}

	// 清空缓存（重用切片）
	comp.CachedRenderData = comp.CachedRenderData[:0]

	// 验证容量不变
	if cap(comp.CachedRenderData) != 10 {
		t.Errorf("Expected capacity to remain 10 after reset, got %d", cap(comp.CachedRenderData))
	}

	// 第二次填充
	for i := 0; i < 3; i++ {
		x := float64(i * 10)
		comp.CachedRenderData = append(comp.CachedRenderData, RenderPartData{
			Img:     ebiten.NewImage(20, 20),
			Frame:   reanim.Frame{X: &x},
			OffsetX: float64(i * 10),
			OffsetY: float64(i * 10),
		})
	}

	if len(comp.CachedRenderData) != 3 {
		t.Errorf("Expected length 3 after second fill, got %d", len(comp.CachedRenderData))
	}

	// 验证容量仍然不变（切片重用成功）
	if cap(comp.CachedRenderData) != 10 {
		t.Errorf("Expected capacity to remain 10 after second fill, got %d", cap(comp.CachedRenderData))
	}
}
