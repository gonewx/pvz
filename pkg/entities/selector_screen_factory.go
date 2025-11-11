package entities

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
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
	// 创建实体（不指定可见轨道和动画，让系统自动处理）
	entity, err := NewSelectorScreenPartialEntity(em, rm, nil, "", 0, 0)
	if err != nil {
		return 0, err
	}

	// Story 13.8: 添加 Reanim 组件后，需要调用 ReanimSystem 初始化动画
	// 但是在这里我们无法访问 ReanimSystem，所以在 MainMenuScene 中初始化
	// 注意：这与植物工厂不同，植物工厂会接收 ReanimSystem 作为参数

	return entity, nil
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

	// 4. Add ReanimComponent (Story 13.8: 简化为新的结构)
	visualTracks := extractVisualTracks(reanimXML, visibleTracks)
	log.Printf("[SelectorScreen] extractVisualTracks 返回: %d 个轨道", len(visualTracks))
	if len(visualTracks) > 0 {
		log.Printf("[SelectorScreen] 前5个轨道: %v", visualTracks[:min(5, len(visualTracks))])
	}

	reanimComp := &components.ReanimComponent{
		// 基础数据
		ReanimName:   "SelectorScreen", // For config lookup and debugging
		ReanimXML:    reanimXML,
		PartImages:   partImages,
		MergedTracks: reanim.BuildMergedTracks(reanimXML),

		// 轨道分类
		VisualTracks:  visualTracks,
		LogicalTracks: []string{},

		// 播放状态
		CurrentFrame:      0,
		FrameAccumulator:  0,
		AnimationFPS:      float64(reanimXML.FPS), // ✅ 从 .reanim 文件读取 FPS，而不是硬编码
		CurrentAnimations: []string{animName},

		// 动画数据
		AnimVisiblesMap: reanim.BuildAnimVisiblesMap(reanimXML),
		// ✅ Story 13.10: TrackAnimationBinding 已删除

		// 配置字段
		ParentTracks: nil,
		HiddenTracks: buildHiddenTracks(reanimXML, visibleTracks),

		// 渲染缓存
		CachedRenderData: []components.RenderPartData{},
		LastRenderFrame:  -1,

		// 控制标志
		IsPaused:   false,
		IsLooping:  false,
		IsFinished: false,
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

// extractVisualTracks 从 ReanimXML 中提取视觉轨道列表
// 如果提供了 visibleTracks，则只包含指定的轨道
func extractVisualTracks(reanimXML *reanim.ReanimXML, visibleTracks map[string]bool) []string {
	if reanimXML == nil {
		return []string{}
	}

	tracks := []string{}
	for _, track := range reanimXML.Tracks {
		// 如果指定了 visibleTracks，则只包含在列表中的轨道
		if visibleTracks != nil {
			if visibleTracks[track.Name] {
				tracks = append(tracks, track.Name)
			}
		} else {
			// 否则包含所有轨道
			tracks = append(tracks, track.Name)
		}
	}
	return tracks
}

// buildHiddenTracks 根据 visibleTracks 构建隐藏轨道映射
// 如果提供了 visibleTracks，则未包含的轨道会被隐藏
func buildHiddenTracks(reanimXML *reanim.ReanimXML, visibleTracks map[string]bool) map[string]bool {
	if visibleTracks == nil || reanimXML == nil {
		return nil // 不隐藏任何轨道
	}

	hiddenTracks := make(map[string]bool)
	for _, track := range reanimXML.Tracks {
		// 如果轨道不在 visibleTracks 中，则隐藏它
		if !visibleTracks[track.Name] {
			hiddenTracks[track.Name] = true
		}
	}
	return hiddenTracks
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
