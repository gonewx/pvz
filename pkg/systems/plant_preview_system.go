package systems

import (
	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/utils"
)

// PlantPreviewSystem 更新植物预览的位置
// 为实现双图像预览（光标处不透明 + 格子处半透明），系统会计算并存储两个位置：
//  1. 鼠标光标位置（mouseWorldX, mouseWorldY）- 用于渲染不透明图像
//  2. 网格对齐位置（gridAlignedWorldX, gridAlignedWorldY）- 用于渲染半透明预览图像
//
// 注意：所有位置都使用世界坐标系统
type PlantPreviewSystem struct {
	entityManager  *ecs.EntityManager
	gameState      *game.GameState
	lawnGridSystem *LawnGridSystem // Story 8.1: 用于检查行是否启用

	// 鼠标光标位置（世界坐标）- 用于渲染不透明光标图像
	mouseWorldX float64
	mouseWorldY float64

	// 网格对齐位置（世界坐标）- 用于渲染半透明预览图像
	gridAlignedWorldX float64
	gridAlignedWorldY float64

	// 是否在网格内
	isInGrid bool
}

// NewPlantPreviewSystem 创建植物预览系统
// Story 8.1: 添加 lawnGridSystem 参数以支持行启用检查
func NewPlantPreviewSystem(em *ecs.EntityManager, gs *game.GameState, lawnGridSystem *LawnGridSystem) *PlantPreviewSystem {
	return &PlantPreviewSystem{
		entityManager:  em,
		gameState:      gs,
		lawnGridSystem: lawnGridSystem,
	}
}

// Update 更新预览实体的位置
// 计算两个位置供渲染使用（都是世界坐标）：
//  1. 鼠标光标位置（直接跟随鼠标）
//  2. 网格对齐位置（对齐到格子中心）
//
// 拖拽模式支持：当 InputSystem 处于拖拽种植模式时，位置由 InputSystem 直接更新
// PositionComponent，本系统只需读取该位置并计算网格对齐。
func (s *PlantPreviewSystem) Update(deltaTime float64) {
	// 查询所有拥有 PlantPreviewComponent 和 PositionComponent 的实体
	entities := ecs.GetEntitiesWith2[
		*components.PlantPreviewComponent,
		*components.PositionComponent,
	](s.entityManager)

	// 如果没有预览实体，直接返回
	if len(entities) == 0 {
		return
	}

	// 检查是否处于拖拽种植模式
	dragManager := utils.GetDragManager()
	isDragging := dragManager.IsDragging() || dragManager.GetState() == utils.DragStateStarted

	if isDragging {
		// 拖拽模式：从预览实体的 PositionComponent 获取位置（由 InputSystem 更新）
		for _, entityID := range entities {
			pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
			if !ok {
				continue
			}
			// 使用实体位置作为鼠标世界坐标
			s.mouseWorldX = pos.X
			s.mouseWorldY = pos.Y
		}
	} else {
		// 非拖拽模式：从指针位置获取
		mouseX, mouseY := utils.GetPointerPosition()
		mouseScreenX := float64(mouseX)
		mouseScreenY := float64(mouseY)

		// 转换为世界坐标（X轴需要加上相机偏移，Y轴不变）
		s.mouseWorldX = mouseScreenX + s.gameState.CameraX
		s.mouseWorldY = mouseScreenY

		// 非拖拽模式下，更新 PositionComponent
		for _, entityID := range entities {
			pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
			if !ok {
				continue
			}
			pos.X = s.mouseWorldX
			pos.Y = s.mouseWorldY
		}
	}

	// 将世界坐标转换为屏幕坐标以计算网格位置
	mouseScreenX := int(s.mouseWorldX - s.gameState.CameraX)
	mouseScreenY := int(s.mouseWorldY)

	// 将鼠标屏幕坐标转换为网格坐标，并计算网格对齐位置
	col, row, isInGrid := utils.MouseToGridCoords(
		mouseScreenX, mouseScreenY,
		s.gameState.CameraX,
		config.GridWorldStartX, config.GridWorldStartY,
		config.GridColumns, config.GridRows,
		config.CellWidth, config.CellHeight,
	)

	s.isInGrid = isInGrid

	// Story 8.1: 如果在网格内，检查该行是否启用（教学关卡可能禁用部分行）
	if isInGrid && s.lawnGridSystem != nil {
		// 注意：row 是 0-based (0-4)，IsLaneEnabled 使用 1-based (1-5)
		lane := row + 1
		if !s.lawnGridSystem.IsLaneEnabled(lane) {
			// 该行未启用，视为不在网格内（不显示预览）
			s.isInGrid = false
		}
	}

	if s.isInGrid {
		// 在网格内，计算格子中心的屏幕坐标
		gridScreenX, gridScreenY := utils.GridToScreenCoords(
			col, row,
			s.gameState.CameraX,
			config.GridWorldStartX, config.GridWorldStartY,
			config.CellWidth, config.CellHeight,
		)
		// 转换为世界坐标
		s.gridAlignedWorldX = gridScreenX + s.gameState.CameraX
		s.gridAlignedWorldY = gridScreenY
	} else {
		// 不在网格内，网格对齐位置也设为鼠标位置（虽然不会被渲染）
		s.gridAlignedWorldX = s.mouseWorldX
		s.gridAlignedWorldY = s.mouseWorldY
	}
}

// GetPreviewPositions 返回预览图像的两个渲染位置（世界坐标）
// 返回值：
//   - mouseX, mouseY: 鼠标光标位置（世界坐标，用于渲染不透明图像）
//   - gridX, gridY: 网格对齐位置（世界坐标，用于渲染半透明预览图像）
//   - isInGrid: 鼠标是否在网格内（决定是否渲染半透明预览）
func (s *PlantPreviewSystem) GetPreviewPositions() (mouseX, mouseY, gridX, gridY float64, isInGrid bool) {
	return s.mouseWorldX, s.mouseWorldY, s.gridAlignedWorldX, s.gridAlignedWorldY, s.isInGrid
}
