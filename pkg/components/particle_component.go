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
	Age           float64 // Time this particle has been alive (seconds)
	Lifetime      float64 // Total lifetime before particle is destroyed (seconds)
	ParticleLoops bool    // If true, particle resets Age when reaching Lifetime instead of being destroyed

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
	Image       *ebiten.Image // Particle texture/sprite image (full sprite sheet or single frame)
	ImageFrames int           // Number of frames (columns) in the sprite sheet (1 = single image, >1 = sprite sheet)
	ImageRows   int           // Number of rows in the sprite sheet (1 = single row, >1 = multi-row sprite sheet)
	FrameNum    int           // Current frame number (0-based index, used for sprite sheets)
	Additive    bool          // Use additive blending when rendering

	// Force fields (力场效果)
	// Copied from emitter config at spawn time for performance
	Fields []particle.Field // Force fields affecting this particle

	// Collision properties (Story 7.5: ZombieHead 弹跳效果)
	CollisionReflectX     float64             // X轴反弹系数（速度乘数）
	CollisionReflectY     float64             // Y轴反弹系数（速度乘数）
	CollisionReflectCurve []particle.Keyframe // 反弹系数随时间变化曲线
	CollisionSpinMin      float64             // 碰撞时增加的旋转速度范围（最小值）
	CollisionSpinMax      float64             // 碰撞时增加的旋转速度范围（最大值）
	CollisionSpinCurve    []particle.Keyframe // 碰撞旋转效果随时间变化曲线
	GroundY               float64             // 地面约束Y坐标（0表示无地面）

	// System-level properties (Story 7.5: 系统级透明度)
	SystemAlphaKeyframes []particle.Keyframe // 系统级透明度关键帧（基于发射器年龄）
	SystemAlphaInterp    string              // 插值模式
	EmitterAge           float64             // 发射器年龄（用于计算 SystemAlpha）
	EmitterDuration      float64             // 发射器持续时间（用于归一化 SystemAlpha）

	// Position Field 支持（位置动画路径）
	// Position Field 直接设置粒子相对于初始位置的偏移量（覆盖速度积分）
	// 例如：SeedPacket 箭头使用 Position Field 实现"向下晃动再回到原位"的动画
	InitialX        float64 // 粒子初始 X 坐标（用于 Position Field 计算相对位置）
	InitialY        float64 // 粒子初始 Y 坐标
	PositionOffsetX float64 // Position Field 计算的 X 偏移量（每帧更新）
	PositionOffsetY float64 // Position Field 计算的 Y 偏移量

	// Position Field 关键帧（在粒子生成时解析一次，避免每帧重复解析）
	// 支持特殊格式如 "0 [-40 10]"：从初始值 0 插值到范围内的随机目标值
	PositionFieldXKeyframes []particle.Keyframe // Position Field X 轴关键帧
	PositionFieldYKeyframes []particle.Keyframe // Position Field Y 轴关键帧
	PositionFieldXInterp    string              // X 轴插值模式
	PositionFieldYInterp    string              // Y 轴插值模式
	HasPositionField        bool                // 是否有 Position Field（用于快速检查）
}
