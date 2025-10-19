// Package main provides a reward animation verification tool for testing and debugging
// the reward animation system in the PvZ game.
//
// Usage:
//
//	go run cmd/verify_reward/main.go [flags]
//
// Flags:
//
//	--plant <id>         Plant ID to unlock (default: "sunflower")
//	--verbose            Enable verbose logging
//
// Controls:
//
//	Space/Click  - Click card pack to expand / Close reward panel
//	R            - Restart animation from beginning
//	Q            - Quit
//
// Purpose:
//   - Quickly test reward animation without completing full level
//   - Debug parabola trajectory and bounce animation
//   - Verify reward panel display and layout
//   - Adjust animation timing and parameters
package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"

	"github.com/decker502/pvz/pkg/components"
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
	plantFlag   = flag.String("plant", "sunflower", "Plant ID to unlock (e.g., sunflower, peashooter)")
	verboseFlag = flag.Bool("verbose", false, "Enable verbose logging")
)

// Global audio context (must be created only once)
var globalAudioContext *audio.Context

// RewardVerifyGame implements ebiten.Game interface for reward animation verification
type RewardVerifyGame struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager

	// Systems
	rewardSystem      *systems.RewardAnimationSystem
	rewardRenderSys   *systems.RewardPanelRenderSystem
	renderSystem      *systems.RenderSystem
	reanimSystem      *systems.ReanimSystem // 需要用于渲染 ReanimComponent

	// Configuration
	plantID string

	// Debug info
	debugInfo string
}

// NewRewardVerifyGame creates a new reward animation verification game
func NewRewardVerifyGame(plantID string) (*RewardVerifyGame, error) {
	// Initialize resource manager (reuse global audio context)
	if globalAudioContext == nil {
		globalAudioContext = audio.NewContext(48000)
	}
	rm := game.NewResourceManager(globalAudioContext)

	// Load resource configuration
	if err := rm.LoadResourceConfig("assets/config/resources.yaml"); err != nil {
		return nil, fmt.Errorf("failed to load resource config: %w", err)
	}

	// Load LoadingImages resource group (contains Reanim part images)
	log.Println("Loading LoadingImages resource group...")
	if err := rm.LoadResourceGroup("LoadingImages"); err != nil {
		log.Printf("Warning: Failed to load LoadingImages group: %v", err)
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

	// Load reward resources (optional, may not exist yet)
	_, _ = rm.LoadImageByID("IMAGE_AWARDSCREEN_BACK")
	_, _ = rm.LoadImageByID("IMAGE_SEEDPACKET_LARGER")

	// Load Reanim resources (required for rendering plant icons)
	log.Println("Loading Reanim resources...")
	if err := rm.LoadReanimResources(); err != nil {
		return nil, fmt.Errorf("failed to load reanim resources: %w", err)
	}

	// Initialize ECS
	em := ecs.NewEntityManager()
	gs := game.GetGameState()

	// Load LawnStrings for plant names
	lawnStrings, err := game.NewLawnStrings("assets/properties/LawnStrings.txt")
	if err != nil {
		log.Printf("Warning: Failed to load LawnStrings: %v", err)
	} else {
		gs.LawnStrings = lawnStrings
		log.Println("LawnStrings loaded successfully")
	}

	// Set game state
	gs.LevelTime = 0
	gs.CameraX = 0

	// Create systems
	reanimSystem := systems.NewReanimSystem(em)
	renderSystem := systems.NewRenderSystem(em)
	rewardSystem := systems.NewRewardAnimationSystem(em, gs, rm, reanimSystem)
	rewardRenderSys := systems.NewRewardPanelRenderSystem(em, gs, rm)

	game := &RewardVerifyGame{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
		rewardSystem:    rewardSystem,
		rewardRenderSys: rewardRenderSys,
		renderSystem:    renderSystem,
		reanimSystem:    reanimSystem,
		plantID:         plantID,
		debugInfo:       "Reward Animation Verifier - Press Space to start, R to restart, Q to quit",
	}

	// Auto-trigger reward animation
	log.Printf("Triggering reward animation for plant: %s", plantID)
	rewardSystem.TriggerReward(plantID)

	return game, nil
}

// Update updates the game logic
func (g *RewardVerifyGame) Update() error {
	// Handle quit
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		return fmt.Errorf("quit")
	}

	// Handle restart
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		log.Println("Restarting animation...")
		newGame, err := NewRewardVerifyGame(g.plantID)
		if err != nil {
			log.Printf("Failed to restart: %v", err)
		} else {
			*g = *newGame
			return nil
		}
	}

	// Update systems
	dt := 1.0 / 60.0

	// Update reward animation
	g.rewardSystem.Update(dt)

	// Update reanim animations (必须更新才能渲染 ReanimComponent)
	g.reanimSystem.Update(dt)

	// Update debug info
	phase := "Inactive"
	if g.rewardSystem.IsActive() {
		entity := g.rewardSystem.GetEntity()
		if entity != 0 {
			// Try to get RewardAnimationComponent for debug info
			// Note: We can't access private components directly, so we show general state
			phase = "Active"
		}
	}

	if g.rewardSystem.IsCompleted() {
		g.debugInfo = "Animation completed! Press R to restart, Q to quit"
	} else if g.rewardSystem.IsActive() {
		g.debugInfo = fmt.Sprintf("Phase: %s | Press Space to advance | R to restart | Q to quit", phase)
	} else {
		g.debugInfo = "Animation inactive. Press R to restart"
	}

	return nil
}

// Draw renders the game
func (g *RewardVerifyGame) Draw(screen *ebiten.Image) {
	// Clear screen with dark background (for better visibility)
	screen.Fill(color.RGBA{20, 40, 20, 255})

	// Draw background (dimmed to highlight reward)
	backgroundImg := g.resourceManager.GetImageByID("IMAGE_BACKGROUND1")
	if backgroundImg != nil {
		opts := &ebiten.DrawImageOptions{}
		// Scale background to fit screen if needed
		bgBounds := backgroundImg.Bounds()
		bgWidth := float64(bgBounds.Dx())
		bgHeight := float64(bgBounds.Dy())

		scaleX := float64(screenWidth) / bgWidth
		scaleY := float64(screenHeight) / bgHeight
		scale := scaleX
		if scaleY < scaleX {
			scale = scaleY
		}

		opts.GeoM.Scale(scale, scale)
		opts.ColorScale.ScaleAlpha(0.3) // Dim the background
		screen.DrawImage(backgroundImg, opts)
	}

	// Draw reward animation (card pack drop, bounce)
	// RenderSystem will draw entities with PositionComponent and SpriteComponent
	g.renderSystem.Draw(screen, 0) // No camera offset for reward

	// Draw debug info overlay (only showing phase and position, NOT replacing card pack)
	g.drawDebugInfo(screen)

	// Draw reward panel (if showing)
	g.rewardRenderSys.Draw(screen)

	// Draw debug info (only when reward panel is not showing to avoid text overlap)
	if !g.rewardSystem.IsActive() || g.rewardSystem.IsCompleted() {
		plantInfo := g.gameState.GetPlantUnlockManager().GetPlantInfo(g.plantID)
		plantName := plantInfo.NameKey
		if g.gameState.LawnStrings != nil {
			loadedName := g.gameState.LawnStrings.GetString(plantInfo.NameKey)
			if loadedName != "" {
				plantName = loadedName
			}
		}

		debugText := fmt.Sprintf(
			"Reward Animation Verifier\n"+
				"Plant: %s (%s)\n"+
				"%s\n\n"+
				"Animation Phases:\n"+
				"  1. Dropping - Card pack falls from right\n"+
				"  2. Bouncing - Bounce animation (3x)\n"+
				"  3. Expanding - Wait for click (Space)\n"+
				"  4. Showing - Display reward panel\n"+
				"  5. Closing - Fade out and finish\n\n"+
				"Current State:\n"+
				"  Active: %v\n"+
				"  Completed: %v",
			g.plantID,
			plantName,
			g.debugInfo,
			g.rewardSystem.IsActive(),
			g.rewardSystem.IsCompleted(),
		)
		ebitenutil.DebugPrint(screen, debugText)
	}
}

// drawDebugInfo draws debug information overlay (text only, no shapes)
func (g *RewardVerifyGame) drawDebugInfo(screen *ebiten.Image) {
	if !g.rewardSystem.IsActive() {
		return
	}

	entity := g.rewardSystem.GetEntity()
	if entity == 0 {
		return
	}

	// Get position component
	posComp, ok := ecs.GetComponent[*components.PositionComponent](g.entityManager, entity)
	if !ok {
		return
	}

	// Get reward animation component to show current phase
	rewardComp, ok := ecs.GetComponent[*components.RewardAnimationComponent](g.entityManager, entity)
	if !ok {
		return
	}

	// Draw a label showing position and phase (text only, positioned above the card pack)
	label := fmt.Sprintf("Phase: %s\nPos: (%.0f, %.0f)\nTime: %.1fs",
		rewardComp.Phase, posComp.X, posComp.Y, rewardComp.ElapsedTime)
	ebitenutil.DebugPrintAt(screen, label, int(posComp.X-40), int(posComp.Y-80))
}

// Layout returns the game's screen dimensions
func (g *RewardVerifyGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	flag.Parse()

	// Setup logging
	if !*verboseFlag {
		log.SetOutput(os.Stdout)
		log.SetFlags(log.Ltime)
	}

	log.Println("=== Reward Animation Verifier ===")
	log.Printf("Plant ID: %s", *plantFlag)

	// Validate plant ID
	validPlants := []string{"sunflower", "peashooter", "cherrybomb", "wallnut"}
	isValid := false
	for _, p := range validPlants {
		if *plantFlag == p {
			isValid = true
			break
		}
	}
	if !isValid {
		log.Printf("Warning: Plant ID '%s' may not be recognized. Valid plants: %v", *plantFlag, validPlants)
	}

	// Create game
	game, err := NewRewardVerifyGame(*plantFlag)
	if err != nil {
		log.Fatalf("Failed to create game: %v", err)
	}

	// Set window properties
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle(fmt.Sprintf("Reward Animation Verifier - %s", *plantFlag))
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// Run game
	if err := ebiten.RunGame(game); err != nil {
		if err.Error() != "quit" {
			log.Fatalf("Game error: %v", err)
		}
	}

	log.Println("Verifier closed")
}
