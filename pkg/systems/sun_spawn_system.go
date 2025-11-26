package systems

import (
	"log"
	"math/rand"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
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
	enabled         bool    // 是否启用自动生成（教学关卡初始禁用）
}

// NewSunSpawnSystem 创建一个新的阳光生成系统
// 参数:
//   - em: EntityManager 实例
//   - rm: ResourceManager 实例
//   - minX, maxX: 阳光生成的水平范围
//   - minTargetY, maxTargetY: 阳光落地的垂直范围
func NewSunSpawnSystem(em *ecs.EntityManager, rm *game.ResourceManager, minX, maxX, minTargetY, maxTargetY float64) *SunSpawnSystem {
	// 基础间隔: 8秒 ±2秒随机变化
	baseInterval := 8.0
	randomOffset := -2.0 + rand.Float64()*4.0 // -2 到 +2 秒
	initialInterval := baseInterval + randomOffset

	log.Printf("[SunSpawnSystem] Initialized with interval=%.1fs (base 8±2s), area=(%.0f-%.0f, %.0f-%.0f)",
		initialInterval, minX, maxX, minTargetY, maxTargetY)
	return &SunSpawnSystem{
		entityManager:   em,
		resourceManager: rm,
		spawnTimer:      0,
		spawnInterval:   initialInterval, // 原版游戏机制: 8秒±1秒随机
		minX:            minX,
		maxX:            maxX,
		minTargetY:      minTargetY,
		maxTargetY:      maxTargetY,
		enabled:         true, // 默认启用（教学关卡会在初始化后禁用）
	}
}

// Update 更新阳光生成计时器
func (s *SunSpawnSystem) Update(deltaTime float64) {
	// 检查是否启用（教学关卡初始禁用）
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
		// 重置计时器并重新随机化下次间隔
		s.spawnTimer = 0
		baseInterval := 8.0
		randomOffset := -2.0 + rand.Float64()*4.0 // -2 到 +2 秒
		s.spawnInterval = baseInterval + randomOffset

		// 生成随机起始X坐标（增加±80像素的随机偏移）
		baseX := s.minX + rand.Float64()*(s.maxX-s.minX)
		xRandomOffset := -80.0 + rand.Float64()*160.0 // ±80像素
		startX := baseX + xRandomOffset
		if startX < s.minX {
			startX = s.minX
		}
		if startX > s.maxX {
			startX = s.maxX
		}

		// 生成随机落地Y坐标（增加±50像素的随机偏移）
		baseY := s.minTargetY + rand.Float64()*(s.maxTargetY-s.minTargetY)
		yRandomOffset := -50.0 + rand.Float64()*100.0 // ±50像素
		targetY := baseY + yRandomOffset
		if targetY < s.minTargetY {
			targetY = s.minTargetY
		}
		if targetY > s.maxTargetY {
			targetY = s.maxTargetY
		}

		// 边界检查：确保阳光完整显示在屏幕内
		// 使用配置常量而不是硬编码值
		originalX, originalY := startX, targetY
		sunRadius := config.SunOffsetCenterX // 阳光半径 40
		if startX < sunRadius {
			startX = sunRadius
		}
		if startX > config.ScreenWidth-sunRadius {
			startX = config.ScreenWidth - sunRadius
		}
		if targetY < sunRadius {
			targetY = sunRadius
		}
		if targetY > config.ScreenHeight-sunRadius {
			targetY = config.ScreenHeight - sunRadius
		}

		// 记录边界调整（仅当位置被修改时）
		if startX != originalX || targetY != originalY {
			log.Printf("[SunSpawnSystem] 边界检查: (%.1f, %.1f) -> (%.1f, %.1f)",
				originalX, originalY, startX, targetY)
		}

		// 创建阳光实体
		log.Printf("[SunSpawnSystem] *** SPAWNING SUN *** at X=%.1f, targetY=%.1f", startX, targetY)
		sunID := entities.NewSunEntity(s.entityManager, s.resourceManager, startX, targetY)
		log.Printf("[SunSpawnSystem] Created sun entity ID: %d", sunID)

		// Sun.reanim 只有轨道(Sun1, Sun2, Sun3),没有动画定义
		// 使用 AnimationCommand 组件播放配置的"idle"组合（包含所有3个轨道）
		ecs.AddComponent(s.entityManager, sunID, &components.AnimationCommandComponent{
			UnitID:    "sun",
			ComboName: "idle",
			Processed: false,
		})
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
