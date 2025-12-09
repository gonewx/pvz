package systems

import (
	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
)

// PlantCardSystem 负责更新植物卡片的状态
// 包括冷却时间递减、可用性判断、UI状态更新
type PlantCardSystem struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager
}

// NewPlantCardSystem 创建一个新的 PlantCardSystem 实例
func NewPlantCardSystem(em *ecs.EntityManager, gs *game.GameState, rm *game.ResourceManager) *PlantCardSystem {
	return &PlantCardSystem{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
	}
}

// Update 更新所有植物卡片的状态
// 包括冷却时间递减、可用性判断、UI状态和图像更新
// 跳过奖励卡片（有 RewardCardComponent 标记的卡片由 RewardAnimationSystem 管理）
func (s *PlantCardSystem) Update(deltaTime float64) {
	// 查询所有拥有 PlantCardComponent, UIComponent 的实体
	entities := ecs.GetEntitiesWith2[
		*components.PlantCardComponent,
		*components.UIComponent,
	](s.entityManager)

	// 获取当前阳光数量
	currentSun := s.gameState.GetSun()

	// 更新每个卡片实体
	for _, entityID := range entities {
		// 跳过奖励卡片（由 RewardAnimationSystem 管理）
		if _, isRewardCard := ecs.GetComponent[*components.RewardCardComponent](s.entityManager, entityID); isRewardCard {
			continue
		}

		// 获取组件
		card, ok := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		ui, ok := ecs.GetComponent[*components.UIComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 更新冷却时间
		if card.CurrentCooldown > 0 {
			card.CurrentCooldown -= deltaTime
			if card.CurrentCooldown < 0 {
				card.CurrentCooldown = 0
			}
		}

		// 判断可用性
		isAffordable := currentSun >= card.SunCost
		isCooledDown := card.CurrentCooldown <= 0
		card.IsAvailable = isAffordable && isCooledDown

		// 更新 UIComponent 状态
		// 如果不可用（阳光不足或冷却中），设置为 Disabled
		// 否则保持当前状态（可能是 Normal 或 Hovered）
		if !isAffordable || !isCooledDown {
			ui.State = components.UIDisabled
		} else {
			// 只有在当前是 Disabled 状态时才恢复为 Normal
			// 避免覆盖 Hovered 或其他交互状态
			if ui.State == components.UIDisabled {
				ui.State = components.UINormal
			}
		}
	}
}
