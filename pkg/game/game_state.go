package game

import "github.com/decker502/pvz/pkg/components"

// GameState 存储全局游戏状态
// 这是一个单例，用于管理跨场景和跨系统的全局状态数据
type GameState struct {
	Sun int // 当前阳光数量

	// Story 3.2: 种植模式状态
	IsPlantingMode    bool                 // 是否处于种植模式
	SelectedPlantType components.PlantType // 当前选择的植物类型

	// 摄像机位置（世界坐标系统）
	CameraX float64 // 摄像机X位置，用于世界坐标和屏幕坐标转换

	// 未来可扩展：Level, Wave, Score 等
}

// 全局单例实例（这是架构规范允许的唯一全局变量）
var globalGameState *GameState

// GetGameState 返回全局 GameState 单例
// 使用延迟初始化模式，确保整个游戏生命周期只有一个实例
func GetGameState() *GameState {
	if globalGameState == nil {
		globalGameState = &GameState{
			Sun: 500, // 临时增加初始阳光用于测试（原版是50）
		}
	}
	return globalGameState
}

// AddSun 增加阳光，带上限检查
// 阳光上限为9990（原版游戏显示上限）
func (gs *GameState) AddSun(amount int) {
	gs.Sun += amount
	if gs.Sun > 9990 {
		gs.Sun = 9990 // 原版游戏阳光上限
	}
}

// SpendSun 扣除阳光，如果阳光不足返回 false
// 只有当阳光充足时才会扣除，否则返回false表示操作失败
func (gs *GameState) SpendSun(amount int) bool {
	if gs.Sun < amount {
		return false
	}
	gs.Sun -= amount
	return true
}

// GetSun 返回当前阳光值
func (gs *GameState) GetSun() int {
	return gs.Sun
}

// EnterPlantingMode 进入种植模式
// 设置游戏进入种植状态，并记录玩家选择的植物类型
func (gs *GameState) EnterPlantingMode(plantType components.PlantType) {
	gs.IsPlantingMode = true
	gs.SelectedPlantType = plantType
}

// ExitPlantingMode 退出种植模式
// 将游戏状态恢复到正常模式
func (gs *GameState) ExitPlantingMode() {
	gs.IsPlantingMode = false
}

// GetPlantingMode 获取当前种植模式状态
// 返回是否处于种植模式以及选择的植物类型
func (gs *GameState) GetPlantingMode() (bool, components.PlantType) {
	return gs.IsPlantingMode, gs.SelectedPlantType
}
