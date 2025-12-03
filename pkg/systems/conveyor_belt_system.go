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
// 此系统负责：
// - 更新传动动画（6行交错滚动）
// - 按权重生成卡片
// - 管理卡片滑入动画
// - 处理最终波特殊生成
//
// 遵循零耦合原则：
// - 通过组件查询访问传送带状态
// - 使用回调函数与外部系统通信
type ConveyorBeltSystem struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager

	// beltEntity 传送带实体ID
	beltEntity ecs.EntityID

	// cardPool 卡片池配置
	cardPool []CardPoolEntry

	// 常量缓存
	beltAnimSpeed float64 // 传动动画速度（像素/秒）
	slideInSpeed  float64 // 卡片滑入速度（单位/秒）
}

// NewConveyorBeltSystem 创建传送带系统
//
// 参数：
//   - em: 实体管理器
//   - gs: 游戏状态
//   - rm: 资源管理器
//
// 返回：
//   - 传送带系统实例
func NewConveyorBeltSystem(em *ecs.EntityManager, gs *game.GameState, rm *game.ResourceManager) *ConveyorBeltSystem {
	system := &ConveyorBeltSystem{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
		beltAnimSpeed:   config.ConveyorBeltAnimSpeed,
		slideInSpeed:    2.0, // 卡片滑入速度，0.5秒完成
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
//
// 参数：
//   - dt: 时间增量（秒）
func (s *ConveyorBeltSystem) Update(dt float64) {
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
	if !ok {
		return
	}

	// 只有激活状态才更新
	if !beltComp.IsActive {
		return
	}

	// 1. 更新传动动画
	s.updateBeltAnimation(dt, beltComp)

	// 2. 更新卡片生成
	s.updateCardGeneration(dt, beltComp)

	// 3. 更新卡片滑入动画
	s.updateCardSlideIn(dt, beltComp)
}

// updateBeltAnimation 更新传动动画
// 6行交错滚动效果
func (s *ConveyorBeltSystem) updateBeltAnimation(dt float64, beltComp *components.ConveyorBeltComponent) {
	// 滚动偏移量持续增加
	beltComp.ScrollOffset += s.beltAnimSpeed * dt

	// 环绕处理（假设纹理高度约 30 像素每行，6行 = 180）
	// 实际值在渲染时根据纹理计算
	maxOffset := 180.0 // 预设最大偏移，实际会在渲染时取模
	if beltComp.ScrollOffset >= maxOffset {
		beltComp.ScrollOffset -= maxOffset
	}
}

// updateCardGeneration 更新卡片生成
func (s *ConveyorBeltSystem) updateCardGeneration(dt float64, beltComp *components.ConveyorBeltComponent) {
	// 如果传送带已满，不生成新卡片
	if beltComp.IsFull() {
		return
	}

	// 更新生成计时器
	beltComp.GenerationTimer -= dt

	// 计时器到期，生成新卡片
	if beltComp.GenerationTimer <= 0 {
		cardType := s.generateCard()
		s.addCard(beltComp, cardType)

		// 重置计时器
		beltComp.GenerationTimer = beltComp.GenerationInterval

		log.Printf("[ConveyorBeltSystem] Generated card: %s, queue length: %d/%d",
			cardType, len(beltComp.Cards), beltComp.Capacity)
	}
}

// updateCardSlideIn 更新卡片滑入动画
func (s *ConveyorBeltSystem) updateCardSlideIn(dt float64, beltComp *components.ConveyorBeltComponent) {
	for i := range beltComp.Cards {
		card := &beltComp.Cards[i]
		if card.SlideProgress < 1.0 {
			card.SlideProgress += s.slideInSpeed * dt
			if card.SlideProgress > 1.0 {
				card.SlideProgress = 1.0
			}
		}
	}
}

// generateCard 按权重生成卡片类型
//
// 返回：
//   - 生成的卡片类型字符串
func (s *ConveyorBeltSystem) generateCard() string {
	// 计算总权重
	totalWeight := 0
	for _, entry := range s.cardPool {
		totalWeight += entry.Weight
	}

	if totalWeight <= 0 {
		return s.cardPool[0].Type // fallback
	}

	// 随机选择
	roll := rand.Intn(totalWeight)
	cumulative := 0
	for _, entry := range s.cardPool {
		cumulative += entry.Weight
		if roll < cumulative {
			return entry.Type
		}
	}

	return s.cardPool[0].Type // fallback
}

// addCard 添加卡片到传送带
func (s *ConveyorBeltSystem) addCard(beltComp *components.ConveyorBeltComponent, cardType string) bool {
	if beltComp.IsFull() {
		return false
	}

	// 新卡片添加到队列末尾（最右侧）
	card := components.ConveyorCard{
		CardType:      cardType,
		SlideProgress: 0.0, // 从0开始滑入动画
		SlotIndex:     len(beltComp.Cards),
	}

	beltComp.Cards = append(beltComp.Cards, card)
	return true
}

// OnFinalWave 最终波特殊处理
// 强制插入 2-3 个爆炸坚果到队列前端
func (s *ConveyorBeltSystem) OnFinalWave() {
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
	if !ok {
		return
	}

	// 避免重复触发
	if beltComp.FinalWaveTriggered {
		return
	}
	beltComp.FinalWaveTriggered = true

	// 强制插入 2-3 个爆炸坚果
	count := 2 + rand.Intn(2) // 2 或 3

	for i := 0; i < count; i++ {
		s.insertCardToFront(beltComp, components.CardTypeExplodeONut)
	}

	log.Printf("[ConveyorBeltSystem] Final wave: inserted %d explode-o-nuts", count)
}

// insertCardToFront 将卡片插入队列前端
func (s *ConveyorBeltSystem) insertCardToFront(beltComp *components.ConveyorBeltComponent, cardType string) {
	// 创建新卡片
	card := components.ConveyorCard{
		CardType:      cardType,
		SlideProgress: 1.0, // 立即显示，无滑入动画
		SlotIndex:     0,
	}

	// 更新现有卡片的槽位索引
	for i := range beltComp.Cards {
		beltComp.Cards[i].SlotIndex++
	}

	// 插入到前端
	beltComp.Cards = append([]components.ConveyorCard{card}, beltComp.Cards...)

	// 如果超出容量，移除最后一张
	if len(beltComp.Cards) > beltComp.Capacity {
		beltComp.Cards = beltComp.Cards[:beltComp.Capacity]
	}
}

// GetCardAtPosition 获取指定屏幕位置的卡片索引
//
// 参数：
//   - x, y: 屏幕坐标
//   - conveyorX, conveyorY: 传送带左上角位置
//   - cardWidth, cardHeight: 卡片尺寸（用于点击检测），0表示使用默认值
//
// 返回：
//   - 卡片索引，-1 表示没有卡片
func (s *ConveyorBeltSystem) GetCardAtPosition(x, y float64, conveyorX, conveyorY float64, cardWidth, cardHeight float64) int {
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
	if !ok {
		return -1
	}

	// 计算卡片区域参数 - 使用传入的参数，如果为0则使用默认值
	if cardWidth <= 0 {
		cardWidth = config.ConveyorCardWidth
	}
	if cardHeight <= 0 {
		cardHeight = config.ConveyorCardHeight
	}
	cardSpacing := config.ConveyorCardSpacing

	// 检查 Y 坐标是否在传送带范围内
	if y < conveyorY || y > conveyorY+cardHeight {
		return -1
	}

	// 计算点击位置对应的槽位
	relativeX := x - conveyorX
	if relativeX < 0 {
		return -1
	}

	slotWidth := cardWidth + cardSpacing
	slotIndex := int(relativeX / slotWidth)

	// 检查是否有卡片在该槽位
	for i, card := range beltComp.Cards {
		if card.SlotIndex == slotIndex && card.SlideProgress >= 0.9 {
			return i
		}
	}

	return -1
}

// RemoveCard 移除并返回指定索引的卡片
//
// 参数：
//   - index: 卡片在队列中的索引
//
// 返回：
//   - 卡片类型，空字符串表示移除失败
func (s *ConveyorBeltSystem) RemoveCard(index int) string {
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
	if !ok {
		return ""
	}

	if index < 0 || index >= len(beltComp.Cards) {
		return ""
	}

	// 获取卡片类型
	cardType := beltComp.Cards[index].CardType

	// 从队列中移除
	beltComp.Cards = append(beltComp.Cards[:index], beltComp.Cards[index+1:]...)

	// 更新剩余卡片的槽位索引
	for i := range beltComp.Cards {
		beltComp.Cards[i].SlotIndex = i
	}

	// 清除选中状态
	beltComp.SelectedCardIndex = -1

	log.Printf("[ConveyorBeltSystem] Removed card: %s, queue length: %d/%d",
		cardType, len(beltComp.Cards), beltComp.Capacity)

	return cardType
}

// SelectCard 选中指定索引的卡片
//
// 参数：
//   - index: 卡片索引
//
// 返回：
//   - 是否选中成功
func (s *ConveyorBeltSystem) SelectCard(index int) bool {
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
	if !ok {
		return false
	}

	if index < 0 || index >= len(beltComp.Cards) {
		return false
	}

	// 检查卡片滑入动画是否完成
	if beltComp.Cards[index].SlideProgress < 0.9 {
		return false
	}

	beltComp.SelectedCardIndex = index
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
//
// 返回：
//   - 卡片类型，空字符串表示没有选中
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
	beltComp.GenerationTimer = 0 // 立即生成第一张卡片

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
//
// 参数：
//   - worldX: 世界坐标 X
//
// 返回：
//   - 是否可以放置
func (s *ConveyorBeltSystem) IsPlacementValid(worldX float64) bool {
	// 计算列号
	col := int((worldX - config.GridWorldStartX) / config.CellWidth)

	// 只能放在红线左侧（列 < 3，即第 0、1、2 列）
	return col >= 0 && col < config.BowlingRedLineColumn
}
