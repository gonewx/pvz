package systems

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestCooldownDecrement 测试冷却时间递减逻辑
func TestCooldownDecrement(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.Sun = 100 // 设置充足的阳光

	// 创建一个测试实体
	entity := em.CreateEntity()
	em.AddComponent(entity, &components.PlantCardComponent{
		PlantType:       components.PlantSunflower,
		SunCost:         50,
		CooldownTime:    10.0,
		CurrentCooldown: 5.0, // 初始冷却5秒
		IsAvailable:     false,
	})
	em.AddComponent(entity, &components.SpriteComponent{})
	em.AddComponent(entity, &components.UIComponent{State: components.UIDisabled})

	// 创建系统（不需要实际加载图像，我们只测试逻辑）
	system := &PlantCardSystem{
		entityManager: em,
		gameState:     gs,
	}

	// 更新1秒
	system.Update(1.0)

	// 验证冷却时间减少了1秒
	cardComp, _ := em.GetComponent(entity, reflect.TypeOf(&components.PlantCardComponent{}))
	card := cardComp.(*components.PlantCardComponent)

	if card.CurrentCooldown != 4.0 {
		t.Errorf("Expected CurrentCooldown = 4.0, got %f", card.CurrentCooldown)
	}
}

// TestCooldownReachesZero 测试冷却时间到达0并停止
func TestCooldownReachesZero(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.Sun = 100

	entity := em.CreateEntity()
	em.AddComponent(entity, &components.PlantCardComponent{
		PlantType:       components.PlantSunflower,
		SunCost:         50,
		CooldownTime:    10.0,
		CurrentCooldown: 0.5, // 剩余0.5秒
		IsAvailable:     false,
	})
	em.AddComponent(entity, &components.SpriteComponent{})
	em.AddComponent(entity, &components.UIComponent{State: components.UIDisabled})

	system := &PlantCardSystem{
		entityManager: em,
		gameState:     gs,
	}

	// 更新1秒（超过剩余冷却时间）
	system.Update(1.0)

	cardComp, _ := em.GetComponent(entity, reflect.TypeOf(&components.PlantCardComponent{}))
	card := cardComp.(*components.PlantCardComponent)

	// 验证冷却时间为0（不会变成负数）
	if card.CurrentCooldown != 0.0 {
		t.Errorf("Expected CurrentCooldown = 0.0, got %f", card.CurrentCooldown)
	}

	// 验证卡片变为可用
	if !card.IsAvailable {
		t.Error("Expected card to be available after cooldown ends")
	}
}

// TestCardUnavailableWhenNotEnoughSun 测试阳光不足时卡片不可用
func TestCardUnavailableWhenNotEnoughSun(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.Sun = 40 // 阳光不足（向日葵需要50）

	entity := em.CreateEntity()
	em.AddComponent(entity, &components.PlantCardComponent{
		PlantType:       components.PlantSunflower,
		SunCost:         50,
		CooldownTime:    10.0,
		CurrentCooldown: 0.0, // 无冷却
		IsAvailable:     true,
	})
	em.AddComponent(entity, &components.SpriteComponent{})
	em.AddComponent(entity, &components.UIComponent{State: components.UINormal})

	system := &PlantCardSystem{
		entityManager: em,
		gameState:     gs,
	}

	system.Update(0.1)

	cardComp, _ := em.GetComponent(entity, reflect.TypeOf(&components.PlantCardComponent{}))
	card := cardComp.(*components.PlantCardComponent)

	// 验证卡片不可用
	if card.IsAvailable {
		t.Error("Expected card to be unavailable when sun is insufficient")
	}

	// 验证 UI 状态为 Disabled
	uiComp, _ := em.GetComponent(entity, reflect.TypeOf(&components.UIComponent{}))
	ui := uiComp.(*components.UIComponent)
	if ui.State != components.UIDisabled {
		t.Errorf("Expected UI state to be Disabled, got %v", ui.State)
	}
}

// TestCardUnavailableDuringCooldown 测试冷却中卡片不可用
func TestCardUnavailableDuringCooldown(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.Sun = 100 // 阳光充足

	entity := em.CreateEntity()
	em.AddComponent(entity, &components.PlantCardComponent{
		PlantType:       components.PlantSunflower,
		SunCost:         50,
		CooldownTime:    10.0,
		CurrentCooldown: 3.0, // 冷却中
		IsAvailable:     false,
	})
	em.AddComponent(entity, &components.SpriteComponent{})
	em.AddComponent(entity, &components.UIComponent{State: components.UIDisabled})

	system := &PlantCardSystem{
		entityManager: em,
		gameState:     gs,
	}

	system.Update(0.1)

	cardComp, _ := em.GetComponent(entity, reflect.TypeOf(&components.PlantCardComponent{}))
	card := cardComp.(*components.PlantCardComponent)

	// 验证卡片不可用（即使阳光充足）
	if card.IsAvailable {
		t.Error("Expected card to be unavailable during cooldown")
	}
}

// TestCardAvailableWhenConditionsMet 测试条件满足时卡片可用
func TestCardAvailableWhenConditionsMet(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.Sun = 100 // 阳光充足

	entity := em.CreateEntity()
	em.AddComponent(entity, &components.PlantCardComponent{
		PlantType:       components.PlantSunflower,
		SunCost:         50,
		CooldownTime:    10.0,
		CurrentCooldown: 0.0, // 无冷却
		IsAvailable:     false,
	})
	em.AddComponent(entity, &components.SpriteComponent{})
	em.AddComponent(entity, &components.UIComponent{State: components.UIDisabled})

	system := &PlantCardSystem{
		entityManager: em,
		gameState:     gs,
	}

	system.Update(0.1)

	cardComp, _ := em.GetComponent(entity, reflect.TypeOf(&components.PlantCardComponent{}))
	card := cardComp.(*components.PlantCardComponent)

	// 验证卡片可用
	if !card.IsAvailable {
		t.Error("Expected card to be available when sun is sufficient and cooldown is 0")
	}

	// 验证 UI 状态恢复为 Normal
	uiComp, _ := em.GetComponent(entity, reflect.TypeOf(&components.UIComponent{}))
	ui := uiComp.(*components.UIComponent)
	if ui.State != components.UINormal {
		t.Errorf("Expected UI state to be Normal, got %v", ui.State)
	}
}

// TestUIStatePreservesHovered 测试 UI 状态保持 Hovered（不被覆盖）
func TestUIStatePreservesHovered(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.Sun = 100 // 阳光充足

	entity := em.CreateEntity()
	em.AddComponent(entity, &components.PlantCardComponent{
		PlantType:       components.PlantSunflower,
		SunCost:         50,
		CooldownTime:    10.0,
		CurrentCooldown: 0.0, // 无冷却
		IsAvailable:     true,
	})
	em.AddComponent(entity, &components.SpriteComponent{})
	em.AddComponent(entity, &components.UIComponent{State: components.UIHovered}) // 当前是 Hovered 状态

	system := &PlantCardSystem{
		entityManager: em,
		gameState:     gs,
	}

	system.Update(0.1)

	// 验证 UI 状态保持为 Hovered（不被覆盖为 Normal）
	uiComp, _ := em.GetComponent(entity, reflect.TypeOf(&components.UIComponent{}))
	ui := uiComp.(*components.UIComponent)
	if ui.State != components.UIHovered {
		t.Errorf("Expected UI state to remain Hovered, got %v", ui.State)
	}
}

// TestMultipleCards 测试多个卡片同时更新
func TestMultipleCards(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.Sun = 60 // 足够向日葵，不够豌豆射手

	// 创建向日葵卡片
	sunflower := em.CreateEntity()
	em.AddComponent(sunflower, &components.PlantCardComponent{
		PlantType:       components.PlantSunflower,
		SunCost:         50,
		CooldownTime:    10.0,
		CurrentCooldown: 0.0,
		IsAvailable:     false,
	})
	em.AddComponent(sunflower, &components.SpriteComponent{})
	em.AddComponent(sunflower, &components.UIComponent{State: components.UIDisabled})

	// 创建豌豆射手卡片
	peashooter := em.CreateEntity()
	em.AddComponent(peashooter, &components.PlantCardComponent{
		PlantType:       components.PlantPeashooter,
		SunCost:         100,
		CooldownTime:    10.0,
		CurrentCooldown: 0.0,
		IsAvailable:     false,
	})
	em.AddComponent(peashooter, &components.SpriteComponent{})
	em.AddComponent(peashooter, &components.UIComponent{State: components.UIDisabled})

	system := &PlantCardSystem{
		entityManager: em,
		gameState:     gs,
	}

	system.Update(0.1)

	// 验证向日葵卡片可用
	sunflowerCard, _ := em.GetComponent(sunflower, reflect.TypeOf(&components.PlantCardComponent{}))
	if !sunflowerCard.(*components.PlantCardComponent).IsAvailable {
		t.Error("Expected sunflower card to be available")
	}

	// 验证豌豆射手卡片不可用
	peashooterCard, _ := em.GetComponent(peashooter, reflect.TypeOf(&components.PlantCardComponent{}))
	if peashooterCard.(*components.PlantCardComponent).IsAvailable {
		t.Error("Expected peashooter card to be unavailable")
	}
}
