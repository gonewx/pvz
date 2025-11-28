package components

// FinalWaveTextComponent 最终波白字组件
//
// Story 17.7: 用于标记和管理最终波 "FINAL WAVE" 白字显示
// 当最终波倒计时减至 0 时显示，持续 500cs
//
// 该组件遵循 ECS 架构原则，只包含数据，不包含逻辑方法。
type FinalWaveTextComponent struct {
	// Text 显示的文本
	// 默认: "FINAL WAVE"
	Text string

	// ElapsedTimeCs 已显示时间（厘秒）
	// 用于控制动画进度
	ElapsedTimeCs int

	// TotalDurationCs 总显示时长（厘秒）
	// 默认: 500cs
	TotalDurationCs int

	// Scale 当前缩放比例
	// 动画效果：从 3.0 缩小到 1.5
	Scale float64

	// Alpha 透明度
	// 范围: 0.0 - 1.0
	// 动画效果：淡入后保持
	Alpha float64

	// IsActive 是否激活
	// 控制组件是否被系统处理
	IsActive bool

	// IsComplete 是否显示完成
	// 500cs 后设置为 true
	IsComplete bool

	// X, Y 显示位置（屏幕坐标）
	X float64
	Y float64
}

// FinalWaveText 最终波白字默认文本
const FinalWaveText = "FINAL WAVE"
