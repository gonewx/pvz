package systems

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
)

func TestSunFalling(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewSunMovementSystem(em)

	// 创建测试阳光实体
	id := em.CreateEntity()
	em.AddComponent(id, &components.PositionComponent{X: 100, Y: 0})
	em.AddComponent(id, &components.VelocityComponent{VX: 0, VY: 60})
	em.AddComponent(id, &components.SunComponent{State: components.SunFalling, TargetY: 100})

	// 模拟1秒更新
	system.Update(1.0)

	// 验证位置更新
	posComp, _ := em.GetComponent(id, reflect.TypeOf(&components.PositionComponent{}))
	pos := posComp.(*components.PositionComponent)

	if pos.Y != 60 { // 初始0 + 60像素/秒 * 1秒
		t.Errorf("Expected Y=60, got Y=%f", pos.Y)
	}

	// 验证状态仍为 Falling
	sunComp, _ := em.GetComponent(id, reflect.TypeOf(&components.SunComponent{}))
	sun := sunComp.(*components.SunComponent)

	if sun.State != components.SunFalling {
		t.Error("Sun should still be falling")
	}
}

func TestSunLanding(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewSunMovementSystem(em)

	// 创建即将落地的阳光
	id := em.CreateEntity()
	em.AddComponent(id, &components.PositionComponent{X: 100, Y: 90})
	em.AddComponent(id, &components.VelocityComponent{VX: 0, VY: 60})
	em.AddComponent(id, &components.SunComponent{State: components.SunFalling, TargetY: 100})

	// 模拟更新
	system.Update(0.5) // 90 + 60*0.5 = 120 > 100,应该落地

	// 验证状态变为 Landed
	sunComp, _ := em.GetComponent(id, reflect.TypeOf(&components.SunComponent{}))
	sun := sunComp.(*components.SunComponent)

	if sun.State != components.SunLanded {
		t.Error("Sun should have landed")
	}

	// 验证位置设置为精确的目标位置
	posComp, _ := em.GetComponent(id, reflect.TypeOf(&components.PositionComponent{}))
	pos := posComp.(*components.PositionComponent)

	if pos.Y != 100 {
		t.Errorf("Expected Y=100 (target), got Y=%f", pos.Y)
	}

	// 验证速度归零
	velComp, _ := em.GetComponent(id, reflect.TypeOf(&components.VelocityComponent{}))
	vel := velComp.(*components.VelocityComponent)

	if vel.VY != 0 {
		t.Error("Velocity should be zero after landing")
	}
}

func TestSunLandedStaysStill(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewSunMovementSystem(em)

	// 创建已落地的阳光
	id := em.CreateEntity()
	em.AddComponent(id, &components.PositionComponent{X: 100, Y: 200})
	em.AddComponent(id, &components.VelocityComponent{VX: 0, VY: 0})
	em.AddComponent(id, &components.SunComponent{State: components.SunLanded, TargetY: 200})

	// 模拟更新
	system.Update(1.0)

	// 验证位置不变
	posComp, _ := em.GetComponent(id, reflect.TypeOf(&components.PositionComponent{}))
	pos := posComp.(*components.PositionComponent)

	if pos.Y != 200 {
		t.Errorf("Landed sun should stay still at Y=200, got Y=%f", pos.Y)
	}
}

func TestMultipleSuns(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewSunMovementSystem(em)

	// 创建多个阳光实体
	id1 := em.CreateEntity()
	em.AddComponent(id1, &components.PositionComponent{X: 100, Y: 0})
	em.AddComponent(id1, &components.VelocityComponent{VX: 0, VY: 60})
	em.AddComponent(id1, &components.SunComponent{State: components.SunFalling, TargetY: 100})

	id2 := em.CreateEntity()
	em.AddComponent(id2, &components.PositionComponent{X: 200, Y: 50})
	em.AddComponent(id2, &components.VelocityComponent{VX: 0, VY: 60})
	em.AddComponent(id2, &components.SunComponent{State: components.SunFalling, TargetY: 200})

	// 模拟更新
	system.Update(1.0)

	// 验证第一个阳光位置
	posComp1, _ := em.GetComponent(id1, reflect.TypeOf(&components.PositionComponent{}))
	pos1 := posComp1.(*components.PositionComponent)
	if pos1.Y != 60 {
		t.Errorf("Sun 1: Expected Y=60, got Y=%f", pos1.Y)
	}

	// 验证第二个阳光位置
	posComp2, _ := em.GetComponent(id2, reflect.TypeOf(&components.PositionComponent{}))
	pos2 := posComp2.(*components.PositionComponent)
	if pos2.Y != 110 {
		t.Errorf("Sun 2: Expected Y=110, got Y=%f", pos2.Y)
	}
}
