# Epic 19: 关卡 1-5 坚果保龄球 (Level 1-5: Wall-nut Bowling)

## Epic Goal (史诗目标)

实现《植物大战僵尸》第一章第五关"坚果保龄球"关卡，包含**铲子教学阶段**和**保龄球玩法阶段**两个核心阶段。本关作为获取铲子后的第一关，需要实现疯狂戴夫对话系统、强引导教学机制、传送带系统和保龄球物理系统，为玩家提供独特的迷你游戏体验。

---

## Epic Description (史诗描述)

### Existing System Context (现有系统上下文)

**当前相关功能:**
- ✅ 铲子 UI（槽位背景、图标渲染）- `game_scene.go`
- ✅ 关卡系统（`LevelSystem`, `WaveSpawnSystem`）
- ✅ 关卡配置加载（`LevelConfig`, YAML解析）
- ✅ 游戏场景管理（`GameScene`）
- ✅ 植物/僵尸实体系统（ECS 架构）
- ✅ Reanim 动画系统
- ✅ 粒子特效系统
- ✅ 奖励动画系统（`RewardAnimationSystem`）
- ✅ 教学系统基础框架（`TutorialSystem`）

**技术栈:**
- Go 语言 + Ebitengine 引擎
- ECS 架构 (Entity-Component-System)
- YAML 配置驱动
- 资源管理器 (ResourceManager)

**集成点:**
- 关卡配置系统 (`pkg/config/level_config.go`)
- 游戏场景 (`pkg/scenes/game_scene.go`)
- 实体工厂 (`pkg/entities/`)
- 系统层 (`pkg/systems/`)
- 铲子 UI (`pkg/scenes/game_scene_ui.go`)

---

### Enhancement Details (增强功能详情)

本关卡包含两个核心阶段：

#### 阶段 1: 铲子教学 (Shovel Tutorial)

**1.1 预设植物机制**
- 关卡加载后，在草坪上预先生成 3 株豌豆射手
- 位置：(行2, 列6), (行3, 列8), (行4, 列7)
- 模拟"上一关残留"的场景感

**1.2 疯狂戴夫对话系统**
- Dave 角色动画（出现/消失）
- 文本对话框 UI（支持多行文本）
- 文本 key 加载（国际化支持）
- 表情动画支持（MOUTH_SMALL_OH, MOUTH_BIG_SMILE, SHAKE, SCREAM）

**1.3 强引导模式**
- 限制玩家操作（仅允许点击铲子和植物）
- 5秒无操作后显示浮动箭头指向铲子
- 实时检测植物数量，Plant_Count == 0 时触发下一步

**1.4 铲子功能完善**
- 点击铲子后鼠标光标变为铲子图标
- 悬停植物时高亮显示
- 点击植物播放音效 (`plant_dig.wav`) 并移除
- 不返还阳光

#### 阶段 2: 坚果保龄球 (Wall-nut Bowling)

**2.1 传送带系统**
- 位置：屏幕左上方，紧挨铲子卡槽左边
- 容量：最多 10 张卡片
- 视觉效果：ConveyorBelt.png 背景 + ConveyorBelt_backdrop.png 传动动画
- 卡片池：普通坚果 (85%) + 爆炸坚果 (15%)
- 最终波时强制生成 2-3 个爆炸坚果

**2.2 红线限制**
- 在第3列和第4列之间绘制红线 (Wallnut_bowlingstripe.png)
- 玩家只能在红线左侧放置坚果
- 放置到右侧时显示提示 [ADVICE_NOT_PASSED_LINE]

**2.3 保龄球物理系统**

*普通坚果 (Wall-nut):*
- 沿当前行直线滚动（Roll 循环动画）
- 命中僵尸：造成大量伤害（秒杀普通僵尸）
- 弹射逻辑：向相邻行弹开，优先弹向最近的邻行僵尸
- 多次命中后继续弹射，滚出屏幕后销毁

*爆炸坚果 (Explode-o-nut):*
- 红色外观
- 命中僵尸立即爆炸
- 3x3 范围内造成 1800 伤害（等同樱桃炸弹）
- 不弹射，爆炸后直接销毁

**2.4 僵尸配置**
- 僵尸池：普通僵尸、路障僵尸
- 连续高压进攻，无旗帜休息时间
- 进度条显示关卡进度

**2.5 音效**
- `bowling_roll.ogg`: 坚果滚动循环音
- `bowling_strike.ogg`: 击中僵尸撞击声
- `explosion.ogg`: 爆炸坚果爆炸声

---

### Integration Approach (集成方法)

#### 配置文件扩展
```yaml
# data/levels/level-1-5.yaml
id: "1-5"
name: "坚果保龄球"
description: "铲子教学 + 保龄球迷你游戏"

# 场景配置
sceneType: "day"
rowMax: 5
flags: 0  # 无旗帜，使用进度条

# 阶段配置
phases:
  - phase: 1
    type: "shovel_tutorial"
    presetPlants:
      - type: "peashooter"
        row: 2
        col: 6
      - type: "peashooter"
        row: 3
        col: 8
      - type: "peashooter"
        row: 4
        col: 7
    daveDialogue:
      - key: "CRAZY_DAVE_2400"
      - key: "CRAZY_DAVE_2401"
      # ... 其他对话

  - phase: 2
    type: "bowling"
    conveyorBelt:
      enabled: true
      capacity: 10
      cardPool:
        - type: "wallnut_bowling"
          weight: 85
        - type: "explode_o_nut"
          weight: 15
    redLineColumn: 3

# 波次配置
waves:
  # ... 僵尸波次

# 通关奖励
unlockPlants: ["potatomine"]
```

#### 新增组件
```go
// DaveDialogueComponent - 戴夫对话组件
type DaveDialogueComponent struct {
    CurrentLine    int
    DialogueKeys   []string
    IsVisible      bool
    Expression     string  // 表情状态
}

// ConveyorBeltComponent - 传送带组件
type ConveyorBeltComponent struct {
    Cards          []string  // 卡片队列
    Capacity       int
    ScrollOffset   float64
    IsActive       bool
}

// BowlingNutComponent - 保龄球坚果组件
type BowlingNutComponent struct {
    VelocityX      float64
    VelocityY      float64
    IsRolling      bool
    IsExplosive    bool
    BounceCount    int
}

// GuidedTutorialComponent - 强引导教学组件
type GuidedTutorialComponent struct {
    AllowedActions []string  // 允许的操作
    IdleTimer      float64
    ShowArrow      bool
    ArrowTarget    string    // 箭头指向目标
}
```

#### 新增系统
```go
// DaveDialogueSystem - 戴夫对话系统
// 职责: 管理对话流程、显示文本、播放表情动画

// ConveyorBeltSystem - 传送带系统
// 职责: 管理卡片生成、滚动动画、交互逻辑

// BowlingPhysicsSystem - 保龄球物理系统
// 职责: 处理坚果滚动、碰撞检测、弹射逻辑

// GuidedTutorialSystem - 强引导教学系统
// 职责: 限制操作、检测条件、显示引导箭头

// ShovelInteractionSystem - 铲子交互系统
// 职责: 处理铲子选中、植物高亮、移除植物
```

---

## Success Criteria (成功标准)

Epic 成功的判定标准:

1. ✅ **铲子教学阶段**
   - 关卡加载后显示预设的 3 株豌豆射手
   - 戴夫对话按脚本流程正确显示
   - 强引导模式限制玩家只能操作铲子和植物
   - 5秒无操作显示浮动箭头
   - 移除所有植物后正确转场

2. ✅ **传送带系统**
   - 传送带 UI 从上方滑入，位置正确
   - 传动动画流畅（6行交错显示）
   - 卡片按权重正确生成
   - 最终波强制生成爆炸坚果

3. ✅ **保龄球物理**
   - 普通坚果沿直线滚动并正确弹射
   - 爆炸坚果 3x3 范围爆炸
   - 击中反馈（音效、动画）正确

4. ✅ **红线限制**
   - 红线正确渲染
   - 无法在红线右侧放置坚果
   - 显示正确的提示信息

5. ✅ **关卡完整性**
   - 无阳光掉落和阳光槽
   - 进度条正确显示
   - 通关后解锁土豆地雷

---

## Stories (故事列表)

本 Epic 包含以下 Story，按实现顺序排列：

### Story 19.1: 疯狂戴夫对话系统
**目标**: 实现疯狂戴夫角色的对话系统，支持多行文本、表情动画和脚本驱动

**范围**:
- `DaveDialogueComponent` 对话状态组件
- `DaveDialogueSystem` 对话流程管理
- Dave 角色动画（出现/消失/表情）
- 对话框 UI（支持多行文本）
- 文本 key 国际化加载
- 点击/自动推进对话

**验收标准**:
- Dave 角色动画正确播放（出现、消失）
- 对话文本按 key 正确加载显示
- 支持表情动画（MOUTH_SMALL_OH, MOUTH_BIG_SMILE, SHAKE, SCREAM）
- 点击屏幕推进下一条对话
- 对话结束后 Dave 消失

---

### Story 19.2: 铲子交互系统增强
**目标**: 完善铲子的交互功能，包括光标变化、植物高亮和移除功能

**范围**:
- `ShovelInteractionSystem` 铲子交互系统
- 铲子选中状态管理
- 鼠标光标变为铲子图标
- 植物悬停高亮效果
- 植物移除逻辑（不返还阳光）
- 音效播放 (`plant_dig.wav`)

**验收标准**:
- 点击铲子槽位后进入铲子模式
- 鼠标光标显示铲子图标
- 悬停在植物上时植物高亮
- 点击植物播放音效并立即移除
- 移除植物不返还阳光
- 再次点击铲子或右键取消铲子模式

---

### Story 19.3: 强引导教学系统
**目标**: 实现强制性的教学引导机制，限制玩家操作并提供视觉引导

**范围**:
- `GuidedTutorialComponent` 引导状态组件
- `GuidedTutorialSystem` 引导逻辑系统
- 操作限制机制（白名单模式）
- 浮动箭头动画（指向铲子）
- 5秒无操作检测
- 植物数量实时监控

**验收标准**:
- 教学阶段只允许点击铲子和植物
- 其他操作被忽略（不报错）
- 5秒无操作后显示浮动箭头指向铲子
- 操作后箭头消失
- Plant_Count == 0 时触发转场条件

---

### Story 19.4: 预设植物与阶段转场
**目标**: 实现关卡加载时的预设植物机制和阶段之间的转场逻辑

**范围**:
- 关卡配置扩展（presetPlants 字段）
- 预设植物生成逻辑
- 阶段 1 → 阶段 2 转场动画
- 传送带 UI 滑入动画
- 红线绘制

**验收标准**:
- 关卡加载后正确生成 3 株豌豆射手
- 位置与规范一致（行2列6、行3列8、行4列7）
- 移除所有预设植物后触发转场
- 播放戴夫"惊喜"对话
- 传送带从上方滑入
- 红线正确绘制

---

### Story 19.5: 传送带系统
**目标**: 实现传送带 UI 和卡片管理系统

**范围**:
- `ConveyorBeltComponent` 传送带组件
- `ConveyorBeltSystem` 传送带系统
- 传送带 UI 渲染（背景 + 传动动画）
- 卡片队列管理
- 卡片生成算法（权重随机）
- 卡片拖拽交互
- 放置限制（红线左侧）

**验收标准**:
- 传送带显示在正确位置（铲子槽左侧）
- 传动动画流畅（6行交错）
- 最多容纳 10 张卡片
- 卡片按权重生成（普通85%、爆炸15%）
- 拖拽卡片可放置到红线左侧
- 红线右侧放置时显示提示
- 最终波强制生成 2-3 个爆炸坚果

---

### Story 19.6: 保龄球坚果实体与滚动
**目标**: 实现保龄球坚果实体和基础滚动物理

**范围**:
- `BowlingNutComponent` 保龄球组件
- 普通坚果实体工厂
- 爆炸坚果实体工厂
- Roll 循环动画
- 直线滚动物理
- 滚出屏幕销毁

**验收标准**:
- 放置坚果后立即进入滚动状态
- Roll 动画正确循环播放
- 坚果沿放置行直线滚动
- 滚动速度适中（可配置）
- 滚出屏幕右边缘后销毁
- 播放滚动音效 (`bowling_roll.ogg`)

---

### Story 19.7: 保龄球碰撞与弹射系统
**目标**: 实现保龄球的碰撞检测和弹射物理

**范围**:
- `BowlingPhysicsSystem` 保龄球物理系统
- 碰撞检测逻辑
- 伤害计算（秒杀普通僵尸）
- 弹射方向计算
- 优先弹向最近邻行僵尸
- 边缘行反弹处理

**验收标准**:
- 坚果碰撞僵尸造成大量伤害
- 路障僵尸帽子被打掉
- 碰撞后向相邻行弹射
- 优先弹向有僵尸的邻行
- 边缘行（1行/5行）碰墙反弹
- 多次碰撞持续弹射
- 播放撞击音效 (`bowling_strike.ogg`)

---

### Story 19.8: 爆炸坚果机制
**目标**: 实现爆炸坚果的特殊机制

**范围**:
- 爆炸坚果视觉（红色外观）
- 碰撞爆炸逻辑
- 3x3 范围伤害计算（1800伤害）
- 爆炸动画和粒子特效
- 爆炸音效

**验收标准**:
- 爆炸坚果外观为红色
- 碰撞第一个僵尸立即爆炸
- 3x3 范围内所有僵尸受到伤害
- 1800 伤害秒杀范围内僵尸
- 播放爆炸动画和特效
- 播放爆炸音效 (`explosion.ogg`)
- 爆炸后坚果销毁，不弹射

---

### Story 19.9: 僵尸波次配置与进度系统
**目标**: 配置 1-5 关卡的僵尸波次和进度显示

**范围**:
- level-1-5.yaml 波次配置
- 僵尸池配置（普通、路障）
- 连续高压进攻节奏
- 进度条显示（无旗帜）
- 通关条件判定

**验收标准**:
- 僵尸按配置波次生成
- 只出现普通僵尸和路障僵尸
- 进度条正确显示关卡进度
- 无旗帜休息时间
- 消灭所有僵尸后通关

---

### Story 19.10: 关卡完成与奖励
**目标**: 实现关卡完成流程和土豆地雷解锁

**范围**:
- 通关判定逻辑
- 禁用阳光系统
- 奖励动画播放
- 土豆地雷解锁
- 关卡状态保存

**验收标准**:
- 阳光槽隐藏，无阳光掉落
- 消灭最后一只僵尸后触发通关
- 播放关卡完成动画
- 显示土豆地雷解锁提示
- 关卡进度正确保存

---

### Story 19.11: 关卡集成测试与调优
**目标**: 整体测试 1-5 关卡，修复问题并调优

**范围**:
- 完整流程测试（阶段1 → 阶段2）
- 边缘情况处理
- 性能优化
- 难度平衡调整
- 音效/动画时序调整

**验收标准**:
- 铲子教学流程顺畅
- 保龄球玩法正确有趣
- 无明显性能问题（60 FPS）
- 难度适中，有挑战但可通关
- 所有音效正确播放
- 无游戏阻塞性 Bug

---

## Compatibility Requirements (兼容性要求)

- ✅ **向后兼容**: 新增字段为可选，现有关卡配置无需修改
- ✅ **系统隔离**: 特殊关卡逻辑不影响标准关卡
- ✅ **UI一致性**: 传送带、对话框遵循现有设计风格
- ✅ **性能影响**: 保龄球物理计算不降低帧率

---

## Risk Mitigation (风险缓解)

### 主要风险
**保龄球弹射物理复杂，可能影响游戏体验**

**缓解措施**:
- 弹射角度和速度可配置
- 充分测试各种边缘情况
- 提供调试工具可视化弹射轨迹

### 次要风险
**戴夫对话系统是全新系统，开发周期可能较长**

**缓解措施**:
- 先实现基础对话功能，表情动画后续迭代
- 复用现有 UI 组件和动画系统

---

## Rollback Plan (回滚计划)

如果 Epic 实施遇到阻塞问题:

1. **保龄球物理问题**: 简化为直线滚动不弹射
2. **传送带问题**: 临时改为手动放置坚果
3. **对话系统问题**: 使用静态文本替代

回滚步骤:
```bash
# 1. 禁用 1-5 关卡
# 在关卡选择中隐藏 1-5

# 2. 恢复配置文件
git checkout HEAD~1 data/levels/level-1-5.yaml

# 3. 验证其他关卡正常
go run .
```

---

## Definition of Done (完成定义)

Epic 19 视为完成的标准:

### Story 完成状态

- [ ] **Story 19.1**: 疯狂戴夫对话系统
- [ ] **Story 19.2**: 铲子交互系统增强
- [ ] **Story 19.3**: 强引导教学系统
- [ ] **Story 19.4**: 预设植物与阶段转场
- [ ] **Story 19.5**: 传送带系统
- [ ] **Story 19.6**: 保龄球坚果实体与滚动
- [ ] **Story 19.7**: 保龄球碰撞与弹射系统
- [ ] **Story 19.8**: 爆炸坚果机制
- [ ] **Story 19.9**: 僵尸波次配置与进度系统
- [ ] **Story 19.10**: 关卡完成与奖励
- [ ] **Story 19.11**: 关卡集成测试与调优

### 功能完整性标准

- [ ] 铲子教学阶段完整可玩
- [ ] 戴夫对话按脚本流程显示
- [ ] 传送带正确工作
- [ ] 保龄球物理正确
- [ ] 爆炸坚果 3x3 范围爆炸
- [ ] 红线限制正确
- [ ] 无阳光系统
- [ ] 通关后解锁土豆地雷
- [ ] 代码通过测试（覆盖率 80%+）
- [ ] 性能符合要求（60 FPS）

---

## Technical Notes (技术备注)

### 关键设计决策

1. **阶段化设计**
   - 关卡分为两个独立阶段
   - 阶段 1 完成后自动转场到阶段 2
   - 阶段配置存储在 YAML 中

2. **弹射物理算法**
   - 使用简化的 2D 弹射计算
   - 弹射角度固定 45 度
   - 优先级：邻行僵尸 > 边缘反弹 > 直线继续

3. **传送带实现**
   - 使用 ECS 组件管理卡片队列
   - 渲染系统独立处理传动动画
   - 拖拽交互复用现有植物选择逻辑

4. **对话系统扩展性**
   - 支持任意关卡配置对话
   - 表情动画通过 Reanim 实现
   - 文本支持国际化 key

### 参考资料

- 规范文档: `.meta/levels/level1-5.md`
- 现有关卡配置: `data/levels/level-1-4.yaml`
- 铲子 UI: `pkg/scenes/game_scene_ui.go`
- 教学系统: `pkg/systems/tutorial_system.go`

---

## Change Log (变更日志)

| Date | Version | Description | Author |
| :--- | :--- | :--- | :--- |
| 2025-11-30 | 1.0 | Epic 19 初始创建，定义 Level 1-5 坚果保龄球实现范围 | John (PM) |
