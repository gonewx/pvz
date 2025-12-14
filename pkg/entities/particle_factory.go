package entities

import (
	"fmt"
	"log"

	"github.com/gonewx/pvz/internal/particle"
	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
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

	// IMPORTANT: 调试种植粒子数量问题 - 监控 Planting 效果创建次数
	if effectName == "Planting" {
		log.Printf("🌱 [PLANTING DEBUG] CreateParticleEffect 被调用: 位置=(%.1f, %.1f), angleOffset=%.1f°", worldX, worldY, offset)
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

		// 修复：EmitterRadius 支持范围格式 [min max]
		// 例如：Planting.xml 的 "<EmitterRadius>[0 10]</EmitterRadius>" 表示半径在 0-10 之间随机
		emitterRadiusMin, emitterRadiusMax, _, _ := particle.ParseValue(emitterConfig.EmitterRadius)

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

		// 解析发射器位置偏移量（支持范围格式，如 WallnutEatLarge 的 "[-30 10]"）
		emitterOffsetXMin, emitterOffsetXMax, _, _ := particle.ParseValue(emitterConfig.EmitterOffsetX)
		emitterOffsetYMin, emitterOffsetYMax, _, _ := particle.ParseValue(emitterConfig.EmitterOffsetY)

		systemDurationMin, systemDurationMax, _, _ := particle.ParseValue(emitterConfig.SystemDuration)
		systemDuration := particle.RandomInRange(systemDurationMin, systemDurationMax) / 100.0 // centiseconds to seconds

		// Parse SystemLoops: "1" means loop (true), "0" or empty means no loop (false)
		// When SystemLoops is true, emitter resets Age when reaching SystemDuration instead of stopping
		systemLoops := emitterConfig.SystemLoops == "1"

		// Parse ParticleLoops: "1" means loop (true), "0" or empty means no loop (false)
		// When ParticleLoops is true, particles reset Age when reaching Lifetime instead of being destroyed
		particleLoops := emitterConfig.ParticleLoops == "1"

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
			Config:          &emitterConfig, // Story 7.4: 取地址
			Active:          true,
			Age:             0,
			SystemDuration:  systemDuration,
			SystemLoops:     systemLoops,
			ParticleLoops:   particleLoops,
			NextSpawnTime:   0, // Spawn immediately
			ActiveParticles: make([]ecs.EntityID, 0),
			TotalLaunched:   0,
			SpawnRate:       spawnRate,
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
			// 修复：EmitterRadius 保存 min/max（支持范围格式）
			EmitterRadius:    emitterRadiusMin, // Deprecated: 保留用于向后兼容（旧代码可能使用）
			EmitterRadiusMin: emitterRadiusMin,
			EmitterRadiusMax: emitterRadiusMax,
			// Story 10.4: EmitterBox 关键帧（动态发射区域）
			EmitterBoxXKeyframes:    emitterBoxXWidthKf,
			EmitterBoxXInterp:       emitterBoxXInterp,
			EmitterBoxYKeyframes:    emitterBoxYWidthKf,
			EmitterBoxYInterp:       emitterBoxYInterp,
			EmitterBoxXMinKeyframes: emitterBoxXMinKf,
			EmitterBoxYMinKeyframes: emitterBoxYMinKf,
			// EmitterOffset 范围支持（每个粒子生成时随机选择）
			EmitterOffsetX:    emitterOffsetXMin, // 兼容：单值格式时 min=max
			EmitterOffsetY:    emitterOffsetYMin,
			EmitterOffsetXMin: emitterOffsetXMin,
			EmitterOffsetXMax: emitterOffsetXMax,
			EmitterOffsetYMin: emitterOffsetYMin,
			EmitterOffsetYMax: emitterOffsetYMax,
			// Story 7.5: SystemAlpha
			SystemAlphaKeyframes: systemAlphaKeyframes,
			SystemAlphaInterp:    systemAlphaInterp,
			// Story 10.4: SystemPosition (发射器位置插值)
			SystemPositionXKeyframes: systemPosXKeyframes,
			SystemPositionXInterp:    systemPosXInterp,
			SystemPositionYKeyframes: systemPosYKeyframes,
			SystemPositionYInterp:    systemPosYInterp,
			// Story 11.4: 初始位置（用于 SystemPosition 相对偏移计算）
			InitialX: worldX,
			InitialY: worldY,
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
		plantingDebugMsg := ""
		if effectName == "Planting" {
			plantingDebugMsg = fmt.Sprintf(" 🌱 [种植土粒] SpawnMinActive=%d, SpawnMaxLaunched=%d (默认=%d)",
				emitterComp.SpawnMinActive, emitterComp.SpawnMaxLaunched, emitterComp.SpawnMinActive)
		}

		if len(spawnRateKeyframes) > 0 {
			log.Printf("[ParticleFactory] 发射器实体创建成功: ID=%d, Name='%s', SpawnRate=动态(%d个关键帧), SystemDuration=%.2f, isUI=%v%s",
				emitterID, emitterConfig.Name, len(spawnRateKeyframes), systemDuration, isUIParticle, plantingDebugMsg)
		} else {
			log.Printf("[ParticleFactory] 发射器实体创建成功: ID=%d, Name='%s', SpawnRate=%.2f, SystemDuration=%.2f, isUI=%v%s",
				emitterID, emitterConfig.Name, spawnRate, systemDuration, isUIParticle, plantingDebugMsg)
		}
	}

	return firstEmitterID, nil
}

// CreateParticleEffectWithColor creates a particle emitter with custom color override.
// This is useful for effects that need a specific color different from the XML configuration.
//
// Parameters:
//   - em: EntityManager instance for creating entities
//   - rm: ResourceManager instance for loading particle configurations
//   - effectName: Name of the particle effect (e.g., "PottedPlantGlow")
//   - worldX, worldY: World coordinates where the emitter should be positioned
//   - colorR, colorG, colorB: RGB color values (0-1) to override the particle color
//
// Returns:
//   - ecs.EntityID: The ID of the first emitter entity
//   - error: Error if loading configuration fails
//
// Example:
//
//	// Create golden glow for sunflower (R=1.0, G=0.85, B=0.3)
//	emitterID, err := CreateParticleEffectWithColor(em, rm, "PottedPlantGlow", x, y, 1.0, 0.85, 0.3)
func CreateParticleEffectWithColor(em *ecs.EntityManager, rm *game.ResourceManager, effectName string, worldX, worldY float64, colorR, colorG, colorB float64) (ecs.EntityID, error) {
	log.Printf("[ParticleFactory] CreateParticleEffectWithColor 被调用: effectName='%s', 位置=(%.1f, %.1f), 颜色RGB=(%.2f, %.2f, %.2f)",
		effectName, worldX, worldY, colorR, colorG, colorB)

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

	log.Printf("[ParticleFactory] 粒子配置加载成功（带颜色覆盖）: %d 个发射器", len(particleConfig.Emitters))

	// 粒子效果作为整体管理
	sharedPosition := &components.PositionComponent{
		X: worldX,
		Y: worldY,
	}

	var firstEmitterID ecs.EntityID

	for i, emitterConfig := range particleConfig.Emitters {
		emitterID := em.CreateEntity()
		if i == 0 {
			firstEmitterID = emitterID
		}

		em.AddComponent(emitterID, sharedPosition)

		// Parse emitter parameters
		spawnRateMin, spawnRateMax, spawnRateKeyframes, spawnRateInterp := particle.ParseValue(emitterConfig.SpawnRate)
		spawnRate := particle.RandomInRange(spawnRateMin, spawnRateMax)

		spawnMinActiveVal, _, spawnMinActiveKeyframes, spawnMinActiveInterp := particle.ParseValue(emitterConfig.SpawnMinActive)
		spawnMaxActiveVal, _, spawnMaxActiveKeyframes, spawnMaxActiveInterp := particle.ParseValue(emitterConfig.SpawnMaxActive)
		spawnMaxLaunchedVal, _, spawnMaxLaunchedKeyframes, spawnMaxLaunchedInterp := particle.ParseValue(emitterConfig.SpawnMaxLaunched)

		emitterBoxXMin, emitterBoxXMax, emitterBoxXMinKf, emitterBoxXWidthKf, emitterBoxXInterp := particle.ParseRangeValue(emitterConfig.EmitterBoxX)
		emitterBoxYMin, emitterBoxYMax, emitterBoxYMinKf, emitterBoxYWidthKf, emitterBoxYInterp := particle.ParseRangeValue(emitterConfig.EmitterBoxY)

		emitterRadiusMin, emitterRadiusMax, _, _ := particle.ParseValue(emitterConfig.EmitterRadius)

		// 解析发射器位置偏移量（支持范围格式）
		emitterOffsetXMin, emitterOffsetXMax, _, _ := particle.ParseValue(emitterConfig.EmitterOffsetX)
		emitterOffsetYMin, emitterOffsetYMax, _, _ := particle.ParseValue(emitterConfig.EmitterOffsetY)

		systemDurationMin, systemDurationMax, _, _ := particle.ParseValue(emitterConfig.SystemDuration)
		systemDuration := particle.RandomInRange(systemDurationMin, systemDurationMax) / 100.0

		// Parse SystemLoops: "1" means loop (true), "0" or empty means no loop (false)
		systemLoops := emitterConfig.SystemLoops == "1"

		// Parse ParticleLoops: "1" means loop (true), "0" or empty means no loop (false)
		particleLoops := emitterConfig.ParticleLoops == "1"

		_, _, systemAlphaKeyframes, systemAlphaInterp := particle.ParseValue(emitterConfig.SystemAlpha)

		var systemPosXKeyframes, systemPosYKeyframes []particle.Keyframe
		var systemPosXInterp, systemPosYInterp string

		for _, field := range emitterConfig.SystemFields {
			if field.FieldType == "SystemPosition" {
				_, _, systemPosXKeyframes, systemPosXInterp = particle.ParseValue(field.X)
				_, _, systemPosYKeyframes, systemPosYInterp = particle.ParseValue(field.Y)
				break
			}
		}

		// Create EmitterComponent with color override enabled
		emitterComp := &components.EmitterComponent{
			Config:          &emitterConfig,
			Active:          true,
			Age:             0,
			SystemDuration:  systemDuration,
			SystemLoops:     systemLoops,
			ParticleLoops:   particleLoops,
			NextSpawnTime:   0,
			ActiveParticles: make([]ecs.EntityID, 0),
			TotalLaunched:   0,
			SpawnRate:       spawnRate,

			SpawnRateKeyframes: spawnRateKeyframes,
			SpawnRateInterp:    spawnRateInterp,

			SpawnMinActive:            int(spawnMinActiveVal),
			SpawnMinActiveKeyframes:   spawnMinActiveKeyframes,
			SpawnMinActiveInterp:      spawnMinActiveInterp,
			SpawnMaxActive:            int(spawnMaxActiveVal),
			SpawnMaxActiveKeyframes:   spawnMaxActiveKeyframes,
			SpawnMaxActiveInterp:      spawnMaxActiveInterp,
			SpawnMaxLaunched:          int(spawnMaxLaunchedVal),
			SpawnMaxLaunchedKeyframes: spawnMaxLaunchedKeyframes,
			SpawnMaxLaunchedInterp:    spawnMaxLaunchedInterp,

			EmitterBoxX:    emitterBoxXMax - emitterBoxXMin,
			EmitterBoxY:    emitterBoxYMax - emitterBoxYMin,
			EmitterBoxXMin: emitterBoxXMin,
			EmitterBoxYMin: emitterBoxYMin,

			EmitterRadius:    emitterRadiusMin,
			EmitterRadiusMin: emitterRadiusMin,
			EmitterRadiusMax: emitterRadiusMax,

			EmitterBoxXKeyframes:    emitterBoxXWidthKf,
			EmitterBoxXInterp:       emitterBoxXInterp,
			EmitterBoxYKeyframes:    emitterBoxYWidthKf,
			EmitterBoxYInterp:       emitterBoxYInterp,
			EmitterBoxXMinKeyframes: emitterBoxXMinKf,
			EmitterBoxYMinKeyframes: emitterBoxYMinKf,
			// EmitterOffset 范围支持（每个粒子生成时随机选择）
			EmitterOffsetX:    emitterOffsetXMin,
			EmitterOffsetY:    emitterOffsetYMin,
			EmitterOffsetXMin: emitterOffsetXMin,
			EmitterOffsetXMax: emitterOffsetXMax,
			EmitterOffsetYMin: emitterOffsetYMin,
			EmitterOffsetYMax: emitterOffsetYMax,

			SystemAlphaKeyframes: systemAlphaKeyframes,
			SystemAlphaInterp:    systemAlphaInterp,

			SystemPositionXKeyframes: systemPosXKeyframes,
			SystemPositionXInterp:    systemPosXInterp,
			SystemPositionYKeyframes: systemPosYKeyframes,
			SystemPositionYInterp:    systemPosYInterp,

			InitialX: worldX,
			InitialY: worldY,

			AngleOffset: 0,

			// 颜色覆盖设置
			ColorOverrideEnabled: true,
			ColorOverrideR:       colorR,
			ColorOverrideG:       colorG,
			ColorOverrideB:       colorB,
		}
		em.AddComponent(emitterID, emitterComp)

		log.Printf("[ParticleFactory] 发射器实体创建成功（颜色覆盖）: ID=%d, Name='%s', RGB=(%.2f, %.2f, %.2f)",
			emitterID, emitterConfig.Name, colorR, colorG, colorB)
	}

	return firstEmitterID, nil
}

// CreateParticleEffectWithImage creates a particle emitter with custom image override.
// This allows reusing a particle configuration but with a different image.
//
// Parameters:
//   - em: EntityManager instance for creating entities
//   - rm: ResourceManager instance for loading particle configurations
//   - effectName: Name of the particle effect (e.g., "ZombieHead")
//   - worldX, worldY: World coordinates where the emitter should be positioned
//   - imageOverride: Resource ID to use instead of the config's Image (e.g., "IMAGE_ZOMBIEPOLEVAULTERHEAD")
//   - angleOffset: Optional angle offset for launch direction (e.g., 180.0 to flip)
//
// Returns:
//   - ecs.EntityID: The ID of the first emitter entity
//   - error: Error if loading configuration fails
//
// Example:
//
//	// Pole Vaulter zombie head drop: reuse ZombieHead config with different image
//	emitterID, err := CreateParticleEffectWithImage(em, rm, "ZombieHead", x, y, "IMAGE_ZOMBIEPOLEVAULTERHEAD", 180.0)
func CreateParticleEffectWithImage(em *ecs.EntityManager, rm *game.ResourceManager, effectName string, worldX, worldY float64, imageOverride string, angleOffset float64) (ecs.EntityID, error) {
	log.Printf("[ParticleFactory] CreateParticleEffectWithImage 被调用: effectName='%s', 位置=(%.1f, %.1f), imageOverride='%s', angleOffset=%.1f°",
		effectName, worldX, worldY, imageOverride, angleOffset)

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

	log.Printf("[ParticleFactory] 粒子配置加载成功（图片覆盖）: %d 个发射器", len(particleConfig.Emitters))

	// 粒子效果作为整体管理
	sharedPosition := &components.PositionComponent{
		X: worldX,
		Y: worldY,
	}

	var firstEmitterID ecs.EntityID

	for i, emitterConfig := range particleConfig.Emitters {
		emitterID := em.CreateEntity()
		if i == 0 {
			firstEmitterID = emitterID
		}

		em.AddComponent(emitterID, sharedPosition)

		// Parse emitter parameters
		spawnRateMin, spawnRateMax, spawnRateKeyframes, spawnRateInterp := particle.ParseValue(emitterConfig.SpawnRate)
		spawnRate := particle.RandomInRange(spawnRateMin, spawnRateMax)

		spawnMinActiveVal, _, spawnMinActiveKeyframes, spawnMinActiveInterp := particle.ParseValue(emitterConfig.SpawnMinActive)
		spawnMaxActiveVal, _, spawnMaxActiveKeyframes, spawnMaxActiveInterp := particle.ParseValue(emitterConfig.SpawnMaxActive)
		spawnMaxLaunchedVal, _, spawnMaxLaunchedKeyframes, spawnMaxLaunchedInterp := particle.ParseValue(emitterConfig.SpawnMaxLaunched)

		emitterBoxXMin, emitterBoxXMax, emitterBoxXMinKf, emitterBoxXWidthKf, emitterBoxXInterp := particle.ParseRangeValue(emitterConfig.EmitterBoxX)
		emitterBoxYMin, emitterBoxYMax, emitterBoxYMinKf, emitterBoxYWidthKf, emitterBoxYInterp := particle.ParseRangeValue(emitterConfig.EmitterBoxY)

		emitterRadiusMin, emitterRadiusMax, _, _ := particle.ParseValue(emitterConfig.EmitterRadius)

		emitterOffsetXMin, emitterOffsetXMax, _, _ := particle.ParseValue(emitterConfig.EmitterOffsetX)
		emitterOffsetYMin, emitterOffsetYMax, _, _ := particle.ParseValue(emitterConfig.EmitterOffsetY)

		systemDurationMin, systemDurationMax, _, _ := particle.ParseValue(emitterConfig.SystemDuration)
		systemDuration := particle.RandomInRange(systemDurationMin, systemDurationMax) / 100.0

		systemLoops := emitterConfig.SystemLoops == "1"
		particleLoops := emitterConfig.ParticleLoops == "1"

		_, _, systemAlphaKeyframes, systemAlphaInterp := particle.ParseValue(emitterConfig.SystemAlpha)

		var systemPosXKeyframes, systemPosYKeyframes []particle.Keyframe
		var systemPosXInterp, systemPosYInterp string

		for _, field := range emitterConfig.SystemFields {
			if field.FieldType == "SystemPosition" {
				_, _, systemPosXKeyframes, systemPosXInterp = particle.ParseValue(field.X)
				_, _, systemPosYKeyframes, systemPosYInterp = particle.ParseValue(field.Y)
				break
			}
		}

		// Create EmitterComponent with image override enabled
		emitterComp := &components.EmitterComponent{
			Config:          &emitterConfig,
			Active:          true,
			Age:             0,
			SystemDuration:  systemDuration,
			SystemLoops:     systemLoops,
			ParticleLoops:   particleLoops,
			NextSpawnTime:   0,
			ActiveParticles: make([]ecs.EntityID, 0),
			TotalLaunched:   0,
			SpawnRate:       spawnRate,

			SpawnRateKeyframes: spawnRateKeyframes,
			SpawnRateInterp:    spawnRateInterp,

			SpawnMinActive:            int(spawnMinActiveVal),
			SpawnMinActiveKeyframes:   spawnMinActiveKeyframes,
			SpawnMinActiveInterp:      spawnMinActiveInterp,
			SpawnMaxActive:            int(spawnMaxActiveVal),
			SpawnMaxActiveKeyframes:   spawnMaxActiveKeyframes,
			SpawnMaxActiveInterp:      spawnMaxActiveInterp,
			SpawnMaxLaunched:          int(spawnMaxLaunchedVal),
			SpawnMaxLaunchedKeyframes: spawnMaxLaunchedKeyframes,
			SpawnMaxLaunchedInterp:    spawnMaxLaunchedInterp,

			EmitterBoxX:    emitterBoxXMax - emitterBoxXMin,
			EmitterBoxY:    emitterBoxYMax - emitterBoxYMin,
			EmitterBoxXMin: emitterBoxXMin,
			EmitterBoxYMin: emitterBoxYMin,

			EmitterRadius:    emitterRadiusMin,
			EmitterRadiusMin: emitterRadiusMin,
			EmitterRadiusMax: emitterRadiusMax,

			EmitterBoxXKeyframes:    emitterBoxXWidthKf,
			EmitterBoxXInterp:       emitterBoxXInterp,
			EmitterBoxYKeyframes:    emitterBoxYWidthKf,
			EmitterBoxYInterp:       emitterBoxYInterp,
			EmitterBoxXMinKeyframes: emitterBoxXMinKf,
			EmitterBoxYMinKeyframes: emitterBoxYMinKf,

			EmitterOffsetX:    emitterOffsetXMin,
			EmitterOffsetY:    emitterOffsetYMin,
			EmitterOffsetXMin: emitterOffsetXMin,
			EmitterOffsetXMax: emitterOffsetXMax,
			EmitterOffsetYMin: emitterOffsetYMin,
			EmitterOffsetYMax: emitterOffsetYMax,

			SystemAlphaKeyframes: systemAlphaKeyframes,
			SystemAlphaInterp:    systemAlphaInterp,

			SystemPositionXKeyframes: systemPosXKeyframes,
			SystemPositionXInterp:    systemPosXInterp,
			SystemPositionYKeyframes: systemPosYKeyframes,
			SystemPositionYInterp:    systemPosYInterp,

			InitialX: worldX,
			InitialY: worldY,

			AngleOffset: angleOffset,

			// 图片覆盖设置
			ImageOverride: imageOverride,
		}
		em.AddComponent(emitterID, emitterComp)

		log.Printf("[ParticleFactory] 发射器实体创建成功（图片覆盖）: ID=%d, Name='%s', ImageOverride='%s'",
			emitterID, emitterConfig.Name, imageOverride)
	}

	return firstEmitterID, nil
}
