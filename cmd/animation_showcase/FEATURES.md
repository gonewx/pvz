# Animation Showcase 新功能说明

## 概述

本次更新为 animation_showcase 增加了三个主要功能：
1. JPG+PNG 蒙版图片处理
2. Cell 单独展示模式
3. 部件轨道显示/隐藏快捷键

---

## 功能 1: JPG+PNG 蒙版图片处理

### 功能描述
对于 JPG 格式的图片，系统会自动检查是否存在同名的以 `_` 结尾的 PNG 文件作为 Alpha 蒙版。如果存在，会将蒙版应用到 JPG 图片上。

### 实现细节
- **蒙版文件命名规则**: 例如 `image.jpg` 对应的蒙版文件为 `image_.png`
- **蒙版格式**: PNG 文件是 8-bit 调色板模式
- **处理方式**:
  - 白色 (255) = 不透明
  - 黑色 (0) = 透明
  - 灰度值用于抗锯齿
  - 使用预乘 alpha 处理改善边缘质量

### 使用方法
只需将图片和对应的蒙版文件放在同一目录下，系统会自动检测并应用。

### 技术实现
为了避免 Ebitengine 的限制（不能在游戏启动前访问 `ebiten.Image` 像素），本功能使用以下流程：
1. 使用 Go 标准库 `image` 包加载 JPG 和 PNG
2. 在标准库的 `image.Image` 上应用蒙版处理
3. 处理完成后转换为 `ebiten.Image`

**关键点**：全程使用标准库处理，避免在游戏启动前调用 `ebiten.Image.At()` 导致 panic。

### 代码位置
- `cmd/animation_showcase/animation_cell.go`
  - `loadImageWithMask()` - 图片加载入口（第 822-885 行）
  - `applyAlphaMask()` - 蒙版应用逻辑（第 887-932 行）

---

## 功能 2: Cell 单独展示模式

### 功能描述
可以将选中的动画单元以全屏模式单独展示，方便详细查看动画细节。

### 使用方法

#### 网格模式（默认）
1. 使用鼠标点击或方向键选择一个单元
2. 按 **Enter** 键进入单个展示模式

#### 单个展示模式
- 动画会在屏幕中央放大显示
- 按 **Enter** 键返回网格模式
- 按 **←/→** 方向键切换动画

### 操作快捷键

| 模式 | 按键 | 功能 |
|------|------|------|
| 网格模式 | Enter | 切换到单个模式 |
| 单个模式 | Enter | 返回网格模式 |
| 通用 | H | 显示/隐藏帮助 |
| 通用 | Tab | 切换帮助位置 |

### 代码位置
- `cmd/animation_showcase/main.go`
  - `DisplayMode` 枚举 - 模式定义
  - `drawSingleCell()` - 单个模式渲染
  - Enter 键处理逻辑

---

## 功能 3: 部件轨道显示/隐藏快捷键

### 功能描述
在单个展示模式下，可以通过快捷键控制每个部件轨道的显示/隐藏，方便调试和理解动画结构。

### 使用方法

#### 进入轨道编辑模式
1. 在网格模式下选择一个单元
2. 按 **Enter** 进入单个展示模式
3. 屏幕左侧会显示轨道列表

#### 轨道列表说明
```
轨道列表 (F1-F12 切换, R 重置):
F1  ✓ track_name_1
F2  ✗ track_name_2
F3  ✓ track_name_3
...
```

- **F 键标签**: 对应的快捷键
- **✓ (绿色)**: 轨道可见
- **✗ (红色)**: 轨道隐藏
- **轨道名称**: 部件轨道的名称

### 操作快捷键

| 按键 | 功能 |
|------|------|
| F1-F12 | 切换对应轨道的显示/隐藏 |
| R | 重置所有轨道为可见 |
| ←/→ | 切换动画 |
| Enter | 返回网格模式 |

### 应用场景
- **调试动画**: 逐个查看每个部件的运动轨迹
- **理解结构**: 了解动画由哪些部件组成
- **问题排查**: 找出哪个部件导致渲染异常

### 代码位置
- `cmd/animation_showcase/animation_cell.go`
  - `manualHiddenTracks` 字段 - 手动隐藏的轨道
  - `GetVisualTracks()` - 获取轨道列表
  - `ToggleTrackVisibility()` - 切换轨道可见性
  - `IsTrackVisible()` - 检查轨道状态
  - `ResetTrackVisibility()` - 重置轨道状态
- `cmd/animation_showcase/main.go`
  - `handleTrackToggle()` - 快捷键处理
  - `drawTrackList()` - 轨道列表渲染

---

## 完整操作说明

### 网格模式操作
```
PageDown/PageUp  - 翻页
1-9 数字键       - 快速跳转页面
←/→ 方向键       - 切换选中单元的动画
左键点击         - 选中并切换动画
Enter           - 切换到单个模式
H               - 显示/隐藏帮助
Tab             - 切换帮助位置
ESC             - 退出
```

### 单个模式操作
```
←/→ 方向键       - 切换动画
F1-F12          - 切换轨道显示/隐藏
R               - 重置所有轨道可见性
Enter           - 返回网格模式
H               - 显示/隐藏帮助
Tab             - 切换帮助位置
ESC             - 退出
```

---

## 编译和运行

### 编译
```bash
go build -o animation_showcase \
  cmd/animation_showcase/main.go \
  cmd/animation_showcase/animation_cell.go \
  cmd/animation_showcase/config.go \
  cmd/animation_showcase/grid_layout.go
```

### 运行
```bash
# 使用默认配置
./animation_showcase

# 指定配置文件
./animation_showcase --config=cmd/animation_showcase/config.yaml

# 启用详细日志
./animation_showcase --verbose
```

---

## 技术实现要点

### 1. JPG+PNG 蒙版处理
- 使用 Go 标准库 `image` 包处理图片
- 逐像素应用 alpha 通道
- 预乘 alpha 改善边缘质量

### 2. 显示模式切换
- 使用枚举类型 `DisplayMode` 管理状态
- 根据模式动态调整渲染逻辑
- 帮助信息根据模式自适应

### 3. 轨道管理
- 使用 `manualHiddenTracks` map 跟踪手动隐藏的轨道
- 独立于配置文件的 `hiddenTracks`
- 实时更新渲染缓存

---

## 注意事项

1. **图片蒙版**:
   - 蒙版文件必须与 JPG 文件尺寸完全一致
   - 蒙版文件必须是 PNG 格式

2. **轨道切换**:
   - 最多支持 12 个轨道（F1-F12）
   - 如果轨道超过 12 个，只显示前 12 个

3. **性能**:
   - 图片蒙版处理是一次性操作，在加载时完成
   - 轨道切换会触发渲染缓存更新

---

## 更新日期
2025-11-07
