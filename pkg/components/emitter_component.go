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
	// SpawnRate keyframes (for dynamic spawn rate over time)
	SpawnRateKeyframes []particle.Keyframe
	SpawnRateInterp    string

	// SpawnMinActive: Minimum number of particles to keep active
	SpawnMinActive int
	// SpawnMinActive keyframes (for dynamic particle count over time)
	SpawnMinActiveKeyframes []particle.Keyframe
	SpawnMinActiveInterp    string

	// SpawnMaxActive: Maximum number of particles allowed to be active simultaneously
	SpawnMaxActive int
	// SpawnMaxActive keyframes (for dynamic particle count over time)
	SpawnMaxActiveKeyframes []particle.Keyframe
	SpawnMaxActiveInterp    string

	// SpawnMaxLaunched: Maximum total particles to launch (0 = unlimited)
	SpawnMaxLaunched int
	// SpawnMaxLaunched keyframes (for dynamic particle count over time)
	SpawnMaxLaunchedKeyframes []particle.Keyframe
	SpawnMaxLaunchedInterp    string

	// Emitter area properties (发射区域)
	EmitterBoxX    float64 // Horizontal spawn area size (width, max - min)
	EmitterBoxY    float64 // Vertical spawn area size (height, max - min)
	EmitterBoxXMin float64 // Horizontal spawn area minimum (for asymmetric ranges)
	EmitterBoxYMin float64 // Vertical spawn area minimum (for asymmetric ranges)
	EmitterRadius  float64 // Deprecated: Use EmitterRadiusMin/Max instead (保留用于向后兼容)

	// EmitterRadius 范围支持（修复 Bug：支持 [min max] 格式）
	// 例如：Planting.xml 的 EmitterRadius="[0 10]" 表示粒子在半径 0-10 之间随机生成
	EmitterRadiusMin float64 // Minimum spawn radius
	EmitterRadiusMax float64 // Maximum spawn radius

	// Story 10.4: EmitterBox 关键帧支持（动态发射区域变化）
	// 例如：SodRoll.xml 的 EmitterBoxY 从 [-130, 0] 过渡到 [-100, 0]
	EmitterBoxXKeyframes []particle.Keyframe // EmitterBoxX 宽度关键帧
	EmitterBoxXInterp    string              // 插值模式
	EmitterBoxYKeyframes []particle.Keyframe // EmitterBoxY 宽度关键帧
	EmitterBoxYInterp    string              // 插值模式

	// 最小值关键帧（用于非对称范围的插值）
	// 例如：[-130 0] → [-100 0] 需要最小值从 -130 插值到 -100
	EmitterBoxXMinKeyframes []particle.Keyframe // EmitterBoxX 最小值关键帧
	EmitterBoxYMinKeyframes []particle.Keyframe // EmitterBoxY 最小值关键帧

	// Emitter position offset (发射器位置偏移)
	// 相对于发射器实体位置的偏移量，用于将粒子生成位置微调到特定位置
	// 例如：SeedPacket 光晕效果使用 EmitterOffsetY=62 将光晕向下移动
	EmitterOffsetX float64 // Horizontal offset from emitter position
	EmitterOffsetY float64 // Vertical offset from emitter position

	// System-level properties (Story 7.5: ZombieHead 系统透明度)
	SystemAlphaKeyframes []particle.Keyframe // 系统级透明度关键帧（影响所有粒子）
	SystemAlphaInterp    string              // 插值模式

	// Angle offset (角度偏移)
	AngleOffset float64 // LaunchAngle 的偏移量（度），例如 180° 用于翻转方向

	// Particle rotation override (粒子旋转覆盖)
	// 如果非零，将覆盖粒子的初始旋转角度（ParticleSpinAngle）
	// 用于教学箭头等需要特定方向的粒子效果
	ParticleRotationOverride float64

	// Story 10.4: SystemField 支持 (系统级力场效果)
	// SystemPosition: 控制发射器位置随时间变化（如 SodRoll 从左往右滚动）
	SystemPositionXKeyframes []particle.Keyframe // X轴位置关键帧
	SystemPositionXInterp    string              // X轴插值模式
	SystemPositionYKeyframes []particle.Keyframe // Y轴位置关键帧
	SystemPositionYInterp    string              // Y轴插值模式

	// Story 11.4: 初始位置（用于 SystemPosition 相对偏移计算）
	// 当 SystemPosition 关键帧非空时，SystemPosition 的值是相对于初始位置的偏移
	InitialX float64 // 发射器初始X坐标
	InitialY float64 // 发射器初始Y坐标
}
