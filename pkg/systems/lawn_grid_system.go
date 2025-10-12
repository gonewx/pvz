package systems

import (
	"fmt"
	"reflect"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
)

// LawnGridSystem 管理草坪网格的占用状态
// 负责跟踪哪些格子已被植物占用，并提供查询和更新方法
type LawnGridSystem struct {
	entityManager *ecs.EntityManager
}

// NewLawnGridSystem 创建草坪网格系统
func NewLawnGridSystem(em *ecs.EntityManager) *LawnGridSystem {
	return &LawnGridSystem{
		entityManager: em,
	}
}

// IsOccupied 检查指定格子是否已被占用
// 参数:
//   - gridEntity: 草坪网格实体ID
//   - col: 列索引 (0-8)
//   - row: 行索引 (0-4)
//
// 返回:
//   - bool: true 表示格子已被占用，false 表示格子为空
func (s *LawnGridSystem) IsOccupied(gridEntity ecs.EntityID, col, row int) bool {
	// 边界检查
	if !s.isValidGridPosition(col, row) {
		return true // 无效位置视为"已占用"，防止种植
	}

	// 获取 LawnGridComponent
	gridComp, ok := s.entityManager.GetComponent(gridEntity, reflect.TypeOf(&components.LawnGridComponent{}))
	if !ok {
		return true // 无法获取组件，视为已占用
	}

	grid := gridComp.(*components.LawnGridComponent)
	return grid.Occupancy[row][col] != 0
}

// OccupyCell 标记指定格子为被占用状态
// 参数:
//   - gridEntity: 草坪网格实体ID
//   - col: 列索引 (0-8)
//   - row: 行索引 (0-4)
//   - plantEntity: 占用该格子的植物实体ID
//
// 返回:
//   - error: 如果位置无效或格子已被占用，返回错误
func (s *LawnGridSystem) OccupyCell(gridEntity ecs.EntityID, col, row int, plantEntity ecs.EntityID) error {
	// 边界检查
	if !s.isValidGridPosition(col, row) {
		return fmt.Errorf("invalid grid position: col=%d, row=%d (valid range: col 0-8, row 0-4)", col, row)
	}

	// 获取 LawnGridComponent
	gridComp, ok := s.entityManager.GetComponent(gridEntity, reflect.TypeOf(&components.LawnGridComponent{}))
	if !ok {
		return fmt.Errorf("failed to get LawnGridComponent from entity %d", gridEntity)
	}

	grid := gridComp.(*components.LawnGridComponent)

	// 检查格子是否已被占用
	if grid.Occupancy[row][col] != 0 {
		return fmt.Errorf("grid cell (%d, %d) is already occupied by entity %d", col, row, grid.Occupancy[row][col])
	}

	// 标记为占用
	grid.Occupancy[row][col] = plantEntity
	return nil
}

// ReleaseCell 清空指定格子的占用状态
// 参数:
//   - gridEntity: 草坪网格实体ID
//   - col: 列索引 (0-8)
//   - row: 行索引 (0-4)
//
// 返回:
//   - error: 如果位置无效，返回错误
func (s *LawnGridSystem) ReleaseCell(gridEntity ecs.EntityID, col, row int) error {
	// 边界检查
	if !s.isValidGridPosition(col, row) {
		return fmt.Errorf("invalid grid position: col=%d, row=%d (valid range: col 0-8, row 0-4)", col, row)
	}

	// 获取 LawnGridComponent
	gridComp, ok := s.entityManager.GetComponent(gridEntity, reflect.TypeOf(&components.LawnGridComponent{}))
	if !ok {
		return fmt.Errorf("failed to get LawnGridComponent from entity %d", gridEntity)
	}

	grid := gridComp.(*components.LawnGridComponent)

	// 清空占用状态
	grid.Occupancy[row][col] = 0
	return nil
}

// isValidGridPosition 检查网格位置是否有效
func (s *LawnGridSystem) isValidGridPosition(col, row int) bool {
	return col >= 0 && col < config.GridColumns && row >= 0 && row < config.GridRows
}
