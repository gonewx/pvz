package systems

import (
	"log"

	"github.com/gonewx/pvz/pkg/config"
)

// ========================================
// Story 17.8: 血量触发加速刷新
// ========================================

// SetZombieStatsConfig 设置僵尸属性配置
//
// Story 17.8: 用于血量计算
// 在关卡初始化时调用
//
// 参数:
//   - cfg: 僵尸属性配置
func (s *LevelSystem) SetZombieStatsConfig(cfg *config.ZombieStatsConfig) {
	s.zombieStatsConfig = cfg
}

// SetZombiePhysicsConfig 设置僵尸物理配置
//
// Story 17.9: 用于类型化进家判定
// 在关卡初始化时调用
//
// 参数:
//   - cfg: 僵尸物理配置
func (s *LevelSystem) SetZombiePhysicsConfig(cfg *config.ZombiePhysicsConfig) {
	s.zombiePhysics = cfg
}

// initializeWaveHealth 初始化波次血量追踪
//
// Story 17.8: 在波次激活后调用，计算并设置本波僵尸总血量
//
// 参数:
//   - waveIndex: 波次索引（0-based）
func (s *LevelSystem) initializeWaveHealth(waveIndex int) {
	// 不使用 WaveTimingSystem 时跳过
	if !s.useWaveTimingSystem || s.waveTimingSystem == nil {
		return
	}

	// 获取关卡配置
	levelConfig := s.gameState.CurrentLevel
	if levelConfig == nil {
		return
	}

	// 确保波次索引有效
	if waveIndex < 0 || waveIndex >= len(levelConfig.Waves) {
		return
	}

	// 从关卡配置中获取波次僵尸信息
	waveConfig := levelConfig.Waves[waveIndex]
	zombieList := s.extractZombieSpawnInfo(&waveConfig)

	// 调用 WaveTimingSystem 初始化血量
	s.waveTimingSystem.InitializeWaveHealth(zombieList, s.zombieStatsConfig)
}

// extractZombieSpawnInfo 从波次配置中提取僵尸生成信息
//
// Story 17.8: 支持新格式 ZombieGroup 和旧格式 OldZombies
//
// 参数:
//   - waveConfig: 波次配置
//
// 返回:
//   - []ZombieSpawnInfo: 僵尸生成信息列表
func (s *LevelSystem) extractZombieSpawnInfo(waveConfig *config.WaveConfig) []ZombieSpawnInfo {
	var result []ZombieSpawnInfo

	// 处理新格式 ZombieGroup
	for _, group := range waveConfig.Zombies {
		result = append(result, ZombieSpawnInfo{
			Type:  group.Type,
			Count: group.Count,
		})
	}

	// 处理旧格式 OldZombies（向后兼容）
	for _, spawn := range waveConfig.OldZombies {
		count := spawn.Count
		if count == 0 {
			count = 1 // 默认生成 1 只
		}
		result = append(result, ZombieSpawnInfo{
			Type:  spawn.Type,
			Count: count,
		})
	}

	return result
}

// DebugWaveHealthInfo 输出波次血量调试信息
//
// Story 17.8: 调试辅助函数
func (s *LevelSystem) DebugWaveHealthInfo() {
	if s.waveTimingSystem == nil {
		log.Printf("[LevelSystem] WaveTimingSystem not initialized")
		return
	}

	initialHealth, currentHealth, threshold, triggered := s.waveTimingSystem.GetWaveHealthInfo()
	log.Printf("[LevelSystem] Wave Health: initial=%d, current=%d, threshold=%.2f, triggered=%v",
		initialHealth, currentHealth, threshold, triggered)
}
