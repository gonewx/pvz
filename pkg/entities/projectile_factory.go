package entities

import (
	"fmt"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
)

// NewPeaProjectile 创建豌豆子弹实体
// 豌豆子弹从豌豆射手口部发射，以恒定速度向右移动
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载子弹图像）
//   - startX: 子弹起始世界坐标X位置
//   - startY: 子弹起始世界坐标Y位置
//
// 返回:
//   - ecs.EntityID: 创建的子弹实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewPeaProjectile(em *ecs.EntityManager, rm ResourceLoader, startX, startY float64) (ecs.EntityID, error) {
	if em == nil {
		return 0, fmt.Errorf("entity manager cannot be nil")
	}
	if rm == nil {
		return 0, fmt.Errorf("resource manager cannot be nil")
	}

	// 加载豌豆子弹图像
	peaImage, err := rm.LoadImage("assets/images/Effect/PeaBullet.png")
	if err != nil {
		return 0, fmt.Errorf("failed to load pea projectile image: %w", err)
	}

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: startX,
		Y: startY,
	})

	// 添加精灵组件
	em.AddComponent(entityID, &components.SpriteComponent{
		Image: peaImage,
	})

	// 添加速度组件（向右移动）
	em.AddComponent(entityID, &components.VelocityComponent{
		VX: config.PeaBulletSpeed,
		VY: 0,
	})

	// 添加行为组件（标识为豌豆子弹）
	em.AddComponent(entityID, &components.BehaviorComponent{
		Type: components.BehaviorPeaProjectile,
	})

	// 添加碰撞组件（为 Story 4.3 准备）
	em.AddComponent(entityID, &components.CollisionComponent{
		Width:  config.PeaBulletWidth,
		Height: config.PeaBulletHeight,
	})

	return entityID, nil
}
