package behavior

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
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

// handleZombieDyingBehavior 处理僵尸死亡动画播放
// 当死亡动画完成后，删除僵尸实体

func (s *BehaviorSystem) playShootSound() {
	// 如果配置为空字符串，不播放音效（保持原版静音风格）
	if config.PeashooterShootSoundPath == "" {
		return
	}

	// 加载发射音效（如果已加载，会返回缓存的播放器）
	// 音效路径在 pkg/config/unit_config.go 中配置，可根据需要切换测试
	shootSound, err := s.resourceManager.LoadSoundEffect(config.PeashooterShootSoundPath)
	if err != nil {
		// 音效加载失败时不阻止游戏继续运行
		// 在实际项目中可以使用日志系统记录错误
		return
	}

	// 重置播放器位置到开头（允许快速连续播放）
	shootSound.Rewind()

	// 播放音效
	shootSound.Play()
}

// detectPlantCollision 检测僵尸是否与植物发生网格碰撞
// 参数:
//   - zombieRow: 僵尸所在行
//   - zombieCol: 僵尸所在列
//
// 返回:
//   - ecs.EntityID: 植物实体ID（如果碰撞）
//   - bool: 是否发生碰撞
