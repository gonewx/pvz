package config

// 布局配置常量
// 本文件定义了游戏场景中的布局参数，包括网格系统、UI元素位置等

// Lawn Grid Configuration (草坪网格配置)
// 所有坐标使用"世界坐标系"（相对于背景图片左上角）
// 世界坐标是固定的，不随摄像机移动而变化
const (
	// GridWorldStartX 是草坪网格在背景图片中的起始X坐标（世界坐标）
	// 计算方式：屏幕坐标(36) + 游戏摄像机位置(215) = 251
	GridWorldStartX = 251.0

	// GridWorldStartY 是草坪网格在背景图片中的起始Y坐标（世界坐标）
	// Y轴不受摄像机水平移动影响，因此世界坐标等于屏幕坐标
	GridWorldStartY = 72.0

	// GridColumns 是草坪的列数（横向格子数）
	GridColumns = 9

	// GridRows 是草坪的行数（纵向格子数）
	GridRows = 5

	// CellWidth 是每个格子的宽度（像素）
	CellWidth = 80.0

	// CellHeight 是每个格子的高度（像素）
	CellHeight = 100.0
)

// GetGridWorldBounds 返回草坪网格的世界坐标边界
// 返回值：startX, startY, endX, endY
func GetGridWorldBounds() (float64, float64, float64, float64) {
	startX := GridWorldStartX
	startY := GridWorldStartY
	endX := GridWorldStartX + float64(GridColumns)*CellWidth
	endY := GridWorldStartY + float64(GridRows)*CellHeight
	return startX, startY, endX, endY
}
