package systems

import (
	"log"
	"math"
	"math/rand"

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

// GetDynamicSpawnRate 计算当前时刻的 SpawnRate（支持关键帧动画）
// 导出为公开方法以便测试使用
func (ps *ParticleSystem) GetDynamicSpawnRate(emitter *components.EmitterComponent) float64 {
	// 如果有关键帧，使用动态计算
	if len(emitter.SpawnRateKeyframes) > 0 {
		// SpawnRate 的关键帧使用绝对时间（厘秒），而不是归一化时间
		// 例如："50,70 0,90" 表示在 50 厘秒时 SpawnRate=70，在 0 厘秒时 SpawnRate=90
		t := emitter.Age * 100.0 // 转换为厘秒
		value := particlePkg.EvaluateKeyframes(emitter.SpawnRateKeyframes, t, emitter.SpawnRateInterp)
		return value
	}
	// 否则返回静态值
	return emitter.SpawnRate
}

// getDynamicSpawnRate 私有方法（内部使用）
func (ps *ParticleSystem) getDynamicSpawnRate(emitter *components.EmitterComponent) float64 {
	return ps.GetDynamicSpawnRate(emitter)
}

// GetDynamicSpawnMinActive 计算当前时刻的 SpawnMinActive（支持关键帧动画）
// 导出为公开方法以便测试使用
func (ps *ParticleSystem) GetDynamicSpawnMinActive(emitter *components.EmitterComponent) int {
	// 如果有关键帧，使用动态计算
	if len(emitter.SpawnMinActiveKeyframes) > 0 {
		var t float64
		if emitter.SystemDuration > 0 {
			// 有限持续时间 → 使用归一化时间 (0-1)
			t = emitter.Age / emitter.SystemDuration
		} else {
			// 无限持续时间 → 使用绝对时间（秒）
			// Award.xml 格式: "0,1 1,1 4,19.999998" 表示 t=0s值1, t=1s值1, t=4s值20
			t = emitter.Age
		}
		value := particlePkg.EvaluateKeyframes(emitter.SpawnMinActiveKeyframes, t, emitter.SpawnMinActiveInterp)
		return int(value)
	}
	// 否则返回静态值
	return emitter.SpawnMinActive
}

// getDynamicSpawnMinActive 私有方法（内部使用）
func (ps *ParticleSystem) getDynamicSpawnMinActive(emitter *components.EmitterComponent) int {
	return ps.GetDynamicSpawnMinActive(emitter)
}

// GetDynamicSpawnMaxActive 计算当前时刻的 SpawnMaxActive（支持关键帧动画）
// 导出为公开方法以便测试使用
func (ps *ParticleSystem) GetDynamicSpawnMaxActive(emitter *components.EmitterComponent) int {
	// 如果有关键帧，使用动态计算
	if len(emitter.SpawnMaxActiveKeyframes) > 0 {
		var t float64
		if emitter.SystemDuration > 0 {
			t = emitter.Age / emitter.SystemDuration
		} else {
			t = emitter.Age // 绝对时间
		}
		value := particlePkg.EvaluateKeyframes(emitter.SpawnMaxActiveKeyframes, t, emitter.SpawnMaxActiveInterp)
		return int(value)
	}
	// 否则返回静态值
	return emitter.SpawnMaxActive
}

// getDynamicSpawnMaxActive 私有方法（内部使用）
func (ps *ParticleSystem) getDynamicSpawnMaxActive(emitter *components.EmitterComponent) int {
	return ps.GetDynamicSpawnMaxActive(emitter)
}

// GetDynamicSpawnMaxLaunched 计算当前时刻的 SpawnMaxLaunched（支持关键帧动画）
// 导出为公开方法以便测试使用
func (ps *ParticleSystem) GetDynamicSpawnMaxLaunched(emitter *components.EmitterComponent) int {
	// 如果有关键帧，使用动态计算
	if len(emitter.SpawnMaxLaunchedKeyframes) > 0 {
		var t float64
		if emitter.SystemDuration > 0 {
			t = emitter.Age / emitter.SystemDuration
		} else {
			t = emitter.Age // 绝对时间
		}
		value := particlePkg.EvaluateKeyframes(emitter.SpawnMaxLaunchedKeyframes, t, emitter.SpawnMaxLaunchedInterp)
		return int(value)
	}
	// 否则返回静态值
	return emitter.SpawnMaxLaunched
}

// getDynamicSpawnMaxLaunched 私有方法（内部使用）
func (ps *ParticleSystem) getDynamicSpawnMaxLaunched(emitter *components.EmitterComponent) int {
	return ps.GetDynamicSpawnMaxLaunched(emitter)
}

// updateEmitters processes all emitter entities, spawning new particles
// and managing emitter lifecycle.
func (ps *ParticleSystem) updateEmitters(dt float64) {
	// Query all entities with EmitterComponent and PositionComponent
	emitterEntities := ecs.GetEntitiesWith2[
		*components.EmitterComponent,
		*components.PositionComponent,
	](ps.EntityManager)

	// Story 11.4 DEBUG: 跟踪 SodRoll 发射器（前2秒）
	sodRollCount := 0
	for _, emitterID := range emitterEntities {
		emitter, ok := ecs.GetComponent[*components.EmitterComponent](ps.EntityManager, emitterID)
		if ok && emitter.Config != nil && emitter.Config.Image == "IMAGE_DIRTSMALL" {
			sodRollCount++
			if emitter.Age < 2.0 {
				log.Printf("[ParticleSystem] SodRoll 发射器存在: Age=%.3f, Active=%v, TotalLaunched=%d", emitter.Age, emitter.Active, emitter.TotalLaunched)
			}
		}
	}

	for _, emitterID := range emitterEntities {
		// Get emitter component
		emitter, ok := ecs.GetComponent[*components.EmitterComponent](ps.EntityManager, emitterID)
		if !ok {
			continue
		}

		// Get position component
		position, ok := ecs.GetComponent[*components.PositionComponent](ps.EntityManager, emitterID)
		if !ok {
			continue
		}

		// DEBUG: 发射器处理日志（每个发射器每帧都打印会刷屏，已禁用）
		// log.Printf("[ParticleSystem] 处理发射器 ID=%d: Active=%v, Age=%.2f, SpawnRate=%.2f, NextSpawnTime=%.2f",
		// 	emitterID, emitter.Active, emitter.Age, emitter.SpawnRate, emitter.NextSpawnTime)

		// Update emitter age
		emitter.Age += dt

		// 应用 SystemPosition (发射器位置插值)
		// 修复 - SystemPosition 应该是相对于初始位置的偏移，而不是绝对位置
		// 例如：SodRoll.xml 配置 <X>0 740</X><Y>30 0</Y>，发射器从初始位置(228, 320)移动到(228+740, 320+0)
		if len(emitter.SystemPositionXKeyframes) > 0 || len(emitter.SystemPositionYKeyframes) > 0 {
			// 归一化时间 t（0-1），基于发射器年龄和系统持续时间
			t := 0.0
			if emitter.SystemDuration > 0 {
				t = emitter.Age / emitter.SystemDuration
			}

			// Story 11.4 DEBUG: 记录前3帧的位置变化
			oldX, oldY := position.X, position.Y

			// 插值计算 X 位置偏移
			if len(emitter.SystemPositionXKeyframes) > 0 {
				offsetX := particlePkg.EvaluateKeyframes(emitter.SystemPositionXKeyframes, t, emitter.SystemPositionXInterp)
				position.X = emitter.InitialX + offsetX // 初始位置 + 偏移量
			}

			// 插值计算 Y 位置偏移
			if len(emitter.SystemPositionYKeyframes) > 0 {
				offsetY := particlePkg.EvaluateKeyframes(emitter.SystemPositionYKeyframes, t, emitter.SystemPositionYInterp)
				position.Y = emitter.InitialY + offsetY // 初始位置 + 偏移量
			}

			// Story 11.4 DEBUG: 前5帧打印位置变化
			if emitter.Config.Image == "IMAGE_DIRTSMALL" && emitter.TotalLaunched < 10 {
				log.Printf("[ParticleSystem] SystemPosition 应用: t=%.3f, InitialXY=(%.1f,%.1f), 偏移前=(%.1f,%.1f), 偏移后=(%.1f,%.1f)",
					t, emitter.InitialX, emitter.InitialY, oldX, oldY, position.X, position.Y)
			}
		}

		// Check system duration (0 = infinite)
		if emitter.SystemDuration > 0 && emitter.Age >= emitter.SystemDuration {
			// Story 11.4 DEBUG: 记录发射器停止
			if emitter.Config.Image == "IMAGE_DIRTSMALL" {
				log.Printf("[ParticleSystem] 停止 SodRoll 发射器: Age=%.3f >= SystemDuration=%.3f",
					emitter.Age, emitter.SystemDuration)
			}
			emitter.Active = false
		}

		// 修复：在计算 activeCount 之前先清理已删除的粒子
		// 这样 activeCount 才能反映真实的活跃粒子数量
		ps.cleanupDestroyedParticles(emitter)

		// Spawn particles if emitter is active
		if emitter.Active && emitter.Config != nil {
			// 动态计算当前时刻的 Spawn 约束参数（支持关键帧动画）
			spawnRate := ps.getDynamicSpawnRate(emitter)
			spawnMinActive := ps.getDynamicSpawnMinActive(emitter)
			spawnMaxActive := ps.getDynamicSpawnMaxActive(emitter)
			spawnMaxLaunched := ps.getDynamicSpawnMaxLaunched(emitter)

			// Story 11.4 DEBUG: 跟踪 SodRoll 发射器状态
			if emitter.Config.Image == "IMAGE_DIRTSMALL" && emitter.TotalLaunched < 5 {
				log.Printf("[ParticleSystem DEBUG] SodRoll 发射器状态: Active=%v, Age=%.3f, NextSpawnTime=%.3f, TotalLaunched=%d",
					emitter.Active, emitter.Age, emitter.NextSpawnTime, emitter.TotalLaunched)
				log.Printf("[ParticleSystem DEBUG] SpawnRate=%.1f, SpawnMinActive=%d, SpawnMaxActive=%d, SpawnMaxLaunched=%d",
					spawnRate, spawnMinActive, spawnMaxActive, spawnMaxLaunched)
			}

			// 获取当前活跃粒子数量（已清理已删除粒子，准确）
			activeCount := len(emitter.ActiveParticles)

			// SpawnRate=0: 不按时间间隔生成
			// 区分两种模式：
			// 1. 如果 SpawnMaxLaunched=0（未配置），默认等于 SpawnMinActive → 一次性发射
			//    例如：Planting.xml (种植土粒) → 一次性发射 8 个粒子
			// 2. 如果 SpawnMaxLaunched>0，持续补充到 SpawnMinActive 个粒子活跃
			//    例如：Award.xml (奖励动画) → 持续保持粒子数量
			if spawnRate == 0 {
				// 确定最大发射数量
				effectiveMaxLaunched := spawnMaxLaunched
				if effectiveMaxLaunched == 0 {
					// 未配置 SpawnMaxLaunched：默认等于 SpawnMinActive（一次性发射模式）
					effectiveMaxLaunched = spawnMinActive
				}

				// 补充粒子到目标数量（受 SpawnMaxLaunched 限制）
				for activeCount < spawnMinActive && emitter.TotalLaunched < effectiveMaxLaunched {
					// Check spawn constraints
					canSpawn := true
					if spawnMaxActive > 0 && activeCount >= spawnMaxActive {
						canSpawn = false
						break
					}

					if canSpawn {
						ps.spawnParticle(emitterID, emitter, position)
						emitter.TotalLaunched++
						activeCount++ // 更新本地计数
					} else {
						break
					}
				}
			} else if spawnRate > 0 {
				// Continuous spawn mode: spawn particles at regular intervals
				// Story 11.4 DEBUG: 跟踪发射循环
				loopCount := 0
				for emitter.Age >= emitter.NextSpawnTime {
					loopCount++
					// Check spawn constraints (使用动态计算的值)
					canSpawn := true
					if spawnMaxActive > 0 && activeCount >= spawnMaxActive {
						canSpawn = false
						if emitter.Config.Image == "IMAGE_DIRTSMALL" && loopCount == 1 {
							log.Printf("[ParticleSystem DEBUG] SodRoll 无法发射: SpawnMaxActive 限制 (activeCount=%d >= max=%d)", activeCount, spawnMaxActive)
						}
						break // Can't spawn more this frame
					}
					if spawnMaxLaunched > 0 && emitter.TotalLaunched >= spawnMaxLaunched {
						canSpawn = false
						if emitter.Config.Image == "IMAGE_DIRTSMALL" && loopCount == 1 {
							log.Printf("[ParticleSystem DEBUG] SodRoll 无法发射: SpawnMaxLaunched 限制 (launched=%d >= max=%d)", emitter.TotalLaunched, spawnMaxLaunched)
						}
						break // Reached total launch limit
					}

					if canSpawn {
						ps.spawnParticle(emitterID, emitter, position)
						emitter.TotalLaunched++
						activeCount++ // Update local count
					}

					// Update next spawn time (使用动态 SpawnRate)
					emitter.NextSpawnTime += 1.0 / spawnRate

					// Safety check to avoid infinite loop
					if emitter.NextSpawnTime > emitter.Age+10 {
						break
					}
				}

				// Story 11.4 DEBUG: 报告发射情况
				if emitter.Config.Image == "IMAGE_DIRTSMALL" && emitter.TotalLaunched <= 10 && loopCount > 0 {
					log.Printf("[ParticleSystem DEBUG] SodRoll 本帧发射: loopCount=%d, TotalLaunched=%d, NextSpawnTime=%.3f",
						loopCount, emitter.TotalLaunched, emitter.NextSpawnTime)
				}
			}
		}

		// Auto-cleanup emitter entities when finished
		// Delete emitter if it's inactive and has no active particles
		if !emitter.Active && len(emitter.ActiveParticles) == 0 {
			ps.EntityManager.DestroyEntity(emitterID)
			// Note: No need to log here as it's expected behavior
		}
	}
}

// cleanupDestroyedParticles removes dead particle IDs from emitter's active list
func (ps *ParticleSystem) cleanupDestroyedParticles(emitter *components.EmitterComponent) {
	alive := make([]ecs.EntityID, 0, len(emitter.ActiveParticles))

	for _, particleID := range emitter.ActiveParticles {
		// Check if particle still exists
		if ecs.HasComponent[*components.ParticleComponent](ps.EntityManager, particleID) {
			alive = append(alive, particleID)
		}
	}

	emitter.ActiveParticles = alive
}

// updateParticles processes all particle entities, updating their state
// and destroying expired particles.
func (ps *ParticleSystem) updateParticles(dt float64) {
	particleEntities := ecs.GetEntitiesWith2[
		*components.ParticleComponent,
		*components.PositionComponent,
	](ps.EntityManager)

	for _, particleID := range particleEntities {
		// Get particle component
		particle, ok := ecs.GetComponent[*components.ParticleComponent](ps.EntityManager, particleID)
		if !ok {
			continue
		}

		// Get position component
		position, ok := ecs.GetComponent[*components.PositionComponent](ps.EntityManager, particleID)
		if !ok {
			continue
		}

		// Update particle age
		particle.Age += dt

		// Update system age (for SystemAlpha calculation)
		// 修复：EmitterAge 应该是 "发射器创建时的年龄 + 粒子自己的年龄"
		// 而不是独立的计数器（粒子在创建时已经记录了发射器的初始年龄）
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

		// Apply velocity to position (if no Position Field is active)
		// Position Field 直接设置位置，覆盖速度积分
		hasPositionField := false
		for _, field := range particle.Fields {
			if field.FieldType == "Position" {
				hasPositionField = true
				break
			}
		}

		if hasPositionField {
			// Position Field: 使用初始位置 + 偏移量
			position.X = particle.InitialX + particle.PositionOffsetX
			position.Y = particle.InitialY + particle.PositionOffsetY
		} else {
			// 正常模式：基于速度积分更新位置
			position.X += particle.VelocityX * dt
			position.Y += particle.VelocityY * dt
		}

		// Ground collision detection (ZombieHead 弹跳效果)
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

				// DEBUG: 碰撞弹跳日志
				// oldVelocityY := particle.VelocityY
				// log.Printf("[碰撞] Age=%.2fs, Y=%.1f, 旧速度=%.1f, 反弹系数=%.2f",
				// particle.Age, position.Y, oldVelocityY, reflectY)

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

	// 如果 ParticleDuration 未配置（为 0），使用 SystemDuration 作为默认值
	// 这样粒子至少能存活到发射器结束，避免生命周期为 0 导致粒子立即销毁
	if lifetime == 0 && emitter.SystemDuration > 0 {
		lifetime = emitter.SystemDuration
		log.Printf("[ParticleSystem] 警告：ParticleDuration 未配置，使用 SystemDuration 作为默认值: %.2fs", lifetime)
	}

	// Launch speed and angle
	speedMin, speedMax, _, _ := particlePkg.ParseValue(config.LaunchSpeed)
	angleMin, angleMax, _, _ := particlePkg.ParseValue(config.LaunchAngle)

	// DEBUG: 输出解析结果（帮助诊断 LaunchAngle 是否被正确应用）
	log.Printf("[LaunchAngle] 配置='%s' → 解析: min=%.1f, max=%.1f",
		config.LaunchAngle, angleMin, angleMax)

	speed := particlePkg.RandomInRange(speedMin, speedMax)
	angle := particlePkg.RandomInRange(angleMin, angleMax)

	// Story 7.4 修复：如果 LaunchAngle 未定义且发射器类型是 Circle，使用随机 360° 角度
	if angleMin == 0 && angleMax == 0 && config.LaunchAngle == "" && config.EmitterType == "Circle" {
		angle = rand.Float64() * 360.0 // 0-360 度随机
		log.Printf("[LaunchAngle] 检测到 Circle 类型，使用360°随机: %.1f°", angle)
	}

	// DEBUG: 输出最终使用的角度
	log.Printf("[LaunchAngle] 最终角度=%.1f° (范围 [%.1f, %.1f])", angle, angleMin, angleMax)

	// Apply emitter's angle offset (e.g., 180° to flip direction)
	// This keeps particle system decoupled from business logic (zombie direction)
	// Business logic (BehaviorSystem) calculates offset based on entity direction
	angle += emitter.AngleOffset

	// Story 10.4 修正：PvZ 使用数学标准坐标系（Y轴向上）
	// 角度定义（基于数学坐标系）：
	//   0° = 向右，90° = 向上，180° = 向左，270° = 向下
	// 证据：SodRoll.md 明确说 "90度（正上方）到180度（正左方）"
	//
	// 转换到屏幕坐标（Y轴向下）：
	//   - velocityX = speed * cos(angle) （无需转换）
	//   - velocityY = -speed * sin(angle) （取反！因为屏幕Y轴向下）
	//
	// 验证示例：
	//   - SodRoll [90-180°]：90°→上，135°→左上，180°→左 ✓ "向上和向左"
	//   - Planting [110-250°]：110°→左上，180°→左，250°→左下 ✓ "向上和两侧"
	//   - ZombieHead [150-185°]：150°→左上，185°→左 ✓ "向左后方"

	// Convert angle to radians and calculate velocity components
	// LaunchSpeed is in pixels/second, use directly (no conversion needed)
	angleRad := angle * math.Pi / 180.0
	velocityX := speed * math.Cos(angleRad)
	velocityY := -speed * math.Sin(angleRad) // Y轴取反：数学坐标系→屏幕坐标系

	// Initial rotation and spin speed
	spinAngleMin, spinAngleMax, _, _ := particlePkg.ParseValue(config.ParticleSpinAngle)
	spinSpeedMin, spinSpeedMax, spinKeyframes, spinInterp := particlePkg.ParseValue(config.ParticleSpinSpeed)
	initialRotation := particlePkg.RandomInRange(spinAngleMin, spinAngleMax)
	initialSpinSpeed := particlePkg.RandomInRange(spinSpeedMin, spinSpeedMax)
	// 如果未提供 SpinAngle 且配置了 RandomLaunchSpin，则随机初始朝向（0-360 度）
	if (spinAngleMin == 0 && spinAngleMax == 0) && config.RandomLaunchSpin == "1" {
		initialRotation = rand.Float64() * 360.0
	}

	// 应用发射器的粒子旋转覆盖（如果设置）
	// 用于教学箭头等需要特定方向的粒子效果
	if emitter.ParticleRotationOverride != 0 {
		initialRotation = emitter.ParticleRotationOverride
		log.Printf("[ParticleSystem] Applied rotation override: %.1f°", initialRotation)
	}

	// Story 7.5 修复：对于"范围+关键帧"格式，需要将初始值添加到关键帧开头
	// 例如：ParticleSpinSpeed="[-720 720] 0,39.999996" 返回 min=-720, max=720, keyframes=[{0.4, 0}]
	// 需要添加初始关键帧：[{0, initialSpinSpeed}, {0.4, 0}]
	// ParticleSpinSpeed is in degrees/second, use directly (no conversion needed)
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

	// DEBUG: 输出解析结果（帮助诊断 ParticleScale 是否被正确应用）
	log.Printf("[ParticleScale] 配置='%s' → 解析: min=%.2f, max=%.2f, initialScale=%.2f",
		config.ParticleScale, scaleMin, scaleMax, initialScale)

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
	// 应用发射器偏移量（EmitterOffsetX/Y）
	spawnX := emitterPos.X + emitter.EmitterOffsetX
	spawnY := emitterPos.Y + emitter.EmitterOffsetY

	// DEBUG: 输出基础生成位置
	if emitter.TotalLaunched < 3 {
		log.Printf("[DEBUG SpawnBase] 粒子#%d: emitterPos=(%.1f, %.1f), offset=(%.1f, %.1f), spawnBase=(%.1f, %.1f)",
			emitter.TotalLaunched+1, emitterPos.X, emitterPos.Y,
			emitter.EmitterOffsetX, emitter.EmitterOffsetY, spawnX, spawnY)
	}

	// 动态计算 EmitterBox（支持关键帧插值）
	// 修复：正确处理非对称范围和负数偏移
	// 例如：SodRoll.xml 的 EmitterBoxY="[-130 0] [-100 0]"
	//   → minY 从 -130 插值到 -100
	//   → widthY 从 130 插值到 100
	//   → spawnY = minY + rand() * widthY
	dynamicEmitterBoxXMin := emitter.EmitterBoxXMin
	dynamicEmitterBoxYMin := emitter.EmitterBoxYMin
	dynamicEmitterBoxXWidth := emitter.EmitterBoxX
	dynamicEmitterBoxYWidth := emitter.EmitterBoxY

	if len(emitter.EmitterBoxXKeyframes) > 0 || len(emitter.EmitterBoxYKeyframes) > 0 ||
		len(emitter.EmitterBoxXMinKeyframes) > 0 || len(emitter.EmitterBoxYMinKeyframes) > 0 {
		// 归一化时间 t（0-1），基于发射器年龄和系统持续时间
		t := 0.0
		if emitter.SystemDuration > 0 {
			t = emitter.Age / emitter.SystemDuration
		}

		// 插值计算 EmitterBoxX 的最小值和宽度
		if len(emitter.EmitterBoxXMinKeyframes) > 0 {
			dynamicEmitterBoxXMin = particlePkg.EvaluateKeyframes(emitter.EmitterBoxXMinKeyframes, t, emitter.EmitterBoxXInterp)
		}
		if len(emitter.EmitterBoxXKeyframes) > 0 {
			dynamicEmitterBoxXWidth = particlePkg.EvaluateKeyframes(emitter.EmitterBoxXKeyframes, t, emitter.EmitterBoxXInterp)
		}

		// 插值计算 EmitterBoxY 的最小值和宽度
		if len(emitter.EmitterBoxYMinKeyframes) > 0 {
			dynamicEmitterBoxYMin = particlePkg.EvaluateKeyframes(emitter.EmitterBoxYMinKeyframes, t, emitter.EmitterBoxYInterp)
		}
		if len(emitter.EmitterBoxYKeyframes) > 0 {
			dynamicEmitterBoxYWidth = particlePkg.EvaluateKeyframes(emitter.EmitterBoxYKeyframes, t, emitter.EmitterBoxYInterp)
		}

		// DEBUG: 输出前3次的插值结果
		if emitter.TotalLaunched < 3 {
			log.Printf("[EmitterBox] t=%.3f, Y: min=%.1f, width=%.1f, 范围=[%.1f, %.1f]",
				t, dynamicEmitterBoxYMin, dynamicEmitterBoxYWidth,
				dynamicEmitterBoxYMin, dynamicEmitterBoxYMin+dynamicEmitterBoxYWidth)
		}
	}

	if emitter.EmitterRadiusMax > 0 {
		// 修复：EmitterRadius 支持范围格式 [min max]
		// 每个粒子随机选择 min-max 之间的半径
		// 例如：Planting.xml 的 "[0 10]" → 粒子在半径 0-10 之间随机分布
		radius := particlePkg.RandomInRange(emitter.EmitterRadiusMin, emitter.EmitterRadiusMax)

		// 均匀分布在圆形区域内：半径使用 sqrt 随机，角度均匀
		// 使用 sqrt 确保粒子在圆内均匀分布（而不是聚集在中心）
		r := math.Sqrt(rand.Float64()) * radius
		ang := rand.Float64() * 2 * math.Pi
		offsetX := r * math.Cos(ang)
		offsetY := r * math.Sin(ang)

		// DEBUG: 输出前3个粒子的圆形分布参数
		if emitter.TotalLaunched < 3 {
			log.Printf("[DEBUG EmitterRadius] 粒子#%d: radius=%.2f, r=%.2f, angle=%.2f°, offset=(%.2f, %.2f)",
				emitter.TotalLaunched+1, radius, r, ang*180/math.Pi, offsetX, offsetY)
		}

		spawnX += offsetX
		spawnY += offsetY
	} else {
		// 修复：使用非对称范围生成
		// 对于范围 [min, max]，使用 min + rand() * (max - min)
		// 而不是对称的 center ± width/2
		if dynamicEmitterBoxXWidth > 0 {
			spawnX += dynamicEmitterBoxXMin + rand.Float64()*dynamicEmitterBoxXWidth
		}
		if dynamicEmitterBoxYWidth > 0 {
			spawnY += dynamicEmitterBoxYMin + rand.Float64()*dynamicEmitterBoxYWidth
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
	imageRows := 1   // 默认单行
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

			// BUG修复：从资源配置读取精灵图的 rows 信息
			// 例如：IMAGE_DIRTSMALL 配置为 cols=8, rows=2
			// 这样才能正确渲染 40x40 的土粒，而不是拉伸为 40x80
			if cols, rows, ok := ps.ResourceManager.GetImageMetadata(config.Image); ok {
				if rows > 0 {
					imageRows = rows
				}
				// 验证 ImageFrames 与配置的 cols 是否一致
				if cols > 0 && imageFrames != cols {
					log.Printf("[ParticleSystem] 警告：ImageFrames(%d) 与资源配置 cols(%d) 不一致，使用配置值", imageFrames, cols)
					imageFrames = cols
				}
			}

			// 如果是多帧精灵图，选择随机帧
			if imageFrames > 1 {
				frameNum = rand.Intn(imageFrames)
			}
		}
	}

	// Parse collision properties (ZombieHead 弹跳效果)
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
		// CollisionSpin is in degrees/second, use directly (no conversion needed)
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

	// DEBUG: 粒子创建日志
	log.Printf("[DEBUG] 创建粒子: pos=(%.1f,%.1f), velocity=(%.1f,%.1f), angle=%.1f°, speed=%.1f, scale=%.2f, groundY=%.1f, image=%v",
		spawnX, spawnY, velocityX, velocityY, angle, speed, initialScale, groundY, particleImage != nil)

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

		Image:       particleImage, // Loaded from ResourceManager
		ImageFrames: imageFrames,   // 精灵图帧数（列数）
		ImageRows:   imageRows,     // BUG修复：精灵图行数（用于正确计算单帧高度）
		FrameNum:    frameNum,      // 当前帧编号
		Additive:    additive,
		Fields:      config.Fields, // Copy force fields from config

		// Collision properties
		CollisionReflectX:     collisionReflectX,
		CollisionReflectY:     collisionReflectY,
		CollisionReflectCurve: collisionReflectCurve,
		CollisionSpinMin:      collisionSpinMin,
		CollisionSpinMax:      collisionSpinMax,
		CollisionSpinCurve:    collisionSpinCurve,
		GroundY:               groundY,

		// System-level alpha (ZombieHead 系统淡出)
		SystemAlphaKeyframes: emitter.SystemAlphaKeyframes,
		SystemAlphaInterp:    emitter.SystemAlphaInterp,
		EmitterAge:           emitter.Age,            // 使用发射器的当前年龄（修复：粒子应该基于发射器年龄，而不是自己的独立计数器）
		EmitterDuration:      emitter.SystemDuration, // 发射器持续时间（用于归一化）

		// Position Field 支持：保存初始位置
		InitialX:        spawnX,
		InitialY:        spawnY,
		PositionOffsetX: 0,
		PositionOffsetY: 0,
	}

	// Create PositionComponent
	positionComp := &components.PositionComponent{
		X: spawnX,
		Y: spawnY,
	}

	// Add components to particle entity
	ps.EntityManager.AddComponent(particleID, particleComp)
	ps.EntityManager.AddComponent(particleID, positionComp)

	// 如果发射器是UI元素（有UIComponent），则粒子也标记为UI粒子
	// 这样渲染时不会减去cameraX（教学箭头等UI粒子）
	if uiComp, hasUI := ecs.GetComponent[*components.UIComponent](ps.EntityManager, emitterID); hasUI {
		ps.EntityManager.AddComponent(particleID, uiComp) // 复制UIComponent
	}

	// Add particle to emitter's active list
	emitter.ActiveParticles = append(emitter.ActiveParticles, particleID)

	// DEBUG: 临时调试 - 打印 Position Field 的内容
	for _, field := range config.Fields {
		if field.FieldType == "Position" {
			log.Printf("[DEBUG] 粒子创建 - Position Field: X='%s', Y='%s'", field.X, field.Y)
		}
	}

	// DEBUG: 粒子创建日志（每个粒子创建时打印会刷屏，已禁用）
	// log.Printf("[ParticleSystem] 粒子创建完成: ID=%d, 位置=(%.1f, %.1f), 生命周期=%.2fs, Image=%v, Alpha=%.2f, Scale=%.2f, 颜色=(%.2f,%.2f,%.2f), 亮度=%.2f",
	// 	particleID, spawnX, spawnY, lifetime, particleImage != nil, initialAlpha, initialScale, red, green, blue, brightness)
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

	// Apply SystemAlpha (ZombieHead 系统级淡出)
	// SystemAlpha 基于发射器年龄，而不是粒子年龄
	if len(p.SystemAlphaKeyframes) > 0 && p.EmitterDuration > 0 {
		// 计算系统时间归一化值（0-1）
		systemT := p.EmitterAge / p.EmitterDuration
		systemAlpha := particlePkg.EvaluateKeyframes(p.SystemAlphaKeyframes, systemT, p.SystemAlphaInterp)

		// DEBUG: SystemAlpha 调试日志（临时启用）
		if p.Age < 0.05 || int(p.Age*20)%10 == 0 { // 每0.5秒打印一次
			log.Printf("[SystemAlpha] EmitterAge=%.2fs, EmitterDuration=%.2fs, systemT=%.2f, systemAlpha=%.3f, particleAlpha=%.3f → final=%.3f",
				p.EmitterAge, p.EmitterDuration, systemT, systemAlpha, p.Alpha, p.Alpha*systemAlpha)
		}

		// 系统透明度乘以粒子自身透明度
		p.Alpha *= systemAlpha
	}
}

// applyFields applies force field effects to a particle.
// Supports Acceleration and Friction field types.
//
// PopCap Mixed Unit System (基于原版 PvZ 游戏观察)：
// - LaunchSpeed: pixels/second (标准速度单位，直接使用)
// - Acceleration: velocity increment per tick (每 tick 的速度增量)
//   - 1 tick = 0.01 seconds (原版固定时间步长)
//   - 需要除以 0.01 转换为标准加速度 (pixels/second²)
//
// 原理：
// 原版游戏每 0.01 秒更新一次物理，配置值是每次更新的增量。
// 例如：Acceleration Y=17 表示每 tick（0.01s）速度增加 17 px/s
// 转换为标准加速度：17 / 0.01 = 1700 px/s²
// 这样在每帧更新中：velocity += 1700 × dt
func (ps *ParticleSystem) applyFields(p *components.ParticleComponent, dt float64) {
	if p.Lifetime <= 0 {
		return
	}

	// PopCap's original fixed physics time step
	const OriginalTimeStep = 0.01 // 1 centisecond = 0.01 seconds

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

			// Unit conversion: Config values are "velocity increment per tick (0.01s)"
			// Convert to standard acceleration (pixels/second²)
			// 例如：Acceleration=17 表示每 tick 速度增加 17 px/s
			//       → 标准加速度 = 17 / 0.01 = 1700 px/s²
			ax = ax / OriginalTimeStep
			ay = ay / OriginalTimeStep

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

			// Story 10.4 修正：摩擦力单位转换（与加速度一致）
			// 配置值基于原版引擎的固定时间步长（0.01秒/tick）
			// 例如：Friction=0.08 表示每 tick 衰减 8% 的速度
			//       转换为标准系数：0.08 / 0.01 = 8.0 (每秒衰减800%)
			//
			// 验证：Planting 配置 Friction=0.08
			//   - 转换前：0.3秒内仅衰减2.5% ❌ 不符合"非常迅速地减速"
			//   - 转换后：0.3秒内衰减93.4% ✅ 符合文档描述
			frictionX = frictionX / OriginalTimeStep
			frictionY = frictionY / OriginalTimeStep

			// Apply friction (velocity decay)
			p.VelocityX *= (1 - frictionX*dt)
			p.VelocityY *= (1 - frictionY*dt)

		case "Position":
			// Position Field: 直接设置粒子相对于初始位置的偏移量
			// 这是一个动画路径，覆盖基于速度的位置更新
			//
			// 例如: SeedPacket 箭头
			//   <Y>0 Linear 10,50 Linear 0</Y>
			//   - t=0.0 (开始): offsetY = 0
			//   - t=0.5 (中点): offsetY = 10
			//   - t=1.0 (结束): offsetY = 0
			//   效果：箭头向下移动10像素，再回到原位

			// Parse position values (must use keyframes for animation)
			_, _, xKeyframes, xInterp := particlePkg.ParseValue(field.X)
			_, _, yKeyframes, yInterp := particlePkg.ParseValue(field.Y)

			// Calculate position offset for this frame
			if len(xKeyframes) > 0 {
				p.PositionOffsetX = particlePkg.EvaluateKeyframes(xKeyframes, t, xInterp)
			} else if field.X != "" {
				// 静态值（无动画）
				xMin, xMax, _, _ := particlePkg.ParseValue(field.X)
				p.PositionOffsetX = particlePkg.RandomInRange(xMin, xMax)
			}

			if len(yKeyframes) > 0 {
				p.PositionOffsetY = particlePkg.EvaluateKeyframes(yKeyframes, t, yInterp)
			} else if field.Y != "" {
				// 静态值（无动画）
				yMin, yMax, _, _ := particlePkg.ParseValue(field.Y)
				p.PositionOffsetY = particlePkg.RandomInRange(yMin, yMax)
			}

			// Additional field types can be added here
			// case "Attractor":
			//     ...
		}
	}
}

// IsParticleEffectCompleted 检查指定的粒子特效是否播放完成
//
// 判断标准（优雅且系统化）：
//
//	发射器年龄 >= MAX(关键帧最后时间点, SystemDuration) + MAX(ParticleDuration)
//
// 原理：
//   - 发射器在最后时间点停止生成新粒子
//   - 最后生成的粒子在 ParticleDuration 后消失
//   - 所以总时长 = 生成结束时间 + 粒子寿命
//
// 参数:
//   - emitterGroupID: 粒子特效的主发射器实体ID（由 CreateParticleEffect 返回）
//
// 返回:
//   - true: 粒子特效已完成（所有粒子应该已消失）
//   - false: 粒子特效仍在播放
func (ps *ParticleSystem) IsParticleEffectCompleted(emitterGroupID ecs.EntityID) bool {
	if emitterGroupID == 0 {
		return false // 无效的实体ID
	}

	// 获取主发射器的位置组件（用于识别同组发射器）
	mainPos, ok := ecs.GetComponent[*components.PositionComponent](ps.EntityManager, emitterGroupID)
	if !ok {
		return false // 主发射器已被删除
	}

	// 查询所有发射器实体
	allEmitters := ecs.GetEntitiesWith2[
		*components.EmitterComponent,
		*components.PositionComponent,
	](ps.EntityManager)

	// 计算粒子特效的预期完成时间
	var maxCompletionTime float64 = 0
	emitterCount := 0

	for _, emitterID := range allEmitters {
		emitterPos, _ := ecs.GetComponent[*components.PositionComponent](ps.EntityManager, emitterID)

		// 使用指针比较识别同组发射器（CreateParticleEffect 创建的发射器共享同一个 PositionComponent）
		if emitterPos == mainPos {
			emitterComp, ok := ecs.GetComponent[*components.EmitterComponent](ps.EntityManager, emitterID)
			if !ok {
				continue
			}

			emitterCount++

			// 计算该发射器的完成时间
			// = 停止生成新粒子的时间 + 粒子寿命
			var stopSpawningTime float64

			// 1. 如果有 SystemDuration，以它为准
			if emitterComp.SystemDuration > 0 {
				stopSpawningTime = emitterComp.SystemDuration

				// 关键帧时间是相对于 SystemDuration 的百分比
				// 如果关键帧的最后时间点超过了SystemDuration，需要取最大值
				keyframeEndTime := ps.getLastKeyframeTime(emitterComp) * emitterComp.SystemDuration
				if keyframeEndTime > stopSpawningTime {
					stopSpawningTime = keyframeEndTime
				}
			} else {
				// 2. SystemDuration=0 表示由关键帧控制生成
				// 这种情况下，关键帧时间是绝对时间（秒），而不是百分比
				// 直接使用关键帧的最后时间点
				stopSpawningTime = ps.getLastKeyframeTime(emitterComp)
			}

			// 3. 获取粒子寿命（从配置中读取）
			particleLifetime := ps.getParticleLifetime(emitterComp)

			// 4. 该发射器的完成时间 = 停止生成时间 + 粒子寿命
			completionTime := stopSpawningTime + particleLifetime

			// 调试日志：显示每个发射器的计算结果
			emitterName := "Unknown"
			if emitterComp.Config != nil {
				emitterName = emitterComp.Config.Name
			}
			log.Printf("[ParticleSystem] 发射器 '%s': stopSpawningTime=%.2fs, particleLifetime=%.2fs, completionTime=%.2fs",
				emitterName, stopSpawningTime, particleLifetime, completionTime)

			// 5. 取所有发射器中的最大值
			if completionTime > maxCompletionTime {
				maxCompletionTime = completionTime
			}
		}
	}

	// 检查是否所有发射器都已达到完成时间
	// 使用主发射器的年龄作为参考
	mainEmitter, ok := ecs.GetComponent[*components.EmitterComponent](ps.EntityManager, emitterGroupID)
	if !ok {
		return false
	}

	completed := mainEmitter.Age >= maxCompletionTime

	// 调试日志：显示最终结果
	log.Printf("[ParticleSystem] 粒子特效完成检测: emitterCount=%d, maxCompletionTime=%.2fs, currentAge=%.2fs, completed=%v",
		emitterCount, maxCompletionTime, mainEmitter.Age, completed)

	return completed
}

// getLastKeyframeTime 获取发射器关键帧动画的最后时间点
// 返回值：
//   - 如果有 SystemDuration：返回归一化时间 (0-1)，调用者需要乘以 SystemDuration
//   - 如果无 SystemDuration：返回绝对时间（秒），关键帧的 Time 字段是厘秒，需要转换
func (ps *ParticleSystem) getLastKeyframeTime(emitter *components.EmitterComponent) float64 {
	var maxTime float64 = 0

	// 检查所有关键帧数组，找到最大时间
	if len(emitter.SpawnMinActiveKeyframes) > 0 {
		lastKF := emitter.SpawnMinActiveKeyframes[len(emitter.SpawnMinActiveKeyframes)-1]
		if lastKF.Time > maxTime {
			maxTime = lastKF.Time
		}
	}

	if len(emitter.SpawnMaxActiveKeyframes) > 0 {
		lastKF := emitter.SpawnMaxActiveKeyframes[len(emitter.SpawnMaxActiveKeyframes)-1]
		if lastKF.Time > maxTime {
			maxTime = lastKF.Time
		}
	}

	if len(emitter.SpawnMaxLaunchedKeyframes) > 0 {
		lastKF := emitter.SpawnMaxLaunchedKeyframes[len(emitter.SpawnMaxLaunchedKeyframes)-1]
		if lastKF.Time > maxTime {
			maxTime = lastKF.Time
		}
	}

	// 关键帧时间的单位取决于是否有 SystemDuration：
	// - 有 SystemDuration: Time 是百分比（需要除以100）→ 返回归一化值
	// - 无 SystemDuration: Time 是厘秒 → 返回秒
	if emitter.SystemDuration > 0 {
		// 百分比模式：返回归一化时间（0-1）
		return maxTime / 100.0
	} else {
		// 绝对时间模式：厘秒转秒
		return maxTime / 100.0
	}
}

// getParticleLifetime 获取粒子的寿命（从配置中读取 ParticleDuration 字段）
// ParticleDuration 单位是厘秒（centiseconds），需要转换为秒
func (ps *ParticleSystem) getParticleLifetime(emitter *components.EmitterComponent) float64 {
	if emitter.Config == nil {
		return 0
	}

	// ParticleDuration 是 EmitterConfig 的顶级字段，是一个 string 类型
	// 需要使用 ParseValue 解析
	if emitter.Config.ParticleDuration == "" {
		return 0 // 没有配置 ParticleDuration
	}

	// 解析 ParticleDuration（可能是单值、范围或关键帧）
	minVal, maxVal, keyframes, _ := particlePkg.ParseValue(emitter.Config.ParticleDuration)

	// 如果有关键帧，取最后一个关键帧的值
	if len(keyframes) > 0 {
		lastKF := keyframes[len(keyframes)-1]
		return lastKF.Value / 100.0 // 厘秒 → 秒
	}

	// 否则，取静态值（范围的最大值或单值）
	lifetime := maxVal
	if maxVal == 0 {
		lifetime = minVal
	}

	return lifetime / 100.0 // 厘秒 → 秒
}
