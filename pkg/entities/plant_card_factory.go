package entities

import (
	"fmt"
	"image/color"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// NewPlantCardEntity 创建一个植物卡片实体
// Story 6.3 + 8.4: 使用 Reanim 离屏渲染生成植物预览图标，所有内部配置封装
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器
//   - rs: Reanim 系统（用于渲染植物预览）
//   - plantType: 植物类型
//   - x, y: 卡片在屏幕上的位置
//   - cardScale: 卡片整体缩放因子（控制卡片大小，如 0.54 为标准大小，1.0 为原始大小）
//
// 返回: 创建的实体ID和可能的错误
//
// 注意：所有植物卡片的内部配置（背景图、图标缩放、偏移等）都在 config.plant_card_config.go 中定义，
// 不暴露给调用者，确保卡片作为统一整体，由工厂函数完全封装。
func NewPlantCardEntity(em *ecs.EntityManager, rm *game.ResourceManager, rs ReanimSystemInterface, plantType components.PlantType, x, y, cardScale float64) (ecs.EntityID, error) {
	entity := em.CreateEntity()

	// 根据植物类型设置属性
	var sunCost int
	var reanimName string
	var cooldownTime float64

	switch plantType {
	case components.PlantSunflower:
		sunCost = config.SunflowerSunCost // 50
		reanimName = "SunFlower"
		cooldownTime = config.SunflowerRechargeTime // 7.5
	case components.PlantPeashooter:
		sunCost = config.PeashooterSunCost // 100
		reanimName = "PeaShooter"
		cooldownTime = config.PeashooterRechargeTime // 7.5
	case components.PlantWallnut:
		sunCost = config.WallnutCost // 50
		reanimName = "Wallnut"
		cooldownTime = config.WallnutRechargeTime // 30.0
	case components.PlantCherryBomb:
		sunCost = config.CherryBombSunCost // 150
		reanimName = "CherryBomb"
		cooldownTime = config.CherryBombCooldown // 50.0
	default:
		em.DestroyEntity(entity)
		em.RemoveMarkedEntities()
		return 0, fmt.Errorf("unknown plant type: %v", plantType)
	}

	// 加载卡片背景框（从配置获取，所有卡片共享）
	backgroundImg, err := rm.LoadImageByID(config.PlantCardBackgroundID)
	if err != nil {
		em.DestroyEntity(entity)
		em.RemoveMarkedEntities()
		log.Printf("[PlantCardFactory] Failed to load card background: %v", err)
		return 0, fmt.Errorf("failed to load card background: %w", err)
	}

	// 渲染植物预览图标（Reanim 离屏渲染）
	plantIcon, err := RenderPlantIcon(em, rm, rs, reanimName)
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
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// 添加 SpriteComponent (保留兼容性，设为 nil)
	// 实际渲染由 PlantCardRenderSystem 使用 PlantCardComponent 的多层资源
	ecs.AddComponent(em, entity, &components.SpriteComponent{
		Image: nil,
	})

	// 添加 PlantCardComponent (卡片数据 + 渲染资源)
	ecs.AddComponent(em, entity, &components.PlantCardComponent{
		PlantType:        plantType,
		SunCost:          sunCost,
		CooldownTime:     cooldownTime,
		CurrentCooldown:  0.0,
		IsAvailable:      true,
		BackgroundImage:  backgroundImg,
		PlantIconTexture: plantIcon,
		CardScale:        cardScale, // Story 8.4: 保存卡片缩放因子
	})

	// 添加 UIComponent (标记为UI元素)
	ecs.AddComponent(em, entity, &components.UIComponent{
		State: components.UINormal,
	})

	// 添加 ClickableComponent (可点击)
	// 使用缩放后的卡片尺寸作为点击区域
	ecs.AddComponent(em, entity, &components.ClickableComponent{
		Width:     cardWidth * cardScale,
		Height:    cardHeight * cardScale,
		IsEnabled: true,
	})

	log.Printf("[PlantCardFactory] Created plant card (Type: %v, Cost: %d, Icon: %dx%d)",
		plantType, sunCost, plantIcon.Bounds().Dx(), plantIcon.Bounds().Dy())

	return entity, nil
}

// RenderPlantIcon 使用 Reanim 系统离屏渲染植物预览图标
// 该方法创建临时实体，渲染为 80x90 静态纹理后销毁实体
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器
//   - rs: Reanim 系统接口
//   - reanimName: 植物的 Reanim 资源名称 (如 "SunFlower", "PeaShooter")
//
// 返回: 渲染好的植物图标纹理和可能的错误
func RenderPlantIcon(em *ecs.EntityManager, rm *game.ResourceManager, rs ReanimSystemInterface, reanimName string) (*ebiten.Image, error) {
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
	ecs.AddComponent(em, tempEntity, &components.PositionComponent{
		X: float64(iconWidth) / 2,  // 纹理中心 X (40)
		Y: float64(iconHeight) / 2, // 纹理中心 Y (45)
	})

	ecs.AddComponent(em, tempEntity, &components.ReanimComponent{
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

// RenderPlantCard 渲染植物卡片到屏幕
// Story 8.4: 将所有卡片渲染逻辑封装在工厂中，确保卡片作为统一整体
//
// 设计原则：
//   - 所有元素（背景、图标、文字、遮罩）统一应用 card.CardScale 进行整体缩放
//   - 配置文件中的所有尺寸和偏移值基于原始卡片尺寸（100x140）定义
//   - 这确保了无论 cardScale 如何变化，卡片始终保持视觉一致性
//
// 参数:
//   - screen: 目标渲染屏幕
//   - card: 卡片组件
//   - x, y: 卡片左上角位置
//   - sunFont: 阳光数字字体（可选）
//   - sunFontSize: 字体大小（基准值，会应用 cardScale）
//
// 渲染层级：
//   1. 背景框（应用卡片缩放）
//   2. 植物图标（应用卡片缩放和配置的相对缩放）
//   3. 阳光数字（应用卡片缩放，包括字体大小）
//   4. 效果遮罩（冷却/禁用，应用卡片缩放）
func RenderPlantCard(screen *ebiten.Image, card *components.PlantCardComponent, x, y float64, sunFont *text.GoTextFaceSource, sunFontSize float64) {
	if card == nil {
		return
	}

	// 层1: 绘制卡片背景框
	renderCardBackground(screen, card, x, y)

	// 层2: 绘制植物图标
	if card.PlantIconTexture != nil {
		renderPlantIcon(screen, card, x, y)
	}

	// 层3: 绘制阳光数字
	if sunFont != nil {
		renderSunCost(screen, card, x, y, sunFont, sunFontSize)
	}

	// 层4: 绘制效果遮罩（冷却/禁用）
	renderEffectMask(screen, card, x, y)
}

// renderCardBackground 绘制卡片背景框（应用卡片缩放）
func renderCardBackground(screen *ebiten.Image, card *components.PlantCardComponent, x, y float64) {
	if card.BackgroundImage == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(card.CardScale, card.CardScale)
	op.GeoM.Translate(x, y)
	screen.DrawImage(card.BackgroundImage, op)
}

// renderPlantIcon 绘制植物图标（使用配置中的缩放和偏移）
func renderPlantIcon(screen *ebiten.Image, card *components.PlantCardComponent, x, y float64) {
	if card.PlantIconTexture == nil || card.BackgroundImage == nil {
		return
	}

	// 从配置读取植物图标的内部配置
	iconScale := config.PlantCardIconScale
	iconOffsetY := config.PlantCardIconOffsetY

	cardWidth := float64(card.BackgroundImage.Bounds().Dx()) * card.CardScale
	// 图标宽度应用整体缩放（iconScale * cardScale）
	iconWidth := float64(card.PlantIconTexture.Bounds().Dx()) * iconScale * card.CardScale

	// 计算居中偏移
	offsetX := (cardWidth - iconWidth) / 2.0

	op := &ebiten.DrawImageOptions{}
	// 应用整体缩放：iconScale 用于微调，cardScale 用于整体缩放
	op.GeoM.Scale(iconScale*card.CardScale, iconScale*card.CardScale)
	op.GeoM.Translate(x+offsetX, y+iconOffsetY*card.CardScale)
	screen.DrawImage(card.PlantIconTexture, op)
}

// renderSunCost 绘制阳光数字（使用配置中的偏移）
func renderSunCost(screen *ebiten.Image, card *components.PlantCardComponent, x, y float64, sunFont *text.GoTextFaceSource, fontSize float64) {
	if card.BackgroundImage == nil {
		return
	}

	// 从配置读取阳光数字的内部配置
	sunOffsetY := config.PlantCardSunCostOffsetY

	cardWidth := float64(card.BackgroundImage.Bounds().Dx()) * card.CardScale
	cardHeight := float64(card.BackgroundImage.Bounds().Dy()) * card.CardScale

	sunText := fmt.Sprintf("%03d", card.SunCost)

	// 字体大小也应用整体缩放
	face := &text.GoTextFace{
		Source: sunFont,
		Size:   fontSize * card.CardScale,
	}

	// 居中绘制
	textX := x + cardWidth/2
	textY := y + cardHeight - sunOffsetY*card.CardScale

	op := &text.DrawOptions{}
	op.GeoM.Translate(textX, textY)
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 255})
	text.Draw(screen, sunText, face, op)
}

// renderEffectMask 绘制效果遮罩（冷却/禁用）
func renderEffectMask(screen *ebiten.Image, card *components.PlantCardComponent, x, y float64) {
	if card.BackgroundImage == nil {
		return
	}

	cardWidth := float64(card.BackgroundImage.Bounds().Dx()) * card.CardScale
	cardHeight := float64(card.BackgroundImage.Bounds().Dy()) * card.CardScale

	// 转换为整数尺寸，确保至少为1像素
	intCardWidth := int(cardWidth)
	intCardHeight := int(cardHeight)

	// 检查尺寸是否有效
	if intCardWidth <= 0 || intCardHeight <= 0 {
		return // 卡片太小，跳过遮罩绘制
	}

	// 绘制冷却遮罩（从下往上填充）
	cooldownProgress := getCooldownProgress(card)
	if cooldownProgress > 0 {
		maskHeight := cardHeight * cooldownProgress
		intMaskHeight := int(maskHeight)

		// 确保遮罩高度至少为1像素
		if intMaskHeight > 0 {
			mask := ebiten.NewImage(intCardWidth, intMaskHeight)
			mask.Fill(color.RGBA{0, 0, 0, 128}) // 半透明黑色

			maskOp := &ebiten.DrawImageOptions{}
			maskOp.GeoM.Translate(x, y+cardHeight-maskHeight)
			screen.DrawImage(mask, maskOp)
		}
	}

	// 绘制禁用遮罩（阳光不足）
	if isCardDisabled(card) {
		disabledMask := ebiten.NewImage(intCardWidth, intCardHeight)
		disabledMask.Fill(color.RGBA{50, 50, 50, 150}) // 灰色遮罩

		disabledOp := &ebiten.DrawImageOptions{}
		disabledOp.GeoM.Translate(x, y)
		screen.DrawImage(disabledMask, disabledOp)
	}
}

// getCooldownProgress 计算冷却进度（0.0-1.0）
func getCooldownProgress(card *components.PlantCardComponent) float64 {
	if !card.IsAvailable && card.CurrentCooldown > 0 && card.CooldownTime > 0 {
		return card.CurrentCooldown / card.CooldownTime
	}
	return 0.0
}

// isCardDisabled 判断卡片是否处于禁用状态（阳光不足）
func isCardDisabled(card *components.PlantCardComponent) bool {
	return !card.IsAvailable && card.CurrentCooldown == 0
}
