package components

// FlashEffectComponent 闪烁效果组件
// 用于实体受击时的白色闪烁反馈效果（方案A+）
//
// 使用场景：僵尸受击时短暂闪白，增强视觉反馈
type FlashEffectComponent struct {
	// Duration 闪烁持续时间（秒）
	Duration float64

	// Elapsed 已经过的时间（秒）
	Elapsed float64

	// Intensity 闪烁强度（0.0 - 1.0）
	// 1.0 = 完全白色，0.0 = 无效果
	Intensity float64

	// IsActive 是否激活（用于临时禁用效果）
	IsActive bool
}
