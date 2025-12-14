package entities

import (
	"reflect"
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
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
				// pos.Y = 格子中心 + PlantOffsetY（脚底位置）
				expectedX := config.GridWorldStartX + float64(tt.col)*config.CellWidth + config.CellWidth/2
				expectedY := config.GridWorldStartY + float64(tt.row)*config.CellHeight + config.CellHeight/2 + config.PlantOffsetY
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
				if reanim.ReanimXML == nil {
					t.Error("ReanimComponent.ReanimXML should not be nil")
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
			// pos.Y = 格子中心 + PlantOffsetY（脚底位置）
			expectedX := config.GridWorldStartX + float64(corner.col)*config.CellWidth + config.CellWidth/2
			expectedY := config.GridWorldStartY + float64(corner.row)*config.CellHeight + config.CellHeight/2 + config.PlantOffsetY

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
				// Y 坐标 = 格子中心 + PlantOffsetY，使植物脚底对齐到格子底部
				expectedY := config.GridWorldStartY + float64(tt.row)*config.CellHeight + config.CellHeight/2 + config.PlantOffsetY
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
				if reanim.ReanimXML == nil {
					t.Error("ReanimComponent.ReanimXML should not be nil")
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

// TestNewCherryBombEntity 测试樱桃炸弹实体创建
func TestNewCherryBombEntity(t *testing.T) {
	// 初始化资源管理器和实体管理器
	rm := newMockResourceManager()
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.CameraX = 215

	tests := []struct {
		name string
		col  int
		row  int
	}{
		{
			name: "创建樱桃炸弹 (0,0)",
			col:  0,
			row:  0,
		},
		{
			name: "创建樱桃炸弹 (4,2)",
			col:  4,
			row:  2,
		},
		{
			name: "创建樱桃炸弹 (8,4)",
			col:  8,
			row:  4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建樱桃炸弹实体（Epic 14: 已删除 mockRS 参数）
			cherryBombID, err := NewCherryBombEntity(em, rm, gs, tt.col, tt.row)
			if err != nil {
				t.Fatalf("Failed to create cherry bomb entity: %v", err)
			}

			if cherryBombID == 0 {
				t.Fatal("Expected valid entity ID, got 0")
			}

			// 验证 PositionComponent
			posComp, ok := em.GetComponent(cherryBombID, reflect.TypeOf(&components.PositionComponent{}))
			if !ok {
				t.Error("Cherry bomb entity should have PositionComponent")
			} else {
				pos := posComp.(*components.PositionComponent)
				expectedX := config.GridWorldStartX + float64(tt.col)*config.CellWidth + config.CellWidth/2
				// Y 坐标 = 格子中心 + PlantOffsetY，使植物脚底对齐到格子底部
				expectedY := config.GridWorldStartY + float64(tt.row)*config.CellHeight + config.CellHeight/2 + config.PlantOffsetY
				if pos.X != expectedX || pos.Y != expectedY {
					t.Errorf("Position mismatch: got (%.1f, %.1f), want (%.1f, %.1f)",
						pos.X, pos.Y, expectedX, expectedY)
				}
			}

			// 验证 ReanimComponent
			reanimComp, ok := em.GetComponent(cherryBombID, reflect.TypeOf(&components.ReanimComponent{}))
			if !ok {
				t.Error("Cherry bomb entity should have ReanimComponent")
			} else {
				reanim := reanimComp.(*components.ReanimComponent)
				if reanim.ReanimXML == nil {
					t.Error("ReanimComponent.ReanimXML should not be nil")
				}
				if reanim.PartImages == nil {
					t.Error("ReanimComponent.PartImages should not be nil")
				}
				// ✅ Epic 14: 不再检查 CurrentAnimations
				// 新架构中使用 AnimationCommandComponent，CurrentAnimations 在 ReanimSystem.Update() 处理后才填充
			}

			// ✅ Epic 14: 验证 AnimationCommandComponent（替代直接调用 ReanimSystem）
			animCmd, ok := ecs.GetComponent[*components.AnimationCommandComponent](em, cherryBombID)
			if !ok {
				t.Error("Cherry bomb entity should have AnimationCommandComponent")
			} else {
				if animCmd.AnimationName != "anim_idle" {
					t.Errorf("AnimationCommandComponent.AnimationName should be 'anim_idle', got '%s'", animCmd.AnimationName)
				}
				if animCmd.Processed {
					t.Error("AnimationCommand should not be processed yet (Processed=false)")
				}
			}

			// 验证 PlantComponent
			plantComp, ok := em.GetComponent(cherryBombID, reflect.TypeOf(&components.PlantComponent{}))
			if !ok {
				t.Fatal("Cherry bomb entity should have PlantComponent")
			} else {
				plant := plantComp.(*components.PlantComponent)
				if plant.PlantType != components.PlantCherryBomb {
					t.Errorf("PlantType mismatch: got %v, want %v",
						plant.PlantType, components.PlantCherryBomb)
				}
				if plant.GridCol != tt.col {
					t.Errorf("GridCol mismatch: got %d, want %d", plant.GridCol, tt.col)
				}
				if plant.GridRow != tt.row {
					t.Errorf("GridRow mismatch: got %d, want %d", plant.GridRow, tt.row)
				}
			}

			// 验证 BehaviorComponent
			behaviorComp, ok := em.GetComponent(cherryBombID, reflect.TypeOf(&components.BehaviorComponent{}))
			if !ok {
				t.Fatal("Cherry bomb entity should have BehaviorComponent")
			} else {
				behavior := behaviorComp.(*components.BehaviorComponent)
				if behavior.Type != components.BehaviorCherryBomb {
					t.Errorf("BehaviorType mismatch: got %v, want %v",
						behavior.Type, components.BehaviorCherryBomb)
				}
			}

			// 验证 TimerComponent（引信计时器）
			timerComp, ok := em.GetComponent(cherryBombID, reflect.TypeOf(&components.TimerComponent{}))
			if !ok {
				t.Fatal("Cherry bomb entity should have TimerComponent")
			} else {
				timer := timerComp.(*components.TimerComponent)
				if timer.Name != "fuse_timer" {
					t.Errorf("Timer name should be 'fuse_timer', got '%s'", timer.Name)
				}
				if timer.TargetTime != config.CherryBombFuseTime {
					t.Errorf("Timer TargetTime should be %.1f, got %.1f",
						config.CherryBombFuseTime, timer.TargetTime)
				}
				if timer.CurrentTime != 0 {
					t.Errorf("Timer CurrentTime should start at 0, got %.1f", timer.CurrentTime)
				}
				if timer.IsReady {
					t.Error("Timer IsReady should start as false")
				}
			}

			// 验证 CollisionComponent
			collisionComp, ok := em.GetComponent(cherryBombID, reflect.TypeOf(&components.CollisionComponent{}))
			if !ok {
				t.Fatal("Cherry bomb entity should have CollisionComponent")
			} else {
				collision := collisionComp.(*components.CollisionComponent)
				if collision.Width != config.CellWidth {
					t.Errorf("Collision Width should be %.1f, got %.1f",
						config.CellWidth, collision.Width)
				}
				if collision.Height != config.CellHeight {
					t.Errorf("Collision Height should be %.1f, got %.1f",
						config.CellHeight, collision.Height)
				}
			}
		})
	}
}

// TestCherryBombConfiguration 测试樱桃炸弹配置常量
func TestCherryBombConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		constant interface{}
		expected interface{}
	}{
		{
			name:     "阳光消耗应为150",
			constant: config.CherryBombSunCost,
			expected: 150,
		},
		{
			name:     "引信时间应为1.5秒",
			constant: config.CherryBombFuseTime,
			expected: 1.5,
		},
		{
			name:     "爆炸伤害应为1800",
			constant: config.CherryBombDamage,
			expected: 1800,
		},
		{
			name:     "爆炸圆心X偏移应为0像素",
			constant: config.CherryBombExplosionCenterOffsetX,
			expected: 0.0,
		},
		{
			name:     "爆炸圆心Y偏移应为0像素",
			constant: config.CherryBombExplosionCenterOffsetY,
			expected: 0.0,
		},
		{
			name:     "爆炸范围半径应为115像素",
			constant: config.CherryBombExplosionRadius,
			expected: 115.0,
		},
		{
			name:     "冷却时间应为50秒",
			constant: config.CherryBombCooldown,
			expected: 50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Configuration mismatch: got %v, want %v", tt.constant, tt.expected)
			}
		})
	}

	// 验证爆炸伤害足以秒杀所有MVP范围内的僵尸
	t.Run("爆炸伤害应足以秒杀铁桶僵尸", func(t *testing.T) {
		// 铁桶僵尸：270生命值 + 1100护甲 = 1370总生命值
		bucketheadTotalHealth := config.ZombieDefaultHealth + config.BucketheadZombieArmorHealth
		if config.CherryBombDamage < bucketheadTotalHealth {
			t.Errorf("Cherry bomb damage (%d) should be >= buckethead total health (%d)",
				config.CherryBombDamage, bucketheadTotalHealth)
		}
	})
}

// =============================================================================
// Story 18.5: 植物工厂注册表测试
// =============================================================================

// TestGetPlantFactory_KnownTypes 测试已知植物类型的工厂查找
func TestGetPlantFactory_KnownTypes(t *testing.T) {
	testCases := []struct {
		plantType string
		wantFound bool
	}{
		{"sunflower", true},
		{"peashooter", true},
		{"wallnut", true},
		{"cherrybomb", true},
		{"potatomine", true},
		{"snowpea", true},
	}

	for _, tc := range testCases {
		t.Run(tc.plantType, func(t *testing.T) {
			factory, found := GetPlantFactory(tc.plantType)
			if found != tc.wantFound {
				t.Errorf("GetPlantFactory(%q) found = %v, want %v", tc.plantType, found, tc.wantFound)
			}
			if tc.wantFound && factory == nil {
				t.Errorf("GetPlantFactory(%q) returned nil factory", tc.plantType)
			}
		})
	}
}

// TestGetPlantFactory_UnknownType 测试未知植物类型返回 nil
func TestGetPlantFactory_UnknownType(t *testing.T) {
	unknownTypes := []string{
		"unknown",
		"chomper",
		"repeater",
		"",
		"SUNFLOWER", // 区分大小写
	}

	for _, plantType := range unknownTypes {
		t.Run(plantType, func(t *testing.T) {
			factory, found := GetPlantFactory(plantType)
			if found {
				t.Errorf("GetPlantFactory(%q) should not find unknown type", plantType)
			}
			if factory != nil {
				t.Errorf("GetPlantFactory(%q) should return nil for unknown type", plantType)
			}
		})
	}
}

// TestGetDefaultPlantFactory 测试默认工厂返回非 nil
func TestGetDefaultPlantFactory(t *testing.T) {
	factory := GetDefaultPlantFactory()
	if factory == nil {
		t.Error("GetDefaultPlantFactory() returned nil")
	}
}

// TestPlantTagComponent_AddedByFactories 测试工厂函数是否添加 PlantTagComponent
func TestPlantTagComponent_AddedByFactories(t *testing.T) {
	rm := newMockResourceManager()
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.CameraX = 215
	mockRS := &mockReanimSystem{em: em}

	// 测试向日葵工厂
	t.Run("Sunflower", func(t *testing.T) {
		entityID, err := NewPlantEntity(em, rm, gs, mockRS, components.PlantSunflower, 0, 0)
		if err != nil {
			t.Fatalf("Failed to create sunflower: %v", err)
		}
		_, ok := ecs.GetComponent[*components.PlantTagComponent](em, entityID)
		if !ok {
			t.Error("Sunflower should have PlantTagComponent")
		}
	})

	// 测试坚果墙工厂
	t.Run("Wallnut", func(t *testing.T) {
		entityID, err := NewWallnutEntity(em, rm, gs, mockRS, 1, 0)
		if err != nil {
			t.Fatalf("Failed to create wallnut: %v", err)
		}
		_, ok := ecs.GetComponent[*components.PlantTagComponent](em, entityID)
		if !ok {
			t.Error("Wallnut should have PlantTagComponent")
		}
	})

	// 测试樱桃炸弹工厂
	t.Run("CherryBomb", func(t *testing.T) {
		entityID, err := NewCherryBombEntity(em, rm, gs, 2, 0)
		if err != nil {
			t.Fatalf("Failed to create cherrybomb: %v", err)
		}
		_, ok := ecs.GetComponent[*components.PlantTagComponent](em, entityID)
		if !ok {
			t.Error("CherryBomb should have PlantTagComponent")
		}
	})

	// 测试土豆地雷工厂
	t.Run("PotatoMine", func(t *testing.T) {
		entityID, err := NewPotatoMineEntity(em, rm, gs, 3, 0)
		if err != nil {
			t.Fatalf("Failed to create potatomine: %v", err)
		}
		_, ok := ecs.GetComponent[*components.PlantTagComponent](em, entityID)
		if !ok {
			t.Error("PotatoMine should have PlantTagComponent")
		}
	})

	// 测试寒冰射手工厂
	t.Run("SnowPea", func(t *testing.T) {
		entityID, err := NewSnowPeaEntity(em, rm, gs, mockRS, 4, 0)
		if err != nil {
			t.Fatalf("Failed to create snowpea: %v", err)
		}
		_, ok := ecs.GetComponent[*components.PlantTagComponent](em, entityID)
		if !ok {
			t.Error("SnowPea should have PlantTagComponent")
		}
	})
}

// TestPlantTagComponent_QueryMultipleEntities 测试通过 PlantTagComponent 查询多个植物实体
func TestPlantTagComponent_QueryMultipleEntities(t *testing.T) {
	em := ecs.NewEntityManager()

	// 创建多个植物实体
	plantTypes := []components.PlantType{
		components.PlantSunflower,
		components.PlantPeashooter,
		components.PlantWallnut,
	}

	var createdEntities []ecs.EntityID
	for i, pt := range plantTypes {
		entityID := em.CreateEntity()
		em.AddComponent(entityID, &components.PlantTagComponent{})
		em.AddComponent(entityID, &components.PlantComponent{
			PlantType: pt,
			GridRow:   0,
			GridCol:   i,
		})
		createdEntities = append(createdEntities, entityID)
	}

	// 创建一个非植物实体（不添加 PlantTagComponent）
	nonPlantEntity := em.CreateEntity()
	em.AddComponent(nonPlantEntity, &components.PositionComponent{X: 100, Y: 100})

	// 验证查询结果
	plantEntities := ecs.GetEntitiesWith1[*components.PlantTagComponent](em)
	if len(plantEntities) != len(plantTypes) {
		t.Errorf("Expected %d plant entities, got %d", len(plantTypes), len(plantEntities))
	}

	// 验证非植物实体不在结果中
	for _, e := range plantEntities {
		if e == nonPlantEntity {
			t.Error("Non-plant entity should not be in plant query results")
		}
	}
}

// TestPlantFactoryDeps_Structure 测试 PlantFactoryDeps 结构体
func TestPlantFactoryDeps_Structure(t *testing.T) {
	// 测试可以创建空依赖结构
	deps := PlantFactoryDeps{}

	if deps.EntityManager != nil {
		t.Error("Expected nil EntityManager")
	}
	if deps.ResourceLoader != nil {
		t.Error("Expected nil ResourceLoader")
	}
	if deps.GameState != nil {
		t.Error("Expected nil GameState")
	}
	if deps.ReanimSystem != nil {
		t.Error("Expected nil ReanimSystem")
	}

	// 测试可以正确设置依赖
	em := ecs.NewEntityManager()
	deps2 := PlantFactoryDeps{
		EntityManager: em,
	}
	if deps2.EntityManager != em {
		t.Error("EntityManager not set correctly")
	}
}

