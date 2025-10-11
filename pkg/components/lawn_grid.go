package components

import "github.com/decker502/pvz/pkg/ecs"

// LawnGridComponent 标识草坪网格管理器实体
// 用于跟踪哪些格子已被植物占用
//
// Occupancy 是一个二维数组，存储每个格子的占用状态
// [row][col] = EntityID，其中 0 表示空格子
// 网格规格: 5行 x 9列
type LawnGridComponent struct {
	// Occupancy 存储每个格子的占用状态 (0 表示空格子)
	Occupancy [5][9]ecs.EntityID
}
