package components

import (
	"image"
	"testing"

	"github.com/gonewx/pvz/pkg/ecs"
)

// TestShovelInteractionComponent_DefaultValues 测试组件默认值
func TestShovelInteractionComponent_DefaultValues(t *testing.T) {
	comp := &ShovelInteractionComponent{}

	// 验证默认值
	if comp.IsSelected {
		t.Error("Expected IsSelected to be false by default")
	}

	if comp.CursorImage != nil {
		t.Error("Expected CursorImage to be nil by default")
	}

	if comp.HighlightedPlantEntity != 0 {
		t.Errorf("Expected HighlightedPlantEntity to be 0, got %d", comp.HighlightedPlantEntity)
	}

	if comp.CursorAnchorX != 0 {
		t.Errorf("Expected CursorAnchorX to be 0, got %f", comp.CursorAnchorX)
	}

	if comp.CursorAnchorY != 0 {
		t.Errorf("Expected CursorAnchorY to be 0, got %f", comp.CursorAnchorY)
	}
}

// TestShovelInteractionComponent_SetIsSelected 测试设置选中状态
func TestShovelInteractionComponent_SetIsSelected(t *testing.T) {
	comp := &ShovelInteractionComponent{}

	// 设置选中状态
	comp.IsSelected = true
	if !comp.IsSelected {
		t.Error("Expected IsSelected to be true after setting")
	}

	// 取消选中
	comp.IsSelected = false
	if comp.IsSelected {
		t.Error("Expected IsSelected to be false after unsetting")
	}
}

// TestShovelInteractionComponent_SetHighlightedPlant 测试设置高亮植物
func TestShovelInteractionComponent_SetHighlightedPlant(t *testing.T) {
	comp := &ShovelInteractionComponent{}

	// 设置高亮植物
	plantID := ecs.EntityID(123)
	comp.HighlightedPlantEntity = plantID

	if comp.HighlightedPlantEntity != plantID {
		t.Errorf("Expected HighlightedPlantEntity to be %d, got %d", plantID, comp.HighlightedPlantEntity)
	}

	// 清除高亮
	comp.HighlightedPlantEntity = 0
	if comp.HighlightedPlantEntity != 0 {
		t.Errorf("Expected HighlightedPlantEntity to be 0, got %d", comp.HighlightedPlantEntity)
	}
}

// TestShovelInteractionComponent_SetShovelSlotBounds 测试设置铲子槽位边界
func TestShovelInteractionComponent_SetShovelSlotBounds(t *testing.T) {
	comp := &ShovelInteractionComponent{}

	// 设置边界
	bounds := image.Rect(100, 0, 170, 74)
	comp.ShovelSlotBounds = bounds

	if comp.ShovelSlotBounds != bounds {
		t.Errorf("Expected ShovelSlotBounds to be %v, got %v", bounds, comp.ShovelSlotBounds)
	}

	// 测试边界包含点
	testCases := []struct {
		x, y     int
		expected bool
	}{
		{135, 37, true},  // 中心点
		{100, 0, true},   // 左上角
		{169, 73, true},  // 右下角内
		{99, 0, false},   // 左外
		{170, 0, false},  // 右外
		{135, -1, false}, // 上外
		{135, 74, false}, // 下外
	}

	for _, tc := range testCases {
		point := image.Pt(tc.x, tc.y)
		result := point.In(comp.ShovelSlotBounds)
		if result != tc.expected {
			t.Errorf("Point (%d, %d) In bounds: expected %v, got %v", tc.x, tc.y, tc.expected, result)
		}
	}
}

// TestShovelInteractionComponent_CursorAnchor 测试光标锚点偏移
func TestShovelInteractionComponent_CursorAnchor(t *testing.T) {
	comp := &ShovelInteractionComponent{
		CursorAnchorX: 20.0,
		CursorAnchorY: 5.0,
	}

	if comp.CursorAnchorX != 20.0 {
		t.Errorf("Expected CursorAnchorX to be 20.0, got %f", comp.CursorAnchorX)
	}

	if comp.CursorAnchorY != 5.0 {
		t.Errorf("Expected CursorAnchorY to be 5.0, got %f", comp.CursorAnchorY)
	}
}
