package scenes

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
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
// Story 20.5: 传递 SettingsManager 到暂停菜单
func (s *GameScene) initPauseMenuModule(rm *game.ResourceManager) {
	// Story 20.5: 从 GameState 获取 SettingsManager
	var settingsManager *game.SettingsManager
	if s.gameState != nil {
		settingsManager = s.gameState.GetSettingsManager()
	}

	var err error
	s.pauseMenuModule, err = modules.NewPauseMenuModule(
		s.entityManager,
		s.gameState,
		rm,
		s.buttonSystem,
		s.buttonRenderSystem,
		settingsManager, // Story 20.5: 传递 SettingsManager
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

// spawnPresetPlants 生成预设植物
// Story 19.4: 在关卡加载时根据配置生成预设植物
//
// 预设植物用于：
// - 铲子教学关卡 (Level 1-5) 中的初始植物
// - 特殊关卡的预设场景
//
// 处理内容：
// - 遍历 LevelConfig.PresetPlants 配置
// - 调用植物工厂函数创建实体
// - 更新草坪网格占用状态
// - 记录生成日志
func (s *GameScene) spawnPresetPlants() {
	if s.gameState.CurrentLevel == nil {
		log.Printf("[GameScene] spawnPresetPlants: No current level, skipping")
		return
	}

	presetPlants := s.gameState.CurrentLevel.PresetPlants
	if len(presetPlants) == 0 {
		log.Printf("[GameScene] spawnPresetPlants: No preset plants configured")
		return
	}

	log.Printf("[GameScene] Spawning %d preset plants...", len(presetPlants))

	spawnedCount := 0
	for i, preset := range presetPlants {
		// 将 1-based 配置坐标转换为 0-based 代码坐标
		col := preset.Col - 1
		row := preset.Row - 1

		// 验证坐标范围
		if row < 0 || row >= config.GridRows || col < 0 || col >= config.GridColumns {
			log.Printf("[GameScene] ERROR: Invalid preset plant position at index %d: row=%d, col=%d",
				i, preset.Row, preset.Col)
			continue
		}

		// 根据植物类型创建实体
		var entityID ecs.EntityID
		var err error

		switch preset.Type {
		case "peashooter":
			entityID, err = entities.NewPlantEntity(
				s.entityManager,
				s.resourceManager,
				s.gameState,
				s.reanimSystem,
				components.PlantPeashooter,
				col, row,
			)
		case "sunflower":
			entityID, err = entities.NewPlantEntity(
				s.entityManager,
				s.resourceManager,
				s.gameState,
				s.reanimSystem,
				components.PlantSunflower,
				col, row,
			)
		case "wallnut":
			entityID, err = entities.NewWallnutEntity(
				s.entityManager,
				s.resourceManager,
				s.gameState,
				s.reanimSystem,
				col, row,
			)
		case "cherrybomb":
			entityID, err = entities.NewCherryBombEntity(
				s.entityManager,
				s.resourceManager,
				s.gameState,
				col, row,
			)
		default:
			log.Printf("[GameScene] ERROR: Unknown preset plant type '%s' at index %d", preset.Type, i)
			continue
		}

		if err != nil {
			log.Printf("[GameScene] ERROR: Failed to create preset plant '%s' at (%d,%d): %v",
				preset.Type, preset.Row, preset.Col, err)
			continue
		}

		// 更新草坪网格占用状态
		if s.lawnGridSystem != nil && s.lawnGridEntityID != 0 {
			if err := s.lawnGridSystem.OccupyCell(s.lawnGridEntityID, col, row, entityID); err != nil {
				log.Printf("[GameScene] Warning: Failed to occupy grid cell (%d,%d): %v", col, row, err)
			}
		}

		spawnedCount++
		log.Printf("[GameScene] Spawned preset plant '%s' at row=%d, col=%d (Entity ID: %d)",
			preset.Type, preset.Row, preset.Col, entityID)
	}

	log.Printf("[GameScene] Preset plants spawned: %d/%d", spawnedCount, len(presetPlants))
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

	// 获取 gdata Manager
	gdataManager := s.gameState.GetGdataManager()
	if gdataManager == nil {
		log.Printf("[GameScene] Warning: gdata Manager not available, cannot save battle state")
		return
	}

	// 创建序列化器并保存
	serializer := game.NewBattleSerializer(gdataManager)
	if err := serializer.SaveBattle(s.entityManager, s.gameState, currentUser); err != nil {
		log.Printf("[GameScene] ERROR: Failed to save battle state: %v", err)
		return
	}

	log.Printf("[GameScene] Battle state saved successfully for user: %s", currentUser)
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

// showBattleSaveDialog 显示战斗存档选择对话框
//
// Story 18.3: 进入游戏后显示对话框
//
// 流程（修正版）：
//  1. 立即恢复存档数据（植物、僵尸等已显示在场景中）
//  2. 处理一次动画命令（让实体能正确渲染），但保持静止状态
//  3. 显示对话框让玩家选择
//  4. "继续": 直接开始游戏（数据已恢复）
//  5. "重玩关卡": 重新创建场景（清除已恢复的实体）
//  6. "取消": 返回主菜单
func (s *GameScene) showBattleSaveDialog() {
	log.Printf("[GameScene] 检测到战斗存档，立即恢复场景数据")

	// 1. 立即恢复存档数据（场景完整显示）
	s.restoreBattleState()
	s.skipOpeningAnimation()

	// 2. 立即处理一次动画命令（让实体能正确渲染），但不推进动画帧
	// 使用 deltaTime=0 确保动画数据初始化，但保持静止状态
	if s.reanimSystem != nil {
		s.reanimSystem.Update(0)
	}

	log.Printf("[GameScene] 场景数据已恢复，显示对话框")

	// 3. 显示对话框让玩家选择
	dialogEntity, err := entities.NewContinueGameDialogEntity(
		s.entityManager,
		s.resourceManager,
		s.battleSaveInfo,
		WindowWidth,
		WindowHeight,
		// "继续"按钮回调 - 数据已恢复，直接开始游戏
		func() {
			log.Printf("[GameScene] 用户选择继续游戏，删除存档并开始")
			s.battleSaveDialogID = 0
			// Bug Fix: 用户确认继续后才删除存档
			s.deleteBattleSave()
			// 数据已在对话框显示前恢复，无需再加载
			// Bug Fix: 如果关卡有预设植物，需要创建 Dave 对话
			s.createOpeningDaveDialogueIfNeeded()
			// 游戏将在下一帧正常更新
		},
		// "重玩关卡"按钮回调 - 重新创建场景
		func() {
			log.Printf("[GameScene] 用户选择重玩关卡，重新创建场景")
			s.battleSaveDialogID = 0
			// 删除存档
			s.deleteBattleSave()
			// 获取当前关卡ID
			currentLevelID := "1-1"
			if s.gameState.CurrentLevel != nil {
				currentLevelID = s.gameState.CurrentLevel.ID
			}
			// 重新创建场景（清除所有已恢复的实体，正常开始游戏）
			s.sceneManager.SwitchTo(NewGameScene(s.resourceManager, s.sceneManager, currentLevelID))
		},
		// "取消"按钮回调 - 返回主菜单
		func() {
			log.Printf("[GameScene] 用户选择取消，返回主菜单")
			s.battleSaveDialogID = 0
			// 返回主菜单（不删除存档，下次进入还会显示对话框）
			s.sceneManager.SwitchTo(NewMainMenuScene(s.resourceManager, s.sceneManager))
		},
	)

	if err != nil {
		log.Printf("[GameScene] Warning: Failed to create continue game dialog: %v", err)
		// 对话框创建失败，数据已恢复，直接继续游戏
		return
	}

	s.battleSaveDialogID = dialogEntity
	log.Printf("[GameScene] 继续游戏对话框已显示 (对话框ID: %d)", dialogEntity)
}

// createOpeningDaveDialogueIfNeeded 如果需要则创建开场 Dave 对话
//
// Bug Fix: 当有战斗存档时，Dave 对话不在构造函数中创建
// 此方法在用户选择"继续"后被调用，确保 Dave 对话在正确时机创建
//
// 条件：
//   - 当前关卡有预设植物（PresetPlants）
//   - Dave 对话系统已初始化
func (s *GameScene) createOpeningDaveDialogueIfNeeded() {
	// 检查是否需要创建 Dave 对话
	if s.gameState.CurrentLevel == nil || len(s.gameState.CurrentLevel.PresetPlants) == 0 {
		return
	}

	log.Printf("[GameScene] Creating opening Dave dialogue after battle save dialog")

	// 创建开场 Dave 对话（铲子教学阶段）
	openingDialogueKeys := []string{
		"CRAZY_DAVE_2400", // "你好，我的邻居！"
		"CRAZY_DAVE_2401", // "我的名字叫疯狂的戴夫。"
		"CRAZY_DAVE_2402", // "但你叫我疯狂的戴夫就行了。"
		"CRAZY_DAVE_2403", // "听好，我有个惊喜要给你。"
		"CRAZY_DAVE_2404", // "但是首先，我需要你清理一下草坪。"
		"CRAZY_DAVE_2405", // "用你的铲子挖出那些植物！"
		"CRAZY_DAVE_2406", // "开始挖吧！"
	}

	daveEntity, err := entities.NewCrazyDaveEntity(
		s.entityManager,
		s.resourceManager,
		openingDialogueKeys,
		func() {
			// Dave 对话完成回调：激活强引导模式
			log.Printf("[GameScene] Opening Dave dialogue completed, activating guided tutorial mode")
			if s.guidedTutorialSystem != nil {
				s.guidedTutorialSystem.SetActive(true)
			}
		},
	)

	if err != nil {
		log.Printf("[GameScene] ERROR: Failed to create opening Dave entity: %v", err)
		// 跳过 Dave 对话，直接激活强引导模式
		if s.guidedTutorialSystem != nil {
			s.guidedTutorialSystem.SetActive(true)
		}
	} else {
		log.Printf("[GameScene] Opening Dave entity created after battle save dialog: %d", daveEntity)
	}
}

// skipOpeningAnimation 跳过开场动画
//
// Story 18.3: 从存档恢复时跳过开场动画
//
// 处理内容：
//  1. 设置镜头到游戏位置
//  2. 标记开场动画为完成
//  3. 跳过铺草皮动画并正确设置草皮背景
//  4. 通知教学系统铺草皮已完成（让教学可以继续进行）
//  5. 启用自动阳光生成（存档恢复意味着玩家已开始游戏）
func (s *GameScene) skipOpeningAnimation() {
	// 设置镜头到游戏位置
	s.cameraX = config.GameCameraX
	s.isIntroAnimPlaying = false
	if s.cameraSystem != nil {
		s.gameState.CameraX = config.GameCameraX
	}

	// 标记开场动画为完成（使用 Skip 方法）
	if s.openingSystem != nil {
		s.openingSystem.Skip()
	}

	// 跳过铺草皮动画
	s.soddingAnimStarted = true

	// 处理草皮背景：直接合并草皮叠加层到背景
	if s.soddedBackground != nil {
		// Level 1-4: 有完整的已铺草皮背景，直接替换
		log.Printf("[GameScene] 恢复存档: 替换底层背景为已铺草皮版本")
		s.background = s.soddedBackground
		s.soddedBackground = nil
		s.preSoddedImage = nil
	} else if s.preSoddedImage != nil || s.sodRowImage != nil {
		// Level 1-1, 1-2: 需要将草皮叠加层合并到底层背景
		log.Printf("[GameScene] 恢复存档: 合并草皮叠加层到底层背景")
		mergedBg := s.createMergedBackground()
		if mergedBg != nil {
			s.background = mergedBg
			s.preSoddedImage = nil
			log.Printf("[GameScene] 草皮背景合并完成")
		}
	}

	// 通知教学系统铺草皮已完成（让教学可以继续进行）
	// 注意：教学进度会在 restoreTutorialState 中正确恢复
	if s.tutorialSystem != nil {
		s.tutorialSystem.OnSoddingComplete()
		log.Printf("[GameScene] 恢复存档: 通知教学系统铺草皮已完成")
	}

	// 阳光生成：从存档恢复时根据关卡配置决定是否启用
	// Story 19.10: 保龄球关卡（initialSun == 0）禁用阳光生成
	if s.sunSpawnSystem != nil {
		if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.InitialSun == 0 {
			// 保龄球关卡不使用阳光，保持禁用状态
			log.Printf("[GameScene] 恢复存档: 保龄球关卡，阳光生成保持禁用")
		} else {
			// 普通关卡启用阳光生成
			s.sunSpawnSystem.Enable()
			log.Printf("[GameScene] 恢复存档: 启用自动阳光生成")
		}
	}

	log.Printf("[GameScene] 跳过开场动画，直接进入游戏")
}

// restoreBattleState 从战斗存档恢复战斗状态
//
// Story 18.3: 继续游戏对话框与场景恢复
//
// 恢复流程：
//  1. 获取当前用户的存档路径
//  2. 使用 BattleSerializer.LoadBattle() 加载存档数据
//  3. 恢复游戏状态（阳光、波次进度）
//  4. 恢复所有实体（植物、僵尸、子弹、阳光、除草车）
//  5. 恢复草坪网格占用状态
//  6. 成功后删除存档（避免重复加载）
//  7. 失败时记录日志，继续正常游戏
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

	// 获取 gdata Manager
	gdataManager := s.gameState.GetGdataManager()
	if gdataManager == nil {
		log.Printf("[GameScene] Warning: gdata Manager not available, cannot restore battle state")
		return
	}

	log.Printf("[GameScene] 开始从战斗存档恢复用户: %s", currentUser)

	// 创建序列化器并加载
	serializer := game.NewBattleSerializer(gdataManager)
	saveData, err := serializer.LoadBattle(currentUser)
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

	log.Printf("[GameScene] 游戏状态已恢复: Sun=%d, Wave=%d, Time=%.1f, TotalZombiesInLevel=%d, ZombiesKilled=%d, TotalZombiesSpawned=%d, SpawnedWaves=%v",
		s.gameState.Sun, s.gameState.CurrentWaveIndex, s.gameState.LevelTime,
		s.gameState.TotalZombiesInLevel, s.gameState.ZombiesKilled,
		s.gameState.TotalZombiesSpawned, s.gameState.SpawnedWaves)

	// Story 18.3: 恢复所有实体
	s.restorePlants(saveData.Plants)
	s.restoreZombies(saveData.Zombies)
	s.restoreProjectiles(saveData.Projectiles)
	s.restoreSuns(saveData.Suns)
	s.restoreLawnmowers(saveData.Lawnmowers)

	log.Printf("[GameScene] 实体恢复完成: Plants=%d, Zombies=%d, Projectiles=%d, Suns=%d, Lawnmowers=%d",
		len(saveData.Plants), len(saveData.Zombies), len(saveData.Projectiles),
		len(saveData.Suns), len(saveData.Lawnmowers))

	// 显示调整后的击杀计数（可能因跳过死亡僵尸而增加）
	log.Printf("[GameScene] 实体恢复后状态: ZombiesKilled=%d/%d, TotalZombiesSpawned=%d, OnField=%d",
		s.gameState.ZombiesKilled, s.gameState.TotalZombiesInLevel,
		s.gameState.TotalZombiesSpawned,
		s.gameState.TotalZombiesSpawned-s.gameState.ZombiesKilled)

	// 跳过开场动画
	s.isIntroAnimPlaying = false
	s.cameraX = config.GameCameraX
	if s.cameraSystem != nil {
		s.gameState.CameraX = config.GameCameraX
	}

	// 跳过铺草皮动画
	s.soddingAnimStarted = true

	// Story 18.3: 恢复进度条数据
	s.restoreProgressBar(saveData)

	// Story 18.3: 恢复教学状态（如果是教学关卡）
	if saveData.Tutorial != nil {
		s.restoreTutorialState(saveData.Tutorial)
	}

	// Story 18.3: 恢复波次计时系统状态
	// 这是关键：让 WaveTimingSystem 知道当前进度，以便正确触发后续波次
	if s.levelSystem != nil {
		waveTimingSystem := s.levelSystem.GetWaveTimingSystem()
		if waveTimingSystem != nil {
			waveTimingSystem.RestoreState(saveData.CurrentWaveIndex, saveData.LevelTime)
		}
	}

	// Bug Fix: 不再在恢复后立即删除存档
	// 存档删除应该在用户确认"继续"后才执行，这样：
	// - 用户选择"取消"返回主菜单时，存档仍然保留
	// - 用户选择"继续"时，在继续按钮回调中删除存档
	// - 用户选择"重玩关卡"时，在重玩按钮回调中删除存档
	log.Printf("[GameScene] 战斗状态恢复完成! (存档将在用户确认后删除)")
}

// restorePlants 恢复植物实体
//
// Story 18.3: 从存档数据重建植物实体
//
// 恢复内容：
//   - 植物类型和位置（网格行列）
//   - 生命值（当前/最大）
//   - 攻击冷却时间
//   - 草坪网格占用状态
//
// 简化处理：
//   - 动画从 idle 状态开始
//   - 眨眼计时器重置
func (s *GameScene) restorePlants(plants []game.PlantData) {
	for _, plantData := range plants {
		// 将植物类型字符串转换为 PlantType
		plantType := stringToPlantType(plantData.PlantType)
		if plantType == components.PlantUnknown {
			log.Printf("[GameScene] Warning: Unknown plant type '%s', skipping", plantData.PlantType)
			continue
		}

		// 根据植物类型创建实体
		var entityID ecs.EntityID
		var err error

		switch plantType {
		case components.PlantSunflower, components.PlantPeashooter:
			entityID, err = entities.NewPlantEntity(
				s.entityManager,
				s.resourceManager,
				s.gameState,
				s.reanimSystem,
				plantType,
				plantData.GridCol,
				plantData.GridRow,
			)
		case components.PlantWallnut:
			entityID, err = entities.NewWallnutEntity(
				s.entityManager,
				s.resourceManager,
				s.gameState,
				s.reanimSystem,
				plantData.GridCol,
				plantData.GridRow,
			)
		case components.PlantCherryBomb:
			entityID, err = entities.NewCherryBombEntity(
				s.entityManager,
				s.resourceManager,
				s.gameState,
				plantData.GridCol,
				plantData.GridRow,
			)
		default:
			log.Printf("[GameScene] Warning: Unsupported plant type '%s', skipping", plantData.PlantType)
			continue
		}

		if err != nil {
			log.Printf("[GameScene] ERROR: Failed to restore plant %s at (%d,%d): %v",
				plantData.PlantType, plantData.GridRow, plantData.GridCol, err)
			continue
		}

		// 恢复生命值
		if healthComp, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, entityID); ok {
			healthComp.CurrentHealth = plantData.Health
			healthComp.MaxHealth = plantData.MaxHealth
		}

		// Bug Fix: 恢复计时器状态（向日葵阳光生产、豌豆射手攻击冷却等）
		// 必须同时恢复 TargetTime 和 CurrentTime，因为向日葵等植物有变周期机制
		// 向日葵首次周期是 7 秒，后续周期是 24 秒
		if timerComp, ok := ecs.GetComponent[*components.TimerComponent](s.entityManager, entityID); ok {
			// 先恢复 TargetTime（如果保存了）
			if plantData.TimerTargetTime > 0 {
				timerComp.TargetTime = plantData.TimerTargetTime
			}
			// 再计算 CurrentTime
			// 剩余冷却时间 = TargetTime - CurrentTime
			// 所以 CurrentTime = TargetTime - AttackCooldown
			timerComp.CurrentTime = timerComp.TargetTime - plantData.AttackCooldown
			if timerComp.CurrentTime < 0 {
				timerComp.CurrentTime = 0
			}
			// 如果冷却已完成（AttackCooldown <= 0），标记为就绪
			timerComp.IsReady = plantData.AttackCooldown <= 0
			log.Printf("[GameScene] Restored timer for %s: CurrentTime=%.2f, TargetTime=%.2f, IsReady=%v",
				plantData.PlantType, timerComp.CurrentTime, timerComp.TargetTime, timerComp.IsReady)
		}

		// 更新草坪网格占用状态
		if s.lawnGridSystem != nil && s.lawnGridEntityID != 0 {
			if err := s.lawnGridSystem.OccupyCell(s.lawnGridEntityID, plantData.GridCol, plantData.GridRow, entityID); err != nil {
				log.Printf("[GameScene] Warning: Failed to occupy grid cell (%d,%d): %v",
					plantData.GridCol, plantData.GridRow, err)
			}
		}

		log.Printf("[GameScene] Restored plant %s at (%d,%d), health=%d/%d",
			plantData.PlantType, plantData.GridRow, plantData.GridCol, plantData.Health, plantData.MaxHealth)
	}
}

// restoreZombies 恢复僵尸实体
//
// Story 18.3: 从存档数据重建僵尸实体
//
// 恢复内容：
//   - 僵尸类型和位置（X, Y）
//   - 生命值和护甲值
//   - 速度
//   - 行号
//
// 简化处理：
//   - 行为状态简化为 walking（让系统重新判断）
//   - 动画从 walk 状态开始
func (s *GameScene) restoreZombies(zombies []game.ZombieData) {
	for _, zombieData := range zombies {
		// 跳过正在死亡的僵尸
		// 这些僵尸在存档时还在播放死亡动画，尚未被计入 ZombiesKilled
		// 跳过恢复时需要增加击杀计数，否则会导致胜利条件计算错误
		if zombieData.BehaviorType == "dying" || zombieData.BehaviorType == "dying_explosion" {
			log.Printf("[GameScene] Skipping dying zombie at (%.1f, %.1f), incrementing ZombiesKilled", zombieData.X, zombieData.Y)
			s.gameState.ZombiesKilled++
			continue
		}

		// 计算行号（从 Y 坐标推算，如果 Lane 未设置）
		lane := zombieData.Lane
		if lane == 0 {
			// 从 Y 坐标推算行号
			lane = int((zombieData.Y-config.GridWorldStartY)/config.CellHeight) + 1
			if lane < 1 {
				lane = 1
			}
			if lane > 5 {
				lane = 5
			}
		}

		// 根据僵尸类型创建实体
		var entityID ecs.EntityID
		var err error

		switch zombieData.ZombieType {
		case "basic":
			entityID, err = entities.NewZombieEntity(s.entityManager, s.resourceManager, lane-1, zombieData.X)
		case "conehead":
			entityID, err = entities.NewConeheadZombieEntity(s.entityManager, s.resourceManager, lane-1, zombieData.X)
		case "buckethead":
			entityID, err = entities.NewBucketheadZombieEntity(s.entityManager, s.resourceManager, lane-1, zombieData.X)
		default:
			// 默认创建普通僵尸
			entityID, err = entities.NewZombieEntity(s.entityManager, s.resourceManager, lane-1, zombieData.X)
		}

		if err != nil {
			log.Printf("[GameScene] ERROR: Failed to restore zombie %s at (%.1f, %.1f): %v",
				zombieData.ZombieType, zombieData.X, zombieData.Y, err)
			continue
		}

		// 恢复位置（Y 坐标可能需要调整）
		if posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID); ok {
			posComp.X = zombieData.X
			posComp.Y = zombieData.Y
		}

		// 恢复生命值
		if healthComp, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, entityID); ok {
			healthComp.CurrentHealth = zombieData.Health
			healthComp.MaxHealth = zombieData.MaxHealth
		}

		// 恢复护甲值（如果有）
		if zombieData.ArmorHealth > 0 {
			if armorComp, ok := ecs.GetComponent[*components.ArmorComponent](s.entityManager, entityID); ok {
				armorComp.CurrentArmor = zombieData.ArmorHealth
				armorComp.MaxArmor = zombieData.ArmorMax
			}
		}

		// 恢复速度并激活僵尸
		if velComp, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID); ok {
			if zombieData.VelocityX != 0 {
				velComp.VX = zombieData.VelocityX
			} else {
				// 如果没有保存速度，使用默认速度激活（僵尸标准移动速度）
				velComp.VX = -23.0
			}
		}

		// 设置行为状态为 walking（让 BehaviorSystem 重新判断是否需要切换到 eating）
		if behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID); ok {
			if zombieData.IsEating {
				behaviorComp.ZombieAnimState = components.ZombieAnimEating
			} else {
				behaviorComp.ZombieAnimState = components.ZombieAnimWalking
			}
		}

		// 触发走路动画（僵尸工厂默认创建的是 idle 动画）
		// 根据僵尸类型选择正确的 unit ID
		unitID := "zombie"
		comboName := "walk"
		if zombieData.IsEating {
			comboName = "eat" // Bug Fix: 配置中的啃食动画 combo 名称是 "eat"，不是 "eating"
		}
		switch zombieData.ZombieType {
		case "conehead":
			unitID = "zombie_conehead"
		case "buckethead":
			unitID = "zombie_buckethead"
		}
		ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
			UnitID:    unitID,
			ComboName: comboName,
			Processed: false,
		})

		// 添加目标行组件
		ecs.AddComponent(s.entityManager, entityID, &components.ZombieTargetLaneComponent{
			TargetRow: lane - 1,
		})

		log.Printf("[GameScene] Restored zombie %s at (%.1f, %.1f), lane=%d, health=%d/%d",
			zombieData.ZombieType, zombieData.X, zombieData.Y, lane, zombieData.Health, zombieData.MaxHealth)
	}
}

// restoreProjectiles 恢复子弹实体
//
// Story 18.3: 从存档数据重建子弹实体
//
// 恢复内容：
//   - 子弹类型和位置
//   - 速度
//   - 伤害值
//
// 简化处理：
//   - 只恢复豌豆子弹
func (s *GameScene) restoreProjectiles(projectiles []game.ProjectileData) {
	for _, projData := range projectiles {
		// 目前只支持豌豆子弹
		if projData.Type != "pea" {
			log.Printf("[GameScene] Warning: Unsupported projectile type '%s', skipping", projData.Type)
			continue
		}

		// 使用工厂函数创建子弹实体
		entityID, err := entities.NewPeaProjectile(s.entityManager, s.resourceManager, projData.X, projData.Y)
		if err != nil {
			log.Printf("[GameScene] ERROR: Failed to restore projectile at (%.1f, %.1f): %v", projData.X, projData.Y, err)
			continue
		}

		// 恢复速度（如果保存了不同的速度）
		if projData.VelocityX != 0 {
			if velComp, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID); ok {
				velComp.VX = projData.VelocityX
			}
		}

		log.Printf("[GameScene] Restored projectile at (%.1f, %.1f)", projData.X, projData.Y)
	}
}

// restoreSuns 恢复阳光实体
//
// Story 18.3: 从存档数据重建阳光实体
//
// 恢复内容：
//   - 位置
//   - 剩余生命周期
//   - 收集状态
//
// 简化处理：
//   - 收集动画状态重置（正在收集的阳光按已着陆处理）
func (s *GameScene) restoreSuns(suns []game.SunData) {
	for _, sunData := range suns {
		// 跳过正在收集的阳光（简化处理）
		if sunData.IsCollecting {
			log.Printf("[GameScene] Skipping collecting sun at (%.1f, %.1f)", sunData.X, sunData.Y)
			continue
		}

		// 创建静态阳光实体（已着陆状态）
		entityID := entities.NewSunEntityStatic(s.entityManager, s.resourceManager, sunData.X, sunData.Y)

		// 添加动画命令组件，让 ReanimSystem 初始化阳光动画
		ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
			UnitID:    "sun",
			ComboName: "idle",
			Processed: false,
		})

		// 恢复剩余生命周期
		if lifetimeComp, ok := ecs.GetComponent[*components.LifetimeComponent](s.entityManager, entityID); ok {
			if sunData.Lifetime > 0 {
				// 设置当前生命周期为 (最大 - 剩余)
				lifetimeComp.CurrentLifetime = lifetimeComp.MaxLifetime - sunData.Lifetime
				if lifetimeComp.CurrentLifetime < 0 {
					lifetimeComp.CurrentLifetime = 0
				}
			}
		}

		// 确保阳光处于着陆状态
		if sunComp, ok := ecs.GetComponent[*components.SunComponent](s.entityManager, entityID); ok {
			sunComp.State = components.SunLanded
		}

		// 停止下落
		if velComp, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID); ok {
			velComp.VY = 0
		}

		log.Printf("[GameScene] Restored sun at (%.1f, %.1f), lifetime=%.1f", sunData.X, sunData.Y, sunData.Lifetime)
	}
}

// restoreLawnmowers 恢复除草车实体
//
// Story 18.3: 从存档数据重建除草车实体
//
// 恢复内容：
//   - 行号
//   - 位置
//   - 触发状态
//   - 激活状态
//
// 注意：
//   - 已触发且移出屏幕的除草车不恢复
//   - 正在移动的除草车恢复位置和速度
func (s *GameScene) restoreLawnmowers(lawnmowers []game.LawnmowerData) {
	// 首先检查是否已经通过 initLawnmowers 创建了除草车
	// 如果存档中有除草车数据，我们需要先清理默认创建的除草车
	existingLawnmowers := ecs.GetEntitiesWith1[*components.LawnmowerComponent](s.entityManager)
	for _, entityID := range existingLawnmowers {
		s.entityManager.DestroyEntity(entityID)
	}

	for _, lmData := range lawnmowers {
		// 跳过已触发且激活的除草车（已经移出屏幕）
		if lmData.Triggered && lmData.Active && lmData.X > float64(WindowWidth)+100 {
			log.Printf("[GameScene] Skipping lawnmower on lane %d (already moved off-screen)", lmData.Lane)
			continue
		}

		// 创建除草车实体
		entityID, err := entities.NewLawnmowerEntity(s.entityManager, s.resourceManager, lmData.Lane)
		if err != nil {
			log.Printf("[GameScene] ERROR: Failed to restore lawnmower on lane %d: %v", lmData.Lane, err)
			continue
		}

		// 恢复位置
		if posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID); ok {
			posComp.X = lmData.X
		}

		// 恢复状态
		if lmComp, ok := ecs.GetComponent[*components.LawnmowerComponent](s.entityManager, entityID); ok {
			lmComp.IsTriggered = lmData.Triggered
			lmComp.IsMoving = lmData.Active

			// 恢复存档时跳过入场动画，直接进入静止状态
			lmComp.IsEntering = false
		}

		// 设置动画状态：静止的除草车暂停动画，移动中的除草车播放动画
		if reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
			if lmData.Active {
				// 正在移动：播放动画
				reanimComp.IsPaused = false
			} else {
				// 静止状态：暂停动画
				reanimComp.IsPaused = true
			}
		}

		// 如果正在移动，设置速度
		if lmData.Active {
			if velComp, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID); ok {
				velComp.VX = config.LawnmowerSpeed
			} else {
				s.entityManager.AddComponent(entityID, &components.VelocityComponent{
					VX: config.LawnmowerSpeed,
					VY: 0,
				})
			}
		}

		log.Printf("[GameScene] Restored lawnmower on lane %d at X=%.1f, triggered=%v, active=%v",
			lmData.Lane, lmData.X, lmData.Triggered, lmData.Active)
	}
}

// stringToPlantType 将植物类型字符串转换为 PlantType
func stringToPlantType(s string) components.PlantType {
	switch s {
	case "Sunflower", "sunflower":
		return components.PlantSunflower
	case "Peashooter", "peashooter":
		return components.PlantPeashooter
	case "Wallnut", "wallnut":
		return components.PlantWallnut
	case "CherryBomb", "cherrybomb":
		return components.PlantCherryBomb
	default:
		return components.PlantUnknown
	}
}

// restoreProgressBar 恢复进度条数据
//
// Story 18.3: 从存档数据恢复进度条状态
//
// 恢复内容：
//   - 已击杀僵尸数
//   - 当前波次号
//   - 进度百分比（直接设置，无动画过渡）
//   - 显示状态（已有波次时显示进度条）
func (s *GameScene) restoreProgressBar(saveData *game.BattleSaveData) {
	if s.levelProgressBarEntity == 0 {
		log.Printf("[GameScene] Warning: No progress bar entity, cannot restore progress bar data")
		return
	}

	progressBar, ok := ecs.GetComponent[*components.LevelProgressBarComponent](s.entityManager, s.levelProgressBarEntity)
	if !ok {
		log.Printf("[GameScene] Warning: LevelProgressBarComponent not found for entity %d", s.levelProgressBarEntity)
		return
	}

	// 恢复已击杀僵尸数
	progressBar.KilledZombies = saveData.ZombiesKilled

	// 恢复当前波次号（存档中是索引，从0开始；波次号从1开始）
	progressBar.CurrentWaveNum = saveData.CurrentWaveIndex + 1

	// 计算进度百分比
	if progressBar.TotalZombies > 0 {
		progressBar.ProgressPercent = float64(saveData.ZombiesKilled) / float64(progressBar.TotalZombies)
		// 同步到虚拟/现实进度（两者相同，无动画过渡）
		progressBar.VirtualProgress = progressBar.ProgressPercent
		progressBar.RealProgress = progressBar.ProgressPercent

		// 关键修复：设置 LastTrackUpdateCS 为一个足够大的值
		// 这样 updateRealProgress 不会在恢复后立即开始平滑追踪动画
		// 使用当前关卡时间转换为厘秒
		progressBar.GameTickCS = int(saveData.LevelTime * 100)
		progressBar.LastTrackUpdateCS = progressBar.GameTickCS
	}

	// 如果已经开始波次，显示进度条（而非仅显示关卡文本）
	if saveData.CurrentWaveIndex > 0 || saveData.ZombiesKilled > 0 || len(saveData.Zombies) > 0 {
		progressBar.ShowLevelTextOnly = false
	}

	log.Printf("[GameScene] 进度条已恢复: KilledZombies=%d/%d, Wave=%d, Progress=%.2f%% (无动画)",
		progressBar.KilledZombies, progressBar.TotalZombies,
		progressBar.CurrentWaveNum, progressBar.ProgressPercent*100)
}

// restoreTutorialState 恢复教学状态
//
// Story 18.3: 从存档数据恢复教学进度
//
// 恢复内容：
//   - 当前教学步骤索引
//   - 已完成的步骤
//   - 激活状态
//   - 植物和向日葵计数
//
// 注意：
//   - 教学系统的内部状态需要通过 TutorialComponent 恢复
//   - 草坪闪烁状态根据当前步骤自动调整
func (s *GameScene) restoreTutorialState(tutorialData *game.TutorialSaveData) {
	if tutorialData == nil {
		return
	}

	// 查找教学组件
	tutorialEntities := ecs.GetEntitiesWith1[*components.TutorialComponent](s.entityManager)
	if len(tutorialEntities) == 0 {
		log.Printf("[GameScene] Warning: No tutorial entity found, cannot restore tutorial state")
		return
	}

	// 取第一个教学实体（通常只有一个）
	tutorialEntity := tutorialEntities[0]
	tutorial, ok := ecs.GetComponent[*components.TutorialComponent](s.entityManager, tutorialEntity)
	if !ok {
		log.Printf("[GameScene] Warning: TutorialComponent not found")
		return
	}

	// 恢复教学状态
	tutorial.CurrentStepIndex = tutorialData.CurrentStepIndex
	tutorial.IsActive = tutorialData.IsActive

	// 恢复已完成的步骤
	if tutorialData.CompletedSteps != nil {
		tutorial.CompletedSteps = make(map[string]bool)
		for k, v := range tutorialData.CompletedSteps {
			tutorial.CompletedSteps[k] = v
		}
	}

	// 根据恢复的步骤调整游戏状态
	// 如果已经过了 plantPlaced 步骤，禁用草坪闪烁
	if tutorialData.CompletedSteps["plantPlaced"] || tutorialData.PlantCount > 0 {
		if s.lawnGridSystem != nil {
			s.lawnGridSystem.DisableFlash()
		}
	}

	log.Printf("[GameScene] 教学状态已恢复: StepIndex=%d, IsActive=%v, CompletedSteps=%d, PlantCount=%d",
		tutorial.CurrentStepIndex, tutorial.IsActive,
		len(tutorial.CompletedSteps), tutorialData.PlantCount)
}
