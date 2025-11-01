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

	log.Printf("[SelectorScreen] Loaded Reanim: FPS=%d, Tracks=%d, Images=%d",
		reanimXML.FPS, len(reanimXML.Tracks), len(partImages))

	// 3. Create entity
	entityID := em.CreateEntity()

	// 4. Add ReanimComponent
	reanimComp := &components.ReanimComponent{
		Reanim:     reanimXML,
		PartImages: partImages,
		// Animation will be set by MainMenuScene (PlayAnimation "anim_idle")
		CurrentAnim:      "",
		IsLooping:        true,
		IsPaused:         false,
		VisibleTracks:    nil, // Will be set by MainMenuScene based on unlock status
		CurrentFrame:     0,
		FrameAccumulator: 0,
		// Scene animation: disable auto center offset calculation
		FixedCenterOffset: true,
		CenterOffsetX:     0,
		CenterOffsetY:     0,
	}
	em.AddComponent(entityID, reanimComp)

	// 5. Add PositionComponent (SelectorScreen renders at origin)
	// Note: Individual buttons will be positioned by Reanim tracks
	em.AddComponent(entityID, &components.PositionComponent{
		X: 0,
		Y: 0,
	})

	log.Printf("[SelectorScreen] Created entity %d", entityID)
	return entityID, nil
}
