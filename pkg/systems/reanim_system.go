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

// ReanimSystem is the Reanim animation system that manages skeletal animations
// for entities with ReanimComponent.
//
// This system is responsible for:
// - Advancing animation frames based on FPS
// - Implementing frame inheritance (cumulative transformations)
// - Managing animation loops
// - Supporting two playback modes: synchronous (GlobalFrame) and asynchronous (per-animation Frame)
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
// Story 13.2: 统一的帧推进逻辑，不再区分同步/异步模式
// 所有动画使用独立的 AnimState.LogicalFrame
//
// Parameters:
//   - deltaTime: time elapsed since last update (in seconds)
func (s *ReanimSystem) Update(deltaTime float64) {
	// Query all entities with ReanimComponent
	entities := ecs.GetEntitiesWith1[*components.ReanimComponent](s.entityManager)

	for _, id := range entities {
		// Get the ReanimComponent
		reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, id)
		if !exists {
			continue
		}

		// Skip if no Reanim data
		if reanimComp.Reanim == nil {
			continue
		}

		// Skip if animation is paused
		if reanimComp.IsPaused {
			continue
		}

		// ✅ 统一的帧推进逻辑（Story 13.2）
		s.updateAnimationStates(reanimComp, deltaTime)
	}
}

// updateAnimationStates 更新所有动画状态的独立帧索引（Story 13.2）
//
// 每个动画维护自己的 LogicalFrame、Accumulator 和循环状态
//
// Parameters:
//   - comp: ReanimComponent to update
//   - deltaTime: time elapsed since last update (in seconds)
func (s *ReanimSystem) updateAnimationStates(comp *components.ReanimComponent, deltaTime float64) {
	frameTime := 1.0 / float64(comp.Reanim.FPS)

	for _, state := range comp.AnimStates {
		// 跳过非激活动画
		if !state.IsActive {
			// 处理延迟逻辑（如果需要）
			if state.DelayDuration > 0 {
				state.DelayTimer += deltaTime
				if state.DelayTimer >= state.DelayDuration {
					state.IsActive = true
					state.DelayTimer = 0
					state.LogicalFrame = state.StartFrame
				}
			}
			continue
		}

		// 更新帧累加器
		state.Accumulator += deltaTime

		// 推进帧
		for state.Accumulator >= frameTime {
			state.Accumulator -= frameTime
			state.LogicalFrame++

			// 检查循环
			endFrame := state.StartFrame + state.FrameCount
			if state.LogicalFrame >= endFrame {
				if state.IsLooping {
					state.LogicalFrame = state.StartFrame
					// 如果有延迟，停止并重置延迟计时器
					if state.DelayDuration > 0 {
						state.IsActive = false
						state.DelayTimer = 0
					}
				} else {
					state.LogicalFrame = endFrame - 1
					state.IsActive = false
					break
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

// PlayAnimation starts playing the specified animation for the given entity (single animation mode).
//
// Story 13.2: 重构为只管理 AnimStates，移除 GlobalFrame 设置
//
// For playing multiple animations simultaneously, use PlayAnimations() instead.
// For adding animations incrementally, use AddAnimation() instead.
//
// This method:
// - Clears all existing animations (AnimStates map)
// - Adds the new animation via addAnimation()
// - Builds merged tracks and animation tracks
// - Calculates center offset and best preview frame
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

	// ✅ Story 13.2: 移除 GlobalFrame 和 CurrentFrame 设置
	// 设置基本动画状态
	reanimComp.FrameAccumulator = 0.0
	reanimComp.CurrentAnim = animName
	reanimComp.CurrentAnimations = []string{animName}
	reanimComp.IsLooping = true
	reanimComp.IsFinished = false

	// Build merged tracks with frame inheritance (required for rendering)
	reanimComp.MergedTracks = reanim.BuildMergedTracks(reanimComp.Reanim)

	// Clear all animations and add the new one
	reanimComp.AnimStates = make(map[string]*components.AnimState)
	if err := s.addAnimation(reanimComp, animName, true); err != nil {
		return err
	}

	// 单动画时清空 TrackBindings（使用默认行为）
	reanimComp.TrackBindings = nil

	// Store animation tracks in rendering order
	reanimComp.AnimTracks = s.getAnimationTracks(reanimComp)

	// Calculate center offset based on the bounding box of visible parts
	// Skip if FixedCenterOffset is true (prevents position jumping when switching animations)
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
		animVisibles := reanimComp.AnimVisiblesMap[animName]
		if frameIdx < len(animVisibles) && animVisibles[frameIdx] == -1 {
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

// ==================================================================
// Story 6.9: Multi-Animation Overlay API (多动画叠加 API)
// ==================================================================

// addAnimation is an internal helper method that adds a single animation to the Anims map.
//
// This method:
// 1. Validates that the animation exists in the Reanim data
// 2. Initializes the Anims map if needed
// 3. Builds AnimVisiblesMap if not already present
// 4. Creates an AnimState and adds it to the Anims map
// 5. Updates VisibleFrameCount for GlobalFrame loop control
//
// Parameters:
//   - comp: the ReanimComponent to modify
//   - animName: the name of the animation to add (e.g., "anim_idle", "anim_shooting")
//   - isActive: whether the animation should start active (controls frame advancement)
//
// Returns:
//   - An error if the animation doesn't exist in the Reanim data
func (s *ReanimSystem) addAnimation(
	comp *components.ReanimComponent,
	animName string,
	isActive bool,
) error {
	// Validate animation exists
	animTrack := s.getAnimDefinitionTrack(comp, animName)
	if animTrack == nil {
		return fmt.Errorf("animation '%s' not found in Reanim data", animName)
	}

	// Initialize Anims map if needed
	if comp.AnimStates == nil {
		comp.AnimStates = make(map[string]*components.AnimState)
	}

	// Initialize AnimVisiblesMap if needed
	if comp.AnimVisiblesMap == nil {
		comp.AnimVisiblesMap = make(map[string][]int)
	}

	// Build AnimVisiblesMap for this animation if not already present
	if comp.AnimVisiblesMap[animName] == nil {
		comp.AnimVisiblesMap[animName] = s.buildVisiblesArray(comp, animName)
	}

	// Calculate frame count for this animation
	frameCount := len(comp.AnimVisiblesMap[animName])

	// Create AnimState
	comp.AnimStates[animName] = &components.AnimState{
		Name:              animName,
		IsActive:          isActive,
		IsLooping:         true, // Default: animations loop
		LogicalFrame:      0,    // Story 13.2: 重命名自 Frame
		Accumulator:       0.0,
		StartFrame:        0,
		FrameCount:        frameCount,
		RenderWhenStopped: true, // Default: continue rendering when stopped
		DelayTimer:        0.0,
		DelayDuration:     0.0,
	}

	// Update VisibleFrameCount (used for GlobalFrame loop control in sync mode)
	// Use the maximum frame count among all animations
	if frameCount > comp.VisibleFrameCount {
		comp.VisibleFrameCount = frameCount
	}

	return nil
}

// PlayAnimations plays multiple animations simultaneously (multi-animation mode).
//
// Story 13.2: 重构为只管理 AnimStates，移除 GlobalFrame 设置
// Story 13.1: 多动画时自动分析轨道绑定
//
// This method clears all existing animations before adding the new ones.
//
// Parameters:
//   - entityID: the ID of the entity
//   - animNames: slice of animation names to play (e.g., []string{"anim_shooting", "anim_head_idle"})
//
// Returns:
//   - An error if the entity doesn't have a ReanimComponent or any animation doesn't exist
//
// Example:
//
//	// Play both body and head animations for PeaShooter attack
//	rs.PlayAnimations(entityID, []string{"anim_shooting", "anim_head_idle"})
func (s *ReanimSystem) PlayAnimations(entityID ecs.EntityID, animNames []string) error {
	// Get the ReanimComponent
	reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !exists {
		return fmt.Errorf("entity %d does not have a ReanimComponent", entityID)
	}

	// Check if Reanim data is present
	if reanimComp.Reanim == nil {
		return fmt.Errorf("entity %d has a ReanimComponent but no Reanim data", entityID)
	}

	// Validate at least one animation is specified
	if len(animNames) == 0 {
		return fmt.Errorf("at least one animation name must be specified")
	}

	// ✅ Story 13.2: 移除 GlobalFrame 和 CurrentFrame 设置
	// 设置基本动画状态
	primaryAnimName := animNames[0]
	reanimComp.FrameAccumulator = 0.0
	reanimComp.CurrentAnim = primaryAnimName
	reanimComp.CurrentAnimations = animNames
	reanimComp.IsLooping = true
	reanimComp.IsFinished = false

	// 多动画支持说明（Story 13.1）
	if len(animNames) > 1 {
		log.Printf("[ReanimSystem] PlayAnimations 收到 %d 个动画，将使用 TrackBindings 机制进行轨道绑定",
			len(animNames))
	}

	// Build merged tracks with frame inheritance (required for rendering)
	reanimComp.MergedTracks = reanim.BuildMergedTracks(reanimComp.Reanim)

	// Clear all animations and add the new ones
	reanimComp.AnimStates = make(map[string]*components.AnimState)
	for _, animName := range animNames {
		if err := s.addAnimation(reanimComp, animName, true); err != nil {
			return fmt.Errorf("failed to add animation '%s': %w", animName, err)
		}
	}

	// ==================================================================
	// Story 13.1: Auto Track Binding (自动轨道绑定)
	// ==================================================================
	//
	// 多动画时自动分析轨道绑定
	if len(animNames) > 1 {
		bindings := s.AnalyzeTrackBinding(reanimComp, animNames)
		reanimComp.TrackBindings = bindings

		// 输出绑定结果（用于调试）
		log.Printf("[ReanimSystem] 自动轨道绑定 (entity %d):", entityID)
		for track, anim := range bindings {
			log.Printf("  - %s -> %s", track, anim)
		}
	} else {
		// 单个动画时，清空绑定（使用默认行为）
		reanimComp.TrackBindings = nil
	}

	// Store animation tracks in rendering order
	reanimComp.AnimTracks = s.getAnimationTracks(reanimComp)

	// Calculate center offset based on the bounding box of visible parts
	if !reanimComp.FixedCenterOffset {
		s.calculateCenterOffset(reanimComp)
	}

	// Calculate best preview frame
	// Use the primary animation for preview calculation
	bestFrame := 0
	maxVisibleParts := 0

	for frameIdx := 0; frameIdx < reanimComp.VisibleFrameCount; frameIdx++ {
		// Skip invisible frames
		animVisibles := reanimComp.AnimVisiblesMap[primaryAnimName]
		if frameIdx < len(animVisibles) && animVisibles[frameIdx] == -1 {
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

	log.Printf("[ReanimSystem] PlayAnimations: entity %d playing %d animations: %v",
		entityID, len(animNames), animNames)

	return nil
}

// AddAnimation adds an animation to the currently playing animations (incremental mode).
//
// Story 6.9: Enables adding animations without clearing existing ones.
// This is useful for dynamically layering effects (e.g., adding a burning effect on top of walk animation).
//
// Unlike PlayAnimation/PlayAnimations which clear all animations first, this method
// preserves existing animations and adds a new one.
//
// Parameters:
//   - entityID: the ID of the entity
//   - animName: the name of the animation to add (e.g., "anim_burning")
//
// Returns:
//   - An error if the entity doesn't have a ReanimComponent or the animation doesn't exist
//
// Example:
//
//	// Start with walk animation
//	rs.PlayAnimation(entityID, "anim_walk")
//	// Add burning effect on top
//	rs.AddAnimation(entityID, "anim_burning")
func (s *ReanimSystem) AddAnimation(entityID ecs.EntityID, animName string) error {
	// Get the ReanimComponent
	reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !exists {
		return fmt.Errorf("entity %d does not have a ReanimComponent", entityID)
	}

	// Check if Reanim data is present
	if reanimComp.Reanim == nil {
		return fmt.Errorf("entity %d has a ReanimComponent but no Reanim data", entityID)
	}

	// Add the animation (preserves existing animations)
	if err := s.addAnimation(reanimComp, animName, true); err != nil {
		return fmt.Errorf("failed to add animation '%s': %w", animName, err)
	}

	log.Printf("[ReanimSystem] AddAnimation: entity %d added animation '%s' (total: %d)",
		entityID, animName, len(reanimComp.AnimStates))

	return nil
}

// RemoveAnimation removes a specific animation from the currently playing animations.
//
// Story 6.9: Enables removing individual animations without affecting others.
// This is useful for removing temporary effects (e.g., removing burning effect when it expires).
//
// Parameters:
//   - entityID: the ID of the entity
//   - animName: the name of the animation to remove (e.g., "anim_burning")
//
// Returns:
//   - An error if the entity doesn't have a ReanimComponent
//
// Note: It is safe to call this method even if the animation doesn't exist (no-op).
//
// Example:
//
//	// Remove burning effect
//	rs.RemoveAnimation(entityID, "anim_burning")
func (s *ReanimSystem) RemoveAnimation(entityID ecs.EntityID, animName string) error {
	// Get the ReanimComponent
	reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !exists {
		return fmt.Errorf("entity %d does not have a ReanimComponent", entityID)
	}

	// Remove from AnimStates map (safe even if animName doesn't exist)
	delete(reanimComp.AnimStates, animName)

	log.Printf("[ReanimSystem] RemoveAnimation: entity %d removed animation '%s' (remaining: %d)",
		entityID, animName, len(reanimComp.AnimStates))

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

	// Initialize AnimVisiblesMap if needed
	if reanimComp.AnimVisiblesMap == nil {
		reanimComp.AnimVisiblesMap = make(map[string][]int)
	}

	// Build AnimVisibles: all frames are visible (all 0s)
	animVisibles := make([]int, standardFrameCount)
	for i := range animVisibles {
		animVisibles[i] = 0 // 0 = visible
	}
	reanimComp.AnimVisiblesMap["direct_render"] = animVisibles

	// Story 6.8 修复：创建 AnimStates map（必需，否则 shouldRenderTrack 会因 len(activeAnims)==0 失败）
	if reanimComp.AnimStates == nil {
		reanimComp.AnimStates = make(map[string]*components.AnimState)
	}
	reanimComp.AnimStates["direct_render"] = &components.AnimState{
		Name:              "direct_render",
		IsActive:          true, // 必须为 true
		IsLooping:         true,
		LogicalFrame:      0,
		Accumulator:       0.0,
		StartFrame:        0,
		FrameCount:        standardFrameCount,
		RenderWhenStopped: true,
		DelayTimer:        0.0,
		DelayDuration:     0.0,
	}

	// Set visible frame count
	reanimComp.VisibleFrameCount = standardFrameCount

	// Build merged tracks with frame inheritance
	reanimComp.MergedTracks = reanim.BuildMergedTracks(reanimComp.Reanim)

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
	animVisibles := comp.AnimVisiblesMap[comp.CurrentAnim]
	for i := 0; i < len(animVisibles); i++ {
		if animVisibles[i] == 0 {
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
// Story 13.2: 使用主动画的 LogicalFrame 替代 CurrentFrame
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

	// Story 13.2: 获取主动画的 LogicalFrame
	mainAnimState, ok := reanim.AnimStates[reanim.CurrentAnim]
	if !ok {
		return 0, 0, fmt.Errorf("entity %d has no active animation '%s'", entityID, reanim.CurrentAnim)
	}

	currentLogicalFrame := mainAnimState.LogicalFrame

	// 将逻辑帧映射到物理帧索引
	physicalIndex := s.findPhysicalFrameIndex(reanim, currentLogicalFrame)
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
		return 0, 0, fmt.Errorf("track '%s' is hidden at current logical frame %d", trackName, currentLogicalFrame)
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
		return 0, 0, fmt.Errorf("track '%s' has no position data at current logical frame %d", trackName, currentLogicalFrame)
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
	animVisibles := reanim.AnimVisiblesMap[reanim.CurrentAnim]
	if len(animVisibles) == 0 {
		return -1
	}

	// 逻辑帧按区间映射：从第一个0开始到下一个非0之前
	logicalIndex := 0
	for i := 0; i < len(animVisibles); i++ {
		if animVisibles[i] == 0 {
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

	// Initialize AnimVisiblesMap if needed
	if reanimComp.AnimVisiblesMap == nil {
		reanimComp.AnimVisiblesMap = make(map[string][]int)
	}

	// Create AnimVisibles array with only bestFrame marked as visible
	animVisibles := make([]int, maxFrames)
	for i := 0; i < maxFrames; i++ {
		if i == bestFrame {
			animVisibles[i] = 0 // Best frame is visible
		} else {
			animVisibles[i] = -1 // Other frames are hidden
		}
	}
	reanimComp.AnimVisiblesMap["static_preview"] = animVisibles
	reanimComp.VisibleFrameCount = 1 // Only one frame is visible

	// 8. Set static preview state (do not start animation loop)
	reanimComp.IsLooping = false
	reanimComp.IsFinished = true
	reanimComp.CurrentAnim = "static_preview" // Marker for static preview mode

	log.Printf("[ReanimSystem] PrepareStaticPreview: bestFrame=%d (physical), totalFrames=%d",
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
	return reanim.BuildMergedTracks(reanimComp.Reanim)
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
// Story 13.2: 使用主动画的 LogicalFrame
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

	// Story 13.2: 获取主动画的 LogicalFrame
	mainAnimState, hasMainAnim := reanim.AnimStates[reanim.CurrentAnim]
	if !hasMainAnim {
		return 0, 0, fmt.Errorf("entity %d has no active animation '%s'", entityID, reanim.CurrentAnim)
	}

	currentFrame := mainAnimState.LogicalFrame
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

// ==================================================================
// Track-Level Playback Control (Story 12.1)
// ==================================================================

// SetTrackPlayOnce configures a track to play once and then lock at its final frame.
// This is used for one-time animations like tombstone rising or sign dropping.
//
// Parameters:
//   - entity: The entity ID with ReanimComponent
//   - trackName: Name of the track (e.g., "SelectorScreen_Adventure_button")
//
// The track will play normally until it reaches its last visible frame, then lock.
// Locked tracks will not update in subsequent frames.
func (rs *ReanimSystem) SetTrackPlayOnce(entity ecs.EntityID, trackName string) error {
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](rs.entityManager, entity)
	if !ok {
		return fmt.Errorf("entity %d has no ReanimComponent", entity)
	}

	// Initialize TrackConfigs map if needed
	if reanimComp.TrackConfigs == nil {
		reanimComp.TrackConfigs = make(map[string]*components.TrackPlaybackConfig)
	}

	// Create or update config
	if reanimComp.TrackConfigs[trackName] == nil {
		reanimComp.TrackConfigs[trackName] = &components.TrackPlaybackConfig{}
	}
	reanimComp.TrackConfigs[trackName].PlayOnce = true

	return nil
}

// PauseTrack pauses playback of a specific track.
// The track will stop updating but can be resumed later.
func (rs *ReanimSystem) PauseTrack(entity ecs.EntityID, trackName string) error {
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](rs.entityManager, entity)
	if !ok {
		return fmt.Errorf("entity %d has no ReanimComponent", entity)
	}

	// Initialize TrackConfigs map if needed
	if reanimComp.TrackConfigs == nil {
		reanimComp.TrackConfigs = make(map[string]*components.TrackPlaybackConfig)
	}

	// Create or update config
	if reanimComp.TrackConfigs[trackName] == nil {
		reanimComp.TrackConfigs[trackName] = &components.TrackPlaybackConfig{}
	}
	reanimComp.TrackConfigs[trackName].IsPaused = true

	return nil
}

// ResumeTrack resumes playback of a paused track.
func (rs *ReanimSystem) ResumeTrack(entity ecs.EntityID, trackName string) error {
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](rs.entityManager, entity)
	if !ok {
		return fmt.Errorf("entity %d has no ReanimComponent", entity)
	}

	if reanimComp.TrackConfigs == nil || reanimComp.TrackConfigs[trackName] == nil {
		return nil // No config, track is already playing
	}

	reanimComp.TrackConfigs[trackName].IsPaused = false
	return nil
}

// ==================================================================
// Story 13.1: Track Binding Helper Functions (轨道绑定辅助函数)
// ==================================================================

// getVisualTracks 获取所有有图片的轨道列表（视觉轨道）
//
// 视觉轨道定义：至少有一帧包含 ImagePath 的轨道
// 排除：逻辑轨道（如 anim_stem, _ground）、纯动画定义轨道（如 anim_idle）
//
// 参数：
//   - comp: ReanimComponent
//
// 返回：
//   - 视觉轨道名称列表
func (s *ReanimSystem) getVisualTracks(comp *components.ReanimComponent) []string {
	var visualTracks []string

	for trackName, mergedFrames := range comp.MergedTracks {
		// 跳过逻辑轨道
		if LogicalTracks[trackName] {
			continue
		}

		// 跳过动画定义轨道
		if AnimationDefinitionTracks[trackName] {
			continue
		}

		// 检查是否至少有一帧包含图片
		hasImage := false
		for _, frame := range mergedFrames {
			if frame.ImagePath != "" {
				hasImage = true
				break
			}
		}

		if hasImage {
			visualTracks = append(visualTracks, trackName)
		}
	}

	return visualTracks
}

// findVisibleWindow 查找动画的可见窗口（首个可见帧和末尾可见帧）
//
// 参数：
//   - animVisibles: 动画可见性数组（0 = 可见，-1 = 隐藏）
//
// 返回：
//   - firstVisible: 第一个可见帧的索引
//   - lastVisible: 最后一个可见帧的索引
//   - 如果动画完全不可见，返回 (-1, -1)
func (s *ReanimSystem) findVisibleWindow(animVisibles []int) (int, int) {
	firstVisible := -1
	lastVisible := -1

	for i, visibility := range animVisibles {
		if visibility == 0 {
			if firstVisible == -1 {
				firstVisible = i
			}
			lastVisible = i
		}
	}

	return firstVisible, lastVisible
}

// calculatePositionVariance 计算位置方差（用于衡量轨道运动幅度）
//
// 算法原理：
// 1. 计算指定帧范围内所有帧的平均位置（avgX, avgY）
// 2. 计算每帧位置与平均位置的欧氏距离平方和
// 3. 返回方差的平方根（标准差）
//
// 方差越大，说明轨道在该动画中运动越明显
//
// 参数：
//   - frames: 轨道的帧数组（MergedTracks）
//   - start: 起始帧索引
//   - end: 结束帧索引（包含）
//
// 返回：
//   - 位置方差（标准差）
func (s *ReanimSystem) calculatePositionVariance(frames []reanim.Frame, start, end int) float64 {
	if start < 0 || end >= len(frames) || start > end {
		return 0
	}

	// 计算平均位置
	avgX, avgY := 0.0, 0.0
	count := 0
	for i := start; i <= end && i < len(frames); i++ {
		if frames[i].X != nil && frames[i].Y != nil {
			avgX += *frames[i].X
			avgY += *frames[i].Y
			count++
		}
	}

	if count == 0 {
		return 0
	}

	avgX /= float64(count)
	avgY /= float64(count)

	// 计算方差
	variance := 0.0
	for i := start; i <= end && i < len(frames); i++ {
		if frames[i].X != nil && frames[i].Y != nil {
			dx := *frames[i].X - avgX
			dy := *frames[i].Y - avgY
			variance += dx*dx + dy*dy
		}
	}

	return variance / float64(count) // 返回方差（不开方，保持敏感度）
}

// ==================================================================
// Story 13.1: Track Binding Analysis API (轨道绑定分析 API)
// ==================================================================

// AnalyzeTrackBinding 自动分析轨道到动画的绑定关系
//
// 算法原理：
// 1. 对于每个视觉轨道，遍历所有动画
// 2. 计算轨道在该动画时间窗口内的位置方差（运动幅度）
// 3. 将轨道绑定到方差最大的动画（运动最明显 = 最可能属于该动画）
//
// 参数：
//   - comp: ReanimComponent
//   - animNames: 要分析的动画列表（如 ["anim_shooting", "anim_head_idle"]）
//
// 返回：
//   - map[string]string: 轨道绑定（轨道名 -> 动画名）
//
// 示例：
//
//	bindings := rs.AnalyzeTrackBinding(comp, []string{"anim_shooting", "anim_head_idle"})
//	// 可能返回：{"anim_face": "anim_head_idle", "stalk_bottom": "anim_shooting"}
func (s *ReanimSystem) AnalyzeTrackBinding(
	comp *components.ReanimComponent,
	animNames []string,
) map[string]string {
	bindings := make(map[string]string)

	// 获取所有视觉轨道（有图片的轨道）
	visualTracks := s.getVisualTracks(comp)

	for _, trackName := range visualTracks {
		mergedFrames, ok := comp.MergedTracks[trackName]
		if !ok || len(mergedFrames) == 0 {
			continue
		}

		bestAnim := ""
		bestScore := 0.0

		for _, animName := range animNames {
			animVisibles, hasAnim := comp.AnimVisiblesMap[animName]
			if !hasAnim || len(animVisibles) == 0 {
				continue
			}

			// 查找该动画的可见窗口
			firstVisible, lastVisible := s.findVisibleWindow(animVisibles)

			if firstVisible < 0 || lastVisible >= len(mergedFrames) {
				continue
			}

			// 检查轨道在该动画时间窗口内是否有图片
			hasImage := false
			for i := firstVisible; i <= lastVisible && i < len(mergedFrames); i++ {
				if mergedFrames[i].ImagePath != "" {
					hasImage = true
					break
				}
			}

			if !hasImage {
				continue
			}

			// 计算位置方差
			variance := s.calculatePositionVariance(mergedFrames, firstVisible, lastVisible)
			score := 1.0 + variance

			if score > bestScore {
				bestScore = score
				bestAnim = animName
			}
		}

		if bestAnim != "" {
			bindings[trackName] = bestAnim
		}
	}

	return bindings
}

// SetTrackBindings 手动设置轨道绑定关系
//
// 参数：
//   - entityID: 实体 ID
//   - bindings: 轨道绑定（轨道名 -> 动画名）
//
// 返回：
//   - error: 如果轨道或动画不存在
//
// 示例：
//
//	rs.SetTrackBindings(entityID, map[string]string{
//	    "anim_face": "anim_head_idle",
//	    "stalk_bottom": "anim_shooting",
//	})
func (s *ReanimSystem) SetTrackBindings(
	entityID ecs.EntityID,
	bindings map[string]string,
) error {
	reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !exists {
		return fmt.Errorf("entity %d does not have ReanimComponent", entityID)
	}

	// 验证绑定的有效性
	for trackName, animName := range bindings {
		// 检查轨道是否存在
		if _, ok := reanimComp.MergedTracks[trackName]; !ok {
			return fmt.Errorf("track '%s' does not exist", trackName)
		}

		// 检查动画是否存在
		if _, ok := reanimComp.AnimVisiblesMap[animName]; !ok {
			return fmt.Errorf("animation '%s' does not exist", animName)
		}
	}

	// 应用绑定
	reanimComp.TrackBindings = bindings

	return nil
}
