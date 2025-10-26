package systems

import (
	"testing"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
)

// ============================================================================
// Story 10.3: Plant Attack Animation System Tests
// ============================================================================
//
// This file contains comprehensive tests for the plant attack animation system,
// covering all acceptance criteria (AC 1-8) from Story 10.3.
//
// Test Coverage:
// - AC 1: Peashooter switches to attack animation on shoot
// - AC 2: Attack animation auto-returns to idle
// - AC 3: No re-trigger during attack animation
// - AC 4: All shooter plants support attack animation
// - AC 7: Doesn't affect bullet firing logic
// - AC 8: Non-shooter plants unaffected
//
// Note: AC 5 (resource verification) is covered by resource loading tests
//       AC 6 (animation smoothness) requires manual gameplay testing

// ============================================================================
// Unit Tests
// ============================================================================

// TestTriggerPlantAttackAnimation tests the triggerPlantAttackAnimation method
// AC 1: Verifies peashooter switches to attack animation when shooting
func TestTriggerPlantAttackAnimation(t *testing.T) {
	// Given: A peashooter plant entity with idle animation state
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	rs := NewReanimSystem(em)
	gs := game.GetGameState()
	bs := NewBehaviorSystem(em, rm, rs, gs)

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

	// When: triggerPlantAttackAnimation is called (simulating a shoot event)
	// Note: We can't call the private method directly, but we can verify the
	// state change happens through handlePeashooterBehavior by setting up
	// the right conditions

	// Create a zombie to trigger attack
	zombieID := createTestZombie(em, 500.0, 300.0)

	// Set timer to ready state
	timer, ok := ecs.GetComponent[*components.TimerComponent](em, peashooterID)
	if ok {
		timer.CurrentTime = timer.TargetTime + 0.1 // Ready to shoot
	}

	// Call handlePeashooterBehavior which should trigger attack animation
	bs.handlePeashooterBehavior(peashooterID, 0.016, []ecs.EntityID{zombieID})

	// Then: Plant state should change to Attacking
	plant, _ = ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	if plant.AttackAnimState != components.AttackAnimAttacking {
		t.Errorf("Expected AttackAnimState to be Attacking, got %v", plant.AttackAnimState)
	}

	// Then: Reanim component should be playing anim_shooting
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](em, peashooterID)
	if !ok {
		t.Fatal("Failed to get ReanimComponent")
	}
	if reanim.IsLooping {
		t.Error("Attack animation should not be looping (PlayAnimationNoLoop)")
	}
}

// TestUpdatePlantAttackAnimation tests the updatePlantAttackAnimation method
// AC 2: Verifies attack animation auto-returns to idle when finished
func TestUpdatePlantAttackAnimation(t *testing.T) {
	// Given: A peashooter in Attacking state with finished animation
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	rs := NewReanimSystem(em)
	gs := game.GetGameState()
	bs := NewBehaviorSystem(em, rm, rs, gs)

	peashooterID := createTestPeashooter(em, rs)

	// Set plant to Attacking state
	plant, _ := ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	plant.AttackAnimState = components.AttackAnimAttacking

	// Simulate animation finished
	reanim, _ := ecs.GetComponent[*components.ReanimComponent](em, peashooterID)
	reanim.IsFinished = true
	reanim.IsLooping = false

	// When: updatePlantAttackAnimation is called
	bs.updatePlantAttackAnimation(peashooterID, 0.016)

	// Then: Plant state should return to Idle
	plant, _ = ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	if plant.AttackAnimState != components.AttackAnimIdle {
		t.Errorf("Expected AttackAnimState to return to Idle, got %v", plant.AttackAnimState)
	}

	// Then: Animation should switch to anim_full_idle (for peashooter)
	// We can't directly verify the animation name without ReanimSystem cooperation,
	// but we can verify IsFinished is cleared and IsLooping is set
	reanim, _ = ecs.GetComponent[*components.ReanimComponent](em, peashooterID)
	if !reanim.IsLooping {
		t.Error("Idle animation should be looping")
	}
	if reanim.IsFinished {
		t.Error("IsFinished should be cleared when switching to idle")
	}
}

// TestUpdatePlantAttackAnimation_OtherPlants tests idle animation selection for non-peashooter plants
// AC 2: Verifies other plants use anim_idle instead of anim_full_idle
func TestUpdatePlantAttackAnimation_OtherPlants(t *testing.T) {
	// Given: A sunflower in Attacking state (hypothetical, for testing logic)
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	rs := NewReanimSystem(em)
	gs := game.GetGameState()
	bs := NewBehaviorSystem(em, rm, rs, gs)

	// Create a generic plant entity (not peashooter)
	plantID := em.CreateEntity()
	ecs.AddComponent(em, plantID, &components.PlantComponent{
		PlantType:       components.PlantSunflower,
		AttackAnimState: components.AttackAnimAttacking,
	})
	ecs.AddComponent(em, plantID, &components.PositionComponent{X: 300, Y: 300})

	// Add mock ReanimComponent
	reanimXML := createMockReanimData()
	ecs.AddComponent(em, plantID, &components.ReanimComponent{
		Reanim:     reanimXML,
		PartImages: make(map[string]*ebiten.Image),
		IsFinished: true,
		IsLooping:  false,
	})

	// When: updatePlantAttackAnimation is called
	bs.updatePlantAttackAnimation(plantID, 0.016)

	// Then: Plant state should return to Idle
	plant, _ := ecs.GetComponent[*components.PlantComponent](em, plantID)
	if plant.AttackAnimState != components.AttackAnimIdle {
		t.Errorf("Expected sunflower to return to Idle, got %v", plant.AttackAnimState)
	}

	// Note: We would verify anim_idle is played for non-peashooter plants,
	// but without mocking ReanimSystem.PlayAnimation, we rely on code inspection
}

// ============================================================================
// Boundary Tests
// ============================================================================

// TestAttackAnimationNoRetrigger tests that attack cannot be retriggered during animation
// AC 3: Verifies no re-trigger during attack animation playback
func TestAttackAnimationNoRetrigger(t *testing.T) {
	// Given: A peashooter in Attacking state with timer ready
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	rs := NewReanimSystem(em)
	gs := game.GetGameState()
	bs := NewBehaviorSystem(em, rm, rs, gs)

	peashooterID := createTestPeashooter(em, rs)

	// Set to Attacking state (simulating ongoing animation)
	plant, _ := ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	plant.AttackAnimState = components.AttackAnimAttacking

	// Set timer to ready (would normally trigger attack)
	timer, _ := ecs.GetComponent[*components.TimerComponent](em, peashooterID)
	timer.CurrentTime = timer.TargetTime + 0.1

	// Create zombie in range
	zombieID := createTestZombie(em, 500.0, 300.0)

	// Count initial bullets
	initialBulletCount := countBullets(em)

	// When: handlePeashooterBehavior is called during animation
	bs.handlePeashooterBehavior(peashooterID, 0.016, []ecs.EntityID{zombieID})

	// Then: No new bullet should be created (attack is blocked)
	currentBulletCount := countBullets(em)
	if currentBulletCount != initialBulletCount {
		t.Errorf("Attack should be blocked during animation. Bullets before: %d, after: %d",
			initialBulletCount, currentBulletCount)
	}

	// Then: Timer should NOT be reset (attack was skipped)
	timer, _ = ecs.GetComponent[*components.TimerComponent](em, peashooterID)
	if timer.CurrentTime < timer.TargetTime {
		t.Error("Timer should not be reset when attack is blocked by animation state")
	}

	// Then: Plant should still be in Attacking state
	plant, _ = ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	if plant.AttackAnimState != components.AttackAnimAttacking {
		t.Error("Plant should remain in Attacking state until animation finishes")
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

// TestPeashooterAttackAnimationCycle tests the complete attack animation cycle
// AC 1, 2, 3, 7: Full integration test covering shoot → animate → return to idle
func TestPeashooterAttackAnimationCycle(t *testing.T) {
	// Given: A fully configured peashooter and zombie in range
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	rs := NewReanimSystem(em)
	gs := game.GetGameState()
	bs := NewBehaviorSystem(em, rm, rs, gs)

	peashooterID := createTestPeashooter(em, rs)
	zombieID := createTestZombie(em, 500.0, 300.0)

	// Verify initial state
	plant, _ := ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	if plant.AttackAnimState != components.AttackAnimIdle {
		t.Fatalf("Initial state should be Idle, got %v", plant.AttackAnimState)
	}

	// When: Phase 1 - Trigger attack (timer ready + zombie in range)
	timer, _ := ecs.GetComponent[*components.TimerComponent](em, peashooterID)
	timer.CurrentTime = timer.TargetTime + 0.1

	initialBulletCount := countBullets(em)
	bs.handlePeashooterBehavior(peashooterID, 0.016, []ecs.EntityID{zombieID})

	// Then: Phase 1 verification
	// 1. Plant state should change to Attacking
	plant, _ = ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	if plant.AttackAnimState != components.AttackAnimAttacking {
		t.Errorf("Phase 1: Expected state Attacking, got %v", plant.AttackAnimState)
	}

	// 2. Bullet should be created (AC 7: doesn't affect bullet firing logic)
	currentBulletCount := countBullets(em)
	if currentBulletCount != initialBulletCount+1 {
		t.Errorf("Phase 1: Expected 1 bullet created. Before: %d, After: %d",
			initialBulletCount, currentBulletCount)
	}

	// 3. Timer should be reset
	timer, _ = ecs.GetComponent[*components.TimerComponent](em, peashooterID)
	if timer.CurrentTime >= timer.TargetTime {
		t.Error("Phase 1: Timer should be reset after shooting")
	}

	// When: Phase 2 - Animation in progress, try to trigger again
	timer.CurrentTime = timer.TargetTime + 0.1 // Reset timer to ready
	beforeRetriggerBullets := countBullets(em)
	bs.handlePeashooterBehavior(peashooterID, 0.016, []ecs.EntityID{zombieID})

	// Then: Phase 2 verification (AC 3: no re-trigger during animation)
	afterRetriggerBullets := countBullets(em)
	if afterRetriggerBullets != beforeRetriggerBullets {
		t.Error("Phase 2: Attack should be blocked during animation (no new bullets)")
	}

	// When: Phase 3 - Animation finishes
	reanim, _ := ecs.GetComponent[*components.ReanimComponent](em, peashooterID)
	reanim.IsFinished = true
	bs.updatePlantAttackAnimation(peashooterID, 0.016)

	// Then: Phase 3 verification (AC 2: auto-return to idle)
	plant, _ = ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	if plant.AttackAnimState != components.AttackAnimIdle {
		t.Errorf("Phase 3: Expected state to return to Idle, got %v", plant.AttackAnimState)
	}

	reanim, _ = ecs.GetComponent[*components.ReanimComponent](em, peashooterID)
	if !reanim.IsLooping {
		t.Error("Phase 3: Idle animation should be looping")
	}
	if reanim.IsFinished {
		t.Error("Phase 3: IsFinished should be cleared")
	}

	// When: Phase 4 - Can shoot again after returning to idle
	timer, _ = ecs.GetComponent[*components.TimerComponent](em, peashooterID)
	timer.CurrentTime = timer.TargetTime + 0.1
	beforePhase4Bullets := countBullets(em)
	bs.handlePeashooterBehavior(peashooterID, 0.016, []ecs.EntityID{zombieID})

	// Then: Phase 4 verification
	afterPhase4Bullets := countBullets(em)
	if afterPhase4Bullets != beforePhase4Bullets+1 {
		t.Error("Phase 4: Should be able to shoot again after animation completes")
	}

	plant, _ = ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	if plant.AttackAnimState != components.AttackAnimAttacking {
		t.Error("Phase 4: Should enter Attacking state again")
	}
}

// ============================================================================
// Regression Tests
// ============================================================================

// TestNonShooterPlantsUnaffected tests that non-shooter plants are not affected
// AC 8: Verifies sunflower and other non-shooter plants behave normally
func TestNonShooterPlantsUnaffected(t *testing.T) {
	// Given: Various plant types
	testCases := []struct {
		plantType     components.PlantType
		name          string
		expectShooter bool
	}{
		{components.PlantPeashooter, "Peashooter", true},
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
	rs := NewReanimSystem(em)
	gs := game.GetGameState()
	bs := NewBehaviorSystem(em, rm, rs, gs)

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
		Reanim:     reanimXML,
		PartImages: make(map[string]*ebiten.Image),
	})

	// When: sunflower behavior is updated
	initialState := components.AttackAnimIdle
	bs.handleSunflowerBehavior(sunflowerID, 0.016)

	// Then: AttackAnimState should remain unchanged
	plant, _ := ecs.GetComponent[*components.PlantComponent](em, sunflowerID)
	if plant.AttackAnimState != initialState {
		t.Errorf("Sunflower AttackAnimState should not change. Expected %v, got %v",
			initialState, plant.AttackAnimState)
	}

	// When: updatePlantAttackAnimation is called (should be safe to call on non-shooters)
	plant.AttackAnimState = components.AttackAnimAttacking // Artificially set (shouldn't happen in real game)
	reanim, _ := ecs.GetComponent[*components.ReanimComponent](em, sunflowerID)
	reanim.IsFinished = true

	bs.updatePlantAttackAnimation(sunflowerID, 0.016)

	// Then: System should handle gracefully (return to idle)
	plant, _ = ecs.GetComponent[*components.PlantComponent](em, sunflowerID)
	if plant.AttackAnimState != components.AttackAnimIdle {
		t.Error("Even if non-shooter enters Attacking state (bug), system should recover gracefully")
	}
}

// TestIsShooterPlant_FuturePlants tests extensibility for future plant types
// AC 4: Verifies the IsShooterPlant mechanism supports future additions
func TestIsShooterPlant_FuturePlants(t *testing.T) {
	// This test documents the expected behavior for future plant types
	// When new shooter plants are added (snowpea, repeater, etc.),
	// they should be added to the shooterPlants map in plant.go

	// Current implementation check
	if !components.IsShooterPlant(components.PlantPeashooter) {
		t.Error("Peashooter should be identified as shooter plant")
	}

	// Future plants (commented out until implemented):
	// - PlantSnowPea: true
	// - PlantRepeater: true
	// - PlantThreepeater: true
	// - PlantCabbagePult: true
	// - PlantKernelPult: true

	// When adding new shooter plants:
	// 1. Add to shooterPlants map in pkg/components/plant.go
	// 2. Ensure attack animation logic works (no code changes needed in behavior_system.go)
	// 3. Add specific animation name if different from "anim_shooting"
}

// ============================================================================
// Helper Functions
// ============================================================================

// createTestPeashooter creates a test peashooter entity with all required components
func createTestPeashooter(em *ecs.EntityManager, rs *ReanimSystem) ecs.EntityID {
	entityID := em.CreateEntity()

	// Add PlantComponent
	ecs.AddComponent(em, entityID, &components.PlantComponent{
		PlantType:       components.PlantPeashooter,
		GridRow:         2,
		GridCol:         3,
		AttackAnimState: components.AttackAnimIdle,
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

	// Add TimerComponent
	ecs.AddComponent(em, entityID, &components.TimerComponent{
		Name:        "attack_cooldown",
		TargetTime:  1.4,
		CurrentTime: 0,
	})

	// Add mock ReanimComponent
	reanimXML := createMockReanimData()
	ecs.AddComponent(em, entityID, &components.ReanimComponent{
		Reanim:     reanimXML,
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
		},
	}
}

// Note: getTestAudioContext() is already defined in test_helpers.go
// and shared across all test files in this package
