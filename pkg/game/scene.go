package game

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Scene represents a game scene (e.g., main menu, gameplay, pause menu).
// Each scene has its own update and rendering logic.
type Scene interface {
	// Update updates the scene logic based on the elapsed time.
	// deltaTime is the time elapsed since the last update in seconds.
	Update(deltaTime float64)

	// Draw renders the scene to the provided screen.
	// screen is the target image where the scene should be drawn.
	Draw(screen *ebiten.Image)
}
