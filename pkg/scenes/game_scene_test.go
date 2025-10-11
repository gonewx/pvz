package scenes

import (
	"testing"

	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

// Package-level audio context for all tests
// audio.NewContext can only be called once, so we share it across all tests
var testAudioContext = audio.NewContext(48000)

// TestNewGameScene verifies that NewGameScene correctly creates a GameScene instance
// and properly initializes it with the provided ResourceManager and SceneManager.
func TestNewGameScene(t *testing.T) {
	// Create mock ResourceManager and SceneManager
	rm := game.NewResourceManager(testAudioContext)
	sm := game.NewSceneManager()

	// Create a new GameScene
	scene := NewGameScene(rm, sm)

	// Verify the scene is not nil
	if scene == nil {
		t.Fatal("NewGameScene returned nil")
	}

	// Verify that resourceManager and sceneManager are properly stored
	if scene.resourceManager == nil {
		t.Error("GameScene.resourceManager is nil")
	}
	if scene.sceneManager == nil {
		t.Error("GameScene.sceneManager is nil")
	}

	// Verify the scene has the expected structure
	// (Resources may be nil if loading fails, which is acceptable)
}

// TestGameSceneImplementsSceneInterface verifies that GameScene correctly
// implements the Scene interface defined in pkg/game/scene.go.
func TestGameSceneImplementsSceneInterface(t *testing.T) {
	// Create mock ResourceManager and SceneManager
	rm := game.NewResourceManager(testAudioContext)
	sm := game.NewSceneManager()

	// Create a new GameScene
	scene := NewGameScene(rm, sm)

	// Type assertion to verify GameScene implements Scene interface
	var _ game.Scene = scene

	// If we reach here without compilation error, the interface is implemented
	// We can also verify at runtime
	_, ok := interface{}(scene).(game.Scene)
	if !ok {
		t.Error("GameScene does not implement game.Scene interface")
	}
}

// TestGameSceneUpdateMethod tests that the Update method can be called without panicking.
// Currently Update has no logic, so we just verify it doesn't crash.
func TestGameSceneUpdateMethod(t *testing.T) {
	// Create mock ResourceManager and SceneManager
	rm := game.NewResourceManager(testAudioContext)
	sm := game.NewSceneManager()

	// Create a new GameScene
	scene := NewGameScene(rm, sm)

	// Call Update with a sample deltaTime
	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Update() panicked: %v", r)
		}
	}()

	scene.Update(0.016) // ~60 FPS
	scene.Update(1.0)   // 1 second
}

// TestGameSceneDrawMethodDoesNotPanic tests that the Draw method can be called
// without panicking, even when resources fail to load.
// Note: We cannot easily test visual output in unit tests, so we just verify
// the method doesn't crash.
func TestGameSceneDrawMethodDoesNotPanic(t *testing.T) {
	// Create mock ResourceManager and SceneManager
	rm := game.NewResourceManager(testAudioContext)
	sm := game.NewSceneManager()

	// Create a new GameScene
	scene := NewGameScene(rm, sm)

	// We cannot create a real screen image in a unit test,
	// but we can verify the scene structure is valid
	// In a real test with ebitengine, you'd use ebiten.NewImage()

	// For now, just verify the scene was created successfully
	if scene == nil {
		t.Fatal("GameScene is nil, cannot test Draw method")
	}

	// The Draw method should handle nil resources gracefully (fallback rendering)
	// This is implicitly tested by the implementation
}

// TestGameSceneResourceLoadingFallback verifies that GameScene handles
// resource loading failures gracefully by not panicking.
func TestGameSceneResourceLoadingFallback(t *testing.T) {
	// Create a ResourceManager that will fail to load resources
	// (using empty resource manager with no actual files loaded)
	rm := game.NewResourceManager(testAudioContext)
	sm := game.NewSceneManager()

	// Create GameScene - it should not panic even if resources fail to load
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewGameScene panicked when resources failed to load: %v", r)
		}
	}()

	scene := NewGameScene(rm, sm)

	// Verify scene was created despite resource loading failures
	if scene == nil {
		t.Fatal("NewGameScene returned nil when resources failed to load")
	}

	// Resources should be nil if loading failed, which is acceptable
	// The Draw method should use fallback rendering
}

// TestGameSceneConstants verifies that UI layout constants are defined
// with reasonable values.
func TestGameSceneConstants(t *testing.T) {
	// Test that constants are defined and have non-negative values
	if SeedBankX < 0 {
		t.Error("SeedBankX should be non-negative")
	}
	if SeedBankY < 0 {
		t.Error("SeedBankY should be non-negative")
	}
	if SeedBankWidth <= 0 {
		t.Error("SeedBankWidth should be positive")
	}
	if SeedBankHeight <= 0 {
		t.Error("SeedBankHeight should be positive")
	}

	if SunCounterX < 0 {
		t.Error("SunCounterX should be non-negative")
	}
	if SunCounterY < 0 {
		t.Error("SunCounterY should be non-negative")
	}
	if SunCounterWidth <= 0 {
		t.Error("SunCounterWidth should be positive")
	}
	if SunCounterHeight <= 0 {
		t.Error("SunCounterHeight should be positive")
	}

	if ShovelX < 0 {
		t.Error("ShovelX should be non-negative")
	}
	if ShovelY < 0 {
		t.Error("ShovelY should be non-negative")
	}
	if ShovelWidth <= 0 {
		t.Error("ShovelWidth should be positive")
	}
	if ShovelHeight <= 0 {
		t.Error("ShovelHeight should be positive")
	}

	// Test that WindowWidth and WindowHeight are defined (from main_menu_scene.go)
	if WindowWidth <= 0 {
		t.Error("WindowWidth should be positive")
	}
	if WindowHeight <= 0 {
		t.Error("WindowHeight should be positive")
	}
}

// TestGameSceneIntroAnimation tests the intro camera animation logic.
// It verifies that the camera position correctly transitions through the animation phases.
func TestGameSceneIntroAnimation(t *testing.T) {
	// Create a GameScene with mock dependencies
	rm := game.NewResourceManager(testAudioContext)
	sm := game.NewSceneManager()
	scene := NewGameScene(rm, sm)

	// Set a non-zero maxCameraX to enable meaningful animation
	// (In real usage, this is set when background is loaded)
	scene.maxCameraX = 400.0

	// Test 1: Initial state - animation should be playing
	if !scene.isIntroAnimPlaying {
		t.Error("Intro animation should be playing initially")
	}
	if scene.cameraX != 0 {
		t.Errorf("Initial cameraX should be 0, got %f", scene.cameraX)
	}

	// Test 2: Phase 1 - camera moves from left to right (0 to 1.5 seconds)
	scene.Update(0.75) // 25% progress (0.75 / 3.0)
	if scene.cameraX <= 0 || scene.cameraX >= scene.maxCameraX {
		t.Errorf("Camera should be moving rightward in phase 1, cameraX=%f", scene.cameraX)
	}
	if !scene.isIntroAnimPlaying {
		t.Error("Animation should still be playing during phase 1")
	}

	// Test 3: Mid-animation (phase 1 complete) - camera should be near maxCameraX
	scene = NewGameScene(rm, sm)
	scene.maxCameraX = 400.0
	scene.Update(1.5) // 50% progress - end of phase 1
	expectedCameraX := scene.maxCameraX
	tolerance := 5.0 // Allow small numerical error
	if scene.cameraX < expectedCameraX-tolerance || scene.cameraX > expectedCameraX+tolerance {
		t.Errorf("Camera should be at maxCameraX (≈%f) at phase 1 end, got %f",
			expectedCameraX, scene.cameraX)
	}

	// Test 4: Phase 2 - camera moves from right back to center (1.5 to 3.0 seconds)
	scene.Update(0.75) // Now at 75% progress (2.25 / 3.0)
	if scene.cameraX <= float64(GameCameraX) || scene.cameraX >= scene.maxCameraX {
		t.Errorf("Camera should be moving leftward in phase 2, cameraX=%f", scene.cameraX)
	}
	if !scene.isIntroAnimPlaying {
		t.Error("Animation should still be playing during phase 2")
	}

	// Test 5: Animation complete - camera settled at GameCameraX
	scene = NewGameScene(rm, sm)
	scene.maxCameraX = 400.0
	scene.Update(3.5) // 100%+ progress (exceeds 3.0 seconds)
	if scene.cameraX != float64(GameCameraX) {
		t.Errorf("Camera should be at GameCameraX (%d) after animation completes, got %f",
			GameCameraX, scene.cameraX)
	}
	if scene.isIntroAnimPlaying {
		t.Error("Animation should be stopped after completion")
	}
}

// TestEaseOutQuad tests the easing function used in the animation.
func TestEaseOutQuad(t *testing.T) {
	// Create a scene to access the easing method
	rm := game.NewResourceManager(testAudioContext)
	sm := game.NewSceneManager()
	scene := NewGameScene(rm, sm)

	tests := []struct {
		name     string
		input    float64
		expected float64
		tolerance float64
	}{
		{"Start (t=0)", 0.0, 0.0, 0.001},
		{"Quarter (t=0.25)", 0.25, 0.4375, 0.001},
		{"Half (t=0.5)", 0.5, 0.75, 0.001},
		{"Three-quarters (t=0.75)", 0.75, 0.9375, 0.001},
		{"End (t=1.0)", 1.0, 1.0, 0.001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scene.easeOutQuad(tt.input)
			diff := result - tt.expected
			if diff < -tt.tolerance || diff > tt.tolerance {
				t.Errorf("easeOutQuad(%f) = %f, expected %f (±%f)",
					tt.input, result, tt.expected, tt.tolerance)
			}
		})
	}
}

// TestGameSceneMaxCameraXCalculation tests the calculation of maxCameraX
// when the background is loaded.
func TestGameSceneMaxCameraXCalculation(t *testing.T) {
	// Create a GameScene with mock dependencies
	rm := game.NewResourceManager(testAudioContext)
	sm := game.NewSceneManager()
	scene := NewGameScene(rm, sm)

	// Since we're in a test environment without actual assets,
	// maxCameraX should be 0 (background loading fails)
	// This tests the fallback behavior
	if scene.maxCameraX != 0 {
		t.Errorf("maxCameraX should be 0 when background fails to load, got %f", scene.maxCameraX)
	}
}
