# Sprint Change Proposal: 撑杆僵尸胜利动画特殊处理

**日期**: 2025-12-19
**提案人**: Bob (Scrum Master)
**状态**: 已批准

---

## 1. 问题摘要

**触发问题**：撑杆僵尸在僵尸获胜流程（Phase 2）中可能显示不合常理的视觉状态（持杆跑步或跳跃姿态），与进入房子的场景不协调。

**根本原因**：Story 8.8（僵尸获胜流程）设计时假设所有僵尸使用统一的入场动画，未考虑撑杆僵尸的特殊造型。

**问题类型**：新发现的需求（撑杆僵尸的特殊处理需求之前未被考虑）

**影响范围**：
- 仅影响视觉表现，不影响游戏逻辑
- 仅在撑杆僵尸触发失败条件时出现
- 影响关卡：1-6 至 1-10（包含撑杆僵尸的关卡）

---

## 2. 史诗影响摘要

| 史诗 | 影响 |
|------|------|
| Epic 8（第一章关卡） | 需要补丁故事 Story 8.16 |
| 其他史诗 | 无影响 |
| MVP 范围 | 无影响 |

---

## 3. 工件调整需求

| 工件 | 操作 | 说明 |
|------|------|------|
| `pkg/systems/zombies_won_phase_system.go` | 修改 | Phase 2 添加撑杆僵尸分支逻辑 |
| `docs/stories/8.8.story.md` | 更新 | 补充撑杆僵尸特殊处理 AC |
| `docs/stories/8.16.polevaulter-zombieswon-fix.story.md` | 新增 | 补丁故事 |

**无需更新**：
- PRD 文档
- 架构文档
- `pkg/config/zombie_registry.go`
- `data/reanim_config/zombie_polevaulter.yaml`
- `pkg/systems/pole_vault_system.go`

---

## 4. 推荐路径

**选择**：直接调整（选项1）

**理由**：
1. 变更范围最小，影响可控
2. 不废弃任何已完成的工作
3. 完全符合 ECS 架构原则
4. 可以作为 Story 8.8 的补丁故事

**排除的选项**：
- 选项2（回滚）：不适用，问题不是代码错误而是缺少边界处理
- 选项3（范围调整）：不需要，问题不影响 MVP 范围

---

## 5. 具体变更草案

### 5.1 代码变更：`pkg/systems/zombies_won_phase_system.go`

**位置**：`updatePhase2ZombieEntry()` 函数，Phase 2 初始化逻辑

**变更内容**：

```go
// 在 Phase 2 开始时（phaseComp.PhaseTimer == deltaTime）添加：

// 检测是否为撑杆僵尸，需要特殊处理
behaviorComp, hasBehaviorComp := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, phaseComp.TriggerZombieID)
if hasBehaviorComp && behaviorComp.UnitID == types.UnitIDZombiePolevaulter {
    // 撑杆僵尸特殊处理：
    // 1. 立即传送到第2个目标位置（跳过移动阶段）
    if posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, phaseComp.TriggerZombieID); ok {
        target1X := config.GameOverDoorMaskX + Phase2ZombieTarget1OffsetX
        target1Y := config.GameOverDoorMaskY + Phase2ZombieTarget1OffsetY
        target2X := target1X + Phase2ZombieTarget2OffsetX

        posComp.X = target2X
        posComp.Y = target1Y
        log.Printf("[ZombiesWonPhaseSystem] 撑杆僵尸特殊处理: 传送到目标位置 (%.2f, %.2f)", posComp.X, posComp.Y)
    }

    // 2. 强制切换为行走动画（无杆状态）
    ecs.AddComponent(s.entityManager, phaseComp.TriggerZombieID, &components.AnimationCommandComponent{
        UnitID:    types.UnitIDZombiePolevaulter,
        ComboName: "walk",
        Processed: false,
    })

    // 3. 确保撑杆状态为已丢弃
    if poleVault, ok := ecs.GetComponent[*components.PoleVaultComponent](s.entityManager, phaseComp.TriggerZombieID); ok {
        poleVault.HasPole = false
        poleVault.IsJumping = false
    }

    // 4. 标记已到达目标，触发继续向左走入房子
    phaseComp.ZombieReachedTarget1 = true
    phaseComp.ZombieStartedWalking = true
}
```

### 5.2 Story 8.8 文档更新

**位置**：`docs/stories/8.8.story.md` - Phase 2 验收标准

**新增内容**：

```markdown
### Phase 2 补充：撑杆僵尸特殊处理 (2025-12-19 新增)

当触发失败的僵尸为撑杆僵尸时，执行特殊处理：

1. **跳过移动阶段**：撑杆僵尸直接"传送"到第2个目标位置（即将进入房子的位置）
2. **强制行走动画**：切换为 `walk` 动画组合（无杆状态），确保视觉合理
3. **重置撑杆状态**：确保 `PoleVaultComponent.HasPole = false`
4. **同步流程**：与摄像机左移同步，僵尸从第2目标位置开始向左走入房子

**设计原因**：
- 撑杆僵尸的跑步/跳跃动画与进入房子的场景不协调
- 避免在门口出现持杆或跳跃姿态的不合理视觉
```

---

## 6. 高层行动计划

| 步骤 | 任务 | 负责角色 | 状态 |
|------|------|---------|------|
| 1 | 创建补丁故事 Story 8.16 | Scrum Master | ✅ 完成 |
| 2 | 保存变更提案文档 | Scrum Master | ✅ 完成 |
| 3 | 实现代码变更 | Developer | 待开始 |
| 4 | 更新 Story 8.8 文档 | Developer | 待开始 |
| 5 | 添加单元测试 | Developer | 待开始 |
| 6 | 手动验证 | QA | 待开始 |

---

## 7. Agent 交接计划

| 角色 | 任务 |
|------|------|
| **Scrum Master (Bob)** | ✅ 创建补丁故事，保存变更提案 |
| **Developer** | 实现代码变更，编写测试 |
| **QA** | 验证修复效果 |

**无需交接给**：PM、Architect（变更不涉及架构或需求调整）

---

## 8. 风险评估

| 风险 | 等级 | 缓解措施 |
|------|------|---------|
| 影响其他僵尸类型 | 低 | 类型检测确保只影响撑杆僵尸 |
| 动画切换不生效 | 低 | 使用已验证的 AnimationCommandComponent 机制 |
| 位置传送导致穿墙 | 低 | 目标位置使用现有配置常量 |

---

## 9. 审批记录

| 日期 | 操作 | 人员 |
|------|------|------|
| 2025-12-19 | 提案创建 | Bob (Scrum Master) |
| 2025-12-19 | 用户批准 | 用户 |

---

## 10. 变更日志

| 日期 | 版本 | 描述 | 作者 |
|------|------|------|------|
| 2025-12-19 | 1.0 | 初始提案创建 | Bob (Scrum Master) |
