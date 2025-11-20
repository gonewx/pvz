# Sprint Change Proposal: 僵尸获胜游戏结束流程

**日期**: 2025-11-20
**提议者**: Bob (Scrum Master)
**变更类型**: 新功能增强 (Epic 8 / Epic 10 扩展)
**优先级**: 中等
**状态**: 待审批

---

## 📋 执行摘要

**变更触发器**: 产品需求文档 `.meta/levels/zombiewon.md` - 游戏结束（僵尸获胜）流程完整需求

**核心问题**:
当前游戏实现了基础的失败判定逻辑（僵尸到达左侧边界 → 设置 `GameResult = "lose"`），但**缺少原版 PVZ 标志性的"僵尸获胜"过场效果**，包括：
1. 游戏冻结效果（植物停止攻击，镜头锁定）
2. 僵尸走进房子动画
3. 经典惨叫音效 + "吃脑子"动画（ZombiesWon.reanim）
4. 游戏结束对话框（重新尝试/返回主菜单）

**影响范围**: Epic 8（第一章关卡实现）和 Epic 10（游戏体验完善）的交叉区域，属于**关卡流程完整性**的重要组成部分。

**推荐路径**: **直接集成** - 在现有失败判定逻辑基础上扩展，添加僵尸获胜过场动画系统，无需回滚或重新规划。

---

## 🎯 1. 变更触发器与上下文 (Section 1: Change Context)

### 1.1 触发文件分析

**文件**: `.meta/levels/zombiewon.md`
**性质**: 功能需求文档（类似于 epic-level 的详细设计文档）

**需求覆盖范围**:
- ✅ **触发条件**: 僵尸判定框越过底线 X 轴坐标（已实现 - `DefeatBoundaryX = 100.0`）
- ⚠️ **第一阶段**: 入侵与锁定（部分实现 - 仅设置 GameResult，未实现游戏冻结）
- ❌ **第二阶段**: 惨叫与图文展示（未实现 - 核心缺失）
- ❌ **第三阶段**: UI 交互（未实现 - 缺少游戏结束对话框）
- ✅ **资源准备**: 音效、Reanim 动画、字体文件均已存在于 `assets/` 目录

### 1.2 当前实现状态

**已实现部分** (Story 10.2 - 除草车系统):
```go
// pkg/systems/level_system.go:337-342
if pos.X < DefeatBoundaryX {
    s.gameState.SetGameResult("lose")
    log.Printf("[LevelSystem] Defeat! Zombie reached left boundary")

    // 触发僵尸胜利动画
    s.triggerZombiesWon()  // ⚠️ 仅播放动画，无游戏状态控制
    return
}
```

**存在的基础设施**:
- ✅ `NewZombiesWonEntity()` - 已创建 Reanim 动画实体（`pkg/entities/ui_factory.go:179`）
- ✅ `AnimationCommandComponent` - 支持组件驱动的动画播放（Epic 14）
- ✅ `GameState.IsGameOver` - 游戏结束标志位
- ✅ Reanim 动画系统 - 支持 `ZombiesWon.reanim` 文件

**缺失部分**:
1. ❌ **游戏冻结系统** (Game Freeze System)
   - 植物停止攻击动画
   - 子弹/投掷物消失
   - UI 元素隐藏（植物选择栏、菜单按钮、进度条）
   - 背景音乐停止

2. ❌ **僵尸走进房子动画** (Zombie Entry Animation)
   - 触发失败的僵尸继续行走
   - 其他僵尸冻结
   - 动画完成后触发第二阶段

3. ❌ **音效触发系统** (Sound Effect Triggers)
   - `scream.ogg` (惨叫)
   - `chomp_soft.ogg` (咀嚼声)
   - 背景音乐淡出

4. ❌ **游戏结束对话框** (Game Over Dialog)
   - 标题: "游戏结束"
   - 按钮: "再次尝试" / "返回主菜单"
   - 点击空屏或等待 3-5 秒后显示

### 1.3 问题定义

**核心问题**: 当前实现只是**立即设置失败标志**，缺少原版游戏的**过场体验**，导致：
- **用户体验不完整**: 玩家感受不到原版的惊悚+黑色幽默氛围
- **功能完整性缺失**: 僵尸获胜流程是关卡系统的重要组成部分
- **与 PRD 不一致**: `docs/prd/requirements.md` 中 FR13.2 要求"明确的失败条件"，当前实现过于简单

**问题类型**: ✅ **新发现的需求** - 不是技术死路或需求误解，而是需要补全的功能细节

---

## 📊 2. Epic 影响评估 (Section 2: Epic Impact Analysis)

### 2.1 当前 Epic 影响

**Epic 8: 第一章关卡实现 (Chapter 1: Day Level Implementation)**

**状态**: 部分完成 (5/10 Stories Done)

**影响分析**:
- ✅ **可以继续**: 当前 Epic 8 不会被阻塞，已完成的 Stories (8.1-8.5) 无需修改
- ⚠️ **体验不完整**: Story 8.6-8.10 的关卡测试会暴露"失败流程不完整"的问题
- 📝 **建议**: 将"僵尸获胜流程"作为 **Story 8.11** 添加到 Epic 8，确保第一章的完整体验

**具体影响点**:
- **Story 8.6 (1-2 至 1-4 关卡)**: 玩家可能失败，需要完整的失败流程
- **Story 8.10 (集成测试)**: 失败流程是测试的重要部分

**Epic 10: 游戏体验完善 (Game Experience Polish)**

**状态**: 已完成 (4/4 Stories Done)

**影响分析**:
- ✅ **可以继续**: Epic 10 已完成，但缺少"失败体验"这一维度
- ⚠️ **体验不完整**: 当前 Epic 10 聚焦于"暂停菜单、攻击动画、粒子特效、除草车"，但忽略了"失败流程"
- 📝 **建议**: 将"僵尸获胜流程"作为 **Story 10.6** 添加到 Epic 10，与 Story 10.5 (胜利体验) 对称

**具体影响点**:
- **Story 10.1 (暂停菜单)**: 失败流程期间应阻止暂停
- **Story 10.2 (除草车系统)**: 已集成失败判定，需扩展为完整流程

### 2.2 未来 Epic 影响

**无重大影响** - 僵尸获胜流程是第一章关卡的基础功能，不会影响后续 Epic 的技术路线。

### 2.3 Epic 结构建议

**选项 A**: 将"僵尸获胜流程"添加到 Epic 8（推荐）
- **理由**: Epic 8 聚焦"关卡实现"，失败流程是关卡的核心组成部分
- **Story 编号**: 8.11
- **位置**: 插入 Story 8.10 之前（集成测试前）

**选项 B**: 将"僵尸获胜流程"添加到 Epic 10
- **理由**: Epic 10 聚焦"游戏体验"，失败体验是体验的一部分
- **Story 编号**: 10.6
- **位置**: 追加到 Epic 10 末尾

**推荐**: **选项 A** - Epic 8 更合适，因为失败流程是关卡流程的一部分，而非纯粹的"体验优化"。

---

## 📄 3. 项目文档冲突分析 (Section 3: Artifact Conflict Analysis)

### 3.1 PRD (产品需求文档)

**文件**: `docs/prd/requirements.md`

**冲突点**:
```markdown
FR13: 胜利/失败条件 - 必须实现明确的游戏结束条件：
  FR13.1: **胜利** - 玩家成功消灭当前关卡的所有僵尸。
  FR13.2: **失败** - 任意一个僵尸到达屏幕最左侧的房子。
```

**分析**:
- ✅ **触发条件已实现**: FR13.2 的"僵尸到达左侧"判定逻辑已完整
- ❌ **体验细节缺失**: PRD 只描述了"触发条件"，未描述"失败流程"（惨叫、动画、对话框）
- 📝 **需要更新**: 在 FR13.2 下添加子需求，明确失败流程的体验细节

**建议更新**:
```markdown
FR13.2: **失败** - 任意一个僵尸到达屏幕最左侧的房子。
  FR13.2.1: **游戏冻结** - 失败触发后，植物停止攻击，UI 元素隐藏，背景音乐停止
  FR13.2.2: **僵尸入侵动画** - 触发失败的僵尸继续行走至屏幕外
  FR13.2.3: **过场动画** - 播放 ZombiesWon.reanim 动画，伴随惨叫和咀嚼音效
  FR13.2.4: **游戏结束对话框** - 显示"游戏结束"对话框，提供"再次尝试"和"返回主菜单"选项
```

### 3.2 Architecture 文档

**文件**: `docs/architecture/core-systems.md`

**冲突点**: 无直接冲突

**需要更新的部分**:
- **LevelSystem 职责**: 添加"管理僵尸获胜过场流程"到职责列表
- **新增系统**: 可能需要创建 `GameOverSystem` 或在 `LevelSystem` 中扩展

### 3.3 Epic 文档

**文件**: `docs/prd/epic-8-level-chapter1-implementation.md`

**建议修改**:
1. 在 Story 列表中添加 **Story 8.11: 僵尸获胜流程与游戏结束对话框**
2. 更新 "Definition of Done" 部分，添加"失败流程完整实现"标准

**文件**: `docs/prd/epic-10-game-experience-polish.md`

**建议修改**:
- 更新 Epic 描述，明确"游戏体验"包括"胜利体验"和"失败体验"
- （如果选择选项 B）添加 Story 10.6

### 3.4 元数据文档

**文件**: `.meta/levels/zombiewon.md`

**状态**: ✅ 文档完整且清晰，无需修改

**集成方式**: 作为 Story 的 AC (Acceptance Criteria) 直接引用

### 3.5 冲突总结

| 文档 | 冲突级别 | 需要更新 | 优先级 |
|------|---------|---------|-------|
| `docs/prd/requirements.md` | ⚠️ 中等 | 添加 FR13.2 子需求 | 高 |
| `docs/prd/epic-8-level-chapter1-implementation.md` | ⚠️ 中等 | 添加 Story 8.11 | 高 |
| `docs/architecture/core-systems.md` | ℹ️ 轻微 | 更新 LevelSystem 职责 | 中 |
| `.meta/levels/zombiewon.md` | ✅ 无冲突 | 无 | N/A |

---

## 🛤️ 4. 解决方案路径评估 (Section 4: Path Forward Evaluation)

### 选项 1: 直接集成 (Direct Integration) ⭐ **推荐**

**描述**: 在现有 `LevelSystem` 基础上扩展，添加僵尸获胜过场系统

**实施步骤**:
1. 创建 **Story 8.11: 僵尸获胜流程与游戏结束对话框**
2. 扩展 `checkDefeatCondition()` 函数，添加游戏冻结逻辑
3. 创建 `GameFreezeComponent` 标记游戏冻结状态
4. 实现 `ZombiesWonPhaseSystem` 管理三阶段流程
5. 集成音效触发（`scream.ogg`, `chomp_soft.ogg`）
6. 创建游戏结束对话框实体

**工作量评估**:
- **开发时间**: 1-2 个 Sprint（约 2-4 周）
- **复杂度**: 中等
- **风险**: 低（基础设施完整，只需组装）

**优点**:
- ✅ 不破坏现有代码
- ✅ 符合 ECS 架构原则
- ✅ 可复用现有组件（AnimationCommand, DialogComponent）
- ✅ 测试覆盖率高（可单独测试每个阶段）

**缺点**:
- ⚠️ 增加 LevelSystem 复杂度（可通过模块化缓解）

### 选项 2: 暂缓实现 (Defer Implementation) ❌ **不推荐**

**描述**: 继续完成 Epic 8 的其他 Stories，将僵尸获胜流程推迟到后续 Epic

**理由**:
- ❌ **用户体验不完整**: 玩家测试第一章时会遇到"失败没有反馈"的问题
- ❌ **技术债务累积**: 后续补充失败流程会更复杂（需要回测所有关卡）
- ❌ **与 MVP 目标不符**: MVP 应包括完整的关卡流程（胜利 + 失败）

**结论**: **不采纳** - 失败流程是关卡系统的核心功能，不应推迟

### 选项 3: PRD 重新规划 (Re-scope PRD) ❌ **不必要**

**描述**: 修改 PRD，将失败流程定义为"非 MVP 功能"

**理由**:
- ❌ **不符合原版忠实度要求**: NFR2 明确要求"与原版高度一致"
- ❌ **降低产品质量**: 失败流程是原版 PVZ 的标志性体验

**结论**: **不采纳** - 失败流程应作为 MVP 的一部分

### 🎯 推荐路径

**✅ 选项 1: 直接集成**

**理由**:
1. **技术可行性高**: 所有基础设施已完备（Reanim 系统、对话框系统、音效系统）
2. **用户体验完整**: 确保第一章关卡具备完整的胜利+失败流程
3. **成本可控**: 工作量适中（1-2 Sprint），无需重构现有代码
4. **符合架构原则**: 使用 ECS 组件通信，零系统耦合

---

## 📝 5. 具体变更建议 (Section 5: Proposed Changes)

### 5.1 新增 Story: 8.11 僵尸获胜流程与游戏结束对话框

**Story 标题**: Story 8.11: 僵尸获胜流程与游戏结束对话框 (Zombies Won Game Over Flow)

**Story 目标**:
实现完整的僵尸获胜过场效果，包括游戏冻结、僵尸入侵动画、惨叫音效、"吃脑子"动画和游戏结束对话框，完全还原原版 PVZ 的失败体验。

**Story 范围**:

#### Phase 1: 游戏冻结系统 (Game Freeze System)
- **GameFreezeComponent** - 标记游戏冻结状态
- **冻结逻辑**:
  - 植物停止攻击动画（保持最后一帧）
  - 子弹/投掷物消失或悬停
  - UI 元素隐藏（植物选择栏、菜单按钮、进度条）
  - 背景音乐淡出（Fade out ~0.2s）
  - 僵尸除触发者外全部冻结

#### Phase 2: 僵尸入侵动画 (Zombie Entry Animation)
- **ZombiesWonPhaseComponent** - 管理三阶段状态机
  - `Phase 1: GameFreeze` (0.0s - 1.5s)
  - `Phase 2: ZombieEntry` (僵尸走出屏幕，约 2-3s)
  - `Phase 3: ScreamAnimation` (ZombiesWon.reanim 动画 + 音效)
- **僵尸行为**:
  - 触发失败的僵尸继续播放行走动画
  - 其他僵尸冻结（保持最后一帧）
  - 僵尸完全走出屏幕左侧后触发 Phase 3

#### Phase 3: 惨叫与动画 (Scream & Animation)
- **音效触发**:
  - `scream.ogg` (惨叫，僵尸走出屏幕瞬间)
  - `chomp_soft.ogg` (咀嚼声，紧接惨叫后 ~0.5s)
- **动画**:
  - `ZombiesWon.reanim` 的 `anim_screen` 动画在屏幕中央播放
  - 抖动特效（轻微的屏幕晃动，模拟惊悚感）

#### Phase 4: 游戏结束对话框 (Game Over Dialog)
- **触发条件**:
  - 点击屏幕任意位置（Phase 3 期间）
  - 或等待 3-5 秒后自动显示
- **对话框内容**:
  - 标题: "游戏结束" (使用 `LawnStrings.txt` 中的 ZOMBIES_WON 键)
  - 按钮 1: "再次尝试" → 重新加载当前关卡
  - 按钮 2: "返回主菜单" → 切换到主菜单场景
- **UI 设计**: 复用现有 `DialogComponent` 九宫格系统

**验收标准** (Acceptance Criteria):
1. ✅ 僵尸到达左侧边界后，游戏立即冻结（植物停止攻击，UI 隐藏）
2. ✅ 背景音乐在 ~0.2s 内淡出
3. ✅ 触发失败的僵尸继续行走至完全离开屏幕
4. ✅ 僵尸走出屏幕后，播放 `scream.ogg` 音效
5. ✅ 惨叫后 ~0.5s 播放 `chomp_soft.ogg` 音效
6. ✅ `ZombiesWon.reanim` 动画在屏幕中央显示，伴随轻微抖动特效
7. ✅ 玩家点击屏幕或等待 3-5s 后，显示游戏结束对话框
8. ✅ 对话框中文显示正确（"游戏结束"、"再次尝试"、"返回主菜单"）
9. ✅ "再次尝试"按钮点击后，关卡从头开始（阳光、植物、僵尸重置）
10. ✅ "返回主菜单"按钮点击后，返回主菜单场景
11. ✅ 整个流程符合原版 PVZ 的视觉和听觉体验（参考 `.meta/levels/zombiewon.md`）

**技术实现要点**:
- 使用 `AnimationCommandComponent` 播放 ZombiesWon 动画（符合 Epic 14 架构）
- 使用 `GameFreezeComponent` 标记冻结状态，系统通过查询该组件决定是否更新
- 音效通过 `ResourceManager.PlayAudio()` 触发（已有基础设施）
- 对话框复用 `NewDialogEntity()` 工厂函数（Epic 12 基础设施）

**估算工时**: 8-12 小时（约 1-1.5 Sprint）

---

### 5.2 新增组件

**文件**: `pkg/components/game_freeze_component.go`

```go
package components

// GameFreezeComponent 游戏冻结组件
// 标记游戏进入冻结状态（僵尸获胜流程期间）
//
// 系统行为：
// - BehaviorSystem: 检测到此组件时，停止更新植物攻击逻辑
// - PhysicsSystem: 停止子弹移动（可选：直接删除子弹实体）
// - UISystem: 隐藏植物选择栏、菜单按钮、进度条
// - AudioSystem: 淡出背景音乐
type GameFreezeComponent struct {
    IsFrozen bool // 是否已冻结（防止重复冻结）
}
```

**文件**: `pkg/components/zombies_won_phase_component.go`

```go
package components

// ZombiesWonPhaseComponent 僵尸获胜流程阶段组件
// 管理三阶段状态机
type ZombiesWonPhaseComponent struct {
    CurrentPhase int     // 当前阶段 (1: 冻结, 2: 僵尸入侵, 3: 惨叫动画)
    PhaseTimer   float64 // 阶段计时器（秒）

    // Phase 2 专用
    TriggerZombieID ecs.EntityID // 触发失败的僵尸ID
    ZombieExited    bool          // 僵尸是否已走出屏幕

    // Phase 3 专用
    ScreamPlayed    bool // 是否已播放惨叫音效
    ChompPlayed     bool // 是否已播放咀嚼音效
    AnimationReady  bool // 动画是否已准备好显示对话框
}
```

---

### 5.3 新增系统

**文件**: `pkg/systems/zombies_won_phase_system.go`

**职责**:
1. 管理僵尸获胜流程的三阶段状态机
2. Phase 1: 触发游戏冻结（添加 GameFreezeComponent）
3. Phase 2: 监控触发僵尸是否走出屏幕
4. Phase 3: 触发音效和动画播放
5. Phase 4: 监听点击事件或超时，显示对话框

**集成点**:
- 在 `GameScene.Update()` 中添加系统更新
- 在 `LevelSystem.checkDefeatCondition()` 中触发流程（添加 ZombiesWonPhaseComponent）

---

### 5.4 修改现有代码

#### 5.4.1 `pkg/systems/level_system.go`

**修改点 1**: `checkDefeatWithoutLawnmower()` 函数

**当前代码**:
```go
// pkg/systems/level_system.go:337-342
if pos.X < DefeatBoundaryX {
    s.gameState.SetGameResult("lose")
    log.Printf("[LevelSystem] Defeat! Zombie reached left boundary")

    // 触发僵尸胜利动画
    s.triggerZombiesWon()
    return
}
```

**修改后代码**:
```go
if pos.X < DefeatBoundaryX {
    s.gameState.SetGameResult("lose")
    log.Printf("[LevelSystem] Defeat! Zombie (ID:%d) reached left boundary at X=%.0f", entityID, pos.X)

    // 触发僵尸获胜流程（三阶段）
    s.triggerZombiesWonFlow(entityID)  // ✅ 新函数：启动完整流程
    return
}
```

**修改点 2**: 新增 `triggerZombiesWonFlow()` 函数

```go
// triggerZombiesWonFlow 触发僵尸获胜完整流程（三阶段）
func (s *LevelSystem) triggerZombiesWonFlow(triggerZombieID ecs.EntityID) {
    log.Printf("[LevelSystem] Triggering zombies won flow (3-phase)")

    // 创建流程控制实体
    flowEntityID := s.entityManager.CreateEntity()

    // 添加阶段控制组件
    phaseComp := &components.ZombiesWonPhaseComponent{
        CurrentPhase:    1,  // Phase 1: 游戏冻结
        PhaseTimer:      0.0,
        TriggerZombieID: triggerZombieID,
        ZombieExited:    false,
        ScreamPlayed:    false,
        ChompPlayed:     false,
        AnimationReady:  false,
    }
    ecs.AddComponent(s.entityManager, flowEntityID, phaseComp)

    // 添加游戏冻结标记
    freezeComp := &components.GameFreezeComponent{
        IsFrozen: true,
    }
    ecs.AddComponent(s.entityManager, flowEntityID, freezeComp)

    log.Printf("[LevelSystem] Flow entity created (ID:%d), zombie ID:%d", flowEntityID, triggerZombieID)
}
```

**修改点 3**: 废弃旧的 `triggerZombiesWon()` 函数

```go
// triggerZombiesWon 触发僵尸胜利动画（已废弃）
//
// ⚠️ Deprecated: 使用 triggerZombiesWonFlow() 代替，支持完整的三阶段流程
func (s *LevelSystem) triggerZombiesWon() {
    log.Printf("[LevelSystem] ⚠️  triggerZombiesWon() is deprecated, use triggerZombiesWonFlow() instead")
    // ... 保留原实现用于向后兼容，但标记为废弃
}
```

#### 5.4.2 `pkg/systems/behavior_system.go`

**修改点**: 在 `Update()` 开头添加冻结检测

**插入位置**: `func (s *BehaviorSystem) Update(deltaTime float64)` 第一行

**新增代码**:
```go
func (s *BehaviorSystem) Update(deltaTime float64) {
    // 检查游戏是否冻结（僵尸获胜流程期间）
    freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](s.entityManager)
    if len(freezeEntities) > 0 {
        // 游戏冻结期间，只允许触发僵尸继续移动（由 ZombiesWonPhaseSystem 管理）
        // 所有植物停止攻击
        return
    }

    // ... 原有逻辑
}
```

#### 5.4.3 `pkg/scenes/game_scene.go`

**修改点**: 在 `Update()` 中添加 ZombiesWonPhaseSystem 更新

**插入位置**: `LevelSystem.Update()` 之后

**新增代码**:
```go
// 更新僵尸获胜流程系统（如果存在）
if s.zombiesWonPhaseSystem != nil {
    s.zombiesWonPhaseSystem.Update(deltaTime)
}
```

---

### 5.5 新增资源引用

**音效文件** (已存在于 `assets/audio/`):
- `scream.ogg` - 男声惨叫
- `chomp_soft.ogg` - 咀嚼声

**Reanim 动画** (已存在于 `data/reanim/`):
- `ZombiesWon.reanim` - "吃脑子"动画

**配置文件** (需创建):
- `data/reanim_config/zombieswon.yaml` - ZombiesWon 动画配置

**示例配置**:
```yaml
id: "zombieswon"
name: "ZombiesWon"
reanim_file: "data/reanim/ZombiesWon.reanim"
default_animation: "anim_screen"

images:
  IMAGE_REANIM_ZOMBIESWON_SIGN: "assets/reanim/ZombiesWon_sign.png"
  # ... 其他部件图片

available_animations:
  - name: "anim_screen"
    display_name: "屏幕动画"

animation_combos: []  # 单动画，无需组合
```

---

### 5.6 文档更新

#### 5.6.1 `docs/prd/requirements.md`

**修改位置**: FR13.2 失败条件

**修改前**:
```markdown
FR13.2: **失败** - 任意一个僵尸到达屏幕最左侧的房子。
```

**修改后**:
```markdown
FR13.2: **失败** - 任意一个僵尸到达屏幕最左侧的房子。
  FR13.2.1: **游戏冻结** - 失败触发后，植物停止攻击，子弹消失，UI 元素（植物选择栏、菜单按钮、进度条）隐藏，背景音乐淡出
  FR13.2.2: **僵尸入侵动画** - 触发失败的僵尸继续行走至完全离开屏幕左侧边缘，其他僵尸冻结
  FR13.2.3: **过场动画** - 僵尸走出屏幕瞬间播放惨叫音效 (`scream.ogg`)，紧接播放咀嚼音效 (`chomp_soft.ogg`)，屏幕中央显示 `ZombiesWon.reanim` 动画并伴随轻微抖动特效
  FR13.2.4: **游戏结束对话框** - 玩家点击屏幕或等待 3-5 秒后，显示"游戏结束"对话框，提供"再次尝试"（重新加载关卡）和"返回主菜单"选项
  FR13.2.5: **关卡重置** - "再次尝试"按钮点击后，关卡从头开始（阳光、植物、僵尸、除草车全部重置）
```

#### 5.6.2 `docs/prd/epic-8-level-chapter1-implementation.md`

**修改位置 1**: Stories 列表（第 203-434 行之间）

**插入位置**: Story 8.10 之前

**新增内容**:
```markdown
---

### Story 8.11: 僵尸获胜流程与游戏结束对话框 (Zombies Won Game Over Flow)
**目标**: 实现完整的僵尸获胜过场效果

**范围**:
- **Phase 1: 游戏冻结系统** - GameFreezeComponent, 植物停止攻击, UI 隐藏, 音乐淡出
- **Phase 2: 僵尸入侵动画** - 触发僵尸继续行走, 其他僵尸冻结
- **Phase 3: 惨叫与动画** - scream.ogg, chomp_soft.ogg, ZombiesWon.reanim, 屏幕抖动
- **Phase 4: 游戏结束对话框** - "游戏结束", "再次尝试", "返回主菜单"

**验收标准**:
- 僵尸到达左侧边界后，游戏立即冻结
- 触发僵尸继续行走至屏幕外，其他僵尸冻结
- 惨叫和咀嚼音效按顺序播放
- ZombiesWon 动画在屏幕中央显示，伴随轻微抖动
- 对话框在点击或超时后显示
- "再次尝试"按钮重新加载关卡
- "返回主菜单"按钮切换场景
- 完全符合原版 PVZ 的失败体验（参考 `.meta/levels/zombiewon.md`）

**状态**: 📝 待创建

---
```

**修改位置 2**: Definition of Done（第 488-525 行）

**修改部分**: Story 完成状态列表

**修改前**:
```markdown
- [ ] **Story 8.10**: 第一章集成测试与调优 - 📝 待创建
```

**修改后**:
```markdown
- [ ] **Story 8.11**: 僵尸获胜流程与游戏结束对话框 - 📝 待创建
- [ ] **Story 8.10**: 第一章集成测试与调优 - 📝 待创建
```

**修改位置 3**: 功能完整性标准（第 507-525 行）

**新增项**:
```markdown
- [ ] 僵尸获胜流程完整实现
  - [ ] 游戏冻结效果正确（植物停止、UI 隐藏、音乐淡出）
  - [ ] 僵尸入侵动画流畅（触发僵尸行走，其他僵尸冻结）
  - [ ] 音效触发正确（惨叫 + 咀嚼，时序准确）
  - [ ] ZombiesWon 动画显示正确（屏幕中央，伴随抖动）
  - [ ] 游戏结束对话框功能完整（再次尝试 + 返回主菜单）
```

#### 5.6.3 `docs/architecture/core-systems.md`

**修改位置**: LevelSystem 职责描述（第 23-31 行）

**修改前**:
```markdown
## **`LevelSystem` (关卡系统)**
*   **Responsibility:** 负责所有关卡的核心逻辑
*   **Key Interfaces:** `Update(deltaTime float64)`, `CheckVictory()`, `CheckDefeat()`
*   **Dependencies:** `EntityManager`, `WaveSpawnSystem`
```

**修改后**:
```markdown
## **`LevelSystem` (关卡系统)**
*   **Responsibility:** 负责所有关卡的核心逻辑，包括：
    - 管理关卡时间推进和波次触发
    - 检测胜利/失败条件
    - 触发最后一波提示
    - 触发关卡完成奖励动画
    - **触发僵尸获胜流程** (Story 8.11)
*   **Key Interfaces:** `Update(deltaTime float64)`, `CheckVictory()`, `CheckDefeat()`, `triggerZombiesWonFlow(zombieID)`
*   **Dependencies:** `EntityManager`, `WaveSpawnSystem`, `RewardAnimationSystem`, `LawnmowerSystem`, `ZombiesWonPhaseSystem`
```

#### 5.6.4 `CLAUDE.md` (项目开发指南)

**修改位置**: 新增"僵尸获胜流程"章节（插入"核心组件说明"之后）

**新增内容**:
```markdown
## 僵尸获胜流程系统 (Story 8.11)

### 流程概述

僵尸获胜流程分为四个阶段，由 `ZombiesWonPhaseSystem` 统一管理：

1. **Phase 1: 游戏冻结 (Game Freeze)**
   - 触发条件：僵尸到达 `DefeatBoundaryX = 100.0`
   - 行为：添加 `GameFreezeComponent`，所有系统检测后停止更新
   - 持续时间：~1.5 秒

2. **Phase 2: 僵尸入侵 (Zombie Entry)**
   - 触发条件：Phase 1 完成
   - 行为：触发僵尸继续行走，监控其位置直至完全走出屏幕
   - 持续时间：~2-3 秒

3. **Phase 3: 惨叫与动画 (Scream & Animation)**
   - 触发条件：触发僵尸走出屏幕（`X < -100`）
   - 行为：
     - 播放 `scream.ogg` 音效
     - 延迟 ~0.5s 后播放 `chomp_soft.ogg`
     - 显示 `ZombiesWon.reanim` 动画（屏幕中央）
     - 轻微屏幕抖动特效
   - 持续时间：~3-4 秒

4. **Phase 4: 游戏结束对话框 (Game Over Dialog)**
   - 触发条件：玩家点击屏幕 OR 等待 3-5 秒超时
   - 行为：显示游戏结束对话框
     - 标题：「游戏结束」
     - 按钮 1：「再次尝试」→ 重新加载关卡
     - 按钮 2：「返回主菜单」→ 切换场景

### 关键组件

#### GameFreezeComponent
```go
// 标记游戏冻结状态，所有游戏逻辑系统检测后停止更新
type GameFreezeComponent struct {
    IsFrozen bool
}
```

#### ZombiesWonPhaseComponent
```go
// 管理四阶段状态机
type ZombiesWonPhaseComponent struct {
    CurrentPhase    int     // 1-4
    PhaseTimer      float64
    TriggerZombieID ecs.EntityID
    // ... 阶段标志位
}
```

### 系统集成

- **LevelSystem**: 检测失败条件后调用 `triggerZombiesWonFlow(zombieID)`
- **ZombiesWonPhaseSystem**: 管理四阶段流程，触发音效和动画
- **BehaviorSystem**: 检测 `GameFreezeComponent`，冻结时跳过更新
- **UISystem**: 检测 `GameFreezeComponent`，冻结时隐藏 UI
- **AudioSystem**: 在 Phase 1 淡出背景音乐

### 参考文档

- **需求文档**: `.meta/levels/zombiewon.md`
- **Epic 文档**: `docs/prd/epic-8-level-chapter1-implementation.md` (Story 8.11)
```

---

## 🎯 6. 变更总结与下一步 (Section 6: Summary & Next Steps)

### 6.1 变更摘要

| 变更类型 | 变更内容 | 优先级 |
|---------|---------|-------|
| **新增 Story** | Story 8.11: 僵尸获胜流程与游戏结束对话框 | 🔴 高 |
| **新增组件** | `GameFreezeComponent`, `ZombiesWonPhaseComponent` | 🔴 高 |
| **新增系统** | `ZombiesWonPhaseSystem` | 🔴 高 |
| **修改代码** | `LevelSystem.checkDefeatCondition()`, `BehaviorSystem.Update()` | 🔴 高 |
| **文档更新** | PRD (FR13.2), Epic 8 (Story 8.11), Architecture, CLAUDE.md | 🟡 中 |
| **资源配置** | `data/reanim_config/zombieswon.yaml` | 🟡 中 |

### 6.2 实施顺序

**推荐实施顺序**:
1. ✅ **批准 Sprint Change Proposal**（当前文档）
2. 📝 **更新 PRD 和 Epic 文档**（FR13.2, Story 8.11）
3. 🔧 **创建 Story 8.11 详细 Story 文档**（使用 SM Agent 的 `*draft` 命令）
4. 💻 **开发实现**（按 Phase 1-4 顺序开发）
5. 🧪 **单元测试 + 集成测试**（每个 Phase 独立测试）
6. 🎮 **用户验收测试**（完整流程测试）
7. 📚 **更新开发文档**（CLAUDE.md, Architecture.md）

### 6.3 风险缓解

**主要风险 1**: 游戏冻结逻辑可能影响现有系统

**缓解措施**:
- 使用组件标记（`GameFreezeComponent`）而非全局标志位
- 所有系统通过查询组件决定是否更新（符合 ECS 原则）
- 增加单元测试覆盖 `GameFreezeComponent` 的检测逻辑

**主要风险 2**: 音效和动画时序可能不准确

**缓解措施**:
- 使用阶段计时器精确控制时序
- 提供调试参数（`--debug-zombies-won`）快速重现流程
- 参考 `.meta/levels/zombiewon.md` 的时序要求

**主要风险 3**: 对话框集成可能遇到兼容性问题

**缓解措施**:
- 复用 Epic 12 的 `NewDialogEntity()` 基础设施（已验证）
- 与 `PauseMenuSystem` 使用相同的对话框渲染层级

### 6.4 成功标准

**技术标准**:
- ✅ 代码通过所有单元测试（覆盖率 80%+）
- ✅ 集成测试通过（Story 8.10 集成测试）
- ✅ 无性能退化（60 FPS 稳定）

**用户体验标准**:
- ✅ 失败流程与原版 PVZ 高度一致（参考 `.meta/levels/zombiewon.md`）
- ✅ 音效和动画时序准确（惨叫 → 咀嚼 → 抖动）
- ✅ 对话框交互流畅（再次尝试 / 返回主菜单）

**文档完整性**:
- ✅ PRD 更新完成（FR13.2 子需求）
- ✅ Epic 8 更新完成（Story 8.11 添加）
- ✅ Architecture 文档更新（LevelSystem 职责）
- ✅ CLAUDE.md 更新（开发指南章节）

### 6.5 估算工时

| 任务 | 工时 | 负责人 |
|------|------|--------|
| Story 8.11 详细设计 | 2h | Bob (Scrum Master) |
| 组件和系统开发 | 8-12h | Dev Agent |
| 单元测试编写 | 3-4h | Dev Agent |
| 集成测试 | 2-3h | Dev Agent |
| 文档更新 | 2h | Bob (Scrum Master) |
| **总计** | **17-23h** | **约 1-1.5 Sprint** |

---

## 📢 7. 代理人交接计划 (Section 7: Agent Handoff Plan)

### 7.1 下一步行动

**立即行动** (用户审批后):
1. **用户审批 Sprint Change Proposal** - 用户确认变更方案
2. **Bob (Scrum Master)** - 创建 Story 8.11 详细文档（使用 `*draft` 命令）
3. **Bob (Scrum Master)** - 更新 PRD 和 Epic 8 文档
4. **Dev Agent** - 实现 Story 8.11（Phase 1-4）

### 7.2 需要的代理人

| 代理人 | 职责 | 任务 |
|-------|------|------|
| **Bob (Scrum Master)** ⭐ 当前 | 变更管理、Story 创建、文档更新 | 1. 创建 Story 8.11 文档<br>2. 更新 PRD/Epic 文档<br>3. 跟踪实施进度 |
| **Dev Agent** | 代码实现、测试编写 | 1. 实现组件和系统<br>2. 编写单元测试<br>3. 集成测试 |
| **PM (Product Owner)** | 最终审批 | 审批 Sprint Change Proposal |

### 7.3 交接检查清单

**交接给 Dev Agent 前**:
- [ ] Sprint Change Proposal 已获用户批准
- [ ] Story 8.11 详细文档已创建（`docs/stories/8.11.story.md`）
- [ ] PRD 和 Epic 8 文档已更新
- [ ] 技术实现路径已明确（组件、系统、集成点）

**交接给 PM 前**:
- [ ] 所有代码已实现并通过测试
- [ ] 文档已更新（PRD, Epic, Architecture, CLAUDE.md）
- [ ] 集成测试通过
- [ ] 用户验收测试完成

---

## 📎 附录 (Appendix)

### A. 参考文档

- **需求文档**: `.meta/levels/zombiewon.md`
- **PRD**: `docs/prd/requirements.md` (FR13)
- **Epic 8**: `docs/prd/epic-8-level-chapter1-implementation.md`
- **Epic 10**: `docs/prd/epic-10-game-experience-polish.md`
- **Architecture**: `docs/architecture/core-systems.md`
- **开发指南**: `CLAUDE.md`

### B. 关键代码文件

- `pkg/systems/level_system.go` (失败判定逻辑)
- `pkg/entities/ui_factory.go:179` (NewZombiesWonEntity)
- `pkg/components/` (新增 GameFreezeComponent, ZombiesWonPhaseComponent)
- `pkg/systems/zombies_won_phase_system.go` (新增系统)

### C. 资源文件

- `assets/audio/scream.ogg` (惨叫音效)
- `assets/audio/chomp_soft.ogg` (咀嚼音效)
- `data/reanim/ZombiesWon.reanim` (吃脑子动画)
- `data/reanim_config/zombieswon.yaml` (动画配置，待创建)

---

## ✅ 批准签名

**提议者**: Bob (Scrum Master)
**日期**: 2025-11-20

**待批准**:
- [ ] **用户（项目所有者）** - 批准变更方案和实施计划
- [ ] **PM (Product Owner)** - 确认 Story 优先级和资源分配

**批准后下一步**: Bob (Scrum Master) 使用 `*draft` 命令创建 Story 8.11 详细文档

---

**文档版本**: 1.0
**最后更新**: 2025-11-20
**状态**: 📝 待审批
