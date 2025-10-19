package systems

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

// RewardPanelRenderSystem 负责渲染奖励面板 UI。
// 包括背景、植物卡片、文本信息等元素的绘制。
type RewardPanelRenderSystem struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager
	screenWidth     float64
	screenHeight    float64
	titleFont       *text.GoTextFace     // 标题字体
	plantInfoFont   *text.GoTextFace     // 植物名称和描述字体
	sunCostFont     *text.GoTextFace     // 阳光值字体
	buttonFont      *text.GoTextFace     // 按钮字体
}

// NewRewardPanelRenderSystem 创建奖励面板渲染系统。
func NewRewardPanelRenderSystem(em *ecs.EntityManager, gs *game.GameState, rm *game.ResourceManager) *RewardPanelRenderSystem {
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
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
		screenWidth:     800, // TODO: 从配置获取
		screenHeight:    600,
		titleFont:       titleFont,
		plantInfoFont:   plantInfoFont,
		sunCostFont:     sunCostFont,
		buttonFont:      buttonFont,
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

		// 绘制奖励面板
		rprs.drawPanel(screen, panelComp)
	}
}

// drawPanel 绘制奖励面板内容。
func (rprs *RewardPanelRenderSystem) drawPanel(screen *ebiten.Image, panel *components.RewardPanelComponent) {
	// 1. 绘制背景（AwardScreen_Back.jpg）
	rprs.drawBackground(screen, panel.FadeAlpha)

	// 2. 绘制标题文本："你得到了一株新植物！"
	rprs.drawTitle(screen, panel.FadeAlpha)

	// 3. 绘制植物卡片（SeedPacket_Larger.png）
	rprs.drawPlantCard(screen, panel)

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

// drawPlantCard 绘制植物卡片（背景框 + 植物图标）。
func (rprs *RewardPanelRenderSystem) drawPlantCard(screen *ebiten.Image, panel *components.RewardPanelComponent) {
	// 1. 绘制卡片背景框（使用配置中的缩放比例）
	cardImage := rprs.resourceManager.GetImageByID("IMAGE_SEEDPACKET_LARGER")
	if cardImage == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}

	// 应用缩放动画（使用配置中的缩放比例）
	cardWidth, cardHeight := cardImage.Bounds().Dx(), cardImage.Bounds().Dy()
	scaleFactor := panel.CardScale * config.RewardPanelCardScale // 使用配置的缩放比例
	op.GeoM.Translate(-float64(cardWidth)/2, -float64(cardHeight)/2) // 中心对齐
	op.GeoM.Scale(scaleFactor, scaleFactor)
	op.GeoM.Translate(panel.CardX, panel.CardY)

	// 应用透明度
	op.ColorScale.ScaleAlpha(float32(panel.FadeAlpha))

	screen.DrawImage(cardImage, op)

	// 2. 绘制植物图标（叠加在卡片框上，使用配置的Y偏移）
	if panel.PlantIconTexture != nil {
		iconOp := &ebiten.DrawImageOptions{}

		// 图标居中对齐到卡片框
		iconWidth, iconHeight := panel.PlantIconTexture.Bounds().Dx(), panel.PlantIconTexture.Bounds().Dy()
		iconOp.GeoM.Translate(-float64(iconWidth)/2, -float64(iconHeight)/2)                    // 中心对齐
		iconOp.GeoM.Scale(scaleFactor, scaleFactor)                                              // 与卡片框同步缩放
		iconOp.GeoM.Translate(panel.CardX, panel.CardY+config.RewardPanelPlantIconOffsetY*scaleFactor) // 使用配置的Y偏移

		// 应用透明度
		iconOp.ColorScale.ScaleAlpha(float32(panel.FadeAlpha))

		screen.DrawImage(panel.PlantIconTexture, iconOp)
	}

	// 3. 绘制阳光值（在卡片底部）
	rprs.drawSunCost(screen, panel, scaleFactor)
}

// drawSunCost 绘制阳光值（在植物卡片底部）。
func (rprs *RewardPanelRenderSystem) drawSunCost(screen *ebiten.Image, panel *components.RewardPanelComponent, scaleFactor float64) {
	if panel.FadeAlpha < 0.5 || rprs.sunCostFont == nil {
		return
	}

	// 阳光值文本
	sunCostText := fmt.Sprintf("%d", panel.SunCost)

	// 计算位置（使用配置中的偏移量）
	sunCostX := panel.CardX
	sunCostY := panel.CardY + config.RewardPanelSunCostOffsetY*scaleFactor // 使用配置的偏移量

	op := &text.DrawOptions{}
	op.GeoM.Translate(sunCostX, sunCostY)
	op.PrimaryAlign = text.AlignCenter   // 水平居中
	op.SecondaryAlign = text.AlignStart  // 垂直从上开始
	op.ColorScale.ScaleWithColor(config.RewardPanelSunCostColor) // 使用配置的黑色
	op.ColorScale.ScaleAlpha(float32(panel.FadeAlpha))
	text.Draw(screen, sunCostText, rprs.sunCostFont, op)
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
