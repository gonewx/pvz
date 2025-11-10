# Reanim 系统彻底重构方案

## 📋 重构目标

**彻底删除 Epic 13 遗留的问题代码，用 animation_showcase 验证过的实现完全替换**

### 核心问题
1. ❌ `getVisualTracks()` 不检查 `VisibleTracks`/`HiddenTracks`
2. ❌ `PlayDefaultAnimation()` 不应用配置（hidden_tracks、parent_tracks）
3. ❌ ReanimComponent 有278行，包含大量废弃字段
4. ❌ 代码路径复杂，新旧逻辑混杂

### 重构原则
- ✅ **完全基于 animation_showcase 的 AnimationCell 实现**
- ✅ **不考虑向后兼容**：删除所有旧代码
- ✅ **配置驱动**：所有动画行为由 YAML 配置文件控制
- ✅ **简洁明了**：ReanimComponent 只保留必要字段

---

## 🏗️ 新架构设计

### 1. 新的 ReanimComponent（基于 AnimationCell）

```go
// ReanimComponent 新版动画组件（完全基于 animation_showcase 实现）
type ReanimComponent struct {
    // === 基础数据 ===
    ReanimName   string                         // Reanim 文件名
    ReanimXML    *reanim.ReanimXML             // 解析的动画数据
    PartImages   map[string]*ebiten.Image      // 部件图片
    MergedTracks map[string][]reanim.Frame     // 合并轨道

    // === 轨道分类 ===
    VisualTracks  []string                      // 视觉轨道列表（有图片）
    LogicalTracks []string                      // 逻辑轨道列表（无图片，仅变换）

    // === 播放状态 ===
    CurrentFrame     int                         // 当前帧索引
    FrameAccumulator float64                    // 帧累加器
    AnimationFPS     float64                    // 动画帧率
    CurrentAnimations []string                  // 当前播放的动画列表

    // === 动画数据 ===
    AnimVisiblesMap       map[string][]int      // 每个动画的可见性数组
    TrackAnimationBinding map[string]string     // 轨道到动画的绑定

    // === 配置字段 ===
    ParentTracks  map[string]string             // 父子关系
    HiddenTracks  map[string]bool               // 隐藏的轨道（黑名单）

    // === 渲染缓存 ===
    CachedRenderData []RenderPartData           // 渲染数据缓存
    LastRenderFrame  int                        // 上次渲染帧

    // === 控制标志 ===
    IsPaused     bool                           // 是否暂停
    IsLooping    bool                           // 是否循环
    IsFinished   bool                           // 是否完成（非循环动画）
}

// RenderPartData 渲染缓存数据（保持不变）
type RenderPartData struct {
    Img     *ebiten.Image
    Frame   reanim.Frame
    OffsetX float64
    OffsetY float64
}
```

**删除的字段**（约150行）：
- ❌ `AnimStates` - 复杂的多动画状态，改用单一 `CurrentFrame`
- ❌ `TrackBindings` - 改名为 `TrackAnimationBinding`（与 AnimationCell 一致）
- ❌ `VisibleTracks` - 改用 `HiddenTracks`（黑名单模式更清晰）
- ❌ `TrackConfigs` - 不再需要每轨道配置
- ❌ `BestPreviewFrame`, `FixedCenterOffset`, `CenterOffsetX/Y` - 简化
- ❌ 所有 Epic 13 遗留字段

---

### 2. 新的 ReanimSystem API

```go
type ReanimSystem struct {
    entityManager *ecs.EntityManager
    configManager *config.ReanimConfigManager
}

// === 核心 API（仅2个，与 animation_showcase 一致） ===

// Update 更新动画帧（核心循环）
func (s *ReanimSystem) Update(deltaTime float64)

// PlayAnimation 播放单个动画（基础API，不读配置）
// 用于：调试、特殊效果、简单实体
func (s *ReanimSystem) PlayAnimation(entityID ecs.EntityID, animName string) error

// PlayCombo 播放配置文件定义的动画组合（推荐API）
// 用于：所有正常游戏实体（植物、僵尸等）
// 自动应用：hidden_tracks, parent_tracks, binding_strategy
// 参数 comboName 为空字符串时，使用配置文件的 default_combo
func (s *ReanimSystem) PlayCombo(entityID ecs.EntityID, unitID, comboName string) error

// === 辅助方法 ===

// SetConfigManager 设置配置管理器
func (s *ReanimSystem) SetConfigManager(manager *config.ReanimConfigManager)

// prepareRenderCache 准备渲染缓存（内部方法）
func (s *ReanimSystem) prepareRenderCache(comp *ReanimComponent)
```

**删除的 API**（约30个方法）：
- ❌ `SetTrackBindings`, `GetTrackBindings` - 不再暴露，内部自动处理
- ❌ `SetParentTracks`, `GetParentOffset` - 不再暴露，内部自动处理
- ❌ `HideTrack`, `ShowTrack` - 改用配置文件的 hidden_tracks
- ❌ `PlayAnimationNoLoop`, `SetLooping` - 改用配置文件控制
- ❌ 所有废弃的 Epic 13 API

---

### 3. 核心实现逻辑（直接移植 AnimationCell）

#### Update() - 帧推进

```go
func (s *ReanimSystem) Update(deltaTime float64) {
    entities := ecs.GetEntitiesWith1[*ReanimComponent](s.entityManager)

    for _, entityID := range entities {
        comp, _ := ecs.GetComponent[*ReanimComponent](s.entityManager, entityID)

        if comp.IsPaused || comp.IsFinished {
            continue
        }

        // 累加帧
        comp.FrameAccumulator += deltaTime * comp.AnimationFPS

        if comp.FrameAccumulator >= 1.0 {
            comp.FrameAccumulator -= 1.0
            comp.CurrentFrame++

            // 计算总帧数（所有动画的最大可见帧数）
            maxVisibleCount := 0
            for _, animName := range comp.CurrentAnimations {
                visibles := comp.AnimVisiblesMap[animName]
                count := countVisibleFrames(visibles)
                if count > maxVisibleCount {
                    maxVisibleCount = count
                }
            }

            // 循环检查
            if comp.CurrentFrame >= maxVisibleCount {
                if comp.IsLooping {
                    comp.CurrentFrame = 0
                } else {
                    comp.CurrentFrame = maxVisibleCount - 1
                    comp.IsFinished = true
                }
            }
        }
    }
}
```

#### PlayCombo() - 播放组合（完整实现）

```go
func (s *ReanimSystem) PlayCombo(entityID ecs.EntityID, unitID, comboName string) error {
    // 1. 获取组合配置
    combo, err := s.configManager.GetCombo(unitID, comboName)
    if err != nil {
        return err
    }

    comp, _ := ecs.GetComponent[*ReanimComponent](s.entityManager, entityID)

    // 2. 设置动画列表
    comp.CurrentAnimations = combo.Animations
    comp.CurrentFrame = 0
    comp.IsFinished = false

    // 3. 重建动画数据
    comp.AnimVisiblesMap = make(map[string][]int)
    for _, animName := range combo.Animations {
        visibles := buildVisiblesArray(comp.ReanimXML, comp.MergedTracks, animName)
        comp.AnimVisiblesMap[animName] = visibles
    }

    // 4. 设置父子关系
    if len(combo.ParentTracks) > 0 {
        comp.ParentTracks = combo.ParentTracks
    }

    // 5. 设置隐藏轨道
    if len(combo.HiddenTracks) > 0 {
        comp.HiddenTracks = make(map[string]bool)
        for _, track := range combo.HiddenTracks {
            comp.HiddenTracks[track] = true
        }
    }

    // 6. 自动分析轨道绑定
    if combo.BindingStrategy == "auto" {
        comp.TrackAnimationBinding = analyzeTrackBinding(comp)
    } else if combo.BindingStrategy == "manual" {
        comp.TrackAnimationBinding = combo.ManualBindings
    }

    return nil
}
```

#### prepareRenderCache() - 渲染缓存（关键修复）

```go
func (s *ReanimSystem) prepareRenderCache(comp *ReanimComponent) {
    comp.CachedRenderData = comp.CachedRenderData[:0]

    for _, trackName := range comp.VisualTracks {
        // ✅ 关键修复：检查隐藏轨道
        if comp.HiddenTracks != nil && comp.HiddenTracks[trackName] {
            continue  // 跳过隐藏轨道
        }

        // 查找控制该轨道的动画
        animName := comp.TrackAnimationBinding[trackName]
        if animName == "" {
            animName = comp.CurrentAnimations[0]  // 默认使用第一个动画
        }

        visibles := comp.AnimVisiblesMap[animName]
        physicalFrame := mapLogicalToPhysical(comp.CurrentFrame, visibles)

        if physicalFrame < 0 {
            continue
        }

        frames := comp.MergedTracks[trackName]
        if physicalFrame >= len(frames) {
            continue
        }

        frame := frames[physicalFrame]
        if frame.ImagePath == "" {
            continue
        }

        // 计算父子偏移
        offsetX, offsetY := 0.0, 0.0
        if parentTrack, hasParent := comp.ParentTracks[trackName]; hasParent {
            offsetX, offsetY = getParentOffset(comp, parentTrack, animName)
        }

        img := comp.PartImages[frame.ImagePath]
        if img == nil {
            continue
        }

        comp.CachedRenderData = append(comp.CachedRenderData, RenderPartData{
            Img:     img,
            Frame:   frame,
            OffsetX: offsetX,
            OffsetY: offsetY,
        })
    }
}
```

---

## 📂 需要修改的文件

### 核心文件（完全重写）

1. **pkg/components/reanim_component.go** - 278行 → 约80行
   - 删除所有 Epic 13 字段
   - 简化为 AnimationCell 风格

2. **pkg/systems/reanim_system.go** - 2808行 → 约800行
   - 删除所有废弃 API
   - 重写核心逻辑（Update, PlayCombo, prepareRenderCache）
   - 直接移植 AnimationCell 的实现

### 使用方代码（适配新 API）

3. **pkg/entities/plant_factory.go**
   - 所有 `PlayDefaultAnimation()` 调用保持不变（API 兼容）
   - 删除手动设置 `VisibleTracks` 的代码（改用配置文件）

4. **pkg/entities/zombie_factory.go**
   - 同上

5. **pkg/systems/behavior_system.go**
   - 攻击动画切换：改用 `PlayCombo()`

6. **pkg/systems/render_system.go**
   - 渲染逻辑保持不变（使用 `CachedRenderData`）

### 测试文件（需要更新）

7. **pkg/systems/reanim_system_test.go**
   - 删除废弃 API 的测试
   - 更新为新 API 测试

8. **pkg/entities/*_test.go**
   - 更新 Mock 对象

---

## 🚀 实施步骤

### Phase 1: 核心重构（2-3小时）

1. ✅ **备份当前实现**
   ```bash
   cp pkg/components/reanim_component.go pkg/components/reanim_component.go.backup
   cp pkg/systems/reanim_system.go pkg/systems/reanim_system.go.backup
   ```

2. ✅ **重写 ReanimComponent**
   - 创建新的简化结构体
   - 只保留 AnimationCell 的字段

3. ✅ **重写 ReanimSystem 核心方法**
   - `Update()` - 直接移植 AnimationCell.Update()
   - `PlayCombo()` - 直接移植 AnimationCell.SetAnimationCombo()
   - `prepareRenderCache()` - 直接移植 AnimationCell.updateRenderCache()

4. ✅ **删除废弃代码**
   - 删除所有 Epic 13 遗留方法
   - 删除所有不再使用的字段

### Phase 2: 适配使用方（1-2小时）

5. ✅ **更新工厂函数**
   - 删除 `plant_factory.go` 中的 `VisibleTracks` 设置
   - 删除 `zombie_factory.go` 中的硬编码配置

6. ✅ **更新行为系统**
   - 攻击动画切换改用 `PlayCombo()`

7. ✅ **验证编译**
   ```bash
   go build ./...
   ```

### Phase 3: 测试验证（1小时）

8. ✅ **运行游戏测试**
   ```bash
   go run . --verbose
   ```
   - 种植向日葵：验证正常显示
   - 种植豌豆射手：验证正常显示
   - 僵尸出现：验证正常显示

9. ✅ **运行 animation_showcase**
   ```bash
   go run cmd/animation_showcase/*.go
   ```
   - 验证与主游戏行为一致

10. ✅ **运行单元测试**
    ```bash
    go test ./pkg/entities/... -v
    go test ./pkg/systems/... -v
    ```

---

## ✅ 验收标准

### 功能验收

- [ ] **向日葵**：种植后正常显示，生产阳光动画正常
- [ ] **豌豆射手**：种植后正常显示，攻击动画正常（头部跟随身体）
- [ ] **僵尸**：正常显示，行走动画正常
- [ ] **特效**：粒子效果、阳光收集正常

### 代码质量

- [ ] **ReanimComponent**：字段数量 ≤ 15个（当前278行 → 约80行）
- [ ] **ReanimSystem**：代码行数 ≤ 1000行（当前2808行 → 约800行）
- [ ] **无废弃字段**：删除所有 Epic 13 遗留字段
- [ ] **无废弃 API**：删除所有不再使用的方法

### 性能验收

- [ ] **帧率稳定**：60 FPS 无卡顿
- [ ] **渲染正确**：所有动画渲染正确，无闪烁
- [ ] **缓存有效**：`prepareRenderCache()` 正确检查 `HiddenTracks`

---

## 🎯 预期成果

| 指标 | 重构前 | 重构后 | 改进 |
|------|--------|--------|------|
| ReanimComponent 字段数 | 30+ | ~12 | -60% |
| ReanimComponent 代码行数 | 278 | ~80 | -71% |
| ReanimSystem 代码行数 | 2808 | ~800 | -71% |
| API 数量 | 50+ | ~8 | -84% |
| 代码复杂度 | 高（多层抽象） | 低（单一实现） | ✅ |
| 可维护性 | 差（新旧混杂） | 优（清晰简洁） | ✅ |
| Bug 数量 | 多（隐藏轨道不生效） | 0 | ✅ |

---

## 📌 关键决策

### 为什么不考虑向后兼容？

1. **Epic 13 代码质量差**：大量 Bug 和设计缺陷
2. **AnimationCell 已验证**：animation_showcase 运行完美
3. **重构成本低**：修改点集中，影响范围可控
4. **长期收益高**：简化维护，减少 Bug

### 为什么删除这么多代码？

1. **废弃字段**：Epic 13 遗留的无用字段（如 `AnimStates`）
2. **废弃 API**：不再使用的方法（如 `SetTrackBindings`）
3. **复杂抽象**：过度设计的多动画状态管理
4. **重复逻辑**：与 AnimationCell 功能重复的代码

### 如何保证不引入新 Bug？

1. **逐步移植**：每个方法都基于 AnimationCell 验证过的实现
2. **关键修复**：`prepareRenderCache()` 检查 `HiddenTracks`
3. **完整测试**：运行游戏 + 单元测试 + animation_showcase
4. **备份代码**：保留旧代码备份，可快速回滚

---

## 🔥 立即开始？

请确认以上方案后，我将立即开始实施：

1. ✅ Phase 1: 重写核心组件（2-3小时）
2. ✅ Phase 2: 适配使用方（1-2小时）
3. ✅ Phase 3: 测试验证（1小时）

**总耗时**：约 **4-6小时**

是否开始执行？
