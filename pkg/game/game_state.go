package game

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
)

// GameState 存储全局游戏状态
// 这是一个单例，用于管理跨场景和跨系统的全局状态数据
type GameState struct {
	Sun int // 当前阳光数量

	// Story 3.2: 种植模式状态
	IsPlantingMode    bool                 // 是否处于种植模式
	SelectedPlantType components.PlantType // 当前选择的植物类型

	// 摄像机位置（世界坐标系统）
	CameraX float64 // 摄像机X位置，用于世界坐标和屏幕坐标转换

	// Story 5.5: 关卡流程状态
	CurrentLevel          *config.LevelConfig // 当前关卡配置
	LevelTime             float64             // 关卡已进行时间（秒）
	CurrentWaveIndex      int                 // 当前波次索引（0表示第一波）
	SpawnedWaves          []bool              // 每一波是否已生成（用于避免重复生成）
	TotalZombiesSpawned   int                 // 已生成的僵尸总数
	ZombiesKilled         int                 // 已消灭的僵尸数量
	LastWaveCompletedTime float64             // 上一波完成时间（用于计算延迟）
	IsWaitingForNextWave  bool                // 是否正在等待下一波（延迟中）
	IsLevelComplete       bool                // 关卡是否完成
	IsGameOver            bool                // 游戏是否结束（胜利或失败）
	GameResult            string              // 游戏结果："win", "lose", "" (进行中)
	ShowingFinalWave      bool                // 是否正在显示最后一波警告动画

	// Story 8.1: 植物解锁和选卡状态
	plantUnlockManager *PlantUnlockManager // 植物解锁管理器
	SelectedPlants     []string            // 选卡界面选中的植物列表（传递给 GameScene）

	// Story 8.2: 教学系统
	LawnStrings *LawnStrings // 游戏文本字符串管理器（从 LawnStrings.txt 加载）
}

// 全局单例实例（这是架构规范允许的唯一全局变量）
var globalGameState *GameState

// GetGameState 返回全局 GameState 单例
// 使用延迟初始化模式，确保整个游戏生命周期只有一个实例
func GetGameState() *GameState {
	if globalGameState == nil {
		// 加载 LawnStrings.txt（如果加载失败，使用 nil，GetString 会返回 [KEY]）
		lawnStrings, err := NewLawnStrings("assets/properties/LawnStrings.txt")
		if err != nil {
			// 日志记录错误，但不阻止游戏启动（教学文本会显示为 [KEY]）
			// 在生产环境中应该有更好的错误处理
			lawnStrings = nil
		}

		globalGameState = &GameState{
			Sun:                50, // 默认阳光值（加载关卡后会被 levelConfig.InitialSun 覆盖）
			plantUnlockManager: NewPlantUnlockManager(),
			SelectedPlants:     []string{},
			LawnStrings:        lawnStrings,
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

// LoadLevel 加载关卡配置
// 初始化关卡状态，重置所有关卡相关的计数器和标志
func (gs *GameState) LoadLevel(levelConfig *config.LevelConfig) {
	gs.CurrentLevel = levelConfig
	gs.LevelTime = 0
	gs.CurrentWaveIndex = 0
	gs.SpawnedWaves = make([]bool, len(levelConfig.Waves))
	gs.TotalZombiesSpawned = 0
	gs.ZombiesKilled = 0
	gs.IsLevelComplete = false
	gs.IsGameOver = false
	gs.GameResult = ""

	// Story 8.2 QA改进：从关卡配置读取初始阳光值
	gs.Sun = levelConfig.InitialSun
}

// UpdateLevelTime 更新关卡时间
// 在每一帧中调用，累加经过的时间
func (gs *GameState) UpdateLevelTime(deltaTime float64) {
	gs.LevelTime += deltaTime
}

// GetCurrentWave 获取当前应该生成的波次索引
// 原版机制：上一波僵尸全部消灭后，等待 MinDelay 秒后触发下一波
// 返回 -1 表示没有波次需要生成（延迟中或全部生成完毕）
func (gs *GameState) GetCurrentWave() int {
	if gs.CurrentLevel == nil {
		return -1
	}

	// 获取当前场上的僵尸数量（已生成 - 已消灭）
	zombiesOnField := gs.TotalZombiesSpawned - gs.ZombiesKilled

	// DEBUG: 输出状态
	if zombiesOnField == 0 && gs.CurrentWaveIndex < len(gs.CurrentLevel.Waves) {
		log.Printf("[GetCurrentWave] 🔍 DEBUG: WaveIndex=%d, ZombiesOnField=%d, IsWaiting=%v",
			gs.CurrentWaveIndex, zombiesOnField, gs.IsWaitingForNextWave)
	}

	// 第一波：立即触发（游戏开始时）
	if gs.CurrentWaveIndex == 0 && !gs.SpawnedWaves[0] {
		log.Printf("[GetCurrentWave] ✅ 第一波立即触发")
		return 0
	}

	// 后续波次：上一波消灭完毕后，等待 MinDelay 秒
	if zombiesOnField == 0 && gs.CurrentWaveIndex < len(gs.CurrentLevel.Waves) {
		// 检查是否已经标记为等待状态
		if !gs.IsWaitingForNextWave {
			// 第一次检测到场上无僵尸，开始等待
			gs.IsWaitingForNextWave = true
			gs.LastWaveCompletedTime = gs.LevelTime
			return -1 // 进入延迟等待
		}

		// 检查延迟时间是否已过
		currentWaveIndex := gs.CurrentWaveIndex
		if currentWaveIndex < len(gs.CurrentLevel.Waves) && !gs.SpawnedWaves[currentWaveIndex] {
			waveConfig := gs.CurrentLevel.Waves[currentWaveIndex]
			elapsedSinceCompletion := gs.LevelTime - gs.LastWaveCompletedTime

			if elapsedSinceCompletion >= waveConfig.MinDelay {
				// 延迟已过，触发下一波
				gs.IsWaitingForNextWave = false
				return currentWaveIndex
			}
		}
	}

	return -1 // 没有波次需要生成
}

// MarkWaveSpawned 标记波次已生成
// 用于防止同一波次被重复生成
func (gs *GameState) MarkWaveSpawned(waveIndex int) {
	if waveIndex >= 0 && waveIndex < len(gs.SpawnedWaves) {
		gs.SpawnedWaves[waveIndex] = true
		gs.CurrentWaveIndex = waveIndex + 1
	}
}

// IsWaveSpawned 检查波次是否已生成
// 返回 true 表示该波次已经生成过
func (gs *GameState) IsWaveSpawned(waveIndex int) bool {
	if waveIndex < 0 || waveIndex >= len(gs.SpawnedWaves) {
		return false
	}
	return gs.SpawnedWaves[waveIndex]
}

// IncrementZombiesSpawned 增加已生成僵尸计数
// 在僵尸生成时调用
func (gs *GameState) IncrementZombiesSpawned(count int) {
	gs.TotalZombiesSpawned += count
	log.Printf("[GameState] IncrementZombiesSpawned: +%d, Total=%d", count, gs.TotalZombiesSpawned)
}

// IncrementZombiesKilled 增加已消灭僵尸计数
// 在僵尸死亡时调用
func (gs *GameState) IncrementZombiesKilled() {
	gs.ZombiesKilled++
	zombiesOnField := gs.TotalZombiesSpawned - gs.ZombiesKilled
	log.Printf("[GameState] IncrementZombiesKilled: Killed=%d/%d, OnField=%d",
		gs.ZombiesKilled, gs.TotalZombiesSpawned, zombiesOnField)
}

// CheckVictory 检查是否达成胜利条件
// 胜利条件：所有波次已生成 且 所有僵尸已消灭
// 返回 true 表示玩家获胜
func (gs *GameState) CheckVictory() bool {
	if gs.CurrentLevel == nil {
		return false
	}

	// 检查所有波次是否已生成
	allWavesSpawned := true
	for _, spawned := range gs.SpawnedWaves {
		if !spawned {
			allWavesSpawned = false
			break
		}
	}

	// 胜利条件：所有波次已生成 且 已消灭的僵尸数量等于已生成的僵尸总数
	return allWavesSpawned && gs.ZombiesKilled >= gs.TotalZombiesSpawned && gs.TotalZombiesSpawned > 0
}

// SetGameResult 设置游戏结果
// result: "win" 表示胜利, "lose" 表示失败
// 同时会设置 IsGameOver 和 IsLevelComplete 标志
func (gs *GameState) SetGameResult(result string) {
	gs.GameResult = result
	gs.IsGameOver = true
	if result == "win" {
		gs.IsLevelComplete = true
	}
}

// GetLevelProgress 获取关卡进度信息
// 返回当前波次（从1开始）和总波次数
func (gs *GameState) GetLevelProgress() (currentWave int, totalWaves int) {
	if gs.CurrentLevel == nil {
		return 0, 0
	}
	return gs.CurrentWaveIndex, len(gs.CurrentLevel.Waves)
}

// GetPlantUnlockManager 获取植物解锁管理器
// 返回全局植物解锁管理器实例
//
// 返回:
//   - *PlantUnlockManager: 植物解锁管理器实例
func (gs *GameState) GetPlantUnlockManager() *PlantUnlockManager {
	return gs.plantUnlockManager
}

// SetSelectedPlants 设置选卡界面选中的植物列表
// 在选卡界面确认选择后调用，将选中植物保存到 GameState
//
// 参数:
//   - plants: 选中的植物ID列表
func (gs *GameState) SetSelectedPlants(plants []string) {
	gs.SelectedPlants = make([]string, len(plants))
	copy(gs.SelectedPlants, plants)
}

// GetSelectedPlants 获取选卡界面选中的植物列表
// 在 GameScene 初始化时调用，获取玩家选择的植物
//
// 返回:
//   - []string: 选中的植物ID列表
func (gs *GameState) GetSelectedPlants() []string {
	return gs.SelectedPlants
}
