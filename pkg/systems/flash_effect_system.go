package systems

import (
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
)

// FlashEffectSystem 闪烁效果系统（方案A+）
// 管理实体的受击闪烁效果生命周期
type FlashEffectSystem struct {
	entityManager *ecs.EntityManager
}

// NewFlashEffectSystem 创建闪烁效果系统
func NewFlashEffectSystem(em *ecs.EntityManager) *FlashEffectSystem {
	return &FlashEffectSystem{
		entityManager: em,
	}
}

// Update 更新所有闪烁效果
// 参数：
//   - dt: 时间增量（秒）
func (s *FlashEffectSystem) Update(dt float64) {
	// 查询所有拥有闪烁组件的实体
	entities := ecs.GetEntitiesWith1[*components.FlashEffectComponent](s.entityManager)

	for _, entity := range entities {
		flashComp, ok := ecs.GetComponent[*components.FlashEffectComponent](s.entityManager, entity)
		if !ok || !flashComp.IsActive {
			continue
		}

		// 更新已过时间
		flashComp.Elapsed += dt

		// 检查是否超过持续时间
		if flashComp.Elapsed >= flashComp.Duration {
			// 闪烁结束，移除组件
			ecs.RemoveComponent[*components.FlashEffectComponent](s.entityManager, entity)
		}
	}
}
