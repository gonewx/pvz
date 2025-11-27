package config

import (
	"testing"
)

// TestCalculateBottomButtonPosition tests the position calculation for bottom function buttons.
//
// Story 12.2: 底部功能栏重构
func TestCalculateBottomButtonPosition(t *testing.T) {
	tests := []struct {
		name        string
		buttonIndex int
		expectedX   float64
		expectedY   float64
	}{
		{
			name:        "Button 0 (Options)",
			buttonIndex: 0,
			expectedX:   565.0,
			expectedY:   495.0,
		},
		{
			name:        "Button 1 (Help)",
			buttonIndex: 1,
			expectedX:   648.0,
			expectedY:   525.0,
		},
		{
			name:        "Button 2 (Quit)",
			buttonIndex: 2,
			expectedX:   720.0,
			expectedY:   515.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotX, gotY := CalculateBottomButtonPosition(tt.buttonIndex)

			if gotX != tt.expectedX {
				t.Errorf("CalculateBottomButtonPosition(%d) X = %.1f, want %.1f", tt.buttonIndex, gotX, tt.expectedX)
			}

			if gotY != tt.expectedY {
				t.Errorf("CalculateBottomButtonPosition(%d) Y = %.1f, want %.1f", tt.buttonIndex, gotY, tt.expectedY)
			}
		})
	}
}

// TestBottomButtonLayout tests that the button layout is within screen bounds.
//
// Story 12.2: 底部功能栏重构
func TestBottomButtonLayout(t *testing.T) {
	const (
		screenWidth  = 800.0
		screenHeight = 600.0
	)

	// Test all 3 buttons
	for i := 0; i < 3; i++ {
		x, y := CalculateBottomButtonPosition(i)

		// X should be within screen width (with some margin for button width)
		if x < 0 || x >= screenWidth {
			t.Errorf("Button %d X position %.1f is outside screen bounds (0-%.1f)", i, x, screenWidth)
		}

		// Y should be within screen height (with some margin for button height)
		if y < 0 || y >= screenHeight {
			t.Errorf("Button %d Y position %.1f is outside screen bounds (0-%.1f)", i, y, screenHeight)
		}
	}
}

// TestBottomButtonPositionsCount tests that exactly 3 button positions are defined.
//
// Story 12.2: 底部功能栏重构
func TestBottomButtonPositionsCount(t *testing.T) {
	expectedCount := 3 // Options, Help, Quit
	actualCount := len(BottomButtonPositions)

	if actualCount != expectedCount {
		t.Errorf("BottomButtonPositions count = %d, want %d", actualCount, expectedCount)
	}
}

// TestCalculateBottomButtonPositionInvalidIndex tests handling of invalid button indices.
//
// Story 12.2: 底部功能栏重构
func TestCalculateBottomButtonPositionInvalidIndex(t *testing.T) {
	tests := []struct {
		name        string
		buttonIndex int
	}{
		{name: "Negative index", buttonIndex: -1},
		{name: "Too large index", buttonIndex: 3},
		{name: "Way too large index", buttonIndex: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := CalculateBottomButtonPosition(tt.buttonIndex)

			// Invalid index should return (0, 0)
			if x != 0 || y != 0 {
				t.Errorf("CalculateBottomButtonPosition(%d) = (%.1f, %.1f), want (0, 0)", tt.buttonIndex, x, y)
			}
		})
	}
}
