package systems

import (
	"log"
	"math"
	"math/rand"
	"reflect"

	"github.com/hajimehoshi/ebiten/v2"

	particlePkg "github.com/decker502/pvz/internal/particle"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// ParticleSystem manages all particle emitters and individual particles.
// It handles spawning particles from emitters, updating their properties
// each frame (position, rotation, alpha, etc.), and destroying particles
// when their lifetime expires.
//
// The system processes particles in two phases:
//  1. Update all emitters (spawn new particles, check duration limits)
//  2. Update all particles (apply velocity, forces, interpolation)
//
// Follows ECS zero-coupling principle: communicates only through EntityManager.
type ParticleSystem struct {
	EntityManager   *ecs.EntityManager
	ResourceManager *game.ResourceManager
}

// NewParticleSystem creates a new ParticleSystem instance.
func NewParticleSystem(em *ecs.EntityManager, rm *game.ResourceManager) *ParticleSystem {
	return &ParticleSystem{
		EntityManager:   em,
		ResourceManager: rm,
	}
}

// Update processes all emitters and particles for the current frame.
// dt is the delta time in seconds since the last frame.
func (ps *ParticleSystem) Update(dt float64) {
	ps.updateEmitters(dt)
	ps.updateParticles(dt)
}

// updateEmitters processes all emitter entities, spawning new particles
// and managing emitter lifecycle.
func (ps *ParticleSystem) updateEmitters(dt float64) {
	// Query all entities with EmitterComponent and PositionComponent
	emitterType := reflect.TypeOf(&components.EmitterComponent{})
	positionType := reflect.TypeOf(&components.PositionComponent{})
	emitterEntities := ps.EntityManager.GetEntitiesWith(emitterType, positionType)

	// DEBUG: 发射器数量日志（每帧打印会刷屏，已禁用）
	// if len(emitterEntities) > 0 {
	// 	log.Printf("[ParticleSystem] updateEmitters: 找到 %d 个发射器实体", len(emitterEntities))
	// }

	for _, emitterID := range emitterEntities {
		// Get emitter component
		emitterComp, ok := ps.EntityManager.GetComponent(emitterID, emitterType)
		if !ok {
			continue
		}
		emitter := emitterComp.(*components.EmitterComponent)

		// Get position component
		posComp, ok := ps.EntityManager.GetComponent(emitterID, positionType)
		if !ok {
			continue
		}
		position := posComp.(*components.PositionComponent)

		// DEBUG: 发射器处理日志（每个发射器每帧都打印会刷屏，已禁用）
		// log.Printf("[ParticleSystem] 处理发射器 ID=%d: Active=%v, Age=%.2f, SpawnRate=%.2f, NextSpawnTime=%.2f",
		// 	emitterID, emitter.Active, emitter.Age, emitter.SpawnRate, emitter.NextSpawnTime)

		// Update emitter age
		emitter.Age += dt

		// Check system duration (0 = infinite)
		if emitter.SystemDuration > 0 && emitter.Age >= emitter.SystemDuration {
			emitter.Active = false
		}

		// Spawn particles if emitter is active
		if emitter.Active && emitter.Config != nil {
			// Story 7.4 Fix: Handle SpawnRate=0 (instant burst effects)
			// When SpawnRate=0, spawn all particles immediately on first update
			if emitter.SpawnRate == 0 {
				if emitter.TotalLaunched == 0 {
					// Instant burst: spawn SpawnMinActive particles immediately
					targetCount := emitter.SpawnMinActive
					if targetCount == 0 {
						targetCount = 1 // At least one particle
					}

					// DEBUG: 立即生成模式日志（已禁用避免刷屏）
					// log.Printf("[ParticleSystem] 立即生成模式: 生成 %d 个粒子", targetCount)

					for i := 0; i < targetCount; i++ {
						// Check spawn constraints
						activeCount := len(emitter.ActiveParticles)
						canSpawn := true
						if emitter.SpawnMaxActive > 0 && activeCount >= emitter.SpawnMaxActive {
							canSpawn = false
							break
						}
						if emitter.SpawnMaxLaunched > 0 && emitter.TotalLaunched >= emitter.SpawnMaxLaunched {
							canSpawn = false
							break
						}

						if canSpawn {
							ps.spawnParticle(emitterID, emitter, position)
							emitter.TotalLaunched++
						}
					}
				}
			} else if emitter.SpawnRate > 0 {
				// Continuous spawn mode: spawn particles at regular intervals
				for emitter.Age >= emitter.NextSpawnTime {
					activeCount := len(emitter.ActiveParticles)

					// Check spawn constraints
					canSpawn := true
					if emitter.SpawnMaxActive > 0 && activeCount >= emitter.SpawnMaxActive {
						canSpawn = false
						break // Can't spawn more this frame
					}
					if emitter.SpawnMaxLaunched > 0 && emitter.TotalLaunched >= emitter.SpawnMaxLaunched {
						canSpawn = false
						break // Reached total launch limit
					}

					if canSpawn {
						ps.spawnParticle(emitterID, emitter, position)
						emitter.TotalLaunched++
						activeCount++ // Update local count
					}

					// Update next spawn time
					emitter.NextSpawnTime += 1.0 / emitter.SpawnRate

					// Safety check to avoid infinite loop
					if emitter.NextSpawnTime > emitter.Age+10 {
						break
					}
				}
			}
		}

		// Clean up destroyed particles from active list
		ps.cleanupDestroyedParticles(emitter)

		// Story 7.4: Auto-cleanup emitter entities when finished
		// Delete emitter if it's inactive and has no active particles
		if !emitter.Active && len(emitter.ActiveParticles) == 0 {
			ps.EntityManager.DestroyEntity(emitterID)
			// Note: No need to log here as it's expected behavior
		}
	}
}

// cleanupDestroyedParticles removes dead particle IDs from emitter's active list
func (ps *ParticleSystem) cleanupDestroyedParticles(emitter *components.EmitterComponent) {
	particleType := reflect.TypeOf(&components.ParticleComponent{})
	alive := make([]ecs.EntityID, 0, len(emitter.ActiveParticles))

	for _, particleID := range emitter.ActiveParticles {
		// Check if particle still exists
		if ps.EntityManager.HasComponent(particleID, particleType) {
			alive = append(alive, particleID)
		}
	}

	emitter.ActiveParticles = alive
}

// updateParticles processes all particle entities, updating their state
// and destroying expired particles.
func (ps *ParticleSystem) updateParticles(dt float64) {
	particleType := reflect.TypeOf(&components.ParticleComponent{})
	positionType := reflect.TypeOf(&components.PositionComponent{})
	particleEntities := ps.EntityManager.GetEntitiesWith(particleType, positionType)

	for _, particleID := range particleEntities {
		// Get particle component
		particleComp, ok := ps.EntityManager.GetComponent(particleID, particleType)
		if !ok {
			continue
		}
		particle := particleComp.(*components.ParticleComponent)

		// Get position component
		posComp, ok := ps.EntityManager.GetComponent(particleID, positionType)
		if !ok {
			continue
		}
		position := posComp.(*components.PositionComponent)

		// Update particle age
		particle.Age += dt

		// Story 7.5: Update system age (for SystemAlpha calculation)
		particle.EmitterAge += dt

		// Check if particle has expired
		if particle.Age >= particle.Lifetime {
			ps.EntityManager.DestroyEntity(particleID)
			// DEBUG: 销毁粒子日志（每个粒子结束时打印会刷屏，已禁用）
			// log.Printf("[ParticleSystem] 销毁过期粒子: ID=%d, Age=%.2f, Lifetime=%.2f", particleID, particle.Age, particle.Lifetime)
			continue
		}

		// DEBUG: 粒子生命周期信息（每帧打印会刷屏，已禁用）
		// if particle.Age < 0.1 {
		// 	log.Printf("[ParticleSystem] 粒子状态: ID=%d, Age=%.3f/%.2f, Alpha=%.2f, Scale=%.2f",
		// 		particleID, particle.Age, particle.Lifetime, particle.Alpha, particle.Scale)
		// }

		// Apply velocity to position
		position.X += particle.VelocityX * dt
		position.Y += particle.VelocityY * dt

		// Story 7.5: Ground collision detection (ZombieHead 弹跳效果)
		if particle.GroundY > 0 && position.Y >= particle.GroundY {
			// 粒子穿过地面，触发碰撞（只在真正穿过时碰撞）
			if particle.VelocityY > 0 { // 只有向下运动时才碰撞
				position.Y = particle.GroundY // 重置到地面位置

				// 计算反弹系数（可能随时间衰减）
				reflectX := particle.CollisionReflectX
				reflectY := particle.CollisionReflectY
				if len(particle.CollisionReflectCurve) > 0 {
					t := particle.Age / particle.Lifetime
					reflect := particlePkg.EvaluateKeyframes(particle.CollisionReflectCurve, t, "")
					reflectX = reflect
					reflectY = reflect
				}

				// 反弹：速度反向并乘以反弹系数
				particle.VelocityY = -particle.VelocityY * reflectY
				particle.VelocityX = particle.VelocityX * reflectX

				// 碰撞旋转效果（可能随时间衰减）
				if particle.CollisionSpinMin != 0 || particle.CollisionSpinMax != 0 {
					// 从范围随机选择基础碰撞旋转增量
					baseSpin := particlePkg.RandomInRange(particle.CollisionSpinMin, particle.CollisionSpinMax)

					// Story 7.5 修复：应用衰减曲线作为乘数
					// 例如：初始乘数=1，在40%时衰减到0
					spinMultiplier := 1.0
					if len(particle.CollisionSpinCurve) > 0 {
						t := particle.Age / particle.Lifetime
						spinMultiplier = particlePkg.EvaluateKeyframes(particle.CollisionSpinCurve, t, "")
					}

					// 最终效果 = 基础值 * 衰减乘数
					spinEffect := baseSpin * spinMultiplier
					particle.RotationSpeed += spinEffect
				}
			}
		}

		// Apply rotation
		particle.Rotation += particle.RotationSpeed * dt

		// Apply force fields (acceleration, friction, etc.)
		ps.applyFields(particle, dt)

		// Apply interpolation (alpha, scale, spin animations)
		ps.applyInterpolation(particle)
	}
}

// spawnParticle creates a new particle entity based on emitter configuration.
func (ps *ParticleSystem) spawnParticle(emitterID ecs.EntityID, emitter *components.EmitterComponent, emitterPos *components.PositionComponent) {
	if emitter.Config == nil {
		return
	}

	config := emitter.Config
	// DEBUG: 粒子生成日志（每个粒子生成时打印会刷屏，已禁用）
	// log.Printf("[ParticleSystem] spawnParticle 被调用: emitterID=%d, 位置=(%.1f, %.1f), 图片ID=%s",
	// 	emitterID, emitterPos.X, emitterPos.Y, config.Image)

	// Create new particle entity
	particleID := ps.EntityManager.CreateEntity()

	// Parse initial values from config
	// Particle duration (convert centiseconds to seconds)
	durationMin, durationMax, _, _ := particlePkg.ParseValue(config.ParticleDuration)
	lifetime := particlePkg.RandomInRange(durationMin, durationMax) / 100.0 // centiseconds to seconds

	// Launch speed and angle
	speedMin, speedMax, _, _ := particlePkg.ParseValue(config.LaunchSpeed)
	angleMin, angleMax, _, _ := particlePkg.ParseValue(config.LaunchAngle)
	speed := particlePkg.RandomInRange(speedMin, speedMax)
	angle := particlePkg.RandomInRange(angleMin, angleMax)

	// Story 7.4 修复：如果 LaunchAngle 未定义且发射器类型是 Circle，使用随机 360° 角度
	if angleMin == 0 && angleMax == 0 && config.LaunchAngle == "" && config.EmitterType == "Circle" {
		angle = rand.Float64() * 360.0 // 0-360 度随机
	}

	// Story 7.6 修复：PvZ 角度坐标系转换
	// PvZ 使用的坐标系：0° = 向左（僵尸前进方向），180° = 向右（僵尸后方）
	// 屏幕坐标系：0° = 向右，180° = 向左
	// 转换公式：screenAngle = (pvzAngle + 180) % 360
	screenAngle := angle + 180.0
	if screenAngle >= 360.0 {
		screenAngle -= 360.0
	}

	// Convert angle to radians and calculate velocity components
	angleRad := screenAngle * math.Pi / 180.0
	velocityX := speed * math.Cos(angleRad)
	velocityY := -speed * math.Sin(angleRad) // 取反以适配屏幕坐标系（Y轴向下为正）

	// Initial rotation and spin speed
	spinAngleMin, spinAngleMax, _, _ := particlePkg.ParseValue(config.ParticleSpinAngle)
	spinSpeedMin, spinSpeedMax, spinKeyframes, spinInterp := particlePkg.ParseValue(config.ParticleSpinSpeed)
	initialRotation := particlePkg.RandomInRange(spinAngleMin, spinAngleMax)
	initialSpinSpeed := particlePkg.RandomInRange(spinSpeedMin, spinSpeedMax)
	// 如果未提供 SpinAngle 且配置了 RandomLaunchSpin，则随机初始朝向（0-360 度）
	if (spinAngleMin == 0 && spinAngleMax == 0) && config.RandomLaunchSpin == "1" {
		initialRotation = rand.Float64() * 360.0
	}

	// Story 7.5 修复：对于"范围+关键帧"格式，需要将初始值添加到关键帧开头
	// 例如：ParticleSpinSpeed="[-720 720] 0,39.999996" 返回 min=-720, max=720, keyframes=[{0.4, 0}]
	// 需要添加初始关键帧：[{0, initialSpinSpeed}, {0.4, 0}]
	if len(spinKeyframes) > 0 && (spinSpeedMin != 0 || spinSpeedMax != 0) {
		// 检查第一个关键帧是否在时间 0
		if spinKeyframes[0].Time > 0 {
			// 在开头插入初始值关键帧
			spinKeyframes = append([]particlePkg.Keyframe{{Time: 0, Value: initialSpinSpeed}}, spinKeyframes...)
		}
	}

	// Scale
	scaleMin, scaleMax, scaleKeyframes, scaleInterp := particlePkg.ParseValue(config.ParticleScale)
	initialScale := particlePkg.RandomInRange(scaleMin, scaleMax)
	if initialScale == 0 {
		initialScale = 1.0 // Default scale
	}

	// Alpha (transparency)
	alphaMin, alphaMax, alphaKeyframes, alphaInterp := particlePkg.ParseValue(config.ParticleAlpha)
	var initialAlpha float64
	if len(alphaKeyframes) > 0 {
		// Story 7.4 修复：如果有关键帧，从第一个关键帧获取初始值
		initialAlpha = alphaKeyframes[0].Value
	} else {
		initialAlpha = particlePkg.RandomInRange(alphaMin, alphaMax)
		if initialAlpha == 0 {
			initialAlpha = 1.0 // Default fully opaque
		}
	}

	// Color channels
	redMin, redMax, _, _ := particlePkg.ParseValue(config.ParticleRed)
	greenMin, greenMax, _, _ := particlePkg.ParseValue(config.ParticleGreen)
	blueMin, blueMax, _, _ := particlePkg.ParseValue(config.ParticleBlue)
	red := particlePkg.RandomInRange(redMin, redMax)
	green := particlePkg.RandomInRange(greenMin, greenMax)
	blue := particlePkg.RandomInRange(blueMin, blueMax)
	if red == 0 && green == 0 && blue == 0 {
		red, green, blue = 1.0, 1.0, 1.0 // Default white (显示原始纹理颜色)
	}

	// Brightness
	brightnessMin, brightnessMax, _, _ := particlePkg.ParseValue(config.ParticleBrightness)
	brightness := particlePkg.RandomInRange(brightnessMin, brightnessMax)
	if brightness == 0 {
		brightness = 1.0 // Default brightness
	}

	// Spawn position
	// 优先使用圆形发射半径（EmitterRadius），否则回退到方形发射盒（EmitterBoxX/Y）
	spawnX := emitterPos.X
	spawnY := emitterPos.Y
	if emitter.EmitterRadius > 0 {
		// 均匀分布在圆形区域内：半径使用 sqrt 随机，角度均匀
		r := math.Sqrt(rand.Float64()) * emitter.EmitterRadius
		ang := rand.Float64() * 2 * math.Pi
		spawnX += r * math.Cos(ang)
		spawnY += r * math.Sin(ang)
	} else {
		if emitter.EmitterBoxX > 0 {
			spawnX += rand.Float64()*emitter.EmitterBoxX - emitter.EmitterBoxX/2
		}
		if emitter.EmitterBoxY > 0 {
			spawnY += rand.Float64()*emitter.EmitterBoxY - emitter.EmitterBoxY/2
		}
	}

	// Additive blending
	additive := false
	if config.Additive == "1" {
		additive = true
	}

	// Load particle image from ResourceManager (Story 7.4 修复)
	// config.Image 包含资源 ID（如 "IMAGE_ZOMBIEARM"）
	var particleImage *ebiten.Image
	imageFrames := 1 // 默认单帧
	frameNum := 0    // 默认第 0 帧

	if config.Image != "" && ps.ResourceManager != nil {
		img, err := ps.ResourceManager.LoadImageByID(config.Image)
		if err != nil {
			// 图片加载失败，记录错误但不阻塞粒子生成
			// 粒子会创建但不渲染（因为 Image == nil）
			log.Printf("[ParticleSystem] 警告：无法加载粒子图片 '%s': %v", config.Image, err)
		} else {
			particleImage = img

			// 解析 ImageFrames（字符串转整数）
			if config.ImageFrames != "" {
				// ParseValue 返回 (min, max, keyframes, interpolation)
				// 对于简单数字字符串，min == max == 解析后的值
				framesMin, framesMax, _, _ := particlePkg.ParseValue(config.ImageFrames)
				parsedFrames := int(framesMin)
				if parsedFrames == 0 {
					parsedFrames = int(framesMax) // 尝试使用 max 值
				}
				if parsedFrames > 0 {
					imageFrames = parsedFrames
				}
			}

			// 如果是多帧精灵图，选择随机帧
			if imageFrames > 1 {
				frameNum = rand.Intn(imageFrames)
			}
		}
	}

	// Story 7.5: Parse collision properties (ZombieHead 弹跳效果)
	var collisionReflectX, collisionReflectY float64
	var collisionReflectCurve []particlePkg.Keyframe
	var collisionSpinMin, collisionSpinMax float64
	var collisionSpinCurve []particlePkg.Keyframe
	var groundY float64

	if config.CollisionReflect != "" {
		// CollisionReflect 格式: ".3 .3,39.999996 0,50"
		// 第一个值是初始反弹系数，后续是关键帧
		reflectXMin, reflectXMax, reflectKeyframes, _ := particlePkg.ParseValue(config.CollisionReflect)
		collisionReflectX = particlePkg.RandomInRange(reflectXMin, reflectXMax)
		collisionReflectY = collisionReflectX // 默认X和Y使用相同值
		collisionReflectCurve = reflectKeyframes
	}

	if config.CollisionSpin != "" {
		// CollisionSpin 格式: "[-3 -6] 0,39.999996"
		spinMin, spinMax, spinCurve, _ := particlePkg.ParseValue(config.CollisionSpin)
		collisionSpinMin = spinMin
		collisionSpinMax = spinMax
		collisionSpinCurve = spinCurve

		// Story 7.5 修复：对于"范围+关键帧"格式，添加初始乘数关键帧
		// 例如：CollisionSpin="[-3 -6] 0,39.999996" 返回 keyframes=[{0.4, 0}]
		// 表示碰撞旋转效果从100%衰减到0%，需要添加初始关键帧：[{0, 1}, {0.4, 0}]
		if len(spinCurve) > 0 && (spinMin != 0 || spinMax != 0) {
			// 检查第一个关键帧是否在时间 0
			if spinCurve[0].Time > 0 {
				// 在开头插入初始乘数关键帧（值为1，表示100%效果）
				collisionSpinCurve = append([]particlePkg.Keyframe{{Time: 0, Value: 1}}, spinCurve...)
			}
		}
	}

	// 查找 GroundConstraint 字段（相对于发射器位置的偏移量）
	// Story 7.5 修复：GroundConstraint 的 Y 值是相对于粒子生成位置的偏移量
	// 例如：粒子生成在 Y=384，GroundConstraint Y=90，实际地面 = 384 + 90 = 474
	for _, field := range config.Fields {
		if field.FieldType == "GroundConstraint" && field.Y != "" {
			yMin, yMax, _, _ := particlePkg.ParseValue(field.Y)
			groundOffset := particlePkg.RandomInRange(yMin, yMax)
			groundY = spawnY + groundOffset // 相对坐标：发射器Y + 偏移量
			break
		}
	}

	// Create ParticleComponent
	particleComp := &components.ParticleComponent{
		VelocityX:     velocityX,
		VelocityY:     velocityY,
		Rotation:      initialRotation,
		RotationSpeed: initialSpinSpeed,
		Scale:         initialScale,
		Alpha:         initialAlpha,
		Red:           red,
		Green:         green,
		Blue:          blue,
		Brightness:    brightness,
		Age:           0,
		Lifetime:      lifetime,

		AlphaKeyframes:     alphaKeyframes,
		ScaleKeyframes:     scaleKeyframes,
		SpinKeyframes:      spinKeyframes,
		AlphaInterpolation: alphaInterp,
		ScaleInterpolation: scaleInterp,
		SpinInterpolation:  spinInterp,

		Image:       particleImage, // Story 7.4: Loaded from ResourceManager
		ImageFrames: imageFrames,   // Story 7.4: 精灵图帧数
		FrameNum:    frameNum,      // Story 7.4: 当前帧编号
		Additive:    additive,
		Fields:      config.Fields, // Copy force fields from config

		// Story 7.5: Collision properties
		CollisionReflectX:     collisionReflectX,
		CollisionReflectY:     collisionReflectY,
		CollisionReflectCurve: collisionReflectCurve,
		CollisionSpinMin:      collisionSpinMin,
		CollisionSpinMax:      collisionSpinMax,
		CollisionSpinCurve:    collisionSpinCurve,
		GroundY:               groundY,

		// Story 7.5: System-level alpha (ZombieHead 系统淡出)
		SystemAlphaKeyframes: emitter.SystemAlphaKeyframes,
		SystemAlphaInterp:    emitter.SystemAlphaInterp,
		EmitterAge:           0,                      // 系统年龄从0开始，每帧递增
		EmitterDuration:      emitter.SystemDuration, // 发射器持续时间（用于归一化）
	}

	// Create PositionComponent
	positionComp := &components.PositionComponent{
		X: spawnX,
		Y: spawnY,
	}

	// Add components to particle entity
	ps.EntityManager.AddComponent(particleID, particleComp)
	ps.EntityManager.AddComponent(particleID, positionComp)

	// Add particle to emitter's active list
	emitter.ActiveParticles = append(emitter.ActiveParticles, particleID)

	// DEBUG: 粒子创建日志（每个粒子创建时打印会刷屏，已禁用）
	// log.Printf("[ParticleSystem] 粒子创建完成: ID=%d, 位置=(%.1f, %.1f), 生命周期=%.2fs, Image=%v, 颜色=(%.2f,%.2f,%.2f), 亮度=%.2f",
	// 	particleID, spawnX, spawnY, lifetime, particleImage != nil, red, green, blue, brightness)
}

// applyInterpolation updates particle properties based on keyframe animations.
func (ps *ParticleSystem) applyInterpolation(p *components.ParticleComponent) {
	if p.Lifetime <= 0 {
		return
	}

	// Calculate normalized time (0-1)
	t := p.Age / p.Lifetime

	// Apply alpha keyframes
	if len(p.AlphaKeyframes) > 0 {
		p.Alpha = particlePkg.EvaluateKeyframes(p.AlphaKeyframes, t, p.AlphaInterpolation)
	}

	// Apply scale keyframes
	if len(p.ScaleKeyframes) > 0 {
		p.Scale = particlePkg.EvaluateKeyframes(p.ScaleKeyframes, t, p.ScaleInterpolation)
	}

	// Apply spin keyframes (updates rotation speed)
	if len(p.SpinKeyframes) > 0 {
		p.RotationSpeed = particlePkg.EvaluateKeyframes(p.SpinKeyframes, t, p.SpinInterpolation)
	}

	// Story 7.5: Apply SystemAlpha (ZombieHead 系统级淡出)
	// SystemAlpha 基于发射器年龄，而不是粒子年龄
	if len(p.SystemAlphaKeyframes) > 0 && p.EmitterDuration > 0 {
		// 计算系统时间归一化值（0-1）
		systemT := p.EmitterAge / p.EmitterDuration
		systemAlpha := particlePkg.EvaluateKeyframes(p.SystemAlphaKeyframes, systemT, p.SystemAlphaInterp)
		// 系统透明度乘以粒子自身透明度
		p.Alpha *= systemAlpha
	}
}

// applyFields applies force field effects to a particle.
// Supports Acceleration and Friction field types.
func (ps *ParticleSystem) applyFields(p *components.ParticleComponent, dt float64) {
	if p.Lifetime <= 0 {
		return
	}

	// Calculate normalized time (0-1) for time-based fields
	t := p.Age / p.Lifetime

	for _, field := range p.Fields {
		switch field.FieldType {
		case "Acceleration":
			// Parse acceleration values (may be keyframes or static)
			xMin, xMax, xKeyframes, xInterp := particlePkg.ParseValue(field.X)
			yMin, yMax, yKeyframes, yInterp := particlePkg.ParseValue(field.Y)

			// Calculate acceleration for this frame
			var ax, ay float64
			if len(xKeyframes) > 0 {
				ax = particlePkg.EvaluateKeyframes(xKeyframes, t, xInterp)
			} else {
				ax = particlePkg.RandomInRange(xMin, xMax)
			}
			if len(yKeyframes) > 0 {
				ay = particlePkg.EvaluateKeyframes(yKeyframes, t, yInterp)
			} else {
				ay = particlePkg.RandomInRange(yMin, yMax)
			}

			// Apply acceleration to velocity
			p.VelocityX += ax * dt
			p.VelocityY += ay * dt

		case "Friction":
			// Story 7.4 修复：支持摩擦力的 keyframes 插值
			xMin, xMax, xKeyframes, xInterp := particlePkg.ParseValue(field.X)
			yMin, yMax, yKeyframes, yInterp := particlePkg.ParseValue(field.Y)

			// Calculate friction for this frame
			var frictionX, frictionY float64
			if len(xKeyframes) > 0 {
				frictionX = particlePkg.EvaluateKeyframes(xKeyframes, t, xInterp)
			} else {
				frictionX = particlePkg.RandomInRange(xMin, xMax)
			}
			if len(yKeyframes) > 0 {
				frictionY = particlePkg.EvaluateKeyframes(yKeyframes, t, yInterp)
			} else {
				frictionY = particlePkg.RandomInRange(yMin, yMax)
			}

			// Apply friction (velocity decay)
			p.VelocityX *= (1 - frictionX*dt)
			p.VelocityY *= (1 - frictionY*dt)

			// Additional field types can be added here
			// case "Attractor":
			//     ...
		}
	}
}
