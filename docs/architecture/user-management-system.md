# 用户管理系统实现文档

## 概述

用户管理系统允许多个玩家在同一台电脑上使用独立的游戏存档。系统提供完整的用户生命周期管理功能，包括创建、重命名、删除和切换用户。

---

## 核心架构

### 存档文件结构

```
data/saves/
├── users.yaml              # 用户列表元数据
├── player1.yaml            # 用户1的存档
├── player2.yaml            # 用户2的存档
└── ...
```

### users.yaml 格式

```yaml
users:
  - username: "player1"
    createdAt: "2025-11-17T10:00:00Z"
    lastLoginAt: "2025-11-17T12:00:00Z"
  - username: "player2"
    createdAt: "2025-11-16T15:00:00Z"
    lastLoginAt: "2025-11-17T11:00:00Z"
currentUser: "player1"  # 上次登录的用户
```

### 用户存档格式

```yaml
highestLevel: "1-4"
unlockedPlants: ["peashooter", "sunflower", "wallnut"]
unlockedTools: ["shovel"]
hasStartedGame: true
```

---

## SaveManager 多用户 API

### 数据结构

```go
// UserMetadata 用户元数据
type UserMetadata struct {
    Username    string    `yaml:"username"`
    CreatedAt   time.Time `yaml:"createdAt"`
    LastLoginAt time.Time `yaml:"lastLoginAt"`
}

// UserListData 用户列表数据
type UserListData struct {
    Users       []UserMetadata `yaml:"users"`
    CurrentUser string         `yaml:"currentUser"`
}
```

### 核心方法

| 方法 | 功能 | 返回值 |
|------|------|--------|
| `LoadUserList()` | 加载所有用户元数据 | `([]UserMetadata, error)` |
| `GetCurrentUser()` | 获取当前用户名 | `string` |
| `ValidateUsername(username string)` | 验证用户名合法性 | `error` |
| `CreateUser(username string)` | 创建新用户 | `error` |
| `RenameUser(oldName, newName string)` | 重命名用户 | `error` |
| `DeleteUser(username string)` | 删除用户 | `error` |
| `SwitchUser(username string)` | 切换当前用户 | `error` |

### 用户名验证规则

- 允许：英文字母（大小写）、数字、空格
- 禁止：空用户名、特殊字符、下划线
- 正则表达式：`^[a-zA-Z0-9 ]+$`

---

## ECS 组件

### UserSignComponent（木牌组件）

**位置**: `pkg/components/user_sign_component.go`

```go
type UserSignComponent struct {
    CurrentUsername string        // 当前用户名
    IsHovered       bool          // 是否悬停在 woodsign2 区域
    SignNormalImage *ebiten.Image // 正常状态图片
    SignPressImage  *ebiten.Image // 按下状态图片
}
```

**用途**:
- 存储木牌 UI 的状态和图片资源
- 控制悬停效果显示

### UserListComponent（用户列表组件）

**位置**: `pkg/components/user_list_component.go`

```go
type UserListComponent struct {
    Users         []UserInfo // 用户列表（使用 UserInfo 避免循环依赖）
    SelectedIndex int        // 当前选中索引
    CurrentUser   string     // 当前用户名
}

type UserInfo struct {
    Username    string
    CreatedAt   time.Time
    LastLoginAt time.Time
}
```

**用途**:
- 存储用户管理对话框中的用户列表
- 跟踪选中状态

### TextInputComponent（文本输入组件）

**位置**: `pkg/components/text_input_component.go`

```go
type TextInputComponent struct {
    Text          string        // 当前输入文本
    CursorPos     int           // 光标位置
    IsActive      bool          // 是否激活（接收输入）
    MaxLength     int           // 最大长度
    BackgroundImg *ebiten.Image // 输入框背景图片
}
```

**用途**:
- 管理文本输入状态
- 渲染输入框 UI

---

## 对话框工厂

### 用户管理对话框

**位置**: `pkg/entities/user_management_dialog_factory.go`

```go
func NewUserManagementDialogEntity(
    em *ecs.EntityManager,
    rm *game.ResourceManager,
    users []UserInfo,
    currentUser string,
    callback func(UserManagementDialogResult),
    windowWidth, windowHeight int,
) (ecs.EntityID, error)
```

**功能**:
- 显示用户列表
- 提供重命名/删除/切换操作
- 当前用户绿色高亮

### 新建用户对话框

**位置**: `pkg/entities/new_user_dialog_factory.go`

```go
func NewNewUserDialogEntity(
    em *ecs.EntityManager,
    rm *game.ResourceManager,
    windowWidth, windowHeight int,
    callback func(NewUserDialogResult),
) (dialogID ecs.EntityID, inputBoxID ecs.EntityID, error)
```

**功能**:
- 显示用户名输入框
- 验证用户名合法性
- 返回对话框和输入框两个实体 ID

### 重命名用户对话框

**位置**: `pkg/entities/rename_user_dialog_factory.go`

```go
func NewRenameUserDialogEntity(
    em *ecs.EntityManager,
    rm *game.ResourceManager,
    oldUsername string,
    windowWidth, windowHeight int,
    callback func(RenameUserDialogResult),
) (dialogID ecs.EntityID, inputBoxID ecs.EntityID, error)
```

**功能**:
- 输入框预填充当前用户名
- 验证新用户名合法性

### 删除用户确认对话框

**位置**: `pkg/entities/delete_user_dialog_factory.go`

```go
func NewDeleteUserDialogEntity(
    em *ecs.EntityManager,
    rm *game.ResourceManager,
    username string,
    windowWidth, windowHeight int,
    callback func(confirmed bool),
) (ecs.EntityID, error)
```

**功能**:
- 显示确认信息
- 二选一按钮（是/否）

---

## 系统实现

### TextInputSystem（文本输入系统）

**位置**: `pkg/systems/text_input_system.go`

**职责**:
- 处理键盘输入（字母、数字、空格、退格）
- 更新 TextInputComponent.Text
- 管理光标位置

**更新逻辑**:
```go
func (s *TextInputSystem) Update(deltaTime float64) {
    // 1. 查询所有激活的 TextInputComponent
    // 2. 处理键盘输入
    // 3. 更新文本和光标位置
}
```

### TextInputRenderSystem（文本输入渲染系统）

**位置**: `pkg/systems/text_input_render_system.go`

**职责**:
- 渲染输入框背景
- 渲染文本内容
- 渲染闪烁光标（每 0.5 秒切换）

---

## MainMenuScene 集成

### 初始化流程

```go
func NewMainMenuScene(rm *game.ResourceManager, sm *game.SceneManager) *MainMenuScene {
    // 1. 检测首次启动
    users, err := saveManager.LoadUserList()
    if err != nil || len(users) == 0 {
        scene.isFirstLaunch = true
        // 仅播放 anim_open，隐藏木牌和草叶子
    } else {
        scene.isFirstLaunch = false
        // 播放完整开场动画
    }

    // 2. 初始化用户木牌（如果非首次启动）
    if !scene.isFirstLaunch {
        scene.initUserSign()
    }

    // 3. 初始化输入状态（防止场景切换时误触发点击）
    scene.wasMousePressed = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
    scene.wasF1Pressed = ebiten.IsKeyPressed(ebiten.KeyF1)
    scene.wasOPressed = ebiten.IsKeyPressed(ebiten.KeyO)
}
```

### 木牌 UI 实现

**初始化**: `MainMenuScene.initUserSign()`

1. 创建木牌实体
2. 替换 woodsign1 轨道图片为带用户名的版本
3. 加载悬停状态图片（SelectorScreen_WoodSign2_press）
4. 添加 UserSignComponent

**悬停检测**: `MainMenuScene.Update()`

1. 获取 woodsign2 轨道的四边形区域
2. 检测鼠标是否在区域内
3. 切换图片状态

**用户名渲染**: `MainMenuScene.Draw()`

1. 获取 woodsign1 轨道位置
2. 使用 text/v2 API 渲染用户名
3. 居中对齐

### 首次启动流程

```go
func (m *MainMenuScene) Update(deltaTime float64) {
    // 首次启动时显示新建用户对话框
    if m.isFirstLaunch && !m.newUserDialogShown {
        m.showNewUserDialogForFirstLaunch()
        m.newUserDialogShown = true
    }
}
```

**首次启动特殊处理**:
- 隐藏木牌轨道（woodsign1/2/3）
- 隐藏草叶子轨道
- 用户创建成功后，播放木牌和草动画
- 不可跳过对话框

### 用户切换回调

```go
func (m *MainMenuScene) onNewUserCreated(username string) {
    // 1. 重新加载存档
    m.saveManager.Load()
    m.currentLevel = m.saveManager.GetHighestLevel()
    m.hasStartedGame = m.saveManager.GetHasStartedGame()

    // 2. 清除首次启动标记
    m.isFirstLaunch = false

    // 3. 取消隐藏木牌和草轨道
    // 4. 播放木牌动画
    // 5. 初始化用户木牌 UI
    // 6. 更新按钮可见性
}
```

---

## 关键设计决策

### 1. 零耦合原则

**问题**: `components` 包不能引用 `game` 包（避免循环依赖）

**解决方案**:
- 在 `components` 包中定义 `UserInfo` 结构体
- 在 `game` 包中定义 `UserMetadata` 结构体
- 需要时进行转换

### 2. 复用对话框基础设施

**复用内容**:
- `DialogComponent` + 九宫格渲染
- `loadDialogParts()` 加载对话框资源
- `DialogInputSystem` 处理按钮点击

### 3. 类型安全的回调

**使用回调结构体而非 action string**:

```go
type UserManagementDialogResult struct {
    Action       string // "rename", "delete", "switch", "newUser", "cancel"
    Username     string // 选中的用户名
}

type NewUserDialogResult struct {
    Confirmed bool   // 是否确认创建
    Username  string // 新用户名
}
```

### 4. 边缘触发防护

**问题**: 场景切换时鼠标仍按下，导致误触发点击

**解决方案**:
```go
// 在 NewMainMenuScene 初始化时设置当前输入状态
scene.wasMousePressed = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
```

---

## 文件清单

### 新增文件

| 文件路径 | 描述 |
|----------|------|
| `pkg/components/user_sign_component.go` | 木牌 UI 组件 |
| `pkg/components/user_list_component.go` | 用户列表组件 |
| `pkg/entities/user_management_dialog_factory.go` | 用户管理对话框工厂 |
| `pkg/entities/rename_user_dialog_factory.go` | 重命名对话框工厂 |
| `pkg/entities/delete_user_dialog_factory.go` | 删除确认对话框工厂 |

### 修改文件

| 文件路径 | 修改内容 |
|----------|----------|
| `pkg/game/save_manager.go` | 扩展多用户支持（18个新方法） |
| `pkg/game/save_manager_test.go` | 18个新测试用例 |
| `pkg/scenes/main_menu_scene.go` | 木牌 UI + 用户切换 + 首次启动流程 |
| `pkg/systems/dialog_render_system.go` | 用户列表渲染支持 |
| `pkg/systems/dialog_input_system.go` | 用户列表选择支持 |
| `CLAUDE.md` | 添加多用户存档文档 |

---

## 测试覆盖

### SaveManager 测试

| 测试函数 | 覆盖场景 |
|----------|----------|
| `TestLoadUserList_Empty` | 空用户列表 |
| `TestLoadUserList_WithUsers` | 多用户加载 |
| `TestCreateUser` | 正常创建用户 |
| `TestCreateUser_Duplicate` | 重复用户名 |
| `TestCreateUser_Invalid` | 非法用户名 |
| `TestRenameUser` | 正常重命名 |
| `TestDeleteUser` | 正常删除 |
| `TestDeleteUser_Current` | 删除当前用户 |
| `TestSwitchUser` | 用户切换 |
| `TestValidateUsername` | 用户名验证规则 |

### 集成测试场景

1. 首次启动 → 创建用户 → 进入游戏
2. 切换用户 → 加载不同存档 → 验证关卡进度
3. 重命名用户 → 验证文件系统更新
4. 删除用户 → 验证文件删除
5. 删除最后一个用户 → 强制创建新用户

---

## 性能指标

| 操作 | 预期耗时 |
|------|----------|
| 用户列表加载 | < 50ms |
| 用户切换 | < 200ms |
| 对话框打开/关闭 | 60 FPS 流畅 |

---

## 已知问题与修复

### 问题：场景切换时误触发按钮点击

**症状**: 从暂停菜单返回主菜单时，立即弹出"未解锁"对话框

**原因**: `wasMousePressed` 默认值为 `false`，导致第一帧产生虚假边缘触发

**修复**:
```go
// pkg/scenes/main_menu_scene.go:106-115
scene := &MainMenuScene{
    wasMousePressed: ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft),
    wasF1Pressed:    ebiten.IsKeyPressed(ebiten.KeyF1),
    wasOPressed:     ebiten.IsKeyPressed(ebiten.KeyO),
}
```

**提交**: `5474d44 fix: 修复从游戏返回主菜单时误触发按钮点击的问题`

---

## 参考资料

- Story 12.4: `docs/stories/12.4.story.md`
- 编码标准: `docs/architecture/coding-standards.md`
- ECS 架构: `CLAUDE.md#泛型-ecs-api-使用指南`
