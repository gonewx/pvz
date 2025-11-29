package components

// FlagWaveWarningComponent 红字警告组件
//
// Story 17.7: 用于标记和管理旗帜波来袭时的红字警告动画实体
// "A Huge Wave of Zombies is Approaching!"
//
// 该组件遵循 ECS 架构原则，只包含数据，不包含逻辑方法。
type FlagWaveWarningComponent struct {
	// Text 显示的警告文本
	// 默认: "A Huge Wave of Zombies is Approaching!"
	Text string

	// Phase 当前警告阶段
	// 5 = 初始显示（从大缩小）
	// 4 = 停留阶段（725cs）
	// 0 = 已完成
	Phase int

	// ElapsedTimeCs 已显示时间（厘秒）
	// 用于控制动画进度
	ElapsedTimeCs int

	// TotalDurationCs 总显示时长（厘秒）
	// 默认: 750cs
	TotalDurationCs int

	// Scale 当前缩放比例
	// 动画效果：从 2.0 缩小到 1.0
	Scale float64

	// Alpha 透明度
	// 范围: 0.0 - 1.0
	// 动画效果：淡入淡出
	Alpha float64

	// FlashTimer 闪烁计时器（厘秒）
	// 用于控制红字闪烁效果
	FlashTimer float64

	// FlashVisible 闪烁状态
	// true = 可见, false = 隐藏（闪烁效果）
	FlashVisible bool

	// IsActive 是否激活
	// 控制组件是否被系统处理
	IsActive bool

	// X, Y 显示位置（屏幕坐标）
	X float64
	Y float64
}

// FlagWaveWarningText 红字警告默认文本
const FlagWaveWarningText = "A Huge Wave of Zombies is Approaching!"


