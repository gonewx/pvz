# **Epic 5: 游戏流程与高级单位 (Game Flow & Advanced Units)**
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
4.  在最后一波僵尸到来之前，屏幕上会出现“A huge wave of zombies is approaching”的提示。
5.  当玩家消灭所有波次的僵尸后，游戏会暂停，并显示胜利界面。
6.  如果任何一个僵尸走到了屏幕最左端，游戏会立即暂停，并显示失败界面。
