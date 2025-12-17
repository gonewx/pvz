package scenes

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/systems"
	"github.com/gonewx/pvz/pkg/systems/behavior"
	"github.com/hajimehoshi/ebiten/v2"
)

// game_scene_systems.go
// 系统初始化模块 - 将 NewGameScene() 的系统初始化逻辑拆分为职责清晰的函数
//
// 架构说明:
//   - 本文件包含所有系统的初始化逻辑
//   - 按职责分为: 核心系统、游戏逻辑系统、UI系统、关卡特定系统
//   - 每个函数负责一组相关系统的初始化
//   - NewGameScene() 调用这些函数以减少复杂度

// initCoreSystems 初始化核心 ECS 系统
// 包括: RenderSystem, ReanimSystem, LawnGridSystem, InputSystem, SunSystem
func (s *GameScene) initCoreSystems(rm *game.ResourceManager) {
	// 初始化核心渲染和动画系统
	s.renderSystem = systems.NewRenderSystem(s.entityManager)
	s.reanimSystem = systems.NewReanimSystem(s.entityManager)
	log.Printf("[GameScene] Initialized core render systems")

	// Story 13.6: 设置配置管理器
	if configManager := rm.GetReanimConfigManager(); configManager != nil {
		s.reanimSystem.SetConfigManager(configManager)
	}

	// Story 5.4.1: 设置资源加载器，用于运行时单位切换
	s.reanimSystem.SetResourceLoader(rm)

	// 设置 ReanimSystem 引用
	s.renderSystem.SetReanimSystem(s.reanimSystem)
	s.renderSystem.SetResourceManager(rm)

	// 初始化草坪网格系统
	var enabledLanes []int
	if s.gameState.CurrentLevel != nil {
		enabledLanes = s.gameState.CurrentLevel.EnabledLanes
	}
	if len(enabledLanes) == 0 {
		enabledLanes = []int{1, 2, 3, 4, 5}
	}
	s.lawnGridSystem = systems.NewLawnGridSystem(s.entityManager, enabledLanes)
	s.lawnGridEntityID = s.entityManager.CreateEntity()
	s.entityManager.AddComponent(s.lawnGridEntityID, &components.LawnGridComponent{})
	log.Printf("[GameScene] Initialized lawn grid system (Entity ID: %d) with enabled lanes: %v",
		s.lawnGridEntityID, enabledLanes)

	// 初始化阳光系统
	sunCollectionTargetX := float64(config.SeedBankX + config.SunPoolOffsetX)
	sunCollectionTargetY := float64(config.SeedBankY + config.SunPoolOffsetY)

	s.sunMovementSystem = systems.NewSunMovementSystem(s.entityManager)
	s.sunCollectionSystem = systems.NewSunCollectionSystem(
		s.entityManager,
		s.gameState,
		sunCollectionTargetX,
		sunCollectionTargetY,
	)
	s.sunSpawnSystem = systems.NewSunSpawnSystem(
		s.entityManager,
		rm,
		config.SkyDropSunMinX,
		config.SkyDropSunMaxX,
		config.SkyDropSunMinTargetY,
		config.SkyDropSunMaxTargetY,
	)
	log.Printf("[GameScene] Initialized sun systems")

	// 初始化输入系统
	s.inputSystem = systems.NewInputSystem(
		s.entityManager,
		rm,
		s.gameState,
		s.reanimSystem,
		sunCollectionTargetX,
		sunCollectionTargetY,
		s.lawnGridSystem,
		s.lawnGridEntityID,
	)
	log.Printf("[GameScene] Initialized input system")

	// 初始化生命周期系统
	s.lifetimeSystem = systems.NewLifetimeSystem(s.entityManager)

	// 初始化植物预览系统
	s.plantPreviewSystem = systems.NewPlantPreviewSystem(s.entityManager, s.gameState, s.lawnGridSystem)
	s.plantPreviewSystem.SetLawnGridEntityID(s.lawnGridEntityID)
	s.plantPreviewRenderSystem = systems.NewPlantPreviewRenderSystem(s.entityManager, s.plantPreviewSystem)
	log.Printf("[GameScene] Initialized plant preview systems")
}

// initGameplaySystems 初始化游戏逻辑系统
// 包括: BehaviorSystem, PhysicsSystem, LevelSystem, WaveSpawnSystem等
func (s *GameScene) initGameplaySystems(rm *game.ResourceManager) {
	// 初始化行为系统
	s.behaviorSystem = behavior.NewBehaviorSystem(
		s.entityManager,
		rm,
		s.gameState,
		s.lawnGridSystem,
		s.lawnGridEntityID,
		s.reanimSystem,
	)
	log.Printf("[GameScene] Initialized behavior system")

	// 初始化物理系统
	s.physicsSystem = systems.NewPhysicsSystem(s.entityManager, rm)
	log.Printf("[GameScene] Initialized physics system")

	// 初始化波次生成系统
	spawnRules, err := config.LoadSpawnRules("data/spawn_rules.yaml")
	if err != nil {
		log.Printf("[GameScene] Warning: Failed to load spawn rules: %v", err)
		spawnRules = nil
	}

	zombiePhysics, err := config.LoadZombiePhysicsConfig("data/zombie_physics.yaml")
	if err != nil {
		log.Printf("[GameScene] Warning: Failed to load zombie physics config: %v", err)
		zombiePhysics = nil
	}

	s.waveSpawnSystem = systems.NewWaveSpawnSystem(
		s.entityManager,
		rm,
		s.gameState.CurrentLevel,
		s.gameState,
		spawnRules,
		zombiePhysics,
	)
	log.Printf("[GameScene] Initialized wave spawn system (spawn rules: %v, physics: %v)",
		spawnRules != nil, zombiePhysics != nil)

	// 初始化镜头系统
	s.cameraSystem = systems.NewCameraSystem(s.entityManager, s.gameState)
	log.Printf("[GameScene] Initialized camera system")

	// 初始化粒子系统
	s.particleSystem = systems.NewParticleSystem(s.entityManager, s.resourceManager)
	log.Printf("[GameScene] Initialized particle system")

	// 初始化奖励系统
	s.rewardSystem = systems.NewRewardAnimationSystem(
		s.entityManager,
		s.gameState,
		rm,
		s.sceneManager,
		s.reanimSystem,
		s.particleSystem,
		s.renderSystem,
	)
	log.Printf("[GameScene] Initialized reward animation system")

	// 初始化开场动画系统
	if s.gameState.CurrentLevel != nil {
		s.openingSystem = systems.NewOpeningAnimationSystem(
			s.entityManager,
			s.gameState,
			rm,
			s.gameState.CurrentLevel,
			s.cameraSystem,
		)
		if s.openingSystem != nil {
			log.Printf("[GameScene] Initialized opening animation system")
		} else {
			log.Printf("[GameScene] Skipping opening animation (tutorial/skip/special level)")
		}
	}

	// 初始化ReadySetPlant系统
	s.readySetPlantSystem = systems.NewReadySetPlantSystem(s.entityManager, rm)
	log.Printf("[GameScene] Initialized ReadySetPlant animation system")

	// 初始化除草车系统
	s.lawnmowerSystem = systems.NewLawnmowerSystem(s.entityManager, rm, s.gameState)
	log.Printf("[GameScene] Initialized lawnmower system")

	// 初始化关卡系统
	s.levelSystem = systems.NewLevelSystem(
		s.entityManager,
		s.gameState,
		s.waveSpawnSystem,
		rm,
		s.rewardSystem,
		s.lawnmowerSystem,
	)
	if zombiePhysics != nil {
		s.levelSystem.SetZombiePhysicsConfig(zombiePhysics)
	}
	log.Printf("[GameScene] Initialized level system")

	// 初始化最后一波提示系统
	s.finalWaveWarningSystem = systems.NewFinalWaveWarningSystem(s.entityManager)
	log.Printf("[GameScene] Initialized final wave warning system")

	// 初始化僵尸获胜流程系统
	s.zombiesWonPhaseSystem = systems.NewZombiesWonPhaseSystem(
		s.entityManager,
		s.resourceManager,
		s.gameState,
		WindowWidth,
		WindowHeight,
	)
	s.zombiesWonPhaseSystem.SetRetryCallback(func() {
		s.retryLevel()
	})
	log.Printf("[GameScene] Initialized zombies won phase system")

	// 初始化僵尸行转换系统
	s.zombieLaneTransitionSystem = systems.NewZombieLaneTransitionSystem(s.entityManager)
	log.Printf("[GameScene] Initialized zombie lane transition system")

	// 初始化闪烁效果系统
	s.flashEffectSystem = systems.NewFlashEffectSystem(s.entityManager)
	log.Printf("[GameScene] Initialized flash effect system")

	// 初始化僵尸呻吟音效系统
	s.zombieGroanSystem = systems.NewZombieGroanSystem(s.entityManager, s.gameState)
	log.Printf("[GameScene] Initialized zombie groan system")

	// 初始化撑杆僵尸跳跃系统
	s.poleVaultSystem = systems.NewPoleVaultSystem(s.entityManager, s.gameState)
	log.Printf("[GameScene] Initialized pole vault system")

	// 初始化减速效果系统
	s.slowEffectSystem = systems.NewSlowEffectSystem(s.entityManager)
	log.Printf("[GameScene] Initialized slow effect system")

	// 初始化大嘴花系统
	s.chomperSystem = systems.NewChomperSystem(s.entityManager, s.gameState)
	log.Printf("[GameScene] Initialized chomper system")
}

// initUISystems 初始化 UI 系统
// 包括: ButtonSystem, DialogSystem, PauseMenuModule等
func (s *GameScene) initUISystems(rm *game.ResourceManager) {
	// 初始化铺草皮系统
	s.soddingSystem = systems.NewSoddingSystem(s.entityManager, s.resourceManager)
	log.Printf("[GameScene] Initialized sodding animation system")

	// 初始化按钮系统
	s.buttonSystem = systems.NewButtonSystem(s.entityManager)
	s.buttonRenderSystem = systems.NewButtonRenderSystem(s.entityManager)
	log.Printf("[GameScene] Initialized button systems")

	// 初始化滑块和复选框系统
	s.sliderSystem = systems.NewSliderSystem(s.entityManager)
	s.checkboxSystem = systems.NewCheckboxSystem(s.entityManager)
	log.Printf("[GameScene] Initialized slider and checkbox systems")

	// 初始化对话框系统
	titleFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 24)
	if err != nil {
		log.Printf("Warning: Failed to load title font: %v", err)
	}

	messageFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 18)
	if err != nil {
		log.Printf("Warning: Failed to load message font: %v", err)
	}

	buttonFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 20)
	if err != nil {
		log.Printf("Warning: Failed to load button font: %v", err)
	}

	s.dialogInputSystem = systems.NewDialogInputSystem(s.entityManager)
	s.dialogRenderSystem = systems.NewDialogRenderSystem(
		s.entityManager,
		WindowWidth,
		WindowHeight,
		titleFont,
		messageFont,
		buttonFont,
	)
	log.Printf("[GameScene] Initialized dialog systems")

	// 初始化菜单按钮
	s.initMenuButton(rm)

	// 初始化暂停菜单模块
	s.initPauseMenuModule(rm)

	// 初始化关卡进度条系统
	s.initProgressBar(rm)
}

// initLevelSpecificSystems 初始化关卡特定系统
// 包括: TutorialSystem, ConveyorBeltSystem, ShovelInteractionSystem等
func (s *GameScene) initLevelSpecificSystems(rm *game.ResourceManager) {
	if s.gameState.CurrentLevel == nil {
		return
	}

	// 初始化教学系统
	if len(s.gameState.CurrentLevel.TutorialSteps) > 0 {
		s.tutorialSystem = systems.NewTutorialSystem(
			s.entityManager,
			s.gameState,
			s.resourceManager,
			s.lawnGridSystem,
			s.sunSpawnSystem,
			s.gameState.CurrentLevel,
		)
		s.tutorialSystem.SetLevelSystem(s.levelSystem)
		log.Printf("[GameScene] Tutorial system activated for level %s", s.gameState.CurrentLevel.ID)

		// 强制性教学关卡禁用自动阳光生成
		if s.gameState.CurrentLevel.OpeningType == "tutorial" {
			s.sunSpawnSystem.Disable()
			log.Printf("[GameScene] Tutorial level: suns will be spawned by tutorial system")
		}

		// 加载教学字体
		ttFont, err := s.resourceManager.LoadFont("assets/fonts/SimHei.ttf", 28)
		if err != nil {
			log.Printf("FATAL: Failed to load tutorial font: %v", err)
		} else {
			s.tutorialFont = ttFont
			log.Printf("[GameScene] Loaded tutorial font: SimHei.ttf (28px)")
		}
	}

	// 保龄球关卡特殊处理
	if s.gameState.CurrentLevel.InitialSun == 0 {
		s.sunSpawnSystem.Disable()
		log.Printf("[GameScene] Bowling level (initialSun=0): sun spawn DISABLED")

		bowlingFont, err := s.resourceManager.LoadFont("assets/fonts/SimHei.ttf", config.BowlingTutorialTextFontSize)
		if err != nil {
			log.Printf("WARNING: Failed to load bowling tutorial font: %v", err)
		} else {
			s.bowlingTutorialFont = bowlingFont
			log.Printf("[GameScene] Loaded bowling tutorial font: SimHei.ttf (%.0fpx)",
				config.BowlingTutorialTextFontSize)
		}
	}

	// 初始化铲子交互系统
	s.shovelInteractionSystem = systems.NewShovelInteractionSystem(s.entityManager, s.gameState, rm)
	systems.SetShovelStateProvider(s)
	log.Printf("[GameScene] Initialized shovel interaction system")

	// 初始化强引导教学系统
	s.guidedTutorialSystem = systems.NewGuidedTutorialSystem(s.entityManager, s.gameState, rm)
	systems.SetGuidedTutorialShovelProvider(s)
	systems.SetGuidedTutorialStateProvider(s)
	log.Printf("[GameScene] Initialized guided tutorial system")

	// 初始化阶段转场系统
	s.levelPhaseSystem = systems.NewLevelPhaseSystem(s.entityManager, s.gameState, rm)
	log.Printf("[GameScene] Initialized level phase system")

	// 初始化传送带系统
	s.conveyorBeltSystem = systems.NewConveyorBeltSystem(s.entityManager, s.gameState, rm)
	log.Printf("[GameScene] Initialized conveyor belt system")

	// 初始化保龄球坚果滚动系统
	s.bowlingNutSystem = systems.NewBowlingNutSystem(s.entityManager, rm)
	log.Printf("[GameScene] Initialized bowling nut system")

	// 初始化疯狂戴夫对话系统
	s.daveDialogueSystem = systems.NewDaveDialogueSystem(s.entityManager, s.gameState, rm)
	log.Printf("[GameScene] Initialized Dave dialogue system")

	// 初始化选卡界面系统
	if s.gameState.CurrentLevel.EnableSeedSelection {
		seedChooserTitleFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 20)
		if err != nil {
			log.Printf("[GameScene] Warning: Failed to load seed chooser title font: %v", err)
		}
		seedChooserSunFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 14)
		if err != nil {
			log.Printf("[GameScene] Warning: Failed to load seed chooser sun font: %v", err)
		}

		s.seedChooserRenderSystem = systems.NewSeedChooserRenderSystem(
			s.entityManager,
			s.gameState,
			rm,
			s.reanimSystem,
			s.gameState.CurrentLevel,
			seedChooserTitleFont,
			seedChooserSunFont,
		)
		log.Printf("[GameScene] Initialized seed chooser render system")

		s.seedChooserInputSystem = systems.NewSeedChooserInputSystem(
			s.entityManager,
			s.gameState,
			s.seedChooserRenderSystem,
			s.gameState.CurrentLevel,
		)
		log.Printf("[GameScene] Initialized seed chooser input system")

		// 依赖注入：将输入系统注入渲染系统
		s.seedChooserRenderSystem.SetInputSystem(s.seedChooserInputSystem)
	}

	// 配置传送带系统
	s.configureConveyorBelt()
}

// configureConveyorBelt 根据关卡配置初始化传送带参数
func (s *GameScene) configureConveyorBelt() {
	if s.gameState.CurrentLevel == nil || s.gameState.CurrentLevel.ConveyorBelt == nil {
		return
	}

	conveyorConfig := s.gameState.CurrentLevel.ConveyorBelt
	if !conveyorConfig.Enabled {
		return
	}

	// 设置卡片生成间隔
	if conveyorConfig.GenerationInterval > 0 {
		s.conveyorBeltSystem.SetGenerationInterval(conveyorConfig.GenerationInterval)
	}

	// 设置卡片池
	if len(conveyorConfig.CardPool) > 0 {
		cardPool := make([]systems.CardPoolEntry, len(conveyorConfig.CardPool))
		for i, entry := range conveyorConfig.CardPool {
			cardPool[i] = systems.CardPoolEntry{
				Type:   entry.Type,
				Weight: entry.Weight,
			}
		}
		s.conveyorBeltSystem.SetCardPool(cardPool)
	}

	// 设置动态调节配置
	if len(conveyorConfig.PhaseConfigs) > 0 || conveyorConfig.DynamicAdjustment != nil {
		s.conveyorBeltSystem.SetDynamicConfig(
			conveyorConfig.PhaseConfigs,
			conveyorConfig.DynamicAdjustment,
		)
	}

	log.Printf("[GameScene] Conveyor belt configured: interval=%.1fs, pool=%d entries, phases=%d",
		conveyorConfig.GenerationInterval, len(conveyorConfig.CardPool), len(conveyorConfig.PhaseConfigs))

	// 加载传送带卡片渲染资源
	s.loadConveyorCardResources()
}

// setupSystemCallbacks 设置系统间回调
func (s *GameScene) setupSystemCallbacks() {
	if s.gameState.CurrentLevel == nil {
		return
	}

	// 设置强引导教学转场回调
	s.guidedTutorialSystem.SetTransitionCallback(func() {
		log.Printf("[GameScene] GuidedTutorial transition callback triggered")
		s.levelPhaseSystem.StartPhaseTransition(1, 2)
	})

	// 设置阶段转场系统回调
	s.levelPhaseSystem.SetOnDisableGuidedTutorial(func() {
		log.Printf("[GameScene] Disabling guided tutorial mode")
		s.guidedTutorialSystem.SetActive(false)

		// 恢复鼠标和铲子卡槽状态
		if s.shovelSelected {
			s.SetShovelSelected(false)
			log.Printf("[GameScene] Auto-deselected shovel after guided tutorial completed")
		}
		ebiten.SetCursorMode(ebiten.CursorModeVisible)
		if s.shovelInteractionSystem != nil {
			s.shovelInteractionSystem.ClearHighlight()
		}
	})

	s.levelPhaseSystem.SetOnActivateBowling(func() {
		log.Printf("[GameScene] Activating bowling phase")
		if s.conveyorBeltSystem != nil {
			s.conveyorBeltSystem.Activate()
		}
		if s.plantPreviewRenderSystem != nil {
			s.plantPreviewRenderSystem.SetRedLineEnabled(true)
		}
		if s.levelSystem != nil {
			s.levelSystem.ResumeWaveTiming()
			log.Printf("[GameScene] Wave timing resumed for bowling phase")
		}
		s.initLawnmowers()
		log.Printf("[GameScene] Lawnmowers initialized for bowling phase")
	})
}

// initLevelEntities 初始化关卡实体（预设植物、Dave对话等）
func (s *GameScene) initLevelEntities(rm *game.ResourceManager) {
	if s.gameState.CurrentLevel == nil {
		return
	}

	// 如果有战斗存档，跳过预设植物生成
	if !s.hasBattleSave {
		s.spawnPresetPlants()
	} else {
		log.Printf("[GameScene] Skipping preset plants spawn (will restore from battle save)")
	}

	// 创建开场Dave对话
	if len(s.gameState.CurrentLevel.PresetPlants) > 0 && !s.hasBattleSave {
		log.Printf("[GameScene] Level has preset plants, creating opening Dave dialogue")

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
			rm,
			openingDialogueKeys,
			func() {
				log.Printf("[GameScene] Opening Dave dialogue completed, activating guided tutorial mode")
				s.guidedTutorialSystem.SetActive(true)
			},
		)

		if err != nil {
			log.Printf("[GameScene] ERROR: Failed to create opening Dave entity: %v", err)
			s.guidedTutorialSystem.SetActive(true)
		} else {
			log.Printf("[GameScene] Opening Dave entity created: %d", daveEntity)
		}
	}
}
