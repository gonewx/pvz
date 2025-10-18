package components

import (
	"github.com/decker502/pvz/internal/particle"
	"github.com/decker502/pvz/pkg/ecs"
)

// EmitterComponent represents a particle emitter that spawns and manages particles.
// Each emitter uses a configuration (loaded from XML) to determine how particles
// are created, their initial properties, and the emitter's lifecycle.
//
// The emitter tracks its own age, spawn timing, and active particles. The
// ParticleSystem processes emitters each frame to spawn new particles and manage
// their lifecycle.
//
// This is a pure data component following ECS principles - it contains no methods.
type EmitterComponent struct {
	// Configuration reference (来自 Story 7.1)
	Config *particle.EmitterConfig // Pointer to the loaded particle configuration

	// Emitter state (发射器状态)
	Active bool    // Whether the emitter is currently spawning particles
	Age    float64 // Time the emitter has been running (seconds)

	// System properties (系统级设置)
	SystemDuration float64 // Total duration before emitter stops (seconds, 0 = infinite)

	// Spawn timing (发射时机)
	NextSpawnTime float64 // Time (in emitter age) when next particle should spawn

	// Particle tracking (粒子追踪)
	ActiveParticles []ecs.EntityID // List of currently active particle entity IDs
	TotalLaunched   int            // Total number of particles spawned so far

	// Parsed spawn parameters (from Config, parsed at creation time)
	// These values are extracted from the string-based Config fields for performance

	// SpawnRate: Particles spawned per second
	SpawnRate float64

	// SpawnMinActive: Minimum number of particles to keep active
	SpawnMinActive int

	// SpawnMaxActive: Maximum number of particles allowed to be active simultaneously
	SpawnMaxActive int

	// SpawnMaxLaunched: Maximum total particles to launch (0 = unlimited)
	SpawnMaxLaunched int

	// Emitter area properties (发射区域)
	EmitterBoxX   float64 // Horizontal spawn area size (random position within box)
	EmitterBoxY   float64 // Vertical spawn area size
	EmitterRadius float64 // Circular spawn radius (alternative to box)

	// System-level properties (Story 7.5: ZombieHead 系统透明度)
	SystemAlphaKeyframes []particle.Keyframe // 系统级透明度关键帧（影响所有粒子）
	SystemAlphaInterp    string              // 插值模式

	// Angle offset (角度偏移)
	AngleOffset float64 // LaunchAngle 的偏移量（度），例如 180° 用于翻转方向
}
