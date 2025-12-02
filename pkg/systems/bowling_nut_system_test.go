package systems

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
)

// TestNewBowlingNutSystem 测试系统创建
func TestNewBowlingNutSystem(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	if system == nil {
		t.Fatal("NewBowlingNutSystem returned nil")
	}
	if system.entityManager != em {
		t.Error("EntityManager not set correctly")
	}
	if system.soundPlayers == nil {
		t.Error("soundPlayers map not initialized")
	}
}

// TestBowlingNutSystem_RollingUpdate 测试滚动位置更新
func TestBowlingNutSystem_RollingUpdate(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建测试实体
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PositionComponent{X: 300.0, Y: 200.0})
	em.AddComponent(entityID, &components.BowlingNutComponent{
		VelocityX: 250.0,
		IsRolling: true,
	})

	// 更新 1 秒
	system.Update(1.0)

	posComp, ok := ecs.GetComponent[*components.PositionComponent](em, entityID)
	if !ok {
		t.Fatal("PositionComponent not found")
	}

	expectedX := 300.0 + 250.0*1.0 // 初始位置 + 速度 * 时间
	if posComp.X < expectedX-0.1 || posComp.X > expectedX+0.1 {
		t.Errorf("Position X = %f, want %f (±0.1)", posComp.X, expectedX)
	}
}

// TestBowlingNutSystem_BoundaryDestruction 测试边界销毁逻辑
func TestBowlingNutSystem_BoundaryDestruction(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建接近边界的实体
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PositionComponent{X: config.BackgroundWidth - 10, Y: 200.0})
	em.AddComponent(entityID, &components.BowlingNutComponent{
		VelocityX: 250.0,
		IsRolling: true,
	})

	// 更新足够时间使其越界 (10 / 250 = 0.04 秒)
	system.Update(0.1) // X = 1390 + 25 = 1415 > 1400

	// 清理标记的实体
	em.RemoveMarkedEntities()

	// 验证实体已被销毁
	_, exists := ecs.GetComponent[*components.PositionComponent](em, entityID)
	if exists {
		t.Error("Entity should be destroyed after crossing boundary")
	}
}

// TestBowlingNutSystem_YCoordinateUnchanged 测试 Y 坐标不变
func TestBowlingNutSystem_YCoordinateUnchanged(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	initialY := 200.0
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PositionComponent{X: 300.0, Y: initialY})
	em.AddComponent(entityID, &components.BowlingNutComponent{
		VelocityX: 250.0,
		IsRolling: true,
	})

	// 多次更新
	for i := 0; i < 10; i++ {
		system.Update(0.1)
	}

	posComp, ok := ecs.GetComponent[*components.PositionComponent](em, entityID)
	if !ok {
		t.Fatal("PositionComponent not found")
	}

	if posComp.Y != initialY {
		t.Errorf("Y coordinate changed: got %f, want %f", posComp.Y, initialY)
	}
}

// TestBowlingNutSystem_NotRolling 测试不滚动状态
func TestBowlingNutSystem_NotRolling(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	initialX := 300.0
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PositionComponent{X: initialX, Y: 200.0})
	em.AddComponent(entityID, &components.BowlingNutComponent{
		VelocityX: 250.0,
		IsRolling: false, // 不滚动
	})

	// 更新
	system.Update(1.0)

	posComp, ok := ecs.GetComponent[*components.PositionComponent](em, entityID)
	if !ok {
		t.Fatal("PositionComponent not found")
	}

	if posComp.X != initialX {
		t.Errorf("Position should not change when not rolling: got %f, want %f", posComp.X, initialX)
	}
}

// TestBowlingNutSystem_MultipleEntities 测试多个实体更新
func TestBowlingNutSystem_MultipleEntities(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建三个实体
	entity1 := em.CreateEntity()
	em.AddComponent(entity1, &components.PositionComponent{X: 100.0, Y: 100.0})
	em.AddComponent(entity1, &components.BowlingNutComponent{VelocityX: 200.0, IsRolling: true})

	entity2 := em.CreateEntity()
	em.AddComponent(entity2, &components.PositionComponent{X: 200.0, Y: 200.0})
	em.AddComponent(entity2, &components.BowlingNutComponent{VelocityX: 300.0, IsRolling: true})

	entity3 := em.CreateEntity()
	em.AddComponent(entity3, &components.PositionComponent{X: 300.0, Y: 300.0})
	em.AddComponent(entity3, &components.BowlingNutComponent{VelocityX: 250.0, IsRolling: false})

	// 更新 0.5 秒
	system.Update(0.5)

	// 检查实体1
	pos1, _ := ecs.GetComponent[*components.PositionComponent](em, entity1)
	expected1 := 100.0 + 200.0*0.5 // 200
	if pos1.X < expected1-0.1 || pos1.X > expected1+0.1 {
		t.Errorf("Entity1 X = %f, want %f", pos1.X, expected1)
	}

	// 检查实体2
	pos2, _ := ecs.GetComponent[*components.PositionComponent](em, entity2)
	expected2 := 200.0 + 300.0*0.5 // 350
	if pos2.X < expected2-0.1 || pos2.X > expected2+0.1 {
		t.Errorf("Entity2 X = %f, want %f", pos2.X, expected2)
	}

	// 检查实体3（不滚动）
	pos3, _ := ecs.GetComponent[*components.PositionComponent](em, entity3)
	if pos3.X != 300.0 {
		t.Errorf("Entity3 X = %f, want 300.0 (should not move)", pos3.X)
	}
}

// TestBowlingNutSystem_SoundStateFlagSet 测试音效状态标志设置
func TestBowlingNutSystem_SoundStateFlagSet(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PositionComponent{X: 300.0, Y: 200.0})
	em.AddComponent(entityID, &components.BowlingNutComponent{
		VelocityX:    250.0,
		IsRolling:    true,
		SoundPlaying: false,
	})

	// 更新系统
	system.Update(0.1)

	nutComp, _ := ecs.GetComponent[*components.BowlingNutComponent](em, entityID)
	if !nutComp.SoundPlaying {
		t.Error("SoundPlaying flag should be set to true after update")
	}
}

// TestBowlingNutSystem_StopAllSounds 测试停止所有音效
func TestBowlingNutSystem_StopAllSounds(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 手动添加一些音效播放器条目
	system.soundPlayers[1] = nil
	system.soundPlayers[2] = nil

	// 调用停止所有音效
	system.StopAllSounds()

	if len(system.soundPlayers) != 0 {
		t.Errorf("soundPlayers should be empty after StopAllSounds, got %d entries", len(system.soundPlayers))
	}
}

// TestBowlingNutSystem_ZeroVelocity 测试零速度
func TestBowlingNutSystem_ZeroVelocity(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	initialX := 300.0
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PositionComponent{X: initialX, Y: 200.0})
	em.AddComponent(entityID, &components.BowlingNutComponent{
		VelocityX: 0, // 零速度
		IsRolling: true,
	})

	system.Update(1.0)

	posComp, _ := ecs.GetComponent[*components.PositionComponent](em, entityID)
	if posComp.X != initialX {
		t.Errorf("Position should not change with zero velocity: got %f, want %f", posComp.X, initialX)
	}
}

// TestBowlingNutSystem_SmallDeltaTime 测试小时间步长
func TestBowlingNutSystem_SmallDeltaTime(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PositionComponent{X: 300.0, Y: 200.0})
	em.AddComponent(entityID, &components.BowlingNutComponent{
		VelocityX: 250.0,
		IsRolling: true,
	})

	// 非常小的时间步长
	dt := 0.001
	system.Update(dt)

	posComp, _ := ecs.GetComponent[*components.PositionComponent](em, entityID)
	expectedX := 300.0 + 250.0*dt
	if posComp.X < expectedX-0.001 || posComp.X > expectedX+0.001 {
		t.Errorf("Position X = %f, want %f", posComp.X, expectedX)
	}
}

