# CLAUDE.md

> **重要说明**:
> - 本文件为 **Claude Code AI 工具**提供开发上下文，**不是用户文档**
> - 仅包含开发时的技术指导、架构说明和编码规范
> - 用户文档请参见: [README.md](README.md), [快速开始](docs/quickstart.md), [用户手册](docs/user-guide.md), [开发指南](docs/development.md)

---

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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
- 代码中尽量不要添加历史追溯性注释。代码注释应该解释"为什么"，而不是"什么时候修改的"

- reanim 定义中豌豆射手的名称是 `peashootersingle` , 双发豌豆射手是 `peashooter`, 我们游戏定义中使用的是 `peashooter` 和 `repeater` , 注意区别。
- 原版《植物大战僵尸》使用固定时间步长 **0.01秒（1厘秒）** 作为物理更新基准（相当于100FPS）。粒子配置文件中的某些值基于这个时间步长定义，而非真实的"每秒"单位。
- assets/effect 下的所有配置文件都不能修改
- 所有涉及大小 、位置的常量，都需要在配置常量文件中设置，以方便后续手工调整
- 如果要查看日志，需要添加参数 `--verbose`
- 绘制任何元素时，都要考虑是否需要坐标转换。
- 验证粒子系统的实现， 可以使用类似 `go run cmd/particles/main.go --verbose --effect="Planting"  > /tmp/p.log 2>&1` 的命令运行，并查看日志
- 如果没有动画轨道,那可能是简单的动画组件,应该按配置的名称,直接播放就行
- 将刚刚的经验记录起来
- 单动画文件（如 Zombie_hand.reanim）创建 visiblesArray 时，应该填充全 0 数组 [0, 0, 0, ...]
  表示所有帧可见，而不是序列索引 [0, 1, 2, ...]，因为 mapLogicalToPhysical 函数通 animVisibles[i] == 0 判断帧是否可见

---

## 核心架构

### ECS (Entity-Component-System) 架构

本项目使用基于 Go 泛型的 ECS 架构模式。**重要原则**：
- **组件 (Components)**: 仅存储数据，不包含任何方法
- **系统 (Systems)**: 仅包含逻辑，通过查询组件处理实体
- **系统隔离**: 系统之间严禁直接调用，必须通过 EntityManager 或 EventBus 通信

### 泛型 API（推荐使用）

**优先使用泛型 API**，已废弃反射 API：

```go
// ✅ 推荐：泛型 API（类型安全，无需断言）
plantComp, ok := ecs.GetComponent[*components.PlantComponent](em, entity)
if ok {
    plantComp.Health -= 10
}

// 查询实体
entities := ecs.GetEntitiesWith2[
    *components.PlantComponent,
    *components.PositionComponent,
](em)

// ❌ 不推荐：反射 API（已废弃）
comp, ok := em.GetComponent(entity, reflect.TypeOf(&components.PlantComponent{}))
```

### Reanim 动画系统

**Reanim** 是原版 PvZ 的骨骼动画系统，项目完整复刻：

- **动画定义**: `data/reanim/` 目录下的 `.reanim` 文件（XML格式）
- **动画配置**: `data/reanim_config.yaml` 定义动画组合和轨道绑定
- **组件**: `ReanimComponent` 存储动画状态
- **系统**: `ReanimSystem` 处理动画更新和渲染

**动画组合**：多个动画可以组合播放（如身体+头部+手臂分别播放不同动画）
**轨道绑定**：子动画自动绑定到父动画的特定轨道（track）

### 粒子系统

基于 XML 配置的粒子效果系统：
- **配置文件**: `assets/effect/particles/` 下的 XML 文件（不可修改）
- **组件**: `ParticleComponent` 存储粒子状态
- **系统**: `ParticleSystem` 处理生成、更新和渲染
- **批量渲染**: 使用 DrawTriangles 实现高性能批量渲染

## 常用开发命令

### 构建与运行

```bash
# 运行游戏（从项目根目录）
go run .

# 运行游戏并启用详细日志
go run . --verbose

# 构建可执行文件
go build -o pvz-go .

# 构建优化版本
go build -ldflags="-s -w" -o pvz-go .
```

### 测试

```bash
# 运行所有测试（必须在项目根目录）
go test ./...

# 运行带覆盖率的测试
go test -cover ./...

# 运行特定包的测试
go test ./pkg/systems

# 运行特定测试函数
go test ./pkg/systems -run TestBehaviorSystem

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 调试工具

```bash
# 验证粒子系统实现
go run cmd/particles/main.go --verbose --effect="Planting" > /tmp/p.log 2>&1

# 动画查看器
go run cmd/animation_showcase/main.go --reanim=PeaShooter --verbose

# 验证僵尸胜利动画
go run cmd/verify_zombies_won/main.go

# 其他验证工具
go run cmd/verify_opening/main.go          # 开场动画
go run cmd/verify_pause_menu/main.go       # 暂停菜单
go run cmd/verify_reward_panel/main.go     # 奖励面板
```

## 项目结构

```
pvz3/
├── main.go                      # 游戏入口
├── pkg/                         # 核心代码库
│   ├── ecs/                     # ECS 框架（泛型实现）
│   ├── components/              # 组件定义（纯数据）
│   ├── systems/                 # 系统实现（纯逻辑）
│   ├── entities/                # 实体工厂函数
│   ├── scenes/                  # 游戏场景
│   ├── game/                    # 核心管理器
│   ├── config/                  # 配置加载
│   └── utils/                   # 工具函数
├── assets/                      # 游戏资源（images/audio/fonts/effect）
├── data/                        # 外部化数据（YAML配置）
│   ├── levels/                  # 关卡配置
│   ├── reanim/                  # Reanim 动画定义
│   └── reanim_config.yaml       # 动画配置
├── cmd/                         # 调试和验证工具
└── docs/                        # 文档
```

## 编码规范

### Go 代码风格

- 使用 `gofmt` 格式化所有代码
- 遵循 Go 命名约定：
  - 包名: `snake_case`
  - 结构体/接口: `PascalCase`
  - 函数/方法: `PascalCase` (public), `camelCase` (private)
  - 变量: `camelCase`
  - 常量: `PascalCase`

### ECS 特定规范

- **组件**: 仅包含字段，不包含方法
- **系统**: 不相互调用，通过 EntityManager 通信
- **工厂函数**: 使用 `NewXxxEntity()` 创建实体
- **错误处理**: 严禁忽略错误，使用 `fmt.Errorf` 或 `%w` 包装错误
- **依赖注入**: 通过构造函数注入，避免全局变量

### 测试要求

- 核心逻辑包（systems, components）目标覆盖率 ≥ 80%
- 关键系统（BehaviorSystem, PhysicsSystem）目标覆盖率 ≥ 90%
- 测试文件与源文件同包，以 `_test.go` 结尾

## 关卡配置

关卡配置位于 `data/levels/` 目录，使用 YAML 格式。

### 关卡配置格式 (Story 17.2)

```yaml
# data/levels/level-1-4.yaml (新格式示例)
id: "1-4"
name: "前院白天 1-4"
description: "标准关卡"

# Story 17.2 新增顶层字段
flags: 1                       # 旗帜数量，默认从 waves 中 isFlag 数量推断
sceneType: "day"               # 场景类型: day, night, pool, fog, roof, moon
rowMax: 5                      # 最大行数: 5 (前院/屋顶) 或 6 (后院)

# 其他配置字段
openingType: "standard"        # 开场类型: tutorial, standard, special
enabledLanes: [1, 2, 3, 4, 5]  # 启用的行
availablePlants:               # 可用植物
  - peashooter
  - sunflower
initialSun: 50                 # 初始阳光

# 波次配置
waves:
  # 常规波次
  - waveNum: 1                  # Story 17.2: 波次编号（从 1 开始）
    type: "Fixed"               # Story 17.2: 波次类型 (Fixed/ExtraPoints/Final)
    delay: 10
    zombies:
      - type: "basic"
        lanes: [1, 2, 3, 4, 5]
        count: 2
        spawnInterval: 2.0

  # 旗帜波次
  - waveNum: 4
    type: "Final"               # 最终波
    delay: 5
    isFlag: true
    flagIndex: 1
    zombies:
      - type: "basic"
        lanes: [1, 2, 3, 4, 5]
        count: 6
```

### 波次类型 (Story 17.2)

| 类型 | 说明 | 使用场景 |
|------|------|----------|
| `Fixed` | 固定出怪列表 | 标准关卡的常规波次 |
| `ExtraPoints` | 动态点数分配 | 1-10 传送带关卡 |
| `Final` | 最终波 | 关卡最后一波 |

### 场景类型 (Story 17.2)

| 类型 | 章节 | RowMax | 特殊规则 |
|------|------|--------|----------|
| `day` | 白天 | 5 | 无 |
| `night` | 黑夜 | 5 | 墓碑 |
| `pool` | 泳池 | 6 | 第3,4行水路 |
| `fog` | 雾夜 | 6 | 第3,4行水路 + 迷雾 |
| `roof` | 屋顶 | 5 | 左5列斜坡 |
| `moon` | 月夜 | 5 | 墓碑 |

### 向后兼容性

所有新字段都是可选的，现有关卡配置无需修改即可正常加载：
- `Flags`: 默认从 waves 中的 isFlag 数量推断
- `SceneType`: 默认 "day"
- `RowMax`: 默认 5
- `WaveNum`: 默认从数组索引 +1 推断
- `Type`: 默认从 isFlag 字段推断 ("Final" 或 "Fixed")

## 僵尸生成规则 (Story 17.3)

僵尸生成限制检查系统确保只有符合游戏规则的僵尸会被生成。配置文件位于 `data/spawn_rules.yaml`。

### 规则配置格式

```yaml
# 僵尸阶数映射表
zombieTiers:
  basic: 1        # 一阶：第 1 波起可出现
  conehead: 1
  buckethead: 2   # 二阶：第 3 波起
  polevaulter: 3  # 三阶：第 8 波起
  gargantuar: 4   # 四阶：第 15 波起（根据轮数调整）

# 阶数波次限制
tierWaveRestrictions:
  1: 1   # 一阶：第 1 波起
  2: 3   # 二阶：第 3 波起
  3: 8   # 三阶：第 8 波起
  4: 15  # 四阶：第 15 波起（会根据轮数调整）

# 红眼数量上限配置
redEyeRules:
  startRound: 5         # 从第 5 轮开始允许红眼
  capacityPerRound: 1   # 每轮增加 1 个红眼上限

# 场景类型限制
sceneTypeRestrictions:
  waterZombies:        # 水路专属僵尸
    - snorkel
    - dolphinrider
  dancingRestrictions:
    prohibitedScenes:  # 舞王禁止的场景
      - roof
  waterLaneConfig:     # 水路行配置
    pool: [3, 4]
    fog: [3, 4]
```

### 限制规则说明

**1. 阶数限制**：
- 一阶僵尸（普僵、路障）：第 1 波起可出现
- 二阶僵尸（铁桶、读报）：第 3 波起
- 三阶僵尸（撑杆、橄榄球等）：第 8 波起
- 四阶僵尸（巨人）：第 15 波起，根据轮数提前：`MinWave = max(15 - RoundNumber, 1)`

**2. 红眼数量上限**：
```
红眼上限 = 0  (轮数 < 5)
红眼上限 = 轮数 - 4  (轮数 ≥ 5)
```

**3. 场景限制**：
- 水路僵尸（潜水、海豚）只能在水路行（泳池第 3、4 行）生成
- 非水路僵尸不能在水路行生成
- 舞王禁止在屋顶场景出现

### 代码集成

生成限制检查通过纯函数实现，遵循 ECS 零耦合原则：

```go
// 在 WaveSpawnSystem 中使用
valid, reason := ValidateZombieSpawn(
    zombieType,
    lane,
    constraint,  // SpawnConstraintComponent
    roundNumber,
    spawnRules,  // SpawnRulesConfig
)
if !valid {
    log.Printf("Zombie spawn rejected: %s", reason)
    return 0 // 跳过生成
}
```

### 关卡配置覆盖

如需为特定关卡覆盖默认规则，可在关卡配置中添加 `spawnRulesOverride` 字段（未来扩展）。

## 技术参考

- **游戏引擎**: [Ebitengine v2](https://ebiten.org/)
- **Go 版本**: 1.21+
- **配置格式**: YAML (gopkg.in/yaml.v3)
- **架构模式**: Entity-Component-System (ECS)
- **动画系统**: Reanim（原版 PvZ 骨骼动画系统）
- **粒子系统**: XML 配置驱动

## 参考文档

- [README.md](README.md) - 项目概览
- [docs/architecture.md](docs/architecture.md) - 详细架构设计
- [docs/development.md](docs/development.md) - 开发指南
- [docs/prd.md](docs/prd.md) - 产品需求文档
- [docs/quickstart.md](docs/quickstart.md) - 快速开始指南

---
