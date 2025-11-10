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

	// Story 13.6: 配置管理器（用于配置驱动的动画播放）
	configManager *config.ReanimConfigManager
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
	// Story 13.6: DEPRECATED - 使用 PlayCombo() 或 PlayDefaultAnimation() 替代
	log.Printf("⚠️  [DEPRECATED] PlayAnimation() 已废弃，请使用 PlayCombo() 或 PlayDefaultAnimation()")

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
	// Story 13.6: DEPRECATED - 使用 PlayCombo() 替代
	log.Printf("⚠️  [DEPRECATED] PlayAnimations() 已废弃，请使用 PlayCombo()")

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

	// Story 13.4 QA Fix: 优先使用 AnimTracks 保证顺序（渲染 Z-order）
	// 修复 map 迭代顺序随机导致的缓存顺序错误和渲染顺序不一致
	if len(comp.AnimTracks) > 0 {
		for _, track := range comp.AnimTracks {
			trackName := track.Name

			// 跳过逻辑轨道
			if LogicalTracks[trackName] {
				continue
			}

			// 跳过动画定义轨道
			if AnimationDefinitionTracks[trackName] {
				continue
			}

			// 检查 MergedTracks 中是否有该轨道
			mergedFrames, exists := comp.MergedTracks[trackName]
			if !exists {
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

	// 降级：如果没有 AnimTracks，遍历 MergedTracks（顺序不确定）
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

// ==================================================================
// Story 13.3: Parent-Child Offset System API (父子偏移系统 API)
// ==================================================================

// SetParentTracks 设置实体的父子轨道关系（批量设置）
//
// 参数：
//   - entityID: 实体 ID
//   - parentTracks: 父子关系映射（map[子轨道]父轨道）
//
// 返回：
//   - error: 如果实体没有 ReanimComponent
//
// 示例：
//
//	rs.SetParentTracks(entityID, map[string]string{
//	    "anim_face": "anim_stem",  // 头部跟随茎干
//	})
func (s *ReanimSystem) SetParentTracks(
	entityID ecs.EntityID,
	parentTracks map[string]string,
) error {
	reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !exists {
		return fmt.Errorf("entity %d does not have ReanimComponent", entityID)
	}

	// 应用父子关系
	reanimComp.ParentTracks = parentTracks

	log.Printf("[ReanimSystem] SetParentTracks: entity %d, %d parent-child relationships configured",
		entityID, len(parentTracks))

	return nil
}

// SetParentTrack 设置单个轨道的父轨道
//
// 参数：
//   - entityID: 实体 ID
//   - childTrack: 子轨道名称
//   - parentTrack: 父轨道名称
//
// 返回：
//   - error: 如果实体没有 ReanimComponent
//
// 示例：
//
//	rs.SetParentTrack(entityID, "anim_face", "anim_stem")
func (s *ReanimSystem) SetParentTrack(
	entityID ecs.EntityID,
	childTrack, parentTrack string,
) error {
	reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !exists {
		return fmt.Errorf("entity %d does not have ReanimComponent", entityID)
	}

	// 初始化 ParentTracks map（如果需要）
	if reanimComp.ParentTracks == nil {
		reanimComp.ParentTracks = make(map[string]string)
	}

	// 设置父子关系
	reanimComp.ParentTracks[childTrack] = parentTrack

	log.Printf("[ReanimSystem] SetParentTrack: entity %d, '%s' -> '%s'",
		entityID, childTrack, parentTrack)

	return nil
}

// ==================================================================
// Story 13.3: Parent-Child Offset Calculation (父子偏移计算)
// ==================================================================

// getParentOffset 计算父轨道的当前偏移量（相对于初始位置）
//
// 算法原理：
// 1. 找到父轨道控制的动画（通过 TrackBindings）
// 2. 获取父轨道在该动画时间窗口内的第一个可见帧位置（初始位置）
// 3. 获取父轨道的当前位置
// 4. 计算偏移：offset = current - initial
//
// 参数：
//   - parentTrackName: 父轨道名称（如 "anim_stem"）
//   - comp: ReanimComponent 引用
//
// 返回：
//   - offsetX: X 轴偏移量
//   - offsetY: Y 轴偏移量
func (s *ReanimSystem) getParentOffset(parentTrackName string, comp *components.ReanimComponent) (float64, float64) {
	// 步骤 1: 找到父轨道控制的动画
	parentAnim, exists := comp.TrackBindings[parentTrackName]
	if !exists {
		// 父轨道未绑定动画，使用主动画
		parentAnim = comp.CurrentAnim
	}

	// 步骤 2: 获取父轨道的初始位置（第一个可见帧）
	initX, initY, err := s.getFirstVisiblePosition(parentTrackName, parentAnim, comp)
	if err != nil {
		// 父轨道没有初始位置，返回零偏移
		return 0, 0
	}

	// 步骤 3: 获取父轨道的当前位置
	currentX, currentY, err := s.getCurrentPosition(parentTrackName, parentAnim, comp)
	if err != nil {
		// 父轨道没有当前位置，返回零偏移
		return 0, 0
	}

	// 步骤 4: 计算偏移量
	offsetX := currentX - initX
	offsetY := currentY - initY

	// DEBUG: 输出偏移计算（仅在需要调试时取消注释）
	// log.Printf("[ReanimSystem] 父轨道 '%s' 偏移: 初始(%.1f, %.1f) -> 当前(%.1f, %.1f) = 偏移(%.1f, %.1f)",
	// 	parentTrackName, initX, initY, currentX, currentY, offsetX, offsetY)

	return offsetX, offsetY
}

// getFirstVisiblePosition 获取轨道在动画时间窗口内的第一个可见帧位置
//
// 参数：
//   - trackName: 轨道名称（如 "anim_stem"）
//   - animName: 动画名称（如 "anim_shooting"）
//   - comp: ReanimComponent 引用
//
// 返回：
//   - x, y: 第一个可见帧的位置
//   - error: 如果找不到可见帧或轨道不存在
func (s *ReanimSystem) getFirstVisiblePosition(
	trackName, animName string,
	comp *components.ReanimComponent,
) (float64, float64, error) {
	// 获取轨道的累积帧数据
	mergedFrames, ok := comp.MergedTracks[trackName]
	if !ok || len(mergedFrames) == 0 {
		return 0, 0, fmt.Errorf("track '%s' not found or has no frames", trackName)
	}

	// 获取动画的可见性数组
	animVisibles, ok := comp.AnimVisiblesMap[animName]
	if !ok || len(animVisibles) == 0 {
		return 0, 0, fmt.Errorf("animation '%s' has no visibility data", animName)
	}

	// 查找第一个可见帧（visibility = 0）
	for physicalIdx, visibility := range animVisibles {
		if visibility == 0 && physicalIdx < len(mergedFrames) {
			frame := mergedFrames[physicalIdx]

			// 检查帧是否有位置数据
			if frame.X == nil || frame.Y == nil {
				continue // 跳过没有位置数据的帧
			}

			return *frame.X, *frame.Y, nil
		}
	}

	return 0, 0, fmt.Errorf("track '%s' has no visible frames in animation '%s'", trackName, animName)
}

// getCurrentPosition 获取轨道的当前位置
//
// 参数：
//   - trackName: 轨道名称（如 "anim_stem"）
//   - animName: 动画名称（如 "anim_shooting"）
//   - comp: ReanimComponent 引用
//
// 返回：
//   - x, y: 当前帧的位置
//   - error: 如果轨道不存在或当前帧越界
func (s *ReanimSystem) getCurrentPosition(
	trackName, animName string,
	comp *components.ReanimComponent,
) (float64, float64, error) {
	// 获取动画状态
	animState, ok := comp.AnimStates[animName]
	if !ok {
		return 0, 0, fmt.Errorf("animation '%s' is not active", animName)
	}

	logicalFrame := animState.LogicalFrame

	// 获取动画的可见性数组
	animVisibles, ok := comp.AnimVisiblesMap[animName]
	if !ok || len(animVisibles) == 0 {
		return 0, 0, fmt.Errorf("animation '%s' has no visibility data", animName)
	}

	// 将逻辑帧映射到物理帧
	physicalFrame := s.mapLogicalToPhysical(logicalFrame, animVisibles)
	if physicalFrame < 0 {
		return 0, 0, fmt.Errorf("invalid logical frame %d for animation '%s'", logicalFrame, animName)
	}

	// 获取轨道的累积帧数据
	mergedFrames, ok := comp.MergedTracks[trackName]
	if !ok || len(mergedFrames) == 0 {
		return 0, 0, fmt.Errorf("track '%s' not found or has no frames", trackName)
	}

	// 检查物理帧是否越界
	if physicalFrame >= len(mergedFrames) {
		return 0, 0, fmt.Errorf("physical frame %d out of range for track '%s' (len=%d)",
			physicalFrame, trackName, len(mergedFrames))
	}

	frame := mergedFrames[physicalFrame]

	// 检查帧是否有位置数据
	if frame.X == nil || frame.Y == nil {
		return 0, 0, fmt.Errorf("track '%s' has no position data at frame %d", trackName, physicalFrame)
	}

	return *frame.X, *frame.Y, nil
}

// mapLogicalToPhysical 将逻辑帧号映射到物理帧索引
//
// 逻辑帧是可见帧的序号（0, 1, 2, ...）
// 物理帧是数组中的实际索引，包括隐藏帧
//
// 参数：
//   - logicalFrame: 逻辑帧号（从 0 开始）
//   - animVisibles: 动画可见性数组（0 = 可见，-1 = 隐藏）
//
// 返回：
//   - 物理帧索引，如果越界返回 -1
func (s *ReanimSystem) mapLogicalToPhysical(logicalFrame int, animVisibles []int) int {
	logicalIndex := 0
	for physicalIdx, visibility := range animVisibles {
		if visibility == 0 {
			if logicalIndex == logicalFrame {
				return physicalIdx
			}
			logicalIndex++
		}
	}
	return -1 // 逻辑帧越界
}

// ==================================================================
// Story 13.4: Render Cache Optimization (渲染缓存优化)
// ==================================================================

// getCurrentLogicalFrame 获取组件当前的逻辑帧索引（用于缓存失效检测）
//
// 返回主动画（CurrentAnimations[0]）的当前逻辑帧
// 如果没有动画播放，返回 0
//
// 参数：
//   - comp: ReanimComponent 引用
//
// 返回：
//   - 当前逻辑帧索引
func (s *ReanimSystem) getCurrentLogicalFrame(comp *components.ReanimComponent) int {
	// 如果没有播放任何动画，返回 0
	if len(comp.CurrentAnimations) == 0 {
		return 0
	}

	// 使用第一个动画作为主动画
	primaryAnim := comp.CurrentAnimations[0]

	// 获取该动画的状态
	animState, exists := comp.AnimStates[primaryAnim]
	if !exists {
		return 0
	}

	return animState.LogicalFrame
}

// getParentOffsetIfNeeded 如果需要，计算父子偏移
//
// 检查轨道是否有父轨道，并且子父使用不同的动画
// 如果满足条件，返回父轨道的偏移量；否则返回 (0, 0)
//
// 参数：
//   - trackName: 轨道名称
//   - comp: ReanimComponent 引用
//
// 返回：
//   - offsetX: 父轨道的 X 偏移
//   - offsetY: 父轨道的 Y 偏移
func (s *ReanimSystem) getParentOffsetIfNeeded(trackName string, comp *components.ReanimComponent) (float64, float64) {
	// 检查是否有父轨道
	if comp.ParentTracks == nil {
		return 0, 0
	}

	parentName, hasParent := comp.ParentTracks[trackName]
	if !hasParent {
		return 0, 0 // 无父轨道
	}

	// 检查轨道绑定
	if comp.TrackBindings == nil {
		return 0, 0
	}

	childAnim, childExists := comp.TrackBindings[trackName]
	parentAnim, parentExists := comp.TrackBindings[parentName]

	if !childExists || !parentExists {
		return 0, 0
	}

	// 如果子父使用相同的动画，不应用偏移
	if childAnim == parentAnim {
		return 0, 0
	}

	// 调用 Story 13.3 的函数计算父轨道偏移
	return s.getParentOffset(parentName, comp)
}

// prepareRenderCache 为指定组件构建渲染数据缓存（Story 13.4）
//
// 遍历所有可见轨道，计算并缓存渲染数据：
// - 图片引用（从 PartImages 获取）
// - 帧数据（变换信息）
// - 父子偏移（Story 13.3）
//
// 重用 CachedRenderData 切片，避免频繁分配内存
//
// 参数：
//   - comp: ReanimComponent 引用
func (s *ReanimSystem) prepareRenderCache(comp *components.ReanimComponent) {
	// 步骤 1: 清空现有缓存（重用切片，避免分配）
	comp.CachedRenderData = comp.CachedRenderData[:0]

	// 步骤 2: 获取可见轨道列表
	visualTracks := s.getVisualTracks(comp)

	// 步骤 3: 遍历每个轨道，构建缓存
	for _, trackName := range visualTracks {
		var animName string
		var logicalFrame int

		// 3.1: 找到控制该轨道的动画
		// 支持两种模式：TrackBindings 模式和传统模式
		if comp.TrackBindings != nil && len(comp.TrackBindings) > 0 {
			// TrackBindings 模式（Story 13.1）
			var exists bool
			animName, exists = comp.TrackBindings[trackName]
			if !exists {
				continue // 跳过未绑定的轨道
			}
		} else {
			// 传统模式：所有轨道使用相同的主动画
			animName = comp.CurrentAnim
			if animName == "" {
				continue
			}
		}

		// 3.2: 获取动画的当前逻辑帧
		animState, stateExists := comp.AnimStates[animName]
		if !stateExists {
			continue
		}
		logicalFrame = animState.LogicalFrame

		// 3.3: 映射到物理帧
		animVisibles, visiblesExist := comp.AnimVisiblesMap[animName]
		if !visiblesExist {
			continue
		}
		physicalFrame := s.mapLogicalToPhysical(logicalFrame, animVisibles)
		if physicalFrame == -1 {
			continue // 逻辑帧越界
		}

		// 3.4: 获取帧数据
		mergedFrames, tracksExist := comp.MergedTracks[trackName]
		if !tracksExist || physicalFrame >= len(mergedFrames) {
			continue // 越界保护
		}
		frame := mergedFrames[physicalFrame]

		// 3.5: 获取图片引用
		if frame.ImagePath == "" {
			continue // 跳过无图片路径的帧
		}
		img, imgExists := comp.PartImages[frame.ImagePath]
		if !imgExists || img == nil {
			continue // 跳过无图片的帧
		}

		// 3.6: 计算父子偏移（Story 13.3）
		offsetX, offsetY := s.getParentOffsetIfNeeded(trackName, comp)

		// 3.7: 加入缓存
		comp.CachedRenderData = append(comp.CachedRenderData, components.RenderPartData{
			Img:     img,
			Frame:   frame,
			OffsetX: offsetX,
			OffsetY: offsetY,
		})
	}
}

// ==================================================================
// Story 13.5: Configuration-Based Animation Setup (配置驱动的动画设置)
// ==================================================================

// ApplyReanimConfig 将 Reanim 配置应用到指定实体
//
// 此方法根据 YAML 配置文件设置实体的动画组合、轨道绑定、父子关系等。
// 配置文件格式详见：docs/reanim/reanim-config-guide.md
//
// 参数：
//   - entityID: 实体 ID
//   - config: Reanim 配置对象（从 YAML 文件加载）
//
// 返回：
//   - error: 应用失败时的错误
//
// 使用示例：
//
//	config, err := config.LoadReanimConfig("data/reanim_configs/peashooter.yaml")
//	if err != nil {
//	    return err
//	}
//	if err := reanimSystem.ApplyReanimConfig(entityID, config); err != nil {
//	    return err
//	}
func (s *ReanimSystem) ApplyReanimConfig(entityID ecs.EntityID, cfg *config.ReanimConfig) error {
	// 验证实体是否有 ReanimComponent
	comp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !exists {
		return fmt.Errorf("实体 %d 没有 ReanimComponent", entityID)
	}

	// 验证配置
	if cfg == nil {
		return fmt.Errorf("配置对象为空")
	}

	// 应用每个动画组合配置
	for i, combo := range cfg.AnimationCombos {
		// 验证组合配置
		if len(combo.Animations) == 0 {
			return fmt.Errorf("动画组合 #%d '%s' 的动画列表为空", i, combo.Name)
		}

		// 1. 播放动画
		if err := s.PlayAnimations(entityID, combo.Animations); err != nil {
			return fmt.Errorf("播放动画组合 '%s' 失败: %w", combo.Name, err)
		}

		// 2. 设置轨道绑定
		if err := s.applyTrackBindings(entityID, comp, &combo); err != nil {
			return fmt.Errorf("设置轨道绑定失败（组合 '%s'）: %w", combo.Name, err)
		}

		// 3. 设置父子关系
		if len(combo.ParentTracks) > 0 {
			if err := s.SetParentTracks(entityID, combo.ParentTracks); err != nil {
				return fmt.Errorf("设置父子关系失败（组合 '%s'）: %w", combo.Name, err)
			}
		}

		// 4. 设置隐藏轨道
		if len(combo.HiddenTracks) > 0 {
			if err := s.applyHiddenTracks(entityID, comp, combo.HiddenTracks); err != nil {
				return fmt.Errorf("设置隐藏轨道失败（组合 '%s'）: %w", combo.Name, err)
			}
		}

		// 注意：目前只应用第一个组合配置
		// 未来可以扩展为支持多个组合的切换
		break
	}

	return nil
}

// applyTrackBindings 根据配置应用轨道绑定
func (s *ReanimSystem) applyTrackBindings(
	entityID ecs.EntityID,
	comp *components.ReanimComponent,
	combo *config.AnimationComboConfig,
) error {
	// 如果没有指定策略，默认使用 auto
	strategy := combo.BindingStrategy
	if strategy == "" {
		strategy = "auto"
	}

	switch strategy {
	case "auto":
		// 自动绑定已在 PlayAnimations() 中处理
		// 不需要额外操作
		return nil

	case "manual":
		// 手动绑定：使用配置中的绑定关系
		if len(combo.ManualBindings) == 0 {
			return fmt.Errorf("manual 绑定策略但未提供 manual_bindings")
		}
		if err := s.SetTrackBindings(entityID, combo.ManualBindings); err != nil {
			return fmt.Errorf("设置手动绑定失败: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("无效的绑定策略 '%s'，只能是 'auto' 或 'manual'", strategy)
	}
}

// applyHiddenTracks 应用隐藏轨道配置
func (s *ReanimSystem) applyHiddenTracks(
	entityID ecs.EntityID,
	comp *components.ReanimComponent,
	hiddenTracks []string,
) error {
	// 初始化 VisibleTracks（如果需要）
	if comp.VisibleTracks == nil {
		comp.VisibleTracks = make(map[string]bool)
	}

	// 将指定的轨道标记为隐藏
	for _, trackName := range hiddenTracks {
		comp.VisibleTracks[trackName] = false
	}

	return nil
}

// ========================================
// Story 13.6: 配置驱动的动画播放 API
// ========================================

// SetConfigManager 设置配置管理器
//
// 此方法由游戏初始化逻辑调用，设置全局配置管理器。
// 调用此方法后，PlayCombo 和 PlayDefaultAnimation 才能正常工作。
//
// 参数：
//   - manager: 配置管理器实例
func (s *ReanimSystem) SetConfigManager(manager *config.ReanimConfigManager) {
	s.configManager = manager
	log.Printf("[ReanimSystem] 配置管理器已设置")
}

// PlayCombo 播放配置文件中定义的动画组合
//
// 此方法是配置驱动动画播放的核心 API，替代了旧的 PlayAnimation/PlayAnimations。
// 它自动处理动画组合、轨道绑定、父子关系等复杂逻辑。
//
// 参数：
//   - entityID: 实体 ID
//   - unitID: 动画单元 ID（如 "peashooter", "zombie"）
//   - comboName: 组合名称（如 "attack", "idle", "walk"）
//
// 返回：
//   - error: 配置不存在或应用失败时返回错误
//
// 示例：
//
//	// 播放豌豆射手攻击动画（anim_shooting + anim_head_idle）
//	rs.PlayCombo(peashooterID, "peashooter", "attack")
//
//	// 播放僵尸行走动画
//	rs.PlayCombo(zombieID, "zombie", "walk")
//
// 内部逻辑：
//  1. 从配置管理器获取组合配置
//  2. 调用 PlayAnimations(combo.Animations)
//  3. 应用轨道绑定（如果 binding_strategy = auto，已在 PlayAnimations 中处理）
//  4. 应用父子关系（SetParentTracks）
//  5. 应用隐藏轨道（通过 VisibleTracks）
func (s *ReanimSystem) PlayCombo(entityID ecs.EntityID, unitID, comboName string) error {
	// 0. 验证配置管理器已设置
	if s.configManager == nil {
		return fmt.Errorf("配置管理器未设置，请先调用 SetConfigManager()")
	}

	// 1. 获取组合配置
	combo, err := s.configManager.GetCombo(unitID, comboName)
	if err != nil {
		return fmt.Errorf("获取动画组合失败: %w", err)
	}

	// 2. 播放动画（自动处理轨道绑定）
	if err := s.PlayAnimations(entityID, combo.Animations); err != nil {
		return fmt.Errorf("播放动画失败: %w", err)
	}

	// 3. 应用父子关系（如果配置了）
	if len(combo.ParentTracks) > 0 {
		if err := s.SetParentTracks(entityID, combo.ParentTracks); err != nil {
			return fmt.Errorf("应用父子关系失败: %w", err)
		}
	}

	// 4. 应用隐藏轨道（如果配置了）
	if len(combo.HiddenTracks) > 0 {
		reanimComp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
		if exists {
			// 初始化 VisibleTracks（如果尚未初始化）
			if reanimComp.VisibleTracks == nil {
				reanimComp.VisibleTracks = make(map[string]bool)
				// 默认所有轨道可见
				for trackName := range reanimComp.MergedTracks {
					reanimComp.VisibleTracks[trackName] = true
				}
			}

			// 隐藏指定的轨道
			for _, trackName := range combo.HiddenTracks {
				reanimComp.VisibleTracks[trackName] = false
			}
		}
	}

	log.Printf("[ReanimSystem] PlayCombo: entity %d playing %s/%s (%v)",
		entityID, unitID, comboName, combo.Animations)

	return nil
}

// PlayDefaultAnimation 播放配置文件中定义的默认动画
//
// 此方法是 PlayCombo 的便捷版本，自动播放默认动画。
// 通常用于实体初始化时播放待机动画。
//
// 参数：
//   - entityID: 实体 ID
//   - unitID: 动画单元 ID（如 "peashooter", "zombie"）
//
// 返回：
//   - error: 配置不存在或应用失败时返回错误
//
// 示例：
//
//	// 播放豌豆射手默认动画（通常是 anim_idle）
//	rs.PlayDefaultAnimation(peashooterID, "peashooter")
//
//	// 播放僵尸默认动画
//	rs.PlayDefaultAnimation(zombieID, "zombie")
//
// 等效于：
//
//	animName, _ := configManager.GetDefaultAnimation(unitID)
//	rs.PlayAnimation(entityID, animName)
func (s *ReanimSystem) PlayDefaultAnimation(entityID ecs.EntityID, unitID string) error {
	// 0. 验证配置管理器已设置
	if s.configManager == nil {
		return fmt.Errorf("配置管理器未设置，请先调用 SetConfigManager()")
	}

	// 1. 获取默认动画名称
	animName, err := s.configManager.GetDefaultAnimation(unitID)
	if err != nil {
		return fmt.Errorf("获取默认动画失败: %w", err)
	}

	// 2. 播放动画
	if err := s.PlayAnimation(entityID, animName); err != nil {
		return fmt.Errorf("播放默认动画失败: %w", err)
	}

	log.Printf("[ReanimSystem] PlayDefaultAnimation: entity %d playing %s (default: %s)",
		entityID, unitID, animName)

	return nil
}
