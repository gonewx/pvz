package entities

import (
	"fmt"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
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
func NewPeaBulletHitEffect(em *ecs.EntityManager, rm ResourceLoader, x, y float64) (ecs.EntityID, error) {
	if em == nil {
		return 0, fmt.Errorf("entity manager cannot be nil")
	}
	if rm == nil {
		return 0, fmt.Errorf("resource manager cannot be nil")
	}

	// 加载击中效果图像
	// 注意：使用 firepea_spark.png 作为临时方案，因为原始的 PeaBulletHit.png 已删除
	// TODO: 考虑为击中效果创建专用图片或使用粒子效果
	hitImage, err := rm.LoadImage("assets/reanim/firepea_spark.png")
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

	// Story 6.3: 使用 ReanimComponent 替代 SpriteComponent
	// 为单图片实体创建简单的 Reanim 包装
	em.AddComponent(entityID, createSimpleReanimComponent(hitImage, "hit_effect"))

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

// NewFallingPartEffect 创建掉落部件效果实体
// 掉落部件（手臂或头部）以抛物线轨迹飞出，一段时间后消失
//
// 参数:
//   - em: 实体管理器
//   - partImage: 部件图片（从 ReanimComponent 获取）
//   - x: 掉落起点的世界坐标X位置
//   - y: 掉落起点的世界坐标Y位置
//   - velocityX: 初始水平速度（向左为负，向右为正）
//   - velocityY: 初始垂直速度（向上为负）
//
// 返回:
//   - ecs.EntityID: 创建的掉落效果实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewFallingPartEffect(em *ecs.EntityManager, partImage *ebiten.Image, x, y, velocityX, velocityY float64) (ecs.EntityID, error) {
	if em == nil {
		return 0, fmt.Errorf("entity manager cannot be nil")
	}
	if partImage == nil {
		return 0, fmt.Errorf("part image cannot be nil")
	}

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// Story 6.3: 使用 ReanimComponent 替代 SpriteComponent
	// 为单图片实体创建简单的 Reanim 包装
	em.AddComponent(entityID, createSimpleReanimComponent(partImage, "falling_part"))

	// 添加速度组件（抛物线运动）
	em.AddComponent(entityID, &components.VelocityComponent{
		VX: velocityX,
		VY: velocityY,
	})

	// 添加行为组件（标识为掉落部件效果）
	em.AddComponent(entityID, &components.BehaviorComponent{
		Type: components.BehaviorFallingPart,
	})

	// 添加计时器组件（控制效果显示时长，2秒后删除）
	em.AddComponent(entityID, &components.TimerComponent{
		Name:        "falling_part_duration",
		CurrentTime: 0.0,
		TargetTime:  2.0, // 2秒后删除
		IsReady:     false,
	})

	return entityID, nil
}
