# Sprint 变更提案：Reanim 系统重新设计与通用化

**提案编号**: SCP-2025-11-03-001
**提案日期**: 2025-11-03
**提案人**: Sarah (Product Owner)
**状态**: 待批准
**优先级**: 🔴 高（影响 4+ Epic，阻碍未来开发）

---

## 📋 执行摘要

### 变更触发器
用户反馈："我觉得 reanim 模块的实现并没有完整的理解 reanim 的定义，没有完善的实现，有点过度设计。"

### 核心问题
经过系统性分析 127 个 reanim 文件，发现当前 Reanim 系统存在三个根本问题：

1. **架构过度设计** - 为豌豆射手实现了特殊的双动画叠加机制，但无法泛化
2. **缺少统一播放模式抽象** - 未识别 5 种动画模式，每种场景需要特殊处理
3. **对格式理解不完整** - 误解了 f 值语义、轨道类型和播放模型

### 影响范围
- ✅ **已完成 Epic**: Epic 6, 10（部分）, 12（部分）
- ⚠️ **进行中 Epic**: Epic 12（Story 12.1 使用特殊方法）
- 🔴 **未来 Epic**: Epic 8, 11, 以及所有未规划的动画相关功能

### 推荐解决方案
**选项 1：直接调整/集成** - 通过重构 Reanim 系统支持 5 种通用播放模式

- **工作量**: 3-5 天
- **风险**: 低
- **收益**: 消除技术债务，支持所有未来动画需求

---

## 🔍 第一部分：问题分析摘要

### 1.1 触发故事识别

**触发点**: 主菜单场景（Story 12.1）的 SelectorScreen.reanim 动画实现

**相关故事**:
- Story 6.5（Reanim 修复指南）- 已完成
- Story 12.1（主菜单场景 Reanim 集成）- ReadyForReview
- Story 10.3（植物攻击动画）- 已完成

### 1.2 核心问题定义

#### 问题 1：架构过度设计 ❌

**证据**:
- `ReanimComponent` 有 **20+ 个字段**，包括已废弃字段
- 针对豌豆射手的硬编码解决方案：
  ```go
  var HeadTracks = map[string]bool{
      "anim_face": true,
      "idle_mouth": true,
      // ...
  }
  const ReanimStemInitX = 37.6  // 豌豆射手专用
  ```
- 大量废弃代码（Story 6.4 的 `OverlayAnims`）

**影响**:
- 代码复杂度高，新开发者难以理解
- 维护成本高，每个特殊植物需要新的硬编码逻辑

#### 问题 2：缺少统一播放模式抽象 ❌

**证据**:
- 主菜单场景需要特殊的 `PlayAllFrames()` 方法
- 豌豆射手需要特殊的双动画叠加逻辑
- 60% 的植物使用混合模式，但系统缺少通用支持

**实际需求**（基于 127 个文件分析）:
| 模式 | 占比 | 示例文件 | 当前支持 |
|------|------|---------|---------|
| 1. 单轨道简单动画 | 5% | Lilypad.reanim | ✅ 支持 |
| 2. 多轨道骨骼动画 | 20% | Sun.reanim, SunFlower.reanim | ✅ 支持 |
| 3. 序列动画 | 15% | StartReadySetPlant.reanim | ⚠️ 部分支持 |
| 4. 复杂场景动画 | 5% | SelectorScreen.reanim | ❌ 需要特殊方法 |
| 5. 混合模式 | 60% | PotatoMine.reanim, Squash.reanim | ❌ 只支持豌豆射手 |

#### 问题 3：对格式理解不完整 ❌

**误解 A：f 值的语义**
- 当前代码将 `f=-1` 统一理解为"隐藏部件"
- 实际上 f 值有 **3 种不同语义**：
  - **动画定义轨道**: `f=-1` = 状态未激活，`f=0` = 状态激活
  - **混合轨道**: `f=-1` = 使用状态轨道控制，`f=0` = 自主显示
  - **序列动画**: `f=-1` = 隐藏，`f=0` = 显示

**误解 B：轨道类型分类**
- 代码使用硬编码列表识别轨道类型
- 应该基于轨道内容动态识别 3 种类型：
  1. **动画定义轨道**：只有 f 值
  2. **混合轨道**：f 值 + 图片 + 变换（76% 的轨道！）
  3. **纯视觉轨道**：图片 + 变换

**误解 C：播放模型**
- 当前假设每个 Reanim 播放一个动画
- 实际混合模式需要两层：
  - **第一层**：状态机（查询 f 值时间窗口）
  - **第二层**：渲染器（播放部件轨道）

### 1.3 证据汇总

1. **127 个 Reanim 文件的模式分析报告**（详见附录 A）
2. **代码审查发现**:
   - `ReanimComponent`: 271 行，20+ 字段
   - `ReanimSystem`: 包含大量硬编码逻辑
   - 主菜单场景需要 `PlayAllFrames()` 特殊处理
3. **CLAUDE.md 中的警告**:
   > "本系统中所有对 Reanim 动画系统的理解和说明都是猜想，仅作为参考，有可能有错误的理解"

---

## 📊 第二部分：Epic 影响评估

### 2.1 当前 Epic 分析

#### Epic 12: 主菜单系统 - ⚠️ 中度影响

**状态**: 进行中（Story 12.1 ReadyForReview）

**影响**:
- Story 12.1 使用特殊的 `PlayAllFrames()` 方法
- 代码包含针对 SelectorScreen 的大量特殊逻辑：
  ```go
  // Story 12.1: SelectorScreen 使用 PlayAllFrames 模式
  if err := scene.reanimSystem.PlayAllFrames(selectorEntity); err != nil {
      log.Printf("Warning: Failed to play SelectorScreen animation: %v", err)
  }
  ```

**需要修改**:
- 移除 `PlayAllFrames()` 特殊方法
- 使用通用的复杂场景动画播放模式

#### Epic 10: 游戏体验完善 - ⚠️ 中度影响

**状态**: 部分完成（Story 10.3 已完成）

**影响**:
- 当前针对豌豆射手实现了双动画叠加
- 但这是硬编码的，无法用于其他 60% 的植物
- 食人花、土豆雷等植物无法使用标准方法

**需要修改**:
- 将双动画叠加改为通用的混合模式播放
- 移除豌豆射手特殊逻辑

### 2.2 未来 Epic 影响

| Epic | 状态 | 影响程度 | 需要修改 | 优先级 |
|------|------|---------|---------|--------|
| Epic 6（动画系统迁移） | 已完成 | 🔴 高 - 需要重构 | 核心系统重新设计 | 最高 |
| Epic 12（主菜单系统） | 进行中 | 🟠 中 - Story 12.1 需要更新 | 替换 PlayAllFrames() | 高 |
| Epic 10（游戏体验完善） | 部分完成 | 🟠 中 - 攻击动画受限 | 更新攻击动画逻辑 | 高 |
| Epic 8（关卡实现） | 部分完成 | 🟡 中低 - 序列动画需支持 | 可能需要序列播放器 | 中 |
| Epic 11（UI 增强） | 未开始 | 🟢 低 - 简单动画为主 | 可能无需修改 | 低 |
| 未来 Epic（图鉴/商店/花园） | 未规划 | 🔴 高风险 - 依赖通用系统 | 不确定 | 最高 |

**关键发现**:
1. ✅ Epic 6 是根源问题
2. ⚠️ Epic 12 暴露了系统局限性
3. 🔴 技术债务扩散：如不解决，未来每个 Epic 都需要特殊处理
4. 📊 影响范围：至少 4 个已规划 Epic + 所有未来动画相关功能

---

## 📄 第三部分：制品冲突与影响分析

### 3.1 制品冲突汇总表

| 制品 | 位置 | 冲突程度 | 需要修改的内容 | 优先级 |
|------|------|---------|---------------|--------|
| **PRD** | `docs/prd.md` | 🟡 轻微 | Epic 6 目标描述，添加当前状态说明 | 中 |
| **架构文档 - 核心系统** | `docs/architecture/core-systems.md:41-44` | 🔴 严重 | AnimationSystem → ReanimSystem 完整重写 | 高 |
| **架构文档 - 数据模型** | `docs/architecture/data-models.md:28-40` | 🔴 严重 | AnimationComponent → ReanimComponent 完整重写 | 高 |
| **CLAUDE.md** | `CLAUDE.md:200-550` | 🟠 中度 | Reanim 章节重组，按 5 种模式重写 | 高 |
| **前端规范** | `docs/front-end-spec.md` | ✅ 无冲突 | 无需修改 | - |

**总计需要更新**: 4 个文档

### 3.2 详细冲突说明

#### 冲突 1: PRD Epic 6 描述 - 🟡 轻微冲突

**当前表述** (`docs/prd.md:120-121`):
```markdown
*   **Epic 6: 动画系统迁移 - 原版 Reanim 骨骼动画系统**
    *   **目标:** 将简单帧动画系统直接替换为原版 PVZ 的 Reanim 骨骼动画系统，
        实现 100% 还原原版动画效果，支持部件变换和复杂动画表现。
```

**问题**: 目标达成，但实现不完整

**提议修改**:
```markdown
*   **Epic 6: 动画系统迁移 - 原版 Reanim 骨骼动画系统**
    *   **目标:** 实现完整的 Reanim 骨骼动画系统，支持 5 种动画模式（简单动画、
        骨骼动画、序列动画、复杂场景动画、混合模式），100% 还原原版动画效果。
    *   **当前状态:** 部分完成，需要重构以支持通用播放模式（详见 Story 6.6, 6.7）。
```

#### 冲突 2: 架构文档 - 核心系统 - 🔴 严重冲突

**当前描述** (`docs/architecture/core-systems.md:41-44`):
```markdown
## **`AnimationSystem` (动画系统)**
*   **Responsibility:** 更新所有拥有`AnimationComponent`的实体的动画帧。
    它会根据`FrameSpeed`和流逝的时间来决定是否切换到下一帧。
*   **Key Interfaces:** `Update(deltaTime float64)`。
*   **Dependencies:** `EntityManager` (查询所有带动画的实体)。
```

**问题**:
- ❌ `AnimationSystem` 已被删除（Epic 6）
- ✅ 现在使用 `ReanimSystem`
- ❌ 文档未更新

**提议修改**:
```markdown
## **`ReanimSystem` (Reanim 骨骼动画系统)**

*   **Responsibility:** 管理原版 PVZ 的 Reanim 骨骼动画播放。支持 5 种动画模式：
    1. **简单动画**：单轨道循环动画（如 Lilypad）
    2. **骨骼动画**：多部件协同动画（如 Sun, SunFlower）
    3. **序列动画**：时间窗口控制的序列显示（如 StartReadySetPlant）
    4. **复杂场景动画**：独立时间线的大型场景（如 SelectorScreen）
    5. **混合模式**：状态机 + 骨骼动画（如 PotatoMine, Squash, 60% 的植物）

*   **Key Interfaces:**
    - `Update(deltaTime float64)` - 推进动画帧
    - `PlayAnimation(entityID, animName)` - 播放指定动画
    - `SetPlaybackMode(entityID, mode)` - 设置播放模式（用于复杂场景）

*   **Dependencies:** `EntityManager` (查询 ReanimComponent 实体)

*   **Components Used:** `ReanimComponent`, `PositionComponent`

*   **Related Documentation:**
    - `CLAUDE.md` - Reanim 动画系统使用指南
    - `docs/reanim/reanim-format-guide.md` - 格式详解
```

#### 冲突 3: 架构文档 - 数据模型 - 🔴 严重冲突

**当前描述** (`docs/architecture/data-models.md:28-40`):
```markdown
## **`AnimationComponent`**
*   **Purpose:** 管理基于spritesheet的动画。
*   **Go Struct:**
    ```go
    type AnimationComponent struct {
        Frames []*ebiten.Image
        FrameSpeed float64
        FrameCounter float64
        CurrentFrame int
    }
    ```
```

**问题**: `AnimationComponent` 已被删除

**提议修改**:
```markdown
## **`ReanimComponent`**

*   **Purpose:** 存储 Reanim 骨骼动画的状态和配置。包括动画定义、部件图片、
    播放状态、可见性控制、播放模式等。这是原版 PVZ Reanim 系统的核心组件。

*   **Go Struct:** （详见 `pkg/components/reanim_component.go`）
    ```go
    type ReanimComponent struct {
        // 动画定义
        Reanim              *reanim.ReanimXML       // 解析后的 Reanim 数据
        PartImages          map[string]*ebiten.Image // 部件图片映射

        // 播放状态
        CurrentAnim         string                  // 当前动画名
        CurrentFrame        int                     // 当前逻辑帧
        FrameAccumulator    float64                 // 帧累加器（FPS 控制）
        VisibleFrameCount   int                     // 可见帧数
        IsLooping           bool                    // 是否循环
        IsFinished          bool                    // 非循环动画是否完成
        IsPaused            bool                    // 是否暂停

        // 高级特性
        MergedTracks        map[string][]reanim.Frame // 帧继承后的合并轨道
        AnimVisiblesMap     map[string][]int          // 时间窗口映射
        VisibleTracks       map[string]bool           // 可见轨道白名单（复杂场景）

        // 混合模式（Story 6.5）
        IsBlending          bool                    // 是否启用混合模式
        PrimaryAnimation    string                  // 主动画（身体）
        SecondaryAnimation  string                  // 次动画（头部等）

        // 其他字段详见源码...
    }
    ```

*   **Key Fields Explanation:**
    - **Reanim**: 解析后的 Reanim 动画定义（包含轨道、帧数据）
    - **PartImages**: 部件名称到图片的映射（如 "IMAGE_REANIM_PEASHOOTER_HEAD" → Image）
    - **MergedTracks**: 应用帧继承后的完整轨道数据（所有帧都有完整变换）
    - **VisibleTracks**: 轨道可见性白名单（用于复杂场景动画，如 SelectorScreen）
    - **IsBlending**: 是否使用混合模式（60% 的植物需要，如土豆雷、南瓜头）

*   **Usage Patterns by Animation Mode:**
    - **简单/骨骼动画**: 只需 `Reanim`, `PartImages`, `CurrentAnim`
    - **序列动画**: 需要 `AnimVisiblesMap` 来控制时间窗口
    - **复杂场景动画**: 需要 `VisibleTracks` 白名单
    - **混合模式**: 需要 `IsBlending`, `PrimaryAnimation`, `SecondaryAnimation`
```

#### 冲突 4: CLAUDE.md - 🟠 中度冲突

**当前内容** (`CLAUDE.md:200-550`):
- 详细的 Reanim 系统说明
- 针对豌豆射手的双动画叠加机制（硬编码解决方案）
- 混合轨道的特殊说明（76% 发现）

**问题**:
- 将双动画叠加描述为"修复"，实际应该是"通用混合模式"
- 缺少对 5 种动画模式的系统性说明
- API 示例不完整

**提议修改**:
- 重写 Reanim 章节，按 5 种模式组织
- 移除豌豆射手特殊描述，改为通用模式说明
- 为每种模式添加播放示例
- 更新 API 文档

**详细修改内容**: 见附录 B

---

## 🛤️ 第四部分：推荐前进路径

### 4.1 路径对比

我们评估了 3 种可能的路径：

| 维度 | 选项1：直接调整 | 选项2：回滚 | 选项3：缩减范围 |
|------|---------------|------------|---------------|
| **工作量** | 3-5 天 | 15-20 天 | 1-2 天 |
| **技术风险** | 🟢 低 | 🔴 高 | 🟡 中 |
| **长期收益** | 🟢 高 | 🔴 负 | 🟡 中 |
| **代码质量** | 🟢 提升 | 🔴 退化 | 🟡 维持 |
| **MVP 影响** | 🟢 无 | 🔴 重大 | 🟢 无 |
| **技术债务** | 🟢 消除 | 🔴 增加 | 🔴 保留 |

### 4.2 推荐路径：选项 1 - 直接调整/集成 ✅

**理由**:
1. ✅ 成本最合理（3-5 天 vs 15-20 天）
2. ✅ 彻底解决问题，而非掩盖
3. ✅ 提升代码质量和可维护性
4. ✅ 为未来 Epic 铺平道路
5. ✅ 不影响 MVP 交付时间线

**实施方案**:
1. **创建新 Story（Epic 6 补充）**:
   - **Story 6.6**: Reanim 播放模式通用化（2-3 天）
   - **Story 6.7**: 清理废弃代码和过度设计（0.5-1 天）

2. **修改现有 Story**:
   - **Story 12.1**: 移除 `PlayAllFrames()` 特殊处理
   - **Story 10.3**: 移除豌豆射手特殊逻辑

3. **更新文档**（0.5-1 天）:
   - 更新 `docs/architecture/core-systems.md`
   - 更新 `docs/architecture/data-models.md`
   - 更新 `CLAUDE.md` Reanim 章节
   - 更新 `docs/prd.md` Epic 6 描述

---

## 📝 第五部分：具体提议的编辑

### 5.1 新增 Story 6.6: Reanim 播放模式通用化

**Story 文件**: `docs/stories/6.6.story.md`（新建）

```markdown
# Story 6.6: Reanim 播放模式通用化 (Reanim Playback Mode Generalization)

## Status
Draft

## Story
**As a** 开发者,
**I want** to refactor the Reanim system to support 5 general playback modes,
**so that** all 127 Reanim files can be played correctly without special-case code.

## Background

当前 Reanim 系统存在过度设计和不完整实现的问题：
- 为豌豆射手实现了硬编码的双动画叠加
- 为 SelectorScreen 实现了特殊的 `PlayAllFrames()` 方法
- 60% 的植物使用混合模式，但系统缺少通用支持

通过分析 127 个 Reanim 文件，识别出 5 种明确的动画模式（详见 SCP-2025-11-03-001）。

## Acceptance Criteria

### AC1: 识别并实现 5 种播放模式
- Given: Reanim 文件
- When: 系统解析文件
- Then: 自动识别其属于哪种模式
  - 模式 1: 单轨道简单动画
  - 模式 2: 多轨道骨骼动画
  - 模式 3: 序列动画（f 值控制）
  - 模式 4: 复杂场景动画
  - 模式 5: 混合模式（状态机 + 骨骼动画）

### AC2: 实现通用播放 API
- Given: 任意 Reanim 实体
- When: 调用 `PlayAnimation(entityID, animName)`
- Then: 系统根据模式自动选择正确的播放策略

### AC3: 移除硬编码逻辑
- Given: 当前代码库
- When: 重构完成
- Then:
  - 移除 `HeadTracks` 硬编码列表
  - 移除 `ReanimStemInitX/Y` 硬编码常量
  - 移除 `PlayAllFrames()` 特殊方法
  - 豌豆射手使用通用混合模式 API

### AC4: 向后兼容
- Given: 已有实体（豌豆射手、向日葵、除草车等）
- When: 使用新系统
- Then: 动画播放效果与重构前完全一致

### AC5: 性能无退化
- Given: 游戏运行
- When: 使用新系统
- Then: FPS ≥ 60，渲染时间 < 5ms/frame

## Tasks

### Task 1: 设计播放模式架构（0.5 天）
- [ ] 定义 `PlaybackMode` 枚举
- [ ] 设计模式识别算法
- [ ] 设计通用播放策略接口

### Task 2: 实现模式 1-2（简单/骨骼动画）（0.5 天）
- [ ] 实现单轨道简单动画播放器
- [ ] 实现多轨道骨骼动画播放器
- [ ] 测试 Lilypad, Sun, SunFlower

### Task 3: 实现模式 3（序列动画）（0.5 天）
- [ ] 实现序列动画播放器（f 值时间窗口）
- [ ] 测试 StartReadySetPlant

### Task 4: 实现模式 4（复杂场景动画）（0.5 天）
- [ ] 实现复杂场景动画播放器
- [ ] 移除 `PlayAllFrames()` 方法
- [ ] 测试 SelectorScreen

### Task 5: 实现模式 5（混合模式）（1 天）
- [ ] 实现通用混合模式播放器
- [ ] 移除豌豆射手硬编码逻辑
- [ ] 测试 PotatoMine, Squash, PeaShooter

### Task 6: 集成测试与回归测试（0.5 天）
- [ ] 所有已有动画正常播放
- [ ] 性能无退化
- [ ] 无视觉差异

## Estimated Effort
2-3 天

## Priority
🔴 高（阻碍未来开发）
```

### 5.2 新增 Story 6.7: 清理废弃代码和过度设计

**Story 文件**: `docs/stories/6.7.story.md`（新建）

```markdown
# Story 6.7: 清理废弃代码和过度设计 (Clean Up Deprecated Code)

## Status
Draft

## Story
**As a** 开发者,
**I want** to remove deprecated code and over-engineered components,
**so that** the codebase is clean, maintainable, and easy to understand.

## Acceptance Criteria

### AC1: 移除废弃字段
- Given: `ReanimComponent`
- When: 清理完成
- Then: 移除以下废弃字段
  - `BaseAnimName`（已废弃，使用 `CurrentAnim`）
  - `OverlayAnims`（Story 6.4 废弃）
  - `AnimVisibles`（已废弃，使用 `AnimVisiblesMap`）

### AC2: 简化组件结构
- Given: `ReanimComponent`
- When: 清理完成
- Then:
  - 字段数量从 20+ 减少到 15 以内
  - 每个字段有清晰的文档说明
  - 移除所有 "仅供豌豆射手使用" 的字段

### AC3: 移除特殊逻辑
- Given: `ReanimSystem`
- When: 清理完成
- Then:
  - 移除 `HeadTracks` 常量
  - 移除 `ReanimStemInitX/Y` 常量
  - 移除 `PlayAllFrames()` 方法
  - 移除所有 "// 豌豆射手特殊处理" 注释

### AC4: 文档更新
- Given: Story 6.4 标记为已废弃
- When: 清理完成
- Then:
  - 更新 Story 6.4 状态为 Deprecated
  - 添加废弃原因说明
  - 引用 Story 6.6 作为替代方案

## Tasks

### Task 1: 移除废弃字段（0.5 天）
- [ ] 从 `ReanimComponent` 移除废弃字段
- [ ] 更新所有引用这些字段的代码
- [ ] 确保编译通过

### Task 2: 移除特殊逻辑（0.5 天）
- [ ] 从 `ReanimSystem` 移除硬编码常量
- [ ] 移除 `PlayAllFrames()` 方法
- [ ] 清理特殊注释

### Task 3: 文档更新（0.5 天）
- [ ] 更新 Story 6.4 状态
- [ ] 更新 CLAUDE.md（移除废弃系统说明）
- [ ] 更新架构文档

## Estimated Effort
0.5-1 天

## Priority
🟡 中（技术债务清理）
```

### 5.3 修改 Story 12.1: 主菜单场景

**文件**: `docs/stories/12.1.story.md`

**修改位置**: Tasks 部分

**提议修改**:

```markdown
### Task 9: 移除 PlayAllFrames() 特殊处理（依赖 Story 6.6）

- [ ] **Task 9.1**: 等待 Story 6.6 完成
- [ ] **Task 9.2**: 移除 `PlayAllFrames()` 调用
  ```go
  // 修改前
  if err := scene.reanimSystem.PlayAllFrames(selectorEntity); err != nil {
      log.Printf("Warning: Failed to play SelectorScreen animation: %v", err)
  }

  // 修改后
  if err := scene.reanimSystem.PlayAnimation(selectorEntity, "anim_idle"); err != nil {
      log.Printf("Warning: Failed to play SelectorScreen animation: %v", err)
  }
  // 系统自动识别为复杂场景动画模式（模式 4）
  ```
- [ ] **Task 9.3**: 验证主菜单动画正常显示
- [ ] **Task 9.4**: 性能测试（FPS ≥ 60）
```

### 5.4 修改 Story 10.3: 植物攻击动画

**文件**: `docs/stories/10.3.story.md`

**修改位置**: 实现说明

**提议修改**:

```markdown
### 攻击动画实现（使用通用混合模式 - Story 6.6）

**修改前（硬编码豌豆射手）**:
```go
// 启用双动画叠加（仅豌豆射手）
reanimComp.IsBlending = true
reanimComp.PrimaryAnimation = "anim_idle"
reanimComp.SecondaryAnimation = "anim_shooting"
```

**修改后（通用混合模式）**:
```go
// 所有混合模式植物统一使用此方法
reanimSystem.PlayAnimation(entityID, "anim_shooting")
// 系统自动识别为混合模式（模式 5），并自动设置混合参数
```

**支持的植物**（使用混合模式）:
- 豌豆射手、火焰豌豆射手、寒冰射手
- 土豆雷、南瓜头、食人花
- 约 60% 的植物使用此模式
```

### 5.5 修改 PRD

**文件**: `docs/prd.md`

**修改位置**: 第 120-121 行

**修改前**:
```markdown
*   **Epic 6: 动画系统迁移 - 原版 Reanim 骨骼动画系统 (Animation System Migration - Reanim)**
    *   **目标:** 将简单帧动画系统直接替换为原版 PVZ 的 Reanim 骨骼动画系统，实现 100% 还原原版动画效果，支持部件变换和复杂动画表现。
```

**修改后**:
```markdown
*   **Epic 6: 动画系统迁移 - 原版 Reanim 骨骼动画系统 (Animation System Migration - Reanim)**
    *   **目标:** 实现完整的 Reanim 骨骼动画系统，支持 5 种动画模式（简单动画、骨骼动画、序列动画、复杂场景动画、混合模式），100% 还原原版动画效果。
    *   **Stories:** Story 6.1, 6.2, 6.3（已完成）, Story 6.6, 6.7（重构通用化）
    *   **当前状态:** 部分完成，需要重构以支持通用播放模式。详见 `docs/qa/sprint-change-proposal-reanim-system-redesign.md`。
```

### 5.6 修改架构文档 - 核心系统

**文件**: `docs/architecture/core-systems.md`

**修改位置**: 第 41-44 行

**完整替换内容**: 见第三部分 3.2 "冲突 2"

### 5.7 修改架构文档 - 数据模型

**文件**: `docs/architecture/data-models.md`

**修改位置**: 第 28-40 行

**完整替换内容**: 见第三部分 3.2 "冲突 3"

### 5.8 修改 CLAUDE.md

**文件**: `CLAUDE.md`

**修改位置**: 第 200-550 行（Reanim 动画系统使用指南）

**重写原则**:
1. 按 5 种播放模式组织内容
2. 每种模式包含：定义、特征、示例文件、播放方法
3. 移除所有针对特定植物的硬编码描述
4. 添加通用 API 使用示例

**新章节结构**:
```markdown
## Reanim 动画系统使用指南

### 概述
原版《植物大战僵尸》使用 Reanim 骨骼动画系统。本项目完整实现了该系统，
支持 5 种动画模式，可正确播放全部 127 个 Reanim 文件。

### 5 种播放模式

#### 模式 1: 单轨道简单动画
（内容详见附录 B）

#### 模式 2: 多轨道骨骼动画
（内容详见附录 B）

#### 模式 3: 序列动画
（内容详见附录 B）

#### 模式 4: 复杂场景动画
（内容详见附录 B）

#### 模式 5: 混合模式（60% 的植物）
（内容详见附录 B）

### API 使用指南
（详见附录 B）

### 调试技巧
（保留现有内容）
```

---

## ⏱️ 第六部分：实施时间线

### 6.1 整体时间表

| 阶段 | 任务 | 预估时间 | 依赖 | 负责人 |
|------|------|---------|------|--------|
| **阶段 1** | 创建 Story 6.6, 6.7 | 0.5 天 | - | PO (Sarah) |
| **阶段 2** | Story 6.6: Task 1 设计 | 0.5 天 | 阶段 1 | Dev |
| **阶段 3** | Story 6.6: Task 2-5 实现 | 2 天 | 阶段 2 | Dev |
| **阶段 4** | Story 6.6: Task 6 测试 | 0.5 天 | 阶段 3 | Dev + QA |
| **阶段 5** | Story 6.7: 清理代码 | 0.5-1 天 | 阶段 4 | Dev |
| **阶段 6** | 更新文档 | 0.5-1 天 | 阶段 5 | PO + Dev |
| **阶段 7** | 更新 Story 12.1, 10.3 | 0.5 天 | 阶段 4 | Dev |
| **总计** | - | **3-5 天** | - | - |

### 6.2 里程碑

- **M1 (Day 1)**: 设计完成，架构评审通过
- **M2 (Day 3)**: Story 6.6 核心实现完成
- **M3 (Day 4)**: 所有测试通过，代码清理完成
- **M4 (Day 5)**: 文档更新完成，变更完全落地

---

## 📋 第七部分：验收标准

### 7.1 功能验收

- [ ] **FA1**: 所有 127 个 Reanim 文件可正确识别其播放模式
- [ ] **FA2**: 5 种播放模式全部实现并通过单元测试
- [ ] **FA3**: 移除所有硬编码逻辑（HeadTracks, ReanimStemInitX/Y, PlayAllFrames()）
- [ ] **FA4**: 豌豆射手攻击动画使用通用混合模式 API
- [ ] **FA5**: SelectorScreen 使用通用复杂场景动画 API
- [ ] **FA6**: 所有已有实体动画效果与重构前完全一致

### 7.2 性能验收

- [ ] **PA1**: 游戏运行稳定在 60 FPS
- [ ] **PA2**: 渲染时间 < 5ms/frame（无性能退化）
- [ ] **PA3**: 内存使用无明显增加

### 7.3 代码质量验收

- [ ] **CQ1**: `ReanimComponent` 字段数量 ≤ 15
- [ ] **CQ2**: `ReanimSystem` 无硬编码常量
- [ ] **CQ3**: 所有单元测试通过，覆盖率 ≥ 80%
- [ ] **CQ4**: 代码符合编码规范（gofmt, golangci-lint）
- [ ] **CQ5**: 无废弃字段或方法

### 7.4 文档验收

- [ ] **DQ1**: `docs/architecture/core-systems.md` 更新完成
- [ ] **DQ2**: `docs/architecture/data-models.md` 更新完成
- [ ] **DQ3**: `CLAUDE.md` Reanim 章节按 5 种模式重写
- [ ] **DQ4**: `docs/prd.md` Epic 6 描述更新
- [ ] **DQ5**: Story 6.4 标记为 Deprecated
- [ ] **DQ6**: Story 6.6, 6.7 创建并文档化

---

## 🚀 第八部分：下一步行动

### 8.1 立即行动（提案批准后）

1. **PO (Sarah)**:
   - [ ] 创建 Story 6.6 文件 (`docs/stories/6.6.story.md`)
   - [ ] 创建 Story 6.7 文件 (`docs/stories/6.7.story.md`)
   - [ ] 更新 Story 12.1, 10.3 文件

2. **Dev Team**:
   - [ ] 审查提案和新 Story
   - [ ] 评估技术可行性
   - [ ] 确认时间估算

3. **Scrum Master (Bob)**:
   - [ ] 将 Story 6.6, 6.7 加入 Sprint Backlog
   - [ ] 安排 Sprint Planning 会议

### 8.2 Agent 移交计划

根据变更性质，需要移交给以下 Agent：

| Agent | 职责 | 移交内容 |
|-------|------|---------|
| **PM (John)** | 无需移交 | PRD 修改较小，PO 可自行处理 |
| **Architect (Alex)** | ⚠️ 可选移交 | 架构文档更新（如需架构评审） |
| **Dev (Claude)** | ✅ 必须移交 | Story 6.6, 6.7 实现 |
| **PO (Sarah)** | ✅ 已处理 | Story 创建、文档更新 |

**推荐移交顺序**:
1. **Dev** - 实现 Story 6.6（核心重构）
2. **Dev** - 实现 Story 6.7（代码清理）
3. **PO** - 更新所有文档

---

## 📊 第九部分：风险与缓解

### 9.1 风险矩阵

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|---------|
| **R1**: 重构引入新 Bug | 🔴 高 | 🟡 中 | 充分的回归测试；保留 git tag 用于快速回滚 |
| **R2**: 时间估算不准确 | 🟠 中 | 🟡 中 | 预留 buffer 时间；采用迭代方式 |
| **R3**: 性能退化 | 🔴 高 | 🟢 低 | 性能基准测试；持续监控 |
| **R4**: 文档更新不及时 | 🟡 低 | 🟡 中 | 文档更新列入 AC；代码审查检查 |
| **R5**: 影响其他 Epic | 🟠 中 | 🟢 低 | 向后兼容设计；渐进式迁移 |

### 9.2 缓解措施详情

**R1: 重构引入新 Bug**
- ✅ **预防**:
  - 在 feature branch 开发
  - 完整的单元测试和集成测试
  - 代码审查（至少 1 人审查）
- ✅ **检测**:
  - 自动化测试 CI/CD
  - 手动回归测试（所有关卡）
- ✅ **恢复**:
  - 创建 git tag `v0.x-before-reanim-refactor`
  - 回滚时间 < 5 分钟

**R2: 时间估算不准确**
- ✅ **应对**:
  - 采用迭代方式（先实现核心模式）
  - 每日 standup 跟踪进度
  - 预留 20% buffer 时间（总计 4-6 天）

**R3: 性能退化**
- ✅ **预防**:
  - 设计阶段考虑性能
  - 避免反射和重复计算
- ✅ **检测**:
  - 每次提交运行性能基准测试
  - 监控 FPS 和渲染时间
- ✅ **恢复**:
  - 性能分析工具（pprof）
  - 针对性优化瓶颈

---

## 📚 附录

### 附录 A: Reanim 文件模式分析报告

（包含完整的 127 个文件分析，详见之前生成的报告）

**采样文件列表**:
| 文件名 | 模式分类 | 轨道数 | 帧数 | 特征摘要 |
|--------|---------|--------|------|---------|
| Lilypad.reanim | 单轨道简单动画 | 1 | 9 | 简单位置振荡 |
| Sun.reanim | 多轨道骨骼动画 | 10 | ~30 | 太阳身体部件协同 |
| StartReadySetPlant.reanim | 序列动画 | 3 | 长 | f 值控制序列显示 |
| SelectorScreen.reanim | 复杂场景动画 | 500+ | 34033 行 | 独立时间线 |
| PotatoMine.reanim | 混合模式 | 16 | 200+ | 典型混合模式 |
| （完整列表见原始分析报告） |

**关键统计**:
- 总文件数：127
- 模式 1（简单动画）：5%
- 模式 2（骨骼动画）：20%
- 模式 3（序列动画）：15%
- 模式 4（复杂场景）：5%
- 模式 5（混合模式）：60% ⭐

### 附录 B: CLAUDE.md Reanim 章节重写内容

```markdown
## Reanim 动画系统使用指南

### 概述

原版《植物大战僵尸》使用 Reanim 骨骼动画系统。本项目完整实现了该系统，
支持 5 种动画模式，可正确播放全部 127 个 Reanim 文件。

**核心概念**:
- **轨道（Track）**: 动画的组成部分，可以是动画定义、部件或骨骼
- **帧（Frame）**: 轨道中的关键帧，定义变换（位置、缩放、倾斜等）
- **播放模式**: 根据文件结构自动识别的播放策略

### 5 种播放模式

#### 模式 1: 单轨道简单动画

**定义**: 只有 1-2 个轨道，通过简单的位置或缩放变化实现循环动画。

**特征**:
- 轨道数：1-2
- 无状态控制（无 f 值）
- 只有基础变换（x, y, sx, sy）
- 通常是环境装饰或简单物品

**示例文件**: `Lilypad.reanim`

**播放方法**:
```go
// 自动识别为模式 1，直接循环播放
reanimSystem.PlayAnimation(entityID, "anim_idle")
```

---

#### 模式 2: 多轨道骨骼动画

**定义**: 多个轨道代表不同身体部件，协同运动形成完整动画。

**特征**:
- 轨道数：4-30
- 每个轨道代表一个部件
- 无状态控制，所有轨道同步播放
- 使用复杂变换（kx, ky, a 等）

**示例文件**:
- `Sun.reanim`（10 轨道）
- `SunFlower.reanim`（30 轨道，28 个花瓣）

**播放方法**:
```go
// 自动识别为模式 2，所有部件同步播放
reanimSystem.PlayAnimation(entityID, "anim_idle")
```

---

#### 模式 3: 序列动画

**定义**: 使用 f=-1（隐藏）和 f=0（显示）控制不同部件在不同时间出现。

**特征**:
- 有 f 值控制
- 不同轨道有不同的激活时间窗口
- 用于过渡动画或分阶段显示

**示例文件**: `StartReadySetPlant.reanim`（"Ready", "Set", "Plant!" 依次出现）

**播放方法**:
```go
// 自动识别为模式 3，根据 f 值控制显示时机
reanimSystem.PlayAnimation(entityID, "anim_idle")
// 系统自动处理时间窗口
```

---

#### 模式 4: 复杂场景动画

**定义**: 包含数十到数百个轨道，每个轨道有独立的时间线。

**特征**:
- 轨道数：50-500+
- 轨道间独立性强
- 文件行数：10000-40000+
- 用于大型 UI 场景或过场动画

**示例文件**:
- `SelectorScreen.reanim`（34033 行，主菜单）
- `CrazyDave.reanim`（15065 行，疯狂戴夫对话）

**播放方法**:
```go
// 自动识别为模式 4
reanimSystem.PlayAnimation(entityID, "anim_idle")
// 系统自动管理所有独立轨道

// 可选：控制部分轨道显示
reanimComp.VisibleTracks = map[string]bool{
    "SelectorScreen_Adventure_button": true,
    "SelectorScreen_Cloud1": true,
    // ...
}
```

---

#### 模式 5: 混合模式（60% 的植物）⭐

**定义**: 结合状态机和骨骼动画，包含动画定义轨道和部件轨道。

**特征**:
- 有两类轨道：
  1. **状态定义轨道**：只有 f 值，定义动画状态
  2. **部件轨道**：有图片和变换，实际渲染
- 用于大部分植物和僵尸
- 最常见的模式

**示例文件**:
- `PotatoMine.reanim`（土豆雷：5 个状态 + 11 个部件）
- `Squash.reanim`（南瓜头：6 个状态 + 多个部件）
- `PeaShooter.reanim`（豌豆射手）

**两层结构**:
```
状态定义轨道:
  - anim_idle      (f=-1, f=0, f=-1 ...)  // 定义时间窗口
  - anim_attack    (f=-1, ..., f=0, ...)
  - anim_death     (f=-1, ..., f=0, ...)

部件轨道:
  - body           (有图片 + 变换)
  - head           (有图片 + 变换)
  - leaf1          (有图片 + 变换)
  - ...
```

**播放方法**:
```go
// 方法 1: 简单切换（系统自动处理混合）
reanimSystem.PlayAnimation(entityID, "anim_idle")    // 闲置
reanimSystem.PlayAnimation(entityID, "anim_attack")  // 攻击

// 方法 2: 查询可用状态
states := reanimSystem.GetAvailableStates(reanimDef)
// => ["anim_idle", "anim_attack", "anim_death"]

// 系统自动识别为混合模式，根据状态轨道的 f 值
// 控制部件轨道的显示
```

**工作原理**:
1. 播放 "anim_attack"
2. 系统查找 `anim_attack` 状态轨道
3. 根据该轨道的 f 值确定当前是否激活（f=0）
4. 如果激活，渲染所有部件轨道的对应帧
5. 如果未激活（f=-1），跳过渲染

---

### API 使用指南

#### 基础 API

```go
// 1. 播放动画（自动识别模式）
reanimSystem.PlayAnimation(entityID, animName)

// 2. 查询可用动画
animations := reanimSystem.GetAvailableAnimations(reanimDef)

// 3. 查询可用状态（仅混合模式）
states := reanimSystem.GetAvailableStates(reanimDef)

// 4. 检查动画是否完成（非循环动画）
isFinished := reanimComp.IsFinished

// 5. 暂停/恢复动画
reanimComp.IsPaused = true   // 暂停
reanimComp.IsPaused = false  // 恢复
```

#### 高级 API（复杂场景动画）

```go
// 控制轨道可见性（用于模式 4）
reanimComp.VisibleTracks = map[string]bool{
    "SelectorScreen_Adventure_button": true,
    "SelectorScreen_Cloud1": true,
    "SelectorScreen_Cloud2": false,  // 隐藏云朵 2
}

// 查询所有轨道名称
trackNames := reanimSystem.GetTrackNames(reanimDef)
```

#### 常见用法示例

```go
// 向日葵（模式 2）
reanimSystem.PlayAnimation(sunflowerEntity, "anim_idle")

// 土豆雷（模式 5 - 混合模式）
reanimSystem.PlayAnimation(potatoEntity, "anim_idle")   // 闲置
reanimSystem.PlayAnimation(potatoEntity, "anim_armed")  // 就绪
// 爆炸时切换到爆炸动画，非循环
reanimComp.IsLooping = false
reanimSystem.PlayAnimation(potatoEntity, "anim_explode")

// 主菜单（模式 4 - 复杂场景）
reanimSystem.PlayAnimation(selectorEntity, "anim_idle")
// 根据解锁状态控制按钮显示
if !isUnlocked("MiniGames") {
    reanimComp.VisibleTracks["SelectorScreen_MiniGame_button"] = false
}
```

### 调试技巧

（保留现有内容）

### 参考文档

- **Reanim 格式指南**: `docs/reanim/reanim-format-guide.md`
- **5 种模式详解**: `docs/qa/sprint-change-proposal-reanim-system-redesign.md`
- **修复指南**: `docs/reanim/reanim-fix-guide.md`（历史参考）
```

---

## ✅ 总结

本提案通过系统性分析 127 个 Reanim 文件，识别出当前系统的三个根本问题：
1. 架构过度设计
2. 缺少统一播放模式抽象
3. 对格式理解不完整

推荐通过**直接调整/集成**路径解决：
- **工作量**: 3-5 天
- **风险**: 低
- **收益**: 消除技术债务，支持所有未来动画需求

**关键行动**:
1. 创建 Story 6.6（Reanim 播放模式通用化）
2. 创建 Story 6.7（清理废弃代码）
3. 更新 Story 12.1, 10.3
4. 更新 4 个文档

**下一步**: 等待批准后，立即创建新 Story 并开始实施。

---

**批准签名**:

- [ ] 用户（项目负责人）
- [ ] PO (Sarah)
- [ ] Scrum Master (Bob)
- [ ] Tech Lead

**批准日期**: __________
