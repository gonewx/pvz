package config

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

	PlantOffsetY = -10.0
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
