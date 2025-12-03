package components

// HoverHighlightComponent 悬停高亮组件
// 用于实体被鼠标悬停时的持续高亮效果（不闪烁）
//
// 使用场景：铲子模式下悬停植物、可交互实体高亮等
type HoverHighlightComponent struct {
	// Intensity 高亮强度（0.0 - 1.0）
	// 1.0 = 最亮，0.0 = 无效果
	Intensity float64

	// IsActive 是否激活
	IsActive bool
}
