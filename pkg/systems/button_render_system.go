package systems

import (
	"image/color"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ButtonRenderSystem 按钮渲染系统
// 负责渲染所有按钮实体（三段式按钮和简单按钮）
//
// 职责：
//   - 渲染按钮背景（三段式 or 简单图片）
//   - 渲染按钮文字（自动居中）
//   - 根据按钮状态选择不同图片（hover/pressed）
type ButtonRenderSystem struct {
	entityManager *ecs.EntityManager
}

// NewButtonRenderSystem 创建按钮渲染系统
func NewButtonRenderSystem(em *ecs.EntityManager) *ButtonRenderSystem {
	return &ButtonRenderSystem{
		entityManager: em,
	}
}

// Draw 渲染所有按钮
// 查询所有拥有 ButtonComponent 和 PositionComponent 的实体并渲染
func (s *ButtonRenderSystem) Draw(screen *ebiten.Image) {
	// 检查游戏是否冻结（僵尸获胜流程期间）
	freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](s.entityManager)
	isFrozen := len(freezeEntities) > 0

	// 查询所有按钮实体
	entities := ecs.GetEntitiesWith2[*components.ButtonComponent, *components.PositionComponent](s.entityManager)

	for _, entityID := range entities {
		// 冻结时，隐藏菜单按钮（通过 MenuButtonComponent 标记识别）
		if isFrozen {
			if ecs.HasComponent[*components.MenuButtonComponent](s.entityManager, entityID) {
				continue // 跳过菜单按钮
			}
		}

		s.DrawButton(screen, entityID)
	}
}

// DrawButton 渲染单个按钮实体
// 用于需要精确控制渲染顺序的场景（如暂停菜单）
func (s *ButtonRenderSystem) DrawButton(screen *ebiten.Image, entityID ecs.EntityID) {
	button, ok := ecs.GetComponent[*components.ButtonComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 渲染按钮背景
	s.drawButtonBackground(screen, button, pos.X, pos.Y)

	// 渲染按钮文字
	s.drawButtonText(screen, button, pos.X, pos.Y)
}

// drawButtonBackground 渲染按钮背景
func (s *ButtonRenderSystem) drawButtonBackground(screen *ebiten.Image, button *components.ButtonComponent, x, y float64) {
	if button.Type == components.ButtonTypeNineSlice {
		// 三段式可拉伸按钮
		s.drawNineSliceButton(screen, button, x, y)
	} else {
		// 简单图片按钮
		s.drawSimpleButton(screen, button, x, y)
	}
}

// drawNineSliceButton 渲染三段式按钮（左、中、右）
func (s *ButtonRenderSystem) drawNineSliceButton(screen *ebiten.Image, button *components.ButtonComponent, x, y float64) {
	if button.LeftImage == nil || button.MiddleImage == nil || button.RightImage == nil {
		return
	}

	// ✅ 按下状态：按钮向下偏移 2px
	pressOffsetY := 0.0
	if button.State == components.UIClicked {
		pressOffsetY = 2.0
	}

	leftWidth := float64(button.LeftImage.Bounds().Dx())
	rightWidth := float64(button.RightImage.Bounds().Dx())
	middleWidth := button.MiddleWidth

	// 绘制左边缘
	leftOp := &ebiten.DrawImageOptions{}
	leftOp.GeoM.Translate(x, y+pressOffsetY) // ✅ 应用按下偏移
	screen.DrawImage(button.LeftImage, leftOp)

	// 绘制中间（拉伸）
	middleOp := &ebiten.DrawImageOptions{}
	middleOp.GeoM.Scale(middleWidth/float64(button.MiddleImage.Bounds().Dx()), 1.0)
	middleOp.GeoM.Translate(x+leftWidth, y+pressOffsetY) // ✅ 应用按下偏移
	screen.DrawImage(button.MiddleImage, middleOp)

	// 绘制右边缘
	rightOp := &ebiten.DrawImageOptions{}
	rightOp.GeoM.Translate(x+leftWidth+middleWidth, y+pressOffsetY) // ✅ 应用按下偏移
	screen.DrawImage(button.RightImage, rightOp)

	// 更新按钮尺寸（缓存）
	button.Width = leftWidth + middleWidth + rightWidth
	button.Height = float64(button.LeftImage.Bounds().Dy())
}

// drawSimpleButton 渲染简单图片按钮
func (s *ButtonRenderSystem) drawSimpleButton(screen *ebiten.Image, button *components.ButtonComponent, x, y float64) {
	// ✅ 修复：根据状态选择图片
	// - UINormal: 正常图片
	// - UIHovered: 悬停图片（如果有，否则用正常图片）
	// - UIClicked: 按下图片（如果有，否则用悬停或正常图片）
	var img *ebiten.Image
	switch button.State {
	case components.UIHovered:
		// 悬停状态：优先使用悬停图片，否则用正常图片
		if button.HoverImage != nil {
			img = button.HoverImage
		} else {
			img = button.NormalImage
		}
	case components.UIClicked:
		// ✅ 按下状态：优先使用按下图片，否则降级到悬停或正常图片
		if button.PressedImage != nil {
			img = button.PressedImage
		} else if button.HoverImage != nil {
			img = button.HoverImage
		} else {
			img = button.NormalImage
		}
	default:
		img = button.NormalImage
	}

	if img == nil {
		return
	}

	// ✅ 按下状态偏移逻辑：
	// 如果 PressedImage 与 NormalImage 相同，则应用代码偏移（SeedChooser_Button 系列）
	// 如果 PressedImage 与 NormalImage 不同，则不偏移（backtogamebutton 系列，图片自带下陷效果）
	pressOffsetY := 0.0
	if button.State == components.UIClicked {
		if button.PressedImage == button.NormalImage {
			// 图片相同，应用代码偏移
			pressOffsetY = 2.0
		}
		// 图片不同，不偏移（图片本身已有下陷效果）
	}

	// 绘制按钮图片
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y+pressOffsetY) // 应用按下偏移（如果需要）
	screen.DrawImage(img, op)

	// 更新按钮尺寸（缓存）
	button.Width = float64(img.Bounds().Dx())
	button.Height = float64(img.Bounds().Dy())
}

// drawButtonText 渲染按钮文字（自动居中，带阴影效果）
func (s *ButtonRenderSystem) drawButtonText(screen *ebiten.Image, button *components.ButtonComponent, x, y float64) {
	if button.Text == "" || button.Font == nil {
		return
	}

	// ✅ 按下状态偏移逻辑：文字始终跟随按钮视觉效果
	// 两种情况都需要偏移：
	// 1. SeedChooser_Button 系列：图片代码偏移 2px，文字也偏移 2px
	// 2. backtogamebutton 系列：图片自带下陷效果（视觉上向下约 2px），文字也偏移 2px
	// 结论：无论哪种按钮，文字都向下偏移 2px
	pressOffsetY := 0.0
	if button.State == components.UIClicked {
		pressOffsetY = 2.0
	}

	// 计算按钮中心点
	centerX := x + button.Width/2
	centerY := y + button.Height/2 + pressOffsetY // 应用按下偏移

	// 阴影偏移量
	shadowOffsetX := 2.0
	shadowOffsetY := 2.0

	// 为了让"文字+阴影"整体看起来垂直居中，将主文字向上偏移阴影的一半
	visualCenterOffsetY := -shadowOffsetY / 2.0

	// 1. 先绘制阴影（深色文字，偏移位置）
	shadowOp := &text.DrawOptions{}
	shadowOp.LayoutOptions.PrimaryAlign = text.AlignCenter
	shadowOp.LayoutOptions.SecondaryAlign = text.AlignCenter
	shadowOp.GeoM.Translate(centerX+shadowOffsetX, centerY+shadowOffsetY+visualCenterOffsetY)
	shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 180}) // 半透明黑色阴影
	text.Draw(screen, button.Text, button.Font, shadowOp)

	// 2. 再绘制主文字（向上偏移，使整体视觉居中）
	op := &text.DrawOptions{}
	op.LayoutOptions.PrimaryAlign = text.AlignCenter   // 水平居中
	op.LayoutOptions.SecondaryAlign = text.AlignCenter // 垂直居中
	op.GeoM.Translate(centerX, centerY+visualCenterOffsetY)

	// 设置文字颜色
	op.ColorScale.ScaleWithColor(color.RGBA{
		R: button.TextColor[0],
		G: button.TextColor[1],
		B: button.TextColor[2],
		A: button.TextColor[3],
	})

	text.Draw(screen, button.Text, button.Font, op)
}
