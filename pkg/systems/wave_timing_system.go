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

// Update 更新计时器
//
// 执行流程：
//  1. 检查暂停状态
//  2. 将 deltaTime（秒）转换为厘秒
//  3. 递减倒计时
//  4. 当倒计时 <= 1 时触发下一波
//
// 参数：
//   - deltaTime: 自上一帧以来经过的时间（秒）
func (s *WaveTimingSystem) Update(deltaTime float64) {
	timer := s.getTimerComponent()
	if timer == nil {
		return
	}

	// 重置触发标志
	timer.WaveTriggered = false

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
		timer.CountdownCs -= deltaCsInt

		if s.verbose {
			log.Printf("[WaveTimingSystem] Countdown: %d cs (delta: %d cs)", timer.CountdownCs, deltaCsInt)
		}
	}

	// 检查是否触发下一波
	if timer.CountdownCs <= 1 {
		s.triggerNextWave()
	}
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
// 计算公式：2500 + rand.Intn(600) 厘秒
// 范围：2500-3099 厘秒（25-31秒）
func (s *WaveTimingSystem) SetNextWaveCountdown() {
	timer := s.getTimerComponent()
	if timer == nil {
		return
	}

	// 计算随机延迟：2500 + rand(600)
	countdown := RegularWaveBaseDelayCs + rand.Intn(RegularWaveRandomDelayCs)
	timer.CountdownCs = countdown
	timer.LastRefreshTimeCs = countdown
	timer.AccumulatedCs = 0

	log.Printf("[WaveTimingSystem] Next wave countdown set: %d cs (%.2fs)", countdown, float64(countdown)/100)
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

