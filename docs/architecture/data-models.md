# **3. Data Models (数据模型) / Components**

在我们的实体-组件-系统（ECS）架构中，数据模型由各个"组件"的结构体（structs）来定义。实体（如植物、僵尸）是这些组件的集合。

> **最后更新**: 2025-12-09
>
> **重要提示**: 项目使用 **Go 泛型 ECS API**，详见 `pkg/ecs/entity_manager.go`

---

## 组件分类概览

| 分类 | 数量 | 说明 |
|------|------|------|
| 核心组件 | 10+ | 位置、速度、生命值等基础组件 |
| 动画组件 | 3 | Reanim 骨骼动画系统组件 |
| 游戏逻辑组件 | 15+ | 植物、僵尸、波次等游戏逻辑组件 |
| 粒子/特效组件 | 4 | 粒子系统和视觉效果组件 |
| UI 组件 | 15+ | 按钮、对话框、选卡界面等 |
| 特殊组件 | 15+ | 教程、奖励、开场动画等 |

---

## 一、核心组件

### **`PositionComponent`**
*   **Purpose:** 存储实体在游戏世界中的二维坐标。所有需要被渲染或参与物理计算的实体都将拥有此组件。
*   **文件:** `pkg/components/position.go`
*   **Go Struct:**
    ```go
    type PositionComponent struct {
        X, Y float64
    }
    ```

---

### **`VelocityComponent`**
*   **Purpose:** 存储实体的速度向量，用于物理系统更新位置。
*   **文件:** `pkg/components/velocity.go`
*   **Go Struct:**
    ```go
    type VelocityComponent struct {
        X, Y float64
    }
    ```

---

### **`ScaleComponent`**
*   **Purpose:** 存储实体的缩放比例，用于渲染时的大小调整。
*   **文件:** `pkg/components/scale.go`
*   **Go Struct:**
    ```go
    type ScaleComponent struct {
        X, Y float64
    }
    ```

---

### **`HealthComponent`**
*   **Purpose:** 存储实体的生命值信息，包括当前生命值、最大生命值和死亡效果类型。
*   **文件:** `pkg/components/health.go`
*   **Go Struct:**
    ```go
    // DeathEffectType 死亡效果类型
    type DeathEffectType int

    const (
        DeathEffectNormal    DeathEffectType = iota // 普通死亡：头部掉落、手臂掉落
        DeathEffectExplosion                        // 爆炸死亡：烧焦动画
        DeathEffectInstant                          // 瞬间死亡：无粒子效果
    )

    type HealthComponent struct {
        CurrentHealth   int             // 当前生命值
        MaxHealth       int             // 最大生命值
        ArmLost         bool            // 僵尸手臂是否已掉落
        DeathEffectType DeathEffectType // 死亡效果类型
    }
    ```

---

### **`ArmorComponent`**
*   **Purpose:** 存储僵尸的护甲信息（路障、铁桶等）。
*   **文件:** `pkg/components/armor.go`
*   **Go Struct:**
    ```go
    type ArmorComponent struct {
        ArmorHealth    int    // 护甲当前生命值
        MaxArmorHealth int    // 护甲最大生命值
        ArmorType      string // 护甲类型 ("cone", "bucket")
    }
    ```

---

### **`CollisionComponent`**
*   **Purpose:** 定义实体的碰撞区域，用于碰撞检测。
*   **文件:** `pkg/components/collision.go`
*   **Go Struct:**
    ```go
    type CollisionComponent struct {
        Width  float64
        Height float64
        OffsetX, OffsetY float64 // 相对于 Position 的偏移
    }
    ```

---

### **`ClickableComponent`**
*   **Purpose:** 标记实体为可点击，存储点击区域。
*   **文件:** `pkg/components/clickable.go`
*   **Go Struct:**
    ```go
    type ClickableComponent struct {
        Width  float64
        Height float64
        OnClick func() // 点击回调
    }
    ```

---

### **`LifetimeComponent`**
*   **Purpose:** 存储实体的生命周期，到期后自动销毁。
*   **文件:** `pkg/components/lifetime.go`
*   **Go Struct:**
    ```go
    type LifetimeComponent struct {
        Remaining float64 // 剩余时间（秒）
    }
    ```

---

### **`TimerComponent`**
*   **Purpose:** 通用计时器组件，用于攻击冷却、阳光生产等周期性行为。
*   **文件:** `pkg/components/timer.go`
*   **Go Struct:**
    ```go
    type TimerComponent struct {
        Name        string  // 计时器名称
        TargetTime  float64 // 目标时间（秒）
        CurrentTime float64
        IsReady     bool    // 是否已完成
    }
    ```

---

### **`BehaviorComponent`**
*   **Purpose:** 定义实体的行为类型，逻辑系统根据此组件决定如何处理实体。
*   **文件:** `pkg/components/behavior.go`
*   **Go Struct:**
    ```go
    type BehaviorType int

    const (
        // 植物行为
        BehaviorSunflower    BehaviorType = iota // 向日葵：生产阳光
        BehaviorPeashooter                       // 豌豆射手：攻击僵尸
        BehaviorWallnut                          // 坚果墙：防御植物
        BehaviorCherryBomb                       // 樱桃炸弹：范围爆炸
        BehaviorPotatoMine                       // 土豆雷：触发爆炸

        // 投射物行为
        BehaviorPeaProjectile // 豌豆子弹
        BehaviorPeaBulletHit  // 豌豆击中效果

        // 僵尸行为
        BehaviorZombieBasic          // 普通僵尸
        BehaviorZombieConehead       // 路障僵尸
        BehaviorZombieBuckethead     // 铁桶僵尸
        BehaviorZombieFlag           // 旗帜僵尸
        BehaviorZombieEating         // 僵尸啃食状态
        BehaviorZombieDying          // 僵尸死亡（普通）
        BehaviorZombieDyingExplosion // 僵尸死亡（爆炸烧焦）
        BehaviorZombieSquashing      // 僵尸被压扁
        BehaviorZombiePreview        // 僵尸预览（开场动画）

        // 效果行为
        BehaviorFallingPart // 掉落部件（手臂、头部）
    )

    // ZombieAnimState 僵尸动画状态
    type ZombieAnimState int

    const (
        ZombieAnimIdle    ZombieAnimState = iota // 待机状态
        ZombieAnimWalking                        // 行走状态
        ZombieAnimEating                         // 啃食状态
        ZombieAnimDying                          // 死亡状态
    )

    type BehaviorComponent struct {
        Type            BehaviorType    // 行为类型
        ZombieAnimState ZombieAnimState // 僵尸动画状态
        UnitID          string          // 动画配置 ID
        LastEatAnimFrame int            // 上次啃食动画帧
    }
    ```

---

## 二、动画组件

### **`ReanimComponent`** ⭐ 核心组件
*   **Purpose:** Reanim 骨骼动画组件，替代了原有的 `AnimationComponent`。存储动画状态、帧数据、轨道信息等。
*   **文件:** `pkg/components/reanim_component.go`
*   **说明:** 这是项目最重要的动画组件，支持原版 PvZ 的骨骼动画系统。
*   **详细文档:** `docs/reanim/reanim-format-guide.md`
*   **Go Struct:**
    ```go
    type ReanimComponent struct {
        // 动画定义引用
        Definition *reanim.ReanimXML

        // 当前动画状态
        CurrentAnim     string  // 当前播放的动画名称
        CurrentFrame    float64 // 当前帧（支持小数）
        FrameRate       float64 // 播放速率（FPS）
        IsPlaying       bool    // 是否正在播放
        Loop            bool    // 是否循环播放

        // 轨道数据
        Tracks          []*TrackState // 轨道状态数组
        PartImages      map[string]*ebiten.Image // 部件图片映射

        // 动画组合
        ComposedAnims   []*ComposedAnimation // 组合动画列表

        // 帧缓存（处理空帧继承）
        FrameCache      map[string]*CachedFrame
    }
    ```

---

### **`AnimationCommandComponent`**
*   **Purpose:** 动画命令队列，用于管理动画切换和回调。
*   **文件:** `pkg/components/animation_command.go`
*   **Go Struct:**
    ```go
    type AnimationCommand struct {
        AnimName   string       // 目标动画名称
        Loop       bool         // 是否循环
        OnComplete func()       // 完成回调
        Priority   int          // 优先级
    }

    type AnimationCommandComponent struct {
        Queue []AnimationCommand // 命令队列
    }
    ```

---

### **`SquashAnimationComponent`**
*   **Purpose:** 压扁动画组件，用于除草车碾压僵尸的特殊动画效果。
*   **文件:** `pkg/components/squash_animation_component.go`
*   **Go Struct:**
    ```go
    type SquashAnimationComponent struct {
        StartTime     float64 // 动画开始时间
        Duration      float64 // 动画持续时间
        InitialScaleY float64 // 初始 Y 缩放
        TargetScaleY  float64 // 目标 Y 缩放（压扁后）
        InitialY      float64 // 初始 Y 位置
        TargetY       float64 // 目标 Y 位置
    }
    ```

---

### **`SpriteComponent`** ⚠️ 已废弃
*   **Status:** 已被 `ReanimComponent` 替代，仅用于简单的静态图片显示。
*   **文件:** `pkg/components/sprite.go`

---

### **`AnimationComponent`** ⚠️ 已废弃
*   **Status:** 已被 `ReanimComponent` 替代。原用于简单帧动画，现已不再使用。

---

## 三、游戏逻辑组件

### **`PlantComponent`**
*   **Purpose:** 存储植物的游戏逻辑数据。
*   **文件:** `pkg/components/plant.go`
*   **Go Struct:**
    ```go
    type PlantComponent struct {
        PlantType     string  // 植物类型 ("peashooter", "sunflower" 等)
        GridRow       int     // 所在行
        GridCol       int     // 所在列
        AttackTimer   float64 // 攻击计时器
        ProductTimer  float64 // 生产计时器（向日葵用）
        IsArmed       bool    // 是否已武装（土豆雷用）
    }
    ```

---

### **`PlantCardComponent`**
*   **Purpose:** 植物选择栏中的卡片组件。
*   **文件:** `pkg/components/plant_card.go`
*   **Go Struct:**
    ```go
    type PlantCardComponent struct {
        PlantType   string  // 植物类型
        SunCost     int     // 阳光消耗
        CooldownMax float64 // 最大冷却时间
        Cooldown    float64 // 当前冷却时间
        IsSelected  bool    // 是否被选中
        IsAvailable bool    // 是否可用
    }
    ```

---

### **`SunComponent`**
*   **Purpose:** 阳光实体组件。
*   **文件:** `pkg/components/sun.go`
*   **Go Struct:**
    ```go
    type SunComponent struct {
        Value       int     // 阳光值（通常为 25）
        IsCollected bool    // 是否已被收集
        FallTarget  float64 // 掉落目标 Y 坐标
        IsFalling   bool    // 是否正在下落
    }
    ```

---

### **`WaveTimerComponent`**
*   **Purpose:** 波次计时器，控制僵尸波次的生成时机。
*   **文件:** `pkg/components/wave_timer.go`
*   **Go Struct:**
    ```go
    type WaveTimerComponent struct {
        CurrentWave     int     // 当前波次
        TotalWaves      int     // 总波次数
        WaveTimer       float64 // 波次计时器
        WaveInterval    float64 // 波次间隔
        IsFinalWave     bool    // 是否为最终波
    }
    ```

---

### **`ZombieTargetLaneComponent`**
*   **Purpose:** 僵尸目标行组件，指定僵尸应该进入的行。
*   **文件:** `pkg/components/zombie_target_lane.go`

---

### **`ZombieWaveStateComponent`**
*   **Purpose:** 僵尸波次状态，追踪僵尸属于哪个波次。
*   **文件:** `pkg/components/zombie_wave_state.go`

---

### **`LawnmowerComponent`**
*   **Purpose:** 除草车组件，存储除草车状态。
*   **文件:** `pkg/components/lawnmower_component.go`
*   **Go Struct:**
    ```go
    type LawnmowerComponent struct {
        Lane        int     // 所在行
        IsTriggered bool    // 是否已触发
        Speed       float64 // 移动速度
    }
    ```

---

### **`ConveyorBeltComponent`**
*   **Purpose:** 传送带组件，用于传送带关卡（如 1-10）。
*   **文件:** `pkg/components/conveyor_belt_component.go`

---

### **`BowlingNutComponent`**
*   **Purpose:** 保龄球坚果组件，用于 1-5 关卡。
*   **文件:** `pkg/components/bowling_nut_component.go`
*   **Go Struct:**
    ```go
    type BowlingNutComponent struct {
        IsRolling   bool    // 是否正在滚动
        Speed       float64 // 滚动速度
        Lane        int     // 当前行
        IsExplosive bool    // 是否为爆炸坚果
        BounceCount int     // 反弹次数
    }
    ```

---

### **`LevelPhaseComponent`**
*   **Purpose:** 关卡阶段组件，追踪关卡当前阶段（准备、战斗、胜利、失败）。
*   **文件:** `pkg/components/level_phase_component.go`

---

## 四、粒子/特效组件

### **`ParticleComponent`**
*   **Purpose:** 单个粒子的状态数据。
*   **文件:** `pkg/components/particle_component.go`
*   **Go Struct:**
    ```go
    type ParticleComponent struct {
        X, Y          float64 // 位置
        VelX, VelY    float64 // 速度
        Rotation      float64 // 旋转角度
        Scale         float64 // 缩放
        Alpha         float64 // 透明度
        Lifetime      float64 // 剩余生命
        MaxLifetime   float64 // 最大生命
        Color         color.RGBA // 颜色
    }
    ```

---

### **`EmitterComponent`**
*   **Purpose:** 粒子发射器组件，控制粒子的生成。
*   **文件:** `pkg/components/emitter_component.go`
*   **Go Struct:**
    ```go
    type EmitterComponent struct {
        Config      *particle.EmitterConfig // 发射器配置
        SpawnTimer  float64                 // 生成计时器
        ActiveCount int                     // 活跃粒子数
        IsActive    bool                    // 是否活跃
    }
    ```

---

### **`FlashEffectComponent`**
*   **Purpose:** 闪烁效果组件，用于阳光不足提示等。
*   **文件:** `pkg/components/flash_effect_component.go`

---

### **`ShadowComponent`**
*   **Purpose:** 阴影组件，为植物和僵尸渲染脚下阴影。
*   **文件:** `pkg/components/shadow_component.go`
*   **Go Struct:**
    ```go
    type ShadowComponent struct {
        Width   float64 // 阴影宽度
        Height  float64 // 阴影高度
        OffsetY float64 // Y 偏移
        Alpha   float64 // 透明度
    }
    ```

---

## 五、UI 组件

### **`UIComponent`**
*   **Purpose:** 基础 UI 组件，标识实体为 UI 元素。
*   **文件:** `pkg/components/ui_component.go`
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
        State   UIState
        Layer   int     // 渲染层级
        Visible bool    // 是否可见
    }
    ```

---

### **`ButtonComponent`**
*   **Purpose:** 按钮组件���存储按钮状态和回调。
*   **文件:** `pkg/components/button_component.go`

---

### **`DialogComponent`**
*   **Purpose:** 对话框组件，用于弹出对话框显示。
*   **文件:** `pkg/components/dialog_component.go`

---

### **`SliderComponent`**
*   **Purpose:** 滑块组件，用于音量控制等。
*   **文件:** `pkg/components/slider_component.go`

---

### **`CheckboxComponent`**
*   **Purpose:** 复选框组件。
*   **文件:** `pkg/components/checkbox_component.go`

---

### **`TextInputComponent`**
*   **Purpose:** 文本输入组件。
*   **文件:** `pkg/components/text_input_component.go`

---

### **`VirtualKeyboardComponent`**
*   **Purpose:** 虚拟键盘组件，用于移动平台文本输入。
*   **文件:** `pkg/components/virtual_keyboard_component.go`

---

### **`TooltipComponent`**
*   **Purpose:** 提示框组件，用于显示植物名称和状态信息。
*   **文件:** `pkg/components/tooltip_component.go`
*   **Go Struct:**
    ```go
    type TooltipComponent struct {
        IsVisible       bool
        StatusText      string      // 状态提示
        StatusTextColor color.Color
        PlantName       string      // 植物名称
        PlantNameColor  color.Color
        BackgroundColor color.Color
        BorderColor     color.Color
        Padding         float64
        TextSpacing     float64
        Position        struct{ X, Y float64 }
        Width, Height   float64
        TargetEntity    ecs.EntityID
    }
    ```

---

### **`PlantSelectionComponent`**
*   **Purpose:** 选卡界面组件。
*   **文件:** `pkg/components/plant_selection_component.go`

---

### **`PlantPreviewComponent`**
*   **Purpose:** 植物预览组件，在种植模式下显示植物预览。
*   **文件:** `pkg/components/plant_preview.go`

---

### **`PauseMenuComponent`**
*   **Purpose:** 暂停菜单组件。
*   **文件:** `pkg/components/pause_menu_component.go`

---

### **`LevelProgressBarComponent`**
*   **Purpose:** 关卡进度条组件。
*   **文件:** `pkg/components/level_progress_bar_component.go`

---

## 六、特殊组件

### **`CameraComponent`**
*   **Purpose:** 摄像机组件，用于开场动画的镜头控制。
*   **文件:** `pkg/components/camera_component.go`

---

### **`DifficultyComponent`**
*   **Purpose:** 难度组件，存储当前关卡难度参数。
*   **文件:** `pkg/components/difficulty_component.go`

---

### **`SpawnConstraintComponent`**
*   **Purpose:** 生成约束组件，控制僵尸生成规则。
*   **文件:** `pkg/components/spawn_constraint.go`

---

### **`OpeningAnimationComponent`**
*   **Purpose:** 开场动画组件，控制关卡开始时的镜头移动和僵尸预告。
*   **文件:** `pkg/components/opening_animation_component.go`

---

### **`DaveDialogueComponent`**
*   **Purpose:** 疯狂戴夫对话组件。
*   **文件:** `pkg/components/dave_dialogue_component.go`

---

### **`GuidedTutorialComponent`**
*   **Purpose:** 引导教程组件，用于 1-1 等教学关卡。
*   **文件:** `pkg/components/guided_tutorial_component.go`

---

### **`RewardAnimationComponent`**
*   **Purpose:** 奖励动画组件，用于关卡完成时的奖励动画。
*   **文件:** `pkg/components/reward_animation_component.go`

---

### **`RewardCardComponent`**
*   **Purpose:** 奖励卡片组件。
*   **文件:** `pkg/components/reward_card_component.go`

---

### **`RewardPanelComponent`**
*   **Purpose:** 奖励面板组件。
*   **文件:** `pkg/components/reward_panel_component.go`

---

### **`FinalWaveWarningComponent`**
*   **Purpose:** 最终波警告组件。
*   **文件:** `pkg/components/final_wave_warning_component.go`

---

### **`FlagWaveWarningComponent`**
*   **Purpose:** 旗帜波警告组件。
*   **文件:** `pkg/components/flag_wave_warning_component.go`

---

### **`ZombiesWonPhaseComponent`**
*   **Purpose:** 游戏失败阶段组件，控制失败流程。
*   **文件:** `pkg/components/zombies_won_phase_component.go`

---

## 七、组件使用示例

### 使用泛型 API 获取组件

```go
// 获取单个组件
plantComp, ok := ecs.GetComponent[*components.PlantComponent](em, entity)
if ok {
    plantComp.AttackTimer -= dt
}

// 查询拥有多个组件的实体
entities := ecs.GetEntitiesWith3[
    *components.BehaviorComponent,
    *components.PositionComponent,
    *components.HealthComponent,
](em)

for _, entity := range entities {
    behavior, _ := ecs.GetComponent[*components.BehaviorComponent](em, entity)
    pos, _ := ecs.GetComponent[*components.PositionComponent](em, entity)
    health, _ := ecs.GetComponent[*components.HealthComponent](em, entity)

    // 处理实体...
}
```

### 添加组件到实体

```go
entity := em.NewEntity()

ecs.AddComponent(em, entity, &components.PositionComponent{
    X: 100,
    Y: 200,
})

ecs.AddComponent(em, entity, &components.HealthComponent{
    CurrentHealth: 300,
    MaxHealth:     300,
})

ecs.AddComponent(em, entity, &components.BehaviorComponent{
    Type: components.BehaviorPeashooter,
})
```

---

## 参考文档

- **ECS 框架:** `pkg/ecs/entity_manager.go`
- **泛型 API 指南:** `docs/architecture/ecs-generics-migration-guide.md`
- **Reanim 动画系统:** `docs/reanim/reanim-format-guide.md`
- **组件源代码:** `pkg/components/`
