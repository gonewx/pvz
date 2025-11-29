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

	// ========== Story 17.7: 旗帜波特殊计时字段 ==========

	// IsFlagWaveApproaching 是否正在接近旗帜波
	// 当下一波为旗帜波时设置为 true，用于触发红字警告
	IsFlagWaveApproaching bool

	// FlagWaveCountdownPhase 红字警告阶段
	// 0 = 无警告
	// 5 = 显示红字 "A Huge Wave of Zombies is Approaching!"
	// 4 = 红字停留阶段（725cs）
	FlagWaveCountdownPhase int

	// FlagWavePhaseTimeCs 当前阶段已持续时间（厘秒）
	// 用于控制 Phase 4 停留 725cs
	FlagWavePhaseTimeCs int

	// IsFinalWave 是否为最终波
	// 最终波有特殊的 5500cs 倒计时
	IsFinalWave bool

	// WaveElapsedCs 当前波刷出后已过的时间（厘秒）
	// 用于加速刷新判定（需要 > 401cs）
	WaveElapsedCs int

	// HugeWaveWarningTriggered 红字警告是否已触发
	// 防止重复触发警告
	HugeWaveWarningTriggered bool

	// ========== Story 17.8: 血量触发加速刷新字段 ==========

	// WaveInitialHealthCs 本波僵尸初始总血量（加权有效血量）
	// 计算公式: baseHealth + tier1AccessoryHealth + int(tier2AccessoryHealth * 0.20)
	// 在波次开始时由 WaveTimingSystem 设置
	WaveInitialHealthCs int

	// WaveCurrentHealthCs 本波僵尸当前总血量
	// 动态更新，用于与 WaveInitialHealthCs 比较判断血量触发条件
	WaveCurrentHealthCs int

	// HealthTriggerThreshold 血量触发比例阈值
	// 范围 [0.50, 0.65]，每波次随机生成
	// 当 WaveCurrentHealthCs <= WaveInitialHealthCs * HealthTriggerThreshold 时触发加速
	HealthTriggerThreshold float64

	// HealthAccelerationTriggered 是否已触发血量加速
	// 防止同一波内重复触发
	HealthAccelerationTriggered bool
}

