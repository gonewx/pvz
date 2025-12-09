package entities

import (
	"fmt"
	"image/color"
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/types"
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
func NewPlantCardEntity(em *ecs.EntityManager, rm *game.ResourceManager, rs ReanimSystemInterface, plantType components.PlantType, x, y, cardScale float64) (ecs.EntityID, error) {
	entity := em.CreateEntity()

	// 从统一配置获取植物信息
	cfg := config.GetPlantConfig(plantType)
	if cfg == nil {
		em.DestroyEntity(entity)
		em.RemoveMarkedEntities()
		return 0, fmt.Errorf("no config found for plant type: %v", plantType)
	}

	// 根据植物类型设置阳光消耗和冷却时间
	var sunCost int
	var cooldownTime float64

	switch plantType {
	case components.PlantSunflower:
		sunCost = config.SunflowerSunCost
		cooldownTime = config.SunflowerRechargeTime
	case components.PlantPeashooter:
		sunCost = config.PeashooterSunCost
		cooldownTime = config.PeashooterRechargeTime
	case components.PlantWallnut:
		sunCost = config.WallnutCost
		cooldownTime = config.WallnutRechargeTime
	case components.PlantCherryBomb:
		sunCost = config.CherryBombSunCost
		cooldownTime = config.CherryBombCooldown
	case components.PlantPotatoMine:
		sunCost = config.PotatoMineSunCost
		cooldownTime = config.PotatoMineRechargeTime
	default:
		em.DestroyEntity(entity)
		em.RemoveMarkedEntities()
		return 0, fmt.Errorf("unknown plant type: %v", plantType)
	}

	// 加载卡片背景框
	backgroundImg, err := rm.LoadImageByID(config.PlantCardBackgroundID)
	if err != nil {
		em.DestroyEntity(entity)
		em.RemoveMarkedEntities()
		log.Printf("[PlantCardFactory] Failed to load card background: %v", err)
		return 0, fmt.Errorf("failed to load card background: %w", err)
	}

	// 渲染植物预览图标（简化：直接传入 plantType）
	plantIcon, err := RenderPlantIcon(em, rm, rs, plantType)
	if err != nil {
		em.DestroyEntity(entity)
		em.RemoveMarkedEntities()
		log.Printf("[PlantCardFactory] Failed to render plant icon for %s: %v", cfg.ResourceName, err)
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
		Alpha:            1.0,       // Story 8.4: 默认完全不透明
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
//   - plantType: 植物类型（从配置自动获取资源名称等信息）
//
// 返回: 渲染好的植物图标纹理和可能的错误
func RenderPlantIcon(em *ecs.EntityManager, rm *game.ResourceManager, rs ReanimSystemInterface, plantType types.PlantType) (*ebiten.Image, error) {
	// 从配置获取植物资源信息
	cfg := config.GetPlantConfig(plantType)
	if cfg == nil {
		return nil, fmt.Errorf("no config found for plant type %d", plantType)
	}

	// 1. 创建临时实体
	tempEntity := em.CreateEntity()
	defer func() {
		em.DestroyEntity(tempEntity)
		em.RemoveMarkedEntities()
	}()

	// 2. 加载 Reanim 资源（使用资源名称）
	reanimXML := rm.GetReanimXML(cfg.ResourceName)
	partImages := rm.GetReanimPartImages(cfg.ResourceName)

	if reanimXML == nil || partImages == nil {
		log.Printf("[PlantCardFactory] Failed to load Reanim resources for %s", cfg.ResourceName)
		return nil, fmt.Errorf("failed to load Reanim resources for %s", cfg.ResourceName)
	}

	// 3. 创建离屏渲染目标纹理
	iconWidth := 80
	iconHeight := 90

	// 4. 添加必要的组件
	ecs.AddComponent(em, tempEntity, &components.PositionComponent{
		X: float64(iconWidth) / 2,
		Y: float64(iconHeight) / 2,
	})

	ecs.AddComponent(em, tempEntity, &components.ReanimComponent{
		ReanimName: cfg.ResourceName,
		ReanimXML:  reanimXML,
		PartImages: partImages,
	})

	// 5. 准备静态预览（使用植物类型，配置会自动获取）
	if err := rs.PrepareStaticPreview(tempEntity, plantType); err != nil {
		log.Printf("[PlantCardFactory] Warning: Failed to prepare preview for %s: %v", cfg.ResourceName, err)
	}

	// 6. 创建渲染目标纹理
	iconTexture := ebiten.NewImage(iconWidth, iconHeight)

	// 7. 渲染 Reanim 到纹理
	if err := rs.RenderToTexture(tempEntity, iconTexture); err != nil {
		log.Printf("[PlantCardFactory] Failed to render %s to texture: %v", cfg.ResourceName, err)
		return nil, fmt.Errorf("failed to render plant to texture: %w", err)
	}

	log.Printf("[PlantCardFactory] Rendered plant icon: %s (size: %dx%d)", cfg.ResourceName, iconWidth, iconHeight)

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
//  1. 背景框（应用卡片缩放）
//  2. 植物图标（应用卡片缩放和配置的相对缩放）
//  3. 阳光数字（应用卡片缩放，包括字体大小）
//  4. 效果遮罩（冷却/禁用，应用卡片缩放）
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

// renderCardBackground 绘制卡片背景框（应用卡片缩放和透明度）
func renderCardBackground(screen *ebiten.Image, card *components.PlantCardComponent, x, y float64) {
	if card.BackgroundImage == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(card.CardScale, card.CardScale)
	op.GeoM.Translate(x, y)
	// 应用透明度（Story 8.4: 用于淡入淡出动画）
	if card.Alpha < 1.0 {
		op.ColorScale.ScaleAlpha(float32(card.Alpha))
	}
	screen.DrawImage(card.BackgroundImage, op)
}

// renderPlantIcon 绘制植物图标（使用配置中的缩放和偏移，应用透明度）
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
	// 应用透明度（Story 8.4: 用于淡入淡出动画）
	if card.Alpha < 1.0 {
		op.ColorScale.ScaleAlpha(float32(card.Alpha))
	}
	screen.DrawImage(card.PlantIconTexture, op)
}

// renderSunCost 绘制阳光数字（使用配置中的偏移，应用透明度）
func renderSunCost(screen *ebiten.Image, card *components.PlantCardComponent, x, y float64, sunFont *text.GoTextFaceSource, fontSize float64) {
	if card.BackgroundImage == nil {
		return
	}

	// 从配置读取阳光数字的内部配置
	sunOffsetY := config.PlantCardSunCostOffsetY

	cardWidth := float64(card.BackgroundImage.Bounds().Dx()) * card.CardScale
	cardHeight := float64(card.BackgroundImage.Bounds().Dy()) * card.CardScale

	sunText := fmt.Sprintf("%d", card.SunCost)

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
	// 应用透明度（Story 8.4: 用于淡入淡出动画）
	if card.Alpha < 1.0 {
		op.ColorScale.ScaleAlpha(float32(card.Alpha))
	}
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

	isCoolingDown := card.CurrentCooldown > 0
	isDisabled := !card.IsAvailable && card.CurrentCooldown == 0

	// 第1层：冷却中或阳光不足时，绘制浅灰色底层遮罩（表示禁用状态）
	if isCoolingDown || isDisabled {
		baseMask := ebiten.NewImage(intCardWidth, intCardHeight)
		// 浅灰色全遮罩，透明度120
		baseMask.Fill(color.RGBA{0, 0, 0, 120})

		baseMaskOp := &ebiten.DrawImageOptions{}
		baseMaskOp.GeoM.Translate(x, y)
		screen.DrawImage(baseMask, baseMaskOp)
	}

	// 第2层：冷却中时，额外绘制深黑色动态遮罩（从下往上逐渐恢复）
	if isCoolingDown {
		cooldownProgress := getCooldownProgress(card)
		if cooldownProgress > 0 {
			maskHeight := cardHeight * cooldownProgress
			intMaskHeight := int(maskHeight)

			// 确保遮罩高度至少为1像素
			if intMaskHeight > 0 {
				cooldownMask := ebiten.NewImage(intCardWidth, intMaskHeight)
				// 深黑色遮罩，透明度180（比底层遮罩更深，叠加显示冷却进度）
				cooldownMask.Fill(color.RGBA{0, 0, 0, 180})

				cooldownMaskOp := &ebiten.DrawImageOptions{}
				// 遮罩固定在顶部，高度减少时底部先露出（从下到上恢复）
				cooldownMaskOp.GeoM.Translate(x, y)
				screen.DrawImage(cooldownMask, cooldownMaskOp)
			}
		}
	}
}

// getCooldownProgress 计算冷却进度（0.0-1.0）
// 只要有冷却时间就返回进度，不管阳光是否足够
func getCooldownProgress(card *components.PlantCardComponent) float64 {
	if card.CurrentCooldown > 0 && card.CooldownTime > 0 {
		return card.CurrentCooldown / card.CooldownTime
	}
	return 0.0
}
