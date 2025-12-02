package systems

import (
	"log"
	"math"

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
			if particle.ParticleLoops {
				// Loop mode: reset particle age to continue animation
				particle.Age = 0
				// DEBUG: 粒子循环重置日志（已禁用）
				// log.Printf("[ParticleSystem] 粒子循环重置: ID=%d, Lifetime=%.2f", particleID, particle.Lifetime)
			} else {
				// Normal mode: destroy particle
				ps.EntityManager.DestroyEntity(particleID)
				// DEBUG: 销毁粒子日志（每个粒子结束时打印会刷屏，已禁用）
				// log.Printf("[ParticleSystem] 销毁过期粒子: ID=%d, Age=%.2f, Lifetime=%.2f", particleID, particle.Age, particle.Lifetime)
				continue
			}
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

		// Update frame animation (Animated 字段支持)
		// 当 Animated=true 且有多帧时，根据时间更新当前帧
		if particle.Animated && particle.ImageFrames > 1 {
			particle.FrameTime += dt

			// 计算每帧持续时间
			var frameDuration float64
			if particle.AnimationRate > 0 {
				// 使用配置的帧率
				frameDuration = 1.0 / particle.AnimationRate
			} else {
				// 自动计算：在粒子生命周期内播放完所有帧
				frameDuration = particle.Lifetime / float64(particle.ImageFrames)
			}

			// 如果超过当前帧持续时间，切换到下一帧
			if frameDuration > 0 && particle.FrameTime >= frameDuration {
				particle.FrameTime -= frameDuration
				particle.FrameNum = (particle.FrameNum + 1) % particle.ImageFrames
			}
		}

		// Apply force fields (acceleration, friction, etc.)
		ps.applyFields(particle, dt)

		// Apply interpolation (alpha, scale, spin animations)
		ps.applyInterpolation(particle)
	}
}

// spawnParticle creates a new particle entity based on emitter configuration.
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

	// Apply color keyframes (颜色渐变，如 Powie 爆炸从橙色变红色)
	if len(p.RedKeyframes) > 0 {
		p.Red = particlePkg.EvaluateKeyframes(p.RedKeyframes, t, p.RedInterp)
	}
	if len(p.GreenKeyframes) > 0 {
		p.Green = particlePkg.EvaluateKeyframes(p.GreenKeyframes, t, p.GreenInterp)
	}
	if len(p.BlueKeyframes) > 0 {
		p.Blue = particlePkg.EvaluateKeyframes(p.BlueKeyframes, t, p.BlueInterp)
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
			//
			// 优化：使用预解析的关键帧（在 spawnParticle 时解析一次）
			// 支持特殊格式如 "0 [-40 10]"：每个粒子有独立的随机目标值

			// 使用预解析的 Position Field 关键帧
			if len(p.PositionFieldXKeyframes) > 0 {
				p.PositionOffsetX = particlePkg.EvaluateKeyframes(p.PositionFieldXKeyframes, t, p.PositionFieldXInterp)
			}

			if len(p.PositionFieldYKeyframes) > 0 {
				p.PositionOffsetY = particlePkg.EvaluateKeyframes(p.PositionFieldYKeyframes, t, p.PositionFieldYInterp)
			}

		case "Circle":
			// Circle 力场：让粒子围绕发射点做圆周运动
			// 通过给速度添加垂直分量实现旋转效果
			//
			// 原理：对于圆周运动，需要将速度向量旋转一个角度
			// 每帧旋转的角度 = 角速度 * dt
			//
			// 角速度已在 spawnParticle 时解析并存储在 p.CircleAngularVelocity
			if p.CircleAngularVelocity != 0 {
				// 角速度转换：度/秒 → 弧度/帧
				angularVelocityRad := p.CircleAngularVelocity * math.Pi / 180.0

				// 计算旋转角度
				rotationAngle := angularVelocityRad * dt

				// 旋转速度向量
				// 新 Vx = Vx * cos(θ) - Vy * sin(θ)
				// 新 Vy = Vx * sin(θ) + Vy * cos(θ)
				cosA := math.Cos(rotationAngle)
				sinA := math.Sin(rotationAngle)
				newVx := p.VelocityX*cosA - p.VelocityY*sinA
				newVy := p.VelocityX*sinA + p.VelocityY*cosA
				p.VelocityX = newVx
				p.VelocityY = newVy
			}

		case "Away":
			// Away 力场：让粒子远离发射点移动
			// 给速度添加从初始位置指向当前位置的径向分量
			//
			// 径向速度已在 spawnParticle 时解析并存储在 p.AwaySpeed
			if p.AwaySpeed != 0 {
				// 获取当前相对于初始位置的位移
				// 注意：需要从当前速度反推位置偏移，或直接添加径向加速度
				//
				// 简化实现：Away 效果是让粒子向外扩散
				// 如果粒子有速度，增强其径向分量
				// 如果粒子静止，给它一个随机向外的速度
				currentSpeed := math.Sqrt(p.VelocityX*p.VelocityX + p.VelocityY*p.VelocityY)
				if currentSpeed > 0.1 {
					// 沿当前运动方向添加速度
					dirX := p.VelocityX / currentSpeed
					dirY := p.VelocityY / currentSpeed
					// 转换单位：配置值基于 0.01s/tick
					awayAccel := p.AwaySpeed / OriginalTimeStep
					p.VelocityX += dirX * awayAccel * dt
					p.VelocityY += dirY * awayAccel * dt
				}
			}
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
