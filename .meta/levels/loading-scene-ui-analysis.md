# Loading 加载界面 UI 分析文档

> **文档版本**: v1.0
> **创建日期**: 2025-11-19
> **用途**: Loading 场景实现的 UI 结构、动画效果和资源清单

---

## 📋 Loading 界面 UI 元素分析

### 1. **背景层**
- **元素**: 标题屏幕背景（IMAGE_TITLESCREEN）
- **资源**: `assets/images/titlescreen.jpg` (800×600px)
- **实现**: 静态背景图

---

### 2. **Logo 层**
- **元素**: 植物大战僵尸 Logo（IMAGE_PVZ_LOGO）
- **资源**: 需要叠加对应的蒙板，应用预乘 Alpha 概念消除白边残留

图片： `assets/images/PvZ_Logo.jpg`
蒙板图片： `assets/images/PvZ_Logo_.png` 

- **位置**: 屏幕上方居中
- **实现**: 动画效果，需要表现为从屏幕外下落的效果。右下角要同时渲染黑色文字  `TM`,使用字体`assets/fonts/SimHei.ttf`

---

### 3. **进度条组件（核心动画）**

#### 3.1 **泥土底条**
- **元素**: LoadBar_dirt（进度条背景）
- **资源**: `assets/images/LoadBar_dirt.png` (321×53px)
- **位置**: 屏幕底部居中偏下（Y ≈ 450-500）
- **实现**: 静态底座，绘制在草皮条下方

#### 3.2 **草皮进度条**
- **元素**: LoadBar_grass（进度填充）
- **资源**: `assets/images/LoadBar_grass.png` (314×33px)
- **位置**: 叠加在泥土条上方（Y偏移约 10px）
- **动画**:
  - **类型**: 水平裁剪动画（从左到右逐渐显示）
  - **实现**: 使用 `SubImage()` 裁剪，宽度 = `totalWidth × progress%`
  - **速度**: 匀速或缓动（2-3秒完成）
  - **附加动画**: 在不同进度时，要显示附加的小动画
  依次是：loadbar_sprout，loadbar_sprout (镜像)， loadbar_sprout （放大）， loadbar_sprout (镜像)， loadbar_zombiehead
  动画显示的位置可通过配置模块 `pkg/config` 配置
  动画的配置文件在：
  data/reanim_config/loadbar_sprout.yaml
  data/reanim_config/loadbar_zombiehead.yaml

#### 3.3 **草皮卷盖动画（SodRollCap） Reanim）**
- **元素**: 草皮卷盖（SodRollCap）
- **资源**:
  - `data/reanim/SodRoll.reanim`（已配置）
  - `assets/reanim/SodRollCap.png`（草皮卷盖子）
- **动画**: `SodRollCap`（播放SodRollCap轨道）
- **位置**: 跟随进度条右端移动
  - X坐标 = `loadBarX + (314 × progress%)`
  - Y坐标 = `loadBarY - 草皮卷盖动画高度偏移`
- **实现**:
  - 位置随进度更新（通过 PositionComponent）

---

### 4. **文字提示层**
- **元素**: 载入时显示"载入中……" 文字， 完成后显示 "点击开始"
- **位置**: 进度条下方（Y ≈ 520-540）
- **字体**: 使用字体`assets/fonts/SimHei.ttf`
- **颜色**: 载入中土黄色和"点击开始"是红色，加阴影效果
- **实现**: 使用 `text.Draw()` 绘制居中文字

---

### 5. **音效**
- **按钮点击音效**: `SOUND_BUTTONCLICK`
- **加载音效**（可选）:
  - `SOUND_LOADINGBAR_FLOWER` - 加载完成音效
  - `SOUND_LOADINGBAR_ZOMBIE` - 加载期间音效

---

## 📦 资源清单和配置要求

### 必需资源（已准备）

| 资源 ID | 文件路径 | 用途 | 状态 |
|---------|---------|------|------|
| `IMAGE_TITLESCREEN` | `assets/images/titlescreen.jpg` | 背景图 | ✅ 已存在 |
| `IMAGE_PVZ_LOGO` | `assets/images/PvZ_Logo.jpg` | Logo | ✅ 已存在 |
| `IMAGE_LOADBAR_DIRT` | `assets/images/LoadBar_dirt.png` | 进度条底座 | ✅ 已存在 |
| `IMAGE_LOADBAR_GRASS` | `assets/images/LoadBar_grass.png` | 进度条填充 | ✅ 已存在 |
| `IMAGE_REANIM_SODROLLCAP` | `assets/reanim/SodRollCap.png` | 草皮卷盖 | ✅ 已存在 |
| `SOUND_BUTTONCLICK` | `assets/sounds/buttonclick.ogg` | 点击音效 | ✅ 已存在 |
| `SOUND_LOADINGBAR_FLOWER` | `assets/sounds/loadingbar_flower.ogg` | 完成音效 | ✅ 已存在 |
| `SOUND_LOADINGBAR_ZOMBIE` | `assets/sounds/loadingbar_zombie.ogg` | 加载音效 | ✅ 已存在 |

### Reanim 配置（已准备）

- **配置文件**: `data/reanim_config/sodroll.yaml` ✅
- **Reanim 文件**: `data/reanim/SodRoll.reanim` ✅
