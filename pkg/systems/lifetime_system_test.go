package systems

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
)

func TestLifetimeUpdate(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewLifetimeSystem(em)

	// 创建测试实体
	id := em.CreateEntity()
	em.AddComponent(id, &components.LifetimeComponent{
		MaxLifetime:     10.0,
		CurrentLifetime: 0,
		IsExpired:       false,
	})

	// 模拟5秒更新
	system.Update(5.0)

	// 验证生命周期增加
	lifetimeComp, _ := em.GetComponent(id, reflect.TypeOf(&components.LifetimeComponent{}))
	lifetime := lifetimeComp.(*components.LifetimeComponent)

	if lifetime.CurrentLifetime != 5.0 {
		t.Errorf("Expected CurrentLifetime=5.0, got %f", lifetime.CurrentLifetime)
	}

	if lifetime.IsExpired {
		t.Error("Entity should not be expired yet")
	}
}

func TestLifetimeExpiration(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewLifetimeSystem(em)

	// 创建测试实体
	id := em.CreateEntity()
	em.AddComponent(id, &components.LifetimeComponent{
		MaxLifetime:     10.0,
		CurrentLifetime: 0,
		IsExpired:       false,
	})

	// 模拟超过最大生命周期
	system.Update(12.0)

	// 验证过期标记
	lifetimeComp, _ := em.GetComponent(id, reflect.TypeOf(&components.LifetimeComponent{}))
	lifetime := lifetimeComp.(*components.LifetimeComponent)

	if !lifetime.IsExpired {
		t.Error("Entity should be expired")
	}

	// 清理实体
	em.RemoveMarkedEntities()

	// 验证实体已被删除
	if em.HasComponent(id, reflect.TypeOf(&components.LifetimeComponent{})) {
		t.Error("Expired entity should be removed")
	}
}

func TestLifetimeMultipleUpdates(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewLifetimeSystem(em)

	// 创建测试实体
	id := em.CreateEntity()
	em.AddComponent(id, &components.LifetimeComponent{
		MaxLifetime:     10.0,
		CurrentLifetime: 0,
		IsExpired:       false,
	})

	// 多次小步更新
	system.Update(3.0)
	system.Update(3.0)
	system.Update(3.0)

	// 验证累积生命周期
	lifetimeComp, _ := em.GetComponent(id, reflect.TypeOf(&components.LifetimeComponent{}))
	lifetime := lifetimeComp.(*components.LifetimeComponent)

	if lifetime.CurrentLifetime != 9.0 {
		t.Errorf("Expected CurrentLifetime=9.0, got %f", lifetime.CurrentLifetime)
	}

	// 再更新一次,应该过期
	system.Update(2.0)

	lifetimeComp, _ = em.GetComponent(id, reflect.TypeOf(&components.LifetimeComponent{}))
	lifetime = lifetimeComp.(*components.LifetimeComponent)

	if !lifetime.IsExpired {
		t.Error("Entity should be expired after exceeding MaxLifetime")
	}
}

func TestMultipleEntitiesWithDifferentLifetimes(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewLifetimeSystem(em)

	// 创建多个实体,不同生命周期
	id1 := em.CreateEntity()
	em.AddComponent(id1, &components.LifetimeComponent{
		MaxLifetime:     5.0,
		CurrentLifetime: 0,
		IsExpired:       false,
	})

	id2 := em.CreateEntity()
	em.AddComponent(id2, &components.LifetimeComponent{
		MaxLifetime:     10.0,
		CurrentLifetime: 0,
		IsExpired:       false,
	})

	// 模拟7秒更新
	system.Update(7.0)

	// 清理过期实体
	em.RemoveMarkedEntities()

	// 验证第一个实体已删除
	if em.HasComponent(id1, reflect.TypeOf(&components.LifetimeComponent{})) {
		t.Error("Entity 1 should be removed (expired)")
	}

	// 验证第二个实体仍存在
	if !em.HasComponent(id2, reflect.TypeOf(&components.LifetimeComponent{})) {
		t.Error("Entity 2 should still exist")
	}

	// 验证第二个实体未过期
	lifetimeComp, _ := em.GetComponent(id2, reflect.TypeOf(&components.LifetimeComponent{}))
	lifetime := lifetimeComp.(*components.LifetimeComponent)

	if lifetime.IsExpired {
		t.Error("Entity 2 should not be expired yet")
	}
}




