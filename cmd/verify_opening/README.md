# Opening Animation Verifier

快速验证开场动画系统的工具，无需完成整个关卡即可测试镜头移动和僵尸预告效果。

## 用途

- ✅ 快速测试开场动画，无需完成关卡（5秒 vs 5分钟）
- ✅ 调试镜头移动和缓动效果
- ✅ 验证僵尸预告生成逻辑
- ✅ 调整动画时序和参数

## 使用方法

### 基本使用

```bash
# 测试 1-2 关卡（默认）
go run cmd/verify_opening/main.go

# 测试指定关卡
go run cmd/verify_opening/main.go --level=1-3

# 调整镜头速度
go run cmd/verify_opening/main.go --speed=500

# 跳过初始延迟
go run cmd/verify_opening/main.go --skip-delay

# 详细日志
go run cmd/verify_opening/main.go --verbose
```

### 快捷键

- **Space / ESC** - 跳过开场动画
- **R** - 重新开始动画
- **Q** - 退出程序

## 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--level` | `1-2` | 要测试的关卡ID（如 1-2, 1-3, 1-4） |
| `--speed` | `300` | 镜头动画速度（像素/秒） |
| `--skip-delay` | `false` | 跳过初始空闲延迟，立即开始动画 |
| `--verbose` | `false` | 启用详细日志输出 |

## 示例场景

### 场景 1: 快速迭代镜头速度

```bash
# 测试不同速度效果
go run cmd/verify_opening/main.go --speed=200   # 慢速
go run cmd/verify_opening/main.go --speed=300   # 默认
go run cmd/verify_opening/main.go --speed=500   # 快速
```

### 场景 2: 验证僵尸预告

```bash
# 测试 1-3 关卡（多种僵尸类型）
go run cmd/verify_opening/main.go --level=1-3 --verbose
# 观察僵尸预告生成的类型和位置
```

### 场景 3: 调试跳过功能

```bash
# 启动后立即按 Space 键测试跳过
go run cmd/verify_opening/main.go
# 按 Space 键，观察是否正确清理僵尸和镜头
```

## 技术细节

### 加载的系统

- ✅ CameraSystem - 镜头控制
- ✅ OpeningAnimationSystem - 开场动画编排
- ✅ RenderSystem - 渲染背景和僵尸
- ✅ ReanimSystem - 僵尸动画
- ✅ AnimationSystem - 基础动画支持
- ❌ InputSystem - 不需要（玩家交互已禁用）
- ❌ WaveSpawnSystem - 不需要（使用预告僵尸）

### 调试信息显示

屏幕左上角显示：
- 当前状态机状态（idle, cameraMoveRight, showZombies, etc.）
- 镜头当前位置（CameraX）
- 预告僵尸数量
- 操作提示

## 注意事项

1. **教学关卡**（1-1）无开场动画，程序会提示并等待退出
2. **特殊关卡**（如保龄球模式）也无开场动画
3. 只有 `openingType: "standard"` 的关卡才会播放动画

## 故障排查

### 问题：显示 "No opening animation for this level"

**原因**：关卡配置为教学关卡或跳过开场
**解决**：使用标准关卡测试（如 `--level=1-2`）

### 问题：僵尸不显示

**原因**：关卡配置中 Waves 为空
**解决**：检查 `data/levels/level-X-X.yaml` 中是否配置了僵尸波次

### 问题：镜头移动太快/太慢

**解决**：使用 `--speed` 参数调整速度

## 相关文件

- **Story 文档**: `docs/stories/8.3.story.md`
- **系统实现**: `pkg/systems/opening_animation_system.go`
- **镜头系统**: `pkg/systems/camera_system.go`
- **关卡配置**: `data/levels/level-1-2.yaml`
