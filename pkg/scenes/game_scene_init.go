package scenes

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/modules"
	"github.com/gonewx/pvz/pkg/systems"
)

// initPlantCardSystems 初始化植物选择栏模块
func (s *GameScene) initPlantCardSystems(rm *game.ResourceManager) {
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

// initPauseMenuModule 初始化暂停菜单
func (s *GameScene) initPauseMenuModule(rm *game.ResourceManager) {
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
		settingsManager,
		WindowWidth,
		WindowHeight,
		modules.PauseMenuCallbacks{
			OnContinue: func() {
				s.gameState.SetPaused(false) // 恢复游戏
			},
			OnRestart: func() {
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
				if s.levelSystem != nil {
					s.levelSystem.PauseWaveTiming()
				}
			},
			OnResumeMusic: func() {
				if s.levelSystem != nil {
					s.levelSystem.ResumeWaveTiming()
				}
			},
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

// initProgressBar 初始化关卡进度条
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

// initLawnmowers 初始化除草车实体（每行一台，僵尸到达左侧时触发）
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

// spawnPresetPlants 生成预设植物（用于教学关卡等）
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

// saveBattleState 保存当前战斗状态到存档

// createOpeningDaveDialogueIfNeeded 如果需要则创建开场 Dave 对话
func (s *GameScene) createOpeningDaveDialogueIfNeeded() {
	// 检查是否需要创建 Dave 对话
	if s.gameState.CurrentLevel == nil || len(s.gameState.CurrentLevel.PresetPlants) == 0 {
		return
	}

	// 如果场景中已有 Dave 对话实体（从存档恢复），不创建新的
	existingDaves := ecs.GetEntitiesWith1[*components.DaveDialogueComponent](s.entityManager)
	if len(existingDaves) > 0 {
		log.Printf("[GameScene] Skipping Dave dialogue creation: already exists from save restore (entity %d)", existingDaves[0])
		return
	}

	// 检查关卡阶段，如果已进入保龄球阶段，不创建 Dave 对话
	phaseEntities := ecs.GetEntitiesWith1[*components.LevelPhaseComponent](s.entityManager)
	if len(phaseEntities) > 0 {
		if phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](s.entityManager, phaseEntities[0]); ok {
			if phaseComp.CurrentPhase >= 2 {
				log.Printf("[GameScene] Skipping Dave dialogue: already in phase %d (bowling phase)", phaseComp.CurrentPhase)
				return
			}
		}
	}

	// 检查传送带是否已激活（兼容旧存档）
	conveyorEntities := ecs.GetEntitiesWith1[*components.ConveyorBeltComponent](s.entityManager)
	if len(conveyorEntities) > 0 {
		if conveyorComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, conveyorEntities[0]); ok {
			if conveyorComp.IsActive {
				log.Printf("[GameScene] Skipping Dave dialogue: conveyor belt is active (bowling phase)")
				return
			}
		}
	}

	// 检查强引导教学是否已激活
	guidedEntities := ecs.GetEntitiesWith1[*components.GuidedTutorialComponent](s.entityManager)
	if len(guidedEntities) > 0 {
		if guidedComp, ok := ecs.GetComponent[*components.GuidedTutorialComponent](s.entityManager, guidedEntities[0]); ok {
			if guidedComp.IsActive {
				log.Printf("[GameScene] Skipping Dave dialogue: guided tutorial is active (shovel phase)")
				return
			}
		}
	}

	log.Printf("[GameScene] Creating opening Dave dialogue after battle save dialog")

	openingDialogueKeys := []string{
		"CRAZY_DAVE_2400",
		"CRAZY_DAVE_2401",
		"CRAZY_DAVE_2402",
		"CRAZY_DAVE_2403",
		"CRAZY_DAVE_2404",
		"CRAZY_DAVE_2405",
		"CRAZY_DAVE_2406",
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
		if s.guidedTutorialSystem != nil {
			s.guidedTutorialSystem.SetActive(true)
		}
	} else {
		log.Printf("[GameScene] Opening Dave entity created after battle save dialog: %d", daveEntity)
	}
}

// skipOpeningAnimation 跳过开场动画（存档恢复时调用）
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

	if s.soddedBackground != nil {
		log.Printf("[GameScene] 恢复存档: 替换底层背景为已铺草皮版本")
		s.background = s.soddedBackground
		s.soddedBackground = nil
		s.preSoddedImage = nil
	} else if s.preSoddedImage != nil || s.sodRowImage != nil {
		log.Printf("[GameScene] 恢复存档: 合并草皮叠加层到底层背景")
		mergedBg := s.createMergedBackground()
		if mergedBg != nil {
			s.background = mergedBg
			s.preSoddedImage = nil
			log.Printf("[GameScene] 草皮背景合并完成")
		}
	}

	if s.tutorialSystem != nil {
		s.tutorialSystem.OnSoddingComplete()
		log.Printf("[GameScene] 恢复存档: 通知教学系统铺草皮已完成")
	}

	if s.sunSpawnSystem != nil {
		if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.InitialSun == 0 {
			log.Printf("[GameScene] 恢复存档: 保龄球关卡，阳光生成保持禁用")
		} else {
			s.sunSpawnSystem.Enable()
			log.Printf("[GameScene] 恢复存档: 启用自动阳光生成")
		}
	}

	log.Printf("[GameScene] 跳过开场动画，直接进入游戏")
}

// restoreBattleState 从战斗存档恢复战斗状态
