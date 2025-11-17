package components

import "github.com/hajimehoshi/ebiten/v2"

// TextInputComponent 文本输入框组件
// 用于在对话框中输入文本（如玩家名字）
type TextInputComponent struct {
	// 输入框文本
	Text string // 当前输入的文本

	// 输入框样式
	BorderImage      *ebiten.Image // editbox.gif 边框图片（可拉伸）
	BackgroundImage  *ebiten.Image // editbox_.gif 背景图片（可选）
	Width            float64       // 输入框宽度（像素）
	Height           float64       // 输入框高度（像素）

	// 光标状态
	CursorVisible    bool    // 光标是否可见（闪烁效果）
	CursorBlinkTimer float64 // 光标闪烁计时器（秒）
	CursorPosition   int     // 光标位置（字符索引）

	// 输入限制
	MaxLength        int    // 最大字符数（0 = 无限制）
	Placeholder      string // 占位符文本（输入框为空时显示）

	// 焦点状态
	IsFocused        bool // 是否获得焦点（接收键盘输入）

	// 文本渲染偏移（用于长文本滚动）
	TextOffsetX      float64 // 文本水平偏移（像素）

	// 内边距
	PaddingLeft      float64 // 左内边距（像素）
	PaddingRight     float64 // 右内边距（像素）
	PaddingTop       float64 // 上内边距（像素）
	PaddingBottom    float64 // 下内边距（像素）
}
