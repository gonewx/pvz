package systems

import (
	"image"
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
)

// mockShovelStateProvider 模拟铲子状态提供者
type mockShovelStateProvider struct {
	selected bool
	bounds   image.Rectangle
}

func (m *mockShovelStateProvider) IsShovelSelected() bool {
	return m.selected
}

func (m *mockShovelStateProvider) SetShovelSelected(selected bool) {
	m.selected = selected
}

func (m *mockShovelStateProvider) GetShovelSlotBounds() image.Rectangle {
	return m.bounds
}

// createTestShovelInteractionSystem 创建测试用的铲子交互系统
func createTestShovelInteractionSystem() (*ShovelInteractionSystem, *ecs.EntityManager, *game.GameState) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()

	// 创建系统（不加载资源，避免依赖外部文件）
	system := &ShovelInteractionSystem{
		entityManager: em,
		gameState:     gs,
		cursorAnchorX: 20.0,
		cursorAnchorY: 5.0,
	}

	// 创建铲子交互实体
	system.shovelEntity = em.CreateEntity()
	shovelComp := &components.ShovelInteractionComponent{
		IsSelected:             false,
		HighlightedPlantEntity: 0,
		CursorAnchorX:          system.cursorAnchorX,
		CursorAnchorY:          system.cursorAnchorY,
	}
	em.AddComponent(system.shovelEntity, shovelComp)

	return system, em, gs
}

// TestNewShovelInteractionSystem_EntityCreation 测试系统创建实体
func TestNewShovelInteractionSystem_EntityCreation(t *testing.T) {
	system, em, _ := createTestShovelInteractionSystem()

	// 验证实体创建
	if system.shovelEntity == 0 {
		t.Error("Expected shovelEntity to be non-zero")
	}

	// 验证组件添加
	shovelComp, ok := ecs.GetComponent[*components.ShovelInteractionComponent](em, system.shovelEntity)
	if !ok {
		t.Fatal("Expected ShovelInteractionComponent to exist")
	}

	if shovelComp.IsSelected {
		t.Error("Expected IsSelected to be false initially")
	}

	if shovelComp.HighlightedPlantEntity != 0 {
		t.Errorf("Expected HighlightedPlantEntity to be 0, got %d", shovelComp.HighlightedPlantEntity)
	}
}

// TestShovelInteractionSystem_IsShovelMode 测试铲子模式检查
func TestShovelInteractionSystem_IsShovelMode(t *testing.T) {
	system, em, _ := createTestShovelInteractionSystem()

	// 初始状态：非铲子模式
	if system.IsShovelMode() {
		t.Error("Expected IsShovelMode to be false initially")
	}

	// 设置铲子选中状态
	shovelComp, _ := ecs.GetComponent[*components.ShovelInteractionComponent](em, system.shovelEntity)
	shovelComp.IsSelected = true

	// 验证铲子模式
	if !system.IsShovelMode() {
		t.Error("Expected IsShovelMode to be true after setting IsSelected")
	}
}

// TestShovelInteractionSystem_DetectPlantUnderMouse 测试植物检测
func TestShovelInteractionSystem_DetectPlantUnderMouse(t *testing.T) {
	system, em, _ := createTestShovelInteractionSystem()

	// 创建测试植物
	plantEntity := em.CreateEntity()
	em.AddComponent(plantEntity, &components.PlantComponent{
		PlantType: components.PlantSunflower,
		GridRow:   2,
		GridCol:   3,
	})
	em.AddComponent(plantEntity, &components.PositionComponent{
		X: 400.0,
		Y: 300.0,
	})

	// 测试：鼠标在植物位置
	detected := system.detectPlantUnderMouse(400.0, 300.0)
	if detected != plantEntity {
		t.Errorf("Expected detected plant to be %d, got %d", plantEntity, detected)
	}

	// 测试：鼠标在植物边界内
	detected = system.detectPlantUnderMouse(420.0, 320.0)
	if detected != plantEntity {
		t.Errorf("Expected detected plant at edge to be %d, got %d", plantEntity, detected)
	}

	// 测试：鼠标在植物外部
	detected = system.detectPlantUnderMouse(500.0, 300.0)
	if detected != 0 {
		t.Errorf("Expected no plant detected outside, got %d", detected)
	}
}

// TestShovelInteractionSystem_RemovePlant 测试植物移除
func TestShovelInteractionSystem_RemovePlant(t *testing.T) {
	system, em, _ := createTestShovelInteractionSystem()

	// 创建草坪网格
	gridEntity := em.CreateEntity()
	gridComp := &components.LawnGridComponent{}
	em.AddComponent(gridEntity, gridComp)

	// 创建测试植物
	plantEntity := em.CreateEntity()
	em.AddComponent(plantEntity, &components.PlantComponent{
		PlantType: components.PlantSunflower,
		GridRow:   2,
		GridCol:   3,
	})
	em.AddComponent(plantEntity, &components.PositionComponent{
		X: 400.0,
		Y: 300.0,
	})

	// 标记网格占用
	gridComp.Occupancy[2][3] = plantEntity

	// 移除植物
	system.removePlant(plantEntity)

	// 验证网格释放
	if gridComp.Occupancy[2][3] != 0 {
		t.Errorf("Expected grid cell to be empty (0), got %d", gridComp.Occupancy[2][3])
	}

	// 调用 RemoveMarkedEntities 清理
	em.RemoveMarkedEntities()

	// 验证植物实体被移除
	_, ok := ecs.GetComponent[*components.PlantComponent](em, plantEntity)
	if ok {
		t.Error("Expected plant entity to be removed")
	}
}

// TestShovelInteractionSystem_StateProviderIntegration 测试状态提供者集成
func TestShovelInteractionSystem_StateProviderIntegration(t *testing.T) {
	// 创建模拟状态提供者
	mockProvider := &mockShovelStateProvider{
		selected: false,
		bounds:   image.Rect(100, 0, 170, 74),
	}

	// 设置全局状态提供者
	SetShovelStateProvider(mockProvider)
	defer SetShovelStateProvider(nil) // 清理

	// 验证初始状态
	if mockProvider.IsShovelSelected() {
		t.Error("Expected IsShovelSelected to be false initially")
	}

	// 设置选中状态
	mockProvider.SetShovelSelected(true)
	if !mockProvider.IsShovelSelected() {
		t.Error("Expected IsShovelSelected to be true after setting")
	}

	// 验证边界
	bounds := mockProvider.GetShovelSlotBounds()
	if bounds.Min.X != 100 || bounds.Max.X != 170 {
		t.Errorf("Expected bounds X range [100, 170], got [%d, %d]", bounds.Min.X, bounds.Max.X)
	}
}

// TestShovelInteractionSystem_MultiplePlants 测试多个植物检测
func TestShovelInteractionSystem_MultiplePlants(t *testing.T) {
	system, em, _ := createTestShovelInteractionSystem()

	// 创建多个测试植物
	plant1 := em.CreateEntity()
	em.AddComponent(plant1, &components.PlantComponent{GridRow: 1, GridCol: 1})
	em.AddComponent(plant1, &components.PositionComponent{X: 200.0, Y: 200.0})

	plant2 := em.CreateEntity()
	em.AddComponent(plant2, &components.PlantComponent{GridRow: 2, GridCol: 2})
	em.AddComponent(plant2, &components.PositionComponent{X: 300.0, Y: 300.0})

	plant3 := em.CreateEntity()
	em.AddComponent(plant3, &components.PlantComponent{GridRow: 3, GridCol: 3})
	em.AddComponent(plant3, &components.PositionComponent{X: 400.0, Y: 400.0})

	// 测试每个植物的检测
	testCases := []struct {
		worldX   float64
		worldY   float64
		expected ecs.EntityID
	}{
		{200.0, 200.0, plant1},
		{300.0, 300.0, plant2},
		{400.0, 400.0, plant3},
		{600.0, 600.0, 0}, // 空位置
	}

	for _, tc := range testCases {
		detected := system.detectPlantUnderMouse(tc.worldX, tc.worldY)
		if detected != tc.expected {
			t.Errorf("At (%.1f, %.1f): expected %d, got %d", tc.worldX, tc.worldY, tc.expected, detected)
		}
	}
}

// TestShovelInteractionSystem_NoSunReturn 测试移除植物不返还阳光
func TestShovelInteractionSystem_NoSunReturn(t *testing.T) {
	system, em, gs := createTestShovelInteractionSystem()

	// 记录初始阳光值
	initialSun := gs.GetSun()

	// 创建测试植物（假设向日葵花费 50 阳光）
	plantEntity := em.CreateEntity()
	em.AddComponent(plantEntity, &components.PlantComponent{
		PlantType: components.PlantSunflower,
		GridRow:   0,
		GridCol:   0,
	})
	em.AddComponent(plantEntity, &components.PositionComponent{X: 100.0, Y: 100.0})

	// 移除植物
	system.removePlant(plantEntity)
	em.RemoveMarkedEntities()

	// 验证阳光值未增加
	finalSun := gs.GetSun()
	if finalSun != initialSun {
		t.Errorf("Expected sun to remain %d after removal, got %d", initialSun, finalSun)
	}
}

// TestShovelInteractionSystem_GetShovelEntity 测试获取铲子实体ID
func TestShovelInteractionSystem_GetShovelEntity(t *testing.T) {
	system, _, _ := createTestShovelInteractionSystem()

	entityID := system.GetShovelEntity()
	if entityID == 0 {
		t.Error("Expected GetShovelEntity to return non-zero ID")
	}

	if entityID != system.shovelEntity {
		t.Errorf("Expected GetShovelEntity to return %d, got %d", system.shovelEntity, entityID)
	}
}
