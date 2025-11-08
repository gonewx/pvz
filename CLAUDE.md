# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

这是一个使用 Go 语言和 Ebitengine 引擎开发的《植物大战僵尸》复刻项目。项目采用实体-组件-系统(ECS)架构模式,目标是精确复刻原版PC游戏的前院白天关卡。

## 核心开发命令

### 项目初始化与依赖管理
```bash
# 初始化 Go 模块(如果尚未初始化)
go mod init github.com/decker502/pvz

# 添加 Ebitengine 依赖
go get github.com/hajimehoshi/ebiten/v2

# 添加 YAML 解析库
go get gopkg.in/yaml.v3

# 下载所有依赖
go mod download

# 整理依赖
go mod tidy
```

### 构建与运行
```bash
# 运行游戏
go run .

# 构建可执行文件
go build -o pvz-go .

# 构建优化版本(发布用)
go build -ldflags="-s -w" -o pvz-go .

# 交叉编译 Windows 版本
GOOS=windows GOARCH=amd64 go build -o pvz-go.exe .

# 交叉编译 macOS 版本
GOOS=darwin GOARCH=amd64 go build -o pvz-go-mac .
```

### 代码质量
```bash
# 格式化代码
gofmt -w .

# 运行所有测试
go test ./...

# 运行测试并显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 高层架构

### ECS 架构模式

本项目采用实体-组件-系统(Entity-Component-System)架构:

- **实体(Entity)**: 游戏对象的唯一标识符(如植物、僵尸、子弹)
- **组件(Component)**: 纯数据结构,描述实体的属性(如位置、生命值、精灵图)
- **系统(System)**: 处理逻辑的函数,对拥有特定组件的实体进行操作

**关键原则**:
- 组件只包含数据,不包含方法
- 系统之间通过 EntityManager 或 EventBus 通信,不直接调用
- 数据与行为完全分离

### 核心系统层级

```
main.go (游戏入口)
    ↓
SceneManager (场景管理器)
    ↓
├── MainMenuScene (主菜单场景)
└── GameScene (游戏场景)
        ↓
    EntityManager (实体管理器)
        ↓
    ├── InputSystem (输入系统)
    ├── BehaviorSystem (行为系统)
    ├── PhysicsSystem (物理系统)
    ├── ReanimSystem (Reanim动画系统 - Story 6.5)
    ├── ParticleSystem (粒子系统 - Story 7.2)
    ├── UISystem (UI系统)
    └── RenderSystem (渲染系统)
        ├── DrawGameWorld() - 植物、僵尸、子弹
        ├── DrawParticles() - 粒子效果 (Story 7.3)
        └── DrawSuns() - 阳光
```

### 目录结构规范

```plaintext
pvz/
├── main.go                 # 游戏主入口
├── assets/                 # 游戏资源
│   ├── images/             # 图片资源(spritesheets)
│   ├── audio/              # 音频资源
│   └── fonts/              # 字体文件
├── data/                   # 外部化游戏数据
│   ├── levels/             # 关卡配置(YAML)
│   └── units/              # 单位属性文件
├── pkg/                    # 核心代码库
│   ├── components/         # 所有组件定义
│   ├── entities/           # 实体工厂函数
│   ├── systems/            # 所有系统实现
│   ├── scenes/             # 游戏场景
│   ├── ecs/                # ECS框架核心
│   ├── game/               # 游戏核心管理器
│   ├── utils/              # 通用工具函数
│   └── config/             # 配置加载与管理
├── go.mod
└── go.sum
```

## 泛型 ECS API 使用指南

本项目的 ECS 框架已升级为基于 Go 泛型的类型安全 API（Epic 9 - Story 9.1/9.2/9.3）。

### 泛型 API 优势

- ✅ **编译时类型检查**：消除运行时 panic 风险，约 150+ 处潜在错误被编译器捕获
- ✅ **无需类型断言**：代码更简洁，消除了 60+ 处手动类型断言
- ✅ **性能提升**：减少反射开销，综合性能提升约 10%
- ✅ **更好的 IDE 支持**：代码补全、类型推导、重构工具全面支持
- ✅ **代码可读性**：代码行数减少 40-60%，更易维护

### 基本用法

#### 1. 获取组件（GetComponent）

```go
// ❌ 旧方式（反射 API，已废弃）
comp, ok := em.GetComponent(entity, reflect.TypeOf(&components.PlantComponent{}))
if ok {
    plantComp := comp.(*components.PlantComponent) // 手动类型断言，可能 panic
    plantComp.Health -= 10
}

// ✅ 新方式（泛型 API，推荐）
plantComp, ok := ecs.GetComponent[*components.PlantComponent](em, entity)
if ok {
    plantComp.Health -= 10 // 无需类型断言，类型安全
}
```

#### 2. 添加组件（AddComponent）

```go
// ❌ 旧方式
em.AddComponent(entity, &components.PlantComponent{
    PlantType: "Peashooter",
    Health:    300,
})

// ✅ 新方式（类型自动推导）
ecs.AddComponent(em, entity, &components.PlantComponent{
    PlantType: "Peashooter",
    Health:    300,
})
```

#### 3. 检查组件存在性（HasComponent）

```go
// ❌ 旧方式
if em.HasComponent(entity, reflect.TypeOf(&components.PlantComponent{})) {
    // 处理植物逻辑
}

// ✅ 新方式
if ecs.HasComponent[*components.PlantComponent](em, entity) {
    // 处理植物逻辑
}
```

#### 4. 查询实体（GetEntitiesWith）

```go
// ❌ 旧方式（冗长且运行时检查）
entities := em.GetEntitiesWith(
    reflect.TypeOf(&components.BehaviorComponent{}),
    reflect.TypeOf(&components.PlantComponent{}),
    reflect.TypeOf(&components.PositionComponent{}),
)

// ✅ 新方式（编译时类型检查）
entities := ecs.GetEntitiesWith3[
    *components.BehaviorComponent,
    *components.PlantComponent,
    *components.PositionComponent,
](em)
```

**函数选择规则**：
- 查询 1 个组件 → 使用 `GetEntitiesWith1[T1]`
- 查询 2 个组件 → 使用 `GetEntitiesWith2[T1, T2]`
- 查询 3 个组件 → 使用 `GetEntitiesWith3[T1, T2, T3]`
- 查询 4 个组件 → 使用 `GetEntitiesWith4[T1, T2, T3, T4]`
- 查询 5 个组件 → 使用 `GetEntitiesWith5[T1, T2, T3, T4, T5]`
- 查询 5+ 组件 → 使用反射 API 或分步查询

**特点**：
- 无需手动使用 `reflect.TypeOf`
- 代码更简洁
- 与其他泛型 API 风格一致

**解决方案 C** （推荐）：重新设计组件
- 如果需要查询超过 5 个组件，可能说明组件设计过于碎片化
- 考虑合并相关组件或使用组合组件

---

## 核心组件说明

### 必要组件(所有实体必备)
- `PositionComponent`: 存储 X,Y 坐标
- `SpriteComponent`: 存储要绘制的图像引用

### 功能组件
- `AnimationComponent`: 管理基于spritesheet的帧动画
- `HealthComponent`: 生命值管理(CurrentHealth, MaxHealth)
- `BehaviorComponent`: 定义实体行为类型(向日葵、豌豆射手等)
- `TimerComponent`: 通用计时器(用于攻击冷却、生产周期等)
- `UIComponent`: 标记UI元素及其状态
- `VelocityComponent`: 移动速度(用于子弹、僵尸移动)
- `CollisionComponent`: 碰撞检测的边界框
- `ParticleComponent`: 粒子效果数据(位置、速度、颜色、生命周期等) - Story 7.2
- `EmitterComponent`: 粒子发射器配置(生成规则、限制、力场等) - Story 7.2

## 坐标系统使用指南

### 世界坐标 vs 屏幕坐标

本项目使用**世界坐标系统**：
- **世界坐标**：相对于背景图片左上角（固定）
- **屏幕坐标**：相对于游戏窗口左上角（随摄像机移动）
- **转换公式**：`worldX = screenX + cameraX`

### 何时使用哪种坐标？

| 场景 | 使用坐标类型 | 示例 |
|------|------------|------|
| 组件存储位置 | 世界坐标 | `PositionComponent.X/Y` |
| 网格定义 | 世界坐标 | `config.GridWorldStartX` |
| 鼠标输入 | 屏幕坐标 | `ebiten.CursorPosition()` |
| 渲染绘制 | 屏幕坐标 | `screen.DrawImage()` |

## Reanim 动画系统使用指南

**重要说明** 本系统中所有对Reanim 动画系统的理解和说明都是猜想，仅作为参考，有可能有错误的理解，请谨慎使用。

### 概述

Reanim 是原版《植物大战僵尸》使用的骨骼动画系统。本项目实现了完整的 Reanim 渲染支持，经过 Story 6.5 的修复，现已正确理解并实现了 Reanim 格式的三个核心机制。

### 核心机制

#### 1. 混合轨道机制（Hybrid Tracks）

**发现**：统计 5 个 Reanim 文件（87 个轨道）发现，76% 的轨道是混合轨道。

**定义**：混合轨道同时具有：
- ✅ 图片资源（ImagePath）- 可以渲染
- ✅ FrameNum 值（f 值）- 自我控制可见性
- ✅ 变换数据（X, Y, ScaleX 等）- 独立运动

**关键理解**：
- ❌ **错误**：f=-1 表示隐藏该部件
- ✅ **正确**：所有轨道（包括纯视觉轨道）都有 FrameNum 值。纯视觉轨道的 FrameNum 默认为 0（可见）。

**示例**：`PeaShooterSingle.reanim` 中的 `stalk_bottom` 轨道
```
- 有 ImagePath: "IMAGE_REANIM_PEASHOOTER_STALKBOTTOM"
- 有 f 值变化: -1, 0, 1, 2 (定义自己的时间窗口)
- 有位置变化: X, Y 值在每帧变化
```

#### 2. 父子层级关系（Parent-Child Hierarchy）

**问题**：头部动画僵硬不动，不随身体摆动。

**原因**：`anim_stem`（茎干骨骼）作为父节点，头部应该继承它的偏移量。

**解决方案**：
```go
// 定义 anim_stem 初始位置
const (
    ReanimStemInitX = 37.6  // 从 PeaShooterSingle.reanim 提取
    ReanimStemInitY = 48.7
)

// 计算 stem 偏移
func getStemOffset(reanim, physicalFrame) (offsetX, offsetY) {
    currentX, currentY := getStemPosition(physicalFrame)
    offsetX = currentX - ReanimStemInitX
    offsetY = currentY - ReanimStemInitY
    return offsetX, offsetY
}

// 渲染头部时叠加偏移
if isHeadTrack(trackName) {
    x = partX + stemOffsetX
    y = partY + stemOffsetY
}
```

**效果**：头部随身体摆动，动作自然流畅。

### 轨道类型分类

系统识别三种轨道类型：

```go
// 1. 动画定义轨道（只有 f 值，无图片）
AnimationDefinitionTracks = map[string]bool{
    "anim_idle":      true,
    "anim_shooting":  true,
    "anim_head_idle": true,
    "anim_full_idle": true,
}

// 2. 逻辑轨道（定义附着点或父变换，无图片）
LogicalTracks = map[string]bool{
    "anim_stem": true,  // 父骨骼
    "_ground":   true,  // 地面附着点
}

// 3. 混合轨道（有图片 + f 值 + 变换）
// 大部分轨道，通过排除法识别
```

### 渲染判断逻辑

**shouldRenderTrack** 函数实现了正确的渲染规则：

```go
func shouldRenderTrack(track, frame, animName) bool {
    // 步骤 0: VisibleTracks 白名单（最高优先级）
    if VisibleTracks != nil {
        return VisibleTracks[trackName]
    }

    // 步骤 1: 跳过逻辑轨道（无图片）
    if isLogicalTrack(trackName) {
        return false
    }

    // 步骤 2: 检查是否有图片
    if frame.ImagePath == "" {
        return false
    }

    // 步骤 3: 检查时间窗口
    if animVisibles := AnimVisiblesMap[animName]; animVisibles != nil {
        if frame.FrameNum != nil && *frame.FrameNum == -1 {
            return animVisibles[logicalFrame] == 0
        }
    }

    // 步骤 4: 有图片就渲染
    return true
}
```

### 常见误区

#### ❌ 误区 1：f=-1 表示隐藏部件

**错误理解**：看到 `f=-1` 就不渲染该部件

**正确理解**：
- f=-1 表示该部件**使用动画定义轨道**的时间窗口
- 只有当动画定义轨道标记该帧为隐藏时，部件才隐藏
- 76% 的轨道都有 f=-1，如果全部隐藏，游戏将无法显示

#### ❌ 误区 2：头部位置 = 头部轨道的位置

**错误实现**：
```go
x = headFrame.X  // 忽略父骨骼
y = headFrame.Y
```

**正确实现**：
```go
stemOffsetX, stemOffsetY := getStemOffset(primaryFrame)
x = headFrame.X + stemOffsetX  // 叠加父骨骼偏移
y = headFrame.Y + stemOffsetY
```

### 调试技巧

#### 1. 使用 verbose 日志

```bash
go run . --verbose
```

日志会输出：
- stem 偏移计算过程
- 时间窗口构建信息
- 帧继承处理过程

#### 2. 使用对比测试程序

```bash
go run cmd/render_animation_comparison/main.go --verbose
```

显示三种渲染模式：
- 左侧：严格遵守 f=-1（错误）
- 中间：忽略 f=-1（部分正确）
- 右侧：正确的帧继承处理（正确）✅

#### 3. 检查 MergedTracks

```go
// 验证帧继承是否正确
for trackName, frames := range reanimComp.MergedTracks {
    for i, frame := range frames {
        if frame.X == nil || frame.Y == nil {
            log.Printf("ERROR: Frame %d of track %s missing position", i, trackName)
        }
    }
}
```

### API 使用示例

#### 播放动画

```go
// 播放动画
reanimSystem.PlayAnimation(entityID, "anim_idle")

// 攻击动画
reanimSystem.PlayAnimation(entityID, "anim_shooting")
```

#### 控制部件显示（僵尸等）

```go
// 使用 VisibleTracks 白名单
reanimComp.VisibleTracks = map[string]bool{
    "Zombie_body":      true,
    "Zombie_head":      true,
    "Zombie_outerarm":  false,  // 隐藏手臂
}
```

#### 播放多个动画（叠加）

```go
// 豌豆射手攻击：身体动画 + 头部动画同时播放
reanimSystem.PlayAnimations(peashooterID, []string{"anim_shooting", "anim_head_idle"})
```

#### 增量控制动画

```go
// 基础动画
reanimSystem.PlayAnimation(entityID, "anim_walk")

// 叠加特效动画（保留已有动画）
reanimSystem.AddAnimation(entityID, "anim_burning")

// 移除特效动画
reanimSystem.RemoveAnimation(entityID, "anim_burning")
```

#### 轨道绑定机制（Story 13.1）

**自动绑定（推荐）**：
```go
// 播放多个动画时自动分析轨道绑定
// 系统会自动将每个轨道绑定到运动最明显的动画
reanimSystem.PlayAnimations(entityID, []string{"anim_shooting", "anim_head_idle"})

// 内部自动设置 TrackBindings:
//   TrackBindings["anim_face"] = "anim_head_idle"      // 头部用动画A
//   TrackBindings["stalk_bottom"] = "anim_shooting"    // 身体用动画B
```

**手动绑定（高级用法）**：
```go
// 手动配置轨道绑定（用于特殊场景或微调）
reanimSystem.SetTrackBindings(entityID, map[string]string{
    "anim_face":    "anim_head_idle",   // 头部用动画A
    "stalk_bottom": "anim_shooting",    // 身体用动画B
})
```

**查看绑定结果**：
```bash
# 运行游戏时会自动输出绑定信息
go run .

# 日志输出：
# [ReanimSystem] 自动轨道绑定 (entity 123):
#   - anim_face -> anim_head_idle
#   - stalk_bottom -> anim_shooting
```

**绑定原理**：
- 系统计算每个轨道在不同动画时间窗口内的位置方差
- 将轨道绑定到方差最大（运动最明显）的动画
- 实现"头部用动画A，身体用动画B"的复杂组合

### 常见问题（FAQ）

**Q: 如何让植物攻击时头部和身体同时显示？**

A: 使用多动画叠加 API：
```go
reanimSystem.PlayAnimations(entityID, []string{"anim_shooting", "anim_head_idle"})
```

**Q: 如何控制僵尸装备显示？**

A: 使用 VisibleTracks 白名单：
```go
reanimComp.VisibleTracks = map[string]bool{
    "Zombie_body": true,
    "anim_cone":   true,  // 显示路障
}
```

**Q: PlayAnimation 和 PlayAnimations 有什么区别？**

A:
- `PlayAnimation(id, "anim")` - 播放单个动画（清空旧动画）
- `PlayAnimations(id, []string{"anim1", "anim2"})` - 播放多个动画（清空旧动画，同时播放多个）
- `AddAnimation(id, "anim")` - 增量添加（保留已有动画）

**Q: 什么时候使用多动画叠加？**

A: 当一个实体需要同时控制多个部件的动画时，例如：
- 豌豆射手攻击：身体攻击 + 头部摆动
- 僵尸行走时着火：行走动画 + 燃烧特效

### 参考文档

- **Reanim 格式指南**: `docs/reanim/reanim-format-guide.md`
- **修复指南**: `docs/reanim/reanim-fix-guide.md`
- **混合轨道分析**: `docs/reanim/reanim-hybrid-track-discovery.md`
- **Story 6.5**: `docs/stories/6.5.story.md`

---

## 资源管理

### 概述

项目使用统一的 `ResourceManager` 进行资源加载和缓存管理。从 2025年10月开始，资源系统已升级为**基于 YAML 配置的动态资源管理**，支持通过资源 ID 加载资源，提高了可维护性和可扩展性。

### 资源配置文件

**配置文件路径:** `assets/config/resources.yaml`

资源配置文件定义了所有游戏资源及其 ID 映射关系。资源按照**资源组（Resource Groups）**组织，可以批量加载。

**配置结构示例：**
```yaml
version: "1.0"
base_path: assets
groups:
  init:
    images:
      - id: IMAGE_BLANK
        path: properties/blank.png
      - id: IMAGE_POPCAP_LOGO
        path: properties/PopCap_Logo.jpg
  loadingimages:
    images:
      - id: IMAGE_REANIM_SEEDS
        path: reanim/seeds.png
        cols: 9  # 精灵图列数
    sounds:
      - id: SOUND_BUTTONCLICK
        path: sounds/buttonclick.ogg
```
## 文档参考

项目相关文档位于 `docs/` 目录:
- `docs/prd.md`: 产品需求文档
- `docs/architecture.md`: 详细架构设计文档
- `docs/brief.md`: 项目简介
- `docs/front-end-spec.md`: 前端规范

---

## 用户手工维护区域

<!--
在此区域添加您的个人项目笔记、配置、工作流程等内容
此区域内容不会被文档更新脚本覆盖
-->

- 所有资源类文件已经准备好， @assets

- 必须在项目根目录运行测试，所以测试中需要引用的项目目录和文件，也要以相对于项目根目录来组织路径

- 永远不要为了赶时间、或认为篇幅有限、或认为任务复杂，而主观的简化或加速任务的实现。如果有这种情况，要显式的征得我的同意，或授权确认后，才能简化实现。

- 如果遇到网络问题，请尝试使用网络代理 http://127.0.0.1:2080 访问

- 遇到反复无法修复的问题或有不熟悉的第三方库, 尝试使用 `mcp__deepwiki` 工具的`ask_question`方法，查阅最新的文档，以找到最正确的修复方法 。

- 确认功能正常后，再提交git
---

- 原版《植物大战僵尸》使用固定时间步长 **0.01秒（1厘秒）** 作为物理更新基准（相当于100FPS）。粒子配置文件中的某些值基于这个时间步长定义，而非真实的"每秒"单位。
- assets/effect 下的所有配置文件都不能修改
- 所有涉及大小 、位置的常量，都需要在配置常量文件中设置，以方便后续手工调整
- 如果要查看日志，需要添加参数 `--verbose`
- 绘制任何元素时，都要考虑是否需要坐标转换。
- 验证粒子系统的实现， 可以使用类似 `go run cmd/particles/main.go --verbose --effect="Planting"  > /tmp/p.log 2>&1` 的命令运行，并查看日志
- 如果没有动画轨道,那可能是简单的动画组件,应该按配置的名称,直接播放就行
