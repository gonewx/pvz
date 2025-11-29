package systems

import (
	"math"

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

// 闪烁配置常量
const (
	// FlashFrequency 闪烁频率（Hz），越高闪烁越快
	FlashFrequency = 15.0
	// FlashMaxIntensity 最大闪烁强度
	FlashMaxIntensity = 0.6
)

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
			continue
		}

		// 动态计算闪烁强度（使用正弦波实现一闪一闪效果）
		// 使用 |sin(t * frequency * 2π)| 实现快速闪烁
		// 频率 FlashFrequency Hz，即每秒闪烁 FlashFrequency 次
		phase := flashComp.Elapsed * FlashFrequency * 2.0 * math.Pi
		// 使用 sin² 确保强度总是正值且更尖锐的闪烁效果
		sinValue := math.Sin(phase)
		flashComp.Intensity = FlashMaxIntensity * sinValue * sinValue
	}
}
