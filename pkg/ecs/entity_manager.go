// Package ecs 提供实体-组件-系统(Entity-Component-System)架构的核心实现
//
// # ECS 架构概述
//
// EntityManager 是 ECS 架构的核心，负责管理实体和组件的生命周期。
//
// # API 版本
//
// 本包提供两套 API：
//   - **泛型 API** (推荐): 提供编译时类型检查和更好的开发体验 (Story 9.1)
//   - **反射 API** (已废弃): 保留用于向后兼容，将在 Epic 9 完成后考虑移除
//
// # 泛型 API 使用指南
//
// ## 基本操作
//
// 获取组件（无需类型断言）:
//
//	plantComp, ok := ecs.GetComponent[*components.PlantComponent](em, entity)
//	if ok {
//	    plantComp.Health -= 10 // 编译时类型检查
//	}
//
// 添加组件（自动类型推导）:
//
//	ecs.AddComponent(em, entity, &components.PlantComponent{
//	    PlantType: "Peashooter",
//	    Health:    300,
//	})
//
// 检查组件存在性:
//
//	if ecs.HasComponent[*components.PlantComponent](em, entity) {
//	    // 处理植物逻辑
//	}
//
// ## 查询实体
//
// 查询拥有单个组件的实体:
//
//	entities := ecs.GetEntitiesWith1[*components.PlantComponent](em)
//
// 查询拥有多个组件的实体:
//
//	entities := ecs.GetEntitiesWith3[
//	    *components.BehaviorComponent,
//	    *components.PlantComponent,
//	    *components.PositionComponent,
//	](em)
//
// 支持 1-5 个组件的查询（GetEntitiesWith1/2/3/4/5）
//
// # 泛型 API vs 反射 API 对比
//
// ## 代码简洁性
//
// 反射 API（旧，已废弃）:
//
//	entities := em.GetEntitiesWith(
//	    reflect.TypeOf(&components.PlantComponent{}),
//	    reflect.TypeOf(&components.PositionComponent{}),
//	)
//	comp, ok := em.GetComponent(entity, reflect.TypeOf(&components.PlantComponent{}))
//	if ok {
//	    plantComp := comp.(*components.PlantComponent) // 手动类型断言
//	}
//
// 泛型 API（新，推荐）:
//
//	entities := ecs.GetEntitiesWith2[
//	    *components.PlantComponent,
//	    *components.PositionComponent,
//	](em)
//	plantComp, ok := ecs.GetComponent[*components.PlantComponent](em, entity)
//	if ok {
//	    // 无需类型断言，直接使用
//	}
//
// ## 性能对比
//
// 基于 Intel i9-14900KF 的基准测试结果：
//   - 综合系统更新循环: 泛型版本比反射版本快约 10%
//   - 大规模实体查询: 泛型版本比反射版本快约 7-8%
//
// 虽然性能提升有限，但泛型 API 的主要优势在于：
//   - ✅ 编译时类型检查（消除运行时 panic 风险）
//   - ✅ 无需手动类型断言（代码更简洁）
//   - ✅ 更好的 IDE 支持（代码补全、重构）
//   - ✅ 提升代码可读性和可维护性
//
// 详细性能报告参见: docs/architecture/ecs-generics-performance-report.md
//
// # 迁移指南
//
// 对于现有使用反射 API 的代码，请参考迁移指南:
// docs/architecture/ecs-generics-migration-guide.md
//
// # 注意事项
//
//   - 泛型类型参数必须与存储时的类型完全一致（包括指针标记）
//   - 组件统一使用指针类型（如 *PlantComponent）
//   - 泛型 API 的函数定义在包级别，调用时需要包名前缀（ecs.GetComponent）
//
// # 示例：完整的系统更新流程
//
//	func (s *MySystem) Update(dt float64) {
//	    // 1. 查询拥有特定组件的实体
//	    entities := ecs.GetEntitiesWith2[
//	        *components.PlantComponent,
//	        *components.PositionComponent,
//	    ](s.entityManager)
//
//	    // 2. 遍历并处理每个实体
//	    for _, entity := range entities {
//	        // 3. 获取组件（无需类型断言）
//	        plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, entity)
//	        if !ok {
//	            continue
//	        }
//
//	        pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entity)
//	        if !ok {
//	            continue
//	        }
//
//	        // 4. 更新组件数据
//	        plant.Health -= 1
//	        pos.X += 10 * dt
//	    }
//	}
//
package ecs

import "reflect"

// EntityID 是实体的唯一标识符
type EntityID uint64

// EntityManager 管理所有实体和组件
type EntityManager struct {
	nextID uint64
	// 实体-组件映射: EntityID -> ComponentType -> Component实例
	components map[EntityID]map[reflect.Type]interface{}
	// 待删除的实体ID列表
	entitiesToDestroy []EntityID
}

// NewEntityManager 创建一个新的 EntityManager 实例
func NewEntityManager() *EntityManager {
	return &EntityManager{
		nextID:            1, // ID从1开始,0保留为无效ID
		components:        make(map[EntityID]map[reflect.Type]interface{}),
		entitiesToDestroy: make([]EntityID, 0),
	}
}

// CreateEntity 创建新实体并返回唯一ID
func (em *EntityManager) CreateEntity() EntityID {
	id := EntityID(em.nextID)
	em.nextID++
	em.components[id] = make(map[reflect.Type]interface{})
	return id
}

// DestroyEntity 标记实体待删除(不立即删除)
func (em *EntityManager) DestroyEntity(id EntityID) {
	em.entitiesToDestroy = append(em.entitiesToDestroy, id)
}

// AddComponent 为实体添加组件
//
// Deprecated: 推荐使用泛型版本 ecs.AddComponent[T](em, entity, component)
// 新代码应使用泛型 API 以获得类型安全和更好的IDE支持。
// 此方法将在 Epic 9 完成后（Story 9.3+）考虑移除。
func (em *EntityManager) AddComponent(id EntityID, component interface{}) {
	componentType := reflect.TypeOf(component)
	if compMap, exists := em.components[id]; exists {
		compMap[componentType] = component
	}
}

// RemoveComponent 从实体移除指定类型的组件
func (em *EntityManager) RemoveComponent(id EntityID, componentType reflect.Type) {
	if compMap, exists := em.components[id]; exists {
		delete(compMap, componentType)
	}
}

// GetComponent 获取实体的特定类型组件
//
// Deprecated: 推荐使用泛型版本 ecs.GetComponent[T](em, entity)
// 泛型版本提供编译时类型检查，无需手动类型断言。
// 此方法将在 Epic 9 完成后（Story 9.3+）考虑移除。
func (em *EntityManager) GetComponent(id EntityID, componentType reflect.Type) (interface{}, bool) {
	if compMap, exists := em.components[id]; exists {
		if comp, found := compMap[componentType]; found {
			return comp, true
		}
	}
	return nil, false
}

// HasComponent 检查实体是否拥有特定类型组件
//
// Deprecated: 推荐使用泛型版本 ecs.HasComponent[T](em, entity)
// 泛型版本无需创建临时类型对象，代码更简洁。
// 此方法将在 Epic 9 完成后（Story 9.3+）考虑移除。
func (em *EntityManager) HasComponent(id EntityID, componentType reflect.Type) bool {
	if compMap, exists := em.components[id]; exists {
		_, found := compMap[componentType]
		return found
	}
	return false
}

// RemoveMarkedEntities 清理所有标记删除的实体
func (em *EntityManager) RemoveMarkedEntities() {
	for _, id := range em.entitiesToDestroy {
		delete(em.components, id)
	}
	em.entitiesToDestroy = em.entitiesToDestroy[:0] // 清空切片
}

// GetEntitiesWith 查询拥有指定组件类型组合的所有实体
// 参数: componentTypes ...reflect.Type - 需要的组件类型列表
// 返回: []EntityID - 满足条件的实体ID列表
//
// Deprecated: 推荐使用泛型版本 ecs.GetEntitiesWith1/2/3/4/5[T1, T2, ...](em)
// 泛型版本提供编译时类型检查，代码更简洁易读。
// 示例:
//   - 1个组件: ecs.GetEntitiesWith1[*PlantComponent](em)
//   - 3个组件: ecs.GetEntitiesWith3[*Comp1, *Comp2, *Comp3](em)
// 此方法将在 Epic 9 完成后（Story 9.3+）考虑移除。
func (em *EntityManager) GetEntitiesWith(componentTypes ...reflect.Type) []EntityID {
	result := make([]EntityID, 0)

	for id, compMap := range em.components {
		hasAll := true
		for _, ct := range componentTypes {
			if _, found := compMap[ct]; !found {
				hasAll = false
				break
			}
		}
		if hasAll {
			result = append(result, id)
		}
	}

	return result
}

// ========== 泛型 API（Story 9.1） ==========

// GetComponent 获取实体的特定类型组件（泛型版本）
// 类型参数 T 必须与存储时的组件类型完全一致（包括指针标记）
// 返回: (component T, exists bool) - 组件实例和存在性标志
// 示例: plantComp, ok := ecs.GetComponent[*components.PlantComponent](em, entity)
//
// 性能说明：虽然泛型提供了类型安全，但由于底层存储仍使用 reflect.Type 作为 key，
// 因此性能提升有限（约 5-10%）。主要优势在于：
// 1. 编译时类型检查
// 2. 消除手动类型断言
// 3. 代码更简洁易读
func GetComponent[T any](em *EntityManager, entity EntityID) (T, bool) {
	var zero T
	// 注意：这里仍需使用反射获取类型，因为底层存储使用 reflect.Type 作为 key
	// 未来优化方向：改用类型 ID 或代码生成避免反射
	componentType := reflect.TypeOf((*T)(nil)).Elem()

	if compMap, exists := em.components[entity]; exists {
		if comp, found := compMap[componentType]; found {
			// 直接类型断言，Go 编译器会优化掉这个检查
			return comp.(T), true
		}
	}
	return zero, false
}

// AddComponent 为实体添加组件（泛型版本）
// 类型参数 T 自动从参数推导，无需显式指定
// 示例: ecs.AddComponent(em, entity, &components.PlantComponent{Health: 300})
func AddComponent[T any](em *EntityManager, entity EntityID, component T) {
	componentType := reflect.TypeOf(component)
	if compMap, exists := em.components[entity]; exists {
		compMap[componentType] = component
	}
}

// HasComponent 检查实体是否拥有特定类型组件（泛型版本）
// 类型参数 T 指定要检查的组件类型
// 示例: if ecs.HasComponent[*components.PlantComponent](em, entity) { ... }
func HasComponent[T any](em *EntityManager, entity EntityID) bool {
	componentType := reflect.TypeOf((*T)(nil)).Elem()

	if compMap, exists := em.components[entity]; exists {
		_, found := compMap[componentType]
		return found
	}
	return false
}

// getEntitiesWithTypes 内部辅助函数：根据类型列表查询实体
// 被 GetEntitiesWith1~5 复用
func getEntitiesWithTypes(em *EntityManager, componentTypes []reflect.Type) []EntityID {
	result := make([]EntityID, 0)

	for id, compMap := range em.components {
		hasAll := true
		for _, ct := range componentTypes {
			if _, found := compMap[ct]; !found {
				hasAll = false
				break
			}
		}
		if hasAll {
			result = append(result, id)
		}
	}

	return result
}

// GetEntitiesWith1 查询拥有 1 个组件的所有实体（泛型版本）
// 示例: entities := ecs.GetEntitiesWith1[*components.PlantComponent](em)
func GetEntitiesWith1[T1 any](em *EntityManager) []EntityID {
	types := []reflect.Type{
		reflect.TypeOf((*T1)(nil)).Elem(),
	}
	return getEntitiesWithTypes(em, types)
}

// GetEntitiesWith2 查询拥有 2 个组件的所有实体（泛型版本）
// 示例: entities := ecs.GetEntitiesWith2[*components.PlantComponent, *components.PositionComponent](em)
func GetEntitiesWith2[T1, T2 any](em *EntityManager) []EntityID {
	types := []reflect.Type{
		reflect.TypeOf((*T1)(nil)).Elem(),
		reflect.TypeOf((*T2)(nil)).Elem(),
	}
	return getEntitiesWithTypes(em, types)
}

// GetEntitiesWith3 查询拥有 3 个组件的所有实体（泛型版本）
// 示例: entities := ecs.GetEntitiesWith3[*Comp1, *Comp2, *Comp3](em)
func GetEntitiesWith3[T1, T2, T3 any](em *EntityManager) []EntityID {
	types := []reflect.Type{
		reflect.TypeOf((*T1)(nil)).Elem(),
		reflect.TypeOf((*T2)(nil)).Elem(),
		reflect.TypeOf((*T3)(nil)).Elem(),
	}
	return getEntitiesWithTypes(em, types)
}

// GetEntitiesWith4 查询拥有 4 个组件的所有实体（泛型版本）
func GetEntitiesWith4[T1, T2, T3, T4 any](em *EntityManager) []EntityID {
	types := []reflect.Type{
		reflect.TypeOf((*T1)(nil)).Elem(),
		reflect.TypeOf((*T2)(nil)).Elem(),
		reflect.TypeOf((*T3)(nil)).Elem(),
		reflect.TypeOf((*T4)(nil)).Elem(),
	}
	return getEntitiesWithTypes(em, types)
}

// GetEntitiesWith5 查询拥有 5 个组件的所有实体（泛型版本）
func GetEntitiesWith5[T1, T2, T3, T4, T5 any](em *EntityManager) []EntityID {
	types := []reflect.Type{
		reflect.TypeOf((*T1)(nil)).Elem(),
		reflect.TypeOf((*T2)(nil)).Elem(),
		reflect.TypeOf((*T3)(nil)).Elem(),
		reflect.TypeOf((*T4)(nil)).Elem(),
		reflect.TypeOf((*T5)(nil)).Elem(),
	}
	return getEntitiesWithTypes(em, types)
}
