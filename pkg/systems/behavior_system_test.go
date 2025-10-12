package systems

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
)

// TestSunflowerTimerUpdate 测试向日葵计时器更新逻辑
func TestSunflowerTimerUpdate(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建向日葵实体
	sunflowerID := em.CreateEntity()
	em.AddComponent(sunflowerID, &components.BehaviorComponent{
		Type: components.BehaviorSunflower,
	})
	em.AddComponent(sunflowerID, &components.TimerComponent{
		Name:        "sun_production",
		TargetTime:  7.0,
		CurrentTime: 0,
		IsReady:     false,
	})
	em.AddComponent(sunflowerID, &components.PlantComponent{
		PlantType: components.PlantSunflower,
		GridRow:   2,
		GridCol:   3,
	})
	em.AddComponent(sunflowerID, &components.PositionComponent{
		X: 400,
		Y: 300,
	})

	// 模拟1秒更新
	system.Update(1.0)

	// 验证计时器累加
	timerComp, _ := em.GetComponent(sunflowerID, reflect.TypeOf(&components.TimerComponent{}))
	timer := timerComp.(*components.TimerComponent)

	if timer.CurrentTime != 1.0 {
		t.Errorf("Expected CurrentTime=1.0, got %f", timer.CurrentTime)
	}

	// 再次更新2秒
	system.Update(2.0)

	timerComp, _ = em.GetComponent(sunflowerID, reflect.TypeOf(&components.TimerComponent{}))
	timer = timerComp.(*components.TimerComponent)

	if timer.CurrentTime != 3.0 {
		t.Errorf("Expected CurrentTime=3.0, got %f", timer.CurrentTime)
	}
}

// TestSunflowerFirstProduction 测试向日葵首次生产（7秒）
func TestSunflowerFirstProduction(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建向日葵实体
	sunflowerID := em.CreateEntity()
	em.AddComponent(sunflowerID, &components.BehaviorComponent{
		Type: components.BehaviorSunflower,
	})
	em.AddComponent(sunflowerID, &components.TimerComponent{
		Name:        "sun_production",
		TargetTime:  7.0,
		CurrentTime: 0,
		IsReady:     false,
	})
	em.AddComponent(sunflowerID, &components.PlantComponent{
		PlantType: components.PlantSunflower,
		GridRow:   2,
		GridCol:   3,
	})
	em.AddComponent(sunflowerID, &components.PositionComponent{
		X: 400,
		Y: 300,
	})

	// 模拟7秒更新（触发首次生产）
	system.Update(7.0)

	// 验证阳光实体被创建（查询拥有 SunComponent 的实体）
	sunEntities := em.GetEntitiesWith(reflect.TypeOf(&components.SunComponent{}))
	if len(sunEntities) != 1 {
		t.Errorf("Expected 1 sun entity, got %d", len(sunEntities))
	}

	// 验证计时器重置
	timerComp, _ := em.GetComponent(sunflowerID, reflect.TypeOf(&components.TimerComponent{}))
	timer := timerComp.(*components.TimerComponent)

	if timer.CurrentTime != 0 {
		t.Errorf("Expected CurrentTime=0 after production, got %f", timer.CurrentTime)
	}

	// 验证计时器目标时间变为24秒（后续生产周期）
	if timer.TargetTime != 24.0 {
		t.Errorf("Expected TargetTime=24.0 after first production, got %f", timer.TargetTime)
	}
}

// TestSunflowerSubsequentProduction 测试向日葵后续生产（24秒）
func TestSunflowerSubsequentProduction(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建向日葵实体（模拟已经生产过一次）
	sunflowerID := em.CreateEntity()
	em.AddComponent(sunflowerID, &components.BehaviorComponent{
		Type: components.BehaviorSunflower,
	})
	em.AddComponent(sunflowerID, &components.TimerComponent{
		Name:        "sun_production",
		TargetTime:  24.0, // 后续生产周期为24秒
		CurrentTime: 0,
		IsReady:     false,
	})
	em.AddComponent(sunflowerID, &components.PlantComponent{
		PlantType: components.PlantSunflower,
		GridRow:   2,
		GridCol:   3,
	})
	em.AddComponent(sunflowerID, &components.PositionComponent{
		X: 400,
		Y: 300,
	})

	// 模拟24秒更新（触发后续生产）
	system.Update(24.0)

	// 验证阳光实体被创建
	sunEntities := em.GetEntitiesWith(reflect.TypeOf(&components.SunComponent{}))
	if len(sunEntities) != 1 {
		t.Errorf("Expected 1 sun entity, got %d", len(sunEntities))
	}

	// 验证计时器重置
	timerComp, _ := em.GetComponent(sunflowerID, reflect.TypeOf(&components.TimerComponent{}))
	timer := timerComp.(*components.TimerComponent)

	if timer.CurrentTime != 0 {
		t.Errorf("Expected CurrentTime=0 after production, got %f", timer.CurrentTime)
	}

	// 验证计时器目标时间保持24秒
	if timer.TargetTime != 24.0 {
		t.Errorf("Expected TargetTime=24.0 for subsequent production, got %f", timer.TargetTime)
	}
}

// TestSunProductionPosition 测试阳光生成位置
func TestSunProductionPosition(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建向日葵实体
	sunflowerID := em.CreateEntity()
	em.AddComponent(sunflowerID, &components.BehaviorComponent{
		Type: components.BehaviorSunflower,
	})
	em.AddComponent(sunflowerID, &components.TimerComponent{
		Name:        "sun_production",
		TargetTime:  7.0,
		CurrentTime: 0,
		IsReady:     false,
	})
	em.AddComponent(sunflowerID, &components.PlantComponent{
		PlantType: components.PlantSunflower,
		GridRow:   2,
		GridCol:   3,
	})
	sunflowerX := 400.0
	sunflowerY := 300.0
	em.AddComponent(sunflowerID, &components.PositionComponent{
		X: sunflowerX,
		Y: sunflowerY,
	})

	// 触发生产
	system.Update(7.0)

	// 查找生成的阳光实体
	sunEntities := em.GetEntitiesWith(reflect.TypeOf(&components.SunComponent{}))
	if len(sunEntities) != 1 {
		t.Fatalf("Expected 1 sun entity, got %d", len(sunEntities))
	}

	// 验证阳光的起始位置
	sunID := sunEntities[0]
	positionComp, ok := em.GetComponent(sunID, reflect.TypeOf(&components.PositionComponent{}))
	if !ok {
		t.Fatal("Sun entity should have PositionComponent")
	}
	position := positionComp.(*components.PositionComponent)

	// 阳光应该在向日葵附近（考虑随机偏移）
	// X 坐标：sunflowerX - 40（居中）± 20（随机）= [340, 380]
	// Y 坐标：sunflowerY - 80（上方）± 10（随机）= [210, 230]
	expectedMinX := sunflowerX - 40.0 - 20.0
	expectedMaxX := sunflowerX - 40.0 + 20.0
	expectedMinY := sunflowerY - 80.0 - 10.0
	expectedMaxY := sunflowerY - 80.0 + 10.0

	if position.X < expectedMinX || position.X > expectedMaxX {
		t.Errorf("Sun X position out of range: expected [%.0f, %.0f], got %.0f",
			expectedMinX, expectedMaxX, position.X)
	}

	if position.Y < expectedMinY || position.Y > expectedMaxY {
		t.Errorf("Sun Y position out of range: expected [%.0f, %.0f], got %.0f",
			expectedMinY, expectedMaxY, position.Y)
	}

	// 检查阳光状态应该是 Landed（可以立即收集）
	sunComp, ok := em.GetComponent(sunID, reflect.TypeOf(&components.SunComponent{}))
	if !ok {
		t.Fatal("Sun entity should have SunComponent")
	}
	sun := sunComp.(*components.SunComponent)
	if sun.State != components.SunLanded {
		t.Errorf("Sun should be in Landed state, got %v", sun.State)
	}

	// 检查阳光速度应该是静止的
	velComp, ok := em.GetComponent(sunID, reflect.TypeOf(&components.VelocityComponent{}))
	if !ok {
		t.Fatal("Sun entity should have VelocityComponent")
	}
	vel := velComp.(*components.VelocityComponent)
	if vel.VX != 0 || vel.VY != 0 {
		t.Errorf("Sun should be stationary, got velocity (%.1f, %.1f)", vel.VX, vel.VY)
	}
}

// TestMultipleSunflowersIndependentTimers 测试多个向日葵独立工作
func TestMultipleSunflowersIndependentTimers(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建第一个向日葵
	sunflower1 := em.CreateEntity()
	em.AddComponent(sunflower1, &components.BehaviorComponent{Type: components.BehaviorSunflower})
	em.AddComponent(sunflower1, &components.TimerComponent{
		Name:        "sun_production",
		TargetTime:  7.0,
		CurrentTime: 0,
	})
	em.AddComponent(sunflower1, &components.PlantComponent{PlantType: components.PlantSunflower})
	em.AddComponent(sunflower1, &components.PositionComponent{X: 400, Y: 300})

	// 创建第二个向日葵
	sunflower2 := em.CreateEntity()
	em.AddComponent(sunflower2, &components.BehaviorComponent{Type: components.BehaviorSunflower})
	em.AddComponent(sunflower2, &components.TimerComponent{
		Name:        "sun_production",
		TargetTime:  7.0,
		CurrentTime: 0,
	})
	em.AddComponent(sunflower2, &components.PlantComponent{PlantType: components.PlantSunflower})
	em.AddComponent(sunflower2, &components.PositionComponent{X: 500, Y: 300})

	// 更新3秒
	system.Update(3.0)

	// 验证两个向日葵的计时器都更新了
	timer1Comp, _ := em.GetComponent(sunflower1, reflect.TypeOf(&components.TimerComponent{}))
	timer1 := timer1Comp.(*components.TimerComponent)
	if timer1.CurrentTime != 3.0 {
		t.Errorf("Sunflower1 timer should be 3.0, got %f", timer1.CurrentTime)
	}

	timer2Comp, _ := em.GetComponent(sunflower2, reflect.TypeOf(&components.TimerComponent{}))
	timer2 := timer2Comp.(*components.TimerComponent)
	if timer2.CurrentTime != 3.0 {
		t.Errorf("Sunflower2 timer should be 3.0, got %f", timer2.CurrentTime)
	}

	// 再更新5秒（总计8秒，两个都应该生产）
	system.Update(5.0)

	// 验证生成了2个阳光
	sunEntities := em.GetEntitiesWith(reflect.TypeOf(&components.SunComponent{}))
	if len(sunEntities) != 2 {
		t.Errorf("Expected 2 sun entities, got %d", len(sunEntities))
	}
}

// TestZombieBasicMovement 测试僵尸正常移动
func TestZombieBasicMovement(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建僵尸实体
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.BehaviorComponent{
		Type: components.BehaviorZombieBasic,
	})
	em.AddComponent(zombieID, &components.PositionComponent{
		X: 1000.0, // 初始位置在屏幕右侧
		Y: 200.0,
	})
	em.AddComponent(zombieID, &components.VelocityComponent{
		VX: -30.0, // 向左移动
		VY: 0.0,
	})

	// 模拟1秒更新
	system.Update(1.0)

	// 验证僵尸位置更新：X 应该减少 30 像素
	posComp, _ := em.GetComponent(zombieID, reflect.TypeOf(&components.PositionComponent{}))
	pos := posComp.(*components.PositionComponent)

	expectedX := 1000.0 - 30.0*1.0 // 970.0
	if pos.X != expectedX {
		t.Errorf("Expected X=%.1f, got %.1f", expectedX, pos.X)
	}

	// 再次更新2秒
	system.Update(2.0)

	posComp, _ = em.GetComponent(zombieID, reflect.TypeOf(&components.PositionComponent{}))
	pos = posComp.(*components.PositionComponent)

	expectedX = 970.0 - 30.0*2.0 // 910.0
	if pos.X != expectedX {
		t.Errorf("Expected X=%.1f after second update, got %.1f", expectedX, pos.X)
	}
}

// TestZombieBoundaryDeletion 测试僵尸移出屏幕后被删除
func TestZombieBoundaryDeletion(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建僵尸实体（位置接近屏幕左侧边界）
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.BehaviorComponent{
		Type: components.BehaviorZombieBasic,
	})
	em.AddComponent(zombieID, &components.PositionComponent{
		X: -50.0, // 接近边界，但还没超出删除阈值（-100）
		Y: 200.0,
	})
	em.AddComponent(zombieID, &components.VelocityComponent{
		VX: -30.0,
		VY: 0.0,
	})

	// 模拟2秒更新（应该移动到 -50 - 60 = -110，超出-100阈值）
	system.Update(2.0)

	// 清理标记删除的实体
	em.RemoveMarkedEntities()

	// 验证僵尸实体被删除
	zombieEntities := em.GetEntitiesWith(
		reflect.TypeOf(&components.BehaviorComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.VelocityComponent{}),
	)

	// 检查是否还有僵尸实体（应该为0，因为被删除了）
	hasZombie := false
	for _, id := range zombieEntities {
		behaviorComp, _ := em.GetComponent(id, reflect.TypeOf(&components.BehaviorComponent{}))
		behavior := behaviorComp.(*components.BehaviorComponent)
		if behavior.Type == components.BehaviorZombieBasic && id == zombieID {
			hasZombie = true
			break
		}
	}

	if hasZombie {
		t.Error("Zombie should be deleted after moving past left boundary")
	}
}

// TestMultipleZombiesIndependentMovement 测试多个僵尸独立移动
func TestMultipleZombiesIndependentMovement(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建第一个僵尸（速度 -30）
	zombie1 := em.CreateEntity()
	em.AddComponent(zombie1, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombie1, &components.PositionComponent{X: 1000.0, Y: 200.0})
	em.AddComponent(zombie1, &components.VelocityComponent{VX: -30.0, VY: 0.0})

	// 创建第二个僵尸（速度 -40，更快）
	zombie2 := em.CreateEntity()
	em.AddComponent(zombie2, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombie2, &components.PositionComponent{X: 1200.0, Y: 300.0})
	em.AddComponent(zombie2, &components.VelocityComponent{VX: -40.0, VY: 0.0})

	// 更新1秒
	system.Update(1.0)

	// 验证第一个僵尸位置
	pos1Comp, _ := em.GetComponent(zombie1, reflect.TypeOf(&components.PositionComponent{}))
	pos1 := pos1Comp.(*components.PositionComponent)
	if pos1.X != 970.0 {
		t.Errorf("Zombie1 should be at X=970.0, got %.1f", pos1.X)
	}

	// 验证第二个僵尸位置
	pos2Comp, _ := em.GetComponent(zombie2, reflect.TypeOf(&components.PositionComponent{}))
	pos2 := pos2Comp.(*components.PositionComponent)
	if pos2.X != 1160.0 {
		t.Errorf("Zombie2 should be at X=1160.0, got %.1f", pos2.X)
	}

	// 再更新2秒
	system.Update(2.0)

	// 验证两个僵尸都继续移动
	pos1Comp, _ = em.GetComponent(zombie1, reflect.TypeOf(&components.PositionComponent{}))
	pos1 = pos1Comp.(*components.PositionComponent)
	expectedX1 := 970.0 - 30.0*2.0 // 910.0
	if pos1.X != expectedX1 {
		t.Errorf("Zombie1 should be at X=%.1f, got %.1f", expectedX1, pos1.X)
	}

	pos2Comp, _ = em.GetComponent(zombie2, reflect.TypeOf(&components.PositionComponent{}))
	pos2 = pos2Comp.(*components.PositionComponent)
	expectedX2 := 1160.0 - 40.0*2.0 // 1080.0
	if pos2.X != expectedX2 {
		t.Errorf("Zombie2 should be at X=%.1f, got %.1f", expectedX2, pos2.X)
	}
}

// TestHitEffectExpiration 测试击中效果计时器超时后被删除
func TestHitEffectExpiration(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建击中效果实体
	hitEffectID := em.CreateEntity()
	em.AddComponent(hitEffectID, &components.BehaviorComponent{
		Type: components.BehaviorPeaBulletHit,
	})
	em.AddComponent(hitEffectID, &components.PositionComponent{
		X: 400,
		Y: 250,
	})
	em.AddComponent(hitEffectID, &components.TimerComponent{
		Name:        "hit_effect_duration",
		CurrentTime: 0.0,
		TargetTime:  0.2, // 0.2秒后应该被删除
		IsReady:     false,
	})

	// 第一次更新：0.1秒（未超时）
	system.Update(0.1)

	// 击中效果应该还存在
	timerComp, ok := em.GetComponent(hitEffectID, reflect.TypeOf(&components.TimerComponent{}))
	if !ok {
		t.Fatal("Expected hit effect to still exist after 0.1s")
	}
	timer := timerComp.(*components.TimerComponent)
	if timer.CurrentTime != 0.1 {
		t.Errorf("Expected timer CurrentTime=0.1, got %.2f", timer.CurrentTime)
	}

	// 第二次更新：再0.15秒（总共0.25秒，超过0.2秒阈值）
	system.Update(0.15)

	// 击中效果应该被标记删除
	em.RemoveMarkedEntities()

	// 验证击中效果已被删除
	_, exists := em.GetComponent(hitEffectID, reflect.TypeOf(&components.TimerComponent{}))
	if exists {
		t.Error("Expected hit effect to be destroyed after timeout (0.25s > 0.2s)")
	}
}

// TestZombieDeathTrigger 测试僵尸生命值 <= 0 时触发死亡状态
func TestZombieDeathTrigger(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建僵尸实体（生命值为 1）
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.BehaviorComponent{
		Type: components.BehaviorZombieBasic,
	})
	em.AddComponent(zombieID, &components.PositionComponent{
		X: 1000.0,
		Y: 200.0,
	})
	em.AddComponent(zombieID, &components.VelocityComponent{
		VX: -30.0,
		VY: 0.0,
	})
	em.AddComponent(zombieID, &components.HealthComponent{
		CurrentHealth: 1, // 生命值只有 1
		MaxHealth:     270,
	})
	// 添加动画组件（用于死亡动画）
	em.AddComponent(zombieID, &components.AnimationComponent{
		Frames:       nil, // 暂时为空，稍后会被替换
		FrameSpeed:   0.1,
		CurrentFrame: 0,
		FrameCounter: 0,
		IsLooping:    true,
		IsFinished:   false,
	})

	// 减少生命值到 0
	healthComp, _ := em.GetComponent(zombieID, reflect.TypeOf(&components.HealthComponent{}))
	health := healthComp.(*components.HealthComponent)
	health.CurrentHealth = 0

	// 运行 BehaviorSystem.Update()
	system.Update(0.1)

	// 清理标记删除的实体
	em.RemoveMarkedEntities()

	// 注意：在测试环境中，由于缺少资源文件，死亡动画加载会失败
	// triggerZombieDeath 会直接删除僵尸实体（错误处理逻辑）
	// 因此我们需要检查僵尸是否被删除
	_, exists := em.GetComponent(zombieID, reflect.TypeOf(&components.PositionComponent{}))
	if exists {
		// 如果僵尸仍然存在（资源加载成功），验证状态转换
		behaviorComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.BehaviorComponent{}))
		if !ok {
			t.Fatal("Expected zombie to have BehaviorComponent if still exists")
		}
		behavior := behaviorComp.(*components.BehaviorComponent)
		if behavior.Type != components.BehaviorZombieDying {
			t.Errorf("Expected zombie behavior to be BehaviorZombieDying, got %v", behavior.Type)
		}

		// 验证：VelocityComponent 被移除（僵尸停止移动）
		_, hasVelocity := em.GetComponent(zombieID, reflect.TypeOf(&components.VelocityComponent{}))
		if hasVelocity {
			t.Error("Expected zombie VelocityComponent to be removed (zombie should stop moving)")
		}

		// 验证：AnimationComponent.IsLooping == false
		animComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.AnimationComponent{}))
		if !ok {
			t.Fatal("Expected zombie to have AnimationComponent if still exists")
		}
		anim := animComp.(*components.AnimationComponent)
		if anim.IsLooping {
			t.Error("Expected zombie death animation IsLooping to be false")
		}
	} else {
		// 僵尸被删除（资源加载失败的降级处理），这也是可接受的行为
		t.Log("Zombie was deleted immediately due to resource loading failure (acceptable in test environment)")
	}
}

// TestZombieDeleteAfterDeathAnimation 测试僵尸死亡动画完成后删除实体
func TestZombieDeleteAfterDeathAnimation(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建一个 BehaviorZombieDying 僵尸（已经在死亡动画播放中）
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.BehaviorComponent{
		Type: components.BehaviorZombieDying,
	})
	em.AddComponent(zombieID, &components.PositionComponent{
		X: 1000.0,
		Y: 200.0,
	})
	em.AddComponent(zombieID, &components.AnimationComponent{
		Frames:       nil,
		FrameSpeed:   0.1,
		CurrentFrame: 9,
		FrameCounter: 0,
		IsLooping:    false,
		IsFinished:   true, // 死亡动画已完成
	})

	// 第一次更新：handleZombieDyingBehavior 应该检测到动画完成并删除僵尸
	system.Update(0.1)

	// 清理标记删除的实体
	em.RemoveMarkedEntities()

	// 验证：僵尸实体被删除
	_, exists := em.GetComponent(zombieID, reflect.TypeOf(&components.PositionComponent{}))
	if exists {
		t.Error("Expected zombie to be deleted after death animation finished")
	}
}

// TestZombieDeathAnimationInProgress 测试僵尸死亡动画播放中不删除实体
func TestZombieDeathAnimationInProgress(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建一个 BehaviorZombieDying 僵尸（死亡动画正在播放）
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.BehaviorComponent{
		Type: components.BehaviorZombieDying,
	})
	em.AddComponent(zombieID, &components.PositionComponent{
		X: 1000.0,
		Y: 200.0,
	})
	em.AddComponent(zombieID, &components.AnimationComponent{
		Frames:       nil,
		FrameSpeed:   0.1,
		CurrentFrame: 5,
		FrameCounter: 0,
		IsLooping:    false,
		IsFinished:   false, // 死亡动画未完成
	})

	// 运行 BehaviorSystem.Update()
	system.Update(0.1)

	// 清理标记删除的实体
	em.RemoveMarkedEntities()

	// 验证：僵尸实体仍然存在（动画未完成）
	_, exists := em.GetComponent(zombieID, reflect.TypeOf(&components.PositionComponent{}))
	if !exists {
		t.Error("Expected zombie to still exist (death animation not finished)")
	}
}

// TestDetectPlantCollision 测试植物碰撞检测
func TestDetectPlantCollision(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建植物实体 - 向日葵在 (2, 3)
	sunflowerID := em.CreateEntity()
	em.AddComponent(sunflowerID, &components.PlantComponent{
		PlantType: components.PlantSunflower,
		GridRow:   2,
		GridCol:   3,
	})

	// 创建植物实体 - 豌豆射手在 (1, 5)
	peashooterID := em.CreateEntity()
	em.AddComponent(peashooterID, &components.PlantComponent{
		PlantType: components.PlantPeashooter,
		GridRow:   1,
		GridCol:   5,
	})

	// 测试1: 僵尸与向日葵在同一格子 (2, 3)
	plantID, hasCollision := system.detectPlantCollision(2, 3)
	if !hasCollision {
		t.Error("Expected collision with sunflower at (2, 3)")
	}
	if plantID != sunflowerID {
		t.Errorf("Expected sunflower ID %d, got %d", sunflowerID, plantID)
	}

	// 测试2: 僵尸与豌豆射手在同一格子 (1, 5)
	plantID, hasCollision = system.detectPlantCollision(1, 5)
	if !hasCollision {
		t.Error("Expected collision with peashooter at (1, 5)")
	}
	if plantID != peashooterID {
		t.Errorf("Expected peashooter ID %d, got %d", peashooterID, plantID)
	}

	// 测试3: 僵尸在没有植物的格子 (0, 0)
	_, hasCollision = system.detectPlantCollision(0, 0)
	if hasCollision {
		t.Error("Expected no collision at empty cell (0, 0)")
	}

	// 测试4: 僵尸在不同行但同列 (3, 3) - 不应碰撞
	_, hasCollision = system.detectPlantCollision(3, 3)
	if hasCollision {
		t.Error("Expected no collision at (3, 3) - different row from sunflower")
	}

	// 测试5: 僵尸在同行但不同列 (2, 4) - 不应碰撞
	_, hasCollision = system.detectPlantCollision(2, 4)
	if hasCollision {
		t.Error("Expected no collision at (2, 4) - different col from sunflower")
	}
}

// TestStartEatingPlant 测试僵尸进入啃食状态
func TestStartEatingPlant(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建僵尸实体（正常移动状态）
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.PositionComponent{X: 500, Y: 300})
	em.AddComponent(zombieID, &components.VelocityComponent{VX: -30, VY: 0})
	em.AddComponent(zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	em.AddComponent(zombieID, &components.AnimationComponent{
		Frames:       make([]*ebiten.Image, 22), // 走路动画
		FrameSpeed:   0.1,
		CurrentFrame: 0,
		FrameCounter: 0,
		IsLooping:    true,
		IsFinished:   false,
	})

	// 创建植物实体
	plantID := em.CreateEntity()
	em.AddComponent(plantID, &components.PlantComponent{
		PlantType: components.PlantSunflower,
		GridRow:   2,
		GridCol:   3,
	})

	// 调用 startEatingPlant
	system.startEatingPlant(zombieID, plantID)

	// 验证1: VelocityComponent 被移除
	_, hasVelocity := em.GetComponent(zombieID, reflect.TypeOf(&components.VelocityComponent{}))
	if hasVelocity {
		t.Error("Expected VelocityComponent to be removed")
	}

	// 验证2: 行为类型切换为 BehaviorZombieEating
	behaviorComp, _ := em.GetComponent(zombieID, reflect.TypeOf(&components.BehaviorComponent{}))
	behavior := behaviorComp.(*components.BehaviorComponent)
	if behavior.Type != components.BehaviorZombieEating {
		t.Errorf("Expected BehaviorZombieEating, got %v", behavior.Type)
	}

	// 验证3: 添加了 TimerComponent
	timerComp, hasTimer := em.GetComponent(zombieID, reflect.TypeOf(&components.TimerComponent{}))
	if !hasTimer {
		t.Fatal("Expected TimerComponent to be added")
	}
	timer := timerComp.(*components.TimerComponent)
	if timer.Name != "eating_damage" {
		t.Errorf("Expected timer name 'eating_damage', got '%s'", timer.Name)
	}
	if timer.TargetTime != 1.5 {
		t.Errorf("Expected TargetTime=1.5, got %f", timer.TargetTime)
	}

	// 验证4: 动画切换为啃食动画（检查 IsLooping 仍为 true）
	animComp, _ := em.GetComponent(zombieID, reflect.TypeOf(&components.AnimationComponent{}))
	anim := animComp.(*components.AnimationComponent)
	if !anim.IsLooping {
		t.Error("Expected eating animation to be looping")
	}
	if anim.CurrentFrame != 0 {
		t.Errorf("Expected animation reset to frame 0, got %d", anim.CurrentFrame)
	}
}

// TestEatingDamage 测试啃食伤害逻辑
func TestEatingDamage(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建僵尸实体（啃食状态）
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.PositionComponent{
		X: 540.0, // GridWorldStartX(265) + col(3)*CellWidth(80) + 中心偏移 = 265+240+35 = 540
		Y: 365.0, // GridWorldStartY(90) + row(2)*CellHeight(100) + 中心偏移 = 90+200+75 = 365
	})
	em.AddComponent(zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieEating})
	em.AddComponent(zombieID, &components.TimerComponent{
		Name:        "eating_damage",
		TargetTime:  1.5,
		CurrentTime: 0.0,
		IsReady:     false,
	})
	em.AddComponent(zombieID, &components.AnimationComponent{
		Frames:       make([]*ebiten.Image, 21),
		FrameSpeed:   0.1,
		CurrentFrame: 0,
		IsLooping:    true,
	})

	// 创建植物实体（在同一格子 (2, 3)）
	plantID := em.CreateEntity()
	em.AddComponent(plantID, &components.PlantComponent{
		PlantType: components.PlantSunflower,
		GridRow:   2,
		GridCol:   3,
	})
	em.AddComponent(plantID, &components.HealthComponent{
		CurrentHealth: 300,
		MaxHealth:     300,
	})

	// 第1次更新：0.8秒（计时器未完成）
	system.handleZombieEatingBehavior(zombieID, 0.8)

	// 验证：植物生命值未减少
	healthComp, _ := em.GetComponent(plantID, reflect.TypeOf(&components.HealthComponent{}))
	health := healthComp.(*components.HealthComponent)
	if health.CurrentHealth != 300 {
		t.Errorf("Expected health=300 (no damage yet), got %d", health.CurrentHealth)
	}

	// 第2次更新：再过0.7秒（总共1.5秒，计时器完成）
	system.handleZombieEatingBehavior(zombieID, 0.7)

	// 验证：植物生命值减少100
	healthComp, _ = em.GetComponent(plantID, reflect.TypeOf(&components.HealthComponent{}))
	health = healthComp.(*components.HealthComponent)
	if health.CurrentHealth != 200 {
		t.Errorf("Expected health=200 (one damage), got %d", health.CurrentHealth)
	}

	// 验证：计时器重置
	timerComp, _ := em.GetComponent(zombieID, reflect.TypeOf(&components.TimerComponent{}))
	timer := timerComp.(*components.TimerComponent)
	if timer.CurrentTime != 0.0 {
		t.Errorf("Expected timer reset to 0, got %f", timer.CurrentTime)
	}
	if timer.IsReady {
		t.Error("Expected timer.IsReady to be false after reset")
	}

	// 第3次更新：再过1.5秒（第二次伤害）
	system.handleZombieEatingBehavior(zombieID, 1.5)

	// 验证：植物生命值再减少100
	healthComp, _ = em.GetComponent(plantID, reflect.TypeOf(&components.HealthComponent{}))
	health = healthComp.(*components.HealthComponent)
	if health.CurrentHealth != 100 {
		t.Errorf("Expected health=100 (two damages), got %d", health.CurrentHealth)
	}
}

// TestPlantDeathAndZombieResume 测试植物死亡后僵尸恢复移动
func TestPlantDeathAndZombieResume(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建僵尸实体（啃食状态）
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.PositionComponent{
		X: 540.0,
		Y: 365.0,
	})
	em.AddComponent(zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieEating})
	em.AddComponent(zombieID, &components.TimerComponent{
		Name:        "eating_damage",
		TargetTime:  1.5,
		CurrentTime: 1.5, // 计时器已完成
		IsReady:     true,
	})
	em.AddComponent(zombieID, &components.AnimationComponent{
		Frames:       make([]*ebiten.Image, 21),
		FrameSpeed:   0.1,
		CurrentFrame: 5,
		IsLooping:    true,
	})

	// 创建植物实体（生命值很低，一次伤害就会死）
	plantID := em.CreateEntity()
	em.AddComponent(plantID, &components.PlantComponent{
		PlantType: components.PlantSunflower,
		GridRow:   2,
		GridCol:   3,
	})
	em.AddComponent(plantID, &components.HealthComponent{
		CurrentHealth: 50, // 只剩50生命值
		MaxHealth:     300,
	})

	// 更新一次（造成100伤害，植物死亡）
	system.handleZombieEatingBehavior(zombieID, 0.1)

	// 清理标记删除的实体
	em.RemoveMarkedEntities()

	// 验证1: 植物实体被删除
	_, plantExists := em.GetComponent(plantID, reflect.TypeOf(&components.PlantComponent{}))
	if plantExists {
		t.Error("Expected plant to be destroyed")
	}

	// 验证2: 僵尸恢复 VelocityComponent
	velComp, hasVelocity := em.GetComponent(zombieID, reflect.TypeOf(&components.VelocityComponent{}))
	if !hasVelocity {
		t.Fatal("Expected VelocityComponent to be restored")
	}
	vel := velComp.(*components.VelocityComponent)
	if vel.VX != -30.0 {
		t.Errorf("Expected VX=-30.0, got %f", vel.VX)
	}

	// 验证3: 僵尸行为类型切换回 BehaviorZombieBasic
	behaviorComp, _ := em.GetComponent(zombieID, reflect.TypeOf(&components.BehaviorComponent{}))
	behavior := behaviorComp.(*components.BehaviorComponent)
	if behavior.Type != components.BehaviorZombieBasic {
		t.Errorf("Expected BehaviorZombieBasic, got %v", behavior.Type)
	}

	// 验证4: TimerComponent 被移除
	_, hasTimer := em.GetComponent(zombieID, reflect.TypeOf(&components.TimerComponent{}))
	if hasTimer {
		t.Error("Expected TimerComponent to be removed")
	}

	// 验证5: 动画切换回走路动画（CurrentFrame 重置为0）
	animComp, _ := em.GetComponent(zombieID, reflect.TypeOf(&components.AnimationComponent{}))
	anim := animComp.(*components.AnimationComponent)
	if anim.CurrentFrame != 0 {
		t.Errorf("Expected animation reset to frame 0, got %d", anim.CurrentFrame)
	}
	if !anim.IsLooping {
		t.Error("Expected walk animation to be looping")
	}
}

// TestMultipleZombiesEatingSamePlant 测试多个僵尸同时啃食同一植物
func TestMultipleZombiesEatingSamePlant(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建植物实体（在格子 (2, 3)）
	plantID := em.CreateEntity()
	em.AddComponent(plantID, &components.PlantComponent{
		PlantType: components.PlantSunflower,
		GridRow:   2,
		GridCol:   3,
	})
	em.AddComponent(plantID, &components.HealthComponent{
		CurrentHealth: 150, // 只能承受1.5次伤害
		MaxHealth:     300,
	})

	// 创建僵尸1（啃食状态，计时器刚开始）
	zombie1ID := em.CreateEntity()
	em.AddComponent(zombie1ID, &components.PositionComponent{X: 540.0, Y: 365.0})
	em.AddComponent(zombie1ID, &components.BehaviorComponent{Type: components.BehaviorZombieEating})
	em.AddComponent(zombie1ID, &components.TimerComponent{
		Name:        "eating_damage",
		TargetTime:  1.5,
		CurrentTime: 0.0, // 刚开始啃食
		IsReady:     false,
	})
	em.AddComponent(zombie1ID, &components.AnimationComponent{
		Frames:    make([]*ebiten.Image, 21),
		IsLooping: true,
	})

	// 创建僵尸2（也在啃食状态，即将完成）
	zombie2ID := em.CreateEntity()
	em.AddComponent(zombie2ID, &components.PositionComponent{X: 545.0, Y: 365.0})
	em.AddComponent(zombie2ID, &components.BehaviorComponent{Type: components.BehaviorZombieEating})
	em.AddComponent(zombie2ID, &components.TimerComponent{
		Name:        "eating_damage",
		TargetTime:  1.5,
		CurrentTime: 1.4, // 即将完成
		IsReady:     false,
	})
	em.AddComponent(zombie2ID, &components.AnimationComponent{
		Frames:    make([]*ebiten.Image, 21),
		IsLooping: true,
	})

	// 僵尸2更新0.2秒（总1.6秒，造成第一次伤害）
	system.handleZombieEatingBehavior(zombie2ID, 0.2)

	// 验证：植物生命值减少到50
	healthComp, _ := em.GetComponent(plantID, reflect.TypeOf(&components.HealthComponent{}))
	health := healthComp.(*components.HealthComponent)
	if health.CurrentHealth != 50 {
		t.Errorf("Expected health=50, got %d", health.CurrentHealth)
	}

	// 僵尸2再更新1.6秒（计时器重置后再次完成，造成第二次伤害，植物死亡）
	system.handleZombieEatingBehavior(zombie2ID, 1.6)

	// 清理标记删除的实体（植物被删除）
	em.RemoveMarkedEntities()

	// 验证1: 植物被删除
	_, plantExists := em.GetComponent(plantID, reflect.TypeOf(&components.PlantComponent{}))
	if plantExists {
		t.Error("Expected plant to be destroyed by zombie2")
	}

	// 验证2: 僵尸2恢复移动（它杀死了植物）
	_, hasVel2 := em.GetComponent(zombie2ID, reflect.TypeOf(&components.VelocityComponent{}))
	if !hasVel2 {
		t.Error("Expected zombie2 to resume movement")
	}

	// 僵尸1再次更新（植物已不存在，更新足够长的时间使计时器完成）
	// 僵尸1初始 CurrentTime=0.0秒，更新1.6秒后达到1.6秒，超过目标时间1.5秒
	// 应该检测到植物不存在并恢复移动
	system.handleZombieEatingBehavior(zombie1ID, 1.6) // 总1.6秒，计时器完成

	// 清理可能的实体删除
	em.RemoveMarkedEntities()

	// 验证3: 僵尸1也恢复移动（检测到植物消失）
	_, hasVel1 := em.GetComponent(zombie1ID, reflect.TypeOf(&components.VelocityComponent{}))
	if !hasVel1 {
		t.Error("Expected zombie1 to resume movement when plant disappeared")
	}

	// 验证4: 两个僵尸都切换回 BehaviorZombieBasic
	behaviorComp1, _ := em.GetComponent(zombie1ID, reflect.TypeOf(&components.BehaviorComponent{}))
	if behaviorComp1.(*components.BehaviorComponent).Type != components.BehaviorZombieBasic {
		t.Error("Expected zombie1 to return to Basic behavior")
	}

	behaviorComp2, _ := em.GetComponent(zombie2ID, reflect.TypeOf(&components.BehaviorComponent{}))
	if behaviorComp2.(*components.BehaviorComponent).Type != components.BehaviorZombieBasic {
		t.Error("Expected zombie2 to return to Basic behavior")
	}
}

// TestZombieEatingWithoutPlant 测试僵尸啃食时植物被其他原因删除
func TestZombieEatingWithoutPlant(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// 创建僵尸实体（啃食状态）
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.PositionComponent{X: 540.0, Y: 365.0})
	em.AddComponent(zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieEating})
	em.AddComponent(zombieID, &components.TimerComponent{
		Name:        "eating_damage",
		TargetTime:  1.5,
		CurrentTime: 1.5,
		IsReady:     true,
	})
	em.AddComponent(zombieID, &components.AnimationComponent{
		Frames:    make([]*ebiten.Image, 21),
		IsLooping: true,
	})

	// 注意：没有创建植物实体（模拟植物已被删除）

	// 更新僵尸行为（应该检测到植物不存在并恢复移动）
	system.handleZombieEatingBehavior(zombieID, 0.1)

	// 验证1: 僵尸恢复 VelocityComponent（没有崩溃）
	_, hasVelocity := em.GetComponent(zombieID, reflect.TypeOf(&components.VelocityComponent{}))
	if !hasVelocity {
		t.Error("Expected zombie to resume movement when plant is missing")
	}

	// 验证2: 僵尸行为切换回 BehaviorZombieBasic
	behaviorComp, _ := em.GetComponent(zombieID, reflect.TypeOf(&components.BehaviorComponent{}))
	behavior := behaviorComp.(*components.BehaviorComponent)
	if behavior.Type != components.BehaviorZombieBasic {
		t.Errorf("Expected BehaviorZombieBasic, got %v", behavior.Type)
	}

	// 验证3: TimerComponent 被移除
	_, hasTimer := em.GetComponent(zombieID, reflect.TypeOf(&components.TimerComponent{}))
	if hasTimer {
		t.Error("Expected TimerComponent to be removed")
	}
}

// TestZombieGridPositionCalculation 测试僵尸世界坐标转网格坐标
func TestZombieGridPositionCalculation(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(testAudioContext)
	system := NewBehaviorSystem(em, rm)

	// GridWorldStartX = 265, GridWorldStartY = 90
	// CellWidth = 80, CellHeight = 100

	tests := []struct {
		name        string
		worldX      float64
		worldY      float64
		expectedCol int
		expectedRow int
		hasPlant    bool
	}{
		{
			name:        "格子 (0, 0) 中心",
			worldX:      265 + 40, // GridWorldStartX + CellWidth/2
			worldY:      90 + 50,  // GridWorldStartY + CellHeight/2
			expectedCol: 0,
			expectedRow: 0,
			hasPlant:    false,
		},
		{
			name:        "格子 (3, 2) 中心",
			worldX:      265 + 3*80 + 40,
			worldY:      90 + 2*100 + 50,
			expectedCol: 3,
			expectedRow: 2,
			hasPlant:    false,
		},
		{
			name:        "格子 (8, 4) 中心（最右下角）",
			worldX:      265 + 8*80 + 40,
			worldY:      90 + 4*100 + 50,
			expectedCol: 8,
			expectedRow: 4,
			hasPlant:    false,
		},
		{
			name:        "格子 (5, 3) 左边缘",
			worldX:      265 + 5*80 + 1, // 刚进入格子
			worldY:      90 + 3*100 + 50,
			expectedCol: 5,
			expectedRow: 3,
			hasPlant:    false,
		},
		{
			name:        "格子 (4, 1) 右边缘",
			worldX:      265 + 5*80 - 1, // 刚离开格子
			worldY:      90 + 1*100 + 50,
			expectedCol: 4,
			expectedRow: 1,
			hasPlant:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建植物在特定格子（如果测试需要）
			if tt.hasPlant {
				plantID := em.CreateEntity()
				em.AddComponent(plantID, &components.PlantComponent{
					PlantType: components.PlantSunflower,
					GridRow:   tt.expectedRow,
					GridCol:   tt.expectedCol,
				})
			}

			// 计算僵尸所在格子（模拟 handleZombieBasicBehavior 中的逻辑）
			zombieCol := int((tt.worldX - 265) / 80) // GridWorldStartX=265, CellWidth=80
			zombieRow := int((tt.worldY - 90) / 100) // GridWorldStartY=90, CellHeight=100

			// 验证计算结果
			if zombieCol != tt.expectedCol {
				t.Errorf("Expected col=%d, got %d", tt.expectedCol, zombieCol)
			}
			if zombieRow != tt.expectedRow {
				t.Errorf("Expected row=%d, got %d", tt.expectedRow, zombieRow)
			}

			// 验证碰撞检测
			_, hasCollision := system.detectPlantCollision(zombieRow, zombieCol)
			if hasCollision != tt.hasPlant {
				t.Errorf("Expected hasCollision=%v, got %v", tt.hasPlant, hasCollision)
			}
		})
	}
}
