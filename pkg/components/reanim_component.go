package components

import (
	"github.com/decker502/pvz/internal/reanim"
	"github.com/hajimehoshi/ebiten/v2"
)

// ReanimComponent is a Reanim animation component (pure data, no methods).
// It stores the animation data, part images, and current animation state
// for entities using skeletal animations.
//
// This component follows the ECS architecture principle of data-behavior separation:
// all animation logic is implemented in ReanimSystem.
type ReanimComponent struct {
	// Reanim is the parsed Reanim animation data (from internal/reanim package).
	// Contains FPS and track definitions for the animation.
	Reanim *reanim.ReanimXML

	// PartImages maps image reference names to image objects.
	// Key: image reference name (e.g., "IMAGE_REANIM_PEASHOOTER_HEAD")
	// Value: corresponding Ebitengine image object
	PartImages map[string]*ebiten.Image

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

	// AnimVisibles is the visibility array for the current animation.
	// Each element corresponds to a frame: 0 = visible, -1 = hidden.
	// Built from the animation definition track when PlayAnimation is called.
	AnimVisibles []int

	// MergedTracks is the accumulated frame array for each part track.
	// Key: track name (e.g., "head", "body")
	// Value: array of frames with accumulated transformations (frame inheritance applied)
	// Built by buildMergedTracks when PlayAnimation is called.
	MergedTracks map[string][]reanim.Frame

	// AnimTracks is the list of part tracks to render for the current animation, in rendering order.
	// This preserves the Z-order from the Reanim file.
	// Built by getAnimationTracks when PlayAnimation is called.
	AnimTracks []reanim.Track

	// CenterOffsetX and CenterOffsetY are the offsets to center the animation visually.
	// These values are calculated based on the bounding box of all visible parts
	// in the first frame of the animation, to align the visual center with the entity position.
	CenterOffsetX float64
	CenterOffsetY float64

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
}
