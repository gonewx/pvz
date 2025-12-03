# 坐标系统设计 (Coordinate System Design)

## 概述 (Overview)

本项目采用**世界坐标系统**（World Coordinate System）作为主要的空间参考系统。这种设计使得游戏对象的位置独立于摄像机位置，支持未来的摄像机动画和地图滚动功能。

## 坐标系统定义

### 世界坐标系统 (World Coordinates)
- **参考点：** 背景图片的左上角
- **特性：** 固定不变，不随摄像机移动而改变
- **用途：** 所有游戏对象的逻辑位置、网格定义、碰撞检测

### 屏幕坐标系统 (Screen Coordinates)
- **参考点：** 游戏窗口的左上角
- **特性：** 随摄像机位置变化
- **用途：** 渲染时的最终位置、鼠标输入坐标

## 坐标转换

### 公式
```go
// 世界坐标 → 屏幕坐标
screenX = worldX - cameraX
screenY = worldY  // Y轴不受水平摄像机影响

// 屏幕坐标 → 世界坐标
worldX = screenX + cameraX
worldY = screenY
```

### 实现位置
- **转换函数：** `pkg/utils/grid_utils.go`
  - `MouseToGridCoords()` - 鼠标屏幕坐标 → 网格坐标
  - `GridToScreenCoords()` - 网格坐标 → 屏幕坐标
- **摄像机状态：** `game.GameState.CameraX`
- **配置常量：** `pkg/config/layout_config.go`

## 使用示例

### 示例 1：网格位置定义
```go
// pkg/config/layout_config.go
const (
    GridWorldStartX = 251.0  // 网格在背景图片中的X位置
    GridWorldStartY = 72.0   // 网格在背景图片中的Y位置
)
```

### 示例 2：鼠标点击检测
```go
// 获取鼠标位置（屏幕坐标）
mouseX, mouseY := ebiten.CursorPosition()

// 转换为网格坐标
col, row, isValid := utils.MouseToGridCoords(
    mouseX, mouseY,
    gameState.CameraX,  // 当前摄像机位置
    config.GridWorldStartX, config.GridWorldStartY,
    config.GridColumns, config.GridRows,
    config.CellWidth, config.CellHeight,
)
```

### 示例 3：植物渲染
```go
// 植物的世界坐标（存储在PositionComponent中）
worldX := position.X
worldY := position.Y

// 转换为屏幕坐标进行渲染
screenX := worldX - gameState.CameraX
screenY := worldY

// 绘制到屏幕
op := &ebiten.DrawImageOptions{}
op.GeoM.Translate(screenX, screenY)
screen.DrawImage(plantSprite, op)
```

## 摄像机系统

### 当前实现
- **默认位置：** `GameCameraX = 215`（游戏场景居中）
- **开场动画：** 从 x=0 滚动到 x=maxCameraX，再回到 215
- **同步机制：** `game_scene.Update()` 每帧同步 `cameraX` 到 `GameState`

### 扩展性
世界坐标系统为以下未来功能提供了基础：
- 摄像机跟随效果（如跟随选中的植物）
- 地图滚动（支持更大的关卡）
- 镜头震动特效
- 过场动画

## 注意事项

### 开发者须知
1. **组件中存储世界坐标：** `PositionComponent` 应使用世界坐标
2. **渲染时转换：** `RenderSystem` 负责转换为屏幕坐标
3. **输入处理：** `InputSystem` 将鼠标屏幕坐标转换为世界坐标
4. **使用配置常量：** 避免硬编码，使用 `pkg/config/layout_config.go` 中的常量

### 常见错误
- ❌ 在组件中存储屏幕坐标
- ❌ 硬编码网格位置而不使用配置
- ❌ 忘记传递 `cameraX` 参数给坐标转换函数

## 测试覆盖

相关测试文件：
- `pkg/utils/grid_utils_test.go` - 坐标转换在不同 cameraX 下的一致性测试
- `pkg/systems/plant_preview_system_test.go` - 网格坐标系统集成测试

## 变更历史

| 日期 | 版本 | 描述 |
|------|------|------|
| 2025-10-12 | 1.0 | 初始版本：引入世界坐标系统，重构网格坐标转换 |
