package scenes

import (
	"image/color"
	"math"
	"testing"
)

// TestGetSunFlashColor_RedPhase 测试闪烁红色相位
// 规则: phase = timer % cycle, 当 phase < cycle/2 时为红色
func TestGetSunFlashColor_RedPhase(t *testing.T) {
	cycle := 0.3 // cycle/2 = 0.15

	tests := []struct {
		name  string
		timer float64
	}{
		{"开始时刻 (timer=1.0, phase=0.1)", 1.0},  // 1.0 % 0.3 = 0.1 < 0.15
		{"红色相位 (timer=0.9, phase=0.0)", 0.9},  // 0.9 % 0.3 = 0.0 < 0.15
		{"红色相位 (timer=0.7, phase=0.1)", 0.7},  // 0.7 % 0.3 = 0.1 < 0.15
		{"红色相位 (timer=0.1, phase=0.1)", 0.1},  // 0.1 % 0.3 = 0.1 < 0.15
		{"红色相位 (timer=0.05, phase=0.05)", 0.05}, // 0.05 % 0.3 = 0.05 < 0.15
	}

	expectedRed := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSunFlashColor(tt.timer, cycle)
			if result != expectedRed {
				phase := math.Mod(tt.timer, cycle)
				t.Errorf("期望红色 %v, 实际 %v (phase=%.2f)", expectedRed, result, phase)
			}
		})
	}
}

// TestGetSunFlashColor_BlackPhase 测试闪烁黑色相位
// 规则: 当 phase >= cycle/2 时为黑色
func TestGetSunFlashColor_BlackPhase(t *testing.T) {
	cycle := 0.3 // cycle/2 = 0.15

	tests := []struct {
		name  string
		timer float64
	}{
		{"黑色相位 (timer=0.8, phase=0.2)", 0.8},  // 0.8 % 0.3 = 0.2 >= 0.15
		{"黑色相位 (timer=0.5, phase=0.2)", 0.5},  // 0.5 % 0.3 = 0.2 >= 0.15
		{"黑色相位 (timer=0.25, phase=0.25)", 0.25}, // 0.25 % 0.3 = 0.25 >= 0.15
		{"黑色相位 (timer=0.2, phase=0.2)", 0.2},  // 0.2 % 0.3 = 0.2 >= 0.15
	}

	expectedBlack := color.RGBA{R: 0, G: 0, B: 0, A: 255}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSunFlashColor(tt.timer, cycle)
			if result != expectedBlack {
				phase := math.Mod(tt.timer, cycle)
				t.Errorf("期望黑色 %v, 实际 %v (phase=%.2f)", expectedBlack, result, phase)
			}
		})
	}
}

// TestGetSunFlashColor_CycleTransition 测试颜色周期转换
func TestGetSunFlashColor_CycleTransition(t *testing.T) {
	cycle := 0.3

	// 测试完整周期：根据 phase = timer % cycle 的规则
	// phase < cycle/2 (0.15) -> 红色
	// phase >= cycle/2 (0.15) -> 黑色
	timers := []float64{1.0, 0.9, 0.8, 0.7, 0.6, 0.5, 0.4, 0.3, 0.25, 0.2, 0.1, 0.05}
	expectedColors := []color.Color{
		color.RGBA{R: 255, G: 0, B: 0, A: 255}, // 1.0: 1.0 % 0.3 = 0.1 < 0.15 (红)
		color.RGBA{R: 255, G: 0, B: 0, A: 255}, // 0.9: 0.9 % 0.3 = 0.0 < 0.15 (红)
		color.RGBA{R: 0, G: 0, B: 0, A: 255},   // 0.8: 0.8 % 0.3 = 0.2 >= 0.15 (黑)
		color.RGBA{R: 255, G: 0, B: 0, A: 255}, // 0.7: 0.7 % 0.3 = 0.1 < 0.15 (红)
		color.RGBA{R: 255, G: 0, B: 0, A: 255}, // 0.6: 0.6 % 0.3 = 0.0 < 0.15 (红)
		color.RGBA{R: 0, G: 0, B: 0, A: 255},   // 0.5: 0.5 % 0.3 = 0.2 >= 0.15 (黑)
		color.RGBA{R: 255, G: 0, B: 0, A: 255}, // 0.4: 0.4 % 0.3 = 0.1 < 0.15 (红)
		color.RGBA{R: 255, G: 0, B: 0, A: 255}, // 0.3: 0.3 % 0.3 = 0.0 < 0.15 (红)
		color.RGBA{R: 0, G: 0, B: 0, A: 255},   // 0.25: 0.25 % 0.3 = 0.25 >= 0.15 (黑)
		color.RGBA{R: 0, G: 0, B: 0, A: 255},   // 0.2: 0.2 % 0.3 = 0.2 >= 0.15 (黑)
		color.RGBA{R: 255, G: 0, B: 0, A: 255}, // 0.1: 0.1 % 0.3 = 0.1 < 0.15 (红)
		color.RGBA{R: 255, G: 0, B: 0, A: 255}, // 0.05: 0.05 % 0.3 = 0.05 < 0.15 (红)
	}

	for i, timer := range timers {
		result := getSunFlashColor(timer, cycle)
		expected := expectedColors[i]
		if result != expected {
			phase := math.Mod(timer, cycle)
			t.Errorf("timer=%.2f, phase=%.2f: 期望 %v, 实际 %v", timer, phase, expected, result)
		}
	}
}

// TestGetSunFlashColor_DifferentCycles 测试不同周期
func TestGetSunFlashColor_DifferentCycles(t *testing.T) {
	tests := []struct {
		name          string
		cycle         float64
		timer         float64
		expectedColor color.Color
	}{
		{
			name:          "短周期 0.2秒 - 红色相位 (phase≈0.1)",
			cycle:         0.2,
			timer:         0.5, // 0.5 % 0.2 = 0.1 < 0.1 (cycle/2) -> 红色
			expectedColor: color.RGBA{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:          "短周期 0.2秒 - 黑色相位 (phase≈0.2)",
			cycle:         0.2,
			timer:         0.6, // 0.6 % 0.2 ≈ 0.2 >= 0.1 (cycle/2) -> 黑色 (浮点误差)
			expectedColor: color.RGBA{R: 0, G: 0, B: 0, A: 255},
		},
		{
			name:          "长周期 0.5秒 - 红色相位 (phase=0.0)",
			cycle:         0.5,
			timer:         1.0, // 1.0 % 0.5 = 0.0 < 0.25 (cycle/2) -> 红色
			expectedColor: color.RGBA{R: 255, G: 0, B: 0, A: 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSunFlashColor(tt.timer, tt.cycle)
			if result != tt.expectedColor {
				phase := math.Mod(tt.timer, tt.cycle)
				t.Errorf("期望 %v, 实际 %v (phase=%.2f, cycle/2=%.2f)",
					tt.expectedColor, result, phase, tt.cycle/2)
			}
		})
	}
}

// TestGetSunFlashColor_ZeroTimer 测试 timer 为 0 的边界情况
func TestGetSunFlashColor_ZeroTimer(t *testing.T) {
	cycle := 0.3
	timer := 0.0

	result := getSunFlashColor(timer, cycle)
	// 0.0 % 0.3 = 0.0 < 0.15 -> 红色
	expectedRed := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	if result != expectedRed {
		t.Errorf("timer=0 时期望红色 %v, 实际 %v", expectedRed, result)
	}
}
