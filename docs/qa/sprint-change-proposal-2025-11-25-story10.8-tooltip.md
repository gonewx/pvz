# Sprint Change Proposal - 2025-11-25

## Story 10.8 植物卡片 Tooltip 功能补充

**提案日期**: 2025-11-25
**提案人**: Bob (Scrum Master)
**提案类型**: 需求补充
**影响范围**: Story 10.8, Epic 10, PRD, 架构文档

---

## 执行总结 (Executive Summary)

用户在审查 Story 10.8 (阳光不足交互反馈) 时,发现缺少原版《植物大战僵尸》的**植物卡片悬停提示(Tooltip)系统**。本提案建议在 Story 10.8 中集成 Tooltip 功能,以实现完整的卡片交互用户体验。

**关键变更**:
- Story 10.8 扩展: 阳光闪烁 + Tooltip 系统 + 光标切换
- 工作量调整: 4-6 小时 → 7-10 小时 (+3-4 小时)
- 优先级提升: ⭐⭐⭐ 中 → ⭐⭐⭐⭐ 中高
- 文档更新: Story 10.8, Epic 10, PRD Epic 3, 架构文档

**影响评估**:
- ✅ 无技术风险,无需回滚现有工作
- ✅ Epic 10 总工作量仍可控 (58-84 小时)
- ✅ MVP 目标保持不变

---

## 1. 变更触发原因 (Change Trigger)

### 原始需求
Story 10.8 原计划仅实现"阳光不足点击反馈":
- 点击阳光不足卡片时,阳光计数器闪烁 (红/黑)
- 播放无效操作音效

### 发现的缺失功能
用户发现缺少原版 PvZ 的植物卡片 **Tooltip 系统**:
1. 鼠标悬停卡片时无提示框
2. 无状态相关提示(冷却中/阳光不足)
3. 鼠标光标未根据卡片状态切换

### 补充需求
- **提示框 UI**: 黑边浅黄色背景,显示植物名和状态
- **状态提示**:
  - 冷却中: "重新装填中..." (红色小号,第一行)
  - 阳光不足: "没有足够的阳光" (红色小号,第一行)
  - 植物名 (黑色正常,第二行)
- **光标切换**: 可点击状态显示手形,不可点击状态显示默认箭头

---

## 2. Epic 影响分析 (Epic Impact Analysis)

### Epic 10 - 游戏体验完善

**当前进度**: 6/8 Story 完成

**Story 10.8 变更**:
- **原标题**: 阳光不足交互反馈
- **新标题**: 植物卡片交互反馈增强
- **原工作量**: 4-6 小时
- **新工作量**: 7-10 小时 (+3-4 小时)
- **原优先级**: ⭐⭐⭐ 中
- **新优先级**: ⭐⭐⭐⭐ 中高

**Epic 总工作量调整**:
- 原总工作量: 55-80 小时
- 新总工作量: 58-84 小时
- 剩余工作量: 10-14 小时 → 13-18 小时

**结论**: ✅ Epic 10 可正常完成,无需拆分或回滚

---

## 3. 文档更新清单 (Documentation Updates)

### 3.1 Story 10.8 文档

**文件**: `docs/stories/10.8-sun-shortage-feedback.md`

**更新内容**:
1. **Metadata** 更新:
   - 标题: 植物卡片交互反馈增强
   - 优先级: 中 → 中高
   - 工作量: 4-6 小时 → 7-10 小时
   - 变更记录: 新增 2025-11-25 更新

2. **Story 描述** 更新:
   - 从"阳光不足点击反馈"扩展为"完整的卡片交互反馈(悬停 + 点击)"

3. **新增 AC 11-15** (Tooltip 系统):
   - AC 11: 鼠标悬停显示 Tooltip
   - AC 12: Tooltip 内容显示 (状态提示在第一行,植物名在第二行)
   - AC 13: 鼠标光标状态切换
   - AC 14: Tooltip 渲染层级
   - AC 15: 性能要求

4. **新增 Phase 5 Tasks** (Tooltip 实现):
   - Task 5.1: 创建 `TooltipComponent` 组件
   - Task 5.2: 修改 `InputSystem` 处理鼠标悬停
   - Task 5.3: 实现鼠标光标切换逻辑
   - Task 5.4: 创建 `TooltipRenderSystem` 或扩展 `UIRenderSystem`
   - Task 5.5: 集成测试

5. **新增技术实现** (Tooltip 系统架构):
   - TooltipComponent 组件定义
   - InputSystem 集成逻辑
   - 渲染系统伪代码

6. **更新成功标准** (新增 4 项 Tooltip 相关标准)

7. **更新 Definition of Done** (包含 Tooltip 验收)

---

### 3.2 Epic 10 文档

**文件**: `docs/prd/epic-10-game-experience-polish.md`

**更新内容**:
1. **Story 10.8 描述** (L225-238):
   - 标题更新为"植物卡片交互反馈增强"
   - 新增 Tooltip 系统功能描述
   - 明确状态提示在第一行,植物名在第二行

2. **时间估算表格** (L334):
   - Story 10.8 工作量: 4-6 小时 → 7-10 小时
   - 优先级: 中 → 中高
   - 总工作量: 55-80 小时 → 58-84 小时
   - 剩余工作量: 10-14 小时 → 13-18 小时

---

### 3.3 PRD 文档

**文件**: `docs/prd/epic-3-planting-system-deployment.md`

**更新内容**:
在 Story 3.1 (植物卡片 UI) AC 后新增章节 "植物卡片交互增强 (Story 10.8 实现)":
- 鼠标悬停提示 (Tooltip) 完整说明
- 提示框布局:状态提示(第一行) + 植物名(第二行)
- 鼠标光标状态切换逻辑
- 参考 Story 10.8

---

### 3.4 架构文档

**文件**: `docs/architecture/data-models.md`

**更新内容**:
新增 `TooltipComponent` 组件定义:
- **Purpose**: 植物卡片悬停提示组件
- **使用场景**: Story 10.8
- **字段定义**:
  - 文本内容: StatusText (第一行,可选), PlantName (第二行,必需)
  - 样式: BackgroundColor, BorderColor, Padding, TextSpacing
  - 位置和尺寸: Position, Width, Height
  - 关联: TargetEntity
- **相关系统**: InputSystem, UIRenderSystem

---

## 4. 推荐实施路径 (Recommended Path)

### ✅ 选择: 直接集成到 Story 10.8

**理由**:
1. **功能内聚**: Tooltip 与阳光闪烁都属于"植物卡片交互优化"
2. **工作量合理**: +3-4 小时,总计 7-10 小时,仍属中等规模
3. **用户体验完整**: 一次性实现完整的卡片 UX
4. **无技术风险**: 新增组件,不影响现有功能
5. **符合原版**: Tooltip 是原版标准特性

**替代方案及排除理由**:
- ❌ 拆分为 Story 10.9: 增加管理成本,功能割裂
- ❌ 移出 MVP: 用户明确要求 P0,不应妥协

---

## 5. MVP 影响评估 (MVP Impact)

### MVP 范围: ❌ 无变更

**分析**:
- Tooltip 是 UI 增强,不改变核心玩法
- 工作量增加 3-4 小时,Epic 10 总量可控
- MVP 目标"完整游戏体验"保持不变

### MVP 目标: ✅ 符合强化

Tooltip 系统增强了 PRD 核心原则:
- **状态可见性**: 实时显示卡片状态
- **即时反馈**: 悬停即显示信息
- **原版忠实度**: 100% 还原原版 UX

---

## 6. 高层级行动计划 (Action Plan)

### 立即行动 (Scrum Master - Bob)

1. ✅ **更新 Story 10.8**: 应用所有变更 (已完成)
2. ✅ **更新 Epic 10**: 调整描述和工作量 (已完成)
3. ✅ **补充 PRD**: Epic 3 中新增 Tooltip 说明 (已完成)
4. ✅ **更新架构文档**: 新增 TooltipComponent (已完成)
5. **获得用户批准**: 本 Proposal 需用户确认

### 后续开发 (Dev Agent)

**触发条件**: 用户批准 Proposal 后

**输入**: 更新后的 Story 10.8 文档

**任务**:
- Phase 1-4: 阳光闪烁功能 (4-6 小时)
- Phase 5: Tooltip 系统 (3-4 小时)
- 总计: 7-10 小时

**交付**:
- 完整功能代码
- 单元测试 (覆盖率 ≥ 70%)
- 集成测试通过

### QA 验证 (可选)

**触发条件**: Dev 完成实现后

**任务**:
- 验证所有 15 个 AC
- 视觉验证 (对比原版 PVZ)
- 性能测试
- 回归测试

---

## 7. 技术实现要点 (Technical Highlights)

### Tooltip 布局 (关键修正)

**正确布局** (从上到下):
```
┌─────────────────────┐
│  重新装填中...      │ ← 第一行: 状态提示(红色小号, 可选)
│  豌豆射手           │ ← 第二行: 植物名(黑色正常)
└─────────────────────┘
```

**关键点**:
- ✅ 状态提示在**第一行**(上方)
- ✅ 植物名在**第二行**(下方)
- ✅ 两行间距 3-5px
- ✅ 可用状态不显示第一行

### TooltipComponent 核心字段

```go
type TooltipComponent struct {
    IsVisible       bool
    StatusText      string      // 第一行,可选
    StatusTextColor color.Color // 红色
    PlantName       string      // 第二行,必需
    PlantNameColor  color.Color // 黑色
    TextSpacing     float64     // 行间距
    // ...
}
```

### 渲染逻辑 (伪代码)

```go
// 1. 如果有状态提示,先渲染第一行
if tooltip.StatusText != "" {
    drawText(tooltip.StatusText, redColor, smallFont, Y_START)
    Y_START += smallFontHeight + textSpacing
}

// 2. 渲染植物名第二行
drawText(tooltip.PlantName, blackColor, normalFont, Y_START)
```

---

## 8. 风险与缓解 (Risks & Mitigation)

| 风险 | 可能性 | 影响 | 缓解措施 |
|------|--------|------|----------|
| Tooltip 实现超时 | Low | Medium | MVP: 先实现基础功能,样式后续优化 |
| 与其他 UI 冲突 | Low | Medium | 确保 Tooltip 渲染在最上层 |
| 光标切换失效 | Low | Low | 测试多平台,提供降级方案 |
| Sprint 进度影响 | Medium | Low | 优先核心功能,Tooltip 可迭代 |

---

## 9. 成功标准 (Success Criteria)

本次变更成功的标准:

### 文档完整性
- ✅ Story 10.8 包含完整 Tooltip 定义 (15 个 AC, 5 个 Phase)
- ✅ Epic 10 工作量准确反映调整
- ✅ PRD 补充 Tooltip 描述
- ✅ 架构文档新增 TooltipComponent

### 功能完整性
- ✅ Story 10.8 实现阳光闪烁 + Tooltip + 光标切换
- ✅ 所有 15 个 AC 满足
- ✅ Tooltip 布局正确 (状态在上,植物名在下)
- ✅ 视觉还原原版 PVZ

### 项目进度
- ✅ Epic 10 按调整后时间线完成
- ✅ MVP 目标保持不变

### 代码质量
- ✅ 单元测试覆盖率 ≥ 70%
- ✅ 无现有功能回归

---

## 10. 变更检查清单总结 (Checklist Summary)

### Section 1: 触发器与上下文 ✅
- [x] 触发 Story: Story 10.8
- [x] 问题定义: 缺少 Tooltip 系统
- [x] 变更类型: 需求补充
- [x] 初步影响: 工作量 +3-4 小时

### Section 2: Epic 影响 ✅
- [x] 当前 Epic 可完成: 是
- [x] Epic 需修改: Story 10.8 扩展
- [x] 未来 Epic 影响: 无
- [x] Epic 总结: 工作量 58-84 小时

### Section 3: 文档冲突 ✅
- [x] PRD 冲突: 无,已补充
- [x] 架构文档冲突: 无,已新增组件
- [x] 其他文档: 无冲突
- [x] 文档更新清单: 4 个文件已更新

### Section 4: 前进路径 ✅
- [x] 选项 1 (直接集成): ✅ 推荐
- [x] 选项 2 (拆分 Story): ❌ 不推荐
- [x] 选项 3 (MVP 调整): ❌ 不适用
- [x] 最终选择: 选项 1

### Section 5: Proposal 生成 ✅
- [x] 变更分析总结
- [x] Epic 影响总结
- [x] 文档调整清单
- [x] 推荐路径
- [x] MVP 影响
- [x] 行动计划
- [x] Agent 交接计划

### Section 6: 最终审核 ⏳
- [ ] 用户审查 Proposal
- [ ] 用户批准变更
- [ ] 开始实施

---

## 11. 交接说明 (Handoff Instructions)

### 当前 Agent: Bob (Scrum Master) ✅

**已完成**:
- ✅ Sprint Change Proposal 完整起草
- ✅ 所有文档变更已应用
- ✅ Tooltip 布局已修正 (状态在上,植物名在下)

**待用户操作**:
- ⏳ 审查本 Proposal
- ⏳ 批准或要求调整

### 下一个 Agent: Dev Agent

**触发条件**: 用户批准后

**输入文档**:
- `docs/stories/10.8-sun-shortage-feedback.md` (完整 Story,包含 Tooltip)

**任务**:
- 实现 Phase 1-5 所有 Tasks
- 编写单元测试 (目标覆盖率 ≥ 70%)
- 通过集成测试

**交付物**:
- 完整功能代码
- 测试代码
- QA 验证报告 (如需要)

---

## 附录 A: 变更后的 Story 10.8 结构

```
Story 10.8: 植物卡片交互反馈增强
├── Metadata (✅ 已更新)
│   ├── 标题: 植物卡片交互反馈增强
│   ├── 优先级: ⭐⭐⭐⭐ 中高
│   └── 工作量: 7-10 小时
├── Story 描述 (✅ 已更新)
├── Background (原有 + Tooltip 背景)
├── Acceptance Criteria (15 个)
│   ├── AC 1-10: 阳光闪烁功能 (原有)
│   └── AC 11-15: Tooltip 系统 (✅ 新增)
├── Tasks (5 个 Phase)
│   ├── Phase 1-4: 阳光闪烁 (原有)
│   └── Phase 5: Tooltip 实现 (✅ 新增,5 个 Task)
├── Technical Implementation (✅ 已扩展)
│   ├── 闪烁系统 (原有)
│   └── Tooltip 系统 (✅ 新增)
├── Success Criteria (✅ 已更新,新增 4 项)
├── Definition of Done (✅ 已更新,包含 Tooltip)
└── Change Log (✅ 已记录 2025-11-25 变更)
```

---

## 附录 B: 文档变更清单

| 文件 | 状态 | 变更内容 |
|------|------|---------|
| `docs/stories/10.8-sun-shortage-feedback.md` | ✅ 已更新 | 新增 AC 11-15, Phase 5, 技术实现, DoD |
| `docs/prd/epic-10-game-experience-polish.md` | ✅ 已更新 | Story 10.8 描述, 工作量表格 |
| `docs/prd/epic-3-planting-system-deployment.md` | ✅ 已补充 | 新增 Tooltip 功能说明 |
| `docs/architecture/data-models.md` | ✅ 已新增 | TooltipComponent 组件定义 |
| `docs/qa/sprint-change-proposal-2025-11-25-story10.8-tooltip.md` | ✅ 本文档 | 完整变更提案 |

---

## 结论 (Conclusion)

本 Sprint Change Proposal 建议在 Story 10.8 中集成 Tooltip 系统,以实现完整的植物卡片交互用户体验。变更范围清晰,工作量可控,符合 MVP 目标,且无技术风险。

**请批准本提案以继续实施。**

---

**提案人**: Bob (Scrum Master)
**日期**: 2025-11-25
**版本**: 1.0
