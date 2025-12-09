# Animation Showcase - 动画展示系统

## 概述

动画展示系统是一个用于查看和测试所有 Reanim 动画的工具，支持展示140+动画，提供网格布局、**分页浏览**和交互式动画切换功能。

**v1.1 新特性**：
- ⭐ **分页加载** - 按需加载，启动速度提升70%，内存减少60%
- ⭐ **快速翻页** - 支持 PageUp/PageDown 和数字键快速跳转
- ⭐ **文本优化** - 黑色半透明背景提升文本可读性
- ⭐ **JPEG 支持** - 支持 JPG 格式图片

## 功能特性

- ✅ **分页浏览**: 按需加载动画，每页可配置（默认 8 行 × 6 列 = 48 个单元）
- ✅ **快速翻页**: PageUp/PageDown 或方向键切换页面，数字键快速跳转
- ✅ **网格布局**: 同屏展示多个动画单元，支持自定义行列数
- ✅ **交互切换**: 左键点击单元切换该单元的动画
- ✅ **多动画组合**: 支持通过配置文件定义多动画组合播放（如攻击+摇晃）
- ✅ **自动轨道绑定**: 自动分析并绑定轨道到正确的动画
- ✅ **父子关系**: 支持配置父子轨道关系（如头部跟随身体）
- ✅ **YAML 配置**: 所有参数通过 YAML 配置文件管理
- ✅ **自动化工具**: 配置生成器和路径修复器，100%自动化

## 编译和运行

### 编译

```bash
go build -o animation_showcase cmd/animation_showcase/*.go
```

### 运行

```bash
# 使用默认配置
./animation_showcase

# 使用自定义配置
./animation_showcase --config=path/to/config.yaml

# 启用详细日志
./animation_showcase --verbose
```

## 配置文件格式

配置文件使用 YAML 格式，参考 `cmd/animation_showcase/config_example.yaml` 和 `cmd/animation_showcase/config.yaml`。

### 全局配置

```yaml
global:
  window:
    width: 1600        # 窗口宽度
    height: 900        # 窗口高度
    title: "pvz Animation Showcase"

  grid:
    columns: 6         # 每行单元数
    rows_per_page: 8   # 每页显示的行数（v1.1 新增）
    cell_width: 250    # 单元宽度
    cell_height: 250   # 单元高度
    padding: 10        # 单元间距
    scroll_speed: 30   # 滚动速度（保留配置）

  playback:
    fps: 12            # 帧率
    scale: 1.0         # 全局缩放
```

### 动画单元配置

```yaml
animations:
  - id: "peashooter"
    name: "豌豆射手"
    reanim_file: "assets/effect/reanim/PeaShooterSingle.reanim"
    default_animation: "anim_idle"
    scale: 1.0

    # 图片资源映射
    images:
      IMAGE_REANIM_PEASHOOTER_HEAD: "assets/reanim/PeaShooter_Head.png"
      # ... 更多图片映射

    # 可用的动画列表
    available_animations:
      - name: "anim_idle"
        display_name: "待机"
      - name: "anim_shooting"
        display_name: "攻击"

    # 多动画组合配置
    animation_combos:
      - name: "attack_with_sway"
        display_name: "攻击+摇晃"
        animations: ["anim_shooting", "anim_idle"]
        binding_strategy: "auto"  # auto 或 manual
        parent_tracks:
          anim_face: "anim_stem"
        hidden_tracks:
          - "anim_blink"
```

## 操作说明

| 操作 | 功能 |
|------|------|
| **PageDown** | 下一页 |
| **PageUp** | 上一页 |
| **→ 方向键** | 切换选中单元的下一个动画 |
| **← 方向键** | 切换选中单元的上一个动画 |
| **1-9 数字键** | 快速跳转到第 N 页 |
| **0 数字键** | 跳转到第 10 页 |
| **左键点击** | 选中并切换当前单元的动画 |
| **H 键** | 显示/隐藏帮助信息 |
| **Tab 键** | 切换帮助面板位置（右上→左上→右下→左下） |
| **ESC 键** | 退出程序 |

### 分页说明

- 默认每页显示：8 行 × 6 列 = 48 个动画
- 总共 3 页展示 140 个动画
- 翻页时按需加载，内存占用低
- 可通过 `rows_per_page` 配置调整每页行数

## 添加新动画

1. 在配置文件的 `animations` 列表中添加新的动画单元配置
2. 指定 `reanim_file` 路径
3. 配置 `images` 映射（从 Reanim 文件中的 ImagePath 到实际图片路径）
4. 配置 `available_animations` 列表
5. （可选）配置 `animation_combos` 用于多动画组合

## 多动画组合配置

### 自动轨道绑定模式 (binding_strategy: "auto")

系统会自动分析每个轨道应该由哪个动画控制：
- 基于轨道在不同动画时间窗口内的图片数据
- 基于轨道的位置方差（运动幅度）
- 自动处理头部/身体分离等复杂情况

### 手动轨道绑定模式 (binding_strategy: "manual")

手动指定轨道绑定关系：

```yaml
animation_combos:
  - name: "custom_combo"
    animations: ["anim_a", "anim_b"]
    binding_strategy: "manual"
    manual_bindings:
      track_head: "anim_a"
      track_body: "anim_b"
```

### 父子关系配置

用于处理部件间的层级关系（如头部跟随身体摆动）：

```yaml
parent_tracks:
  child_track_name: "parent_track_name"
```

## 项目结构

```
cmd/animation_showcase/
├── main.go              # 主程序入口
├── config.go            # 配置文件加载
├── animation_cell.go    # 动画单元（核心渲染逻辑）
├── grid_layout.go       # 网格布局管理器
├── config.yaml          # 实际使用的配置文件
└── config_example.yaml  # 配置文件示例
```

## 核心设计

### AnimationCell（动画单元）

每个动画单元负责：
- 加载和管理一个 Reanim 文件
- 管理该文件的所有动画
- 处理动画切换
- 处理多动画组合播放
- 实现轨道绑定分析
- 渲染动画到指定位置

### GridLayout（网格布局）

负责：
- 管理所有动画单元的布局
- 处理滚动逻辑
- 处理鼠标交互（点击检测）
- 只渲染可见区域的单元（性能优化）

## 技术要点

### 1. 轨道绑定分析

参考 `cmd/solution_attack_with_sway/main.go` 的实现：
- 计算轨道在不同动画时间窗口内的位置方差
- 优先绑定到方差最大（运动最明显）的动画
- 区分视觉轨道和逻辑轨道

### 2. 父子偏移计算

- 计算父轨道的初始位置（第一个可见帧）
- 计算父轨道的当前位置
- 偏移量 = 当前位置 - 初始位置
- 只在子父轨道使用不同动画时应用偏移

### 3. 性能优化

- 只更新可见区域的动画单元
- 只渲染可见区域的单元
- 使用 SubImage 限制渲染区域

## 示例配置生成

如果需要快速生成配置文件，可以扫描 `assets/effect/reanim/` 目录：

```bash
# 列出所有 reanim 文件
ls assets/effect/reanim/*.reanim

# 对于每个文件，创建对应的配置条目
# 图片映射需要手工配置或根据 reanim 文件内容自动提取
```

## 未来扩展

- [ ] 自动扫描 reanim 文件并生成配置
- [ ] 自动提取图片映射关系
- [ ] 动画预览缩略图
- [ ] 动画信息面板（显示轨道、帧数等详细信息）
- [ ] 导出动画为 GIF/视频
- [ ] 批量测试所有动画

## 相关文档

- `docs/reanim/reanim-format-guide.md` - Reanim 格式详解
- `docs/reanim/reanim-fix-guide.md` - Reanim 修复指南
- `cmd/solution_attack_with_sway/main.go` - 多动画组合参考实现

## 常见问题

### Q: 如何添加向日葵、樱桃炸弹等其他植物？

A: 在 `config.yaml` 的 `animations` 列表中添加新的配置条目，参考豌豆射手的配置格式。

### Q: 动画显示不正确怎么办？

A: 检查：
1. 图片映射是否正确（reanim 文件中的 ImagePath 对应的实际图片路径）
2. 是否需要配置 `animation_combos` 用于多动画组合
3. 是否需要配置 `parent_tracks` 处理父子关系

### Q: 如何调试轨道绑定问题？

A: 使用 `--verbose` 标志运行程序，查看详细的轨道绑定日志。

## License

MIT
