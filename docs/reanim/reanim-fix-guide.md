# Reanim 渲染系统修复指南

> 基于正确理解的实现修复路线图

**⚠️ 重要更新（2025-11-05）**：
- 本文档中提到的"双动画叠加机制"和 `buildMergedTracks` 局部实现已被废弃
- 现在统一使用 `internal/reanim/parser.go` 中的 `BuildMergedTracks` 函数
- 所有轨道的所有帧都设置 FrameNum 值（默认为0），包括纯视觉轨道
- 详见：`docs/qa/sprint-change-proposal-buildMergedTracks-deduplication.md`

---

## 问题诊断

### 当前实现的问题

通过对比测试程序 `cmd/render_animation_comparison/main.go` 的三种渲染模式，我们发现了当前实现的问题：

**问题症状：**
- ✅ 头部显示正常
- ❌ 身体部件在攻击时消失或显示不完整
- ❌ 头部不随身体摆动（僵硬）
- ❌ 眨眼动画可能缺失

**根本原因：**
1. 错误地对所有轨道的 `f=-1` 都进行隐藏处理
2. 没有实现双动画叠加机制
3. 没有实现 anim_stem 父子层级关系

---

## 修复路线图

### 阶段 1：修复帧继承机制 ✅

**问题：** 直接读取原始帧，没有处理 nil 值继承

**影响文件：**
- `pkg/systems/reanim_system.go`

**修复方法：**

```go
// ❌ 错误代码
func (rs *ReanimSystem) GetFrame(track *reanim.Track, frameIndex int) reanim.Frame {
    if frameIndex < len(track.Frames) {
        return track.Frames[frameIndex]  // 可能有 nil 值
    }
    return reanim.Frame{}
}

// ✅ 正确代码
func (rs *ReanimSystem) buildMergedTracks(reanimXML *reanim.ReanimXML) map[string][]reanim.Frame {
    // 见 docs/reanim-format-guide.md 第 4.1 节
    // 实现累积继承逻辑
}
```

**验证：**
- [ ] 所有物理帧都有完整的变换数据（无 nil）
- [ ] 空帧正确继承前一帧的值

---

### 阶段 2：修复 f 值判断逻辑 🔥 **关键**

**问题：** 对部件轨道的 `f=-1` 也进行隐藏

**影响文件：**
- `pkg/systems/render_system.go` (DrawReanimEntity 函数)
- `pkg/systems/reanim_system.go` (GetVisibleTracks 函数)

**当前错误代码定位：**

```go
// 在 render_system.go 或 reanim_system.go 中查找类似代码：
if mergedFrame.FrameNum != nil && *mergedFrame.FrameNum == -1 {
    inVisibleTracks := false
    if reanim.VisibleTracks != nil && len(reanim.VisibleTracks) > 0 {
        inVisibleTracks = reanim.VisibleTracks[track.Name]
    }
    if !inVisibleTracks {
        continue // ❌ 这里导致身体部件在 anim_shooting 时被跳过
    }
}
```

**修复步骤：**

#### 步骤 2.1：识别轨道类型

```go
// 添加辅助函数
func isAnimationDefinitionTrack(trackName string) bool {
    definitionTracks := map[string]bool{
        "anim_idle":       true,
        "anim_shooting":   true,
        "anim_head_idle":  true,
        "anim_full_idle":  true,
    }
    return definitionTracks[trackName]
}

func isLogicalTrack(trackName string) bool {
    logicalTracks := map[string]bool{
        "anim_stem": true,
    }
    return logicalTracks[trackName]
}
```

#### 步骤 2.2：构建时间窗口映射

```go
type ReanimComponent struct {
    // ... 现有字段 ...

    // 新增字段
    AnimVisibles map[string][]int  // 每个动画的时间窗口映射
    CurrentAnimation string          // 当前播放的动画
}

// 初始化时构建
func buildAnimVisibles(reanimXML *reanim.ReanimXML, animName string, standardFrameCount int) []int {
    // 见 docs/reanim-format-guide.md 第 4.2 节
}
```

#### 步骤 2.3：修改渲染判断逻辑

```go
// ❌ 删除或注释掉旧的 f=-1 检查
// if mergedFrame.FrameNum != nil && *mergedFrame.FrameNum == -1 {
//     continue
// }

// ✅ 新的检查逻辑
func (rs *RenderSystem) shouldRenderTrack(
    reanimComp *components.ReanimComponent,
    trackName string,
    physicalFrame int,
) bool {
    // 1. 跳过动画定义轨道（它们不渲染）
    if isAnimationDefinitionTrack(trackName) {
        return false
    }

    // 2. 跳过逻辑轨道（它们不渲染）
    if isLogicalTrack(trackName) {
        return false
    }

    // 3. 检查当前动画的时间窗口
    animVisibles := reanimComp.AnimVisibles[reanimComp.CurrentAnimation]
    if animVisibles[physicalFrame] == -1 {
        return false  // 窗口关闭
    }

    // 4. 检查是否有图片
    mergedFrame := reanimComp.MergedTracks[trackName][physicalFrame]
    if mergedFrame.ImagePath == "" {
        return false
    }

    return true
}
```

**验证：**
- [ ] 待机状态：显示完整植物
- [ ] 攻击状态：仍然显示完整植物（不再只显示头部）

---

### 阶段 3：实现双动画叠加 🔥 **关键**

**问题：** 攻击时只播放 anim_shooting，身体静止

**影响文件：**
- `pkg/systems/behavior_system.go` (PlantBehavior)
- `pkg/systems/reanim_system.go`

**修复步骤：**

#### 步骤 3.1：添加双动画支持

```go
type ReanimComponent struct {
    // ... 现有字段 ...

    // 新增字段
    IsBlending          bool     // 是否在混合两个动画
    PrimaryAnimation    string   // 主动画（如 anim_idle）
    SecondaryAnimation  string   // 次动画（如 anim_shooting）
}
```

#### 步骤 3.2：定义轨道归属

```go
// 在 pkg/systems/reanim_system.go 中添加
var headTracks = map[string]bool{
    "anim_face":         true,
    "idle_mouth":        true,
    "anim_blink":        true,
    "idle_shoot_blink":  true,
    "anim_sprout":       true,
}

func isHeadTrack(trackName string) bool {
    return headTracks[trackName]
}
```

#### 步骤 3.3：修改 PlayAnimation

```go
// ❌ 旧的实现
func (rs *ReanimSystem) PlayAnimation(entity ecs.Entity, animName string, loop bool) {
    reanimComp.CurrentAnimation = animName
    reanimComp.Loop = loop
    reanimComp.CurrentFrame = 0
}

// ✅ 新的实现
func (rs *ReanimSystem) PlayAnimation(entity ecs.Entity, animName string, loop bool) {
    reanimComp := ecs.GetComponent[*components.ReanimComponent](rs.em, entity)

    if animName == "anim_shooting" {
        // 攻击动画：启用混合模式
        reanimComp.IsBlending = true
        reanimComp.PrimaryAnimation = "anim_idle"      // 身体继续摆动
        reanimComp.SecondaryAnimation = "anim_shooting" // 头部射击
    } else {
        // 其他动画：单一模式
        reanimComp.IsBlending = false
        reanimComp.PrimaryAnimation = animName
        reanimComp.SecondaryAnimation = ""
    }

    reanimComp.Loop = loop
    reanimComp.CurrentFrame = 0
}
```

#### 步骤 3.4：修改渲染逻辑

```go
func (rs *RenderSystem) DrawReanimEntity(entity ecs.Entity, screen *ebiten.Image) {
    reanimComp := ecs.GetComponent[*components.ReanimComponent](rs.em, entity)
    posComp := ecs.GetComponent[*components.PositionComponent](rs.em, entity)

    if reanimComp.IsBlending {
        // 双动画模式
        rs.drawBlendedAnimation(entity, screen, posComp.X, posComp.Y)
    } else {
        // 单动画模式
        rs.drawSingleAnimation(entity, screen, posComp.X, posComp.Y)
    }
}

func (rs *RenderSystem) drawBlendedAnimation(
    entity ecs.Entity,
    screen *ebiten.Image,
    worldX, worldY float64,
) {
    reanimComp := ecs.GetComponent[*components.ReanimComponent](rs.em, entity)

    // 获取两个动画的物理帧
    primaryFrameIndices := reanimComp.AnimVisibles[reanimComp.PrimaryAnimation]
    secondaryFrameIndices := reanimComp.AnimVisibles[reanimComp.SecondaryAnimation]

    logicalFrame := reanimComp.CurrentFrame
    primaryPhysicalFrame := primaryFrameIndices[logicalFrame % len(primaryFrameIndices)]
    secondaryPhysicalFrame := secondaryFrameIndices[logicalFrame % len(secondaryFrameIndices)]

    // 遍历所有轨道
    for _, trackName := range reanimComp.VisualTracks {
        var physicalFrame int

        if isHeadTrack(trackName) {
            physicalFrame = secondaryPhysicalFrame  // 头部用 anim_shooting
        } else {
            physicalFrame = primaryPhysicalFrame    // 身体用 anim_idle
        }

        if !rs.shouldRenderTrack(reanimComp, trackName, physicalFrame) {
            continue
        }

        mergedFrame := reanimComp.MergedTracks[trackName][physicalFrame]
        rs.drawReanimPart(screen, mergedFrame, worldX, worldY)
    }
}
```

**验证：**
- [ ] 攻击时身体继续摆动
- [ ] 攻击时头部做射击动作
- [ ] 两个动画同步流畅

---

### 阶段 4：实现 anim_stem 父子层级 🔥 **关键**

**问题：** 头部不随身体摆动

**影响文件：**
- `pkg/systems/render_system.go`

**修复步骤：**

#### 步骤 4.1：获取 anim_stem 偏移

```go
func (rs *RenderSystem) getStemOffset(
    reanimComp *components.ReanimComponent,
    idlePhysicalFrame int,
) (float64, float64) {
    // anim_stem 的初始位置（从 reanim 文件中提取）
    const stemInitX = 37.6
    const stemInitY = 48.7

    stemFrames, ok := reanimComp.MergedTracks["anim_stem"]
    if !ok || idlePhysicalFrame >= len(stemFrames) {
        return 0, 0
    }

    stemFrame := stemFrames[idlePhysicalFrame]

    currentX := stemInitX
    currentY := stemInitY

    if stemFrame.X != nil {
        currentX = *stemFrame.X
    }
    if stemFrame.Y != nil {
        currentY = *stemFrame.Y
    }

    return currentX - stemInitX, currentY - stemInitY
}
```

#### 步骤 4.2：应用偏移到头部

```go
func (rs *RenderSystem) drawBlendedAnimation(
    entity ecs.Entity,
    screen *ebiten.Image,
    worldX, worldY float64,
) {
    reanimComp := ecs.GetComponent[*components.ReanimComponent](rs.em, entity)

    primaryPhysicalFrame := ...
    secondaryPhysicalFrame := ...

    // 获取 anim_stem 偏移
    stemOffsetX, stemOffsetY := rs.getStemOffset(reanimComp, primaryPhysicalFrame)

    for _, trackName := range reanimComp.VisualTracks {
        var physicalFrame int
        var applystemOffset bool

        if isHeadTrack(trackName) {
            physicalFrame = secondaryPhysicalFrame
            applystemOffset = true  // 头部需要偏移
        } else {
            physicalFrame = primaryPhysicalFrame
            applyStreamOffset = false
        }

        if !rs.shouldRenderTrack(reanimComp, trackName, physicalFrame) {
            continue
        }

        mergedFrame := reanimComp.MergedTracks[trackName][physicalFrame]

        // 应用 anim_stem 偏移
        if applyStreamOffset {
            if mergedFrame.X != nil {
                x := *mergedFrame.X + stemOffsetX
                mergedFrame.X = &x
            }
            if mergedFrame.Y != nil {
                y := *mergedFrame.Y + stemOffsetY
                mergedFrame.Y = &y
            }
        }

        rs.drawReanimPart(screen, mergedFrame, worldX, worldY)
    }
}
```

**验证：**
- [ ] 攻击时头部随身体一起摆动
- [ ] 摆动幅度与身体一致
- [ ] 头部相对位置保持正确

---

### 阶段 5：优化和清理

#### 步骤 5.1：移除 VisibleTracks 白名单机制

**原因：** 这是错误理解的产物，不再需要

```go
// ❌ 删除这些代码
type ReanimComponent struct {
    // ...
    VisibleTracks map[string]bool  // 删除
}

// 删除相关的白名单初始化代码
```

#### 步骤 5.2：添加配置支持

```go
// 在 config/constants.go 中添加
const (
    // Reanim 配置
    ReanimStemInitX = 37.6
    ReanimStemInitY = 48.7
)

// 动画定义轨道列表
var AnimationDefinitionTracks = map[string]bool{
    "anim_idle":       true,
    "anim_shooting":   true,
    "anim_head_idle":  true,
    "anim_full_idle":  true,
}

// 头部轨道列表
var HeadTracks = map[string]bool{
    "anim_face":         true,
    "idle_mouth":        true,
    "anim_blink":        true,
    "idle_shoot_blink":  true,
    "anim_sprout":       true,
}
```

#### 步骤 5.3：添加日志和调试

```go
// 在开发模式下输出调试信息
if config.DevMode {
    log.Printf("[Reanim] Playing animation: %s (blending: %v)",
        animName, reanimComp.IsBlending)
    log.Printf("[Reanim] Stem offset: (%.1f, %.1f)", stemOffsetX, stemOffsetY)
}
```

---

## 测试验证

### 测试用例

#### 测试 1：待机动画
```
场景：豌豆射手处于待机状态
预期：
  - ✅ 身体完整显示（叶子、茎干）
  - ✅ 头部显示（脸、嘴巴）
  - ✅ 身体有轻微摆动
  - ✅ 头部随身体一起摆动
```

#### 测试 2：攻击动画
```
场景：豌豆射手进入攻击状态
预期：
  - ✅ 身体继续摆动（不僵硬）
  - ✅ 头部做射击动作（嘴巴张开）
  - ✅ 头部随身体摆动（不是固定位置）
  - ✅ 在物理帧 64-68 出现眨眼
```

#### 测试 3：动画切换
```
场景：从待机切换到攻击，再切换回待机
预期：
  - ✅ 切换流畅，无跳帧
  - ✅ 身体摆动连续
  - ✅ 没有部件消失或闪烁
```

### 对比验证

使用测试程序进行对比：

```bash
# 运行三种模式对比
go run cmd/render_animation_comparison/main.go

# 观察三个画布：
# 左：严格模式（只有头部）
# 中：忽略模式（完整植物，头部不摆动）
# 右：双动画模式（完整植物，头部摆动）✅ 正确

# 右侧应该与原版游戏一致
```

---

## 迁移检查清单

### 代码修改

- [ ] 实现 `buildMergedTracks` 函数
- [ ] 实现 `buildAnimVisibles` 函数
- [ ] 添加轨道类型识别函数
- [ ] 修改 `shouldRenderTrack` 判断逻辑
- [ ] 添加 `IsBlending` 支持到 ReanimComponent
- [ ] 实现 `drawBlendedAnimation` 函数
- [ ] 实现 `getStemOffset` 函数
- [ ] 修改 `PlayAnimation` 启用混合模式
- [ ] 移除 VisibleTracks 白名单
- [ ] 添加配置常量

### 测试验证

- [ ] 待机动画正常
- [ ] 攻击动画正常
- [ ] 头部随身体摆动
- [ ] 眨眼动画出现
- [ ] 动画切换流畅
- [ ] 没有部件消失
- [ ] 性能无明显下降

### 文档更新

- [ ] 更新架构文档说明 Reanim 系统
- [ ] 添加代码注释说明关键逻辑
- [ ] 更新 CHANGELOG

---

## 性能考虑

### 优化建议

1. **缓存合并轨道**
   - 在加载 Reanim 时预构建，不要每帧计算

2. **缓存时间窗口**
   - AnimVisibles 在初始化时构建一次

3. **避免重复计算**
   - anim_stem 偏移每帧只计算一次

4. **批量渲染**
   - 收集所有需要渲染的部件，统一提交

### 预期性能影响

- **内存增加**：约 +10% (缓存合并轨道)
- **CPU 使用**：约 -5% (减少重复计算)
- **渲染时间**：无明显变化

---

## 回滚方案

如果修复后出现问题，可以：

1. **保留旧代码**
   ```go
   // TODO: 临时保留旧实现，验证后删除
   func (rs *RenderSystem) DrawReanimEntity_OLD(...) {
       // 旧的实现
   }
   ```

2. **使用功能开关**
   ```go
   if config.UseNewReanimSystem {
       rs.drawBlendedAnimation(...)
   } else {
       rs.drawSingleAnimation_OLD(...)
   }
   ```

3. **Git 分支管理**
   ```bash
   git checkout -b feature/reanim-fix
   # 修复完成后
   git merge feature/reanim-fix
   ```

---

## 常见问题

### Q1: 修复后仍然只显示头部？

**检查：**
- 是否正确识别了动画定义轨道？
- `shouldRenderTrack` 是否正确实现？
- 是否移除了旧的 f=-1 检查？

### Q2: 头部不随身体摆动？

**检查：**
- 是否实现了 `getStemOffset`？
- 是否在头部轨道应用了偏移？
- anim_stem 初始位置常量是否正确？

### Q3: 身体不摆动了？

**检查：**
- 攻击时是否正确设置了 `PrimaryAnimation = "anim_idle"`？
- 身体轨道是否使用了 primaryPhysicalFrame？

### Q4: 眨眼动画没有出现？

**检查：**
- `idle_shoot_blink` 是否在 headTracks 列表中？
- 是否正确使用了 secondaryPhysicalFrame？

---

## 下一步

修复完成后，建议：

1. 测试其他植物的 Reanim 动画
2. 实现僵尸的 Reanim 渲染
3. 优化性能和内存使用
4. 添加动画编辑器/调试工具

---

**文档版本：** v1.0
**更新日期：** 2025-10-29
**相关文档：** `docs/reanim-format-guide.md`
