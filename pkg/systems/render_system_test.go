package systems

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestRenderSystemQuery(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)

	// 创建测试图像
	testImage := ebiten.NewImage(10, 10)

	// 创建拥有位置和精灵组件的实体
	id1 := em.CreateEntity()
	em.AddComponent(id1, &components.PositionComponent{X: 100, Y: 200})
	em.AddComponent(id1, &components.SpriteComponent{Image: testImage})

	// 创建只有位置组件的实体(不应被渲染)
	id2 := em.CreateEntity()
	em.AddComponent(id2, &components.PositionComponent{X: 50, Y: 50})

	// 验证系统能正确查询
	entities := em.GetEntitiesWith(
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.SpriteComponent{}),
	)

	if len(entities) != 1 {
		t.Errorf("Expected 1 renderable entity, got %d", len(entities))
	}

	if len(entities) > 0 && entities[0] != id1 {
		t.Error("Should find id1 as renderable entity")
	}

	// Draw 应该不会崩溃
	screen := ebiten.NewImage(800, 600)
	system.Draw(screen, 0.0) // cameraX = 0
}

func TestRenderSystemWithNilImage(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)

	// 创建有位置但图片为nil的实体
	id := em.CreateEntity()
	em.AddComponent(id, &components.PositionComponent{X: 100, Y: 200})
	em.AddComponent(id, &components.SpriteComponent{Image: nil})

	// Draw 应该跳过nil图片而不崩溃
	screen := ebiten.NewImage(800, 600)
	system.Draw(screen, 0.0) // cameraX = 0

	// 如果没有panic,测试通过
}

func TestRenderSystemMultipleEntities(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)

	// 创建测试图像
	testImage := ebiten.NewImage(10, 10)

	// 创建多个可渲染实体
	for i := 0; i < 5; i++ {
		id := em.CreateEntity()
		em.AddComponent(id, &components.PositionComponent{X: float64(i * 100), Y: 100})
		em.AddComponent(id, &components.SpriteComponent{Image: testImage})
	}

	// 验证查询结果
	entities := em.GetEntitiesWith(
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.SpriteComponent{}),
	)

	if len(entities) != 5 {
		t.Errorf("Expected 5 renderable entities, got %d", len(entities))
	}

	// Draw 应该不会崩溃
	screen := ebiten.NewImage(800, 600)
	system.Draw(screen, 0.0) // cameraX = 0
}

func TestRenderSystemEmptyScene(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)

	// 没有实体
	screen := ebiten.NewImage(800, 600)

	// Draw 应该不会崩溃
	system.Draw(screen, 0.0) // cameraX = 0
}
