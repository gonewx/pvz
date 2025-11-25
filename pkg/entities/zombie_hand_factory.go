package entities

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// NewZombieHandEntity creates a Zombie Hand Reanim entity for the transition animation.
// This entity displays a zombie hand rising from the ground.
//
// The Zombie_hand.reanim file contains a single animation sequence (no named animations):
//   - arm轨道: 僵尸手臂从地下升起
//   - hand轨道: 僵尸手掌跟随手臂移动
//   - finger*轨道: 手指动画
//   - rock*轨道: 石头碎片效果
//
// Parameters:
//   - em: Entity manager for creating the entity
//   - rm: Resource manager for loading Reanim data and images
//   - x, y: Position of the hand entity
//
// Returns:
//   - ecs.EntityID: The created entity ID
//   - error: Error if resource loading fails
//
// Story 12.6 Task 2.2: Create zombie hand entity in main menu
func NewZombieHandEntity(em *ecs.EntityManager, rm *game.ResourceManager, x, y float64) (ecs.EntityID, error) {
	// 1. Get Reanim data from cache (already loaded by LoadReanimResources)
	reanimXML := rm.GetReanimXML("Zombie_hand")
	if reanimXML == nil {
		return 0, fmt.Errorf("Zombie_hand.reanim not found in cache")
	}

	// 2. Get part images from cache
	partImages := rm.GetReanimPartImages("Zombie_hand")
	if partImages == nil {
		return 0, fmt.Errorf("Zombie_hand part images not found in cache")
	}

	// 3. Create entity
	entityID := em.CreateEntity()

	// 4. Add PositionComponent
	ecs.AddComponent(em, entityID, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// 5. Extract visual tracks (all tracks are visual, no logical tracks)
	visualTracks := extractVisualTracks(reanimXML, nil)
	log.Printf("[ZombieHand] Created entity with %d visual tracks", len(visualTracks))

	// 6. Add ReanimComponent
	// Note: Zombie_hand.reanim is a single animation file (no named animations)
	// We create a synthetic animation called "_root" to represent the entire file
	mergedTracks := reanim.BuildMergedTracks(reanimXML)

	// Build visibles array for single-animation file
	// For Zombie_hand.reanim, this is a single-animation file without named animations.
	// Each track has its own visibility pattern (some start at frame 4, others at frame 8).
	//
	// We use a simple sequential visibles array [0, 1, 2, ..., maxFrames-1]
	// to indicate that the animation plays all frames sequentially.
	// The per-track visibility (f=-1 hidden frames) is handled by prepareRenderCache
	// checking each track's individual FrameNum values.
	maxFrames := 0
	for _, frames := range mergedTracks {
		if len(frames) > maxFrames {
			maxFrames = len(frames)
		}
	}

	// 对于单动画文件，所有帧都是可见的
	// mapLogicalToPhysical 期望：0 = 可见，非0 = 隐藏
	// 所以数组应该全是 0，表示所有帧都可见
	visiblesArray := make([]int, maxFrames)
	for i := 0; i < maxFrames; i++ {
		visiblesArray[i] = 0 // All frames are visible
	}

	log.Printf("[ZombieHand] Built visibles array: len=%d (all frames visible with value 0)",
		len(visiblesArray))

	reanimComp := &components.ReanimComponent{
		// 基础数据
		ReanimName:   "Zombie_hand",
		ReanimXML:    reanimXML,
		PartImages:   partImages,
		MergedTracks: mergedTracks,

		// 轨道分类
		VisualTracks:  visualTracks,
		LogicalTracks: []string{},

		// 播放状态
		CurrentFrame:      0,
		FrameAccumulator:  0.0,
		AnimationFPS:      float64(reanimXML.FPS),
		CurrentAnimations: []string{"_root"}, // Synthetic animation name for single-animation files

		// 动画数据 (pre-built for single-animation files)
		AnimVisiblesMap: map[string][]int{
			"_root": visiblesArray,
		},

		// 配置字段
		ParentTracks: nil,
		HiddenTracks: nil,

		// 渲染缓存
		CachedRenderData: []components.RenderPartData{},
		LastRenderFrame:  -1,

		// 控制标志
		IsPaused:   true,  // Start paused (will be played on button click)
		IsLooping:  false, // Single playthrough (transition animation)
		IsFinished: false,

		// 中心偏移 (for coordinate transformation)
		CenterOffsetX: 0,
		CenterOffsetY: 0,
	}

	ecs.AddComponent(em, entityID, reanimComp)

	log.Printf("[ZombieHand] Entity created: ID=%d, position=(%.1f, %.1f), FPS=%d, totalFrames=%d",
		entityID, x, y, reanimXML.FPS, len(visiblesArray))

	return entityID, nil
}
