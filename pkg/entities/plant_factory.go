package entities

import (
	"fmt"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/utils"
)

// NewPlantEntity 创建植物实体
// 根据植物类型和网格位置创建一个完整的植物实体，包含位置、图像和植物组件
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载植物图像）
//   - plantType: 植物类型（向日葵、豌豆射手等）
//   - col: 网格列索引 (0-8)
//   - row: 网格行索引 (0-4)
//
// 返回:
//   - ecs.EntityID: 创建的植物实体ID，如果失败返回 0
//   - error: 如果创建失败返回错误信息
func NewPlantEntity(em *ecs.EntityManager, rm *game.ResourceManager, plantType components.PlantType, col, row int) (ecs.EntityID, error) {
	// 计算格子中心坐标
	centerX, centerY := utils.GridToScreenCoords(col, row)

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

	// TODO: 未来可以添加 AnimationComponent 支持植物动画

	return entityID, nil
}
