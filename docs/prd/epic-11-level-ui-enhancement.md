# Epic 11: 关卡 UI 增强 (Level UI Enhancement)

## Epic Goal (史诗目标)

完善《植物大战僵尸》关卡界面的视觉体验和信息展示，实现铺草皮土粒飞溅特效、最后一波僵尸提示动画和完整的关卡进度条系统，提升游戏的沉浸感和信息可读性，确保关卡 UI 符合原版游戏标准。

---

## Epic Description (史诗描述)

### Existing System Context (现有系统上下文)

**当前相关功能:**
- ✅ 粒子特效系统 (`ParticleSystem`, `EmitterComponent` - Epic 7)
- ✅ 关卡流程管理 (`LevelSystem`, `WaveSpawnSystem` - Story 5.5)
- ✅ Reanim 动画系统 (`ReanimSystem` - Epic 6)
- ✅ 关卡配置系统 (`LevelConfig` - Story 8.1)
- ✅ 植物种植粒子特效 (`Planting` 粒子 - Story 10.4)
- ✅ 资源管理器 (`ResourceManager`)

**技术栈:**
- Go 语言 + Ebitengine 引擎
- ECS 架构 (Entity-Component-System)
- XML 配置驱动粒子系统
- Reanim 骨骼动画系统
- YAML 关卡配置

**集成点:**
- 关卡场景 (`pkg/scenes/game_scene.go`)
- 关卡系统 (`pkg/systems/level_system.go`)
- 渲染系统 (`pkg/systems/render_system.go`)
- 实体工厂 (`pkg/entities/`)

---

### Enhancement Details (增强功能详情)

**核心功能模块:**

#### 1. 铺草皮土粒飞溅特效 (Sod Roll Particle Effect)
- **触发时机**: 关卡开始时，草皮从右向左铺设动画
- **粒子效果**: 每铺一段草皮，产生土粒飞溅效果
- **原版配置**: 使用 `assets/effect/particles/SodRoll.xml` 粒子配置
- **视觉还原**: 土粒从草皮边缘向外飞溅，自然落下
- **性能考虑**: 粒子批量渲染，铺草皮动画不阻塞游戏流程

#### 2. 最后一波僵尸提示动画 (Final Wave Warning Animation)
- **触发条件**: 最后一波僵尸（旗帜波）到来前 3 秒
- **提示动画**: 屏幕中央播放 `FinalWave.reanim` 动画
- **音效配合**: 播放"僵尸来袭"音效
- **原版忠实**: 使用原版 Reanim 动画资源
- **交互处理**: 提示动画播放期间游戏继续运行

#### 3. 关卡进度条完善 (Level Progress Bar Enhancement)

**当前问题:**
- ❌ 进度条位置不准确
- ❌ 进度显示不正确（绿色进度条缺失）
- ❌ 僵尸头图标位置不随进度移动
- ❌ 缺少关卡文本显示（如"关卡 1-1"）

**完善目标:**
- ✅ **位置优化**: 进度条显示在屏幕右下角正确位置
- ✅ **绿色进度条**: 使用 `FlagMeterLevelProgress.png` 显示当前进度
- ✅ **僵尸头跟随**: 僵尸头图标位置随进度百分比移动
- ✅ **关卡文本**: 开场前显示"关卡 1-1"，进攻开始后显示完整进度条
- ✅ **资源使用**:
  - `FlagMeter.png` - 进度条背景
  - `FlagMeterLevelProgress.png` - 绿色进度填充
  - `FlagMeterParts.png` - 旗帜和僵尸头图标
- ✅ **进度计算**: 基于已消灭僵尸数 / 总僵尸数

---

### Integration Approach (集成方法)

#### 1. 铺草皮粒子特效集成

**新增配置:**
```go
// LevelConfig 扩展（data/levels/level-X-Y.yaml）
sodRollAnimation: true  // 是否启用铺草皮动画（教学关卡为 false）
```

**GameScene 初始化修改:**
```go
// GameScene.initLevel() 中添加
if levelConfig.SodRollAnimation {
    s.playSodRollAnimation()  // 播放铺草皮动画 + 粒子效果
}
```

**粒子效果触发:**
- 铺草皮动画每前进一格，触发一次 SodRoll 粒子效果
- 使用 `entities.NewSodRollParticleEffect(em, rm, worldX, worldY)`

#### 2. 最后一波提示动画集成

**新增组件:**
```go
// FinalWaveWarningComponent - 最后一波提示组件
type FinalWaveWarningComponent struct {
    IsTriggered   bool      // 是否已触发
    AnimEntity    EntityID  // FinalWave.reanim 动画实体
    DisplayTime   float64   // 显示时长（秒）
}
```

**LevelSystem 修改:**
```go
// LevelSystem.Update() 中添加检测逻辑
func (s *LevelSystem) checkFinalWaveWarning(deltaTime float64) {
    if !s.finalWaveWarningTriggered && s.isFinalWaveApproaching() {
        s.triggerFinalWaveWarning()
    }
}

func (s *LevelSystem) isFinalWaveApproaching() bool {
    // 检测下一波是否是最后一波，且时间剩余 <= 3 秒
    nextWave := s.getNextWave()
    return nextWave.IsFinal && (nextWave.Time - s.currentTime) <= 3.0
}
```

**动画播放:**
- 使用 `ReanimSystem.PlayAnimation()` 播放 `FinalWave.reanim`
- 动画实体添加 `UIComponent` 标记，渲染在最上层
- 播放 2-3 秒后自动移除实体

#### 3. 关卡进度条完善集成

**新增组件:**
```go
// LevelProgressBarComponent - 关卡进度条组件
type LevelProgressBarComponent struct {
    // 资源
    BackgroundImage     *ebiten.Image  // FlagMeter.png
    ProgressBarImage    *ebiten.Image  // FlagMeterLevelProgress.png
    PartsImage          *ebiten.Image  // FlagMeterParts.png（精灵图）

    // 进度数据
    TotalZombies        int      // 总僵尸数
    KilledZombies       int      // 已击杀僵尸数
    ProgressPercent     float64  // 进度百分比 (0.0 - 1.0)

    // 显示配置
    LevelText           string   // 关卡文本（如"关卡 1-1"）
    ShowLevelTextOnly   bool     // 是否只显示文本（进攻前）

    // 位置配置
    X, Y                float64  // 进度条位置（右下角）
    ZombieHeadOffsetX   float64  // 僵尸头相对进度条的 X 偏移
}
```

**渲染逻辑:**
```go
// LevelProgressBarRenderSystem.Draw()
func (s *LevelProgressBarRenderSystem) Draw(screen *ebiten.Image) {
    if pb.ShowLevelTextOnly {
        // 只绘制关卡文本（屏幕中央上方）
        s.drawLevelText(screen, pb.LevelText)
    } else {
        // 绘制完整进度条
        s.drawProgressBar(screen, pb)
        s.drawProgressFill(screen, pb)  // 绿色进度条
        s.drawZombieHead(screen, pb)    // 僵尸头图标
        s.drawLevelText(screen, pb.LevelText)
    }
}
```

**进度更新:**
- 僵尸死亡时，`LevelSystem` 更新 `KilledZombies`
- 计算进度百分比：`ProgressPercent = KilledZombies / TotalZombies`
- 僵尸头位置：`zombieHeadX = progressBarStartX + progressBarWidth * ProgressPercent`

**关卡文本切换:**
- 关卡开始前：`ShowLevelTextOnly = true`（只显示"关卡 1-1"）
- 第一波僵尸生成后：`ShowLevelTextOnly = false`（显示完整进度条）

---

### Stories (用户故事)

本 Epic 包含 5 个用户故事，按优先级和实现顺序排序：

1. **Story 11.1: 修复植物卡片图标渲染通用性问题** ✅ **Done**
   - 修复向日葵等植物在奖励动画中的卡片图标渲染问题
   - 实现 `PrepareStaticPreview` 方法，不依赖动画定义轨道
   - 实现通用的植物卡片图标渲染机制（支持所有植物类型）
   - 优先级：⭐⭐⭐⭐⭐ 高（Brownfield 修复）
   - 详见：`docs/stories/11.1.fix-plant-card-icon-rendering.md`

2. **Story 11.2: 关卡进度条完善** ✅ **Done**
   - 修复进度条位置和显示问题
   - 实现绿色进度条填充（使用 `FlagMeterLevelProgress.png`）
   - 实现僵尸头跟随进度移动
   - 实现关卡文本显示（开场前/进攻中）
   - 优先级：⭐⭐⭐⭐⭐ 高
   - 详见：`docs/stories/11.2.story.md`

3. **Story 11.3: 最后一波僵尸提示动画**
   - 实现最后一波检测逻辑
   - 播放 FinalWave.reanim 动画
   - 配合音效和视觉提示
   - 优先级：⭐⭐⭐⭐ 中高

4. **Story 11.4: 铺草皮土粒飞溅特效**
   - 实现铺草皮动画（可选）
   - 触发 SodRoll.xml 粒子效果
   - 性能优化和视觉调优
   - 优先级：⭐⭐⭐ 中

5. **Story 11.5: 原版进度条机制实现** 🆕 **Draft**
   - 实现双段式进度条结构（红字波段 + 普通波段）
   - 实现双重进度计算（时间进度 vs 血量削减取最大值）
   - 实现虚拟/现实双层进度条平滑追踪
   - 支持进度超过右端点并在下波回退
   - 优先级：⭐⭐⭐⭐ 高
   - 详见：`docs/stories/11.5.story.md`
   - 变更提案：`docs/sprint-change-proposals/2025-11-28-story-11.5-progress-bar-mechanism.md`

---

### Compatibility Requirements (兼容性要求)

- ✅ **ECS 架构一致性**: 所有新功能遵循 ECS 模式，零耦合原则
- ✅ **泛型 API 使用**: 使用 Epic 9 的泛型 ECS API，类型安全
- ✅ **资源管理**: 通过 ResourceManager 统一加载资源
- ✅ **配置驱动**: 关卡配置 YAML 支持新增字段
- ✅ **性能考虑**: 不影响现有系统的帧率和流畅度
- ✅ **原版忠实**: 所有功能还原原版 PC 游戏效果

---

### Risk Mitigation (风险缓解)

**主要风险:**
1. **进度条计算错误** - 僵尸总数统计不准确导致进度显示错误
2. **粒子性能影响** - 铺草皮粒子过多影响帧率
3. **提示动画时机** - 最后一波提示过早或过晚
4. **资源加载失败** - FlagMeter 资源缺失或路径错误

**缓解措施:**
1. **僵尸计数验证** - 关卡加载时预计算总僵尸数，运行时校验
2. **粒子限制** - SodRoll 粒子每格限制数量，使用对象池
3. **精确计时** - 使用关卡时间戳检测，3 秒提前量可配置
4. **资源校验** - 启动时校验关键资源存在性，缺失时报错

**回滚计划:**
- 每个 Story 独立实现，可单独禁用
- 使用功能开关（如配置文件中的 `enableFinalWaveWarning: true`）
- 不影响核心游戏流程，回滚不破坏已有功能

---

### Definition of Done (完成定义)

- ✅ Story 11.1（植物卡片图标渲染修复）已完成，QA 通过
- ✅ 所有剩余 3 个用户故事完成，AC 全部满足
- ✅ 关卡进度条位置准确，进度显示正确，僵尸头跟随流畅
- ✅ 关卡文本在正确时机显示（开场前/进攻中）
- ✅ 最后一波提示动画准时触发，视觉效果符合原版
- ✅ 铺草皮粒子特效自然流畅，不影响性能
- ✅ 现有功能无回归，通过 QA 测试
- ✅ 代码符合编码规范，通过 linter 检查
- ✅ 所有新功能有对应的单元测试（覆盖率 > 80%）
- ✅ 文档更新（CLAUDE.md）

---

## Success Criteria (成功标准)

本 Epic 成功完成的标准：

1. **功能完整性** - 所有 3 个功能模块实现并正常工作
2. **原版忠实度** - 进度条、提示动画、粒子效果还原原版表现
3. **性能稳定** - 帧率稳定在 60 FPS，无明显卡顿
4. **用户体验** - 信息展示清晰，视觉反馈及时准确
5. **代码质量** - 遵循 ECS 架构，代码可读性和可维护性高
6. **测试覆盖** - 所有关键功能有单元测试和集成测试
7. **无回归** - 现有功能（关卡流程、UI 显示）无破坏

---

## Dependencies (依赖关系)

**前置依赖:**
- Epic 5（关卡流程管理）已完成 - 提供波次系统
- Epic 6（Reanim 动画系统）已完成 - 支持 FinalWave.reanim
- Epic 7（粒子系统）已完成 - 支持 SodRoll.xml
- Epic 8（关卡配置系统）已完成 - 提供关卡配置结构

**并行依赖:**
- 3 个 Story 可并行开发（独立模块）

**后续依赖:**
- 无（Epic 11 是增强功能，不阻塞其他 Epic）

---

## Timeline Estimate (时间估算)

基于项目历史速度和复杂度：

| Story | 预估工作量 | 状态 | 优先级 |
|-------|-----------|------|--------|
| Story 11.1: 修复植物卡片图标渲染 | 已完成 | ✅ Done | 高 |
| Story 11.2: 关卡进度条完善 | 8-12 小时 | Draft | 高 |
| Story 11.3: 最后一波提示动画 | 6-8 小时 | 待创建 | 中高 |
| Story 11.4: 铺草皮粒子特效 | 4-6 小时 | 待创建 | 中 |
| **总计（剩余）** | **18-26 小时** | - | - |

**建议实施顺序:**
1. Story 11.1（植物卡片图标修复）- ✅ 已完成
2. Story 11.2（进度条完善）- 当前优先级最高
3. Story 11.3（最后一波提示）- 原版重要特性
4. Story 11.4（铺草皮特效）- 视觉锦上添花

---

## Notes (备注)

- 本 Epic 专注于关卡 UI 和视觉体验的最后完善
- 所有功能必须忠实还原原版游戏效果，不接受简化实现
- 关卡进度条是玩家获取关卡进度信息的唯一途径，必须准确无误
- 铺草皮特效是关卡开场的仪式感体现，建议保留
- 最后一波提示是原版游戏的经典设计，必须精确实现
