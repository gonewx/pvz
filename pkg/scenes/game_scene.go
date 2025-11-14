package scenes

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/modules"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/decker502/pvz/pkg/systems/behavior"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// GameScene represents the main gameplay screen.
// This is where the actual Plants vs Zombies gameplay will occur.
// It manages the game state, UI elements, and the ECS system.
type GameScene struct {
	resourceManager *game.ResourceManager
	sceneManager    *game.SceneManager
	gameState       *game.GameState // Global game state (阳光、关卡进度等)

	// UI Image Resources
	background   *ebiten.Image // Lawn background (草坪背景)
	seedBank     *ebiten.Image // Plant selection bar (植物选择栏)
	sunCounterBG *ebiten.Image // Sun counter background (阳光计数器背景)
	shovelSlot   *ebiten.Image // Shovel slot background (铲子槽位背景)
	shovel       *ebiten.Image // Shovel icon (铲子图标)
	// Progress bar resources (进度条资源)
	flagMeter     *ebiten.Image // Flag meter background (进度条背景，2行切片)
	flagMeterProg *ebiten.Image // Flag meter progress bar (进度条填充)
	flagMeterFlag *ebiten.Image // Flag meter flags/parts (进度条标志，3列切片)

	// Story 8.2 QA改进：草皮叠加层（随动画进度渐进显示）
	sodRowImage        *ebiten.Image // 草皮叠加图片（sod1row.jpg 或 sod3row.jpg）
	soddedBackground   *ebiten.Image // 已铺草皮完整背景（IMAGE_BACKGROUND1），用于 Level 1-4 双背景叠加
	soddingAnimDelay   float64       // 铺草皮动画延迟时间（秒）
	soddingAnimStarted bool          // 铺草皮动画是否已启动
	soddingAnimTimer   float64       // 铺草皮动画延迟计时器
	sodDebugPrinted    bool          // 草皮叠加图调试日志是否已打印

	// Story 8.6 QA修正: 预渲染草皮支持
	preSoddedImage *ebiten.Image // 预渲染的草皮图片(仅包含指定行的草皮)

	// 性能优化：缓存草皮渲染参数，避免每帧重复计算
	sodOverlayX float64 // 草皮世界坐标X（缓存）
	sodOverlayY float64 // 草皮世界坐标Y（缓存）
	sodWidth    int     // 草皮图片宽度（缓存）
	sodHeight   int     // 草皮图片高度（缓存）

	// Font Resources
	sunCounterFont *text.GoTextFace // Font for sun counter display
	plantCardFont  *text.GoTextFace // Font for plant card sun cost display

	// Camera and Animation
	cameraX            float64 // Camera X position (controls which part of background to show)
	maxCameraX         float64 // Maximum camera X position (rightmost edge of background)
	isIntroAnimPlaying bool    // Whether the intro animation is currently playing
	introAnimTimer     float64 // Timer for intro animation

	// ECS Framework and Systems
	entityManager     *ecs.EntityManager
	sunSpawnSystem    *systems.SunSpawnSystem
	sunMovementSystem *systems.SunMovementSystem
	lifetimeSystem    *systems.LifetimeSystem
	renderSystem      *systems.RenderSystem
	inputSystem       *systems.InputSystem
	// Story 6.3: Reanim 动画系统（替代旧的 AnimationSystem）
	reanimSystem        *systems.ReanimSystem
	sunCollectionSystem *systems.SunCollectionSystem

	// 植物选择栏模块（Story 3.1 架构优化：封装所有选卡功能）
	// 替代原有的分散系统：
	//   - plantCardSystem       *systems.PlantCardSystem       (已移至模块内部)
	//   - plantCardRenderSystem *systems.PlantCardRenderSystem (已移至模块内部)
	// 优点：
	//   - 高内聚：所有选卡功能封装在单一模块中
	//   - 低耦合：通过清晰的接口与其他系统交互
	//   - 可复用：支持在不同场景（游戏中、选卡界面、图鉴）使用
	plantSelectionModule *modules.PlantSelectionModule

	// Story 3.2: Plant Preview Systems
	plantPreviewSystem       *systems.PlantPreviewSystem
	plantPreviewRenderSystem *systems.PlantPreviewRenderSystem

	// Story 3.3: Lawn Grid System
	lawnGridSystem   *systems.LawnGridSystem // 草坪网格管理系统
	lawnGridEntityID ecs.EntityID            // 草坪网格实体ID

	// Story 3.4: Behavior System
	behaviorSystem *behavior.BehaviorSystem // 植物行为系统（向日葵生产阳光等）

	// Story 4.3: Physics System
	physicsSystem *systems.PhysicsSystem // 物理系统（碰撞检测）

	// Story 5.5: Level Management Systems
	levelSystem     *systems.LevelSystem     // 关卡管理系统
	waveSpawnSystem *systems.WaveSpawnSystem // 波次生成系统

	// Zombie Lane Transition System - 僵尸行转换系统
	zombieLaneTransitionSystem *systems.ZombieLaneTransitionSystem // 僵尸移动到目标行系统

	// Story 7.2: Particle System
	particleSystem *systems.ParticleSystem // 粒子系统（粒子特效）

	// 方案A+：Flash Effect System
	flashEffectSystem *systems.FlashEffectSystem // 闪烁效果系统（僵尸受击闪烁）

	// Story 8.2: Tutorial System
	tutorialSystem *systems.TutorialSystem // 教学系统（关卡 1-1 教学引导）
	tutorialFont   interface{}             // 教学文本字体（*utils.BitmapFont 或 *text.GoTextFace）

	// Story 8.2 QA改进：完整的铺草皮动画系统
	soddingSystem *systems.SoddingSystem // 铺草皮动画系统（SodRoll 滚动动画）

	// Story 8.3: Camera and Opening Animation Systems
	cameraSystem  *systems.CameraSystem           // 镜头控制系统（镜头移动、缓动）
	openingSystem *systems.OpeningAnimationSystem // 开场动画系统（僵尸预告、跳过）

	// Story 8.3 + 8.4重构: Reward Animation System (完全封装奖励流程)
	// 内部自动管理卡片包动画、粒子效果、奖励面板渲染等所有细节
	rewardSystem *systems.RewardAnimationSystem

	// Button Systems (按钮系统 - ECS 架构)
	buttonSystem       *systems.ButtonSystem       // 按钮交互系统
	buttonRenderSystem *systems.ButtonRenderSystem // 按钮渲染系统
	menuButtonEntity   ecs.EntityID                // 菜单按钮实体ID

	// Story 10.1: Pause Menu Systems (暂停菜单系统)
	pauseMenuModule *modules.PauseMenuModule // 暂停菜单模块（Story 10.1）

	// Story 10.2: Lawnmower System (除草车系统)
	lawnmowerSystem *systems.LawnmowerSystem // 除草车系统（最后防线）

	// Story 11.2: Level Progress Bar (关卡进度条)
	levelProgressBarRenderSystem *systems.LevelProgressBarRenderSystem // 进度条渲染系统
	levelProgressBarEntity       ecs.EntityID                          // 进度条实体ID

	// Story 11.3: Final Wave Warning System (最后一波提示系统)
	finalWaveWarningSystem *systems.FinalWaveWarningSystem // 最后一波提示动画系统
}

// NewGameScene creates and returns a new GameScene instance.
// It loads all necessary UI resources and initializes the game scene.
//
// Parameters:
//   - rm: The ResourceManager instance used to load game resources.
//   - sm: The SceneManager instance used to switch between scenes.
//   - levelID: The level ID to load (e.g., "1-1", "1-2"). Will be converted to "data/levels/level-{id}.yaml"
//
// Returns:
//   - A pointer to the newly created GameScene.
//
// If any UI resources fail to load, the scene will use fallback rendering methods.
func NewGameScene(rm *game.ResourceManager, sm *game.SceneManager, levelID string) *GameScene {
	scene := &GameScene{
		resourceManager: rm,
		sceneManager:    sm,
		gameState:       game.GetGameState(), // Get global game state singleton
		// DEBUG: Skip intro animation for faster testing
		// Initialize camera at the leftmost position for intro animation
		// cameraX:            0,
		// isIntroAnimPlaying: true,
		cameraX:            config.GameCameraX, // 直接设置为游戏镜头位置，跳过开场动画
		isIntroAnimPlaying: false,              // 禁用开场动画
		introAnimTimer:     0,
	}

	// Load all UI resources
	scene.loadResources()

	// Story 6.3: Load all Reanim resources (XML and part images)
	// CRITICAL: Reanim resources are required for all entity animations.
	// If loading fails in production, log fatal error.
	// In test environments without assets, this will fail gracefully.
	if err := rm.LoadReanimResources(); err != nil {
		log.Printf("[GameScene] FATAL: Failed to load Reanim resources: %v", err)
		log.Printf("[GameScene] Game cannot function properly without animation resources")
		// Note: In production with real assets, this indicates a critical setup error.
		// The game will likely crash later when trying to access nil Reanim data.
		// In test environments, tests may pass if they don't use Reanim features.
	} else {
		log.Printf("[GameScene] Successfully loaded all Reanim resources")
	}

	// Initialize ECS framework
	scene.entityManager = ecs.NewEntityManager()

	// Story 5.5 & 8.1 & 8.6: Load level configuration FIRST (before creating systems that depend on it)
	// Story 8.6: Convert levelID to file path (e.g., "1-2" → "data/levels/level-1-2.yaml")
	// CRITICAL: This must happen before:
	//   1. LawnGridSystem (needs EnabledLanes)
	//   2. initPlantCardSystems() (needs AvailablePlants)
	//   3. WaveSpawnSystem (needs wave configuration)
	levelFilePath := fmt.Sprintf("data/levels/level-%s.yaml", levelID)
	levelConfig, err := config.LoadLevelConfig(levelFilePath)
	if err != nil {
		log.Printf("[GameScene] FATAL: Failed to load level config '%s': %v", levelFilePath, err)
		log.Printf("[GameScene] Game cannot start without level configuration")
	} else {
		scene.gameState.LoadLevel(levelConfig)
		log.Printf("[GameScene] Loaded level: %s (%d waves, %d plants available, enabled lanes: %v)",
			levelConfig.Name, len(levelConfig.Waves), len(levelConfig.AvailablePlants), levelConfig.EnabledLanes)
	}

	// Initialize systems
	scene.renderSystem = systems.NewRenderSystem(scene.entityManager)
	scene.sunMovementSystem = systems.NewSunMovementSystem(scene.entityManager)
	scene.lifetimeSystem = systems.NewLifetimeSystem(scene.entityManager)
	// TODO(Story 6.3): 迁移到 ReanimSystem
	// scene.animationSystem = systems.NewAnimationSystem(scene.entityManager)

	// Calculate sun collection target position from sun pool icon position
	// This ensures the suns fly to the exact center of the sun pool icon (not the text)
	sunCollectionTargetX := float64(config.SeedBankX + config.SunPoolOffsetX)
	sunCollectionTargetY := float64(config.SeedBankY + config.SunPoolOffsetY)

	// Story 3.3 & 8.1: Initialize lawn grid system and entity with enabled lanes
	// Now CurrentLevel is loaded, so we can read EnabledLanes correctly
	var enabledLanes []int
	if scene.gameState.CurrentLevel != nil {
		enabledLanes = scene.gameState.CurrentLevel.EnabledLanes
	}
	if len(enabledLanes) == 0 {
		enabledLanes = []int{1, 2, 3, 4, 5} // 默认所有行启用
	}
	scene.lawnGridSystem = systems.NewLawnGridSystem(scene.entityManager, enabledLanes)
	scene.lawnGridEntityID = scene.entityManager.CreateEntity()
	scene.entityManager.AddComponent(scene.lawnGridEntityID, &components.LawnGridComponent{})
	log.Printf("[GameScene] Initialized lawn grid system (Entity ID: %d) with enabled lanes: %v", scene.lawnGridEntityID, enabledLanes)

	// Story 6.3: Initialize Reanim system (must be before InputSystem)
	scene.reanimSystem = systems.NewReanimSystem(scene.entityManager)
	log.Printf("[GameScene] Initialized Reanim system")

	// Story 13.6: 设置配置管理器
	if configManager := rm.GetReanimConfigManager(); configManager != nil {
		scene.reanimSystem.SetConfigManager(configManager)
	}

	// ✅ 修复：设置 ReanimSystem 引用，以便 RenderSystem 调用 GetRenderData()
	scene.renderSystem.SetReanimSystem(scene.reanimSystem)

	// Initialize input system with sun counter target position and lawn grid system (Story 2.4 + Story 3.3)
	// Story 6.3: Pass reanimSystem to InputSystem for plant animation initialization
	scene.inputSystem = systems.NewInputSystem(
		scene.entityManager,
		rm,
		scene.gameState,
		scene.reanimSystem,     // Story 6.3: Reanim 系统
		sunCollectionTargetX,   // sunCounterX - 阳光计数器X坐标
		sunCollectionTargetY,   // sunCounterY - 阳光计数器Y坐标
		scene.lawnGridSystem,   // Story 3.3: 草坪网格系统
		scene.lawnGridEntityID, // Story 3.3: 草坪网格实体ID
	)

	// Initialize sun collection system with the same target position
	scene.sunCollectionSystem = systems.NewSunCollectionSystem(
		scene.entityManager,
		scene.gameState,      // 传入 GameState 以便在阳光到达时增加数值
		sunCollectionTargetX, // targetX
		sunCollectionTargetY, // targetY
	)

	// Initialize sun spawn system with lawn area parameters
	scene.sunSpawnSystem = systems.NewSunSpawnSystem(
		scene.entityManager,
		rm,
		250.0, // minX - 草坪左边界
		900.0, // maxX - 草坪右边界
		100.0, // minTargetY - 草坪上边界
		550.0, // maxTargetY - 草坪下边界
	)

	// Story 8.2 QA改进：关卡加载后，加载草皮相关资源
	scene.loadSoddingResources()

	// Story 3.1: Initialize plant card systems
	// NOW CurrentLevel is loaded, so availablePlants will be read correctly
	scene.initPlantCardSystems(rm)

	// Story 3.2: Initialize plant preview systems
	// PlantPreviewRenderSystem 需要引用 PlantPreviewSystem 来获取两个渲染位置
	// Story 8.1: PlantPreviewSystem 需要 LawnGridSystem 来检查行是否启用
	scene.plantPreviewSystem = systems.NewPlantPreviewSystem(scene.entityManager, scene.gameState, scene.lawnGridSystem)
	// 修复: 使用静态图像预览,不需要 ReanimSystem
	scene.plantPreviewRenderSystem = systems.NewPlantPreviewRenderSystem(scene.entityManager, scene.plantPreviewSystem)

	// Story 3.4: Initialize behavior system (sunflower sun production, etc.)
	// Story 14.3: Epic 14 - Removed ReanimSystem dependency, using AnimationCommand component
	// Story 5.5: Pass GameState for zombie death counting
	// Bug Fix: Pass LawnGridSystem for plant death grid release
	scene.behaviorSystem = behavior.NewBehaviorSystem(
		scene.entityManager,
		rm,
		scene.gameState,
		scene.lawnGridSystem,
		scene.lawnGridEntityID,
	)
	log.Printf("[GameScene] Initialized behavior system for plant behaviors")

	// Story 4.3: Initialize physics system (collision detection)
	scene.physicsSystem = systems.NewPhysicsSystem(scene.entityManager, rm)
	log.Printf("[GameScene] Initialized physics system for collision detection")

	// Story 5.5: Initialize level management systems
	// 1. Create WaveSpawnSystem (LevelSystem depends on it)
	// Story 14.3: Epic 14 - Removed ReanimSystem dependency
	scene.waveSpawnSystem = systems.NewWaveSpawnSystem(scene.entityManager, rm, scene.gameState.CurrentLevel, scene.gameState)
	log.Printf("[GameScene] Initialized wave spawn system")

	// Pre-spawn all zombies for the level (they will be activated wave by wave)
	// 关卡僵尸应该是根据配置,在开始前就已经生成好了,以便开场前展现给用户
	if scene.gameState.CurrentLevel != nil && len(scene.gameState.CurrentLevel.Waves) > 0 {
		totalZombies := scene.waveSpawnSystem.PreSpawnAllWaves()
		log.Printf("[GameScene] Pre-spawned %d zombies for showcase", totalZombies)

		// ❌ 不应该在预生成时计数，只有在激活僵尸时才计数
		// 预生成的僵尸处于未激活状态，不参与游戏逻辑
		// scene.gameState.IncrementZombiesSpawned(totalZombies) // 已删除
	}

	// Story 8.3: Create CameraSystem (always create, used by opening animation)
	scene.cameraSystem = systems.NewCameraSystem(scene.entityManager, scene.gameState)
	log.Printf("[GameScene] Initialized camera system")

	// Story 7.2: Initialize particle system (must be before RewardAnimationSystem)
	// Story 7.4: Added ResourceManager parameter for loading particle images
	scene.particleSystem = systems.NewParticleSystem(scene.entityManager, scene.resourceManager)
	log.Printf("[GameScene] Initialized particle system for visual effects")

	// Story 8.3 + 8.4重构: Create RewardAnimationSystem (完全封装，无需单独创建面板渲染系统)
	// RewardAnimationSystem内部自动创建和管理RewardPanelRenderSystem
	scene.rewardSystem = systems.NewRewardAnimationSystem(scene.entityManager, scene.gameState, rm, scene.sceneManager, scene.reanimSystem, scene.particleSystem, scene.renderSystem)
	log.Printf("[GameScene] Initialized reward animation system (fully encapsulated)")

	// Story 8.3: Create OpeningAnimationSystem (conditionally, may return nil)
	scene.openingSystem = systems.NewOpeningAnimationSystem(scene.entityManager, scene.gameState, rm, levelConfig, scene.cameraSystem)
	if scene.openingSystem != nil {
		log.Printf("[GameScene] Initialized opening animation system")
	} else {
		log.Printf("[GameScene] Skipping opening animation system (tutorial/skip/special level)")
	}

	// Story 10.2: Create LawnmowerSystem (除草车系统 - 最后防线)
	// Story 10.3: 传递 ReanimSystem 用于播放僵尸死亡动画
	scene.lawnmowerSystem = systems.NewLawnmowerSystem(scene.entityManager, rm, scene.gameState)
	log.Printf("[GameScene] Initialized lawnmower system")

	// Story 10.2: 除草车实体将在铺草皮动画完成后创建（见铺草皮回调）
	// 原版行为：草皮铺完后才显示除草车

	// 2. Create LevelSystem (需要 RewardAnimationSystem 和 LawnmowerSystem)
	// Story 14.3: Epic 14 - Removed ReanimSystem dependency
	scene.levelSystem = systems.NewLevelSystem(scene.entityManager, scene.gameState, scene.waveSpawnSystem, rm, scene.rewardSystem, scene.lawnmowerSystem)
	log.Printf("[GameScene] Initialized level system")

	// Story 11.3: Create FinalWaveWarningSystem (最后一波提示系统)
	scene.finalWaveWarningSystem = systems.NewFinalWaveWarningSystem(scene.entityManager)
	log.Printf("[GameScene] Initialized final wave warning system")

	// 3. Create ZombieLaneTransitionSystem (僵尸行转换系统)
	scene.zombieLaneTransitionSystem = systems.NewZombieLaneTransitionSystem(scene.entityManager)
	log.Printf("[GameScene] Initialized zombie lane transition system")

	// 方案A+：Initialize flash effect system
	scene.flashEffectSystem = systems.NewFlashEffectSystem(scene.entityManager)
	log.Printf("[GameScene] Initialized flash effect system for hit feedback")

	// Story 8.2: Initialize tutorial system (if this is a tutorial level)
	if scene.gameState.CurrentLevel != nil && scene.gameState.CurrentLevel.OpeningType == "tutorial" && len(scene.gameState.CurrentLevel.TutorialSteps) > 0 {
		scene.tutorialSystem = systems.NewTutorialSystem(scene.entityManager, scene.gameState, scene.resourceManager, scene.lawnGridSystem, scene.sunSpawnSystem, scene.waveSpawnSystem, scene.gameState.CurrentLevel)
		log.Printf("[GameScene] Tutorial system activated for level %s", scene.gameState.CurrentLevel.ID)

		// 禁用自动阳光生成（第一次收集阳光后由 TutorialSystem 启用）
		scene.sunSpawnSystem.Disable()

		// Load tutorial font (使用简体中文黑体字体 SimHei.ttf)
		ttFont, err := scene.resourceManager.LoadFont("assets/fonts/SimHei.ttf", 28)
		if err != nil {
			log.Printf("FATAL: Failed to load tutorial font SimHei.ttf: %v", err)
		} else {
			scene.tutorialFont = ttFont
			log.Printf("[GameScene] Loaded tutorial font: SimHei.ttf (28px)")
		}

		// Story 8.2: 教学关卡不预生成阳光
		// 阳光由教学系统在特定步骤触发生成（种植第一个豌豆射手后、收集第一颗阳光后）
		log.Printf("[GameScene] Tutorial level: suns will be spawned by tutorial system")
	}

	// Story 8.2 QA改进：初始化铺草皮动画系统
	scene.soddingSystem = systems.NewSoddingSystem(scene.entityManager, scene.resourceManager)
	log.Printf("[GameScene] Initialized sodding animation system")

	// 按钮系统初始化（ECS 架构）
	scene.buttonSystem = systems.NewButtonSystem(scene.entityManager)
	scene.buttonRenderSystem = systems.NewButtonRenderSystem(scene.entityManager)
	log.Printf("[GameScene] Initialized button systems")

	// 创建菜单按钮实体
	scene.initMenuButton(rm)

	// Story 10.1: 初始化暂停菜单系统
	scene.initPauseMenuModule(rm)

	// Story 11.2: 初始化关卡进度条系统
	scene.initProgressBar(rm)

	return scene
}

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

// initPauseMenu 初始化暂停菜单（ECS 架构）
// Story 10.1: 创建暂停菜单实体和三个按钮
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
			},
			OnResumeMusic: func() {
				// TODO: 恢复 BGM（当BGM系统实现后）
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

// loadResources loads all UI images required for the game scene.
// If a resource fails to load, it logs a warning but continues.
// The Draw method will use fallback rendering for missing resources.
func (s *GameScene) loadResources() {
	// Story 8.2 QA改进：根据关卡配置加载背景
	// 如果关卡配置了特定背景，使用配置的背景；否则使用默认背景
	backgroundImageID := "IMAGE_BACKGROUND1"
	if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.BackgroundImage != "" {
		backgroundImageID = s.gameState.CurrentLevel.BackgroundImage
		log.Printf("[GameScene] 使用关卡配置的背景: %s", backgroundImageID)
	}

	// Load lawn background
	bg, err := s.resourceManager.LoadImageByID(backgroundImageID)
	if err != nil {
		log.Printf("Warning: Failed to load lawn background %s: %v", backgroundImageID, err)
		log.Printf("Will use fallback solid color background")
	} else {
		s.background = bg
		// Calculate maximum camera position (rightmost edge)
		bgWidth := bg.Bounds().Dx()
		s.maxCameraX = float64(bgWidth - WindowWidth)
		if s.maxCameraX < 0 {
			s.maxCameraX = 0 // Background is smaller than window
		}
	}

	// Load seed bank (植物选择栏背景)
	seedBank, err := s.resourceManager.LoadImageByID("IMAGE_SEEDBANK")
	if err != nil {
		log.Printf("Warning: Failed to load seed bank image: %v", err)
		log.Printf("Will use fallback rendering for seed bank")
	} else {
		s.seedBank = seedBank
	}

	// Load shovel slot background
	shovelSlot, err := s.resourceManager.LoadImageByID("IMAGE_SHOVELBANK")
	if err != nil {
		log.Printf("Warning: Failed to load shovel slot: %v", err)
	} else {
		s.shovelSlot = shovelSlot
	}

	// Load shovel icon
	shovel, err := s.resourceManager.LoadImageByID("IMAGE_SHOVEL")
	if err != nil {
		log.Printf("Warning: Failed to load shovel icon: %v", err)
	} else {
		s.shovel = shovel
	}

	// Load font for sun counter (使用黑体)
	font, err := s.resourceManager.LoadFont("assets/fonts/SimHei.ttf", config.SunCounterFontSize)
	if err != nil {
		log.Printf("Warning: Failed to load sun counter font: %v", err)
		log.Printf("Will use fallback debug text rendering")
	} else {
		s.sunCounterFont = font
	}

	// Load font for plant card sun cost (使用黑体，字体大小从配置读取)
	cardFont, err := s.resourceManager.LoadFont("assets/fonts/SimHei.ttf", float64(config.PlantCardSunCostFontSize))
	if err != nil {
		log.Printf("Warning: Failed to load plant card font: %v", err)
		log.Printf("Will use fallback debug text rendering for card cost")
	} else {
		s.plantCardFont = cardFont
	}

	// Load progress bar resources (进度条资源)
	flagMeter, err := s.resourceManager.LoadImageByID("IMAGE_FLAGMETER")
	if err != nil {
		log.Printf("Warning: Failed to load progress bar background: %v", err)
	} else {
		s.flagMeter = flagMeter
	}

	flagMeterProg, err := s.resourceManager.LoadImageByID("IMAGE_FLAGMETERLEVELPROGRESS")
	if err != nil {
		log.Printf("Warning: Failed to load progress bar fill: %v", err)
	} else {
		s.flagMeterProg = flagMeterProg
	}

	flagMeterFlag, err := s.resourceManager.LoadImageByID("IMAGE_FLAGMETERPARTS")
	if err != nil {
		log.Printf("Warning: Failed to load progress bar flags: %v", err)
	} else {
		s.flagMeterFlag = flagMeterFlag
	}

	// Note: Sun counter background is drawn procedurally for now
	// A dedicated image can be loaded here in the future if needed
	// Menu button resources are now loaded via ButtonFactory (ECS architecture)
}

// loadSoddingResources loads sodding animation resources after level config is loaded.
// Story 8.2 QA改进：铺草皮动画资源加载
// Story 8.3: 添加奖励面板资源加载
//
// This method must be called AFTER the level configuration is loaded,
// because it depends on CurrentLevel.SodRowImage and CurrentLevel.ShowSoddingAnim.
func (s *GameScene) loadSoddingResources() {
	// Story 8.3: 加载 LoadingImages 资源组（包含按钮等 UI 资源）
	// 包含 IMAGE_SEEDCHOOSER_BUTTON 等资源
	if err := s.resourceManager.LoadResourceGroup("LoadingImages"); err != nil {
		log.Printf("Warning: Failed to load LoadingImages resources: %v", err)
	} else {
		log.Printf("[GameScene] 加载 UI 资源组成功 (LoadingImages)")
	}

	// Story 8.3: 加载奖励面板资源（延迟加载组）
	// 包含 AwardScreen_Back.jpg 等资源
	if err := s.resourceManager.LoadResourceGroup("DelayLoad_AwardScreen"); err != nil {
		log.Printf("Warning: Failed to load reward panel resources: %v", err)
	} else {
		log.Printf("[GameScene] 加载奖励面板资源组成功 (DelayLoad_AwardScreen)")
	}

	// 检查是否需要加载未铺草皮背景资源组
	if s.gameState.CurrentLevel != nil &&
		(s.gameState.CurrentLevel.BackgroundImage == "IMAGE_BACKGROUND1UNSODDED" ||
			s.gameState.CurrentLevel.SodRowImage != "") {
		if err := s.resourceManager.LoadResourceGroup("DelayLoad_BackgroundUnsodded"); err != nil {
			log.Printf("Warning: Failed to load BackgroundUnsodded resource group: %v", err)
		} else {
			log.Printf("[GameScene] 加载未铺草皮背景资源组成功")
		}
	}

	// 重新加载背景（如果需要切换到未铺草皮背景）
	if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.BackgroundImage == "IMAGE_BACKGROUND1UNSODDED" {
		bg, err := s.resourceManager.LoadImageByID("IMAGE_BACKGROUND1UNSODDED")
		if err != nil {
			log.Printf("Warning: Failed to load unsodded background: %v", err)
		} else {
			s.background = bg
			// 重新计算摄像机边界
			bgWidth := bg.Bounds().Dx()
			s.maxCameraX = float64(bgWidth - WindowWidth)
			if s.maxCameraX < 0 {
				s.maxCameraX = 0
			}
			log.Printf("[GameScene] 切换到未铺草皮背景")
		}
	}

	// Story 8.2 QA改进：加载草皮叠加图片（用于动画播放时的叠加渲染）
	// 重构简化：所有叠加层X坐标从 0 开始，Y坐标对齐到草皮行位置
	// - 启用行为连续3行（如 [2,3,4]）→ 使用 IMAGE_SOD3ROW（整体效果，无边缘）
	// - 启用行为5行，动画行为 [1,5]（Level 1-4）→ 双背景叠加（IMAGE_BACKGROUND1）
	// - 其他情况 → 使用 IMAGE_SOD1ROW（逐行渲染）
	// - 两阶段渲染（Level 1-2）→ 初始使用 IMAGE_SOD1ROW，动画时使用 IMAGE_SOD3ROW
	if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.ShowSoddingAnim {
		enabledLanes := s.gameState.CurrentLevel.EnabledLanes
		animLanes := s.gameState.CurrentLevel.SoddingAnimLanes
		if len(animLanes) == 0 {
			animLanes = enabledLanes
		}

		// 检测启用行是否为连续3行
		isConsecutive3Rows := len(enabledLanes) == 3 &&
			enabledLanes[1] == enabledLanes[0]+1 &&
			enabledLanes[2] == enabledLanes[1]+1

		// 检测是否为 Level 1-4 场景（5行启用，动画行为 [1,5]）
		isLevel14Pattern := len(enabledLanes) == 5 &&
			len(animLanes) == 2 &&
			animLanes[0] == 1 && animLanes[1] == 5

		// 检测是否需要两阶段渲染（配置了 sodRowImageAnim）
		hasTwoStageRendering := s.gameState.CurrentLevel.SodRowImageAnim != ""

		if isLevel14Pattern && hasTwoStageRendering && s.gameState.CurrentLevel.SodRowImageAnim == "IMAGE_BACKGROUND1" {
			// Level 1-4: 双背景叠加模式，使用 sodRowImageAnim="IMAGE_BACKGROUND1" 作为叠加层
			log.Printf("[GameScene] 检测到 Level 1-4 双背景叠加模式：sodRowImageAnim=%s", s.gameState.CurrentLevel.SodRowImageAnim)
			log.Printf("[GameScene] 底层=未铺草皮+预渲染(IMAGE_SOD3ROW), 叠加层=IMAGE_BACKGROUND1")

			// 加载已铺草皮完整背景（IMAGE_BACKGROUND1）作为叠加层
			soddedBg, err := s.resourceManager.LoadImageByID("IMAGE_BACKGROUND1")
			if err != nil {
				log.Printf("Warning: Failed to load IMAGE_BACKGROUND1: %v", err)
			} else {
				s.soddedBackground = soddedBg
				log.Printf("[GameScene] ✅ 加载已铺草皮背景作为叠加层: IMAGE_BACKGROUND1")
			}

			// 重构简化：叠加背景从 (0,0) 开始（与底层背景完全对齐）
			s.sodOverlayX = 0
			s.sodOverlayY = 0
			log.Printf("[GameScene] 双背景叠加模式：叠加层从 (0,0) 开始")

		} else if isLevel14Pattern {
			// Level 1-4（旧版兼容）: 双背景叠加模式，加载 IMAGE_BACKGROUND1 作为叠加层
			log.Printf("[GameScene] 检测到 Level 1-4 模式：双背景叠加（底层=未铺草皮+预渲染，叠加层=IMAGE_BACKGROUND1）")

			// 加载已铺草皮完整背景（IMAGE_BACKGROUND1）
			soddedBg, err := s.resourceManager.LoadImageByID("IMAGE_BACKGROUND1")
			if err != nil {
				log.Printf("Warning: Failed to load IMAGE_BACKGROUND1: %v", err)
			} else {
				s.soddedBackground = soddedBg
				log.Printf("[GameScene] ✅ 加载已铺草皮背景: IMAGE_BACKGROUND1")
			}

			// 重构简化：叠加背景从 (0,0) 开始（与底层背景完全对齐）
			s.sodOverlayX = 0
			s.sodOverlayY = 0
			log.Printf("[GameScene] 双背景叠加模式：叠加层从 (0,0) 开始")

		} else if hasTwoStageRendering && isConsecutive3Rows {
			// 两阶段渲染模式（Level 1-2）
			// 阶段1（初始化）：使用 sodRowImage（IMAGE_SOD1ROW）预渲染指定行
			// 阶段2（动画播放）：使用 sodRowImageAnim（IMAGE_SOD3ROW）叠加渲染
			log.Printf("[GameScene] 检测到两阶段渲染模式：初始=%s, 动画=%s",
				s.gameState.CurrentLevel.SodRowImage, s.gameState.CurrentLevel.SodRowImageAnim)

			// 加载动画阶段使用的草皮图片（IMAGE_SOD3ROW）
			sod3RowImage, err := s.resourceManager.LoadImageWithAlphaMask(
				"assets/images/sod3row.jpg",
				"assets/images/sod3row_.png",
			)
			if err != nil {
				log.Printf("Warning: Failed to composite sod3row image: %v", err)
			} else {
				s.sodRowImage = sod3RowImage
				log.Printf("[GameScene] ✅ 合成草皮叠加图片 (RGB + Alpha): IMAGE_SOD3ROW (动画阶段)")

				// 性能优化：缓存草皮图片尺寸
				sodBounds := sod3RowImage.Bounds()
				s.sodWidth = sodBounds.Dx()
				s.sodHeight = sodBounds.Dy()

				// 计算草皮叠加层Y坐标（对齐到第一行的顶部）
				firstLane := enabledLanes[0]
				sodOverlayY := config.GridWorldStartY + float64(firstLane-1)*config.CellHeight

				// X坐标从0开始
				s.sodOverlayX = 0
				s.sodOverlayY = sodOverlayY
				log.Printf("[GameScene] 草皮叠加层: 位置(0, %.1f) 尺寸(%dx%d)", sodOverlayY, s.sodWidth, s.sodHeight)
			}

		} else if isConsecutive3Rows {
			// 连续3行：使用 IMAGE_SOD3ROW（整体草皮，无边缘分界线）
			sod3RowImage, err := s.resourceManager.LoadImageWithAlphaMask(
				"assets/images/sod3row.jpg",
				"assets/images/sod3row_.png",
			)
			if err != nil {
				log.Printf("Warning: Failed to composite sod row image: %v", err)
			} else {
				s.sodRowImage = sod3RowImage
				log.Printf("[GameScene] ✅ 合成草皮叠加图片 (RGB + Alpha): IMAGE_SOD3ROW (启用行: %v)", enabledLanes)

				// 性能优化：缓存草皮图片尺寸
				sodBounds := sod3RowImage.Bounds()
				s.sodWidth = sodBounds.Dx()
				s.sodHeight = sodBounds.Dy()

				// 计算草皮叠加层Y坐标（对齐到第一行的顶部）
				firstLane := enabledLanes[0]
				sodOverlayY := config.GridWorldStartY + float64(firstLane-1)*config.CellHeight

				// X坐标从0开始
				s.sodOverlayX = 0
				s.sodOverlayY = sodOverlayY
				log.Printf("[GameScene] 草皮叠加层: 位置(0, %.1f) 尺寸(%dx%d)", sodOverlayY, s.sodWidth, s.sodHeight)
			}
		} else {
			// 其他情况：使用 IMAGE_SOD1ROW（单行草皮）
			sod1RowImage, err := s.resourceManager.LoadImageWithAlphaMask(
				"assets/images/sod1row.jpg",
				"assets/images/sod1row_.png",
			)
			if err != nil {
				log.Printf("Warning: Failed to composite sod row image: %v", err)
			} else {
				s.sodRowImage = sod1RowImage
				log.Printf("[GameScene] ✅ 合成草皮叠加图片 (RGB + Alpha): IMAGE_SOD1ROW (启用行: %v)", enabledLanes)

				// 性能优化：缓存草皮图片尺寸
				sodBounds := sod1RowImage.Bounds()
				s.sodWidth = sodBounds.Dx()
				s.sodHeight = sodBounds.Dy()

				// 计算草皮叠加层Y坐标（对齐到动画行的顶部）
				// 单行模式下，使用第一个动画行的位置
				firstAnimLane := animLanes[0]
				sodOverlayY := config.GridWorldStartY + float64(firstAnimLane-1)*config.CellHeight

				// X坐标从0开始
				s.sodOverlayX = 0
				s.sodOverlayY = sodOverlayY
				log.Printf("[GameScene] 草皮叠加层: 位置(0, %.1f) 尺寸(%dx%d)", sodOverlayY, s.sodWidth, s.sodHeight)
			}
		}

		// 启动铺草皮动画
		s.soddingAnimDelay = s.gameState.CurrentLevel.SoddingAnimDelay
		s.soddingAnimStarted = false
		s.soddingAnimTimer = 0
		log.Printf("[GameScene] 设置铺草皮动画延迟: %.1f 秒", s.soddingAnimDelay)
	}

	// Story 8.6 QA修正 + 统一草皮渲染重构：
	// 为所有启用的行预渲染草皮到背景副本，用于双背景叠加渲染
	//
	// 设计思路（两阶段渲染模式 - Level 1-2）：
	// - 底层背景：未铺草皮背景 + preSoddedLanes 草皮（IMAGE_SOD1ROW）
	// - 叠加层：未铺草皮背景 + 所有启用行草皮（sodRowImageAnim 如 IMAGE_SOD3ROW）
	// - 动画播放时：叠加层渐进显示，覆盖底层的 IMAGE_SOD1ROW，展现完整的 IMAGE_SOD3ROW
	//
	// Level 1-3: preSoddedLanes=[2,3,4], ShowSoddingAnim=false
	// - 直接渲染3行草皮到背景，无需动画
	if s.gameState.CurrentLevel != nil && s.background != nil && (s.gameState.CurrentLevel.ShowSoddingAnim || len(s.gameState.CurrentLevel.PreSoddedLanes) > 0) {
		enabledLanes := s.gameState.CurrentLevel.EnabledLanes
		preSoddedLanes := s.gameState.CurrentLevel.PreSoddedLanes
		hasTwoStageRendering := s.gameState.CurrentLevel.SodRowImageAnim != ""

		// 步骤1：预渲染底层背景的草皮（preSoddedLanes）
		if len(preSoddedLanes) > 0 {
			// 检查预铺行是否是连续的3行（第2,3,4行）
			isConsecutive3Rows := len(preSoddedLanes) == 3 &&
				preSoddedLanes[0] == 2 && preSoddedLanes[1] == 3 && preSoddedLanes[2] == 4

			// 根据配置的 sodRowImage 选择使用的草皮图片
			var sodRowRGB *ebiten.Image
			var sodImageID string
			var err error

			if s.gameState.CurrentLevel.SodRowImage == "IMAGE_SOD3ROW" && isConsecutive3Rows {
				// 使用 IMAGE_SOD3ROW（3行整体草皮）
				sodRowRGB, err = s.resourceManager.LoadImageWithAlphaMask(
					"assets/images/sod3row.jpg",
					"assets/images/sod3row_.png",
				)
				sodImageID = "IMAGE_SOD3ROW"
			} else {
				// 默认使用 IMAGE_SOD1ROW（单行草皮）
				sodRowRGB, err = s.resourceManager.LoadImageWithAlphaMask(
					"assets/images/sod1row.jpg",
					"assets/images/sod1row_.png",
				)
				sodImageID = "IMAGE_SOD1ROW"
			}

			if err != nil {
				log.Printf("[GameScene] Error: 无法加载草皮图片 %s: %v", sodImageID, err)
				return
			}

			log.Printf("[GameScene] 预渲染底层背景草皮: 预铺行=%v (使用 %s)", preSoddedLanes, sodImageID)

			// 根据草皮图片类型选择渲染方式
			if sodImageID == "IMAGE_SOD3ROW" && isConsecutive3Rows {
				// IMAGE_SOD3ROW：整体渲染3行草皮（无需循环）
				// 图片覆盖第2,3,4行，图片中心应该对齐到第3行（中间行）的中心
				sodBounds := sodRowRGB.Bounds()
				sodHeight := float64(sodBounds.Dy())

				// 计算中间行（第3行）的中心Y坐标
				middleLane := preSoddedLanes[1] // 第3行（索引1）
				middleRowCenterY := config.GridWorldStartY + float64(middleLane-1)*config.CellHeight + config.CellHeight/2.0

				// 草皮Y坐标 = 中间行中心 - 草皮高度的一半 + 偏移
				// 这样图片的中心对齐到第3行的中心，覆盖第2,3,4行
				dstY := middleRowCenterY - sodHeight/2.0 + config.SodOverlayOffsetY

				// 草皮X坐标
				dstX := config.GridWorldStartX + config.SodOverlayOffsetX

				// 整体绘制到底层背景
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(dstX, dstY)
				s.background.DrawImage(sodRowRGB, op)

				log.Printf("[GameScene] ✅ 底层背景预渲染第 %v 行草皮 (IMAGE_SOD3ROW 整体): 中心对齐第%d行, 位置(%.1f,%.1f)",
					preSoddedLanes, middleLane, dstX, dstY)
			} else {
				// IMAGE_SOD1ROW：逐行渲染单行草皮
				for _, lane := range preSoddedLanes {
					sodBounds := sodRowRGB.Bounds()
					sodHeight := float64(sodBounds.Dy())

					// 计算目标行的中心Y坐标
					rowCenterY := config.GridWorldStartY + float64(lane-1)*config.CellHeight + config.CellHeight/2.0

					// 草皮Y坐标 = 行中心 - 草皮高度的一半 + 偏移
					dstY := rowCenterY - sodHeight/2.0 + config.SodOverlayOffsetY

					// 草皮X坐标
					dstX := config.GridWorldStartX + config.SodOverlayOffsetX

					// 绘制到底层背景
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(dstX, dstY)
					s.background.DrawImage(sodRowRGB, op)

					log.Printf("[GameScene] ✅ 底层背景预渲染第 %d 行草皮 (IMAGE_SOD1ROW): 位置(%.1f,%.1f)", lane, dstX, dstY)
				}
			}
		}

		// 步骤2：预渲染叠加层背景（用于动画时渐进显示）
		// 创建新的背景图片副本（叠加层）
		bgBounds := s.background.Bounds()
		newBackground := ebiten.NewImage(bgBounds.Dx(), bgBounds.Dy())

		// 1. 绘制原始背景（现在已包含 preSoddedLanes 草皮）
		op := &ebiten.DrawImageOptions{}
		newBackground.DrawImage(s.background, op)

		// 2. 预渲染叠加层草皮
		// 设计原理：
		// - preSoddedLanes：控制底层背景预渲染哪些行（动画开始前就可见）
		// - 叠加层：始终包含所有启用行的草皮，用于动画时逐渐显示
		//
		// 示例：
		// Level 1-1: preSoddedLanes=[], enabledLanes=[3]
		//   → 底层无草皮，叠加层有第3行草皮，通过裁剪逐渐显示
		// Level 1-2: preSoddedLanes=[2,4], enabledLanes=[2,3,4]
		//   → 底层有2/4行草皮，叠加层有2/3/4行草皮，动画显示第3行
		lanesToPreRender := enabledLanes

		if len(lanesToPreRender) > 0 {
			// 检查是否是连续的3行
			isConsecutive3Rows := len(lanesToPreRender) == 3 &&
				lanesToPreRender[1] == lanesToPreRender[0]+1 &&
				lanesToPreRender[2] == lanesToPreRender[1]+1

			log.Printf("[GameScene] 预渲染叠加层草皮: 启用行=%v, 预铺行=%v, 实际预渲染=%v, 两阶段模式=%v",
				enabledLanes, preSoddedLanes, lanesToPreRender, hasTwoStageRendering)

			// 根据行数和两阶段模式选择渲染方式
			if hasTwoStageRendering && isConsecutive3Rows {
				// 两阶段模式：叠加层使用 IMAGE_SOD3ROW
				sod3RowRGB, err := s.resourceManager.LoadImageWithAlphaMask(
					"assets/images/sod3row.jpg",
					"assets/images/sod3row_.png",
				)
				if err != nil {
					log.Printf("[GameScene] Error: 无法加载3行草皮图片: %v", err)
					return
				}

				// 计算中间行的中心Y坐标
				middleLane := lanesToPreRender[1] // 中间行
				rowCenterY := config.GridWorldStartY + float64(middleLane-1)*config.CellHeight + config.CellHeight/2.0

				// 3行草皮图片的高度
				sodBounds := sod3RowRGB.Bounds()
				sodHeight := float64(sodBounds.Dy())

				// 草皮Y坐标 = 中间行中心 - 草皮高度的一半 + 偏移
				dstY := rowCenterY - sodHeight/2.0 + config.SodOverlayOffsetY

				// 草皮X坐标
				dstX := config.GridWorldStartX + config.SodOverlayOffsetX

				// 一次性绘制3行草皮到叠加层
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(dstX, dstY)
				newBackground.DrawImage(sod3RowRGB, op)

				log.Printf("[GameScene] ✅ 叠加层预渲染第 %v 行草皮 (IMAGE_SOD3ROW): 位置(%.1f,%.1f)", lanesToPreRender, dstX, dstY)

			} else if isConsecutive3Rows {
				// 连续3行但非两阶段模式：使用 IMAGE_SOD3ROW
				sod3RowRGB, err := s.resourceManager.LoadImageWithAlphaMask(
					"assets/images/sod3row.jpg",
					"assets/images/sod3row_.png",
				)
				if err != nil {
					log.Printf("[GameScene] Error: 无法加载3行草皮图片: %v", err)
					return
				}

				middleLane := lanesToPreRender[1]
				rowCenterY := config.GridWorldStartY + float64(middleLane-1)*config.CellHeight + config.CellHeight/2.0
				sodBounds := sod3RowRGB.Bounds()
				sodHeight := float64(sodBounds.Dy())
				dstY := rowCenterY - sodHeight/2.0 + config.SodOverlayOffsetY
				dstX := config.GridWorldStartX + config.SodOverlayOffsetX

				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(dstX, dstY)
				newBackground.DrawImage(sod3RowRGB, op)

				log.Printf("[GameScene] ✅ 使用 IMAGE_SOD3ROW 一次性预渲染第 %v 行草皮: 背景位置(%.1f,%.1f)", lanesToPreRender, dstX, dstY)

			} else {
				// 使用 IMAGE_SOD1ROW 循环渲染每行
				sod1RowRGB, err := s.resourceManager.LoadImageWithAlphaMask(
					"assets/images/sod1row.jpg",
					"assets/images/sod1row_.png",
				)
				if err != nil {
					log.Printf("[GameScene] Error: 无法加载单行草皮图片: %v", err)
					return
				}

				// 为每个需要预渲染的行绘制单行草皮
				for _, lane := range lanesToPreRender {
					sodBounds := sod1RowRGB.Bounds()
					sodHeight := float64(sodBounds.Dy())
					rowCenterY := config.GridWorldStartY + float64(lane-1)*config.CellHeight + config.CellHeight/2.0
					dstY := rowCenterY - sodHeight/2.0 + config.SodOverlayOffsetY
					dstX := config.GridWorldStartX + config.SodOverlayOffsetX

					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(dstX, dstY)
					newBackground.DrawImage(sod1RowRGB, op)

					log.Printf("[GameScene] ✅ 使用 IMAGE_SOD1ROW 预渲染第 %d 行草皮: 背景位置(%.1f,%.1f)", lane, dstX, dstY)
				}
			}
		}

		// 3. 保存预渲染背景副本（用于草皮叠加渲染）
		s.preSoddedImage = newBackground

		log.Printf("[GameScene] ✅ 创建预渲染背景副本用于草皮叠加 (preSoddedLanes: %v, 两阶段模式: %v)", preSoddedLanes, hasTwoStageRendering)
	}
}

// Update updates the game scene logic.
// deltaTime is the time elapsed since the last update in seconds.
//
// This method handles:
//   - Intro animation (camera scrolling left → right → center)
//   - ECS system updates (input, sun spawning, movement, collection, lifetime management)
//   - System execution order ensures correct game logic flow
//   - Story 10.1: Pause menu (只更新 UI 系统，跳过游戏逻辑)
func (s *GameScene) Update(deltaTime float64) {
	// Story 10.1: 更新暂停菜单模块
	if s.pauseMenuModule != nil {
		s.pauseMenuModule.Update(deltaTime)
	}

	// Story 10.1: Check if game is paused
	if s.gameState.IsPaused {
		// 暂停时只更新 UI 系统（按钮交互、暂停菜单）
		if s.buttonSystem != nil {
			s.buttonSystem.Update(deltaTime)
		}
		return // 跳过所有游戏逻辑系统
	}
	// Story 8.2 QA改进：铺草皮动画系统更新（必须在开场动画之前）
	if s.soddingSystem != nil {
		s.soddingSystem.Update(deltaTime)
	}

	// Story 8.3: Check if opening animation is playing
	if s.openingSystem != nil && !s.openingSystem.IsCompleted() {
		// 开场动画期间，只更新镜头系统、开场动画系统和 Reanim 系统（僵尸动画需要）
		s.cameraSystem.Update(deltaTime)
		s.openingSystem.Update(deltaTime)
		s.reanimSystem.Update(deltaTime) // 更新僵尸 idle 动画

		// 同步镜头位置到本地 cameraX（用于渲染）
		s.cameraX = s.gameState.CameraX
		return // 暂停其他游戏系统
	}

	// 开场动画刚完成，触发铺草皮动画（如果配置了且还未启动）
	// 修正：检查 ShowSoddingAnim 和 SodRollAnimation 配置
	if s.openingSystem != nil && s.openingSystem.IsCompleted() && !s.soddingAnimStarted && s.soddingSystem != nil {
		// 检查是否应该播放铺草皮动画
		shouldPlayAnim := s.gameState.CurrentLevel.ShowSoddingAnim || s.gameState.CurrentLevel.SodRollAnimation

		if shouldPlayAnim {
			log.Printf("[GameScene] 开场动画完成，启动铺草皮动画")

			// 启动动画，传递启用的行列表、草皮位置和图片高度
			enabledLanes := s.gameState.CurrentLevel.EnabledLanes
			// Story 8.6 QA修正: 获取需要播放动画的行列表
			animLanes := s.gameState.CurrentLevel.SoddingAnimLanes
			// Story 11.4: 读取粒子特效配置
			enableParticles := s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.SodRollParticles
			s.soddingSystem.StartAnimation(func() {
				// 动画完成回调：通知教学系统可以开始了
				log.Printf("[GameScene] 铺草皮动画完成")

				// Story 11.5 修复：动画完成后处理草皮叠加层
				// 原理：动画期间使用叠加层裁剪显示，完成后将草皮合并到底层背景
				if s.soddedBackground != nil {
					// Level 1-4: 有完整的已铺草皮背景，直接替换
					log.Printf("[GameScene] 替换底层背景: IMAGE_BACKGROUND1UNSODDED → IMAGE_BACKGROUND1")
					s.background = s.soddedBackground
					s.soddedBackground = nil
					s.preSoddedImage = nil
				} else if s.preSoddedImage != nil || s.sodRowImage != nil {
					// Level 1-1, 1-2: 需要将草皮叠加层合并到底层背景
					log.Printf("[GameScene] 合并草皮叠加层到底层背景")
					mergedBg := s.createMergedBackground()
					if mergedBg != nil {
						// 原子操作：先替换背景，再清空叠加层，确保渲染不中断
						s.background = mergedBg
						s.preSoddedImage = nil
						s.sodRowImage = nil
					} else {
						// 合并失败，保持叠加层不清空，避免草皮消失
						log.Printf("[GameScene] 警告：合并背景失败，保持叠加层")
					}
				}

				if s.tutorialSystem != nil {
					s.tutorialSystem.OnSoddingComplete()
				}
				// Story 10.2: 铺草皮完成后创建除草车（原版行为）
				s.initLawnmowers()
			}, enabledLanes, animLanes, s.sodOverlayX, float64(s.sodHeight), enableParticles)

			s.soddingAnimStarted = true
			// 标记开场动画系统为 nil，避免重复检查
			s.openingSystem = nil
			return // 等待铺草皮动画完成
		} else {
			// 不播放铺草皮动画，直接完成
			log.Printf("[GameScene] 开场动画完成，关卡配置禁用铺草皮动画，跳过")
			s.soddingAnimStarted = true
			s.openingSystem = nil
			// 立即初始化除草车（无需等待动画）
			s.initLawnmowers()
			// 通知教学系统
			if s.tutorialSystem != nil {
				s.tutorialSystem.OnSoddingComplete()
			}
			// 继续游戏流程，不return
		}
	}

	// Story 8.3: 如果铺草皮动画正在播放，暂停其他游戏系统
	if s.soddingSystem != nil && s.soddingSystem.IsPlaying() {
		// 铺草皮动画期间，只更新铺草皮系统、镜头系统和 Reanim 系统（草皮卷动画需要）
		s.cameraSystem.Update(deltaTime)
		s.reanimSystem.Update(deltaTime)   // 更新草皮卷动画帧
		s.particleSystem.Update(deltaTime) // Story 11.4: 更新粒子系统（土粒飞溅特效）
		s.cameraX = s.gameState.CameraX
		return // 暂停其他游戏系统（包括僵尸激活）
	}

	// 如果没有开场动画，使用延迟启动铺草皮动画（原逻辑）
	// 修正：检查 ShowSoddingAnim 和 SodRollAnimation 配置
	if s.openingSystem == nil && s.soddingSystem != nil && !s.soddingAnimStarted {
		// 检查是否应该播放铺草皮动画
		shouldPlayAnim := s.gameState.CurrentLevel.ShowSoddingAnim || s.gameState.CurrentLevel.SodRollAnimation

		if shouldPlayAnim && s.soddingAnimDelay >= 0 {
			s.soddingAnimTimer += deltaTime
			if s.soddingAnimTimer >= s.soddingAnimDelay {
				log.Printf("[GameScene] 启动铺草皮动画（延迟 %.1f 秒后）", s.soddingAnimDelay)

				// 启动动画，传递启用的行列表、草皮位置和图片高度
				enabledLanes := s.gameState.CurrentLevel.EnabledLanes
				// Story 8.6 QA修正: 获取需要播放动画的行列表
				animLanes := s.gameState.CurrentLevel.SoddingAnimLanes
				// Story 11.4: 读取粒子特效配置
				enableParticles := s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.SodRollParticles
				s.soddingSystem.StartAnimation(func() {
					// 动画完成回调：通知教学系统可以开始了
					log.Printf("[GameScene] 铺草皮动画完成")

					// Story 11.5 修复：动画完成后处理草皮叠加层
					// 原理：动画期间使用叠加层裁剪显示，完成后将草皮合并到底层背景
					if s.soddedBackground != nil {
						// Level 1-4: 有完整的已铺草皮背景，直接替换
						log.Printf("[GameScene] 替换底层背景: IMAGE_BACKGROUND1UNSODDED → IMAGE_BACKGROUND1")
						s.background = s.soddedBackground
						s.soddedBackground = nil
						s.preSoddedImage = nil
					} else if s.preSoddedImage != nil || s.sodRowImage != nil {
						// Level 1-1, 1-2: 需要将草皮叠加层合并到底层背景
						log.Printf("[GameScene] 合并草皮叠加层到底层背景")
						mergedBg := s.createMergedBackground()
						if mergedBg != nil {
							// 原子操作：先替换背景，再清空叠加层，确保渲染不中断
							s.background = mergedBg
							s.preSoddedImage = nil
							s.sodRowImage = nil
						} else {
							// 合并失败，保持叠加层不清空，避免草皮消失
							log.Printf("[GameScene] 警告：合并背景失败，保持叠加层")
						}
					}

					if s.tutorialSystem != nil {
						s.tutorialSystem.OnSoddingComplete()
					}
					// Story 10.2: 铺草皮完成后创建除草车（原版行为）
					s.initLawnmowers()
				}, enabledLanes, animLanes, s.sodOverlayX, float64(s.sodHeight), enableParticles)

				s.soddingAnimStarted = true
			}
		} else if !shouldPlayAnim {
			// 不播放铺草皮动画，但如果有 preSoddedLanes，需要显示预渲染的草皮
			preSoddedLanes := s.gameState.CurrentLevel.PreSoddedLanes
			if len(preSoddedLanes) > 0 {
				// 有预铺草皮配置，需要显示草皮但不播放动画
				// 将预渲染的背景副本（preSoddedImage）设置为永久可见
				log.Printf("[GameScene] 关卡配置禁用动画，但有预铺草皮 %v，显示预渲染背景", preSoddedLanes)
				// 注意：preSoddedImage 已在初始化时渲染好，这里不需要额外操作
				// 只需确保在 Draw() 中能正确显示
			} else {
				log.Printf("[GameScene] 关卡配置禁用铺草皮动画，无预铺草皮，跳过")
			}
			s.soddingAnimStarted = true
			// 立即初始化除草车（无需等待动画）
			s.initLawnmowers()
			// 通知教学系统
			if s.tutorialSystem != nil {
				s.tutorialSystem.OnSoddingComplete()
			}
		}
	}

	// Handle intro animation
	if s.isIntroAnimPlaying {
		s.updateIntroAnimation(deltaTime)
		// 同步摄像机位置到全局状态（即使在动画期间也保持同步）
		s.gameState.CameraX = s.cameraX
		return // Don't update game systems during intro animation
	}

	// 同步摄像机位置到全局状态（供所有系统使用）
	s.gameState.CameraX = s.cameraX

	// Story 5.5: Check if game is over (win or lose)
	// If game is over, stop updating game systems but allow reward animation to play
	if s.gameState.IsGameOver {
		// 游戏结束时仍然更新奖励系统和必要的动画系统
		// 这样玩家可以看到完整的奖励动画流程
		s.rewardSystem.Update(deltaTime)   // 奖励动画系统（卡片包动画）
		s.reanimSystem.Update(deltaTime)   // Reanim 系统（植物卡片动画）
		s.particleSystem.Update(deltaTime) // 粒子系统（光晕效果）
		return                             // 停止其他游戏系统（僵尸移动、植物攻击等）
	}

	// Update all ECS systems in order (order matters for correct game logic)
	s.levelSystem.Update(deltaTime)                // 0. Update level system (Story 5.5: wave spawning, victory/defeat)
	s.rewardSystem.Update(deltaTime)               // 0.1. Update reward animation system (Story 8.3: 卡片包动画)
	s.finalWaveWarningSystem.Update(deltaTime)     // 0.2. Update final wave warning (Story 11.3: 自动清理提示动画)
	s.zombieLaneTransitionSystem.Update(deltaTime) // 0.5. Update zombie lane transitions (move to target lane before attacking)

	// Story 3.1 架构优化：使用模块化方式更新植物选择栏
	if s.plantSelectionModule != nil {
		s.plantSelectionModule.Update(deltaTime) // 1. Update plant card states (before input)
	}

	s.inputSystem.Update(deltaTime, s.cameraX) // 2. Process player input (highest priority, 传递摄像机位置)

	// 3. Generate new suns
	// 教学关卡：在第一次收集阳光后启用自动生成（由 TutorialSystem 控制）
	// 非教学关卡：始终启用自动生成
	s.sunSpawnSystem.Update(deltaTime)

	s.sunMovementSystem.Update(deltaTime)   // 4. Move suns (includes collection animation)
	s.sunCollectionSystem.Update(deltaTime) // 5. Check if collection is complete
	s.behaviorSystem.Update(deltaTime)      // 6. Update plant behaviors (Story 3.4)
	// Story 10.2: Update lawnmower system (除草车系统)
	if s.lawnmowerSystem != nil {
		s.lawnmowerSystem.Update(deltaTime) // 6.5. Check lawnmower triggers and move lawnmowers
	}
	s.physicsSystem.Update(deltaTime) // 7. Check collisions (Story 4.3)
	// Story 6.3: Reanim 动画系统（替代旧的 AnimationSystem）
	s.reanimSystem.Update(deltaTime)   // 8. Update Reanim animation frames
	s.particleSystem.Update(deltaTime) // 9. Update particle effects (Story 7.2)
	// 方案A+：闪烁效果系统
	s.flashEffectSystem.Update(deltaTime) // 9.3. Update flash effects (hit feedback)
	// Story 8.2: Tutorial system (only if active)
	if s.tutorialSystem != nil {
		s.tutorialSystem.Update(deltaTime) // 9.5. Update tutorial text display
	}
	// Story 3.2: 植物预览系统 - 更新预览位置（双图像支持）
	s.plantPreviewSystem.Update(deltaTime) // 10. Update plant preview position (dual-image support)
	s.lawnGridSystem.Update(deltaTime)     // 10.5. Update lawn flash animation (Story 8.2)
	// ECS 按钮系统更新（交互检测）
	if s.buttonSystem != nil {
		s.buttonSystem.Update(deltaTime) // 10.7. Update button interactions (hover, click)
	}
	s.lifetimeSystem.Update(deltaTime)     // 11. Check for expired entities
	s.entityManager.RemoveMarkedEntities() // 12. Clean up deleted entities (always last)
}

// updateIntroAnimation updates the intro camera animation that showcases the entire lawn.
// The animation has two phases:
//   - Phase 1 (0.0-0.5): Camera scrolls from left edge (0) to right edge (maxCameraX)
//   - Phase 2 (0.5-1.0): Camera scrolls back from right edge to gameplay position (GameCameraX)
//
// Both phases use an ease-out quadratic easing function for smooth motion.
func (s *GameScene) updateIntroAnimation(deltaTime float64) {
	s.introAnimTimer += deltaTime
	progress := s.introAnimTimer / config.IntroAnimDuration

	if progress >= 1.0 {
		// Animation complete, camera settled at gameplay position
		s.cameraX = config.GameCameraX
		s.isIntroAnimPlaying = false
		return
	}

	if progress < 0.5 {
		// Phase 1: Scroll from left (0) to right (maxCameraX)
		phaseProgress := progress / 0.5
		easedProgress := s.easeOutQuad(phaseProgress)
		s.cameraX = easedProgress * s.maxCameraX
	} else {
		// Phase 2: Scroll from right (maxCameraX) back to center (config.GameCameraX)
		phaseProgress := (progress - 0.5) / 0.5
		easedProgress := s.easeOutQuad(phaseProgress)
		s.cameraX = s.maxCameraX + easedProgress*(config.GameCameraX-s.maxCameraX)
	}
}

// easeOutQuad applies an ease-out quadratic easing function to the input value.
// Formula: 1 - (1-t)^2
// This creates a smooth deceleration effect.
//
// Parameters:
//   - t: Input value in range [0, 1]
//
// Returns:
//   - Eased value in range [0, 1]
func (s *GameScene) easeOutQuad(t float64) float64 {
	return 1 - (1-t)*(1-t)
}

// Draw renders the game scene to the screen.
// It draws the lawn background, all game entities, and UI elements in the correct order.
// Rendering order (back to front, 符合原版PVZ):
// 1. Background (lawn) - 草坪背景
// 2. UI base layer (seed bank, shovel) - UI基础元素
// 3. Plant cards - 卡片栏（在游戏实体下方）
// 4. Game entities (plants, zombies, projectiles) - 游戏世界实体
// 5. UI overlay (sun counter text) - UI文字（始终可见）
// 6. Plant preview - 植物拖拽预览
// 7. Suns (阳光) - 最顶层，确保可点击
func (s *GameScene) Draw(screen *ebiten.Image) {
	// Layer 1: Draw lawn background
	s.drawBackground(screen)

	// Layer 2: Draw UI base elements (seed bank, shovel, plant cards)
	// 按照原版PVZ设计，UI元素在游戏世界实体下方渲染
	s.drawSeedBank(screen)
	s.drawShovel(screen)

	// 使用 ECS 按钮系统渲染菜单按钮
	if s.buttonRenderSystem != nil {
		s.buttonRenderSystem.Draw(screen)
	}

	// Layer 3: Draw plant cards (Story 3.1 架构优化)
	// 在植物和僵尸下方渲染，符合原版PVZ设计
	if s.plantSelectionModule != nil {
		s.plantSelectionModule.Draw(screen)
	}

	// Layer 4: Draw game world entities (plants, zombies, projectiles) - 不包括阳光
	// 游戏实体在UI卡片上方，这样植物和僵尸可以被看清
	// 传递 cameraX 以正确转换世界坐标到屏幕坐标
	s.renderSystem.DrawGameWorld(screen, s.cameraX)

	// Layer 4.5: Draw lawn flash effect (Story 8.2 教学)
	// 草坪闪烁效果，用于教学提示玩家可以种植
	s.drawLawnFlash(screen)

	// Layer 5: Draw UI overlays (sun counter text)
	// 文字始终在最上层以确保可读性
	s.drawSunCounter(screen)

	// Layer 6: Draw particle effects (Story 7.3)
	// 只渲染游戏世界的粒子（爆炸、溅射等），过滤掉 UI 粒子
	// UI 粒子（如奖励动画）由各自的系统在更高层级渲染
	s.renderSystem.DrawGameWorldParticles(screen, s.cameraX)

	// Layer 7: Draw plant preview (Story 3.2)
	// 拖拽预览在所有内容上方
	s.plantPreviewRenderSystem.Draw(screen, s.cameraX)

	// Layer 8: Draw suns (阳光) - 最顶层
	// 阳光在最顶层以确保始终可点击
	s.renderSystem.DrawSuns(screen, s.cameraX)

	// Layer 8.5: Draw tutorial text (Story 8.2)
	// 教学文本在阳光之下、UI之上
	if s.tutorialSystem != nil && s.tutorialFont != nil {
		s.renderSystem.DrawTutorialText(screen, s.tutorialFont)
	}

	// Layer 8.6: Draw UI particles (教学箭头、奖励粒子等)
	// UI 粒子不受摄像机影响，渲染在教学文本之后
	s.renderSystem.DrawParticles(screen, 0)

	// Layer 9: Draw level progress bar (Story 5.5 - 正式版本)
	// 右下角图形化进度条
	s.drawProgressBar(screen)

	// Layer 10: Draw last wave warning (Story 5.5) - DISABLED for production
	// 最后一波提示（如果需要显示）（开发调试用，已禁用）
	// s.drawLastWaveWarning(screen) // 已禁用：改为使用 FinalWave.reanim 动画

	// Layer 10.5: Draw reward panel (Story 8.3 + 8.4)
	// Story 8.4重构：RewardAnimationSystem完全封装奖励面板渲染
	// 内部自动管理面板和植物卡片的渲染，调用者只需调用Draw方法
	s.rewardSystem.Draw(screen)

	// Story 8.3: 移除 "You Win" 覆盖层逻辑
	// 奖励流程完成后通过"下一关"按钮进入下一关，不再显示 You Win
	// Layer 11: Draw game result overlay (Story 5.5) - DISABLED for Story 8.3
	// s.drawGameResultOverlay(screen) // 已禁用：改为通过奖励面板的"下一关"按钮进入下一关

	// DEBUG: Draw particle test instructions (Story 7.4 debugging) - DISABLED
	// s.drawParticleTestInstructions(screen)

	// DEBUG: Draw grid boundaries and SodRoll debug lines (Story 3.3 debugging)
	s.drawGridDebug(screen)

	// Story 10.1: Draw pause menu (最顶层 - 在所有其他元素之上)
	if s.pauseMenuModule != nil {
		s.pauseMenuModule.Draw(screen)
	}
}

// drawBackground renders the lawn background.
// The background image is larger than the window, and we display only a portion of it.
// The cameraX value determines which horizontal section of the background is visible.
// During the intro animation, the camera scrolls left → right → center to showcase the entire scene.
func (s *GameScene) drawBackground(screen *ebiten.Image) {
	if s.background != nil {
		// Get background image dimensions
		bounds := s.background.Bounds()
		bgWidth := bounds.Dx()
		bgHeight := bounds.Dy()

		// Calculate the viewport rectangle based on camera position
		// We want to show a WindowWidth x WindowHeight portion of the background
		viewportX := int(s.cameraX)
		viewportY := 0

		// Ensure we don't go out of bounds
		if viewportX+WindowWidth > bgWidth {
			viewportX = bgWidth - WindowWidth
		}
		if viewportX < 0 {
			viewportX = 0
		}

		// If background is smaller than window height, center it vertically
		if bgHeight > WindowHeight {
			// Center the viewport vertically if background is taller
			viewportY = (bgHeight - WindowHeight) / 2
		}

		// Create a sub-image representing the visible portion
		viewportRect := image.Rect(
			viewportX,
			viewportY,
			viewportX+WindowWidth,
			viewportY+WindowHeight,
		)

		// Extract the visible portion of the background
		visibleBG := s.background.SubImage(viewportRect).(*ebiten.Image)

		// Draw the visible portion at (0, 0)
		op := &ebiten.DrawImageOptions{}
		screen.DrawImage(visibleBG, op)

		// 统一草皮渲染：所有关卡使用双背景叠加模式
		//
		// 设计原理：
		// - 底层：未铺草皮背景 (IMAGE_BACKGROUND1UNSODDED) + 预渲染的preSoddedLanes草皮
		// - 叠加层：预渲染背景副本 (preSoddedImage) 或完整铺草皮背景 (soddedBackground)
		// - 根据草皮卷位置 (sodRollCenterX)，渐进显示叠加层（从世界坐标0裁剪到草皮卷中心）
		//
		// 优势：
		// - 统一逻辑：1-1/1-2/1-4 使用相同代码路径
		// - 坐标清晰：所有计算使用世界坐标，最后转屏幕坐标
		// - 易于维护：无需区分单行/连续行/全行模式
		if s.soddingSystem != nil && s.gameState.CurrentLevel != nil {
			// 选择叠加背景（优先使用完整背景，否则使用预渲染背景）
			overlayBg := s.soddedBackground
			usingPreSoddedImage := false
			if overlayBg == nil {
				overlayBg = s.preSoddedImage
				usingPreSoddedImage = true
			}

			if overlayBg != nil {
				// 获取草皮卷当前位置（世界坐标X）
				sodRollCenterX := s.soddingSystem.GetSodRollCenterX()
				animProgress := s.soddingSystem.GetProgress()

				// 计算可见宽度（从世界坐标 0 到草皮卷中心）
				visibleWorldWidth := int(sodRollCenterX)

				// 特殊处理：如果有预铺草皮（preSoddedLanes）且动画未启动，显示整个预渲染背景
				// 这样可以在动画开始前显示预铺的草皮（如 1-2 关的第2/4行）
				// Level 1-1: preSoddedLanes=[] → 动画前不显示叠加层
				// Level 1-2: preSoddedLanes=[2,4] → 动画前显示预铺草皮
				hasPreSoddedLanes := len(s.gameState.CurrentLevel.PreSoddedLanes) > 0
				if usingPreSoddedImage && !s.soddingSystem.HasStarted() && hasPreSoddedLanes {
					bgBounds := overlayBg.Bounds()
					visibleWorldWidth = bgBounds.Dx()
				}

				// 优化：动画接近完成时（≥99%），直接显示完整叠加层，避免切换时闪烁
				if animProgress >= 0.99 {
					bgBounds := overlayBg.Bounds()
					visibleWorldWidth = bgBounds.Dx()
				}

				// 只有草皮卷到达可见位置后才渲染叠加层
				if visibleWorldWidth > 0 {
					// 获取叠加背景尺寸
					bgBounds := overlayBg.Bounds()
					bgWidth := bgBounds.Dx()
					bgHeight := bgBounds.Dy()

					// 限制可见宽度不超过背景宽度
					if visibleWorldWidth > bgWidth {
						visibleWorldWidth = bgWidth
					}

					// 计算视口裁剪区域（世界坐标）
					// viewportX 是摄像机在世界坐标中的位置
					overlayViewportX := viewportX
					overlayViewportY := viewportY

					// 水平方向：裁剪从 viewportX 到 min(visibleWorldWidth, viewportX + WindowWidth)
					overlayViewportEndX := visibleWorldWidth
					if overlayViewportX < 0 {
						overlayViewportX = 0
					}
					// 不能超过可见宽度
					if overlayViewportEndX >= overlayViewportX {
						// 垂直方向：显示整个高度
						overlayViewportEndY := overlayViewportY + WindowHeight
						if overlayViewportEndY > bgHeight {
							overlayViewportEndY = bgHeight
						}
						if overlayViewportY < 0 {
							overlayViewportY = 0
						}

						// 裁剪叠加背景的可见部分（世界坐标裁剪）
						overlayRect := image.Rect(
							overlayViewportX,
							overlayViewportY,
							overlayViewportEndX,
							overlayViewportEndY,
						)

						visibleOverlay := overlayBg.SubImage(overlayRect).(*ebiten.Image)

						// 世界坐标 → 屏幕坐标转换
						screenX := float64(overlayViewportX) - float64(viewportX)
						screenY := 0.0 // Y轴无摄像机移动

						// 绘制叠加背景到屏幕
						overlayOp := &ebiten.DrawImageOptions{}
						overlayOp.GeoM.Translate(screenX, screenY)
						screen.DrawImage(visibleOverlay, overlayOp)

						// DEBUG: 打印叠加背景信息（每10帧输出一次）
						if frameIndex := int(s.soddingSystem.GetProgress() * 48); frameIndex%10 == 0 || frameIndex == 0 || frameIndex >= 47 {
							log.Printf("[草皮叠加] 帧:%d, 可见世界宽度: %d px, sodRollCenterX: %.1f, 差值: %.1f px",
								frameIndex, visibleWorldWidth, sodRollCenterX, sodRollCenterX-float64(visibleWorldWidth))
						}
					}
				}
			}
		}
	} else {
		// Fallback: Draw a green background to simulate grass
		screen.Fill(color.RGBA{R: 34, G: 139, B: 34, A: 255}) // Forest green
	}
}

// createMergedBackground 创建合并了草皮叠加层的背景图片
// 在铺草皮动画完成后调用，将底层背景和草皮叠加层合并成一个新的完整背景
// 返回合并后的背景图片，如果失败返回 nil
func (s *GameScene) createMergedBackground() *ebiten.Image {
	if s.background == nil {
		log.Printf("[createMergedBackground] 错误：底层背景为空")
		return nil
	}

	// 选择叠加图层（preSoddedImage 或 sodRowImage）
	var overlayImg *ebiten.Image
	if s.preSoddedImage != nil {
		overlayImg = s.preSoddedImage
		log.Printf("[createMergedBackground] 使用 preSoddedImage 作为叠加层")
	} else if s.sodRowImage != nil {
		overlayImg = s.sodRowImage
		log.Printf("[createMergedBackground] 使用 sodRowImage 作为叠加层")
	} else {
		log.Printf("[createMergedBackground] 错误：没有可用的草皮叠加层")
		return nil
	}

	// 获取背景尺寸
	bgBounds := s.background.Bounds()
	bgWidth := bgBounds.Dx()
	bgHeight := bgBounds.Dy()

	log.Printf("[createMergedBackground] 创建合并背景: 尺寸 %dx%d", bgWidth, bgHeight)

	// 创建新的背景图片
	mergedBg := ebiten.NewImage(bgWidth, bgHeight)

	// 1. 绘制底层背景
	op := &ebiten.DrawImageOptions{}
	mergedBg.DrawImage(s.background, op)

	// 2. 绘制草皮叠加层
	// 使用 preSoddedImage 时，它已经包含了正确位置的草皮
	// 使用 sodRowImage 时，需要根据配置的位置绘制
	if s.preSoddedImage != nil {
		// preSoddedImage 已经是完整的背景副本，直接使用
		mergedBg.DrawImage(overlayImg, op)
	} else if s.sodRowImage != nil {
		// sodRowImage 需要放置在正确的位置
		overlayOp := &ebiten.DrawImageOptions{}
		overlayOp.GeoM.Translate(float64(s.sodOverlayX), float64(s.sodOverlayY))
		mergedBg.DrawImage(overlayImg, overlayOp)
	}

	log.Printf("[createMergedBackground] 成功创建合并背景")
	return mergedBg
}

// drawSeedBank renders the plant selection bar at the top left of the screen.
// If the seed bank image is not loaded, it draws a simple rectangle as fallback.
func (s *GameScene) drawSeedBank(screen *ebiten.Image) {
	if s.seedBank != nil {
		// Draw the seed bank image at the top left corner
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(config.SeedBankX, config.SeedBankY)
		screen.DrawImage(s.seedBank, op)
	} else {
		// Fallback: Draw a dark brown rectangle
		ebitenutil.DrawRect(screen,
			config.SeedBankX, config.SeedBankY,
			config.SeedBankWidth, config.SeedBankHeight,
			color.RGBA{R: 101, G: 67, B: 33, A: 255}) // Dark brown
	}
}

// drawSunCounter renders the sun counter value on the seed bank.
// Note: The sun counter background and gold frame are already part of the bar5.png image,
// so we don't need to draw them separately. This method displays the sun count number.
// The text is horizontally centered to accommodate dynamic value lengths (e.g., 50, 150, 9990).
func (s *GameScene) drawSunCounter(screen *ebiten.Image) {
	// Get current sun value from game state
	sunValue := s.gameState.GetSun()
	sunText := fmt.Sprintf("%d", sunValue)

	if s.sunCounterFont != nil {
		// Measure text width for centering
		textWidth, _ := text.Measure(sunText, s.sunCounterFont, 0)

		// Calculate centered position
		// Base position is relative to SeedBank
		centerX := float64(config.SeedBankX + config.SunCounterOffsetX)
		centerY := float64(config.SeedBankY + config.SunCounterOffsetY)

		// Adjust X to center the text horizontally
		sunDisplayX := centerX - textWidth/2
		sunDisplayY := centerY

		// Use custom font with color
		op := &text.DrawOptions{}
		op.GeoM.Translate(sunDisplayX, sunDisplayY)

		// Set text color to black for better visibility on the beige background
		op.ColorScale.ScaleWithColor(color.RGBA{R: 0, G: 0, B: 0, A: 255})

		text.Draw(screen, sunText, s.sunCounterFont, op)
	} else {
		// Fallback: Use debug text if font failed to load
		// Note: Debug text doesn't support centering easily
		sunDisplayX := config.SeedBankX + config.SunCounterOffsetX
		sunDisplayY := config.SeedBankY + config.SunCounterOffsetY
		ebitenutil.DebugPrintAt(screen, sunText, sunDisplayX, sunDisplayY)
	}
}

// drawShovel renders the shovel slot and icon at the right side of the seed bank.
// The shovel will be used in future stories for removing plants.
// Story 8.5: 1-1关（教学关卡）不显示铲子
// Story 8.6: 检查铲子是否已解锁（1-4关完成后才解锁）
func (s *GameScene) drawShovel(screen *ebiten.Image) {
	// 教学关卡不显示铲子（玩家还不需要学习移除植物）
	if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.OpeningType == "tutorial" {
		return
	}

	// Story 8.6: 检查铲子是否已解锁
	// 铲子在完成 1-4 关卡后解锁
	if !s.gameState.IsToolUnlocked("shovel") {
		return
	}

	// Draw shovel slot background first
	if s.shovelSlot != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(config.ShovelX, config.ShovelY)
		screen.DrawImage(s.shovelSlot, op)
	}

	// Draw shovel icon on top of the slot
	if s.shovel != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(config.ShovelX, config.ShovelY)
		screen.DrawImage(s.shovel, op)
	} else if s.shovelSlot == nil {
		// Fallback: Draw a gray rectangle if both images are missing
		ebitenutil.DrawRect(screen,
			config.ShovelX, config.ShovelY,
			config.ShovelWidth, config.ShovelHeight,
			color.RGBA{R: 128, G: 128, B: 128, A: 255}) // Gray
	}
}

// drawParticleTestInstructions 绘制粒子效果测试说明（调试用）
// Story 7.4: 方便测试粒子效果，无需通过攻击触发
func (s *GameScene) drawParticleTestInstructions(screen *ebiten.Image) {
	// 只在非游戏结束状态下显示
	if s.gameState.IsGameOver {
		return
	}

	// 测试说明（屏幕左下角）
	instructions := []string{
		"[粒子测试] P=豌豆溅射 | B=爆炸 | A=奖励光效 | Z=僵尸头 | L=种植土粒",
	}

	// 绘制半透明背景
	y := float64(WindowHeight - 25)
	bgPadding := 5.0
	ebitenutil.DrawRect(screen,
		10-bgPadding,
		y-bgPadding,
		300,
		20,
		color.RGBA{R: 0, G: 0, B: 0, A: 120})

	// 绘制文本
	for i, line := range instructions {
		yPos := int(y) + i*15
		ebitenutil.DebugPrintAt(screen, line, 10, yPos)
	}
}

// drawGridDebug 绘制草坪网格边界（调试用）
// 在开发阶段帮助可视化可种植区域
func (s *GameScene) drawGridDebug(screen *ebiten.Image) {
	// Story 8.2 QA: 临时启用调试绘制，验证草坪布局和SodRoll动画
	// 在种植模式或SodRoll动画期间显示
	showDebug := s.gameState.IsPlantingMode
	// if !showDebug && s.soddingSystem != nil {
	// 	// 如果SodRoll动画启动过（包括正在播放和已完成），也显示调试信息
	// 	showDebug = s.soddingSystem.HasStarted()
	// }

	if !showDebug {
		return
	}

	// 使用统一的网格参数（从 config.layout_config.go）
	// 注意：这里使用的是世界坐标，需要转换为屏幕坐标
	gridWorldStartX := config.GridWorldStartX
	gridWorldStartY := config.GridWorldStartY
	gridColumns := config.GridColumns
	gridRows := config.GridRows
	cellWidth := config.CellWidth
	cellHeight := config.CellHeight

	// 将网格世界坐标转换为屏幕坐标
	gridScreenStartX := gridWorldStartX - s.cameraX
	gridScreenStartY := gridWorldStartY

	// 绘制网格线
	gridColor := color.RGBA{R: 255, G: 255, B: 0, A: 128} // 半透明黄色

	// 绘制垂直线
	for col := 0; col <= gridColumns; col++ {
		x := gridScreenStartX + float64(col)*cellWidth
		ebitenutil.DrawLine(screen, x, gridScreenStartY, x, gridScreenStartY+float64(gridRows)*cellHeight, gridColor)
	}

	// 绘制水平线
	for row := 0; row <= gridRows; row++ {
		y := gridScreenStartY + float64(row)*cellHeight
		ebitenutil.DrawLine(screen, gridScreenStartX, y, gridScreenStartX+float64(gridColumns)*cellWidth, y, gridColor)
	}

	// Story 8.2 QA: 绘制草皮叠加图边界（红色矩形）
	if s.sodRowImage != nil {
		// 性能优化：使用缓存的尺寸和位置
		sodWidth := float64(s.sodWidth)
		sodHeight := float64(s.sodHeight)
		sodOverlayX := s.sodOverlayX
		sodOverlayY := s.sodOverlayY

		// 转换为屏幕坐标
		sodScreenX := sodOverlayX - s.cameraX
		sodScreenY := sodOverlayY

		// Story 8.2 QA：调试可视化（临时启用以调试粒子）

		// 绘制草皮边界（红色矩形框，不填充）
		sodColor := color.RGBA{R: 255, G: 0, B: 0, A: 128} // 半透明红色
		thickness := 2.0
		// 顶边
		ebitenutil.DrawRect(screen, sodScreenX, sodScreenY, sodWidth, thickness, sodColor)
		// 底边
		ebitenutil.DrawRect(screen, sodScreenX, sodScreenY+sodHeight-thickness, sodWidth, thickness, sodColor)
		// 左边
		ebitenutil.DrawRect(screen, sodScreenX, sodScreenY, thickness, sodHeight, sodColor)
		// 右边
		ebitenutil.DrawRect(screen, sodScreenX+sodWidth-thickness, sodScreenY, thickness, sodHeight, sodColor)

		// 绘制草皮卷左、中、右边缘标记
		// 只要动画启动过就绘制（包括已完成的状态）
		if s.soddingSystem != nil && s.soddingSystem.HasStarted() {
			leftEdge, centerX, rightEdge := s.soddingSystem.GetSodRollEdges()

			// 转换为屏幕坐标
			leftScreenX := leftEdge - s.cameraX
			centerScreenX := centerX - s.cameraX
			rightScreenX := rightEdge - s.cameraX

			// DEBUG: 每10帧打印一次
			frameIndex := int(s.soddingSystem.GetProgress() * 48)
			if frameIndex%10 == 0 || frameIndex == 0 {
				log.Printf("[调试线] 帧:%d, 世界坐标: 左=%.1f 中=%.1f 右=%.1f, 屏幕坐标: 左=%.1f 中=%.1f 右=%.1f",
					frameIndex, leftEdge, centerX, rightEdge, leftScreenX, centerScreenX, rightScreenX)
			}

			// 绘制三条竖线（加粗，更明显）
			lineHeight := sodHeight + 40.0
			lineStartY := sodScreenY - 20.0
			lineWidth := 4.0 // 加粗线条

			// 左边缘 - 红色
			leftColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
			ebitenutil.DrawRect(screen, leftScreenX-lineWidth/2, lineStartY, lineWidth, lineHeight, leftColor)

			// 中心 - 绿色
			centerColor := color.RGBA{R: 0, G: 255, B: 0, A: 255}
			ebitenutil.DrawRect(screen, centerScreenX-lineWidth/2, lineStartY, lineWidth, lineHeight, centerColor)

			// 右边缘 - 蓝色
			rightColor := color.RGBA{R: 0, G: 0, B: 255, A: 255}
			ebitenutil.DrawRect(screen, rightScreenX-lineWidth/2, lineStartY, lineWidth, lineHeight, rightColor)
		}

		// 详细调试信息
		debugInfo := fmt.Sprintf("Sod: world(%.0f,%.0f) screen(%.0f,%.0f) size(%.0fx%.0f) cam(%.0f)",
			sodOverlayX, sodOverlayY, sodScreenX, sodScreenY, sodWidth, sodHeight, s.cameraX)
		ebitenutil.DebugPrintAt(screen, debugInfo, 10, 30)

		// 草皮卷边缘位置调试信息
		if s.soddingSystem != nil && s.soddingSystem.IsPlaying() {
			leftEdge, centerX, rightEdge := s.soddingSystem.GetSodRollEdges()
			progress := s.soddingSystem.GetProgress()
			edgeInfo := fmt.Sprintf("草皮卷: 左(%.0f) 中(%.0f) 右(%.0f) 进度(%.1f%%)",
				leftEdge, centerX, rightEdge, progress*100)
			ebitenutil.DebugPrintAt(screen, edgeInfo, 10, 50)
		}
	}

	// 绘制坐标信息文本
	debugText := fmt.Sprintf("Grid: (%.0f, %.0f), Cell: %.0fx%.0f", gridWorldStartX, gridWorldStartY, cellWidth, cellHeight)
	ebitenutil.DebugPrintAt(screen, debugText, 10, 10)
}

// drawLevelProgress renders the level progress indicator (Story 5.5)
// Displays current wave number and total waves in the bottom-right corner
// Format: "Wave X/Y"
func (s *GameScene) drawLevelProgress(screen *ebiten.Image) {
	// 只在关卡加载后显示进度
	if s.gameState.CurrentLevel == nil {
		return
	}

	// 获取当前波次和总波次
	currentWave, totalWaves := s.gameState.GetLevelProgress()

	// 如果还没有生成任何波次，显示"Wave 0/Y"
	// 当第一波生成后，显示"Wave 1/Y"，以此类推
	progressText := fmt.Sprintf("Wave %d/%d", currentWave, totalWaves)

	// 计算文本位置（屏幕右下角）
	// 使用阳光计数器字体（复用已有字体资源）
	if s.sunCounterFont == nil {
		return // 字体未加载时不渲染
	}

	// 测量文本宽度以便右对齐
	textWidth := text.Advance(progressText, s.sunCounterFont)

	// 位置：右下角，留10像素边距
	x := float64(WindowWidth) - textWidth - 10
	y := float64(WindowHeight) - 30.0 // 距离底部30像素

	// 绘制半透明黑色背景（提高可读性）
	bgPadding := 5.0
	ebitenutil.DrawRect(screen,
		x-bgPadding,
		y-float64(s.sunCounterFont.Metrics().HAscent)-bgPadding,
		textWidth+bgPadding*2,
		float64(s.sunCounterFont.Metrics().HAscent+s.sunCounterFont.Metrics().HDescent)+bgPadding*2,
		color.RGBA{R: 0, G: 0, B: 0, A: 150})

	// 绘制文本（白色）
	textOp := &text.DrawOptions{}
	textOp.GeoM.Translate(x, y)
	textOp.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, progressText, s.sunCounterFont, textOp)
}

// drawLastWaveWarning renders the "A huge wave of zombies is approaching!" warning (Story 5.5)
// Only displays when the last wave is about to start (controlled by LevelSystem)
func (s *GameScene) drawLastWaveWarning(screen *ebiten.Image) {
	// 检查是否需要显示最后一波提示
	// 这里通过检查时间和波次状态来决定是否显示
	if s.gameState.CurrentLevel == nil {
		return
	}

	totalWaves := len(s.gameState.CurrentLevel.Waves)
	if totalWaves == 0 {
		return
	}

	// 获取当前波次索引和最后一波索引
	currentWaveIndex := s.gameState.CurrentWaveIndex
	lastWaveIndex := totalWaves - 1

	// 显示条件：进入最后一波等待期（倒数第二波消灭完毕）
	// 显示时长：直到最后一波生成前（约 minDelay 秒）
	if currentWaveIndex == lastWaveIndex && !s.gameState.IsWaveSpawned(lastWaveIndex) {

		// 绘制警告文本（屏幕中央上方）
		warningText := "A huge wave of zombies is approaching!"

		if s.sunCounterFont == nil {
			return
		}

		// 测量文本宽度以便居中
		textWidth := text.Advance(warningText, s.sunCounterFont)

		// 位置：屏幕中央上方
		x := (float64(WindowWidth) - textWidth) / 2.0
		y := 150.0

		// 绘制半透明红色背景
		bgPadding := 10.0
		ebitenutil.DrawRect(screen,
			x-bgPadding,
			y-float64(s.sunCounterFont.Metrics().HAscent)-bgPadding,
			textWidth+bgPadding*2,
			float64(s.sunCounterFont.Metrics().HAscent+s.sunCounterFont.Metrics().HDescent)+bgPadding*2,
			color.RGBA{R: 139, G: 0, B: 0, A: 200}) // 深红色背景

		// 绘制文本（黄色，更醒目）
		textOp := &text.DrawOptions{}
		textOp.GeoM.Translate(x, y)
		textOp.ColorScale.ScaleWithColor(color.RGBA{R: 255, G: 255, B: 0, A: 255}) // 黄色
		text.Draw(screen, warningText, s.sunCounterFont, textOp)
	}
}

// drawGameResultOverlay renders the victory or defeat overlay (Story 5.5)
// Displays when the game ends (IsGameOver = true)
// Story 8.3: 奖励流程期间不显示 You Win，让玩家专注于奖励动画
func (s *GameScene) drawGameResultOverlay(screen *ebiten.Image) {
	// 只在游戏结束时显示
	if !s.gameState.IsGameOver {
		return
	}

	// Story 8.3: 如果奖励动画正在播放，不显示游戏结果覆盖层
	// 奖励流程完成后才显示 You Win 或直接进入下一关
	if s.rewardSystem != nil && s.rewardSystem.IsActive() {
		return // 奖励动画播放期间，隐藏 You Win
	}

	// 根据游戏结果选择显示内容
	var overlayColor color.Color
	var resultText string

	switch s.gameState.GameResult {
	case "win":
		// 胜利：半透明黑色背景，绿色文本
		overlayColor = color.RGBA{R: 0, G: 0, B: 0, A: 150}
		resultText = "YOU WIN!"
	case "lose":
		// 失败：半透明红色背景，白色文本
		overlayColor = color.RGBA{R: 100, G: 0, B: 0, A: 180}
		resultText = "THE ZOMBIES ATE YOUR BRAINS!"
	default:
		return // 游戏结果未知，不显示任何内容
	}

	// 绘制全屏半透明遮罩
	ebitenutil.DrawRect(screen, 0, 0, float64(WindowWidth), float64(WindowHeight), overlayColor)

	// 绘制结果文本（屏幕中央）
	if s.sunCounterFont == nil {
		return
	}

	// 测量文本宽度以便居中
	textWidth := text.Advance(resultText, s.sunCounterFont)

	// 位置：屏幕中央
	x := (float64(WindowWidth) - textWidth) / 2.0
	y := float64(WindowHeight) / 2.0

	// 根据游戏结果选择文本颜色
	var textColor color.Color
	if s.gameState.GameResult == "win" {
		textColor = color.RGBA{R: 0, G: 255, B: 0, A: 255} // 绿色
	} else {
		textColor = color.White
	}

	// 绘制文本
	textOp := &text.DrawOptions{}
	textOp.GeoM.Translate(x, y)
	textOp.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, resultText, s.sunCounterFont, textOp)

	// 绘制提示文字（"按ESC返回主菜单"等）
	hintText := "Press ESC to return to main menu"
	hintTextWidth := text.Advance(hintText, s.sunCounterFont)
	hintX := (float64(WindowWidth) - hintTextWidth) / 2.0
	hintY := y + 40.0

	hintOp := &text.DrawOptions{}
	hintOp.GeoM.Translate(hintX, hintY)
	hintOp.ColorScale.ScaleWithColor(color.RGBA{R: 200, G: 200, B: 200, A: 255}) // 浅灰色
	text.Draw(screen, hintText, s.sunCounterFont, hintOp)
}

// drawLawnFlash 绘制草坪闪烁效果（Story 8.2 教学）
// 在玩家选择植物卡片后，已铺设草皮的行会有明暗变化的闪烁效果（由明变暗）
// 使用黑色半透明遮罩实现草皮颜色变暗
// 只在关卡指定的启用行（enabledLanes）上显示闪烁效果
func (s *GameScene) drawLawnFlash(screen *ebiten.Image) {
	alpha := s.lawnGridSystem.GetFlashAlpha()
	if alpha <= 0 {
		return // 没有闪烁效果，直接返回
	}

	// 获取启用的行列表
	enabledLanes := s.lawnGridSystem.EnabledLanes
	if len(enabledLanes) == 0 {
		return // 没有启用的行
	}

	// 为每个启用的行单独绘制闪烁效果
	for _, lane := range enabledLanes {
		// 计算该行的世界坐标范围
		// lane 是 1-based (1-5)，需要转换为 0-based (0-4)
		rowIndex := lane - 1

		// 行的Y坐标范围
		rowStartY := config.GridWorldStartY + float64(rowIndex)*config.CellHeight
		rowEndY := rowStartY + config.CellHeight

		// 行的X坐标范围（整个草坪宽度）
		rowStartX := config.GridWorldStartX
		rowEndX := config.GridWorldStartX + float64(config.GridColumns)*config.CellWidth

		// 转换为屏幕坐标
		screenStartX := rowStartX - s.cameraX
		screenStartY := rowStartY
		width := rowEndX - rowStartX
		height := rowEndY - rowStartY

		// 创建黑色半透明遮罩（让草皮变暗）
		flashImage := ebiten.NewImage(int(width), int(height))
		flashImage.Fill(color.RGBA{0, 0, 0, uint8(alpha * 255)}) // 黑色，alpha 0.0-0.3

		// 绘制到屏幕
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(screenStartX, screenStartY)
		screen.DrawImage(flashImage, op)
	}
}

// drawProgressBar 渲染右下角的进度条（使用原版资源）
// 显示当前关卡进度和已消灭的僵尸波次
func (s *GameScene) drawProgressBar(screen *ebiten.Image) {
	// Story 11.2: 使用 ECS 进度条渲染系统
	if s.levelProgressBarRenderSystem != nil {
		s.levelProgressBarRenderSystem.Draw(screen)
	}
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
