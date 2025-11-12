package systems

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestSunSpawnSystem_SpawnAfter8Seconds 测试8秒后生成阳光
func TestSunSpawnSystem_SpawnAfter8Seconds(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(nil)

	system := NewSunSpawnSystem(em, rm, 250.0, 900.0, 100.0, 550.0)

	// 更新8秒
	system.Update(8.0)

	// 检查是否生成了阳光实体
	sunEntities := ecs.GetEntitiesWith1[*components.SunComponent](em)
	if len(sunEntities) != 1 {
		t.Errorf("Expected 1 sun after 8 seconds, got %d", len(sunEntities))
	}

	// 检查计时器是否重置
	if system.spawnTimer != 0 {
		t.Errorf("Expected spawnTimer reset to 0, got %.2f", system.spawnTimer)
	}

	t.Logf("✓ 1 sun spawned after 8 seconds, timer reset")
}

// TestSunSpawnSystem_MultipleCycles 测试多个生成周期
func TestSunSpawnSystem_MultipleCycles(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(nil)

	system := NewSunSpawnSystem(em, rm, 250.0, 900.0, 100.0, 550.0)

	// 更新24秒（应该生成3个阳光）
	for i := 0; i < 24; i++ {
		system.Update(1.0)
	}

	sunEntities := ecs.GetEntitiesWith1[*components.SunComponent](em)
	if len(sunEntities) != 3 {
		t.Errorf("Expected 3 suns after 24 seconds, got %d", len(sunEntities))
	}

	t.Logf("✓ 3 suns spawned after 24 seconds")
}
