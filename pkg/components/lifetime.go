package components

// LifetimeComponent 管理实体的生命周期
// 用于自动清理存在时间超过上限的实体(如阳光、子弹)
type LifetimeComponent struct {
	MaxLifetime     float64 // 最大生命周期(秒)
	CurrentLifetime float64 // 当前已存在时间(秒)
	IsExpired       bool    // 是否已过期
}




