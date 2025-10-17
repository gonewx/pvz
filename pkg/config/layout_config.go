package config

// 布局配置常量
// 本文件定义了游戏场景中的布局参数，包括网格系统、UI元素位置等

// Lawn Grid Configuration (草坪网格配置)
// 所有坐标使用"世界坐标系"（相对于背景图片左上角）
// 世界坐标是固定的，不随摄像机移动而变化
const (
	// GridWorldStartX 是草坪网格在背景图片中的起始X坐标（世界坐标）
	// 计算方式：屏幕坐标 + 游戏摄像机位置(220)
	GridWorldStartX = 252.0

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

	// GridWorldEndX 是草坪网格在背景图片中的结束X坐标（世界坐标）
	// 计算方式：起始X + 列数 * 格子宽度 = 251 + 9*80 = 971
	// 用于判断实体是否在草坪范围内（如豌豆射手攻击范围检测）
	GridWorldEndX = GridWorldStartX + float64(GridColumns)*CellWidth // 971.0

	// SodRoll 铺草皮动画配置
	// SodRoll.reanim 是特殊的场景动画，坐标系统与普通实体不同

	// SodRollBaseY 是 SodRoll动画整体的视觉中心Y坐标
	// 计算方法：综合SodRoll主体(y=244, 68x141缩放0.8)和SodRollCap盖子(y=326.7, 73x71缩放0.8)
	// 第1帧整体边界框：Y[244.0, 383.5]，视觉中心Y=(244.0+383.5)/2=313.75≈313.8
	// 注意：不应使用单一部件的Y坐标，会导致整体偏移
	SodRollBaseY = 313.8

	// SodRowWidth 是草皮图片的宽度（像素）
	// sod1row_.png 和 sod3row_.png 都是 771 像素宽
	SodRowWidth = 771.0
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

// CalculateSodRollPosition 根据启用的行范围计算 SodRoll 实体的 Position
// SodRoll.reanim 的坐标是相对于实体Position的偏移量
// 通过调整实体Position，让reanim的基准Y(244)对齐到目标行的中心
//
// 参数：
//   - enabledLanes: 启用的行列表，如 [3] 或 [2,3,4]（1-based）
//
// 返回：
//   - posX: SodRoll 实体的世界X坐标（应为0，让reanim的X直接等于世界坐标）
//   - posY: SodRoll 实体的世界Y坐标（动态调整，让动画对齐目标行）
func CalculateSodRollPosition(enabledLanes []int) (posX, posY float64) {
	if len(enabledLanes) == 0 {
		return 0, 0
	}

	// 找到启用行的范围
	minLane := enabledLanes[0]
	maxLane := enabledLanes[0]
	for _, lane := range enabledLanes {
		if lane < minLane {
			minLane = lane
		}
		if lane > maxLane {
			maxLane = lane
		}
	}

	// 计算目标行范围的中心Y坐标（世界坐标）
	// 例如：第2-4行，中心在第3行
	centerLane := float64(minLane+maxLane) / 2.0
	targetCenterY := GridWorldStartY + (centerLane-1.0)*CellHeight + CellHeight/2.0

	// 计算实体Position的偏移
	// 让 SodRoll动画的整体视觉中心(313.8) 对齐到目标中心Y
	// 最终渲染位置 = Position.Y + reanim部件Y
	// 整体视觉中心 = Position.Y + 313.8 = targetCenterY
	posX = 0                          // X固定为0，让reanim的X坐标直接等于世界坐标
	posY = targetCenterY - SodRollBaseY // Y动态调整，让整体视觉中心对齐目标行中心

	return posX, posY
}

// CalculateSodOverlayPosition 根据启用的行范围计算草皮叠加图的渲染位置
// 草皮叠加图（sod1row/sod3row）应该覆盖启用行的网格区域
//
// 参数：
//   - enabledLanes: 启用的行列表，如 [3] 或 [2,3,4]（1-based）
//   - sodImageHeight: 草皮图片的高度（sod1row=127, sod3row=355）
//
// 返回：
//   - sodX: 草皮叠加图的世界X坐标
//   - sodY: 草皮叠加图的世界Y坐标
func CalculateSodOverlayPosition(enabledLanes []int, sodImageHeight float64) (sodX, sodY float64) {
	if len(enabledLanes) == 0 {
		return 0, 0
	}

	// 找到启用行的范围
	minLane := enabledLanes[0]
	for _, lane := range enabledLanes {
		if lane < minLane {
			minLane = lane
		}
	}

	// 计算启用行范围的网格起始Y坐标（顶部对齐）
	// Story 8.2 QA修正：顶部对齐，不使用居中对齐
	// 原因：单行草皮图片（127px）高于行高（100px），居中对齐会导致
	//      向上溢出到上一行（-13.5px），影响视觉效果。顶部对齐可以让
	//      溢出部分全部在下方（+27px），符合原版游戏的显示效果。
	gridStartY := GridWorldStartY + float64(minLane-1)*CellHeight
	sodY = gridStartY // 顶部对齐，不添加 centerOffset

	// X坐标：草皮叠加图应该覆盖网格区域，不是从动画起点(10.3)开始
	// 草皮图片宽771px，网格宽720px（9列×80）
	// 草皮应该从网格起点稍左开始，覆盖整个网格区域
	// 左侧留约30px余量，让草皮边缘与网格左侧对齐
	sodX = GridWorldStartX - 30.0 // ≈ 252 - 30 = 222

	return sodX, sodY
}
