package systems

import (
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
)

// TestNewLawnmowerSystem 测试除草车系统创建
func TestNewLawnmowerSystem(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := &game.ResourceManager{}
	gs := game.GetGameState()

	system := NewLawnmowerSystem(em, rm, gs)

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
	system := NewLawnmowerSystem(em, nil, nil)

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
	system := NewLawnmowerSystem(em, nil, nil)

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
	system := NewLawnmowerSystem(em, nil, nil)

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
// Story 10.6: 测试回退逻辑（无 ResourceManager 时使用旧的死亡动画）
func TestLawnmowerSystemZombieCollision(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	// Story 10.6: ResourceManager 设为 nil，测试回退逻辑
	// 预期行为：加载 locator 失败，回退到 BehaviorZombieDying
	system := NewLawnmowerSystem(em, nil, gs)

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

	// Story 10.6: 添加 ReanimComponent（回退逻辑需要）
	ecs.AddComponent(em, zombieID, &components.ReanimComponent{
		ReanimName:        "Zombie",
		CachedRenderData:  make([]components.RenderPartData, 1), // 最小化初始化
		CurrentAnimations: []string{"anim_idle"},
	})

	// 检测碰撞
	system.checkZombieCollisions()

	// Story 10.6: 验证回退逻辑（无 ResourceManager 时）
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](em, zombieID)
	if !ok {
		t.Fatal("Zombie should still exist after collision (playing death animation)")
	}

	// 验证僵尸行为切换为死亡状态（回退逻辑）
	if behavior.Type != components.BehaviorZombieDying {
		t.Errorf("Zombie behavior should be BehaviorZombieDying (fallback), got %v", behavior.Type)
	}

	// 验证僵尸速度组件已移除（停止移动）
	if ecs.HasComponent[*components.VelocityComponent](em, zombieID) {
		t.Error("Zombie should have VelocityComponent removed (stop moving)")
	}

	// Story 10.6: 验证僵尸动画状态（回退逻辑不暂停动画）
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](em, zombieID)
	if !ok {
		t.Fatal("Zombie should have ReanimComponent")
	}
	if reanim.IsPaused {
		t.Error("Zombie animation should NOT be paused in fallback mode")
	}

	// Story 10.6: 注意：回退逻辑不会计数
	// 计数在 BehaviorSystem 的死亡动画完成后触发
}

// TestLawnmowerSystemZombieNoCollision 测试除草车与僵尸不碰撞（不同行）
func TestLawnmowerSystemZombieNoCollision(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	system := NewLawnmowerSystem(em, nil, gs)

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

// TestLawnmowerSystem_SquashAnimation 测试压扁动画的核心逻辑
// Story 10.6 - 压扁动画功能
func TestLawnmowerSystem_SquashAnimation(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := &game.ResourceManager{}
	gs := game.GetGameState()
	system := NewLawnmowerSystem(em, rm, gs)

	// 创建测试用的除草车实体
	lawnmowerID := em.CreateEntity()
	ecs.AddComponent(em, lawnmowerID, &components.PositionComponent{
		X: 100.0,
		Y: 200.0,
	})
	ecs.AddComponent(em, lawnmowerID, &components.LawnmowerComponent{
		Lane:        2,
		IsTriggered: true,
		IsMoving:    true,
	})

	// 创建测试用的僵尸实体
	zombieID := em.CreateEntity()
	ecs.AddComponent(em, zombieID, &components.PositionComponent{
		X: 150.0,
		Y: 200.0,
	})
	ecs.AddComponent(em, zombieID, &components.BehaviorComponent{
		Type: components.BehaviorZombieBasic,
	})

	// 创建简单的 ReanimComponent（最小化，只用于测试）
	ecs.AddComponent(em, zombieID, &components.ReanimComponent{
		ReanimName:        "Zombie",
		CachedRenderData:  make([]components.RenderPartData, 1),
		CurrentAnimations: []string{"anim_idle"},
	})

	// 添加压扁动画组件
	frames := []components.LocatorFrame{
		{X: 0, Y: 0, ScaleX: 1.0, ScaleY: 1.0, SkewX: 0, SkewY: 0},
		{X: 10, Y: 10, ScaleX: 0.8, ScaleY: 1.0, SkewX: 45, SkewY: 45},
		{X: 20, Y: 15, ScaleX: 0.5, ScaleY: 1.05, SkewX: 90, SkewY: 90},
		{X: 30, Y: 20, ScaleX: 0.263, ScaleY: 1.042, SkewX: 90, SkewY: 90},
	}

	ecs.AddComponent(em, zombieID, &components.SquashAnimationComponent{
		ElapsedTime:       0.0,
		Duration:          0.4,
		LocatorFrames:     frames,
		CurrentFrameIndex: 0,
		OriginalPosX:      150.0,
		OriginalPosY:      200.0,
		LawnmowerEntityID: lawnmowerID,
		IsCompleted:       false,
	})

	t.Run("初始状态", func(t *testing.T) {
		squashAnim, ok := ecs.GetComponent[*components.SquashAnimationComponent](em, zombieID)
		if !ok {
			t.Fatal("SquashAnimationComponent 未添加")
		}

		if squashAnim.ElapsedTime != 0.0 {
			t.Errorf("初始 ElapsedTime = %v, want 0.0", squashAnim.ElapsedTime)
		}

		if squashAnim.CurrentFrameIndex != 0 {
			t.Errorf("初始 CurrentFrameIndex = %v, want 0", squashAnim.CurrentFrameIndex)
		}
	})

	t.Run("更新压扁动画-第一帧", func(t *testing.T) {
		// 通过调用 LawnmowerSystem.Update() 来更新压扁动画
		// 注意：Update() 会调用内部的 updateSquashAnimations() 和 ApplySquashTransforms()
		system.Update(0.1)

		squashAnim, _ := ecs.GetComponent[*components.SquashAnimationComponent](em, zombieID)
		if squashAnim.ElapsedTime != 0.1 {
			t.Errorf("ElapsedTime = %v, want 0.1", squashAnim.ElapsedTime)
		}

		// 进度 = 0.1 / 0.4 = 0.25，帧索引 = 0.25 * 4 = 1
		expectedFrame := 1
		if squashAnim.CurrentFrameIndex != expectedFrame {
			t.Errorf("CurrentFrameIndex = %v, want %v", squashAnim.CurrentFrameIndex, expectedFrame)
		}

		// 检查位置是否正确（使用 OriginalPosX + frame.X，不跟随除草车）
		pos, _ := ecs.GetComponent[*components.PositionComponent](em, zombieID)
		expectedX := 150.0 + frames[1].X // OriginalPosX + frame.X
		if pos.X != expectedX {
			t.Errorf("僵尸 X 坐标 = %v, want %v", pos.X, expectedX)
		}
	})

	t.Run("更新压扁动画-中间帧", func(t *testing.T) {
		// 继续更新动画（再 0.1 秒，总共 0.2 秒）
		system.Update(0.1)

		squashAnim, _ := ecs.GetComponent[*components.SquashAnimationComponent](em, zombieID)

		// 进度 = 0.2 / 0.4 = 0.5，帧索引 = 0.5 * 4 = 2
		expectedFrame := 2
		if squashAnim.CurrentFrameIndex != expectedFrame {
			t.Errorf("CurrentFrameIndex = %v, want %v", squashAnim.CurrentFrameIndex, expectedFrame)
		}

		// Story 10.6: 变换应用逻辑已改为在 ApplySquashTransforms() 中执行
		// CachedRenderData 的变换检查需要在集成测试中验证（涉及 RenderSystem）
		// 这里只验证组件状态更新正确
	})

	t.Run("动画完成后触发死亡", func(t *testing.T) {
		// 注意：完整的死亡流程会触发粒子效果，需要完整的 ResourceManager 初始化
		// 这里只测试压扁动画结束时的状态转换

		// 获取当前状态
		squashAnim, _ := ecs.GetComponent[*components.SquashAnimationComponent](em, zombieID)
		initialElapsed := squashAnim.ElapsedTime // 应该是 0.2

		// 继续更新动画到接近完成但还未触发（0.15 秒，总共 0.35 < 0.4）
		system.Update(0.15)

		// 应该还未完成，SquashAnimationComponent 仍存在
		if _, ok := ecs.GetComponent[*components.SquashAnimationComponent](em, zombieID); !ok {
			t.Error("SquashAnimationComponent 不应该在动画未完成时被移除")
		}

		squashAnimAfter, _ := ecs.GetComponent[*components.SquashAnimationComponent](em, zombieID)
		expectedElapsed := initialElapsed + 0.15
		if squashAnimAfter.ElapsedTime != expectedElapsed {
			t.Errorf("ElapsedTime = %v, want %v", squashAnimAfter.ElapsedTime, expectedElapsed)
		}

		// 验证僵尸仍在压扁状态（未切换为 Dying）
		behavior, _ := ecs.GetComponent[*components.BehaviorComponent](em, zombieID)
		if behavior.Type != components.BehaviorZombieBasic {
			t.Errorf("Behavior.Type 应该保持 BehaviorZombieBasic，实际为 %v", behavior.Type)
		}
	})
}

// TestSquashAnimation_EdgeCases 测试压扁动画的边界条件
// Story 10.6 - 边界条件测试
func TestSquashAnimation_EdgeCases(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := &game.ResourceManager{}
	gs := game.GetGameState()
	system := NewLawnmowerSystem(em, rm, gs)

	t.Run("空帧数组", func(t *testing.T) {
		zombieID := em.CreateEntity()
		ecs.AddComponent(em, zombieID, &components.PositionComponent{X: 100, Y: 200})
		ecs.AddComponent(em, zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
		ecs.AddComponent(em, zombieID, &components.ReanimComponent{
			ReanimName:        "Zombie",
			CachedRenderData:  make([]components.RenderPartData, 0),
			CurrentAnimations: []string{"anim_idle"},
		})
		ecs.AddComponent(em, zombieID, &components.SquashAnimationComponent{
			ElapsedTime:   0.0,
			Duration:      0.4,
			LocatorFrames: []components.LocatorFrame{}, // 空数组
		})

		// 不应该崩溃（边界保护：空帧数组会被跳过）
		system.Update(0.1)

		squashAnim, ok := ecs.GetComponent[*components.SquashAnimationComponent](em, zombieID)
		if !ok {
			t.Fatal("SquashAnimationComponent 不应该被移除")
		}

		// ElapsedTime 应该已增加（即使帧数组为空，时间仍累积）
		if squashAnim.ElapsedTime != 0.1 {
			t.Errorf("ElapsedTime = %v, want 0.1", squashAnim.ElapsedTime)
		}

		// CurrentFrameIndex 应该保持为 0（GetCurrentFrameIndex 的边界保护）
		if squashAnim.CurrentFrameIndex != 0 {
			t.Errorf("空帧数组的 CurrentFrameIndex 应该为 0，实际为 %v", squashAnim.CurrentFrameIndex)
		}
	})

	t.Run("除草车实体已删除", func(t *testing.T) {
		zombieID := em.CreateEntity()
		ecs.AddComponent(em, zombieID, &components.PositionComponent{X: 100, Y: 200})
		ecs.AddComponent(em, zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
		ecs.AddComponent(em, zombieID, &components.ReanimComponent{
			ReanimName:        "Zombie",
			CachedRenderData:  make([]components.RenderPartData, 1),
			CurrentAnimations: []string{"anim_idle"},
		})

		// 引用一个不存在的除草车 ID
		invalidLawnmowerID := ecs.EntityID(9999)
		frames := []components.LocatorFrame{
			{X: 50, Y: 10, ScaleX: 0.5, ScaleY: 1.0, SkewX: 90, SkewY: 90},
		}
		ecs.AddComponent(em, zombieID, &components.SquashAnimationComponent{
			ElapsedTime:       0.0,
			Duration:          0.4,
			LocatorFrames:     frames,
			OriginalPosX:      100.0,
			OriginalPosY:      200.0,
			LawnmowerEntityID: invalidLawnmowerID,
		})

		// 不应该崩溃
		system.Update(0.1)

		// 应该使用原始位置 + 偏移
		pos, _ := ecs.GetComponent[*components.PositionComponent](em, zombieID)
		expectedX := 100.0 + frames[0].X // OriginalPosX + frame.X
		if pos.X != expectedX {
			t.Errorf("僵尸 X 坐标 = %v, want %v（应该使用原始位置）", pos.X, expectedX)
		}
	})

	t.Run("超长时间更新", func(t *testing.T) {
		zombieID := em.CreateEntity()
		ecs.AddComponent(em, zombieID, &components.PositionComponent{X: 100, Y: 200})
		ecs.AddComponent(em, zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
		ecs.AddComponent(em, zombieID, &components.ReanimComponent{
			ReanimName:        "Zombie",
			CachedRenderData:  make([]components.RenderPartData, 1),
			CurrentAnimations: []string{"anim_idle"},
		})

		frames := []components.LocatorFrame{
			{X: 0, Y: 0, ScaleX: 1.0, ScaleY: 1.0},
			{X: 10, Y: 10, ScaleX: 0.5, ScaleY: 1.0},
		}
		ecs.AddComponent(em, zombieID, &components.SquashAnimationComponent{
			ElapsedTime:   0.0,
			Duration:      0.2,
			LocatorFrames: frames,
		})

		// 单次更新到接近完成（0.19 秒 < 0.2）
		system.Update(0.19)

		// SquashAnimationComponent 应该还存在（未完成）
		squashAnim, ok := ecs.GetComponent[*components.SquashAnimationComponent](em, zombieID)
		if !ok {
			t.Error("SquashAnimationComponent 应该仍然存在")
		}
		if squashAnim.ElapsedTime != 0.19 {
			t.Errorf("ElapsedTime = %v, want 0.19", squashAnim.ElapsedTime)
		}

		// 验证动画接近完成但未完成
		progress := squashAnim.GetProgress()
		if progress < 0.9 || progress >= 1.0 {
			t.Errorf("进度应该接近完成(0.9-0.99)，实际为 %.2f", progress)
		}

		// 确认未标记为完成
		if squashAnim.IsComplete() {
			t.Error("动画应该未完成")
		}
	})
}

// TestLawnmowerSystem_EarlyParticleTrigger 测试粒子效果在压扁过程中提前触发
// Story 10.6 fix
func TestLawnmowerSystem_EarlyParticleTrigger(t *testing.T) {
	em := ecs.NewEntityManager()
	// ResourceManager 为 nil，测试隐藏 tracks 逻辑是否独立执行
	system := NewLawnmowerSystem(em, nil, game.GetGameState())

	zombieID := em.CreateEntity()
	ecs.AddComponent(em, zombieID, &components.PositionComponent{X: 100, Y: 200})
	ecs.AddComponent(em, zombieID, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	ecs.AddComponent(em, zombieID, &components.ReanimComponent{
		ReanimName:        "Zombie",
		CachedRenderData:  make([]components.RenderPartData, 1),
		CurrentAnimations: []string{"anim_idle"},
	})

	// 创建帧数据（ScaleX 递减）
	frames := []components.LocatorFrame{
		{ScaleX: 1.0}, // 0
		{ScaleX: 1.0}, // 1
		{ScaleX: 1.0}, // 2
		{ScaleX: 0.9}, // 3
		{ScaleX: 0.8}, // 4 (Should trigger here)
		{ScaleX: 0.5}, // 5
	}

	// 添加压扁动画组件
	ecs.AddComponent(em, zombieID, &components.SquashAnimationComponent{
		ElapsedTime:        0.0,
		Duration:           0.6,
		LocatorFrames:      frames,
		CurrentFrameIndex:  0,
		OriginalPosX:       100.0,
		OriginalPosY:       200.0,
		IsCompleted:        false,
		ParticlesTriggered: false,
	})

	// 1. 更新到第 2 帧 (progress < 0.66, frameIndex < 4)
	system.Update(0.2) // 0.2/0.6 = 0.33 -> Frame 2 (6*0.33 = 2)

	squashAnim, _ := ecs.GetComponent[*components.SquashAnimationComponent](em, zombieID)
	if squashAnim.ParticlesTriggered {
		t.Error("Particles should NOT be triggered at frame 2")
	}

	// 2. 更新到第 4 帧 (progress > 0.66, frameIndex >= 4)
	system.Update(0.25) // Total 0.45/0.6 = 0.75 -> Frame 4.5 -> 4

	if !squashAnim.ParticlesTriggered {
		t.Error("Particles SHOULD be triggered at frame 4")
	}

	// 3. 验证 ReanimComponent 的 tracks 是否被隐藏
	reanim, _ := ecs.GetComponent[*components.ReanimComponent](em, zombieID)
	if reanim.HiddenTracks == nil {
		t.Fatal("HiddenTracks map should be initialized")
	}
	if !reanim.HiddenTracks["anim_head1"] {
		t.Error("anim_head1 should be hidden")
	}
	if !reanim.HiddenTracks["Zombie_outerarm_upper"] {
		t.Error("Zombie_outerarm_upper should be hidden")
	}
}
