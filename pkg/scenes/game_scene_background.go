package scenes

import (
	"image"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

// drawBackground renders the lawn background.
// The background image is larger than the window, and we display only a portion of it.
// The cameraX value determines which horizontal section of the background is visible.
// During the intro animation, the camera scrolls left → right → center to showcase the entire scene.
func (s *GameScene) drawBackground(screen *ebiten.Image) {
	if s.background != nil {
		// Get background image dimensions
		bounds := s.background.Bounds()
		bgWidth := bounds.Dx()
		bgHeight := bounds.Dy()

		// Calculate the viewport rectangle based on camera position
		// We want to show a WindowWidth x WindowHeight portion of the background
		viewportX := int(s.cameraX)
		viewportY := 0

		// Ensure we don't go out of bounds
		if viewportX+WindowWidth > bgWidth {
			viewportX = bgWidth - WindowWidth
		}
		if viewportX < 0 {
			viewportX = 0
		}

		// If background is smaller than window height, center it vertically
		if bgHeight > WindowHeight {
			// Center the viewport vertically if background is taller
			viewportY = (bgHeight - WindowHeight) / 2
		}

		// Create a sub-image representing the visible portion
		viewportRect := image.Rect(
			viewportX,
			viewportY,
			viewportX+WindowWidth,
			viewportY+WindowHeight,
		)

		// Extract the visible portion of the background
		visibleBG := s.background.SubImage(viewportRect).(*ebiten.Image)

		// Draw the visible portion at (0, 0)
		op := &ebiten.DrawImageOptions{}
		screen.DrawImage(visibleBG, op)

		// 统一草皮渲染：所有关卡使用双背景叠加模式
		s.drawSoddingOverlay(screen, viewportX, viewportY)
	} else {
		// Fallback: Draw a green background to simulate grass
		screen.Fill(color.RGBA{R: 34, G: 139, B: 34, A: 255}) // Forest green
	}
}

// drawSoddingOverlay draws sodding animation overlay on background.
// 统一草皮渲染：所有关卡使用双背景叠加模式
//
// 设计原理：
// - 底层：未铺草皮背景 (IMAGE_BACKGROUND1UNSODDED) + 预渲染的preSoddedLanes草皮
// - 叠加层：预渲染背景副本 (preSoddedImage) 或完整铺草皮背景 (soddedBackground)
// - 根据草皮卷位置 (sodRollCenterX)，渐进显示叠加层（从世界坐标0裁剪到草皮卷中心）
//
// 优势：
// - 统一逻辑：1-1/1-2/1-4 使用相同代码路径
// - 坐标清晰：所有计算使用世界坐标，最后转屏幕坐标
// - 易于维护：无需区分单行/连续行/全行模式
func (s *GameScene) drawSoddingOverlay(screen *ebiten.Image, viewportX, viewportY int) {
	if s.soddingSystem == nil || s.gameState.CurrentLevel == nil {
		return
	}

	// 选择叠加背景（优先使用完整背景，否则使用预渲染背景）
	overlayBg := s.soddedBackground
	if overlayBg == nil {
		overlayBg = s.preSoddedImage
	}

	if overlayBg == nil {
		return
	}

	// 获取草皮卷当前位置（世界坐标X）
	sodRollCenterX := s.soddingSystem.GetSodRollCenterX()
	animProgress := s.soddingSystem.GetProgress()

	// 计算可见宽度（从世界坐标 0 到草皮卷中心）
	visibleWorldWidth := int(sodRollCenterX)

	// 重要修复：动画未启动时，不应显示叠加层
	// 底层背景 (s.background) 已经预渲染了 preSoddedLanes 草皮
	// 叠加层 (preSoddedImage) 包含所有启用行的草皮，应该在动画启动后才渐进显示
	//
	// Level 1-1: preSoddedLanes=[] → 动画前底层无草皮，动画开始后显示第3行
	// Level 1-2: preSoddedLanes=[3] → 动画前底层已有第3行草皮，动画开始后显示2,4行
	//
	// 因此：动画未启动时，直接返回不显示叠加层（底层背景已包含预铺草皮）
	if !s.soddingSystem.HasStarted() {
		return
	}

	// 优化：动画接近完成时（≥99%），直接显示完整叠加层，避免切换时闪烁
	if animProgress >= 0.99 {
		bgBounds := overlayBg.Bounds()
		visibleWorldWidth = bgBounds.Dx()
	}

	// 只有草皮卷到达可见位置后才渲染叠加层
	if visibleWorldWidth <= 0 {
		return
	}

	// 获取叠加背景尺寸
	bgBounds := overlayBg.Bounds()
	bgWidth := bgBounds.Dx()
	bgHeight := bgBounds.Dy()

	// 限制可见宽度不超过背景宽度
	if visibleWorldWidth > bgWidth {
		visibleWorldWidth = bgWidth
	}

	// 计算视口裁剪区域（世界坐标）
	// viewportX 是摄像机在世界坐标中的位置
	overlayViewportX := viewportX
	overlayViewportY := viewportY

	// 水平方向：裁剪从 viewportX 到 min(visibleWorldWidth, viewportX + WindowWidth)
	overlayViewportEndX := visibleWorldWidth
	if overlayViewportX < 0 {
		overlayViewportX = 0
	}

	// 不能超过可见宽度
	if overlayViewportEndX < overlayViewportX {
		return
	}

	// 垂直方向：显示整个高度
	overlayViewportEndY := overlayViewportY + WindowHeight
	if overlayViewportEndY > bgHeight {
		overlayViewportEndY = bgHeight
	}
	if overlayViewportY < 0 {
		overlayViewportY = 0
	}

	// 裁剪叠加背景的可见部分（世界坐标裁剪）
	overlayRect := image.Rect(
		overlayViewportX,
		overlayViewportY,
		overlayViewportEndX,
		overlayViewportEndY,
	)

	visibleOverlay := overlayBg.SubImage(overlayRect).(*ebiten.Image)

	// 世界坐标 → 屏幕坐标转换
	screenX := float64(overlayViewportX) - float64(viewportX)
	screenY := 0.0 // Y轴无摄像机移动

	// 绘制叠加背景到屏幕
	overlayOp := &ebiten.DrawImageOptions{}
	overlayOp.GeoM.Translate(screenX, screenY)
	screen.DrawImage(visibleOverlay, overlayOp)

	// DEBUG: 打印叠加背景信息（每10帧输出一次）
	if frameIndex := int(s.soddingSystem.GetProgress() * 48); frameIndex%10 == 0 || frameIndex == 0 || frameIndex >= 47 {
		log.Printf("[草皮叠加] 帧:%d, 可见世界宽度: %d px, sodRollCenterX: %.1f, 差值: %.1f px",
			frameIndex, visibleWorldWidth, sodRollCenterX, sodRollCenterX-float64(visibleWorldWidth))
	}
}

// createMergedBackground 创建合并了草皮叠加层的背景图片
// 在铺草皮动画完成后调用，将底层背景和草皮叠加层合并成一个新的完整背景
// 返回合并后的背景图片，如果失败返回 nil
func (s *GameScene) createMergedBackground() *ebiten.Image {
	if s.background == nil {
		log.Printf("[createMergedBackground] 错误：底层背景为空")
		return nil
	}

	// 选择叠加图层（preSoddedImage 或 sodRowImage）
	var overlayImg *ebiten.Image
	if s.preSoddedImage != nil {
		overlayImg = s.preSoddedImage
		log.Printf("[createMergedBackground] 使用 preSoddedImage 作为叠加层")
	} else if s.sodRowImage != nil {
		overlayImg = s.sodRowImage
		log.Printf("[createMergedBackground] 使用 sodRowImage 作为叠加层")
	} else {
		log.Printf("[createMergedBackground] 错误：没有可用的草皮叠加层")
		return nil
	}

	// 获取背景尺寸
	bgBounds := s.background.Bounds()
	bgWidth := bgBounds.Dx()
	bgHeight := bgBounds.Dy()

	log.Printf("[createMergedBackground] 创建合并背景: 尺寸 %dx%d", bgWidth, bgHeight)

	// 创建新的背景图片
	mergedBg := ebiten.NewImage(bgWidth, bgHeight)

	// 1. 绘制底层背景
	op := &ebiten.DrawImageOptions{}
	mergedBg.DrawImage(s.background, op)

	// 2. 绘制草皮叠加层
	// 使用 preSoddedImage 时，它已经包含了正确位置的草皮
	// 使用 sodRowImage 时，需要根据配置的位置绘制
	if s.preSoddedImage != nil {
		// preSoddedImage 已经是完整的背景副本，直接使用
		mergedBg.DrawImage(overlayImg, op)
	} else if s.sodRowImage != nil {
		// sodRowImage 需要放置在正确的位置
		overlayOp := &ebiten.DrawImageOptions{}
		overlayOp.GeoM.Translate(float64(s.sodOverlayX), float64(s.sodOverlayY))
		mergedBg.DrawImage(overlayImg, overlayOp)
	}

	log.Printf("[createMergedBackground] 成功创建合并背景")
	return mergedBg
}
