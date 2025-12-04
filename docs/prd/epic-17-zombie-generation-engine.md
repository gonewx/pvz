# Epic 17: 僵尸生成引擎 (Zombie Generation Engine)

## 史诗目标

实现《植物大战僵尸》冒险模式的精确僵尸生成系统，包括难度动态调整、行分配平滑权重算法、波次计时控制和旗帜波特殊机制，完全还原原版游戏的出怪逻辑和游戏节奏。

## 背景与动机

### 当前实现状况

现有的 `WaveSpawnSystem` 是一个简化的僵尸生成系统：
- **固定延迟触发**: 基于 `delay` 字段的简单计时器
- **随机行选择**: 从 `lanes` 数组中均匀随机选择
- **无难度调整**: 不支持轮数和级别容量计算
- **无加速刷新**: 不支持血量触发的波次加速机制

### 原版机制复杂性

根据逆向工程分析，原版 PvZ 冒险模式的僵尸生成包含以下核心机制：

1. **难度引擎**: 基于"轮数"的动态难度调整
2. **级别容量上限**: 限制每波僵尸的总"级别"
3. **平滑权重行分配**: 复杂的行选择算法，避免连续出同一行
4. **刷新计时器**: 区分常规波/旗帜波/最终波的不同计时逻辑
5. **加速刷新机制**: 基于血量阈值的波次提前触发
6. **红字警告**: 旗帜波前的特殊提示时序

### 业务价值

- **原版忠实度**: 实现 95%+ 的原版出怪逻辑还原
- **游戏体验**: 确保关卡难度曲线和节奏感符合原版
- **可扩展性**: 支持未来章节（夜晚、泳池、屋顶）的僵尸生成扩展
- **数据驱动**: 所有参数可配置，便于数值调整

## Epic 概览

### 模块架构

```
┌─────────────────────────────────────────────────────────────┐
│                   Zombie Generation Engine                   │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌──────────────────┐  ┌─────────────┐ │
│  │  Difficulty     │  │  Script Parser   │  │  Lane       │ │
│  │  Engine         │  │  & Executor      │  │  Allocator  │ │
│  │                 │  │                  │  │             │ │
│  │  - Rounds       │  │  - YAML Parser   │  │  - Smooth   │ │
│  │  - Level Cap    │  │  - Fixed Script  │  │    Weight   │ │
│  │  - Zombie Stats │  │  - Constraints   │  │  - Legal    │ │
│  └────────┬────────┘  └────────┬─────────┘  │    Rows     │ │
│           │                    │            └──────┬──────┘ │
│           └────────────┬───────┴───────────────────┘        │
│                        ▼                                    │
│              ┌─────────────────────┐                        │
│              │   Wave Timing       │                        │
│              │   Controller        │                        │
│              │                     │                        │
│              │   - Refresh Timer   │                        │
│              │   - Flag Wave       │                        │
│              │   - Speed Trigger   │                        │
│              └─────────────────────┘                        │
└─────────────────────────────────────────────────────────────┘
```

### 核心组件

| 组件 | 职责 | 复杂度 |
|------|------|--------|
| **DifficultyEngine** | 轮数计算、级别容量上限 | 中 |
| **ZombieScriptParser** | 关卡脚本 YAML 解析、验证 | 低 |
| **LaneAllocator** | 平滑权重行分配算法 | 高 |
| **WaveTimingController** | 波次计时、加速刷新、旗帜波 | 高 |
| **SpawnConstraintChecker** | 僵尸生成限制检查 | 中 |

### 关卡脚本格式升级

**当前格式**:
```yaml
waves:
  - delay: 5
    zombies:
      - type: "basic"
        lanes: [3]
        count: 1
```

**目标格式** (兼容原版机制):
```yaml
levelID: "1-1"
flags: 1
sceneType: "day"
rowMax: 5

waves:
  - waveNum: 1
    type: "Fixed"
    zombies:
      - id: "basic"
        count: 3
      - id: "conehead"
        count: 1
    extraPoints: 0
    laneRestriction: null
```

## Stories (故事列表)

Epic 17 包含 **10 个 Story**:

---

### Story 17.1: 难度引擎 - 轮数与级别容量计算

> **As a** 开发者,
> **I want** to implement the difficulty engine with round calculation and level capacity caps,
> **so that** zombie spawning follows the original game's dynamic difficulty system.

#### Acceptance Criteria

1. **轮数计算**:
   - 实现公式: `RoundNumber = TotalCompletedFlags / 2 - 1`
   - 支持一周目/二周目状态判断
   - 正确处理负数轮数（一周目早期关卡）

2. **级别容量上限**:
   - 实现公式: `CapacityCap = int(int((CurrentWaveNum + RoundNumber * WavesPerRound) * 0.8) / 2) + 1`
   - 大波（旗帜波）容量 × 2.5 并向零取整
   - 级别上限限制每波僵尸的级别总和

3. **僵尸级别数据**:
   - 普通僵尸: 级别 1
   - 路障僵尸: 级别 2
   - 铁桶僵尸: 级别 4
   - 巨人僵尸: 级别 10

4. **配置文件**:
   - 创建 `data/zombie_stats.yaml` 存储僵尸属性
   - 支持热加载和数值调整

5. **单元测试**: 覆盖率 ≥ 80%

#### 技术实现

**新增文件**:
- `pkg/config/difficulty.go` - 难度引擎配置
- `pkg/systems/difficulty_engine.go` - 难度计算系统
- `data/zombie_stats.yaml` - 僵尸属性配置

**预估工作量**: 6-8 小时

---

### Story 17.2: 关卡脚本格式升级与解析器

> **As a** 开发者,
> **I want** to upgrade the level script format to support original PvZ wave definitions,
> **so that** the game can use accurate fixed zombie sequences.

#### Acceptance Criteria

1. **新脚本格式**:
   - 支持 `levelID`, `flags`, `sceneType`, `rowMax` 顶层字段
   - 支持 `waves[].waveNum`, `waves[].type`, `waves[].zombies`, `waves[].extraPoints`
   - 向后兼容现有关卡配置

2. **波次类型**:
   - `Fixed`: 固定出怪列表
   - `ExtraPoints`: 动态点数分配（用于 1-10 等特殊关卡）
   - `Final`: 最终波

3. **解析器实现**:
   - 扩展 `pkg/config/level.go` 支持新字段
   - 验证脚本完整性（必填字段、类型检查）
   - 错误提示友好

4. **迁移脚本**:
   - 将现有 `data/levels/*.yaml` 迁移到新格式
   - 保留旧格式支持（降级兼容）

5. **单元测试**: 解析器测试覆盖率 ≥ 90%

#### 技术实现

**修改文件**:
- `pkg/config/level.go` - 扩展配置结构
- `data/levels/*.yaml` - 更新关卡配置

**预估工作量**: 4-6 小时

---

### Story 17.3: 僵尸生成限制检查系统

> **As a** 开发者,
> **I want** to implement constraint checking before spawning zombies,
> **so that** only valid zombies are spawned according to game rules.

#### Acceptance Criteria

1. **出怪类型检查**:
   - 验证僵尸类型是否在当前关卡允许列表中
   - 支持场地类型限制（水路僵尸、舞王限制等）

2. **阶数限制**:
   - 一阶僵尸: 第 1 波起可出现
   - 二阶僵尸: 第 3 波起
   - 三阶僵尸: 第 8 波起
   - 四阶僵尸: 第 15 波起（根据轮数调整）

3. **红眼数量上限**:
   - 初始上限: 0
   - 轮数 ≥ 5 起，每轮 +1
   - 生成前检查已生成红眼数量

4. **配置驱动**:
   - 限制规则存储在配置文件中
   - 支持按关卡覆盖默认规则

5. **单元测试**: 覆盖率 ≥ 80%

#### 技术实现

**新增文件**:
- `pkg/systems/spawn_constraint_system.go` - 生成限制检查

**预估工作量**: 4-6 小时

---

### Story 17.4: 平滑权重行分配算法 - 核心实现

> **As a** 开发者,
> **I want** to implement the smooth weight lane allocation algorithm,
> **so that** zombies are distributed across lanes in a natural, non-repetitive pattern.

#### Acceptance Criteria

1. **行信息初始化**:
   - 每行维护: `Weight`, `LastPicked`, `SecondLastPicked`
   - 冒险模式初始权重为 1
   - 更新计数器逻辑（选中行重置，其他行递增）

2. **平滑权重计算**:
   - 权重占比: `WeightP = Weight[i] / Sum(Weight)`
   - 影响因子 PLast 和 PSecondLast 公式实现
   - 平滑权重: `SmoothWeight = WeightP * clamp(PLast + PSecondLast, 0.01, 100)`

3. **抽取逻辑**:
   - 在 `[0, Sum(SmoothWeight))` 范围内随机抽取
   - 累积和方式确定选中行
   - 无可选行时默认第六行

4. **可视化调试**:
   - 支持日志输出行权重分布
   - 调试模式下显示行选择概率

5. **单元测试**:
   - 边界条件测试（单行、全零权重）
   - 分布均匀性统计测试
   - 覆盖率 ≥ 90%

#### 技术实现

**新增文件**:
- `pkg/systems/lane_allocator.go` - 行分配器
- `pkg/components/lane_state.go` - 行状态组件

**预估工作量**: 8-12 小时

---

### Story 17.5: 合法行判定系统

> **As a** 开发者,
> **I want** to implement legal row checking for different zombie types,
> **so that** zombies are only spawned in valid lanes.

#### Acceptance Criteria

1. **基本不合法条件**:
   - 行号越界 (< 1 或 > 6)
   - 无草皮关卡的裸地行
   - 非泳池/浓雾关卡的第 6 行

2. **水路限制**:
   - 后院第 3、4 行为水路
   - 水路只允许: 潜水、海豚、鸭子救生圈版僵尸
   - 非水路对海豚/潜水不合法

3. **舞王僵尸限制**:
   - 非后院场景：上下相邻行必须是草地
   - 屋顶场景完全不合法

4. **配置驱动**:
   - 僵尸-行合法性规则在配置文件中定义
   - 支持按场景类型覆盖规则

5. **单元测试**: 覆盖率 ≥ 85%

#### 技术实现

**修改文件**:
- `pkg/systems/lane_allocator.go` - 添加合法性检查

**预估工作量**: 4-6 小时

---

### Story 17.6: 波次刷新计时器 - 常规波次

> **As a** 开发者,
> **I want** to implement the wave refresh timing system for regular waves,
> **so that** zombie waves spawn at correct intervals.

#### Acceptance Criteria

1. **开场倒计时**:
   - 首次选卡后: 无额外倒计时
   - 非首次: 600 厘秒 (6秒) 倒计时
   - 从 599 减少到 1 时触发第一波

2. **常规波次计时**:
   - 刷新时设置: `2500 + rand(600)` 厘秒 (25-31秒)
   - 倒计时降至 1 时触发下一波

3. **计时器组件**:
   - 创建 `WaveTimerComponent` 存储计时状态
   - 支持暂停/恢复

4. **日志输出**:
   - 记录每波触发时间点
   - 支持调试模式显示计时器状态

5. **单元测试**: 覆盖率 ≥ 80%

#### 技术实现

**新增文件**:
- `pkg/components/wave_timer.go` - 波次计时组件
- `pkg/systems/wave_timing_system.go` - 波次计时系统

**预估工作量**: 6-8 小时

---

### Story 17.7: 旗帜波特殊计时与红字警告

> **As a** 开发者,
> **I want** to implement flag wave special timing and "huge wave" warning,
> **so that** the game follows original flag wave mechanics.

#### Acceptance Criteria

1. **旗帜波前一波 (第 9/19 波)**:
   - 倒计时设置为 4500 厘秒 (45秒)
   - 加速刷新: 刷出 401cs 后且倒计时 > 200cs，消灭本波僵尸（除伴舞）后设为 200cs

2. **红字警告**:
   - 倒计时降至 5 时显示红字
   - 在 4 停留 725 厘秒
   - 红字倒计时结束后 (~750cs) 触发下一波

3. **最终波 (第 20 波)**:
   - 倒计时设置为 5500 厘秒 (55秒)
   - 减至 0 时激活白字
   - 场上僵尸消灭后显示白字 500cs 结束关卡

4. **UI 集成**:
   - 红字 "A Huge Wave of Zombies is Approaching!" 动画
   - 白字 "FINAL WAVE" 动画
   - 使用原版字体和样式

5. **单元测试**: 覆盖率 ≥ 80%

#### 技术实现

**修改文件**:
- `pkg/systems/wave_timing_system.go` - 添加旗帜波逻辑
- `pkg/scenes/game_scene.go` - UI 渲染集成

**预估工作量**: 8-10 小时

---

### Story 17.8: 加速刷新机制 - 血量触发

> **As a** 开发者,
> **I want** to implement health-triggered wave acceleration,
> **so that** the game speeds up when zombies are quickly defeated.

#### Acceptance Criteria

1. **血量计算**:
   - 总血量 = 本体血量 + I类饰品血量 + 0.20 × II类饰品血量
   - I类饰品: 路障 370, 铁桶 1100, 橄榄球帽, 雪橇车, 气球, 矿工帽, 僵尸坚果
   - II类饰品: 报纸, 铁栅门, 扶梯

2. **刷新激活条件**:
   - 常规波次（非大波）
   - 本波刷出 ≥ 401 厘秒
   - 当前血量 ≤ 刷新激活血量 (总血量 × 0.50~0.65)
   - 触发后倒计时立即设为 200 厘秒

3. **血量追踪**:
   - 追踪本波所有僵尸的血量变化
   - 僵尸死亡时更新总血量

4. **配置参数**:
   - 血量触发比例可配置 (默认 0.50~0.65 随机)
   - 最小触发时间可配置 (默认 401cs)

5. **单元测试**: 覆盖率 ≥ 80%

#### 技术实现

**修改文件**:
- `pkg/systems/wave_timing_system.go` - 添加加速刷新逻辑
- `pkg/components/wave_timer.go` - 添加血量追踪字段

**预估工作量**: 6-8 小时

---

### Story 17.9: 僵尸物理与边界 - 坐标系统

> **As a** 开发者,
> **I want** to implement accurate zombie spawn coordinates and boundary detection,
> **so that** zombies appear and behave according to original game physics.

#### Acceptance Criteria

1. **坐标系**:
   - 以 Level 1-1 格子左上角为原点 (0, 0)
   - 行坐标: 前院僵尸 Y 坐标计算

2. **出生点 X 坐标**:
   - 普通波: 780~819 (随机)
   - 旗帜波: 820~859 (随机)
   - 旗帜僵尸: 固定 800
   - 冰车: 800~809
   - 巨人: 845~854

3. **进家判定**:
   - 普僵/路障/铁桶: X ≤ -100
   - 橄榄/冰车/投篮: X ≤ -175
   - 舞王/伴舞/潜水: X ≤ -130
   - 撑杆/巨人: X ≤ -150

4. **配置文件**:
   - `data/zombie_physics.yaml` 存储物理参数
   - 支持按僵尸类型自定义

5. **单元测试**: 覆盖率 ≥ 80%

#### 技术实现

**新增文件**:
- `data/zombie_physics.yaml` - 僵尸物理配置
- `pkg/config/zombie_physics.go` - 物理配置加载

**修改文件**:
- `pkg/systems/wave_spawn_system.go` - 使用精确坐标
- `pkg/systems/level_system.go` - 使用类型化进家判定

**预估工作量**: 4-6 小时

---

### Story 17.10: 集成测试与关卡验证

> **As a** 开发者,
> **I want** to verify the zombie generation engine through integration tests,
> **so that** all mechanics work correctly together.

#### Acceptance Criteria

1. **关卡 1-1 至 1-4 验证**:
   - 每波僵尸数量正确
   - 波次时间间隔符合预期
   - 行分配无连续重复

2. **难度曲线验证**:
   - 一周目难度曲线正确
   - 轮数计算验证

3. **边界条件测试**:
   - 单行关卡 (1-1)
   - 多旗帜关卡 (1-7, 1-9)
   - 最终波触发

4. **性能测试**:
   - 20 波僵尸生成无卡顿
   - 行分配算法 < 1ms

5. **回归测试**:
   - 现有关卡功能无回归
   - 向后兼容性验证

#### 技术实现

**新增文件**:
- `pkg/systems/zombie_generation_test.go` - 集成测试

**预估工作量**: 6-8 小时

---

## Success Criteria (成功标准)

1. ✅ 难度引擎正确计算轮数和级别容量
2. ✅ 关卡脚本格式升级完成，向后兼容
3. ✅ 平滑权重行分配算法实现，分布自然
4. ✅ 波次计时系统完整，包括旗帜波和最终波
5. ✅ 加速刷新机制正确触发
6. ✅ 僵尸出生坐标和进家判定准确
7. ✅ 关卡 1-1 至 1-4 验证通过
8. ✅ 单元测试覆盖率 ≥ 80%
9. ✅ 集成测试全部通过
10. ✅ 性能符合要求 (60 FPS)

## Technical Implementation (技术实现)

### 新增组件

| 组件 | 文件 | 职责 |
|------|------|------|
| `LaneStateComponent` | `pkg/components/lane_state.go` | 行权重和选中历史 |
| `WaveTimerComponent` | `pkg/components/wave_timer.go` | 波次计时状态 |
| `DifficultyComponent` | `pkg/components/difficulty.go` | 难度参数 |

### 新增/修改系统

| 系统 | 文件 | 职责 |
|------|------|------|
| `DifficultyEngine` | `pkg/systems/difficulty_engine.go` | 难度计算 |
| `LaneAllocator` | `pkg/systems/lane_allocator.go` | 行分配 |
| `WaveTimingSystem` | `pkg/systems/wave_timing_system.go` | 波次计时 |
| `SpawnConstraintSystem` | `pkg/systems/spawn_constraint_system.go` | 生成约束 |
| `WaveSpawnSystem` | `pkg/systems/wave_spawn_system.go` | 重构整合 |

### 新增配置文件

| 文件 | 内容 |
|------|------|
| `data/zombie_stats.yaml` | 僵尸属性（级别、权重、血量） |
| `data/zombie_physics.yaml` | 僵尸物理参数（坐标、边界） |
| `data/spawn_rules.yaml` | 生成规则（阶数、红眼上限） |

### 系统交互流程

```
┌─────────────┐     ┌─────────────────┐     ┌────────────────┐
│ LevelSystem │────▶│ WaveTimingSystem│────▶│ WaveSpawnSystem│
└─────────────┘     └────────┬────────┘     └───────┬────────┘
                             │                      │
                             ▼                      ▼
                    ┌────────────────┐     ┌────────────────┐
                    │ DifficultyEngine│     │ LaneAllocator  │
                    └────────────────┘     └───────┬────────┘
                                                   │
                                                   ▼
                                          ┌────────────────┐
                                          │ConstraintChecker│
                                          └────────────────┘
```

## Dependencies and Blockers (依赖与阻塞)

### 前置依赖

- ✅ Epic 1-11 已完成（基础框架、ECS、关卡系统）
- ✅ Epic 14 ECS 解耦完成
- ✅ Epic 16 坐标系统重构完成

### 无阻塞项

本 Epic 不阻塞其他 Epic，可独立开发。

### 后续扩展

- Epic 18+: 夜晚/泳池/屋顶场景僵尸生成（依赖本 Epic）
- 特殊僵尸行为（撑杆跳跃、巨人扔小鬼）

## Timeline Estimate (时间估算)

| Story | 预估工作量 | 优先级 | 依赖 |
|-------|-----------|--------|------|
| Story 17.1: 难度引擎 | 6-8 小时 | 高 | 无 |
| Story 17.2: 脚本格式升级 | 4-6 小时 | 高 | 无 |
| Story 17.3: 生成限制检查 | 4-6 小时 | 中 | 17.1 |
| Story 17.4: 平滑权重算法 | 8-12 小时 | 高 | 无 |
| Story 17.5: 合法行判定 | 4-6 小时 | 中 | 17.4 |
| Story 17.6: 常规波次计时 | 6-8 小时 | 高 | 无 |
| Story 17.7: 旗帜波特殊计时 | 8-10 小时 | 高 | 17.6 |
| Story 17.8: 加速刷新机制 | 6-8 小时 | 中 | 17.6 |
| Story 17.9: 僵尸物理坐标 | 4-6 小时 | 中 | 无 |
| Story 17.10: 集成测试 | 6-8 小时 | 高 | 17.1-17.9 |
| **总计** | **56-78 小时** | - | - |

**预估周期**: 2-3 周（单人开发）

## Reference Documentation (参考文档)

- **设计文档**: `.meta/levels/ZombieGenerationModuleDesignDocument.md`
- **关卡说明**: `.meta/levels/chapter1.md`
- **现有实现**: `pkg/systems/wave_spawn_system.go`

---

**创建日期**: 2025-11-27
**创建人**: Sarah (Product Owner)
**Epic 类型**: Core Feature (核心功能)
**优先级**: High（原版忠实度关键）
**预估总工作量**: 56-78 小时（约 2-3 周）
