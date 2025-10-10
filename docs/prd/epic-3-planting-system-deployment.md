# **Epic 3: 植物系统与部署 (Planting System & Deployment)**
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
