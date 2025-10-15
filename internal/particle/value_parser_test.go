package particle

import (
	"math"
	"testing"
)

// TestParseValue_FixedValue tests parsing of fixed value format
func TestParseValue_FixedValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMin  float64
		wantMax  float64
		wantKeys bool
	}{
		{"Integer", "1500", 1500, 1500, false},
		{"Float", "3.14", 3.14, 3.14, false},
		{"Negative", "-10.5", -10.5, -10.5, false},
		{"Zero", "0", 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			min, max, keyframes, interp := ParseValue(tt.input)
			if min != tt.wantMin {
				t.Errorf("ParseValue(%q) min = %v, want %v", tt.input, min, tt.wantMin)
			}
			if max != tt.wantMax {
				t.Errorf("ParseValue(%q) max = %v, want %v", tt.input, max, tt.wantMax)
			}
			if (keyframes != nil) != tt.wantKeys {
				t.Errorf("ParseValue(%q) keyframes = %v, want hasKeyframes=%v", tt.input, keyframes, tt.wantKeys)
			}
			if interp != "" {
				t.Errorf("ParseValue(%q) interpolation = %q, want empty", tt.input, interp)
			}
		})
	}
}

// TestParseValue_Range tests parsing of range format
func TestParseValue_Range(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMin  float64
		wantMax  float64
		wantKeys bool
	}{
		{"Float range", "[0.7 0.9]", 0.7, 0.9, false},
		{"Integer range", "[10 20]", 10, 20, false},
		{"Negative range", "[-5 -2]", -5, -2, false},
		{"Mixed range", "[-1.5 2.5]", -1.5, 2.5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			min, max, keyframes, _ := ParseValue(tt.input)
			if min != tt.wantMin {
				t.Errorf("ParseValue(%q) min = %v, want %v", tt.input, min, tt.wantMin)
			}
			if max != tt.wantMax {
				t.Errorf("ParseValue(%q) max = %v, want %v", tt.input, max, tt.wantMax)
			}
			if (keyframes != nil) != tt.wantKeys {
				t.Errorf("ParseValue(%q) keyframes = %v, want hasKeyframes=%v", tt.input, keyframes, tt.wantKeys)
			}
		})
	}
}

// TestParseValue_Keyframes tests parsing of keyframe format
func TestParseValue_Keyframes(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
		wantFirst Keyframe
		wantLast  Keyframe
	}{
		{
			name:      "Simple keyframes",
			input:     "0,2 1,2 4,21",
			wantCount: 3,
			wantFirst: Keyframe{Time: 0, Value: 2},
			wantLast:  Keyframe{Time: 4, Value: 21},
		},
		{
			name:      "Two keyframes",
			input:     "0,1 1,0",
			wantCount: 2,
			wantFirst: Keyframe{Time: 0, Value: 1},
			wantLast:  Keyframe{Time: 1, Value: 0},
		},
		{
			name:      "Float keyframes",
			input:     "0.5,10.5 0.75,20.25",
			wantCount: 2,
			wantFirst: Keyframe{Time: 0.5, Value: 10.5},
			wantLast:  Keyframe{Time: 0.75, Value: 20.25},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, keyframes, _ := ParseValue(tt.input)
			if len(keyframes) != tt.wantCount {
				t.Errorf("ParseValue(%q) keyframe count = %d, want %d", tt.input, len(keyframes), tt.wantCount)
			}
			if len(keyframes) > 0 {
				if keyframes[0] != tt.wantFirst {
					t.Errorf("ParseValue(%q) first keyframe = %v, want %v", tt.input, keyframes[0], tt.wantFirst)
				}
				if keyframes[len(keyframes)-1] != tt.wantLast {
					t.Errorf("ParseValue(%q) last keyframe = %v, want %v", tt.input, keyframes[len(keyframes)-1], tt.wantLast)
				}
			}
		})
	}
}

// TestParseValue_Interpolation tests parsing of interpolation keywords
func TestParseValue_Interpolation(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantInterp    string
		wantKeysCount int
	}{
		{
			name:          "Linear interpolation",
			input:         ".4 Linear 10,9.999999",
			wantInterp:    "Linear",
			wantKeysCount: 2,
		},
		{
			name:          "EaseIn interpolation",
			input:         "0,0 EaseIn 1,100",
			wantInterp:    "EaseIn",
			wantKeysCount: 2,
		},
		{
			name:          "FastInOutWeak interpolation",
			input:         "0,1 FastInOutWeak 1,0",
			wantInterp:    "FastInOutWeak",
			wantKeysCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, keyframes, interp := ParseValue(tt.input)
			if interp != tt.wantInterp {
				t.Errorf("ParseValue(%q) interpolation = %q, want %q", tt.input, interp, tt.wantInterp)
			}
			if len(keyframes) != tt.wantKeysCount {
				t.Errorf("ParseValue(%q) keyframe count = %d, want %d", tt.input, len(keyframes), tt.wantKeysCount)
			}
		})
	}
}

// TestParseValue_EdgeCases tests edge cases and invalid input
func TestParseValue_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantMin float64
		wantMax float64
	}{
		{"Empty string", "", 0, 0},
		{"Whitespace only", "   ", 0, 0},
		{"Invalid format", "abc", 0, 0},
		{"Incomplete range", "[10", 0, 0},
		{"Malformed keyframes", "0,", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			min, max, _, _ := ParseValue(tt.input)
			if min != tt.wantMin || max != tt.wantMax {
				t.Errorf("ParseValue(%q) = (%v, %v), want (%v, %v)", tt.input, min, max, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestEvaluateKeyframes_Linear tests linear interpolation
func TestEvaluateKeyframes_Linear(t *testing.T) {
	keyframes := []Keyframe{
		{Time: 0, Value: 0},
		{Time: 1, Value: 100},
	}

	tests := []struct {
		name string
		t    float64
		want float64
	}{
		{"Start", 0.0, 0},
		{"Quarter", 0.25, 25},
		{"Half", 0.5, 50},
		{"ThreeQuarter", 0.75, 75},
		{"End", 1.0, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateKeyframes(keyframes, tt.t, "Linear")
			if math.Abs(got-tt.want) > 0.0001 {
				t.Errorf("EvaluateKeyframes(t=%v) = %v, want %v", tt.t, got, tt.want)
			}
		})
	}
}

// TestEvaluateKeyframes_MultipleSegments tests interpolation across multiple keyframes
func TestEvaluateKeyframes_MultipleSegments(t *testing.T) {
	keyframes := []Keyframe{
		{Time: 0, Value: 0},
		{Time: 0.5, Value: 50},
		{Time: 1, Value: 0},
	}

	tests := []struct {
		t    float64
		want float64
	}{
		{0.0, 0},
		{0.25, 25},
		{0.5, 50},
		{0.75, 25},
		{1.0, 0},
	}

	for _, tt := range tests {
		got := EvaluateKeyframes(keyframes, tt.t, "Linear")
		if math.Abs(got-tt.want) > 0.0001 {
			t.Errorf("EvaluateKeyframes(t=%v) = %v, want %v", tt.t, got, tt.want)
		}
	}
}

// TestEvaluateKeyframes_EdgeCases tests edge cases
func TestEvaluateKeyframes_EdgeCases(t *testing.T) {
	t.Run("Empty keyframes", func(t *testing.T) {
		got := EvaluateKeyframes([]Keyframe{}, 0.5, "Linear")
		if got != 0 {
			t.Errorf("EvaluateKeyframes(empty) = %v, want 0", got)
		}
	})

	t.Run("Single keyframe", func(t *testing.T) {
		keyframes := []Keyframe{{Time: 0, Value: 42}}
		got := EvaluateKeyframes(keyframes, 0.5, "Linear")
		if got != 42 {
			t.Errorf("EvaluateKeyframes(single) = %v, want 42", got)
		}
	})

	t.Run("Out of bounds (below)", func(t *testing.T) {
		keyframes := []Keyframe{{Time: 0, Value: 0}, {Time: 1, Value: 100}}
		got := EvaluateKeyframes(keyframes, -0.5, "Linear")
		if got != 0 {
			t.Errorf("EvaluateKeyframes(t=-0.5) = %v, want 0 (clamped)", got)
		}
	})

	t.Run("Out of bounds (above)", func(t *testing.T) {
		keyframes := []Keyframe{{Time: 0, Value: 0}, {Time: 1, Value: 100}}
		got := EvaluateKeyframes(keyframes, 1.5, "Linear")
		if got != 100 {
			t.Errorf("EvaluateKeyframes(t=1.5) = %v, want 100 (clamped)", got)
		}
	})
}

// TestEvaluateKeyframes_Interpolations tests different interpolation modes
func TestEvaluateKeyframes_Interpolations(t *testing.T) {
	keyframes := []Keyframe{
		{Time: 0, Value: 0},
		{Time: 1, Value: 100},
	}

	t.Run("EaseIn", func(t *testing.T) {
		// At t=0.5, EaseIn (quadratic) should give 0 + 0.25 * 100 = 25
		got := EvaluateKeyframes(keyframes, 0.5, "EaseIn")
		if math.Abs(got-25) > 0.0001 {
			t.Errorf("EvaluateKeyframes(EaseIn, t=0.5) = %v, want 25", got)
		}
	})

	t.Run("EaseOut", func(t *testing.T) {
		// At t=0.5, EaseOut should give 0 + 0.75 * 100 = 75
		got := EvaluateKeyframes(keyframes, 0.5, "EaseOut")
		if math.Abs(got-75) > 0.0001 {
			t.Errorf("EvaluateKeyframes(EaseOut, t=0.5) = %v, want 75", got)
		}
	})

	t.Run("Unknown interpolation defaults to Linear", func(t *testing.T) {
		got := EvaluateKeyframes(keyframes, 0.5, "UnknownMode")
		if math.Abs(got-50) > 0.0001 {
			t.Errorf("EvaluateKeyframes(Unknown, t=0.5) = %v, want 50 (linear fallback)", got)
		}
	})
}

// TestRandomInRange tests range randomization
func TestRandomInRange(t *testing.T) {
	t.Run("Basic range", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			got := RandomInRange(10, 20)
			if got < 10 || got > 20 {
				t.Errorf("RandomInRange(10, 20) = %v, out of range", got)
			}
		}
	})

	t.Run("Equal min and max", func(t *testing.T) {
		got := RandomInRange(5, 5)
		if got != 5 {
			t.Errorf("RandomInRange(5, 5) = %v, want 5", got)
		}
	})

	t.Run("Inverted range (min > max)", func(t *testing.T) {
		got := RandomInRange(20, 10)
		if got != 20 {
			t.Errorf("RandomInRange(20, 10) = %v, want 20 (should return min)", got)
		}
	})

	t.Run("Negative range", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			got := RandomInRange(-10, -5)
			if got < -10 || got > -5 {
				t.Errorf("RandomInRange(-10, -5) = %v, out of range", got)
			}
		}
	})
}
