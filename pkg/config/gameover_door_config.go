package config

// 游戏结束房门渲染相关配置
// Story 8.8 - Task 6: 僵尸获胜流程房门打开视觉效果
//
// 房门由两部分组成：
//   - Interior Overlay (内部阴影层)：房门右半部分/阴影，渲染在僵尸下层
//   - Mask (门板层)：房门左半部分/门板，渲染在僵尸上层，遮挡僵尸以模拟进屋效果
//
// 坐标说明：
//   - 坐标为世界坐标（相对于背景图左上角）
//   - 渲染时会自动转换为屏幕坐标（减去 cameraX）
//   - 背景图尺寸：1400x600
//   - 房门位于背景图左上角的房子区域

// GameOverDoorInteriorOverlayX 房门内部阴影层 X 坐标（世界坐标）
// 图片尺寸：70 x 195
// 调整指南：向右增加，向左减少
const GameOverDoorInteriorOverlayX = 94.0

// GameOverDoorInteriorOverlayY 房门内部阴影层 Y 坐标（世界坐标）
// 调整指南：向下增加，向上减少
const GameOverDoorInteriorOverlayY = 224.0

// GameOverDoorMaskX 房门门板层 X 坐标（世界坐标）
// 图片尺寸：47 x 248
// 门板位于阴影左侧
// 调整指南：向右增加，向左减少
const GameOverDoorMaskX = 90.0

// GameOverDoorMaskY 房门门板层 Y 坐标（世界坐标）
// 调整指南：向下增加，向上减少
const GameOverDoorMaskY = 202.0
