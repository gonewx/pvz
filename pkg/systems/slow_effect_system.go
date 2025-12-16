package systems

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
)

// SlowEffectSystem 管理减速效果
// 处理冰豌豆等减速效果的应用、持续时间和移除
type SlowEffectSystem struct {
	entityManager *ecs.EntityManager
}

// NewSlowEffectSystem 创建减速效果系统
func NewSlowEffectSystem(em *ecs.EntityManager) *SlowEffectSystem {
	return &SlowEffectSystem{
		entityManager: em,
	}
}

// Update 更新所有拥有减速效果的实体
func (s *SlowEffectSystem) Update(deltaTime float64) {
	// 查询所有拥有 SlowedComponent 的实体
	// 注意：不再要求 VelocityComponent，因为啃食中的僵尸没有 VelocityComponent
	entities := ecs.GetEntitiesWith1[*components.SlowedComponent](s.entityManager)

	for _, id := range entities {
		slowed, ok := ecs.GetComponent[*components.SlowedComponent](s.entityManager, id)
		if !ok {
			continue
		}

		// 如果减速效果还未应用速度减慢
		if !slowed.Applied {
			// 对有 VelocityComponent 的实体应用移动速度减慢
			if velocity, hasVel := ecs.GetComponent[*components.VelocityComponent](s.entityManager, id); hasVel {
				slowed.OriginalVX = velocity.VX
				slowed.OriginalVY = velocity.VY
				velocity.VX *= slowed.SpeedMultiplier
				velocity.VY *= slowed.SpeedMultiplier
				log.Printf("[SlowEffectSystem] 实体 %d: 应用减速效果，速度从 (%.2f, %.2f) 降至 (%.2f, %.2f)，持续 %.1f 秒",
					id, slowed.OriginalVX, slowed.OriginalVY, velocity.VX, velocity.VY, slowed.Duration)
			}
			slowed.Applied = true

			// Story 8.9: 应用动画减速视觉效果
			s.applyAnimationSlowdown(id, slowed.SpeedMultiplier)
		}

		// Story 8.9 补充：确保新切换的动画也被减速
		// 例如：僵尸被减速后开始啃食，啃食动画也需要被减速
		s.ensureAnimationSlowdown(id, slowed.SpeedMultiplier)

		// 更新减速持续时间
		slowed.Duration -= deltaTime

		// 检查减速效果是否结束
		if slowed.Duration <= 0 {
			// 恢复原始速度（如果有 VelocityComponent）
			if velocity, hasVel := ecs.GetComponent[*components.VelocityComponent](s.entityManager, id); hasVel {
				velocity.VX = slowed.OriginalVX
				velocity.VY = slowed.OriginalVY
				log.Printf("[SlowEffectSystem] 实体 %d: 减速效果结束，速度恢复至 (%.2f, %.2f)",
					id, velocity.VX, velocity.VY)
			}

			// Story 8.9: 恢复动画速度
			s.restoreAnimationSpeed(id)

			// 移除减速组件（使用泛型版本）
			ecs.RemoveComponent[*components.SlowedComponent](s.entityManager, id)
		}
	}
}

// applyAnimationSlowdown 应用动画减速视觉效果
// Story 8.9: 被减速的僵尸动画播放速度也会降低
func (s *SlowEffectSystem) applyAnimationSlowdown(entityID ecs.EntityID, speedMultiplier float64) {
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 初始化 AnimationSpeedOverrides（如果为空）
	if reanim.AnimationSpeedOverrides == nil {
		reanim.AnimationSpeedOverrides = make(map[string]float64)
	}

	// 对所有当前播放的动画应用减速
	for _, animName := range reanim.CurrentAnimations {
		reanim.AnimationSpeedOverrides[animName] = speedMultiplier
		log.Printf("[SlowEffectSystem] 实体 %d: 动画 '%s' 速度设为 %.2f", entityID, animName, speedMultiplier)
	}

	// 启用视觉效果标记
	if slowed, ok := ecs.GetComponent[*components.SlowedComponent](s.entityManager, entityID); ok {
		slowed.VisualEffect = true
	}
}

// ensureAnimationSlowdown 确保新切换的动画也被减速
// Story 8.9 补充：当僵尸切换动画（如从行走切换到啃食）时，新动画也需要被减速
func (s *SlowEffectSystem) ensureAnimationSlowdown(entityID ecs.EntityID, speedMultiplier float64) {
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 如果还没有 AnimationSpeedOverrides，初始化它
	if reanim.AnimationSpeedOverrides == nil {
		reanim.AnimationSpeedOverrides = make(map[string]float64)
	}

	// 检查当前播放的动画是否都已被减速
	for _, animName := range reanim.CurrentAnimations {
		if _, exists := reanim.AnimationSpeedOverrides[animName]; !exists {
			// 新动画还没有被减速，应用减速
			reanim.AnimationSpeedOverrides[animName] = speedMultiplier
			log.Printf("[SlowEffectSystem] 实体 %d: 新动画 '%s' 速度设为 %.2f（动画切换后）", entityID, animName, speedMultiplier)
		}
	}
}

// restoreAnimationSpeed 恢复动画正常速度
// Story 8.9: 减速效果结束时恢复动画正常速度
func (s *SlowEffectSystem) restoreAnimationSpeed(entityID ecs.EntityID) {
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 清除所有动画的速度覆盖
	if reanim.AnimationSpeedOverrides != nil {
		for animName := range reanim.AnimationSpeedOverrides {
			delete(reanim.AnimationSpeedOverrides, animName)
			log.Printf("[SlowEffectSystem] 实体 %d: 动画 '%s' 速度已恢复", entityID, animName)
		}
	}
}

// ApplySlowEffect 对指定实体应用减速效果
// 如果实体已有减速效果，刷新持续时间
func ApplySlowEffect(em *ecs.EntityManager, entityID ecs.EntityID, multiplier, duration float64) {
	// 检查实体是否已有减速效果
	if slowed, ok := ecs.GetComponent[*components.SlowedComponent](em, entityID); ok {
		// 刷新持续时间
		slowed.Duration = duration
		log.Printf("[SlowEffectSystem] 实体 %d: 刷新减速效果，持续时间重置为 %.1f 秒", entityID, duration)
		return
	}

	// 添加新的减速效果
	ecs.AddComponent(em, entityID, &components.SlowedComponent{
		SpeedMultiplier: multiplier,
		Duration:        duration,
		VisualEffect:    true,
		Applied:         false,
	})
	log.Printf("[SlowEffectSystem] 实体 %d: 添加减速效果，倍率 %.2f，持续 %.1f 秒", entityID, multiplier, duration)
}
