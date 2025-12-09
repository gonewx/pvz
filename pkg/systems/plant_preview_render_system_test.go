package systems

import (
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
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
	previewSystem := NewPlantPreviewSystem(em, gs, nil)
	renderSystem := NewPlantPreviewRenderSystem(em, previewSystem)

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	// 不应该panic
	renderSystem.Draw(screen, 0.0)
}

// TestDrawPreviewEntity 测试绘制预览实体（使用静态SpriteComponent）
func TestDrawPreviewEntity(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	previewSystem := NewPlantPreviewSystem(em, gs, nil)
	renderSystem := NewPlantPreviewRenderSystem(em, previewSystem)

	// 创建一个预览实体（使用静态图像）
	entityID := em.CreateEntity()

	// 添加位置组件
	ecs.AddComponent(em, entityID, &components.PositionComponent{
		X: 400,
		Y: 300,
	})

	// 创建一个测试图像
	testImage := ebiten.NewImage(50, 50)

	// 添加精灵组件
	ecs.AddComponent(em, entityID, &components.SpriteComponent{
		Image: testImage,
	})

	// 添加预览组件
	ecs.AddComponent(em, entityID, &components.PlantPreviewComponent{
		PlantType: components.PlantSunflower,
		Alpha:     0.5,
	})

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	// 不应该panic
	renderSystem.Draw(screen, 0.0)
}

// TestDrawMultiplePreviewEntities 测试绘制多个预览实体
func TestDrawMultiplePreviewEntities(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	previewSystem := NewPlantPreviewSystem(em, gs, nil)
	renderSystem := NewPlantPreviewRenderSystem(em, previewSystem)

	// 创建测试图像
	testImage := ebiten.NewImage(50, 50)

	// 创建两个预览实体
	for i := 0; i < 2; i++ {
		entityID := em.CreateEntity()

		ecs.AddComponent(em, entityID, &components.PositionComponent{
			X: float64(100 * (i + 1)),
			Y: 300,
		})

		ecs.AddComponent(em, entityID, &components.SpriteComponent{
			Image: testImage,
		})

		ecs.AddComponent(em, entityID, &components.PlantPreviewComponent{
			PlantType: components.PlantPeashooter,
			Alpha:     0.5,
		})
	}

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	// 不应该panic
	renderSystem.Draw(screen, 0.0)
}

// TestDrawWithCameraOffset 测试带摄像机偏移的绘制
func TestDrawWithCameraOffset(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	previewSystem := NewPlantPreviewSystem(em, gs, nil)
	renderSystem := NewPlantPreviewRenderSystem(em, previewSystem)

	// 创建测试图像
	testImage := ebiten.NewImage(50, 50)

	// 创建预览实体
	entityID := em.CreateEntity()

	ecs.AddComponent(em, entityID, &components.PositionComponent{
		X: 400,
		Y: 300,
	})

	ecs.AddComponent(em, entityID, &components.SpriteComponent{
		Image: testImage,
	})

	ecs.AddComponent(em, entityID, &components.PlantPreviewComponent{
		PlantType: components.PlantWallnut,
		Alpha:     0.5,
	})

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	// 测试不同的摄像机偏移值
	cameraOffsets := []float64{0.0, 100.0, -100.0, 500.0}
	for _, cameraX := range cameraOffsets {
		// 不应该panic
		renderSystem.Draw(screen, cameraX)
	}
}

// TestDrawWithMissingComponents 测试缺少组件时的处理
func TestDrawWithMissingComponents(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	previewSystem := NewPlantPreviewSystem(em, gs, nil)
	renderSystem := NewPlantPreviewRenderSystem(em, previewSystem)

	// 创建一个只有部分组件的实体（应该被跳过）
	entityID := em.CreateEntity()

	ecs.AddComponent(em, entityID, &components.PositionComponent{
		X: 400,
		Y: 300,
	})

	ecs.AddComponent(em, entityID, &components.PlantPreviewComponent{
		PlantType: components.PlantCherryBomb,
		Alpha:     0.5,
	})
	// 故意不添加 SpriteComponent

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	// 不应该panic（系统应该优雅地处理缺失的组件）
	renderSystem.Draw(screen, 0.0)
}
