package config

// Loading Scene 配置常量

const (
	// LoadingLogoTargetY Logo 最终 Y 坐标位置（下落动画目标）
	LoadingLogoTargetY float64 = 10

	// LoadingBarX 进度条 X 坐标（泥土底条居中）
	LoadingBarX float64 = 243

	// LoadingBarY 进度条 Y 坐标（泥土底条位置）
	LoadingBarY float64 = 536

	// LoadingGrassOffsetX 草皮条相对泥土条的 X 偏移
	LoadingGrassOffsetX float64 = -5

	// LoadingGrassOffsetY 草皮条相对泥土条的 Y 偏移
	LoadingGrassOffsetY float64 = -18

	// LoadingTextOffsetX 文字提示 X 偏移
	LoadingTextOffsetX float64 = 0

	// LoadingTextY 文字提示 Y 坐标
	LoadingTextY float64 = 552

	// LoadingDuration 加载动画总时长（秒）
	LoadingDuration float64 = 1.0

	// LoadingLogoAnimDuration Logo 下落动画时长（秒）
	LoadingLogoAnimDuration float64 = 1.5

	// LoadingTextFontSize 加载文字字体大小
	LoadingTextFontSize float64 = 20
)

// LoadingSproutTriggers 小动画触发进度阈值
var LoadingSproutTriggers = []float64{0.2, 0.4, 0.6, 0.8, 1.0}
