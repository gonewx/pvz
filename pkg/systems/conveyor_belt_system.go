package systems

import (
	"log"
	"math/rand"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// CardPoolEntry 卡片池条目
// 定义卡片类型和权重
type CardPoolEntry struct {
	Type   string // 卡片类型: "wallnut_bowling", "explode_o_nut"
	Weight int    // 权重值（越大越容易生成）
}

// ConveyorBeltSystem 传送带系统
// Story 19.5: 管理传送带动画、卡片生成和交互
//
// 核心设计：
// - 履带和卡片使用相同速度移动，保持相对静止
// - 卡片从右侧进入，持续向左移动
// - 到达左边界或碰到前面停止的卡片时停止
type ConveyorBeltSystem struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager

	// beltEntity 传送带实体ID
	beltEntity ecs.EntityID

	// cardPool 卡片池配置
	cardPool []CardPoolEntry

	// 常量缓存
	beltSpeed float64 // 传送带速度（像素/秒），同时控制履带和卡片

	// 布局参数缓存（需要在运行时计算）
	cardWidth      float64 // 卡片宽度
	beltWidth      float64 // 传送带宽度
	leftPadding    float64 // 左侧内边距
	movingSpacing  float64 // 移动时的间距
	stoppedSpacing float64 // 停止后的间距
}

// NewConveyorBeltSystem 创建传送带系统
func NewConveyorBeltSystem(em *ecs.EntityManager, gs *game.GameState, rm *game.ResourceManager) *ConveyorBeltSystem {
	system := &ConveyorBeltSystem{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
		beltSpeed:       config.ConveyorBeltSpeed,
		cardWidth:       100.0 * config.ConveyorCardScale, // 默认卡片宽度
		beltWidth:       config.ConveyorBeltWidth,
		leftPadding:     config.ConveyorBeltLeftPadding,
		movingSpacing:   config.ConveyorCardMovingSpacing,
		stoppedSpacing:  config.ConveyorCardStoppedSpacing,
	}

	// 初始化默认卡片池（85% 普通坚果，15% 爆炸坚果）
	system.cardPool = []CardPoolEntry{
		{Type: components.CardTypeWallnutBowling, Weight: 85},
		{Type: components.CardTypeExplodeONut, Weight: 15},
	}

	// 创建传送带实体
	system.beltEntity = em.CreateEntity()

	// 初始化传送带组件
	beltComp := components.NewConveyorBeltComponent()
	em.AddComponent(system.beltEntity, beltComp)

	log.Printf("[ConveyorBeltSystem] Initialized (Entity ID: %d), capacity=%d, interval=%.1fs",
		system.beltEntity, beltComp.Capacity, beltComp.GenerationInterval)

	return system
}

// Update 更新系统状态
func (s *ConveyorBeltSystem) Update(dt float64) {
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
	if !ok {
		return
	}

	// 只有激活状态才更新
	if !beltComp.IsActive {
		return
	}

	// 1. 更新传动动画（履带滚动）
	s.updateBeltAnimation(dt, beltComp)

	// 2. 更新卡片生成
	s.updateCardGeneration(dt, beltComp)

	// 3. 更新卡片移动（与履带同速度）
	s.updateCardMovement(dt, beltComp)
}

// updateBeltAnimation 更新传动动画
func (s *ConveyorBeltSystem) updateBeltAnimation(dt float64, beltComp *components.ConveyorBeltComponent) {
	// 滚动偏移量持续增加
	beltComp.ScrollOffset += s.beltSpeed * dt

	// 环绕处理
	maxOffset := 180.0
	if beltComp.ScrollOffset >= maxOffset {
		beltComp.ScrollOffset -= maxOffset
	}
}

// updateCardGeneration 更新卡片生成
func (s *ConveyorBeltSystem) updateCardGeneration(dt float64, beltComp *components.ConveyorBeltComponent) {
	if beltComp.IsFull() {
		return
	}

	beltComp.GenerationTimer -= dt

	if beltComp.GenerationTimer <= 0 {
		cardType := s.generateCard()
		s.addCard(beltComp, cardType)
		beltComp.GenerationTimer = beltComp.GenerationInterval

		log.Printf("[ConveyorBeltSystem] Generated card: %s, queue length: %d/%d",
			cardType, len(beltComp.Cards), beltComp.Capacity)
	}
}

// updateCardMovement 更新卡片移动
// 核心逻辑：
// - 所有未停止的卡片以相同速度向左移动
// - 到达左边界或碰到前面停止的卡片时停止
func (s *ConveyorBeltSystem) updateCardMovement(dt float64, beltComp *components.ConveyorBeltComponent) {
	moveDistance := s.beltSpeed * dt

	for i := range beltComp.Cards {
		card := &beltComp.Cards[i]

		if card.IsStopped {
			continue
		}

		// 向左移动
		card.PositionX -= moveDistance

		// 计算停止位置
		stopX := s.calculateStopPosition(beltComp, i)

		// 检查是否应该停止
		if card.PositionX <= stopX {
			card.PositionX = stopX
			card.IsStopped = true
		}
	}
}

// calculateStopPosition 计算卡片的停止位置
func (s *ConveyorBeltSystem) calculateStopPosition(beltComp *components.ConveyorBeltComponent, cardIndex int) float64 {
	// 左边界位置
	leftBoundary := s.leftPadding

	// 如果是第一张卡片，停在左边界
	if cardIndex == 0 {
		return leftBoundary
	}

	// 否则，查找前面已停止的卡片
	for i := cardIndex - 1; i >= 0; i-- {
		prevCard := &beltComp.Cards[i]
		if prevCard.IsStopped {
			// 停在前面卡片的右侧 + 停止间距
			return prevCard.PositionX + s.cardWidth + s.stoppedSpacing
		}
	}

	// 没有找到停止的卡片，停在左边界
	return leftBoundary
}

// generateCard 按权重生成卡片类型
func (s *ConveyorBeltSystem) generateCard() string {
	totalWeight := 0
	for _, entry := range s.cardPool {
		totalWeight += entry.Weight
	}

	if totalWeight <= 0 {
		return s.cardPool[0].Type
	}

	roll := rand.Intn(totalWeight)
	cumulative := 0
	for _, entry := range s.cardPool {
		cumulative += entry.Weight
		if roll < cumulative {
			return entry.Type
		}
	}

	return s.cardPool[0].Type
}

// addCard 添加卡片到传送带
func (s *ConveyorBeltSystem) addCard(beltComp *components.ConveyorBeltComponent, cardType string) bool {
	if beltComp.IsFull() {
		return false
	}

	// 计算新卡片的起始位置
	// 新卡片从传送带右边界外开始，与最右边的卡片保持移动间距
	var startX float64

	// 默认起始位置（传送带右边界外）
	defaultStartX := s.beltWidth + s.cardWidth

	if len(beltComp.Cards) > 0 {
		// 找到最右边的卡片
		rightmostX := 0.0
		for _, card := range beltComp.Cards {
			if card.PositionX > rightmostX {
				rightmostX = card.PositionX
			}
		}

		// 新卡片位置 = 最右边卡片右边缘 + 移动间距
		desiredX := rightmostX + s.cardWidth + s.movingSpacing

		// 取两者中较大的值：确保新卡片至少在传送带外，同时保持间距
		if desiredX > defaultStartX {
			startX = desiredX
		} else {
			startX = defaultStartX
		}
	} else {
		// 第一张卡片，从右侧外部开始
		startX = defaultStartX
	}

	card := components.ConveyorCard{
		CardType:  cardType,
		PositionX: startX,
		IsStopped: false,
	}

	beltComp.Cards = append(beltComp.Cards, card)
	return true
}

// OnFinalWave 最终波特殊处理
func (s *ConveyorBeltSystem) OnFinalWave() {
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
	if !ok {
		return
	}

	if beltComp.FinalWaveTriggered {
		return
	}
	beltComp.FinalWaveTriggered = true

	count := 2 + rand.Intn(2)
	for i := 0; i < count; i++ {
		s.insertCardToFront(beltComp, components.CardTypeExplodeONut)
	}

	log.Printf("[ConveyorBeltSystem] Final wave: inserted %d explode-o-nuts", count)
}

// insertCardToFront 将卡片插入队列前端
func (s *ConveyorBeltSystem) insertCardToFront(beltComp *components.ConveyorBeltComponent, cardType string) {
	// 新卡片插入到最左边，已停止状态
	card := components.ConveyorCard{
		CardType:  cardType,
		PositionX: s.leftPadding,
		IsStopped: true,
	}

	// 将现有卡片向右移动（增加位置偏移）
	offset := s.cardWidth + s.stoppedSpacing
	for i := range beltComp.Cards {
		beltComp.Cards[i].PositionX += offset
		// 如果卡片被挤出边界，标记为未停止，让它继续移动
		if beltComp.Cards[i].IsStopped && beltComp.Cards[i].PositionX > s.beltWidth {
			beltComp.Cards[i].IsStopped = false
		}
	}

	// 插入到前端
	beltComp.Cards = append([]components.ConveyorCard{card}, beltComp.Cards...)

	// 如果超出容量，移除最后一张
	if len(beltComp.Cards) > beltComp.Capacity {
		beltComp.Cards = beltComp.Cards[:beltComp.Capacity]
	}
}

// GetCardAtPosition 获取指定屏幕位置的卡片索引
func (s *ConveyorBeltSystem) GetCardAtPosition(x, y float64, conveyorX, conveyorY float64, cardWidth, cardHeight float64) int {
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
	if !ok {
		return -1
	}

	if cardWidth <= 0 {
		cardWidth = s.cardWidth
	}
	if cardHeight <= 0 {
		cardHeight = 140.0 * config.ConveyorCardScale
	}

	// 检查 Y 坐标是否在传送带范围内
	if y < conveyorY || y > conveyorY+cardHeight {
		return -1
	}

	// 计算点击位置相对于传送带的 X
	relativeX := x - conveyorX

	// 检查每张卡片（包括移动中的卡片）
	for i, card := range beltComp.Cards {
		// 检查点击是否在卡片范围内
		if relativeX >= card.PositionX && relativeX <= card.PositionX+cardWidth {
			return i
		}
	}

	return -1
}

// GetCardAtPositionForHover 获取指定屏幕位置的卡片索引（用于悬停检测）
// 与 GetCardAtPosition 相同，但用于悬停检测场景
func (s *ConveyorBeltSystem) GetCardAtPositionForHover(x, y float64, conveyorX, conveyorY float64, cardWidth, cardHeight float64) int {
	return s.GetCardAtPosition(x, y, conveyorX, conveyorY, cardWidth, cardHeight)
}

// RemoveCard 移除并返回指定索引的卡片
func (s *ConveyorBeltSystem) RemoveCard(index int) string {
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
	if !ok {
		return ""
	}

	if index < 0 || index >= len(beltComp.Cards) {
		return ""
	}

	cardType := beltComp.Cards[index].CardType

	// 从队列中移除
	beltComp.Cards = append(beltComp.Cards[:index], beltComp.Cards[index+1:]...)

	// 清除选中状态
	beltComp.SelectedCardIndex = -1

	// 重新检查停止状态（前面的卡片被移除后，后面的卡片可能需要继续移动）
	for i := range beltComp.Cards {
		if beltComp.Cards[i].IsStopped {
			// 重新计算是否应该停止
			stopX := s.calculateStopPosition(beltComp, i)
			if beltComp.Cards[i].PositionX > stopX+0.1 {
				// 卡片位置大于停止位置，需要继续移动
				beltComp.Cards[i].IsStopped = false
			}
		}
	}

	log.Printf("[ConveyorBeltSystem] Removed card: %s, queue length: %d/%d",
		cardType, len(beltComp.Cards), beltComp.Capacity)

	return cardType
}

// SelectCard 选中指定索引的卡片
func (s *ConveyorBeltSystem) SelectCard(index int) bool {
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
	if !ok {
		return false
	}

	if index < 0 || index >= len(beltComp.Cards) {
		return false
	}

	// 允许选中任意卡片（包括移动中的卡片）
	beltComp.SelectedCardIndex = index
	log.Printf("[ConveyorBeltSystem] 选中卡片: index=%d, type=%s, isStopped=%v",
		index, beltComp.Cards[index].CardType, beltComp.Cards[index].IsStopped)
	return true
}

// DeselectCard 取消选中
func (s *ConveyorBeltSystem) DeselectCard() {
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
	if !ok {
		return
	}
	beltComp.SelectedCardIndex = -1
}

// GetSelectedCard 获取当前选中的卡片类型
func (s *ConveyorBeltSystem) GetSelectedCard() string {
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
	if !ok {
		return ""
	}

	if beltComp.SelectedCardIndex < 0 || beltComp.SelectedCardIndex >= len(beltComp.Cards) {
		return ""
	}

	return beltComp.Cards[beltComp.SelectedCardIndex].CardType
}

// Activate 激活传送带
func (s *ConveyorBeltSystem) Activate() {
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
	if !ok {
		return
	}

	beltComp.IsActive = true
	beltComp.GenerationTimer = 0
	log.Printf("[ConveyorBeltSystem] Activated")
}

// Deactivate 停用传送带
func (s *ConveyorBeltSystem) Deactivate() {
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
	if !ok {
		return
	}

	beltComp.IsActive = false
	log.Printf("[ConveyorBeltSystem] Deactivated")
}

// IsActive 检查传送带是否激活
func (s *ConveyorBeltSystem) IsActive() bool {
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
	if !ok {
		return false
	}
	return beltComp.IsActive
}

// GetBeltEntity 获取传送带实体ID
func (s *ConveyorBeltSystem) GetBeltEntity() ecs.EntityID {
	return s.beltEntity
}

// SetCardPool 设置卡片池配置
func (s *ConveyorBeltSystem) SetCardPool(pool []CardPoolEntry) {
	if len(pool) > 0 {
		s.cardPool = pool
		log.Printf("[ConveyorBeltSystem] Card pool updated: %d entries", len(pool))
	}
}

// SetGenerationInterval 设置卡片生成间隔
func (s *ConveyorBeltSystem) SetGenerationInterval(interval float64) {
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
	if !ok {
		return
	}

	if interval > 0 {
		beltComp.GenerationInterval = interval
		log.Printf("[ConveyorBeltSystem] Generation interval set to %.1fs", interval)
	}
}

// IsPlacementValid 检查放置位置是否有效（红线左侧）
func (s *ConveyorBeltSystem) IsPlacementValid(worldX float64) bool {
	col := int((worldX - config.GridWorldStartX) / config.CellWidth)
	return col >= 0 && col < config.BowlingRedLineColumn
}

// SetCardWidth 设置卡片宽度（在渲染时调用）
func (s *ConveyorBeltSystem) SetCardWidth(width float64) {
	if width > 0 {
		s.cardWidth = width
	}
}
