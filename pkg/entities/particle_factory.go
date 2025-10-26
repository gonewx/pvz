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
		// Story 7.x: SpawnRate 支持关键帧动画（修复 GraveStoneRise 等粒子效果）
		spawnRateMin, spawnRateMax, spawnRateKeyframes, spawnRateInterp := particle.ParseValue(emitterConfig.SpawnRate)
		spawnRate := particle.RandomInRange(spawnRateMin, spawnRateMax)

		// Parse spawn constraints (支持关键帧动画)
		spawnMinActiveVal, _, spawnMinActiveKeyframes, spawnMinActiveInterp := particle.ParseValue(emitterConfig.SpawnMinActive)
		spawnMaxActiveVal, _, spawnMaxActiveKeyframes, spawnMaxActiveInterp := particle.ParseValue(emitterConfig.SpawnMaxActive)
		spawnMaxLaunchedVal, _, spawnMaxLaunchedKeyframes, spawnMaxLaunchedInterp := particle.ParseValue(emitterConfig.SpawnMaxLaunched)

		// Story 10.4: 解析 EmitterBox 关键帧（支持动态发射区域变化）
		// 使用 ParseRangeValue 专门处理范围类型，保留负数和非对称范围信息
		// 例如：SodRoll.xml 的 EmitterBoxY="[-130 0] [-100 0]"
		//   → initialMin=-130, initialMax=0
		//   → minKeyframes=[{0,-130}, {1,-100}], widthKeyframes=[{0,130}, {1,100}]
		emitterBoxXMin, emitterBoxXMax, emitterBoxXMinKf, emitterBoxXWidthKf, emitterBoxXInterp := particle.ParseRangeValue(emitterConfig.EmitterBoxX)
		emitterBoxYMin, emitterBoxYMax, emitterBoxYMinKf, emitterBoxYWidthKf, emitterBoxYInterp := particle.ParseRangeValue(emitterConfig.EmitterBoxY)
		emitterRadiusVal, _, _, _ := particle.ParseValue(emitterConfig.EmitterRadius)

		// DEBUG: 输出 EmitterBox 关键帧解析结果
		if len(emitterBoxXWidthKf) > 0 || len(emitterBoxYWidthKf) > 0 {
			log.Printf("[ParticleFactory] EmitterBox 关键帧解析:")
			if len(emitterBoxXWidthKf) > 0 {
				log.Printf("  X: min=%v, width关键帧=%v", emitterBoxXMinKf, emitterBoxXWidthKf)
			}
			if len(emitterBoxYWidthKf) > 0 {
				log.Printf("  Y: min=%v, width关键帧=%v", emitterBoxYMinKf, emitterBoxYWidthKf)
			}
		}

		// 解析发射器位置偏移量
		emitterOffsetXVal, _, _, _ := particle.ParseValue(emitterConfig.EmitterOffsetX)
		emitterOffsetYVal, _, _, _ := particle.ParseValue(emitterConfig.EmitterOffsetY)

		systemDurationMin, systemDurationMax, _, _ := particle.ParseValue(emitterConfig.SystemDuration)
		systemDuration := particle.RandomInRange(systemDurationMin, systemDurationMax) / 100.0 // centiseconds to seconds

		// Story 7.5: Parse SystemAlpha (ZombieHead 系统级透明度)
		_, _, systemAlphaKeyframes, systemAlphaInterp := particle.ParseValue(emitterConfig.SystemAlpha)

		// Story 10.4: Parse SystemFields (SystemPosition 等系统级力场)
		// 例如：SodRoll.xml 中的 <SystemField><FieldType>SystemPosition</FieldType><X>0 740</X><Y>30 0</Y></SystemField>
		var systemPosXKeyframes, systemPosYKeyframes []particle.Keyframe
		var systemPosXInterp, systemPosYInterp string

		for _, field := range emitterConfig.SystemFields {
			if field.FieldType == "SystemPosition" {
				// 解析 X 和 Y 的关键帧
				_, _, systemPosXKeyframes, systemPosXInterp = particle.ParseValue(field.X)
				_, _, systemPosYKeyframes, systemPosYInterp = particle.ParseValue(field.Y)
				log.Printf("[ParticleFactory] SystemPosition 解析成功: X=%d个关键帧, Y=%d个关键帧",
					len(systemPosXKeyframes), len(systemPosYKeyframes))
				break // 只处理第一个 SystemPosition
			}
		}

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
			// 保存 SpawnRate 关键帧数据（用于动态生成率控制）
			SpawnRateKeyframes: spawnRateKeyframes,
			SpawnRateInterp:    spawnRateInterp,
			// 保存 Spawn 约束关键帧数据（用于动态粒子数量控制）
			SpawnMinActive:            int(spawnMinActiveVal),
			SpawnMinActiveKeyframes:   spawnMinActiveKeyframes,
			SpawnMinActiveInterp:      spawnMinActiveInterp,
			SpawnMaxActive:            int(spawnMaxActiveVal),
			SpawnMaxActiveKeyframes:   spawnMaxActiveKeyframes,
			SpawnMaxActiveInterp:      spawnMaxActiveInterp,
			SpawnMaxLaunched:          int(spawnMaxLaunchedVal),
			SpawnMaxLaunchedKeyframes: spawnMaxLaunchedKeyframes,
			SpawnMaxLaunchedInterp:    spawnMaxLaunchedInterp,
			// EmitterBox: 初始范围宽度（用于单范围格式和双范围初始值）
			EmitterBoxX:    emitterBoxXMax - emitterBoxXMin,
			EmitterBoxY:    emitterBoxYMax - emitterBoxYMin,
			EmitterBoxXMin: emitterBoxXMin,
			EmitterBoxYMin: emitterBoxYMin,
			EmitterRadius:  emitterRadiusVal,
			// Story 10.4: EmitterBox 关键帧（动态发射区域）
			EmitterBoxXKeyframes:    emitterBoxXWidthKf,
			EmitterBoxXInterp:       emitterBoxXInterp,
			EmitterBoxYKeyframes:    emitterBoxYWidthKf,
			EmitterBoxYInterp:       emitterBoxYInterp,
			EmitterBoxXMinKeyframes: emitterBoxXMinKf,
			EmitterBoxYMinKeyframes: emitterBoxYMinKf,
			EmitterOffsetX:          emitterOffsetXVal,
			EmitterOffsetY:          emitterOffsetYVal,
			// Story 7.5: SystemAlpha
			SystemAlphaKeyframes: systemAlphaKeyframes,
			SystemAlphaInterp:    systemAlphaInterp,
			// Story 10.4: SystemPosition (发射器位置插值)
			SystemPositionXKeyframes: systemPosXKeyframes,
			SystemPositionXInterp:    systemPosXInterp,
			SystemPositionYKeyframes: systemPosYKeyframes,
			SystemPositionYInterp:    systemPosYInterp,
			// Angle offset
			AngleOffset: offset,
		}
		em.AddComponent(emitterID, emitterComp)

		// 如果标记为UI粒子，给发射器添加 UIComponent
		// 这样所有从这个发射器生成的粒子都会继承 UIComponent
		if isUIParticle {
			em.AddComponent(emitterID, &components.UIComponent{})
		}

		// Story 10.4: 改进日志，显示 SpawnRate 关键帧信息
		if len(spawnRateKeyframes) > 0 {
			log.Printf("[ParticleFactory] 发射器实体创建成功: ID=%d, Name='%s', SpawnRate=动态(%d个关键帧), SystemDuration=%.2f, isUI=%v",
				emitterID, emitterConfig.Name, len(spawnRateKeyframes), systemDuration, isUIParticle)
		} else {
			log.Printf("[ParticleFactory] 发射器实体创建成功: ID=%d, Name='%s', SpawnRate=%.2f, SystemDuration=%.2f, isUI=%v",
				emitterID, emitterConfig.Name, spawnRate, systemDuration, isUIParticle)
		}
	}

	return firstEmitterID, nil
}
