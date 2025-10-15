// Package particle provides data structures and parsing functionality for
// Plants vs. Zombies particle effect configurations.
package particle

import (
	"math"
	"math/rand"
	"strconv"
	"strings"
)

// Keyframe represents a single keyframe in an animation curve.
// Used for animating particle properties over time (e.g., alpha, scale, spin).
type Keyframe struct {
	Time  float64 // Normalized time (0-1) or absolute time
	Value float64 // Value at this keyframe
}

// ParseValue parses a value string from particle configuration.
// Supports multiple formats:
//   - Fixed value: "1500" → min=1500, max=1500, keyframes=nil
//   - Range: "[0.7 0.9]" → min=0.7, max=0.9, keyframes=nil
//   - Keyframes: "0,2 1,2 4,21" → keyframes=[{time:0, value:2}, {time:1, value:2}, {time:4, value:21}]
//   - Interpolation: ".4 Linear 10,9.999999" → keyframes with interpolation="Linear"
//
// Returns:
//   - min, max: Range values (if not keyframes)
//   - keyframes: Parsed keyframe array (if keyframes format)
//   - interpolation: Interpolation mode ("Linear", "EaseIn", etc.)
func ParseValue(s string) (min, max float64, keyframes []Keyframe, interpolation string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, nil, ""
	}

	// Check for range format: "[min max]"
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		rangeStr := strings.TrimPrefix(s, "[")
		rangeStr = strings.TrimSuffix(rangeStr, "]")
		parts := strings.Fields(rangeStr)
		if len(parts) == 2 {
			min, _ = strconv.ParseFloat(parts[0], 64)
			max, _ = strconv.ParseFloat(parts[1], 64)
			return min, max, nil, ""
		}
		// Fallback if parsing fails
		return 0, 0, nil, ""
	}

	// Check for interpolation keywords
	interpolationKeywords := []string{"Linear", "EaseIn", "EaseOut", "FastInOutWeak"}
	for _, keyword := range interpolationKeywords {
		if strings.Contains(s, keyword) {
			interpolation = keyword
			// Remove keyword from string for further parsing
			s = strings.ReplaceAll(s, keyword, "")
			s = strings.TrimSpace(s)
			break
		}
	}

	// Check for keyframes format: contains comma or has interpolation keyword
	if strings.Contains(s, ",") || interpolation != "" {
		parts := strings.Fields(s)
		keyframes = make([]Keyframe, 0, len(parts))
		currentTime := 0.0

		for _, part := range parts {
			if strings.Contains(part, ",") {
				// Normal keyframe format: "time,value"
				pair := strings.Split(part, ",")
				if len(pair) == 2 {
					time, err1 := strconv.ParseFloat(pair[0], 64)
					value, err2 := strconv.ParseFloat(pair[1], 64)
					if err1 == nil && err2 == nil {
						keyframes = append(keyframes, Keyframe{Time: time, Value: value})
						currentTime = time
					}
				}
			} else {
				// Single value: treat as value at time=0 (or current sequence)
				value, err := strconv.ParseFloat(part, 64)
				if err == nil {
					// If this is the first keyframe, use time=0
					if len(keyframes) == 0 {
						keyframes = append(keyframes, Keyframe{Time: 0, Value: value})
					} else {
						// Otherwise, use incremental time (edge case)
						keyframes = append(keyframes, Keyframe{Time: currentTime + 1, Value: value})
					}
				}
			}
		}
		if len(keyframes) > 0 {
			return 0, 0, keyframes, interpolation
		}
	}

	// Fixed value format
	value, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return value, value, nil, ""
	}

	// Fallback: return zero
	return 0, 0, nil, ""
}

// EvaluateKeyframes calculates the interpolated value at time t (0-1)
// using the provided keyframes and interpolation mode.
//
// Parameters:
//   - keyframes: Array of keyframes (must be sorted by Time)
//   - t: Normalized time (0-1)
//   - interpolation: Interpolation mode ("Linear", "EaseIn", etc.)
//
// Returns the interpolated value at time t.
func EvaluateKeyframes(keyframes []Keyframe, t float64, interpolation string) float64 {
	if len(keyframes) == 0 {
		return 0
	}
	if len(keyframes) == 1 {
		return keyframes[0].Value
	}

	// Clamp t to [0, 1]
	t = math.Max(0, math.Min(1, t))

	// Find the keyframe interval containing t
	for i := 0; i < len(keyframes)-1; i++ {
		k0 := keyframes[i]
		k1 := keyframes[i+1]

		if t >= k0.Time && t <= k1.Time {
			// Calculate ratio within this interval
			duration := k1.Time - k0.Time
			if duration <= 0 {
				return k0.Value
			}
			ratio := (t - k0.Time) / duration

			// Apply interpolation
			switch interpolation {
			case "Linear", "": // Default is linear
				return k0.Value + ratio*(k1.Value-k0.Value)
			case "EaseIn":
				ratio = ratio * ratio // Quadratic ease-in
				return k0.Value + ratio*(k1.Value-k0.Value)
			case "EaseOut":
				ratio = 1 - (1-ratio)*(1-ratio) // Quadratic ease-out
				return k0.Value + ratio*(k1.Value-k0.Value)
			case "FastInOutWeak":
				// Simplified cubic interpolation
				ratio = ratio * ratio * (3 - 2*ratio)
				return k0.Value + ratio*(k1.Value-k0.Value)
			default:
				// Unknown interpolation, use linear
				return k0.Value + ratio*(k1.Value-k0.Value)
			}
		}
	}

	// If t is beyond the last keyframe, return the last value
	return keyframes[len(keyframes)-1].Value
}

// RandomInRange returns a random float64 in the range [min, max].
func RandomInRange(min, max float64) float64 {
	if min >= max {
		return min
	}
	return min + rand.Float64()*(max-min)
}
