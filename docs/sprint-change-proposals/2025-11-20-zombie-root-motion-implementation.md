# Sprint Change Proposal - 僵尸移动系统优化（根运动法实现）

**文档类型**: Sprint Change Proposal
**创建日期**: 2025-11-20
**创建人**: Bob (Scrum Master)
**状态**: ✅ 已批准
**优先级**: 中
**预估工作量**: 6-10 小时（1-2 个工作日）

---

## 执行摘要

**变更触发器**: 根据 `.meta/reanim/僵尸移动说明.md` 文档，项目需要采用**根运动（Root Motion）法**修正僵尸移动动画效果，以解决当前"滑步"问题。

**核心问题**: 当前实现使用**固定速度法（方案 A）**，僵尸移动速度通过硬编码的 `VelocityComponent.VX = -150.0` 控制，与 Reanim 动画中 `_ground` 轨道的 X 位移数据不同步，导致僵尸脚步与地面不匹配（滑步现象）。

**建议方案**: 实施**根运动（Root Motion）法（方案 B）**，直接利用 Reanim 动画中的 `_ground` 轨道数据驱动僵尸位移，确保脚步与地面完美同步。

**影响范围**:
- 系统修改: `PhysicsSystem` 或 `BehaviorSystem`（僵尸移动逻辑）
- 新增工具: `pkg/utils/root_motion.go`（根运动计算）
- 组件扩展: `ReanimComponent`（添加 `LastGroundX/Y` 字段）
- 无 PRD 或架构冲突

---

## 1. 变更上下文分析（Change Context）

### 1.1 触发问题

**问题来源**: `.meta/reanim/僵尸移动说明.md`

**问题描述**:
> XML 动画定义文件只负责"原地动作"（即肢体相对于僵尸中心点的摆动），它本身不会修改僵尸在游戏地图（世界坐标系）中的实际 X/Y 坐标。如果只播放动画而不移动僵尸的游戏对象坐标，僵尸看起来就像是在跑步机上原地走路（出现"滑步"现象）。

**当前实现（方案 A - 固定速度法）**:
```go
// pkg/systems/wave_spawn_system.go:205
vel.VX = -150.0 // 僵尸标准移动速度（硬编码）

// pkg/systems/behavior/zombie_behavior_handler.go:69-71
position.X += velocity.VX * deltaTime
position.Y += velocity.VY * deltaTime
```

**问题本质**:
1. 僵尸速度与动画播放速度独立计算
2. 未利用 Reanim 文件中的 `_ground` 轨道位移数据
3. 速度需要手动调整以匹配动画（维护成本高）

### 1.2 问题分类

- ✅ **技术优化** - 利用现有数据提升视觉表现质量
- ❌ 非技术限制/死胡同
- ❌ 非新需求
- ❌ 非需求误解

### 1.3 初步影响

**视觉表现**:
- 僵尸脚步与地面不完美同步
- 与原版 PVZ 的精细动画效果存在差距

**技术债务**:
- 未充分利用原版 Reanim 资源的设计意图
- 每次调整僵尸速度需要手动测试和调优

**用户体验**:
- 降低游戏视觉质量和沉浸感

### 1.4 证据

**文档证据**:
- `.meta/reanim/僵尸移动说明.md` 详细说明了方案 B（根运动法）的优势

**代码证据**:
- `pkg/systems/wave_spawn_system.go:205` - 硬编码速度 `-150.0`
- `pkg/systems/behavior/zombie_behavior_handler.go:69-71` - 固定速度法实现

**原版设计证据**:
- 原版 PVZ 的 Reanim 文件中 `_ground` 轨道专门用于指导位移计算
- 该轨道的作用是告诉开发者："如果想要不滑步，这个僵尸在一个动画循环内应该移动这么多距离"

---

## 2. Epic 影响评估（Epic Impact Analysis）

### 2.1 当前 Epic 分析

**受影响的 Epic**: ✅ 无（所有僵尸相关 Epic 已完成）

**Epic 状态**:
- Epic 4: 基础僵尸与战斗逻辑 - ✅ 已完成
- Epic 5: 游戏流程与高级单位 - ✅ 已完成
- Epic 6/13: Reanim 动画系统 - ✅ 已完成

**当前 Epic 修改需求**: 无需修改已完成的 Epic

### 2.2 未来 Epic 分析

**潜在影响**:
- 如果未来有新的僵尸类型或动画扩展，根运动系统将自动支持（无需调整代码）
- 符合"数据驱动"设计原则，降低维护成本

**依赖关系变化**: ✅ 无

### 2.3 Epic 影响总结

✅ **无需创建新 Epic** - 属于现有动画系统（Epic 6/13）的优化和完善

---

## 3. 项目文档冲突分析（Artifact Conflict Analysis）

### 3.1 PRD 冲突检查

**冲突评估**: ✅ **无冲突，反而增强 PRD 目标达成**

**相关需求**:
```yaml
NFR2: 忠实度
  描述: 所有的游戏数值（如植物攻击力、僵尸生命值、阳光值、冷却时间）和行为节奏都应与原版PC游戏保持高度一致。
```

**分析**: 根运动法更符合原版设计意图，提升动画忠实度，强化 NFR2 的达成。

### 3.2 架构文档冲突检查

**冲突评估**: ✅ **无冲突，符合现有架构**

**架构一致性**:
- ✅ 符合 ECS 架构原则（组件数据驱动）
- ✅ 利用现有 `ReanimComponent` 数据，无需新组件类型
- ✅ 变更仅涉及系统层（`PhysicsSystem` 或 `BehaviorSystem`）
- ✅ 遵循"数据与行为分离"原则

**系统职责**:
- `BehaviorSystem` - 负责僵尸行为逻辑（移动、啃食、死亡）
- `ReanimSystem` - 负责动画播放和数据管理
- `PhysicsSystem` - 负责物理更新（可选位置）

### 3.3 前端规范冲突检查

**冲突评估**: ✅ N/A（纯后端逻辑变更）

### 3.4 其他文档冲突检查

**冲突评估**: ✅ 无

**需要更新的文档**:

| 文档 | 更新内容 | 优先级 | 预估时间 |
|------|---------|--------|---------|
| `CLAUDE.md` | 添加"根运动系统"章节说明 | 中 | 30 分钟 |
| `docs/architecture/coordinate-system.md` | 补充僵尸移动机制说明 | 低 | 15 分钟 |

### 3.5 文档冲突总结

✅ **无冲突** - 所有变更与现有文档和架构完全兼容

---

## 4. 前进路径评估（Path Forward Evaluation）

### 选项 1: 直接实现根运动法（推荐）✅

**描述**: 在现有代码基础上实现根运动位移计算

#### 实施方案

**1. 新增工具函数** - `pkg/utils/root_motion.go`

```go
package utils

import (
    "github.com/gonewx/pvz/pkg/components"
    "github.com/gonewx/pvz/pkg/reanim"
)

// CalculateRootMotionDelta 计算根运动位移增量
//
// 从 Reanim 动画的 _ground 轨道读取当前帧与上一帧的位移差值
//
// 参数:
//   - reanimComp: Reanim 组件（包含动画数据和当前帧信息）
//   - groundTrack: _ground 轨道数据
//
// 返回:
//   - deltaX: X 轴位移增量（世界坐标单位）
//   - deltaY: Y 轴位移增量
//
// 注意: 当动画循环重置时（从最后一帧跳回第一帧），自动检测并返回 0（避免瞬移）
func CalculateRootMotionDelta(
    reanimComp *components.ReanimComponent,
    groundTrack *reanim.Track,
) (deltaX, deltaY float64) {
    // 获取当前帧索引
    currentFrame := reanimComp.CurrentFrame

    // 获取当前帧的 _ground 轨道位置
    currentGroundX, currentGroundY := getGroundPosition(groundTrack, currentFrame)

    // 计算位移增量
    deltaX = currentGroundX - reanimComp.LastGroundX
    deltaY = currentGroundY - reanimComp.LastGroundY

    // 检测动画循环重置（瞬移检测）
    // 如果位移过大（例如 > 100 像素），认为是循环重置，返回 0
    if abs(deltaX) > 100 || abs(deltaY) > 100 {
        deltaX, deltaY = 0, 0
    }

    // 更新 LastGroundX/Y
    reanimComp.LastGroundX = currentGroundX
    reanimComp.LastGroundY = currentGroundY

    return deltaX, deltaY
}

// getGroundPosition 获取指定帧的 _ground 轨道位置
func getGroundPosition(track *reanim.Track, frameIndex int) (x, y float64) {
    // 实现细节：从 Track.Frames 中获取帧数据
    // 处理空帧继承（Reanim 特性）
}
```

**2. 修改系统** - `pkg/systems/behavior/zombie_behavior_handler.go`

**变更位置**: `handleZombieBasicBehavior` 函数（第 69-71 行）

**修改前**:
```go
// 更新位置：根据速度和时间增量移动僵尸
position.X += velocity.VX * deltaTime
position.Y += velocity.VY * deltaTime
```

**修改后**:
```go
// 使用根运动（Root Motion）计算位移
if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
    // 尝试获取 _ground 轨道
    groundTrack := getGroundTrack(reanim) // 辅助函数，从 reanim 数据中获取

    if groundTrack != nil {
        // 使用根运动法
        deltaX, deltaY := utils.CalculateRootMotionDelta(reanim, groundTrack)
        position.X += deltaX
        position.Y += deltaY

        if s.verbose {
            log.Printf("[RootMotion] Zombie %d moved by root motion: deltaX=%.2f, deltaY=%.2f",
                entityID, deltaX, deltaY)
        }
    } else {
        // 后备方案：如果没有 _ground 轨道，使用固定速度
        position.X += velocity.VX * deltaTime
        position.Y += velocity.VY * deltaTime
    }
} else {
    // 后备方案：没有 Reanim 组件时使用固定速度
    position.X += velocity.VX * deltaTime
    position.Y += velocity.VY * deltaTime
}
```

**3. 组件扩展** - `pkg/components/reanim_component.go`

**添加字段**（在 `ReanimComponent` 结构体中）:
```go
// 根运动（Root Motion）相关字段
LastGroundX float64 // 上一帧 _ground 轨道的 X 坐标
LastGroundY float64 // 上一帧 _ground 轨道的 Y 坐标
```

**初始化**（在僵尸工厂函数中）:
```go
// pkg/entities/zombie_factory.go
reanimComp.LastGroundX = 0.0
reanimComp.LastGroundY = 0.0
```

#### 优势

- ✅ **完美解决滑步问题** - 脚步与地面绝对锁定
- ✅ **完全利用原版数据** - 符合原版设计意图
- ✅ **无需手动调整参数** - 数据驱动，自动适配
- ✅ **自动支持所有僵尸类型** - 包括未来新增的僵尸

#### 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|---------|
| `_ground` 轨道数据缺失 | 低 | 中 | 实现后备方案（固定速度法） |
| 动画循环边界处理错误 | 中 | 中 | 充分测试边界情况，添加日志 |
| 性能下降 | 低 | 高 | 性能基准测试，必要时优化 |

#### 预估工作量

**总计**: 6-10 小时

**详细拆解**:
- Phase 1: 设计与原型（2-3 小时）
- Phase 2: 核心实现（3-4 小时）
- Phase 3: 测试与优化（2-3 小时）
- Phase 4: 文档更新（1 小时）

---

### 选项 2: 改进固定速度法（向后兼容）

**描述**: 保持固定速度法，但根据 `_ground` 轨道数据自动计算最佳速度

#### 实施方案

1. 在僵尸工厂函数中分析 `_ground` 轨道
2. 计算动画循环的平均速度：`speed = totalDistance / cycleDuration`
3. 将计算结果写入 `VelocityComponent`

#### 优势

- ✅ 最小化代码变更
- ✅ 保持现有架构

#### 劣势

- ❌ **仍无法完美同步** - 动画播放速度受 FPS 波动影响
- ❌ **未完全利用原版设计** - 只利用了静态数据（平均速度）

#### 预估工作量

3-5 小时

---

### 选项 3: 混合方案（精准+性能平衡）

**描述**: 使用根运动法计算位移，但保留 `VelocityComponent` 作为缓存

#### 实施方案

1. 根运动系统计算帧间位移
2. 将结果同步到 `VelocityComponent.VX`（用于碰撞检测等）

#### 优势

- ✅ 精准的脚步同步
- ✅ 保持与现有系统的兼容性（碰撞检测等依赖 `VelocityComponent`）

#### 劣势

- ⚠️ 复杂度增加

#### 预估工作量

8-12 小时

---

### 🏆 推荐路径: **选项 1 - 直接实现根运动法**

#### 推荐理由

1. **技术正确性**: 完全符合原版设计意图
2. **可维护性**: 无需手动调整参数，数据驱动
3. **扩展性**: 自动支持所有现有和未来的僵尸动画
4. **工作量合理**: 6-10 小时（约 1-2 个工作日）
5. **风险可控**: 有明确的后备方案和测试计划

---

## 5. Sprint Change Proposal 组件（详细变更清单）

### 5.1 文件变更清单

#### 新增文件

| 文件路径 | 描述 | 代码行数（预估） |
|---------|-----|--------------|
| `pkg/utils/root_motion.go` | 根运动计算工具函数 | 80-100 行 |
| `pkg/utils/root_motion_test.go` | 根运动单元测试 | 120-150 行 |

#### 修改文件

| 文件路径 | 修改描述 | 修改行数（预估） |
|---------|---------|--------------|
| `pkg/systems/behavior/zombie_behavior_handler.go` | 应用根运动位移计算 | +15 行 / -3 行 |
| `pkg/components/reanim_component.go` | 添加 `LastGroundX/Y` 字段 | +3 行 |
| `pkg/entities/zombie_factory.go` | 初始化 `LastGroundX/Y` | +2 行（每个工厂函数） |
| `CLAUDE.md` | 添加根运动系统说明 | +50 行 |
| `docs/architecture/coordinate-system.md` | 补充僵尸移动机制 | +30 行 |

#### 删除文件

✅ 无

---

### 5.2 具体代码变更

#### **pkg/utils/root_motion.go**（新增）

```go
package utils

import (
	"log"
	"math"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/reanim"
)

// CalculateRootMotionDelta 计算根运动位移增量
//
// 从 Reanim 动画的 _ground 轨道读取当前帧与上一帧的位移差值
//
// 工作原理:
//   1. 获取当前帧的 _ground 轨道 X/Y 坐标
//   2. 与上一帧的坐标进行对比，计算增量
//   3. 检测动画循环重置（防止瞬移）
//   4. 更新 LastGroundX/Y 用于下一次计算
//
// 参数:
//   - reanimComp: Reanim 组件（包含动画数据和当前帧信息）
//   - groundTrackName: _ground 轨道名称（通常为 "_ground"）
//
// 返回:
//   - deltaX: X 轴位移增量（世界坐标单位）
//   - deltaY: Y 轴位移增量
//   - error: 如果轨道不存在或数据无效返回错误
//
// 注意:
//   - 当动画循环重置时（从最后一帧跳回第一帧），自动检测并返回 0（避免瞬移）
//   - 调用方需要在 ReanimComponent 初始化时设置 LastGroundX/Y = 0
func CalculateRootMotionDelta(
	reanimComp *components.ReanimComponent,
	groundTrackName string,
) (deltaX, deltaY float64, err error) {
	// 验证参数
	if reanimComp == nil {
		return 0, 0, fmt.Errorf("reanimComp cannot be nil")
	}

	// 获取 _ground 轨道数据
	groundTrack := reanimComp.GetTrack(groundTrackName)
	if groundTrack == nil {
		return 0, 0, fmt.Errorf("ground track '%s' not found", groundTrackName)
	}

	// 获取当前动画的物理帧索引
	// 注意：需要使用 CurrentPhysicalFrames map，因为多个动画可能同时播放
	currentAnimName := reanimComp.GetPrimaryAnimation() // 主动画（如 anim_walk）
	physicalFrame, ok := reanimComp.CurrentPhysicalFrames[currentAnimName]
	if !ok {
		return 0, 0, fmt.Errorf("no physical frame for animation '%s'", currentAnimName)
	}

	// 获取当前帧的 _ground 轨道位置
	currentGroundX, currentGroundY := getGroundPosition(groundTrack, physicalFrame)

	// 计算位移增量
	deltaX = currentGroundX - reanimComp.LastGroundX
	deltaY = currentGroundY - reanimComp.LastGroundY

	// 检测动画循环重置（瞬移检测）
	// 如果位移过大（例如 > 100 像素），认为是循环重置，返回 0
	const MAX_DELTA = 100.0
	if math.Abs(deltaX) > MAX_DELTA || math.Abs(deltaY) > MAX_DELTA {
		log.Printf("[RootMotion] Loop reset detected: deltaX=%.2f, deltaY=%.2f -> resetting to 0", deltaX, deltaY)
		deltaX, deltaY = 0, 0
	}

	// 更新 LastGroundX/Y 用于下一次计算
	reanimComp.LastGroundX = currentGroundX
	reanimComp.LastGroundY = currentGroundY

	return deltaX, deltaY, nil
}

// getGroundPosition 获取指定帧的 _ground 轨道位置
//
// 参数:
//   - track: _ground 轨道数据
//   - frameIndex: 物理帧索引
//
// 返回:
//   - x: X 坐标
//   - y: Y 坐标
//
// 注意: 处理空帧继承（Reanim 特性）
func getGroundPosition(track *reanim.Track, frameIndex int) (x, y float64) {
	if track == nil || len(track.Frames) == 0 {
		return 0, 0
	}

	// 边界检查
	if frameIndex < 0 {
		frameIndex = 0
	}
	if frameIndex >= len(track.Frames) {
		frameIndex = len(track.Frames) - 1
	}

	// 获取帧数据
	frame := track.Frames[frameIndex]

	// Reanim 空帧继承：如果当前帧的 X/Y 为 0，向前查找最近的非空帧
	if frame.X == 0 && frame.Y == 0 && frameIndex > 0 {
		for i := frameIndex - 1; i >= 0; i-- {
			if track.Frames[i].X != 0 || track.Frames[i].Y != 0 {
				return track.Frames[i].X, track.Frames[i].Y
			}
		}
	}

	return frame.X, frame.Y
}

// GetPrimaryAnimation 获取主动画名称（辅助函数）
//
// 返回当前播放的主要动画（通常是第一个动画）
func (rc *components.ReanimComponent) GetPrimaryAnimation() string {
	if len(rc.CurrentAnimations) == 0 {
		return ""
	}
	return rc.CurrentAnimations[0]
}

// GetTrack 获取指定轨道（辅助函数）
//
// 注意：需要在 ReanimComponent 中实现此方法
func (rc *components.ReanimComponent) GetTrack(trackName string) *reanim.Track {
	// 实现逻辑：从 ReanimData 中查找 trackName
	// 返回 Track 数据结构
}
```

---

#### **pkg/components/reanim_component.go**（修改）

```go
// ReanimComponent Reanim 动画组件
type ReanimComponent struct {
	// ... 现有字段 ...

	// 根运动（Root Motion）相关字段
	LastGroundX float64 // 上一帧 _ground 轨道的 X 坐标（用于计算帧间增量）
	LastGroundY float64 // 上一帧 _ground 轨道的 Y 坐标
}
```

---

#### **pkg/entities/zombie_factory.go**（修改）

```go
// NewZombieEntity 创建普通僵尸实体
func NewZombieEntity(...) (ecs.EntityID, error) {
	// ... 现有代码 ...

	// 创建 Reanim 组件
	reanimComp := &components.ReanimComponent{
		// ... 现有字段 ...

		// 初始化根运动字段
		LastGroundX: 0.0,
		LastGroundY: 0.0,
	}
	em.AddComponent(entityID, reanimComp)

	// ... 后续代码 ...
}
```

**说明**: 同样的修改应用于 `NewConeheadZombieEntity` 和 `NewBucketheadZombieEntity`

---

#### **pkg/systems/behavior/zombie_behavior_handler.go**（修改）

**修改位置**: `handleZombieBasicBehavior` 函数（第 69-71 行）

**修改前**:
```go
// 更新位置：根据速度和时间增量移动僵尸
position.X += velocity.VX * deltaTime
position.Y += velocity.VY * deltaTime
```

**修改后**:
```go
// 使用根运动（Root Motion）计算位移
reanim, hasReanim := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
if hasReanim {
	// 尝试使用根运动法
	deltaX, deltaY, err := utils.CalculateRootMotionDelta(reanim, "_ground")

	if err == nil {
		// 根运动成功：应用位移增量
		position.X += deltaX
		position.Y += deltaY

		// DEBUG 日志（可选，通过 verbose 标志控制）
		if s.verbose {
			log.Printf("[RootMotion] Zombie %d moved by root motion: deltaX=%.2f, deltaY=%.2f",
				entityID, deltaX, deltaY)
		}
	} else {
		// 根运动失败（例如 _ground 轨道不存在）：回退到固定速度法
		log.Printf("[RootMotion] WARNING: Root motion failed for zombie %d: %v, falling back to fixed velocity",
			entityID, err)
		position.X += velocity.VX * deltaTime
		position.Y += velocity.VY * deltaTime
	}
} else {
	// 后备方案：没有 Reanim 组件时使用固定速度
	position.X += velocity.VX * deltaTime
	position.Y += velocity.VY * deltaTime
}
```

---

#### **pkg/utils/root_motion_test.go**（新增）

```go
package utils

import (
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/reanim"
)

// TestCalculateRootMotionDelta_NormalMovement 测试正常帧间位移
func TestCalculateRootMotionDelta_NormalMovement(t *testing.T) {
	// 创建测试用的 ReanimComponent
	reanimComp := &components.ReanimComponent{
		LastGroundX: 10.0,
		LastGroundY: 20.0,
		CurrentPhysicalFrames: map[string]int{
			"anim_walk": 5,
		},
		CurrentAnimations: []string{"anim_walk"},
	}

	// 创建测试用的 _ground 轨道
	groundTrack := &reanim.Track{
		Name: "_ground",
		Frames: []reanim.Frame{
			{X: 0.0, Y: 0.0},   // Frame 0
			{X: 5.0, Y: 0.0},   // Frame 1
			{X: 10.0, Y: 0.0},  // Frame 2
			{X: 15.0, Y: 0.0},  // Frame 3
			{X: 20.0, Y: 0.0},  // Frame 4
			{X: 25.0, Y: 0.0},  // Frame 5（当前帧）
		},
	}

	// Mock GetTrack 方法
	// （需要在 ReanimComponent 中实现）

	// 执行计算
	deltaX, deltaY, err := CalculateRootMotionDelta(reanimComp, "_ground")

	// 验证结果
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedDeltaX := 25.0 - 10.0 // 15.0
	expectedDeltaY := 0.0 - 20.0  // -20.0

	if deltaX != expectedDeltaX {
		t.Errorf("Expected deltaX=%.2f, got %.2f", expectedDeltaX, deltaX)
	}
	if deltaY != expectedDeltaY {
		t.Errorf("Expected deltaY=%.2f, got %.2f", expectedDeltaY, deltaY)
	}

	// 验证 LastGroundX/Y 已更新
	if reanimComp.LastGroundX != 25.0 {
		t.Errorf("Expected LastGroundX=25.0, got %.2f", reanimComp.LastGroundX)
	}
	if reanimComp.LastGroundY != 0.0 {
		t.Errorf("Expected LastGroundY=0.0, got %.2f", reanimComp.LastGroundY)
	}
}

// TestCalculateRootMotionDelta_LoopReset 测试动画循环重置（防瞬移）
func TestCalculateRootMotionDelta_LoopReset(t *testing.T) {
	// 模拟动画从最后一帧（X=200）跳回第一帧（X=0）
	reanimComp := &components.ReanimComponent{
		LastGroundX: 200.0, // 上一帧在动画末尾
		LastGroundY: 0.0,
		CurrentPhysicalFrames: map[string]int{
			"anim_walk": 0, // 当前帧在动画开头
		},
		CurrentAnimations: []string{"anim_walk"},
	}

	groundTrack := &reanim.Track{
		Name: "_ground",
		Frames: []reanim.Frame{
			{X: 0.0, Y: 0.0}, // Frame 0（当前帧）
		},
	}

	// 执行计算
	deltaX, deltaY, err := CalculateRootMotionDelta(reanimComp, "_ground")

	// 验证结果：应该返回 0（检测到瞬移）
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if deltaX != 0.0 {
		t.Errorf("Expected deltaX=0.0 (loop reset), got %.2f", deltaX)
	}
	if deltaY != 0.0 {
		t.Errorf("Expected deltaY=0.0 (loop reset), got %.2f", deltaY)
	}
}

// TestCalculateRootMotionDelta_MissingTrack 测试 _ground 轨道不存在
func TestCalculateRootMotionDelta_MissingTrack(t *testing.T) {
	reanimComp := &components.ReanimComponent{
		// ... 配置 ...
	}

	// 执行计算（轨道不存在）
	deltaX, deltaY, err := CalculateRootMotionDelta(reanimComp, "_ground")

	// 验证结果：应该返回错误
	if err == nil {
		t.Fatal("Expected error when track is missing, got nil")
	}

	if deltaX != 0.0 || deltaY != 0.0 {
		t.Errorf("Expected zero delta on error, got deltaX=%.2f, deltaY=%.2f", deltaX, deltaY)
	}
}
```

---

### 5.3 配置变更

✅ **无需配置变更** - 所有数据已存在于 Reanim 文件中

---

### 5.4 数据库变更

✅ N/A（无数据库）

---

### 5.5 部署变更

✅ **无部署变更** - 纯代码逻辑优化

---

## 6. PRD MVP 影响分析

### 6.1 MVP 范围变化

**MVP 范围**: ✅ 无变化

**MVP 核心目标**:
- FR9.1: 僵尸沿固定行从右向左移动 - ✅ 保持不变（实现方式优化）
- NFR2: 忠实度 - ✅ **增强**（动画更符合原版标准）

### 6.2 核心目标影响

✅ **正面影响** - 提升 NFR2（忠实度）的达成度

### 6.3 功能增减

**新增功能**:
- 根运动系统（技术实现层，用户无感知）

**删除功能**:
- 无

**修改功能**:
- 僵尸移动逻辑（从固定速度法优化为根运动法）

### 6.4 MVP 影响总结

✅ **无负面影响，反而增强 MVP 质量**

---

## 7. 高层行动计划（High-Level Action Plan）

### Phase 1: 设计与原型（2-3 小时）

**任务**:
1. 详细设计 `CalculateRootMotionDelta` 函数逻辑
2. 创建原型测试脚本（读取 `Zombie.reanim` 的 `_ground` 轨道）
3. 验证数据格式和边界情况（动画循环重置、空帧继承）

**交付物**:
- 技术设计文档（Markdown）
- 原型代码（可运行的测试脚本）
- _ground 轨道数据分析报告

**验收标准**:
- ✅ 能成功读取 `Zombie.reanim` 的 `_ground` 轨道
- ✅ 能正确计算帧间位移增量
- ✅ 能检测动画循环重置

---

### Phase 2: 核心实现（3-4 小时）

**任务**:
1. 实现 `pkg/utils/root_motion.go`
2. 修改 `pkg/systems/behavior/zombie_behavior_handler.go`
3. 扩展 `pkg/components/reanim_component.go`
4. 修改僵尸工厂函数（初始化 `LastGroundX/Y`）

**交付物**:
- 完整的根运动工具函数
- 集成到僵尸行为系统

**验收标准**:
- ✅ 代码编译通过
- ✅ 符合项目编码规范（`gofmt`, `golint`）
- ✅ 包含详细的代码注释（GoDoc 格式）

---

### Phase 3: 测试与优化（2-3 小时）

**任务**:
1. 编写单元测试（`pkg/utils/root_motion_test.go`）
   - 测试用例 1: 正常帧间位移
   - 测试用例 2: 动画循环重置
   - 测试用例 3: _ground 轨道不存在
   - 测试用例 4: 空帧继承
2. 集成测试（关卡 1-1 验证僵尸移动）
3. 性能分析（确保无性能回归）
4. 视觉验证（观察僵尸脚步是否与地面同步）

**交付物**:
- 单元测试套件（覆盖率 > 80%）
- 集成测试报告
- 性能基准测试报告

**验收标准**:
- ✅ 所有单元测试通过
- ✅ 僵尸移动无滑步现象
- ✅ 性能无明显下降（< 5%）
- ✅ 关卡 1-1 至 1-10 正常运行

---

### Phase 4: 文档更新（1 小时）

**任务**:
1. 更新 `CLAUDE.md`（添加根运动系统说明）
2. 更新 `docs/architecture/coordinate-system.md`（补充僵尸移动机制）
3. 编写变更日志（CHANGELOG.md）

**交付物**:
- 更新后的文档

**验收标准**:
- ✅ 文档清晰易懂
- ✅ 包含代码示例
- ✅ 符合项目文档规范

---

## 8. Agent 协作计划（Agent Handoff Plan）

| Agent 角色 | 职责 | 阶段 | 交付物 |
|-----------|-----|------|--------|
| **SM (Scrum Master)** | 创建 Sprint Change Proposal | ✅ 完成 | 本文档 |
| **Dev (开发者)** | 实现根运动系统 | Phase 1-3 | 代码 + 测试 |
| **QA (测试)** | 验证动画同步效果 | Phase 3 | 测试报告 |
| **SM (Scrum Master)** | 跟踪进度，协调资源 | 全程 | 进度报告 |

### Handoff 流程

1. **SM → Dev**:
   - 提供 Sprint Change Proposal（本文档）
   - 提供 `.meta/reanim/僵尸移动说明.md` 技术参考
   - 明确验收标准

2. **Dev → QA**:
   - 提供完整代码实现
   - 提供单元测试套件
   - 提供集成测试指南

3. **QA → SM**:
   - 提供测试报告
   - 标记发现的问题

4. **SM → PM/PO** (可选):
   - 如果需要调整 PRD 或架构文档
   - 提供变更建议

---

## 9. 成功标准（Success Criteria）

### 9.1 技术成功标准

1. ✅ **动画同步**: 僵尸脚步与地面完美同步（无滑步现象）
2. ✅ **兼容性**: 所有僵尸类型（basic, conehead, buckethead）正常工作
3. ✅ **性能**: 性能无回归（60 FPS 稳定，帧时间 < 16.67ms）
4. ✅ **测试覆盖率**: 单元测试覆盖率 > 80%
5. ✅ **集成测试**: 关卡 1-1 至 1-10 正常运行
6. ✅ **代码质量**: 符合项目编码规范（`gofmt`, `golint`）
7. ✅ **文档完整**: CLAUDE.md 和架构文档已更新

### 9.2 用户体验成功标准

1. ✅ **视觉质量**: 僵尸移动动画流畅自然
2. ✅ **原版忠实度**: 符合原版 PVZ 的动画表现
3. ✅ **无感知变更**: 玩家无需学习新操作，透明升级

### 9.3 项目成功标准

1. ✅ **零回归**: 现有功能无任何破坏
2. ✅ **可维护性**: 代码清晰易懂，有完整注释
3. ✅ **可扩展性**: 支持未来新增僵尸类型

---

## 10. 风险缓解计划（Risk Mitigation）

### 10.1 风险评估矩阵

| 风险 | 概率 | 影响 | 风险等级 | 缓解措施 |
|------|------|------|---------|---------|
| `_ground` 轨道数据缺失 | 低 (10%) | 中 | 🟡 低 | 实现后备方案（固定速度法） |
| 动画循环边界处理错误 | 中 (30%) | 中 | 🟠 中 | 充分测试边界情况，添加日志 |
| 性能下降 | 低 (10%) | 高 | 🟠 中 | 性能基准测试，必要时优化 |
| 与现有系统冲突 | 低 (5%) | 中 | 🟡 低 | 保留 `VelocityComponent` 作为后备 |
| 空帧继承处理错误 | 中 (20%) | 低 | 🟡 低 | 参考现有 Reanim 系统实现 |

### 10.2 详细缓解措施

#### 风险 1: `_ground` 轨道数据缺失

**场景**: 某些僵尸的 Reanim 文件可能没有 `_ground` 轨道

**缓解措施**:
```go
// 在 CalculateRootMotionDelta 中实现后备方案
if groundTrack == nil {
    return 0, 0, fmt.Errorf("ground track not found")
}

// 在 zombie_behavior_handler.go 中捕获错误
if err != nil {
    // 回退到固定速度法
    position.X += velocity.VX * deltaTime
    position.Y += velocity.VY * deltaTime
}
```

**验证方法**:
- 检查所有僵尸的 Reanim 文件（`Zombie.reanim`, `ZombieConehead.reanim`, `ZombieBuckethead.reanim`）
- 确认 `_ground` 轨道存在

---

#### 风险 2: 动画循环边界处理错误

**场景**: 动画从最后一帧跳回第一帧时，位移增量异常巨大

**缓解措施**:
```go
// 在 CalculateRootMotionDelta 中实现瞬移检测
const MAX_DELTA = 100.0
if math.Abs(deltaX) > MAX_DELTA || math.Abs(deltaY) > MAX_DELTA {
    log.Printf("[RootMotion] Loop reset detected: deltaX=%.2f -> resetting to 0", deltaX)
    deltaX, deltaY = 0, 0
}
```

**验证方法**:
- 单元测试：模拟动画循环重置场景
- 集成测试：观察僵尸移动 5-10 个动画循环

---

#### 风险 3: 性能下降

**场景**: 根运动计算增加 CPU 开销

**缓解措施**:
- 使用性能基准测试（`go test -bench`）
- 如果性能下降 > 5%，考虑缓存优化：
  ```go
  // 缓存 _ground 轨道引用（避免每帧查找）
  if reanimComp.CachedGroundTrack == nil {
      reanimComp.CachedGroundTrack = reanimComp.GetTrack("_ground")
  }
  ```

**验证方法**:
- 性能基准测试：对比根运动法 vs 固定速度法
- 集成测试：关卡 1-10（大量僵尸同屏）保持 60 FPS

---

## 11. 回滚计划（Rollback Plan）

### 11.1 Git 分支策略

**功能分支**: `feature/zombie-root-motion`

**保护分支**: `main`（不允许直接推送）

**合并流程**:
1. 在 `feature/zombie-root-motion` 分支完成开发
2. 创建 Pull Request
3. 通过 Code Review 和 QA 测试
4. 合并到 `main` 分支

### 11.2 回滚触发条件

如果发生以下情况，立即回滚：

| 触发条件 | 严重性 | 回滚方式 |
|---------|-------|---------|
| 性能下降 > 10% | 🔴 高 | 立即回滚 |
| 僵尸移动出现异常（瞬移、静止） | 🔴 高 | 立即回滚 |
| 单元测试失败率 > 20% | 🟠 中 | 修复或回滚 |
| 集成测试失败 | 🟠 中 | 修复或回滚 |

### 11.3 回滚步骤

**方式 1: Git Revert（推荐）**
```bash
# 如果已合并到 main，使用 revert
git revert <commit-hash>
git push origin main
```

**方式 2: 分支切换（紧急情况）**
```bash
# 临时切换回上一个稳定分支
git checkout main
git reset --hard <previous-commit>
git push origin main --force  # 需要管理员权限
```

**方式 3: 代码级回滚（最小影响）**
```go
// 在 zombie_behavior_handler.go 中，临时禁用根运动
const USE_ROOT_MOTION = false  // 设置为 false 即可回退

if USE_ROOT_MOTION {
    // 根运动法
    deltaX, deltaY, err := utils.CalculateRootMotionDelta(...)
    // ...
} else {
    // 固定速度法（回退）
    position.X += velocity.VX * deltaTime
    position.Y += velocity.VY * deltaTime
}
```

### 11.4 回滚后续措施

1. **问题分析**: 分析回滚原因，记录日志
2. **Bug 修复**: 在 `feature/zombie-root-motion` 分支修复问题
3. **重新测试**: 通过所有测试后再次提交 PR
4. **文档更新**: 记录问题和解决方案（Lessons Learned）

---

## 12. 预估时间线（Timeline）

### 12.1 总工作量

**总计**: 6-10 小时（1-2 个工作日）

### 12.2 详细时间拆解

| 阶段 | 任务 | 预估时间 | 负责人 |
|------|-----|---------|-------|
| **Phase 1** | 设计与原型 | 2-3 小时 | Dev |
| - | 设计 `CalculateRootMotionDelta` 逻辑 | 1 小时 | Dev |
| - | 创建原型测试脚本 | 1 小时 | Dev |
| - | 验证数据格式和边界情况 | 0.5-1 小时 | Dev |
| **Phase 2** | 核心实现 | 3-4 小时 | Dev |
| - | 实现 `pkg/utils/root_motion.go` | 1.5-2 小时 | Dev |
| - | 修改 `zombie_behavior_handler.go` | 0.5 小时 | Dev |
| - | 扩展 `ReanimComponent` | 0.5 小时 | Dev |
| - | 修改僵尸工厂函数 | 0.5-1 小时 | Dev |
| **Phase 3** | 测试与优化 | 2-3 小时 | Dev + QA |
| - | 编写单元测试 | 1-1.5 小时 | Dev |
| - | 集成测试 | 0.5-1 小时 | QA |
| - | 性能分析 | 0.5 小时 | Dev |
| **Phase 4** | 文档更新 | 1 小时 | Dev + SM |
| - | 更新 CLAUDE.md | 0.5 小时 | Dev |
| - | 更新架构文档 | 0.5 小时 | Dev/SM |

### 12.3 建议时间点

**开始时间**:
- ✅ **立即开始**（无阻塞依赖）
- 或在下一个 Sprint 的开始阶段完成

**里程碑**:
- **Day 1 上午**: 完成 Phase 1（设计与原型）
- **Day 1 下午**: 完成 Phase 2（核心实现）
- **Day 2 上午**: 完成 Phase 3（测试与优化）
- **Day 2 下午**: 完成 Phase 4（文档更新）

### 12.4 依赖关系

**前置依赖**:
- ✅ Reanim 系统已完成（Epic 6/13）
- ✅ 僵尸移动系统已完成（Epic 4/5）

**并行依赖**:
- ✅ 无（可独立开发）

**后续依赖**:
- ✅ 无（其他功能不依赖此变更）

---

## 13. 最终审查与批准（Final Review & Approval）

### 13.1 检查清单完成情况

✅ **Section 1: 变更上下文分析** - 完成
✅ **Section 2: Epic 影响评估** - 完成
✅ **Section 3: 项目文档冲突分析** - 完成
✅ **Section 4: 前进路径评估** - 完成
✅ **Section 5: Sprint Change Proposal 组件** - 完成
✅ **Section 6: PRD MVP 影响分析** - 完成
✅ **Section 7: 高层行动计划** - 完成
✅ **Section 8: Agent 协作计划** - 完成
✅ **Section 9: 成功标准** - 完成
✅ **Section 10: 风险缓解计划** - 完成
✅ **Section 11: 回滚计划** - 完成
✅ **Section 12: 预估时间线** - 完成

### 13.2 Proposal 质量验证

**准确性**: ✅ 所有分析基于实际代码和文档
**完整性**: ✅ 涵盖所有必要的 Change Checklist 项
**可执行性**: ✅ 提供详细的代码变更和实施计划
**风险评估**: ✅ 识别并缓解主要风险

### 13.3 用户批准状态

**状态**: ✅ **已批准**（2025-11-20）

**批准人**: 用户

**批准备注**:
- 方案技术合理，符合原版设计意图
- 风险可控，有明确的后备方案
- 工作量合理（1-2 个工作日）

---

## 14. 下一步行动（Next Steps）

### 14.1 立即行动

1. **创建功能分支**:
   ```bash
   git checkout -b feature/zombie-root-motion
   ```

2. **交付给 Dev Agent**:
   - 提供本 Sprint Change Proposal
   - 提供 `.meta/reanim/僵尸移动说明.md`
   - 明确验收标准

3. **开始 Phase 1（设计与原型）**

### 14.2 监控指标

| 指标 | 目标值 | 监控频率 |
|------|-------|---------|
| 开发进度 | 按时完成 | 每日 |
| 单元测试覆盖率 | > 80% | 每次提交 |
| 性能基准 | < 5% 下降 | Phase 3 |
| 集成测试通过率 | 100% | Phase 3 |

---

## 15. 参考文档（References）

### 15.1 技术参考

1. **`.meta/reanim/僵尸移动说明.md`** - 根运动法技术说明
2. **Zombie.reanim** - 僵尸动画数据（`_ground` 轨道）
3. **Epic 6/13 PRD** - Reanim 系统设计文档

### 15.2 代码参考

1. **`pkg/systems/reanim_system.go`** - Reanim 系统实现（空帧继承逻辑）
2. **`pkg/systems/behavior/zombie_behavior_handler.go`** - 僵尸行为处理
3. **`pkg/systems/wave_spawn_system.go`** - 僵尸激活逻辑（硬编码速度）

### 15.3 架构参考

1. **`docs/architecture/coordinate-system.md`** - 坐标系统说明
2. **`CLAUDE.md`** - ECS 架构原则

---

## 16. 附录（Appendix）

### 附录 A: Reanim `_ground` 轨道数据示例

**文件**: `data/reanim/Zombie.reanim`

```xml
<track>
  <name>_ground</name>
  <t><x>0</x><y>0</y><f>0</f></t>       <!-- Frame 0: 起点 -->
  <t><x>5</x><f>1</f></t>               <!-- Frame 1: X 移动到 5 -->
  <t><x>10</x><f>2</f></t>              <!-- Frame 2: X 移动到 10 -->
  <t><x>15</x><f>3</f></t>              <!-- Frame 3: X 移动到 15 -->
  <!-- ... -->
  <t><x>50</x><f>12</f></t>             <!-- Frame 12: X 移动到 50（循环结束） -->
</track>
```

**分析**:
- 动画循环长度: 12 帧
- 总位移: 50 像素
- FPS: 12（配置文件中定义）
- 循环时长: 12 / 12 = 1 秒
- 平均速度: 50 / 1 = 50 像素/秒

**当前硬编码速度**: -150.0 像素/秒（不匹配！）

**根运动法**: 自动从 `_ground` 轨道读取，无需手动计算

---

### 附录 B: 性能基准测试模板

**文件**: `pkg/utils/root_motion_bench_test.go`

```go
package utils

import (
	"testing"
)

func BenchmarkCalculateRootMotionDelta(b *testing.B) {
	// 准备测试数据
	reanimComp := &components.ReanimComponent{
		// ... 初始化 ...
	}
	groundTrack := &reanim.Track{
		// ... 初始化 ...
	}

	// 基准测试
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateRootMotionDelta(reanimComp, "_ground")
	}
}

func BenchmarkFixedVelocityMethod(b *testing.B) {
	// 对比基准：固定速度法
	position := 100.0
	velocity := -150.0
	deltaTime := 0.016667 // 60 FPS

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		position += velocity * deltaTime
	}
}
```

**运行命令**:
```bash
go test -bench=. -benchmem pkg/utils
```

**预期结果**:
- 根运动法: ~100-200 ns/op
- 固定速度法: ~10-20 ns/op
- 性能差异: < 5%（在系统总体性能中可忽略）

---

### 附录 C: 集成测试验证清单

**测试关卡**: 1-1 至 1-10

**验证项**:

| 测试项 | 验证方法 | 预期结果 |
|-------|---------|---------|
| 僵尸脚步同步 | 目视观察 | 僵尸脚步与地面完美匹配 |
| 动画循环平滑 | 观察 5+ 循环 | 无瞬移或卡顿 |
| 多僵尸同屏 | 关卡 1-10（10+ 僵尸） | 所有僵尸移动正常 |
| 性能稳定 | FPS 监控 | 60 FPS 稳定 |
| 不同僵尸类型 | basic, conehead, buckethead | 所有类型正常 |
| 边界情况 | 僵尸到达屏幕左侧 | 正确触发除草车/失败 |

---

### 附录 D: 变更日志（CHANGELOG.md）

**版本**: v0.9.0 (TBD)

**新增**:
- ✨ 根运动（Root Motion）系统 - 僵尸移动使用 Reanim `_ground` 轨道数据驱动

**优化**:
- ⚡ 僵尸移动动画同步性提升，消除滑步现象

**修复**:
- 🐛 无（纯优化变更）

**文档**:
- 📝 更新 CLAUDE.md - 添加根运动系统说明
- 📝 更新 docs/architecture/coordinate-system.md - 补充僵尸移动机制

---

## 文档结束

**最后更新**: 2025-11-20
**文档版本**: v1.0
**状态**: ✅ 已批准，准备实施

---

## 签名（Signatures）

| 角色 | 姓名 | 签名日期 | 状态 |
|------|-----|---------|------|
| Scrum Master | Bob | 2025-11-20 | ✅ 已创建 |
| 用户 | - | 2025-11-20 | ✅ 已批准 |
| Dev Agent | - | TBD | ⏳ 待实施 |
| QA | - | TBD | ⏳ 待测试 |

---

**End of Document**
