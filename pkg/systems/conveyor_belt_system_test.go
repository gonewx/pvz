package systems

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
)

// createTestConveyorBeltSystem 创建测试用传送带系统
func createTestConveyorBeltSystem() (*ConveyorBeltSystem, *ecs.EntityManager) {
	em := ecs.NewEntityManager()
	system := NewConveyorBeltSystem(em, nil, nil)
	return system, em
}

func TestNewConveyorBeltSystem(t *testing.T) {
	system, em := createTestConveyorBeltSystem()

	// 验证系统初始化
	if system == nil {
		t.Fatal("Expected system to be non-nil")
	}

	// 验证传送带实体已创建
	beltEntity := system.GetBeltEntity()
	beltComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](em, beltEntity)
	if !ok {
		t.Fatal("Expected ConveyorBeltComponent to be attached to belt entity")
	}

	// 验证默认配置
	if beltComp.Capacity != components.DefaultConveyorCapacity {
		t.Errorf("Expected capacity %d, got %d", components.DefaultConveyorCapacity, beltComp.Capacity)
	}

	// Story 19.12: 验证 NextSpacing 初始化
	if beltComp.NextSpacing != config.ConveyorNutSpacing {
		t.Errorf("Expected NextSpacing %.1f, got %.1f", config.ConveyorNutSpacing, beltComp.NextSpacing)
	}
}

// Story 19.12: 测试传送带启动时第一个坚果在右侧生成
func TestConveyorBeltSystem_ActivateSpawnsFirstCardAtRight(t *testing.T) {
	system, em := createTestConveyorBeltSystem()

	// 激活传送带
	system.Activate()

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())

	// 验证第一个坚果已生成
	if len(beltComp.Cards) != 1 {
		t.Fatalf("Expected 1 card after activation, got %d", len(beltComp.Cards))
	}

	// 验证第一个坚果在右侧（PositionX = ConveyorBeltWidth）
	expectedX := config.ConveyorBeltWidth
	if beltComp.Cards[0].PositionX != expectedX {
		t.Errorf("Expected first card PositionX=%.1f, got %.1f", expectedX, beltComp.Cards[0].PositionX)
	}

	// 验证第一个坚果未到达左边缘
	if beltComp.Cards[0].IsAtLeftEdge {
		t.Error("Expected first card IsAtLeftEdge=false")
	}
}

func TestConveyorBeltSystem_Activate(t *testing.T) {
	system, em := createTestConveyorBeltSystem()

	// 初始状态应为未激活
	if system.IsActive() {
		t.Error("Expected system to be inactive initially")
	}

	// 激活
	system.Activate()
	if !system.IsActive() {
		t.Error("Expected system to be active after Activate()")
	}

	// 验证组件状态
	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())
	if !beltComp.IsActive {
		t.Error("Expected component IsActive to be true")
	}

	// 停用
	system.Deactivate()
	if system.IsActive() {
		t.Error("Expected system to be inactive after Deactivate()")
	}
}

// Story 19.12: 测试坚果位置随时间向左移动
func TestConveyorBeltSystem_CardsMove(t *testing.T) {
	system, em := createTestConveyorBeltSystem()

	// 激活传送带
	system.Activate()

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())
	initialX := beltComp.Cards[0].PositionX

	// 更新 1 秒
	system.Update(1.0)

	// 验证位置向左移动
	newX := beltComp.Cards[0].PositionX
	expectedMove := config.ConveyorBeltMoveSpeed * 1.0
	expectedX := initialX - expectedMove

	if newX > expectedX+0.1 || newX < expectedX-0.1 {
		t.Errorf("Expected card PositionX=%.1f after 1s, got %.1f (moved %.1f)", expectedX, newX, initialX-newX)
	}
}

// Story 19.12: 测试坚果到达左边缘后停止
func TestConveyorBeltSystem_CardStopsAtLeftEdge(t *testing.T) {
	system, em := createTestConveyorBeltSystem()

	// 激活传送带
	system.Activate()

	// 移动足够长时间让卡片到达左边缘
	// ConveyorBeltWidth = 450, ConveyorBeltMoveSpeed = 30, 需要 15 秒
	for i := 0; i < 200; i++ {
		system.Update(0.1) // 20秒
	}

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())

	// 验证第一个卡片已到达左边缘
	if len(beltComp.Cards) == 0 {
		t.Fatal("Expected at least 1 card")
	}

	firstCard := beltComp.Cards[0]
	if firstCard.PositionX != config.ConveyorNutStopX {
		t.Errorf("Expected card PositionX=%.1f (stopX), got %.1f", config.ConveyorNutStopX, firstCard.PositionX)
	}

	if !firstCard.IsAtLeftEdge {
		t.Error("Expected card IsAtLeftEdge=true")
	}
}

// Story 19.12: 测试间隔生成逻辑
func TestConveyorBeltSystem_SpawnsBySpacing(t *testing.T) {
	system, em := createTestConveyorBeltSystem()

	// 激活传送带
	system.Activate()

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())

	// 初始应该有 1 个卡片
	if len(beltComp.Cards) != 1 {
		t.Fatalf("Expected 1 card after activation, got %d", len(beltComp.Cards))
	}

	// 模拟传送带移动足够距离（移动 > spacing 才生成新卡片）
	// spacing = 80, speed = 30, 需要 80/30 ≈ 2.67 秒
	for i := 0; i < 40; i++ {
		system.Update(0.1) // 4秒
	}

	// 验证应该生成了更多卡片
	if len(beltComp.Cards) <= 1 {
		t.Errorf("Expected more than 1 card after spacing distance, got %d", len(beltComp.Cards))
	}
}

func TestConveyorBeltSystem_CardGeneration(t *testing.T) {
	system, _ := createTestConveyorBeltSystem()

	// 生成大量卡片统计分布
	wallnutCount := 0
	explodeCount := 0
	total := 10000

	for i := 0; i < total; i++ {
		cardType := system.generateCard()
		if cardType == components.CardTypeWallnutBowling {
			wallnutCount++
		} else if cardType == components.CardTypeExplodeONut {
			explodeCount++
		}
	}

	// 验证分布接近 85/15（允许 5% 误差）
	wallnutRatio := float64(wallnutCount) / float64(total)
	if wallnutRatio < 0.80 || wallnutRatio > 0.90 {
		t.Errorf("Expected wallnut ratio ~0.85, got %.3f", wallnutRatio)
	}

	explodeRatio := float64(explodeCount) / float64(total)
	if explodeRatio < 0.10 || explodeRatio > 0.20 {
		t.Errorf("Expected explode ratio ~0.15, got %.3f", explodeRatio)
	}
}

func TestConveyorBeltSystem_CapacityLimit(t *testing.T) {
	system, em := createTestConveyorBeltSystem()

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())
	beltComp.Capacity = 5

	// 填满传送带
	for i := 0; i < 5; i++ {
		system.addCard(beltComp, components.CardTypeWallnutBowling)
	}

	if len(beltComp.Cards) != 5 {
		t.Errorf("Expected 5 cards, got %d", len(beltComp.Cards))
	}

	// 尝试添加第 6 张卡片（应该失败）
	added := system.addCard(beltComp, components.CardTypeWallnutBowling)
	if added {
		t.Error("Expected addCard to return false when full")
	}

	if len(beltComp.Cards) != 5 {
		t.Errorf("Expected 5 cards after failed add, got %d", len(beltComp.Cards))
	}
}

func TestConveyorBeltSystem_RemoveCard(t *testing.T) {
	system, em := createTestConveyorBeltSystem()

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())

	// 添加卡片
	system.addCard(beltComp, components.CardTypeWallnutBowling)
	system.addCard(beltComp, components.CardTypeExplodeONut)
	system.addCard(beltComp, components.CardTypeWallnutBowling)

	// 移除中间卡片
	removedType := system.RemoveCard(1)
	if removedType != components.CardTypeExplodeONut {
		t.Errorf("Expected removed card type '%s', got '%s'", components.CardTypeExplodeONut, removedType)
	}

	// 验证剩余卡片
	if len(beltComp.Cards) != 2 {
		t.Errorf("Expected 2 cards after removal, got %d", len(beltComp.Cards))
	}

	// 移除无效索引
	removedType = system.RemoveCard(10)
	if removedType != "" {
		t.Errorf("Expected empty string for invalid index, got '%s'", removedType)
	}
}

func TestConveyorBeltSystem_FinalWave(t *testing.T) {
	system, em := createTestConveyorBeltSystem()

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())

	// 添加 5 张普通坚果
	for i := 0; i < 5; i++ {
		system.addCard(beltComp, components.CardTypeWallnutBowling)
	}

	initialCount := len(beltComp.Cards)

	// 触发最终波
	system.OnFinalWave()

	// 验证添加了 2-3 个爆炸坚果
	addedCount := len(beltComp.Cards) - initialCount
	if addedCount < 2 || addedCount > 3 {
		t.Errorf("Expected 2-3 explode-o-nuts added, got %d", addedCount)
	}

	// 验证前面的卡片是爆炸坚果
	explodeCount := 0
	for i := 0; i < addedCount && i < len(beltComp.Cards); i++ {
		if beltComp.Cards[i].CardType == components.CardTypeExplodeONut {
			explodeCount++
		}
	}

	if explodeCount < 2 {
		t.Errorf("Expected at least 2 explode-o-nuts at front, got %d", explodeCount)
	}

	// Story 19.12: 验证插入的卡片在左边缘位置
	if len(beltComp.Cards) > 0 {
		firstCard := beltComp.Cards[0]
		if firstCard.PositionX != config.ConveyorNutStopX {
			t.Errorf("Expected inserted card PositionX=%.1f, got %.1f", config.ConveyorNutStopX, firstCard.PositionX)
		}
		if !firstCard.IsAtLeftEdge {
			t.Error("Expected inserted card IsAtLeftEdge=true")
		}
	}

	// 验证 FinalWaveTriggered 标志
	if !beltComp.FinalWaveTriggered {
		t.Error("Expected FinalWaveTriggered to be true")
	}

	// 再次触发不应该添加更多
	countBefore := len(beltComp.Cards)
	system.OnFinalWave()
	if len(beltComp.Cards) != countBefore {
		t.Error("Expected no change after second OnFinalWave call")
	}
}

// Story 19.12: 测试基于 PositionX 的选中逻辑
func TestConveyorBeltSystem_SelectAndDeselect(t *testing.T) {
	system, em := createTestConveyorBeltSystem()

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())

	// 添加卡片（在传送带范围内）
	system.addCard(beltComp, components.CardTypeWallnutBowling)
	beltComp.Cards[0].PositionX = 100.0 // 在传送带范围内

	// 初始无选中
	if system.GetSelectedCard() != "" {
		t.Error("Expected no card selected initially")
	}

	// 选中卡片
	if !system.SelectCard(0) {
		t.Error("Expected SelectCard to return true")
	}

	selectedType := system.GetSelectedCard()
	if selectedType != components.CardTypeWallnutBowling {
		t.Errorf("Expected selected card '%s', got '%s'", components.CardTypeWallnutBowling, selectedType)
	}

	// 取消选中
	system.DeselectCard()
	if system.GetSelectedCard() != "" {
		t.Error("Expected no card selected after Deselect")
	}

	// Story 19.12: 卡片完全在传送带外时不可选中
	system.addCard(beltComp, components.CardTypeExplodeONut)
	beltComp.Cards[1].PositionX = config.ConveyorBeltWidth + 100 // 完全在传送带外

	if system.SelectCard(1) {
		t.Error("Expected SelectCard to return false for card outside belt")
	}
}

func TestConveyorBeltSystem_PlacementValidation(t *testing.T) {
	system, _ := createTestConveyorBeltSystem()

	// 使用 config.GridWorldStartX = 255, config.CellWidth = 80, config.BowlingRedLineColumn = 3
	// 红线 X = 255 + 3 * 80 = 495

	// 测试红线左侧（有效）
	// 第 0 列中心: 255 + 0.5 * 80 = 295
	if !system.IsPlacementValid(295.0) {
		t.Error("Expected placement at column 0 to be valid")
	}

	// 第 2 列中心: 255 + 2.5 * 80 = 455
	if !system.IsPlacementValid(455.0) {
		t.Error("Expected placement at column 2 to be valid")
	}

	// 测试红线右侧（无效）
	// 第 3 列中心: 255 + 3.5 * 80 = 535
	if system.IsPlacementValid(535.0) {
		t.Error("Expected placement at column 3 to be invalid")
	}

	// 第 5 列中心: 255 + 5.5 * 80 = 695
	if system.IsPlacementValid(695.0) {
		t.Error("Expected placement at column 5 to be invalid")
	}
}

func TestConveyorBeltSystem_SetCardPool(t *testing.T) {
	system, _ := createTestConveyorBeltSystem()

	// 设置自定义卡片池（100% 爆炸坚果）
	customPool := []CardPoolEntry{
		{Type: components.CardTypeExplodeONut, Weight: 100},
	}
	system.SetCardPool(customPool)

	// 验证生成的都是爆炸坚果
	for i := 0; i < 100; i++ {
		cardType := system.generateCard()
		if cardType != components.CardTypeExplodeONut {
			t.Errorf("Expected all cards to be explode_o_nut with custom pool, got '%s'", cardType)
			break
		}
	}
}

// Story 19.12: 测试 SetNutSpacing
func TestConveyorBeltSystem_SetNutSpacing(t *testing.T) {
	system, _ := createTestConveyorBeltSystem()

	// 设置新间隔
	newSpacing := 120.0
	system.SetNutSpacing(newSpacing)

	// 验证间隔已更新
	if system.nutSpacing != newSpacing {
		t.Errorf("Expected nutSpacing=%.1f, got %.1f", newSpacing, system.nutSpacing)
	}

	// 设置无效间隔（应该被忽略）
	system.SetNutSpacing(-10.0)
	if system.nutSpacing != newSpacing {
		t.Errorf("Expected nutSpacing to remain %.1f after invalid set, got %.1f", newSpacing, system.nutSpacing)
	}
}

func TestConveyorBeltSystem_Update(t *testing.T) {
	system, em := createTestConveyorBeltSystem()

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())

	// 未激活时不更新
	system.Update(0.1)
	if len(beltComp.Cards) > 0 {
		t.Error("Expected no cards when inactive")
	}

	// 激活并更新
	system.Activate()

	// Story 19.12: 激活时应该已经生成第一个卡片
	if len(beltComp.Cards) == 0 {
		t.Error("Expected cards to be generated after Activate")
	}

	// 更新多次，应该生成更多卡片
	initialCount := len(beltComp.Cards)
	for i := 0; i < 100; i++ {
		system.Update(0.1) // 10秒
	}

	if len(beltComp.Cards) <= initialCount {
		t.Error("Expected more cards to be generated after Update")
	}
}

// Story 19.12: 测试点击检测基于 PositionX
func TestConveyorBeltSystem_GetCardAtPosition(t *testing.T) {
	system, em := createTestConveyorBeltSystem()

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())

	// 添加一个卡片
	system.addCard(beltComp, components.CardTypeWallnutBowling)
	beltComp.Cards[0].PositionX = 50.0 // 卡片在传送带 X=50 位置

	// 模拟屏幕坐标
	conveyorX := 100.0 // 传送带左上角 X
	cardStartY := 20.0 // 卡片起始 Y
	cardWidth := 40.0
	cardHeight := 60.0

	// 卡片屏幕位置: conveyorX + PositionX = 100 + 50 = 150
	// 点击在卡片范围内 (150-190, 20-80)
	cardIndex := system.GetCardAtPosition(160.0, 40.0, conveyorX, cardStartY, cardWidth, cardHeight)
	if cardIndex != 0 {
		t.Errorf("Expected cardIndex=0 when clicking on card, got %d", cardIndex)
	}

	// 点击在卡片范围外
	cardIndex = system.GetCardAtPosition(300.0, 40.0, conveyorX, cardStartY, cardWidth, cardHeight)
	if cardIndex != -1 {
		t.Errorf("Expected cardIndex=-1 when clicking outside card, got %d", cardIndex)
	}

	// Y 坐标在卡片范围外
	cardIndex = system.GetCardAtPosition(160.0, 100.0, conveyorX, cardStartY, cardWidth, cardHeight)
	if cardIndex != -1 {
		t.Errorf("Expected cardIndex=-1 when Y is outside card, got %d", cardIndex)
	}
}

// Story 19.12: 测试大间隔随机生成
func TestConveyorBeltSystem_LargeSpacing(t *testing.T) {
	system, em := createTestConveyorBeltSystem()

	// 激活并运行足够长时间
	system.Activate()

	// 运行 100 秒，应该有机会触发大间隔
	for i := 0; i < 1000; i++ {
		system.Update(0.1)
	}

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())

	// 验证 NextSpacing 可能是大间隔（概率性测试）
	// 由于是随机的，我们只验证 NextSpacing 被正确设置
	if beltComp.NextSpacing < config.ConveyorNutSpacing {
		t.Errorf("Expected NextSpacing >= %.1f, got %.1f", config.ConveyorNutSpacing, beltComp.NextSpacing)
	}
}
