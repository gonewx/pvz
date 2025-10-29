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
			name:        "教学关卡不创建系统",
			skipOpening: false,
			openingType: "tutorial",
			shouldBeNil: true,
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
			reanimSystem := NewReanimSystem(em)

			// 创建 OpeningAnimationSystem
			system := NewOpeningAnimationSystem(em, gs, rm, levelConfig, cameraSystem, reanimSystem)

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
	reanimSystem := NewReanimSystem(em)
	system := NewOpeningAnimationSystem(em, gs, rm, levelConfig, cameraSystem, reanimSystem)

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

	// Story 8.3: 不再生成预览僵尸，直接使用 WaveSpawnSystem 预生成的关卡僵尸
	// 因此 ZombieEntities 列表应为空
	if len(openingComp.ZombieEntities) != 0 {
		t.Errorf("不应生成预告僵尸实体（Story 8.3 改用关卡僵尸），实际生成 %d 个", len(openingComp.ZombieEntities))
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

	// Story 8.3: 不再生成预览僵尸，ZombieEntities 始终为空
	if len(openingComp.ZombieEntities) != 0 {
		t.Errorf("ZombieEntities 应始终为空（Story 8.3 改用关卡僵尸），剩余 %d 个", len(openingComp.ZombieEntities))
	}
}

// TestOpeningAnimationSystem_Skip 测试跳过功能。
func TestOpeningAnimationSystem_Skip(t *testing.T) {
	// 创建测试依赖
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
	reanimSystem := NewReanimSystem(em)
	system := NewOpeningAnimationSystem(em, gs, rm, levelConfig, cameraSystem, reanimSystem)

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

	// Story 8.3: 不再生成预览僵尸
	if len(openingComp.ZombieEntities) != 0 {
		t.Errorf("不应生成预告僵尸（Story 8.3 改用关卡僵尸），实际生成 %d 个", len(openingComp.ZombieEntities))
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

	// Story 8.3: ZombieEntities 始终为空
	if len(openingComp.ZombieEntities) != 0 {
		t.Errorf("ZombieEntities 应始终为空（Story 8.3 改用关卡僵尸），剩余 %d 个", len(openingComp.ZombieEntities))
	}

	// 验证镜头动画是否停止
	if cameraSystem.IsAnimating() {
		t.Errorf("Skip 应停止镜头动画")
	}
}

// TestOpeningAnimationSystem_ZombiePreview 已废弃。
// Story 8.3: 不再生成预览僵尸，直接使用 WaveSpawnSystem 预生成的关卡僵尸。
// 开场动画期间僵尸保持静止（IsActivated=false），动画结束后激活。
// 保留此注释以说明功能变更。

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
	reanimSystem := NewReanimSystem(em)
	system := NewOpeningAnimationSystem(em, gs, rm, levelConfig, cameraSystem, reanimSystem)

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
func TestOpeningAnimationSystem_GetUniqueZombieTypes(t *testing.T) {
	tests := []struct {
		name          string
		waves         []config.WaveConfig
		expectedTypes []string
	}{
		{
			name: "单一类型",
			waves: []config.WaveConfig{
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 3}}},
			},
			expectedTypes: []string{"basic"},
		},
		{
			name: "多类型去重",
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
			name: "跨波次去重",
			waves: []config.WaveConfig{
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 3}}},
				{OldZombies: []config.ZombieSpawn{{Type: "basic", Lane: 2}}},
				{OldZombies: []config.ZombieSpawn{{Type: "conehead", Lane: 1}}},
			},
			expectedTypes: []string{"basic", "conehead"},
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
			reanimSystem := NewReanimSystem(em)
			system := NewOpeningAnimationSystem(em, gs, rm, levelConfig, cameraSystem, reanimSystem)

			if system == nil {
				t.Fatal("系统创建失败")
			}

			// 调用 getUniqueZombieTypes
			result := system.getUniqueZombieTypes()

			// 验证数量
			if len(result) != len(tt.expectedTypes) {
				t.Errorf("僵尸类型数量不匹配：期望 %d，实际 %d", len(tt.expectedTypes), len(result))
			}

			// 验证每个类型都存在（不关心顺序）
			typeSet := make(map[string]bool)
			for _, typ := range result {
				typeSet[typ] = true
			}

			for _, expectedType := range tt.expectedTypes {
				if !typeSet[expectedType] {
					t.Errorf("缺少僵尸类型：%s", expectedType)
				}
			}
		})
	}
}
