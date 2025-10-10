package game

import (
	"github.com/decker502/pvz/pkg/scenes"
	"github.com/hajimehoshi/ebiten/v2"
)

// SceneManager manages the game's high-level state by controlling which scene is active.
// It ensures only one scene's Update and Draw methods are called at any given time.
type SceneManager struct {
	currentScene scenes.Scene
}

// NewSceneManager creates and returns a new SceneManager instance.
// The manager starts with no active scene; use SwitchTo to set the initial scene.
func NewSceneManager() *SceneManager {
	return &SceneManager{
		currentScene: nil,
	}
}

// SwitchTo changes the active scene to the provided scene.
// The new scene's Update and Draw methods will be called on subsequent game loop iterations.
func (sm *SceneManager) SwitchTo(scene scenes.Scene) {
	sm.currentScene = scene
}

// Update updates the currently active scene.
// If no scene is active, this method does nothing.
// deltaTime is the time elapsed since the last update in seconds.
func (sm *SceneManager) Update(deltaTime float64) {
	if sm.currentScene != nil {
		sm.currentScene.Update(deltaTime)
	}
}

// Draw renders the currently active scene to the provided screen.
// If no scene is active, this method does nothing.
func (sm *SceneManager) Draw(screen *ebiten.Image) {
	if sm.currentScene != nil {
		sm.currentScene.Draw(screen)
	}
}
