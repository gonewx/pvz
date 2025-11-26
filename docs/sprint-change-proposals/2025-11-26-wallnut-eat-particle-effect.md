# Sprint 变更提案：坚果墙受击粒子效果

**日期**: 2025-11-26
**状态**: ✅ 已实现
**优先级**: 高（需要立即实现）

---

## 问题摘要

**问题**: 坚果墙被僵尸啃食时缺少粒子碎屑效果

**触发来源**:
- `.meta/prompts.md` 第 68-69 行
- `.meta/data/plant_hit_effects.md` 第 11-12 行

**影响范围**: 仅影响视觉反馈，不影响游戏核心机制

**现有实现状态**:
- ✅ 坚果墙外观破损状态（`Wallnut_cracked1.png`、`Wallnut_cracked2.png`）已实现
- ✅ 坚果墙被啃食时的粒子碎屑效果（`WallnutEatLarge.xml`、`WallnutEatSmall.xml`）已实现

---

## Epic 影响摘要

| 影响项 | 状态 |
|--------|------|
| 当前 Epic | 无需修改 |
| 未来 Epic | 无影响 |
| 新增 Epic | 不需要 |
| 文档调整 | 无需修改 |

---

## 推荐路径

**选项 1: 直接调整/集成** - 在现有代码中添加粒子效果触发逻辑

---

## 具体修改提案（已实现）

### 粒子效果触发逻辑

两种粒子效果有不同的触发时机：

| 效果 | 触发时机 | 位置 |
|------|----------|------|
| `WallnutEatSmall` | 每次啃食伤害时 | 僵尸嘴巴位置（啃食接触点） |
| `WallnutEatLarge` | 受损状态变化时（图片切换） | 坚果墙位置 |

### 修改文件

#### 1. `pkg/systems/behavior/zombie_behavior_handler.go`

在 `handleZombieEatingBehavior` 函数中，每次啃食伤害后触发小碎屑效果：

```go
// 坚果墙被啃食时触发小碎屑粒子效果
// WallnutEatSmall: 每次啃食伤害时触发
if plantComp, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, plantID); ok {
    if plantComp.PlantType == components.PlantWallnut {
        // 粒子位置：僵尸嘴巴位置（啃食接触点）
        particleX := pos.X + config.ZombieEatParticleOffsetX
        particleY := pos.Y + config.ZombieEatParticleOffsetY
        _, err := entities.CreateParticleEffect(
            s.entityManager,
            s.resourceManager,
            "WallnutEatSmall",
            particleX,
            particleY,
        )
        if err != nil {
            log.Printf("[BehaviorSystem] 警告：创建坚果墙小碎屑粒子效果失败: %v", err)
        }
    }
}
```

#### 2. `pkg/systems/behavior/plant_behavior_handler.go`

在 `handleWallnutBehavior` 函数中，受损状态变化时触发大碎屑效果：

```go
// 如果图片不同，则替换并触发大碎屑粒子效果
if currentBodyImage != targetBodyImage {
    // 检查是否是从更好的状态变为更差的状态
    if hasPlant && newDamageState > plantComp.WallnutDamageState {
        // 状态变差，触发大碎屑粒子效果
        if plantPos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID); ok {
            _, err := entities.CreateParticleEffect(
                s.entityManager,
                s.resourceManager,
                "WallnutEatLarge",
                plantPos.X,
                plantPos.Y,
            )
            // ...
        }
        plantComp.WallnutDamageState = newDamageState
    }
    // 切换图片...
}
```

#### 3. `pkg/components/plant.go`

添加坚果墙受损状态跟踪字段：

```go
// WallnutDamageState 坚果墙受损状态（0=完好, 1=轻伤, 2=重伤）
WallnutDamageState int
```

#### 4. `pkg/config/unit_config.go`

添加僵尸嘴巴位置偏移常量：

```go
// ZombieEatParticleOffsetX 僵尸啃食粒子效果 X 偏移量
ZombieEatParticleOffsetX = 35.0

// ZombieEatParticleOffsetY 僵尸啃食粒子效果 Y 偏移量
ZombieEatParticleOffsetY = -20.0
```

#### 5. 修复图片路径大小写

修正 `handleWallnutBehavior` 中的图片路径，使用正确的大小写：
- `assets/reanim/Wallnut_body.png`
- `assets/reanim/Wallnut_cracked1.png`
- `assets/reanim/Wallnut_cracked2.png`

### 资源文件

粒子效果配置文件已存在：
- `assets/effect/particles/WallnutEatLarge.xml` - 大碎屑效果
- `assets/effect/particles/WallnutEatSmall.xml` - 小碎屑效果

---

## 验收标准

1. ✅ 僵尸啃食坚果墙时，每次造成伤害都会产生小碎屑粒子效果（WallnutEatSmall）
2. ✅ 坚果墙受损状态变化时，产生大碎屑粒子效果（WallnutEatLarge）
3. ✅ 小碎屑粒子位置在僵尸嘴巴位置（啃食接触点）
4. ✅ 坚果墙受损图片正确切换（修复文件名大小写问题）
5. ✅ 不影响其他植物的啃食行为
6. ✅ 不影响游戏性能

---

## 测试方法

1. 启动游戏，进入 1-3 或 1-4 关卡（有坚果墙可用）
2. 种植坚果墙，等待僵尸接近并开始啃食
3. 观察每次啃食时是否有小碎屑粒子效果产生（在接触点位置）
4. 观察坚果墙受损状态变化时是否有大碎屑粒子效果产生
5. 确认坚果墙图片正确切换（完好→轻伤→重伤）

---

## 行动计划

| 步骤 | 任务 | 负责角色 | 状态 |
|------|------|----------|------|
| 1 | 修复图片路径大小写问题 | 开发 | ✅ |
| 2 | 修改粒子触发逻辑（Small/Large 分离） | 开发 | ✅ |
| 3 | 添加 WallnutDamageState 字段跟踪状态变化 | 开发 | ✅ |
| 4 | 修正粒子位置为僵尸嘴巴接触点 | 开发 | ✅ |
| 5 | 添加配置常量 | 开发 | ✅ |

---

## 变更检查清单完成状态

- [x] 第 1 节：理解触发与上下文
- [x] 第 2 节：Epic 影响评估
- [x] 第 3 节：文档影响分析
- [x] 第 4 节：前进路径评估
- [x] 第 5 节：变更提案组件
- [x] 第 6 节：最终审查与交接

---

## 批准记录

- **批准时间**: 2025-11-26
- **批准人**: 用户

## 实现记录

- **实现时间**: 2025-11-26
- **实现人**: James (Dev Agent)
- **修改文件**:
  - `pkg/systems/behavior/zombie_behavior_handler.go`
  - `pkg/systems/behavior/plant_behavior_handler.go`
  - `pkg/components/plant.go`
  - `pkg/config/unit_config.go`
