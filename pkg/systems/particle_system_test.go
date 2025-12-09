package systems

import (
	"math"
	"reflect"
	"testing"

	"github.com/gonewx/pvz/internal/particle"
	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
)

// TestParticleSystem_ParticleLifecycle tests that particles age and are destroyed when expired
func TestParticleSystem_ParticleLifecycle(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create test particle with 1 second lifetime
	particleID := em.CreateEntity()
	particleComp := &components.ParticleComponent{
		Age:      0,
		Lifetime: 1.0, // 1 second
	}
	posComp := &components.PositionComponent{X: 100, Y: 100}
	em.AddComponent(particleID, particleComp)
	em.AddComponent(particleID, posComp)

	// Update for 0.5 seconds (particle should still be alive)
	ps.Update(0.5)
	if !em.HasComponent(particleID, reflect.TypeOf(&components.ParticleComponent{})) {
		t.Error("Particle should still exist after 0.5 seconds")
	}
	if particleComp.Age != 0.5 {
		t.Errorf("Particle age should be 0.5, got %v", particleComp.Age)
	}

	// Update for another 0.6 seconds (total 1.1s, should be destroyed)
	ps.Update(0.6)
	em.RemoveMarkedEntities()
	if em.HasComponent(particleID, reflect.TypeOf(&components.ParticleComponent{})) {
		t.Error("Particle should be destroyed after exceeding lifetime")
	}
}

// TestParticleSystem_EmitterSpawnsParticles tests that emitters spawn particles at the correct rate
func TestParticleSystem_EmitterSpawnsParticles(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create test emitter with SpawnRate = 10 particles/second
	emitterID := em.CreateEntity()
	emitterComp := &components.EmitterComponent{
		Config: &particle.EmitterConfig{
			ParticleDuration: "1000", // 1 second
			LaunchSpeed:      "100",
			LaunchAngle:      "0",
		},
		Active:           true,
		Age:              0,
		SystemDuration:   0, // infinite
		NextSpawnTime:    0,
		ActiveParticles:  make([]ecs.EntityID, 0),
		TotalLaunched:    0,
		SpawnRate:        10, // 10 particles per second
		SpawnMaxActive:   100,
		SpawnMaxLaunched: 0, // unlimited
	}
	posComp := &components.PositionComponent{X: 200, Y: 200}
	em.AddComponent(emitterID, emitterComp)
	em.AddComponent(emitterID, posComp)

	// Update for 0.5 seconds
	ps.Update(0.5)

	// Should have spawned ~5 particles (10 particles/sec * 0.5sec)
	// Allow tolerance due to timing
	if emitterComp.TotalLaunched < 4 || emitterComp.TotalLaunched > 6 {
		t.Errorf("Expected ~5 particles, got %d", emitterComp.TotalLaunched)
	}
}

// TestParticleSystem_EmitterMaxActive tests that emitter respects max active particle limit
func TestParticleSystem_EmitterMaxActive(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create emitter with low MaxActive limit
	emitterID := em.CreateEntity()
	emitterComp := &components.EmitterComponent{
		Config: &particle.EmitterConfig{
			ParticleDuration: "10000", // 10 seconds (long lifetime so particles don't die)
			LaunchSpeed:      "100",
			LaunchAngle:      "0",
		},
		Active:           true,
		Age:              0,
		SystemDuration:   0,
		NextSpawnTime:    0,
		ActiveParticles:  make([]ecs.EntityID, 0),
		TotalLaunched:    0,
		SpawnRate:        100,  // Very high rate
		SpawnMaxActive:   5,    // Only allow 5 active particles
		SpawnMaxLaunched: 1000, // High launch limit
	}
	posComp := &components.PositionComponent{X: 200, Y: 200}
	em.AddComponent(emitterID, emitterComp)
	em.AddComponent(emitterID, posComp)

	// Update for 1 second (would normally spawn ~100 particles)
	ps.Update(1.0)

	// Should only have 5 active particles due to limit
	if len(emitterComp.ActiveParticles) > 5 {
		t.Errorf("Expected max 5 active particles, got %d", len(emitterComp.ActiveParticles))
	}
}

// TestParticleSystem_EmitterMaxLaunched tests that emitter respects max launched limit
func TestParticleSystem_EmitterMaxLaunched(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create emitter with low MaxLaunched limit
	emitterID := em.CreateEntity()
	emitterComp := &components.EmitterComponent{
		Config: &particle.EmitterConfig{
			ParticleDuration: "100", // 0.1 second (short lifetime so particles die quickly)
			LaunchSpeed:      "100",
			LaunchAngle:      "0",
		},
		Active:           true,
		Age:              0,
		SystemDuration:   0,
		NextSpawnTime:    0,
		ActiveParticles:  make([]ecs.EntityID, 0),
		TotalLaunched:    0,
		SpawnRate:        50,   // High rate
		SpawnMaxActive:   1000, // High active limit
		SpawnMaxLaunched: 10,   // Only allow 10 total particles
	}
	posComp := &components.PositionComponent{X: 200, Y: 200}
	em.AddComponent(emitterID, emitterComp)
	em.AddComponent(emitterID, posComp)

	// Update for 1 second
	ps.Update(1.0)

	// Should have launched exactly 10 particles
	if emitterComp.TotalLaunched != 10 {
		t.Errorf("Expected 10 total launched particles, got %d", emitterComp.TotalLaunched)
	}
}

// TestParticleSystem_EmitterSystemDuration tests that emitter stops after system duration
func TestParticleSystem_EmitterSystemDuration(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create emitter with 1 second duration
	emitterID := em.CreateEntity()
	emitterComp := &components.EmitterComponent{
		Config: &particle.EmitterConfig{
			ParticleDuration: "1000",
			LaunchSpeed:      "100",
			LaunchAngle:      "0",
		},
		Active:           true,
		Age:              0,
		SystemDuration:   1.0, // Stop after 1 second
		NextSpawnTime:    0,
		ActiveParticles:  make([]ecs.EntityID, 0),
		TotalLaunched:    0,
		SpawnRate:        10,
		SpawnMaxActive:   100,
		SpawnMaxLaunched: 0,
	}
	posComp := &components.PositionComponent{X: 200, Y: 200}
	em.AddComponent(emitterID, emitterComp)
	em.AddComponent(emitterID, posComp)

	// Update for 0.5 seconds (should be active)
	ps.Update(0.5)
	if !emitterComp.Active {
		t.Error("Emitter should still be active before duration expires")
	}

	// Update for another 0.6 seconds (total 1.1s, should be inactive)
	ps.Update(0.6)
	if emitterComp.Active {
		t.Error("Emitter should be inactive after duration expires")
	}
}

// TestParticleSystem_ParticleVelocity tests that particles move according to velocity
func TestParticleSystem_ParticleVelocity(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create particle with velocity
	particleID := em.CreateEntity()
	particleComp := &components.ParticleComponent{
		VelocityX: 100, // 100 pixels/second to the right
		VelocityY: 50,  // 50 pixels/second downward
		Age:       0,
		Lifetime:  10,
	}
	posComp := &components.PositionComponent{X: 0, Y: 0}
	em.AddComponent(particleID, particleComp)
	em.AddComponent(particleID, posComp)

	// Update for 1 second
	ps.Update(1.0)

	// Position should have changed by velocity * dt
	if posComp.X != 100 {
		t.Errorf("Particle X should be 100, got %v", posComp.X)
	}
	if posComp.Y != 50 {
		t.Errorf("Particle Y should be 50, got %v", posComp.Y)
	}
}

// TestParticleSystem_ParticleRotation tests that particles rotate
func TestParticleSystem_ParticleRotation(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create particle with rotation speed
	particleID := em.CreateEntity()
	particleComp := &components.ParticleComponent{
		Rotation:      0,
		RotationSpeed: 180, // 180 degrees/second
		Age:           0,
		Lifetime:      10,
	}
	posComp := &components.PositionComponent{X: 0, Y: 0}
	em.AddComponent(particleID, particleComp)
	em.AddComponent(particleID, posComp)

	// Update for 1 second
	ps.Update(1.0)

	// Rotation should have increased by RotationSpeed * dt
	if particleComp.Rotation != 180 {
		t.Errorf("Particle rotation should be 180, got %v", particleComp.Rotation)
	}
}

// TestParticleSystem_AlphaInterpolation tests alpha keyframe animation
func TestParticleSystem_AlphaInterpolation(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create particle with alpha keyframes (fade in/out)
	particleID := em.CreateEntity()
	particleComp := &components.ParticleComponent{
		Alpha: 0,
		AlphaKeyframes: []particle.Keyframe{
			{Time: 0, Value: 0},   // Start transparent
			{Time: 0.5, Value: 1}, // Fade to opaque at midpoint
			{Time: 1, Value: 0},   // Fade back to transparent
		},
		AlphaInterpolation: "Linear",
		Age:                0,
		Lifetime:           2, // 2 seconds
	}
	posComp := &components.PositionComponent{X: 0, Y: 0}
	em.AddComponent(particleID, particleComp)
	em.AddComponent(particleID, posComp)

	// Update to 0.5 seconds (t=0.25, should be alpha ~0.5)
	ps.Update(0.5)
	if particleComp.Alpha < 0.4 || particleComp.Alpha > 0.6 {
		t.Errorf("Alpha at t=0.25 should be ~0.5, got %v", particleComp.Alpha)
	}

	// Update to 1.0 seconds (t=0.5, should be alpha=1)
	ps.Update(0.5)
	if particleComp.Alpha < 0.9 || particleComp.Alpha > 1.0 {
		t.Errorf("Alpha at t=0.5 should be ~1.0, got %v", particleComp.Alpha)
	}
}

// TestParticleSystem_ScaleInterpolation tests scale keyframe animation
func TestParticleSystem_ScaleInterpolation(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create particle with scale keyframes (grow and shrink)
	particleID := em.CreateEntity()
	particleComp := &components.ParticleComponent{
		Scale: 0,
		ScaleKeyframes: []particle.Keyframe{
			{Time: 0, Value: 0},   // Start tiny
			{Time: 0.5, Value: 2}, // Grow to 2x at midpoint
			{Time: 1, Value: 0},   // Shrink back to 0
		},
		ScaleInterpolation: "Linear",
		Age:                0,
		Lifetime:           1, // 1 second
	}
	posComp := &components.PositionComponent{X: 0, Y: 0}
	em.AddComponent(particleID, particleComp)
	em.AddComponent(particleID, posComp)

	// Update to 0.25 seconds (t=0.25, should be scale ~1.0)
	ps.Update(0.25)
	if particleComp.Scale < 0.9 || particleComp.Scale > 1.1 {
		t.Errorf("Scale at t=0.25 should be ~1.0, got %v", particleComp.Scale)
	}
}

// TestParticleSystem_SpinInterpolation tests spin keyframe animation
func TestParticleSystem_SpinInterpolation(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create particle with spin keyframes
	particleID := em.CreateEntity()
	particleComp := &components.ParticleComponent{
		RotationSpeed: 0,
		SpinKeyframes: []particle.Keyframe{
			{Time: 0, Value: 0},   // Start stationary
			{Time: 1, Value: 360}, // Accelerate to 360 deg/sec
		},
		SpinInterpolation: "Linear",
		Age:               0,
		Lifetime:          2, // 2 seconds
	}
	posComp := &components.PositionComponent{X: 0, Y: 0}
	em.AddComponent(particleID, particleComp)
	em.AddComponent(particleID, posComp)

	// Update to 1 second (t=0.5, should be rotationSpeed ~180)
	ps.Update(1.0)
	if particleComp.RotationSpeed < 170 || particleComp.RotationSpeed > 190 {
		t.Errorf("RotationSpeed at t=0.5 should be ~180, got %v", particleComp.RotationSpeed)
	}
}

// TestParticleSystem_AccelerationField tests acceleration force field
func TestParticleSystem_AccelerationField(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create particle with downward acceleration (gravity)
	// PopCap Mixed Unit System:
	// - Acceleration config value is "velocity increment per 0.01s"
	// - Engine converts to pixels/second²: value / 0.01
	particleID := em.CreateEntity()
	particleComp := &components.ParticleComponent{
		VelocityX: 0,
		VelocityY: 0,
		Age:       0,
		Lifetime:  10,
		Fields: []particle.Field{
			{
				FieldType: "Acceleration",
				X:         "0",
				Y:         "1", // velocity increases by 1 px/s every 0.01s → 100 px/s²
			},
		},
	}
	posComp := &components.PositionComponent{X: 0, Y: 0}
	em.AddComponent(particleID, particleComp)
	em.AddComponent(particleID, posComp)

	initialVelocityY := particleComp.VelocityY

	// Update for 1 second
	ps.Update(1.0)

	// Velocity should have increased by acceleration * dt
	// Config value 1 → converts to 100 px/s² → velocity increases by 100
	expectedVelocityY := initialVelocityY + 100*1.0
	if particleComp.VelocityY < expectedVelocityY-1 || particleComp.VelocityY > expectedVelocityY+1 {
		t.Errorf("VelocityY should be ~%v, got %v", expectedVelocityY, particleComp.VelocityY)
	}
}

// TestParticleSystem_FrictionField tests friction force field
func TestParticleSystem_FrictionField(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create particle with friction
	// PopCap Unit System: Friction values are per-tick (0.01 second) values
	// To achieve 50% friction per second, use 0.005 (will be converted to 0.5 per second)
	// Config value 0.005 → 0.005/0.01 = 0.5 (50% friction per second)
	particleID := em.CreateEntity()
	particleComp := &components.ParticleComponent{
		VelocityX: 100, // Initial velocity
		VelocityY: 0,
		Age:       0,
		Lifetime:  10,
		Fields: []particle.Field{
			{
				FieldType: "Friction",
				X:         "0.005", // 50% friction per second after unit conversion
				Y:         "0",
			},
		},
	}
	posComp := &components.PositionComponent{X: 0, Y: 0}
	em.AddComponent(particleID, particleComp)
	em.AddComponent(particleID, posComp)

	initialVelocityX := particleComp.VelocityX

	// Update for 1 second
	ps.Update(1.0)

	// Velocity should have decreased by friction
	// Config value 0.005 → converted to 0.5 per second
	// After 1 second: velocity = initialVelocity * (1 - 0.5 * 1.0) = 50
	expectedVelocityX := initialVelocityX * (1 - 0.5*1.0)
	if particleComp.VelocityX < expectedVelocityX-1 || particleComp.VelocityX > expectedVelocityX+1 {
		t.Errorf("VelocityX should be ~%v, got %v", expectedVelocityX, particleComp.VelocityX)
	}
}

// TestParticleSystem_EdgeCases tests edge cases and error handling
func TestParticleSystem_EdgeCases(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	t.Run("Particle with zero lifetime", func(t *testing.T) {
		particleID := em.CreateEntity()
		particleComp := &components.ParticleComponent{
			Age:      0,
			Lifetime: 0, // Zero lifetime
		}
		posComp := &components.PositionComponent{X: 0, Y: 0}
		em.AddComponent(particleID, particleComp)
		em.AddComponent(particleID, posComp)

		// Should be destroyed immediately
		ps.Update(0.01)
		em.RemoveMarkedEntities()
		if em.HasComponent(particleID, reflect.TypeOf(&components.ParticleComponent{})) {
			t.Error("Particle with zero lifetime should be destroyed immediately")
		}
	})

	t.Run("Emitter with nil config", func(t *testing.T) {
		emitterID := em.CreateEntity()
		emitterComp := &components.EmitterComponent{
			Config:          nil, // Nil config
			Active:          true,
			Age:             0,
			NextSpawnTime:   0,
			ActiveParticles: make([]ecs.EntityID, 0),
			SpawnRate:       10,
		}
		posComp := &components.PositionComponent{X: 0, Y: 0}
		em.AddComponent(emitterID, emitterComp)
		em.AddComponent(emitterID, posComp)

		// Should not crash
		ps.Update(0.1)
		if emitterComp.TotalLaunched > 0 {
			t.Error("Emitter with nil config should not spawn particles")
		}
	})

	t.Run("Particle cleanup from emitter active list", func(t *testing.T) {
		emitterID := em.CreateEntity()
		emitterComp := &components.EmitterComponent{
			Config: &particle.EmitterConfig{
				ParticleDuration: "100", // 0.1 second
				LaunchSpeed:      "100",
				LaunchAngle:      "0",
			},
			Active:           true,
			Age:              0,
			SystemDuration:   0,
			NextSpawnTime:    0,
			ActiveParticles:  make([]ecs.EntityID, 0),
			TotalLaunched:    0,
			SpawnRate:        10,
			SpawnMaxActive:   100,
			SpawnMaxLaunched: 0,
		}
		posComp := &components.PositionComponent{X: 0, Y: 0}
		em.AddComponent(emitterID, emitterComp)
		em.AddComponent(emitterID, posComp)

		// Spawn some particles
		ps.Update(0.5)
		initialActive := len(emitterComp.ActiveParticles)

		// Wait for particles to expire
		ps.Update(1.0)
		em.RemoveMarkedEntities()

		// Update emitter again to clean up dead particles
		ps.Update(0.01)

		// Active particle list should be empty (all particles expired)
		if len(emitterComp.ActiveParticles) >= initialActive {
			t.Errorf("Active particle list should be cleaned up, had %d, now %d", initialActive, len(emitterComp.ActiveParticles))
		}
	})
}

// TestParticleSystem_NoMemoryLeak tests that emitters and particles are properly cleaned up
// Story 7.4: Verify no memory leak when triggering 100 particle effects
func TestParticleSystem_NoMemoryLeak(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create 100 emitters with short-lived particles
	createdEmitters := make([]ecs.EntityID, 0, 100)
	for i := 0; i < 100; i++ {
		emitterID := em.CreateEntity()
		createdEmitters = append(createdEmitters, emitterID)
		emitterComp := &components.EmitterComponent{
			Config: &particle.EmitterConfig{
				ParticleDuration: "100", // 0.1 second (short lifetime)
				LaunchSpeed:      "100",
				LaunchAngle:      "0",
			},
			Active:           true,
			Age:              0,
			SystemDuration:   0.2, // Stop after 0.2 seconds
			NextSpawnTime:    0,
			ActiveParticles:  make([]ecs.EntityID, 0),
			TotalLaunched:    0,
			SpawnRate:        10,  // 10 particles/second
			SpawnMaxActive:   100, // Allow many active
			SpawnMaxLaunched: 5,   // Only spawn 5 particles total per emitter
		}
		posComp := &components.PositionComponent{X: 100, Y: 100}
		em.AddComponent(emitterID, emitterComp)
		em.AddComponent(emitterID, posComp)
	}

	// Verify all emitters were created
	emitterType := reflect.TypeOf(&components.EmitterComponent{})
	initialEmitterCount := len(em.GetEntitiesWith(emitterType))
	if initialEmitterCount != 100 {
		t.Errorf("Expected 100 emitters after creation, got %d", initialEmitterCount)
	}

	// Update for 0.5 seconds (spawn particles and let emitters finish)
	for i := 0; i < 50; i++ {
		ps.Update(0.01)
		em.RemoveMarkedEntities() // Clean up marked entities
	}

	// Update for another 1 second to let all particles expire
	for i := 0; i < 100; i++ {
		ps.Update(0.01)
		em.RemoveMarkedEntities() // Clean up marked entities
	}

	// Verify all emitters and particles have been cleaned up
	// Check for remaining particle entities
	particleEntities := em.GetEntitiesWith(reflect.TypeOf(&components.ParticleComponent{}))
	if len(particleEntities) > 0 {
		t.Errorf("Memory leak detected: found %d remaining particle entities (should be 0)", len(particleEntities))
	}

	// Check for remaining emitter entities
	emitterEntities := em.GetEntitiesWith(reflect.TypeOf(&components.EmitterComponent{}))
	if len(emitterEntities) > 0 {
		t.Errorf("Memory leak detected: found %d remaining emitter entities (should be 0)", len(emitterEntities))
	}

	// Additional verification: check that all created emitters no longer have EmitterComponent
	remainingCount := 0
	for _, emitterID := range createdEmitters {
		if em.HasComponent(emitterID, emitterType) {
			remainingCount++
		}
	}
	if remainingCount > 0 {
		t.Errorf("Memory leak detected: %d out of 100 emitters were not cleaned up", remainingCount)
	}
}

// TestParticleSystem_EmitterAutoCleanup tests that emitters are automatically deleted when finished
// Story 7.4: Verify emitter cleanup when inactive and all particles gone
func TestParticleSystem_EmitterAutoCleanup(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create emitter that will become inactive
	emitterID := em.CreateEntity()
	emitterComp := &components.EmitterComponent{
		Config: &particle.EmitterConfig{
			ParticleDuration: "100", // 0.1 second
			LaunchSpeed:      "100",
			LaunchAngle:      "0",
		},
		Active:           true,
		Age:              0,
		SystemDuration:   0.2, // Stop after 0.2 seconds
		NextSpawnTime:    0,
		ActiveParticles:  make([]ecs.EntityID, 0),
		TotalLaunched:    0,
		SpawnRate:        10,
		SpawnMaxActive:   100,
		SpawnMaxLaunched: 3, // Only spawn 3 particles
	}
	posComp := &components.PositionComponent{X: 100, Y: 100}
	em.AddComponent(emitterID, emitterComp)
	em.AddComponent(emitterID, posComp)

	// Update for 0.3 seconds (emitter should become inactive)
	ps.Update(0.3)
	if emitterComp.Active {
		t.Error("Emitter should be inactive after SystemDuration")
	}

	// Emitter should still exist (has active particles)
	if !em.HasComponent(emitterID, reflect.TypeOf(&components.EmitterComponent{})) {
		t.Error("Emitter should still exist while particles are active")
	}

	// Update for another 1 second (all particles should expire)
	ps.Update(1.0)
	em.RemoveMarkedEntities()

	// Update once more to trigger emitter cleanup
	ps.Update(0.01)
	em.RemoveMarkedEntities()

	// Emitter should be deleted (inactive and no active particles)
	if em.HasComponent(emitterID, reflect.TypeOf(&components.EmitterComponent{})) {
		t.Error("Emitter should be auto-deleted when inactive and all particles gone")
	}
}

// TestParticleSystem_GroundCollision tests that particles bounce when hitting ground
func TestParticleSystem_GroundCollision(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create particle with downward velocity and ground constraint
	particleID := em.CreateEntity()
	particleComp := &components.ParticleComponent{
		VelocityX:         100,
		VelocityY:         165, // Moving downward
		Age:               0,
		Lifetime:          2.0,
		GroundY:           390.0, // Ground at Y=390
		CollisionReflectX: 0.3,   // 30% bounce
		CollisionReflectY: 0.3,   // 30% bounce
		CollisionReflectCurve: []particle.Keyframe{
			{Time: 0.0, Value: 0.3},
			{Time: 0.4, Value: 0.3},
			{Time: 0.5, Value: 0.0}, // Lose bounciness at 50% lifetime
		},
		Fields: []particle.Field{
			{FieldType: "Acceleration", Y: "0.17"}, // 0.17 pixel/centisecond = 17 pixels/sec² gravity
		},
	}
	posComp := &components.PositionComponent{X: 512, Y: 300} // Starting above ground
	em.AddComponent(particleID, particleComp)
	em.AddComponent(particleID, posComp)

	// Update until particle reaches ground (~0.5 seconds)
	dt := 1.0 / 60.0
	bounced := false
	maxIterations := 120 // 2 seconds max

	for i := 0; i < maxIterations; i++ {
		oldVelocityY := particleComp.VelocityY

		ps.Update(dt)

		// Check if bounce occurred (velocity changed from positive to negative)
		if oldVelocityY > 0 && particleComp.VelocityY < 0 {
			bounced = true

			// Verify bounce properties
			if posComp.Y < 389 || posComp.Y > 391 {
				t.Errorf("Particle should bounce at ground level (390), but bounced at Y=%.1f", posComp.Y)
			}

			// Verify velocity reduced by bounce coefficient (~30%)
			expectedBounceVelocity := -oldVelocityY * 0.3
			tolerance := 5.0
			if particleComp.VelocityY < expectedBounceVelocity-tolerance || particleComp.VelocityY > expectedBounceVelocity+tolerance {
				t.Errorf("Bounce velocity should be ~%.1f, got %.1f", expectedBounceVelocity, particleComp.VelocityY)
			}

			break
		}

		// Stop if particle reaches ground area
		if posComp.Y >= 390 && particleComp.VelocityY > 0 {
			// Give it a few more frames to process collision
			if i > 30 {
				break
			}
		}
	}

	if !bounced {
		t.Errorf("Particle should have bounced off the ground, but didn't. Final position: Y=%.1f, VelocityY=%.1f",
			posComp.Y, particleComp.VelocityY)
	}
}

// TestParticleSystem_GroundCollision_NoGroundConstraint tests that particles don't bounce without ground
func TestParticleSystem_GroundCollision_NoGroundConstraint(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create particle WITHOUT ground constraint
	particleID := em.CreateEntity()
	particleComp := &components.ParticleComponent{
		VelocityY:         100, // Moving downward
		Age:               0,
		Lifetime:          2.0,
		GroundY:           0, // No ground constraint
		CollisionReflectY: 0.3,
	}
	posComp := &components.PositionComponent{X: 512, Y: 300}
	em.AddComponent(particleID, particleComp)
	em.AddComponent(particleID, posComp)

	// Update for 1 second
	for i := 0; i < 60; i++ {
		ps.Update(1.0 / 60.0)
	}

	// Velocity should remain positive (no bounce)
	if particleComp.VelocityY <= 0 {
		t.Error("Particle without ground constraint should not bounce")
	}
}

// TestParticleSystem_GroundCollision_MultipleBounces tests multiple bounces with energy loss
func TestParticleSystem_GroundCollision_MultipleBounces(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create particle that can bounce multiple times
	particleID := em.CreateEntity()
	particleComp := &components.ParticleComponent{
		VelocityY:         165, // Strong downward velocity
		Age:               0,
		Lifetime:          3.0, // Long lifetime to allow multiple bounces
		GroundY:           390.0,
		CollisionReflectY: 0.3,
		Fields: []particle.Field{
			{FieldType: "Acceleration", Y: "0.17"}, // 0.17 pixel/centisecond = 17 pixels/sec² gravity
		},
	}
	posComp := &components.PositionComponent{X: 512, Y: 300}
	em.AddComponent(particleID, particleComp)
	em.AddComponent(particleID, posComp)

	// Track bounces
	bounceCount := 0
	lastVelocityY := particleComp.VelocityY
	dt := 1.0 / 60.0

	for i := 0; i < 180; i++ { // 3 seconds
		ps.Update(dt)

		// Detect bounce (velocity changed from positive to negative)
		if lastVelocityY > 0 && particleComp.VelocityY < 0 {
			bounceCount++
		}

		lastVelocityY = particleComp.VelocityY

		// Stop if velocity becomes very small
		if i > 60 && particleComp.VelocityY > -5 && particleComp.VelocityY < 5 {
			break
		}
	}

	// Should have at least 1 bounce (typically 2-3 bounces before settling)
	if bounceCount < 1 {
		t.Errorf("Expected at least 1 bounce, got %d", bounceCount)
	}

	t.Logf("Particle bounced %d times before settling", bounceCount)
}

// TestParticleSystem_GroundCollision_ReflectCurveDecay tests bounce coefficient decay over time
func TestParticleSystem_GroundCollision_ReflectCurveDecay(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create particle with bounce decay curve
	particleID := em.CreateEntity()
	particleComp := &components.ParticleComponent{
		VelocityY:         50, // Moderate downward velocity
		Age:               0,
		Lifetime:          1.0,
		GroundY:           390.0,
		CollisionReflectY: 0.3,
		CollisionReflectCurve: []particle.Keyframe{
			{Time: 0.0, Value: 0.3}, // 30% bounce at start
			{Time: 0.5, Value: 0.0}, // No bounce at 50% lifetime
		},
	}
	posComp := &components.PositionComponent{X: 512, Y: 350}
	em.AddComponent(particleID, particleComp)
	em.AddComponent(particleID, posComp)

	// Update to 60% of lifetime (should have no bounce)
	for i := 0; i < 36; i++ { // 0.6 seconds
		ps.Update(1.0 / 60.0)
	}

	// Manually trigger collision at 60% lifetime
	posComp.Y = 391 // Below ground
	particleComp.VelocityY = 50
	particleComp.Age = 0.6

	oldVelocity := particleComp.VelocityY
	ps.Update(1.0 / 60.0)

	// At 60% lifetime (> 50%), bounce should be 0 (no bounce)
	if particleComp.VelocityY < 0 {
		t.Errorf("Particle should not bounce after 50%% lifetime (bounciness decayed to 0)")
	}

	t.Logf("Collision at age 60%%: velocity %.1f → %.1f (no bounce as expected)", oldVelocity, particleComp.VelocityY)
}

// TestParticleSystem_CollisionSpin tests that particles gain spin on collision
func TestParticleSystem_CollisionSpin(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create particle with collision spin (NO decay curve to ensure effect is active)
	particleID := em.CreateEntity()
	particleComp := &components.ParticleComponent{
		VelocityY:         100,
		RotationSpeed:     0, // No initial spin
		Age:               0,
		Lifetime:          3.0, // Longer lifetime
		GroundY:           390.0,
		CollisionReflectY: 0.3,
		CollisionSpinMin:  -6.0, // Random spin range
		CollisionSpinMax:  -3.0,
		// No CollisionSpinCurve - effect stays at 100%
	}
	posComp := &components.PositionComponent{X: 512, Y: 300}
	em.AddComponent(particleID, particleComp)
	em.AddComponent(particleID, posComp)

	initialRotationSpeed := particleComp.RotationSpeed

	// Update until collision
	dt := 1.0 / 60.0
	collisionDetected := false
	for i := 0; i < 120; i++ {
		oldVelocityY := particleComp.VelocityY
		ps.Update(dt)

		// Check if collision occurred
		if oldVelocityY > 0 && particleComp.VelocityY < 0 {
			collisionDetected = true

			// Rotation speed should have changed
			if particleComp.RotationSpeed == initialRotationSpeed {
				t.Errorf("Particle should gain rotation speed on collision, but it stayed at %.1f", initialRotationSpeed)
			}

			// Should be within the expected range [-6, -3]
			if particleComp.RotationSpeed > -3 || particleComp.RotationSpeed < -6 {
				t.Logf("Warning: Collision spin %.1f outside expected range [-6, -3]", particleComp.RotationSpeed)
			}

			t.Logf("Collision spin: %.1f → %.1f (added %.1f)",
				initialRotationSpeed, particleComp.RotationSpeed, particleComp.RotationSpeed-initialRotationSpeed)
			break
		}
	}

	if !collisionDetected {
		t.Error("No collision detected - test setup issue")
	}
}

// TestParticleSystem_EmitterInstantBurst tests SpawnRate=0 (instant burst mode)
func TestParticleSystem_EmitterInstantBurst(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create emitter with SpawnRate=0 (instant burst)
	emitterID := em.CreateEntity()
	emitterComp := &components.EmitterComponent{
		Config: &particle.EmitterConfig{
			ParticleDuration: "1000",
			LaunchSpeed:      "100",
			LaunchAngle:      "0",
		},
		Active:           true,
		Age:              0,
		SystemDuration:   0,
		NextSpawnTime:    0,
		ActiveParticles:  make([]ecs.EntityID, 0),
		TotalLaunched:    0,
		SpawnRate:        0, // Instant burst
		SpawnMinActive:   5, // Spawn 5 particles immediately
		SpawnMaxActive:   100,
		SpawnMaxLaunched: 0,
	}
	posComp := &components.PositionComponent{X: 200, Y: 200}
	em.AddComponent(emitterID, emitterComp)
	em.AddComponent(emitterID, posComp)

	// First update should spawn all particles at once
	ps.Update(0.016)

	if emitterComp.TotalLaunched != 5 {
		t.Errorf("Instant burst should spawn 5 particles immediately, got %d", emitterComp.TotalLaunched)
	}

	// Second update should not spawn more
	ps.Update(0.016)
	if emitterComp.TotalLaunched != 5 {
		t.Error("Instant burst should only spawn once")
	}
}

// TestParticleSystem_EmitterCircleType tests Circle emitter with EmitterRadius
func TestParticleSystem_EmitterCircleType(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create Circle emitter with EmitterRadius
	emitterID := em.CreateEntity()
	emitterComp := &components.EmitterComponent{
		Config: &particle.EmitterConfig{
			ParticleDuration: "1000",
			LaunchSpeed:      "100",
			LaunchAngle:      "", // No angle - Circle type uses 360° random
			EmitterType:      "Circle",
		},
		Active:           true,
		Age:              0,
		SystemDuration:   0,
		NextSpawnTime:    0,
		ActiveParticles:  make([]ecs.EntityID, 0),
		TotalLaunched:    0,
		SpawnRate:        10,
		SpawnMinActive:   0,
		SpawnMaxActive:   100,
		SpawnMaxLaunched: 0,
		EmitterRadius:    50.0, // Spawn in circle radius 50
	}
	posComp := &components.PositionComponent{X: 400, Y: 300}
	em.AddComponent(emitterID, emitterComp)
	em.AddComponent(emitterID, posComp)

	// Spawn some particles
	ps.Update(0.5)

	if emitterComp.TotalLaunched < 3 {
		t.Errorf("Circle emitter should spawn particles, got %d", emitterComp.TotalLaunched)
	}

	t.Logf("Circle emitter spawned %d particles in radius 50", emitterComp.TotalLaunched)
}

// TestParticleSystem_SystemAlpha tests system-level alpha animation
func TestParticleSystem_SystemAlpha(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create particle with system alpha curve (fade out at end)
	particleID := em.CreateEntity()
	particleComp := &components.ParticleComponent{
		Alpha:    1.0,
		Age:      0,
		Lifetime: 2.0,
		SystemAlphaKeyframes: []particle.Keyframe{
			{Time: 0.0, Value: 1.0},  // Opaque at start
			{Time: 0.95, Value: 1.0}, // Stay opaque
			{Time: 1.0, Value: 0.0},  // Fade out at end
		},
		SystemAlphaInterp: "Linear",
		EmitterAge:        0,
		EmitterDuration:   2.0,
	}
	posComp := &components.PositionComponent{X: 0, Y: 0}
	em.AddComponent(particleID, particleComp)
	em.AddComponent(particleID, posComp)

	// Update to 90% of emitter duration (t=1.8s)
	for i := 0; i < 108; i++ { // 1.8 seconds
		ps.Update(1.0 / 60.0)
	}

	// At 90%, SystemAlpha should still be ~1.0 (before fade)
	if particleComp.Alpha < 0.9 {
		t.Errorf("Alpha at 90%% should be ~1.0, got %.2f", particleComp.Alpha)
	}

	t.Logf("System alpha at 90%%: %.2f (opaque)", particleComp.Alpha)
}

// TestParticleSystem_RandomLaunchSpin tests random initial rotation
func TestParticleSystem_RandomLaunchSpin(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create emitter with RandomLaunchSpin
	emitterID := em.CreateEntity()
	emitterComp := &components.EmitterComponent{
		Config: &particle.EmitterConfig{
			ParticleDuration:  "1000",
			LaunchSpeed:       "100",
			LaunchAngle:       "0",
			RandomLaunchSpin:  "1", // Enable random initial rotation
			ParticleSpinAngle: "",  // No fixed angle
		},
		Active:           true,
		Age:              0,
		SystemDuration:   0,
		NextSpawnTime:    0,
		ActiveParticles:  make([]ecs.EntityID, 0),
		TotalLaunched:    0,
		SpawnRate:        0,  // Instant burst
		SpawnMinActive:   10, // Spawn 10 particles
		SpawnMaxActive:   100,
		SpawnMaxLaunched: 0,
	}
	posComp := &components.PositionComponent{X: 200, Y: 200}
	em.AddComponent(emitterID, emitterComp)
	em.AddComponent(emitterID, posComp)

	ps.Update(0.016)

	// Check that particles have different random rotations
	particleType := reflect.TypeOf(&components.ParticleComponent{})
	particles := em.GetEntitiesWith(particleType)

	rotations := make(map[float64]bool)
	for _, particleID := range particles {
		particleComp, _ := em.GetComponent(particleID, particleType)
		particle := particleComp.(*components.ParticleComponent)
		rotations[particle.Rotation] = true
	}

	// With 10 particles and random 0-360, we should have at least 3 different values
	if len(rotations) < 3 {
		t.Errorf("RandomLaunchSpin should produce varied rotations, got %d unique values out of %d particles",
			len(rotations), len(particles))
	}

	t.Logf("RandomLaunchSpin: %d unique rotations out of %d particles", len(rotations), len(particles))
}

// TestParticleSystem_ColorDefaults tests default color values
func TestParticleSystem_ColorDefaults(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create emitter with no color/brightness specified
	emitterID := em.CreateEntity()
	emitterComp := &components.EmitterComponent{
		Config: &particle.EmitterConfig{
			ParticleDuration: "1000",
			LaunchSpeed:      "100",
			LaunchAngle:      "0",
			// No ParticleRed, ParticleGreen, ParticleBlue, ParticleBrightness
		},
		Active:           true,
		Age:              0,
		SystemDuration:   0,
		NextSpawnTime:    0,
		ActiveParticles:  make([]ecs.EntityID, 0),
		TotalLaunched:    0,
		SpawnRate:        0, // Instant burst
		SpawnMinActive:   1,
		SpawnMaxActive:   100,
		SpawnMaxLaunched: 0,
	}
	posComp := &components.PositionComponent{X: 200, Y: 200}
	em.AddComponent(emitterID, emitterComp)
	em.AddComponent(emitterID, posComp)

	ps.Update(0.016)

	// Check particle has default white color and normal brightness
	particleType := reflect.TypeOf(&components.ParticleComponent{})
	particles := em.GetEntitiesWith(particleType)

	if len(particles) == 0 {
		t.Fatal("No particle spawned")
	}

	particleComp, _ := em.GetComponent(particles[0], particleType)
	particle := particleComp.(*components.ParticleComponent)

	if particle.Red != 1.0 || particle.Green != 1.0 || particle.Blue != 1.0 {
		t.Errorf("Default color should be white (1,1,1), got (%.1f,%.1f,%.1f)",
			particle.Red, particle.Green, particle.Blue)
	}

	if particle.Brightness != 1.0 {
		t.Errorf("Default brightness should be 1.0, got %.1f", particle.Brightness)
	}
}

// TestParticleSystem_MissingParticleDuration tests that particles with missing
// ParticleDuration field use SystemDuration as fallback (Bug fix for ZombieNewspaper)
func TestParticleSystem_MissingParticleDuration(t *testing.T) {
	em := ecs.NewEntityManager()
	ps := NewParticleSystem(em, nil)

	// Create emitter with SystemDuration but NO ParticleDuration
	emitterID := em.CreateEntity()
	emitterPos := &components.PositionComponent{X: 100, Y: 100}
	emitterComp := &components.EmitterComponent{
		Config: &particle.EmitterConfig{
			Name: "TestEmitter",
			// ParticleDuration: "",  // 故意省略，模拟 ZombieNewspaper.xml
			SystemDuration: "40", // 0.4 秒
			LaunchSpeed:    "200",
			LaunchAngle:    "[120 180]",
			Image:          "IMAGE_TEST",
		},
		Active:         true,
		SystemDuration: 0.4, // 0.4 秒
		SpawnMinActive: 1,
		NextSpawnTime:  0,
	}
	em.AddComponent(emitterID, emitterPos)
	em.AddComponent(emitterID, emitterComp)

	// Spawn a particle (should use SystemDuration as fallback)
	ps.Update(0.016) // 触发粒子生成

	// Find the particle entity (查找除了发射器之外创建的粒子实体)
	var particleID ecs.EntityID
	var particleComp *components.ParticleComponent

	// 遍历所有实体，查找粒子
	particleType := reflect.TypeOf(&components.ParticleComponent{})
	emitterType := reflect.TypeOf(&components.EmitterComponent{})

	for id := ecs.EntityID(1); id <= ecs.EntityID(10); id++ {
		// 检查实体是否存在且有 ParticleComponent
		if !em.HasComponent(id, particleType) {
			continue
		}
		// 如果有 EmitterComponent，这是发射器，跳过
		if em.HasComponent(id, emitterType) {
			continue
		}
		// 找到真正的粒子实体
		comp, exists := em.GetComponent(id, particleType)
		if exists {
			particleID = id
			particleComp = comp.(*components.ParticleComponent)
			break
		}
	}

	if particleID == 0 {
		t.Fatal("No particle was created")
	}

	// 粒子应该使用 SystemDuration (0.4s) 作为 Lifetime
	expectedLifetime := 0.4
	if math.Abs(particleComp.Lifetime-expectedLifetime) > 0.001 {
		t.Errorf("Particle lifetime = %.3f, want %.3f (should use SystemDuration as fallback)",
			particleComp.Lifetime, expectedLifetime)
	}

	// 验证粒子不会立即过期
	if particleComp.Age >= particleComp.Lifetime {
		t.Errorf("Particle expired immediately: Age=%.3f, Lifetime=%.3f", particleComp.Age, particleComp.Lifetime)
	}
}
