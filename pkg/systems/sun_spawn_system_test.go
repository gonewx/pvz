package systems

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

// 共享的 audio context,避免重复创建
var testAudioContext = audio.NewContext(48000)

func TestSunSpawnTimer(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewSunSpawnSystem(em, rm, 250, 900, 100, 550)

	// 初始状态:没有阳光实体
	entities := em.GetEntitiesWith(reflect.TypeOf(&components.SunComponent{}))
	if len(entities) != 0 {
		t.Error("Should have no sun entities initially")
	}

	// 更新5秒(未到达8秒间隔)
	system.Update(5.0)

	entities = em.GetEntitiesWith(reflect.TypeOf(&components.SunComponent{}))
	if len(entities) != 0 {
		t.Error("Should not spawn sun before interval")
	}

	// 再更新3秒(总计8秒,到达间隔)
	system.Update(3.0)

	entities = em.GetEntitiesWith(reflect.TypeOf(&components.SunComponent{}))
	if len(entities) != 1 {
		t.Errorf("Should spawn 1 sun after 8 seconds, got %d", len(entities))
	}
}

func TestSunSpawnMultiple(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewSunSpawnSystem(em, rm, 250, 900, 100, 550)

	// 模拟多个生成周期
	system.Update(8.0) // 第一次生成
	system.Update(8.0) // 第二次生成
	system.Update(8.0) // 第三次生成

	entities := em.GetEntitiesWith(reflect.TypeOf(&components.SunComponent{}))
	if len(entities) != 3 {
		t.Errorf("Should spawn 3 suns after 24 seconds, got %d", len(entities))
	}
}

func TestSunSpawnPositionRange(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)

	minX, maxX := 250.0, 900.0
	minY, maxY := 100.0, 550.0

	system := NewSunSpawnSystem(em, rm, minX, maxX, minY, maxY)

	// 生成一个阳光
	system.Update(8.0)

	entities := em.GetEntitiesWith(reflect.TypeOf(&components.SunComponent{}))
	if len(entities) != 1 {
		t.Fatal("Should spawn 1 sun")
	}

	// 验证位置在范围内
	id := entities[0]
	posComp, _ := em.GetComponent(id, reflect.TypeOf(&components.PositionComponent{}))
	pos := posComp.(*components.PositionComponent)

	if pos.X < minX || pos.X > maxX {
		t.Errorf("Sun X position %f should be between %f and %f", pos.X, minX, maxX)
	}

	// 初始Y应该在屏幕顶部外(-50)
	if pos.Y != -50 {
		t.Errorf("Sun initial Y should be -50, got %f", pos.Y)
	}

	// 验证目标Y在范围内
	sunComp, _ := em.GetComponent(id, reflect.TypeOf(&components.SunComponent{}))
	sun := sunComp.(*components.SunComponent)

	if sun.TargetY < minY || sun.TargetY > maxY {
		t.Errorf("Sun target Y %f should be between %f and %f", sun.TargetY, minY, maxY)
	}
}

func TestSunSpawnInitialState(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewSunSpawnSystem(em, rm, 250, 900, 100, 550)

	// 生成一个阳光
	system.Update(8.0)

	entities := em.GetEntitiesWith(reflect.TypeOf(&components.SunComponent{}))
	if len(entities) != 1 {
		t.Fatal("Should spawn 1 sun")
	}

	id := entities[0]

	// 验证阳光初始状态为 Falling
	sunComp, _ := em.GetComponent(id, reflect.TypeOf(&components.SunComponent{}))
	sun := sunComp.(*components.SunComponent)

	if sun.State != components.SunFalling {
		t.Error("Sun initial state should be Falling")
	}

	// 验证速度组件
	velComp, found := em.GetComponent(id, reflect.TypeOf(&components.VelocityComponent{}))
	if !found {
		t.Fatal("Sun should have velocity component")
	}
	vel := velComp.(*components.VelocityComponent)

	if vel.VX != 0 || vel.VY != 60 {
		t.Errorf("Sun velocity should be (0, 60), got (%f, %f)", vel.VX, vel.VY)
	}

	// 验证生命周期组件
	lifetimeComp, found := em.GetComponent(id, reflect.TypeOf(&components.LifetimeComponent{}))
	if !found {
		t.Fatal("Sun should have lifetime component")
	}
	lifetime := lifetimeComp.(*components.LifetimeComponent)

	if lifetime.MaxLifetime != 15.0 {
		t.Errorf("Sun max lifetime should be 15.0, got %f", lifetime.MaxLifetime)
	}

	if lifetime.CurrentLifetime != 0 {
		t.Errorf("Sun current lifetime should be 0, got %f", lifetime.CurrentLifetime)
	}
}

func TestSunSpawnTimerReset(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewSunSpawnSystem(em, rm, 250, 900, 100, 550)

	// 第一次生成
	system.Update(8.0)

	entities := em.GetEntitiesWith(reflect.TypeOf(&components.SunComponent{}))
	if len(entities) != 1 {
		t.Error("Should spawn 1 sun after 8 seconds")
	}

	// 更新5秒(计时器应该重置后累加)
	system.Update(5.0)

	entities = em.GetEntitiesWith(reflect.TypeOf(&components.SunComponent{}))
	if len(entities) != 1 {
		t.Error("Should not spawn again before next interval")
	}

	// 再更新3秒(总计又8秒)
	system.Update(3.0)

	entities = em.GetEntitiesWith(reflect.TypeOf(&components.SunComponent{}))
	if len(entities) != 2 {
		t.Errorf("Should spawn 2nd sun after another 8 seconds, got %d total", len(entities))
	}
}
