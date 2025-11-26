# Sprint 变更提案：坚果墙受击粒子效果

**日期**: 2025-11-26
**状态**: ✅ 已批准
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
- ❌ 坚果墙被啃食时的粒子碎屑效果（`WallnutEatLarge.xml`、`WallnutEatSmall.xml`）未实现

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

## 具体修改提案

### 修改文件

**`pkg/systems/behavior/zombie_behavior_handler.go`**

在 `handleZombieEatingBehavior` 函数中（约第 589 行），当僵尸对植物造成伤害后，添加坚果墙粒子效果触发逻辑：

```go
// 在 plantHealth.CurrentHealth -= config.ZombieEatingDamage 之后添加：

// 坚果墙被啃食时触发粒子效果
if plantComp, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, plantID); ok {
    if plantComp.PlantType == components.PlantWallnut {
        // 获取植物位置
        if plantPos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, plantID); ok {
            // 随机选择大碎屑或小碎屑效果
            effectName := "WallnutEatSmall"
            if rand.Float64() < 0.3 { // 30% 概率触发大碎屑
                effectName = "WallnutEatLarge"
            }
            _, err := entities.CreateParticleEffect(
                s.entityManager,
                s.resourceManager,
                effectName,
                plantPos.X,
                plantPos.Y,
            )
            if err != nil {
                log.Printf("[BehaviorSystem] 警告：创建坚果墙碎屑粒子效果失败: %v", err)
            }
        }
    }
}
```

### 资源文件

粒子效果配置文件已存在：
- `assets/effect/particles/WallnutEatLarge.xml` - 大碎屑效果
- `assets/effect/particles/WallnutEatSmall.xml` - 小碎屑效果

---

## 验收标准

1. ✅ 僵尸啃食坚果墙时，每次造成伤害都会产生碎屑粒子效果
2. ✅ 粒子效果位置与坚果墙位置一致
3. ✅ 大碎屑（`WallnutEatLarge`）和小碎屑（`WallnutEatSmall`）随机出现（约 30%/70%）
4. ✅ 不影响其他植物的啃食行为
5. ✅ 不影响游戏性能

---

## 测试方法

1. 启动游戏，进入 1-3 或 1-4 关卡（有坚果墙可用）
2. 种植坚果墙，等待僵尸接近并开始啃食
3. 观察是否有碎屑粒子效果产生
4. 确认粒子位置正确、效果自然
5. 验证大小碎屑随机出现

---

## 行动计划

| 步骤 | 任务 | 负责角色 |
|------|------|----------|
| 1 | 修改 `zombie_behavior_handler.go` 添加粒子触发逻辑 | 开发 |
| 2 | 添加必要的 import（`math/rand`、`entities`） | 开发 |
| 3 | 运行游戏验证粒子效果 | 开发 |
| 4 | 确认不影响现有功能 | 开发 |

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
