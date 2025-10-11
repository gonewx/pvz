package components

// PlantComponent 标识实体为植物
// 包含植物类型和所在格子位置信息
//
// 此组件用于标记场景中已种植的植物实体，
// 并记录该植物在草坪网格中的位置
type PlantComponent struct {
	// PlantType 植物类型（向日葵、豌豆射手等）
	PlantType PlantType
	// GridRow 所在草坪行 (0-4, 从上到下)
	GridRow int
	// GridCol 所在草坪列 (0-8, 从左到右)
	GridCol int
}
