package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/scenes"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

// Game represents the main game structure.
// It implements the ebiten.Game interface to provide the core game loop.
type Game struct {
	sceneManager *game.SceneManager
}

// Update updates the game logic.
// This method is called every tick (typically 60 times per second).
// Returns an error if the game should terminate.
func (g *Game) Update() error {
	// Calculate delta time (assuming 60 TPS - Ticks Per Second)
	deltaTime := 1.0 / 60.0
	g.sceneManager.Update(deltaTime)
	return nil
}

// Draw renders the game screen.
// This method is called every frame to draw the game content.
func (g *Game) Draw(screen *ebiten.Image) {
	g.sceneManager.Draw(screen)
}

// Layout returns the game's logical screen size.
// This size is independent of the actual window size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 800, 600
}

func main() {
	// Flags
	verboseFlag := flag.Bool("verbose", false, "Enable verbose logging (default off)")
	levelFlag := flag.String("level", "", "Specify which level to load (e.g., '1-2', '1-3'). If not set, loads from save or defaults to 1-1")
	flag.Parse()

	// Initialize audio context with 48000 Hz sample rate
	audioContext := audio.NewContext(48000)

	// Create resource manager
	resourceManager := game.NewResourceManager(audioContext)

	// Quiet logs by default unless --verbose
	if !*verboseFlag {
		// 丢弃所有 log.Printf 输出；仅使用下方 fmt.Fprintln 输出关键错误
		log.SetOutput(io.Discard)
		log.SetFlags(0)
	}

	// Load resource configuration from YAML
	if err := resourceManager.LoadResourceConfig("assets/config/resources.yaml"); err != nil {
		if *verboseFlag {
			log.Printf("Failed to load resource config: %v", err)
		} else {
			fmt.Fprintln(os.Stderr, "资源配置加载失败（使用 --verbose 查看详细日志）")
		}
		os.Exit(1)
	}

	// Load all Reanim resources (auto-scan assets/effect/reanim directory)
	if err := resourceManager.LoadReanimResources(); err != nil {
		if *verboseFlag {
			log.Printf("Failed to load Reanim resources: %v", err)
		} else {
			fmt.Fprintln(os.Stderr, "Reanim 资源加载失败（使用 --verbose 查看详细日志）")
		}
		os.Exit(1)
	}

	// Load Reanim playback mode configuration
	// This provides explicit mode configuration for key animations (SelectorScreen, PeaShooter, etc.)
	// Falls back to heuristic algorithm if config not found
	if err := config.LoadReanimPlaybackConfig(); err != nil {
		// Non-fatal error: will use heuristic algorithm as fallback
		log.Printf("⚠️  Reanim playback config not loaded (will use heuristic algorithm): %v", err)
	}

	// Load Reanim 配置管理器 (Story 13.6)
	reanimConfigManager, err := config.NewReanimConfigManager("data/reanim_config.yaml")
	if err != nil {
		if *verboseFlag {
			log.Printf("Failed to load Reanim config: %v", err)
		} else {
			fmt.Fprintln(os.Stderr, "Reanim 配置加载失败（使用 --verbose 查看详细日志）")
		}
		os.Exit(1)
	}
	log.Printf("[Config] 加载 Reanim 配置: data/reanim_config.yaml")
	log.Printf("[Config] 成功加载 %d 个动画单元配置", len(reanimConfigManager.ListUnits()))
	log.Printf("[Config] 配置管理器初始化完成")

	// 将配置管理器传递给 ResourceManager
	resourceManager.SetReanimConfigManager(reanimConfigManager)

	// Create scene manager
	sceneManager := game.NewSceneManager()

	// 设置场景工厂函数，用于在奖励动画完成后加载下一关
	sceneManager.SetSceneFactory(func(levelID string) game.Scene {
		return scenes.NewGameScene(resourceManager, sceneManager, levelID)
	})

	// Determine which level to load:
	// 1. Command line --level flag (highest priority)
	// 2. Highest completed level from save file (default)
	// 3. Fallback to 1-1 (new game)
	levelToLoad := *levelFlag
	if levelToLoad == "" {
		// No --level flag, try to load from save
		gameState := game.GetGameState()
		saveManager := gameState.GetSaveManager()
		if err := saveManager.Load(); err == nil {
			// Save file exists, get highest level
			highestLevel := saveManager.GetHighestLevel()
			if highestLevel != "" {
				levelToLoad = highestLevel
				if *verboseFlag {
					log.Printf("[main] Loading from save: highest level = %s", highestLevel)
				}
			}
		}
	}

	// Fallback to 1-1 if still empty
	if levelToLoad == "" {
		levelToLoad = "1-1"
		if *verboseFlag {
			log.Printf("[main] No save found, starting new game at level 1-1")
		}
	}

	if *verboseFlag {
		log.Printf("[main] Starting level: %s", levelToLoad)
	}

	// 根据 --level 参数决定启动场景
	if *levelFlag != "" {
		// 如果指定了 --level 参数，直接启动游戏场景（跳过主菜单）
		if *verboseFlag {
			log.Printf("[main] --level flag detected, skipping main menu")
		}
		gameScene := scenes.NewGameScene(resourceManager, sceneManager, levelToLoad)
		sceneManager.SwitchTo(gameScene)
	} else {
		// 否则启动主菜单场景
		mainMenuScene := scenes.NewMainMenuScene(resourceManager, sceneManager)
		sceneManager.SwitchTo(mainMenuScene)
	}

	// Create a new game instance with the scene manager
	gameInstance := &Game{
		sceneManager: sceneManager,
	}

	// Set window properties
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("植物大战僵尸 - Go复刻版")

	// Start the game loop
	// This will call Update() and Draw() repeatedly until the window is closed
	if err := ebiten.RunGame(gameInstance); err != nil {
		if *verboseFlag {
			log.Printf("RunGame error: %v", err)
		} else {
			fmt.Fprintln(os.Stderr, "程序运行异常（使用 --verbose 查看详细日志）")
		}
		os.Exit(1)
	}
}
