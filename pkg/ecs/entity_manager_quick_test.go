package ecs

import (
	"reflect"
	"testing"
)

// ========== 快速性能验证测试（小数据集） ==========

// TestPerformanceComparison_Quick 快速对比反射 vs 泛型性能
// 使用小数据集快速验证性能差异
func TestPerformanceComparison_Quick(t *testing.T) {
	const entityCount = 100
	const iterations = 1000

	// 创建测试数据
	em := setupBenchmarkEntities(entityCount, 3)

	// 测试反射版本 GetEntitiesWith
	reflectionStart := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < iterations; i++ {
			_ = em.GetEntitiesWith(
				reflect.TypeOf(&benchmarkComp1{}),
				reflect.TypeOf(&benchmarkComp2{}),
				reflect.TypeOf(&benchmarkComp3{}),
			)
		}
	})

	// 测试泛型版本 GetEntitiesWith
	genericStart := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < iterations; i++ {
			_ = GetEntitiesWith3[*benchmarkComp1, *benchmarkComp2, *benchmarkComp3](em)
		}
	})

	// 计算性能提升百分比
	reflectionTime := float64(reflectionStart.NsPerOp())
	genericTime := float64(genericStart.NsPerOp())
	improvement := ((reflectionTime - genericTime) / reflectionTime) * 100

	t.Logf("=== GetEntitiesWith 性能对比 ===")
	t.Logf("反射版本: %d ns/op", reflectionStart.NsPerOp())
	t.Logf("泛型版本: %d ns/op", genericStart.NsPerOp())
	t.Logf("性能提升: %.2f%%", improvement)

	if improvement < 0 {
		t.Logf("⚠️  警告：泛型版本比反射版本慢 %.2f%%", -improvement)
	} else if improvement < 30 {
		t.Logf("⚠️  警告：性能提升 %.2f%% 低于目标 30%%", improvement)
	} else {
		t.Logf("✅ 性能提升 %.2f%% 达到预期（≥30%%）", improvement)
	}

	// 测试 GetComponent
	entity := EntityID(1)

	reflectionGetComp := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < iterations; i++ {
			comp, ok := em.GetComponent(entity, reflect.TypeOf(&benchmarkComp1{}))
			if ok {
				_ = comp.(*benchmarkComp1)
			}
		}
	})

	genericGetComp := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < iterations; i++ {
			_, _ = GetComponent[*benchmarkComp1](em, entity)
		}
	})

	reflectionGetTime := float64(reflectionGetComp.NsPerOp())
	genericGetTime := float64(genericGetComp.NsPerOp())
	getImprov := ((reflectionGetTime - genericGetTime) / reflectionGetTime) * 100

	t.Logf("\n=== GetComponent 性能对比 ===")
	t.Logf("反射版本: %d ns/op", reflectionGetComp.NsPerOp())
	t.Logf("泛型版本: %d ns/op", genericGetComp.NsPerOp())
	t.Logf("性能提升: %.2f%%", getImprov)

	if getImprov >= 30 {
		t.Logf("✅ 性能提升 %.2f%% 达到预期（≥30%%）", getImprov)
	}
}

// TestGenericAPI_Correctness 验证泛型 API 的正确性
func TestGenericAPI_Correctness(t *testing.T) {
	em := NewEntityManager()

	// 创建实体
	entity := em.CreateEntity()

	// 测试 AddComponent
	t.Run("AddComponent", func(t *testing.T) {
		comp1 := &benchmarkComp1{Value1: 42, Value2: 3.14}
		AddComponent(em, entity, comp1)

		// 验证组件已添加
		if !HasComponent[*benchmarkComp1](em, entity) {
			t.Fatal("AddComponent 失败：组件未添加")
		}
	})

	// 测试 GetComponent
	t.Run("GetComponent", func(t *testing.T) {
		comp, ok := GetComponent[*benchmarkComp1](em, entity)
		if !ok {
			t.Fatal("GetComponent 失败：组件不存在")
		}
		if comp.Value1 != 42 || comp.Value2 != 3.14 {
			t.Fatalf("GetComponent 失败：组件值不正确 (Value1=%d, Value2=%f)", comp.Value1, comp.Value2)
		}
	})

	// 测试 HasComponent
	t.Run("HasComponent", func(t *testing.T) {
		if !HasComponent[*benchmarkComp1](em, entity) {
			t.Fatal("HasComponent 失败：应返回 true")
		}
		if HasComponent[*benchmarkComp5](em, entity) {
			t.Fatal("HasComponent 失败：应返回 false（组件不存在）")
		}
	})

	// 测试 GetEntitiesWith
	t.Run("GetEntitiesWith", func(t *testing.T) {
		// 添加更多组件
		AddComponent(em, entity, &benchmarkComp2{Name: "Test", Data: nil})
		AddComponent(em, entity, &benchmarkComp3{X: 1.0, Y: 2.0, Angle: 0.0})

		// 查询拥有 3 个组件的实体
		entities := GetEntitiesWith3[*benchmarkComp1, *benchmarkComp2, *benchmarkComp3](em)
		if len(entities) != 1 {
			t.Fatalf("GetEntitiesWith3 失败：期望 1 个实体，实际 %d 个", len(entities))
		}
		if entities[0] != entity {
			t.Fatal("GetEntitiesWith3 失败：返回的实体ID不正确")
		}

		// 查询拥有 2 个组件的实体
		entities2 := GetEntitiesWith2[*benchmarkComp1, *benchmarkComp2](em)
		if len(entities2) != 1 {
			t.Fatalf("GetEntitiesWith2 失败：期望 1 个实体，实际 %d 个", len(entities2))
		}

		// 查询拥有 1 个组件的实体
		entities1 := GetEntitiesWith1[*benchmarkComp1](em)
		if len(entities1) != 1 {
			t.Fatalf("GetEntitiesWith1 失败：期望 1 个实体，实际 %d 个", len(entities1))
		}
	})

	// 测试多实体场景
	t.Run("MultipleEntities", func(t *testing.T) {
		em2 := NewEntityManager()

		// 创建 10 个实体，都有 comp1 和 comp2
		for i := 0; i < 10; i++ {
			e := em2.CreateEntity()
			AddComponent(em2, e, &benchmarkComp1{Value1: i, Value2: float64(i)})
			AddComponent(em2, e, &benchmarkComp2{Name: "Test", Data: nil})
		}

		// 查询拥有 comp1 和 comp2 的实体
		entities := GetEntitiesWith2[*benchmarkComp1, *benchmarkComp2](em2)
		if len(entities) != 10 {
			t.Fatalf("MultipleEntities 失败：期望 10 个实体，实际 %d 个", len(entities))
		}
	})
}
