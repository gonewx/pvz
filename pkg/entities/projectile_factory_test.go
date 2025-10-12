package entities

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestNewPeaProjectile 测试豌豆子弹实体创建
func TestNewPeaProjectile(t *testing.T) {
	// 初始化资源管理器和实体管理器（使用共享的 testAudioContext）
	rm := game.NewResourceManager(testAudioContext)
	em := ecs.NewEntityManager()

	tests := []struct {
		name    string
		startX  float64
		startY  float64
		wantErr bool
	}{
		{
			name:    "创建豌豆子弹在标准位置",
			startX:  300.0,
			startY:  250.0,
			wantErr: false,
		},
		{
			name:    "创建豌豆子弹在屏幕左侧",
			startX:  100.0,
			startY:  150.0,
			wantErr: false,
		},
		{
			name:    "创建豌豆子弹在屏幕右侧",
			startX:  800.0,
			startY:  400.0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建豌豆子弹实体
			projectileID, err := NewPeaProjectile(em, rm, tt.startX, tt.startY)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewPeaProjectile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if projectileID == 0 {
				t.Fatal("Expected valid entity ID, got 0")
			}

			// 验证 PositionComponent
			posComp, ok := em.GetComponent(projectileID, reflect.TypeOf(&components.PositionComponent{}))
			if !ok {
				t.Error("Projectile entity should have PositionComponent")
			} else {
				pos := posComp.(*components.PositionComponent)
				if pos.X != tt.startX {
					t.Errorf("Expected X %.1f, got %.1f", tt.startX, pos.X)
				}
				if pos.Y != tt.startY {
					t.Errorf("Expected Y %.1f, got %.1f", tt.startY, pos.Y)
				}
			}

			// 验证 SpriteComponent
			spriteComp, ok := em.GetComponent(projectileID, reflect.TypeOf(&components.SpriteComponent{}))
			if !ok {
				t.Error("Projectile entity should have SpriteComponent")
			} else {
				sprite := spriteComp.(*components.SpriteComponent)
				if sprite.Image == nil {
					t.Error("Projectile sprite image should not be nil")
				}
			}

			// 验证 VelocityComponent
			velComp, ok := em.GetComponent(projectileID, reflect.TypeOf(&components.VelocityComponent{}))
			if !ok {
				t.Error("Projectile entity should have VelocityComponent")
			} else {
				vel := velComp.(*components.VelocityComponent)
				if vel.VX != config.PeaBulletSpeed {
					t.Errorf("Expected VX %.1f, got %.1f", config.PeaBulletSpeed, vel.VX)
				}
				if vel.VY != 0.0 {
					t.Errorf("Expected VY 0.0, got %.1f", vel.VY)
				}
			}

			// 验证 BehaviorComponent
			behaviorComp, ok := em.GetComponent(projectileID, reflect.TypeOf(&components.BehaviorComponent{}))
			if !ok {
				t.Error("Projectile entity should have BehaviorComponent")
			} else {
				behavior := behaviorComp.(*components.BehaviorComponent)
				if behavior.Type != components.BehaviorPeaProjectile {
					t.Errorf("Expected BehaviorPeaProjectile, got %v", behavior.Type)
				}
			}

			// 验证 CollisionComponent
			collisionComp, ok := em.GetComponent(projectileID, reflect.TypeOf(&components.CollisionComponent{}))
			if !ok {
				t.Error("Projectile entity should have CollisionComponent")
			} else {
				collision := collisionComp.(*components.CollisionComponent)
				if collision.Width != config.PeaBulletWidth {
					t.Errorf("Expected Width %.1f, got %.1f", config.PeaBulletWidth, collision.Width)
				}
				if collision.Height != config.PeaBulletHeight {
					t.Errorf("Expected Height %.1f, got %.1f", config.PeaBulletHeight, collision.Height)
				}
			}
		})
	}
}

// TestNewPeaProjectile_NilParams 测试 nil 参数错误处理
func TestNewPeaProjectile_NilParams(t *testing.T) {
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
			projectileID, err := NewPeaProjectile(tt.em, tt.rm, 100.0, 100.0)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPeaProjectile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && projectileID == 0 {
				t.Error("Expected valid entity ID when no error")
			}
		})
	}
}
