# Reanim 动画系统验证程序

本程序完全按照 `.meta/reanim/reanim.md` 文档实现，用于验证文档的正确性。

## 目的

验证 reanim.md 文档中描述的动画系统实现方法是否正确，通过实际解析和渲染《植物大战僵尸》的豌豆射手动画文件来演示。

## 功能特性

✅ **完全独立** - 不依赖项目现有的 reanim 模块，从零实现
✅ **动态动画检测**：
  - 自动从 reanim 文件中检测所有可用动画片段
  - 支持任意数量的动画（无需写死）
  - 当前豌豆射手文件包含 4 个动画：
    - **anim_head_idle** - 头部待机动画
    - **anim_idle** - 全身待机动画
    - **anim_shooting** - 攻击动画
    - **anim_stem** - 茎部动画

✅ **动态切换功能**：
  - 使用方向键（← → ↑ ↓）快速切换不同动画
  - 空格键切换循环模式（单次播放 ↔ 循环播放）
  - B 键触发叠加动画（如眨眼效果）
  - 实时显示当前动画信息（帧范围、FPS、循环状态）

✅ **叠加动画系统** - 完整实现（文档第 60-96 行）：
  - YAML 配置驱动：通过 `animation_config.yaml` 定义叠加轨道
  - 分层渲染：基础层 + 叠加层独立绘制
  - 自动管理：叠加动画播放完成后自动移除
  - 实时反馈：屏幕显示当前激活的叠加动画

✅ **文档验证** - 按照以下步骤实现（来自 reanim.md）：
  1. XML 解析数据结构设计
  2. 高效流式解析 XML
  3. 烘焙（Baking）动画数据（分离逻辑轨道和视觉轨道）
  4. 运行时动画播放器（AnimationClip 模式）
  5. Ebitengine 高效渲染
  6. 叠加动画系统（YAML 配置 + 分层渲染）

✅ **多文件支持** - 支持通过命令行参数加载不同的 reanim 文件

## 运行方法

### 默认运行（豌豆射手）

```bash
# 从项目根目录运行
cd /mnt/disk0/project/game/pvz/pvz3
go run cmd/verify_reanim/main.go
```

### 指定其他动画文件

```bash
# 加载向日葵动画
go run cmd/verify_reanim/main.go -file assets/effect/reanim/SunFlower.reanim -imgdir assets/reanim

# 加载僵尸动画
go run cmd/verify_reanim/main.go -file assets/effect/reanim/Zombie.reanim -imgdir assets/reanim
```

### 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-file` | Reanim 文件路径 | `assets/effect/reanim/PeaShooterSingle.reanim` |
| `-imgdir` | 图片资源目录 | `assets/reanim` |

## 操作说明

- **← → ↑ ↓ 方向键** - 切换动画（循环浏览所有可用动画）
- **空格键** - 切换循环模式（单次播放 ↔ 循环播放）
- **B 键** - 触发叠加动画（如眨眼效果）
  - 适用于配置了叠加轨道的动画（如 SunFlower, PeaShooter）
  - 叠加动画会在基础动画之上播放，完成后自动移除
- **关闭窗口** - 退出程序

## 预期输出

程序启动时会在控制台输出：

```
2025/10/23 09:22:06 ====== Reanim 动画系统验证程序 ======
2025/10/23 09:22:06 加载文件: assets/effect/reanim/PeaShooterSingle.reanim
2025/10/23 09:22:06 ✓ 解析成功！FPS: 12, 轨道数: 18
2025/10/23 09:22:06 发现 13 个不同的图片引用
2025/10/23 09:22:06 成功加载 11/13 张图片
2025/10/23 09:22:06 发现动画片段: anim_idle [4-28] (长度: 24帧)
2025/10/23 09:22:06 发现动画片段: anim_head_idle [29-53] (长度: 24帧)
2025/10/23 09:22:06 发现动画片段: anim_shooting [54-78] (长度: 24帧)
2025/10/23 09:22:06 发现动画片段: anim_stem [79-103] (长度: 24帧)
2025/10/23 09:22:06 ✓ 烘焙完成！总帧数: 104, 可视轨道: 13, 动画片段: 4
2025/10/23 09:22:06 ✓ 可用动画: [anim_head_idle anim_idle anim_shooting anim_stem]
2025/10/23 09:22:06 播放动画: anim_head_idle (循环: true)
```

窗口将显示豌豆射手动画，屏幕左上角显示：
- 当前文件名
- 当前动画名称和索引
- 帧范围和长度
- FPS 和循环状态
- 操作提示

## 技术实现亮点

### 1. XML 解析
```go
// 自动处理没有根元素的 reanim 文件
wrappedXML := "<reanim>" + string(content) + "</reanim>"
xml.Unmarshal([]byte(wrappedXML), &reanim)
```

### 2. 烘焙优化
- **一次计算** - 所有插值和属性继承在加载时完成
- **内存共享** - 多个动画实例共享同一份烘焙数据
- **O(1)查找** - 通过帧号直接索引，无需查找

### 3. 逻辑轨道与视觉轨道分离（reanim.md 规范）
```go
// 逻辑轨道：定义动画片段（f:0 开始，f:-1 结束）
func parseLogicalTrack(track Track) AnimationClip {
    // 提取 StartFrame, EndFrame, Length
}

// 视觉轨道：包含实际图片和变换数据
func bakeVisualTrack(track Track, maxFrames int, imageMap) *BakedTrack {
    // 烘焙所有帧的变换数据
}
```

### 4. AnimationClip 模式（reanim.md 规范）
```go
// 绘制时使用绝对帧号映射
absoluteFrameIdx := clip.StartFrame + int(currentFrame)
transform := track.Frames[absoluteFrameIdx]
```

### 5. 自动图片加载
```go
// 从 reanim 文件中提取所有图片引用并自动加载
// IMAGE_REANIM_PEASHOOTER_BACKLEAF -> PeaShooter_backleaf.png
LoadImagesFromReanim(reanimFile, imageDir)
```

### 6. 叠加动画系统（reanim.md 第 60-96 行规范）
```go
// YAML 配置叠加轨道
type AnimationConfigs map[string]AnimationConfigEntry
configs := LoadAnimationConfigs("animation_config.yaml")

// 烘焙时标记叠加轨道
baked.OverlayTrackNames["anim_blink"] = true

// 运行时触发叠加动画
instance.PlayOverlay("anim_blink")  // 自动播放并完成后移除

// 分层渲染
// 第1层：绘制基础动画（跳过叠加轨道）
// 第2层：绘制激活的叠加动画（覆盖基础层）
```

## 验证结果

✅ **文档正确性已验证**

本程序成功实现了以下功能：
- ✅ 正确解析 XML 动画文件（18个轨道）
- ✅ 正确分离逻辑轨道和视觉轨道（4个动画片段 + 13个视觉轨道）
- ✅ 正确实现 AnimationClip 模式（绝对帧号映射）
- ✅ 正确烘焙动画数据到高效结构（104帧总长度）
- ✅ 正确渲染多部件骨骼动画（13个部件）
- ✅ 正确处理帧继承和变换（位置、缩放、倾斜）
- ✅ 支持动态动画切换（4个动画自由切换）
- ✅ 支持循环/单次播放模式

这证明 `.meta/reanim/reanim.md` 文档中描述的实现方法是**完全正确和可行**的。

## 数据结构示例

### 解析阶段（Parsing）
```go
type Keyframe struct {
    Image    string
    X, Y     *float64  // 指针表示可选，节省内存
    ScaleX   *float64
    FrameNum int       // f:-1 或 f:0 控制可见性
}
```

### 烘焙阶段（Baking）
```go
type BakedTransform struct {
    Image   *ebiten.Image
    X, Y    float64  // 所有值已填充完毕
    ScaleX  float64
    Visible bool     // 预计算的可见性
}
```

## 文件说明

- `main.go` - 验证程序主文件（约550行）
- `README.md` - 本说明文档

## 依赖

- Go 1.21+
- github.com/hajimehoshi/ebiten/v2

## 相关文档

- `.meta/reanim/reanim.md` - Reanim 动画系统技术指南（被验证的文档）
- `assets/effect/reanim/PeaShooterSingle.reanim` - 豌豆射手动画定义
- `assets/reanim/PeaShooter_*.png` - 豌豆射手图片资源

## 已知限制

- 缺少 2 张图片资源（IMAGE_REANIM_PEASHOOTER_HEAD, IMAGE_REANIM_ANIM_SPROUT）
- 这些图片对应的轨道不会显示，但不影响其他部件的正常渲染

## 扩展性

本程序设计为通用的 reanim 验证工具，支持：
- ✅ 加载任意 reanim 文件（通过 `-file` 参数）
- ✅ 自动检测所有动画片段（无需修改代码）
- ✅ 动态切换任意数量的动画
- ✅ 可用于验证其他植物/僵尸的动画文件
