package systems

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestNewLawnmowerSystem 测试除草车系统创建
func TestNewLawnmowerSystem(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := &game.ResourceManager{}
	gs := game.GetGameState()

	system := NewLawnmowerSystem(em, rm, nil, gs)

	if system == nil {
		t.Fatal("NewLawnmowerSystem should return non-nil")
	}
	if system.entityManager != em {
		t.Error("EntityManager not set correctly")
	}
	if system.resourceManager != rm {
		t.Error("ResourceManager not set correctly")
	}
	if system.gameState != gs {
		t.Error("GameState not set correctly")
	}
	if system.stateEntityID == 0 {
		t.Error("StateEntityID should be non-zero")
	}

	// 验证全局状态组件已创建
	state, ok := ecs.GetComponent[*components.LawnmowerStateComponent](em, system.stateEntityID)
	if !ok {
		t.Fatal("LawnmowerStateComponent should be created")
	}
	if state.UsedLanes == nil {
		t.Error("UsedLanes map should be initialized")
	}
	if len(state.UsedLanes) != 0 {
		t.Error("UsedLanes should be empty initially")
	}
}

// TestLawnmowerSystemGetEntityLane 测试计算实体所在行
func TestLawnmowerSystemGetEntityLane(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewLawnmowerSystem(em, nil, nil, nil)

	tests := []struct {
		y        float64
		expected int
	}{
		{config.GridWorldStartY + 0*config.CellHeight + config.CellHeight/2, 1},
		{config.GridWorldStartY + 1*config.CellHeight + config.CellHeight/2, 2},
		{config.GridWorldStartY + 2*config.CellHeight + config.CellHeight/2, 3},
		{config.GridWorldStartY + 3*config.CellHeight + config.CellHeight/2, 4},
		{config.GridWorldStartY + 4*config.CellHeight + config.CellHeight/2, 5},
		{0.0, 1},                         // 超出下界
		{config.GridWorldStartY - 10, 1}, // 负偏移
		{1000.0, 5},                      // 超出上界
	}

	for _, tt := range tests {
		lane := system.getEntityLane(tt.y)
		if lane != tt.expected {
			t.Errorf("getEntityLane(%.1f) = %d, expected %d", tt.y, lane, tt.expected)
		}
	}
}

// TestLawnmowerSystemUpdatePosition 测试除草车位置更新
func TestLawnmowerSystemUpdatePosition(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewLawnmowerSystem(em, nil, nil, nil)

	// 创建一个移动中的除草车
	lawnmowerID := em.CreateEntity()
	ecs.AddComponent(em, lawnmowerID, &components.PositionComponent{
		X: 100.0,
		Y: 200.0,
	})
	ecs.AddComponent(em, lawnmowerID, &components.LawnmowerComponent{
		Lane:        3,
		IsTriggered: true,
		IsMoving:    true,
		Speed:       300.0,
	})

	// 更新1秒
	system.updateLawnmowerPositions(1.0)

	// 验证位置更新
	pos, _ := ecs.GetComponent[*components.PositionComponent](em, lawnmowerID)
	expected := 100.0 + 300.0*1.0 // 起始位置 + 速度 * 时间
	if pos.X != expected {
		t.Errorf("Expected X=%.1f after 1s, got %.1f", expected, pos.X)
	}
}

// TestLawnmowerSystemCompletion 测试除草车离开屏幕检测
func TestLawnmowerSystemCompletion(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewLawnmowerSystem(em, nil, nil, nil)

	// 创建一个移动中的除草车（已经离开屏幕）
	lawnmowerID := em.CreateEntity()
	ecs.AddComponent(em, lawnmowerID, &components.PositionComponent{
		X: config.LawnmowerDeletionBoundary + 10, // 超过删除边界
		Y: 200.0,
	})
	ecs.AddComponent(em, lawnmowerID, &components.LawnmowerComponent{
		Lane:        3,
		IsTriggered: true,
		IsMoving:    true,
		Speed:       300.0,
	})

	// 检查完成
	system.checkLawnmowerCompletion()

	// 实体标记为待删除，需要调用 RemoveMarkedEntities 才会真正删除
	em.RemoveMarkedEntities()

	// 验证实体已删除（通过检查组件是否还存在）
	_, exists := ecs.GetComponent[*components.LawnmowerComponent](em, lawnmowerID)
	if exists {
		t.Error("Lawnmower should be destroyed after leaving screen")
	}

	// 验证状态已更新
	state, _ := ecs.GetComponent[*components.LawnmowerStateComponent](em, system.stateEntityID)
	if !state.UsedLanes[3] {
		t.Error("Lane 3 should be marked as used")
	}
}

// TestIsZombieType 测试僵尸类型判断
func TestIsZombieType(t *testing.T) {
	tests := []struct {
		behaviorType components.BehaviorType
		expected     bool
	}{
		{components.BehaviorZombieBasic, true},
		{components.BehaviorZombieConehead, true},
		{components.BehaviorZombieBuckethead, true},
		{components.BehaviorPeashooter, false},
		{components.BehaviorSunflower, false},
		{components.BehaviorWallnut, false},
	}

	for _, tt := range tests {
		result := isZombieType(tt.behaviorType)
		if result != tt.expected {
			t.Errorf("isZombieType(%v) = %v, expected %v", tt.behaviorType, result, tt.expected)
		}
	}
}

// TestLawnmowerSystemZombieCollision 测试除草车与僵尸碰撞
func TestLawnmowerSystemZombieCollision(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	system := NewLawnmowerSystem(em, nil, nil, gs)

	// 创建移动中的除草车（第3行）
	lawnmowerID := em.CreateEntity()
	lane := 3
	lawnmowerY := config.GridWorldStartY + float64(lane-1)*config.CellHeight + config.CellHeight/2
	ecs.AddComponent(em, lawnmowerID, &components.PositionComponent{
		X: 500.0,
		Y: lawnmowerY,
	})
	ecs.AddComponent(em, lawnmowerID, &components.LawnmowerComponent{
		Lane:        lane,
		IsTriggered: true,
		IsMoving:    true,
		Speed:       300.0,
	})

	// 创建同行的僵尸（在碰撞范围内）
	zombieID := em.CreateEntity()
	ecs.AddComponent(em, zombieID, &components.PositionComponent{
		X: 510.0, // 距离除草车10像素（< CollisionRange）
		Y: lawnmowerY,
	})
	ecs.AddComponent(em, zombieID, &components.BehaviorComponent{
		Type: components.BehaviorZombieBasic,
	})
	ecs.AddComponent(em, zombieID, &components.HealthComponent{
		CurrentHealth: 270,
		MaxHealth:     270,
	})

	// 记录初始消灭僵尸数
	initialKills := gs.ZombiesKilled

	// 检测碰撞
	system.checkZombieCollisions()

	// 实体标记为待删除，需要调用 RemoveMarkedEntities 才会真正删除
	em.RemoveMarkedEntities()

	// 验证僵尸实体已被删除（原版行为：除草车碾压的僵尸直接删除，不播放死亡动画）
	_, exists := ecs.GetComponent[*components.HealthComponent](em, zombieID)
	if exists {
		t.Error("Zombie should be destroyed immediately after lawnmower collision")
	}

	// 验证消灭僵尸计数增加
	if gs.ZombiesKilled != initialKills+1 {
		t.Errorf("Zombies killed should increment from %d to %d, got %d",
			initialKills, initialKills+1, gs.ZombiesKilled)
	}
}

// TestLawnmowerSystemZombieNoCollision 测试除草车与僵尸不碰撞（不同行）
func TestLawnmowerSystemZombieNoCollision(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	system := NewLawnmowerSystem(em, nil, nil, gs)

	// 创建移动中的除草车（第3行）
	lawnmowerID := em.CreateEntity()
	lane := 3
	lawnmowerY := config.GridWorldStartY + float64(lane-1)*config.CellHeight + config.CellHeight/2
	ecs.AddComponent(em, lawnmowerID, &components.PositionComponent{
		X: 500.0,
		Y: lawnmowerY,
	})
	ecs.AddComponent(em, lawnmowerID, &components.LawnmowerComponent{
		Lane:        lane,
		IsTriggered: true,
		IsMoving:    true,
		Speed:       300.0,
	})

	// 创建不同行的僵尸（第2行）
	zombieID := em.CreateEntity()
	zombieY := config.GridWorldStartY + float64(2-1)*config.CellHeight + config.CellHeight/2
	ecs.AddComponent(em, zombieID, &components.PositionComponent{
		X: 500.0, // X坐标相同，但Y坐标不同
		Y: zombieY,
	})
	ecs.AddComponent(em, zombieID, &components.BehaviorComponent{
		Type: components.BehaviorZombieBasic,
	})
	ecs.AddComponent(em, zombieID, &components.HealthComponent{
		CurrentHealth: 270,
		MaxHealth:     270,
	})

	// 检测碰撞
	system.checkZombieCollisions()

	// 验证僵尸生命值未改变
	health, _ := ecs.GetComponent[*components.HealthComponent](em, zombieID)
	if health.CurrentHealth != 270 {
		t.Errorf("Zombie health should remain 270 (different lane), got %d", health.CurrentHealth)
	}
}
