package components

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// HelpPanelComponent 帮助面板组件
//
// 用途：
//   - 显示游戏帮助信息（操作说明、玩法指南等）
//   - 使用便笺背景 + 帮助文本的双层渲染
//   - 支持 Alpha 蒙板渲染（边缘透明效果）
//
// 资源构成：
//   - 便笺背景：ZombieNote.jpg +Alpha 蒙板 ZombieNote_.png
//   - 帮助文本：ZombieNoteHelp.png + Alpha 蒙板 ZombieNoteHelpBlack.png
//
// 渲染顺序：
//  1. 半透明遮罩（覆盖整个屏幕）
//  2. 便笺背景（带 Alpha 蒙板）
//  3. 帮助文本（带 Alpha 蒙板，叠加在便笺上）
//  4. 确定按钮（在便笺下方）
//
// Story 12.3: 对话框系统基础
type HelpPanelComponent struct {
	// 合成后的图片（预处理，避免每帧重新合成）
	BackgroundImage *ebiten.Image // 便笺背景（RGB + Alpha 蒙板合成）
	HelpTextImage   *ebiten.Image // 帮助文本（RGB + Alpha 蒙板合成）

	// 按钮实体 ID
	ConfirmButtonEntity uint64 // "确定"按钮实体 ID

	// 面板状态
	IsActive bool // 是否激活（显示）

	// 面板尺寸（用于居中计算）
	Width  float64
	Height float64
}
