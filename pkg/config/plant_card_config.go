package config

// 植物卡片配置常量
//
// 设计原则：
// 1. 所有尺寸和偏移值都基于原始卡片尺寸（100x140）定义
// 2. 渲染时会统一应用 PlantCardScale 进行整体缩放
// 3. 这确保了卡片作为一个整体，所有元素（背景、图标、文字）按比例缩放
const (
	// PlantCardScale 标准植物卡片缩放因子
	// 用于选卡界面、奖励卡片包等所有场景，确保所有卡片大小一致
	// 原始卡片尺寸 100x140，缩放后 50x70
	PlantCardScale = 0.50

	// PlantCardBackgroundID 卡片背景图片资源ID
	// 背景图尺寸：100x140 像素
	PlantCardBackgroundID = "IMAGE_SEEDPACKET_LARGER"

	// PlantCardIconScale 植物图标相对于原始大小的缩放因子（相对缩放）
	// 图标原始尺寸：80x90 像素（离屏渲染生成）
	// 设为 1.0 表示图标使用原始尺寸，然后随 PlantCardScale 整体缩放
	// 可调整此值来微调图标在卡片中的大小（如 0.8 会让图标稍小）
	PlantCardIconScale = 1.0

	// PlantCardIconOffsetY 植物图标相对于卡片顶部的Y偏移（像素）
	// 基于原始卡片高度（140px），渲染时会乘以 PlantCardScale
	PlantCardIconOffsetY = 15.0

	// PlantCardSunCostOffsetY 阳光数字相对于卡片底部的Y偏移（像素）
	// 基于原始卡片高度（140px），渲染时会乘以 PlantCardScale
	PlantCardSunCostOffsetY = 18.0

	// PlantCardSunCostFontSize 阳光数字字体大小（基准值）
	// 渲染时会乘以 PlantCardScale，确保字体大小随卡片整体缩放
	PlantCardSunCostFontSize = 20
)
