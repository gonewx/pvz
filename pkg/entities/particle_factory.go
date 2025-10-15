package entities

import (
	"fmt"

	"github.com/decker502/pvz/internal/particle"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// CreateParticleEffect creates a particle emitter entity at the specified world position.
// The emitter will spawn particles according to its configuration loaded from XML.
//
// Parameters:
//   - em: EntityManager instance for creating entities
//   - rm: ResourceManager instance for loading particle configurations
//   - effectName: Name of the particle effect (e.g., "Award", "BossExplosion")
//   - worldX, worldY: World coordinates where the emitter should be positioned
//
// Returns:
//   - ecs.EntityID: The ID of the created emitter entity
//   - error: Error if loading configuration fails
//
// Example:
//
//	emitterID, err := CreateParticleEffect(entityManager, resourceManager, "Award", 400, 300)
//	if err != nil {
//	    log.Printf("Failed to create particle effect: %v", err)
//	}
func CreateParticleEffect(em *ecs.EntityManager, rm *game.ResourceManager, effectName string, worldX, worldY float64) (ecs.EntityID, error) {
	// Load particle configuration from ResourceManager
	particleConfig, err := rm.LoadParticleConfig(effectName)
	if err != nil {
		return 0, fmt.Errorf("failed to load particle config '%s': %w", effectName, err)
	}

	// Validate that configuration has at least one emitter
	if len(particleConfig.Emitters) == 0 {
		return 0, fmt.Errorf("particle config '%s' has no emitters", effectName)
	}

	// Create emitter entity
	emitterID := em.CreateEntity()

	// Add PositionComponent
	positionComp := &components.PositionComponent{
		X: worldX,
		Y: worldY,
	}
	em.AddComponent(emitterID, positionComp)

	// Use the first emitter configuration
	// (Some effects may have multiple emitters, but for simplicity we support single emitter for now)
	emitterConfig := &particleConfig.Emitters[0]

	// Parse emitter parameters from string-based configuration
	spawnRateMin, spawnRateMax, _, _ := particle.ParseValue(emitterConfig.SpawnRate)
	spawnRate := particle.RandomInRange(spawnRateMin, spawnRateMax)

	spawnMinActiveVal, _, _, _ := particle.ParseValue(emitterConfig.SpawnMinActive)
	spawnMaxActiveVal, _, _, _ := particle.ParseValue(emitterConfig.SpawnMaxActive)
	spawnMaxLaunchedVal, _, _, _ := particle.ParseValue(emitterConfig.SpawnMaxLaunched)

	emitterBoxXVal, _, _, _ := particle.ParseValue(emitterConfig.EmitterBoxX)
	emitterBoxYVal, _, _, _ := particle.ParseValue(emitterConfig.EmitterBoxY)
	emitterRadiusVal, _, _, _ := particle.ParseValue(emitterConfig.EmitterRadius)

	systemDurationMin, systemDurationMax, _, _ := particle.ParseValue(emitterConfig.SystemDuration)
	systemDuration := particle.RandomInRange(systemDurationMin, systemDurationMax) / 1000.0 // ms to seconds

	// Create EmitterComponent
	emitterComp := &components.EmitterComponent{
		Config:           emitterConfig,
		Active:           true,
		Age:              0,
		SystemDuration:   systemDuration,
		NextSpawnTime:    0, // Spawn immediately
		ActiveParticles:  make([]ecs.EntityID, 0),
		TotalLaunched:    0,
		SpawnRate:        spawnRate,
		SpawnMinActive:   int(spawnMinActiveVal),
		SpawnMaxActive:   int(spawnMaxActiveVal),
		SpawnMaxLaunched: int(spawnMaxLaunchedVal),
		EmitterBoxX:      emitterBoxXVal,
		EmitterBoxY:      emitterBoxYVal,
		EmitterRadius:    emitterRadiusVal,
	}
	em.AddComponent(emitterID, emitterComp)

	return emitterID, nil
}
