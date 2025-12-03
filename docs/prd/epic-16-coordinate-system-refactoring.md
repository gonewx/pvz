# Epic 16: Reanim 坐标系统重构 - Brownfield Enhancement

## Epic Goal

重构 Reanim 动画渲染系统的坐标转换逻辑，将散落在多个系统中的重复计算封装为统一的工具库，降低认知负担，提高代码可维护性，并减少 50% 的坐标计算相关错误。

## Epic Description

### Existing System Context

**当前相关功能:**
- Reanim 动画渲染系统使用 **BoundingBox 中心锚点**方案
- 坐标转换逻辑散落在 3 个系统中：
  - `pkg/systems/render_reanim.go` - 渲染系统
  - `pkg/systems/input_system.go` - 点击检测
  - `pkg/systems/sodding_system.go` - 草皮系统
- 每个系统都手工计算：`pos.X - cameraX - CenterOffsetX`
- 开发者需要理解 5 个坐标概念：实体中心、摄像机偏移、CenterOffset、部件相对坐标、父子偏移

**技术栈:**
- Go 1.x
- ECS 架构（Entity-Component-System）
- Ebiten v2 游戏引擎
- 泛型 API（Epic 9 已完成）

**集成点:**
- EntityManager（组件查询）
- ReanimComponent（动画组件）
- PositionComponent（位置组件）
- UIComponent（UI 标识）
- RenderSystem、InputSystem、SoddingSystem

### Enhancement Details

**要添加/更改的内容:**

1. **创建坐标转换工具库** (`pkg/utils/coordinates.go`)
   - 提供包级纯函数封装常见坐标转换
   - 核心函数：
     - `GetRenderScreenOrigin()` - 渲染原点（屏幕坐标）
     - `GetClickableCenter()` - 点击中心（世界坐标）
     - `GetRenderOrigin()` - 渲染原点（世界坐标）
     - `ReanimLocalToWorld()` - 局部坐标转世界坐标
     - `WorldToScreen()` - 世界坐标转屏幕坐标

2. **重构现有系统**
   - 渲染系统：用工具函数替换手工计算
   - 点击检测：用工具函数替换手工计算
   - 草皮系统：用工具函数替换手工计算

3. **统一 Animation Showcase 工具**
   - 将 `cmd/animation_showcase` 改回中心锚点
   - 复用游戏系统的坐标工具
   - 消除两套系统的认知混淆

**集成方式:**
- 纯函数，无副作用，通过 EntityManager 查询组件
- 完全符合 ECS 零耦合原则（参考 Epic 14 改造）
- 向后兼容：可与现有代码共存，渐进式重构
- 零性能开销：编译器可内联

**成功标准:**
- 代码行数减少 **56-57%**
- 重复代码从 **3 处** → **1 处**
- 坐标计算错误率减少 **50%**
- 新人上手时间从 **2 天** → **0.5 天**
- 单元测试覆盖率 **100%**
- 所有现有测试通过，无回归

### 背景文档

本 Epic 基于以下技术分析文档：
- `ANCHOR_ANALYSIS_REPORT.md` - 锚点方案深度分析
- `REFACTORING_PROPOSAL_COORDINATE_SYSTEM.md` - 重构提案（方案 A：工具库）
- `ARCHITECTURE_REVIEW_COORDINATE_REFACTORING.md` - 架构师审查（已批准）

## Stories

### Story 16.1: 创建坐标转换工具库并完善文档

**描述:** 创建 `pkg/utils/coordinates.go` 工具库，实现核心坐标转换函数，编写完整的单元测试和文档。

**关键任务:**
- 创建包级函数（非空结构体）
- 实现 5 个核心函数（返回 `error` 而非 `ok bool`）
- 表驱动测试，覆盖率 100%
- 基准测试确认零性能开销
- 创建 ADR-001（Architecture Decision Record）
- 更新 CLAUDE.md 坐标系统章节
- 添加迁移指南（旧 API → 新 API）

**优先级:** P0（最高）
**估计工作量:** 1-2 天

---

### Story 16.2: 重构核心系统的坐标计算逻辑

**描述:** 将渲染系统、点击检测、草皮系统中的手工坐标计算替换为工具库函数。

**关键任务:**
- 重构 `pkg/systems/render_reanim.go`（阶段 2）
- 重构 `pkg/systems/input_system.go`（阶段 3）
- 重构 `pkg/systems/sodding_system.go`（阶段 4）
- 添加针对新 API 的集成测试
- 手工验证游戏画面无异常
- 性能测试无回归（FPS 不下降）

**优先级:** P0（最高）
**估计工作量:** 1-1.5 天

---

### Story 16.3: 统一 Animation Showcase 工具

**描述:** 修改 `cmd/animation_showcase` 使用游戏系统的坐标工具，消除两套系统的不一致性。

**关键任务:**
- 修改 `cmd/animation_showcase/animation_cell.go`
- 改回中心锚点方案
- 复用 `pkg/utils/coordinates.go` 工具库
- 验证 Showcase 渲染效果与游戏一致

**优先级:** P2（可选优化）
**估计工作量:** 1 天

## Compatibility Requirements

- [x] **现有 API 保持不变**
  - 不修改组件定义（PositionComponent、ReanimComponent）
  - 不修改系统接口
  - 工具库是新增的，现有代码可继续运行

- [x] **数据库 schema 无变化**
  - 本 Epic 不涉及数据库

- [x] **UI 变化遵循现有模式**
  - 本 Epic 不涉及 UI 变更
  - Showcase 工具的变更不影响游戏

- [x] **性能影响最小化**
  - 纯函数，编译器可内联
  - 基准测试确认零性能开销
  - 架构审查确认无性能回归风险

## Risk Mitigation

### Primary Risk

**风险:** 坐标计算公式修改导致渲染位置错误，影响游戏画面

**严重性:** 中等（影响用户体验，但不会崩溃）

### Mitigation

1. **充分的单元测试**
   - 表驱动测试覆盖所有边界情况
   - 测试有/无 ReanimComponent 的情况
   - 测试 UI vs 游戏实体的区别
   - 测试摄像机偏移计算

2. **渐进式重构**
   - 先创建工具库并充分测试
   - 逐个系统重构（渲染 → 点击 → 草皮）
   - 每个阶段独立验证

3. **现有测试保护**
   - 所有现有单元测试必须通过
   - 集成测试必须通过
   - 手工回归测试（植物种植、僵尸移动、阳光点击）

4. **性能监控**
   - 基准测试确认无性能回归
   - FPS 监控

### Rollback Plan

如果发现问题：

1. **阶段 1 回滚**（工具库创建）
   - 删除 `pkg/utils/coordinates.go`
   - 恢复相关测试文件
   - 风险：低（未集成到系统）

2. **阶段 2-4 回滚**（系统重构）
   - Git revert 相关提交
   - 恢复手工计算逻辑
   - 风险：低（有完整的 Git 历史）

3. **验证回滚成功**
   - 运行所有测试
   - 手工验证游戏功能正常

## Definition of Done

- [ ] **所有 Stories 完成**
  - Story 16.1: 工具库创建 + 文档（Done）
  - Story 16.2: 核心系统重构（Done）
  - Story 16.3: Showcase 统一（Done）

- [ ] **所有验收标准满足**
  - 单元测试覆盖率 100%
  - 所有现有测试通过
  - 性能无回归

- [ ] **现有功能验证**
  - 植物种植位置正确
  - 僵尸移动路径正确
  - 阳光点击区域正确
  - 草皮铺设位置正确
  - 动画渲染位置正确

- [ ] **集成点正常工作**
  - EntityManager 组件查询正常
  - ReanimComponent 读取正常
  - PositionComponent 读取正常
  - 摄像机偏移应用正确

- [ ] **文档更新完成**
  - ADR-001 创建
  - CLAUDE.md 坐标系统章节更新
  - 迁移指南添加
  - API 文档（GoDoc）完整

- [ ] **无回归**
  - 所有现有功能正常
  - 性能未下降
  - 代码质量提升

## Technical Context

### 架构原则符合性

本 Epic 完全符合项目的 ECS 架构原则：

✅ **零耦合原则**（Epic 14 成功案例）
- 工具库是纯函数，无状态
- 系统不直接调用系统
- 通过 EntityManager 查询组件

✅ **数据-行为分离**
- 不修改组件定义（Component 只包含数据）
- 行为封装在工具函数中

✅ **渐进式复杂度**
- 简单开始（4 个核心函数）
- 按需扩展（未来可添加 BoundingBox、距离计算等）

### ROI 分析

根据架构审查报告：

| 指标 | 数据 |
|------|------|
| **投入** | 4 天工作量 |
| **回报** | 每次修改节省 60% 时间 |
| **修改频率** | 每月 2-3 次 |
| **回本周期** | 约 **1 个月** |

### 长期影响

- 新人上手时间：**2 天** → **0.5 天**
- 修改坐标公式：**3 处** → **1 处**
- Bug 修复时间：减少 **40%**
- 技术债务：完全偿还

## Dependencies

### 前置依赖

- [x] Epic 9（ECS 泛型重构）已完成
- [x] Epic 14（ECS 系统耦合解除）已完成
- [x] 架构审查已批准

### 后续依赖

无。本 Epic 为技术债务偿还，不阻塞其他 Epic。

## Notes

### 架构决策

本 Epic 采用**方案 A：坐标转换工具库**，理由：
- 最小改动（不破坏现有架构）
- 低风险（纯函数，无副作用）
- 零性能开销（编译器可内联）
- 符合 ECS 零耦合原则

**拒绝的方案**：
- 方案 B（组件方法增强）：违反数据-行为分离原则
- 方案 C（渲染上下文对象）：改动过大，风险较高

### 关键改进点（架构审查建议）

1. **API 设计**：使用包级函数，而非空结构体
2. **错误处理**：返回 `error`，而非 `ok bool`
3. **文档先行**：先创建 ADR 和更新 CLAUDE.md
4. **测试策略**：表驱动测试 + 基准测试

---

**Epic Status:** Draft
**Created:** 2025-11-14
**Owner:** Product Owner (Sarah)
**Approved By:** System Architect (Winston)
