package components

import "testing"

// TestWaveTimerComponent_FieldInitialization 测试组件字段初始化
func TestWaveTimerComponent_FieldInitialization(t *testing.T) {
	// 创建组件并检查默认值
	timer := &WaveTimerComponent{}

	// 检查所有字段的零值
	if timer.CountdownCs != 0 {
		t.Errorf("Expected CountdownCs = 0, got %d", timer.CountdownCs)
	}

	if timer.AccumulatedCs != 0 {
		t.Errorf("Expected AccumulatedCs = 0, got %f", timer.AccumulatedCs)
	}

	if timer.IsFirstWave != false {
		t.Errorf("Expected IsFirstWave = false, got %v", timer.IsFirstWave)
	}

	if timer.CurrentWaveIndex != 0 {
		t.Errorf("Expected CurrentWaveIndex = 0, got %d", timer.CurrentWaveIndex)
	}

	if timer.TotalWaves != 0 {
		t.Errorf("Expected TotalWaves = 0, got %d", timer.TotalWaves)
	}

	if timer.IsPaused != false {
		t.Errorf("Expected IsPaused = false, got %v", timer.IsPaused)
	}

	if timer.WaveStartedAt != 0 {
		t.Errorf("Expected WaveStartedAt = 0, got %f", timer.WaveStartedAt)
	}

	if timer.LastRefreshTimeCs != 0 {
		t.Errorf("Expected LastRefreshTimeCs = 0, got %d", timer.LastRefreshTimeCs)
	}

	if timer.WaveTriggered != false {
		t.Errorf("Expected WaveTriggered = false, got %v", timer.WaveTriggered)
	}
}

// TestWaveTimerComponent_CustomInitialization 测试自定义初始化
func TestWaveTimerComponent_CustomInitialization(t *testing.T) {
	// 创建组件并设置自定义值
	timer := &WaveTimerComponent{
		CountdownCs:       2500,
		AccumulatedCs:     0.5,
		IsFirstWave:       true,
		CurrentWaveIndex:  3,
		TotalWaves:        10,
		IsPaused:          true,
		WaveStartedAt:     15.5,
		LastRefreshTimeCs: 2800,
		WaveTriggered:     true,
	}

	// 检查所有字段
	if timer.CountdownCs != 2500 {
		t.Errorf("Expected CountdownCs = 2500, got %d", timer.CountdownCs)
	}

	if timer.AccumulatedCs != 0.5 {
		t.Errorf("Expected AccumulatedCs = 0.5, got %f", timer.AccumulatedCs)
	}

	if timer.IsFirstWave != true {
		t.Errorf("Expected IsFirstWave = true, got %v", timer.IsFirstWave)
	}

	if timer.CurrentWaveIndex != 3 {
		t.Errorf("Expected CurrentWaveIndex = 3, got %d", timer.CurrentWaveIndex)
	}

	if timer.TotalWaves != 10 {
		t.Errorf("Expected TotalWaves = 10, got %d", timer.TotalWaves)
	}

	if timer.IsPaused != true {
		t.Errorf("Expected IsPaused = true, got %v", timer.IsPaused)
	}

	if timer.WaveStartedAt != 15.5 {
		t.Errorf("Expected WaveStartedAt = 15.5, got %f", timer.WaveStartedAt)
	}

	if timer.LastRefreshTimeCs != 2800 {
		t.Errorf("Expected LastRefreshTimeCs = 2800, got %d", timer.LastRefreshTimeCs)
	}

	if timer.WaveTriggered != true {
		t.Errorf("Expected WaveTriggered = true, got %v", timer.WaveTriggered)
	}
}

// TestWaveTimerComponent_FieldMutation 测试字段修改
func TestWaveTimerComponent_FieldMutation(t *testing.T) {
	timer := &WaveTimerComponent{
		CountdownCs:      2500,
		CurrentWaveIndex: 0,
	}

	// 模拟倒计时递减
	timer.CountdownCs -= 100
	if timer.CountdownCs != 2400 {
		t.Errorf("Expected CountdownCs = 2400 after decrement, got %d", timer.CountdownCs)
	}

	// 模拟波次触发
	timer.WaveTriggered = true
	timer.CurrentWaveIndex++
	if timer.CurrentWaveIndex != 1 {
		t.Errorf("Expected CurrentWaveIndex = 1 after increment, got %d", timer.CurrentWaveIndex)
	}

	// 模拟暂停
	timer.IsPaused = true
	if !timer.IsPaused {
		t.Error("Expected IsPaused = true after setting")
	}
}

