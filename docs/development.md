# 开发指南

欢迎为《植物大战僵尸 Go 复刻版》贡献代码！本指南将帮助您了解项目架构、开发流程和编码规范。

## 📖 目录

1. [开发环境设置](#开发环境设置)
2. [项目架构](#项目架构)
3. [代码贡献流程](#代码贡献流程)
4. [编码规范](#编码规范)
5. [测试指南](#测试指南)
6. [调试技巧](#调试技巧)
7. [常见开发任务](#常见开发任务)

---

## 🛠️ 开发环境设置

### 必需工具

| 工具 | 版本要求 | 用途 |
|------|---------|------|
| **Go** | 1.21+ | 编程语言 |
| **Git** | 2.0+ | 版本控制 |
| **IDE** | - | 代码编辑（推荐 VSCode 或 GoLand） |

### 推荐 IDE 配置

#### VSCode

**推荐扩展**:
```json
{
  "recommendations": [
    "golang.go",           // Go 官方扩展
    "eamodio.gitlens",     // Git 增强
    "streetsidesoftware.code-spell-checker" // 拼写检查
  ]
}
```

**settings.json**:
```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "go.formatTool": "gofmt",
  "go.testFlags": ["-v"],
  "editor.formatOnSave": true
}
```

#### GoLand

- 启用 Go Modules 支持
- 配置代码格式化为 `gofmt`
- 启用自动导入优化

### 克隆仓库并设置

```bash
# 1. Fork 仓库到您的 GitHub 账号

# 2. 克隆您的 Fork
git clone https://github.com/YOUR_USERNAME/pvz3.git
cd pvz3

# 3. 添加上游仓库
git remote add upstream https://github.com/ORIGINAL_REPO/pvz3.git

# 4. 下载依赖
go mod download

# 5. 验证构建
go build .

# 6. 运行测试
go test ./...
```

### 构建工具

项目使用 Makefile 管理构建流程，运行 `make help` 查看所有可用命令：

```bash
make help           # 显示帮助信息
make build          # 构建当前平台
make build-linux    # 构建 Linux 版本
make build-windows  # 构建 Windows 版本
make build-darwin   # 构建 macOS 版本 (需要 macOS)
make build-wasm     # 构建 WebAssembly 版本
```

### 图标和资源

项目图标位于 `assets/icons/` 目录，包含各平台所需的图标格式：

```bash
# 生成 Windows .syso 图标资源
make generate-icons

# 打包 Linux 发布包（含图标和 .desktop 文件）
make package-linux

# 构建 macOS .app 包 (需要 macOS)
make build-darwin-app

# 构建 Android APK
make build-apk

# 查看 iOS 图标使用说明
make ios-icons-info
```

**图标目录结构**：
```
assets/icons/
├── windows/        # Windows ico 和多尺寸 png
├── macos/          # macOS icon.iconset (可转换为 .icns)
├── linux/          # Linux 多尺寸 png
├── ios/            # iOS AppIcon.appiconset
├── android/        # Android mipmap 图标
└── web/            # Web favicon 和 PWA 图标
```

---

## 🏗️ 项目架构

### 架构模式：ECS (Entity-Component-System)

本项目采用 **Entity-Component-System** 架构，这是游戏开发中常用的模式。

#### 核心概念

```
Entity (实体)
    ↓
Component (组件) - 纯数据，无逻辑
    ↓
System (系统) - 纯逻辑，处理组件
```

**示例**:
```go
// 实体：豌豆射手
peashooterID := em.NewEntity()

// 组件：位置
ecs.AddComponent(em, peashooterID, &components.PositionComponent{
    X: 100, Y: 200,
})

// 组件：植物属性
ecs.AddComponent(em, peashooterID, &components.PlantComponent{
    PlantType: "Peashooter",
    Health:    300,
})

// 系统：BehaviorSystem 处理豌豆射手攻击逻辑
```

### 目录结构详解

```
pvz3/
├── main.go                      # 游戏入口，初始化 Ebitengine
│
├── pkg/                         # 核心代码库
│   ├── ecs/                     # ECS 框架核心
│   │   ├── entity_manager.go   # 实体管理器
│   │   ├── generics.go          # 泛型 API（推荐使用）
│   │   └── ...
│   │
│   ├── components/              # 组件定义（纯数据）
│   │   ├── position.go          # 位置组件
│   │   ├── plant.go             # 植物组件
│   │   ├── zombie.go            # 僵尸组件
│   │   ├── reanim.go            # Reanim 动画组件
│   │   └── ...
│   │
│   ├── systems/                 # 系统实现（纯逻辑）
│   │   ├── behavior_system.go   # 行为系统（攻击、生产）
│   │   ├── input_system.go      # 输入处理
│   │   ├── physics_system.go    # 物理（移动、碰撞）
│   │   ├── reanim_system.go     # Reanim 动画系统
│   │   ├── particle_system.go   # 粒子系统
│   │   ├── render_system.go     # 渲染系统
│   │   └── ...
│   │
│   ├── entities/                # 实体工厂函数
│   │   ├── plant_factory.go     # 创建植物实体
│   │   ├── zombie_factory.go    # 创建僵尸实体
│   │   └── ...
│   │
│   ├── scenes/                  # 游戏场景
│   │   ├── game_scene.go        # 游戏主场景
│   │   ├── menu_scene.go        # 主菜单场景
│   │   └── ...
│   │
│   ├── game/                    # 游戏核心管理器
│   │   ├── scene_manager.go     # 场景管理
│   │   ├── resource_manager.go  # 资源加载
│   │   └── ...
│   │
│   ├── config/                  # 配置加载与管理
│   │   ├── level_config.go      # 关卡配置
│   │   ├── reanim_config.go     # Reanim 配置
│   │   └── ...
│   │
│   └── utils/                   # 通用工具函数
│       ├── math.go              # 数学工具
│       └── ...
│
├── assets/                      # 游戏资源（不提交到 Git）
│   ├── images/                  # 图片
│   ├── audio/                   # 音频
│   └── effect/                  # 粒子配置
│
├── data/                        # 外部化游戏数据（YAML）
│   ├── levels/                  # 关卡配置
│   ├── reanim/                  # Reanim 动画定义
│   └── reanim_config.yaml       # Reanim 动画配置
│
├── docs/                        # 文档
│   ├── prd/                     # 产品需求文档
│   ├── architecture/            # 架构文档
│   └── ...
│
└── CLAUDE.md                    # Claude Code 开发指南
```

### 关键系统说明

详细的架构设计请参见：
- **[架构文档](architecture.md)** - 完整的技术架构
- **[CLAUDE.md](../CLAUDE.md)** - ECS 使用指南、Reanim 系统、编码规范

---

## 🤝 代码贡献流程

### 开发工作流

```
1. 同步上游仓库
   ↓
2. 创建 Feature 分支
   ↓
3. 编写代码 + 测试
   ↓
4. 提交代码（规范的 Commit Message）
   ↓
5. 推送到您的 Fork
   ↓
6. 创建 Pull Request
   ↓
7. Code Review + 修改
   ↓
8. 合并到主分支
```

### 详细步骤

#### 1. 同步上游仓库

```bash
# 获取上游更新
git fetch upstream

# 切换到主分支
git checkout main

# 合并上游更新
git merge upstream/main

# 推送到您的 Fork
git push origin main
```

#### 2. 创建 Feature 分支

```bash
# 创建并切换到新分支
git checkout -b feature/add-chomper-plant

# 命名规范:
# - feature/xxx  - 新功能
# - fix/xxx      - Bug 修复
# - refactor/xxx - 代码重构
# - docs/xxx     - 文档更新
```

#### 3. 编写代码

遵循 [编码规范](#编码规范) 和 [测试指南](#测试指南)。

#### 4. 提交代码

```bash
# 添加文件
git add .

# 提交（规范的 Commit Message）
git commit -m "feat(plant): 添加大嘴花植物实现

- 实现 ChomperComponent 组件
- 实现吞噬僵尸行为逻辑
- 添加咀嚼状态管理
- 单元测试覆盖率 85%

Refs #123"
```

**Commit Message 规范**:
```
<type>(<scope>): <subject>

<body>

<footer>
```

**type 类型**:
- `feat`: 新功能
- `fix`: Bug 修复
- `refactor`: 重构
- `docs`: 文档
- `test`: 测试
- `chore`: 构建/工具

**示例**:
```
feat(ecs): 添加泛型 GetEntitiesWith API

- 实现 GetEntitiesWith1/2/3/4/5 泛型函数
- 替换反射实现，性能提升 30%
- 更新所有系统使用新 API

Closes #45
```

#### 5. 推送到 Fork

```bash
git push origin feature/add-chomper-plant
```

#### 6. 创建 Pull Request

1. 访问您的 Fork 页面
2. 点击 "New Pull Request"
3. 填写 PR 描述：
   ```markdown
   ## 变更说明
   添加大嘴花植物完整实现

   ## 变更内容
   - [ ] 新增 ChomperComponent
   - [ ] 实现吞噬行为逻辑
   - [ ] 添加单元测试

   ## 测试
   - [x] 单元测试通过
   - [x] 集成测试通过
   - [x] 手动测试通过

   ## 截图
   （如果有 UI 变更）

   Closes #123
   ```

#### 7. Code Review

- 响应审查意见
- 及时修改代码
- 保持讨论专业和友好

---

## 📏 编码规范

### Go 代码规范

#### 1. 使用 gofmt 格式化

```bash
# 格式化所有代码
gofmt -w .

# IDE 配置为保存时自动格式化
```

#### 2. 遵循 Go 命名约定

```go
// ✅ 好的命名
type PlantComponent struct { ... }
func NewPeashooterEntity() ecs.EntityID { ... }
const MaxPlants = 50

// ❌ 不好的命名
type plantcomp struct { ... }  // 应使用 PascalCase
func new_entity() { ... }       // 应使用 camelCase
```

#### 3. 添加 GoDoc 注释

```go
// PlantComponent 存储植物的核心属性数据。
//
// 该组件包含植物的类型、生命值和行为状态。
// 所有植物实体都必须包含此组件。
type PlantComponent struct {
    // PlantType 是植物的类型标识符（如 "Peashooter", "Sunflower"）
    PlantType string

    // Health 是植物的当前生命值
    Health int
}
```

### ECS 编码规范

#### 1. 使用泛型 API（推荐）

```go
// ✅ 推荐：使用泛型 API
plantComp, ok := ecs.GetComponent[*components.PlantComponent](em, entityID)
if ok {
    plantComp.Health -= 10  // 类型安全，无需断言
}

// ❌ 不推荐：使用反射 API（已废弃）
comp, ok := em.GetComponent(entityID, reflect.TypeOf(&components.PlantComponent{}))
if ok {
    plantComp := comp.(*components.PlantComponent)  // 需要手动断言
}
```

#### 2. 组件只包含数据

```go
// ✅ 正确：组件只包含数据
type PlantComponent struct {
    PlantType string
    Health    int
    AttackDamage int
}

// ❌ 错误：组件包含方法
type PlantComponent struct {
    PlantType string
    Health    int
}
func (p *PlantComponent) Attack() { ... }  // 不应在组件中定义方法
```

#### 3. 系统只包含逻辑

```go
// ✅ 正确：系统处理逻辑
type BehaviorSystem struct {
    em *ecs.EntityManager
}

func (s *BehaviorSystem) Update(dt float64) {
    // 查询实体并处理逻辑
    entities := ecs.GetEntitiesWith2[
        *components.PlantComponent,
        *components.PositionComponent,
    ](s.em)

    for _, entity := range entities {
        // 处理逻辑...
    }
}
```

#### 4. 使用实体工厂函数

```go
// ✅ 正确：使用工厂函数创建实体
func NewPeashooterEntity(em *ecs.EntityManager, x, y float64) ecs.EntityID {
    entity := em.NewEntity()

    ecs.AddComponent(em, entity, &components.PositionComponent{X: x, Y: y})
    ecs.AddComponent(em, entity, &components.PlantComponent{
        PlantType: "Peashooter",
        Health:    300,
    })

    return entity
}

// 使用
peashooterID := NewPeashooterEntity(em, 100, 200)
```

### 完整的编码规范

详细的编码规范请参见 **[CLAUDE.md](../CLAUDE.md)** 中的相关章节。

---

## 🧪 测试指南

### 测试类型

#### 1. 单元测试

测试单个组件或函数的逻辑。

**示例**:
```go
// pkg/systems/behavior_system_test.go
package systems

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestPeashooterAttackBehavior(t *testing.T) {
    // 设置
    em := ecs.NewEntityManager()
    bs := NewBehaviorSystem(em)

    peashooter := createTestPeashooter(em)
    zombie := createTestZombie(em)

    // 执行
    bs.Update(1.0)

    // 验证
    bullets := ecs.GetEntitiesWith1[*components.BulletComponent](em)
    assert.Equal(t, 1, len(bullets), "应该发射一颗子弹")
}
```

#### 2. 集成测试

测试多个系统协同工作。

```go
func TestPlantZombieCombat(t *testing.T) {
    // 创建完整场景
    scene := createTestGameScene()

    // 模拟游戏循环
    for i := 0; i < 100; i++ {
        scene.Update(0.016) // 60 FPS
    }

    // 验证僵尸被消灭
    zombies := ecs.GetEntitiesWith1[*components.ZombieComponent](scene.em)
    assert.Equal(t, 0, len(zombies))
}
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./pkg/systems

# 运行特定测试函数
go test ./pkg/systems -run TestPeashooterAttackBehavior

# 显示详细输出
go test -v ./...

# 查看覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 测试覆盖率目标

- **单元测试覆盖率**: ≥ 80%
- **关键系统覆盖率**: ≥ 90%（BehaviorSystem, PhysicsSystem等）

---

## 🐛 调试技巧

### 1. 启用详细日志

```bash
# 运行游戏时启用 verbose 日志
go run . --verbose
```

**日志输出示例**:
```
[ReanimSystem] 播放动画组合: peashooter/attack
[ReanimSystem] 自动轨道绑定:
  - anim_face -> anim_head_idle
  - stalk_bottom -> anim_shooting
[ParticleSystem] 生成粒子效果: Planting (100 粒子)
[BehaviorSystem] 豌豆射手 (entity 456) 发射子弹
```

### 2. 使用 Delve 调试器

```bash
# 安装 Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 启动调试
dlv debug .

# 在 Delve 中设置断点
(dlv) break pkg/systems/behavior_system.go:42
(dlv) continue
```

### 3. VSCode 调试配置

`.vscode/launch.json`:
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Game",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}",
      "args": ["--verbose"]
    }
  ]
}
```

### 4. 常见调试场景

#### 调试 Reanim 动画问题
```bash
# 使用动画查看器工具
go run cmd/animation_showcase/main.go --reanim=PeaShooter --verbose
```

#### 调试粒子系统
```bash
# 使用粒子测试工具
go run cmd/particles/main.go --effect=Planting --verbose
```

---

## 🔨 常见开发任务

### 添加新植物

#### 1. 定义组件（如果需要）
```go
// pkg/components/plant.go
// 通常使用已有的 PlantComponent
```

#### 2. 创建工厂函数
```go
// pkg/entities/plant_factory.go
func NewChomperEntity(em *ecs.EntityManager, x, y float64) ecs.EntityID {
    entity := em.NewEntity()

    ecs.AddComponent(em, entity, &components.PositionComponent{X: x, Y: y})
    ecs.AddComponent(em, entity, &components.PlantComponent{
        PlantType:    "Chomper",
        Health:       300,
        AttackDamage: 0,  // 大嘴花直接吞噬
    })
    ecs.AddComponent(em, entity, &components.ReanimComponent{
        ReanimID:   "chomper",
        CurrentAnim: "anim_idle",
    })

    return entity
}
```

#### 3. 实现行为逻辑
```go
// pkg/systems/behavior_system.go
func (s *BehaviorSystem) updateChomperBehavior(entity ecs.EntityID, plant *components.PlantComponent, ...) {
    // 检测范围内僵尸
    // 吞噬僵尸
    // 进入咀嚼状态
}
```

#### 4. 添加配置
```yaml
# data/reanim_config.yaml
- id: chomper
  name: Chomper
  reanim_file: assets/effect/reanim/Chomper.reanim
  default_animation: anim_idle
  combos:
    idle:
      animations: ["anim_idle"]
    attack:
      animations: ["anim_bite"]
```

#### 5. 编写测试
```go
// pkg/systems/behavior_system_test.go
func TestChomperSwallowZombie(t *testing.T) {
    // ...测试逻辑
}
```

### 添加新关卡

#### 1. 创建关卡配置
```yaml
# data/levels/level-2-1.yaml
id: "2-1"
chapter: 2
name: "夜晚第一关"
environment: "night"
flags: 2
enabledLanes: [1, 2, 3, 4, 5]
availablePlants:
  - peashooter
  - sunflower
  - wallnut
  - puffshroom
zombieWaves:
  - wave: 1
    time: 10.0
    zombies:
      - type: normal
        lane: 3
```

#### 2. 测试关卡
```bash
go run . --level=2-1 --verbose
```

---

## 📚 相关文档

### 技术文档
- **[架构文档](architecture.md)** - 完整的技术架构设计
- **[CLAUDE.md](../CLAUDE.md)** - ECS 使用指南、Reanim 系统详解
- **[PRD](prd.md)** - 产品需求文档

### 参考资料
- **[Ebitengine 文档](https://ebiten.org/)** - 游戏引擎官方文档
- **[Go 语言规范](https://go.dev/ref/spec)** - Go 官方语言规范
- **[ECS 模式](https://github.com/SanderMertens/ecs-faq)** - ECS 架构常见问题

---

## 🆘 获取帮助

- **提交 Issue**: [项目 Issues](../../issues)
- **参与讨论**: [GitHub Discussions](../../discussions)
- **查看 Wiki**: [项目 Wiki](../../wiki)

---

**感谢您的贡献！** 🌻💻✨
