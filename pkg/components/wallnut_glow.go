package components

// WallnutHitGlowComponent 坚果墙被啃食时的发光效果组件
// 当坚果墙被僵尸啃食时，会产生一闪一闪的发光效果
//
// 工作原理：
//   - 僵尸啃食坚果墙时，每次造成伤害添加/更新此组件
//   - 每帧 Intensity 会按 FadeSpeed 递减
//   - 渲染系统检测此组件，将发光颜色叠加到坚果墙
//   - 当 Intensity <= 0 时，组件被移除
type WallnutHitGlowComponent struct {
	// Intensity 当前发光强度（0-1）
	// 1.0 = 最亮，0.0 = 无发光
	Intensity float64

	// FadeSpeed 发光衰减速度（每秒衰减量）
	// 默认 4.0 表示 0.25 秒内完全衰减（快速闪烁效果）
	FadeSpeed float64

	// ColorR 发光颜色红色通道（叠加到原色上）
	ColorR float64

	// ColorG 发光颜色绿色通道
	ColorG float64

	// ColorB 发光颜色蓝色通道
	ColorB float64
}

