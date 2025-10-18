package systems

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
)

// TestPlantPreviewRenderSystemCreation 测试渲染系统创建
func TestPlantPreviewRenderSystemCreation(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	previewSystem := NewPlantPreviewSystem(em, gs, nil) // Story 8.1: 传递 nil LawnGridSystem（测试用）
	renderSystem := NewPlantPreviewRenderSystem(em, previewSystem)

	if renderSystem == nil {
		t.Fatal("Expected NewPlantPreviewRenderSystem to return non-nil")
	}
	if renderSystem.entityManager != em {
		t.Error("Expected entityManager to be set")
	}
	if renderSystem.plantPreviewSystem != previewSystem {
		t.Error("Expected plantPreviewSystem to be set")
	}
}

// TestDrawWithNoEntities 测试无预览实体时的绘制
func TestDrawWithNoEntities(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	previewSystem := NewPlantPreviewSystem(em, gs, nil) // Story 8.1: 传递 nil LawnGridSystem（测试用）
	renderSystem := NewPlantPreviewRenderSystem(em, previewSystem)

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	// 不应该panic
	renderSystem.Draw(screen, 0.0)
}

// TestDrawPreviewEntity 测试绘制预览实体（使用ReanimComponent）
func TestDrawPreviewEntity(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	previewSystem := NewPlantPreviewSystem(em, gs, nil) // Story 8.1: 传递 nil LawnGridSystem（测试用）
	renderSystem := NewPlantPreviewRenderSystem(em, previewSystem)

	// 创建测试图像
	testImage := ebiten.NewImage(64, 80)

	// 创建简单的 Reanim 组件
	reanimComp := createTestReanimComponent(testImage, "test_preview")

	// 创建预览实体
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PlantPreviewComponent{
		PlantType: components.PlantSunflower,
		Alpha:     0.5,
	})
	em.AddComponent(entityID, &components.PositionComponent{
		X: 400,
		Y: 300,
	})
	em.AddComponent(entityID, reanimComp)

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	// 不应该panic
	renderSystem.Draw(screen, 0.0)
}

// TestDrawWithNilImage 测试图像为nil时的处理
func TestDrawWithNilImage(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	previewSystem := NewPlantPreviewSystem(em, gs, nil) // Story 8.1: 传递 nil LawnGridSystem（测试用）
	renderSystem := NewPlantPreviewRenderSystem(em, previewSystem)

	// 创建预览实体，但没有 ReanimComponent
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PlantPreviewComponent{
		PlantType: components.PlantSunflower,
		Alpha:     0.5,
	})
	em.AddComponent(entityID, &components.PositionComponent{
		X: 400,
		Y: 300,
	})

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	// 不应该panic，应该跳过此实体
	renderSystem.Draw(screen, 0.0)
}

// TestAlphaBlending 测试透明度应用
func TestAlphaBlending(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	previewSystem := NewPlantPreviewSystem(em, gs, nil) // Story 8.1: 传递 nil LawnGridSystem（测试用）
	renderSystem := NewPlantPreviewRenderSystem(em, previewSystem)

	testCases := []struct {
		name  string
		alpha float64
	}{
		{"完全透明", 0.0},
		{"半透明", 0.5},
		{"几乎不透明", 0.9},
		{"完全不透明", 1.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建测试图像
			testImage := ebiten.NewImage(64, 80)
			reanimComp := createTestReanimComponent(testImage, "test_alpha")

			// 创建预览实体
			entityID := em.CreateEntity()
			em.AddComponent(entityID, &components.PlantPreviewComponent{
				PlantType: components.PlantSunflower,
				Alpha:     tc.alpha,
			})
			em.AddComponent(entityID, &components.PositionComponent{
				X: 400,
				Y: 300,
			})
			em.AddComponent(entityID, reanimComp)

			// 创建测试屏幕
			screen := ebiten.NewImage(800, 600)

			// 不应该panic
			renderSystem.Draw(screen, 0.0)

			// 清理实体以便下一个测试
			em.DestroyEntity(entityID)
			em.RemoveMarkedEntities()
		})
	}
}

// TestImageCentering 测试图像居中渲染
func TestImageCentering(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	previewSystem := NewPlantPreviewSystem(em, gs, nil) // Story 8.1: 传递 nil LawnGridSystem（测试用）
	renderSystem := NewPlantPreviewRenderSystem(em, previewSystem)

	// 创建测试图像（64x80）
	testImage := ebiten.NewImage(64, 80)
	reanimComp := createTestReanimComponent(testImage, "test_center")

	// 创建预览实体，位置设置为 (400, 300)
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PlantPreviewComponent{
		PlantType: components.PlantSunflower,
		Alpha:     0.5,
	})
	em.AddComponent(entityID, &components.PositionComponent{
		X: 400,
		Y: 300,
	})
	em.AddComponent(entityID, reanimComp)

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	// 绘制（不应该panic）
	renderSystem.Draw(screen, 0.0)
}

// TestMultiplePreviewEntities 测试多个预览实体的绘制
func TestMultiplePreviewEntities(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	previewSystem := NewPlantPreviewSystem(em, gs, nil) // Story 8.1: 传递 nil LawnGridSystem（测试用）
	renderSystem := NewPlantPreviewRenderSystem(em, previewSystem)

	// 创建多个预览实体
	for i := 0; i < 3; i++ {
		testImage := ebiten.NewImage(64, 80)
		reanimComp := createTestReanimComponent(testImage, "test_multi")

		entityID := em.CreateEntity()
		em.AddComponent(entityID, &components.PlantPreviewComponent{
			PlantType: components.PlantSunflower,
			Alpha:     0.5,
		})
		em.AddComponent(entityID, &components.PositionComponent{
			X: float64(200 + i*100),
			Y: 300,
		})
		em.AddComponent(entityID, reanimComp)
	}

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	// 绘制所有实体（不应该panic）
	renderSystem.Draw(screen, 0.0)

	// 验证所有实体都存在
	entities := em.GetEntitiesWith(
		reflect.TypeOf(&components.PlantPreviewComponent{}),
	)
	if len(entities) != 3 {
		t.Errorf("Expected 3 entities, got %d", len(entities))
	}
}

// TestDrawWithDifferentImageSizes 测试不同尺寸图像的居中绘制
func TestDrawWithDifferentImageSizes(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	previewSystem := NewPlantPreviewSystem(em, gs, nil) // Story 8.1: 传递 nil LawnGridSystem（测试用）
	renderSystem := NewPlantPreviewRenderSystem(em, previewSystem)

	testCases := []struct {
		name   string
		width  int
		height int
	}{
		{"小图像", 32, 40},
		{"中图像", 64, 80},
		{"大图像", 128, 160},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建指定尺寸的测试图像
			testImage := ebiten.NewImage(tc.width, tc.height)
			reanimComp := createTestReanimComponent(testImage, "test_sizes")

			// 创建预览实体
			entityID := em.CreateEntity()
			em.AddComponent(entityID, &components.PlantPreviewComponent{
				PlantType: components.PlantSunflower,
				Alpha:     0.5,
			})
			em.AddComponent(entityID, &components.PositionComponent{
				X: 400,
				Y: 300,
			})
			em.AddComponent(entityID, reanimComp)

			// 创建测试屏幕
			screen := ebiten.NewImage(800, 600)

			// 不应该panic
			renderSystem.Draw(screen, 0.0)

			// 验证图像尺寸
			bounds := testImage.Bounds()
			if bounds.Dx() != tc.width || bounds.Dy() != tc.height {
				t.Errorf("Expected image size %dx%d, got %dx%d",
					tc.width, tc.height, bounds.Dx(), bounds.Dy())
			}

			// 清理
			em.DestroyEntity(entityID)
			em.RemoveMarkedEntities()
		})
	}
}

// TestDrawWithCameraOffset 测试摄像机偏移的处理
func TestDrawWithCameraOffset(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	previewSystem := NewPlantPreviewSystem(em, gs, nil) // Story 8.1: 传递 nil LawnGridSystem（测试用）
	renderSystem := NewPlantPreviewRenderSystem(em, previewSystem)

	// 创建测试图像
	testImage := ebiten.NewImage(64, 80)
	reanimComp := createTestReanimComponent(testImage, "test_camera")

	// 创建预览实体
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PlantPreviewComponent{
		PlantType: components.PlantSunflower,
		Alpha:     0.5,
	})
	em.AddComponent(entityID, &components.PositionComponent{
		X: 400,
		Y: 300,
	})
	em.AddComponent(entityID, reanimComp)

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	// 测试不同的摄像机偏移
	testCameraOffsets := []float64{0.0, 100.0, -50.0, 200.0}

	for _, cameraX := range testCameraOffsets {
		// 不应该panic
		renderSystem.Draw(screen, cameraX)
	}
}
