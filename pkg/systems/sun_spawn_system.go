package systems

import (
	"log"
	"math/rand"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
)

// SunSpawnSystem 管理阳光的定时生成
type SunSpawnSystem struct {
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager
	spawnTimer      float64 // 当前计时器
	spawnInterval   float64 // 生成间隔(秒)
	sunDroppedCount int     // 已掉落阳光计数（用于计算间隔）
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
	system := &SunSpawnSystem{
		entityManager:   em,
		resourceManager: rm,
		spawnTimer:      0,
		sunDroppedCount: 0, // 初始为 0
		minX:            minX,
		maxX:            maxX,
		minTargetY:      minTargetY,
		maxTargetY:      maxTargetY,
		enabled:         true, // 默认启用（教学关卡会在初始化后禁用）
	}
	// 使用原版公式计算初始间隔
	system.spawnInterval = system.calculateNextInterval()

	log.Printf("[SunSpawnSystem] Initialized with interval=%.2fs (count=%d), area=(%.0f-%.0f, %.0f-%.0f)",
		system.spawnInterval, system.sunDroppedCount, minX, maxX, minTargetY, maxTargetY)
	return system
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
		// 重置计时器
		s.spawnTimer = 0

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
		log.Printf("[SunSpawnSystem] *** SPAWNING SUN #%d *** at X=%.1f, targetY=%.1f", s.sunDroppedCount+1, startX, targetY)
		sunID := entities.NewSunEntity(s.entityManager, s.resourceManager, startX, targetY)

		// 更新计数和间隔
		s.sunDroppedCount++
		s.spawnInterval = s.calculateNextInterval()
		log.Printf("[SunSpawnSystem] Created sun entity ID: %d, next interval=%.2fs (count=%d)",
			sunID, s.spawnInterval, s.sunDroppedCount)

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

// Reset 重置阳光生成系统状态（关卡重新开始时调用）
func (s *SunSpawnSystem) Reset() {
	s.spawnTimer = 0
	s.sunDroppedCount = 0
	s.spawnInterval = s.calculateNextInterval()
	log.Printf("[SunSpawnSystem] Reset: interval=%.2fs", s.spawnInterval)
}

// calculateNextInterval 计算下一次阳光掉落间隔
// 原版公式: min{count * 10 + 425, 950} + rand(0~275)
// 单位: 厘秒 (centiseconds), 1cs = 0.01秒
// 参考: .meta/prompts.md - 原版游戏机制
// 间隔范围:
//   - count=0: 4.25~7.00秒
//   - count=52+: 9.50~12.25秒
func (s *SunSpawnSystem) calculateNextInterval() float64 {
	// 计算基础间隔 (厘秒)
	baseCS := s.sunDroppedCount*10 + 425
	if baseCS > 950 {
		baseCS = 950
	}
	// 添加随机偏移 (0-275 厘秒)
	randomCS := rand.Intn(276)
	// 总间隔 (厘秒)
	totalCS := baseCS + randomCS
	// 转换为秒
	return float64(totalCS) / 100.0
}
