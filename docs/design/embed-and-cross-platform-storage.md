# Go Embed 资源嵌入与跨平台存储设计方案

> **文档类型**: 技术设计方案
> **创建日期**: 2024-12-03
> **状态**: 待实现
> **优先级**: 高

---

## 1. 背景与目标

### 1.1 需求来源

项目需要实现以下目标：
1. 使用 Go 的 `embed` 功能将资源打包进可执行程序
2. 支持跨平台运行：桌面 (Windows/macOS/Linux)、移动端 (Android/iOS)、Web (WASM)
3. 用户存档和设置需要持久化存储在各平台的标准位置

### 1.2 当前问题

- 所有资源通过 `os.Open/ReadFile` 在运行时加载，无法打包成单一可执行文件
- 存档目录 `data/saves/` 硬编码在项目中，不符合各平台规范
- 缺少游戏设置（音量、全屏等）的存储机制

---

## 2. 目录结构分析

### 2.1 当前目录结构

```
pvz/
├── assets/                  # ~43MB - 游戏二进制资源
│   ├── reanim/             # 13M - 动画部件图片 (PNG)
│   ├── images/             # 6.6M - 界面图片
│   ├── sounds/             # 8M - 音效文件
│   ├── fonts/              # 字体文件
│   ├── config/             # 资源配置 (YAML)
│   ├── particles/          # 粒子图片资源
│   └── properties/         # 属性配置
│
├── data/                    # ~11MB - 数据定义文件
│   ├── reanim/             # 9.9M - .reanim 动画定义 (XML)
│   ├── reanim_config/      # 604K - 动画配置 (YAML)
│   ├── particles/          # 448K - 粒子特效定义 (XML) ← 从 assets/effect/ 迁移
│   ├── levels/             # 28K - 关卡配置 (YAML)
│   ├── saves/              # ⚠️ 用户存档 - 不应被 embed
│   └── *.yaml              # 其他配置文件
```

**设计原则**：
- `assets/` = 二进制资源（图片、音效、字体）
- `data/` = 数据定义文件（XML、YAML 配置）
- 粒子系统遵循与 reanim 相同的分离模式：
  - 定义文件 (XML) → `data/particles/`
  - 图片资源 (PNG) → `assets/particles/`

### 2.2 Embed 适用性评估

| 目录 | 大小 | 适合 Embed | 说明 |
|------|------|------------|------|
| `assets/` (全部) | ~43M | ✅ 是 | 二进制游戏资源 |
| `data/reanim/` | 9.9M | ✅ 是 | 动画定义文件 |
| `data/reanim_config/` | 604K | ✅ 是 | 动画配置 |
| `data/particles/` | 448K | ✅ 是 | 粒子特效定义 |
| `data/levels/` | 28K | ✅ 是 | 关卡配置 |
| `data/*.yaml` | ~10K | ✅ 是 | 配置文件 |
| `data/saves/` | 动态 | ❌ **否** | 运行时用户数据 |

**合计 Embed 大小**: 约 54MB

---

## 3. 跨平台存储方案

### 3.1 技术选型：`github.com/quasilyte/gdata`

经过调研，选择使用 **gdata** 库作为跨平台存储方案。

**选择理由**：
1. 专为 Ebitengine 游戏设计
2. 覆盖所有目标平台（桌面/移动/WASM）
3. 遵循各平台的数据存储规范
4. 简单的 key-value API
5. MIT 许可证，积极维护（2024年8月发布 v2.0.0）

**依赖信息**：
```
github.com/quasilyte/gdata/v2
```

### 3.2 各平台存储位置

| 平台 | 存储实现 | 路径/机制 |
|------|----------|-----------|
| **Windows** | 文件系统 | `%APPDATA%/pvz_newx/` |
| **Linux** | 文件系统 | `~/.local/share/pvz_newx/` |
| **macOS** | 文件系统 | `~/.local/share/pvz_newx/` |
| **Android** | 文件系统 | App data directory |
| **iOS** | 文件系统 | App data directory |
| **Browser/WASM** | localStorage | 浏览器本地存储 (限制 ~5MB) |

**应用名称**: `pvz_newx`

### 3.3 WASM 限制说明

- localStorage 存储上限约 **5MB**（实际更少，因 UTF-16 编码）
- 对于 PvZ 存档（用户数据 + 关卡进度），5MB 足够
- 如果未来需要更大存储，可考虑升级到 IndexedDB 或 OPFS

---

## 4. 数据存储结构设计

### 4.1 数据类型区分

| 类型 | 内容 | 特点 | XDG 规范目录 |
|------|------|------|--------------|
| **Save Data** | 游戏进度 | 用户生成，不可恢复 | `XDG_DATA_HOME` |
| **Settings** | 用户偏好 | 可重置为默认值 | `XDG_CONFIG_HOME` |

### 4.2 gdata 存储逻辑结构

```
gdata 存储（逻辑结构）
├── settings/
│   └── global              # 全局游戏设置（音量、全屏等）
│
└── saves/
    ├── users               # 用户列表 (UserListData)
    ├── {username}          # 用户进度存档 (SaveData)
    └── {username}_battle   # 用户战斗存档 (BattleSaveData)
```

### 4.3 物理存储位置示例

**Linux**:
```
~/.local/share/pvz_newx/
├── settings/
│   └── global
└── saves/
    ├── users
    ├── peter
    ├── peter_battle
    ├── alice
    └── alice_battle
```

**Windows**:
```
%APPDATA%/pvz_newx/
├── settings/
│   └── global
└── saves/
    └── ...
```

---

## 5. 数据结构定义

### 5.1 GameSettings - 全局游戏设置

```go
// pkg/game/settings.go

// GameSettings 全局游戏设置
// 注意：这些设置是全局的，不绑定到特定用户（与原版 PvZ 一致）
type GameSettings struct {
    // 音频设置
    MusicVolume  float64 `yaml:"musicVolume"`  // 音乐音量 0.0 ~ 1.0
    SoundVolume  float64 `yaml:"soundVolume"`  // 音效音量 0.0 ~ 1.0
    MusicEnabled bool    `yaml:"musicEnabled"` // 音乐开关
    SoundEnabled bool    `yaml:"soundEnabled"` // 音效开关

    // 显示设置
    Fullscreen bool `yaml:"fullscreen"` // 启动时是否全屏

    // 未来可扩展
    // Language   string `yaml:"language"`   // 语言设置
    // VSync      bool   `yaml:"vsync"`      // 垂直同步
    // ScreenMode string `yaml:"screenMode"` // 窗口/全屏/无边框
}

// DefaultSettings 返回默认设置
func DefaultSettings() *GameSettings {
    return &GameSettings{
        MusicVolume:  0.7,
        SoundVolume:  0.8,
        MusicEnabled: true,
        SoundEnabled: true,
        Fullscreen:   false,
    }
}
```

### 5.2 SaveData - 用户存档（已有，无需修改结构）

```go
// pkg/game/save_manager.go (已有)

type SaveData struct {
    HighestLevel   string   `yaml:"highestLevel"`
    UnlockedPlants []string `yaml:"unlockedPlants"`
    UnlockedTools  []string `yaml:"unlockedTools"`
    HasStartedGame bool     `yaml:"hasStartedGame"`
}
```

### 5.3 UserListData - 用户列表（已有，无需修改结构）

```go
// pkg/game/save_manager.go (已有)

type UserListData struct {
    Users       []UserMetadata `yaml:"users"`
    CurrentUser string         `yaml:"currentUser"`
}

type UserMetadata struct {
    Username    string    `yaml:"username"`
    CreatedAt   time.Time `yaml:"createdAt"`
    LastLoginAt time.Time `yaml:"lastLoginAt"`
}
```

---

## 6. API 设计

### 6.1 gdata 基础用法

```go
import "github.com/quasilyte/gdata/v2"

// 初始化（应用启动时调用一次）
manager, err := gdata.Open(gdata.Config{
    AppName: "pvz_newx",
})
if err != nil {
    log.Fatalf("Failed to open gdata: %v", err)
}

// 保存数据
err = manager.SaveObjectProp("saves", "peter", saveDataBytes)

// 加载数据
data, err := manager.LoadObjectProp("saves", "peter")

// 检查是否存在
exists := manager.ObjectPropExists("saves", "peter")

// 列出所有属性
props, err := manager.ListObjectProps("saves")
// 返回: ["users", "peter", "peter_battle", "alice", ...]

// 删除数据
err = manager.DeleteObjectProp("saves", "peter")
```

### 6.2 SettingsManager - 新增

```go
// pkg/game/settings_manager.go

type SettingsManager struct {
    gdataManager *gdata.Manager
    settings     *GameSettings
}

func NewSettingsManager(gdataManager *gdata.Manager) (*SettingsManager, error)
func (sm *SettingsManager) Load() error
func (sm *SettingsManager) Save() error
func (sm *SettingsManager) GetSettings() *GameSettings
func (sm *SettingsManager) SetMusicVolume(volume float64)
func (sm *SettingsManager) SetSoundVolume(volume float64)
func (sm *SettingsManager) SetFullscreen(enabled bool)
```

### 6.3 SaveManager - 重构

```go
// pkg/game/save_manager.go (重构)

type SaveManager struct {
    gdataManager *gdata.Manager  // 替换原来的 saveDir
    currentUser  string
    data         *SaveData
    userList     *UserListData
}

// 构造函数签名变更
func NewSaveManager(gdataManager *gdata.Manager) (*SaveManager, error)

// 内部方法变更（外部 API 保持不变）
// - 使用 gdataManager.SaveObjectProp() 替代 os.WriteFile()
// - 使用 gdataManager.LoadObjectProp() 替代 os.ReadFile()
```

---

## 7. Embed 实现方案

### 7.1 Embed 声明文件

```go
// pkg/embedded/embed.go

package embedded

import "embed"

//go:embed assets
var Assets embed.FS

//go:embed data/reanim data/reanim_config data/particles data/levels
//go:embed data/reanim_config.yaml data/spawn_rules.yaml
//go:embed data/zombie_physics.yaml data/zombie_stats.yaml
var Data embed.FS
```

**注意**: `data/saves/` 目录被排除在 embed 之外。

### 7.2 ResourceManager 修改

```go
// pkg/game/resource_manager.go

import (
    "github.com/decker502/pvz/pkg/embedded"
    "io/fs"
)

// LoadImage 修改为从 embed.FS 加载
func (rm *ResourceManager) LoadImage(path string) (*ebiten.Image, error) {
    // 检查缓存
    if cachedImage, exists := rm.imageCache[path]; exists {
        return cachedImage, nil
    }

    // 从 embed.FS 读取
    var file fs.File
    var err error

    if strings.HasPrefix(path, "assets/") {
        file, err = embedded.Assets.Open(path)
    } else if strings.HasPrefix(path, "data/") {
        file, err = embedded.Data.Open(path)
    } else {
        return nil, fmt.Errorf("unknown resource path: %s", path)
    }

    if err != nil {
        return nil, fmt.Errorf("failed to open embedded file %s: %w", path, err)
    }
    defer file.Close()

    // 解码图片...
}
```

---

## 8. 迁移计划

### 8.1 实现顺序

1. **Phase 1: 添加 gdata 依赖**
   - `go get github.com/quasilyte/gdata/v2`
   - 创建全局 gdata.Manager 初始化

2. **Phase 2: 实现 SettingsManager**
   - 创建 `pkg/game/settings_manager.go`
   - 实现 GameSettings 结构
   - 集成到 GameState

3. **Phase 3: 重构 SaveManager**
   - 修改 SaveManager 使用 gdata API
   - 保持外部 API 不变，仅修改内部实现
   - 添加数据迁移逻辑（可选：迁移旧存档）

4. **Phase 4: 实现 Embed**
   - 创建 `pkg/embedded/embed.go`
   - 修改 ResourceManager 从 embed.FS 加载
   - 修改所有资源加载代码

5. **Phase 5: 暂停面板 UI**
   - 实现音量滑块控件
   - 实现全屏复选框
   - 绑定 SettingsManager

### 8.2 向后兼容

- 首次启动时检测旧存档 `data/saves/`
- 自动迁移到新的 gdata 存储位置
- 迁移成功后可选择删除旧文件

---

## 9. 测试计划

### 9.1 单元测试

- [ ] SettingsManager 加载/保存测试
- [ ] SaveManager gdata 集成测试
- [ ] Embed 资源加载测试

### 9.2 平台测试

- [ ] Linux 桌面测试
- [ ] Windows 桌面测试
- [ ] macOS 桌面测试（如有条件）
- [ ] WASM 浏览器测试
- [ ] Android 测试（如有条件）

### 9.3 迁移测试

- [ ] 旧存档迁移测试
- [ ] 新安装测试（无旧数据）

---

## 10. 参考资料

- [gdata - Cross-platform game data storage](https://github.com/quasilyte/gdata)
- [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html)
- [Go embed 文档](https://pkg.go.dev/embed)
- [Ebitengine WebAssembly docs](https://ebitengine.org/en/documents/webassembly.html)
- [Ebitengine Mobile docs](https://github.com/hajimehoshi/ebiten/wiki/Mobile)
- [localStorage vs IndexedDB comparison](https://rxdb.info/articles/localstorage-indexeddb-cookies-opfs-sqlite-wasm.html)
- [Go proposal: os.UserDataDir](https://github.com/golang/go/issues/62382)

---

## 11. 决策记录

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 存储库 | gdata | 专为 Ebitengine 设计，跨平台支持完整 |
| 应用名称 | `pvz_newx` | 用户指定 |
| 设置范围 | 全局 | 与原版 PvZ 一致，设置不绑定用户 |
| Windows 目录 | APPDATA | gdata 默认行为 |
| 数据格式 | YAML | 与项目其他配置保持一致 |

---

## 12. 变更历史

| 日期 | 版本 | 变更内容 |
|------|------|----------|
| 2024-12-03 | 1.1 | 更新目录结构：粒子定义从 assets/effect/ 迁移到 data/particles/ |
| 2024-12-03 | 1.0 | 初始版本，完整设计方案 |
