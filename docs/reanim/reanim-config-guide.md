# Reanim 配置指南 (Reanim Configuration Guide)

本文档详细说明 Reanim 动画系统的 YAML 配置格式，用于声明式定义动画组合、轨道绑定和父子关系。

---

## 目录

1. [配置文件概述](#配置文件概述)
2. [配置文件格式](#配置文件格式)
3. [字段详细说明](#字段详细说明)
4. [配置示例](#配置示例)
5. [最佳实践](#最佳实践)
6. [常见问题](#常见问题)

---

## 配置文件概述

### 为什么使用配置文件？

**问题**：之前动画配置硬编码在代码中，修改困难，维护成本高。

**解决方案**：使用 YAML 配置文件声明动画组合，实现配置与代码分离。

**优势**：
- ✅ 修改配置无需重新编译代码
- ✅ 配置易读易写，非程序员也能理解
- ✅ 支持版本控制，可追踪配置历史
- ✅ 降低维护成本，提高开发效率

### 配置文件位置

所有 Reanim 配置文件存放在：
```
data/reanim_configs/
├── peashooter.yaml      # 豌豆射手配置
├── sunflower.yaml       # 向日葵配置
└── zombie.yaml          # 僵尸配置
```

**命名规范**：`{实体名}.yaml`（使用小写字母和下划线）

---

## 配置文件格式

### YAML Schema

```yaml
# Reanim 文件路径（必填）
reanim_file: string

# 动画列表（可选，用于文档和验证）
animations:
  - name: string           # 动画名称
    display_name: string   # 显示名称（可选）

# 动画组合配置（必填）
animation_combos:
  - name: string                   # 组合名称（必填）
    display_name: string           # 显示名称（可选）
    animations: [string]           # 动画列表（必填）
    binding_strategy: string       # 绑定策略（可选："auto" 或 "manual"）
    manual_bindings:               # 手动绑定（可选）
      track_name: animation_name
    parent_tracks:                 # 父子关系（可选）
      child_track: parent_track
    hidden_tracks: [string]        # 隐藏轨道（可选）
```

### Go 数据结构映射

```go
type ReanimConfig struct {
    ReanimFile      string                   `yaml:"reanim_file"`
    Animations      []AnimationDef           `yaml:"animations,omitempty"`
    AnimationCombos []AnimationComboConfig   `yaml:"animation_combos"`
}

type AnimationDef struct {
    Name        string `yaml:"name"`
    DisplayName string `yaml:"display_name,omitempty"`
}

type AnimationComboConfig struct {
    Name             string            `yaml:"name"`
    DisplayName      string            `yaml:"display_name,omitempty"`
    Animations       []string          `yaml:"animations"`
    BindingStrategy  string            `yaml:"binding_strategy,omitempty"`
    ManualBindings   map[string]string `yaml:"manual_bindings,omitempty"`
    ParentTracks     map[string]string `yaml:"parent_tracks,omitempty"`
    HiddenTracks     []string          `yaml:"hidden_tracks,omitempty"`
}
```

---

## 字段详细说明

### 顶层字段

#### `reanim_file` (必填)

- **类型**: `string`
- **说明**: Reanim 动画文件的路径（相对于项目根目录）
- **示例**: `assets/effect/reanim/PeaShooterSingle.reanim`

#### `animations` (可选)

- **类型**: `[]AnimationDef`
- **说明**: 动画列表，用于文档说明和配置验证
- **用途**:
  - 提供动画的可读描述（`display_name`）
  - 验证 `animation_combos` 中的动画引用是否有效
- **建议**: 推荐填写，方便理解和维护

#### `animation_combos` (必填)

- **类型**: `[]AnimationComboConfig`
- **说明**: 动画组合配置列表，定义如何组合多个动画

---

### AnimationDef 字段

#### `name` (必填)

- **类型**: `string`
- **说明**: 动画名称，必须与 Reanim 文件中的动画名称一致
- **示例**: `anim_idle`, `anim_shooting`, `anim_head_idle`

#### `display_name` (可选)

- **类型**: `string`
- **说明**: 动画的显示名称（中文描述）
- **示例**: `待机`, `攻击`, `头部摇晃`

---

### AnimationComboConfig 字段

#### `name` (必填)

- **类型**: `string`
- **说明**: 动画组合的唯一标识符，代码中使用此名称引用组合
- **示例**: `idle`, `attack_with_sway`

#### `display_name` (可选)

- **类型**: `string`
- **说明**: 组合的显示名称（用于调试和日志）
- **示例**: `待机`, `攻击+摇晃`

#### `animations` (必填)

- **类型**: `[]string`
- **说明**: 需要同时播放的动画名称列表
- **示例**: `["anim_shooting", "anim_head_idle"]`
- **注意**: 列表中的动画会同时播放，系统会自动处理轨道分配

#### `binding_strategy` (可选)

- **类型**: `string`
- **可选值**: `"auto"` 或 `"manual"`
- **默认值**: `"auto"`
- **说明**:
  - `auto`: 系统自动分析每个轨道应该绑定到哪个动画（推荐）
  - `manual`: 手动指定轨道绑定关系
- **何时使用 manual**:
  - 自动绑定结果不理想时
  - 需要精细控制特定轨道的动画时

#### `manual_bindings` (可选)

- **类型**: `map[string]string`
- **说明**: 手动指定轨道到动画的绑定关系
- **格式**: `轨道名: 动画名`
- **示例**:
  ```yaml
  manual_bindings:
    anim_face: anim_head_idle      # 头部轨道使用头部动画
    stalk_bottom: anim_shooting    # 身体轨道使用攻击动画
  ```
- **注意**: 只在 `binding_strategy: manual` 时有效

#### `parent_tracks` (可选)

- **类型**: `map[string]string`
- **说明**: 定义轨道的父子关系，子轨道会继承父轨道的位置偏移
- **格式**: `子轨道名: 父轨道名`
- **示例**:
  ```yaml
  parent_tracks:
    anim_face: anim_stem    # anim_face 的父轨道是 anim_stem
  ```
- **效果**: 子轨道会随父轨道一起运动（如头部随身体摆动）

#### `hidden_tracks` (可选)

- **类型**: `[]string`
- **说明**: 需要隐藏的轨道列表
- **示例**: `["anim_blink", "anim_shadow"]`
- **用途**:
  - 隐藏不需要显示的特效轨道
  - 根据游戏状态动态控制部件显示（如僵尸装备）

---

## 配置示例

### 示例 1：豌豆射手 - 攻击组合

```yaml
# data/reanim_configs/peashooter.yaml

# Reanim 文件路径
reanim_file: assets/effect/reanim/PeaShooterSingle.reanim

# 动画列表（用于文档和验证）
animations:
  - name: anim_idle
    display_name: 待机
  - name: anim_shooting
    display_name: 攻击
  - name: anim_head_idle
    display_name: 头部摇晃

# 动画组合配置
animation_combos:
  # 待机动画
  - name: idle
    display_name: 待机
    animations:
      - anim_idle
    binding_strategy: auto

  # 攻击+头部摇晃组合
  - name: attack_with_sway
    display_name: 攻击+摇晃
    animations:
      - anim_shooting
      - anim_head_idle

    # 使用自动绑定（推荐）
    binding_strategy: auto

    # 定义父子关系
    parent_tracks:
      anim_face: anim_stem    # 头部跟随茎干运动
```

### 示例 2：向日葵 - 简单配置

```yaml
# data/reanim_configs/sunflower.yaml

reanim_file: assets/effect/reanim/Sunflower.reanim

animations:
  - name: anim_idle
    display_name: 待机

animation_combos:
  - name: idle
    display_name: 待机
    animations:
      - anim_idle
    binding_strategy: auto
```

### 示例 3：僵尸 - 手动绑定 + 隐藏轨道

```yaml
# data/reanim_configs/zombie.yaml

reanim_file: assets/effect/reanim/Zombie.reanim

animations:
  - name: anim_walk
    display_name: 行走
  - name: anim_attack
    display_name: 攻击
  - name: anim_head_blink
    display_name: 眨眼

animation_combos:
  # 行走动画
  - name: walk
    display_name: 行走
    animations:
      - anim_walk
    binding_strategy: auto

  # 攻击动画（手动绑定）
  - name: attack_with_blink
    display_name: 攻击+眨眼
    animations:
      - anim_attack
      - anim_head_blink

    # 使用手动绑定
    binding_strategy: manual
    manual_bindings:
      Zombie_head: anim_head_blink     # 头部用眨眼动画
      Zombie_body: anim_attack         # 身体用攻击动画
      Zombie_outerarm: anim_attack     # 手臂用攻击动画

    # 隐藏路障（初始状态）
    hidden_tracks:
      - anim_cone    # 隐藏路障装备
```

---

## 最佳实践

### 1. 优先使用自动绑定

**推荐**：
```yaml
binding_strategy: auto
```

**原因**：
- 系统会自动分析每个轨道的运动特征
- 减少配置复杂度
- 大多数情况下效果良好

**何时使用手动绑定**：
- 自动绑定结果不符合预期
- 需要精细控制特定轨道

### 2. 添加详细注释

**推荐**：
```yaml
animation_combos:
  - name: attack_with_sway
    display_name: 攻击+摇晃
    animations:
      - anim_shooting    # 身体攻击动画
      - anim_head_idle   # 头部摇晃动画

    # 头部需要跟随茎干运动
    parent_tracks:
      anim_face: anim_stem
```

**好处**：
- 方便团队成员理解配置
- 便于后续维护和调整

### 3. 使用有意义的命名

**推荐**：
- 组合名称：`attack_with_sway`, `walk_with_fire`
- 显示名称：`攻击+摇晃`, `行走+燃烧`

**避免**：
- 组合名称：`combo1`, `test`

### 4. 填写 animations 列表

**推荐**：
```yaml
animations:
  - name: anim_idle
    display_name: 待机
  - name: anim_shooting
    display_name: 攻击
```

**好处**：
- 提供动画的可读描述
- 启用配置验证（检测无效引用）
- 方便文档生成

### 5. 测试配置变更

**流程**：
1. 修改配置文件
2. 运行游戏（无需重新编译）
3. 观察动画效果
4. 调整配置直到满意
5. 提交配置到版本控制

---

## 常见问题

### Q1: 如何调试配置加载失败？

**答**: 配置加载失败会返回详细的错误信息：

```
配置文件 data/reanim_configs/peashooter.yaml 验证失败:
动画组合 'attack' 引用了不存在的动画 'anim_shoot'
```

**常见错误**：
- 缺少必填字段 `reanim_file`
- 动画名称拼写错误
- 引用了不存在的动画
- 绑定策略拼写错误（只能是 `auto` 或 `manual`）

### Q2: 配置修改后需要重新编译吗？

**答**: **不需要**。配置文件在运行时加载，修改配置后直接运行游戏即可生效。

### Q3: 如何知道 Reanim 文件中有哪些动画？

**答**: 使用工具查看 Reanim 文件内容：

```bash
# 使用 animation_showcase 查看所有动画
go run cmd/animation_showcase/main.go --reanim=assets/effect/reanim/PeaShooterSingle.reanim
```

### Q4: 自动绑定和手动绑定有什么区别？

**答**:

| 特性 | 自动绑定 (auto) | 手动绑定 (manual) |
|------|----------------|------------------|
| 配置复杂度 | 低 | 高 |
| 灵活性 | 中等 | 高 |
| 适用场景 | 大多数情况 | 需要精细控制 |
| 工作原理 | 系统自动分析轨道运动特征 | 手动指定每个轨道的动画 |

**推荐**: 优先使用自动绑定，只有在效果不理想时才使用手动绑定。

### Q5: 如何隐藏特定轨道？

**答**: 使用 `hidden_tracks` 字段：

```yaml
animation_combos:
  - name: attack
    animations:
      - anim_attack
    hidden_tracks:
      - anim_blink      # 隐藏眨眼轨道
      - anim_shadow     # 隐藏阴影轨道
```

### Q6: 父子关系配置有什么作用？

**答**: 父子关系使子轨道跟随父轨道运动。

**示例**：豌豆射手的头部（`anim_face`）需要跟随茎干（`anim_stem`）摆动：

```yaml
parent_tracks:
  anim_face: anim_stem    # 头部跟随茎干
```

**效果**：当茎干摆动时，头部会同步移动，动画更自然。

### Q7: 可以在一个配置文件中定义多个组合吗？

**答**: **可以**。一个配置文件可以定义多个动画组合：

```yaml
animation_combos:
  - name: idle
    animations: [anim_idle]

  - name: attack
    animations: [anim_shooting, anim_head_idle]

  - name: victory
    animations: [anim_celebrate]
```

### Q8: 配置文件支持注释吗？

**答**: **支持**。YAML 格式支持 `#` 注释：

```yaml
# 这是注释
reanim_file: assets/effect/reanim/Test.reanim

animations:
  - name: anim_idle    # 待机动画
```

---

## 下一步

- 查看更多示例配置：`data/reanim_configs/`
- 了解 Reanim 系统 API：`CLAUDE.md` - Reanim 动画系统使用指南
- 查看 Reanim 格式说明：`docs/reanim/reanim-format-guide.md`

---

**版本**: v1.0
**更新日期**: 2025-11-07
**维护者**: Bob (SM)
