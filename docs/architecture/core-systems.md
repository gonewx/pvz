# **4. Core Systems (核心系统)**

游戏的行为逻辑由一系列专门的“系统”驱动。每个系统在一个游戏循环（Update tick）中运行，查询包含特定组件组合的实体，并对这些组件的数据进行操作。

---
## **`SceneManager` (场景管理器)**
*   **Responsibility:** 管理游戏的宏观状态，如主菜单、游戏场景、暂停菜单等。它负责切换当前活动的场景，并确保只有一个场景的`Update`和`Draw`方法在被调用。
*   **Key Interfaces:** `Update(deltaTime float64)`, `Draw(screen *ebiten.Image)`, `SwitchTo(sceneName string)`。
*   **Dependencies:** 无。它是最高层的控制器。

---
## **`EntityManager` (实体管理器)**
*   **Responsibility:** 负责所有实体（Entities）和组件（Components）的创建、销毁和存储。它是ECS模式的核心数据库。提供查询功能，例如“给我所有同时拥有`PositionComponent`和`SpriteComponent`的实体”。
*   **Key Interfaces:** `NewEntity()`, `AddComponent(entityID, component)`, `GetComponent(entityID, componentType)`, `QueryByComponents(componentTypes ...)`。
*   **Dependencies:** 无。它是游戏世界状态的核心。

---
## **`InputSystem` (输入系统)**
*   **Responsibility:** 捕获并处理所有原始玩家输入（鼠标点击、按键）。它将原始输入转换为游戏内的具体“意图”或“事件”，例如 `SunClickedEvent`, `PlantCardClickedEvent`, `GridCellClickedEvent`。
*   **Key Interfaces:** `Update(deltaTime float64)`。
*   **Dependencies:** `EntityManager` (查询可点击的对象), Event Bus。

---
## **`BehaviorSystem` (行为系统)**
*   **Responsibility:** 这是游戏逻辑的核心。它根据实体的`BehaviorComponent`来执行具体的行为。例如：
    *   **向日葵:** 管理其`TimerComponent`，在计时器结束后创建阳光实体。
    *   **豌豆射手:** 扫描同一行的僵尸，管理攻击`TimerComponent`，在计时器结束后创建子弹实体。
    *   **僵尸:** 控制其移动，检测并啃食植物。
*   **Key Interfaces:** `Update(deltaTime float64)`。
*   **Dependencies:** `EntityManager` (查询并更新实体和组件)。

---
## **`PhysicsSystem` (物理系统)**
*   **Responsibility:** 处理移动和碰撞检测。
    *   **移动:** 根据实体的`VelocityComponent`（如果需要的话）更新其`PositionComponent`。
    *   **碰撞:** 检查所有子弹和所有僵尸的`PositionComponent`和`CollisionComponent`，当发生重叠时，发布`CollisionEvent`事件。
*   **Key Interfaces:** `Update(deltaTime float64)`。
*   **Dependencies:** `EntityManager`。

---
## **`AnimationSystem` (动画系统)**
*   **Responsibility:** 更新所有拥有`AnimationComponent`的实体的动画帧。它会根据`FrameSpeed`和流逝的时间来决定是否切换到下一帧。
*   **Key Interfaces:** `Update(deltaTime float64)`。
*   **Dependencies:** `EntityManager` (查询所有带动画的实体)。

---
## **`RenderSystem` (渲染系统)**
*   **Responsibility:** 迭代所有拥有`PositionComponent`和`SpriteComponent`的实体，并使用Ebitengine的绘图函数将它们绘制到屏幕上的正确位置。这是唯一一个与绘图API直接交互的系统。
*   **Key Interfaces:** `Draw(screen *ebiten.Image)`。
*   **Dependencies:** `EntityManager`。

---
## **`UISystem` (UI系统)**
*   **Responsibility:** 专门处理UI元素的逻辑。查询所有拥有`UIComponent`的实体，根据`GameState`（如阳光数量、冷却时间）更新其状态和`SpriteComponent`。处理UI元素的动画（如按钮点击效果）。
*   **Key Interfaces:** `Update(deltaTime float64)`。
*   **Dependencies:** `EntityManager`, `InputSystem` (接收UI点击事件), `GameState`。
