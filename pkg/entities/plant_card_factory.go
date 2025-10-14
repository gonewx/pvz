package entities

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
)

// NewPlantCardEntity 创建一个植物卡片实体
// Story 6.3: 使用 Reanim 离屏渲染生成植物预览图标
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器
//   - rs: Reanim 系统（用于渲染植物预览）
//   - plantType: 植物类型
//   - x, y: 卡片在屏幕上的位置
//   - cardScale: 卡片缩放因子（用于计算正确的点击区域）
//
// 返回: 创建的实体ID和可能的错误
func NewPlantCardEntity(em *ecs.EntityManager, rm *game.ResourceManager, rs ReanimSystemInterface, plantType components.PlantType, x, y, cardScale float64) (ecs.EntityID, error) {
	entity := em.CreateEntity()

	// 根据植物类型设置属性
	var sunCost int
	var reanimName string
	var cooldownTime float64

	switch plantType {
	case components.PlantSunflower:
		sunCost = 50
		reanimName = "SunFlower"
		cooldownTime = 7.5
	case components.PlantPeashooter:
		sunCost = 100
		reanimName = "PeaShooter"
		cooldownTime = 7.5
	case components.PlantWallnut:
		sunCost = 50
		reanimName = "Wallnut"
		cooldownTime = 30.0
	default:
		em.DestroyEntity(entity)
		em.RemoveMarkedEntities()
		return 0, fmt.Errorf("unknown plant type: %v", plantType)
	}

	// 加载卡片背景框（所有卡片共享）
	backgroundImg, err := rm.LoadImageByID("IMAGE_REANIM_SEEDPACKET_LARGER")
	if err != nil {
		em.DestroyEntity(entity)
		em.RemoveMarkedEntities()
		log.Printf("[PlantCardFactory] Failed to load card background: %v", err)
		return 0, fmt.Errorf("failed to load card background: %w", err)
	}

	// 渲染植物预览图标（Reanim 离屏渲染）
	plantIcon, err := renderPlantIcon(em, rm, rs, reanimName)
	if err != nil {
		em.DestroyEntity(entity)
		em.RemoveMarkedEntities()
		log.Printf("[PlantCardFactory] Failed to render plant icon for %s: %v", reanimName, err)
		return 0, fmt.Errorf("failed to render plant icon: %w", err)
	}

	// 获取卡片背景的实际尺寸（用于点击区域）
	bounds := backgroundImg.Bounds()
	cardWidth := float64(bounds.Dx())
	cardHeight := float64(bounds.Dy())

	// 添加 PositionComponent (卡片在选择栏的位置)
	em.AddComponent(entity, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// 添加 SpriteComponent (保留兼容性，设为 nil)
	// 实际渲染由 PlantCardRenderSystem 使用 PlantCardComponent 的多层资源
	em.AddComponent(entity, &components.SpriteComponent{
		Image: nil,
	})

	// 添加 PlantCardComponent (卡片数据 + 渲染资源)
	em.AddComponent(entity, &components.PlantCardComponent{
		PlantType:        plantType,
		SunCost:          sunCost,
		CooldownTime:     cooldownTime,
		CurrentCooldown:  0.0,
		IsAvailable:      true,
		BackgroundImage:  backgroundImg,
		PlantIconTexture: plantIcon,
	})

	// 添加 UIComponent (标记为UI元素)
	em.AddComponent(entity, &components.UIComponent{
		State: components.UINormal,
	})

	// 添加 ClickableComponent (可点击)
	// 使用缩放后的卡片尺寸作为点击区域
	em.AddComponent(entity, &components.ClickableComponent{
		Width:     cardWidth * cardScale,
		Height:    cardHeight * cardScale,
		IsEnabled: true,
	})

	log.Printf("[PlantCardFactory] Created plant card (Type: %v, Cost: %d, Icon: %dx%d)",
		plantType, sunCost, plantIcon.Bounds().Dx(), plantIcon.Bounds().Dy())

	return entity, nil
}

// renderPlantIcon 使用 Reanim 系统离屏渲染植物预览图标
//
// 实现步骤：
// 1. 创建临时 Reanim 实体
// 2. 加载植物的 Reanim 资源
// 3. 播放 idle 动画
// 4. 离屏渲染到小纹理 (50x70)
// 5. 清理临时实体
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器
//   - rs: Reanim 系统
//   - reanimName: 植物的 Reanim 资源名称 (如 "SunFlower", "PeaShooter")
//
// 返回: 渲染好的植物图标纹理
func renderPlantIcon(em *ecs.EntityManager, rm *game.ResourceManager, rs ReanimSystemInterface, reanimName string) (*ebiten.Image, error) {
	// 1. 创建临时实体
	tempEntity := em.CreateEntity()
	defer func() {
		em.DestroyEntity(tempEntity)
		em.RemoveMarkedEntities()
	}()

	// 2. 加载 Reanim 资源
	reanimXML := rm.GetReanimXML(reanimName)
	partImages := rm.GetReanimPartImages(reanimName)

	if reanimXML == nil || partImages == nil {
		return nil, fmt.Errorf("failed to load Reanim resources for %s", reanimName)
	}

	// 3. 创建离屏渲染目标纹理
	// 使用更大的纹理尺寸以避免植物边缘被裁剪
	// 原因：RenderSystem 会应用 CenterOffset，如果纹理太小，部分内容会超出边界
	iconWidth := 80
	iconHeight := 90

	// 4. 添加必要的组件
	// 将植物位置设置为纹理中心，确保有足够边距容纳 CenterOffset
	em.AddComponent(tempEntity, &components.PositionComponent{
		X: float64(iconWidth) / 2,  // 纹理中心 X (40)
		Y: float64(iconHeight) / 2, // 纹理中心 Y (45)
	})

	em.AddComponent(tempEntity, &components.ReanimComponent{
		Reanim:     reanimXML,
		PartImages: partImages,
	})

	// 5. 播放 idle 动画（取第一帧作为预览）
	animName := "anim_idle"
	if reanimName == "PeaShooter" {
		animName = "anim_full_idle" // 豌豆射手使用完整待机动画
	}

	if err := rs.PlayAnimation(tempEntity, animName); err != nil {
		log.Printf("[PlantCardFactory] Warning: Failed to play animation %s for %s: %v", animName, reanimName, err)
		// 继续执行，使用默认姿态
	}

	// 6. 创建渲染目标纹理
	iconTexture := ebiten.NewImage(iconWidth, iconHeight)

	// 7. 渲染 Reanim 到纹理
	// 注意：需要临时调用 ReanimSystem 的渲染逻辑
	// 这里使用一个辅助方法来渲染单个实体
	if err := rs.RenderToTexture(tempEntity, iconTexture); err != nil {
		return nil, fmt.Errorf("failed to render plant to texture: %w", err)
	}

	log.Printf("[PlantCardFactory] Rendered plant icon: %s (%dx%d)", reanimName, iconWidth, iconHeight)

	return iconTexture, nil
}
