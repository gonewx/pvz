package systems

import (
	"math"
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
)

// TestCameraSystem_NewCameraSystem 测试镜头系统的创建
func TestCameraSystem_NewCameraSystem(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState() // 使用单例

	cs := NewCameraSystem(em, gs)

	if cs == nil {
		t.Fatal("NewCameraSystem returned nil")
	}

	if cs.entityManager != em {
		t.Error("EntityManager not set correctly")
	}

	if cs.gameState != gs {
		t.Error("GameState not set correctly")
	}

	// 验证镜头实体已创建
	if cs.cameraEntity == 0 {
		t.Error("Camera entity not created")
	}

	// 验证镜头组件已添加
	cameraComp, ok := ecs.GetComponent[*components.CameraComponent](em, cs.cameraEntity)
	if !ok {
		t.Fatal("CameraComponent not added to camera entity")
	}

	// 验证默认值
	if cameraComp.TargetX != 0 || cameraComp.TargetY != 0 {
		t.Errorf("Expected default target (0, 0), got (%.0f, %.0f)", cameraComp.TargetX, cameraComp.TargetY)
	}

	if cameraComp.AnimationSpeed != 300 {
		t.Errorf("Expected default speed 300, got %.0f", cameraComp.AnimationSpeed)
	}

	if cameraComp.IsAnimating {
		t.Error("Expected IsAnimating to be false by default")
	}

	if cameraComp.EasingType != "easeInOut" {
		t.Errorf("Expected default easing type 'easeInOut', got %s", cameraComp.EasingType)
	}
}

// TestCameraSystem_MoveTo 测试镜头移动功能
func TestCameraSystem_MoveTo(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	initialCameraX := gs.CameraX                   // 保存初始值
	gs.CameraX = 0                                 // 重置为 0 用于测试
	defer func() { gs.CameraX = initialCameraX }() // 测试结束后恢复

	cs := NewCameraSystem(em, gs)

	// 测试移动到目标位置
	targetX := 800.0
	targetY := 0.0
	speed := 300.0

	cs.MoveTo(targetX, targetY, speed)

	cameraComp, ok := ecs.GetComponent[*components.CameraComponent](em, cs.cameraEntity)
	if !ok {
		t.Fatal("CameraComponent not found")
	}

	// 验证目标位置已设置
	if cameraComp.TargetX != targetX {
		t.Errorf("Expected TargetX=%.0f, got %.0f", targetX, cameraComp.TargetX)
	}

	if cameraComp.TargetY != targetY {
		t.Errorf("Expected TargetY=%.0f, got %.0f", targetY, cameraComp.TargetY)
	}

	// 验证速度已设置
	if cameraComp.AnimationSpeed != speed {
		t.Errorf("Expected speed=%.0f, got %.0f", speed, cameraComp.AnimationSpeed)
	}

	// 验证动画已启动
	if !cameraComp.IsAnimating {
		t.Error("Expected IsAnimating to be true after MoveTo")
	}

	// 验证起点和总距离已记录
	if cameraComp.StartX != 0 {
		t.Errorf("Expected StartX=0, got %.0f", cameraComp.StartX)
	}

	expectedDistance := math.Abs(targetX - 0)
	if cameraComp.TotalDistance != expectedDistance {
		t.Errorf("Expected TotalDistance=%.0f, got %.0f", expectedDistance, cameraComp.TotalDistance)
	}
}

// TestCameraSystem_Update_Linear 测试线性镜头移动
func TestCameraSystem_Update_Linear(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	initialCameraX := gs.CameraX
	gs.CameraX = 0
	defer func() { gs.CameraX = initialCameraX }()

	cs := NewCameraSystem(em, gs)

	// 设置线性移动
	cameraComp, _ := ecs.GetComponent[*components.CameraComponent](em, cs.cameraEntity)
	cameraComp.EasingType = "linear"
	cameraComp.TargetX = 100.0
	cameraComp.AnimationSpeed = 50.0 // 50 px/s
	cameraComp.IsAnimating = true
	cameraComp.StartX = 0
	cameraComp.TotalDistance = 100.0

	// 模拟 1 秒更新（应该移动 50px）
	dt := 1.0
	cs.Update(dt)

	// 验证镜头位置（线性移动，1秒应该移动50px）
	expectedX := 50.0
	if math.Abs(gs.CameraX-expectedX) > 1.0 {
		t.Errorf("Expected CameraX≈%.0f, got %.2f", expectedX, gs.CameraX)
	}

	// 验证仍在动画中
	cameraComp, _ = ecs.GetComponent[*components.CameraComponent](em, cs.cameraEntity)
	if !cameraComp.IsAnimating {
		t.Error("Expected IsAnimating to be true (not reached target yet)")
	}
}

// TestCameraSystem_Update_ReachTarget 测试镜头到达目标
func TestCameraSystem_Update_ReachTarget(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	initialCameraX := gs.CameraX
	gs.CameraX = 97.0 // 距离目标 3px
	defer func() { gs.CameraX = initialCameraX }()

	cs := NewCameraSystem(em, gs)

	// 设置接近目标的位置
	cameraComp, _ := ecs.GetComponent[*components.CameraComponent](em, cs.cameraEntity)
	cameraComp.TargetX = 100.0
	cameraComp.AnimationSpeed = 50.0
	cameraComp.IsAnimating = true
	cameraComp.StartX = 0
	cameraComp.TotalDistance = 100.0

	// 更新（应该到达目标）
	cs.Update(0.1)

	// 验证镜头已到达目标
	if gs.CameraX != 100.0 {
		t.Errorf("Expected CameraX=100.0 (reached target), got %.2f", gs.CameraX)
	}

	// 验证动画已停止
	cameraComp, _ = ecs.GetComponent[*components.CameraComponent](em, cs.cameraEntity)
	if cameraComp.IsAnimating {
		t.Error("Expected IsAnimating to be false (reached target)")
	}
}

// TestCameraSystem_Update_RangeLimit 测试镜头范围限制
func TestCameraSystem_Update_RangeLimit(t *testing.T) {
	gs := game.GetGameState()
	initialCameraX := gs.CameraX
	defer func() { gs.CameraX = initialCameraX }()

	// 测试左边界限制
	t.Run("LeftBoundary", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs.CameraX = 0
		cs := NewCameraSystem(em, gs)

		cameraComp, _ := ecs.GetComponent[*components.CameraComponent](em, cs.cameraEntity)
		cameraComp.TargetX = -100.0 // 超出左边界
		cameraComp.AnimationSpeed = 100.0
		cameraComp.IsAnimating = true
		cameraComp.StartX = 0
		cameraComp.TotalDistance = 100.0

		cs.Update(2.0)

		// 验证不会超出左边界
		if gs.CameraX < CameraMinX {
			t.Errorf("CameraX (%.2f) exceeds left boundary (%.0f)", gs.CameraX, CameraMinX)
		}
	})

	// 测试右边界限制
	t.Run("RightBoundary", func(t *testing.T) {
		em := ecs.NewEntityManager()
		gs.CameraX = 0
		cs := NewCameraSystem(em, gs)

		cameraComp, _ := ecs.GetComponent[*components.CameraComponent](em, cs.cameraEntity)
		cameraComp.TargetX = 1000.0 // 超出右边界
		cameraComp.AnimationSpeed = 100.0
		cameraComp.IsAnimating = true
		cameraComp.StartX = 0
		cameraComp.TotalDistance = 1000.0

		cs.Update(20.0)

		// 验证不会超出右边界
		if gs.CameraX > CameraMaxX {
			t.Errorf("CameraX (%.2f) exceeds right boundary (%.0f)", gs.CameraX, CameraMaxX)
		}
	})
}

// TestCameraSystem_StopAnimation 测试停止动画
func TestCameraSystem_StopAnimation(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	initialCameraX := gs.CameraX
	gs.CameraX = 0
	defer func() { gs.CameraX = initialCameraX }()

	cs := NewCameraSystem(em, gs)

	// 启动动画
	cs.MoveTo(800.0, 0.0, 300.0)

	cameraComp, _ := ecs.GetComponent[*components.CameraComponent](em, cs.cameraEntity)
	if !cameraComp.IsAnimating {
		t.Fatal("Expected animation to be started")
	}

	// 停止动画
	cs.StopAnimation()

	// 验证动画已停止
	cameraComp, _ = ecs.GetComponent[*components.CameraComponent](em, cs.cameraEntity)
	if cameraComp.IsAnimating {
		t.Error("Expected IsAnimating to be false after StopAnimation")
	}

	// 验证镜头已立即到达目标位置
	if gs.CameraX != 800.0 {
		t.Errorf("Expected CameraX=800.0 (jumped to target), got %.2f", gs.CameraX)
	}
}

// TestCameraSystem_IsAnimating 测试动画状态查询
func TestCameraSystem_IsAnimating(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	initialCameraX := gs.CameraX
	gs.CameraX = 0
	defer func() { gs.CameraX = initialCameraX }()

	cs := NewCameraSystem(em, gs)

	// 初始状态：未动画
	if cs.IsAnimating() {
		t.Error("Expected IsAnimating to be false initially")
	}

	// 启动动画
	cs.MoveTo(800.0, 0.0, 300.0)

	if !cs.IsAnimating() {
		t.Error("Expected IsAnimating to be true after MoveTo")
	}

	// 停止动画
	cs.StopAnimation()

	if cs.IsAnimating() {
		t.Error("Expected IsAnimating to be false after StopAnimation")
	}
}

// TestCameraSystem_EasingFunction 测试缓动函数
func TestCameraSystem_EasingFunction(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	cs := NewCameraSystem(em, gs)

	// 测试 easeInOutQuad
	t.Run("EaseInOutQuad", func(t *testing.T) {
		testCases := []struct {
			t        float64
			expected float64
		}{
			{0.0, 0.0},    // 起点
			{0.25, 0.125}, // 前半段加速
			{0.5, 0.5},    // 中点
			{0.75, 0.875}, // 后半段减速
			{1.0, 1.0},    // 终点
		}

		for _, tc := range testCases {
			result := cs.easeInOutQuad(tc.t)
			if math.Abs(result-tc.expected) > 0.001 {
				t.Errorf("easeInOutQuad(%.2f) = %.3f, want %.3f", tc.t, result, tc.expected)
			}
		}
	})

	// 测试 easeOutQuad
	t.Run("EaseOutQuad", func(t *testing.T) {
		testCases := []struct {
			t        float64
			expected float64
		}{
			{0.0, 0.0},  // 起点
			{0.5, 0.75}, // 快速减速
			{1.0, 1.0},  // 终点
		}

		for _, tc := range testCases {
			result := cs.easeOutQuad(tc.t)
			if math.Abs(result-tc.expected) > 0.001 {
				t.Errorf("easeOutQuad(%.2f) = %.3f, want %.3f", tc.t, result, tc.expected)
			}
		}
	})
}

// TestCameraSystem_Update_NotAnimating 测试非动画状态下的更新
func TestCameraSystem_Update_NotAnimating(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	initialCameraX := gs.CameraX

	cs := NewCameraSystem(em, gs)

	// 未启动动画，直接更新
	cs.Update(1.0)

	// 验证镜头位置未改变
	if gs.CameraX != initialCameraX {
		t.Errorf("Expected CameraX to remain %.2f, got %.2f", initialCameraX, gs.CameraX)
	}
}

// TestCameraSystem_Update_PreventOvershoot 测试防止越过目标
func TestCameraSystem_Update_PreventOvershoot(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	initialCameraX := gs.CameraX
	gs.CameraX = 5.0
	defer func() { gs.CameraX = initialCameraX }()

	cs := NewCameraSystem(em, gs)

	// 设置一个很大的速度，可能导致越过目标
	cameraComp, _ := ecs.GetComponent[*components.CameraComponent](em, cs.cameraEntity)
	cameraComp.TargetX = 10.0
	cameraComp.AnimationSpeed = 1000.0 // 非常快
	cameraComp.IsAnimating = true
	cameraComp.StartX = 0
	cameraComp.TotalDistance = 10.0

	// 更新一个大的时间步长
	cs.Update(1.0)

	// 验证没有越过目标
	if gs.CameraX > cameraComp.TargetX {
		t.Errorf("CameraX (%.2f) overshot target (%.0f)", gs.CameraX, cameraComp.TargetX)
	}

	// 验证已到达目标
	if gs.CameraX != cameraComp.TargetX {
		t.Errorf("Expected CameraX=%.0f, got %.2f", cameraComp.TargetX, gs.CameraX)
	}
}
