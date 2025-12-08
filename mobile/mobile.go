//go:build mobile

// Package mobile 提供 ebitenmobile 绑定入口
//
// 此包用于构建 Android (.aar) 和 iOS (.xcframework) 包。
// 使用 ebitenmobile 工具构建时会自动调用 init() 函数。
//
// 此文件仅在使用 -tags mobile 构建时编译。
// 使用 Makefile 构建（推荐）：
//
//	make build-android    # Android
//	make build-ios        # iOS (仅 macOS)
//
// 手动构建：
//
//	# Android
//	make prepare-mobile && ebitenmobile bind -target android -tags mobile -androidapi 23 -javapkg com.decker.pvz -o build/android/pvz.aar -v ./mobile
//
//	# iOS (仅 macOS)
//	make prepare-mobile && ebitenmobile bind -target ios -tags mobile -o build/ios/PVZ.xcframework -v ./mobile
package mobile

import (
	"image/color"
	"log"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/mobile"

	"github.com/decker502/pvz/pkg/app"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/embedded"
)

// lazyGame 是一个延迟初始化的游戏包装器
// 它在第一次 Update/Draw 调用时才真正初始化游戏
// 这样可以避免在 init() 中过早初始化导致的 Android 黑屏问题
type lazyGame struct {
	once    sync.Once
	realApp *app.App
	initErr error
}

func (g *lazyGame) initialize() {
	g.once.Do(func() {
		log.Println("[Mobile] Starting lazy initialization...")

		// 初始化嵌入资源
		embedded.Init(assetsFS, dataFS)
		log.Println("[Mobile] Embedded resources initialized")

		// 创建游戏应用，使用默认配置
		cfg := app.Config{
			Verbose:          true,  // Enable verbose logging for debugging
			Level:            "",    // 使用存档或默认关卡
			SkipLoadingScene: false, // 显示加载场景
		}

		var err error
		g.realApp, err = app.NewApp(cfg)
		if err != nil {
			log.Printf("[Mobile] Game initialization failed: %v", err)
			g.initErr = err
			return
		}
		log.Println("[Mobile] Game initialized successfully")
	})
}

func (g *lazyGame) Update() error {
	g.initialize()
	if g.initErr != nil {
		// 初始化失败时返回错误（但不会 panic）
		return nil
	}
	if g.realApp != nil {
		return g.realApp.Update()
	}
	return nil
}

func (g *lazyGame) Draw(screen *ebiten.Image) {
	g.initialize()
	if g.initErr != nil {
		// 初始化失败时显示错误信息（红色背景）
		screen.Fill(color.RGBA{255, 0, 0, 255})
		return
	}
	if g.realApp != nil {
		g.realApp.Draw(screen)
	}
}

func (g *lazyGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Layout 在初始化之前也会被调用，返回固定大小
	if g.realApp != nil {
		return g.realApp.Layout(outsideWidth, outsideHeight)
	}
	return config.GameWindowWidth, config.GameWindowHeight
}

func init() {
	log.Println("[Mobile] init() called, registering lazy game...")

	// 使用延迟初始化的游戏包装器
	// 真正的初始化在第一次 Update/Draw 时进行
	mobile.SetGame(&lazyGame{})

	log.Println("[Mobile] Lazy game registered")
}

// Dummy 是一个空导出函数，确保包被 ebitenmobile 正确识别
func Dummy() {}
