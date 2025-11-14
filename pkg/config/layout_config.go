package config

import "github.com/decker502/pvz/internal/reanim"

// 布局配置常量
// 本文件定义了游戏场景中的布局参数，包括网格系统、UI元素位置等

// Background Configuration (背景配置)
const (
	// BackgroundWidth 背景图片宽度（像素）
	// 用于限制僵尸生成位置和镜头移动范围
	BackgroundWidth = 1400.0

	// BackgroundHeight 背景图片高度（像素）
	BackgroundHeight = 600.0
)

// Camera Configuration (摄像机配置)
const (
	// GameCameraX 是游戏摄像机的X位置（世界坐标）
	// 这是游戏正常运行时的摄像机位置，镜头居中对准草坪
	GameCameraX = 220.0

	// GridScreenStartX 是草坪网格在屏幕坐标系中的起始X坐标
	// 这是草坪相对于屏幕左侧的距离
	GridScreenStartX = 35.0
)

// Lawn Grid Configuration (草坪网格配置)
// 所有坐标使用"世界坐标系"（相对于背景图片左上角）
// 世界坐标是固定的，不随摄像机移动而变化
const (
	// GridWorldStartX 是草坪网格在背景图片中的起始X坐标（世界坐标）
	// 计算方式：屏幕坐标 + 游戏摄像机位置
	GridWorldStartX = GridScreenStartX + GameCameraX

	// GridWorldStartY 是草坪网格在背景图片中的起始Y坐标（世界坐标）
	// Y轴不受摄像机水平移动影响，因此世界坐标等于屏幕坐标
	GridWorldStartY = 78.0

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
	SodRollStartOffsetX = 0.0 // 相对于 GridWorldStartX 的偏移量

	// SodRoll 动画Y偏移（相对于目标行中心）
	// 调整此值可以改变草皮卷的垂直位置
	SodRollOffsetY = -15.0 // 相对于行中心的Y偏移量

	// 草皮叠加图X偏移（相对于网格起点）
	// 调整此值可以改变草皮显示的水平位置
	// SodOverlayOffsetX = -26.0 // 相对于 GridWorldStartX 的偏移量
	SodOverlayOffsetX = -20.0 // 相对于 GridWorldStartX 的偏移量

	// 草皮叠加图Y偏移（相对于目标行中心）
	// 调整此值可以改变草皮显示的垂直位置
	SodOverlayOffsetY = -3.0 // 相对于行中心的Y偏移量

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

	// ========== 僵尸生成位置配置参数（可手工调节） ==========

	// ZombieSpawnMinX 僵尸生成的最小X坐标（世界坐标）
	// 用于开场预览和正常游戏，僵尸在此范围内随机分布
	ZombieSpawnMinX = 1050.0

	// ZombieSpawnMaxX 僵尸生成的最大X坐标（世界坐标）
	// 不能超过背景宽度，留出边距避免僵尸贴边
	// 这是默认值，适用于第3、4、5行（没有特殊配置的行）
	ZombieSpawnMaxX = 1250.0 // 减少范围，让僵尸更快进入画面

	// ZombieSpawnMaxX_Row1 第1行僵尸生成的最大X坐标（世界坐标）
	// 建议值范围：1000.0 - 1350.0
	// 调整此值可以控制第1行僵尸的生成范围
	ZombieSpawnMaxX_Row1 = 1150.0

	// ZombieSpawnMaxX_Row2 第2行僵尸生成的最大X坐标（世界坐标）
	// 建议值范围：1000.0 - 1350.0
	// 调整此值可以控制第2行僵尸的生成范围
	ZombieSpawnMaxX_Row2 = 1200.0

	// OpeningZombiePreviewX 开场动画僵尸预览位置X坐标（已废弃，使用 ZombieSpawnMinX/MaxX 范围）
	// 保留此常量以保持向后兼容
	OpeningZombiePreviewX = 1200.0

	// ========== UI元素位置配置参数（可手工调节） ==========

	// SeedBankX 种子栏X坐标（屏幕坐标，像素）
	SeedBankX = 10

	// SeedBankY 种子栏Y坐标（屏幕坐标，像素）
	SeedBankY = 0

	// SeedBankWidth 种子栏宽度（像素）
	SeedBankWidth = 500

	// SeedBankHeight 种子栏高度（像素）
	SeedBankHeight = 87

	// SunCounterOffsetX 阳光计数器相对于 SeedBank 的 X 偏移量（像素）
	SunCounterOffsetX = 40

	// SunCounterOffsetY 阳光计数器相对于 SeedBank 的 Y 偏移量（像素）
	// 这是文字显示位置
	SunCounterOffsetY = 64

	// SunCounterWidth 阳光计数器宽度（像素）
	SunCounterWidth = 130

	// SunCounterHeight 阳光计数器高度（像素）
	SunCounterHeight = 60

	// SunCounterFontSize 阳光数值字体大小（像素）
	SunCounterFontSize = 18.0

	// SunPoolOffsetX 阳光池图标相对于 SeedBank 的 X 偏移量（像素）
	// 阳光收集目标位置（与文字对齐）
	SunPoolOffsetX = 40

	// SunPoolOffsetY 阳光池图标相对于 SeedBank 的 Y 偏移量（像素）
	// 阳光收集目标位置（在文字上方，池子中心）
	SunPoolOffsetY = 32

	// PlantCardStartOffsetX 第一张植物卡片相对于 SeedBank 的 X 偏移量（像素）
	PlantCardStartOffsetX = 84

	// PlantCardOffsetY 植物卡片相对于 SeedBank 的 Y 偏移量（像素）
	PlantCardOffsetY = 8

	// PlantCardSpacing 卡片槽之间的间距（像素）
	// 包含卡槽边框，每个卡槽约76px宽
	PlantCardSpacing = 60

	// ShovelX 铲子X坐标（屏幕坐标，像素）
	// 位于种子栏右侧（bar5.png width=612 + small gap）
	ShovelX = 620

	// ShovelY 铲子Y坐标（屏幕坐标，像素）
	ShovelY = 8

	// ShovelWidth 铲子宽度（像素）
	ShovelWidth = 70

	// ShovelHeight 铲子高度（像素）
	ShovelHeight = 74

	// IntroAnimDuration 开场动画时长（秒）
	IntroAnimDuration = 3.0

	// CameraScrollSpeed 开场动画摄像机滚动速度（像素/秒）
	CameraScrollSpeed = 100

	// MenuButtonOffsetFromRight 菜单按钮距离屏幕右边缘的距离（像素）
	MenuButtonOffsetFromRight = 145.0

	// MenuButtonOffsetFromTop 菜单按钮距离屏幕顶部的距离（像素）
	MenuButtonOffsetFromTop = 0.0

	// MenuButtonTextPadding 菜单按钮文字左右内边距（像素）
	MenuButtonTextPadding = 16.0

	// MenuButtonTextWidth 菜单按钮文字宽度（"菜单"两个字，像素）
	MenuButtonTextWidth = 32.0

	// ProgressBarOffsetFromRight 进度条距离屏幕右边缘的距离（像素）
	ProgressBarOffsetFromRight = 170.0

	// ProgressBarOffsetFromBottom 进度条距离屏幕底部的距离（像素）
	ProgressBarOffsetFromBottom = 60.0

	// ProgressBarZombieHeadOffsetX 僵尸头图标X偏移（相对于进度条左上角，像素）
	ProgressBarZombieHeadOffsetX = 8.0

	// ProgressBarZombieHeadOffsetY 僵尸头图标Y偏移（相对于进度条左上角，像素）
	ProgressBarZombieHeadOffsetY = 2.0

	// ProgressBarFillOffsetX 进度条填充X偏移（相对于进度条左上角，像素）
	ProgressBarFillOffsetX = 35.0

	// ProgressBarFillOffsetY 进度条填充Y偏移（相对于进度条左上角，像素）
	ProgressBarFillOffsetY = 16.0

	// ProgressBarLevelTextOffsetX 关卡编号文字X偏移（相对于进度条背景右边缘，像素）
	ProgressBarLevelTextOffsetX = 5.0

	// ProgressBarLevelTextOffsetY 关卡编号文字Y偏移（相对于进度条左上角，像素）
	ProgressBarLevelTextOffsetY = 8.0

	// ========== 暂停菜单配置参数（Story 10.1）（可手工调节） ==========

	// PauseMenuOverlayAlpha 暂停菜单遮罩透明度（0-255）
	PauseMenuOverlayAlpha = 150

	// PauseMenuBackToGameButtonFontSize "返回游戏"按钮字体大小（可调节）
	PauseMenuBackToGameButtonFontSize = 45.0

	// PauseMenuInnerButtonFontSize "重新开始"和"主菜单"按钮字体大小（可调节）
	PauseMenuInnerButtonFontSize = 20.0

	// PauseMenuBackToGameButtonOffsetY "返回游戏"按钮Y偏移（相对于屏幕中心，像素）
	// 正值向下，负值向上。调整此值控制按钮相对于墓碑背景的垂直位置
	PauseMenuBackToGameButtonOffsetY = 132.0

	// PauseMenuRestartButtonOffsetY "重新开始"按钮Y偏移（相对于屏幕中心，像素）
	// 此按钮在墓碑内部，位于顶部
	// 建议值范围：50.0 - 100.0
	PauseMenuRestartButtonOffsetY = 30.0

	// PauseMenuMainMenuButtonOffsetY "主菜单"按钮Y偏移（相对于屏幕中心，像素）
	// 此按钮在墓碑内部，位于中间偏下
	// 建议值范围：120.0 - 160.0
	PauseMenuMainMenuButtonOffsetY = 75.0

	// ========== 暂停菜单UI元素偏移（Story 10.1）（可手工调节） ==========

	// PauseMenuMusicSliderOffsetX 音乐滑动条X偏移（相对于屏幕中心，像素）
	PauseMenuMusicSliderOffsetX = 0.0

	// PauseMenuMusicSliderOffsetY 音乐滑动条Y偏移（相对于屏幕中心，像素）
	PauseMenuMusicSliderOffsetY = -120.0

	// PauseMenuSoundSliderOffsetX 音效滑动条X偏移（相对于屏幕中心，像素）
	PauseMenuSoundSliderOffsetX = 0.0

	// PauseMenuSoundSliderOffsetY 音效滑动条Y偏移（相对于屏幕中心，像素）
	PauseMenuSoundSliderOffsetY = -90.0

	// PauseMenu3DCheckboxOffsetX 3D加速复选框X偏移（相对于屏幕中心，像素）
	PauseMenu3DCheckboxOffsetX = 60.0

	// PauseMenu3DCheckboxOffsetY 3D加速复选框Y偏移（相对于屏幕中心，像素）
	PauseMenu3DCheckboxOffsetY = -60.0

	// PauseMenuFullscreenCheckboxOffsetX 全屏复选框X偏移（相对于屏幕中心，像素）
	PauseMenuFullscreenCheckboxOffsetX = 60.0

	// PauseMenuFullscreenCheckboxOffsetY 全屏复选框Y偏移（相对于屏幕中心，像素）
	PauseMenuFullscreenCheckboxOffsetY = -30.0

	// PauseMenuLabelFontSize UI元素标签文字字体大小（像素）
	PauseMenuLabelFontSize = 16.0

	// PauseMenuLabelOffsetX 标签文字X偏移（相对于滑动条/复选框位置，像素）
	// 负值表示在左侧
	PauseMenuLabelOffsetX = -80.0

	// PauseMenuLabelOffsetY 标签文字Y偏移（相对于滑动条/复选框位置，像素）
	// 用于垂直居中对齐
	PauseMenuLabelOffsetY = 0.0

	// PauseMenuInnerButtonWidth 墓碑内部按钮总宽度（像素）
	// 包含左右边框，中间部分宽度 = 总宽度 - 左右边框宽度
	PauseMenuInnerButtonWidth = 180.0

	// ========== 除草车配置参数（Story 10.2）（可手工调节） ==========

	// LawnmowerStartX 除草车初始X位置（世界坐标，像素）
	// 除草车放置在草坪左侧台阶上
	// 注意：必须在摄像机视野内（worldX >= GameCameraX = 220）
	// 设置为 230，屏幕坐标 = 230 - 220 = 10（刚好在屏幕左边缘）
	// 原建议值范围：30.0 - 100.0（错误，会在视野外）
	// 正确范围：220.0 - 260.0（在摄像机视野内，草坪左侧）
	LawnmowerStartX = 225.0

	// LawnmowerSpeed 除草车移动速度（像素/秒）
	// 原版除草车快速向右移动的速度
	// 建议值范围：200.0 - 400.0
	LawnmowerSpeed = 300.0

	// LawnmowerTriggerBoundary 除草车触发边界（世界坐标，像素）
	// 僵尸X坐标小于此值时触发除草车
	// 应该在除草车右侧，当僵尸靠近除草车时触发
	// 设置为 240（比除草车位置 230 稍右），给除草车留出启动空间
	LawnmowerTriggerBoundary = 240.0

	// LawnmowerDeletionBoundary 除草车删除边界（世界坐标，像素）
	// 除草车X坐标超过此值时删除实体
	// 通常设置为背景宽度
	LawnmowerDeletionBoundary = BackgroundWidth

	// LawnmowerWidth 除草车宽度（像素）
	// 用于碰撞检测和渲染
	LawnmowerWidth = 80.0

	// LawnmowerHeight 除草车高度（像素）
	// 用于碰撞检测和渲染
	LawnmowerHeight = 60.0

	// LawnmowerCollisionRange 除草车碰撞检测范围（像素）
	// 僵尸在除草车前后此范围内时视为碰撞
	// 建议值范围：30.0 - 80.0
	LawnmowerCollisionRange = 50.0
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
