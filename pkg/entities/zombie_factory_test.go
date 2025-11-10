package entities

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
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

	mockRS := &mockReanimSystem{em: em}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建僵尸实体
			zombieID, err := NewZombieEntity(em, rm, mockRS, tt.row, tt.spawnX)
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
				if len(reanim.CurrentAnimations) == 0 {
					t.Error("ReanimComponent.CurrentAnimations should not be empty")
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

	mockRS := &mockReanimSystem{em: em}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zombieID, err := NewZombieEntity(tt.em, tt.rm, mockRS, 0, 1450.0)
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

	mockRS := &mockReanimSystem{em: em}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建路障僵尸实体
			zombieID, err := NewConeheadZombieEntity(em, rm, mockRS, tt.row, tt.spawnX)
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

	mockRS := &mockReanimSystem{em: em}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建铁桶僵尸实体
			zombieID, err := NewBucketheadZombieEntity(em, rm, mockRS, tt.row, tt.spawnX)
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
