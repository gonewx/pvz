package components

import "github.com/hajimehoshi/ebiten/v2"

// SliderComponent 滑动条组件
// 用于音量控制等需要滑动调整数值的UI元素
type SliderComponent struct {
	// 滑动条图片
	SlotImage *ebiten.Image // 滑槽图片
	KnobImage *ebiten.Image // 滑块图片

	// 滑动条尺寸
	SlotWidth  float64 // 滑槽宽度
	SlotHeight float64 // 滑槽高度
	KnobWidth  float64 // 滑块宽度
	KnobHeight float64 // 滑块高度

	// 当前值（0.0 - 1.0）
	Value float64

	// 标签文字
	Label     string
	LabelFont *ebiten.Image // 预渲染的文字图片（可选）

	// 状态
	IsDragging bool // 是否正在拖动
	IsHovered  bool // 是否鼠标悬停

	// 回调函数
	OnValueChange func(value float64) // 值改变时的回调

	// 音效
	ClickSoundID string // 点击/开始拖拽时播放的音效ID
}
