package utils

// 草坪网格参数常量
// 这些常量定义了游戏草坪的网格布局,用于植物种植系统
const (
	GridStartX  = 250.0 // 网格起始X坐标
	GridStartY  = 90.0  // 网格起始Y坐标
	GridColumns = 9     // 网格列数
	GridRows    = 5     // 网格行数
	CellWidth   = 80.0  // 每格宽度
	CellHeight  = 100.0 // 每格高度
)

// MouseToGridCoords 将鼠标屏幕坐标转换为草坪网格坐标
// 参数:
//   - mouseX, mouseY: 鼠标的屏幕坐标
//
// 返回:
//   - col: 列索引 (0-8)
//   - row: 行索引 (0-4)
//   - isValid: 是否在有效网格范围内
func MouseToGridCoords(mouseX, mouseY int) (col, row int, isValid bool) {
	x := float64(mouseX)
	y := float64(mouseY)

	// 检查是否在网格范围内
	gridEndX := GridStartX + float64(GridColumns)*CellWidth
	gridEndY := GridStartY + float64(GridRows)*CellHeight

	if x < GridStartX || x >= gridEndX || y < GridStartY || y >= gridEndY {
		return 0, 0, false
	}

	// 计算列和行索引
	col = int((x - GridStartX) / CellWidth)
	row = int((y - GridStartY) / CellHeight)

	// 边界检查（防止浮点数计算误差导致的越界）
	if col < 0 {
		col = 0
	} else if col >= GridColumns {
		col = GridColumns - 1
	}

	if row < 0 {
		row = 0
	} else if row >= GridRows {
		row = GridRows - 1
	}

	return col, row, true
}

// GridToScreenCoords 将草坪网格坐标转换为屏幕中心坐标
// 参数:
//   - col: 列索引 (0-8)
//   - row: 行索引 (0-4)
//
// 返回:
//   - centerX, centerY: 格子中心的屏幕坐标
func GridToScreenCoords(col, row int) (centerX, centerY float64) {
	centerX = GridStartX + float64(col)*CellWidth + CellWidth/2
	centerY = GridStartY + float64(row)*CellHeight + CellHeight/2
	return centerX, centerY
}
