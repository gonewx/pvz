package components

// RewardAnimationComponent 管理奖励动画的状态和数据。
// 用于控制卡片包从屏幕右侧掉落到草坪中央的抛物线动画，
// 以及后续的弹跳、展开、展示奖励面板等阶段。
type RewardAnimationComponent struct {
	// Phase 表示当前动画阶段：
	// - "dropping": 抛物线运动阶段
	// - "bouncing": 弹跳动画阶段
	// - "expanding": 等待玩家点击展开阶段
	// - "showing": 显示奖励面板阶段
	// - "closing": 关闭奖励面板阶段
	Phase string

	// ElapsedTime 记录当前阶段已用时间（秒）
	ElapsedTime float64

	// StartX, StartY 抛物线动画的起点坐标（世界坐标）
	StartX, StartY float64

	// TargetX, TargetY 抛物线动画的终点坐标（世界坐标）
	TargetX, TargetY float64

	// VelocityX, VelocityY 抛物线运动的速度（像素/秒）
	VelocityX, VelocityY float64

	// PlantID 解锁的植物ID（如 "sunflower"）
	PlantID string

	// BounceCount 弹跳次数计数器
	BounceCount int

	// InitialBounceAmplitude 初始弹跳振幅（像素）
	InitialBounceAmplitude float64
}
