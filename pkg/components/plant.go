package components

// AttackAnimState 攻击动画状态
// Story 10.3: 用于管理植物攻击动画状态转换
type AttackAnimState int

const (
	// AttackAnimIdle 空闲状态（播放 anim_idle）
	AttackAnimIdle AttackAnimState = iota
	// AttackAnimAttacking 攻击中（播放 anim_shooting）
	AttackAnimAttacking
)

// PlantComponent 标识实体为植物
// 包含植物类型和所在格子位置信息
//
// 此组件用于标记场景中已种植的植物实体，
// 并记录该植物在草坪网格中的位置
type PlantComponent struct {
	// PlantType 植物类型（向日葵、豌豆射手等）
	PlantType PlantType
	// GridRow 所在草坪行 (0-4, 从上到下)
	GridRow int
	// GridCol 所在草坪列 (0-8, 从左到右)
	GridCol int

	// Story 10.3: 攻击动画状态管理
	// AttackAnimState 当前攻击动画状态
	AttackAnimState AttackAnimState

	// PendingProjectile 是否有待发射的子弹
	// true 表示攻击动画已开始，等待关键帧到达时创建子弹
	// Story 10.5: 使用配置关键帧方案（方案 B），在 config.PeashooterShootingFireFrame 帧创建子弹
	PendingProjectile bool

	// LastFiredFrame 上次发射子弹时的帧号
	// 用于防止在同一个关键帧内重复发射子弹（循环动画问题）
	LastFiredFrame int

	// LastMouthX 上一帧 idle_mouth 轨道的 X 坐标（局部坐标）
	// 用于检测 X 坐标从增大变为减小（达到峰值，触发子弹发射）
	// idle_mouth 是嘴部部件，在攻击动画中随头部伸出而向右移动
	//
	// 注意：Story 10.5 当前未使用此字段（采用配置关键帧方案 B）
	// 保留此字段以备未来扩展（如需要峰值检测算法的特殊植物）
	LastMouthX float64

	// BlinkTimer 眨眼计时器（秒）
	// 当计时器 <= 0 时，触发眨眼动画并重置为随机值（3-5秒）
	// 注意：眨眼动画通过 PlayAnimation() 切换实现，不使用动画叠加
	BlinkTimer float64

	// WallnutDamageState 坚果墙受损状态（0=完好, 1=轻伤, 2=重伤）
	// 用于跟踪坚果墙的损坏程度，状态变化时触发大碎屑粒子效果
	WallnutDamageState int

	// WallnutBeingEaten 坚果墙是否正在被啃食
	// 用于控制动画切换（被啃食时播放 anim_blink_twitch，不摇摆）
	WallnutBeingEaten bool

	// WallnutBlinkTimer 坚果墙眨眼计时器（秒）
	// 被啃食时，每隔一段时间随机播放 anim_blink_twice 或 anim_blink_thrice
	WallnutBlinkTimer float64

	// WallnutBlinkDuration 坚果墙眨眼动画剩余持续时间（秒）
	// 当 > 0 时表示正在播放眨眼动画，递减到 0 后切换回静止状态
	WallnutBlinkDuration float64
}

// Story 10.3: 射手类植物列表（用于判断是否需要攻击动画）
var shooterPlants = map[PlantType]bool{
	PlantPeashooter: true,
	// 未来扩展：
	// PlantSnowPea:    true,
	// PlantRepeater:   true,
	// PlantCabbagePult: true,
}

// IsShooterPlant 判断植物是否是射手类（需要攻击动画）
// Story 10.3: 用于区分射手类植物和非射手类植物
func IsShooterPlant(plantType PlantType) bool {
	return shooterPlants[plantType]
}
