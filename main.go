package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

// Game represents the main game structure.
// It implements the ebiten.Game interface to provide the core game loop.
type Game struct{}

// Update updates the game logic.
// This method is called every tick (typically 60 times per second).
// Returns an error if the game should terminate.
func (g *Game) Update() error {
	return nil
}

// Draw renders the game screen.
// This method is called every frame to draw the game content.
func (g *Game) Draw(screen *ebiten.Image) {
	// Fill the screen with a light green color (lawn-like background)
	screen.Fill(color.RGBA{R: 144, G: 238, B: 144, A: 255})
}

// Layout returns the game's logical screen size.
// This size is independent of the actual window size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 800, 600
}

func main() {
	// Create a new game instance
	game := &Game{}

	// Set window properties
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("植物大战僵尸 - Go复刻版")

	// Start the game loop
	// This will call Update() and Draw() repeatedly until the window is closed
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
