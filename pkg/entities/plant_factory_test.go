package entities

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestNewPlantEntity 测试植物实体创建
func TestNewPlantEntity(t *testing.T) {
	// 初始化资源管理器和实体管理器
	rm := newMockResourceManager()
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.CameraX = 215 // 设置为游戏默认摄像机位置

	tests := []struct {
		name      string
		plantType components.PlantType
		col       int
		row       int
	}{
		{
			name:      "创建向日葵 (0,0)",
			plantType: components.PlantSunflower,
			col:       0,
			row:       0,
		},
		{
			name:      "创建豌豆射手 (4,2)",
			plantType: components.PlantPeashooter,
			col:       4,
			row:       2,
		},
		{
			name:      "创建向日葵 (8,4)",
			plantType: components.PlantSunflower,
			col:       8,
			row:       4,
		},
	}

	// 创建 mock ReanimSystem
	mockRS := &mockReanimSystem{em: em}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建植物实体
			plantID, err := NewPlantEntity(em, rm, gs, mockRS, tt.plantType, tt.col, tt.row)
			if err != nil {
				t.Fatalf("Failed to create plant entity: %v", err)
			}

			if plantID == 0 {
				t.Fatal("Expected valid entity ID, got 0")
			}

			// 验证 PositionComponent
			posComp, ok := em.GetComponent(plantID, reflect.TypeOf(&components.PositionComponent{}))
			if !ok {
				t.Error("Plant entity should have PositionComponent")
			} else {
				pos := posComp.(*components.PositionComponent)
				// Story 6.3: 实体位置使用世界坐标（不受摄像机影响）
				expectedX := config.GridWorldStartX + float64(tt.col)*config.CellWidth + config.CellWidth/2
				expectedY := config.GridWorldStartY + float64(tt.row)*config.CellHeight + config.CellHeight/2
				if pos.X != expectedX || pos.Y != expectedY {
					t.Errorf("Position mismatch: got (%.1f, %.1f), want (%.1f, %.1f)",
						pos.X, pos.Y, expectedX, expectedY)
				}
			}

			// 验证 ReanimComponent
			reanimComp, ok := em.GetComponent(plantID, reflect.TypeOf(&components.ReanimComponent{}))
			if !ok {
				t.Error("Plant entity should have ReanimComponent")
			} else {
				reanim := reanimComp.(*components.ReanimComponent)
				if reanim.Reanim == nil {
					t.Error("ReanimComponent.Reanim should not be nil")
				}
				if reanim.PartImages == nil {
					t.Error("ReanimComponent.PartImages should not be nil")
				}
			}

			// 验证 PlantComponent
			plantComp, ok := em.GetComponent(plantID, reflect.TypeOf(&components.PlantComponent{}))
			if !ok {
				t.Error("Plant entity should have PlantComponent")
			} else {
				plant := plantComp.(*components.PlantComponent)
				if plant.PlantType != tt.plantType {
					t.Errorf("PlantType mismatch: got %v, want %v", plant.PlantType, tt.plantType)
				}
				if plant.GridCol != tt.col {
					t.Errorf("GridCol mismatch: got %d, want %d", plant.GridCol, tt.col)
				}
				if plant.GridRow != tt.row {
					t.Errorf("GridRow mismatch: got %d, want %d", plant.GridRow, tt.row)
				}
			}
		})
	}
}

// TestNewPlantEntity_AllPlantTypes 测试所有植物类型
func TestNewPlantEntity_AllPlantTypes(t *testing.T) {
	rm := newMockResourceManager()
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.CameraX = 215 // 设置为游戏默认摄像机位置
	mockRS := &mockReanimSystem{em: em}

	plantTypes := []struct {
		name      string
		plantType components.PlantType
	}{
		{"Sunflower", components.PlantSunflower},
		{"Peashooter", components.PlantPeashooter},
	}

	for _, pt := range plantTypes {
		t.Run(pt.name, func(t *testing.T) {
			plantID, err := NewPlantEntity(em, rm, gs, mockRS, pt.plantType, 4, 2)
			if err != nil {
				t.Fatalf("Failed to create plant of type %s: %v", pt.name, err)
			}

			if plantID == 0 {
				t.Fatalf("Expected valid entity ID for plant type %s", pt.name)
			}

			// 验证植物组件的类型正确
			plantComp, ok := em.GetComponent(plantID, reflect.TypeOf(&components.PlantComponent{}))
			if !ok {
				t.Fatalf("Plant entity should have PlantComponent")
			}

			plant := plantComp.(*components.PlantComponent)
			if plant.PlantType != pt.plantType {
				t.Errorf("PlantType mismatch: got %v, want %v", plant.PlantType, pt.plantType)
			}
		})
	}
}

// TestNewPlantEntity_PositionCalculation 测试位置计算的准确性
func TestNewPlantEntity_PositionCalculation(t *testing.T) {
	rm := newMockResourceManager()
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.CameraX = 215 // 设置为游戏默认摄像机位置
	mockRS := &mockReanimSystem{em: em}

	// 测试网格的所有四个角
	corners := []struct {
		name string
		col  int
		row  int
	}{
		{"左上角", 0, 0},
		{"右上角", 8, 0},
		{"左下角", 0, 4},
		{"右下角", 8, 4},
	}

	for _, corner := range corners {
		t.Run(corner.name, func(t *testing.T) {
			plantID, err := NewPlantEntity(em, rm, gs, mockRS, components.PlantSunflower, corner.col, corner.row)
			if err != nil {
				t.Fatalf("Failed to create plant at %s: %v", corner.name, err)
			}

			posComp, ok := em.GetComponent(plantID, reflect.TypeOf(&components.PositionComponent{}))
			if !ok {
				t.Fatal("Plant should have PositionComponent")
			}

			pos := posComp.(*components.PositionComponent)
			// Story 6.3: 实体位置使用世界坐标（不受摄像机影响）
			expectedX := config.GridWorldStartX + float64(corner.col)*config.CellWidth + config.CellWidth/2
			expectedY := config.GridWorldStartY + float64(corner.row)*config.CellHeight + config.CellHeight/2

			if pos.X != expectedX || pos.Y != expectedY {
				t.Errorf("%s position incorrect: got (%.1f, %.1f), want (%.1f, %.1f)",
					corner.name, pos.X, pos.Y, expectedX, expectedY)
			}
		})
	}
}

// TestPlantHasHealthComponent 测试植物实体包含生命值组件
func TestPlantHasHealthComponent(t *testing.T) {
	rm := newMockResourceManager()
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.CameraX = 215
	mockRS := &mockReanimSystem{em: em}

	tests := []struct {
		name           string
		plantType      components.PlantType
		expectedHealth int
	}{
		{
			name:           "向日葵拥有生命值",
			plantType:      components.PlantSunflower,
			expectedHealth: config.SunflowerDefaultHealth,
		},
		{
			name:           "豌豆射手拥有生命值",
			plantType:      components.PlantPeashooter,
			expectedHealth: config.PeashooterDefaultHealth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建植物实体
			plantID, err := NewPlantEntity(em, rm, gs, mockRS, tt.plantType, 4, 2)
			if err != nil {
				t.Fatalf("Failed to create plant: %v", err)
			}

			// 验证 HealthComponent 存在
			healthComp, ok := em.GetComponent(plantID, reflect.TypeOf(&components.HealthComponent{}))
			if !ok {
				t.Fatal("Plant entity should have HealthComponent")
			}

			// 验证生命值正确初始化
			health := healthComp.(*components.HealthComponent)
			if health.CurrentHealth != tt.expectedHealth {
				t.Errorf("CurrentHealth mismatch: got %d, want %d",
					health.CurrentHealth, tt.expectedHealth)
			}
			if health.MaxHealth != tt.expectedHealth {
				t.Errorf("MaxHealth mismatch: got %d, want %d",
					health.MaxHealth, tt.expectedHealth)
			}
		})
	}
}

// TestNewWallnutEntity 测试坚果墙实体创建
func TestNewWallnutEntity(t *testing.T) {
	// 初始化资源管理器和实体管理器
	rm := newMockResourceManager()
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.CameraX = 215
	mockRS := &mockReanimSystem{em: em}

	tests := []struct {
		name string
		col  int
		row  int
	}{
		{
			name: "创建坚果墙 (0,0)",
			col:  0,
			row:  0,
		},
		{
			name: "创建坚果墙 (4,2)",
			col:  4,
			row:  2,
		},
		{
			name: "创建坚果墙 (8,4)",
			col:  8,
			row:  4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建坚果墙实体
			wallnutID, err := NewWallnutEntity(em, rm, gs, mockRS, tt.col, tt.row)
			if err != nil {
				t.Fatalf("Failed to create wallnut entity: %v", err)
			}

			if wallnutID == 0 {
				t.Fatal("Expected valid entity ID, got 0")
			}

			// 验证 PositionComponent
			posComp, ok := em.GetComponent(wallnutID, reflect.TypeOf(&components.PositionComponent{}))
			if !ok {
				t.Error("Wallnut entity should have PositionComponent")
			} else {
				pos := posComp.(*components.PositionComponent)
				expectedX := config.GridWorldStartX + float64(tt.col)*config.CellWidth + config.CellWidth/2
				expectedY := config.GridWorldStartY + float64(tt.row)*config.CellHeight + config.CellHeight/2
				if pos.X != expectedX || pos.Y != expectedY {
					t.Errorf("Position mismatch: got (%.1f, %.1f), want (%.1f, %.1f)",
						pos.X, pos.Y, expectedX, expectedY)
				}
			}

			// 验证 ReanimComponent
			reanimComp, ok := em.GetComponent(wallnutID, reflect.TypeOf(&components.ReanimComponent{}))
			if !ok {
				t.Error("Wallnut entity should have ReanimComponent")
			} else {
				reanim := reanimComp.(*components.ReanimComponent)
				if reanim.Reanim == nil {
					t.Error("ReanimComponent.Reanim should not be nil")
				}
				if reanim.PartImages == nil {
					t.Error("ReanimComponent.PartImages should not be nil")
				}
			}

			// 验证 PlantComponent（用于碰撞检测）
			plantComp, ok := em.GetComponent(wallnutID, reflect.TypeOf(&components.PlantComponent{}))
			if !ok {
				t.Fatal("Wallnut entity should have PlantComponent")
			} else {
				plant := plantComp.(*components.PlantComponent)
				if plant.PlantType != components.PlantWallnut {
					t.Errorf("PlantType mismatch: got %v, want %v",
						plant.PlantType, components.PlantWallnut)
				}
				if plant.GridCol != tt.col {
					t.Errorf("GridCol mismatch: got %d, want %d", plant.GridCol, tt.col)
				}
				if plant.GridRow != tt.row {
					t.Errorf("GridRow mismatch: got %d, want %d", plant.GridRow, tt.row)
				}
			}

			// 验证 HealthComponent（关键：坚果墙应有高生命值）
			healthComp, ok := em.GetComponent(wallnutID, reflect.TypeOf(&components.HealthComponent{}))
			if !ok {
				t.Fatal("Wallnut entity should have HealthComponent")
			} else {
				health := healthComp.(*components.HealthComponent)
				if health.CurrentHealth != config.WallnutDefaultHealth {
					t.Errorf("CurrentHealth mismatch: got %d, want %d",
						health.CurrentHealth, config.WallnutDefaultHealth)
				}
				if health.MaxHealth != config.WallnutDefaultHealth {
					t.Errorf("MaxHealth mismatch: got %d, want %d",
						health.MaxHealth, config.WallnutDefaultHealth)
				}
				// 验证坚果墙生命值远高于其他植物
				if health.MaxHealth <= config.SunflowerDefaultHealth || health.MaxHealth <= config.PeashooterDefaultHealth {
					t.Errorf("Wallnut health (%d) should be much higher than other plants (Sunflower: %d, Peashooter: %d)",
						health.MaxHealth, config.SunflowerDefaultHealth, config.PeashooterDefaultHealth)
				}
			}

			// 验证 BehaviorComponent（行为类型应为 BehaviorWallnut）
			behaviorComp, ok := em.GetComponent(wallnutID, reflect.TypeOf(&components.BehaviorComponent{}))
			if !ok {
				t.Fatal("Wallnut entity should have BehaviorComponent")
			} else {
				behavior := behaviorComp.(*components.BehaviorComponent)
				if behavior.Type != components.BehaviorWallnut {
					t.Errorf("BehaviorType mismatch: got %v, want %v",
						behavior.Type, components.BehaviorWallnut)
				}
			}

			// 验证 CollisionComponent
			collisionComp, ok := em.GetComponent(wallnutID, reflect.TypeOf(&components.CollisionComponent{}))
			if !ok {
				t.Fatal("Wallnut entity should have CollisionComponent")
			} else {
				collision := collisionComp.(*components.CollisionComponent)
				if collision.Width <= 0 || collision.Height <= 0 {
					t.Errorf("Collision box has invalid size: Width=%f, Height=%f",
						collision.Width, collision.Height)
				}
			}
		})
	}
}

// TestWallnutHealthConfiguration 测试坚果墙生命值配置
func TestWallnutHealthConfiguration(t *testing.T) {
	rm := newMockResourceManager()
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.CameraX = 215
	mockRS := &mockReanimSystem{em: em}

	// 创建坚果墙
	wallnutID, err := NewWallnutEntity(em, rm, gs, mockRS, 4, 2)
	if err != nil {
		t.Fatalf("Failed to create wallnut entity: %v", err)
	}

	// 获取生命值组件
	healthComp, ok := em.GetComponent(wallnutID, reflect.TypeOf(&components.HealthComponent{}))
	if !ok {
		t.Fatal("Wallnut should have HealthComponent")
	}

	health := healthComp.(*components.HealthComponent)

	// 验证坚果墙生命值为 4000
	if health.MaxHealth != 4000 {
		t.Errorf("Wallnut MaxHealth should be 4000, got %d", health.MaxHealth)
	}

	if health.CurrentHealth != 4000 {
		t.Errorf("Wallnut CurrentHealth should start at 4000, got %d", health.CurrentHealth)
	}

	// 验证坚果墙生命值是向日葵的 13 倍以上
	sunflowerHealth := config.SunflowerDefaultHealth
	ratio := float64(health.MaxHealth) / float64(sunflowerHealth)
	if ratio < 13.0 {
		t.Errorf("Wallnut health should be at least 13x Sunflower health, got %.1fx", ratio)
	}
}
