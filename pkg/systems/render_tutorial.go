// render_tutorial.go - 教学文本渲染相关方法
//
// 本文件包含 RenderSystem 的教学文本渲染功能：
//   - DrawTutorialText: 绘制教学提示文本
//   - UpdateTutorialTextTime: 更新教学文本显示时间
//   - drawCenteredTextTTF: 使用 TrueType 字体绘制居中文本
//
// 所有方法都是 RenderSystem 的成员方法（接收者：*RenderSystem）。
// 使用相同的 package systems，可以直接访问 RenderSystem 的私有字段。

package systems

import (
	"image/color"
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// DrawTutorialText 绘制教学文本（Story 8.2 - TrueType 字体版本）
// 在屏幕底部中央显示教学提示文本，带半透明黑色背景条
// 参数:
//   - screen: 绘制目标屏幕
//   - tutorialFont: 教学字体（SimHei.ttf 或其他 TrueType 字体）
//   - bowlingFont: Level 1-5 保龄球关卡专用大字体（可为 nil）
func (s *RenderSystem) DrawTutorialText(screen *ebiten.Image, tutorialFont interface{}, bowlingFont interface{}) {
	// 查询教学文本实体
	textEntities := ecs.GetEntitiesWith1[*components.TutorialTextComponent](s.entityManager)

	if len(textEntities) == 0 {
		return // 无教学文本实体
	}

	for _, entity := range textEntities {
		textComp, ok := ecs.GetComponent[*components.TutorialTextComponent](s.entityManager, entity)
		if !ok {
			continue
		}

		// 如果文本为空，跳过渲染
		if textComp.Text == "" {
			continue
		}

		// 获取屏幕尺寸
		screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

		// 根据教学类型选择位置配置和字体
		var bgOffsetFromBottom, textOffsetFromBottom, bgHeight float64
		var activeFont interface{}
		if textComp.IsBowling {
			// Level 1-5 保龄球关卡：专用配置（更小背景、更靠下、更大字体）
			bgOffsetFromBottom = config.BowlingTutorialTextBackgroundOffsetFromBottom
			textOffsetFromBottom = config.BowlingTutorialTextOffsetFromBottom
			bgHeight = config.BowlingTutorialTextBackgroundHeight
			// 优先使用保龄球专用字体
			if bowlingFont != nil {
				activeFont = bowlingFont
			} else {
				activeFont = tutorialFont
			}
		} else if textComp.IsAdvisory {
			// 提示性教学（Level 1-2）：更靠下
			bgOffsetFromBottom = config.AdvisoryTutorialTextBackgroundOffsetFromBottom
			textOffsetFromBottom = config.AdvisoryTutorialTextOffsetFromBottom
			bgHeight = config.TutorialTextBackgroundHeight
			activeFont = tutorialFont
		} else {
			// 强制性教学（Level 1-1）：标准位置
			bgOffsetFromBottom = config.TutorialTextBackgroundOffsetFromBottom
			textOffsetFromBottom = config.TutorialTextOffsetFromBottom
			bgHeight = config.TutorialTextBackgroundHeight
			activeFont = tutorialFont
		}

		// 绘制半透明黑色背景条（横贯整个屏幕）
		bgY := float64(screenHeight) - bgOffsetFromBottom
		ebitenutil.DrawRect(screen, 0, bgY, float64(screenWidth), bgHeight,
			color.RGBA{0, 0, 0, uint8(config.TutorialTextBackgroundAlpha)})

		// 计算文本位置（底部中央）
		textX := float64(screenWidth) / 2
		textY := float64(screenHeight) - textOffsetFromBottom

		// 检查是否为 TrueType 字体
		if ttFont, ok := activeFont.(*text.GoTextFace); ok && ttFont != nil {
			// 使用 TrueType 字体绘制（浅黄色文字 + 黑色描边）
			s.drawCenteredTextTTF(screen, textComp.Text, textX, textY, ttFont)
		} else if bFont, ok := activeFont.(*utils.BitmapFont); ok && bFont != nil {
			// 备选：位图字体（不支持中文，已废弃）
			log.Printf("[RenderSystem] WARNING: BitmapFont does not support Chinese, using fallback")
			bFont.DrawText(screen, textComp.Text, textX, textY, "center")
		} else {
			log.Printf("[RenderSystem] ERROR: Unknown font type or nil font!")
		}
	}
}

// UpdateTutorialTextTime 更新所有 TutorialTextComponent 的显示时间
// 如果超过 MaxDisplayTime 则自动销毁实体
// 参数：
//   - dt: 时间增量（秒）
func (s *RenderSystem) UpdateTutorialTextTime(dt float64) {
	textEntities := ecs.GetEntitiesWith1[*components.TutorialTextComponent](s.entityManager)

	for _, entity := range textEntities {
		textComp, ok := ecs.GetComponent[*components.TutorialTextComponent](s.entityManager, entity)
		if !ok {
			continue
		}

		// 更新显示时间
		textComp.DisplayTime += dt

		// 如果有最大显示时间限制，检查是否需要销毁
		if textComp.MaxDisplayTime > 0 && textComp.DisplayTime >= textComp.MaxDisplayTime {
			s.entityManager.DestroyEntity(entity)
			log.Printf("[RenderSystem] 教学文本自动消失: text='%s'", textComp.Text)
		}
	}
}

// drawCenteredTextTTF 使用 TrueType 字体绘制居中文本（带黑色描边）
// 教学文本效果：浅黄色文字 + 黑色描边
// 参数:
//   - screen: 绘制目标屏幕
//   - textStr: 文本内容
//   - centerX: 文本中心X坐标
//   - centerY: 文本中心Y坐标
//   - fontFace: TrueType 字体
func (s *RenderSystem) drawCenteredTextTTF(screen *ebiten.Image, textStr string, centerX, centerY float64, fontFace *text.GoTextFace) {
	// 测量文本宽度
	width, _ := text.Measure(textStr, fontFace, 0)

	// 计算左上角坐标（居中对齐）
	x := centerX - width/2
	y := centerY

	// Step 1: 绘制黑色描边（在8个方向偏移1-2像素）
	strokeColor := color.RGBA{R: 0, G: 0, B: 0, A: 255} // 黑色
	strokeOffsets := []struct{ dx, dy float64 }{
		{-1, -1}, {0, -1}, {1, -1}, // 上
		{-1, 0}, {1, 0}, // 左右
		{-1, 1}, {0, 1}, {1, 1}, // 下
	}

	for _, offset := range strokeOffsets {
		op := &text.DrawOptions{}
		op.GeoM.Translate(x+offset.dx, y+offset.dy)
		op.ColorScale.ScaleWithColor(strokeColor)
		text.Draw(screen, textStr, fontFace, op)
	}

	// Step 2: 绘制浅黄色主文本（在中心）
	// 使用浅黄色 RGB(255, 242, 0)
	textColor := color.RGBA{R: 238, G: 232, B: 170, A: 0}
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, textStr, fontFace, op)
}
