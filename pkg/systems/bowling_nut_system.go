package systems

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

// BowlingNutSystem 保龄球坚果滚动系统
// Story 19.6: 处理保龄球坚果的滚动移动和边界销毁
//
// 职责：
// - 更新坚果的水平位置
// - 检测边界并销毁越界坚果
// - 管理滚动音效播放
type BowlingNutSystem struct {
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager

	// 滚动音效播放器映射（entityID -> player）
	soundPlayers map[ecs.EntityID]*audio.Player
}

// NewBowlingNutSystem 创建保龄球坚果滚动系统
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载音效）
//
// 返回:
//   - *BowlingNutSystem: 系统实例
func NewBowlingNutSystem(em *ecs.EntityManager, rm *game.ResourceManager) *BowlingNutSystem {
	return &BowlingNutSystem{
		entityManager:   em,
		resourceManager: rm,
		soundPlayers:    make(map[ecs.EntityID]*audio.Player),
	}
}

// Update 更新所有保龄球坚果的位置
//
// 参数:
//   - dt: 帧间隔时间（秒）
//
// 处理逻辑：
// 1. 查询所有 BowlingNutComponent 实体
// 2. 如果正在滚动，更新 X 位置
// 3. 检查是否超出边界，超出则销毁
// 4. 管理滚动音效
func (s *BowlingNutSystem) Update(dt float64) {
	// 查询所有保龄球坚果实体
	entities := ecs.GetEntitiesWith2[*components.BowlingNutComponent, *components.PositionComponent](s.entityManager)

	for _, entityID := range entities {
		nutComp, nutOk := ecs.GetComponent[*components.BowlingNutComponent](s.entityManager, entityID)
		posComp, posOk := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)

		if !nutOk || !posOk {
			continue
		}

		// 如果正在滚动，更新位置
		if nutComp.IsRolling {
			// 更新 X 位置
			posComp.X += nutComp.VelocityX * dt

			// 开始播放滚动音效（如果还没播放）
			if !nutComp.SoundPlaying {
				s.startRollingSound(entityID)
				nutComp.SoundPlaying = true
			}
		}

		// 检查边界：X 坐标超过背景宽度时销毁
		if posComp.X > config.BackgroundWidth {
			log.Printf("[BowlingNutSystem] 坚果越界销毁: entityID=%d, X=%.1f", entityID, posComp.X)

			// 停止音效
			s.stopRollingSound(entityID)

			// 标记实体销毁
			s.entityManager.DestroyEntity(entityID)
		}
	}

	// 清理已销毁实体的音效播放器
	s.cleanupSoundPlayers()
}

// startRollingSound 开始播放滚动音效
func (s *BowlingNutSystem) startRollingSound(entityID ecs.EntityID) {
	if s.resourceManager == nil {
		return
	}

	// 检查是否已有播放器
	if _, exists := s.soundPlayers[entityID]; exists {
		return
	}

	// 加载音效
	soundPath := config.BowlingRollSoundPath
	player, err := s.resourceManager.LoadSoundEffect(soundPath)
	if err != nil {
		log.Printf("[BowlingNutSystem] 加载滚动音效失败: %v", err)
		return
	}

	// 设置循环播放
	// 注意：Ebitengine 的 audio.Player 不直接支持循环
	// 需要在音效结束时重新播放
	player.Rewind()
	player.Play()

	s.soundPlayers[entityID] = player
	log.Printf("[BowlingNutSystem] 开始播放滚动音效: entityID=%d", entityID)
}

// stopRollingSound 停止滚动音效
func (s *BowlingNutSystem) stopRollingSound(entityID ecs.EntityID) {
	if player, exists := s.soundPlayers[entityID]; exists {
		if player != nil {
			player.Pause()
		}
		delete(s.soundPlayers, entityID)
		log.Printf("[BowlingNutSystem] 停止滚动音效: entityID=%d", entityID)
	}
}

// cleanupSoundPlayers 清理已销毁实体的音效播放器
func (s *BowlingNutSystem) cleanupSoundPlayers() {
	for entityID, player := range s.soundPlayers {
		// 检查实体是否还存在
		if _, ok := ecs.GetComponent[*components.BowlingNutComponent](s.entityManager, entityID); !ok {
			if player != nil {
				player.Pause()
			}
			delete(s.soundPlayers, entityID)
		}
	}
}

// StopAllSounds 停止所有滚动音效
// 在场景切换或游戏结束时调用
func (s *BowlingNutSystem) StopAllSounds() {
	for entityID, player := range s.soundPlayers {
		if player != nil {
			player.Pause()
		}
		delete(s.soundPlayers, entityID)
	}
	log.Printf("[BowlingNutSystem] 停止所有滚动音效")
}

