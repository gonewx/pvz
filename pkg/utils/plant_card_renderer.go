package utils

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// PlantCardRenderOptions 定义植物卡片渲染的所有可配置选项。
// 该结构体支持灵活的渲染配置，可用于选卡界面、奖励面板、图鉴等多种场景。
type PlantCardRenderOptions struct {
	// 必需字段
	Screen          *ebiten.Image // 渲染目标屏幕
	X, Y            float64       // 卡片左上角位置（世界坐标）
	BackgroundImage *ebiten.Image // 卡片背景框图片
	SunCost         int           // 阳光消耗数字

	// 可选字段 - 植物图标
	PlantIconImage  *ebiten.Image // 植物图标图片（nil 表示不绘制图标）
	PlantIconScale  float64       // 植物图标缩放比例（默认 1.0）
	PlantIconOffsetY float64      // 植物图标 Y 轴偏移（相对于卡片顶部）

	// 可选字段 - 阳光文字
	SunFont        *text.GoTextFaceSource // 阳光数字字体（nil 表示使用调试文本）
	SunFontSize    float64                // 阳光字体大小（默认 12.0）
	SunTextOffsetY float64                // 阳光文字 Y 轴偏移（相对于卡片底部）
	SunTextColor   color.Color            // 阳光文字颜色（默认黑色）

	// 可选字段 - 缩放和效果
	CardScale        float64 // 卡片整体缩放比例（默认 1.0）
	CooldownProgress float64 // 冷却进度（0.0-1.0，0 表示无冷却）
	IsDisabled       bool    // 是否禁用（显示全屏淡遮罩）
	Alpha            float64 // 整体透明度（0.0-1.0，默认 1.0）
}

// PlantCardRenderer 提供植物卡片的通用渲染功能。
// 该工具类用于消除渲染系统间的重复代码，提高可维护性和扩展性。
//
// 使用示例：
//
//	renderer := NewPlantCardRenderer()
//	renderer.Render(PlantCardRenderOptions{
//	    Screen:          screen,
//	    X:               100,
//	    Y:               200,
//	    BackgroundImage: cardBg,
//	    PlantIconImage:  icon,
//	    SunCost:         100,
//	    SunFont:         font,
//	    CardScale:       0.8,
//	})
type PlantCardRenderer struct{}

// NewPlantCardRenderer 创建一个新的植物卡片渲染器。
func NewPlantCardRenderer() *PlantCardRenderer {
	return &PlantCardRenderer{}
}

// Render 根据提供的选项渲染植物卡片。
// 渲染顺序：背景框 → 植物图标 → 阳光数字 → 效果遮罩。
func (r *PlantCardRenderer) Render(opts PlantCardRenderOptions) {
	// 应用默认值
	r.applyDefaults(&opts)

	// 计算缩放后的卡片尺寸（用于后续居中和遮罩）
	cardWidth, cardHeight := r.getScaledCardSize(opts)

	// 层1: 绘制卡片背景框
	r.drawBackground(opts)

	// 层2: 绘制植物图标（如果提供）
	if opts.PlantIconImage != nil {
		r.drawPlantIcon(opts, cardWidth, cardHeight)
	}

	// 层3: 绘制阳光数字
	r.drawSunCost(opts, cardWidth, cardHeight)

	// 层4: 绘制效果遮罩（冷却/禁用）
	r.drawEffectMask(opts, cardWidth, cardHeight)
}

// applyDefaults 为未设置的可选字段应用默认值。
func (r *PlantCardRenderer) applyDefaults(opts *PlantCardRenderOptions) {
	if opts.CardScale == 0 {
		opts.CardScale = 1.0
	}
	if opts.PlantIconScale == 0 {
		opts.PlantIconScale = 1.0
	}
	if opts.Alpha == 0 {
		opts.Alpha = 1.0
	}
	if opts.SunFontSize == 0 {
		opts.SunFontSize = 12.0
	}
	if opts.SunTextColor == nil {
		opts.SunTextColor = color.RGBA{0, 0, 0, 255} // 默认黑色
	}
}

// getScaledCardSize 计算缩放后的卡片尺寸。
func (r *PlantCardRenderer) getScaledCardSize(opts PlantCardRenderOptions) (width, height float64) {
	if opts.BackgroundImage != nil {
		bounds := opts.BackgroundImage.Bounds()
		width = float64(bounds.Dx()) * opts.CardScale
		height = float64(bounds.Dy()) * opts.CardScale
	}
	return
}

// drawBackground 绘制卡片背景框。
func (r *PlantCardRenderer) drawBackground(opts PlantCardRenderOptions) {
	if opts.BackgroundImage == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(opts.CardScale, opts.CardScale)
	op.GeoM.Translate(opts.X, opts.Y)
	op.ColorScale.ScaleAlpha(float32(opts.Alpha))

	opts.Screen.DrawImage(opts.BackgroundImage, op)
}

// drawPlantIcon 绘制植物图标（居中对齐）。
func (r *PlantCardRenderer) drawPlantIcon(opts PlantCardRenderOptions, cardWidth, cardHeight float64) {
	if opts.PlantIconImage == nil {
		return
	}

	iconBounds := opts.PlantIconImage.Bounds()
	iconWidth := float64(iconBounds.Dx()) * opts.PlantIconScale

	// 计算居中偏移
	offsetX := (cardWidth - iconWidth) / 2.0
	offsetY := opts.PlantIconOffsetY

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(opts.PlantIconScale, opts.PlantIconScale)
	op.GeoM.Translate(opts.X+offsetX, opts.Y+offsetY)
	op.ColorScale.ScaleAlpha(float32(opts.Alpha))

	opts.Screen.DrawImage(opts.PlantIconImage, op)
}

// drawSunCost 绘制阳光消耗数字（底部居中）。
func (r *PlantCardRenderer) drawSunCost(opts PlantCardRenderOptions, cardWidth, cardHeight float64) {
	sunText := fmt.Sprintf("%d", opts.SunCost)

	if opts.SunFont != nil {
		// 使用自定义字体渲染
		face := &text.GoTextFace{
			Source: opts.SunFont,
			Size:   opts.SunFontSize,
		}

		textWidth, _ := text.Measure(sunText, face, 0)

		// 底部居中位置
		textX := opts.X + (cardWidth-textWidth)/2.0
		textY := opts.Y + cardHeight - opts.SunTextOffsetY

		op := &text.DrawOptions{}
		op.GeoM.Translate(textX, textY)
		op.ColorScale.ScaleWithColor(opts.SunTextColor)
		op.ColorScale.ScaleAlpha(float32(opts.Alpha))

		text.Draw(opts.Screen, sunText, face, op)
	} else {
		// 使用调试文本（临时方案）
		textX := int(opts.X + cardWidth/2.0 - 10.0) // 粗略居中
		textY := int(opts.Y + cardHeight - opts.SunTextOffsetY)
		ebitenutil.DebugPrintAt(opts.Screen, sunText, textX, textY)
	}
}

// drawEffectMask 绘制效果遮罩（冷却进度或禁用状态）。
func (r *PlantCardRenderer) drawEffectMask(opts PlantCardRenderOptions, cardWidth, cardHeight float64) {
	if cardHeight == 0 {
		return
	}

	// 冷却遮罩：从上往下的渐进遮罩
	if opts.CooldownProgress > 0 {
		coverHeight := cardHeight * opts.CooldownProgress
		maskColor := color.RGBA{0, 0, 0, 160} // 半透明黑色
		ebitenutil.DrawRect(opts.Screen, opts.X, opts.Y, cardWidth, coverHeight, maskColor)
	} else if opts.IsDisabled {
		// 禁用遮罩：全屏淡遮罩
		maskColor := color.RGBA{0, 0, 0, 100} // 更淡的黑色
		ebitenutil.DrawRect(opts.Screen, opts.X, opts.Y, cardWidth, cardHeight, maskColor)
	}
}
