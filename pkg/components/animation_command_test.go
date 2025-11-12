package components

import (
	"testing"
)

// TestAnimationCommandComponent_ComboMode 测试配置组合模式
func TestAnimationCommandComponent_ComboMode(t *testing.T) {
	cmd := &AnimationCommandComponent{
		UnitID:    "zombie",
		ComboName: "death",
		Processed: false,
		Timestamp: 1.5,
	}

	// 验证字段
	if cmd.UnitID != "zombie" {
		t.Errorf("Expected UnitID 'zombie', got '%s'", cmd.UnitID)
	}
	if cmd.ComboName != "death" {
		t.Errorf("Expected ComboName 'death', got '%s'", cmd.ComboName)
	}
	if cmd.Processed {
		t.Error("Expected Processed to be false")
	}
	if cmd.AnimationName != "" {
		t.Error("Expected AnimationName to be empty in combo mode")
	}
}

// TestAnimationCommandComponent_SingleMode 测试单动画模式
func TestAnimationCommandComponent_SingleMode(t *testing.T) {
	cmd := &AnimationCommandComponent{
		AnimationName: "FinalWave",
		Processed:     false,
	}

	// 验证字段
	if cmd.AnimationName != "FinalWave" {
		t.Errorf("Expected AnimationName 'FinalWave', got '%s'", cmd.AnimationName)
	}
	if cmd.UnitID != "" {
		t.Error("Expected UnitID to be empty in single mode")
	}
	if cmd.ComboName != "" {
		t.Error("Expected ComboName to be empty in single mode")
	}
}

// TestAnimationCommandComponent_DefaultCombo 测试默认组合(ComboName 为空)
func TestAnimationCommandComponent_DefaultCombo(t *testing.T) {
	cmd := &AnimationCommandComponent{
		UnitID:    "peashooter",
		ComboName: "", // 使用默认 combo
	}

	// 验证字段
	if cmd.UnitID != "peashooter" {
		t.Errorf("Expected UnitID 'peashooter', got '%s'", cmd.UnitID)
	}
	if cmd.ComboName != "" {
		t.Errorf("Expected empty ComboName for default combo, got '%s'", cmd.ComboName)
	}
}

// TestAnimationCommandComponent_ZeroValue 测试零值
func TestAnimationCommandComponent_ZeroValue(t *testing.T) {
	var cmd AnimationCommandComponent

	// 验证零值
	if cmd.UnitID != "" {
		t.Error("Expected zero UnitID")
	}
	if cmd.ComboName != "" {
		t.Error("Expected zero ComboName")
	}
	if cmd.AnimationName != "" {
		t.Error("Expected zero AnimationName")
	}
	if cmd.Processed {
		t.Error("Expected Processed to be false (zero value)")
	}
	if cmd.Timestamp != 0.0 {
		t.Error("Expected Timestamp to be 0.0 (zero value)")
	}
}
