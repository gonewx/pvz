package systems

import (
	"image/color"
	"reflect"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// PlantCardRenderSystem 负责渲染植物卡片
// 包括卡片图像和冷却覆盖层
type PlantCardRenderSystem struct {
	entityManager *ecs.EntityManager
	cardScale     float64 // 卡片缩放因子
}

// NewPlantCardRenderSystem 创建一个新的 PlantCardRenderSystem 实例
func NewPlantCardRenderSystem(em *ecs.EntityManager, cardScale float64) *PlantCardRenderSystem {
	return &PlantCardRenderSystem{
		entityManager: em,
		cardScale:     cardScale,
	}
}

// Draw 渲染所有植物卡片到屏幕
// 包括卡片图像和冷却覆盖层（卡片图片本身已包含阳光数值）
func (s *PlantCardRenderSystem) Draw(screen *ebiten.Image) {
	// 查询所有拥有 PlantCardComponent, PositionComponent, SpriteComponent 的实体
	entities := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.PlantCardComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.SpriteComponent{}),
	)

	for _, entityID := range entities {
		// 获取组件
		cardComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.PlantCardComponent{}))
		if !ok {
			continue
		}
		card := cardComp.(*components.PlantCardComponent)

		posComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.PositionComponent{}))
		if !ok {
			continue
		}
		pos := posComp.(*components.PositionComponent)

		spriteComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.SpriteComponent{}))
		if !ok {
			continue
		}
		sprite := spriteComp.(*components.SpriteComponent)

		// 获取卡片图像的实际尺寸
		var cardWidth, cardHeight float64
		if sprite.Image != nil {
			bounds := sprite.Image.Bounds()
			cardWidth = float64(bounds.Dx())
			cardHeight = float64(bounds.Dy())
		}

		// 计算缩放后的尺寸
		scaledWidth := cardWidth * s.cardScale
		scaledHeight := cardHeight * s.cardScale

		// 绘制卡片图像（应用缩放）
		if sprite.Image != nil {
			op := &ebiten.DrawImageOptions{}
			// 应用缩放
			op.GeoM.Scale(s.cardScale, s.cardScale)
			// 平移到目标位置
			op.GeoM.Translate(pos.X, pos.Y)
			screen.DrawImage(sprite.Image, op)
		}

		// 如果卡片不可用，绘制变暗效果
		if !card.IsAvailable && scaledHeight > 0 {
			if card.CurrentCooldown > 0 {
				// 冷却中：绘制黑色半透明覆盖层（从上往下渐进）
				progress := card.CurrentCooldown / card.CooldownTime
				coverHeight := scaledHeight * progress
				ebitenutil.DrawRect(screen, pos.X, pos.Y, scaledWidth, coverHeight,
					color.RGBA{0, 0, 0, 160}) // 半透明黑色
			} else {
				// 阳光不足：绘制全屏黑色半透明覆盖层（更淡）
				ebitenutil.DrawRect(screen, pos.X, pos.Y, scaledWidth, scaledHeight,
					color.RGBA{0, 0, 0, 100}) // 更淡的黑色，表示不可用
			}
		}

		// 注意：卡片图片本身已包含阳光数值，不需要额外绘制文本
	}
}
