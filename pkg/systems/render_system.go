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
// 渲染顺序：先渲染植物，再渲染阳光，确保阳光显示在植物上方
func (s *RenderSystem) Draw(screen *ebiten.Image) {
	// 查询所有拥有 PositionComponent 和 SpriteComponent 的实体
	entities := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.SpriteComponent{}),
	)

	// 第一遍：渲染植物（底层）
	for _, id := range entities {
		// 跳过植物卡片实体（它们由 PlantCardRenderSystem 专门渲染）
		if _, hasPlantCard := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PlantCardComponent{})); hasPlantCard {
			continue
		}

		// 跳过植物预览实体（它们由 PlantPreviewRenderSystem 专门渲染）
		if _, hasPlantPreview := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PlantPreviewComponent{})); hasPlantPreview {
			continue
		}

		// 只渲染植物
		_, isPlant := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PlantComponent{}))
		if !isPlant {
			continue // 跳过非植物实体
		}

		s.drawEntity(screen, id, true)
	}

	// 第二遍：渲染阳光和其他实体（上层）
	for _, id := range entities {
		// 跳过植物卡片实体
		if _, hasPlantCard := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PlantCardComponent{})); hasPlantCard {
			continue
		}

		// 跳过植物预览实体
		if _, hasPlantPreview := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PlantPreviewComponent{})); hasPlantPreview {
			continue
		}

		// 只渲染非植物实体（阳光等）
		_, isPlant := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PlantComponent{}))
		if isPlant {
			continue // 跳过植物（已经在第一遍渲染了）
		}

		s.drawEntity(screen, id, false)
	}
}

// drawEntity 绘制单个实体
func (s *RenderSystem) drawEntity(screen *ebiten.Image, id ecs.EntityID, isPlant bool) {
	// 获取组件
	posComp, _ := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PositionComponent{}))
	spriteComp, _ := s.entityManager.GetComponent(id, reflect.TypeOf(&components.SpriteComponent{}))

	// 类型断言
	pos := posComp.(*components.PositionComponent)
	sprite := spriteComp.(*components.SpriteComponent)

	// 如果没有图片,跳过
	if sprite.Image == nil {
		return
	}

	// 获取图像尺寸
	bounds := sprite.Image.Bounds()
	imageWidth := float64(bounds.Dx())
	imageHeight := float64(bounds.Dy())

	var drawX, drawY float64
	if isPlant {
		// 植物实体：图像中心对齐到位置坐标
		drawX = pos.X - imageWidth/2
		drawY = pos.Y - imageHeight/2
	} else {
		// 其他实体（如阳光）：图像左上角对齐到位置坐标
		drawX = pos.X
		drawY = pos.Y
	}

	// 创建绘制选项
	op := &ebiten.DrawImageOptions{}

	// 设置位置平移
	op.GeoM.Translate(drawX, drawY)

	// 绘制到屏幕
	screen.DrawImage(sprite.Image, op)
}
