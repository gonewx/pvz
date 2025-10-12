package utils

// 草坪网格坐标转换工具函数
// 本文件提供通用的网格坐标系统转换函数
// 使用世界坐标系统：所有坐标相对于背景图片左上角，不随摄像机移动而变化
//
// 坐标系统说明：
//   - 世界坐标：相对于背景图片左上角的绝对坐标（固定）
//   - 屏幕坐标：相对于游戏窗口左上角的坐标（随摄像机移动）
//   - 转换公式：worldX = screenX + cameraX
//              screenX = worldX - cameraX

// MouseToGridCoords 将鼠标屏幕坐标转换为草坪网格坐标
//
// 参数:
//   - mouseX, mouseY: 鼠标的屏幕坐标（相对于游戏窗口左上角）
//   - cameraX: 当前摄像机的X位置（世界坐标偏移量）
//   - gridWorldStartX, gridWorldStartY: 网格在背景图片中的起始位置（世界坐标）
//   - columns, rows: 网格的列数和行数
//   - cellWidth, cellHeight: 每个格子的宽度和高度（像素）
//
// 返回:
//   - col: 列索引 (0 到 columns-1)
//   - row: 行索引 (0 到 rows-1)
//   - isValid: 是否在有效网格范围内
func MouseToGridCoords(
	mouseX, mouseY int,
	cameraX float64,
	gridWorldStartX, gridWorldStartY float64,
	columns, rows int,
	cellWidth, cellHeight float64,
) (col, row int, isValid bool) {
	// 将鼠标屏幕坐标转换为世界坐标
	worldX := float64(mouseX) + cameraX
	worldY := float64(mouseY)

	// 计算网格的世界坐标边界
	gridEndX := gridWorldStartX + float64(columns)*cellWidth
	gridEndY := gridWorldStartY + float64(rows)*cellHeight

	// 检查是否在网格范围内
	if worldX < gridWorldStartX || worldX >= gridEndX || worldY < gridWorldStartY || worldY >= gridEndY {
		return 0, 0, false
	}

	// 计算列和行索引
	col = int((worldX - gridWorldStartX) / cellWidth)
	row = int((worldY - gridWorldStartY) / cellHeight)

	// 边界检查（防止浮点数计算误差导致的越界）
	if col < 0 {
		col = 0
	} else if col >= columns {
		col = columns - 1
	}

	if row < 0 {
		row = 0
	} else if row >= rows {
		row = rows - 1
	}

	return col, row, true
}

// GridToScreenCoords 将草坪网格坐标转换为屏幕中心坐标
//
// 参数:
//   - col: 列索引 (0 到 columns-1)
//   - row: 行索引 (0 到 rows-1)
//   - cameraX: 当前摄像机的X位置（世界坐标偏移量）
//   - gridWorldStartX, gridWorldStartY: 网格在背景图片中的起始位置（世界坐标）
//   - cellWidth, cellHeight: 每个格子的宽度和高度（像素）
//
// 返回:
//   - centerX, centerY: 格子中心的屏幕坐标
func GridToScreenCoords(
	col, row int,
	cameraX float64,
	gridWorldStartX, gridWorldStartY float64,
	cellWidth, cellHeight float64,
) (centerX, centerY float64) {
	// 先计算格子中心的世界坐标
	worldCenterX := gridWorldStartX + float64(col)*cellWidth + cellWidth/2
	worldCenterY := gridWorldStartY + float64(row)*cellHeight + cellHeight/2

	// 转换为屏幕坐标
	centerX = worldCenterX - cameraX
	centerY = worldCenterY

	return centerX, centerY
}

// GetEntityRow 根据实体的世界Y坐标计算其所在的行索引
//
// 此函数用于判断豌豆射手和僵尸是否在同一行，以决定是否发射子弹
//
// 参数:
//   - worldY: 实体的世界Y坐标（相对于背景图片左上角）
//   - gridWorldStartY: 网格在背景图片中的起始Y位置（世界坐标）
//   - cellHeight: 每个格子的高度（像素）
//
// 返回:
//   - row: 行索引 (0-based)
//
// 示例:
//   - 如果 gridWorldStartY=100, cellHeight=80
//   - worldY=180 → row=1 (第二行)
//   - worldY=260 → row=2 (第三行)
func GetEntityRow(worldY, gridWorldStartY, cellHeight float64) int {
	return int((worldY - gridWorldStartY) / cellHeight)
}
