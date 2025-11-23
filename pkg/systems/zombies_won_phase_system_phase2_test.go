package systems

import (
	"math"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// 计算第1个目标位置（房门口，与系统代码保持一致）
func getZombieTarget1Position() (float64, float64) {
	targetX := config.GameOverDoorMaskX + Phase2ZombieTarget1OffsetX
	targetY := config.GameOverDoorMaskY + Phase2ZombieTarget1OffsetY
	return targetX, targetY
}

// 计算第2个目标位置（即将进入房子）
func getZombieTarget2X() float64 {
	target1X := config.GameOverDoorMaskX + Phase2ZombieTarget1OffsetX
	return target1X + Phase2ZombieTarget2OffsetX
}

// TestZombiesWonPhaseSystem_Phase2ZombieReachesTargetThenWalksIntoDoor 测试僵尸到达第1个目标位置后继续向左走进门
// Story 8.8 AC: 僵尸到达房门目标位置后，只在 X 轴继续前进（VY=0），模拟走进房门效果
func TestZombiesWonPhaseSystem_Phase2ZombieReachesTarget1ThenWalksIntoDoor(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := &game.GameState{}
	system := NewZombiesWonPhaseSystem(em, nil, gs, 800, 600)

	target1X, target1Y := getZombieTarget1Position()

	// 创建触发僵尸，已经非常接近第1个目标位置（在阈值范围内）
	zombieID := em.CreateEntity()
	ecs.AddComponent(em, zombieID, &components.PositionComponent{
		X: target1X + 2, // 距离第1个目标很近
		Y: target1Y - 2,
	})
	ecs.AddComponent(em, zombieID, &components.VelocityComponent{})
	ecs.AddComponent(em, zombieID, &components.ReanimComponent{
		CurrentAnimations: []string{"anim_walk"},
	})

	phaseID := em.CreateEntity()
	phaseComp := &components.ZombiesWonPhaseComponent{
		CurrentPhase:         2,
		PhaseTimer:           0.5,
		TriggerZombieID:      zombieID,
		CameraMovedToTarget:  true,
		ZombieStartedWalking: true,
	}
	ecs.AddComponent(em, phaseID, phaseComp)

	system.updatePhase2ZombieEntry(phaseID, phaseComp, 0.016)

	velComp, ok := ecs.GetComponent[*components.VelocityComponent](em, zombieID)
	if !ok {
		t.Fatalf("velocity component missing")
	}

	// 僵尸到达第1个目标后应该继续向左走进门
	if velComp.VX != -Phase2ZombieHorizontalSpeed {
		t.Fatalf("expected VX to continue moving left (-%.2f), got %.2f", Phase2ZombieHorizontalSpeed, velComp.VX)
	}
	// Y 方向应该停止（VY=0）
	if velComp.VY != 0 {
		t.Fatalf("expected VY to be 0 after reaching target 1, got %.2f", velComp.VY)
	}

	// ZombieReachedTarget1 应该被标记为 true
	if !phaseComp.ZombieReachedTarget1 {
		t.Fatalf("expected ZombieReachedTarget1 to be true after reaching target 1")
	}

	// ZombieReachedTarget（第2个目标）应该仍然是 false
	if phaseComp.ZombieReachedTarget {
		t.Fatalf("expected ZombieReachedTarget to be false (zombie just reached target 1)")
	}

	// X 坐标应该被 snap 到第1个目标位置，Y 坐标保持不变（避免突然跳变）
	posComp, _ := ecs.GetComponent[*components.PositionComponent](em, zombieID)
	initialY := target1Y - 2 // 测试设置的初始 Y 坐标
	if posComp.X != target1X {
		t.Fatalf("expected X to snap to target 1 X (%.2f), got %.2f", target1X, posComp.X)
	}
	if posComp.Y != initialY {
		t.Fatalf("expected Y to remain unchanged (%.2f), got %.2f", initialY, posComp.Y)
	}
}

// TestZombiesWonPhaseSystem_Phase2ZombieMovesInStraightLine 测试僵尸沿直线走向第1个目标
// 向量移动：僵尸应该直线走向目标，而不是分别在 X/Y 轴移动
func TestZombiesWonPhaseSystem_Phase2ZombieMovesInStraightLine(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := &game.GameState{}
	system := NewZombiesWonPhaseSystem(em, nil, gs, 800, 600)

	target1X, target1Y := getZombieTarget1Position()

	// 创建触发僵尸，远离第1个目标位置
	startX := target1X + 100
	startY := target1Y - 80 // 需要向下移动
	zombieID := em.CreateEntity()
	ecs.AddComponent(em, zombieID, &components.PositionComponent{
		X: startX,
		Y: startY,
	})
	ecs.AddComponent(em, zombieID, &components.VelocityComponent{})
	ecs.AddComponent(em, zombieID, &components.ReanimComponent{
		CurrentAnimations: []string{"anim_walk"},
	})

	phaseID := em.CreateEntity()
	phaseComp := &components.ZombiesWonPhaseComponent{
		CurrentPhase:         2,
		PhaseTimer:           0.5,
		TriggerZombieID:      zombieID,
		CameraMovedToTarget:  true,
		ZombieStartedWalking: true,
	}
	ecs.AddComponent(em, phaseID, phaseComp)

	system.updatePhase2ZombieEntry(phaseID, phaseComp, 0.016)

	velComp, ok := ecs.GetComponent[*components.VelocityComponent](em, zombieID)
	if !ok {
		t.Fatalf("velocity component missing")
	}

	// 计算预期的方向向量
	dx := target1X - startX // 负数（向左）
	dy := target1Y - startY // 正数（向下）
	distance := math.Sqrt(dx*dx + dy*dy)
	expectedDirX := dx / distance
	expectedDirY := dy / distance

	// 验证速度方向正确（沿直线走向目标）
	actualSpeed := math.Sqrt(velComp.VX*velComp.VX + velComp.VY*velComp.VY)
	actualDirX := velComp.VX / actualSpeed
	actualDirY := velComp.VY / actualSpeed

	// 允许微小误差
	if math.Abs(actualDirX-expectedDirX) > 0.01 || math.Abs(actualDirY-expectedDirY) > 0.01 {
		t.Fatalf("expected direction (%.4f, %.4f), got (%.4f, %.4f)",
			expectedDirX, expectedDirY, actualDirX, actualDirY)
	}

	// VX 应该是负数（向左）
	if velComp.VX >= 0 {
		t.Fatalf("expected VX < 0 (moving left), got %.2f", velComp.VX)
	}
	// VY 应该是正数（向下，因为 target1Y > startY）
	if velComp.VY <= 0 {
		t.Fatalf("expected VY > 0 (moving down toward target 1), got %.2f", velComp.VY)
	}

	// ZombieReachedTarget1 和 ZombieReachedTarget 应该仍然是 false
	if phaseComp.ZombieReachedTarget1 || phaseComp.ZombieReachedTarget {
		t.Fatalf("expected ZombieReachedTarget1/ZombieReachedTarget to be false while moving toward target")
	}

	t.Logf("Zombie velocity: VX=%.2f, VY=%.2f", velComp.VX, velComp.VY)
	t.Logf("Direction: (%.4f, %.4f)", actualDirX, actualDirY)
}

// TestZombiesWonPhaseSystem_Phase2TransitionsToPhase3 测试进入 Phase 3 的条件
// 僵尸必须到达第2个目标位置（即将进入房子）才能触发 Phase 3
func TestZombiesWonPhaseSystem_Phase2TransitionsToPhase3(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := &game.GameState{}
	system := NewZombiesWonPhaseSystem(em, nil, gs, 800, 600)

	_, target1Y := getZombieTarget1Position()
	target2X := getZombieTarget2X()

	// 僵尸已经到达第1个目标，并且已经走到第2个目标位置
	zombieID := em.CreateEntity()
	ecs.AddComponent(em, zombieID, &components.PositionComponent{
		X: target2X - 1, // 已经到达第2个目标（X <= target2X）
		Y: target1Y,
	})
	ecs.AddComponent(em, zombieID, &components.VelocityComponent{})
	ecs.AddComponent(em, zombieID, &components.ReanimComponent{
		CurrentAnimations: []string{"anim_walk"},
	})

	phaseID := em.CreateEntity()
	phaseComp := &components.ZombiesWonPhaseComponent{
		CurrentPhase:          2,
		PhaseTimer:            0.5,
		TriggerZombieID:       zombieID,
		CameraMovedToTarget:   true, // 摄像机已到位
		ZombieStartedWalking:  true,
		ZombieReachedTarget1:  true, // 已到达第1个目标
		ZombieReachedTarget:   false,
	}
	ecs.AddComponent(em, phaseID, phaseComp)

	system.updatePhase2ZombieEntry(phaseID, phaseComp, 0.016)

	// 验证转换到 Phase 3
	if phaseComp.CurrentPhase != 3 {
		t.Fatalf("expected to transition to Phase 3, got phase %d", phaseComp.CurrentPhase)
	}
	if !phaseComp.ZombieReachedTarget {
		t.Fatalf("expected ZombieReachedTarget flag to be true")
	}

	velComp, _ := ecs.GetComponent[*components.VelocityComponent](em, zombieID)

	// 僵尸到达第2个目标后继续向左移动（VX=-30, VY=0）
	if velComp.VX != -Phase2ZombieHorizontalSpeed {
		t.Fatalf("expected VX to be -%.2f (continue walking into door), got %.2f", Phase2ZombieHorizontalSpeed, velComp.VX)
	}
	if velComp.VY != 0 {
		t.Fatalf("expected VY to be 0, got %.2f", velComp.VY)
	}

	t.Logf("Phase 3 triggered at zombie X=%.2f (target2X=%.2f)", target2X-1, target2X)
}

func TestZombiesWonPhaseSystem_Phase2CameraAndZombieMoveSimultaneously(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := &game.GameState{
		CameraX: 400.0, // 初始摄像机位置
	}
	system := NewZombiesWonPhaseSystem(em, nil, gs, 800, 600)

	target1X, target1Y := getZombieTarget1Position()

	// 创建触发僵尸，位于第1个目标位置远处
	zombieID := em.CreateEntity()
	ecs.AddComponent(em, zombieID, &components.PositionComponent{
		X: 500.0, // 远离目标
		Y: target1Y,
	})
	ecs.AddComponent(em, zombieID, &components.VelocityComponent{})
	ecs.AddComponent(em, zombieID, &components.ReanimComponent{
		CurrentAnimations: []string{"anim_walk"},
	})

	phaseID := em.CreateEntity()
	phaseComp := &components.ZombiesWonPhaseComponent{
		CurrentPhase:    2,
		PhaseTimer:      0.0,
		TriggerZombieID: zombieID,
	}
	ecs.AddComponent(em, phaseID, phaseComp)

	// 第一帧：初始化
	deltaTime := 0.016
	phaseComp.PhaseTimer = deltaTime
	system.updatePhase2ZombieEntry(phaseID, phaseComp, deltaTime)

	// 验证摄像机和僵尸同时开始移动
	if !phaseComp.ZombieStartedWalking {
		t.Fatalf("expected zombie to start walking immediately in Phase 2")
	}

	// 第二帧：验证同时移动
	initialCameraX := gs.CameraX
	phaseComp.PhaseTimer += deltaTime
	system.updatePhase2ZombieEntry(phaseID, phaseComp, deltaTime)

	// 摄像机应该向左移动
	if gs.CameraX >= initialCameraX {
		t.Fatalf("expected camera to move left, but CameraX=%f (initial=%f)", gs.CameraX, initialCameraX)
	}

	// 僵尸应该向左移动
	velComp, _ := ecs.GetComponent[*components.VelocityComponent](em, zombieID)
	if velComp.VX >= 0 {
		t.Fatalf("expected zombie to move left (VX < 0), but VX=%f", velComp.VX)
	}

	t.Logf("Camera moved from %f to %f", initialCameraX, gs.CameraX)
	t.Logf("Zombie VX=%f, VY=%f", velComp.VX, velComp.VY)
	t.Logf("Target 1 position: (%f, %f)", target1X, target1Y)
}
