package systems

import (
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
)

// createTestConveyorBeltSystem 创建测试用传送带系统
func createTestConveyorBeltSystem() (*ConveyorBeltSystem, *ecs.EntityManager) {
	em := ecs.NewEntityManager()
	system := NewConveyorBeltSystem(em, nil, nil)
	return system, em
}

// createTestConveyorBeltSystemWithGameState 创建测试用传送带系统（带 GameState）
func createTestConveyorBeltSystemWithGameState() (*ConveyorBeltSystem, *ecs.EntityManager, *game.GameState) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	system := NewConveyorBeltSystem(em, gs, nil)
	return system, em, gs
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

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())

	// 生成大量卡片统计分布
	wallnutCount := 0
	explodeCount := 0
	total := 10000

	for i := 0; i < total; i++ {
		cardType := system.generateCard(beltComp)
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
	system, em := createTestConveyorBeltSystem()

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())

	// 设置自定义卡片池（100% 爆炸坚果）
	customPool := []CardPoolEntry{
		{Type: components.CardTypeExplodeONut, Weight: 100},
	}
	system.SetCardPool(customPool)

	// 验证生成的都是爆炸坚果
	for i := 0; i < 100; i++ {
		cardType := system.generateCard(beltComp)
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

	// Story 19.12: 使用足够大的时间增量来触发卡片生成
	// 默认生成间隔为 3.0 秒，需要累积足够时间
	for i := 0; i < 35; i++ { // 3.5 秒
		system.Update(0.1)
	}

	if len(beltComp.Cards) == 0 {
		t.Error("Expected cards to be generated after Update")
	}
}

// ========================================
// Story 19.12: 动态调节系统测试
// ========================================

func TestConveyorBeltSystem_PhaseDetection(t *testing.T) {
	system, em, gs := createTestConveyorBeltSystemWithGameState()

	// 加载测试关卡配置（10 波）
	testLevelConfig := &config.LevelConfig{
		ID:   "test-1",
		Name: "Test Level",
		Waves: []config.WaveConfig{
			{WaveNum: 1}, {WaveNum: 2}, {WaveNum: 3}, {WaveNum: 4}, {WaveNum: 5},
			{WaveNum: 6}, {WaveNum: 7}, {WaveNum: 8}, {WaveNum: 9}, {WaveNum: 10},
		},
	}
	gs.LoadLevel(testLevelConfig)

	_ = em

	tests := []struct {
		currentWave   int
		expectedPhase int
	}{
		{0, 1},  // 0/10 = 0% → 前期
		{1, 1},  // 1/10 = 10% → 前期
		{2, 1},  // 2/10 = 20% → 前期
		{3, 2},  // 3/10 = 30% → 中期
		{5, 2},  // 5/10 = 50% → 中期
		{6, 2},  // 6/10 = 60% → 中期
		{7, 3},  // 7/10 = 70% → 终盘
		{9, 3},  // 9/10 = 90% → 终盘
		{10, 3}, // 10/10 = 100% → 终盘
	}

	for _, tt := range tests {
		gs.CurrentWaveIndex = tt.currentWave
		phase := system.getCurrentPhase()
		if phase != tt.expectedPhase {
			t.Errorf("currentWave=%d: expected phase %d, got %d", tt.currentWave, tt.expectedPhase, phase)
		}
	}
}

func TestConveyorBeltSystem_PhaseDetection_FewWaves(t *testing.T) {
	system, em, gs := createTestConveyorBeltSystemWithGameState()

	// 加载测试关卡配置（2 波 - 简化波次）
	testLevelConfig := &config.LevelConfig{
		ID:   "test-2",
		Name: "Test Level",
		Waves: []config.WaveConfig{
			{WaveNum: 1}, {WaveNum: 2},
		},
	}
	gs.LoadLevel(testLevelConfig)

	_ = em

	// 简化波次支持：当波次过少时使用中期配置
	for waveIdx := 0; waveIdx <= 2; waveIdx++ {
		gs.CurrentWaveIndex = waveIdx
		phase := system.getCurrentPhase()
		if phase != 2 {
			t.Errorf("currentWave=%d (few waves): expected phase 2 (中期), got %d", waveIdx, phase)
		}
	}
}

func TestConveyorBeltSystem_DynamicWeight(t *testing.T) {
	system, em, gs := createTestConveyorBeltSystemWithGameState()

	// 加载测试关卡配置
	testLevelConfig := &config.LevelConfig{
		ID:   "test-3",
		Name: "Test Level",
		Waves: []config.WaveConfig{
			{WaveNum: 1}, {WaveNum: 2}, {WaveNum: 3}, {WaveNum: 4}, {WaveNum: 5},
			{WaveNum: 6}, {WaveNum: 7}, {WaveNum: 8}, {WaveNum: 9}, {WaveNum: 10},
		},
	}
	gs.LoadLevel(testLevelConfig)

	// 设置动态配置
	phaseConfigs := []config.PhaseConfig{
		{ProgressThreshold: 0.0, ExplodeNutWeight: 10, IntervalMin: 3.0, IntervalMax: 3.5},
		{ProgressThreshold: 0.3, ExplodeNutWeight: 20, IntervalMin: 2.2, IntervalMax: 2.5},
		{ProgressThreshold: 0.7, ExplodeNutWeight: 30, IntervalMin: 1.5, IntervalMax: 1.8},
	}
	system.SetDynamicConfig(phaseConfigs, nil)

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())

	// 测试前期（10% 爆炸坚果）
	gs.CurrentWaveIndex = 0
	explodeCount := 0
	total := 1000
	for i := 0; i < total; i++ {
		if system.generateCard(beltComp) == components.CardTypeExplodeONut {
			explodeCount++
		}
	}
	ratio := float64(explodeCount) / float64(total)
	if ratio < 0.05 || ratio > 0.15 {
		t.Errorf("前期: expected explode ratio ~0.10, got %.3f", ratio)
	}

	// 测试终盘（30% 爆炸坚果）
	gs.CurrentWaveIndex = 8 // 80% 进度
	explodeCount = 0
	for i := 0; i < total; i++ {
		if system.generateCard(beltComp) == components.CardTypeExplodeONut {
			explodeCount++
		}
	}
	ratio = float64(explodeCount) / float64(total)
	if ratio < 0.25 || ratio > 0.40 {
		t.Errorf("终盘: expected explode ratio ~0.30, got %.3f", ratio)
	}
}

func TestConveyorBeltSystem_EmptyBeltEmergency(t *testing.T) {
	system, em, _ := createTestConveyorBeltSystemWithGameState()

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())

	// 设置动态调节配置
	dynamicCfg := &config.DynamicAdjustmentConfig{
		EmptyBeltThreshold: 3.0,
	}
	system.SetDynamicConfig(nil, dynamicCfg)

	// 确保传送带为空
	beltComp.Cards = nil
	if !beltComp.IsEmpty() {
		t.Fatal("Expected belt to be empty")
	}

	// 模拟 2.5 秒空带（不应触发）
	for i := 0; i < 250; i++ {
		system.checkEmptyBeltEmergency(0.01, beltComp)
	}
	if len(beltComp.Cards) != 0 {
		t.Errorf("Expected no cards after 2.5s, got %d", len(beltComp.Cards))
	}

	// 再模拟 0.6 秒（超过 3 秒阈值）
	for i := 0; i < 60; i++ {
		system.checkEmptyBeltEmergency(0.01, beltComp)
	}

	// 应该自动生成一个普通坚果
	if len(beltComp.Cards) != 1 {
		t.Errorf("Expected 1 card after 3s+ empty, got %d", len(beltComp.Cards))
	}
	if len(beltComp.Cards) > 0 && beltComp.Cards[0].CardType != components.CardTypeWallnutBowling {
		t.Errorf("Expected wallnut_bowling, got %s", beltComp.Cards[0].CardType)
	}
}

func TestConveyorBeltSystem_FullBeltThrottle(t *testing.T) {
	system, em, _ := createTestConveyorBeltSystemWithGameState()

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())
	beltComp.Capacity = 5

	// 设置动态调节配置
	dynamicCfg := &config.DynamicAdjustmentConfig{
		FullBeltThreshold:          8.0,
		FullBeltThrottleMultiplier: 1.5,
	}
	system.SetDynamicConfig(nil, dynamicCfg)

	// 填满传送带
	for i := 0; i < 5; i++ {
		beltComp.Cards = append(beltComp.Cards, components.ConveyorCard{
			CardType:  components.CardTypeWallnutBowling,
			PositionX: float64(i * 100),
			IsStopped: true,
		})
	}

	if !beltComp.IsFull() {
		t.Fatal("Expected belt to be full")
	}

	// 初始状态不应降频
	if beltComp.IsThrottled {
		t.Error("Expected not throttled initially")
	}

	// 模拟 7 秒满带（不应触发）
	for i := 0; i < 700; i++ {
		system.checkFullBeltThrottle(0.01, beltComp)
	}
	if beltComp.IsThrottled {
		t.Error("Expected not throttled after 7s")
	}

	// 再模拟 1.1 秒（超过 8 秒阈值）
	for i := 0; i < 110; i++ {
		system.checkFullBeltThrottle(0.01, beltComp)
	}

	// 应该进入降频状态
	if !beltComp.IsThrottled {
		t.Error("Expected throttled after 8s+")
	}

	// 移除 3 张卡片（低于满容量 2 格）
	beltComp.Cards = beltComp.Cards[:2]
	beltComp.FullDuration = 0 // 重置计时

	system.checkFullBeltThrottle(0.01, beltComp)

	// 应该解除降频
	if beltComp.IsThrottled {
		t.Error("Expected throttle released after cards removed")
	}
}

func TestConveyorBeltSystem_CrisisExplodeNut(t *testing.T) {
	system, em, gs := createTestConveyorBeltSystemWithGameState()

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())

	// 设置动态调节配置
	dynamicCfg := &config.DynamicAdjustmentConfig{
		CrisisExplodeNutCooldown: 5.0,
		CrisisZombieCount:        2,
		CrisisDistanceThreshold:  300.0,
	}
	system.SetDynamicConfig(nil, dynamicCfg)

	// 设置关卡时间超过冷却时间
	gs.LevelTime = 10.0
	beltComp.LastExplodeNutTime = 0.0

	// 创建僵尸实体（在同一行，接近安全线）
	row := 2
	zombieY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2

	// 创建第一个僵尸
	zombie1 := em.CreateEntity()
	em.AddComponent(zombie1, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombie1, &components.PositionComponent{
		X: config.GridWorldStartX + 200, // 在危机距离内
		Y: zombieY,
	})

	// 创建第二个僵尸（同一行）
	zombie2 := em.CreateEntity()
	em.AddComponent(zombie2, &components.BehaviorComponent{Type: components.BehaviorZombieConehead})
	em.AddComponent(zombie2, &components.PositionComponent{
		X: config.GridWorldStartX + 150, // 在危机距离内
		Y: zombieY,
	})

	// 初始状态不应强制生成
	if beltComp.ForceExplodeNut {
		t.Error("Expected ForceExplodeNut to be false initially")
	}

	// 检测危机
	system.checkCrisisExplodeNut(beltComp)

	// 应该标记强制生成爆炸坚果
	if !beltComp.ForceExplodeNut {
		t.Error("Expected ForceExplodeNut to be true after crisis detected")
	}
}

func TestConveyorBeltSystem_ForceExplodeNutGeneration(t *testing.T) {
	system, em, gs := createTestConveyorBeltSystemWithGameState()

	beltComp, _ := ecs.GetComponent[*components.ConveyorBeltComponent](em, system.GetBeltEntity())

	// 设置 ForceExplodeNut 标志
	beltComp.ForceExplodeNut = true
	beltComp.LastExplodeNutTime = 0.0
	gs.LevelTime = 10.0

	// 生成卡片
	cardType := system.generateCard(beltComp)

	// 应该强制生成爆炸坚果
	if cardType != components.CardTypeExplodeONut {
		t.Errorf("Expected explode_o_nut when ForceExplodeNut=true, got %s", cardType)
	}

	// 标志应该被重置
	if beltComp.ForceExplodeNut {
		t.Error("Expected ForceExplodeNut to be reset after generation")
	}

	// LastExplodeNutTime 应该被更新
	if beltComp.LastExplodeNutTime != gs.LevelTime {
		t.Errorf("Expected LastExplodeNutTime=%f, got %f", gs.LevelTime, beltComp.LastExplodeNutTime)
	}
}

func TestConveyorBeltSystem_DynamicInterval(t *testing.T) {
	system, em, gs := createTestConveyorBeltSystemWithGameState()

	// 加载测试关卡配置
	testLevelConfig := &config.LevelConfig{
		ID:   "test-4",
		Name: "Test Level",
		Waves: []config.WaveConfig{
			{WaveNum: 1}, {WaveNum: 2}, {WaveNum: 3}, {WaveNum: 4}, {WaveNum: 5},
			{WaveNum: 6}, {WaveNum: 7}, {WaveNum: 8}, {WaveNum: 9}, {WaveNum: 10},
		},
	}
	gs.LoadLevel(testLevelConfig)

	_ = em

	// 设置动态配置
	phaseConfigs := []config.PhaseConfig{
		{ProgressThreshold: 0.0, ExplodeNutWeight: 10, IntervalMin: 3.0, IntervalMax: 3.5},
		{ProgressThreshold: 0.3, ExplodeNutWeight: 20, IntervalMin: 2.2, IntervalMax: 2.5},
		{ProgressThreshold: 0.7, ExplodeNutWeight: 30, IntervalMin: 1.5, IntervalMax: 1.8},
	}
	system.SetDynamicConfig(phaseConfigs, nil)

	// 测试前期间隔
	gs.CurrentWaveIndex = 0
	for i := 0; i < 100; i++ {
		interval := system.getPhaseGenerationInterval()
		if interval < 3.0 || interval > 3.5 {
			t.Errorf("前期: expected interval in [3.0, 3.5], got %.3f", interval)
			break
		}
	}

	// 测试终盘间隔
	gs.CurrentWaveIndex = 8
	for i := 0; i < 100; i++ {
		interval := system.getPhaseGenerationInterval()
		if interval < 1.5 || interval > 1.8 {
			t.Errorf("终盘: expected interval in [1.5, 1.8], got %.3f", interval)
			break
		}
	}
}
