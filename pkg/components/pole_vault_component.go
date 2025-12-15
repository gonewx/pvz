package components

// PoleVaultComponent 撑杆僵尸特有组件
// Story 8.9: 定义撑杆僵尸的跳跃状态和行为
//
// 撑杆僵尸的特殊机制：
// - 持杆时高速移动（约 54 像素/秒，普通僵尸的 1.8 倍）
// - 遇到第一个植物时触发跳跃，跳过该植物
// - 跳跃动画期间，body1 轨道位移制造视觉效果，实体位置不变
// - 跳跃完成时，实体位置补偿 body1 重置，保持视觉连续
// - 跳跃后丢弃撑杆，速度降为普通速度（30 像素/秒）
// - 跳跃后不再具备跳跃能力，遇到植物时正常攻击
type PoleVaultComponent struct {
	// HasPole 是否持有撑杆
	// true: 持杆状态，高速移动，可以跳跃
	// false: 已跳跃或失去撑杆，普通速度移动
	HasPole bool

	// IsJumping 是否正在跳跃中
	// 跳跃动画播放期间为 true
	IsJumping bool

	// TargetPlantEntityID 要跳过的植物实体ID
	// 用于在跳跃过程中避免与该植物发生碰撞
	TargetPlantEntityID uint64

	// NeedPositionCompensation 是否需要应用位移补偿
	// 跳跃完成后设为 true，延迟一帧应用，避免动画切换时的视觉跳变
	NeedPositionCompensation bool

	// JumpElapsedTime 跳跃已过时间（秒）
	// 用于判断当前是否在腾空阶段
	// 起跳阶段（0-0.5秒）和落地阶段（2.5秒后）可被子弹击中
	// 腾空阶段（0.5-2.5秒）免疫子弹
	JumpElapsedTime float64

	// JumpSoundPlayed 跳跃音效是否已播放
	// 用于确保音效只在起跳时播放一次
	JumpSoundPlayed bool
}
