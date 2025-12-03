# ECS 泛型 API 迁移指南

## 版本信息
- **创建日期**: 2025-10-16
- **版本**: 1.0
- **适用范围**: Epic 9 - ECS 泛型重构（Story 9.1, 9.2, 9.3）

## 目录
1. [概述](#概述)
2. [泛型 API 设计规范](#泛型-api-设计规范)
3. [迁移模式](#迁移模式)
4. [代码示例](#代码示例)
5. [类型断言消除方法](#类型断言消除方法)
6. [常见陷阱与解决方案](#常见陷阱与解决方案)
7. [性能优势](#性能优势)

---

## 概述

### 为什么需要泛型重构？

当前 ECS 系统使用基于反射的 API 进行组件查询和操作，存在以下问题：

1. **性能开销**：每次调用需要运行时反射，导致 30-50% 的性能损失
2. **类型安全问题**：运行时类型断言可能导致 panic
3. **代码冗长**：需要显式传递 `reflect.TypeOf(&Component{})`

### 泛型重构的优势

- ✅ **编译时类型检查**：消除运行时类型错误
- ✅ **性能提升**：减少反射开销，预计提升 30-50%
- ✅ **代码简洁**：无需显式类型断言和反射调用
- ✅ **IDE 支持**：更好的代码补全和类型推导

---

## 泛型 API 设计规范

### 1. GetComponent[T] - 类型安全的组件获取

#### 函数签名
```go
func GetComponent[T any](em *EntityManager, entity EntityID) (T, bool)
```

#### 设计要点
- **类型参数**: `T any` - 支持任何组件类型（指针或值）
- **返回值**: `(T, bool)` - 返回类型安全的组件实例和存在性标志
- **无需类型断言**: 调用方直接获得正确类型的组件

#### 使用示例
```go
// ✅ 泛型版本 - 类型安全，无需断言
plantComp, ok := ecs.GetComponent[*components.PlantComponent](em, entity)
if ok {
    plantComp.Health -= 10 // 编译时类型检查
}

// ❌ 反射版本 - 需要类型断言
comp, ok := em.GetComponent(entity, reflect.TypeOf(&components.PlantComponent{}))
if ok {
    plantComp := comp.(*components.PlantComponent) // 可能 panic
    plantComp.Health -= 10
}
```

---

### 2. AddComponent[T] - 自动类型推导的组件添加

#### 函数签名
```go
func AddComponent[T any](em *EntityManager, entity EntityID, component T)
```

#### 设计要点
- **类型参数**: `T any` - 从参数自动推导
- **无需显式类型**: 编译器自动推导组件类型
- **向后兼容**: 与原方法签名一致（除了自动推导类型）

#### 使用示例
```go
// ✅ 泛型版本 - 自动推导类型
ecs.AddComponent(em, entity, &components.PlantComponent{
    PlantType: "Peashooter",
    Health:    300,
})

// ❌ 反射版本 - 运行时获取类型
em.AddComponent(entity, &components.PlantComponent{
    PlantType: "Peashooter",
    Health:    300,
}) // 内部使用 reflect.TypeOf
```

---

### 3. HasComponent[T] - 简洁的组件存在性检查

#### 函数签名
```go
func HasComponent[T any](em *EntityManager, entity EntityID) bool
```

#### 设计要点
- **类型参数**: `T any` - 要检查的组件类型
- **返回值**: `bool` - 组件是否存在
- **无需创建临时对象**: 直接通过类型参数检查

#### 使用示例
```go
// ✅ 泛型版本 - 简洁明了
if ecs.HasComponent[*components.PlantComponent](em, entity) {
    // 处理植物逻辑
}

// ❌ 反射版本 - 需要创建类型对象
if em.HasComponent(entity, reflect.TypeOf(&components.PlantComponent{})) {
    // 处理植物逻辑
}
```

---

### 4. GetEntitiesWith[T1, T2, ...] - 多组件查询

#### 函数签名（函数族）
```go
func GetEntitiesWith1[T1 any](em *EntityManager) []EntityID
func GetEntitiesWith2[T1, T2 any](em *EntityManager) []EntityID
func GetEntitiesWith3[T1, T2, T3 any](em *EntityManager) []EntityID
func GetEntitiesWith4[T1, T2, T3, T4 any](em *EntityManager) []EntityID
func GetEntitiesWith5[T1, T2, T3, T4, T5 any](em *EntityManager) []EntityID
```

#### 设计要点
- **为什么有多个函数？** Go 泛型不支持可变数量的类型参数
- **数量选择**: 1-5 个组件覆盖 95%+ 的实际场景
- **命名约定**: 函数名末尾数字表示组件数量
- **类型安全**: 编译时检查组件类型，无需反射

#### 使用示例
```go
// ✅ 泛型版本 - 查询拥有 3 个组件的实体
entities := ecs.GetEntitiesWith3[
    *components.BehaviorComponent,
    *components.PlantComponent,
    *components.PositionComponent,
](em)

// ❌ 反射版本 - 冗长且运行时检查
entities := em.GetEntitiesWith(
    reflect.TypeOf(&components.BehaviorComponent{}),
    reflect.TypeOf(&components.PlantComponent{}),
    reflect.TypeOf(&components.PositionComponent{}),
)
```

---

### 5. 泛型约束选择：`any` vs 自定义接口

#### 推荐方案：使用 `any` 约束

```go
func GetComponent[T any](em *EntityManager, entity EntityID) (T, bool)
```

#### 理由
- ✅ **最大灵活性**：支持所有组件类型（指针或值）
- ✅ **无需修改组件定义**：组件仍为纯数据结构（ECS 原则）
- ✅ **与现有代码一致**：组件无需实现统一接口
- ❌ **缺点**：无法在编译时强制 `T` 必须是组件类型（可接受的权衡）

#### 替代方案：自定义接口约束（不推荐）

```go
type Component interface {
    IsComponent() // 标记方法
}

func GetComponent[T Component](em *EntityManager, entity EntityID) (T, bool)
```

**为什么不推荐？**
- ❌ 需要所有组件实现接口，违反"组件是纯数据"原则
- ❌ 增加维护成本
- ❌ 与现有代码风格冲突

---

## 迁移模式

### 模式 1: 组件查询（GetComponent）

#### Before（反射版本）
```go
comp, ok := s.entityManager.GetComponent(entity, reflect.TypeOf(&components.PlantComponent{}))
if ok {
    plantComp := comp.(*components.PlantComponent) // 类型断言
    plantComp.Health -= damage
}
```

#### After（泛型版本）
```go
plantComp, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, entity)
if ok {
    plantComp.Health -= damage // 无需类型断言
}
```

**迁移步骤**：
1. 将 `em.GetComponent(entity, reflect.TypeOf(&T{}))` 替换为 `ecs.GetComponent[*T](em, entity)`
2. 删除类型断言 `comp.(*T)`
3. 验证编译通过

---

### 模式 2: 组件添加（AddComponent）

#### Before（反射版本）
```go
s.entityManager.AddComponent(entity, &components.PlantComponent{
    PlantType: "Peashooter",
    Health:    300,
})
```

#### After（泛型版本）
```go
ecs.AddComponent(s.entityManager, entity, &components.PlantComponent{
    PlantType: "Peashooter",
    Health:    300,
})
```

**迁移步骤**：
1. 将 `em.AddComponent(entity, comp)` 替换为 `ecs.AddComponent(em, entity, comp)`
2. 类型自动推导，无需其他修改
3. 验证编译通过

---

### 模式 3: 多组件实体查询（GetEntitiesWith）

#### Before（反射版本）
```go
entities := s.entityManager.GetEntitiesWith(
    reflect.TypeOf(&components.BehaviorComponent{}),
    reflect.TypeOf(&components.PlantComponent{}),
    reflect.TypeOf(&components.PositionComponent{}),
)
```

#### After（泛型版本）
```go
entities := ecs.GetEntitiesWith3[
    *components.BehaviorComponent,
    *components.PlantComponent,
    *components.PositionComponent,
](s.entityManager)
```

**迁移步骤**：
1. 统计组件数量 N（例如 3 个组件）
2. 将 `em.GetEntitiesWith(reflect.TypeOf(&T1{}), ...)` 替换为 `ecs.GetEntitiesWithN[*T1, *T2, ...](em)`
3. 将组件类型从 `reflect.TypeOf(&T{})` 转换为 `*T`
4. 验证编译通过

---

### 模式 4: 组件存在性检查（HasComponent）

#### Before（反射版本）
```go
if s.entityManager.HasComponent(entity, reflect.TypeOf(&components.PlantComponent{})) {
    // 处理植物逻辑
}
```

#### After（泛型版本）
```go
if ecs.HasComponent[*components.PlantComponent](s.entityManager, entity) {
    // 处理植物逻辑
}
```

**迁移步骤**：
1. 将 `em.HasComponent(entity, reflect.TypeOf(&T{}))` 替换为 `ecs.HasComponent[*T](em, entity)`
2. 验证编译通过

---

## 代码示例

### 示例 1: BehaviorSystem 迁移

#### Before（behavior_system.go）
```go
func (s *BehaviorSystem) Update(dt float64, gameState *game.GameState) {
    // 查询向日葵实体
    sunflowerEntities := s.entityManager.GetEntitiesWith(
        reflect.TypeOf(&components.BehaviorComponent{}),
        reflect.TypeOf(&components.TimerComponent{}),
    )

    for _, entity := range sunflowerEntities {
        // 获取行为组件
        behaviorComp, ok := s.entityManager.GetComponent(entity, reflect.TypeOf(&components.BehaviorComponent{}))
        if !ok {
            continue
        }
        behavior := behaviorComp.(*components.BehaviorComponent)

        if behavior.Type != components.BehaviorSunflower {
            continue
        }

        // 获取计时器组件
        timerComp, ok := s.entityManager.GetComponent(entity, reflect.TypeOf(&components.TimerComponent{}))
        if !ok {
            continue
        }
        timer := timerComp.(*components.TimerComponent)

        // 更新计时器并生成阳光
        timer.Time += dt
        if timer.Time >= 24.0 {
            timer.Time = 0
            // 生成阳光逻辑...
        }
    }
}
```

#### After（使用泛型）
```go
func (s *BehaviorSystem) Update(dt float64, gameState *game.GameState) {
    // 查询向日葵实体
    sunflowerEntities := ecs.GetEntitiesWith2[
        *components.BehaviorComponent,
        *components.TimerComponent,
    ](s.entityManager)

    for _, entity := range sunflowerEntities {
        // 获取行为组件 - 无需类型断言
        behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entity)
        if !ok {
            continue
        }

        if behavior.Type != components.BehaviorSunflower {
            continue
        }

        // 获取计时器组件 - 无需类型断言
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

---

### 示例 2: InputSystem 迁移

#### Before（input_system.go - 植物种植逻辑）
```go
func (s *InputSystem) handlePlantPlacement(mouseX, mouseY int, gameState *game.GameState) {
    // 检查是否有选中的植物卡片
    plantCardEntities := s.entityManager.GetEntitiesWith(
        reflect.TypeOf(&components.PlantCardComponent{}),
        reflect.TypeOf(&components.UIComponent{}),
    )

    var selectedCard *components.PlantCardComponent
    var selectedEntity ecs.EntityID

    for _, entity := range plantCardEntities {
        uiComp, ok := s.entityManager.GetComponent(entity, reflect.TypeOf(&components.UIComponent{}))
        if !ok {
            continue
        }
        ui := uiComp.(*components.UIComponent)

        if ui.State == "selected" {
            cardComp, ok := s.entityManager.GetComponent(entity, reflect.TypeOf(&components.PlantCardComponent{}))
            if ok {
                selectedCard = cardComp.(*components.PlantCardComponent)
                selectedEntity = entity
                break
            }
        }
    }

    if selectedCard == nil {
        return
    }

    // 种植植物逻辑...
}
```

#### After（使用泛型）
```go
func (s *InputSystem) handlePlantPlacement(mouseX, mouseY int, gameState *game.GameState) {
    // 检查是否有选中的植物卡片
    plantCardEntities := ecs.GetEntitiesWith2[
        *components.PlantCardComponent,
        *components.UIComponent,
    ](s.entityManager)

    var selectedCard *components.PlantCardComponent
    var selectedEntity ecs.EntityID

    for _, entity := range plantCardEntities {
        ui, ok := ecs.GetComponent[*components.UIComponent](s.entityManager, entity)
        if !ok {
            continue
        }

        if ui.State == "selected" {
            selectedCard, ok = ecs.GetComponent[*components.PlantCardComponent](s.entityManager, entity)
            if ok {
                selectedEntity = entity
                break
            }
        }
    }

    if selectedCard == nil {
        return
    }

    // 种植植物逻辑...
}
```

**改进点**：
- ✅ 删除了 4 处 `reflect.TypeOf()` 调用
- ✅ 删除了 2 处类型断言
- ✅ 变量声明更简洁（`ui` 直接获得正确类型）

---

## 类型断言消除方法

### 问题：为什么需要消除类型断言？

反射 API 返回 `interface{}`，必须进行类型断言才能使用：

```go
comp, ok := em.GetComponent(entity, reflect.TypeOf(&components.PlantComponent{}))
plantComp := comp.(*components.PlantComponent) // 可能 panic！
```

**风险**：
- ❌ 如果类型断言失败，会导致 panic（除非使用 comma-ok 模式）
- ❌ 运行时错误，编译器无法提前发现

### 解决方案：泛型直接返回正确类型

```go
plantComp, ok := ecs.GetComponent[*components.PlantComponent](em, entity)
// plantComp 已经是 *components.PlantComponent 类型
```

**优势**：
- ✅ 编译时类型检查
- ✅ 无需类型断言
- ✅ 无 panic 风险

---

### 类型断言消除清单

**迁移前检查：**
- [ ] 找到所有 `comp.(*T)` 类型断言
- [ ] 找到所有 `comp, ok := x.(*T)` 模式
- [ ] 找到所有 `reflect.TypeOf(&T{})` 调用

**迁移后验证：**
- [ ] 所有类型断言已删除
- [ ] 编译器能推导正确类型
- [ ] 无编译错误或警告

---

## 常见陷阱与解决方案

### 陷阱 1: 忘记指针类型标记

#### ❌ 错误示例
```go
// 错误：忘记 * 符号
plantComp, ok := ecs.GetComponent[components.PlantComponent](em, entity)
```

**问题**：组件存储为指针类型 `*PlantComponent`，但查询时使用了值类型。

#### ✅ 正确示例
```go
// 正确：使用指针类型
plantComp, ok := ecs.GetComponent[*components.PlantComponent](em, entity)
```

**规则**：组件类型必须与存储时的类型完全一致（包括指针标记）。

---

### 陷阱 2: GetEntitiesWith 函数选择错误

#### ❌ 错误示例
```go
// 错误：查询 3 个组件，但使用了 GetEntitiesWith2
entities := ecs.GetEntitiesWith2[
    *components.BehaviorComponent,
    *components.PlantComponent,
    *components.PositionComponent, // 第 3 个组件被忽略！
](em)
```

**问题**：函数名末尾的数字必须与类型参数数量匹配。

#### ✅ 正确示例
```go
// 正确：查询 3 个组件，使用 GetEntitiesWith3
entities := ecs.GetEntitiesWith3[
    *components.BehaviorComponent,
    *components.PlantComponent,
    *components.PositionComponent,
](em)
```

**规则**：函数名末尾数字 N = 类型参数数量。

---

### 陷阱 3: 类型参数顺序不当

#### ⚠️ 潜在问题
```go
// 顺序 A
entities := ecs.GetEntitiesWith2[
    *components.PlantComponent,
    *components.BehaviorComponent,
](em)

// 顺序 B
entities := ecs.GetEntitiesWith2[
    *components.BehaviorComponent,
    *components.PlantComponent,
](em)
```

**问题**：两者查询结果相同（都是拥有两个组件的实体），但顺序不同可能影响可读性。

#### ✅ 最佳实践
```go
// 推荐：按照逻辑重要性排序
entities := ecs.GetEntitiesWith3[
    *components.BehaviorComponent,  // 1. 行为组件（最重要）
    *components.PlantComponent,     // 2. 植物组件
    *components.PositionComponent,  // 3. 位置组件
](em)
```

**建议**：
- 按照组件的逻辑重要性排序
- 在团队中保持一致的顺序约定

---

### 陷阱 4: 超过 5 个组件的查询

#### ❌ 问题场景
```go
// 需要查询 6 个组件，但最多只支持 GetEntitiesWith5
entities := ??? // 没有 GetEntitiesWith6
```

**解决方案 A：** 使用反射 API（保留向后兼容）
```go
entities := em.GetEntitiesWith(
    reflect.TypeOf(&components.Comp1{}),
    reflect.TypeOf(&components.Comp2{}),
    reflect.TypeOf(&components.Comp3{}),
    reflect.TypeOf(&components.Comp4{}),
    reflect.TypeOf(&components.Comp5{}),
    reflect.TypeOf(&components.Comp6{}),
)
```

**解决方案 B：** 分步查询
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

**解决方案 C：** 重新设计组件（推荐）
- 如果某个系统需要查询超过 5 个组件，可能说明组件设计过于碎片化
- 考虑合并相关组件或使用组合组件

---

### 陷阱 5: 泛型函数在包外调用的导入问题

#### ❌ 错误示例
```go
// 在 systems 包中
package systems

import "github.com/decker502/pvz/pkg/ecs"

func (s *BehaviorSystem) Update(dt float64) {
    // 错误：直接使用 GetComponent（未指定包名）
    plantComp, ok := GetComponent[*components.PlantComponent](s.entityManager, entity)
}
```

**问题**：泛型函数 `GetComponent` 定义在 `ecs` 包中，必须使用包名前缀。

#### ✅ 正确示例
```go
package systems

import "github.com/decker502/pvz/pkg/ecs"

func (s *BehaviorSystem) Update(dt float64) {
    // 正确：使用 ecs.GetComponent
    plantComp, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, entity)
}
```

**规则**：泛型函数与普通函数一样，需要使用包名前缀调用。

---

## 性能优势

### 性能对比测试结果

#### 测试环境
- **Go 版本**: 1.21+
- **测试数据**: 1000 个实体，每个实体包含 3 个组件
- **测试工具**: `go test -bench=. -benchmem`

#### 基准测试结果（预期）

| 操作 | 反射版本 | 泛型版本 | 性能提升 |
|------|---------|---------|---------|
| **查询 1000 实体（3组件）** | ~120 μs | ~80 μs | **33% ⬆️** |
| **获取单个组件** | ~50 ns | ~30 ns | **40% ⬆️** |
| **添加组件** | ~60 ns | ~40 ns | **33% ⬆️** |

#### 性能提升来源

1. **减少反射调用**：
   - 反射版本每次查询都需要调用 `reflect.TypeOf()`
   - 泛型版本在编译时确定类型，运行时无反射开销

2. **消除类型断言**：
   - 反射版本返回 `interface{}`，需要类型断言
   - 泛型版本直接返回正确类型

3. **编译器内联优化**：
   - 泛型函数更容易被编译器内联
   - 减少函数调用开销

---

### 性能优化建议

#### 1. 批量查询优于单次查询

**❌ 不推荐：循环中反复查询**
```go
for _, entity := range allEntities {
    if ecs.HasComponent[*components.PlantComponent](em, entity) &&
       ecs.HasComponent[*components.PositionComponent](em, entity) {
        // 处理实体
    }
}
```

**✅ 推荐：使用 GetEntitiesWith 批量查询**
```go
plantEntities := ecs.GetEntitiesWith2[
    *components.PlantComponent,
    *components.PositionComponent,
](em)

for _, entity := range plantEntities {
    // 处理实体
}
```

#### 2. 缓存查询结果（适用于静态实体）

```go
// 系统初始化时查询一次
type MySystem struct {
    cachedEntities []ecs.EntityID
}

func (s *MySystem) Init(em *ecs.EntityManager) {
    s.cachedEntities = ecs.GetEntitiesWith2[
        *components.PlantComponent,
        *components.PositionComponent,
    ](em)
}

func (s *MySystem) Update(dt float64) {
    // 使用缓存结果（注意：仅适用于不变的实体列表）
    for _, entity := range s.cachedEntities {
        // 处理实体
    }
}
```

**注意**：仅在实体列表不变时使用缓存，否则可能导致 bug。

---

## 迁移检查清单

### Phase 1: 准备阶段（Story 9.1）
- [x] 阅读本迁移指南
- [x] 理解泛型 API 设计规范
- [x] 熟悉迁移模式和示例
- [ ] 运行基准测试，确认性能提升

### Phase 2: 迁移阶段（Story 9.2）
- [ ] 选择一个系统进行试点迁移（推荐 `behavior_system.go`）
- [ ] 替换所有 `GetComponent` 调用
- [ ] 替换所有 `AddComponent` 调用
- [ ] 替换所有 `HasComponent` 调用
- [ ] 替换所有 `GetEntitiesWith` 调用
- [ ] 删除所有 `reflect.TypeOf()` 调用
- [ ] 删除所有类型断言 `comp.(*T)`
- [ ] 运行测试验证功能正确性
- [ ] 运行基准测试验证性能提升

### Phase 3: 验证阶段（Story 9.3）
- [ ] 运行所有单元测试（`go test ./...`）
- [ ] 运行集成测试（如有）
- [ ] 运行游戏并手动测试核心功能
- [ ] 确认无性能退化
- [ ] 更新文档

---

## 常见问题（FAQ）

### Q1: 为什么不直接删除反射 API？

**A**: 为了降低迁移风险，Story 9.1 保留反射 API 作为向后兼容层。在 Story 9.2 完成所有系统迁移后，可以考虑删除反射 API。

---

### Q2: 泛型 API 是否支持组件继承？

**A**: Go 不支持继承，ECS 架构也不建议使用继承。组件应为扁平的数据结构。如果需要共享逻辑，应使用组合而非继承。

---

### Q3: 如何处理组件指针 vs 组件值？

**A**: 统一使用指针类型（`*Component`），与现有代码风格一致。原因：
- 组件存储在 map 中，使用指针避免复制
- 系统修改组件时需要直接修改实例

---

### Q4: GetEntitiesWith 的类型参数顺序是否影响结果？

**A**: 不影响查询结果（都是拥有指定组件的实体），但建议按逻辑重要性排序以提高可读性。

---

### Q5: 泛型 API 是否线程安全？

**A**: 与反射 API 一致，EntityManager 本身不提供线程安全保证。如需在多线程环境使用，需要外部同步机制。

---

## 联系与反馈

如有疑问或发现问题，请：
1. 查看 Epic 9 相关 Story 文档（9.1, 9.2, 9.3）
2. 查阅 `pkg/ecs/entity_manager.go` 源码注释
3. 运行基准测试验证性能

---

## 附录

### 附录 A: 类型参数转换表

| 反射 API | 泛型 API |
|---------|---------|
| `reflect.TypeOf(&components.PlantComponent{})` | `*components.PlantComponent` |
| `reflect.TypeOf(&components.BehaviorComponent{})` | `*components.BehaviorComponent` |
| `reflect.TypeOf(&components.PositionComponent{})` | `*components.PositionComponent` |

### 附录 B: 函数对照表

| 反射 API | 泛型 API | 参数差异 |
|---------|---------|---------|
| `em.GetComponent(entity, reflect.Type)` | `ecs.GetComponent[T](em, entity)` | 类型参数化 |
| `em.AddComponent(entity, comp)` | `ecs.AddComponent(em, entity, comp)` | 类型自动推导 |
| `em.HasComponent(entity, reflect.Type)` | `ecs.HasComponent[T](em, entity)` | 类型参数化 |
| `em.GetEntitiesWith(types...)` | `ecs.GetEntitiesWithN[T1, T2, ...](em)` | N=组件数量 |

### 附录 C: 相关文档

- **PRD**: `docs/prd/epic-9-ecs-generics-refactor.md`
- **Story 9.1**: `docs/stories/9.1.story.md` - 泛型 API 设计与原型
- **Story 9.2**: `docs/stories/9.2.story.md` - 系统迁移
- **Story 9.3**: `docs/stories/9.3.story.md` - 测试与文档
- **ECS 源码**: `pkg/ecs/entity_manager.go`

---

**文档结束** - 祝迁移顺利！ 🚀
