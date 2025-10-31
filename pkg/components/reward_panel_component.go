package components

import "github.com/hajimehoshi/ebiten/v2"

// RewardPanelComponent 管理奖励面板的显示状态和动画数据。
// 用于展示新解锁的植物或工具信息，包括名称、描述和图标动画。
type RewardPanelComponent struct {
	// RewardType 奖励类型："plant" 或 "tool"（默认为空，向后兼容）
	RewardType string

	// PlantID 解锁的植物ID（如 "sunflower"）- RewardType="plant" 时使用
	PlantID string

	// ToolID 解锁的工具ID（如 "shovel"）- RewardType="tool" 时使用
	ToolID string

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

	// 注意：卡片位置由 RewardPanelRenderSystem 根据配置自动计算（水平居中，垂直位置从配置读取）

	// IsVisible 面板是否可见
	IsVisible bool

	// FadeAlpha 淡入淡出透明度（0.0 - 1.0）
	FadeAlpha float64

	// AnimationTime 动画时间计数器（秒）
	AnimationTime float64
}
