package components

import (
	"testing"

	"github.com/gonewx/pvz/pkg/ecs"
)

func TestZombiesWonPhaseComponent_InitialState(t *testing.T) {
	comp := &ZombiesWonPhaseComponent{
		CurrentPhase:         1,
		PhaseTimer:           0.0,
		TriggerZombieID:      42,
		ZombieReachedTarget:  false,
		CameraMovedToTarget:  false,
		InitialCameraX:       0.0,
		ZombieStartedWalking: false,
		ScreamPlayed:         false,
		ChompPlayed:          false,
		AnimationReady:       false,
		ScreenShakeTime:      0.0,
		DialogShown:          false,
		WaitTimer:            0.0,
	}

	if comp.CurrentPhase != 1 {
		t.Errorf("Expected CurrentPhase=1, got=%d", comp.CurrentPhase)
	}
	if comp.TriggerZombieID != 42 {
		t.Errorf("Expected TriggerZombieID=42, got=%d", comp.TriggerZombieID)
	}
	if comp.ZombieReachedTarget {
		t.Error("Expected ZombieReachedTarget=false")
	}
	if comp.CameraMovedToTarget {
		t.Error("Expected CameraMovedToTarget=false")
	}
	if comp.ZombieStartedWalking {
		t.Error("Expected ZombieStartedWalking=false")
	}
	if comp.ScreamPlayed {
		t.Error("Expected ScreamPlayed=false")
	}
	if comp.ChompPlayed {
		t.Error("Expected ChompPlayed=false")
	}
	if comp.AnimationReady {
		t.Error("Expected AnimationReady=false")
	}
	if comp.DialogShown {
		t.Error("Expected DialogShown=false")
	}
}

func TestZombiesWonPhaseComponent_PhaseTransitions(t *testing.T) {
	tests := []struct {
		name         string
		currentPhase int
		nextPhase    int
	}{
		{"Phase 1 to 2", 1, 2},
		{"Phase 2 to 3", 2, 3},
		{"Phase 3 to 4", 3, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := &ZombiesWonPhaseComponent{
				CurrentPhase: tt.currentPhase,
			}

			// 模拟阶段转换
			comp.CurrentPhase = tt.nextPhase

			if comp.CurrentPhase != tt.nextPhase {
				t.Errorf("Expected CurrentPhase=%d, got=%d", tt.nextPhase, comp.CurrentPhase)
			}
		})
	}
}

func TestZombiesWonPhaseComponent_Phase2Fields(t *testing.T) {
	var zombieID ecs.EntityID = 100

	comp := &ZombiesWonPhaseComponent{
		CurrentPhase:         2,
		TriggerZombieID:      zombieID,
		ZombieReachedTarget:  false,
		CameraMovedToTarget:  false,
		InitialCameraX:       400.0,
		ZombieStartedWalking: false,
	}

	if comp.TriggerZombieID != zombieID {
		t.Errorf("Expected TriggerZombieID=%d, got=%d", zombieID, comp.TriggerZombieID)
	}

	if comp.InitialCameraX != 400.0 {
		t.Errorf("Expected InitialCameraX=400.0, got=%f", comp.InitialCameraX)
	}

	// 模拟僵尸开始行走
	comp.ZombieStartedWalking = true
	if !comp.ZombieStartedWalking {
		t.Error("Expected ZombieStartedWalking=true after zombie starts walking")
	}

	// 模拟摄像机移动到目标位置
	comp.CameraMovedToTarget = true
	if !comp.CameraMovedToTarget {
		t.Error("Expected CameraMovedToTarget=true after camera reaches target")
	}

	// 模拟僵尸到达目标位置
	comp.ZombieReachedTarget = true
	if !comp.ZombieReachedTarget {
		t.Error("Expected ZombieReachedTarget=true after zombie reaches target")
	}

	// 验证两个条件都满足时可以进入 Phase 3
	if comp.CameraMovedToTarget && comp.ZombieReachedTarget {
		// 这是进入 Phase 3 的条件
		if comp.CurrentPhase != 2 {
			t.Error("Expected to remain in Phase 2 until system transitions")
		}
	}
}

func TestZombiesWonPhaseComponent_Phase3Fields(t *testing.T) {
	comp := &ZombiesWonPhaseComponent{
		CurrentPhase:    3,
		ScreamPlayed:    false,
		ChompPlayed:     false,
		AnimationReady:  false,
		ScreenShakeTime: 0.0,
	}

	// 模拟音效播放
	comp.ScreamPlayed = true
	if !comp.ScreamPlayed {
		t.Error("Expected ScreamPlayed=true")
	}

	// 模拟咀嚼音效播放
	comp.ChompPlayed = true
	if !comp.ChompPlayed {
		t.Error("Expected ChompPlayed=true")
	}

	// 模拟屏幕抖动计时器更新
	comp.ScreenShakeTime = 1.5
	if comp.ScreenShakeTime != 1.5 {
		t.Errorf("Expected ScreenShakeTime=1.5, got=%f", comp.ScreenShakeTime)
	}

	// 模拟动画准备完毕
	comp.AnimationReady = true
	if !comp.AnimationReady {
		t.Error("Expected AnimationReady=true")
	}
}

func TestZombiesWonPhaseComponent_Phase4Fields(t *testing.T) {
	comp := &ZombiesWonPhaseComponent{
		CurrentPhase: 4,
		DialogShown:  false,
		WaitTimer:    0.0,
	}

	// 模拟等待计时器更新
	comp.WaitTimer = 2.5
	if comp.WaitTimer != 2.5 {
		t.Errorf("Expected WaitTimer=2.5, got=%f", comp.WaitTimer)
	}

	// 模拟对话框显示
	comp.DialogShown = true
	if !comp.DialogShown {
		t.Error("Expected DialogShown=true")
	}
}

func TestZombiesWonPhaseComponent_TimerUpdate(t *testing.T) {
	comp := &ZombiesWonPhaseComponent{
		CurrentPhase: 1,
		PhaseTimer:   0.0,
	}

	// 模拟时间推进
	deltaTime := 0.016 // 1 frame at 60 FPS
	comp.PhaseTimer += deltaTime

	if comp.PhaseTimer < deltaTime {
		t.Errorf("Expected PhaseTimer >= %f, got=%f", deltaTime, comp.PhaseTimer)
	}
}
