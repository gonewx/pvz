package components

import (
	"image/color"

	"github.com/decker502/pvz/pkg/ecs"
)

// TooltipComponent 提示框组件
// Story 10.8: 鼠标悬停植物卡片时显示提示信息
//
// 提示框布局（从上到下）：
//   1. 状态提示（第一行，可选）- 如"重新装填中..."、"没有足够的阳光"
//   2. 植物名称（第二行，必需）- 如"豌豆射手"、"向日葵"
//
// 样式规范：
//   - 背景色: 浅黄色 (#FFFFCC)
//   - 边框: 黑色 1-2px 实线
//   - 内边距: 8-10px
//   - 圆角: 0px (矩形，符合原版风格)
//   - 状态提示颜色: 红色 (警告状态)
//   - 植物名颜色: 黑色 (正常状态)
type TooltipComponent struct {
	// IsVisible 是否显示 Tooltip
	IsVisible bool

	// 文本内容 (渲染顺序: 状态提示在上, 植物名在下)
	StatusText      string      // 状态提示 (第一行, 可选): "重新装填中..." 或 "没有足够的阳光"
	StatusTextColor color.Color // 状态提示颜色 (红色)
	PlantName       string      // 植物名 (第二行, 必需): 如 "豌豆射手"
	PlantNameColor  color.Color // 植物名颜色 (黑色)

	// 样式
	BackgroundColor color.Color // 背景色(浅黄色 #FFFFCC)
	BorderColor     color.Color // 边框色(黑色)
	Padding         float64     // 内边距 (8-10px)
	TextSpacing     float64     // 两行文本间距 (3-5px)

	// 位置和尺寸
	X      float64 // Tooltip 左上角 X 坐标
	Y      float64 // Tooltip 左上角 Y 坐标
	Width  float64 // Tooltip 宽度（根据文本动态计算）
	Height float64 // Tooltip 高度（根据文本动态计算）

	// 关联实体
	TargetEntity ecs.EntityID // 目标卡片实体 ID
}

// NewTooltipComponent 创建 Tooltip 组件
// 返回一个初始化好的 TooltipComponent，使用默认颜色和样式
func NewTooltipComponent() *TooltipComponent {
	return &TooltipComponent{
		IsVisible:       false,
		BackgroundColor: color.RGBA{R: 255, G: 255, B: 204, A: 255}, // #FFFFCC 浅黄色
		BorderColor:     color.RGBA{R: 0, G: 0, B: 0, A: 255},       // 黑色
		StatusTextColor: color.RGBA{R: 255, G: 0, B: 0, A: 255},     // 红色
		PlantNameColor:  color.RGBA{R: 0, G: 0, B: 0, A: 255},       // 黑色
		Padding:         8.0,
		TextSpacing:     4.0,
	}
}
