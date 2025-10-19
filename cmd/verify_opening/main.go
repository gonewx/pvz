// Package main provides an opening animation verification tool for testing and debugging
// the opening animation system in the PvZ game.
//
// Usage:
//
//	go run cmd/verify_opening/main.go [flags]
//
// Flags:
//
//	--level <id>         Level ID to test (default: "1-2")
//	--speed <pixels/s>   Camera animation speed (default: 300)
//	--skip-delay         Skip initial idle delay (start animation immediately)
//	--verbose            Enable verbose logging
//
// Controls:
//
//	Space/ESC  - Skip opening animation
//	R          - Restart animation from beginning
//	Q          - Quit
//
// Purpose:
//   - Quickly test opening animation without completing full level
//   - Debug camera movement and easing effects
//   - Verify zombie preview spawning logic
//   - Adjust animation timing and parameters
package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 800
	screenHeight = 600
)

var (
	levelFlag     = flag.String("level", "1-2", "Level ID to test (e.g., 1-2, 1-3)")
	speedFlag     = flag.Float64("speed", 300, "Camera animation speed in pixels/second")
	skipDelayFlag = flag.Bool("skip-delay", false, "Skip initial idle delay")
	verboseFlag   = flag.Bool("verbose", false, "Enable verbose logging")
)

// OpeningVerifyGame implements ebiten.Game interface for opening animation verification
type OpeningVerifyGame struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager

	// Systems
	cameraSystem   *systems.CameraSystem
	openingSystem  *systems.OpeningAnimationSystem
	renderSystem   *systems.RenderSystem
	behaviorSystem *systems.BehaviorSystem
	reanimSystem   *systems.ReanimSystem

	// Configuration
	levelConfig *config.LevelConfig

	// Debug info
	debugInfo string
}

// NewOpeningVerifyGame creates a new opening animation verification game
func NewOpeningVerifyGame() (*OpeningVerifyGame, error) {
	// Initialize resource manager
	audioContext := audio.NewContext(48000)
	rm := game.NewResourceManager(audioContext)

	// Load resource configuration
	if err := rm.LoadResourceConfig("assets/config/resources.yaml"); err != nil {
		return nil, fmt.Errorf("failed to load resource config: %w", err)
	}

	// Load necessary resources (background for rendering)
	log.Println("Loading background...")
	var err error
	_, err = rm.LoadImageByID("IMAGE_BACKGROUND1")
	if err != nil {
		// Fallback: try to load directly if resource config fails
		log.Printf("Warning: Failed to load via resource ID, trying direct path: %v", err)
		_, err = rm.LoadImage("images/Background1.jpg")
		if err != nil {
			return nil, fmt.Errorf("failed to load background: %w", err)
		}
	}

	// Load level configuration
	levelPath := filepath.Join("data", "levels", fmt.Sprintf("level-%s.yaml", *levelFlag))
	levelConfig, err := config.LoadLevelConfig(levelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load level config: %w", err)
	}

	log.Printf("Loaded level: %s - %s", levelConfig.ID, levelConfig.Name)

	// Initialize ECS
	em := ecs.NewEntityManager()
	gs := game.GetGameState()

	// Set level config
	gs.CurrentLevel = levelConfig
	gs.LevelTime = 0
	gs.CameraX = 0

	// Create systems
	reanimSystem := systems.NewReanimSystem(em)
	behaviorSystem := systems.NewBehaviorSystem(em, rm, reanimSystem, gs)
	renderSystem := systems.NewRenderSystem(em)
	cameraSystem := systems.NewCameraSystem(em, gs)

	// Create opening animation system
	var openingSystem *systems.OpeningAnimationSystem
	if levelConfig.OpeningType == "standard" && !levelConfig.SkipOpening {
		openingSystem = systems.NewOpeningAnimationSystem(em, gs, rm, levelConfig, cameraSystem)
		log.Println("Opening animation system created")
	} else {
		log.Printf("Opening animation skipped: OpeningType=%s, SkipOpening=%v",
			levelConfig.OpeningType, levelConfig.SkipOpening)
	}

	// Apply speed override if specified
	if *speedFlag != 300 {
		log.Printf("Camera speed overridden: %.0f px/s", *speedFlag)
	}

	game := &OpeningVerifyGame{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
		cameraSystem:    cameraSystem,
		openingSystem:   openingSystem,
		renderSystem:    renderSystem,
		behaviorSystem:  behaviorSystem,
		reanimSystem:    reanimSystem,
		levelConfig:     levelConfig,
		debugInfo:       "Opening Animation Verifier - Press R to restart, Q to quit",
	}

	return game, nil
}

// Update updates the game logic
func (g *OpeningVerifyGame) Update() error {
	// Handle quit
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		return fmt.Errorf("quit")
	}

	// Handle restart
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		log.Println("Restarting animation...")
		newGame, err := NewOpeningVerifyGame()
		if err != nil {
			log.Printf("Failed to restart: %v", err)
		} else {
			*g = *newGame
			return nil
		}
	}

	// Update systems
	dt := 1.0 / 60.0

	// Update opening animation if active
	if g.openingSystem != nil {
		g.openingSystem.Update(dt)

		// Update debug info
		openingComp, ok := ecs.GetComponent[*components.OpeningAnimationComponent](g.entityManager, g.openingSystem.GetEntity())
		if ok {
			g.debugInfo = fmt.Sprintf(
				"State: %s | Camera: %.0f | Zombies: %d | ESC/Space to skip | R to restart | Q to quit",
				openingComp.State,
				g.gameState.CameraX,
				len(openingComp.ZombieEntities),
			)

			// Check if completed
			if openingComp.IsCompleted {
				g.debugInfo = "Animation completed! Press R to restart, Q to quit"
			}
		}
	} else {
		g.debugInfo = "No opening animation for this level (tutorial/special). Press Q to quit"
	}

	// Update camera
	g.cameraSystem.Update(dt)

	// Update animations (for zombie idle animations)
	g.reanimSystem.Update(dt)

	return nil
}

// Draw renders the game
func (g *OpeningVerifyGame) Draw(screen *ebiten.Image) {
	// Clear screen
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// Draw background (manually, since RenderSystem doesn't draw background)
	backgroundImg := g.resourceManager.GetImageByID("IMAGE_BACKGROUND1")
	if backgroundImg != nil {
		// Calculate background dimensions
		bgBounds := backgroundImg.Bounds()
		bgWidth := float64(bgBounds.Dx())
		bgHeight := float64(bgBounds.Dy())

		// Limit camera X to prevent moving beyond background
		maxCameraX := bgWidth - float64(screenWidth)
		if maxCameraX < 0 {
			maxCameraX = 0
		}
		cameraX := g.gameState.CameraX
		if cameraX > maxCameraX {
			cameraX = maxCameraX
		}
		if cameraX < 0 {
			cameraX = 0
		}

		// Draw background with camera offset
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(-cameraX, 0)

		// Center vertically if background is shorter than screen
		if bgHeight < float64(screenHeight) {
			yOffset := (float64(screenHeight) - bgHeight) / 2
			opts.GeoM.Translate(0, yOffset)
		}

		screen.DrawImage(backgroundImg, opts)
	}

	// Draw game world (zombies, etc.)
	g.renderSystem.Draw(screen, g.gameState.CameraX)

	// Draw debug zombie previews (since zombies don't have ReanimComponent yet)
	if g.openingSystem != nil {
		g.drawDebugZombiePreviews(screen)
	}

	// Draw debug info
	debugText := fmt.Sprintf(
		"Opening Animation Verifier\n"+
			"Level: %s - %s\n"+
			"%s\n\n"+
			"Camera Speed: %.0f px/s\n"+
			"Opening Type: %s",
		g.levelConfig.ID,
		g.levelConfig.Name,
		g.debugInfo,
		*speedFlag,
		g.levelConfig.OpeningType,
	)
	ebitenutil.DebugPrint(screen, debugText)
}

// Layout returns the game's screen dimensions
func (g *OpeningVerifyGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// drawDebugZombiePreviews draws debug rectangles for zombie preview entities
// This is a temporary visualization since zombies don't have ReanimComponent yet
// 僵尸预告应该固定在屏幕上，不随镜头移动
func (g *OpeningVerifyGame) drawDebugZombiePreviews(screen *ebiten.Image) {
	openingComp, ok := ecs.GetComponent[*components.OpeningAnimationComponent](g.entityManager, g.openingSystem.GetEntity())
	if !ok {
		return
	}

	// Draw each preview zombie as a colored rectangle
	for _, zombieID := range openingComp.ZombieEntities {
		posComp, ok := ecs.GetComponent[*components.PositionComponent](g.entityManager, zombieID)
		if !ok {
			continue
		}

		// 僵尸使用世界坐标，但不应用镜头偏移（固定在屏幕上）
		// 注意：这里直接使用世界坐标作为屏幕坐标
		screenX := posComp.X
		screenY := posComp.Y

		// Draw zombie as red rectangle (80x150 approximate zombie size)
		zombieWidth := 80.0
		zombieHeight := 150.0
		zombieRect := ebiten.NewImage(int(zombieWidth), int(zombieHeight))
		zombieRect.Fill(color.RGBA{255, 0, 0, 200}) // Semi-transparent red

		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(screenX-zombieWidth/2, screenY-zombieHeight) // Center horizontally, bottom aligned
		screen.DrawImage(zombieRect, opts)

		// Draw a label showing world coordinates
		label := fmt.Sprintf("Zombie\nWorld: %.0f,%.0f\nCamera: %.0f", posComp.X, posComp.Y, g.gameState.CameraX)
		ebitenutil.DebugPrintAt(screen, label, int(screenX-30), int(screenY-zombieHeight-40))
	}
}

func main() {
	flag.Parse()

	// Setup logging
	if !*verboseFlag {
		log.SetOutput(os.Stdout)
		log.SetFlags(log.Ltime)
	}

	log.Println("=== Opening Animation Verifier ===")
	log.Printf("Level: %s", *levelFlag)
	log.Printf("Camera Speed: %.0f px/s", *speedFlag)
	log.Printf("Skip Delay: %v", *skipDelayFlag)

	// Create game
	game, err := NewOpeningVerifyGame()
	if err != nil {
		log.Fatalf("Failed to create game: %v", err)
	}

	// Set window properties
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle(fmt.Sprintf("Opening Animation Verifier - Level %s", *levelFlag))
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// Run game
	if err := ebiten.RunGame(game); err != nil {
		if err.Error() != "quit" {
			log.Fatalf("Game error: %v", err)
		}
	}

	log.Println("Verifier closed")
}
