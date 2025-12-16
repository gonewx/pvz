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
//   - laneIndex: 子弹所在行号（0-based，用于同行碰撞检测）
//
// 返回:
//   - ecs.EntityID: 创建的子弹实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewPeaProjectile(em *ecs.EntityManager, rm ResourceLoader, startX, startY float64, laneIndex int) (ecs.EntityID, error) {
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
	// Story 8.9 修复：设置 LaneIndex 用于同行碰撞检测
	em.AddComponent(entityID, &components.CollisionComponent{
		Width:     config.PeaBulletWidth,
		Height:    config.PeaBulletHeight,
		LaneIndex: laneIndex,
	})

	// 添加阴影组件
	shadowSize := config.GetShadowSize("pea_projectile")
	em.AddComponent(entityID, &components.ShadowComponent{
		Width:   shadowSize.Width,
		Height:  shadowSize.Height,
		Alpha:   config.DefaultShadowAlpha,
		OffsetY: config.ProjectileShadowOffsetY,
	})

	return entityID, nil
}

// NewSnowPeaProjectile 创建冰豌豆子弹实体
// Story 8.9: 冰豌豆从寒冰射手口部发射，以恒定速度向右移动
// 命中僵尸时造成伤害并附加减速效果
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载子弹图像）
//   - startX: 子弹起始世界坐标X位置
//   - startY: 子弹起始世界坐标Y位置
//   - laneIndex: 子弹所在行号（0-based，用于同行碰撞检测）
//
// 返回:
//   - ecs.EntityID: 创建的子弹实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewSnowPeaProjectile(em *ecs.EntityManager, rm ResourceLoader, startX, startY float64, laneIndex int) (ecs.EntityID, error) {
	if em == nil {
		return 0, fmt.Errorf("entity manager cannot be nil")
	}
	if rm == nil {
		return 0, fmt.Errorf("resource manager cannot be nil")
	}

	// 加载冰豌豆子弹图像
	snowPeaImage, err := rm.LoadImage("assets/images/ProjectileSnowPea.png")
	if err != nil {
		return 0, fmt.Errorf("failed to load snow pea projectile image: %w", err)
	}

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: startX,
		Y: startY,
	})

	// 为冰豌豆创建简化的 Reanim 包装（无动画轨道）
	reanimComp := createSimpleReanimComponent(snowPeaImage, "snow_pea")
	em.AddComponent(entityID, reanimComp)

	log.Printf("[ProjectileFactory] 创建冰豌豆 %d: ReanimName=%s", entityID, reanimComp.ReanimName)

	// 添加速度组件（向右移动，与普通豌豆速度相同）
	em.AddComponent(entityID, &components.VelocityComponent{
		VX: config.PeaBulletSpeed,
		VY: 0,
	})

	// 添加行为组件（标识为冰豌豆子弹）
	em.AddComponent(entityID, &components.BehaviorComponent{
		Type: components.BehaviorSnowPeaProjectile,
	})

	// 添加碰撞组件（与普通豌豆相同尺寸）
	// Story 8.9 修复：设置 LaneIndex 用于同行碰撞检测
	em.AddComponent(entityID, &components.CollisionComponent{
		Width:     config.PeaBulletWidth,
		Height:    config.PeaBulletHeight,
		LaneIndex: laneIndex,
	})

	// 添加阴影组件
	shadowSize := config.GetShadowSize("snowpea_projectile")
	em.AddComponent(entityID, &components.ShadowComponent{
		Width:   shadowSize.Width,
		Height:  shadowSize.Height,
		Alpha:   config.DefaultShadowAlpha,
		OffsetY: config.ProjectileShadowOffsetY,
	})

	return entityID, nil
}
