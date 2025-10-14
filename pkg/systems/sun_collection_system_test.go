package systems

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestSunCollectionArrival 测试阳光到达目标位置后被删除
func TestSunCollectionArrival(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.CameraX = 0 // 重置 CameraX 确保测试一致性
	targetX := 21.0
	targetY := 80.0
	system := NewSunCollectionSystem(em, gs, targetX, targetY)

	// 创建正在收集的阳光，位置非常接近目标（在阈值内）
	id := em.CreateEntity()
	em.AddComponent(id, &components.PositionComponent{X: 22.0, Y: 81.0}) // 距离目标约1.4像素
	em.AddComponent(id, &components.SunComponent{State: components.SunCollecting})

	// 更新系统
	system.Update(0.016) // 一帧

	// 验证实体被标记删除
	entities := em.GetEntitiesWith(
		reflect.TypeOf(&components.SunComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
	)

	// 实体应该被标记删除，但在RemoveMarkedEntities调用前仍然存在
	if len(entities) != 1 {
		t.Errorf("Expected 1 entity before cleanup, got %d", len(entities))
	}

	// 清理标记的实体
	em.RemoveMarkedEntities()

	// 现在应该被删除了
	entities = em.GetEntitiesWith(
		reflect.TypeOf(&components.SunComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
	)

	if len(entities) != 0 {
		t.Errorf("Expected 0 entities after cleanup, got %d", len(entities))
	}
}

// TestSunCollectionNotYetArrived 测试阳光未到达目标时保持存在
func TestSunCollectionNotYetArrived(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.CameraX = 0 // 重置 CameraX 确保测试一致性
	targetX := 21.0
	targetY := 80.0
	system := NewSunCollectionSystem(em, gs, targetX, targetY)

	// 创建正在收集的阳光，距离目标较远
	id := em.CreateEntity()
	em.AddComponent(id, &components.PositionComponent{X: 200.0, Y: 300.0}) // 距离目标很远
	em.AddComponent(id, &components.SunComponent{State: components.SunCollecting})

	// 更新系统
	system.Update(0.016)

	// 验证实体仍然存在
	entities := em.GetEntitiesWith(
		reflect.TypeOf(&components.SunComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
	)

	if len(entities) != 1 {
		t.Errorf("Expected 1 entity (not yet arrived), got %d", len(entities))
	}

	// 验证实体ID正确
	if len(entities) > 0 && entities[0] != id {
		t.Error("Entity ID mismatch")
	}

	// 清理标记的实体（不应该有）
	em.RemoveMarkedEntities()

	entities = em.GetEntitiesWith(
		reflect.TypeOf(&components.SunComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
	)

	if len(entities) != 1 {
		t.Errorf("Entity should still exist after cleanup, got %d entities", len(entities))
	}
}

// TestSunCollectionDistanceThreshold 测试距离阈值检测（边界情况）
func TestSunCollectionDistanceThreshold(t *testing.T) {
	targetX := 21.0
	targetY := 80.0
	threshold := 10.0 // 阈值是10像素

	tests := []struct {
		name         string
		sunX         float64
		sunY         float64
		shouldArrive bool
		description  string
	}{
		{"正好在阈值边界上", 21.0, 70.0, false, "距离正好10像素,边界上不删除"},
		{"稍微超出阈值", 21.0, 69.5, false, "距离10.5像素,超出阈值"},
		{"在目标位置", 21.0, 80.0, true, "距离0,肯定到达"},
		{"在阈值内", 21.0, 75.0, true, "距离5像素,在阈值内"},
		{"X轴上在阈值内", 26.0, 80.0, true, "X轴偏移5像素"},
		{"Y轴上在阈值内", 21.0, 85.0, true, "Y轴偏移5像素"},
		{"对角线在阈值内", 26.0, 87.0, true, "对角线距离约8.6像素"},
		{"对角线超出阈值", 28.0, 88.0, false, "对角线距离约10.6像素"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建新的EntityManager以隔离测试
			emTest := ecs.NewEntityManager()
			gsTest := game.GetGameState()
			gsTest.CameraX = 0 // 重置 CameraX 确保测试一致性
			systemTest := NewSunCollectionSystem(emTest, gsTest, targetX, targetY)

			id := emTest.CreateEntity()
			emTest.AddComponent(id, &components.PositionComponent{X: tt.sunX, Y: tt.sunY})
			emTest.AddComponent(id, &components.SunComponent{State: components.SunCollecting})

			// 更新系统
			systemTest.Update(0.016)

			// 清理标记的实体
			emTest.RemoveMarkedEntities()

			// 检查实体是否被删除
			entities := emTest.GetEntitiesWith(
				reflect.TypeOf(&components.SunComponent{}),
				reflect.TypeOf(&components.PositionComponent{}),
			)

			if tt.shouldArrive {
				if len(entities) != 0 {
					t.Errorf("%s: Expected sun to be deleted (distance < %.1f), but it still exists", tt.description, threshold)
				}
			} else {
				if len(entities) != 1 {
					t.Errorf("%s: Expected sun to remain (distance >= %.1f), but got %d entities", tt.description, threshold, len(entities))
				}
			}
		})
	}
}

// TestSunCollectionOnlyCollectingState 测试只有SunCollecting状态的阳光会被处理
func TestSunCollectionOnlyCollectingState(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.CameraX = 0 // 重置 CameraX 确保测试一致性
	targetX := 21.0
	targetY := 80.0
	system := NewSunCollectionSystem(em, gs, targetX, targetY)

	// 创建已落地的阳光（非收集状态），位置在目标附近
	id1 := em.CreateEntity()
	em.AddComponent(id1, &components.PositionComponent{X: 22.0, Y: 81.0})
	em.AddComponent(id1, &components.SunComponent{State: components.SunLanded}) // 不是SunCollecting

	// 创建正在掉落的阳光，位置在目标附近
	id2 := em.CreateEntity()
	em.AddComponent(id2, &components.PositionComponent{X: 22.0, Y: 81.0})
	em.AddComponent(id2, &components.SunComponent{State: components.SunFalling}) // 不是SunCollecting

	// 更新系统
	system.Update(0.016)

	// 清理标记的实体
	em.RemoveMarkedEntities()

	// 验证非SunCollecting状态的阳光都还存在
	entities := em.GetEntitiesWith(
		reflect.TypeOf(&components.SunComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
	)

	if len(entities) != 2 {
		t.Errorf("Expected 2 entities (non-collecting suns should not be deleted), got %d", len(entities))
	}
}

// TestSunCollectionMultipleSuns 测试同时收集多个阳光
func TestSunCollectionMultipleSuns(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	gs.CameraX = 0 // 重置 CameraX 确保测试一致性
	targetX := 21.0
	targetY := 80.0
	system := NewSunCollectionSystem(em, gs, targetX, targetY)

	// 创建3个正在收集的阳光
	// 第1个：已到达
	em.CreateEntity()
	em.AddComponent(ecs.EntityID(1), &components.PositionComponent{X: 22.0, Y: 81.0})
	em.AddComponent(ecs.EntityID(1), &components.SunComponent{State: components.SunCollecting})

	// 第2个：未到达
	em.CreateEntity()
	em.AddComponent(ecs.EntityID(2), &components.PositionComponent{X: 200.0, Y: 300.0})
	em.AddComponent(ecs.EntityID(2), &components.SunComponent{State: components.SunCollecting})

	// 第3个：已到达
	em.CreateEntity()
	em.AddComponent(ecs.EntityID(3), &components.PositionComponent{X: 20.0, Y: 79.0})
	em.AddComponent(ecs.EntityID(3), &components.SunComponent{State: components.SunCollecting})

	// 更新系统
	system.Update(0.016)

	// 清理标记的实体
	em.RemoveMarkedEntities()

	// 应该剩下1个实体（未到达的那个）
	entities := em.GetEntitiesWith(
		reflect.TypeOf(&components.SunComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
	)

	if len(entities) != 1 {
		t.Errorf("Expected 1 entity remaining (not yet arrived), got %d", len(entities))
	}

	// 验证剩下的是ID=2的实体
	if len(entities) > 0 && entities[0] != ecs.EntityID(2) {
		t.Errorf("Expected entity ID 2 to remain, got %d", entities[0])
	}
}
