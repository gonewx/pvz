package systems

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
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

// TestIsInGrid 测试网格边界检测
func TestIsInGrid(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	system := NewPlantPreviewSystem(em, gs)

	testCases := []struct {
		name     string
		x, y     float64
		expected bool
	}{
		{"网格内中心", 500, 300, true},
		{"网格左上角", 250, 90, true},
		{"网格右下角边界内", 969, 589, true},
		{"网格左边界外", 249, 300, false},
		{"网格右边界外", 970, 300, false},
		{"网格上边界外", 500, 89, false},
		{"网格下边界外", 500, 590, false},
		{"网格外左上", 100, 50, false},
		{"网格外右下", 1000, 600, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := system.isInGrid(tc.x, tc.y)
			if result != tc.expected {
				t.Errorf("isInGrid(%.1f, %.1f) = %v, expected %v", tc.x, tc.y, result, tc.expected)
			}
		})
	}
}

// TestSnapToGridCenter 测试对齐到格子中心
func TestSnapToGridCenter(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	system := NewPlantPreviewSystem(em, gs)

	testCases := []struct {
		name                 string
		mouseX, mouseY       float64
		expectedX, expectedY float64
	}{
		{
			name:   "第一格(0,0)中心",
			mouseX: 250, mouseY: 90,
			expectedX: 290, expectedY: 140, // 250 + 40, 90 + 50
		},
		{
			name:   "第二格(1,0)任意点",
			mouseX: 350, mouseY: 100,
			expectedX: 370, expectedY: 140, // 250 + 1*80 + 40, 90 + 0*100 + 50
		},
		{
			name:   "第二行第二格(1,1)",
			mouseX: 350, mouseY: 250,
			expectedX: 370, expectedY: 240, // 250 + 1*80 + 40, 90 + 1*100 + 50
		},
		{
			name:   "最后一格(8,4)",
			mouseX: 900, mouseY: 500,
			expectedX: 930, expectedY: 540, // 250 + 8*80 + 40, 90 + 4*100 + 50
		},
		{
			name:   "格子边界",
			mouseX: 330, mouseY: 190,
			expectedX: 370, expectedY: 240, // 应对齐到 (1,1)
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			centerX, centerY := system.snapToGridCenter(tc.mouseX, tc.mouseY)
			if centerX != tc.expectedX || centerY != tc.expectedY {
				t.Errorf("snapToGridCenter(%.1f, %.1f) = (%.1f, %.1f), expected (%.1f, %.1f)",
					tc.mouseX, tc.mouseY, centerX, centerY, tc.expectedX, tc.expectedY)
			}
		})
	}
}

// TestSnapToGridCenterBoundary 测试网格边界情况的对齐
func TestSnapToGridCenterBoundary(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	system := NewPlantPreviewSystem(em, gs)

	// 测试负坐标（应该对齐到第一格）
	centerX, centerY := system.snapToGridCenter(-100, -100)
	expectedX, expectedY := 290.0, 140.0 // 第一格中心
	if centerX != expectedX || centerY != expectedY {
		t.Errorf("负坐标应对齐到第一格, got (%.1f, %.1f), expected (%.1f, %.1f)",
			centerX, centerY, expectedX, expectedY)
	}

	// 测试超大坐标（应该对齐到最后一格）
	centerX, centerY = system.snapToGridCenter(10000, 10000)
	expectedX, expectedY = 930.0, 540.0 // 最后一格中心 (8,4)
	if centerX != expectedX || centerY != expectedY {
		t.Errorf("超大坐标应对齐到最后一格, got (%.1f, %.1f), expected (%.1f, %.1f)",
			centerX, centerY, expectedX, expectedY)
	}
}

// TestUpdateWithNoEntities 测试无预览实体时的更新
func TestUpdateWithNoEntities(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	system := NewPlantPreviewSystem(em, gs)

	// 不应该panic
	system.Update(0.016)
}

// TestUpdatePreviewPosition 测试预览位置更新
// 注意：此测试需要模拟鼠标输入，在单元测试中可能难以实现
// 这里仅测试系统的基本功能，实际行为需要通过集成测试验证
func TestUpdatePreviewPosition(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	system := NewPlantPreviewSystem(em, gs)

	// 创建预览实体
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PlantPreviewComponent{
		PlantType: components.PlantSunflower,
		Alpha:     0.5,
	})
	em.AddComponent(entityID, &components.PositionComponent{
		X: 0,
		Y: 0,
	})

	// 调用 Update（注意：在实际游戏中鼠标位置由 ebiten 提供，测试中无法模拟）
	system.Update(0.016)

	// 验证实体仍然存在且没有错误
	posComp, exists := em.GetComponent(entityID, reflect.TypeOf(&components.PositionComponent{}))
	if !exists {
		t.Fatal("Expected PositionComponent to still exist after Update")
	}

	pos := posComp.(*components.PositionComponent)
	// 位置会根据模拟的鼠标位置更新，这里只验证组件存在
	_ = pos
}
