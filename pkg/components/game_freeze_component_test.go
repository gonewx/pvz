package components

import "testing"

func TestGameFreezeComponent(t *testing.T) {
	tests := []struct {
		name     string
		isFrozen bool
	}{
		{
			name:     "Frozen",
			isFrozen: true,
		},
		{
			name:     "Not Frozen",
			isFrozen: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := &GameFreezeComponent{
				IsFrozen: tt.isFrozen,
			}

			if comp.IsFrozen != tt.isFrozen {
				t.Errorf("Expected IsFrozen=%v, got=%v", tt.isFrozen, comp.IsFrozen)
			}
		})
	}
}
