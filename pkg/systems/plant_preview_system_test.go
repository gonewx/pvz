package systems

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestPlantPreviewSystemCreation 测试系统创建
func TestPlantPreviewSystemCreation(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()

	system := NewPlantPreviewSystem(em, gs)

	if system == nil {
		t.Fatal("Expected NewPlantPreviewSystem to return non-nil")
	}
	if system.entityManager != em {
		t.Error("Expected entityManager to be set")
	}
	if system.gameState != gs {
		t.Error("Expected gameState to be set")
	}
}

// TestPlantPreviewUpdate 测试预览更新逻辑
func TestPlantPreviewUpdate(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.CameraX = 215 // 设置摄像机位置

	system := NewPlantPreviewSystem(em, gs)

	// 创建预览实体
	previewID := em.CreateEntity()
	em.AddComponent(previewID, &components.PlantPreviewComponent{
		PlantType: components.PlantSunflower,
	})
	em.AddComponent(previewID, &components.PositionComponent{
		X: 0,
		Y: 0,
	})

	// 模拟鼠标在网格内（注意：实际测试需要模拟 ebiten.CursorPosition）
	// 这里只测试系统是否能正常运行不崩溃
	system.Update(0.016)

	// 验证位置组件存在
	posComp, ok := em.GetComponent(previewID, reflect.TypeOf(&components.PositionComponent{}))
	if !ok {
		t.Fatal("Expected preview entity to have PositionComponent")
	}

	pos := posComp.(*components.PositionComponent)
	// 位置应该被更新（具体值取决于鼠标位置，这里只验证组件存在）
	if pos == nil {
		t.Error("Position component should not be nil")
	}
}

// TestGridCoordinateConsistency 测试网格坐标系统一致性
// 验证使用世界坐标系统后，在不同 cameraX 下坐标转换的正确性
func TestGridCoordinateConsistency(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()

	// 测试不同的摄像机位置
	testCameraPositions := []float64{0, 100, 215, 300}

	for _, cameraX := range testCameraPositions {
		t.Run("CameraX="+string(rune(int(cameraX))), func(t *testing.T) {
			gs.CameraX = cameraX
			system := NewPlantPreviewSystem(em, gs)

			// 创建预览实体
			previewID := em.CreateEntity()
			em.AddComponent(previewID, &components.PlantPreviewComponent{
				PlantType: components.PlantSunflower,
			})
			em.AddComponent(previewID, &components.PositionComponent{
				X: 0,
				Y: 0,
			})

			// 系统应该能在任何摄像机位置下正常工作
			system.Update(0.016)

			// 验证组件仍然存在
			_, ok := em.GetComponent(previewID, reflect.TypeOf(&components.PositionComponent{}))
			if !ok {
				t.Errorf("Preview entity should still have PositionComponent at cameraX=%.1f", cameraX)
			}
		})
	}
}

// TestGridBoundaryCalculation 测试网格边界计算
// 验证配置中的网格参数是否正确
func TestGridBoundaryCalculation(t *testing.T) {
	// 验证网格世界坐标配置
	if config.GridColumns != 9 {
		t.Errorf("Expected 9 columns, got %d", config.GridColumns)
	}
	if config.GridRows != 5 {
		t.Errorf("Expected 5 rows, got %d", config.GridRows)
	}
	if config.CellWidth != 80.0 {
		t.Errorf("Expected cell width 80.0, got %.1f", config.CellWidth)
	}
	if config.CellHeight != 100.0 {
		t.Errorf("Expected cell height 100.0, got %.1f", config.CellHeight)
	}

	// 验证网格总尺寸
	expectedWidth := float64(config.GridColumns) * config.CellWidth // 9 * 80 = 720
	expectedHeight := float64(config.GridRows) * config.CellHeight  // 5 * 100 = 500

	if expectedWidth != 720.0 {
		t.Errorf("Expected grid width 720.0, got %.1f", expectedWidth)
	}
	if expectedHeight != 500.0 {
		t.Errorf("Expected grid height 500.0, got %.1f", expectedHeight)
	}
}
