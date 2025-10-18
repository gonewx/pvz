package systems

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// ReanimSystem is the Reanim animation system that manages skeletal animations
// for entities with ReanimComponent.
//
// This system is responsible for:
// - Advancing animation frames based on FPS
// - Implementing frame inheritance (cumulative transformations)
// - Managing animation loops
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
	}
}

// getAnimDefinitionTrack returns the animation definition track for the given animation name.
//
// Animation definition tracks are tracks whose names start with "anim_" (e.g., "anim_idle", "anim_shooting").
// These tracks control the overall animation visibility and timing.
//
// Parameters:
//   - comp: the ReanimComponent containing the Reanim data
//   - animName: the name of the animation to find (e.g., "anim_idle")
//
// Returns:
//   - A pointer to the Track if found, nil otherwise
func (s *ReanimSystem) getAnimDefinitionTrack(comp *components.ReanimComponent, animName string) *reanim.Track {
	if comp.Reanim == nil {
		return nil
	}

	// Iterate through all tracks to find the one with the matching name
	for i := range comp.Reanim.Tracks {
		if comp.Reanim.Tracks[i].Name == animName {
			return &comp.Reanim.Tracks[i]
		}
	}

	return nil
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

	// Build visibility array
	reanimComp.AnimVisibles = s.buildVisiblesArray(reanimComp, animName)

	// Calculate visible frame count (number of frames with visibility 0)
	visibleCount := 0
	for _, v := range reanimComp.AnimVisibles {
		if v == 0 {
			visibleCount++
		}
	}
	reanimComp.VisibleFrameCount = visibleCount

	// Build merged tracks with frame inheritance
	reanimComp.MergedTracks = s.buildMergedTracks(reanimComp)

	// Store animation tracks in rendering order
	reanimComp.AnimTracks = s.getAnimationTracks(reanimComp)

	// Calculate center offset based on the bounding box of visible parts in the first frame
	s.calculateCenterOffset(reanimComp)

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

		// Skip hidden frames
		if frame.FrameNum != nil && *frame.FrameNum == -1 {
			continue
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

	// 调试输出：打印中心偏移计算结果
	log.Printf("[ReanimSystem] 计算中心偏移 - 边界框: X[%.1f, %.1f], Y[%.1f, %.1f], 中心偏移: (%.1f, %.1f)",
		minX, maxX, minY, maxY, comp.CenterOffsetX, comp.CenterOffsetY)
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
