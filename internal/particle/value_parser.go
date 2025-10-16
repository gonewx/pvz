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
//   - Double range: "[0.4 0.6] [0.8 1.2]" → generates keyframes with random start/end values
//   - Keyframes: "0,2 1,2 4,21" → keyframes=[{time:0, value:2}, {time:1, value:2}, {time:4, value:21}]
//   - Interpolation: ".4 Linear 10,9.999999" → keyframes with interpolation="Linear"
//   - PopCap format: ".9,70 0" → initialValue=0.9 at time=0%, finalValue=0 at time=70%
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

	// Story 7.5 修复：检查"范围+关键帧"组合格式 "[-720 720] 0,39.999996"
	// 这种格式用于定义：初始值从范围随机，然后随时间衰减到目标值
	if strings.HasPrefix(s, "[") && strings.Contains(s, "]") {
		closeBracketIdx := strings.Index(s, "]")
		if closeBracketIdx > 0 && closeBracketIdx < len(s)-1 {
			afterBracket := strings.TrimSpace(s[closeBracketIdx+1:])
			// 检查右括号后是否还有内容（关键帧部分）
			if afterBracket != "" {
				// 分离范围部分和关键帧部分
				rangeStr := s[:closeBracketIdx+1]
				keyframeStr := afterBracket

				// 解析范围部分
				rangeStr = strings.TrimPrefix(rangeStr, "[")
				rangeStr = strings.TrimSuffix(rangeStr, "]")
				rangeParts := strings.Fields(rangeStr)

				// 解析关键帧部分
				// 对于"范围+关键帧"格式，关键帧使用特殊格式："value,timePercent"
				// 例如："0,39.999996" 表示在39.999996%时值为0
				var kf []Keyframe
				var interp string

				// 处理关键帧字符串：可能包含多个"value,timePercent"对
				keyframeParts := strings.Fields(keyframeStr)
				for _, part := range keyframeParts {
					if strings.Contains(part, ",") {
						pair := strings.Split(part, ",")
						if len(pair) == 2 {
							val, err1 := strconv.ParseFloat(pair[0], 64)
							timePercent, err2 := strconv.ParseFloat(pair[1], 64)
							if err1 == nil && err2 == nil {
								// "value,timePercent" 格式
								// timePercent > 1 表示百分比（需要除以100）
								time := timePercent / 100.0
								if timePercent <= 1.0 {
									// 已经是归一化值（0-1），不需要除以100
									time = timePercent
								}
								kf = append(kf, Keyframe{Time: time, Value: val})
							}
						}
					}
				}

				if len(rangeParts) == 2 {
					rangeMin, err1 := strconv.ParseFloat(rangeParts[0], 64)
					rangeMax, err2 := strconv.ParseFloat(rangeParts[1], 64)
					if err1 == nil && err2 == nil {
						// 成功解析范围+关键帧格式
						// 返回范围值和关键帧（关键帧将在 spawnParticle 中与初始值结合）
						return rangeMin, rangeMax, kf, interp
					}
				}
			}
		}
	}

	// Story 7.4 修复：检查双范围格式 "[min1 max1] [min2 max2]"
	// 例如: "[.4 .6] [.8 1.2]" 表示初始范围和结束范围
	if strings.Count(s, "[") == 2 && strings.Count(s, "]") == 2 {
		parts := strings.Split(s, "]")
		if len(parts) >= 2 {
			// 解析第一个范围
			range1Str := strings.TrimPrefix(strings.TrimSpace(parts[0]), "[")
			range1Parts := strings.Fields(range1Str)

			// 解析第二个范围
			range2Str := strings.TrimPrefix(strings.TrimSpace(parts[1]), "[")
			range2Str = strings.TrimSuffix(range2Str, "]")
			range2Parts := strings.Fields(range2Str)

			if len(range1Parts) == 2 && len(range2Parts) == 2 {
				startMin, err1 := strconv.ParseFloat(range1Parts[0], 64)
				startMax, err2 := strconv.ParseFloat(range1Parts[1], 64)
				endMin, err3 := strconv.ParseFloat(range2Parts[0], 64)
				endMax, err4 := strconv.ParseFloat(range2Parts[1], 64)

				if err1 == nil && err2 == nil && err3 == nil && err4 == nil {
					// 随机生成初始值和结束值
					startValue := RandomInRange(startMin, startMax)
					endValue := RandomInRange(endMin, endMax)

					// 创建从 0 到 1 的 keyframes
					keyframes = []Keyframe{
						{Time: 0, Value: startValue},
						{Time: 1, Value: endValue},
					}
					return 0, 0, keyframes, "Linear" // 默认线性插值
				}
			}
		}
	}

	// Check for range format: "[min max]" or "[value]"
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		rangeStr := strings.TrimPrefix(s, "[")
		rangeStr = strings.TrimSuffix(rangeStr, "]")
		parts := strings.Fields(rangeStr)
		if len(parts) == 2 {
			// 范围格式: "[min max]"
			min, _ = strconv.ParseFloat(parts[0], 64)
			max, _ = strconv.ParseFloat(parts[1], 64)
			return min, max, nil, ""
		} else if len(parts) == 1 {
			// 单值格式: "[value]" - 作为固定值处理
			val, err := strconv.ParseFloat(parts[0], 64)
			if err == nil {
				return val, val, nil, ""
			}
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
		keyframes = make([]Keyframe, 0, len(parts)+1)

		// Story 7.5 修复：处理混合格式 ".3 .3,39.999996 0,50"
		// 这种格式包含：初始值 + 多个"value,timePercent"对
		var initialValue *float64 // 初始值（如果存在）
		processedIndices := make(map[int]bool) // 标记已处理的索引

		for i, part := range parts {
			if processedIndices[i] {
				continue // 跳过已处理的部分
			}

			if strings.Contains(part, ",") {
				pair := strings.Split(part, ",")
				if len(pair) == 2 {
					val1, err1 := strconv.ParseFloat(pair[0], 64)
					val2, err2 := strconv.ParseFloat(pair[1], 64)
					if err1 == nil && err2 == nil {
						// 检查是否是 PopCap 格式："initialValue,timePercent finalValue"
						// 条件：val2 > 1（表示百分比），且后面还有单独的值（不含逗号）
						if val2 > 1 && i+1 < len(parts) && !strings.Contains(parts[i+1], ",") {
							// 尝试解析后续值
							nextVal, err := strconv.ParseFloat(parts[i+1], 64)
							if err == nil {
								// PopCap 格式: "initialValue,timePercent finalValue"
								initialValuePop := val1
								timePercent := val2
								finalValue := nextVal

								// 添加初始关键帧
								keyframes = append(keyframes, Keyframe{Time: 0, Value: initialValuePop})
								// 添加结束关键帧（时间归一化：percent/100）
								keyframes = append(keyframes, Keyframe{Time: timePercent / 100.0, Value: finalValue})
								// 标记下一个值已处理
								processedIndices[i+1] = true
								continue
							}
						}

						// Story 7.5 新增："value,timePercent" 格式（用于 CollisionReflect 等）
						// 条件：有初始值 且 val2看起来像百分比（>10）
						// 这是 PvZ 特有格式，用于混合格式如 ".3 .3,39.999996 0,50"
						// 例如：".3,39.999996" → value=0.3, time=39.999996%（有前导初始值）
						// 但 "0.5,10.5" → time=0.5, value=10.5（无前导值，标准格式）
						if initialValue != nil && val2 > 10 && val2 < 200 {
							// value,timePercent 格式（需要有初始值上下文）
							time := val2 / 100.0
							value := val1
							keyframes = append(keyframes, Keyframe{Time: time, Value: value})
						} else {
							// 标准 "time,value" 格式
							keyframes = append(keyframes, Keyframe{Time: val1, Value: val2})
						}
					}
				}
			} else {
				// 单独的值（可能是初始值）
				value, err := strconv.ParseFloat(part, 64)
				if err == nil {
					// 如果还没有关键帧，这个值作为初始值
					if len(keyframes) == 0 && initialValue == nil {
						initialValue = &value
						keyframes = append(keyframes, Keyframe{Time: 0, Value: value})
					}
					// 否则忽略（可能是格式错误）
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

	// Story 7.4 修复：如果 t 小于第一个关键帧的时间，返回第一个关键帧的值
	if t < keyframes[0].Time {
		return keyframes[0].Value
	}

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
