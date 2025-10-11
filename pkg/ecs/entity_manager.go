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
func (em *EntityManager) GetComponent(id EntityID, componentType reflect.Type) (interface{}, bool) {
	if compMap, exists := em.components[id]; exists {
		if comp, found := compMap[componentType]; found {
			return comp, true
		}
	}
	return nil, false
}

// HasComponent 检查实体是否拥有特定类型组件
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




