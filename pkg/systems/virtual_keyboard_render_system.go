package systems

import (
	"image/color"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// 虚拟键盘视觉常量
var (
	// 键盘背景颜色（半透明深色）
	keyboardBackgroundColor = color.RGBA{R: 30, G: 30, B: 40, A: 230}

	// 按键正常状态颜色
	keyNormalColor = color.RGBA{R: 70, G: 70, B: 80, A: 255}

	// 按键按下状态颜色
	keyPressedColor = color.RGBA{R: 40, G: 40, B: 50, A: 255}

	// 特殊按键颜色（Shift, 123/ABC）
	keySpecialColor = color.RGBA{R: 60, G: 60, B: 70, A: 255}

	// Shift 激活时的颜色
	keyShiftActiveColor = color.RGBA{R: 80, G: 100, B: 120, A: 255}

	// 确定按键颜色
	keyDoneColor = color.RGBA{R: 60, G: 100, B: 60, A: 255}

	// 按键边框颜色
	keyBorderColor = color.RGBA{R: 100, G: 100, B: 110, A: 255}

	// 按键文字颜色
	keyTextColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}

	// 按键圆角半径
	keyBorderRadius = float32(6.0)
)

// 虚拟键盘字体大小
const virtualKeyboardFontSize = 24.0

// VirtualKeyboardRenderSystem 虚拟键盘渲染系统
type VirtualKeyboardRenderSystem struct {
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager
	keyFont         *text.GoTextFace // TrueType 字体
}

// NewVirtualKeyboardRenderSystem 创建虚拟键盘渲染系统
func NewVirtualKeyboardRenderSystem(em *ecs.EntityManager, rm *game.ResourceManager) *VirtualKeyboardRenderSystem {
	sys := &VirtualKeyboardRenderSystem{
		entityManager:   em,
		resourceManager: rm,
	}

	// 加载 TrueType 字体（SimHei 支持中文和英文）
	keyFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", virtualKeyboardFontSize)
	if err != nil {
		log.Printf("[VirtualKeyboardRenderSystem] Warning: Failed to load SimHei font: %v", err)
	} else {
		sys.keyFont = keyFont
		log.Printf("[VirtualKeyboardRenderSystem] TrueType font loaded successfully (size=%.0f)", virtualKeyboardFontSize)
	}

	return sys
}

// Draw 渲染虚拟键盘
func (s *VirtualKeyboardRenderSystem) Draw(screen *ebiten.Image) {
	keyboards := ecs.GetEntitiesWith1[*components.VirtualKeyboardComponent](s.entityManager)

	for _, kbEntity := range keyboards {
		kb, ok := ecs.GetComponent[*components.VirtualKeyboardComponent](s.entityManager, kbEntity)
		if !ok || !kb.IsVisible {
			continue
		}

		s.drawKeyboard(screen, kb)
	}
}

// drawKeyboard 绘制键盘
func (s *VirtualKeyboardRenderSystem) drawKeyboard(screen *ebiten.Image, kb *components.VirtualKeyboardComponent) {
	// 计算键盘背景区域
	layout := entities.CalculateKeyboardLayout(kb)
	if len(layout) == 0 {
		return
	}

	// 计算键盘背景尺寸
	lastRow := layout[len(layout)-1]
	if len(lastRow) == 0 {
		return
	}

	keyboardTop := kb.KeyboardY - 10 // 上边距
	keyboardBottom := lastRow[0].Y + lastRow[0].Height + 10
	keyboardHeight := keyboardBottom - keyboardTop

	// 绘制键盘背景
	vector.DrawFilledRect(
		screen,
		0,
		float32(keyboardTop),
		float32(kb.ScreenWidth),
		float32(keyboardHeight),
		keyboardBackgroundColor,
		true,
	)

	// 绘制所有按键
	allKeys := entities.GetAllKeys(kb)
	for _, key := range allKeys {
		s.drawKey(screen, kb, &key)
	}
}

// drawKey 绘制单个按键
func (s *VirtualKeyboardRenderSystem) drawKey(screen *ebiten.Image, kb *components.VirtualKeyboardComponent, key *components.KeyInfo) {
	// 确定按键颜色
	bgColor := s.getKeyBackgroundColor(kb, key)

	// 绘制按键背景（圆角矩形）
	s.drawRoundedRect(screen, float32(key.X), float32(key.Y), float32(key.Width), float32(key.Height), keyBorderRadius, bgColor)

	// 绘制按键边框
	s.drawRoundedRectBorder(screen, float32(key.X), float32(key.Y), float32(key.Width), float32(key.Height), keyBorderRadius, keyBorderColor)

	// 绘制按键文字
	s.drawKeyLabel(screen, key)
}

// getKeyBackgroundColor 获取按键背景颜色
func (s *VirtualKeyboardRenderSystem) getKeyBackgroundColor(kb *components.VirtualKeyboardComponent, key *components.KeyInfo) color.RGBA {
	// 检查是否被按下
	if kb.PressedKey == key.Action {
		return keyPressedColor
	}

	// 特殊按键颜色
	switch key.Action {
	case "SHIFT":
		if kb.ShiftActive {
			return keyShiftActiveColor
		}
		return keySpecialColor
	case "123", "ABC":
		return keySpecialColor
	case "DONE":
		return keyDoneColor
	case "BACKSPACE":
		return keySpecialColor
	default:
		return keyNormalColor
	}
}

// drawRoundedRect 绘制填充圆角矩形
func (s *VirtualKeyboardRenderSystem) drawRoundedRect(screen *ebiten.Image, x, y, width, height, radius float32, clr color.RGBA) {
	// 使用 vector.DrawFilledRect 绘制主体矩形
	// 由于 Ebitengine 的 vector 包不直接支持圆角，我们使用普通矩形
	vector.DrawFilledRect(screen, x, y, width, height, clr, true)
}

// drawRoundedRectBorder 绘制圆角矩形边框
func (s *VirtualKeyboardRenderSystem) drawRoundedRectBorder(screen *ebiten.Image, x, y, width, height, radius float32, clr color.RGBA) {
	strokeWidth := float32(1.0)
	// 上边
	vector.StrokeLine(screen, x, y, x+width, y, strokeWidth, clr, true)
	// 下边
	vector.StrokeLine(screen, x, y+height, x+width, y+height, strokeWidth, clr, true)
	// 左边
	vector.StrokeLine(screen, x, y, x, y+height, strokeWidth, clr, true)
	// 右边
	vector.StrokeLine(screen, x+width, y, x+width, y+height, strokeWidth, clr, true)
}

// drawKeyLabel 绘制按键标签
func (s *VirtualKeyboardRenderSystem) drawKeyLabel(screen *ebiten.Image, key *components.KeyInfo) {
	label := key.Label
	if label == "" {
		// 空格键显示下划线或不显示
		return
	}

	// 计算文字居中位置
	centerX := key.X + key.Width/2
	centerY := key.Y + key.Height/2

	// 使用 TrueType 字体绘制
	if s.keyFont != nil {
		// 测量文本宽度和高度
		textWidth, textHeight := text.Measure(label, s.keyFont, 0)

		// 计算左上角位置（居中）
		textX := centerX - textWidth/2
		textY := centerY - textHeight/2

		// 绘制文本
		op := &text.DrawOptions{}
		op.GeoM.Translate(textX, textY)
		op.ColorScale.ScaleWithColor(keyTextColor)
		text.Draw(screen, label, s.keyFont, op)
	} else {
		// 如果字体不可用，使用简单的文字绘制（调试用）
		s.drawSimpleText(screen, label, centerX, centerY)
	}
}

// drawSimpleText 简单文字绘制（备用方案）
func (s *VirtualKeyboardRenderSystem) drawSimpleText(screen *ebiten.Image, text string, x, y float64) {
	// 简单实现：绘制一个小矩形表示文字位置（调试用）
	vector.DrawFilledRect(screen, float32(x-2), float32(y-2), 4, 4, keyTextColor, true)
}
