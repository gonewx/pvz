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
}
