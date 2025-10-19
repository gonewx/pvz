package systems

import (
	"image/color"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// RewardPanelRenderSystem 负责渲染奖励面板 UI。
// 包括背景、植物卡片、文本信息等元素的绘制。
type RewardPanelRenderSystem struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager
	screenWidth     float64
	screenHeight    float64
}

// NewRewardPanelRenderSystem 创建奖励面板渲染系统。
func NewRewardPanelRenderSystem(em *ecs.EntityManager, gs *game.GameState, rm *game.ResourceManager) *RewardPanelRenderSystem {
	return &RewardPanelRenderSystem{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
		screenWidth:     800, // TODO: 从配置获取
		screenHeight:    600,
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

	// 5. 绘制提示文本："点击任意位置继续"
	rprs.drawHint(screen, panel.FadeAlpha)
}

// drawBackground 绘制奖励背景。
func (rprs *RewardPanelRenderSystem) drawBackground(screen *ebiten.Image, alpha float64) {
	bgImage := rprs.resourceManager.GetImageByID("IMAGE_AWARD_SCREEN_BACK")
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
	// 从 LawnStrings 获取标题文本
	titleText := "你得到了一株新植物！" // 默认文本
	if rprs.gameState.LawnStrings != nil {
		titleText = rprs.gameState.LawnStrings.GetString("NEW_PLANT")
		if titleText == "" {
			titleText = "你得到了一株新植物！"
		}
	}

	// TODO: Story 8.3 后续优化 - 使用 BitmapFont 或 TTF 字体渲染标题
	// 当前使用 ebitenutil.DebugPrint 作为占位符
	_ = titleText // 暂时避免未使用警告

	// 绘制简单的占位文本（调试用）
	if alpha > 0.5 {
		ebitenutil.DebugPrintAt(screen, titleText, int(rprs.screenWidth/2-100), int(rprs.screenHeight*0.2))
	}
}

// drawPlantCard 绘制植物卡片。
func (rprs *RewardPanelRenderSystem) drawPlantCard(screen *ebiten.Image, panel *components.RewardPanelComponent) {
	cardImage := rprs.resourceManager.GetImageByID("IMAGE_SEED_PACKET_LARGER")
	if cardImage == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}

	// 应用缩放动画
	cardWidth, cardHeight := cardImage.Bounds().Dx(), cardImage.Bounds().Dy()
	op.GeoM.Translate(-float64(cardWidth)/2, -float64(cardHeight)/2) // 中心对齐
	op.GeoM.Scale(panel.CardScale, panel.CardScale)
	op.GeoM.Translate(panel.CardX, panel.CardY)

	// 应用透明度
	op.ColorScale.ScaleAlpha(float32(panel.FadeAlpha))

	screen.DrawImage(cardImage, op)
}

// drawPlantInfo 绘制植物名称和描述。
func (rprs *RewardPanelRenderSystem) drawPlantInfo(screen *ebiten.Image, panel *components.RewardPanelComponent) {
	// TODO: Story 8.3 后续优化 - 使用 BitmapFont 或 TTF 字体渲染
	// 当前使用 ebitenutil.DebugPrint 作为占位符

	if panel.FadeAlpha > 0.5 {
		// 绘制植物名称
		nameX := int(rprs.screenWidth / 2 - 50)
		nameY := int(rprs.screenHeight * 0.55)
		ebitenutil.DebugPrintAt(screen, panel.PlantName, nameX, nameY)

		// 绘制植物描述
		descX := int(rprs.screenWidth / 2 - 100)
		descY := int(rprs.screenHeight * 0.7)
		ebitenutil.DebugPrintAt(screen, panel.PlantDescription, descX, descY)
	}
}

// drawHint 绘制提示文本。
func (rprs *RewardPanelRenderSystem) drawHint(screen *ebiten.Image, alpha float64) {
	hintText := "点击任意位置继续" // 默认文本
	if rprs.gameState.LawnStrings != nil {
		hintText = rprs.gameState.LawnStrings.GetString("CLICK_TO_CONTINUE")
		if hintText == "" {
			hintText = "点击任意位置继续"
		}
	}

	// TODO: Story 8.3 后续优化 - 使用 BitmapFont 或 TTF 字体渲染
	// 当前使用 ebitenutil.DebugPrint 作为占位符

	if alpha > 0.5 {
		hintX := int(rprs.screenWidth / 2 - 60)
		hintY := int(rprs.screenHeight * 0.9)
		ebitenutil.DebugPrintAt(screen, hintText, hintX, hintY)
	}
}
