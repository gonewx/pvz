package config

// Story 11.4: 铺草皮粒子特效配置常量
//
// 这些常量定义了 SodRoll 粒子发射器相对于草皮卷动画的位置偏移
// 用于微调粒子效果的视觉表现

// SodRollParticleOffsetX 粒子发射器X轴偏移量（相对于草皮卷中心，世界坐标）
// 正值向右偏移，负值向左偏移
// 默认值：0（与草皮卷中心对齐）
const SodRollParticleOffsetX float64 = 40

// SodRollParticleOffsetY 粒子发射器Y轴偏移量（相对于草皮卷中心，世界坐标）
// 正值向下偏移，负值向上偏移
// 默认值：-30（向上偏移 30 像素，因为 SystemPosition 会在初始时添加 +30 的偏移）
// 这样应用 SystemPosition 后，发射器正好在草皮卷中心
const SodRollParticleOffsetY float64 = 30

// 注释说明：
// - 粒子发射器位置 = 草皮卷中心位置 + 偏移量
// - 根据视觉效果需要调整这些偏移量
// - SodRoll.xml 中的 SystemPosition 字段会自动处理发射器的移动动画
// - 这些偏移量仅影响发射器的起始位置
