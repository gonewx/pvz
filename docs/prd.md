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
3.  当玩家阳光数量不足时，对应的植物卡片会变暗或显示为不可用状态。
4.  当一个植物卡片处于冷却状态时，卡片会从下往上逐渐被灰色覆盖，模拟冷却计时器。
5.  当冷却结束后，卡片恢复正常可用状态。

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
