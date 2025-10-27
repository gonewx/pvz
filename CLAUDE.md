# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

这是一个使用 Go 语言和 Ebitengine 引擎开发的《植物大战僵尸》复刻项目。项目采用实体-组件-系统(ECS)架构模式,目标是精确复刻原版PC游戏的前院白天关卡。

## 核心开发命令

### 项目初始化与依赖管理
```bash
# 初始化 Go 模块(如果尚未初始化)
go mod init github.com/decker502/pvz

# 添加 Ebitengine 依赖
go get github.com/hajimehoshi/ebiten/v2

# 添加 YAML 解析库
go get gopkg.in/yaml.v3

# 下载所有依赖
go mod download

# 整理依赖
go mod tidy
```

### 构建与运行
```bash
# 运行游戏
go run .

# 构建可执行文件
go build -o pvz-go .

# 构建优化版本(发布用)
go build -ldflags="-s -w" -o pvz-go .

# 交叉编译 Windows 版本
GOOS=windows GOARCH=amd64 go build -o pvz-go.exe .

# 交叉编译 macOS 版本
GOOS=darwin GOARCH=amd64 go build -o pvz-go-mac .
```

### 代码质量
```bash
# 格式化代码
gofmt -w .

# 运行所有测试
go test ./...

# 运行测试并显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 高层架构

### ECS 架构模式

本项目采用实体-组件-系统(Entity-Component-System)架构:

- **实体(Entity)**: 游戏对象的唯一标识符(如植物、僵尸、子弹)
- **组件(Component)**: 纯数据结构,描述实体的属性(如位置、生命值、精灵图)
- **系统(System)**: 处理逻辑的函数,对拥有特定组件的实体进行操作

**关键原则**:
- 组件只包含数据,不包含方法
- 系统之间通过 EntityManager 或 EventBus 通信,不直接调用
- 数据与行为完全分离

### 核心系统层级

```
main.go (游戏入口)
    ↓
SceneManager (场景管理器)
    ↓
├── MainMenuScene (主菜单场景)
└── GameScene (游戏场景)
        ↓
    EntityManager (实体管理器)
        ↓
    ├── InputSystem (输入系统)
    ├── BehaviorSystem (行为系统)
    ├── PhysicsSystem (物理系统)
    ├── AnimationSystem (动画系统)
    ├── ParticleSystem (粒子系统 - Story 7.2)
    ├── UISystem (UI系统)
    └── RenderSystem (渲染系统)
        ├── DrawGameWorld() - 植物、僵尸、子弹
        ├── DrawParticles() - 粒子效果 (Story 7.3)
        └── DrawSuns() - 阳光
```

### 目录结构规范

```plaintext
pvz/
├── main.go                 # 游戏主入口
├── assets/                 # 游戏资源
│   ├── images/             # 图片资源(spritesheets)
│   ├── audio/              # 音频资源
│   └── fonts/              # 字体文件
├── data/                   # 外部化游戏数据
│   ├── levels/             # 关卡配置(YAML)
│   └── units/              # 单位属性文件
├── pkg/                    # 核心代码库
│   ├── components/         # 所有组件定义
│   ├── entities/           # 实体工厂函数
│   ├── systems/            # 所有系统实现
│   ├── scenes/             # 游戏场景
│   ├── ecs/                # ECS框架核心
│   ├── game/               # 游戏核心管理器
│   ├── utils/              # 通用工具函数
│   └── config/             # 配置加载与管理
├── go.mod
└── go.sum
```

## 泛型 ECS API 使用指南

本项目的 ECS 框架已升级为基于 Go 泛型的类型安全 API（Epic 9 - Story 9.1/9.2/9.3）。

### 泛型 API 优势

- ✅ **编译时类型检查**：消除运行时 panic 风险，约 150+ 处潜在错误被编译器捕获
- ✅ **无需类型断言**：代码更简洁，消除了 60+ 处手动类型断言
- ✅ **性能提升**：减少反射开销，综合性能提升约 10%
- ✅ **更好的 IDE 支持**：代码补全、类型推导、重构工具全面支持
- ✅ **代码可读性**：代码行数减少 40-60%，更易维护

### 基本用法

#### 1. 获取组件（GetComponent）

```go
// ❌ 旧方式（反射 API，已废弃）
comp, ok := em.GetComponent(entity, reflect.TypeOf(&components.PlantComponent{}))
if ok {
    plantComp := comp.(*components.PlantComponent) // 手动类型断言，可能 panic
    plantComp.Health -= 10
}

// ✅ 新方式（泛型 API，推荐）
plantComp, ok := ecs.GetComponent[*components.PlantComponent](em, entity)
if ok {
    plantComp.Health -= 10 // 无需类型断言，类型安全
}
```

#### 2. 添加组件（AddComponent）

```go
// ❌ 旧方式
em.AddComponent(entity, &components.PlantComponent{
    PlantType: "Peashooter",
    Health:    300,
})

// ✅ 新方式（类型自动推导）
ecs.AddComponent(em, entity, &components.PlantComponent{
    PlantType: "Peashooter",
    Health:    300,
})
```

#### 3. 检查组件存在性（HasComponent）

```go
// ❌ 旧方式
if em.HasComponent(entity, reflect.TypeOf(&components.PlantComponent{})) {
    // 处理植物逻辑
}

// ✅ 新方式
if ecs.HasComponent[*components.PlantComponent](em, entity) {
    // 处理植物逻辑
}
```

#### 4. 查询实体（GetEntitiesWith）

```go
// ❌ 旧方式（冗长且运行时检查）
entities := em.GetEntitiesWith(
    reflect.TypeOf(&components.BehaviorComponent{}),
    reflect.TypeOf(&components.PlantComponent{}),
    reflect.TypeOf(&components.PositionComponent{}),
)

// ✅ 新方式（编译时类型检查）
entities := ecs.GetEntitiesWith3[
    *components.BehaviorComponent,
    *components.PlantComponent,
    *components.PositionComponent,
](em)
```

**函数选择规则**：
- 查询 1 个组件 → 使用 `GetEntitiesWith1[T1]`
- 查询 2 个组件 → 使用 `GetEntitiesWith2[T1, T2]`
- 查询 3 个组件 → 使用 `GetEntitiesWith3[T1, T2, T3]`
- 查询 4 个组件 → 使用 `GetEntitiesWith4[T1, T2, T3, T4]`
- 查询 5 个组件 → 使用 `GetEntitiesWith5[T1, T2, T3, T4, T5]`
- 查询 5+ 组件 → 使用反射 API 或分步查询

**特点**：
- 无需手动使用 `reflect.TypeOf`
- 代码更简洁
- 与其他泛型 API 风格一致

**解决方案 C** （推荐）：重新设计组件
- 如果需要查询超过 5 个组件，可能说明组件设计过于碎片化
- 考虑合并相关组件或使用组合组件

---

## 核心组件说明

### 必要组件(所有实体必备)
- `PositionComponent`: 存储 X,Y 坐标
- `SpriteComponent`: 存储要绘制的图像引用

### 功能组件
- `AnimationComponent`: 管理基于spritesheet的帧动画
- `HealthComponent`: 生命值管理(CurrentHealth, MaxHealth)
- `BehaviorComponent`: 定义实体行为类型(向日葵、豌豆射手等)
- `TimerComponent`: 通用计时器(用于攻击冷却、生产周期等)
- `UIComponent`: 标记UI元素及其状态
- `VelocityComponent`: 移动速度(用于子弹、僵尸移动)
- `CollisionComponent`: 碰撞检测的边界框
- `ParticleComponent`: 粒子效果数据(位置、速度、颜色、生命周期等) - Story 7.2
- `EmitterComponent`: 粒子发射器配置(生成规则、限制、力场等) - Story 7.2

## 坐标系统使用指南

### 世界坐标 vs 屏幕坐标

本项目使用**世界坐标系统**：
- **世界坐标**：相对于背景图片左上角（固定）
- **屏幕坐标**：相对于游戏窗口左上角（随摄像机移动）
- **转换公式**：`worldX = screenX + cameraX`

### 何时使用哪种坐标？

| 场景 | 使用坐标类型 | 示例 |
|------|------------|------|
| 组件存储位置 | 世界坐标 | `PositionComponent.X/Y` |
| 网格定义 | 世界坐标 | `config.GridWorldStartX` |
| 鼠标输入 | 屏幕坐标 | `ebiten.CursorPosition()` |
| 渲染绘制 | 屏幕坐标 | `screen.DrawImage()` |

## 资源管理

### 概述

项目使用统一的 `ResourceManager` 进行资源加载和缓存管理。从 2025年10月开始，资源系统已升级为**基于 YAML 配置的动态资源管理**，支持通过资源 ID 加载资源，提高了可维护性和可扩展性。

### 资源配置文件

**配置文件路径:** `assets/config/resources.yaml`

资源配置文件定义了所有游戏资源及其 ID 映射关系。资源按照**资源组（Resource Groups）**组织，可以批量加载。

**配置结构示例：**
```yaml
version: "1.0"
base_path: assets
groups:
  init:
    images:
      - id: IMAGE_BLANK
        path: properties/blank.png
      - id: IMAGE_POPCAP_LOGO
        path: properties/PopCap_Logo.jpg
  loadingimages:
    images:
      - id: IMAGE_REANIM_SEEDS
        path: reanim/seeds.png
        cols: 9  # 精灵图列数
    sounds:
      - id: SOUND_BUTTONCLICK
        path: sounds/buttonclick.ogg
```
## 文档参考

项目相关文档位于 `docs/` 目录:
- `docs/prd.md`: 产品需求文档
- `docs/architecture.md`: 详细架构设计文档
- `docs/brief.md`: 项目简介
- `docs/front-end-spec.md`: 前端规范

---

## 用户手工维护区域

<!--
在此区域添加您的个人项目笔记、配置、工作流程等内容
此区域内容不会被文档更新脚本覆盖
-->

- 所有资源类文件已经准备好， @assets

- 永远不要为了赶时间、或认为篇幅有限、或认为任务复杂，而主观的简化或加速任务的实现。如果有这种情况，要显式的征得我的同意，或授权确认后，才能简化实现。

- 如果遇到网络问题，请尝试使用网络代理 http://127.0.0.1:2080 访问

- 遇到反复无法修复的问题或有不熟悉的第三方库, 尝试使用 `mcp__deepwiki` 工具的`ask_question`方法，查阅最新的文档，以找到最正确的修复方法 。

- 确认功能正常后，再提交git
---

- 原版《植物大战僵尸》使用固定时间步长 **0.01秒（1厘秒）** 作为物理更新基准（相当于100FPS）。粒子配置文件中的某些值基于这个时间步长定义，而非真实的"每秒"单位。
- assets/effect 下的所有配置文件都不能修改
- 所有涉及大小 、位置的常量，都需要在配置常量文件中设置，以方便后续手工调整
- 如果要查看日志，需要添加参数 `--verbose`
- 绘制任何元素时，都要考虑是否需要坐标转换。
- 验证粒子系统的实现， 可以使用类似 `go run cmd/particles/main.go --verbose --effect="Planting"  > /tmp/p.log 2>&1` 的命令运行，并查看日志
- 如果没有动画轨道,那可能是简单的动画组件,应该按配置的名称,直接
播放就行