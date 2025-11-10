# Epic 12: 主菜单系统 (Main Menu System)

## Epic Goal (史诗目标)

实现完整的《植物大战僵尸》主菜单界面，包括石碑菜单、用户管理、功能按钮、对话框系统和动画效果，为玩家提供直观的游戏导航和沉浸式的视觉体验，完全还原原版游戏的主菜单交互逻辑。

---

## Epic Description (史诗描述)

### Existing System Context (现有系统上下文)

**当前相关功能:**
- ✅ 基础主菜单场景 (`MainMenuScene` - 已实现)
- ✅ 背景图片加载和渲染
- ✅ 背景音乐播放 (titlescreen.ogg)
- ✅ 基础按钮交互（冒险模式、退出游戏）
- ✅ 场景切换系统 (`SceneManager`)
- ✅ 资源管理系统 (`ResourceManager`)
- ✅ 存档系统 (`SaveManager` - Epic 8)
- ✅ Reanim 动画系统 (`ReanimSystem` - Epic 6)

**技术栈:**
- Go 语言 + Ebitengine 引擎
- ECS 架构 (Entity-Component-System)
- YAML 配置驱动资源管理
- Reanim 骨骼动画系统
- 事件驱动的 UI 交互

**集��点:**
- 主菜单场景 (`pkg/scenes/main_menu_scene.go`)
- UI 组件 (`pkg/components/ui_component.go`, `button_component.go`)
- 对话框系统 (`pkg/entities/dialog_factory.go` - 需创建)
- 渲染系统 (`pkg/systems/render_system.go`)
- 实体工厂 (`pkg/entities/`)

---

### Enhancement Details (增强功能详情)

#### 当前问题与缺失功能

**❌ 缺失的核心功能：**

1. **石碑菜单系统不完整**
   - 当前只有冒险模式按钮
   - 缺少：玩玩小游戏、解谜模式、生存模式
   - 缺少：关卡进度显示（如"LEVEL 1-4"）
   - 按钮布局不符合原版规格

2. **用户管理系统缺失**
   - 无用户名显示
   - 无存档管理功能（新建/重命名/删除）
   - 无左上角木牌 UI

3. **底部功能栏不符合规格**
   - 当前退出按钮位置和样式不正确
   - 缺少：选项/帮助/退出 三个花瓶样式按钮
   - 缺少按钮交互反馈（悬停态切换）

4. **对话框系统缺失**
   - 无未解锁模式提示对话框
   - 无设置对话框
   - 无帮助对话框
   - 缺少九宫格拉伸渲染支持

5. **动画系统未集成**
   - 缺少 SelectorScreen.reanim 动画播放
   - 缺少僵尸手掌升起动画
   - 缺少云朵飘动效果

6. **解锁功能区缺失**
   - 缺少图鉴入口（2-4 后解锁）
   - 缺少商店入口（3-4 后解锁）
   - 缺少禅境花园入口（5-4 后解锁）
   - 缺少成就入口（年度版特色）

---

### Enhancement Modules (功能模块详情)

#### 1. 石碑菜单系统 (Tombstone Menu System)

**核心功能：**
- 4 个主模式按钮（冒险/小游戏/解谜/生存）
- 关卡进度显示（从存档读取，如"LEVEL 1-4"）
- 按钮状态管理（正常/悬停/点击/锁定）
- 解锁状态判断（基于游戏进度）

**资源需求：**
- `SelectorScreen.reanim` - 主菜单动画
- `SelectorScreen_Adventure_Button.png` - 冒险模式按钮
- `SelectorScreen_Adventure_Highlight.png` - 冒险模式高亮
- `SelectorScreen_StartAdventure_Text.png` - 开始冒险文字
- `SelectorScreen_LevelNumbers.png` - 关卡数字精灵图（0-9）
- 类似资源用于其他模式按钮

**实现要点：**
- 按钮布局符合原版截图
- 鼠标悬停高亮效果
- 点击未解锁模式弹出提示对话框
- 关卡进度动态更新

#### 2. 用户管理系统 (User Management System)

**核心功能：**
- 左上角木牌 UI 显示
- 用户名显示（从存档读取）
- 存档切换对话框
- 存档管理（新建/重命名/删除）

**资源需求：**
- 木牌背景图片
- 对话框资源（dialog_*.png）
- 文本输入框样式

**实现要点：**
- 点击木牌打开存档管理对话框
- 显示所有存档列表
- 支持创建新存档
- 支持切换存档（重新加载游戏）

#### 3. 底部功能栏 (Bottom Function Bar)

**核心功能：**
- 3 个花瓶样式按钮（选项/帮助/退出）
- 按钮交互反馈（正常态/悬停态）
- 点击音效

**资源需求：**
- `SelectorScreen_Options1.png` / `SelectorScreen_Options2.png`
- `SelectorScreen_Help1.png` / `SelectorScreen_Help2.png`
- `SelectorScreen_Quit1.png` / `SelectorScreen_Quit2.png`
- `SOUND_BUTTONCLICK` - 按钮点击音效

**实现要点：**
- 按钮位置在右下角
- 鼠标悬停切换图片
- 点击触发对应功能

#### 4. 对话框系统 (Dialog System)

**核心功能：**
- 通用对话框组件（九宫格拉伸）
- 未解锁提示对话框
- 设置对话框
- 帮助对话框
- 存档管理对话框

**资源需求：**
- `dialog_*.png` - 对话框九宫格资源（13 张图片）
- `dialog_header.png` - 对话框头部（骷髅装饰）

**实现要点：**
- 九宫格拉伸算法实现
- 居中显示，带半透明遮罩
- 点击"确定"或外部区域关闭
- ESC 键关闭

#### 5. 动画系统集成 (Animation System Integration)

**核心功能：**
- SelectorScreen.reanim 动画播放
- 僵尸手掌升起动画
- 云朵飘动效果
- 按钮高亮动画

**资源需求：**
- `SelectorScreen.reanim` - 主菜单动画
- `Zombie_hand.reanim` - 僵尸手掌动画

**实现要点：**
- 点击菜单时播放僵尸手掌动画
- 云朵缓慢横向移动
- 与现有 ReanimSystem 集成

#### 6. 解锁功能入口 (Unlockable Features)

**核心功能：**
- 图鉴入口（2-4 后解锁）
- 商店入口（3-4 后解锁）
- 禅境花园入口（5-4 后解锁）
- 成就入口（默认解锁）

**资源需求：**
- 各功能入口的图标和按钮
- 灰色/高亮状态资源

**实现要点：**
- 根据游戏进度动态显示/隐藏
- 未解锁时显示灰色或不显示
- 点击跳转到对应场景（或未实现提示）

---

### Integration Approach (集成方法)

#### 1. 石碑菜单系统集成

**新增组件：**
```go
// MenuButtonComponent - 主菜单按钮组件
type MenuButtonComponent struct {
    ButtonType      MenuButtonType  // 按钮类型（冒险/小游戏/解谜/生存）
    IsUnlocked      bool            // 是否已解锁
    LevelProgress   string          // 关卡进度（如"1-4"）
    NormalImage     *ebiten.Image   // 正常状态图片
    HighlightImage  *ebiten.Image   // 高亮状态图片
    State           UIState         // UI 状态（正常/悬停/点击）
}
```

**MainMenuScene 修改：**
```go
// MainMenuScene 扩展
type MainMenuScene struct {
    // ... 现有字段
    menuButtons     []ecs.EntityID  // 菜单按钮实体列表
    currentLevel    string          // 当前关卡（从存档读取）
}

// initMenuButtons - 初始化4个主模式按钮
func (m *MainMenuScene) initMenuButtons() {
    // 加载存档，获取当前进度
    saveManager := game.GetGameState().GetSaveManager()
    m.currentLevel = saveManager.GetHighestLevel()

    // 创建4个按钮实体
    adventureBtn := entities.NewMenuButton(em, rm, MenuTypeAdventure, true, m.currentLevel)
    miniGameBtn := entities.NewMenuButton(em, rm, MenuTypeMiniGame, isUnlocked("3-2"), "")
    puzzleBtn := entities.NewMenuButton(em, rm, MenuTypePuzzle, isCompleted(), "")
    survivalBtn := entities.NewMenuButton(em, rm, MenuTypeSurvival, isCompleted(), "")

    m.menuButtons = []ecs.EntityID{adventureBtn, miniGameBtn, puzzleBtn, survivalBtn}
}
```

#### 2. 对话框系统集成

**新增实体工厂：**
```go
// pkg/entities/dialog_factory.go

// NewDialogEntity - 创建通用对话框实体
func NewDialogEntity(
    em *ecs.EntityManager,
    rm *game.ResourceManager,
    title string,
    message string,
    buttons []string,
) ecs.EntityID {
    entity := em.CreateEntity()

    // 加载对话框九宫格资源
    dialogParts := loadDialogParts(rm)

    // 添加对话框组件
    dialogComp := &components.DialogComponent{
        Title:       title,
        Message:     message,
        Buttons:     buttons,
        Parts:       dialogParts,
        IsVisible:   true,
    }
    ecs.AddComponent(em, entity, dialogComp)

    // 添加位置组件（居中）
    centerX := (WindowWidth - dialogWidth) / 2
    centerY := (WindowHeight - dialogHeight) / 2
    ecs.AddComponent(em, entity, &components.PositionComponent{X: centerX, Y: centerY})

    return entity
}

// renderNinePatch - 九宫格拉伸渲染
func renderNinePatch(screen *ebiten.Image, parts DialogParts, x, y, width, height float64) {
    // 实现九宫格拉伸算法
    // ...
}
```

**使用示例：**
```go
// 点击未解锁模式时
func (m *MainMenuScene) showUnlockedDialog(modeName string) {
    dialogEntity := entities.NewDialogEntity(
        m.entityManager,
        m.resourceManager,
        "未解锁！",
        fmt.Sprintf("进行更多新冒险来解锁%s。", modeName),
        []string{"确定"},
    )
    m.currentDialog = dialogEntity
}
```

#### 3. 底部功能栏集成

**MainMenuScene 修改：**
```go
// initBottomButtons - 初始化底部3个花瓶按钮
func (m *MainMenuScene) initBottomButtons() {
    optionsBtn := entities.NewBottomButton(em, rm, BottomButtonOptions,
        rm.LoadImageByID("IMAGE_SELECTOR_OPTIONS1"),
        rm.LoadImageByID("IMAGE_SELECTOR_OPTIONS2"))

    helpBtn := entities.NewBottomButton(em, rm, BottomButtonHelp,
        rm.LoadImageByID("IMAGE_SELECTOR_HELP1"),
        rm.LoadImageByID("IMAGE_SELECTOR_HELP2"))

    quitBtn := entities.NewBottomButton(em, rm, BottomButtonQuit,
        rm.LoadImageByID("IMAGE_SELECTOR_QUIT1"),
        rm.LoadImageByID("IMAGE_SELECTOR_QUIT2"))

    m.bottomButtons = []ecs.EntityID{optionsBtn, helpBtn, quitBtn}
}
```

#### 4. 动画系统集成

**MainMenuScene 修改：**
```go
// playZombieHandAnimation - 播放僵尸手掌动画
func (m *MainMenuScene) playZombieHandAnimation(x, y float64) {
    entity := em.CreateEntity()

    // 加载 Zombie_hand.reanim
    reanimComp := &components.ReanimComponent{
        ReanimName: "Zombie_hand",
        // ... 其他配置
    }
    ecs.AddComponent(em, entity, reanimComp)
    ecs.AddComponent(em, entity, &components.PositionComponent{X: x, Y: y})

    // 播放动画
    m.reanimSystem.PlayAnimation(entity, "anim_rise")

    // 动画结束后销毁实体
    // ...
}

// updateCloudAnimation - 更新云朵动画
func (m *MainMenuScene) updateCloudAnimation(deltaTime float64) {
    // 云朵缓慢从右向左移动
    for _, cloud := range m.clouds {
        posComp, _ := ecs.GetComponent[*components.PositionComponent](em, cloud)
        posComp.X -= CloudSpeed * deltaTime

        // 移出屏幕后重置位置
        if posComp.X < -CloudWidth {
            posComp.X = WindowWidth
        }
    }
}
```

---

### Stories (用户故事)

本 Epic 包含 7 个用户故事，按优先级和实现顺序排序：

#### 阶段 1：核心菜单功能（高优先级）

1. **Story 12.1: 石碑菜单系统完善**
   - **前置条件**: 墓碑升起动画已包含在默认主菜单 Reanim 中，无需重新实现
   - 实现 4 个主模式按钮的交互逻辑（冒险/小游戏/解谜/生存）
   - 添加关卡进度显示（从存档读取，使用 SelectorScreen_LevelNumbers.png）
   - 调整按钮布局符合原版规格
   - 实现按钮状态管理：
     - **高亮状态**: 鼠标悬停时替换为 `_Highlight` 后缀图片，播放石块摩擦音效
     - **锁定状态**: 覆盖 `_shadow_` 半透明图片，点击弹出未解锁提示对话框
   - 实现解锁条件判断：
     - 冒险模式：始终解锁
     - 玩玩小游戏：通关 3-2 后解锁
     - 解谜模式：完成冒险模式后解锁
     - 生存模式：完成冒险模式后解锁
   - 冒险模式按钮特殊逻辑：
     - 新用户显示 `SelectorScreen_StartAdventure_button`
     - 非新用户显示 `SelectorScreen_Adventure_button` + 关卡进度
   - 优先级：⭐⭐⭐⭐⭐ 高
   - 估计工作量：8-12 小时

2. **Story 12.2: 底部功能栏重构**
   - 替换当前退出按钮为 3 个花瓶样式按钮
   - 加载花瓶按钮资源（Options/Help/Quit）
   - 实现按钮交互（正常态/悬停态切换）
   - 添加点击音效
   - 优先级：⭐⭐⭐⭐⭐ 高
   - 估计工作量：4-6 小时

3. **Story 12.3: 对话框系统基础**
   - 创建通用对话框组件（九宫格拉伸）
   - 实现未解锁模式提示对话框
   - 点击未解锁模式显示对话框
   - 实现对话框关闭逻辑（确定/ESC/外部点击）
   - 优先级：⭐⭐⭐⭐⭐ 高
   - 估计工作量：6-8 小时

#### 阶段 2：用户管理与解锁功能（中优先级）

4. **Story 12.4: 用户管理 UI**
   - 实现左上角木牌 UI
   - 显示当前用户名（从存档读取）
   - 实现存档切换对话框
   - 支持存档管理（新建/重命名/删除）
   - 优先级：⭐⭐⭐⭐ 中高
   - 估计工作量：8-10 小时

5. **Story 12.5: 解锁功能入口**
   - 实现图鉴入口（2-4 后解锁）
   - 实现商店入口（3-4 后解锁）
   - 实现禅境花园入口（5-4 后解锁）
   - 实现成就入口（默认解锁）
   - 根据游戏进度动态显示/隐藏
   - 优先级：⭐⭐⭐ 中
   - 估计工作量：6-8 小时

#### 阶段 3：动画与视觉完善（中低优先级）

6. **Story 12.6: Reanim 动画集成**
   - 加载 SelectorScreen.reanim
   - 播放僵尸手掌升起动画
   - 播放石碑升起动画
   - 与现有 ReanimSystem 集成
   - 优先级：⭐⭐⭐ 中
   - 估计工作量：6-8 小时

7. **Story 12.7: 背景动态效果**
   - 实现云朵缓慢飘动
   - 实现草丛轻微晃动
   - 优化动画性能
   - 优先级：⭐⭐ 低
   - 估计工作量：4-6 小时

---

### Compatibility Requirements (兼容性要求)

- ✅ **ECS 架构一致性**: 所有新功能遵循 ECS 模式，零耦合原则
- ✅ **泛型 API 使用**: 使用 Epic 9 的泛型 ECS API，类型安全
- ✅ **资源管理**: 通过 ResourceManager 统一加载资源
- ✅ **场景管理**: 通过 SceneManager 统一场景切换
- ✅ **存档系统集成**: 与 SaveManager 集成，读取游戏进度
- ✅ **Reanim 系统集成**: 与现有 ReanimSystem 集成，播放动画
- ✅ **性能考虑**: 不影响游戏启动速度和运行帧率
- ✅ **原版忠实**: 所有功能还原原版 PC 游戏效果

---

### Risk Mitigation (风险缓解)

**主要风险:**

1. **对话框九宫格拉伸实现复杂** - 需要正确处理边角拉伸
2. **动画资源加载失败** - SelectorScreen.reanim 可能缺失或损坏
3. **存档系统集成问题** - 存档格式不兼容
4. **UI 布局不符合原版** - 按钮位置和大小偏差
5. **性能问题** - 过多动画和实体影响帧率

**缓解措施:**

1. **九宫格拉伸测试** - 单独测试九宫格拉伸算法，确保正确
2. **资源校验** - 启动时校验关键资源存在性，缺失时报错
3. **存档兼容性测试** - 测试新旧存档格式的兼容性
4. **UI 对比测试** - 与原版截图对比，确保布局一致
5. **性能监控** - 监控帧率和内存使用，优化性能瓶颈

**回滚计划:**
- 每个 Story 独立实现，可单独禁用
- 使用功能开关（如配置文件中的 `enableDialogSystem: true`）
- 不影响核心游戏流程，回滚不破坏已有功能

---

### Definition of Done (完成定义)

- ✅ 所有 7 个用户故事完成，AC 全部满足
- ✅ 石碑菜单显示 4 个模式按钮，布局符合原版
- ✅ 关卡进度正确显示（从存档读取）
- ✅ 底部 3 个花瓶按钮正确显示，交互正常
- ✅ 对话框系统实现，九宫格拉伸正确
- ✅ 未解锁模式提示对话框正常弹出
- ✅ 用户管理 UI 显示，存档切换功能正常
- ✅ 解锁功能入口根据进度动态显示
- ✅ 动画系统集成，僵尸手掌和云朵动画播放正常
- ✅ 现有功能无回归，通过 QA 测试
- ✅ 代码符合编码规范，通过 linter 检查
- ✅ 所有新功能有对应的单元测试（覆盖率 > 80%）
- ✅ 文档更新（CLAUDE.md、PRD）

---

## Success Criteria (成功标准)

本 Epic 成功完成的标准：

1. **功能完整性** - 所有核心菜单功能实现并正常工作
2. **原版忠实度** - UI 布局、交互逻辑、视觉效果还原原版
3. **用户体验** - 菜单导航流畅，交互反馈及时
4. **性能稳定** - 主菜单加载快速，运行流畅
5. **代码质量** - 遵循 ECS 架构，代码可读性和可维护性高
6. **测试覆盖** - 所有关键功能有单元测试和集成测试
7. **无回归** - 现有功能（场景切换、资源加载）无破坏

---

## Dependencies (依赖关系)

**前置依赖:**
- Epic 1（游戏基础与主循环）已完成 - 提供场景管理
- Epic 2（核心资源与玩家交互）已完成 - 提供资源管理
- Epic 6（Reanim 动画系统）已完成 - 支持主菜单动画
- Epic 8（关卡实现）已完成 - 提供存档系统

**并行依赖:**
- 阶段 1 的 3 个 Story 可并行开发
- 阶段 2 的 2 个 Story 可并行开发
- 阶段 3 的 2 个 Story 可并行开发

**后续依赖:**
- Epic 13（图鉴系统）- 依赖 Story 12.5 的图鉴入口
- Epic 14（商店系统）- 依赖 Story 12.5 的商店入口
- Epic 15（禅境花园）- 依赖 Story 12.5 的花园入口

---

## Timeline Estimate (时间估算)

基于项目历史速度和复杂度：

| Story | 预估工作量 | 状态 | 优先级 |
|-------|-----------|------|--------|
| Story 12.1: 石碑菜单系统完善 | 8-12 小时 | 待创建 | 高 |
| Story 12.2: 底部功能栏重构 | 4-6 小时 | 待创建 | 高 |
| Story 12.3: 对话框系统基础 | 6-8 小时 | 待创建 | 高 |
| Story 12.4: 用户管理 UI | 8-10 小时 | 待创建 | 中高 |
| Story 12.5: 解锁功能入口 | 6-8 小时 | 待创建 | 中 |
| Story 12.6: Reanim 动画集成 | 6-8 小时 | 待创建 | 中 |
| Story 12.7: 背景动态效果 | 4-6 小时 | 待创建 | 低 |
| **总计** | **42-58 小时** | - | - |

**建议实施顺序:**
1. 阶段 1（必做）- Story 12.1、12.2、12.3 - 18-26 小时
2. 阶段 2（重要）- Story 12.4、12.5 - 14-18 小时
3. 阶段 3（润色）- Story 12.6、12.7 - 10-14 小时

**里程碑:**
- **Milestone 1**: 阶段 1 完成 - 主菜单基本可用
- **Milestone 2**: 阶段 2 完成 - 主菜单功能完整
- **Milestone 3**: 阶段 3 完成 - 主菜单视觉完善

---

## Notes (备注)

- 本 Epic 是游戏的"门面"，直接影响玩家的第一印象
- 所有功能必须忠实还原原版游戏效果，不接受简化实现
- 对话框系统是通用组件，后续其他场景也会使用
- 存档管理功能需要与现有 SaveManager 紧密集成
- 动画效果是提升沉浸感的关键，建议保留
- 解锁功能入口为后续 Epic 铺路，优先级可适当降低

---

## Change Log (变更日志)

| Date | Version | Description | Author |
|------|---------|-------------|--------|
| 2025-11-01 | 1.0 | 初始 Epic 创建，基于 main-menu-spec.md 和 data.md 分析 | Sarah (PO) |
