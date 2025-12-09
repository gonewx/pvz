package systems

import (
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
)

// ========================================
// Story 8.8: End-to-End Integration Tests
// ========================================

// TestZombiesWonPhaseSystem_Phase1ToPhase2Transition 测试 Phase 1 到 Phase 2 的过渡
func TestZombiesWonPhaseSystem_Phase1ToPhase2Transition(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := &game.GameState{
		CameraX: 400.0, // 初始摄像机位置
	}
	system := NewZombiesWonPhaseSystem(em, nil, gs, 800, 600)

	// 创建触发僵尸
	zombieID := em.CreateEntity()
	ecs.AddComponent(em, zombieID, &components.PositionComponent{
		X: 300.0,
		Y: 300.0,
	})
	ecs.AddComponent(em, zombieID, &components.VelocityComponent{})

	// 使用 StartZombiesWonFlow 创建流程实体
	flowEntityID := StartZombiesWonFlow(em, zombieID)

	// 验证初始状态
	phaseComp, ok := ecs.GetComponent[*components.ZombiesWonPhaseComponent](em, flowEntityID)
	if !ok {
		t.Fatalf("Phase component not found")
	}
	if phaseComp.CurrentPhase != 1 {
		t.Errorf("expected Phase 1, got %d", phaseComp.CurrentPhase)
	}

	// 模拟 Phase 1 冻结期间（1.5 秒）
	for i := 0; i < 94; i++ { // 94 帧 * 0.016 秒 ≈ 1.5 秒
		system.Update(0.016)
	}

	// 验证过渡到 Phase 2
	if phaseComp.CurrentPhase != 2 {
		t.Errorf("expected transition to Phase 2 after 1.5 seconds, got Phase %d", phaseComp.CurrentPhase)
	}

	// 验证原始摄像机位置已保存
	if system.originalCameraX != 400.0 {
		t.Errorf("expected originalCameraX=400.0, got %f", system.originalCameraX)
	}
}

// TestZombiesWonPhaseSystem_Phase2SimultaneousMovement 测试 Phase 2 摄像机和僵尸同时移动
func TestZombiesWonPhaseSystem_Phase2SimultaneousMovement(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := &game.GameState{
		CameraX: 400.0,
	}
	system := NewZombiesWonPhaseSystem(em, nil, gs, 800, 600)

	_, targetY := getZombieTarget1Position()

	// 创建触发僵尸（远离目标位置）
	zombieID := em.CreateEntity()
	ecs.AddComponent(em, zombieID, &components.PositionComponent{
		X: 500.0,
		Y: targetY,
	})
	ecs.AddComponent(em, zombieID, &components.VelocityComponent{})
	ecs.AddComponent(em, zombieID, &components.ReanimComponent{
		CurrentAnimations: []string{"anim_walk"},
	})

	// 创建流程实体并直接进入 Phase 2
	flowEntityID := em.CreateEntity()
	phaseComp := &components.ZombiesWonPhaseComponent{
		CurrentPhase:    2,
		PhaseTimer:      0.0,
		TriggerZombieID: zombieID,
	}
	ecs.AddComponent(em, flowEntityID, phaseComp)

	initialCameraX := gs.CameraX
	initialZombieX := 500.0

	// 执行几帧更新
	for i := 0; i < 10; i++ {
		system.Update(0.016)
	}

	// 验证摄像机向左移动
	if gs.CameraX >= initialCameraX {
		t.Errorf("Camera should move left. Initial: %f, Current: %f", initialCameraX, gs.CameraX)
	}

	// 验证僵尸向左移动
	posComp, ok := ecs.GetComponent[*components.PositionComponent](em, zombieID)
	if !ok {
		t.Fatalf("Position component not found")
	}

	velComp, ok := ecs.GetComponent[*components.VelocityComponent](em, zombieID)
	if !ok {
		t.Fatalf("Velocity component not found")
	}

	// 僵尸应该在移动（VX < 0）
	if velComp.VX >= 0 {
		t.Errorf("Zombie should move left (VX < 0). VX: %f", velComp.VX)
	}

	// 验证同时执行标志
	if !phaseComp.ZombieStartedWalking {
		t.Error("ZombieStartedWalking should be true")
	}

	t.Logf("Camera moved from %f to %f", initialCameraX, gs.CameraX)
	t.Logf("Zombie at X=%f (initial=%f), VX=%f", posComp.X, initialZombieX, velComp.VX)
}

// TestZombiesWonPhaseSystem_Phase2ToPhase3Transition 测试 Phase 2 到 Phase 3 的过渡
func TestZombiesWonPhaseSystem_Phase2ToPhase3Transition(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := &game.GameState{
		CameraX: 0.0, // 摄像机已经到达目标
	}
	system := NewZombiesWonPhaseSystem(em, nil, gs, 800, 600)

	_, target1Y := getZombieTarget1Position()
	target2X := getZombieTarget2X()

	// 创建触发僵尸（已经到达第2个目标位置，即将进入房子）
	zombieID := em.CreateEntity()
	ecs.AddComponent(em, zombieID, &components.PositionComponent{
		X: target2X - 1, // 已经到达第2个目标（X <= target2X）
		Y: target1Y,
	})
	ecs.AddComponent(em, zombieID, &components.VelocityComponent{})
	ecs.AddComponent(em, zombieID, &components.ReanimComponent{
		CurrentAnimations: []string{"anim_walk"},
	})

	// 创建流程实体（Phase 2，摄像机已到位，僵尸已到达第1个目标）
	flowEntityID := em.CreateEntity()
	phaseComp := &components.ZombiesWonPhaseComponent{
		CurrentPhase:         2,
		PhaseTimer:           0.5,
		TriggerZombieID:      zombieID,
		CameraMovedToTarget:  true,
		ZombieStartedWalking: true,
		ZombieReachedTarget1: true, // 已到达第1个目标
	}
	ecs.AddComponent(em, flowEntityID, phaseComp)

	// 执行一帧更新
	system.Update(0.016)

	// 验证过渡到 Phase 3
	if phaseComp.CurrentPhase != 3 {
		t.Errorf("expected transition to Phase 3, got Phase %d", phaseComp.CurrentPhase)
	}

	// 验证僵尸到达标志
	if !phaseComp.ZombieReachedTarget {
		t.Error("ZombieReachedTarget should be true")
	}

	// 验证僵尸继续向左移动（模拟走进门效果）
	velComp, ok := ecs.GetComponent[*components.VelocityComponent](em, zombieID)
	if !ok {
		t.Fatalf("Velocity component not found")
	}
	if velComp.VX != -Phase2ZombieHorizontalSpeed {
		t.Errorf("Zombie should continue walking left (VX=%f), got VX=%f",
			-Phase2ZombieHorizontalSpeed, velComp.VX)
	}
	if velComp.VY != 0 {
		t.Errorf("Zombie VY should be 0, got %f", velComp.VY)
	}
}

// TestZombiesWonPhaseSystem_Phase3AudioAndAnimation 测试 Phase 3 音效和动画触发
func TestZombiesWonPhaseSystem_Phase3AudioAndAnimation(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(nil) // 无音频上下文，测试不会崩溃
	gs := &game.GameState{
		CameraX: 0.0,
	}
	system := NewZombiesWonPhaseSystem(em, rm, gs, 800, 600)

	// 创建触发僵尸（已走出画面）
	zombieID := em.CreateEntity()
	ecs.AddComponent(em, zombieID, &components.PositionComponent{
		X: -200.0, // 已经走出画面
		Y: 300.0,
	})
	ecs.AddComponent(em, zombieID, &components.VelocityComponent{
		VX: -150.0,
	})

	// 创建流程实体（Phase 3）
	flowEntityID := em.CreateEntity()
	phaseComp := &components.ZombiesWonPhaseComponent{
		CurrentPhase:        3,
		PhaseTimer:          0.0,
		TriggerZombieID:     zombieID,
		ZombieReachedTarget: true,
	}
	ecs.AddComponent(em, flowEntityID, phaseComp)

	// 第一帧：触发惨叫音效
	system.Update(0.016)
	if !phaseComp.ScreamPlayed {
		t.Error("ScreamPlayed should be true after first frame")
	}
	if phaseComp.ChompPlayed {
		t.Error("ChompPlayed should still be false (delay not reached)")
	}

	// 继续更新直到咀嚼音效触发（0.5 秒后）
	for i := 0; i < 31; i++ { // 约 0.5 秒
		system.Update(0.016)
	}

	if !phaseComp.ChompPlayed {
		t.Error("ChompPlayed should be true after 0.5 seconds")
	}

	// 验证僵尸已停止（因为 X < -100）
	velComp, _ := ecs.GetComponent[*components.VelocityComponent](em, zombieID)
	if velComp.VX != 0 {
		t.Errorf("Zombie should stop when X < -100, VX=%f", velComp.VX)
	}
}

// TestZombiesWonPhaseSystem_FullFlowSimulation 端到端完整流程测试
// 注意：此测试模拟位置更新（因为 MovementSystem 不在测试范围内）
func TestZombiesWonPhaseSystem_FullFlowSimulation(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := &game.GameState{
		CameraX: 100.0, // 初始摄像机位置（非常近，加速测试）
	}
	system := NewZombiesWonPhaseSystem(em, nil, gs, 800, 600)

	target1X, target1Y := getZombieTarget1Position()
	target2X := getZombieTarget2X()

	// 创建触发僵尸（非常接近目标位置，加速测试）
	zombieID := em.CreateEntity()
	posComp := &components.PositionComponent{
		X: target1X + 10, // 距离第1个目标 10 像素
		Y: target1Y + 5,  // 距离第1个目标 5 像素
	}
	ecs.AddComponent(em, zombieID, posComp)
	velComp := &components.VelocityComponent{}
	ecs.AddComponent(em, zombieID, velComp)
	ecs.AddComponent(em, zombieID, &components.ReanimComponent{
		CurrentAnimations: []string{"anim_walk"},
	})

	// 启动流程
	flowEntityID := StartZombiesWonFlow(em, zombieID)

	phaseComp, _ := ecs.GetComponent[*components.ZombiesWonPhaseComponent](em, flowEntityID)

	// Phase 1: 冻结 1.5 秒
	t.Log("Phase 1: Game Freeze")
	frameCount := 0
	for phaseComp.CurrentPhase == 1 && frameCount < 200 {
		system.Update(0.016)
		frameCount++
	}
	if phaseComp.CurrentPhase != 2 {
		t.Fatalf("Failed to transition to Phase 2 after %d frames", frameCount)
	}
	t.Logf("Transitioned to Phase 2 after %d frames (~%.2f seconds)", frameCount, float64(frameCount)*0.016)

	// Phase 2: 摄像机和僵尸移动
	// 注意：需要手动模拟位置更新（因为 MovementSystem 不在测试范围内）
	t.Log("Phase 2: Camera and Zombie Movement")
	frameCount = 0
	deltaTime := 0.016
	// 增加帧数限制以适应两个目标位置的逻辑
	// 僵尸需要先到达 target1，然后继续走到 target2
	for phaseComp.CurrentPhase == 2 && frameCount < 500 {
		system.Update(deltaTime)

		// 手动模拟位置更新（MovementSystem 的职责）
		posComp.X += velComp.VX * deltaTime
		posComp.Y += velComp.VY * deltaTime

		frameCount++
	}
	if phaseComp.CurrentPhase != 3 {
		t.Fatalf("Failed to transition to Phase 3 after %d frames. Camera: %f, Zombie: (%f, %f), Target1: (%f, %f), Target2X: %f",
			frameCount, gs.CameraX, posComp.X, posComp.Y, target1X, target1Y, target2X)
	}
	t.Logf("Transitioned to Phase 3 after %d frames (~%.2f seconds)", frameCount, float64(frameCount)*deltaTime)

	// 验证摄像机到达目标位置
	if gs.CameraX > Phase2CameraTargetX+0.1 {
		t.Errorf("Camera should reach target (0), got %f", gs.CameraX)
	}

	// 验证僵尸到达第2个目标位置（允许微小误差）
	if posComp.X > target2X+Phase2ZombieReachThreshold {
		t.Errorf("Zombie should be at or past target 2 X (%f), got %f", target2X, posComp.X)
	}

	t.Log("Phase 3: Scream and Animation (not fully tested - requires animation assets)")

	// 验证 Phase 3 状态正确
	if !phaseComp.ZombieReachedTarget {
		t.Error("ZombieReachedTarget should be true")
	}
	if !phaseComp.CameraMovedToTarget {
		t.Error("CameraMovedToTarget should be true")
	}
}

// TestZombiesWonPhaseSystem_GameFreezeComponentPresent 测试 GameFreezeComponent 存在性
func TestZombiesWonPhaseSystem_GameFreezeComponentPresent(t *testing.T) {
	em := ecs.NewEntityManager()

	// 创建触发僵尸
	zombieID := em.CreateEntity()
	ecs.AddComponent(em, zombieID, &components.PositionComponent{X: 300, Y: 300})

	// 启动流程
	flowEntityID := StartZombiesWonFlow(em, zombieID)

	// 验证 GameFreezeComponent 存在
	freezeComp, ok := ecs.GetComponent[*components.GameFreezeComponent](em, flowEntityID)
	if !ok {
		t.Fatal("GameFreezeComponent should be present on flow entity")
	}

	// 验证冻结状态
	if !freezeComp.IsFrozen {
		t.Error("IsFrozen should be true")
	}
}

// TestZombiesWonPhaseSystem_TargetPositionCalculation 测试目标位置计算
func TestZombiesWonPhaseSystem_TargetPositionCalculation(t *testing.T) {
	// 测试第1个目标位置
	expectedTarget1X := config.GameOverDoorMaskX + Phase2ZombieTarget1OffsetX
	expectedTarget1Y := config.GameOverDoorMaskY + Phase2ZombieTarget1OffsetY

	actualTarget1X, actualTarget1Y := getZombieTarget1Position()

	if actualTarget1X != expectedTarget1X {
		t.Errorf("Target 1 X mismatch: expected %f, got %f", expectedTarget1X, actualTarget1X)
	}
	if actualTarget1Y != expectedTarget1Y {
		t.Errorf("Target 1 Y mismatch: expected %f, got %f", expectedTarget1Y, actualTarget1Y)
	}

	// 测试第2个目标位置
	expectedTarget2X := expectedTarget1X + Phase2ZombieTarget2OffsetX
	actualTarget2X := getZombieTarget2X()

	if actualTarget2X != expectedTarget2X {
		t.Errorf("Target 2 X mismatch: expected %f, got %f", expectedTarget2X, actualTarget2X)
	}

	t.Logf("Target 1 position: (%f, %f)", actualTarget1X, actualTarget1Y)
	t.Logf("Target 2 X: %f", actualTarget2X)
	t.Logf("GameOverDoorMaskX: %f, GameOverDoorMaskY: %f", config.GameOverDoorMaskX, config.GameOverDoorMaskY)
	t.Logf("Offsets: Target1 X=%f, Y=%f; Target2 X=%f", Phase2ZombieTarget1OffsetX, Phase2ZombieTarget1OffsetY, Phase2ZombieTarget2OffsetX)
}
