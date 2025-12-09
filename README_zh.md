# 植物大战僵尸 AI 复刻版

> 使用 Go 语言和 Ebitengine 引擎开发的《植物大战僵尸》精确复刻项目

[![CI](https://github.com/gonewx/pvz/actions/workflows/ci.yml/badge.svg)](https://github.com/gonewx/pvz/actions/workflows/ci.yml)
[![Release](https://github.com/gonewx/pvz/actions/workflows/release.yml/badge.svg)](https://github.com/gonewx/pvz/actions/workflows/release.yml)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

简体中文 | [English](README.md)

## 🎮 项目简介

本项目是对经典塔防游戏《植物大战僵尸》PC中文年度版的精确复刻，专注于学习和实践 Go 游戏开发。采用现代化的 Entity-Component-System (ECS) 架构模式，使用 Go 泛型实现类型安全的游戏逻辑，目标是 100% 还原原版游戏的前院白天关卡体验。

**核心特点：**
- 🏗️ **ECS 架构** - 基于 Go 泛型的类型安全 ECS 框架
- 🎨 **原版动画** - 完整实现 Reanim 骨骼动画系统
- ✨ **粒子特效** - XML 配置驱动的粒子系统
- 📊 **数据驱动** - YAML 配置文件管理游戏数据
- 🎯 **高忠实度** - 精确复刻原版游戏数值和行为

## ✨ 功能特性

### 已实现功能

#### 核心系统
- ✅ **游戏框架** - 场景管理、状态机、主循环
- ✅ **资源管理** - 统一的图片、音频、配置加载系统
- ✅ **阳光系统** - 天空掉落和向日葵生产
- ✅ **植物系统** - 种植、冷却、卡片选择
- ✅ **僵尸系统** - AI、移动、攻击、生命值
- ✅ **战斗系统** - 子弹碰撞、伤害计算
- ✅ **关卡系统** - 波次管理、进度条、胜负判定

#### 动画与特效
- ✅ **Reanim 动画系统** - 原版骨骼动画，支持部件变换
- ✅ **粒子特效系统** - 爆炸、溅射、土粒飞溅等
- ✅ **动画组合机制** - 多动画叠加、轨道绑定、父子偏移
- ✅ **配置驱动动画** - YAML 配置管理动画组合

#### 植物（MVP范围）
- 🌻 向日葵 (Sunflower)
- 🌱 豌豆射手 (Peashooter)
- 🛡️ 坚果墙 (Wall-nut)
- 💣 樱桃炸弹 (Cherry Bomb)

#### 僵尸类型
- 🧟 普通僵尸 (Normal Zombie)
- 🚧 路障僵尸 (Conehead Zombie)

#### 关卡内容
- ✅ **第一章（前院白天）** - 关卡 1-1 至 1-5
- ✅ **教学系统** - 1-1 单行草地引导
- ✅ **特殊关卡** - 1-5 坚果保龄球
- ✅ **开场动画** - 镜头平移、僵尸预告
- ✅ **选卡界面** - 植物选择、解锁系统

#### UI 与体验
- ✅ **主菜单系统** - 开始冒险、退出游戏
- ✅ **暂停菜单** - 继续、重新开始、返回主菜单
- ✅ **铲子工具** - 移除植物
- ✅ **除草车防线** - 最后防线机制
- ✅ **关卡进度条** - 旗帜、最后一波提示

## 🚀 快速开始

### 环境要求

- **Go 版本**: 1.24 或更高
- **操作系统**: Windows / macOS / Linux / Android / WASM
- **内存**: 至少 2GB RAM
- **显卡**: 支持 OpenGL 2.1+

### 安装与运行

```bash
# 1. 克隆仓库
git clone https://github.com/gonewx/pvz
cd pvz

# 2. 下载依赖
go mod download

# 3. 运行游戏
go run .
```

游戏将以 800x600 窗口启动。

### 构建可执行文件

```bash
# 使用 Makefile 构建（推荐）
make build                # 构建当前平台
make build-linux          # 构建 Linux (amd64 + arm64)
make build-windows        # 构建 Windows (amd64 + arm64)
make build-darwin         # 构建 macOS (需要 macOS 主机)
make build-wasm           # 构建 WebAssembly

# 手动构建
go build -o pvz-go .

# 构建优化版本（体积更小）
go build -ldflags="-s -w" -o pvz-go .
```

### 构建带图标的发布版本

```bash
# 生成 Windows 图标资源 (.syso)
make generate-icons

# 打包 Linux 发布包（含图标和 .desktop）
make package-linux

# 构建 macOS .app 包（需要 macOS）
make build-darwin-app

# 构建 Android APK
make build-apk

# 查看 iOS 图标使用说明
make ios-icons-info
```

详细说明请参见 **[快速开始指南](docs/quickstart.md)**

## 📖 文档

### 用户文档
- **[快速开始指南](docs/quickstart.md)** - 5 分钟上手
- **[用户手册](docs/user-guide.md)** - 游戏操作和功能说明

### 开发文档
- **[开发指南](docs/development.md)** - 代码贡献和开发指引
- **[产品需求文档 (PRD)](docs/prd.md)** - 完整的功能规范
- **[架构文档](docs/architecture.md)** - 技术架构设计

> **注意**: `CLAUDE.md` 是为 Claude Code AI 工具提供的开发上下文，包含 ECS 架构、Reanim 系统等技术细节，主要面向开发者。

## 🏗️ 项目结构

```
pvz/
├── main.go                 # 游戏入口
├── assets/                 # 游戏资源
│   ├── images/             # 图片资源（spritesheets）
│   ├── audio/              # 音频资源
│   ├── fonts/              # 字体文件
│   ├── effect/             # 粒子配置
│   └── icons/              # 应用图标（多平台）
│       ├── windows/        # Windows ico 和 png
│       ├── macos/          # macOS iconset
│       ├── linux/          # Linux 多尺寸 png
│       ├── ios/            # iOS AppIcon.appiconset
│       ├── android/        # Android mipmap 图标
│       └── web/            # Web favicon 和 PWA 图标
├── data/                   # 外部化游戏数据
│   ├── levels/             # 关卡配置（YAML）
|   ├── reanim/             # Reanim 动画定义
│   └── reanim_config.yaml  # 动画配置
├── pkg/                    # 核心代码库
│   ├── components/         # 所有组件定义
│   ├── entities/           # 实体工厂函数
│   ├── systems/            # 所有系统实现
│   ├── scenes/             # 游戏场景
│   ├── ecs/                # ECS 框架核心
│   ├── game/               # 游戏核心管理器
│   ├── utils/              # 通用工具函数
│   └── config/             # 配置加载与管理
├── scripts/                # 构建脚本
│   ├── build-apk.sh        # Android APK 构建
│   ├── Info.plist          # macOS 应用配置
│   └── pvz.desktop         # Linux 桌面入口
├── docs/                   # 文档
└── .meta/                  # 参考资料和元数据
```

## 🎯 技术栈

- **语言**: Go 1.21+
- **游戏引擎**: [Ebitengine v2](https://ebiten.org/)
- **架构模式**: Entity-Component-System (ECS)
- **配置格式**: YAML
- **测试框架**: Go 原生 testing

### 核心技术亮点

1. **Go 泛型 ECS** - 编译时类型安全，性能提升 10-30%
2. **Reanim 骨骼动画** - 100% 还原原版动画系统
3. **数据驱动设计** - 所有游戏数值外部化配置
4. **高性能粒子系统** - DrawTriangles 批量渲染

## 🎮 游戏操作

### 基本控制
- **鼠标左键** - 收集阳光、选择植物、种植植物
- **鼠标右键** - 取消植物选择
- **ESC 键** - 暂停/继续游戏
- **--verbose** - 启用详细日志（调试）

### 游戏流程
1. 从主菜单选择"开始冒险"
2. 在选卡界面选择植物（最多 6-10 个）
3. 等待阳光，选择植物卡片
4. 点击草坪格子种植植物
5. 防御僵尸，完成所有波次

详细操作说明请参见 **[用户手册](docs/user-guide.md)**

## 🧪 测试

```bash
# 运行所有测试
go test ./...

# 运行测试并显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 🤝 贡献

欢迎贡献代码！本项目主要用于学习 Go 游戏开发。

### 贡献流程
1. Fork 本仓库
2. 创建 feature 分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

详细指南请参见 **[开发指南](docs/development.md)**

## 📊 项目状态

### MVP 范围
✅ **已完成** - 前院白天完整体验（第一章全部 10 个关卡）

### 完成的 Epics
- ✅ Epic 1: 游戏基础框架与主循环
- ✅ Epic 2: 核心资源与玩家交互
- ✅ Epic 3: 植物系统与部署
- ✅ Epic 4: 基础僵尸与战斗逻辑
- ✅ Epic 5: 游戏流程与高级单位
- ✅ Epic 6: Reanim 动画系统迁移
- ✅ Epic 7: 粒子特效系统
- ✅ Epic 8: 第一章关卡实现
- ✅ Epic 9: ECS 框架泛型化重构
- ✅ Epic 10: 游戏体验完善
- ✅ Epic 11: 关卡 UI 增强
- ✅ Epic 12: 主菜单系统
- ✅ Epic 13: Reanim 动画系统现代化重构

### 后续计划
- 🔄 Epic 14+: 更多关卡和功能（待规划）

## 📜 许可证

本项目仅用于学习和技术研究目的。

详见 [免责声明](DISCLAIMER_zh.md) 了解重要法律声明。

## 🙏 致谢

- **原版游戏**: PopCap Games 的《植物大战僵尸》
- **游戏引擎**: [Ebitengine](https://ebiten.org/) 团队
- **开发工具**: Claude Code AI

## 📞 联系方式

如有问题或建议，请通过以下方式联系：
- 提交 [Issue](../../issues)
- 发起 [Discussion](../../discussions)

---

**注意**: 本项目仅用于学习和技术研究，不用于商业用途。所有游戏资源版权归原作者所有。
