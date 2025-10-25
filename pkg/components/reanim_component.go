package components

import (
	"github.com/decker502/pvz/internal/reanim"
	"github.com/hajimehoshi/ebiten/v2"
)

// AnimLayer represents a single overlay animation layer (Story 6.4 - 已废弃).
//
// ⚠️ Deprecated (2025-10-24): 此结构体当前不使用，保留以备未来扩展。
//
// 经验证，原版游戏不使用动画叠加机制。所有动画通过简单的 PlayAnimation() 切换实现。
// 请勿在业务代码中使用此结构体。
//
// 相关文档：
//   - Sprint Change Proposal: docs/qa/sprint-change-proposal-story-6.4-animation-mechanism.md
//   - Story 6.4: docs/stories/6.4.story.md (标记为 Deprecated)
//
// ---
//
// 原始说明（历史记录，仅供参考）：
//
// Overlay animations are short-lived animations that play on top of the base animation,
// allowing for effects like blinking eyes, damage flashes, or attack effects.
//
// Example:
//
//	Base animation: "anim_idle" (continuous loop, controls body movement)
//	Overlay animation: "anim_blink" (one-shot, overrides mouth/eye tracks for 2-3 frames)
//
// This is a pure data structure with no methods, following ECS architecture principles.
type AnimLayer struct {
	// AnimName is the name of the overlay animation (e.g., "anim_blink").
	AnimName string

	// CurrentFrame is the current logical frame number for this layer (0-based).
	CurrentFrame int

	// FrameAccumulator is the frame accumulator for precise FPS control.
	// Accumulates deltaTime until it reaches the time for one animation frame (1.0/fps).
	FrameAccumulator float64

	// IsOneShot determines whether the animation plays once and is automatically removed.
	// If true, the layer will be removed from OverlayAnims when it completes.
	// If false, the animation loops continuously.
	IsOneShot bool

	// IsFinished indicates whether a one-shot animation has completed.
	// Set to true when CurrentFrame >= VisibleFrameCount for one-shot animations.
	// The layer will be removed in the next Update cycle.
	IsFinished bool

	// VisibleFrameCount is the number of visible frames in this overlay animation.
	// Built from the animation definition track when PlayAnimationOverlay is called.
	VisibleFrameCount int

	// AnimVisibles is the visibility array for this overlay animation.
	// Each element corresponds to a frame: 0 = visible, -1 = hidden.
	// Built from the animation definition track when PlayAnimationOverlay is called.
	AnimVisibles []int

	// AnimTracks is the list of part tracks to render for this overlay animation.
	// These tracks override the base animation's tracks with the same name.
	// Built when PlayAnimationOverlay is called.
	AnimTracks []reanim.Track
}

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

	// ====================================================================
	// Animation Overlay System (Story 6.4 - 已废弃，保留以备未来扩展)
	// ====================================================================
	//
	// ⚠️ 注意：以下字段当前不使用，所有动画通过简单的 PlayAnimation() 切换
	//
	// ⚠️ 原因：经验证，原版游戏不使用叠加机制，使用 VisibleTracks 控制部件显示
	//   - 原版通过动画定义中的 VisibleTracks（可见轨道列表）控制部件显示
	//   - 例如：anim_shooting 包含 stalk_bottom, stalk_top, anim_head_idle 所有需要的部件
	//   - 无需手动控制部件显示，也无需使用动画叠加
	//
	// ⚠️ 保留：避免大规模代码删除，为未来可能的扩展（Mod、特殊效果）保留
	//
	// ⚠️ 不应在业务代码中使用：请使用 PlayAnimation() 代替 PlayAnimationOverlay()
	//
	// 相关文档：
	//   - Sprint Change Proposal: docs/qa/sprint-change-proposal-story-6.4-animation-mechanism.md
	//   - Story 6.4: docs/stories/6.4.story.md (标记为 Deprecated)
	//   - Story 10.3: docs/stories/10.3.story.md (使用正确的简单切换方法)
	//
	// ====================================================================

	// BaseAnimName [未使用] 基础动画名称（如 "anim_idle"）
	// Deprecated: 使用 CurrentAnim 字段代替
	BaseAnimName string

	// OverlayAnims [未使用] 叠加动画列表
	// Deprecated: 使用 PlayAnimation() 简单切换代替 PlayAnimationOverlay()
	OverlayAnims []AnimLayer
}
