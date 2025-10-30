package systems

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// ==================================================================
// Story 6.5: Reanim Track Type Constants (轨道类型常量)
// ==================================================================
//
// These constants define the track types used in the Reanim system.
// Based on analysis of 5 Reanim files (87 tracks total):
// - 76% are hybrid tracks (images + f values + transforms)
// - 23% are animation definition tracks (only f values)
// - <1% are pure visual tracks (only images)
// - <1% are logical tracks (no images, only transforms)
//
// Reference: docs/reanim/reanim-hybrid-track-discovery.md

// AnimationDefinitionTracks are tracks that only define time windows.
// They have FrameNum values but no images or transforms.
// Example: anim_idle, anim_shooting, anim_head_idle, anim_full_idle
var AnimationDefinitionTracks = map[string]bool{
	"anim_idle":      true,
	"anim_shooting":  true,
	"anim_head_idle": true,
	"anim_full_idle": true,
}

// LogicalTracks are tracks that define attachment points or parent transforms.
// They have position/transform data but no images.
// Example: anim_stem (parent bone for head parts), _ground (ground attachment point)
var LogicalTracks = map[string]bool{
	"anim_stem": true,
	"_ground":   true,
}

// HeadTracks are tracks that belong to the head part group.
// These tracks use the secondary animation (e.g., anim_shooting) in dual-animation mode.
// Head tracks also inherit anim_stem offsets for parent-child hierarchy.
var HeadTracks = map[string]bool{
	"anim_face":        true,
	"idle_mouth":       true,
	"anim_blink":       true,
	"idle_shoot_blink": true,
	"anim_sprout":      true,
}

// ReanimStemInitX and ReanimStemInitY are the initial position of anim_stem.
// These values are extracted from PeaShooterSingle.reanim at frame 4 (first visible frame).
// Used to calculate stem offset for head parts: offset = current_pos - init_pos
const (
	ReanimStemInitX = 37.6
	ReanimStemInitY = 48.7
)

// ReanimSystem is the Reanim animation system that manages skeletal animations
// for entities with ReanimComponent.
//
// This system is responsible for:
// - Advancing animation frames based on FPS
// - Implementing frame inheritance (cumulative transformations)
// - Managing animation loops
// - Supporting dual-animation blending (Story 6.5)
//
// All animation logic is centralized in this system, following the ECS
// architecture principle of data-behavior separation.
type ReanimSystem struct {
	entityManager *ecs.EntityManager
}

// NewReanimSystem creates a new Reanim animation system.
//
// Parameters:
//   - em: the EntityManager that manages all entities and components
//
// Returns:
//   - A pointer to the newly created ReanimSystem
func NewReanimSystem(em *ecs.EntityManager) *ReanimSystem {
	return &ReanimSystem{
		entityManager: em,
	}
}

// Update updates all Reanim components by advancing animation frames.
//
// This method:
// - Queries all entities with ReanimComponent
// - Advances the frame counter based on FPS
// - Updates the current frame when enough time has passed
// - Loops the animation when it reaches the end
//
// Parameters:
//   - deltaTime: time elapsed since last update (in seconds, currently unused as we use frame-based timing)
func (s *ReanimSystem) Update(deltaTime float64) {
	// Query all entities with ReanimComponent
	entities := ecs.GetEntitiesWith1[*components.ReanimComponent](s.entityManager)

	for _, id := range entities {
		// Get the ReanimComponent
		reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, id)
		if !exists {
			continue
		}

		// Skip if no Reanim data or animation is not set
		if reanimComp.Reanim == nil || reanimComp.CurrentAnim == "" {
			continue
		}

		// Skip if animation is paused (e.g., lawnmower before trigger)
		if reanimComp.IsPaused {
			continue
		}

		// Get FPS from Reanim data, default to 12 if not set
		fps := float64(reanimComp.Reanim.FPS)
		if fps == 0 {
			fps = 12.0 // Default FPS for PVZ animations
		}

		// Calculate time per frame (seconds)
		timePerFrame := 1.0 / fps

		// Accumulate deltaTime
		reanimComp.FrameAccumulator += deltaTime

		// Advance frames based on accumulated time
		// 限制每次Update只推进1帧，避免跳帧导致视觉抖动
		if reanimComp.FrameAccumulator >= timePerFrame {
			reanimComp.FrameAccumulator -= timePerFrame
			reanimComp.CurrentFrame++

			// Loop animation when reaching the end (only if IsLooping is true)
			if reanimComp.CurrentFrame >= reanimComp.VisibleFrameCount {
				if reanimComp.IsLooping {
					reanimComp.CurrentFrame = 0
				} else {
					// Non-looping animation: stay at the last frame and mark as finished
					reanimComp.CurrentFrame = reanimComp.VisibleFrameCount - 1
					reanimComp.FrameAccumulator = 0 // Reset accumulator
					// 只在第一次标记完成时打印日志，避免每帧重复输出
					if !reanimComp.IsFinished {
						reanimComp.IsFinished = true
						log.Printf("[ReanimSystem] 非循环动画完成: Entity=%d, Anim=%s, Frame=%d/%d",
							id, reanimComp.CurrentAnim, reanimComp.CurrentFrame, reanimComp.VisibleFrameCount)
					}
				}
			}
		}

		// Story 6.4: Update overlay animations
		for i := 0; i < len(reanimComp.OverlayAnims); i++ {
			layer := &reanimComp.OverlayAnims[i]

			// Accumulate deltaTime for the overlay layer
			layer.FrameAccumulator += deltaTime

			// Advance frames for the overlay layer
			if layer.FrameAccumulator >= timePerFrame {
				layer.FrameAccumulator -= timePerFrame
				layer.CurrentFrame++

				// Check if the overlay animation has finished
				if layer.CurrentFrame >= layer.VisibleFrameCount {
					if layer.IsOneShot {
						// Mark as finished (will be removed in the next step)
						layer.IsFinished = true
					} else {
						// Loop the overlay animation
						layer.CurrentFrame = 0
					}
				}
			}
		}

		// Story 6.4: Remove finished overlay animations
		// Use the filter pattern to safely remove elements from the slice
		i := 0
		for _, layer := range reanimComp.OverlayAnims {
			if !layer.IsFinished {
				reanimComp.OverlayAnims[i] = layer
				i++
			}
		}
		reanimComp.OverlayAnims = reanimComp.OverlayAnims[:i]
	}
}

// getAnimDefinitionTrack returns the animation definition track for the given animation name.
//
// Animation definition tracks are tracks whose names start with "anim_" (e.g., "anim_idle", "anim_shooting").
// These tracks control the overall animation visibility and timing.
//
// Important: This method validates that the found track is actually an animation definition track
// (has FrameNum but no image/transform data). This prevents accidentally using part tracks or
// transform tracks as animation definitions.
//
// Parameters:
//   - comp: the ReanimComponent containing the Reanim data
//   - animName: the name of the animation to find (e.g., "anim_idle")
//
// Returns:
//   - A pointer to the Track if found and valid, nil otherwise
func (s *ReanimSystem) getAnimDefinitionTrack(comp *components.ReanimComponent, animName string) *reanim.Track {
	if comp.Reanim == nil {
		return nil
	}

	// Iterate through all tracks to find the one with the matching name
	for i := range comp.Reanim.Tracks {
		track := &comp.Reanim.Tracks[i]
		if track.Name == animName {
			// Story 8.6 QA修正: 移除多余的动画定义轨道限制
			// 原因: 某些植物（如向日葵）的 reanim 文件中，动画轨道同时包含动画定义和部件渲染数据
			// 这是原版游戏的正常结构，不应该被限制
			// 只要轨道名称匹配，就认为是有效的动画定义

			// 对于标准动画定义轨道（只包含 FrameNum），直接返回
			if s.isAnimationDefinitionTrack(track) {
				return track
			}

			// 对于包含图片/变换的轨道，也允许作为动画定义使用
			// 这种情况在原版游戏中很常见，例如：
			// - SunFlower.reanim 的 anim_idle 轨道
			// - FinalWave.reanim 的单轨道动画
			log.Printf("[ReanimSystem] Using animation track '%s' (contains images/transforms, valid for animation definition)", animName)
			return track
		}
	}

	return nil
}

// isAnimationDefinitionTrack validates if a track is an animation definition track.
//
// Reanim files have multiple track types:
//  1. Animation definition tracks: only FrameNum, no images, no transforms
//     Examples: anim_idle, anim_shooting, anim_full_idle
//  2. Part tracks: have images and transforms
//     Examples: backleaf, frontleaf, stalk_bottom, anim_face
//  3. Transform tracks: have transforms but no images (for bone transforms)
//     Examples: anim_stem
//  4. Hybrid tracks: have images + transforms + FrameNum (overlay animations)
//     Examples: anim_blink, idle_shoot_blink
//
// This method returns true only for type 1 (animation definition tracks).
func (s *ReanimSystem) isAnimationDefinitionTrack(track *reanim.Track) bool {
	hasImageRef := false
	hasTransform := false
	hasFrameNum := false

	for _, frame := range track.Frames {
		// Check for image references
		if frame.ImagePath != "" {
			hasImageRef = true
		}
		// Check for transform data
		if frame.X != nil || frame.Y != nil || frame.ScaleX != nil || frame.ScaleY != nil {
			hasTransform = true
		}
		// Check for FrameNum
		if frame.FrameNum != nil {
			hasFrameNum = true
		}
	}

	// Animation definition track: has FrameNum, but no images or transforms
	return hasFrameNum && !hasImageRef && !hasTransform
}

// ==================================================================
// Story 6.5: Track Type Helper Functions (轨道类型辅助函数)
// ==================================================================

// isAnimationDefinitionTrackByName checks if a track is an animation definition track by name.
// This is a fast check using the hardcoded list of known animation definition tracks.
func (s *ReanimSystem) isAnimationDefinitionTrackByName(trackName string) bool {
	return AnimationDefinitionTracks[trackName]
}

// isLogicalTrack checks if a track is a logical track (no images, only transforms).
// Logical tracks like anim_stem define attachment points or parent bones.
func (s *ReanimSystem) isLogicalTrack(trackName string) bool {
	return LogicalTracks[trackName]
}

// isHeadTrack checks if a track belongs to the head part group.
// Head tracks use the secondary animation in dual-animation mode.
func (s *ReanimSystem) isHeadTrack(trackName string) bool {
	return HeadTracks[trackName]
}

// hasFrameNumValues checks if a track has any FrameNum values.
// Used to distinguish hybrid tracks (with f values) from pure visual tracks (without f values).
func (s *ReanimSystem) hasFrameNumValues(track *reanim.Track) bool {
	for _, frame := range track.Frames {
		if frame.FrameNum != nil {
			return true
		}
	}
	return false
}

// buildVisiblesArray builds the visibility array for the given animation.
//
// The visibility array determines which frames should be visible during animation playback.
// Each element corresponds to a frame: 0 = visible, -1 = hidden.
// This is built from the animation definition track (e.g., "anim_idle").
//
// Frame inheritance is applied: if a frame's FrameNum is nil, it inherits the value
// from the previous frame. The first frame defaults to 0 (visible) if not specified.
//
// Parameters:
//   - comp: the ReanimComponent containing the Reanim data
//   - animName: the name of the animation (e.g., "anim_idle")
//
// Returns:
//   - An array of visibility values (length = standard frame count)
func (s *ReanimSystem) buildVisiblesArray(comp *components.ReanimComponent, animName string) []int {
	// Get the animation definition track
	animTrack := s.getAnimDefinitionTrack(comp, animName)
	if animTrack == nil {
		return []int{}
	}

	// Determine the standard frame count (max frames across all tracks)
	standardFrameCount := 0
	for _, track := range comp.Reanim.Tracks {
		if len(track.Frames) > standardFrameCount {
			standardFrameCount = len(track.Frames)
		}
	}

	if standardFrameCount == 0 {
		return []int{}
	}

	// Build the visibility array with frame inheritance
	visibles := make([]int, standardFrameCount)
	currentValue := 0 // Default to visible for the first frame

	for i := 0; i < standardFrameCount; i++ {
		if i < len(animTrack.Frames) {
			frame := animTrack.Frames[i]
			// If FrameNum is specified, use it; otherwise inherit from previous frame
			if frame.FrameNum != nil {
				currentValue = *frame.FrameNum
			}
		}
		// Assign the current value (either explicitly set or inherited)
		visibles[i] = currentValue
	}

	return visibles
}

// buildMergedTracks builds accumulated frame arrays for all part tracks.
//
// This is the core of the frame inheritance mechanism. For each part track (e.g., "head", "body"),
// it creates an array of frames where each frame contains the accumulated transformations
// from all previous frames.
//
// Frame inheritance rules:
// - If a field (X, Y, ScaleX, etc.) is nil in a frame, it inherits the value from the previous frame
// - Each frame in the output array has independent pointers (no shared addresses)
// - Initial values: X=0, Y=0, ScaleX=1, ScaleY=1, SkewX=0, SkewY=0, FrameNum=0, ImagePath=""
//
// Parameters:
//   - comp: the ReanimComponent containing the Reanim data
//
// Returns:
//   - A map of track name to array of accumulated frames (length = standard frame count)
func (s *ReanimSystem) buildMergedTracks(comp *components.ReanimComponent) map[string][]reanim.Frame {
	if comp.Reanim == nil {
		return map[string][]reanim.Frame{}
	}

	// Determine the standard frame count (max frames across all tracks)
	standardFrameCount := 0
	for _, track := range comp.Reanim.Tracks {
		if len(track.Frames) > standardFrameCount {
			standardFrameCount = len(track.Frames)
		}
	}

	if standardFrameCount == 0 {
		return map[string][]reanim.Frame{}
	}

	mergedTracks := make(map[string][]reanim.Frame)

	// Process ALL tracks, including animation definition tracks
	// Some plants (like SunFlower) have their head images in anim_* tracks
	for _, track := range comp.Reanim.Tracks {
		// Initialize accumulated state
		accX := 0.0
		accY := 0.0
		accSX := 1.0
		accSY := 1.0
		accKX := 0.0
		accKY := 0.0
		accF := 0
		accImg := ""

		// Build merged frames array for this track
		mergedFrames := make([]reanim.Frame, standardFrameCount)

		for i := 0; i < standardFrameCount; i++ {
			// If the original track has a frame at this index, update accumulated state
			if i < len(track.Frames) {
				frame := track.Frames[i]

				// Update accumulated values only if field is not nil
				if frame.X != nil {
					accX = *frame.X
				}
				if frame.Y != nil {
					accY = *frame.Y
				}
				if frame.ScaleX != nil {
					accSX = *frame.ScaleX
				}
				if frame.ScaleY != nil {
					accSY = *frame.ScaleY
				}
				if frame.SkewX != nil {
					accKX = *frame.SkewX
				}
				if frame.SkewY != nil {
					accKY = *frame.SkewY
				}
				if frame.FrameNum != nil {
					accF = *frame.FrameNum
				}
				if frame.ImagePath != "" {
					accImg = frame.ImagePath
				}
			}

			// Create a new frame with independent pointers (avoid pointer sharing)
			// Each frame gets its own copy of the accumulated values
			x := accX
			y := accY
			sx := accSX
			sy := accSY
			kx := accKX
			ky := accKY
			f := accF

			mergedFrames[i] = reanim.Frame{
				X:         &x,
				Y:         &y,
				ScaleX:    &sx,
				ScaleY:    &sy,
				SkewX:     &kx,
				SkewY:     &ky,
				FrameNum:  &f,
				ImagePath: accImg,
			}
		}

		mergedTracks[track.Name] = mergedFrames
	}

	return mergedTracks
}

// getAnimationTracks returns all part tracks that should be rendered for the animation.
//
// This includes ALL tracks that contain image references, INCLUDING animation definition tracks.
// Some plants (like SunFlower) have their head images in the anim_* tracks!
// The order of tracks in the returned slice determines the rendering order (Z-order).
//
// Parameters:
//   - comp: the ReanimComponent containing the Reanim data
//
// Returns:
//   - A slice of tracks in rendering order
func (s *ReanimSystem) getAnimationTracks(comp *components.ReanimComponent) []reanim.Track {
	if comp.Reanim == nil {
		return nil
	}

	var result []reanim.Track
	for _, track := range comp.Reanim.Tracks {
		// Include ALL tracks that have at least one frame with an image
		// This includes both part tracks (e.g., "head", "body") and animation tracks (e.g., "anim_idle")
		// because some plants store part images in animation tracks
		hasImage := false
		for _, frame := range track.Frames {
			if frame.ImagePath != "" {
				hasImage = true
				break
			}
		}

		if hasImage {
			result = append(result, track)
		}
	}
	return result
}

// PlayAnimation starts playing the specified animation for the given entity.
//
// This method:
// - Resets the animation state (frame = 0, counter = 0)
// - Builds the visibility array from the animation definition track
// - Builds the merged tracks with frame inheritance
// - Stores the animation tracks in rendering order
// - Updates the component with the new animation state
// - Story 6.5: Enables dual-animation blending for anim_shooting
//
// Parameters:
//   - entityID: the ID of the entity to play the animation on
//   - animName: the name of the animation to play (e.g., "anim_idle")
//
// Returns:
//   - An error if the entity doesn't have a ReanimComponent or the animation doesn't exist
func (s *ReanimSystem) PlayAnimation(entityID ecs.EntityID, animName string) error {
	// Get the ReanimComponent
	reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !exists {
		return fmt.Errorf("entity %d does not have a ReanimComponent", entityID)
	}

	// Check if Reanim data is present
	if reanimComp.Reanim == nil {
		return fmt.Errorf("entity %d has a ReanimComponent but no Reanim data", entityID)
	}

	// Check if the animation exists
	animTrack := s.getAnimDefinitionTrack(reanimComp, animName)
	if animTrack == nil {
		return fmt.Errorf("animation '%s' not found in Reanim data for entity %d", animName, entityID)
	}

	// Reset animation state
	reanimComp.CurrentFrame = 0
	reanimComp.FrameAccumulator = 0.0
	reanimComp.CurrentAnim = animName
	reanimComp.IsLooping = true   // Default: animations loop
	reanimComp.IsFinished = false // Reset finished flag

	// Story 6.4: Set base animation name and clear overlay animations
	// When switching base animations, all overlay animations are cleared
	reanimComp.BaseAnimName = animName
	reanimComp.OverlayAnims = []components.AnimLayer{} // Clear all overlay animations

	// ==================================================================
	// Story 6.5: Dual-Animation Blending (双动画叠加)
	// ==================================================================
	//
	// When playing anim_shooting, enable dual-animation mode:
	// - PrimaryAnimation: anim_idle (body parts continue swaying)
	// - SecondaryAnimation: anim_shooting (head parts shoot)
	//
	// This fixes the bug where only the head was visible during shooting.
	if animName == "anim_shooting" {
		reanimComp.IsBlending = true
		reanimComp.PrimaryAnimation = "anim_idle"
		reanimComp.SecondaryAnimation = "anim_shooting"

		log.Printf("[ReanimSystem] Entity %d: Enabling dual-animation blending (idle + shooting)", entityID)
	} else {
		// Single animation mode
		reanimComp.IsBlending = false
		reanimComp.PrimaryAnimation = animName
		reanimComp.SecondaryAnimation = ""
	}

	// Initialize AnimVisiblesMap if needed
	if reanimComp.AnimVisiblesMap == nil {
		reanimComp.AnimVisiblesMap = make(map[string][]int)
	}

	// Build visibility array for the current animation
	reanimComp.AnimVisibles = s.buildVisiblesArray(reanimComp, animName)
	reanimComp.AnimVisiblesMap[animName] = reanimComp.AnimVisibles

	// Calculate visible frame count (number of frames with visibility 0)
	visibleCount := 0
	for _, v := range reanimComp.AnimVisibles {
		if v == 0 {
			visibleCount++
		}
	}
	reanimComp.VisibleFrameCount = visibleCount

	// Story 6.5: If dual-animation mode, also build visibility for the primary animation
	if reanimComp.IsBlending && reanimComp.PrimaryAnimation != animName {
		primaryVisibles := s.buildVisiblesArray(reanimComp, reanimComp.PrimaryAnimation)
		reanimComp.AnimVisiblesMap[reanimComp.PrimaryAnimation] = primaryVisibles

		log.Printf("[ReanimSystem] Built visibility for primary animation '%s': %d visible frames",
			reanimComp.PrimaryAnimation, len(primaryVisibles))
	}

	// Build merged tracks with frame inheritance
	reanimComp.MergedTracks = s.buildMergedTracks(reanimComp)

	// Store animation tracks in rendering order
	reanimComp.AnimTracks = s.getAnimationTracks(reanimComp)

	// Calculate center offset based on the bounding box of visible parts in the first frame
	// 如果 FixedCenterOffset 为 true，则跳过重新计算（保持创建时的值）
	// 这样可以避免不同动画包围盒大小不同导致的位置跳动
	if !reanimComp.FixedCenterOffset {
		s.calculateCenterOffset(reanimComp)
	} else {
		log.Printf("[ReanimSystem] 动画 '%s' 使用固定中心偏移，跳过重新计算", animName)
	}

	// Story 10.3: Calculate best preview frame (frame with most visible parts)
	// This is used by RenderPlantIcon to ensure preview shows the most complete representation
	bestFrame := 0
	maxVisibleParts := 0

	for frameIdx := 0; frameIdx < reanimComp.VisibleFrameCount; frameIdx++ {
		// Skip invisible frames
		if frameIdx < len(reanimComp.AnimVisibles) && reanimComp.AnimVisibles[frameIdx] == -1 {
			continue
		}

		// Count visible parts in this frame
		visiblePartsCount := 0
		for _, mergedFrames := range reanimComp.MergedTracks {
			if frameIdx < len(mergedFrames) && mergedFrames[frameIdx].ImagePath != "" {
				visiblePartsCount++
			}
		}

		// Update best frame if this frame has more visible parts
		if visiblePartsCount > maxVisibleParts {
			maxVisibleParts = visiblePartsCount
			bestFrame = frameIdx
		}
	}

	reanimComp.BestPreviewFrame = bestFrame

	return nil
}

// PlayAnimationNoLoop starts playing the specified animation for the given entity WITHOUT looping.
// This is used for one-shot animations like death animations.
//
// The animation will play once and stay at the last frame when it reaches the end.
//
// Parameters:
//   - entityID: the ID of the entity to play the animation on
//   - animName: the name of the animation to play (e.g., "anim_death")
//
// Returns:
//   - An error if the entity doesn't have a ReanimComponent or the animation doesn't exist
func (s *ReanimSystem) PlayAnimationNoLoop(entityID ecs.EntityID, animName string) error {
	// Use the main PlayAnimation method to set up the animation
	if err := s.PlayAnimation(entityID, animName); err != nil {
		return err
	}

	// Override IsLooping to false for non-looping animations
	reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !exists {
		return fmt.Errorf("entity %d does not have a ReanimComponent", entityID)
	}
	reanimComp.IsLooping = false

	return nil
}

// PlayAnimationOverlay starts playing an overlay animation on top of the base animation (Story 6.4).
//
// ⚠️ Deprecated (2025-10-24): 此方法当前不使用，所有动画通过简单的 PlayAnimation() 切换实现。
//
// 经验证，原版《植物大战僵尸》不使用动画叠加机制。所有动画（包括攻击、眨眼、状态切换）
// 都通过简单的 PlayAnimation() 切换实现，部件显示由 VisibleTracks 机制控制。
//
// 保留此方法以备未来可能的扩展（如 Mod 支持、特殊效果），但当前不应在业务代码中调用。
//
// 推荐使用：
//   - PlayAnimation(entityID, "anim_shooting")  // 切换到攻击动画
//   - PlayAnimation(entityID, "anim_idle")      // 切换回空闲动画
//
// VisibleTracks 机制说明：
//   - 原版游戏通过动画定义中的 VisibleTracks（可见轨道列表）控制部件显示
//   - 例如：anim_shooting 包含 stalk_bottom, stalk_top, anim_head_idle 所有需要的部件
//   - 渲染系统自动根据轨道定义渲染所有可见部件
//   - 无需手动控制部件显示，也无需使用动画叠加
//
// 相关文档：
//   - Sprint Change Proposal: docs/qa/sprint-change-proposal-story-6.4-animation-mechanism.md
//   - Story 6.4: docs/stories/6.4.story.md (标记为 Deprecated)
//   - Story 10.3: docs/stories/10.3.story.md (使用正确的简单切换方法)
//
// ---
//
// 原始说明（历史记录，仅供参考）：
//
// Overlay animations are rendered after the base animation and can override specific tracks.
// This is used for effects like blinking eyes, damage flashes, or attack effects.
//
// Example:
//
//	Base animation: "anim_idle" (continuous, controls body)
//	Overlay animation: "anim_blink" (one-shot, overrides mouth/eye tracks)
//
// Parameters:
//   - entityID: the entity to play the overlay animation on
//   - animName: the name of the overlay animation (e.g., "anim_blink")
//   - playOnce: if true, the animation plays once and is automatically removed;
//     if false, the animation loops continuously
//
// Returns:
//   - An error if the entity doesn't have a ReanimComponent or the animation doesn't exist
//
// Deprecated: 使用 PlayAnimation() 代替
func (s *ReanimSystem) PlayAnimationOverlay(entityID ecs.EntityID, animName string, playOnce bool) error {
	// Get the ReanimComponent
	reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !exists {
		return fmt.Errorf("entity %d does not have a ReanimComponent", entityID)
	}

	// Check if Reanim data is present
	if reanimComp.Reanim == nil {
		return fmt.Errorf("entity %d has a ReanimComponent but no Reanim data", entityID)
	}

	// Check if the overlay animation exists
	animTrack := s.getAnimDefinitionTrack(reanimComp, animName)
	if animTrack == nil {
		return fmt.Errorf("overlay animation '%s' not found in Reanim data for entity %d", animName, entityID)
	}

	// Build visibility array for the overlay animation
	animVisibles := s.buildVisiblesArray(reanimComp, animName)

	// Calculate visible frame count
	visibleCount := 0
	for _, v := range animVisibles {
		if v == 0 {
			visibleCount++
		}
	}

	// Get animation tracks for the overlay animation
	// Note: getAnimationTracks() returns ALL tracks with images, not just those for a specific animation.
	// For overlay animations, we use the same tracks as the base animation, because overlay animations
	// typically override specific tracks (like mouth/eyes) rather than defining completely new tracks.
	// The rendering system will check each track to see if the overlay has data for it.
	animTracks := s.getAnimationTracks(reanimComp)

	// Create a new overlay animation layer
	newLayer := components.AnimLayer{
		AnimName:          animName,
		CurrentFrame:      0,
		FrameAccumulator:  0.0,
		IsOneShot:         playOnce,
		IsFinished:        false,
		VisibleFrameCount: visibleCount,
		AnimVisibles:      animVisibles,
		AnimTracks:        animTracks,
	}

	// Add the overlay layer to the list
	reanimComp.OverlayAnims = append(reanimComp.OverlayAnims, newLayer)

	log.Printf("[ReanimSystem] PlayAnimationOverlay: Entity=%d, Overlay=%s, PlayOnce=%v, VisibleFrames=%d",
		entityID, animName, playOnce, visibleCount)

	return nil
}

// InitializeDirectRender initializes a ReanimComponent for direct rendering without animation definitions.
// This is used for entities like Sun that have only track definitions (no <anim> tags).
// All tracks will be rendered simultaneously, and all frames are visible.
//
// This method calculates CenterOffset to center the animation visually (suitable for grid-based entities).
//
// Parameters:
//   - entityID: the entity to initialize
//
// Returns:
//   - An error if the entity doesn't have a ReanimComponent
func (s *ReanimSystem) InitializeDirectRender(entityID ecs.EntityID) error {
	return s.initializeDirectRenderInternal(entityID, true)
}

// InitializeSceneAnimation initializes a ReanimComponent for scene animations.
// Scene animations (like SodRoll) have absolute coordinates defined in the reanim file,
// and do not need CenterOffset adjustment.
//
// Parameters:
//   - entityID: the entity to initialize
//
// Returns:
//   - An error if the entity doesn't have a ReanimComponent
func (s *ReanimSystem) InitializeSceneAnimation(entityID ecs.EntityID) error {
	return s.initializeDirectRenderInternal(entityID, false)
}

// initializeDirectRenderInternal is the internal implementation shared by both
// InitializeDirectRender and InitializeSceneAnimation.
//
// Parameters:
//   - entityID: the entity to initialize
//   - calculateCenter: whether to calculate CenterOffset (true for entities, false for scenes)
//
// Returns:
//   - An error if the entity doesn't have a ReanimComponent
func (s *ReanimSystem) initializeDirectRenderInternal(entityID ecs.EntityID, calculateCenter bool) error {
	// Get the ReanimComponent
	reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !exists {
		return fmt.Errorf("entity %d does not have a ReanimComponent", entityID)
	}

	// Check if Reanim data is present
	if reanimComp.Reanim == nil {
		return fmt.Errorf("entity %d has a ReanimComponent but no Reanim data", entityID)
	}

	log.Printf("[ReanimSystem] InitializeDirectRender for entity %d", entityID)

	// Set animation state (required for rendering)
	reanimComp.CurrentAnim = "direct_render" // Non-empty string to pass RenderSystem check
	reanimComp.CurrentFrame = 0
	reanimComp.FrameAccumulator = 0.0
	reanimComp.IsFinished = false

	// Calculate standard frame count (max frames across all tracks)
	standardFrameCount := 0
	for _, track := range reanimComp.Reanim.Tracks {
		if len(track.Frames) > standardFrameCount {
			standardFrameCount = len(track.Frames)
		}
	}
	if standardFrameCount == 0 {
		standardFrameCount = 1 // At least 1 frame
	}

	log.Printf("[ReanimSystem] Standard frame count: %d", standardFrameCount)

	// Build AnimVisibles: all frames are visible (all 0s)
	reanimComp.AnimVisibles = make([]int, standardFrameCount)
	for i := range reanimComp.AnimVisibles {
		reanimComp.AnimVisibles[i] = 0 // 0 = visible
	}

	// Set visible frame count
	reanimComp.VisibleFrameCount = standardFrameCount

	// Build merged tracks with frame inheritance
	reanimComp.MergedTracks = s.buildMergedTracks(reanimComp)

	// Store all tracks in rendering order
	reanimComp.AnimTracks = s.getAnimationTracks(reanimComp)

	log.Printf("[ReanimSystem] Built %d merged tracks, %d anim tracks", len(reanimComp.MergedTracks), len(reanimComp.AnimTracks))

	// Calculate center offset based on the bounding box of visible parts in the first frame
	// Skip this for scene animations (they have absolute coordinates)
	if calculateCenter {
		s.calculateCenterOffset(reanimComp)
	} else {
		// Scene animations: no center offset needed
		reanimComp.CenterOffsetX = 0
		reanimComp.CenterOffsetY = 0
		log.Printf("[ReanimSystem] Scene animation: CenterOffset set to (0, 0)")
	}

	return nil
}

// calculateCenterOffset calculates the offset needed to center the animation visually.
// It computes the bounding box of all visible parts in the first logical frame (frame 0),
// then calculates the center of that bounding box as the offset.
//
// IMPORTANT: This function now considers the actual image dimensions when calculating
// the bounding box, ensuring that the entire visual area of each part is included.
func (s *ReanimSystem) calculateCenterOffset(comp *components.ReanimComponent) {
	// Find the first visible frame (physical frame index)
	physicalIndex := -1
	logicalFrame := 0
	for i := 0; i < len(comp.AnimVisibles); i++ {
		if comp.AnimVisibles[i] == 0 {
			if logicalFrame == 0 {
				physicalIndex = i
				break
			}
			logicalFrame++
		}
	}

	if physicalIndex < 0 {
		// No visible frames, use zero offset
		comp.CenterOffsetX = 0
		comp.CenterOffsetY = 0
		return
	}

	// Calculate bounding box of all visible parts in the first frame
	// IMPORTANT: We now include the actual image dimensions, not just the position points
	minX, maxX := 9999.0, -9999.0
	minY, maxY := 9999.0, -9999.0
	hasVisibleParts := false

	for _, track := range comp.AnimTracks {
		// 如果设置了 VisibleTracks，只计算白名单中的轨道
		if comp.VisibleTracks != nil && len(comp.VisibleTracks) > 0 {
			if !comp.VisibleTracks[track.Name] {
				continue
			}
		}

		mergedFrames, ok := comp.MergedTracks[track.Name]
		if !ok || physicalIndex >= len(mergedFrames) {
			continue
		}

		frame := mergedFrames[physicalIndex]

		// Skip hidden frames (f=-1), UNLESS in VisibleTracks whitelist
		if frame.FrameNum != nil && *frame.FrameNum == -1 {
			// 检查是否在白名单中
			inVisibleTracks := false
			if comp.VisibleTracks != nil && len(comp.VisibleTracks) > 0 {
				inVisibleTracks = comp.VisibleTracks[track.Name]
			}
			if !inVisibleTracks {
				continue // 非白名单轨道，遵守 f=-1，跳过
			}
			// 白名单轨道，忽略 f=-1，继续计算边界
		}

		// Skip frames without images
		if frame.ImagePath == "" {
			continue
		}

		// Get part position
		x, y := 0.0, 0.0
		if frame.X != nil {
			x = *frame.X
		}
		if frame.Y != nil {
			y = *frame.Y
		}

		// Get part scale (default to 1.0)
		scaleX, scaleY := 1.0, 1.0
		if frame.ScaleX != nil {
			scaleX = *frame.ScaleX
		}
		if frame.ScaleY != nil {
			scaleY = *frame.ScaleY
		}

		// Get image dimensions
		img, exists := comp.PartImages[frame.ImagePath]
		if !exists || img == nil {
			// If image not found, fall back to position-only calculation
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
			hasVisibleParts = true
			continue
		}

		// Calculate actual bounding box including image dimensions
		bounds := img.Bounds()
		imgWidth := float64(bounds.Dx()) * scaleX
		imgHeight := float64(bounds.Dy()) * scaleY

		// The part's bounding box extends from (x, y) to (x + width, y + height)
		partMinX := x
		partMaxX := x + imgWidth
		partMinY := y
		partMaxY := y + imgHeight

		// Update overall bounding box
		if partMinX < minX {
			minX = partMinX
		}
		if partMaxX > maxX {
			maxX = partMaxX
		}
		if partMinY < minY {
			minY = partMinY
		}
		if partMaxY > maxY {
			maxY = partMaxY
		}
		hasVisibleParts = true
	}

	if !hasVisibleParts {
		// No visible parts, use zero offset
		comp.CenterOffsetX = 0
		comp.CenterOffsetY = 0
		return
	}

	// Calculate center of bounding box
	comp.CenterOffsetX = (minX + maxX) / 2
	comp.CenterOffsetY = (minY + maxY) / 2

	// DEBUG: 输出中心偏移量（用于调试动画切换时的位置跳动问题）
	log.Printf("[ReanimSystem] 动画 '%s' 中心偏移: (%.1f, %.1f), 包围盒: [%.1f, %.1f] -> [%.1f, %.1f]",
		comp.CurrentAnim, comp.CenterOffsetX, comp.CenterOffsetY, minX, minY, maxX, maxY)
}

// HideTrack hides a specific animation track (part) for the given entity.
// This is used for dynamic part visibility changes (e.g., zombie losing arms/head).
//
// Parameters:
//   - entityID: the ID of the entity
//   - trackName: the name of the track to hide (e.g., "Zombie_outerarm_hand")
//
// Returns:
//   - An error if the entity doesn't have a ReanimComponent or VisibleTracks is not initialized
func (s *ReanimSystem) HideTrack(entityID ecs.EntityID, trackName string) error {
	reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !exists {
		return fmt.Errorf("entity %d does not have a ReanimComponent", entityID)
	}

	// If VisibleTracks is not initialized, we can't hide tracks
	if reanimComp.VisibleTracks == nil {
		return fmt.Errorf("entity %d uses blacklist mode (VisibleTracks is nil), HideTrack not supported", entityID)
	}

	// Remove from visible tracks (whitelist mode)
	delete(reanimComp.VisibleTracks, trackName)
	return nil
}

// ShowTrack shows a specific animation track (part) for the given entity.
// This is used to restore previously hidden parts.
//
// Parameters:
//   - entityID: the ID of the entity
//   - trackName: the name of the track to show (e.g., "Zombie_outerarm_hand")
//
// Returns:
//   - An error if the entity doesn't have a ReanimComponent or VisibleTracks is not initialized
func (s *ReanimSystem) ShowTrack(entityID ecs.EntityID, trackName string) error {
	reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !exists {
		return fmt.Errorf("entity %d does not have a ReanimComponent", entityID)
	}

	// If VisibleTracks is not initialized, we can't show tracks
	if reanimComp.VisibleTracks == nil {
		return fmt.Errorf("entity %d uses blacklist mode (VisibleTracks is nil), ShowTrack not supported", entityID)
	}

	// Add to visible tracks (whitelist mode)
	reanimComp.VisibleTracks[trackName] = true
	return nil
}

// IsTrackVisible checks if a specific animation track (part) is currently visible.
//
// Parameters:
//   - entityID: the ID of the entity
//   - trackName: the name of the track to check
//
// Returns:
//   - bool: true if the track is visible, false otherwise
//   - error: error if the entity doesn't have a ReanimComponent
func (s *ReanimSystem) IsTrackVisible(entityID ecs.EntityID, trackName string) (bool, error) {
	reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !exists {
		return false, fmt.Errorf("entity %d does not have a ReanimComponent", entityID)
	}

	// If VisibleTracks is not initialized, assume blacklist mode (all visible by default)
	if reanimComp.VisibleTracks == nil || len(reanimComp.VisibleTracks) == 0 {
		return true, nil
	}

	// Whitelist mode: check if track is in the visible list
	return reanimComp.VisibleTracks[trackName], nil
}

// HidePartGroup hides a group of animation tracks defined in PartGroups.
// This provides a high-level semantic interface (e.g., "arm", "head") without
// requiring the caller to know specific track names.
//
// Parameters:
//   - entityID: the ID of the entity
//   - groupName: the name of the part group (e.g., "arm", "head", "armor")
//
// Returns:
//   - error: if the entity doesn't have a ReanimComponent, PartGroups is not configured,
//     or the group name doesn't exist
func (s *ReanimSystem) HidePartGroup(entityID ecs.EntityID, groupName string) error {
	reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !exists {
		return fmt.Errorf("entity %d does not have a ReanimComponent", entityID)
	}

	// Check if PartGroups is configured
	if reanimComp.PartGroups == nil {
		return fmt.Errorf("entity %d does not have PartGroups configured", entityID)
	}

	// Get the track list for this group
	tracks, ok := reanimComp.PartGroups[groupName]
	if !ok {
		return fmt.Errorf("part group '%s' not found in entity %d", groupName, entityID)
	}

	// Hide all tracks in the group
	for _, track := range tracks {
		if err := s.HideTrack(entityID, track); err != nil {
			// Log warning but continue (some tracks might not exist for all variants)
			log.Printf("[ReanimSystem] Warning: failed to hide track %s in group %s: %v", track, groupName, err)
		}
	}

	return nil
}

// ShowPartGroup shows a group of animation tracks defined in PartGroups.
// This is the reverse operation of HidePartGroup.
//
// Parameters:
//   - entityID: the ID of the entity
//   - groupName: the name of the part group (e.g., "arm", "head", "armor")
//
// Returns:
//   - error: if the entity doesn't have a ReanimComponent, PartGroups is not configured,
//     or the group name doesn't exist
func (s *ReanimSystem) ShowPartGroup(entityID ecs.EntityID, groupName string) error {
	reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !exists {
		return fmt.Errorf("entity %d does not have a ReanimComponent", entityID)
	}

	// Check if PartGroups is configured
	if reanimComp.PartGroups == nil {
		return fmt.Errorf("entity %d does not have PartGroups configured", entityID)
	}

	// Get the track list for this group
	tracks, ok := reanimComp.PartGroups[groupName]
	if !ok {
		return fmt.Errorf("part group '%s' not found in entity %d", groupName, entityID)
	}

	// Show all tracks in the group
	for _, track := range tracks {
		if err := s.ShowTrack(entityID, track); err != nil {
			log.Printf("[ReanimSystem] Warning: failed to show track %s in group %s: %v", track, groupName, err)
		}
	}

	return nil
}

// GetPartGroupImage 获取部件组中第一个可见部件的图片
// 用于创建掉落效果时获取部件的图片资源
//
// Parameters:
//   - entityID: the ID of the entity
//   - groupName: the name of the part group (e.g., "arm", "head")
//
// Returns:
//   - *ebiten.Image: the image of the first visible part in the group, or nil if not found
//   - error: if the entity doesn't have a ReanimComponent or the group doesn't exist
func (s *ReanimSystem) GetPartGroupImage(entityID ecs.EntityID, groupName string) (*ebiten.Image, error) {
	reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !exists {
		return nil, fmt.Errorf("entity %d does not have a ReanimComponent", entityID)
	}

	// Check if PartGroups is configured
	if reanimComp.PartGroups == nil {
		return nil, fmt.Errorf("entity %d does not have PartGroups configured", entityID)
	}

	// Get the track list for this group
	tracks, ok := reanimComp.PartGroups[groupName]
	if !ok {
		return nil, fmt.Errorf("part group '%s' not found in entity %d", groupName, entityID)
	}

	// Try to find any image from this part group
	// We iterate through all tracks in the group and try to find an image
	if reanimComp.PartImages != nil {
		for _, trackName := range tracks {
			// Look through MergedTracks to find the ImagePath for this track
			if mergedFrames, ok := reanimComp.MergedTracks[trackName]; ok && len(mergedFrames) > 0 {
				// Try the first frame that has an image
				for _, frame := range mergedFrames {
					if frame.ImagePath != "" {
						if img, exists := reanimComp.PartImages[frame.ImagePath]; exists && img != nil {
							return img, nil
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("no image found for part group '%s' in entity %d", groupName, entityID)
}

// RenderToTexture 将指定实体的 Reanim 渲染到目标纹理（离屏渲染）
// 用于生成植物卡片的预览图标
//
// 实现说明：
// 为了避免重复复杂的渲染逻辑，这个方法会临时创建一个 RenderSystem
// 并调用其 renderReanimEntity 方法渲染到目标纹理
//
// Parameters:
//   - entityID: the ID of the entity to render
//   - target: the target texture to render to (should be pre-created with appropriate size)
//
// Returns:
//   - error: if the entity doesn't have required components or rendering fails
func (s *ReanimSystem) RenderToTexture(entityID ecs.EntityID, target *ebiten.Image) error {
	// 验证实体拥有必要的组件
	_, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	_, hasReanim := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)

	if !hasPos || !hasReanim {
		return fmt.Errorf("entity %d missing required components for rendering", entityID)
	}

	// 创建临时的 RenderSystem 实例进行渲染
	// 注意：RenderSystem 不需要复杂初始化，只需要 EntityManager
	tempRenderSystem := NewRenderSystem(s.entityManager)

	// 渲染到目标纹理
	// cameraX = 0 因为我们渲染的是一个孤立的图标，不需要考虑摄像机
	tempRenderSystem.renderReanimEntity(target, entityID, 0)

	return nil
}

// GetTrackPosition 获取指定轨道在当前帧的世界坐标位置
// 用于定位游戏逻辑需要的特殊点位（如子弹发射点）
//
// Parameters:
//   - entityID: the ID of the entity
//   - trackName: the name of the track (e.g., "anim_stem")
//
// Returns:
//   - x, y: 轨道在世界坐标系中的位置（已应用实体位置和中心偏移）
//   - error: if the entity doesn't have required components or track doesn't exist
func (s *ReanimSystem) GetTrackPosition(entityID ecs.EntityID, trackName string) (float64, float64, error) {
	// 获取必要的组件
	pos, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	reanim, hasReanim := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)

	if !hasPos || !hasReanim {
		return 0, 0, fmt.Errorf("entity %d missing required components", entityID)
	}

	// 将逻辑帧映射到物理帧索引
	physicalIndex := s.findPhysicalFrameIndex(reanim, reanim.CurrentFrame)
	if physicalIndex < 0 {
		return 0, 0, fmt.Errorf("invalid frame index")
	}

	// 获取轨道的累积帧数据
	mergedFrames, ok := reanim.MergedTracks[trackName]
	if !ok || physicalIndex >= len(mergedFrames) {
		return 0, 0, fmt.Errorf("track '%s' not found or frame out of range", trackName)
	}

	frame := mergedFrames[physicalIndex]

	// 检查轨道在当前帧是否可见
	// 如果 frame.FrameNum 不为 nil 且值为 -1，表示轨道隐藏
	// 但是：如果轨道在 VisibleTracks 白名单中，跳过 f=-1 检查（强制可见）
	isInWhitelist := false
	if reanim.VisibleTracks != nil {
		isInWhitelist = reanim.VisibleTracks[trackName]
	}

	if !isInWhitelist && frame.FrameNum != nil && *frame.FrameNum == -1 {
		return 0, 0, fmt.Errorf("track '%s' is hidden at current frame %d", trackName, reanim.CurrentFrame)
	}

	// 获取轨道的局部位置（相对于动画原点）
	localX, localY := 0.0, 0.0
	if frame.X != nil {
		localX = *frame.X
	}
	if frame.Y != nil {
		localY = *frame.Y
	}

	// 如果X或Y为nil，说明轨道没有位置数据（可能是定位器轨道在未初始化状态）
	if frame.X == nil || frame.Y == nil {
		return 0, 0, fmt.Errorf("track '%s' has no position data at current frame %d", trackName, reanim.CurrentFrame)
	}

	// DEBUG: 输出世界坐标计算过程
	// log.Printf("[DEBUG] GetTrackPosition 坐标计算: localX=%.1f, pos.X=%.1f, CenterOffsetX=%.1f",
	// 	localX, pos.X, reanim.CenterOffsetX)

	// 转换为世界坐标
	// worldPos = entityPos + (trackLocalPos - centerOffset)
	worldX := pos.X + (localX - reanim.CenterOffsetX)
	worldY := pos.Y + (localY - reanim.CenterOffsetY)

	// log.Printf("[DEBUG] GetTrackPosition 结果: worldX = %.1f + (%.1f - %.1f) = %.1f",
	// 	pos.X, localX, reanim.CenterOffsetX, worldX)

	return worldX, worldY, nil
}

// findPhysicalFrameIndex 将逻辑帧号映射到物理帧索引
// 这是 RenderSystem 中同名方法的复制，因为需要在 ReanimSystem 中使用
func (s *ReanimSystem) findPhysicalFrameIndex(reanim *components.ReanimComponent, logicalFrameNum int) int {
	if len(reanim.AnimVisibles) == 0 {
		return -1
	}

	// 逻辑帧按区间映射：从第一个0开始到下一个非0之前
	logicalIndex := 0
	for i := 0; i < len(reanim.AnimVisibles); i++ {
		if reanim.AnimVisibles[i] == 0 {
			if logicalIndex == logicalFrameNum {
				return i
			}
			logicalIndex++
		}
	}

	return -1
}

// PrepareStaticPreview prepares a Reanim entity for static preview (e.g., plant card icons).
//
// This method is specifically designed for static preview scenarios (plant cards, almanac, shop),
// as opposed to PlayAnimation which is for dynamic playback.
//
// Key differences from PlayAnimation:
// - PlayAnimation: requires animation definition tracks, used for dynamic playback
// - PrepareStaticPreview: works with part tracks only, used for static rendering
//
// Strategy:
// 1. Does not depend on animation definition tracks, directly analyzes all part tracks
// 2. Finds the "first complete visible frame" (all parts have images and f>=0)
// 3. If not found, uses heuristic strategy (middle of animation, ~40% position)
// 4. Checks config.PlantPreviewFrameOverride for manual override (Story 11.1 - Strategy 3)
// 5. Sets static preview state (IsLooping=false, IsFinished=true)
//
// Parameters:
//   - entityID: the ID of the entity to prepare for static preview
//   - reanimName: the Reanim resource name (e.g., "SunFlower", "PeaShooterSingle")
//
// Returns:
//   - An error if the entity doesn't have a ReanimComponent
func (s *ReanimSystem) PrepareStaticPreview(entityID ecs.EntityID, reanimName string) error {
	// Get the ReanimComponent
	reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !exists {
		return fmt.Errorf("entity %d does not have a ReanimComponent", entityID)
	}

	// Check if Reanim data is present
	if reanimComp.Reanim == nil {
		return fmt.Errorf("entity %d has a ReanimComponent but no Reanim data", entityID)
	}

	// 1. Build merged tracks for preview (does not depend on animation definition track)
	reanimComp.MergedTracks = s.buildMergedTracksForPreview(reanimComp)

	// Store all part tracks for rendering
	reanimComp.AnimTracks = s.getPartTracks(reanimComp)

	// 2. Strategy 1: Find the first complete visible frame
	bestFrame := s.findFirstCompleteVisibleFrame(reanimComp)

	log.Printf("[ReanimSystem] findFirstCompleteVisibleFrame for %s: bestFrame=%d", reanimName, bestFrame)

	// DEBUG: 输出所有轨道在 bestFrame 的状态
	if bestFrame >= 0 {
		log.Printf("[ReanimSystem] Tracks at bestFrame %d:", bestFrame)
		for trackName, frames := range reanimComp.MergedTracks {
			if bestFrame < len(frames) {
				frame := frames[bestFrame]
				fValue := -999
				if frame.FrameNum != nil {
					fValue = *frame.FrameNum
				}
				log.Printf("  - %s: ImagePath=%s, f=%d", trackName, frame.ImagePath, fValue)
			}
		}
	}

	// 3. Strategy 2: If not found, use heuristic fallback
	if bestFrame < 0 {
		bestFrame = s.findPreviewFrameHeuristic(reanimComp)
		log.Printf("[ReanimSystem] No complete frame found, using heuristic frame %d", bestFrame)
	}

	// 4. Strategy 3: Check config override (manual frame specification)
	if overrideFrame, hasOverride := config.PlantPreviewFrameOverride[reanimName]; hasOverride {
		log.Printf("[ReanimSystem] Using config override frame %d for %s (auto-selected was %d)",
			overrideFrame, reanimName, bestFrame)
		bestFrame = overrideFrame
	}

	// 5. Apply preview frame
	reanimComp.BestPreviewFrame = bestFrame

	// 6. Calculate center offset for this specific frame
	s.calculateCenterOffsetForFrame(reanimComp, bestFrame)

	// 7. Build AnimVisibles array for static preview
	// IMPORTANT: We need to ensure that CurrentFrame (logical) maps to bestFrame (physical).
	// Strategy: Mark all frames as hidden (-1) except the bestFrame as visible (0).
	// This way, logical frame 0 will map to physical frame bestFrame.
	maxFrames := 0
	for _, frames := range reanimComp.MergedTracks {
		if len(frames) > maxFrames {
			maxFrames = len(frames)
		}
	}

	// Create AnimVisibles array with only bestFrame marked as visible
	reanimComp.AnimVisibles = make([]int, maxFrames)
	for i := 0; i < maxFrames; i++ {
		if i == bestFrame {
			reanimComp.AnimVisibles[i] = 0 // Best frame is visible
		} else {
			reanimComp.AnimVisibles[i] = -1 // Other frames are hidden
		}
	}
	reanimComp.VisibleFrameCount = 1 // Only one frame is visible

	// 8. Set static preview state (do not start animation loop)
	reanimComp.IsLooping = false
	reanimComp.IsFinished = true
	reanimComp.CurrentAnim = "static_preview" // Marker for static preview mode
	reanimComp.CurrentFrame = 0               // Logical frame 0 maps to physical frame bestFrame

	log.Printf("[ReanimSystem] PrepareStaticPreview: bestFrame=%d (physical), logicalFrame=0, totalFrames=%d",
		bestFrame, maxFrames)

	return nil
}

// getPartTracks returns all part tracks (tracks with images).
//
// This excludes pure animation definition tracks (only FrameNum, no images/transforms).
// Part tracks include:
// - Part tracks with images: backleaf, stalk_bottom, head, etc.
// - Hybrid tracks with images + transforms: some anim_* tracks in certain plants
//
// Parameters:
//   - reanimComp: the ReanimComponent containing the Reanim data
//
// Returns:
//   - A slice of tracks that have at least one frame with an image
func (s *ReanimSystem) getPartTracks(reanimComp *components.ReanimComponent) []reanim.Track {
	if reanimComp.Reanim == nil {
		return nil
	}

	var result []reanim.Track
	for _, track := range reanimComp.Reanim.Tracks {
		// Check if this track has at least one frame with an image
		hasImage := false
		for _, frame := range track.Frames {
			if frame.ImagePath != "" {
				hasImage = true
				break
			}
		}

		if hasImage {
			result = append(result, track)
		}
	}
	return result
}

// buildMergedTracksForPreview builds merged frame arrays for all part tracks WITHOUT depending on animation definition tracks.
//
// This differs from buildMergedTracks in that:
// - buildMergedTracks: used for dynamic playback, depends on animation definition track visibility
// - buildMergedTracksForPreview: used for static preview, directly processes all part tracks with frame inheritance
//
// Parameters:
//   - reanimComp: the ReanimComponent containing the Reanim data
//
// Returns:
//   - A map of track name to merged frame array
//
// Design Decision (Story 11.1 - QA Feedback):
// This method directly calls buildMergedTracks, which is SAFE and CORRECT because:
//
// 1. buildMergedTracks processes ALL tracks in the Reanim file, not just animation definition tracks
// 2. For each track, it applies frame inheritance (cumulative transformations) to build merged frames
// 3. The merged frames include all part tracks (with images) AND animation definition tracks (frame numbers only)
// 4. Static preview only USES the part tracks (filtered by VisibleTracks whitelist during rendering)
// 5. Animation definition tracks in merged data are harmless - they are simply ignored during rendering
//
// The alternative (implementing a separate buildMergedTracksForPreview that filters out animation
// definition tracks) would be UNNECESSARY complexity because:
// - It duplicates ~50 lines of frame inheritance logic
// - The filtering already happens at render time via VisibleTracks whitelist
// - Performance impact is negligible (few extra map entries)
//
// This design follows the DRY principle and maintains consistency with the existing animation system.
func (s *ReanimSystem) buildMergedTracksForPreview(reanimComp *components.ReanimComponent) map[string][]reanim.Frame {
	// Reuse the existing buildMergedTracks logic, which already processes ALL tracks
	// including part tracks, regardless of animation definition tracks
	return s.buildMergedTracks(reanimComp)
}

// findFirstCompleteVisibleFrame finds the first frame where all parts are visible.
//
// A "complete visible frame" is defined as:
// - All part tracks have data at this frame
// - Each part has an image (ImagePath != "")
// - Each part is not hidden (f >= 0)
//
// Parameters:
//   - reanimComp: the ReanimComponent containing the merged tracks
//
// Returns:
//   - The frame index of the first complete frame, or -1 if not found
func (s *ReanimSystem) findFirstCompleteVisibleFrame(reanimComp *components.ReanimComponent) int {
	if len(reanimComp.MergedTracks) == 0 {
		return -1
	}

	// Determine max frame count
	maxFrames := 0
	for _, frames := range reanimComp.MergedTracks {
		if len(frames) > maxFrames {
			maxFrames = len(frames)
		}
	}

	if maxFrames == 0 {
		return -1
	}

	// Get all RENDERABLE part track names (exclude empty tracks, logical tracks, and definition tracks)
	// Also consider VisibleTracks whitelist if set
	var partTrackNames []string
	for trackName, frames := range reanimComp.MergedTracks {
		if len(frames) == 0 {
			continue
		}

		// Skip logical tracks (no images, like anim_stem, _ground)
		if LogicalTracks[trackName] {
			continue
		}

		// Check if track has at least one frame with an image
		hasImage := false
		for _, frame := range frames {
			if frame.ImagePath != "" {
				hasImage = true
				break
			}
		}
		if !hasImage {
			continue // Skip tracks without images (pure animation definition tracks)
		}

		// If VisibleTracks is set, only include whitelisted tracks
		if reanimComp.VisibleTracks != nil && len(reanimComp.VisibleTracks) > 0 {
			if !reanimComp.VisibleTracks[trackName] {
				continue // Not in whitelist, skip
			}
		}

		partTrackNames = append(partTrackNames, trackName)
	}

	// Iterate through frames to find the first complete one
	for frameIdx := 0; frameIdx < maxFrames; frameIdx++ {
		allPartsVisible := true

		for _, trackName := range partTrackNames {
			mergedFrames := reanimComp.MergedTracks[trackName]
			if frameIdx >= len(mergedFrames) {
				allPartsVisible = false
				break
			}

			frame := mergedFrames[frameIdx]

			// Check if part has an image (should always be true due to filtering above, but double-check)
			if frame.ImagePath == "" {
				allPartsVisible = false
				break
			}

			// Check if part is not hidden (f != -1)
			// Exception: if track is in VisibleTracks whitelist, ignore f=-1
			if frame.FrameNum != nil && *frame.FrameNum == -1 {
				// Check if in whitelist
				inWhitelist := false
				if reanimComp.VisibleTracks != nil && len(reanimComp.VisibleTracks) > 0 {
					inWhitelist = reanimComp.VisibleTracks[trackName]
				}
				if !inWhitelist {
					allPartsVisible = false
					break
				}
				// In whitelist, ignore f=-1, continue
			}
		}

		if allPartsVisible {
			return frameIdx
		}
	}

	return -1
}

// findPreviewFrameHeuristic selects a preview frame using heuristic strategy.
//
// Strategy: Choose the frame at ~40% of the animation length.
//
// Rationale:
// - Animation structure pattern:
//   - First 10%: fade-in/preparation (some parts may be invisible)
//   - Middle 30-60%: core action (relatively stable)
//   - Last part: fade-out/transition
//
// - 40% position is usually in the stable region of the core action
//
// Parameters:
//   - reanimComp: the ReanimComponent containing the merged tracks
//
// Returns:
//   - The frame index at ~40% of the animation length, or 0 if no frames exist
func (s *ReanimSystem) findPreviewFrameHeuristic(reanimComp *components.ReanimComponent) int {
	if len(reanimComp.MergedTracks) == 0 {
		return 0
	}

	// Determine max frame count
	maxFrames := 0
	for _, frames := range reanimComp.MergedTracks {
		if len(frames) > maxFrames {
			maxFrames = len(frames)
		}
	}

	if maxFrames == 0 {
		return 0
	}

	// Choose frame at 40% position
	heuristicFrame := int(float64(maxFrames) * 0.4)

	// Ensure frame is within bounds
	if heuristicFrame >= maxFrames {
		heuristicFrame = maxFrames - 1
	}
	if heuristicFrame < 0 {
		heuristicFrame = 0
	}

	return heuristicFrame
}

// calculateCenterOffsetForFrame calculates the center offset for a specific frame.
//
// This is similar to calculateCenterOffset, but allows specifying which frame to use
// instead of always using the first frame. This is needed for static previews where
// we want to center based on the selected preview frame.
//
// Parameters:
//   - comp: the ReanimComponent containing the merged tracks
//   - frameIndex: the frame index to calculate center offset for
func (s *ReanimSystem) calculateCenterOffsetForFrame(comp *components.ReanimComponent, frameIndex int) {
	// Calculate bounding box of all visible parts in the specified frame
	minX, maxX := 9999.0, -9999.0
	minY, maxY := 9999.0, -9999.0
	hasVisibleParts := false

	for _, track := range comp.AnimTracks {
		// If VisibleTracks is set, only calculate for whitelisted tracks
		if comp.VisibleTracks != nil && len(comp.VisibleTracks) > 0 {
			if !comp.VisibleTracks[track.Name] {
				continue
			}
		}

		mergedFrames, ok := comp.MergedTracks[track.Name]
		if !ok || frameIndex >= len(mergedFrames) {
			continue
		}

		frame := mergedFrames[frameIndex]

		// Skip hidden frames (f=-1), UNLESS in VisibleTracks whitelist
		if frame.FrameNum != nil && *frame.FrameNum == -1 {
			// Check if in whitelist
			inVisibleTracks := false
			if comp.VisibleTracks != nil && len(comp.VisibleTracks) > 0 {
				inVisibleTracks = comp.VisibleTracks[track.Name]
			}
			if !inVisibleTracks {
				continue // Non-whitelisted track, respect f=-1, skip
			}
			// Whitelisted track, ignore f=-1, continue calculation
		}

		// Skip frames without images
		if frame.ImagePath == "" {
			continue
		}

		// Get part position
		x, y := 0.0, 0.0
		if frame.X != nil {
			x = *frame.X
		}
		if frame.Y != nil {
			y = *frame.Y
		}

		// Get part scale (default to 1.0)
		scaleX, scaleY := 1.0, 1.0
		if frame.ScaleX != nil {
			scaleX = *frame.ScaleX
		}
		if frame.ScaleY != nil {
			scaleY = *frame.ScaleY
		}

		// Get image dimensions
		img, exists := comp.PartImages[frame.ImagePath]
		if !exists || img == nil {
			// If image not found, fall back to position-only calculation
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
			hasVisibleParts = true
			continue
		}

		// Calculate actual bounding box including image dimensions
		bounds := img.Bounds()
		imgWidth := float64(bounds.Dx()) * scaleX
		imgHeight := float64(bounds.Dy()) * scaleY

		// Calculate bounding box corners
		// IMPORTANT: Reanim images have anchor point at TOP-LEFT corner (0,0)
		// NOT at center! (See render_system.go line 400-404)
		left := x
		right := x + imgWidth
		top := y
		bottom := y + imgHeight

		// Update bounding box
		if left < minX {
			minX = left
		}
		if right > maxX {
			maxX = right
		}
		if top < minY {
			minY = top
		}
		if bottom > maxY {
			maxY = bottom
		}

		hasVisibleParts = true
	}

	if !hasVisibleParts {
		// No visible parts, use zero offset
		comp.CenterOffsetX = 0
		comp.CenterOffsetY = 0
		return
	}

	// Calculate center offset
	centerX := (minX + maxX) / 2
	centerY := (minY + maxY) / 2

	comp.CenterOffsetX = centerX
	comp.CenterOffsetY = centerY

	log.Printf("[ReanimSystem] 计算中心偏移（帧%d） - 边界框: X[%.1f, %.1f], Y[%.1f, %.1f], 中心偏移: (%.1f, %.1f)",
		frameIndex, minX, maxX, minY, maxY, centerX, centerY)
}

// GetTrackTransform 获取指定轨道的当前变换矩阵（局部坐标）
//
// Story 10.5: 用于动画帧事件监听，获取部件实时位置
//
// 参数：
//   - entityID: 实体 ID
//   - trackName: 轨道名称（如 "idle_mouth", "anim_stem"）
//
// 返回：
//   - x, y: 轨道当前帧的局部坐标（相对于实体中心）
//   - error: 如果实体无动画组件或轨道不存在
//
// 使用场景：
//   - 子弹发射：在关键帧获取嘴部位置，精确创建子弹
//   - 特效锚点：在动画特定帧创建粒子效果
//   - 碰撞检测：获取部件实时位置进行精确碰撞判定
//
// 注意：
//   - 返回的是局部坐标，需要加上实体世界坐标才能得到最终位置
//   - 如果轨道在当前动画中不存在，返回错误
func (rs *ReanimSystem) GetTrackTransform(entityID ecs.EntityID, trackName string) (x, y float64, err error) {
	// 获取 Reanim 组件
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](rs.entityManager, entityID)
	if !ok {
		return 0, 0, fmt.Errorf("entity %d does not have ReanimComponent", entityID)
	}

	// 使用 MergedTracks（已合并帧继承的轨道）
	mergedFrames, ok := reanim.MergedTracks[trackName]
	if !ok {
		return 0, 0, fmt.Errorf("track '%s' not found in animation '%s'", trackName, reanim.CurrentAnim)
	}

	// 获取当前逻辑帧号
	currentFrame := reanim.CurrentFrame
	if currentFrame < 0 || currentFrame >= len(mergedFrames) {
		// 帧号越界，使用最后一帧
		currentFrame = len(mergedFrames) - 1
		if currentFrame < 0 {
			return 0, 0, fmt.Errorf("track '%s' has no frames", trackName)
		}
	}

	// 获取当前帧的变换
	frame := mergedFrames[currentFrame]

	// 提取坐标（默认为 0, 0）
	x = 0.0
	y = 0.0
	if frame.X != nil {
		x = *frame.X
	}
	if frame.Y != nil {
		y = *frame.Y
	}

	return x, y, nil
}
