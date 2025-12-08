package components

import (
	"testing"
)

// TestVirtualKeyboardComponent_InitialState 测试虚拟键盘组件初始状态
func TestVirtualKeyboardComponent_InitialState(t *testing.T) {
	kb := &VirtualKeyboardComponent{
		IsVisible:   false,
		ShiftActive: false,
		NumericMode: false,
		KeyStates:   make(map[string]bool),
	}

	if kb.IsVisible {
		t.Error("IsVisible should be false initially")
	}
	if kb.ShiftActive {
		t.Error("ShiftActive should be false initially")
	}
	if kb.NumericMode {
		t.Error("NumericMode should be false initially")
	}
}

// TestVirtualKeyboardComponent_ShiftToggle 测试 Shift 切换
func TestVirtualKeyboardComponent_ShiftToggle(t *testing.T) {
	kb := &VirtualKeyboardComponent{
		ShiftActive: false,
	}

	// 切换到大写
	kb.ShiftActive = !kb.ShiftActive
	if !kb.ShiftActive {
		t.Error("ShiftActive should be true after toggle")
	}

	// 再次切换回小写
	kb.ShiftActive = !kb.ShiftActive
	if kb.ShiftActive {
		t.Error("ShiftActive should be false after second toggle")
	}
}

// TestVirtualKeyboardComponent_NumericModeToggle 测试数字模式切换
func TestVirtualKeyboardComponent_NumericModeToggle(t *testing.T) {
	kb := &VirtualKeyboardComponent{
		NumericMode: false,
	}

	// 切换到数字模式
	kb.NumericMode = true
	if !kb.NumericMode {
		t.Error("NumericMode should be true after setting")
	}

	// 切换回字母模式
	kb.NumericMode = false
	if kb.NumericMode {
		t.Error("NumericMode should be false after clearing")
	}
}

// TestKeyboardLayoutLower 测试小写布局完整性
func TestKeyboardLayoutLower(t *testing.T) {
	// 验证有4行
	if len(KeyboardLayoutLower) != 4 {
		t.Errorf("KeyboardLayoutLower should have 4 rows, got %d", len(KeyboardLayoutLower))
	}

	// 验证第一行有10个键（q-p）
	if len(KeyboardLayoutLower[0]) != 10 {
		t.Errorf("First row should have 10 keys, got %d", len(KeyboardLayoutLower[0]))
	}

	// 验证第一行是 qwertyuiop
	expectedFirstRow := []string{"q", "w", "e", "r", "t", "y", "u", "i", "o", "p"}
	for i, key := range KeyboardLayoutLower[0] {
		if key != expectedFirstRow[i] {
			t.Errorf("First row key %d: expected %s, got %s", i, expectedFirstRow[i], key)
		}
	}

	// 验证第二行有9个键（a-l）
	if len(KeyboardLayoutLower[1]) != 9 {
		t.Errorf("Second row should have 9 keys, got %d", len(KeyboardLayoutLower[1]))
	}

	// 验证第三行有9个键（包含 SHIFT 和 BACKSPACE）
	if len(KeyboardLayoutLower[2]) != 9 {
		t.Errorf("Third row should have 9 keys, got %d", len(KeyboardLayoutLower[2]))
	}

	// 验证第三行第一个是 SHIFT
	if KeyboardLayoutLower[2][0] != "SHIFT" {
		t.Errorf("Third row first key should be SHIFT, got %s", KeyboardLayoutLower[2][0])
	}

	// 验证第三行最后一个是 BACKSPACE
	if KeyboardLayoutLower[2][8] != "BACKSPACE" {
		t.Errorf("Third row last key should be BACKSPACE, got %s", KeyboardLayoutLower[2][8])
	}

	// 验证第四行有3个键（123, SPACE, DONE）
	if len(KeyboardLayoutLower[3]) != 3 {
		t.Errorf("Fourth row should have 3 keys, got %d", len(KeyboardLayoutLower[3]))
	}

	// 验证第四行键名
	if KeyboardLayoutLower[3][0] != "123" {
		t.Errorf("Fourth row first key should be 123, got %s", KeyboardLayoutLower[3][0])
	}
	if KeyboardLayoutLower[3][1] != "SPACE" {
		t.Errorf("Fourth row second key should be SPACE, got %s", KeyboardLayoutLower[3][1])
	}
	if KeyboardLayoutLower[3][2] != "DONE" {
		t.Errorf("Fourth row third key should be DONE, got %s", KeyboardLayoutLower[3][2])
	}
}

// TestKeyboardLayoutUpper 测试大写布局完整性
func TestKeyboardLayoutUpper(t *testing.T) {
	// 验证有4行
	if len(KeyboardLayoutUpper) != 4 {
		t.Errorf("KeyboardLayoutUpper should have 4 rows, got %d", len(KeyboardLayoutUpper))
	}

	// 验证第一行是大写 QWERTYUIOP
	expectedFirstRow := []string{"Q", "W", "E", "R", "T", "Y", "U", "I", "O", "P"}
	for i, key := range KeyboardLayoutUpper[0] {
		if key != expectedFirstRow[i] {
			t.Errorf("First row key %d: expected %s, got %s", i, expectedFirstRow[i], key)
		}
	}

	// 验证第二行是大写 ASDFGHJKL
	expectedSecondRow := []string{"A", "S", "D", "F", "G", "H", "J", "K", "L"}
	for i, key := range KeyboardLayoutUpper[1] {
		if key != expectedSecondRow[i] {
			t.Errorf("Second row key %d: expected %s, got %s", i, expectedSecondRow[i], key)
		}
	}
}

// TestKeyboardLayoutNumeric 测试数字布局完整性
func TestKeyboardLayoutNumeric(t *testing.T) {
	// 验证有2行
	if len(KeyboardLayoutNumeric) != 2 {
		t.Errorf("KeyboardLayoutNumeric should have 2 rows, got %d", len(KeyboardLayoutNumeric))
	}

	// 验证第一行是 1234567890
	expectedFirstRow := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"}
	if len(KeyboardLayoutNumeric[0]) != 10 {
		t.Errorf("First row should have 10 keys, got %d", len(KeyboardLayoutNumeric[0]))
	}
	for i, key := range KeyboardLayoutNumeric[0] {
		if key != expectedFirstRow[i] {
			t.Errorf("First row key %d: expected %s, got %s", i, expectedFirstRow[i], key)
		}
	}

	// 验证第二行有3个键（ABC, SPACE, DONE）
	if len(KeyboardLayoutNumeric[1]) != 3 {
		t.Errorf("Second row should have 3 keys, got %d", len(KeyboardLayoutNumeric[1]))
	}

	// 验证第二行键名
	if KeyboardLayoutNumeric[1][0] != "ABC" {
		t.Errorf("Second row first key should be ABC, got %s", KeyboardLayoutNumeric[1][0])
	}
}

// TestGetKeyWidthFactor 测试按键宽度倍数
func TestGetKeyWidthFactor(t *testing.T) {
	tests := []struct {
		action   string
		expected float64
	}{
		{"a", KeyWidthNormal},
		{"Z", KeyWidthNormal},
		{"5", KeyWidthNormal},
		{"SHIFT", KeyWidthShift},
		{"BACKSPACE", KeyWidthBackspace},
		{"123", KeyWidth123},
		{"ABC", KeyWidth123},
		{"SPACE", KeyWidthSpace},
		{"DONE", KeyWidthDone},
	}

	for _, tt := range tests {
		got := GetKeyWidthFactor(tt.action)
		if got != tt.expected {
			t.Errorf("GetKeyWidthFactor(%s): expected %.1f, got %.1f", tt.action, tt.expected, got)
		}
	}
}

// TestGetKeyLabel 测试按键标签
func TestGetKeyLabel(t *testing.T) {
	tests := []struct {
		action   string
		expected string
	}{
		{"a", "a"},
		{"Z", "Z"},
		{"5", "5"},
		{"SHIFT", LabelShift},
		{"BACKSPACE", LabelBackspace},
		{"SPACE", LabelSpace},
		{"DONE", LabelDone},
		{"123", Label123},
		{"ABC", LabelABC},
	}

	for _, tt := range tests {
		got := GetKeyLabel(tt.action)
		if got != tt.expected {
			t.Errorf("GetKeyLabel(%s): expected %s, got %s", tt.action, tt.expected, got)
		}
	}
}

// TestIsSpecialKey 测试特殊按键判断
func TestIsSpecialKey(t *testing.T) {
	specialKeys := []string{"SHIFT", "BACKSPACE", "SPACE", "DONE", "123", "ABC"}
	normalKeys := []string{"a", "Z", "5", "q", "P"}

	for _, key := range specialKeys {
		if !IsSpecialKey(key) {
			t.Errorf("IsSpecialKey(%s): expected true, got false", key)
		}
	}

	for _, key := range normalKeys {
		if IsSpecialKey(key) {
			t.Errorf("IsSpecialKey(%s): expected false, got true", key)
		}
	}
}
