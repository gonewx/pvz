package entities

import (
	"reflect"
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
)

// TestNewZombieEntity 测试僵尸实体创建
func TestNewZombieEntity(t *testing.T) {
	// 初始化资源管理器和实体管理器
	rm := newMockResourceManager()
	em := ecs.NewEntityManager()

	tests := []struct {
		name    string
		row     int
		spawnX  float64
		wantErr bool
	}{
		{
			name:    "创建僵尸在第0行",
			row:     0,
			spawnX:  1450.0,
			wantErr: false,
		},
		{
			name:    "创建僵尸在第2行",
			row:     2,
			spawnX:  1500.0,
			wantErr: false,
		},
		{
			name:    "创建僵尸在第4行",
			row:     4,
			spawnX:  1450.0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建僵尸实体
			zombieID, err := NewZombieEntity(em, rm, tt.row, tt.spawnX)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewZombieEntity() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if zombieID == 0 {
				t.Fatal("Expected valid entity ID, got 0")
			}

			// 验证 PositionComponent
			posComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.PositionComponent{}))
			if !ok {
				t.Error("Zombie entity should have PositionComponent")
			} else {
				pos := posComp.(*components.PositionComponent)
				// Y坐标 = 行起点 + 行偏移 + 行中心偏移 + 垂直修正
				expectedY := config.GridWorldStartY + float64(tt.row)*config.CellHeight + config.CellHeight/2.0 + config.ZombieVerticalOffset
				if pos.X != tt.spawnX {
					t.Errorf("Position X mismatch: got %.1f, want %.1f", pos.X, tt.spawnX)
				}
				if pos.Y != expectedY {
					t.Errorf("Position Y mismatch: got %.1f, want %.1f", pos.Y, expectedY)
				}
			}

			// 验证 ReanimComponent（替代 SpriteComponent + AnimationComponent）
			reanimComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.ReanimComponent{}))
			if !ok {
				t.Error("Zombie entity should have ReanimComponent")
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
			// Story 17.10: 使用 UnitID + ComboName 模式（配置驱动）
			animCmd, ok := ecs.GetComponent[*components.AnimationCommandComponent](em, zombieID)
			if !ok {
				t.Error("Zombie entity should have AnimationCommandComponent")
			} else {
				if animCmd.UnitID == "" && animCmd.AnimationName == "" {
					t.Error("AnimationCommandComponent should have UnitID or AnimationName")
				}
				if animCmd.Processed {
					t.Error("AnimationCommand should not be processed yet (Processed=false)")
				}
			}

			// 验证 VelocityComponent
			// Story 8.3: 僵尸预生成时速度为 0，等待激活
			velComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.VelocityComponent{}))
			if !ok {
				t.Error("Zombie entity should have VelocityComponent")
			} else {
				vel := velComp.(*components.VelocityComponent)
				if vel.VX != 0.0 {
					t.Errorf("Expected VX 0.0 (待命状态), got %.1f", vel.VX)
				}
				if vel.VY != 0.0 {
					t.Errorf("Expected VY 0.0, got %.1f", vel.VY)
				}
			}

			// 验证 BehaviorComponent
			behaviorComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.BehaviorComponent{}))
			if !ok {
				t.Error("Zombie entity should have BehaviorComponent")
			} else {
				behavior := behaviorComp.(*components.BehaviorComponent)
				if behavior.Type != components.BehaviorZombieBasic {
					t.Errorf("Expected BehaviorZombieBasic, got %v", behavior.Type)
				}
			}

			// 验证 HealthComponent
			healthComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.HealthComponent{}))
			if !ok {
				t.Error("Zombie entity should have HealthComponent")
			} else {
				health := healthComp.(*components.HealthComponent)
				if health.CurrentHealth != 270 {
					t.Errorf("Expected CurrentHealth 270, got %d", health.CurrentHealth)
				}
				if health.MaxHealth != 270 {
					t.Errorf("Expected MaxHealth 270, got %d", health.MaxHealth)
				}
			}
		})
	}
}

// TestNewZombieEntity_ErrorHandling 测试错误处理
func TestNewZombieEntity_ErrorHandling(t *testing.T) {
	rm := newMockResourceManager()
	em := ecs.NewEntityManager()

	tests := []struct {
		name    string
		em      *ecs.EntityManager
		rm      ResourceLoader
		wantErr bool
	}{
		{
			name:    "EntityManager为nil",
			em:      nil,
			rm:      rm,
			wantErr: true,
		},
		{
			name:    "ResourceManager为nil",
			em:      em,
			rm:      nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zombieID, err := NewZombieEntity(tt.em, tt.rm, 0, 1450.0)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewZombieEntity() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && zombieID == 0 {
				t.Error("Expected valid entity ID when no error")
			}
		})
	}
}

// TestNewConeheadZombieEntity 测试路障僵尸实体创建
func TestNewConeheadZombieEntity(t *testing.T) {
	// 初始化资源管理器和实体管理器
	rm := newMockResourceManager()
	em := ecs.NewEntityManager()

	tests := []struct {
		name    string
		row     int
		spawnX  float64
		wantErr bool
	}{
		{
			name:    "创建路障僵尸在第0行",
			row:     0,
			spawnX:  1450.0,
			wantErr: false,
		},
		{
			name:    "创建路障僵尸在第2行",
			row:     2,
			spawnX:  1500.0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建路障僵尸实体
			zombieID, err := NewConeheadZombieEntity(em, rm, tt.row, tt.spawnX)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewConeheadZombieEntity() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if zombieID == 0 {
				t.Fatal("Expected valid entity ID, got 0")
			}

			// 验证 BehaviorComponent
			behaviorComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.BehaviorComponent{}))
			if !ok {
				t.Error("Conehead zombie entity should have BehaviorComponent")
			} else {
				behavior := behaviorComp.(*components.BehaviorComponent)
				if behavior.Type != components.BehaviorZombieConehead {
					t.Errorf("Expected BehaviorZombieConehead, got %v", behavior.Type)
				}
			}

			// 验证 ArmorComponent (关键特性)
			armorComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.ArmorComponent{}))
			if !ok {
				t.Error("Conehead zombie entity should have ArmorComponent")
			} else {
				armor := armorComp.(*components.ArmorComponent)
				if armor.CurrentArmor != config.ConeheadZombieArmorHealth {
					t.Errorf("Expected CurrentArmor %d, got %d", config.ConeheadZombieArmorHealth, armor.CurrentArmor)
				}
				if armor.MaxArmor != config.ConeheadZombieArmorHealth {
					t.Errorf("Expected MaxArmor %d, got %d", config.ConeheadZombieArmorHealth, armor.MaxArmor)
				}
			}

			// 验证 HealthComponent (身体生命值)
			healthComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.HealthComponent{}))
			if !ok {
				t.Error("Conehead zombie entity should have HealthComponent")
			} else {
				health := healthComp.(*components.HealthComponent)
				if health.CurrentHealth != config.ZombieDefaultHealth {
					t.Errorf("Expected CurrentHealth %d, got %d", config.ZombieDefaultHealth, health.CurrentHealth)
				}
				if health.MaxHealth != config.ZombieDefaultHealth {
					t.Errorf("Expected MaxHealth %d, got %d", config.ZombieDefaultHealth, health.MaxHealth)
				}
			}

			// Story 6.3: 验证 ReanimComponent
			reanimComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.ReanimComponent{}))
			if !ok {
				t.Error("Conehead zombie entity should have ReanimComponent")
			} else {
				reanim := reanimComp.(*components.ReanimComponent)
				if reanim.ReanimXML == nil {
					t.Error("ReanimComponent.ReanimXML should not be nil")
				}
				if reanim.PartImages == nil {
					t.Error("ReanimComponent.PartImages should not be nil")
				}
			}
		})
	}
}

// TestNewBucketheadZombieEntity 测试铁桶僵尸实体创建
func TestNewBucketheadZombieEntity(t *testing.T) {
	// 初始化资源管理器和实体管理器
	rm := newMockResourceManager()
	em := ecs.NewEntityManager()

	tests := []struct {
		name    string
		row     int
		spawnX  float64
		wantErr bool
	}{
		{
			name:    "创建铁桶僵尸在第0行",
			row:     0,
			spawnX:  1450.0,
			wantErr: false,
		},
		{
			name:    "创建铁桶僵尸在第4行",
			row:     4,
			spawnX:  1500.0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建铁桶僵尸实体
			zombieID, err := NewBucketheadZombieEntity(em, rm, tt.row, tt.spawnX)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewBucketheadZombieEntity() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if zombieID == 0 {
				t.Fatal("Expected valid entity ID, got 0")
			}

			// 验证 BehaviorComponent
			behaviorComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.BehaviorComponent{}))
			if !ok {
				t.Error("Buckethead zombie entity should have BehaviorComponent")
			} else {
				behavior := behaviorComp.(*components.BehaviorComponent)
				if behavior.Type != components.BehaviorZombieBuckethead {
					t.Errorf("Expected BehaviorZombieBuckethead, got %v", behavior.Type)
				}
			}

			// 验证 ArmorComponent (关键特性)
			armorComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.ArmorComponent{}))
			if !ok {
				t.Error("Buckethead zombie entity should have ArmorComponent")
			} else {
				armor := armorComp.(*components.ArmorComponent)
				if armor.CurrentArmor != config.BucketheadZombieArmorHealth {
					t.Errorf("Expected CurrentArmor %d, got %d", config.BucketheadZombieArmorHealth, armor.CurrentArmor)
				}
				if armor.MaxArmor != config.BucketheadZombieArmorHealth {
					t.Errorf("Expected MaxArmor %d, got %d", config.BucketheadZombieArmorHealth, armor.MaxArmor)
				}
			}

			// 验证 HealthComponent (身体生命值)
			healthComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.HealthComponent{}))
			if !ok {
				t.Error("Buckethead zombie entity should have HealthComponent")
			} else {
				health := healthComp.(*components.HealthComponent)
				if health.CurrentHealth != config.ZombieDefaultHealth {
					t.Errorf("Expected CurrentHealth %d, got %d", config.ZombieDefaultHealth, health.CurrentHealth)
				}
				if health.MaxHealth != config.ZombieDefaultHealth {
					t.Errorf("Expected MaxHealth %d, got %d", config.ZombieDefaultHealth, health.MaxHealth)
				}
			}

			// Story 6.3: 验证 ReanimComponent
			reanimComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.ReanimComponent{}))
			if !ok {
				t.Error("Buckethead zombie entity should have ReanimComponent")
			} else {
				reanim := reanimComp.(*components.ReanimComponent)
				if reanim.ReanimXML == nil {
					t.Error("ReanimComponent.ReanimXML should not be nil")
				}
				if reanim.PartImages == nil {
					t.Error("ReanimComponent.PartImages should not be nil")
				}
			}
		})
	}
}

// TestConeheadZombieTotalHealth 测试路障僵尸总有效生命值
func TestConeheadZombieTotalHealth(t *testing.T) {
	// 路障僵尸总生命值 = 护甲值 + 身体生命值
	expectedTotal := config.ConeheadZombieArmorHealth + config.ZombieDefaultHealth
	actualTotal := 370 + 270 // 根据配置

	if actualTotal != 640 {
		t.Errorf("路障僵尸总生命值应为640，实际为 %d", actualTotal)
	}

	if expectedTotal != 640 {
		t.Errorf("配置中路障僵尸总生命值应为640，实际为 %d", expectedTotal)
	}
}

// TestBucketheadZombieTotalHealth 测试铁桶僵尸总有效生命值
func TestBucketheadZombieTotalHealth(t *testing.T) {
	// 铁桶僵尸总生命值 = 护甲值 + 身体生命值
	expectedTotal := config.BucketheadZombieArmorHealth + config.ZombieDefaultHealth
	actualTotal := 1100 + 270 // 根据配置

	if actualTotal != 1370 {
		t.Errorf("铁桶僵尸总生命值应为1370，实际为 %d", actualTotal)
	}

	if expectedTotal != 1370 {
		t.Errorf("配置中铁桶僵尸总生命值应为1370，实际为 %d", expectedTotal)
	}
}

// =============================================================================
// Story 18.4: 僵尸工厂注册表测试
// =============================================================================

// TestGetZombieFactory_KnownTypes 测试已知僵尸类型的工厂查找
func TestGetZombieFactory_KnownTypes(t *testing.T) {
	tests := []struct {
		unitID      string
		description string
	}{
		{"zombie", "普通僵尸"},
		{"zombie_conehead", "路障僵尸"},
		{"zombie_buckethead", "铁桶僵尸"},
		{"zombie_flag", "旗帜僵尸"},
		{"zombie_polevaulter", "撑杆僵尸"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			factory, found := GetZombieFactory(tt.unitID)
			if !found {
				t.Errorf("GetZombieFactory(%q) should find factory for %s", tt.unitID, tt.description)
			}
			if factory == nil {
				t.Errorf("GetZombieFactory(%q) returned nil factory", tt.unitID)
			}
		})
	}
}

// TestGetZombieFactory_UnknownType 测试未知僵尸类型的工厂查找
func TestGetZombieFactory_UnknownType(t *testing.T) {
	unknownTypes := []string{
		"zombie_unknown",
		"zombie_dancing",
		"zombie_gargantuar",
		"invalid",
		"",
	}

	for _, unitID := range unknownTypes {
		t.Run(unitID, func(t *testing.T) {
			factory, found := GetZombieFactory(unitID)
			if found {
				t.Errorf("GetZombieFactory(%q) should not find factory for unknown type", unitID)
			}
			if factory != nil {
				t.Errorf("GetZombieFactory(%q) should return nil for unknown type", unitID)
			}
		})
	}
}

// TestGetDefaultZombieFactory 测试默认工厂函数
func TestGetDefaultZombieFactory(t *testing.T) {
	factory := GetDefaultZombieFactory()
	if factory == nil {
		t.Fatal("GetDefaultZombieFactory() should not return nil")
	}

	// 验证默认工厂能创建普通僵尸
	rm := newMockResourceManager()
	em := ecs.NewEntityManager()

	entityID, err := factory(em, rm, 0, 1450.0)
	if err != nil {
		t.Fatalf("Default factory failed to create zombie: %v", err)
	}
	if entityID == 0 {
		t.Error("Default factory returned invalid entity ID")
	}

	// 验证是普通僵尸
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](em, entityID)
	if !ok {
		t.Fatal("Zombie should have BehaviorComponent")
	}
	if behavior.Type != components.BehaviorZombieBasic {
		t.Errorf("Default factory should create basic zombie, got %v", behavior.Type)
	}
}

// TestZombieFactoryRegistration_CreateEntity 测试工厂注册表创建实体
func TestZombieFactoryRegistration_CreateEntity(t *testing.T) {
	rm := newMockResourceManager()
	em := ecs.NewEntityManager()

	tests := []struct {
		unitID       string
		expectedType components.BehaviorType
	}{
		{"zombie", components.BehaviorZombieBasic},
		{"zombie_conehead", components.BehaviorZombieConehead},
		{"zombie_buckethead", components.BehaviorZombieBuckethead},
		{"zombie_flag", components.BehaviorZombieFlag},
		{"zombie_polevaulter", components.BehaviorZombiePolevaulter},
	}

	for _, tt := range tests {
		t.Run(tt.unitID, func(t *testing.T) {
			factory, found := GetZombieFactory(tt.unitID)
			if !found {
				t.Fatalf("Factory not found for %s", tt.unitID)
			}

			entityID, err := factory(em, rm, 2, 1450.0)
			if err != nil {
				t.Fatalf("Factory failed to create %s: %v", tt.unitID, err)
			}

			behavior, ok := ecs.GetComponent[*components.BehaviorComponent](em, entityID)
			if !ok {
				t.Fatalf("%s should have BehaviorComponent", tt.unitID)
			}
			if behavior.Type != tt.expectedType {
				t.Errorf("%s: expected behavior type %v, got %v", tt.unitID, tt.expectedType, behavior.Type)
			}

			// 验证 ZombieTagComponent（Story 18.4 关键）
			_, hasTag := ecs.GetComponent[*components.ZombieTagComponent](em, entityID)
			if !hasTag {
				t.Errorf("%s should have ZombieTagComponent", tt.unitID)
			}
		})
	}
}
