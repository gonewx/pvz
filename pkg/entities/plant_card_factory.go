package entities

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// NewPlantCardEntity 创建一个植物卡片实体
// 根据植物类型设置相应的属性（消耗、冷却、图像）
// 返回创建的实体ID和可能的错误
func NewPlantCardEntity(em *ecs.EntityManager, rm *game.ResourceManager, plantType components.PlantType, x, y float64) (ecs.EntityID, error) {
	entity := em.CreateEntity()

	// 根据植物类型设置属性
	var sunCost int
	var imagePath string
	var cooldownTime float64

	switch plantType {
	case components.PlantSunflower:
		sunCost = 50
		imagePath = "assets/images/Cards/card_sunFlower.png"
		cooldownTime = 7.5
	case components.PlantPeashooter:
		sunCost = 100
		imagePath = "assets/images/Cards/card_peashooter.png"
		cooldownTime = 7.5
	case components.PlantWallnut:
		sunCost = 50
		imagePath = "assets/images/Cards/card_wallnut.png"
		cooldownTime = 30.0 // 坚果墙冷却时间为 30 秒
	}

	// 加载卡片图像
	cardImage, err := rm.LoadImage(imagePath)
	if err != nil {
		// 删除已创建但无法完成的实体
		em.DestroyEntity(entity)
		em.RemoveMarkedEntities()
		log.Printf("[PlantCardFactory] Failed to load card image for %v: %v", plantType, err)
		return 0, fmt.Errorf("failed to load plant card image %s: %w", imagePath, err)
	}

	// 获取卡片图像的实际尺寸
	bounds := cardImage.Bounds()
	cardWidth := float64(bounds.Dx())
	cardHeight := float64(bounds.Dy())

	// 添加 PositionComponent (卡片在选择栏的位置)
	em.AddComponent(entity, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// 添加 SpriteComponent (正常卡片图像)
	em.AddComponent(entity, &components.SpriteComponent{
		Image: cardImage,
	})

	// 添加 PlantCardComponent (卡片数据)
	em.AddComponent(entity, &components.PlantCardComponent{
		PlantType:       plantType,
		SunCost:         sunCost,
		CooldownTime:    cooldownTime,
		CurrentCooldown: 0.0, // 初始无冷却
		IsAvailable:     true,
	})

	// 添加 UIComponent (标记为UI元素)
	em.AddComponent(entity, &components.UIComponent{
		State: components.UINormal,
	})

	// 添加 ClickableComponent (可点击)
	// 使用卡片图像的实际尺寸
	em.AddComponent(entity, &components.ClickableComponent{
		Width:     cardWidth,
		Height:    cardHeight,
		IsEnabled: true, // 初始状态为可点击
	})

	return entity, nil
}
