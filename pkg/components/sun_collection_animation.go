package components

// SunCollectionAnimationComponent 存储阳光收集动画的状态
// 用于实现阳光飞向阳光池的缓动动画和缩放效果
//
// 工作流程：
//  1. InputSystem 点击阳光时添加此组件，记录起点和终点
//  2. SunMovementSystem 每帧根据 Progress 计算缓动位置和缩放
//  3. Progress 从 0.0 增长到 1.0 时，动画完成
//  4. SunCollectionSystem 检测到达目标后删除实体
type SunCollectionAnimationComponent struct {
	// StartX 起点X坐标（世界坐标）
	StartX float64

	// StartY 起点Y坐标（世界坐标）
	StartY float64

	// TargetX 终点X坐标（世界坐标）
	TargetX float64

	// TargetY 终点Y坐标（世界坐标）
	TargetY float64

	// Progress 当前动画进度（0.0 = 起点，1.0 = 终点）
	// SunMovementSystem 会根据 deltaTime 递增此值
	Progress float64

	// Duration 动画总时长（秒）
	// 例如：0.6 秒完成整个收集动画
	Duration float64
}
