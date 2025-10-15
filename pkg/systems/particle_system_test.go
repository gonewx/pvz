package systems

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/internal/particle"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
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
				Y:         "100", // 100 pixels/sec^2 downward
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
	particleID := em.CreateEntity()
	particleComp := &components.ParticleComponent{
		VelocityX: 100, // Initial velocity
		VelocityY: 0,
		Age:       0,
		Lifetime:  10,
		Fields: []particle.Field{
			{
				FieldType: "Friction",
				X:         "0.5", // 50% friction per second
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
			SpawnRate:        10,   // 10 particles/second
			SpawnMaxActive:   100,  // Allow many active
			SpawnMaxLaunched: 5,    // Only spawn 5 particles total per emitter
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

