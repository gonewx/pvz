package utils

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// LoadReanimImages loads all sprite part images referenced in a Reanim animation.
// It collects all unique image references from the animation tracks, maps them to
// actual file names, and loads the corresponding PNG files.
//
// Parameters:
//   - reanim: The parsed Reanim animation data
//   - imagesPath: Path to the directory containing part images, e.g., "assets/reanim/"
//
// Returns:
//   - map[string]*ebiten.Image: Map of image reference names to loaded images
//   - error: Loading error, or nil if successful
//
// Image Reference Mapping:
//   - XML reference: "IMAGE_REANIM_PEASHOOTER_HEAD"
//   - File name: "peashooter_head.png" (lowercase, without "IMAGE_REANIM_" prefix, .png extension)
//
// Example:
//
//	reanim, _ := reanim.ParseReanimFile("assets/effect/reanim/PeaShooter.reanim")
//	images, err := LoadReanimImages(reanim, "assets/reanim/")
//	if err != nil {
//	    log.Fatalf("Failed to load images: %v", err)
//	}
//	headImage := images["IMAGE_REANIM_PEASHOOTER_HEAD"]
func LoadReanimImages(reanimData *reanim.ReanimXML, imagesPath string) (map[string]*ebiten.Image, error) {
	if reanimData == nil {
		return nil, fmt.Errorf("reanim data is nil")
	}

	// Collect all unique image references
	imageRefs := collectImageReferences(reanimData)

	// Load images with deduplication
	images := make(map[string]*ebiten.Image)
	for ref := range imageRefs {
		// Skip empty references
		if ref == "" {
			continue
		}

		// Map reference name to file name
		fileName := mapImageNameToFile(ref)

		// Build full path
		fullPath := filepath.Join(imagesPath, fileName)

		// Load image
		img, _, err := ebitenutil.NewImageFromFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load image '%s' (ref: %s): %w", fullPath, ref, err)
		}

		images[ref] = img
	}

	return images, nil
}

// collectImageReferences collects all unique image references from the reanim data.
// It traverses all tracks and frames to find ImagePath values.
func collectImageReferences(reanimData *reanim.ReanimXML) map[string]bool {
	refs := make(map[string]bool)

	for _, track := range reanimData.Tracks {
		for _, frame := range track.Frames {
			if frame.ImagePath != "" {
				refs[frame.ImagePath] = true
			}
		}
	}

	return refs
}

// mapImageNameToFile converts an image reference name to a file name.
// Mapping rules:
//   - Remove "IMAGE_REANIM_" prefix
//   - Convert to lowercase
//   - Add ".png" extension
//
// Examples:
//   - "IMAGE_REANIM_PEASHOOTER_HEAD" -> "peashooter_head.png"
//   - "IMAGE_REANIM_SUNFLOWER_BODY" -> "sunflower_body.png"
func mapImageNameToFile(ref string) string {
	// Remove "IMAGE_REANIM_" prefix if present
	name := strings.TrimPrefix(ref, "IMAGE_REANIM_")

	// Convert to lowercase
	name = strings.ToLower(name)

	// Add .png extension
	return name + ".png"
}
