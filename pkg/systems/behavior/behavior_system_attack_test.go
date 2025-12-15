package behavior

import (
	"testing"

	"github.com/gonewx/pvz/internal/reanim"
	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
)

// ============================================================================
// Plant Attack Animation System Tests (Simplified)
// ============================================================================
//
// 重构后的攻击系统测试
// - 攻击动画完全控制攻击节奏，不需要冷却计时器
// - 有僵尸时播放攻击动画（循环）
// - 动画到达关键帧时发射子弹
// - 没有僵尸时切回 idle

// ============================================================================
// Unit Tests
// ============================================================================

// TestTriggerPlantAttackAnimation tests that peashooter switches to attack animation when zombie is detected
func TestTriggerPlantAttackAnimation(t *testing.T) {
	// Given: A peashooter plant entity with idle animation state
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	rs := systems.NewReanimSystem(em)

	configManager, err := config.NewReanimConfigManager("data/reanim_config.yaml")
	if err != nil {
		t.Skipf("跳过测试：无法加载配置文件: %v", err)
	}
	rs.SetConfigManager(configManager)

	gs := game.GetGameState()
	bs := createTestBehaviorSystem(em, rm, gs)

	// Create peashooter entity
	peashooterID := createTestPeashooter(em, rs)

	// Verify initial state is Idle
	plant, ok := ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	if !ok {
		t.Fatal("Failed to get PlantComponent")
	}
	if plant.AttackAnimState != components.AttackAnimIdle {
		t.Errorf("Initial state should be AttackAnimIdle, got %v", plant.AttackAnimState)
	}

	// Create a zombie to trigger attack
	zombieID := createTestZombie(em, 500.0, 300.0)

	// When: handlePeashooterBehavior is called with zombie in range
	bs.handlePeashooterBehavior(peashooterID, 0.016, []ecs.EntityID{zombieID})

	// Then: Plant state should change to Attacking
	plant, _ = ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	if plant.AttackAnimState != components.AttackAnimAttacking {
		t.Errorf("Expected AttackAnimState to be Attacking, got %v", plant.AttackAnimState)
	}

	// Then: Reanim component should be playing (attack animation is looping)
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](em, peashooterID)
	if !ok {
		t.Fatal("Failed to get ReanimComponent")
	}
	if !reanim.IsLooping {
		t.Error("Attack animation should be looping")
	}
}

// TestUpdatePlantAttackAnimation_ReturnToIdle tests that plant returns to idle when no zombies
func TestUpdatePlantAttackAnimation_ReturnToIdle(t *testing.T) {
	// Given: A peashooter in Attacking state
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	rs := systems.NewReanimSystem(em)

	configManager, err := config.NewReanimConfigManager("data/reanim_config.yaml")
	if err != nil {
		t.Skipf("跳过测试：无法加载配置文件: %v", err)
	}
	rs.SetConfigManager(configManager)

	gs := game.GetGameState()
	bs := createTestBehaviorSystem(em, rm, gs)

	peashooterID := createTestPeashooter(em, rs)

	// Set plant to Attacking state
	plant, _ := ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	plant.AttackAnimState = components.AttackAnimAttacking

	// Attack animation is looping
	reanim, _ := ecs.GetComponent[*components.ReanimComponent](em, peashooterID)
	reanim.IsLooping = true
	reanim.IsFinished = false

	// When: handlePeashooterBehavior is called with NO zombies
	bs.handlePeashooterBehavior(peashooterID, 0.016, []ecs.EntityID{})

	// Then: Plant state should return to Idle
	plant, _ = ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	if plant.AttackAnimState != components.AttackAnimIdle {
		t.Errorf("Expected AttackAnimState to return to Idle when no zombies, got %v", plant.AttackAnimState)
	}
}

// TestUpdatePlantAttackAnimation_FireOnKeyframe tests that bullet fires on keyframe
func TestUpdatePlantAttackAnimation_FireOnKeyframe(t *testing.T) {
	// Given: A peashooter in Attacking state
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	rs := systems.NewReanimSystem(em)
	gs := game.GetGameState()
	bs := createTestBehaviorSystem(em, rm, gs)

	peashooterID := createTestPeashooter(em, rs)

	// Set plant to Attacking state
	plant, _ := ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	plant.AttackAnimState = components.AttackAnimAttacking
	plant.LastFiredFrame = -1 // Allow firing

	// Set animation to keyframe
	reanim, _ := ecs.GetComponent[*components.ReanimComponent](em, peashooterID)
	reanim.CurrentFrame = config.PeashooterShootingFireFrame

	initialBulletCount := countBullets(em)

	// When: updatePlantAttackAnimation is called
	bs.updatePlantAttackAnimation(peashooterID, 0.016)

	// Then: Bullet should be created
	currentBulletCount := countBullets(em)
	if currentBulletCount != initialBulletCount+1 {
		t.Errorf("Expected 1 bullet created at keyframe. Before: %d, After: %d",
			initialBulletCount, currentBulletCount)
	}

	// And: LastFiredFrame should be updated to prevent re-fire
	plant, _ = ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	if plant.LastFiredFrame != config.PeashooterShootingFireFrame {
		t.Errorf("LastFiredFrame should be %d, got %d",
			config.PeashooterShootingFireFrame, plant.LastFiredFrame)
	}
}

// TestUpdatePlantAttackAnimation_NoDoubleFireOnSameFrame tests no double fire on same frame
func TestUpdatePlantAttackAnimation_NoDoubleFireOnSameFrame(t *testing.T) {
	// Given: A peashooter that just fired
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	rs := systems.NewReanimSystem(em)
	gs := game.GetGameState()
	bs := createTestBehaviorSystem(em, rm, gs)

	peashooterID := createTestPeashooter(em, rs)

	// Set plant to Attacking state with LastFiredFrame set to keyframe
	plant, _ := ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	plant.AttackAnimState = components.AttackAnimAttacking
	plant.LastFiredFrame = config.PeashooterShootingFireFrame // Already fired on this frame

	// Set animation to same keyframe
	reanim, _ := ecs.GetComponent[*components.ReanimComponent](em, peashooterID)
	reanim.CurrentFrame = config.PeashooterShootingFireFrame

	initialBulletCount := countBullets(em)

	// When: updatePlantAttackAnimation is called again
	bs.updatePlantAttackAnimation(peashooterID, 0.016)

	// Then: No additional bullet should be created
	currentBulletCount := countBullets(em)
	if currentBulletCount != initialBulletCount {
		t.Errorf("Should not fire again on same frame. Before: %d, After: %d",
			initialBulletCount, currentBulletCount)
	}
}

// TestNonShooterPlantsUnaffected tests that non-shooter plants are not affected
func TestNonShooterPlantsUnaffected(t *testing.T) {
	// Given: Various plant types
	testCases := []struct {
		plantType     components.PlantType
		name          string
		expectShooter bool
	}{
		{components.PlantPeashooter, "Peashooter", true},
		{components.PlantSnowPea, "SnowPea", true},
		{components.PlantSunflower, "Sunflower", false},
		{components.PlantWallnut, "Wallnut", false},
		{components.PlantCherryBomb, "CherryBomb", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// When: IsShooterPlant is called
			result := components.IsShooterPlant(tc.plantType)

			// Then: Result should match expectation
			if result != tc.expectShooter {
				t.Errorf("IsShooterPlant(%s) = %v, expected %v",
					tc.name, result, tc.expectShooter)
			}
		})
	}

	// Given: A sunflower entity (non-shooter plant)
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	gs := game.GetGameState()
	bs := createTestBehaviorSystem(em, rm, gs)

	sunflowerID := em.CreateEntity()
	ecs.AddComponent(em, sunflowerID, &components.PlantComponent{
		PlantType:       components.PlantSunflower,
		AttackAnimState: components.AttackAnimIdle,
	})
	ecs.AddComponent(em, sunflowerID, &components.BehaviorComponent{
		Type: components.BehaviorSunflower,
	})
	ecs.AddComponent(em, sunflowerID, &components.PositionComponent{X: 300, Y: 300})
	ecs.AddComponent(em, sunflowerID, &components.TimerComponent{
		TargetTime:  7.0,
		CurrentTime: 0,
	})

	// Add mock ReanimComponent
	reanimXML := createMockReanimData()
	ecs.AddComponent(em, sunflowerID, &components.ReanimComponent{
		ReanimXML:  reanimXML,
		PartImages: make(map[string]*ebiten.Image),
	})

	// When: sunflower behavior is updated
	bs.handleSunflowerBehavior(sunflowerID, 0.016)

	// Then: AttackAnimState should remain unchanged (sunflower doesn't attack)
	plant, _ := ecs.GetComponent[*components.PlantComponent](em, sunflowerID)
	if plant.AttackAnimState != components.AttackAnimIdle {
		t.Errorf("Sunflower AttackAnimState should not change. Expected Idle, got %v",
			plant.AttackAnimState)
	}

	// Verify: updatePlantAttackAnimation is safe to call on non-shooters
	bs.updatePlantAttackAnimation(sunflowerID, 0.016)

	// Then: State should still be Idle
	plant, _ = ecs.GetComponent[*components.PlantComponent](em, sunflowerID)
	if plant.AttackAnimState != components.AttackAnimIdle {
		t.Error("Non-shooter plants should remain in Idle state")
	}
}

// TestPeashooterAttackAnimationCycle tests the complete attack cycle
func TestPeashooterAttackAnimationCycle(t *testing.T) {
	// Given: A fully configured peashooter and zombie in range
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	rs := systems.NewReanimSystem(em)
	gs := game.GetGameState()
	bs := createTestBehaviorSystem(em, rm, gs)

	peashooterID := createTestPeashooter(em, rs)
	zombieID := createTestZombie(em, 500.0, 300.0)

	// Verify initial state
	plant, _ := ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	if plant.AttackAnimState != components.AttackAnimIdle {
		t.Fatalf("Initial state should be Idle, got %v", plant.AttackAnimState)
	}

	// Phase 1: Trigger attack (zombie in range)
	bs.handlePeashooterBehavior(peashooterID, 0.016, []ecs.EntityID{zombieID})

	// Verify: Plant state should change to Attacking
	plant, _ = ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	if plant.AttackAnimState != components.AttackAnimAttacking {
		t.Errorf("Phase 1: Expected state Attacking, got %v", plant.AttackAnimState)
	}

	// Phase 2: Simulate animation advancing to keyframe to trigger bullet
	reanim, _ := ecs.GetComponent[*components.ReanimComponent](em, peashooterID)
	initialBulletCount := countBullets(em)

	// Advance to keyframe
	reanim.CurrentFrame = config.PeashooterShootingFireFrame
	plant.LastFiredFrame = -1 // Reset to allow firing
	bs.updatePlantAttackAnimation(peashooterID, 0.016)

	// Verify: Bullet should be created at keyframe
	currentBulletCount := countBullets(em)
	if currentBulletCount != initialBulletCount+1 {
		t.Errorf("Phase 2: Expected 1 bullet created at keyframe. Before: %d, After: %d",
			initialBulletCount, currentBulletCount)
	}

	// Phase 3: Return to idle when zombies are removed
	em.DestroyEntity(zombieID)
	em.RemoveMarkedEntities()

	// Call handlePeashooterBehavior with empty zombie list
	bs.handlePeashooterBehavior(peashooterID, 0.016, []ecs.EntityID{})

	// Verify: Plant should return to Idle when no zombies
	plant, _ = ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	if plant.AttackAnimState != components.AttackAnimIdle {
		t.Errorf("Phase 3: Expected state to return to Idle when no zombies, got %v", plant.AttackAnimState)
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// createTestPeashooter creates a test peashooter entity with all required components
// Note: No TimerComponent needed for attack (animation controls attack rhythm)
func createTestPeashooter(em *ecs.EntityManager, rs *systems.ReanimSystem) ecs.EntityID {
	entityID := em.CreateEntity()

	// Add PlantComponent
	ecs.AddComponent(em, entityID, &components.PlantComponent{
		PlantType:       components.PlantPeashooter,
		GridRow:         2,
		GridCol:         3,
		AttackAnimState: components.AttackAnimIdle,
		LastFiredFrame:  -1,
	})

	// Add BehaviorComponent
	ecs.AddComponent(em, entityID, &components.BehaviorComponent{
		Type: components.BehaviorPeashooter,
	})

	// Add PositionComponent
	ecs.AddComponent(em, entityID, &components.PositionComponent{
		X: 400.0,
		Y: 300.0,
	})

	// Add mock ReanimComponent
	reanimXML := createMockReanimData()
	ecs.AddComponent(em, entityID, &components.ReanimComponent{
		ReanimXML:  reanimXML,
		PartImages: make(map[string]*ebiten.Image),
		IsLooping:  true,
		IsFinished: false,
	})

	return entityID
}

// createTestZombie creates a test zombie entity at specified position
func createTestZombie(em *ecs.EntityManager, x, y float64) ecs.EntityID {
	entityID := em.CreateEntity()

	ecs.AddComponent(em, entityID, &components.BehaviorComponent{
		Type: components.BehaviorZombieBasic,
	})

	ecs.AddComponent(em, entityID, &components.PositionComponent{
		X: x,
		Y: y,
	})

	ecs.AddComponent(em, entityID, &components.VelocityComponent{
		VX: -20.0,
	})

	return entityID
}

// countBullets counts the number of bullet entities in the entity manager
func countBullets(em *ecs.EntityManager) int {
	entities := ecs.GetEntitiesWith2[
		*components.BehaviorComponent,
		*components.VelocityComponent,
	](em)

	count := 0
	for _, entityID := range entities {
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](em, entityID)
		if ok && behavior.Type == components.BehaviorPeaProjectile {
			count++
		}
	}
	return count
}

// createMockReanimData creates minimal reanim data for testing
func createMockReanimData() *reanim.ReanimXML {
	return &reanim.ReanimXML{
		FPS: 12,
		Tracks: []reanim.Track{
			{
				Name: "anim_idle",
				Frames: []reanim.Frame{
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(-1)},
				},
			},
			{
				Name: "anim_full_idle",
				Frames: []reanim.Frame{
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(-1)},
				},
			},
			{
				Name: "anim_shooting",
				Frames: []reanim.Frame{
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(-1)},
				},
			},
			{
				Name: "anim_head_idle",
				Frames: []reanim.Frame{
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(-1)},
				},
			},
		},
	}
}

// Note: getTestAudioContext() is already defined in test_helpers.go
// and shared across all test files in this package

// Helper function
func intPtr(i int) *int {
	return &i
}
