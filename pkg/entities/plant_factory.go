package entities

import (
	"fmt"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// NewPlantEntity 创建植物实体
// 根据植物类型和网格位置创建一个完整的植物实体，包含位置、图像和植物组件
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载植物图像）
//   - gs: 游戏状态（用于获取摄像机位置）
//   - plantType: 植物类型（向日葵、豌豆射手等）
//   - col: 网格列索引 (0-8)
//   - row: 网格行索引 (0-4)
//
// 返回:
//   - ecs.EntityID: 创建的植物实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewPlantEntity(em *ecs.EntityManager, rm *game.ResourceManager, gs *game.GameState, plantType components.PlantType, col, row int) (ecs.EntityID, error) {
	// 计算格子中心坐标（使用世界坐标系统）
	// 注意：PositionComponent 存储世界坐标，不受摄像机影响
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

		// 加载向日葵动画帧（18帧动画）
		frames := make([]*ebiten.Image, 18)
		for i := 0; i < 18; i++ {
			framePath := fmt.Sprintf("assets/images/Plants/SunFlower/SunFlower_%d.png", i+1)
			frameImage, err := rm.LoadImage(framePath)
			if err != nil {
				return 0, fmt.Errorf("failed to load sunflower animation frame %d: %w", i+1, err)
			}
			frames[i] = frameImage
		}

		// 添加动画组件
		// 向日葵一直播放待机动画（循环）
		em.AddComponent(entityID, &components.AnimationComponent{
			Frames:       frames,
			FrameSpeed:   0.08, // 0.08 秒/帧，完整动画约 1.44 秒
			CurrentFrame: 0,
			FrameCounter: 0,
			IsLooping:    true,  // 循环播放待机动画
			IsFinished:   false, // 动画一直播放
		})
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

		// 加载豌豆射手动画帧（13帧动画）
		frames := make([]*ebiten.Image, 13)
		for i := 0; i < 13; i++ {
			framePath := fmt.Sprintf("assets/images/Plants/Peashooter/Peashooter_%d.png", i+1)
			frameImage, err := rm.LoadImage(framePath)
			if err != nil {
				return 0, fmt.Errorf("failed to load peashooter animation frame %d: %w", i+1, err)
			}
			frames[i] = frameImage
		}

		// 添加动画组件
		// 豌豆射手持续循环播放动画（无论是否在攻击）
		em.AddComponent(entityID, &components.AnimationComponent{
			Frames:       frames,
			FrameSpeed:   0.08, // 0.08 秒/帧，完整动画约 1.04 秒
			CurrentFrame: 0,
			FrameCounter: 0,
			IsLooping:    true,  // 循环播放动画
			IsFinished:   false, // 动画一直播放
		})
	}

	return entityID, nil
}

// NewWallnutEntity 创建坚果墙实体
// 坚果墙是一种高生命值的防御植物，没有攻击能力，根据生命值百分比显示不同的外观状态
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载坚果墙图像和动画）
//   - gs: 游戏状态（用于获取摄像机位置）
//   - col: 网格列索引 (0-8)
//   - row: 网格行索引 (0-4)
//
// 返回:
//   - ecs.EntityID: 创建的坚果墙实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewWallnutEntity(em *ecs.EntityManager, rm *game.ResourceManager, gs *game.GameState, col, row int) (ecs.EntityID, error) {
	// 计算格子中心坐标（使用世界坐标系统）
	worldCenterX := config.GridWorldStartX + float64(col)*config.CellWidth + config.CellWidth/2
	worldCenterY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2

	// 加载坚果墙完好状态动画帧（初始状态）
	fullHealthFrames, err := utils.LoadWallnutFullHealthAnimation(rm)
	if err != nil {
		return 0, fmt.Errorf("failed to load wallnut full health animation: %w", err)
	}

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件（使用世界坐标）
	em.AddComponent(entityID, &components.PositionComponent{
		X: worldCenterX,
		Y: worldCenterY,
	})

	// 添加精灵组件（使用完好状态的第一帧）
	em.AddComponent(entityID, &components.SpriteComponent{
		Image: fullHealthFrames[0],
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

	// 添加动画组件（初始为完好状态动画，循环播放）
	em.AddComponent(entityID, &components.AnimationComponent{
		Frames:       fullHealthFrames,
		FrameSpeed:   config.WallnutFrameSpeed, // 0.1 秒/帧
		CurrentFrame: 0,
		FrameCounter: 0,
		IsLooping:    true,  // 循环播放待机动画
		IsFinished:   false, // 动画一直播放
	})

	// 添加碰撞组件（用于僵尸碰撞检测）
	// 坚果墙的碰撞盒与普通植物类似
	em.AddComponent(entityID, &components.CollisionComponent{
		Width:  config.CellWidth * 0.8,  // 碰撞盒宽度略小于格子宽度
		Height: config.CellHeight * 0.8, // 碰撞盒高度略小于格子高度
	})

	return entityID, nil
}
