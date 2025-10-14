package game

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// Global audio context shared by all tests
// Ebitengine only allows one audio context to be created
var testAudioContext *audio.Context

// TestMain sets up the shared audio context before running tests
func TestMain(m *testing.M) {
	// Create the audio context once for all tests
	testAudioContext = audio.NewContext(48000)

	// Run all tests
	exitCode := m.Run()

	// Exit with the test result code
	os.Exit(exitCode)
}

// createTestImage creates a simple test PNG image for testing purposes.
func createTestImage(path string) error {
	// Create a simple 10x10 blue image
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	blue := color.RGBA{R: 0, G: 0, B: 255, A: 255}
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, blue)
		}
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Save the image
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

// TestNewResourceManager tests the creation of a new ResourceManager instance.
func TestNewResourceManager(t *testing.T) {
	rm := NewResourceManager(testAudioContext)

	if rm == nil {
		t.Fatal("NewResourceManager returned nil")
	}

	if rm.imageCache == nil {
		t.Error("imageCache is nil")
	}

	if rm.audioCache == nil {
		t.Error("audioCache is nil")
	}

	if rm.audioContext != testAudioContext {
		t.Error("audioContext not set correctly")
	}
}

// TestLoadImage_Success tests successful image loading.
func TestLoadImage_Success(t *testing.T) {
	// Setup: Create a test image
	testImagePath := "testdata/test.png"
	if err := createTestImage(testImagePath); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	defer os.RemoveAll("testdata") // Cleanup

	// Create ResourceManager
	rm := NewResourceManager(testAudioContext)

	// Test: Load the image
	img, err := rm.LoadImage(testImagePath)
	if err != nil {
		t.Fatalf("LoadImage failed: %v", err)
	}

	if img == nil {
		t.Fatal("LoadImage returned nil image")
	}

	// Verify dimensions
	bounds := img.Bounds()
	if bounds.Dx() != 10 || bounds.Dy() != 10 {
		t.Errorf("Image dimensions incorrect: got %dx%d, want 10x10", bounds.Dx(), bounds.Dy())
	}
}

// TestLoadImage_CachingMechanism tests that images are cached properly.
func TestLoadImage_CachingMechanism(t *testing.T) {
	// Setup: Create a test image
	testImagePath := "testdata/test_cache.png"
	if err := createTestImage(testImagePath); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	defer os.RemoveAll("testdata") // Cleanup

	// Create ResourceManager
	// Use shared testAudioContext
	rm := NewResourceManager(testAudioContext)

	// Load the image twice
	img1, err1 := rm.LoadImage(testImagePath)
	if err1 != nil {
		t.Fatalf("First LoadImage failed: %v", err1)
	}

	img2, err2 := rm.LoadImage(testImagePath)
	if err2 != nil {
		t.Fatalf("Second LoadImage failed: %v", err2)
	}

	// Verify they are the same instance (cached)
	if img1 != img2 {
		t.Error("Images are not cached - different instances returned")
	}
}

// TestLoadImage_FileNotFound tests error handling when file doesn't exist.
func TestLoadImage_FileNotFound(t *testing.T) {
	// Use shared testAudioContext
	rm := NewResourceManager(testAudioContext)

	// Test: Try to load a non-existent image
	_, err := rm.LoadImage("nonexistent.png")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

// TestLoadImage_InvalidFormat tests error handling for invalid image format.
func TestLoadImage_InvalidFormat(t *testing.T) {
	// Setup: Create an invalid image file
	invalidPath := "testdata/invalid.png"
	if err := os.MkdirAll("testdata", 0755); err != nil {
		t.Fatalf("Failed to create testdata directory: %v", err)
	}
	defer os.RemoveAll("testdata") // Cleanup

	// Write some invalid data
	if err := os.WriteFile(invalidPath, []byte("not a valid png"), 0644); err != nil {
		t.Fatalf("Failed to create invalid file: %v", err)
	}

	// Use shared testAudioContext
	rm := NewResourceManager(testAudioContext)

	// Test: Try to load the invalid image
	_, err := rm.LoadImage(invalidPath)
	if err == nil {
		t.Error("Expected error for invalid image format, got nil")
	}
}

// TestGetImage tests retrieving images from cache.
func TestGetImage(t *testing.T) {
	// Setup: Create a test image
	testImagePath := "testdata/test_get.png"
	if err := createTestImage(testImagePath); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	defer os.RemoveAll("testdata") // Cleanup

	// Use shared testAudioContext
	rm := NewResourceManager(testAudioContext)

	// Test: Get image before loading - should be nil
	img := rm.GetImage(testImagePath)
	if img != nil {
		t.Error("GetImage should return nil for non-loaded image")
	}

	// Load the image
	loadedImg, err := rm.LoadImage(testImagePath)
	if err != nil {
		t.Fatalf("LoadImage failed: %v", err)
	}

	// Test: Get image after loading - should return the same instance
	cachedImg := rm.GetImage(testImagePath)
	if cachedImg == nil {
		t.Error("GetImage returned nil for loaded image")
	}

	if cachedImg != loadedImg {
		t.Error("GetImage returned different instance than LoadImage")
	}
}

// TestLoadAudio_FileNotFound tests audio loading with non-existent file.
func TestLoadAudio_FileNotFound(t *testing.T) {
	// Use shared testAudioContext
	rm := NewResourceManager(testAudioContext)

	// Test: Try to load a non-existent audio file
	_, err := rm.LoadAudio("nonexistent.mp3")
	if err == nil {
		t.Error("Expected error for non-existent audio file, got nil")
	}
}

// TestLoadAudio_UnsupportedFormat tests audio loading with unsupported format.
func TestLoadAudio_UnsupportedFormat(t *testing.T) {
	// Setup: Create a dummy file with unsupported extension
	unsupportedPath := "testdata/test.wav"
	if err := os.MkdirAll("testdata", 0755); err != nil {
		t.Fatalf("Failed to create testdata directory: %v", err)
	}
	defer os.RemoveAll("testdata") // Cleanup

	if err := os.WriteFile(unsupportedPath, []byte("dummy data"), 0644); err != nil {
		t.Fatalf("Failed to create dummy file: %v", err)
	}

	// Use shared testAudioContext
	rm := NewResourceManager(testAudioContext)

	// Test: Try to load the unsupported format
	_, err := rm.LoadAudio(unsupportedPath)
	if err == nil {
		t.Error("Expected error for unsupported audio format, got nil")
	}
}

// TestGetAudioPlayer tests retrieving audio players from cache.
func TestGetAudioPlayer(t *testing.T) {
	// Use shared testAudioContext
	rm := NewResourceManager(testAudioContext)

	// Test: Get audio player before loading - should be nil
	player := rm.GetAudioPlayer("test.mp3")
	if player != nil {
		t.Error("GetAudioPlayer should return nil for non-loaded audio")
	}
}

// TestLoadParticleConfig_Success tests successful particle configuration loading.
func TestLoadParticleConfig_Success(t *testing.T) {
	rm := NewResourceManager(testAudioContext)

	// Test: Load a real particle configuration file
	config, err := rm.LoadParticleConfig("../../assets/effect/particles/BlastMark")
	if err != nil {
		t.Fatalf("LoadParticleConfig failed: %v", err)
	}

	if config == nil {
		t.Fatal("LoadParticleConfig returned nil config")
	}

	// Verify the configuration has at least one emitter
	if len(config.Emitters) == 0 {
		t.Error("Configuration has no emitters")
	}

	// BlastMark.xml should have exactly 1 emitter
	if len(config.Emitters) != 1 {
		t.Errorf("Expected 1 emitter, got %d", len(config.Emitters))
	}
}

// TestLoadParticleConfig_Caching tests that particle configurations are cached properly.
func TestLoadParticleConfig_Caching(t *testing.T) {
	rm := NewResourceManager(testAudioContext)

	configName := "../../assets/effect/particles/BlastMark"

	// Load the same configuration twice
	config1, err1 := rm.LoadParticleConfig(configName)
	if err1 != nil {
		t.Fatalf("First LoadParticleConfig failed: %v", err1)
	}

	config2, err2 := rm.LoadParticleConfig(configName)
	if err2 != nil {
		t.Fatalf("Second LoadParticleConfig failed: %v", err2)
	}

	// Verify they are the same instance (cached)
	if config1 != config2 {
		t.Error("Configurations are not cached - different instances returned")
	}
}

// TestLoadParticleConfig_FileNotFound tests error handling when file doesn't exist.
func TestLoadParticleConfig_FileNotFound(t *testing.T) {
	rm := NewResourceManager(testAudioContext)

	// Test: Try to load a non-existent configuration
	_, err := rm.LoadParticleConfig("NonExistentParticle")
	if err == nil {
		t.Error("Expected error for non-existent particle config, got nil")
	}
}

// TestGetParticleConfig tests retrieving particle configurations from cache.
func TestGetParticleConfig(t *testing.T) {
	rm := NewResourceManager(testAudioContext)

	configName := "../../assets/effect/particles/BlastMark"

	// Test: Get config before loading - should be nil
	config := rm.GetParticleConfig(configName)
	if config != nil {
		t.Error("GetParticleConfig should return nil for non-loaded config")
	}

	// Load the configuration
	loadedConfig, err := rm.LoadParticleConfig(configName)
	if err != nil {
		t.Fatalf("LoadParticleConfig failed: %v", err)
	}

	// Test: Get config after loading - should return the same instance
	cachedConfig := rm.GetParticleConfig(configName)
	if cachedConfig == nil {
		t.Error("GetParticleConfig returned nil for loaded config")
	}

	if cachedConfig != loadedConfig {
		t.Error("GetParticleConfig returned different instance than LoadParticleConfig")
	}
}

// TestLoadImage_ParticleTexture tests loading particle texture images.
func TestLoadImage_ParticleTexture(t *testing.T) {
	rm := NewResourceManager(testAudioContext)

	// Test: Load an actual particle texture
	img, err := rm.LoadImage("../../assets/particles/BlastMark.png")
	if err != nil {
		t.Fatalf("Failed to load particle texture: %v", err)
	}

	if img == nil {
		t.Fatal("LoadImage returned nil for particle texture")
	}

	// Verify the image has non-zero dimensions
	bounds := img.Bounds()
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		t.Errorf("Particle texture has invalid dimensions: %dx%d", bounds.Dx(), bounds.Dy())
	}
}

// TestLoadParticleConfig_MultipleEmitters tests loading a configuration with multiple emitters.
func TestLoadParticleConfig_MultipleEmitters(t *testing.T) {
	rm := NewResourceManager(testAudioContext)

	// Test: Load Award.xml which has 13 emitters
	config, err := rm.LoadParticleConfig("../../assets/effect/particles/Award")
	if err != nil {
		t.Fatalf("LoadParticleConfig failed for Award: %v", err)
	}

	// Award.xml should have 13 emitters
	expectedEmitters := 13
	if len(config.Emitters) != expectedEmitters {
		t.Errorf("Expected %d emitters in Award.xml, got %d", expectedEmitters, len(config.Emitters))
	}

	// Verify at least one emitter has expected properties
	if len(config.Emitters) > 0 {
		firstEmitter := config.Emitters[0]
		if firstEmitter.Name == "" {
			t.Error("First emitter should have a name")
		}
		if firstEmitter.Image == "" {
			t.Error("First emitter should have an image reference")
		}
	}
}
