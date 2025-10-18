package components

// ZombieTargetLaneComponent 僵尸目标行组件
//
// 用于跟踪僵尸需要移动到的目标行（有效行）
// 当僵尸在非有效行生成时，它会先在屏幕右侧站位，
// 然后在进攻时（进入屏幕前）移动到目标行
type ZombieTargetLaneComponent struct {
	// TargetRow 目标行索引（0-4，0-based）
	TargetRow int

	// HasReachedTargetLane 是否已到达目标行
	// true 表示僵尸已经在目标行上，可以正常进攻
	// false 表示僵尸还在移动到目标行的过程中
	HasReachedTargetLane bool
}
