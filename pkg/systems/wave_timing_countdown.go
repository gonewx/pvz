package systems

import (
	"log"
	"math/rand"

	"github.com/gonewx/pvz/pkg/game"
)

// ============================================================================
// 波次倒计时和触发逻辑
// ============================================================================

// triggerNextWave 触发下一波
func (s *WaveTimingSystem) triggerNextWave() {
	timer := s.getTimerComponent()
	if timer == nil {
		return
	}

	// 设置触发标志
	timer.WaveTriggered = true
	timer.WaveStartedAt = s.gameState.LevelTime

	waveIndex := timer.CurrentWaveIndex
	log.Printf("[WaveTimingSystem] ✅ Wave %d triggered at time %.2fs", waveIndex+1, timer.WaveStartedAt)

	// Story 10.9: 僵尸入场时播放音效
	// - 第一波：SOUND_SIREN + SOUND_AWOOGA
	// - 旗帜波/最终波：SOUND_AWOOGA（hugewave/finalwave 音效在提示文本出现时播放）
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		if waveIndex == 0 {
			// 第一波：播放 siren + awooga
			audioManager.PlaySound("SOUND_SIREN")
			audioManager.PlaySound("SOUND_AWOOGA")
			log.Printf("[WaveTimingSystem] Playing SOUND_SIREN + SOUND_AWOOGA for first wave")
		} else if timer.IsFlagWaveApproaching || timer.IsFinalWave {
			// 旗帜波或最终波：僵尸入场时只播放 awooga
			audioManager.PlaySound("SOUND_AWOOGA")
			log.Printf("[WaveTimingSystem] Playing SOUND_AWOOGA for flag/final wave entry")
		}
	}

	// 递增波次索引（下一次会触发下一波）
	timer.CurrentWaveIndex++

	// 如果还有后续波次，设置下一波倒计时
	if timer.CurrentWaveIndex < timer.TotalWaves {
		s.SetNextWaveCountdown()
	} else {
		// 所有波次已触发，清除相关标志和计时器
		timer.IsFlagWaveApproaching = false
		timer.IsFinalWave = false
		timer.LastRefreshTimeCs = 0
		// 设置一个很大的值防止 Update() 中再次触发
		// 不能设为 0，否则 CountdownCs <= 1 条件会再次触发 triggerNextWave()
		timer.CountdownCs = 999999
		timer.WaveElapsedCs = 0
		log.Printf("[WaveTimingSystem] All waves triggered. Timer stopped.")
	}
}

// SetNextWaveCountdown 设置下一波倒计时
//
// Story 17.7: 根据下一波类型设置不同的倒计时：
//   - 旗帜波前一波：4500cs（45秒）
//   - 最终波：5500cs（55秒）
//   - 常规波：2500 + rand(600) 厘秒（25-31秒）
//
// Bug Fix: 旗帜波和最终波独立判断
//   - 如果某波既是旗帜波又是最终波，两个标志都会设置为 true
//   - 倒计时取两者的最大值（5500cs）
//   - 警告队列会同时添加 "huge_wave" 和 "final_wave"
func (s *WaveTimingSystem) SetNextWaveCountdown() {
	timer := s.getTimerComponent()
	if timer == nil {
		return
	}

	// 重置波次已过时间（用于加速刷新判定）
	timer.WaveElapsedCs = 0

	// 清空警告队列和索引
	timer.PendingWarnings = nil
	timer.CurrentWarningIndex = 0

	// Story 17.7 + Bug Fix: 独立判断旗帜波和最终波
	nextWaveIndex := timer.CurrentWaveIndex
	var countdown int
	var waveTypes []string

	isFlagWave := s.isNextWaveFlagWave(nextWaveIndex)
	isFinal := s.isFinalWave(nextWaveIndex)

	// 设置标志位（独立判断，可以同时为 true）
	timer.IsFlagWaveApproaching = isFlagWave
	timer.IsFinalWave = isFinal

	if isFlagWave {
		timer.HugeWaveWarningTriggered = false
		timer.PendingWarnings = append(timer.PendingWarnings, "huge_wave")
		waveTypes = append(waveTypes, "flag wave prefix")
		countdown = FlagWavePrefixDelayCs // 4500cs
	}

	if isFinal {
		timer.FinalWaveWarningTriggered = false
		timer.PendingWarnings = append(timer.PendingWarnings, "final_wave")
		waveTypes = append(waveTypes, "final wave")
		// 最终波倒计时 5500cs，如果同时是旗帜波取最大值
		if countdown < FinalWaveDelayCs {
			countdown = FinalWaveDelayCs
		}
	}

	// 如果既不是旗帜波也不是最终波，则为常规波
	if !isFlagWave && !isFinal {
		countdown = RegularWaveBaseDelayCs + rand.Intn(RegularWaveRandomDelayCs)
		waveTypes = append(waveTypes, "regular wave")
	}

	timer.CountdownCs = countdown
	timer.LastRefreshTimeCs = countdown
	timer.AccumulatedCs = 0

	// 构建日志中的类型字符串
	waveTypeStr := "regular wave"
	if len(waveTypes) > 0 {
		waveTypeStr = waveTypes[0]
		for i := 1; i < len(waveTypes); i++ {
			waveTypeStr += " + " + waveTypes[i]
		}
	}

	log.Printf("[WaveTimingSystem] Next wave countdown set: %d cs (%.2fs) [%s, wave %d], pending warnings: %v",
		countdown, float64(countdown)/100, waveTypeStr, nextWaveIndex+1, timer.PendingWarnings)
}

// isNextWaveFlagWave 判断下一波是否为旗帜波
//
// Story 17.7: 检查关卡配置中下一波的 IsFlag 字段
//
// 参数：
//   - nextWaveIndex: 下一波的索引（0-based）
//
// 返回：
//   - bool: true 表示下一波是旗帜波
func (s *WaveTimingSystem) isNextWaveFlagWave(nextWaveIndex int) bool {
	if s.levelConfig == nil {
		return false
	}

	if nextWaveIndex < 0 || nextWaveIndex >= len(s.levelConfig.Waves) {
		return false
	}

	return s.levelConfig.Waves[nextWaveIndex].IsFlag
}

// isFinalWave 判断指定波次是否为最终波
//
// Story 17.7: 最终波 = 最后一个标记为 isFlag 的波次，或关卡最后一波
//
// 参数：
//   - waveIndex: 波次索引（0-based）
//
// 返回：
//   - bool: true 表示是最终波
func (s *WaveTimingSystem) isFinalWave(waveIndex int) bool {
	if s.levelConfig == nil {
		return false
	}

	totalWaves := len(s.levelConfig.Waves)
	if totalWaves == 0 {
		return false
	}

	// 最后一波是最终波
	if waveIndex == totalWaves-1 {
		return true
	}

	// 检查是否为关卡配置中标记的最终波（Type="Final"）
	if waveIndex >= 0 && waveIndex < totalWaves {
		return s.levelConfig.Waves[waveIndex].Type == "Final"
	}

	return false
}

// TriggerNextWaveImmediately 立即触发下一波
//
// 用于教学关卡：当玩家完成种植条件后，立即触发第一波僵尸
// 同时恢复计时器，让后续波次由计时系统管理
//
// 返回：
//   - int: 触发的波次索引（-1 表示失败）
func (s *WaveTimingSystem) TriggerNextWaveImmediately() int {
	timer := s.getTimerComponent()
	if timer == nil {
		return -1
	}

	// 记录触发的波次索引
	waveIndex := timer.CurrentWaveIndex

	// 检查是否还有波次可触发
	if waveIndex >= timer.TotalWaves {
		log.Printf("[WaveTimingSystem] No more waves to trigger (current: %d, total: %d)", waveIndex, timer.TotalWaves)
		return -1
	}

	// 恢复计时器
	timer.IsPaused = false

	// 立即触发下一波（triggerNextWave 会自动为后续波次设置倒计时）
	s.triggerNextWave()

	log.Printf("[WaveTimingSystem] Immediately triggered wave %d, timer resumed for subsequent waves", waveIndex+1)

	return waveIndex
}

// IsWaveTriggered 检查本帧是否触发了波次
//
// 返回：
//   - bool: true 表示本帧触发了波次
//   - int: 触发的波次索引（-1 表示未触发）
func (s *WaveTimingSystem) IsWaveTriggered() (bool, int) {
	timer := s.getTimerComponent()
	if timer == nil {
		return false, -1
	}

	if timer.WaveTriggered {
		// 返回刚触发的波次索引（CurrentWaveIndex 已经递增，所以要 -1）
		return true, timer.CurrentWaveIndex - 1
	}

	return false, -1
}

// ClearWaveTriggered 清除波次触发标志
// LevelSystem 处理完触发事件后调用
func (s *WaveTimingSystem) ClearWaveTriggered() {
	timer := s.getTimerComponent()
	if timer == nil {
		return
	}

	timer.WaveTriggered = false
}

// GetCountdownSeconds 获取当前倒计时（秒）
// 用于调试显示
func (s *WaveTimingSystem) GetCountdownSeconds() float64 {
	timer := s.getTimerComponent()
	if timer == nil {
		return 0
	}

	return float64(timer.CountdownCs) / 100
}

// GetCurrentWaveIndex 获取当前等待的波次索引
func (s *WaveTimingSystem) GetCurrentWaveIndex() int {
	timer := s.getTimerComponent()
	if timer == nil {
		return 0
	}

	return timer.CurrentWaveIndex
}

// GetWaveElapsedCs 获取当前波已过时间（厘秒）
//
// Story 17.7: 供调试和测试使用
//
// 返回：
//   - int: 当前波刷出后已过的厘秒数
func (s *WaveTimingSystem) GetWaveElapsedCs() int {
	timer := s.getTimerComponent()
	if timer == nil {
		return 0
	}

	return timer.WaveElapsedCs
}

// IsFlagWaveApproaching 检查是否正在接近旗帜波
//
// Story 17.7: 供 LevelSystem 检查是否需要调用加速刷新检查
//
// 返回：
//   - bool: true 表示正在接近旗帜波
func (s *WaveTimingSystem) IsFlagWaveApproaching() bool {
	timer := s.getTimerComponent()
	if timer == nil {
		return false
	}

	return timer.IsFlagWaveApproaching
}

// GetNextWaveDelay 获取下一波的初始倒计时（秒）
//
// Story 11.5: 用于进度条的时间进度计算
// 返回当前波次触发后到下一波的预计延迟时间
//
// 修复说明：
// 此函数在 checkAndSpawnWaves() 中调用，此时 triggerNextWave() 已经执行完毕，
// CurrentWaveIndex 已递增，LastRefreshTimeCs 已经被 SetNextWaveCountdown() 设置。
// 所以直接返回 LastRefreshTimeCs 即可，不需要重新计算。
//
// 返回:
//   - float64: 下一波延迟时间（秒），如果是最后一波返回 0
func (s *WaveTimingSystem) GetNextWaveDelay() float64 {
	timer := s.getTimerComponent()
	if timer == nil {
		return 0
	}

	// 如果已经是最后一波之后，返回 0
	if timer.CurrentWaveIndex >= timer.TotalWaves {
		return 0
	}

	// 直接使用已设置的 LastRefreshTimeCs
	// 这个值在 SetNextWaveCountdown() 中已经正确设置
	if timer.LastRefreshTimeCs > 0 {
		return float64(timer.LastRefreshTimeCs) / 100.0
	}

	return 0
}

// CheckAcceleratedRefresh 检查并执行加速刷新
//
// Story 17.7: 旗帜波前一波的加速刷新逻辑（消灭触发）
//
// 加速刷新条件：
//   - 当前波刷出时间 > 401cs
//   - 当前倒计时 > 200cs
//   - 本波僵尸已全部消灭（除伴舞）
//
// 当条件满足时，将倒计时设为 200cs
//
// 参数：
//   - allZombiesCleared: 是否所有僵尸已消灭（由 LevelSystem 提供）
//
// 返回：
//   - bool: true 表示触发了加速刷新
func (s *WaveTimingSystem) CheckAcceleratedRefresh(allZombiesCleared bool) bool {
	timer := s.getTimerComponent()
	if timer == nil {
		return false
	}

	// 只在接近旗帜波时才检查加速刷新
	if !timer.IsFlagWaveApproaching {
		return false
	}

	// 红字警告阶段不加速
	if timer.FlagWaveCountdownPhase > 0 {
		return false
	}

	// 检查加速刷新条件
	// 1. 刷出时间 > 401cs
	if timer.WaveElapsedCs <= AcceleratedRefreshMinTimeCs {
		return false
	}

	// 2. 倒计时 > 200cs
	if timer.CountdownCs <= AcceleratedRefreshCountdownCs {
		return false
	}

	// 3. 本波僵尸已全部消灭
	if !allZombiesCleared {
		return false
	}

	// 触发加速刷新
	oldCountdown := timer.CountdownCs
	timer.CountdownCs = AcceleratedRefreshCountdownCs
	timer.AccumulatedCs = 0

	log.Printf("[WaveTimingSystem] ⚡ Accelerated refresh triggered! Countdown: %d cs → %d cs (elapsed: %d cs)",
		oldCountdown, AcceleratedRefreshCountdownCs, timer.WaveElapsedCs)

	return true
}

// CheckHealthAcceleratedRefresh 检查并执行血量触发的加速刷新
//
// Story 17.8: 常规波次（非旗帜波前）的血量触发加速刷新逻辑
//
// 加速刷新条件：
//   - 非旗帜波前（!IsFlagWaveApproaching）
//   - 本波刷出时间 > 401cs
//   - 当前倒计时 > 200cs
//   - 当前血量 <= 初始血量 × 阈值（0.50~0.65）
//   - 未触发过血量加速
//
// 当条件满足时，将倒计时设为 200cs
//
// 参数：
//   - currentHealth: 当前僵尸总血量（由调用方计算提供）
//
// 返回：
//   - bool: true 表示触发了加速刷新
func (s *WaveTimingSystem) CheckHealthAcceleratedRefresh(currentHealth int) bool {
	timer := s.getTimerComponent()
	if timer == nil {
		return false
	}

	// 只在常规波次（非旗帜波前）检查血量加速
	if timer.IsFlagWaveApproaching {
		return false
	}

	// 红字警告阶段不加速
	if timer.FlagWaveCountdownPhase > 0 {
		return false
	}

	// 已触发过血量加速，不重复触发
	if timer.HealthAccelerationTriggered {
		return false
	}

	// 检查加速刷新条件
	// 1. 刷出时间 > 401cs
	if timer.WaveElapsedCs <= AcceleratedRefreshMinTimeCs {
		return false
	}

	// 2. 倒计时 > 200cs
	if timer.CountdownCs <= AcceleratedRefreshCountdownCs {
		return false
	}

	// 3. 初始血量必须 > 0（有僵尸生成）
	if timer.WaveInitialHealthCs <= 0 {
		return false
	}

	// 4. 当前血量 <= 初始血量 × 阈值
	threshold := float64(timer.WaveInitialHealthCs) * timer.HealthTriggerThreshold
	if float64(currentHealth) > threshold {
		return false
	}

	// 触发血量加速刷新
	oldCountdown := timer.CountdownCs
	timer.CountdownCs = AcceleratedRefreshCountdownCs
	timer.AccumulatedCs = 0
	timer.HealthAccelerationTriggered = true

	log.Printf("[WaveTimingSystem] 🩸 Health-triggered acceleration! Countdown: %d cs → %d cs (health: %d/%d, threshold: %.0f)",
		oldCountdown, AcceleratedRefreshCountdownCs, currentHealth, timer.WaveInitialHealthCs, threshold)

	return true
}
