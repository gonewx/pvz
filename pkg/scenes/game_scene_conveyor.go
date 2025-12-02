package scenes

import (
	"image"
	"image/color"
	"math"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// drawConveyorBelt 绘制传送带
// Story 19.5: 传送带 UI 渲染
//
// 渲染顺序：
// 1. 传送带背景（ConveyorBelt_backdrop）
// 2. 传动动画（6行交错滚动）
// 3. 卡片
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

// calculateConveyorX 计算传送带 X 位置
// 传送带右边缘紧挨铲子槽位左边缘
func (s *GameScene) calculateConveyorX() float64 {
	// 铲子 X 位置
	shovelX := float64(config.ShovelX) // 默认值
	if s.seedBank != nil {
		seedBankWidth := float64(s.seedBank.Bounds().Dx())
		shovelX = float64(config.SeedBankX) + seedBankWidth + float64(config.ShovelGapFromSeedBank)
	}

	// 传送带 X = 铲子 X - 传送带宽度
	conveyorX := shovelX - config.ConveyorBeltWidth

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

// drawConveyorBeltAnimation 绘制传送带传动动画
// 6 行交错滚动，模拟履带效果
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

	// 获取纹理尺寸
	imgBounds := s.conveyorBelt.Bounds()
	imgWidth := imgBounds.Dx()
	imgHeight := imgBounds.Dy()
	rowHeight := imgHeight / config.ConveyorBeltRowCount

	// 计算传送带实际渲染宽度
	beltRenderWidth := config.ConveyorBeltWidth - config.ConveyorBeltPadding*2
	beltRenderX := x + config.ConveyorBeltPadding
	beltRenderY := y + config.ConveyorBeltPadding

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

	// 卡片布局参数
	cardWidth := config.ConveyorCardWidth
	cardHeight := config.ConveyorCardHeight
	cardSpacing := config.ConveyorCardSpacing
	padding := config.ConveyorBeltPadding

	// 卡片起始位置（从左到右排列）
	cardStartX := conveyorX + padding
	cardStartY := conveyorY + padding + 10 // 居中偏移

	// 遍历绘制每张卡片
	for i, card := range beltComp.Cards {
		// 计算卡片 X 位置
		// 滑入动画：新卡片从右侧滑入
		targetX := cardStartX + float64(card.SlotIndex)*(cardWidth+cardSpacing)
		slideOffset := (1.0 - card.SlideProgress) * (cardWidth + cardSpacing)
		cardX := targetX + slideOffset

		cardY := cardStartY

		// 检查是否选中
		isSelected := beltComp.SelectedCardIndex == i

		// 绘制卡片
		s.drawConveyorCard(screen, card.CardType, cardX, cardY, cardWidth, cardHeight, isSelected)
	}
}

// drawConveyorCard 绘制单张卡片
func (s *GameScene) drawConveyorCard(screen *ebiten.Image, cardType string, x, y, width, height float64, isSelected bool) {
	// 卡片背景颜色（根据类型）
	var bgColor color.RGBA
	var borderColor color.RGBA

	switch cardType {
	case components.CardTypeWallnutBowling:
		bgColor = color.RGBA{R: 160, G: 120, B: 80, A: 255}   // 棕色（坚果色）
		borderColor = color.RGBA{R: 100, G: 70, B: 40, A: 255}
	case components.CardTypeExplodeONut:
		bgColor = color.RGBA{R: 200, G: 80, B: 80, A: 255}    // 红色（爆炸坚果）
		borderColor = color.RGBA{R: 150, G: 50, B: 50, A: 255}
	default:
		bgColor = color.RGBA{R: 128, G: 128, B: 128, A: 255}  // 灰色
		borderColor = color.RGBA{R: 80, G: 80, B: 80, A: 255}
	}

	// 选中状态：高亮边框
	if isSelected {
		borderColor = color.RGBA{R: 255, G: 215, B: 0, A: 255} // 金色
	}

	// 绘制卡片背景
	ebitenutil.DrawRect(screen, x, y, width, height, bgColor)

	// 绘制边框
	borderWidth := 2.0
	// 上
	ebitenutil.DrawRect(screen, x, y, width, borderWidth, borderColor)
	// 下
	ebitenutil.DrawRect(screen, x, y+height-borderWidth, width, borderWidth, borderColor)
	// 左
	ebitenutil.DrawRect(screen, x, y, borderWidth, height, borderColor)
	// 右
	ebitenutil.DrawRect(screen, x+width-borderWidth, y, borderWidth, height, borderColor)

	// TODO(Story 19.6): 绘制卡片图标（坚果/爆炸坚果图片）
	// 当前使用颜色区分，后续添加真实图标

	// 绘制卡片类型标识（临时文字）
	// 使用简单的首字母标识
	var label string
	switch cardType {
	case components.CardTypeWallnutBowling:
		label = "W"
	case components.CardTypeExplodeONut:
		label = "E"
	}

	// 简单文字标识（居中）
	labelX := x + width/2 - 4
	labelY := y + height/2 - 8
	ebitenutil.DebugPrintAt(screen, label, int(labelX), int(labelY))
}

// getConveyorBeltBounds 获取传送带边界（屏幕坐标）
// 用于点击检测
func (s *GameScene) getConveyorBeltBounds() (x, y, width, height float64) {
	if s.levelPhaseSystem == nil || !s.levelPhaseSystem.IsConveyorBeltVisible() {
		return 0, 0, 0, 0
	}

	conveyorX := s.calculateConveyorX()
	conveyorY := s.levelPhaseSystem.GetConveyorBeltY()

	return conveyorX, conveyorY, config.ConveyorBeltWidth, config.ConveyorCardHeight + config.ConveyorBeltPadding*2 + 20
}

// handleConveyorBeltClick 处理传送带卡片点击
// Story 19.5: 卡片选中逻辑
//
// 返回：
//   - 是否消费了点击事件
func (s *GameScene) handleConveyorBeltClick(mouseX, mouseY int) bool {
	if s.conveyorBeltSystem == nil || !s.conveyorBeltSystem.IsActive() {
		return false
	}

	// 获取传送带边界
	conveyorX, conveyorY, _, _ := s.getConveyorBeltBounds()
	if conveyorX == 0 && conveyorY == 0 {
		return false
	}

	// 检查点击是否在传送带卡片上
	cardIndex := s.conveyorBeltSystem.GetCardAtPosition(
		float64(mouseX), float64(mouseY),
		conveyorX+config.ConveyorBeltPadding,
		conveyorY+config.ConveyorBeltPadding+10,
	)

	if cardIndex >= 0 {
		// 选中卡片
		s.conveyorBeltSystem.SelectCard(cardIndex)
		return true
	}

	return false
}

// handleConveyorCardPlacement 处理传送带卡片放置
// Story 19.5: 卡片放置到草坪
//
// 参数：
//   - worldX, worldY: 世界坐标
//
// 返回：
//   - 是否成功放置
func (s *GameScene) handleConveyorCardPlacement(worldX, worldY float64) bool {
	if s.conveyorBeltSystem == nil {
		return false
	}

	// 获取当前选中的卡片
	cardType := s.conveyorBeltSystem.GetSelectedCard()
	if cardType == "" {
		return false
	}

	// 检查放置位置是否有效（红线左侧）
	if !s.conveyorBeltSystem.IsPlacementValid(worldX) {
		// 显示提示文字
		s.showPlacementHint()
		// 取消选中，卡片返回传送带
		s.conveyorBeltSystem.DeselectCard()
		return false
	}

	// 计算放置的行列
	col := int((worldX - config.GridWorldStartX) / config.CellWidth)
	row := int((worldY - config.GridWorldStartY) / config.CellHeight)

	// 验证行列有效性
	if row < 0 || row >= config.GridRows || col < 0 || col >= config.GridColumns {
		s.conveyorBeltSystem.DeselectCard()
		return false
	}

	// 获取选中的卡片索引
	beltEntity := s.conveyorBeltSystem.GetBeltEntity()
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, beltEntity)
	if !ok || beltComp.SelectedCardIndex < 0 {
		return false
	}

	// 移除卡片
	removedCardType := s.conveyorBeltSystem.RemoveCard(beltComp.SelectedCardIndex)
	if removedCardType == "" {
		return false
	}

	// TODO(Story 19.6): 创建保龄球坚果实体
	// 当前使用日志占位
	// log.Printf("[GameScene] Placed %s at row=%d, col=%d (worldX=%.1f, worldY=%.1f)",
	// 	removedCardType, row+1, col+1, worldX, worldY)

	// 取消选中状态
	s.conveyorBeltSystem.DeselectCard()

	return true
}

// showPlacementHint 显示放置限制提示
// Story 19.5: 使用 [ADVICE_NOT_PASSED_LINE] 文本 key
func (s *GameScene) showPlacementHint() {
	// TODO: 使用教学系统显示提示文字 [ADVICE_NOT_PASSED_LINE]
	// 当前使用日志占位
	// log.Printf("[GameScene] Cannot place card: must be placed to the left of the red line")
}

// isConveyorCardSelected 检查是否有传送带卡片被选中
func (s *GameScene) isConveyorCardSelected() bool {
	if s.conveyorBeltSystem == nil {
		return false
	}
	return s.conveyorBeltSystem.GetSelectedCard() != ""
}

// updateConveyorBeltClick 处理传送带点击和卡片放置
// Story 19.5: 卡片选中和放置逻辑
func (s *GameScene) updateConveyorBeltClick() {
	// 检查传送带是否激活
	if s.conveyorBeltSystem == nil || !s.conveyorBeltSystem.IsActive() {
		return
	}

	// 检测左键点击
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}

	mouseX, mouseY := ebiten.CursorPosition()

	// 如果已选中卡片，检测是否在草坪上放置
	if s.isConveyorCardSelected() {
		// 转换为世界坐标
		worldX := float64(mouseX) + s.cameraX
		worldY := float64(mouseY)

		// 检查是否点击在草坪区域
		if worldY >= config.GridWorldStartY && worldY < config.GridWorldStartY+float64(config.GridRows)*config.CellHeight {
			s.handleConveyorCardPlacement(worldX, worldY)
			return
		}

		// 点击在草坪外，取消选中
		s.conveyorBeltSystem.DeselectCard()
		return
	}

	// 未选中卡片，检测是否点击传送带卡片
	s.handleConveyorBeltClick(mouseX, mouseY)
}

// drawConveyorCardPreview 绘制选中卡片的拖拽预览
// Story 19.5: 在鼠标位置显示选中的卡片
func (s *GameScene) drawConveyorCardPreview(screen *ebiten.Image) {
	if !s.isConveyorCardSelected() {
		return
	}

	cardType := s.conveyorBeltSystem.GetSelectedCard()
	if cardType == "" {
		return
	}

	// 获取鼠标位置
	mouseX, mouseY := ebiten.CursorPosition()

	// 绘制卡片预览（半透明）
	previewWidth := config.ConveyorCardWidth * 1.5
	previewHeight := config.ConveyorCardHeight * 1.5
	previewX := float64(mouseX) - previewWidth/2
	previewY := float64(mouseY) - previewHeight/2

	// 检查放置位置是否有效
	worldX := float64(mouseX) + s.cameraX
	isValid := s.conveyorBeltSystem.IsPlacementValid(worldX)

	// 绘制预览（有效为绿色边框，无效为红色边框）
	var borderColor color.RGBA
	if isValid {
		borderColor = color.RGBA{R: 0, G: 255, B: 0, A: 200} // 绿色
	} else {
		borderColor = color.RGBA{R: 255, G: 0, B: 0, A: 200} // 红色
	}

	// 卡片背景（半透明）
	var bgColor color.RGBA
	switch cardType {
	case components.CardTypeWallnutBowling:
		bgColor = color.RGBA{R: 160, G: 120, B: 80, A: 180}
	case components.CardTypeExplodeONut:
		bgColor = color.RGBA{R: 200, G: 80, B: 80, A: 180}
	default:
		bgColor = color.RGBA{R: 128, G: 128, B: 128, A: 180}
	}

	ebitenutil.DrawRect(screen, previewX, previewY, previewWidth, previewHeight, bgColor)

	// 绘制边框
	borderWidth := 3.0
	ebitenutil.DrawRect(screen, previewX, previewY, previewWidth, borderWidth, borderColor)
	ebitenutil.DrawRect(screen, previewX, previewY+previewHeight-borderWidth, previewWidth, borderWidth, borderColor)
	ebitenutil.DrawRect(screen, previewX, previewY, borderWidth, previewHeight, borderColor)
	ebitenutil.DrawRect(screen, previewX+previewWidth-borderWidth, previewY, borderWidth, previewHeight, borderColor)
}
