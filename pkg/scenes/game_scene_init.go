package scenes

import (
	"log"

	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/modules"
	"github.com/decker502/pvz/pkg/systems"
)

// initPlantCardSystems initializes the plant selection module.
// Story 3.1 架构优化：使用 PlantSelectionModule 统一管理所有选卡功能
// Story 8.3: 使用 PlantUnlockManager 统一管理植物可用性
//
// 重构说明：
//   - 旧方式：直接在 GameScene 中创建卡片实体和系统（分散）
//   - 新方式：使用 PlantSelectionModule 统一封装（内聚）
//
// 优点：
//   - 高内聚：所有选卡功能封装在单一模块中
//   - 低耦合：GameScene 只通过模块接口交互
//   - 可复用：模块可在不同场景（游戏中、选卡界面）使用
func (s *GameScene) initPlantCardSystems(rm *game.ResourceManager) {
	// Story 8.3: 获取当前关卡配置
	// 注意：CurrentLevel 从 GameState.LoadLevel() 加载
	levelConfig := s.gameState.CurrentLevel
	if levelConfig == nil {
		log.Printf("[GameScene] Warning: No level config found, using default plant cards")
		levelConfig = &config.LevelConfig{
			AvailablePlants: []string{"sunflower", "peashooter", "wallnut", "cherrybomb"},
		}
	}

	// 创建植物选择栏模块
	var err error
	s.plantSelectionModule, err = modules.NewPlantSelectionModule(
		s.entityManager,
		s.gameState,
		rm,
		s.reanimSystem,
		levelConfig,
		s.plantCardFont,
		config.SeedBankX,
		config.SeedBankY,
	)
	if err != nil {
		log.Printf("[GameScene] Error: Failed to initialize plant selection module: %v", err)
		// 游戏可在没有卡片的情况下运行（用于测试环境）
		return
	}

	log.Printf("[GameScene] Plant selection module initialized successfully")
}

// initMenuButton 初始化菜单按钮（ECS 架构）
// 创建可复用的三段式按钮实体，文字自动居中
func (s *GameScene) initMenuButton(rm *game.ResourceManager) {
	// 计算菜单按钮位置（右上角）
	buttonX := float64(WindowWidth) - config.MenuButtonOffsetFromRight
	buttonY := config.MenuButtonOffsetFromTop

	// 按钮中间部分宽度（根据文字长度）
	middleWidth := config.MenuButtonTextWidth + config.MenuButtonTextPadding*2

	// 创建菜单按钮实体
	var err error
	s.menuButtonEntity, err = entities.NewMenuButton(
		s.entityManager,
		rm,
		buttonX,
		buttonY,
		"菜单",                     // 按钮文字
		20.0,                     // 字体大小
		[4]uint8{0, 200, 0, 255}, // 绿色文字
		middleWidth,              // 中间宽度
		func() {
			// Story 10.1: 点击菜单按钮打开暂停菜单
			log.Printf("[GameScene] Menu button clicked! Opening pause menu...")
			if s.pauseMenuModule != nil {
				s.pauseMenuModule.Show()
			}
		},
	)

	if err != nil {
		log.Printf("[GameScene] Warning: Failed to create menu button: %v", err)
	} else {
		log.Printf("[GameScene] Menu button created successfully (Entity ID: %d)", s.menuButtonEntity)
	}
}

// initPauseMenuModule 初始化暂停菜单（ECS 架构）
// Story 10.1: 创建暂停菜单实体和三个按钮
// Story 18.2: 添加战斗存档保存回调
func (s *GameScene) initPauseMenuModule(rm *game.ResourceManager) {
	var err error
	s.pauseMenuModule, err = modules.NewPauseMenuModule(
		s.entityManager,
		s.gameState,
		rm,
		s.buttonSystem,
		s.buttonRenderSystem,
		WindowWidth,
		WindowHeight,
		modules.PauseMenuCallbacks{
			OnContinue: func() {
				s.gameState.SetPaused(false) // 恢复游戏
			},
			OnRestart: func() {
				// Story 18.2: 重新开始时删除战斗存档
				s.deleteBattleSave()
				// 重新加载当前关卡（使用当前关卡ID）
				currentLevelID := "1-1" // 默认值
				if s.gameState.CurrentLevel != nil {
					currentLevelID = s.gameState.CurrentLevel.ID
				}
				s.sceneManager.SwitchTo(NewGameScene(s.resourceManager, s.sceneManager, currentLevelID))
			},
			OnMainMenu: func() {
				// 返回主菜单
				s.sceneManager.SwitchTo(NewMainMenuScene(s.resourceManager, s.sceneManager))
			},
			OnPauseMusic: func() {
				// TODO: 暂停 BGM（当BGM系统实现后）
				// Story 17.6: 暂停波次计时
				if s.levelSystem != nil {
					s.levelSystem.PauseWaveTiming()
				}
			},
			OnResumeMusic: func() {
				// TODO: 恢复 BGM（当BGM系统实现后）
				// Story 17.6: 恢复波次计时
				if s.levelSystem != nil {
					s.levelSystem.ResumeWaveTiming()
				}
			},
			// Story 18.2: 保存战斗状态回调
			OnSaveBattle: func() {
				s.saveBattleState()
			},
		},
	)
	if err != nil {
		log.Printf("[GameScene] Warning: Failed to initialize pause menu module: %v", err)
	} else {
		log.Printf("[GameScene] Pause menu module initialized")
	}
}

// initProgressBar 初始化关卡进度条（Story 11.2）
// 创建进度条实体和渲染系统，关联到 LevelSystem
func (s *GameScene) initProgressBar(rm *game.ResourceManager) {
	// 加载字体（用于关卡文本）
	font, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.LevelTextFontSize)
	if err != nil {
		log.Printf("[GameScene] ERROR: Failed to load progress bar font: %v", err)
		return
	}
	log.Printf("[GameScene] Loaded progress bar font: SimHei.ttf (%.0fpx)", config.LevelTextFontSize)

	// 创建进度条渲染系统
	s.levelProgressBarRenderSystem = systems.NewLevelProgressBarRenderSystem(s.entityManager, font)
	log.Printf("[GameScene] Created progress bar render system")

	// 创建进度条实体（位置会在渲染时根据右对齐动态计算）
	progressBarEntity, err := entities.NewLevelProgressBarEntity(
		s.entityManager,
		rm,
	)
	if err != nil {
		log.Printf("[GameScene] ERROR: Failed to create progress bar entity: %v", err)
		return
	}

	s.levelProgressBarEntity = progressBarEntity

	// 关联到 LevelSystem（让 LevelSystem 初始化进度条数据）
	s.levelSystem.SetProgressBarEntity(progressBarEntity)

	log.Printf("[GameScene] Level progress bar initialized (Entity ID: %d)", progressBarEntity)
}

// initLawnmowers 初始化除草车实体
// Story 10.2: 在每个启用的行上创建一台除草车
//
// 除草车是每行的最后防线：
// - 僵尸到达左侧边界时自动触发
// - 沿该行向右快速移动，消灭路径上的所有僵尸
// - 每行只有一台除草车，使用后不可恢复
func (s *GameScene) initLawnmowers() {
	if s.gameState.CurrentLevel == nil {
		log.Printf("[GameScene] Warning: No current level, skipping lawnmower initialization")
		return
	}

	// 获取关卡启用的行
	enabledLanes := s.gameState.CurrentLevel.EnabledLanes
	if len(enabledLanes) == 0 {
		// 如果未配置EnabledLanes，默认启用所有5行
		enabledLanes = []int{1, 2, 3, 4, 5}
	}

	// 为每个启用的行创建除草车
	for _, lane := range enabledLanes {
		lawnmowerID, err := entities.NewLawnmowerEntity(
			s.entityManager,
			s.resourceManager,
			lane,
		)

		if err != nil {
			log.Printf("[GameScene] ERROR: Failed to create lawnmower for lane %d: %v", lane, err)
			continue
		}

		log.Printf("[GameScene] Created lawnmower for lane %d (Entity ID: %d)", lane, lawnmowerID)
	}

	log.Printf("[GameScene] Initialized %d lawnmowers for enabled lanes: %v", len(enabledLanes), enabledLanes)
}

// saveBattleState 保存当前战斗状态
//
// Story 18.2: 战斗存档保存触发
//
// 调用时机：
//   - 玩家点击暂停菜单的"返回主菜单"按钮
//
// 保存内容：
//   - 关卡ID、时间、阳光
//   - 波次进度
//   - 所有实体状态（植物、僵尸、子弹、阳光、除草车）
func (s *GameScene) saveBattleState() {
	// 获取当前用户
	saveManager := s.gameState.GetSaveManager()
	currentUser := saveManager.GetCurrentUser()
	if currentUser == "" {
		log.Printf("[GameScene] Warning: No current user, cannot save battle state")
		return
	}

	// 获取存档文件路径
	battleSavePath := saveManager.GetBattleSavePath(currentUser)

	// 创建序列化器并保存
	serializer := game.NewBattleSerializer()
	if err := serializer.SaveBattle(s.entityManager, s.gameState, battleSavePath); err != nil {
		log.Printf("[GameScene] ERROR: Failed to save battle state: %v", err)
		return
	}

	log.Printf("[GameScene] Battle state saved successfully to %s", battleSavePath)
}

// deleteBattleSave 删除当前用户的战斗存档
//
// Story 18.2: 重新开始时删除存档
//
// 调用时机：
//   - 玩家点击暂停菜单的"重新开始"按钮
//   - 游戏胜利后（进入下一关）
//   - 从存档恢复后
func (s *GameScene) deleteBattleSave() {
	// 获取当前用户
	saveManager := s.gameState.GetSaveManager()
	currentUser := saveManager.GetCurrentUser()
	if currentUser == "" {
		log.Printf("[GameScene] Warning: No current user, cannot delete battle save")
		return
	}

	// 删除存档
	if err := saveManager.DeleteBattleSave(currentUser); err != nil {
		log.Printf("[GameScene] ERROR: Failed to delete battle save: %v", err)
		return
	}

	log.Printf("[GameScene] Battle save deleted for user: %s", currentUser)
}

// restoreBattleState 从战斗存档恢复战斗状态
//
// Story 18.2: 战斗存档自动加载
//
// 恢复流程：
//  1. 获取当前用户的存档路径
//  2. 使用 BattleSerializer.LoadBattle() 加载存档数据
//  3. 恢复游戏状态（阳光、波次进度）
//  4. 成功后删除存档（避免重复加载）
//  5. 失败时记录日志，继续正常游戏
//
// 注意：实体恢复（植物、僵尸等）将在 Story 18.3 中实现
func (s *GameScene) restoreBattleState() {
	saveManager := s.gameState.GetSaveManager()
	currentUser := saveManager.GetCurrentUser()
	if currentUser == "" {
		log.Printf("[GameScene] Warning: No current user, cannot restore battle state")
		return
	}

	// 检查是否有存档
	if !saveManager.HasBattleSave(currentUser) {
		log.Printf("[GameScene] No battle save found for user: %s", currentUser)
		return
	}

	// 获取存档路径
	battleSavePath := saveManager.GetBattleSavePath(currentUser)
	log.Printf("[GameScene] 开始从战斗存档恢复: %s", battleSavePath)

	// 创建序列化器并加载
	serializer := game.NewBattleSerializer()
	saveData, err := serializer.LoadBattle(battleSavePath)
	if err != nil {
		log.Printf("[GameScene] ERROR: Failed to load battle data: %v", err)
		log.Printf("[GameScene] 继续正常游戏...")
		return
	}

	// 恢复游戏状态
	s.gameState.Sun = saveData.Sun
	s.gameState.LevelTime = saveData.LevelTime
	s.gameState.CurrentWaveIndex = saveData.CurrentWaveIndex
	if len(saveData.SpawnedWaves) > 0 {
		s.gameState.SpawnedWaves = make([]bool, len(saveData.SpawnedWaves))
		copy(s.gameState.SpawnedWaves, saveData.SpawnedWaves)
	}
	s.gameState.TotalZombiesSpawned = saveData.TotalZombiesSpawned
	s.gameState.ZombiesKilled = saveData.ZombiesKilled

	log.Printf("[GameScene] 游戏状态已恢复: Sun=%d, Wave=%d, Time=%.1f",
		s.gameState.Sun, s.gameState.CurrentWaveIndex, s.gameState.LevelTime)

	// TODO: Story 18.3 将实现实体恢复（植物、僵尸等）
	log.Printf("[GameScene] 实体恢复: Plants=%d, Zombies=%d (待 Story 18.3 实现)",
		len(saveData.Plants), len(saveData.Zombies))

	// 跳过开场动画
	s.isIntroAnimPlaying = false
	s.cameraX = config.GameCameraX
	if s.cameraSystem != nil {
		s.gameState.CameraX = config.GameCameraX
	}

	// 跳过铺草皮动画
	s.soddingAnimStarted = true

	// 恢复后删除存档
	if err := saveManager.DeleteBattleSave(currentUser); err != nil {
		log.Printf("[GameScene] Warning: Failed to delete battle save after restore: %v", err)
	} else {
		log.Printf("[GameScene] 存档已删除（恢复成功后）")
	}

	log.Printf("[GameScene] 战斗状态恢复完成!")
}
