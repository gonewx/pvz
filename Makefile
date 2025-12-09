# ============================================================================
# PvZ - Plants vs. Zombies Clone
# Makefile for Cross-Platform Build
# ============================================================================

# 应用名称
APP_NAME := pvz

# Go 模块路径
MODULE := github.com/decker502/pvz

# 构建输出目录
BUILD_DIR := build

# 版本号（从 git tag 获取，如无则使用 dev）
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Go 构建标志（去除调试符号，减小二进制体积）
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION)"

# 当前平台
GOOS_CURRENT := $(shell go env GOOS)
GOARCH_CURRENT := $(shell go env GOARCH)

# WASM 相关路径
GOROOT := $(shell go env GOROOT)
# Go 1.25+ 使用 lib/wasm，旧版本使用 misc/wasm
WASM_EXEC_JS := $(shell if [ -f "$(GOROOT)/lib/wasm/wasm_exec.js" ]; then echo "$(GOROOT)/lib/wasm/wasm_exec.js"; else echo "$(GOROOT)/misc/wasm/wasm_exec.js"; fi)
WASM_HTML_TEMPLATE := scripts/wasm_index.html

# 移动端配置
ANDROID_API := 23
JAVA_PKG := com.decker.pvz

# 图标相关路径
ICON_DIR := assets/icons
SCRIPTS_DIR := scripts

# ============================================================================
# 默认目标
# ============================================================================
.PHONY: help
help: ## 显示帮助信息
	@echo ""
	@echo "PvZ 构建系统 - 版本 $(VERSION)"
	@echo "============================================"
	@echo ""
	@echo "使用方法: make <target>"
	@echo ""
	@echo "可用目标:"
	@echo ""
	@echo "  桌面构建命令:"
	@echo "    build              构建当前平台可执行文件"
	@echo "    build-linux        构建 Linux (amd64 + arm64)"
	@echo "    build-linux-amd64  构建 Linux amd64"
	@echo "    build-linux-arm64  构建 Linux arm64"
	@echo "    build-windows      构建 Windows (amd64 + arm64)"
	@echo "    build-windows-amd64 构建 Windows amd64"
	@echo "    build-windows-arm64 构建 Windows arm64"
	@echo "    build-darwin       构建 macOS (amd64 + arm64 + Universal)"
	@echo "    build-darwin-amd64 构建 macOS amd64 (需要 macOS 主机)"
	@echo "    build-darwin-arm64 构建 macOS arm64 (需要 macOS 主机)"
	@echo "    build-darwin-universal 构建 macOS Universal Binary"
	@echo "    build-wasm         构建 WebAssembly"
	@echo "    serve-wasm         构建并启动 WASM 本地服务器"
	@echo ""
	@echo "  移动端构建命令:"
	@echo "    install-ebitenmobile 安装 ebitenmobile 工具"
	@echo "    prepare-mobile     准备移动端资源（复制 assets/data 到 mobile/）"
	@echo "    build-android      构建 Android .aar (需要 Android SDK/NDK)"
	@echo "    build-ios          构建 iOS .xcframework (需要 macOS + Xcode)"
	@echo "    build-mobile       构建所有移动端平台"
	@echo ""
	@echo "  全平台构建:"
	@echo "    all                构建所有桌面平台 + WASM"
	@echo ""
	@echo "  其他命令:"
	@echo "    generate-icons     生成 Windows .syso 资源文件"
	@echo "    clean              清理构建目录"
	@echo "    clean-mobile       清理移动端资源副本"
	@echo "    help               显示此帮助信息"
	@echo ""
	@echo "构建产物输出到: $(BUILD_DIR)/{platform}/"
	@echo ""

# ============================================================================
# 当前平台构建
# ============================================================================
.PHONY: build
build: ## 构建当前平台可执行文件
	@echo "==> 构建当前平台 ($(GOOS_CURRENT)-$(GOARCH_CURRENT))..."
	@mkdir -p $(BUILD_DIR)/$(GOOS_CURRENT)-$(GOARCH_CURRENT)
ifeq ($(GOOS_CURRENT),windows)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(GOOS_CURRENT)-$(GOARCH_CURRENT)/$(APP_NAME).exe .
else
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(GOOS_CURRENT)-$(GOARCH_CURRENT)/$(APP_NAME) .
endif
	@echo "==> 构建完成: $(BUILD_DIR)/$(GOOS_CURRENT)-$(GOARCH_CURRENT)/"

# ============================================================================
# Linux 构建
# ============================================================================
.PHONY: build-linux build-linux-amd64 build-linux-arm64

build-linux: build-linux-amd64 build-linux-arm64 ## 构建 Linux (amd64 + arm64)
	@echo "==> Linux 构建完成"

build-linux-amd64: ## 构建 Linux amd64
	@echo "==> 构建 Linux amd64..."
	@mkdir -p $(BUILD_DIR)/linux-amd64
ifeq ($(GOOS_CURRENT)-$(GOARCH_CURRENT),linux-amd64)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/linux-amd64/$(APP_NAME) .
else
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/linux-amd64/$(APP_NAME) .
endif
	@echo "==> 完成: $(BUILD_DIR)/linux-amd64/$(APP_NAME)"

build-linux-arm64: ## 构建 Linux arm64
	@echo "==> 构建 Linux arm64..."
	@mkdir -p $(BUILD_DIR)/linux-arm64
ifeq ($(GOOS_CURRENT)-$(GOARCH_CURRENT),linux-arm64)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/linux-arm64/$(APP_NAME) .
	@echo "==> 完成: $(BUILD_DIR)/linux-arm64/$(APP_NAME)"
else
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/linux-arm64/$(APP_NAME) . 2>/dev/null \
		&& echo "==> 完成: $(BUILD_DIR)/linux-arm64/$(APP_NAME)" \
		|| echo "==> 跳过 Linux arm64 (需要 arm64 交叉编译工具链或在 arm64 主机上运行)"
endif

# ============================================================================
# Windows 构建
# ============================================================================
.PHONY: build-windows build-windows-amd64 build-windows-arm64

build-windows: build-windows-amd64 build-windows-arm64 ## 构建 Windows (amd64 + arm64)
	@echo "==> Windows 构建完成"

build-windows-amd64: ## 构建 Windows amd64
	@echo "==> 构建 Windows amd64..."
	@mkdir -p $(BUILD_DIR)/windows-amd64
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/windows-amd64/$(APP_NAME).exe .
	@echo "==> 完成: $(BUILD_DIR)/windows-amd64/$(APP_NAME).exe"

build-windows-arm64: ## 构建 Windows arm64
	@echo "==> 构建 Windows arm64..."
	@mkdir -p $(BUILD_DIR)/windows-arm64
	@CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/windows-arm64/$(APP_NAME).exe .
	@echo "==> 完成: $(BUILD_DIR)/windows-arm64/$(APP_NAME).exe"

# ============================================================================
# macOS 构建 (需要在 macOS 主机上运行)
# ============================================================================
.PHONY: build-darwin build-darwin-amd64 build-darwin-arm64 build-darwin-universal

build-darwin: build-darwin-amd64 build-darwin-arm64 build-darwin-universal ## 构建 macOS (amd64 + arm64 + Universal)
	@echo "==> macOS 构建完成"

build-darwin-amd64: ## 构建 macOS amd64 (需要 macOS 主机)
ifeq ($(GOOS_CURRENT),darwin)
	@echo "==> 构建 macOS amd64..."
	@mkdir -p $(BUILD_DIR)/darwin-amd64
	@CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/darwin-amd64/$(APP_NAME) .
	@echo "==> 完成: $(BUILD_DIR)/darwin-amd64/$(APP_NAME)"
else
	@echo "==> 跳过 macOS amd64 构建 (需要在 macOS 主机上运行)"
endif

build-darwin-arm64: ## 构建 macOS arm64 (需要 macOS 主机)
ifeq ($(GOOS_CURRENT),darwin)
	@echo "==> 构建 macOS arm64..."
	@mkdir -p $(BUILD_DIR)/darwin-arm64
	@CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/darwin-arm64/$(APP_NAME) .
	@echo "==> 完成: $(BUILD_DIR)/darwin-arm64/$(APP_NAME)"
else
	@echo "==> 跳过 macOS arm64 构建 (需要在 macOS 主机上运行)"
endif

build-darwin-universal: build-darwin-amd64 build-darwin-arm64 ## 构建 macOS Universal Binary
ifeq ($(GOOS_CURRENT),darwin)
	@echo "==> 构建 macOS Universal Binary..."
	@mkdir -p $(BUILD_DIR)/darwin-universal
	@lipo -create \
		$(BUILD_DIR)/darwin-amd64/$(APP_NAME) \
		$(BUILD_DIR)/darwin-arm64/$(APP_NAME) \
		-output $(BUILD_DIR)/darwin-universal/$(APP_NAME)
	@echo "==> 完成: $(BUILD_DIR)/darwin-universal/$(APP_NAME)"
else
	@echo "==> 跳过 macOS Universal 构建 (需要在 macOS 主机上运行)"
endif

# ============================================================================
# WASM 构建
# ============================================================================
.PHONY: build-wasm

build-wasm: ## 构建 WebAssembly
	@echo "==> 构建 WebAssembly..."
	@mkdir -p $(BUILD_DIR)/wasm
	@GOOS=js GOARCH=wasm go build $(LDFLAGS) -o $(BUILD_DIR)/wasm/$(APP_NAME).wasm .
	@echo "==> 复制 wasm_exec.js..."
	@cp $(WASM_EXEC_JS) $(BUILD_DIR)/wasm/wasm_exec.js
	@echo "==> 复制 index.html..."
	@cp $(WASM_HTML_TEMPLATE) $(BUILD_DIR)/wasm/index.html
	@echo "==> 完成: $(BUILD_DIR)/wasm/"
	@ls -la $(BUILD_DIR)/wasm/

# ============================================================================
# WASM 本地服务器
# ============================================================================
.PHONY: serve-wasm

serve-wasm: build-wasm ## 构建并启动 WASM 本地服务器 (http://localhost:8080)
	@echo "==> 启动本地服务器..."
	@echo "==> 在浏览器中打开: http://localhost:8080"
	@echo "==> 按 Ctrl+C 停止服务器"
	@cd $(BUILD_DIR)/wasm && python3 -m http.server 8080

# ============================================================================
# 移动端构建
# ============================================================================
.PHONY: install-ebitenmobile prepare-mobile clean-mobile build-android build-ios build-mobile

install-ebitenmobile: ## 安装 ebitenmobile 工具
	@echo "==> 安装 ebitenmobile..."
	@go install github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@latest
	@echo "==> 安装完成"
	@echo "==> 验证安装:"
	@ebitenmobile version || echo "警告: ebitenmobile 未在 PATH 中，请确保 GOPATH/bin 在 PATH 中"

prepare-mobile: ## 准备移动端资源（复制 assets/data 到 mobile/）
	@echo "==> 准备移动端资源..."
	@rm -rf mobile/assets mobile/data
	@cp -r assets mobile/assets
	@cp -r data mobile/data
	@echo "==> 资源复制完成"

clean-mobile: ## 清理移动端资源副本
	@echo "==> 清理移动端资源..."
	@rm -rf mobile/assets mobile/data
	@echo "==> 清理完成"

build-android: prepare-mobile ## 构建 Android .aar (需要 Android SDK/NDK)
	@echo "==> 构建 Android AAR..."
	@echo "==> Android API: $(ANDROID_API)"
	@echo "==> Java 包名: $(JAVA_PKG)"
	@mkdir -p $(BUILD_DIR)/android
	@ebitenmobile bind -target android -tags mobile -androidapi $(ANDROID_API) -javapkg $(JAVA_PKG) -o $(BUILD_DIR)/android/$(APP_NAME).aar -v ./mobile
	@echo "==> 完成: $(BUILD_DIR)/android/$(APP_NAME).aar"
	@ls -la $(BUILD_DIR)/android/

build-ios: prepare-mobile ## 构建 iOS .xcframework (需要 macOS + Xcode)
ifeq ($(GOOS_CURRENT),darwin)
	@echo "==> 构建 iOS XCFramework..."
	@mkdir -p $(BUILD_DIR)/ios
	@ebitenmobile bind -target ios -tags mobile -o $(BUILD_DIR)/ios/PVZ.xcframework -v ./mobile
	@echo "==> 完成: $(BUILD_DIR)/ios/PVZ.xcframework"
	@ls -la $(BUILD_DIR)/ios/
else
	@echo "==> 跳过 iOS 构建 (需要在 macOS 主机上运行)"
endif

build-mobile: build-android build-ios ## 构建所有移动端平台
	@echo "==> 移动端构建完成"

build-apk: ## 构建 Android APK (需要 JDK)
	@./scripts/build-apk.sh

sign-apk: ## 签名 APK 文件 (用法: make sign-apk APK=path/to/app.apk)
	@./scripts/sign-apk.sh $(APK)

# ============================================================================
# 全平台构建
# ============================================================================
.PHONY: all

all: build-linux build-windows build-wasm ## 构建所有平台
ifeq ($(GOOS_CURRENT),darwin)
	@$(MAKE) build-darwin
endif
	@echo ""
	@echo "============================================"
	@echo "全平台构建完成!"
	@echo "============================================"
	@echo ""
	@ls -la $(BUILD_DIR)/

# ============================================================================
# 清理
# ============================================================================
.PHONY: clean

clean: clean-mobile ## 清理构建目录
	@echo "==> 清理构建目录..."
	@rm -rf $(BUILD_DIR)
	@echo "==> 清理完成"

# ============================================================================
# 默认目标设置
# ============================================================================
.DEFAULT_GOAL := help

# ============================================================================
# 图标和资源生成
# ============================================================================
.PHONY: generate-icons generate-windows-syso build-darwin-app

generate-icons: generate-windows-syso ## 生成所有平台图标资源

generate-windows-syso: ## 生成 Windows .syso 资源文件 (需要 goversioninfo)
	@echo "==> 生成 Windows 资源文件..."
	@which goversioninfo > /dev/null || (echo "安装 goversioninfo..." && go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest)
	@goversioninfo -64 -o resource_windows_amd64.syso
	@goversioninfo -arm -o resource_windows_arm64.syso
	@echo "==> 完成: resource_windows_amd64.syso, resource_windows_arm64.syso"

build-darwin-app: build-darwin-universal ## 构建 macOS .app 包 (需要 macOS 主机)
ifeq ($(GOOS_CURRENT),darwin)
	@echo "==> 创建 macOS .app 包..."
	@mkdir -p $(BUILD_DIR)/darwin-app/$(APP_NAME).app/Contents/MacOS
	@mkdir -p $(BUILD_DIR)/darwin-app/$(APP_NAME).app/Contents/Resources
	@cp $(BUILD_DIR)/darwin-universal/$(APP_NAME) $(BUILD_DIR)/darwin-app/$(APP_NAME).app/Contents/MacOS/
	@cp $(SCRIPTS_DIR)/Info.plist $(BUILD_DIR)/darwin-app/$(APP_NAME).app/Contents/
	@if [ -d "$(ICON_DIR)/macos/icon.iconset" ]; then \
		iconutil -c icns $(ICON_DIR)/macos/icon.iconset -o $(BUILD_DIR)/darwin-app/$(APP_NAME).app/Contents/Resources/icon.icns; \
	fi
	@echo "==> 完成: $(BUILD_DIR)/darwin-app/$(APP_NAME).app"
else
	@echo "==> 跳过 macOS .app 构建 (需要在 macOS 主机上运行)"
endif

# ============================================================================
# Linux 打包 (带图标和 .desktop 文件)
# ============================================================================
.PHONY: package-linux

package-linux: build-linux-amd64 ## 打包 Linux 发布包 (包含图标和 .desktop)
	@echo "==> 打包 Linux 发布包..."
	@mkdir -p $(BUILD_DIR)/linux-package/usr/bin
	@mkdir -p $(BUILD_DIR)/linux-package/usr/share/applications
	@mkdir -p $(BUILD_DIR)/linux-package/usr/share/icons/hicolor/16x16/apps
	@mkdir -p $(BUILD_DIR)/linux-package/usr/share/icons/hicolor/32x32/apps
	@mkdir -p $(BUILD_DIR)/linux-package/usr/share/icons/hicolor/48x48/apps
	@mkdir -p $(BUILD_DIR)/linux-package/usr/share/icons/hicolor/64x64/apps
	@mkdir -p $(BUILD_DIR)/linux-package/usr/share/icons/hicolor/128x128/apps
	@mkdir -p $(BUILD_DIR)/linux-package/usr/share/icons/hicolor/256x256/apps
	@mkdir -p $(BUILD_DIR)/linux-package/usr/share/icons/hicolor/512x512/apps
	@cp $(BUILD_DIR)/linux-amd64/$(APP_NAME) $(BUILD_DIR)/linux-package/usr/bin/
	@cp $(SCRIPTS_DIR)/pvz.desktop $(BUILD_DIR)/linux-package/usr/share/applications/
	@cp $(ICON_DIR)/linux/icon_16.png $(BUILD_DIR)/linux-package/usr/share/icons/hicolor/16x16/apps/pvz.png
	@cp $(ICON_DIR)/linux/icon_32.png $(BUILD_DIR)/linux-package/usr/share/icons/hicolor/32x32/apps/pvz.png
	@cp $(ICON_DIR)/linux/icon_48.png $(BUILD_DIR)/linux-package/usr/share/icons/hicolor/48x48/apps/pvz.png
	@cp $(ICON_DIR)/linux/icon_64.png $(BUILD_DIR)/linux-package/usr/share/icons/hicolor/64x64/apps/pvz.png
	@cp $(ICON_DIR)/linux/icon_128.png $(BUILD_DIR)/linux-package/usr/share/icons/hicolor/128x128/apps/pvz.png
	@cp $(ICON_DIR)/linux/icon_256.png $(BUILD_DIR)/linux-package/usr/share/icons/hicolor/256x256/apps/pvz.png
	@cp $(ICON_DIR)/linux/icon_512.png $(BUILD_DIR)/linux-package/usr/share/icons/hicolor/512x512/apps/pvz.png
	@echo "==> 完成: $(BUILD_DIR)/linux-package/"
	@echo "==> 可使用以下命令安装: sudo cp -r $(BUILD_DIR)/linux-package/* /"

# ============================================================================
# iOS 图标说明
# ============================================================================
.PHONY: ios-icons-info

ios-icons-info: ## 显示 iOS 图标使用说明
	@echo ""
	@echo "iOS 图标使用说明"
	@echo "============================================"
	@echo ""
	@echo "iOS 图标已生成在: $(ICON_DIR)/ios/AppIcon.appiconset/"
	@echo ""
	@echo "使用方法:"
	@echo "  1. 在 Xcode 中打开你的 iOS 项目"
	@echo "  2. 在项目导航器中找到 Assets.xcassets"
	@echo "  3. 右键点击 Assets.xcassets -> Show in Finder"
	@echo "  4. 将 $(ICON_DIR)/ios/AppIcon.appiconset 文件夹"
	@echo "     复制到 Assets.xcassets 目录下"
	@echo "  5. 返回 Xcode，图标应该自动显示"
	@echo ""
	@echo "或者使用以下命令复制 (在 macOS 上):"
	@echo "  cp -r $(ICON_DIR)/ios/AppIcon.appiconset /path/to/YourProject/Assets.xcassets/"
	@echo ""

