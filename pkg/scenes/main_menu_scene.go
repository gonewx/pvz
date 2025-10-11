package scenes

import (
	"image/color"
	"log"

	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

// MainMenuScene represents the main menu screen of the game.
// It displays when the game starts and allows the player to navigate to other scenes.
type MainMenuScene struct {
	resourceManager *game.ResourceManager
	backgroundImage *ebiten.Image
	bgmPlayer       *audio.Player
}

// NewMainMenuScene creates and returns a new MainMenuScene instance.
// It loads the main menu background image from assets.
//
// Parameters:
//   - rm: The ResourceManager instance used to load game resources.
//
// Returns:
//   - A pointer to the newly created MainMenuScene.
//
// If the background image fails to load, the scene will fall back to a solid color background.
func NewMainMenuScene(rm *game.ResourceManager) *MainMenuScene {
	scene := &MainMenuScene{
		resourceManager: rm,
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

	return scene
}

// Update updates the main menu scene logic.
// deltaTime is the time elapsed since the last update in seconds.
func (m *MainMenuScene) Update(deltaTime float64) {
	// Ensure background music is playing
	if m.bgmPlayer != nil && !m.bgmPlayer.IsPlaying() {
		m.bgmPlayer.Play()
	}

	// No other update logic needed for now
	// Future: Handle menu navigation, button clicks, etc.
}

// Draw renders the main menu scene to the screen.
// If a background image is loaded, it draws the image.
// Otherwise, it uses a dark blue fallback background.
func (m *MainMenuScene) Draw(screen *ebiten.Image) {
	if m.backgroundImage != nil {
		// Draw the background image
		screen.DrawImage(m.backgroundImage, nil)
	} else {
		// Fallback: Fill the screen with a dark blue color (midnight blue)
		screen.Fill(color.RGBA{R: 25, G: 25, B: 112, A: 255})
	}
}
