# 僵尸获胜流程验证程序 (Story 8.8)

## 概述

该验证程序用于可视化测试和调试 Story 8.8 实现的僵尸获胜流程（四阶段）：

1. **Phase 1: 游戏冻结** (1.5秒)
   - 植物停止攻击动画
   - 子弹消失
   - UI 元素隐藏
   - 背景音乐淡出

2. **Phase 2: 僵尸入侵** (动态时长)
   - 僵尸继续行走至屏幕外 (X < -100)
   - 摄像机平滑左移至世界坐标 0

3. **Phase 3: 惨叫与动画** (3-4秒)
   - 播放惨叫音效 (`scream.ogg`)
   - 延迟 0.5秒 播放咀嚼音效 (`chomp_soft.ogg`)
   - 显示 `ZombiesWon.reanim` 动画
   - 屏幕轻微抖动 (±5 像素, 10Hz)

4. **Phase 4: 游戏结束对话框**
   - 等待玩家点击或 3-5秒 超时
   - 显示游戏结束对话框（模拟）

## 编译

```bash
go build -o bin/verify_zombies_won ./cmd/verify_zombies_won/main.go
```

## 运行

### 基本用法

```bash
./bin/verify_zombies_won
```

启动后，按 **Space** 键触发僵尸获胜流程。

### 命令行参数

```bash
# 跳过 Phase 1（直接从 Phase 2 开始）
./bin/verify_zombies_won --skip-phase1

# 跳过 Phase 2（直接从 Phase 3 开始）
./bin/verify_zombies_won --skip-phase2

# 跳过 Phase 3（直接从 Phase 4 开始）
./bin/verify_zombies_won --skip-phase3

# 快速模式（时间流逝加速 3 倍）
./bin/verify_zombies_won --fast

# 显示详细调试信息
./bin/verify_zombies_won --verbose

# 组合使用
./bin/verify_zombies_won --skip-phase1 --fast --verbose
```

## 快捷键

| 按键 | 功能 |
|------|------|
| **Space** | 启动僵尸获胜流程 |
| **1** | 跳转到 Phase 1（游戏冻结） |
| **2** | 跳转到 Phase 2（僵尸入侵） |
| **3** | 跳转到 Phase 3（惨叫动画） |
| **4** | 跳转到 Phase 4（游戏结束对话框） |
| **R** | 重启验证程序 |
| **Q** | 退出程序 |

## 屏幕调试信息

程序运行时会在屏幕左上角显示：

- 当前阶段编号和计时器
- Phase 2：僵尸位置和摄像机位置
- Phase 3：音效播放状态（✅ 标记）
- Phase 4：等待对话框提示
- 完成状态和快捷键提示

## 验证要点

### Phase 1 验证
- ✅ 植物和子弹停止更新
- ✅ UI 元素隐藏（理论上，本验证程序未渲染 UI）
- ✅ 背景音乐淡出（需要实际播放背景音乐才能验证）

### Phase 2 验证
- ✅ 僵尸继续向左移动（VX = -150.0）
- ✅ 僵尸 X 坐标逐渐减小，直到 < -100
- ✅ 摄像机平滑左移（速度 200 像素/秒），直到 X = 0

### Phase 3 验证
- ✅ 惨叫音效播放（日志显示 "惨叫音效已播放"）
- ✅ 咀嚼音效延迟 0.5秒 播放（日志显示 "咀嚼音效已播放"）
- ✅ `ZombiesWon` 动画在屏幕中央显示（需要确认 reanim 资源加载正确）
- ✅ 屏幕抖动效果（观察背景是否轻微晃动）

### Phase 4 验证
- ✅ 等待 3-5 秒后自动显示对话框（日志显示 "显示游戏结束对话框"）
- ✅ 点击屏幕任意位置可提前触发对话框（可选，本验证程序未实现）

## 常见问题

### Q: 僵尸动画没有显示？
**A**: 确保已正确加载 Reanim 资源。检查 `data/reanim_config/zombie.yaml` 是否存在。

### Q: 音效没有播放？
**A**: 音效播放依赖音频文件 `assets/audio/scream.ogg` 和 `chomp_soft.ogg`。检查文件是否存在。

### Q: 摄像机没有移动？
**A**: 摄像机移动仅在 Phase 2 期间生效。确保僵尸已触发流程并进入 Phase 2。

### Q: 如何快速测试某个特定阶段？
**A**: 使用快捷键 1-4 直接跳转到指定阶段，或使用命令行参数 `--skip-phase1/2/3`。

## 开发备注

### 系统依赖
- `ZombiesWonPhaseSystem` - 核心流程管理系统
- `ReanimSystem` - 动画播放系统
- `RenderSystem` - 游戏世界渲染系统
- `ResourceManager` - 资源加载和音频管理

### 组件使用
- `ZombiesWonPhaseComponent` - 阶段状态机组件
- `GameFreezeComponent` - 游戏冻结标记组件
- `AnimationCommandComponent` - 动画播放命令组件（Epic 14）

### 对话框回调
验证程序中的对话框显示为模拟日志输出（`onShowDialog()`）。
实际游戏中会调用 `entities.NewGameOverDialogEntity()` 创建真实对话框。

## 相关文档

- Story 文档: `docs/stories/8.8.story.md`
- Sprint Change Proposal: `docs/sprint-change-proposals/2025-11-20-zombies-won-game-over-flow.md`
- 元数据参考: `.meta/levels/zombiewon.md`

## 作者

Claude Code (James) - Story 8.8 实现
