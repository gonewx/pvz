package components

// PlantType 定义植物的类型
type PlantType int

const (
	// PlantSunflower 向日葵
	PlantSunflower PlantType = iota
	// PlantPeashooter 豌豆射手
	PlantPeashooter
)

// PlantCardComponent 表示植物选择卡片的数据
// 包含植物类型、消耗、冷却等信息
// 此组件用于 ECS 架构中标识植物卡片实体，并存储其状态数据
type PlantCardComponent struct {
	// PlantType 植物类型（向日葵或豌豆射手）
	PlantType PlantType
	// SunCost 种植消耗的阳光数量
	SunCost int
	// CooldownTime 冷却总时间（秒）
	CooldownTime float64
	// CurrentCooldown 当前剩余冷却时间（秒）
	CurrentCooldown float64
	// IsAvailable 是否可用（考虑阳光和冷却）
	IsAvailable bool
}
