package scenes

import (
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// TestButtonHighlightLogic tests the button highlight state switching logic.
//
// Story 12.1 Task 5.4: Unit tests for button highlight effect
func TestButtonHighlightLogic(t *testing.T) {
	// Create a minimal scene for testing
	em := ecs.NewEntityManager()
	scene := &MainMenuScene{
		entityManager:         em,
		buttonHitboxes:        config.MenuButtonHitboxes,
		hoveredButton:         "",
		currentLevel:          "1-1", // Default level (only Adventure unlocked)
		buttonNormalImages:    make(map[string]*ebiten.Image),
		buttonHighlightImages: make(map[string]*ebiten.Image),
		lastHoveredButton:     "",
	}

	// Create a mock SelectorScreen entity with ReanimComponent
	selectorEntity := em.CreateEntity()
	scene.selectorScreenEntity = selectorEntity

	// Create mock images (1x1 pixel images for testing)
	normalImg := ebiten.NewImage(1, 1)
	highlightImg := ebiten.NewImage(1, 1)
	// Note: We don't need to fill the images with different colors for testing,
	// since we're only testing image reference switching, not visual appearance

	// Setup button images
	scene.buttonNormalImages["SelectorScreen_Adventure_button"] = normalImg
	scene.buttonHighlightImages["SelectorScreen_Adventure_button"] = highlightImg

	// Create ReanimComponent with PartImages
	reanimComp := &components.ReanimComponent{
		PartImages: map[string]*ebiten.Image{
			"IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_BUTTON": normalImg,
		},
	}
	em.AddComponent(selectorEntity, reanimComp)

	// Test Case 1: No hover → image should be normal
	scene.hoveredButton = ""
	scene.updateButtonHighlight()

	currentImg := reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_BUTTON"]
	if currentImg != normalImg {
		t.Errorf("Test Case 1 Failed: Expected normal image when not hovering, got highlight")
	}

	// Test Case 2: Hover over unlocked button → image should be highlight
	scene.hoveredButton = "SelectorScreen_Adventure_button"
	scene.updateButtonHighlight()

	currentImg = reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_BUTTON"]
	if currentImg != highlightImg {
		t.Errorf("Test Case 2 Failed: Expected highlight image when hovering, got normal")
	}

	// Test Case 3: Stop hovering → image should restore to normal
	scene.hoveredButton = ""
	scene.updateButtonHighlight()

	currentImg = reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_BUTTON"]
	if currentImg != normalImg {
		t.Errorf("Test Case 3 Failed: Expected normal image after stop hovering, got highlight")
	}
}

// TestButtonHighlightLockedButton tests that locked buttons don't get highlighted.
//
// Story 12.1 Task 5.4: Unit tests for button highlight effect
func TestButtonHighlightLockedButton(t *testing.T) {
	// Create a minimal scene for testing
	em := ecs.NewEntityManager()
	scene := &MainMenuScene{
		entityManager:         em,
		buttonHitboxes:        config.MenuButtonHitboxes,
		hoveredButton:         "",
		currentLevel:          "1-1", // Only Adventure unlocked
		buttonNormalImages:    make(map[string]*ebiten.Image),
		buttonHighlightImages: make(map[string]*ebiten.Image),
		lastHoveredButton:     "",
	}

	// Create a mock SelectorScreen entity with ReanimComponent
	selectorEntity := em.CreateEntity()
	scene.selectorScreenEntity = selectorEntity

	// Create mock images
	normalImg := ebiten.NewImage(1, 1)
	highlightImg := ebiten.NewImage(1, 1)

	// Setup button images for Survival button (locked at level 1-1)
	scene.buttonNormalImages["SelectorScreen_Survival_button"] = normalImg
	scene.buttonHighlightImages["SelectorScreen_Survival_button"] = highlightImg

	// Create ReanimComponent with PartImages
	reanimComp := &components.ReanimComponent{
		PartImages: map[string]*ebiten.Image{
			"IMAGE_REANIM_SELECTORSCREEN_SURVIVAL_BUTTON": normalImg,
		},
	}
	em.AddComponent(selectorEntity, reanimComp)

	// Test: Hover over locked button → image should remain normal
	scene.hoveredButton = "SelectorScreen_Survival_button"
	scene.updateButtonHighlight()

	currentImg := reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_SURVIVAL_BUTTON"]
	if currentImg != normalImg {
		t.Errorf("TestLockedButton Failed: Locked button should not be highlighted")
	}
}

// TestLastHoveredButtonTracking tests that sound effect is only played once per hover.
//
// Story 12.1 Task 5.4: Unit tests for button highlight effect
func TestLastHoveredButtonTracking(t *testing.T) {
	// Create a minimal scene for testing
	em := ecs.NewEntityManager()
	scene := &MainMenuScene{
		entityManager:         em,
		buttonHitboxes:        config.MenuButtonHitboxes,
		hoveredButton:         "",
		currentLevel:          "1-1",
		buttonNormalImages:    make(map[string]*ebiten.Image),
		buttonHighlightImages: make(map[string]*ebiten.Image),
		lastHoveredButton:     "",
	}

	// Create a mock SelectorScreen entity with ReanimComponent
	selectorEntity := em.CreateEntity()
	scene.selectorScreenEntity = selectorEntity

	// Create mock images
	normalImg := ebiten.NewImage(1, 1)
	highlightImg := ebiten.NewImage(1, 1)

	// Setup button images
	scene.buttonNormalImages["SelectorScreen_Adventure_button"] = normalImg
	scene.buttonHighlightImages["SelectorScreen_Adventure_button"] = highlightImg

	// Create ReanimComponent with PartImages
	reanimComp := &components.ReanimComponent{
		PartImages: map[string]*ebiten.Image{
			"IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_BUTTON": normalImg,
		},
	}
	em.AddComponent(selectorEntity, reanimComp)

	// Test: First hover → lastHoveredButton should be set
	scene.hoveredButton = "SelectorScreen_Adventure_button"
	scene.updateButtonHighlight()

	if scene.lastHoveredButton != "SelectorScreen_Adventure_button" {
		t.Errorf("Test Failed: lastHoveredButton should be set on first hover")
	}

	// Test: Continue hovering same button → lastHoveredButton should remain same
	// (This ensures sound effect is not played repeatedly)
	scene.updateButtonHighlight()

	if scene.lastHoveredButton != "SelectorScreen_Adventure_button" {
		t.Errorf("Test Failed: lastHoveredButton should remain same on continued hover")
	}

	// Test: Stop hovering → lastHoveredButton should be cleared
	scene.hoveredButton = ""
	scene.updateButtonHighlight()

	if scene.lastHoveredButton != "" {
		t.Errorf("Test Failed: lastHoveredButton should be cleared when not hovering")
	}

	// Test: Hover different button → lastHoveredButton should update
	scene.buttonNormalImages["SelectorScreen_Survival_button"] = normalImg
	scene.buttonHighlightImages["SelectorScreen_Survival_button"] = highlightImg
	reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_SURVIVAL_BUTTON"] = normalImg

	// Unlock Survival for this test (set high level)
	scene.currentLevel = "5-10"

	scene.hoveredButton = "SelectorScreen_Survival_button"
	scene.updateButtonHighlight()

	if scene.lastHoveredButton != "SelectorScreen_Survival_button" {
		t.Errorf("Test Failed: lastHoveredButton should update when hovering different button")
	}
}

// TestButtonHighlightWithMultipleButtons tests highlight behavior with multiple buttons.
//
// Story 12.1 Task 5.4: Unit tests for button highlight effect
func TestButtonHighlightWithMultipleButtons(t *testing.T) {
	// Create a minimal scene for testing
	em := ecs.NewEntityManager()
	scene := &MainMenuScene{
		entityManager:         em,
		buttonHitboxes:        config.MenuButtonHitboxes,
		hoveredButton:         "",
		currentLevel:          "5-10", // All buttons unlocked
		buttonNormalImages:    make(map[string]*ebiten.Image),
		buttonHighlightImages: make(map[string]*ebiten.Image),
		lastHoveredButton:     "",
	}

	// Create a mock SelectorScreen entity with ReanimComponent
	selectorEntity := em.CreateEntity()
	scene.selectorScreenEntity = selectorEntity

	// Create mock images for all buttons
	adventureNormal := ebiten.NewImage(1, 1)
	adventureHighlight := ebiten.NewImage(1, 1)
	survivalNormal := ebiten.NewImage(1, 1)
	survivalHighlight := ebiten.NewImage(1, 1)

	// Setup button images
	scene.buttonNormalImages["SelectorScreen_Adventure_button"] = adventureNormal
	scene.buttonHighlightImages["SelectorScreen_Adventure_button"] = adventureHighlight
	scene.buttonNormalImages["SelectorScreen_Survival_button"] = survivalNormal
	scene.buttonHighlightImages["SelectorScreen_Survival_button"] = survivalHighlight

	// Create ReanimComponent with PartImages
	reanimComp := &components.ReanimComponent{
		PartImages: map[string]*ebiten.Image{
			"IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_BUTTON": adventureNormal,
			"IMAGE_REANIM_SELECTORSCREEN_SURVIVAL_BUTTON":  survivalNormal,
		},
	}
	em.AddComponent(selectorEntity, reanimComp)

	// Test: Hover Adventure button → only Adventure should be highlighted
	scene.hoveredButton = "SelectorScreen_Adventure_button"
	scene.updateButtonHighlight()

	if reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_BUTTON"] != adventureHighlight {
		t.Errorf("Test Failed: Adventure button should be highlighted")
	}
	if reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_SURVIVAL_BUTTON"] != survivalNormal {
		t.Errorf("Test Failed: Survival button should remain normal")
	}

	// Test: Switch to Survival button → Adventure should restore, Survival should highlight
	scene.hoveredButton = "SelectorScreen_Survival_button"
	scene.updateButtonHighlight()

	if reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_BUTTON"] != adventureNormal {
		t.Errorf("Test Failed: Adventure button should restore to normal")
	}
	if reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_SURVIVAL_BUTTON"] != survivalHighlight {
		t.Errorf("Test Failed: Survival button should be highlighted")
	}

	// Test: Stop hovering → all buttons should restore to normal
	scene.hoveredButton = ""
	scene.updateButtonHighlight()

	if reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_BUTTON"] != adventureNormal {
		t.Errorf("Test Failed: Adventure button should restore to normal after stop hovering")
	}
	if reanimComp.PartImages["IMAGE_REANIM_SELECTORSCREEN_SURVIVAL_BUTTON"] != survivalNormal {
		t.Errorf("Test Failed: Survival button should restore to normal after stop hovering")
	}
}
