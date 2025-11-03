package entities

import (
	"fmt"
	"log"

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
	em.AddComponent(entityID, reanimComp)

	// 5. Add PositionComponent
	em.AddComponent(entityID, &components.PositionComponent{
		X: x,
		Y: y,
	})

	log.Printf("[SelectorScreen] Created partial entity %d (tracks=%d, anim=%s)", entityID, len(visibleTracks), animName)
	return entityID, nil
}
