# **5. Epic List (史诗列表)**

*   **Epic 1: 游戏基础框架与主循环 (Game Foundation & Main Loop)**
    *   **目标:** 搭建整个Go+Ebitengine项目的基本结构，创建一个可以运行的空窗口，并实现游戏的核心状态管理和主菜单。这是所有后续功能的基础。
*   **Epic 2: 核心资源与玩家交互 (Core Resources & Player Interaction)**
    *   **目标:** 实现阳光的生成、收集和计数系统，并建立玩家与游戏世界的基础交互，如通过鼠标点击进行操作。
*   **Epic 3: 植物系统与部署 (Planting System & Deployment)**
    *   **目标:** 实现完整的植物种植流程，包括从UI卡片栏选择植物、消耗阳光、放置在草坪上，并处理冷却逻辑。我们将首先实现向日葵和豌豆射手。
*   **Epic 4: 基础僵尸与战斗逻辑 (Basic Zombies & Combat Logic)**
    *   **目标:** 在游戏中引入基础的僵尸（普通僵尸），实现僵尸的移动、植物的自动攻击（豌豆射手）以及子弹与僵尸的碰撞和伤害计算。
*   **Epic 5: 游戏流程与高级单位 (Game Flow & Advanced Units)**
    *   **目标:** 实现完整的关卡流程控制（僵尸波次、进度条），并引入更复杂的单位（坚果墙、樱桃炸弹、路障/铁桶僵尸）来完成MVP的全部核心玩法。
*   **Epic 6: 动画系统迁移 - 原版 Reanim 骨骼动画系统 (Animation System Migration - Reanim)**
    *   **目标:** 将简单帧动画系统直接替换为原版 PVZ 的 Reanim 骨骼动画系统，实现 100% 还原原版动画效果，支持部件变换和复杂动画表现。
*   **Epic 7: 粒子特效系统 (Particle Effect System) - Brownfield Enhancement**
    *   **目标:** 实现完整的粒子特效系统，支持原版PVZ的所有视觉特效（僵尸肢体掉落、爆炸、溅射等），通过解析XML配置和高性能批量渲染技术，为游戏提供丰富的视觉反馈。
*   **Epic 8: 第一章关卡实现 - 前院白天 (Chapter 1: Day Level Implementation)**
    *   **目标:** 实现第一章(前院白天)的全部 10 个关卡,包括原版忠实的关卡机制、教学引导系统(1-1单行草地引导)、开场动画系统(可配置跳过)、选卡界面和特殊关卡类型(1-5坚果保龄球、1-10传送带模式),为玩家提供完整的冒险模式第一章体验。
*   **Epic 9: ECS 框架泛型化重构 (ECS Generics Refactor) - Brownfield Enhancement**
    *   **目标:** 将当前基于反射的 ECS 框架迁移到 Go 泛型实现，通过编译时类型检查和性能优化，消除运行时反射开销，提升代码可读性和类型安全性，为后续开发提供更高效的开发体验。
*   **Epic 10: 游戏体验完善 (Game Experience Polish) - Brownfield Enhancement**
    *   **目标:** 完善核心游戏体验，实现暂停菜单、植物攻击动画、粒子特效、除草车防线系统和音效系统，提升游戏的完整性和可玩性，确保所有核心机制符合原版游戏标准。包含音效系统集成（Story 10.9-10.12）：核心游戏音效、戴夫语音、植物音效和僵尸音效。
*   **Epic 11: 关卡 UI 增强 (Level UI Enhancement)**
    *   **目标:** 完善《植物大战僵尸》关卡界面的视觉体验和信息展示，实现铺草皮土粒飞溅特效、最后一波僵尸提示动画和完整的关卡进度条系统，提升游戏的沉浸感和信息可读性，确保关卡 UI 符合原版游戏标准。
*   **Epic 12: 主菜单系统 (Main Menu System)**
    *   **目标:** 实现完整的《植物大战僵尸》主菜单界面，包括石碑菜单、用户管理、功能按钮、对话框系统和动画效果，为玩家提供直观的游戏导航和沉浸式的视觉体验，完全还原原版游戏的主菜单交互逻辑。
*   **Epic 13: Reanim 动画系统现代化重构 (Reanim Animation System Modernization)**
    *   **目标:** 基于 `cmd/animation_showcase` 中对 Reanim 格式的深入理解和正确实现，重构现有 Reanim 动画系统，引入轨道绑定机制（Track Animation Binding）、父子偏移系统（Parent-Child Offset）和渲染缓存优化，解决多动画组合播放的技术债务，提升系统性能和可维护性。
*   **Epic 14: ECS 系统耦合解除重构 (ECS System Decoupling Refactor)**
    *   **目标:** 解除 Reanim 动画系统与其他游戏系统之间的直接耦合，通过引入组件驱动的动画控制机制（AnimationCommand），使系统间通信完全符合 ECS 架构的零耦合原则，提升代码可维护性、可测试性和架构一致性。
*   **Epic 15: pkg/systems 代码质量改进 (Code Quality Improvement)**
    *   **目标:** 系统性改进 `pkg/systems` 目录的代码质量，通过精简冗余注释、拆分巨婴文件、优化代码结构，提升代码的可读性、可维护性和团队协作效率，确保代码库符合专业工程标准。
*   **Epic 16: Reanim 坐标系统重构 (Coordinate System Refactoring) - Brownfield Enhancement**
    *   **目标:** 重构 Reanim 动画渲染系统的坐标转换逻辑，将散落在多个系统中的重复计算封装为统一的工具库，降低认知负担，提高代码可维护性，减少 50% 的坐标计算相关错误，并完全符合 ECS 零耦合原则。
*   **Epic 17: 僵尸生成引擎 (Zombie Generation Engine)**
    *   **目标:** 实现《植物大战僵尸》冒险模式的精确僵尸生成系统，包括难度动态调整、行分配平滑权重算法、波次计时控制和旗帜波特殊机制，完全还原原版游戏的出怪逻辑和游戏节奏。
*   **Epic 18: 游戏战斗存档系统 (Battle Save System) - Brownfield Enhancement**
    *   **目标:** 为《植物大战僵尸》复刻版实现战斗中存档/读档功能。玩家点击暂停菜单的"主菜单"按钮时自动保存当前战斗进度，再次进入冒险模式时可选择继续游戏或重玩关卡，提供简洁的用户交互体验。
*   **Epic 19: 关卡 1-5 坚果保龄球 (Level 1-5: Wall-nut Bowling)** ⭐**NEW**
    *   **目标:** 实现第一章第五关"坚果保龄球"，包含铲子教学阶段（疯狂戴夫对话、强引导机制、预设植物）和保龄球玩法阶段（传送带系统、保龄球物理、爆炸坚果），为玩家提供独特的迷你游戏体验。
*   **Epic 20: Go Embed 资源嵌入与跨平台存储 (Go Embed & Cross-Platform Storage)** ⭐**NEW**
    *   **目标:** 使用 Go 的 `embed` 功能将游戏资源（assets/ 和 data/）打包进可执行程序，实现单一可执行文件分发；使用 gdata 库实现跨平台用户数据存储（存档、设置），支持桌面端、移动端和 WASM 平台。
