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
// Story 19.5 & 19.12: 管理传送带动画、卡片生成和交互
//
// 此系统负责：
// - 更新传动动画（6行交错滚动）
// - 按权重生成卡片
// - 管理卡片移动（随传送带向左移动）
// - 处理最终波特殊生成
//
// Story 19.12 修正：
// - 坚果以固定间隔摆放，随传送带被动向左移动
// - 使用距离驱动生成，而非时间计时器
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
	moveSpeed     float64 // 传送带移动速度（像素/秒）
	nutSpacing    float64 // 坚果基础间隔（像素）
	stopX         float64 // 坚果停止的左边缘 X 位置
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
		moveSpeed:       config.ConveyorBeltMoveSpeed,
		nutSpacing:      config.ConveyorNutSpacing,
		stopX:           config.ConveyorNutStopX,
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
	beltComp.NextSpacing = system.nutSpacing
	em.AddComponent(system.beltEntity, beltComp)

	log.Printf("[ConveyorBeltSystem] Initialized (Entity ID: %d), capacity=%d, moveSpeed=%.1f px/s, spacing=%.1f px",
		system.beltEntity, beltComp.Capacity, system.moveSpeed, system.nutSpacing)

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

	// 2. Story 19.12: 更新坚果移动和生成
	s.updateNutMovementAndSpawning(dt, beltComp)
}

// updateBeltAnimation 更新传动动画
// 履带一直匀速滚动，不管是否有卡片在滑入
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

// updateNutMovementAndSpawning 更新坚果移动和生成
// Story 19.12: 距离驱动的生成逻辑
func (s *ConveyorBeltSystem) updateNutMovementAndSpawning(dt float64, beltComp *components.ConveyorBeltComponent) {
	// 1. 更新所有卡片位置（向左移动）
	for i := range beltComp.Cards {
		card := &beltComp.Cards[i]
		if !card.IsAtLeftEdge {
			card.PositionX -= s.moveSpeed * dt
			if card.PositionX <= s.stopX {
				card.PositionX = s.stopX
				card.IsAtLeftEdge = true
			}
		}
	}

	// 2. 检查是否需要生成新坚果
	if beltComp.IsFull() {
		return // 传送带已满
	}

	if len(beltComp.Cards) == 0 {
		return // 第一个坚果在 Activate() 中生成
	}

	// 找到最右侧坚果（队列最后一个）
	rightmostCard := &beltComp.Cards[len(beltComp.Cards)-1]

	// 获取传送带宽度
	beltWidth := s.getBeltWidth()

	// 当最右侧坚果移动了足够距离时，生成新坚果
	// 生成阈值 = 传送带右边缘 - 下一个间隔
	spawnThreshold := beltWidth - beltComp.NextSpacing
	if rightmostCard.PositionX <= spawnThreshold {
		s.spawnCardAtRight(beltComp)

		// 计算下一个间隔（可能是大间隔）
		beltComp.NextSpacing = s.nutSpacing
		if rand.Float64() < config.ConveyorLargeSpacingChance {
			beltComp.NextSpacing *= config.ConveyorLargeSpacingMultiplier
		}
	}
}

// spawnCardAtRight 在传送带右侧生成新卡片
func (s *ConveyorBeltSystem) spawnCardAtRight(beltComp *components.ConveyorBeltComponent) {
	if beltComp.IsFull() {
		return
	}

	cardType := s.generateCard()
	beltWidth := s.getBeltWidth()

	card := components.ConveyorCard{
		CardType:     cardType,
		PositionX:    beltWidth, // 从右边缘开始
		IsAtLeftEdge: false,
	}

	beltComp.Cards = append(beltComp.Cards, card)

	log.Printf("[ConveyorBeltSystem] Spawned card at right: %s, posX=%.1f, queue=%d/%d",
		cardType, beltWidth, len(beltComp.Cards), beltComp.Capacity)
}

// getBeltWidth 获取传送带宽度
func (s *ConveyorBeltSystem) getBeltWidth() float64 {
	return config.ConveyorBeltWidth
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

// addCard 添加卡片到传送带（用于测试）
// Story 19.12: 新卡片添加到右侧，使用 PositionX
func (s *ConveyorBeltSystem) addCard(beltComp *components.ConveyorBeltComponent, cardType string) bool {
	if beltComp.IsFull() {
		return false
	}

	beltWidth := s.getBeltWidth()

	// 新卡片添加到队列末尾（最右侧）
	card := components.ConveyorCard{
		CardType:     cardType,
		PositionX:    beltWidth, // 从右边缘开始
		IsAtLeftEdge: false,
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
// Story 19.12: 用于最终波强制插入爆炸坚果
func (s *ConveyorBeltSystem) insertCardToFront(beltComp *components.ConveyorBeltComponent, cardType string) {
	// 创建新卡片（在左边缘位置，立即可见）
	card := components.ConveyorCard{
		CardType:     cardType,
		PositionX:    s.stopX, // 在左边缘位置
		IsAtLeftEdge: true,    // 已到达左边缘
	}

	// 插入到前端
	beltComp.Cards = append([]components.ConveyorCard{card}, beltComp.Cards...)

	// 如果超出容量，移除最后一张
	if len(beltComp.Cards) > beltComp.Capacity {
		beltComp.Cards = beltComp.Cards[:beltComp.Capacity]
	}
}

// GetCardAtPosition 获取指定屏幕位置的卡片索引
// Story 19.12: 合并原来的两个方法，使用 PositionX 计算实时位置
//
// 参数：
//   - x, y: 屏幕坐标
//   - conveyorX: 传送带左上角X位置
//   - cardStartY: 卡片起始Y位置
//   - cardWidth, cardHeight: 卡片尺寸
//
// 返回：
//   - 卡片索引，-1 表示没有卡片
func (s *ConveyorBeltSystem) GetCardAtPosition(x, y float64, conveyorX, cardStartY float64, cardWidth, cardHeight float64) int {
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
	if !ok {
		return -1
	}

	// 检查 Y 坐标是否在卡片范围内
	if y < cardStartY || y > cardStartY+cardHeight {
		return -1
	}

	beltWidth := s.getBeltWidth()

	// Story 19.12: 使用 PositionX 计算卡片实时屏幕位置
	for i, card := range beltComp.Cards {
		// 卡片屏幕 X 位置 = 传送带 X + 卡片局部 X
		cardScreenX := conveyorX + card.PositionX

		// 检查卡片是否在传送带可见范围内
		if card.PositionX > beltWidth {
			continue // 完全在传送带外，跳过
		}

		// 检查点击是否在卡片区域内
		if x >= cardScreenX && x <= cardScreenX+cardWidth {
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

	// Story 19.12: 不再需要更新 SlotIndex，因为使用 PositionX 来定位

	// 清除选中状态
	beltComp.SelectedCardIndex = -1

	log.Printf("[ConveyorBeltSystem] Removed card: %s, queue length: %d/%d",
		cardType, len(beltComp.Cards), beltComp.Capacity)

	return cardType
}

// SelectCard 选中指定索引的卡片
// Story 19.12: 基于 PositionX 判断卡片可见性
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

	// Story 19.12: 允许选中任何可见的卡片（无论是否到达左边缘）
	// 卡片只要在传送带范围内就可以被选中
	beltWidth := s.getBeltWidth()
	card := &beltComp.Cards[index]

	// 检查卡片是否在传送带可见范围内（至少部分可见）
	if card.PositionX > beltWidth {
		return false // 完全在传送带外，不可选中
	}

	beltComp.SelectedCardIndex = index
	log.Printf("[ConveyorBeltSystem] Card selected: index=%d, type=%s, posX=%.1f, atLeftEdge=%v",
		index, card.CardType, card.PositionX, card.IsAtLeftEdge)
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
// Story 19.12: 激活时在右侧生成第一个坚果
func (s *ConveyorBeltSystem) Activate() {
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, s.beltEntity)
	if !ok {
		return
	}

	beltComp.IsActive = true

	// Story 19.12: 激活时在右侧生成第一个坚果
	if len(beltComp.Cards) == 0 {
		s.spawnCardAtRight(beltComp)
	}

	// 初始化下一个间隔
	beltComp.NextSpacing = s.nutSpacing

	log.Printf("[ConveyorBeltSystem] Activated, first card spawned at right")
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

// SetNutSpacing 设置坚果间隔
// Story 19.12: 替代原来的 SetGenerationInterval
func (s *ConveyorBeltSystem) SetNutSpacing(spacing float64) {
	if spacing > 0 {
		s.nutSpacing = spacing
		log.Printf("[ConveyorBeltSystem] Nut spacing set to %.1f px", spacing)
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
