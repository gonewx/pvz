package game

import (
	"os"
	"testing"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// TestLoadResourceConfig tests loading the YAML resource configuration
func TestLoadResourceConfig(t *testing.T) {
	// Skip if the resource config file doesn't exist
	configPath := "../../assets/config/resources.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("Skipping test - resource config file not found:", configPath)
	}

	// Create a ResourceManager (without audio context for this test)
	rm := &ResourceManager{
		imageCache:       make(map[string]*ebiten.Image),
		audioCache:       make(map[string]*audio.Player),
		fontFaceCache:    make(map[string]*text.GoTextFace),
		reanimXMLCache:   make(map[string]*reanim.ReanimXML),
		reanimImageCache: make(map[string]map[string]*ebiten.Image),
		resourceMap:      make(map[string]string),
	}

	// Load the resource configuration
	err := rm.LoadResourceConfig(configPath)
	if err != nil {
		t.Fatalf("LoadResourceConfig failed: %v", err)
	}

	// Verify config was loaded
	if rm.config == nil {
		t.Fatal("Config is nil after loading")
	}

	// Verify base path
	if rm.config.BasePath != "assets" {
		t.Errorf("Expected base_path 'assets', got '%s'", rm.config.BasePath)
	}

	// Verify some groups exist
	expectedGroups := []string{"init", "loadingimages", "loadingsounds"}
	for _, groupName := range expectedGroups {
		if _, exists := rm.config.Groups[groupName]; !exists {
			t.Errorf("Expected group '%s' not found in config", groupName)
		}
	}

	// Verify resource map was built
	if len(rm.resourceMap) == 0 {
		t.Error("Resource map is empty after loading config")
	}

	// Verify some specific resource IDs are mapped
	expectedIDs := []string{"IMAGE_BLANK", "SOUND_BUTTONCLICK"}
	for _, id := range expectedIDs {
		if _, exists := rm.resourceMap[id]; !exists {
			t.Errorf("Expected resource ID '%s' not found in resource map", id)
		}
	}
}

// TestLoadImageByID tests loading an image by resource ID
func TestLoadImageByID(t *testing.T) {
	// Skip if the resource config file doesn't exist
	configPath := "../../assets/config/resources.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("Skipping test - resource config file not found:", configPath)
	}

	// Create a ResourceManager
	rm := &ResourceManager{
		imageCache:       make(map[string]*ebiten.Image),
		audioCache:       make(map[string]*audio.Player),
		fontFaceCache:    make(map[string]*text.GoTextFace),
		reanimXMLCache:   make(map[string]*reanim.ReanimXML),
		reanimImageCache: make(map[string]map[string]*ebiten.Image),
		resourceMap:      make(map[string]string),
	}

	// Test loading without config - should fail
	_, err := rm.LoadImageByID("IMAGE_BLANK")
	if err == nil {
		t.Error("Expected error when loading image without config")
	}

	// Load config
	if err := rm.LoadResourceConfig(configPath); err != nil {
		t.Fatalf("LoadResourceConfig failed: %v", err)
	}

	// Test loading non-existent resource ID - should fail
	_, err = rm.LoadImageByID("NON_EXISTENT_ID")
	if err == nil {
		t.Error("Expected error when loading non-existent resource ID")
	}

	// Note: We can't test actual image loading in unit tests without the actual files
	// Integration tests would cover that
}

// TestGetImageByID tests retrieving a cached image by resource ID
func TestGetImageByID(t *testing.T) {
	// Create a ResourceManager
	rm := &ResourceManager{
		imageCache:       make(map[string]*ebiten.Image),
		audioCache:       make(map[string]*audio.Player),
		fontFaceCache:    make(map[string]*text.GoTextFace),
		reanimXMLCache:   make(map[string]*reanim.ReanimXML),
		reanimImageCache: make(map[string]map[string]*ebiten.Image),
		resourceMap:      make(map[string]string),
	}

	// Test without config loaded
	img := rm.GetImageByID("IMAGE_BLANK")
	if img != nil {
		t.Error("Expected nil when getting image without config")
	}

	// Skip if the resource config file doesn't exist
	configPath := "../../assets/config/resources.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("Skipping test - resource config file not found:", configPath)
	}

	// Load config
	if err := rm.LoadResourceConfig(configPath); err != nil {
		t.Fatalf("LoadResourceConfig failed: %v", err)
	}

	// Test getting non-existent resource ID
	img = rm.GetImageByID("NON_EXISTENT_ID")
	if img != nil {
		t.Error("Expected nil for non-existent resource ID")
	}
}

// TestBuildFullPath tests the buildFullPath helper function
func TestBuildFullPath(t *testing.T) {
	tests := []struct {
		basePath     string
		relativePath string
		expected     string
	}{
		{"assets", "images/background1.png", "assets/images/background1.png"},
		{"assets", "/images/background1.png", "assets/images/background1.png"},
		{"", "images/background1.png", "images/background1.png"},
		{"assets", "properties/blank", "assets/properties/blank"},
	}

	for _, test := range tests {
		result := buildFullPath(test.basePath, test.relativePath)
		if result != test.expected {
			t.Errorf("buildFullPath(%q, %q) = %q, expected %q",
				test.basePath, test.relativePath, result, test.expected)
		}
	}
}
