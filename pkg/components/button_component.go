package components

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ButtonType 定义按钮的渲染类型
type ButtonType int

const (
	// ButtonTypeNineSlice 三段式可拉伸按钮（左、中、右）
	ButtonTypeNineSlice ButtonType = iota
	// ButtonTypeSimple 简单图片按钮
	ButtonTypeSimple
)

// ButtonComponent 按钮组件（ECS 架构）
// 包含按钮的所有数据：外观、文字、状态、回调
//
// 设计原则：
//   - 纯数据组件，不包含任何方法
//   - 支持三段式可拉伸按钮和简单图片按钮
//   - 支持文字自动居中显示
//   - 支持点击回调
type ButtonComponent struct {
	// Type 按钮类型（三段式 or 简单图片）
	Type ButtonType

	// ===== 三段式按钮资源（ButtonTypeNineSlice）=====
	// LeftImage 左边缘图片
	LeftImage *ebiten.Image
	// MiddleImage 中间可拉伸图片
	MiddleImage *ebiten.Image
	// RightImage 右边缘图片
	RightImage *ebiten.Image
	// MiddleWidth 中间部分的宽度（像素）
	MiddleWidth float64

	// ===== 简单按钮资源（ButtonTypeSimple）=====
	// NormalImage 正常状态图片
	NormalImage *ebiten.Image
	// HoverImage 悬停状态图片（可选）
	HoverImage *ebiten.Image
	// PressedImage 按下状态图片（可选）
	PressedImage *ebiten.Image

	// ===== 按钮文字 =====
	// Text 按钮上显示的文字
	Text string
	// Font 文字字体
	Font *text.GoTextFace
	// TextColor 文字颜色（RGBA）
	TextColor [4]uint8 // R, G, B, A

	// ===== 按钮尺寸（自动计算）=====
	// Width 按钮总宽度（像素，自动计算）
	Width float64
	// Height 按钮高度（像素，自动计算）
	Height float64

	// ===== 按钮状态 =====
	// State 当前交互状态（Normal/Hover/Clicked/Disabled）
	State UIState
	// Enabled 是否启用（禁用时不响应点击）
	Enabled bool

	// ===== 点击回调 =====
	// OnClick 点击回调函数
	OnClick func()
}
