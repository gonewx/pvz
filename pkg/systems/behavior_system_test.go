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

	// 阳光应该在向日葵上方（X相同，Y约小20像素）
	if position.X != sunflowerX {
		t.Errorf("Expected sun X=%f, got %f", sunflowerX, position.X)
	}

	// 注意：阳光的Y坐标是startY（屏幕顶部外），不是 sunflowerY-20
	// 检查阳光是否有 VelocityComponent（说明它会下落）
	_, hasVelocity := em.GetComponent(sunID, reflect.TypeOf(&components.VelocityComponent{}))
	if !hasVelocity {
		t.Error("Sun entity should have VelocityComponent to fall")
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
