package components

// SunflowerGlowComponent 向日葵脸部发光效果组件
// 当向日葵生产阳光时，脸部会发出金色渐变亮光
//
// 工作原理：
//   - 向日葵生产阳光时，添加此组件并设置 Intensity = 0，IsRising = true
//   - 亮起阶段：每帧 Intensity 按 RiseSpeed 递增，直到达到 MaxIntensity
//   - 达到最大后，IsRising = false，开始衰减阶段
//   - 衰减阶段：每帧 Intensity 按 FadeSpeed 递减
//   - 渲染系统检测此组件，将发光颜色叠加到向日葵头部
//   - 当 Intensity <= 0 时，组件被移除
type SunflowerGlowComponent struct {
	// Intensity 当前发光强度（0-1）
	// 1.0 = 最亮，0.0 = 无发光
	Intensity float64

	// MaxIntensity 最大发光强度
	// 亮起阶段达到此值后开始衰减
	MaxIntensity float64

	// IsRising 是否处于亮起阶段
	// true = 正在变亮，false = 正在衰减
	IsRising bool

	// RiseSpeed 发光亮起速度（每秒增加量）
	// 0.5 表示 2 秒内从 0 到 1.0
	RiseSpeed float64

	// FadeSpeed 发光衰减速度（每秒衰减量）
	// 0.2 表示 5 秒内从最亮完全衰减
	FadeSpeed float64

	// ColorR 发光颜色红色通道（叠加到原色上）
	ColorR float64

	// ColorG 发光颜色绿色通道
	ColorG float64

	// ColorB 发光颜色蓝色通道
	ColorB float64
}

