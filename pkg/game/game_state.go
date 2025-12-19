package game

import (
	"fmt"
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/utils"
	"github.com/quasilyte/gdata/v2"
)

// SeedSelectionPhase 选卡阶段枚举
// Story 8.12: 定义选卡界面的不同阶段状态
type SeedSelectionPhase int

const (
	// SeedSelectionNone 不在选卡阶段（正常游戏流程）
	SeedSelectionNone SeedSelectionPhase = iota
	// SeedSelectionActive 选卡界面激活中（玩家正在选择植物）
	SeedSelectionActive
	// SeedSelectionConfirmed 选卡完成（玩家点击了"一起摇滚吧!"）
	SeedSelectionConfirmed
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

	// Story 8.12: 选卡系统状态
	SeedSelectionPhase SeedSelectionPhase // 选卡阶段状态
	SeedChooserPlants  []string           // 选卡界面当前选中的植物列表（临时状态）

	// Story 5.5: 关卡流程状态
	CurrentLevel          *config.LevelConfig // 当前关卡配置
	LevelTime             float64             // 关卡已进行时间（秒）
	CurrentWaveIndex      int                 // 当前波次索引（0表示第一波）
	SpawnedWaves          []bool              // 每一波是否已生成（用于避免重复生成）
	TotalZombiesInLevel   int                 // 关卡配置中的总僵尸数（用于胜利条件）
	TotalZombiesSpawned   int                 // 已激活的僵尸总数（用于计算场上僵尸数）
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

	// Story 8.6: 关卡进度保存系统
	saveManager *SaveManager // 保存管理器（关卡进度、植物解锁、工具解锁）

	// Story 10.1: 暂停菜单系统
	IsPaused bool // 游戏是否暂停

	// Story 10.8: 阳光计数器闪烁反馈
	SunFlashTimer    float64 // 闪烁剩余时间（秒），值 > 0 时触发闪烁动画，0 时停止
	SunFlashCycle    float64 // 闪烁周期（秒），红色 ↔ 黑色切换周期，默认 0.3 秒
	SunFlashDuration float64 // 闪烁总持续时间（秒），默认 1.0 秒（约 3 次完整闪烁）

	// Story 17.1: 难度引擎数据
	TotalCompletedFlags int // 已完成的旗帜总数（跨关卡累计）
	WavesPerRound       int // 每轮波次数（默认20）

	// Story 20.1: 跨平台存储管理器
	// 使用 gdata 库实现跨平台数据存储（桌面端、移动端、WASM）
	// 如果初始化失败，gdataManager 为 nil，游戏仍可运行但无法持久化数据
	gdataManager *gdata.Manager

	// Story 20.2: 设置管理器
	// 管理游戏设置（音量、全屏等），使用 gdata 进行跨平台存储
	// 如果初始化失败，settingsManager 为 nil，游戏使用默认设置
	settingsManager *SettingsManager

	// 音频管理器
	// 管理游戏中所有音效和背景音乐的播放，支持音量控制
	// 需要通过 SetAudioManager 设置，由持有 ResourceManager 的组件初始化
	audioManager *AudioManager

	// Story 8.14: 来信奖励淡入淡出状态
	// 用于 fadingOut/fadingIn 阶段的黑色遮罩透明度（0.0 - 1.0）
	NoteFadeAlpha float32
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

		// Story 20.1: 初始化 gdata Manager（跨平台存储）
		// 注意：必须在 SaveManager 之前初始化，因为 SaveManager 需要 gdataManager

		// Story 21.4: 确保 Android 存储目录存在
		// gdata 在 Android 上不会自动创建存储目录，需要预先创建
		if err := utils.EnsureStorageDir(); err != nil {
			log.Printf("[GameState] Warning: Failed to ensure storage directory: %v", err)
		}

		log.Printf("[GameState] Initializing gdata Manager...")
		gdataManager, err := gdata.Open(gdata.Config{
			AppName: "pvz_newx",
		})
		if err != nil {
			log.Printf("[GameState] Warning: Failed to initialize gdata Manager: %v", err)
			// 降级方案：gdataManager 为 nil，游戏继续运行
			gdataManager = nil
		} else {
			// 记录存储路径（用于调试）
			testPath := gdataManager.ObjectPropPath("_test", "_test")
			log.Printf("[GameState] gdata initialized successfully, storage path: %s", testPath)
		}

		// Story 20.3: 初始化保存管理器（必须在 gdataManager 之后）
		saveManager, err := NewSaveManager(gdataManager)
		if err != nil {
			log.Printf("[GameState] Warning: Failed to initialize SaveManager: %v", err)
			// 如果保存管理器初始化失败，使用 nil（游戏可以运行，但无法保存进度）
			saveManager = nil
		}

		// Story 20.2: 初始化 SettingsManager（必须在 gdataManager 之后）
		var settingsManager *SettingsManager
		settingsManager, err = NewSettingsManager(gdataManager)
		if err != nil {
			log.Printf("[GameState] Warning: Failed to initialize SettingsManager: %v", err)
			// 降级方案：settingsManager 为 nil，游戏使用默认设置
			settingsManager = nil
		}

		globalGameState = &GameState{
			Sun:                50, // 默认阳光值（加载关卡后会被 levelConfig.InitialSun 覆盖）
			plantUnlockManager: NewPlantUnlockManager(),
			SelectedPlants:     []string{},
			LawnStrings:        lawnStrings,
			saveManager:        saveManager,
			// Story 10.8: 初始化闪烁参数
			SunFlashCycle:    0.3,
			SunFlashDuration: 1.0,
			// Story 17.1: 初始化难度引擎数据
			TotalCompletedFlags: 0,
			WavesPerRound:       20, // 默认每轮20波
			// Story 20.1: 跨平台存储管理器
			gdataManager: gdataManager,
			// Story 20.2: 设置管理器
			settingsManager: settingsManager,
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

// GetNextLevelID 获取下一关的关卡ID
// 如果没有下一关，返回空字符串
func (gs *GameState) GetNextLevelID() string {
	if gs.CurrentLevel == nil {
		return ""
	}

	// 解析当前关卡ID (格式: "1-1", "1-2", etc.)
	var chapter, level int
	_, err := fmt.Sscanf(gs.CurrentLevel.ID, "%d-%d", &chapter, &level)
	if err != nil {
		log.Printf("[GameState] Failed to parse level ID: %s", gs.CurrentLevel.ID)
		return ""
	}

	// 简单递增关卡号
	nextLevel := level + 1
	nextLevelID := fmt.Sprintf("%d-%d", chapter, nextLevel)

	// 检查下一关配置文件是否存在
	// 如果不存在，返回空字符串表示没有下一关
	nextLevelPath := fmt.Sprintf("data/levels/level-%s.yaml", nextLevelID)
	if _, err := config.LoadLevelConfig(nextLevelPath); err != nil {
		log.Printf("[GameState] No next level found: %s (error: %v)", nextLevelID, err)
		return "" // 没有下一关了
	}

	return nextLevelID
}

// LoadLevel 加载关卡配置
// 初始化关卡状态，重置所有关卡相关的计数器和标志
func (gs *GameState) LoadLevel(levelConfig *config.LevelConfig) {
	gs.CurrentLevel = levelConfig
	gs.LevelTime = 0
	gs.CurrentWaveIndex = 0
	gs.SpawnedWaves = make([]bool, len(levelConfig.Waves))

	// 计算关卡总僵尸数（从配置文件读取所有波次的僵尸数量）
	// 用于胜利条件判断：必须消灭配置中的所有僵尸才算胜利
	totalZombies := 0
	for _, wave := range levelConfig.Waves {
		// 新格式：使用 Zombies 字段
		for _, zombieGroup := range wave.Zombies {
			totalZombies += zombieGroup.Count
		}
		// 旧格式：使用 OldZombies 字段（向后兼容）
		for _, zombieSpawn := range wave.OldZombies {
			totalZombies += zombieSpawn.Count
		}
	}
	gs.TotalZombiesInLevel = totalZombies // 关卡配置中的总僵尸数（固定不变）
	gs.TotalZombiesSpawned = 0            // 已激活的僵尸数（激活时增加）
	log.Printf("[GameState] LoadLevel: %s, Total zombies in config: %d", levelConfig.ID, totalZombies)

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
// Story 17.6: 波次计时由 WaveTimingSystem 自动管理
// 返回 -1 表示没有波次需要生成（等待中或全部生成完毕）
func (gs *GameState) GetCurrentWave() int {
	if gs.CurrentLevel == nil {
		return -1
	}

	// Story 17.6: 波次计时由 WaveTimingSystem 管理
	// 此方法仅作为后备逻辑，当 WaveTimingSystem 未启用时使用
	// 简化逻辑：场上无僵尸且有未生成的波次时，立即触发下一波

	// 获取当前场上的僵尸数量（已生成 - 已消灭）
	zombiesOnField := gs.TotalZombiesSpawned - gs.ZombiesKilled

	// DEBUG: 输出状态
	if zombiesOnField == 0 && gs.CurrentWaveIndex < len(gs.CurrentLevel.Waves) {
		log.Printf("[GetCurrentWave] 🔍 DEBUG: WaveIndex=%d, ZombiesOnField=%d, IsWaiting=%v",
			gs.CurrentWaveIndex, zombiesOnField, gs.IsWaitingForNextWave)
	}

	// 第一波：立即触发
	if gs.CurrentWaveIndex == 0 && !gs.SpawnedWaves[0] {
		log.Printf("[GetCurrentWave] ✅ 第一波立即触发")
		return 0
	}

	// 后续波次：场上无僵尸时触发下一波
	if zombiesOnField == 0 && gs.CurrentWaveIndex < len(gs.CurrentLevel.Waves) {
		currentWaveIndex := gs.CurrentWaveIndex
		if currentWaveIndex < len(gs.CurrentLevel.Waves) && !gs.SpawnedWaves[currentWaveIndex] {
			log.Printf("[GetCurrentWave] ✅ 波次 %d 触发（场上无僵尸）", currentWaveIndex+1)
			return currentWaveIndex
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

// IncrementZombiesSpawned 增加已激活僵尸计数
// 在僵尸激活时调用（用于计算场上僵尸数 = TotalZombiesSpawned - ZombiesKilled）
func (gs *GameState) IncrementZombiesSpawned(count int) {
	gs.TotalZombiesSpawned += count
	log.Printf("[GameState] IncrementZombiesSpawned: +%d, Activated=%d, Total=%d, Killed=%d, OnField=%d",
		count, gs.TotalZombiesSpawned, gs.TotalZombiesInLevel, gs.ZombiesKilled, gs.TotalZombiesSpawned-gs.ZombiesKilled)
}

// IncrementZombiesKilled 增加已消灭僵尸计数
// 在僵尸死亡时调用
func (gs *GameState) IncrementZombiesKilled() {
	gs.ZombiesKilled++
	zombiesOnField := gs.TotalZombiesSpawned - gs.ZombiesKilled
	log.Printf("[GameState] IncrementZombiesKilled: Killed=%d/%d (config), Activated=%d, OnField=%d",
		gs.ZombiesKilled, gs.TotalZombiesInLevel, gs.TotalZombiesSpawned, zombiesOnField)
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
	for i, spawned := range gs.SpawnedWaves {
		if !spawned {
			allWavesSpawned = false
			log.Printf("[GameState] CheckVictory: wave %d not spawned (SpawnedWaves=%v)", i, gs.SpawnedWaves)
			break
		}
	}

	// 胜利条件：
	// 1. 所有波次已生成（allWavesSpawned = true）
	// 2. 场上没有存活的僵尸（TotalZombiesSpawned - ZombiesKilled <= 0）
	// 注意：必须消灭所有已激活的僵尸，而不是配置中的僵尸数量
	// 使用 <= 0 而非 == 0，处理可能的计数误差（如重复计数导致 ZombiesKilled > TotalZombiesSpawned）
	zombiesOnField := gs.TotalZombiesSpawned - gs.ZombiesKilled
	result := allWavesSpawned && gs.TotalZombiesSpawned > 0 && zombiesOnField <= 0

	// 调试日志：当接近胜利条件时输出
	if allWavesSpawned || zombiesOnField <= 2 {
		log.Printf("[GameState] CheckVictory: allWavesSpawned=%v, ZombiesKilled=%d, TotalZombiesSpawned=%d, ZombiesOnField=%d, result=%v",
			allWavesSpawned, gs.ZombiesKilled, gs.TotalZombiesSpawned, zombiesOnField, result)
	}

	return result
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

// SetPaused 设置暂停状态
// Story 10.1: 用于控制游戏暂停/恢复
func (gs *GameState) SetPaused(paused bool) {
	gs.IsPaused = paused
}

// TogglePause 切换暂停状态
// Story 10.1: ESC 快捷键使用
func (gs *GameState) TogglePause() {
	gs.IsPaused = !gs.IsPaused
}

// TriggerSunFlash 触发阳光计数器闪烁
// Story 10.8: 当玩家点击阳光不足的卡片时调用
func (gs *GameState) TriggerSunFlash() {
	gs.SunFlashTimer = gs.SunFlashDuration
}

// UpdateSunFlash 更新闪烁计时器
// Story 10.8: 在每帧更新中调用，递减闪烁计时器
func (gs *GameState) UpdateSunFlash(deltaTime float64) {
	if gs.SunFlashTimer > 0 {
		gs.SunFlashTimer -= deltaTime
		if gs.SunFlashTimer < 0 {
			gs.SunFlashTimer = 0
		}
	}
}

// ========================================
// Story 8.6: 关卡进度保存系统
// ========================================

// GetSaveManager 获取保存管理器
//
// 返回：
//   - *SaveManager: 保存管理器实例，如果未初始化返回 nil
func (gs *GameState) GetSaveManager() *SaveManager {
	return gs.saveManager
}

// GetGdataManager 获取 gdata 跨平台存储管理器
//
// Story 20.1: 返回 gdata.Manager 实例，用于跨平台数据存储
// 如果初始化失败，返回 nil（调用方需检查）
//
// 返回：
//   - *gdata.Manager: gdata 管理器实例，如果未初始化返回 nil
func (gs *GameState) GetGdataManager() *gdata.Manager {
	return gs.gdataManager
}

// GetSettingsManager 获取设置管理器
//
// Story 20.2: 返回 SettingsManager 实例，用于管理游戏设置
// 如果初始化失败，返回 nil（调用方需检查）
//
// 返回：
//   - *SettingsManager: 设置管理器实例，如果未初始化返回 nil
func (gs *GameState) GetSettingsManager() *SettingsManager {
	return gs.settingsManager
}

// SaveProgress 保存当前游戏进度
//
// 在关卡完成时调用，保存关卡进度、解锁植物和工具
//
// 返回：
//   - error: 如果保存失败返回错误
func (gs *GameState) SaveProgress() error {
	if gs.saveManager == nil {
		return fmt.Errorf("save manager not initialized")
	}

	// 保存到文件
	return gs.saveManager.Save()
}

// CompleteLevel 完成关卡，更新进度并保存
//
// Story 8.6: 关卡完成时调用
//
// 参数：
//   - levelID: 完成的关卡ID，如 "1-2"
//   - rewardPlant: 奖励的植物ID（可为空）
//   - unlockTools: 解锁的工具列表（可为空）
//
// 返回：
//   - error: 如果保存失败返回错误
func (gs *GameState) CompleteLevel(levelID string, rewardPlant string, unlockTools []string) error {
	if gs.saveManager == nil {
		return fmt.Errorf("save manager not initialized")
	}

	// 更新最高完成关卡
	gs.saveManager.SetHighestLevel(levelID)

	// 解锁奖励植物
	if rewardPlant != "" {
		gs.saveManager.UnlockPlant(rewardPlant)
		// 同时更新 PlantUnlockManager
		if gs.plantUnlockManager != nil {
			gs.plantUnlockManager.UnlockPlant(rewardPlant)
		}
		log.Printf("[GameState] Unlocked plant: %s (reward for completing %s)", rewardPlant, levelID)
	}

	// 解锁工具
	for _, tool := range unlockTools {
		gs.saveManager.UnlockTool(tool)
		log.Printf("[GameState] Unlocked tool: %s (reward for completing %s)", tool, levelID)
	}

	// 保存进度
	return gs.SaveProgress()
}

// IsToolUnlocked 检查工具是否已解锁
//
// 参数：
//   - toolID: 工具ID，如 "shovel"
//
// 返回：
//   - bool: true 表示已解锁，false 表示未解锁
func (gs *GameState) IsToolUnlocked(toolID string) bool {
	if gs.saveManager == nil {
		return false
	}
	return gs.saveManager.IsToolUnlocked(toolID)
}

// ========================================
// Story 17.1: 难度引擎辅助方法
// ========================================

// GetCurrentRoundNumber 获取当前轮数
// 公式: RoundNumber = TotalCompletedFlags / 2 - 1
//
// 返回:
//   - int: 当前轮数（可能为负数，表示一周目早期关卡）
func (gs *GameState) GetCurrentRoundNumber() int {
	return gs.TotalCompletedFlags/2 - 1
}

// GetWaveCapacity 获取指定波次的级别容量上限
// 公式: CapacityCap = int(int((CurrentWaveNum + RoundNumber * WavesPerRound) * 0.8) / 2) + 1
// 旗帜波（大波）容量 × 2.5 并向零取整
//
// 参数:
//   - waveNum: 当前波次号（从1开始）
//   - isFlagWave: 是否为旗帜波（大波）
//
// 返回:
//   - int: 级别容量上限
func (gs *GameState) GetWaveCapacity(waveNum int, isFlagWave bool) int {
	roundNumber := gs.GetCurrentRoundNumber()
	wavesPerRound := gs.WavesPerRound
	if wavesPerRound <= 0 {
		wavesPerRound = 20 // 默认值
	}

	base := int(int(float64(waveNum+roundNumber*wavesPerRound)*0.8)/2) + 1
	if isFlagWave {
		return int(float64(base) * 2.5)
	}
	return base
}

// IncrementCompletedFlags 增加已完成旗帜计数
// 在完成关卡旗帜波时调用
//
// 参数:
//   - count: 增加的旗帜数量
func (gs *GameState) IncrementCompletedFlags(count int) {
	gs.TotalCompletedFlags += count
	log.Printf("[GameState] IncrementCompletedFlags: +%d, Total=%d, RoundNumber=%d",
		count, gs.TotalCompletedFlags, gs.GetCurrentRoundNumber())
}

// IsSecondPlaythrough 检查是否为二周目
// 一周目完成需要约50旗（25个常规关卡 × 2旗/关卡）
//
// 返回:
//   - bool: true 表示二周目，false 表示一周目
func (gs *GameState) IsSecondPlaythrough() bool {
	return gs.TotalCompletedFlags >= 50
}

// ========================================
// 音频管理器
// ========================================

// SetAudioManager 设置音频管理器
// 由持有 ResourceManager 的场景初始化后设置
//
// 参数：
//   - am: AudioManager 实例
func (gs *GameState) SetAudioManager(am *AudioManager) {
	gs.audioManager = am
}

// GetAudioManager 获取音频管理器
//
// 返回：
//   - *AudioManager: 音频管理器实例，如果未设置返回 nil
func (gs *GameState) GetAudioManager() *AudioManager {
	return gs.audioManager
}

// ========================================
// Story 8.12: 选卡系统
// ========================================

// EnterSeedSelection 进入选卡阶段
// 在开场动画僵尸预览结束后调用
func (gs *GameState) EnterSeedSelection() {
	gs.SeedSelectionPhase = SeedSelectionActive
	gs.SeedChooserPlants = []string{}
	log.Printf("[GameState] 进入选卡阶段")
}

// ExitSeedSelection 退出选卡阶段
// 玩家点击"一起摇滚吧!"按钮后调用
func (gs *GameState) ExitSeedSelection() {
	gs.SeedSelectionPhase = SeedSelectionConfirmed
	// 将选卡结果复制到 SelectedPlants（用于 GameScene 初始化卡槽）
	gs.SelectedPlants = make([]string, len(gs.SeedChooserPlants))
	copy(gs.SelectedPlants, gs.SeedChooserPlants)
	log.Printf("[GameState] 退出选卡阶段，已选植物: %v", gs.SelectedPlants)
}

// CompleteSeedSelection 完成选卡流程
// 镜头左移完成后调用，重置选卡状态
func (gs *GameState) CompleteSeedSelection() {
	gs.SeedSelectionPhase = SeedSelectionNone
	gs.SeedChooserPlants = nil
	log.Printf("[GameState] 选卡流程完成")
}

// IsSeedSelectionActive 检查是否处于选卡阶段
func (gs *GameState) IsSeedSelectionActive() bool {
	return gs.SeedSelectionPhase == SeedSelectionActive
}

// IsSeedSelectionConfirmed 检查选卡是否已确认
func (gs *GameState) IsSeedSelectionConfirmed() bool {
	return gs.SeedSelectionPhase == SeedSelectionConfirmed
}

// AddSeedChooserPlant 在选卡界面添加一个植物
// 返回 true 表示添加成功，false 表示卡槽已满或植物已存在
func (gs *GameState) AddSeedChooserPlant(plantID string) bool {
	// 检查是否已满（最多6个）
	if len(gs.SeedChooserPlants) >= 6 {
		return false
	}
	// 检查是否已存在
	for _, p := range gs.SeedChooserPlants {
		if p == plantID {
			return false
		}
	}
	gs.SeedChooserPlants = append(gs.SeedChooserPlants, plantID)
	log.Printf("[GameState] 选卡添加: %s, 当前选择: %v", plantID, gs.SeedChooserPlants)
	return true
}

// RemoveSeedChooserPlant 从选卡界面移除一个植物
// 返回 true 表示移除成功
func (gs *GameState) RemoveSeedChooserPlant(plantID string) bool {
	for i, p := range gs.SeedChooserPlants {
		if p == plantID {
			gs.SeedChooserPlants = append(gs.SeedChooserPlants[:i], gs.SeedChooserPlants[i+1:]...)
			log.Printf("[GameState] 选卡移除: %s, 当前选择: %v", plantID, gs.SeedChooserPlants)
			return true
		}
	}
	return false
}

// GetSeedChooserPlants 获取选卡界面当前选中的植物列表
func (gs *GameState) GetSeedChooserPlants() []string {
	return gs.SeedChooserPlants
}

// IsSeedChooserFull 检查选卡槽位是否已满
func (gs *GameState) IsSeedChooserFull() bool {
	return len(gs.SeedChooserPlants) >= 6
}

// IsSeedChooserPlantSelected 检查某个植物是否已被选中
func (gs *GameState) IsSeedChooserPlantSelected(plantID string) bool {
	for _, p := range gs.SeedChooserPlants {
		if p == plantID {
			return true
		}
	}
	return false
}
