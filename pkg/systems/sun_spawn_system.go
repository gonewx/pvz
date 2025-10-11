package systems

import (
	"math/rand"

	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
)

// SunSpawnSystem 管理阳光的定时生成
type SunSpawnSystem struct {
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager
	spawnTimer      float64 // 当前计时器
	spawnInterval   float64 // 生成间隔(秒)
	minX            float64 // 阳光生成的最小X坐标
	maxX            float64 // 阳光生成的最大X坐标
	minTargetY      float64 // 阳光落地的最小Y坐标
	maxTargetY      float64 // 阳光落地的最大Y坐标
}

// NewSunSpawnSystem 创建一个新的阳光生成系统
// 参数:
//   - em: EntityManager 实例
//   - rm: ResourceManager 实例
//   - minX, maxX: 阳光生成的水平范围
//   - minTargetY, maxTargetY: 阳光落地的垂直范围
func NewSunSpawnSystem(em *ecs.EntityManager, rm *game.ResourceManager, minX, maxX, minTargetY, maxTargetY float64) *SunSpawnSystem {
	return &SunSpawnSystem{
		entityManager:   em,
		resourceManager: rm,
		spawnTimer:      0,
		spawnInterval:   8.0, // 原版游戏机制: 每8秒生成一次
		minX:            minX,
		maxX:            maxX,
		minTargetY:      minTargetY,
		maxTargetY:      maxTargetY,
	}
}

// Update 更新阳光生成计时器
func (s *SunSpawnSystem) Update(deltaTime float64) {
	// 累加计时器
	s.spawnTimer += deltaTime

	// 检查是否到达生成间隔
	if s.spawnTimer >= s.spawnInterval {
		// 重置计时器
		s.spawnTimer = 0

		// 生成随机起始X坐标
		startX := s.minX + rand.Float64()*(s.maxX-s.minX)

		// 生成随机落地Y坐标
		targetY := s.minTargetY + rand.Float64()*(s.maxTargetY-s.minTargetY)

		// 创建阳光实体
		entities.NewSunEntity(s.entityManager, s.resourceManager, startX, targetY)
	}
}
