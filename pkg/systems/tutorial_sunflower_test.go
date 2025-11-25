package systems

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// 创建用于测试的 TutorialSystem
func createTestTutorialSystemForSunflower(tutorialSteps []config.TutorialStep) (*TutorialSystem, *ecs.EntityManager, *game.GameState) {
	em := ecs.NewEntityManager()
	gs := &game.GameState{}

	levelConfig := &config.LevelConfig{
		ID:            "1-2",
		TutorialSteps: tutorialSteps,
	}

	// 创建 mock LawnGridSystem
	lgs := &LawnGridSystem{entityManager: em}

	// 创建 mock SunSpawnSystem
	sss := &SunSpawnSystem{entityManager: em}

	// 创建 mock WaveSpawnSystem
	wss := &WaveSpawnSystem{entityManager: em, gameState: gs}

	ts := NewTutorialSystem(em, gs, nil, lgs, sss, wss, levelConfig)

	// 模拟铺草皮完成
	ts.OnSoddingComplete()

	return ts, em, gs
}

// TestSunflowerTriggerConditions 测试向日葵教学触发器
func TestSunflowerTriggerConditions(t *testing.T) {
	tests := []struct {
		name           string
		trigger        string
		sunflowerCount int
		timeElapsed    float64
		expected       bool
	}{
		{
			name:           "sunflowerCount1OrTimeout - 1颗向日葵触发",
			trigger:        "sunflowerCount1OrTimeout",
			sunflowerCount: 1,
			timeElapsed:    0,
			expected:       true,
		},
		{
			name:           "sunflowerCount1OrTimeout - 0颗但10秒超时触发",
			trigger:        "sunflowerCount1OrTimeout",
			sunflowerCount: 0,
			timeElapsed:    10.0,
			expected:       true,
		},
		{
			name:           "sunflowerCount1OrTimeout - 0颗9秒不触发",
			trigger:        "sunflowerCount1OrTimeout",
			sunflowerCount: 0,
			timeElapsed:    9.0,
			expected:       false,
		},
		{
			name:           "sunflowerCount2OrTimeout - 2颗向日葵触发",
			trigger:        "sunflowerCount2OrTimeout",
			sunflowerCount: 2,
			timeElapsed:    0,
			expected:       true,
		},
		{
			name:           "sunflowerCount2OrTimeout - 1颗但10秒超时触发",
			trigger:        "sunflowerCount2OrTimeout",
			sunflowerCount: 1,
			timeElapsed:    10.0,
			expected:       true,
		},
		{
			name:           "sunflowerCount2OrTimeout - 1颗5秒不触发",
			trigger:        "sunflowerCount2OrTimeout",
			sunflowerCount: 1,
			timeElapsed:    5.0,
			expected:       false,
		},
		{
			name:           "sunflowerReminder - 20秒后仍不足3颗触发",
			trigger:        "sunflowerReminder",
			sunflowerCount: 2,
			timeElapsed:    20.0,
			expected:       true,
		},
		{
			name:           "sunflowerReminder - 20秒后已有3颗也触发（用于跳过步骤）",
			trigger:        "sunflowerReminder",
			sunflowerCount: 3,
			timeElapsed:    20.0,
			expected:       true, // 新行为：也触发，但跳过文本显示
		},
		{
			name:           "sunflowerReminder - 15秒不足3颗不触发",
			trigger:        "sunflowerReminder",
			sunflowerCount: 2,
			timeElapsed:    15.0,
			expected:       false,
		},
		{
			name:           "sunflowerReminder - 5秒已有3颗也触发（快速种植跳过）",
			trigger:        "sunflowerReminder",
			sunflowerCount: 3,
			timeElapsed:    5.0,
			expected:       true, // 新行为：时间未到但已有3颗，也触发以跳过步骤
		},
		{
			name:           "sunflowerCount3 - 3颗向日葵触发",
			trigger:        "sunflowerCount3",
			sunflowerCount: 3,
			timeElapsed:    0,
			expected:       true,
		},
		{
			name:           "sunflowerCount3 - 2颗向日葵不触发",
			trigger:        "sunflowerCount3",
			sunflowerCount: 2,
			timeElapsed:    0,
			expected:       false,
		},
		{
			name:           "sunflowerCount3 - 5颗向日葵也触发",
			trigger:        "sunflowerCount3",
			sunflowerCount: 5,
			timeElapsed:    0,
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tutorialSteps := []config.TutorialStep{
				{Trigger: tt.trigger, TextKey: "TEST", Action: "sunflowerHint"},
			}
			ts, _, _ := createTestTutorialSystemForSunflower(tutorialSteps)

			// 设置测试状态
			ts.sunflowerCount = tt.sunflowerCount
			ts.stepTimeElapsed = tt.timeElapsed

			result := ts.checkTriggerCondition(tt.trigger)
			if result != tt.expected {
				t.Errorf("checkTriggerCondition(%q) = %v, expected %v (sunflowerCount=%d, timeElapsed=%.1f)",
					tt.trigger, result, tt.expected, tt.sunflowerCount, tt.timeElapsed)
			}
		})
	}
}

// TestFindSunflowerCard 测试查找向日葵卡片
func TestFindSunflowerCard(t *testing.T) {
	tutorialSteps := []config.TutorialStep{
		{Trigger: "gameStart", TextKey: "TEST", Action: "sunflowerHint"},
	}
	ts, em, _ := createTestTutorialSystemForSunflower(tutorialSteps)

	// 测试没有卡片时返回0
	cardID := ts.findSunflowerCard()
	if cardID != 0 {
		t.Errorf("findSunflowerCard() should return 0 when no cards exist, got %d", cardID)
	}

	// 创建豌豆射手卡片
	peashooterCardEntity := em.CreateEntity()
	ecs.AddComponent(em, peashooterCardEntity, &components.PlantCardComponent{
		PlantType: components.PlantPeashooter,
	})
	ecs.AddComponent(em, peashooterCardEntity, &components.PositionComponent{X: 100, Y: 100})

	// 仍然应该返回0（没有向日葵卡片）
	cardID = ts.findSunflowerCard()
	if cardID != 0 {
		t.Errorf("findSunflowerCard() should return 0 when only peashooter card exists, got %d", cardID)
	}

	// 创建向日葵卡片
	sunflowerCardEntity := em.CreateEntity()
	ecs.AddComponent(em, sunflowerCardEntity, &components.PlantCardComponent{
		PlantType: components.PlantSunflower,
	})
	ecs.AddComponent(em, sunflowerCardEntity, &components.PositionComponent{X: 200, Y: 100})

	// 现在应该能找到向日葵卡片
	cardID = ts.findSunflowerCard()
	if cardID != sunflowerCardEntity {
		t.Errorf("findSunflowerCard() = %d, expected %d", cardID, sunflowerCardEntity)
	}
}

// TestSunflowerCountTracking 测试向日葵计数跟踪
func TestSunflowerCountTracking(t *testing.T) {
	tutorialSteps := []config.TutorialStep{
		{Trigger: "gameStart", TextKey: "TEST", Action: "sunflowerHint"},
	}
	ts, em, _ := createTestTutorialSystemForSunflower(tutorialSteps)

	// 初始状态：0颗向日葵
	ts.updateTrackingState()
	if ts.sunflowerCount != 0 {
		t.Errorf("sunflowerCount = %d, expected 0", ts.sunflowerCount)
	}

	// 创建1颗向日葵
	sunflower1 := em.CreateEntity()
	ecs.AddComponent(em, sunflower1, &components.PlantComponent{
		PlantType: components.PlantSunflower,
	})

	ts.updateTrackingState()
	if ts.sunflowerCount != 1 {
		t.Errorf("sunflowerCount = %d, expected 1", ts.sunflowerCount)
	}

	// 创建1颗豌豆射手（不应影响向日葵计数）
	peashooter := em.CreateEntity()
	ecs.AddComponent(em, peashooter, &components.PlantComponent{
		PlantType: components.PlantPeashooter,
	})

	ts.updateTrackingState()
	if ts.sunflowerCount != 1 {
		t.Errorf("sunflowerCount = %d, expected 1 (peashooter should not count)", ts.sunflowerCount)
	}

	// 创建2颗更多向日葵
	sunflower2 := em.CreateEntity()
	ecs.AddComponent(em, sunflower2, &components.PlantComponent{
		PlantType: components.PlantSunflower,
	})
	sunflower3 := em.CreateEntity()
	ecs.AddComponent(em, sunflower3, &components.PlantComponent{
		PlantType: components.PlantSunflower,
	})

	ts.updateTrackingState()
	if ts.sunflowerCount != 3 {
		t.Errorf("sunflowerCount = %d, expected 3", ts.sunflowerCount)
	}
}

// TestStepTimeElapsedUpdate 测试步骤计时器更新
func TestStepTimeElapsedUpdate(t *testing.T) {
	tutorialSteps := []config.TutorialStep{
		{Trigger: "sunflowerCount1OrTimeout", TextKey: "TEST1", Action: "sunflowerHint"},
		{Trigger: "sunflowerCount2OrTimeout", TextKey: "TEST2", Action: "sunflowerHint"},
	}
	ts, _, _ := createTestTutorialSystemForSunflower(tutorialSteps)

	// 初始计时器应为0
	if ts.stepTimeElapsed != 0 {
		t.Errorf("stepTimeElapsed = %.2f, expected 0", ts.stepTimeElapsed)
	}

	// 更新 5 秒
	ts.Update(5.0)

	// 计时器应增加
	if ts.stepTimeElapsed < 5.0 {
		t.Errorf("stepTimeElapsed = %.2f, expected >= 5.0", ts.stepTimeElapsed)
	}
}

// TestSunflowerTutorialProgression 测试向日葵教学流程推进
func TestSunflowerTutorialProgression(t *testing.T) {
	tutorialSteps := []config.TutorialStep{
		{Trigger: "gameStart", TextKey: "ADVICE_PLANT_SUNFLOWER1", Action: "sunflowerHint"},
		{Trigger: "sunflowerCount1OrTimeout", TextKey: "ADVICE_PLANT_SUNFLOWER2", Action: "sunflowerHint"},
		{Trigger: "sunflowerCount3", TextKey: "", Action: "completeTutorial"},
	}
	ts, em, _ := createTestTutorialSystemForSunflower(tutorialSteps)

	// 获取教学组件
	tutorial, ok := ecs.GetComponent[*components.TutorialComponent](em, ts.tutorialEntity)
	if !ok {
		t.Fatal("TutorialComponent not found")
	}

	// 初始步骤应为0
	if tutorial.CurrentStepIndex != 0 {
		t.Errorf("CurrentStepIndex = %d, expected 0", tutorial.CurrentStepIndex)
	}

	// 更新触发 gameStart
	ts.Update(0.016)
	if tutorial.CurrentStepIndex != 1 {
		t.Errorf("After gameStart, CurrentStepIndex = %d, expected 1", tutorial.CurrentStepIndex)
	}

	// 种植1颗向日葵
	sunflower1 := em.CreateEntity()
	ecs.AddComponent(em, sunflower1, &components.PlantComponent{
		PlantType: components.PlantSunflower,
	})
	ts.updateTrackingState()
	ts.Update(0.016)

	if tutorial.CurrentStepIndex != 2 {
		t.Errorf("After 1 sunflower, CurrentStepIndex = %d, expected 2", tutorial.CurrentStepIndex)
	}

	// 种植更多向日葵达到3颗
	sunflower2 := em.CreateEntity()
	ecs.AddComponent(em, sunflower2, &components.PlantComponent{
		PlantType: components.PlantSunflower,
	})
	sunflower3 := em.CreateEntity()
	ecs.AddComponent(em, sunflower3, &components.PlantComponent{
		PlantType: components.PlantSunflower,
	})
	ts.updateTrackingState()
	ts.Update(0.016)

	// 注意：completeTutorial action 会提前 return，不会执行 CurrentStepIndex++
	// 所以步骤索引仍然是 2，但教学已标记为完成
	if tutorial.CurrentStepIndex != 2 {
		t.Errorf("After 3 sunflowers, CurrentStepIndex = %d, expected 2 (completeTutorial returns early)", tutorial.CurrentStepIndex)
	}

	// 教学应该已完成
	if tutorial.IsActive {
		t.Error("Tutorial should be inactive after completing all steps")
	}
}
