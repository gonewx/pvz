package config

import "github.com/decker502/pvz/internal/reanim"

// 布局配置常量
// 本文件定义了游戏场景中的布局参数，包括网格系统、UI元素位置等

// Lawn Grid Configuration (草坪网格配置)
// 所有坐标使用"世界坐标系"（相对于背景图片左上角）
// 世界坐标是固定的，不随摄像机移动而变化
const (
	// GridWorldStartX 是草坪网格在背景图片中的起始X坐标（世界坐标）
	// 计算方式：屏幕坐标 + 游戏摄像机位置(220)
	GridWorldStartX = 263.0

	// GridWorldStartY 是草坪网格在背景图片中的起始Y坐标（世界坐标）
	// Y轴不受摄像机水平移动影响，因此世界坐标等于屏幕坐标
	GridWorldStartY = 76.0

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

	// ========== 草皮动画配置参数（可手工调节） ==========

	// SodRoll 动画起点X（相对于网格起点的偏移）
	// 调整此值可以改变草皮卷从哪里开始滚动
	SodRollStartOffsetX = -35.0 // 相对于 GridWorldStartX 的偏移量

	// SodRoll 动画Y偏移（相对于目标行中心）
	// 调整此值可以改变草皮卷的垂直位置
	SodRollOffsetY = -8.0 // 相对于行中心的Y偏移量

	// 草皮叠加图X偏移（相对于网格起点）
	// 调整此值可以改变草皮显示的水平位置
	SodOverlayOffsetX = -26.0 // 相对于 GridWorldStartX 的偏移量

	// 草皮叠加图Y偏移（相对于目标行中心）
	// 调整此值可以改变草皮显示的垂直位置
	SodOverlayOffsetY = 0.0 // 相对于行中心的Y偏移量

	// ========== 教学文本UI配置参数（可手工调节） ==========

	// TutorialTextBackgroundHeight 教学文本背景条高度（像素）
	TutorialTextBackgroundHeight = 100.0

	// TutorialTextOffsetFromBottom 教学文本距离屏幕底部的偏移（像素）
	// 文字Y坐标 = screenHeight - TutorialTextOffsetFromBottom
	TutorialTextOffsetFromBottom = 140.0

	// TutorialTextBackgroundOffsetFromBottom 教学文本背景条距离屏幕底部的偏移（像素）
	// 背景条顶部Y坐标 = screenHeight - TutorialTextBackgroundOffsetFromBottom
	TutorialTextBackgroundOffsetFromBottom = 180.0

	// TutorialTextBackgroundAlpha 教学文本背景条的透明度（0-255）
	TutorialTextBackgroundAlpha = 128
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

// CalculateSodRollPosition 计算草皮卷动画的起点位置（世界坐标）
// 动画终点由 reanim 文件的最后一帧决定，不需要配置
// 参数：
//   - enabledLanes: 启用的行列表
//   - sodImageHeight: 草皮图片高度（未使用，保留接口兼容性）
//   - reanimXML: SodRoll.reanim 数据（用于计算Y坐标对齐）
//
// 返回：
//   - startX: 动画起点X（世界坐标）= 网格起点 + 配置偏移
//   - startY: 动画起点Y（世界坐标）= 自动对齐到目标行中心
func CalculateSodRollPosition(enabledLanes []int, sodImageHeight float64, reanimXML *reanim.ReanimXML) (startX, startY float64) {
	if len(enabledLanes) == 0 {
		return 0, 0
	}

	// 计算目标行的中心Y坐标
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
	centerLane := float64(minLane+maxLane) / 2.0
	targetCenterY := GridWorldStartY + (centerLane-1.0)*CellHeight + CellHeight/2.0

	// X坐标：使用手工配置的偏移量
	startX = GridWorldStartX + SodRollStartOffsetX

	// Y坐标：自动对齐到目标行中心（需要 reanim 包围盒信息）
	if reanimXML != nil {
		// 从 reanim 数据计算包围盒
		var minY, maxY *float64
		for _, track := range reanimXML.Tracks {
			for _, frame := range track.Frames {
				if frame.Y != nil {
					y := *frame.Y
					if minY == nil || y < *minY {
						minY = &y
					}
					if maxY == nil || y > *maxY {
						maxY = &y
					}
				}
			}
		}

		// 如果找到了Y坐标，计算包围盒中心并对齐
		if minY != nil && maxY != nil {
			animCenterY := (*minY + *maxY) / 2.0
			startY = targetCenterY - animCenterY + SodRollOffsetY
		} else {
			// 降级：直接使用目标中心Y
			startY = targetCenterY + SodRollOffsetY
		}
	} else {
		// 没有 reanim 数据：直接使用目标中心Y
		startY = targetCenterY + SodRollOffsetY
	}

	return startX, startY
}

// CalculateSodOverlayPosition 计算草皮叠加图的渲染位置（世界坐标）
// 参数：
//   - enabledLanes: 启用的行列表
//   - sodImageHeight: 草皮图片高度
//
// 返回：
//   - sodX: 草皮叠加图左上角X坐标
//   - sodY: 草皮叠加图左上角Y坐标
func CalculateSodOverlayPosition(enabledLanes []int, sodImageHeight float64) (sodX, sodY float64) {
	if len(enabledLanes) == 0 {
		return 0, 0
	}

	// 计算目标行的中心Y坐标
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
	centerLane := float64(minLane+maxLane) / 2.0
	rowCenterY := GridWorldStartY + (centerLane-1.0)*CellHeight + CellHeight/2.0

	// 计算X坐标（网格起点 + 偏移）
	sodX = GridWorldStartX + SodOverlayOffsetX

	// 计算Y坐标（行中心 - 图片高度的一半 + 偏移）
	sodY = rowCenterY - sodImageHeight/2.0 + SodOverlayOffsetY

	return sodX, sodY
}
