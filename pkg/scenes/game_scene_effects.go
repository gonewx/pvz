package scenes

import (
	"image/color"
	"log"
	"math"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// drawLawnFlash 绘制草坪闪烁效果（Story 8.2 教学）
// Story 8.2.1 修复：使用草皮的Alpha蒙版来实现遮罩闪烁
// 草皮边缘不规则，需要用蒙版图片（如 sod1row_.png）来确保遮罩只覆盖草皮部分
func (s *GameScene) drawLawnFlash(screen *ebiten.Image) {
	alpha := s.lawnGridSystem.GetFlashAlpha()
	if alpha <= 0 {
		if !s.lawnFlashLogged {
			log.Printf("[GameScene] drawLawnFlash: alpha=%.3f (<=0), returning early", alpha)
			s.lawnFlashLogged = true
		}
		return // 没有闪烁效果，直接返回
	}

	// 获取启用的行列表
	enabledLanes := s.lawnGridSystem.EnabledLanes
	if len(enabledLanes) == 0 {
		log.Printf("[GameScene] drawLawnFlash: no enabled lanes, returning")
		return // 没有启用的行
	}

	// Story 8.2.1：使用草皮图像及其Alpha蒙版
	// 检查是否有草皮图像
	if s.sodRowImage == nil {
		log.Printf("[GameScene] drawLawnFlash: sodRowImage is nil, returning")
		return // 没有草皮图像，无需闪烁
	}

	// DEBUG: 检查 preSoddedImage (如果使用预渲染背景)
	if s.preSoddedImage != nil {
		log.Printf("[GameScene] drawLawnFlash: WARNING - preSoddedImage exists, sodRowImage may have been cleared after animation")
	}

	// 使用缓存的草皮尺寸和位置
	sodWorldX := s.sodOverlayX
	sodWorldY := s.sodOverlayY

	// Story 8.2.1 调试：记录草皮闪烁信息（只记录一次）
	if !s.lawnFlashLogged {
		log.Printf("[GameScene] drawLawnFlash: alpha=%.3f, sodPos=(%.1f,%.1f), using sodRowImage with alpha mask",
			alpha, sodWorldX, sodWorldY)
		log.Printf("[GameScene] drawLawnFlash: worldPos=(%.1f,%.1f), cameraX=%.1f, screenPos=(%.1f,%.1f)",
			sodWorldX, sodWorldY, s.cameraX, sodWorldX-s.cameraX, sodWorldY)
		s.lawnFlashLogged = true
	}

	// 转换为屏幕坐标
	screenX := sodWorldX - s.cameraX
	screenY := sodWorldY

	// Story 8.2.1: 使用草皮图像本身作为遮罩基础
	// 创建一个与草皮图像相同尺寸的黑色图像
	// 然后应用草皮的Alpha通道作为遮罩
	sodBounds := s.sodRowImage.Bounds()
	flashImage := ebiten.NewImage(sodBounds.Dx(), sodBounds.Dy())
	flashImage.Fill(color.RGBA{0, 0, 0, 255}) // 纯黑色

	// 使用草皮图像的Alpha通道作为遮罩
	// DrawImageOptions.CompositeMode 设置为 DestinationIn 模式
	// 这会保留黑色，但使用源图像的Alpha通道
	maskOp := &ebiten.DrawImageOptions{}
	maskOp.CompositeMode = ebiten.CompositeModeDestinationIn
	flashImage.DrawImage(s.sodRowImage, maskOp)

	// 应用闪烁Alpha到整个遮罩
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(screenX, screenY)
	op.ColorScale.ScaleAlpha(float32(alpha))
	screen.DrawImage(flashImage, op)
}

// drawCardFlash 绘制卡片闪烁效果（Story 8.2.1 教学）
// 遮罩式闪烁：在高亮的卡片上绘制黑色半透明遮罩，通过Alpha变化实现明暗闪烁
// 使用正弦波实现平滑的明暗变化（与草皮闪烁相同原理）
func (s *GameScene) drawCardFlash(screen *ebiten.Image) {
	// 查找教学组件
	tutorialEntities := ecs.GetEntitiesWith1[*components.TutorialComponent](s.entityManager)
	if len(tutorialEntities) == 0 {
		return
	}

	tutorial, ok := ecs.GetComponent[*components.TutorialComponent](s.entityManager, tutorialEntities[0])
	if !ok || tutorial.HighlightedCardEntity == 0 {
		return // 没有高亮卡片
	}

	// 获取高亮卡片的位置和组件
	pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, tutorial.HighlightedCardEntity)
	if !ok {
		return
	}

	card, ok := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, tutorial.HighlightedCardEntity)
	if !ok {
		return
	}

	// 计算闪烁 Alpha（0.0 - 0.3，使用正弦波）
	// 与草皮闪烁相同：使用黑色遮罩，Alpha变化实现明暗闪烁
	// Alpha 0.0 = 完全透明（卡片正常亮度）
	// Alpha 0.3 = 黑色遮罩（卡片变暗）
	const maxAlpha = 0.3
	phase := 2 * math.Pi * tutorial.FlashTimer / tutorial.FlashCycleDuration
	alpha := maxAlpha * (0.5 + 0.5*math.Sin(phase))

	// 卡片尺寸（使用实际的缩放因子）
	const cardOriginalWidth = 100.0
	const cardOriginalHeight = 140.0
	cardWidth := cardOriginalWidth * card.CardScale
	cardHeight := cardOriginalHeight * card.CardScale

	// 创建黑色半透明遮罩（与草皮闪烁相同）
	flashImage := ebiten.NewImage(int(cardWidth), int(cardHeight))
	flashImage.Fill(color.RGBA{0, 0, 0, uint8(alpha * 255)}) // 黑色，alpha 0.0-0.3

	// 绘制到屏幕
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(pos.X, pos.Y)
	screen.DrawImage(flashImage, op)
}

// drawBowlingRedLine 绘制保龄球关卡红线（Story 19.4）
// 在第 3 列和第 4 列之间绘制红色条纹，用于保龄球阶段视觉提示
//
// 渲染层级：在草坪背景之上、植物之下
// 红线位置：config.GridWorldStartX + config.BowlingRedLineColumn * config.CellWidth
//
// 条件：只有当 LevelPhaseSystem.ShouldShowRedLine() 返回 true 时才绘制
func (s *GameScene) drawBowlingRedLine(screen *ebiten.Image) {
	// 检查 LevelPhaseSystem 是否存在
	if s.levelPhaseSystem == nil {
		return
	}

	// 检查是否应该显示红线
	if !s.levelPhaseSystem.ShouldShowRedLine() {
		return
	}

	// 检查红线图片是否已加载
	if s.bowlingRedLine == nil {
		log.Printf("[GameScene] Warning: Bowling red line image not loaded")
		return
	}

	// 计算红线位置（世界坐标）
	// 红线位于第 3 列和第 4 列之间
	redLineWorldX := config.GridWorldStartX + float64(config.BowlingRedLineColumn)*config.CellWidth

	// 获取红线图片尺寸
	redLineBounds := s.bowlingRedLine.Bounds()
	redLineWidth := float64(redLineBounds.Dx())
	redLineHeight := float64(redLineBounds.Dy())

	// 红线 Y 位置：从草坪顶部开始，覆盖整个草坪高度
	redLineWorldY := config.GridWorldStartY + config.BowlingRedLineOffsetY

	// 如果红线图片高度不够覆盖整个草坪，需要拉伸
	// 否则直接使用原始图片
	totalLawnHeight := float64(config.GridRows) * config.CellHeight

	// 转换为屏幕坐标
	screenX := redLineWorldX - s.cameraX - redLineWidth/2 // 居中对齐到列线
	screenY := redLineWorldY

	// 绘制红线
	op := &ebiten.DrawImageOptions{}

	// 如果需要拉伸高度
	if redLineHeight < totalLawnHeight {
		scaleY := totalLawnHeight / redLineHeight
		op.GeoM.Scale(1, scaleY)
	}

	op.GeoM.Translate(screenX, screenY)
	screen.DrawImage(s.bowlingRedLine, op)
}
