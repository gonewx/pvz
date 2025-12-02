package components

// CollisionComponent 定义实体的碰撞检测边界框
// 用于物理系统检测实体之间的碰撞（如子弹与僵尸）
type CollisionComponent struct {
	Width   float64 // 碰撞盒宽度（像素）
	Height  float64 // 碰撞盒高度（像素）
	OffsetX float64 // 碰撞盒相对于实体位置的X偏移量（像素），正值向右偏移
	OffsetY float64 // 碰撞盒相对于实体位置的Y偏移量（像素），正值向下偏移
}
