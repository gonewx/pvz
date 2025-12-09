package entities

import (
	"fmt"

	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
)

// getAnimVisiblesMapKeys 辅助函数：获取 map 的 keys
func getAnimVisiblesMapKeys(m map[string][]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

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
	peaImage, err := rm.LoadImage("assets/images/ProjectilePea.png")
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

	// Story 6.3: 游戏世界实体统一使用 ReanimComponent 渲染
	// 为单图片实体创建简化的 Reanim 包装（无动画轨道）
	// 注意：UI 元素（植物卡片）仍使用 SpriteComponent，由专门的渲染系统处理
	reanimComp := createSimpleReanimComponent(peaImage, "pea")
	em.AddComponent(entityID, reanimComp)

	// ✅ Debug: 打印子弹创建信息
	log.Printf("[ProjectileFactory] 创建子弹 %d: ReanimName=%s, VisualTracks=%v, CurrentAnimations=%v, AnimVisiblesMap keys=%v",
		entityID, reanimComp.ReanimName, reanimComp.VisualTracks, reanimComp.CurrentAnimations, getAnimVisiblesMapKeys(reanimComp.AnimVisiblesMap))

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
