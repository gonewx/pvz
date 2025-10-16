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

// ========== 泛型 API 单元测试（Story 9.1） ==========

// TestGenericGetComponent 测试泛型版本的 GetComponent
func TestGenericGetComponent(t *testing.T) {
	em := NewEntityManager()
	entity := em.CreateEntity()

	t.Run("GetExistingComponent", func(t *testing.T) {
		// 添加组件
		pos := &testPositionComponent{X: 150, Y: 250}
		AddComponent(em, entity, pos)

		// 获取组件 - 无需类型断言
		retrieved, ok := GetComponent[*testPositionComponent](em, entity)
		if !ok {
			t.Fatal("Component should be found")
		}
		if retrieved.X != 150 || retrieved.Y != 250 {
			t.Errorf("Component data mismatch, expected (150, 250), got (%f, %f)", retrieved.X, retrieved.Y)
		}
	})

	t.Run("GetNonExistingComponent", func(t *testing.T) {
		// 尝试获取不存在的组件
		_, ok := GetComponent[*testVelocityComponent](em, entity)
		if ok {
			t.Error("Should not find non-existing component")
		}
	})

	t.Run("TypeSafety", func(t *testing.T) {
		// 泛型版本确保类型安全，编译时检查
		// 这个测试主要验证返回值类型正确
		retrieved, ok := GetComponent[*testPositionComponent](em, entity)
		if ok {
			// 无需类型断言，直接使用
			_ = retrieved.X
			_ = retrieved.Y
			// 如果类型不匹配，编译器会报错
		}
	})

	t.Run("GetFromNonExistentEntity", func(t *testing.T) {
		// 测试从不存在的实体获取组件
		_, ok := GetComponent[*testPositionComponent](em, EntityID(9999))
		if ok {
			t.Error("Should not find component from non-existent entity")
		}
	})
}

// TestGenericAddComponent 测试泛型版本的 AddComponent
func TestGenericAddComponent(t *testing.T) {
	em := NewEntityManager()
	entity := em.CreateEntity()

	t.Run("AddNewComponent", func(t *testing.T) {
		// 添加新组件
		AddComponent(em, entity, &testPositionComponent{X: 100, Y: 200})

		// 验证组件已添加
		if !HasComponent[*testPositionComponent](em, entity) {
			t.Error("Component should be added")
		}

		// 验证组件数据正确
		comp, ok := GetComponent[*testPositionComponent](em, entity)
		if !ok {
			t.Fatal("Component should exist")
		}
		if comp.X != 100 || comp.Y != 200 {
			t.Error("Component data incorrect")
		}
	})

	t.Run("OverwriteExistingComponent", func(t *testing.T) {
		// 再次添加同类型组件（覆盖）
		AddComponent(em, entity, &testPositionComponent{X: 300, Y: 400})

		// 验证组件数据已更新
		comp, ok := GetComponent[*testPositionComponent](em, entity)
		if !ok {
			t.Fatal("Component should exist")
		}
		if comp.X != 300 || comp.Y != 400 {
			t.Errorf("Component data should be updated, got (%f, %f)", comp.X, comp.Y)
		}
	})

	t.Run("AddMultipleComponentTypes", func(t *testing.T) {
		// 添加多个不同类型的组件
		em2 := NewEntityManager()
		entity2 := em2.CreateEntity()

		AddComponent(em2, entity2, &testPositionComponent{X: 10, Y: 20})
		AddComponent(em2, entity2, &testVelocityComponent{VX: 5, VY: 10})

		// 验证两个组件都存在
		if !HasComponent[*testPositionComponent](em2, entity2) {
			t.Error("Position component should exist")
		}
		if !HasComponent[*testVelocityComponent](em2, entity2) {
			t.Error("Velocity component should exist")
		}
	})
}

// TestGenericHasComponent 测试泛型版本的 HasComponent
func TestGenericHasComponent(t *testing.T) {
	em := NewEntityManager()
	entity := em.CreateEntity()

	t.Run("ExistingComponent", func(t *testing.T) {
		AddComponent(em, entity, &testPositionComponent{X: 1, Y: 2})

		if !HasComponent[*testPositionComponent](em, entity) {
			t.Error("Should return true for existing component")
		}
	})

	t.Run("NonExistingComponent", func(t *testing.T) {
		if HasComponent[*testVelocityComponent](em, entity) {
			t.Error("Should return false for non-existing component")
		}
	})

	t.Run("AfterAddingComponent", func(t *testing.T) {
		em2 := NewEntityManager()
		entity2 := em2.CreateEntity()

		// 添加前
		if HasComponent[*testPositionComponent](em2, entity2) {
			t.Error("Should return false before adding")
		}

		// 添加后
		AddComponent(em2, entity2, &testPositionComponent{})
		if !HasComponent[*testPositionComponent](em2, entity2) {
			t.Error("Should return true after adding")
		}
	})

	t.Run("NonExistentEntity", func(t *testing.T) {
		// 测试不存在的实体
		if HasComponent[*testPositionComponent](em, EntityID(9999)) {
			t.Error("Should return false for non-existent entity")
		}
	})
}

// TestGenericGetEntitiesWith 测试泛型版本的 GetEntitiesWith 系列函数
func TestGenericGetEntitiesWith(t *testing.T) {
	t.Run("GetEntitiesWith1", func(t *testing.T) {
		em := NewEntityManager()

		// 创建实体
		e1 := em.CreateEntity()
		AddComponent(em, e1, &testPositionComponent{X: 1, Y: 1})

		e2 := em.CreateEntity()
		AddComponent(em, e2, &testPositionComponent{X: 2, Y: 2})

		e3 := em.CreateEntity()
		AddComponent(em, e3, &testVelocityComponent{VX: 1, VY: 1})

		// 查询拥有 Position 组件的实体
		entities := GetEntitiesWith1[*testPositionComponent](em)
		if len(entities) != 2 {
			t.Errorf("Expected 2 entities, got %d", len(entities))
		}

		// 验证返回的实体ID正确
		hasE1, hasE2 := false, false
		for _, e := range entities {
			if e == e1 {
				hasE1 = true
			}
			if e == e2 {
				hasE2 = true
			}
		}
		if !hasE1 || !hasE2 {
			t.Error("Should return e1 and e2")
		}
	})

	t.Run("GetEntitiesWith2", func(t *testing.T) {
		em := NewEntityManager()

		// 创建实体：同时拥有 Position + Velocity
		e1 := em.CreateEntity()
		AddComponent(em, e1, &testPositionComponent{X: 1, Y: 1})
		AddComponent(em, e1, &testVelocityComponent{VX: 1, VY: 1})

		// 创建实体：只有 Position
		e2 := em.CreateEntity()
		AddComponent(em, e2, &testPositionComponent{X: 2, Y: 2})

		// 创建实体：只有 Velocity
		e3 := em.CreateEntity()
		AddComponent(em, e3, &testVelocityComponent{VX: 2, VY: 2})

		// 查询拥有两个组件的实体
		entities := GetEntitiesWith2[*testPositionComponent, *testVelocityComponent](em)
		if len(entities) != 1 {
			t.Errorf("Expected 1 entity, got %d", len(entities))
		}
		if len(entities) > 0 && entities[0] != e1 {
			t.Error("Should return only e1")
		}
	})

	t.Run("GetEntitiesWith3", func(t *testing.T) {
		em := NewEntityManager()

		// 定义第三个测试组件
		type testHealthComponent struct {
			Health int
		}

		// 创建实体：拥有 3 个组件
		e1 := em.CreateEntity()
		AddComponent(em, e1, &testPositionComponent{X: 1, Y: 1})
		AddComponent(em, e1, &testVelocityComponent{VX: 1, VY: 1})
		AddComponent(em, e1, &testHealthComponent{Health: 100})

		// 创建实体：只有 2 个组件
		e2 := em.CreateEntity()
		AddComponent(em, e2, &testPositionComponent{X: 2, Y: 2})
		AddComponent(em, e2, &testVelocityComponent{VX: 2, VY: 2})

		// 查询拥有 3 个组件的实体
		entities := GetEntitiesWith3[
			*testPositionComponent,
			*testVelocityComponent,
			*testHealthComponent,
		](em)

		if len(entities) != 1 {
			t.Errorf("Expected 1 entity, got %d", len(entities))
		}
		if len(entities) > 0 && entities[0] != e1 {
			t.Error("Should return only e1")
		}
	})

	t.Run("EmptyResultQuery", func(t *testing.T) {
		em := NewEntityManager()

		// 创建实体但不添加任何组件
		em.CreateEntity()
		em.CreateEntity()

		// 查询应返回空结果
		entities := GetEntitiesWith1[*testPositionComponent](em)
		if len(entities) != 0 {
			t.Errorf("Expected 0 entities, got %d", len(entities))
		}
	})

	t.Run("LargeScaleQuery", func(t *testing.T) {
		em := NewEntityManager()

		// 创建 100 个实体
		count := 100
		for i := 0; i < count; i++ {
			e := em.CreateEntity()
			AddComponent(em, e, &testPositionComponent{X: float64(i), Y: float64(i)})

			// 只有一半的实体有 Velocity 组件
			if i%2 == 0 {
				AddComponent(em, e, &testVelocityComponent{VX: 1, VY: 1})
			}
		}

		// 查询拥有 Position 组件的实体
		posEntities := GetEntitiesWith1[*testPositionComponent](em)
		if len(posEntities) != count {
			t.Errorf("Expected %d entities with Position, got %d", count, len(posEntities))
		}

		// 查询拥有两个组件的实体
		bothEntities := GetEntitiesWith2[*testPositionComponent, *testVelocityComponent](em)
		if len(bothEntities) != count/2 {
			t.Errorf("Expected %d entities with both components, got %d", count/2, len(bothEntities))
		}
	})
}

// TestGenericGetEntitiesWith4And5 测试 4 和 5 个组件查询
func TestGenericGetEntitiesWith4And5(t *testing.T) {
	// 定义额外的测试组件
	type testHealthComponent struct {
		Health int
	}
	type testTagComponent struct {
		Tag string
	}
	type testTimerComponent struct {
		Time float64
	}

	t.Run("GetEntitiesWith4", func(t *testing.T) {
		em := NewEntityManager()

		// 创建拥有 4 个组件的实体
		e1 := em.CreateEntity()
		AddComponent(em, e1, &testPositionComponent{X: 1, Y: 1})
		AddComponent(em, e1, &testVelocityComponent{VX: 1, VY: 1})
		AddComponent(em, e1, &testHealthComponent{Health: 100})
		AddComponent(em, e1, &testTagComponent{Tag: "player"})

		// 创建只有 3 个组件的实体
		e2 := em.CreateEntity()
		AddComponent(em, e2, &testPositionComponent{X: 2, Y: 2})
		AddComponent(em, e2, &testVelocityComponent{VX: 2, VY: 2})
		AddComponent(em, e2, &testHealthComponent{Health: 50})

		// 查询拥有 4 个组件的实体
		entities := GetEntitiesWith4[
			*testPositionComponent,
			*testVelocityComponent,
			*testHealthComponent,
			*testTagComponent,
		](em)

		if len(entities) != 1 {
			t.Errorf("Expected 1 entity, got %d", len(entities))
		}
		if len(entities) > 0 && entities[0] != e1 {
			t.Error("Should return only e1")
		}
	})

	t.Run("GetEntitiesWith5", func(t *testing.T) {
		em := NewEntityManager()

		// 创建拥有 5 个组件的实体
		e1 := em.CreateEntity()
		AddComponent(em, e1, &testPositionComponent{X: 1, Y: 1})
		AddComponent(em, e1, &testVelocityComponent{VX: 1, VY: 1})
		AddComponent(em, e1, &testHealthComponent{Health: 100})
		AddComponent(em, e1, &testTagComponent{Tag: "player"})
		AddComponent(em, e1, &testTimerComponent{Time: 0.0})

		// 创建只有 4 个组件的实体
		e2 := em.CreateEntity()
		AddComponent(em, e2, &testPositionComponent{X: 2, Y: 2})
		AddComponent(em, e2, &testVelocityComponent{VX: 2, VY: 2})
		AddComponent(em, e2, &testHealthComponent{Health: 50})
		AddComponent(em, e2, &testTagComponent{Tag: "enemy"})

		// 查询拥有 5 个组件的实体
		entities := GetEntitiesWith5[
			*testPositionComponent,
			*testVelocityComponent,
			*testHealthComponent,
			*testTagComponent,
			*testTimerComponent,
		](em)

		if len(entities) != 1 {
			t.Errorf("Expected 1 entity, got %d", len(entities))
		}
		if len(entities) > 0 && entities[0] != e1 {
			t.Error("Should return only e1")
		}
	})
}

// TestGenericAPIConsistency 测试泛型 API 与反射 API 的行为一致性
func TestGenericAPIConsistency(t *testing.T) {
	em := NewEntityManager()
	entity := em.CreateEntity()

	// 使用反射 API 添加组件
	em.AddComponent(entity, &testPositionComponent{X: 100, Y: 200})

	t.Run("GetComponent_Consistency", func(t *testing.T) {
		// 泛型 API 应能获取反射 API 添加的组件
		comp, ok := GetComponent[*testPositionComponent](em, entity)
		if !ok {
			t.Fatal("Generic API should find component added by reflection API")
		}
		if comp.X != 100 || comp.Y != 200 {
			t.Error("Data mismatch between APIs")
		}

		// 反射 API 也应能获取
		refComp, ok := em.GetComponent(entity, reflect.TypeOf(&testPositionComponent{}))
		if !ok {
			t.Fatal("Reflection API should find component")
		}
		refPos := refComp.(*testPositionComponent)
		if refPos.X != 100 || refPos.Y != 200 {
			t.Error("Data mismatch in reflection API")
		}
	})

	t.Run("AddComponent_Consistency", func(t *testing.T) {
		em2 := NewEntityManager()
		entity2 := em2.CreateEntity()

		// 使用泛型 API 添加组件
		AddComponent(em2, entity2, &testVelocityComponent{VX: 10, VY: 20})

		// 反射 API 应能查询到
		if !em2.HasComponent(entity2, reflect.TypeOf(&testVelocityComponent{})) {
			t.Error("Reflection API should find component added by generic API")
		}

		// 反射 API 应能获取正确数据
		refComp, ok := em2.GetComponent(entity2, reflect.TypeOf(&testVelocityComponent{}))
		if !ok {
			t.Fatal("Reflection API should get component")
		}
		refVel := refComp.(*testVelocityComponent)
		if refVel.VX != 10 || refVel.VY != 20 {
			t.Error("Data mismatch between APIs")
		}
	})

	t.Run("GetEntitiesWith_Consistency", func(t *testing.T) {
		em3 := NewEntityManager()

		// 创建测试实体
		for i := 0; i < 5; i++ {
			e := em3.CreateEntity()
			AddComponent(em3, e, &testPositionComponent{X: float64(i), Y: float64(i)})
		}

		// 泛型 API 查询
		genericEntities := GetEntitiesWith1[*testPositionComponent](em3)

		// 反射 API 查询
		reflectionEntities := em3.GetEntitiesWith(reflect.TypeOf(&testPositionComponent{}))

		// 结果应一致
		if len(genericEntities) != len(reflectionEntities) {
			t.Errorf("Result count mismatch: generic=%d, reflection=%d",
				len(genericEntities), len(reflectionEntities))
		}

		// 验证实体ID集合相同
		genericSet := make(map[EntityID]bool)
		for _, e := range genericEntities {
			genericSet[e] = true
		}

		for _, e := range reflectionEntities {
			if !genericSet[e] {
				t.Errorf("Entity %d found by reflection but not by generic API", e)
			}
		}
	})
}

