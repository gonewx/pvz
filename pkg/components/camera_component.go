package components

// CameraComponent 管理镜头的目标位置和动画状态。
// 用于实现平滑的镜头移动效果（如开场动画的左右扫描）。
type CameraComponent struct {
	// TargetX 目标X坐标（世界坐标）
	TargetX float64

	// TargetY 目标Y坐标（世界坐标，当前未使用，预留）
	TargetY float64

	// AnimationSpeed 动画速度（像素/秒）
	AnimationSpeed float64

	// IsAnimating 是否正在动画中
	IsAnimating bool

	// EasingType 缓动类型：
	// - "linear": 线性运动
	// - "easeInOut": 二次缓动（先加速后减速）
	// - "easeOut": 减速运动
	EasingType string

	// StartX 动画起始X坐标（用于计算进度）
	StartX float64

	// TotalDistance 总移动距离（用于计算进度）
	TotalDistance float64
}
