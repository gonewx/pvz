package scenes

import (
	"fmt"
	"image/color"
	"log"

	"github.com/decker502/pvz/pkg/config"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// drawGridDebug 绘制草坪网格边界（调试用）
// 在开发阶段帮助可视化可种植区域
// 注：调试绘制已禁用，如需启用请设置 debugEnabled = true
func (s *GameScene) drawGridDebug(screen *ebiten.Image) {
	const debugEnabled = false
	if !debugEnabled {
		return
	}

	// Story 8.2 QA: 临时启用调试绘制，验证草坪布局和SodRoll动画
	// 在种植模式或SodRoll动画期间显示
	showDebug := s.gameState.IsPlantingMode
	// if !showDebug && s.soddingSystem != nil {
	// 	// 如果SodRoll动画启动过（包括正在播放和已完成），也显示调试信息
	// 	showDebug = s.soddingSystem.HasStarted()
	// }

	if !showDebug {
		return
	}

	// 使用统一的网格参数（从 config.layout_config.go）
	// 注意：这里使用的是世界坐标，需要转换为屏幕坐标
	gridWorldStartX := config.GridWorldStartX
	gridWorldStartY := config.GridWorldStartY
	gridColumns := config.GridColumns
	gridRows := config.GridRows
	cellWidth := config.CellWidth
	cellHeight := config.CellHeight

	// 将网格世界坐标转换为屏幕坐标
	gridScreenStartX := gridWorldStartX - s.cameraX
	gridScreenStartY := gridWorldStartY

	// 绘制网格线
	gridColor := color.RGBA{R: 255, G: 255, B: 0, A: 128} // 半透明黄色

	// 绘制垂直线
	for col := 0; col <= gridColumns; col++ {
		x := gridScreenStartX + float64(col)*cellWidth
		ebitenutil.DrawLine(screen, x, gridScreenStartY, x, gridScreenStartY+float64(gridRows)*cellHeight, gridColor)
	}

	// 绘制水平线
	for row := 0; row <= gridRows; row++ {
		y := gridScreenStartY + float64(row)*cellHeight
		ebitenutil.DrawLine(screen, gridScreenStartX, y, gridScreenStartX+float64(gridColumns)*cellWidth, y, gridColor)
	}

	// Story 8.2 QA: 绘制草皮叠加图边界（红色矩形）
	if s.sodRowImage != nil {
		// 性能优化：使用缓存的尺寸和位置
		sodWidth := float64(s.sodWidth)
		sodHeight := float64(s.sodHeight)
		sodOverlayX := s.sodOverlayX
		sodOverlayY := s.sodOverlayY

		// 转换为屏幕坐标
		sodScreenX := sodOverlayX - s.cameraX
		sodScreenY := sodOverlayY

		// Story 8.2 QA：调试可视化（临时启用以调试粒子）

		// 绘制草皮边界（红色矩形框，不填充）
		sodColor := color.RGBA{R: 255, G: 0, B: 0, A: 128} // 半透明红色
		thickness := 2.0
		// 顶边
		ebitenutil.DrawRect(screen, sodScreenX, sodScreenY, sodWidth, thickness, sodColor)
		// 底边
		ebitenutil.DrawRect(screen, sodScreenX, sodScreenY+sodHeight-thickness, sodWidth, thickness, sodColor)
		// 左边
		ebitenutil.DrawRect(screen, sodScreenX, sodScreenY, thickness, sodHeight, sodColor)
		// 右边
		ebitenutil.DrawRect(screen, sodScreenX+sodWidth-thickness, sodScreenY, thickness, sodHeight, sodColor)

		// 绘制草皮卷左、中、右边缘标记
		// 只要动画启动过就绘制（包括已完成的状态）
		if s.soddingSystem != nil && s.soddingSystem.HasStarted() {
			leftEdge, centerX, rightEdge := s.soddingSystem.GetSodRollEdges()

			// 转换为屏幕坐标
			leftScreenX := leftEdge - s.cameraX
			centerScreenX := centerX - s.cameraX
			rightScreenX := rightEdge - s.cameraX

			// DEBUG: 每10帧打印一次
			frameIndex := int(s.soddingSystem.GetProgress() * 48)
			if frameIndex%10 == 0 || frameIndex == 0 {
				log.Printf("[调试线] 帧:%d, 世界坐标: 左=%.1f 中=%.1f 右=%.1f, 屏幕坐标: 左=%.1f 中=%.1f 右=%.1f",
					frameIndex, leftEdge, centerX, rightEdge, leftScreenX, centerScreenX, rightScreenX)
			}

			// 绘制三条竖线（加粗，更明显）
			lineHeight := sodHeight + 40.0
			lineStartY := sodScreenY - 20.0
			lineWidth := 4.0 // 加粗线条

			// 左边缘 - 红色
			leftColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
			ebitenutil.DrawRect(screen, leftScreenX-lineWidth/2, lineStartY, lineWidth, lineHeight, leftColor)

			// 中心 - 绿色
			centerColor := color.RGBA{R: 0, G: 255, B: 0, A: 255}
			ebitenutil.DrawRect(screen, centerScreenX-lineWidth/2, lineStartY, lineWidth, lineHeight, centerColor)

			// 右边缘 - 蓝色
			rightColor := color.RGBA{R: 0, G: 0, B: 255, A: 255}
			ebitenutil.DrawRect(screen, rightScreenX-lineWidth/2, lineStartY, lineWidth, lineHeight, rightColor)
		}

		// 详细调试信息
		debugInfo := fmt.Sprintf("Sod: world(%.0f,%.0f) screen(%.0f,%.0f) size(%.0fx%.0f) cam(%.0f)",
			sodOverlayX, sodOverlayY, sodScreenX, sodScreenY, sodWidth, sodHeight, s.cameraX)
		ebitenutil.DebugPrintAt(screen, debugInfo, 10, 30)

		// 草皮卷边缘位置调试信息
		if s.soddingSystem != nil && s.soddingSystem.IsPlaying() {
			leftEdge, centerX, rightEdge := s.soddingSystem.GetSodRollEdges()
			progress := s.soddingSystem.GetProgress()
			edgeInfo := fmt.Sprintf("草皮卷: 左(%.0f) 中(%.0f) 右(%.0f) 进度(%.1f%%)",
				leftEdge, centerX, rightEdge, progress*100)
			ebitenutil.DebugPrintAt(screen, edgeInfo, 10, 50)
		}
	}

	// 绘制坐标信息文本
	debugText := fmt.Sprintf("Grid: (%.0f, %.0f), Cell: %.0fx%.0f", gridWorldStartX, gridWorldStartY, cellWidth, cellHeight)
	ebitenutil.DebugPrintAt(screen, debugText, 10, 10)
}

// drawParticleTestInstructions 绘制粒子效果测试说明（调试用）
// Story 7.4: 方便测试粒子效果，无需通过攻击触发
func (s *GameScene) drawParticleTestInstructions(screen *ebiten.Image) {
	// 只在非游戏结束状态下显示
	if s.gameState.IsGameOver {
		return
	}

	// 测试说明（屏幕左下角）
	instructions := []string{
		"[粒子测试] P=豌豆溅射 | B=爆炸 | A=奖励光效 | Z=僵尸头 | L=种植土粒",
	}

	// 绘制半透明背景
	y := float64(WindowHeight - 25)
	bgPadding := 5.0
	ebitenutil.DrawRect(screen,
		10-bgPadding,
		y-bgPadding,
		300,
		20,
		color.RGBA{R: 0, G: 0, B: 0, A: 120})

	// 绘制文本
	for i, line := range instructions {
		yPos := int(y) + i*15
		ebitenutil.DebugPrintAt(screen, line, 10, yPos)
	}
}
