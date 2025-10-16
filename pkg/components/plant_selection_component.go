package components

// PlantSelectionComponent 选卡界面植物选择状态组件
// 用于存储玩家在选卡界面中选择的植物列表和确认状态
// 这是一个纯数据组件，所有逻辑由 PlantSelectionSystem 处理
type PlantSelectionComponent struct {
	SelectedPlants []string // 已选择的植物ID列表（如 ["peashooter", "sunflower"]）
	MaxSlots       int      // 最大可选植物槽位数（通常为6-8个）
	IsConfirmed    bool     // 是否已确认选择（点击"开战"按钮后设为true）
}
