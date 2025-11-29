package components

// LawnmowerComponent 除草车组件
// 用于表示每个除草车实体的状态和属性
type LawnmowerComponent struct {
	Lane        int     // 所在行（1-5，与EnabledLanes一致）
	IsTriggered bool    // 是否已触发（僵尸到达左侧）
	IsMoving    bool    // 是否正在移动
	Speed       float64 // 移动速度（像素/秒），默认 300.0

	// 入场动画状态
	IsEntering   bool    // 是否正在播放入场动画
	EnterStartX  float64 // 入场动画起始X位置（屏幕左侧外）
	EnterTargetX float64 // 入场动画目标X位置（最终停靠位置）
	EnterSpeed   float64 // 入场动画移动速度（像素/秒）
	EnterDelay   float64 // 入场动画延迟时间（秒，用于错开每行的动画）
	EnterTimer   float64 // 入场动画计时器（秒）
}

// LawnmowerStateComponent 全局除草车状态组件
// 挂载在一个全局实体上，跟踪所有行的除草车使用状态
type LawnmowerStateComponent struct {
	UsedLanes map[int]bool // 键：行号（1-5），值：是否已使用
}
