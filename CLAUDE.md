# CLAUDE.md

> **重要说明**:
> - 本文件为 **Claude Code AI 工具**提供开发上下文，**不是用户文档**
> - 仅包含开发时的技术指导、架构说明和编码规范
> - 用户文档请参见: [README.md](README.md), [快速开始](docs/quickstart.md), [用户手册](docs/user-guide.md), [开发指南](docs/development.md)

---

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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
    ├── ReanimSystem (Reanim动画系统 - Story 6.5)
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
- `ReanimComponent`: Reanim 动画组件，支持复杂的多轨道动画系统 - Epic 13

## Reanim 动画配置系统（Epic 13）

本项目使用**配置驱动**的 Reanim 动画系统，所有动画配置统一在 YAML 文件中管理，无需修改代码即可调整动画组合。

### 配置文件位置

**配置目录**：`data/reanim_config/` (Story 13.9 - 多文件架构)

这是一个分片配置目录，每个动画单元对应一个独立的 YAML 文件，便于定位、编辑和维护。

**全局配置文件**：`data/reanim_config.yaml`（仅包含全局设置）

**示例结构**：
```
data/
├── reanim_config.yaml          # 全局配置
└── reanim_config/              # 动画单元配置目录
    ├── peashooter.yaml         # 豌豆射手配置
    ├── sunflower.yaml          # 向日葵配置
    ├── zombie.yaml             # 僵尸配置
    ├── zombie_conehead.yaml    # 路障僵尸配置
    └── ... (共 142 个文件)
```

### 配置文件格式

**全局配置文件** (`data/reanim_config.yaml`):
```yaml
global:
  playback:
    tps: 60      # 游戏目标 TPS
    fps: 12      # 默认动画 FPS
```

**单个动画单元配置** (`data/reanim_config/peashooter.yaml`):
```yaml
id: "peashooter"                    # 动画单元 ID
name: "PeaShooter"                  # 显示名称
reanim_file: "data/reanim/PeaShooterSingle.reanim"
default_animation: "anim_idle"     # 默认动画

images:                             # 图片映射
  IMAGE_REANIM_PEASHOOTER_HEAD: "assets/reanim/PeaShooter_Head.png"
  # ... 其他图片

available_animations:               # 可用动画列表
  - name: "anim_idle"
    display_name: "待机"
  - name: "anim_shooting"
    display_name: "攻击"

animation_combos:                   # 动画组合配置
  - name: "attack"                  # 组合名称
    display_name: "攻击+摇晃"
    animations: ["anim_shooting", "anim_head_idle"]  # 多动画组合
    parent_tracks:                  # 父子关系定义
      anim_face: "anim_stem"       # 头部跟随茎干
```

### 渲染架构（Story 13.10 - 正向渲染逻辑）

**核心设计理念**：动画是主体，轨道是数据源

从 Story 13.10 开始，Reanim 系统采用**正向渲染逻辑**（动画→轨道），完全删除了轨道绑定机制（TrackAnimationBinding）。

**旧逻辑（已废弃）**：
```go
// ❌ 反向逻辑：从轨道查找控制动画
for _, trackName := range visualTracks {
    controllingAnim := findControllingAnimation(trackName)  // 需要复杂的绑定分析
    frame := getFrame(controllingAnim, trackName)
    render(frame)
}
```

**新逻辑（当前实现）**：
```go
// ✅ 正向逻辑：从动画遍历轨道
for _, animName := range currentAnimations {
    physicalFrame := getPhysicalFrame(animName)
    if isAnimationVisible(animName, physicalFrame) {
        for _, trackName := range visualTracks {
            frame := tracks[trackName][physicalFrame]
            if frame.ImagePath != "" {
                render(frame)  // 后面的动画自然覆盖前面的
            }
        }
    }
}
```

**优势**：
- ✅ **无需轨道绑定**：删除了 analyzeTrackBinding、findControllingAnimation 等复杂机制（净减少约 241 行代码）
- ✅ **自然的图层覆盖**：后播放的动画自动覆盖前面的动画，符合 Reanim 格式的原始设计
- ✅ **多阶段动画**：轨道可以在不同动画阶段显示不同内容（如草叶子：开场滑入，循环摇摆）
- ✅ **代码更简洁**：渲染逻辑从 ~150 行减少到 ~80 行

**示例：SelectorScreen 多动画渲染**
```
CurrentAnimations = ["anim_open", "anim_grass"]

渲染结果：
├─ 背景（SelectorScreen_BG）：来自 anim_open（anim_grass 未覆盖）
└─ 叶子（leaf_SelectorScreen_Leaves）：来自 anim_grass（覆盖了 anim_open）
```

### 配置驱动的动画播放 API

**推荐方式**：使用配置驱动的 API（Epic 13 - Story 13.6）

```go
// ✅ 推荐：使用 PlayCombo 播放配置的动画组合
rs.PlayCombo(entityID, "peashooter", "attack")
// 自动处理：
// - 播放 anim_shooting + anim_head_idle
// - 应用父子关系（头部跟随身体）
// - 使用正向渲染逻辑（动画→轨道）

// ✅ 推荐：使用 PlayDefaultAnimation 播放默认动画
rs.PlayDefaultAnimation(entityID, "peashooter")
// 自动播放 default_animation 配置的动画

// ❌ 不推荐：硬编码动画名称（已废弃）
rs.PlayAnimations(entityID, []string{"anim_shooting", "anim_head_idle"})
```

### 配置管理器

游戏启动时会自动加载配置：

```go
// 配置目录在 main.go 中预加载 (Story 13.9)
configManager, err := config.NewReanimConfigManager("data/reanim_config")
if err != nil {
    log.Fatalf("加载 Reanim 配置失败: %v", err)
}

// 配置管理器传递给 ReanimSystem
reanimSystem.SetConfigManager(configManager)
```

### 添加新实体的动画配置

1. 在 `data/reanim_config/` 目录中创建新的配置文件：
   ```yaml
   # data/reanim_config/new_plant.yaml
   id: "new_plant"                  # 新实体 ID
   name: "NewPlant"
   reanim_file: "data/reanim/NewPlant.reanim"
   default_animation: "anim_idle"
   # ... 配置其他字段
   ```

2. 在代码中使用配置驱动的 API：
   ```go
   // 实体初始化时播放默认动画
   rs.PlayDefaultAnimation(entityID, "new_plant")

   // 需要时播放特定组合
   rs.PlayCombo(entityID, "new_plant", "attack")
   ```

### 关键设计原则

- ✅ **配置优于硬编码**：动画组合在配置文件中定义
- ✅ **集中管理**：所有配置在单个文件中，便于维护
- ✅ **自动绑定**：系统自动处理轨道绑定和父子关系
- ✅ **热修改**：修改配置文件无需重新编译代码

### 详细文档参考

- **配置指南**：[docs/reanim/reanim-config-guide.md](docs/reanim/reanim-config-guide.md)
- **Epic 13 PRD**：[docs/prd/epic-13-reanim-modernization.md](docs/prd/epic-13-reanim-modernization.md)


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

### Reanim 坐标系统和锚点 (Epic 13 重要经验)

**关键发现：渲染系统把 Reanim 文件中的坐标当作图片左上角使用**

#### 渲染公式

```go
// RenderSystem 的实际渲染位置计算
baseScreenX = pos.X - cameraX - CenterOffsetX
partX = frame.X  // 来自 Reanim 文件的 X 坐标
finalScreenX = baseScreenX + partX

// 转换为世界坐标
worldX = finalScreenX + cameraX = pos.X - CenterOffsetX + partX
```

**关键点：**
- `frame.X`（Reanim 文件中的 X）被当作**图片左上角**的相对坐标
- 图片使用左上角锚点渲染（`x0 = tx, y0 = ty`）
- 不需要手动转换中心到左上角

**问题：**草皮叠加层应该铺到哪里？

**关键设计决策：**
- 草皮卷是一个圆柱体（SodRoll），滚动时逐渐展开草皮
- 草皮应该铺到**草皮卷的中心位置**（圆柱体的中心线），而不是左边缘或右边缘
- 物理上合理：草皮从卷的中心展开，铺设在地面上

**错误理解：**
```go
// ❌ 错误：草皮铺到草皮卷的左边缘
grassRightEdge := pos.X - CenterOffsetX + SodRoll.X
// 结果：草皮铺设不足，未覆盖应该覆盖的区域
```

**正确实现：**
```go
// ✅ 正确：草皮铺到草皮卷的中心（sodding_system.go:calculateCurrentCenterX）
sodRollCenterX := SodRoll.X + scaledHalfWidth  // 图片中心
worldRightEdgeX := pos.X - CenterOffsetX + sodRollCenterX
// GetSodRollCenterX() 返回此值，用于裁剪草皮叠加层的可见宽度
```

## 文档参考

### 用户文档
- **[README.md](README.md)** - 项目概览、功能特性
- **[快速开始](docs/quickstart.md)** - 环境设置、构建运行
- **[用户手册](docs/user-guide.md)** - 游戏操作、植物僵尸详解
- **[开发指南](docs/development.md)** - 代码贡献、开发流程

### 技术文档
- **[PRD](docs/prd.md)** - 产品需求文档
- **[Architecture](docs/architecture.md)** - 详细架构设计文档
- **[Brief](docs/brief.md)** - 项目简介
- **[Frontend Spec](docs/front-end-spec.md)** - 前端规范

---

## 用户手工维护区域

<!--
在此区域添加您的个人项目笔记、配置、工作流程等内容
此区域内容不会被文档更新脚本覆盖
-->

- 所有资源类文件已经准备好， @assets

- 必须在项目根目录运行测试，所以测试中需要引用的项目目录和文件，也要以相对于项目根目录来组织路径

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
- 如果没有动画轨道,那可能是简单的动画组件,应该按配置的名称,直接播放就行
- 将刚刚的经验记录起来