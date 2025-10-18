package systems

import (
	"log"
	"math/rand"

	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
)

// SunSpawnSystem 管理阳光的定时生成
type SunSpawnSystem struct {
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager
	reanimSystem    *ReanimSystem // Story 8.2 QA修复：用于初始化阳光动画
	spawnTimer      float64       // 当前计时器
	spawnInterval   float64       // 生成间隔(秒)
	minX            float64       // 阳光生成的最小X坐标
	maxX            float64       // 阳光生成的最大X坐标
	minTargetY      float64       // 阳光落地的最小Y坐标
	maxTargetY      float64       // 阳光落地的最大Y坐标
	enabled         bool          // Story 8.2: 是否启用自动生成（教学关卡初始禁用）
}

// NewSunSpawnSystem 创建一个新的阳光生成系统
// 参数:
//   - em: EntityManager 实例
//   - rm: ResourceManager 实例
//   - rs: ReanimSystem 实例（用于初始化阳光动画）
//   - minX, maxX: 阳光生成的水平范围
//   - minTargetY, maxTargetY: 阳光落地的垂直范围
func NewSunSpawnSystem(em *ecs.EntityManager, rm *game.ResourceManager, rs *ReanimSystem, minX, maxX, minTargetY, maxTargetY float64) *SunSpawnSystem {
	log.Printf("[SunSpawnSystem] Initialized with interval=%.1fs, area=(%.0f-%.0f, %.0f-%.0f)",
		8.0, minX, maxX, minTargetY, maxTargetY)
	return &SunSpawnSystem{
		entityManager:   em,
		resourceManager: rm,
		reanimSystem:    rs,
		spawnTimer:      0,
		spawnInterval:   8.0, // 原版游戏机制: 每8秒生成一次
		minX:            minX,
		maxX:            maxX,
		minTargetY:      minTargetY,
		maxTargetY:      maxTargetY,
		enabled:         true, // 默认启用（教学关卡会在初始化后禁用）
	}
}

// Update 更新阳光生成计时器
func (s *SunSpawnSystem) Update(deltaTime float64) {
	// Story 8.2: 检查是否启用（教学关卡初始禁用）
	if !s.enabled {
		return
	}

	// 累加计时器
	s.spawnTimer += deltaTime

	// DEBUG: 每秒输出一次计时器状态
	if int(s.spawnTimer)%1 == 0 && s.spawnTimer-float64(int(s.spawnTimer)) < deltaTime {
		log.Printf("[SunSpawnSystem] Timer: %.2f / %.2f seconds", s.spawnTimer, s.spawnInterval)
	}

	// 检查是否到达生成间隔
	if s.spawnTimer >= s.spawnInterval {
		// 重置计时器
		s.spawnTimer = 0

		// 生成随机起始X坐标
		startX := s.minX + rand.Float64()*(s.maxX-s.minX)

		// 生成随机落地Y坐标
		targetY := s.minTargetY + rand.Float64()*(s.maxTargetY-s.minTargetY)

		// 创建阳光实体
		log.Printf("[SunSpawnSystem] *** SPAWNING SUN *** at X=%.1f, targetY=%.1f", startX, targetY)
		sunID := entities.NewSunEntity(s.entityManager, s.resourceManager, startX, targetY)
		log.Printf("[SunSpawnSystem] Created sun entity ID: %d", sunID)

		// Story 8.2 QA修复：初始化阳光动画（Sun.reanim 无 anim 定义，使用直接渲染模式）
		if err := s.reanimSystem.InitializeDirectRender(sunID); err != nil {
			log.Printf("[SunSpawnSystem] WARNING: Failed to initialize sun animation: %v", err)
		}
	}
}

// Enable 启用阳光自动生成（教学关卡在第一次收集阳光后调用）
func (s *SunSpawnSystem) Enable() {
	s.enabled = true
	log.Printf("[SunSpawnSystem] Auto spawn ENABLED")
}

// Disable 禁用阳光自动生成（教学关卡初始化时调用）
func (s *SunSpawnSystem) Disable() {
	s.enabled = false
	log.Printf("[SunSpawnSystem] Auto spawn DISABLED")
}