package systems

import (
	"fmt"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
)

// LawnGridSystem 管理草坪网格的占用状态
// 负责跟踪哪些格子已被植物占用，并提供查询和更新方法
// Story 8.1: 支持行数限制，禁用特定行
type LawnGridSystem struct {
	entityManager *ecs.EntityManager
	EnabledLanes  []int // 启用的行列表（1-based），如 [1,2,3] 表示前3行可用
}

// NewLawnGridSystem 创建草坪网格系统
// 参数:
//   - em: EntityManager 实例
//   - enabledLanes: 启用的行列表（1-based），如 [1,2,3]。如果为空或 nil，默认所有5行启用
//
// 返回:
//   - *LawnGridSystem: 草坪网格系统实例
func NewLawnGridSystem(em *ecs.EntityManager, enabledLanes []int) *LawnGridSystem {
	// 如果未指定启用的行，默认所有行启用
	if len(enabledLanes) == 0 {
		enabledLanes = []int{1, 2, 3, 4, 5}
	}

	return &LawnGridSystem{
		entityManager: em,
		EnabledLanes:  enabledLanes,
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
	grid, ok := ecs.GetComponent[*components.LawnGridComponent](s.entityManager, gridEntity)
	if !ok {
		return true // 无法获取组件，视为已占用
	}

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
	grid, ok := ecs.GetComponent[*components.LawnGridComponent](s.entityManager, gridEntity)
	if !ok {
		return fmt.Errorf("failed to get LawnGridComponent from entity %d", gridEntity)
	}

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
	grid, ok := ecs.GetComponent[*components.LawnGridComponent](s.entityManager, gridEntity)
	if !ok {
		return fmt.Errorf("failed to get LawnGridComponent from entity %d", gridEntity)
	}

	// 清空占用状态
	grid.Occupancy[row][col] = 0
	return nil
}

// isValidGridPosition 检查网格位置是否有效
func (s *LawnGridSystem) isValidGridPosition(col, row int) bool {
	return col >= 0 && col < config.GridColumns && row >= 0 && row < config.GridRows
}

// IsLaneEnabled 检查指定行是否启用（Story 8.1）
// 参数:
//   - lane: 行索引（1-based），如 1 表示第一行
//
// 返回:
//   - bool: true 表示该行已启用，false 表示该行被禁用
func (s *LawnGridSystem) IsLaneEnabled(lane int) bool {
	// 如果未设置 EnabledLanes，默认所有行启用
	if len(s.EnabledLanes) == 0 {
		return lane >= 1 && lane <= config.GridRows
	}

	// 检查行是否在启用列表中
	for _, enabledLane := range s.EnabledLanes {
		if enabledLane == lane {
			return true
		}
	}
	return false
}
