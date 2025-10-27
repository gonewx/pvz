# 植物大战僵尸 Go 复刻版 Product Requirements Document (PRD)

## **1. Goals and Background Context (目标与背景上下文)**

### **Goals (目标)**
*   **主要目标：** 成功掌握使用Go语言和Ebitengine引擎进行2D游戏开发的核心流程和关键技术。
*   **次要目标：** 创建一个功能完整、可独立运行的游戏项目，作为个人作品集中的一个高质量范例。
*   **扩展目标：** 构建一个模块化、可扩展的代码基础，为未来添加更多关卡和游戏模式提供可能性。

### **Background Context (背景上下文)**
本项目是对经典塔防游戏《植物大战僵尸》PC中文年度版的精确复刻，专注于学习和实践。与从零开始的项目不同，本项目的核心挑战在于技术实现而非创意设计。所有必要的游戏美术和音频资源均已备好，使开发团队能将全部精力集中在游戏逻辑、核心机制和引擎技术的应用上。MVP（最小可行产品）范围明确为实现完整的前院白天关卡，为后续内容的添加打下坚实基础。

### **Change Log (变更日志)**
| Date | Version | Description | Author |
| :--- | :--- | :--- | :--- |
| 2025-10-10 | 1.0 | Initial draft creation from Project Brief. | John (PM) |
| 2025-10-13 | 1.1 | Added Epic 6: Animation System Migration - Reanim. | Sarah (PO) |
| 2025-10-15 | 1.2 | Added Epic 7: Particle Effect System - Brownfield Enhancement. | Sarah (PO) |
| 2025-10-16 | 1.3 | Added Epic 8: Chapter 1 Level Implementation - Day Levels 1-1 to 1-10. | Sarah (PO) |
| 2025-10-16 | 1.4 | Added Epic 9: ECS Generics Refactor - Brownfield Enhancement. | Sarah (PO) |
| 2025-10-20 | 1.5 | Added Epic 10: Game Experience Polish - Brownfield Enhancement. | Sarah (PO) |
| 2025-10-26 | 1.6 | Added Epic 11: Level UI Enhancement (进度条完善、最后一波提示、铺草皮特效). | Bob (Scrum Master) |

## **2. Requirements (需求)**

### **Functional (功能性需求)**
*   **FR1: 主菜单系统** - 游戏必须提供一个主菜单界面，包含可交互的“开始冒险”和“退出游戏”按钮。
*   **FR2: 游戏场景加载** - 玩家点击“开始冒险”后，系统必须能加载前院白天场景的草坪网格、背景和UI元素。
*   **FR3: 阳光生成机制** - 系统必须实现两种阳光生成方式：
    *   FR3.1: 天空中会周期性地随机掉落阳光。
    *   FR3.2: 玩家种植的向日葵会周期性地生产阳光。
*   **FR4: 阳光收集与计数** - 玩家必须能通过点击阳光来收集它们，并在UI的阳光计数器上实时更新总数。
*   **FR5: 植物选择与冷却** - 游戏界面必须展示一个植物选择栏，包含MVP范围内的植物卡片（向日葵、豌豆射手、坚果墙、樱桃炸弹）。选择一个植物后，该卡片进入冷却倒计时，期间不可用。
*   **FR6: 植物种植** - 玩家必须能在花费足够阳光的前提下，将选择的植物种植在草坪的有效格子内。
*   **FR7: 植物行为** - 已种植的植物必须能执行其核心行为：
    *   FR7.1: **向日葵** - 按预定周期生产阳光。
    *   FR7.2: **豌豆射手** - 当同一行出现僵尸时，能自动发射豌豆子弹进行攻击。
    *   FR7.3: **坚果墙** - 能阻挡僵尸前进，并具有独立的生命值。
    *   FR7.4: **樱桃炸弹** - 种植后短暂延迟即爆炸，消灭周围格子内的所有僵尸。
*   **FR8: 僵尸生成** - 系统必须能根据关卡设计，从屏幕右侧生成一波又一波的僵尸，包括普通、路障和铁桶僵尸。
*   **FR9: 僵尸行为** - 僵尸必须能执行其核心行为：
    *   FR9.1: 沿固定行从右向左移动。
    *   FR9.2: 遇到植物时，会停下来进行啃食。
    *   FR9.3: 具有独立的生命值，当生命值降为零时会死亡并消失。
    *   FR9.4: 路障和铁桶僵尸在防护物被摧毁前具有更高的防御力。
*   **FR10: 伤害与碰撞** - 必须实现豌豆子弹与僵尸的碰撞检测，并根据攻击力计算对僵尸造成的伤害。
*   **FR11: 铲子功能** - 玩家必须能使用铲子工具，移除草坪上任意一个已种植的植物。
*   **FR12: 游戏进程** - 游戏必须能显示关卡进度条，并在最后一波僵尸来临时给予提示。
*   **FR13: 胜利/失败条件** - 必须实现明确的游戏结束条件：
    *   FR13.1: **胜利** - 玩家成功消灭当前关卡的所有僵尸。
    *   FR13.2: **失败** - 任意一个僵尸到达屏幕最左侧的房子。

### **Non-Functional (非功能性需求)**
*   **NFR1: 性能** - 游戏在满负荷场景（大量植物、僵尸和子弹同屏）下，必须保持流畅运行（目标60 FPS）。
*   **NFR2: 忠实度** - 所有的游戏数值（如植物攻击力、僵尸生命值、阳光值、冷却时间）和行为节奏都应与原版PC游戏保持高度一致。
*   **NFR3: 平台兼容性** - 游戏必须能通过Ebitengine成功打包，并在Windows、macOS和Linux平台上独立运行。
*   **NFR4: 可扩展性** - 代码架构必须是模块化的，以便在MVP完成后，能以最小的代价添加新的植物、僵尸或游戏场景。

## **3. User Interface Design Goals (用户界面设计目标)**

### **Overall UX Vision (整体用户体验愿景)**
整体用户体验旨在精确复制原版PC游戏的核心交互逻辑和视觉反馈。玩家应获得与2009年原版游戏几乎完全一致的、直观且令人怀念的塔防游戏体验。所有的UI元素都应服务于核心玩法，信息清晰易读，操作流畅直接。

### **Key Interaction Paradigms (关键交互模式)**
*   **直接操控:** 玩家通过鼠标点击与所有游戏元素进行交互（收集阳光、选择卡片、种植植物、使用铲子）。
*   **即时反馈:** 每一次有效的玩家操作都应有即时的视觉和听觉反馈（如点击阳光的“噗”声和消失动画，选择卡片时的音效）。
*   **状态可见性:** 游戏的核心状态（阳光数量、植物卡片冷却状态、关卡进度）必须始终清晰可见。

### **Core Screens and Views (核心屏幕与视图)**
*   **主菜单 (Main Menu):** 包含墓碑背景，提供“开始冒险”和“退出游戏”的选项。
*   **游戏主界面 (In-Game UI):** 这是核心视图，包含：
    *   **植物选择栏:** 位于屏幕左上角，横向排列所有可用的植物卡片。
    *   **阳光计数器:** 位于植物选择栏左侧，实时显示玩家拥有的阳光数量。
    *   **游戏区域:** 中央的5x9草坪网格。
    *   **铲子:** 位于植物选择栏下方，用于移除植物。
    *   **关卡进度条:** 位于屏幕右下角，显示当前僵尸波次和旗帜。
*   **游戏胜利/失败界面 (Win/Loss Screen):** 游戏结束后弹出的信息界面，宣告游戏结果。

### **Accessibility (可访问性)**
*   **目标:** 无特定增强要求，与原版保持一致。主要通过清晰的视觉设计和独特的音效来区分不同事件。

### **Branding (品牌)**
*   **目标:** 100%忠实于原版《植物大战僵尸》的品牌风格，使用已备好的所有官方UI素材、字体（或类似字体）和Logo。

### **Target Device and Platforms (目标设备与平台)**
*   **目标:** Web Responsive (响应式网页)。虽然优先支持PC，但采用响应式设计将使未来移植到不同分辨率或平台的可能性更大，同时也能在不同大小的PC窗口中有良好表现。

## **4. Technical Assumptions (技术假设)**

### **Repository Structure: Monorepo (单体仓库)**
*   我们将采用单体仓库（Monorepo）的结构。尽管当前项目只有一个核心应用，但这种结构为未来可能的扩展（如独立的关卡编辑器、后端服务等）提供了便利，并且便于管理共享代码和配置。

### **Service Architecture (服务架构)**
*   **架构:** Monolith (单体应用)。
*   **说明:** 整个游戏将作为一个独立的、单一的编译后可执行文件。所有游戏逻辑、渲染和状态管理都在这一个进程中完成。这对于一个客户端游戏来说是标准且最高效的架构，完全符合我们的学习目标。

### **Testing Requirements (测试要求)**
*   **要求:** Unit + Integration (单元测试 + 集成测试)。
*   **说明:** 我们将对核心的、无副作用的游戏逻辑（例如伤害计算、阳光生成算法）编写单元测试，以确保其正确性。对于模块间的交互（例如植物与僵尸的互动），我们将通过集成测试来验证。这有助于我们学习如何在Go中编写可测试的游戏代码。

### **Additional Technical Assumptions and Requests (其他技术假设与要求)**
*   **语言和引擎:** 严格使用Go语言和Ebitengine游戏引擎的最新稳定版本。
*   **依赖最小化:** 优先使用Go标准库和Ebitengine内置功能，谨慎引入第三方依赖，以保持项目简洁和聚焦。
*   **数据驱动:** 鼓励采用数据驱动的设计。植物、僵尸和关卡的属性（如生命值、攻击力、生成顺序）应从外部配置文件（如JSON或YAML）加载，而不是硬编码在代码中。这极大地简化了数值调整和未来内容的扩展。
*   **平台:** 优先为PC平台（Windows, macOS, Linux）构建，但代码实现应考虑跨平台兼容性，为未来可能的Web（WASM）或移动端移植做准备。

## **5. Epic List (史诗列表)**

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
    *   **目标:** 实现《植物大战僵尸》第一章(前院白天)的全部 10 个关卡,包括原版忠实的关卡机制、教学引导系统(1-1单行草地引导)、开场动画系统(可配置跳过)、选卡界面和特殊关卡类型(1-5坚果保龄球、1-10传送带模式),为玩家提供完整的冒险模式第一章体验。详见 [Epic 8 详细文档](prd/epic-8-level-chapter1-implementation.md)。
*   **Epic 9: ECS 框架泛型化重构 (ECS Generics Refactor) - Brownfield Enhancement**
    *   **目标:** 将当前基于反射的 ECS 框架迁移到 Go 泛型实现，通过编译时类型检查和性能优化，消除运行时反射开销，提升代码可读性和类型安全性，为后续开发提供更高效的开发体验。详见 [Epic 9 详细文档](prd/epic-9-ecs-generics-refactor.md)。
*   **Epic 10: 游戏体验完善 (Game Experience Polish) - Brownfield Enhancement**
    *   **目标:** 完善核心游戏体验，实现暂停菜单、植物攻击动画、粒子特效和除草车防线系统，提升游戏的完整性和可玩性，确保所有核心机制符合原版游戏标准。详见 [Epic 10 详细文档](prd/epic-10-game-experience-polish.md)。
*   **Epic 11: 关卡 UI 增强 (Level UI Enhancement)**
    *   **目标:** 完善《植物大战僵尸》关卡界面的视觉体验和信息展示，实现铺草皮土粒飞溅特效、最后一波僵尸提示动画和完整的关卡进度条系统，提升游戏的沉浸感和信息可读性，确保关卡 UI 符合原版游戏标准。详见 [Epic 11 详细文档](prd/epic-11-level-ui-enhancement.md)。

## **Epic 1: 游戏基础框架与主循环 (Game Foundation & Main Loop)**
**史诗目标:** 搭建整个Go+Ebitengine项目的基本结构，创建一个可以运行的空窗口，并实现游戏的核心状态管理和主菜单。这是所有后续功能的基础。

---
**Story 1.1: 项目初始化与窗口创建**
> **As a** 开发者,
> **I want** to set up the Go project structure and initialize an Ebitengine application,
> **so that** I have a running window which will serve as the canvas for the game.

**Acceptance Criteria:**
1.  项目遵循Go Modules的标准结构。
2.  Ebitengine被成功添加为项目依赖。
3.  运行 `go run .` 指令后，屏幕上会显示一个固定大小（例如800x600像素）的空白窗口。
4.  窗口标题应设置为"植物大战僵尸 - Go复刻版"。
5.  可以通过点击窗口的关闭按钮正常退出应用程序。

---
**Story 1.2: 游戏状态机与场景管理**
> **As a** 开发者,
> **I want** to implement a basic game state machine,
> **so that** the game can switch between different scenes like 'Main Menu' and 'In-Game'.

**Acceptance Criteria:**
1.  存在一个游戏状态管理器（例如 `SceneManager`）。
2.  定义了至少两种场景状态：`MainMenuScene` 和 `GameScene`。
3.  游戏启动时，默认进入并显示 `MainMenuScene`。
4.  `SceneManager` 提供了切换场景的功能（例如 `SwitchToScene(...)`）。
5.  每个场景都有自己的 `Update` 和 `Draw` 逻辑。

---
**Story 1.3: 资源管理器框架**
> **As a** 开发者,
> **I want** to create a resource manager that can load image and audio files,
> **so that** game assets can be centrally managed and accessed by any part of the game.

**Acceptance Criteria:**
1.  存在一个资源管理器（例如 `ResourceManager`），在游戏启动时初始化。
2.  可以成功加载一个PNG格式的图片文件（例如主菜单背景）并在场景中使用。
3.  可以成功加载一个音频文件（例如主菜单背景音乐）并循环播放。
4.  如果资源加载失败，游戏应能打印错误日志而不是直接崩溃。
5.  资源应只被加载一次，并在内存中重复使用。

---
**Story 1.4: 主菜单UI与交互**
> **As a** 玩家,
> **I want** to see and interact with the main menu,
> **so that** I can start the game or exit.

**Acceptance Criteria:**
1.  主菜单场景 (`MainMenuScene`) 必须显示正确的背景图片。
2.  主菜单必须显示"开始冒险"和"退出游戏"两个按钮（暂时可以是文字或简单图形）。
3.  鼠标悬停在按钮上时，按钮有视觉变化（例如变色或放大）。
4.  点击"开始冒险"按钮后，游戏状态会通过 `SceneManager` 切换到 `GameScene`。
5.  点击"退出游戏"按钮后，应用程序会正常关闭。

## **Epic 2: 核心资源与玩家交互 (Core Resources & Player Interaction)**
**史诗目标:** 实现阳光的生成、收集和计数系统，并建立玩家与游戏世界的基础交互，如通过鼠标点击进行操作。

---
**Story 2.1: 游戏场景UI框架**
> **As a** 开发者,
> **I want** to load and display the basic in-game UI elements for the daytime lawn scene,
> **so that** I have the foundational layout for placing game state information.

**Acceptance Criteria:**
1.  `GameScene` 启动时，会绘制正确的草坪背景。
2.  屏幕左上角会绘制植物选择栏的背景框。
3.  植物选择栏左侧会绘制阳光计数器的背景框。
4.  铲子的图标和背景会绘制在植物选择栏下方。
5.  所有UI元素的位置和大小应与原版游戏布局一致。

---
**Story 2.2: 游戏全局状态管理**
> **As a** 开发者,
> **I want** to create a central game state manager,
> **so that** I can track and modify global variables like the player's current sun count.

**Acceptance Criteria:**
1.  存在一个全局可访问的游戏状态实例（例如 `GameState`）。
2.  `GameState` 包含一个 `Sun` 字段，用于存储当前阳光数量，初始值为50。
3.  UI上的阳光计数器能读取并正确显示 `GameState.Sun` 的值。
4.  提供增加和减少阳光的方法（例如 `AddSun(amount)` 和 `SpendSun(amount)`）。
5.  当调用 `AddSun` 或 `SpendSun` 后，UI上的阳光计数器会实时更新。

---
**Story 2.3: 自然阳光掉落**
> **As a** 玩家,
> **I want** to see suns periodically fall from the sky,
> **so that** I can collect them as a primary resource.

**Acceptance Criteria:**
1.  游戏开始后，会按照一定的时间间隔（例如5-10秒）从屏幕顶部的随机位置生成一个阳光单位。
2.  阳光单位会以平滑的动画垂直下落，并停留在草坪区域内的一个随机位置。
3.  阳光单位在地面上停留一段时间后（例如8秒），如果没有被收集，会自动消失。
4.  同一时间屏幕上可以存在多个掉落的阳光。

---
**Story 2.4: 阳光收集**
> **As a** 玩家,
> **I want** to be able to click on suns to collect them,
> **so that** I can increase my sun resource count.

**Acceptance Criteria:**
1.  当鼠标点击一个阳光单位时，该阳光单位会播放一个飞向左上角阳光计数器的动画。
2.  当动画结束时，阳光单位从屏幕上消失。
3.  同时，全局的阳光数量会增加25（`GameState.AddSun(25)`）。
4.  阳光单位在被点击后，就不能再次被点击。
5.  收集阳光时会播放正确的音效。

## **Epic 3: 植物系统与部署 (Planting System & Deployment)**
**史诗目标:** 实现完整的植物种植流程，包括从UI卡片栏选择植物、消耗阳光、放置在草坪上，并处理冷却逻辑。我们将首先实现向日葵和豌豆射手。

---
**Story 3.1: 植物卡片UI与状态**
> **As a** 玩家,
> **I want** to see the plant cards in the selection bar,
> **so that** I know which plants are available for planting.

**Acceptance Criteria:**
1.  植物选择栏中会正确显示向日葵和豌豆射手的卡片图像。
2.  每个卡片下方会显示种植所需的阳光数量（向日葵:50, 豌豆射手:100）。
3.  当玩家阳光数量不足时或处于冷却状态时，对应的植物卡片会变暗或显示为不可用状态。
4.  当一个植物卡片处于冷却状态时， 卡片会被比阳光不足时的更深的半透明黑色覆盖，并从下到上渐渐恢复，模拟冷却计时器。
5.  当冷却结束后，判断阳光数量如果足够，卡片恢复正常可用状态，如果阳光不够，保持变暗状态。

---
**Story 3.2: 植物选择与跟随**
> **As a** 玩家,
> **I want** to be able to click on an available plant card,
> **so that** I can prepare to plant it on the lawn.

**Acceptance Criteria:**
1.  当玩家点击一个可用（阳光充足且非冷却）的植物卡片时，鼠标指针会变成该植物的半透明图像。
2.  同时，游戏进入“种植模式”，全局游戏状态会记录当前被选择的植物类型。
3.  当鼠标在草坪网格上移动时，半透明的植物图像会跟随鼠标，并自动对齐到鼠标所在的格子中心。
4.  此时，再次点击植物卡片栏或点击鼠标右键，可以取消“种植模式”，鼠标恢复正常指针。

---
**Story 3.3: 植物种植到草坪**
> **As a** 玩家,
> **I want** to be able to place the selected plant onto a valid grid cell,
> **so that** it appears on the lawn and starts performing its function.

**Acceptance Criteria:**
1.  在“种植模式”下，当玩家在草坪的一个空格子上点击鼠标左键时，会在该格子种下一个新的植物实例。
2.  种植成功后，玩家的阳光总数会扣除该植物对应的消耗值。
3.  同时，该植物的卡片会立即进入冷却状态。
4.  游戏退出“种植模式”，鼠标恢复正常。
5.  种植时会播放正确的音效。
6.  不能在已经有植物的格子上种植新的植物。

---
**Story 3.4: 向日葵行为实现**
> **As a** 开发者,
> **I want** to implement the behavior for the Sunflower,
> **so that** it can produce suns for the player.

**Acceptance Criteria:**
1.  向日葵被种植后，会开始一个内部计时器。
2.  当计时器到达预定周期后，向日葵会在自身附近生成一个阳光单位。
3.  该阳光单位的行为与天生掉落的阳光完全一致（可以被点击收集）。
4.  生产阳光后，向日葵的内部计时器会重置。
5.  向日葵会播放其“生产阳光”的动画。

## **Epic 4: 基础僵尸与战斗逻辑 (Basic Zombies & Combat Logic)**
**史诗目标:** 在游戏中引入基础的僵尸（普通僵尸），实现僵尸的移动、植物的自动攻击（豌豆射手）以及子弹与僵尸的碰撞和伤害计算。

---
**Story 4.1: 基础僵尸生成与移动**
> **As a** 开发者,
> **I want** to spawn a basic zombie that moves from right to left,
> **so that** the player has an antagonist to defend against.

**Acceptance Criteria:**
1.  游戏可以从屏幕右侧边缘之外的特定行生成一个普通僵尸。
2.  僵尸生成后，会以恒定的速度沿其所在的行从右向左移动。
3.  僵尸会播放其走路的动画。
4.  僵尸移动时，其视觉层次应正确处理（例如，不会覆盖在上方的UI元素上）。

---
**Story 4.2: 豌豆射手行为实现**
> **As a** 开发者,
> **I want** to implement the behavior for the Peashooter,
> **so that** it can attack zombies in its lane.

**Acceptance Criteria:**
1.  豌豆射手被种植后，会周期性地扫描其所在的行。
2.  当其正前方（右侧）的同一行出现了僵尸时，豌豆射手会进入攻击状态。
3.  在攻击状态下，豌豆射手会按照固定的时间间隔，从其口部发射豌豆子弹。
4.  发射子弹时，豌豆射手会播放其攻击动画。
5.  如果没有僵尸在其行上，豌豆射手会保持静止（idle）状态。

---
**Story 4.3: 子弹移动与碰撞**
> **As a** 开发者,
> **I want** to make the pea projectile move and detect collisions with zombies,
> **so that** damage can be dealt.

**Acceptance Criteria:**
1.  豌豆子弹从豌豆射手处被发射后，会以恒定的速度沿直线向右移动。
2.  子弹需要有一个碰撞体（例如矩形）。
3.  当子弹的碰撞体与一个僵尸的碰撞体发生重叠时，判定为命中。
4.  命中后，子弹会消失，并播放一个“击中”的视觉效果（例如一个水花）。
5.  子弹飞出屏幕范围后会自动销毁，以避免内存泄漏。

---
**Story 4.4: 僵尸生命值与死亡**
> **As a** 开发者,
> **I want** zombies to have health and be defeated,
> **so that** the player's defense is meaningful.

**Acceptance Criteria:**
1.  每个僵尸实例都有一个独立的生命值（Health Points）。
2.  当一个僵尸被豌豆子弹命中时，其生命值会减少一个固定的数值。
3.  当僵尸生命值降到0或以下时，僵尸会死亡。
4.  僵尸死亡时，会先播放其死亡动画（例如头掉下来），动画播放完毕后，僵尸对象会从游戏中移除。
5.  击中僵尸时会播放正确的音效。

## **Epic 5: 游戏流程与高级单位 (Game Flow & Advanced Units)**
**史诗目标:** 实现完整的关卡流程控制（僵尸波次、进度条），并引入更复杂的单位（坚果墙、樱桃炸弹、路障/铁桶僵尸）来完成MVP的全部核心玩法。

---
**Story 5.1: 僵尸啃食与植物生命值**
> **As a** 僵尸,
> **I want** to stop and eat plants I encounter,
> **so that** I can clear a path to the player's house.

**Acceptance Criteria:**
1.  所有植物（向日葵、豌豆射手等）现在都拥有独立的生命值。
2.  当僵尸移动到与植物同一个格子时，它会停止移动并开始播放啃食动画。
3.  在啃食状态下，僵尸会周期性地对植物造成伤害，减少植物的生命值。
4.  当植物生命值降到0或以下时，植物会从草坪上消失。
5.  植物被消灭后，僵尸会继续向左移动。
6.  啃食植物时会播放正确的音效。

---
**Story 5.2: 坚果墙行为实现**
> **As a** 玩家,
> **I want** to plant a Wall-nut with high health,
> **so that** I can effectively block zombies.

**Acceptance Criteria:**
1.  玩家可以在植物选择栏中选择并种植坚果墙。
2.  坚果墙被种植后，会保持在原地，没有攻击行为。
3.  坚果墙拥有比其他植物高得多的生命值。
4.  坚果墙会根据其剩余生命值百分比，显示不同的受损外观（例如出现裂痕）。
5.  当生命值被耗尽时，坚果墙会消失。

---
**Story 5.3: 高级僵尸行为实现**
> **As a** 玩家,
> **I want** to face tougher zombies like Conehead and Buckethead,
> **so that** the game provides a greater challenge.

**Acceptance Criteria:**
1.  游戏中可以生成路障僵尸和铁桶僵尸。
2.  路障僵尸和铁桶僵尸拥有一个额外的“头部防具”生命值层。
3.  当僵尸受到伤害时，优先扣除其“头部防具”的生命值。
4.  当“头部防具”生命值耗尽时，其外观会改变（路障/铁桶掉落），并且僵尸的行为和生命值会变为一个普通僵尸。
5.  这两种僵尸的总有效生命值远高于普通僵尸。

---
**Story 5.4: 樱桃炸弹行为实现**
> **As a** 玩家,
> **I want** to use a Cherry Bomb to clear a group of zombies,
> **so that** I can handle emergency situations.

**Acceptance Criteria:**
1.  玩家可以在植物选择栏中选择并种植樱桃炸弹。
2.  种植后，樱桃炸弹会有一个短暂的“引信”动画。
3.  动画结束后，樱桃炸弹会发生爆炸，并立即从游戏中消失。
4.  爆炸会对以自身为中心的3x3范围内的所有僵尸造成巨量伤害，足以消灭MVP范围内的任何僵尸。
5.  爆炸时会播放正确的视觉和声音效果。

---
**Story 5.5: 关卡流程管理**
> **As a** 玩家,
> **I want** to experience a structured level with waves of zombies and a clear goal,
> **so that** the game feels like a complete experience.

**Acceptance Criteria:**
1.  游戏会从一个外部配置文件（例如 `level-1-1.json`）加载关卡数据。
2.  关卡数据定义了僵尸出现的波次、每一波的僵尸类型和数量。
3.  游戏界面右下角会有一个关卡进度条，显示当前波次和总波次。
4.  在最后一波僵尸到来之前，屏幕上会出现"A huge wave of zombies is approaching"的提示。
5.  当玩家消灭所有波次的僵尸后，游戏会暂停，并显示胜利界面。
6.  如果任何一个僵尸走到了屏幕最左端，游戏会立即暂停，并显示失败界面。

## **Epic 6: 动画系统迁移 - 原版 Reanim 骨骼动画系统 (Animation System Migration - Reanim)**
**史诗目标:** 将当前基于简单帧数组的动画系统直接替换为原版 PVZ 的 Reanim 骨骼动画系统，实现 100% 还原原版动画效果，支持部件变换和复杂动画表现。

**背景:** 现有的简单帧数组动画系统无法实现原版 PVZ 的精细动画效果。项目已准备好原版 Reanim 资源（`assets/effect/reanim/*.reanim` 和部件图片），并有参考实现可供借鉴。本 Epic 采用直接替换策略，不考虑向后兼容性，以简化实现。

---
**Story 6.1: Reanim 基础设施（解析器和资源加载）**
> **As a** 开发者,
> **I want** to parse Reanim XML files and load sprite parts,
> **so that** I can use original PVZ animation data.

**Acceptance Criteria:**
1.  实现 `ReanimXML`, `Track`, `Frame` 数据结构（对应原版格式）。
2.  实现 XML 解析器 `internal/reanim/parser.go`。
3.  解析器能成功解析至少 3 个 Reanim 文件（PeaShooter, Sunflower, Wallnut）。
4.  实现资源加载器扩展，支持按 Reanim 定义加载部件图片。
5.  单元测试覆盖率 ≥ 80%。
6.  与参考实现 `test_animation_viewer.go` 的数据结构对比验证。

---
**Story 6.2: ReanimComponent 和 ReanimSystem（核心动画逻辑）**
> **As a** 开发者,
> **I want** to replace the old animation system with Reanim system,
> **so that** entities can use complex skeletal animations.

**Acceptance Criteria:**
1.  **删除** `pkg/components/animation.go` 和 `pkg/systems/animation_system.go`（旧动画系统）。
2.  **创建** `pkg/components/reanim_component.go`（新组件，包含部件图片映射和动画状态）。
3.  **创建** `pkg/systems/reanim_system.go`（新系统，实现动画播放逻辑）。
4.  实现帧推进、循环、FPS 控制逻辑。
5.  实现帧缓存机制（处理空帧继承，原版特性）。
6.  单元测试覆盖率 ≥ 80%。

---
**Story 6.3: 渲染系统改造和实体迁移**
> **As a** 开发者,
> **I want** to update the render system and all entities to use Reanim,
> **so that** the game displays animations with original PVZ quality.

**Acceptance Criteria:**
1.  **修改** `pkg/systems/render_system.go`，删除旧渲染逻辑，实现 Reanim 部件渲染。
2.  支持部件变换：位置（X, Y）、缩放（ScaleX, ScaleY）、倾斜（SkewX, SkewY）。
3.  **更新** 所有实体工厂函数（豌豆射手、向日葵、坚果墙、僵尸等）使用 `ReanimComponent`。
4.  游戏可正常启动，显示主菜单和游戏场景。
5.  关卡 1-1 可正常运行，所有动画流畅播放（60 FPS）。
6.  动画效果与参考实现 `test_animation_viewer` 一致。

## **6. Checklist Results Report (清单检查结果报告)**

*   **审查开始...**
*   **对照清单:** `.bmad-core/checklists/pm-checklist.md`
*   **1. 问题定义与上下文:** ✅ (清晰，继承自项目简报)
*   **2. MVP范围定义:** ✅ (核心功能和范围外内容都非常明确)
*   **3. 用户体验需求:** ✅ (明确定义为“一比一复刻”)
*   **4. 功能性需求:** ✅ (FR/NFR列表完整且具体)
*   **5. 史诗与故事结构:** ✅ (史诗逻辑递进，故事分解合理)
*   **6. 技术指导:** ✅ (技术假设清晰，为架构师提供了明确方向)
*   **审查结论:** **READY FOR ARCHITECT**. PRD内容全面，结构合理，已达到可以交付给架构师进行技术设计的标准。

## **Epic 7: 粒子特效系统 (Particle Effect System) - Brownfield Enhancement**

**史诗目标:** 实现完整的粒子特效系统，支持原版PVZ的所有视觉特效（僵尸肢体掉落、爆炸、溅射等），通过解析XML配置和高性能批量渲染技术，为游戏提供丰富的视觉反馈。

### **Story 7.1: 粒子配置解析系统**
> **As a** 开发者,
> **I want** to parse particle XML configuration files and load particle textures,
> **so that** I can use original PVZ particle effect data.

**Acceptance Criteria:**
1. 实现粒子配置数据结构：`ParticleConfig`, `EmitterConfig`, `Field`等
2. 实现XML解析器 `internal/particle/parser.go`
3. 解析器能成功解析至少5个代表性粒子配置（Award.xml, BossExplosion.xml, CabbageSplat.xml等）
4. 支持40+种粒子属性的解析（Spawn*, Particle*, Launch*, Field等）
5. 扩展`ResourceManager`，添加`LoadParticleConfig(name string)`方法
6. 粒子贴图加载集成到`ResourceManager`
7. 单元测试覆盖率 ≥ 80%
8. 解析结果与XML配置一致性验证

---

### **Story 7.2: 粒子发射器核心引擎**
> **As a** 开发者,
> **I want** to implement particle emitter logic and lifecycle management,
> **so that** particles can be spawned, updated, and destroyed correctly.

**Acceptance Criteria:**
1. **创建**: `pkg/components/particle_component.go`（粒子组件）
   - 存储单个粒子的状态（位置、速度、旋转、缩放、透明度、颜色、生命周期等）
   - 纯数据组件，无方法
2. **创建**: `pkg/components/emitter_component.go`（发射器组件）
   - 存储发射器配置引用和状态
   - 管理发射时机、活跃粒子数等
3. **创建**: `pkg/systems/particle_system.go`（粒子系统）
   - 实现`Update(dt float64)`方法：更新所有粒子和发射器
   - 粒子生命周期管理：生成、更新、销毁
   - 动画插值系统：支持线性插值和关键帧动画（Alpha, Scale, Spin等）
   - 力场系统：支持重力、摩擦、加速度等
4. 实现发射器逻辑：
   - 按配置生成粒子（SpawnMinActive, SpawnRate等）
   - 发射速度和角度（LaunchSpeed, LaunchAngle）
   - 发射区域（EmitterBox, EmitterRadius, EmitterType）
5. 单元测试：验证粒子生命周期、动画插值、力场计算正确性
6. 单元测试覆盖率 ≥ 80%

---

### **Story 7.3: 粒子渲染系统与ECS集成**
> **As a** 开发者,
> **I want** to render particles efficiently using Ebitengine DrawTriangles,
> **so that** the game can display thousands of particles at 60 FPS.

**Acceptance Criteria:**
1. **修改**: `pkg/systems/render_system.go`
   - 添加`DrawParticles(screen *ebiten.Image, cameraX float64)`方法
   - 使用`ebiten.DrawTriangles`批量渲染粒子（每个粒子2个三角形）
   - 支持加法混合模式（Additive Blending）：
     ```go
     op.Blend = ebiten.Blend{
         BlendFactorSourceRGB:      ebiten.BlendFactorOne,
         BlendFactorDestinationRGB: ebiten.BlendFactorOne,
         BlendOperationRGB:         ebiten.BlendOperationAdd,
         // ...
     }
     ```
   - 支持粒子属性：位置、旋转、缩放、透明度、颜色、亮度
   - 按粒子配置的混合模式渲染（Additive vs Normal）
2. **优化**: 复用顶点数组，避免每帧重新分配内存
3. **优化**: 关闭粒子贴图的mipmaps（如果需要）
4. **集成**: 粒子渲染插入到正确的渲染层级（在游戏世界和UI之间）
5. **创建**: 粒子实体工厂函数 `pkg/entities/particle_factory.go`
   - `CreateParticleEffect(name string, x, y float64)`
6. 性能测试：1000个粒子同屏渲染，保持60 FPS
7. 视觉验证：粒子渲染效果与原版PVZ一致（加法混合、颜色混合等）

---

### **Story 7.4: 游戏效果集成与事件触发**
> **As a** 玩家,
> **I want** to see particle effects when zombies die, explosions occur, and projectiles hit,
> **so that** I get rich visual feedback during gameplay.

**Acceptance Criteria:**
1. **修改**: `pkg/systems/behavior_system.go`
   - 僵尸死亡时触发粒子效果（手臂掉落、头掉落等）
   - 樱桃炸弹爆炸时触发爆炸粒子效果
2. **修改**: `pkg/systems/collision_system.go`（如存在）或相关系统
   - 豌豆击中僵尸时触发溅射粒子效果
   - 卷心菜击中时触发卷心菜溅射效果
3. **实现**: 至少5种游戏粒子效果：
   - 僵尸死亡效果（ZombieDeath.xml或类似）
   - 爆炸效果（BossExplosion.xml或类似）
   - 豌豆溅射（PeaSplat.xml或类似）
   - 卷心菜溅射（CabbageSplat.xml）
   - 奖励光效（Award.xml或类似）
4. **创建**: 粒子效果触发辅助函数：
   - `SpawnParticleEffect(effectName string, worldX, worldY float64)`
5. 集成测试：游戏运行时，上述场景能正确触发粒子效果
6. 集成测试：粒子效果播放完成后自动清理，无内存泄漏
7. 性能测试：战斗场景（多僵尸死亡、多爆炸）保持60 FPS
8. 视觉验证：效果与原版PVZ一致

---

## **Epic 8: 第一章关卡实现 - 前院白天 (Chapter 1: Day Level Implementation)**

**史诗目标:** 实现《植物大战僵尸》第一章(前院白天)的全部 10 个关卡,包括原版忠实的关卡机制、教学引导系统、开场动画系统、选卡界面和特殊关卡类型,为玩家提供完整的冒险模式第一章体验。

### Epic Overview (史诗概览)

Epic 8 将实现第一章的完整关卡体验,从教学关卡 1-1 到终极挑战关卡 1-10,包括两种特殊关卡类型(坚果保龄球和传送带模式)。

**关卡清单:**

| 关卡 | 场地 | 旗帜 | 新植物 | 新僵尸 | 特殊机制 |
|------|------|------|--------|--------|----------|
| **1-1** | 1行(第3行) | 无 | 豌豆射手 | 普通僵尸 | 教学引导 ⭐ |
| **1-2** | 3行(中间) | 1面 | 向日葵 | - | 标准开场 |
| **1-3** | 3行(中间) | 1面 | 樱桃炸弹 | 路障僵尸 | 标准开场 |
| **1-4** | 5行(完整) | 1面 | 坚果墙 | - | 解锁铲子 |
| **1-5** | 5行(完整) | 无 | 土豆雷 | - | **坚果保龄球** ⭐ |
| **1-6** | 5行(完整) | 2面 | 寒冰射手 | 撑杆僵尸 | 标准开场 |
| **1-7** | 5行(完整) | 2面 | 大嘴花 | - | 标准开场 |
| **1-8** | 5行(完整) | 2面 | 双发射手 | 铁桶僵尸 | 难度拐点 |
| **1-9** | 5行(完整) | 2面 | - | - | 综合挑战 + 僵尸来信 |
| **1-10** | 5行(完整) | 无 | 小喷菇 | - | **传送带模式** ⭐ |

**关键系统:**
1. **教学引导系统** - 1-1 关卡的强制引导机制
2. **关卡奖励动画系统** - 卡片包掉落动画、新植物介绍面板（Story 8.3 Phase 1）
3. **开场动画系统** - 镜头平移、僵尸预告(可配置跳过)（Story 8.3 Phase 2）
4. **选卡界面** - 植物选择、解锁系统
5. **特殊关卡系统** - 坚果保龄球、传送带模式
6. **关卡进度系统** - 解锁管理、进度保存

### Stories (故事列表)

Epic 8 包含 10 个 Story（含 2 个技术债务 Story）:

1. **Story 8.1**: 关卡配置系统增强与选卡界面 ✅
2. **Story 8.2**: 关卡 1-1 教学引导系统 ✅
3. **Story 8.3**: 关卡完成流程与开场动画系统 ✅
   - Phase 1: 关卡奖励动画系统（卡片包掉落、新植物介绍面板）
   - Phase 2: 开场动画系统与镜头控制（镜头平移、僵尸预告）
4. **Story 8.4**: 植物卡片渲染模块化重构（Technical Debt）✅
5. **Story 8.5**: 粒子系统渲染层级分离与ECS架构优化（Technical Debt）✅
6. **Story 8.6**: 关卡 1-2 至 1-4 实现(标准教学关卡) 📝
7. **Story 8.7**: 特殊关卡类型系统 - 坚果保龄球(1-5) 📝
8. **Story 8.8**: 关卡 1-6 至 1-9 实现(2面旗帜标准关卡) 📝
9. **Story 8.9**: 特殊关卡类型系统 - 传送带模式(1-10) 📝
10. **Story 8.10**: 第一章集成测试与调优 📝

**详细设计文档**: 完整的 Epic 8 规范请参见 [Epic 8 详细文档](prd/epic-8-level-chapter1-implementation.md)。

### Success Criteria (成功标准)

1. ✅ 所有 10 个关卡可玩并符合原版体验(参考 `.meta/levels/chapter1.md`)
2. ✅ 教学系统在 1-1 正常工作(1行草地、强制引导、2-3只僵尸)
3. ✅ 开场动画系统可工作且可跳过(skipOpening 配置)
4. ✅ 选卡界面功能完整
5. ✅ 特殊关卡正确实现(1-5 坚果保龄球、1-10 传送带模式)
6. ✅ 旗帜系统正确工作(1-2~1-4: 1面, 1-6~1-9: 2面)
7. ✅ 关卡进度系统能保存和加载
8. ✅ 新僵尸机制正确(撑杆僵尸跳跃、铁桶僵尸高血量)
9. ✅ 代码通过测试(单元测试 + 集成测试, 覆盖率 80%+)
10. ✅ 性能符合要求(60 FPS)

### Technical Implementation (技术实现)

**新增系统:**
- `TutorialSystem` - 教学流程管理
- `PlantSelectionSystem` - 选卡界面管理
- `CameraSystem` - 镜头控制
- `OpeningAnimationSystem` - 开场编排
- `SpecialLevelRuleSystem` - 特殊关卡规则
- `ConveyorSystem` - 传送带机制

**新增组件:**
- `TutorialComponent` - 教学状态
- `PlantSelectionComponent` - 选卡状态
- `CameraComponent` - 镜头动画
- `ConveyorComponent` - 传送带状态

**配置扩展示例:**
```yaml
# data/levels/level-1-1.yaml
id: "1-1"
openingType: "tutorial"           # 开场类型
enabledLanes: [3]                 # 启用的行
availablePlants: []               # 可用植物
skipOpening: false                # 跳过开场(调试)

tutorialSteps:
  - trigger: "gameStart"
    text: "天空中会掉落阳光，点击收集它们！"
    action: "waitForSunCollect"
```

**参考文档:**
- 关卡详细说明: `.meta/levels/chapter1.md`
- Epic 8 完整规范: `docs/prd/epic-8-level-chapter1-implementation.md`

---

## **Epic 9: ECS 框架泛型化重构 (ECS Generics Refactor) - Brownfield Enhancement**

**史诗目标:** 将当前基于反射的 ECS 框架迁移到 Go 泛型实现，通过编译时类型检查和性能优化，消除运行时反射开销，提升代码可读性和类型安全性，为后续开发提供更高效的开发体验。

### Epic Overview (史诗概览)

Epic 9 是一次技术债重构，旨在将项目的核心 ECS（Entity-Component-System）架构从反射实现升级为 Go 泛型实现。当前系统使用 `reflect.TypeOf()` 进行组件查询，存在性能开销、类型安全问题和代码冗长问题。

**当前问题:**
- **性能开销**: 每次组件查询都需要运行时反射（预计 30-50% 性能损失）
- **类型安全**: 运行时类型断言，编译时无法检测错误
- **代码可读性**: 大量 `reflect.TypeOf()` 调用，代码冗长

**目标改进:**
```go
// ❌ 当前反射实现
entities := em.GetEntitiesWith(
    reflect.TypeOf(&components.PlantComponent{}),
    reflect.TypeOf(&components.PositionComponent{}),
)
comp, ok := em.GetComponent(entity, reflect.TypeOf(&components.PlantComponent{}))
plantComp := comp.(*components.PlantComponent) // 运行时类型断言

// ✅ 目标泛型实现
entities := ecs.GetEntitiesWith[
    *components.PlantComponent,
    *components.PositionComponent,
](em)
plantComp, ok := ecs.GetComponent[*components.PlantComponent](em, entity)
// 编译时类型安全，无需类型断言
```

### Stories (故事列表)

Epic 9 包含 3 个 Story:

#### **Story 9.1: 泛型 API 设计、原型验证与 EntityManager 核心重构**
> **As a** 开发者,
> **I want** to design and implement generic ECS APIs for EntityManager,
> **so that** I can replace reflection-based component queries with compile-time type-safe generics.

**Acceptance Criteria:**
1. 设计泛型 API 接口规范（`GetEntitiesWith[T1, T2, ...]`, `GetComponent[T]`, `AddComponent[T]`）
2. 创建原型并进行性能基准测试（对比反射 vs 泛型）
3. 确认性能提升达到预期（目标 30%+ 提升）
4. 编写迁移指南文档
5. 实现泛型版本的核心 EntityManager 方法
6. 更新 `pkg/ecs/entity_manager_test.go`，添加泛型 API 测试
7. 单元测试覆盖率 ≥ 80%

**预估工作量**: 6-9 小时

---

#### **Story 9.2: 系统批量迁移（17+ 系统文件）**
> **As a** 开发者,
> **I want** to migrate all game systems from reflection-based ECS to generic ECS,
> **so that** the entire codebase benefits from type safety and performance improvements.

**Acceptance Criteria:**
1. **第一阶段**: 迁移最复杂系统（`BehaviorSystem`, `InputSystem`）验证可行性
2. **第二阶段**: 批量迁移中等复杂度系统（8个系统文件）
   - `render_system.go`, `physics_system.go`, `reanim_system.go`, `particle_system.go`
   - `plant_selection_system.go` ⭐（解除 Story 8.1 阻塞）
   - `plant_card_system.go`, `plant_preview_system.go`
3. **第三阶段**: 迁移剩余系统（9个文件）+ 所有测试文件
4. 所有系统编译通过，消除 `reflect.TypeOf` 调用
5. 所有单元测试通过
6. 迁移日志记录完整

**预估工作量**: 8-12 小时

---

#### **Story 9.3: 全面测试、性能基准与文档更新**
> **As a** 开发者和项目维护者,
> **I want** to verify the refactor through comprehensive testing and update documentation,
> **so that** the project is stable, performant, and maintainable.

**Acceptance Criteria:**
1. **单元测试验证**: 运行 `go test ./...`，所有测试通过，覆盖率 ≥ 80%
2. **集成测试验证**: 游戏核心功能（植物种植、僵尸战斗、粒子效果、关卡加载）正常
3. **性能基准测试**:
   - 创建 `pkg/ecs/entity_manager_benchmark_test.go`
   - 测试场景：查询 1000 实体、添加/删除组件、大规模实体创建
   - 对比反射版本 vs 泛型版本，记录性能提升（预期 30-50%）
4. **文档更新**:
   - 更新 `CLAUDE.md`，添加"泛型 ECS 使用指南"章节
   - 更新代码注释，符合 GoDoc 规范
   - （可选）创建迁移指南 `docs/architecture/ecs-generics-migration-guide.md`

**预估工作量**: 2-4 小时

---

### Success Criteria (成功标准)

1. ✅ 所有 17+ 系统文件成功迁移到泛型 API
2. ✅ `reflect.TypeOf` 调用在系统层面消除（EntityManager 内部可保留）
3. ✅ 性能基准测试显示查询速度提升 30-50%
4. ✅ 所有单元测试和集成测试通过（`go test ./...`）
5. ✅ 游戏功能无回归（植物、僵尸、粒子、关卡正常）
6. ✅ 代码可读性显著提升，样板代码减少 50%
7. ✅ CLAUDE.md 更新完成，包含泛型 ECS 使用指南
8. ✅ Story 8.1 阻塞解除（`PlantSelectionSystem` 可使用泛型编译）

### Technical Implementation (技术实现)

**核心 API 设计:**
```go
// pkg/ecs/generics.go (新文件)

// GetEntitiesWith 查询拥有指定组件类型的所有实体（泛型版本）
func GetEntitiesWith[T1, T2, T3 any](em *EntityManager) []EntityID {
    // 使用类型参数进行查询，无需反射
}

// GetComponent 获取实体的特定类型组件（泛型版本）
func GetComponent[T any](em *EntityManager, entity EntityID) (T, bool) {
    // 返回具体类型，无需类型断言
}

// AddComponent 添加组件（泛型版本）
func AddComponent[T any](em *EntityManager, entity EntityID, component T) {
    // 类型安全的添加
}

// HasComponent 检查组件（泛型版本）
func HasComponent[T any](em *EntityManager, entity EntityID) bool {
    // 编译时类型检查
}
```

**迁移策略:**
1. **渐进式迁移**: 先验证最复杂系统，再批量迁移
2. **向后兼容**: 保留旧反射 API（可选），新旧 API 可共存
3. **可中断设计**: 任意阶段可暂停，已迁移系统立即受益

**风险缓解:**
- Git 分支: `feature/ecs-generics-refactor`
- 每个 Story 完成后创建 Tag（`epic9-story1-complete`）
- 回滚计划: 可回退到上一个稳定 Tag

### Dependencies and Blockers (依赖与阻塞)

**当前阻塞:**
- ❌ **Story 8.1**: `PlantSelectionSystem` 因反射冗长性暂时搁置

**解除阻塞:**
- ✅ Story 9.2 完成后，Story 8.1 可继续开发

**建议时间点:**
- **立即开始**: 在 Epic 8 后续 Story 之前完成，避免积累更多反射代码
- **或**: 作为独立 Sprint，集中 1-2 周完成

### Performance Expectations (性能预期)

| 操作 | 反射版本 | 泛型版本 | 预期提升 |
|------|---------|---------|---------|
| 查询 1000 实体（3组件） | 120 μs | 60-80 μs | 30-50% ⬆️ |
| 获取单个组件 | 50 ns | 20-30 ns | 40-60% ⬆️ |
| 添加组件 | 60 ns | 30-40 ns | 33-50% ⬆️ |

**长期收益:**
- ✅ 编译时类型检查：100% 运行时错误消除
- ✅ 代码可读性：显著提升（减少 50% 样板代码）
- ✅ 开发体验：更好的 IDE 自动补全和类型推导

### Reference Documentation (参考文档)

- **技术债文档**: `docs/technical-debt/ecs-generic-refactor.md`
- **Epic 9 完整规范**: `docs/prd/epic-9-ecs-generics-refactor.md`
- **Go 泛型官方文档**: [https://go.dev/doc/tutorial/generics](https://go.dev/doc/tutorial/generics)
- **ECS 泛型最佳实践**: [arche - A fast, minimalist Entity Component System](https://github.com/mlange-42/arche)

---

**创建日期**: 2025-10-16
**创建人**: Sarah (Product Owner)
**Epic 类型**: Brownfield Enhancement (架构优化)
**优先级**: High（阻塞 Story 8.1）
**预估总工作量**: 16-25 小时（3-5 个工作日）

---

## **Epic 10: 游戏体验完善 (Game Experience Polish) - Brownfield Enhancement**

**史诗目标:** 完善核心游戏体验，实现暂停菜单、植物攻击动画、粒子特效和除草车防线系统，提升游戏的完整性和可玩性，确保所有核心机制符合原版游戏标准。

### Epic Overview (史诗概览)

Epic 10 是对核心游戏体验的最后完善，实现后游戏基本达到 MVP 标准。本 Epic 包含 4 个关键功能模块，每个模块都是原版游戏的标志性特性。

**核心功能模块:**

1. **暂停菜单系统** - 用户体验核心功能
   - 暂停/恢复游戏
   - 菜单面板（继续、重新开始、返回主菜单）
   - 音乐控制
   - ESC 键快捷键

2. **除草车最后防线系统** - 原版标志性机制
   - 每行左侧台阶放置除草车
   - 僵尸到达左侧触发
   - 除草车自动清除所在行所有僵尸
   - 一次性使用，失败条件增强

3. **植物攻击动画系统** - 视觉表现增强
   - 植物发射子弹时切换攻击动画
   - 动画状态机管理
   - 支持多种射手植物
   - 使用原版 Reanim 动画

4. **植物种植粒子特效** - 视觉反馈完善
   - 种植时土粒飞溅效果
   - 使用原版粒子配置
   - 抛物线运动
   - 性能优化

### Stories (故事列表)

Epic 10 包含 4 个 Story，按优先级排序：

#### **Story 10.1: 暂停菜单系统**
> **As a** 玩家,
> **I want** to pause the game and access a menu with options,
> **so that** I can take a break, restart the level, or return to the main menu without losing progress.

**Acceptance Criteria:**
1. 点击右上角的"菜单"按钮时，游戏暂停，所有游戏逻辑系统停止更新
2. 暂停时显示半透明遮罩和暂停菜单面板，包含三个按钮：继续、重新开始、返回主菜单
3. 点击"继续"按钮时，关闭暂停菜单，游戏恢复运行
4. 点击"重新开始"按钮时，重新加载当前关卡
5. 点击"返回主菜单"按钮时，返回主菜单场景
6. 暂停时，背景音乐音量降低或暂停，恢复时音乐恢复正常
7. 暂停时，游戏世界的所有交互被屏蔽
8. 按 ESC 键也能切换暂停/恢复状态

**优先级**: ⭐⭐⭐⭐ 中高  
**预估工作量**: 8-12 小时

---

#### **Story 10.2: 除草车最后防线系统**
> **As a** 玩家,
> **I want** lawnmowers as a last line of defense on each lane,
> **so that** I have one final chance to stop zombies from reaching my house.

**Acceptance Criteria:**
1. 游戏开始时，每个有效行的左侧台阶上放置一辆除草车
2. 除草车正确显示在对应行的左侧，使用原版除草车图像和动画
3. 当僵尸到达屏幕左侧边界（X < 100）时，该行的除草车自动触发
4. 除草车触发后从左向右快速移动（速度约 300 像素/秒），播放行驶动画
5. 除草车移动过程中，消灭路径上所在行的所有僵尸
6. 除草车移动到屏幕右侧后消失，该行除草车标记为已使用
7. 除草车触发时播放对应的音效
8. 除草车使用后，该行再有僵尸到达左侧边界则游戏失败
9. 所有行的除草车都用完后，任意僵尸到达左侧边界直接失败
10. 除草车状态在游戏重启时正确重置

**优先级**: ⭐⭐⭐⭐⭐ 高  
**预估工作量**: 12-16 小时

---

#### **Story 10.3: 植物攻击动画系统**
> **As a** 玩家,
> **I want** plants to play attack animations when shooting projectiles,
> **so that** the game visually communicates plant actions and feels more alive.

**Acceptance Criteria:**
1. 豌豆射手发射子弹时，自动切换到攻击动画（anim_shooting）
2. 攻击动画播放完毕后，自动切换回空闲动画（anim_idle）
3. 攻击动画播放期间，植物不应再次触发攻击动画
4. 所有射手类植物都支持攻击动画
5. 攻击动画使用原版 Reanim 动画资源
6. 动画切换自然流畅，无明显跳帧或闪烁
7. 攻击动画不影响子弹发射逻辑
8. 向日葵等非射手植物不受此功能影响

**优先级**: ⭐⭐⭐⭐ 中高  
**预估工作量**: 6-8 小时

---

#### **Story 10.4: 植物种植粒子特效**
> **As a** 玩家,
> **I want** to see soil particles splash when planting plants,
> **so that** the planting action feels more impactful and visually satisfying.

**Acceptance Criteria:**
1. 玩家成功种植植物时，在种植位置生成土粒飞溅粒子效果
2. 粒子效果使用原版配置文件（PlantingPool.xml 或类似文件）
3. 土粒粒子从地面向上飞溅，然后落下，形成自然的抛物线运动
4. 粒子效果持续 0.5-1.0 秒后自动消失
5. 粒子颜色和大小符合原版表现
6. 所有植物种植时都触发粒子效果
7. 粒子效果在正确的渲染层级显示
8. 粒子系统性能良好，同时种植多个植物不影响帧率

**优先级**: ⭐⭐⭐ 中  
**预估工作量**: 4-6 小时

---

### Success Criteria (成功标准)

1. ✅ 所有 4 个用户故事完成，AC 全部满足
2. ✅ 暂停菜单功能完整，UI 美观，交互流畅
3. ✅ 植物攻击动画自然切换，无卡顿或闪烁
4. ✅ 种植粒子特效符合原版表现，性能良好
5. ✅ 除草车系统完整，触发准确，动画流畅
6. ✅ 现有功能无回归，通过 QA 测试
7. ✅ 代码符合编码规范，通过 linter 检查
8. ✅ 所有新功能有对应的单元测试（覆盖率 > 80%）
9. ✅ 文档更新（AGENTS.md, CLAUDE.md）
10. ✅ 游戏体验达到 MVP 标准

### Technical Implementation (技术实现)

**新增组件:**
- `PauseMenuComponent` - 暂停菜单状态
- `LawnmowerComponent` - 除草车数据
- `LawnmowerStateComponent` - 全局除草车状态
- `AttackAnimState` - 攻击动画状态（在 PlantComponent 中）

**新增系统:**
- `PauseMenuRenderSystem` - 暂停菜单渲染
- `LawnmowerSystem` - 除草车触发和移动管理

**系统修改:**
- `GameScene.Update()` - 暂停状态检测
- `InputSystem.handleLawnClick()` - 种植粒子效果触发
- `BehaviorSystem` - 植物攻击动画管理
- `LevelSystem.checkDefeatCondition()` - 除草车状态检测

**工厂函数:**
- `NewPauseMenuEntity()` - 暂停菜单实体
- `NewLawnmowerEntity()` - 除草车实体
- `NewPlantingParticleEffect()` - 种植粒子效果

### Architecture Highlights (架构亮点)

1. **ECS 架构一致性** - 所有新功能遵循 ECS 模式，零耦合原则
2. **泛型 API 使用** - 使用 Epic 9 的泛型 ECS API，类型安全
3. **原版忠实还原** - 所有功能使用原版资源和配置
4. **性能优化** - 粒子批量渲染，除草车碰撞优化
5. **可扩展性** - 模块化设计，易于添加新功能

### Dependencies and Blockers (依赖与阻塞)

**前置依赖:**
- ✅ Epic 1-9 已完成（ECS 框架、Reanim 系统、粒子系统）
- ✅ 菜单按钮 UI 已创建（Story 8.5）

**并行依赖:**
- Story 10.1 和 10.2 可并行开发（独立模块）
- Story 10.3 和 10.4 可并行开发（独立模块）

**后续依赖:**
- Epic 11（如有）可能依赖暂停菜单系统
- 第一章关卡完整体验依赖除草车系统

### Timeline Estimate (时间估算)

| Story | 预估工作量 | 优先级 |
|-------|-----------|--------|
| Story 10.1: 暂停菜单系统 | 8-12 小时 | 中高 |
| Story 10.2: 除草车系统 | 12-16 小时 | 高 |
| Story 10.3: 植物攻击动画 | 6-8 小时 | 中高 |
| Story 10.4: 种植粒子特效 | 4-6 小时 | 中 |
| **总计** | **30-42 小时** | - |

**建议实施顺序:**
1. Story 10.2（除草车）- 最高优先级，影响游戏完整性
2. Story 10.1（暂停菜单）- 用户体验核心功能
3. Story 10.3（攻击动画）- 视觉表现增强
4. Story 10.4（种植特效）- 视觉表现锦上添花

### Reference Documentation (参考文档)

- **Epic 10 完整规范**: `docs/prd/epic-10-game-experience-polish.md`
- **Story 10.1 详细文档**: `docs/stories/10.1.story.md`
- **Story 10.2 详细文档**: `docs/stories/10.2.story.md`
- **Story 10.3 详细文档**: `docs/stories/10.3.story.md`
- **Story 10.4 详细文档**: `docs/stories/10.4.story.md`

---

**创建日期**: 2025-10-21
**创建人**: Sarah (Product Owner)
**Epic 类型**: Brownfield Enhancement (体验完善)
**优先级**: High（完成 MVP 必需）
**预估总工作量**: 30-42 小时（6-8 个工作日）
