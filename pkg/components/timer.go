package components

// TimerComponent 通用计时器组件
// 用于处理需要时间延迟的行为（如生产周期、攻击冷却）
type TimerComponent struct {
	Name        string  // 计时器名称，如 "sun_production"
	TargetTime  float64 // 目标时间（秒）
	CurrentTime float64 // 当前已过时间（秒）
	IsReady     bool    // 计时器是否已完成
}
