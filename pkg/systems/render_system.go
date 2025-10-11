package systems

import (
	"reflect"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// RenderSystem 管理所有实体的渲染
type RenderSystem struct {
	entityManager *ecs.EntityManager
}

// NewRenderSystem 创建一个新的渲染系统
func NewRenderSystem(em *ecs.EntityManager) *RenderSystem {
	return &RenderSystem{
		entityManager: em,
	}
}

// Draw 绘制所有拥有位置和精灵组件的实体
func (s *RenderSystem) Draw(screen *ebiten.Image) {
	// 查询所有拥有 PositionComponent 和 SpriteComponent 的实体
	entities := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.SpriteComponent{}),
	)

	for _, id := range entities {
		// 跳过植物卡片实体（它们由 PlantCardRenderSystem 专门渲染）
		if _, hasPlantCard := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PlantCardComponent{})); hasPlantCard {
			continue
		}

		// 获取组件
		posComp, _ := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PositionComponent{}))
		spriteComp, _ := s.entityManager.GetComponent(id, reflect.TypeOf(&components.SpriteComponent{}))

		// 类型断言
		pos := posComp.(*components.PositionComponent)
		sprite := spriteComp.(*components.SpriteComponent)

		// 如果没有图片,跳过
		if sprite.Image == nil {
			continue
		}

		// 创建绘制选项
		op := &ebiten.DrawImageOptions{}

		// 设置位置平移
		op.GeoM.Translate(pos.X, pos.Y)

		// 绘制到屏幕
		screen.DrawImage(sprite.Image, op)
	}
}
