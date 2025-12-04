# Epic 13: Reanim 动画系统现代化重构 (Reanim Animation System Modernization)

## Epic Goal (史诗目标)

基于 `cmd/animation_showcase` 中对 Reanim 格式的深入理解和正确实现，重构现有 Reanim 动画系统，引入轨道绑定机制（Track Animation Binding）、父子偏移系统（Parent-Child Offset）和渲染缓存优化，解决多动画组合播放的技术债务，提升系统性能和可维护性。

---

## Epic Description (史诗描述)

### Existing System Context (现有系统上下文)

**当前 Reanim 系统状态:**
- ✅ 基础 Reanim 解析和播放 (`internal/reanim`, Story 6.x)
- ✅ 帧继承机制 (`BuildMergedTracks`, Story 6.5)
- ✅ 多动画 API (`PlayAnimations`, Story 6.9)
- ✅ 轨道可见性控制 (`VisibleTracks` 白名单)
- ✅ 同步/异步播放模式 (`GlobalFrame` vs `TrackMapping`)
- ⚠️ **架构复杂度过高**：同步/异步模式切换、多种帧索引字段
- ⚠️ **多动画组合存在缺陷**：假设所有轨道共享同一物理帧，无法实现"头部用动画A，身体用动画B"
- ⚠️ **缺少父子偏移系统**：头部无法正确跟随身体摆动（如豌豆射手攻击时）
- ⚠️ **性能优化不足**：每帧重复计算渲染数据，无缓存机制

**技术栈:**
- Go 语言 + Ebitengine 引擎
- ECS 架构 (Entity-Component-System)
- Reanim XML 格式解析 (`internal/reanim`)
- 核心文件：
  - `pkg/systems/reanim_system.go` (1857 行，需重构)
  - `pkg/components/reanim_component.go` (245 行，需简化)
  - `pkg/systems/render_system.go` (渲染逻辑需优化)

**参考实现:**
- ✅ `cmd/animation_showcase/animation_cell.go` - 正确的 Reanim 实现
  - 轨道绑定分析算法 (`analyzeTrackBinding`)
  - 父子偏移计算 (`getParentOffset`)
  - 渲染缓存机制 (`cachedRenderData`)
  - 自动轨道绑定评分系统（基于位置方差）

---

### Problem Description (问题描述)

#### 当前架构的核心问题

**问题 1: 多动画组合的错误假设** ⭐**Critical**

当前 `PlayAnimations(["anim_shooting", "anim_head_idle"])` 的实现：
```go
// ❌ 错误假设：所有轨道共享同一个 GlobalFrame
for _, trackName := range visualTracks {
    physicalFrame := mapLogicalToPhysical(GlobalFrame, animVisibles)
    // 所有轨道都使用相同的 physicalFrame！
}
```

**实际需求**（animation_showcase 的正确理解）：
```go
// ✅ 正确实现：每个轨道独立选择动画和帧
trackBindings := map[string]string{
    "anim_face": "anim_head_idle",  // 头部用动画A
    "stalk_bottom": "anim_shooting", // 身体用动画B
}
// 不同轨道可以在不同动画的不同物理帧
```

**影响范围**：
- 豌豆射手攻击动画：头部应该保持摇晃，身体播放射击
- 僵尸动画：头部、身体、手臂可能需要独立控制
- 所有使用 `PlayAnimations` 的场景（虽然代码中标记了"简化系统不再使用"，但架构仍保留）

---

**问题 2: 缺少父子偏移系统** ⭐**High Priority**

当前系统没有实现父子关系的偏移计算，导致：
```
豌豆射手攻击时：
- 身体（anim_stem）摆动：X 从 37.6 移动到 40.0
- 头部（anim_face）应该跟随：需要叠加偏移 +2.4
- ❌ 实际效果：头部位置固定，看起来僵硬
```

**animation_showcase 的正确实现**：
```go
// 计算父轨道偏移量
offsetX, offsetY := getParentOffset(parentTrackName)
// 当子轨道和父轨道使用不同动画时，应用偏移
if childAnimName != parentAnimName {
    x = partX + offsetX
    y = partY + offsetY
}
```

**影响范围**：
- 所有植物的头部动画（豌豆射手、向日葵等）
- 僵尸的多部件动画
- 任何需要部件跟随其他部件运动的场景

---

**问题 3: 架构复杂度过高** ⭐**Technical Debt**

当前系统有多个冗余字段和模式：
```go
type ReanimComponent struct {
    // 帧索引字段过多（冗余）
    GlobalFrame int       // 同步模式使用
    CurrentFrame int      // 向后兼容字段

    // 多动画系统过于复杂
    Anims map[string]*AnimState  // 支持同步/异步双模式
    TrackMapping map[string]string // 仅异步模式使用

    // 字段职责不清晰
    AnimVisiblesMap map[string][]int // 用于什么？
    MergedTracks map[string][]Frame  // 用于什么？
}
```

**改进方向**：
- 统一帧管理：每个动画有独立的逻辑帧索引
- 清晰的轨道绑定：`trackBindings` 明确定义哪个轨道由哪个动画控制
- 移除冗余模式：只保留必要的灵活性

---

**问题 4: 性能优化不足** ⭐**Performance**

当前渲染逻辑每帧重复计算：
```go
func (rs *RenderSystem) renderReanimEntity() {
    for _, track := range tracks {
        // ❌ 每帧重新计算
        physicalFrame := findPhysicalFrame(...)
        frame := mergedTracks[trackName][physicalFrame]
        img := partImages[frame.ImagePath]
        // 重复的父子偏移计算
        // 重复的可见性检查
    }
}
```

**animation_showcase 的优化**：
```go
// ✅ 使用缓存
if currentFrame != lastRenderFrame {
    updateRenderCache() // 只在帧变化时更新
}
// 快速渲染缓存的数据
for _, data := range cachedRenderData {
    drawPart(data.img, data.frame, data.offsetX, data.offsetY)
}
```

**性能影响**：
- 100+ 实体同时渲染时 CPU 占用高
- 复杂动画（多轨道、多部件）性能差

---

### Enhancement Modules (功能模块详情)

#### 1. 轨道绑定机制 (Track Animation Binding System)

**核心功能：**
- 每个轨道可以绑定到不同的动画
- 支持自动轨道绑定分析（基于位置方差算法）
- 支持手动配置轨道绑定
- 支持运行时动态修改绑定

**实现要点：**
```go
type ReanimComponent struct {
    // 新增字段
    TrackBindings map[string]string // map[轨道名]动画名

    // 示例：
    // TrackBindings["anim_face"] = "anim_head_idle"
    // TrackBindings["stalk_bottom"] = "anim_shooting"
}
```

**自动绑定算法：**
```go
func analyzeTrackBinding(tracks, animations) map[string]string {
    bindings := make(map[string]string)

    for trackName, frames := range tracks {
        bestAnim := ""
        bestScore := 0.0

        for animName := range animations {
            // 计算轨道在该动画时间窗口内的位置方差
            variance := calculatePositionVariance(frames, animWindow)
            score := 1.0 + variance // 方差越大，越可能属于该动画

            if score > bestScore {
                bestScore = score
                bestAnim = animName
            }
        }

        bindings[trackName] = bestAnim
    }
    return bindings
}
```

**参考代码：**
- `cmd/animation_showcase/animation_cell.go:247-329` - 完整实现

---

#### 2. 简化的多动画播放逻辑

**核心改进：**
- 移除同步/异步模式切换的复杂性
- 每个动画维护独立的逻辑帧索引
- 通过 `TrackBindings` 实现轨道到动画的映射
- 统一的帧推进逻辑

**新的组件结构：**
```go
type ReanimComponent struct {
    // 简化后的字段
    CurrentAnimations []string              // 当前播放的动画列表
    AnimStates        map[string]*AnimState // 每个动画的状态
    TrackBindings     map[string]string     // 轨道绑定

    // 移除的冗余字段
    // - GlobalFrame (不再需要)
    // - TrackMapping (被 TrackBindings 取代)
    // - 同步/异步模式标志
}

type AnimState struct {
    LogicalFrame int     // 当前逻辑帧（0-based）
    FrameCount   int     // 总逻辑帧数
    Accumulator  float64 // 帧累加器
}
```

**新的 API：**
```go
// 播放单个动画（向后兼容）
PlayAnimation(entityID, "anim_idle")

// 播放多个动画（新实现）
PlayAnimations(entityID, []string{"anim_shooting", "anim_head_idle"})
// 内部：自动分析并设置 TrackBindings

// 手动配置轨道绑定
SetTrackBindings(entityID, map[string]string{
    "anim_face": "anim_head_idle",
    "stalk_bottom": "anim_shooting",
})
```

---

#### 3. 父子偏移系统 (Parent-Child Offset System)

**核心功能：**
- 定义轨道的父子关系（配置或默认规则）
- 计算父轨道的初始位置和当前位置
- 当子轨道和父轨道使用不同动画时，应用偏移

**实现逻辑：**
```go
type ReanimComponent struct {
    ParentTracks map[string]string // map[子轨道]父轨道
    // 例如：ParentTracks["anim_face"] = "anim_stem"
}

func getParentOffset(parentTrackName, animations, bindings) (offsetX, offsetY float64) {
    // 1. 找到父轨道控制的动画
    parentAnim := bindings[parentTrackName]

    // 2. 获取父轨道的初始位置（第一个可见帧）
    initX, initY := getFirstVisiblePosition(parentTrackName, parentAnim)

    // 3. 获取父轨道的当前位置
    currentX, currentY := getCurrentPosition(parentTrackName, parentAnim)

    // 4. 计算偏移
    offsetX = currentX - initX
    offsetY = currentY - initY
    return
}

// 渲染时应用偏移
func renderTrack(trackName) {
    x, y := getTrackPosition(trackName)

    // 检查是否有父轨道
    if parentName, hasParent := comp.ParentTracks[trackName]; hasParent {
        childAnim := comp.TrackBindings[trackName]
        parentAnim := comp.TrackBindings[parentName]

        // 只有当子父使用不同动画时才应用偏移
        if childAnim != parentAnim {
            offsetX, offsetY := getParentOffset(parentName)
            x += offsetX
            y += offsetY
        }
    }

    drawPart(x, y, ...)
}
```

**参考代码：**
- `cmd/animation_showcase/animation_cell.go:454-499` - 父子偏移计算
- `cmd/animation_showcase/animation_cell.go:399-408` - 渲染时偏移应用

---

#### 4. 渲染缓存优化 (Render Cache Optimization)

**核心策略：**
- 缓存每帧的渲染数据（图片引用、变换、偏移）
- 只在帧变化时更新缓存
- 渲染时直接使用缓存数据

**实现结构：**
```go
type renderPartData struct {
    Img      *ebiten.Image
    Frame    reanim.Frame
    OffsetX  float64
    OffsetY  float64
}

type ReanimComponent struct {
    // 缓存相关字段
    CachedRenderData []renderPartData
    LastRenderFrame  int
}

func (rs *ReanimSystem) prepareRenderCache(comp) {
    comp.CachedRenderData = comp.CachedRenderData[:0] // 重用切片

    for _, trackName := range visualTracks {
        // 1. 找到控制该轨道的动画
        animName := comp.TrackBindings[trackName]

        // 2. 计算物理帧索引
        logicalFrame := comp.AnimStates[animName].LogicalFrame
        physicalFrame := mapLogicalToPhysical(logicalFrame, animVisibles)

        // 3. 获取帧数据
        frame := comp.MergedTracks[trackName][physicalFrame]

        // 4. 计算父子偏移（如果有）
        offsetX, offsetY := calculateParentOffset(trackName, comp)

        // 5. 加入缓存
        comp.CachedRenderData = append(comp.CachedRenderData, renderPartData{
            Img:     comp.PartImages[frame.ImagePath],
            Frame:   frame,
            OffsetX: offsetX,
            OffsetY: offsetY,
        })
    }
}

func (rs *RenderSystem) renderReanimEntity(comp) {
    // 检查缓存是否有效
    currentFrame := comp.AnimStates[comp.CurrentAnimations[0]].LogicalFrame
    if currentFrame != comp.LastRenderFrame {
        rs.reanimSystem.prepareRenderCache(comp)
        comp.LastRenderFrame = currentFrame
    }

    // 快速渲染缓存数据
    for _, data := range comp.CachedRenderData {
        drawPart(data.Img, data.Frame, data.OffsetX, data.OffsetY)
    }
}
```

**性能收益：**
- 减少重复计算：轨道绑定查找、物理帧映射、父子偏移计算
- 预期性能提升：20-30%（基于 animation_showcase 的观察）

**参考代码：**
- `cmd/animation_showcase/animation_cell.go:368-422` - 缓存更新逻辑
- `cmd/animation_showcase/animation_cell.go:349-366` - 渲染使用缓存

---

#### 5. 配置系统升级 (Configuration System Enhancement)

**核心功能：**
- YAML 配置支持动画组合定义
- 支持轨道绑定策略（auto/manual）
- 支持父子关系配置
- 支持隐藏轨道配置

**配置格式：**

**注**：Story 13.6 中配置已迁移为集中单文件 `data/reanim_config.yaml`。以下为单个实体在集中配置文件中的结构示例：

```yaml
# data/reanim_config.yaml (集中配置文件片段)
animations:
  - id: "peashooter"
    name: "PeaShooter"
    reanim_file: "data/reanim/PeaShooterSingle.reanim"
    default_animation: "anim_idle"

    images:
      IMAGE_REANIM_PEASHOOTER_HEAD: "assets/reanim/PeaShooter_Head.png"
      # ... 其他图片映射

    available_animations:
      - name: anim_idle
        display_name: 待机
      - name: anim_shooting
        display_name: 攻击
      - name: anim_head_idle
        display_name: 头部摇晃

    animation_combos:
      - name: attack
        display_name: 攻击+摇晃
        animations:
          - anim_shooting
          - anim_head_idle

        # 轨道绑定策略
        binding_strategy: auto  # auto 或 manual

        # 父子关系定义
        parent_tracks:
          anim_face: anim_stem  # anim_face 的父轨道是 anim_stem

        # 隐藏轨道
        hidden_tracks:
          - anim_blink
```

**加载逻辑：**
```go
func LoadReanimConfig(configPath string) (*ReanimConfig, error) {
    // 解析 YAML
    config := parseYAML(configPath)

    // 返回配置对象
    return config, nil
}

func ApplyReanimConfig(entityID, configPath) error {
    config := LoadReanimConfig(configPath)

    // 设置动画组合
    for _, combo := range config.AnimationCombos {
        if combo.BindingStrategy == "auto" {
            // 使用自动绑定分析
            bindings := analyzeTrackBinding(...)
        } else {
            // 使用手动配置
            bindings := combo.ManualBindings
        }

        reanimComp.TrackBindings = bindings
        reanimComp.ParentTracks = combo.ParentTracks
        // ...
    }
}
```

**参考配置：**
- `cmd/animation_showcase/config.yaml` - 完整配置示例
- `cmd/animation_showcase/config_fixed.yaml` - 140 个动画的配置

---

### Stories Overview (Story 概览)

#### Story 13.1: 引入轨道绑定机制
**目标**: 添加 `TrackBindings` 字段和自动绑定分析算法，使每个轨道可以绑定到不同的动画。

**核心交付物**:
- `ReanimComponent.TrackBindings` 字段
- `analyzeTrackBinding()` 自动分析算法
- `SetTrackBindings()` API
- 单元测试覆盖轨道绑定逻辑

**AC**: 能够为豌豆射手配置"头部用 anim_head_idle，身体用 anim_shooting"

---

#### Story 13.2: 简化多动画播放逻辑
**目标**: 移除同步/异步模式复杂性，统一使用轨道绑定系统。

**核心交付物**:
- 移除 `GlobalFrame` 和 `TrackMapping` 字段
- 重构 `PlayAnimations()` 使用 `TrackBindings`
- 统一的帧推进逻辑 `updateAnimationStates()`
- 向后兼容测试（确保现有代码不受影响）

**AC**: 所有现有动画（豌豆射手、向日葵等）正常播放，无回归

---

#### Story 13.3: 父子偏移系统
**目标**: 实现父子关系定义和偏移计算，修复头部不跟随身体摆动的问题。

**核心交付物**:
- `ReanimComponent.ParentTracks` 字段
- `getParentOffset()` 偏移计算函数
- 渲染时偏移应用逻辑
- 豌豆射手攻击动画修复验证

**AC**: 豌豆射手攻击时，头部正确跟随身体摆动

---

#### Story 13.4: 渲染系统优化
**目标**: 引入渲染缓存机制，减少重复计算，提升性能。

**核心交付物**:
- `ReanimComponent.CachedRenderData` 字段
- `prepareRenderCache()` 缓存更新函数
- 修改 `RenderSystem.renderReanimEntity()` 使用缓存
- 性能基准测试（对比优化前后）

**AC**: 渲染性能提升 20%+（基于基准测试）

---

#### Story 13.5: 配置系统升级
**目标**: 支持 YAML 配置动画组合和轨道绑定，减少硬编码。

**核心交付物**:
- Reanim 配置文件格式定义（YAML schema）
- `LoadReanimConfig()` 加载函数
- `ApplyReanimConfig()` 应用函数
- 示例配置文件（豌豆射手、向日葵）

**AC**: 通过配置文件定义豌豆射手的攻击组合，无需修改代码

---

## Dependencies (依赖关系)

### Internal Dependencies (内部依赖)
- **Epic 6**: Reanim 系统基础（已完成）
  - 依赖：`internal/reanim` 解析器
  - 依赖：`BuildMergedTracks` 帧继承机制
  - 依赖：`ReanimSystem` 和 `RenderSystem` 基础架构

### Story Dependencies (Story 依赖)
```
Story 13.1 (轨道绑定)
    ↓
Story 13.2 (简化多动画) ← 依赖 13.1
    ↓
Story 13.3 (父子偏移) ← 依赖 13.1, 13.2
    ↓
Story 13.4 (渲染优化) ← 依赖 13.1, 13.2, 13.3
    ↓
Story 13.5 (配置系统) ← 依赖 13.1, 13.2, 13.3
```

**关键路径**: 13.1 → 13.2 → 13.3 → 13.4
**可并行**: Story 13.5 可以在 13.3 完成后并行开发

---

## Risks and Mitigations (风险与缓解)

### Risk 1: 回归风险 - 现有动画受影响 ⭐**Critical**
**风险描述**: 重构可能破坏现有的植物、僵尸、UI 动画

**影响范围**:
- 所有使用 `PlayAnimation()` 的实体（豌豆射手、向日葵、僵尸等）
- 所有使用 `PlayAnimations()` 的实体（可能有，虽然标记为"不再使用"）
- 主菜单的 `SelectorScreen.reanim` 动画

**缓解措施**:
1. **向后兼容 API**: 保留现有 `PlayAnimation()` API，内部重定向到新系统
2. **回归测试套件**: 创建视觉回归测试，对比重构前后的渲染结果
3. **分阶段迁移**:
   - Phase 1: 添加新字段，旧逻辑继续工作
   - Phase 2: 逐步迁移到新逻辑
   - Phase 3: 移除旧逻辑（确认无回归后）
4. **Feature Flag**: 使用配置开关，允许回退到旧系统

**验证标准**:
- 所有现有动画测试通过（100% pass rate）
- 视觉对比测试无差异（pixel-perfect 或 SSIM > 0.95）

---

### Risk 2: 性能回退 ⭐**High**
**风险描述**: 新的轨道绑定查找可能引入性能开销

**潜在影响**:
- 每帧需要查询 `TrackBindings` map (O(1) 但有 map 开销)
- 父子偏移计算增加 CPU 占用
- 缓存机制如果实现不当，可能反而降低性能

**缓解措施**:
1. **性能基准测试**: 建立基准，监控每个 Story 的性能影响
2. **缓存优先**: Story 13.4 优先实现，抵消前面的性能开销
3. **Profiling**: 使用 `pprof` 分析热点，针对性优化
4. **批量预计算**: 在 `prepareRenderCache()` 中一次性完成所有计算

**性能目标**:
- 渲染帧率不低于重构前（60 FPS 保持）
- 整体性能提升 20%+（Story 13.4 完成后）

---

### Risk 3: 配置复杂度 ⭐**Medium**
**风险描述**: 新的配置系统可能让用户困惑，增加学习成本

**潜在影响**:
- 开发者需要学习 YAML 配置格式
- 调试配置错误可能困难
- 文档维护成本增加

**缓解措施**:
1. **配置生成工具**: 提供自动生成配置的工具（参考 `animation_showcase/generate_config.go`）
2. **配置验证**: 加载时验证配置，提供清晰的错误信息
3. **默认策略**: 提供智能默认值，简单场景无需配置
4. **完善文档**: 更新 `CLAUDE.md` 和 `docs/reanim/` 文档

**文档更新**:
- `CLAUDE.md` - 添加 Reanim 配置指南
- `docs/reanim/reanim-config-guide.md` - 详细配置手册
- 示例配置文件（至少 5 个不同复杂度的例子）

---

### Risk 4: 技术债务转移 ⭐**Low**
**风险描述**: 重构可能引入新的技术债务

**潜在债务**:
- `animation_showcase` 的代码可能不完全适用于游戏场景
- 过度优化可能降低可读性
- 新字段命名可能不够清晰

**缓解措施**:
1. **Code Review**: 严格的代码审查，关注可读性和可维护性
2. **命名一致性**: 统一命名规范（参考现有代码风格）
3. **文档驱动**: 先写文档，后写代码，确保设计清晰
4. **技术评审**: 每个 Story 完成后进行技术评审

---

## Compatibility Requirements (兼容性要求)

### API 兼容性
- ✅ **向后兼容**: 现有 `PlayAnimation(entityID, "anim_idle")` API 保持不变
- ✅ **向后兼容**: 现有 `PlayAnimations(entityID, []string{...})` API 保持签名不变
- ⚠️ **内部重构**: API 内部实现改为使用新的轨道绑定系统
- ✅ **新增 API**: `SetTrackBindings()`, `SetParentTracks()` 不影响现有代码

### 数据兼容性
- ✅ **组件字段**: 新增字段不影响现有序列化/反序列化
- ✅ **存档兼容**: Reanim 配置不涉及存档数据
- ✅ **资源兼容**: 不改变 Reanim 文件格式或资源路径

### 性能兼容性
- ✅ **无性能回退**: 渲染帧率不低于重构前
- ✅ **内存占用**: 缓存字段增加内存，但幅度可控（< 5%）

---

## Definition of Done (完成定义)

### Epic-Level DoD

**代码完成**:
- [ ] 所有 5 个 Story 的代码完成并合并
- [ ] 代码通过 CI/CD 流水线（编译、测试、Lint）
- [ ] 代码审查通过（至少 1 名 reviewer 批准）

**功能验证**:
- [ ] 所有 AC (Acceptance Criteria) 通过
- [ ] 回归测试通过（现有动画无破坏）
- [ ] 性能基准测试达标（≥ 20% 提升）

**文档完成**:
- [ ] `CLAUDE.md` 更新（Reanim 配置指南）
- [ ] `docs/reanim/reanim-config-guide.md` 创建
- [ ] API 文档更新（`pkg/systems/reanim_system.go` 注释）
- [ ] 示例配置文件提供（至少 3 个）

**测试覆盖**:
- [ ] 单元测试覆盖率 ≥ 80%（新增代码）
- [ ] 集成测试覆盖主要场景（豌豆射手、僵尸、主菜单）
- [ ] 性能测试基准建立

**技术债务清理**:
- [ ] 移除 `GlobalFrame`、`TrackMapping` 等废弃字段（如果确认无回归）
- [ ] 清理注释中的"简化系统不再使用"等临时标记
- [ ] 代码风格统一（`gofmt`, `golint`）

---

## Success Metrics (成功指标)

### Quantitative Metrics (定量指标)

**性能指标**:
- 渲染性能提升 ≥ 20%（基于 100 实体同屏基准测试）
- 帧率稳定性提升（帧时间方差降低 ≥ 15%）
- 内存占用增加 < 5%（可接受的缓存成本）

**代码质量**:
- `reanim_system.go` 行数减少 ≥ 10%（从 1857 行）
- 循环复杂度降低 ≥ 20%（简化控制流）
- 单元测试覆盖率 ≥ 80%

**开发效率**:
- 添加新动画组合配置时间：< 5 分钟（使用 YAML）
- 调试动画问题时间：减少 30%（更清晰的架构）

---

### Qualitative Metrics (定性指标)

**用户体验**:
- 豌豆射手攻击动画自然流畅（头部跟随身体）
- 所有现有动画无视觉差异（回归测试通过）
- 主菜单动画正常播放（云朵、草丛等）

**开发者体验**:
- 配置系统易于理解（文档清晰）
- 调试信息更丰富（日志输出轨道绑定信息）
- 代码可读性提升（架构简化）

**技术架构**:
- 职责清晰：组件只存储数据，系统负责逻辑
- 扩展性强：添加新动画组合无需修改核心代码
- 可维护性高：减少技术债务

---

## Timeline Estimate (时间估算)

### Story-Level Estimates (Story 级别估算)

| Story | 描述 | 估算 | 依赖 | 风险 |
|-------|------|------|------|------|
| 13.1 | 轨道绑定机制 | 3-5 天 | Epic 6 | 中 |
| 13.2 | 简化多动画逻辑 | 5-7 天 | 13.1 | 高 |
| 13.3 | 父子偏移系统 | 3-4 天 | 13.1, 13.2 | 中 |
| 13.4 | 渲染系统优化 | 2-3 天 | 13.1-13.3 | 低 |
| 13.5 | 配置系统升级 | 2-3 天 | 13.1-13.3 | 低 |
| **总计** | | **15-22 天** | | |

### Phase Timeline (阶段时间线)

**Phase 1: 基础设施 (Week 1-2)**
- Story 13.1: 轨道绑定机制 (3-5 天)
- Story 13.2: 简化多动画逻辑 (5-7 天)
- **Milestone**: 能够配置"头部+身体"独立动画

**Phase 2: 核心功能 (Week 2-3)**
- Story 13.3: 父子偏移系统 (3-4 天)
- **Milestone**: 豌豆射手攻击动画修复

**Phase 3: 优化与配置 (Week 3-4)**
- Story 13.4: 渲染系统优化 (2-3 天)
- Story 13.5: 配置系统升级 (2-3 天)
- **Milestone**: 性能提升 20%+，配置化完成

**Phase 4: 验证与文档 (Week 4)**
- 回归测试 (1-2 天)
- 性能基准测试 (1 天)
- 文档更新 (1-2 天)
- **Milestone**: Epic 完成

**总时间线**: 3-4 周（根据实际开发进度调整）

---

## Reference Materials (参考资料)

### Internal Documentation (内部文档)
- `docs/reanim/reanim-format-guide.md` - Reanim 格式详解
- `docs/reanim/reanim-fix-guide.md` - Reanim 修复指南
- `docs/reanim/reanim-hybrid-track-discovery.md` - 混合轨道分析
- `docs/stories/6.5.story.md` - Reanim 帧继承实现
- `docs/stories/6.9.story.md` - 多动画叠加 API

### Reference Implementation (参考实现)
- `cmd/animation_showcase/animation_cell.go` - 正确的 Reanim 实现
- `cmd/animation_showcase/README.md` - 功能说明
- `cmd/animation_showcase/FEATURES.md` - 技术实现细节
- `cmd/animation_showcase/PROJECT_SUMMARY.md` - 项目总结

### External Resources (外部资源)
- 原版 PVZ 动画分析（截图、视频）
- Reanim 格式逆向工程文档（社区）

---

## Notes (备注)

### Design Decisions (设计决策)

**决策 1: 为什么选择轨道绑定而非轨道映射？**
- **背景**: 现有 `TrackMapping` 假设异步模式，架构复杂
- **决策**: 使用 `TrackBindings` 统一同步/异步模式
- **理由**: 更简洁，更符合 Reanim 格式的真实含义
- **参考**: `animation_showcase` 的成功实践

**决策 2: 缓存粒度选择**
- **选项 A**: 缓存整个渲染图像（`*ebiten.Image`）
- **选项 B**: 缓存渲染数据（图片引用+变换+偏移）
- **决策**: 选择 B
- **理由**: 灵活性更高，内存占用更低，支持运行时修改

**决策 3: 配置文件位置和格式**
- **初始决策（Story 13.5）**: `data/reanim_configs/`（分散多文件）
- **最终决策（Story 13.6）**: `data/reanim_config.yaml`（集中单文件）
- **理由**:
  - 集中管理便于维护和批量操作
  - 与 `animation_showcase` 的设计一致
  - 避免配置文件分散导致的不一致
  - 可配置性强，便于调试，符合项目设计哲学

### Future Enhancements (未来增强)

**Epic 后续可能的改进**:
1. **可视化调试工具**: 实时查看轨道绑定和父子关系
2. **动画编辑器**: GUI 工具配置动画组合
3. **性能分析面板**: 实时监控渲染性能
4. **更多自动化**: 自动检测父子关系（基于轨道名称规则）

---

## Story List (Story 列表)

1. ✅ [Story 13.1: 引入轨道绑定机制](../stories/13.1.story.md)
2. ✅ [Story 13.2: 简化多动画播放逻辑](../stories/13.2.story.md)
3. ✅ [Story 13.3: 父子偏移系统](../stories/13.3.story.md)
4. ✅ [Story 13.4: 渲染系统优化](../stories/13.4.story.md)
5. ✅ [Story 13.5: 配置系统升级](../stories/13.5.story.md)
6. ✅ [Story 13.6: 配置驱动的动画播放迁移](../stories/13.6.story.md)
7. ✅ [Story 13.7: 清理配置系统回退逻辑和重复配置](../stories/13.7.story.md)

**Phase 3 完成**: Story 13.7 已完成，所有回退逻辑和重复配置已清理，代码减少 134 行。

---

**Epic Owner**: PO (Sarah)
**Created**: 2025-11-07
**Status**: Planning
**Priority**: High (技术债务 + 性能优化)
