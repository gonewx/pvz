package systems

import (
	"github.com/gonewx/pvz/pkg/config"
)

// DifficultyEngine 难度引擎
// 负责计算轮数和级别容量上限，为僵尸生成系统提供难度数据支持
type DifficultyEngine struct {
	zombieStats *config.ZombieStatsConfig
}

// NewDifficultyEngine 创建新的难度引擎实例
func NewDifficultyEngine(zombieStats *config.ZombieStatsConfig) *DifficultyEngine {
	return &DifficultyEngine{
		zombieStats: zombieStats,
	}
}

// CalculateRoundNumber 计算轮数
// 公式: RoundNumber = TotalCompletedFlags / 2 - 1
// 参数:
//
//	totalCompletedFlags - 已完成的旗帜总数
//
// 返回:
//
//	轮数（可能为负数，表示一周目早期关卡）
func (d *DifficultyEngine) CalculateRoundNumber(totalCompletedFlags int) int {
	return totalCompletedFlags/2 - 1
}

// CalculateLevelCapacity 计算级别容量上限
// 公式: CapacityCap = int(int((CurrentWaveNum + RoundNumber * WavesPerRound) * 0.8) / 2) + 1
// 大波（旗帜波）容量 × 2.5 并向零取整
// 参数:
//
//	waveNum - 当前波次号（从1开始）
//	roundNumber - 轮数
//	wavesPerRound - 每轮波次数（默认20）
//	isFlagWave - 是否为旗帜波（大波）
//
// 返回:
//
//	级别容量上限
func (d *DifficultyEngine) CalculateLevelCapacity(waveNum, roundNumber, wavesPerRound int, isFlagWave bool) int {
	base := int(int(float64(waveNum+roundNumber*wavesPerRound)*0.8)/2) + 1
	if isFlagWave {
		// 大波（旗帜波）容量 × 2.5 并向零取整
		return int(float64(base) * 2.5)
	}
	return base
}

// GetZombieLevel 获取指定僵尸类型的级别
// 如果僵尸类型不存在于配置中，返回默认级别 1
func (d *DifficultyEngine) GetZombieLevel(zombieType string) int {
	if d.zombieStats == nil {
		return 1
	}
	return d.zombieStats.GetZombieLevel(zombieType)
}

// CalculateTotalLevel 计算一组僵尸的级别总和
func (d *DifficultyEngine) CalculateTotalLevel(zombieTypes []string) int {
	total := 0
	for _, zombieType := range zombieTypes {
		total += d.GetZombieLevel(zombieType)
	}
	return total
}

// ValidateWaveCapacity 验证一波僵尸是否符合级别容量限制
// 参数:
//
//	zombieTypes - 本波僵尸类型列表
//	capacityCap - 级别容量上限
//
// 返回:
//
//	true 如果级别总和不超过容量上限，否则 false
func (d *DifficultyEngine) ValidateWaveCapacity(zombieTypes []string, capacityCap int) bool {
	totalLevel := d.CalculateTotalLevel(zombieTypes)
	return totalLevel <= capacityCap
}

// GetZombieStats 获取僵尸属性配置
func (d *DifficultyEngine) GetZombieStats() *config.ZombieStatsConfig {
	return d.zombieStats
}
