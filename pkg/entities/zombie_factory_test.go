package entities

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestNewZombieEntity 测试僵尸实体创建
func TestNewZombieEntity(t *testing.T) {
	// 初始化资源管理器和实体管理器
	rm := game.NewResourceManager(testAudioContext)
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
				expectedY := config.GridWorldStartY + float64(tt.row)*config.CellHeight + 30.0
				if pos.X != tt.spawnX {
					t.Errorf("Position X mismatch: got %.1f, want %.1f", pos.X, tt.spawnX)
				}
				if pos.Y != expectedY {
					t.Errorf("Position Y mismatch: got %.1f, want %.1f", pos.Y, expectedY)
				}
			}

			// 验证 SpriteComponent
			spriteComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.SpriteComponent{}))
			if !ok {
				t.Error("Zombie entity should have SpriteComponent")
			} else {
				sprite := spriteComp.(*components.SpriteComponent)
				if sprite.Image == nil {
					t.Error("SpriteComponent Image should not be nil")
				}
			}

			// 验证 AnimationComponent
			animComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.AnimationComponent{}))
			if !ok {
				t.Error("Zombie entity should have AnimationComponent")
			} else {
				anim := animComp.(*components.AnimationComponent)
				if len(anim.Frames) != 22 {
					t.Errorf("Expected 22 animation frames, got %d", len(anim.Frames))
				}
				if anim.FrameSpeed != 0.1 {
					t.Errorf("Expected FrameSpeed 0.1, got %.2f", anim.FrameSpeed)
				}
				if !anim.IsLooping {
					t.Error("Zombie animation should loop")
				}
			}

			// 验证 VelocityComponent
			velComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.VelocityComponent{}))
			if !ok {
				t.Error("Zombie entity should have VelocityComponent")
			} else {
				vel := velComp.(*components.VelocityComponent)
				if vel.VX != -30.0 {
					t.Errorf("Expected VX -30.0, got %.1f", vel.VX)
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
	rm := game.NewResourceManager(testAudioContext)
	em := ecs.NewEntityManager()

	tests := []struct {
		name    string
		em      *ecs.EntityManager
		rm      *game.ResourceManager
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
