package components

// CollisionComponent 定义实体的碰撞检测边界框
// 用于物理系统检测实体之间的碰撞（如子弹与僵尸）
type CollisionComponent struct {
	Width   float64 // 碰撞盒宽度（像素）
	Height  float64 // 碰撞盒高度（像素）
	OffsetX float64 // 碰撞盒相对于实体位置的X偏移量（像素），正值向右偏移
	OffsetY float64 // 碰撞盒相对于实体位置的Y偏移量（像素），正值向下偏移

	// LaneIndex 实体所在的行号（0-based，0-4 对应 5 行草坪）
	// Story 8.9 修复：用于子弹与僵尸的同行碰撞检测
	// 撑杆僵尸跳跃时碰撞盒可能跨行，但 LaneIndex 保持不变
	// 子弹只能命中同一行的僵尸，即使 AABB 碰撞盒有重叠
	LaneIndex int
}
