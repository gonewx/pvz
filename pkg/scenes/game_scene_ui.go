package scenes

import (
	"fmt"
	"image/color"
	"math"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// getSunFlashColor 计算闪烁颜色（方波）
// Story 10.8: 阳光不足反馈的闪烁效果
//
// 参数:
//   - timer: 当前闪烁时间（从 duration 倒计时到 0）
//   - cycle: 闪烁周期（秒）
//
// 返回:
//   - color.Color: 红色或黑色
func getSunFlashColor(timer float64, cycle float64) color.Color {
	// 方波逻辑：根据时间模除周期，确定当前相位
	phase := math.Mod(timer, cycle)

	if phase < cycle/2 {
		return color.RGBA{R: 255, G: 0, B: 0, A: 255} // 红色 (#FF0000)
	} else {
		return color.RGBA{R: 0, G: 0, B: 0, A: 255} // 黑色 (#000000)
	}
}

// drawSeedBank renders the plant selection bar at the top left of the screen.
// If the seed bank image is not loaded, it draws a simple rectangle as fallback.
// Story 19.5: 保龄球模式使用传送带，不显示植物选择栏
// 滑入动画：从上向下滑入，类似传送带入场效果
func (s *GameScene) drawSeedBank(screen *ebiten.Image) {
	// Story 19.5: 保龄球模式（initialSun == 0）不显示植物选择栏
	if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.InitialSun == 0 {
		return
	}

	// Story 8.8: 游戏冻结时隐藏植物选择栏
	freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](s.entityManager)
	if len(freezeEntities) > 0 {
		return
	}

	// 获取当前滑入动画 Y 位置
	currentY := s.getSeedBankCurrentY()

	if s.seedBank != nil {
		// Draw the seed bank image at the top left corner
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(config.SeedBankX, currentY)
		screen.DrawImage(s.seedBank, op)
	} else {
		// Fallback: Draw a dark brown rectangle
		ebitenutil.DrawRect(screen,
			config.SeedBankX, currentY,
			config.SeedBankWidth, config.SeedBankHeight,
			color.RGBA{R: 101, G: 67, B: 33, A: 255}) // Dark brown
	}
}

// drawSunCounter renders the sun counter value on the seed bank.
// Note: The sun counter background and gold frame are already part of the bar5.png image,
// so we don't need to draw them separately. This method displays the sun count number.
// The text is horizontally centered to accommodate dynamic value lengths (e.g., 50, 150, 9990).
// Story 10.8: 添加阳光不足时的闪烁效果（红黑闪烁）
// Story 19.10: 保龄球关卡（initialSun == 0）不显示阳光槽
// 滑入动画：与植物选择栏同步滑入
func (s *GameScene) drawSunCounter(screen *ebiten.Image) {
	// Story 19.10: 保龄球关卡（initialSun == 0）不显示阳光槽
	if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.InitialSun == 0 {
		return
	}

	// Story 8.8: 游戏冻结时隐藏阳光计数器
	freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](s.entityManager)
	if len(freezeEntities) > 0 {
		return
	}

	// 获取当前滑入动画 Y 位置
	currentSeedBankY := s.getSeedBankCurrentY()

	// Get current sun value from game state
	sunValue := s.gameState.GetSun()
	sunText := fmt.Sprintf("%d", sunValue)

	if s.sunCounterFont != nil {
		// Measure text width for centering
		textWidth, _ := text.Measure(sunText, s.sunCounterFont, 0)

		// Calculate centered position
		// Base position is relative to SeedBank (使用动态 Y 位置)
		centerX := float64(config.SeedBankX + config.SunCounterOffsetX)
		centerY := currentSeedBankY + float64(config.SunCounterOffsetY)

		// Adjust X to center the text horizontally
		sunDisplayX := centerX - textWidth/2
		sunDisplayY := centerY

		// Use custom font with color
		op := &text.DrawOptions{}
		op.GeoM.Translate(sunDisplayX, sunDisplayY)

		// Story 10.8: 确定文本颜色（闪烁或默认黑色）
		var textColor color.Color
		if s.gameState.SunFlashTimer > 0 {
			// 正在闪烁：使用闪烁颜色（红 ↔ 黑）
			textColor = getSunFlashColor(s.gameState.SunFlashTimer, s.gameState.SunFlashCycle)
		} else {
			// 默认颜色：黑色（更好的可见性在米黄色背景上）
			textColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}
		}

		// Set text color
		op.ColorScale.ScaleWithColor(textColor)

		text.Draw(screen, sunText, s.sunCounterFont, op)
	} else {
		// Fallback: Use debug text if font failed to load
		// Note: Debug text doesn't support centering easily
		sunDisplayX := config.SeedBankX + config.SunCounterOffsetX
		sunDisplayY := int(currentSeedBankY) + config.SunCounterOffsetY
		ebitenutil.DebugPrintAt(screen, sunText, sunDisplayX, sunDisplayY)
	}
}

// drawShovel renders the shovel slot and icon at the right side of the seed bank.
// The shovel will be used in future stories for removing plants.
// Story 8.5: 1-1关（教学关卡）不显示铲子
// Story 8.6: 检查铲子是否已解锁（1-4关完成后才解锁）
// Story 19.5: 保龄球模式使用固定位置，不依赖选择栏
// Story 19.x QA: 铲子教学关卡（有预设植物）强制显示铲子
// 铲子位置紧挨选择栏右侧，与选择栏上对齐
// 滑入动画：与植物选择栏同步滑入（非保龄球模式）
func (s *GameScene) drawShovel(screen *ebiten.Image) {
	// 教学关卡不显示铲子（玩家还不需要学习移除植物）
	// 但是：铲子教学关卡（Level 1-5，有预设植物）需要显示铲子
	if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.OpeningType == "tutorial" {
		// 铲子教学关卡：有预设植物的关卡应该显示铲子
		if len(s.gameState.CurrentLevel.PresetPlants) == 0 {
			return
		}
	}

	// Story 8.6: 检查铲子是否已解锁
	// 铲子在完成 1-4 关卡后解锁
	// 例外：铲子教学关卡（Level 1-5）强制显示铲子（用于教学）
	isShovelTutorialLevel := s.gameState.CurrentLevel != nil && len(s.gameState.CurrentLevel.PresetPlants) > 0
	if !s.gameState.IsToolUnlocked("shovel") && !isShovelTutorialLevel {
		return
	}

	// 计算铲子位置
	var shovelX float64
	var shovelY float64
	// Story 19.5: 保龄球模式（initialSun == 0）使用相对于菜单按钮的位置
	// Story 19.x QA: 铲子位置相对于菜单按钮偏左 10px
	if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.InitialSun == 0 {
		// 菜单按钮 X 位置
		menuButtonX := float64(WindowWidth) - config.MenuButtonOffsetFromRight
		// 铲子右边缘到菜单按钮左边缘的距离为 BowlingShovelGapFromMenuButton
		// 铲子 X = 菜单按钮 X - 间距 - 铲子宽度
		shovelX = menuButtonX - float64(config.BowlingShovelGapFromMenuButton) - float64(config.ShovelWidth)
		// 保龄球模式使用固定 Y 位置
		shovelY = float64(config.BowlingShovelY)
	} else if s.seedBank != nil {
		// 普通模式根据选择栏图片宽度动态计算
		seedBankWidth := float64(s.seedBank.Bounds().Dx())
		shovelX = float64(config.SeedBankX) + seedBankWidth + float64(config.ShovelGapFromSeedBank)
		// 普通模式：与植物选择栏同步滑入
		shovelY = s.getSeedBankCurrentY() + float64(config.ShovelY-config.SeedBankY)
	} else {
		shovelX = float64(config.ShovelX) // 默认值
		shovelY = s.getSeedBankCurrentY() + float64(config.ShovelY-config.SeedBankY)
	}

	// Draw shovel slot background first
	if s.shovelSlot != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(shovelX, shovelY)
		screen.DrawImage(s.shovelSlot, op)
	}

	// Draw shovel icon on top of the slot (centered in the slot)
	// 原版行为：铲子选中后从卡槽消失，变成鼠标光标
	if s.shovel != nil && !s.shovelSelected {
		op := &ebiten.DrawImageOptions{}
		shovelBounds := s.shovel.Bounds()
		shovelImgW := float64(shovelBounds.Dx())
		shovelImgH := float64(shovelBounds.Dy())
		// 计算居中偏移：(卡槽尺寸 - 图片尺寸) / 2
		slotW := float64(config.ShovelWidth)
		slotH := float64(config.ShovelHeight)
		offsetX := (slotW - shovelImgW) / 2.0
		offsetY := (slotH - shovelImgH) / 2.0
		op.GeoM.Translate(shovelX+offsetX, shovelY+offsetY)
		screen.DrawImage(s.shovel, op)
	} else if s.shovelSlot == nil && !s.shovelSelected {
		// Fallback: Draw a gray rectangle if both images are missing
		ebitenutil.DrawRect(screen,
			shovelX, shovelY,
			config.ShovelWidth, config.ShovelHeight,
			color.RGBA{R: 128, G: 128, B: 128, A: 255}) // Gray
	}
}

// drawProgressBar 渲染右下角的进度条（使用原版资源）
// 显示当前关卡进度和已消灭的僵尸波次
func (s *GameScene) drawProgressBar(screen *ebiten.Image) {
	// Story 11.2: 使用 ECS 进度条渲染系统
	if s.levelProgressBarRenderSystem != nil {
		s.levelProgressBarRenderSystem.Draw(screen)
	}
}

// drawLevelProgress renders the level progress indicator (Story 5.5)
// Displays current wave number and total waves in the bottom-right corner
// Format: "Wave X/Y"
func (s *GameScene) drawLevelProgress(screen *ebiten.Image) {
	// 只在关卡加载后显示进度
	if s.gameState.CurrentLevel == nil {
		return
	}

	// 获取当前波次和总波次
	currentWave, totalWaves := s.gameState.GetLevelProgress()

	// 如果还没有生成任何波次，显示"Wave 0/Y"
	// 当第一波生成后，显示"Wave 1/Y"，以此类推
	progressText := fmt.Sprintf("Wave %d/%d", currentWave, totalWaves)

	// 计算文本位置（屏幕右下角）
	// 使用阳光计数器字体（复用已有字体资源）
	if s.sunCounterFont == nil {
		return // 字体未加载时不渲染
	}

	// 测量文本宽度以便右对齐
	textWidth := text.Advance(progressText, s.sunCounterFont)

	// 位置：右下角，留10像素边距
	x := float64(WindowWidth) - textWidth - 10
	y := float64(WindowHeight) - 30.0 // 距离底部30像素

	// 绘制半透明黑色背景（提高可读性）
	bgPadding := 5.0
	ebitenutil.DrawRect(screen,
		x-bgPadding,
		y-float64(s.sunCounterFont.Metrics().HAscent)-bgPadding,
		textWidth+bgPadding*2,
		float64(s.sunCounterFont.Metrics().HAscent+s.sunCounterFont.Metrics().HDescent)+bgPadding*2,
		color.RGBA{R: 0, G: 0, B: 0, A: 150})

	// 绘制文本（白色）
	textOp := &text.DrawOptions{}
	textOp.GeoM.Translate(x, y)
	textOp.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, progressText, s.sunCounterFont, textOp)
}

// drawGameResultOverlay renders the victory or defeat overlay (Story 5.5)
// Displays when the game ends (IsGameOver = true)
// Story 8.3: 奖励流程期间不显示 You Win，让玩家专注于奖励动画
func (s *GameScene) drawGameResultOverlay(screen *ebiten.Image) {
	// 只在游戏结束时显示
	if !s.gameState.IsGameOver {
		return
	}

	// Story 8.3: 如果奖励动画正在播放，不显示游戏结果覆盖层
	// 奖励流程完成后才显示 You Win 或直接进入下一关
	if s.rewardSystem != nil && s.rewardSystem.IsActive() {
		return // 奖励动画播放期间，隐藏 You Win
	}

	// 根据游戏结果选择显示内容
	var overlayColor color.Color
	var resultText string

	switch s.gameState.GameResult {
	case "win":
		// 胜利：半透明黑色背景，绿色文本
		overlayColor = color.RGBA{R: 0, G: 0, B: 0, A: 150}
		resultText = "YOU WIN!"
	case "lose":
		// 失败：半透明红色背景，白色文本
		overlayColor = color.RGBA{R: 100, G: 0, B: 0, A: 180}
		resultText = "THE ZOMBIES ATE YOUR BRAINS!"
	default:
		return // 游戏结果未知，不显示任何内容
	}

	// 绘制全屏半透明遮罩
	ebitenutil.DrawRect(screen, 0, 0, float64(WindowWidth), float64(WindowHeight), overlayColor)

	// 绘制结果文本（屏幕中央）
	if s.sunCounterFont == nil {
		return
	}

	// 测量文本宽度以便居中
	textWidth := text.Advance(resultText, s.sunCounterFont)

	// 位置：屏幕中央
	x := (float64(WindowWidth) - textWidth) / 2.0
	y := float64(WindowHeight) / 2.0

	// 根据游戏结果选择文本颜色
	var textColor color.Color
	if s.gameState.GameResult == "win" {
		textColor = color.RGBA{R: 0, G: 255, B: 0, A: 255} // 绿色
	} else {
		textColor = color.White
	}

	// 绘制文本
	textOp := &text.DrawOptions{}
	textOp.GeoM.Translate(x, y)
	textOp.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, resultText, s.sunCounterFont, textOp)

	// 绘制提示文字（"按ESC返回主菜单"等）
	hintText := "Press ESC to return to main menu"
	hintTextWidth := text.Advance(hintText, s.sunCounterFont)
	hintX := (float64(WindowWidth) - hintTextWidth) / 2.0
	hintY := y + 40.0

	hintOp := &text.DrawOptions{}
	hintOp.GeoM.Translate(hintX, hintY)
	hintOp.ColorScale.ScaleWithColor(color.RGBA{R: 200, G: 200, B: 200, A: 255}) // 浅灰色
	text.Draw(screen, hintText, s.sunCounterFont, hintOp)
}

// drawLastWaveWarning renders the "A huge wave of zombies is approaching!" warning (Story 5.5)
// Only displays when the last wave is about to start (controlled by LevelSystem)
func (s *GameScene) drawLastWaveWarning(screen *ebiten.Image) {
	// 检查是否需要显示最后一波提示
	// 这里通过检查时间和波次状态来决定是否显示
	if s.gameState.CurrentLevel == nil {
		return
	}

	totalWaves := len(s.gameState.CurrentLevel.Waves)
	if totalWaves == 0 {
		return
	}

	// 获取当前波次索引和最后一波索引
	currentWaveIndex := s.gameState.CurrentWaveIndex
	lastWaveIndex := totalWaves - 1

	// 显示条件：进入最后一波等待期（倒数第二波消灭完毕）
	// 显示时长：直到最后一波生成前（约 minDelay 秒）
	if currentWaveIndex == lastWaveIndex && !s.gameState.IsWaveSpawned(lastWaveIndex) {

		// 绘制警告文本（屏幕中央上方）
		warningText := "A huge wave of zombies is approaching!"

		if s.sunCounterFont == nil {
			return
		}

		// 测量文本宽度以便居中
		textWidth := text.Advance(warningText, s.sunCounterFont)

		// 位置：屏幕中央上方
		x := (float64(WindowWidth) - textWidth) / 2.0
		y := 150.0

		// 绘制半透明红色背景
		bgPadding := 10.0
		ebitenutil.DrawRect(screen,
			x-bgPadding,
			y-float64(s.sunCounterFont.Metrics().HAscent)-bgPadding,
			textWidth+bgPadding*2,
			float64(s.sunCounterFont.Metrics().HAscent+s.sunCounterFont.Metrics().HDescent)+bgPadding*2,
			color.RGBA{R: 139, G: 0, B: 0, A: 200}) // 深红色背景

		// 绘制文本（黄色，更醒目）
		textOp := &text.DrawOptions{}
		textOp.GeoM.Translate(x, y)
		textOp.ColorScale.ScaleWithColor(color.RGBA{R: 255, G: 255, B: 0, A: 255}) // 黄色
		text.Draw(screen, warningText, s.sunCounterFont, textOp)
	}
}

// drawHugeWaveWarning 渲染红字警告 "A Huge Wave of Zombies is Approaching!"
// Story 17.7: 旗帜波前的红字警告动画
// Story 17.7 补充任务：优先使用 HouseofTerror28 位图字体预渲染的红色文字图片
func (s *GameScene) drawHugeWaveWarning(screen *ebiten.Image) {
	// 查询红字警告实体
	warningEntities := ecs.GetEntitiesWith1[*components.FlagWaveWarningComponent](s.entityManager)
	if len(warningEntities) == 0 {
		return
	}

	// 获取警告组件
	warningComp, ok := ecs.GetComponent[*components.FlagWaveWarningComponent](s.entityManager, warningEntities[0])
	if !ok || !warningComp.IsActive {
		return
	}

	// 闪烁效果：不可见时跳过渲染
	if !warningComp.FlashVisible {
		return
	}

	// 优先使用预渲染的位图字体图片
	if warningComp.TextImage != nil {
		s.drawHugeWaveWarningBitmap(screen, warningComp)
		return
	}

	// 回退：使用 sunCounterFont 渲染文本
	s.drawHugeWaveWarningText(screen, warningComp)
}

// drawHugeWaveWarningBitmap 使用预渲染的位图字体图片绘制警告
// Story 17.7 补充任务：使用 HouseofTerror28 位图字体渲染的红色文字
func (s *GameScene) drawHugeWaveWarningBitmap(screen *ebiten.Image, warningComp *components.FlagWaveWarningComponent) {
	textImage := warningComp.TextImage
	imgWidth := float64(textImage.Bounds().Dx())
	imgHeight := float64(textImage.Bounds().Dy())

	// 计算缩放后的尺寸
	scaledWidth := imgWidth * warningComp.Scale
	scaledHeight := imgHeight * warningComp.Scale

	// 绘制半透明黑色背景（提高可读性）
	bgPadding := 15.0 * warningComp.Scale
	x := warningComp.X - scaledWidth/2
	y := warningComp.Y
	ebitenutil.DrawRect(screen,
		x-bgPadding,
		y-bgPadding,
		scaledWidth+bgPadding*2,
		scaledHeight+bgPadding*2,
		color.RGBA{R: 0, G: 0, B: 0, A: 180})

	// 绘制预渲染的红色文字图片
	op := &ebiten.DrawImageOptions{}
	// 先移动到原点（图片左上角）
	op.GeoM.Translate(-imgWidth/2, 0)
	// 应用缩放
	op.GeoM.Scale(warningComp.Scale, warningComp.Scale)
	// 移动到目标位置
	op.GeoM.Translate(warningComp.X, warningComp.Y)

	// 应用透明度
	op.ColorScale.ScaleAlpha(float32(warningComp.Alpha))

	screen.DrawImage(textImage, op)
}

// drawHugeWaveWarningText 使用系统字体绘制警告（回退方案）
// Story 17.7: 当位图字体不可用时使用 sunCounterFont
func (s *GameScene) drawHugeWaveWarningText(screen *ebiten.Image, warningComp *components.FlagWaveWarningComponent) {
	// 使用阳光计数器字体
	if s.sunCounterFont == nil {
		return
	}

	warningText := warningComp.Text

	// 测量文本宽度以便居中
	textWidth := text.Advance(warningText, s.sunCounterFont)

	// 计算位置（带缩放）
	scaledWidth := textWidth * warningComp.Scale
	x := warningComp.X - scaledWidth/2
	y := warningComp.Y

	// 绘制半透明黑色背景（提高可读性）
	bgPadding := 15.0 * warningComp.Scale
	fontMetrics := s.sunCounterFont.Metrics()
	bgHeight := float64(fontMetrics.HAscent+fontMetrics.HDescent) * warningComp.Scale
	ebitenutil.DrawRect(screen,
		x-bgPadding,
		y-bgPadding,
		scaledWidth+bgPadding*2,
		bgHeight+bgPadding*2,
		color.RGBA{R: 0, G: 0, B: 0, A: 180})

	// 绘制红色警告文本
	textOp := &text.DrawOptions{}
	textOp.GeoM.Translate(-textWidth/2, 0) // 先移到原点
	textOp.GeoM.Scale(warningComp.Scale, warningComp.Scale)
	textOp.GeoM.Translate(warningComp.X, warningComp.Y)

	// 红色文字 (#FF0000)，带透明度
	alpha := uint8(255 * warningComp.Alpha)
	textOp.ColorScale.ScaleWithColor(color.RGBA{R: 255, G: 0, B: 0, A: alpha})

	text.Draw(screen, warningText, s.sunCounterFont, textOp)
}

// drawTooltip 渲染植物卡片的 Tooltip
// Story 10.8: 鼠标悬停植物卡片时显示提示信息
func (s *GameScene) drawTooltip(screen *ebiten.Image) {
	// 查询 Tooltip 实体
	tooltipEntities := ecs.GetEntitiesWith1[*components.TooltipComponent](s.entityManager)
	if len(tooltipEntities) == 0 {
		return
	}

	// 获取第一个 Tooltip（游戏中只有一个全局 Tooltip）
	tooltip, ok := ecs.GetComponent[*components.TooltipComponent](s.entityManager, tooltipEntities[0])
	if !ok || !tooltip.IsVisible {
		return
	}

	// 使用阳光计数器字体（sunCounterFont）
	if s.sunCounterFont == nil {
		return
	}

	// 测量文本尺寸
	plantNameWidth, _ := text.Measure(tooltip.PlantName, s.sunCounterFont, 0)
	statusTextWidth := 0.0
	if tooltip.StatusText != "" {
		statusTextWidth, _ = text.Measure(tooltip.StatusText, s.sunCounterFont, 0)
	}

	// 计算 Tooltip 尺寸（包含边距）
	tooltipWidth := plantNameWidth
	if statusTextWidth > tooltipWidth {
		tooltipWidth = statusTextWidth
	}
	tooltipWidth += tooltip.Padding * 2

	// 字体高度
	fontMetrics := s.sunCounterFont.Metrics()
	lineHeight := float64(fontMetrics.HAscent + fontMetrics.HDescent)

	// 计算 Tooltip 高度（包含边距）
	tooltipHeight := tooltip.Padding*2 + lineHeight
	if tooltip.StatusText != "" {
		tooltipHeight += lineHeight + tooltip.TextSpacing
	}

	// 计算 Tooltip 位置（卡片下方居中，因为卡片在屏幕顶部）
	// tooltip.X 是卡片中心 X 坐标，tooltip.Y 是卡片底部 Y 坐标
	tooltipX := tooltip.X - tooltipWidth/2
	tooltipY := tooltip.Y + 5 // 卡片下方 5px

	// 确保 Tooltip 不超出屏幕边界
	if tooltipX < 5 {
		tooltipX = 5
	}
	if tooltipX+tooltipWidth > float64(WindowWidth)-5 {
		tooltipX = float64(WindowWidth) - tooltipWidth - 5
	}

	// 1. 绘制背景
	ebitenutil.DrawRect(screen, tooltipX, tooltipY, tooltipWidth, tooltipHeight, tooltip.BackgroundColor)

	// 2. 绘制边框（黑色，1px）
	borderWidth := 1.0
	// 上边框
	ebitenutil.DrawRect(screen, tooltipX, tooltipY, tooltipWidth, borderWidth, tooltip.BorderColor)
	// 下边框
	ebitenutil.DrawRect(screen, tooltipX, tooltipY+tooltipHeight-borderWidth, tooltipWidth, borderWidth, tooltip.BorderColor)
	// 左边框
	ebitenutil.DrawRect(screen, tooltipX, tooltipY, borderWidth, tooltipHeight, tooltip.BorderColor)
	// 右边框
	ebitenutil.DrawRect(screen, tooltipX+tooltipWidth-borderWidth, tooltipY, borderWidth, tooltipHeight, tooltip.BorderColor)

	// 3. 渲染文本（从 Tooltip 顶部 + padding 开始）
	// Ebitengine text.Draw 的 Y 坐标是文字顶部位置
	currentY := tooltipY + tooltip.Padding

	// 3.1 渲染状态提示（第一行，如果存在）
	if tooltip.StatusText != "" {
		statusX := tooltipX + (tooltipWidth-statusTextWidth)/2
		statusOp := &text.DrawOptions{}
		statusOp.GeoM.Translate(statusX, currentY)
		statusOp.ColorScale.ScaleWithColor(tooltip.StatusTextColor)
		text.Draw(screen, tooltip.StatusText, s.sunCounterFont, statusOp)
		currentY += lineHeight + tooltip.TextSpacing
	}

	// 3.2 渲染植物名
	plantNameX := tooltipX + (tooltipWidth-plantNameWidth)/2
	plantNameOp := &text.DrawOptions{}
	plantNameOp.GeoM.Translate(plantNameX, currentY)
	plantNameOp.ColorScale.ScaleWithColor(tooltip.PlantNameColor)
	text.Draw(screen, tooltip.PlantName, s.sunCounterFont, plantNameOp)
}
