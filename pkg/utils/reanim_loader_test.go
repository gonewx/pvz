package utils

import (
	"testing"

	"github.com/decker502/pvz/internal/reanim"
)

// TestMapImageNameToFile tests the image reference to file name mapping
func TestMapImageNameToFile(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{
			name:     "PeaShooter head",
			ref:      "IMAGE_REANIM_PEASHOOTER_HEAD",
			expected: "peashooter_head.png",
		},
		{
			name:     "SunFlower body",
			ref:      "IMAGE_REANIM_SUNFLOWER_BODY",
			expected: "sunflower_body.png",
		},
		{
			name:     "Wallnut",
			ref:      "IMAGE_REANIM_WALLNUT",
			expected: "wallnut.png",
		},
		{
			name:     "Reference without prefix",
			ref:      "PEASHOOTER_HEAD",
			expected: "peashooter_head.png",
		},
		{
			name:     "Empty reference",
			ref:      "",
			expected: ".png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapImageNameToFile(tt.ref)
			if result != tt.expected {
				t.Errorf("mapImageNameToFile(%s) = %s, expected %s", tt.ref, result, tt.expected)
			}
		})
	}
}

// TestCollectImageReferences tests that image references are correctly collected
func TestCollectImageReferences(t *testing.T) {
	// Create a test reanim with some image references
	reanimData := &reanim.ReanimXML{
		FPS: 12,
		Tracks: []reanim.Track{
			{
				Name: "head",
				Frames: []reanim.Frame{
					{ImagePath: "IMAGE_REANIM_PEASHOOTER_HEAD"},
					{ImagePath: ""}, // Empty reference should not be collected
					{ImagePath: "IMAGE_REANIM_PEASHOOTER_HEAD"}, // Duplicate
				},
			},
			{
				Name: "backleaf",
				Frames: []reanim.Frame{
					{ImagePath: "IMAGE_REANIM_PEASHOOTER_BACKLEAF"},
				},
			},
		},
	}

	refs := collectImageReferences(reanimData)

	// Should have 2 unique references (empty string is not collected)
	expectedRefs := map[string]bool{
		"IMAGE_REANIM_PEASHOOTER_HEAD":     true,
		"IMAGE_REANIM_PEASHOOTER_BACKLEAF": true,
	}

	if len(refs) != len(expectedRefs) {
		t.Errorf("Expected %d unique references, got %d", len(expectedRefs), len(refs))
	}

	for ref := range expectedRefs {
		if !refs[ref] {
			t.Errorf("Expected reference '%s' not found", ref)
		}
	}

	// Verify empty string is not collected
	if refs[""] {
		t.Errorf("Empty reference should not be collected")
	}
}

// TestLoadReanimImages_Success tests successful image loading
func TestLoadReanimImages_Success(t *testing.T) {
	// Parse a real reanim file
	reanimData, err := reanim.ParseReanimFile("../../assets/effect/reanim/PeaShooter.reanim")
	if err != nil {
		t.Fatalf("Failed to parse PeaShooter.reanim: %v", err)
	}

	// Load images
	images, err := LoadReanimImages(reanimData, "../../assets/reanim/")
	if err != nil {
		t.Fatalf("Failed to load images: %v", err)
	}

	// Verify images were loaded
	if len(images) == 0 {
		t.Errorf("Expected at least one image to be loaded, got 0")
	}

	// Verify each image is valid
	for ref, img := range images {
		if img == nil {
			t.Errorf("Image for reference '%s' is nil", ref)
		} else {
			bounds := img.Bounds()
			if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
				t.Errorf("Image for reference '%s' has invalid dimensions: %dx%d", ref, bounds.Dx(), bounds.Dy())
			}
		}
	}
}

// TestLoadReanimImages_Deduplication tests that duplicate references only load once
func TestLoadReanimImages_Deduplication(t *testing.T) {
	// Create a test reanim with duplicate image references
	reanimData := &reanim.ReanimXML{
		FPS: 12,
		Tracks: []reanim.Track{
			{
				Name: "head1",
				Frames: []reanim.Frame{
					{ImagePath: "IMAGE_REANIM_PEASHOOTER_HEAD"},
					{ImagePath: "IMAGE_REANIM_PEASHOOTER_HEAD"}, // Duplicate
				},
			},
			{
				Name: "head2",
				Frames: []reanim.Frame{
					{ImagePath: "IMAGE_REANIM_PEASHOOTER_HEAD"}, // Another duplicate
				},
			},
		},
	}

	images, err := LoadReanimImages(reanimData, "../../assets/reanim/")
	if err != nil {
		t.Fatalf("Failed to load images: %v", err)
	}

	// Should only have one entry for the duplicated reference
	if len(images) != 1 {
		t.Errorf("Expected 1 unique image (deduplication), got %d", len(images))
	}

	// Verify the image is valid
	img := images["IMAGE_REANIM_PEASHOOTER_HEAD"]
	if img == nil {
		t.Errorf("Expected image for 'IMAGE_REANIM_PEASHOOTER_HEAD' to be loaded")
	}
}

// TestLoadReanimImages_Errors tests error handling scenarios
func TestLoadReanimImages_Errors(t *testing.T) {
	tests := []struct {
		name        string
		reanimData  *reanim.ReanimXML
		imagesPath  string
		expectError bool
	}{
		{
			name:        "Nil reanim data",
			reanimData:  nil,
			imagesPath:  "../../assets/reanim/",
			expectError: true,
		},
		{
			name: "Invalid image path",
			reanimData: &reanim.ReanimXML{
				FPS: 12,
				Tracks: []reanim.Track{
					{
						Name: "head",
						Frames: []reanim.Frame{
							{ImagePath: "IMAGE_REANIM_NONEXISTENT"},
						},
					},
				},
			},
			imagesPath:  "../../assets/reanim/",
			expectError: true,
		},
		{
			name: "Invalid images directory",
			reanimData: &reanim.ReanimXML{
				FPS: 12,
				Tracks: []reanim.Track{
					{
						Name: "head",
						Frames: []reanim.Frame{
							{ImagePath: "IMAGE_REANIM_PEASHOOTER_HEAD"},
						},
					},
				},
			},
			imagesPath:  "/nonexistent/path/",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images, err := LoadReanimImages(tt.reanimData, tt.imagesPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				if images != nil {
					t.Errorf("Expected nil images on error, got %v", images)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

// TestLoadReanimImages_EmptyReferences tests that empty references are skipped
func TestLoadReanimImages_EmptyReferences(t *testing.T) {
	reanimData := &reanim.ReanimXML{
		FPS: 12,
		Tracks: []reanim.Track{
			{
				Name: "head",
				Frames: []reanim.Frame{
					{ImagePath: "IMAGE_REANIM_PEASHOOTER_HEAD"},
					{ImagePath: ""}, // Empty reference should be skipped
					{ImagePath: "IMAGE_REANIM_PEASHOOTER_BACKLEAF"},
				},
			},
		},
	}

	images, err := LoadReanimImages(reanimData, "../../assets/reanim/")
	if err != nil {
		t.Fatalf("Failed to load images: %v", err)
	}

	// Should have 2 images (empty reference skipped)
	if len(images) != 2 {
		t.Errorf("Expected 2 images (empty reference skipped), got %d", len(images))
	}

	// Verify empty reference is not in the map
	if _, exists := images[""]; exists {
		t.Errorf("Empty reference should not be in the images map")
	}
}
