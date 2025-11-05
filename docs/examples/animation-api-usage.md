# Reanim 动画 API 使用示例

本文档提供了完整的、可运行的 Reanim 动画 API 使用示例。

## 基础用法

### 单动画播放

最常见的场景，播放一个动画：

```go
// 向日葵空闲动画
func createSunflower(em *ecs.EntityManager, reanimSys *systems.ReanimSystem) ecs.Entity {
    entity := em.NewEntity()

    // ... 创建实体和组件 ...

    // 播放空闲动画
    reanimSys.PlayAnimation(entity, "anim_idle")

    return entity
}
```

### 多动画叠加

当一个实体需要同时控制多个部件时：

```go
// 豌豆射手攻击：身体攻击动画 + 头部空闲动画
func peashooterStartShooting(entityID ecs.Entity, reanimSys *systems.ReanimSystem) {
    // 方法 1：使用 PlayAnimations（推荐）
    reanimSys.PlayAnimations(entityID, []string{"anim_shooting", "anim_head_idle"})

    // 方法 2：使用 AddAnimation（增量式）
    // reanimSys.PlayAnimation(entityID, "anim_shooting")
    // reanimSys.AddAnimation(entityID, "anim_head_idle")
}

// 攻击结束，恢复空闲
func peashooterStopShooting(entityID ecs.Entity, reanimSys *systems.ReanimSystem) {
    reanimSys.PlayAnimation(entityID, "anim_idle")
}
```

### 增量控制动画

保留已有动画，叠加新动画：

```go
// 僵尸行走时着火
func zombieStartBurning(entityID ecs.Entity, reanimSys *systems.ReanimSystem) {
    // 保留行走动画，叠加燃烧特效
    reanimSys.AddAnimation(entityID, "anim_burning")
}

// 熄灭火焰
func zombieStopBurning(entityID ecs.Entity, reanimSys *systems.ReanimSystem) {
    reanimSys.RemoveAnimation(entityID, "anim_burning")
    // 行走动画继续播放
}
```

## 常见场景

### 场景 1：植物攻击动画

完整的豌豆射手攻击逻辑：

```go
// pkg/systems/behavior_system.go

func (bs *BehaviorSystem) handlePeashooterBehavior(
    entityID ecs.Entity,
    plantComp *components.PlantComponent,
    deltaTime float64,
) {
    // 检测是否有僵尸
    hasTarget := bs.detectZombiesInLane(plantComp.LaneIndex)

    if hasTarget {
        // 进入攻击状态
        if plantComp.State != "shooting" {
            plantComp.State = "shooting"
            // 播放攻击动画（身体 + 头部）
            bs.reanimSystem.PlayAnimations(entityID, []string{
                "anim_shooting",
                "anim_head_idle",
            })
        }

        // 攻击逻辑...
        plantComp.ShootTimer += deltaTime
        if plantComp.ShootTimer >= plantComp.ShootInterval {
            bs.shootPea(entityID, plantComp)
            plantComp.ShootTimer = 0
        }
    } else {
        // 恢复空闲状态
        if plantComp.State != "idle" {
            plantComp.State = "idle"
            // 播放空闲动画
            bs.reanimSystem.PlayAnimation(entityID, "anim_idle")
        }
    }
}
```

### 场景 2：僵尸装备控制

路障僵尸的装备显示：

```go
// pkg/entities/zombie_factory.go

func CreateConeheadZombie(
    em *ecs.EntityManager,
    rm *game.ResourceManager,
    x, y float64,
) ecs.Entity {
    entity := em.NewEntity()

    // 创建 ReanimComponent
    reanimComp := &components.ReanimComponent{
        // ... 其他字段 ...

        // 使用 VisibleTracks 控制装备显示
        VisibleTracks: map[string]bool{
            "Zombie_body":       true,  // 身体
            "Zombie_head":       true,  // 头部
            "Zombie_outerarm":   true,  // 外侧手臂
            "Zombie_innerarm":   true,  // 内侧手臂
            "anim_cone":         true,  // 路障装备
            "anim_screendoor":   false, // 铁门（隐藏）
            "anim_football_helmet": false, // 橄榄球头盔（隐藏）
        },
    }

    ecs.AddComponent(em, entity, reanimComp)

    return entity
}

// 路障被打掉时
func removeCone(entityID ecs.Entity, em *ecs.EntityManager) {
    reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](em, entityID)
    if ok {
        reanimComp.VisibleTracks["anim_cone"] = false  // 隐藏路障
    }
}
```

### 场景 3：主菜单动画（异步模式）

SelectorScreen 云朵独立循环：

```go
// pkg/entities/selector_screen_factory.go

func CreateSelectorScreen(
    em *ecs.EntityManager,
    rm *game.ResourceManager,
) ecs.Entity {
    entity := em.NewEntity()

    reanimComp := &components.ReanimComponent{
        // ... 其他字段 ...

        // 设置异步模式：每个云朵独立循环
        TrackMapping: map[string]string{
            "Cloud1": "anim_cloud1",
            "Cloud2": "anim_cloud2",
            "Cloud3": "anim_cloud3",
            "Cloud4": "anim_cloud4",
            "Cloud5": "anim_cloud5",
            "Cloud6": "anim_cloud6",
            "Cloud7": "anim_cloud7",
            "Grass1":  "anim_grass1",
            "Grass2":  "anim_grass2",
            "Grass3":  "anim_grass3",
            "Grass4":  "anim_grass4",
            "Grass5":  "anim_grass5",
            "Grass6":  "anim_grass6",
            "Grass7":  "anim_grass7",
        },

        // 初始化每个动画的独立状态
        Anims: map[string]*components.AnimState{
            "anim_cloud1": {
                Name:      "anim_cloud1",
                IsActive:  true,
                IsLooping: true,
                Frame:     0,  // 从第 0 帧开始
            },
            "anim_cloud2": {
                Name:      "anim_cloud2",
                IsActive:  true,
                IsLooping: true,
                Frame:     20,  // 从第 20 帧开始（错开）
            },
            // ... 其他云朵和草丛动画 ...
        },
    }

    ecs.AddComponent(em, entity, reanimComp)

    return entity
}
```

## API 参考

### PlayAnimation

播放单个动画，清空所有已有动画。

```go
func (rs *ReanimSystem) PlayAnimation(entityID ecs.Entity, animName string) error
```

**参数：**
- `entityID`: 实体 ID
- `animName`: 动画名称（如 "anim_idle"）

**返回：**
- `error`: 如果动画不存在或实体无 ReanimComponent，返回错误

**示例：**
```go
reanimSystem.PlayAnimation(sunflowerID, "anim_idle")
```

### PlayAnimations

播放多个动画（叠加），清空所有已有动画。

```go
func (rs *ReanimSystem) PlayAnimations(entityID ecs.Entity, animNames []string) error
```

**参数：**
- `entityID`: 实体 ID
- `animNames`: 动画名称列表

**返回：**
- `error`: 如果任一动画不存在，返回错误

**示例：**
```go
reanimSystem.PlayAnimations(peashooterID, []string{"anim_shooting", "anim_head_idle"})
```

### AddAnimation

增量添加动画，保留已有动画。

```go
func (rs *ReanimSystem) AddAnimation(entityID ecs.Entity, animName string) error
```

**参数：**
- `entityID`: 实体 ID
- `animName`: 动画名称

**返回：**
- `error`: 如果动画不存在，返回错误

**示例：**
```go
reanimSystem.AddAnimation(zombieID, "anim_burning")
```

### RemoveAnimation

移除指定动画，保留其他动画。

```go
func (rs *ReanimSystem) RemoveAnimation(entityID ecs.Entity, animName string) error
```

**参数：**
- `entityID`: 实体 ID
- `animName`: 要移除的动画名称

**返回：**
- `error`: 如果实体无 ReanimComponent，返回错误

**示例：**
```go
reanimSystem.RemoveAnimation(zombieID, "anim_burning")
```

## 常见问题

**Q: 如何判断使用 PlayAnimation 还是 PlayAnimations？**

A:
- 如果实体只需要播放一个动画（如向日葵空闲），使用 `PlayAnimation`
- 如果实体需要同时控制多个部件（如豌豆射手攻击时身体和头部），使用 `PlayAnimations`

**Q: AddAnimation 和 PlayAnimations 的区别？**

A:
- `PlayAnimations`：清空所有已有动画，然后添加新动画
- `AddAnimation`：保留已有动画，叠加新动画

**Q: 如何调试动画问题？**

A: 使用 `--verbose` 标志运行游戏：
```bash
go run . --verbose
```
这会输出动画播放的详细日志。

---

**版本**: 1.0
**最后更新**: 2025-11-05
**相关 Story**: 6.8, 6.9, 6.10
