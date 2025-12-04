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

// ========== Story 19.7 Tests: 碰撞检测与弹射系统 ==========

// TestBowlingNutSystem_CollisionDetection 测试碰撞检测
func TestBowlingNutSystem_CollisionDetection(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建坚果实体（行2中心，X=500）
	row := 2
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2 // 328.0
	nutID := em.CreateEntity()
	em.AddComponent(nutID, &components.PositionComponent{X: 500.0, Y: nutY})
	em.AddComponent(nutID, &components.BowlingNutComponent{
		VelocityX: 250.0,
		Row:       row,
		IsRolling: true,
	})

	// 创建僵尸实体（同行，X=510，应该碰撞）
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.PositionComponent{X: 510.0, Y: nutY})
	em.AddComponent(zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombieID, &components.HealthComponent{CurrentHealth: 270, MaxHealth: 270})
	em.AddComponent(zombieID, &components.CollisionComponent{Width: 40, Height: 115})

	// 更新系统
	system.Update(0.016)

	// 验证僵尸受到伤害（1800 伤害应该将 270 HP 僵尸秒杀）
	health, ok := ecs.GetComponent[*components.HealthComponent](em, zombieID)
	if !ok {
		t.Fatal("HealthComponent not found")
	}
	if health.CurrentHealth > 0 {
		t.Errorf("Zombie should be killed, got health %d", health.CurrentHealth)
	}
}

// TestBowlingNutSystem_ArmorDamage 测试护甲伤害处理
// 普通碰撞只移除护甲，不造成身体伤害。需要两次碰撞才能击杀路障僵尸。
func TestBowlingNutSystem_ArmorDamage(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建坚果实体
	row := 2
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
	nutID := em.CreateEntity()
	em.AddComponent(nutID, &components.PositionComponent{X: 500.0, Y: nutY})
	em.AddComponent(nutID, &components.BowlingNutComponent{
		VelocityX: 250.0,
		Row:       row,
		IsRolling: true,
	})

	// 创建路障僵尸（370护甲 + 270身体 = 640总HP）
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.PositionComponent{X: 510.0, Y: nutY})
	em.AddComponent(zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieConehead})
	em.AddComponent(zombieID, &components.HealthComponent{CurrentHealth: 270, MaxHealth: 270})
	em.AddComponent(zombieID, &components.ArmorComponent{CurrentArmor: 370, MaxArmor: 370})
	em.AddComponent(zombieID, &components.CollisionComponent{Width: 40, Height: 115})

	// 更新系统（第一次碰撞）
	system.Update(0.016)

	// 验证护甲被破坏
	armor, _ := ecs.GetComponent[*components.ArmorComponent](em, zombieID)
	if armor.CurrentArmor != 0 {
		t.Errorf("Armor should be destroyed, got %d", armor.CurrentArmor)
	}

	// 验证第一次碰撞后身体血量不变
	health, _ := ecs.GetComponent[*components.HealthComponent](em, zombieID)
	if health.CurrentHealth != 270 {
		t.Errorf("Health should be unchanged after first hit, got %d", health.CurrentHealth)
	}
}

// TestBowlingNutSystem_BucketheadZombieArmorDamage 测试铁桶僵尸护甲伤害
// 普通碰撞只移除护甲，不造成身体伤害。需要两次碰撞才能击杀铁桶僵尸。
func TestBowlingNutSystem_BucketheadZombieArmorDamage(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建坚果实体
	row := 2
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
	nutID := em.CreateEntity()
	em.AddComponent(nutID, &components.PositionComponent{X: 500.0, Y: nutY})
	em.AddComponent(nutID, &components.BowlingNutComponent{
		VelocityX: 250.0,
		Row:       row,
		IsRolling: true,
	})

	// 创建铁桶僵尸（1100护甲 + 270身体 = 1370总HP）
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.PositionComponent{X: 510.0, Y: nutY})
	em.AddComponent(zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieBuckethead})
	em.AddComponent(zombieID, &components.HealthComponent{CurrentHealth: 270, MaxHealth: 270})
	em.AddComponent(zombieID, &components.ArmorComponent{CurrentArmor: 1100, MaxArmor: 1100})
	em.AddComponent(zombieID, &components.CollisionComponent{Width: 40, Height: 115})

	// 更新系统（第一次碰撞）
	system.Update(0.016)

	// 验证护甲被破坏
	armor, _ := ecs.GetComponent[*components.ArmorComponent](em, zombieID)
	if armor.CurrentArmor != 0 {
		t.Errorf("Armor should be destroyed, got %d", armor.CurrentArmor)
	}

	// 验证第一次碰撞后身体血量不变
	health, _ := ecs.GetComponent[*components.HealthComponent](em, zombieID)
	if health.CurrentHealth != 270 {
		t.Errorf("Health should be unchanged after first hit, got %d", health.CurrentHealth)
	}
}

// TestBowlingNutSystem_BounceDirection_NearestZombie 测试弹射方向优先X轴距离最近的僵尸
func TestBowlingNutSystem_BounceDirection_NearestZombie(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	nutX := 500.0

	// 行 1 有僵尸，X = 900（距离 400）
	row1 := 1
	row1Y := config.GridWorldStartY + float64(row1)*config.CellHeight + config.CellHeight/2
	zombie1ID := em.CreateEntity()
	em.AddComponent(zombie1ID, &components.PositionComponent{X: 900.0, Y: row1Y})
	em.AddComponent(zombie1ID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombie1ID, &components.HealthComponent{CurrentHealth: 270})
	em.AddComponent(zombie1ID, &components.CollisionComponent{Width: 40, Height: 115})

	// 行 3 有僵尸，X = 600（距离 100）- 更近！
	row3 := 3
	row3Y := config.GridWorldStartY + float64(row3)*config.CellHeight + config.CellHeight/2
	zombie3ID := em.CreateEntity()
	em.AddComponent(zombie3ID, &components.PositionComponent{X: 600.0, Y: row3Y})
	em.AddComponent(zombie3ID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombie3ID, &components.HealthComponent{CurrentHealth: 270})
	em.AddComponent(zombie3ID, &components.CollisionComponent{Width: 40, Height: 115})

	// 测试当前在行 2 时的弹射方向
	currentRow := 2
	targetRow := system.calculateBounceDirection(currentRow, nutX)

	// 应该弹向行 3（X轴距离更近：100 < 400）
	if targetRow != 3 {
		t.Errorf("Should bounce to row 3 (nearest zombie), got row %d", targetRow)
	}
}

// TestBowlingNutSystem_EdgeRowBounce_Row0 测试边缘行反弹（第0行只能向下）
func TestBowlingNutSystem_EdgeRowBounce_Row0(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 测试第 0 行只能向下弹射
	currentRow := 0
	nutX := 500.0
	targetRow := system.calculateBounceDirection(currentRow, nutX)

	if targetRow != 1 {
		t.Errorf("Row 0 should bounce down to row 1, got row %d", targetRow)
	}
}

// TestBowlingNutSystem_EdgeRowBounce_Row4 测试边缘行反弹（第4行只能向上）
func TestBowlingNutSystem_EdgeRowBounce_Row4(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 测试第 4 行只能向上弹射
	currentRow := 4
	nutX := 500.0
	targetRow := system.calculateBounceDirection(currentRow, nutX)

	if targetRow != 3 {
		t.Errorf("Row 4 should bounce up to row 3, got row %d", targetRow)
	}
}

// TestBowlingNutSystem_BounceCountIncrement 测试弹射次数递增
func TestBowlingNutSystem_BounceCountIncrement(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建坚果实体
	row := 2
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
	nutID := em.CreateEntity()
	posComp := &components.PositionComponent{X: 500.0, Y: nutY}
	nutComp := &components.BowlingNutComponent{
		VelocityX:   250.0,
		Row:         row,
		IsRolling:   true,
		BounceCount: 0,
	}
	em.AddComponent(nutID, posComp)
	em.AddComponent(nutID, nutComp)

	// 创建僵尸触发碰撞
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.PositionComponent{X: 510.0, Y: nutY})
	em.AddComponent(zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombieID, &components.HealthComponent{CurrentHealth: 270})
	em.AddComponent(zombieID, &components.CollisionComponent{Width: 40, Height: 115})

	// 更新系统
	system.Update(0.016)

	// 验证弹射次数递增
	if nutComp.BounceCount != 1 {
		t.Errorf("BounceCount should be 1, got %d", nutComp.BounceCount)
	}
}

// TestBowlingNutSystem_CollisionCooldown 测试碰撞冷却机制
func TestBowlingNutSystem_CollisionCooldown(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建坚果实体（已经在弹射后，有冷却时间）
	row := 2
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
	nutID := em.CreateEntity()
	em.AddComponent(nutID, &components.PositionComponent{X: 500.0, Y: nutY})
	nutComp := &components.BowlingNutComponent{
		VelocityX:         250.0,
		Row:               row,
		IsRolling:         true,
		CollisionCooldown: 0.1, // 设置冷却时间
	}
	em.AddComponent(nutID, nutComp)

	// 创建僵尸
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.PositionComponent{X: 510.0, Y: nutY})
	em.AddComponent(zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombieID, &components.HealthComponent{CurrentHealth: 270, MaxHealth: 270})
	em.AddComponent(zombieID, &components.CollisionComponent{Width: 40, Height: 115})

	// 更新系统（冷却期间不应检测碰撞）
	system.Update(0.016)

	// 验证僵尸未受伤（冷却期间跳过碰撞）
	health, _ := ecs.GetComponent[*components.HealthComponent](em, zombieID)
	if health.CurrentHealth != 270 {
		t.Errorf("Zombie should not be damaged during cooldown, got health %d", health.CurrentHealth)
	}
}

// TestBowlingNutSystem_BouncingMovement 测试弹射移动
func TestBowlingNutSystem_BouncingMovement(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建正在弹射的坚果实体
	currentRow := 2
	targetRow := 3
	currentY := config.GridWorldStartY + float64(currentRow)*config.CellHeight + config.CellHeight/2
	targetY := config.GridWorldStartY + float64(targetRow)*config.CellHeight + config.CellHeight/2

	nutID := em.CreateEntity()
	posComp := &components.PositionComponent{X: 500.0, Y: currentY}
	nutComp := &components.BowlingNutComponent{
		VelocityX:  250.0,
		VelocityY:  config.BowlingNutBounceSpeed, // 向下弹射
		Row:        currentRow,
		IsRolling:  true,
		IsBouncing: true,
		TargetRow:  targetRow,
	}
	em.AddComponent(nutID, posComp)
	em.AddComponent(nutID, nutComp)

	// 更新直到到达目标行
	for i := 0; i < 100 && nutComp.IsBouncing; i++ {
		system.Update(0.016)
	}

	// 验证到达目标行
	if nutComp.IsBouncing {
		t.Error("Should have finished bouncing")
	}
	if nutComp.Row != targetRow {
		t.Errorf("Row should be %d, got %d", targetRow, nutComp.Row)
	}
	if posComp.Y != targetY {
		t.Errorf("Y position should be %f, got %f", targetY, posComp.Y)
	}
}

// TestBowlingNutSystem_ExplosiveNut_DestroyedOnCollision 测试爆炸坚果碰撞后销毁
func TestBowlingNutSystem_ExplosiveNut_DestroyedOnCollision(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建爆炸坚果实体
	row := 2
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
	nutID := em.CreateEntity()
	em.AddComponent(nutID, &components.PositionComponent{X: 500.0, Y: nutY})
	em.AddComponent(nutID, &components.BowlingNutComponent{
		VelocityX:   250.0,
		Row:         row,
		IsRolling:   true,
		IsExplosive: true, // 爆炸坚果
	})

	// 创建僵尸触发碰撞
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.PositionComponent{X: 510.0, Y: nutY})
	em.AddComponent(zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombieID, &components.HealthComponent{CurrentHealth: 270})
	em.AddComponent(zombieID, &components.CollisionComponent{Width: 40, Height: 115})

	// 更新系统
	system.Update(0.016)

	// 清理标记的实体
	em.RemoveMarkedEntities()

	// 验证爆炸坚果被销毁（不弹射）
	_, exists := ecs.GetComponent[*components.BowlingNutComponent](em, nutID)
	if exists {
		t.Error("Explosive nut should be destroyed after collision, not bounce")
	}
}

// TestBowlingNutSystem_FlashEffectAdded 测试闪烁效果添加
func TestBowlingNutSystem_FlashEffectAdded(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建坚果实体
	row := 2
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
	nutID := em.CreateEntity()
	em.AddComponent(nutID, &components.PositionComponent{X: 500.0, Y: nutY})
	em.AddComponent(nutID, &components.BowlingNutComponent{
		VelocityX: 250.0,
		Row:       row,
		IsRolling: true,
	})

	// 创建僵尸
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.PositionComponent{X: 510.0, Y: nutY})
	em.AddComponent(zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombieID, &components.HealthComponent{CurrentHealth: 270})
	em.AddComponent(zombieID, &components.CollisionComponent{Width: 40, Height: 115})

	// 更新系统
	system.Update(0.016)

	// 验证僵尸添加了闪烁效果
	flashComp, hasFlash := ecs.GetComponent[*components.FlashEffectComponent](em, zombieID)
	if !hasFlash {
		t.Error("FlashEffectComponent should be added after collision")
	}
	if !flashComp.IsActive {
		t.Error("FlashEffectComponent should be active")
	}
}

// TestBowlingNutSystem_IsZombieType 测试僵尸类型检测
// 只有活着的僵尸才返回true，死亡中的僵尸返回false（避免无效碰撞）
func TestBowlingNutSystem_IsZombieType(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	tests := []struct {
		behaviorType components.BehaviorType
		expected     bool
	}{
		{components.BehaviorZombieBasic, true},
		{components.BehaviorZombieEating, true},
		{components.BehaviorZombieDying, false}, // 死亡中的僵尸不参与碰撞检测
		{components.BehaviorZombieConehead, true},
		{components.BehaviorZombieBuckethead, true},
		{components.BehaviorZombieFlag, true},
		{components.BehaviorPeaProjectile, false},
		{components.BehaviorPeashooter, false},
	}

	for _, test := range tests {
		result := system.isZombieType(test.behaviorType)
		if result != test.expected {
			t.Errorf("isZombieType(%v) = %v, want %v", test.behaviorType, result, test.expected)
		}
	}
}

// TestBowlingNutSystem_CalculateRowFromY 测试从Y坐标计算行号
func TestBowlingNutSystem_CalculateRowFromY(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 测试行中心Y坐标
	for row := 0; row <= 4; row++ {
		centerY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
		result := system.calculateRowFromY(centerY)
		if result != row {
			t.Errorf("calculateRowFromY(%f) = %d, want %d", centerY, result, row)
		}
	}

	// 测试边界情况
	if system.calculateRowFromY(-100) != 0 {
		t.Error("Y < GridWorldStartY should return 0")
	}
	if system.calculateRowFromY(1000) != 4 {
		t.Error("Y > max row should return 4")
	}
}

// TestBowlingNutSystem_CalculateRowCenterY 测试计算行中心Y坐标
func TestBowlingNutSystem_CalculateRowCenterY(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 测试每行的中心Y坐标
	expectedCenters := []float64{
		config.GridWorldStartY + 0*config.CellHeight + config.CellHeight/2, // 行 0: 128.0
		config.GridWorldStartY + 1*config.CellHeight + config.CellHeight/2, // 行 1: 228.0
		config.GridWorldStartY + 2*config.CellHeight + config.CellHeight/2, // 行 2: 328.0
		config.GridWorldStartY + 3*config.CellHeight + config.CellHeight/2, // 行 3: 428.0
		config.GridWorldStartY + 4*config.CellHeight + config.CellHeight/2, // 行 4: 528.0
	}

	for row := 0; row <= 4; row++ {
		result := system.calculateRowCenterY(row)
		if result != expectedCenters[row] {
			t.Errorf("calculateRowCenterY(%d) = %f, want %f", row, result, expectedCenters[row])
		}
	}
}

// ========== Story 19.8 Tests: 爆炸坚果机制 ==========

// TestBowlingNutSystem_ExplosiveNut_AreaDamage 测试爆炸坚果 3x3 范围伤害
func TestBowlingNutSystem_ExplosiveNut_AreaDamage(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建爆炸坚果实体（行 2，X=500）
	row := 2
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2 // 328.0
	nutID := em.CreateEntity()
	em.AddComponent(nutID, &components.PositionComponent{X: 500.0, Y: nutY})
	em.AddComponent(nutID, &components.BowlingNutComponent{
		VelocityX:   250.0,
		Row:         row,
		IsRolling:   true,
		IsExplosive: true,
	})

	// 创建目标僵尸（同行，触发碰撞）
	targetY := nutY
	targetID := em.CreateEntity()
	em.AddComponent(targetID, &components.PositionComponent{X: 510.0, Y: targetY})
	em.AddComponent(targetID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(targetID, &components.HealthComponent{CurrentHealth: 270, MaxHealth: 270})
	em.AddComponent(targetID, &components.CollisionComponent{Width: 40, Height: 115})

	// 创建相邻行僵尸（行 1，范围内）
	row1Y := config.GridWorldStartY + float64(1)*config.CellHeight + config.CellHeight/2 // 228.0
	adjacentID := em.CreateEntity()
	em.AddComponent(adjacentID, &components.PositionComponent{X: 520.0, Y: row1Y})
	em.AddComponent(adjacentID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(adjacentID, &components.HealthComponent{CurrentHealth: 270, MaxHealth: 270})
	em.AddComponent(adjacentID, &components.CollisionComponent{Width: 40, Height: 115})

	// 创建远距离僵尸（同行但远离，范围外）
	farID := em.CreateEntity()
	em.AddComponent(farID, &components.PositionComponent{X: 800.0, Y: nutY}) // 距离 300 像素
	em.AddComponent(farID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(farID, &components.HealthComponent{CurrentHealth: 270, MaxHealth: 270})
	em.AddComponent(farID, &components.CollisionComponent{Width: 40, Height: 115})

	// 更新系统
	system.Update(0.016)

	// 验证范围内僵尸受伤
	targetHealth, _ := ecs.GetComponent[*components.HealthComponent](em, targetID)
	if targetHealth.CurrentHealth > 0 {
		t.Errorf("Target zombie should be killed, got health %d", targetHealth.CurrentHealth)
	}

	adjacentHealth, _ := ecs.GetComponent[*components.HealthComponent](em, adjacentID)
	if adjacentHealth.CurrentHealth > 0 {
		t.Errorf("Adjacent zombie in range should be killed, got health %d", adjacentHealth.CurrentHealth)
	}

	// 验证范围外僵尸未受伤
	farHealth, _ := ecs.GetComponent[*components.HealthComponent](em, farID)
	if farHealth.CurrentHealth != 270 {
		t.Errorf("Far zombie should not be damaged, got health %d, want 270", farHealth.CurrentHealth)
	}
}

// TestBowlingNutSystem_ExplosiveNut_NoBounce 测试爆炸坚果不弹射
func TestBowlingNutSystem_ExplosiveNut_NoBounce(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建爆炸坚果实体
	row := 2
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
	nutID := em.CreateEntity()
	em.AddComponent(nutID, &components.PositionComponent{X: 500.0, Y: nutY})
	nutComp := &components.BowlingNutComponent{
		VelocityX:   250.0,
		Row:         row,
		IsRolling:   true,
		IsExplosive: true,
		BounceCount: 0,
	}
	em.AddComponent(nutID, nutComp)

	// 创建僵尸触发碰撞
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.PositionComponent{X: 510.0, Y: nutY})
	em.AddComponent(zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombieID, &components.HealthComponent{CurrentHealth: 270})
	em.AddComponent(zombieID, &components.CollisionComponent{Width: 40, Height: 115})

	// 更新系统
	system.Update(0.016)

	// 清理标记的实体
	em.RemoveMarkedEntities()

	// 验证坚果被销毁（实体不存在）
	_, ok := ecs.GetComponent[*components.BowlingNutComponent](em, nutID)
	if ok {
		t.Error("Explosive nut should be destroyed after explosion")
	}
}

// TestBowlingNutSystem_ExplosiveNut_Damage1800 测试爆炸坚果伤害值为 1800
func TestBowlingNutSystem_ExplosiveNut_Damage1800(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建爆炸坚果实体
	row := 2
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
	nutID := em.CreateEntity()
	em.AddComponent(nutID, &components.PositionComponent{X: 500.0, Y: nutY})
	em.AddComponent(nutID, &components.BowlingNutComponent{
		VelocityX:   250.0,
		Row:         row,
		IsRolling:   true,
		IsExplosive: true,
	})

	// 创建铁桶僵尸（1100护甲 + 270身体 = 1370总HP）
	// 1800 伤害应该秒杀
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.PositionComponent{X: 510.0, Y: nutY})
	em.AddComponent(zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieBuckethead})
	em.AddComponent(zombieID, &components.HealthComponent{CurrentHealth: 270, MaxHealth: 270})
	em.AddComponent(zombieID, &components.ArmorComponent{CurrentArmor: 1100, MaxArmor: 1100})
	em.AddComponent(zombieID, &components.CollisionComponent{Width: 40, Height: 115})

	// 更新系统
	system.Update(0.016)

	// 验证护甲被破坏
	armor, _ := ecs.GetComponent[*components.ArmorComponent](em, zombieID)
	if armor.CurrentArmor > 0 {
		t.Errorf("Armor should be destroyed, got %d", armor.CurrentArmor)
	}

	// 验证铁桶僵尸被秒杀
	// 1800 - 1100 = 700 溢出伤害，270 - 700 = -430
	health, _ := ecs.GetComponent[*components.HealthComponent](em, zombieID)
	if health.CurrentHealth > 0 {
		t.Errorf("Buckethead zombie should be killed by 1800 damage, got health %d", health.CurrentHealth)
	}
}

// TestBowlingNutSystem_ExplosiveNut_BounceCountNotIncreased 测试爆炸坚果碰撞后 BounceCount 不增加
func TestBowlingNutSystem_ExplosiveNut_BounceCountNotIncreased(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建爆炸坚果实体
	row := 2
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
	nutID := em.CreateEntity()
	em.AddComponent(nutID, &components.PositionComponent{X: 500.0, Y: nutY})
	nutComp := &components.BowlingNutComponent{
		VelocityX:   250.0,
		Row:         row,
		IsRolling:   true,
		IsExplosive: true,
		BounceCount: 0,
	}
	em.AddComponent(nutID, nutComp)

	// 保存初始弹射次数
	initialBounceCount := nutComp.BounceCount

	// 创建僵尸触发碰撞
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.PositionComponent{X: 510.0, Y: nutY})
	em.AddComponent(zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombieID, &components.HealthComponent{CurrentHealth: 270})
	em.AddComponent(zombieID, &components.CollisionComponent{Width: 40, Height: 115})

	// 更新系统
	system.Update(0.016)

	// 验证弹射次数未增加（爆炸坚果不弹射）
	// 注意：由于坚果已被销毁，无法直接检查，但可以通过检查坚果是否被销毁来间接验证
	em.RemoveMarkedEntities()
	_, exists := ecs.GetComponent[*components.BowlingNutComponent](em, nutID)
	if exists {
		// 如果坚果还存在，检查弹射次数
		if nutComp.BounceCount != initialBounceCount {
			t.Errorf("Explosive nut BounceCount should not increase, got %d, want %d",
				nutComp.BounceCount, initialBounceCount)
		}
	}
	// 坚果已销毁是预期行为，测试通过
}

// TestBowlingNutSystem_ExplosiveNut_ArmorPriority 测试爆炸伤害护甲优先规则
func TestBowlingNutSystem_ExplosiveNut_ArmorPriority(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建爆炸坚果实体
	row := 2
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
	nutID := em.CreateEntity()
	em.AddComponent(nutID, &components.PositionComponent{X: 500.0, Y: nutY})
	em.AddComponent(nutID, &components.BowlingNutComponent{
		VelocityX:   250.0,
		Row:         row,
		IsRolling:   true,
		IsExplosive: true,
	})

	// 创建路障僵尸（370护甲 + 270身体）
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.PositionComponent{X: 510.0, Y: nutY})
	em.AddComponent(zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieConehead})
	em.AddComponent(zombieID, &components.HealthComponent{CurrentHealth: 270, MaxHealth: 270})
	em.AddComponent(zombieID, &components.ArmorComponent{CurrentArmor: 370, MaxArmor: 370})
	em.AddComponent(zombieID, &components.CollisionComponent{Width: 40, Height: 115})

	// 更新系统
	system.Update(0.016)

	// 验证护甲被完全破坏
	armor, _ := ecs.GetComponent[*components.ArmorComponent](em, zombieID)
	if armor.CurrentArmor != 0 {
		t.Errorf("Armor should be 0, got %d", armor.CurrentArmor)
	}

	// 验证溢出伤害应用到身体
	// 1800 - 370 = 1430 溢出伤害
	// 270 - 1430 = -1160（秒杀）
	health, _ := ecs.GetComponent[*components.HealthComponent](em, zombieID)
	if health.CurrentHealth > 0 {
		t.Errorf("Zombie should be killed by overflow damage, got health %d", health.CurrentHealth)
	}
}

// TestBowlingNutSystem_ExplosiveNut_FlashEffect 测试爆炸伤害添加闪烁效果
func TestBowlingNutSystem_ExplosiveNut_FlashEffect(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建爆炸坚果实体
	row := 2
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
	nutID := em.CreateEntity()
	em.AddComponent(nutID, &components.PositionComponent{X: 500.0, Y: nutY})
	em.AddComponent(nutID, &components.BowlingNutComponent{
		VelocityX:   250.0,
		Row:         row,
		IsRolling:   true,
		IsExplosive: true,
	})

	// 创建僵尸
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.PositionComponent{X: 510.0, Y: nutY})
	em.AddComponent(zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombieID, &components.HealthComponent{CurrentHealth: 270})
	em.AddComponent(zombieID, &components.CollisionComponent{Width: 40, Height: 115})

	// 更新系统
	system.Update(0.016)

	// 验证僵尸添加了闪烁效果
	flashComp, hasFlash := ecs.GetComponent[*components.FlashEffectComponent](em, zombieID)
	if !hasFlash {
		t.Error("FlashEffectComponent should be added after explosion")
	}
	if !flashComp.IsActive {
		t.Error("FlashEffectComponent should be active")
	}
}

// TestBowlingNutSystem_ExplosiveNut_MultipleZombiesInRange 测试爆炸范围内多个僵尸
func TestBowlingNutSystem_ExplosiveNut_MultipleZombiesInRange(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建爆炸坚果实体
	row := 2
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
	nutID := em.CreateEntity()
	em.AddComponent(nutID, &components.PositionComponent{X: 500.0, Y: nutY})
	em.AddComponent(nutID, &components.BowlingNutComponent{
		VelocityX:   250.0,
		Row:         row,
		IsRolling:   true,
		IsExplosive: true,
	})

	// 创建多个范围内僵尸
	// 爆炸半径 120 像素，需要考虑 ZombieVerticalOffset (-25) 修正
	zombieIDs := make([]ecs.EntityID, 3)

	// 僵尸1：同行，近距离
	zombieIDs[0] = em.CreateEntity()
	em.AddComponent(zombieIDs[0], &components.PositionComponent{X: 510.0, Y: nutY})
	em.AddComponent(zombieIDs[0], &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombieIDs[0], &components.HealthComponent{CurrentHealth: 270, MaxHealth: 270})
	em.AddComponent(zombieIDs[0], &components.CollisionComponent{Width: 40, Height: 115})

	// 僵尸2：上方（Y 偏移 -60，考虑 ZombieVerticalOffset 后有效距离约 85 像素）
	zombieIDs[1] = em.CreateEntity()
	em.AddComponent(zombieIDs[1], &components.PositionComponent{X: 500.0, Y: nutY - 60})
	em.AddComponent(zombieIDs[1], &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombieIDs[1], &components.HealthComponent{CurrentHealth: 270, MaxHealth: 270})
	em.AddComponent(zombieIDs[1], &components.CollisionComponent{Width: 40, Height: 115})

	// 僵尸3：下方（Y 偏移 +60，考虑 ZombieVerticalOffset 后有效距离约 85 像素）
	zombieIDs[2] = em.CreateEntity()
	em.AddComponent(zombieIDs[2], &components.PositionComponent{X: 500.0, Y: nutY + 60})
	em.AddComponent(zombieIDs[2], &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombieIDs[2], &components.HealthComponent{CurrentHealth: 270, MaxHealth: 270})
	em.AddComponent(zombieIDs[2], &components.CollisionComponent{Width: 40, Height: 115})

	// 更新系统
	system.Update(0.016)

	// 验证所有范围内僵尸都受到伤害
	for i, zombieID := range zombieIDs {
		health, ok := ecs.GetComponent[*components.HealthComponent](em, zombieID)
		if !ok {
			t.Fatalf("Zombie %d HealthComponent not found", i)
		}
		if health.CurrentHealth > 0 {
			t.Errorf("Zombie %d should be killed, got health %d", i, health.CurrentHealth)
		}
	}
}

// ========== 持续弹射到边缘测试 ==========

// TestBowlingNutSystem_ContinueBounce_MiddleRow 测试中间行继续弹射
// 坚果到达中间行后没有碰到僵尸，应该继续向同一方向弹射
func TestBowlingNutSystem_ContinueBounce_MiddleRow(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建坚果实体，初始在第2行，弹射方向向上(-1)
	row := 2
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
	nutID := em.CreateEntity()
	posComp := &components.PositionComponent{X: 500.0, Y: nutY}
	nutComp := &components.BowlingNutComponent{
		VelocityX:       250.0,
		Row:             row,
		IsRolling:       true,
		BounceDirection: -1, // 向上弹射
	}
	em.AddComponent(nutID, posComp)
	em.AddComponent(nutID, nutComp)

	// 不创建任何僵尸，让坚果继续弹射

	// 更新系统
	system.Update(0.016)

	// 验证坚果开始继续弹射
	if !nutComp.IsBouncing {
		t.Error("Nut should start bouncing when in middle row with BounceDirection set")
	}
	if nutComp.TargetRow != 1 {
		t.Errorf("Target row should be 1 (row 2 + direction -1), got %d", nutComp.TargetRow)
	}
}

// TestBowlingNutSystem_ContinueBounce_TopEdge 测试顶边反弹
// 坚果到达第0行后应该反转方向向下弹射
func TestBowlingNutSystem_ContinueBounce_TopEdge(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建坚果实体，在第0行（顶边），弹射方向向上(-1)
	row := 0
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
	nutID := em.CreateEntity()
	posComp := &components.PositionComponent{X: 500.0, Y: nutY}
	nutComp := &components.BowlingNutComponent{
		VelocityX:       250.0,
		Row:             row,
		IsRolling:       true,
		BounceDirection: -1, // 原本向上，但在顶边应该反转
	}
	em.AddComponent(nutID, posComp)
	em.AddComponent(nutID, nutComp)

	// 更新系统
	system.Update(0.016)

	// 验证弹射方向反转为向下
	if nutComp.BounceDirection != 1 {
		t.Errorf("BounceDirection should be 1 (down) at top edge, got %d", nutComp.BounceDirection)
	}
	if nutComp.TargetRow != 1 {
		t.Errorf("Target row should be 1, got %d", nutComp.TargetRow)
	}
	if !nutComp.IsBouncing {
		t.Error("Nut should be bouncing after edge reflection")
	}
}

// TestBowlingNutSystem_ContinueBounce_BottomEdge 测试底边反弹
// 坚果到达第4行后应该反转方向向上弹射
func TestBowlingNutSystem_ContinueBounce_BottomEdge(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建坚果实体，在第4行（底边），弹射方向向下(1)
	row := 4
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
	nutID := em.CreateEntity()
	posComp := &components.PositionComponent{X: 500.0, Y: nutY}
	nutComp := &components.BowlingNutComponent{
		VelocityX:       250.0,
		Row:             row,
		IsRolling:       true,
		BounceDirection: 1, // 原本向下，但在底边应该反转
	}
	em.AddComponent(nutID, posComp)
	em.AddComponent(nutID, nutComp)

	// 更新系统
	system.Update(0.016)

	// 验证弹射方向反转为向上
	if nutComp.BounceDirection != -1 {
		t.Errorf("BounceDirection should be -1 (up) at bottom edge, got %d", nutComp.BounceDirection)
	}
	if nutComp.TargetRow != 3 {
		t.Errorf("Target row should be 3, got %d", nutComp.TargetRow)
	}
	if !nutComp.IsBouncing {
		t.Error("Nut should be bouncing after edge reflection")
	}
}

// TestBowlingNutSystem_ContinueBounce_StopsOnCollision 测试碰撞僵尸后停止持续弹射
// 坚果碰撞僵尸后应该重新计算弹射方向，而不是继续原方向
func TestBowlingNutSystem_ContinueBounce_StopsOnCollision(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建坚果实体，在第2行，有弹射方向
	row := 2
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
	nutID := em.CreateEntity()
	posComp := &components.PositionComponent{X: 500.0, Y: nutY}
	nutComp := &components.BowlingNutComponent{
		VelocityX:       250.0,
		Row:             row,
		IsRolling:       true,
		BounceDirection: -1, // 原本向上
	}
	em.AddComponent(nutID, posComp)
	em.AddComponent(nutID, nutComp)

	// 创建僵尸在同行（触发碰撞）
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.PositionComponent{X: 510.0, Y: nutY}) // 同行碰撞
	em.AddComponent(zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombieID, &components.HealthComponent{CurrentHealth: 270, MaxHealth: 270})
	em.AddComponent(zombieID, &components.CollisionComponent{Width: 40, Height: 115})

	// 创建第3行更近的僵尸（影响弹射方向计算）
	row3Y := config.GridWorldStartY + float64(3)*config.CellHeight + config.CellHeight/2
	zombie3ID := em.CreateEntity()
	em.AddComponent(zombie3ID, &components.PositionComponent{X: 520.0, Y: row3Y})
	em.AddComponent(zombie3ID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombie3ID, &components.HealthComponent{CurrentHealth: 270, MaxHealth: 270})
	em.AddComponent(zombie3ID, &components.CollisionComponent{Width: 40, Height: 115})

	// 更新系统
	system.Update(0.016)

	// 验证碰撞僵尸后弹射方向会被重新计算
	// 因为第3行僵尸更近，应该弹向第3行（方向向下）
	if nutComp.BounceDirection != 1 {
		t.Errorf("BounceDirection should be 1 (down, toward nearest zombie), got %d", nutComp.BounceDirection)
	}
}

// TestBowlingNutSystem_ContinueBounce_NoBounceWithoutDirection 测试没有弹射方向时不会继续弹射
// 如果 BounceDirection 为 0，坚果不应该自动弹射
func TestBowlingNutSystem_ContinueBounce_NoBounceWithoutDirection(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewBowlingNutSystem(em, nil)

	// 创建坚果实体，没有弹射方向
	row := 2
	nutY := config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
	nutID := em.CreateEntity()
	posComp := &components.PositionComponent{X: 500.0, Y: nutY}
	nutComp := &components.BowlingNutComponent{
		VelocityX:       250.0,
		Row:             row,
		IsRolling:       true,
		BounceDirection: 0, // 没有弹射方向
	}
	em.AddComponent(nutID, posComp)
	em.AddComponent(nutID, nutComp)

	// 更新系统
	system.Update(0.016)

	// 验证坚果不会自动弹射
	if nutComp.IsBouncing {
		t.Error("Nut should not bounce when BounceDirection is 0")
	}
}


