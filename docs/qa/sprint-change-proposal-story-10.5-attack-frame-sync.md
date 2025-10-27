# Sprint Change Proposal: Story 10.5 - 植物攻击动画帧事件同步

**提案日期：** 2025-10-27
**提案人：** Bob (Scrum Master)
**审查人：** [待填写]
**批准状态：** ✅ **Approved** (用户已确认采用方案 B)

---

## 📋 执行摘要

**问题：** 植物攻击动画与子弹发射时机不同步。子弹在动画开始时立即创建，而非在"身体猛地前倾"的关键帧创建。

**解决方案：** 采用**方案 B（配置关键帧）**，使用 `config.PeashooterShootingFireFrame` 精确匹配关键帧，实现零延迟发射。

**影响范围：** Epic 10 (游戏体验完善) → 创建新 Story 10.5

**工作量：** 预计 3-4 小时

**批准状态：** ✅ 用户已批准方案 B

---

## 1. 问题概述 (Issue Summary)

### 1.1 触发问题

**用户报告：**
> "植物在攻击状态时，子弹的发射时机不正确，要在攻击动画播放到身体向后微微压缩蓄力，然后猛地前倾时，一颗绿色的豌豆被'噗'地一声射出。"

**当前行为：**
```go
// behavior_system.go:494-505
if hasZombieInLine {
    s.reanimSystem.PlayAnimationNoLoop(entityID, "anim_shooting")  // 播放攻击动画

    // ❌ 立即创建子弹（与动画开始同时）
    bulletStartX := peashooterPos.X + config.PeaBulletOffsetX
    bulletStartY := peashooterPos.Y + config.PeaBulletOffsetY
    bulletID, _ := entities.NewPeaProjectile(s.entityManager, s.resourceManager, bulletStartX, bulletStartY)
}
```

**预期行为（来自白皮书）：**
> **豌豆射手攻击动画** (`.meta/whitepaper.md:334-340`)：
> 1. 嘴巴迅速向前嘟起
> 2. **身体向后微微压缩蓄力，然后猛地前倾** ← 在此刻发射子弹
> 3. 一颗绿色的豌豆被"噗"地一声射出
> 4. 发射后身体回弹

**核心问题：**
- ❌ 子弹在动画开始时立即创建
- ❌ 使用固定偏移量，未考虑动画中头部位置的动态变化
- ❌ `PlantComponent.PendingProjectile` 和 `LastMouthX` 字段未被使用（技术债务）

---

### 1.2 影响范围

**用户体验影响：**
- 🟡 **中等严重性** - 视觉表现与游戏逻辑不同步
- 缺乏原版游戏的打击感
- 影响"忠实复刻"目标（PRD 核心目标）

**技术影响：**
- 影响所有射手类植物（豌豆射手、寒冰射手、双发射手等）
- 代码注释已标记为"未来优化"，但未实施

---

## 2. Epic 影响分析 (Epic Impact Summary)

### 2.1 当前 Epic 状态

**Epic 10: 游戏体验完善 (Game Experience Polish)**
- ✅ Story 10.1-10.3: 已完成
- ✅ Story 10.4: 植物种植粒子特效 - Done
- ⏳ **Story 10.5: 植物攻击动画帧事件同步** - **新增（本提案）**

**影响评估：**
- ✅ **Epic 10 可以继续** - 通过创建 Story 10.5 完善攻击动画系统
- 不涉及 Epic 目标或范围的重大变更

### 2.2 未来 Epic 影响

- Epic 11+（未来关卡） - 🟢 **无影响**
- Epic 9（ECS 泛型重构） - 🟢 **无影响**（已完成）
- Epic 6（动画系统） - 🟢 **无影响**（Reanim 系统稳定）

**结论：** 修复对未来 Epic 透明，无连锁影响。

---

## 3. 文档调整需求 (Artifact Adjustment Needs)

### 3.1 需要更新的文档

| 文档 | 章节 | 变更类型 | 变更内容 |
|------|------|---------|---------|
| `docs/stories/10.5.story.md` | 新建 Story | 创建文件 | ✅ 已创建 |
| `docs/stories/10.3.story.md` | Completion Notes | 新增说明 | 添加"子弹发射时机需在 Story 10.5 进一步优化" |
| `docs/prd/epic-10-game-experience-polish.md` | Story List | 新增条目 | 添加 Story 10.5 到 Epic 10 |
| `pkg/systems/behavior_system.go` | 代码注释 | 更新注释 | 移除"未来优化"注释，替换为实际实现 |
| `pkg/config/plant_config.go` | 新建文件 | 创建文件 | 定义 `PeashooterShootingFireFrame = 5` |

---

## 4. 推荐解决方案 (Recommended Path Forward)

### 4.1 方案对比

#### ❌ 方案 A：峰值检测（已否决）

**思路：** 监听 `idle_mouth` 轨道的 X 坐标峰值（从增大变为减小）。

**问题：**
- ⚠️ **存在 1 帧延迟** - 在峰值的下一帧才检测到
- ⚠️ 需要额外字段（`SecondLastMouthX`）
- ⚠️ 增速放缓的阈值需要调优

**用户关切：**
> "检测峰值后，已经是下一帧了，会不会造成有延迟发射的感觉？"

**结论：** ❌ 用户关切合理，方案 A 被否决。

---

#### ✅ 方案 B：配置关键帧（已批准）

**思路：** 在配置中指定发射帧号（`PeashooterShootingFireFrame = 5`），直接帧号匹配。

**代码示例：**
```go
// 精确匹配关键帧（零延迟）
if reanim.CurrentFrame == config.PeashooterShootingFireFrame {
    createBullet()  // 在关键帧精确发射
}
```

**优点：**
- ✅ **零延迟** - 精确在关键帧发射
- ✅ **逻辑简单** - 整数比较，易于调试
- ✅ **性能最优** - O(1) 复杂度
- ✅ **符合原版** - 原版游戏也是基于帧号触发事件
- ✅ **易于调优** - 修改常量即可，无需改动代码

**用户批准：**
> "使用方案 B"

**结论：** ✅ 采用方案 B。

---

### 4.2 关键帧号确定方法

**初始值推算：**

根据白皮书和 Reanim 系统默认 FPS (12)：
- 攻击动画时长：约 0.5-0.7 秒
- 总帧数：12 fps × 0.6s = **7-8 帧**
- 动画阶段划分：
  - **Frame 0-2**: 嘴巴向前嘟起（准备）
  - **Frame 3-4**: 身体向后压缩（蓄力）
  - **Frame 5**: 身体猛地前倾（峰值）← **发射帧**
  - **Frame 6-7**: 身体回弹

**初始配置值：** `PeashooterShootingFireFrame = 5`

**手动调优方法（Task 5）：**
1. 运行游戏 `go run . --verbose`
2. 观察日志输出的 `CurrentFrame` 值
3. 调整配置常量（+/- 1 帧）
4. 反复测试直到视觉完美同步

---

### 4.3 实施计划 (Story 10.5)

**AC (Acceptance Criteria):**
1. 子弹在关键帧（Frame 5）创建，而非动画开始时
2. 使用配置的关键帧号，零延迟发射
3. 子弹起始位置使用 `idle_mouth` 轨道的实时坐标
4. 视觉同步符合原版游戏表现
5. 关键帧号可配置，支持未来扩展
6. 性能无明显下降（O(1) 整数比较）
7. 代码清晰，易于调试
8. 激活 `PendingProjectile` 字段，消除技术债务

**Tasks:**
1. **添加配置常量** (0.5 小时)
   - 创建 `pkg/config/plant_config.go`
   - 定义 `PeashooterShootingFireFrame = 5`

2. **修改 handlePeashooterBehavior()** (0.5 小时)
   - 删除立即创建子弹的代码
   - 设置 `plant.PendingProjectile = true`

3. **修改 updatePlantAttackAnimation()** (1.5 小时)
   - 添加关键帧检测逻辑
   - 查询 `idle_mouth` 轨道实时坐标
   - 在关键帧创建子弹

4. **扩展 ReanimSystem API** (0.5 小时)
   - 添加 `GetTrackTransform(entityID, trackName)` 方法
   - 返回轨道当前帧的局部坐标

5. **测试和调优** (1 小时)
   - 单元测试
   - 手动调优关键帧号
   - 验证视觉同步

**预计工作量：** 4 小时

---

## 5. 具体代码变更 (Proposed Code Edits)

### 变更 1: 创建配置常量

**文件：** `pkg/config/plant_config.go`（新建）

```go
package config

// 植物攻击动画关键帧配置
// Story 10.5: 定义射手类植物的子弹发射关键帧号

const (
	// PeashooterShootingFireFrame 豌豆射手攻击动画的子弹发射帧号
	//
	// 基于原版游戏白皮书分析（12 fps，攻击动画 0.6秒）：
	//   - Frame 0-2: 嘴巴向前嘟起（准备）
	//   - Frame 3-4: 身体向后压缩（蓄力）
	//   - Frame 5: 身体猛地前倾（峰值）← 发射子弹
	//   - Frame 6-7: 身体回弹，嘴巴恢复
	//
	// 注意：
	//   - 帧号从 0 开始计数
	//   - 如视觉不同步，可手动调整此值
	//   - 调整步长：+/- 1 帧，通过 --verbose 日志观察
	PeashooterShootingFireFrame = 5

	// 未来扩展：其他射手植物
	// SnowPeaShootingFireFrame    = 5  // 寒冰射手
	// RepeaterShootingFireFrame1  = 5  // 双发射手（第一发）
	// RepeaterShootingFireFrame2  = 8  // 双发射手（第二发）
)
```

---

### 变更 2: 修改 handlePeashooterBehavior()

**文件：** `pkg/systems/behavior_system.go`

**删除的代码（第 507-526 行）：**
```go
// ❌ 删除：立即创建子弹的逻辑
bulletStartX := peashooterPos.X + config.PeaBulletOffsetX
bulletStartY := peashooterPos.Y + config.PeaBulletOffsetY
s.playShootSound()
bulletID, err := entities.NewPeaProjectile(...)
timer.CurrentTime = 0
```

**新增的代码（替换上述逻辑）：**
```go
// Story 10.5: 设置"等待发射"状态，不立即创建子弹
plant.PendingProjectile = true
log.Printf("[BehaviorSystem] 豌豆射手 %d 进入攻击状态，等待关键帧(%d)发射子弹",
    entityID, config.PeashooterShootingFireFrame)

// 重置计时器
timer.CurrentTime = 0
```

---

### 变更 3: 修改 updatePlantAttackAnimation()

**文件：** `pkg/systems/behavior_system.go`

**在现有逻辑后添加（第 1490 行之后）：**
```go
// Story 10.5: 关键帧事件监听 - 子弹发射时机同步
if plant.PendingProjectile {
    // 查询当前帧号
    reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
    if !ok {
        return
    }

    // 精确匹配发射帧（零延迟）
    if reanim.CurrentFrame == config.PeashooterShootingFireFrame {
        log.Printf("[BehaviorSystem] 豌豆射手 %d 到达关键帧(%d)，发射子弹！",
            entityID, reanim.CurrentFrame)

        // 获取 idle_mouth 的实时坐标（局部坐标）
        mouthX, mouthY, err := s.reanimSystem.GetTrackTransform(entityID, "idle_mouth")
        if err != nil {
            log.Printf("[BehaviorSystem] 查询 idle_mouth 轨道失败: %v，使用固定偏移", err)
            // 降级：使用固定偏移
            pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
            mouthX = config.PeaBulletOffsetX
            mouthY = config.PeaBulletOffsetY
        }

        // 获取植物世界坐标
        pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
        if !ok {
            return
        }

        // 转换局部坐标 → 世界坐标
        bulletStartX := pos.X + mouthX
        bulletStartY := pos.Y + mouthY

        log.Printf("[BehaviorSystem] 发射子弹，位置: (%.1f, %.1f)（实时轨道坐标）",
            bulletStartX, bulletStartY)

        // 播放发射音效
        s.playShootSound()

        // 创建豌豆子弹实体
        bulletID, err := entities.NewPeaProjectile(s.entityManager, s.resourceManager, bulletStartX, bulletStartY)
        if err != nil {
            log.Printf("[BehaviorSystem] 创建豌豆子弹失败: %v", err)
        } else {
            log.Printf("[BehaviorSystem] 豌豆射手 %d 发射子弹 %d（零延迟帧同步）", entityID, bulletID)
        }

        // 清除"等待发射"状态
        plant.PendingProjectile = false
    }
}
```

---

### 变更 4: 扩展 ReanimSystem API

**文件：** `pkg/systems/reanim_system.go`

**新增方法：**
```go
// GetTrackTransform 获取指定轨道的当前变换矩阵（局部坐标）
//
// Story 10.5: 用于动画帧事件监听，获取部件实时位置
//
// 参数：
//   - entityID: 实体 ID
//   - trackName: 轨道名称（如 "idle_mouth", "anim_stem"）
//
// 返回：
//   - x, y: 轨道当前帧的局部坐标（相对于实体中心）
//   - error: 如果实体无动画组件或轨道不存在
func (rs *ReanimSystem) GetTrackTransform(entityID ecs.EntityID, trackName string) (x, y float64, err error) {
    // 获取 Reanim 组件
    reanim, ok := ecs.GetComponent[*components.ReanimComponent](rs.entityManager, entityID)
    if !ok {
        return 0, 0, fmt.Errorf("entity %d does not have ReanimComponent", entityID)
    }

    // 查找当前播放的动画定义
    animDef, ok := reanim.ReanimDefCache[reanim.CurrentAnimName]
    if !ok {
        return 0, 0, fmt.Errorf("animation '%s' not found in cache", reanim.CurrentAnimName)
    }

    // 查找指定轨道
    for _, track := range animDef.Tracks {
        if track.Name == trackName {
            // 获取当前帧的变换
            currentFrame := reanim.CurrentFrame
            if currentFrame < 0 || currentFrame >= len(track.Transforms) {
                currentFrame = len(track.Transforms) - 1
                if currentFrame < 0 {
                    return 0, 0, fmt.Errorf("track '%s' has no transforms", trackName)
                }
            }

            transform := track.Transforms[currentFrame]
            return transform.X, transform.Y, nil
        }
    }

    return 0, 0, fmt.Errorf("track '%s' not found in animation '%s'", trackName, reanim.CurrentAnimName)
}
```

---

## 6. PRD MVP 影响 (PRD MVP Impact)

**MVP 范围影响：** 🟢 **无影响**

- 攻击动画时机属于游戏体验细节，不影响 MVP 核心功能
- 游戏仍可正常游玩
- 修复是质量提升，而非功能增减

**MVP 范围：** 保持不变（Epic 1-5 核心功能）

---

## 7. 下一步行动计划 (High-Level Action Plan)

### 7.1 文档更新

- [x] 创建 Story 10.5 文档（`docs/stories/10.5.story.md`）✅
- [ ] 更新 Story 10.3 Completion Notes
- [ ] 更新 Epic 10 Story 列表

### 7.2 开发任务（交接给 Dev Agent）

- [ ] 创建 `pkg/config/plant_config.go`
- [ ] 修改 `BehaviorSystem.handlePeashooterBehavior()`
- [ ] 修改 `BehaviorSystem.updatePlantAttackAnimation()`
- [ ] 扩展 `ReanimSystem.GetTrackTransform()` API
- [ ] 添加单元测试
- [ ] 手动调优关键帧号
- [ ] 验证视觉同步

---

## 8. Agent Handoff Plan (代理交接计划)

**交接到：** Dev Agent (开发代理)

**交接内容：**
1. ✅ 本 Sprint Change Proposal 文档
2. ✅ Story 10.5 完整文档（`docs/stories/10.5.story.md`）
3. ✅ 详细代码变更建议（上述第 5 节）
4. ✅ 关键帧号初始值（Frame 5）和调优方法

**后续流程：**
1. **Dev Agent** → 实施 Story 10.5
2. **手动调优** → 验证 Frame 5 是否正确，必要时调整
3. **QA Agent** → 验证实现，运行测试
4. **Story Owner** → 标记 Story 10.5 为 Done

---

## 9. 风险评估 (Risks)

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|---------|
| 关键帧号不准确（Frame 5 偏差） | 中 | 低 | 手动调优，观察 --verbose 日志 |
| ReanimSystem API 性能问题 | 低 | 低 | O(n) 轨道遍历，n 通常 10-20 |
| 不同植物动画差异大 | 低 | 低 | 使用配置常量，每个植物独立配置 |

---

## 10. 成功标准 (Success Criteria)

**验收标准：**
1. ✅ 子弹在攻击动画的 Frame 5（或调优后的帧）创建
2. ✅ 视觉表现与原版游戏一致
3. ✅ 零延迟（关键帧到达时立即发射）
4. ✅ 性能无下降（FPS 稳定在 60）
5. ✅ 所有自动化测试通过
6. ✅ `PendingProjectile` 字段被正确使用
7. ✅ 代码清晰，技术债务清零

---

## 11. 批准签名 (Approval Signatures)

**提案人：** Bob (Scrum Master) - 2025-10-27
**用户批准：** ✅ 已确认采用方案 B - 2025-10-27
**状态：** ✅ **Approved - Ready for Implementation**

---

## 附录 A：方案 B 技术优势详解

### A.1 零延迟原理

**方案 A（峰值检测）的延迟问题：**
```
Frame N:   mouthX = 45.0 (峰值)
Frame N+1: mouthX = 44.5 (回落) ← 在此帧检测到峰值，创建子弹
延迟：1 帧 = 16.67ms (60fps) 或 8.33ms (120fps)
```

**方案 B（关键帧）无延迟：**
```
Frame 5:   currentFrame == 5 → 立即创建子弹
延迟：0 帧 = 0ms
```

### A.2 性能对比

| 指标 | 方案 A（峰值检测） | 方案 B（关键帧） |
|------|------------------|----------------|
| 延迟 | 1 帧 (16.67ms @ 60fps) | 0 帧 |
| CPU 开销 | O(1) 比较 + 浮点运算 | O(1) 整数比较 |
| 内存开销 | +16 字节（`SecondLastMouthX`） | 0 |
| 代码复杂度 | 高（峰值检测算法） | 低（整数匹配） |
| 调试难度 | 中（需调整阈值） | 低（直接调整帧号） |

**结论：** 方案 B 在所有指标上均优于方案 A。

---

## 附录 B：关键帧号调优指南

### B.1 启用详细日志

```bash
go run . --verbose > /tmp/game.log 2>&1
```

### B.2 观察关键帧号

在日志中查找：
```
[BehaviorSystem] 豌豆射手 X 到达关键帧(5)，发射子弹！
[ReanimSystem] Entity X CurrentFrame: 5, AnimName: anim_shooting
```

### B.3 调整步骤

1. **视觉观察** - 子弹是否在头部前倾时发射？
2. **太早** → `PeashooterShootingFireFrame = 6` (+1)
3. **太晚** → `PeashooterShootingFireFrame = 4` (-1)
4. **重新测试** → 反复调整直到完美同步

### B.4 经验值参考

- **12 fps 动画** → 通常为总帧数的 60-80%
- **豌豆射手** → 7-8 帧总长，建议 Frame 5-6
- **双发射手** → 可能需要两个关键帧（Frame 5, Frame 8）

---

**文档结束** - 准备交接给 Dev Agent 实施 Story 10.5 🚀
