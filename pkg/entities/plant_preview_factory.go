package entities

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/utils"
)

// NewPlantPreviewEntity 创建植物预览实体
// 根据植物类型加载对应的图像，并创建一个半透明的预览实体
// 该实体会跟随鼠标移动，并在草坪网格内自动对齐到格子中心
func NewPlantPreviewEntity(em *ecs.EntityManager, rm *game.ResourceManager, plantType components.PlantType, x, y float64) ecs.EntityID {
	// 获取植物预览图像路径
	imagePath := utils.GetPlantPreviewImagePath(plantType)

	// 加载植物图像
	plantImage, err := rm.LoadImage(imagePath)
	if err != nil {
		log.Printf("[PlantPreviewFactory] Failed to load plant image %s: %v", imagePath, err)
		return 0 // 返回无效ID
	}

	// 创建实体
	entityID := em.CreateEntity()

	// 添加位置组件
	em.AddComponent(entityID, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// 添加精灵组件
	em.AddComponent(entityID, &components.SpriteComponent{
		Image: plantImage,
	})

	// 添加植物预览组件
	em.AddComponent(entityID, &components.PlantPreviewComponent{
		PlantType: plantType,
		Alpha:     0.5, // 半透明效果
	})

	log.Printf("[PlantPreviewFactory] Created plant preview entity (ID: %d, Type: %v) at (%.1f, %.1f)",
		entityID, plantType, x, y)

	return entityID
}
