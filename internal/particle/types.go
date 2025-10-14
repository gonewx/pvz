// Package particle provides data structures and parsing functionality for
// Plants vs. Zombies particle effect configurations.
//
// This package handles XML-based particle configuration files that define
// emitters, particle behaviors, and visual effects.
package particle

// ParticleConfig represents the root structure of a particle effect configuration.
// A single particle effect may contain multiple emitters working together.
type ParticleConfig struct {
	Emitters []EmitterConfig `xml:"Emitter"`
}

// EmitterConfig represents a single particle emitter configuration.
// Each emitter defines how particles are spawned, rendered, and behave over time.
//
// Most fields use string types to preserve the original XML format, which may contain:
//   - Fixed values: "1500"
//   - Ranges: "[0.7 0.9]" (random value between min and max)
//   - Keyframes: "0,2 1,2 4,21" (time,value pairs)
//   - Interpolation keywords: "Linear", "FastInOutWeak", etc.
//
// These strings will be parsed at runtime (in Story 7.2) when the particle system
// needs to evaluate the actual values.
type EmitterConfig struct {
	// Name is the unique identifier for this emitter
	Name string `xml:"Name"`

	// Spawn properties (控制粒子发射)
	SpawnMinActive   string `xml:"SpawnMinActive,omitempty"`   // Minimum active particles
	SpawnMaxActive   string `xml:"SpawnMaxActive,omitempty"`   // Maximum active particles
	SpawnMaxLaunched string `xml:"SpawnMaxLaunched,omitempty"` // Maximum total particles to launch
	SpawnRate        string `xml:"SpawnRate,omitempty"`        // Particles spawned per second

	// Particle properties (粒子视觉属性)
	ParticleDuration    string `xml:"ParticleDuration,omitempty"`    // Lifetime in milliseconds
	ParticleAlpha       string `xml:"ParticleAlpha,omitempty"`       // Transparency (0-1)
	ParticleScale       string `xml:"ParticleScale,omitempty"`       // Size multiplier
	ParticleSpinAngle   string `xml:"ParticleSpinAngle,omitempty"`   // Initial rotation angle
	ParticleSpinSpeed   string `xml:"ParticleSpinSpeed,omitempty"`   // Rotation speed (degrees/sec)
	ParticleRed         string `xml:"ParticleRed,omitempty"`         // Red channel (0-1)
	ParticleGreen       string `xml:"ParticleGreen,omitempty"`       // Green channel (0-1)
	ParticleBlue        string `xml:"ParticleBlue,omitempty"`        // Blue channel (0-1)
	ParticleBrightness  string `xml:"ParticleBrightness,omitempty"`  // Brightness multiplier
	ParticleLoops       string `xml:"ParticleLoops,omitempty"`       // Animation loop count
	ParticleStretch     string `xml:"ParticleStretch,omitempty"`     // Stretch effect
	ParticlesDontFollow string `xml:"ParticlesDontFollow,omitempty"` // Don't follow emitter movement

	// Launch properties (发射参数)
	LaunchSpeed      string `xml:"LaunchSpeed,omitempty"`      // Initial velocity
	LaunchAngle      string `xml:"LaunchAngle,omitempty"`      // Launch direction (degrees)
	AlignLaunchSpin  string `xml:"AlignLaunchSpin,omitempty"`  // Align rotation to launch direction
	RandomLaunchSpin string `xml:"RandomLaunchSpin,omitempty"` // Random initial spin
	RandomStartTime  string `xml:"RandomStartTime,omitempty"`  // Random animation start time

	// Emitter properties (发射器配置)
	EmitterBoxX    string `xml:"EmitterBoxX,omitempty"`    // Spawn area width (horizontal)
	EmitterBoxY    string `xml:"EmitterBoxY,omitempty"`    // Spawn area height (vertical)
	EmitterRadius  string `xml:"EmitterRadius,omitempty"`  // Spawn radius (for circular emitters)
	EmitterType    string `xml:"EmitterType,omitempty"`    // Emitter shape type
	EmitterSkewX   string `xml:"EmitterSkewX,omitempty"`   // Horizontal skew
	EmitterOffsetX string `xml:"EmitterOffsetX,omitempty"` // Horizontal offset from emitter position
	EmitterOffsetY string `xml:"EmitterOffsetY,omitempty"` // Vertical offset from emitter position

	// System properties (系统级设置)
	SystemDuration string `xml:"SystemDuration,omitempty"` // Total effect duration (milliseconds)
	SystemAlpha    string `xml:"SystemAlpha,omitempty"`    // System-wide alpha modifier
	SystemLoops    string `xml:"SystemLoops,omitempty"`    // Number of times system repeats
	SystemField    string `xml:"SystemField,omitempty"`    // System-wide field effects

	// Image properties (贴图配置)
	Image         string `xml:"Image,omitempty"`         // Resource ID of the particle texture
	ImageFrames   string `xml:"ImageFrames,omitempty"`   // Number of animation frames
	ImageRow      string `xml:"ImageRow,omitempty"`      // Sprite sheet row
	ImageCol      string `xml:"ImageCol,omitempty"`      // Sprite sheet column
	Animated      string `xml:"Animated,omitempty"`      // Enable frame animation (0 or 1)
	AnimationRate string `xml:"AnimationRate,omitempty"` // Frames per second

	// Rendering properties (渲染模式)
	Additive     string `xml:"Additive,omitempty"`     // Additive blending (0 or 1)
	FullScreen   string `xml:"FullScreen,omitempty"`   // Full screen effect (0 or 1)
	HardwareOnly string `xml:"HardwareOnly,omitempty"` // Hardware acceleration required
	ClipTop      string `xml:"ClipTop,omitempty"`      // Top clipping

	// Cross-fade and lifecycle (过渡与生命周期)
	CrossFadeDuration string `xml:"CrossFadeDuration,omitempty"` // Fade duration when transitioning
	DieIfOverloaded   string `xml:"DieIfOverloaded,omitempty"`   // Kill effect if system overloaded

	// Collision properties (碰撞属性)
	CollisionReflect string `xml:"CollisionReflect,omitempty"` // Bounce coefficient
	CollisionSpin    string `xml:"CollisionSpin,omitempty"`    // Spin on collision

	// Fields (力场配置)
	Fields []Field `xml:"Field"` // Force fields affecting particles
}

// Field represents a force field that affects particle behavior.
// Fields can apply gravity, friction, acceleration, or other physics effects.
type Field struct {
	FieldType string `xml:"FieldType"`   // Type of field (e.g., "Acceleration", "Friction", "Attractor")
	X         string `xml:"X,omitempty"` // Horizontal force component (may be keyframes or range)
	Y         string `xml:"Y,omitempty"` // Vertical force component (may be keyframes or range)
}
