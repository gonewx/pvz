package components

// ScaleComponent 存储实体级别的缩放因子
// 用于在渲染时对整个实体进行缩放（如阳光收集时的缩小动画）
//
// 与 ReanimComponent 的 Frame.ScaleX/ScaleY 不同：
// - Frame.ScaleX/ScaleY 是动画数据（Reanim 文件定义）
// - ScaleComponent 是实体级别的额外缩放（叠加在动画缩放之上）
//
// 最终缩放 = Frame.ScaleX * ScaleComponent.ScaleX
type ScaleComponent struct {
	// ScaleX X轴缩放因子（1.0 = 原始大小，0.5 = 50%，2.0 = 200%）
	ScaleX float64

	// ScaleY Y轴缩放因子（1.0 = 原始大小，0.5 = 50%，2.0 = 200%）
	ScaleY float64
}
