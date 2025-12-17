package systems

import (
	"log"
)

// ============================================================================
// 旗帜波警告处理
// ============================================================================

// updateFlagWaveWarningPhase 更新红字警告阶段
//
// Story 17.7: 处理红字警告的阶段转换
//   - Phase 5: 显示红字（短暂）
//   - Phase 4: 停留 725cs
//   - Phase 结束: 触发旗帜波
//
// 参数：
//   - deltaCsInt: 本帧经过的厘秒数
func (s *WaveTimingSystem) updateFlagWaveWarningPhase(deltaCsInt int) {
	timer := s.getTimerComponent()
	if timer == nil {
		return
	}

	timer.FlagWavePhaseTimeCs += deltaCsInt

	switch timer.FlagWaveCountdownPhase {
	case 5:
		// Phase 5: 红字显示阶段，很快转到 Phase 4
		// 当倒计时从 5 减到 4 时转换
		if timer.CountdownCs <= 4 {
			timer.FlagWaveCountdownPhase = 4
			timer.FlagWavePhaseTimeCs = 0
			log.Printf("[WaveTimingSystem] Huge wave warning entering Phase 4 (725cs hold)")
		}
	case 4:
		// Phase 4: 红字停留阶段
		// 每秒输出一次日志，避免玩家以为卡死
		if timer.FlagWavePhaseTimeCs%100 == 0 {
			log.Printf("[WaveTimingSystem] Huge wave warning Phase 4 holding... (%d/%d cs)",
				timer.FlagWavePhaseTimeCs, FlagWavePhase4DurationCs)
		}

		if timer.FlagWavePhaseTimeCs >= FlagWavePhase4DurationCs {
			// 停留结束，触发旗帜波
			log.Printf("[WaveTimingSystem] Huge wave warning Phase 4 complete, triggering flag wave")
			timer.FlagWaveCountdownPhase = 0
			timer.FlagWavePhaseTimeCs = 0
			timer.IsFlagWaveApproaching = false
			// 重置倒计时，防止 Update 中再次触发
			timer.CountdownCs = 9999
			s.triggerNextWave()
		}
	}
}

// checkFlagWaveWarningPhase 检查是否进入红字警告阶段
//
// Story 17.7: 当倒计时 = 5 时进入 Phase 5，显示红字警告
func (s *WaveTimingSystem) checkFlagWaveWarningPhase() {
	timer := s.getTimerComponent()
	if timer == nil {
		return
	}

	// 倒计时 <= 5 时进入 Phase 5
	if timer.CountdownCs <= 5 && timer.FlagWaveCountdownPhase == 0 {
		timer.FlagWaveCountdownPhase = 5
		timer.FlagWavePhaseTimeCs = 0
		timer.HugeWaveWarningTriggered = true
		log.Printf("[WaveTimingSystem] Huge wave warning triggered! Entering Phase 5")
	}
}

// GetFlagWaveWarningPhase 获取当前红字警告阶段
//
// Story 17.7: 供 UI 渲染系统检查是否显示红字
//
// 返回：
//   - int: 当前阶段（0=无, 5=显示红字, 4=停留）
func (s *WaveTimingSystem) GetFlagWaveWarningPhase() int {
	timer := s.getTimerComponent()
	if timer == nil {
		return 0
	}

	return timer.FlagWaveCountdownPhase
}

// IsHugeWaveWarningActive 检查红字警告是否激活
//
// Story 17.7: 供 UI 渲染系统检查是否显示红字
//
// 返回：
//   - bool: true 表示红字警告正在显示
func (s *WaveTimingSystem) IsHugeWaveWarningActive() bool {
	timer := s.getTimerComponent()
	if timer == nil {
		return false
	}

	return timer.FlagWaveCountdownPhase > 0
}

// ========== Bug Fix: 警告队列管理方法 ==========

// GetCurrentWarning 获取当前待显示的警告类型
//
// 返回：
//   - string: 当前警告类型 ("huge_wave", "final_wave", 或 "" 表示无警告)
func (s *WaveTimingSystem) GetCurrentWarning() string {
	timer := s.getTimerComponent()
	if timer == nil {
		return ""
	}

	if timer.CurrentWarningIndex >= len(timer.PendingWarnings) {
		return ""
	}

	return timer.PendingWarnings[timer.CurrentWarningIndex]
}

// AdvanceWarningQueue 推进警告队列到下一个警告
//
// 当一个警告动画播放完成后调用，将队列索引+1
func (s *WaveTimingSystem) AdvanceWarningQueue() {
	timer := s.getTimerComponent()
	if timer == nil {
		return
	}

	if timer.CurrentWarningIndex < len(timer.PendingWarnings) {
		oldWarning := timer.PendingWarnings[timer.CurrentWarningIndex]
		timer.CurrentWarningIndex++
		log.Printf("[WaveTimingSystem] Warning queue advanced: %s completed (index %d -> %d)",
			oldWarning, timer.CurrentWarningIndex-1, timer.CurrentWarningIndex)
	}
}

// HasPendingWarnings 检查是否还有待显示的警告
//
// 返回：
//   - bool: true 表示还有警告需要显示
func (s *WaveTimingSystem) HasPendingWarnings() bool {
	timer := s.getTimerComponent()
	if timer == nil {
		return false
	}

	return timer.CurrentWarningIndex < len(timer.PendingWarnings)
}

// GetPendingWarnings 获取待显示的警告列表（用于调试）
//
// 返回：
//   - []string: 所有待显示的警告类型
//   - int: 当前警告索引
func (s *WaveTimingSystem) GetPendingWarnings() ([]string, int) {
	timer := s.getTimerComponent()
	if timer == nil {
		return nil, 0
	}

	return timer.PendingWarnings, timer.CurrentWarningIndex
}

// IsFinalWaveWarningPending 检查是否有最终波警告待显示
//
// 返回：
//   - bool: true 表示最终波警告待显示
func (s *WaveTimingSystem) IsFinalWaveWarningPending() bool {
	timer := s.getTimerComponent()
	if timer == nil {
		return false
	}

	for i := timer.CurrentWarningIndex; i < len(timer.PendingWarnings); i++ {
		if timer.PendingWarnings[i] == "final_wave" {
			return true
		}
	}
	return false
}
