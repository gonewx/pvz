# 用户管理系统实现文档

## 概述

用户管理系统允许多个玩家在同一台电脑上使用独立的游戏存档。系统提供完整的用户生命周期管理功能，包括创建、重命名、删除和切换用户。

---

## 产品级交互流程

### 1. 首次启动流程（零用户场景）

**用户体验目标**：新玩家第一次启动游戏时，通过友好的引导流程创建自己的游戏档案。

**交互步骤**：

1. **游戏启动** → 检测无任何用户存档
2. **主菜单加载** → 仅显示背景展开动画（不显示木牌和草地）
3. **自动弹出对话框** → 标题"新用户"，提示"请输入你的名字："
4. **玩家输入用户名** → 文本输入框支持字母、数字、空格
5. **点击"好"按钮**：
   - **验证通过** → 创建用户存档 → 关闭对话框 → 播放木牌和草地动画 → 显示用户名
   - **验证失败**（空用户名）→ 弹出提示"请输入你的名字，以创建新的用户档案"
6. **点击"取消"按钮** → 显示强制创建提示（必须创建用户才能进入游戏）

**实现思路**：

- **检测机制**：主菜单初始化时，查询用户列表文件，如果为空或不存在，标记为首次启动
- **动画控制**：首次启动时隐藏木牌和草叶子轨道，仅播放背景展开动画
- **强制创建**：禁用对话框的 ESC 关闭和外部点击关闭功能
- **状态转换**：用户创建成功后，取消隐藏木牌轨道，播放完整动画，更新主菜单状态

### 2. 正常启动流程（已有用户场景）

**用户体验目标**：老玩家启动游戏时，直接看到欢迎界面和上次使用的用户名。

**交互步骤**：

1. **游戏启动** → 加载上次登录的用户存档
2. **主菜单加载** → 播放完整开场动画（背景 + 木牌 + 草地）
3. **显示用户信息**：
   - 木牌左侧显示标题"欢迎回来，我的朋友！"
   - 木牌右侧显示当前用户名
   - 木牌底部显示提示"如果这不是你的存档，请点我"
4. **进入游戏** → 点击冒险模式按钮，使用当前用户存档

**实现思路**：

- **自动登录**：读取用户列表文件中的 `currentUser` 字段
- **存档加载**：加载对应用户的存档文件，初始化关卡进度和解锁状态
- **UI 渲染**：在木牌轨道上叠加渲染用户名文本

### 3. 用户管理交互流程

**用户体验目标**：玩家可以方便地在多个存档之间切换，或管理现有存档。

**交互步骤**：

1. **触发入口** → 鼠标悬停在木牌底部区域（woodsign2）
   - 木牌变为按下状态图片
   - 鼠标光标变为手形
2. **点击木牌** → 打开用户管理对话框
3. **对话框显示**：
   - 标题"你叫啥？"
   - 用户列表（黑色背景区域）：
     - 当前用户绿色高亮
     - 其他用户白色背景
     - 列表末尾显示"（建立一位新用户）"选项
   - 四个操作按钮：
     - 左上：重命名
     - 右上：删除
     - 左下：好
     - 右下：取消
4. **选择列表项** → 点击用户名，绿色高亮移动到该项
5. **操作按钮交互** → 见下面各子流程

**实现思路**：

- **悬停检测**：使用木牌轨道的四边形区域进行点击检测
- **图片切换**：悬停时替换 woodsign2 轨道的图片资源
- **列表渲染**：动态生成用户列表项，根据 `currentUser` 标记选中状态
- **状态同步**：列表选中项与按钮操作联动

### 4. 切换用户流程

**用户体验目标**：快速切换到其他玩家的存档，立即看到对应的游戏进度。

**交互步骤**：

1. **在用户列表中选择其他用户** → 绿色高亮移动到该用户
2. **点击"好"按钮**：
   - 如果选中用户 = 当前用户 → 直接关闭对话框（无操作）
   - 如果选中用户 ≠ 当前用户 → 执行切换流程：
     - 保存当前用户的游戏状态
     - 切换到新用户存档
     - 更新主菜单显示（关卡进度、解锁状态）
     - 关闭对话框
3. **主菜单刷新** → 显示新用户的游戏进度
4. **木牌更新** → 显示新用户名

**实现思路**：

- **存档切换**：先保存当前存档，再加载目标用户存档
- **状态更新**：重新读取关卡进度，更新按钮解锁状态
- **用户列表更新**：更新 `currentUser` 和 `lastLoginAt` 时间戳
- **UI 刷新**：清除旧状态，重新渲染木牌用户名和关卡数字

### 5. 新建用户流程

**用户体验目标**：快速创建新存档，支持家庭多人游戏场景。

**交互步骤**：

1. **在用户列表中点击"（建立一位新用户）"** → 弹出新建用户对话框
2. **对话框显示**：
   - 标题"新用户"
   - 描述"请输入你的名字："
   - 文本输入框（空白）
   - 按钮：好 | 取消
3. **输入用户名** → 实时显示输入内容，支持字母、数字、空格
4. **点击"好"按钮**：
   - **验证通过** → 创建新用户存档 → 关闭新建对话框 → 用户列表刷新（新用户出现在列表中）
   - **验证失败**（空用户名）→ 弹出错误提示对话框
   - **验证失败**（用户名已存在）→ 弹出错误提示对话框
5. **点击"取消"按钮** → 关闭新建对话框，返回用户管理对话框

**实现思路**：

- **对话框嵌套**：新建对话框叠加在用户管理对话框之上
- **输入验证**：实时验证用户名格式，提交时验证唯一性
- **列表刷新**：创建成功后，重新加载用户列表，新用户自动出现在列表中
- **错误处理**：使用独立的错误对话框，关闭后返回输入框

### 6. 重命名用户流程

**用户体验目标**：修正用户名错误或调整个性化昵称。

**交互步骤**：

1. **在用户列表中选择要重命名的用户** → 绿色高亮
2. **点击"重命名"按钮** → 弹出重命名对话框
3. **对话框显示**：
   - 标题"重命名用户"
   - 文本输入框（预填充当前用户名）
   - 按钮：好 | 取消
4. **修改用户名** → 光标自动定位到文本末尾
5. **点击"好"按钮**：
   - **验证通过** → 重命名存档文件 → 更新用户列表 → 关闭重命名对话框 → 刷新用户列表
   - **验证失败** → 弹出错误提示
6. **如果重命名的是当前用户** → 木牌上的用户名也同步更新

**实现思路**：

- **文件重命名**：同时重命名存档文件（oldname.yaml → newname.yaml）和用户列表元数据
- **预填充输入框**：初始化文本输入组件时设置默认值
- **原子操作**：先重命名文件，成功后再更新用户列表，确保数据一致性
- **UI 同步**：如果重命名当前用户，立即更新木牌显示

### 7. 删除用户流程

**用户体验目标**：安全地删除不需要的存档，防止误操作。

**交互步骤**：

1. **在用户列表中选择要删除的用户** → 绿色高亮
2. **点击"删除"按钮** → 弹出确认对话框
3. **确认对话框显示**：
   - 标题"你确定吗？"
   - 描述"从玩家簿中永久删除 '{username}'!"
   - 按钮：是 | 否
4. **点击"是"按钮**：
   - 删除存档文件
   - 从用户列表中移除
   - 关闭确认对话框
   - **场景 A：删除的是其他用户** → 返回用户管理对话框，列表刷新
   - **场景 B：删除的是当前用户** → 自动切换到其他用户，更新主菜单
   - **场景 C：删除最后一个用户** → 进入强制创建新用户流程（类似首次启动）
5. **点击"否"按钮** → 关闭确认对话框，返回用户管理对话框

**实现思路**：

- **二次确认**：使用独立的确认对话框，防止误删
- **级联删除**：同时删除存档文件和用户列表元数据
- **智能切换**：删除当前用户时，自动选择列表中的第一个其他用户
- **边界处理**：删除最后一个用户时，清空木牌显示，强制创建新用户

### 8. 木牌悬停交互

**用户体验目标**：提供清晰的视觉反馈，让玩家知道木牌是可点击的。

**交互步骤**：

1. **鼠标移动到木牌底部区域** → 检测鼠标是否进入 woodsign2 的四边形区域
2. **悬停状态激活**：
   - 木牌底部区域变为按下状态图片（更亮或有高光效果）
   - 鼠标光标从箭头变为手形
3. **鼠标移出区域** → 恢复正常状态图片，光标恢复为箭头
4. **点击木牌** → 打开用户管理对话框

**实现思路**：

- **区域检测**：使用 Reanim 轨道的变换矩阵计算四边形点击区域
- **图片切换**：动态替换 woodsign2 轨道的图片资源
- **光标管理**：使用 Ebiten 的光标 API 切换光标形状
- **状态管理**：使用布尔标志跟踪悬停状态，避免重复切换

### 9. 输入框交互细节

**用户体验目标**：提供流畅的文本输入体验，符合用户习惯。

**交互特性**：

1. **输入支持**：
   - 字母输入（大小写）
   - 数字输入（0-9）
   - 空格输入
   - 退格删除（Backspace）
2. **视觉反馈**：
   - 闪烁光标（每 0.5 秒切换显示/隐藏）
   - 光标位置始终在文本末尾
   - 输入框背景使用游戏原版 editbox.gif
3. **输入限制**：
   - 最大长度限制（防止超长用户名）
   - 禁止输入特殊字符（实时过滤）
4. **键盘导航**：
   - Enter 键等同于点击"好"按钮
   - ESC 键等同于点击"取消"按钮（首次启动除外）

**实现思路**：

- **事件驱动**：监听键盘事件，逐字符更新文本组件
- **字符过滤**：使用正则表达式验证输入字符，非法字符直接忽略
- **光标动画**：使用定时器控制光标闪烁状态
- **焦点管理**：对话框打开时自动激活输入框，接收键盘输入

### 10. 错误处理与用户反馈

**用户体验目标**：清晰地告知玩家错误原因，引导正确操作。

**错误场景与提示**：

1. **空用户名**：
   - 提示："请输入你的名字，以创建新的用户档案。档案用于保存游戏积分和进度。"
   - 对话框标题："输入你的名字"

2. **用户名已存在**：
   - 提示："用户名 '{username}' 已存在，请使用其他名字。"
   - 对话框标题："用户名冲突"

3. **非法字符**：
   - 实时过滤，不显示错误提示（用户输入时自动忽略）

4. **文件系统错误**：
   - 提示："存档文件操作失败：{错误详情}"
   - 对话框标题："系统错误"

**实现思路**：

- **验证时机**：提交时验证（点击"好"按钮时）
- **错误对话框**：使用独立的简单对话框显示错误信息
- **非阻塞设计**：错误对话框关闭后，保留输入内容，用户可继续修改
- **防止叠加**：同一时间只显示一个错误对话框

---

## 技术实现架构

### 整体设计原则

本系统遵循以下核心设计原则：

1. **ECS 零耦合原则**：所有系统通过 EntityManager 和组件通信，不直接调用其他系统
2. **数据驱动设计**：用户信息、对话框状态、UI 状态全部存储在组件中
3. **单一职责原则**：每个组件、系统、工厂函数只负责一个明确的功能
4. **复用优先原则**：复用现有的对话框基础设施，避免重复开发
5. **类型安全优先**：使用泛型 ECS API 和结构化回调，减少运行时错误

### 存档管理层

**职责边界**：
- SaveManager 负责文件系统操作和数据持久化
- 不负责 UI 渲染和用户交互
- 不依赖 ECS 框架

**关键设计**：
- **双文件结构**：用户列表（users.yaml）与用户存档（{username}.yaml）分离
- **原子操作保证**：先操作文件系统，成功后再更新内存数据
- **时间戳追踪**：记录创建时间和最后登录时间，支持未来功能扩展
- **验证前置**：在文件操作前完成所有验证，减少回滚复杂度

**错误处理策略**：
- 文件不存在：返回空列表（首次启动场景）
- 文件损坏：记录错误日志，返回错误给调用方
- 写入失败：保留旧数据，不修改内存状态
- 并发冲突：单线程游戏无需处理

### ECS 组件层

**组件职责分离**：

1. **UserSignComponent**：
   - 存储木牌 UI 状态（悬停、图片资源）
   - 不包含渲染逻辑（由 RenderSystem 处理）
   - 不包含用户数据（从 SaveManager 获取）

2. **UserListComponent**：
   - 存储对话框中的用户列表快照
   - 跟踪选中状态（索引）
   - 使用 UserInfo 而非 UserMetadata（避免循环依赖）

3. **TextInputComponent**：
   - 存储输入框状态（文本、光标位置）
   - 通用组件，可用于任何需要文本输入的场景
   - 不耦合用户管理逻辑

**避免循环依赖的设计**：
- `components` 包定义 UserInfo 结构体（轻量级）
- `game` 包定义 UserMetadata 结构体（完整元数据）
- 需要时进行类型转换（UserMetadata → UserInfo）

### 对话框工厂层

**工厂模式设计**：
- 每个对话框类型对应一个独立的工厂函数
- 工厂函数负责创建完整的实体树（对话框 + 子实体）
- 返回实体 ID，调用方负责生命周期管理

**回调机制设计**：
- 使用结构化回调（而非字符串 action）
- 回调参数包含所有必要信息（用户名、确认状态等）
- 回调执行时，对话框实体可能已销毁（通过实体 ID 通信）

**对话框嵌套处理**：
- 子对话框（如错误提示）不影响父对话框状态
- 使用 z-order 控制渲染顺序
- 独立的实体 ID 跟踪（currentDialog, currentErrorDialogID）

### 系统交互层

**TextInputSystem 设计**：
- **单一激活原则**：同一时间只有一个输入框接收输入
- **事件驱动模式**：监听键盘事件，更新组件状态
- **字符过滤机制**：使用白名单过滤，非法字符直接忽略
- **光标管理**：独立的定时器控制闪烁状态

**DialogInputSystem 扩展**：
- **列表选择支持**：点击列表项更新选中索引
- **按钮回调触发**：点击按钮时执行对应的回调函数
- **焦点管理**：对话框打开时自动激活输入框

**DialogRenderSystem 扩展**：
- **用户列表渲染**：动态计算列表项位置和背景颜色
- **文本居中渲染**：使用 text/v2 API 实现多语言支持
- **z-order 管理**：子对话框在父对话框之上

### 主菜单集成层

**初始化时机控制**：
- **场景创建时**：检测首次启动，设置动画隐藏状态
- **第一帧更新前**：显示新建用户对话框（如果首次启动）
- **用户创建成功后**：播放木牌动画，初始化用户 UI

**状态转换管理**：
```
首次启动状态：
- isFirstLaunch = true
- 隐藏木牌轨道（woodsign1/2/3）
- 隐藏草叶子轨道
- 显示新建用户对话框

↓ 用户创建成功

正常状态：
- isFirstLaunch = false
- 取消隐藏木牌轨道
- 播放木牌和草动画
- 初始化用户木牌 UI
```

**木牌 UI 实现细节**：
- **图片替换策略**：动态修改 Reanim 轨道的图片资源
- **文本渲染位置**：基于轨道的变换矩阵计算屏幕坐标
- **悬停检测范围**：使用 woodsign2 轨道的四边形区域
- **光标切换时机**：悬停进入/退出时同步更新

**输入状态初始化**（防止误触发）：
```
问题：场景切换时，鼠标可能仍处于按下状态
解决：初始化时读取当前输入状态，而非默认值 false
好处：防止第一帧产生虚假的边缘触发事件
```

### 数据流设计

**用户创建流程**：
```
用户输入用户名
    ↓
TextInputComponent.Text 更新
    ↓
点击"好"按钮 → 触发回调
    ↓
回调验证用户名 → SaveManager.ValidateUsername()
    ↓
验证通过 → SaveManager.CreateUser()
    ↓
文件系统操作 → 创建存档文件 + 更新用户列表
    ↓
回调通知主菜单 → onNewUserCreated()
    ↓
主菜单更新状态 → 重新加载存档 + 刷新 UI
```

**用户切换流程**：
```
选择列表项 → UserListComponent.SelectedIndex 更新
    ↓
点击"好"按钮 → 读取选中用户名
    ↓
调用 SaveManager.SwitchUser(newUsername)
    ↓
保存当前用户存档 → 切换到新用户 → 加载新存档
    ↓
更新 users.yaml 中的 currentUser
    ↓
主菜单刷新 → 重新读取关卡进度 + 更新按钮状态
    ↓
木牌 UI 更新 → 替换用户名文本
```

**删除用户流程（复杂场景）**：
```
场景 A：删除其他用户
    → 删除存档文件
    → 更新用户列表
    → 刷新对话框

场景 B：删除当前用户（还有其他用户）
    → 删除存档文件
    → 自动切换到第一个其他用户
    → 加载新用户存档
    → 更新主菜单

场景 C：删除最后一个用户
    → 删除存档文件
    → 清空用户列表
    → 关闭用户管理对话框
    → 清空木牌显示
    → 标记 isFirstLaunch = true
    → 弹出新建用户对话框（强制创建）
```

### 性能优化策略

**减少文件 I/O**：
- 用户列表缓存在内存中，只在需要时刷新
- 存档切换时才执行文件读写
- 使用增量更新而非全量重写

**渲染优化**：
- 用户列表项懒加载（未来扩展）
- 文本渲染结果缓存（避免每帧重新计算）
- 对话框背景预渲染（九宫格拉伸结果缓存）

**内存管理**：
- 对话框关闭时立即销毁实体
- 输入框组件复用（不创建新实体）
- 图片资源共享（不重复加载）

### 测试策略

**单元测试覆盖**：
- SaveManager 的所有公开方法
- 用户名验证逻辑的边界条件
- 文件系统错误处理

**集成测试场景**：
- 首次启动 → 创建用户 → 进入游戏
- 切换用户 → 验证存档加载
- 删除用户 → 验证文件删除
- 重命名用户 → 验证文件重命名

**边界测试**：
- 空用户名
- 超长用户名
- 特殊字符
- 重复用户名
- 删除最后一个用户
- 文件系统错误（权限、磁盘满）

### 可扩展性设计

**未来功能扩展点**：

1. **用户头像系统**：
   - UserMetadata 增加 avatarID 字段
   - 木牌 UI 增加头像渲染
   - 用户管理对话框增加头像选择器

2. **游戏统计**：
   - UserMetadata 增加游戏时长、胜率等统计字段
   - 用户列表显示统计信息
   - 支持按统计排序

3. **云存档同步**：
   - 增加远程存储适配器
   - 冲突解决策略
   - 离线模式支持

4. **用户权限管理**：
   - 增加管理员用户概念
   - 支持用户锁定和解锁
   - 家长控制功能

**扩展原则**：
- 向后兼容旧版存档格式
- 新功能通过配置开关控制
- 保持核心逻辑简洁，扩展功能模块化

---

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
