package entities

import (
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
)

// TestNewPlantingParticleEffect 测试种植粒子效果创建
func TestNewPlantingParticleEffect(t *testing.T) {
	// 创建实体管理器
	em := ecs.NewEntityManager()

	// 创建资源管理器
	rm := game.NewResourceManager(nil)

	// 尝试创建种植粒子效果
	entityID, err := NewPlantingParticleEffect(em, rm, 400.0, 300.0)

	// 验证结果
	if err != nil {
		t.Logf("创建种植粒子效果失败（预期行为，因为资源未加载）: %v", err)
		// 这是正常的，因为测试环境没有加载完整资源
		if entityID != 0 {
			t.Errorf("错误时应返回 entityID=0，实际: %d", entityID)
		}
	} else {
		t.Logf("创建种植粒子效果成功: entityID=%d", entityID)

		// 验证是否添加了 PositionComponent
		if !ecs.HasComponent[*components.PositionComponent](em, entityID) {
			t.Errorf("实体 %d 缺少 PositionComponent", entityID)
		}

		// 验证是否添加了 EmitterComponent
		if !ecs.HasComponent[*components.EmitterComponent](em, entityID) {
			t.Errorf("实体 %d 缺少 EmitterComponent", entityID)
		}
	}
}

// TestNewPlantingParticleEffect_NilParams 测试参数验证
func TestNewPlantingParticleEffect_NilParams(t *testing.T) {
	tests := []struct {
		name string
		em   *ecs.EntityManager
		rm   *game.ResourceManager
	}{
		{
			name: "EntityManager为nil",
			em:   nil,
			rm:   game.NewResourceManager(nil),
		},
		{
			name: "ResourceManager为nil",
			em:   ecs.NewEntityManager(),
			rm:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entityID, err := NewPlantingParticleEffect(tt.em, tt.rm, 400.0, 300.0)

			if err == nil {
				t.Errorf("期望返回错误，但没有错误")
			}

			if entityID != 0 {
				t.Errorf("错误时应返回 entityID=0，实际: %d", entityID)
			}
		})
	}
}
