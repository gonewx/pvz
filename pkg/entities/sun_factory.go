package entities

import (
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// NewSunEntity 创建一个阳光实体
// 参数:
//   - manager: EntityManager 实例
//   - rm: ResourceManager 实例,用于加载阳光图片
//   - startX: 起始X坐标(屏幕顶部)
//   - targetY: 目标落地Y坐标
//
// 返回: 创建的实体ID
func NewSunEntity(manager *ecs.EntityManager, rm *game.ResourceManager, startX, targetY float64) ecs.EntityID {
	// 创建实体
	id := manager.CreateEntity()

	// 加载阳光图片资源 (使用 reanim 目录的阳光图片)
	sunImage, err := rm.LoadImage("assets/reanim/sun1.png")
	if err != nil {
		// 如果加载失败,尝试使用GIF
		sunImage, _ = rm.LoadImage("assets/images/interface/Sun.gif")
	}

	// 添加位置组件 (屏幕顶部外)
	manager.AddComponent(id, &components.PositionComponent{
		X: startX,
		Y: -50, // 屏幕顶部外
	})

	// Story 6.3: 游戏世界实体统一使用 ReanimComponent 渲染
	// 为单图片实体创建简化的 Reanim 包装（无动画轨道）
	// 注意：UI 元素（植物卡片）仍使用 SpriteComponent，由专门的渲染系统处理
	manager.AddComponent(id, createSimpleReanimComponent(sunImage, "sun"))

	// 添加速度组件 (原版掉落速度: 60像素/秒)
	manager.AddComponent(id, &components.VelocityComponent{
		VX: 0,
		VY: 60,
	})

	// 添加生命周期组件 (掉落2秒+停留13秒 = 15秒总生命周期)
	manager.AddComponent(id, &components.LifetimeComponent{
		MaxLifetime:     15.0,
		CurrentLifetime: 0,
		IsExpired:       false,
	})

	// 添加阳光组件
	manager.AddComponent(id, &components.SunComponent{
		State:   components.SunFalling,
		TargetY: targetY,
	})

	// 添加可点击组件 (阳光图片尺寸约80x80像素)
	manager.AddComponent(id, &components.ClickableComponent{
		Width:     80,
		Height:    80,
		IsEnabled: true,
	})

	return id
}
