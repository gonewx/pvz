package entities

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// NewSunEntity 创建一个阳光实体
// 参数:
//   - manager: EntityManager 实例
//   - rm: ResourceManager 实例,用于加载阳光动画
//   - startX: 起始X坐标(屏幕顶部)
//   - targetY: 目标落地Y坐标
//
// 返回: 创建的实体ID
//
// 注意：创建后需要调用 ReanimSystem.InitializeDirectRender() 来初始化动画
func NewSunEntity(manager *ecs.EntityManager, rm *game.ResourceManager, startX, targetY float64) ecs.EntityID {
	return newSunEntityInternal(manager, rm, startX, -50, targetY, components.SunFalling)
}

// NewSunEntityStatic 创建一个静态阳光实体（直接出现在目标位置，不下落）
// 用于教学关卡的预生成阳光
func NewSunEntityStatic(manager *ecs.EntityManager, rm *game.ResourceManager, x, y float64) ecs.EntityID {
	return newSunEntityInternal(manager, rm, x, y, y, components.SunLanded)
}

// NewPlantSunEntity 创建向日葵生产的阳光实体（抛物线运动）
// 参数:
//   - manager: EntityManager 实例
//   - rm: ResourceManager 实例
//   - startX, startY: 起始位置（向日葵中心）
//   - targetX, targetY: 目标位置（落地点）
//
// 返回: 创建的实体ID
func NewPlantSunEntity(manager *ecs.EntityManager, rm *game.ResourceManager, startX, startY, targetX, targetY float64) ecs.EntityID {
	return newSunEntityInternal(manager, rm, startX, startY, targetY, components.SunRising)
}

// newSunEntityInternal 内部函数，创建阳光实体
func newSunEntityInternal(manager *ecs.EntityManager, rm *game.ResourceManager, startX, startY, targetY float64, initialState components.SunState) ecs.EntityID {
	// 创建实体
	id := manager.CreateEntity()

	// Story 8.2 QA修复：使用 Reanim 系统加载阳光动画
	// Sun.reanim 包含3张图片的动画效果
	reanimXML := rm.GetReanimXML("Sun")
	reanimPartImages := rm.GetReanimPartImages("Sun")

	log.Printf("[SunFactory] Creating sun entity ID=%d at (%.1f, %.1f)", id, startX, targetY)
	log.Printf("[SunFactory] Reanim XML: %v, Part Images: %v", reanimXML != nil, reanimPartImages != nil)
	if reanimPartImages != nil {
		log.Printf("[SunFactory] Part image count: %d", len(reanimPartImages))
	}

	// 添加位置组件
	manager.AddComponent(id, &components.PositionComponent{
		X: startX,
		Y: startY, // 使用传入的起始Y坐标
	})

	// 添加 ReanimComponent（使用完整的阳光动画）
	if reanimXML != nil && reanimPartImages != nil {
		log.Printf("[SunFactory] Adding ReanimComponent with Reanim data")
		// Story 8.2 QA修复：阳光的 Sun.reanim 只有 track，没有 anim 定义
		// 只添加基础 ReanimComponent，动画初始化由 ReanimSystem.InitializeDirectRender() 完成
		manager.AddComponent(id, &components.ReanimComponent{
			ReanimXML:  reanimXML,
			PartImages: reanimPartImages,
			IsLooping:  true, // 循环播放
		})
	} else {
		// 降级方案：使用单张图片（如果 Reanim 加载失败）
		log.Printf("[SunFactory] WARNING: Reanim not available, using fallback image")
		sunImage, err := rm.LoadImageByID("IMAGE_REANIM_SUN1")
		if err != nil {
			// 如果仍然失败，使用备用图片
			log.Printf("[SunFactory] ERROR: Failed to load IMAGE_REANIM_SUN1: %v", err)
			sunImage, _ = rm.LoadImage("assets/images/SunBank.png")
		}
		manager.AddComponent(id, createSimpleReanimComponent(sunImage, "sun"))
	}

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
		State:   initialState, // 使用传入的初始状态
		TargetY: targetY,
	})

	// 添加可点击组件（使用配置文件中的点击范围）
	// Story 8.2 QA改进：点击范围可在 config/unit_config.go 中调整
	manager.AddComponent(id, &components.ClickableComponent{
		Width:     config.SunClickableWidth,
		Height:    config.SunClickableHeight,
		IsEnabled: true,
	})

	return id
}
