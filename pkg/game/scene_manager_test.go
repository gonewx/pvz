package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// MockScene is a mock implementation of the Scene interface for testing.
type MockScene struct {
	updateCalled bool
	drawCalled   bool
	deltaTime    float64
}

// Update records that Update was called and stores the deltaTime.
func (m *MockScene) Update(deltaTime float64) {
	m.updateCalled = true
	m.deltaTime = deltaTime
}

// Draw records that Draw was called.
func (m *MockScene) Draw(screen *ebiten.Image) {
	m.drawCalled = true
}

// TestNewSceneManager verifies that NewSceneManager creates a valid instance.
func TestNewSceneManager(t *testing.T) {
	sm := NewSceneManager()
	if sm == nil {
		t.Fatal("NewSceneManager() returned nil")
	}
	if sm.currentScene != nil {
		t.Error("Expected currentScene to be nil initially")
	}
}

// TestSceneManagerSwitchTo verifies that SwitchTo correctly changes the active scene.
func TestSceneManagerSwitchTo(t *testing.T) {
	sm := NewSceneManager()
	mockScene := &MockScene{}

	sm.SwitchTo(mockScene)

	if sm.currentScene != mockScene {
		t.Error("SwitchTo did not set the current scene correctly")
	}
}

// TestSceneManagerUpdate verifies that Update calls the current scene's Update method.
func TestSceneManagerUpdate(t *testing.T) {
	sm := NewSceneManager()
	mockScene := &MockScene{}
	sm.SwitchTo(mockScene)

	deltaTime := 0.016 // ~60 FPS
	sm.Update(deltaTime)

	if !mockScene.updateCalled {
		t.Error("Scene's Update method was not called")
	}
	if mockScene.deltaTime != deltaTime {
		t.Errorf("Expected deltaTime %.3f, got %.3f", deltaTime, mockScene.deltaTime)
	}
}

// TestSceneManagerUpdateNoScene verifies that Update handles nil scene gracefully.
func TestSceneManagerUpdateNoScene(t *testing.T) {
	sm := NewSceneManager()
	// Don't set any scene, currentScene should be nil
	sm.Update(0.016) // Should not panic
}

// TestSceneManagerDraw verifies that Draw calls the current scene's Draw method.
func TestSceneManagerDraw(t *testing.T) {
	sm := NewSceneManager()
	mockScene := &MockScene{}
	sm.SwitchTo(mockScene)

	// Create a dummy screen image
	screen := ebiten.NewImage(800, 600)
	sm.Draw(screen)

	if !mockScene.drawCalled {
		t.Error("Scene's Draw method was not called")
	}
}

// TestSceneManagerDrawNoScene verifies that Draw handles nil scene gracefully.
func TestSceneManagerDrawNoScene(t *testing.T) {
	sm := NewSceneManager()
	screen := ebiten.NewImage(800, 600)
	// Don't set any scene, currentScene should be nil
	sm.Draw(screen) // Should not panic
}

// TestSceneManagerSwitchBetweenScenes verifies switching between multiple scenes.
func TestSceneManagerSwitchBetweenScenes(t *testing.T) {
	sm := NewSceneManager()
	scene1 := &MockScene{}
	scene2 := &MockScene{}

	// Switch to scene1
	sm.SwitchTo(scene1)
	sm.Update(0.016)

	if !scene1.updateCalled {
		t.Error("Scene1's Update was not called")
	}
	if scene2.updateCalled {
		t.Error("Scene2's Update should not have been called yet")
	}

	// Switch to scene2
	sm.SwitchTo(scene2)
	sm.Update(0.016)

	if !scene2.updateCalled {
		t.Error("Scene2's Update was not called after switching")
	}
}
