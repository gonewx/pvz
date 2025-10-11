package components

// SunState 表示阳光的状态
type SunState int

const (
	SunFalling    SunState = iota // 正在下落
	SunLanded                     // 已落地,静止
	SunCollecting                 // 正在被收集(Story 2.4)
)

// SunComponent 标记实体为阳光,并存储阳光特定的状态
type SunComponent struct {
	State   SunState // 当前状态
	TargetY float64  // 目标落地Y坐标
}
