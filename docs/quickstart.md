# 快速开始指南

本指南将帮助您在 **5 分钟内**运行《植物大战僵尸 Go 复刻版》。

## 📋 环境要求

### 必需软件

| 软件 | 最低版本 | 推荐版本 | 说明 |
|------|---------|---------|------|
| **Go** | 1.21 | 1.22+ | [下载地址](https://golang.org/dl/) |
| **Git** | 2.0+ | 最新版 | 用于克隆仓库 |

### 系统要求

| 组件 | 最低要求 | 推荐配置 |
|------|---------|---------|
| **操作系统** | Windows 10 / macOS 10.15 / Linux (Kernel 4.15+) | 最新稳定版 |
| **内存** | 2GB RAM | 4GB+ RAM |
| **显卡** | OpenGL 2.1+ | OpenGL 3.0+ |
| **磁盘空间** | 500MB | 1GB |

### 验证环境

在开始前，请验证 Go 已正确安装：

```bash
# 检查 Go 版本
go version
# 应输出: go version go1.21.x 或更高

# 检查 Go 环境
go env GOPATH
# 应输出 Go 工作目录路径
```

## 🚀 快速开始（5分钟）

### 步骤 1: 克隆项目

```bash
# 克隆仓库
git clone <repository-url>

# 进入项目目录
cd pvz
```

### 步骤 2: 下载依赖

```bash
# 下载项目依赖
go mod download

# 整理依赖（可选，推荐）
go mod tidy
```

**预期输出**:
```
go: downloading github.com/hajimehoshi/ebiten/v2 v2.x.x
go: downloading gopkg.in/yaml.v3 v3.x.x
...
```

### 步骤 3: 验证资源文件

确保 `assets/` 目录包含所有必需资源：

```bash
# 检查关键资源目录
ls -la assets/images
ls -la assets/effect
ls -la data/reanim
ls -la data/levels
```

**必需的资源**:
- ✅ `assets/images/` - 游戏图片资源
- ✅ `assets/effect/` - 粒子配置
- ✅ `data/levels/` - 关卡配置文件

### 步骤 4: 运行游戏

```bash
# 直接运行（开发模式）
go run .
```

**预期结果**:
- 🎮 游戏窗口（800x600）正常打开
- 🎵 背景音乐开始播放
- 🖼️ 主菜单界面显示

**成功！** 您现在可以开始游戏了。点击"开始冒险"进入关卡选择。

## 🔧 进阶操作

### 构建可执行文件

如果您想创建独立的可执行文件：

```bash
# 构建当前平台版本
go build -o pvz-go .

# 运行可执行文件
./pvz-go  # Linux/macOS
pvz-go.exe  # Windows
```

### 优化构建（减小文件体积）

```bash
# 构建优化版本（移除调试信息和符号表）
go build -ldflags="-s -w" -o pvz-go .
```

**体积对比**:
- 普通构建: ~30-40 MB
- 优化构建: ~20-25 MB

### 交叉编译（跨平台构建）

```bash
# 编译 Windows 64位版本（在任意平台）
GOOS=windows GOARCH=amd64 go build -o pvz-go-windows.exe .

# 编译 macOS 64位版本
GOOS=darwin GOARCH=amd64 go build -o pvz-go-macos .

# 编译 Linux 64位版本
GOOS=linux GOARCH=amd64 go build -o pvz-go-linux .
```

### 启用详细日志（调试）

```bash
# 运行游戏并显示详细日志
go run . --verbose
```

**日志输出示例**:
```
[ReanimSystem] 自动轨道绑定 (entity 123):
  - anim_face -> anim_head_idle
  - stalk_bottom -> anim_shooting
[ParticleSystem] 生成粒子效果: Planting (100 粒子)
```

## ❓ 常见问题

### 问题 1: "missing go.sum entry" 错误

**原因**: 依赖缓存不一致

**解决方案**:
```bash
go mod tidy
go mod download
```

### 问题 2: 游戏启动后黑屏

**可能原因**:
1. 资源文件缺失
2. OpenGL 版本不支持

**解决方案**:
```bash
# 1. 检查 assets 目录是否完整
ls -R assets/ | head -20

# 2. 验证 OpenGL 支持（Linux）
glxinfo | grep "OpenGL version"

# 3. 尝试运行详细日志模式
go run . --verbose
```

### 问题 3: 编译错误 "cannot find package"

**原因**: Ebitengine 依赖的系统库缺失（仅 Linux）

**解决方案**:

**Ubuntu/Debian**:
```bash
sudo apt-get install libc6-dev libglu1-mesa-dev libgl1-mesa-dev \
  libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev \
  libxxf86vm-dev libasound2-dev pkg-config
```

**Fedora/RedHat**:
```bash
sudo dnf install mesa-libGL-devel mesa-libGLU-devel \
  libXcursor-devel libXi-devel libXinerama-devel \
  libXrandr-devel libXxf86vm-devel alsa-lib-devel
```

**Arch Linux**:
```bash
sudo pacman -S mesa libxcursor libxi libxinerama libxrandr
```

### 问题 4: 性能问题（FPS 低于 60）

**解决方案**:
```bash
# 1. 使用优化构建
go build -ldflags="-s -w" -o pvz-go .

# 2. 关闭不必要的后台程序

# 3. 降低游戏复杂度（调试）
# 在 main.go 中调整粒子数量限制
```

### 问题 5: macOS 提示"无法验证开发者"

**原因**: macOS Gatekeeper 安全限制

**解决方案**:
```bash
# 允许运行未签名应用
xattr -d com.apple.quarantine pvz-go-macos

# 或在"系统偏好设置 > 安全性与隐私"中手动允许
```

### 问题 6: Windows Defender 误报病毒

**原因**: Go 编译的可执行文件可能被误判

**解决方案**:
1. 将文件添加到 Windows Defender 例外列表
2. 使用 `go run .` 直接运行源代码

## 📱 移动端构建

### Android APK 构建

#### 环境要求

| 组件 | 说明 | 安装方法 |
|------|------|---------|
| **Android SDK** | Android 开发工具包 | [下载 Android Studio](https://developer.android.com/studio) |
| **Android NDK** | 原生开发工具包 | 通过 SDK Manager 安装 |
| **JDK** | Java 开发工具包 (8+) | `sudo apt install openjdk-17-jdk` |
| **ebitenmobile** | Ebitengine 移动端工具 | `go install github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@latest` |

#### 设置环境变量

```bash
# 设置 Android SDK 路径
export ANDROID_HOME=/path/to/android-sdk
export ANDROID_NDK_HOME=$ANDROID_HOME/ndk/<version>

# 验证环境
echo $ANDROID_HOME
javac -version
ebitenmobile version
```

#### 构建 APK（一键构建）

```bash
# 设置环境变量后，运行一键构建脚本
export ANDROID_HOME=/home/decker/app/android/sdk
export ANDROID_NDK_HOME=$ANDROID_HOME/ndk/27.2.12479018

# 构建未签名 APK（用于测试）
make build-apk
```

**构建流程**:
1. ✅ 编译 Go 代码为 AAR 库
2. ✅ 创建 Android 项目结构
3. ✅ 生成 AndroidManifest.xml 和 MainActivity
4. ✅ 使用 Gradle 构建 APK

**输出文件**: `build/pvz-unsigned.apk`

#### 安装测试

```bash
# 通过 adb 安装到 Android 设备
adb install -r build/pvz-unsigned.apk

# 查看设备列表
adb devices
```

#### 签名 APK（发布版本）

```bash
# 生成签名 APK
make sign-apk APK=build/pvz-unsigned.apk

# 或使用脚本
./scripts/sign-apk.sh build/pvz-unsigned.apk
```

**输出文件**: `build/pvz-unsigned-signed.apk`

**注意**: 签名使用测试密钥库（密码: `android`），发布到商店前需要使用正式密钥。

#### 手动构建（分步）

如果需要自定义构建流程：

```bash
# 1. 仅构建 AAR 库
make prepare-mobile
make build-android

# 2. 手动创建 Android 项目并集成 AAR
# （参考 scripts/build-apk.sh）

# 3. 使用 Gradle 构建
cd build/android-project
./gradlew assembleRelease
```

#### 常见问题

**Q: "javac: command not found"**
```bash
# 安装 JDK
sudo apt install openjdk-17-jdk

# 验证安装
javac -version
```

**Q: "Android SDK not found"**
```bash
# 设置正确的 ANDROID_HOME
export ANDROID_HOME=/path/to/android/sdk

# 或安装 Android Studio 后使用默认路径
export ANDROID_HOME=$HOME/Android/Sdk  # Linux
```

**Q: "ebitenmobile: command not found"**
```bash
# 安装 ebitenmobile
go install github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@latest

# 确保 $GOPATH/bin 在 PATH 中
export PATH=$PATH:$(go env GOPATH)/bin
```

**Q: APK 安装失败**
```bash
# 卸载旧版本
adb uninstall com.decker.pvz

# 重新安装
adb install -r build/pvz-unsigned.apk

# 查看详细错误
adb logcat | grep pvz
```

### iOS 构建（仅 macOS）

```bash
# 构建 iOS Framework
make build-ios

# 输出: build/ios/PVZ.xcframework
```

**注意**: iOS 构建仅支持在 macOS 系统上运行，需要 Xcode。

## 🎮 下一步

环境设置完成后，您可以：

1. **开始游戏** - 阅读 [用户手册](user-guide.md) 了解游戏操作
2. **查看代码** - 阅读 [开发指南](development.md) 了解项目架构
3. **运行测试** - 执行 `go test ./...` 验证代码质量
4. **查看 Epic** - 浏览 `docs/prd/` 目录了解功能实现

## 📚 相关文档

- **[README.md](../README.md)** - 项目概览
- **[用户手册](user-guide.md)** - 游戏操作说明
- **[开发指南](development.md)** - 代码贡献指引
- **[架构文档](architecture.md)** - 技术架构设计

## 🆘 寻求帮助

如果遇到未列出的问题：

1. 检查 [Issues](../../issues) 是否有类似问题
2. 搜索 [Discussions](../../discussions)
3. 提交新的 Issue（请附上错误日志）

---

**祝您游戏愉快！** 🌻🧟‍♂️
