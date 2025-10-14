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

	// Story 4.1: Test Zombie Spawn (临时测试代码)
	// 在游戏开始后3秒，在第3行生成一个测试僵尸
	testZombieTimer   float64 // 测试僵尸生成计时器
	testZombieSpawned bool    // 是否已生成测试僵尸

	// Story 4.2: Test Peashooter Behavior (临时测试代码)
	testPeashooterTimer   float64 // 测试豌豆射手种植计时器
	testPeashooterSpawned bool    // 是否已种植测试豌豆射手
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
		// Initialize camera at the leftmost position for intro animation
		cameraX:            0,
		isIntroAnimPlaying: true,
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

	// Story 3.3: Initialize lawn grid system and entity
	scene.lawnGridSystem = systems.NewLawnGridSystem(scene.entityManager)
	scene.lawnGridEntityID = scene.entityManager.CreateEntity()
	scene.entityManager.AddComponent(scene.lawnGridEntityID, &components.LawnGridComponent{})
	log.Printf("[GameScene] Initialized lawn grid system (Entity ID: %d)", scene.lawnGridEntityID)

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
	scene.behaviorSystem = systems.NewBehaviorSystem(scene.entityManager, rm, scene.reanimSystem)
	log.Printf("[GameScene] Initialized behavior system for plant behaviors")

	// Story 4.3: Initialize physics system (collision detection)
	scene.physicsSystem = systems.NewPhysicsSystem(scene.entityManager, rm)
	log.Printf("[GameScene] Initialized physics system for collision detection")

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
	seedBank, err := s.resourceManager.LoadImageByID("IMAGE_REANIM_SEEDBANK")
	if err != nil {
		log.Printf("Warning: Failed to load seed bank image: %v", err)
		log.Printf("Will use fallback rendering for seed bank")
	} else {
		s.seedBank = seedBank
	}

	// Load shovel slot background
	shovelSlot, err := s.resourceManager.LoadImageByID("IMAGE_REANIM_SHOVELBANK")
	if err != nil {
		log.Printf("Warning: Failed to load shovel slot: %v", err)
	} else {
		s.shovelSlot = shovelSlot
	}

	// Load shovel icon
	shovel, err := s.resourceManager.LoadImageByID("IMAGE_REANIM_SHOVEL")
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

	// Story 4.1 & 5.3: Test zombie spawn (临时测试代码)
	// 在游戏开始后4秒，生成多种类型的测试僵尸
	if !s.testZombieSpawned {
		s.testZombieTimer += deltaTime
		if s.testZombieTimer >= 4.0 { // 改为4秒，让豌豆射手先种植
			// 计算生成位置：屏幕右侧外约50像素
			spawnX := s.cameraX + WindowWidth + 50.0 // 屏幕右侧外50像素

			// Story 5.3: 生成三种不同类型的僵尸进行测试
			// Story 6.3: 传递 reanimSystem 给僵尸工厂函数
			// 第1行: 普通僵尸
			log.Printf("[GameScene] 生成普通僵尸：行=0")
			zombieID1, err := entities.NewZombieEntity(s.entityManager, s.resourceManager, s.reanimSystem, 0, spawnX)
			if err != nil {
				log.Printf("[GameScene] 生成普通僵尸失败: %v", err)
			} else {
				log.Printf("[GameScene] 成功生成普通僵尸 (ID: %d)", zombieID1)
			}

			// 第2行: 路障僵尸 (Conehead)
			log.Printf("[GameScene] 生成路障僵尸：行=1")
			zombieID2, err := entities.NewConeheadZombieEntity(s.entityManager, s.resourceManager, s.reanimSystem, 1, spawnX)
			if err != nil {
				log.Printf("[GameScene] 生成路障僵尸失败: %v", err)
			} else {
				log.Printf("[GameScene] 成功生成路障僵尸 (ID: %d, 护甲370+生命270=640总HP)", zombieID2)
			}

			// 第3行: 铁桶僵尸 (Buckethead)
			log.Printf("[GameScene] 生成铁桶僵尸：行=2")
			zombieID3, err := entities.NewBucketheadZombieEntity(s.entityManager, s.resourceManager, s.reanimSystem, 2, spawnX)
			if err != nil {
				log.Printf("[GameScene] 生成铁桶僵尸失败: %v", err)
			} else {
				log.Printf("[GameScene] 成功生成铁桶僵尸 (ID: %d, 护甲1100+生命270=1370总HP)", zombieID3)
			}

			s.testZombieSpawned = true
		}
	}

	// Story 4.2: Test peashooter behavior (临时测试代码)
	// 在游戏开始后3秒，在第3行种植一个豌豆射手
	if !s.testPeashooterSpawned {
		s.testPeashooterTimer += deltaTime
		if s.testPeashooterTimer >= 3.0 {
			// 在第3行（row=2）第3列（col=2）种植豌豆射手
			col := 2
			row := 2

			log.Printf("[GameScene] 种植测试豌豆射手：col=%d, row=%d", col, row)
			// Story 6.3: 传递 reanimSystem 给工厂函数
			peashooterID, err := entities.NewPlantEntity(
				s.entityManager,
				s.resourceManager,
				s.gameState,
				s.reanimSystem,
				components.PlantPeashooter,
				col,
				row,
			)
			if err != nil {
				log.Printf("[GameScene] 种植豌豆射手失败: %v", err)
			} else {
				log.Printf("[GameScene] 成功种植测试豌豆射手 (ID: %d)", peashooterID)
			}
			s.testPeashooterSpawned = true
		}
	}

	// Update all ECS systems in order (order matters for correct game logic)
	s.plantCardSystem.Update(deltaTime)        // 1. Update plant card states (before input)
	s.inputSystem.Update(deltaTime, s.cameraX) // 2. Process player input (highest priority, 传递摄像机位置)
	s.sunSpawnSystem.Update(deltaTime)         // 3. Generate new suns
	s.sunMovementSystem.Update(deltaTime)      // 4. Move suns (includes collection animation)
	s.sunCollectionSystem.Update(deltaTime)    // 5. Check if collection is complete
	s.behaviorSystem.Update(deltaTime)         // 6. Update plant behaviors (Story 3.4)
	s.physicsSystem.Update(deltaTime)          // 7. Check collisions (Story 4.3)
	// Story 6.3: Reanim 动画系统（替代旧的 AnimationSystem）
	s.reanimSystem.Update(deltaTime) // 8. Update Reanim animation frames
	// Story 3.2: 植物预览系统 - 更新预览位置（双图像支持）
	s.plantPreviewSystem.Update(deltaTime) // 9. Update plant preview position (dual-image support)
	s.lifetimeSystem.Update(deltaTime)     // 10. Check for expired entities
	s.entityManager.RemoveMarkedEntities() // 11. Clean up deleted entities (always last)
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

	// Layer 6: Draw plant preview (Story 3.2)
	// 拖拽预览在所有内容上方
	s.plantPreviewRenderSystem.Draw(screen, s.cameraX)

	// Layer 7: Draw suns (阳光) - 最顶层
	// 阳光在最顶层以确保始终可点击
	s.renderSystem.DrawSuns(screen, s.cameraX)

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
