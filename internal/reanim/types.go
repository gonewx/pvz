// Package reanim provides data structures and parsers for PVZ Reanim animation files.
// Reanim files define skeletal animations used in Plants vs. Zombies,
// containing frame-by-frame transformations for sprite parts.
package reanim

// ReanimXML is the root structure of a Reanim animation file.
// It contains the frame rate (FPS) and a list of animation tracks.
type ReanimXML struct {
	// FPS is the frame rate of the animation, typically 12 for PVZ animations
	FPS int `xml:"fps"`

	// Tracks is the list of animation tracks, which can be:
	// - Animation definition tracks (names starting with "anim_")
	// - Part animation tracks (e.g., "head", "body", "backleaf")
	Tracks []Track `xml:"track"`
}

// Track represents a single animation track, which can be either an animation
// definition track (e.g., "anim_idle", "anim_shooting") that controls overall
// animation visibility, or a part animation track (e.g., "head", "body") that
// defines transformations for individual sprite parts.
type Track struct {
	// Name is the track name, e.g., "anim_idle", "head", "body"
	Name string `xml:"name"`

	// Frames is the sequence of animation frames in this track
	Frames []Frame `xml:"t"`
}

// Frame represents a single animation frame. All fields are optional and use
// pointer types to support null values. When a field is null, its value is
// inherited from the previous frame (cumulative inheritance).
type Frame struct {
	// FrameNum controls frame visibility:
	// - nil: inherit from previous frame
	// - -1: hide this part in current frame
	// - 0 or positive: show this part
	FrameNum *int `xml:"f,omitempty"`

	// X is the X position offset in pixels
	X *float64 `xml:"x,omitempty"`

	// Y is the Y position offset in pixels
	Y *float64 `xml:"y,omitempty"`

	// ScaleX is the X-axis scale factor (1.0 = normal size)
	ScaleX *float64 `xml:"sx,omitempty"`

	// ScaleY is the Y-axis scale factor (1.0 = normal size)
	ScaleY *float64 `xml:"sy,omitempty"`

	// SkewX is the X-axis skew angle in degrees (NOT radians).
	// FlashReanimExport script exports angles in degrees after converting from radians.
	// When rendering, convert to radians: kxRad = kx * π / 180
	SkewX *float64 `xml:"kx,omitempty"`

	// SkewY is the Y-axis skew angle in degrees (NOT radians).
	// Note: FlashReanimExport applies a negation when exporting (degreesKY = -RadToDeg(skewY)).
	// When rendering, apply: c = -sin(ky * π / 180) * scaleY to compensate.
	SkewY *float64 `xml:"ky,omitempty"`

	// ImagePath is the sprite part image reference, e.g., "IMAGE_REANIM_PEASHOOTER_HEAD"
	ImagePath string `xml:"i,omitempty"`
}
