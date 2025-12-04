package systems

import (
	"image/color"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/utils"
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
	sunCostFont     *text.GoTextFace // 阳光数字字体（用于卡片）

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

	// 加载阳光数字字体（用于植物卡片）
	sunCostFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.PlantCardSunCostFontSize)
	if err != nil {
		log.Printf("[RewardPanelRenderSystem] Warning: Failed to load sun cost font: %v", err)
		sunCostFont = nil
	}

	return &RewardPanelRenderSystem{
		entityManager:     em,
		gameState:         gs,
		resourceManager:   rm,
		reanimSystem:      rs,
		screenWidth:       800,
		screenHeight:      600,
		titleFont:         titleFont,
		plantInfoFont:     plantInfoFont,
		buttonFont:        buttonFont,
		sunCostFont:       sunCostFont,
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

		// 奖励面板作为独立模块，直接渲染所有元素
		// 不再依赖外部 PlantCardRenderSystem
		rprs.drawPanel(screen, entity, panelComp)
	}
}

// drawPanel 绘制奖励面板内容。
func (rprs *RewardPanelRenderSystem) drawPanel(screen *ebiten.Image, panelEntity ecs.EntityID, panel *components.RewardPanelComponent) {
	// 1. 绘制背景（AwardScreen_Back.jpg）
	rprs.drawBackground(screen, panel.FadeAlpha)

	// 2. 绘制标题文本："你得到了一株新植物！" 或 "你得到了一个新工具！"
	rprs.drawTitle(screen, panel)

	// 3. 绘制奖励卡片/图标（根据类型选择）
	rprs.drawRewardCard(screen, panel)

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
func (rprs *RewardPanelRenderSystem) drawTitle(screen *ebiten.Image, panel *components.RewardPanelComponent) {
	if panel.FadeAlpha < 0.5 || rprs.titleFont == nil {
		return
	}

	// 根据奖励类型��择标题文本
	var titleText string
	if panel.RewardType == "tool" {
		// 工具奖励标题 - 从 LawnStrings 加载
		titleText = "你得到一把铁铲！" // 默认文本
		if rprs.gameState.LawnStrings != nil {
			titleText = rprs.gameState.LawnStrings.GetString("GOT_SHOVEL")
			if titleText == "" {
				titleText = "你得到一把铁铲！"
			}
		}
	} else {
		titleText = "你得到了一株新植物！" // 植物奖励标题（默认）
		if rprs.gameState.LawnStrings != nil {
			titleText = rprs.gameState.LawnStrings.GetString("NEW_PLANT")
			if titleText == "" {
				titleText = "你得���了一株新植物！"
			}
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
	shadowOp.ColorScale.ScaleAlpha(float32(panel.FadeAlpha))
	text.Draw(screen, titleText, rprs.titleFont, shadowOp)

	// 2. 再绘制主文字（橙黄色）
	op := &text.DrawOptions{}
	op.GeoM.Translate(titleX, titleY)
	op.PrimaryAlign = text.AlignCenter                         // 水平居中
	op.SecondaryAlign = text.AlignStart                        // 垂直从上开始
	op.ColorScale.ScaleWithColor(config.RewardPanelTitleColor) // 使用配置的橙黄色
	op.ColorScale.ScaleAlpha(float32(panel.FadeAlpha))
	text.Draw(screen, titleText, rprs.titleFont, op)
}

// drawRewardCard 绘制奖励卡片/图标（统一入口）
// 根据 RewardType 选择绘制植物卡片或工具图标
func (rprs *RewardPanelRenderSystem) drawRewardCard(screen *ebiten.Image, panel *components.RewardPanelComponent) {
	if panel.RewardType == "tool" {
		rprs.drawToolIcon(screen, panel)
	} else {
		rprs.drawPlantCard(screen, panel)
	}
}

// drawPlantCard 绘制植物卡片（独立渲染，不依赖外部系统）。
// 使用统一的植物卡片渲染函数
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

	// 使用 ReanimSystem 渲染植物图标（直接传入 plantType）
	plantIcon, err := entities.RenderPlantIcon(
		rprs.entityManager,
		rprs.resourceManager,
		rprs.reanimSystem,
		plantType,
	)
	if err != nil {
		log.Printf("[RewardPanelRenderSystem] Failed to render plant icon: %v", err)
		return
	}
	tempCard.PlantIconTexture = plantIcon

	// 调用统一渲染函数绘制卡片
	// 从 GoTextFace 中提取 GoTextFaceSource
	var sunFontSource *text.GoTextFaceSource
	if rprs.sunCostFont != nil {
		sunFontSource = rprs.sunCostFont.Source
	}
	entities.RenderPlantCard(
		screen,
		tempCard,
		cardX,
		cardY,
		sunFontSource,                   // 阳光数字字体源
		config.PlantCardSunCostFontSize, // 字体大小
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

// calculateCardPosition 计算卡片在显示区域内的居中位置
// 使用配置的卡片显示区域坐标，让卡片在区域内水平和垂直居中
// 返回: (cardX, cardY) 屏幕坐标（卡片左上角）
func (rprs *RewardPanelRenderSystem) calculateCardPosition(cardScale float64) (float64, float64) {
	// 背景在屏幕上的偏移
	bgWidth := config.RewardPanelBackgroundWidth
	bgHeight := config.RewardPanelBackgroundHeight
	offsetX := (rprs.screenWidth - bgWidth) / 2
	offsetY := (rprs.screenHeight - bgHeight) / 2

	// 获取配置的卡片显示区域（相对于背景图片）
	boxLeft := offsetX + config.RewardPanelCardBoxLeft
	boxTop := offsetY + config.RewardPanelCardBoxTop
	boxWidth := config.RewardPanelCardBoxWidth
	boxHeight := config.RewardPanelCardBoxHeight

	// 获取卡片缩放后的尺寸
	// SeedPacket_Larger.png 原始尺寸约为 100x140
	cardOriginalWidth := 100.0
	cardOriginalHeight := 140.0
	cardWidth := cardOriginalWidth * cardScale
	cardHeight := cardOriginalHeight * cardScale

	// 在显示区域内居中
	cardX := boxLeft + (boxWidth-cardWidth)/2
	cardY := boxTop + (boxHeight-cardHeight)/2

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
		cardComp.Alpha = panel.FadeAlpha // 同步面板透明度，使卡片与面板一起淡入淡出
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
	case "potatomine":
		return components.PlantPotatoMine
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
		return "PeaShooterSingle" // 修正为普通豌豆射手资源
	case components.PlantCherryBomb:
		return "CherryBomb"
	case components.PlantWallnut:
		return "Wallnut" // 修复：与资源加载时的名称一致（小写n）
	case components.PlantPotatoMine:
		return "PotatoMine"
	default:
		return ""
	}
}

// getConfigID 返回配置文件中的ID（Story 13.8）
func (rprs *RewardPanelRenderSystem) getConfigID(plantType components.PlantType) string {
	switch plantType {
	case components.PlantSunflower:
		return "sunflower"
	case components.PlantPeashooter:
		return "peashooter"
	case components.PlantCherryBomb:
		return "cherrybomb"
	case components.PlantWallnut:
		return "wallnut"
	case components.PlantPotatoMine:
		return "potatomine"
	default:
		return ""
	}
}

// drawToolIcon 绘制工具图标（铲子）
func (rprs *RewardPanelRenderSystem) drawToolIcon(screen *ebiten.Image, panel *components.RewardPanelComponent) {
	// 降低透明度阈值，让图标更早显示（与面板淡入同步）
	if panel.FadeAlpha < 0.01 {
		return // 透明度太低时不绘制
	}

	// 加载铲子图片（奖励面板使用高清版本）
	shovelImage := rprs.resourceManager.GetImageByID("IMAGE_SHOVEL_HI_RES")
	if shovelImage == nil {
		log.Printf("[RewardPanelRenderSystem] Warning: Failed to load IMAGE_SHOVEL_HI_RES")
		return
	}

	// 计算显示区域的中心位置
	bgWidth := config.RewardPanelBackgroundWidth
	bgHeight := config.RewardPanelBackgroundHeight
	offsetX := (rprs.screenWidth - bgWidth) / 2
	offsetY := (rprs.screenHeight - bgHeight) / 2

	// 获取配置的卡片显示区域（相对于背景图片）
	boxLeft := offsetX + config.RewardPanelCardBoxLeft
	boxTop := offsetY + config.RewardPanelCardBoxTop
	boxWidth := config.RewardPanelCardBoxWidth
	boxHeight := config.RewardPanelCardBoxHeight

	// 计算显示区域的中心位置（这是图片的锚点）
	centerX := boxLeft + boxWidth/2
	centerY := boxTop + boxHeight/2

	// 应用缩放动画
	iconScale := panel.CardScale * config.RewardPanelCardScale
	op := &ebiten.DrawImageOptions{}

	// 居中图片（将图片中心作为锚点）
	bounds := shovelImage.Bounds()
	op.GeoM.Translate(-float64(bounds.Dx())/2, -float64(bounds.Dy())/2)

	// 缩放（随时间变化）
	op.GeoM.Scale(iconScale, iconScale)

	// 移动到显示区域中心位置
	op.GeoM.Translate(centerX, centerY)

	// 应用透明度
	op.ColorScale.ScaleAlpha(float32(panel.FadeAlpha))

	screen.DrawImage(shovelImage, op)
}

// drawPlantInfo 绘制植物名称和描述。
// 名称使用 RewardPanelPlantNameY 配置的位置（在卡片正下方）
// 描述使用描述框坐标范围 (360,260)-(540,470)，支持多行文本垂直居中。
func (rprs *RewardPanelRenderSystem) drawPlantInfo(screen *ebiten.Image, panel *components.RewardPanelComponent) {
	if panel.FadeAlpha < 0.5 || rprs.plantInfoFont == nil {
		return
	}

	// 使用配置中的背景尺寸和位置比例
	bgWidth := config.RewardPanelBackgroundWidth
	bgHeight := config.RewardPanelBackgroundHeight

	// 计算背景在屏幕上的位置偏移
	offsetX := (rprs.screenWidth - bgWidth) / 2
	offsetY := (rprs.screenHeight - bgHeight) / 2

	// 1. 绘制植物名称（在卡片正下方，使用 RewardPanelPlantNameY）
	if panel.PlantName != "" {
		nameX := offsetX + bgWidth/2                             // 背景中心X
		nameY := offsetY + bgHeight*config.RewardPanelPlantNameY // 使用配置的Y位置

		// 1.1 先绘制阴影（黑色，稍微偏移）
		shadowOp := &text.DrawOptions{}
		shadowOp.GeoM.Translate(nameX+2, nameY+2) // 阴影偏移2像素
		shadowOp.PrimaryAlign = text.AlignCenter
		shadowOp.SecondaryAlign = text.AlignStart
		shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 128}) // 半透明黑色
		shadowOp.ColorScale.ScaleAlpha(float32(panel.FadeAlpha))
		text.Draw(screen, panel.PlantName, rprs.plantInfoFont, shadowOp)

		// 1.2 再绘制主文字（金黄色）
		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(nameX, nameY)
		nameOp.PrimaryAlign = text.AlignCenter                             // 水平居中
		nameOp.SecondaryAlign = text.AlignStart                            // 垂直从上开始
		nameOp.ColorScale.ScaleWithColor(config.RewardPanelPlantNameColor) // 金黄色
		nameOp.ColorScale.ScaleAlpha(float32(panel.FadeAlpha))
		text.Draw(screen, panel.PlantName, rprs.plantInfoFont, nameOp)
	}

	// 2. 绘制植物描述（在描述框内，垂直居中）
	if panel.PlantDescription != "" {
		// 使用配置的描述框坐标范围（相对于背景图片）
		// 需要加上背景在屏幕上的偏移量
		boxLeft := offsetX + config.RewardPanelDescBoxLeft
		boxTop := offsetY + config.RewardPanelDescBoxTop
		boxWidth := config.RewardPanelDescBoxWidth
		boxHeight := config.RewardPanelDescBoxHeight

		// 计算描述框中心X坐标（用于水平居中）
		boxCenterX := boxLeft + boxWidth/2

		// 使用 WrapText 进行自动换行（基于描述框宽度）
		lines := utils.WrapText(
			panel.PlantDescription,
			rprs.plantInfoFont,
			boxWidth-20, // 留出左右边距各10像素
		)

		// 计算总文本高度
		totalTextHeight := float64(len(lines)) * config.RewardPanelDescriptionLineSpacing
		if len(lines) > 0 {
			// 减去最后一行多余的行间距
			totalTextHeight -= (config.RewardPanelDescriptionLineSpacing - float64(rprs.plantInfoFont.Size))
		}

		// 计算垂直居中的起始Y坐标
		startY := boxTop + (boxHeight-totalTextHeight)/2

		// 绘制每一行
		currentY := startY
		for _, line := range lines {
			// 绘制主文字（深蓝黑色，无阴影）
			descOp := &text.DrawOptions{}
			descOp.GeoM.Translate(boxCenterX, currentY)
			descOp.PrimaryAlign = text.AlignCenter                               // 水平居中
			descOp.SecondaryAlign = text.AlignStart                              // 垂直从上开始
			descOp.ColorScale.ScaleWithColor(config.RewardPanelDescriptionColor) // 深蓝黑色
			descOp.ColorScale.ScaleAlpha(float32(panel.FadeAlpha))
			text.Draw(screen, line, rprs.plantInfoFont, descOp)

			// 移动到下一行
			currentY += config.RewardPanelDescriptionLineSpacing
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

// isHoveringNextButton 检查鼠标是否悬停在"下一关"按钮上
func (rprs *RewardPanelRenderSystem) isHoveringNextButton() bool {
	// 获取鼠标位置
	mouseX, mouseY := ebiten.CursorPosition()

	// 计算按钮位置（与 drawNextLevelButton 逻辑一致）
	bgWidth := config.RewardPanelBackgroundWidth
	bgHeight := config.RewardPanelBackgroundHeight
	offsetX := (rprs.screenWidth - bgWidth) / 2
	offsetY := (rprs.screenHeight - bgHeight) / 2

	// 按钮中心位置
	buttonX := offsetX + bgWidth/2
	buttonY := offsetY + bgHeight*config.RewardPanelButtonY

	// 按钮尺寸（根据 IMAGE_SEEDCHOOSER_BUTTON 的尺寸，约 155x38）
	buttonWidth := 155.0
	buttonHeight := 50.0 // 增大高度以提高可点击性

	// AABB 碰撞检测
	halfWidth := buttonWidth / 2
	halfHeight := buttonHeight / 2

	return float64(mouseX) >= buttonX-halfWidth && float64(mouseX) <= buttonX+halfWidth &&
		float64(mouseY) >= buttonY-halfHeight && float64(mouseY) <= buttonY+halfHeight
}

// drawNextLevelButton 绘制"下一关"按钮（在面板底部）。
// 当鼠标悬停时，使用高亮版本的按钮图片
func (rprs *RewardPanelRenderSystem) drawNextLevelButton(screen *ebiten.Image, alpha float64) {
	if alpha < 0.5 {
		return
	}

	// 检查是否悬停
	isHovered := rprs.isHoveringNextButton()

	// 根据悬停状态选择按钮图片
	var buttonImage *ebiten.Image
	if isHovered {
		// 悬停时使用高亮图片
		buttonImage = rprs.resourceManager.GetImageByID("IMAGE_SEEDCHOOSER_BUTTON_GLOW")
	}
	if buttonImage == nil {
		// 使用普通按钮图片
		buttonImage = rprs.resourceManager.GetImageByID("IMAGE_SEEDCHOOSER_BUTTON")
	}

	if buttonImage == nil {
		log.Printf("[RewardPanelRenderSystem] WARNING: IMAGE_SEEDCHOOSER_BUTTON not loaded! Drawing fallback button")
		// 如果资源未加载，绘制简单的矩形按钮作为后备
		rprs.drawFallbackButton(screen, alpha)
		return
	}

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
	// 获取当前动画的 AnimVisibles
	if len(reanim.CurrentAnimations) == 0 {
		return -1
	}
	animVisibles := reanim.AnimVisiblesMap[reanim.CurrentAnimations[0]]
	if len(animVisibles) == 0 {
		return -1
	}

	logicalIndex := 0
	for i := 0; i < len(animVisibles); i++ {
		if animVisibles[i] == 0 {
			if logicalIndex == logicalFrameNum {
				return i
			}
			logicalIndex++
		}
	}

	return -1
}
