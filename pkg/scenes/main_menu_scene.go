package scenes

import (
	"image/color"
	"log"
	"os"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

const (
	// WindowWidth is the logical width of the game window in pixels.
	WindowWidth = 800
	// WindowHeight is the logical height of the game window in pixels.
	WindowHeight = 600
)

// MainMenuScene represents the main menu screen of the game.
// It displays when the game starts and allows the player to navigate to other scenes.
type MainMenuScene struct {
	resourceManager *game.ResourceManager
	sceneManager    *game.SceneManager
	backgroundImage *ebiten.Image
	bgmPlayer       *audio.Player
	buttons         []components.Button
	wasMousePressed bool // Track mouse state from previous frame to detect click edges

	// Story 12.1: SelectorScreen Reanim entity and systems
	entityManager        *ecs.EntityManager
	reanimSystem         *systems.ReanimSystem
	renderSystem         *systems.RenderSystem
	selectorScreenEntity ecs.EntityID

	// Story 12.1: Button state management
	buttonHitboxes []config.MenuButtonHitbox
	hoveredButton  string // Current hovered button track name (empty = no hover)
	currentLevel   string // Current highest level from save (format: "X-Y")

	// Debug flag (only print once)
	debugPrinted bool
}

// NewMainMenuScene creates and returns a new MainMenuScene instance.
// It loads the main menu background image and initializes interactive buttons.
//
// Parameters:
//   - rm: The ResourceManager instance used to load game resources.
//   - sm: The SceneManager instance used to switch between scenes.
//
// Returns:
//   - A pointer to the newly created MainMenuScene.
//
// If the background image fails to load, the scene will fall back to a solid color background.
func NewMainMenuScene(rm *game.ResourceManager, sm *game.SceneManager) *MainMenuScene {
	scene := &MainMenuScene{
		resourceManager: rm,
		sceneManager:    sm,
	}

	// Story 12.1: Initialize ECS systems for SelectorScreen Reanim
	scene.entityManager = ecs.NewEntityManager()
	scene.reanimSystem = systems.NewReanimSystem(scene.entityManager)
	scene.renderSystem = systems.NewRenderSystem(scene.entityManager)
	log.Printf("[MainMenuScene] Initialized ECS systems")

	// Story 12.1: Create SelectorScreen Reanim entity
	selectorEntity, err := entities.NewSelectorScreenEntity(scene.entityManager, rm)
	if err != nil {
		log.Printf("Warning: Failed to create SelectorScreen entity: %v", err)
		log.Printf("Main menu will use fallback rendering")
		scene.selectorScreenEntity = 0
	} else {
		scene.selectorScreenEntity = selectorEntity

		// Story 6.6: 使用通用播放模式（自动检测为 ModeComplexScene）
		// SelectorScreen 是一个复杂的场景动画，包含 500+ 轨道：
		// - 背景/按钮：静止显示
		// - 草丛：循环摆动
		// - 云朵/墓碑：独立动画循环
		// 系统会自动检测为复杂场景模式，使用 ComplexScenePlaybackStrategy
		if err := scene.reanimSystem.PlayAnimation(selectorEntity, "anim_idle"); err != nil {
			log.Printf("Warning: Failed to play SelectorScreen animation: %v", err)
		} else {
			log.Printf("[MainMenuScene] 使用通用播放模式播放主菜单动画（自动检测：ComplexScene）")
		}
	}

	// Story 12.1: Initialize button hitboxes
	scene.buttonHitboxes = config.MenuButtonHitboxes

	// Story 12.1: Load current level from save
	gameState := game.GetGameState()
	saveManager := gameState.GetSaveManager()
	if err := saveManager.Load(); err == nil {
		scene.currentLevel = saveManager.GetHighestLevel()
		if scene.currentLevel == "" {
			scene.currentLevel = "1-1" // Default for new players
		}
		log.Printf("[MainMenuScene] Loaded highest level: %s", scene.currentLevel)
	} else {
		scene.currentLevel = "1-1" // Default for new players
		log.Printf("[MainMenuScene] No save file, defaulting to level 1-1")
	}

	// Story 12.1: Update button visibility based on unlock status
	scene.updateButtonVisibility()

	// Load background image (fallback if SelectorScreen fails)
	img, err := rm.LoadImageByID("IMAGE_REANIM_SELECTORSCREEN_BG")
	if err != nil {
		log.Printf("Warning: Failed to load main menu background: %v", err)
		log.Printf("The game will use a fallback solid color background")
		// Fallback: keep backgroundImage as nil, will use solid color in Draw()
	} else {
		scene.backgroundImage = img
	}

	// Load background music (using titlescreen music from loaderbar group)
	// Note: Need to ensure loaderbar group is loaded before this
	player, err := rm.LoadSoundEffect("assets/sounds/titlescreen.ogg")
	if err != nil {
		log.Printf("Warning: Failed to load main menu music: %v", err)
		// Continue without music
	} else {
		scene.bgmPlayer = player
	}

	// Initialize buttons
	scene.initButtons()

	return scene
}

// Update updates the main menu scene logic.
// deltaTime is the time elapsed since the last update in seconds.
func (m *MainMenuScene) Update(deltaTime float64) {
	// Ensure background music is playing
	if m.bgmPlayer != nil && !m.bgmPlayer.IsPlaying() {
		m.bgmPlayer.Play()
	}

	// Story 12.1: Update Reanim system (animate clouds, flowers, etc.)
	if m.reanimSystem != nil {
		m.reanimSystem.Update(deltaTime)
	}

	// Get mouse position
	mouseX, mouseY := ebiten.CursorPosition()

	// Check if mouse button is currently pressed
	isMousePressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	// Detect click edge (button was just pressed this frame)
	isMouseClicked := isMousePressed && !m.wasMousePressed

	// Story 12.1: Check SelectorScreen button hitboxes
	m.hoveredButton = "" // Reset hovered button
	for _, hitbox := range m.buttonHitboxes {
		// Check if mouse is in hitbox
		if isPointInRect(float64(mouseX), float64(mouseY), hitbox.X, hitbox.Y, hitbox.Width, hitbox.Height) {
			m.hoveredButton = hitbox.TrackName

			if isMouseClicked {
				// Button clicked
				m.onMenuButtonClicked(hitbox.ButtonType)
			}
			break // Only one button can be hovered at a time
		}
	}

	// Update old-style button states based on mouse position and clicks
	for i := range m.buttons {
		btn := &m.buttons[i]

		// Check if mouse is hovering over this button
		if isPointInRect(float64(mouseX), float64(mouseY), btn.X, btn.Y, btn.Width, btn.Height) {
			// Mouse is over the button
			if isMouseClicked {
				// Button was clicked
				btn.State = components.UIClicked
				if btn.OnClick != nil {
					btn.OnClick()
				}
			} else {
				// Button is hovered but not clicked
				btn.State = components.UIHovered
			}
		} else {
			// Mouse is not over the button
			btn.State = components.UINormal
		}
	}

	// Remember mouse state for next frame
	m.wasMousePressed = isMousePressed
}

// Draw renders the main menu scene to the screen.
// If a background image is loaded, it draws the image.
// Otherwise, it uses a dark blue fallback background.
func (m *MainMenuScene) Draw(screen *ebiten.Image) {
	// Story 12.1: Draw SelectorScreen Reanim (contains background, buttons, decorations)
	if m.renderSystem != nil && m.selectorScreenEntity != 0 {
		// 使用 RenderSystem.Draw() 渲染所有实体（包括 SelectorScreen）
		// 使用 cameraX = 0（主菜单没有摄像机偏移）
		m.renderSystem.Draw(screen, 0)

		// Note: Old m.buttons drawing removed - SelectorScreen Reanim handles all button rendering
	} else {
		// Fallback: Draw background image if SelectorScreen failed to load
		if m.backgroundImage != nil {
			// Scale background image to fit window size if needed
			bounds := m.backgroundImage.Bounds()
			bgWidth := float64(bounds.Dx())
			bgHeight := float64(bounds.Dy())

			// Calculate scale factors
			scaleX := WindowWidth / bgWidth
			scaleY := WindowHeight / bgHeight

			// Create draw options with scaling
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(scaleX, scaleY)

			// Draw the background image
			screen.DrawImage(m.backgroundImage, op)
		} else {
			// Fallback: Fill the screen with a dark blue color (midnight blue)
			screen.Fill(color.RGBA{R: 25, G: 25, B: 112, A: 255})
		}

		// Fallback: Draw old-style buttons only if Reanim failed to load
		for _, btn := range m.buttons {
			// Skip drawing if button has no image
			if btn.NormalImage == nil {
				continue
			}

			// Select which image to draw based on button state
			var img *ebiten.Image
			if btn.State == components.UIHovered && btn.HoverImage != nil {
				// Use hover image if available
				img = btn.HoverImage
			} else {
				// Use normal image
				img = btn.NormalImage
			}

			// Create draw options
			op := &ebiten.DrawImageOptions{}

			// Apply visual effects for hovered state (if no hover image available)
			if btn.State == components.UIHovered && btn.HoverImage == nil {
				// Make button brighter when hovered
				op.ColorM.Scale(1.2, 1.2, 1.2, 1.0)
			}

			// Position the button
			op.GeoM.Translate(btn.X, btn.Y)

			// Draw the button
			screen.DrawImage(img, op)
		}
	}
}

// initButtons initializes the menu buttons with their positions, images, and click handlers.
func (m *MainMenuScene) initButtons() {
	// Load button images using resource IDs
	adventureNormal, err := m.resourceManager.LoadImageByID("IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_BUTTON")
	if err != nil {
		log.Printf("Warning: Failed to load adventure button normal image: %v", err)
		adventureNormal = nil
	}

	adventureHover, err := m.resourceManager.LoadImageByID("IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_HIGHLIGHT")
	if err != nil {
		log.Printf("Warning: Failed to load adventure button hover image: %v", err)
		adventureHover = nil
	}

	// For exit button, we'll use a simple button image
	exitNormal, err := m.resourceManager.LoadImageByID("IMAGE_BUTTON_MIDDLE")
	if err != nil {
		log.Printf("Warning: Failed to load exit button image: %v", err)
		exitNormal = nil
	}

	// Calculate button positions (centered on screen)
	// Adventure button dimensions (estimate based on typical asset size)
	var adventureWidth, adventureHeight float64 = 200, 80
	if adventureNormal != nil {
		bounds := adventureNormal.Bounds()
		adventureWidth = float64(bounds.Dx())
		adventureHeight = float64(bounds.Dy())
	}

	// Exit button dimensions
	var exitWidth, exitHeight float64 = 150, 60
	if exitNormal != nil {
		bounds := exitNormal.Bounds()
		exitWidth = float64(bounds.Dx())
		exitHeight = float64(bounds.Dy())
	}

	// Position buttons vertically centered with spacing
	const buttonSpacing = 30.0
	adventureX := (WindowWidth - adventureWidth) / 2
	adventureY := WindowHeight/2 - adventureHeight - buttonSpacing/2

	exitX := (WindowWidth - exitWidth) / 2
	exitY := WindowHeight/2 + buttonSpacing/2

	// Initialize button array
	m.buttons = []components.Button{
		{
			X:           adventureX,
			Y:           adventureY,
			Width:       adventureWidth,
			Height:      adventureHeight,
			NormalImage: adventureNormal,
			HoverImage:  adventureHover,
			State:       components.UINormal,
			OnClick:     m.onStartAdventureClicked,
		},
		{
			X:           exitX,
			Y:           exitY,
			Width:       exitWidth,
			Height:      exitHeight,
			NormalImage: exitNormal,
			HoverImage:  nil, // Will use color/scale effects instead
			State:       components.UINormal,
			OnClick:     m.onExitClicked,
		},
	}
}

// onStartAdventureClicked handles the "Start Adventure" button click.
// It switches the current scene to the GameScene.
func (m *MainMenuScene) onStartAdventureClicked() {
	log.Println("Start Adventure button clicked")

	// Story 8.6: Load level from save file or default to 1-1
	levelToLoad := "1-1" // Default to first level
	gameState := game.GetGameState()
	saveManager := gameState.GetSaveManager()
	if err := saveManager.Load(); err == nil {
		// Save file exists, get highest level
		highestLevel := saveManager.GetHighestLevel()
		if highestLevel != "" {
			levelToLoad = highestLevel
			log.Printf("[MainMenu] Loading from save: highest level = %s", highestLevel)
		}
	}

	// Pass ResourceManager, SceneManager, and levelID to GameScene
	gameScene := NewGameScene(m.resourceManager, m.sceneManager, levelToLoad)
	m.sceneManager.SwitchTo(gameScene)
}

// onExitClicked handles the "Exit Game" button click.
// It terminates the application.
func (m *MainMenuScene) onExitClicked() {
	log.Println("Exit Game button clicked")
	os.Exit(0)
}

// isPointInRect checks if a point (px, py) is inside a rectangle defined by (x, y, width, height).
// Returns true if the point is within the rectangle bounds (inclusive), false otherwise.
func isPointInRect(px, py, x, y, width, height float64) bool {
	return px >= x && px <= x+width && py >= y && py <= y+height
}

// updateButtonVisibility updates the visibility of SelectorScreen buttons based on unlock status.
// This method controls which buttons are visible in the Reanim animation by setting the VisibleTracks whitelist.
//
// Unlock rules:
//   - Adventure mode: Always visible
//   - Challenges mode: Visible if level >= 3-2
//   - Vasebreaker mode: Visible if level >= 5-10
//   - Survival mode: Visible if level >= 5-10
//
// Story 12.1: Main Menu Tombstone System Enhancement
func (m *MainMenuScene) updateButtonVisibility() {
	if m.selectorScreenEntity == 0 {
		return // SelectorScreen entity not created, skip
	}

	// Get ReanimComponent from SelectorScreen entity
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](m.entityManager, m.selectorScreenEntity)
	if !ok {
		log.Printf("[MainMenuScene] Warning: SelectorScreen entity has no ReanimComponent")
		return
	}

	// 实验：完全移除 VisibleTracks 白名单，让所有轨道依赖动画定义轨道的 f 值自然控制
	// 这样可以验证 anim_grass, anim_open 等动画定义轨道是否能正确控制纯视觉轨道
	reanimComp.VisibleTracks = nil

	log.Printf("[MainMenuScene] 实验模式：移除 VisibleTracks 白名单，所有轨道依赖 f 值控制")
	log.Printf("[MainMenuScene] Button visibility (level=%s): Adventure=%v, Challenges=%v, Vasebreaker=%v, Survival=%v",
		m.currentLevel,
		config.IsMenuModeUnlocked(config.MenuButtonAdventure, m.currentLevel),
		config.IsMenuModeUnlocked(config.MenuButtonChallenges, m.currentLevel),
		config.IsMenuModeUnlocked(config.MenuButtonVasebreaker, m.currentLevel),
		config.IsMenuModeUnlocked(config.MenuButtonSurvival, m.currentLevel))
}

// onMenuButtonClicked handles clicks on SelectorScreen menu buttons.
// Checks unlock status and routes to appropriate handler.
//
// Parameters:
//   - buttonType: The type of button that was clicked
//
// Story 12.1: Main Menu Tombstone System Enhancement
func (m *MainMenuScene) onMenuButtonClicked(buttonType config.MenuButtonType) {
	log.Printf("[MainMenuScene] Button clicked: %v", buttonType)

	// Check if button is unlocked
	if !config.IsMenuModeUnlocked(buttonType, m.currentLevel) {
		log.Printf("[MainMenuScene] Button is locked (requires higher level)")

		// Play button click sound (shadow buttons also have click feedback)
		player, err := m.resourceManager.LoadSoundEffect("assets/sounds/buttonclick.ogg")
		if err != nil {
			log.Printf("[MainMenuScene] Warning: Failed to load button click sound: %v", err)
		} else {
			player.Rewind()
			player.Play()
		}

		// Show unlock dialog (Story 12.3 - temporarily use log)
		log.Printf("[MainMenuScene] TODO Story 12.3: Show unlock dialog for mode %v", buttonType)
		return
	}

	// Play button click sound
	// Note: SOUND_BUTTONCLICK should be loaded in initialization
	player, err := m.resourceManager.LoadSoundEffect("assets/sounds/buttonclick.ogg")
	if err != nil {
		log.Printf("[MainMenuScene] Warning: Failed to load button click sound: %v", err)
	} else {
		player.Rewind()
		player.Play()
	}

	// Route to appropriate handler based on button type
	switch buttonType {
	case config.MenuButtonAdventure:
		// Start adventure mode
		log.Printf("[MainMenuScene] Starting Adventure mode")
		m.onStartAdventureClicked()

	case config.MenuButtonChallenges:
		// TODO: Implement challenges/mini-games mode
		log.Printf("[MainMenuScene] Challenges mode - Not yet implemented")

	case config.MenuButtonVasebreaker:
		// TODO: Implement vasebreaker/puzzle mode
		log.Printf("[MainMenuScene] Vasebreaker mode - Not yet implemented")

	case config.MenuButtonSurvival:
		// TODO: Implement survival mode
		log.Printf("[MainMenuScene] Survival mode - Not yet implemented")

	default:
		log.Printf("[MainMenuScene] Warning: Unknown button type: %v", buttonType)
	}
}
