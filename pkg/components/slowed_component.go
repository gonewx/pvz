package components

// SlowedComponent 存储实体的减速状态
// 用于实现寒冰射手的冰豌豆减速效果
type SlowedComponent struct {
	// SpeedMultiplier 速度倍率（0.5 表示减速 50%）
	// 当前速度 = 原始速度 * SpeedMultiplier
	SpeedMultiplier float64

	// Duration 剩余持续时间（秒）
	// 当 Duration <= 0 时，减速效果结束
	Duration float64

	// VisualEffect 是否显示视觉效果
	// true 表示显示冰蓝色叠加层或动画减慢效果
	VisualEffect bool

	// OriginalVX 原始 X 轴速度（用于恢复）
	// 在减速效果结束时恢复到此速度
	OriginalVX float64

	// OriginalVY 原始 Y 轴速度（用于恢复）
	OriginalVY float64

	// Applied 标记减速效果是否已应用到 VelocityComponent
	// 避免重复应用减速倍率
	Applied bool
}

// SlowEffectConfig 减速效果配置常量
const (
	// DefaultSlowSpeedMultiplier 默认减速倍率（减速 50%）
	DefaultSlowSpeedMultiplier = 0.5

	// DefaultSlowDuration 默认减速持续时间（10 秒）
	DefaultSlowDuration = 10.0
)
