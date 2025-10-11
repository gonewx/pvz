package components

import "github.com/hajimehoshi/ebiten/v2"

// UIState represents the current state of a UI element (e.g., button).
type UIState int

const (
	// UINormal indicates the UI element is in its default state.
	UINormal UIState = iota
	// UIHovered indicates the mouse cursor is hovering over the UI element.
	UIHovered
	// UIClicked indicates the UI element is being clicked.
	UIClicked
	// UIDisabled indicates the UI element is disabled and cannot be interacted with.
	UIDisabled
)

// UIComponent is a component that marks an entity as a UI element
// and tracks its interaction state.
// This component is intended for use in the ECS architecture to identify
// UI entities and manage their visual and interaction states.
type UIComponent struct {
	// State is the current interaction state of the UI element.
	State UIState
}

// Button represents a clickable UI button with position, size, images, and click behavior.
// This is a simplified button implementation for use in scenes before the full ECS UI system is built.
type Button struct {
	// X is the X coordinate of the button's top-left corner in screen space.
	X float64
	// Y is the Y coordinate of the button's top-left corner in screen space.
	Y float64
	// Width is the width of the button in pixels.
	Width float64
	// Height is the height of the button in pixels.
	Height float64
	// NormalImage is the image displayed when the button is in normal state.
	NormalImage *ebiten.Image
	// HoverImage is the image displayed when the mouse hovers over the button.
	// If nil, visual feedback will be achieved through other means (e.g., color tint, scaling).
	HoverImage *ebiten.Image
	// State is the current interaction state of the button.
	State UIState
	// OnClick is the callback function invoked when the button is clicked.
	OnClick func()
}
