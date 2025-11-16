package utils

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// CompositeImages creates a new image by compositing a transparent overlay on top of a base image.
// This is commonly used in PVZ for background layers, where:
//   - baseImage: A JPG image with black or solid background
//   - overlayImage: A PNG image with transparency to layer on top
//
// Returns:
//   - A new ebiten.Image containing the composited result
//   - The overlay is drawn on top of the base using alpha blending
//
// Usage Example (SelectorScreen backgrounds):
//
//	base := rm.LoadImageByID("IMAGE_REANIM_SELECTORSCREEN_BG_CENTER")
//	overlay := rm.LoadImageByID("IMAGE_REANIM_SELECTORSCREEN_BG_CENTER_OVERLAY")
//	composited := utils.CompositeImages(base, overlay)
//
// This function is reusable across the project for any image layering needs.
func CompositeImages(baseImage, overlayImage *ebiten.Image) *ebiten.Image {
	if baseImage == nil {
		return overlayImage
	}
	if overlayImage == nil {
		return baseImage
	}

	// Create a new image with the same dimensions as the base
	bounds := baseImage.Bounds()
	composited := ebiten.NewImage(bounds.Dx(), bounds.Dy())

	// Draw base image
	composited.DrawImage(baseImage, &ebiten.DrawImageOptions{})

	// Draw overlay image on top (alpha blending automatically applied)
	composited.DrawImage(overlayImage, &ebiten.DrawImageOptions{})

	return composited
}

// ApplyAlphaMask applies an alpha mask to a base image using pixel-level compositing.
// This is used in PVZ to remove black backgrounds from JPG images using PNG masks.
//
// Parameters:
//   - baseImage: The source image (e.g., JPG with black background or PNG with RGB data)
//   - maskImage: The alpha mask image (8-bit or RGBA PNG)
//   - White pixels in mask = fully opaque
//   - Black pixels in mask = fully transparent
//   - Gray pixels = partial transparency
//
// Returns:
//   - A new image with the mask applied (RGBA format)
//   - Returns nil if baseImage is nil
//   - Returns baseImage unchanged if maskImage is nil (graceful degradation)
//
// Algorithm:
//   - Reads pixels from both images
//   - Combines RGB from baseImage with Alpha from maskImage's luminance
//   - Creates a new RGBA image with proper transparency
//
// Usage Examples:
//
//	// Example 1: Help panel background (Story 12.3)
//	bgJPG := rm.LoadImage("assets/images/ZombieNote.jpg")
//	bgMask := rm.LoadImage("assets/images/ZombieNote_.png")
//	maskedBG := utils.ApplyAlphaMask(bgJPG, bgMask)  // Transparent edges
//
//	// Example 2: Help panel text overlay
//	textPNG := rm.LoadImage("assets/images/ZombieNoteHelp.png")
//	textMask := rm.LoadImage("assets/images/ZombieNoteHelpBlack.png")
//	maskedText := utils.ApplyAlphaMask(textPNG, textMask)  // Transparent text
//
// Performance:
//   - Pixel-level operation, should be called during initialization, not per-frame
//   - Cache the result in a component or module for repeated use
//
// Story 12.3: 对话框系统基础 - 帮助面板蒙板叠加
func ApplyAlphaMask(baseImage, maskImage *ebiten.Image) *ebiten.Image {
	if baseImage == nil {
		return nil
	}
	if maskImage == nil {
		// Graceful degradation: return original image if mask is missing
		return baseImage
	}

	// Get image dimensions
	baseBounds := baseImage.Bounds()
	maskBounds := maskImage.Bounds()

	width := baseBounds.Dx()
	height := baseBounds.Dy()

	// Check dimension mismatch (graceful degradation)
	if width != maskBounds.Dx() || height != maskBounds.Dy() {
		// Return original image if dimensions don't match
		return baseImage
	}

	// Create output image (RGBA format)
	result := ebiten.NewImage(width, height)

	// Read pixel data from both images
	basePixels := make([]byte, width*height*4)
	baseImage.ReadPixels(basePixels)

	maskPixels := make([]byte, width*height*4)
	maskImage.ReadPixels(maskPixels)

	// Composite RGBA data
	outputPixels := make([]byte, width*height*4)
	for i := 0; i < width*height; i++ {
		// RGB from base image (color data)
		outputPixels[i*4+0] = basePixels[i*4+0] // R
		outputPixels[i*4+1] = basePixels[i*4+1] // G
		outputPixels[i*4+2] = basePixels[i*4+2] // B

		// Alpha from mask's luminance (brightness value)
		// Mask is typically grayscale, so R=G=B, use R channel
		// Higher luminance = more opaque
		outputPixels[i*4+3] = maskPixels[i*4+0] // A = mask's R channel
	}

	// Write composed pixels to result image
	result.WritePixels(outputPixels)

	return result
}

// CompositeImagesAt creates a composited image with overlay at a specific position.
// Useful when overlay needs to be offset from the base image origin.
//
// Parameters:
//   - baseImage: The background/base layer
//   - overlayImage: The transparent overlay layer
//   - offsetX, offsetY: Position offset for the overlay relative to base (0,0)
func CompositeImagesAt(baseImage, overlayImage *ebiten.Image, offsetX, offsetY float64) *ebiten.Image {
	if baseImage == nil {
		return overlayImage
	}
	if overlayImage == nil {
		return baseImage
	}

	// Create a new image with the same dimensions as the base
	bounds := baseImage.Bounds()
	composited := ebiten.NewImage(bounds.Dx(), bounds.Dy())

	// Draw base image
	composited.DrawImage(baseImage, &ebiten.DrawImageOptions{})

	// Draw overlay image at offset position
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(offsetX, offsetY)
	composited.DrawImage(overlayImage, op)

	return composited
}

// CropImage creates a sub-image from the source image.
// This is a convenience wrapper around SubImage.
//
// Parameters:
//   - src: The source image to crop
//   - rect: The rectangle region to extract
//
// Returns:
//   - A new ebiten.Image containing only the cropped region
func CropImage(src *ebiten.Image, rect image.Rectangle) *ebiten.Image {
	if src == nil {
		return nil
	}

	// Ensure rect is within bounds
	bounds := src.Bounds()
	if rect.Min.X < bounds.Min.X {
		rect.Min.X = bounds.Min.X
	}
	if rect.Min.Y < bounds.Min.Y {
		rect.Min.Y = bounds.Min.Y
	}
	if rect.Max.X > bounds.Max.X {
		rect.Max.X = bounds.Max.X
	}
	if rect.Max.Y > bounds.Max.Y {
		rect.Max.Y = bounds.Max.Y
	}

	return src.SubImage(rect).(*ebiten.Image)
}
