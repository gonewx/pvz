# **4. Core Systems (核心系统)**

游戏的行为逻辑由一系列专门的"系统"驱动。每个系统在一个游戏循环（Update tick）中运行，查询包含特定组件组合的实体，并对这些组件的数据进行操作。

> **最后更新**: 2025-12-09
>
> **重要提示**: 项目使用 **Go 泛型 ECS API**，详见 `pkg/ecs/entity_manager.go`

---

## 系统分类概览

| 分类 | 数量 | 说明 |
|------|------|------|
| 核心管理器 | 3 | 场景、实体、输入管理 |
| Reanim 动画系统 | 5 | 骨骼动画更新和渲染 |
| 粒子系统 | 2 | 粒子生成、更新和渲染 |
| 游戏逻辑系统 | 15+ | 关卡、阳光、波次等核心逻辑 |
| 行为子系统 | 4 | 植物、僵尸、投射物行为处理 |
| 僵尸/难度系统 | 5 | 难度引擎、生成约束、行分配 |
| 警告/提示系统 | 3 | 旗帜波、最终波、闪烁效果 |
| 教程/动画系统 | 6 | 教程、开场动画、戴夫对话等 |
| 奖励系统 | 2 | 奖励动画和面板渲染 |
| UI 系统 | 15+ | 按钮、对话框、选卡界面等 |

---

## 一、核心管理器

### **`SceneManager`**
*   **Purpose:** 管理游戏的宏观状态，如主菜单、游戏场景、暂停菜单等。负责切换当前活动的场景，确保只有一个场景的 `Update` 和 `Draw` 方法在被调用。
*   **文件:** `pkg/game/scene_manager.go`
*   **Key Interfaces:** `Update(deltaTime float64)`, `Draw(screen *ebiten.Image)`, `SwitchTo(sceneName string)`
*   **Dependencies:** 无。它是最高层的控制器。

---

### **`EntityManager`**
*   **Purpose:** 负责所有实体（Entities）和组件（Components）的创建、销毁和存储。它是 ECS 模式的核心数据库，提供泛型查询功能。
*   **文件:** `pkg/ecs/entity_manager.go`
*   **Key Interfaces:**
    ```go
    // 泛型 API（推荐）
    ecs.GetComponent[*components.PlantComponent](em, entity)
    ecs.GetEntitiesWith2[*components.PlantComponent, *components.PositionComponent](em)
    ecs.AddComponent(em, entity, component)
    ecs.HasComponent[*components.PlantComponent](em, entity)
    ```
*   **Dependencies:** 无。它是游戏世界状态的核心。

---

### **`InputSystem`**
*   **Purpose:** 捕获并处理所有原始玩家输入（鼠标点击、按键）。将原始输入转换为游戏内的具体"意图"或"事件"。
*   **文件:** `pkg/systems/input_system.go`
*   **Key Interfaces:** `Update(deltaTime float64)`
*   **Dependencies:** `EntityManager`, Event Bus

---

## 二、Reanim 动画系统 ⭐

### **`ReanimSystem`**
*   **Purpose:** 核心骨骼动画系统，替代了原有的 AnimationSystem。管理 Reanim 动画的播放、帧推进和状态控制。
*   **文件:** `pkg/systems/reanim_system.go`
*   **Key Interfaces:** `Update(dt float64, em *ecs.EntityManager)`
*   **Dependencies:** `EntityManager`, `ReanimComponent`

---

### **`ReanimUpdate`**
*   **Purpose:** 动画更新逻辑，处理帧推进、循环、FPS 控制和帧缓存（空帧继承）。
*   **文件:** `pkg/systems/reanim_update.go`
*   **说明:** 实现原版 PvZ 的帧缓存机制，处理动画中的空帧继承。

---

### **`ReanimHelpers`**
*   **Purpose:** 动画辅助函数，提供轨道查询、帧数据获取等工具方法。
*   **文件:** `pkg/systems/reanim_helpers.go`

---

### **`ReanimRender` / `RenderReanim`**
*   **Purpose:** Reanim 动画渲染，支持部件变换（位置、缩放、倾斜、旋转）和组合动画渲染。
*   **文件:** `pkg/systems/reanim_render.go`, `pkg/systems/render_reanim.go`

---

### **`ReanimAPI`**
*   **Purpose:** 动画 API 层，提供高层动画控制接口（播放、暂停、切换动画等）。
*   **文件:** `pkg/systems/reanim_api.go`

---

## 三、粒子系统

### **`ParticleSystem`**
*   **Purpose:** 粒子效果系统，支持原版 PvZ 的所有视觉特效。处理粒子生成、更新、物理模拟和渲染。
*   **文件:** `pkg/systems/particle_system.go`
*   **Key Interfaces:** `Update(dt float64)`, `Draw(screen *ebiten.Image, cameraX float64)`
*   **特性:**
    - 支持 40+ 种粒子属性解析
    - 支持加法混合模式（Additive Blending）
    - 使用 DrawTriangles 批量渲染
    - 动画插值系统（Alpha、Scale、Spin 等）
    - 力场系统（重力、摩擦、加速度）

---

### **`ParticleEmitter`**
*   **Purpose:** 粒子发射器逻辑，控制粒子的生成时机、速度、角度和区域。
*   **文件:** `pkg/systems/particle_emitter.go`

---

## 四、游戏逻辑系统

### **`LevelSystem`**
*   **Purpose:** 关卡管理系统，加载关卡配置、管理游戏进程、检测胜利/失败条件。
*   **文件:** `pkg/systems/level_system.go`

---

### **`LevelPhaseSystem`**
*   **Purpose:** 关卡阶段系统，追踪关卡当前阶段（准备、战斗、胜利、失败）。
*   **文件:** `pkg/systems/level_phase_system.go`

---

### **`WaveSpawnSystem`**
*   **Purpose:** 波次生成系统，根据关卡配置生成僵尸波次。
*   **文件:** `pkg/systems/wave_spawn_system.go`

---

### **`WaveTimingSystem`**
*   **Purpose:** 波次计时系统，控制常规波次、旗帜波和最终波的时机。
*   **文件:** `pkg/systems/wave_timing_system.go`

---

### **`SunSpawnSystem`**
*   **Purpose:** 阳光生成系统，控制天空阳光的周��性生成。
*   **文件:** `pkg/systems/sun_spawn_system.go`

---

### **`SunMovementSystem`**
*   **Purpose:** 阳光移动系统，处理阳光下落、收集飞行动画。
*   **文件:** `pkg/systems/sun_movement_system.go`

---

### **`SunCollectionSystem`**
*   **Purpose:** 阳光收集系统，处理阳光点击收集和计数更新。
*   **文件:** `pkg/systems/sun_collection_system.go`

---

### **`LawnmowerSystem`**
*   **Purpose:** 除草车系统，管理除草车触发、移动和僵尸消灭。
*   **文件:** `pkg/systems/lawnmower_system.go`
*   **特性:**
    - 僵尸到达左侧触发
    - 清除所在行所有僵尸
    - 一次性使用

---

### **`BowlingNutSystem`**
*   **Purpose:** 保龄球坚果系统，用于 1-5 关卡的保龄球玩法。
*   **文件:** `pkg/systems/bowling_nut_system.go`
*   **特性:**
    - 滚动物理模拟
    - 反弹机制
    - 爆炸坚果支持

---

### **`ConveyorBeltSystem`**
*   **Purpose:** 传送带系统，用于传送带关卡（如 1-10）。
*   **文件:** `pkg/systems/conveyor_belt_system.go`

---

### **`ShovelInteractionSystem`**
*   **Purpose:** 铲子交互系统，处理植物移除逻辑。
*   **文件:** `pkg/systems/shovel_interaction_system.go`

---

### **`SoddingSystem`**
*   **Purpose:** 铺草皮系统，处��草皮特效和泥土粒子。
*   **文件:** `pkg/systems/sodding_system.go`

---

### **`LawnGridSystem`**
*   **Purpose:** 草坪网格系统，管理网格状态和坐标转换。
*   **文件:** `pkg/systems/lawn_grid_system.go`

---

### **`LifetimeSystem`**
*   **Purpose:** 生命周期系统，管理实体的自动销毁。
*   **文件:** `pkg/systems/lifetime_system.go`

---

### **`PhysicsSystem`**
*   **Purpose:** 物理系统，处理移动和碰撞检测。
*   **文件:** `pkg/systems/physics_system.go`
*   **功能:**
    - 根据 VelocityComponent 更新 PositionComponent
    - 检测子弹与僵尸碰撞
    - 发布 CollisionEvent 事件

---

## 五、行为子系统

### **`BehaviorSystem`**
*   **Purpose:** 行为系统核心，根据实体的 BehaviorComponent 分发到具体的行为处理器。
*   **文件:** `pkg/systems/behavior/behavior_system.go`
*   **Key Interfaces:** `Update(deltaTime float64)`

---

### **`PlantBehaviorHandler`**
*   **Purpose:** 植物行为处理器，处理向日葵生产、豌豆射手攻击、坚果防御等。
*   **文件:** `pkg/systems/behavior/plant_behavior_handler.go`

---

### **`ZombieBehaviorHandler`**
*   **Purpose:** 僵尸行为处理器，处理僵尸移动、啃食、死亡等状态。
*   **文件:** `pkg/systems/behavior/zombie_behavior_handler.go`

---

### **`ProjectileBehaviorHandler`**
*   **Purpose:** 投射物行为处理器，处理豌豆子弹移动、命中检测等。
*   **文件:** `pkg/systems/behavior/projectile_behavior_handler.go`

---

## 六、僵尸/难度系统

### **`DifficultyEngine`**
*   **Purpose:** 难度引擎，计算轮数、级别容量上限，动态调整游戏难度。
*   **文件:** `pkg/systems/difficulty_engine.go`

---

### **`SpawnConstraintSystem`**
*   **Purpose:** 生成约束系统，检查僵尸生成限制条件。
*   **文件:** `pkg/systems/spawn_constraint_system.go`

---

### **`LaneAllocator`**
*   **Purpose:** 行分配���法，实现平滑权重行分配，确保僵尸分布自然。
*   **文件:** `pkg/systems/lane_allocator.go`

---

### **`ZombieGroanSystem`**
*   **Purpose:** 僵尸呻吟系统，管理僵尸音效播放。
*   **文件:** `pkg/systems/zombie_groan_system.go`

---

### **`ZombieLaneTransitionSystem`**
*   **Purpose:** 僵尸行切换系统，处理僵尸跨行移动（如撑杆僵尸跳跃后）。
*   **文件:** `pkg/systems/zombie_lane_transition_system.go`

---

## 七、警告/提示系统

### **`FlagWaveWarningSystem`**
*   **Purpose:** 旗帜波警告系统，在旗帜波到来前显示红字警告。
*   **文件:** `pkg/systems/flag_wave_warning_system.go`

---

### **`FinalWaveWarningSystem`**
*   **Purpose:** 最终波警告系统，显示 "A huge wave of zombies is approaching" 提示。
*   **文件:** `pkg/systems/final_wave_warning_system.go`

---

### **`FlashEffectSystem`**
*   **Purpose:** 闪烁效果系统，用于阳光不足提示等闪烁效果。
*   **文件:** `pkg/systems/flash_effect_system.go`

---

## 八、教程/动画系统

### **`TutorialSystem`**
*   **Purpose:** 教程系统，管理新手引导流程。
*   **文件:** `pkg/systems/tutorial_system.go`

---

### **`GuidedTutorialSystem`**
*   **Purpose:** 引导教程系统，用于 1-1 等教学关卡的强制引导机制。
*   **文件:** `pkg/systems/guided_tutorial_system.go`

---

### **`OpeningAnimationSystem`**
*   **Purpose:** 开场动画系统，控制关卡开始时的镜头移动和僵尸预告。
*   **文件:** `pkg/systems/opening_animation_system.go`

---

### **`ReadySetPlantSystem`**
*   **Purpose:** Ready-Set-Plant 系统，播放关卡开始动画。
*   **文件:** `pkg/systems/readysetplant_system.go`

---

### **`DaveDialogueSystem`**
*   **Purpose:** 疯狂戴夫对话系统，管理戴夫对话显示和交互。
*   **文件:** `pkg/systems/dave_dialogue_system.go`

---

### **`ZombiesWonPhaseSystem`**
*   **Purpose:** 游戏失败流程系统，控制失败动画和游戏结束逻辑。
*   **文件:** `pkg/systems/zombies_won_phase_system.go`

---

### **`CameraSystem`**
*   **Purpose:** 摄像机系统，用于开场动画的镜头控制和平移。
*   **文件:** `pkg/systems/camera_system.go`

---

## 九、奖励系统

### **`RewardAnimationSystem`**
*   **Purpose:** 奖励动画系统，控制关卡完成时的奖励动画播放。
*   **文件:** `pkg/systems/reward_animation_system.go`

---

### **`RewardPanelRenderSystem`**
*   **Purpose:** 奖励面板渲染系统，显示新植物介绍面板。
*   **文件:** `pkg/systems/reward_panel_render_system.go`

---

## 十、UI 系统

### **`RenderSystem`**
*   **Purpose:** 主渲染系统，迭代所有可渲染实体并绘制到屏幕。
*   **文件:** `pkg/systems/render_system.go`
*   **Key Interfaces:** `Draw(screen *ebiten.Image)`

---

### **`PlantCardSystem`**
*   **Purpose:** 植物卡片逻辑系统，处理卡片状态、冷却和阳光检测。
*   **文件:** `pkg/systems/plant_card_system.go`

---

### **`PlantCardRenderSystem`**
*   **Purpose:** 植物卡片渲染系统，绘制卡片 UI 和冷却遮罩。
*   **文件:** `pkg/systems/plant_card_render_system.go`

---

### **`PlantPreviewSystem`**
*   **Purpose:** 植物预览逻辑系统，在种植模式下显示植物预览。
*   **文件:** `pkg/systems/plant_preview_system.go`

---

### **`PlantPreviewRenderSystem`**
*   **Purpose:** 植物预览渲染系统，绘制跟随鼠标的植物预览。
*   **文件:** `pkg/systems/plant_preview_render_system.go`

---

### **`PlantSelectionSystem`**
*   **Purpose:** 选卡界面系统，管理关卡开始前的植物选择。
*   **文件:** `pkg/systems/plant_selection_system.go`

---

### **`ButtonSystem`**
*   **Purpose:** 按钮逻辑系统，处理按钮状态和点击事件。
*   **文件:** `pkg/systems/button_system.go`

---

### **`ButtonRenderSystem`**
*   **Purpose:** 按钮渲染系统，绘制按钮不同状态的外观。
*   **文件:** `pkg/systems/button_render_system.go`

---

### **`SliderSystem`**
*   **Purpose:** 滑块系统，用于音量控制等。
*   **文件:** `pkg/systems/slider_system.go`

---

### **`CheckboxSystem`**
*   **Purpose:** 复选框系统。
*   **文件:** `pkg/systems/checkbox_system.go`

---

### **`TextInputSystem`**
*   **Purpose:** 文本输入逻辑系统。
*   **文件:** `pkg/systems/text_input_system.go`

---

### **`TextInputRenderSystem`**
*   **Purpose:** 文本输入渲染系统。
*   **文件:** `pkg/systems/text_input_render_system.go`

---

### **`VirtualKeyboardSystem`**
*   **Purpose:** 虚拟键盘逻辑系统，用于移动平台文本输入。
*   **文件:** `pkg/systems/virtual_keyboard_system.go`

---

### **`VirtualKeyboardRenderSystem`**
*   **Purpose:** 虚拟键盘渲染系统。
*   **文件:** `pkg/systems/virtual_keyboard_render_system.go`

---

### **`DialogInputSystem`**
*   **Purpose:** 对话框输入系统，处理对话框按钮交互。
*   **文件:** `pkg/systems/dialog_input_system.go`

---

### **`DialogRenderSystem`**
*   **Purpose:** 对话框渲染系统。
*   **文件:** `pkg/systems/dialog_render_system.go`

---

### **`PauseMenuRenderSystem`**
*   **Purpose:** 暂停菜单渲染系统。
*   **文件:** `pkg/systems/pause_menu_render_system.go`

---

### **`LevelProgressBarRenderSystem`**
*   **Purpose:** 关卡进度条渲染系统。
*   **文件:** `pkg/systems/level_progress_bar_render_system.go`

---

## 十一、系统使用示例

### 系统更新顺序

```go
func (g *GameScene) Update() error {
    dt := 1.0 / 60.0  // 固定时间步长

    // 1. 输入处理
    g.inputSystem.Update(dt, g.em)

    // 2. 游戏逻辑
    g.behaviorSystem.Update(dt, g.em)
    g.physicsSystem.Update(dt, g.em)
    g.waveTimingSystem.Update(dt, g.em)
    g.waveSpawnSystem.Update(dt, g.em)

    // 3. 动画和粒子
    g.reanimSystem.Update(dt, g.em)
    g.particleSystem.Update(dt)

    // 4. 生命周期
    g.lifetimeSystem.Update(dt, g.em)

    return nil
}
```

### 使用泛型 API 查询实体

```go
// 查询所有植物实体
plants := ecs.GetEntitiesWith2[
    *components.PlantComponent,
    *components.PositionComponent,
](em)

for _, entity := range plants {
    plant, _ := ecs.GetComponent[*components.PlantComponent](em, entity)
    pos, _ := ecs.GetComponent[*components.PositionComponent](em, entity)

    // 处理植物逻辑...
}
```

---

## 参考文档

- **ECS 框架:** `pkg/ecs/entity_manager.go`
- **泛型 API 指南:** `docs/architecture/ecs-generics-migration-guide.md`
- **Reanim 动画系统:** `docs/reanim/reanim-format-guide.md`
- **系统源代码:** `pkg/systems/`
