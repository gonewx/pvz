package entities

import (
	"fmt"
	"log"

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
//   - angleOffset: (Optional) Angle offset in degrees to add to LaunchAngle (e.g., 180 to flip direction)
//
// Returns:
//   - ecs.EntityID: The ID of the created emitter entity
//   - error: Error if loading configuration fails
//
// Example:
//
//	// Simple usage (no offset)
//	emitterID, err := CreateParticleEffect(entityManager, resourceManager, "Award", 400, 300)
//
//	// With angle offset (flip direction for zombie walking right)
//	emitterID, err := CreateParticleEffect(entityManager, resourceManager, "ZombieHead", 400, 300, 180.0)
func CreateParticleEffect(em *ecs.EntityManager, rm *game.ResourceManager, effectName string, worldX, worldY float64, angleOffset ...float64) (ecs.EntityID, error) {
	// Determine angle offset (default: 0 = no offset)
	offset := 0.0
	if len(angleOffset) > 0 {
		offset = angleOffset[0]
	}

	log.Printf("[ParticleFactory] CreateParticleEffect 被调用: effectName='%s', 位置=(%.1f, %.1f), angleOffset=%.1f°",
		effectName, worldX, worldY, offset)

	// Load particle configuration from ResourceManager
	particleConfig, err := rm.LoadParticleConfig(effectName)
	if err != nil {
		log.Printf("[ParticleFactory] 加载粒子配置失败: %v", err)
		return 0, fmt.Errorf("failed to load particle config '%s': %w", effectName, err)
	}

	// Validate that configuration has at least one emitter
	if len(particleConfig.Emitters) == 0 {
		log.Printf("[ParticleFactory] 粒子配置没有发射器")
		return 0, fmt.Errorf("particle config '%s' has no emitters", effectName)
	}

	log.Printf("[ParticleFactory] 粒子配置加载成功: %d 个发射器", len(particleConfig.Emitters))

	// Story 7.4 修复：创建所有发射器（而不只是第一个）
	// 例如：PeaSplat 有2个发射器 - PeaSplat 和 PeaSplatBits
	var firstEmitterID ecs.EntityID

	for i, emitterConfig := range particleConfig.Emitters {
		// Create emitter entity
		emitterID := em.CreateEntity()
		if i == 0 {
			firstEmitterID = emitterID // 保存第一个ID用于返回
		}

		// Add PositionComponent
		positionComp := &components.PositionComponent{
			X: worldX,
			Y: worldY,
		}
		em.AddComponent(emitterID, positionComp)

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
		systemDuration := particle.RandomInRange(systemDurationMin, systemDurationMax) / 100.0 // centiseconds to seconds

		// Story 7.5: Parse SystemAlpha (ZombieHead 系统级透明度)
		_, _, systemAlphaKeyframes, systemAlphaInterp := particle.ParseValue(emitterConfig.SystemAlpha)

		// Create EmitterComponent
		emitterComp := &components.EmitterComponent{
			Config:           &emitterConfig, // Story 7.4: 取地址
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
			// Story 7.5: SystemAlpha
			SystemAlphaKeyframes: systemAlphaKeyframes,
			SystemAlphaInterp:    systemAlphaInterp,
			// Angle offset
			AngleOffset: offset,
		}
		em.AddComponent(emitterID, emitterComp)

		log.Printf("[ParticleFactory] 发射器实体创建成功: ID=%d, Name='%s', SpawnRate=%.2f, SystemDuration=%.2f",
			emitterID, emitterConfig.Name, spawnRate, systemDuration)
	}

	return firstEmitterID, nil
}
