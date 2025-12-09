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
- **动画配置**: `data/reanim_config/*.yaml` 定义动画组合和轨道绑定
- **组件**: `ReanimComponent` 存储动画状态
- **系统**: `ReanimSystem` 处理动画更新和渲染

**动画组合**：多个动画可以组合播放（如身体+头部+手臂分别播放不同动画）
**轨道绑定**：子动画自动绑定到父动画的特定轨道（track）

### 粒子系统

基于 XML 配置的粒子效果系统：
- **配置文件**: `data/particles/` 下的 XML 文件（不可修改）
- **组件**: `ParticleComponent` 存储粒子状态
- **系统**: `ParticleSystem` 处理生成、更新和渲染
- **批量渲染**: 使用 DrawTriangles 实现高性能批量渲染

## 项目结构

```
pvz/
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
│   └── icons/                   # 应用图标（多平台）
│       ├── windows/             # Windows ico + png
│       ├── macos/               # macOS icon.iconset
│       ├── linux/               # Linux 多尺寸 png
│       ├── ios/                 # iOS AppIcon.appiconset
│       ├── android/             # Android mipmap 图标
│       └── web/                 # Web favicon + PWA 图标
├── data/                        # 外部化数据（YAML配置）
│   ├── levels/                  # 关卡配置
│   ├── reanim/                  # Reanim 动画定义
│   └── reanim_config.yaml       # 动画配置
├── scripts/                     # 构建脚本
│   ├── build-apk.sh             # Android APK 构建
│   ├── Info.plist               # macOS 应用配置
│   └── pvz.desktop              # Linux 桌面入口
├── cmd/                         # 调试和验证工具
└── docs/                        # 文档
```

## 构建命令

使用 Makefile 管理构建流程：

```bash
make help              # 显示帮助
make build             # 构建当前平台
make build-windows     # 构建 Windows
make build-linux       # 构建 Linux
make build-darwin      # 构建 macOS (需要 macOS)
make build-wasm        # 构建 WebAssembly
make generate-icons    # 生成 Windows .syso 图标资源
make package-linux     # 打包 Linux 发布包
make build-darwin-app  # 构建 macOS .app 包
make build-apk         # 构建 Android APK
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

## 关卡配置

关卡配置位于 `data/levels/` 目录，使用 YAML 格式。

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
