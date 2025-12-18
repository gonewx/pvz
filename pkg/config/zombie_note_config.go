package config

// ============================================================================
// 僵尸来信面板配置常量（Story 8.14）
// ============================================================================
//
// 本文件定义了僵尸来信面板的所有布局和动画参数。
// 修改这些常量可以调整面板的视觉效果和交互行为。

// === 淡入淡出动画配置 ===

// ZombieNoteFadeOutDuration 淡出持续时间（秒）
// 点击来信卡包后，画面渐暗到黑色的时间
const ZombieNoteFadeOutDuration = 0.5

// ZombieNoteFadeInDuration 淡入持续时间（秒）
// 黑屏后，来信面板渐显的时间
const ZombieNoteFadeInDuration = 0.5

// === 草皮背景裁剪配置 ===
//
// 从游戏背景图 (background1.jpg) 中裁剪一部分作为面板背景
// 指定起始点和裁剪宽度，高度根据目标屏幕比例自动计算
//
// 裁剪区域基于草坪网格坐标系计算：
//   - 起始点: (0.5格, 1.2格)
//   - 宽度: 约 6.7 格
//
// 像素计算公式：
//   X = GridWorldStartX + 格数 * CellWidth
//   Y = GridWorldStartY + 格数 * CellHeight
//
// 其中：
//   GridWorldStartX = 255.0
//   GridWorldStartY = 78.0
//   CellWidth = 80.0
//   CellHeight = 100.0

// NotePanelLawnCropX 草皮背景裁剪起始 X 坐标
// = 255 + 0.5 * 80 = 295
const NotePanelLawnCropX = 350

// NotePanelLawnCropY 草皮背景裁剪起始 Y 坐标
// = 78 + 1.2 * 100 = 198
const NotePanelLawnCropY = 150

// NotePanelLawnCropWidth 草皮背景裁剪宽度
// = 6.7 * 80 = 536
const NotePanelLawnCropWidth = 400

// NotePanelLawnBackgroundImage 草皮背景图片资源 ID
const NotePanelLawnBackgroundImage = "IMAGE_BACKGROUND1"

// === 面板布局配置 ===

// ZombieNotePanelOverlayAlpha 背景遮罩透明度（0-255）
// 半透明黑色遮罩，用于聚焦面板内容
const ZombieNotePanelOverlayAlpha = 128

// === 标题文字配置 ===

// ZombieNoteTitleKey LawnStrings 中的标题文本键
// 对应文本："你发现了张便签："
const ZombieNoteTitleKey = "FOUND_NOTE"

// ZombieNoteTitleFontSize 标题字体大小
const ZombieNoteTitleFontSize = 26.0

// ZombieNoteTitleOffsetY 标题文字距离面板顶部的偏移（像素）
const ZombieNoteTitleOffsetY = 60.0

// ZombieNoteTitleColor 标题文字颜色（复用奖励面板标题颜色）
var ZombieNoteTitleColor = RewardPanelTitleColor

// === 信件图片配置 ===

// ZombieNoteImageOffsetY 信件图片距离标题的偏移（像素）
const ZombieNoteImageOffsetY = 20.0

// === 按钮配置 ===

// ZombieNoteButtonY "下一关"按钮的Y位置比例（相对于屏幕高度）
// 复用奖励面板的按钮位置配置
const ZombieNoteButtonY = RewardPanelButtonY

// ZombieNoteButtonTextFontSize 按钮文字字体大小（复用奖励面板按钮字体大小）
const ZombieNoteButtonTextFontSize = RewardPanelButtonTextFontSize

// ZombieNoteMainMenuButtonX 主菜单按钮相对于屏幕宽度的 X 位置比例
const ZombieNoteMainMenuButtonX = RewardPanelMainMenuButtonX

// ZombieNoteMainMenuButtonY 主菜单按钮相对于屏幕高度的 Y 位置比例
const ZombieNoteMainMenuButtonY = RewardPanelMainMenuButtonY

// ZombieNoteMainMenuButtonFontSize 主菜单按钮字体大小（复用奖励面板配置）
const ZombieNoteMainMenuButtonFontSize = RewardPanelMainMenuButtonFontSize

// ZombieNoteMainMenuButtonTextColor 主菜单按钮文字颜色（复用奖励面板配置）
var ZombieNoteMainMenuButtonTextColor = RewardPanelMainMenuButtonTextColor

// === 资源路径配置 ===

// GetZombieNoteImagePath 根据 noteID 获取信件内容图片路径
// noteID: 如 "zombienote1" 对应 ZombieNote1.png
func GetZombieNoteImagePath(noteID string) string {
	switch noteID {
	case "zombienote1":
		return "assets/images/ZombieNote1.png"
	case "zombienote2":
		return "assets/images/ZombieNote2.png"
	case "zombienote3":
		return "assets/images/ZombieNote3.png"
	case "zombienote4":
		return "assets/images/ZombieNote4.png"
	default:
		return "assets/images/ZombieNote1.png"
	}
}

// ZombieNoteBackgroundJPG 便笺背景 JPG 路径
const ZombieNoteBackgroundJPG = "assets/images/ZombieNote.jpg"

// ZombieNoteBackgroundMask 便笺背景 Alpha 蒙板路径
const ZombieNoteBackgroundMask = "assets/images/ZombieNote_.png"
