package components

import (
	"github.com/decker502/pvz/internal/particle"
	"github.com/hajimehoshi/ebiten/v2"
)

// ParticleComponent represents a single particle instance in the particle system.
// It stores all the runtime state for an individual particle, including its
// position, velocity, visual properties, and lifecycle information.
//
// Particles are typically created and managed by the ParticleSystem, which
// updates their properties each frame and removes them when their lifetime expires.
//
// This is a pure data component following ECS principles - it contains no methods.
type ParticleComponent struct {
	// Position (世界坐标) - Note: Managed via separate PositionComponent
	// VelocityX and VelocityY are used to update position each frame

	// Velocity (速度, 像素/秒)
	VelocityX float64
	VelocityY float64

	// Rotation (旋转, 角度)
	Rotation      float64 // Current rotation angle in degrees
	RotationSpeed float64 // Rotation speed in degrees per second

	// Scale (缩放倍数)
	Scale float64 // Scale multiplier (1.0 = original size)

	// Transparency (透明度, 0-1)
	Alpha float64 // 0 = fully transparent, 1 = fully opaque

	// Color channels (颜色通道, 0-1)
	Red   float64 // Red channel multiplier
	Green float64 // Green channel multiplier
	Blue  float64 // Blue channel multiplier

	// Brightness (亮度乘数)
	Brightness float64 // Brightness multiplier applied to final color

	// Lifecycle (生命周期, 秒)
	Age      float64 // Time this particle has been alive (seconds)
	Lifetime float64 // Total lifetime before particle is destroyed (seconds)

	// Animation keyframes (动画关键帧)
	// These are used by ParticleSystem to interpolate values over the particle's lifetime
	AlphaKeyframes []particle.Keyframe // Keyframes for alpha animation
	ScaleKeyframes []particle.Keyframe // Keyframes for scale animation
	SpinKeyframes  []particle.Keyframe // Keyframes for rotation speed animation

	// Interpolation modes for keyframe animations
	AlphaInterpolation string // Interpolation mode for alpha ("Linear", "EaseIn", etc.)
	ScaleInterpolation string // Interpolation mode for scale
	SpinInterpolation  string // Interpolation mode for spin

	// Rendering properties
	Image    *ebiten.Image // Particle texture/sprite image
	Additive bool          // Use additive blending when rendering

	// Force fields (力场效果)
	// Copied from emitter config at spawn time for performance
	Fields []particle.Field // Force fields affecting this particle
}
