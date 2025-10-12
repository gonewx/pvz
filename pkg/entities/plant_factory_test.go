package entities

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

// 包级别的共享 audio context（避免重复创建）
var testAudioContext = audio.NewContext(48000)

// TestNewPlantEntity 测试植物实体创建
func TestNewPlantEntity(t *testing.T) {
	// 初始化资源管理器和实体管理器
	rm := game.NewResourceManager(testAudioContext)
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建植物实体
			plantID, err := NewPlantEntity(em, rm, gs, tt.plantType, tt.col, tt.row)
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
				expectedX, expectedY := utils.GridToScreenCoords(
					tt.col, tt.row,
					gs.CameraX,
					config.GridWorldStartX, config.GridWorldStartY,
					config.CellWidth, config.CellHeight,
				)
				if pos.X != expectedX || pos.Y != expectedY {
					t.Errorf("Position mismatch: got (%.1f, %.1f), want (%.1f, %.1f)",
						pos.X, pos.Y, expectedX, expectedY)
				}
			}

			// 验证 SpriteComponent
			spriteComp, ok := em.GetComponent(plantID, reflect.TypeOf(&components.SpriteComponent{}))
			if !ok {
				t.Error("Plant entity should have SpriteComponent")
			} else {
				sprite := spriteComp.(*components.SpriteComponent)
				if sprite.Image == nil {
					t.Error("SpriteComponent.Image should not be nil")
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
	rm := game.NewResourceManager(testAudioContext)
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.CameraX = 215 // 设置为游戏默认摄像机位置

	plantTypes := []struct {
		name      string
		plantType components.PlantType
	}{
		{"Sunflower", components.PlantSunflower},
		{"Peashooter", components.PlantPeashooter},
	}

	for _, pt := range plantTypes {
		t.Run(pt.name, func(t *testing.T) {
			plantID, err := NewPlantEntity(em, rm, gs, pt.plantType, 4, 2)
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
	rm := game.NewResourceManager(testAudioContext)
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.CameraX = 215 // 设置为游戏默认摄像机位置

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
			plantID, err := NewPlantEntity(em, rm, gs, components.PlantSunflower, corner.col, corner.row)
			if err != nil {
				t.Fatalf("Failed to create plant at %s: %v", corner.name, err)
			}

			posComp, ok := em.GetComponent(plantID, reflect.TypeOf(&components.PositionComponent{}))
			if !ok {
				t.Fatal("Plant should have PositionComponent")
			}

			pos := posComp.(*components.PositionComponent)
			expectedX, expectedY := utils.GridToScreenCoords(
				corner.col, corner.row,
				gs.CameraX,
				config.GridWorldStartX, config.GridWorldStartY,
				config.CellWidth, config.CellHeight,
			)

			if pos.X != expectedX || pos.Y != expectedY {
				t.Errorf("%s position incorrect: got (%.1f, %.1f), want (%.1f, %.1f)",
					corner.name, pos.X, pos.Y, expectedX, expectedY)
			}
		})
	}
}

// TestPlantHasHealthComponent 测试植物实体包含生命值组件
func TestPlantHasHealthComponent(t *testing.T) {
	rm := game.NewResourceManager(testAudioContext)
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.CameraX = 215

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
			plantID, err := NewPlantEntity(em, rm, gs, tt.plantType, 4, 2)
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
