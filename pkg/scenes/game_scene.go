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
	SeedBankX      = 0
	SeedBankY      = 0
	SeedBankWidth  = 500
	SeedBankHeight = 87

	// Sun Counter (阳光计数器) - relative to SeedBank position
	SunCounterOffsetX  = 45 // 相对于 SeedBank 的 X 偏移量
	SunCounterOffsetY  = 69 // 相对于 SeedBank 的 Y 偏移量
	SunCounterWidth    = 130
	SunCounterHeight   = 60
	SunCounterFontSize = 28.0 // 阳光数值字体大小（像素）

	// Plant Cards (植物卡片) - relative to SeedBank position
	PlantCardStartOffsetX = 92   // 第一张卡片相对于 SeedBank 的 X 偏移量
	PlantCardOffsetY      = 8    // 卡片相对于 SeedBank 的 Y 偏移量
	PlantCardSpacing      = 65   // 卡片槽之间的间距（包含卡槽边框，每个卡槽约76px宽）
	PlantCardScale        = 0.95 // 卡片缩放因子（原始卡片约64x89，缩放后约54x76适配卡槽）

	// Shovel (铲子) - positioned to the right of seed bank
	ShovelX      = 620 // To the right of seed bank (bar5.png width=612 + small gap)
	ShovelY      = 8
	ShovelWidth  = 70
	ShovelHeight = 74

	// Camera and Animation Constants
	// The background image is wider than the window, we show only a portion
	IntroAnimDuration = 3.0 // Duration of intro animation in seconds
	CameraScrollSpeed = 100 // Pixels per second for intro animation
	GameCameraX       = 215 // Final camera X position for gameplay (centered on lawn)
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

	// Camera and Animation
	cameraX            float64 // Camera X position (controls which part of background to show)
	maxCameraX         float64 // Maximum camera X position (rightmost edge of background)
	isIntroAnimPlaying bool    // Whether the intro animation is currently playing
	introAnimTimer     float64 // Timer for intro animation

	// ECS Framework and Systems
	entityManager       *ecs.EntityManager
	sunSpawnSystem      *systems.SunSpawnSystem
	sunMovementSystem   *systems.SunMovementSystem
	lifetimeSystem      *systems.LifetimeSystem
	renderSystem        *systems.RenderSystem
	inputSystem         *systems.InputSystem
	animationSystem     *systems.AnimationSystem
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

	// Initialize ECS framework
	scene.entityManager = ecs.NewEntityManager()

	// Initialize systems
	scene.renderSystem = systems.NewRenderSystem(scene.entityManager)
	scene.sunMovementSystem = systems.NewSunMovementSystem(scene.entityManager)
	scene.lifetimeSystem = systems.NewLifetimeSystem(scene.entityManager)
	scene.animationSystem = systems.NewAnimationSystem(scene.entityManager)

	// Calculate sun collection target position from sun counter UI position
	// This ensures the suns fly to the exact center of the sun counter display
	sunCollectionTargetX := float64(SeedBankX + SunCounterOffsetX)
	sunCollectionTargetY := float64(SeedBankY + SunCounterOffsetY)

	// Story 3.3: Initialize lawn grid system and entity
	scene.lawnGridSystem = systems.NewLawnGridSystem(scene.entityManager)
	scene.lawnGridEntityID = scene.entityManager.CreateEntity()
	scene.entityManager.AddComponent(scene.lawnGridEntityID, &components.LawnGridComponent{})
	log.Printf("[GameScene] Initialized lawn grid system (Entity ID: %d)", scene.lawnGridEntityID)

	// Initialize input system with sun counter target position and lawn grid system (Story 2.4 + Story 3.3)
	scene.inputSystem = systems.NewInputSystem(
		scene.entityManager,
		rm,
		scene.gameState,
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
	scene.plantPreviewSystem = systems.NewPlantPreviewSystem(scene.entityManager, scene.gameState)
	scene.plantPreviewRenderSystem = systems.NewPlantPreviewRenderSystem(scene.entityManager)

	// Story 3.4: Initialize behavior system (sunflower sun production, etc.)
	scene.behaviorSystem = systems.NewBehaviorSystem(scene.entityManager, rm)
	log.Printf("[GameScene] Initialized behavior system for plant behaviors")

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
	_, err := entities.NewPlantCardEntity(s.entityManager, rm, components.PlantSunflower, firstCardX, cardY)
	if err != nil {
		log.Printf("Warning: Failed to create sunflower card: %v", err)
		// 继续执行，游戏在没有卡片的情况下也能运行（用于测试环境）
	}

	// 豌豆射手卡片（第二张）
	secondCardX := firstCardX + PlantCardSpacing
	_, err = entities.NewPlantCardEntity(s.entityManager, rm, components.PlantPeashooter, secondCardX, cardY)
	if err != nil {
		log.Printf("Warning: Failed to create peashooter card: %v", err)
		// 继续执行，游戏在没有卡片的情况下也能运行（用于测试环境）
	}

	// Initialize PlantCardSystem
	s.plantCardSystem = systems.NewPlantCardSystem(
		s.entityManager,
		s.gameState,
		rm,
	)

	// Initialize PlantCardRenderSystem
	s.plantCardRenderSystem = systems.NewPlantCardRenderSystem(
		s.entityManager,
		PlantCardScale, // 使用常量定义的缩放因子
	)
}

// loadResources loads all UI images required for the game scene.
// If a resource fails to load, it logs a warning but continues.
// The Draw method will use fallback rendering for missing resources.
func (s *GameScene) loadResources() {
	// Load lawn background
	bg, err := s.resourceManager.LoadImage("assets/images/Background/background1.jpg")
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
	seedBank, err := s.resourceManager.LoadImage("assets/images/interface/bar5.png")
	if err != nil {
		log.Printf("Warning: Failed to load seed bank image: %v", err)
		log.Printf("Will use fallback rendering for seed bank")
	} else {
		s.seedBank = seedBank
	}

	// Load shovel slot background
	shovelSlot, err := s.resourceManager.LoadImage("assets/images/interface/shovelSlot.png")
	if err != nil {
		log.Printf("Warning: Failed to load shovel slot: %v", err)
	} else {
		s.shovelSlot = shovelSlot
	}

	// Load shovel icon
	shovel, err := s.resourceManager.LoadImage("assets/images/interface/shovel2.png")
	if err != nil {
		log.Printf("Warning: Failed to load shovel icon: %v", err)
	} else {
		s.shovel = shovel
	}

	// Load font for sun counter
	font, err := s.resourceManager.LoadFont("assets/fonts/briannetod.ttf", SunCounterFontSize)
	if err != nil {
		log.Printf("Warning: Failed to load sun counter font: %v", err)
		log.Printf("Will use fallback debug text rendering")
	} else {
		s.sunCounterFont = font
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

	// Update all ECS systems in order (order matters for correct game logic)
	s.plantCardSystem.Update(deltaTime)     // 1. Update plant card states (before input)
	s.inputSystem.Update(deltaTime)         // 2. Process player input (highest priority)
	s.sunSpawnSystem.Update(deltaTime)      // 3. Generate new suns
	s.sunMovementSystem.Update(deltaTime)   // 4. Move suns (includes collection animation)
	s.sunCollectionSystem.Update(deltaTime) // 5. Check if collection is complete
	s.behaviorSystem.Update(deltaTime)      // 6. Update plant behaviors (Story 3.4)
	s.animationSystem.Update(deltaTime)     // 7. Update animation frames
	s.plantPreviewSystem.Update(deltaTime)  // 8. Update plant preview position (Story 3.2)
	s.lifetimeSystem.Update(deltaTime)      // 9. Check for expired entities
	s.entityManager.RemoveMarkedEntities()  // 10. Clean up deleted entities (always last)
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
// Rendering order (back to front):
// 1. Background (lawn)
// 2. UI base layer (seed bank, shovel) - drawn first so suns appear on top
// 3. Game entities (suns, plants, zombies) - drawn on top of UI
// 4. UI overlay (sun counter text) - drawn last for best visibility
// 5. Plant cards (on top of everything)
func (s *GameScene) Draw(screen *ebiten.Image) {
	// Layer 1: Draw lawn background
	s.drawBackground(screen)

	// Layer 2: Draw UI base elements (seed bank and shovel)
	// These are drawn before entities so suns can fly over them
	s.drawSeedBank(screen)
	s.drawShovel(screen)

	// Layer 3: Draw all game entities (suns, plants, zombies, etc.)
	// This ensures suns appear on top of the seed bank
	s.renderSystem.Draw(screen)

	// Layer 4: Draw UI overlays (sun counter text)
	// Drawn last to ensure text is always visible
	s.drawSunCounter(screen)

	// Layer 5: Draw plant cards (Story 3.1)
	// Drawn on top of everything for best visibility
	s.plantCardRenderSystem.Draw(screen)

	// Layer 6: Draw plant preview (Story 3.2)
	// Drawn on top of everything including plant cards
	s.plantPreviewRenderSystem.Draw(screen)

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
