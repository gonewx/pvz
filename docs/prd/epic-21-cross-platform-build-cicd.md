# Epic 21: 跨平台构建与 CI/CD (Cross-Platform Build & CI/CD)

> **创建日期**: 2025-12-04
> **创建人**: Sarah (Product Owner)
> **Epic 类型**: Brownfield Enhancement (运维增强)
> **优先级**: High
> **预估总工作量**: 16-24 小时（约 2-4 天）

---

## 史诗目标

为《植物大战僵尸》复刻版创建完整的跨平台构建系统和 CI/CD 流水线，支持 Linux、Windows、macOS、Android、iOS 和 Web (WASM) 六大平台的自动化构建与发布。

---

## Epic Overview (史诗概览)

### 背景与动机

当前项目存在以下问题：

1. **缺少构建脚本**：没有统一的构建脚本，开发者需要手动执行构建命令
2. **无 CI/CD 流水线**：没有 GitHub Actions 工作流，无法自动化测试和构建
3. **多平台发布困难**：缺少跨平台编译的标准化流程，难以发布到多个平台

### 现有系统上下文

| 项目 | 详情 |
|------|------|
| **技术栈** | Go 1.25.1 + Ebitengine v2.9.0 |
| **资源嵌入** | Epic 20 已实现 `embed` 打包，可生成单一可执行文件 |
| **模块路径** | `github.com/decker502/pvz` |
| **依赖管理** | Go Modules |

### 解决方案

| 问题 | 解决方案 |
|------|----------|
| 缺少构建脚本 | 创建 Makefile，统一管理所有平台构建命令 |
| 无 CI/CD | 创建 GitHub Actions 工作流，自动化构建与发布 |
| 多平台发布 | 使用 Go 交叉编译 + ebitenmobile 工具 |

### 平台支持矩阵

| 平台 | GOOS/GOARCH | 工具 | 输出格式 |
|------|-------------|------|----------|
| **Linux (amd64)** | `linux/amd64` | go build | 可执行文件 |
| **Linux (arm64)** | `linux/arm64` | go build | 可执行文件 |
| **Windows (amd64)** | `windows/amd64` | go build | `.exe` |
| **Windows (arm64)** | `windows/arm64` | go build | `.exe` |
| **macOS (amd64)** | `darwin/amd64` | go build + CGO | 可执行文件 |
| **macOS (arm64)** | `darwin/arm64` | go build + CGO | 可执行文件 |
| **macOS (Universal)** | - | lipo | Universal Binary |
| **Web (WASM)** | `js/wasm` | go build | `.wasm` + HTML |
| **Android** | - | ebitenmobile | `.aar` |
| **iOS** | - | ebitenmobile | `.xcframework` |

---

## Stories (故事列表)

Epic 21 包含 **3 个 Story**：

### Story 21.1: 本地构建脚本 (Makefile)

> **As a** 开发者,
> **I want** a unified Makefile for building the game,
> **so that** I can easily build for any platform with a single command.

**范围**：
- 创建 `Makefile`，包含所有构建目标
- 支持桌面平台：Linux (amd64/arm64), Windows (amd64/arm64), macOS (amd64/arm64/universal)
- 支持 Web 平台：WASM 编译 + HTML 模板
- 提供 `make help` 显示所有可用命令
- 提供 `make clean` 清理构建产物
- 提供 `make all` 构建所有桌面平台

**Acceptance Criteria:**
1. `make build` 构建当前平台可执行文件
2. `make build-linux` 构建 Linux amd64/arm64
3. `make build-windows` 构建 Windows amd64/arm64
4. `make build-darwin` 构建 macOS amd64/arm64 并生成 Universal Binary
5. `make build-wasm` 构建 WASM 并生成可运行的 HTML 页面
6. `make all` 构建所有桌面平台
7. `make clean` 清理 `build/` 目录
8. `make help` 显示帮助信息
9. 构建产物输出到 `build/{platform}/` 目录

**优先级**: ⭐⭐⭐⭐⭐ 高
**预估工作量**: 4-6 小时

---

### Story 21.2: GitHub Actions CI/CD (桌面与 Web)

> **As a** 开发者,
> **I want** GitHub Actions to automatically build and release the game,
> **so that** every release is consistently built for all platforms.

**范围**：
- 创建 `.github/workflows/ci.yml` 用于 PR 检查（测试、lint）
- 创建 `.github/workflows/release.yml` 用于发布构建
- 支持 tag 触发自动发布
- 自动上传构建产物到 GitHub Releases
- 支持平台：Linux, Windows, macOS, Web (WASM)

**Acceptance Criteria:**
1. PR 提交时自动运行 `go test` 和 `go vet`
2. 推送 tag (如 `v1.0.0`) 时触发 Release 工作流
3. Release 工作流在 3 个 runner 上并行构建：
   - `ubuntu-latest`: Linux amd64/arm64, Windows (交叉编译), WASM
   - `macos-latest`: macOS amd64/arm64 + Universal Binary
4. 构建产物自动打包为 `.zip` 或 `.tar.gz`
5. 构建产物自动上传到 GitHub Releases
6. WASM 构建包含 `index.html` 和 `wasm_exec.js`
7. 工作流状态徽章可嵌入 README

**优先级**: ⭐⭐⭐⭐⭐ 高
**预估工作量**: 6-10 小时

---

### Story 21.3: 移动端构建 (ebitenmobile)

> **As a** 开发者,
> **I want** to build Android and iOS packages,
> **so that** the game can be distributed on mobile app stores.

**范围**：
- 创建移动端入口文件 `mobile/mobile.go`
- 集成 `ebitenmobile` 工具
- Android: 生成 `.aar` 库文件
- iOS: 生成 `.xcframework` 框架
- 更新 Makefile 添加移动端构建目标
- 可选：GitHub Actions 移动端构建

**Acceptance Criteria:**
1. 创建 `mobile/mobile.go` 导出游戏入口
2. `make build-android` 生成 `build/android/pvz.aar`
3. `make build-ios` 生成 `build/ios/PVZ.xcframework`
4. 提供 Android 示例项目结构说明
5. 提供 iOS 示例项目结构说明
6. 文档说明如何将 `.aar` 和 `.xcframework` 集成到原生项目

**优先级**: ⭐⭐⭐ 中
**预估工作量**: 6-8 小时

---

## Compatibility Requirements (兼容性要求)

- [x] 不修改现有游戏代码逻辑
- [x] 不影响现有 `go build` 命令
- [x] 构建产物与 Epic 20 的 embed 方案兼容
- [x] 支持 Go 1.21+ (项目使用 1.25.1)

---

## Risk Mitigation (风险缓解)

| 风险 | 缓解措施 |
|------|----------|
| macOS 交叉编译需要 CGO | 使用 macOS runner 原生构建 |
| 移动端需要 SDK 环境 | 使用 GitHub Actions 托管的 Android SDK 和 Xcode |
| WASM 运行时兼容性 | 使用 Go 官方 `wasm_exec.js` |

**回滚计划**: 删除 `.github/workflows/` 和 `Makefile`，无任何副作用

---

## Definition of Done (完成定义)

- [ ] Makefile 包含所有平台构建目标
- [ ] GitHub Actions CI 工作流正常运行
- [ ] GitHub Actions Release 工作流可自动发布
- [ ] 所有桌面平台构建产物可正常运行
- [ ] WASM 版本可在浏览器中运行
- [ ] 移动端库文件可正常生成
- [ ] README 更新构建说明和徽章

---

## Technical References (技术参考)

### Ebitengine 跨平台编译命令

```bash
# 桌面平台
GOOS=linux GOARCH=amd64 go build -o pvz-linux-amd64
GOOS=windows GOARCH=amd64 go build -o pvz-windows-amd64.exe
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o pvz-darwin-amd64
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o pvz-darwin-arm64
lipo -create pvz-darwin-amd64 pvz-darwin-arm64 -output pvz-darwin-universal

# Web (WASM)
GOOS=js GOARCH=wasm go build -o pvz.wasm

# 移动端
go install github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@latest
ebitenmobile bind -target android -javapkg com.decker.pvz -o pvz.aar ./mobile
ebitenmobile bind -target ios -o PVZ.xcframework ./mobile
```

### Linux 构建依赖

```bash
# 32-bit 支持 (可选)
sudo apt-get install gcc-multilib
sudo apt-get install libasound2-dev libgl1-mesa-dev libxcursor-dev \
  libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev
```

---

## Dependencies (依赖)

### 前置依赖

- ✅ Epic 20 (资源嵌入) - 已完成，游戏可生成单一可执行文件

### 工具依赖

| 工具 | 用途 | 安装方式 |
|------|------|----------|
| Go 1.21+ | 编译器 | 官方安装 |
| Make | 构建工具 | 系统自带 |
| ebitenmobile | 移动端构建 | `go install` |
| lipo | macOS Universal Binary | Xcode 自带 |

---

## 建议执行顺序

```
Story 21.1 (Makefile)
    ↓
Story 21.2 (GitHub Actions CI/CD)
    ↓
Story 21.3 (移动端构建)
```

---

## 变更历史

| 日期 | 版本 | 变更内容 |
|------|------|----------|
| 2025-12-04 | 1.0 | 初始版本，基于用户需求创建 |
