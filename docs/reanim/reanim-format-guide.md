# Reanim 动画格式完全理解指南

> 基于对 PeaShooterSingle.reanim 的深度分析得出的正确理解

**⚠️ 重要更新（2025-11-05）**：
- 本文档中提到的"双动画叠加机制"和 `buildMergedTracks` 局部实现已被废弃
- 现在统一使用 `internal/reanim/parser.go` 中的 `BuildMergedTracks` 函数
- 所有轨道的所有帧都设置 FrameNum 值（默认为0），包括纯视觉轨道
- 详见：`docs/qa/sprint-change-proposal-buildMergedTracks-deduplication.md`

---

## 目录

- [1. 概述](#1-概述)
- [2. 核心概念](#2-核心概念)
- [3. 关键发现](#3-关键发现)
- [4. 正确的渲染算法](#4-正确的渲染算法)
- [5. 实现示例](#5-实现示例)
- [6. 常见误区](#6-常见误区)

---

## 1. 概述

Reanim 是《植物大战僵尸》使用的骨骼动画格式，基于 XML 定义。它使用**时间窗口机制**和**父子层级关系**实现复杂的动画效果。

### 1.1 文件结构

```xml
<fps>12</fps>                    <!-- 帧率 -->
<track>
  <name>轨道名称</name>
  <t><x>...</x><y>...</y>...</t>  <!-- 物理帧数据 -->
  <t>...</t>
  ...
</track>
<track>...</track>
...
```

---

## 2. 核心概念

### 2.1 轨道类型

Reanim 中有**四种**轨道类型：

| 类型 | 特征 | 作用 | 示例 | 占比 |
|------|------|------|------|------|
| **混合轨道** ⭐ | 有图片 + 有 f 值 | **自我控制可见性的视觉部件** | SunFlower 所有轨道、Zombie_body | **76%** |
| **动画定义轨道** | 无图片，只有 f 值 | 控制时间窗口 | `anim_idle`, `anim_shooting` | 23% |
| **纯视觉轨道** | 有图片，无 f 值 | 被动渲染的部件 | 部分 Squash 轨道 | <1% |
| **逻辑轨道** | 无图片，有位置 | 定义挂载点 | `anim_stem`, `_ground` | <1% |

**⚠️ 重要发现：混合轨道是最常见的类型！**

**混合轨道示例：**
```xml
<track>
  <name>anim_idle</name>
  <t><f>-1</f></t>              <!-- 帧 0-3: 隐藏 -->
  <t></t><t></t><t></t>
  <t><x>14.3</x><y>20.4</y>     <!-- 帧 4+: 显示 -->
      <sx>0.8</sx><sy>0.712</sy>
      <f>0</f>                   <!-- f=0 开启显示 -->
      <i>IMAGE_REANIM_SUNFLOWER_HEAD</i></t>  <!-- 有图片 -->
</track>
```

**关键特点：**
- ✅ 既是视觉轨道（有图片，可以渲染）
- ✅ 又是控制轨道（有 f 值，可以控制自己的可见性）
- ✅ 不依赖外部动画定义轨道

### 2.2 帧继承机制

**关键规则：nil 值继承上一物理帧的值**

```xml
<t><x>10</x><y>20</y><f>0</f></t>  <!-- 物理帧 0: x=10, y=20, f=0 -->
<t></t>                             <!-- 物理帧 1: x=10, y=20, f=0 (继承) -->
<t><x>15</x></t>                    <!-- 物理帧 2: x=15, y=20, f=0 (部分继承) -->
```

**实现代码：**

```go
// 错误：直接读取原始帧
frame := track.Frames[physicalIndex]

// 正确：构建累积合并后的帧
mergedFrames := buildMergedTracks(reanimXML, standardFrameCount)
frame := mergedFrames[trackName][physicalIndex]
```

### 2.3 f (FrameNum) 值的真实含义

**重大发现：f 值的含义取决于轨道类型！**

#### 混合轨道中的 f 值（最常见）⭐

**关键规则：混合轨道通过自己的 f 值控制自己的可见性**

```xml
<track>
  <name>anim_idle</name>                 <!-- 混合轨道：有图片 + 有 f 值 -->
  <t><f>-1</f></t>                        <!-- 帧 0-3: 隐藏 -->
  <t></t><t></t><t></t>
  <t><x>14.3</x><y>20.4</y>               <!-- 帧 4+: 显示 -->
      <f>0</f>                             <!-- f=0 开启显示 -->
      <i>IMAGE_REANIM_SUNFLOWER_HEAD</i></t>
</track>
```

**渲染判断：**
```go
// 混合轨道：检查自己的 f 值
if track.HasImage() && track.HasFValues() {
    if mergedFrame.FrameNum == 0 {
        render(mergedFrame)  // ✅ 渲染
    } else {
        skip()               // ❌ 跳过
    }
}
```

#### 动画定义轨道中的 f 值

**关键规则：控制时间窗口，影响纯视觉轨道**

| f 值 | 含义 | 效果 |
|------|------|------|
| `f=0` | 时间窗口**打开** | 在此物理帧渲染部件 |
| `f=-1` | 时间窗口**关闭** | 在此物理帧不渲染任何部件 |

**示例：anim_shooting**

```xml
<track>
  <name>anim_shooting</name>
  <t><f>-1</f></t>  <!-- 帧 0-53: 窗口关闭 -->
  ...
  <t><f>0</f></t>   <!-- 帧 54: 窗口打开 -->
  <t></t>           <!-- 帧 55-78: 继承 f=0，窗口持续打开 -->
  ...
  <t><f>-1</f></t>  <!-- 帧 79+: 窗口关闭 -->
</track>
```

结果：**物理帧 54-78 是 anim_shooting 的活动窗口**

#### 纯视觉轨道中的 f 值（罕见）

**关键发现：纯视觉轨道没有自己的 f 值，依赖动画定义轨道**

```xml
<track>
  <name>anim_eye</name>             <!-- 纯视觉轨道：只有图片，无 f 值 -->
  <t><x>25.3</x><y>18.9</y>          <!-- 所有帧都没有 f 标签 -->
      <i>IMAGE_REANIM_SQUASH_EYEBROWS</i></t>
  ...
</track>
```

**渲染判断：**
```go
// 纯视觉轨道：依赖动画定义轨道的时间窗口
if track.HasImage() && !track.HasFValues() {
    if animDefTrack.FrameNum == 0 {
        render(mergedFrame)  // ✅ 在窗口内渲染
    }
}
```

---

## 3. 关键发现

### 3.1 渲染规则的正确理解

#### ✅ 正确规则（基于混合轨道发现）

```
渲染判断流程：

1. 跳过纯动画定义轨道（无图片，不渲染）
2. 跳过逻辑轨道（无图片，只是挂载点）

3. 混合轨道（76% 的情况）：
   - 检查自己的 f 值
   - f=0 → 渲染
   - f=-1 → 跳过

4. 纯视觉轨道（<1% 的情况）：
   - 检查动画定义轨道的时间窗口
   - 窗口打开 → 渲染
   - 窗口关闭 → 跳过
```

**代码实现：**
```go
func shouldRenderTrack(
    track *reanim.Track,
    mergedFrame reanim.Frame,
    animDefFrame reanim.Frame,
) bool {
    // 1. 必须有图片才能渲染
    if mergedFrame.ImagePath == "" {
        return false
    }

    // 2. 混合轨道：检查自己的 f 值
    if track.HasFValues() {
        return mergedFrame.FrameNum != nil && *mergedFrame.FrameNum == 0
    }

    // 3. 纯视觉轨道：检查动画定义轨道
    return animDefFrame.FrameNum != nil && *animDefFrame.FrameNum == 0
}
```

**结果：** 显示完整植物（符合原版游戏）

### 3.2 anim_stem：头部挂载点

**发现：anim_stem 是逻辑轨道（无图片），定义头部应该挂载的位置**

#### 数据分析

| 阶段 | 物理帧 | f 值 | 位置 | 说明 |
|------|--------|------|------|------|
| **anim_idle** | 4-28 | 0 | 摆动 (37.6→47.4, 48.7→42.8) | 茎干摆动时，anim_stem 跟随移动 |
| **anim_shooting** | 54-78 | -1 (继承) | 固定 (37.6, 48.7) | 射击时，anim_stem 保持静止 |

#### 作用

```
anim_stem 是父节点，头部部件是子节点：

头部最终位置 = anim_stem位置 + 头部相对偏移

在 anim_idle 中：
  - anim_stem 随茎干摆动
  - 头部继承这个摆动，实现"头随身体摆动"

在 anim_shooting 中：
  - anim_stem 保持静止
  - 头部只做自己的射击动作
```

### 3.3 双动画叠加机制

**原版游戏的渲染策略：同时播放 anim_idle (身体) + anim_shooting (头部)**

#### 轨道分类

```go
// 身体轨道（使用 anim_idle）
bodyTracks := []string{
    "backleaf",
    "backleaf_left_tip",
    "backleaf_right_tip",
    "stalk_bottom",
    "stalk_top",
    "frontleaf",
    "frontleaf_right_tip",
    "frontleaf_tip_left",
}

// 头部轨道（使用 anim_shooting + anim_stem 偏移）
headTracks := []string{
    "anim_face",
    "idle_mouth",
    "anim_blink",
    "idle_shoot_blink",
    "anim_sprout",
}
```

#### 渲染逻辑

```
身体部件：
  物理帧 = anim_idle 时间窗口[逻辑帧]
  位置 = 直接使用部件在该物理帧的位置

头部部件：
  物理帧_shooting = anim_shooting 时间窗口[逻辑帧]
  物理帧_idle = anim_idle 时间窗口[逻辑帧]

  anim_stem偏移 = anim_stem[物理帧_idle] - anim_stem初始位置
  位置 = 部件位置[物理帧_shooting] + anim_stem偏移
```

---

## 4. 正确的渲染算法

### 4.1 构建合并轨道

```go
func buildMergedTracks(reanimXML *reanim.ReanimXML, standardFrameCount int) map[string][]reanim.Frame {
    mergedTracks := make(map[string][]reanim.Frame)

    for _, track := range reanimXML.Tracks {
        // 累积变量
        accX, accY := 0.0, 0.0
        accSX, accSY := 1.0, 1.0
        accKX, accKY := 0.0, 0.0
        accF := 0
        accImg := ""

        mergedFrames := make([]reanim.Frame, standardFrameCount)

        for i := 0; i < standardFrameCount; i++ {
            if i < len(track.Frames) {
                frame := track.Frames[i]

                // 更新累积值（nil 则继承）
                if frame.X != nil {
                    accX = *frame.X
                }
                if frame.Y != nil {
                    accY = *frame.Y
                }
                if frame.ScaleX != nil {
                    accSX = *frame.ScaleX
                }
                if frame.ScaleY != nil {
                    accSY = *frame.ScaleY
                }
                if frame.SkewX != nil {
                    accKX = *frame.SkewX
                }
                if frame.SkewY != nil {
                    accKY = *frame.SkewY
                }
                if frame.FrameNum != nil {
                    accF = *frame.FrameNum
                }
                if frame.ImagePath != "" {
                    accImg = frame.ImagePath
                }
            }

            // 存储合并后的帧
            x, y := accX, accY
            sx, sy := accSX, accSY
            kx, ky := accKX, accKY
            f := accF

            mergedFrames[i] = reanim.Frame{
                X:         &x,
                Y:         &y,
                ScaleX:    &sx,
                ScaleY:    &sy,
                SkewX:     &kx,
                SkewY:     &ky,
                FrameNum:  &f,
                ImagePath: accImg,
            }
        }

        mergedTracks[track.Name] = mergedFrames
    }

    return mergedTracks
}
```

### 4.2 构建时间窗口映射

```go
func buildAnimVisibles(animDefTrack *reanim.Track, standardFrameCount int) []int {
    animVisibles := make([]int, standardFrameCount)
    currentValue := 0

    for i := 0; i < standardFrameCount; i++ {
        if i < len(animDefTrack.Frames) {
            frame := animDefTrack.Frames[i]
            if frame.FrameNum != nil {
                currentValue = *frame.FrameNum
            }
        }
        animVisibles[i] = currentValue
    }

    // 提取窗口打开的物理帧
    visibleFrameIndices := []int{}
    for i, v := range animVisibles {
        if v == 0 {
            visibleFrameIndices = append(visibleFrameIndices, i)
        }
    }

    return visibleFrameIndices
}
```

### 4.3 渲染单个动画

```go
func renderSingleAnimation(
    mergedTracks map[string][]reanim.Frame,
    visualTracks []string,
    visibleFrameIndices []int,
    logicalFrame int,
) {
    physicalFrame := visibleFrameIndices[logicalFrame]

    for _, trackName := range visualTracks {
        frame := mergedTracks[trackName][physicalFrame]

        // 只检查是否有图片，忽略 f 值
        if frame.ImagePath == "" {
            continue
        }

        renderPart(frame)
    }
}
```

### 4.4 渲染双动画叠加

```go
func renderDualAnimation(
    mergedTracks map[string][]reanim.Frame,
    visualTracks []string,
    idleVisibleFrames []int,
    shootingVisibleFrames []int,
    logicalFrame int,
) {
    idlePhysicalFrame := idleVisibleFrames[logicalFrame % len(idleVisibleFrames)]
    shootingPhysicalFrame := shootingVisibleFrames[logicalFrame]

    // 获取 anim_stem 偏移
    const stemInitX, stemInitY = 37.6, 48.7
    stemFrame := mergedTracks["anim_stem"][idlePhysicalFrame]
    stemOffsetX := *stemFrame.X - stemInitX
    stemOffsetY := *stemFrame.Y - stemInitY

    // 头部轨道
    headTracks := map[string]bool{
        "anim_face": true,
        "idle_mouth": true,
        "anim_blink": true,
        "idle_shoot_blink": true,
        "anim_sprout": true,
    }

    for _, trackName := range visualTracks {
        var frame reanim.Frame

        if headTracks[trackName] {
            // 头部：使用 anim_shooting + anim_stem 偏移
            frame = mergedTracks[trackName][shootingPhysicalFrame]

            if frame.FrameNum != nil && *frame.FrameNum == -1 {
                continue
            }

            // 叠加 anim_stem 偏移
            if frame.X != nil {
                x := *frame.X + stemOffsetX
                frame.X = &x
            }
            if frame.Y != nil {
                y := *frame.Y + stemOffsetY
                frame.Y = &y
            }
        } else {
            // 身体：使用 anim_idle
            frame = mergedTracks[trackName][idlePhysicalFrame]
        }

        if frame.ImagePath == "" {
            continue
        }

        renderPart(frame)
    }
}
```

---

## 5. 实现示例

### 5.1 完整的豌豆射手渲染

```go
type PlantAnimation struct {
    reanimXML           *reanim.ReanimXML
    mergedTracks        map[string][]reanim.Frame
    visualTracks        []string
    idleVisibleFrames   []int
    shootingVisibleFrames []int

    currentLogicalFrame int
    isAttacking         bool
}

func (pa *PlantAnimation) Update() {
    pa.currentLogicalFrame++

    // 根据状态选择时间窗口
    maxFrames := len(pa.idleVisibleFrames)
    if pa.isAttacking {
        maxFrames = len(pa.shootingVisibleFrames)
    }

    if pa.currentLogicalFrame >= maxFrames {
        pa.currentLogicalFrame = 0
    }
}

func (pa *PlantAnimation) Render() {
    if pa.isAttacking {
        // 攻击状态：双动画叠加
        renderDualAnimation(
            pa.mergedTracks,
            pa.visualTracks,
            pa.idleVisibleFrames,
            pa.shootingVisibleFrames,
            pa.currentLogicalFrame,
        )
    } else {
        // 待机状态：只播放 anim_idle
        renderSingleAnimation(
            pa.mergedTracks,
            pa.visualTracks,
            pa.idleVisibleFrames,
            pa.currentLogicalFrame,
        )
    }
}
```

### 5.2 眨眼动画

眨眼通过 `idle_shoot_blink` 轨道实现，属于头部动画：

```
物理帧 64: IMAGE_REANIM_PEASHOOTER_BLINK1
物理帧 66: IMAGE_REANIM_PEASHOOTER_BLINK2
物理帧 68: IMAGE_REANIM_PEASHOOTER_BLINK1
物理帧 70: f=-1 (眨眼结束)
```

**关键：** 眨眼轨道与 `anim_face` 在同一位置渲染，通过覆盖实现眨眼效果。

---

## 6. 常见误区

### 误区 1：严格遵守所有 f=-1

**错误代码：**

```go
// ❌ 错误：这会导致身体消失
if frame.FrameNum != nil && *frame.FrameNum == -1 {
    continue
}
```

**问题：**
- 在 anim_shooting 期间，身体部件的 f=-1
- 严格遵守会跳过身体渲染
- 结果只显示头部

**修正：**

```go
// ✅ 正确：只检查动画定义轨道，部件轨道忽略 f 值
if animDefFrame.FrameNum == 0 && partFrame.ImagePath != "" {
    render(partFrame)
}
```

### 误区 2：忽略父子层级关系

**错误代码：**

```go
// ❌ 错误：头部不会随身体摆动
headFrame := mergedTracks["anim_face"][shootingPhysicalFrame]
render(headFrame)
```

**问题：**
- 没有考虑 anim_stem 的偏移
- 头部位置固定，不会随身体摆动

**修正：**

```go
// ✅ 正确：头部继承 anim_stem 偏移
stemOffset := getStemOffset(idlePhysicalFrame)
headFrame := mergedTracks["anim_face"][shootingPhysicalFrame]
headFrame.X += stemOffset.X
headFrame.Y += stemOffset.Y
render(headFrame)
```

### 误区 3：直接读取原始帧数据

**错误代码：**

```go
// ❌ 错误：没有处理帧继承
frame := track.Frames[physicalIndex]
```

**问题：**
- 原始帧可能有 nil 值
- 需要从前面的帧继承

**修正：**

```go
// ✅ 正确：使用合并后的轨道
mergedTracks := buildMergedTracks(reanimXML, standardFrameCount)
frame := mergedTracks[trackName][physicalIndex]
```

### 误区 4：混淆动画定义轨道和部件轨道

**错误理解：**

```
anim_shooting 轨道有图片，应该渲染它
```

**实际情况：**

```
anim_shooting 是动画定义轨道，没有图片
它的作用是控制时间窗口，不是直接渲染
```

### 误区 5：不理解双动画叠加

**错误理解：**

```
攻击时只播放 anim_shooting
→ 结果：只有头部在动，身体僵硬
```

**正确理解：**

```
攻击时同时播放：
  - anim_idle（身体摆动）
  - anim_shooting（头部射击）
  - 头部通过 anim_stem 继承身体摆动
→ 结果：完整的攻击动画
```

---

## 7. 验证方法

### 7.1 创建对比测试程序

建议创建三个渲染模式对比：

1. **严格模式**：严格遵守所有 f=-1
   - 预期：只显示头部

2. **忽略模式**：忽略部件 f=-1
   - 预期：显示完整植物，但头部不随身体摆动

3. **双动画模式**：anim_idle + anim_shooting + anim_stem 偏移
   - 预期：完整植物，头部随身体摆动（与原版一致）

### 7.2 关键验证点

- [ ] 身体部件在攻击时可见
- [ ] 头部在攻击时可见
- [ ] 头部随身体摆动
- [ ] 眨眼动画正常播放
- [ ] 茎干摆动幅度正确
- [ ] 时间窗口切换正确

---

## 8. 总结

### 核心要点

1. **四种轨道类型**：
   - **混合轨道（76%）⭐**：有图片 + 有 f 值，自我控制可见性
   - 动画定义轨道（23%）：无图片，只有 f 值，控制时间窗口
   - 纯视觉轨道（<1%）：有图片，无 f 值，依赖时间窗口
   - 逻辑轨道（<1%）：无图片，有位置，定义挂载点

2. **帧继承机制**：nil 值从上一物理帧累积继承

3. **f 值含义分层**：
   - 混合轨道：检查自己的 f 值（f=0 显示，f=-1 隐藏）
   - 动画定义轨道：控制时间窗口（影响纯视觉轨道）
   - 纯视觉轨道：依赖动画定义轨道的时间窗口

4. **父子层级**：anim_stem 作为挂载点，头部继承其偏移

5. **双动画叠加**：身体用 idle，头部用 shooting + stem 偏移

### 正确的实现流程

```
1. 加载 Reanim 文件
   ↓
2. 构建合并轨道（处理帧继承）
   ↓
3. 构建时间窗口映射
   ↓
4. 根据状态选择渲染模式：
   - 待机：渲染 anim_idle
   - 攻击：渲染 anim_idle (身体) + anim_shooting (头部 + stem偏移)
   ↓
5. 对每个部件：
   - 获取对应物理帧的合并数据
   - 检查是否有图片
   - 应用变换矩阵
   - 渲染
```

---

## 9. 参考资料

### 相关文件

- 测试程序：`cmd/render_animation_comparison/main.go`
- Reanim 解析：`internal/reanim/reanim.go`
- 示例文件：
  - `assets/effect/reanim/PeaShooterSingle.reanim`（双动画系统）
  - `assets/effect/reanim/SunFlower.reanim`（100% 混合轨道）
  - `assets/effect/reanim/Wallnut.reanim`（简单结构）
  - `assets/effect/reanim/Squash.reanim`（多动画状态）
  - `assets/effect/reanim/Zombie.reanim`（复杂混合结构）

### 已完成研究

- [x] ✅ 其他植物的 reanim 结构对比（已分析 4 个实体）
- [x] ✅ 混合轨道机制的发现（76% 占比）
- [x] ✅ 僵尸动画的特殊机制（30 个混合轨道）

### 进一步研究

- [ ] 粒子效果与 reanim 的协同
- [ ] 多层父子关系的处理（超过 2 层的情况）
- [ ] 特殊植物（如大嘴花、投手类）的动画机制

---

**文档版本：** v2.0 - 混合轨道发现版
**更新日期：** 2025-10-29
**基于研究：**
- PeaShooterSingle.reanim（双动画系统）
- SunFlower.reanim（混合轨道典型案例）
- Wallnut.reanim、Squash.reanim、Zombie.reanim（普适性验证）

**重大更新（v2.0）：**
- ✅ 发现混合轨道类型（76% 占比）
- ✅ 修正 f 值渲染规则
- ✅ 验证跨实体普适性（4 种植物 + 僵尸）
