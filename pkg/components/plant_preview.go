package components

// PlantPreviewComponent 标记实体为植物预览（跟随鼠标的半透明植物图像）
// 用于在玩家选择植物卡片后，显示即将种植的植物的半透明预览效果
// 此组件与 PositionComponent 和 SpriteComponent 配合使用
type PlantPreviewComponent struct {
	// PlantType 预览的植物类型
	PlantType PlantType

	// Alpha 透明度 (0.0-1.0)，建议值为 0.5
	// 0.0 表示完全透明，1.0 表示完全不透明
	Alpha float64
}


