package scenes

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/modules"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/decker502/pvz/pkg/systems/behavior"
	"github.com/hajimehoshi/ebiten/v2"
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

	// Story 8.2.1: 草皮闪烁调试标志
	lawnFlashLogged bool // 是否已记录草皮闪烁信息（避免重复日志）

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
	cameraSystem        *systems.CameraSystem           // 镜头控制系统（镜头移动、缓动）
	openingSystem       *systems.OpeningAnimationSystem // 开场动画系统（僵尸预告、跳过）
	readySetPlantSystem *systems.ReadySetPlantSystem    // ReadySetPlant 动画系统（铺草皮后播放）

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

	// Story 8.8: Zombies Won Flow System (僵尸获胜流程系统)
	zombiesWonPhaseSystem *systems.ZombiesWonPhaseSystem // 僵尸获胜四阶段流程系统

	// Dialog Systems (对话框系统 - ECS ���构)
	dialogInputSystem  *systems.DialogInputSystem  // 对话框输入系统（处理对话框交互）
	dialogRenderSystem *systems.DialogRenderSystem // 对话框渲染系统（渲染对话框和按钮）

	// Cursor state tracking (光标状态追踪)
	lastCursorShape ebiten.CursorShapeType // 上一帧的光标形状（避免不必要的API调用）

	// Story 8.3.1: 僵尸预生成状态标志
	// 实际关卡僵尸在开场动画完成后才预生成，与预览僵尸完全独立
	zombiesPreSpawned bool

	// Story 18.3: 战斗存档对话框
	hasBattleSave         bool                 // 是否有战斗存档
	battleSaveInfo        *game.BattleSaveInfo // 战斗存档信息
	battleSaveDialogShown bool                 // 对话框是否已显示
	battleSaveDialogID    ecs.EntityID         // 对话框实体ID
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

	// Reset game state flags for new session
	scene.gameState.SetPaused(false)
	scene.gameState.IsGameOver = false
	scene.gameState.GameResult = ""

	// Story 18.3: 检测战斗存档（进入游戏后立即检测）
	saveManager := scene.gameState.GetSaveManager()
	currentUser := saveManager.GetCurrentUser()
	if currentUser != "" && saveManager.HasBattleSave(currentUser) {
		scene.hasBattleSave = true
		scene.battleSaveInfo, _ = saveManager.GetBattleSaveInfo(currentUser)
		if scene.battleSaveInfo != nil {
			log.Printf("[GameScene] 检测到战斗存档: 关卡=%s, 波次=%d, 阳光=%d",
				scene.battleSaveInfo.LevelID,
				scene.battleSaveInfo.WaveIndex+1,
				scene.battleSaveInfo.Sun)
		}
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

	// Story 5.4.1: 设置资源加载器，用于运行时单位切换（如僵尸切换到烧焦僵尸）
	scene.reanimSystem.SetResourceLoader(rm)

	// ✅ 修复：设置 ReanimSystem 引用，以便 RenderSystem 调用 GetRenderData()
	scene.renderSystem.SetReanimSystem(scene.reanimSystem)

	// Story 8.8 - Task 6: 设置 ResourceManager 引用，以便 RenderSystem 加载房门图片
	scene.renderSystem.SetResourceManager(rm)

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
	// 使用配置常量确保阳光完整显示在屏幕内
	scene.sunSpawnSystem = systems.NewSunSpawnSystem(
		scene.entityManager,
		rm,
		config.SkyDropSunMinX,       // minX - 阳光中心最小 X 坐标
		config.SkyDropSunMaxX,       // maxX - 阳光中心最大 X 坐标
		config.SkyDropSunMinTargetY, // minTargetY - 阳光落地最小 Y 坐标
		config.SkyDropSunMaxTargetY, // maxTargetY - 阳光落地最大 Y 坐标
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
	// Story 17.3: Load spawn rules config (optional, nil means no constraint checking)
	spawnRules, err := config.LoadSpawnRules("data/spawn_rules.yaml")
	if err != nil {
		log.Printf("[GameScene] Warning: Failed to load spawn rules: %v (constraint checking disabled)", err)
		spawnRules = nil
	}
	// Story 17.9: Load zombie physics config (optional, nil means use default coordinates)
	zombiePhysics, err := config.LoadZombiePhysicsConfig("data/zombie_physics.yaml")
	if err != nil {
		log.Printf("[GameScene] Warning: Failed to load zombie physics config: %v (using default coordinates)", err)
		zombiePhysics = nil
	}
	scene.waveSpawnSystem = systems.NewWaveSpawnSystem(scene.entityManager, rm, scene.gameState.CurrentLevel, scene.gameState, spawnRules, zombiePhysics)
	log.Printf("[GameScene] Initialized wave spawn system (spawn rules enabled: %v, physics config enabled: %v)", spawnRules != nil, zombiePhysics != nil)

	// Pre-spawn all zombies for the level (they will be activated wave by wave)
	// Story 8.3.1: 僵尸预生成时机取决于是否有开场动画
	// - 有开场动画：延迟到开场动画完成后预生成（见 Update() 方法）
	// - 无开场动画：立即预生成
	// 注意：预览僵尸由 OpeningAnimationSystem 独立生成，与此处的关卡僵尸完全独立

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
		// Story 8.3.1: 有开场动画时，僵尸预生成延迟到动画完成后
		// zombiesPreSpawned 保持为 false，在 Update() 中检测动画完成后预生成
	} else {
		log.Printf("[GameScene] Skipping opening animation system (tutorial/skip/special level)")
		// 实时生成模式：不再预生成僵尸，由 WaveTimingSystem 触发时实时生成
	}

	// Create ReadySetPlantSystem (在铺草皮完成、UI 显示后播放)
	scene.readySetPlantSystem = systems.NewReadySetPlantSystem(scene.entityManager, rm)
	log.Printf("[GameScene] Initialized ReadySetPlant animation system")

	// Story 10.2: Create LawnmowerSystem (除草车系统 - 最后防线)
	// Story 10.3: 传递 ReanimSystem 用于播放僵尸死亡动画
	scene.lawnmowerSystem = systems.NewLawnmowerSystem(scene.entityManager, rm, scene.gameState)
	log.Printf("[GameScene] Initialized lawnmower system")

	// Story 10.2: 除草车实体将在铺草皮动画完成后创建（见铺草皮回调）
	// 原版行为：草皮铺完后才显示除草车

	// 2. Create LevelSystem (需要 RewardAnimationSystem 和 LawnmowerSystem)
	// Story 14.3: Epic 14 - Removed ReanimSystem dependency
	scene.levelSystem = systems.NewLevelSystem(scene.entityManager, scene.gameState, scene.waveSpawnSystem, rm, scene.rewardSystem, scene.lawnmowerSystem)
	// Story 17.9: 设置僵尸物理配置（用于类型化进家判定）
	if zombiePhysics != nil {
		scene.levelSystem.SetZombiePhysicsConfig(zombiePhysics)
	}
	log.Printf("[GameScene] Initialized level system")

	// Story 11.3: Create FinalWaveWarningSystem (最后一波提示系统)
	scene.finalWaveWarningSystem = systems.NewFinalWaveWarningSystem(scene.entityManager)
	log.Printf("[GameScene] Initialized final wave warning system")

	// Story 8.8: Create ZombiesWonPhaseSystem (僵尸获胜流程系统)
	scene.zombiesWonPhaseSystem = systems.NewZombiesWonPhaseSystem(
		scene.entityManager,
		scene.resourceManager,
		scene.gameState,
		WindowWidth,
		WindowHeight,
	)
	log.Printf("[GameScene] Initialized zombies won phase system")
	// Story 8.8: 设置"再次尝试"回调
	scene.zombiesWonPhaseSystem.SetRetryCallback(func() {
		scene.retryLevel()
	})

	// 3. Create ZombieLaneTransitionSystem (僵尸行转换系统)
	scene.zombieLaneTransitionSystem = systems.NewZombieLaneTransitionSystem(scene.entityManager)
	log.Printf("[GameScene] Initialized zombie lane transition system")

	// 方案A+：Initialize flash effect system
	scene.flashEffectSystem = systems.NewFlashEffectSystem(scene.entityManager)
	log.Printf("[GameScene] Initialized flash effect system for hit feedback")

	// Story 8.2: Initialize tutorial system (if this is a tutorial level)
	if scene.gameState.CurrentLevel != nil && len(scene.gameState.CurrentLevel.TutorialSteps) > 0 {
		scene.tutorialSystem = systems.NewTutorialSystem(scene.entityManager, scene.gameState, scene.resourceManager, scene.lawnGridSystem, scene.sunSpawnSystem, scene.gameState.CurrentLevel)
		// Story 17.6+统一：设置 LevelSystem 引用，用于访问 WaveTimingSystem
		scene.tutorialSystem.SetLevelSystem(scene.levelSystem)
		log.Printf("[GameScene] Tutorial system activated for level %s", scene.gameState.CurrentLevel.ID)

		// 仅强制性教学关卡禁用自动阳光生成
		if scene.gameState.CurrentLevel.OpeningType == "tutorial" {
			// 禁用自动阳光生成（第一次收集阳光后由 TutorialSystem 启用）
			scene.sunSpawnSystem.Disable()
			log.Printf("[GameScene] Tutorial level: suns will be spawned by tutorial system")
		}

		// Load tutorial font (使用简体中文黑体字体 SimHei.ttf)
		ttFont, err := scene.resourceManager.LoadFont("assets/fonts/SimHei.ttf", 28)
		if err != nil {
			log.Printf("FATAL: Failed to load tutorial font SimHei.ttf: %v", err)
		} else {
			scene.tutorialFont = ttFont
			log.Printf("[GameScene] Loaded tutorial font: SimHei.ttf (28px)")
		}
	}

	// Story 8.2 QA改进：初始化铺草皮动画系统
	scene.soddingSystem = systems.NewSoddingSystem(scene.entityManager, scene.resourceManager)
	log.Printf("[GameScene] Initialized sodding animation system")

	// 按钮系统初始化（ECS 架构）
	scene.buttonSystem = systems.NewButtonSystem(scene.entityManager)
	scene.buttonRenderSystem = systems.NewButtonRenderSystem(scene.entityManager)
	log.Printf("[GameScene] Initialized button systems")

	// 对话框系统初始化（ECS 架构）
	// Story 8.8: Load dialog fonts for DialogRenderSystem
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

	scene.dialogInputSystem = systems.NewDialogInputSystem(scene.entityManager)
	scene.dialogRenderSystem = systems.NewDialogRenderSystem(scene.entityManager, WindowWidth, WindowHeight, titleFont, messageFont, buttonFont)
	log.Printf("[GameScene] Initialized dialog systems (input + render)")

	// 创建菜单按钮实体
	scene.initMenuButton(rm)

	// Story 10.1: 初始化暂停菜单系统
	scene.initPauseMenuModule(rm)

	// Story 11.2: 初始化关卡进度条系统
	scene.initProgressBar(rm)

	return scene
}

// NewGameSceneFromBattleSave 创建从战斗存档恢复的游戏场景
//
// Story 18.3: 简化实现
//
// 说明：
//
//	现在 NewGameScene 会自动检测战斗存档并处理，
//	此函数仅作为别名保留以保持向后兼容性。
//
// 参数：
//   - rm: 资源管理器
//   - sm: 场景管理器
//   - levelID: 关卡ID（从存档中获取）
//
// 返回：
//   - GameScene 实例（会自动检测存档并显示对话框）
func NewGameSceneFromBattleSave(rm *game.ResourceManager, sm *game.SceneManager, levelID string) *GameScene {
	log.Printf("[GameScene] NewGameSceneFromBattleSave 调用，将使用标准构造函数: level=%s", levelID)
	// 直接使用标准构造函数，它会自动检测存档并处理
	return NewGameScene(rm, sm, levelID)
}

// initPlantCardSystems initializes the plant selection module.
// Story 3.1 架构优化：使用 PlantSelectionModule 统一管理所有选卡功能
// Story 8.3: 使用 PlantUnlockManager 统一管理植物可用性

// This method handles:
//   - Intro animation (camera scrolling left → right → center)
//   - ECS system updates (input, sun spawning, movement, collection, lifetime management)
//   - System execution order ensures correct game logic flow
//   - Story 10.1: Pause menu (只更新 UI 系统，跳过游戏逻辑)
func (s *GameScene) Update(deltaTime float64) {
	// Story 18.3: 战斗存档对话框优先于所有其他逻辑
	// 如果有战斗存档且对话框未显示，先显示对话框
	if s.hasBattleSave && !s.battleSaveDialogShown {
		s.showBattleSaveDialog()
		s.battleSaveDialogShown = true
		return // 等待对话框显示
	}

	// 如果对话框正在显示，只更新对话框系统
	if s.battleSaveDialogID != 0 {
		if s.dialogInputSystem != nil {
			s.dialogInputSystem.Update(deltaTime)
			s.entityManager.RemoveMarkedEntities()
		}
		// 检查对话框是否已关闭
		if _, ok := ecs.GetComponent[*components.DialogComponent](s.entityManager, s.battleSaveDialogID); !ok {
			s.battleSaveDialogID = 0 // 对话框已关闭
		}
		s.updateMouseCursor()
		return // 对话框打开时阻止其他更新
	}

	// DEBUG: Check for GameFreezeComponent on every frame to debug freeze issue
	freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](s.entityManager)
	if len(freezeEntities) > 0 && s.zombiesWonPhaseSystem == nil {
		log.Printf("[GameScene] ⚠️ WARNING: GameFreezeComponent found but ZombiesWonPhaseSystem is nil! Count: %d", len(freezeEntities))
	}

	// Story 10.1: 更新暂停菜单模块
	if s.pauseMenuModule != nil {
		s.pauseMenuModule.Update(deltaTime)
	}

	// Story 10.1: Check if game is paused
	if s.gameState.IsPaused {
		// 暂停时只更新 UI 系统（按钮交互、暂停菜单、对话框）
		if s.buttonSystem != nil {
			s.buttonSystem.Update(deltaTime)
		}
		// ✅ ECS 架构修复: 更新对话框输入系统（暂停菜单可能包含对话框）
		if s.dialogInputSystem != nil {
			s.dialogInputSystem.Update(deltaTime)
			s.entityManager.RemoveMarkedEntities()
		}
		// ✅ 暂停时也需要更新鼠标光标（按钮悬停效果）
		s.updateMouseCursor()
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
		// 实时生成模式：不再预生成僵尸，由 WaveTimingSystem 触发时实时生成

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
				// Story 8.2.1 修复：保留 sodRowImage 用于草坪闪烁效果
				if s.soddedBackground != nil {
					// Level 1-4: 有完整的已铺草皮背景，直接替换
					log.Printf("[GameScene] 替换底层背景: IMAGE_BACKGROUND1UNSODDED → IMAGE_BACKGROUND1")
					s.background = s.soddedBackground
					s.soddedBackground = nil
					s.preSoddedImage = nil
					// Story 8.2.1: 保留 sodRowImage 用于草坪闪烁
					log.Printf("[GameScene] 保留 sodRowImage 用于草坪闪烁效果")
				} else if s.preSoddedImage != nil || s.sodRowImage != nil {
					// Level 1-1, 1-2: 需要将草皮叠加层合并到底层背景
					log.Printf("[GameScene] 合并草皮叠加层到底层背景")
					mergedBg := s.createMergedBackground()
					if mergedBg != nil {
						// 原子操作：先替换背景，再清空preSoddedImage
						s.background = mergedBg
						s.preSoddedImage = nil
						// Story 8.2.1: 保留 sodRowImage 用于草坪闪烁，不清空
						log.Printf("[GameScene] 背景合并完成，保留 sodRowImage 用于草坪闪烁")
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

				// 铺草皮完成后播放 ReadySetPlant 动画（仅限配置启用的关卡）
				// 此时 UI（植物选择栏、除草车）已显示
				if s.readySetPlantSystem != nil && s.gameState.CurrentLevel.ShowReadySetPlant {
					s.readySetPlantSystem.Start()
				}
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

			// 播放 ReadySetPlant 动画（仅限配置启用的关卡）
			if s.readySetPlantSystem != nil && s.gameState.CurrentLevel.ShowReadySetPlant {
				s.readySetPlantSystem.Start()
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
					// Story 8.2.1 修复：保留 sodRowImage 用于草坪闪烁效果
					if s.soddedBackground != nil {
						// Level 1-4: 有完整的已铺草皮背景，直接替换
						log.Printf("[GameScene] 替换底层背景: IMAGE_BACKGROUND1UNSODDED → IMAGE_BACKGROUND1")
						s.background = s.soddedBackground
						s.soddedBackground = nil
						s.preSoddedImage = nil
						// Story 8.2.1: 保留 sodRowImage 用于草坪闪烁
						log.Printf("[GameScene] 保留 sodRowImage 用于草坪闪烁效果")
					} else if s.preSoddedImage != nil || s.sodRowImage != nil {
						// Level 1-1, 1-2: 需要将草皮叠加层合并到底层背景
						log.Printf("[GameScene] 合并草皮叠加层到底层背景")
						mergedBg := s.createMergedBackground()
						if mergedBg != nil {
							// 原子操作：先替换背景，再清空preSoddedImage
							s.background = mergedBg
							s.preSoddedImage = nil
							// Story 8.2.1: 保留 sodRowImage 用于草坪闪烁，不清空
							log.Printf("[GameScene] 背景合并完成，保留 sodRowImage 用于草坪闪烁")
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
	// Story 8.8: 如果游戏结束且 ZombiesWonPhaseSystem 激活，则反向同步（让系统控制摄像机）
	if s.gameState.IsGameOver {
		// 游戏结束时，ZombiesWonPhaseSystem 控制摄像机移动
		// 所以需要反向同步：从 GameState.CameraX 更新到 s.cameraX
		s.cameraX = s.gameState.CameraX
	} else {
		// 正常游戏时，GameScene 控制摄像机
		s.gameState.CameraX = s.cameraX
	}

	// Story 5.5: Check if game is over (win or lose)
	// If game is over, stop updating game systems but allow reward animation to play
	if s.gameState.IsGameOver {
		// 游戏结束时仍然更新奖励系统和必要的动画系统
		// 这样玩家可以看到完整的奖励动画流程
		s.rewardSystem.Update(deltaTime)   // 奖励动画系统（卡片包动画）
		s.reanimSystem.Update(deltaTime)   // Reanim 系统（植物卡片动画）
		s.particleSystem.Update(deltaTime) // 粒子系统（光晕效果）

		// Story 10.6: 除草车系统（压扁动画需要继续播放）
		if s.lawnmowerSystem != nil {
			s.lawnmowerSystem.Update(deltaTime)
		}

		// Story 8.8: 僵尸获胜流程需要继续更新
		if s.zombiesWonPhaseSystem != nil {
			s.zombiesWonPhaseSystem.Update(deltaTime)
		}
		// Story 8.8: 触发僵尸需要继续移动（BehaviorSystem 会检测冻结状态）
		s.behaviorSystem.Update(deltaTime)

		// Story 8.8: 游戏结束时也需要更新对话框输入系统（处理按钮点击）
		if s.dialogInputSystem != nil {
			s.dialogInputSystem.Update(deltaTime)
			s.entityManager.RemoveMarkedEntities()
		}
		// 更新鼠标光标（按钮悬停效果）
		s.updateMouseCursor()

		return // 停止其他游戏系统（僵尸移动、植物攻击等）
	}

	// Update all ECS systems in order (order matters for correct game logic)
	s.levelSystem.Update(deltaTime)            // 0. Update level system (Story 5.5: wave spawning, victory/defeat)
	s.rewardSystem.Update(deltaTime)           // 0.1. Update reward animation system (Story 8.3: 卡片包动画)
	s.finalWaveWarningSystem.Update(deltaTime) // 0.2. Update final wave warning (Story 11.3: 自动清理提示动画)
	if s.zombiesWonPhaseSystem != nil {
		s.zombiesWonPhaseSystem.Update(deltaTime) // 0.3. Update zombies won flow (Story 8.8: 僵尸获胜四阶段流程)
	}
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
	s.reanimSystem.Update(deltaTime) // 8. Update Reanim animation frames
	// Story 8.3: ReadySetPlant 动画系统（铺草皮完成后播放）
	if s.readySetPlantSystem != nil {
		s.readySetPlantSystem.Update(deltaTime) // 8.5. Update ReadySetPlant animation duration
	}

	s.particleSystem.Update(deltaTime) // 9. Update particle effects (Story 7.2)
	// 方案A+：闪烁效果系统
	s.flashEffectSystem.Update(deltaTime) // 9.3. Update flash effects (hit feedback)
	// Story 10.8: 更新阳光计数器闪烁计时器
	s.gameState.UpdateSunFlash(deltaTime) // 9.4. Update sun flash timer (sun shortage feedback)
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

	// 13. Update mouse cursor based on component states
	s.updateMouseCursor()
}

// updateIntroAnimation updates the intro camera animation that showcases the entire lawn.
// The animation has two phases:
//   - Phase 1 (0.0-0.5): Camera scrolls from left edge (0) to right edge (maxCameraX)
//   - Phase 2 (0.5-1.0): Camera scrolls back from right edge to gameplay position (GameCameraX)
//
// Both phases use an ease-out quadratic easing function for smooth motion.

// easeOutQuad applies an ease-out quadratic easing function to the input value.
// Formula: 1 - (1-t)^2
// This creates a smooth deceleration effect.
//
// Parameters:
//   - t: Input value in range [0, 1]
//
// Returns:
//   - Eased value in range [0, 1]

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

	// Story 8.3.1: 开场动画或铺草皮动画期间隐藏 UI 元素
	// 注意：需要检查 soddingAnimStarted 来避免开场动画完成和铺草皮动画开始之间的闪现
	isOpeningPlaying := s.openingSystem != nil && !s.openingSystem.IsCompleted()
	isSoddingPlaying := s.soddingSystem != nil && s.soddingSystem.IsPlaying()
	// 如果有开场动画系统且铺草皮动画还未启动，也要隐藏 UI（过渡期间）
	isWaitingForSodding := s.openingSystem != nil && s.openingSystem.IsCompleted() && !s.soddingAnimStarted
	hideUI := isOpeningPlaying || isSoddingPlaying || isWaitingForSodding

	// Layer 2: Draw UI base elements (seed bank, shovel, plant cards)
	// 按照原版PVZ设计，UI元素在游戏世界实体下方渲染
	if !hideUI {
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
	}

	// Layer 4: Draw game world entities (plants, zombies, projectiles) - 不包括阳光
	// 游戏实体在UI卡片上方，这样植物和僵尸可以被看清
	// 传递 cameraX 以正确转换世界坐标到屏幕坐标
	s.renderSystem.DrawGameWorld(screen, s.cameraX)

	// 开场动画用户名显示（在游戏世界之后、其他UI之前）
	// 显示 "{username}的房子" 文本，白色带黑色阴影
	if s.openingSystem != nil {
		s.openingSystem.Draw(screen)
	}

	// Layer 4.5: Draw lawn flash effect (Story 8.2 教学)
	// 草坪闪烁效果，用于教学提示玩家可以种植
	s.drawLawnFlash(screen)

	// Story 8.2.1: Draw card flash effect (教学)
	// 卡片闪烁效果，用于提示玩家点击卡片
	s.drawCardFlash(screen)

	// Layer 5: Draw UI overlays (sun counter text)
	// 文字始终在最上层以确保可读性
	// Story 8.3.1: 开场动画或铺草皮动画期间隐藏阳光计数器
	if !hideUI {
		s.drawSunCounter(screen)
	}

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
	// Story 8.3.1: 开场动画或铺草皮动画期间隐藏进度条
	if !hideUI {
		s.drawProgressBar(screen)
	}

	// Layer 10: Draw last wave warning (Story 5.5) - DISABLED for production
	// 最后一波提示（如果需要显示）（开发调试用，已禁用）
	// s.drawLastWaveWarning(screen) // 已禁用：改为使用 FinalWave.reanim 动画

	// Layer 10.1: Draw huge wave warning (Story 17.7)
	// 红字警告 "A Huge Wave of Zombies is Approaching!"
	s.drawHugeWaveWarning(screen)

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

	// Story 8.8: Draw UI elements (ZombiesWon animation, dialogs, etc.)
	// 渲染所有标记为 UIComponent 的实体（在暂停菜单之前）
	s.renderSystem.DrawUIElements(screen)

	// Story 8.8: Draw dialog boxes (game over dialog, etc.)
	// 对话框在 UI 元素之后渲染，确保显示在最上层
	if s.dialogRenderSystem != nil {
		s.dialogRenderSystem.Draw(screen)
	}

	// Story 10.8: Draw Tooltip (植物卡片提示框)
	// Tooltip 在对话框之后、暂停菜单之前渲染
	s.drawTooltip(screen)

	// Story 10.1: Draw pause menu (最顶层 - 在所有其他元素之上)
	if s.pauseMenuModule != nil {
		s.pauseMenuModule.Draw(screen)
	}
}
