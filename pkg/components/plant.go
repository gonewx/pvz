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

	// Story 6.4: 眨眼动画计时器
	// BlinkTimer 眨眼计时器（秒）
	// 当计时器 <= 0 时，触发眨眼动画并重置为随机值（3-5秒）
	BlinkTimer float64
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
