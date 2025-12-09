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
// Note: Attack animation is now looping (IsLooping=true) until no zombies are present
func TestTriggerPlantAttackAnimation(t *testing.T) {
	// Given: A peashooter plant entity with idle animation state
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	rs := systems.NewReanimSystem(em)

	// Story 13.6: 设置配置管理器以支持配置驱动的动画播放
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

	// Then: Reanim component should be playing (attack animation is now looping)
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](em, peashooterID)
	if !ok {
		t.Fatal("Failed to get ReanimComponent")
	}
	// Note: Attack animation is now looping until no zombies are detected
	if !reanim.IsLooping {
		t.Error("Attack animation should be looping (new behavior)")
	}
}

// TestUpdatePlantAttackAnimation tests return to idle via handlePeashooterBehavior
// AC 2: Verifies attack animation returns to idle when no zombies are present
// Note: New behavior - attack animation is looping, and returns to idle via handlePeashooterBehavior
// when no zombies are detected, not via updatePlantAttackAnimation
func TestUpdatePlantAttackAnimation(t *testing.T) {
	// Given: A peashooter in Attacking state
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	rs := systems.NewReanimSystem(em)

	// Story 13.6: 设置配置管理器以支持配置驱动的动画播放
	configManager, err := config.NewReanimConfigManager("data/reanim_config.yaml")
	if err != nil {
		t.Skipf("跳过测试：无法加载配置文件: %v", err)
	}
	rs.SetConfigManager(configManager)

	gs := game.GetGameState()
	bs := createTestBehaviorSystem(em, rm, gs)

	peashooterID := createTestPeashooter(em, rs)

	// Set plant to Attacking state with looping animation (new behavior)
	plant, _ := ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	plant.AttackAnimState = components.AttackAnimAttacking

	// Attack animation is now looping
	reanim, _ := ecs.GetComponent[*components.ReanimComponent](em, peashooterID)
	reanim.IsLooping = true
	reanim.IsFinished = false

	// When: handlePeashooterBehavior is called with NO zombies
	// This should trigger return to idle state
	bs.handlePeashooterBehavior(peashooterID, 0.016, []ecs.EntityID{}) // Empty zombie list

	// Then: Plant state should return to Idle (because no zombies)
	plant, _ = ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	if plant.AttackAnimState != components.AttackAnimIdle {
		t.Errorf("Expected AttackAnimState to return to Idle when no zombies, got %v", plant.AttackAnimState)
	}
}

// TestUpdatePlantAttackAnimation_OtherPlants tests idle state transition for non-peashooter plants
// Note: This test verifies that non-shooter plants (like sunflower) don't have attack animation
// states in the first place - they stay in Idle
func TestUpdatePlantAttackAnimation_OtherPlants(t *testing.T) {
	// Given: A sunflower plant (non-shooter)
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	rs := systems.NewReanimSystem(em)

	// Story 13.6: 设置配置管理器以支持配置驱动的动画播放
	configManager, err := config.NewReanimConfigManager("data/reanim_config.yaml")
	if err != nil {
		t.Skipf("跳过测试：无法加载配置文件: %v", err)
	}
	rs.SetConfigManager(configManager)

	gs := game.GetGameState()
	bs := createTestBehaviorSystem(em, rm, gs)

	// Create a sunflower plant entity (non-shooter - should stay in Idle)
	plantID := em.CreateEntity()
	ecs.AddComponent(em, plantID, &components.PlantComponent{
		PlantType:       components.PlantSunflower,
		AttackAnimState: components.AttackAnimIdle, // Sunflower starts and stays in Idle
	})
	ecs.AddComponent(em, plantID, &components.PositionComponent{X: 300, Y: 300})

	// Add mock ReanimComponent
	reanimXML := createMockReanimData()
	ecs.AddComponent(em, plantID, &components.ReanimComponent{
		ReanimXML:  reanimXML,
		PartImages: make(map[string]*ebiten.Image),
		IsFinished: false,
		IsLooping:  true,
	})

	// When: updatePlantAttackAnimation is called
	bs.updatePlantAttackAnimation(plantID, 0.016)

	// Then: Plant state should remain Idle (sunflower doesn't attack)
	plant, _ := ecs.GetComponent[*components.PlantComponent](em, plantID)
	if plant.AttackAnimState != components.AttackAnimIdle {
		t.Errorf("Expected sunflower to stay in Idle, got %v", plant.AttackAnimState)
	}
}

// ============================================================================
// Boundary Tests
// ============================================================================

// TestAttackAnimationNoRetrigger tests that attack continues properly during animation
// AC 3: Verifies attack continues with proper frame-based firing during animation
// Note: New behavior - attack animation is looping, timer can trigger PendingProjectile
// but bullets are only fired on specific keyframes
func TestAttackAnimationNoRetrigger(t *testing.T) {
	// Given: A peashooter in Attacking state with timer ready
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	rs := systems.NewReanimSystem(em)
	gs := game.GetGameState()
	bs := createTestBehaviorSystem(em, rm, gs)

	peashooterID := createTestPeashooter(em, rs)

	// Set to Attacking state (simulating ongoing animation)
	plant, _ := ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	plant.AttackAnimState = components.AttackAnimAttacking

	// Create zombie in range
	zombieID := createTestZombie(em, 500.0, 300.0)

	// When: handlePeashooterBehavior is called during animation
	bs.handlePeashooterBehavior(peashooterID, 0.016, []ecs.EntityID{zombieID})

	// Then: Plant should still be in Attacking state (because zombies are present)
	plant, _ = ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	if plant.AttackAnimState != components.AttackAnimAttacking {
		t.Error("Plant should remain in Attacking state while zombies are present")
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

// TestPeashooterAttackAnimationCycle tests the attack → bullet → return to idle cycle
// Note: New behavior - attack animation is looping, bullets fire on keyframes,
// and return to idle happens when no zombies are detected
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

	// Phase 1: Trigger attack (timer ready + zombie in range)
	timer, _ := ecs.GetComponent[*components.TimerComponent](em, peashooterID)
	timer.CurrentTime = timer.TargetTime + 0.1

	initialBulletCount := countBullets(em)
	bs.handlePeashooterBehavior(peashooterID, 0.016, []ecs.EntityID{zombieID})

	// Verify: Plant state should change to Attacking
	plant, _ = ecs.GetComponent[*components.PlantComponent](em, peashooterID)
	if plant.AttackAnimState != components.AttackAnimAttacking {
		t.Errorf("Phase 1: Expected state Attacking, got %v", plant.AttackAnimState)
	}

	// Phase 2: Simulate animation advancing to keyframe 10 to trigger bullet creation
	reanim, _ := ecs.GetComponent[*components.ReanimComponent](em, peashooterID)
	for i := 0; i <= 10; i++ {
		reanim.CurrentFrame = i
		// Reset LastFiredFrame to allow firing (simulating animation loop)
		if i == 0 {
			plant.LastFiredFrame = -1
		}
		bs.updatePlantAttackAnimation(peashooterID, 0.016)
	}

	// Verify: Bullet should be created at keyframe 10
	currentBulletCount := countBullets(em)
	if currentBulletCount != initialBulletCount+1 {
		t.Errorf("Phase 2: Expected 1 bullet created at keyframe 10. Before: %d, After: %d",
			initialBulletCount, currentBulletCount)
	}

	// Phase 3: Return to idle when zombies are removed
	// Remove the zombie by destroying it
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
	initialState := components.AttackAnimIdle
	bs.handleSunflowerBehavior(sunflowerID, 0.016)

	// Then: AttackAnimState should remain unchanged (sunflower doesn't attack)
	plant, _ := ecs.GetComponent[*components.PlantComponent](em, sunflowerID)
	if plant.AttackAnimState != initialState {
		t.Errorf("Sunflower AttackAnimState should not change. Expected %v, got %v",
			initialState, plant.AttackAnimState)
	}

	// Verify: updatePlantAttackAnimation is safe to call on non-shooters
	// Note: For non-shooters, it simply returns early since AttackAnimState is Idle
	bs.updatePlantAttackAnimation(sunflowerID, 0.016)

	// Then: State should still be Idle
	plant, _ = ecs.GetComponent[*components.PlantComponent](em, sunflowerID)
	if plant.AttackAnimState != components.AttackAnimIdle {
		t.Error("Non-shooter plants should remain in Idle state")
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
func createTestPeashooter(em *ecs.EntityManager, rs *systems.ReanimSystem) ecs.EntityID {
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
			// Story 6.9: Add anim_head_idle for multi-animation overlay support
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
