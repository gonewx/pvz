package systems

import (
	"testing"

	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
)

// TestSeedChooserInputSystem_IsPlantSelectedOrPending 测试植物选中状态检测
func TestSeedChooserInputSystem_IsPlantSelectedOrPending(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.SeedChooserPlants = []string{} // 重置选卡状态

	// 创建渲染系统（仅用于测试，不会实际渲染）
	levelConfig := &config.LevelConfig{
		AvailablePlants: []string{"peashooter", "sunflower", "wallnut"},
	}
	renderSystem := &SeedChooserRenderSystem{
		entityManager: em,
		gameState:     gs,
		levelConfig:   levelConfig,
	}

	// 创建输入系统
	inputSystem := NewSeedChooserInputSystem(em, gs, renderSystem, levelConfig)

	// 测试1：未选中的植物
	if inputSystem.IsPlantSelectedOrPending("peashooter") {
		t.Error("Expected peashooter to not be selected or pending")
	}

	// 测试2：添加到待处理列表
	inputSystem.pendingSlotPlants = append(inputSystem.pendingSlotPlants, "peashooter")
	if !inputSystem.IsPlantSelectedOrPending("peashooter") {
		t.Error("Expected peashooter to be pending")
	}

	// 测试3：添加到 GameState
	gs.AddSeedChooserPlant("sunflower")
	if !inputSystem.IsPlantSelectedOrPending("sunflower") {
		t.Error("Expected sunflower to be selected in GameState")
	}

	// 测试4：未选中的植物仍然返回 false
	if inputSystem.IsPlantSelectedOrPending("wallnut") {
		t.Error("Expected wallnut to not be selected or pending")
	}
}

// TestSeedChooserInputSystem_IsSeedChooserFullIncludingPending 测试卡槽满状态检测
func TestSeedChooserInputSystem_IsSeedChooserFullIncludingPending(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.SeedChooserPlants = []string{} // 重置选卡状态

	levelConfig := &config.LevelConfig{
		AvailablePlants: []string{"peashooter", "sunflower", "wallnut", "cherrybomb", "potatomine", "snowpea", "chomper"},
	}
	renderSystem := &SeedChooserRenderSystem{
		entityManager: em,
		gameState:     gs,
		levelConfig:   levelConfig,
	}

	inputSystem := NewSeedChooserInputSystem(em, gs, renderSystem, levelConfig)

	// 测试1：空卡槽
	if inputSystem.IsSeedChooserFullIncludingPending() {
		t.Error("Expected seed chooser to not be full when empty")
	}

	// 测试2：添加 5 个植物到 GameState
	gs.AddSeedChooserPlant("peashooter")
	gs.AddSeedChooserPlant("sunflower")
	gs.AddSeedChooserPlant("wallnut")
	gs.AddSeedChooserPlant("cherrybomb")
	gs.AddSeedChooserPlant("potatomine")

	if inputSystem.IsSeedChooserFullIncludingPending() {
		t.Error("Expected seed chooser to not be full with 5 plants")
	}

	// 测试3：添加 1 个待处理植物，总共 6 个
	inputSystem.pendingSlotPlants = append(inputSystem.pendingSlotPlants, "snowpea")
	if !inputSystem.IsSeedChooserFullIncludingPending() {
		t.Error("Expected seed chooser to be full with 5 selected + 1 pending = 6")
	}
}

// TestSeedChooserInputSystem_GetFlyingCards 测试获取飞行卡片
func TestSeedChooserInputSystem_GetFlyingCards(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.SeedChooserPlants = []string{} // 重置选卡状态

	levelConfig := &config.LevelConfig{}
	renderSystem := &SeedChooserRenderSystem{
		entityManager: em,
		gameState:     gs,
		levelConfig:   levelConfig,
	}

	inputSystem := NewSeedChooserInputSystem(em, gs, renderSystem, levelConfig)

	// 测试1：初始状态无飞行卡片
	cards := inputSystem.GetFlyingCards()
	if len(cards) != 0 {
		t.Errorf("Expected 0 flying cards, got %d", len(cards))
	}

	// 测试2：添加飞行卡片
	inputSystem.flyingCards = append(inputSystem.flyingCards, FlyingCard{
		PlantID:   "peashooter",
		StartX:    100,
		StartY:    200,
		EndX:      300,
		EndY:      100,
		Progress:  0.5,
		IsFlying:  true,
		Direction: true,
	})

	cards = inputSystem.GetFlyingCards()
	if len(cards) != 1 {
		t.Errorf("Expected 1 flying card, got %d", len(cards))
	}
	if cards[0].PlantID != "peashooter" {
		t.Errorf("Expected plant ID 'peashooter', got '%s'", cards[0].PlantID)
	}
	if cards[0].Progress != 0.5 {
		t.Errorf("Expected progress 0.5, got %f", cards[0].Progress)
	}
}

// TestSeedChooserInputSystem_HoverState 测试悬停状态
func TestSeedChooserInputSystem_HoverState(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.SeedChooserPlants = []string{} // 重置选卡状态

	levelConfig := &config.LevelConfig{}
	renderSystem := &SeedChooserRenderSystem{
		entityManager: em,
		gameState:     gs,
		levelConfig:   levelConfig,
	}

	inputSystem := NewSeedChooserInputSystem(em, gs, renderSystem, levelConfig)

	// 测试1：初始状态无悬停
	if inputSystem.GetHoveredPlantID() != "" {
		t.Error("Expected no hovered plant initially")
	}
	if inputSystem.GetHoveredZombieType() != "" {
		t.Error("Expected no hovered zombie initially")
	}
	if inputSystem.IsHoveringButton() {
		t.Error("Expected not hovering button initially")
	}

	// 测试2：设置悬停状态
	inputSystem.hoveredPlantID = "sunflower"
	inputSystem.hoveredZombieType = "zombie"
	inputSystem.isHoveringButton = true
	inputSystem.hoveredMouseX = 100
	inputSystem.hoveredMouseY = 200

	if inputSystem.GetHoveredPlantID() != "sunflower" {
		t.Errorf("Expected hovered plant 'sunflower', got '%s'", inputSystem.GetHoveredPlantID())
	}
	if inputSystem.GetHoveredZombieType() != "zombie" {
		t.Errorf("Expected hovered zombie 'zombie', got '%s'", inputSystem.GetHoveredZombieType())
	}
	if !inputSystem.IsHoveringButton() {
		t.Error("Expected hovering button")
	}

	mouseX, mouseY := inputSystem.GetHoveredMousePosition()
	if mouseX != 100 || mouseY != 200 {
		t.Errorf("Expected mouse position (100, 200), got (%d, %d)", mouseX, mouseY)
	}
}

// TestSeedChooserInputSystem_ShouldShowHandCursor 测试手形光标显示
func TestSeedChooserInputSystem_ShouldShowHandCursor(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.SeedChooserPlants = []string{} // 重置选卡状态

	levelConfig := &config.LevelConfig{}
	renderSystem := &SeedChooserRenderSystem{
		entityManager: em,
		gameState:     gs,
		levelConfig:   levelConfig,
	}

	inputSystem := NewSeedChooserInputSystem(em, gs, renderSystem, levelConfig)

	// 测试1：初始状态不显示手形光标
	if inputSystem.ShouldShowHandCursor() {
		t.Error("Expected no hand cursor initially")
	}

	// 测试2：悬停在卡片上
	inputSystem.isHoveringCard = true
	if !inputSystem.ShouldShowHandCursor() {
		t.Error("Expected hand cursor when hovering card")
	}

	// 测试3：悬停在按钮上
	inputSystem.isHoveringCard = false
	inputSystem.isHoveringButton = true
	if !inputSystem.ShouldShowHandCursor() {
		t.Error("Expected hand cursor when hovering button")
	}

	// 测试4：悬停在卡槽卡片上
	inputSystem.isHoveringButton = false
	inputSystem.isHoveringSlotCard = true
	if !inputSystem.ShouldShowHandCursor() {
		t.Error("Expected hand cursor when hovering slot card")
	}
}

// TestSeedChooserInputSystem_RemovePendingPlant 测试移除待处理植物
func TestSeedChooserInputSystem_RemovePendingPlant(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.SeedChooserPlants = []string{} // 重置选卡状态

	levelConfig := &config.LevelConfig{}
	renderSystem := &SeedChooserRenderSystem{
		entityManager: em,
		gameState:     gs,
		levelConfig:   levelConfig,
	}

	inputSystem := NewSeedChooserInputSystem(em, gs, renderSystem, levelConfig)

	// 添加待处理植物
	inputSystem.pendingSlotPlants = []string{"peashooter", "sunflower", "wallnut"}

	// 测试1：移除中间的植物
	inputSystem.removePendingPlant("sunflower")
	if len(inputSystem.pendingSlotPlants) != 2 {
		t.Errorf("Expected 2 pending plants, got %d", len(inputSystem.pendingSlotPlants))
	}
	if inputSystem.pendingSlotPlants[0] != "peashooter" || inputSystem.pendingSlotPlants[1] != "wallnut" {
		t.Errorf("Expected [peashooter, wallnut], got %v", inputSystem.pendingSlotPlants)
	}

	// 测试2：移除第一个植物
	inputSystem.removePendingPlant("peashooter")
	if len(inputSystem.pendingSlotPlants) != 1 {
		t.Errorf("Expected 1 pending plant, got %d", len(inputSystem.pendingSlotPlants))
	}
	if inputSystem.pendingSlotPlants[0] != "wallnut" {
		t.Errorf("Expected [wallnut], got %v", inputSystem.pendingSlotPlants)
	}

	// 测试3：移除不存在的植物（不应报错）
	inputSystem.removePendingPlant("cherrybomb")
	if len(inputSystem.pendingSlotPlants) != 1 {
		t.Errorf("Expected 1 pending plant after removing non-existent, got %d", len(inputSystem.pendingSlotPlants))
	}
}

// TestFlyingCard_Position 测试飞行卡片位置计算
func TestFlyingCard_Position(t *testing.T) {
	card := FlyingCard{
		PlantID:   "peashooter",
		StartX:    0,
		StartY:    100,
		EndX:      200,
		EndY:      0,
		Progress:  0.5,
		IsFlying:  true,
		Direction: true,
	}

	// 测试中间位置
	x, y := GetFlyingCardPosition(card)

	// 预期: X 应该在 0 和 200 之间
	// Y 使用抛物线轨迹，会高于线性插值位置（即低于起点和终点的线性中点）
	if x < 0 || x > 200 {
		t.Errorf("Expected X between 0 and 200, got %f", x)
	}
	// Y 由于抛物线弧度会比线性插值更低（在屏幕坐标系中，Y 向下增加，弧度使其向上凸起）
	// 这里只验证 Y 不是 NaN 或异常值
	if y != y { // 检查 NaN
		t.Errorf("Y is NaN")
	}

	// 测试起始位置
	card.Progress = 0
	x, y = GetFlyingCardPosition(card)
	if x != 0 || y != 100 {
		t.Errorf("Expected start position (0, 100), got (%f, %f)", x, y)
	}

	// 测试结束位置
	card.Progress = 1
	x, y = GetFlyingCardPosition(card)
	if x != 200 || y != 0 {
		t.Errorf("Expected end position (200, 0), got (%f, %f)", x, y)
	}
}
