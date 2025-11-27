package components

// DifficultyComponent 存储难度引擎相关的数据
// 用于计算轮数和级别容量上限
type DifficultyComponent struct {
	TotalCompletedFlags int  // 已完成的旗帜总数（跨关卡累计）
	RoundNumber         int  // 计算的轮数（可能为负数）
	IsSecondPlaythrough bool // 是否为二周目
	WavesPerRound       int  // 每轮波次数（默认20）
}
