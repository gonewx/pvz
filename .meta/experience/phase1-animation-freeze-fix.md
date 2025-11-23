# Phase 1 动画冻结问题修复

**时间**: 2025-11-23
**Story**: 8.8 - 僵尸获胜流程与游戏结束对话框
**问题**: Phase 1 冻结期间触发僵尸动画继续播放

---

## 问题描述

**用户报告**: "僵尸不是在移动,是没有冻结,一直在原地播放动画"

**期望行为**: Phase 1 游戏冻结期间（1.5秒），所有非UI实体的动画应该暂停，包括触发僵尸。

**实际行为**: 触发僵尸的动画在 Phase 1 期间继续播放，虽然位置没有移动，但动画帧在更新。

---

## 根本原因

**错误代码** (`pkg/systems/reanim_system.go:567-586`)：

```go
// Story 8.8: 检查游戏是否冻结（僵尸获胜流程期间）
freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](s.entityManager)
isFrozen := len(freezeEntities) > 0
var triggerZombieID ecs.EntityID = 0

if isFrozen {
    // 获取触发僵尸的ID
    phaseEntities := ecs.GetEntitiesWith1[*components.ZombiesWonPhaseComponent](s.entityManager)
    for _, phaseEntityID := range phaseEntities {
        phaseComp, ok := ecs.GetComponent[*components.ZombiesWonPhaseComponent](s.entityManager, phaseEntityID)
        if ok {
            triggerZombieID = phaseComp.TriggerZombieID
            break
        }
    }
}

// ❌ 问题：只检查触发僵尸ID，没有考虑当前阶段
if isFrozen && triggerZombieID != 0 && id != triggerZombieID {
    continue // 只暂停非触发僵尸的动画
}
// 触发僵尸的动画继续播放
```

**问题分析**：
- 旧逻辑只区分"触发僵尸"和"其他僵尸"
- 没有考虑流程阶段（Phase 1 vs Phase 2+）
- Phase 1 期间，触发僵尸的动画不应该播放

---

## 修复方案

**关键改进**：添加 `currentPhase` 跟踪，实现阶段感知的动画冻结逻辑。

**修复后的代码** (`pkg/systems/reanim_system.go:567-615`)：

```go
// Story 8.8: 检查游戏是否冻结（僵尸获胜流程期间）
// Phase 1: 所有实体动画暂停（包括触发僵尸）
// Phase 2+: 只有触发僵尸的动画继续，其他实体暂停
freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](s.entityManager)
isFrozen := len(freezeEntities) > 0
var triggerZombieID ecs.EntityID = 0
var currentPhase int = 0  // ✅ 新增：跟踪当前阶段

if isFrozen {
    // 获取触发僵尸的ID和当前阶段
    phaseEntities := ecs.GetEntitiesWith1[*components.ZombiesWonPhaseComponent](s.entityManager)
    for _, phaseEntityID := range phaseEntities {
        phaseComp, ok := ecs.GetComponent[*components.ZombiesWonPhaseComponent](s.entityManager, phaseEntityID)
        if ok {
            triggerZombieID = phaseComp.TriggerZombieID
            currentPhase = phaseComp.CurrentPhase  // ✅ 读取阶段信息
            break
        }
    }
}

// ✅ 修复：阶段感知的动画暂停逻辑
if isFrozen {
    // 检查是否是 UI 元素
    _, isUI := ecs.GetComponent[*components.UIComponent](s.entityManager, id)

    if !isUI {
        // Phase 1: 所有非UI实体动画暂停（包括触发僵尸）
        if currentPhase == 1 {
            continue  // ✅ Phase 1 期间所有动画冻结
        }

        // Phase 2+: 只有触发僵尸的动画继续
        if triggerZombieID != 0 && id != triggerZombieID {
            continue
        }
    }
    // UI 元素继续更新（不跳过）
}
```

**逻辑流程**：
```
游戏冻结状态？
├─ 否 → 所有动画正常播放
└─ 是 → 检查实体类型
    ├─ UI 元素 → 动画继续播放
    └─ 非 UI 元素 → 检查阶段
        ├─ Phase 1 → 所有动画冻结（包括触发僵尸）
        └─ Phase 2+ → 检查是否是触发僵尸
            ├─ 是触发僵尸 → 动画继续播放
            └─ 其他实体 → 动画冻结
```

---

## 调试过程

### 1. 初步观察

用户报告：僵尸在 Phase 1 期间"没有冻结，一直在原地播放动画"

初步怀疑：`Phase1FreezeDuration` 常量未生效

### 2. 日志分析

用户建议使用 `/tmp/pvz.log` 定位问题。

**关键日志发现**：
```log
2025/11/23 16:27:13 [ZombiesWonPhaseSystem] Phase 1: Game frozen, music fade out (TODO)
2025/11/23 16:27:13 [ZombiesWonPhaseSystem] Entity 4: Phase 1, Timer 0.00
...
2025/11/23 16:27:14 [ZombiesWonPhaseSystem] Entity 4: Phase 1, Timer 0.98
2025/11/23 16:27:14 [ZombiesWonPhaseSystem] Phase 1 -> Phase 2 (zombie entry)
```

**发现**：
- ✅ Phase 1 持续时间正确（1.0 秒，用户测试时改为 1.0s）
- ✅ 没有 "Trigger zombie moving" 日志 → 位置确实没变
- ❌ 动画帧仍在更新 → 问题在 ReanimSystem

### 3. 用户澄清

用户补充说明："时间是我测试时改的，不应该影响。补充说明一下，僵尸不是在移动，是没有冻结，一直在原地播放动画"

**关键信息**：
- 位置冻结 ✅ (BehaviorSystem 正确检查了 `currentPhase >= 2`)
- 动画播放 ❌ (ReanimSystem 未检查阶段)

### 4. 定位问题

检查 `pkg/systems/reanim_system.go:567-586`，发现冻结逻辑缺少阶段判断：

```go
// ❌ 只区分触发僵尸 vs 其他僵尸，未区分 Phase 1 vs Phase 2+
if isFrozen && triggerZombieID != 0 && id != triggerZombieID {
    continue
}
```

**问题确认**：触发僵尸的动画在所有阶段都继续播放。

---

## 测试验证

**测试覆盖**：

所有 11 个 Story 8.8 测试通过 ✅：

```
PASS: TestZombiesWonPhaseSystem_Phase2ZombieReachesTarget1ThenWalksIntoDoor
PASS: TestZombiesWonPhaseSystem_Phase2ZombieMovesInStraightLine
PASS: TestZombiesWonPhaseSystem_Phase2TransitionsToPhase3
PASS: TestZombiesWonPhaseSystem_Phase2CameraAndZombieMoveSimultaneously
PASS: TestZombiesWonPhaseSystem_Phase1ToPhase2Transition
PASS: TestZombiesWonPhaseSystem_Phase2SimultaneousMovement
PASS: TestZombiesWonPhaseSystem_Phase2ToPhase3Transition
PASS: TestZombiesWonPhaseSystem_Phase3AudioAndAnimation
PASS: TestZombiesWonPhaseSystem_FullFlowSimulation
PASS: TestZombiesWonPhaseSystem_GameFreezeComponentPresent
PASS: TestZombiesWonPhaseSystem_TargetPositionCalculation
```

**测试适配**：
- 测试原本只检查 Phase 1 持续时间和 Phase 2 转换
- 不需要修改测试代码
- 修复后的行为符合所有测试预期

---

## 经验总结

### 核心教训

1. **阶段感知的状态管理**
   - 游戏冻结不是"一刀切"的状态
   - 不同阶段可能有不同的冻结规则
   - Phase 1: 完全冻结（包括触发实体）
   - Phase 2+: 选择性冻结（触发实体继续）

2. **系统间协调**
   - BehaviorSystem 正确检查了 Phase 2+ (位置冻结)
   - ReanimSystem 缺少阶段检查 (动画未冻结)
   - 需要确保所有相关系统使用一致的冻结逻辑

3. **日志驱动调试**
   - 用户提供的运行日志 `/tmp/pvz.log` 至关重要
   - 对比"位置"和"动画"的日志，快速定位问题
   - 日志显示 Phase 1 期间没有"Trigger zombie moving"，但动画帧在更新

4. **ECS 零耦合原则**
   - 通过组件通信 (ZombiesWonPhaseComponent.CurrentPhase)
   - ReanimSystem 不需要引用 ZombiesWonPhaseSystem
   - 查询 PhaseComponent 获取当前阶段信息

### 代码模式

#### 错误模式：忽略阶段信息

```go
// ❌ 只区分实体类型，未区分阶段
if isFrozen && id != triggerZombieID {
    continue // 暂停非触发僵尸
}
```

#### 正确模式：阶段感知的冻结逻辑

```go
// ✅ 区分阶段 + 实体类型
if isFrozen {
    _, isUI := ecs.GetComponent[*components.UIComponent](em, id)

    if !isUI {
        // Phase 1: 所有非UI实体冻结
        if currentPhase == 1 {
            continue
        }

        // Phase 2+: 只有触发实体继续
        if id != triggerZombieID {
            continue
        }
    }
}
```

---

## 相关文件

- **修复文件**: `pkg/systems/reanim_system.go:567-615`
- **阶段组件**: `pkg/components/zombies_won_phase_component.go`
- **测试文件**: `pkg/systems/zombies_won_phase_system_test.go`
- **调试日志**: `/tmp/pvz.log` (运行时生成)

---

## 参考资料

- Story 8.8: `docs/stories/8.8.story.md`
- Phase 2 修复: `.meta/experience/zombies-won-clipping-rendering.md`
- ECS 架构: `CLAUDE.md#ECS 架构模式`
