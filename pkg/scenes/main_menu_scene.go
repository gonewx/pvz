package scenes

import (
	"image/color"
	"log"
	"os"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/game"
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

	// Load background image
	img, err := rm.LoadImage("assets/images/interface/MainMenu.png")
	if err != nil {
		log.Printf("Warning: Failed to load main menu background: %v", err)
		log.Printf("The game will use a fallback solid color background")
		// Fallback: keep backgroundImage as nil, will use solid color in Draw()
	} else {
		scene.backgroundImage = img
	}

	// Load background music
	player, err := rm.LoadAudio("assets/audio/Music/mainmenubgm.mp3")
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

	// Get mouse position
	mouseX, mouseY := ebiten.CursorPosition()

	// Check if mouse button is currently pressed
	isMousePressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	// Detect click edge (button was just pressed this frame)
	isMouseClicked := isMousePressed && !m.wasMousePressed

	// Update button states based on mouse position and clicks
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
	// Draw background
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

	// Draw buttons
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

// initButtons initializes the menu buttons with their positions, images, and click handlers.
func (m *MainMenuScene) initButtons() {
	// Load button images
	adventureNormal, err := m.resourceManager.LoadImage("assets/images/game_menu/Adventure_0.png")
	if err != nil {
		log.Printf("Warning: Failed to load adventure button normal image: %v", err)
		adventureNormal = nil
	}

	adventureHover, err := m.resourceManager.LoadImage("assets/images/game_menu/Adventure_1.png")
	if err != nil {
		log.Printf("Warning: Failed to load adventure button hover image: %v", err)
		adventureHover = nil
	}

	// For exit button, we'll use a simple button image
	exitNormal, err := m.resourceManager.LoadImage("assets/images/interface/button.png")
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
	gameScene := NewGameScene()
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
