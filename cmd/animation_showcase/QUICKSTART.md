# 快速开始指南

## 第一步：生成配置文件

使用配置生成工具自动扫描所有 reanim 文件：

```bash
go run cmd/animation_showcase/generate_config.go > cmd/animation_showcase/config_all.yaml
```

这会生成包含所有动画的配置模板。

## 第二步：自动修复图片路径

使用图片路径修复脚本自动替换所有图片路径：

```bash
go run cmd/animation_showcase/fix_image_paths.go cmd/animation_showcase/config_all.yaml > cmd/animation_showcase/config_fixed.yaml
```

这会自动：
- 扫描 `assets/reanim` 目录的所有图片（png 和 jpg）
- 匹配配置文件中的图片引用
- 替换为正确的图片路径
- 支持特殊字符和空格
- **100% 自动化（无需手工配置）**

## 第三步：编译并运行

```bash
# 正确的编译命令（在项目根目录执行）
cd /path/to/pvz3
go build -o animation_showcase ./cmd/animation_showcase

# 运行（使用默认配置 - 只有豌豆射手）
./animation_showcase --config=cmd/animation_showcase/config.yaml

# 运行（使用完整配置 - 所有140个动画，分3页显示）
./animation_showcase --config=cmd/animation_showcase/config_fixed.yaml --verbose
```

**注意**：
- 必须从项目根目录运行 `go build`
- 使用 `./cmd/animation_showcase` 作为包路径
- build tag 会自动排除工具文件（generate_config.go 和 fix_image_paths.go）

## 第四步：使用分页功能 ⭐ NEW

### 操作说明

| 操作 | 功能 |
|------|------|
| **PageDown** | 下一页 |
| **PageUp** | 上一页 |
| **→ 方向键** | 切换选中单元的下一个动画 |
| **← 方向键** | 切换选中单元的上一个动画 |
| **1-9 数字键** | 快速跳转到第 N 页 |
| **0 数字键** | 跳转到第 10 页 |
| **左键点击** | 选中并切换当前单元的动画 |
| **H 键** | 显示/隐藏帮助 |
| **Tab 键** | 切换帮助位置（四角循环） |
| **ESC** | 退出 |

### 分页说明

默认配置：
- **每页显示**: 8 行 × 6 列 = 48 个动画
- **总页数**: 3 页（140 个动画）
- **加载方式**: 按需加载，节省内存和启动时间

启动日志示例：
```
✓ 加载配置成功: 140 个动画单元
✓ 分页配置: 每页 8 行 × 6 列 = 48 个单元, 共 3 页
=== 加载第 1/3 页 ===
✓ 成功加载 48 个动画单元
```

## 一键完成所有步骤

```bash
# 生成配置 -> 修复路径 -> 编译 -> 运行
go run cmd/animation_showcase/generate_config.go 2>/dev/null | \
go run cmd/animation_showcase/fix_image_paths.go /dev/stdin > cmd/animation_showcase/config_fixed.yaml && \
go build -o animation_showcase ./cmd/animation_showcase && \
./animation_showcase --config=cmd/animation_showcase/config_fixed.yaml
```

## 自定义配置

### 调整每页显示的动画数量

编辑 `config_fixed.yaml`：

```yaml
global:
  grid:
    columns: 6           # 每行显示的单元数
    rows_per_page: 8     # 每页显示的行数（可调整）
    cell_width: 250
    cell_height: 250
```

**推荐配置**：

| 屏幕分辨率 | columns | rows_per_page | 每页单元数 |
|-----------|---------|---------------|----------|
| 1920x1080 | 6 | 8 | 48 |
| 1600x900  | 6 | 6 | 36 |
| 1366x768  | 5 | 5 | 25 |
| 2560x1440 | 8 | 10 | 80 |

### 示例：高性能模式（小页面）

```yaml
global:
  grid:
    columns: 4
    rows_per_page: 4  # 只加载 16 个单元
    cell_width: 300
    cell_height: 300
```

### 示例：全景模式（大页面）

```yaml
global:
  grid:
    columns: 8
    rows_per_page: 12  # 加载 96 个单元
    cell_width: 200
    cell_height: 200
```

## 性能优化说明 ⭐ NEW

**分页加载带来的优化**：
- ✅ 启动速度提升 70%（从 3-5 秒降至 <1 秒）
- ✅ 内存占用减少 60%（从 ~200MB 降至 ~80MB）
- ✅ 支持快速翻页（< 0.5 秒响应）

详见 `PERFORMANCE.md` 了解更多。

## 简化配置示例

如果只想测试几个动画，可以创建简化配置：

```yaml
# config_simple.yaml
global:
  window:
    width: 800
    height: 600
  grid:
    columns: 2
    rows_per_page: 2
    cell_width: 300
    cell_height: 300

animations:
  - id: "peashooter"
    name: "豌豆射手"
    reanim_file: "assets/effect/reanim/PeaShooterSingle.reanim"
    default_animation: "anim_idle"
    # ... 省略图片映射
```

## 测试多动画组合

在配置文件中添加 `animation_combos` 部分：

```yaml
animations:
  - id: "peashooter"
    # ... 基本配置

    available_animations:
      - name: "anim_idle"
        display_name: "待机"
      - name: "anim_shooting"
        display_name: "攻击"

    animation_combos:
      - name: "attack_with_sway"
        display_name: "攻击+摇晃"
        animations: ["anim_shooting", "anim_idle"]
        binding_strategy: "auto"
        parent_tracks:
          anim_face: "anim_stem"
        hidden_tracks:
          - "anim_blink"
```

点击该单元即可在不同动画间切换，包括组合动画。

## 常用命令

```bash
# 只编译（检查语法）
go build ./cmd/animation_showcase

# 查看所有 reanim 文件
ls assets/effect/reanim/*.reanim | wc -l

# 查找特定动画的图片
grep "IMAGE_REANIM" assets/effect/reanim/PeaShooterSingle.reanim

# 生成完整配置
go run cmd/animation_showcase/generate_config.go > config_full.yaml

# 修复图片路径
go run cmd/animation_showcase/fix_image_paths.go config_full.yaml > config_fixed.yaml
```

## 故障排除

### 问题：动画不显示

**解决方案**：
1. 检查图片路径是否正确
2. 使用 `--verbose` 查看详细日志
3. 确认 reanim 文件路径正确
4. 检查是否添加了 JPEG 支持（已在 v1.1 版本中添加）

### 问题：动画显示不正确

**解决方案**：
1. 检查是否需要配置 `animation_combos`
2. 检查是否需要配置 `parent_tracks`
3. 参考 `cmd/solution_attack_with_sway/main.go` 的实现

### 问题：编译错误

**解决方案**：
1. 确保使用正确的编译命令（不要包含工具文件）
2. 检查 Go 版本（需要 Go 1.18+）
3. 运行 `go mod tidy` 更新依赖

### 问题：翻页没有反应

**解决方案**：
1. 确认总页数 > 1（查看启动日志）
2. 检查是否在窗口获得焦点状态
3. 使用数字键尝试跳转页面

### 问题：文本看不清 ⭐ FIXED

**解决方案**：
- 已在 v1.1 版本中优化
- 文本现在显示在黑色半透明背景上，清晰度大幅提升

## 下一步

- 阅读 `README.md` 了解详细功能
- 参考 `PERFORMANCE.md` 了解性能优化细节
- 查看 `cmd/solution_attack_with_sway/main.go` 学习多动画组合原理
- 阅读 `PROJECT_SUMMARY.md` 了解项目整体架构
