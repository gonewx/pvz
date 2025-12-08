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
// Story 19.12: 支持动态调节系统（三阶段权重/间隔、空带保底、满带降频、危机保底）
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

	// cardPool 卡片池配置（默认值，被动态配置覆盖）
	cardPool []CardPoolEntry

	// 常量缓存
	beltSpeed float64 // 传送带速度（像素/秒），同时控制履带和卡片

	// 布局参数缓存（需要在运行时计算）
	cardWidth      float64 // 卡片宽度
	beltWidth      float64 // 传送带宽度
	leftPadding    float64 // 左侧内边距
	minSpacing float64 // 卡片最小间距
	startOffsetX   float64 // 卡片起始位置偏移

	// Story 19.12: 动态调节配置
	phaseConfigs      []config.PhaseConfig           // 各阶段配置
	dynamicAdjustment *config.DynamicAdjustmentConfig // 动态调节参数
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
		minSpacing:      config.ConveyorCardMinSpacing,
		startOffsetX:    config.ConveyorCardStartOffsetX,
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

	// Story 19.12: 动态调节检测
	// 2. 检查空带补发保底
	s.checkEmptyBeltEmergency(dt, beltComp)

	// 3. 检查满带降频调节
	s.checkFullBeltThrottle(dt, beltComp)

	// 4. 检查危机爆炸坚果保底
	s.checkCrisisExplodeNut(beltComp)

	// 5. 更新卡片生成（使用动态间隔）
	s.updateCardGeneration(dt, beltComp)

	// 6. 更新卡片移动（与履带同速度）
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
// Story 19.12: 完全依赖动态时间间隔控制生成频率
func (s *ConveyorBeltSystem) updateCardGeneration(dt float64, beltComp *components.ConveyorBeltComponent) {
	if beltComp.IsFull() {
		return
	}

	// Story 19.12: 使用缓存的生成间隔，避免每帧随机
	// 如果 CurrentInterval <= 0，说明需要初始化
	if beltComp.CurrentInterval <= 0 {
		beltComp.CurrentInterval = s.getPhaseGenerationInterval()
	}

	currentInterval := beltComp.CurrentInterval

	// Story 19.12: 如果处于降频状态，增加间隔
	if beltComp.IsThrottled {
		throttleMultiplier := 1.5 // 默认降频倍率
		if s.dynamicAdjustment != nil && s.dynamicAdjustment.FullBeltThrottleMultiplier > 0 {
			throttleMultiplier = s.dynamicAdjustment.FullBeltThrottleMultiplier
		}
		currentInterval *= throttleMultiplier
	}

	// 更新生成计时器
	beltComp.GenerationTimer += dt

	// 检查是否达到生成间隔
	if beltComp.GenerationTimer < currentInterval {
		return
	}

	// 生成新卡片
	cardType := s.generateCard(beltComp)
	s.addCard(beltComp, cardType)
	// 重置计时器并计算下一次的生成间隔
	beltComp.GenerationTimer = 0
	beltComp.CurrentInterval = s.getPhaseGenerationInterval()
}

// updateCardMovement 更新卡片移动
// 核心逻辑：
// - 所有未停止的卡片以相同速度向左移动
// - 到达左边界或碰到前面停止的卡片时停止
// - 移动中的卡片之间也保持最小间距
func (s *ConveyorBeltSystem) updateCardMovement(dt float64, beltComp *components.ConveyorBeltComponent) {
	moveDistance := s.beltSpeed * dt

	for i := range beltComp.Cards {
		card := &beltComp.Cards[i]

		if card.IsStopped {
			continue
		}

		// 向左移动
		newX := card.PositionX - moveDistance

		// 检查与前面卡片的最小间距（无论前面卡片是否停止）
		if i > 0 {
			prevCard := &beltComp.Cards[i-1]
			minX := prevCard.PositionX + s.cardWidth + s.minSpacing
			if newX < minX {
				newX = minX
			}
		}

		card.PositionX = newX

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
			return prevCard.PositionX + s.cardWidth + s.minSpacing
		}
	}

	// 没有找到停止的卡片，停在左边界
	return leftBoundary
}

// generateCard 按权重生成卡片类型
// Story 19.12: 支持动态卡片池和强制生成
func (s *ConveyorBeltSystem) generateCard(beltComp *components.ConveyorBeltComponent) string {
	// Story 19.12: 检查是否强制生成爆炸坚果
	if beltComp.ForceExplodeNut {
		beltComp.ForceExplodeNut = false
		if s.gameState != nil {
			beltComp.LastExplodeNutTime = s.gameState.LevelTime
		}
		log.Printf("[ConveyorBeltSystem] Forced explode-o-nut spawn (crisis response)")
		return components.CardTypeExplodeONut
	}

	// Story 19.12: 使用动态卡片池
	pool := s.getPhaseCardPool()

	totalWeight := 0
	for _, entry := range pool {
		totalWeight += entry.Weight
	}

	if totalWeight <= 0 {
		return pool[0].Type
	}

	roll := rand.Intn(totalWeight)
	cumulative := 0
	for _, entry := range pool {
		cumulative += entry.Weight
		if roll < cumulative {
			cardType := entry.Type
			// 记录爆炸坚果生成时间
			if cardType == components.CardTypeExplodeONut && s.gameState != nil {
				beltComp.LastExplodeNutTime = s.gameState.LevelTime
			}
			return cardType
		}
	}

	return pool[0].Type
}

// addCard 添加卡片到传送带
func (s *ConveyorBeltSystem) addCard(beltComp *components.ConveyorBeltComponent, cardType string) bool {
	if beltComp.IsFull() {
		return false
	}

	// 卡片始终从默认起始位置开始（传送带右边界外）
	startX := s.beltWidth + s.cardWidth + s.startOffsetX

	card := components.ConveyorCard{
		CardType:  cardType,
		PositionX: startX,
		IsStopped: false,
	}

	beltComp.Cards = append(beltComp.Cards, card)

	log.Printf("[ConveyorBeltSystem] Generated card: %s, queue length: %d/%d, startX=%.1f, timer=%.2f, interval=%.2f",
		cardType, len(beltComp.Cards), beltComp.Capacity, startX, beltComp.GenerationTimer, beltComp.CurrentInterval)

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
	offset := s.cardWidth + s.minSpacing
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
// 参数说明：
//   - x, y: 屏幕点击位置
//   - conveyorX: 传送带左边缘的屏幕 X 坐标
//   - conveyorY: 卡片区域的屏幕 Y 坐标
//   - cardWidth, cardHeight: 卡片尺寸
//
// 注意：只检测传送带可见区域内的卡片部分，被裁剪的部分不响应点击
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

	// 计算传送带可见区域边界（相对于传送带左边缘）
	// 与绘制逻辑保持一致：beltRightEdge = beltWidth - rightPadding
	rightBoundary := s.beltWidth - config.ConveyorBeltRightPadding

	// 检查每张卡片（包括移动中的卡片）
	for i, card := range beltComp.Cards {
		// 计算卡片的可见范围
		cardLeft := card.PositionX
		cardRight := card.PositionX + cardWidth

		// 右侧裁剪：如果卡片右边缘超出可见区域，截断到右边界
		if cardRight > rightBoundary {
			cardRight = rightBoundary
		}

		// 跳过完全不可见的卡片（卡片左边缘 >= 右边界）
		if cardLeft >= rightBoundary {
			continue
		}

		// 检查点击是否在卡片的可见范围内
		if relativeX >= cardLeft && relativeX <= cardRight {
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
	// Story 19.12: 激活时初始化生成间隔
	beltComp.CurrentInterval = s.getPhaseGenerationInterval()

	// 激活时立即生成第一张卡，让玩家可以立即开始操作
	cardType := s.generateCard(beltComp)
	s.addCard(beltComp, cardType)

	log.Printf("[ConveyorBeltSystem] Activated, interval=%.2fs, first card generated", beltComp.CurrentInterval)
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

// ========================================
// Story 19.12: 动态调节系统方法
// ========================================

// SetDynamicConfig 设置动态调节配置
// 在关卡加载时调用，从关卡配置中读取动态调节参数
func (s *ConveyorBeltSystem) SetDynamicConfig(phaseConfigs []config.PhaseConfig, dynamicAdjustment *config.DynamicAdjustmentConfig) {
	s.phaseConfigs = phaseConfigs
	s.dynamicAdjustment = dynamicAdjustment
	log.Printf("[ConveyorBeltSystem] Dynamic config set: %d phases, dynamicAdjustment=%v",
		len(phaseConfigs), dynamicAdjustment != nil)
}

// getLevelProgress 获取关卡进度百分比
// 返回 0.0 到 1.0 之间的值
func (s *ConveyorBeltSystem) getLevelProgress() float64 {
	currentWave, totalWaves := s.gameState.GetLevelProgress()
	if totalWaves == 0 {
		return 0.0
	}
	return float64(currentWave) / float64(totalWaves)
}

// getCurrentPhase 获取当前阶段
// 返回 1（前期）、2（中期）或 3（终盘）
func (s *ConveyorBeltSystem) getCurrentPhase() int {
	_, totalWaves := s.gameState.GetLevelProgress()

	// 简化波次支持：当波次过少时使用中期配置
	if totalWaves <= 2 {
		return 2 // 中期
	}

	progress := s.getLevelProgress()
	switch {
	case progress < 0.3:
		return 1 // 前期
	case progress < 0.7:
		return 2 // 中期
	default:
		return 3 // 终盘
	}
}

// getPhaseCardPool 获取当前阶段的卡片池
// 根据阶段返回不同的爆炸坚果权重
func (s *ConveyorBeltSystem) getPhaseCardPool() []CardPoolEntry {
	// 如果没有配置阶段配置，使用默认卡片池
	if len(s.phaseConfigs) == 0 {
		return s.cardPool
	}

	// 如果 gameState 为 nil，使用默认卡片池
	if s.gameState == nil {
		return s.cardPool
	}

	progress := s.getLevelProgress()

	// 查找当前进度对应的阶段配置
	var currentConfig *config.PhaseConfig
	for i := len(s.phaseConfigs) - 1; i >= 0; i-- {
		if progress >= s.phaseConfigs[i].ProgressThreshold {
			currentConfig = &s.phaseConfigs[i]
			break
		}
	}

	// 如果没有匹配的配置，使用第一个配置
	if currentConfig == nil && len(s.phaseConfigs) > 0 {
		currentConfig = &s.phaseConfigs[0]
	}

	// 如果仍然没有配置，使用默认卡片池
	if currentConfig == nil {
		return s.cardPool
	}

	// 构建卡片池：普通坚果 + 爆炸坚果
	wallnutWeight := 100 - currentConfig.ExplodeNutWeight
	return []CardPoolEntry{
		{Type: components.CardTypeWallnutBowling, Weight: wallnutWeight},
		{Type: components.CardTypeExplodeONut, Weight: currentConfig.ExplodeNutWeight},
	}
}

// getPhaseGenerationInterval 获取当前阶段的生成间隔
// 返回当前阶段配置的随机间隔
func (s *ConveyorBeltSystem) getPhaseGenerationInterval() float64 {
	// 如果没有配置阶段配置，返回默认间隔
	if len(s.phaseConfigs) == 0 {
		beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
		if ok {
			return beltComp.GenerationInterval
		}
		return components.DefaultCardGenerationInterval
	}

	// 如果 gameState 为 nil，返回默认间隔
	if s.gameState == nil {
		return components.DefaultCardGenerationInterval
	}

	progress := s.getLevelProgress()

	// 查找当前进度对应的阶段配置
	var currentConfig *config.PhaseConfig
	for i := len(s.phaseConfigs) - 1; i >= 0; i-- {
		if progress >= s.phaseConfigs[i].ProgressThreshold {
			currentConfig = &s.phaseConfigs[i]
			break
		}
	}

	// 如果没有匹配的配置，使用第一个配置
	if currentConfig == nil && len(s.phaseConfigs) > 0 {
		currentConfig = &s.phaseConfigs[0]
	}

	// 如果仍然没有配置，返回默认间隔
	if currentConfig == nil {
		return components.DefaultCardGenerationInterval
	}

	// 在 intervalMin 和 intervalMax 之间随机
	intervalRange := currentConfig.IntervalMax - currentConfig.IntervalMin
	return currentConfig.IntervalMin + rand.Float64()*intervalRange
}

// checkEmptyBeltEmergency 检查空带补发保底
// AC 3: 传送带连续 3 秒为空时，立即触发一次额外生成（必出普通坚果）
func (s *ConveyorBeltSystem) checkEmptyBeltEmergency(dt float64, beltComp *components.ConveyorBeltComponent) {
	// 获取空带阈值
	threshold := 3.0 // 默认值
	if s.dynamicAdjustment != nil && s.dynamicAdjustment.EmptyBeltThreshold > 0 {
		threshold = s.dynamicAdjustment.EmptyBeltThreshold
	}

	if beltComp.IsEmpty() {
		beltComp.EmptyDuration += dt
		if beltComp.EmptyDuration >= threshold {
			// 紧急补发普通坚果
			s.addCard(beltComp, components.CardTypeWallnutBowling)
			beltComp.EmptyDuration = 0
			// 同步重置正常生成计时器，避免空带保底和正常生成在短时间内连续触发
			beltComp.GenerationTimer = 0
			beltComp.CurrentInterval = s.getPhaseGenerationInterval()
			log.Printf("[ConveyorBeltSystem] Emergency: Belt empty for %.1fs, spawned wallnut", threshold)
		}
	} else {
		beltComp.EmptyDuration = 0
	}
}

// checkFullBeltThrottle 检查满带降频调节
// AC 4: 传送带连续 8 秒处于满状态时，刷新间隔提高 50%
func (s *ConveyorBeltSystem) checkFullBeltThrottle(dt float64, beltComp *components.ConveyorBeltComponent) {
	// 获取满带阈值
	threshold := 8.0 // 默认值
	if s.dynamicAdjustment != nil && s.dynamicAdjustment.FullBeltThreshold > 0 {
		threshold = s.dynamicAdjustment.FullBeltThreshold
	}

	if beltComp.IsFull() {
		beltComp.FullDuration += dt
		if beltComp.FullDuration >= threshold && !beltComp.IsThrottled {
			beltComp.IsThrottled = true
			log.Printf("[ConveyorBeltSystem] Throttle: Belt full for %.1fs, reducing spawn rate", threshold)
		}
	} else {
		beltComp.FullDuration = 0
		// 低于满容量 2 格时解除降频
		if beltComp.CardCount() <= beltComp.Capacity-2 {
			if beltComp.IsThrottled {
				log.Printf("[ConveyorBeltSystem] Throttle released: CardCount=%d, Capacity=%d",
					beltComp.CardCount(), beltComp.Capacity)
			}
			beltComp.IsThrottled = false
		}
	}
}

// checkCrisisExplodeNut 检查危机爆炸坚果保底
// AC 5: 检测到同一行有 2+ 僵尸距离安全线 ≤ 阈值时，强制下次生成爆炸坚果
func (s *ConveyorBeltSystem) checkCrisisExplodeNut(beltComp *components.ConveyorBeltComponent) {
	// 如果已经标记强制生成，等待生成
	if beltComp.ForceExplodeNut {
		return
	}

	// 如果没有动态调节配置，跳过危机检测
	if s.dynamicAdjustment == nil {
		return
	}

	// 检查上次生成爆炸坚果是否超过冷却时间
	levelTime := s.gameState.LevelTime
	if levelTime-beltComp.LastExplodeNutTime < s.dynamicAdjustment.CrisisExplodeNutCooldown {
		return
	}

	// 查询所有具有 BehaviorComponent 和 PositionComponent 的实体
	entities := ecs.GetEntitiesWith2[*components.BehaviorComponent, *components.PositionComponent](s.entityManager)

	// 安全线位置（家门口）
	safeLineX := config.GridWorldStartX

	// 按行统计接近安全线的僵尸数量
	laneCounts := make(map[int]int)
	for _, entity := range entities {
		behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entity)
		if !ok {
			continue
		}

		// 检查是否为僵尸（排除死亡中的僵尸）
		if !s.isActiveZombie(behaviorComp) {
			continue
		}

		posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entity)
		if !ok {
			continue
		}

		// 使用配置常量定义危机距离阈值
		if posComp.X <= safeLineX+s.dynamicAdjustment.CrisisDistanceThreshold {
			row := int((posComp.Y - config.GridWorldStartY) / config.CellHeight)
			laneCounts[row]++
		}
	}

	// 检查是否有行存在危机（使用配置常量定义僵尸数量阈值）
	for _, count := range laneCounts {
		if count >= s.dynamicAdjustment.CrisisZombieCount {
			beltComp.ForceExplodeNut = true
			log.Printf("[ConveyorBeltSystem] Crisis detected: forcing explode-o-nut spawn")
			break
		}
	}
}

// isActiveZombie 检查实体是否为活动状态的僵尸
func (s *ConveyorBeltSystem) isActiveZombie(behaviorComp *components.BehaviorComponent) bool {
	switch behaviorComp.Type {
	case components.BehaviorZombieBasic,
		components.BehaviorZombieEating,
		components.BehaviorZombieConehead,
		components.BehaviorZombieBuckethead,
		components.BehaviorZombieFlag:
		return true
	default:
		return false
	}
}
