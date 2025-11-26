package components

// SunflowerGlowComponent 向日葵脸部发光效果组件
// 当向日葵生产阳光时，脸部会发出金色渐变亮光
//
// 工作原理：
//   - 向日葵生产阳光时，添加此组件并设置 Intensity = 1.0
//   - 每帧 Intensity 会按 FadeSpeed 递减
//   - 渲染系统检测此组件，将发光颜色叠加到向日葵头部
//   - 当 Intensity <= 0 时，组件被移除
type SunflowerGlowComponent struct {
	// Intensity 当前发光强度（0-1）
	// 1.0 = 最亮，0.0 = 无发光
	Intensity float64

	// FadeSpeed 发光衰减速度（每秒衰减量）
	// 默认 2.0 表示 0.5 秒内完全衰减
	FadeSpeed float64

	// ColorR 发光颜色红色通道（叠加到原色上）
	ColorR float64

	// ColorG 发光颜色绿色通道
	ColorG float64

	// ColorB 发光颜色蓝色通道
	ColorB float64
}

