package components

// PoleVaultComponent 撑杆僵尸特有组件
// Story 8.9: 定义撑杆僵尸的跳跃状态和行为
//
// 撑杆僵尸的特殊机制：
// - 持杆时高速移动（8.5 像素/秒，普通僵尸的 1.8 倍）
// - 遇到第一个植物时触发跳跃，跳过该植物
// - 跳跃后丢弃撑杆，速度降为普通速度（4.7 像素/秒）
// - 跳跃后不再具备跳跃能力，遇到植物时正常攻击
type PoleVaultComponent struct {
	// HasPole 是否持有撑杆
	// true: 持杆状态，高速移动，可以跳跃
	// false: 已跳跃或失去撑杆，普通速度移动
	HasPole bool

	// IsJumping 是否正在跳跃中
	// 跳跃动画播放期间为 true
	IsJumping bool

	// JumpProgress 跳跃进度 (0.0 - 1.0)
	// 用于计算跳跃过程中的位置插值
	JumpProgress float64

	// JumpStartX 跳跃起始 X 坐标
	// 用于计算跳跃轨迹
	JumpStartX float64

	// JumpTargetX 跳跃目标 X 坐标
	// 跳跃完成后的最终位置
	JumpTargetX float64

	// TargetPlantEntityID 要跳过的植物实体ID
	// 用于在跳跃过程中避免与该植物发生碰撞
	TargetPlantEntityID uint64
}
