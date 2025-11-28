package config

// ==============================
// 关卡进度条配置常量
// ==============================
// 注意：所有位置和偏移需要根据原版游戏截图手工调整

const (
	// ----------------
	// 进度条位置配置（屏幕右下角，右对齐）
	// ----------------
	// 窗口尺寸：800x600
	// 背景图尺寸：158x54
	ProgressBarRightMargin  float64 = 40  // 进度条距离屏幕右边缘的距离
	ProgressBarBottomMargin float64 = 0   // 进度条距离屏幕底部的距离
	ScreenWidth             float64 = 800 // 游戏窗口宽度
	ScreenHeight            float64 = 600 // 游戏窗口高度

	// ----------------
	// 进度条尺寸
	// ----------------
	ProgressBarBackgroundWidth  float64 = 158.0 // FlagMeter.png 背景框宽度
	ProgressBarBackgroundHeight float64 = 27.0  // FlagMeter.png 背景框单行高度（54/2）
	ProgressBarWidth            int     = 158   // 进度条实际宽度（与背景框宽度一致）
	ProgressBarHeight           int     = 27    // 进度条实际高度（背景框单行高度）

	// ----------------
	// FlagMeterLevelProgress.png 装饰条配置
	// ----------------
	LevelProgressDecorationOffsetY float64 = -6 // 装饰条垂直偏移手工调整（正值向下，负值向上）

	// ----------------
	// 进度条填充起始偏移（相对于背景图左上角）
	// ----------------
	ProgressBarStartOffsetX float64 = 10 // 绿色填充条X轴起始偏移
	ProgressBarStartOffsetY float64 = 5  // 绿色填充条Y轴起始偏移

	// ----------------
	// FlagMeterParts.png 精灵图配置
	// ----------------
	// 注意：FlagMeterParts.png 包含3个等宽的部分（僵尸头、分隔线、旗帜）
	// X坐标会在渲染时根据图片实际宽度自动计算
	PartsImageColumns int = 3 // 精灵图列数（僵尸头、分隔线、旗帜）

	// 僵尸头配置
	ZombieHeadOffsetX float64 = 0  // 僵尸头X轴偏移手工调整（正值向右，负值向左）
	ZombieHeadOffsetY float64 = -2 // 僵尸头Y轴偏移（相对于进度条中心）

	// 旗帜图标配置
	FlagIconOffsetY float64 = -15 // 旗帜Y轴偏移（相对于进度条中心）

	// ----------------
	// 关卡文本配置（右对齐）
	// ----------------
	LevelTextFontSize    float64 = 20                     // 关卡文本字体大小
	LevelTextRightMargin float64 = ProgressBarRightMargin // 关卡文本距离屏幕右边缘的距离（进度条隐藏时使用）
	LevelTextOffsetX     float64 = 20                     // 关卡文本右边缘距离进度条左边缘的距离（负值在左外侧，进度条显示时使用）
	LevelTextOffsetY     float64 = 10                     // 关卡文本Y轴偏移（相对于进度条顶部）
)

// ==============================
// Story 11.5: 原版进度条机制常量
// ==============================

const (
	// ----------------
	// 进度条结构常量
	// ----------------
	// 原版进度条总长度为 150 格（逻辑单位）
	ProgressBarTotalLength int = 150

	// 红字波段长度（每波 12 格）
	// 每次红字波刷新时，进度立即 +12
	ProgressBarFlagSegmentLength int = 12

	// 普通波段基础长度（无尽模式为 126）
	// 计算公式：TotalLength - (FlagCount × FlagSegmentLength)
	// 例如：1个旗帜波 = 150 - 12 = 138，2个旗帜波 = 150 - 24 = 126
	ProgressBarNormalSegmentDefault int = 126

	// ----------------
	// 虚拟/现实追踪常量
	// ----------------
	// 追踪速度（厘秒间隔）
	// 落后 1-6 格时，每 20cs (0.2秒) 前进一格
	SlowTrackIntervalCS int = 20

	// 落后 7+ 格时，每 5cs (0.05秒) 前进一格
	FastTrackIntervalCS int = 5

	// 快速追踪阈值（格数）
	// 当虚拟进度超过现实进度 7 格以上时，使用快速追踪
	FastTrackThreshold int = 7

	// ----------------
	// 时间常量
	// ----------------
	// 厘秒转换系数：1秒 = 100厘秒
	CentisecondsPerSecond float64 = 100.0
)

// DebugProgressBar 调试模式开关（启用后绘制边界框）
const DebugProgressBar bool = false
