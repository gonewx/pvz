package systems

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// 测试辅助函数

// createTestLevelConfig 创建测试用关卡配置
func createTestLevelConfig(waveCount int) *config.LevelConfig {
	waves := make([]config.WaveConfig, waveCount)
	for i := 0; i < waveCount; i++ {
		waves[i] = config.WaveConfig{
			Delay:    0,
			MinDelay: 5.0,
			Zombies: []config.ZombieGroup{
				{Type: "basic", Count: 2, Lanes: []int{1, 2, 3}},
			},
		}
	}
	return &config.LevelConfig{
		ID:    "test-level",
		Waves: waves,
	}
}

// createTestGameState 创建测试用 GameState
func createTestGameState() *game.GameState {
	// 使用反射或直接创建，但 GetGameState 是单例
	// 为测试目的，我们直接使用单例
	return game.GetGameState()
}

// resetGameState 重置 GameState（用于测试隔离）
func resetGameState(gs *game.GameState, levelConfig *config.LevelConfig) {
	gs.LoadLevel(levelConfig)
	gs.LevelTime = 0
	gs.IsGameOver = false
	gs.GameResult = ""
}

// TestWaveTimingSystem_Creation 测试系统创建
func TestWaveTimingSystem_Creation(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := createTestGameState()
	levelConfig := createTestLevelConfig(5)
	resetGameState(gs, levelConfig)

	system := NewWaveTimingSystem(em, gs, levelConfig)

	if system == nil {
		t.Fatal("Expected system to be created, got nil")
	}

	if system.timerEntityID == 0 {
		t.Error("Expected timer entity ID to be non-zero")
	}

	// 检查计时器组件是否创建
	timer := system.getTimerComponent()
	if timer == nil {
		t.Fatal("Expected timer component to be created")
	}

	if timer.TotalWaves != 5 {
		t.Errorf("Expected TotalWaves = 5, got %d", timer.TotalWaves)
	}
}

// TestWaveTimingSystem_InitializeTimer_FirstPlaythrough 测试首次游戏初始化
func TestWaveTimingSystem_InitializeTimer_FirstPlaythrough(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := createTestGameState()
	levelConfig := createTestLevelConfig(3)
	resetGameState(gs, levelConfig)

	system := NewWaveTimingSystem(em, gs, levelConfig)
	system.InitializeTimer(true) // 首次游戏

	timer := system.getTimerComponent()
	if timer == nil {
		t.Fatal("Timer component not found")
	}

	// 首次游戏：立即触发（CountdownCs = 0）
	if timer.CountdownCs != 0 {
		t.Errorf("Expected CountdownCs = 0 for first playthrough, got %d", timer.CountdownCs)
	}

	if !timer.IsFirstWave {
		t.Error("Expected IsFirstWave = true for first playthrough")
	}
}

// TestWaveTimingSystem_InitializeTimer_SubsequentPlaythrough 测试非首次游戏初始化
func TestWaveTimingSystem_InitializeTimer_SubsequentPlaythrough(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := createTestGameState()
	levelConfig := createTestLevelConfig(3)
	resetGameState(gs, levelConfig)

	system := NewWaveTimingSystem(em, gs, levelConfig)
	system.InitializeTimer(false) // 非首次游戏

	timer := system.getTimerComponent()
	if timer == nil {
		t.Fatal("Timer component not found")
	}

	// 非首次游戏：599 厘秒延迟
	if timer.CountdownCs != FirstWaveDelayCs {
		t.Errorf("Expected CountdownCs = %d for subsequent playthrough, got %d", FirstWaveDelayCs, timer.CountdownCs)
	}

	if timer.IsFirstWave {
		t.Error("Expected IsFirstWave = false for subsequent playthrough")
	}
}

// TestWaveTimingSystem_Update_ImmediateTrigger 测试首波立即触发
func TestWaveTimingSystem_Update_ImmediateTrigger(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := createTestGameState()
	levelConfig := createTestLevelConfig(3)
	resetGameState(gs, levelConfig)

	system := NewWaveTimingSystem(em, gs, levelConfig)
	system.InitializeTimer(true) // 首次游戏，立即触发

	// 更新一帧
	system.Update(0.01)

	// 检查是否触发
	triggered, waveIndex := system.IsWaveTriggered()
	if !triggered {
		t.Error("Expected wave to be triggered on first update")
	}
	if waveIndex != 0 {
		t.Errorf("Expected waveIndex = 0, got %d", waveIndex)
	}
}

// TestWaveTimingSystem_Update_DelayedTrigger 测试延迟触发
func TestWaveTimingSystem_Update_DelayedTrigger(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := createTestGameState()
	levelConfig := createTestLevelConfig(3)
	resetGameState(gs, levelConfig)

	system := NewWaveTimingSystem(em, gs, levelConfig)
	system.InitializeTimer(false) // 非首次游戏，599cs 延迟

	timer := system.getTimerComponent()
	initialCountdown := timer.CountdownCs

	// 更新 1 秒（100 厘秒）
	system.Update(1.0)

	// 检查倒计时递减
	if timer.CountdownCs >= initialCountdown {
		t.Error("Expected countdown to decrease after update")
	}

	// 不应该触发（还有约 5 秒）
	triggered, _ := system.IsWaveTriggered()
	if triggered {
		t.Error("Wave should not be triggered yet")
	}
}

// TestWaveTimingSystem_Update_CountdownToTrigger 测试倒计时到1时触发
func TestWaveTimingSystem_Update_CountdownToTrigger(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := createTestGameState()
	levelConfig := createTestLevelConfig(3)
	resetGameState(gs, levelConfig)

	system := NewWaveTimingSystem(em, gs, levelConfig)

	// 直接设置倒计时为 2
	timer := system.getTimerComponent()
	timer.CountdownCs = 2
	timer.CurrentWaveIndex = 0 // 等待第一波

	// 更新 0.02 秒（2 厘秒）
	system.Update(0.02)

	// 检查是否触发
	triggered, waveIndex := system.IsWaveTriggered()
	if !triggered {
		t.Error("Expected wave to be triggered when countdown <= 1")
	}
	if waveIndex != 0 {
		t.Errorf("Expected waveIndex = 0, got %d", waveIndex)
	}
}

// TestWaveTimingSystem_SetNextWaveCountdown 测试设置下一波倒计时
func TestWaveTimingSystem_SetNextWaveCountdown(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := createTestGameState()
	levelConfig := createTestLevelConfig(5)
	resetGameState(gs, levelConfig)

	system := NewWaveTimingSystem(em, gs, levelConfig)

	// 多次测试随机值范围
	for i := 0; i < 100; i++ {
		system.SetNextWaveCountdown()

		timer := system.getTimerComponent()
		countdown := timer.CountdownCs

		// 检查范围：2500-3099 厘秒
		minExpected := RegularWaveBaseDelayCs
		maxExpected := RegularWaveBaseDelayCs + RegularWaveRandomDelayCs - 1

		if countdown < minExpected || countdown > maxExpected {
			t.Errorf("Expected countdown in range [%d, %d], got %d", minExpected, maxExpected, countdown)
		}
	}
}

// TestWaveTimingSystem_PauseResume 测试暂停/恢复功能
func TestWaveTimingSystem_PauseResume(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := createTestGameState()
	levelConfig := createTestLevelConfig(3)
	resetGameState(gs, levelConfig)

	system := NewWaveTimingSystem(em, gs, levelConfig)

	// 设置倒计时
	timer := system.getTimerComponent()
	timer.CountdownCs = 1000

	// 暂停
	system.Pause()

	if !timer.IsPaused {
		t.Error("Expected timer to be paused")
	}

	// 更新 5 秒
	initialCountdown := timer.CountdownCs
	system.Update(5.0)

	// 倒计时不应变化
	if timer.CountdownCs != initialCountdown {
		t.Errorf("Expected countdown unchanged during pause, got %d (was %d)", timer.CountdownCs, initialCountdown)
	}

	// 恢复
	system.Resume()

	if timer.IsPaused {
		t.Error("Expected timer to be resumed")
	}

	// 更新 1 秒
	system.Update(1.0)

	// 倒计时应该递减
	if timer.CountdownCs >= initialCountdown {
		t.Error("Expected countdown to decrease after resume")
	}
}

// TestWaveTimingSystem_MultipleWaves 测试多波次触发
func TestWaveTimingSystem_MultipleWaves(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := createTestGameState()
	levelConfig := createTestLevelConfig(3)
	resetGameState(gs, levelConfig)

	system := NewWaveTimingSystem(em, gs, levelConfig)
	system.InitializeTimer(true) // 首次游戏

	// 触发第一波
	system.Update(0.01)
	triggered, waveIndex := system.IsWaveTriggered()
	if !triggered || waveIndex != 0 {
		t.Errorf("Expected wave 0 to be triggered, got triggered=%v, waveIndex=%d", triggered, waveIndex)
	}
	system.ClearWaveTriggered()

	// 检查已设置下一波倒计时
	timer := system.getTimerComponent()
	if timer.CountdownCs < RegularWaveBaseDelayCs {
		t.Errorf("Expected next wave countdown >= %d, got %d", RegularWaveBaseDelayCs, timer.CountdownCs)
	}

	// 检查当前波次索引
	if timer.CurrentWaveIndex != 1 {
		t.Errorf("Expected CurrentWaveIndex = 1, got %d", timer.CurrentWaveIndex)
	}
}

// TestWaveTimingSystem_AllWavesComplete 测试所有波次完成
func TestWaveTimingSystem_AllWavesComplete(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := createTestGameState()
	levelConfig := createTestLevelConfig(2) // 只有 2 波
	resetGameState(gs, levelConfig)

	system := NewWaveTimingSystem(em, gs, levelConfig)

	timer := system.getTimerComponent()
	timer.CurrentWaveIndex = 2 // 设置为已完成所有波次
	timer.CountdownCs = 0

	// 更新
	system.Update(0.01)

	// 不应触发新波次
	triggered, _ := system.IsWaveTriggered()
	if triggered {
		t.Error("Should not trigger wave when all waves are complete")
	}
}

// TestWaveTimingSystem_NegativeCountdownProtection 测试负数倒计时保护
func TestWaveTimingSystem_NegativeCountdownProtection(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := createTestGameState()
	levelConfig := createTestLevelConfig(3)
	resetGameState(gs, levelConfig)

	system := NewWaveTimingSystem(em, gs, levelConfig)

	timer := system.getTimerComponent()
	timer.CountdownCs = 1
	timer.CurrentWaveIndex = 0

	// 更新较大的时间步长
	system.Update(1.0) // 100 厘秒

	// 检查触发
	triggered, _ := system.IsWaveTriggered()
	if !triggered {
		t.Error("Expected wave to be triggered")
	}

	// 检查 CurrentWaveIndex 已递增
	if timer.CurrentWaveIndex != 1 {
		t.Errorf("Expected CurrentWaveIndex = 1, got %d", timer.CurrentWaveIndex)
	}
}

// TestWaveTimingSystem_ClearWaveTriggered 测试清除触发标志
func TestWaveTimingSystem_ClearWaveTriggered(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := createTestGameState()
	levelConfig := createTestLevelConfig(3)
	resetGameState(gs, levelConfig)

	system := NewWaveTimingSystem(em, gs, levelConfig)
	system.InitializeTimer(true)

	// 触发第一波
	system.Update(0.01)

	triggered, _ := system.IsWaveTriggered()
	if !triggered {
		t.Error("Expected wave to be triggered")
	}

	// 清除标志
	system.ClearWaveTriggered()

	triggered, _ = system.IsWaveTriggered()
	if triggered {
		t.Error("Expected wave triggered flag to be cleared")
	}
}

// TestWaveTimingSystem_GetCountdownSeconds 测试获取倒计时秒数
func TestWaveTimingSystem_GetCountdownSeconds(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := createTestGameState()
	levelConfig := createTestLevelConfig(3)
	resetGameState(gs, levelConfig)

	system := NewWaveTimingSystem(em, gs, levelConfig)

	timer := system.getTimerComponent()
	timer.CountdownCs = 2500

	seconds := system.GetCountdownSeconds()
	expected := 25.0

	if seconds != expected {
		t.Errorf("Expected %.2f seconds, got %.2f", expected, seconds)
	}
}

// TestWaveTimingSystem_GetCurrentWaveIndex 测试获取当前波次索引
func TestWaveTimingSystem_GetCurrentWaveIndex(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := createTestGameState()
	levelConfig := createTestLevelConfig(3)
	resetGameState(gs, levelConfig)

	system := NewWaveTimingSystem(em, gs, levelConfig)

	timer := system.getTimerComponent()
	timer.CurrentWaveIndex = 2

	index := system.GetCurrentWaveIndex()
	if index != 2 {
		t.Errorf("Expected current wave index = 2, got %d", index)
	}
}

// TestWaveTimingSystem_AccumulatedCsHandling 测试累积厘秒处理
func TestWaveTimingSystem_AccumulatedCsHandling(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := createTestGameState()
	levelConfig := createTestLevelConfig(3)
	resetGameState(gs, levelConfig)

	system := NewWaveTimingSystem(em, gs, levelConfig)

	timer := system.getTimerComponent()
	timer.CountdownCs = 1000
	timer.AccumulatedCs = 0

	// 更新 0.005 秒（0.5 厘秒，不足 1 厘秒）
	system.Update(0.005)

	// 累积值应该增加
	if timer.AccumulatedCs < 0.4 || timer.AccumulatedCs > 0.6 {
		t.Errorf("Expected AccumulatedCs around 0.5, got %f", timer.AccumulatedCs)
	}

	// 再更新 0.005 秒
	system.Update(0.005)

	// 现在应该递减 1 厘秒
	if timer.CountdownCs != 999 {
		t.Errorf("Expected CountdownCs = 999, got %d", timer.CountdownCs)
	}
}

// TestWaveTimingSystem_TimerEntityID 测试获取计时器实体ID
func TestWaveTimingSystem_TimerEntityID(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := createTestGameState()
	levelConfig := createTestLevelConfig(3)
	resetGameState(gs, levelConfig)

	system := NewWaveTimingSystem(em, gs, levelConfig)

	entityID := system.GetTimerEntityID()
	if entityID == 0 {
		t.Error("Expected non-zero timer entity ID")
	}

	// 验证可以通过 EntityManager 获取组件
	timer, ok := ecs.GetComponent[*components.WaveTimerComponent](em, entityID)
	if !ok || timer == nil {
		t.Error("Expected to retrieve timer component via entity ID")
	}
}

