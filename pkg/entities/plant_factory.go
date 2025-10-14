package entities

import (
	"fmt"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/utils"
)

// ReanimSystemInterface 定义 ReanimSystem 的接口，用于工厂函数依赖注入
// 这样可以避免循环依赖，同时方便测试
type ReanimSystemInterface interface {
	PlayAnimation(entityID ecs.EntityID, animName string) error
	PlayAnimationNoLoop(entityID ecs.EntityID, animName string) error
}

// NewPlantEntity 创建植物实体
// 根据植物类型和网格位置创建一个完整的植物实体，包含位置、图像和植物组件
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载植物图像和 Reanim 资源）
//   - gs: 游戏状态（用于获取摄像机位置）
//   - rs: Reanim 系统（用于初始化动画）
//   - plantType: 植物类型（向日葵、豌豆射手等）
//   - col: 网格列索引 (0-8)
//   - row: 网格行索引 (0-4)
//
// 返回:
//   - ecs.EntityID: 创建的植物实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewPlantEntity(em *ecs.EntityManager, rm ResourceLoader, gs *game.GameState, rs ReanimSystemInterface, plantType components.PlantType, col, row int) (ecs.EntityID, error) {
	// 计算植物原点坐标（使用世界坐标系统）
	// Reanim 坐标系统：部件坐标从原点开始绘制
	// 中心偏移会由 ReanimSystem 自动计算并在渲染时应用
	worldCenterX := config.GridWorldStartX + float64(col)*config.CellWidth + config.CellWidth/2
	worldCenterY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2

	// 获取植物图像路径
	imagePath := utils.GetPlantImagePath(plantType)

	// 加载植物图像
	plantImage, err := rm.LoadImage(imagePath)
	if err != nil {
		return 0, fmt.Errorf("failed to load plant image %s: %w", imagePath, err)
	}

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（使用世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: worldCenterX,
		Y: worldCenterY,
	})

	// 添加精灵组件
	em.AddComponent(entityID, &components.SpriteComponent{
		Image: plantImage,
	})

	// 添加植物组件
	em.AddComponent(entityID, &components.PlantComponent{
		PlantType: plantType,
		GridRow:   row,
		GridCol:   col,
	})

	// 为向日葵添加特定组件
	if plantType == components.PlantSunflower {
		// 添加生命值组件
		em.AddComponent(entityID, &components.HealthComponent{
			CurrentHealth: config.SunflowerDefaultHealth,
			MaxHealth:     config.SunflowerDefaultHealth,
		})

		// 添加行为组件
		em.AddComponent(entityID, &components.BehaviorComponent{
			Type: components.BehaviorSunflower,
		})

		// 添加计时器组件（首次生产周期为 7 秒）
		em.AddComponent(entityID, &components.TimerComponent{
			Name:        "sun_production",
			TargetTime:  7.0,
			CurrentTime: 0,
			IsReady:     false,
		})

		// Story 6.3: 使用 ReanimComponent 替代 AnimationComponent
		// 从 ResourceManager 获取向日葵的 Reanim 数据和部件图片
		reanimXML := rm.GetReanimXML("SunFlower")
		partImages := rm.GetReanimPartImages("SunFlower")

		if reanimXML == nil || partImages == nil {
			return 0, fmt.Errorf("failed to load SunFlower Reanim resources")
		}

		// 添加 ReanimComponent
		em.AddComponent(entityID, &components.ReanimComponent{
			Reanim:     reanimXML,
			PartImages: partImages,
		})

		// 使用 ReanimSystem 初始化动画（播放待机动画）
		if err := rs.PlayAnimation(entityID, "anim_idle"); err != nil {
			return 0, fmt.Errorf("failed to play SunFlower idle animation: %w", err)
		}
	}

	// 为豌豆射手添加特定组件
	if plantType == components.PlantPeashooter {
		// 添加生命值组件
		em.AddComponent(entityID, &components.HealthComponent{
			CurrentHealth: config.PeashooterDefaultHealth,
			MaxHealth:     config.PeashooterDefaultHealth,
		})

		// 添加行为组件
		em.AddComponent(entityID, &components.BehaviorComponent{
			Type: components.BehaviorPeashooter,
		})

		// 添加攻击冷却计时器
		em.AddComponent(entityID, &components.TimerComponent{
			Name:        "attack_cooldown",
			TargetTime:  1.4, // 攻击间隔 1.4 秒
			CurrentTime: 0,
			IsReady:     false,
		})

		// Story 6.3: 使用 ReanimComponent 替代 AnimationComponent
		// 从 ResourceManager 获取豌豆射手的 Reanim 数据和部件图片
		reanimXML := rm.GetReanimXML("PeaShooter")
		partImages := rm.GetReanimPartImages("PeaShooter")

		if reanimXML == nil || partImages == nil {
			return 0, fmt.Errorf("failed to load PeaShooter Reanim resources")
		}

		// 添加 ReanimComponent
		em.AddComponent(entityID, &components.ReanimComponent{
			Reanim:     reanimXML,
			PartImages: partImages,
		})

		// 使用 ReanimSystem 初始化动画（播放完整待机动画，包括头部）
		// 注意：豌豆射手的 anim_idle 只显示茎和叶子，anim_full_idle 才显示完整植物
		if err := rs.PlayAnimation(entityID, "anim_full_idle"); err != nil {
			return 0, fmt.Errorf("failed to play PeaShooter idle animation: %w", err)
		}
	}

	return entityID, nil
}

// NewWallnutEntity 创建坚果墙实体
// 坚果墙是一种高生命值的防御植物，没有攻击能力，根据生命值百分比显示不同的外观状态
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载坚果墙图像和 Reanim 资源）
//   - gs: 游戏状态（用于获取摄像机位置）
//   - rs: Reanim 系统（用于初始化动画）
//   - col: 网格列索引 (0-8)
//   - row: 网格行索引 (0-4)
//
// 返回:
//   - ecs.EntityID: 创建的坚果墙实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewWallnutEntity(em *ecs.EntityManager, rm ResourceLoader, gs *game.GameState, rs ReanimSystemInterface, col, row int) (ecs.EntityID, error) {
	// 计算格子中心坐标（使用世界坐标系统）
	worldCenterX := config.GridWorldStartX + float64(col)*config.CellWidth + config.CellWidth/2
	worldCenterY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（使用世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: worldCenterX,
		Y: worldCenterY,
	})

	// Story 6.3: 使用 ReanimComponent 替代 AnimationComponent
	// 从 ResourceManager 获取坚果墙的 Reanim 数据和部件图片
	// 注意：ResourceManager 加载时使用 "Wallnut"（与文件名匹配）
	reanimXML := rm.GetReanimXML("Wallnut")
	partImages := rm.GetReanimPartImages("Wallnut")

	if reanimXML == nil || partImages == nil {
		return 0, fmt.Errorf("failed to load Wallnut Reanim resources")
	}

	// 添加精灵组件（使用占位图像作为后备）
	// TODO (TD-6.3-6): 未来可以完全移除 SpriteComponent，仅使用 ReanimComponent
	placeholderImage, err := rm.LoadImage("assets/images/Plants/WallNut/WallNut_1.png")
	if err != nil {
		// 如果无法加载占位图像（如测试环境），使用空图像
		placeholderImage, _ = rm.LoadImage("placeholder") // Mock 会返回测试图像
	}
	em.AddComponent(entityID, &components.SpriteComponent{
		Image: placeholderImage,
	})

	// 添加植物组件（用于碰撞检测和网格位置追踪）
	em.AddComponent(entityID, &components.PlantComponent{
		PlantType: components.PlantWallnut,
		GridRow:   row,
		GridCol:   col,
	})

	// 添加生命值组件（坚果墙拥有极高的生命值）
	em.AddComponent(entityID, &components.HealthComponent{
		CurrentHealth: config.WallnutDefaultHealth, // 4000
		MaxHealth:     config.WallnutDefaultHealth,
	})

	// 添加行为组件（坚果墙行为）
	em.AddComponent(entityID, &components.BehaviorComponent{
		Type: components.BehaviorWallnut,
	})

	// 添加 ReanimComponent
	em.AddComponent(entityID, &components.ReanimComponent{
		Reanim:     reanimXML,
		PartImages: partImages,
	})

	// 使用 ReanimSystem 初始化动画（播放完好状态的待机动画）
	if err := rs.PlayAnimation(entityID, "anim_idle"); err != nil {
		return 0, fmt.Errorf("failed to play WallNut idle animation: %w", err)
	}

	// 添加碰撞组件（用于僵尸碰撞检测）
	// 坚果墙的碰撞盒与普通植物类似
	em.AddComponent(entityID, &components.CollisionComponent{
		Width:  config.CellWidth * 0.8,  // 碰撞盒宽度略小于格子宽度
		Height: config.CellHeight * 0.8, // 碰撞盒高度略小于格子高度
	})

	return entityID, nil
}
