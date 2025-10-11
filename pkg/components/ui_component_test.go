package components

import "testing"

// TestUIState tests that UIState constants are defined correctly.
func TestUIState(t *testing.T) {
	tests := []struct {
		name  string
		state UIState
		value int
	}{
		{"UINormal should be 0", UINormal, 0},
		{"UIHovered should be 1", UIHovered, 1},
		{"UIClicked should be 2", UIClicked, 2},
		{"UIDisabled should be 3", UIDisabled, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.state) != tt.value {
				t.Errorf("Expected %s to be %d, got %d", tt.name, tt.value, int(tt.state))
			}
		})
	}
}

// TestUIComponent tests the UIComponent struct.
func TestUIComponent(t *testing.T) {
	// Test creating a UIComponent with Normal state
	component := UIComponent{State: UINormal}
	if component.State != UINormal {
		t.Errorf("Expected state to be UINormal, got %v", component.State)
	}

	// Test state transitions
	component.State = UIHovered
	if component.State != UIHovered {
		t.Errorf("Expected state to be UIHovered after transition, got %v", component.State)
	}

	component.State = UIClicked
	if component.State != UIClicked {
		t.Errorf("Expected state to be UIClicked after transition, got %v", component.State)
	}

	component.State = UIDisabled
	if component.State != UIDisabled {
		t.Errorf("Expected state to be UIDisabled after transition, got %v", component.State)
	}

	// Test back to normal
	component.State = UINormal
	if component.State != UINormal {
		t.Errorf("Expected state to be UINormal after reset, got %v", component.State)
	}
}

// TestButton tests the Button struct.
func TestButton(t *testing.T) {
	// Track if callback was invoked
	callbackInvoked := false
	callback := func() {
		callbackInvoked = true
	}

	// Create a button
	button := Button{
		X:           100,
		Y:           200,
		Width:       150,
		Height:      50,
		NormalImage: nil, // Would be an actual image in real usage
		HoverImage:  nil,
		State:       UINormal,
		OnClick:     callback,
	}

	// Test button properties
	if button.X != 100 {
		t.Errorf("Expected X to be 100, got %v", button.X)
	}
	if button.Y != 200 {
		t.Errorf("Expected Y to be 200, got %v", button.Y)
	}
	if button.Width != 150 {
		t.Errorf("Expected Width to be 150, got %v", button.Width)
	}
	if button.Height != 50 {
		t.Errorf("Expected Height to be 50, got %v", button.Height)
	}
	if button.State != UINormal {
		t.Errorf("Expected State to be UINormal, got %v", button.State)
	}

	// Test callback
	if callbackInvoked {
		t.Error("Callback should not be invoked yet")
	}

	// Invoke callback
	button.OnClick()
	if !callbackInvoked {
		t.Error("Callback should be invoked")
	}

	// Test state change
	button.State = UIHovered
	if button.State != UIHovered {
		t.Errorf("Expected State to be UIHovered, got %v", button.State)
	}
}
