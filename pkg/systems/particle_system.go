package systems

import (
	"math"
	"math/rand"
	"reflect"

	"github.com/decker502/pvz/internal/particle"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
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
	EntityManager *ecs.EntityManager
}

// NewParticleSystem creates a new ParticleSystem instance.
func NewParticleSystem(em *ecs.EntityManager) *ParticleSystem {
	return &ParticleSystem{
		EntityManager: em,
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

		// Update emitter age
		emitter.Age += dt

		// Check system duration (0 = infinite)
		if emitter.SystemDuration > 0 && emitter.Age >= emitter.SystemDuration {
			emitter.Active = false
		}

		// Spawn particles if emitter is active
		// Loop to spawn multiple particles in single frame if needed
		if emitter.Active && emitter.SpawnRate > 0 && emitter.Config != nil {
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

		// Clean up destroyed particles from active list
		ps.cleanupDestroyedParticles(emitter)
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

		// Check if particle has expired
		if particle.Age >= particle.Lifetime {
			ps.EntityManager.DestroyEntity(particleID)
			continue
		}

		// Apply velocity to position
		position.X += particle.VelocityX * dt
		position.Y += particle.VelocityY * dt

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

	// Create new particle entity
	particleID := ps.EntityManager.CreateEntity()

	// Parse initial values from config
	// Particle duration (convert milliseconds to seconds)
	durationMin, durationMax, _, _ := particle.ParseValue(config.ParticleDuration)
	lifetime := particle.RandomInRange(durationMin, durationMax) / 1000.0 // ms to seconds

	// Launch speed and angle
	speedMin, speedMax, _, _ := particle.ParseValue(config.LaunchSpeed)
	angleMin, angleMax, _, _ := particle.ParseValue(config.LaunchAngle)
	speed := particle.RandomInRange(speedMin, speedMax)
	angle := particle.RandomInRange(angleMin, angleMax)

	// Convert angle to radians and calculate velocity components
	angleRad := angle * math.Pi / 180.0
	velocityX := speed * math.Cos(angleRad)
	velocityY := speed * math.Sin(angleRad)

	// Initial rotation and spin speed
	spinAngleMin, spinAngleMax, _, _ := particle.ParseValue(config.ParticleSpinAngle)
	spinSpeedMin, spinSpeedMax, spinKeyframes, spinInterp := particle.ParseValue(config.ParticleSpinSpeed)
	initialRotation := particle.RandomInRange(spinAngleMin, spinAngleMax)
	initialSpinSpeed := particle.RandomInRange(spinSpeedMin, spinSpeedMax)

	// Scale
	scaleMin, scaleMax, scaleKeyframes, scaleInterp := particle.ParseValue(config.ParticleScale)
	initialScale := particle.RandomInRange(scaleMin, scaleMax)
	if initialScale == 0 {
		initialScale = 1.0 // Default scale
	}

	// Alpha (transparency)
	alphaMin, alphaMax, alphaKeyframes, alphaInterp := particle.ParseValue(config.ParticleAlpha)
	initialAlpha := particle.RandomInRange(alphaMin, alphaMax)
	if initialAlpha == 0 && len(alphaKeyframes) == 0 {
		initialAlpha = 1.0 // Default fully opaque
	}

	// Color channels
	redMin, redMax, _, _ := particle.ParseValue(config.ParticleRed)
	greenMin, greenMax, _, _ := particle.ParseValue(config.ParticleGreen)
	blueMin, blueMax, _, _ := particle.ParseValue(config.ParticleBlue)
	red := particle.RandomInRange(redMin, redMax)
	green := particle.RandomInRange(greenMin, greenMax)
	blue := particle.RandomInRange(blueMin, blueMax)
	if red == 0 && green == 0 && blue == 0 {
		red, green, blue = 1.0, 1.0, 1.0 // Default white
	}

	// Brightness
	brightnessMin, brightnessMax, _, _ := particle.ParseValue(config.ParticleBrightness)
	brightness := particle.RandomInRange(brightnessMin, brightnessMax)
	if brightness == 0 {
		brightness = 1.0 // Default brightness
	}

	// Spawn position (emitter position + random offset within EmitterBox)
	spawnX := emitterPos.X
	spawnY := emitterPos.Y
	if emitter.EmitterBoxX > 0 {
		spawnX += rand.Float64()*emitter.EmitterBoxX - emitter.EmitterBoxX/2
	}
	if emitter.EmitterBoxY > 0 {
		spawnY += rand.Float64()*emitter.EmitterBoxY - emitter.EmitterBoxY/2
	}

	// Additive blending
	additive := false
	if config.Additive == "1" {
		additive = true
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

		Image:    nil, // Will be loaded by ResourceManager (Task 6)
		Additive: additive,
		Fields:   config.Fields, // Copy force fields from config
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
		p.Alpha = particle.EvaluateKeyframes(p.AlphaKeyframes, t, p.AlphaInterpolation)
	}

	// Apply scale keyframes
	if len(p.ScaleKeyframes) > 0 {
		p.Scale = particle.EvaluateKeyframes(p.ScaleKeyframes, t, p.ScaleInterpolation)
	}

	// Apply spin keyframes (updates rotation speed)
	if len(p.SpinKeyframes) > 0 {
		p.RotationSpeed = particle.EvaluateKeyframes(p.SpinKeyframes, t, p.SpinInterpolation)
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
			xMin, xMax, xKeyframes, xInterp := particle.ParseValue(field.X)
			yMin, yMax, yKeyframes, yInterp := particle.ParseValue(field.Y)

			// Calculate acceleration for this frame
			var ax, ay float64
			if len(xKeyframes) > 0 {
				ax = particle.EvaluateKeyframes(xKeyframes, t, xInterp)
			} else {
				ax = particle.RandomInRange(xMin, xMax)
			}
			if len(yKeyframes) > 0 {
				ay = particle.EvaluateKeyframes(yKeyframes, t, yInterp)
			} else {
				ay = particle.RandomInRange(yMin, yMax)
			}

			// Apply acceleration to velocity
			p.VelocityX += ax * dt
			p.VelocityY += ay * dt

		case "Friction":
			// Parse friction coefficients
			xMin, xMax, _, _ := particle.ParseValue(field.X)
			yMin, yMax, _, _ := particle.ParseValue(field.Y)

			frictionX := particle.RandomInRange(xMin, xMax)
			frictionY := particle.RandomInRange(yMin, yMax)

			// Apply friction (velocity decay)
			p.VelocityX *= (1 - frictionX*dt)
			p.VelocityY *= (1 - frictionY*dt)

			// Additional field types can be added here
			// case "Attractor":
			//     ...
		}
	}
}
