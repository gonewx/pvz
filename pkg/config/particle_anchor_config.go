package config

// ParticleAnchorOffset 定义粒子效果的视觉锚点偏移
// 用于将调用者期望的"视觉中心"转换为 CreateParticleEffect 所需的"物理锚点"
//
// 背景：
//   - 粒子系统渲染时，粒子图片以 PositionComponent 为中心对齐
//   - EmitterOffsetX/Y 定义发射器相对于锚点的偏移
//   - 某些粒子效果的"视觉中心"不在锚点上（如 SeedPacket 的光晕中心在锚点下方62px）
//
// 用法：
//
//	调用者只需提供期望的"视觉中心"位置（如卡片中心），
//	通过 GetParticleAnchorOffset 获取偏移量，计算实际锚点位置
//
// 示例：
//
//	// 期望光晕中心对齐卡片中心
//	offsetX, offsetY := config.GetParticleAnchorOffset("SeedPacket")
//	anchorX := cardCenterX + offsetX  // 0
//	anchorY := cardCenterY + offsetY  // -62，锚点上移62px
//	CreateParticleEffect(..., anchorX, anchorY, ...)
//	// 结果：箭头在锚点处（卡片上方），光晕中心在锚点下方62px（卡片中心）
type ParticleAnchorOffset struct {
	OffsetX float64 // X轴偏移（正数向右）
	OffsetY float64 // Y轴偏移（正数向下，负数向上）
	Comment string  // 视觉效果说明
}

// Story 10.4: 植物种植粒子效果配置
const (
	// PlantingParticleEffect 种植粒子效果名称（原版配置）
	// 使用 Planting.xml 配置：地面土粒飞溅效果
	PlantingParticleEffect = "Planting"

	// PlantingParticleBackup 备用粒子效果名称
	// 当主配置加载失败时使用
	PlantingParticleBackup = "SodRoll"

	// PlantingParticleAngleOffset 种植粒子效果的角度偏移（度）
	// 用于调整粒子发射方向，例如：
	//   -90: 向左偏转 90 度
	//     0: 不偏转（使用 XML 原始角度）
	//    90: 向右偏转 90 度
	//   180: 反向（上下翻转）
	// 注：原版 Planting.xml 的 LaunchAngle 为 [110 250]（向上飞溅）
	PlantingParticleAngleOffset = -90.0
)

// ParticleAnchorOffsets 粒子效果锚点偏移配置表
//
// 设计原则：
//   - 不修改原版 XML 文件（保持原版兼容性）
//   - 不暴露 XML 内部数值（通过语义化配置抽象）
//   - 集中管理所有粒子效果的视觉锚点
//
// 配置依据：
//   - 分析 XML 中 EmitterOffsetX/Y 的定义
//   - 分析粒子图片尺寸（如光晕 145px 高）
//   - 根据视觉效果要求（如"光晕中心对齐目标"）计算锚点偏移
var ParticleAnchorOffsets = map[string]ParticleAnchorOffset{
	// SeedPacket: 植物卡片选中效果（箭头+光晕）
	// XML 定义：
	//   - Emitter 0 (箭头): EmitterOffsetY=0, 图片26px高
	//   - Emitter 1 (光晕): EmitterOffsetY=62, 图片145px高
	// 视觉要求：
	//   - 箭头在目标上方指向目标（动画向下晃动50px）
	//   - 光晕中心对齐目标中心（145px高的光晕包裹70px高的卡片）
	// 计算：
	//   - 光晕中心 = 锚点Y + 62
	//   - 期望光晕中心 = 卡片中心
	//   - 锚点Y = 卡片中心 - 62
	"SeedPacket": {
		OffsetX: 0,
		OffsetY: -62,
		Comment: "光晕中心对齐卡片中心，箭头在卡片上方",
	},

	// AwardPickupArrow: 奖励拾取箭头（箭头+光晕）
	// 与 SeedPacket 视觉效果相同
	"AwardPickupArrow": {
		OffsetX: 0,
		OffsetY: -62,
		Comment: "光晕中心对齐奖励中心，箭头在奖励上方",
	},

	// Planting: 植物种植土粒飞溅效果
	// 调用者提供格子中心坐标（GridToWorldCoords 返回值）
	// 粒子效果应该在植物根部（格子底部）显示
	// 计算：
	//   - 格子中心 = GridWorldStartY + row * CellHeight + CellHeight/2
	//   - 植物根部 = 格子中心 + CellHeight/2 = 格子中心 + 50.0
	//   - OffsetY = +50.0（向下移动到根部）
	"Planting": {
		OffsetX: 0,
		OffsetY: 30.0, // CellHeight / 2，从格子中心移动到底部（根部）
		Comment: "土粒从植物根部（格子底部）飞溅",
	},

	// 注：其他未列出的粒子效果使用默认值（OffsetX=0, OffsetY=0）
	// 即锚点 = 视觉中心，调用者提供的坐标直接作为粒子锚点
}

// GetParticleAnchorOffset 获取粒子效果的锚点偏移
//
// 参数：
//   - effectName: 粒子效果名称（如 "SeedPacket", "Award"）
//
// 返回：
//   - offsetX: X轴偏移量（应用于调用者期望的视觉中心X坐标）
//   - offsetY: Y轴偏移量（应用于调用者期望的视觉中心Y坐标）
//
// 示例：
//
//	offsetX, offsetY := GetParticleAnchorOffset("SeedPacket")
//	anchorX := visualCenterX + offsetX
//	anchorY := visualCenterY + offsetY
//	CreateParticleEffect(em, rm, "SeedPacket", anchorX, anchorY, ...)
func GetParticleAnchorOffset(effectName string) (offsetX, offsetY float64) {
	if anchor, exists := ParticleAnchorOffsets[effectName]; exists {
		return anchor.OffsetX, anchor.OffsetY
	}
	// 默认值：锚点 = 视觉中心（无偏移）
	return 0, 0
}
