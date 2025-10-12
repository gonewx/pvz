package systems

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestSunflowerTimerUpdate 测试向日葵计时器更新逻辑
func TestSunflowerTimerUpdate(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建向日葵实体
	sunflowerID := em.CreateEntity()
	em.AddComponent(sunflowerID, &components.BehaviorComponent{
		Type: components.BehaviorSunflower,
	})
	em.AddComponent(sunflowerID, &components.TimerComponent{
		Name:        "sun_production",
		TargetTime:  7.0,
		CurrentTime: 0,
		IsReady:     false,
	})
	em.AddComponent(sunflowerID, &components.PlantComponent{
		PlantType: components.PlantSunflower,
		GridRow:   2,
		GridCol:   3,
	})
	em.AddComponent(sunflowerID, &components.PositionComponent{
		X: 400,
		Y: 300,
	})

	// 模拟1秒更新
	system.Update(1.0)

	// 验证计时器累加
	timerComp, _ := em.GetComponent(sunflowerID, reflect.TypeOf(&components.TimerComponent{}))
	timer := timerComp.(*components.TimerComponent)

	if timer.CurrentTime != 1.0 {
		t.Errorf("Expected CurrentTime=1.0, got %f", timer.CurrentTime)
	}

	// 再次更新2秒
	system.Update(2.0)

	timerComp, _ = em.GetComponent(sunflowerID, reflect.TypeOf(&components.TimerComponent{}))
	timer = timerComp.(*components.TimerComponent)

	if timer.CurrentTime != 3.0 {
		t.Errorf("Expected CurrentTime=3.0, got %f", timer.CurrentTime)
	}
}

// TestSunflowerFirstProduction 测试向日葵首次生产（7秒）
func TestSunflowerFirstProduction(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建向日葵实体
	sunflowerID := em.CreateEntity()
	em.AddComponent(sunflowerID, &components.BehaviorComponent{
		Type: components.BehaviorSunflower,
	})
	em.AddComponent(sunflowerID, &components.TimerComponent{
		Name:        "sun_production",
		TargetTime:  7.0,
		CurrentTime: 0,
		IsReady:     false,
	})
	em.AddComponent(sunflowerID, &components.PlantComponent{
		PlantType: components.PlantSunflower,
		GridRow:   2,
		GridCol:   3,
	})
	em.AddComponent(sunflowerID, &components.PositionComponent{
		X: 400,
		Y: 300,
	})

	// 模拟7秒更新（触发首次生产）
	system.Update(7.0)

	// 验证阳光实体被创建（查询拥有 SunComponent 的实体）
	sunEntities := em.GetEntitiesWith(reflect.TypeOf(&components.SunComponent{}))
	if len(sunEntities) != 1 {
		t.Errorf("Expected 1 sun entity, got %d", len(sunEntities))
	}

	// 验证计时器重置
	timerComp, _ := em.GetComponent(sunflowerID, reflect.TypeOf(&components.TimerComponent{}))
	timer := timerComp.(*components.TimerComponent)

	if timer.CurrentTime != 0 {
		t.Errorf("Expected CurrentTime=0 after production, got %f", timer.CurrentTime)
	}

	// 验证计时器目标时间变为24秒（后续生产周期）
	if timer.TargetTime != 24.0 {
		t.Errorf("Expected TargetTime=24.0 after first production, got %f", timer.TargetTime)
	}
}

// TestSunflowerSubsequentProduction 测试向日葵后续生产（24秒）
func TestSunflowerSubsequentProduction(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建向日葵实体（模拟已经生产过一次）
	sunflowerID := em.CreateEntity()
	em.AddComponent(sunflowerID, &components.BehaviorComponent{
		Type: components.BehaviorSunflower,
	})
	em.AddComponent(sunflowerID, &components.TimerComponent{
		Name:        "sun_production",
		TargetTime:  24.0, // 后续生产周期为24秒
		CurrentTime: 0,
		IsReady:     false,
	})
	em.AddComponent(sunflowerID, &components.PlantComponent{
		PlantType: components.PlantSunflower,
		GridRow:   2,
		GridCol:   3,
	})
	em.AddComponent(sunflowerID, &components.PositionComponent{
		X: 400,
		Y: 300,
	})

	// 模拟24秒更新（触发后续生产）
	system.Update(24.0)

	// 验证阳光实体被创建
	sunEntities := em.GetEntitiesWith(reflect.TypeOf(&components.SunComponent{}))
	if len(sunEntities) != 1 {
		t.Errorf("Expected 1 sun entity, got %d", len(sunEntities))
	}

	// 验证计时器重置
	timerComp, _ := em.GetComponent(sunflowerID, reflect.TypeOf(&components.TimerComponent{}))
	timer := timerComp.(*components.TimerComponent)

	if timer.CurrentTime != 0 {
		t.Errorf("Expected CurrentTime=0 after production, got %f", timer.CurrentTime)
	}

	// 验证计时器目标时间保持24秒
	if timer.TargetTime != 24.0 {
		t.Errorf("Expected TargetTime=24.0 for subsequent production, got %f", timer.TargetTime)
	}
}

// TestSunProductionPosition 测试阳光生成位置
func TestSunProductionPosition(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建向日葵实体
	sunflowerID := em.CreateEntity()
	em.AddComponent(sunflowerID, &components.BehaviorComponent{
		Type: components.BehaviorSunflower,
	})
	em.AddComponent(sunflowerID, &components.TimerComponent{
		Name:        "sun_production",
		TargetTime:  7.0,
		CurrentTime: 0,
		IsReady:     false,
	})
	em.AddComponent(sunflowerID, &components.PlantComponent{
		PlantType: components.PlantSunflower,
		GridRow:   2,
		GridCol:   3,
	})
	sunflowerX := 400.0
	sunflowerY := 300.0
	em.AddComponent(sunflowerID, &components.PositionComponent{
		X: sunflowerX,
		Y: sunflowerY,
	})

	// 触发生产
	system.Update(7.0)

	// 查找生成的阳光实体
	sunEntities := em.GetEntitiesWith(reflect.TypeOf(&components.SunComponent{}))
	if len(sunEntities) != 1 {
		t.Fatalf("Expected 1 sun entity, got %d", len(sunEntities))
	}

	// 验证阳光的起始位置
	sunID := sunEntities[0]
	positionComp, ok := em.GetComponent(sunID, reflect.TypeOf(&components.PositionComponent{}))
	if !ok {
		t.Fatal("Sun entity should have PositionComponent")
	}
	position := positionComp.(*components.PositionComponent)

	// 阳光应该在向日葵附近（考虑随机偏移）
	// X 坐标：sunflowerX - 40（居中）± 20（随机）= [340, 380]
	// Y 坐标：sunflowerY - 80（上方）± 10（随机）= [210, 230]
	expectedMinX := sunflowerX - 40.0 - 20.0
	expectedMaxX := sunflowerX - 40.0 + 20.0
	expectedMinY := sunflowerY - 80.0 - 10.0
	expectedMaxY := sunflowerY - 80.0 + 10.0

	if position.X < expectedMinX || position.X > expectedMaxX {
		t.Errorf("Sun X position out of range: expected [%.0f, %.0f], got %.0f",
			expectedMinX, expectedMaxX, position.X)
	}

	if position.Y < expectedMinY || position.Y > expectedMaxY {
		t.Errorf("Sun Y position out of range: expected [%.0f, %.0f], got %.0f",
			expectedMinY, expectedMaxY, position.Y)
	}

	// 检查阳光状态应该是 Landed（可以立即收集）
	sunComp, ok := em.GetComponent(sunID, reflect.TypeOf(&components.SunComponent{}))
	if !ok {
		t.Fatal("Sun entity should have SunComponent")
	}
	sun := sunComp.(*components.SunComponent)
	if sun.State != components.SunLanded {
		t.Errorf("Sun should be in Landed state, got %v", sun.State)
	}

	// 检查阳光速度应该是静止的
	velComp, ok := em.GetComponent(sunID, reflect.TypeOf(&components.VelocityComponent{}))
	if !ok {
		t.Fatal("Sun entity should have VelocityComponent")
	}
	vel := velComp.(*components.VelocityComponent)
	if vel.VX != 0 || vel.VY != 0 {
		t.Errorf("Sun should be stationary, got velocity (%.1f, %.1f)", vel.VX, vel.VY)
	}
}

// TestMultipleSunflowersIndependentTimers 测试多个向日葵独立工作
func TestMultipleSunflowersIndependentTimers(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建第一个向日葵
	sunflower1 := em.CreateEntity()
	em.AddComponent(sunflower1, &components.BehaviorComponent{Type: components.BehaviorSunflower})
	em.AddComponent(sunflower1, &components.TimerComponent{
		Name:        "sun_production",
		TargetTime:  7.0,
		CurrentTime: 0,
	})
	em.AddComponent(sunflower1, &components.PlantComponent{PlantType: components.PlantSunflower})
	em.AddComponent(sunflower1, &components.PositionComponent{X: 400, Y: 300})

	// 创建第二个向日葵
	sunflower2 := em.CreateEntity()
	em.AddComponent(sunflower2, &components.BehaviorComponent{Type: components.BehaviorSunflower})
	em.AddComponent(sunflower2, &components.TimerComponent{
		Name:        "sun_production",
		TargetTime:  7.0,
		CurrentTime: 0,
	})
	em.AddComponent(sunflower2, &components.PlantComponent{PlantType: components.PlantSunflower})
	em.AddComponent(sunflower2, &components.PositionComponent{X: 500, Y: 300})

	// 更新3秒
	system.Update(3.0)

	// 验证两个向日葵的计时器都更新了
	timer1Comp, _ := em.GetComponent(sunflower1, reflect.TypeOf(&components.TimerComponent{}))
	timer1 := timer1Comp.(*components.TimerComponent)
	if timer1.CurrentTime != 3.0 {
		t.Errorf("Sunflower1 timer should be 3.0, got %f", timer1.CurrentTime)
	}

	timer2Comp, _ := em.GetComponent(sunflower2, reflect.TypeOf(&components.TimerComponent{}))
	timer2 := timer2Comp.(*components.TimerComponent)
	if timer2.CurrentTime != 3.0 {
		t.Errorf("Sunflower2 timer should be 3.0, got %f", timer2.CurrentTime)
	}

	// 再更新5秒（总计8秒，两个都应该生产）
	system.Update(5.0)

	// 验证生成了2个阳光
	sunEntities := em.GetEntitiesWith(reflect.TypeOf(&components.SunComponent{}))
	if len(sunEntities) != 2 {
		t.Errorf("Expected 2 sun entities, got %d", len(sunEntities))
	}
}

// TestZombieBasicMovement 测试僵尸正常移动
func TestZombieBasicMovement(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建僵尸实体
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.BehaviorComponent{
		Type: components.BehaviorZombieBasic,
	})
	em.AddComponent(zombieID, &components.PositionComponent{
		X: 1000.0, // 初始位置在屏幕右侧
		Y: 200.0,
	})
	em.AddComponent(zombieID, &components.VelocityComponent{
		VX: -30.0, // 向左移动
		VY: 0.0,
	})

	// 模拟1秒更新
	system.Update(1.0)

	// 验证僵尸位置更新：X 应该减少 30 像素
	posComp, _ := em.GetComponent(zombieID, reflect.TypeOf(&components.PositionComponent{}))
	pos := posComp.(*components.PositionComponent)

	expectedX := 1000.0 - 30.0*1.0 // 970.0
	if pos.X != expectedX {
		t.Errorf("Expected X=%.1f, got %.1f", expectedX, pos.X)
	}

	// 再次更新2秒
	system.Update(2.0)

	posComp, _ = em.GetComponent(zombieID, reflect.TypeOf(&components.PositionComponent{}))
	pos = posComp.(*components.PositionComponent)

	expectedX = 970.0 - 30.0*2.0 // 910.0
	if pos.X != expectedX {
		t.Errorf("Expected X=%.1f after second update, got %.1f", expectedX, pos.X)
	}
}

// TestZombieBoundaryDeletion 测试僵尸移出屏幕后被删除
func TestZombieBoundaryDeletion(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建僵尸实体（位置接近屏幕左侧边界）
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.BehaviorComponent{
		Type: components.BehaviorZombieBasic,
	})
	em.AddComponent(zombieID, &components.PositionComponent{
		X: -50.0, // 接近边界，但还没超出删除阈值（-100）
		Y: 200.0,
	})
	em.AddComponent(zombieID, &components.VelocityComponent{
		VX: -30.0,
		VY: 0.0,
	})

	// 模拟2秒更新（应该移动到 -50 - 60 = -110，超出-100阈值）
	system.Update(2.0)

	// 清理标记删除的实体
	em.RemoveMarkedEntities()

	// 验证僵尸实体被删除
	zombieEntities := em.GetEntitiesWith(
		reflect.TypeOf(&components.BehaviorComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.VelocityComponent{}),
	)

	// 检查是否还有僵尸实体（应该为0，因为被删除了）
	hasZombie := false
	for _, id := range zombieEntities {
		behaviorComp, _ := em.GetComponent(id, reflect.TypeOf(&components.BehaviorComponent{}))
		behavior := behaviorComp.(*components.BehaviorComponent)
		if behavior.Type == components.BehaviorZombieBasic && id == zombieID {
			hasZombie = true
			break
		}
	}

	if hasZombie {
		t.Error("Zombie should be deleted after moving past left boundary")
	}
}

// TestMultipleZombiesIndependentMovement 测试多个僵尸独立移动
func TestMultipleZombiesIndependentMovement(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建第一个僵尸（速度 -30）
	zombie1 := em.CreateEntity()
	em.AddComponent(zombie1, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombie1, &components.PositionComponent{X: 1000.0, Y: 200.0})
	em.AddComponent(zombie1, &components.VelocityComponent{VX: -30.0, VY: 0.0})

	// 创建第二个僵尸（速度 -40，更快）
	zombie2 := em.CreateEntity()
	em.AddComponent(zombie2, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombie2, &components.PositionComponent{X: 1200.0, Y: 300.0})
	em.AddComponent(zombie2, &components.VelocityComponent{VX: -40.0, VY: 0.0})

	// 更新1秒
	system.Update(1.0)

	// 验证第一个僵尸位置
	pos1Comp, _ := em.GetComponent(zombie1, reflect.TypeOf(&components.PositionComponent{}))
	pos1 := pos1Comp.(*components.PositionComponent)
	if pos1.X != 970.0 {
		t.Errorf("Zombie1 should be at X=970.0, got %.1f", pos1.X)
	}

	// 验证第二个僵尸位置
	pos2Comp, _ := em.GetComponent(zombie2, reflect.TypeOf(&components.PositionComponent{}))
	pos2 := pos2Comp.(*components.PositionComponent)
	if pos2.X != 1160.0 {
		t.Errorf("Zombie2 should be at X=1160.0, got %.1f", pos2.X)
	}

	// 再更新2秒
	system.Update(2.0)

	// 验证两个僵尸都继续移动
	pos1Comp, _ = em.GetComponent(zombie1, reflect.TypeOf(&components.PositionComponent{}))
	pos1 = pos1Comp.(*components.PositionComponent)
	expectedX1 := 970.0 - 30.0*2.0 // 910.0
	if pos1.X != expectedX1 {
		t.Errorf("Zombie1 should be at X=%.1f, got %.1f", expectedX1, pos1.X)
	}

	pos2Comp, _ = em.GetComponent(zombie2, reflect.TypeOf(&components.PositionComponent{}))
	pos2 = pos2Comp.(*components.PositionComponent)
	expectedX2 := 1160.0 - 40.0*2.0 // 1080.0
	if pos2.X != expectedX2 {
		t.Errorf("Zombie2 should be at X=%.1f, got %.1f", expectedX2, pos2.X)
	}
}

// TestHitEffectExpiration 测试击中效果计时器超时后被删除
func TestHitEffectExpiration(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建击中效果实体
	hitEffectID := em.CreateEntity()
	em.AddComponent(hitEffectID, &components.BehaviorComponent{
		Type: components.BehaviorPeaBulletHit,
	})
	em.AddComponent(hitEffectID, &components.PositionComponent{
		X: 400,
		Y: 250,
	})
	em.AddComponent(hitEffectID, &components.TimerComponent{
		Name:        "hit_effect_duration",
		CurrentTime: 0.0,
		TargetTime:  0.2, // 0.2秒后应该被删除
		IsReady:     false,
	})

	// 第一次更新：0.1秒（未超时）
	system.Update(0.1)

	// 击中效果应该还存在
	timerComp, ok := em.GetComponent(hitEffectID, reflect.TypeOf(&components.TimerComponent{}))
	if !ok {
		t.Fatal("Expected hit effect to still exist after 0.1s")
	}
	timer := timerComp.(*components.TimerComponent)
	if timer.CurrentTime != 0.1 {
		t.Errorf("Expected timer CurrentTime=0.1, got %.2f", timer.CurrentTime)
	}

	// 第二次更新：再0.15秒（总共0.25秒，超过0.2秒阈值）
	system.Update(0.15)

	// 击中效果应该被标记删除
	em.RemoveMarkedEntities()

	// 验证击中效果已被删除
	_, exists := em.GetComponent(hitEffectID, reflect.TypeOf(&components.TimerComponent{}))
	if exists {
		t.Error("Expected hit effect to be destroyed after timeout (0.25s > 0.2s)")
	}
}
