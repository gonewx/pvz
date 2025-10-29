# 📋 Sprint Change Proposal
## 关卡 1-3 和 1-4 配置调整

**日期**: 2025-10-28
**提议人**: Bob (Scrum Master)
**状态**: ✅ 已批准

---

## 1. 变更概述 (Issue Summary)

**触发器**: `.meta/levels/chapter1.md` 文件中关卡描述更新

**核心问题**:
关卡 1-3 和 1-4 的当前实现配置与更新后的关卡描述不匹配：

| 关卡 | 当前配置 | 更新后的描述 | 差异 |
|------|----------|--------------|------|
| **1-3** | 5行草地，2面旗帜 | **3行草地，1面旗帜** | ❌ 行数不匹配，旗帜数不匹配 |
| **1-4** | 5行草地，2面旗帜 | **5行草地，1面旗帜** | ⚠️ 旗帜数不匹配 |

**问题类型**: 新发现的需求 / 基于设计文档更新的必要调整

**即时影响**:
- 关卡难度递进不符合设计意图
- 1-3 关卡过早引入 5行全场作战
- 波次配置与旗帜数不匹配

---

## 2. Epic/Story 影响分析

### ✅ 当前 Epic 状态
**Epic 8: Chapter 1 Level Implementation** - 可以继续，但需要修正关卡配置

### 📝 受影响的 Stories
- **Story 8.1** (关卡配置增强): 需要更新 level-1-3.yaml 和 level-1-4.yaml
- **其他 Stories**: 无影响（变更仅限于数据配置）

### 🔮 未来 Epics
- **无影响**: 此变更不影响后续 Epic 的规划

---

## 3. 文档与工件冲突分析

### 📄 需要更新的文档

| 文档 | 影响 | 所需变更 |
|------|------|----------|
| **`.meta/levels/chapter1.md`** | ✅ 已更新 | 无需变更（已是最新） |
| **`data/levels/level-1-3.yaml`** | ❌ 过时 | 需要修改配置 |
| **`data/levels/level-1-4.yaml`** | ❌ 过时 | 需要修改配置 |

### 🏗️ 架构影响
- **无架构变更**: 此变更仅涉及配置数据，不影响代码架构

---

## 4. 推荐路径 (Recommended Path Forward)

### ✅ **选择方案：直接调整配置**

**理由**:
1. **影响范围极小**: 只需修改 2 个 YAML 配置文件
2. **无代码变更**: 不涉及任何逻辑代码修改
3. **零技术风险**: 配置调整不会引入新bug
4. **工作量极低**: 预计 15 分钟内完成

---

## 5. 具体修改建议 (Proposed Edits)

### 📝 修改 1: `data/levels/level-1-3.yaml`

#### 变更 A: 场地布局（第7行）
```yaml
# 修改前
enabledLanes: [1, 2, 3, 4, 5] # 全部5行草地

# 修改后
enabledLanes: [2, 3, 4]       # 中间3行草地
```

#### 变更 B: 描述（第3行）
```yaml
# 修改前
description: "全行作战，引入路障僵尸"

# 修改后
description: "精英怪和范围爆发伤害，引入路障僵尸"
```

#### 变更 C: 草皮配置（第21-32行）
```yaml
# 修改前
# 草皮配置（全行）
# 关卡 1-3: 延续上一关场景，第3行草皮已铺好，其他行需要铺草皮
backgroundImage: "IMAGE_BACKGROUND1UNSODDED"  # 使用未铺草皮背景
sodRowImage: "IMAGE_SOD1ROW"                  # 使用单行草地图片（用于动画）
showSoddingAnim: true                         # 播放铺草皮动画
soddingAnimDelay: 0.0                         # 开场动画完成后立即播放
preSoddedLanes: [3]                           # 预先渲染第3行草皮（初始化时直接显示）

# Story 11.4：铺草皮粒子特效配置
sodRollAnimation: true                        # 启用铺草皮动画
sodRollParticles: true                        # 启用土粒飞溅特效
soddingAnimLanes: [1, 2, 4, 5]                # 在第1,2,4,5行播放动画（第3行跳过）

# 修改后
# 草皮配置（三行）
# 关卡 1-3: 延续上一关场景，第3行草皮已铺好，只铺第2/4行
backgroundImage: "IMAGE_BACKGROUND1UNSODDED"  # 使用未铺草皮背景
sodRowImage: "IMAGE_SOD3ROW"                  # 3行草地图片
showSoddingAnim: true                         # 播放铺草皮动画
soddingAnimDelay: 0.0                         # 开场动画完成后立即播放
preSoddedLanes: [3]                           # 预先渲染第3行草皮（初始化时直接显示）

# Story 11.4：铺草皮粒子特效配置
sodRollAnimation: true                        # 启用铺草皮动画
sodRollParticles: true                        # 启用土粒飞溅特效
soddingAnimLanes: [2, 4]                      # 只在第2/4行播放动画（第3行跳过）
```

#### 变更 D: 波次配置（完全重写，简化为1面旗帜）

**删除**: 第34行到文件结尾的所有现有波次配置

**替换为**:
```yaml
# 波次配置：1面旗帜（2个小波次 + 1个旗帜波）
waves:
  # 第1波：游戏开始后20秒，3个僵尸（含1个路障）
  - delay: 20
    zombies:
      - type: "basic"
        lanes: [2, 3, 4]  # 中间3行
        count: 2
        spawnInterval: 2.0
      - type: "conehead"  # 路障僵尸（新增）
        lanes: [2, 3, 4]
        count: 1
        spawnInterval: 0

  # 第2波：第1波消灭后15秒，4个僵尸（含1个路障）
  - minDelay: 15
    zombies:
      - type: "basic"
        lanes: [2, 3, 4]
        count: 3
        spawnInterval: 2.0
      - type: "conehead"
        lanes: [2, 3, 4]
        count: 1
        spawnInterval: 0

  # 旗帜波（第3波）：进度条80%位置，6个僵尸（含2个路障）
  - isFlag: true
    flagIndex: 1  # 第1面旗帜
    minDelay: 5
    zombies:
      - type: "basic"
        lanes: [2, 3, 4]
        count: 4
        spawnInterval: 1.5
      - type: "conehead"
        lanes: [2, 3, 4]
        count: 2
        spawnInterval: 1.2
```

---

### 📝 修改 2: `data/levels/level-1-4.yaml`

#### 变更 A: 描述（第3行）
```yaml
# 修改前
description: "综合教学测试，引入铁桶僵尸"

# 修改后
description: "防御型植物概念，完整5行战场"
```

#### 变更 B: 奖励配置（第16-19行）
```yaml
# 修改前
# Story 8.3: 奖励配置
rewardPlant: "potatomine"     # 完成后解锁土豆雷

# 修改后
# Story 8.3: 奖励配置
rewardPlant: ""               # 无奖励植物

# Story 8.6: 工具解锁
unlockTools: ["shovel"]       # 完成后解锁铲子工具
```

#### 变更 C: 草皮配置（第19-22行，完全重写）

**删除**:
```yaml
# 草皮配置（全行）
backgroundImage: "IMAGE_BACKGROUND1"  # 使用标准背景
sodRowImage: "IMAGE_SOD1ROW"          # 全行草地图片
```

**替换为**:
```yaml
# 草皮配置（全行）
# 关卡 1-4: 延续上一关场景，第3行草皮已铺好，其他行需要铺草皮
backgroundImage: "IMAGE_BACKGROUND1UNSODDED"  # 使用未铺草皮背景
sodRowImage: "IMAGE_SOD1ROW"                  # 使用单行草地图片（用于动画）
showSoddingAnim: true                         # 播放铺草皮动画
soddingAnimDelay: 0.0                         # 开场动画完成后立即播放
preSoddedLanes: [3]                           # 预先渲染第3行草皮（初始化时直接显示）

# Story 11.4：铺草皮粒子特效配置
sodRollAnimation: true                        # 启用铺草皮动画
sodRollParticles: true                        # 启用土粒飞溅特效
soddingAnimLanes: [1, 2, 4, 5]                # 在第1,2,4,5行播放动画（第3行跳过）
```

#### 变更 D: 波次配置（完全重写，简化为1面旗帜，移除铁桶僵尸）

**删除**: 第24行到文件结尾的所有现有波次配置

**替换为**:
```yaml
# 波次配置：1面旗帜（2个小波次 + 1个旗帜波）
waves:
  # 第1波：游戏开始后20秒，4个僵尸（含1个路障）
  - delay: 20
    zombies:
      - type: "basic"
        lanes: [1, 2, 3, 4, 5]
        count: 3
        spawnInterval: 2.0
      - type: "conehead"
        lanes: [1, 2, 3, 4, 5]
        count: 1
        spawnInterval: 0

  # 第2波：第1波消灭后15秒，5个僵尸（含2个路障）
  - minDelay: 15
    zombies:
      - type: "basic"
        lanes: [1, 2, 3, 4, 5]
        count: 3
        spawnInterval: 2.0
      - type: "conehead"
        lanes: [1, 2, 3, 4, 5]
        count: 2
        spawnInterval: 1.5

  # 旗帜波（第3波）：进度条80%位置，7个僵尸（含2个路障）
  - isFlag: true
    flagIndex: 1  # 第1面旗帜
    minDelay: 5
    zombies:
      - type: "basic"
        lanes: [1, 2, 3, 4, 5]
        count: 5
        spawnInterval: 1.2
      - type: "conehead"
        lanes: [1, 2, 3, 4, 5]
        count: 2
        spawnInterval: 1.5
```

---

## 6. 验证步骤

执行以下命令验证修改：

```bash
# 验证关卡 1-3（应该是3行草地，1面旗帜）
go run main.go --level 1-3 --verbose

# 验证关卡 1-4（应该是5行草地，1面旗帜）
go run main.go --level 1-4 --verbose
```

**检查点**:
- ✅ 关卡 1-3 只有中间3行可种植
- ✅ 关卡 1-3 草皮动画在第2,4行播放
- ✅ 关卡 1-3 只有1个旗帜波
- ✅ 关卡 1-3 出现路障僵尸
- ✅ 关卡 1-4 全部5行可种植
- ✅ 关卡 1-4 草皮动画在第1,2,4,5行播放
- ✅ 关卡 1-4 只有1个旗帜波
- ✅ 关卡 1-4 没有铁桶僵尸

---

## 7. 成功标准

✅ **配置验证**:
- level-1-3.yaml 反映了 3行草地 + 1面旗帜
- level-1-4.yaml 反映了 5行草地 + 1面旗帜

✅ **功能验证**:
- 关卡难度递进合理：1-1 (1行) → 1-2 (3行) → 1-3 (3行) → 1-4 (5行)
- 草皮动画在两个关卡中正常播放
- 旗帜数量正确

---

## 8. 风险评估

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| 配置语法错误 | 🟡 中 | 🟢 低 | YAML 语法检查，运行验证 |
| 波次平衡 | 🟡 中 | 🟡 中 | 保留调整空间 |
| 草皮渲染 | 🟢 低 | 🟢 低 | 代码已支持多行配置 |

**总体风险**: 🟢 **低风险**

---

## 9. 审批记录

- **提案日期**: 2025-10-28
- **审批人**: 用户
- **审批状态**: ✅ 已批准
- **预计完成时间**: 15 分钟

---

## 📋 附录：变更检查清单

- [x] 理解触发器与上下文
- [x] Epic 影响评估
- [x] 文档冲突分析
- [x] 路径评估
- [x] Sprint Change Proposal 完成
- [x] 用户审批
