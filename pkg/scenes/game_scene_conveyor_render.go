package scenes

import (
	"image"
	"image/color"
	"math"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// ============================================================================
// 传送带渲染相关方法
// Story 15.5: 从 game_scene_conveyor.go 拆分出传送带渲染逻辑（约500行）
// ============================================================================

func (s *GameScene) drawConveyorBelt(screen *ebiten.Image) {
	// 检查传送带是否可见
	if s.levelPhaseSystem == nil || !s.levelPhaseSystem.IsConveyorBeltVisible() {
		return
	}

	// 获取传送带 Y 位置（由 LevelPhaseSystem 控制滑入动画）
	conveyorY := s.levelPhaseSystem.GetConveyorBeltY()

	// 计算传送带 X 位置（紧挨铲子槽位左侧）
	conveyorX := s.calculateConveyorX()

	// 1. 绘制背景
	s.drawConveyorBackdrop(screen, conveyorX, conveyorY)

	// 2. 绘制传动动画
	s.drawConveyorBeltAnimation(screen, conveyorX, conveyorY)

	// 3. 绘制卡片
	s.drawConveyorCards(screen, conveyorX, conveyorY)
}

// calculateConveyorX 计算传送带 X 位置（右边缘紧挨铲子槽位）
func (s *GameScene) calculateConveyorX() float64 {
	// 计算铲子 X 位置（与 drawShovel 保持一致）
	var shovelX float64

	// 保龄球模式（initialSun == 0）使用相对于菜单按钮的位置
	if s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.InitialSun == 0 {
		menuButtonX := float64(WindowWidth) - config.MenuButtonOffsetFromRight
		shovelX = menuButtonX - float64(config.BowlingShovelGapFromMenuButton) - float64(config.ShovelWidth)
	} else if s.seedBank != nil {
		// 普通模式根据选择栏图片宽度动态计算
		seedBankWidth := float64(s.seedBank.Bounds().Dx())
		shovelX = float64(config.SeedBankX) + seedBankWidth + float64(config.ShovelGapFromSeedBank)
	} else {
		shovelX = float64(config.ShovelX) // 默认值
	}

	// 使用实际背景图片宽度来定位，确保右边紧挨铲子卡槽
	backdropWidth := config.ConveyorBeltWidth // 默认值
	if s.conveyorBeltBackdrop != nil {
		backdropWidth = float64(s.conveyorBeltBackdrop.Bounds().Dx())
	}

	// 传送带 X = 铲子 X - 传送带背景宽度
	conveyorX := shovelX - backdropWidth

	return conveyorX
}

// drawConveyorBackdrop 绘制传送带背景
func (s *GameScene) drawConveyorBackdrop(screen *ebiten.Image, x, y float64) {
	if s.conveyorBeltBackdrop == nil {
		// Fallback: 绘制深灰色矩形
		ebitenutil.DrawRect(screen, x, y, config.ConveyorBeltWidth, 80, color.RGBA{R: 60, G: 60, B: 60, A: 255})
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	screen.DrawImage(s.conveyorBeltBackdrop, op)
}

// drawConveyorBeltAnimation 绘制传送带传动动画（6行交错滚动）
func (s *GameScene) drawConveyorBeltAnimation(screen *ebiten.Image, x, y float64) {
	if s.conveyorBelt == nil {
		return
	}

	// 获取传送带组件
	var scrollOffset float64 = 0
	if s.conveyorBeltSystem != nil {
		beltEntity := s.conveyorBeltSystem.GetBeltEntity()
		if beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, beltEntity); ok {
			scrollOffset = beltComp.ScrollOffset
		}
	}

	// 获取履带纹理尺寸
	imgBounds := s.conveyorBelt.Bounds()
	imgWidth := imgBounds.Dx()
	imgHeight := imgBounds.Dy()
	rowHeight := imgHeight / config.ConveyorBeltRowCount

	// 使用履带图片的实际宽度作为渲染宽度
	beltRenderWidth := float64(imgWidth)
	beltRenderX := x + config.ConveyorBeltLeftPadding + config.ConveyorBeltAnimOffsetX
	beltRenderY := y + config.ConveyorBeltLeftPadding + config.ConveyorBeltAnimOffsetY

	// 渲染 6 行交错滚动
	for row := 0; row < config.ConveyorBeltRowCount; row++ {
		// 计算该行的水平偏移
		// 偶数行向左滚动，奇数行向右滚动
		offset := scrollOffset
		if row%2 == 1 {
			offset = -scrollOffset
		}

		// 环绕处理
		offset = math.Mod(offset, float64(imgWidth))
		if offset < 0 {
			offset += float64(imgWidth)
		}

		// 绘制该行（可能需要绘制两次以实现无缝循环）
		rowY := beltRenderY + float64(row*rowHeight)

		// 裁剪源图像区域（该行）
		srcRect := image.Rect(0, row*rowHeight, imgWidth, (row+1)*rowHeight)

		// 第一段
		s.drawBeltRowSegment(screen, srcRect, beltRenderX, rowY, beltRenderWidth, float64(rowHeight), offset)
	}
}

// drawBeltRowSegment 绘制传送带的一行（带水平偏移）
func (s *GameScene) drawBeltRowSegment(screen *ebiten.Image, srcRect image.Rectangle, x, y, width, height, offset float64) {
	if s.conveyorBelt == nil {
		return
	}

	srcWidth := float64(srcRect.Dx())

	// 裁剪目标区域
	// 使用 SubImage 获取该行
	rowImage := s.conveyorBelt.SubImage(srcRect).(*ebiten.Image)

	// 计算需要绘制的段数（可能需要重复绘制以填满宽度）
	startX := -offset
	for startX < width {
		op := &ebiten.DrawImageOptions{}

		// 计算实际绘制位置和裁剪
		drawX := x + startX
		clipLeft := 0.0
		clipRight := srcWidth

		// 左边裁剪
		if drawX < x {
			clipLeft = x - drawX
			drawX = x
		}

		// 右边裁剪
		if drawX+srcWidth-clipLeft > x+width {
			clipRight = x + width - drawX + clipLeft
		}

		// 如果有有效区域则绘制
		if clipRight > clipLeft {
			// 创建裁剪后的子图像
			subSrcRect := image.Rect(int(clipLeft), 0, int(clipRight), srcRect.Dy())
			subImage := rowImage.SubImage(subSrcRect).(*ebiten.Image)

			op.GeoM.Translate(drawX, y)
			screen.DrawImage(subImage, op)
		}

		startX += srcWidth
	}
}

// drawConveyorCards 绘制传送带上的卡片
func (s *GameScene) drawConveyorCards(screen *ebiten.Image, conveyorX, conveyorY float64) {
	if s.conveyorBeltSystem == nil {
		return
	}

	// 获取传送带组件
	beltEntity := s.conveyorBeltSystem.GetBeltEntity()
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, beltEntity)
	if !ok {
		return
	}

	// 卡片尺寸 - 使用缩放比例配置
	var originalCardWidth, originalCardHeight float64
	if s.conveyorCardBackground != nil {
		bgBounds := s.conveyorCardBackground.Bounds()
		originalCardWidth = float64(bgBounds.Dx())
		originalCardHeight = float64(bgBounds.Dy())
	} else {
		originalCardWidth = 100.0
		originalCardHeight = 140.0
	}

	cardScale := config.ConveyorCardScale
	cardWidth := originalCardWidth * cardScale
	cardHeight := originalCardHeight * cardScale

	// 通知系统当前卡片宽度（用于碰撞检测）
	s.conveyorBeltSystem.SetCardWidth(cardWidth)

	// 获取传送带背景尺寸
	var beltHeight, beltWidth float64
	if s.conveyorBeltBackdrop != nil {
		beltHeight = float64(s.conveyorBeltBackdrop.Bounds().Dy())
		beltWidth = float64(s.conveyorBeltBackdrop.Bounds().Dx())
	} else {
		beltHeight = 80.0
		beltWidth = config.ConveyorBeltWidth
	}

	// 垂直位置：垂直居中 + 上边距微调
	cardY := conveyorY + (beltHeight-cardHeight)/2 + config.ConveyorBeltTopPadding

	// 传送带可见区域边界（用于裁剪）
	beltLeftEdge := conveyorX + config.ConveyorBeltLeftPadding
	beltRightEdge := conveyorX + beltWidth - config.ConveyorBeltRightPadding

	// 遍历绘制每张卡片
	for i, card := range beltComp.Cards {
		// 卡片 X 位置：传送带左边界 + 卡片的 PositionX
		cardX := conveyorX + card.PositionX

		// 检查是否选中
		isSelected := beltComp.SelectedCardIndex == i

		// 计算裁剪参数
		clipRatio := 1.0

		// 右侧裁剪：卡片超出右边界
		if cardX+cardWidth > beltRightEdge {
			visibleWidth := beltRightEdge - cardX
			if visibleWidth <= 0 {
				continue // 完全在右边界外，不绘制
			}
			clipRatio = visibleWidth / cardWidth
		}

		// 左侧裁剪：卡片超出左边界（通常不会发生）
		if cardX < beltLeftEdge {
			continue // 超出左边界，不绘制
		}

		// 绘制卡片（带裁剪）
		s.drawConveyorCardClipped(screen, card.CardType, cardX, cardY, cardWidth, cardHeight, isSelected, clipRatio)
	}
}

// drawConveyorCard 绘制单张卡片（背景 + 植物图标 + 选中遮罩）
func (s *GameScene) drawConveyorCard(screen *ebiten.Image, cardType string, x, y, width, height float64, isSelected bool) {
	// 使用真实卡片图像渲染
	if s.conveyorCardBackground != nil && s.conveyorWallnutIcon != nil {
		s.drawConveyorCardWithImages(screen, cardType, x, y, width, height, isSelected)
		return
	}

	// 回退：使用简单矩形绘制（资源未加载时）
	s.drawConveyorCardFallback(screen, cardType, x, y, width, height, isSelected)
}

// drawConveyorCardClipped 绘制带裁剪的卡片（clipRatio: 0-1，可见比例）
func (s *GameScene) drawConveyorCardClipped(screen *ebiten.Image, cardType string, x, y, width, height float64, isSelected bool, clipRatio float64) {
	// 如果完全可见，直接调用普通绘制
	if clipRatio >= 1.0 {
		s.drawConveyorCard(screen, cardType, x, y, width, height, isSelected)
		return
	}

	// 如果完全不可见，不绘制
	if clipRatio <= 0 {
		return
	}

	// 使用真实卡片图像渲染（带裁剪）
	if s.conveyorCardBackground != nil && s.conveyorWallnutIcon != nil {
		s.drawConveyorCardWithImagesClipped(screen, cardType, x, y, width, height, isSelected, clipRatio)
		return
	}

	// 回退：使用简单矩形绘制（资源未加载时）
	s.drawConveyorCardFallbackClipped(screen, cardType, x, y, width, height, isSelected, clipRatio)
}

// drawConveyorCardWithImagesClipped 使用图像绘制带裁剪的传送带卡片
// Story 8.15: 支持所有植物类型的图标渲染
func (s *GameScene) drawConveyorCardWithImagesClipped(screen *ebiten.Image, cardType string, x, y, width, height float64, isSelected bool, clipRatio float64) {
	// 计算卡片缩放因子
	bgBounds := s.conveyorCardBackground.Bounds()
	bgWidth := float64(bgBounds.Dx())
	bgHeight := float64(bgBounds.Dy())
	cardScale := width / bgWidth

	// 计算裁剪后的源图像区域（从左侧开始裁剪）
	clipWidth := int(bgWidth * clipRatio)
	if clipWidth <= 0 {
		return
	}

	// 1. 绘制卡片背景框（裁剪）
	srcRect := image.Rect(0, 0, clipWidth, int(bgHeight))
	clippedBg := s.conveyorCardBackground.SubImage(srcRect).(*ebiten.Image)

	bgOp := &ebiten.DrawImageOptions{}
	bgOp.GeoM.Scale(cardScale, cardScale)
	bgOp.GeoM.Translate(x, y)
	screen.DrawImage(clippedBg, bgOp)

	// 2. 绘制植物图标（需要判断是否在裁剪范围内）
	// Story 8.15: 从缓存中查找植物图标
	var plantIcon *ebiten.Image
	if s.conveyorPlantIcons != nil {
		plantIcon = s.conveyorPlantIcons[cardType]
	}
	// 回退到坚果图标
	if plantIcon == nil {
		switch cardType {
		case components.CardTypeExplodeONut:
			plantIcon = s.conveyorExplodeNutIcon
		default:
			plantIcon = s.conveyorWallnutIcon
		}
	}

	if plantIcon != nil {
		iconScale := config.PlantCardIconScale * cardScale
		iconOffsetY := config.PlantCardIconOffsetY * cardScale

		iconWidth := float64(plantIcon.Bounds().Dx()) * iconScale
		scaledBgWidth := bgWidth * cardScale

		// 图标水平居中
		iconOffsetX := (scaledBgWidth - iconWidth) / 2.0

		// 计算图标在卡片中的可见区域
		visibleCardWidth := float64(clipWidth) * cardScale
		iconX := x + iconOffsetX
		iconRightEdge := iconX + iconWidth

		// 如果图标完全在裁剪区域外，不绘制
		if iconX >= x+visibleCardWidth {
			// 图标完全不可见
		} else if iconRightEdge <= x+visibleCardWidth {
			// 图标完全可见
			iconOp := &ebiten.DrawImageOptions{}
			iconOp.GeoM.Scale(iconScale, iconScale)
			iconOp.GeoM.Translate(iconX, y+iconOffsetY)

			if cardType == components.CardTypeExplodeONut {
				iconOp.ColorScale.Scale(1.0, 0.6, 0.6, 1.0)
			}
			screen.DrawImage(plantIcon, iconOp)
		} else {
			// 图标部分可见，需要裁剪
			iconVisibleWidth := x + visibleCardWidth - iconX
			iconClipRatio := iconVisibleWidth / iconWidth
			iconSrcWidth := int(float64(plantIcon.Bounds().Dx()) * iconClipRatio)
			if iconSrcWidth > 0 {
				iconSrcRect := image.Rect(0, 0, iconSrcWidth, plantIcon.Bounds().Dy())
				clippedIcon := plantIcon.SubImage(iconSrcRect).(*ebiten.Image)

				iconOp := &ebiten.DrawImageOptions{}
				iconOp.GeoM.Scale(iconScale, iconScale)
				iconOp.GeoM.Translate(iconX, y+iconOffsetY)

				if cardType == components.CardTypeExplodeONut {
					iconOp.ColorScale.Scale(1.0, 0.6, 0.6, 1.0)
				}
				screen.DrawImage(clippedIcon, iconOp)
			}
		}
	}

	// 3. 选中状态：绘制禁用遮罩（裁剪）
	if isSelected {
		scaledWidth := float64(clipWidth) * cardScale
		scaledHeight := bgHeight * cardScale
		intWidth := int(scaledWidth)
		intHeight := int(scaledHeight)

		if intWidth > 0 && intHeight > 0 {
			mask := ebiten.NewImage(intWidth, intHeight)
			mask.Fill(color.RGBA{R: 0, G: 0, B: 0, A: uint8(config.ConveyorCardSelectedOverlayAlpha)})

			maskOp := &ebiten.DrawImageOptions{}
			maskOp.GeoM.Translate(x, y)
			screen.DrawImage(mask, maskOp)
		}
	}
}

// drawConveyorCardFallbackClipped 使用简单矩形绘制带裁剪的卡片（回退方案）
func (s *GameScene) drawConveyorCardFallbackClipped(screen *ebiten.Image, cardType string, x, y, width, height float64, isSelected bool, clipRatio float64) {
	// 卡片背景颜色（根据类型）
	var bgColor color.RGBA

	switch cardType {
	case components.CardTypeWallnutBowling:
		bgColor = color.RGBA{R: 160, G: 120, B: 80, A: 255} // 棕色
	case components.CardTypeExplodeONut:
		bgColor = color.RGBA{R: 200, G: 80, B: 80, A: 255} // 红色
	default:
		bgColor = color.RGBA{R: 128, G: 128, B: 128, A: 255} // 灰色
	}

	// 绘制裁剪后的卡片背景
	visibleWidth := width * clipRatio
	ebitenutil.DrawRect(screen, x, y, visibleWidth, height, bgColor)

	// 选中状态添加禁用遮罩
	if isSelected {
		overlayColor := color.RGBA{
			R: 0,
			G: 0,
			B: 0,
			A: uint8(config.ConveyorCardSelectedOverlayAlpha),
		}
		ebitenutil.DrawRect(screen, x, y, visibleWidth, height, overlayColor)
	}
}

// drawConveyorCardWithImages 使用图像绘制传送带卡片（背景框 + 图标 + 遮罩）
// Story 8.15: 支持所有植物类型的图标渲染
func (s *GameScene) drawConveyorCardWithImages(screen *ebiten.Image, cardType string, x, y, width, height float64, isSelected bool) {
	// 计算卡片缩放因子
	// 原始卡片背景尺寸约 100x140，传送带卡片目标尺寸由 width/height 决定
	bgBounds := s.conveyorCardBackground.Bounds()
	bgWidth := float64(bgBounds.Dx())
	bgHeight := float64(bgBounds.Dy())

	// 使用宽度计算缩放因子，保持比例
	cardScale := width / bgWidth

	// 1. 绘制卡片背景框
	bgOp := &ebiten.DrawImageOptions{}
	bgOp.GeoM.Scale(cardScale, cardScale)
	bgOp.GeoM.Translate(x, y)
	screen.DrawImage(s.conveyorCardBackground, bgOp)

	// 2. 绘制植物图标
	// Story 8.15: 从缓存中查找植物图标
	var plantIcon *ebiten.Image
	if s.conveyorPlantIcons != nil {
		plantIcon = s.conveyorPlantIcons[cardType]
	}
	// 回退到坚果图标
	if plantIcon == nil {
		switch cardType {
		case components.CardTypeExplodeONut:
			plantIcon = s.conveyorExplodeNutIcon
		default:
			plantIcon = s.conveyorWallnutIcon
		}
	}

	if plantIcon != nil {
		iconScale := config.PlantCardIconScale * cardScale
		iconOffsetY := config.PlantCardIconOffsetY * cardScale

		iconWidth := float64(plantIcon.Bounds().Dx()) * iconScale
		scaledBgWidth := bgWidth * cardScale

		// 图标水平居中
		iconOffsetX := (scaledBgWidth - iconWidth) / 2.0

		iconOp := &ebiten.DrawImageOptions{}
		iconOp.GeoM.Scale(iconScale, iconScale)
		iconOp.GeoM.Translate(x+iconOffsetX, y+iconOffsetY)

		// 爆炸坚果添加红色染色
		if cardType == components.CardTypeExplodeONut {
			// 红色染色：增加红色通道，降低绿蓝通道
			iconOp.ColorScale.Scale(1.0, 0.6, 0.6, 1.0)
		}

		screen.DrawImage(plantIcon, iconOp)
	}

	// 3. 选中状态：绘制禁用遮罩
	if isSelected {
		scaledWidth := bgWidth * cardScale
		scaledHeight := bgHeight * cardScale
		intWidth := int(scaledWidth)
		intHeight := int(scaledHeight)

		if intWidth > 0 && intHeight > 0 {
			mask := ebiten.NewImage(intWidth, intHeight)
			mask.Fill(color.RGBA{R: 0, G: 0, B: 0, A: uint8(config.ConveyorCardSelectedOverlayAlpha)})

			maskOp := &ebiten.DrawImageOptions{}
			maskOp.GeoM.Translate(x, y)
			screen.DrawImage(mask, maskOp)
		}
	}
}

// drawConveyorCardFallback 使用简单矩形绘制卡片（回退方案）
func (s *GameScene) drawConveyorCardFallback(screen *ebiten.Image, cardType string, x, y, width, height float64, isSelected bool) {
	// 卡片背景颜色（根据类型）
	var bgColor color.RGBA

	switch cardType {
	case components.CardTypeWallnutBowling:
		bgColor = color.RGBA{R: 160, G: 120, B: 80, A: 255} // 棕色
	case components.CardTypeExplodeONut:
		bgColor = color.RGBA{R: 200, G: 80, B: 80, A: 255} // 红色
	default:
		bgColor = color.RGBA{R: 128, G: 128, B: 128, A: 255} // 灰色
	}

	// 绘制卡片背景
	ebitenutil.DrawRect(screen, x, y, width, height, bgColor)

	// 选中状态添加禁用遮罩
	if isSelected {
		overlayColor := color.RGBA{
			R: 0,
			G: 0,
			B: 0,
			A: uint8(config.ConveyorCardSelectedOverlayAlpha),
		}
		ebitenutil.DrawRect(screen, x, y, width, height, overlayColor)
	}
}

func (s *GameScene) drawConveyorCardPreview(screen *ebiten.Image) {
}
