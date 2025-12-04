package systems

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
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

func TestConveyorBeltSystem_CardGeneration(t *testing.T) {
	system, em := createTestConveyorBeltSystem()

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

	_ = em // 避免未使用警告
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

	// 验证剩余卡片的类型
	if beltComp.Cards[0].CardType != components.CardTypeWallnutBowling {
		t.Errorf("Card 0: expected CardType '%s', got '%s'", components.CardTypeWallnutBowling, beltComp.Cards[0].CardType)
	}
	if beltComp.Cards[1].CardType != components.CardTypeWallnutBowling {
		t.Errorf("Card 1: expected CardType '%s', got '%s'", components.CardTypeWallnutBowling, beltComp.Cards[1].CardType)
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

func TestConveyorBeltSystem_SelectAndDeselect(t *testing.T) {
	system, em := createTestConveyorBeltSystem()

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())

	// 添加卡片
	system.addCard(beltComp, components.CardTypeWallnutBowling)
	beltComp.Cards[0].IsStopped = true // 卡片已停止，可被选中

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

	// 选中正在移动的卡片也应成功（支持传送过程中选择卡片）
	system.addCard(beltComp, components.CardTypeExplodeONut)
	beltComp.Cards[1].IsStopped = false // 卡片仍在移动

	if !system.SelectCard(1) {
		t.Error("Expected SelectCard to return true for moving card (conveyor belt cards should be selectable during transit)")
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
	beltComp.GenerationTimer = 0 // 立即生成

	// 更新多次，应该生成卡片
	for i := 0; i < 5; i++ {
		beltComp.GenerationTimer = 0
		system.Update(0.1)
	}

	if len(beltComp.Cards) == 0 {
		t.Error("Expected cards to be generated after Update")
	}
}
