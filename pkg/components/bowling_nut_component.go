package components

// BowlingNutComponent 保龄球坚果组件
// Story 19.6: 保龄球坚果实体的核心组件
// Story 19.7: 添加弹射相关字段
//
// 此组件用于追踪保龄球坚果的状态，包括：
// - 滚动速度和方向
// - 所在行号
// - 是否为爆炸坚果
// - 弹射次数和弹射状态
type BowlingNutComponent struct {
	// VelocityX 水平移动速度（像素/秒）
	// 正值表示向右移动
	VelocityX float64

	// VelocityY 垂直移动速度（像素/秒）
	// Story 19.7: 弹射时的垂直速度
	// 正值向下，负值向上
	VelocityY float64

	// Row 所在行号（0-4）
	// 坚果只在放置的行内移动
	Row int

	// IsRolling 是否正在滚动
	// 放置后立即设为 true
	IsRolling bool

	// IsBouncing 是否正在弹射中
	// Story 19.7: 弹射开始时设为 true，到达目标行后设为 false
	IsBouncing bool

	// TargetRow 弹射目标行（0-4）
	// Story 19.7: 弹射的目标行号
	TargetRow int

	// IsExplosive 是否为爆炸坚果
	// true: 爆炸坚果（碰撞时 3x3 范围爆炸）
	// false: 普通坚果（碰撞后弹射）
	IsExplosive bool

	// BounceCount 弹射次数
	// Story 19.7: 普通坚果每次弹射后增加
	BounceCount int

	// CollisionCooldown 碰撞冷却时间（秒）
	// Story 19.7: 防止同一帧多次碰撞同一僵尸
	// 碰撞后设置冷却时间，冷却期间不检测碰撞
	CollisionCooldown float64

	// SoundPlaying 是否正在播放滚动音效
	// 用于避免重复播放和正确停止音效
	SoundPlaying bool

	// BounceDirection 弹射方向
	// -1 = 向上，1 = 向下，0 = 未弹射/已停止
	// 用于持续弹射到边缘的逻辑：弹射后如果没碰到僵尸，
	// 应该继续向同一方向弹射直到到达边缘行才反弹
	BounceDirection int
}

// BowlingNutType 保龄球坚果类型常量
const (
	// BowlingNutTypeNormal 普通保龄球坚果
	BowlingNutTypeNormal = "normal"

	// BowlingNutTypeExplosive 爆炸坚果
	BowlingNutTypeExplosive = "explosive"
)

