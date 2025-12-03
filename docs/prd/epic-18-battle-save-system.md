# Epic 18: 游戏战斗存档系统 (Battle Save System)

## 史诗目标

为《植物大战僵尸》复刻版实现战斗中存档/读档功能。玩家点击暂停菜单的"主菜单"按钮时自动保存当前战斗进度，再次进入冒险模式时可选择继续游戏或重玩关卡，提供简洁的用户交互体验。

## 背景与动机

### 当前实现状况

现有的 `SaveManager` (Story 8.6) 实现了关卡进度保存系统：
- **存储格式**: YAML
- **存储内容**: 最高完成关卡、解锁植物、解锁工具
- **存储位置**: `data/saves/{username}.yaml`
- **触发时机**: 关卡完成时自动保存

**不足之处**：
- 不支持游戏中途保存
- 退出关卡会丢失当前战斗进度
- 无法从中断处继续游戏

### 业务价值

- **用户体验**: 允许玩家随时退出游戏，下次继续
- **数据安全**: 防止意外退出导致进度丢失
- **原版忠实**: 原版 PvZ 支持战斗中保存功能

## Epic 概览

### 核心功能

| 功能 | 说明 |
|------|------|
| **自动存档** | 点击暂停菜单"主菜单"按钮时自动保存战斗状态 |
| **存档检测** | 进入冒险模式时检测是否有未完成的战斗存档 |
| **继续游戏对话框** | 显示"继续游戏？"对话框，提供继续/重玩/取消选项 |
| **场景恢复** | 从存档恢复完整的游戏场景（植物、僵尸、阳光等） |

### 技术方案

- **存档格式**: gob 二进制序列化（紧凑、快速、防修改）
- **存档位置**: `data/saves/{username}_battle.sav`
- **架构模式**: 遵循 ECS 架构，组件数据序列化

### 用户交互流程

**保存流程**:
```
游戏中 → 按 ESC → 暂停菜单 → 点击"主菜单" → 自动保存战斗进度 → 返回主菜单
```

**加载流程**:
```
主菜单 → 点击"冒险模式" → 检测到存档 → 显示对话框
         ↓
    ┌────────────────────────────────┐
    │        继续游戏?               │
    │                                │
    │  你想继续当前游戏还是重玩此关卡？│
    │                                │
    │    [继续]      [重玩关卡]       │
    │           [取消]               │
    └────────────────────────────────┘
         ↓                ↓              ↓
     加载存档         删除存档并        返回主菜单
     继续游戏         重新开始关卡
```

## Stories (故事列表)

Epic 18 包含 **3 个 Story**:

---

### Story 18.1: 战斗状态序列化系统

> **As a** 开发者,
> **I want** to implement battle state serialization using gob binary format,
> **so that** the game can save and restore complete battle state.

#### Acceptance Criteria

1. **BattleSaveData 数据结构**:
   - 定义完整的战斗状态数据结构
   - 包含关卡状态、资源状态、所有实体数据
   - 支持版本号用于兼容性检查

2. **实体数据序列化**:
   - 植物: 类型、网格位置、生命值、攻击状态、冷却时间
   - 僵尸: 类型、位置、速度、生命值、护甲、行为状态
   - 子弹: 类型、位置、速度、伤害
   - 阳光: 位置、生命周期状态
   - 除草车: 所在行、是否已触发

3. **BattleSerializer 实现**:
   - `SaveBattle(em, gameState, filePath) error`
   - `LoadBattle(filePath) (*BattleSaveData, error)`
   - gob 编码/解码错误处理

4. **SaveManager 扩展**:
   - `HasBattleSave(username) bool`
   - `GetBattleSaveInfo(username) *BattleSaveInfo`
   - `DeleteBattleSave(username) error`
   - `GetBattleSavePath(username) string`

5. **单元测试**: 覆盖率 ≥ 80%

#### 技术实现

**新增文件**:
- `pkg/game/battle_save_data.go` - 战斗存档数据结构
- `pkg/game/battle_serializer.go` - gob 序列化器

**修改文件**:
- `pkg/game/save_manager.go` - 添加战斗存档管理方法

**预估工作量**: 8-10 小时

---

### Story 18.2: 保存触发与自动加载

> **As a** 玩家,
> **I want** the game to automatically save my battle progress when I return to main menu,
> **so that** I don't lose my progress when exiting the game.

#### Acceptance Criteria

1. **暂停菜单"主菜单"按钮触发保存**:
   - 修改 `PauseMenuModule.onMainMenu` 回调
   - 调用 `BattleSerializer.SaveBattle()` 保存当前状态
   - 保存成功后才返回主菜单
   - 保存失败时显示错误提示

2. **冒险模式入口检测存档**:
   - 修改 `MainMenuScene.onStartAdventureClicked()`
   - 检查 `SaveManager.HasBattleSave(username)`
   - 有存档时触发继续游戏对话框（Story 18.3）
   - 无存档时正常进入关卡选择

3. **存档信息读取**:
   - 实现 `BattleSaveInfo` 结构（关卡ID、保存时间）
   - 用于对话框显示存档信息

4. **日志输出**:
   - 记录保存/加载操作
   - 支持调试模式显示详细信息

5. **单元测试**: 覆盖率 ≥ 80%

#### 技术实现

**修改文件**:
- `pkg/modules/pause_menu_module.go` - 主菜单按钮触发保存
- `pkg/scenes/main_menu_scene.go` - 冒险模式检测存档

**预估工作量**: 4-6 小时

---

### Story 18.3: 继续游戏对话框与场景恢复

> **As a** 玩家,
> **I want** to choose whether to continue my saved game or restart the level,
> **so that** I have control over my game progress.

#### Acceptance Criteria

1. **继续游戏对话框**:
   - Title: "继续游戏?"
   - Message: "你想继续当前游戏还是重玩此关卡？"
   - 按钮布局（两行）:
     - 第一行: [继续] [重玩关卡]
     - 第二行: [取消]
   - 复用 `DialogComponent` 和现有对话框样式

2. **按钮行为**:
   - **继续**: 调用 `GameScene.LoadFromSave()` 恢复游戏
   - **重玩关卡**: 删除存档，正常开始该关卡
   - **取消**: 关闭对话框，返回主菜单

3. **GameScene.LoadFromSave() 实现**:
   - 恢复 GameState 状态（阳光、波次、时间等）
   - 重建所有实体（植物、僵尸、子弹、阳光、除草车）
   - 恢复草坪网格状态
   - 恢复系统状态（波次计时等）

4. **动画状态恢复**:
   - Reanim 动画从保存的帧恢复
   - 允许轻微的动画跳跃（可接受）

5. **错误处理**:
   - 存档损坏时提示用户
   - 提供删除损坏存档的选项

6. **单元测试**: 覆盖率 ≥ 80%

#### 技术实现

**修改文件**:
- `pkg/scenes/main_menu_scene.go` - 创建继续游戏对话框
- `pkg/scenes/game_scene.go` - 添加 `LoadFromSave()` 方法

**预估工作量**: 8-10 小时

---

## 需要保存的战斗状态

### 关卡状态

| 字段 | 类型 | 说明 |
|------|------|------|
| `LevelID` | string | 关卡ID (如 "1-2") |
| `LevelTime` | float64 | 关卡已进行时间（秒） |
| `CurrentWaveIndex` | int | 当前波次索引 |
| `SpawnedWaves` | []bool | 已生成波次标记 |
| `TotalZombiesSpawned` | int | 已生成僵尸总数 |
| `ZombiesKilled` | int | 已消灭僵尸数 |

### 资源状态

| 字段 | 类型 | 说明 |
|------|------|------|
| `Sun` | int | 当前阳光数量 |

### 实体数据

#### PlantData
```go
type PlantData struct {
    PlantType    string  // 植物类型
    GridRow      int     // 网格行
    GridCol      int     // 网格列
    Health       int     // 当前生命值
    MaxHealth    int     // 最大生命值
    CooldownTime float64 // 剩余冷却时间
    // Reanim 状态（可选）
}
```

#### ZombieData
```go
type ZombieData struct {
    ZombieType   string  // 僵尸类型
    X            float64 // X 坐标
    Y            float64 // Y 坐标
    VelocityX    float64 // X 速度
    Health       int     // 当前生命值
    MaxHealth    int     // 最大生命值
    ArmorHealth  int     // 护甲生命值（如有）
    Lane         int     // 所在行
    BehaviorType string  // 当前行为类型
}
```

#### ProjectileData
```go
type ProjectileData struct {
    Type      string  // 子弹类型
    X         float64 // X 坐标
    Y         float64 // Y 坐标
    VelocityX float64 // X 速度
    Damage    int     // 伤害值
    Lane      int     // 所在行
}
```

#### SunData
```go
type SunData struct {
    X         float64 // X 坐标
    Y         float64 // Y 坐标
    Lifetime  float64 // 剩余生命周期
    IsDropped bool    // 是否为天降阳光
}
```

#### LawnmowerData
```go
type LawnmowerData struct {
    Lane      int  // 所在行
    Triggered bool // 是否已触发
    X         float64 // X 坐标（如已触发）
}
```

## Success Criteria (成功标准)

1. ✅ 点击暂停菜单"主菜单"按钮自动保存战斗进度
2. ✅ 进入冒险模式时正确检测存档
3. ✅ 继续游戏对话框显示正确，按钮布局符合设计
4. ✅ "继续"按钮恢复游戏状态，与保存时一致
5. ✅ "重玩关卡"按钮删除存档并重新开始
6. ✅ "取消"按钮返回主菜单
7. ✅ 单元测试覆盖率 ≥ 80%
8. ✅ 无回归问题

## Technical Implementation (技术实现)

### 新增文件

| 文件 | 职责 |
|------|------|
| `pkg/game/battle_save_data.go` | 战斗存档数据结构定义 |
| `pkg/game/battle_serializer.go` | gob 序列化/反序列化 |

### 修改文件

| 文件 | 修改内容 |
|------|----------|
| `pkg/game/save_manager.go` | 添加战斗存档管理方法 |
| `pkg/modules/pause_menu_module.go` | 主菜单按钮触发保存 |
| `pkg/scenes/main_menu_scene.go` | 冒险模式检测存档，创建对话框 |
| `pkg/scenes/game_scene.go` | 添加 `LoadFromSave()` 方法 |

### 存档文件结构

```
data/
└── saves/
    ├── {username}.yaml           # 关卡进度存档（现有）
    └── {username}_battle.sav     # 战斗存档（新增，gob 二进制）
```

## Dependencies and Blockers (依赖与阻塞)

### 前置依赖

- ✅ Story 8.6 SaveManager 已实现
- ✅ Story 10.1 暂停菜单系统已实现
- ✅ Story 12.4 用户管理系统已实现
- ✅ DialogComponent 对话框组件已实现

### 无阻塞项

本 Epic 不阻塞其他 Epic，可独立开发。

## Timeline Estimate (时间估算)

| Story | 预估工作量 | 优先级 | 依赖 |
|-------|-----------|--------|------|
| Story 18.1: 战斗状态序列化系统 | 8-10 小时 | 高 | 无 |
| Story 18.2: 保存触发与自动加载 | 4-6 小时 | 高 | 18.1 |
| Story 18.3: 继续游戏对话框与场景恢复 | 8-10 小时 | 高 | 18.1, 18.2 |
| **总计** | **20-26 小时** | - | - |

**预估周期**: 3-5 天（单人开发）

## Reference Documentation (参考文档)

- **现有保存系统**: `pkg/game/save_manager.go`
- **暂停菜单模块**: `pkg/modules/pause_menu_module.go`
- **对话框组件**: `pkg/components/dialog_component.go`
- **游戏场景**: `pkg/scenes/game_scene.go`

---

**创建日期**: 2025-11-29
**创建人**: John (Product Manager)
**Epic 类型**: Brownfield Enhancement (棕地增强)
**优先级**: Medium（用户体验改进）
**预估总工作量**: 20-26 小时（约 3-5 天）
