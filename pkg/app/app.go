// Package app 提供游戏应用的核心包装器
//
// 该包将游戏初始化逻辑从 main 包提取出来，使其可以被桌面端和移动端共用。
// 桌面端通过 main.go 调用 NewApp()，移动端通过 mobile/mobile.go 调用。
package app

import (
	"fmt"
	"image/color"
	"io"
	"log"

	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/scenes"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Config 定义应用启动配置
type Config struct {
	// Verbose 启用详细日志输出
	Verbose bool
	// Level 指定要加载的关卡（如 "1-2"），为空则从存档加载或默认 1-1
	Level string
	// SkipLoadingScene 跳过加载场景，直接进入游戏（用于 --level 参数）
	SkipLoadingScene bool
}

// App 是游戏应用的核心包装器，实现 ebiten.Game 接口
type App struct {
	sceneManager             *game.SceneManager
	verbose                  bool
	pendingWindowSizeReset   bool // 延迟设置窗口大小标志
	windowSizeResetCountdown int  // 延迟帧数
}

// NewApp 创建并初始化游戏应用
//
// 调用此函数前，必须先调用 embedded.Init() 初始化嵌入资源。
func NewApp(cfg Config) (*App, error) {
	// 配置日志输出
	if !cfg.Verbose {
		log.SetOutput(io.Discard)
		log.SetFlags(0)
	}

	// 初始化音频上下文
	audioContext := audio.NewContext(48000)

	// 创建资源管理器
	resourceManager := game.NewResourceManager(audioContext)

	// 加载资源配置
	if err := resourceManager.LoadResourceConfig("assets/config/resources.yaml"); err != nil {
		return nil, fmt.Errorf("资源配置加载失败: %w", err)
	}

	// 加载 Reanim 资源
	if err := resourceManager.LoadReanimResources(); err != nil {
		return nil, fmt.Errorf("Reanim 资源加载失败: %w", err)
	}

	// 加载 Reanim 配置管理器
	reanimConfigManager, err := config.NewReanimConfigManager("data/reanim_config")
	if err != nil {
		return nil, fmt.Errorf("Reanim 配置加载失败: %w", err)
	}
	log.Printf("[Config] 加载 Reanim 配置目录: data/reanim_config/")
	log.Printf("[Config] 成功加载 %d 个动画单元配置", len(reanimConfigManager.ListUnits()))
	log.Printf("[Config] 配置管理器初始化完成")

	// 将配置管理器传递给 ResourceManager
	resourceManager.SetReanimConfigManager(reanimConfigManager)

	// 初始化 AudioManager 并设置到 GameState
	gameState := game.GetGameState()
	audioManager := game.NewAudioManager(resourceManager, gameState.GetSettingsManager())
	gameState.SetAudioManager(audioManager)
	log.Printf("[App] AudioManager initialized")

	// 创建场景管理器
	sceneManager := game.NewSceneManager()
	sceneManager.SetSceneFactory(func(levelID string) game.Scene {
		return scenes.NewGameScene(resourceManager, sceneManager, levelID)
	})

	// 确定加载哪个关卡
	levelToLoad := cfg.Level
	if levelToLoad == "" {
		// 尝试从存档加载
		gameState := game.GetGameState()
		saveManager := gameState.GetSaveManager()
		if err := saveManager.Load(); err == nil {
			highestLevel := saveManager.GetHighestLevel()
			if highestLevel != "" {
				levelToLoad = highestLevel
				log.Printf("[App] Loading from save: highest level = %s", highestLevel)
			}
		}
	}

	// 默认关卡
	if levelToLoad == "" {
		levelToLoad = "1-1"
		log.Printf("[App] No save found, starting new game at level 1-1")
	}

	log.Printf("[App] Starting level: %s", levelToLoad)

	// 根据配置决定启动场景
	if cfg.SkipLoadingScene {
		log.Printf("[App] SkipLoadingScene enabled, skipping loading scene and main menu")
		gameScene := scenes.NewGameScene(resourceManager, sceneManager, levelToLoad)
		sceneManager.SwitchTo(gameScene)
	} else {
		loadingScene := scenes.NewLoadingScene(resourceManager, sceneManager, reanimConfigManager)
		sceneManager.SwitchTo(loadingScene)
	}

	return &App{
		sceneManager: sceneManager,
		verbose:      cfg.Verbose,
	}, nil
}

// Update 更新游戏逻辑
// 每个 tick 调用一次（通常每秒 60 次）
func (a *App) Update() error {
	// 延迟设置窗口大小（退出全屏后需要等待几帧才能正确设置）
	if a.pendingWindowSizeReset {
		a.windowSizeResetCountdown--
		if a.windowSizeResetCountdown <= 0 {
			ebiten.SetWindowSize(config.GameWindowWidth, config.GameWindowHeight)
			log.Printf("[App] Delayed SetWindowSize(%d, %d)", config.GameWindowWidth, config.GameWindowHeight)
			a.pendingWindowSizeReset = false
		}
	}

	// F11 切换全屏
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		isFullscreen := ebiten.IsFullscreen()
		if isFullscreen {
			// 退出全屏
			ebiten.SetFullscreen(false)
			if ebiten.IsWindowMaximized() || ebiten.IsWindowMinimized() {
				ebiten.RestoreWindow()
			}
			// 延迟几帧后设置窗口大小，让窗口管理器有时间处理
			a.pendingWindowSizeReset = true
			a.windowSizeResetCountdown = 3
			log.Printf("[App] Exit fullscreen, will reset window size in 3 frames")
		} else {
			ebiten.SetFullscreen(true)
		}
	}

	deltaTime := 1.0 / 60.0
	a.sceneManager.Update(deltaTime)
	return nil
}

// Draw 绘制游戏画面
// 每帧调用一次
func (a *App) Draw(screen *ebiten.Image) {
	a.sceneManager.Draw(screen)
}

// DrawFinalScreen 实现 FinalScreenDrawer 接口
// 用于控制全屏时的缩放和 letterbox 颜色
func (a *App) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM) {
	// 先填充黑色背景（全屏时左右两边为黑色）
	screen.Fill(color.Black)
	// 使用线性滤波绘制游戏画面，提高缩放质量
	op := &ebiten.DrawImageOptions{}
	op.GeoM = geoM
	op.Filter = ebiten.FilterLinear // 使用线性滤波减少锯齿和模糊
	screen.DrawImage(offscreen, op)
}

// Layout 返回游戏的逻辑屏幕尺寸
// 此尺寸独立于实际窗口大小，Ebitengine 会自动处理缩放
func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return config.GameWindowWidth, config.GameWindowHeight
}

// GetSceneManager 返回场景管理器
// 用于在游戏关闭时保存存档
func (a *App) GetSceneManager() *game.SceneManager {
	return a.sceneManager
}

// IsVerbose 返回是否启用了详细日志
func (a *App) IsVerbose() bool {
	return a.verbose
}
