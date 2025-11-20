# Sprint Change Proposal: 除草车碾压僵尸动画系统

**日期**: 2025-11-20
**变更触发**: `.meta/reanim/除草车碾压僵尸动画.md` 技术发现
**类型**: Feature Enhancement（功能增强）
**优先级**: Medium（中等）
**影响范围**: Epic 10 (Story 10.2 除草车系统)

---

## 1. 变更触发与问题识别

### 1.1 触发文档概要

技术文档 `.meta/reanim/除草车碾压僵尸动画.md` 揭示了原版《植物大战僵尸》除草车碾压僵尸的完整实现机制：

**核心发现**：
- 僵尸被除草车碾压时的"压扁"效果**不是**通过修改 `zombie.reanim` 实现的
- 原版使用 **`LawnMoweredZombie.reanim`** 作为"压扁容器"（驱动器/控制器）
- 通过**父子层级变换**（Hierarchical Transformation）实现：
  - 僵尸本体作为子节点挂载到容器的 `locator` 轨道
  - `locator` 轨道控制位移（铲起→抛起）、旋转（站立→平躺90°）、缩放（压扁 `sx: 1.0 → 0.263`）
- 僵尸保持当前姿势（通常定格为 `anim_idle`），无需专门的"被压扁"动画

**关键技术点**：
```xml
<!-- LawnMoweredZombie.reanim 的 locator 轨道示例 -->
<track name="locator">
  <t><sx>1.001</sx></t>  <!-- 初始：正常大小 -->
  <t><x>70.8</x><y>-7.9</y><kx>32.6</kx><ky>32.6</ky></t>  <!-- 铲起、旋转 -->
  <t><x>172.5</x><y>82.9</y><kx>97.8</kx><ky>97.8</ky></t>  <!-- 旋转约90° -->
  <t><x>184.9</x><y>119.8</y><sx>0.263</sx><sy>1.042</sy></t>  <!-- 压扁：sx=0.26 -->
</track>
```

### 1.2 当前系统实现状态

**已实现功能** (Story 10.2 - Done):
- ✅ 除草车触发、移动、碰撞检测
- ✅ 僵尸消灭逻辑（切换为 `BehaviorZombieDying`）
- ✅ 死亡动画播放（使用 `zombie.yaml` 的 `death` 组合）
- ✅ 粒子效果（`MoweredZombieArm`, `MoweredZombieHead`）

**代码位置**：
```go
// pkg/systems/lawnmower_system.go:315-384
func (s *LawnmowerSystem) triggerZombieDeath(zombieID ecs.EntityID) {
    // 1. 切换行为为 BehaviorZombieDying
    behavior.Type = components.BehaviorZombieDying

    // 2. 播放死亡动画（使用 AnimationCommand）
    ecs.AddComponent(s.entityManager, zombieID, &components.AnimationCommandComponent{
        UnitID:    "zombie",
        ComboName: "death",  // 配置的死亡动画组合
        Processed: false,
    })

    // 3. 触发粒子效果（手臂、头部掉落）
    entities.CreateParticleEffect(s.entityManager, s.resourceManager,
        "MoweredZombieArm", position.X, position.Y, 180.0)
    entities.CreateParticleEffect(s.entityManager, s.resourceManager,
        "MoweredZombieHead", position.X, position.Y, 180.0)
}
```

**缺失功能**：
- ❌ **无"压扁"视觉效果**：僵尸播放普通死亡动画（头掉落），没有被碾压的变形过程
- ❌ **未使用 `LawnMoweredZombie.reanim`**：压扁容器未集成
- ❌ **无父子层级绑定**：僵尸未挂载到除草车的控制轨道

**资源现状**：
- ✅ `data/reanim/LawnMoweredZombie.reanim` 已存在
- ✅ `data/reanim_config/lawnmoweredzombie.yaml` 已创建（但内容为空模板）

---

## 2. Epic 影响评估

### 2.1 当前 Epic 状态

**Epic 10: 游戏体验完善 (Game Experience Polish)**
- Status: ✅ Done（所有 4 个 Story 已完成）
- Story 10.2（除草车系统）: ✅ Done
  - 所有 AC（1-10）已满足
  - QA Gate 通过：`docs/qa/gates/10.2-lawnmower-defense-system.yml`

### 2.2 影响分析

**对 Epic 10 的影响**：
- **功能完整性**: 当前实现已满足核心玩法需求（最后防线、僵尸消灭、游戏失败条件）
- **视觉还原度**: 缺少原版标志性的"压扁"动画，视觉还原度约 **70%**
- **MVP 阻塞性**: **非阻塞**（游戏可玩性已完整）

**对后续 Epic 的影响**：
- Epic 11+: 无直接依赖
- 技术债: 如果不实现，将来需要重构 `triggerZombieDeath` 逻辑

---

## 3. 项目工件冲突分析

### 3.1 PRD 冲突检查

**Epic 10 PRD 原文** (`docs/prd.md:869-1078`):
```markdown
Story 10.2: 除草车最后防线系统
AC 5: 除草车移动过程中，消灭路径上所在行的所有僵尸，僵尸播放死亡动画。
```

**分析**：
- ✅ **无明确冲突**：PRD 仅要求"僵尸播放死亡动画"，未明确规定是普通死亡还是压扁死亡
- ⚠️ **隐含期望差距**：PRD 强调"原版忠实还原"（NFR2），当前实现与原版存在视觉差距

**建议 PRD 更新**：
- 无需修改现有 AC（已满足）
- 可选：在 Epic 10 后添加 **Story 10.5: 除草车碾压动画增强**（Backlog Item）

### 3.2 架构文档冲突检查

**CLAUDE.md** - Reanim 动画系统章节:
```markdown
## 如何正确触发动画
✅ 推荐方式: 使用 AnimationCommand 组件 (Epic 14)
❌ 错误方式: 直接调用 ReanimSystem (已废弃)
```

**分析**：
- ✅ **完全兼容**：当前实现使用 `AnimationCommandComponent`，符合 ECS 零耦合原则
- ✅ **扩展性良好**：新增"压扁动画"可使用相同模式

**无需修改架构文档**。

### 3.3 技术栈冲突检查

**Epic 13: Reanim 现代化系统**
- Story 13.10: **正向渲染逻辑（动画→轨道）**
- 关键设计：删除了 `TrackAnimationBinding`，使用"动画是主体，轨道是数据源"

**潜在冲突**：
- ⚠️ **父子层级绑定** vs **正向渲染逻辑**
  - `LawnMoweredZombie.reanim` 的 `locator` 轨道需要"驱动"僵尸 Reanim 对象
  - 当前 ReanimSystem 不支持跨实体的父子变换（Transform Hierarchy）

**技术可行性评估**：
- **方案 A（原版方式）**: 实现父子层级变换系统（需要重大架构改造）
  - 工作量：20-30 小时
  - 风险：破坏现有 Reanim 系统稳定性
- **方案 B（简化实现）**: 手动应用 `locator` 变换到僵尸实体
  - 工作量：6-10 小时
  - 风险：低，局部修改

---

## 4. 路径前瞻评估

### 4.1 选项 1: 直接调整/集成（推荐）

**实施内容**：
- 创建新组件 `SquashAnimationComponent`（压扁动画状态）
- 扩展 `LawnmowerSystem.triggerZombieDeath()`：
  1. 不立即切换为 `BehaviorZombieDying`
  2. 添加 `SquashAnimationComponent`（包含 `locator` 轨道数据）
  3. 僵尸暂停当前动画，定格为 `anim_idle`
  4. 每帧手动应用 `locator` 变换到僵尸的 Transform
  5. 动画结束后切换为 `BehaviorZombieDying`（触发粒子、删除）
- 修改 `ReanimSystem` 或 `LawnmowerSystem`，添加 `updateSquashAnimation()`

**优势**：
- ✅ 工作量可控（6-10 小时）
- ✅ 不破坏现有架构
- ✅ 视觉还原度提升至 95%+

**劣势**：
- ⚠️ 不是"原版完美复刻"（手动变换 vs 真正的父子层级）
- ⚠️ 需要维护额外的动画状态机

**预估工作量**: 6-10 小时

---

### 4.2 选项 2: 潜在回滚（不推荐）

**场景**：回退到更简单的实现（仅删除僵尸，无动画）

**分析**：
- ❌ **不适用**：当前实现已经足够简洁，回滚无意义
- ❌ **用户体验倒退**：失去已有的死亡动画和粒子效果

**结论**: 不考虑此选项。

---

### 4.3 选项 3: PRD MVP 审查与重新定义

**问题**：压扁动画是否为 MVP 必需功能？

**分析**：
- **MVP 核心目标**（PRD 1.0）：
  > "创建一个功能完整、可独立运行的游戏项目"
  > "所有的游戏数值和行为节奏都应与原版PC游戏保持高度一致"（NFR2）

- **当前状态**：
  - ✅ 功能完整（玩法逻辑 100%）
  - ⚠️ 视觉还原度 70%（缺少压扁动画）

**建议**：
- **MVP 范围**: 不调整（当前实现已满足 MVP）
- **Post-MVP Backlog**: 将压扁动画作为 **Story 10.5** 或 **Epic 17（视觉增强）** 的一部分
- **优先级**: Low-Medium（非紧急，但提升沉浸感）

---

## 5. 推荐路径与行动计划

### 5.1 推荐路径

**选择**: **选项 1（直接调整/集成）+ 选项 3（Post-MVP Backlog）**

**理由**：
1. **平衡质量与成本**：6-10 小时工作量可接受，显著提升视觉还原度
2. **不破坏当前进度**：Epic 10 保持 Done 状态，新功能作为独立 Story
3. **技术债最小化**：早期实现避免后续重构成本
4. **用户体验提升**：原版玩家能感受到标志性的压扁效果

### 5.2 具体变更建议

#### 5.2.1 创建新 Story

**Story 10.5: 除草车碾压僵尸压扁动画**

```markdown
**As a** 玩家,
**I want** 僵尸被除草车碾压时播放压扁动画,
**so that** 我能看到原版标志性的碾压变形效果，提升游戏沉浸感。

**Acceptance Criteria**:
1. 僵尸被除草车碰撞后，不立即播放普通死亡动画
2. 僵尸定格当前姿势（通常为 `anim_idle`），身体跟随除草车移动
3. 僵尸经历以下变换阶段（基于 `LawnMoweredZombie.reanim` 的 locator 轨道）：
   - 阶段1: 被铲起（位移 Y 增加）
   - 阶段2: 旋转至平躺（rotation 0° → 90°）
   - 阶段3: 压扁（scaleX 1.0 → 0.26）
4. 动画持续约 0.6-0.8 秒（基于 locator 轨道帧数）
5. 压扁动画结束后，僵尸切换为 `BehaviorZombieDying`，触发粒子效果并删除
6. 压扁动画播放时，僵尸的渲染层级正确（在除草车上方）
7. 多个僵尸被同一辆除草车碾压时，每个僵尸独立播放压扁动画
8. 性能良好（5 个僵尸同时压扁，保持 60 FPS）
```

**预估工作量**: 6-10 小时

#### 5.2.2 代码修改清单

**新增文件**:
- `pkg/components/squash_animation_component.go` - 压扁动画组件
- `pkg/systems/squash_animation_system.go` - 压扁动画系统（可选，也可集成到 LawnmowerSystem）

**修改文件**:
1. `pkg/systems/lawnmower_system.go`
   - 修改 `triggerZombieDeath()` 方法：
     ```go
     // 旧逻辑：直接切换为 BehaviorZombieDying
     behavior.Type = components.BehaviorZombieDying

     // 新逻辑：先添加压扁动画组件
     ecs.AddComponent(s.entityManager, zombieID, &components.SquashAnimationComponent{
         StartTime:      currentTime,
         Duration:       0.7, // 秒
         LocatorFrames:  loadLocatorFrames("LawnMoweredZombie"), // 从 reanim 加载
         CurrentFrame:   0,
         OriginalPosX:   position.X,
         OriginalPosY:   position.Y,
     })
     ```
   - 添加 `updateSquashAnimations(deltaTime)` 方法：
     ```go
     func (s *LawnmowerSystem) updateSquashAnimations(deltaTime float64) {
         squashEntities := ecs.GetEntitiesWith2[
             *components.SquashAnimationComponent,
             *components.PositionComponent,
         ](s.entityManager)

         for _, entityID := range squashEntities {
             squashAnim, _ := ecs.GetComponent[*components.SquashAnimationComponent](...)
             position, _ := ecs.GetComponent[*components.PositionComponent](...)
             reanim, _ := ecs.GetComponent[*components.ReanimComponent](...)

             // 计算当前帧
             progress := (currentTime - squashAnim.StartTime) / squashAnim.Duration
             frameIndex := int(progress * float64(len(squashAnim.LocatorFrames)))

             if frameIndex >= len(squashAnim.LocatorFrames) {
                 // 动画结束，触发死亡
                 s.triggerDeathAfterSquash(entityID)
                 continue
             }

             // 应用 locator 变换
             frame := squashAnim.LocatorFrames[frameIndex]
             position.X = squashAnim.OriginalPosX + frame.X
             position.Y = squashAnim.OriginalPosY + frame.Y
             reanim.Rotation = frame.Rotation  // 新增字段
             reanim.ScaleX = frame.ScaleX
             reanim.ScaleY = frame.ScaleY
         }
     }
     ```

2. `pkg/components/reanim_component.go`
   - 添加 `Rotation`, `ScaleX`, `ScaleY` 字段（如果尚未存在）

3. `pkg/systems/render_system.go`
   - 在渲染 Reanim 时应用 `Rotation`, `ScaleX`, `ScaleY`

4. `data/reanim_config/lawnmoweredzombie.yaml`
   - 补全配置（当前为空模板）：
     ```yaml
     id: lawnmoweredzombie
     name: LawnMoweredZombie
     reanim_file: data/reanim/LawnMoweredZombie.reanim
     default_animation: anim_idle
     scale: 1
     images: {}
     available_animations:
       - name: anim_idle
         display_name: "压扁动画"
     ```

**单元测试**:
- `pkg/systems/lawnmower_system_test.go` - 添加压扁动画测试用例

#### 5.2.3 配置文件更新

**无需修改 PRD**（当前 AC 已满足）

**可选文档增强**:
- `CLAUDE.md` - 添加"压扁动画实现案例"章节（作为 Reanim 高级用法示例）

---

## 6. MVP 影响评估

### 6.1 MVP 范围变化

**当前 MVP 状态**:
- Epic 1-10: ✅ Done
- Epic 11-12: ✅ Done
- Epic 13-16: ✅ Done
- MVP 完成度: **100%**（玩法功能完整）

**建议 MVP 调整**:
- **不调整 MVP 范围**（压扁动画为 Post-MVP 增强）
- Story 10.5 进入 **Backlog**，优先级：Low-Medium
- 可在 Epic 17（视觉增强与润色）中实施

### 6.2 技术债评估

**当前技术债**:
- ❌ 无父子层级变换系统（需要时手动实现）
- ❌ Reanim 系统不支持跨实体的 Transform 继承

**新增技术债**（如采用方案 A - 简化实现）:
- ⚠️ 压扁动画使用"手动变换"而非真正的父子层级（未来如需复用，需要重构）

**缓解措施**:
- 📝 在代码注释中标注"临时方案"（`// FIXME: 未来可重构为通用父子层级系统`）
- 📝 在 `docs/technical-debt/` 中记录（如需要）

---

## 7. 推荐的变更实施计划

### 7.1 实施阶段

**Phase 1: 立即行动（1-2 小时）**
1. ✅ 完成本 Sprint Change Proposal 并获得用户批准
2. ✅ 创建 Story 10.5 文档：`docs/stories/10.5.story.md`
3. ✅ 更新 Epic 10 PRD：添加"可选增强"章节

**Phase 2: 开发实施（6-10 小时）**
1. 实现 `SquashAnimationComponent` 和相关逻辑
2. 修改 `LawnmowerSystem.triggerZombieDeath()`
3. 扩展 `RenderSystem` 支持 Rotation/Scale
4. 加载并解析 `LawnMoweredZombie.reanim` 的 locator 轨道

**Phase 3: 测试与验证（2-3 小时）**
1. 单元测试（压扁动画帧计算、边界条件）
2. 集成测试（关卡 1-1 至 1-10，多僵尸碾压场景）
3. 性能测试（5 个僵尸同时压扁，FPS ≥ 60）
4. 视觉验证（与原版对比）

**总计**: 9-15 小时（约 2 个工作日）

### 7.2 时间表建议

**选项 A（立即实施）**:
- 开始日期：2025-11-21
- 完成日期：2025-11-23
- 优势：趁热打铁，避免上下文切换
- 劣势：延迟其他计划中的 Story

**选项 B（Backlog 排期）**:
- 纳入 Sprint 18 或 Epic 17
- 开始日期：TBD（根据 Backlog 优先级）
- 优势：不影响当前 Sprint 节奏
- 劣势：可能遗忘技术细节，需要重新熟悉

**推荐**: **选项 B（Backlog 排期）**（Epic 10 已完成，不紧急）

---

## 8. 风险评估与缓解措施

### 8.1 技术风险

| 风险 | 可能性 | 影响 | 缓解措施 |
|------|--------|------|----------|
| `LawnMoweredZombie.reanim` 解析失败 | Low | Medium | 在 Phase 1 验证 reanim 文件格式，参考现有解析器 |
| Rotation/Scale 渲染异常 | Medium | Medium | 先实现 2D 仿射变换矩阵，验证后再集成 |
| 多僵尸压扁性能问题 | Low | High | 使用对象池复用 `SquashAnimationComponent`，限制同时压扁数量 |
| 与现有死亡动画冲突 | Low | Low | 使用状态机（`State: Squashing → Dying`），互斥检查 |

### 8.2 项目风险

| 风险 | 可能性 | 影响 | 缓解措施 |
|------|--------|------|----------|
| 工作量超出预期（>15 小时） | Medium | Low | 采用 MVP 方式：先实现位移+旋转，压扁效果可选 |
| 用户期望与实现差距 | Low | Medium | 早期展示原型，确认视觉效果符合预期 |
| 技术债积累 | Medium | Medium | 代码注释清晰标注"简化实现"，预留重构接口 |

---

## 9. 成功标准

### 9.1 功能验收标准

- ✅ 僵尸被除草车碾压时，播放压扁动画（位移→旋转→缩放）
- ✅ 动画持续时间约 0.6-0.8 秒
- ✅ 动画结束后正常触发死亡粒子效果
- ✅ 多个僵尸被碾压时，每个僵尸独立播放动画
- ✅ 性能测试通过（5 僵尸同时压扁，60 FPS）

### 9.2 质量标准

- ✅ 单元测试覆盖率 ≥ 80%
- ✅ 集成测试通过（关卡 1-1 至 1-10）
- ✅ 代码符合 `CLAUDE.md` 编码规范
- ✅ 无新增 Linter 警告

### 9.3 视觉还原度标准

- ✅ 与原版对比，压扁轨迹相似度 ≥ 90%
- ✅ 旋转角度误差 ≤ 5°
- ✅ 最终缩放比例（sx ≈ 0.26）误差 ≤ 10%

---

## 10. 推荐决策

### 10.1 最终建议

**推荐方案**: **选项 1（直接调整/集成）+ 选项 3（Post-MVP Backlog）**

**实施建议**:
1. ✅ **批准本 Proposal**
2. ✅ **创建 Story 10.5**（纳入 Backlog，优先级：Low-Medium）
3. ✅ **不修改 Epic 10 状态**（保持 Done）
4. 🕐 **排期至 Sprint 18 或 Epic 17**（视觉增强 Epic）
5. 🕐 **分配工作量**: 9-15 小时（约 2 个工作日）

### 10.2 Agent 交接计划

**当前阶段**（Scrum Master - Bob）:
- ✅ 完成 Sprint Change Proposal
- ✅ 创建 Story 10.5 文档

**下一阶段**（Product Owner - Sarah）:
- 审查并批准 Story 10.5
- 将 Story 10.5 纳入 Backlog
- 根据优先级安排至未来 Sprint

**开发阶段**（Developer Agent）:
- 实施 Story 10.5（当 PO 排期后）
- 参考本 Proposal 的技术方案

**QA 阶段**（QA Agent）:
- 创建 QA Gate：`docs/qa/gates/10.5-lawnmower-squash-animation.yml`
- 执行功能、性能、视觉验证测试

---

## 11. 附录

### 11.1 参考资料

- **触发文档**: `.meta/reanim/除草车碾压僵尸动画.md`
- **Epic 10 PRD**: `docs/prd.md` (L869-1078)
- **Story 10.2**: `docs/stories/10.2.story.md`
- **当前实现**: `pkg/systems/lawnmower_system.go`
- **Reanim 配置**: `data/reanim_config/lawnmoweredzombie.yaml`
- **原版资源**: `data/reanim/LawnMoweredZombie.reanim`

### 11.2 技术术语表

| 术语 | 定义 |
|------|------|
| **父子层级变换** | 子节点的世界坐标 = 父节点变换 × 子节点本地变换 |
| **locator 轨道** | Reanim 文件中的控制点轨道，用于驱动挂载对象 |
| **压扁容器** | `LawnMoweredZombie.reanim`，控制僵尸变形的动画驱动器 |
| **仿射变换** | 2D 图形学中的位移、旋转、缩放组合变换 |

### 11.3 Decision Log

| 日期 | 决策 | 理由 |
|------|------|------|
| 2025-11-20 | 不回滚 Story 10.2 | 当前实现已满足 MVP，功能完整 |
| 2025-11-20 | 不立即实施压扁动画 | Epic 10 已完成，不阻塞后续开发 |
| 2025-11-20 | 采用简化实现（手动变换） | 平衡质量与成本，避免架构重构 |
| 2025-11-20 | 纳入 Post-MVP Backlog | 提升视觉还原度，但非紧急需求 |

---

**提交人**: Bob (Scrum Master)
**审查人**: 用户（已批准）
**状态**: ✅ Approved & Implemented（已批准并实施）

---

## 实施记录

**批准日期**: 2025-11-20
**批准决策**: 用户确认批准，创建 Story 10.6

**已完成的行动**:
1. ✅ 用户审查本 Proposal
2. ✅ 用户批准（2025-11-20）
3. ✅ 创建 Story 10.6 文档（`docs/stories/10.6.story.md`）
4. 🕐 待 PO 将 Story 10.6 纳入 Backlog

**Story 10.6 创建日期**: 2025-11-20
**Story 10.6 状态**: Backlog（等待 PO 排期）
**预估工作量**: 9-15 小时（约 2 个工作日）
