package systems

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
)

// TestIsOccupied 测试占用检测功能
func TestIsOccupied(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewLawnGridSystem(em)

	// 创建草坪网格实体
	gridEntity := em.CreateEntity()
	gridComp := &components.LawnGridComponent{}
	em.AddComponent(gridEntity, gridComp)

	// 测试空格子
	if system.IsOccupied(gridEntity, 0, 0) {
		t.Error("Expected empty cell (0,0) to be unoccupied")
	}

	// 手动占用一个格子
	gridComp.Occupancy[2][3] = 999

	// 测试占用的格子
	if !system.IsOccupied(gridEntity, 3, 2) {
		t.Error("Expected occupied cell (3,2) to be occupied")
	}

	// 测试其他空格子
	if system.IsOccupied(gridEntity, 4, 2) {
		t.Error("Expected empty cell (4,2) to be unoccupied")
	}
}

// TestOccupyCell 测试标记格子占用
func TestOccupyCell(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewLawnGridSystem(em)

	// 创建草坪网格实体
	gridEntity := em.CreateEntity()
	gridComp := &components.LawnGridComponent{}
	em.AddComponent(gridEntity, gridComp)

	// 创建植物实体
	plantEntity := em.CreateEntity()

	// 测试占用空格子
	err := system.OccupyCell(gridEntity, 4, 2, plantEntity)
	if err != nil {
		t.Errorf("Failed to occupy empty cell: %v", err)
	}

	// 验证格子确实被占用
	if !system.IsOccupied(gridEntity, 4, 2) {
		t.Error("Cell should be occupied after OccupyCell")
	}

	// 验证存储的实体ID正确
	if gridComp.Occupancy[2][4] != plantEntity {
		t.Errorf("Expected entity ID %d, got %d", plantEntity, gridComp.Occupancy[2][4])
	}

	// 测试重复占用同一格子（应该失败）
	anotherPlant := em.CreateEntity()
	err = system.OccupyCell(gridEntity, 4, 2, anotherPlant)
	if err == nil {
		t.Error("Expected error when occupying already occupied cell")
	}
}

// TestReleaseCell 测试释放格子
func TestReleaseCell(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewLawnGridSystem(em)

	// 创建草坪网格实体
	gridEntity := em.CreateEntity()
	gridComp := &components.LawnGridComponent{}
	em.AddComponent(gridEntity, gridComp)

	// 先占用一个格子
	plantEntity := em.CreateEntity()
	err := system.OccupyCell(gridEntity, 5, 3, plantEntity)
	if err != nil {
		t.Fatalf("Failed to occupy cell: %v", err)
	}

	// 验证格子被占用
	if !system.IsOccupied(gridEntity, 5, 3) {
		t.Error("Cell should be occupied before release")
	}

	// 释放格子
	err = system.ReleaseCell(gridEntity, 5, 3)
	if err != nil {
		t.Errorf("Failed to release cell: %v", err)
	}

	// 验证格子被释放
	if system.IsOccupied(gridEntity, 5, 3) {
		t.Error("Cell should be unoccupied after release")
	}

	// 验证数组值为0
	if gridComp.Occupancy[3][5] != 0 {
		t.Errorf("Expected occupancy to be 0, got %d", gridComp.Occupancy[3][5])
	}
}

// TestBoundaryChecks 测试边界检查
func TestBoundaryChecks(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewLawnGridSystem(em)

	// 创建草坪网格实体
	gridEntity := em.CreateEntity()
	gridComp := &components.LawnGridComponent{}
	em.AddComponent(gridEntity, gridComp)

	plantEntity := em.CreateEntity()

	tests := []struct {
		name string
		col  int
		row  int
	}{
		{"负数列", -1, 2},
		{"负数行", 2, -1},
		{"列越界", 9, 2},
		{"行越界", 2, 5},
		{"完全越界", 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 测试 OccupyCell
			err := system.OccupyCell(gridEntity, tt.col, tt.row, plantEntity)
			if err == nil {
				t.Errorf("OccupyCell should return error for invalid position (%d, %d)", tt.col, tt.row)
			}

			// 测试 ReleaseCell
			err = system.ReleaseCell(gridEntity, tt.col, tt.row)
			if err == nil {
				t.Errorf("ReleaseCell should return error for invalid position (%d, %d)", tt.col, tt.row)
			}

			// 测试 IsOccupied（无效位置应返回true，表示"已占用"以防止种植）
			if !system.IsOccupied(gridEntity, tt.col, tt.row) {
				t.Errorf("IsOccupied should return true for invalid position (%d, %d)", tt.col, tt.row)
			}
		})
	}
}

// TestValidBoundaries 测试有效边界值
func TestValidBoundaries(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewLawnGridSystem(em)

	// 创建草坪网格实体
	gridEntity := em.CreateEntity()
	gridComp := &components.LawnGridComponent{}
	em.AddComponent(gridEntity, gridComp)

	plantEntity := em.CreateEntity()

	// 测试四个角的有效位置
	validCorners := []struct {
		name string
		col  int
		row  int
	}{
		{"左上角", 0, 0},
		{"右上角", 8, 0},
		{"左下角", 0, 4},
		{"右下角", 8, 4},
	}

	for _, corner := range validCorners {
		t.Run(corner.name, func(t *testing.T) {
			err := system.OccupyCell(gridEntity, corner.col, corner.row, plantEntity)
			if err != nil {
				t.Errorf("Should successfully occupy valid position %s (%d, %d): %v",
					corner.name, corner.col, corner.row, err)
			}

			// 清理
			system.ReleaseCell(gridEntity, corner.col, corner.row)
		})
	}
}

// TestMultipleOccupations 测试多个格子的占用
func TestMultipleOccupations(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewLawnGridSystem(em)

	// 创建草坪网格实体
	gridEntity := em.CreateEntity()
	gridComp := &components.LawnGridComponent{}
	em.AddComponent(gridEntity, gridComp)

	// 创建多个植物并占用不同格子
	positions := []struct {
		col int
		row int
	}{
		{0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 4},
	}

	plantEntities := make([]ecs.EntityID, len(positions))
	for i, pos := range positions {
		plantEntities[i] = em.CreateEntity()
		err := system.OccupyCell(gridEntity, pos.col, pos.row, plantEntities[i])
		if err != nil {
			t.Errorf("Failed to occupy cell (%d, %d): %v", pos.col, pos.row, err)
		}
	}

	// 验证所有格子都被正确占用
	for i, pos := range positions {
		if !system.IsOccupied(gridEntity, pos.col, pos.row) {
			t.Errorf("Cell (%d, %d) should be occupied", pos.col, pos.row)
		}
		if gridComp.Occupancy[pos.row][pos.col] != plantEntities[i] {
			t.Errorf("Cell (%d, %d) has wrong entity ID", pos.col, pos.row)
		}
	}

	// 释放所有格子
	for _, pos := range positions {
		err := system.ReleaseCell(gridEntity, pos.col, pos.row)
		if err != nil {
			t.Errorf("Failed to release cell (%d, %d): %v", pos.col, pos.row, err)
		}
	}

	// 验证所有格子都被释放
	for _, pos := range positions {
		if system.IsOccupied(gridEntity, pos.col, pos.row) {
			t.Errorf("Cell (%d, %d) should be unoccupied after release", pos.col, pos.row)
		}
	}
}
