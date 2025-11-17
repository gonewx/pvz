package components

import "github.com/hajimehoshi/ebiten/v2"

// BottomButtonType 定义底部功能按钮类型
type BottomButtonType int

const (
	// BottomButtonNone 无按钮（用于表示无悬停状态）
	BottomButtonNone BottomButtonType = -1
	// BottomButtonOptions 选项（设置）按钮
	BottomButtonOptions BottomButtonType = 0
	// BottomButtonHelp 帮助按钮
	BottomButtonHelp BottomButtonType = 1
	// BottomButtonQuit 退出按钮
	BottomButtonQuit BottomButtonType = 2
)

// BottomButtonComponent 底部功能按钮组件
// 用于主菜单右下角的 3 个花瓶样式按钮（选项/帮助/退出）
//
// 设计原则：
//   - 纯数据组件，不包含任何方法
//   - 支持正常态和悬停态图片切换
//   - 每个按钮类型有唯一的标识符
type BottomButtonComponent struct {
	// ButtonType 按钮类型（Options/Help/Quit）
	ButtonType BottomButtonType

	// NormalImage 正常状态图片
	NormalImage *ebiten.Image

	// HoverImage 悬停状态图片
	HoverImage *ebiten.Image

	// State 当前交互状态（Normal/Hovered/Clicked/Disabled）
	State UIState

	// Width 按钮宽度（像素）
	Width float64

	// Height 按钮高度（像素）
	Height float64
}
