package components

import (
	"testing"
)

// TestDaveDialogueComponent_Init 测试组件初始化
func TestDaveDialogueComponent_Init(t *testing.T) {
	comp := &DaveDialogueComponent{
		DialogueKeys:     []string{"CRAZY_DAVE_2400", "CRAZY_DAVE_2401"},
		CurrentLineIndex: 0,
		State:            DaveStateHidden,
	}

	if len(comp.DialogueKeys) != 2 {
		t.Errorf("Expected 2 dialogue keys, got %d", len(comp.DialogueKeys))
	}

	if comp.State != DaveStateHidden {
		t.Errorf("Expected state Hidden, got %v", comp.State)
	}

	if comp.CurrentLineIndex != 0 {
		t.Errorf("Expected CurrentLineIndex 0, got %d", comp.CurrentLineIndex)
	}
}

// TestDaveDialogueComponent_DefaultValues 测试组件默认值
func TestDaveDialogueComponent_DefaultValues(t *testing.T) {
	comp := &DaveDialogueComponent{}

	if comp.IsVisible != false {
		t.Error("Expected IsVisible to be false by default")
	}

	if comp.State != DaveStateHidden {
		t.Errorf("Expected default state to be Hidden (0), got %v", comp.State)
	}

	if comp.CurrentText != "" {
		t.Error("Expected CurrentText to be empty by default")
	}

	if comp.Expression != "" {
		t.Error("Expected Expression to be empty by default")
	}
}

// TestDaveState_String 测试状态枚举的字符串表示
func TestDaveState_String(t *testing.T) {
	tests := []struct {
		state    DaveState
		expected string
	}{
		{DaveStateHidden, "Hidden"},
		{DaveStateEntering, "Entering"},
		{DaveStateTalking, "Talking"},
		{DaveStateLeaving, "Leaving"},
		{DaveState(99), "Unknown"}, // 测试未知状态
	}

	for _, test := range tests {
		result := test.state.String()
		if result != test.expected {
			t.Errorf("DaveState(%d).String() = %s, expected %s", test.state, result, test.expected)
		}
	}
}

// TestDaveDialogueComponent_WithCallback 测试带回调的组件
func TestDaveDialogueComponent_WithCallback(t *testing.T) {
	callbackCalled := false
	callback := func() {
		callbackCalled = true
	}

	comp := &DaveDialogueComponent{
		DialogueKeys:       []string{"TEST_KEY"},
		OnCompleteCallback: callback,
	}

	// 验证回调已设置
	if comp.OnCompleteCallback == nil {
		t.Error("Expected OnCompleteCallback to be set")
	}

	// 调用回调
	comp.OnCompleteCallback()

	if !callbackCalled {
		t.Error("Expected callback to be called")
	}
}

// TestDaveDialogueComponent_BubbleOffset 测试气泡偏移设置
func TestDaveDialogueComponent_BubbleOffset(t *testing.T) {
	comp := &DaveDialogueComponent{
		BubbleOffsetX: 180.0,
		BubbleOffsetY: -300.0,
	}

	if comp.BubbleOffsetX != 180.0 {
		t.Errorf("Expected BubbleOffsetX 180.0, got %f", comp.BubbleOffsetX)
	}

	if comp.BubbleOffsetY != -300.0 {
		t.Errorf("Expected BubbleOffsetY -300.0, got %f", comp.BubbleOffsetY)
	}
}

// TestDaveDialogueComponent_Expressions 测试表情指令列表
func TestDaveDialogueComponent_Expressions(t *testing.T) {
	comp := &DaveDialogueComponent{
		CurrentExpressions: []string{"MOUTH_SMALL_OH", "SCREAM"},
	}

	if len(comp.CurrentExpressions) != 2 {
		t.Errorf("Expected 2 expressions, got %d", len(comp.CurrentExpressions))
	}

	if comp.CurrentExpressions[0] != "MOUTH_SMALL_OH" {
		t.Errorf("Expected first expression MOUTH_SMALL_OH, got %s", comp.CurrentExpressions[0])
	}

	if comp.CurrentExpressions[1] != "SCREAM" {
		t.Errorf("Expected second expression SCREAM, got %s", comp.CurrentExpressions[1])
	}
}
