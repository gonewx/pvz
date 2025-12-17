package systems

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
)

// ============================================================================
// 波次计时系统核心（WaveTimingSystem）
// ============================================================================

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
	// 红字警告在倒计时=4时停留。原设定 725cs 可能过长导致玩家以为卡死
	// 调整为 400cs (4秒)
	FlagWavePhase4DurationCs = 400

	// FlagWarningTotalDurationCs 红字总显示时间（厘秒）
	// 约 450cs
	FlagWarningTotalDurationCs = 450

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
// 关卡配置可通过 FirstWaveDelay 字段覆盖默认值：
//   - 设置为 0 表示立即开始（传送带/保龄球关卡）
//   - 设置为正数表示指定延迟时间（秒）
//   - 未设置（nil）表示使用默认值
//
// 参数：
//   - isFirstPlaythrough: 是否为首次游戏（一周目首次）
//   - levelConfig: 关卡配置
func (s *WaveTimingSystem) InitializeTimerWithDelay(isFirstPlaythrough bool, levelConfig *config.LevelConfig) {
	timer := s.getTimerComponent()
	if timer == nil {
		log.Printf("[WaveTimingSystem] ERROR: Timer component not found")
		return
	}

	// 检查关卡配置是否指定了首波延迟
	var firstWaveDelaySec float64
	if levelConfig != nil && levelConfig.FirstWaveDelay != nil {
		// 使用关卡配置的首波延迟
		firstWaveDelaySec = *levelConfig.FirstWaveDelay
		log.Printf("[WaveTimingSystem] Using level-configured first wave delay: %.1f sec", firstWaveDelaySec)
	} else if isFirstPlaythrough {
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

			// 如果在 Phase 4 (Hold)，则不递减倒计时（保持波次不触发）
			if timer.FlagWaveCountdownPhase == 4 {
				return
			}
			// Phase 5 (Red Text) 需要继续递减倒计时，以便转换到 Phase 4
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
