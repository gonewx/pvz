package utils

import (
	"github.com/decker502/pvz/pkg/components"
	"github.com/hajimehoshi/ebiten/v2"
)

// RenderNinePatch 使用九宫格拉伸渲染对话框
// x, y: 对话框左上角位置
// width, height: 对话框总大小
// parts: 九宫格图片资源
func RenderNinePatch(screen *ebiten.Image, parts *components.DialogParts, x, y, width, height float64) {
	if parts == nil {
		return
	}

	// 1. 获取边角大小（从图片获取）
	var cornerWidth, cornerHeight float64
	if parts.TopLeft != nil {
		tlBounds := parts.TopLeft.Bounds()
		cornerWidth = float64(tlBounds.Dx())
		cornerHeight = float64(tlBounds.Dy())
	}

	// 如果没有边角图片，无法渲染
	if cornerWidth == 0 || cornerHeight == 0 {
		return
	}

	// 2. 计算拉伸区域大小
	stretchWidth := width - cornerWidth*2    // 中间区域宽度
	stretchHeight := height - cornerHeight*2 // 中间区域高度

	// 确保拉伸区域不为负数
	if stretchWidth < 0 {
		stretchWidth = 0
	}
	if stretchHeight < 0 {
		stretchHeight = 0
	}

	// 3. 绘制四个边角（固定位置，不拉伸）
	if parts.TopLeft != nil {
		drawImage(screen, parts.TopLeft, x, y, 1.0, 1.0)
	}

	if parts.TopRight != nil {
		drawImage(screen, parts.TopRight, x+width-cornerWidth, y, 1.0, 1.0)
	}

	if parts.BottomLeft != nil {
		drawImage(screen, parts.BottomLeft, x, y+height-cornerHeight, 1.0, 1.0)
	}

	if parts.BottomRight != nil {
		drawImage(screen, parts.BottomRight, x+width-cornerWidth, y+height-cornerHeight, 1.0, 1.0)
	}

	// 4. 绘制四个边缘（单向拉伸）
	if parts.TopMiddle != nil && stretchWidth > 0 {
		tmBounds := parts.TopMiddle.Bounds()
		scaleX := stretchWidth / float64(tmBounds.Dx())
		drawImage(screen, parts.TopMiddle, x+cornerWidth, y, scaleX, 1.0)
	}

	if parts.BottomMiddle != nil && stretchWidth > 0 {
		bmBounds := parts.BottomMiddle.Bounds()
		scaleX := stretchWidth / float64(bmBounds.Dx())
		drawImage(screen, parts.BottomMiddle, x+cornerWidth, y+height-cornerHeight, scaleX, 1.0)
	}

	if parts.CenterLeft != nil && stretchHeight > 0 {
		clBounds := parts.CenterLeft.Bounds()
		scaleY := stretchHeight / float64(clBounds.Dy())
		drawImage(screen, parts.CenterLeft, x, y+cornerHeight, 1.0, scaleY)
	}

	if parts.CenterRight != nil && stretchHeight > 0 {
		crBounds := parts.CenterRight.Bounds()
		scaleY := stretchHeight / float64(crBounds.Dy())
		drawImage(screen, parts.CenterRight, x+width-cornerWidth, y+cornerHeight, 1.0, scaleY)
	}

	// 5. 绘制中心区域（双向拉伸）
	if parts.CenterMiddle != nil && stretchWidth > 0 && stretchHeight > 0 {
		cmBounds := parts.CenterMiddle.Bounds()
		scaleXCenter := stretchWidth / float64(cmBounds.Dx())
		scaleYCenter := stretchHeight / float64(cmBounds.Dy())
		drawImage(screen, parts.CenterMiddle, x+cornerWidth, y+cornerHeight, scaleXCenter, scaleYCenter)
	}
}

// drawImage 辅助函数：绘制带缩放的图片
func drawImage(screen *ebiten.Image, img *ebiten.Image, x, y, scaleX, scaleY float64) {
	if img == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Translate(x, y)
	screen.DrawImage(img, op)
}
