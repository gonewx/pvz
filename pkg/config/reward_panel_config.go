package config

import "image/color"

// 奖励面板配置常量
// 用于配置奖励面板UI的位置、大小、颜色等参数
//
// 注意：植物卡片的内部配置（图标缩放、阳光偏移等）在 plant_card_config.go 中定义

const (
	// 奖励面板背景尺寸（固定为 800x600）
	RewardPanelBackgroundWidth  = 800.0
	RewardPanelBackgroundHeight = 600.0

	// 文本垂直位置（相对于背景高度的比例，0.0-1.0）
	// 这些值可以手动调整以校准文本位置

	// RewardPanelTitleY 标题文字的Y位置比例（"你得到了一株新植物！"）
	RewardPanelTitleY = 0.065 // 在顶部边框内

	// RewardPanelCardYRatio 植物卡片的Y位置比例（相对于背景高度）
	// 卡片X位置自动居中，无需配置
	RewardPanelCardYRatio = 0.22 // 在背景中上部

	// RewardPanelPlantNameY 植物名称的Y位置比例（如"向日葵"）
	RewardPanelPlantNameY = 0.52 // 在卡片正下方

	// RewardPanelDescriptionY 植物描述的Y位置比例（如"提供你额外的阳光"）
	RewardPanelDescriptionY = 0.66 // 在卷轴区域

	// RewardPanelButtonY 按钮的Y位置比例（"下一关"按钮）
	RewardPanelButtonY = 0.88 // 在面板底部

	// RewardPanelCardScale 植物卡片缩放比例（相对于原始大小）
	// 1.0 = 原始大小，适合奖励面板展示
	RewardPanelCardScale = 1.0

	// 字体大小
	RewardPanelTitleFontSize      = 30 // 标题字体大小
	RewardPanelPlantInfoFontSize  = 22 // 植物名称和描述字体大小
	RewardPanelButtonTextFontSize = 20 // 按钮文字字体大小
)

// 文本颜色配置（根据原版截图）
var (
	// RewardPanelTitleColor 标题文字颜色（橙黄色）
	// 原版颜色：RGB(255, 200, 0) - 更偏橙色的金黄
	RewardPanelTitleColor = color.RGBA{255, 200, 0, 255}

	// RewardPanelPlantNameColor 植物名称颜色（金黄色）
	// 原版颜色：RGB(255, 215, 0) - 金色
	RewardPanelPlantNameColor = color.RGBA{255, 200, 0, 255}

	// RewardPanelDescriptionColor 植物描述颜色（深蓝黑色）
	// 原版颜色：RGB(30, 50, 70) - 深蓝偏黑
	RewardPanelDescriptionColor = color.RGBA{30, 50, 70, 255}

	// RewardPanelButtonTextColor 按钮文字颜色（橙黄色）
	// 原版颜色：RGB(255, 200, 50) - 橙黄色
	RewardPanelButtonTextColor = color.RGBA{255, 200, 0, 255}
)
