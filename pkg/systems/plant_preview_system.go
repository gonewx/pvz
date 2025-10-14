package systems

import (
	"reflect"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// PlantPreviewSystem 更新植物预览的位置
// 为实现双图像预览（光标处不透明 + 格子处半透明），系统会计算并存储两个位置：
//   1. 鼠标光标位置（mouseWorldX, mouseWorldY）- 用于渲染不透明图像
//   2. 网格对齐位置（gridAlignedWorldX, gridAlignedWorldY）- 用于渲染半透明预览图像
// 注意：所有位置都使用世界坐标系统
type PlantPreviewSystem struct {
	entityManager *ecs.EntityManager
	gameState     *game.GameState

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
func NewPlantPreviewSystem(em *ecs.EntityManager, gs *game.GameState) *PlantPreviewSystem {
	return &PlantPreviewSystem{
		entityManager: em,
		gameState:     gs,
	}
}

// Update 更新预览实体的位置
// 计算两个位置供渲染使用（都是世界坐标）：
//   1. 鼠标光标位置（直接跟随鼠标）
//   2. 网格对齐位置（对齐到格子中心）
func (s *PlantPreviewSystem) Update(deltaTime float64) {
	// 查询所有拥有 PlantPreviewComponent 和 PositionComponent 的实体
	entities := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.PlantPreviewComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
	)

	// 如果没有预览实体，直接返回
	if len(entities) == 0 {
		return
	}

	// 获取鼠标位置（屏幕坐标）
	mouseX, mouseY := ebiten.CursorPosition()
	mouseScreenX := float64(mouseX)
	mouseScreenY := float64(mouseY)

	// 转换为世界坐标（X轴需要加上相机偏移，Y轴不变）
	s.mouseWorldX = mouseScreenX + s.gameState.CameraX
	s.mouseWorldY = mouseScreenY

	// 将鼠标屏幕坐标转换为网格坐标，并计算网格对齐位置
	col, row, isInGrid := utils.MouseToGridCoords(
		mouseX, mouseY,
		s.gameState.CameraX,
		config.GridWorldStartX, config.GridWorldStartY,
		config.GridColumns, config.GridRows,
		config.CellWidth, config.CellHeight,
	)

	s.isInGrid = isInGrid

	if isInGrid {
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

	// 注意：虽然渲染系统通过 GetPreviewPositions() 获取位置，
	// 但我们仍需要更新 PositionComponent 以保持实体状态一致
	// （其他系统可能依赖 PositionComponent）
	for _, entityID := range entities {
		posComp, _ := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.PositionComponent{}))
		pos := posComp.(*components.PositionComponent)
		// 使用鼠标的世界坐标更新实体的基础位置
		pos.X = s.mouseWorldX
		pos.Y = s.mouseWorldY
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
