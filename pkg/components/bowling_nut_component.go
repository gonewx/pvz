package components

// BowlingNutComponent 保龄球坚果组件
// Story 19.6: 保龄球坚果实体的核心组件
//
// 此组件用于追踪保龄球坚果的状态，包括：
// - 滚动速度和方向
// - 所在行号
// - 是否为爆炸坚果
// - 弹射次数（为 Story 19.7 预留）
type BowlingNutComponent struct {
	// VelocityX 水平移动速度（像素/秒）
	// 正值表示向右移动
	VelocityX float64

	// Row 所在行号（0-4）
	// 坚果只在放置的行内移动
	Row int

	// IsRolling 是否正在滚动
	// 放置后立即设为 true
	IsRolling bool

	// IsExplosive 是否为爆炸坚果
	// true: 爆炸坚果（碰撞时 3x3 范围爆炸）
	// false: 普通坚果（碰撞后弹射）
	IsExplosive bool

	// BounceCount 弹射次数
	// Story 19.7 使用：普通坚果每次弹射后增加
	// 本 Story 不使用，预留字段
	BounceCount int

	// SoundPlaying 是否正在播放滚动音效
	// 用于避免重复播放和正确停止音效
	SoundPlaying bool
}

// BowlingNutType 保龄球坚果类型常量
const (
	// BowlingNutTypeNormal 普通保龄球坚果
	BowlingNutTypeNormal = "normal"

	// BowlingNutTypeExplosive 爆炸坚果
	BowlingNutTypeExplosive = "explosive"
)

