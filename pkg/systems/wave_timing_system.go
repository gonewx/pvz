package systems

import (
	"log"
	"math/rand"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// 波次计时常量（厘秒）
const (
	// FirstWaveDelayCs 非首次游戏开场倒计时（厘秒）
	// 原版：600cs = 6秒，从 599 递减到 1 触发
	FirstWaveDelayCs = 599

	// RegularWaveBaseDelayCs 常规波次基础延迟（厘秒）
	// 原版：2500cs = 25秒
	RegularWaveBaseDelayCs = 2500

	// RegularWaveRandomDelayCs 常规波次随机延迟范围（厘秒）
	// 原版：rand(600)，范围 [0, 600)
	RegularWaveRandomDelayCs = 600

	// ========== Story 17.7: 旗帜波特殊计时常量 ==========

	// FlagWavePrefixDelayCs 旗帜波前一波延迟（厘秒）
	// 原版：4500cs = 45秒
	FlagWavePrefixDelayCs = 4500

	// FinalWaveDelayCs 最终波延迟（厘秒）
	// 原版：5500cs = 55秒
	FinalWaveDelayCs = 5500

	// FlagWavePhase4DurationCs Phase 4 停留时间（厘秒）
	// 红字警告在倒计时=4时停留 725cs
	FlagWavePhase4DurationCs = 725

	// FlagWarningTotalDurationCs 红字总显示时间（厘秒）
	// 约 750cs（从 Phase 5 到触发下一波）
	FlagWarningTotalDurationCs = 750

	// FinalWaveTextDurationCs 白字 "FINAL WAVE" 显示时间（厘秒）
	// 白字显示 500cs 后检查胜利条件
	FinalWaveTextDurationCs = 500

	// AcceleratedRefreshMinTimeCs 加速刷新最小刷出时间（厘秒）
	// 刷出 > 401cs 后才能触发加速刷新
	AcceleratedRefreshMinTimeCs = 401

	// AcceleratedRefreshCountdownCs 加速后倒计时设置值（厘秒）
	// 加速刷新触发后，将倒计时设为 200cs
	AcceleratedRefreshCountdownCs = 200
)

// WaveTimingSystem 波次计时系统
//
// 职责：
//   - 管理波次刷新计时器
//   - 处理开场倒计时逻辑（首波 vs 非首波）
//   - 计算并设置常规波次延迟
//   - 支持暂停/恢复
//
// 架构说明：
//   - 使用 WaveTimerComponent 存储状态
//   - 通过 WaveTriggered 标志与 LevelSystem 通信
//   - 遵循零耦合原则：不直接调用其他系统
type WaveTimingSystem struct {
	entityManager *ecs.EntityManager
	gameState     *game.GameState
	levelConfig   *config.LevelConfig

	// timerEntityID 计时器组件所在的实体ID
	timerEntityID ecs.EntityID

	// verbose 是否输出详细日志
	verbose bool
}

// NewWaveTimingSystem 创建波次计时系统
//
// 参数：
//   - em: 实体管理器
//   - gs: 游戏状态单例
//   - levelConfig: 关卡配置
//
// 返回：
//   - *WaveTimingSystem: 波次计时系统实例
func NewWaveTimingSystem(em *ecs.EntityManager, gs *game.GameState, levelConfig *config.LevelConfig) *WaveTimingSystem {
	system := &WaveTimingSystem{
		entityManager: em,
		gameState:     gs,
		levelConfig:   levelConfig,
		verbose:       false,
	}

	// 创建计时器实体
	system.createTimerEntity()

	return system
}

// createTimerEntity 创建计时器组件实体
func (s *WaveTimingSystem) createTimerEntity() {
	// 创建实体
	entityID := s.entityManager.CreateEntity()
	s.timerEntityID = entityID

	// 计算总波次数
	totalWaves := 0
	if s.levelConfig != nil {
		totalWaves = len(s.levelConfig.Waves)
	}

	// 添加计时器组件
	timerComp := &components.WaveTimerComponent{
		CountdownCs:       0,
		AccumulatedCs:     0,
		IsFirstWave:       true,
		CurrentWaveIndex:  0,
		TotalWaves:        totalWaves,
		IsPaused:          false,
		WaveStartedAt:     0,
		LastRefreshTimeCs: 0,
		WaveTriggered:     false,
	}

	ecs.AddComponent(s.entityManager, entityID, timerComp)

	log.Printf("[WaveTimingSystem] Created timer entity (ID: %d), total waves: %d", entityID, totalWaves)
}

// InitializeTimer 初始化计时器
//
// 根据是否为首次游戏设置不同的初始倒计时：
//   - 首次选卡后：立即开始第一波（CountdownCs = 0）
//   - 非首次：600 厘秒（6秒）倒计时
//
// 已废弃：请使用 InitializeTimerWithDelay，支持从关卡配置读取首波延迟
//
// 参数：
//   - isFirstPlaythrough: 是否为首次游戏（一周目首次）
func (s *WaveTimingSystem) InitializeTimer(isFirstPlaythrough bool) {
	timer := s.getTimerComponent()
	if timer == nil {
		log.Printf("[WaveTimingSystem] ERROR: Timer component not found")
		return
	}

	if isFirstPlaythrough {
		// 首次选卡后：立即触发第一波
		timer.CountdownCs = 0
		timer.IsFirstWave = true
		log.Printf("[WaveTimingSystem] Initialized for first playthrough: immediate first wave")
	} else {
		// 非首次：设置开场倒计时
		timer.CountdownCs = FirstWaveDelayCs
		timer.IsFirstWave = false
		timer.LastRefreshTimeCs = FirstWaveDelayCs
		log.Printf("[WaveTimingSystem] Initialized for subsequent playthrough: %d cs delay", FirstWaveDelayCs)
	}

	timer.CurrentWaveIndex = 0
	timer.WaveTriggered = false
	timer.AccumulatedCs = 0
}

// InitializeTimerWithDelay 使用关卡配置初始化计时器
//
// Story 17.6: delay 字段已移除，使用默认首波延迟
// 首次游戏：20 秒延迟（让玩家有时间布置防线）
// 非首次：6 秒延迟
//
// 参数：
//   - isFirstPlaythrough: 是否为首次游戏（一周目首次）
//   - levelConfig: 关卡配置（保留参数用于将来扩展）
func (s *WaveTimingSystem) InitializeTimerWithDelay(isFirstPlaythrough bool, levelConfig *config.LevelConfig) {
	timer := s.getTimerComponent()
	if timer == nil {
		log.Printf("[WaveTimingSystem] ERROR: Timer component not found")
		return
	}

	// Story 17.6: delay 字段已从 WaveConfig 移除，使用默认延迟
	var firstWaveDelaySec float64
	if isFirstPlaythrough {
		// 首次游戏默认 20 秒延迟（让玩家有时间布置防线）
		firstWaveDelaySec = 20.0
	} else {
		// 非首次游戏默认 6 秒延迟
		firstWaveDelaySec = 6.0
	}

	// 转换为厘秒
	firstWaveDelayCs := int(firstWaveDelaySec * 100)

	timer.CountdownCs = firstWaveDelayCs
	timer.IsFirstWave = true
	timer.LastRefreshTimeCs = firstWaveDelayCs
	timer.CurrentWaveIndex = 0
	timer.WaveTriggered = false
	timer.AccumulatedCs = 0

	log.Printf("[WaveTimingSystem] Initialized: %d cs (%.1f sec) delay for first wave (firstPlaythrough=%v)",
		firstWaveDelayCs, firstWaveDelaySec, isFirstPlaythrough)
}

// Update 更新计时器
//
// 执行流程：
//  1. 检查暂停状态
//  2. 将 deltaTime（秒）转换为厘秒
//  3. 递减倒计时
//  4. Story 17.7: 处理红字警告阶段（旗帜波前）
//  5. Story 17.7: 处理最终波白字逻辑
//  6. 当倒计时 <= 1 时触发下一波
//
// 参数：
//   - deltaTime: 自上一帧以来经过的时间（秒）
func (s *WaveTimingSystem) Update(deltaTime float64) {
	timer := s.getTimerComponent()
	if timer == nil {
		return
	}

	// 注意：不在这里重置 WaveTriggered 标志
	// WaveTriggered 只在 ClearWaveTriggered() 中重置
	// 这确保 TriggerNextWaveImmediately() 设置的标志能被 LevelSystem 正确处理

	// 暂停时不更新
	if timer.IsPaused {
		return
	}

	// Story 17.7: 处理最终波白字显示逻辑
	if timer.FinalWaveTextActive {
		s.updateFinalWaveText(deltaTime)
		return // 白字显示期间不更新其他计时
	}

	// 检查是否已完成所有波次
	if timer.CurrentWaveIndex >= timer.TotalWaves {
		return
	}

	// 将 deltaTime（秒）转换为厘秒并累积
	deltaCsFloat := deltaTime * 100
	timer.AccumulatedCs += deltaCsFloat

	// 取整数部分递减，保留小数部分
	deltaCsInt := int(timer.AccumulatedCs)
	if deltaCsInt > 0 {
		timer.AccumulatedCs -= float64(deltaCsInt)

		// Story 17.7: 处理红字警告阶段
		if timer.FlagWaveCountdownPhase > 0 {
			s.updateFlagWaveWarningPhase(deltaCsInt)
			return // 红字警告阶段不递减倒计时
		}

		timer.CountdownCs -= deltaCsInt

		// 更新波次已过时间（用于加速刷新）
		timer.WaveElapsedCs += deltaCsInt

		if s.verbose {
			log.Printf("[WaveTimingSystem] Countdown: %d cs (delta: %d cs)", timer.CountdownCs, deltaCsInt)
		}
	}

	// Story 17.7: 检查是否进入红字警告阶段
	if timer.IsFlagWaveApproaching && !timer.HugeWaveWarningTriggered {
		s.checkFlagWaveWarningPhase()
	}

	// 检查是否触发下一波
	if timer.CountdownCs <= 1 && timer.FlagWaveCountdownPhase == 0 {
		s.triggerNextWave()
	}
}

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
		// Phase 4: 红字停留阶段（725cs）
		if timer.FlagWavePhaseTimeCs >= FlagWavePhase4DurationCs {
			// 停留结束，触发旗帜波
			log.Printf("[WaveTimingSystem] Huge wave warning Phase 4 complete, triggering flag wave")
			timer.FlagWaveCountdownPhase = 0
			timer.FlagWavePhaseTimeCs = 0
			timer.IsFlagWaveApproaching = false
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

// updateFinalWaveText 更新最终波白字显示
//
// Story 17.7: 白字 "FINAL WAVE" 显示 500cs 后设置完成标志
//
// 参数：
//   - deltaTime: 自上一帧以来经过的时间（秒）
func (s *WaveTimingSystem) updateFinalWaveText(deltaTime float64) {
	timer := s.getTimerComponent()
	if timer == nil {
		return
	}

	// 累积白字显示时间
	deltaCsFloat := deltaTime * 100
	timer.FinalWaveTextTimeCs += int(deltaCsFloat)

	if timer.FinalWaveTextTimeCs >= FinalWaveTextDurationCs {
		log.Printf("[WaveTimingSystem] Final wave text display complete (500cs)")
		// 注意：此处不重置 FinalWaveTextActive，让 LevelSystem 检查胜利条件
	}
}

// ActivateFinalWaveText 激活最终波白字显示
//
// Story 17.7: 当最终波倒计时减至 0 时调用
func (s *WaveTimingSystem) ActivateFinalWaveText() {
	timer := s.getTimerComponent()
	if timer == nil {
		return
	}

	timer.FinalWaveTextActive = true
	timer.FinalWaveTextTimeCs = 0
	log.Printf("[WaveTimingSystem] Final wave text activated!")
}

// IsFinalWaveTextComplete 检查白字显示是否完成
//
// Story 17.7: 供 LevelSystem 检查胜利条件
//
// 返回：
//   - bool: true 表示白字显示已完成 500cs
func (s *WaveTimingSystem) IsFinalWaveTextComplete() bool {
	timer := s.getTimerComponent()
	if timer == nil {
		return false
	}

	return timer.FinalWaveTextActive && timer.FinalWaveTextTimeCs >= FinalWaveTextDurationCs
}

// IsFinalWaveTextActive 检查白字是否正在显示
//
// Story 17.7: 供 UI 渲染系统检查是否显示白字
//
// 返回：
//   - bool: true 表示白字正在显示
func (s *WaveTimingSystem) IsFinalWaveTextActive() bool {
	timer := s.getTimerComponent()
	if timer == nil {
		return false
	}

	return timer.FinalWaveTextActive
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

	// 递增波次索引（下一次会触发下一波）
	timer.CurrentWaveIndex++

	// 如果还有后续波次，设置下一波倒计时
	if timer.CurrentWaveIndex < timer.TotalWaves {
		s.SetNextWaveCountdown()
	}
}

// SetNextWaveCountdown 设置下一波倒计时
//
// Story 17.7: 根据下一波类型设置不同的倒计时：
//   - 旗帜波前一波：4500cs（45秒）
//   - 最终波：5500cs（55秒）
//   - 常规波：2500 + rand(600) 厘秒（25-31秒）
func (s *WaveTimingSystem) SetNextWaveCountdown() {
	timer := s.getTimerComponent()
	if timer == nil {
		return
	}

	// 重置波次已过时间（用于加速刷新判定）
	timer.WaveElapsedCs = 0

	// Story 17.7: 根据下一波类型设置倒计时
	nextWaveIndex := timer.CurrentWaveIndex
	var countdown int
	var waveType string

	if s.isNextWaveFlagWave(nextWaveIndex) {
		// 旗帜波前一波：4500cs
		countdown = FlagWavePrefixDelayCs
		timer.IsFlagWaveApproaching = true
		timer.HugeWaveWarningTriggered = false
		waveType = "flag wave prefix"
	} else if s.isFinalWave(nextWaveIndex) {
		// 最终波：5500cs
		countdown = FinalWaveDelayCs
		timer.IsFinalWave = true
		waveType = "final wave"
	} else {
		// 常规波：2500 + rand(600)
		countdown = RegularWaveBaseDelayCs + rand.Intn(RegularWaveRandomDelayCs)
		timer.IsFlagWaveApproaching = false
		timer.IsFinalWave = false
		waveType = "regular wave"
	}

	timer.CountdownCs = countdown
	timer.LastRefreshTimeCs = countdown
	timer.AccumulatedCs = 0

	log.Printf("[WaveTimingSystem] Next wave countdown set: %d cs (%.2fs) [%s, wave %d]",
		countdown, float64(countdown)/100, waveType, nextWaveIndex+1)
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

// Pause 暂停计时器
func (s *WaveTimingSystem) Pause() {
	timer := s.getTimerComponent()
	if timer == nil {
		return
	}

	timer.IsPaused = true
	log.Printf("[WaveTimingSystem] Timer paused at %d cs", timer.CountdownCs)
}

// Resume 恢复计时器
func (s *WaveTimingSystem) Resume() {
	timer := s.getTimerComponent()
	if timer == nil {
		return
	}

	timer.IsPaused = false
	log.Printf("[WaveTimingSystem] Timer resumed at %d cs", timer.CountdownCs)
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

// SetVerbose 设置是否输出详细日志
func (s *WaveTimingSystem) SetVerbose(verbose bool) {
	s.verbose = verbose
}

// getTimerComponent 获取计时器组件
func (s *WaveTimingSystem) getTimerComponent() *components.WaveTimerComponent {
	timer, ok := ecs.GetComponent[*components.WaveTimerComponent](s.entityManager, s.timerEntityID)
	if !ok {
		return nil
	}
	return timer
}

// GetTimerEntityID 获取计时器实体ID（用于测试）
func (s *WaveTimingSystem) GetTimerEntityID() ecs.EntityID {
	return s.timerEntityID
}

// ========== Story 17.8: 血量计算与追踪 ==========

// CalculateZombieEffectiveHealth 计算僵尸有效血量
//
// Story 17.8: 血量计算公式
// 有效血量 = 本体血量 + I类饰品血量 + 0.20 × II类饰品血量
//
// I类饰品: 路障(370), 铁桶(1100), 橄榄球帽, 雪橇车, 气球, 矿工帽, 僵尸坚果
// II类饰品: 报纸, 铁栅门, 扶梯
//
// 参数:
//   - baseHealth: 本体血量
//   - tier1Health: I类饰品血量
//   - tier2Health: II类饰品血量
//
// 返回:
//   - int: 有效血量
func CalculateZombieEffectiveHealth(baseHealth, tier1Health, tier2Health int) int {
	return baseHealth + tier1Health + int(float64(tier2Health)*0.20)
}

// GetZombieTypeEffectiveHealth 从配置获取僵尸类型的有效血量
//
// Story 17.8: 根据僵尸类型查询配置，计算有效血量
//
// 参数:
//   - zombieStatsConfig: 僵尸属性配置
//   - zombieType: 僵尸类型名称
//
// 返回:
//   - int: 有效血量（类型不存在时返回 270，即默认普僵血量）
func GetZombieTypeEffectiveHealth(zombieStatsConfig *config.ZombieStatsConfig, zombieType string) int {
	if zombieStatsConfig == nil {
		return 270 // 默认普僵血量
	}

	stats, ok := zombieStatsConfig.GetZombieStats(zombieType)
	if !ok {
		return 270 // 未知类型使用默认值
	}

	return CalculateZombieEffectiveHealth(stats.BaseHealth, stats.Tier1AccessoryHealth, stats.Tier2AccessoryHealth)
}

// ZombieSpawnInfo 描述单个僵尸生成信息
// 用于 InitializeWaveHealth 计算波次总血量
type ZombieSpawnInfo struct {
	Type  string // 僵尸类型
	Count int    // 数量
}

// InitializeWaveHealth 初始化波次血量追踪
//
// Story 17.8: 在波次开始时调用，计算并记录本波僵尸总血量
//
// 参数:
//   - zombieList: 本波僵尸列表（类型和数量）
//   - zombieStatsConfig: 僵尸属性配置
func (s *WaveTimingSystem) InitializeWaveHealth(zombieList []ZombieSpawnInfo, zombieStatsConfig *config.ZombieStatsConfig) {
	timer := s.getTimerComponent()
	if timer == nil {
		return
	}

	// 计算本波僵尸总有效血量
	totalHealth := 0
	for _, zombie := range zombieList {
		effectiveHealth := GetZombieTypeEffectiveHealth(zombieStatsConfig, zombie.Type)
		totalHealth += effectiveHealth * zombie.Count
	}

	// 设置初始血量和当前血量
	timer.WaveInitialHealthCs = totalHealth
	timer.WaveCurrentHealthCs = totalHealth

	// 随机生成血量触发阈值 [0.50, 0.65]
	timer.HealthTriggerThreshold = 0.50 + rand.Float64()*0.15

	// 重置血量加速触发标志
	timer.HealthAccelerationTriggered = false

	log.Printf("[WaveTimingSystem] Wave health initialized: total=%d, threshold=%.2f (%.0f hp)",
		totalHealth, timer.HealthTriggerThreshold, float64(totalHealth)*timer.HealthTriggerThreshold)
}

// UpdateWaveCurrentHealth 更新波次当前血量
//
// Story 17.8: 由 LevelSystem 或外部系统调用，更新当前血量
//
// 参数:
//   - currentHealth: 当前僵尸总血量
func (s *WaveTimingSystem) UpdateWaveCurrentHealth(currentHealth int) {
	timer := s.getTimerComponent()
	if timer == nil {
		return
	}

	timer.WaveCurrentHealthCs = currentHealth
}

// GetWaveHealthInfo 获取波次血量信息（用于调试和测试）
//
// Story 17.8: 返回当前波次的血量追踪信息
//
// 返回:
//   - initialHealth: 初始总血量
//   - currentHealth: 当前总血量
//   - threshold: 血量触发阈值
//   - triggered: 是否已触发血量加速
func (s *WaveTimingSystem) GetWaveHealthInfo() (initialHealth, currentHealth int, threshold float64, triggered bool) {
	timer := s.getTimerComponent()
	if timer == nil {
		return 0, 0, 0, false
	}

	return timer.WaveInitialHealthCs, timer.WaveCurrentHealthCs, timer.HealthTriggerThreshold, timer.HealthAccelerationTriggered
}

// CalculateCurrentWaveHealth 计算当前波次僵尸的实时总血量
//
// Story 17.8: 遍历所有本波僵尸，累加 Health + Armor
// 由 LevelSystem 调用以获取实时血量
//
// 参数:
//   - em: 实体管理器
//   - currentWaveIndex: 当前波次索引（0-based）
//
// 返回:
//   - int: 当前僵尸总血量（health + armor）
func CalculateCurrentWaveHealth(em *ecs.EntityManager, currentWaveIndex int) int {
	totalHealth := 0

	// 遍历所有具有 ZombieWaveStateComponent 的实体
	entities := ecs.GetEntitiesWith1[*components.ZombieWaveStateComponent](em)
	for _, entity := range entities {
		waveState, ok := ecs.GetComponent[*components.ZombieWaveStateComponent](em, entity)
		if !ok {
			continue
		}

		// 筛选本波僵尸
		if waveState.WaveIndex != currentWaveIndex {
			continue
		}

		// 累加血量
		health, hasHealth := ecs.GetComponent[*components.HealthComponent](em, entity)
		if hasHealth && health.CurrentHealth > 0 {
			totalHealth += health.CurrentHealth
		}

		// 累加护甲
		armor, hasArmor := ecs.GetComponent[*components.ArmorComponent](em, entity)
		if hasArmor && armor.CurrentArmor > 0 {
			totalHealth += armor.CurrentArmor
		}
	}

	return totalHealth
}

