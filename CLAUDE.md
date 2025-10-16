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

#### 5. 移除组件（RemoveComponent）

```go
// ❌ 旧方式（反射 API，已废弃）
em.RemoveComponent(zombieID, reflect.TypeOf(&components.VelocityComponent{}))

// ✅ 新方式（泛型 API，推荐）
ecs.RemoveComponent[*components.VelocityComponent](em, zombieID)
```

**特点**：
- 无需手动使用 `reflect.TypeOf`
- 代码更简洁
- 与其他泛型 API 风格一致

### 完整示例：BehaviorSystem 迁移

#### Before（反射版本）

```go
func (s *BehaviorSystem) Update(dt float64, gameState *game.GameState) {
    // 查询向日葵实体（冗长）
    sunflowerEntities := s.entityManager.GetEntitiesWith(
        reflect.TypeOf(&components.BehaviorComponent{}),
        reflect.TypeOf(&components.TimerComponent{}),
    )

    for _, entity := range sunflowerEntities {
        // 获取行为组件（需要类型断言）
        behaviorComp, ok := s.entityManager.GetComponent(entity, reflect.TypeOf(&components.BehaviorComponent{}))
        if !ok {
            continue
        }
        behavior := behaviorComp.(*components.BehaviorComponent) // 可能 panic

        if behavior.Type != components.BehaviorSunflower {
            continue
        }

        // 获取计时器组件（需要类型断言）
        timerComp, ok := s.entityManager.GetComponent(entity, reflect.TypeOf(&components.TimerComponent{}))
        if !ok {
            continue
        }
        timer := timerComp.(*components.TimerComponent) // 可能 panic

        // 更新计时器并生成阳光
        timer.Time += dt
        if timer.Time >= 24.0 {
            timer.Time = 0
            // 生成阳光逻辑...
        }
    }
}
```

#### After（泛型版本）

```go
func (s *BehaviorSystem) Update(dt float64, gameState *game.GameState) {
    // 查询向日葵实体（简洁）
    sunflowerEntities := ecs.GetEntitiesWith2[
        *components.BehaviorComponent,
        *components.TimerComponent,
    ](s.entityManager)

    for _, entity := range sunflowerEntities {
        // 获取行为组件（无需类型断言）
        behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entity)
        if !ok {
            continue
        }

        if behavior.Type != components.BehaviorSunflower {
            continue
        }

        // 获取计时器组件（无需类型断言）
        timer, ok := ecs.GetComponent[*components.TimerComponent](s.entityManager, entity)
        if !ok {
            continue
        }

        // 更新计时器并生成阳光
        timer.Time += dt
        if timer.Time >= 24.0 {
            timer.Time = 0
            // 生成阳光逻辑...
        }
    }
}
```

**改进点**：
- ✅ 删除了 4 处 `reflect.TypeOf()` 调用
- ✅ 删除了 2 处类型断言 `comp.(*T)`
- ✅ 代码更简洁，可读性提升
- ✅ 编译时类型检查，更安全

### 常见陷阱与解决方案

#### 陷阱 1: 忘记指针类型标记 `*`

```go
// ❌ 错误：忘记 * 符号
plantComp, ok := ecs.GetComponent[components.PlantComponent](em, entity)

// ✅ 正确：使用指针类型
plantComp, ok := ecs.GetComponent[*components.PlantComponent](em, entity)
```

**规则**：组件类型必须与存储时的类型完全一致（包括指针标记）。

#### 陷阱 2: GetEntitiesWith 函数选择错误

```go
// ❌ 错误：查询 3 个组件，但使用了 GetEntitiesWith2
entities := ecs.GetEntitiesWith2[
    *components.BehaviorComponent,
    *components.PlantComponent,
    *components.PositionComponent, // 第 3 个组件被忽略！
](em)

// ✅ 正确：查询 3 个组件，使用 GetEntitiesWith3
entities := ecs.GetEntitiesWith3[
    *components.BehaviorComponent,
    *components.PlantComponent,
    *components.PositionComponent,
](em)
```

**规则**：函数名末尾数字 N = 类型参数数量。

#### 陷阱 3: 超过 5 个组件的查询

如果需要查询超过 5 个组件：

**解决方案 A**：使用反射 API（保留向后兼容）
```go
entities := em.GetEntitiesWith(
    reflect.TypeOf(&components.Comp1{}),
    // ... 6+ 个组件
)
```

**解决方案 B**：分步查询
```go
// 先查询前 5 个组件
entities := ecs.GetEntitiesWith5[*Comp1, *Comp2, *Comp3, *Comp4, *Comp5](em)

// 再过滤第 6 个组件
result := make([]ecs.EntityID, 0)
for _, entity := range entities {
    if ecs.HasComponent[*Comp6](em, entity) {
        result = append(result, entity)
    }
}
```

**解决方案 C** （推荐）：重新设计组件
- 如果需要查询超过 5 个组件，可能说明组件设计过于碎片化
- 考虑合并相关组件或使用组合组件

### 性能对比

基于 Intel i9-14900KF 的基准测试结果（Story 9.1 & 9.3）：

| 操作 | 反射版本 | 泛型版本 | 性能提升 |
|------|---------|---------|---------|
| **查询 1000 实体（3组件）** | 95.7 μs | 86.2 μs | **10.0% ⬆️** |
| **查询 1000 实体（5组件）** | 90.0 μs | 78.3 μs | **13.0% ⬆️** |
| **获取单个组件** | 7.5 ns | 10.6 ns | -41.3% ⬇️ |
| **添加组件** | 168.5 ns | 172.2 ns | -2.2% ⬇️ |

**性能分析**：
- ✅ **大规模查询场景**：泛型版本显著更快（10-13% 提升）
- ⚠️ **单组件操作**：反射版本略快（编译器优化）
- ✅ **综合场景**：泛型版本整体性能提升约 10%

**主要优势在于类型安全和代码可读性，而非纯粹性能提升。**

### 迁移指南

完整的迁移指南参见：`docs/architecture/ecs-generics-migration-guide.md`

**迁移检查清单**：
- [ ] 替换所有 `GetComponent` 调用为泛型版本
- [ ] 替换所有 `AddComponent` 调用为泛型版本
- [ ] 替换所有 `HasComponent` 调用为泛型版本
- [ ] 替换所有 `GetEntitiesWith` 调用为泛型版本
- [ ] 删除所有 `reflect.TypeOf()` 调用
- [ ] 删除所有类型断言 `comp.(*T)`
- [ ] 移除不再需要的 `import "reflect"`
- [ ] 运行测试验证功能正确性

### 相关文档

- **迁移指南**：`docs/architecture/ecs-generics-migration-guide.md`
- **性能报告**：`docs/architecture/ecs-generics-performance-report.md`
- **ECS 源码**：`pkg/ecs/entity_manager.go`
- **Story 9.1**：泛型 API 设计与原型
- **Story 9.2**：系统批量迁移（18 个系统文件）
- **Story 9.3**：全面测试与文档更新

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

### 判断规则

**快速判断流程**：
```
问：这是游戏玩法实体（植物/僵尸/子弹）吗？
└─ 是 → 使用 ReanimComponent

问：这是 UI 元素（卡片/按钮/预览）吗？
└─ 是 → 检查是否需要动画
    └─ 需要动画（植物预览）→ ReanimComponent
    └─ 不需要动画（卡片）→ SpriteComponent

有疑问？检查实体是否有 UIComponent 或 PlantCardComponent
```

### 辅助函数

- `createSimpleReanimComponent(image, name)`: 将单图片包装为 ReanimComponent
  - 用于：阳光、子弹、简单特效
  - 目的：保持渲染管线一致性，避免混合两种渲染路径

### 相关文档

- **架构决策记录**：`docs/architecture/adr/001-component-strategy.md`（如有）
- **Story 6.3 详细说明**：`docs/stories/6.3.story.md`
- **渲染系统文档**：`pkg/systems/render_system.go`（文件头部注释）

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

### 测试重点
- 核心逻辑包: `systems`, `components` 需达到 80% 覆盖率
- UI 和场景相关包无强制要求
- 重点测试独立的、无副作用的函数和算法

### 运行单个测试
```bash
# 运行特定包的测试
go test ./pkg/systems

# 运行特定测试函数
go test -run TestDamageCalculation ./pkg/systems

# 运行测试并查看详细输出
go test -v ./pkg/systems
```

## 开发工作流程

### 添加新植物类型

1. **定义组件数据** (pkg/components/):
   ```go
   // 如果需要新的特殊属性,创建新组件
   type SpecialAbilityComponent struct {
       AbilityType string
       Cooldown float64
   }
   ```

2. **更新 BehaviorType** (pkg/components/behavior.go):
   ```go
   const (
       // ... 现有类型
       BehaviorNewPlant BehaviorType = iota
   )
   ```

3. **创建实体工厂** (pkg/entities/plant_factory.go):
   ```go
   func NewPlantEntity(manager *ecs.EntityManager, plantType BehaviorType) EntityID {
       // 创建实体并添加所需组件
   }
   ```

4. **实现行为逻辑** (pkg/systems/behavior_system.go):
   ```go
   // 在 BehaviorSystem.Update() 中添加新植物的行为处理
   ```

5. **添加测试** (pkg/systems/behavior_system_test.go)

6. **配置数据** (data/units/plants.yaml):
   ```yaml
   newplant:
     name: "新植物"
     cost: 100
     health: 300
     # ... 其他属性
   ```

### 添加新僵尸类型
流程与添加新植物类似,关注点在僵尸特定的行为逻辑(移动、啃食)。

## 关键工作流程

### 玩家收集阳光
1. InputSystem 检测鼠标点击阳光实体
2. GameState.AddSun(25) 更新阳光数量
3. InputSystem 标记阳光实体待删除
4. UISystem 读取 GameState 并更新 UI 显示

### 豌豆射手攻击
1. BehaviorSystem 查询豌豆射手和同行僵尸
2. 如有僵尸,BehaviorSystem 创建豌豆子弹实体
3. PhysicsSystem 移动子弹并检测碰撞
4. 碰撞时发布 CollisionEvent
5. DamageSystem 处理事件,减少僵尸生命值
6. BehaviorSystem 检测生命值<=0,标记僵尸待删除

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

### 性能特性

**Story 7.3 性能测试结果：**
- ✅ 1000 粒子渲染：**0.22ms** (目标 <3ms，超出 14 倍)
- ✅ 顶点数组复用：预分配 4000 顶点，6000 索引
- ✅ 批量渲染：按混合模式分组，减少 DrawTriangles 调用
- ✅ 测试覆盖率：buildParticleVertices 100%, DrawParticles 89.1%

### 创建粒子效果

```go
// 使用 ParticleFactory 创建粒子效果
effectID := entities.CreateParticleEffect(
    entityManager,
    x, y,                    // 位置
    particleConfig,          // 粒子配置 (从 XML 加载)
    resourceManager,         // 资源管理器
)
```

**粒子配置来源：**
- `assets/reanim/particles/` 目录下的 XML 文件 (如 `Award.xml`, `Splash.xml`)
- 配置包含：发射规则、粒子属性、动画曲线、力场效果

### 渲染特性

**支持的粒子属性：**
- **位置 & 运动**: X, Y, VelocityX, VelocityY
- **变换**: Rotation, Scale
- **颜色**: Red, Green, Blue, Alpha, Brightness
- **混合模式**: Additive（加法混合）vs Normal（普通混合）

**混合模式说明：**
| 混合模式 | 效果 | 使用场景 |
|---------|------|---------|
| Normal (Alpha Blending) | 半透明叠加 | 烟雾、灰尘 |
| Additive Blending | 发光叠加（重叠更亮） | 爆炸、火焰、光效 |

**渲染层级：**
```
GameScene.Draw() 渲染顺序：
  1. 背景
  2. 游戏世界（植物、僵尸、子弹）
  3. UI（植物卡片）
  4. 粒子效果 ← 在 UI 之上，阳光之下
  5. 植物预览
  6. 阳光（最顶层）
```

### 粒子生命周期

```
创建粒子
  ↓
ParticleSystem.Update()
  ├── 更新年龄 (Age += deltaTime)
  ├── 插值动画 (Alpha, Scale, Spin)
  ├── 应用力场 (加速度、摩擦力)
  └── 检查生命周期 (Age > Lifetime → 删除)
  ↓
RenderSystem.DrawParticles()
  ├── 查询所有粒子实体
  ├── 按混合模式分组
  ├── 生成顶点（变换、颜色）
  └── 批量渲染
```

### 技术细节

**顶点生成流程：**
1. 计算粒子矩形四角（中心对齐）
2. 应用旋转变换（旋转矩阵）
3. 应用缩放变换
4. 平移到世界位置
5. 转换为屏幕坐标（减去 cameraX）
6. 设置顶点颜色（RGB * Brightness, Alpha）

**批量渲染优化：**
- 按混合模式分组（先 Normal，再 Additive）
- 同一批次合并为一次 DrawTriangles 调用
- 顶点数组每帧重置而非重新分配

### 使用示例

```go
// 在僵尸死亡时创建爆炸粒子效果
if zombie.Health <= 0 {
    // 加载粒子配置
    config := resourceManager.GetParticleConfig("Explosion")

    // 创建粒子效果实体
    effectID := entities.CreateParticleEffect(
        entityManager,
        zombie.X, zombie.Y,  // 僵尸位置
        config,
        resourceManager,
    )

    // 粒子系统会自动管理生命周期
    // 渲染系统会自动渲染粒子
}
```

**参考文档：**
- Story 7.2: `docs/stories/7.2.story.md` (粒子系统核心)
- Story 7.3: `docs/stories/7.3.story.md` (粒子渲染)
- 测试文件: `pkg/systems/render_system_particle_test.go`

## 数据驱动设计

### 关卡配置示例 (data/levels/level_1-1.yaml)
```yaml
level:
  id: "1-1"
  name: "前院白天 1-1"
  waves:
    - time: 10
      zombies:
        - type: basic
          lane: 3
          count: 1
    - time: 30
      zombies:
        - type: basic
          lane: 1
          count: 2
  # ...
```

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

#### 4. 传统路径方式加载（向后兼容）

```go
// 旧方式：通过硬编码路径加载（仍然支持，但不推荐新代码使用）
img, err := rm.LoadImage("assets/images/background1.jpg")
```

### 资源类型

#### 图片资源
- **简单图片**: 单张图片文件
  ```yaml
  - id: IMAGE_BACKGROUND1
    path: images/background1.jpg
  ```

- **精灵图（Sprite Sheet）**: 包含多个子图像的图集
  ```yaml
  - id: IMAGE_REANIM_SEEDS
    path: reanim/seeds.png
    cols: 9  # 9列
    rows: 1  # 1行（可选，默认为1）
  ```

#### 音频资源
- **背景音乐**: 自动循环播放
  ```go
  player, err := rm.LoadAudio("assets/audio/Music/mainmenubgm.mp3")
  player.Play() // 无限循环
  ```

- **音效**: 单次播放
  ```go
  player, err := rm.LoadSoundEffect("assets/audio/Sound/points.ogg")
  player.Play() // 播放一次
  ```

#### Reanim 动画资源
- 自动加载 Reanim XML 和部件图片
  ```go
  if err := rm.LoadReanimResources(); err != nil {
      log.Fatal(err)
  }

  // 获取 Reanim 数据
  reanimXML := rm.GetReanimXML("PeaShooter")
  partImages := rm.GetReanimPartImages("PeaShooter")
  ```

### 资源命名规范

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

### 最佳实践

1. **优先使用资源 ID**: 新代码应使用 `LoadImageByID()` 而不是硬编码路径
   ```go
   // 推荐 ✅
   img, err := rm.LoadImageByID("IMAGE_BACKGROUND1")

   // 不推荐 ❌
   img, err := rm.LoadImage("assets/images/background1.jpg")
   ```

2. **批量加载**: 在场景切换时使用 `LoadResourceGroup()` 批量加载
   ```go
   // 进入游戏场景时
   if err := rm.LoadResourceGroup("delayload_background1"); err != nil {
       return err
   }
   ```

3. **资源复用**: ResourceManager 自动缓存，同一资源不会重复加载
   ```go
   img1, _ := rm.LoadImageByID("IMAGE_BACKGROUND1") // 从文件加载
   img2 := rm.GetImageByID("IMAGE_BACKGROUND1")     // 从缓存获取（更快）
   // img1 和 img2 指向同一对象
   ```

4. **错误处理**: 始终检查资源加载错误
   ```go
   img, err := rm.LoadImageByID("IMAGE_BACKGROUND1")
   if err != nil {
       return fmt.Errorf("failed to load background: %w", err)
   }
   ```

### 添加新资源

#### 步骤 1: 将资源文件放到正确的目录
```bash
# 图片
assets/images/your_image.png

# Reanim 部件
assets/reanim/your_part.png

# 音效
assets/sounds/your_sound.ogg
```

#### 步骤 2: 在 `assets/config/resources.yaml` 中添加定义
```yaml
groups:
  your_group:
    images:
      - id: IMAGE_YOUR_IMAGE
        path: images/your_image.png
    sounds:
      - id: SOUND_YOUR_SOUND
        path: sounds/your_sound.ogg
```

#### 步骤 3: 在代码中使用
```go
// 单独加载
img, err := rm.LoadImageByID("IMAGE_YOUR_IMAGE")

// 或批量加载整组
rm.LoadResourceGroup("your_group")
```

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

### 迁移指南（从旧系统到新系统）

**旧代码（硬编码路径）:**
```go
img, err := rm.LoadImage("assets/images/background1.jpg")
```

**新代码（资源 ID）:**
```go
img, err := rm.LoadImageByID("IMAGE_BACKGROUND1")
```

**迁移清单：**
- [ ] 在 `resources.yaml` 中为资源定义 ID
- [ ] 将 `LoadImage(path)` 替换为 `LoadImageByID(id)`
- [ ] 将 `GetImage(path)` 替换为 `GetImageByID(id)`
- [ ] 测试验证资源能正常加载

### 故障排查

**问题 1: "resource config not loaded"**
- **原因**: 未调用 `LoadResourceConfig()`
- **解决**: 在 `main.go` 中确保在加载任何资源前调用此方法

**问题 2: "resource ID not found: IMAGE_XXX"**
- **原因**: 资源 ID 未在 `resources.yaml` 中定义
- **解决**: 检查配置文件，添加缺失的资源定义

**问题 3: "failed to open image file"**
- **原因**: 文件路径错误或文件不存在
- **解决**: 验证 `path` 相对于 `base_path` 的路径正确性

### 资源缓存策略

- **图片**: 加载一次，永久缓存（直到程序退出）
- **音频**: 加载一次，永久缓存
- **字体**: 按 `(path, size)` 组合缓存
- **Reanim**: XML 和部件图片分别缓存

**注意**: 缓存使用标准 Go map，非线程安全。当前单线程游戏循环无需考虑同步。

## 故障排查

### 常见问题

**窗口无法创建**:
- 检查 Ebitengine 是否正确安装: `go get github.com/hajimehoshi/ebiten/v2`
- 确认图形驱动已更新

**资源加载失败**:
- 验证资源文件路径正确
- 检查工作目录是否为项目根目录

**性能下降**:
- 使用 Go pprof 进行性能分析: `go test -cpuprofile=cpu.prof -bench .`
- 检查是否有频繁的内存分配
- 验证渲染批次数量

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

4. **Git 提交**: 每个功能点完成后及时提交,保持提交历史清晰

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