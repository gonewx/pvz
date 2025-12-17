package systems

import (
	"log"
	"math/rand"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
)

// ============================================================================
// 血量计算与状态恢复
// ============================================================================

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

// ========== 状态恢复 ==========

// RestoreState 从存档恢复波次计时状态
//
// Story 18.3: 存档恢复时同步波次计时系统状态
//
// 参数：
//   - currentWaveIndex: 当前波次索引（0-based，表示下一个要触发的波次）
//   - levelTime: 关卡已进行时间（秒）
//
// 恢复内容：
//   - 设置 CurrentWaveIndex 为 currentWaveIndex（这是下一个要触发的波次）
//   - 设置下一波的倒计时
//   - 取消暂停状态
func (s *WaveTimingSystem) RestoreState(currentWaveIndex int, levelTime float64) {
	timer := s.getTimerComponent()
	if timer == nil {
		log.Printf("[WaveTimingSystem] ERROR: Timer component not found during restore")
		return
	}

	// currentWaveIndex 是"下一个要触发的波次索引"
	// 例如：如果 currentWaveIndex=3，表示波次 0,1,2 已触发，下一波是波次 3

	// 检查是否所有波次都已完成
	if currentWaveIndex >= timer.TotalWaves {
		// 所有波次已生成，不需要继续计时
		timer.CurrentWaveIndex = timer.TotalWaves
		timer.IsPaused = false
		log.Printf("[WaveTimingSystem] Restore: All waves already spawned (index=%d, total=%d)",
			currentWaveIndex, timer.TotalWaves)
		return
	}

	// 设置当前波次索引（下一个要触发的波次）
	timer.CurrentWaveIndex = currentWaveIndex
	timer.IsFirstWave = false

	// 设置已经过的时间（厘秒）
	timer.AccumulatedCs = levelTime * 100

	// 设置下一波倒计时
	s.SetNextWaveCountdown()

	// 取消暂停
	timer.IsPaused = false

	log.Printf("[WaveTimingSystem] Restore: Next wave=%d, countdown=%d cs, accumulated=%.0f cs",
		currentWaveIndex+1, timer.CountdownCs, timer.AccumulatedCs)
}

// RestoreTimerState 从存档恢复精确的计时器状态
//
// v6 新增：精确恢复计时器的内部状态，而不是重新计算
//
// 当使用 v6 存档时，可以精确恢复以下状态：
//   - CountdownCs: 当前倒计时（厘秒）
//   - WaveElapsedCs: 当前波次已过时间（厘秒）
//   - IsFlagWaveApproaching: 是否正在接近旗帜波
//   - IsFinalWave: 是否是最终波
//
// 这比 RestoreState 中的 SetNextWaveCountdown() 更精确，
// 因为 SetNextWaveCountdown() 会重新计算倒计时，而不是恢复保存时的精确值。
//
// 参数：
//   - countdownCs: 当前倒计时（厘秒）
//   - waveElapsedCs: 当前波次已过时间（厘秒）
//   - isFlagWaveApproaching: 是否正在接近旗帜波
//   - isFinalWave: 是否是最终波
func (s *WaveTimingSystem) RestoreTimerState(countdownCs, waveElapsedCs int, isFlagWaveApproaching, isFinalWave bool) {
	timer := s.getTimerComponent()
	if timer == nil {
		log.Printf("[WaveTimingSystem] ERROR: Timer component not found during RestoreTimerState")
		return
	}

	// 精确恢复计时器状态
	timer.CountdownCs = countdownCs
	timer.WaveElapsedCs = waveElapsedCs
	timer.IsFlagWaveApproaching = isFlagWaveApproaching
	timer.IsFinalWave = isFinalWave

	// 同步 LastRefreshTimeCs（用于进度条计算）
	timer.LastRefreshTimeCs = countdownCs

	log.Printf("[WaveTimingSystem] RestoreTimerState: CountdownCs=%d, WaveElapsedCs=%d, IsFlagWaveApproaching=%v, IsFinalWave=%v",
		countdownCs, waveElapsedCs, isFlagWaveApproaching, isFinalWave)
}
