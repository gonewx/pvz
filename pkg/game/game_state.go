package game

// GameState 存储全局游戏状态
// 这是一个单例，用于管理跨场景和跨系统的全局状态数据
type GameState struct {
	Sun int // 当前阳光数量
	// 未来可扩展：Level, Wave, Score 等
}

// 全局单例实例（这是架构规范允许的唯一全局变量）
var globalGameState *GameState

// GetGameState 返回全局 GameState 单例
// 使用延迟初始化模式，确保整个游戏生命周期只有一个实例
func GetGameState() *GameState {
	if globalGameState == nil {
		globalGameState = &GameState{
			Sun: 50, // 初始阳光值
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





