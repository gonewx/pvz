package config

// SeedChooser 选卡界面布局常量配置
// Story 8.12: 定义选卡界面的所有坐标、尺寸和动画参数

// ============================================================================
// 选卡面板位置和尺寸
// ============================================================================

const (
	// SeedChooserPanelX 选卡面板左上角X坐标
	SeedChooserPanelX = 0
	// SeedChooserPanelY 选卡面板左上角Y坐标
	SeedChooserPanelY = 87
	// SeedChooserPanelWidth 选卡面板宽度
	SeedChooserPanelWidth = 446
	// SeedChooserPanelHeight 选卡面板高度
	SeedChooserPanelHeight = 513
)

// ============================================================================
// 卡槽区域（顶部6个空位）
// 复用游戏中植物卡片的间隔配置 PlantCardSpacing
// ============================================================================

const (
	// SeedSlotStartX 卡槽区域起始X坐标（第一个卡槽左边缘）
	SeedSlotStartX = 84
	// SeedSlotStartY 卡槽区域起始Y坐标
	SeedSlotStartY = 8
	// SeedSlotWidth 单个卡槽宽度（复用全局植物卡片宽度）
	SeedSlotWidth = PlantCardWidth
	// SeedSlotHeight 单个卡槽高度（复用全局植物卡片高度）
	SeedSlotHeight = PlantCardHeight
	// SeedSlotSpacing 卡槽之间的水平间距
	// 复用游戏中植物卡片间隔：PlantCardSpacing - SeedSlotWidth = 59 - 50 = 9
	SeedSlotSpacing = PlantCardSpacing - SeedSlotWidth
	// SeedSlotCount 卡槽数量
	SeedSlotCount = 6
)

// ============================================================================
// 植物仓库区域（可选植物列表）
// 复用游戏中植物卡片的间隔配置 PlantCardSpacing
// ============================================================================

const (
	// SeedInventoryStartX 植物仓库起始X坐标
	SeedInventoryStartX = 20
	// SeedInventoryStartY 植物仓库起始Y坐标
	SeedInventoryStartY = 128
	// SeedInventoryCardWidth 仓库卡片宽度（复用全局植物卡片宽度）
	SeedInventoryCardWidth = PlantCardWidth
	// SeedInventoryCardHeight 仓库卡片高度（复用全局植物卡片高度）
	SeedInventoryCardHeight = PlantCardHeight
	// SeedInventoryCardSpacingX 卡片水平间距
	SeedInventoryCardSpacingX = 54 - SeedInventoryCardWidth
	// SeedInventoryCardSpacingY 卡片垂直间距
	SeedInventoryCardSpacingY = 3
	// SeedInventoryCardsPerRow 每行卡片数量
	SeedInventoryCardsPerRow = 8
)

// ============================================================================
// "一起摇滚吧!" 按钮
// ============================================================================

const (
	// SeedChooserButtonX 按钮中心X坐标
	SeedChooserButtonX = 223
	// SeedChooserButtonY 按钮中心Y坐标
	SeedChooserButtonY = 560
	// SeedChooserButtonWidth 按钮宽度
	SeedChooserButtonWidth = 158
	// SeedChooserButtonHeight 按钮高度
	SeedChooserButtonHeight = 42
)

// ============================================================================
// 僵尸预览区域（屏幕右侧）
// ============================================================================

const (
	// ZombiePreviewStartX 僵尸预览区域起始X坐标
	ZombiePreviewStartX = 600
	// ZombiePreviewStartY 僵尸预览区域起始Y坐标
	ZombiePreviewStartY = 100
	// ZombiePreviewSpacingX 僵尸之间的水平间距
	ZombiePreviewSpacingX = 80
	// ZombiePreviewSpacingY 僵尸之间的垂直间距
	ZombiePreviewSpacingY = 90
	// ZombiePreviewScale 僵尸预览缩放比例
	ZombiePreviewScale = 0.8
)

// ============================================================================
// 卡片飞行动画
// ============================================================================

const (
	// CardFlyDuration 卡片飞入/飞出动画时长（秒）
	CardFlyDuration = 0.3
	// CardFlyArcHeight 抛物线最高点高度（像素）
	CardFlyArcHeight = 50.0
)

// ============================================================================
// 悬停提示
// ============================================================================

const (
	// TooltipOffsetX 提示框相对于鼠标的X偏移
	TooltipOffsetX = 10
	// TooltipOffsetY 提示框相对于鼠标的Y偏移
	TooltipOffsetY = 20
	// TooltipPadding 提示框内边距
	TooltipPadding = 8
	// TooltipLineSpacing 提示框行间距
	TooltipLineSpacing = 4
)

// ============================================================================
// 选卡界面标题
// ============================================================================

const (
	// SeedChooserTitleX 标题中心X坐标
	SeedChooserTitleX = 223
	// SeedChooserTitleY 标题Y坐标
	SeedChooserTitleY = 93
	// SeedChooserTitle 标题文本
	SeedChooserTitle = "选择你的植物"
)
