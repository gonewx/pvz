package systems

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestOpeningAnimationSystem_NewOpeningAnimationSystem 测试系统创建和初始化。
func TestOpeningAnimationSystem_NewOpeningAnimationSystem(t *testing.T) {
	tests := []struct {
		name          string
		skipOpening   bool
		openingType   string
		specialRules  string
		shouldBeNil   bool
		expectedState string
	}{
		{
			name:          "标准关卡创建开场动画系统",
			skipOpening:   false,
			openingType:   "standard",
			specialRules:  "",
			shouldBeNil:   false,
			expectedState: "idle",
		},
		{
			name:        "SkipOpening=true不创建系统",
			skipOpening: true,
			openingType: "standard",
			shouldBeNil: true,
		},
		{
			name:          "教学关卡也创建开场动画系统",
			skipOpening:   false,
			openingType:   "tutorial",
			shouldBeNil:   false,
			expectedState: "idle",
		},
		{
			name:         "特殊规则关卡不创建系统",
			skipOpening:  false,
			openingType:  "standard",
			specialRules: "bowling",
			shouldBeNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试依赖
			em := ecs.NewEntityManager()
			gs := game.GetGameState()
			rm := game.NewResourceManager(nil)
			levelConfig := &config.LevelConfig{
				SkipOpening:  tt.skipOpening,
				OpeningType:  tt.openingType,
				SpecialRules: tt.specialRules,
				Waves: []config.WaveConfig{
					{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 3}}},
				},
			}

			// 创建 CameraSystem 和 ReanimSystem
			cameraSystem := NewCameraSystem(em, gs)

			// 创建 OpeningAnimationSystem
			system := NewOpeningAnimationSystem(em, gs, rm, levelConfig, cameraSystem)

			// 验证系统是否按预期创建
			if tt.shouldBeNil {
				if system != nil {
					t.Errorf("预期系统为 nil，但实际创建了系统")
				}
			} else {
				if system == nil {
					t.Fatalf("预期创建系统，但系统为 nil")
				}

				// 验证初始状态
				openingComp, ok := ecs.GetComponent[*components.OpeningAnimationComponent](em, system.openingEntity)
				if !ok {
					t.Fatalf("未找到 OpeningAnimationComponent")
				}

				if openingComp.State != tt.expectedState {
					t.Errorf("初始状态不匹配：期望 %s，实际 %s", tt.expectedState, openingComp.State)
				}

				if openingComp.IsCompleted {
					t.Errorf("初始状态不应标记为已完成")
				}

				if openingComp.IsSkipped {
					t.Errorf("初始状态不应标记为已跳过")
				}

				if len(openingComp.ZombieEntities) != 0 {
					t.Errorf("初始僵尸实体列表应为空")
				}
			}
		})
	}
}

// TestOpeningAnimationSystem_StateMachine 测试状态机流转。
// Story 8.3.1: 恢复预览僵尸生成逻辑
func TestOpeningAnimationSystem_StateMachine(t *testing.T) {
	// 创建测试依赖
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)
	levelConfig := &config.LevelConfig{
		SkipOpening:  false,
		OpeningType:  "standard",
		SpecialRules: "",
		Waves: []config.WaveConfig{
			{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 3}}},
		},
	}

	cameraSystem := NewCameraSystem(em, gs)
	system := NewOpeningAnimationSystem(em, gs, rm, levelConfig, cameraSystem)

	if system == nil {
		t.Fatal("系统创建失败")
	}

	// 验证初始状态：idle
	openingComp, _ := ecs.GetComponent[*components.OpeningAnimationComponent](em, system.openingEntity)
	if openingComp.State != "idle" {
		t.Errorf("初始状态应为 idle，实际为 %s", openingComp.State)
	}

	// 模拟 idle 状态持续 0.5 秒
	system.Update(0.5)
	openingComp, _ = ecs.GetComponent[*components.OpeningAnimationComponent](em, system.openingEntity)
	if openingComp.State != "cameraMoveRight" {
		t.Errorf("0.5秒后应切换到 cameraMoveRight，实际为 %s", openingComp.State)
	}

	// 验证镜头是否开始移动
	if !cameraSystem.IsAnimating() {
		t.Errorf("镜头应开始移动")
	}

	// 模拟镜头移动完成（等待镜头到达目标）
	// 镜头速度 300px/s，距离 800px，需要约 2.67 秒
	for i := 0; i < 30; i++ {
		system.Update(0.1)
		cameraSystem.Update(0.1)
		if !cameraSystem.IsAnimating() {
			break
		}
	}

	// 再更新一帧，触发状态切换
	system.Update(0.1)
	openingComp, _ = ecs.GetComponent[*components.OpeningAnimationComponent](em, system.openingEntity)
	if openingComp.State != "showZombies" {
		t.Errorf("镜头完成后应切换到 showZombies，实际为 %s", openingComp.State)
	}

	// Story 8.3.1: 恢复预览僵尸生成，ZombieEntities 应有实体
	// 注意：测试环境下没有 Reanim 资源，所以僵尸实体会创建但没有动画组件
	// 1波关卡（简单）= 3只预览僵尸
	expectedPreviewCount := 3
	if len(openingComp.ZombieEntities) != expectedPreviewCount {
		t.Errorf("showZombies 状态应生成 %d 个预览僵尸，实际 %d 个", expectedPreviewCount, len(openingComp.ZombieEntities))
	}

	// 模拟展示僵尸 2 秒
	system.Update(2.0)
	openingComp, _ = ecs.GetComponent[*components.OpeningAnimationComponent](em, system.openingEntity)
	if openingComp.State != "cameraMoveLeft" {
		t.Errorf("2秒后应切换到 cameraMoveLeft，实际为 %s", openingComp.State)
	}

	// 验证镜头是否开始返回
	if !cameraSystem.IsAnimating() {
		t.Errorf("镜头应开始返回")
	}

	// 模拟镜头返回完成
	for i := 0; i < 30; i++ {
		system.Update(0.1)
		cameraSystem.Update(0.1)
		if !cameraSystem.IsAnimating() {
			break
		}
	}

	// 再更新一帧，触发状态切换
	system.Update(0.1)
	openingComp, _ = ecs.GetComponent[*components.OpeningAnimationComponent](em, system.openingEntity)
	if openingComp.State != "gameStart" {
		t.Errorf("镜头返回后应切换到 gameStart，实际为 %s", openingComp.State)
	}

	// 再次更新以执行 gameStart 状态的逻辑（清理僵尸、标记完成）
	system.Update(0.1)
	openingComp, _ = ecs.GetComponent[*components.OpeningAnimationComponent](em, system.openingEntity)

	// 验证是否已完成
	if !openingComp.IsCompleted {
		t.Errorf("gameStart 状态应标记为已完成")
	}

	// Story 8.3.1: gameStart 状态会清理所有预览僵尸
	if len(openingComp.ZombieEntities) != 0 {
		t.Errorf("gameStart 后 ZombieEntities 应为空（已清理），剩余 %d 个", len(openingComp.ZombieEntities))
	}
}

// TestOpeningAnimationSystem_SpawnPreviewZombies 测试预览僵尸生成。
// Story 8.3.1: 预览僵尸是独立的展示实体
func TestOpeningAnimationSystem_SpawnPreviewZombies(t *testing.T) {
	tests := []struct {
		name          string
		waves         []config.WaveConfig
		expectedCount int // 期望的预览僵尸数量
	}{
		{
			name: "简单关卡(1波)生成3只预览僵尸",
			waves: []config.WaveConfig{
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 3}}},
			},
			expectedCount: 3,
		},
		{
			name: "简单关卡(2波)生成3只预览僵尸",
			waves: []config.WaveConfig{
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 3}}},
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 2}}},
			},
			expectedCount: 3,
		},
		{
			name: "中等关卡(3波)生成5只预览僵尸",
			waves: []config.WaveConfig{
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 3}}},
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 2}}},
				{OldZombies: []config.ZombieSpawn{{Type: "conehead", Lane: 1}}},
			},
			expectedCount: 5,
		},
		{
			name: "中等关卡(5波)生成5只预览僵尸",
			waves: []config.WaveConfig{
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 3}}},
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 2}}},
				{OldZombies: []config.ZombieSpawn{{Type: "conehead", Lane: 1}}},
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 4}}},
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 5}}},
			},
			expectedCount: 5,
		},
		{
			name: "困难关卡(6波)生成8只预览僵尸",
			waves: []config.WaveConfig{
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 3}}},
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 2}}},
				{OldZombies: []config.ZombieSpawn{{Type: "conehead", Lane: 1}}},
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 4}}},
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 5}}},
				{OldZombies: []config.ZombieSpawn{{Type: "buckethead", Lane: 3}}},
			},
			expectedCount: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			em := ecs.NewEntityManager()
			gs := game.GetGameState()
			rm := game.NewResourceManager(nil)
			levelConfig := &config.LevelConfig{
				SkipOpening:  false,
				OpeningType:  "standard",
				SpecialRules: "",
				Waves:        tt.waves,
			}

			cameraSystem := NewCameraSystem(em, gs)
			system := NewOpeningAnimationSystem(em, gs, rm, levelConfig, cameraSystem)

			if system == nil {
				t.Fatal("系统创建失败")
			}

			// 推进到 showZombies 状态触发预览僵尸生成
			system.Update(0.5) // idle → cameraMoveRight
			for i := 0; i < 30; i++ {
				system.Update(0.1)
				cameraSystem.Update(0.1)
				if !cameraSystem.IsAnimating() {
					break
				}
			}
			system.Update(0.1) // cameraMoveRight → showZombies

			openingComp, _ := ecs.GetComponent[*components.OpeningAnimationComponent](em, system.openingEntity)
			if openingComp.State != "showZombies" {
				t.Fatalf("未能到达 showZombies 状态，当前状态：%s", openingComp.State)
			}

			// 验证预览僵尸数量
			if len(openingComp.ZombieEntities) != tt.expectedCount {
				t.Errorf("预览僵尸数量不匹配：期望 %d，实际 %d", tt.expectedCount, len(openingComp.ZombieEntities))
			}

			// 验证每个预览僵尸实体都有正确的组件
			for i, entityID := range openingComp.ZombieEntities {
				// 检查位置组件
				posComp, ok := ecs.GetComponent[*components.PositionComponent](em, entityID)
				if !ok {
					t.Errorf("预览僵尸 %d 缺少 PositionComponent", i)
					continue
				}

				// 验证位置在正确范围内 (X: 1050-1250)
				if posComp.X < config.ZombieSpawnMinX || posComp.X > config.ZombieSpawnMaxX {
					t.Errorf("预览僵尸 %d 的X坐标 %.0f 超出范围 [%.0f, %.0f]", i, posComp.X, config.ZombieSpawnMinX, config.ZombieSpawnMaxX)
				}

				// 检查行为组件
				behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](em, entityID)
				if !ok {
					t.Errorf("预览僵尸 %d 缺少 BehaviorComponent", i)
					continue
				}

				// 验证使用预览行为类型
				if behaviorComp.Type != components.BehaviorZombiePreview {
					t.Errorf("预览僵尸 %d 的行为类型应为 BehaviorZombiePreview，实际为 %v", i, behaviorComp.Type)
				}
			}
		})
	}
}

// TestOpeningAnimationSystem_ClearPreviewZombies 测试预览僵尸清理。
// Story 8.3.1: 镜头返回后销毁所有预览僵尸实体
func TestOpeningAnimationSystem_ClearPreviewZombies(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)
	levelConfig := &config.LevelConfig{
		SkipOpening:  false,
		OpeningType:  "standard",
		SpecialRules: "",
		Waves: []config.WaveConfig{
			{OldZombies: []config.ZombieSpawn{
				{Type: "basic", Lane: 3},
				{Type: "conehead", Lane: 2},
			}},
		},
	}

	cameraSystem := NewCameraSystem(em, gs)
	system := NewOpeningAnimationSystem(em, gs, rm, levelConfig, cameraSystem)

	if system == nil {
		t.Fatal("系统创建失败")
	}

	// 推进到 showZombies 状态
	system.Update(0.5) // idle → cameraMoveRight
	for i := 0; i < 30; i++ {
		system.Update(0.1)
		cameraSystem.Update(0.1)
		if !cameraSystem.IsAnimating() {
			break
		}
	}
	system.Update(0.1) // cameraMoveRight → showZombies

	openingComp, _ := ecs.GetComponent[*components.OpeningAnimationComponent](em, system.openingEntity)
	if openingComp.State != "showZombies" {
		t.Fatalf("未能到达 showZombies 状态，当前状态：%s", openingComp.State)
	}

	// 记录生成的僵尸实体ID
	zombieEntitiesBefore := make([]ecs.EntityID, len(openingComp.ZombieEntities))
	copy(zombieEntitiesBefore, openingComp.ZombieEntities)

	if len(zombieEntitiesBefore) == 0 {
		t.Fatal("应该生成预览僵尸实体")
	}

	// 继续推进到 gameStart 状态（会清理僵尸）
	system.Update(2.0) // showZombies → cameraMoveLeft
	for i := 0; i < 30; i++ {
		system.Update(0.1)
		cameraSystem.Update(0.1)
		if !cameraSystem.IsAnimating() {
			break
		}
	}
	system.Update(0.1) // cameraMoveLeft → gameStart
	system.Update(0.1) // 执行 gameStart 逻辑（清理僵尸）

	openingComp, _ = ecs.GetComponent[*components.OpeningAnimationComponent](em, system.openingEntity)

	// 验证 ZombieEntities 列表已清空
	if len(openingComp.ZombieEntities) != 0 {
		t.Errorf("ZombieEntities 列表应为空，实际有 %d 个", len(openingComp.ZombieEntities))
	}

	// 验证实体已从 EntityManager 中销毁
	// 注意：DestroyEntity 只是标记实体待删除，需要调用 RemoveMarkedEntities 才会真正删除
	// 在测试中，我们验证组件已被移除（因为 clearPreviewZombies 会销毁实体）
	em.RemoveMarkedEntities()
	for _, entityID := range zombieEntitiesBefore {
		_, hasPos := ecs.GetComponent[*components.PositionComponent](em, entityID)
		if hasPos {
			t.Errorf("僵尸实体 %d 应已销毁（仍有 PositionComponent）", entityID)
		}
	}
}

// TestOpeningAnimationSystem_SkipClearsZombies 测试跳过时清理预览僵尸。
// Story 8.3.1: 跳过开场动画时（ESC/Space），预览僵尸也被正确清理
func TestOpeningAnimationSystem_SkipClearsZombies(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)
	levelConfig := &config.LevelConfig{
		SkipOpening:  false,
		OpeningType:  "standard",
		SpecialRules: "",
		Waves: []config.WaveConfig{
			{OldZombies: []config.ZombieSpawn{
				{Type: "basic", Lane: 3},
				{Type: "conehead", Lane: 2},
			}},
		},
	}

	cameraSystem := NewCameraSystem(em, gs)
	system := NewOpeningAnimationSystem(em, gs, rm, levelConfig, cameraSystem)

	if system == nil {
		t.Fatal("系统创建失败")
	}

	// 推进到 showZombies 状态
	system.Update(0.5) // idle → cameraMoveRight
	for i := 0; i < 30; i++ {
		system.Update(0.1)
		cameraSystem.Update(0.1)
		if !cameraSystem.IsAnimating() {
			break
		}
	}
	system.Update(0.1) // cameraMoveRight → showZombies

	openingComp, _ := ecs.GetComponent[*components.OpeningAnimationComponent](em, system.openingEntity)
	if openingComp.State != "showZombies" {
		t.Fatalf("未能到达 showZombies 状态，当前状态：%s", openingComp.State)
	}

	// 记录生成的僵尸实体ID
	zombieEntitiesBefore := make([]ecs.EntityID, len(openingComp.ZombieEntities))
	copy(zombieEntitiesBefore, openingComp.ZombieEntities)

	if len(zombieEntitiesBefore) == 0 {
		t.Fatal("应该生成预览僵尸实体")
	}

	// 调用 Skip 方法
	system.Skip()

	// 验证状态
	openingComp, _ = ecs.GetComponent[*components.OpeningAnimationComponent](em, system.openingEntity)
	if openingComp.State != "gameStart" {
		t.Errorf("Skip 后应切换到 gameStart，实际为 %s", openingComp.State)
	}

	if !openingComp.IsSkipped {
		t.Errorf("IsSkipped 应为 true")
	}

	if !openingComp.IsCompleted {
		t.Errorf("IsCompleted 应为 true")
	}

	// Story 8.3.1: Skip 应清理所有预览僵尸
	if len(openingComp.ZombieEntities) != 0 {
		t.Errorf("Skip 后 ZombieEntities 应为空，实际有 %d 个", len(openingComp.ZombieEntities))
	}

	// 验证实体已从 EntityManager 中销毁
	// 注意：DestroyEntity 只是标记实体待删除，需要调用 RemoveMarkedEntities 才会真正删除
	em.RemoveMarkedEntities()
	for _, entityID := range zombieEntitiesBefore {
		_, hasPos := ecs.GetComponent[*components.PositionComponent](em, entityID)
		if hasPos {
			t.Errorf("僵尸实体 %d 应已销毁（仍有 PositionComponent）", entityID)
		}
	}

	// 验证镜头动画是否停止
	if cameraSystem.IsAnimating() {
		t.Errorf("Skip 应停止镜头动画")
	}
}

// TestOpeningAnimationSystem_PreviewZombieCountConfig 测试配置的预览僵尸数量。
// Story 8.3.1: 如果配置了 PreviewZombieCount 则使用配置值
func TestOpeningAnimationSystem_PreviewZombieCountConfig(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)
	levelConfig := &config.LevelConfig{
		SkipOpening:        false,
		OpeningType:        "standard",
		SpecialRules:       "",
		PreviewZombieCount: 10, // 配置值优先
		Waves: []config.WaveConfig{
			{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 3}}}, // 1波
		},
	}

	cameraSystem := NewCameraSystem(em, gs)
	system := NewOpeningAnimationSystem(em, gs, rm, levelConfig, cameraSystem)

	if system == nil {
		t.Fatal("系统创建失败")
	}

	// 推进到 showZombies 状态
	system.Update(0.5) // idle → cameraMoveRight
	for i := 0; i < 30; i++ {
		system.Update(0.1)
		cameraSystem.Update(0.1)
		if !cameraSystem.IsAnimating() {
			break
		}
	}
	system.Update(0.1) // cameraMoveRight → showZombies

	openingComp, _ := ecs.GetComponent[*components.OpeningAnimationComponent](em, system.openingEntity)

	// 验证使用配置值而非自动计算（1波关卡自动计算应该是3只）
	expectedCount := 10
	if len(openingComp.ZombieEntities) != expectedCount {
		t.Errorf("应使用配置的预览僵尸数量 %d，实际 %d", expectedCount, len(openingComp.ZombieEntities))
	}
}

// TestOpeningAnimationSystem_IsCompleted 测试完成状态查询。
func TestOpeningAnimationSystem_IsCompleted(t *testing.T) {
	// 测试 nil 系统
	var nilSystem *OpeningAnimationSystem
	if !nilSystem.IsCompleted() {
		t.Errorf("nil 系统应返回 true")
	}

	// 测试正常系统
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)
	levelConfig := &config.LevelConfig{
		SkipOpening:  false,
		OpeningType:  "standard",
		SpecialRules: "",
		Waves: []config.WaveConfig{
			{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 3}}},
		},
	}

	cameraSystem := NewCameraSystem(em, gs)
	system := NewOpeningAnimationSystem(em, gs, rm, levelConfig, cameraSystem)

	if system == nil {
		t.Fatal("系统创建失败")
	}

	// 初始状态应为未完成
	if system.IsCompleted() {
		t.Errorf("初始状态应为未完成")
	}

	// 调用 Skip 方法
	system.Skip()

	// 跳过后应为已完成
	if !system.IsCompleted() {
		t.Errorf("Skip 后应为已完成")
	}
}

// TestOpeningAnimationSystem_GetUniqueZombieTypes 测试僵尸类型去重逻辑。
// Story 8.3.1: 支持新格式 ZombieGroup 和旧格式 ZombieSpawn
func TestOpeningAnimationSystem_GetUniqueZombieTypes(t *testing.T) {
	tests := []struct {
		name          string
		waves         []config.WaveConfig
		expectedTypes []string
	}{
		{
			name: "旧格式单一类型",
			waves: []config.WaveConfig{
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 3}}},
			},
			expectedTypes: []string{"basic"},
		},
		{
			name: "旧格式多类型去重",
			waves: []config.WaveConfig{
				{OldZombies: []config.ZombieSpawn{
					{Type: "basic", Lane: 3},
					{Type: "basic", Lane: 2},
					{Type: "conehead", Lane: 1},
				}},
			},
			expectedTypes: []string{"basic", "conehead"},
		},
		{
			name: "旧格式跨波次去重",
			waves: []config.WaveConfig{
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 3}}},
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 2}}},
				{OldZombies: []config.ZombieSpawn{{Type: "conehead", Lane: 1}}},
			},
			expectedTypes: []string{"basic", "conehead"},
		},
		{
			name: "新格式ZombieGroup",
			waves: []config.WaveConfig{
				{Zombies: []config.ZombieGroup{
					{Type: "basic", Lanes: []int{2, 3, 4}, Count: 3},
					{Type: "conehead", Lanes: []int{3}, Count: 1},
				}},
			},
			expectedTypes: []string{"basic", "conehead"},
		},
		{
			name: "混合格式（新旧兼容）",
			waves: []config.WaveConfig{
				{
					Zombies:    []config.ZombieGroup{{Type: "basic", Lanes: []int{3}, Count: 1}},
					OldZombies: []config.ZombieSpawn{{Type: "conehead", Lane: 2}},
				},
				{
					OldZombies: []config.ZombieSpawn{{Type: "buckethead", Lane: 1}},
				},
			},
			expectedTypes: []string{"basic", "conehead", "buckethead"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			em := ecs.NewEntityManager()
			gs := game.GetGameState()
			rm := game.NewResourceManager(nil)
			levelConfig := &config.LevelConfig{
				SkipOpening:  false,
				OpeningType:  "standard",
				SpecialRules: "",
				Waves:        tt.waves,
			}

			cameraSystem := NewCameraSystem(em, gs)
			system := NewOpeningAnimationSystem(em, gs, rm, levelConfig, cameraSystem)

			if system == nil {
				t.Fatal("系统创建失败")
			}

			// 调用 getUniqueZombieTypes
			result := system.getUniqueZombieTypes()

			// 验证数量
			if len(result) != len(tt.expectedTypes) {
				t.Errorf("僵尸类型数量不匹配：期望 %d，实际 %d (结果: %v)", len(tt.expectedTypes), len(result), result)
			}

			// 验证每个类型都存在（不关心顺序）
			typeSet := make(map[string]bool)
			for _, typ := range result {
				typeSet[typ] = true
			}

			for _, expectedType := range tt.expectedTypes {
				if !typeSet[expectedType] {
					t.Errorf("缺少僵尸类型：%s (结果: %v)", expectedType, result)
				}
			}
		})
	}
}

// TestOpeningAnimationSystem_CalculatePreviewZombieCount 测试预览僵尸数量计算。
func TestOpeningAnimationSystem_CalculatePreviewZombieCount(t *testing.T) {
	tests := []struct {
		name               string
		waveCount          int
		previewZombieCount int // 配置值
		expectedCount      int
	}{
		{"配置值优先", 1, 7, 7},
		{"简单关卡1波", 1, 0, 3},
		{"简单关卡2波", 2, 0, 3},
		{"中等关卡3波", 3, 0, 5},
		{"中等关卡5波", 5, 0, 5},
		{"困难关卡6波", 6, 0, 8},
		{"困难关卡10波", 10, 0, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			em := ecs.NewEntityManager()
			gs := game.GetGameState()
			rm := game.NewResourceManager(nil)

			// 生成指定数量的波次
			waves := make([]config.WaveConfig, tt.waveCount)
			for i := 0; i < tt.waveCount; i++ {
				waves[i] = config.WaveConfig{
					OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 3}},
				}
			}

			levelConfig := &config.LevelConfig{
				SkipOpening:        false,
				OpeningType:        "standard",
				SpecialRules:       "",
				Waves:              waves,
				PreviewZombieCount: tt.previewZombieCount,
			}

			cameraSystem := NewCameraSystem(em, gs)
			system := NewOpeningAnimationSystem(em, gs, rm, levelConfig, cameraSystem)

			if system == nil {
				t.Fatal("系统创建失败")
			}

			// 调用 calculatePreviewZombieCount
			result := system.calculatePreviewZombieCount()

			if result != tt.expectedCount {
				t.Errorf("预览僵尸数量计算错误：期望 %d，实际 %d", tt.expectedCount, result)
			}
		})
	}
}
