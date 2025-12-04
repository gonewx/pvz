# **Epic 2: 核心资源与玩家交互 (Core Resources & Player Interaction)**
**史诗目标:** 实现阳光的生成、收集和计数系统，并建立玩家与游戏世界的基础交互，如通过鼠标点击进行操作。

---
**Story 2.1: 游戏场景UI框架**
> **As a** 开发者,
> **I want** to load and display the basic in-game UI elements for the daytime lawn scene,
> **so that** I have the foundational layout for placing game state information.

**Acceptance Criteria:**
1.  `GameScene` 啟動時，會繪製正確的草坪背景。
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
