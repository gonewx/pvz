package behavior

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
)

func (s *BehaviorSystem) handlePeaProjectileBehavior(entityID ecs.EntityID, deltaTime float64) {
	// 获取位置组件
	position, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 获取速度组件
	velocity, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 更新位置：根据速度和时间增量移动子弹
	position.X += velocity.VX * deltaTime
	position.Y += velocity.VY * deltaTime

	// 边界检查：如果子弹飞出屏幕右侧，标记删除
	if position.X > config.PeaBulletDeletionBoundary {
		log.Printf("[BehaviorSystem] 豌豆子弹 %d 飞出屏幕右侧 (X=%.1f)，标记删除", entityID, position.X)
		s.entityManager.DestroyEntity(entityID)
	}
}

// handleHitEffectBehavior 处理击中效果的生命周期
// 击中效果会在显示一段时间后自动消失

func (s *BehaviorSystem) handleHitEffectBehavior(entityID ecs.EntityID, deltaTime float64) {
	// 获取计时器组件
	timer, ok := ecs.GetComponent[*components.TimerComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 更新计时器
	timer.CurrentTime += deltaTime

	// 检查计时器是否完成（超时）
	if timer.CurrentTime >= timer.TargetTime {
		// 击中效果生命周期结束，标记删除
		s.entityManager.DestroyEntity(entityID)
	}
}

// playShootSound 播放豌豆射手发射子弹的音效
// 使用 AudioManager 统一管理音效（Story 10.9）
func (s *BehaviorSystem) playShootSound() {
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_THROW")
	}
}

// detectPlantCollision 检测僵尸是否与植物发生网格碰撞
// 参数:
//   - zombieRow: 僵尸所在行
//   - zombieCol: 僵尸所在列
//
// 返回:
//   - ecs.EntityID: 植物实体ID（如果碰撞）
//   - bool: 是否发生碰撞
