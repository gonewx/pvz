package systems

import (
	"reflect"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// PlantPreviewRenderSystem 渲染植物预览的半透明图像
type PlantPreviewRenderSystem struct {
	entityManager *ecs.EntityManager
}

// NewPlantPreviewRenderSystem 创建植物预览渲染系统
func NewPlantPreviewRenderSystem(em *ecs.EntityManager) *PlantPreviewRenderSystem {
	return &PlantPreviewRenderSystem{
		entityManager: em,
	}
}

// Draw 渲染所有植物预览实体
func (s *PlantPreviewRenderSystem) Draw(screen *ebiten.Image) {
	// 查询所有拥有 PlantPreviewComponent, PositionComponent, SpriteComponent 的实体
	entities := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.PlantPreviewComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.SpriteComponent{}),
	)

	for _, entityID := range entities {
		// 获取组件
		previewComp, _ := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.PlantPreviewComponent{}))
		posComp, _ := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.PositionComponent{}))
		spriteComp, _ := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.SpriteComponent{}))

		preview := previewComp.(*components.PlantPreviewComponent)
		pos := posComp.(*components.PositionComponent)
		sprite := spriteComp.(*components.SpriteComponent)

		// 如果没有图像，跳过
		if sprite.Image == nil {
			continue
		}

		// 获取图像尺寸
		bounds := sprite.Image.Bounds()
		imageWidth := float64(bounds.Dx())
		imageHeight := float64(bounds.Dy())

		// 计算绘制位置（图像中心对齐到位置坐标）
		drawX := pos.X - imageWidth/2
		drawY := pos.Y - imageHeight/2

		// 创建绘制选项
		opts := &ebiten.DrawImageOptions{}

		// 设置位移
		opts.GeoM.Translate(drawX, drawY)

		// 设置透明度
		opts.ColorScale.ScaleAlpha(float32(preview.Alpha))

		// 绘制图像
		screen.DrawImage(sprite.Image, opts)
	}
}



