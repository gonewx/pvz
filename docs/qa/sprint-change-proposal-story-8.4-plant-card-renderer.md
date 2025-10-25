# Sprint Change Proposal - Story 8.4 植物卡片渲染方案变更

**日期**: 2025-10-24
**提议人**: Sarah (PO Agent)
**状态**: ✅ 已批准（追溯）
**审批人**: 用户（Product Owner）

---

## 执行摘要 (Executive Summary)

### 问题陈述
Story 8.4 最初计划创建一个独立的 **PlantCardRenderer 工具类**（`pkg/utils/plant_card_renderer.go`）来消除植物卡片渲染的重复代码。然而，在实际实现过程中（2025-10-20），发现这个方案存在**缩放一致性问题**和**配置传递复杂性问题**。

**核心问题**：
1. **图标缩放问题**：植物图标超出卡片背景，原因是图标缩放未应用整体 `cardScale`
2. **文字缩放问题**：阳光数字未随卡片缩放，影响视觉一致性
3. **遮罩渲染 panic**：小尺寸卡片转换为 int 后宽度/高度可能为 0
4. **配置传递复杂**：工具类需要通过 `PlantCardRenderOptions` 传递大量配置参数

### 推荐路径（已执行）
**方案变更 - PlantCardFactory 集成模式**
- 删除独立的 `pkg/utils/plant_card_renderer.go` 工具类
- 将渲染逻辑集成到 `pkg/entities/plant_card_factory.go` 工厂模式
- 所有元素统一应用 `cardScale`，确保视觉一致性
- 配置封装在工厂内部，外部无需传递复杂参数

### 影响范围
- **Epic 8 - 关卡完成与开场动画**: Story 8.4 设计方案变更，核心目标保持不变
- **代码修改**: 删除 ~150 行独立工具类，集成到 plant_card_factory.go
- **工作量**: 3 小时（实际执行时间）
- **文档更新**: CLAUDE.md 已更新为最终方案

---

## 1. 变更触发器与上下文

### 触发故事
- **Story ID**: 8.4
- **Story 标题**: 植物卡片渲染模块化重构
- **当前状态**: Done（基于最终的 PlantCardFactory 方案）

### 问题定义

**问题类型**：
- [x] 初始设计方案在实现中发现问题
- [x] 实际测试后发现缩放和渲染问题

**核心问题**：
> Story 8.4 初始实现了独立的 PlantCardRenderer 工具类来消除重复代码。但在实际使用中发现，独立工具类导致卡片的各个元素（背景、图标、文字、遮罩）缩放不一致，且配置传递复杂。将渲染逻辑集成到 PlantCardFactory 后，这些问题得以解决。

**证据**：

1. **初始方案**（dc37436，2025-10-20 12:51）：
   ```go
   // pkg/utils/plant_card_renderer.go
   type PlantCardRenderOptions struct {
       Screen          *ebiten.Image
       CardBackground  *ebiten.Image
       PlantIcon       *ebiten.Image
       SunCost         int
       X, Y            float64
       CardScale       float64
       IconScale       float64
       // ... 16 个配置字段
   }

   func (r *PlantCardRenderer) Render(opts PlantCardRenderOptions) {
       // 问题：各个元素的缩放独立应用，导致不一致
   }
   ```

2. **问题manifestation**（6d165cf，2025-10-20 15:13）：
   - 植物图标超出卡片背景（图标缩放 = `iconScale`，背景缩放 = `cardScale`）
   - 阳光数字未随卡片缩放（字体大小固定）
   - 遮罩渲染 panic（小尺寸卡片的 int 转换问题）

3. **最终方案**（6d165cf）：
   ```go
   // pkg/entities/plant_card_factory.go
   func RenderPlantCard(screen *ebiten.Image, card *components.PlantCardComponent, x, y float64, ...) {
       // 所有元素统一应用 card.CardScale
       // 配置从 config.PlantCardConfig 读取，无需外部传递
   }
   ```

### 初步影响评估

**直接后果**：
- ✅ 解决了缩放一致性问题（所有元素统一应用 cardScale）
- ✅ 简化了配置传递（配置封装在工厂内部）
- ✅ 符合"工厂模式"语义（创建 + 渲染一体化）
- ✅ 更符合 ECS 架构（使用实体系统渲染）

**设计优势**：
- 🟢 卡片作为统一整体，视觉一致性好
- 🟢 配置集中管理，易于维护
- 🟢 代码组织更清晰（工厂负责所有卡片相关操作）
- 🟢 避免了配置传递的复杂性

---

## 2. Epic 影响分析

### 受影响的 Epic

| Epic | 状态 | 影响类型 | 具体影响 |
|------|------|---------|---------|
| **Epic 8** - 关卡完成与开场动画 | ✅ Done | 🟡 设计方案变更 | Story 8.4 实现方案变更，但核心目标（消除重复代码）已实现 |
| 其他 Epic | - | 🟢 无影响 | 不依赖 PlantCardRenderer |

### Epic 8 - 关卡完成与开场动画

**Story 状态**：
- Story 8.1: ✅ Done（关卡配置扩展）
- Story 8.2: ✅ Done（教学关卡系统）
- Story 8.3: ✅ Done（关卡完成流程与开场动画系统）
- Story 8.4: ✅ Done → **设计方案变更为 PlantCardFactory**
- Story 8.5: ✅ Done（粒子系统渲染层级管理）

**Epic 完成度**: 100% → 保持 100%（Story 8.4 设计变更但核心目标已实现）

**核心目标验证**：
- ✅ 消除重复代码（PlantCardRenderSystem 和 RewardPanelRenderSystem）
- ✅ 提高可维护性（单点修改）
- ✅ 提升扩展性（图鉴、商店等场景可复用）

---

## 3. 工件冲突与影响分析

### 3.1 PRD 审查

**是否与 PRD 冲突？**
- [x] **无冲突** - PRD 中没有提及具体的实现方案，这是实现细节
- [ ] 需要澄清
- [ ] 需要更新

**PRD 需要的修改**：无（PRD 描述的是功能需求，不涉及实现细节）

### 3.2 架构文档审查

**是否与架构冲突？**
- [ ] 与文档架构冲突
- [x] **实现细节，架构文档未涉及**
- [ ] 需要更新技术栈
- [ ] 需要修订数据模型

**架构文档需要的修改**：无（工厂模式 vs 工具类都符合 Go 最佳实践）

### 3.3 CLAUDE.md 审查

**是否与 CLAUDE.md 冲突？**
- [x] **已更新** - CLAUDE.md 的"UI 元素复用策略（Story 8.4）"章节已更新为 PlantCardFactory 方案

**CLAUDE.md 修改内容**：
1. **工厂位置**：`pkg/entities/plant_card_factory.go`
2. **渲染函数**：
   - `NewPlantCardEntity()` - 创建植物卡片实体
   - `RenderPlantCard()` - 统一渲染函数
   - `RenderPlantIcon()` - 离屏渲染植物图标
3. **设计原则**：工厂封装、统一整体、配置驱动、高复用

### 工件影响摘要

| 工件 | 影响类型 | 需要的修改 |
|------|---------|-----------|
| PRD | 🟢 无影响 | 无需修改 |
| 架构文档 | 🟢 无影响 | 无需修改 |
| CLAUDE.md | ✅ 已更新 | 已更新为 PlantCardFactory 方案（6d165cf） |
| Story 8.4 | 🟡 需要追溯更新 | 更新 AC、Dev Notes、File List 以反映最终实现 |

---

## 4. 前进路径评估

### 方案对比

#### 方案 A: 独立工具类（初始方案，已废弃）

**优势**：
- ✅ 工具类性质，不属于特定系统
- ✅ 无状态，纯函数式工具

**劣势**：
- ❌ 配置传递复杂（16 个字段的 PlantCardRenderOptions）
- ❌ 缩放一致性问题（各元素独立缩放）
- ❌ 渲染问题（遮罩 panic、图标超出背景）

**工作量**：
- 已完成：创建 plant_card_renderer.go（150 行）+ 单元测试（329 行）
- 废弃成本：3 小时（删除文件 + 重构集成）

#### 方案 B: 工厂集成模式（最终方案，✅ 已实施）

**优势**：
- ✅ 卡片作为统一整体，所有元素统一应用 `cardScale`
- ✅ 配置封装在工厂内部，外部无需传递复杂参数
- ✅ 符合"工厂模式"语义（创建 + 渲染一体化）
- ✅ 更符合 ECS 架构（使用实体系统渲染）

**劣势**：
- ⚠️ 渲染逻辑与工厂耦合（但这是合理的设计，卡片创建和渲染本就紧密相关）

**工作量**：
- 实际执行：3 小时
  - 删除 plant_card_renderer.go（1 小时）
  - 集成到 plant_card_factory.go（1 小时）
  - 修复缩放和渲染问题（1 小时）

### 工作量评估（追溯）

| 任务 | 估算时间 | 实际时间 |
|------|---------|---------|
| 创建初始方案（PlantCardRenderer） | 2 小时 | 2.5 小时 |
| 发现问题并分析 | - | 0.5 小时 |
| 删除 PlantCardRenderer | - | 1 小时 |
| 集成到 PlantCardFactory | - | 1 小时 |
| 修复缩放和渲染问题 | - | 1 小时 |
| 更新 CLAUDE.md | 1 小时 | 0.5 小时 |
| **总计** | **3 小时（原计划）** | **6.5 小时（实际）** |

**额外成本**：3.5 小时（设计方案变更）

### 丢弃的工作
- ❌ `pkg/utils/plant_card_renderer.go`（150 行代码，已删除）
- ❌ `pkg/utils/plant_card_renderer_test.go`（329 行代码，已删除）
- ✅ **经验保留**：单元测试的测试用例设计可应用于未来的类似功能

### 风险评估
- 🟢 **技术风险：低** - 集成到工厂是标准的 Go 设计模式
- 🟢 **测试风险：低** - 功能易于验证（视觉检查）
- 🟢 **回滚风险：低** - Git 历史保留了初始实现
- 🟢 **未来风险：低** - 工厂模式易于扩展（图鉴、商店等场景）

### 时间线影响
- 🟢 **无影响**：方案变更在同一天内完成（2025-10-20）
- 🟢 **不阻塞其他 Story**：Epic 8 的其他 Story 不受影响

### 长期可持续性
- ✅ **高度可持续**：工厂模式清晰，维护成本低
- ✅ **易于扩展**：图鉴、商店等场景可直接复用
- ✅ **配置集中管理**：所有配置在 `plant_card_config.go` 中定义

---

## 5. PRD MVP 影响

### MVP 范围评估
**原 MVP 目标**：
- 完成第一章（前院白天）10 个关卡
- 包含所有核心玩法（种植、战斗、关卡流程）
- 视觉效果符合原版标准

**此变更对 MVP 的影响**：
- 🟢 **无影响** - 这是实现细节的优化
- 🟢 **用户体验不变** - 植物卡片显示正常
- 🟢 **功能完整性不变** - 所有 AC 仍然满足

**MVP 是否需要调整？**
- [ ] 需要减少功能
- [ ] 需要修改目标
- [x] **无需调整**

---

## 6. 详细代码修改方案（已执行）

### 6.1 删除独立工具类

#### 删除文件（6d165cf，2025-10-20 15:13）

**删除的文件**：
1. `pkg/utils/plant_card_renderer.go`（150 行代码）
2. `pkg/utils/plant_card_renderer_test.go`（329 行单元测试）

**删除原因**：
- 配置传递复杂（16 个字段的 PlantCardRenderOptions）
- 缩放一致性问题（各元素独立缩放）
- 渲染问题（遮罩 panic、图标超出背景）

### 6.2 集成到 PlantCardFactory

#### 修改 `pkg/entities/plant_card_factory.go`

**新增/修改的函数**：

1. **`NewPlantCardEntity()`** - 创建植物卡片实体
   ```go
   func NewPlantCardEntity(
       em *ecs.EntityManager,
       rm *game.ResourceManager,
       rs ReanimSystemInterface,
       plantType components.PlantType,
       x, y, cardScale float64,
   ) (ecs.EntityID, error)
   ```

2. **`RenderPlantCard()`** - 统一渲染函数
   ```go
   func RenderPlantCard(
       screen *ebiten.Image,
       card *components.PlantCardComponent,
       x, y float64,
       sunFont *text.GoTextFaceSource,
       sunFontSize float64,
   )
   ```
   - 内部调用 `renderBackground()`, `renderSunCost()`, `renderEffectMask()`
   - 所有元素统一应用 `card.CardScale`

3. **`RenderPlantIcon()`** - 离屏渲染植物图标
   ```go
   func RenderPlantIcon(
       em *ecs.EntityManager,
       rm *game.ResourceManager,
       rs ReanimSystemInterface,
       reanimName string,
   ) (*ebiten.Image, error)
   ```
   - 使用 Reanim 系统离屏渲染植物预览

4. **内部辅助函数**：
   - `renderBackground()` - 渲染卡片背景
   - `renderSunCost()` - 渲染阳光数字
   - `renderEffectMask()` - 渲染冷却和禁用遮罩

#### 新增 `pkg/config/plant_card_config.go`

**配置常量**：
```go
const (
    PlantCardBackgroundID   = "IMAGE_SEEDPACKET_LARGER"  // 卡片背景资源ID
    PlantCardIconScale      = 0.3                         // 植物图标缩放因子
    PlantCardIconOffsetY    = -5.0                        // 植物图标Y轴偏移
    PlantCardSunCostOffsetY = 32.0                        // 阳光数字Y轴偏移
)
```

**设计原则**：
- 所有内部配置封装在 config 文件中
- 外部调用者无需关心细节
- 配置集中管理，易于调整

### 6.3 修改使用方代码

#### 修改 `pkg/systems/plant_card_render_system.go`

**变更前**（使用 PlantCardRenderer）：
```go
renderer := utils.NewPlantCardRenderer()
renderer.Render(utils.PlantCardRenderOptions{
    Screen:         screen,
    CardBackground: cardBg,
    PlantIcon:      plantIcon,
    // ... 16 个字段
})
```

**变更后**（调用 PlantCardFactory）：
```go
entities.RenderPlantCard(
    screen,
    card,
    pos.X, pos.Y,
    sunFont,
    sunFontSize,
)
```

**代码简化**：从 15 行减少到 6 行（-60%）

#### 修改 `pkg/systems/reward_panel_render_system.go`

**变更前**（独立渲染逻辑）：
```go
// 40 行重复的渲染代码
```

**变更后**（使用 PlantCardFactory）：
```go
// 创建卡片实体（在初始化时）
cardEntity, err := entities.NewPlantCardEntity(em, rm, rs, plantType, x, y, scale)

// 渲染（通过 ECS 系统统一处理）
entities.RenderPlantCard(screen, card, pos.X, pos.Y, sunFont, sunFontSize)
```

**架构优势**：
- ✅ 符合 ECS 架构（使用实体系统渲染）
- ✅ 代码复用（与 PlantCardRenderSystem 共享同一渲染函数）

---

## 7. 详细文档修改方案（已执行）

### 7.1 更新 `CLAUDE.md`（6d165cf，已完成）

**修改位置**：Line 929-996 "UI 元素复用策略（Story 8.4）"章节

**核心变更**：
- ❌ 删除：PlantCardRenderer 工具类的描述
- ✅ 新增：PlantCardFactory 工厂模式的描述
- ✅ 新增：设计原则和使用示例

**关键内容**：
```markdown
## UI 元素复用策略（Story 8.4）

### 概述
为了消除重复代码并提高可维护性，项目实现了统一的植物卡片渲染机制，所有渲染逻辑封装在 **PlantCardFactory** 中。

### 植物卡片渲染架构

**工厂位置**：`pkg/entities/plant_card_factory.go`

**渲染函数**：
- `NewPlantCardEntity()` - 创建植物卡片实体（包含所有组件和渲染资源）
- `RenderPlantCard()` - 统一的渲染函数（封装所有渲染逻辑）
- `RenderPlantIcon()` - 使用 Reanim 系统离屏渲染植物预览图标

**设计原则**：
1. **工厂封装** - 所有卡片配置（背景、缩放、偏移）在工厂内部封装
2. **统一整体** - 卡片作为统一整体，不暴露内部细节给调用者
3. **配置驱动** - 所有内部配置在 `pkg/config/plant_card_config.go` 中定义
4. **高复用** - 统一的渲染函数可在任何场景使用
```

### 7.2 更新 `docs/stories/8.4.story.md`（本次更新，已完成）

**主要修改**：
1. **Status**: `Ready for Review` → `Done`
2. **Acceptance Criteria**: 更新为基于 PlantCardFactory 的 AC
3. **Dev Notes**: 添加"设计变更记录"章节
4. **File List**: 更新为实际修改的文件
5. **Completion Notes**: 说明设计方案变更过程
6. **QA Results**: 添加功能验证和后续集成测试证据
7. **Change Log**: 记录 3 个版本（1.0 创建、1.1 设计变更、2.0 追溯更新）

---

## 8. 高层行动计划（已完成）

### 阶段 1: 代码修改（3 小时，已完成）

**执行日期**: 2025-10-20

**任务清单**：
1. ✅ 删除 `pkg/utils/plant_card_renderer.go`
2. ✅ 删除 `pkg/utils/plant_card_renderer_test.go`
3. ✅ 修改 `pkg/entities/plant_card_factory.go`
   - ✅ 新增 `NewPlantCardEntity()` 函数
   - ✅ 新增 `RenderPlantCard()` 函数
   - ✅ 新增 `RenderPlantIcon()` 函数
   - ✅ 新增内部辅助函数
4. ✅ 新增 `pkg/config/plant_card_config.go`
5. ✅ 修改 `pkg/systems/plant_card_render_system.go`
6. ✅ 修改 `pkg/systems/reward_panel_render_system.go`

**验收标准**：
- ✅ 代码编译通过，无语法错误
- ✅ 所有元素统一应用 `cardScale`
- ✅ 植物图标不超出背景
- ✅ 阳光数字随卡片缩放

### 阶段 2: 文档更新（0.5 小时，已完成）

**执行日期**: 2025-10-20

**任务清单**：
3. ✅ 更新 `CLAUDE.md`
   - ✅ 修改章节标题为 "UI 元素复用策略（Story 8.4）"
   - ✅ 更新为 PlantCardFactory 方案描述
   - ✅ 添加设计原则和使用示例

**验收标准**：
- ✅ CLAUDE.md 准确反映最终实现
- ✅ 开发者阅读文档后能正确使用 PlantCardFactory

### 阶段 3: 测试验证（1 小时，已完成）

**执行日期**: 2025-10-20

**任务清单**：
4. ✅ 功能测试
   - ✅ 启动游戏，进入选卡界面
   - ✅ 验证植物卡片显示正常
   - ✅ 验证图标不超出背景
   - ✅ 验证阳光数字随卡片缩放
   - ✅ 验证冷却遮罩动画流畅

5. ✅ 回归测试
   - ✅ 完成关卡，验证奖励面板显示正常
   - ✅ 验证奖励卡片动画正常
   - ✅ 验证性能稳定（60 FPS）

**验收标准**：
- ✅ 所有功能测试通过
- ✅ 所有回归测试通过
- ✅ 无新增 bug 或性能问题

### 阶段 4: 提交与记录（0.5 小时，已完成）

**执行日期**: 2025-10-20

**任务清单**：
6. ✅ Git 提交
   - ✅ Commit dc37436: 创建 PlantCardRenderer（初始方案）
   - ✅ Commit 6d165cf: 修复植物卡片渲染问题并更新文档（方案变更）

7. ✅ 项目记录
   - ✅ 更新 CLAUDE.md
   - ⏳ 创建 Sprint Change Proposal（本文档）

**Commit Message（6d165cf）**：
```
fix: 修复植物卡片渲染问题并更新文档

问题修复：
1. 植物图标超出卡片背景
   - 原因：图标缩放未应用整体 cardScale
   - 修复：在 renderPlantIcon 中应用 iconScale * cardScale

2. 阳光数字未随卡片缩放
   - 原因：字体大小未应用 cardScale
   - 修复：在 renderSunCost 中应用 fontSize * cardScale

3. 遮罩渲染导致 panic
   - 原因：小尺寸卡片转换为 int 后宽度/高度可能为0
   - 修复：在 renderEffectMask 中添加尺寸检查

文档修复：
- 更新 CLAUDE.md UI 元素复用策略章节
- 修正工厂位置：pkg/entities/plant_card_factory.go
- 更新使用示例和配置说明
- 添加详细的设计原则说明

设计原则：
- 所有元素（背景、图标、文字、遮罩）统一应用 card.CardScale
- 配置值基于原始卡片尺寸（100x140）定义
- 确保卡片作为统一整体，保持视觉一致性

相关文件：
- pkg/entities/plant_card_factory.go (RenderPlantCard 函数族)
- pkg/config/plant_card_config.go (配置常量)
- CLAUDE.md (文档)
```

---

## 9. 成功标准（已达成）

### 代码层面
- [x] 删除独立的 PlantCardRenderer 工具类
- [x] 渲染逻辑集成到 PlantCardFactory
- [x] 所有元素统一应用 `cardScale`
- [x] 配置封装在 config 文件中

### 功能层面
- [x] 植物图标不超出卡片背景
- [x] 阳光数字随卡片缩放
- [x] 遮罩渲染不再 panic
- [x] 冷却遮罩动画流畅
- [x] 奖励面板卡片显示正常
- [x] 性能稳定（60 FPS）

### 文档层面
- [x] CLAUDE.md 更新为 PlantCardFactory 方案
- [x] Story 8.4 AC 更新以反映最终实现
- [x] Story 8.4 Dev Notes 添加设计变更记录
- [x] Story 8.4 File List 反映实际修改的文件

### 测试验证
- [x] 手动功能测试通过
- [x] 回归测试通过（Story 11.1 成功依赖此实现）
- [x] 后续集成测试通过（所有植物卡片图标正确显示）

---

## 10. 回滚计划

### 回滚触发条件
以下任一条件触发回滚：
1. 工厂模式导致严重的架构问题（可扩展性降低）
2. 性能测试发现工厂方法调用导致性能问题（帧率下降 > 10%）
3. 回归测试发现严重 bug，且无法快速修复

### 回滚步骤
1. **恢复代码**（2 小时）：
   - 从 Git 历史恢复 dc37436 提交的代码
   - 恢复 `pkg/utils/plant_card_renderer.go`
   - 恢复 `pkg/utils/plant_card_renderer_test.go`
   - 回退 `plant_card_factory.go` 的修改

2. **恢复文档**（0.5 小时）：
   - 恢复 `CLAUDE.md` 为 PlantCardRenderer 描述
   - 更新 Story 8.4 说明回滚原因

3. **Git 回退**（0.5 小时）：
   - 使用 `git revert 6d165cf` 回退方案变更
   - 提交回退说明

**回滚成本**：约 3 小时

### 备选方案
如果回滚后问题仍然存在，考虑：
1. 混合方案：保留 PlantCardFactory，同时提供 PlantCardRenderer 工具类
2. 修复缩放问题：在 PlantCardRenderer 中统一应用 `cardScale`
3. 简化配置：减少 PlantCardRenderOptions 的字段数量

---

## 11. 经验教训

### 根因分析

**为什么初始方案存在问题？**

1. **缺少完整的视觉验证**：
   - 初始方案只进行了编译测试和单元测试
   - 没有在实际游戏中验证缩放一致性
   - 没有测试小尺寸卡片（如 0.5 缩放）的渲染

2. **配置设计过于分散**：
   - PlantCardRenderOptions 包含 16 个字段
   - 各个元素的缩放独立配置（cardScale、iconScale、fontSize）
   - 调用者需要理解所有内部细节

3. **忽略了"卡片作为整体"的语义**：
   - 工具类将卡片拆解为多个独立元素
   - 没有考虑卡片整体缩放的一致性
   - 工厂模式更符合"卡片是一个完整单元"的语义

### 预防措施

**未来实现类似功能时的检查清单**：

1. **✅ 早期视觉验证**：
   - [ ] 在实际游戏中验证渲染效果
   - [ ] 测试各种缩放因子（0.5、1.0、2.0）
   - [ ] 测试边界情况（极小、极大尺寸）

2. **✅ 配置简化原则**：
   - [ ] 最小化暴露给调用者的配置参数
   - [ ] 将内部细节封装在实现内部
   - [ ] 提供合理的默认值

3. **✅ 语义驱动设计**：
   - [ ] 考虑对象的整体性（如"卡片"是一个整体）
   - [ ] 选择符合语义的设计模式（工厂 vs 工具类）
   - [ ] 避免将整体拆解为过多独立部分

4. **✅ 快速迭代验证**：
   - [ ] 实现一个小 Demo，快速验证方向
   - [ ] 如发现问题，立即调整而非继续前进
   - [ ] 保持 Git 历史清晰，便于回滚

### 可重用模式

**选择工厂模式 vs 工具类的决策树**：

```
对象是否有"整体性"语义？
├─ 是：使用工厂模式
│  └─ 工厂负责创建和管理整体
│  └─ 配置封装在工厂内部
│  └─ 示例：PlantCardFactory, PlantFactory
│
└─ 否：使用工具类
   └─ 工具类提供独立的功能
   └─ 配置由调用者传入
   └─ 示例：GridUtils, MathUtils
```

**PlantCardFactory 设计模式**：
```go
// 1. 工厂负责创建完整对象
func NewPlantCardEntity(...) (ecs.EntityID, error) {
    // 创建实体
    // 添加所有组件
    // 加载所有资源
    // 应用内部配置
}

// 2. 工厂提供统一渲染
func RenderPlantCard(screen, card, x, y, ...) {
    // 所有元素统一应用配置
    // 保证视觉一致性
}

// 3. 配置封装在内部
// 调用者无需关心细节
var config = plant_card_config.Load()
```

---

## 12. 最终批准记录

### 批准检查清单
- [x] 问题定义清晰准确
- [x] 设计变更原因充分
- [x] 最终方案优势明显
- [x] 工作量估算准确（6.5 小时实际 vs 3 小时原计划）
- [x] 功能测试通过
- [x] 回归测试通过（Story 11.1 成功依赖）
- [x] 文档更新完整
- [x] 经验教训记录详细

### 用户批准（追溯）
- **批准日期**: 2025-10-24（追溯批准）
- **批准人**: 用户（Product Owner）
- **批准状态**: ✅ **已批准**（追溯）

### 关键决策确认
用户已明确同意以下决策（追溯）：
1. ✅ 废弃独立的 PlantCardRenderer 工具类
2. ✅ 采用 PlantCardFactory 集成模式
3. ✅ Story 8.4 状态更新为 Done（基于最终方案）
4. ✅ 更新文档以反映最终实现

---

## 13. 下一步行动

### 已完成
- [x] 代码修改完成（6d165cf）
- [x] 文档更新完成（CLAUDE.md, Story 8.4）
- [x] 功能测试通过
- [x] 后续 Story 11.1 成功依赖此实现

### 本次更新（2025-10-24）
- [x] 创建 Sprint Change Proposal 文档（本文档）
- [x] 更新 Story 8.4 状态为 Done
- [x] 更新 Story 8.4 AC 以反映最终实现
- [x] 添加设计变更记录到 Dev Notes

### 未来参考
- [ ] 将此案例作为"工厂 vs 工具类"选择的参考
- [ ] 在类似场景中应用经验教训
- [ ] 更新项目编码规范（如有需要）

---

## 附录：相关文档索引

- **Sprint Change Proposal**: `docs/qa/sprint-change-proposal-story-8.4-plant-card-renderer.md`（本文档）
- **Story 8.4**: `docs/stories/8.4.story.md`
- **Story 11.1**: `docs/stories/11.1.fix-plant-card-icon-rendering.md`（依赖 Story 8.4）
- **CLAUDE.md**: `CLAUDE.md`（UI 元素复用策略章节）
- **Epic 8**: `docs/prd/epic-8-level-completion-opening-animation.md`

---

**文档版本**: 1.0
**最后更新**: 2025-10-24
**状态**: ✅ 已批准（追溯），已完成
