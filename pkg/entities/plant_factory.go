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
	centerX, centerY := utils.GridToScreenCoords(
		col, row,
		gs.CameraX,
		config.GridWorldStartX, config.GridWorldStartY,
		config.CellWidth, config.CellHeight,
	)

	// 获取植物图像路径
	imagePath := utils.GetPlantImagePath(plantType)

	// 加载植物图像
	plantImage, err := rm.LoadImage(imagePath)
	if err != nil {
		return 0, fmt.Errorf("failed to load plant image %s: %w", imagePath, err)
	}

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件
	em.AddComponent(entityID, &components.PositionComponent{
		X: centerX,
		Y: centerY,
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

	return entityID, nil
}
