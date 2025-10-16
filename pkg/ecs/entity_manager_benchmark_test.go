package ecs

import (
	"reflect"
	"testing"
)

// ========== 测试组件定义 ==========

type benchmarkComp1 struct {
	Value1 int
	Value2 float64
}

type benchmarkComp2 struct {
	Name string
	Data []byte
}

type benchmarkComp3 struct {
	X, Y  float64
	Angle float64
}

type benchmarkComp4 struct {
	Health    int
	MaxHealth int
}

type benchmarkComp5 struct {
	Active bool
	Timer  float64
}

// ========== 辅助函数：创建测试数据 ==========

// setupBenchmarkEntities 创建指定数量的实体，每个实体包含指定组件
func setupBenchmarkEntities(count int, compsPerEntity int) *EntityManager {
	em := NewEntityManager()

	for i := 0; i < count; i++ {
		entity := em.CreateEntity()

		// 根据 compsPerEntity 添加组件
		if compsPerEntity >= 1 {
			em.AddComponent(entity, &benchmarkComp1{Value1: i, Value2: float64(i) * 1.5})
		}
		if compsPerEntity >= 2 {
			em.AddComponent(entity, &benchmarkComp2{Name: "Entity", Data: make([]byte, 10)})
		}
		if compsPerEntity >= 3 {
			em.AddComponent(entity, &benchmarkComp3{X: float64(i), Y: float64(i * 2), Angle: 0.0})
		}
		if compsPerEntity >= 4 {
			em.AddComponent(entity, &benchmarkComp4{Health: 100, MaxHealth: 100})
		}
		if compsPerEntity >= 5 {
			em.AddComponent(entity, &benchmarkComp5{Active: true, Timer: 0.0})
		}
	}

	return em
}

// ========== 基准测试：GetEntitiesWith（反射 vs 泛型）==========

// BenchmarkGetEntitiesWith_Reflection 测试反射版本查询 1000 实体（3组件）
func BenchmarkGetEntitiesWith_Reflection(b *testing.B) {
	em := setupBenchmarkEntities(1000, 3)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = em.GetEntitiesWith(
			reflect.TypeOf(&benchmarkComp1{}),
			reflect.TypeOf(&benchmarkComp2{}),
			reflect.TypeOf(&benchmarkComp3{}),
		)
	}
}

// BenchmarkGetEntitiesWith_Generic 测试泛型版本查询 1000 实体（3组件）
func BenchmarkGetEntitiesWith_Generic(b *testing.B) {
	em := setupBenchmarkEntities(1000, 3)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetEntitiesWith3[*benchmarkComp1, *benchmarkComp2, *benchmarkComp3](em)
	}
}

// BenchmarkGetEntitiesWith_Reflection_1Comp 测试反射版本查询 1000 实体（1组件）
func BenchmarkGetEntitiesWith_Reflection_1Comp(b *testing.B) {
	em := setupBenchmarkEntities(1000, 3)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = em.GetEntitiesWith(reflect.TypeOf(&benchmarkComp1{}))
	}
}

// BenchmarkGetEntitiesWith_Generic_1Comp 测试泛型版本查询 1000 实体（1组件）
func BenchmarkGetEntitiesWith_Generic_1Comp(b *testing.B) {
	em := setupBenchmarkEntities(1000, 3)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetEntitiesWith1[*benchmarkComp1](em)
	}
}

// BenchmarkGetEntitiesWith_Reflection_5Comp 测试反射版本查询 1000 实体（5组件）
func BenchmarkGetEntitiesWith_Reflection_5Comp(b *testing.B) {
	em := setupBenchmarkEntities(1000, 5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = em.GetEntitiesWith(
			reflect.TypeOf(&benchmarkComp1{}),
			reflect.TypeOf(&benchmarkComp2{}),
			reflect.TypeOf(&benchmarkComp3{}),
			reflect.TypeOf(&benchmarkComp4{}),
			reflect.TypeOf(&benchmarkComp5{}),
		)
	}
}

// BenchmarkGetEntitiesWith_Generic_5Comp 测试泛型版本查询 1000 实体（5组件）
func BenchmarkGetEntitiesWith_Generic_5Comp(b *testing.B) {
	em := setupBenchmarkEntities(1000, 5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetEntitiesWith5[*benchmarkComp1, *benchmarkComp2, *benchmarkComp3, *benchmarkComp4, *benchmarkComp5](em)
	}
}

// ========== 基准测试：GetComponent（反射 vs 泛型）==========

// BenchmarkGetComponent_Reflection 测试反射版本获取单个组件
func BenchmarkGetComponent_Reflection(b *testing.B) {
	em := setupBenchmarkEntities(1, 3)
	entity := EntityID(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		comp, ok := em.GetComponent(entity, reflect.TypeOf(&benchmarkComp1{}))
		if ok {
			_ = comp.(*benchmarkComp1) // 类型断言
		}
	}
}

// BenchmarkGetComponent_Generic 测试泛型版本获取单个组件
func BenchmarkGetComponent_Generic(b *testing.B) {
	em := setupBenchmarkEntities(1, 3)
	entity := EntityID(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, ok := GetComponent[*benchmarkComp1](em, entity)
		if !ok {
			b.Fatal("component not found")
		}
	}
}

// BenchmarkGetComponent_Reflection_NotFound 测试反射版本获取不存在的组件
func BenchmarkGetComponent_Reflection_NotFound(b *testing.B) {
	em := setupBenchmarkEntities(1, 1) // 只有 comp1
	entity := EntityID(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = em.GetComponent(entity, reflect.TypeOf(&benchmarkComp5{}))
	}
}

// BenchmarkGetComponent_Generic_NotFound 测试泛型版本获取不存在的组件
func BenchmarkGetComponent_Generic_NotFound(b *testing.B) {
	em := setupBenchmarkEntities(1, 1) // 只有 comp1
	entity := EntityID(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetComponent[*benchmarkComp5](em, entity)
	}
}

// ========== 基准测试：AddComponent（反射 vs 泛型）==========

// BenchmarkAddComponent_Reflection 测试反射版本添加组件
func BenchmarkAddComponent_Reflection(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		em := NewEntityManager()
		entity := em.CreateEntity()
		b.StartTimer()

		em.AddComponent(entity, &benchmarkComp1{Value1: 42, Value2: 3.14})
	}
}

// BenchmarkAddComponent_Generic 测试泛型版本添加组件
func BenchmarkAddComponent_Generic(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		em := NewEntityManager()
		entity := em.CreateEntity()
		b.StartTimer()

		AddComponent(em, entity, &benchmarkComp1{Value1: 42, Value2: 3.14})
	}
}

// BenchmarkAddComponent_Reflection_Multiple 测试反射版本批量添加组件
func BenchmarkAddComponent_Reflection_Multiple(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		em := NewEntityManager()
		entity := em.CreateEntity()
		b.StartTimer()

		em.AddComponent(entity, &benchmarkComp1{Value1: 1, Value2: 1.0})
		em.AddComponent(entity, &benchmarkComp2{Name: "Test", Data: nil})
		em.AddComponent(entity, &benchmarkComp3{X: 0, Y: 0, Angle: 0})
	}
}

// BenchmarkAddComponent_Generic_Multiple 测试泛型版本批量添加组件
func BenchmarkAddComponent_Generic_Multiple(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		em := NewEntityManager()
		entity := em.CreateEntity()
		b.StartTimer()

		AddComponent(em, entity, &benchmarkComp1{Value1: 1, Value2: 1.0})
		AddComponent(em, entity, &benchmarkComp2{Name: "Test", Data: nil})
		AddComponent(em, entity, &benchmarkComp3{X: 0, Y: 0, Angle: 0})
	}
}

// ========== 基准测试：HasComponent（反射 vs 泛型）==========

// BenchmarkHasComponent_Reflection 测试反射版本检查组件存在性
func BenchmarkHasComponent_Reflection(b *testing.B) {
	em := setupBenchmarkEntities(1, 3)
	entity := EntityID(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = em.HasComponent(entity, reflect.TypeOf(&benchmarkComp1{}))
	}
}

// BenchmarkHasComponent_Generic 测试泛型版本检查组件存在性
func BenchmarkHasComponent_Generic(b *testing.B) {
	em := setupBenchmarkEntities(1, 3)
	entity := EntityID(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = HasComponent[*benchmarkComp1](em, entity)
	}
}

// ========== 综合基准测试：模拟真实系统更新循环 ==========

// BenchmarkSystemUpdate_Reflection 模拟反射版本的系统更新循环
func BenchmarkSystemUpdate_Reflection(b *testing.B) {
	em := setupBenchmarkEntities(100, 3)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 查询实体
		entities := em.GetEntitiesWith(
			reflect.TypeOf(&benchmarkComp1{}),
			reflect.TypeOf(&benchmarkComp3{}),
		)

		// 处理每个实体
		for _, entity := range entities {
			comp1, ok := em.GetComponent(entity, reflect.TypeOf(&benchmarkComp1{}))
			if !ok {
				continue
			}
			c1 := comp1.(*benchmarkComp1)

			comp3, ok := em.GetComponent(entity, reflect.TypeOf(&benchmarkComp3{}))
			if !ok {
				continue
			}
			c3 := comp3.(*benchmarkComp3)

			// 模拟更新逻辑
			c1.Value1++
			c3.X += 1.0
		}
	}
}

// BenchmarkSystemUpdate_Generic 模拟泛型版本的系统更新循环
func BenchmarkSystemUpdate_Generic(b *testing.B) {
	em := setupBenchmarkEntities(100, 3)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 查询实体
		entities := GetEntitiesWith2[*benchmarkComp1, *benchmarkComp3](em)

		// 处理每个实体
		for _, entity := range entities {
			c1, ok := GetComponent[*benchmarkComp1](em, entity)
			if !ok {
				continue
			}

			c3, ok := GetComponent[*benchmarkComp3](em, entity)
			if !ok {
				continue
			}

			// 模拟更新逻辑
			c1.Value1++
			c3.X += 1.0
		}
	}
}
