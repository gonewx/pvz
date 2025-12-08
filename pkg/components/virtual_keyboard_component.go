package components

import "github.com/decker502/pvz/pkg/ecs"

// VirtualKeyboardComponent 虚拟键盘组件
// 用于移动端的屏幕键盘输入
type VirtualKeyboardComponent struct {
	// 显示状态
	IsVisible bool // 键盘是否可见

	// 模式状态
	ShiftActive bool // 是否大写模式
	NumericMode bool // 是否数字模式

	// 按键状态
	KeyStates     map[string]bool // 按键按下状态 (key label -> pressed)
	PressedKey    string          // 当前被按下的按键（用于视觉反馈）
	PressedTimer  float64         // 按下状态计时器（用于短暂高亮）

	// 输入消费状态（用于阻止事件穿透）
	InputConsumedThisFrame bool // 本帧是否消费了输入事件

	// 目标输入实体
	TargetInputEntity ecs.EntityID // 目标文本输入框实体

	// 布局配置（由系统在初始化时计算）
	KeyWidth      float64 // 按键宽度（像素）
	KeyHeight     float64 // 按键高度（像素）
	KeySpacing    float64 // 按键间距（像素）
	KeyboardY     float64 // 键盘Y坐标（屏幕底部）
	KeyboardX     float64 // 键盘X坐标（左边缘）
	ScreenWidth   float64 // 屏幕宽度（用于居中计算）
	ScreenHeight  float64 // 屏幕高度
}

// KeyInfo 按键信息（用于布局计算和点击检测）
type KeyInfo struct {
	Label       string  // 显示的文字
	Action      string  // 按键动作（SHIFT, BACKSPACE, SPACE, DONE, 123, ABC, 或字符本身）
	X           float64 // 按键左上角 X
	Y           float64 // 按键左上角 Y
	Width       float64 // 按键宽度
	Height      float64 // 按键高度
	WidthFactor float64 // 宽度倍数（相对于标准按键）
}

// 键盘布局定义

// KeyboardLayoutLower 小写字母布局
var KeyboardLayoutLower = [][]string{
	{"q", "w", "e", "r", "t", "y", "u", "i", "o", "p"},
	{"a", "s", "d", "f", "g", "h", "j", "k", "l"},
	{"SHIFT", "z", "x", "c", "v", "b", "n", "m", "BACKSPACE"},
	{"123", "SPACE", "DONE"},
}

// KeyboardLayoutUpper 大写字母布局
var KeyboardLayoutUpper = [][]string{
	{"Q", "W", "E", "R", "T", "Y", "U", "I", "O", "P"},
	{"A", "S", "D", "F", "G", "H", "J", "K", "L"},
	{"SHIFT", "Z", "X", "C", "V", "B", "N", "M", "BACKSPACE"},
	{"123", "SPACE", "DONE"},
}

// KeyboardLayoutNumeric 数字布局
var KeyboardLayoutNumeric = [][]string{
	{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"},
	{"ABC", "SPACE", "DONE"},
}

// 特殊按键宽度倍数
const (
	KeyWidthNormal    = 1.0 // 普通按键
	KeyWidthShift     = 1.5 // Shift 键
	KeyWidthBackspace = 1.5 // 退格键
	KeyWidth123       = 1.5 // 123/ABC 键
	KeyWidthSpace     = 4.0 // 空格键
	KeyWidthDone      = 2.0 // 确定键
)

// 特殊按键显示标签
const (
	LabelShift     = "Shift"
	LabelBackspace = "Del"
	LabelSpace     = ""         // 空格不显示文字
	LabelDone      = "Done"
	Label123       = "123"
	LabelABC       = "ABC"
)

// GetKeyWidthFactor 获取按键的宽度倍数
func GetKeyWidthFactor(action string) float64 {
	switch action {
	case "SHIFT":
		return KeyWidthShift
	case "BACKSPACE":
		return KeyWidthBackspace
	case "123", "ABC":
		return KeyWidth123
	case "SPACE":
		return KeyWidthSpace
	case "DONE":
		return KeyWidthDone
	default:
		return KeyWidthNormal
	}
}

// GetKeyLabel 获取按键的显示标签
func GetKeyLabel(action string) string {
	switch action {
	case "SHIFT":
		return LabelShift
	case "BACKSPACE":
		return LabelBackspace
	case "SPACE":
		return LabelSpace
	case "DONE":
		return LabelDone
	case "123":
		return Label123
	case "ABC":
		return LabelABC
	default:
		return action // 字母/数字直接返回
	}
}

// IsSpecialKey 判断是否为特殊按键（非字符输入）
func IsSpecialKey(action string) bool {
	switch action {
	case "SHIFT", "BACKSPACE", "SPACE", "DONE", "123", "ABC":
		return true
	default:
		return false
	}
}
