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
	"github.com/decker502/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	// UI Layout Constants - positions and sizes for UI elements
	// Seed Bank (植物选择栏)
	SeedBankX      = 10
	SeedBankY      = 0
	SeedBankWidth  = 500
	SeedBankHeight = 87

	// Sun Counter (阳光计数器) - relative to SeedBank position
	SunCounterOffsetX  = 40 // 相对于 SeedBank 的 X 偏移量
	SunCounterOffsetY  = 64 // 相对于 SeedBank 的 Y 偏移量
	SunCounterWidth    = 130
	SunCounterHeight   = 60
	SunCounterFontSize = 18.0 // 阳光数值字体大小（像素）- 增大以提高可读性

	// Plant Cards (植物卡片) - relative to SeedBank position
	PlantCardStartOffsetX = 84   // 第一张卡片相对于 SeedBank 的 X 偏移量
	PlantCardOffsetY      = 8    // 卡片相对于 SeedBank 的 Y 偏移量
	PlantCardSpacing      = 60   // 卡片槽之间的间距（包含卡槽边框，每个卡槽约76px宽）
	// PlantCardScale 已移至 config.PlantCardScale（统一配置管理）
	PlantCardScale        = config.PlantCardScale // 卡片缩放因子（0.50）

	// Story 8.4: 卡片内部配置（图标缩放、偏移等）已移至 config.plant_card_config.go，不再在此定义

	// Shovel (铲子) - positioned to the right of seed bank
	ShovelX      = 620 // To the right of seed bank (bar5.png width=612 + small gap)
	ShovelY      = 8
	ShovelWidth  = 70
	ShovelHeight = 74

	// Animation Constants
	// The background image is wider than the window, we show only a portion
	IntroAnimDuration = 3.0 // Duration of intro animation in seconds
	CameraScrollSpeed = 100 // Pixels per second for intro animation
	// GameCameraX 已移至 config.GameCameraX (统一配置管理)
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

	// Story 8.2 QA改进：草皮叠加层（随动画进度渐进显示）
	sodRowImage        *ebiten.Image // 草皮叠加图片（sod1row.jpg）
	soddingAnimDelay   float64       // 铺草皮动画延迟时间（秒）
	soddingAnimStarted bool          // 铺草皮动画是否已启动
	soddingAnimTimer   float64       // 铺草皮动画延迟计时器
	sodDebugPrinted    bool          // 草皮叠加图调试日志是否已打印

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

	// Story 3.1: Plant Card Systems
	plantCardSystem       *systems.PlantCardSystem
	plantCardRenderSystem *systems.PlantCardRenderSystem

	// Story 3.2: Plant Preview Systems
	plantPreviewSystem       *systems.PlantPreviewSystem
	plantPreviewRenderSystem *systems.PlantPreviewRenderSystem

	// Story 3.3: Lawn Grid System
	lawnGridSystem   *systems.LawnGridSystem // 草坪网格管理系统
	lawnGridEntityID ecs.EntityID            // 草坪网格实体ID

	// Story 3.4: Behavior System
	behaviorSystem *systems.BehaviorSystem // 植物行为系统（向日葵生产阳光等）

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
	rewardSystem  *systems.RewardAnimationSystem  // 奖励动画系统（关卡完成奖励）

	// Story 8.4: Reward Panel Render System
	rewardPanelRenderSystem *systems.RewardPanelRenderSystem // 奖励面板渲染系统（新植物介绍）
}

// NewGameScene creates and returns a new GameScene instance.
// It loads all necessary UI resources and initializes the game scene.
//
// Parameters:
//   - rm: The ResourceManager instance used to load game resources.
//   - sm: The SceneManager instance used to switch between scenes.
//
// Returns:
//   - A pointer to the newly created GameScene.
//
// If any UI resources fail to load, the scene will use fallback rendering methods.
func NewGameScene(rm *game.ResourceManager, sm *game.SceneManager) *GameScene {
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

	// Story 5.5 & 8.1: Load level configuration FIRST (before creating systems that depend on it)
	// CRITICAL: This must happen before:
	//   1. LawnGridSystem (needs EnabledLanes)
	//   2. initPlantCardSystems() (needs AvailablePlants)
	//   3. WaveSpawnSystem (needs wave configuration)
	levelConfig, err := config.LoadLevelConfig("data/levels/level-1-1.yaml")
	if err != nil {
		log.Printf("[GameScene] FATAL: Failed to load level config: %v", err)
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

	// Calculate sun collection target position from sun counter UI position
	// This ensures the suns fly to the exact center of the sun counter display
	sunCollectionTargetX := float64(SeedBankX + SunCounterOffsetX)
	sunCollectionTargetY := float64(SeedBankY + SunCounterOffsetY)

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
		scene.reanimSystem, // Story 8.2 QA修复：传入 ReanimSystem 用于初始化阳光动画
		250.0,              // minX - 草坪左边界
		900.0,              // maxX - 草坪右边界
		100.0,              // minTargetY - 草坪上边界
		550.0,              // maxTargetY - 草坪下边界
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
	scene.plantPreviewRenderSystem = systems.NewPlantPreviewRenderSystem(scene.entityManager, scene.plantPreviewSystem)

	// Story 3.4: Initialize behavior system (sunflower sun production, etc.)
	// Story 6.3: Pass ReanimSystem for zombie animation state changes
	// Story 5.5: Pass GameState for zombie death counting
	scene.behaviorSystem = systems.NewBehaviorSystem(scene.entityManager, rm, scene.reanimSystem, scene.gameState)
	log.Printf("[GameScene] Initialized behavior system for plant behaviors")

	// Story 4.3: Initialize physics system (collision detection)
	scene.physicsSystem = systems.NewPhysicsSystem(scene.entityManager, rm)
	log.Printf("[GameScene] Initialized physics system for collision detection")

	// Story 5.5: Initialize level management systems
	// 1. Create WaveSpawnSystem (LevelSystem depends on it)
	scene.waveSpawnSystem = systems.NewWaveSpawnSystem(scene.entityManager, rm, scene.reanimSystem, scene.gameState.CurrentLevel)
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

	// Story 8.3: Create RewardAnimationSystem (需要 ReanimSystem 和 ParticleSystem)
	scene.rewardSystem = systems.NewRewardAnimationSystem(scene.entityManager, scene.gameState, rm, scene.reanimSystem, scene.particleSystem)
	log.Printf("[GameScene] Initialized reward animation system")

	// Story 8.4: Create RewardPanelRenderSystem (新植物介绍面板渲染)
	scene.rewardPanelRenderSystem = systems.NewRewardPanelRenderSystem(scene.entityManager, scene.gameState, rm, scene.reanimSystem)
	log.Printf("[GameScene] Initialized reward panel render system")

	// Story 8.3: Create OpeningAnimationSystem (conditionally, may return nil)
	scene.openingSystem = systems.NewOpeningAnimationSystem(scene.entityManager, scene.gameState, rm, levelConfig, scene.cameraSystem, scene.reanimSystem)
	if scene.openingSystem != nil {
		log.Printf("[GameScene] Initialized opening animation system")
	} else {
		log.Printf("[GameScene] Skipping opening animation system (tutorial/skip/special level)")
	}

	// 2. Create LevelSystem (需要 RewardAnimationSystem)
	scene.levelSystem = systems.NewLevelSystem(scene.entityManager, scene.gameState, scene.waveSpawnSystem, rm, scene.reanimSystem, scene.rewardSystem)
	log.Printf("[GameScene] Initialized level system")

	// 3. Create ZombieLaneTransitionSystem (僵尸行转换系统)
	scene.zombieLaneTransitionSystem = systems.NewZombieLaneTransitionSystem(scene.entityManager)
	log.Printf("[GameScene] Initialized zombie lane transition system")

	// 方案A+：Initialize flash effect system
	scene.flashEffectSystem = systems.NewFlashEffectSystem(scene.entityManager)
	log.Printf("[GameScene] Initialized flash effect system for hit feedback")

	// Story 8.2: Initialize tutorial system (if this is a tutorial level)
	if scene.gameState.CurrentLevel != nil && scene.gameState.CurrentLevel.OpeningType == "tutorial" && len(scene.gameState.CurrentLevel.TutorialSteps) > 0 {
		scene.tutorialSystem = systems.NewTutorialSystem(scene.entityManager, scene.gameState, scene.resourceManager, scene.reanimSystem, scene.lawnGridSystem, scene.sunSpawnSystem, scene.waveSpawnSystem, scene.gameState.CurrentLevel)
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
	scene.soddingSystem = systems.NewSoddingSystem(scene.entityManager, scene.resourceManager, scene.reanimSystem)
	log.Printf("[GameScene] Initialized sodding animation system")

	return scene
}

// initPlantCardSystems initializes the plant card systems and creates plant card entities.
// Story 3.1: Plant Card UI and State
// Story 8.1: 根据关卡配置创建植物卡片
// Story 8.3: 使用 PlantUnlockManager 统一管理植物可用性
func (s *GameScene) initPlantCardSystems(rm *game.ResourceManager) {
	// Story 8.3: 通过 PlantUnlockManager 统一获取可用植物列表（方案 A）
	availablePlants := s.gameState.GetPlantUnlockManager().GetAvailablePlantsForLevel(s.gameState.CurrentLevel)
	log.Printf("[GameScene] Creating %d plant cards: %v", len(availablePlants), availablePlants)

	// 植物名称到类型的映射
	plantTypeMap := map[string]components.PlantType{
		"sunflower":  components.PlantSunflower,
		"peashooter": components.PlantPeashooter,
		"wallnut":    components.PlantWallnut,
		"cherrybomb": components.PlantCherryBomb,
	}

	// 计算第一张卡片的位置
	firstCardX := float64(SeedBankX + PlantCardStartOffsetX)
	cardY := float64(SeedBankY + PlantCardOffsetY)

	// 根据配置创建植物卡片
	for i, plantName := range availablePlants {
		plantType, ok := plantTypeMap[plantName]
		if !ok {
			log.Printf("Warning: Unknown plant type '%s' in level config, skipping", plantName)
			continue
		}

		// 计算卡片位置（水平排列）
		cardX := firstCardX + float64(i)*PlantCardSpacing

		// 创建卡片
		_, err := entities.NewPlantCardEntity(s.entityManager, rm, s.reanimSystem, plantType, cardX, cardY, PlantCardScale)
		if err != nil {
			log.Printf("Warning: Failed to create %s card: %v", plantName, err)
			// 继续执行，游戏在没有卡片的情况下也能运行（用于测试环境）
		}
	}

	// Initialize PlantCardSystem
	s.plantCardSystem = systems.NewPlantCardSystem(
		s.entityManager,
		s.gameState,
		rm,
	)

	// Initialize PlantCardRenderSystem (Story 6.3 + 8.4: 配置内部封装)
	// 所有内部配置（图标缩放、偏移等）从 config.plant_card_config.go 读取
	s.plantCardRenderSystem = systems.NewPlantCardRenderSystem(
		s.entityManager,
		s.plantCardFont, // 阳光数字字体
	)
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
	font, err := s.resourceManager.LoadFont("assets/fonts/SimHei.ttf", SunCounterFontSize)
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

	// Note: Sun counter background is drawn procedurally for now
	// A dedicated image can be loaded here in the future if needed
}

// loadSoddingResources loads sodding animation resources after level config is loaded.
// Story 8.2 QA改进：铺草皮动画资源加载
//
// This method must be called AFTER the level configuration is loaded,
// because it depends on CurrentLevel.SodRowImage and CurrentLevel.ShowSoddingAnim.
func (s *GameScene) loadSoddingResources() {
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

	// Story 8.2 QA改进：加载草皮叠加图片（RGB + Alpha 合成）
	// 原版 PVZ 使用分离的 RGB 和 Alpha 存储来节省空间：
	// - sod1row.jpg = 彩色草地内容 (JPEG)
	// - sod1row_.png = Alpha 蒙版 (PNG 灰度图，白色=不透明，黑色=透明)
	// - sod3row.jpg = 三行草皮彩色内容
	// - sod3row_.png = 三行草皮 Alpha 蒙版
	if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.SodRowImage != "" {
		// 根据配置选择加载不同的草皮图片
		var rgbPath, alphaPath string
		sodImageID := s.gameState.CurrentLevel.SodRowImage

		switch sodImageID {
		case "IMAGE_SOD1ROW":
			rgbPath = "assets/images/sod1row.jpg"
			alphaPath = "assets/images/sod1row_.png"
		case "IMAGE_SOD3ROW":
			rgbPath = "assets/images/sod3row.jpg"
			alphaPath = "assets/images/sod3row_.png"
		default:
			log.Printf("Warning: Unknown sod row image ID: %s", sodImageID)
			rgbPath = "assets/images/sod1row.jpg"
			alphaPath = "assets/images/sod1row_.png"
		}

		// 合成 RGB + Alpha 图片
		sodRowImage, err := s.resourceManager.LoadImageWithAlphaMask(rgbPath, alphaPath)
		if err != nil {
			log.Printf("Warning: Failed to composite sod row image: %v", err)
		} else {
			s.sodRowImage = sodRowImage
			log.Printf("[GameScene] ✅ 合成草皮叠加图片 (RGB + Alpha): %s", sodImageID)

			// 性能优化：缓存草皮图片尺寸和位置，避免每帧重复计算
			sodBounds := sodRowImage.Bounds()
			s.sodWidth = sodBounds.Dx()
			s.sodHeight = sodBounds.Dy()

			// 计算并缓存草皮世界坐标
			enabledLanes := s.gameState.CurrentLevel.EnabledLanes
			s.sodOverlayX, s.sodOverlayY = config.CalculateSodOverlayPosition(enabledLanes, float64(s.sodHeight))
			log.Printf("[GameScene] 缓存草皮渲染参数: 位置(%.1f, %.1f) 尺寸(%dx%d)", s.sodOverlayX, s.sodOverlayY, s.sodWidth, s.sodHeight)

			// 启动铺草皮动画（如果配置了）
			if s.gameState.CurrentLevel.ShowSoddingAnim {
				// 记录延迟时间，在 Update() 中延迟启动动画
				s.soddingAnimDelay = s.gameState.CurrentLevel.SoddingAnimDelay
				s.soddingAnimStarted = false
				s.soddingAnimTimer = 0
				log.Printf("[GameScene] 设置铺草皮动画延迟: %.1f 秒", s.soddingAnimDelay)
			} else {
				// 不播放动画，草皮在渲染时会以100%进度显示
				log.Printf("[GameScene] 跳过铺草皮动画，草皮将直接显示")
			}
		}
	}
}

// Update updates the game scene logic.
// deltaTime is the time elapsed since the last update in seconds.
//
// This method handles:
//   - Intro animation (camera scrolling left → right → center)
//   - ECS system updates (input, sun spawning, movement, collection, lifetime management)
//   - System execution order ensures correct game logic flow
func (s *GameScene) Update(deltaTime float64) {
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
	if s.openingSystem != nil && s.openingSystem.IsCompleted() && !s.soddingAnimStarted && s.soddingSystem != nil {
		log.Printf("[GameScene] 开场动画完成，启动铺草皮动画")

		// 启动动画，传递启用的行列表、草皮位置和图片高度
		enabledLanes := s.gameState.CurrentLevel.EnabledLanes
		s.soddingSystem.StartAnimation(func() {
			// 动画完成回调：通知教学系统可以开始了
			log.Printf("[GameScene] 铺草皮动画完成")
			if s.tutorialSystem != nil {
				s.tutorialSystem.OnSoddingComplete()
			}
		}, enabledLanes, s.sodOverlayX, float64(s.sodHeight))

		s.soddingAnimStarted = true
		// 标记开场动画系统为 nil，避免重复检查
		s.openingSystem = nil
		return // 等待铺草皮动画完成
	}

	// Story 8.3: 如果铺草皮动画正在播放，暂停其他游戏系统
	if s.soddingSystem != nil && s.soddingSystem.IsPlaying() {
		// 铺草皮动画期间，只更新铺草皮系统、镜头系统和 Reanim 系统（草皮卷动画需要）
		s.cameraSystem.Update(deltaTime)
		s.reanimSystem.Update(deltaTime) // 更新草皮卷动画帧
		s.cameraX = s.gameState.CameraX
		return // 暂停其他游戏系统（包括僵尸激活）
	}

	// 如果没有开场动画，使用延迟启动铺草皮动画（原逻辑）
	if s.openingSystem == nil && s.soddingSystem != nil && !s.soddingAnimStarted && s.soddingAnimDelay > 0 {
		s.soddingAnimTimer += deltaTime
		if s.soddingAnimTimer >= s.soddingAnimDelay {
			log.Printf("[GameScene] 启动铺草皮动画（延迟 %.1f 秒后）", s.soddingAnimDelay)

			// 启动动画，传递启用的行列表、草皮位置和图片高度
			enabledLanes := s.gameState.CurrentLevel.EnabledLanes
			s.soddingSystem.StartAnimation(func() {
				// 动画完成回调：通知教学系统可以开始了
				log.Printf("[GameScene] 铺草皮动画完成")
				if s.tutorialSystem != nil {
					s.tutorialSystem.OnSoddingComplete()
				}
			}, enabledLanes, s.sodOverlayX, float64(s.sodHeight))

			s.soddingAnimStarted = true
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
		return // 停止其他游戏系统（僵尸移动、植物攻击等）
	}

	// Update all ECS systems in order (order matters for correct game logic)
	s.levelSystem.Update(deltaTime)                // 0. Update level system (Story 5.5: wave spawning, victory/defeat)
	s.rewardSystem.Update(deltaTime)               // 0.1. Update reward animation system (Story 8.3: 卡片包动画)
	s.zombieLaneTransitionSystem.Update(deltaTime) // 0.5. Update zombie lane transitions (move to target lane before attacking)
	s.plantCardSystem.Update(deltaTime)            // 1. Update plant card states (before input)
	s.inputSystem.Update(deltaTime, s.cameraX)     // 2. Process player input (highest priority, 传递摄像机位置)

	// 3. Generate new suns
	// 教学关卡：在第一次收集阳光后启用自动生成（由 TutorialSystem 控制）
	// 非教学关卡：始终启用自动生成
	s.sunSpawnSystem.Update(deltaTime)

	s.sunMovementSystem.Update(deltaTime)   // 4. Move suns (includes collection animation)
	s.sunCollectionSystem.Update(deltaTime) // 5. Check if collection is complete
	s.behaviorSystem.Update(deltaTime)      // 6. Update plant behaviors (Story 3.4)
	s.physicsSystem.Update(deltaTime)       // 7. Check collisions (Story 4.3)
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
	progress := s.introAnimTimer / IntroAnimDuration

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

	// Layer 3: Draw plant cards (Story 3.1)
	// 在植物和僵尸下方渲染，符合原版PVZ设计
	s.plantCardRenderSystem.Draw(screen)

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
	// 粒子效果在UI和游戏世界之间，提供视觉特效（爆炸、溅射等）
	// 粒子应该覆盖植物卡片和游戏实体，但在植物预览和阳光之下
	s.renderSystem.DrawParticles(screen, s.cameraX)

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

	// Layer 9: Draw level progress UI (Story 5.5)
	// 进度条显示当前波次进度
	s.drawLevelProgress(screen)

	// Layer 10: Draw last wave warning (Story 5.5)
	// 最后一波提示（如果需要显示）
	s.drawLastWaveWarning(screen)

	// Layer 10.5: Draw reward panel (Story 8.3 + 8.4)
	// 奖励面板（关卡完成后显示新植物介绍）
	// 在游戏结果覆盖层之前渲染，因为它是正常游戏流程的一部分
	s.rewardPanelRenderSystem.Draw(screen)

	// Story 8.3: 移除 "You Win" 覆盖层逻辑
	// 奖励流程完成后通过"下一关"按钮进入下一关，不再显示 You Win
	// Layer 11: Draw game result overlay (Story 5.5) - DISABLED for Story 8.3
	// s.drawGameResultOverlay(screen) // 已禁用：改为通过奖励面板的"下一关"按钮进入下一关

	// DEBUG: Draw particle test instructions (Story 7.4 debugging)
	s.drawParticleTestInstructions(screen)

	// DEBUG: Draw grid boundaries (Story 3.3 debugging)
	s.drawGridDebug(screen)

	// DEBUG: Draw FPS counter to check performance
	fps := ebiten.ActualFPS()
	fpsText := fmt.Sprintf("FPS: %.1f", fps)
	ebitenutil.DebugPrintAt(screen, fpsText, WindowWidth-100, 10)
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

		// Story 8.2 QA改进：在背景上叠加草皮图片（随草皮卷位置同步显示）
		if s.sodRowImage != nil && s.gameState.CurrentLevel != nil {
			// 性能优化：使用缓存的尺寸和位置，避免每帧重复计算
			sodWidth := s.sodWidth
			sodHeight := s.sodHeight
			sodOverlayX := s.sodOverlayX
			sodOverlayY := s.sodOverlayY

			// 获取草皮卷当前位置（世界坐标X，中心）
			// 草皮可见宽度 = 草皮卷中心X - 动画起点X
			var sodRollCenterX float64
			var animStartX float64
			if s.soddingSystem != nil {
				sodRollCenterX = s.soddingSystem.GetSodRollCenterX() // 返回草皮卷中心X
				animStartX = s.soddingSystem.GetAnimStartX()         // 返回动画起点X
			} else {
				// 没有 soddingSystem：不显示草皮
				sodRollCenterX = sodOverlayX - 10
				animStartX = sodOverlayX
			}

			// 根据草皮卷中心位置计算可见宽度（从动画起点到草皮卷中心）
			// visibleWidth = sodRollCenterX（草皮卷中心） - animStartX（动画起点）
			visibleWidth := int(sodRollCenterX - animStartX)
			if visibleWidth > sodWidth {
				visibleWidth = sodWidth
			}
			if visibleWidth <= 0 {
				return // 草皮卷还没到，不显示
			}

			// 计算视口裁剪区域（水平方向）
			sodViewportX := viewportX - int(sodOverlayX)
			if sodViewportX < 0 {
				sodViewportX = 0
			}

			// 限制裁剪区域不超过可见宽度（草皮卷位置控制）
			sodViewportEndX := sodViewportX + WindowWidth
			if sodViewportEndX > visibleWidth {
				sodViewportEndX = visibleWidth
			}

			// 垂直方向：裁剪草皮图片的全部高度
			sodViewportRect := image.Rect(
				sodViewportX,
				0, // 草皮图片从顶部开始裁剪
				sodViewportEndX,
				sodHeight, // 裁剪整个高度
			)

			visibleSod := s.sodRowImage.SubImage(sodViewportRect).(*ebiten.Image)

			// 计算草皮在屏幕上的绘制位置
			screenX := sodOverlayX - float64(viewportX)
			screenY := sodOverlayY - float64(viewportY)

			// Story 8.2 QA调试：打印草皮叠加图的坐标信息（每次都打印，便于对比）
			// log.Printf("=== 草皮叠加图渲染调试 ===")
			// log.Printf("草皮图尺寸: %dx%d", sodWidth, sodHeight)
			// log.Printf("启用行: %v", s.gameState.CurrentLevel.EnabledLanes)
			// log.Printf("草皮左边缘: %.1f, 草皮右边缘: %.1f (宽度%.1f)", sodOverlayX, sodOverlayX+float64(sodWidth), float64(sodWidth))
			// log.Printf("动画起点: %.1f", animStartX)
			// log.Printf("草皮卷中心: %.1f", sodRollCenterX)
			// log.Printf("草皮可见宽度: %d/%d px (%.1f%%)", visibleWidth, sodWidth, float64(visibleWidth)/float64(sodWidth)*100)
			// log.Printf("草皮应显示到: %.1f (起点%.1f + 可见宽度%d)", animStartX+float64(visibleWidth), animStartX, visibleWidth)
			// log.Printf("差距: 草皮卷中心 - 草皮应显示到 = %.1f - %.1f = %.1f px",
			// 	sodRollCenterX, animStartX+float64(visibleWidth), sodRollCenterX-(animStartX+float64(visibleWidth)))
			// log.Printf("屏幕坐标: (%.1f, %.1f)", screenX, screenY)

			if !s.sodDebugPrinted {
				log.Printf("草皮世界坐标: (%.1f, %.1f)", sodOverlayX, sodOverlayY)

				// 计算草皮覆盖的行范围
				sodTopY := sodOverlayY
				sodBottomY := sodOverlayY + float64(sodHeight)
				log.Printf("草皮Y范围: [%.1f - %.1f]", sodTopY, sodBottomY)

				// 对比各行的Y范围
				for row := 1; row <= 5; row++ {
					rowStartY := config.GridWorldStartY + float64(row-1)*config.CellHeight
					rowEndY := rowStartY + config.CellHeight
					rowCenterY := rowStartY + config.CellHeight/2

					// 检查草皮是否覆盖这一行（使用严格不等式，边界接触不算覆盖）
					overlaps := sodTopY < rowEndY && sodBottomY > rowStartY
					overlap := ""
					if overlaps {
						overlap = " ← 草皮覆盖"
					}
					log.Printf("第%d行: Y[%.1f - %.1f] 中心=%.1f%s", row, rowStartY, rowEndY, rowCenterY, overlap)
				}

				s.sodDebugPrinted = true
			}

			// 绘制草皮图片到正确位置
			sodOp := &ebiten.DrawImageOptions{}
			sodOp.GeoM.Translate(screenX, screenY)
			screen.DrawImage(visibleSod, sodOp)
		}
	} else {
		// Fallback: Draw a green background to simulate grass
		screen.Fill(color.RGBA{R: 34, G: 139, B: 34, A: 255}) // Forest green
	}
}

// drawSeedBank renders the plant selection bar at the top left of the screen.
// If the seed bank image is not loaded, it draws a simple rectangle as fallback.
func (s *GameScene) drawSeedBank(screen *ebiten.Image) {
	if s.seedBank != nil {
		// Draw the seed bank image at the top left corner
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(SeedBankX, SeedBankY)
		screen.DrawImage(s.seedBank, op)
	} else {
		// Fallback: Draw a dark brown rectangle
		ebitenutil.DrawRect(screen,
			SeedBankX, SeedBankY,
			SeedBankWidth, SeedBankHeight,
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
		centerX := float64(SeedBankX + SunCounterOffsetX)
		centerY := float64(SeedBankY + SunCounterOffsetY)

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
		sunDisplayX := SeedBankX + SunCounterOffsetX
		sunDisplayY := SeedBankY + SunCounterOffsetY
		ebitenutil.DebugPrintAt(screen, sunText, sunDisplayX, sunDisplayY)
	}
}

// drawShovel renders the shovel slot and icon at the right side of the seed bank.
// The shovel will be used in future stories for removing plants.
func (s *GameScene) drawShovel(screen *ebiten.Image) {
	// Draw shovel slot background first
	if s.shovelSlot != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(ShovelX, ShovelY)
		screen.DrawImage(s.shovelSlot, op)
	}

	// Draw shovel icon on top of the slot
	if s.shovel != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(ShovelX, ShovelY)
		screen.DrawImage(s.shovel, op)
	} else if s.shovelSlot == nil {
		// Fallback: Draw a gray rectangle if both images are missing
		ebitenutil.DrawRect(screen,
			ShovelX, ShovelY,
			ShovelWidth, ShovelHeight,
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
		"[粒子测试] P=豌豆溅射 | B=爆炸 | A=奖励光效 | Z=僵尸头",
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
	// Story 8.2 QA: 临时启用调试绘制，验证草坪布局
	// 只在种植模式（选中植物卡片）时显示网格线
	if !s.gameState.IsPlantingMode {
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

		// Story 8.2 QA：调试可视化（已禁用）
		/*
			// 绘制草皮边界（红色矩形框，不填充）
			sodColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
			thickness := 3.0
			// 顶边
			ebitenutil.DrawRect(screen, sodScreenX, sodScreenY, sodWidth, thickness, sodColor)
			// 底边
			ebitenutil.DrawRect(screen, sodScreenX, sodScreenY+sodHeight-thickness, sodWidth, thickness, sodColor)
			// 左边
			ebitenutil.DrawRect(screen, sodScreenX, sodScreenY, thickness, sodHeight, sodColor)
			// 右边
			ebitenutil.DrawRect(screen, sodScreenX+sodWidth-thickness, sodScreenY, thickness, sodHeight, sodColor)

			// 绘制草皮卷中心位置标记（绿色十字）
			if s.soddingSystem != nil && s.soddingSystem.IsPlaying() {
				sodRollCenterX := s.soddingSystem.GetSodRollCenterX()
				sodRollScreenX := sodRollCenterX - s.cameraX

				// 绘制绿色十字标记（中心位置）
				crossSize := 10.0
				crossColor := color.RGBA{R: 0, G: 255, B: 0, A: 255}
				// 竖线
				ebitenutil.DrawRect(screen, sodRollScreenX-1, sodScreenY-crossSize, 2, sodHeight+crossSize*2, crossColor)
				// 横线（在草皮中间）
				midY := sodScreenY + sodHeight/2.0
				ebitenutil.DrawRect(screen, sodRollScreenX-crossSize, midY-1, crossSize*2, 2, crossColor)
			}
		*/

		// 详细调试信息
		debugInfo := fmt.Sprintf("Sod: world(%.0f,%.0f) screen(%.0f,%.0f) size(%.0fx%.0f) cam(%.0f)",
			sodOverlayX, sodOverlayY, sodScreenX, sodScreenY, sodWidth, sodHeight, s.cameraX)
		ebitenutil.DebugPrintAt(screen, debugInfo, 10, 30)

		// 草皮卷中心位置调试信息
		if s.soddingSystem != nil && s.soddingSystem.IsPlaying() {
			sodRollCenterX := s.soddingSystem.GetSodRollCenterX() // 返回中心X坐标
			progress := s.soddingSystem.GetProgress()
			capInfo := fmt.Sprintf("草皮卷中心: world(%.0f) screen(%.0f) progress(%.1f%%)",
				sodRollCenterX, sodRollCenterX-s.cameraX, progress*100)
			ebitenutil.DebugPrintAt(screen, capInfo, 10, 50)
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
