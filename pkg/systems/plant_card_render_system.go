package systems

import (
	"fmt"
	"image/color"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// PlantCardRenderSystem 负责渲染植物卡片
// Story 6.3: 多层渲染架构 - 背景 + 植物图标 + 阳光数字 + 效果遮罩
type PlantCardRenderSystem struct {
	entityManager    *ecs.EntityManager
	cardScale        float64          // 卡片背景缩放因子
	plantIconScale   float64          // 植物图标缩放因子（可配置）
	plantIconOffsetY float64          // 植物图标垂直偏移（距离顶部的像素，可配置）
	sunTextOffsetY   float64          // 阳光数字垂直偏移（距离底部的像素，可配置）
	sunFont          *text.GoTextFace // 阳光数字字体（可选，如果未提供则使用调试文本）
}

// NewPlantCardRenderSystem 创建一个新的 PlantCardRenderSystem 实例
// 参数:
//   - em: 实体管理器
//   - cardScale: 卡片背景缩放因子（如 0.54）
//   - plantIconScale: 植物图标缩放因子（如 0.7 表示缩小到 70%）
//   - plantIconOffsetY: 植物图标垂直偏移（像素，如 3.0）
//   - sunTextOffsetY: 阳光数字距离底部的偏移（像素，如 15.0）
//   - sunFont: 阳光数字字体（可选，如果为 nil 则使用调试文本）
func NewPlantCardRenderSystem(em *ecs.EntityManager, cardScale, plantIconScale, plantIconOffsetY, sunTextOffsetY float64, sunFont *text.GoTextFace) *PlantCardRenderSystem {
	return &PlantCardRenderSystem{
		entityManager:    em,
		cardScale:        cardScale,
		plantIconScale:   plantIconScale,
		plantIconOffsetY: plantIconOffsetY,
		sunTextOffsetY:   sunTextOffsetY,
		sunFont:          sunFont, // 使用传入的字体
	}
}

// Draw 渲染所有植物卡片到屏幕
// Story 6.3: 多层渲染流程
// 层1: 卡片背景框
// 层2: 植物图标（Reanim离屏渲染的纹理）
// 层3: 阳光数字
// 层4: 冷却遮罩/禁用效果
func (s *PlantCardRenderSystem) Draw(screen *ebiten.Image) {
	// 查询所有拥有 PlantCardComponent, PositionComponent 的实体
	entities := ecs.GetEntitiesWith2[
		*components.PlantCardComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, entityID := range entities {
		// 获取组件
		card, ok := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 绘制多层卡片
		s.drawCardLayers(screen, card, pos.X, pos.Y)
	}
}

// drawCardLayers 绘制单个卡片的所有层
func (s *PlantCardRenderSystem) drawCardLayers(screen *ebiten.Image, card *components.PlantCardComponent, x, y float64) {
	// 层1: 绘制卡片背景框
	if card.BackgroundImage != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(s.cardScale, s.cardScale)
		op.GeoM.Translate(x, y)
		screen.DrawImage(card.BackgroundImage, op)
	}

	// 获取缩放后的卡片尺寸（用于居中和遮罩）
	var bgWidth, bgHeight float64
	if card.BackgroundImage != nil {
		bounds := card.BackgroundImage.Bounds()
		bgWidth = float64(bounds.Dx()) * s.cardScale
		bgHeight = float64(bounds.Dy()) * s.cardScale
	}

	// 层2: 绘制植物图标（居中对齐，可配置缩放）
	if card.PlantIconTexture != nil {
		iconBounds := card.PlantIconTexture.Bounds()
		iconWidth := float64(iconBounds.Dx()) * s.plantIconScale

		// 计算居中偏移
		offsetX := (bgWidth - iconWidth) / 2.0
		offsetY := s.plantIconOffsetY

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(s.plantIconScale, s.plantIconScale)
		op.GeoM.Translate(x+offsetX, y+offsetY)
		screen.DrawImage(card.PlantIconTexture, op)
	}

	// 层3: 绘制阳光数字（底部居中）
	s.drawSunCost(screen, x, y, bgWidth, bgHeight, card.SunCost)

	// 层4: 绘制效果遮罩（冷却/禁用）
	if !card.IsAvailable && bgHeight > 0 {
		if card.CurrentCooldown > 0 {
			// 冷却中：从上往下的渐进遮罩
			progress := card.CurrentCooldown / card.CooldownTime
			coverHeight := bgHeight * progress
			ebitenutil.DrawRect(screen, x, y, bgWidth, coverHeight,
				color.RGBA{0, 0, 0, 160}) // 半透明黑色
		} else {
			// 阳光不足：全屏淡遮罩
			ebitenutil.DrawRect(screen, x, y, bgWidth, bgHeight,
				color.RGBA{0, 0, 0, 100}) // 更淡的黑色
		}
	}
}

// drawSunCost 绘制阳光消耗数字（底部居中，黑色文本）
func (s *PlantCardRenderSystem) drawSunCost(screen *ebiten.Image, cardX, cardY, cardWidth, cardHeight float64, sunCost int) {
	sunText := fmt.Sprintf("%d", sunCost)

	if s.sunFont != nil {
		// 使用自定义字体渲染
		textWidth, _ := text.Measure(sunText, s.sunFont, 0)

		// 底部居中位置（使用可配置的偏移）
		textX := cardX + (cardWidth-textWidth)/2.0
		textY := cardY + cardHeight - s.sunTextOffsetY

		op := &text.DrawOptions{}
		op.GeoM.Translate(textX, textY)
		op.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 255}) // 黑色文字

		text.Draw(screen, sunText, s.sunFont, op)
	} else {
		// 使用调试文本（临时方案）
		// 注意：ebitenutil.DebugPrintAt 默认是白色，需要用自定义方式绘制黑色文本
		textX := int(cardX + cardWidth/2.0 - 10.0) // 粗略居中
		textY := int(cardY + cardHeight - s.sunTextOffsetY)
		ebitenutil.DebugPrintAt(screen, sunText, textX, textY)
	}
}
