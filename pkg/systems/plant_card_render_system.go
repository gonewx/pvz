package systems

import (
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// PlantCardRenderSystem 负责渲染植物卡片
// Story 6.3: 多层渲染架构 - 背景 + 植物图标 + 阳光数字 + 效果遮罩
// Story 8.4: 使用 PlantCardRenderer 进行渲染，消除重复代码
type PlantCardRenderSystem struct {
	entityManager    *ecs.EntityManager
	renderer         *utils.PlantCardRenderer       // Story 8.4: 通用卡片渲染器
	cardScale        float64                        // 卡片背景缩放因子
	plantIconScale   float64                        // 植物图标缩放因子（可配置）
	plantIconOffsetY float64                        // 植物图标垂直偏移（距离顶部的像素，可配置）
	sunTextOffsetY   float64                        // 阳光数字垂直偏移（距离底部的像素，可配置）
	sunFont          *text.GoTextFaceSource         // 阳光数字字体源（Story 8.4: 改为 GoTextFaceSource）
	sunFontSize      float64                        // 阳光字体大小
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
	// Story 8.4: 提取字体源，适配 PlantCardRenderer
	var fontSource *text.GoTextFaceSource
	var fontSize float64 = 12.0 // 默认字体大小
	if sunFont != nil {
		fontSource = sunFont.Source
		fontSize = sunFont.Size
	}

	return &PlantCardRenderSystem{
		entityManager:    em,
		renderer:         utils.NewPlantCardRenderer(), // Story 8.4: 初始化渲染器
		cardScale:        cardScale,
		plantIconScale:   plantIconScale,
		plantIconOffsetY: plantIconOffsetY,
		sunTextOffsetY:   sunTextOffsetY,
		sunFont:          fontSource,
		sunFontSize:      fontSize,
	}
}

// Draw 渲染所有植物卡片到屏幕
// Story 8.4: 使用 PlantCardRenderer 进行渲染
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

		// Story 8.4: 使用 PlantCardRenderer 渲染卡片
		s.renderer.Render(utils.PlantCardRenderOptions{
			Screen:           screen,
			X:                pos.X,
			Y:                pos.Y,
			BackgroundImage:  card.BackgroundImage,
			PlantIconImage:   card.PlantIconTexture,
			SunCost:          card.SunCost,
			SunFont:          s.sunFont,
			SunFontSize:      s.sunFontSize,
			SunTextOffsetY:   s.sunTextOffsetY,
			CardScale:        s.cardScale,
			PlantIconScale:   s.plantIconScale,
			PlantIconOffsetY: s.plantIconOffsetY,
			CooldownProgress: s.getCooldownProgress(card),
			IsDisabled:       s.isCardDisabled(card),
			Alpha:            1.0,
		})
	}
}

// getCooldownProgress 计算冷却进度（0.0-1.0）
func (s *PlantCardRenderSystem) getCooldownProgress(card *components.PlantCardComponent) float64 {
	if !card.IsAvailable && card.CurrentCooldown > 0 && card.CooldownTime > 0 {
		return card.CurrentCooldown / card.CooldownTime
	}
	return 0.0
}

// isCardDisabled 判断卡片是否处于禁用状态（阳光不足）
func (s *PlantCardRenderSystem) isCardDisabled(card *components.PlantCardComponent) bool {
	return !card.IsAvailable && card.CurrentCooldown == 0
}
