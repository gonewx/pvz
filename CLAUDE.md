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

# 或使用 goimports(自动添加/移除导入)
goimports -w .

# 代码检查
golangci-lint run

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

## 组件使用策略（重要）

### Story 6.3 迁移范围澄清

**常见误解**：
- ❌ "Story 6.3 要替代所有 SpriteComponent"
- ✅ "Story 6.3 只迁移游戏世界实体到 ReanimComponent"

**实际架构**：
```
游戏世界实体             UI 元素
    ↓                       ↓
ReanimComponent      SpriteComponent
    ↓                       ↓
RenderSystem       PlantCardRenderSystem
```

### ReanimComponent vs SpriteComponent 使用规则

#### 何时使用 ReanimComponent？
- ✅ **游戏世界实体**：植物、僵尸、子弹、阳光、特效
- ✅ **需要复杂动画**：多部件、骨骼动画、变换效果
- ✅ **特点**：支持多部件渲染、骨骼变换、帧继承

#### 何时使用 SpriteComponent？
- ✅ **UI 元素**：植物卡片、按钮、菜单
- ✅ **静态图片**：背景、装饰元素
- ✅ **特点**：简单高效，适合不需要复杂动画的元素

### 为什么 UI 元素不使用 ReanimComponent？

1. ✅ **性能优化**：UI 不需要复杂的多部件动画系统
2. ✅ **关注点分离**：UI 渲染逻辑与游戏逻辑分离
3. ✅ **简单性**：SpriteComponent 更适合静态/简单动画的 UI
4. ✅ **专门的渲染系统**：UI 有特殊需求（如冷却遮罩、缩放）

### 实体组件映射表

| 实体类型 | 组件类型 | 渲染系统 | 说明 |
|---------|---------|---------|------|
| 🌱 植物 | ReanimComponent | RenderSystem | 完整动画系统 |
| 🧟 僵尸 | ReanimComponent | RenderSystem | 完整动画系统 |
| ☀️ 阳光 | ReanimComponent | RenderSystem | 简化包装（单图片） |
| 🟢 子弹 | ReanimComponent | RenderSystem | 简化包装（单图片） |
| 💥 特效 | ReanimComponent | RenderSystem | 简化包装（单图片） |
| ✨ 粒子 | ParticleComponent | RenderSystem.DrawParticles() | 高性能批量渲染 (Story 7.3) |
| 👻 植物预览 | ReanimComponent | PlantPreviewRenderSystem | 完整动画（双图像渲染） |
| 🎴 植物卡片 | SpriteComponent | PlantCardRenderSystem | UI 元素 |

### 相关文档

- **架构决策记录**：`docs/architecture/adr/001-component-strategy.md`（如有）
- **Story 6.3 详细说明**：`docs/stories/6.3.story.md`
- **渲染系统文档**：`pkg/systems/render_system.go`（文件头部注释）

## UI 元素复用策略（Story 8.4）

### 概述

为了消除重复代码并提高可维护性，项目实现了统一的植物卡片渲染机制，所有渲染逻辑封装在 **PlantCardFactory** 中。

### 植物卡片渲染架构

**工厂位置**：`pkg/entities/plant_card_factory.go`

**渲染函数**：
- `NewPlantCardEntity()` - 创建植物卡片实体（包含所有组件和渲染资源）
- `RenderPlantCard()` - 统一的渲染函数（封装所有渲染逻辑）
- `RenderPlantIcon()` - 使用 Reanim 系统离屏渲染植物预览图标

**渲染功能**：
- 背景框（应用卡片缩放）
- 植物图标（使用配置的缩放和偏移）
- 阳光数字（使用配置的偏移）
- 冷却遮罩（从下往上填充）
- 禁用遮罩（阳光不足时的灰色遮罩）

**使用示例**：

```go
// 1. 创建植物卡片实体（在初始化时）
cardEntity, err := entities.NewPlantCardEntity(
    em,                        // 实体管理器
    rm,                        // 资源管理器
    rs,                        // Reanim系统（用于渲染植物预览）
    components.PlantPeashooter, // 植物类型
    100, 50,                   // 卡片位置(x, y)
    0.54,                      // 卡片缩放因子
)

// 2. 在渲染系统中调用统一渲染函数
entities.RenderPlantCard(
    screen,      // 渲染目标
    card,        // PlantCardComponent
    pos.X, pos.Y, // 卡片位置
    sunFont,     // 阳光字体（可选）
    sunFontSize, // 字体大小
)
```

### 当前使用场景

| 系统 | 使用方式 | 说明 |
|------|---------|------|
| PlantCardRenderSystem | 直接调用 `entities.RenderPlantCard()` | 选卡界面植物卡片渲染 |
| RewardPanelRenderSystem | 调用 `entities.NewPlantCardEntity()` 创建实体 | 奖励面板通过实体系统统一渲染 |

### 设计原则

1. **工厂封装** - 所有卡片配置（背景、缩放、偏移）在工厂内部封装
2. **统一整体** - 卡片作为统一整体，不暴露内部细节给调用者
3. **配置驱动** - 所有内部配置在 `pkg/config/plant_card_config.go` 中定义
4. **高复用** - 统一的渲染函数可在任何场景使用

### 配置说明

**配置文件**：`pkg/config/plant_card_config.go`

**配置项**：
- `PlantCardBackgroundID` - 卡片背景图资源ID
- `PlantCardIconScale` - 植物图标缩放因子
- `PlantCardIconOffsetY` - 植物图标Y轴偏移
- `PlantCardSunCostOffsetY` - 阳光数字Y轴偏移

### 扩展性

**未来支持的新功能**：
- 图鉴界面：通过 `NewPlantCardEntity()` 创建卡片实体
- 商店界面：通过 `NewPlantCardEntity()` 创建卡片实体
- 自定义效果：在 `RenderPlantCard()` 中添加新的渲染层级

**相关文档**：`docs/stories/8.4.story.md`

## 植物选择栏模块化架构（架构优化）

### 概述

植物选择栏作为游戏的核心 UI 模块，已通过 `PlantSelectionModule` 实现模块化封装，遵循 ECS 架构的"高内聚、低耦合"原则。

### 设计原则

1. **高内聚** - 所有选卡相关功能（选择、取消、冷却、渲染）封装在单一模块中
2. **低耦合** - 通过清晰的接口与其他系统交互，不直接访问外部系统
3. **可复用** - 支持在不同场景（游戏中、选卡界面、图鉴）使用

### 架构对比

#### ❌ 旧架构（已重构前）
```
GameScene
  ├── 直接创建 PlantCardEntity (耦合)
  ├── PlantCardSystem (更新状态)
  └── PlantCardRenderSystem (渲染)
```
**问题**：
- 职责分散在多处
- GameScene 直接管理卡片创建
- 难以复用到其他场景

#### ✅ 新架构（当前）
```
PlantSelectionModule
  ├── PlantSelectionSystem (选卡逻辑)
  ├── PlantCardSystem (卡片状态)
  └── PlantCardRenderSystem (渲染)

GameScene 通过模块接口调用：
  ├── plantSelectionModule.Update()
  └── plantSelectionModule.Draw()
```
**优点**：
- ✅ 所有功能封装在模块内
- ✅ GameScene 只通过接口交互
- ✅ 可轻松移植到其他场景

### 组件和系统映射

| 组件 | 系统 | 职责 |
|------|------|------|
| PlantSelectionComponent | PlantSelectionSystem | 管理选择列表、确认状态 |
| PlantCardComponent | PlantCardSystem | 更新卡片冷却、可用性 |
| PlantCardComponent | PlantCardRenderSystem | 渲染卡片（背景、图标、遮罩） |
| PositionComponent | InputSystem | 处理点击事件 |

### 使用示例

#### 初始化模块

```go
// 在 GameScene.initPlantCardSystems() 中
plantSelectionModule, err := modules.NewPlantSelectionModule(
    entityManager,
    gameState,
    resourceManager,
    reanimSystem,
    levelConfig,        // 从关卡配置读取可用植物
    plantCardFont,
    SeedBankX,
    SeedBankY,
)
```

#### 更新和渲染

```go
// 在 GameScene.Update() 中
func (s *GameScene) Update(dt float64) {
    if s.plantSelectionModule != nil {
        s.plantSelectionModule.Update(dt)
    }
}

// 在 GameScene.Draw() 中
func (s *GameScene) Draw(screen *ebiten.Image) {
    if s.plantSelectionModule != nil {
        s.plantSelectionModule.Draw(screen)
    }
}
```

#### 获取选择结果

```go
// 获取玩家选择的植物列表
selectedPlants := plantSelectionModule.GetSelectedPlants()
// 返回：[]string{"peashooter", "sunflower", "wallnut"}
```

### 代码位置

**模块位置**: `pkg/modules/plant_selection_module.go`

**核心方法**:
- `NewPlantSelectionModule()` - 创建模块实例
- `Update(deltaTime)` - 更新所有子系统
- `Draw(screen)` - 渲染所有植物卡片
- `GetSelectedPlants()` - 获取选择结果
- `Cleanup()` - 清理资源

**使用场景**:
- **GameScene** - 游戏中的植物选择栏（当前）
- **PlantSelectionScene** - 关卡开始前的选卡界面（Epic 8 待实现）
- **图鉴界面** - 展示所有植物（未来扩展）
- **商店界面** - 购买新植物（未来扩展）

### 扩展性

**未来支持的新功能**：
- 图鉴界面：创建 `PlantAlmanacScene`，使用 `PlantSelectionModule` 管理卡片显示
- 商店界面：创建 `ShopScene`，使用 `PlantSelectionModule` 管理卡片购买
- 对战选择：创建 `BattleSelectScene`，使用 `PlantSelectionModule` 管理卡片选择

**统一模式**：
```go
// 所有使用植物卡片的场景都采用相同模式：
module := modules.NewPlantSelectionModule(...)
module.Update(dt)
module.Draw(screen)
selectedPlants := module.GetSelectedPlants()
```

**相关文档**：Sprint Change Proposal - 植物选择栏架构一致性审查

## 粒子系统渲染层级管理（Story 8.5）

### 问题背景

在实现奖励动画系统时，发现了**渲染层级冲突**问题：
- 奖励动画的粒子效果和植物选择栏的卡片渲染顺序错误
- 两处使用同样的 `PlantCardRenderSystem`，无法分别控制渲染层级
- 不同系统的卡片会被错误地处理（状态更新、点击事件）

### 解决方案：系统自管理模式（System Self-Management）

#### 核心设计原则

**符合 ECS 架构**：
- ✅ **系统负责自己的实体** - `RewardAnimationSystem` 管理自己的卡片
- ✅ **低耦合** - 不依赖全局的 `PlantCardRenderSystem`
- ✅ **职责清晰** - 每个系统只关心自己的实体
- ✅ **扩展性强** - 新增模块不影响现有系统

#### RewardCardComponent 标记组件

**位置**：`pkg/components/reward_card_component.go`

**用途**：
- 区分奖励卡片和选择栏卡片
- 允许不同系统有针对性地处理实体

**使用**：
```go
// 在 RewardAnimationSystem 创建奖励卡片时添加标记
ecs.AddComponent(em, cardEntity, &components.RewardCardComponent{})
```

#### 粒子渲染层级分离

**两个渲染方法**：

1. **`RenderSystem.DrawGameWorldParticles()`** - 只渲染游戏世界粒子
   ```go
   // GameScene.Draw() - Layer 6
   s.renderSystem.DrawGameWorldParticles(screen, s.cameraX)
   // 过滤掉有 UIComponent 的粒子
   ```

2. **`RenderSystem.DrawParticles()`** - 渲染所有粒子
   ```go
   // RewardAnimationSystem.Draw() 内部调用
   ras.renderSystem.DrawParticles(screen, cameraOffsetX)
   // 包含 UI 粒子
   ```

#### 系统自管理渲染

**RewardAnimationSystem** 直接渲染自己的实体：

```go
func (ras *RewardAnimationSystem) Draw(screen *ebiten.Image) {
    // 1. 绘制 Reanim 实体
    ras.renderSystem.Draw(screen, cameraOffsetX)
    
    // 2. 绘制粒子效果（包含 UI 粒子）
    ras.renderSystem.DrawParticles(screen, cameraOffsetX)
    
    // 3. 直接渲染自己管理的卡片实体
    if card, ok := ecs.GetComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity); ok {
        if pos, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.rewardEntity); ok {
            entities.RenderPlantCard(screen, card, pos.X, pos.Y, ras.sunFont, ras.sunFontSize)
        }
    }
}
```

#### 系统自动过滤

**三个系统自动跳过奖励卡片**：

1. **PlantCardRenderSystem.Draw()** - 跳过渲染
   ```go
   // 跳过奖励卡片（由 RewardAnimationSystem 自行渲染）
   if _, isRewardCard := ecs.GetComponent[*components.RewardCardComponent](s.entityManager, entityID); isRewardCard {
       continue
   }
   ```

2. **PlantCardSystem.Update()** - 跳过状态更新
   ```go
   // 跳过奖励卡片（由 RewardAnimationSystem 管理）
   if _, isRewardCard := ecs.GetComponent[*components.RewardCardComponent](s.entityManager, entityID); isRewardCard {
       continue
   }
   ```

3. **InputSystem.handlePlantCardClick()** - 跳过点击处理
   ```go
   // 跳过奖励卡片（奖励卡片不应被点击选择）
   if _, isRewardCard := ecs.GetComponent[*components.RewardCardComponent](s.entityManager, entityID); isRewardCard {
       continue
   }
   ```

### 最终渲染顺序

```
GameScene.Draw():
  → Layer 1-5: 背景、植物、僵尸等
  → Layer 6: PlantCardRenderSystem.Draw()  (选择栏卡片)
  → Layer 6: renderSystem.DrawGameWorldParticles()  (游戏世界粒子)
  → Layer 7+: UI 层

RewardAnimationSystem.Draw() (独立渲染):
  → Reanim 实体
  → UI 粒子效果
  → 奖励卡片 (最上层)
```

### 架构对比

#### ❌ 旧架构（问题）
```
PlantCardRenderSystem.Draw()
  → 渲染所有卡片（包括奖励卡片）
  → 无法区分渲染层级
  → 需要复杂的过滤逻辑

GameScene.Draw()
  → renderSystem.DrawParticles() 
  → 渲染所有粒子（包括 UI 粒子）
  → 奖励粒子在错误的层级
```

#### ✅ 新架构（优化）
```
GameScene.Draw() - Layer 6
  → PlantCardRenderSystem.Draw() (自动过滤奖励卡片)
  → renderSystem.DrawGameWorldParticles() (只渲染游戏粒子)

RewardAnimationSystem.Draw() (完全封装)
  → 直接渲染自己的卡片实体
  → renderSystem.DrawParticles() (包含 UI 粒子)
  → 不影响其他系统
```

### 设计优势

1. **符合 ECS 原则** - 系统管理自己的实体
2. **低耦合** - 系统之间独立，互不影响
3. **高内聚** - 相关逻辑封装在同一个系统内
4. **易扩展** - 新增模块（如商店、图鉴）可以采用相同模式
5. **易维护** - 职责清晰，修改范围明确

### 扩展建议

**未来类似场景**：
- 图鉴界面：创建 `PlantAlmanacSystem`，自己管理和渲染卡片
- 商店界面：创建 `ShopSystem`，自己管理和渲染卡片
- 对战选择：创建 `BattleSelectSystem`，自己管理和渲染卡片

**统一模式**：
```go
// 1. 创建标记组件 (如 AlmanacCardComponent)
// 2. 系统内部直接渲染
// 3. 其他系统自动过滤
```

**相关文档**：`docs/stories/8.5.story.md`

## 编码规范

### 命名约定
| 元素 | 规范 | 示例 |
|------|------|------|
| 包名 | snake_case | render_system |
| 结构体/接口 | PascalCase | PositionComponent |
| 公开方法/函数 | PascalCase | Update() |
| 私有方法/函数 | camelCase | calculateDamage() |
| 变量 | camelCase | currentHealth |
| 常量 | PascalCase | DefaultZombieSpeed |
| 结构体字段 | PascalCase | X, Y float64 |

### 关键规则

1. **零耦合原则**: System 之间严禁直接调用,必须通过 EntityManager 或 EventBus 通信

2. **数据-行为分离**: Component 中严禁包含方法,所有逻辑在 System 中实现

3. **接口优先**: 函数签名优先使用接口而非具体类型

4. **错误处理**: 严禁忽略错误,必须检查所有可能返回 error 的函数
   ```go
   // 正确
   if err := doSomething(); err != nil {
       return fmt.Errorf("failed to do something: %w", err)
   }

   // 错误
   doSomething() // 忽略了可能的错误
   ```

5. **禁止全局变量**: 除了管理全局状态的单例(如 GameState),严禁使用全局变量。依赖通过构造函数注入。

6. **必须注释**: 所有公开的函数、方法、结构体和接口必须有 GoDoc 注释

7. **ECS 泛型 API 使用规范** (Epic 9):
   - **优先使用泛型 API**: 所有新代码和重构代码必须使用泛型 ECS API
   - **反射 API 已废弃**: `em.GetComponent()`, `em.GetEntitiesWith()` 等方法标记为 `@Deprecated`，仅用于向后兼容
   - **类型参数必须带 `*`**: 组件类型必须与存储时一致，例如 `GetComponent[*components.PlantComponent]`
   - **函数选择规则**: `GetEntitiesWithN` 的 N 必须等于组件数量（1-5）
   - **性能考虑**: 泛型 API 在大规模查询场景性能更优（10-13% 提升）
   
   ```go
   // ✅ 推荐：泛型 API
   plantComp, ok := ecs.GetComponent[*components.PlantComponent](em, entity)
   entities := ecs.GetEntitiesWith3[*Comp1, *Comp2, *Comp3](em)
   
   // ❌ 不推荐：反射 API（已废弃）
   comp, ok := em.GetComponent(entity, reflect.TypeOf(&components.PlantComponent{}))
   entities := em.GetEntitiesWith(reflect.TypeOf(&Comp1{}), ...)
   ```

### 代码格式化
- 提交前必须运行 `gofmt` 或 `goimports`
- 使用 `golangci-lint` 进行代码质量检查

## 测试策略

### 测试金字塔
- **单元测试**: 重点,覆盖率目标 80%+
- **集成测试**: 辅助,测试系统间交互
- **端到端测试**: 暂不考虑

### 测试文件组织
- 测试文件与源文件在同一包内
- 测试文件以 `_test.go` 结尾
- 使用 Go 标准库的 `testing` 包

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

### 坐标转换工具

使用 `pkg/utils/grid_utils.go` 中的函数：
```go
// 鼠标 → 网格
col, row, valid := utils.MouseToGridCoords(mouseX, mouseY, gs.CameraX, ...)

// 网格 → 屏幕
screenX, screenY := utils.GridToScreenCoords(col, row, gs.CameraX, ...)
```

**详细设计参见：** `docs/architecture/coordinate-system.md`

## 粒子系统使用指南

### 概述

粒子系统 (Story 7.2 + 7.3) 提供高性能的视觉特效渲染，用于爆炸、溅射、光效等游戏效果。

**架构组成：**
- **ParticleComponent** (Story 7.2): 单个粒子的数据（位置、速度、颜色、生命周期）
- **EmitterComponent** (Story 7.2): 粒子发射器配置（生成规则、限制、力场）
- **ParticleSystem** (Story 7.2): 更新粒子生命周期、动画插值
- **RenderSystem.DrawParticles()** (Story 7.3): 高性能批量渲染

**粒子配置来源：**
- `assets/reanim/particles/` 目录下的 XML 文件 (如 `Award.xml`, `Splash.xml`)
- 配置包含：发射规则、粒子属性、动画曲线、力场效果

## 数据驱动设计

### 关卡配置增强 (Story 8.1)

关卡配置系统已扩展，支持更多游戏玩法配置选项。

#### 关卡配置字段说明

```yaml
id: "1-1"
name: "前院白天 1-1"
description: "教学关卡：学习基本的植物种植和僵尸防御"

# Story 8.1: 新增配置字段
openingType: "tutorial"       # 开场类型："tutorial", "standard", "special"
enabledLanes: [3]             # 启用的行列表（1-5），如 [3] 表示只有第3行
availablePlants:              # 可用植物ID列表
  - "peashooter"
skipOpening: true             # 是否跳过开场动画（调试用）
tutorialSteps: []             # 教学步骤（Story 8.2 使用）
specialRules: ""              # 特殊规则："bowling", "conveyor"（Story 8.5/8.7 使用）

# 波次配置
waves:
  - time: 10
    zombies:
      - type: "basic"
        lane: 3
        count: 1
```

#### 字段详解

**openingType** - 控制关卡开场动画类型
- `"tutorial"`: 教学关卡（如 1-1），无开场动画，直接进入
- `"standard"`: 标准关卡，播放镜头右移预告僵尸动画
- `"special"`: 特殊关卡（如 1-5, 1-10），显示特殊标题卡
- 默认值：`"standard"`

**enabledLanes** - 启用的草坪行数
- 例如：`[3]` 表示只启用第3行（1-1 教学关卡）
- 例如：`[2, 3, 4]` 表示只启用中间3行（1-2, 1-3 关卡）
- 默认值：`[1, 2, 3, 4, 5]`（所有行）
- 用途：限制关卡场地，增加难度变化

**availablePlants** - 本关可用的植物ID列表
- 用于选卡界面显示可选植物
- 与 `PlantUnlockManager` 的交集为最终可选植物
- 例如：`["peashooter", "sunflower"]`
- 默认值：`[]`（空列表，表示所有已解锁植物）

**skipOpening** - 调试开关
- `true`: 跳过开场动画直接进入游戏
- `false`: 播放开场动画
- 默认值：`false`

**tutorialSteps** - 教学步骤配置（Story 8.2 使用）
```yaml
tutorialSteps:
  - trigger: "gameStart"
    text: "天空中会掉落阳光,点击收集它们!"
    action: "waitForSunCollect"
```

**specialRules** - 特殊规则类型（Story 8.5/8.7 使用）
- `"bowling"`: 坚果保龄球模式
- `"conveyor"`: 传送带模式
- 默认值：`""`（标准模式）

### 单位属性示例 (data/units/plants.yaml)
```yaml
peashooter:
  name: "豌豆射手"
  cost: 100
  health: 300
  damage: 20
  attack_speed: 1.4
  cooldown: 7.5
```

## 性能优化要点

1. **对象池**: 频繁创建/销毁的对象(如豌豆子弹)使用对象池
2. **避免动态分配**: 在游戏循环中避免频繁的内存分配
3. **批量处理**: System 应批量处理实体,而非逐个处理
4. **精灵图优化**: 使用纹理图集减少绘制调用

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

### 资源管理 API

#### 1. 初始化和配置加载

```go
// 在 main.go 中初始化
audioContext := audio.NewContext(48000)
rm := game.NewResourceManager(audioContext)

// 加载资源配置（必须在加载任何资源前调用）
if err := rm.LoadResourceConfig("assets/config/resources.yaml"); err != nil {
    log.Fatal("Failed to load resource config:", err)
}
```

#### 2. 通过资源 ID 加载资源（推荐）

```go
// 加载图片 - 通过资源 ID
img, err := rm.LoadImageByID("IMAGE_BACKGROUND1")
if err != nil {
    log.Printf("Failed to load image: %v", err)
}

// 获取已加载的图片
img = rm.GetImageByID("IMAGE_BACKGROUND1")
```

#### 3. 批量加载资源组

```go
// 加载整个资源组
if err := rm.LoadResourceGroup("init"); err != nil {
    log.Fatal("Failed to load init resources:", err)
}

// 加载游戏场景所需资源
if err := rm.LoadResourceGroup("loadingimages"); err != nil {
    log.Fatal("Failed to load game resources:", err)
}
```

#### 资源 ID 命名
- **图片**: `IMAGE_<NAME>` (例如: `IMAGE_BACKGROUND1`, `IMAGE_BLANK`)
- **Reanim图片**: `IMAGE_REANIM_<NAME>` (例如: `IMAGE_REANIM_SEEDS`)
- **音效**: `SOUND_<NAME>` (例如: `SOUND_BUTTONCLICK`)
- **音乐**: `MUSIC_<NAME>` (例如: `MUSIC_MAINMENU`)
- **字体**: `FONT_<NAME>` (例如: `FONT_HOUSEOFTERROR28`)

#### 文件路径规范
- 路径相对于 `base_path` (默认为 `assets`)
- 可以省略文件扩展名（系统会自动添加）
  - 图片默认 `.png`
  - 音效默认 `.ogg`

### 资源加载时序

```
游戏启动
  ↓
创建 ResourceManager
  ↓
LoadResourceConfig("assets/config/resources.yaml")  ← 必须第一步
  ↓
LoadResourceGroup("init")                            ← 加载初始资源
  ↓
场景切换时 LoadResourceGroup("specific_scene")      ← 按需加载
  ↓
使用 GetImageByID/GetAudioPlayer 获取缓存资源       ← 快速访问
```


### 故障排查

## 故障排查

### 常见问题

## 文档参考

项目相关文档位于 `docs/` 目录:
- `docs/prd.md`: 产品需求文档
- `docs/architecture.md`: 详细架构设计文档
- `docs/brief.md`: 项目简介
- `docs/front-end-spec.md`: 前端规范

## 特别注意事项

1. **忠实度优先**: 所有游戏数值(攻击力、生命值、冷却时间等)应与原版PC游戏保持一致

2. **模块化设计**: 代码应设计为便于未来添加新植物、僵尸或场景

3. **测试驱动**: 复杂逻辑实现前先编写单元测试

4. **Git 提交**: 每个story完成后，及时提交,保持提交历史清晰

## 开发顺序建议

按照 Epic 顺序开发(参考 docs/prd.md):
1. Epic 1: 游戏基础框架与主循环
2. Epic 2: 核心资源与玩家交互
3. Epic 3: 植物系统与部署
4. Epic 4: 基础僵尸与战斗逻辑
5. Epic 5: 游戏流程与高级单位

每个 Epic 包含多个 Story,建议按 Story 顺序逐步实现。

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
- 所有游戏实体的锚点策略要一致,都使用中心对齐
- 原版《植物大战僵尸》使用固定时间步长 **0.01秒（1厘秒）** 作为物理更新基准（相当于100FPS）。粒子配置文件中的某些值基于这个时间步长定义，而非真实的"每秒"单位。
- assets/effect 下的所有配置文件都不能修改
- 所有涉及大小 、位置的常量，都需要在配置常量文件中设置，以方便后续手工调整
- 如果要查看日志，需要添加参数 `--verbose`