package main

import (
	"log"

	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/scenes"
	"github.com/hajimehoshi/ebiten/v2"
)

// Game represents the main game structure.
// It implements the ebiten.Game interface to provide the core game loop.
type Game struct {
	sceneManager *game.SceneManager
}

// Update updates the game logic.
// This method is called every tick (typically 60 times per second).
// Returns an error if the game should terminate.
func (g *Game) Update() error {
	// Calculate delta time (assuming 60 TPS - Ticks Per Second)
	deltaTime := 1.0 / 60.0
	g.sceneManager.Update(deltaTime)
	return nil
}

// Draw renders the game screen.
// This method is called every frame to draw the game content.
func (g *Game) Draw(screen *ebiten.Image) {
	g.sceneManager.Draw(screen)
}

// Layout returns the game's logical screen size.
// This size is independent of the actual window size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 800, 600
}

func main() {
	// Create scene manager
	sceneManager := game.NewSceneManager()

	// Create and set the initial scene (Main Menu)
	mainMenuScene := scenes.NewMainMenuScene()
	sceneManager.SwitchTo(mainMenuScene)

	// Create a new game instance with the scene manager
	gameInstance := &Game{
		sceneManager: sceneManager,
	}

	// Set window properties
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("植物大战僵尸 - Go复刻版")

	// Start the game loop
	// This will call Update() and Draw() repeatedly until the window is closed
	if err := ebiten.RunGame(gameInstance); err != nil {
		log.Fatal(err)
	}
}
