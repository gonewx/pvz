# Epic 10: 游戏体验完善 (Game Experience Polish)

## Epic Goal (史诗目标)

完善《植物大战僵尸》核心游戏体验，实现暂停菜单、植物攻击动画、粒子特效和除草车防线系统，提升游戏的完整性和可玩性，确保所有核心机制符合原版游戏标准。

---

## Epic Description (史诗描述)

### Existing System Context (现有系统上下文)

**当前相关功能:**
- ✅ 菜单按钮 UI (`ButtonComponent`, `ButtonSystem`)
- ✅ Reanim 动画系统 (`ReanimSystem`)
- ✅ 粒子特效系统 (`ParticleSystem`, `EmitterComponent`)
- ✅ 关卡流程管理 (`LevelSystem`, `WaveSpawnSystem`)
- ✅ 植物攻击系统 (`BehaviorSystem` - 豌豆射手发射子弹)
- ✅ 碰撞检测系统 (`PhysicsSystem`)
- ✅ ECS 架构成熟，支持灵活扩展

**技术栈:**
- Go 语言 + Ebitengine 引擎
- ECS 架构 (Entity-Component-System)
- Reanim 骨骼动画系统
- XML 配置驱动粒子系统
- 资源管理器 (ResourceManager)

**集成点:**
- 游戏场景 (`pkg/scenes/game_scene.go`)
- 实体工厂 (`pkg/entities/`)
- 系统层 (`pkg/systems/`)
- 组件层 (`pkg/components/`)

---

### Enhancement Details (增强功能详情)

**核心功能模块:**

#### 1. 暂停菜单系统 (Pause Menu System) - ✅ Done
- **暂停机制**: 点击菜单按钮暂停游戏，所有系统停止更新
- **菜单界面**: 显示暂停菜单面板（继续、重新开始、返回主菜单）
- **音效控制**: 暂停时背景音乐降低音量，恢复时还原
- **输入阻断**: 暂停时屏蔽游戏世界的交互（种植、收集阳光等）

#### 2. 植物攻击动画系统 (Plant Attack Animation) - ✅ Done
- **动画切换**: 植物发射子弹时自动切换到攻击动画
- **动画状态机**: 空闲 → 攻击 → 空闲的平滑过渡
- **多植物支持**: 豌豆射手、寒冰射手、双发射手等
- **原版忠实**: 使用原版 Reanim 动画资源

#### 3. 植物种植粒子特效 (Plant Planting Particle Effect) - ✅ Done
- **土粒飞溅**: 种植植物时地面爆发土粒特效
- **原版效果**: 使用原版粒子配置文件（PlantingPool.xml 或类似）
- **位置同步**: 粒子效果精确显示在种植格子位置
- **性能优化**: 粒子批量渲染，不影响帧率

#### 4. 除草车最后防线系统 (Lawnmower Final Defense) - ✅ Done
- **初始化**: 每行左侧台阶放置一辆除草车
- **触发条件**: 僵尸到达屏幕最左侧（x < 100）时触发
- **除草动画**: 除草车从左向右高速移动，清除所在行所有僵尸
- **一次性使用**: 每行除草车只能触发一次，消失后该行失去最后防线
- **失败判定**: 除草车使用后，该行再有僵尸到达左侧则游戏失败

---

### Integration Approach (集成方法)

#### 1. 暂停菜单系统集成

**新增组件:**
```go
// PauseMenuComponent - 暂停菜单状态组件
type PauseMenuComponent struct {
    IsActive      bool
    MenuButtons   []ecs.EntityID  // 菜单按钮实体列表
    BlurOverlay   *ebiten.Image    // 半透明遮罩
}
```

**系统修改:**
```go
// GameScene.Update() 暂停检测
if s.gameState.IsPaused {
    // 只更新 UI 系统，跳过游戏逻辑系统
    s.buttonSystem.Update(deltaTime)
    s.pauseMenuSystem.Update(deltaTime)
    return
}
// 正常更新所有系统...
```

#### 2. 植物攻击动画集成

**BehaviorSystem 修改:**
```go
// handlePeashooterBehavior() 中添加动画切换
if shouldShoot {
    // 切换到攻击动画
    if reanim, ok := ecs.GetComponent[*components.ReanimComponent](em, entity); ok {
        reanim.ChangeAnimation("anim_shooting")
    }
    // 发射子弹...
}
```

**动画状态管理:**
- 攻击动画播放完毕后自动切换回空闲动画
- 使用 Reanim 系统的 `IsAnimationFinished()` 检测

#### 3. 植物种植粒子特效集成

**InputSystem 修改:**
```go
// handlePlantPlacement() 中添加粒子生成
if plantSuccessful {
    // 生成土粒飞溅粒子效果
    entities.NewPlantingParticleEffect(
        s.entityManager,
        s.particleSystem,
        worldX, worldY,  // 种植位置
    )
}
```

**粒子配置:**
- 使用 `assets/effect/particles/PlantingPool.xml` 或类似配置
- 粒子生命周期 0.5-1.0 秒
- 向上飞溅后落下的抛物线运动

#### 4. 除草车系统集成

**新增组件:**
```go
// LawnmowerComponent - 除草车组件
type LawnmowerComponent struct {
    Lane         int      // 所在行（1-5）
    IsTriggered  bool     // 是否已触发
    IsMoving     bool     // 是否正在移动
    Speed        float64  // 移动速度（像素/秒）
}
```

**新增系统:**
```go
// LawnmowerSystem - 除草车系统
// 职责:
// - 检测触发条件（僵尸到达左侧边界）
// - 驱动除草车移动
// - 检测并消灭路径上的僵尸
// - 管理除草车状态（未触发/移动中/已消失）
```

**LevelSystem 集成:**
```go
// checkDefeatCondition() 修改
func (s *LevelSystem) checkDefeatCondition() {
    for each lane {
        if zombieReachedLeft && lawnmowerUsed {
            s.gameState.SetGameOver(false) // 游戏失败
        }
    }
}
```

---

### Stories (用户故事)

本 Epic 包含 **8 个用户故事**，按优先级和实施顺序排列：

#### 原计划 Story (Story 10.1-10.4)

1. **Story 10.1: 暂停菜单系统**
   - 实现完整的暂停/恢复机制
   - 创建暂停菜单 UI（继续、重新开始、退出）
   - 音效和音乐控制
   - 优先级：⭐⭐⭐⭐ 中高

2. **Story 10.2: 除草车最后防线系统**
   - 初始化每行除草车
   - 触发条件检测
   - 除草动画和僵尸清除
   - 失败判定增强
   - 优先级：⭐⭐⭐⭐⭐ 高
   - **状态**: ✅ Done

3. **Story 10.3: 植物攻击动画系统**
   - 植物发射子弹时切换到攻击动画
   - 动画状态机管理
   - 支持多种植物类型
   - 优先级：⭐⭐⭐⭐ 中高

4. **Story 10.4: 植物种植粒子特效**
   - 种植时生成土粒飞溅效果
   - 使用原版粒子配置
   - 位置同步和性能优化
   - 优先级：⭐⭐⭐ 中

#### 扩展 Story (Story 10.5-10.8)

5. **Story 10.5: 植物攻击动画帧事件同步** (Sprint Change Proposal - 2025-10-27)
   - 子弹在攻击动画关键帧发射，而非动画开始时
   - 使用配置关键帧实现零延迟帧同步
   - 子弹起始位置使用实时轨道坐标
   - 优先级：⭐⭐⭐⭐ 中高
   - **状态**: ✅ Done

6. **Story 10.6: 除草车碾压僵尸压扁动画** (Sprint Change Proposal - 2025-11-20)
   - 僵尸被除草车碾压时播放压扁动画 (位移、旋转、缩放)
   - 使用 `LawnMoweredZombie.reanim` 的 locator 轨道变换数据
   - 压扁动画结束后触发粒子效果
   - 优先级：⭐⭐⭐ 中
   - **状态**: ✅ Done

7. **Story 10.7: 植物和僵尸阴影渲染** (Sprint Change Proposal - 2025-11-24)
   - 所有植物和僵尸脚下渲染椭圆形阴影
   - 阴影大小根据实体类型自适应
   - 阴影透明度 60-70%, 渲染在实体下方
   - 优先级：⭐⭐⭐⭐ 中高
   - **状态**: ✅ Done
   - **文档**: `docs/stories/10.7-plant-zombie-shadow-rendering.md`

8. **Story 10.8: 植物卡片交互反馈增强** (Sprint Change Proposal - 2025-11-24, Updated 2025-11-25)
   - **阳光不足点击反馈**:
     - 点击阳光不足卡片时，阳光计数器数字闪烁 (红/黑)
     - 闪烁周期 0.3秒, 持续 1.0秒 (约 3 次)
     - 播放无效操作音效
   - **植物卡片 Tooltip 系统** (新增):
     - 鼠标悬停卡片显示提示框 (黑边浅黄色背景)
     - 提示框显示植物名(黑色居中) + 状态提示(红色小号)
     - 状态提示在第一行: "重新装填中..." (冷却) / "没有足够的阳光" (阳光不足)
     - 植物名在第二行
     - 不可点击状态下鼠标光标不变为手形
   - 优先级：⭐⭐⭐⭐ 中高
   - **状态**: ✅ Done
   - **文档**: `docs/stories/10.8-sun-shortage-feedback.md`

#### 音效系统模块 (Story 10.9-10.12) - NEW

9. **Story 10.9: 音效系统集成** (Sprint Change Proposal - 2025-12-02)
   - 核心游戏音效集成（胜利/失败、波次警告、割草机、僵尸音效）
   - 音效路径统一修正（`assets/audio/Sound/` → `assets/sounds/`）
   - 植物相关音效（土豆地雷、攻击植物、冰冻效果）
   - 护甲击中音效、环境特效音效
   - 优先级：⭐⭐⭐⭐⭐ 高
   - **状态**: 📝 Draft
   - **文档**: `docs/stories/10.9.sound-effect-integration.story.md`
   - **音效清单**: `docs/sound-effects-inventory.md`

10. **Story 10.10: 戴夫语音系统** (待创建)
    - 疯狂戴夫对话语音集成
    - 短语、长语、超长语音效支持
    - 与对话系统集成
    - 优先级：⭐⭐⭐⭐ 中高
    - **状态**: 📝 Planned

11. **Story 10.11: 完整植物音效系统** (待创建)
    - 窝瓜、火爆辣椒、玉米投手等植物音效
    - 黄油定身效果音效
    - 优先级：⭐⭐⭐ 中
    - **状态**: 📝 Planned

12. **Story 10.12: 特殊僵尸音效系统** (待创建)
    - 报纸僵尸、撑杆跳僵尸、小丑僵尸、舞王僵尸等
    - 巨人僵尸脚步/砸地音效
    - 优先级：⭐⭐⭐ 中
    - **状态**: 📝 Planned

---

### Compatibility Requirements (兼容性要求)

- ✅ **ECS 架构一致性**: 所有新功能遵循 ECS 模式，零耦合原则
- ✅ **泛型 API 使用**: 使用 Epic 9 的泛型 ECS API，类型安全
- ✅ **资源管理**: 通过 ResourceManager 统一加载资源
- ✅ **配置驱动**: 尽可能使用配置文件（如粒子 XML）
- ✅ **性能考虑**: 不影响现有系统的帧率和流畅度
- ✅ **原版忠实**: 所有功能还原原版 PC 游戏效果

---

### Risk Mitigation (风险缓解)

**主要风险:**
1. **暂停机制影响游戏状态** - 某些系统未正确暂停导致状态不一致
2. **动画切换冲突** - 攻击动画与其他动画（如死亡动画）冲突
3. **粒子系统性能** - 大量粒子同时生成影响帧率
4. **除草车触发时机** - 僵尸已进入房屋但除草车未触发

**缓解措施:**
1. **暂停标志管理** - 在 GameState 中使用统一的 `IsPaused` 标志
2. **动画优先级** - 建立动画状态机，死亡动画优先级最高
3. **粒子限制** - 限制同屏粒子数量，使用对象池
4. **精确边界检测** - 除草车触发点在房屋入口前 50 像素

**回滚计划:**
- 每个 Story 独立实现，可单独禁用
- 使用功能开关（如配置文件中的 `enablePauseMenu: true`）
- 不影响核心游戏流程，回滚不破坏已有功能

---

### Definition of Done (完成定义)

- ✅ 所有 4 个用户故事完成，AC 全部满足
- ✅ 暂停菜单功能完整，UI 美观，交互流畅
- ✅ 植物攻击动画自然切换，无卡顿或闪烁
- ✅ 种植粒子特效符合原版表现，性能良好
- ✅ 除草车系统完整，触发准确，动画流畅
- ✅ 现有功能无回归，通过 QA 测试
- ✅ 代码符合编码规范，通过 linter 检查
- ✅ 所有新功能有对应的单元测试（覆盖率 > 80%）
- ✅ 文档更新（AGENTS.md, CLAUDE.md）

---

## Success Criteria (成功标准)

本 Epic 成功完成的标准：

1. **功能完整性** - 所有 8 个 Story 实现并正常工作
   - ✅ Story 10.1-10.4: 原计划功能模块
   - ✅ Story 10.5-10.6: 已完成的扩展 Story (Done)
   - 📝 Story 10.7-10.8: 待实现的视觉增强 Story (Pending)
2. **原版忠实度** - 所有效果还原原版 PC 游戏表现
3. **性能稳定** - 帧率稳定在 60 FPS，无明显卡顿
4. **用户体验** - 暂停菜单、动画切换、特效表现、阴影效果、交互反馈自然流畅
5. **代码质量** - 遵循 ECS 架构，代码可读性和可维护性高
6. **测试覆盖** - 所有关键功能有单元测试和集成测试
7. **无回归** - 现有功能（种植、战斗、关卡流程）无破坏

---

## Dependencies (依赖关系)

**前置依赖:**
- Epic 1-9 已完成（ECS 框架、Reanim 系统、粒子系统）
- 菜单按钮 UI 已创建（Story 8.5 QA 改进）

**并行依赖:**
- Story 10.1 和 10.2 可并行开发（独立模块）
- Story 10.3 和 10.4 可并行开发（独立模块）

**后续依赖:**
- Epic 11（如有）可能依赖暂停菜单系统
- 第一章关卡完整体验依赖除草车系统

---

## Timeline Estimate (时间估算)

基于项目历史速度和复杂度：

| Story | 预估工作量 | 优先级 | 状态 |
|-------|-----------|--------|------|
| Story 10.1: 暂停菜单系统 | 8-12 小时 | 中高 | ✅ Done |
| Story 10.2: 除草车系统 | 12-16 小时 | 高 | ✅ Done |
| Story 10.3: 植物攻击动画 | 6-8 小时 | 中高 | ✅ Done |
| Story 10.4: 种植粒子特效 | 4-6 小时 | 中 | ✅ Done |
| Story 10.5: 攻击帧事件同步 | 6-9 小时 | 中高 | ✅ Done |
| Story 10.6: 除草车压扁动画 | 9-15 小时 | 中 | ✅ Done |
| Story 10.7: 阴影渲染 | 6-8 小时 | 中高 | ✅ Done |
| Story 10.8: 卡片交互反馈 | 7-10 小时 | 中高 | ✅ Done |
| Story 10.9: 音效系统集成 | 12-18 小时 | 高 | 📝 Draft |
| Story 10.10: 戴夫语音系统 | 4-6 小时 | 中高 | 📝 Planned |
| Story 10.11: 完整植物音效 | 6-10 小时 | 中 | 📝 Planned |
| Story 10.12: 特殊僵尸音效 | 8-12 小时 | 中 | 📝 Planned |
| **总计** | **88-130 小时** | - | 8/12 Done |

**已完成工作量**: 58-84 小时 (Story 10.1-10.8)
**剩余工作量**: 30-46 小时 (Story 10.9-10.12)

**建议实施顺序 (剩余 Story):**
1. Story 10.9（音效系统集成）- 核心游戏音效，高优先级
2. Story 10.10（戴夫语音系统）- 与对话系统集成，中高优先级
3. Story 10.11（完整植物音效）- 植物战斗体验
4. Story 10.12（特殊僵尸音效）- 僵尸战斗体验

---

## Notes (备注)

- 本 Epic 是对核心游戏体验的最后完善，实现后游戏基本达到 MVP 标准
- 所有功能必须忠实还原原版游戏效果，不接受简化实现
- 性能优化是关键，确保在低端设备上也能流畅运行
- 除草车系统是原版游戏的标志性机制，必须精确实现









