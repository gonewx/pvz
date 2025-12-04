package systems

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestNewLevelPhaseSystem 测试系统创建
func TestNewLevelPhaseSystem(t *testing.T) {
	t.Run("creates system with valid initial state", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		var rm *game.ResourceManager = nil // 允许 nil，因为测试不需要实际资源

		system := NewLevelPhaseSystem(em, gs, rm)

		if system == nil {
			t.Fatal("Expected non-nil system")
		}

		// 验证初始阶段为 1
		if system.GetCurrentPhase() != 1 {
			t.Errorf("Expected initial phase 1, got %d", system.GetCurrentPhase())
		}

		// 验证不在转场中
		if system.IsTransitioning() {
			t.Error("Expected not transitioning initially")
		}

		// 验证红线不显示
		if system.ShouldShowRedLine() {
			t.Error("Expected red line not shown initially")
		}

		// 验证传送带不可见
		if system.IsConveyorBeltVisible() {
			t.Error("Expected conveyor belt not visible initially")
		}
	})

	t.Run("creates phase entity with component", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()

		system := NewLevelPhaseSystem(em, gs, nil)

		phaseEntity := system.GetPhaseEntity()
		if phaseEntity == 0 {
			t.Error("Expected non-zero phase entity ID")
		}

		// 验证实体有 LevelPhaseComponent
		phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](em, phaseEntity)
		if !ok {
			t.Fatal("Expected LevelPhaseComponent on phase entity")
		}

		if phaseComp.CurrentPhase != 1 {
			t.Errorf("Expected CurrentPhase 1, got %d", phaseComp.CurrentPhase)
		}
		if phaseComp.PhaseState != components.PhaseStateActive {
			t.Errorf("Expected PhaseState 'active', got '%s'", phaseComp.PhaseState)
		}
	})
}

// TestLevelPhaseSystem_IsInPhase 测试阶段检查
func TestLevelPhaseSystem_IsInPhase(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	system := NewLevelPhaseSystem(em, gs, nil)

	t.Run("initially in phase 1", func(t *testing.T) {
		if !system.IsInPhase(1) {
			t.Error("Expected to be in phase 1")
		}
		if system.IsInPhase(2) {
			t.Error("Expected not to be in phase 2")
		}
	})
}

// TestLevelPhaseSystem_StartPhaseTransition 测试启动阶段转场
func TestLevelPhaseSystem_StartPhaseTransition(t *testing.T) {
	t.Run("starts transition from correct phase", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		system := NewLevelPhaseSystem(em, gs, nil)

		// 启动从阶段 1 到阶段 2 的转场
		system.StartPhaseTransition(1, 2)

		// 验证进入转场状态
		if !system.IsTransitioning() {
			t.Error("Expected to be transitioning after StartPhaseTransition")
		}

		// 验证仍在阶段 1（转场未完成）
		if !system.IsInPhase(1) {
			t.Error("Expected to still be in phase 1 during transition")
		}
	})

	t.Run("ignores transition from wrong phase", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		system := NewLevelPhaseSystem(em, gs, nil)

		// 尝试从阶段 2 开始转场（但当前在阶段 1）
		system.StartPhaseTransition(2, 3)

		// 应该被忽略，不进入转场状态
		if system.IsTransitioning() {
			t.Error("Expected not to be transitioning when starting from wrong phase")
		}
	})
}

// TestLevelPhaseSystem_Callbacks 测试回调设置
func TestLevelPhaseSystem_Callbacks(t *testing.T) {
	t.Run("SetOnDisableGuidedTutorial", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		system := NewLevelPhaseSystem(em, gs, nil)

		called := false
		system.SetOnDisableGuidedTutorial(func() {
			called = true
		})

		// 启动转场触发回调
		system.StartPhaseTransition(1, 2)
		system.Update(0.016) // 一帧更新

		if !called {
			t.Error("Expected OnDisableGuidedTutorial callback to be called")
		}
	})

	t.Run("SetOnActivateBowling", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		system := NewLevelPhaseSystem(em, gs, nil)

		system.SetOnActivateBowling(func() {})

		// 验证回调被设置
		if system.onActivateBowling == nil {
			t.Error("Expected onActivateBowling callback to be set")
		}
	})

	t.Run("SetOnTransitionComplete", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		system := NewLevelPhaseSystem(em, gs, nil)

		system.SetOnTransitionComplete(func() {})

		// 验证回调被设置
		if system.onTransitionComplete == nil {
			t.Error("Expected onTransitionComplete callback to be set")
		}
	})
}

// TestLevelPhaseSystem_Update 测试更新逻辑
func TestLevelPhaseSystem_Update(t *testing.T) {
	t.Run("no update when not transitioning", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		system := NewLevelPhaseSystem(em, gs, nil)

		// 未启动转场时更新不应改变状态
		initialPhase := system.GetCurrentPhase()
		system.Update(0.016)

		if system.GetCurrentPhase() != initialPhase {
			t.Error("Expected phase not to change when not transitioning")
		}
	})

	t.Run("advances transition step after disable guided", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		system := NewLevelPhaseSystem(em, gs, nil)

		// 启动转场
		system.StartPhaseTransition(1, 2)

		// 获取阶段组件验证状态
		phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](em, system.GetPhaseEntity())
		if !ok {
			t.Fatal("Expected phase component")
		}

		// 验证初始转场步骤
		if phaseComp.TransitionStep != components.TransitionStepDisableGuided {
			t.Errorf("Expected TransitionStepDisableGuided, got %d", phaseComp.TransitionStep)
		}

		// 执行一次更新
		system.Update(0.016)

		// 因为 ResourceLoader 为 nil，Dave 对话被跳过，直接进入传送带滑入步骤
		// 这是预期行为，测试环境下不需要实际资源
		if phaseComp.TransitionStep != components.TransitionStepConveyorSlide {
			t.Errorf("Expected TransitionStepConveyorSlide after update (Dave skipped due to nil ResourceLoader), got %d", phaseComp.TransitionStep)
		}
	})
}

// TestLevelPhaseSystem_ConveyorSlideAnimation 测试传送带滑入动画
func TestLevelPhaseSystem_ConveyorSlideAnimation(t *testing.T) {
	t.Run("conveyor slide updates Y position", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		system := NewLevelPhaseSystem(em, gs, nil)

		// 直接设置到传送带滑入步骤
		phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](em, system.GetPhaseEntity())
		if !ok {
			t.Fatal("Expected phase component")
		}

		phaseComp.PhaseState = components.PhaseStateTransitioning
		phaseComp.TransitionStep = components.TransitionStepConveyorSlide
		phaseComp.ConveyorBeltVisible = true
		phaseComp.ConveyorBeltY = -100.0
		phaseComp.TransitionProgress = 0

		initialY := system.GetConveyorBeltY()

		// 更新一帧
		system.Update(0.1) // 0.1 秒

		// Y 位置应该改变
		newY := system.GetConveyorBeltY()
		if newY == initialY {
			t.Error("Expected conveyor belt Y to change after update")
		}
	})

	t.Run("conveyor visible after slide starts", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		system := NewLevelPhaseSystem(em, gs, nil)

		// 设置到传送带滑入步骤
		phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](em, system.GetPhaseEntity())
		if !ok {
			t.Fatal("Expected phase component")
		}

		phaseComp.PhaseState = components.PhaseStateTransitioning
		phaseComp.TransitionStep = components.TransitionStepConveyorSlide
		phaseComp.ConveyorBeltVisible = true

		if !system.IsConveyorBeltVisible() {
			t.Error("Expected conveyor belt to be visible during slide")
		}
	})
}

// TestLevelPhaseSystem_RedLineVisibility 测试红线可见性
func TestLevelPhaseSystem_RedLineVisibility(t *testing.T) {
	t.Run("red line not visible initially", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		system := NewLevelPhaseSystem(em, gs, nil)

		if system.ShouldShowRedLine() {
			t.Error("Expected red line not visible initially")
		}
	})

	t.Run("red line visible after step 4", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		system := NewLevelPhaseSystem(em, gs, nil)

		// 设置到显示红线步骤
		phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](em, system.GetPhaseEntity())
		if !ok {
			t.Fatal("Expected phase component")
		}

		phaseComp.PhaseState = components.PhaseStateTransitioning
		phaseComp.TransitionStep = components.TransitionStepShowRedLine

		// 执行更新
		system.Update(0.016)

		// 红线应该可见
		if !system.ShouldShowRedLine() {
			t.Error("Expected red line to be visible after step 4")
		}
	})
}

// TestLevelPhaseSystem_PhaseCompletion 测试阶段完成
func TestLevelPhaseSystem_PhaseCompletion(t *testing.T) {
	t.Run("transition completes and enters phase 2", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		system := NewLevelPhaseSystem(em, gs, nil)

		// 设置到激活保龄球步骤
		phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](em, system.GetPhaseEntity())
		if !ok {
			t.Fatal("Expected phase component")
		}

		phaseComp.PhaseState = components.PhaseStateTransitioning
		phaseComp.TransitionStep = components.TransitionStepActivateBowling

		// 执行更新
		system.Update(0.016)

		// 应该进入阶段 2
		if !system.IsInPhase(2) {
			t.Error("Expected to be in phase 2 after completion")
		}

		// 不应该再处于转场状态
		if system.IsTransitioning() {
			t.Error("Expected not to be transitioning after completion")
		}
	})

	t.Run("OnActivateBowling callback called on completion", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		system := NewLevelPhaseSystem(em, gs, nil)

		called := false
		system.SetOnActivateBowling(func() {
			called = true
		})

		// 设置到激活保龄球步骤
		phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](em, system.GetPhaseEntity())
		if !ok {
			t.Fatal("Expected phase component")
		}

		phaseComp.PhaseState = components.PhaseStateTransitioning
		phaseComp.TransitionStep = components.TransitionStepActivateBowling

		// 执行更新
		system.Update(0.016)

		if !called {
			t.Error("Expected OnActivateBowling callback to be called")
		}
	})
}

// TestLevelPhaseSystem_GetPhaseEntity 测试获取阶段实体
func TestLevelPhaseSystem_GetPhaseEntity(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	system := NewLevelPhaseSystem(em, gs, nil)

	entity := system.GetPhaseEntity()
	if entity == 0 {
		t.Error("Expected non-zero entity ID")
	}

	// 验证实体有组件（这证明实体存在）
	_, ok := ecs.GetComponent[*components.LevelPhaseComponent](em, entity)
	if !ok {
		t.Error("Expected phase entity to have LevelPhaseComponent")
	}
}

// TestLevelPhaseSystem_GetConveyorBeltY 测试获取传送带 Y 位置
func TestLevelPhaseSystem_GetConveyorBeltY(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	system := NewLevelPhaseSystem(em, gs, nil)

	// 初始 Y 位置应该是起始位置
	y := system.GetConveyorBeltY()

	// 验证返回的是有效值（应该是配置的起始位置）
	if y > 0 {
		t.Errorf("Expected negative initial Y position, got %f", y)
	}
}

// TestLevelPhaseSystem_SetDaveDialogueEntityID 测试设置 Dave 对话实体 ID
// Bug Fix: 从存档恢复时需要更新 Dave 对话实体 ID
func TestLevelPhaseSystem_SetDaveDialogueEntityID(t *testing.T) {
	t.Run("updates DaveDialogueEntityID in component", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		system := NewLevelPhaseSystem(em, gs, nil)

		// 设置新的 Dave 对话实体 ID
		newEntityID := ecs.EntityID(42)
		system.SetDaveDialogueEntityID(newEntityID)

		// 验证组件中的 ID 已更新
		phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](em, system.GetPhaseEntity())
		if !ok {
			t.Fatal("Expected phase component")
		}

		if phaseComp.DaveDialogueEntityID != int(newEntityID) {
			t.Errorf("Expected DaveDialogueEntityID %d, got %d", newEntityID, phaseComp.DaveDialogueEntityID)
		}
	})
}

// TestLevelPhaseSystem_AdvanceToConveyorSlide 测试推进到传送带滑入步骤
// Bug Fix: 从存档恢复 Dave 对话完成时调用
func TestLevelPhaseSystem_AdvanceToConveyorSlide(t *testing.T) {
	t.Run("advances to conveyor slide step", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		system := NewLevelPhaseSystem(em, gs, nil)

		// 设置到 Dave 对话步骤
		phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](em, system.GetPhaseEntity())
		if !ok {
			t.Fatal("Expected phase component")
		}
		phaseComp.PhaseState = components.PhaseStateTransitioning
		phaseComp.TransitionStep = components.TransitionStepDaveDialogue

		// 调用公开方法推进到传送带滑入
		system.AdvanceToConveyorSlide()

		// 验证状态已更新
		if phaseComp.TransitionStep != components.TransitionStepConveyorSlide {
			t.Errorf("Expected TransitionStep %d, got %d",
				components.TransitionStepConveyorSlide, phaseComp.TransitionStep)
		}

		// 传送带应该可见
		if !phaseComp.ConveyorBeltVisible {
			t.Error("Expected conveyor belt to be visible")
		}

		// 进度应该重置
		if phaseComp.TransitionProgress != 0 {
			t.Errorf("Expected TransitionProgress 0, got %f", phaseComp.TransitionProgress)
		}
	})
}

// TestLevelPhaseSystem_IsInDaveDialogueStep 测试检查是否在 Dave 对话步骤
// Bug Fix: 用于存档恢复时判断是否需要设置特殊回调
func TestLevelPhaseSystem_IsInDaveDialogueStep(t *testing.T) {
	t.Run("returns false when not transitioning", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		system := NewLevelPhaseSystem(em, gs, nil)

		if system.IsInDaveDialogueStep() {
			t.Error("Expected false when not transitioning")
		}
	})

	t.Run("returns false when transitioning but not in Dave dialogue step", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		system := NewLevelPhaseSystem(em, gs, nil)

		// 设置为转场状态但不是 Dave 对话步骤
		phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](em, system.GetPhaseEntity())
		if !ok {
			t.Fatal("Expected phase component")
		}
		phaseComp.PhaseState = components.PhaseStateTransitioning
		phaseComp.TransitionStep = components.TransitionStepConveyorSlide

		if system.IsInDaveDialogueStep() {
			t.Error("Expected false when not in Dave dialogue step")
		}
	})

	t.Run("returns true when transitioning and in Dave dialogue step", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs := game.GetGameState()
		system := NewLevelPhaseSystem(em, gs, nil)

		// 设置为转场状态且在 Dave 对话步骤
		phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](em, system.GetPhaseEntity())
		if !ok {
			t.Fatal("Expected phase component")
		}
		phaseComp.PhaseState = components.PhaseStateTransitioning
		phaseComp.TransitionStep = components.TransitionStepDaveDialogue

		if !system.IsInDaveDialogueStep() {
			t.Error("Expected true when in Dave dialogue step during transition")
		}
	})
}
