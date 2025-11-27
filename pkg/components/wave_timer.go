package components

// WaveTimerComponent 波次计时器组件
// 存储波次刷新计时状态，供 WaveTimingSystem 使用
// 注意：遵循 ECS 原则，组件仅存储数据，不包含方法
//
// 时间单位说明：
// 原版 PvZ 使用厘秒 (centiseconds, cs) 作为物理更新基准
// - 1 厘秒 = 0.01 秒
// - 100 厘秒 = 1 秒
// - 游戏逻辑以 100FPS 运行（固定时间步长 0.01 秒）
type WaveTimerComponent struct {
	// CountdownCs 当前倒计时（厘秒）
	// 每帧递减，当 <= 1 时触发下一波
	CountdownCs int

	// AccumulatedCs 累积的小数部分（用于精确计时）
	// 因为 deltaTime 转换为厘秒可能有小数，需要累积
	AccumulatedCs float64

	// IsFirstWave 是否为首波（用于开场倒计时逻辑）
	// 首波：立即触发（CountdownCs = 0）
	// 非首波：设置 CountdownCs = 599（约6秒）
	IsFirstWave bool

	// CurrentWaveIndex 当前波次索引（0-based）
	// 表示下一个要触发的波次
	CurrentWaveIndex int

	// TotalWaves 总波次数
	// 从关卡配置中读取
	TotalWaves int

	// IsPaused 是否暂停
	// 暂停时倒计时不递减
	IsPaused bool

	// WaveStartedAt 波次开始时间戳（调试用）
	// 记录最近一波触发的时间点（秒）
	WaveStartedAt float64

	// LastRefreshTimeCs 上次刷新设置的计时（调试用）
	// 记录 SetNextWaveCountdown 设置的值
	LastRefreshTimeCs int

	// WaveTriggered 是否触发了波次（本帧）
	// Update 后如果为 true，表示本帧应触发下一波
	// 由 LevelSystem 读取后重置
	WaveTriggered bool
}

