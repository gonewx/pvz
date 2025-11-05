# Sprint Change Proposal
## buildMergedTracks 重复实现与逻辑错误修复

**日期**: 2025-11-05
**提案人**: Bob (Scrum Master)
**触发点**: 人工代码审查发现重复实现
**严重性**: 🔴 高 - 生产代码存在逻辑错误
**状态**: ✅ 已批准

---

## 1. 问题总结 (Issue Summary)

### 1.1 核心问题

`buildMergedTracks` 函数在项目中存在**3处重复实现**（共210行重复代码），且**生产代码中有功能错误**：

| 实现位置 | f 值继承逻辑 | 默认值 | 状态 |
|---------|------------|--------|------|
| `internal/reanim/parser.go:120` | ✅ 累积继承 | ✅ `accF := 0` | ✅ **正确** |
| `pkg/systems/reanim_system.go:380` | ❌ hasFrameNum 检测 | ❌ nil | ❌ **错误** |
| `cmd/render_animation_comparison/main.go:452` | ✅ 累积继承 | ✅ `accF := 0` | ✅ 正确但重复 |

### 1.2 f 值语义澄清

经过分析，**f 值是可见性标志**，不是视觉属性：
- `f=0` → **显示**该帧
- `f=-1` → **隐藏**该帧
- 未设置 → **继承上一帧的可见性**（需要累积继承）
- 第一帧默认 `f=0`（默认显示）

### 1.3 错误逻辑分析

**错误实现**（`reanim_system.go:402-472`）：
```go
// ❌ Story 6.6/6.7 引入的错误逻辑
hasFrameNum := false
for _, frame := range track.Frames {
    if frame.FrameNum != nil {
        hasFrameNum = true
        break
    }
}
// ...
var frameNumPtr *int
if hasFrameNum {  // ❌ 纯视觉轨道设为 nil，应该是 f=0
    f := accF
    frameNumPtr = &f
}
```

**问题**：纯视觉轨道（如 `leaf1`）没有任何 f 值时，`FrameNum` 被设为 `nil`，而不是默认的 `0`（显示）。

### 1.4 引入时间线

- ✅ **Story 6.5**: 首次实现，在 `parser.go` 中是正确的
- ❌ **Story 6.6 & 6.7** (commit f723108): 引入错误的 hasFrameNum 检测
- ⚠️ **代码注释标记为 "Story 12.1 修复"** - 这是错误标注

### 1.5 使用情况

**正确版本被使用**：
- `pkg/entities/selector_screen_factory.go:140` ✅
- 3个测试工具 ✅

**错误版本被使用**：
- `pkg/systems/reanim_system.go` - **所有动画播放** ❌

---

## 2. Epic 影响总结 (Epic Impact)

### 2.1 当前 Epic

**Epic 6: 动画系统迁移** - ✅ 已完成

| Story | 状态 | 影响 |
|-------|------|------|
| 6.1-6.5 | Done | ✅ 实现正确 |
| 6.6 & 6.7 | Done | ❌ 引入错误逻辑 |

**结论**：
- ✅ Epic 不需要重新开发
- ⚠️ 需要**热修复** (Hotfix)

### 2.2 未来 Epic

所有后续 Epic (7, 8, 10, 11, 12) 都依赖 Reanim 系统，但**修复对它们透明**，无需调整计划。

---

## 3. 文档调整需求 (Artifact Adjustments)

| 文档 | 影响 | 需要更新 | 优先级 |
|------|------|---------|--------|
| PRD | ✅ 无冲突 | ❌ 否 | - |
| 架构文档 | ✅ 无冲突 | ❌ 否 | - |
| **CLAUDE.md** | ⚠️ 过时说明 | ✅ **是** | 🔴 高 |
| Story 文档 | ✅ 历史记录 | ❌ 否 | - |
| 技术指南 | ⚠️ 可能有错误示例 | ✅ 是 | 🟡 中 |

**CLAUDE.md 需要更新的内容**：
- Line 280-281: f 值说明有误导
- Line 290-318: 双动画叠加机制已废弃（Story 6.6/6.7）
- 需要更新为新的播放模式通用化机制

---

## 4. 推荐路径 (Recommended Path Forward)

### ✅ **选项 1: 直接调整/集成**（已选择）

**方案**：
1. 保留 `internal/reanim/parser.go` 的 `BuildMergedTracks`（正确版本）
2. 删除 `pkg/systems/reanim_system.go` 的 `buildMergedTracks`
3. 删除 `cmd/render_animation_comparison/main.go` 的 `buildMergedTracks`
4. 更新 `ReanimSystem` 改用 `reanim.BuildMergedTracks(comp.Reanim)`

**工作量**: 1天（代码0.5天 + 文档0.25天 + 测试0.25天）
**风险**: 🟢 低
**收益**: 消除210行重复代码 + 修复逻辑错误

---

## 5. PRD MVP 影响 (MVP Impact)

✅ **无影响** - 这是技术实现细节问题，不影响功能范围或 MVP 目标。

---

## 6. 高层行动计划 (High-Level Action Plan)

### Phase 1: 代码重构（0.5天）

1. **删除重复实现**
   - 删除 `pkg/systems/reanim_system.go:380-490` (buildMergedTracks 方法)
   - 删除 `cmd/render_animation_comparison/main.go:452-521`

2. **更新调用点**（约 8 处）
   - `pkg/systems/reanim_system.go:604` - PlayAnimation
   - `pkg/systems/reanim_system.go:812` - SetAnimation
   - `pkg/systems/reanim_system.go:1343` - buildMergedTracksForPreview
   - `pkg/systems/reanim_system_test.go:146, 193` - 测试代码

   **修改方式**：
   ```go
   // ❌ 旧代码
   reanimComp.MergedTracks = s.buildMergedTracks(reanimComp)

   // ✅ 新代码
   reanimComp.MergedTracks = reanim.BuildMergedTracks(reanimComp.Reanim)
   ```

3. **删除错误的注释**
   - 删除所有 "Story 12.1 修复" 注释（错误标注）
   - 删除 hasFrameNum 相关注释

### Phase 2: 文档更新（0.25天）

1. **更新 CLAUDE.md**
   - 删除 Line 290-318（双动画叠加机制说明）
   - 修正 Line 280-281（f 值语义说明）
   - 添加新的播放模式通用化说明

2. **检查技术指南**
   - `docs/reanim/reanim-format-guide.md` - 验证示例代码
   - `docs/reanim/reanim-fix-guide.md` - 更新修复指南

### Phase 3: 测试验证（0.25天）

1. **单元测试**
   - 运行 `go test ./pkg/systems/reanim_system_test.go`
   - 运行 `go test ./internal/reanim/...`

2. **集成测试**
   - 测试植物动画（豌豆射手、向日葵、坚果墙）
   - 测试僵尸动画
   - 测试 SelectorScreen 动画

3. **视觉验证**
   - 验证纯视觉轨道默认显示（如 leaf1）
   - 验证 f=-1 帧正确隐藏
   - 验证 f=0 帧正确显示

---

## 7. 具体代码修改提案 (Proposed Code Changes)

### 7.1 删除 `pkg/systems/reanim_system.go` 中的 buildMergedTracks

**文件**: `pkg/systems/reanim_system.go`
**行号**: 335-490 (共155行)

```diff
- // buildMergedTracks builds accumulated frame arrays for each track by applying frame inheritance.
- //
- // Story 6.5: Frame Inheritance Mechanism (帧继承机制)
- // ...（删除整个函数，共155行）
- func (s *ReanimSystem) buildMergedTracks(comp *components.ReanimComponent) map[string][]reanim.Frame {
-     // ... 实现代码 ...
- }
```

### 7.2 更新 PlayAnimation 方法

**文件**: `pkg/systems/reanim_system.go`
**行号**: ~604

```diff
 func (s *ReanimSystem) PlayAnimation(entityID uint64, animName string) error {
     // ...
-    reanimComp.MergedTracks = s.buildMergedTracks(reanimComp)
+    reanimComp.MergedTracks = reanim.BuildMergedTracks(reanimComp.Reanim)

     return nil
 }
```

### 7.3 更新 SetAnimation 方法

**文件**: `pkg/systems/reanim_system.go`
**行号**: ~812

```diff
 func (s *ReanimSystem) SetAnimation(entityID uint64, animName string) error {
     // ...
-    reanimComp.MergedTracks = s.buildMergedTracks(reanimComp)
+    reanimComp.MergedTracks = reanim.BuildMergedTracks(reanimComp.Reanim)

     return nil
 }
```

### 7.4 更新 buildMergedTracksForPreview 方法

**文件**: `pkg/systems/reanim_system.go`
**行号**: ~1490-1493

```diff
 func (s *ReanimSystem) buildMergedTracksForPreview(reanimComp *components.ReanimComponent) map[string][]reanim.Frame {
-    // Reuse the existing buildMergedTracks logic, which already processes ALL tracks
-    // This is correct because buildMergedTracks doesn't filter tracks.
-    return s.buildMergedTracks(reanimComp)
+    // Use the centralized BuildMergedTracks from parser package
+    return reanim.BuildMergedTracks(reanimComp.Reanim)
 }
```

### 7.5 更新测试代码

**文件**: `pkg/systems/reanim_system_test.go`
**行号**: 146, 193

```diff
-tc.reanimComp.MergedTracks = rs.buildMergedTracks(tc.reanimComp)
+tc.reanimComp.MergedTracks = reanim.BuildMergedTracks(tc.reanimComp.Reanim)
```

```diff
-reanimComp.MergedTracks = rs.buildMergedTracks(reanimComp)
+reanimComp.MergedTracks = reanim.BuildMergedTracks(reanimComp.Reanim)
```

### 7.6 删除 cmd/render_animation_comparison 中的重复实现

**文件**: `cmd/render_animation_comparison/main.go`
**行号**: 452-521 (共70行)

```diff
- func buildMergedTracks(reanimXML *reanim.ReanimXML, standardFrameCount int) map[string][]reanim.Frame {
-     // ... 实现代码 ...
- }
```

**更新调用**（Line 90）：
```diff
- mergedTracks := buildMergedTracks(reanimXML, standardFrameCount)
+ mergedTracks := reanim.BuildMergedTracks(reanimXML)
```

### 7.7 删除 hasFrameNumValues 方法（已废弃）

**文件**: `pkg/systems/reanim_system.go`
**行号**: ~271-277

```diff
- // hasFrameNumValues checks if a track has any FrameNum values.
- // Used to distinguish hybrid tracks (with f values) from pure visual tracks (without f values).
- func (s *ReanimSystem) hasFrameNumValues(track *reanim.Track) bool {
-     // ... 实现代码 ...
- }
```

---

## 8. Agent 交接计划 (Agent Handoff Plan)

| 角色 | 职责 | 交接内容 |
|------|------|---------|
| **Dev Agent** | 执行代码重构 | 本提案 Section 7（具体代码修改） |
| **Dev Agent** | 运行测试验证 | 本提案 Phase 3（测试验证） |
| **Dev Agent** | 更新文档 | 本提案 Phase 2（文档更新） |

**优先级**：🔴 高 - 建议立即开始

---

## 9. 验收标准 (Acceptance Criteria)

- [ ] ✅ 所有重复的 `buildMergedTracks` 实现已删除
- [ ] ✅ `ReanimSystem` 使用 `reanim.BuildMergedTracks`
- [ ] ✅ 所有单元测试通过
- [ ] ✅ 植物和僵尸动画显示正常
- [ ] ✅ SelectorScreen 动画显示正常
- [ ] ✅ CLAUDE.md 已更新（删除过时说明）
- [ ] ✅ 代码编译无错误和警告
- [ ] ✅ 游戏运行稳定在 60 FPS

---

## 10. 回滚计划 (Rollback Plan)

**如果修复导致问题**：
- Git revert 单个提交（< 1分钟）
- 已有测试覆盖，低风险

---

## 11. 批准记录

**批准人**: 用户
**批准时间**: 2025-11-05
**批准状态**: ✅ 已批准
**下一步**: 交接给 Dev Agent 执行

---

**提案完成时间**: 预计 1 工作日
**建议开始时间**: 立即
