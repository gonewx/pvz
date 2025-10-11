package systems

import (
	"math"
	"reflect"

	"github.com/decker502/pvz/pkg/components"
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

		// 判断鼠标是否在草坪网格内
		if s.isInGrid(float64(mouseX), float64(mouseY)) {
			// 在网格内，对齐到格子中心
			centerX, centerY := s.snapToGridCenter(float64(mouseX), float64(mouseY))
			pos.X = centerX
			pos.Y = centerY
		} else {
			// 不在网格内，直接跟随鼠标
			pos.X = float64(mouseX)
			pos.Y = float64(mouseY)
		}
	}
}

// isInGrid 判断坐标是否在草坪网格内
func (s *PlantPreviewSystem) isInGrid(x, y float64) bool {
	gridEndX := utils.GridStartX + float64(utils.GridColumns)*utils.CellWidth
	gridEndY := utils.GridStartY + float64(utils.GridRows)*utils.CellHeight

	return x >= utils.GridStartX && x < gridEndX &&
		y >= utils.GridStartY && y < gridEndY
}

// snapToGridCenter 将坐标对齐到最近的格子中心
func (s *PlantPreviewSystem) snapToGridCenter(x, y float64) (float64, float64) {
	// 计算列和行索引
	col := int(math.Floor((x - utils.GridStartX) / utils.CellWidth))
	row := int(math.Floor((y - utils.GridStartY) / utils.CellHeight))

	// 边界检查（防止越界）
	if col < 0 {
		col = 0
	} else if col >= utils.GridColumns {
		col = utils.GridColumns - 1
	}

	if row < 0 {
		row = 0
	} else if row >= utils.GridRows {
		row = utils.GridRows - 1
	}

	// 计算格子中心坐标
	centerX := utils.GridStartX + float64(col)*utils.CellWidth + utils.CellWidth/2
	centerY := utils.GridStartY + float64(row)*utils.CellHeight + utils.CellHeight/2

	return centerX, centerY
}
