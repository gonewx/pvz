package components

// TutorialTextComponent 教学文本UI组件
// 控制教学提示文本的显示和渲染属性
// 用于在屏幕底部中央显示教学引导文本
type TutorialTextComponent struct {
	// Text 教学文本内容（从 LawnStrings.txt 加载）
	Text string

	// DisplayTime 已显示时间（秒）
	DisplayTime float64

	// MaxDisplayTime 最大显示时间（秒）
	// 0 表示无限显示，直到教学步骤完成
	MaxDisplayTime float64

	// BackgroundAlpha 背景透明度（0.0-1.0）
	// 当前版本不使用背景框，保留用于后续优化
	BackgroundAlpha float64
}

