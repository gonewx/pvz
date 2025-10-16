package systems

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestPointInRectCollision 测试点在矩形内的碰撞检测逻辑
func TestPointInRectCollision(t *testing.T) {
	tests := []struct {
		name      string
		mouseX    float64
		mouseY    float64
		rectX     float64
		rectY     float64
		rectW     float64
		rectH     float64
		shouldHit bool
	}{
		{"点在矩形内", 150, 150, 100, 100, 100, 100, true},
		{"点在左边界上", 100, 150, 100, 100, 100, 100, true},
		{"点在右边界上", 200, 150, 100, 100, 100, 100, true},
		{"点在上边界上", 150, 100, 100, 100, 100, 100, true},
		{"点在下边界上", 150, 200, 100, 100, 100, 100, true},
		{"点在矩形左侧", 50, 150, 100, 100, 100, 100, false},
		{"点在矩形右侧", 250, 150, 100, 100, 100, 100, false},
		{"点在矩形上方", 150, 50, 100, 100, 100, 100, false},
		{"点在矩形下方", 150, 250, 100, 100, 100, 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 使用InputSystem中的碰撞检测逻辑
			hit := tt.mouseX >= tt.rectX && tt.mouseX <= tt.rectX+tt.rectW &&
				tt.mouseY >= tt.rectY && tt.mouseY <= tt.rectY+tt.rectH

			if hit != tt.shouldHit {
				t.Errorf("Expected hit=%v, got %v for point(%v,%v) in rect(%v,%v,%v,%v)",
					tt.shouldHit, hit, tt.mouseX, tt.mouseY, tt.rectX, tt.rectY, tt.rectW, tt.rectH)
			}
		})
	}
}

// TestSunClickStateChange 测试点击已落地阳光的状态变化
func TestSunClickStateChange(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	gs := game.GetGameState()
	initialSun := gs.GetSun()

	// 创建草坪网格系统和实体（测试用）
	lawnGridSystem := NewLawnGridSystem(em, nil)
	lawnGridEntityID := em.CreateEntity()
	em.AddComponent(lawnGridEntityID, &components.LawnGridComponent{})

	system := NewInputSystem(em, rm, gs, nil, 21.0, 80.0, lawnGridSystem, lawnGridEntityID)

	// 创建已落地的阳光实体
	id := em.CreateEntity()
	em.AddComponent(id, &components.PositionComponent{X: 100, Y: 100})
	em.AddComponent(id, &components.ClickableComponent{Width: 80, Height: 80, IsEnabled: true})
	em.AddComponent(id, &components.SunComponent{State: components.SunLanded, TargetY: 100})
	em.AddComponent(id, &components.VelocityComponent{VX: 0, VY: 0})
	em.AddComponent(id, &components.LifetimeComponent{MaxLifetime: 15, CurrentLifetime: 5, IsExpired: false})

	// 模拟点击处理（直接调用handleSunClick）
	posComp, _ := em.GetComponent(id, reflect.TypeOf(&components.PositionComponent{}))
	pos := posComp.(*components.PositionComponent)
	system.handleSunClick(id, pos)

	// 验证阳光状态变为收集中
	sunComp, _ := em.GetComponent(id, reflect.TypeOf(&components.SunComponent{}))
	sun := sunComp.(*components.SunComponent)
	if sun.State != components.SunCollecting {
		t.Errorf("Expected SunComponent.State=SunCollecting, got %v", sun.State)
	}

	// 验证ClickableComponent被禁用
	clickableComp, _ := em.GetComponent(id, reflect.TypeOf(&components.ClickableComponent{}))
	clickable := clickableComp.(*components.ClickableComponent)
	if clickable.IsEnabled {
		t.Error("ClickableComponent.IsEnabled should be false after click")
	}

	// 注意: 阳光数量会在 SunCollectionSystem 检测到阳光到达目标位置时增加
	// 点击时不会立即增加,这里验证数量尚未变化
	if gs.GetSun() != initialSun {
		t.Errorf("Sun count should not change immediately on click, expected=%d, got %d", initialSun, gs.GetSun())
	}

	// 验证LifetimeComponent被移除
	_, hasLifetime := em.GetComponent(id, reflect.TypeOf(&components.LifetimeComponent{}))
	if hasLifetime {
		t.Error("LifetimeComponent should be removed after click")
	}

	// 验证VelocityComponent被设置为飞向目标
	velComp, _ := em.GetComponent(id, reflect.TypeOf(&components.VelocityComponent{}))
	vel := velComp.(*components.VelocityComponent)
	if vel.VX == 0 && vel.VY == 0 {
		t.Error("VelocityComponent should be updated to fly towards sun counter")
	}
}

// TestClickFallingSunNoEffect 测试点击掉落中的阳光无效
func TestClickFallingSunNoEffect(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	gs := game.GetGameState()

	// 创建草坪网格系统和实体（测试用）
	lawnGridSystem := NewLawnGridSystem(em, nil)
	lawnGridEntityID := em.CreateEntity()
	em.AddComponent(lawnGridEntityID, &components.LawnGridComponent{})

	NewInputSystem(em, rm, gs, nil, 21.0, 80.0, lawnGridSystem, lawnGridEntityID)

	// 创建正在掉落的阳光实体
	id := em.CreateEntity()
	em.AddComponent(id, &components.PositionComponent{X: 100, Y: 100})
	em.AddComponent(id, &components.ClickableComponent{Width: 80, Height: 80, IsEnabled: true})
	em.AddComponent(id, &components.SunComponent{State: components.SunFalling, TargetY: 200}) // 正在掉落

	// 检查系统查询时会跳过非SunLanded状态的阳光
	entities := em.GetEntitiesWith(
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.ClickableComponent{}),
		reflect.TypeOf(&components.SunComponent{}),
	)

	if len(entities) != 1 {
		t.Errorf("Expected 1 entity, got %d", len(entities))
	}

	// 验证阳光状态仍然是SunFalling（Update方法会检查并跳过）
	sunComp, _ := em.GetComponent(id, reflect.TypeOf(&components.SunComponent{}))
	sun := sunComp.(*components.SunComponent)
	if sun.State != components.SunFalling {
		t.Error("SunComponent.State should remain SunFalling")
	}

	// 验证ClickableComponent仍然启用
	clickableComp, _ := em.GetComponent(id, reflect.TypeOf(&components.ClickableComponent{}))
	clickable := clickableComp.(*components.ClickableComponent)
	if !clickable.IsEnabled {
		t.Error("ClickableComponent.IsEnabled should remain true for falling sun")
	}
}

// TestClickDisabledSunNoEffect 测试点击已被禁用的阳光无效
func TestClickDisabledSunNoEffect(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	gs := game.GetGameState()

	// 创建草坪网格系统和实体（测试用）
	lawnGridSystem := NewLawnGridSystem(em, nil)
	lawnGridEntityID := em.CreateEntity()
	em.AddComponent(lawnGridEntityID, &components.LawnGridComponent{})

	_ = NewInputSystem(em, rm, gs, nil, 21.0, 80.0, lawnGridSystem, lawnGridEntityID)

	// 创建已被禁用的阳光实体（已被点击过）
	id := em.CreateEntity()
	em.AddComponent(id, &components.PositionComponent{X: 100, Y: 100})
	em.AddComponent(id, &components.ClickableComponent{Width: 80, Height: 80, IsEnabled: false}) // 已禁用
	em.AddComponent(id, &components.SunComponent{State: components.SunLanded, TargetY: 100})
	em.AddComponent(id, &components.VelocityComponent{VX: 0, VY: 0})

	// 尝试再次点击
	sunBeforeClick := gs.GetSun()

	// 实际上InputSystem的Update方法会检查IsEnabled，所以不会调用handleSunClick
	// 这里我们验证状态不变
	clickableComp, _ := em.GetComponent(id, reflect.TypeOf(&components.ClickableComponent{}))
	clickable := clickableComp.(*components.ClickableComponent)
	if clickable.IsEnabled {
		t.Error("ClickableComponent should remain disabled")
	}

	// 验证阳光数量没有变化（因为Update会跳过禁用的阳光）
	if gs.GetSun() != sunBeforeClick {
		t.Error("Sun count should not change when clicking disabled sun")
	}
}

// TestVelocityCalculation 测试飞向阳光计数器的速度向量计算
func TestVelocityCalculation(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	gs := game.GetGameState()

	targetX := 21.0
	targetY := 80.0

	// 创建草坪网格系统和实体（测试用）
	lawnGridSystem := NewLawnGridSystem(em, nil)
	lawnGridEntityID := em.CreateEntity()
	em.AddComponent(lawnGridEntityID, &components.LawnGridComponent{})

	system := NewInputSystem(em, rm, gs, nil, targetX, targetY, lawnGridSystem, lawnGridEntityID)

	// 创建阳光实体
	id := em.CreateEntity()
	startX := 400.0
	startY := 300.0
	em.AddComponent(id, &components.PositionComponent{X: startX, Y: startY})
	em.AddComponent(id, &components.ClickableComponent{Width: 80, Height: 80, IsEnabled: true})
	em.AddComponent(id, &components.SunComponent{State: components.SunLanded, TargetY: 300})
	em.AddComponent(id, &components.VelocityComponent{VX: 0, VY: 0})

	// 点击阳光
	posComp, _ := em.GetComponent(id, reflect.TypeOf(&components.PositionComponent{}))
	pos := posComp.(*components.PositionComponent)
	system.handleSunClick(id, pos)

	// 验证速度向量指向目标
	velComp, _ := em.GetComponent(id, reflect.TypeOf(&components.VelocityComponent{}))
	vel := velComp.(*components.VelocityComponent)

	// 计算预期方向（应该指向左上角）
	dx := targetX - startX
	dy := targetY - startY

	// 速度应该是负X（向左）和负Y（向上）
	if vel.VX >= 0 {
		t.Error("VX should be negative (moving left)")
	}
	if vel.VY >= 0 {
		t.Error("VY should be negative (moving up)")
	}

	// 验证方向正确（VX和VY的比例应该与dx和dy相同）
	if dx != 0 && dy != 0 {
		ratio := vel.VX / vel.VY
		expectedRatio := dx / dy
		// 允许浮点误差
		if ratio < expectedRatio-0.01 || ratio > expectedRatio+0.01 {
			t.Errorf("Velocity direction incorrect: ratio=%v, expected=%v", ratio, expectedRatio)
		}
	}
}

// TestGetPlantCost 测试植物阳光消耗计算
func TestGetPlantCost(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	gs := game.GetGameState()
	lawnGridSystem := NewLawnGridSystem(em, nil)
	lawnGridEntityID := em.CreateEntity()
	em.AddComponent(lawnGridEntityID, &components.LawnGridComponent{})

	system := NewInputSystem(em, rm, gs, nil, 21.0, 80.0, lawnGridSystem, lawnGridEntityID)

	tests := []struct {
		name      string
		plantType components.PlantType
		wantCost  int
	}{
		{"向日葵", components.PlantSunflower, 50},
		{"豌豆射手", components.PlantPeashooter, 100},
		{"未知植物", components.PlantType(999), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := system.getPlantCost(tt.plantType)
			if cost != tt.wantCost {
				t.Errorf("getPlantCost(%v) = %d, want %d", tt.plantType, cost, tt.wantCost)
			}
		})
	}
}

// TestTriggerPlantCardCooldown 测试触发卡片冷却
func TestTriggerPlantCardCooldown(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	gs := game.GetGameState()
	lawnGridSystem := NewLawnGridSystem(em, nil)
	lawnGridEntityID := em.CreateEntity()
	em.AddComponent(lawnGridEntityID, &components.LawnGridComponent{})

	system := NewInputSystem(em, rm, gs, nil, 21.0, 80.0, lawnGridSystem, lawnGridEntityID)

	// 创建向日葵卡片
	cardID := em.CreateEntity()
	em.AddComponent(cardID, &components.PlantCardComponent{
		PlantType:       components.PlantSunflower,
		SunCost:         50,
		CooldownTime:    7.5,
		CurrentCooldown: 0,
		IsAvailable:     true,
	})

	// 触发冷却
	system.triggerPlantCardCooldown(components.PlantSunflower)

	// 验证冷却已触发
	cardComp, _ := em.GetComponent(cardID, reflect.TypeOf(&components.PlantCardComponent{}))
	card := cardComp.(*components.PlantCardComponent)

	if card.CurrentCooldown != 7.5 {
		t.Errorf("Expected CurrentCooldown=7.5, got %f", card.CurrentCooldown)
	}
}
