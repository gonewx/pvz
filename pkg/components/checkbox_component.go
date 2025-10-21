package components

import "github.com/hajimehoshi/ebiten/v2"

// CheckboxComponent 复选框组件
// 用于开关选项（如全屏、3D加速等）
type CheckboxComponent struct {
	// 复选框图片
	UncheckedImage *ebiten.Image // 未选中状态图片
	CheckedImage   *ebiten.Image // 选中状态图片

	// 当前状态
	IsChecked bool

	// 标签文字
	Label     string
	LabelFont *ebiten.Image // 预渲染的文字图片（可选）

	// 回调函数
	OnToggle func(isChecked bool) // 状态切换时的回调
}
