package systems

import (
	"log"
	"math"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// FlyingCard 表示正在飞行中的卡片动画状态
type FlyingCard struct {
	PlantID   string  // 植物ID
	StartX    float64 // 起始X坐标
	StartY    float64 // 起始Y坐标
	EndX      float64 // 目标X坐标
	EndY      float64 // 目标Y坐标
	Progress  float64 // 动画进度 0.0-1.0
	IsFlying  bool    // 是否正在飞行
	Direction bool    // true=飞入卡槽, false=飞回仓库
	SlotIndex int     // 目标卡槽索引（仅飞入卡槽时有效）
}

// SeedChooserInputSystem 选卡界面交互系统
// Story 8.12: 处理选卡界面的点击检测、卡片选择/取消、按钮点击
type SeedChooserInputSystem struct {
	entityManager *ecs.EntityManager
	gameState     *game.GameState
	renderSystem  *SeedChooserRenderSystem
	levelConfig   *config.LevelConfig

	// 飞行动画状态
	flyingCards []FlyingCard

	// 待添加到卡槽的植物（飞行动画完成后才真正添加到 GameState）
	pendingSlotPlants []string

	// 悬停状态（用于 Tooltip 显示和手形光标）
	hoveredPlantID     string // 当前悬停的植物ID（空表示无悬停）
	hoveredZombieType  string // 当前悬停的僵尸类型（空表示无悬停）
	hoveredMouseX      int    // 鼠标 X 坐标
	hoveredMouseY      int    // 鼠标 Y 坐标
	isHoveringButton   bool   // 是否悬停在按钮上
	isHoveringCard     bool   // 是否悬停在可点击的卡片上
	isHoveringSlotCard bool   // 是否悬停在卡槽卡片上
	isHoveringZombie   bool   // 是否悬停在预览僵尸上
}

// NewSeedChooserInputSystem 创建选卡界面交互系统
func NewSeedChooserInputSystem(
	em *ecs.EntityManager,
	gs *game.GameState,
	rs *SeedChooserRenderSystem,
	levelConfig *config.LevelConfig,
) *SeedChooserInputSystem {
	return &SeedChooserInputSystem{
		entityManager:     em,
		gameState:         gs,
		renderSystem:      rs,
		levelConfig:       levelConfig,
		flyingCards:       make([]FlyingCard, 0),
		pendingSlotPlants: make([]string, 0),
	}
}

// Update 更新选卡界面交互
func (s *SeedChooserInputSystem) Update(deltaTime float64) {
	// 只在选卡阶段激活时处理输入
	if !s.gameState.IsSeedSelectionActive() {
		return
	}

	// 更新飞行动画
	s.updateFlyingCards(deltaTime)

	// 更新悬停状态（每帧检测）
	s.updateHoverState()

	// 检测点击
	justPressed, mouseX, mouseY := utils.IsJustTouchedOrClicked()
	if justPressed {
		s.handleClick(mouseX, mouseY)
	}
}

// updateHoverState 更新悬停状态
func (s *SeedChooserInputSystem) updateHoverState() {
	mouseX, mouseY := ebiten.CursorPosition()
	s.hoveredMouseX = mouseX
	s.hoveredMouseY = mouseY
	s.hoveredPlantID = ""
	s.hoveredZombieType = ""
	s.isHoveringButton = false
	s.isHoveringCard = false
	s.isHoveringSlotCard = false
	s.isHoveringZombie = false

	// 检查是否悬停在仓库卡片上
	availablePlants := s.renderSystem.GetAvailablePlants()
	for i, plantID := range availablePlants {
		x, y, width, height := s.renderSystem.GetInventoryCardBounds(i)
		if float64(mouseX) >= x && float64(mouseX) <= x+width &&
			float64(mouseY) >= y && float64(mouseY) <= y+height {
			s.hoveredPlantID = plantID
			// 只有未选中的卡片且卡槽未满时才是可点击的
			if !s.IsPlantSelectedOrPending(plantID) && !s.IsSeedChooserFullIncludingPending() {
				s.isHoveringCard = true
			}
			return
		}
	}

	// 检查是否悬停在卡槽上（已选中的卡片 + 已完成飞行的待确认卡片）
	selectedPlants := s.gameState.GetSeedChooserPlants()
	for i := 0; i < len(selectedPlants); i++ {
		x, y, width, height := s.renderSystem.GetSlotBounds(i)
		if float64(mouseX) >= x && float64(mouseX) <= x+width &&
			float64(mouseY) >= y && float64(mouseY) <= y+height {
			s.hoveredPlantID = selectedPlants[i]
			s.isHoveringSlotCard = true
			return
		}
	}

	// 检查是否悬停在按钮上
	bx, by, bw, bh := s.renderSystem.GetButtonBounds()
	if float64(mouseX) >= bx && float64(mouseX) <= bx+bw &&
		float64(mouseY) >= by && float64(mouseY) <= by+bh {
		s.isHoveringButton = true
		return
	}

	// 检查是否悬停在预览僵尸上
	// 查找 OpeningAnimationComponent 获取预览僵尸列表
	// 优先检测Y值更大的僵尸（渲染在前面/下方的僵尸）
	openingEntities := ecs.GetEntitiesWith1[*components.OpeningAnimationComponent](s.entityManager)
	var bestZombieType string
	var bestZombieY float64 = -1
	for _, entity := range openingEntities {
		openingComp, ok := ecs.GetComponent[*components.OpeningAnimationComponent](s.entityManager, entity)
		if !ok {
			continue
		}
		// 遍历预览僵尸，找到Y值最大的匹配僵尸
		for _, zombieEntity := range openingComp.ZombieEntities {
			posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieEntity)
			if !ok {
				continue
			}
			// 将世界坐标转换为屏幕坐标
			screenX := posComp.X - s.gameState.CameraX
			screenY := posComp.Y
			// 僵尸碰撞框大小（大约 60x100 像素）
			zombieWidth := 60.0
			zombieHeight := 100.0
			// 检查鼠标是否在僵尸范围内
			if float64(mouseX) >= screenX-zombieWidth/2 && float64(mouseX) <= screenX+zombieWidth/2 &&
				float64(mouseY) >= screenY-zombieHeight && float64(mouseY) <= screenY {
				// 获取僵尸类型，选择Y值最大的（渲染在最前面）
				if animCmd, ok := ecs.GetComponent[*components.AnimationCommandComponent](s.entityManager, zombieEntity); ok {
					if screenY > bestZombieY {
						bestZombieY = screenY
						bestZombieType = animCmd.UnitID
					}
				}
			}
		}
	}
	// 设置悬停的僵尸类型
	if bestZombieType != "" {
		s.hoveredZombieType = bestZombieType
		s.isHoveringZombie = true
	}
}

// handleClick 处理点击事件
func (s *SeedChooserInputSystem) handleClick(mouseX, mouseY int) {
	// 1. 检测"一起摇滚吧!"按钮点击
	if s.checkButtonClick(mouseX, mouseY) {
		return
	}

	// 2. 检测卡槽点击（取消选择）
	if s.checkSlotClick(mouseX, mouseY) {
		return
	}

	// 3. 检测仓库卡片点击（选择植物）
	s.checkInventoryClick(mouseX, mouseY)
}

// checkButtonClick 检测"一起摇滚吧!"按钮点击
func (s *SeedChooserInputSystem) checkButtonClick(mouseX, mouseY int) bool {
	// 检查卡槽是否已满
	if !s.gameState.IsSeedChooserFull() {
		return false // 按钮禁用状态，不响应点击
	}

	// 获取按钮边界
	x, y, width, height := s.renderSystem.GetButtonBounds()
	if float64(mouseX) >= x && float64(mouseX) <= x+width &&
		float64(mouseY) >= y && float64(mouseY) <= y+height {
		// 点击了按钮，确认选卡
		s.confirmSelection()
		return true
	}

	return false
}

// checkSlotClick 检测卡槽点击（取消选择）
func (s *SeedChooserInputSystem) checkSlotClick(mouseX, mouseY int) bool {
	selectedPlants := s.gameState.GetSeedChooserPlants()

	for i := 0; i < len(selectedPlants); i++ {
		x, y, width, height := s.renderSystem.GetSlotBounds(i)
		if float64(mouseX) >= x && float64(mouseX) <= x+width &&
			float64(mouseY) >= y && float64(mouseY) <= y+height {
			// 点击了已选卡片，取消选择
			plantID := selectedPlants[i]
			s.deselectPlant(plantID, i)
			return true
		}
	}

	return false
}

// checkInventoryClick 检测仓库卡片点击（选择植物）
func (s *SeedChooserInputSystem) checkInventoryClick(mouseX, mouseY int) {
	// 检查卡槽是否已满（包括待处理的飞行卡片）
	if s.IsSeedChooserFullIncludingPending() {
		return // 卡槽已满，不响应仓库点击
	}

	availablePlants := s.renderSystem.GetAvailablePlants()

	for i, plantID := range availablePlants {
		// 检查是否已被选中或正在飞行中
		if s.IsPlantSelectedOrPending(plantID) {
			continue // 已选中或正在飞行的卡片不响应点击
		}

		x, y, width, height := s.renderSystem.GetInventoryCardBounds(i)
		if float64(mouseX) >= x && float64(mouseX) <= x+width &&
			float64(mouseY) >= y && float64(mouseY) <= y+height {
			// 点击了仓库卡片，选择植物
			s.selectPlant(plantID, i)
			return
		}
	}
}

// selectPlant 选择植物（创建飞行动画，动画完成后才添加到卡槽）
func (s *SeedChooserInputSystem) selectPlant(plantID string, inventoryIndex int) {
	// 先添加到待处理列表（用于禁用状态显示和卡槽计数）
	s.pendingSlotPlants = append(s.pendingSlotPlants, plantID)

	// 播放选择音效
	s.playSeedLiftSound()

	// 获取起始和结束位置
	startX, startY, _, _ := s.renderSystem.GetInventoryCardBounds(inventoryIndex)
	// 计算目标卡槽索引：已选数量 + 待处理数量 - 1
	slotIndex := len(s.gameState.GetSeedChooserPlants()) + len(s.pendingSlotPlants) - 1
	endX, endY, _, _ := s.renderSystem.GetSlotBounds(slotIndex)

	// 创建飞行动画
	s.flyingCards = append(s.flyingCards, FlyingCard{
		PlantID:   plantID,
		StartX:    startX,
		StartY:    startY,
		EndX:      endX,
		EndY:      endY,
		Progress:  0,
		IsFlying:  true,
		Direction: true, // 飞入卡槽
		SlotIndex: slotIndex,
	})

	log.Printf("[SeedChooserInputSystem] 选择植物: %s (从仓库位置 %d 飞入卡槽 %d)", plantID, inventoryIndex, slotIndex)
}

// deselectPlant 取消选择植物（从卡槽移除）
func (s *SeedChooserInputSystem) deselectPlant(plantID string, slotIndex int) {
	// 从选择列表移除
	if !s.gameState.RemoveSeedChooserPlant(plantID) {
		return // 移除失败
	}

	// 播放取消音效（与选择共用同一音效）
	s.playSeedLiftSound()

	// 获取起始和结束位置
	startX, startY, _, _ := s.renderSystem.GetSlotBounds(slotIndex)

	// 找到植物在仓库中的原始位置
	availablePlants := s.renderSystem.GetAvailablePlants()
	var inventoryIndex int
	for i, p := range availablePlants {
		if p == plantID {
			inventoryIndex = i
			break
		}
	}
	endX, endY, _, _ := s.renderSystem.GetInventoryCardBounds(inventoryIndex)

	// 创建飞行动画
	s.flyingCards = append(s.flyingCards, FlyingCard{
		PlantID:   plantID,
		StartX:    startX,
		StartY:    startY,
		EndX:      endX,
		EndY:      endY,
		Progress:  0,
		IsFlying:  true,
		Direction: false, // 飞回仓库
	})

	log.Printf("[SeedChooserInputSystem] 取消选择植物: %s (从卡槽 %d 飞回仓库位置 %d)", plantID, slotIndex, inventoryIndex)
}

// confirmSelection 确认选卡
func (s *SeedChooserInputSystem) confirmSelection() {
	// 播放按钮点击音效
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_TAP")
	}

	// 退出选卡阶段
	s.gameState.ExitSeedSelection()

	log.Printf("[SeedChooserInputSystem] 确认选卡，已选植物: %v", s.gameState.GetSelectedPlants())
}

// playSeedLiftSound 播放选卡音效
func (s *SeedChooserInputSystem) playSeedLiftSound() {
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_SEEDLIFT")
	}
}

// updateFlyingCards 更新飞行卡片动画
func (s *SeedChooserInputSystem) updateFlyingCards(deltaTime float64) {
	// 过滤掉已完成的动画
	activeCards := make([]FlyingCard, 0)

	for i := range s.flyingCards {
		card := &s.flyingCards[i]
		if !card.IsFlying {
			continue
		}

		// 更新动画进度
		card.Progress += deltaTime / config.CardFlyDuration
		if card.Progress >= 1.0 {
			card.Progress = 1.0
			card.IsFlying = false

			// 飞行动画完成，处理卡片状态
			if card.Direction {
				// 飞入卡槽完成：从待处理列表移除，添加到 GameState
				s.removePendingPlant(card.PlantID)
				s.gameState.AddSeedChooserPlant(card.PlantID)
				log.Printf("[SeedChooserInputSystem] 卡片飞入完成，添加到卡槽: %s", card.PlantID)
			}
			// 飞回仓库不需要额外处理（已在 deselectPlant 中从 GameState 移除）
		} else {
			activeCards = append(activeCards, *card)
		}
	}

	s.flyingCards = activeCards
}

// removePendingPlant 从待处理列表中移除植物
func (s *SeedChooserInputSystem) removePendingPlant(plantID string) {
	for i, p := range s.pendingSlotPlants {
		if p == plantID {
			s.pendingSlotPlants = append(s.pendingSlotPlants[:i], s.pendingSlotPlants[i+1:]...)
			return
		}
	}
}

// IsPlantSelectedOrPending 检查植物是否已被选中或正在飞行中
func (s *SeedChooserInputSystem) IsPlantSelectedOrPending(plantID string) bool {
	// 检查 GameState 中已选
	if s.gameState.IsSeedChooserPlantSelected(plantID) {
		return true
	}
	// 检查待处理列表
	for _, p := range s.pendingSlotPlants {
		if p == plantID {
			return true
		}
	}
	return false
}

// IsSeedChooserFullIncludingPending 检查卡槽是否已满（包括待处理的飞行卡片）
func (s *SeedChooserInputSystem) IsSeedChooserFullIncludingPending() bool {
	totalSelected := len(s.gameState.GetSeedChooserPlants()) + len(s.pendingSlotPlants)
	return totalSelected >= config.SeedSlotCount
}

// IsPlantFlyingToSlot 检查植物是否正在飞向卡槽
func (s *SeedChooserInputSystem) IsPlantFlyingToSlot(plantID string) bool {
	for _, card := range s.flyingCards {
		if card.PlantID == plantID && card.Direction && card.IsFlying {
			return true
		}
	}
	return false
}

// ShouldShowHandCursor 检查是否应该显示手形光标
func (s *SeedChooserInputSystem) ShouldShowHandCursor() bool {
	return s.isHoveringCard || s.isHoveringSlotCard || s.isHoveringButton
}

// GetFlyingCards 获取当前飞行中的卡片列表（供渲染系统使用）
func (s *SeedChooserInputSystem) GetFlyingCards() []FlyingCard {
	return s.flyingCards
}

// GetFlyingCardPosition 获取飞行卡片的当前位置（抛物线轨迹）
func GetFlyingCardPosition(card FlyingCard) (x, y float64) {
	// 使用 ease-out 缓动
	t := easeOutQuadFloat(card.Progress)

	// 线性插值 X
	x = card.StartX + (card.EndX-card.StartX)*t

	// 线性插值 Y + 抛物线弧度
	linearY := card.StartY + (card.EndY-card.StartY)*t
	arcHeight := config.CardFlyArcHeight
	// 抛物线：在中点（t=0.5）达到最高点
	arc := -4 * arcHeight * (t - 0.5) * (t - 0.5) + arcHeight
	y = linearY - arc

	return x, y
}

// easeOutQuadFloat ease-out 二次缓动
func easeOutQuadFloat(t float64) float64 {
	return 1 - math.Pow(1-t, 2)
}

// HandleKeyInput 处理键盘输入（可选的快捷键）
func (s *SeedChooserInputSystem) HandleKeyInput() {
	// Enter 键确认选卡（如果卡槽已满）
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if s.gameState.IsSeedChooserFull() {
			s.confirmSelection()
		}
	}
}

// GetHoveredPlantID 获取当前悬停的植物ID
func (s *SeedChooserInputSystem) GetHoveredPlantID() string {
	return s.hoveredPlantID
}

// GetHoveredMousePosition 获取悬停时的鼠标位置
func (s *SeedChooserInputSystem) GetHoveredMousePosition() (int, int) {
	return s.hoveredMouseX, s.hoveredMouseY
}

// IsHoveringButton 检查是否悬停在按钮上
func (s *SeedChooserInputSystem) IsHoveringButton() bool {
	return s.isHoveringButton
}

// GetHoveredZombieType 获取当前悬停的僵尸类型
func (s *SeedChooserInputSystem) GetHoveredZombieType() string {
	return s.hoveredZombieType
}

// IsHoveringZombie 检查是否悬停在预览僵尸上
func (s *SeedChooserInputSystem) IsHoveringZombie() bool {
	return s.isHoveringZombie
}
