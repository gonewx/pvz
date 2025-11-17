package components

import "github.com/hajimehoshi/ebiten/v2"

// UserSignComponent 木牌UI组件
//
// Story 12.4: 用户管理 UI
//
// 用于主菜单左上角的用户名显示木牌
// 当用户悬停/点击时，显示不同的图片
type UserSignComponent struct {
	// 当前用户名
	CurrentUsername string

	// 悬停状态
	IsHovered bool

	// 木牌图片资源（正常状态 + 按下状态）
	// 这些图片从 Reanim 轨道或资源管理器加载
	SignNormalImage *ebiten.Image // 正常状态图片（SelectorScreen_WoodSign2）
	SignPressImage  *ebiten.Image // 按下状态图片（SelectorScreen_WoodSign2_press）
}
