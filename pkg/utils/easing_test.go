package utils

import (
	"math"
	"testing"
)

// TestEaseLinear 测试线性缓动函数
func TestEaseLinear(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"起点", 0.0, 0.0},
		{"中点", 0.5, 0.5},
		{"终点", 1.0, 1.0},
		{"四分之一", 0.25, 0.25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EaseLinear(tt.input)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("EaseLinear(%v) = %v, 期望 %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestEaseOutCubic 测试三次方缓出函数
func TestEaseOutCubic(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"起点", 0.0, 0.0},
		{"终点", 1.0, 1.0},
		{"中点", 0.5, 0.875}, // 1 - (1-0.5)^3 = 1 - 0.125 = 0.875
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EaseOutCubic(tt.input)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("EaseOutCubic(%v) = %v, 期望 %v", tt.input, result, tt.expected)
			}
		})
	}

	// 验证"开始快，结束慢"的特性
	t.Run("开始快于线性", func(t *testing.T) {
		// 在前半段（p < 0.5），缓出函数应该比线性快
		for p := 0.1; p < 0.5; p += 0.1 {
			eased := EaseOutCubic(p)
			linear := EaseLinear(p)
			if eased <= linear {
				t.Errorf("EaseOutCubic(%v) = %v 应该大于线性值 %v（开始快）", p, eased, linear)
			}
		}
	})

	t.Run("整体快于线性", func(t *testing.T) {
		// EaseOut 的"结束慢"指的是速度减缓，而非位置落后
		// 由于前半段加速，整个过程中位置都会领先或等于线性
		for p := 0.0; p <= 1.0; p += 0.1 {
			eased := EaseOutCubic(p)
			linear := EaseLinear(p)
			// 允许微小的浮点误差
			if eased < linear-0.001 {
				t.Errorf("EaseOutCubic(%v) = %v 不应该落后于线性值 %v", p, eased, linear)
			}
		}
	})
}

// TestEaseInCubic 测试三次方缓入函数
func TestEaseInCubic(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"起点", 0.0, 0.0},
		{"终点", 1.0, 1.0},
		{"中点", 0.5, 0.125}, // 0.5^3 = 0.125
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EaseInCubic(tt.input)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("EaseInCubic(%v) = %v, 期望 %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestEaseOutQuad 测试二次方缓出函数
func TestEaseOutQuad(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"起点", 0.0, 0.0},
		{"终点", 1.0, 1.0},
		{"中点", 0.5, 0.75}, // 1 - (1-0.5)^2 = 1 - 0.25 = 0.75
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EaseOutQuad(tt.input)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("EaseOutQuad(%v) = %v, 期望 %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestLerp 测试线性插值函数
func TestLerp(t *testing.T) {
	tests := []struct {
		name     string
		a        float64
		b        float64
		t        float64
		expected float64
	}{
		{"起点", 0.0, 100.0, 0.0, 0.0},
		{"中点", 0.0, 100.0, 0.5, 50.0},
		{"终点", 0.0, 100.0, 1.0, 100.0},
		{"四分之一", 0.0, 100.0, 0.25, 25.0},
		{"负数范围", -50.0, 50.0, 0.5, 0.0},
		{"逆向范围", 100.0, 0.0, 0.5, 50.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Lerp(tt.a, tt.b, tt.t)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("Lerp(%v, %v, %v) = %v, 期望 %v", tt.a, tt.b, tt.t, result, tt.expected)
			}
		})
	}
}

// TestEaseOutCubicWithLerp 测试缓动函数与插值结合使用
// 模拟阳光收集动画的实际使用场景
func TestEaseOutCubicWithLerp(t *testing.T) {
	// 模拟阳光从 (100, 200) 飞向 (50, 50) 的动画
	startX, startY := 100.0, 200.0
	targetX, targetY := 50.0, 50.0

	// 测试不同进度下的位置
	tests := []struct {
		progress float64
		// 预期位置会更靠近终点（因为缓出）
	}{
		{0.0},
		{0.25},
		{0.5},
		{0.75},
		{1.0},
	}

	for _, tt := range tests {
		easedProgress := EaseOutCubic(tt.progress)
		x := Lerp(startX, targetX, easedProgress)
		y := Lerp(startY, targetY, easedProgress)

		// 验证边界
		if tt.progress == 0.0 {
			if math.Abs(x-startX) > 0.001 || math.Abs(y-startY) > 0.001 {
				t.Errorf("进度 0.0 时应该在起点: (%v, %v), 实际: (%v, %v)", startX, startY, x, y)
			}
		}
		if tt.progress == 1.0 {
			if math.Abs(x-targetX) > 0.001 || math.Abs(y-targetY) > 0.001 {
				t.Errorf("进度 1.0 时应该在终点: (%v, %v), 实际: (%v, %v)", targetX, targetY, x, y)
			}
		}

		// 验证 X 和 Y 都在起点和终点之间
		if x < targetX || x > startX {
			t.Errorf("X 坐标 %v 超出范围 [%v, %v]", x, targetX, startX)
		}
		if y < targetY || y > startY {
			t.Errorf("Y 坐标 %v 超出范围 [%v, %v]", y, targetY, startY)
		}
	}
}

// TestScaleAnimation 测试缩放动画（从 1.0 到 0.85）
func TestScaleAnimation(t *testing.T) {
	startScale := 1.0
	endScale := 0.85

	tests := []struct {
		progress      float64
		expectedScale float64
	}{
		{0.0, 1.0},
		{1.0, 0.85},
		{0.5, 0.86875}, // 1.0 + (0.85 - 1.0) * 0.875 = 1.0 - 0.15 * 0.875 = 0.86875
	}

	for _, tt := range tests {
		easedProgress := EaseOutCubic(tt.progress)
		scale := Lerp(startScale, endScale, easedProgress)

		if math.Abs(scale-tt.expectedScale) > 0.001 {
			t.Errorf("进度 %v 时，缩放应该是 %v，实际: %v (easedProgress=%v)", tt.progress, tt.expectedScale, scale, easedProgress)
		}

		// 验证缩放在合理范围内
		if scale < endScale || scale > startScale {
			t.Errorf("缩放 %v 超出范围 [%v, %v]", scale, endScale, startScale)
		}
	}
}
