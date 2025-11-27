package components

// LaneStateComponent 行状态组件
//
// 用于平滑权重行分配算法，存储每行的权重和选取历史
type LaneStateComponent struct {
	LaneIndex        int     // 行号（1-6）
	Weight           float64 // 行权重（冒险模式初始值为 1）
	LastPicked       int     // 距离上次被选取的计数器
	SecondLastPicked int     // 距离上上次被选取的计数器
}
