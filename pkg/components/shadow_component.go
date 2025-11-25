package components

// ShadowComponent 阴影组件
// 存储实体的阴影渲染参数
// 用于在实体脚下渲染椭圆形阴影,增加场景深度感
type ShadowComponent struct {
	// Width 阴影宽度 (像素)
	Width float64

	// Height 阴影高度 (像素)
	Height float64

	// Alpha 阴影透明度 (0.0-1.0)
	// 典型值: 0.65 (65% 透明度)
	Alpha float32

	// OffsetY 阴影垂直偏移 (像素)
	// 用于微调阴影位置,使其对齐实体脚底
	// 默认: 0
	OffsetY float64
}
