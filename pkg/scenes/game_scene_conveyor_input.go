package scenes

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ============================================================================
// 传送带输入处理相关方法
// Story 15.5: 从 game_scene_conveyor.go 拆分出传送带输入逻辑（约550行）
// ============================================================================

// getConveyorBeltBounds 获取传送带边界（屏幕坐标）
// 用于点击检测
func (s *GameScene) getConveyorBeltBounds() (x, y, width, height float64) {
	if s.levelPhaseSystem == nil || !s.levelPhaseSystem.IsConveyorBeltVisible() {
		return 0, 0, 0, 0
	}

	conveyorX := s.calculateConveyorX()
	conveyorY := s.levelPhaseSystem.GetConveyorBeltY()

	// 卡片高度使用比例配置计算（原始高度约 140px * 缩放比例）
	cardHeight := 140.0 * config.ConveyorCardScale

	return conveyorX, conveyorY, config.ConveyorBeltWidth, cardHeight + config.ConveyorBeltLeftPadding*2 + 20
}

// isMouseOverConveyorCard 检测鼠标是否悬停在传送带卡片上
// 用于更新鼠标光标形状
func (s *GameScene) isMouseOverConveyorCard() bool {
	if s.conveyorBeltSystem == nil || !s.conveyorBeltSystem.IsActive() {
		return false
	}

	// 如果已选中卡片，不检测悬停（避免光标闪烁）
	if s.isConveyorCardSelected() {
		return false
	}

	// 获取传送带边界
	conveyorX, conveyorY, _, _ := s.getConveyorBeltBounds()
	if conveyorX == 0 && conveyorY == 0 {
		return false
	}

	// 使用缩放比例计算卡片尺寸
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

	// 获取传送带背景高度，用于计算垂直居中的Y偏移
	var beltHeight float64
	if s.conveyorBeltBackdrop != nil {
		beltHeight = float64(s.conveyorBeltBackdrop.Bounds().Dy())
	} else {
		beltHeight = 80.0
	}
	// 垂直居中偏移
	cardStartY := conveyorY + (beltHeight-cardHeight)/2 + config.ConveyorBeltTopPadding

	// 获取鼠标位置
	mouseX, mouseY := utils.GetPointerPosition()

	// 检测是否悬停在任意卡片上（包括移动中的卡片）
	// 注意：card.PositionX 是相对于传送带左边缘的位置（已包含 leftPadding 偏移）
	cardIndex := s.conveyorBeltSystem.GetCardAtPositionForHover(
		float64(mouseX), float64(mouseY),
		conveyorX,
		cardStartY,
		cardWidth, cardHeight,
	)

	return cardIndex >= 0
}

// handleConveyorBeltClick 处理传送带卡片点击，返回是否消费了事件
func (s *GameScene) handleConveyorBeltClick(mouseX, mouseY int) bool {
	if s.conveyorBeltSystem == nil || !s.conveyorBeltSystem.IsActive() {
		return false
	}

	// 获取传送带边界
	conveyorX, conveyorY, _, _ := s.getConveyorBeltBounds()
	if conveyorX == 0 && conveyorY == 0 {
		return false
	}

	// 使用缩放比例计算卡片尺寸
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

	// 获取传送带背景高度，用于计算垂直居中的Y偏移
	var beltHeight float64
	if s.conveyorBeltBackdrop != nil {
		beltHeight = float64(s.conveyorBeltBackdrop.Bounds().Dy())
	} else {
		beltHeight = 80.0
	}
	// 垂直居中偏移
	cardStartY := conveyorY + (beltHeight-cardHeight)/2

	// 检查点击是否在传送带卡片上
	// 注意：card.PositionX 是相对于传送带左边缘的位置（已包含 leftPadding 偏移）
	// 所以这里传入 conveyorX，而不是 conveyorX + leftPadding
	cardIndex := s.conveyorBeltSystem.GetCardAtPosition(
		float64(mouseX), float64(mouseY),
		conveyorX,
		cardStartY,
		cardWidth, cardHeight,
	)

	if cardIndex >= 0 {
		// 选中卡片
		s.conveyorBeltSystem.SelectCard(cardIndex)

		// 播放选中植物卡片音效 (seedlift.ogg)
		if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
			audioManager.PlaySound("SOUND_SEEDLIFT")
		}
		return true
	}

	return false
}

// handleConveyorCardPlacement 处理传送带卡片放置，返回是否成功
func (s *GameScene) handleConveyorCardPlacement(worldX, worldY float64) bool {
	if s.conveyorBeltSystem == nil {
		return false
	}

	// 获取当前选中的卡片
	cardType := s.conveyorBeltSystem.GetSelectedCard()
	if cardType == "" {
		return false
	}

	if !s.conveyorBeltSystem.IsPlacementValid(worldX) {
		s.showPlacementHint()
		return false
	}

	// 计算放置的行列
	col := int((worldX - config.GridWorldStartX) / config.CellWidth)
	row := int((worldY - config.GridWorldStartY) / config.CellHeight)

	// 验证行列有效性
	if row < 0 || row >= config.GridRows || col < 0 || col >= config.GridColumns {
		s.conveyorBeltSystem.DeselectCard()
		s.destroyConveyorCardPreview()
		return false
	}

	beltEntity := s.conveyorBeltSystem.GetBeltEntity()
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, beltEntity)
	if !ok || beltComp.SelectedCardIndex < 0 {
		return false
	}

	removedCardType := s.conveyorBeltSystem.RemoveCard(beltComp.SelectedCardIndex)
	if removedCardType == "" {
		return false
	}

	isExplosive := removedCardType == components.CardTypeExplodeONut

	entityID, err := entities.NewBowlingNutEntity(
		s.entityManager,
		s.resourceManager,
		row,
		col,
		isExplosive,
	)

	if err != nil {
		log.Printf("[GameScene] 创建保龄球坚果失败: %v", err)
		s.conveyorBeltSystem.DeselectCard()
		s.destroyConveyorCardPreview()
		return false
	}

	log.Printf("[GameScene] 放置保龄球坚果: entityID=%d, type=%s, row=%d, col=%d",
		entityID, removedCardType, row+1, col+1)

	s.conveyorBeltSystem.DeselectCard()
	s.destroyConveyorCardPreview()

	return true
}

// showPlacementHint 显示放置限制提示（红线左侧才能放置）
func (s *GameScene) showPlacementHint() {
	textKey := "ADVICE_NOT_PASSED_LINE"
	var text string
	if s.gameState != nil && s.gameState.LawnStrings != nil {
		text = s.gameState.LawnStrings.GetString(textKey)
	}
	if text == "" {
		text = "在红线的左边才能放坚果墙"
	}

	textEntity := s.entityManager.CreateEntity()
	textComp := &components.TutorialTextComponent{
		Text:            text,
		DisplayTime:     0,
		MaxDisplayTime:  config.AdvisoryTutorialTextDisplayDuration,
		BackgroundAlpha: 0.5,
		IsAdvisory:      false,
		IsBowling:       true,
	}
	s.entityManager.AddComponent(textEntity, textComp)

	log.Printf("[GameScene] 显示放置提示: '%s' (Entity ID: %d)", text, textEntity)
}

// isConveyorCardSelected 检查是否有传送带卡片被选中
func (s *GameScene) isConveyorCardSelected() bool {
	if s.conveyorBeltSystem == nil {
		return false
	}
	return s.conveyorBeltSystem.GetSelectedCard() != ""
}

// createConveyorCardPreview 创建传送带卡片预览实体（复用 PlantPreviewComponent）
func (s *GameScene) createConveyorCardPreview() {
	// 先销毁已存在的预览
	s.destroyConveyorCardPreview()

	cardType := s.conveyorBeltSystem.GetSelectedCard()
	if cardType == "" {
		return
	}

	isExplosive := cardType == components.CardTypeExplodeONut

	var previewIcon *ebiten.Image
	if isExplosive {
		previewIcon = s.conveyorExplodeNutIcon
	} else {
		previewIcon = s.conveyorWallnutIcon
	}

	if previewIcon == nil {
		log.Printf("[GameScene] No preview icon available for conveyor card")
		return
	}

	mouseX, mouseY := utils.GetPointerPosition()
	worldX := float64(mouseX) + s.cameraX
	worldY := float64(mouseY)

	entityID := s.entityManager.CreateEntity()

	ecs.AddComponent(s.entityManager, entityID, &components.PositionComponent{
		X: worldX,
		Y: worldY,
	})

	ecs.AddComponent(s.entityManager, entityID, &components.SpriteComponent{
		Image: previewIcon,
	})

	ecs.AddComponent(s.entityManager, entityID, &components.PlantPreviewComponent{
		PlantType:   components.PlantWallnut,
		Alpha:       0.5,
		IsExplosive: isExplosive,
	})

	log.Printf("[GameScene] 创建传送带卡片预览: entityID=%d, explosive=%v", entityID, isExplosive)
}

// destroyConveyorCardPreview 销毁传送带卡片预览实体
func (s *GameScene) destroyConveyorCardPreview() {
	entities := ecs.GetEntitiesWith1[*components.PlantPreviewComponent](s.entityManager)

	for _, entityID := range entities {
		s.entityManager.DestroyEntity(entityID)
	}

	if len(entities) > 0 {
		s.entityManager.RemoveMarkedEntities()
	}
}

// updateConveyorBeltClick 处理传送带点击和卡片放置
func (s *GameScene) updateConveyorBeltClick() {
	// 检查传送带是否激活
	if s.conveyorBeltSystem == nil || !s.conveyorBeltSystem.IsActive() {
		return
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		if s.isConveyorCardSelected() {
			s.conveyorBeltSystem.DeselectCard()
			s.destroyConveyorCardPreview()
			log.Printf("[GameScene] 右键取消选中传送带卡片")
			return
		}
	}

	dragManager := utils.GetDragManager()

	if dragManager.IsTouchDrag() || s.isDragConveyorPlanting {
		if s.handleConveyorDragPlanting() {
			return // 拖拽种植已处理，跳过传统点击逻辑
		}
	}

	// ========================================================================
	// 传统点击处理（桌面端鼠标点击）
	// ========================================================================
	// 如果铲子被选中，跳过传送带卡片点击处理（避免同时选中）
	if s.shovelSelected {
		return
	}

	justPressed, pressX, pressY := utils.IsPointerJustPressed()
	if !justPressed {
		return
	}

	mouseX, mouseY := pressX, pressY

	if s.isConveyorCardSelected() {
		worldX := float64(mouseX) + s.cameraX
		worldY := float64(mouseY)

		if worldY >= config.GridWorldStartY && worldY < config.GridWorldStartY+float64(config.GridRows)*config.CellHeight {
			if s.handleConveyorCardPlacement(worldX, worldY) {
				s.destroyConveyorCardPreview()
			}
			return
		}

		s.conveyorBeltSystem.DeselectCard()
		s.destroyConveyorCardPreview()
		return
	}

	if s.handleConveyorBeltClick(mouseX, mouseY) {
		s.createConveyorCardPreview()
	}
}

// drawConveyorCardPreview 绘制选中卡片的拖拽预览（由 PlantPreviewRenderSystem 渲染）

// handleConveyorDragPlanting 处理传送带卡片的拖拽种植逻辑（移动端）
func (s *GameScene) handleConveyorDragPlanting() bool {
	dragManager := utils.GetDragManager()
	dragInfo := dragManager.GetInfo()

	switch dragInfo.State {
	case utils.DragStateStarted:
		// 拖拽刚开始，检测是否从传送带卡片开始
		return s.handleConveyorDragStart(dragInfo)

	case utils.DragStateDragging:
		// 拖拽进行中，更新预览位置
		if s.isDragConveyorPlanting {
			s.updateConveyorDragPreview(dragInfo)
			return true
		}

	case utils.DragStateEnded:
		// 拖拽结束，尝试放置或取消
		if s.isDragConveyorPlanting {
			s.handleConveyorDragEnd(dragInfo)
			return true
		}
	}

	return false
}

// handleConveyorDragStart 处理传送带卡片拖拽开始（仅触摸输入）
func (s *GameScene) handleConveyorDragStart(dragInfo utils.DragInfo) bool {
	if !dragInfo.IsTouchInput {
		return false
	}

	if s.shovelSelected {
		return false
	}

	// 获取传送带边界
	conveyorX, conveyorY, _, _ := s.getConveyorBeltBounds()
	if conveyorX == 0 && conveyorY == 0 {
		return false
	}

	// 使用缩放比例计算卡片尺寸
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

	var beltHeight float64
	if s.conveyorBeltBackdrop != nil {
		beltHeight = float64(s.conveyorBeltBackdrop.Bounds().Dy())
	} else {
		beltHeight = 80.0
	}
	cardStartY := conveyorY + (beltHeight-cardHeight)/2 + config.ConveyorBeltTopPadding

	cardIndex := s.conveyorBeltSystem.GetCardAtPosition(
		float64(dragInfo.StartX), float64(dragInfo.StartY),
		conveyorX,
		cardStartY,
		cardWidth, cardHeight,
	)

	if cardIndex < 0 {
		return false
	}

	beltEntity := s.conveyorBeltSystem.GetBeltEntity()
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, beltEntity)
	if !ok || cardIndex >= len(beltComp.Cards) {
		return false
	}

	cardType := beltComp.Cards[cardIndex].CardType

	s.isDragConveyorPlanting = true
	s.dragConveyorCardIndex = cardIndex
	s.dragConveyorCardType = cardType

	s.conveyorBeltSystem.SelectCard(cardIndex)

	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_SEEDLIFT")
	}

	s.createConveyorCardPreview()

	log.Printf("[GameScene] 传送带卡片拖拽开始: cardIndex=%d, cardType=%s",
		cardIndex, cardType)

	return true
}

// updateConveyorDragPreview 更新拖拽预览位置
func (s *GameScene) updateConveyorDragPreview(dragInfo utils.DragInfo) {
	previewEntities := ecs.GetEntitiesWith1[*components.PlantPreviewComponent](s.entityManager)
	for _, entityID := range previewEntities {
		pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		if ok {
			pos.X = float64(dragInfo.CurrentX) + s.cameraX
			pos.Y = float64(dragInfo.CurrentY)
		}
	}
}

// handleConveyorDragEnd 处理拖拽结束
func (s *GameScene) handleConveyorDragEnd(dragInfo utils.DragInfo) {
	defer s.cancelConveyorDragPlanting()

	worldX := float64(dragInfo.CurrentX) + s.cameraX
	worldY := float64(dragInfo.CurrentY)

	if worldY < config.GridWorldStartY || worldY >= config.GridWorldStartY+float64(config.GridRows)*config.CellHeight {
		log.Printf("[GameScene] 传送带卡片拖拽结束: 位置在草坪外，取消种植")
		return
	}

	if !s.conveyorBeltSystem.IsPlacementValid(worldX) {
		s.showPlacementHint()
		log.Printf("[GameScene] 传送带卡片拖拽结束: 位置在红线右侧，取消种植")
		return
	}

	// 计算放置的行列
	col := int((worldX - config.GridWorldStartX) / config.CellWidth)
	row := int((worldY - config.GridWorldStartY) / config.CellHeight)

	// 验证行列有效性
	if row < 0 || row >= config.GridRows || col < 0 || col >= config.GridColumns {
		log.Printf("[GameScene] 传送带卡片拖拽结束: 行列无效 row=%d, col=%d", row, col)
		return
	}

	removedCardType := s.conveyorBeltSystem.RemoveCard(s.dragConveyorCardIndex)
	if removedCardType == "" {
		log.Printf("[GameScene] 传送带卡片拖拽结束: 移除卡片失败")
		return
	}

	isExplosive := removedCardType == components.CardTypeExplodeONut

	entityID, err := entities.NewBowlingNutEntity(
		s.entityManager,
		s.resourceManager,
		row,
		col,
		isExplosive,
	)

	if err != nil {
		log.Printf("[GameScene] 传送带卡片拖拽结束: 创建保龄球坚果失败: %v", err)
		return
	}

	log.Printf("[GameScene] 传送带卡片拖拽种植成功: entityID=%d, type=%s, row=%d, col=%d",
		entityID, removedCardType, row+1, col+1)
}

// cancelConveyorDragPlanting 取消拖拽种植模式
func (s *GameScene) cancelConveyorDragPlanting() {
	if !s.isDragConveyorPlanting {
		return
	}

	s.conveyorBeltSystem.DeselectCard()
	s.destroyConveyorCardPreview()

	s.isDragConveyorPlanting = false
	s.dragConveyorCardIndex = -1
	s.dragConveyorCardType = ""
}
