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
// 重要：粒子效果作为整体管理
//   - 返回第一个发射器ID作为"效果组"的代表
//   - 所有发射器共享相同的 PositionComponent 实例
//   - 调用者只需操作返回的ID，所有发射器自动同步
//
// Parameters:
//   - em: EntityManager instance for creating entities
//   - rm: ResourceManager instance for loading particle configurations
//   - effectName: Name of the particle effect (e.g., "Award", "BossExplosion")
//   - worldX, worldY: World coordinates where the emitter should be positioned
//   - options: Optional parameters (angleOffset, isUIParticle)
//
// Returns:
//   - ecs.EntityID: The ID of the first emitter entity (use as effect group reference)
//   - error: Error if loading configuration fails
//
// Example:
//
//	// Simple usage (no options)
//	emitterID, err := CreateParticleEffect(entityManager, resourceManager, "Award", 400, 300)
//
//	// With angle offset (flip direction for zombie walking right)
//	emitterID, err := CreateParticleEffect(entityManager, resourceManager, "ZombieHead", 400, 300, 180.0)
//
//	// Mark as UI particle (not affected by camera)
//	emitterID, err := CreateParticleEffect(entityManager, resourceManager, "SeedPacket", 400, 300, 0.0, true)
func CreateParticleEffect(em *ecs.EntityManager, rm *game.ResourceManager, effectName string, worldX, worldY float64, options ...interface{}) (ecs.EntityID, error) {
	// Parse optional parameters
	offset := 0.0
	isUIParticle := false

	for i, opt := range options {
		switch v := opt.(type) {
		case float64:
			if i == 0 {
				offset = v // First float64 is angle offset
			}
		case bool:
			isUIParticle = v // First bool is isUIParticle flag
		}
	}

	log.Printf("[ParticleFactory] CreateParticleEffect 被调用: effectName='%s', 位置=(%.1f, %.1f), angleOffset=%.1f°, isUIParticle=%v",
		effectName, worldX, worldY, offset, isUIParticle)

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

	// 粒子效果作为整体管理：
	// 所有发射器共享同一个 PositionComponent 实例
	// 这样更新位置时，所有发射器自动同步
	sharedPosition := &components.PositionComponent{
		X: worldX,
		Y: worldY,
	}

	// Story 7.4 修复：创建所有发射器（而不只是第一个）
	// 例如：SeedPacket 有2个发射器 - 箭头 + 光晕
	var firstEmitterID ecs.EntityID

	for i, emitterConfig := range particleConfig.Emitters {
		// Create emitter entity
		emitterID := em.CreateEntity()
		if i == 0 {
			firstEmitterID = emitterID // 保存第一个ID用于返回
		}

		// Add shared PositionComponent（所有发射器共享同一个实例）
		em.AddComponent(emitterID, sharedPosition)

		// Parse emitter parameters from string-based configuration
		spawnRateMin, spawnRateMax, _, _ := particle.ParseValue(emitterConfig.SpawnRate)
		spawnRate := particle.RandomInRange(spawnRateMin, spawnRateMax)

		spawnMinActiveVal, _, _, _ := particle.ParseValue(emitterConfig.SpawnMinActive)
		spawnMaxActiveVal, _, _, _ := particle.ParseValue(emitterConfig.SpawnMaxActive)
		spawnMaxLaunchedVal, _, _, _ := particle.ParseValue(emitterConfig.SpawnMaxLaunched)

		emitterBoxXVal, _, _, _ := particle.ParseValue(emitterConfig.EmitterBoxX)
		emitterBoxYVal, _, _, _ := particle.ParseValue(emitterConfig.EmitterBoxY)
		emitterRadiusVal, _, _, _ := particle.ParseValue(emitterConfig.EmitterRadius)

		// 解析发射器位置偏移量
		emitterOffsetXVal, _, _, _ := particle.ParseValue(emitterConfig.EmitterOffsetX)
		emitterOffsetYVal, _, _, _ := particle.ParseValue(emitterConfig.EmitterOffsetY)

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
			EmitterOffsetX:   emitterOffsetXVal,
			EmitterOffsetY:   emitterOffsetYVal,
			// Story 7.5: SystemAlpha
			SystemAlphaKeyframes: systemAlphaKeyframes,
			SystemAlphaInterp:    systemAlphaInterp,
			// Angle offset
			AngleOffset: offset,
		}
		em.AddComponent(emitterID, emitterComp)

		// 如果标记为UI粒子，给发射器添加 UIComponent
		// 这样所有从这个发射器生成的粒子都会继承 UIComponent
		if isUIParticle {
			em.AddComponent(emitterID, &components.UIComponent{})
		}

		log.Printf("[ParticleFactory] 发射器实体创建成功: ID=%d, Name='%s', SpawnRate=%.2f, SystemDuration=%.2f, isUI=%v",
			emitterID, emitterConfig.Name, spawnRate, systemDuration, isUIParticle)
	}

	return firstEmitterID, nil
}
