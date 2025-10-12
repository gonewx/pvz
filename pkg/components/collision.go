package components

// CollisionComponent 定义实体的碰撞检测边界框
// 用于物理系统检测实体之间的碰撞（如子弹与僵尸）
type CollisionComponent struct {
	Width  float64 // 碰撞盒宽度（像素）
	Height float64 // 碰撞盒高度（像素）
}
