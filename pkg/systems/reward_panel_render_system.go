package systems

import (
	"image/color"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// RewardPanelRenderSystem 负责渲染奖励面板 UI。
// 包括背景、植物卡片、文本信息等元素的绘制。
// Story 8.4: 使用 PlantCardRenderer 进行卡片渲染，消除重复代码
type RewardPanelRenderSystem struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager
	reanimSystem    *ReanimSystem // 用于渲染植物 Reanim
	cardRenderer    *utils.PlantCardRenderer // Story 8.4: 通用卡片渲染器
	screenWidth     float64
	screenHeight    float64
	titleFont       *text.GoTextFace // 标题字体
	plantInfoFont   *text.GoTextFace // 植物名称和描述字体
	sunCostFont     *text.GoTextFace // 阳光值字体
	buttonFont      *text.GoTextFace // 按钮字体

	// 植物显示实体映射：面板实体ID -> 植物显示实体ID
	plantDisplayEntities map[ecs.EntityID]ecs.EntityID
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

	sunCostFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.RewardPanelSunCostFontSize)
	if err != nil {
		log.Printf("[RewardPanelRenderSystem] Warning: Failed to load sun cost font: %v", err)
	}

	buttonFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.RewardPanelButtonTextFontSize)
	if err != nil {
		log.Printf("[RewardPanelRenderSystem] Warning: Failed to load button font: %v", err)
	}

	return &RewardPanelRenderSystem{
		entityManager:        em,
		gameState:            gs,
		resourceManager:      rm,
		reanimSystem:         rs,
		cardRenderer:         utils.NewPlantCardRenderer(), // Story 8.4: 初始化渲染器
		screenWidth:          800, // TODO: 从配置获取
		screenHeight:         600,
		titleFont:            titleFont,
		plantInfoFont:        plantInfoFont,
		sunCostFont:          sunCostFont,
		buttonFont:           buttonFont,
		plantDisplayEntities: make(map[ecs.EntityID]ecs.EntityID),
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

		// 确保植物显示实体存在
		rprs.ensurePlantDisplayEntity(entity, panelComp)

		// 绘制奖励面板
		rprs.drawPanel(screen, entity, panelComp)
	}
}

// drawPanel 绘制奖励面板内容。
func (rprs *RewardPanelRenderSystem) drawPanel(screen *ebiten.Image, panelEntity ecs.EntityID, panel *components.RewardPanelComponent) {
	// 1. 绘制背景（AwardScreen_Back.jpg）
	rprs.drawBackground(screen, panel.FadeAlpha)

	// 2. 绘制标题文本："你得到了一株新植物！"
	rprs.drawTitle(screen, panel.FadeAlpha)

	// 3. 绘制植物卡片（SeedPacket_Larger.png）
	rprs.drawPlantCard(screen, panelEntity, panel)

	// 4. 绘制植物名称和描述
	rprs.drawPlantInfo(screen, panel)

	// 5. 绘制"下一关"按钮
	rprs.drawNextLevelButton(screen, panel.FadeAlpha)

	// 6. 绘制提示文本："点击任意位置继续"（现在改为点击按钮）
	// rprs.drawHint(screen, panel.FadeAlpha) // 不再需要，按钮已经说明了
}

// drawBackground 绘制奖励背景。
func (rprs *RewardPanelRenderSystem) drawBackground(screen *ebiten.Image, alpha float64) {
	bgImage := rprs.resourceManager.GetImageByID("IMAGE_AWARDSCREEN_BACK")
	if bgImage == nil {
		// 如果背景图未加载，绘制半透明黑色遮罩
		overlay := ebiten.NewImage(int(rprs.screenWidth), int(rprs.screenHeight))
		overlay.Fill(color.RGBA{0, 0, 0, uint8(alpha * 200)})
		screen.DrawImage(overlay, &ebiten.DrawImageOptions{})
		return
	}

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
	titleX := offsetX + bgWidth/2                   // 背景中心X
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
	op.PrimaryAlign = text.AlignCenter   // 水平居中
	op.SecondaryAlign = text.AlignStart  // 垂直从上开始
	op.ColorScale.ScaleWithColor(config.RewardPanelTitleColor) // 使用配置的橙黄色
	op.ColorScale.ScaleAlpha(float32(alpha))
	text.Draw(screen, titleText, rprs.titleFont, op)
}

// drawPlantCard 绘制植物卡片（背景框 + 植物 Reanim + 阳光数字）。
// Story 8.4: 使用 PlantCardRenderer 渲染背景框和阳光数字
func (rprs *RewardPanelRenderSystem) drawPlantCard(screen *ebiten.Image, panelEntity ecs.EntityID, panel *components.RewardPanelComponent) {
	cardImage := rprs.resourceManager.GetImageByID("IMAGE_SEEDPACKET_LARGER")
	if cardImage == nil {
		return
	}

	// 计算缩放因子
	cardWidth, cardHeight := cardImage.Bounds().Dx(), cardImage.Bounds().Dy()
	scaleFactor := panel.CardScale * config.RewardPanelCardScale

	// 计算卡片左上角位置（PlantCardRenderer 需要左上角坐标）
	cardX := panel.CardX - float64(cardWidth)*scaleFactor/2.0
	cardY := panel.CardY - float64(cardHeight)*scaleFactor/2.0

	// Story 8.4: 使用 PlantCardRenderer 渲染背景框和阳光数字
	var sunFontSource *text.GoTextFaceSource
	var sunFontSize float64
	if rprs.sunCostFont != nil {
		sunFontSource = rprs.sunCostFont.Source
		sunFontSize = rprs.sunCostFont.Size
	}

	rprs.cardRenderer.Render(utils.PlantCardRenderOptions{
		Screen:          screen,
		X:               cardX,
		Y:               cardY,
		BackgroundImage: cardImage,
		SunCost:         panel.SunCost,
		SunFont:         sunFontSource,
		SunFontSize:     sunFontSize,
		SunTextOffsetY:  config.RewardPanelSunCostOffsetY * scaleFactor,
		SunTextColor:    config.RewardPanelSunCostColor,
		CardScale:       scaleFactor,
		Alpha:           panel.FadeAlpha,
	})

	// 保留 Reanim 渲染逻辑：渲染植物 Reanim 实体（叠加在卡片框上）
	plantEntityID, exists := rprs.plantDisplayEntities[panelEntity]
	if exists && plantEntityID != 0 {
		// 更新植物实体的位置（跟随卡片位置和偏移）
		if posComp, ok := ecs.GetComponent[*components.PositionComponent](rprs.entityManager, plantEntityID); ok {
			posComp.X = panel.CardX
			posComp.Y = panel.CardY + config.RewardPanelPlantIconOffsetY*scaleFactor
		}

		// 渲染植物 Reanim
		rprs.renderPlantReanimEntity(screen, plantEntityID, scaleFactor, panel.FadeAlpha)
	}
}

// loadPlantIcon 根据植物ID加载植物图标（头部图片）。
func (rprs *RewardPanelRenderSystem) loadPlantIcon(plantID string) *ebiten.Image {
	// 根据不同植物加载对应的头部图片
	var iconID string
	switch plantID {
	case "sunflower":
		iconID = "IMAGE_REANIM_SUNFLOWER_HEAD"
	case "peashooter":
		iconID = "IMAGE_REANIM_PEASHOOTER_HEAD"
	case "cherrybomb":
		iconID = "IMAGE_REANIM_CHERRYBOMB"
	case "wallnut":
		iconID = "IMAGE_REANIM_WALLNUT_BODY"
	default:
		log.Printf("[RewardPanelRenderSystem] Unknown plant ID: %s", plantID)
		return nil
	}

	icon := rprs.resourceManager.GetImageByID(iconID)
	if icon == nil {
		log.Printf("[RewardPanelRenderSystem] Failed to load plant icon: %s", iconID)
	}
	return icon
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
		nameX := offsetX + bgWidth/2 // 背景中心X
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
			op.PrimaryAlign = text.AlignCenter   // 水平居中
			op.SecondaryAlign = text.AlignStart  // 垂直从上开始
			op.ColorScale.ScaleWithColor(config.RewardPanelPlantNameColor) // 使用配置的金黄色
			op.ColorScale.ScaleAlpha(float32(panel.FadeAlpha))
			text.Draw(screen, panel.PlantName, rprs.plantInfoFont, op)
		}

		// 绘制植物描述（使用配置中的颜色和位置，使用植物信息字体）
		descX := offsetX + bgWidth/2 // 背景中心X
		descY := offsetY + bgHeight*config.RewardPanelDescriptionY // 使用配置的Y位置

		if panel.PlantDescription != "" {
			op := &text.DrawOptions{}
			op.GeoM.Translate(descX, descY)
			op.PrimaryAlign = text.AlignCenter   // 水平居中
			op.SecondaryAlign = text.AlignStart  // 垂直从上开始
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
	op.GeoM.Translate(hintX-70, hintY) // 居中偏移
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
		textOp.PrimaryAlign = text.AlignCenter  // 水平居中
		textOp.SecondaryAlign = text.AlignCenter // 垂直居中
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
		op.GeoM.Translate(buttonX-30, buttonY-10) // 居中偏移
		op.ColorScale.ScaleWithColor(color.RGBA{255, 255, 255, 255}) // 白色文字
		op.ColorScale.ScaleAlpha(float32(alpha))
		text.Draw(screen, buttonText, rprs.titleFont, op)
	}
}

// ensurePlantDisplayEntity 确保植物显示实体存在，如果不存在则创建。
func (rprs *RewardPanelRenderSystem) ensurePlantDisplayEntity(panelEntity ecs.EntityID, panel *components.RewardPanelComponent) {
	// 检查是否已经有显示实体
	if _, exists := rprs.plantDisplayEntities[panelEntity]; exists {
		return
	}

	// 创建植物显示实体
	if panel.PlantID == "" {
		return
	}

	plantEntity := rprs.createPlantDisplayEntity(panel.PlantID, panel.CardX, panel.CardY)
	if plantEntity != 0 {
		rprs.plantDisplayEntities[panelEntity] = plantEntity
		log.Printf("[RewardPanelRenderSystem] Created plant display entity %d for panel %d (plant: %s)",
			plantEntity, panelEntity, panel.PlantID)
	}
}

// createPlantDisplayEntity 创建用于显示的植物 Reanim 实体。
func (rprs *RewardPanelRenderSystem) createPlantDisplayEntity(plantID string, x, y float64) ecs.EntityID {
	// 根据植物ID获取 Reanim 名称
	reanimName := ""
	animName := ""

	switch plantID {
	case "sunflower":
		reanimName = "SunFlower"
		animName = "anim_idle"
	case "peashooter":
		reanimName = "PeaShooter"
		animName = "anim_full_idle" // 豌豆射手需要完整待机动画才显示头部
	case "cherrybomb":
		reanimName = "CherryBomb"
		animName = "anim_idle"
	case "wallnut":
		reanimName = "Wallnut"
		animName = "anim_idle"
	default:
		log.Printf("[RewardPanelRenderSystem] Unknown plant ID: %s", plantID)
		return 0
	}

	// 加载 Reanim 数据
	reanimXML := rprs.resourceManager.GetReanimXML(reanimName)
	partImages := rprs.resourceManager.GetReanimPartImages(reanimName)

	if reanimXML == nil || partImages == nil {
		log.Printf("[RewardPanelRenderSystem] Failed to load Reanim resources for %s", reanimName)
		return 0
	}

	// 创建实体
	entityID := rprs.entityManager.CreateEntity()

	// 添加位置组件
	ecs.AddComponent(rprs.entityManager, entityID, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// 添加 Reanim 组件
	ecs.AddComponent(rprs.entityManager, entityID, &components.ReanimComponent{
		Reanim:     reanimXML,
		PartImages: partImages,
	})

	// 播放动画
	if err := rprs.reanimSystem.PlayAnimation(entityID, animName); err != nil {
		log.Printf("[RewardPanelRenderSystem] Failed to play animation %s: %v", animName, err)
		rprs.entityManager.DestroyEntity(entityID)
		return 0
	}

	return entityID
}

// renderPlantReanimEntity 渲染植物 Reanim 实体（简化版）。
// 参数:
//   - screen: 绘制目标屏幕
//   - id: 实体ID
//   - scale: 缩放比例
//   - alpha: 透明度 (0.0-1.0)
func (rprs *RewardPanelRenderSystem) renderPlantReanimEntity(screen *ebiten.Image, id ecs.EntityID, scale float64, alpha float64) {
	// 获取组件
	pos, hasPosComp := ecs.GetComponent[*components.PositionComponent](rprs.entityManager, id)
	reanim, hasReanimComp := ecs.GetComponent[*components.ReanimComponent](rprs.entityManager, id)

	if !hasPosComp || !hasReanimComp {
		return
	}

	// 如果没有当前动画或动画轨道，跳过
	if reanim.CurrentAnim == "" || len(reanim.AnimTracks) == 0 {
		return
	}

	// 将逻辑帧映射到物理帧索引
	physicalIndex := rprs.findPhysicalFrameIndex(reanim, reanim.CurrentFrame)
	if physicalIndex < 0 {
		return
	}

	// 计算绘制原点（应用 CenterOffset）
	// UI 坐标不需要摄像机转换
	screenX := pos.X - reanim.CenterOffsetX*scale
	screenY := pos.Y - reanim.CenterOffsetY*scale

	// 遍历所有轨道并渲染部件
	for _, track := range reanim.AnimTracks {
		if len(track.Frames) == 0 {
			continue
		}

		// 获取当前帧的变换
		if physicalIndex >= len(track.Frames) {
			continue
		}

		transform := track.Frames[physicalIndex]

		// 获取部件图片
		partImage, exists := reanim.PartImages[track.Name]
		if !exists || partImage == nil {
			continue
		}

		// 构建变换矩阵
		op := &ebiten.DrawImageOptions{}

		// 1. 锚点偏移（图片左上角移动到锚点）
		imgWidth := float64(partImage.Bounds().Dx())
		imgHeight := float64(partImage.Bounds().Dy())
		op.GeoM.Translate(-imgWidth/2, -imgHeight/2) // 图片中心作为锚点

		// 2. 应用部件变换
		if transform.ScaleX != nil && transform.ScaleY != nil {
			op.GeoM.Scale(*transform.ScaleX, *transform.ScaleY)
		}
		// Note: Reanim Frame 没有 RotZ 字段，旋转通过 SkewX/SkewY 实现
		if transform.X != nil && transform.Y != nil {
			op.GeoM.Translate(float64(*transform.X), float64(*transform.Y))
		}

		// 3. 应用额外的缩放（卡片缩放）
		op.GeoM.Scale(scale, scale)

		// 4. 移动到最终位置
		op.GeoM.Translate(screenX, screenY)

		// 应用透明度
		op.ColorScale.ScaleAlpha(float32(alpha))

		// 渲染部件
		screen.DrawImage(partImage, op)
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
