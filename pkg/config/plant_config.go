package config

import "github.com/decker502/pvz/pkg/types"

// PlantResourceConfig 植物资源配置（统一管理）
// 将植物的资源名称、配置ID、预览帧、隐藏轨道等配置集中管理
type PlantResourceConfig struct {
	ResourceName     string   // Reanim 资源名称（如 "SunFlower"）
	ConfigID         string   // reanim_config.yaml 中的 ID（如 "sunflower"）
	PreviewFrame     int      // 预览帧索引（-1 表示自动选择）
	PreviewAnimation string   // 预览动画名称（如 "anim_glow"），空则使用第一个 combo
	HiddenTracks     []string // 预览隐藏轨道（黑名单模式，nil 表示显示所有）
}

// PlantConfigs 植物配置表（使用 types.PlantType 作为键）
var PlantConfigs = map[types.PlantType]*PlantResourceConfig{
	types.PlantSunflower: {
		ResourceName:     "SunFlower",
		ConfigID:         "sunflower",
		PreviewFrame:     10,
		PreviewAnimation: "anim_idle",
		HiddenTracks: []string{
			"anim_blink", // 隐藏眨眼轨道
		},
	},
	types.PlantPeashooter: {
		ResourceName:     "PeaShooterSingle",
		ConfigID:         "peashootersingle",
		PreviewFrame:     0,
		PreviewAnimation: "anim_full_idle",
		HiddenTracks: []string{
			"anim_blink",       // 隐藏眨眼轨道
			"idle_shoot_blink", // 隐藏射击眨眼轨道
		},
	},
	types.PlantWallnut: {
		ResourceName: "Wallnut",
		ConfigID:     "wallnut",
		PreviewFrame: -1, // 自动选择
		HiddenTracks: []string{
			"anim_blink", // 隐藏眨眼轨道
		},
	},
	types.PlantCherryBomb: {
		ResourceName: "CherryBomb",
		ConfigID:     "cherrybomb",
		PreviewFrame: 0,
		HiddenTracks: nil, // 无需隐藏轨道
	},
	types.PlantPotatoMine: {
		ResourceName:     "PotatoMine",
		ConfigID:         "potatomine",
		PreviewFrame:     -1,          // 自动选择
		PreviewAnimation: "anim_glow", // 与种植后动画一致
		HiddenTracks: []string{
			"anim_blink", // 隐藏眨眼轨道
		},
	},
}

// GetPlantConfig 获取植物配置
func GetPlantConfig(plantType types.PlantType) *PlantResourceConfig {
	if cfg, ok := PlantConfigs[plantType]; ok {
		return cfg
	}
	return nil
}

// GetPlantPreviewFrame 获取植物预览帧配置
// 返回值：帧索引，-1 表示自动选择
func GetPlantPreviewFrame(plantType types.PlantType) int {
	if cfg := GetPlantConfig(plantType); cfg != nil {
		return cfg.PreviewFrame
	}
	return -1
}

// GetPlantHiddenTracks 获取植物预览隐藏轨道配置（黑名单模式）
// 返回值：要隐藏的轨道列表，nil 表示显示所有轨道
func GetPlantHiddenTracks(plantType types.PlantType) []string {
	if cfg := GetPlantConfig(plantType); cfg != nil {
		return cfg.HiddenTracks
	}
	return nil
}

// 植物攻击动画关键帧配置
// 本文件定义了射手类植物的子弹发射关键帧号

const (
	// PeashooterShootingFireFrame 豌豆射手攻击动画的子弹发射帧号
	//
	// 初始推算（基于白皮书分析）：
	//   - 攻击动画时长: 0.5-0.7 秒
	//   - 默认 FPS: 12
	//   - 总帧数: 12 fps × 0.6s = 7-8 帧
	//   - 初始推测: Frame 5（身体猛地前倾时）
	//
	// 调优后的实际值：
	//   经过运行时测试和视觉验证，发现最佳帧号为 Frame 10
	//   在此帧时，子弹发射时机与攻击动画峰值完美同步
	//   （注：实际动画总帧数可能比推算值更多）
	//
	// 帧阶段划分（实测）：
	//   - Frame 0-3: 嘴巴向前嘟起（准备）
	//   - Frame 4-7: 身体向后压缩（蓄力）
	//   - Frame 8-10: 身体猛地前倾（峰值）← 发射子弹（Frame 10）
	//   - Frame 11+: 身体回弹，嘴巴恢复
	//
	// 注意：
	//   - 帧号从 0 开始计数
	//   - 如视觉不同步，可手动调整此值（通过观察 --verbose 日志）
	//   - 调整步长：+/- 1 帧，反复测试直到完美同步
	//
	// Story 10.5: 植物攻击动画帧事件同步
	PeashooterShootingFireFrame = 10

	// 未来扩展：其他射手植物的关键帧
	// SnowPeaShootingFireFrame    = 5  // 寒冰射手（与豌豆射手动画相同）
	// RepeaterShootingFireFrame1  = 5  // 双发射手（第一发）
	// RepeaterShootingFireFrame2  = 8  // 双发射手（第二发，延迟约 0.25秒）

	PlantOffsetY = 0.0
)

// 天空掉落阳光生成范围配置
// 确保阳光完整显示在屏幕内，不会部分或全部超出屏幕边界
// 阳光应该落在草坪区域内，避免与左侧卡片槽重叠
const (
	// SkyDropSunMinX 天空掉落阳光的最小 X 坐标（中心坐标）
	// 草坪起始X约为 255，加上阳光半径 40，确保阳光完整在草坪范围内
	SkyDropSunMinX = 280.0

	// SkyDropSunMaxX 天空掉落阳光的最大 X 坐标（中心坐标）
	// 屏幕宽度 800 - 阳光半径 40 = 760
	SkyDropSunMaxX = 760.0

	// SkyDropSunMinTargetY 天空掉落阳光落地的最小 Y 坐标（中心坐标）
	// 需要考虑阳光半径（40px），确保顶边不超出屏幕
	SkyDropSunMinTargetY = 100.0

	// SkyDropSunMaxTargetY 天空掉落阳光落地的最大 Y 坐标（中心坐标）
	// 屏幕高度 600 - 阳光半径 40 = 560，但考虑 UI 元素留出空间
	SkyDropSunMaxTargetY = 500.0
)

// 向日葵阳光生产位置配置
// Story 12.1: 向日葵阳光生产动画效果
const (
	// SunOffsetCenterX 阳光图像居中偏移（阳光约80px宽，居中需要减去40px）
	SunOffsetCenterX = 40.0

	// SunRandomOffsetRangeX 随机水平偏移范围（阳光落点X轴随机偏移 ±30px）
	// 实际偏移范围：[-30, +30] 像素
	SunRandomOffsetRangeX = 60.0

	// SunRandomOffsetRangeY 随机垂直偏移范围（阳光落点Y轴随机偏移 ±20px）
	// 实际偏移范围：[-20, +20] 像素
	SunRandomOffsetRangeY = 40.0

	// SunDropBelowPlantOffset 阳光目标位置相对于向日葵视觉中心的垂直偏移（向下）
	// 向日葵生产的阳光应该落在植物下方，这个值决定了阳光落点在视觉中心下方多少像素
	// 建议值：40-60像素（视觉上自然，不会太远也不会太近）
	SunDropBelowPlantOffset = 50.0
)

// 向日葵阳光生产调试配置
var (
	// SunflowerProduceSunEnabled 控制向日葵是否生产阳光
	// true: 正常生产阳光
	// false: 禁用阳光生产（调试发光效果时使用）
	SunflowerProduceSunEnabled = true // 调试模式：暂时禁用阳光生产
)

// 向日葵脸部发光效果配置
// 向日葵生产阳光时，整个向日葵（特别是脸部）会发出渐变的金色亮光
const (
	// SunflowerGlowColorR 向日葵发光效果的红色通道（叠加值）
	// 金色发光：R=1.5, G=1.0, B=0.3 （高于1.0才能有明显发光效果）
	SunflowerGlowColorR = 2.5

	// SunflowerGlowColorG 向日葵发光效果的绿色通道（叠加值）
	SunflowerGlowColorG = 1.0

	// SunflowerGlowColorB 向日葵发光效果的蓝色通道（叠加值）
	SunflowerGlowColorB = 0.3

	// SunflowerGlowRiseSpeed 向日葵发光效果的亮起速度（每秒）
	// 0.5 表示 2 秒内从最暗到最亮
	// 值越小，亮起越缓慢
	// 建议值：0.3-1.0
	SunflowerGlowRiseSpeed = 1.0

	// SunflowerGlowFadeSpeed 向日葵发光效果的衰减速度（每秒）
	// 0.2 表示 5 秒内从最亮完全衰减
	// 建议值：0.2-0.5
	SunflowerGlowFadeSpeed = 0.2

	// SunflowerGlowPrewarmTime 向日葵发光效果提前触发时间（秒）
	// 发光效果在阳光生产前多少秒开始亮起
	// 建议值：0.3-0.8
	SunflowerGlowPrewarmTime = 1.0
)
