# 植物卡片系统重构评估报告

## 📋 问题确认

**用户理解正确！** 原版PVZ资源采用的是**组合式设计**：

- ❌ **旧实现**：每个植物一张完整的静态卡片图 (`card_sunFlower.png`, `card_peashooter.png`)
- ✅ **新资源**：卡片背景 + 植物图标动态组合 (`SeedPacket_Larger.png` + `seeds.png`)

---

## 🔍 资源结构分析

### 1. 新资源文件

| 文件名 | 尺寸 | 用途 | XML定义 |
|--------|------|------|---------|
| `SeedPacket_Larger.png` | 100x140 | 卡片背景框 | `IMAGE_SEEDPACKET_LARGER` |
| `seeds.png` | 450x70 (9列) | 植物图标精灵图 | `IMAGE_SEEDS` (cols="9") |
| `SeedPacketSilhouette.png` | - | 卡片禁用状态轮廓 | `IMAGE_SEEDPACKETSILHOUETTE` |
| `SeedPacketFlash.png` | - | 卡片闪光效果 | `PARTICLE_SEEDPACKETFLASH` |

### 2. seeds.png 切片信息

```
总尺寸: 450x70 像素
列数: 9
每个植物图标: 50x70 像素
```

**推测的图标顺序**（基于原版PVZ）：
```
索引 0: 豌豆射手 (Peashooter)
索引 1: 向日葵 (Sunflower)
索引 2: 樱桃炸弹 (Cherry Bomb)
索引 3: 坚果墙 (Wall-nut)
索引 4: 土豆雷 (Potato Mine)
索引 5: 寒冰射手 (Snow Pea)
索引 6: 大嘴花 (Chomper)
索引 7: 双发射手 (Repeater)
索引 8: 喷菇 (Puff-shroom)
```

### 3. 渲染层次结构

原版PVZ卡片由多层组成：
```
[底层] 卡片背景框 (SeedPacket_Larger.png)
  ↓
[中层] 植物图标 (seeds.png切片)
  ↓
[上层] 阳光数字 (动态绘制文本)
  ↓
[特效] 冷却遮罩 (黑色半透明矩形，从上往下)
  ↓
[特效] 禁用状态 (灰色或轮廓)
  ↓
[特效] 闪光效果 (SeedPacketFlash.png)
```

---

## 🎯 当前实现的问题

### 问题1: 硬编码的旧资源路径

**文件**: `pkg/entities/plant_card_factory.go:26-36`

```go
case components.PlantSunflower:
    imagePath = "assets/images/Cards/card_sunFlower.png"  // ❌ 文件不存在
case components.PlantPeashooter:
    imagePath = "assets/images/Cards/card_peashooter.png" // ❌ 文件不存在
case components.PlantWallnut:
    imagePath = "assets/images/Cards/card_wallnut.png"    // ❌ 文件不存在
```

**结果**: 卡片图片加载失败，界面不显示卡片

### 问题2: 设计不匹配

当前设计使用 `SpriteComponent` 存储**单张完整卡片图**，但新资源需要：
- 背景框
- 植物图标（从精灵图切片）
- 动态文本（阳光数字）
- 动态遮罩（冷却/禁用效果）

---

## 🛠️ 重构方案

### 方案概述

保持 `SpriteComponent` 架构（符合CLAUDE.md的UI设计原则），但改为**多层组合渲染**。

### 核心改动

#### 1. 扩展 `PlantCardComponent` 存储多层资源

```go
type PlantCardComponent struct {
    PlantType       PlantType
    SunCost         int
    CooldownTime    float64
    CurrentCooldown float64
    IsAvailable     bool

    // 新增：多层资源引用
    BackgroundImage  *ebiten.Image // 卡片背景框
    PlantIconImage   *ebiten.Image // 植物图标（从seeds.png切片）
    PlantIconIndex   int            // 在seeds.png中的索引位置
}
```

**优点**:
- 不引入新组件，保持架构简洁
- 所有渲染数据集中管理

#### 2. 修改 `plant_card_factory.go` 使用资源ID

```go
func NewPlantCardEntity(em *ecs.EntityManager, rm *game.ResourceManager, plantType components.PlantType, x, y float64) (ecs.EntityID, error) {
    // 加载共享资源（所有卡片共用同一个背景）
    backgroundImg, err := rm.LoadImageByID("IMAGE_SEEDPACKET_LARGER")
    if err != nil {
        return 0, fmt.Errorf("failed to load card background: %w", err)
    }

    // 加载并切片seeds.png
    seedsImg, err := rm.LoadImageByID("IMAGE_SEEDS")
    if err != nil {
        return 0, fmt.Errorf("failed to load seeds spritesheet: %w", err)
    }

    // 根据植物类型确定图标索引和属性
    var iconIndex int
    var sunCost int
    var cooldownTime float64

    switch plantType {
    case components.PlantPeashooter:
        iconIndex = 0
        sunCost = 100
        cooldownTime = 7.5
    case components.PlantSunflower:
        iconIndex = 1
        sunCost = 50
        cooldownTime = 7.5
    case components.PlantWallnut:
        iconIndex = 3
        sunCost = 50
        cooldownTime = 30.0
    }

    // 切片提取植物图标
    iconWidth := 50
    iconHeight := 70
    plantIcon := seedsImg.SubImage(image.Rect(
        iconIndex*iconWidth, 0,
        (iconIndex+1)*iconWidth, iconHeight,
    )).(*ebiten.Image)

    // 创建实体并添加组件
    entity := em.CreateEntity()

    em.AddComponent(entity, &components.PositionComponent{X: x, Y: y})

    // SpriteComponent 不再使用（保留兼容性，设为nil）
    em.AddComponent(entity, &components.SpriteComponent{Image: nil})

    em.AddComponent(entity, &components.PlantCardComponent{
        PlantType:        plantType,
        SunCost:          sunCost,
        CooldownTime:     cooldownTime,
        CurrentCooldown:  0.0,
        IsAvailable:      true,
        BackgroundImage:  backgroundImg,
        PlantIconImage:   plantIcon,
        PlantIconIndex:   iconIndex,
    })

    // 其他组件...
    return entity, nil
}
```

#### 3. 重构 `PlantCardRenderSystem` 实现多层渲染

```go
func (s *PlantCardRenderSystem) Draw(screen *ebiten.Image) {
    entities := s.entityManager.GetEntitiesWith(
        reflect.TypeOf(&components.PlantCardComponent{}),
        reflect.TypeOf(&components.PositionComponent{}),
    )

    for _, entityID := range entities {
        card := // 获取 PlantCardComponent
        pos := // 获取 PositionComponent

        // 层1: 绘制卡片背景
        if card.BackgroundImage != nil {
            op := &ebiten.DrawImageOptions{}
            op.GeoM.Scale(s.cardScale, s.cardScale)
            op.GeoM.Translate(pos.X, pos.Y)
            screen.DrawImage(card.BackgroundImage, op)
        }

        // 层2: 绘制植物图标（居中对齐）
        if card.PlantIconImage != nil {
            op := &ebiten.DrawImageOptions{}
            // 计算居中偏移
            offsetX := (100 - 50) / 2.0 * s.cardScale
            offsetY := 10.0 * s.cardScale
            op.GeoM.Scale(s.cardScale, s.cardScale)
            op.GeoM.Translate(pos.X+offsetX, pos.Y+offsetY)
            screen.DrawImage(card.PlantIconImage, op)
        }

        // 层3: 绘制阳光数字
        s.drawSunCost(screen, pos.X, pos.Y, card.SunCost)

        // 层4: 绘制冷却遮罩
        if card.CurrentCooldown > 0 {
            s.drawCooldownMask(screen, pos.X, pos.Y, card)
        }

        // 层5: 绘制禁用效果
        if !card.IsAvailable {
            s.drawDisabledEffect(screen, pos.X, pos.Y)
        }
    }
}
```

#### 4. 更新 YAML 配置

确保以下资源ID已正确添加到 `assets/config/resources.yaml`：

```yaml
loadingimages:
    images:
        - id: IMAGE_SEEDS
          path: images/seeds.png
          cols: 9
        - id: IMAGE_SEEDPACKET_LARGER
          path: images/SeedPacket_Larger.png
        - id: IMAGE_SEEDPACKETSILHOUETTE
          path: images/SeedPacketSilhouette.png
```

---

## 🎨 视觉效果对比

### 旧实现
```
[静态完整卡片图]
  └─ 包含背景+植物+数字，无法动态调整
```

### 新实现
```
[卡片背景框]
  └─ [植物图标] (可替换)
      └─ [阳光数字] (动态绘制)
          └─ [冷却遮罩] (动态渐变)
              └─ [禁用效果] (可选)
```

**优势**：
- ✅ 真实还原原版PVZ视觉效果
- ✅ 支持动态冷却进度显示
- ✅ 资源占用更小（共享背景框）
- ✅ 易于扩展新植物（只需添加seeds.png索引）

---

## 📝 实施步骤

1. ✅ **分析资源结构** - 已完成
2. ⏳ **更新YAML配置** - 添加IMAGE_SEEDS等资源ID
3. ⏳ **扩展PlantCardComponent** - 添加多层图像字段
4. ⏳ **重构plant_card_factory.go** - 使用新资源ID和切片逻辑
5. ⏳ **重构PlantCardRenderSystem** - 实现多层渲染
6. ⏳ **添加字体渲染** - 绘制阳光数字
7. ⏳ **测试验证** - 确认卡片正常显示和交互

---

## ⚠️ 注意事项

1. **保持SpriteComponent架构**: 卡片仍是UI元素，不使用ReanimComponent（符合CLAUDE.md设计原则）

2. **seeds.png索引映射**: 需要验证实际图标顺序，可能需要调整索引

3. **字体加载**: 需要确保阳光数字的字体资源已加载（可能需要添加到ResourceManager）

4. **性能优化**: 背景图可以作为静态资源缓存，避免重复加载

5. **向后兼容**: 保留SpriteComponent（设为nil），避免影响其他系统

---

## 🎯 总结

**评估结论**: 用户的理解完全正确。当前卡片系统需要从"单图模式"重构为"组合模式"以匹配新资源结构。

**推荐方案**: 扩展PlantCardComponent存储多层资源，重构渲染系统实现动态组合绘制。

**工作量估计**: 中等（约2-3小时）
- 代码修改: 3个文件
- 配置更新: 1个YAML文件
- 测试验证: 30分钟

**优先级**: 🔴 高 - 阻止卡片显示，影响核心玩法
