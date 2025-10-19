package components

import "github.com/hajimehoshi/ebiten/v2"

// RewardPanelComponent 管理奖励面板的显示状态和动画数据。
// 用于展示新解锁的植物信息，包括植物名称、描述和卡片动画。
type RewardPanelComponent struct {
	// PlantID 解锁的植物ID（如 "sunflower"）
	PlantID string

	// PlantName 植物名称（从 LawnStrings 加载）
	PlantName string

	// PlantDescription 植物描述（从 LawnStrings 加载）
	PlantDescription string

	// SunCost 植物的阳光消耗值
	SunCost int

	// PlantIconTexture 植物图标纹理（Reanim 离屏渲染）
	PlantIconTexture *ebiten.Image

	// CardScale 卡片缩放比例（动画用，从 0.5 渐变到 1.5）
	CardScale float64

	// CardX, CardY 卡片位置（屏幕坐标）
	CardX, CardY float64

	// IsVisible 面板是否可见
	IsVisible bool

	// FadeAlpha 淡入淡出透明度（0.0 - 1.0）
	FadeAlpha float64

	// AnimationTime 动画时间计数器（秒）
	AnimationTime float64
}
