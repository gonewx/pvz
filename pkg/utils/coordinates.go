// Package utils 提供游戏开发中常用的工具函数
//
// coordinates.go 提供坐标转换工具库，用于处理 Reanim 动画的坐标计算。
// 本工具库消除重复代码、降低认知负担、减少坐标计算错误，提高代码可维护性。
//
// # 坐标系统概述
//
// 本项目使用以下坐标系统：
//   - **世界坐标**：相对于背景图片左上角（固定）
//   - **屏幕坐标**：相对于游戏窗口左上角（随摄像机移动）
//   - **实体锚点**：BoundingBox 中心锚点（PositionComponent.X/Y 代表实体的视觉中心）
//   - **图片锚点**：左上角（Ebiten 默认行为）
//
// # 核心转换公式
//
// 渲染原点计算（世界坐标 → 屏幕坐标）：
//
//	baseScreenX = pos.X - cameraX - CenterOffsetX
//	baseScreenY = pos.Y - CenterOffsetY
//
// 其中：
//   - pos.X/Y：实体中心的世界坐标（来自 PositionComponent）
//   - cameraX：摄像机水平偏移（UI 元素为 0）
//   - CenterOffsetX/Y：BoundingBox 中心偏移（来自 ReanimComponent，在动画初始化时计算一次）
//
// # 使用场景
//
// - **渲染系统**：使用 GetRenderScreenOrigin 计算屏幕坐标进行绘制
// - **点击检测**：使用 GetClickableCenter 计算点击中心（世界坐标）
// - **草皮系统**：使用 ReanimLocalToWorld 将 Reanim 局部坐标转换为世界坐标
// - **通用转换**：使用 WorldToScreen 执行世界坐标到屏幕坐标的转换
//
// # 错误处理
//
// 所有需要 ReanimComponent 的函数在实体缺少该组件时返回 ErrNoReanimComponent。
// 调用者可使用 errors.Is 检查错误类型：
//
//	screenX, screenY, err := coordinates.GetRenderScreenOrigin(em, entityID, pos, cameraX)
//	if err != nil {
//	    if errors.Is(err, coordinates.ErrNoReanimComponent) {
//	        // 处理无动画组件的情况
//	        return
//	    }
//	    log.Printf("获取渲染坐标失败: %v", err)
//	    return
//	}
//	// 正常使用 screenX, screenY
package utils

import (
	"errors"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
)

// ErrNoReanimComponent 表示实体缺少 ReanimComponent 组件
// 当坐标转换函数需要 CenterOffset 但实体没有动画组件时返回此错误
var ErrNoReanimComponent = errors.New("entity has no ReanimComponent")

// GetRenderScreenOrigin 计算 Reanim 实体的渲染原点（屏幕坐标）
//
// 此函数是最常用的坐标转换函数，主要用于渲染系统。
// 它将实体的世界坐标转换为屏幕坐标，考虑摄像机偏移和 BoundingBox 中心偏移。
//
// # 参数
//
//   - em: ECS 实体管理器，用于查询组件
//   - entityID: 实体 ID
//   - pos: 实体的位置组件（世界坐标）
//   - cameraX: 摄像机水平偏移（游戏场景中通常 > 0，UI 场景中为 0）
//
// # 返回值
//
//   - screenX, screenY: 渲染原点的屏幕坐标（左上角基准）
//   - err: 如果实体缺少 ReanimComponent，返回 ErrNoReanimComponent
//
// # 计算公式
//
//	screenX = pos.X - effectiveCameraX - CenterOffsetX
//	screenY = pos.Y - CenterOffsetY
//
// 其中，UI 元素的 effectiveCameraX = 0（不受摄像机影响）
//
// # 使用示例
//
//	// 渲染系统中计算基础屏幕坐标
//	baseScreenX, baseScreenY, err := GetRenderScreenOrigin(em, entityID, pos, cameraX)
//	if err != nil {
//	    return err
//	}
//	// 叠加部件相对坐标
//	partX := frame.X + partData.OffsetX
//	partY := frame.Y + partData.OffsetY
//	finalX := baseScreenX + partX
//	finalY := baseScreenY + partY
//	// 绘制部件
//	screen.DrawImage(partImage, op)
func GetRenderScreenOrigin(
	em *ecs.EntityManager,
	entityID ecs.EntityID,
	pos *components.PositionComponent,
	cameraX float64,
) (screenX, screenY float64, err error) {
	// 查询 ReanimComponent 以获取 CenterOffset
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](em, entityID)
	if !ok {
		return 0, 0, ErrNoReanimComponent
	}

	// 检查是否是 UI 元素（UI 元素不受摄像机影响）
	_, isUI := ecs.GetComponent[*components.UIComponent](em, entityID)
	effectiveCameraX := cameraX
	if isUI {
		effectiveCameraX = 0 // UI 元素使用屏幕坐标，不应用摄像机偏移
	}

	// 获取实体整体缩放（ScaleComponent 和 ReanimComponent.ScaleX/Y）
	entityScaleX := 1.0
	entityScaleY := 1.0
	if scaleComp, hasScale := ecs.GetComponent[*components.ScaleComponent](em, entityID); hasScale {
		entityScaleX = scaleComp.ScaleX
		entityScaleY = scaleComp.ScaleY
	}
	if reanimComp.ScaleX != 0 {
		entityScaleX *= reanimComp.ScaleX
	}
	if reanimComp.ScaleY != 0 {
		entityScaleY *= reanimComp.ScaleY
	}

	// 计算屏幕坐标（世界坐标 - 摄像机偏移 - 居中偏移 * 缩放）
	// CenterOffset 是在 scale=1.0 时计算的，需要乘以当前缩放比例
	screenX = pos.X - effectiveCameraX - reanimComp.CenterOffsetX*entityScaleX
	screenY = pos.Y - reanimComp.CenterOffsetY*entityScaleY

	return screenX, screenY, nil
}

// GetClickableCenter 计算 Reanim 实体的点击中心（世界坐标）
//
// 此函数用于点击检测系统，计算实体的视觉中心位置（世界坐标）。
// 点击中心对齐于 BoundingBox 中心，用于判断鼠标是否点击了实体。
//
// # 参数
//
//   - em: ECS 实体管理器，用于查询组件
//   - entityID: 实体 ID
//   - pos: 实体的位置组件（世界坐标）
//
// # 返回值
//
//   - centerX, centerY: 点击中心的世界坐标
//   - err: 如果实体缺少 ReanimComponent，返回 ErrNoReanimComponent
//
// # 计算公式
//
//	centerX = pos.X - CenterOffsetX
//	centerY = pos.Y - CenterOffsetY
//
// # 使用示例
//
//	// 点击检测系统中计算点击中心
//	clickCenterX, clickCenterY, err := GetClickableCenter(em, entityID, pos)
//	if err != nil {
//	    continue // 跳过没有动画组件的实体
//	}
//	halfWidth := clickable.Width / 2.0
//	halfHeight := clickable.Height / 2.0
//	// 检测鼠标是否在点击区域内
//	if mouseWorldX >= clickCenterX-halfWidth &&
//	   mouseWorldX <= clickCenterX+halfWidth &&
//	   mouseWorldY >= clickCenterY-halfHeight &&
//	   mouseWorldY <= clickCenterY+halfHeight {
//	    // 点击命中
//	}
func GetClickableCenter(
	em *ecs.EntityManager,
	entityID ecs.EntityID,
	pos *components.PositionComponent,
) (centerX, centerY float64, err error) {
	// 查询 ReanimComponent 以获取 CenterOffset
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](em, entityID)
	if !ok {
		return 0, 0, ErrNoReanimComponent
	}

	// 获取实体整体缩放（ScaleComponent 和 ReanimComponent.ScaleX/Y）
	entityScaleX := 1.0
	entityScaleY := 1.0
	if scaleComp, hasScale := ecs.GetComponent[*components.ScaleComponent](em, entityID); hasScale {
		entityScaleX = scaleComp.ScaleX
		entityScaleY = scaleComp.ScaleY
	}
	if reanimComp.ScaleX != 0 {
		entityScaleX *= reanimComp.ScaleX
	}
	if reanimComp.ScaleY != 0 {
		entityScaleY *= reanimComp.ScaleY
	}

	// 计算视觉中心（世界坐标）
	// CenterOffset 是在 scale=1.0 时计算的，需要乘以当前缩放比例
	centerX = pos.X - reanimComp.CenterOffsetX*entityScaleX
	centerY = pos.Y - reanimComp.CenterOffsetY*entityScaleY

	return centerX, centerY, nil
}

// GetRenderOrigin 计算 Reanim 实体的渲染原点（世界坐标）
//
// 此函数用于需要世界坐标的场景（如草皮系统）。
// 它计算实体渲染原点的世界坐标（不考虑摄像机偏移）。
//
// # 参数
//
//   - em: ECS 实体管理器，用于查询组件
//   - entityID: 实体 ID
//   - pos: 实体的位置组件（世界坐标）
//
// # 返回值
//
//   - originX, originY: 渲染原点的世界坐标（左上角基准）
//   - err: 如果实体缺少 ReanimComponent，返回 ErrNoReanimComponent
//
// # 计算公式
//
//	originX = pos.X - CenterOffsetX
//	originY = pos.Y - CenterOffsetY
//
// # 使用示例
//
//	// 草皮系统中计算渲染原点（世界坐标）
//	originX, originY, err := GetRenderOrigin(em, entityID, pos)
//	if err != nil {
//	    return err
//	}
//	// 计算草皮边缘的世界坐标
//	leftEdgeWorld := originX + sodRollLeftEdge
//	rightEdgeWorld := originX + sodRollRightEdge
func GetRenderOrigin(
	em *ecs.EntityManager,
	entityID ecs.EntityID,
	pos *components.PositionComponent,
) (originX, originY float64, err error) {
	// 查询 ReanimComponent 以获取 CenterOffset
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](em, entityID)
	if !ok {
		return 0, 0, ErrNoReanimComponent
	}

	// 获取实体整体缩放（ScaleComponent 和 ReanimComponent.ScaleX/Y）
	entityScaleX := 1.0
	entityScaleY := 1.0
	if scaleComp, hasScale := ecs.GetComponent[*components.ScaleComponent](em, entityID); hasScale {
		entityScaleX = scaleComp.ScaleX
		entityScaleY = scaleComp.ScaleY
	}
	if reanimComp.ScaleX != 0 {
		entityScaleX *= reanimComp.ScaleX
	}
	if reanimComp.ScaleY != 0 {
		entityScaleY *= reanimComp.ScaleY
	}

	// 计算渲染原点（世界坐标）
	// CenterOffset 是在 scale=1.0 时计算的，需要乘以当前缩放比例
	originX = pos.X - reanimComp.CenterOffsetX*entityScaleX
	originY = pos.Y - reanimComp.CenterOffsetY*entityScaleY

	return originX, originY, nil
}

// ReanimLocalToWorld 将 Reanim 局部坐标转换为世界坐标
//
// 此函数用于草皮系统等需要将 Reanim 部件的局部坐标转换为世界坐标的场景。
// 局部坐标是相对于实体渲染原点（左上角）的偏移量。
//
// # 参数
//
//   - em: ECS 实体管理器，用于查询组件
//   - entityID: 实体 ID
//   - pos: 实体的位置组件（世界坐标）
//   - localX, localY: Reanim 局部坐标（相对于渲染原点的偏移）
//
// # 返回值
//
//   - worldX, worldY: 转换后的世界坐标
//   - err: 如果实体缺少 ReanimComponent，返回 ErrNoReanimComponent
//
// # 计算公式
//
//	worldX = pos.X - CenterOffsetX + localX
//	worldY = pos.Y - CenterOffsetY + localY
//
// # 使用示例
//
//	// 草皮系统中计算草皮卷中心的世界坐标
//	sodRollCenterX := frame.X + scaledHalfWidth  // Reanim 局部坐标
//	worldCenterX, worldCenterY, err := ReanimLocalToWorld(em, entityID, pos, sodRollCenterX, 0)
//	if err != nil {
//	    return err
//	}
//	// worldCenterX 现在是草皮卷中心的世界坐标
func ReanimLocalToWorld(
	em *ecs.EntityManager,
	entityID ecs.EntityID,
	pos *components.PositionComponent,
	localX, localY float64,
) (worldX, worldY float64, err error) {
	// 查询 ReanimComponent 以获取 CenterOffset
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](em, entityID)
	if !ok {
		return 0, 0, ErrNoReanimComponent
	}

	// 转换局部坐标到世界坐标
	worldX = pos.X - reanimComp.CenterOffsetX + localX
	worldY = pos.Y - reanimComp.CenterOffsetY + localY

	return worldX, worldY, nil
}

// WorldToScreen 将世界坐标转换为屏幕坐标
//
// 此函数执行通用的世界坐标到屏幕坐标转换。
// 这是一个纯计算函数，不需要查询组件。
//
// # 参数
//
//   - worldX, worldY: 世界坐标
//   - cameraX: 摄像机水平偏移（游戏场景中通常 > 0，UI 场景中为 0）
//   - isUI: 是否是 UI 元素（UI 元素不受摄像机影响）
//
// # 返回值
//
//   - screenX, screenY: 转换后的屏幕坐标
//
// # 计算公式
//
//	screenX = worldX - effectiveCameraX  （UI 元素：effectiveCameraX = 0）
//	screenY = worldY
//
// # 使用示例
//
//	// 通用坐标转换
//	screenX, screenY := WorldToScreen(worldX, worldY, cameraX, false)
//	// UI 元素坐标转换
//	uiScreenX, uiScreenY := WorldToScreen(uiWorldX, uiWorldY, cameraX, true)
func WorldToScreen(worldX, worldY float64, cameraX float64, isUI bool) (screenX, screenY float64) {
	effectiveCameraX := cameraX
	if isUI {
		effectiveCameraX = 0 // UI 元素不受摄像机影响
	}

	screenX = worldX - effectiveCameraX
	screenY = worldY

	return screenX, screenY
}
