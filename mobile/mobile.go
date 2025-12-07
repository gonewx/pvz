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
	"log"

	"github.com/hajimehoshi/ebiten/v2/mobile"

	"github.com/decker502/pvz/pkg/app"
	"github.com/decker502/pvz/pkg/embedded"
)

func init() {
	// 初始化嵌入资源
	// assetsFS 和 dataFS 在 embed.go 中声明
	embedded.Init(assetsFS, dataFS)

	// 创建游戏应用，使用默认配置
	cfg := app.Config{
		Verbose:          true,  // Enable verbose logging for debugging
		Level:            "",    // 使用存档或默认关卡
		SkipLoadingScene: false, // 显示加载场景
	}

	gameApp, err := app.NewApp(cfg)
	if err != nil {
		log.Fatalf("游戏初始化失败: %v", err)
	}

	// 注册游戏到 ebitenmobile
	mobile.SetGame(gameApp)
}

// Dummy 是一个空导出函数，确保包被 ebitenmobile 正确识别
func Dummy() {}
