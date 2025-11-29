package systems

import (
	"fmt"
	"image"
	"image/color"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// LevelProgressBarRenderSystem 关卡进度条渲染系统
type LevelProgressBarRenderSystem struct {
	entityManager *ecs.EntityManager
	font          *text.GoTextFace // 关卡文本字体
}

// NewLevelProgressBarRenderSystem 创建进度条渲染系统
func NewLevelProgressBarRenderSystem(em *ecs.EntityManager, font *text.GoTextFace) *LevelProgressBarRenderSystem {
	return &LevelProgressBarRenderSystem{
		entityManager: em,
		font:          font,
	}
}

// Draw 渲染所有进度条实体
func (s *LevelProgressBarRenderSystem) Draw(screen *ebiten.Image) {
	// 检查游戏是否冻结（僵尸获胜流程期间）
	// 冻结时隐藏进度条
	freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](s.entityManager)
	if len(freezeEntities) > 0 {
		return // 游戏冻结，不渲染进度条
	}

	// 查询所有拥有 LevelProgressBarComponent 的实体
	entities := ecs.GetEntitiesWith1[*components.LevelProgressBarComponent](s.entityManager)

	if config.DebugProgressBar && len(entities) == 0 {
		// 调试：如果没有找到进度条实体
		ebitenutil.DebugPrint(screen, "DEBUG: No progress bar entities found!")
	}

	for _, entityID := range entities {
		progressBar, ok := ecs.GetComponent[*components.LevelProgressBarComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 调试：显示当前状态
		if config.DebugProgressBar {
			debugMsg := fmt.Sprintf("ProgressBar: pos(%.0f,%.0f) textOnly=%v text=%s",
				progressBar.X, progressBar.Y, progressBar.ShowLevelTextOnly, progressBar.LevelText)
			ebitenutil.DebugPrintAt(screen, debugMsg, 10, 100)
		}

		// 根据显示模式选择渲染内容
		if progressBar.ShowLevelTextOnly {
			// 开场前：只显示关卡文本（不显示进度条背景）
			s.drawLevelText(screen, progressBar)
		} else {
			// 进攻中：显示完整进度条（背景 + 填充 + 僵尸头 + 旗帜 + 文字）
			s.drawFullProgressBar(screen, progressBar)
		}

		// 调试模式：绘制边界框
		if config.DebugProgressBar {
			s.drawDebugBounds(screen, progressBar)
		}
	}
}

// drawFullProgressBar 绘制完整进度条（背景 + 填充 + 僵尸头 + 旗帜 + 文本）
func (s *LevelProgressBarRenderSystem) drawFullProgressBar(screen *ebiten.Image, progressBar *components.LevelProgressBarComponent) {
	// 计算右对齐位置
	bgWidth := config.ProgressBarWidth
	progressBarX := config.ScreenWidth - float64(bgWidth) - config.ProgressBarRightMargin
	progressBarY := config.ScreenHeight - float64(config.ProgressBarHeight) - config.ProgressBarBottomMargin

	// 更新组件位置（用于其他渲染方法）
	progressBar.X = progressBarX
	progressBar.Y = progressBarY

	// 1. 绘制背景框（FlagMeter.png 的第1行：空背景）
	if progressBar.BackgroundImage != nil {
		bgBounds := progressBar.BackgroundImage.Bounds()
		bgWidthPx := bgBounds.Dx()
		bgHeight := bgBounds.Dy() / 2 // 2行图，取上半部分

		// 裁剪第1行（空背景）
		emptyBg := progressBar.BackgroundImage.SubImage(
			image.Rect(0, 0, bgWidthPx, bgHeight),
		).(*ebiten.Image)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(progressBar.X, progressBar.Y)
		screen.DrawImage(emptyBg, op)

		// 2. 根据进度裁剪并绘制第2行（绿色填充，从右到左）
		fillWidth := int(float64(bgWidth) * progressBar.ProgressPercent)
		if fillWidth > 0 {
			// 从第2行的右侧开始裁剪（从右到左填充）
			fillStartX := bgWidth - fillWidth
			filledBg := progressBar.BackgroundImage.SubImage(
				image.Rect(fillStartX, bgHeight, bgWidth, bgHeight*2),
			).(*ebiten.Image)

			// 绘制到对应的右侧位置
			fillOp := &ebiten.DrawImageOptions{}
			fillOp.GeoM.Translate(progressBar.X+float64(fillStartX), progressBar.Y)
			screen.DrawImage(filledBg, fillOp)
		}
	}

	// 3. 绘制 FlagMeterLevelProgress.png（绿色装饰条）
	s.drawLevelProgressDecoration(screen, progressBar)

	// 4. 绘制旗帜图标
	s.drawFlags(screen, progressBar)

	// 5. 绘制僵尸头（跟随进度）
	s.drawZombieHead(screen, progressBar)

	// 6. 绘制关卡文本
	s.drawLevelText(screen, progressBar)
}

// drawLevelProgressDecoration 绘制 FlagMeterLevelProgress.png（绿色装饰条）
// 垂直对齐：图片的垂直中线对齐背景框的下边沿
// 水平对齐：居中在背景框内
func (s *LevelProgressBarRenderSystem) drawLevelProgressDecoration(screen *ebiten.Image, progressBar *components.LevelProgressBarComponent) {
	if progressBar.ProgressBarImage == nil {
		return
	}

	// 获取背景框和装饰条的尺寸
	bgHeight := config.ProgressBarBackgroundHeight // FlagMeter.png 单行高度（54/2）
	decorBounds := progressBar.ProgressBarImage.Bounds()
	decorWidth := float64(decorBounds.Dx())  // 86
	decorHeight := float64(decorBounds.Dy()) // 11

	// 背景框的尺寸
	bgWidth := config.ProgressBarBackgroundWidth

	// 计算位置
	// 水平居中：(背景宽度 - 装饰条宽度) / 2
	decorX := progressBar.X + (bgWidth-decorWidth)/2

	// 垂直对齐：装饰条的垂直中线对齐背景框的下边沿 + 手工调整偏移
	// 背景框下边沿 Y = progressBar.Y + bgHeight
	// 装饰条垂直中线 = decorHeight / 2
	decorY := progressBar.Y + bgHeight - decorHeight/2 + config.LevelProgressDecorationOffsetY

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(decorX, decorY)
	screen.DrawImage(progressBar.ProgressBarImage, op)
}

// drawZombieHead 绘制僵尸头图标（跟随进度）
func (s *LevelProgressBarRenderSystem) drawZombieHead(screen *ebiten.Image, progressBar *components.LevelProgressBarComponent) {
	if progressBar.PartsImage == nil {
		return
	}

	// 获取精灵图尺寸
	bounds := progressBar.PartsImage.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	// 计算每个部分的宽度（3个等宽部分）
	partWidth := imgWidth / config.PartsImageColumns

	// 从精灵图裁剪僵尸头图标（第1列，索引0）
	zombieHeadRect := image.Rect(
		0,         // X起始位置：第1列
		0,         // Y起始位置
		partWidth, // X结束位置
		imgHeight, // Y结束位置
	)
	zombieHeadImage := progressBar.PartsImage.SubImage(zombieHeadRect).(*ebiten.Image)

	// 有效进度区域配置
	// 背景框 158 像素，有效区域 150 像素，左边距 4 像素
	leftMargin := config.ProgressBarLeftMargin         // 4 像素
	effectiveWidth := config.ProgressBarEffectiveWidth // 150 像素

	// 计算僵尸头位置（从右边开始，随进度向左移动）
	// 进度0% = 最右边，进度100% = 最左边
	headX := progressBar.X + leftMargin + effectiveWidth*(1.0-progressBar.ProgressPercent) + config.ZombieHeadOffsetX - float64(partWidth)/2.0
	headY := progressBar.Y + config.ZombieHeadOffsetY

	// 绘制僵尸头
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(headX, headY)
	screen.DrawImage(zombieHeadImage, op)
}

// drawFlags 绘制旗帜图标（旗杆 + 旗帜）
func (s *LevelProgressBarRenderSystem) drawFlags(screen *ebiten.Image, progressBar *components.LevelProgressBarComponent) {
	if progressBar.PartsImage == nil || len(progressBar.FlagPositions) == 0 {
		return
	}

	// 获取精灵图尺寸
	bounds := progressBar.PartsImage.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	// 计算每个部分的宽度（3个等宽部分：僵尸头、旗杆、旗帜）
	partWidth := imgWidth / config.PartsImageColumns

	// 从精灵图裁剪旗杆图标（第2列，索引1）
	poleRect := image.Rect(
		partWidth,   // X起始位置：第2列
		0,           // Y起始位置
		partWidth*2, // X结束位置
		imgHeight,   // Y结束位置
	)
	poleImage := progressBar.PartsImage.SubImage(poleRect).(*ebiten.Image)

	// 从精灵图裁剪旗帜图标（第3列，索引2）
	flagRect := image.Rect(
		partWidth*2, // X起始位置：第3列
		0,           // Y起始位置
		imgWidth,    // X结束位置（图片右边界）
		imgHeight,   // Y结束位置
	)
	flagImage := progressBar.PartsImage.SubImage(flagRect).(*ebiten.Image)

	// 有效进度区域配置
	// 背景框 158 像素，有效区域 150 像素，左边距 4 像素
	leftMargin := config.ProgressBarLeftMargin         // 4 像素
	effectiveWidth := config.ProgressBarEffectiveWidth // 150 像素

	// 在每个旗帜位置绘制旗杆和旗帜
	for _, flagPos := range progressBar.FlagPositions {
		// 旗帜左边缘对齐到红字波段右边缘
		// 8% 位置 = 4 + 150*0.08 = 16 像素，旗帜显示在红字波段右侧
		flagX := progressBar.X + leftMargin + effectiveWidth*flagPos

		// 先绘制旗杆（独立 Y 偏移）
		poleOp := &ebiten.DrawImageOptions{}
		poleOp.GeoM.Translate(flagX, progressBar.Y+config.FlagPoleOffsetY)
		screen.DrawImage(poleImage, poleOp)

		// 再绘制旗帜（独立 Y 偏移）
		flagOp := &ebiten.DrawImageOptions{}
		flagOp.GeoM.Translate(flagX, progressBar.Y+config.FlagIconOffsetY)
		screen.DrawImage(flagImage, flagOp)
	}
}

// drawLevelText 绘制关卡文本（黑色描边，橙黄色文字，右对齐）
func (s *LevelProgressBarRenderSystem) drawLevelText(screen *ebiten.Image, progressBar *components.LevelProgressBarComponent) {
	if s.font == nil || progressBar.LevelText == "" {
		return
	}

	// 计算文本宽度
	textWidth, _ := text.Measure(progressBar.LevelText, s.font, 0)

	// 计算文本位置（右对齐）
	var textX, textY float64
	if progressBar.ShowLevelTextOnly {
		// 进度条隐藏时：文本直接右对齐到屏幕边缘
		textX = config.ScreenWidth - textWidth - config.LevelTextRightMargin
		textY = 600 - float64(config.ProgressBarHeight) - config.ProgressBarBottomMargin + config.LevelTextOffsetY
	} else {
		// 进度条显示时：文本右边缘在进度条左边缘的左侧
		// 文本右边缘位置 = 进度条左边缘 - 间距
		// textX = (进度条左边缘 - 间距) - 文本宽度
		textX = progressBar.X - config.LevelTextOffsetX - textWidth
		textY = progressBar.Y + config.LevelTextOffsetY
	}

	// 描边宽度（像素）
	outlineWidth := 2.0

	// 1. 先绘制黑色描边（8个方向）
	outlineColor := color.RGBA{0, 0, 0, 255} // 不透明黑色描边
	directions := []struct{ dx, dy float64 }{
		{-outlineWidth, -outlineWidth}, // 左上
		{0, -outlineWidth},             // 上
		{outlineWidth, -outlineWidth},  // 右上
		{-outlineWidth, 0},             // 左
		{outlineWidth, 0},              // 右
		{-outlineWidth, outlineWidth},  // 左下
		{0, outlineWidth},              // 下
		{outlineWidth, outlineWidth},   // 右下
	}

	for _, dir := range directions {
		outlineOp := &text.DrawOptions{}
		outlineOp.GeoM.Translate(textX+dir.dx, textY+dir.dy)
		outlineOp.ColorScale.ScaleWithColor(outlineColor)
		text.Draw(screen, progressBar.LevelText, s.font, outlineOp)
	}

	// 2. 再绘制橙黄色文本（主体）
	op := &text.DrawOptions{}
	op.GeoM.Translate(textX, textY)
	op.ColorScale.ScaleWithColor(color.RGBA{255, 200, 0, 255}) // 橙黄色
	text.Draw(screen, progressBar.LevelText, s.font, op)
}

// drawDebugBounds 绘制调试边界框（开发调试用）
func (s *LevelProgressBarRenderSystem) drawDebugBounds(screen *ebiten.Image, progressBar *components.LevelProgressBarComponent) {
	// 绘制进度条背景框边界
	if progressBar.BackgroundImage != nil {
		bounds := progressBar.BackgroundImage.Bounds()
		ebitenutil.DrawRect(screen,
			progressBar.X, progressBar.Y,
			float64(bounds.Dx()), float64(bounds.Dy()),
			color.RGBA{R: 255, G: 0, B: 0, A: 100}, // 红色半透明
		)
	}

	// 绘制进度条填充区域边界
	ebitenutil.DrawRect(screen,
		progressBar.X+config.ProgressBarStartOffsetX,
		progressBar.Y+config.ProgressBarStartOffsetY,
		float64(config.ProgressBarWidth),
		float64(config.ProgressBarHeight),
		color.RGBA{R: 0, G: 255, B: 0, A: 100}, // 绿色半透明
	)

	// 绘制调试文本
	debugText := fmt.Sprintf("Progress: %.2f%% (%d/%d)",
		progressBar.ProgressPercent*100,
		progressBar.KilledZombies,
		progressBar.TotalZombies,
	)
	ebitenutil.DebugPrintAt(screen, debugText, int(progressBar.X), int(progressBar.Y)-20)
}
