# Sprint 变更提案：Level 1-2 向日葵教学系统

**日期**: 2025-11-25
**提案人**: Bob (Scrum Master)
**状态**: 已批准

---

## 1. 问题摘要

**问题描述**：Level 1-2 关卡缺少向日葵教学系统。

**设计文档参考**：`.meta/levels/chapter1.md` 第162-179行

**问题性质**：需求遗漏实现（非 PRD 变更）

**预期行为**：
- 游戏开始时显示箭头指向向日葵卡片，卡片闪烁
- 按条件/时间依次显示教学文字
- 种植≥3颗向日葵后教学自动完成

**当前行为**：
- `level-1-2.yaml` 中无 `tutorialSteps` 配置
- 无向日葵教学提示

---

## 2. Epic 影响

| Epic | 影响 |
|------|------|
| **Epic 8 - Story 8.6** | 需补充向日葵教学验收标准 |
| 其他 Epic | 无影响 |

**结论**：变更纳入现有 Story 8.6，无需创建新 Story。

---

## 3. 工件调整

| 工件 | 变更类型 |
|------|----------|
| `docs/prd/epic-8-level-chapter1-implementation.md` | 更新 Story 8.6 验收标准 |
| `data/levels/level-1-2.yaml` | 添加 `tutorialSteps` 配置 |
| `pkg/systems/tutorial_system.go` | 扩展新触发器和向日葵教学逻辑 |
| 测试文件 | 新增向日葵教学流程验证 |

---

## 4. 推荐方案

**方案**：直接调整（在 Story 8.6 范围内实现）

**理由**：
- 需求已明确定义，无歧义
- 现有 TutorialSystem 架构完全支持
- 工作量可控（2-3天）
- 风险低

---

## 5. 关键决策

| 决策点 | 结论 | 理由 |
|--------|------|------|
| 教学模式 | **提示性**（非强制） | Level 1-2 非纯教学关卡，玩家应有自由度 |
| 时间阈值 | **10秒**超时触发 | 给玩家足够反应时间，同时推进教学 |
| 完成条件 | 种植≥3颗向日葵后**自动隐藏** | 避免干扰正常游戏 |

---

## 6. 具体变更

### 6.1 更新 Story 8.6 验收标准

**文件**: `docs/prd/epic-8-level-chapter1-implementation.md`

**新增验收标准**:
```markdown
- Level 1-2 向日葵教学系统：
  - 游戏开始时显示箭头指向向日葵卡片，卡片闪烁
  - 按条件/时间依次显示教学文字（ADVICE_PLANT_SUNFLOWER1/2/3, ADVICE_MORE_SUNFLOWERS）
  - 种植≥3颗向日葵后教学自动完成
  - 提示性教学（非强制，玩家可忽略）
```

---

### 6.2 更新 level-1-2.yaml

**文件**: `data/levels/level-1-2.yaml`

**新增内容**:
```yaml
# 向日葵教学步骤配置（提示性教学）
tutorialSteps:
  # 步骤1：游戏开始，提示向日葵重要性
  - trigger: "gameStart"
    textKey: "ADVICE_PLANT_SUNFLOWER1"
    action: "sunflowerHint"

  # 步骤2：种植1颗向日葵或10秒后
  - trigger: "sunflowerCount1OrTimeout"
    textKey: "ADVICE_PLANT_SUNFLOWER2"
    action: "sunflowerHint"

  # 步骤3：种植2颗向日葵或10秒后
  - trigger: "sunflowerCount2OrTimeout"
    textKey: "ADVICE_MORE_SUNFLOWERS"
    action: "sunflowerHint"

  # 步骤4：20秒后仍不足3颗，再次提醒
  - trigger: "sunflowerReminder"
    textKey: "ADVICE_PLANT_SUNFLOWER3"
    action: "sunflowerHint"

  # 步骤5：种植≥3颗向日葵，教学完成
  - trigger: "sunflowerCount3"
    textKey: ""
    action: "completeTutorial"
```

---

### 6.3 扩展 TutorialSystem

**文件**: `pkg/systems/tutorial_system.go`

#### 新增状态变量
```go
sunflowerCount      int     // 向日葵种植计数
stepTimeElapsed     float64 // 当前步骤经过时间（用于超时触发）
```

#### 新增方法：findSunflowerCard()
```go
// findSunflowerCard 查找向日葵卡片实体
func (s *TutorialSystem) findSunflowerCard() ecs.EntityID {
    cardEntities := ecs.GetEntitiesWith1[*components.PlantCardComponent](s.entityManager)
    for _, cardID := range cardEntities {
        card, ok := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, cardID)
        if ok && card.PlantType == components.PlantSunflower {
            return cardID
        }
    }
    return 0
}
```

#### 新增触发器
```go
case "sunflowerCount1OrTimeout":
    return s.sunflowerCount >= 1 || s.stepTimeElapsed >= 10.0

case "sunflowerCount2OrTimeout":
    return s.sunflowerCount >= 2 || s.stepTimeElapsed >= 10.0

case "sunflowerReminder":
    return s.stepTimeElapsed >= 20.0 && s.sunflowerCount < 3

case "sunflowerCount3":
    return s.sunflowerCount >= 3
```

#### 向日葵计数逻辑
```go
// 在 updateTrackingState() 中添加
s.sunflowerCount = 0
plantEntities := ecs.GetEntitiesWith1[*components.PlantComponent](s.entityManager)
for _, plantID := range plantEntities {
    plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, plantID)
    if ok && plant.Type == components.PlantSunflower {
        s.sunflowerCount++
    }
}
```

---

## 7. 教学文字序列

| 触发条件 | TextKey | 文本内容 |
|----------|---------|----------|
| 游戏开始 | `ADVICE_PLANT_SUNFLOWER1` | "向日葵是极其重要的植物！" |
| 种植1颗或10秒超时 | `ADVICE_PLANT_SUNFLOWER2` | "请至少种下三棵向日葵！" |
| 种植2颗或10秒超时 | `ADVICE_MORE_SUNFLOWERS` | "向日葵越多，你种植物的速度越快！" |
| 20秒后仍不足3颗 | `ADVICE_PLANT_SUNFLOWER3` | "至少种下三棵向日葵，来提高你在僵尸进攻下的生存率！" |
| 种植≥3颗 | (无文本) | 教学完成，隐藏提示 |

---

## 8. 行动计划

| 步骤 | 任务 | 负责角色 |
|------|------|----------|
| 1 | 更新 Epic 8 文档，补充 Story 8.6 验收标准 | PO |
| 2 | 修改 `level-1-2.yaml` 添加 `tutorialSteps` | Dev |
| 3 | 扩展 `TutorialSystem` 实现新触发器 | Dev |
| 4 | 添加单元测试 | Dev |
| 5 | 手动测试游戏内效果 | QA |

---

## 9. 验证标准

### 功能验证
- [ ] 进入 level 1-2 后，向日葵卡片显示箭头指示器和闪烁效果
- [ ] 教学文字按序列正确显示
- [ ] 种植≥3颗向日葵后，箭头和教学文字自动隐藏
- [ ] 玩家可忽略教学提示，正常进行游戏

### 回归验证
- [ ] Level 1-1 强制性教学不受影响
- [ ] 其他关卡正常运行

---

## 10. 批准记录

| 角色 | 姓名 | 日期 | 状态 |
|------|------|------|------|
| Scrum Master | Bob | 2025-11-25 | ✅ 提案 |
| 用户 | - | 2025-11-25 | ✅ 批准 |

---

## 附录：设计文档原文

来源：`.meta/levels/chapter1.md` 第162-179行

```
#### 教学 - 选择与种植植物

- 和 level 1-1 在向日葵卡片显示相同的箭头加闪烁的效果，以及鼠标悬停的效果
  > 教学文字显示 （文本资源是预先加载好的，参考level 1-1 ）

  开始依次显示 ADVICE_PLANT_SUNFLOWER1(开始显示) ADVICE_PLANT_SUNFLOWER2 （种植一颗或到预定时长时显示） ADVICE_MORE_SUNFLOWERS （种植两颗或到预定时长时显示），  ADVICE_PLANT_SUNFLOWER3（如果过了比较久后，还不足三棵向日葵就再提示）

    [ADVICE_PLANT_SUNFLOWER1]
    向日葵是极其重要的植物！

    [ADVICE_PLANT_SUNFLOWER2]
    请至少种下三棵向日葵！

    [ADVICE_PLANT_SUNFLOWER3]
    至少种下三棵向日葵，来提高你在僵尸进攻下的生存率！

    [ADVICE_MORE_SUNFLOWERS]
    向日葵越多，你种植物的速度越快！
```
