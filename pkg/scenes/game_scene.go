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
	SunCounterOffsetY  = 63 // 相对于 SeedBank 的 Y 偏移量
	SunCounterWidth    = 130
	SunCounterHeight   = 60
	SunCounterFontSize = 18.0 // 阳光数值字体大小（像素）

	// Plant Cards (植物卡片) - relative to SeedBank position
	PlantCardStartOffsetX = 84   // 第一张卡片相对于 SeedBank 的 X 偏移量
	PlantCardOffsetY      = 8    // 卡片相对于 SeedBank 的 Y 偏移量
	PlantCardSpacing      = 60   // 卡片槽之间的间距（包含卡槽边框，每个卡槽约76px宽）
	PlantCardScale        = 0.50 // 卡片背景缩放因子（实际图片100x140，缩放后54x76适配卡槽）
	PlantCardFontSize     = 15.0 // 卡片阳光数字字体大小（像素）

	// Plant Card Icon (卡片上的植物图标) - Story 6.3 可配置参数
	PlantIconScale   = 0.55 // 植物图标缩放因子（原始80x90，可调整此值来改变图标大小）
	PlantIconOffsetY = 5.0  // 植物图标距离卡片顶部的偏移（像素，可调整垂直位置）
	SunTextOffsetY   = 16.0 // 阳光数字距离卡片底部的偏移（像素，可调整文本位置）

	// Shovel (铲子) - positioned to the right of seed bank
	ShovelX      = 620 // To the right of seed bank (bar5.png width=612 + small gap)
	ShovelY      = 8
	ShovelWidth  = 70
	ShovelHeight = 74

	// Camera and Animation Constants
	// The background image is wider than the window, we show only a portion
	IntroAnimDuration = 3.0 // Duration of intro animation in seconds
	CameraScrollSpeed = 100 // Pixels per second for intro animation
	GameCameraX       = 220 // Final camera X position for gameplay (centered on lawn)
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

	// Story 7.2: Particle System
	particleSystem *systems.ParticleSystem // 粒子系统（粒子特效）
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
		cameraX:            GameCameraX, // 直接设置为游戏镜头位置，跳过开场动画
		isIntroAnimPlaying: false,       // 禁用开场动画
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
		250.0, // minX - 草坪左边界
		900.0, // maxX - 草坪右边界
		100.0, // minTargetY - 草坪上边界
		550.0, // maxTargetY - 草坪下边界
	)

	// Story 3.1: Initialize plant card systems
	scene.initPlantCardSystems(rm)

	// Story 3.2: Initialize plant preview systems
	// PlantPreviewRenderSystem 需要引用 PlantPreviewSystem 来获取两个渲染位置
	scene.plantPreviewSystem = systems.NewPlantPreviewSystem(scene.entityManager, scene.gameState)
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

	// 2. Create LevelSystem
	scene.levelSystem = systems.NewLevelSystem(scene.entityManager, scene.gameState, scene.waveSpawnSystem)
	log.Printf("[GameScene] Initialized level system")

	// 3. Load level configuration
	levelConfig, err := config.LoadLevelConfig("data/levels/level-1-1.yaml")
	if err != nil {
		log.Printf("[GameScene] FATAL: Failed to load level config: %v", err)
		log.Printf("[GameScene] Game cannot start without level configuration")
	} else {
		scene.gameState.LoadLevel(levelConfig)
		log.Printf("[GameScene] Loaded level: %s (%d waves)", levelConfig.Name, len(levelConfig.Waves))
	}

	// Story 7.2: Initialize particle system
	// Story 7.4: Added ResourceManager parameter for loading particle images
	scene.particleSystem = systems.NewParticleSystem(scene.entityManager, scene.resourceManager)
	log.Printf("[GameScene] Initialized particle system for visual effects")

	return scene
}

// initPlantCardSystems initializes the plant card systems and creates plant card entities.
// Story 3.1: Plant Card UI and State
func (s *GameScene) initPlantCardSystems(rm *game.ResourceManager) {
	// Create plant card entities
	// 使用相对定位（相对于 SeedBank），与阳光计数器定位方式一致
	// 这提高了代码可维护性，当 SeedBank 位置改变时，卡片会自动跟随

	// 计算第一张卡片的绝对位置
	firstCardX := float64(SeedBankX + PlantCardStartOffsetX)
	cardY := float64(SeedBankY + PlantCardOffsetY)

	// 向日葵卡片（第一张）
	_, err := entities.NewPlantCardEntity(s.entityManager, rm, s.reanimSystem, components.PlantSunflower, firstCardX, cardY, PlantCardScale)
	if err != nil {
		log.Printf("Warning: Failed to create sunflower card: %v", err)
		// 继续执行，游戏在没有卡片的情况下也能运行（用于测试环境）
	}

	// 豌豆射手卡片（第二张）
	secondCardX := firstCardX + PlantCardSpacing
	_, err = entities.NewPlantCardEntity(s.entityManager, rm, s.reanimSystem, components.PlantPeashooter, secondCardX, cardY, PlantCardScale)
	if err != nil {
		log.Printf("Warning: Failed to create peashooter card: %v", err)
		// 继续执行，游戏在没有卡片的情况下也能运行（用于测试环境）
	}

	// 坚果墙卡片（第三张）
	thirdCardX := secondCardX + PlantCardSpacing
	_, err = entities.NewPlantCardEntity(s.entityManager, rm, s.reanimSystem, components.PlantWallnut, thirdCardX, cardY, PlantCardScale)
	if err != nil {
		log.Printf("Warning: Failed to create wallnut card: %v", err)
		// 继续执行，游戏在没有卡片的情况下也能运行（用于测试环境）
	}

	// 樱桃炸弹卡片（第四张）Story 5.4
	fourthCardX := thirdCardX + PlantCardSpacing
	_, err = entities.NewPlantCardEntity(s.entityManager, rm, s.reanimSystem, components.PlantCherryBomb, fourthCardX, cardY, PlantCardScale)
	if err != nil {
		log.Printf("Warning: Failed to create cherry bomb card: %v", err)
		// 继续执行，游戏在没有卡片的情况下也能运行（用于测试环境）
	}

	// Initialize PlantCardSystem
	s.plantCardSystem = systems.NewPlantCardSystem(
		s.entityManager,
		s.gameState,
		rm,
	)

	// Initialize PlantCardRenderSystem (Story 6.3: 可配置的多层渲染)
	s.plantCardRenderSystem = systems.NewPlantCardRenderSystem(
		s.entityManager,
		PlantCardScale,   // 卡片背景缩放因子 (0.50)
		PlantIconScale,   // 植物图标缩放因子 (0.55, 可调整)
		PlantIconOffsetY, // 植物图标距离顶部的偏移 (5.0 像素, 可调整)
		SunTextOffsetY,   // 阳光数字距离底部的偏移 (18.0 像素, 可调整)
		s.plantCardFont,  // 阳光数字字体（黑色渲染）
	)
}

// loadResources loads all UI images required for the game scene.
// If a resource fails to load, it logs a warning but continues.
// The Draw method will use fallback rendering for missing resources.
func (s *GameScene) loadResources() {
	// Load lawn background
	bg, err := s.resourceManager.LoadImageByID("IMAGE_BACKGROUND1")
	if err != nil {
		log.Printf("Warning: Failed to load lawn background: %v", err)
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

	// Load font for sun counter
	font, err := s.resourceManager.LoadFont("assets/fonts/SimHei.ttf", SunCounterFontSize)
	if err != nil {
		log.Printf("Warning: Failed to load sun counter font: %v", err)
		log.Printf("Will use fallback debug text rendering")
	} else {
		s.sunCounterFont = font
	}

	// Load font for plant card sun cost
	cardFont, err := s.resourceManager.LoadFont("assets/fonts/SimHei.ttf", PlantCardFontSize)
	if err != nil {
		log.Printf("Warning: Failed to load plant card font: %v", err)
		log.Printf("Will use fallback debug text rendering for card cost")
	} else {
		s.plantCardFont = cardFont
	}

	// Note: Sun counter background is drawn procedurally for now
	// A dedicated image can be loaded here in the future if needed
}

// Update updates the game scene logic.
// deltaTime is the time elapsed since the last update in seconds.
//
// This method handles:
//   - Intro animation (camera scrolling left → right → center)
//   - ECS system updates (input, sun spawning, movement, collection, lifetime management)
//   - System execution order ensures correct game logic flow
func (s *GameScene) Update(deltaTime float64) {
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
	// If game is over, stop updating game systems but allow rendering
	if s.gameState.IsGameOver {
		// 游戏结束时不更新游戏系统，只保留渲染
		// 这样玩家可以看到最终的游戏状态（僵尸位置、植物状态等）
		return
	}

	// Update all ECS systems in order (order matters for correct game logic)
	s.levelSystem.Update(deltaTime)            // 0. Update level system (Story 5.5: wave spawning, victory/defeat)
	s.plantCardSystem.Update(deltaTime)        // 1. Update plant card states (before input)
	s.inputSystem.Update(deltaTime, s.cameraX) // 2. Process player input (highest priority, 传递摄像机位置)
	s.sunSpawnSystem.Update(deltaTime)         // 3. Generate new suns
	s.sunMovementSystem.Update(deltaTime)      // 4. Move suns (includes collection animation)
	s.sunCollectionSystem.Update(deltaTime)    // 5. Check if collection is complete
	s.behaviorSystem.Update(deltaTime)         // 6. Update plant behaviors (Story 3.4)
	s.physicsSystem.Update(deltaTime)          // 7. Check collisions (Story 4.3)
	// Story 6.3: Reanim 动画系统（替代旧的 AnimationSystem）
	s.reanimSystem.Update(deltaTime)   // 8. Update Reanim animation frames
	s.particleSystem.Update(deltaTime) // 9. Update particle effects (Story 7.2)
	// Story 3.2: 植物预览系统 - 更新预览位置（双图像支持）
	s.plantPreviewSystem.Update(deltaTime) // 10. Update plant preview position (dual-image support)
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
		s.cameraX = GameCameraX
		s.isIntroAnimPlaying = false
		return
	}

	if progress < 0.5 {
		// Phase 1: Scroll from left (0) to right (maxCameraX)
		phaseProgress := progress / 0.5
		easedProgress := s.easeOutQuad(phaseProgress)
		s.cameraX = easedProgress * s.maxCameraX
	} else {
		// Phase 2: Scroll from right (maxCameraX) back to center (GameCameraX)
		phaseProgress := (progress - 0.5) / 0.5
		easedProgress := s.easeOutQuad(phaseProgress)
		s.cameraX = s.maxCameraX + easedProgress*(GameCameraX-s.maxCameraX)
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

	// Layer 9: Draw level progress UI (Story 5.5)
	// 进度条显示当前波次进度
	s.drawLevelProgress(screen)

	// Layer 10: Draw last wave warning (Story 5.5)
	// 最后一波提示（如果需要显示）
	s.drawLastWaveWarning(screen)

	// Layer 11: Draw game result overlay (Story 5.5)
	// 胜利/失败界面（如果游戏结束）
	s.drawGameResultOverlay(screen)

	// DEBUG: Draw particle test instructions (Story 7.4 debugging)
	s.drawParticleTestInstructions(screen)

	// DEBUG: Draw grid boundaries (Story 3.3 debugging)
	s.drawGridDebug(screen)
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
	// 只在种植模式下显示网格
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

	// 获取最后一波的时间
	lastWaveTime := s.gameState.CurrentLevel.Waves[totalWaves-1].Time
	warningTime := lastWaveTime - systems.LastWaveWarningTime

	// 检查是否在提示时间窗口内（提示显示5秒）
	if s.gameState.LevelTime >= warningTime &&
		s.gameState.LevelTime < lastWaveTime &&
		!s.gameState.IsWaveSpawned(totalWaves-1) {

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
func (s *GameScene) drawGameResultOverlay(screen *ebiten.Image) {
	// 只在游戏结束时显示
	if !s.gameState.IsGameOver {
		return
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
