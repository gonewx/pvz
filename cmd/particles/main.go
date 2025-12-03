// Package main provides a particle effect viewer tool for testing and debugging
// particle systems in the PvZ game.
//
// Usage:
//
//	go run cmd/particles/main.go [flags]
//
// Flags:
//
//	--filter <keyword>    Initial filter by name (e.g., --filter=Pea)
//	--effect <name>       Start with specific effect (e.g., --effect=PeaSplat)
//	--auto-play           Automatically cycle through effects every 3 seconds
//
// Controls:
//
//	Mouse Click       - Spawn particle effect at cursor position
//	Left/Right Arrow  - Switch to previous/next effect
//	Page Up/Down      - Jump 10 effects forward/backward
//	Home/End          - Jump to first/last effect
//	0-9               - Quick jump to effect by index (0=10th, 1=1st, etc.)
//	Space             - Spawn effect at screen center
//	C                 - Spawn combo effect (ZombieHead + Arms)
//	P                 - Toggle pause (停止切换，观看完整动画)
//	F or /            - Enter search mode
//	R                 - Clear all active particles
//	[ / ]             - Decrease/increase angle offset by 15° (Story 7.6: 测试粒子方向)
//	\                 - Reset angle offset to 0°
//	Q/Escape          - Quit
//
// Search Mode (press F or /):
//
//	Type letters      - Filter effects by name
//	Backspace         - Delete last character
//	Enter/Escape      - Exit search mode
package main

import (
	"flag"
	"fmt"
	"image/color"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 1024
	screenHeight = 768
)

var (
	filterFlag   = flag.String("filter", "", "Initial filter by name keyword")
	effectFlag   = flag.String("effect", "", "Start with specific effect name")
	autoPlayFlag = flag.Bool("auto-play", false, "Auto cycle through effects every 3 seconds")
	verboseFlag  = flag.Bool("verbose", false, "Enable verbose logging (default off)")
)

// ParticleViewerGame implements ebiten.Game interface for the particle viewer
type ParticleViewerGame struct {
	entityManager   *ecs.EntityManager
	particleSystem  *systems.ParticleSystem
	renderSystem    *systems.RenderSystem
	resourceManager *game.ResourceManager

	// Particle effect lists
	allEffectNames      []string // All available effect names
	filteredEffectNames []string // Currently filtered list
	currentIndex        int      // Current effect index in filtered list

	// Search mode
	searchMode  bool   // Whether in search mode
	searchQuery string // Current search query

	// Auto-play mode
	autoPlay      bool
	lastSpawnTime time.Time

	// Pause mode (for watching complete animation)
	paused bool

	// Angle offset (Story 7.6: 用于测试粒子方向)
	angleOffset float64

	// UI state
	statusMessage string
}

// NewParticleViewerGame creates a new particle viewer game instance
func NewParticleViewerGame() (*ParticleViewerGame, error) {
	// Initialize resource manager
	audioContext := audio.NewContext(48000)
	rm := game.NewResourceManager(audioContext)
	if err := rm.LoadResourceConfig("assets/config/resources.yaml"); err != nil {
		return nil, fmt.Errorf("failed to load resource config: %w", err)
	}

	// Load only init group (minimal resources)
	if err := rm.LoadResourceGroup("Init"); err != nil {
		log.Printf("Warning: Failed to load Init group: %v (continuing anyway)", err)
	}

	// Load LoadingImages group for particle textures
	if err := rm.LoadResourceGroup("LoadingImages"); err != nil {
		log.Printf("Warning: Failed to load LoadingImages group: %v", err)
	}

	// Note: Particle images are now preloaded from loadingimages group

	// Initialize ECS
	em := ecs.NewEntityManager()

	// Initialize systems
	ps := systems.NewParticleSystem(em, rm)
	rs := systems.NewRenderSystem(em)

	// Load all particle effect names
	allNames, err := loadAllParticleEffectNames()
	if err != nil {
		return nil, fmt.Errorf("failed to load particle effects: %w", err)
	}

	if len(allNames) == 0 {
		return nil, fmt.Errorf("no particle effects found")
	}

	// Apply initial filter if specified
	initialQuery := *filterFlag
	filteredNames := filterEffects(allNames, initialQuery)

	if len(filteredNames) == 0 {
		log.Printf("Warning: No effects match initial filter %q, showing all", initialQuery)
		filteredNames = allNames
		initialQuery = ""
	}

	// Find starting index
	startIndex := 0
	if *effectFlag != "" {
		for i, name := range filteredNames {
			if name == *effectFlag {
				startIndex = i
				break
			}
		}
	}

	game := &ParticleViewerGame{
		entityManager:       em,
		particleSystem:      ps,
		renderSystem:        rs,
		resourceManager:     rm,
		allEffectNames:      allNames,
		filteredEffectNames: filteredNames,
		currentIndex:        startIndex,
		searchMode:          false,
		searchQuery:         initialQuery,
		autoPlay:            *autoPlayFlag,
		lastSpawnTime:       time.Now(),
		angleOffset:         0.0, // Story 7.6: 默认无偏移
	}

	game.updateStatusMessage()
	log.Printf("Particle Viewer initialized: %d total effects, %d after filter", len(allNames), len(filteredNames))
	if startIndex < len(filteredNames) {
		log.Printf("Starting with: %s", filteredNames[startIndex])
	}

	// 启动时自动在屏幕中心生成当前选择的粒子效果，避免空白屏幕
	game.spawnCurrentEffect(screenWidth/2, screenHeight/2)

	return game, nil
}

// loadAllParticleEffectNames scans the particles directory and returns all effect names
func loadAllParticleEffectNames() ([]string, error) {
	particlesDir := "data/particles"

	// Scan directory for XML files
	files, err := filepath.Glob(filepath.Join(particlesDir, "*.xml"))
	if err != nil {
		return nil, fmt.Errorf("failed to scan particles directory: %w", err)
	}

	names := make([]string, 0, len(files))

	for _, filePath := range files {
		// Extract filename without extension
		baseName := filepath.Base(filePath)
		effectName := strings.TrimSuffix(baseName, filepath.Ext(baseName))
		names = append(names, effectName)
	}

	// Sort alphabetically
	sort.Strings(names)

	return names, nil
}

// filterEffects returns effects matching the query (case-insensitive substring match)
func filterEffects(allNames []string, query string) []string {
	if query == "" {
		return allNames
	}

	queryLower := strings.ToLower(query)
	filtered := make([]string, 0)

	for _, name := range allNames {
		if strings.Contains(strings.ToLower(name), queryLower) {
			filtered = append(filtered, name)
		}
	}

	return filtered
}

// Update updates the game state
func (g *ParticleViewerGame) Update() error {
	dt := 1.0 / 60.0 // 60 FPS

	// Handle search mode input
	if g.searchMode {
		return g.updateSearchMode()
	}

	// Normal mode input
	return g.updateNormalMode(dt)
}

// updateSearchMode handles input when in search mode
func (g *ParticleViewerGame) updateSearchMode() error {
	// Exit search mode
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.searchMode = false
		g.statusMessage = fmt.Sprintf("Search: %q (%d results)", g.searchQuery, len(g.filteredEffectNames))
		log.Printf("Exited search mode. Query: %q, Results: %d", g.searchQuery, len(g.filteredEffectNames))
		return nil
	}

	// Backspace to delete character
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if len(g.searchQuery) > 0 {
			g.searchQuery = g.searchQuery[:len(g.searchQuery)-1]
			g.applySearch()
		}
		return nil
	}

	// Capture text input
	runes := ebiten.AppendInputChars(nil)
	if len(runes) > 0 {
		for _, r := range runes {
			// Only accept alphanumeric and some special characters
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
				g.searchQuery += string(r)
			}
		}
		g.applySearch()
	}

	return nil
}

// applySearch filters the effect list and resets index
func (g *ParticleViewerGame) applySearch() {
	g.filteredEffectNames = filterEffects(g.allEffectNames, g.searchQuery)

	// Reset to first item
	if len(g.filteredEffectNames) > 0 {
		g.currentIndex = 0
	}

	log.Printf("Search query: %q, Results: %d", g.searchQuery, len(g.filteredEffectNames))
}

// updateNormalMode handles input when in normal mode
func (g *ParticleViewerGame) updateNormalMode(dt float64) error {
	// Quit
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return fmt.Errorf("quit requested")
	}

	// Enter search mode
	if inpututil.IsKeyJustPressed(ebiten.KeyF) || inpututil.IsKeyJustPressed(ebiten.KeySlash) {
		g.searchMode = true
		g.statusMessage = "Search mode: Type to filter effects..."
		log.Println("Entered search mode")
		return nil
	}

	// Toggle pause (停止切换，专注观看完整动画)
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.paused = !g.paused
		if g.paused {
			g.statusMessage = "⏸ PAUSED - Press P to resume, Space to spawn more"
			log.Println("Paused - watching complete animation")
		} else {
			g.statusMessage = "▶ Resumed"
			log.Println("Resumed")
		}
		return nil
	}

	// Navigation (只在非暂停状态下允许切换)
	if !g.paused {
		// Quick jump with number keys (0-9)
		for i := 0; i <= 9; i++ {
			key := ebiten.Key(int(ebiten.Key0) + i)
			if inpututil.IsKeyJustPressed(key) {
				targetIndex := i
				if i == 0 {
					targetIndex = 10 // 0 key jumps to 10th effect
				}
				targetIndex-- // Convert to 0-based index

				if targetIndex >= 0 && targetIndex < len(g.filteredEffectNames) {
					g.currentIndex = targetIndex
					g.updateStatusMessage()
					g.spawnCurrentEffect(screenWidth/2, screenHeight/2)
				}
				return nil
			}
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
			g.previousEffect()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
			g.nextEffect()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
			g.jumpEffects(-10)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
			g.jumpEffects(10)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyHome) {
			g.currentIndex = 0
			g.updateStatusMessage()
			g.spawnCurrentEffect(screenWidth/2, screenHeight/2)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
			if len(g.filteredEffectNames) > 0 {
				g.currentIndex = len(g.filteredEffectNames) - 1
				g.updateStatusMessage()
				g.spawnCurrentEffect(screenWidth/2, screenHeight/2)
			}
		}
	}

	// Clear particles
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.clearAllParticles()
		g.statusMessage = "Cleared all particles"
	}

	// Story 7.6: Angle offset controls (测试粒子方向)
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketLeft) { // [ 键
		g.angleOffset -= 15.0 // 减少 15°
		g.statusMessage = fmt.Sprintf("Angle offset: %.0f°", g.angleOffset)
		log.Printf("Angle offset changed to %.0f°", g.angleOffset)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketRight) { // ] 键
		g.angleOffset += 15.0 // 增加 15°
		g.statusMessage = fmt.Sprintf("Angle offset: %.0f°", g.angleOffset)
		log.Printf("Angle offset changed to %.0f°", g.angleOffset)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackslash) { // \ 键
		g.angleOffset = 0.0 // 重置为 0°
		g.statusMessage = "Angle offset reset to 0°"
		log.Println("Angle offset reset to 0°")
	}

	// Spawn combo effect (ZombieHead + Arms)
	// TODO: Implement spawnComboEffect for combo effects from assets/config/combo_effects.yaml
	/*
		if inpututil.IsKeyJustPressed(ebiten.KeyC) {
			centerX, centerY := screenWidth/2, screenHeight/2
			g.spawnComboEffect("ZombieDeath", float64(centerX), float64(centerY))
		}
	*/

	// Spawn at center
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.spawnCurrentEffect(screenWidth/2, screenHeight/2)
	}

	// Spawn at mouse click
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		g.spawnCurrentEffect(float64(x), float64(y))
	}

	// Auto-play mode
	if g.autoPlay {
		if time.Since(g.lastSpawnTime) > 3*time.Second {
			g.spawnCurrentEffect(screenWidth/2, screenHeight/2)
			g.nextEffect()
			g.lastSpawnTime = time.Now()
		}
	}

	// Update systems
	g.particleSystem.Update(dt)

	// Clean up deleted entities (粒子过期清理)
	g.entityManager.RemoveMarkedEntities()

	return nil
}

// Draw renders the game screen
func (g *ParticleViewerGame) Draw(screen *ebiten.Image) {
	// Clear screen with dark background
	screen.Fill(color.RGBA{25, 25, 38, 255})

	// Draw particles (both game world and UI particles)
	// 大多数粒子效果是游戏世界粒子，需要使用 DrawGameWorldParticles
	g.renderSystem.DrawGameWorldParticles(screen, 0) // cameraX = 0 for centered view
	g.renderSystem.DrawParticles(screen, 0)          // 同时渲染 UI 粒子(如果有)

	// Draw UI overlay
	g.drawUI(screen)
}

// drawUI draws the overlay UI with effect info and controls
func (g *ParticleViewerGame) drawUI(screen *ebiten.Image) {
	if len(g.filteredEffectNames) == 0 {
		ebitenutil.DebugPrintAt(screen, "No effects match current filter", 10, 10)
		return
	}

	currentEffect := g.filteredEffectNames[g.currentIndex]

	// Title
	title := fmt.Sprintf("Particle Viewer - Effect %d/%d", g.currentIndex+1, len(g.filteredEffectNames))
	ebitenutil.DebugPrintAt(screen, title, 10, 10)

	// Search query status
	if g.searchQuery != "" {
		searchStatus := fmt.Sprintf("Filter: \"%s\" (%d/%d effects)", g.searchQuery, len(g.filteredEffectNames), len(g.allEffectNames))
		ebitenutil.DebugPrintAt(screen, searchStatus, 10, 30)
	}

	// Current effect name with emitter info
	emitterCount := g.countActiveEmitters()
	effectInfo := fmt.Sprintf("Effect: %s (%d emitters in this XML)", currentEffect, emitterCount)
	ebitenutil.DebugPrintAt(screen, effectInfo, 10, 50)

	// Particle count
	particleCount := g.countActiveParticles()
	particleInfo := fmt.Sprintf("Active Particles: %d", particleCount)
	ebitenutil.DebugPrintAt(screen, particleInfo, 10, 70)

	// Story 7.6: Angle offset display
	angleInfo := fmt.Sprintf("Angle Offset: %.0f° (use [/]/\\ to adjust)", g.angleOffset)
	ebitenutil.DebugPrintAt(screen, angleInfo, 10, 90)

	// Note about complete effect
	if emitterCount > 1 {
		noteInfo := fmt.Sprintf("(All %d emitters are active - complete effect shown)", emitterCount)
		ebitenutil.DebugPrintAt(screen, noteInfo, 10, 110)
	} else {
		emitterInfo := fmt.Sprintf("Active Emitters: %d", emitterCount)
		ebitenutil.DebugPrintAt(screen, emitterInfo, 10, 110)
	}

	// Search mode indicator
	if g.searchMode {
		searchPrompt := fmt.Sprintf("SEARCH: %s_", g.searchQuery)
		ebitenutil.DebugPrintAt(screen, searchPrompt, 10, 130)
		ebitenutil.DebugPrintAt(screen, "(Type to filter, Backspace to delete, Enter/Esc to exit)", 10, 150)
	} else if g.statusMessage != "" {
		ebitenutil.DebugPrintAt(screen, g.statusMessage, 10, 130)
	}

	// Controls (bottom left)
	controls := []string{
		"Navigation: <-/-> = Next/Prev  PgUp/PgDn = Jump 10  Home/End = First/Last  1-9 = Quick Jump",
		"Actions:    Click/Space = Spawn  R = Clear  P = Pause  F/Slash = Search  Q = Quit",
		"Angle:      [ = -15°  ] = +15°  \\ = Reset to 0°  (Test particle direction)",
	}
	y := screenHeight - len(controls)*20 - 10
	for i, line := range controls {
		ebitenutil.DebugPrintAt(screen, line, 10, y+i*20)
	}

	// Pause indicator (top right)
	if g.paused {
		pauseInfo := "⏸ PAUSED (Press P to resume)"
		ebitenutil.DebugPrintAt(screen, pauseInfo, screenWidth-300, 10)
	} else if g.autoPlay {
		autoPlayInfo := "▶ AUTO-PLAY MODE"
		ebitenutil.DebugPrintAt(screen, autoPlayInfo, screenWidth-200, 10)
	}
}

// Layout returns the game's logical screen size
func (g *ParticleViewerGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// spawnCurrentEffect spawns the currently selected particle effect at the given position
func (g *ParticleViewerGame) spawnCurrentEffect(x, y float64) {
	if len(g.filteredEffectNames) == 0 {
		g.statusMessage = "No effects to spawn"
		return
	}

	effectName := g.filteredEffectNames[g.currentIndex]

	// Story 7.6: 传递角度偏移参数
	_, err := entities.CreateParticleEffect(g.entityManager, g.resourceManager, effectName, x, y, g.angleOffset)
	if err != nil {
		log.Printf("Failed to create effect %s: %v", effectName, err)
		g.statusMessage = fmt.Sprintf("Error: %v", err)
	} else {
		log.Printf("Spawned effect: %s at (%.0f, %.0f) with angle offset %.0f°", effectName, x, y, g.angleOffset)
		g.statusMessage = fmt.Sprintf("Spawned: %s (angle: %.0f°)", effectName, g.angleOffset)
	}
}

// nextEffect switches to the next effect in the list and spawns it
func (g *ParticleViewerGame) nextEffect() {
	if len(g.filteredEffectNames) == 0 {
		return
	}
	g.currentIndex = (g.currentIndex + 1) % len(g.filteredEffectNames)
	g.updateStatusMessage()
	// Auto-spawn the new effect at center
	g.spawnCurrentEffect(screenWidth/2, screenHeight/2)
}

// previousEffect switches to the previous effect in the list and spawns it
func (g *ParticleViewerGame) previousEffect() {
	if len(g.filteredEffectNames) == 0 {
		return
	}
	g.currentIndex = (g.currentIndex - 1 + len(g.filteredEffectNames)) % len(g.filteredEffectNames)
	g.updateStatusMessage()
	// Auto-spawn the new effect at center
	g.spawnCurrentEffect(screenWidth/2, screenHeight/2)
}

// jumpEffects jumps forward or backward by delta effects and spawns
func (g *ParticleViewerGame) jumpEffects(delta int) {
	if len(g.filteredEffectNames) == 0 {
		return
	}
	g.currentIndex = (g.currentIndex + delta) % len(g.filteredEffectNames)
	if g.currentIndex < 0 {
		g.currentIndex += len(g.filteredEffectNames)
	}
	g.updateStatusMessage()
	// Auto-spawn the new effect at center
	g.spawnCurrentEffect(screenWidth/2, screenHeight/2)
}

// clearAllParticles removes all active particles and emitters
func (g *ParticleViewerGame) clearAllParticles() {
	// Query all particle entities
	particles := g.entityManager.GetEntitiesWith(reflect.TypeOf(&components.ParticleComponent{}))

	// Query all emitter entities
	emitters := g.entityManager.GetEntitiesWith(reflect.TypeOf(&components.EmitterComponent{}))

	// Destroy all
	for _, id := range particles {
		g.entityManager.DestroyEntity(id)
	}
	for _, id := range emitters {
		g.entityManager.DestroyEntity(id)
	}

	log.Printf("Cleared %d particles and %d emitters", len(particles), len(emitters))
}

// countActiveParticles returns the number of active particles
func (g *ParticleViewerGame) countActiveParticles() int {
	particles := g.entityManager.GetEntitiesWith(reflect.TypeOf(&components.ParticleComponent{}))
	return len(particles)
}

// countActiveEmitters returns the number of active emitters
func (g *ParticleViewerGame) countActiveEmitters() int {
	emitters := g.entityManager.GetEntitiesWith(reflect.TypeOf(&components.EmitterComponent{}))
	return len(emitters)
}

// updateStatusMessage updates the status message when switching effects
func (g *ParticleViewerGame) updateStatusMessage() {
	if len(g.filteredEffectNames) == 0 {
		g.statusMessage = "No effects available"
		return
	}
	effectName := g.filteredEffectNames[g.currentIndex]
	g.statusMessage = fmt.Sprintf("Selected: %s", effectName)
	log.Printf("Current effect: %s (%d/%d)", effectName, g.currentIndex+1, len(g.filteredEffectNames))
}

func main() {
	flag.Parse()

	log.Println("=== PvZ Particle Effect Viewer ===")
	log.Printf("Initial filter: %q", *filterFlag)
	log.Printf("Start effect: %q", *effectFlag)
	log.Printf("Auto-play: %v", *autoPlayFlag)

	game, err := NewParticleViewerGame()
	if err != nil {
		log.Fatal("Failed to initialize game:", err)
	}

	// 默认静音运行：抑制大量系统级调试日志；如需详细调试，传入 --verbose
	if !*verboseFlag {
		log.SetOutput(io.Discard)
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("PvZ Particle Effect Viewer")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(game); err != nil {
		if err.Error() != "quit requested" {
			log.Fatal(err)
		}
	}

	log.Println("Particle viewer closed")
	os.Exit(0)
}
