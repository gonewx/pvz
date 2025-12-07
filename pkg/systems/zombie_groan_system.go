package systems

import (
	"math/rand"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// ZombieGroanSystem 僵尸呻吟音效系统
// 当场上有僵尸时，随机播放呻吟音效，营造紧张氛围
type ZombieGroanSystem struct {
	entityManager *ecs.EntityManager
	gameState     *game.GameState
	nextGroanTime float64 // 下次播放呻吟的时间
}

// groanSounds 呻吟音效列表
var groanSounds = []string{
	"SOUND_GROAN",
	"SOUND_GROAN2",
	"SOUND_GROAN3",
	"SOUND_GROAN4",
	"SOUND_GROAN5",
	"SOUND_GROAN6",
}

// NewZombieGroanSystem 创建僵尸呻吟音效系统
func NewZombieGroanSystem(em *ecs.EntityManager, gs *game.GameState) *ZombieGroanSystem {
	return &ZombieGroanSystem{
		entityManager: em,
		gameState:     gs,
		nextGroanTime: 0,
	}
}

// Update 更新系统
// 检查场上是否有活着的僵尸，如果有则随机播放呻吟音效
func (s *ZombieGroanSystem) Update(deltaTime float64) {
	// 累积时间
	s.nextGroanTime -= deltaTime

	// 还没到播放时间
	if s.nextGroanTime > 0 {
		return
	}

	// 检查是否有活着的僵尸
	if !s.hasActiveZombies() {
		// 没有僵尸，重置计时器
		s.nextGroanTime = s.getRandomInterval()
		return
	}

	// 播放随机呻吟音效
	s.playRandomGroan()

	// 设置下次播放时间
	s.nextGroanTime = s.getRandomInterval()
}

// hasActiveZombies 检查场上是否有活着的僵尸
func (s *ZombieGroanSystem) hasActiveZombies() bool {
	// 查询所有有 BehaviorComponent 的实体
	entities := ecs.GetEntitiesWith1[*components.BehaviorComponent](s.entityManager)

	for _, entityID := range entities {
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 检查是否是活着的僵尸（移动中或正在吃植物）
		switch behavior.Type {
		case components.BehaviorZombieBasic,
			components.BehaviorZombieEating,
			components.BehaviorZombieConehead,
			components.BehaviorZombieBuckethead,
			components.BehaviorZombieFlag:
			return true
		}
	}

	return false
}

// playRandomGroan 播放随机呻吟音效
func (s *ZombieGroanSystem) playRandomGroan() {
	audioManager := s.gameState.GetAudioManager()
	if audioManager == nil {
		return
	}

	// 随机选择一个呻吟音效
	randomIndex := rand.Intn(len(groanSounds))
	audioManager.PlaySound(groanSounds[randomIndex])
}

// getRandomInterval 获取随机的播放间隔
func (s *ZombieGroanSystem) getRandomInterval() float64 {
	minInterval := config.ZombieGroanMinInterval
	maxInterval := config.ZombieGroanMaxInterval
	return minInterval + rand.Float64()*(maxInterval-minInterval)
}

