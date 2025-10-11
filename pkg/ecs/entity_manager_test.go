package ecs

import (
	"reflect"
	"testing"
)

// 测试组件类型定义
type testPositionComponent struct {
	X, Y float64
}

type testVelocityComponent struct {
	VX, VY float64
}

func TestCreateEntity(t *testing.T) {
	em := NewEntityManager()
	id1 := em.CreateEntity()
	id2 := em.CreateEntity()

	// 测试实体ID唯一性
	if id1 == id2 {
		t.Error("Entity IDs should be unique")
	}

	// 测试ID从1开始
	if id1 != 1 {
		t.Errorf("First entity ID should be 1, got %d", id1)
	}

	if id2 != 2 {
		t.Errorf("Second entity ID should be 2, got %d", id2)
	}
}

func TestAddAndGetComponent(t *testing.T) {
	em := NewEntityManager()
	id := em.CreateEntity()

	// 添加组件
	pos := &testPositionComponent{X: 100, Y: 200}
	em.AddComponent(id, pos)

	// 获取组件
	comp, found := em.GetComponent(id, reflect.TypeOf(&testPositionComponent{}))
	if !found {
		t.Error("Component should be found")
	}

	retrieved := comp.(*testPositionComponent)
	if retrieved.X != 100 || retrieved.Y != 200 {
		t.Errorf("Component data mismatch, expected (100, 200), got (%f, %f)", retrieved.X, retrieved.Y)
	}
}

func TestHasComponent(t *testing.T) {
	em := NewEntityManager()
	id := em.CreateEntity()

	// 未添加组件前应该返回false
	if em.HasComponent(id, reflect.TypeOf(&testPositionComponent{})) {
		t.Error("Should not have component before adding")
	}

	// 添加组件
	em.AddComponent(id, &testPositionComponent{})

	// 添加后应该返回true
	if !em.HasComponent(id, reflect.TypeOf(&testPositionComponent{})) {
		t.Error("Should have component after adding")
	}
}

func TestDestroyEntity(t *testing.T) {
	em := NewEntityManager()
	id := em.CreateEntity()
	em.AddComponent(id, &testPositionComponent{})

	// 标记删除
	em.DestroyEntity(id)

	// 清理前实体仍存在
	if !em.HasComponent(id, reflect.TypeOf(&testPositionComponent{})) {
		t.Error("Entity should still exist before cleanup")
	}

	// 清理后实体消失
	em.RemoveMarkedEntities()
	if em.HasComponent(id, reflect.TypeOf(&testPositionComponent{})) {
		t.Error("Entity should be removed after cleanup")
	}
}

func TestGetEntitiesWith(t *testing.T) {
	em := NewEntityManager()

	// 创建不同组件组合的实体
	id1 := em.CreateEntity()
	em.AddComponent(id1, &testPositionComponent{})
	em.AddComponent(id1, &testVelocityComponent{})

	id2 := em.CreateEntity()
	em.AddComponent(id2, &testPositionComponent{})

	id3 := em.CreateEntity()
	em.AddComponent(id3, &testVelocityComponent{})

	// 查询拥有 Position+Velocity 的实体
	entities := em.GetEntitiesWith(
		reflect.TypeOf(&testPositionComponent{}),
		reflect.TypeOf(&testVelocityComponent{}),
	)

	if len(entities) != 1 {
		t.Errorf("Expected 1 entity with both components, got %d", len(entities))
	}

	if len(entities) > 0 && entities[0] != id1 {
		t.Error("Query should return only id1")
	}

	// 查询只拥有 Position 的实体
	posEntities := em.GetEntitiesWith(reflect.TypeOf(&testPositionComponent{}))
	if len(posEntities) != 2 {
		t.Errorf("Expected 2 entities with Position component, got %d", len(posEntities))
	}
}

func TestMultipleComponentTypes(t *testing.T) {
	em := NewEntityManager()
	id := em.CreateEntity()

	// 添加多个不同类型的组件
	em.AddComponent(id, &testPositionComponent{X: 10, Y: 20})
	em.AddComponent(id, &testVelocityComponent{VX: 5, VY: 10})

	// 验证两个组件都能正确获取
	posComp, found := em.GetComponent(id, reflect.TypeOf(&testPositionComponent{}))
	if !found {
		t.Error("Position component should be found")
	}
	pos := posComp.(*testPositionComponent)
	if pos.X != 10 || pos.Y != 20 {
		t.Error("Position component data mismatch")
	}

	velComp, found := em.GetComponent(id, reflect.TypeOf(&testVelocityComponent{}))
	if !found {
		t.Error("Velocity component should be found")
	}
	vel := velComp.(*testVelocityComponent)
	if vel.VX != 5 || vel.VY != 10 {
		t.Error("Velocity component data mismatch")
	}
}

func TestDestroyMultipleEntities(t *testing.T) {
	em := NewEntityManager()

	// 创建多个实体
	id1 := em.CreateEntity()
	id2 := em.CreateEntity()
	id3 := em.CreateEntity()

	em.AddComponent(id1, &testPositionComponent{})
	em.AddComponent(id2, &testPositionComponent{})
	em.AddComponent(id3, &testPositionComponent{})

	// 标记两个实体删除
	em.DestroyEntity(id1)
	em.DestroyEntity(id3)

	// 清理
	em.RemoveMarkedEntities()

	// 验证只有id2存在
	if em.HasComponent(id1, reflect.TypeOf(&testPositionComponent{})) {
		t.Error("id1 should be removed")
	}
	if !em.HasComponent(id2, reflect.TypeOf(&testPositionComponent{})) {
		t.Error("id2 should still exist")
	}
	if em.HasComponent(id3, reflect.TypeOf(&testPositionComponent{})) {
		t.Error("id3 should be removed")
	}
}
