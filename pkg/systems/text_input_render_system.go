package systems

import (
	"image"
	"image/color"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// TextInputRenderSystem 文本输入框渲染系统
// 负责绘制输入框边框、背景、文本和光标
type TextInputRenderSystem struct {
	entityManager *ecs.EntityManager
	font          *text.GoTextFace // 文本字体
}

// NewTextInputRenderSystem 创建文本输入框渲染系统
func NewTextInputRenderSystem(em *ecs.EntityManager, font *text.GoTextFace) *TextInputRenderSystem {
	return &TextInputRenderSystem{
		entityManager: em,
		font:          font,
	}
}

// Draw 绘制所有文本输入框
// ✅ Story 12.4: 此方法已废弃，输入框现在由 DialogRenderSystem 负责渲染
// 保留此方法是为了向后兼容，实际上不会被调用
func (s *TextInputRenderSystem) Draw(screen *ebiten.Image) {
	// 空实现 - 输入框现在由 DialogRenderSystem 渲染
	// DialogRenderSystem 会在渲染每个对话框后立即渲染其子实体（输入框）
	// 这样确保输入框跟随父对话框的z-order
}

// DrawInputBox 绘制单个输入框（公开方法，供 DialogRenderSystem 调用）
// Story 12.4: 从 drawInputBox 重命名为 DrawInputBox
func (s *TextInputRenderSystem) DrawInputBox(screen *ebiten.Image, input *components.TextInputComponent, pos *components.PositionComponent) {
	x := pos.X
	y := pos.Y
	width := input.Width
	height := input.Height

	// 1. 绘制边框（editbox.gif，金黄色边框）
	// 注意：不绘制背景层（BackgroundImage），只绘制边框避免黑边
	if input.BorderImage != nil {
		s.drawStretchedImage(screen, input.BorderImage, x, y, width, height)
	}

	// 2. 绘制文本或占位符
	textX := x + input.PaddingLeft
	textY := y + height/2 // 垂直居中（忽略 PaddingTop）

	if input.Text == "" && input.Placeholder != "" && !input.IsFocused {
		// 显示占位符（灰色）
		s.drawText(screen, input.Placeholder, textX, textY, color.RGBA{150, 150, 150, 255})
	} else if input.Text != "" {
		// 显示实际文本（绿色 - 游戏内菜单按钮样式）
		s.drawText(screen, input.Text, textX+input.TextOffsetX, textY, color.RGBA{0, 200, 0, 255})
	}

	// 3. 绘制光标（闪烁的竖线）
	if input.IsFocused && input.CursorVisible {
		s.drawCursor(screen, input, pos, textX, textY)
	}
}

// drawStretchedImage 绘制水平拉伸的图片（九宫格拉伸）
// 将图片分为左、中、右三部分，中间部分水平拉伸
func (s *TextInputRenderSystem) drawStretchedImage(screen *ebiten.Image, img *ebiten.Image, x, y, targetWidth, targetHeight float64) {
	bounds := img.Bounds()
	imgWidth := float64(bounds.Dx())
	imgHeight := float64(bounds.Dy())

	// 定义边缘宽度（不拉伸的部分）
	// 注意：增大 edgeWidth 以跳过边框图片边缘的黑色部分
	const edgeWidth = 10.0 // 左右各 10 像素不拉伸（避免黑边）

	// 如果目标宽度小于原始宽度，直接缩放绘制
	if targetWidth < imgWidth {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(targetWidth/imgWidth, targetHeight/imgHeight)
		op.GeoM.Translate(x, y)
		screen.DrawImage(img, op)
		return
	}

	// 九宫格拉伸：左边缘 + 中间拉伸 + 右边缘
	scaleY := targetHeight / imgHeight

	// 1. 绘制左边缘（不拉伸）
	leftPart := img.SubImage(image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: int(edgeWidth), Y: bounds.Dy()},
	}).(*ebiten.Image)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1.0, scaleY)
	op.GeoM.Translate(x, y)
	screen.DrawImage(leftPart, op)

	// 2. 绘制中间部分（水平拉伸）
	middlePart := img.SubImage(image.Rectangle{
		Min: image.Point{X: int(edgeWidth), Y: 0},
		Max: image.Point{X: int(imgWidth - edgeWidth), Y: bounds.Dy()},
	}).(*ebiten.Image)

	middleWidth := targetWidth - edgeWidth*2
	middleScaleX := middleWidth / (imgWidth - edgeWidth*2)

	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(middleScaleX, scaleY)
	op.GeoM.Translate(x+edgeWidth, y)
	screen.DrawImage(middlePart, op)

	// 3. 绘制右边缘（不拉伸）
	rightPart := img.SubImage(image.Rectangle{
		Min: image.Point{X: int(imgWidth - edgeWidth), Y: 0},
		Max: image.Point{X: bounds.Dx(), Y: bounds.Dy()},
	}).(*ebiten.Image)

	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1.0, scaleY)
	op.GeoM.Translate(x+targetWidth-edgeWidth, y)
	screen.DrawImage(rightPart, op)
}

// drawText 绘制文本
func (s *TextInputRenderSystem) drawText(screen *ebiten.Image, txt string, x, y float64, clr color.Color) {
	if s.font == nil || txt == "" {
		return
	}

	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)

	// 垂直居中对齐
	op.PrimaryAlign = text.AlignStart
	op.SecondaryAlign = text.AlignCenter

	text.Draw(screen, txt, s.font, op)
}

// drawCursor 绘制光标
func (s *TextInputRenderSystem) drawCursor(screen *ebiten.Image, input *components.TextInputComponent, pos *components.PositionComponent, textX, textY float64) {
	if s.font == nil {
		return
	}

	// 计算光标位置（光标在第 N 个字符后面）
	runes := []rune(input.Text)
	textBeforeCursor := string(runes[:input.CursorPosition])

	// 测量文本宽度
	var textWidth float64
	if textBeforeCursor != "" {
		w, _ := text.Measure(textBeforeCursor, s.font, 0)
		textWidth = w
	}

	cursorX := textX + input.TextOffsetX + textWidth
	cursorY := textY - input.Height/4 // 光标顶部
	cursorHeight := input.Height / 2  // 光标高度

	// 绘制光标（2像素宽的白色竖线）
	cursorImg := ebiten.NewImage(2, int(cursorHeight))
	cursorImg.Fill(color.RGBA{255, 255, 255, 255})

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(cursorX, cursorY)
	screen.DrawImage(cursorImg, op)
}
