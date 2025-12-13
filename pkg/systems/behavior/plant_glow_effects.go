package behavior

import (
	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
)

// updateSunflowerGlowEffects 更新所有向日葵脸部发光效果
// 亮起阶段：每帧增加发光强度，直到达到最大值
// 衰减阶段：每帧降低发光强度，直到归零
// 当强度归零时，移除发光组件
func (s *BehaviorSystem) updateSunflowerGlowEffects(deltaTime float64) {
	// 查询所有拥有向日葵发光组件的实体
	entities := ecs.GetEntitiesWith1[*components.SunflowerGlowComponent](s.entityManager)

	for _, entityID := range entities {
		glowComp, ok := ecs.GetComponent[*components.SunflowerGlowComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		if glowComp.IsRising {
			// 亮起阶段：增加强度
			glowComp.Intensity += glowComp.RiseSpeed * deltaTime
			if glowComp.Intensity >= glowComp.MaxIntensity {
				glowComp.Intensity = glowComp.MaxIntensity
				glowComp.IsRising = false // 切换到衰减阶段
			}
		} else {
			// 衰减阶段：降低强度
			glowComp.Intensity -= glowComp.FadeSpeed * deltaTime

			// 如果强度归零，移除组件
			if glowComp.Intensity <= 0 {
				ecs.RemoveComponent[*components.SunflowerGlowComponent](s.entityManager, entityID)
			}
		}
	}
}

// updateWallnutHitGlowEffects 更新所有坚果墙被啃食发光效果
// 每帧降低发光强度，实现一闪一闪的效果
// 当强度归零时，移除发光组件
func (s *BehaviorSystem) updateWallnutHitGlowEffects(deltaTime float64) {
	// 查询所有拥有坚果墙发光组件的实体
	glowEntities := ecs.GetEntitiesWith1[*components.WallnutHitGlowComponent](s.entityManager)

	for _, entityID := range glowEntities {
		glowComp, ok := ecs.GetComponent[*components.WallnutHitGlowComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 降低发光强度
		glowComp.Intensity -= glowComp.FadeSpeed * deltaTime

		// 如果强度归零，移除组件
		if glowComp.Intensity <= 0 {
			ecs.RemoveComponent[*components.WallnutHitGlowComponent](s.entityManager, entityID)
		}
	}
}
