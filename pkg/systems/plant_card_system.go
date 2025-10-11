package systems

import (
	"reflect"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
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
func (s *PlantCardSystem) Update(deltaTime float64) {
	// 查询所有拥有 PlantCardComponent, UIComponent 的实体
	entities := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.PlantCardComponent{}),
		reflect.TypeOf(&components.UIComponent{}),
	)

	// 获取当前阳光数量
	currentSun := s.gameState.GetSun()

	// 更新每个卡片实体
	for _, entityID := range entities {
		// 获取组件
		cardComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.PlantCardComponent{}))
		if !ok {
			continue
		}
		card := cardComp.(*components.PlantCardComponent)

		uiComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.UIComponent{}))
		if !ok {
			continue
		}
		ui := uiComp.(*components.UIComponent)

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
