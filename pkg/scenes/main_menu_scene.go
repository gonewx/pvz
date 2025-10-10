package scenes

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// MainMenuScene represents the main menu screen of the game.
// It displays when the game starts and allows the player to navigate to other scenes.
type MainMenuScene struct{}

// NewMainMenuScene creates and returns a new MainMenuScene instance.
func NewMainMenuScene() *MainMenuScene {
	return &MainMenuScene{}
}

// Update updates the main menu scene logic.
// deltaTime is the time elapsed since the last update in seconds.
func (m *MainMenuScene) Update(deltaTime float64) {
	// No update logic needed for now
	// Future: Handle menu navigation, button clicks, etc.
}

// Draw renders the main menu scene to the screen.
// Uses a dark blue background to distinguish it from other scenes.
func (m *MainMenuScene) Draw(screen *ebiten.Image) {
	// Fill the screen with a dark blue color (midnight blue)
	screen.Fill(color.RGBA{R: 25, G: 25, B: 112, A: 255})
}
