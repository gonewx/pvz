package systems

import (
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// PlantCardRenderSystem 负责渲染植物卡片
// Story 6.3 + 8.4: 渲染逻辑封装在 entities.PlantCardFactory 中
// 本系统只负责遍历实体并调用统一的渲染函数
type PlantCardRenderSystem struct {
	entityManager *ecs.EntityManager
	sunFont       *text.GoTextFaceSource // 阳光数字字体源
	sunFontSize   float64                // 阳光字体大小
}

// NewPlantCardRenderSystem 创建一个新的 PlantCardRenderSystem 实例
// 参数:
//   - em: 实体管理器
//   - sunFont: 阳光数字字体（可选，如果为 nil 则不渲染阳光数字）
func NewPlantCardRenderSystem(em *ecs.EntityManager, sunFont *text.GoTextFace) *PlantCardRenderSystem {
	var fontSource *text.GoTextFaceSource
	var fontSize float64 = 20.0 // 默认字体大小
	if sunFont != nil {
		fontSource = sunFont.Source
		fontSize = sunFont.Size
	}

	return &PlantCardRenderSystem{
		entityManager: em,
		sunFont:       fontSource,
		sunFontSize:   fontSize,
	}
}

// Draw 渲染所有植物卡片到屏幕
// 自动过滤奖励卡片（有 RewardCardComponent 标记的卡片）
// 奖励卡片由各自的系统（如 RewardAnimationSystem）自行渲染
func (s *PlantCardRenderSystem) Draw(screen *ebiten.Image) {
	// 查询所有拥有 PlantCardComponent, PositionComponent 的实体
	cardEntities := ecs.GetEntitiesWith2[
		*components.PlantCardComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, entityID := range cardEntities {
		// 跳过奖励卡片（由 RewardAnimationSystem 自行渲染）
		if _, isRewardCard := ecs.GetComponent[*components.RewardCardComponent](s.entityManager, entityID); isRewardCard {
			continue
		}

		// 获取组件
		card, ok := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 调用工厂的统一渲染函数（所有渲染逻辑封装在工厂中）
		entities.RenderPlantCard(screen, card, pos.X, pos.Y, s.sunFont, s.sunFontSize)
	}
}
