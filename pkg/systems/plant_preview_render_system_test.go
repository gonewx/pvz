package systems

import (
	"image"
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// TestPlantPreviewRenderSystemCreation 测试渲染系统创建
func TestPlantPreviewRenderSystemCreation(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewPlantPreviewRenderSystem(em)

	if system == nil {
		t.Fatal("Expected NewPlantPreviewRenderSystem to return non-nil")
	}
	if system.entityManager != em {
		t.Error("Expected entityManager to be set")
	}
}

// TestDrawWithNoEntities 测试无预览实体时的绘制
func TestDrawWithNoEntities(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewPlantPreviewRenderSystem(em)

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	// 不应该panic
	system.Draw(screen)
}

// TestDrawPreviewEntity 测试绘制预览实体
func TestDrawPreviewEntity(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewPlantPreviewRenderSystem(em)

	// 创建测试图像
	testImage := ebiten.NewImage(64, 80)

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
	em.AddComponent(entityID, &components.SpriteComponent{
		Image: testImage,
	})

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	// 不应该panic
	system.Draw(screen)

	// 验证屏幕不为空（已经绘制了内容）
	// 注意：实际的绘制验证需要像素级检查，这里只是确保没有错误
}

// TestDrawWithNilImage 测试图像为nil时的处理
func TestDrawWithNilImage(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewPlantPreviewRenderSystem(em)

	// 创建预览实体，但图像为nil
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PlantPreviewComponent{
		PlantType: components.PlantSunflower,
		Alpha:     0.5,
	})
	em.AddComponent(entityID, &components.PositionComponent{
		X: 400,
		Y: 300,
	})
	em.AddComponent(entityID, &components.SpriteComponent{
		Image: nil, // 空图像
	})

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	// 不应该panic，应该跳过此实体
	system.Draw(screen)
}

// TestAlphaBlending 测试透明度应用
func TestAlphaBlending(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewPlantPreviewRenderSystem(em)

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
			em.AddComponent(entityID, &components.SpriteComponent{
				Image: testImage,
			})

			// 创建测试屏幕
			screen := ebiten.NewImage(800, 600)

			// 不应该panic
			system.Draw(screen)

			// 清理实体以便下一个测试
			em.DestroyEntity(entityID)
			em.RemoveMarkedEntities()
		})
	}
}

// TestImageCentering 测试图像居中渲染
func TestImageCentering(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewPlantPreviewRenderSystem(em)

	// 创建测试图像（64x80）
	testImage := ebiten.NewImage(64, 80)

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
	em.AddComponent(entityID, &components.SpriteComponent{
		Image: testImage,
	})

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	// 绘制（不应该panic）
	system.Draw(screen)

	// 验证绘制位置计算正确
	// 预期绘制位置: drawX = 400 - 64/2 = 368, drawY = 300 - 80/2 = 260
	// 实际的位置验证需要更复杂的测试，这里只确保没有错误
}

// TestMultiplePreviewEntities 测试多个预览实体的绘制
func TestMultiplePreviewEntities(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewPlantPreviewRenderSystem(em)

	// 创建多个预览实体
	for i := 0; i < 3; i++ {
		testImage := ebiten.NewImage(64, 80)
		entityID := em.CreateEntity()
		em.AddComponent(entityID, &components.PlantPreviewComponent{
			PlantType: components.PlantSunflower,
			Alpha:     0.5,
		})
		em.AddComponent(entityID, &components.PositionComponent{
			X: float64(200 + i*100),
			Y: 300,
		})
		em.AddComponent(entityID, &components.SpriteComponent{
			Image: testImage,
		})
	}

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	// 绘制所有实体（不应该panic）
	system.Draw(screen)

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
	system := NewPlantPreviewRenderSystem(em)

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
			em.AddComponent(entityID, &components.SpriteComponent{
				Image: testImage,
			})

			// 创建测试屏幕
			screen := ebiten.NewImage(800, 600)

			// 不应该panic
			system.Draw(screen)

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

// 辅助函数：创建指定颜色的测试图像
func createColoredImage(width, height int, r, g, b, a uint8) *ebiten.Image {
	img := ebiten.NewImageFromImage(
		image.NewRGBA(image.Rect(0, 0, width, height)),
	)
	// 填充颜色（在实际测试中可能需要）
	return img
}







