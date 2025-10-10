package scenes

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// GameScene represents the main gameplay screen.
// This is where the actual Plants vs Zombies gameplay will occur.
type GameScene struct{}

// NewGameScene creates and returns a new GameScene instance.
func NewGameScene() *GameScene {
	return &GameScene{}
}

// Update updates the game scene logic.
// deltaTime is the time elapsed since the last update in seconds.
func (g *GameScene) Update(deltaTime float64) {
	// No update logic needed for now
	// Future: Update ECS systems, handle game logic, etc.
}

// Draw renders the game scene to the screen.
// Uses a dark gray background to distinguish it from other scenes.
func (g *GameScene) Draw(screen *ebiten.Image) {
	// Fill the screen with a dark gray color
	screen.Fill(color.RGBA{R: 64, G: 64, B: 64, A: 255})
}
