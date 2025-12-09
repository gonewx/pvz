package components

// ZombieWaveStateComponent 僵尸波次状态组件
//
// 用于标记僵尸属于哪一波，以及当前是否已激活（开始进攻）
type ZombieWaveStateComponent struct {
	// WaveIndex 所属波次索引（0-based，0表示第一波）
	WaveIndex int

	// IsActivated 是否已激活（开始进攻）
	// false: 僵尸在屏幕右侧站位等待
	// true: 僵尸开始向左移动进攻
	IsActivated bool

	// IndexInWave 在本波中的索引（0, 1, 2...）
	// 用于计算站位位置（避免重叠）
	IndexInWave int

	// ActivationDelay 激活延迟时间（秒）
	// 波次触发后，僵尸需等待此时间后才开始移动
	// 用于实现同一波次僵尸错开入场的散落效果
	ActivationDelay float64

	// ActivationTimer 激活计时器（秒）
	// 波次触发后开始计时，达到 ActivationDelay 后激活
	ActivationTimer float64

	// IsPendingActivation 是否等待激活中
	// true: 波次已触发但延迟未到，正在等待
	// false: 尚未触发或已完成激活
	IsPendingActivation bool
}
