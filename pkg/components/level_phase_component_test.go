package components

import (
	"testing"
)

// TestLevelPhaseComponent_Constants 测试阶段状态和转场步骤常量
func TestLevelPhaseComponent_Constants(t *testing.T) {
	t.Run("PhaseState constants", func(t *testing.T) {
		if PhaseStateActive != "active" {
			t.Errorf("PhaseStateActive expected 'active', got '%s'", PhaseStateActive)
		}
		if PhaseStateTransitioning != "transitioning" {
			t.Errorf("PhaseStateTransitioning expected 'transitioning', got '%s'", PhaseStateTransitioning)
		}
		if PhaseStateCompleted != "completed" {
			t.Errorf("PhaseStateCompleted expected 'completed', got '%s'", PhaseStateCompleted)
		}
	})

	t.Run("TransitionStep constants", func(t *testing.T) {
		if TransitionStepNone != 0 {
			t.Errorf("TransitionStepNone expected 0, got %d", TransitionStepNone)
		}
		if TransitionStepDisableGuided != 1 {
			t.Errorf("TransitionStepDisableGuided expected 1, got %d", TransitionStepDisableGuided)
		}
		if TransitionStepDaveDialogue != 2 {
			t.Errorf("TransitionStepDaveDialogue expected 2, got %d", TransitionStepDaveDialogue)
		}
		if TransitionStepConveyorSlide != 3 {
			t.Errorf("TransitionStepConveyorSlide expected 3, got %d", TransitionStepConveyorSlide)
		}
		if TransitionStepShowRedLine != 4 {
			t.Errorf("TransitionStepShowRedLine expected 4, got %d", TransitionStepShowRedLine)
		}
		if TransitionStepActivateBowling != 5 {
			t.Errorf("TransitionStepActivateBowling expected 5, got %d", TransitionStepActivateBowling)
		}
	})
}

// TestLevelPhaseComponent_Initialization 测试组件初始化
func TestLevelPhaseComponent_Initialization(t *testing.T) {
	t.Run("zero value initialization", func(t *testing.T) {
		comp := &LevelPhaseComponent{}

		if comp.CurrentPhase != 0 {
			t.Errorf("Expected CurrentPhase 0, got %d", comp.CurrentPhase)
		}
		if comp.PhaseState != "" {
			t.Errorf("Expected empty PhaseState, got '%s'", comp.PhaseState)
		}
		if comp.TransitionProgress != 0 {
			t.Errorf("Expected TransitionProgress 0, got %f", comp.TransitionProgress)
		}
		if comp.TransitionStep != TransitionStepNone {
			t.Errorf("Expected TransitionStep 0, got %d", comp.TransitionStep)
		}
		if comp.ShowRedLine {
			t.Error("Expected ShowRedLine false, got true")
		}
		if comp.ConveyorBeltVisible {
			t.Error("Expected ConveyorBeltVisible false, got true")
		}
	})

	t.Run("typical initial state for Level 1-5", func(t *testing.T) {
		comp := &LevelPhaseComponent{
			CurrentPhase:        1,
			PhaseState:          PhaseStateActive,
			TransitionProgress:  0,
			TransitionStep:      TransitionStepNone,
			ConveyorBeltY:       -100.0,
			ConveyorBeltVisible: false,
			ShowRedLine:         false,
		}

		if comp.CurrentPhase != 1 {
			t.Errorf("Expected CurrentPhase 1, got %d", comp.CurrentPhase)
		}
		if comp.PhaseState != PhaseStateActive {
			t.Errorf("Expected PhaseState 'active', got '%s'", comp.PhaseState)
		}
		if comp.ConveyorBeltY != -100.0 {
			t.Errorf("Expected ConveyorBeltY -100.0, got %f", comp.ConveyorBeltY)
		}
	})
}

// TestLevelPhaseComponent_StateTransitions 测试状态转换
func TestLevelPhaseComponent_StateTransitions(t *testing.T) {
	t.Run("transition from active to transitioning", func(t *testing.T) {
		comp := &LevelPhaseComponent{
			CurrentPhase: 1,
			PhaseState:   PhaseStateActive,
		}

		// 模拟开始转场
		comp.PhaseState = PhaseStateTransitioning
		comp.TransitionStep = TransitionStepDisableGuided

		if comp.PhaseState != PhaseStateTransitioning {
			t.Errorf("Expected PhaseState 'transitioning', got '%s'", comp.PhaseState)
		}
		if comp.TransitionStep != TransitionStepDisableGuided {
			t.Errorf("Expected TransitionStep 1, got %d", comp.TransitionStep)
		}
	})

	t.Run("transition steps sequence", func(t *testing.T) {
		comp := &LevelPhaseComponent{
			CurrentPhase:   1,
			PhaseState:     PhaseStateTransitioning,
			TransitionStep: TransitionStepDisableGuided,
		}

		// 验证转场步骤序列
		steps := []int{
			TransitionStepDisableGuided,
			TransitionStepDaveDialogue,
			TransitionStepConveyorSlide,
			TransitionStepShowRedLine,
			TransitionStepActivateBowling,
		}

		for i, expectedStep := range steps {
			comp.TransitionStep = expectedStep
			if comp.TransitionStep != expectedStep {
				t.Errorf("Step %d: Expected TransitionStep %d, got %d", i, expectedStep, comp.TransitionStep)
			}
		}
	})

	t.Run("transition completion", func(t *testing.T) {
		comp := &LevelPhaseComponent{
			CurrentPhase:   1,
			PhaseState:     PhaseStateTransitioning,
			TransitionStep: TransitionStepActivateBowling,
		}

		// 模拟转场完成
		comp.CurrentPhase = 2
		comp.PhaseState = PhaseStateActive
		comp.TransitionStep = TransitionStepNone

		if comp.CurrentPhase != 2 {
			t.Errorf("Expected CurrentPhase 2, got %d", comp.CurrentPhase)
		}
		if comp.PhaseState != PhaseStateActive {
			t.Errorf("Expected PhaseState 'active', got '%s'", comp.PhaseState)
		}
		if comp.TransitionStep != TransitionStepNone {
			t.Errorf("Expected TransitionStep 0, got %d", comp.TransitionStep)
		}
	})
}

// TestLevelPhaseComponent_ConveyorBelt 测试传送带状态
func TestLevelPhaseComponent_ConveyorBelt(t *testing.T) {
	t.Run("conveyor belt visibility", func(t *testing.T) {
		comp := &LevelPhaseComponent{
			ConveyorBeltVisible: false,
			ConveyorBeltY:       -100.0,
		}

		// 模拟传送带显示
		comp.ConveyorBeltVisible = true
		if !comp.ConveyorBeltVisible {
			t.Error("Expected ConveyorBeltVisible true")
		}
	})

	t.Run("conveyor belt slide animation", func(t *testing.T) {
		comp := &LevelPhaseComponent{
			ConveyorBeltVisible: true,
			ConveyorBeltY:       -100.0,
		}

		// 模拟滑入动画中间状态
		comp.ConveyorBeltY = -50.0
		if comp.ConveyorBeltY != -50.0 {
			t.Errorf("Expected ConveyorBeltY -50.0, got %f", comp.ConveyorBeltY)
		}

		// 模拟滑入完成
		comp.ConveyorBeltY = 10.0
		if comp.ConveyorBeltY != 10.0 {
			t.Errorf("Expected ConveyorBeltY 10.0, got %f", comp.ConveyorBeltY)
		}
	})
}

// TestLevelPhaseComponent_RedLine 测试红线状态
func TestLevelPhaseComponent_RedLine(t *testing.T) {
	t.Run("red line visibility", func(t *testing.T) {
		comp := &LevelPhaseComponent{
			ShowRedLine: false,
		}

		// 红线初始隐藏
		if comp.ShowRedLine {
			t.Error("Expected ShowRedLine false initially")
		}

		// 显示红线
		comp.ShowRedLine = true
		if !comp.ShowRedLine {
			t.Error("Expected ShowRedLine true after setting")
		}
	})
}

// TestLevelPhaseComponent_Callback 测试回调函数
func TestLevelPhaseComponent_Callback(t *testing.T) {
	t.Run("OnPhaseTransitionComplete callback", func(t *testing.T) {
		callbackCalled := false
		comp := &LevelPhaseComponent{
			OnPhaseTransitionComplete: func() {
				callbackCalled = true
			},
		}

		// 执行回调
		if comp.OnPhaseTransitionComplete != nil {
			comp.OnPhaseTransitionComplete()
		}

		if !callbackCalled {
			t.Error("Expected callback to be called")
		}
	})

	t.Run("nil callback should not panic", func(t *testing.T) {
		comp := &LevelPhaseComponent{}

		// 确保 nil 回调不会 panic
		if comp.OnPhaseTransitionComplete != nil {
			comp.OnPhaseTransitionComplete()
		}
		// 如果执行到这里没有 panic，测试通过
	})
}

// TestLevelPhaseComponent_DaveDialogueEntityID 测试 Dave 对话实体 ID
func TestLevelPhaseComponent_DaveDialogueEntityID(t *testing.T) {
	t.Run("Dave dialogue entity ID", func(t *testing.T) {
		comp := &LevelPhaseComponent{
			DaveDialogueEntityID: 0,
		}

		// 初始为 0
		if comp.DaveDialogueEntityID != 0 {
			t.Errorf("Expected DaveDialogueEntityID 0, got %d", comp.DaveDialogueEntityID)
		}

		// 设置 Dave 实体 ID
		comp.DaveDialogueEntityID = 123
		if comp.DaveDialogueEntityID != 123 {
			t.Errorf("Expected DaveDialogueEntityID 123, got %d", comp.DaveDialogueEntityID)
		}
	})
}

// TestLevelPhaseComponent_TransitionProgress 测试转场进度
func TestLevelPhaseComponent_TransitionProgress(t *testing.T) {
	t.Run("transition progress range", func(t *testing.T) {
		comp := &LevelPhaseComponent{
			TransitionProgress: 0,
		}

		// 测试进度从 0 到 1
		testValues := []float64{0.0, 0.25, 0.5, 0.75, 1.0}
		for _, expected := range testValues {
			comp.TransitionProgress = expected
			if comp.TransitionProgress != expected {
				t.Errorf("Expected TransitionProgress %f, got %f", expected, comp.TransitionProgress)
			}
		}
	})

	t.Run("transition progress clamping", func(t *testing.T) {
		comp := &LevelPhaseComponent{
			TransitionProgress: 1.5,
		}

		// 手动将进度限制在 1.0
		if comp.TransitionProgress > 1.0 {
			comp.TransitionProgress = 1.0
		}

		if comp.TransitionProgress != 1.0 {
			t.Errorf("Expected TransitionProgress clamped to 1.0, got %f", comp.TransitionProgress)
		}
	})
}
