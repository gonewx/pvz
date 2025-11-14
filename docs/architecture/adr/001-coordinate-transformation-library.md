# ADR-001: 坐标转换工具库 (Coordinate Transformation Library)

## Status
**Accepted** - 2025-11-14

## Context

### 问题背景

在《植物大战僵尸》复刻项目中，Reanim 动画系统的坐标计算逻辑存在严重的代码重复和可维护性问题：

1. **坐标转换逻辑散落在 3 个系统中**：
   - `pkg/systems/render_reanim.go:109-110` - 渲染系统
   - `pkg/systems/input_system.go:243-244` - 点击检测系统
   - `pkg/systems/sodding_system.go:456-459` - 草皮系统
   - 每处都在手工计算：`pos.X - cameraX - CenterOffsetX`

2. **高认知负担**：
   开发者需要同时理解 5 个坐标概念：
   - 实体中心的世界坐标 (`pos.X`)
   - 摄像机偏移 (`cameraX`，UI 为 0)
   - BoundingBox 中心偏移 (`CenterOffsetX`)
   - Reanim 部件相对坐标 (`frame.X`)
   - 父子关系偏移 (`partData.OffsetX`)

3. **重复代码问题**：
   - 违反 DRY (Don't Repeat Yourself) 原则
   - 容易出错（忘记减 CenterOffset、错误应用摄像机偏移）
   - 修改公式需要改 3+ 处
   - 增加调试难度

4. **现有实现示例**：
   ```go
   // 渲染系统 (render_reanim.go)
   _, isUI := ecs.GetComponent[*components.UIComponent](em, id)
   effectiveCameraX := cameraX
   if isUI {
       effectiveCameraX = 0
   }
   baseScreenX := pos.X - effectiveCameraX - reanimComp.CenterOffsetX
   baseScreenY := pos.Y - reanimComp.CenterOffsetY

   // 点击检测 (input_system.go)
   clickCenterX := pos.X - reanimComp.CenterOffsetX
   clickCenterY := pos.Y - reanimComp.CenterOffsetY

   // 草皮系统 (sodding_system.go)
   worldLeftEdge := posComp.X - reanimComp.CenterOffsetX + leftEdge
   worldCenterX := posComp.X - reanimComp.CenterOffsetX + centerX
   ```

### 坐标系统概览

项目使用 BoundingBox 中心锚点方案：
- `PositionComponent.X/Y` 代表实体的**视觉中心**
- 在动画初始化时计算一次 `CenterOffset` 并缓存
- 渲染原点 = Position - CenterOffset（左上角基准）

## Decision

采用**方案 A：坐标转换工具库**，创建 `pkg/utils/coordinates.go`，提供 5 个包级函数：

1. **`GetRenderScreenOrigin(em, entityID, pos, cameraX)`**
   - 用途：渲染系统使用（最常用）
   - 计算：`(pos.X - cameraX - CenterOffsetX, pos.Y - CenterOffsetY)`
   - 特殊处理：UI 元素不应用摄像机偏移

2. **`GetClickableCenter(em, entityID, pos)`**
   - 用途：点击检测系统
   - 计算：`(pos.X - CenterOffsetX, pos.Y - CenterOffsetY)`

3. **`GetRenderOrigin(em, entityID, pos)`**
   - 用途：草皮系统等需要世界坐标的场景
   - 计算：`(pos.X - CenterOffsetX, pos.Y - CenterOffsetY)`

4. **`ReanimLocalToWorld(em, entityID, pos, localX, localY)`**
   - 用途：局部坐标转世界坐标
   - 计算：`(pos.X - CenterOffsetX + localX, pos.Y - CenterOffsetY + localY)`

5. **`WorldToScreen(worldX, worldY, cameraX, isUI)`**
   - 用途：通用坐标转换
   - 计算：`(worldX - effectiveCameraX, worldY)`

### 关键设计决策

根据架构审查 (Winston - 系统架构师) 的建议：
- ✅ **使用包级函数**，而非空结构体 `CoordinateHelper{}`
- ✅ **返回 `error`**，而非 `ok bool`（更清晰的错误语义）
- ✅ **表驱动测试 + 基准测试**
- ✅ **符合 ECS 架构原则**：
  - 零耦合：纯函数，通过 EntityManager 查询组件
  - 数据-行为分离：不修改组件定义
  - 渐进式复杂度：简单开始，按需扩展

## Consequences

### 正面影响

1. **代码简化**：
   - 代码行数减少 56-57%
   - Before: 7-8 行（手工计算）
   - After: 3 行（调用工具函数）

2. **降低认知负担**：
   - 开发者无需记住复杂的坐标转换公式
   - 函数名称清晰表达意图
   - 完整的 GoDoc 文档和示例

3. **零性能开销**：
   - **基准测试结果**（Intel i9-14900KF）：
     - `GetRenderScreenOrigin`: 20.07 ns/op, 0 allocs/op
     - `GetClickableCenter`: 11.27 ns/op, 0 allocs/op
     - `GetRenderOrigin`: 10.82 ns/op, 0 allocs/op
     - `ReanimLocalToWorld`: 11.78 ns/op, 0 allocs/op
     - `WorldToScreen`: 0.13 ns/op, 0 allocs/op
     - **手工计算**: 20.47 ns/op, 0 allocs/op
   - **结论**：工具函数甚至比手工计算稍快 (20.07ns vs 20.47ns)
   - 编译器成功内联，零内存分配

4. **提高代码质量**：
   - DRY 原则
   - 单一真理来源 (Single Source of Truth)
   - 更易于单元测试
   - 100% 测试覆盖率

5. **改善可维护性**：
   - 修改公式只需改 1 处
   - 错误更容易发现和修复
   - 新开发者更容易理解

### 负面影响

1. **学习成本**：
   - 新开发者需要学习 5 个新 API
   - **缓解措施**：完整的文档和示例（CLAUDE.md）

2. **间接调用开销**（理论上）：
   - **实际影响**：无，编译器成功内联
   - 基准测试证明零性能开销

3. **额外文件**：
   - 新增 `coordinates.go` (311 行) 和 `coordinates_test.go` (698 行)
   - **收益**：消除了 3 个系统中的重复代码

## Alternatives Considered

### 方案 B：组件方法增强

在 `ReanimComponent` 中添加方法：
```go
func (r *ReanimComponent) GetRenderScreenOrigin(pos *PositionComponent, cameraX float64, isUI bool) (float64, float64) {
    effectiveCameraX := cameraX
    if isUI {
        effectiveCameraX = 0
    }
    return pos.X - effectiveCameraX - r.CenterOffsetX, pos.Y - r.CenterOffsetY
}
```

**拒绝理由**：
- ❌ **违反 ECS 架构原则**："数据-行为分离"
- ❌ **组件应该只包含数据**，不包含方法
- ❌ **破坏架构一致性**：与项目现有 ECS 实践冲突

### 方案 C：渲染上下文对象

创建 `RenderContext` 对象封装坐标转换：
```go
type RenderContext struct {
    em      *ecs.EntityManager
    cameraX float64
}

func (rc *RenderContext) GetScreenOrigin(entityID ecs.EntityID, pos *PositionComponent) (float64, float64, error) {
    // 实现
}
```

**拒绝理由**：
- ❌ **改动过大**：需要重构所有系统以传递 RenderContext
- ❌ **过度设计**：5 个简单函数不需要对象封装
- ❌ **引入状态**：RenderContext 持有状态 (cameraX)，增加复杂度

## Architecture Review

**审查人**：Winston (系统架构师)
**审查日期**：2025-11-14
**审查结论**：✅ **批准通过**，附带架构改进建议

**关键建议**（已采纳）：
1. 使用包级函数，而非空结构体
2. 返回 `error`，而非 `ok bool`
3. 表驱动测试 + 基准测试
4. 文档先行

**架构符合性验证**：
- ✅ 零耦合原则：纯函数，无状态，通过 EntityManager 查询组件
- ✅ 数据-行为分离：不修改组件定义，行为封装在工具函数中
- ✅ 渐进式复杂度：简单开始（5 个核心函数），按需扩展

**详细审查文档**：`ARCHITECTURE_REVIEW_COORDINATE_REFACTORING.md`

## Implementation

### 文件结构

**新增文件**：
- `pkg/utils/coordinates.go` (311 行)
- `pkg/utils/coordinates_test.go` (698 行)

**后续重构** (Story 16.2)：
- `pkg/systems/render_reanim.go`
- `pkg/systems/input_system.go`
- `pkg/systems/sodding_system.go`

### 测试覆盖率

**单元测试**：
- 覆盖率：**100%**（所有函数）
- 测试用例：27 个（表驱动测试）
- 测试场景：
  - 有/无 ReanimComponent
  - UI vs 游戏实体
  - 摄像机偏移应用
  - 边界情况（零值、负值、大数值）
  - 错误路径

**基准测试**：
- 6 个基准测试（包含手工计算对比）
- 验证零性能开销目标

### 代码质量

- ✅ `gofmt` 格式化
- ✅ `golangci-lint` 通过
- ✅ 完整 GoDoc 注释
- ✅ 使用示例

## References

1. **Epic 16**: 坐标系统重构
2. **Story 16.1**: 创建坐标转换工具库并完善文档
3. **架构审查文档**: `ARCHITECTURE_REVIEW_COORDINATE_REFACTORING.md`
4. **问题分析**: `REFACTORING_PROPOSAL_COORDINATE_SYSTEM.md`
5. **锚点分析**: `ANCHOR_ANALYSIS_REPORT.md`
6. **Reanim 渲染审计**: `REANIM_RENDERING_AUDIT.md`
7. **ECS 架构原则**: `docs/architecture/coding-standards.md` (零耦合原则)
8. **Epic 14**: ECS 系统耦合解除重构（参考案例）

## Decision Makers

- **提议人**: Bob (Scrum Master)
- **架构审查**: Winston (系统架构师)
- **批准人**: Winston (系统架构师)
- **实施人**: James (Dev Agent)

## Timeline

- **2025-11-14**: ADR 创建
- **2025-11-14**: Story 16.1 实施完成
- **未来**: Story 16.2 - 重构核心系统以使用工具库

---

**最后更新**: 2025-11-14
**文档版本**: 1.0
