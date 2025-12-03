# **Epic 4: 基础僵尸与战斗逻辑 (Basic Zombies & Combat Logic)**
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
2.  当其正前方（右侧）的同一行出现了僵尸时，豌豆射手开始发射子弹。
3.  豌豆射手会按照固定的时间间隔（1.4秒），从其口部发射豌豆子弹。
4.  豌豆射手持续循环播放其动画（无论是否在攻击）。
5.  如果没有僵尸在其行上，豌豆射手停止发射子弹，但动画继续播放。

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
