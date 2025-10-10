# **3. Data Models (数据模型) / Components**

在我们的实体-组件-系统（ECS）架构中，数据模型由各个“组件”的结构体（structs）来定义。实体（如植物、僵尸）是这些组件的集合。以下是实现MVP所需的核心组件定义。

---
## **`PositionComponent`**
*   **Purpose:** 存储一个实体在游戏世界中的二维坐标。所有需要被渲染或参与物理计算的实体都将拥有此组件。
*   **Go Struct:**
    ```go
    type PositionComponent struct {
        X, Y float64
    }
    ```

---
## **`SpriteComponent`**
*   **Purpose:** 存储实体的视觉表现信息，主要是一个指向当前需要绘制的图像的引用。
*   **Go Struct:**
    ```go
    import "github.com/hajimehoshi/ebiten/v2"

    type SpriteComponent struct {
        Image *ebiten.Image
    }
    ```

---
## **`AnimationComponent`**
*   **Purpose:** 管理基于spritesheet的动画。它存储了动画的所有帧、播放速度以及当前状态。
*   **Go Struct:**
    ```go
    import "github.com/hajimehoshi/ebiten/v2"

    type AnimationComponent struct {
        Frames []*ebiten.Image // 动画的所有帧
        FrameSpeed float64      // 帧之间的延迟秒数
        FrameCounter float64
        CurrentFrame int
    }
    ```

---
## **`HealthComponent`**
*   **Purpose:** 存储实体的生命值信息，包括当前生命值和最大生命值。适用于所有可被伤害的单位，如植物和僵尸。
*   **Go Struct:**
    ```go
    type HealthComponent struct {
        CurrentHealth int
        MaxHealth     int
    }
    ```
---
## **`BehaviorComponent`**
*   **Purpose:** 定义实体的行为类型，例如“向日葵”、“豌豆射手”、“普通僵尸”。逻辑系统（Systems）会根据这个组件来决定如何处理一个实体。
*   **Go Struct:**
    ```go
    type BehaviorType int
    const (
        BehaviorSunflower BehaviorType = iota
        BehaviorPeashooter
        BehaviorWallnut
        BehaviorCherryBomb
        BehaviorZombieBasic
        BehaviorZombieConehead
        BehaviorZombieBuckethead
    )

    type BehaviorComponent struct {
        Type BehaviorType
    }
    ```
---
## **`TimerComponent`**
*   **Purpose:** 一个通用的计时器组件，用于处理需要时间延迟的行为，如植物的攻击冷却、向日葵的阳光生产周期等。
*   **Go Struct:**
    ```go
    type TimerComponent struct {
        Name string // 计时器名称，如 "attack_cooldown"
        TargetTime float64 // 目标时间（秒）
        CurrentTime float64
        IsReady bool   // 计时器是否已完成
    }
    ```
---
## **`UIComponent`**
*   **Purpose:** 这是一个标记组件，用于标识一个实体是UI元素。它也可以包含UI相关的状态数据。
*   **Go Struct:**
    ```go
    type UIState int
    const (
        UINormal UIState = iota
        UIHovered
        UIClicked
        UIDisabled
    )
    type UIComponent struct {
        State UIState
        // ... other UI related data
    }
    ```
