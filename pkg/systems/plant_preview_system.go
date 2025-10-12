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
// 使预览跟随鼠标移动，并在草坪网格内自动对齐到格子中心
type PlantPreviewSystem struct {
	entityManager *ecs.EntityManager
	gameState     *game.GameState
}

// NewPlantPreviewSystem 创建植物预览系统
func NewPlantPreviewSystem(em *ecs.EntityManager, gs *game.GameState) *PlantPreviewSystem {
	return &PlantPreviewSystem{
		entityManager: em,
		gameState:     gs,
	}
}

// Update 更新预览实体的位置
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

	// 获取鼠标位置
	mouseX, mouseY := ebiten.CursorPosition()

	for _, entityID := range entities {
		// 获取位置组件
		posComp, _ := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.PositionComponent{}))
		pos := posComp.(*components.PositionComponent)

		// 将鼠标屏幕坐标转换为网格坐标
		col, row, isInGrid := utils.MouseToGridCoords(
			mouseX, mouseY,
			s.gameState.CameraX,
			config.GridWorldStartX, config.GridWorldStartY,
			config.GridColumns, config.GridRows,
			config.CellWidth, config.CellHeight,
		)

		if isInGrid {
			// 在网格内，对齐到格子中心（转换为屏幕坐标）
			centerX, centerY := utils.GridToScreenCoords(
				col, row,
				s.gameState.CameraX,
				config.GridWorldStartX, config.GridWorldStartY,
				config.CellWidth, config.CellHeight,
			)
			pos.X = centerX
			pos.Y = centerY
		} else {
			// 不在网格内，直接跟随鼠标
			pos.X = float64(mouseX)
			pos.Y = float64(mouseY)
		}
	}
}


