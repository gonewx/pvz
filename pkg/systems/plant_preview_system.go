package systems

import (
	"math"
	"reflect"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
)

// 草坪网格参数
const (
	gridStartX  = 250.0 // 网格起始X坐标
	gridStartY  = 90.0  // 网格起始Y坐标
	gridColumns = 9     // 网格列数
	gridRows    = 5     // 网格行数
	cellWidth   = 80.0  // 每格宽度
	cellHeight  = 100.0 // 每格高度
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
	gridEndX := gridStartX + float64(gridColumns)*cellWidth
	gridEndY := gridStartY + float64(gridRows)*cellHeight

	return x >= gridStartX && x < gridEndX &&
		y >= gridStartY && y < gridEndY
}

// snapToGridCenter 将坐标对齐到最近的格子中心
func (s *PlantPreviewSystem) snapToGridCenter(x, y float64) (float64, float64) {
	// 计算列和行索引
	col := int(math.Floor((x - gridStartX) / cellWidth))
	row := int(math.Floor((y - gridStartY) / cellHeight))

	// 边界检查（防止越界）
	if col < 0 {
		col = 0
	} else if col >= gridColumns {
		col = gridColumns - 1
	}

	if row < 0 {
		row = 0
	} else if row >= gridRows {
		row = gridRows - 1
	}

	// 计算格子中心坐标
	centerX := gridStartX + float64(col)*cellWidth + cellWidth/2
	centerY := gridStartY + float64(row)*cellHeight + cellHeight/2

	return centerX, centerY
}
