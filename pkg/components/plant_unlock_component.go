package components

// PlantUnlockComponent 植物解锁状态组件
// 用于存储全局的植物解锁进度
// 选卡界面会根据此组件过滤可用植物（已解锁 ∩ 关卡配置允许）
type PlantUnlockComponent struct {
	UnlockedPlants map[string]bool // 已解锁植物映射表，key为植物ID（如"peashooter"），value为解锁状态
}
