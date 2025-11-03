package components

import (
	"github.com/decker502/pvz/internal/reanim"
	"github.com/hajimehoshi/ebiten/v2"
)

// TrackPlaybackConfig defines playback behavior for an individual track (Story 12.1).
// This allows fine-grained control over track behavior at the business logic level.
type TrackPlaybackConfig struct {
	// PlayOnce indicates the track should play once and then lock at the final frame.
	// When true, the track will stop updating after reaching its last visible frame.
	// Used for one-time animations like tombstone rising or sign dropping.
	PlayOnce bool

	// IsLocked indicates the track has finished playing and is locked at a specific frame.
	// When true, ReanimSystem will not update this track's frame index.
	// Set automatically when PlayOnce track completes.
	IsLocked bool

	// LockedFrame is the frame number where the track is locked.
	// Only used when IsLocked is true.
	LockedFrame int

	// IsPaused indicates the track should temporarily stop updating.
	// Unlike IsLocked, paused tracks can be resumed.
	IsPaused bool
}

// ReanimComponent is a Reanim animation component (pure data, no methods).
// It stores the animation data, part images, and current animation state
// for entities using skeletal animations.
//
// This component follows the ECS architecture principle of data-behavior separation:
// all animation logic is implemented in ReanimSystem.
//
// Fields are organized into three logical groups:
// 1. Animation Definition - The animation data and playback mode
// 2. Playback State - Current runtime state of the animation
// 3. Advanced Features - Blending, caching, and control features
type ReanimComponent struct {
	// ==========================================================================
	// Animation Definition (动画定义)
	// ==========================================================================

	// PlaybackMode is the auto-detected playback mode (Story 6.6).
	// Automatically set by ReanimSystem when PlayAnimation is called.
	// Five modes: Simple, Skeleton, Sequence, ComplexScene, Blended.
	// Note: This is an int enum value, not an interface (preserves component purity).
	// The actual strategy interface is managed in ReanimSystem.
	PlaybackMode int

	// Reanim is the parsed Reanim animation data (from internal/reanim package).
	// Contains FPS and track definitions for the animation.
	Reanim *reanim.ReanimXML

	// PartImages maps image reference names to image objects.
	// Key: image reference name (e.g., "IMAGE_REANIM_PEASHOOTER_HEAD")
	// Value: corresponding Ebitengine image object
	PartImages map[string]*ebiten.Image

	// ==========================================================================
	// Playback State (播放状态)
	// ==========================================================================

	// CurrentAnim is the name of the currently playing animation (e.g., "anim_idle").
	CurrentAnim string

	// CurrentFrame is the current logical frame number (0-based index).
	// This is the frame number in the animation sequence, not the game loop frame.
	CurrentFrame int

	// FrameAccumulator is the frame accumulator for precise FPS control (float64).
	// Accumulates deltaTime until it reaches the time for one animation frame (1.0/fps).
	// This ensures accurate playback speed regardless of game loop framerate.
	FrameAccumulator float64

	// VisibleFrameCount is the number of visible frames in the current animation.
	// Used to determine when to loop the animation (when CurrentFrame >= VisibleFrameCount).
	VisibleFrameCount int

	// IsLooping determines whether the animation should loop.
	// If true, the animation will restart from frame 0 when it reaches the end.
	// If false, the animation will stay at the last frame (used for death animations).
	IsLooping bool

	// IsFinished indicates whether a non-looping animation has completed.
	// This is only set to true for non-looping animations when they reach the last frame.
	// For looping animations, this is always false.
	IsFinished bool

	// IsPaused determines whether the animation should update.
	// If true, ReanimSystem will skip updating CurrentFrame and FrameAccumulator.
	// Used for entities that should remain static (e.g., lawnmowers before trigger).
	IsPaused bool

	// ==========================================================================
	// Advanced Features (高级特性)
	// ==========================================================================

	// AnimVisiblesMap stores time windows for multiple animations (Story 6.5).
	// Key: animation name (e.g., "anim_idle", "anim_shooting")
	// Value: array mapping physical frame index to visibility (0 = visible, -1 = hidden)
	// This supports dual-animation blending where different parts use different animations.
	AnimVisiblesMap map[string][]int

	// MergedTracks is the accumulated frame array for each part track.
	// Key: track name (e.g., "head", "body")
	// Value: array of frames with accumulated transformations (frame inheritance applied)
	// Built by buildMergedTracks when PlayAnimation is called.
	MergedTracks map[string][]reanim.Frame

	// --- Dual Animation Blending (Story 6.5) ---

	// IsBlending indicates whether dual-animation blending is active.
	// When true, the entity renders using two animations simultaneously:
	// - PrimaryAnimation for body parts (e.g., "anim_idle")
	// - SecondaryAnimation for head parts (e.g., "anim_shooting")
	IsBlending bool

	// PrimaryAnimation is the base animation for body parts (e.g., "anim_idle").
	// Used when IsBlending is true.
	PrimaryAnimation string

	// SecondaryAnimation is the overlay animation for head parts (e.g., "anim_shooting").
	// Used when IsBlending is true.
	SecondaryAnimation string

	// --- Rendering and Caching ---

	// AnimTracks is the list of part tracks to render for the current animation, in rendering order.
	// This preserves the Z-order from the Reanim file.
	// Built by getAnimationTracks when PlayAnimation is called.
	AnimTracks []reanim.Track

	// CenterOffsetX and CenterOffsetY are the offsets to center the animation visually.
	// These values are calculated based on the bounding box of all visible parts
	// in the first frame of the animation, to align the visual center with the entity position.
	CenterOffsetX float64
	CenterOffsetY float64

	// FixedCenterOffset 是否使用固定的中心偏移量（避免动画切换时的位置跳动）
	// 如果为 true，则 CenterOffsetX/Y 在实体创建时计算一次后固定不变
	// 如果为 false，则每次 PlayAnimation 时重新计算 CenterOffset
	// 用于解决不同动画包围盒大小不同导致的位置跳动问题
	FixedCenterOffset bool

	// BestPreviewFrame is the optimal frame index for generating preview icons (Story 10.3).
	// This frame is automatically calculated when PlayAnimation is called by finding the frame
	// with the most visible parts (highest count of ImagePath != "").
	// Used by RenderPlantIcon to ensure preview shows the most complete representation of the entity.
	// Default: 0 (first frame)
	BestPreviewFrame int

	// --- Visibility Control ---

	// VisibleTracks is a whitelist of track names that should be rendered.
	// If this map is not nil and not empty, ONLY tracks in this map will be rendered.
	// This provides a clear "opt-in" approach for complex entities like zombies.
	// Use ReanimSystem.HideTrack() and ShowTrack() to manage track visibility.
	VisibleTracks map[string]bool

	// PartGroups defines logical groupings of animation tracks (pure data configuration).
	// This allows high-level operations like "hide arm" without knowing specific track names.
	// Example:
	//   PartGroups: map[string][]string{
	//       "arm":   {"Zombie_outerarm_hand", "Zombie_outerarm_upper", "Zombie_outerarm_lower"},
	//       "head":  {"anim_head1", "anim_head2"},
	//       "armor": {"anim_cone"},
	//   }
	// Use ReanimSystem.HidePartGroup() and ShowPartGroup() to manage part group visibility.
	PartGroups map[string][]string

	// TrackConfigs stores per-track playback configuration (Story 12.1).
	// Key: track name (e.g., "SelectorScreen_Adventure_button", "Cloud1")
	// Value: playback configuration for this track
	// This allows business logic to control individual track behavior:
	// - Some tracks play once and lock (e.g., tombstone rising)
	// - Some tracks loop continuously (e.g., clouds, grass)
	// - Some tracks can be paused/resumed
	TrackConfigs map[string]*TrackPlaybackConfig
}
