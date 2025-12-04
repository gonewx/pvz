# Epic 20: Go Embed 资源嵌入与跨平台存储 (Go Embed & Cross-Platform Storage)

> **创建日期**: 2024-12-03
> **创建人**: Bob (Scrum Master)
> **Epic 类型**: Brownfield Enhancement (架构优化)
> **优先级**: High
> **预估总工作量**: 24-35 小时（约 4-6 天）
> **设计文档**: [docs/design/embed-and-cross-platform-storage.md](../design/embed-and-cross-platform-storage.md)

---

## 史诗目标

使用 Go 的 `embed` 功能将游戏资源（assets/ 和 data/）打包进可执行程序，实现单一可执行文件分发；使用 gdata 库实现跨平台用户数据存储（存档、设置），支持桌面端、移动端和 WASM 平台。

---

## Epic Overview (史诗概览)

### 背景与动机

当前项目存在以下问题：

1. **资源加载问题**：所有资源通过 `os.Open/os.ReadFile` 在运行时加载，无法打包成单一可执行文件
2. **存档存储问题**：存档目录 `data/saves/` 硬编码在项目中，不符合各平台规范
3. **缺少设置管理**：缺少游戏设置（音量、全屏等）的存储机制

### 解决方案

| 问题 | 解决方案 |
|------|----------|
| 资源打包 | Go `embed` 将 assets/ 和 data/（排除 saves/）嵌入可执行文件 |
| 跨平台存储 | 使用 `github.com/quasilyte/gdata/v2` 库 |
| 设置管理 | 新增 SettingsManager，存储音量、全屏等设置 |

### 技术选型

| 项目 | 选择 | 理由 |
|------|------|------|
| 存储库 | gdata v2 | 专为 Ebitengine 设计，跨平台支持完整 |
| 应用名称 | `pvz_newx` | 用户指定 |
| 数据格式 | YAML | 与项目其他配置保持一致 |

### 平台支持

| 平台 | 存储位置 |
|------|----------|
| **Windows** | `%APPDATA%/pvz_newx/` |
| **Linux** | `~/.local/share/pvz_newx/` |
| **macOS** | `~/.local/share/pvz_newx/` |
| **Android/iOS** | App data directory |
| **Browser/WASM** | localStorage |

---

## Stories (故事列表)

Epic 20 包含 **5 个 Story**：

### Story 20.1: gdata 依赖与 Manager 初始化

> **As a** 开发者,
> **I want** to integrate gdata library and initialize global Manager,
> **so that** the application has cross-platform storage capability.

**Acceptance Criteria:**
1. 添加 `github.com/quasilyte/gdata/v2` 依赖到 `go.mod`
2. 在 GameState 中创建全局 `gdata.Manager` 实例
3. 应用启动时初始化 Manager，使用应用名 `pvz_newx`
4. 处理初始化错误，提供降级方案
5. 单元测试覆盖率 ≥ 80%

**优先级**: ⭐⭐⭐⭐⭐ 高
**预估工作量**: 2-3 小时

---

### Story 20.2: SettingsManager 实现

> **As a** 玩家,
> **I want** my game settings (volume, fullscreen) to be saved,
> **so that** I don't need to reconfigure every time I start the game.

**Acceptance Criteria:**
1. 创建 `pkg/game/settings_manager.go`
2. 实现 `GameSettings` 结构（MusicVolume, SoundVolume, MusicEnabled, SoundEnabled, Fullscreen）
3. 实现 `SettingsManager` 类（Load, Save, GetSettings, SetXxx 方法）
4. 使用 gdata API 存储在 `settings/global` 路径
5. 提供 `DefaultSettings()` 返回默认值
6. 集成到 GameState
7. 单元测试覆盖率 ≥ 80%

**优先级**: ⭐⭐⭐⭐ 中高
**预估工作量**: 4-6 小时

---

### Story 20.3: SaveManager 重构 (gdata API)

> **As a** 开发者,
> **I want** to refactor SaveManager to use gdata API,
> **so that** user saves work across all platforms.

**Acceptance Criteria:**
1. 修改 `SaveManager` 构造函数，接收 `*gdata.Manager` 参数
2. 替换所有 `os.ReadFile/WriteFile` 为 gdata API
3. 用户数据存储在 `saves/` 路径下
4. 保持外部 API 不变（GetCurrentUser, SaveProgress 等）
5. 删除旧的 `data/saves/` 目录依赖（不需要向后兼容）
6. 更新 BattleSerializer 使用新路径
7. 单元测试覆盖率 ≥ 80%

**优先级**: ⭐⭐⭐⭐⭐ 高
**预估工作量**: 6-8 小时

---

### Story 20.4: Embed 实现 (资源加载重构)

> **As a** 开发者,
> **I want** to embed all game resources into the executable,
> **so that** the game can be distributed as a single file.

**Acceptance Criteria:**
1. 创建 `pkg/embedded/embed.go`，声明 embed.FS 变量
2. Embed 范围：`assets/` (44MB) + `data/`（排除 saves/，约 11MB）
3. 修改 `ResourceManager.LoadImage()` 从 embed.FS 加载
4. 修改 `ResourceManager.LoadAudio()` 从 embed.FS 加载
5. 修改所有 `pkg/config/*.go` 从 embed.FS 加载
6. 修改 `pkg/utils/bitmap_font.go` 从 embed.FS 加载
7. 修改 `pkg/game/lawn_strings.go` 从 embed.FS 加载
8. 游戏能正常运行，所有资源加载成功
9. 编译后生成单一可执行文件（约 55MB+）
10. 单元测试覆盖率 ≥ 80%

**优先级**: ⭐⭐⭐⭐⭐ 高
**预估工作量**: 8-12 小时

---

### Story 20.5: 暂停面板 UI (音量/全屏设置)

> **As a** 玩家,
> **I want** to adjust volume and fullscreen settings from the pause menu,
> **so that** I can customize my game experience.

**Acceptance Criteria:**
1. 在暂停菜单中添加音量滑块控件（音乐、音效各一个）
2. 在暂停菜单中添加全屏复选框
3. 滑块和复选框绑定 SettingsManager
4. 修改设置后实时生效（音量立即改变，全屏立即切换）
5. 退出暂停菜单时自动保存设置
6. UI 样式与原版 PvZ 一致
7. 单元测试覆盖率 ≥ 80%

**优先级**: ⭐⭐⭐ 中
**预估工作量**: 4-6 小时

---

## Success Criteria (成功标准)

1. ✅ 游戏编译后生成单一可执行文件
2. ✅ 可执行文件大小约 55-60MB（含所有资源）
3. ✅ 游戏设置（音量、全屏）在各平台正确保存和加载
4. ✅ 用户存档在各平台正确保存和加载
5. ✅ WASM 版本在浏览器中正常运行
6. ✅ 所有 Story 单元测试覆盖率 ≥ 80%
7. ✅ 性能无回归（60 FPS）
8. ✅ 无功能回归

---

## Technical Implementation (技术实现)

### 新增文件

| 文件 | 职责 |
|------|------|
| `pkg/embedded/embed.go` | embed.FS 声明 |
| `pkg/game/settings_manager.go` | GameSettings 管理 |

### 修改文件

| 文件 | 修改内容 |
|------|----------|
| `go.mod` | 添加 gdata 依赖 |
| `pkg/game/game_state.go` | 初始化 gdata.Manager、SettingsManager |
| `pkg/game/resource_manager.go` | 从 embed.FS 加载资源 |
| `pkg/game/save_manager.go` | 使用 gdata API |
| `pkg/game/battle_serializer.go` | 使用 gdata 路径 |
| `pkg/config/*.go` (6个) | 从 embed.FS 加载 |
| `pkg/utils/bitmap_font.go` | 从 embed.FS 加载 |
| `pkg/game/lawn_strings.go` | 从 embed.FS 加载 |
| `pkg/modules/pause_menu_module.go` | 添加音量/全屏 UI |

### 数据存储结构

```
gdata 存储（逻辑结构）
├── settings/
│   └── global              # 全局游戏设置
│
└── saves/
    ├── users               # 用户列表
    ├── {username}          # 用户进度存档
    └── {username}_battle   # 用户战斗存档
```

---

## Dependencies and Blockers (依赖与阻塞)

### 前置依赖

- ✅ 无硬性前置依赖

### 阻塞关系

- **Epic 18 (战斗存档)**: 建议 Story 20.3 先于 Epic 18 完成，Epic 18 可直接使用 gdata API

### 建议执行顺序

```
Story 20.1 (gdata 初始化)
    ↓
Story 20.2 (SettingsManager) ←─┐
    ↓                          │ 可并行
Story 20.3 (SaveManager 重构) ←┘
    ↓
Story 20.4 (Embed 实现)
    ↓
Story 20.5 (暂停面板 UI)
```

---

## Timeline Estimate (时间估算)

| Story | 预估工作量 | 优先级 | 依赖 |
|-------|-----------|--------|------|
| Story 20.1 | 2-3 小时 | 高 | 无 |
| Story 20.2 | 4-6 小时 | 中高 | 20.1 |
| Story 20.3 | 6-8 小时 | 高 | 20.1 |
| Story 20.4 | 8-12 小时 | 高 | 无 |
| Story 20.5 | 4-6 小时 | 中 | 20.2 |
| **总计** | **24-35 小时** | - | - |

**预估周期**: 4-6 天（单人开发）

---

## Reference Documentation (参考文档)

- **设计文档**: [docs/design/embed-and-cross-platform-storage.md](../design/embed-and-cross-platform-storage.md)
- **gdata 库**: [github.com/quasilyte/gdata](https://github.com/quasilyte/gdata)
- **Go embed 文档**: [pkg.go.dev/embed](https://pkg.go.dev/embed)
- **现有 SaveManager**: `pkg/game/save_manager.go`
- **现有 ResourceManager**: `pkg/game/resource_manager.go`

---

## 变更历史

| 日期 | 版本 | 变更内容 |
|------|------|----------|
| 2024-12-03 | 1.0 | 初始版本，基于 Sprint Change Proposal 创建 |
