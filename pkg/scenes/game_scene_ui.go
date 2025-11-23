package scenes

import (
	"fmt"
	"image/color"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// drawSeedBank renders the plant selection bar at the top left of the screen.
// If the seed bank image is not loaded, it draws a simple rectangle as fallback.
func (s *GameScene) drawSeedBank(screen *ebiten.Image) {
	// Story 8.8: 游戏冻结时隐藏植物选择栏
	freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](s.entityManager)
	if len(freezeEntities) > 0 {
		return
	}

	if s.seedBank != nil {
		// Draw the seed bank image at the top left corner
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(config.SeedBankX, config.SeedBankY)
		screen.DrawImage(s.seedBank, op)
	} else {
		// Fallback: Draw a dark brown rectangle
		ebitenutil.DrawRect(screen,
			config.SeedBankX, config.SeedBankY,
			config.SeedBankWidth, config.SeedBankHeight,
			color.RGBA{R: 101, G: 67, B: 33, A: 255}) // Dark brown
	}
}

// drawSunCounter renders the sun counter value on the seed bank.
// Note: The sun counter background and gold frame are already part of the bar5.png image,
// so we don't need to draw them separately. This method displays the sun count number.
// The text is horizontally centered to accommodate dynamic value lengths (e.g., 50, 150, 9990).
func (s *GameScene) drawSunCounter(screen *ebiten.Image) {
	// Story 8.8: 游戏冻结时隐藏阳光计数器
	freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](s.entityManager)
	if len(freezeEntities) > 0 {
		return
	}

	// Get current sun value from game state
	sunValue := s.gameState.GetSun()
	sunText := fmt.Sprintf("%d", sunValue)

	if s.sunCounterFont != nil {
		// Measure text width for centering
		textWidth, _ := text.Measure(sunText, s.sunCounterFont, 0)

		// Calculate centered position
		// Base position is relative to SeedBank
		centerX := float64(config.SeedBankX + config.SunCounterOffsetX)
		centerY := float64(config.SeedBankY + config.SunCounterOffsetY)

		// Adjust X to center the text horizontally
		sunDisplayX := centerX - textWidth/2
		sunDisplayY := centerY

		// Use custom font with color
		op := &text.DrawOptions{}
		op.GeoM.Translate(sunDisplayX, sunDisplayY)

		// Set text color to black for better visibility on the beige background
		op.ColorScale.ScaleWithColor(color.RGBA{R: 0, G: 0, B: 0, A: 255})

		text.Draw(screen, sunText, s.sunCounterFont, op)
	} else {
		// Fallback: Use debug text if font failed to load
		// Note: Debug text doesn't support centering easily
		sunDisplayX := config.SeedBankX + config.SunCounterOffsetX
		sunDisplayY := config.SeedBankY + config.SunCounterOffsetY
		ebitenutil.DebugPrintAt(screen, sunText, sunDisplayX, sunDisplayY)
	}
}

// drawShovel renders the shovel slot and icon at the right side of the seed bank.
// The shovel will be used in future stories for removing plants.
// Story 8.5: 1-1关（教学关卡）不显示铲子
// Story 8.6: 检查铲子是否已解锁（1-4关完成后才解锁）
func (s *GameScene) drawShovel(screen *ebiten.Image) {
	// 教学关卡不显示铲子（玩家还不需要学习移除植物）
	if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.OpeningType == "tutorial" {
		return
	}

	// Story 8.6: 检查铲子是否已解锁
	// 铲子在完成 1-4 关卡后解锁
	if !s.gameState.IsToolUnlocked("shovel") {
		return
	}

	// Draw shovel slot background first
	if s.shovelSlot != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(config.ShovelX, config.ShovelY)
		screen.DrawImage(s.shovelSlot, op)
	}

	// Draw shovel icon on top of the slot
	if s.shovel != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(config.ShovelX, config.ShovelY)
		screen.DrawImage(s.shovel, op)
	} else if s.shovelSlot == nil {
		// Fallback: Draw a gray rectangle if both images are missing
		ebitenutil.DrawRect(screen,
			config.ShovelX, config.ShovelY,
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
