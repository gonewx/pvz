package systems

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestPlantSelectionSystem_SelectPlant 测试选择植物功能
func TestPlantSelectionSystem_SelectPlant(t *testing.T) {
	// 创建测试环境
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil) // 音频上下文为 nil（测试中不需要）
	levelConfig := &config.LevelConfig{}

	system := NewPlantSelectionSystem(em, rm, gs, levelConfig)

	// 创建选卡组件实体
	selectionEntity := em.CreateEntity()
	ecs.AddComponent(em, selectionEntity, &components.PlantSelectionComponent{
		SelectedPlants: []string{},
		MaxSlots:       6,
		IsConfirmed:    false,
	})

	// 测试选择第一株植物
	err := system.SelectPlant("peashooter")
	if err != nil {
		t.Fatalf("Expected no error selecting peashooter, got: %v", err)
	}

	// 验证植物已添加
	selected := system.GetSelectedPlants()
	if len(selected) != 1 {
		t.Fatalf("Expected 1 selected plant, got %d", len(selected))
	}
	if selected[0] != "peashooter" {
		t.Errorf("Expected peashooter, got %s", selected[0])
	}

	// 测试选择第二株植物
	err = system.SelectPlant("sunflower")
	if err != nil {
		t.Fatalf("Expected no error selecting sunflower, got: %v", err)
	}

	selected = system.GetSelectedPlants()
	if len(selected) != 2 {
		t.Fatalf("Expected 2 selected plants, got %d", len(selected))
	}
}

// TestPlantSelectionSystem_SelectPlant_AlreadySelected 测试选择已选中的植物
func TestPlantSelectionSystem_SelectPlant_AlreadySelected(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)
	levelConfig := &config.LevelConfig{}

	system := NewPlantSelectionSystem(em, rm, gs, levelConfig)

	selectionEntity := em.CreateEntity()
	ecs.AddComponent(em, selectionEntity, &components.PlantSelectionComponent{
		SelectedPlants: []string{"peashooter"},
		MaxSlots:       6,
		IsConfirmed:    false,
	})

	// 尝试再次选择相同植物
	err := system.SelectPlant("peashooter")
	if err == nil {
		t.Error("Expected error when selecting already selected plant, got nil")
	}
}

// TestPlantSelectionSystem_SelectPlant_MaxSlots 测试最大槽位限制
func TestPlantSelectionSystem_SelectPlant_MaxSlots(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)
	levelConfig := &config.LevelConfig{}

	system := NewPlantSelectionSystem(em, rm, gs, levelConfig)

	selectionEntity := em.CreateEntity()
	ecs.AddComponent(em, selectionEntity, &components.PlantSelectionComponent{
		SelectedPlants: []string{"plant1", "plant2", "plant3"},
		MaxSlots:       3,
		IsConfirmed:    false,
	})

	// 尝试选择第4株植物（超过最大槽位）
	err := system.SelectPlant("plant4")
	if err == nil {
		t.Error("Expected error when exceeding max slots, got nil")
	}
}

// TestPlantSelectionSystem_DeselectPlant 测试取消选择功能
func TestPlantSelectionSystem_DeselectPlant(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)
	levelConfig := &config.LevelConfig{}

	system := NewPlantSelectionSystem(em, rm, gs, levelConfig)

	selectionEntity := em.CreateEntity()
	ecs.AddComponent(em, selectionEntity, &components.PlantSelectionComponent{
		SelectedPlants: []string{"peashooter", "sunflower", "wallnut"},
		MaxSlots:       6,
		IsConfirmed:    false,
	})

	// 取消选择中间的植物
	err := system.DeselectPlant("sunflower")
	if err != nil {
		t.Fatalf("Expected no error deselecting sunflower, got: %v", err)
	}

	// 验证植物已移除
	selected := system.GetSelectedPlants()
	if len(selected) != 2 {
		t.Fatalf("Expected 2 selected plants after deselect, got %d", len(selected))
	}

	// 验证剩余植物顺序
	if selected[0] != "peashooter" || selected[1] != "wallnut" {
		t.Errorf("Expected [peashooter, wallnut], got %v", selected)
	}
}

// TestPlantSelectionSystem_DeselectPlant_NotSelected 测试取消选择未选中的植物
func TestPlantSelectionSystem_DeselectPlant_NotSelected(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)
	levelConfig := &config.LevelConfig{}

	system := NewPlantSelectionSystem(em, rm, gs, levelConfig)

	selectionEntity := em.CreateEntity()
	ecs.AddComponent(em, selectionEntity, &components.PlantSelectionComponent{
		SelectedPlants: []string{"peashooter"},
		MaxSlots:       6,
		IsConfirmed:    false,
	})

	// 尝试取消选择未选中的植物
	err := system.DeselectPlant("sunflower")
	if err == nil {
		t.Error("Expected error when deselecting non-selected plant, got nil")
	}
}

// TestPlantSelectionSystem_ConfirmSelection 测试确认选择
func TestPlantSelectionSystem_ConfirmSelection(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)
	levelConfig := &config.LevelConfig{}

	system := NewPlantSelectionSystem(em, rm, gs, levelConfig)

	selectionEntity := em.CreateEntity()
	selectionComp := &components.PlantSelectionComponent{
		SelectedPlants: []string{"peashooter", "sunflower"},
		MaxSlots:       6,
		IsConfirmed:    false,
	}
	ecs.AddComponent(em, selectionEntity, selectionComp)

	// 确认选择
	err := system.ConfirmSelection()
	if err != nil {
		t.Fatalf("Expected no error confirming selection, got: %v", err)
	}

	// 验证确认标志
	if !selectionComp.IsConfirmed {
		t.Error("Expected IsConfirmed to be true after confirmation")
	}

	// 注意：GameState 不直接存储 SelectedPlants 字段
	// 选中的植物信息保存在 PlantSelectionComponent 中
	// 这里我们验证组件本身的状态即可
}

// TestPlantSelectionSystem_ConfirmSelection_NoPlants 测试确认选择（无植物）
func TestPlantSelectionSystem_ConfirmSelection_NoPlants(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)
	levelConfig := &config.LevelConfig{}

	system := NewPlantSelectionSystem(em, rm, gs, levelConfig)

	selectionEntity := em.CreateEntity()
	ecs.AddComponent(em, selectionEntity, &components.PlantSelectionComponent{
		SelectedPlants: []string{}, // 未选择任何植物
		MaxSlots:       6,
		IsConfirmed:    false,
	})

	// 尝试确认选择（应失败）
	err := system.ConfirmSelection()
	if err == nil {
		t.Error("Expected error when confirming with no plants selected, got nil")
	}
}

// TestPlantSelectionSystem_GetSelectedPlants 测试获取选中植物列表
func TestPlantSelectionSystem_GetSelectedPlants(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)
	levelConfig := &config.LevelConfig{}

	system := NewPlantSelectionSystem(em, rm, gs, levelConfig)

	selectionEntity := em.CreateEntity()
	ecs.AddComponent(em, selectionEntity, &components.PlantSelectionComponent{
		SelectedPlants: []string{"peashooter", "sunflower", "wallnut"},
		MaxSlots:       6,
		IsConfirmed:    false,
	})

	// 获取选中植物
	selected := system.GetSelectedPlants()

	// 验证数量
	if len(selected) != 3 {
		t.Fatalf("Expected 3 selected plants, got %d", len(selected))
	}

	// 验证内容
	expectedPlants := []string{"peashooter", "sunflower", "wallnut"}
	for i, plant := range expectedPlants {
		if selected[i] != plant {
			t.Errorf("Expected selected[%d] = %s, got %s", i, plant, selected[i])
		}
	}

	// 验证返回的是副本（修改不影响原数据）
	selected[0] = "modified"
	newSelected := system.GetSelectedPlants()
	if newSelected[0] == "modified" {
		t.Error("GetSelectedPlants should return a copy, not original slice")
	}
}
