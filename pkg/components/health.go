package components

// HealthComponent 存储实体的生命值信息
// 用于僵尸、植物等可被攻击的实体
type HealthComponent struct {
	CurrentHealth int // 当前生命值
	MaxHealth     int // 最大生命值
}
