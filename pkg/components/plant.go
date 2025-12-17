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

// PotatoMinePhase 土豆地雷阶段状态
// 用于管理土豆地雷的生命周期状态转换
type PotatoMinePhase int

const (
	// PotatoMineArming 武装阶段：埋在地下，播放 anim_idle，等待武装完成
	PotatoMineArming PotatoMinePhase = iota
	// PotatoMineRising 升起阶段：播放 anim_rise 动画
	PotatoMineRising
	// PotatoMineArmed 待机阶段：武装完成，播放 anim_armed + anim_light，等待僵尸触发
	PotatoMineArmed
	// PotatoMineExploding 爆炸阶段：播放 anim_mashed，造成伤害后销毁
	PotatoMineExploding
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

	// AttackAnimState 当前攻击动画状态（空闲/攻击中）
	AttackAnimState AttackAnimState

	// LastFiredFrame 上次发射子弹时的帧号
	// 用于防止在同一个关键帧内重复发射子弹（动画可能在同一帧停留多个 Update）
	LastFiredFrame int

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

	// Story 8.9: 土豆地雷相关字段
	// PotatoMinePhase 土豆地雷当前阶段
	// 使用明确的阶段状态来管理土豆地雷的生命周期
	PotatoMinePhase PotatoMinePhase

	// ArmingTimer 武装计时器（秒）
	// 土豆地雷种植后需要一定时间武装，武装完成后才能触发爆炸
	ArmingTimer float64

	// IsArmed 是否已武装（已废弃，使用 PotatoMinePhase 代替）
	// 保留用于向后兼容
	IsArmed bool

	// IsExploding 是否正在爆炸（已废弃，使用 PotatoMinePhase 代替）
	// 保留用于向后兼容
	IsExploding bool

	// WarningLightSpeed 警告灯闪烁动画速度倍率
	// 根据最近僵尸距离动态调整，僵尸越近速度越快
	// 默认 1.0，最快可达 4.0
	// 注意：此字段已废弃，改用 WarningLightTimer + WarningLightOn 实现
	WarningLightSpeed float64

	// WarningLightTimer 警告灯闪烁计时器（秒）
	// 计时器到期时切换灯的亮灭状态
	WarningLightTimer float64

	// WarningLightOn 警告灯是否亮起（红灯状态）
	// true: 显示 anim_glow（红灯），隐藏 anim_light（灰灯）
	// false: 显示 anim_light（灰灯），隐藏 anim_glow（红灯）
	WarningLightOn bool

	// WarningLightInterval 当前闪烁间隔（秒）
	// 根据最近僵尸距离动态计算，距离越近间隔越短（闪得越快）
	WarningLightInterval float64

	// WarningLightInitialized 警告灯是否已初始化
	// 用于在 PlayCombo 完成后执行一次性初始化
	WarningLightInitialized bool

	// BeingEaten 植物是否正在被僵尸啃食
	// 用于控制动画状态（被啃食时停止摇晃动画，保持静止）
	// 此字段为通用字段，适用于豌豆射手、向日葵等需要在被啃食时停止摇晃的植物
	BeingEaten bool

	// Story 8.12: 双发射手连发相关字段
	// BurstShotsRemaining 剩余连发次数
	// 双发射手每次攻击发射2颗豌豆，第一颗发射后此值为1，第二颗发射后为0
	BurstShotsRemaining int

	// BurstTimer 连发间隔计时器（秒）
	// 控制双发射手两颗豌豆之间的发射间隔
	BurstTimer float64
}

// Story 10.3: 射手类植物列表（用于判断是否需要攻击动画）
var shooterPlants = map[PlantType]bool{
	PlantPeashooter: true,
	PlantSnowPea:    true, // Story 8.9: 寒冰射手
	PlantRepeater:   true, // Story 8.12: 双发射手
}

// IsShooterPlant 判断植物是否是射手类（需要攻击动画）
// Story 10.3: 用于区分射手类植物和非射手类植物
func IsShooterPlant(plantType PlantType) bool {
	return shooterPlants[plantType]
}
