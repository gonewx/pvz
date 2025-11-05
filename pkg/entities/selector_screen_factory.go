package entities

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// NewSelectorScreenEntity creates a SelectorScreen Reanim entity for the main menu.
// This entity displays the tombstone menu buttons, background decorations, clouds, and flowers.
//
// The SelectorScreen.reanim file contains:
//   - All 4 main menu buttons (Adventure, Challenges, Survival, ZenGarden)
//   - Background images (BG, BG_Center, BG_Left, BG_Right)
//   - Animated decorations (clouds, flowers, leaves)
//   - Button shadows and other visual elements
//
// Parameters:
//   - em: Entity manager for creating the entity
//   - rm: Resource manager for loading Reanim data and images
//
// Returns:
//   - ecs.EntityID: The created entity ID
//   - error: Error if resource loading fails
//
// Story 12.1: Main Menu Tombstone System Enhancement
func NewSelectorScreenEntity(em *ecs.EntityManager, rm *game.ResourceManager) (ecs.EntityID, error) {
	return NewSelectorScreenPartialEntity(em, rm, nil, "", 0, 0)
}

// NewSelectorScreenPartialEntity creates a SelectorScreen Reanim entity with specific visible tracks.
//
// This function allows creating multiple entities from the same SelectorScreen.reanim file,
// where each entity only displays a subset of tracks and plays a specific animation.
//
// Parameters:
//   - em: Entity manager for creating the entity
//   - rm: Resource manager for loading Reanim data and images
//   - visibleTracks: Map of track names to show (nil = show all tracks)
//   - animName: Animation to play (empty = no animation)
//   - x, y: Position offsets
//
// Returns:
//   - ecs.EntityID: The created entity ID
//   - error: Error if resource loading fails
func NewSelectorScreenPartialEntity(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	visibleTracks map[string]bool,
	animName string,
	x, y float64,
) (ecs.EntityID, error) {
	// 1. Get Reanim data from cache (already loaded by LoadReanimResources)
	reanimXML := rm.GetReanimXML("SelectorScreen")
	if reanimXML == nil {
		return 0, fmt.Errorf("SelectorScreen.reanim not found in cache")
	}

	// 2. Get part images from cache
	partImages := rm.GetReanimPartImages("SelectorScreen")
	if partImages == nil {
		return 0, fmt.Errorf("SelectorScreen part images not found in cache")
	}

	// 3. Create entity
	entityID := em.CreateEntity()

	// 4. Add ReanimComponent
	reanimComp := &components.ReanimComponent{
		ReanimName:        "SelectorScreen", // For config lookup and debugging
		Reanim:            reanimXML,
		PartImages:        partImages,
		CurrentAnim:       "",
		IsLooping:         true,
		IsPaused:          false,
		VisibleTracks:     visibleTracks,
		CurrentFrame:      0,
		FrameAccumulator:  0,
		FixedCenterOffset: true,
		CenterOffsetX:     0,
		CenterOffsetY:     0,
	}

	// 5. Initialize independent animations (for ComplexScene mode)
	// 如果没有 visibleTracks（完整 SelectorScreen），则初始化独立动画
	if visibleTracks == nil {
		if err := initializeIndependentAnimations(reanimComp, rm); err != nil {
			log.Printf("[SelectorScreen] Warning: Failed to initialize independent animations: %v", err)
		}
	}

	em.AddComponent(entityID, reanimComp)

	// 6. Add PositionComponent
	em.AddComponent(entityID, &components.PositionComponent{
		X: x,
		Y: y,
	})

	log.Printf("[SelectorScreen] Created partial entity %d (tracks=%d, anim=%s)", entityID, len(visibleTracks), animName)
	return entityID, nil
}

// initializeIndependentAnimations 初始化 SelectorScreen 的独立动画系统
//
// 从配置文件读取独立动画列表，并为每个动画创建状态对象。
//
// 参数：
//   - reanimComp: ReanimComponent 组件
//   - rm: Resource manager（用于访问 AnimVisiblesMap）
//
// 返回：error 如果初始化失败
func initializeIndependentAnimations(reanimComp *components.ReanimComponent, rm *game.ResourceManager) error {
	// 1. 从配置读取独立动画列表
	animConfig, found := config.GetAnimationConfig("SelectorScreen")
	if !found {
		return fmt.Errorf("SelectorScreen not found in config")
	}

	if len(animConfig.IndependentAnimations) == 0 {
		log.Printf("[SelectorScreen] No independent animations defined in config")
		return nil
	}

	// 2. 初始化 Anims map
	reanimComp.Anims = make(map[string]*components.AnimState)

	// 3. 构建 AnimVisiblesMap（需要获取时间窗口信息）
	// 使用 reanim 包的函数构建时间窗口
	animVisiblesMap := reanim.BuildAnimVisiblesMap(reanimComp.Reanim)
	reanimComp.AnimVisiblesMap = animVisiblesMap

	// 4. 构建 MergedTracks（帧继承）
	// 这是渲染所必需的，必须在初始化时构建
	reanimComp.MergedTracks = reanim.BuildMergedTracks(reanimComp.Reanim)
	log.Printf("[SelectorScreen] Built MergedTracks: %d tracks", len(reanimComp.MergedTracks))

	// 5. 为每个独立动画创建状态 + 构建轨道映射
	reanimComp.TrackMapping = make(map[string]string)

	for _, animName := range animConfig.IndependentAnimations {
		// 计算动画的起始帧和帧数
		startFrame := 0
		frameCount := 0
		if animVisibles, exists := animVisiblesMap[animName]; exists {
			// 找到第一个可见帧（visibles[i] != -1）
			firstVisible := -1
			for i := 0; i < len(animVisibles); i++ {
				if animVisibles[i] != -1 {
					firstVisible = i
					break
				}
			}

			// 找到最后一个可见帧（visibles[i] != -1）
			lastVisible := -1
			for i := len(animVisibles) - 1; i >= 0; i-- {
				if animVisibles[i] != -1 {
					lastVisible = i
					break
				}
			}

			if firstVisible >= 0 && lastVisible >= 0 {
				startFrame = firstVisible
				frameCount = lastVisible - firstVisible + 1
			} else {
				// 没有可见帧，使用物理帧总数
				startFrame = 0
				frameCount = len(animVisibles)
			}
		}

		if frameCount == 0 {
			log.Printf("[SelectorScreen] Warning: Animation '%s' has no frames, skipping", animName)
			continue
		}

		// 创建默认状态
		state := &components.AnimState{
			Name:              animName,
			Frame:             startFrame, // 从可见帧开始
			Accumulator:       0,
			StartFrame:        startFrame,
			FrameCount:        frameCount,
			IsLooping:         true,  // 默认循环
			IsActive:          true,  // 默认激活
			RenderWhenStopped: true,  // 默认停止后仍渲染（向后兼容）
			DelayTimer:        0,
			DelayDuration:     0, // 默认无延迟
		}

		// 应用配置中的自定义参数 + 构建轨道映射
		if animConfig.IndependentAnimConfigs != nil {
			if customConfig, exists := animConfig.IndependentAnimConfigs[animName]; exists {
				// 应用延迟
				state.DelayDuration = customConfig.DelayDuration

				// 应用循环设置
				if customConfig.IsLooping != nil {
					state.IsLooping = *customConfig.IsLooping
				}

				// 应用激活设置
				if customConfig.IsActive != nil {
					state.IsActive = *customConfig.IsActive
				}

				// 应用 RenderWhenStopped 设置（新增）
				if customConfig.RenderWhenStopped != nil {
					state.RenderWhenStopped = *customConfig.RenderWhenStopped
				}

				// 应用 LockAtFrame 设置（新增）
				if customConfig.LockAtFrame != nil {
					state.Frame = *customConfig.LockAtFrame
					state.IsActive = false // 锁定在指定帧，停止推进
					log.Printf("[SelectorScreen] 🔒 Animation '%s' locked at frame %d",
						animName, *customConfig.LockAtFrame)
				}

				// 构建轨道映射（配置的特殊规则）
				if len(customConfig.ControlledTracks) > 0 {
					for _, trackName := range customConfig.ControlledTracks {
						reanimComp.TrackMapping[trackName] = animName
						log.Printf("[SelectorScreen] 🔗 Track mapping: '%s' → '%s' (from config)",
							trackName, animName)
					}
				}
			}
		}

		reanimComp.Anims[animName] = state
		log.Printf("[SelectorScreen] ✅ Initialized independent animation '%s' (frames=%d, loop=%v, active=%v, render_stopped=%v, delay=%.1fs)",
			animName, frameCount, state.IsLooping, state.IsActive, state.RenderWhenStopped, state.DelayDuration)
	}

	log.Printf("[SelectorScreen] ✅ Initialized %d independent animations", len(reanimComp.Anims))
	return nil
}
