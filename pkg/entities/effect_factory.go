package entities

import (
	"fmt"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// NewPeaBulletHitEffect 创建豌豆子弹击中效果实体
// 击中效果在指定位置显示短暂的水花动画后自动消失
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载击中效果图像）
//   - x: 击中效果的世界坐标X位置（通常为子弹碰撞位置）
//   - y: 击中效果的世界坐标Y位置（通常为子弹碰撞位置）
//
// 返回:
//   - ecs.EntityID: 创建的击中效果实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewPeaBulletHitEffect(em *ecs.EntityManager, rm *game.ResourceManager, x, y float64) (ecs.EntityID, error) {
	if em == nil {
		return 0, fmt.Errorf("entity manager cannot be nil")
	}
	if rm == nil {
		return 0, fmt.Errorf("resource manager cannot be nil")
	}

	// 加载击中效果图像
	hitImage, err := rm.LoadImage("assets/images/Effect/PeaBulletHit.png")
	if err != nil {
		return 0, fmt.Errorf("failed to load hit effect image: %w", err)
	}

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// 添加精灵组件（击中效果图像）
	em.AddComponent(entityID, &components.SpriteComponent{
		Image: hitImage,
	})

	// 添加行为组件（标识为击中效果）
	em.AddComponent(entityID, &components.BehaviorComponent{
		Type: components.BehaviorPeaBulletHit,
	})

	// 添加计时器组件（控制效果显示时长）
	em.AddComponent(entityID, &components.TimerComponent{
		Name:        "hit_effect_duration",
		CurrentTime: 0.0,
		TargetTime:  config.HitEffectDuration, // 0.2秒后删除
		IsReady:     false,
	})

	return entityID, nil
}
