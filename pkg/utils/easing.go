package utils

import "math"

// Easing Functions (缓动函数)
//
// 缓动函数用于控制动画的速度曲线，使动画看起来更自然。
// 所有函数接受一个进度值 t ∈ [0, 1]，返回缓动后的值 ∈ [0, 1]。
//
// 参考：https://easings.net/

// EaseLinear 线性缓动（无缓动）
// 返回值 = 输入值（匀速运动）
func EaseLinear(t float64) float64 {
	return t
}

// EaseOutCubic 三次方缓出
// 特点：开始快，结束慢（推荐用于"飞向目标"动画）
// 公式：f(t) = 1 - (1-t)³
func EaseOutCubic(t float64) float64 {
	return 1 - math.Pow(1-t, 3)
}

// EaseInCubic 三次方缓入
// 特点：开始慢，结束快
// 公式：f(t) = t³
func EaseInCubic(t float64) float64 {
	return t * t * t
}

// EaseInOutCubic 三次方缓入缓出
// 特点：开始慢，中间快，结束慢
// 公式：
//
//	t < 0.5: f(t) = 4t³
//	t >= 0.5: f(t) = 1 - (-2t + 2)³ / 2
func EaseInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	return 1 - math.Pow(-2*t+2, 3)/2
}

// EaseOutQuad 二次方缓出
// 特点：开始较快，结束慢（比 Cubic 更柔和）
// 公式：f(t) = 1 - (1-t)²
func EaseOutQuad(t float64) float64 {
	return 1 - (1-t)*(1-t)
}

// EaseInQuad 二次方缓入
// 特点：开始慢，结束较快
// 公式：f(t) = t²
func EaseInQuad(t float64) float64 {
	return t * t
}

// EaseOutExpo 指数缓出
// 特点：开始非常快，结束非常慢（适合"弹性"效果）
// 公式：f(t) = 1 - 2^(-10t)
func EaseOutExpo(t float64) float64 {
	if t >= 1.0 {
		return 1.0
	}
	return 1 - math.Pow(2, -10*t)
}

// Lerp 线性插值
// 在 a 和 b 之间根据 t 插值
// t=0 返回 a，t=1 返回 b
func Lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}
