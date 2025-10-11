package scenes

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"github.com/decker502/pvz/pkg/game"
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
// It manages the game state, UI elements, and will eventually contain the ECS system.
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

	return scene
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
//   - Temporary keyboard input for testing sun system (Story 2.2)
//   - Future: ECS system updates, input handling, game state management
func (s *GameScene) Update(deltaTime float64) {
	// Handle intro animation
	if s.isIntroAnimPlaying {
		s.updateIntroAnimation(deltaTime)
	}

	// Temporary debug code: Test sun system with keyboard input (Story 2.2)
	// Press 'A' to add 25 sun, press 'S' to spend 50 sun
	// TODO: Remove or comment out after Story 2.4 implements real sun collection
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		s.gameState.AddSun(25)
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		s.gameState.SpendSun(50)
	}

	// Future: Update ECS systems, handle game logic, etc.
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
// It draws the lawn background and all UI elements in the correct order.
func (s *GameScene) Draw(screen *ebiten.Image) {
	// Draw lawn background
	s.drawBackground(screen)

	// Draw UI elements
	s.drawSeedBank(screen)
	s.drawSunCounter(screen)
	s.drawShovel(screen)
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
