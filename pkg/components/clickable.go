package components

// ClickableComponent 标记实体可以被鼠标点击
// 定义了可点击区域的尺寸和是否启用点击
type ClickableComponent struct {
	Width     float64 // 可点击区域的宽度(像素)
	Height    float64 // 可点击区域的高度(像素)
	IsEnabled bool    // 是否可以被点击(用于禁用已点击的对象)
}
