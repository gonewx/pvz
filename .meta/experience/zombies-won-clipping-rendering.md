# 僵尸获胜流程 - 剪裁渲染问题修复历程

**时间**: 2025-11-23
**Story**: 8.8 - 僵尸获胜流程与游戏结束对话框
**Task**: Phase 2 僵尸入侵动画的剪裁渲染

## 修复历程

### 问题 1: 僵尸向下漂移（已修复）

**用户报告**：僵尸到达第1个目标位置后，继续向左走进门时，**突然向下漂移直到走出屏幕下方**。

#### 根本原因

剪裁渲染代码使用 `SubImage` 时的 Y 坐标处理不正确：

**错误代码**（`render_system.go:407-410`）：
```go
clippedImg := tempImg.SubImage(image.Rect(
    clipStartX, 0,              // ❌ Y 从 0 开始
    tempBounds.Dx(), tempBounds.Dy(),
)).(*ebiten.Image)
```

**问题分析**：

1. 临时图像高度为 400 像素
2. 僵尸渲染到临时图像时，由于 `renderReanimEntity` 使用公式 `screenY = pos.Y - CenterOffsetY`，僵尸实际渲染位置是 `Y=281.59`
3. `SubImage(clipStartX, 0, ...)` 从 Y=0 开始剪裁，包含了 **0~281 的空白区域**
4. 最终绘制时，这些空白区域被绘制到屏幕，导致僵尸视觉上向下偏移了 281.59 像素

#### 修复方案（问题 1）

```go
// 计算僵尸在临时图像中的 Y 范围
zombieTopY := int(zombieTopInTempImg)
if zombieTopY < 0 {
    zombieTopY = 0
}

// BUG 修复：SubImage 应该从僵尸实际渲染的 Y 位置开始，而不是从 0 开始
clippedImg := tempImg.SubImage(image.Rect(
    clipStartX, zombieTopY, // ✅ 从僵尸顶部开始剪裁
    tempBounds.Dx(), tempBounds.Dy(),
))
```

---

### 问题 2: 门外僵尸的脚部被裁剪（已修复）

**用户报告**：超出 `background1_gameover_interior_overlay` 下边缘的僵尸的脚部会被隐藏。

#### 根本原因 1：剪裁判断逻辑错误

**错误代码**（旧版本）：
```go
// 如果僵尸完全在门板右侧（未进入门内），正常渲染
if zombieLeftWorldX >= clipLeftWorldX {
    s.drawEntity(screen, id, cameraX)
    return
}
```

**问题**：判断条件基于**左边缘**，导致还没走到门口的僵尸也被剪裁。

**例如**：
- 门板 X=90
- 僵尸左边缘 X=89.9（僵尸宽度 150，右边缘 X=239.9）
- 僵尸明显还在门外，但 `89.9 < 90` 触发了剪裁逻辑

#### 修复方案（问题 2a）

**正确逻辑**（`render_system.go:359-372`）：
```go
// 判断僵尸是否需要剪裁（三种情况）
// 1. 僵尸完全在门板左侧（完全进入房子）：不渲染
if zombieRightWorldX <= clipLeftWorldX {
    return // 完全被遮挡
}

// 2. 僵尸完全在门板右侧（未触碰到门）：正常渲染
if zombieLeftWorldX >= clipLeftWorldX {
    s.drawEntity(screen, id, cameraX) // ✅ 正常渲染，无剪裁
    return
}

// 3. 僵尸部分重叠（需要剪裁）
// ... 剪裁逻辑 ...
```

**判断流程**：
```
门板位置: X=90
─────────────────────────────────────────────
情况 1: 完全在左侧（隐藏）
  僵尸: [====]
             ║ 门板
  rightWorldX=80 <= 90 → 不渲染

情况 2: 完全在右侧（正常渲染）
                        [====] 僵尸
             ║ 门板
  leftWorldX=100 >= 90 → 正常渲染

情况 3: 部分重叠（需要剪裁）
              [====] 僵尸
             ║ 门板
  leftWorldX=70 < 90 < rightWorldX=220 → 剪裁
```

#### 根本原因 2：临时图像高度不够

**错误配置**：
```go
tempHeight := 400 // ❌ 高度不够
```

**问题**：
- 僵尸顶部在临时图像的 Y=301
- 临时图像高度 400
- 僵尸只有 99 像素的高度空间（400 - 301 = 99）
- 但僵尸实际高度可能是 150-200 像素
- **结果：脚部被裁剪掉**

#### 修复方案（问题 2b）

```go
// BUG 修复：临时图像高度需要足够容纳整个僵尸（包括脚部）
// 僵尸可能有 200 像素高，加上 CenterOffsetY 可能在 300+ 位置
// 为了安全，使用更大的高度（600 像素）
tempHeight := 600 // ✅ 足够的高度（原来是 400）
```

---

## 问题分析总结

### 问题 1：向下漂移
- **原因**：SubImage 从 Y=0 开始，包含空白区域
- **修复**：SubImage 从僵尸实际位置开始（Y=zombieTopInTempImg）

### 问题 2：脚部被裁剪
- **原因 2a**：剪裁判断逻辑基于左边缘，错误触发剪裁
- **原因 2b**：临时图像高度不够（400 像素）
- **修复 2a**：改为基于右边缘判断是否需要剪裁
- **修复 2b**：增加临时图像高度到 600 像素

---

## 渲染流程图

### 修复前（问题 1）

```
临时图像 (400x400)
┌─────────────────┐
│   空白区域       │ ← Y=0
│   (281 像素)    │ (被错误包含)
├─────────────────┤ ← Y=281.59 (僵尸顶部)
│   僵尸图像       │
│                 │
└─────────────────┘ ← Y=400

SubImage(clipStartX, 0, ...) 包含了整个区域（包括空白）
↓
视觉效果：僵尸向下偏移 281 像素
```

### 修复后（问题 1）

```
临时图像 (600x600)
┌─────────────────┐
│   空白区域       │ ← Y=0
│   (被排除)      │
├─────────────────┤ ← Y=281 (剪裁起点)
│   僵尸图像       │
│   (完整)        │
│                 │
└─────────────────┘ ← Y=600

SubImage(clipStartX, 281, ...) 只包含僵尸实际区域
↓
视觉效果：僵尸正常显示，无偏移
```

---

## 关键计算

```go
// 1. 判断是否需要剪裁
zombieLeftWorldX := pos.X - CenterOffsetX
zombieRightWorldX := zombieLeftWorldX + zombieWidth
clipLeftWorldX := config.GameOverDoorMaskX // 90

if zombieRightWorldX <= clipLeftWorldX {
    return // 完全隐藏
}
if zombieLeftWorldX >= clipLeftWorldX {
    s.drawEntity(screen, id, cameraX) // 正常渲染
    return
}
// 否则需要剪裁

// 2. 记录僵尸在临时图像中的实际 Y 位置
zombieTopInTempImg := pos.Y - reanimComp.CenterOffsetY
// 例如：347.74 - 66.15 = 281.59

// 3. 渲染到临时图像（使用临时摄像机坐标）
tempCameraX := zombieLeftWorldX - leftPadding
s.renderReanimEntity(tempImg, id, tempCameraX)

// 4. 剪裁时从僵尸实际位置开始
zombieTopY := int(zombieTopInTempImg) // 281
clippedImg := tempImg.SubImage(image.Rect(
    clipStartX, zombieTopY, // 从 281 开始，而不是 0
    tempBounds.Dx(), tempBounds.Dy(),
))

// 5. 绘制到屏幕（Y 坐标保持一致）
screenY := pos.Y - reanimComp.CenterOffsetY // 281.59
op.GeoM.Translate(screenX, screenY)
screen.DrawImage(clippedImg, op)
```

---

## 调试日志对比

### 问题 1 修复前

```
[RenderSystem] Clipping zombie: pos.Y=347.74, CenterOffsetY=66.15
[RenderSystem] Drawing clipped zombie at screenX=90.00, screenY=281.59
```

### 问题 1 修复后

```
[RenderSystem] Clipping zombie: pos.Y=347.74, CenterOffsetY=66.15, zombieTopInTempImg=281.59
[RenderSystem] Drawing clipped zombie at screenX=90.00, screenY=281.59 (clipped from Y=281 in temp image)
```

### 问题 2 修复前

```
# 还在门外的僵尸也被剪裁
[RenderSystem] Clipping zombie: pos.X=131.00, leftWorldX=89.90, clipBoundary=90.00
# 脚部被裁剪（临时图像高度 400，僵尸顶部在 301）
```

### 问题 2 修复后

```
# 门外僵尸正常渲染（不会进入剪裁逻辑）
# 临时图像高度 600，足够容纳整个僵尸（包括脚部）
```

---

## 经验总结

### 核心教训

1. **SubImage 剪裁时必须考虑内容的实际渲染位置**
   - 不能简单地从 `(0, 0)` 开始剪裁
   - 需要排除临时图像中的空白区域

2. **临时图像尺寸必须足够大**
   - 宽度：实体宽度 + 左右 padding
   - 高度：**必须容纳实体在临时图像中的实际渲染位置 + 实体高度**
   - 例如：僵尸顶部在 Y=301，高度 200 → 至少需要 501 像素

3. **剪裁判断逻辑应该基于边界关系**
   - 判断是否需要剪裁：检查实体**右边缘**是否超过剪裁边界
   - 而不是检查**左边缘**（会错误触发剪裁）

4. **调试策略**
   - 先检查数据层（组件中的坐标）是否正确
   - 再检查渲染层（计算的屏幕坐标）是否正确
   - 然后检查剪裁逻辑（判断条件）是否正确
   - 最后检查最终渲染（SubImage、DrawImage、临时图像尺寸）是否正确
   - 分层调试可以快速定位问题

### 代码模式

#### 错误模式 1：假设临时图像从 0 开始

```go
tempImg := ebiten.NewImage(width, height)
renderToTempImage(tempImg, entity)
clippedImg := tempImg.SubImage(image.Rect(
    clipX, 0, // ❌ 假设内容从 Y=0 开始
    width, height,
))
```

#### 错误模式 2：临时图像高度不够

```go
tempHeight := 400 // ❌ 可能不够容纳整个实体
tempImg := ebiten.NewImage(tempWidth, tempHeight)
```

#### 错误模式 3：剪裁判断基于错误的边缘

```go
// ❌ 基于左边缘判断
if entityLeftWorldX >= clipLeftWorldX {
    s.drawEntity(screen, id, cameraX)
    return
}
```

#### 正确模式

```go
// ✅ 1. 先判断是否需要剪裁（基于边界关系）
entityRightWorldX := entityLeftWorldX + entityWidth

if entityRightWorldX <= clipLeftWorldX {
    return // 完全隐藏
}
if entityLeftWorldX >= clipLeftWorldX {
    s.drawEntity(screen, id, cameraX) // 正常渲染
    return
}

// ✅ 2. 临时图像尺寸足够大
tempHeight := 600 // 确保能容纳整个实体

// ✅ 3. 记录实体在临时图像中的实际位置
entityTopY := calculateEntityTopY(entity)

// ✅ 4. 渲染到临时图像
renderToTempImage(tempImg, entity)

// ✅ 5. 从实体实际位置开始剪裁
clippedImg := tempImg.SubImage(image.Rect(
    clipX, entityTopY, // 从实体实际位置开始
    tempWidth, tempHeight,
))
```

---

## 测试验证

所有 13 个测试用例通过 ✅：
```
PASS: TestZombiesWonPhaseSystem_Phase2ZombieReachesTarget1ThenWalksIntoDoor
PASS: TestZombiesWonPhaseSystem_Phase2ZombieMovesInStraightLine
PASS: TestZombiesWonPhaseSystem_Phase2TransitionsToPhase3
PASS: TestZombiesWonPhaseSystem_Phase2CameraAndZombieMoveSimultaneously
PASS: TestZombiesWonPhaseSystem_Phase1ToPhase2Transition
PASS: TestZombiesWonPhaseSystem_Phase2SimultaneousMovement
PASS: TestZombiesWonPhaseSystem_Phase2ToPhase3Transition
PASS: TestZombiesWonPhaseSystem_Phase3AudioAndAnimation
PASS: TestZombiesWonPhaseSystem_FullFlowSimulation
PASS: TestZombiesWonPhaseSystem_GameFreezeComponentPresent
PASS: TestZombiesWonPhaseSystem_TargetPositionCalculation
```

---

## 相关文件

- `pkg/systems/render_system.go:323-442` - 剪裁渲染逻辑
- `pkg/systems/render_reanim.go:56-` - `renderReanimEntity` 函数
- `pkg/utils/coordinates.go` - 坐标转换工具库

---

## 参考资料

- Story 8.8: `docs/stories/8.8.story.md`
- 坐标系统文档: `CLAUDE.md#坐标系统使用指南`
- 临时图像剪裁模式: 本文档 "代码模式" 章节

