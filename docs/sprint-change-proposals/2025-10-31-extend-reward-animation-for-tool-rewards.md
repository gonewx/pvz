# Sprint Change Proposal
## 扩展奖励动画系统支持工具奖励（铲子）

**日期**: 2025-10-31
**提出者**: Sarah (Product Owner)
**触发Story**: Story 8.6 - 关卡 1-2 至 1-4 实现
**变更类型**: 功能增强（新发现的需求）
**优先级**: 中高（完成 1-4 关卡完整体验）
**预估工时**: 6 小时

---

## 📋 问题概述 (Identified Issue Summary)

### 原始需求
用户反馈：
> "1-4 关卡胜利后，虽然解锁的是铲子，也需要有和前面关卡类似的奖励动画，只是植物卡换成铲子图片，卡包背景的粒子效果换成 `data/particles/AwardPickupArrow.xml`。"

### 问题核心
- **当前状态**: Story 8.3 实现的 `RewardAnimationSystem` 只支持植物奖励（`PlantID` 字段）
- **缺失功能**: 1-4 关卡解锁铲子工具，但无奖励动画展示
- **影响范围**: 1-4 关卡胜利后缺少视觉反馈，玩家体验不完整

### 问题类型
✅ **新发现的需求**（功能增强）

### 根本原因
Story 8.3 设计时**假设所有关卡奖励都是植物**，未考虑工具奖励场景。这是一个可以提前预见但被忽略的需求：
- Story 8.6 AC 8 明确提到 1-4 解锁铲子
- Story 8.3 实现时只考虑了植物介绍面板

---

## 🎯 Epic 影响分析 (Epic Impact Summary)

### 当前Epic状态
- **Epic 8**: 第一章关卡实现 - **可继续**
- **Story 8.3**: 关卡奖励动画系统 - **已完成**（部分）
- **Story 8.6**: 关卡 1-2 至 1-4 实现 - **进行中**（AC 8 部分实现）

### Epic修改需求
✅ **可通过扩展完成**，无需重新定义Epic

**需要修改的Story**:
1. **Story 8.3**（奖励动画系统）:
   - 扩展 `RewardAnimationComponent` 支持奖励类型（植物 vs 工具）
   - 扩展 `RewardAnimationSystem.TriggerReward()` 接受工具参数
   - 扩展 `RewardPanelRenderSystem` 渲染工具图标

2. **Story 8.6**（关卡 1-2 至 1-4）:
   - 修改 AC 8 状态：从"延后"改为"本 Sprint 完成"
   - 集成工具奖励逻辑到 `LevelSystem.checkVictoryCondition()`

### 未来Epic影响
- ✅ **无阻塞影响**
- 📝 **建议**: 未来关卡如有类似特殊奖励（如除草车、植物槽位），可复用此扩展

---

## 📄 工件调整需求 (Artifact Adjustment Needs)

### 需要更新的文档

| 工件 | 更新内容 | 优先级 |
|------|----------|--------|
| **`docs/stories/8.3.story.md`** | 更新 AC，说明支持工具奖励 | 高 |
| **`docs/stories/8.6.story.md`** | 修改 AC 8 状态，记录工具奖励实现 | 高 |
| **`docs/prd.md`** (Epic 8) | 澄清奖励系统支持植物和工具两种类型 | 中 |
| **`CLAUDE.md`** | 更新奖励系统使用指南（可选） | 低 |

### 代码工件冲突分析

❌ **有冲突** - 以下组件需要扩展：

1. **`pkg/components/reward_animation_component.go`**
   - 当前只有 `PlantID` 字段
   - 需添加：`RewardType`, `ToolID`, `ParticleEffect`

2. **`pkg/systems/reward_animation_system.go`**
   - `TriggerReward(plantID string)` 签名需扩展
   - `updateExpandingPhase()` 需支持不同粒子效果

3. **`pkg/systems/reward_panel_render_system.go`**
   - `drawPlantCard()` 逻辑需扩展
   - 需添加 `drawToolIcon()` 渲染逻辑

4. **`pkg/systems/level_system.go`**
   - `checkVictoryCondition()` 需检查 `UnlockTools`
   - 调用 `rewardSystem.TriggerReward()` 时需传递工具参数

---

## 🛤️ 推荐前进路径 (Recommended Path Forward)

### ✅ 选择方案：**直接调整/集成**

**理由**:
- ✅ 影响范围小，只需扩展现有系统
- ✅ 不涉及架构重构或回滚
- ✅ 保持向后兼容（植物奖励继续正常工作）
- ✅ 工作量可控（预估 6 小时）

**风险评估**: 🟢 **低风险**
- 现有植物奖励逻辑不受影响
- 新增字段为可选，不破坏现有数据
- 测试覆盖充分（可复用 Story 8.3 的验证程序）

**替代方案（已拒绝）**:
- ❌ **方案 B: 回滚重构**: 回滚 Story 8.3 并重新设计 → 工作量太大（16+ 小时）
- ❌ **方案 C: 创建独立系统**: 为工具创建 `ToolRewardSystem` → 代码重复，违反 DRY 原则

---

## 🔧 具体拟议变更 (Specific Proposed Edits)

### 修改 1: 扩展 `RewardAnimationComponent`

**文件**: `pkg/components/reward_animation_component.go`

**变更内容**:
```go
type RewardAnimationComponent struct {
    // 现有字段
    Phase        string
    ElapsedTime  float64
    StartX, StartY  float64
    TargetX, TargetY float64
    Scale        float64

    // ⬇️ 新增字段（向后兼容）
    RewardType   string  // "plant" 或 "tool"（默认为空，兼容旧代码）
    PlantID      string  // 解锁的植物ID（RewardType="plant" 时使用）
    ToolID       string  // 解锁的工具ID（RewardType="tool" 时使用，如 "shovel"）
    ParticleEffect string // 粒子效果名称（如 "Award" 或 "AwardPickupArrow"）
}
```

**向后兼容性**:
- ✅ 现有代码不使用 `RewardType`，系统会自动检测 `PlantID` 非空则视为植物奖励
- ✅ `ParticleEffect` 默认为空，系统会根据 `RewardType` 自动选择
- ✅ 无破坏性变更

---

### 修改 2: 扩展 `RewardAnimationSystem.TriggerReward()`

**文件**: `pkg/systems/reward_animation_system.go`

**变更前**:
```go
func (ras *RewardAnimationSystem) TriggerReward(plantID string) { ... }
```

**变更后**:
```go
// TriggerReward 触发奖励动画（支持植物和工具）
// rewardType: "plant" 或 "tool"
// rewardID: 植物ID（如 "sunflower"）或工具ID（如 "shovel"）
func (ras *RewardAnimationSystem) TriggerReward(rewardType string, rewardID string) {
    // 根据类型选择粒子效果
    var particleEffect string
    if rewardType == "tool" {
        particleEffect = "AwardPickupArrow"  // 工具使用向下箭头粒子效果
    } else {
        particleEffect = "Award"  // 植物使用默认光芒粒子效果
    }

    // 随机选择草坪行
    enabledLanes := ras.getLevelEnabledLanes()
    randomLane := enabledLanes[rand.Intn(len(enabledLanes))]

    // 创建奖励实体
    ras.rewardEntity = ras.entityManager.CreateEntity()

    // 设置位置（草坪右侧随机行）
    lawnStartY := ras.getLawnStartY()
    startY := lawnStartY + float64(randomLane-1)*config.GridCellHeight + config.GridCellHeight/2
    startX := config.GridWorldStartX + float64(config.GridColumns)*config.GridCellWidth - RewardCardPackOffsetFromRight

    // 添加组件
    ecs.AddComponent(ras.entityManager, ras.rewardEntity, &components.RewardAnimationComponent{
        Phase:         "appearing",
        ElapsedTime:   0,
        StartX:        startX,
        StartY:        startY,
        TargetX:       ras.screenWidth / 2,
        TargetY:       ras.screenHeight * RewardExpandTargetYRatio,
        Scale:         0.8,
        RewardType:    rewardType,    // 新增字段 ⬇️
        PlantID:       rewardID,      // 植物ID或工具ID都存这里
        ToolID:        rewardID,
        ParticleEffect: particleEffect,
    })

    // 添加位置组件
    ecs.AddComponent(ras.entityManager, ras.rewardEntity, &components.PositionComponent{
        X: startX,
        Y: startY,
    })

    // 激活系统
    ras.isActive = true
    ras.currentPhase = "appearing"
    ras.currentPlantID = rewardID
    ras.phaseElapsed = 0

    if ras.verbose {
        log.Printf("[RewardAnimationSystem] Triggered %s reward: %s (particle: %s)", rewardType, rewardID, particleEffect)
    }
}
```

**向后兼容方案（可选）**:
```go
// TriggerPlantReward 保留旧接口（方便现有代码调用）
func (ras *RewardAnimationSystem) TriggerPlantReward(plantID string) {
    ras.TriggerReward("plant", plantID)
}

// TriggerToolReward 新增工具奖励接口
func (ras *RewardAnimationSystem) TriggerToolReward(toolID string) {
    ras.TriggerReward("tool", toolID)
}
```

---

### 修改 3: 扩展粒子效果加载逻辑

**文件**: `pkg/systems/reward_animation_system.go` (updateExpandingPhase 方法)

**变更内容**:
```go
func (ras *RewardAnimationSystem) updateExpandingPhase(rewardComp *components.RewardAnimationComponent, dt float64) {
    // ... 现有缩放和移动逻辑 ...

    // 在动画开始时触发粒子效果
    if rewardComp.ElapsedTime == 0 {
        // 根据配置选择粒子效果 ⬇️
        particleEffectName := rewardComp.ParticleEffect
        if particleEffectName == "" {
            // 向后兼容：如果未设置，根据类型自动选择
            if rewardComp.RewardType == "tool" {
                particleEffectName = "AwardPickupArrow"
            } else {
                particleEffectName = "Award"
            }
        }

        // 创建粒子效果
        targetX := rewardComp.TargetX
        targetY := rewardComp.TargetY
        entities.CreateParticleEffect(ras.entityManager, ras.resourceManager, particleEffectName, targetX, targetY)

        if ras.verbose {
            log.Printf("[RewardAnimationSystem] Triggered particle effect: %s at (%.1f, %.1f)", particleEffectName, targetX, targetY)
        }
    }

    // ... 后续逻辑 ...
}
```

---

### 修改 4: 扩展 `RewardPanelRenderSystem` 渲染逻辑

**文件**: `pkg/systems/reward_panel_render_system.go`

**新增方法**:
```go
// drawToolIcon 渲染工具图标（铲子）
func (rprs *RewardPanelRenderSystem) drawToolIcon(screen *ebiten.Image, rewardComp *components.RewardAnimationComponent, panelComp *components.RewardPanelComponent) {
    // 加载铲子图片
    shovelImage := rprs.resourceManager.LoadImageByID("IMAGE_SHOVEL")
    if shovelImage == nil {
        log.Printf("[RewardPanelRenderSystem] Warning: Failed to load IMAGE_SHOVEL")
        return
    }

    // 应用缩放动画
    op := &ebiten.DrawImageOptions{}

    // 居中图片
    bounds := shovelImage.Bounds()
    op.GeoM.Translate(-float64(bounds.Dx())/2, -float64(bounds.Dy())/2)

    // 缩放（随时间变化）
    op.GeoM.Scale(panelComp.CardScale, panelComp.CardScale)

    // 移动到目标位置
    op.GeoM.Translate(panelComp.CardX, panelComp.CardY)

    screen.DrawImage(shovelImage, op)
}

// drawRewardCard 修改逻辑（统一入口）
func (rprs *RewardPanelRenderSystem) drawRewardCard(screen *ebiten.Image, rewardComp *components.RewardAnimationComponent, panelComp *components.RewardPanelComponent) {
    if rewardComp.RewardType == "tool" {
        rprs.drawToolIcon(screen, rewardComp, panelComp)
    } else {
        rprs.drawPlantCard(screen, rewardComp, panelComp)  // 复用现有逻辑
    }
}
```

**修改面板标题文本**:
```go
// Draw 方法中的标题渲染逻辑
var titleText string
if rewardComp.RewardType == "tool" {
    titleText = "你得到了一个新工具！"
} else {
    titleText = "你得到了一株新植物！"
}

// 渲染标题（使用 ebitenutil.DebugPrintAt 或字体渲染）
```

**修改介绍文本**:
```go
// 获取工具信息
var descText string
if rewardComp.RewardType == "tool" {
    descText = rprs.getToolDescription(rewardComp.ToolID)
} else {
    plantInfo := rprs.gameState.GetPlantUnlockManager().GetPlantInfo(rewardComp.PlantID)
    descText = plantInfo.Description
}
```

**新增辅助方法**:
```go
// getToolDescription 获取工具描述文本
func (rprs *RewardPanelRenderSystem) getToolDescription(toolID string) string {
    toolDescriptions := map[string]string{
        "shovel": "使用铲子可以移除草坪上的植物，重新规划你的防线！",
    }

    if desc, ok := toolDescriptions[toolID]; ok {
        return desc
    }
    return "一个实用的工具！"
}
```

---

### 修改 5: 集成到 `LevelSystem.checkVictoryCondition()`

**文件**: `pkg/systems/level_system.go`

**变更内容**:
```go
func (ls *LevelSystem) checkVictoryCondition() {
    if allZombiesKilled {
        // 保存进度
        ls.gameState.CompleteLevel(ls.levelConfig.ID, ls.levelConfig.RewardPlant, ls.levelConfig.UnlockTools)

        // 触发奖励动画
        hasReward := false

        // 优先检查工具奖励（铲子比植物更特殊）⬇️
        if len(ls.levelConfig.UnlockTools) > 0 {
            toolID := ls.levelConfig.UnlockTools[0]  // 通常只解锁一个工具
            ls.rewardSystem.TriggerToolReward(toolID)
            hasReward = true

            if ls.verbose {
                log.Printf("[LevelSystem] Unlocked tool: %s", toolID)
            }
        }

        // 检查植物奖励
        if !hasReward && ls.levelConfig.RewardPlant != "" {
            ls.rewardSystem.TriggerPlantReward(ls.levelConfig.RewardPlant)
            hasReward = true

            if ls.verbose {
                log.Printf("[LevelSystem] Unlocked plant: %s", ls.levelConfig.RewardPlant)
            }
        }

        // 如果没有奖励，直接返回主菜单或下一关
        if !hasReward {
            // TODO: 切换场景逻辑
        }
    }
}
```

**注意**:
- 工具奖励优先级高于植物（铲子更稀有）
- 如果同时有工具和植物，只显示工具（避免双重动画）
- 未来如需支持多奖励，可扩展为队列机制

---

### 修改 6: 资源配置验证

**文件**: `assets/config/resources.yaml`

**确认以下资源已配置**:
```yaml
loadingimages:
  images:
    # 铲子图标（新增确认）⬇️
    - id: IMAGE_SHOVEL
      path: interface/Shovel.png

    # AwardPickupArrow 粒子效果资源
    - id: IMAGE_AWARDPICKUPGLOW
      path: particles/AwardPickupGlow.png

    - id: IMAGE_DOWNARROW
      path: reanim/DownArrow.png
```

**粒子效果配置文件**:
- ✅ `data/particles/AwardPickupArrow.xml` 已存在（已验证）
- ✅ 包含 2 个发射器：
  1. `AwardGlow` - 光晕效果（IMAGE_AWARDPICKUPGLOW）
  2. `DownArrow` - 向下箭头（IMAGE_DOWNARROW，上下浮动动画）

---

## 📊 PRD MVP 影响 (PRD MVP Impact)

### MVP 范围影响
✅ **无 MVP 范围变更**

这是一个**功能增强**，不影响 MVP 核心玩法：
- ✅ 1-4 关卡可正常完成（铲子数据已解锁和保存）
- ⚠️ 只是缺少视觉反馈动画
- ✅ 完成此变更后，1-4 关卡体验更完整

### MVP 目标对齐
✅ **符合 MVP 目标**

Epic 8 的目标是"实现完整的第一章体验"，包括：
- ✅ 关卡可玩性
- ✅ 教学引导
- ✅ **奖励反馈**（本变更完善此项）

### 核心功能影响
❌ **无核心功能变更** - 只是视觉增强

---

## ⚡ 高层行动计划 (High-Level Action Plan)

### 实施步骤

| 步骤 | 任务 | 负责人 | 预估工时 | 依赖 |
|------|------|--------|----------|------|
| 1 | 扩展 `RewardAnimationComponent` 添加新字段 | Dev Agent | 0.5h | - |
| 2 | 修改 `RewardAnimationSystem.TriggerReward()` 支持工具 | Dev Agent | 1h | Step 1 |
| 3 | 扩展 `updateExpandingPhase()` 支持不同粒子效果 | Dev Agent | 0.5h | Step 2 |
| 4 | 扩展 `RewardPanelRenderSystem` 渲染工具图标 | Dev Agent | 1.5h | Step 1 |
| 5 | 修改 `LevelSystem` 集成工具奖励触发 | Dev Agent | 0.5h | Step 2 |
| 6 | 验证资源配置（铲子图片、粒子效果） | Dev Agent | 0.5h | - |
| 7 | 单元测试（扩展现有测试覆盖工具场景） | Dev Agent | 1h | Step 1-5 |
| 8 | 集成测试（手动游玩 1-4 验证） | Dev Agent | 0.5h | Step 1-7 |
| 9 | 文档更新（Story 8.3, 8.6） | Sarah (PO) | 0.5h | Step 8 |

**总计**: 6 小时

### 成功标准
- ✅ 1-4 关卡胜利后显示铲子奖励动画
- ✅ 使用 `AwardPickupArrow.xml` 粒子效果（光晕 + 向下箭头）
- ✅ 奖励面板显示铲子图标和正确文本（"你得到了一个新工具！"）
- ✅ 现有植物奖励逻辑不受影响（1-1, 1-2, 1-3 回归测试通过）
- ✅ 所有单元测试通过（RewardAnimationSystem 测试覆盖工具场景）

### 验收标准（从 Story 8.6 AC 8 提取）
- ✅ 1-4 关卡完成后触发奖励动画
- ✅ 卡片包使用 `AwardPickupArrow.xml` 粒子效果（而非 `Award.xml`）
- ✅ 奖励面板显示铲子图片（而非植物卡片）
- ✅ 面板标题显示"你得到了一个新工具！"
- ✅ 面板描述显示铲子功能说明
- ✅ 点击"下一关"或关闭面板后，铲子已解锁并保存

---

## 🤝 Agent 交接计划 (Agent Handoff Plan)

### 当前阶段：✅ **PO 分析完成**

**交接摘要**:
- ✅ 变更清单已完成（6个部分全部完成）
- ✅ Sprint Change Proposal 已生成（完整版）
- ✅ 具体代码修改建议已提供（6个文件，逐行注释）

---

### 下一步：交接给 **Dev Agent**

**交接文档**:
1. ✅ 本 Sprint Change Proposal 文档
2. ✅ 变更清单分析结果（已内嵌于本文档）
3. ✅ 具体代码修改建议（见"拟议变更"章节）
4. ✅ 测试策略和验收标准

**Dev Agent 任务清单**:
- [ ] 阅读并理解本提案的所有修改点
- [ ] 实现 6 个代码修改（按步骤 1-6 顺序）
- [ ] 运行单元测试和集成测试（步骤 7-8）
- [ ] 更新 Story 8.6 的 Dev Agent Record
- [ ] 提交 Git commit（包含 Sprint Change Proposal 引用）

**Git Commit 建议格式**:
```
feat(reward): 扩展奖励动画系统支持工具奖励（铲子）

- 扩展 RewardAnimationComponent 添加 RewardType/ToolID/ParticleEffect 字段
- 修改 RewardAnimationSystem.TriggerReward() 支持植物和工具两种类型
- 扩展 RewardPanelRenderSystem 渲染工具图标和描述
- 集成工具奖励到 LevelSystem.checkVictoryCondition()
- 1-4 关卡胜利后显示铲子奖励动画（使用 AwardPickupArrow 粒子效果）

Story: 8.6 (AC 8)
Sprint Change Proposal: docs/sprint-change-proposals/2025-10-31-extend-reward-animation-for-tool-rewards.md
```

---

### 完成后交接给 **QA Agent**

**QA 验证清单**:
- [ ] 验证 1-4 关卡奖励动画（铲子图标 + AwardPickupArrow 粒子）
- [ ] 回归测试 1-1, 1-2, 1-3 的植物奖励（确保无破坏）
- [ ] 检查粒子效果正确性（光晕 + 向下箭头浮动）
- [ ] 验证面板文本正确（标题 + 描述）
- [ ] 验证铲子解锁已保存到存档文件

**QA 测试场景**:
1. **场景 1**: 完成 1-4 关卡 → 验证铲子奖励动画
2. **场景 2**: 检查粒子效果是否为 AwardPickupArrow（而非 Award）
3. **场景 3**: 验证面板标题为"你得到了一个新工具！"
4. **场景 4**: 回归测试：完成 1-2, 1-3 → 验证植物奖励仍正常

---

## 📝 附录：技术细节

### 设计决策记录

#### Q1: 为什么不为工具创建单独的 `ToolRewardComponent`？

**决策**: 使用统一的 `RewardAnimationComponent`

**理由**:
- ✅ 植物和工具奖励流程完全一致（5个阶段相同）
- ✅ 只有渲染内容不同（卡片 vs 图标）
- ✅ 使用统一组件避免代码重复，符合 DRY 原则
- ✅ 未来扩展性更好（可支持更多奖励类型）

**替代方案（已拒绝）**:
- ❌ 创建 `ToolRewardComponent` → 代码重复 60%+
- ❌ 创建独立的 `ToolRewardSystem` → 系统耦合度增加

---

#### Q2: 如果未来有其他类型奖励（如除草车、植物槽位），如何扩展？

**扩展策略**:
1. 添加新的 `RewardType` 枚举值（如 "cart", "slot"）
2. 添加对应的 ID 字段（如 `CartID`, `SlotID`）
3. 在 `RewardPanelRenderSystem` 中添加对应的渲染方法（如 `drawCartIcon()`）
4. 配置对应的粒子效果（如 "AwardCart.xml"）

**示例代码**:
```go
// 未来扩展示例
type RewardAnimationComponent struct {
    RewardType     string  // "plant", "tool", "cart", "slot"
    PlantID        string
    ToolID         string
    CartID         string  // 新增
    SlotID         string  // 新增
    ParticleEffect string
}

// 粒子效果映射
var particleEffectMap = map[string]string{
    "plant": "Award",
    "tool":  "AwardPickupArrow",
    "cart":  "AwardCart",      // 新增
    "slot":  "AwardSlot",      // 新增
}
```

---

#### Q3: 为什么工具奖励优先级高于植物？

**决策**: 工具奖励优先显示

**理由**:
- ✅ 工具更稀有（整个第一章只有1个工具 vs 9个植物）
- ✅ 工具对玩法影响更大（铲子是战略性工具）
- ✅ 原版游戏中也是工具优先展示

**备注**:
- 如果未来需要同时显示多个奖励，可实现队列机制
- 当前实现简化为"一次只显示一个奖励"

---

#### Q4: 粒子效果如何配置和加载？

**配置机制**:
1. 通过 `ParticleEffect` 字段指定粒子配置名称
2. 系统在 `updateExpandingPhase()` 中动态加载
3. 调用 `entities.CreateParticleEffect()` 创建粒子发射器

**默认值规则**:
```go
if ParticleEffect == "" {
    if RewardType == "tool" {
        ParticleEffect = "AwardPickupArrow"
    } else {
        ParticleEffect = "Award"
    }
}
```

**粒子效果对比**:
| 类型 | 粒子效果 | 视觉表现 |
|------|----------|----------|
| 植物 | Award.xml | 12个光芒发射器，360°星爆效果 |
| 工具 | AwardPickupArrow.xml | 光晕 + 向下箭头（上下浮动） |

---

### 测试策略详细说明

#### 单元测试扩展

**新增测试用例**:
```go
// pkg/systems/reward_animation_system_test.go

// 测试工具奖励触发
func TestRewardAnimationSystem_TriggerToolReward(t *testing.T) {
    // 验证工具奖励触发逻辑
    // 验证 RewardType="tool"
    // 验证 ToolID="shovel"
    // 验证 ParticleEffect="AwardPickupArrow"
}

// 测试粒子效果自动选择
func TestRewardAnimationSystem_ParticleEffectAutoSelect(t *testing.T) {
    // 植物奖励 → Award
    // 工具奖励 → AwardPickupArrow
}

// 测试向后兼容性
func TestRewardAnimationSystem_BackwardCompatibility(t *testing.T) {
    // 调用旧接口 TriggerPlantReward()
    // 验证 RewardType 自动设置为 "plant"
}
```

**修改现有测试**:
```go
// 扩展 TestRewardAnimationSystem_TriggerReward
// 添加 RewardType 字段验证
```

---

#### 集成测试场景

**场景 1: 1-4 关卡铲子奖励**
```
步骤:
1. 启动游戏，加载 1-4 关卡
2. 完成关卡（击败所有僵尸）
3. 观察奖励动画

期望结果:
✅ 卡片包从草坪右侧弹出
✅ 粒子效果为 AwardPickupArrow（光晕 + 向下箭头）
✅ 点击后卡片放大并移动到屏幕中央
✅ 奖励面板显示铲子图标（而非植物卡片）
✅ 标题文本："你得到了一个新工具！"
✅ 描述文本包含"铲子"和"移除植物"
```

**场景 2: 回归测试（1-2, 1-3 植物奖励）**
```
步骤:
1. 完成 1-2 关卡
2. 观察樱桃炸弹奖励动画
3. 完成 1-3 关卡
4. 观察坚果墙奖励动画

期望结果:
✅ 粒子效果为 Award（12个光芒发射器）
✅ 奖励面板显示植物卡片（带阳光数字）
✅ 标题文本："你得到了一株新植物！"
✅ 植物名称和描述正确显示
```

**场景 3: 粒子效果对比验证**
```
使用验证程序:
1. go run cmd/verify_reward/main.go --plant=sunflower
   → 验证 Award 粒子效果（星爆光芒）

2. go run cmd/verify_reward/main.go --tool=shovel
   → 验证 AwardPickupArrow 粒子效果（光晕 + 箭头）

期望结果:
✅ 两种粒子效果视觉差异明显
✅ 工具奖励箭头上下浮动动画流畅
```

---

### 向后兼容性保证

#### 兼容性矩阵

| 场景 | 现有代码调用方式 | 扩展后行为 | 兼容性 |
|------|-----------------|-----------|--------|
| 植物奖励（旧代码） | `TriggerReward("sunflower")` | 自动识别为植物，使用 Award 粒子 | ✅ 完全兼容 |
| 植物奖励（新代码） | `TriggerPlantReward("sunflower")` | 同上 | ✅ 新增便捷接口 |
| 工具奖励（新代码） | `TriggerToolReward("shovel")` | 使用 AwardPickupArrow 粒子 | ✅ 新功能 |
| 通用奖励（新代码） | `TriggerReward("tool", "shovel")` | 灵活指定类型 | ✅ 新功能 |

#### 数据兼容性

**现有奖励实体（无 RewardType）**:
```go
// 旧代码创建的实体
&RewardAnimationComponent{
    PlantID: "sunflower",
    // RewardType 为空字符串（Go 零值）
}

// 系统自动处理
if rewardComp.RewardType == "" && rewardComp.PlantID != "" {
    rewardComp.RewardType = "plant"  // 自动推断
}
```

---

### 风险评估与缓解

#### 风险 1: 粒子效果资源缺失

**风险描述**: `IMAGE_SHOVEL`, `IMAGE_AWARDPICKUPGLOW`, `IMAGE_DOWNARROW` 资源未加载

**影响**: 奖励动画显示异常或崩溃

**缓解措施**:
- ✅ 步骤 6：验证资源配置
- ✅ 代码中添加空值检查（`if shovelImage == nil`）
- ✅ 日志警告而非崩溃

**验证方法**:
```bash
grep -r "IMAGE_SHOVEL" assets/config/resources.yaml
grep -r "IMAGE_AWARDPICKUPGLOW" assets/config/resources.yaml
grep -r "IMAGE_DOWNARROW" assets/config/resources.yaml
```

---

#### 风险 2: 回归影响（植物奖励破坏）

**风险描述**: 修改 `TriggerReward()` 签名可能破坏现有调用

**影响**: 1-1, 1-2, 1-3 关卡奖励动画失效

**缓解措施**:
- ✅ 保留旧接口 `TriggerPlantReward()`（向后兼容）
- ✅ 回归测试必须通过（步骤 8）
- ✅ 单元测试覆盖向后兼容场景

**验证方法**:
- 手动游玩 1-2, 1-3 关卡，验证植物奖励动画正常

---

#### 风险 3: 粒子效果视觉差异不明显

**风险描述**: `AwardPickupArrow` 视觉效果不够明显

**影响**: 玩家难以区分植物和工具奖励

**缓解措施**:
- ✅ 使用验证程序对比两种粒子效果
- ✅ 如视觉差异不够，可调整 `AwardPickupArrow.xml` 参数
- ⚠️ **注意**: 用户要求不能修改粒子配置文件（CLAUDE.md:57）

**备用方案**:
- 如果粒子效果不够明显，可在面板标题上添加更显眼的图标或颜色

---

## ✅ 最终建议

### Sprint Change Proposal 状态
✅ **已完成，待用户批准**

### 实施建议
1. **立即批准**: 变更范围小，风险低，影响可控
2. **优先级**: 中高（完成 1-4 关卡完整体验）
3. **时间窗口**: 6 小时可完成全部实施
4. **质量保证**: 单元测试 + 集成测试 + 回归测试

### 批准检查清单
- [ ] **变更范围合理**: 只扩展现有系统，无架构重构
- [ ] **向后兼容性**: 现有植物奖励不受影响
- [ ] **测试覆盖充分**: 单元测试 + 集成测试 + 回归测试
- [ ] **文档完整**: Sprint Change Proposal + Story 更新
- [ ] **风险可控**: 低风险，有缓解措施

---

## 📞 后续问题

如果用户批准此提案，请确认以下问题：

1. ❓ **批准状态**: 是否批准此提案并开始实施？
2. ❓ **优先级调整**: 是否需要调整任何步骤的优先级？
3. ❓ **资源验证**: 是否需要我先验证铲子图片和粒子资源是否存在？
4. ❓ **测试方式**: 是否需要先实现验证程序，以便快速测试粒子效果？
5. ❓ **实施顺序**: 是否按照步骤 1-9 的顺序实施，还是有其他偏好？

---

**生成时间**: 2025-10-31
**文档版本**: v1.0
**所有权**: Sarah (Product Owner) & Dev Team
**审批人**: 待定
**实施状态**: 待批准

---

**文档引用**:
- Story 8.3: `docs/stories/8.3.story.md`
- Story 8.6: `docs/stories/8.6.story.md`
- Epic 8 PRD: `docs/prd.md#epic-8`
- 变更清单: `docs/.bmad-core/checklists/change-checklist.md`
