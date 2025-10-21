package systems

import (
	"image/color"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// RewardPanelRenderSystem 负责渲染奖励面板 UI。
// 包括背景、文本信息等元素的绘制。
// 植物卡片通过创建实体，由 PlantCardRenderSystem 统一渲染。
type RewardPanelRenderSystem struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager
	reanimSystem    *ReanimSystem // 用于离屏渲染植物图标
	screenWidth     float64
	screenHeight    float64
	titleFont       *text.GoTextFace // 标题字体
	plantInfoFont   *text.GoTextFace // 植物名称和描述字体
	buttonFont      *text.GoTextFace // 按钮字体

	// 植物卡片实体映射：面板实体ID -> 卡片实体ID
	plantCardEntities map[ecs.EntityID]ecs.EntityID
}

// NewRewardPanelRenderSystem 创建奖励面板渲染系统。
func NewRewardPanelRenderSystem(em *ecs.EntityManager, gs *game.GameState, rm *game.ResourceManager, rs *ReanimSystem) *RewardPanelRenderSystem {
	// 加载中文 TTF 字体（使用配置中的4种字体大小）
	titleFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.RewardPanelTitleFontSize)
	if err != nil {
		log.Printf("[RewardPanelRenderSystem] Warning: Failed to load title font: %v", err)
	}

	plantInfoFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.RewardPanelPlantInfoFontSize)
	if err != nil {
		log.Printf("[RewardPanelRenderSystem] Warning: Failed to load plant info font: %v", err)
	}

	buttonFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.RewardPanelButtonTextFontSize)
	if err != nil {
		log.Printf("[RewardPanelRenderSystem] Warning: Failed to load button font: %v", err)
	}

	return &RewardPanelRenderSystem{
		entityManager:     em,
		gameState:         gs,
		resourceManager:   rm,
		reanimSystem:      rs,
		screenWidth:       800, // TODO: 从配置获取
		screenHeight:      600,
		titleFont:         titleFont,
		plantInfoFont:     plantInfoFont,
		buttonFont:        buttonFont,
		plantCardEntities: make(map[ecs.EntityID]ecs.EntityID), // 初始化卡片实体映射
	}
}

// Draw 绘制奖励面板到屏幕。
func (rprs *RewardPanelRenderSystem) Draw(screen *ebiten.Image) {
	// 查询所有奖励面板实体
	panelEntities := ecs.GetEntitiesWith1[*components.RewardPanelComponent](rprs.entityManager)

	for _, entity := range panelEntities {
		panelComp, ok := ecs.GetComponent[*components.RewardPanelComponent](rprs.entityManager, entity)
		if !ok || !panelComp.IsVisible {
			continue
		}

		// Story 8.4: 奖励面板作为独立模块，直接渲染所有元素
		// 不再依赖外部 PlantCardRenderSystem
		rprs.drawPanel(screen, entity, panelComp)
	}
}

// drawPanel 绘制奖励面板内容。
func (rprs *RewardPanelRenderSystem) drawPanel(screen *ebiten.Image, panelEntity ecs.EntityID, panel *components.RewardPanelComponent) {
	// 1. 绘制背景（AwardScreen_Back.jpg）
	rprs.drawBackground(screen, panel.FadeAlpha)

	// 2. 绘制标题文本："你得到了一株新植物！"
	rprs.drawTitle(screen, panel.FadeAlpha)

	// 3. 绘制植物卡片（独立渲染，不依赖外部系统）
	rprs.drawPlantCard(screen, panel)

	// 4. 绘制植物名称和描述
	rprs.drawPlantInfo(screen, panel)

	// 5. 绘制"下一关"按钮
	rprs.drawNextLevelButton(screen, panel.FadeAlpha)
}

// drawBackground 绘制奖励背景。
func (rprs *RewardPanelRenderSystem) drawBackground(screen *ebiten.Image, alpha float64) {
	bgImage := rprs.resourceManager.GetImageByID("IMAGE_AWARDSCREEN_BACK")
	if bgImage == nil {
		log.Printf("[RewardPanelRenderSystem] WARNING: IMAGE_AWARDSCREEN_BACK not loaded! Drawing fallback overlay")
		// 如果背景图未加载，绘制半透明黑色遮罩
		overlay := ebiten.NewImage(int(rprs.screenWidth), int(rprs.screenHeight))
		overlay.Fill(color.RGBA{0, 0, 0, uint8(alpha * 200)})
		screen.DrawImage(overlay, &ebiten.DrawImageOptions{})
		return
	}

	log.Printf("[RewardPanelRenderSystem] Drawing background with alpha=%.2f", alpha)
	// 绘制背景图（全屏）
	op := &ebiten.DrawImageOptions{}

	// 缩放到屏幕尺寸
	bgWidth, bgHeight := bgImage.Bounds().Dx(), bgImage.Bounds().Dy()
	scaleX := rprs.screenWidth / float64(bgWidth)
	scaleY := rprs.screenHeight / float64(bgHeight)
	op.GeoM.Scale(scaleX, scaleY)

	// 应用透明度
	op.ColorScale.ScaleAlpha(float32(alpha))

	screen.DrawImage(bgImage, op)
}

// drawTitle 绘制标题文本。
func (rprs *RewardPanelRenderSystem) drawTitle(screen *ebiten.Image, alpha float64) {
	if alpha < 0.5 || rprs.titleFont == nil {
		return
	}

	// 从 LawnStrings 获取标题文本
	titleText := "你得到了一株新植物！" // 默认文本
	if rprs.gameState.LawnStrings != nil {
		titleText = rprs.gameState.LawnStrings.GetString("NEW_PLANT")
		if titleText == "" {
			titleText = "你得到了一株新植物！"
		}
	}

	// 使用配置中的背景尺寸和位置比例
	bgWidth := config.RewardPanelBackgroundWidth
	bgHeight := config.RewardPanelBackgroundHeight

	// 计算背景在屏幕上的位置偏移
	offsetX := (rprs.screenWidth - bgWidth) / 2
	offsetY := (rprs.screenHeight - bgHeight) / 2

	// 使用 TTF 字体渲染标题（使用配置中的颜色和位置，带阴影效果）
	titleX := offsetX + bgWidth/2                         // 背景中心X
	titleY := offsetY + bgHeight*config.RewardPanelTitleY // 使用配置的Y位置

	// 1. 先绘制阴影（黑色，稍微偏移）
	shadowOp := &text.DrawOptions{}
	shadowOp.GeoM.Translate(titleX+2, titleY+2) // 阴影偏移2像素
	shadowOp.PrimaryAlign = text.AlignCenter
	shadowOp.SecondaryAlign = text.AlignStart
	shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 128}) // 半透明黑色
	shadowOp.ColorScale.ScaleAlpha(float32(alpha))
	text.Draw(screen, titleText, rprs.titleFont, shadowOp)

	// 2. 再绘制主文字（橙黄色）
	op := &text.DrawOptions{}
	op.GeoM.Translate(titleX, titleY)
	op.PrimaryAlign = text.AlignCenter                         // 水平居中
	op.SecondaryAlign = text.AlignStart                        // 垂直从上开始
	op.ColorScale.ScaleWithColor(config.RewardPanelTitleColor) // 使用配置的橙黄色
	op.ColorScale.ScaleAlpha(float32(alpha))
	text.Draw(screen, titleText, rprs.titleFont, op)
}

// drawPlantCard 绘制植物卡片（独立渲染，不依赖外部系统）。
// Story 8.4: 使用统一的植物卡片渲染函数
func (rprs *RewardPanelRenderSystem) drawPlantCard(screen *ebiten.Image, panel *components.RewardPanelComponent) {
	// 降低透明度阈值，让卡片更早显示（与面板淡入同步）
	if panel.FadeAlpha < 0.01 {
		return // 透明度太低时不绘制
	}

	// 映射 plantID 到 PlantType
	plantType := rprs.getPlantType(panel.PlantID)
	if plantType == components.PlantUnknown {
		log.Printf("[RewardPanelRenderSystem] Unknown plant ID: %s", panel.PlantID)
		return
	}

	// 计算卡片缩放因子和位置
	cardScale := panel.CardScale * config.RewardPanelCardScale
	cardX, cardY := rprs.calculateCardPosition(cardScale)

	// 创建临时的 PlantCardComponent 用于渲染
	// 使用 entities.RenderPlantCard 统一渲染函数（Story 8.4）
	tempCard := &components.PlantCardComponent{
		PlantType:       plantType,
		CardScale:       cardScale,
		SunCost:         panel.SunCost,
		CooldownTime:    0, // 奖励卡片没有冷却
		CurrentCooldown: 0,
		IsAvailable:     true,
		Alpha:           panel.FadeAlpha, // 使用面板的淡入透明度
	}

	// 加载卡片资源（背景和植物图标）
	tempCard.BackgroundImage = rprs.resourceManager.GetImageByID(config.PlantCardBackgroundID)
	if tempCard.BackgroundImage == nil {
		log.Printf("[RewardPanelRenderSystem] WARNING: Card background image not loaded! ID: %s", config.PlantCardBackgroundID)
	}

	// 获取 Reanim 名称
	reanimName := rprs.getReanimName(plantType)
	if reanimName == "" {
		log.Printf("[RewardPanelRenderSystem] No reanim name for plant type: %d", plantType)
		return
	}

	// 使用 ReanimSystem 渲染植物图标
	plantIcon, err := entities.RenderPlantIcon(
		rprs.entityManager,
		rprs.resourceManager,
		rprs.reanimSystem,
		reanimName,
	)
	if err != nil {
		log.Printf("[RewardPanelRenderSystem] Failed to render plant icon: %v", err)
		return
	}
	tempCard.PlantIconTexture = plantIcon

	// 调用统一渲染函数绘制卡片
	entities.RenderPlantCard(
		screen,
		tempCard,
		cardX,
		cardY,
		nil, // sunFont（不显示阳光数字）
		0,   // sunFontSize
	)
}

// ensurePlantCardEntity 确保植物卡片实体存在，如果不存在则创建。
// 植物卡片实体由 PlantCardRenderSystem 统一渲染。
func (rprs *RewardPanelRenderSystem) ensurePlantCardEntity(panelEntity ecs.EntityID, panel *components.RewardPanelComponent) {
	// 检查是否已经创建了卡片实体
	if _, exists := rprs.plantCardEntities[panelEntity]; exists {
		// 卡片实体已存在，更新位置
		rprs.updatePlantCardEntity(panelEntity, panel)
		return
	}

	// 映射 plantID 到 PlantType
	plantType := rprs.getPlantType(panel.PlantID)
	if plantType == components.PlantUnknown {
		log.Printf("[RewardPanelRenderSystem] Unknown plant ID: %s", panel.PlantID)
		return
	}

	// 计算卡片缩放因子
	cardScale := panel.CardScale * config.RewardPanelCardScale

	// 自动计算卡片位置（水平居中，垂直位置从配置读取）
	cardX, cardY := rprs.calculateCardPosition(cardScale)

	// 创建植物卡片实体（使用现有的工厂函数）
	cardEntity, err := entities.NewPlantCardEntity(
		rprs.entityManager,
		rprs.resourceManager,
		rprs.reanimSystem,
		plantType,
		cardX,
		cardY,
		cardScale,
	)

	if err != nil {
		log.Printf("[RewardPanelRenderSystem] Failed to create plant card entity: %v", err)
		return
	}

	// 保存卡片实体ID
	rprs.plantCardEntities[panelEntity] = cardEntity
	log.Printf("[RewardPanelRenderSystem] Created plant card entity %d for panel %d (plant: %s)",
		cardEntity, panelEntity, panel.PlantID)
}

// calculateCardPosition 计算卡片的居中位置
// 返回: (cardX, cardY) 屏幕坐标
func (rprs *RewardPanelRenderSystem) calculateCardPosition(cardScale float64) (float64, float64) {
	// 背景在屏幕上的偏移
	bgWidth := config.RewardPanelBackgroundWidth
	bgHeight := config.RewardPanelBackgroundHeight
	offsetX := (rprs.screenWidth - bgWidth) / 2
	offsetY := (rprs.screenHeight - bgHeight) / 2

	// 获取卡片原始尺寸（假设为固定值，可从资源获取）
	// SeedPacket_Larger.png 原始尺寸约为 100x140
	cardOriginalWidth := 100.0
	cardWidth := cardOriginalWidth * cardScale

	// 自动计算水平居中位置
	cardX := offsetX + (bgWidth-cardWidth)/2

	// 从配置读取垂直位置比例
	cardY := offsetY + bgHeight*config.RewardPanelCardYRatio

	return cardX, cardY
}

// updatePlantCardEntity 更新卡片实体的位置和透明度。
func (rprs *RewardPanelRenderSystem) updatePlantCardEntity(panelEntity ecs.EntityID, panel *components.RewardPanelComponent) {
	cardEntity, exists := rprs.plantCardEntities[panelEntity]
	if !exists {
		return
	}

	// 重新计算位置（如果面板大小改变）
	cardScale := panel.CardScale * config.RewardPanelCardScale
	cardX, cardY := rprs.calculateCardPosition(cardScale)

	// 更新位置
	if posComp, ok := ecs.GetComponent[*components.PositionComponent](rprs.entityManager, cardEntity); ok {
		posComp.X = cardX
		posComp.Y = cardY
	}

	// 更新卡片缩放和透明度（同步面板的淡入淡出效果）
	if cardComp, ok := ecs.GetComponent[*components.PlantCardComponent](rprs.entityManager, cardEntity); ok {
		cardComp.CardScale = cardScale
		cardComp.Alpha = panel.FadeAlpha // Story 8.4: 同步面板透明度，使卡片与面板一起淡入淡出
	}
}

// getPlantType 将 plantID 映射到 PlantType 枚举。
func (rprs *RewardPanelRenderSystem) getPlantType(plantID string) components.PlantType {
	switch plantID {
	case "sunflower":
		return components.PlantSunflower
	case "peashooter":
		return components.PlantPeashooter
	case "cherrybomb":
		return components.PlantCherryBomb
	case "wallnut":
		return components.PlantWallnut
	default:
		return components.PlantUnknown
	}
}

// getReanimName 根据 PlantType 获取 Reanim 名称
// 注意：必须与 ResourceManager.LoadReanimResources() 中的名称完全一致
func (rprs *RewardPanelRenderSystem) getReanimName(plantType components.PlantType) string {
	switch plantType {
	case components.PlantSunflower:
		return "SunFlower" // 修复：与资源加载时的名称一致
	case components.PlantPeashooter:
		return "PeaShooter" // 修复：与资源加载时的名称一致
	case components.PlantCherryBomb:
		return "CherryBomb"
	case components.PlantWallnut:
		return "Wallnut" // 修复：与资源加载时的名称一致（小写n）
	default:
		return ""
	}
}

// drawPlantInfo 绘制植物名称和描述。
func (rprs *RewardPanelRenderSystem) drawPlantInfo(screen *ebiten.Image, panel *components.RewardPanelComponent) {
	if panel.FadeAlpha > 0.5 && rprs.plantInfoFont != nil {
		// 使用配置中的背景尺寸
		bgWidth := config.RewardPanelBackgroundWidth
		bgHeight := config.RewardPanelBackgroundHeight
		offsetX := (rprs.screenWidth - bgWidth) / 2
		offsetY := (rprs.screenHeight - bgHeight) / 2

		// 绘制植物名称（使用配置中的颜色和位置，带阴影效果）
		nameX := offsetX + bgWidth/2                             // 背景中心X
		nameY := offsetY + bgHeight*config.RewardPanelPlantNameY // 使用配置的Y位置

		if panel.PlantName != "" {
			// 1. 先绘制阴影（黑色，稍微偏移）
			shadowOp := &text.DrawOptions{}
			shadowOp.GeoM.Translate(nameX+2, nameY+2) // 阴影偏移2像素
			shadowOp.PrimaryAlign = text.AlignCenter
			shadowOp.SecondaryAlign = text.AlignStart
			shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 128}) // 半透明黑色
			shadowOp.ColorScale.ScaleAlpha(float32(panel.FadeAlpha))
			text.Draw(screen, panel.PlantName, rprs.plantInfoFont, shadowOp)

			// 2. 再绘制主文字（金黄色）
			op := &text.DrawOptions{}
			op.GeoM.Translate(nameX, nameY)
			op.PrimaryAlign = text.AlignCenter                             // 水平居中
			op.SecondaryAlign = text.AlignStart                            // 垂直从上开始
			op.ColorScale.ScaleWithColor(config.RewardPanelPlantNameColor) // 使用配置的金黄色
			op.ColorScale.ScaleAlpha(float32(panel.FadeAlpha))
			text.Draw(screen, panel.PlantName, rprs.plantInfoFont, op)
		}

		// 绘制植物描述（使用配置中的颜色和位置，使用植物信息字体）
		descX := offsetX + bgWidth/2                               // 背景中心X
		descY := offsetY + bgHeight*config.RewardPanelDescriptionY // 使用配置的Y位置

		if panel.PlantDescription != "" {
			op := &text.DrawOptions{}
			op.GeoM.Translate(descX, descY)
			op.PrimaryAlign = text.AlignCenter                               // 水平居中
			op.SecondaryAlign = text.AlignStart                              // 垂直从上开始
			op.ColorScale.ScaleWithColor(config.RewardPanelDescriptionColor) // 使用配置的深蓝黑色
			op.ColorScale.ScaleAlpha(float32(panel.FadeAlpha))
			text.Draw(screen, panel.PlantDescription, rprs.plantInfoFont, op) // 使用植物信息字体（与植物名称一样大）
		}
	}
}

// drawHint 绘制提示文本。
func (rprs *RewardPanelRenderSystem) drawHint(screen *ebiten.Image, alpha float64) {
	if alpha < 0.5 || rprs.plantInfoFont == nil {
		return
	}

	hintText := "点击任意位置继续" // 默认文本
	if rprs.gameState.LawnStrings != nil {
		hintText = rprs.gameState.LawnStrings.GetString("CLICK_TO_CONTINUE")
		if hintText == "" {
			hintText = "点击任意位置继续"
		}
	}

	// 使用 TTF 字体渲染提示
	hintX := rprs.screenWidth / 2
	hintY := rprs.screenHeight * 0.85

	op := &text.DrawOptions{}
	op.GeoM.Translate(hintX-70, hintY)                           // 居中偏移
	op.ColorScale.ScaleWithColor(color.RGBA{200, 200, 200, 255}) // 灰白色
	op.ColorScale.ScaleAlpha(float32(alpha))
	text.Draw(screen, hintText, rprs.plantInfoFont, op)
}

// drawNextLevelButton 绘制"下一关"按钮（在面板底部）。
func (rprs *RewardPanelRenderSystem) drawNextLevelButton(screen *ebiten.Image, alpha float64) {
	if alpha < 0.5 {
		return
	}

	// 加载按钮图片（使用 SeedChooser_Button.png）
	buttonImage := rprs.resourceManager.GetImageByID("IMAGE_SEEDCHOOSER_BUTTON")
	if buttonImage == nil {
		log.Printf("[RewardPanelRenderSystem] WARNING: IMAGE_SEEDCHOOSER_BUTTON not loaded! Drawing fallback button")
		// 如果资源未加载，绘制简单的矩形按钮作为后备
		rprs.drawFallbackButton(screen, alpha)
		return
	}

	log.Printf("[RewardPanelRenderSystem] Drawing next level button with alpha=%.2f", alpha)

	// 奖励背景是 800x600
	bgWidth := config.RewardPanelBackgroundWidth
	bgHeight := config.RewardPanelBackgroundHeight
	offsetX := (rprs.screenWidth - bgWidth) / 2
	offsetY := (rprs.screenHeight - bgHeight) / 2

	// 计算按钮位置（使用配置的Y位置）
	buttonX := offsetX + bgWidth/2
	buttonY := offsetY + bgHeight*config.RewardPanelButtonY // 使用配置的Y位置

	// 绘制按钮图片
	buttonWidth := float64(buttonImage.Bounds().Dx())
	buttonHeight := float64(buttonImage.Bounds().Dy())

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(buttonX-buttonWidth/2, buttonY-buttonHeight/2)
	op.ColorScale.ScaleAlpha(float32(alpha))
	screen.DrawImage(buttonImage, op)

	// 绘制按钮文字（"下一关"，带阴影效果）
	if rprs.buttonFont != nil {
		buttonText := "下一关"

		// 1. 先绘制阴影（黑色，稍微偏移）
		shadowOp := &text.DrawOptions{}
		shadowOp.GeoM.Translate(buttonX+2, buttonY+2) // 阴影偏移2像素
		shadowOp.PrimaryAlign = text.AlignCenter
		shadowOp.SecondaryAlign = text.AlignCenter
		shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 128}) // 半透明黑色
		shadowOp.ColorScale.ScaleAlpha(float32(alpha))
		text.Draw(screen, buttonText, rprs.buttonFont, shadowOp)

		// 2. 再绘制主文字（橙黄色）
		textOp := &text.DrawOptions{}
		textOp.GeoM.Translate(buttonX, buttonY)
		textOp.PrimaryAlign = text.AlignCenter                              // 水平居中
		textOp.SecondaryAlign = text.AlignCenter                            // 垂直居中
		textOp.ColorScale.ScaleWithColor(config.RewardPanelButtonTextColor) // 使用配置的橙黄色
		textOp.ColorScale.ScaleAlpha(float32(alpha))
		text.Draw(screen, buttonText, rprs.buttonFont, textOp)
	}
}

// drawFallbackButton 绘制后备按钮（当资源未加载时）。
func (rprs *RewardPanelRenderSystem) drawFallbackButton(screen *ebiten.Image, alpha float64) {
	buttonX := rprs.screenWidth / 2
	buttonY := rprs.screenHeight * 0.88

	// 绘制按钮背景（简单的矩形）
	buttonWidth := 120.0
	buttonHeight := 40.0
	buttonRect := ebiten.NewImage(int(buttonWidth), int(buttonHeight))
	buttonRect.Fill(color.RGBA{60, 120, 60, uint8(alpha * 255)}) // 绿色背景

	rectOp := &ebiten.DrawImageOptions{}
	rectOp.GeoM.Translate(buttonX-buttonWidth/2, buttonY-buttonHeight/2)
	screen.DrawImage(buttonRect, rectOp)

	// 绘制按钮文字
	if rprs.titleFont != nil {
		buttonText := "下一关"
		op := &text.DrawOptions{}
		op.GeoM.Translate(buttonX-30, buttonY-10)                    // 居中偏移
		op.ColorScale.ScaleWithColor(color.RGBA{255, 255, 255, 255}) // 白色文字
		op.ColorScale.ScaleAlpha(float32(alpha))
		text.Draw(screen, buttonText, rprs.titleFont, op)
	}
}

// findPhysicalFrameIndex 将逻辑帧号映射到物理帧索引（简化版）。
func (rprs *RewardPanelRenderSystem) findPhysicalFrameIndex(reanim *components.ReanimComponent, logicalFrameNum int) int {
	if len(reanim.AnimVisibles) == 0 {
		return -1
	}

	logicalIndex := 0
	for i := 0; i < len(reanim.AnimVisibles); i++ {
		if reanim.AnimVisibles[i] == 0 {
			if logicalIndex == logicalFrameNum {
				return i
			}
			logicalIndex++
		}
	}

	return -1
}
