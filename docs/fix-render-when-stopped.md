# anim_open 墓碑消失问题修复

## 问题描述

当 `anim_open` 配置了 `render_when_stopped: true` 后，墓碑动画播放完成后仍然消失了。期望行为是墓碑在动画完成后保持显示在最后一帧。

## 问题现象

用户报告：
> anim_open render_when_stopped 配置了true, 完成后, 墓碑消失了
>
> 我知道了, 这里有实现错误, anim_open 动画播放是播放实际有效帧,现在是播放全部帧,导致播放完后,消失,等全部帧播放完后，才又显示.

观察到的行为：
1. 墓碑升起动画播放（约 0.6 秒，13 帧）✅
2. 墓碑消失 ❌（不应该发生）
3. 等待很长时间（约 35 秒）
4. 墓碑重新出现 ✅

## 根本原因分析

### 问题：FrameCount 设置错误

**位置**: `pkg/entities/selector_screen_factory.go:148-151`

**错误代码**:
```go
// 获取动画的物理帧总数
frameCount := 0
if animVisibles, exists := animVisiblesMap[animName]; exists {
    frameCount = len(animVisibles) // ❌ 使用物理帧总数（706 帧）
}
```

### anim_open 动画特征

- **物理帧总数**: 706 帧（轨道长度）
- **有效帧数**: 13 帧（第 0-12 帧）
- **FPS**: 20
- **FrameNum 值**:
  - 第 0-12 帧: `f=0`（可见）→ 墓碑升起动画
  - 第 13-705 帧: `f=-1`（隐藏）→ 不应该播放这些帧

### AnimVisiblesMap 数据

```
AnimVisiblesMap["anim_open"] = [
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,  // 第 0-12 帧：可见
    -1, -1, -1, ..., -1                      // 第 13-705 帧：隐藏
]
```

### 错误执行流程

1. **初始化**: `FrameCount = 706`（错误！）
2. **播放阶段**:
   - 第 0-12 帧: `visibles[0-12] = 0`，墓碑可见 ✅
   - 第 13-705 帧: `visibles[13-705] = -1`，墓碑隐藏 ❌
3. **停止**: 动画在第 705 帧停止，`visibles[705] = -1`，墓碑隐藏 ❌

### 正确执行流程

1. **初始化**: `FrameCount = 13`（正确！）
2. **播放阶段**:
   - 第 0-12 帧: `visibles[0-12] = 0`，墓碑可见 ✅
3. **停止**: 动画在第 12 帧停止，`visibles[12] = 0`，墓碑可见 ✅

## 修复方案

### 修复1: 计算可见帧区间（核心修复）

**位置**: `pkg/entities/selector_screen_factory.go:146-196`

对于每个独立动画，计算**第一个可见帧**和**最后一个可见帧**：

```go
// 找到第一个可见帧（visibles[i] != -1）
firstVisible := -1
for i := 0; i < len(animVisibles); i++ {
    if animVisibles[i] != -1 {
        firstVisible = i
        break
    }
}

// 找到最后一个可见帧（visibles[i] != -1）
lastVisible := -1
for i := len(animVisibles) - 1; i >= 0; i-- {
    if animVisibles[i] != -1 {
        lastVisible = i
        break
    }
}

if firstVisible >= 0 && lastVisible >= 0 {
    startFrame = firstVisible      // 起始帧
    frameCount = lastVisible - firstVisible + 1  // 帧数
}
```

**示例**:
```
anim_open (非循环):
- AnimVisiblesMap: [0,0,...,0,-1,-1,...-1]
- firstVisible: 0, lastVisible: 12
- StartFrame: 0, FrameCount: 13
- 播放: 0-12 帧，停在 12 帧 ✅

anim_grass (循环):
- AnimVisiblesMap: [-1,...,-1,0,0,...,0,-1,...,-1]
- firstVisible: 78, lastVisible: 102
- StartFrame: 78, FrameCount: 25
- 播放: 78-102 帧，循环回 78 帧 ✅
```

### 修复2: 添加 StartFrame 字段

**位置**: `pkg/components/reanim_component.go:20-27`

添加 `StartFrame` 字段支持从非零帧开始循环：

```go
type IndependentAnimState struct {
    // ...

    // StartFrame 起始帧索引（循环动画从此帧开始）
    // 用于支持"跳过前面的隐藏帧"场景
    // 例如：anim_grass 的可见帧是 78-102，StartFrame=78
    StartFrame int

    // FrameCount 总帧数（从 StartFrame 开始计算）
    // 例如：anim_grass 的 StartFrame=78, FrameCount=25 (78到102)
    FrameCount int

    // ...
}
```

### 修复3: 循环时跳回 StartFrame

**位置**: `pkg/systems/reanim_playback_strategy.go:359-382`

```go
// 计算结束帧（StartFrame + FrameCount）
endFrame := state.StartFrame + state.FrameCount

// 检查是否到达动画末尾
if state.CurrentFrame >= endFrame {
    if state.IsLooping {
        // 循环播放：重置到起始帧，继续播放
        state.CurrentFrame = state.StartFrame  // ✅ 不是 0
    } else {
        // 非循环动画：停在最后一帧
        state.CurrentFrame = endFrame - 1
        state.IsActive = false
    }
}
```

### 修复4: render_when_stopped 检查

**位置**: `pkg/systems/render_system.go:526-529`

```go
// 检查 render_when_stopped：如果动画已停止且配置为不渲染，则跳过
if !controllingState.IsActive && !controllingState.RenderWhenStopped {
    continue // 动画已停止，且配置为不渲染，跳过此轨道
}
```

## 修复效果

修复后：

### anim_open（墓碑，非循环）
1. ✅ `StartFrame = 0, FrameCount = 13`
2. ✅ 播放 0-12 帧（约 0.65 秒）
3. ✅ 停在第 12 帧（可见）
4. ✅ 墓碑在动画完成后保持显示

### anim_grass（草丛，循环）
1. ✅ `StartFrame = 78, FrameCount = 25`
2. ✅ 播放 78-102 帧（约 1.25 秒）
3. ✅ 循环回第 78 帧（不是 0）
4. ✅ 草丛不停晃动，不会消失 ✅

### 通用效果
- ✅ 循环动画只播放可见帧区间，不浪费时间在隐藏帧上
- ✅ 非循环动画停在最后可见帧，符合预期
- ✅ `render_when_stopped` 配置正常工作

## 技术细节

### 可见帧区间计算

对于动画定义轨道（以 `anim_` 开头），计算可见帧区间：

```
firstVisible = 第一个 visibles[i] != -1 的索引
lastVisible = 最后一个 visibles[i] != -1 的索引
StartFrame = firstVisible
FrameCount = lastVisible - firstVisible + 1
```

**示例**：
```
轨道名称          AnimVisiblesMap                          StartFrame  FrameCount  播放区间
anim_open       [0,0,...,0,-1,-1,...,-1]                 0           13          0-12
anim_grass      [-1,...,-1,0,0,...,0,-1,...,-1]          78          25          78-102
anim_cloud1     [0,0,0,...,0] (全部可见)                 0           706         0-705
```

### 循环逻辑

```go
// 初始化
state.CurrentFrame = state.StartFrame  // 从可见帧开始
state.FrameCount = 可见帧数量

// 帧推进
state.CurrentFrame++

// 循环检查
endFrame = state.StartFrame + state.FrameCount
if state.CurrentFrame >= endFrame {
    state.CurrentFrame = state.StartFrame  // 跳回起始帧
}
```

### render_when_stopped 的作用

`render_when_stopped` 字段控制**动画停止后的渲染行为**：

#### 场景 1: render_when_stopped = true（默认）
- **用途**: 墓碑、按钮等需要保持显示的元素
- **行为**: 动画播放完成后，继续渲染最后一帧
- **示例**: `anim_open`（墓碑升起后保持显示）

#### 场景 2: render_when_stopped = false
- **用途**: 特效、粒子等播放完后应该消失的元素
- **行为**: 动画播放完成后，停止渲染（消失）
- **示例**: 爆炸特效、闪光效果等

#### 实现位置

**渲染检查**（`pkg/systems/render_system.go:526-529`）：
```go
// 检查 render_when_stopped：如果动画已停止且配置为不渲染，则跳过
if !controllingState.IsActive && !controllingState.RenderWhenStopped {
    continue // 动画已停止，且配置为不渲染，跳过此轨道
}
```

#### 配置示例

```yaml
# 墓碑动画 - 播放完后保持显示
anim_open:
  is_looping: false
  render_when_stopped: true  # 保持显示 ✅

# 特效动画 - 播放完后消失
anim_explosion:
  is_looping: false
  render_when_stopped: false  # 消失 ✅
```

### 控制的轨道

`anim_open` 控制以下 11 个轨道：
1. SelectorScreen_BG_Right（墓碑背景）
2. SelectorScreen_Adventure_shadow
3. SelectorScreen_Adventure_button
4. SelectorScreen_Survival_shadow
5. SelectorScreen_Survival_button
6. SelectorScreen_Challenges_shadow
7. SelectorScreen_Challenges_button
8. SelectorScreen_ZenGarden_shadow
9. SelectorScreen_ZenGarden_button
10. SelectorScreen_StartAdventure_shadow
11. SelectorScreen_StartAdventure_button

## 测试验证

运行游戏后：
- 墓碑升起动画播放约 0.65 秒（13 帧 ÷ 20 FPS）
- 墓碑保持显示在最后一帧 ✅
- 不会出现"消失后重新出现"的问题 ✅

日志输出：
```
[SelectorScreen] ✅ Initialized independent animation 'anim_open' (frames=13, loop=false, active=true, render_stopped=true, delay=0.0s)
[ComplexScene] Animation 'anim_open' stopped at frame 12
```

## 相关文件

- `pkg/entities/selector_screen_factory.go:146-196` - 可见帧区间计算（核心修复）
- `pkg/components/reanim_component.go:10-50` - 添加 StartFrame 字段
- `pkg/systems/reanim_playback_strategy.go:340-383` - 循环时跳回 StartFrame
- `pkg/systems/render_system.go:526-529` - render_when_stopped 渲染检查
- `pkg/config/reanim_playback_config.go:32` - 配置结构定义
- `data/reanim_playback_config.yaml:79-91` - anim_grass 配置示例
- `internal/reanim/parser.go:59-107` - AnimVisiblesMap 构建逻辑
- `docs/fix-render-when-stopped.md` - 本文档

## 日期

2025-11-05

## 总结

这个问题有**两个层面**：

### 问题1: FrameCount 设置错误（已修复 ✅）
- ❌ **错误**: 使用物理帧总数（706 帧），导致播放了大量隐藏帧
- ✅ **正确**: 使用可见帧区间（例如 anim_open: 13 帧，anim_grass: 25 帧）

### 问题2: 循环动画从0开始（已修复 ✅）
- ❌ **错误**: 循环时总是跳回第 0 帧
- ✅ **正确**: 循环时跳回 `StartFrame`（可见帧的起始位置）

**修复方案**：
1. 添加 `StartFrame` 字段到 `IndependentAnimState`
2. 计算可见帧区间：`firstVisible` 到 `lastVisible`
3. 初始化：`CurrentFrame = StartFrame`, `FrameCount = 可见帧数量`
4. 循环：`CurrentFrame` 达到 `StartFrame + FrameCount` 时，跳回 `StartFrame`

**关键教训**：
- 对于 Reanim 动画系统，必须区分**物理帧总数**（轨道长度）和**可见帧区间**（实际播放范围）
- 循环动画应该在**可见帧区间**内循环，而不是从第 0 帧开始
