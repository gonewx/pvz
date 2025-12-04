package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/decker502/pvz/pkg/app"
	"github.com/decker502/pvz/pkg/embedded"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	// 初始化 embedded 包（必须在任何资源加载之前）
	embedded.Init(assetsFS, dataFS)

	// 解析命令行参数
	verboseFlag := flag.Bool("verbose", false, "Enable verbose logging (default off)")
	levelFlag := flag.String("level", "", "Specify which level to load (e.g., '1-2', '1-3'). If not set, loads from save or defaults to 1-1")
	flag.Parse()

	// 创建应用配置
	cfg := app.Config{
		Verbose:          *verboseFlag,
		Level:            *levelFlag,
		SkipLoadingScene: *levelFlag != "", // 如果指定了关卡，跳过加载场景
	}

	// 创建游戏应用
	gameApp, err := app.NewApp(cfg)
	if err != nil {
		if *verboseFlag {
			log.Printf("游戏初始化失败: %v", err)
		} else {
			fmt.Fprintln(os.Stderr, "游戏初始化失败（使用 --verbose 查看详细日志）")
		}
		os.Exit(1)
	}

	// 设置窗口属性
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("植物大战僵尸 - Go复刻版")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// 应用保存的全屏设置
	gameState := game.GetGameState()
	if settingsManager := gameState.GetSettingsManager(); settingsManager != nil {
		settings := settingsManager.GetSettings()
		if settings.Fullscreen {
			ebiten.SetFullscreen(true)
			if *verboseFlag {
				log.Printf("[main] Applying saved fullscreen setting: true")
			}
		}
	}

	// 启动游戏循环
	if err := ebiten.RunGame(gameApp); err != nil {
		if *verboseFlag {
			log.Printf("RunGame error: %v", err)
		} else {
			fmt.Fprintln(os.Stderr, "程序运行异常（使用 --verbose 查看详细日志）")
		}
		os.Exit(1)
	}

	// 游戏关闭时自动保存战斗存档
	if sceneManager := gameApp.GetSceneManager(); sceneManager != nil {
		if currentScene := sceneManager.GetCurrentScene(); currentScene != nil {
			if saveable, ok := currentScene.(game.Saveable); ok {
				if *verboseFlag {
					log.Printf("[main] 游戏关闭，检查是否需要保存存档...")
				}
				saveable.SaveOnExit()
			}
		}
	}
}
