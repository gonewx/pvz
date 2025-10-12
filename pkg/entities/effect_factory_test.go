package entities

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestNewPeaBulletHitEffect 测试豌豆子弹击中效果实体创建
func TestNewPeaBulletHitEffect(t *testing.T) {
	// 初始化资源管理器和实体管理器（使用共享的 testAudioContext）
	rm := game.NewResourceManager(testAudioContext)
	em := ecs.NewEntityManager()

	tests := []struct {
		name    string
		x       float64
		y       float64
		wantErr bool
	}{
		{
			name:    "创建击中效果在标准位置",
			x:       300.0,
			y:       250.0,
			wantErr: false,
		},
		{
			name:    "创建击中效果在屏幕左侧",
			x:       100.0,
			y:       150.0,
			wantErr: false,
		},
		{
			name:    "创建击中效果在屏幕右侧",
			x:       800.0,
			y:       400.0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建击中效果实体
			effectID, err := NewPeaBulletHitEffect(em, rm, tt.x, tt.y)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewPeaBulletHitEffect() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if effectID == 0 {
				t.Fatal("Expected valid entity ID, got 0")
			}

			// 验证 PositionComponent
			posComp, ok := em.GetComponent(effectID, reflect.TypeOf(&components.PositionComponent{}))
			if !ok {
				t.Error("Hit effect entity should have PositionComponent")
			} else {
				pos := posComp.(*components.PositionComponent)
				if pos.X != tt.x {
					t.Errorf("Expected X %.1f, got %.1f", tt.x, pos.X)
				}
				if pos.Y != tt.y {
					t.Errorf("Expected Y %.1f, got %.1f", tt.y, pos.Y)
				}
			}

			// 验证 SpriteComponent
			spriteComp, ok := em.GetComponent(effectID, reflect.TypeOf(&components.SpriteComponent{}))
			if !ok {
				t.Error("Hit effect entity should have SpriteComponent")
			} else {
				sprite := spriteComp.(*components.SpriteComponent)
				if sprite.Image == nil {
					t.Error("Hit effect sprite image should not be nil")
				}
			}

			// 验证 BehaviorComponent
			behaviorComp, ok := em.GetComponent(effectID, reflect.TypeOf(&components.BehaviorComponent{}))
			if !ok {
				t.Error("Hit effect entity should have BehaviorComponent")
			} else {
				behavior := behaviorComp.(*components.BehaviorComponent)
				if behavior.Type != components.BehaviorPeaBulletHit {
					t.Errorf("Expected BehaviorPeaBulletHit, got %v", behavior.Type)
				}
			}

			// 验证 TimerComponent
			timerComp, ok := em.GetComponent(effectID, reflect.TypeOf(&components.TimerComponent{}))
			if !ok {
				t.Error("Hit effect entity should have TimerComponent")
			} else {
				timer := timerComp.(*components.TimerComponent)
				if timer.Name != "hit_effect_duration" {
					t.Errorf("Expected timer name 'hit_effect_duration', got '%s'", timer.Name)
				}
				if timer.TargetTime != config.HitEffectDuration {
					t.Errorf("Expected TargetTime %.2f, got %.2f", config.HitEffectDuration, timer.TargetTime)
				}
				if timer.CurrentTime != 0.0 {
					t.Errorf("Expected CurrentTime 0.0, got %.2f", timer.CurrentTime)
				}
			}
		})
	}
}

// TestNewPeaBulletHitEffect_NilParams 测试 nil 参数错误处理
func TestNewPeaBulletHitEffect_NilParams(t *testing.T) {
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
			effectID, err := NewPeaBulletHitEffect(tt.em, tt.rm, 100.0, 100.0)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPeaBulletHitEffect() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && effectID == 0 {
				t.Error("Expected valid entity ID when no error")
			}
			if tt.wantErr && effectID != 0 {
				t.Error("Expected entity ID 0 when error occurs")
			}
		})
	}
}
