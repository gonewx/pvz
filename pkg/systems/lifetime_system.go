package systems

import (
	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
)

// LifetimeSystem 管理实体的生命周期
type LifetimeSystem struct {
	entityManager *ecs.EntityManager
}

// NewLifetimeSystem 创建一个新的生命周期系统
func NewLifetimeSystem(em *ecs.EntityManager) *LifetimeSystem {
	return &LifetimeSystem{
		entityManager: em,
	}
}

// Update 更新所有拥有生命周期组件的实体
func (s *LifetimeSystem) Update(deltaTime float64) {
	// 查询所有拥有 LifetimeComponent 的实体
	entities := ecs.GetEntitiesWith1[*components.LifetimeComponent](s.entityManager)

	for _, id := range entities {
		// 获取生命周期组件
		lifetime, ok := ecs.GetComponent[*components.LifetimeComponent](s.entityManager, id)
		if !ok {
			continue
		}

		// 增加当前生命时间
		lifetime.CurrentLifetime += deltaTime

		// 检查是否过期
		if lifetime.CurrentLifetime >= lifetime.MaxLifetime {
			lifetime.IsExpired = true
		}

		// 如果已过期,标记实体待删除
		if lifetime.IsExpired {
			s.entityManager.DestroyEntity(id)
		}
	}
}
