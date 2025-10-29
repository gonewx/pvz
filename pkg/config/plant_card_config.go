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

// PlantPreviewVisibleTracks 植物卡片预览图标的可见轨道白名单配置
// Story 11.1: 用于控制静态预览中哪些轨道应该显示
//
// 设计原则：
// - 排除眨眼动画轨道（anim_blink, idle_shoot_blink 等）
// - 排除动画定义轨道（只有 FrameNum，无图像）
// - 只包含构成植物"标准姿态"的部件轨道
var PlantPreviewVisibleTracks = map[string]map[string]bool{
	// 豌豆射手预览轨道
	"PeaShooterSingle": {
		"stalk_bottom":        true,
		"stalk_top":           true,
		"backleaf":            true,
		"backleaf_left_tip":   true,
		"backleaf_right_tip":  true,
		"frontleaf":           true,
		"frontleaf_right_tip": true,
		"frontleaf_tip_left":  true,
		"anim_sprout":         true, // 头后的小嫩叶
		"anim_head_idle":      true,
		"anim_face":           true,
		"idle_mouth":          true,
		// 注意：不包含 anim_blink, idle_shoot_blink（眨眼轨道）
	},

	// 向日葵预览轨道
	"SunFlower": {
		"anim_idle":           true,
		"backleaf":            true,
		"backleaf_left_tip":   true,
		"backleaf_right_tip":  true,
		"stalk_bottom":        true,
		"stalk_top":           true,
		"frontleaf":           true,
		"frontleaf_right_tip": true,
		"frontleaf_left_tip":  true,
		// 花瓣轨道（按 Z-order 顺序）
		"SunFlower_leftpetal8":       true,
		"SunFlower_leftpetal7":       true,
		"SunFlower_leftpetal6":       true,
		"SunFlower_leftpetal5":       true,
		"SunFlower_leftpetal4":       true,
		"SunFlower_leftpetal3":       true,
		"SunFlower_leftpetal2":       true,
		"SunFlower_leftpetal1":       true,
		"SunFlower_bottompetals":     true,
		"SunFlower_rightpetal9":      true,
		"SunFlower_rightpetal8":      true,
		"SunFlower_rightpetal7":      true,
		"SunFlower_rightpetal6":      true,
		"SunFlower_rightpetal5":      true,
		"SunFlower_rightpetal4":      true,
		"SunFlower_rightpetal3":      true,
		"SunFlower_rightpetal2":      true,
		"SunFlower_rightpetal1":      true,
		"SunFlower_toppetals":        true,
		"SunFlower_center":           true,
		"SunFlower_face":             true,
		"SunFlower_facehappy":        true,
		"SunFlower_facehappytalking": true,
		// 注意：不包含 anim_blink（眨眼轨道）
	},

	// 坚果墙预览轨道
	"Wallnut": {
		"_ground":   true,
		"anim_face": true,
		// 注意：不包含 anim_blink_twitch, anim_blink_twice, anim_blink_thrice
	},

	// 樱桃炸弹预览轨道
	"CherryBomb": {
		"CherryBomb_leftstem":    true,
		"CherryBomb_left1":       true,
		"CherryBomb_left3":       true,
		"CherryBomb_lefteye11":   true,
		"CherryBomb_lefteye21":   true,
		"CherryBomb_leftmouth1":  true,
		"CherryBomb_rightstem":   true,
		"CherryBomb_right1":      true,
		"CherryBomb_right3":      true,
		"CherryBomb_righteye11":  true,
		"CherryBomb_righteye21":  true,
		"CherryBomb_rightmouth1": true,
		// 根据实际 reanim 文件补充其他轨道
	},
}

// PlantPreviewFrameOverride 植物卡片预览图标的手工指定帧配置
// Story 11.1 - 策略 3：可选配置覆盖
//
// 用途：当 PrepareStaticPreview 的自动选择算法（第一个完整帧 + 启发式 fallback）
// 选择的帧不理想时，可以手工指定使用特定的帧索引
//
// 使用方式：
// - 如果某个植物在此 map 中有配置，优先使用配置的帧索引
// - 否则使用自动选择算法
//
// 示例：
// - "SunFlower": 10 表示向日葵使用第 10 帧作为预览
var PlantPreviewFrameOverride = map[string]int{
	// 当前所有植物都使用自动选择，如果发现某个植物预览不理想，在此添加配置
	// 例如：
	// "SunFlower": 10,  // 手工指定向日葵使用第10帧
	"CherryBomb": 0,
}
