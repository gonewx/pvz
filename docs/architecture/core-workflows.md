# **5. Core Workflows (核心工作流)**

以下序列图展示了几个关键游戏场景中，不同系统之间的交互流程。

---
## **工作流 1: 玩家收集阳光 (Player Collects a Sun)**
此图展示了从玩家点击阳光到UI更新的完整流程。

```mermaid
sequenceDiagram
    participant Player
    participant InputSystem
    participant GameState
    participant UIRenderSystem

    Player->>InputSystem: Click on Sun Entity
    InputSystem->>GameState: AddSun(25)
    GameState-->>InputSystem: Sun count updated
    InputSystem->>EntityManager: Mark Sun Entity for Deletion

    loop Game Update Loop
        UIRenderSystem->>GameState: GetCurrentSun()
        GameState-->>UIRenderSystem: Returns new sun count
        UIRenderSystem->>Screen: Draw updated sun count UI
    end
```

---
## **工作流 2: 豌豆射手攻击僵尸 (Peashooter Shoots a Zombie)**
此图展示了豌豆射手自动索敌、发射子弹，以及子弹命中僵尸的全过程。

```mermaid
sequenceDiagram
    participant BehaviorSystem
    participant EntityManager
    participant PhysicsSystem
    participant EventBus

    loop Game Update Loop
        BehaviorSystem->>EntityManager: Query Peashooters and Zombies
        EntityManager-->>BehaviorSystem: Return entities in same lane
        
        alt Zombie in range
            BehaviorSystem->>EntityManager: Create Pea Bullet Entity
        end

        PhysicsSystem->>EntityManager: Query Bullets and Zombies
        EntityManager-->>PhysicsSystem: Return entities with Position
        PhysicsSystem->>PhysicsSystem: Check for collision between Bullet and Zombie
        
        alt Collision detected
            PhysicsSystem->>EventBus: Publish CollisionEvent(Bullet, Zombie)
            PhysicsSystem->>EntityManager: Mark Bullet Entity for Deletion
        end
    end
```
---
## **工作流 3: 伤害计算与僵尸死亡 (Damage Calculation & Zombie Death)**
此图紧接上一个流程，展示了碰撞事件被处理，最终导致僵尸死亡的流程。

```mermaid
sequenceDiagram
    participant EventBus
    participant DamageSystem
    participant EntityManager
    participant BehaviorSystem

    EventBus->>DamageSystem: Notify CollisionEvent(Bullet, Zombie)
    DamageSystem->>EntityManager: Get HealthComponent for Zombie
    EntityManager-->>DamageSystem: Return Zombie's HealthComponent
    DamageSystem->>DamageSystem: Subtract bullet damage from health
    DamageSystem->>EntityManager: Update Zombie's HealthComponent

    loop Game Update Loop
        BehaviorSystem->>EntityManager: Query Zombies with Health <= 0
        EntityManager-->>BehaviorSystem: Return dying zombies
        
        alt Zombie has dying animation
             BehaviorSystem->>EntityManager: Change Zombie's Animation to 'Dying'
        else Zombie has no dying animation
             BehaviorSystem->>EntityManager: Mark Zombie Entity for Deletion
        end
    end

```
