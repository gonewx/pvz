package config

// UI 布局相关的常量配置
// 包括底部功能按钮、对话框位置等 UI 元素的布局参数
//
// Story 12.2: 底部功能栏重构

// BottomButtonPosition 底部按钮位置配置
type BottomButtonPosition struct {
	X float64 // 按钮 X 坐标
	Y float64 // 按钮 Y 坐标
}

// BottomButtonPositions 底部功能按钮位置配置（可独立调整每个按钮）
// 索引顺序：0=选项(Options), 1=帮助(Help), 2=退出(Quit)
//
// 调整指南：
//   - X: 向右增加，向左减少
//   - Y: 向下增加，向上减少
//   - 按钮尺寸：选项 81x31, 帮助 48x22, 退出 47x27
//   - 屏幕尺寸：800x600
//
// Story 12.2: 底部功能栏重构（动画跟随版本）
var BottomButtonPositions = []BottomButtonPosition{
	// 选项按钮 (Options) - 81x31
	{X: 565.0, Y: 495.0},

	// 帮助按钮 (Help) - 48x22
	{X: 648.0, Y: 525.0}, // 500 + 81 + 10 = 591

	// 退出按钮 (Quit) - 47x27
	{X: 720.0, Y: 515.0}, // 591 + 48 + 10 = 649
}

// CalculateBottomButtonPosition 计算第 N 个底部按钮的位置
//
// 参数：
//   - buttonIndex: 按钮索引（0=选项, 1=帮助, 2=退出）
//
// 返回：
//   - x: 按钮 X 坐标
//   - y: 按钮 Y 坐标
//
// Story 12.2: 底部功能栏重构
func CalculateBottomButtonPosition(buttonIndex int) (x, y float64) {
	// 验证索引范围
	if buttonIndex < 0 || buttonIndex >= len(BottomButtonPositions) {
		return 0, 0
	}

	pos := BottomButtonPositions[buttonIndex]
	return pos.X, pos.Y
}

// BottomButtonClickPadding 底部按钮点击区域扩展（像素）
// 在实际按钮区域四周扩展此值，让点击更容易
//
// Story 12.2: 底部功能栏 - 可点击区域扩展
const BottomButtonClickPadding = 8.0
