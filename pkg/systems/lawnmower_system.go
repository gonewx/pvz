package systems

import (
	"log"
	"math"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
)

// LawnmowerSystem 除草车系统
// 职责：
// - 检测触发条件（僵尸到达左侧边界）
// - 触发除草车移动
// - 更新除草车位置
// - 检测并消灭路径上的僵尸
// - 管理除草车状态（未触发/移动中/已使用）
type LawnmowerSystem struct {
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager // 用于播放音效
	reanimSystem    *ReanimSystem         // Story 10.3: 用于播放僵尸死亡动画
	gameState       *game.GameState       // 用于增加消灭僵尸计数
	stateEntityID   ecs.EntityID          // 全局状态实体ID
}

// NewLawnmowerSystem 创建除草车系统
// 参数:
//   - em: EntityManager 实例
//   - rm: ResourceManager 实例（用于播放音效和粒子效果）
//   - rs: ReanimSystem 实例（Story 10.3: 用于播放僵尸死亡动画）
//   - gs: GameState 实例（用于记录游戏统计）
//
// 返回:
//   - *LawnmowerSystem: 除草车系统实例
func NewLawnmowerSystem(em *ecs.EntityManager, rm *game.ResourceManager, rs *ReanimSystem, gs *game.GameState) *LawnmowerSystem {
	// 创建全局状态实体
	stateEntity := em.CreateEntity()
	ecs.AddComponent(em, stateEntity, &components.LawnmowerStateComponent{
		UsedLanes: make(map[int]bool),
	})

	return &LawnmowerSystem{
		entityManager:   em,
		resourceManager: rm,
		reanimSystem:    rs,
		gameState:       gs,
		stateEntityID:   stateEntity,
	}
}

// Update 更新除草车系统
// 参数:
//   - deltaTime: 自上次更新以来的时间（秒）
func (s *LawnmowerSystem) Update(deltaTime float64) {
	// 1. 检测触发条件
	s.checkTriggerConditions()

	// 2. 更新移动中的除草车位置
	s.updateLawnmowerPositions(deltaTime)

	// 3. 检���并消灭僵尸
	s.checkZombieCollisions()

	// 4. 检测除草车离开屏幕
	s.checkLawnmowerCompletion()
}

// checkTriggerConditions 检测是否有僵尸到达左侧，触发除草车
// 对应 Task 5: 实现除草车触发逻辑
func (s *LawnmowerSystem) checkTriggerConditions() {
	// 查询所有僵尸实体
	zombieEntities := ecs.GetEntitiesWith2[
		*components.BehaviorComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, zombieID := range zombieEntities {
		behavior, _ := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)

		// 只检查僵尸类型（跳过植物等其他实体）
		if !isZombieType(behavior.Type) {
			continue
		}

		// 僵尸到达左侧边界
		if pos.X < config.LawnmowerTriggerBoundary {
			// 计算僵尸所在行（1-5）
			lane := s.getEntityLane(pos.Y)

			// 触发该行的除草车
			s.triggerLawnmower(lane)
		}
	}
}

// triggerLawnmower 触发指定行的除草车
// 参数:
//   - lane: 行号（1-5）
func (s *LawnmowerSystem) triggerLawnmower(lane int) {
	// 查询该行的除草车
	lawnmowers := ecs.GetEntitiesWith2[
		*components.LawnmowerComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, lawnmowerID := range lawnmowers {
		lawnmower, _ := ecs.GetComponent[*components.LawnmowerComponent](s.entityManager, lawnmowerID)

		if lawnmower.Lane == lane && !lawnmower.IsTriggered {
			// 触发除草车
			lawnmower.IsTriggered = true
			lawnmower.IsMoving = true

			// 播放音效（使用资源 ID 而不是相对路径）
			if s.resourceManager != nil {
				player := s.resourceManager.GetAudioPlayer("SOUND_LAWNMOWER")
				if player != nil {
					player.Rewind()
					player.Play()
				} else {
					log.Printf("[LawnmowerSystem] Warning: Lawnmower audio (SOUND_LAWNMOWER) not loaded")
				}
			}

			// 恢复动画播放（触发后开始播放车轮滚动动画）
			// 注意：不切换动画，继续使用 anim_normal，只是取消暂停
			if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, lawnmowerID); ok {
				reanim.IsPaused = false // 恢复播放，车轮开始滚动、轻微晃动
				log.Printf("[LawnmowerSystem] Resumed animation for lawnmower on lane %d", lane)
			}

			log.Printf("[LawnmowerSystem] Lawnmower triggered on lane %d", lane)
			break
		}
	}
}

// updateLawnmowerPositions 更新移动中的除草车位置
// 参数:
//   - deltaTime: 自上次更新以来的时间（秒）
func (s *LawnmowerSystem) updateLawnmowerPositions(deltaTime float64) {
	lawnmowers := ecs.GetEntitiesWith2[
		*components.LawnmowerComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, lawnmowerID := range lawnmowers {
		lawnmower, _ := ecs.GetComponent[*components.LawnmowerComponent](s.entityManager, lawnmowerID)
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, lawnmowerID)

		if lawnmower.IsMoving {
			// 向右移动
			pos.X += lawnmower.Speed * deltaTime
		}
	}
}

// checkZombieCollisions 检测除草车与僵尸的碰撞，消灭僵尸
// 对应 Task 6: 实现僵尸消灭逻辑
func (s *LawnmowerSystem) checkZombieCollisions() {
	// 获取所有移动中的除草车
	lawnmowers := ecs.GetEntitiesWith2[
		*components.LawnmowerComponent,
		*components.PositionComponent,
	](s.entityManager)

	// 获取所有僵尸实体
	zombieEntities := ecs.GetEntitiesWith3[
		*components.BehaviorComponent,
		*components.PositionComponent,
		*components.HealthComponent,
	](s.entityManager)

	for _, lawnmowerID := range lawnmowers {
		lawnmower, _ := ecs.GetComponent[*components.LawnmowerComponent](s.entityManager, lawnmowerID)
		lawnmowerPos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, lawnmowerID)

		// 只处理移动中的除草车
		if !lawnmower.IsMoving {
			continue
		}

		// 检测同行的僵尸碰撞
		for _, zombieID := range zombieEntities {
			behavior, _ := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
			zombiePos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
			health, _ := ecs.GetComponent[*components.HealthComponent](s.entityManager, zombieID)

			// 只检查僵尸类型
			if !isZombieType(behavior.Type) {
				continue
			}

			// 跳过已死亡的僵尸
			if health.CurrentHealth <= 0 {
				continue
			}

			// 只杀死已激活的僵尸（跳过预生成但未激活的僵尸）
			waveState, hasWaveState := ecs.GetComponent[*components.ZombieWaveStateComponent](s.entityManager, zombieID)
			if hasWaveState && !waveState.IsActivated {
				// 预生成的僵尸尚未激活，不应被除草车杀死
				continue
			}

			// 计算僵尸所在行
			zombieLane := s.getEntityLane(zombiePos.Y)

			// 检查是否在同一行
			if zombieLane != lawnmower.Lane {
				continue
			}

			// 碰撞检测：僵尸 X 坐标在除草车 X ± CollisionRange 范围内
			distance := math.Abs(zombiePos.X - lawnmowerPos.X)
			if distance < config.LawnmowerCollisionRange {
				// Story 10.3: 除草车碾压僵尸，触发死亡动画
				// 不再直接删除，而是播放死亡动画和粒子效果

				// 注意：不在这里调用 IncrementZombiesKilled()
				// 计数统一在 BehaviorSystem.handleZombieDyingBehavior() 中进行
				// 避免重复计数（除草车触发一次 + 死亡动画完成一次）

				log.Printf("[LawnmowerSystem] Lawnmower on lane %d killed zombie at (%.1f, %.1f)",
					lawnmower.Lane, zombiePos.X, zombiePos.Y)

				// 触发僵尸死亡（播放动画和粒子效果）
				s.triggerZombieDeath(zombieID)
			}
		}
	}
}

// checkLawnmowerCompletion 检测除草车是否离开屏幕
func (s *LawnmowerSystem) checkLawnmowerCompletion() {
	// 获取全局状态组件
	state, ok := ecs.GetComponent[*components.LawnmowerStateComponent](s.entityManager, s.stateEntityID)
	if !ok {
		log.Printf("[LawnmowerSystem] Warning: LawnmowerStateComponent not found")
		return
	}

	lawnmowers := ecs.GetEntitiesWith2[
		*components.LawnmowerComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, lawnmowerID := range lawnmowers {
		lawnmower, _ := ecs.GetComponent[*components.LawnmowerComponent](s.entityManager, lawnmowerID)
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, lawnmowerID)

		// 检测除草车是否离开屏幕（X > 删除边界）
		if lawnmower.IsMoving && pos.X > config.LawnmowerDeletionBoundary {
			// 标记该行除草车为已使用
			state.UsedLanes[lawnmower.Lane] = true

			// 删除除草车实体
			s.entityManager.DestroyEntity(lawnmowerID)

			log.Printf("[LawnmowerSystem] Lawnmower on lane %d completed and removed", lawnmower.Lane)
		}
	}
}

// getEntityLane 根据实体的Y坐标计算所在行（1-5）
// 参数:
//   - y: 实体的世界坐标Y
//
// 返回:
//   - int: 行号（1-5）
func (s *LawnmowerSystem) getEntityLane(y float64) int {
	// 计算相对于网格起点的Y偏移
	offsetY := y - config.GridWorldStartY

	// 计算行号（0-4）
	row := int(offsetY / config.CellHeight)

	// 转换为行号（1-5）
	lane := row + 1

	// 限制范围
	if lane < 1 {
		lane = 1
	}
	if lane > 5 {
		lane = 5
	}

	return lane
}

// GetStateEntityID 返回全局状态实体ID（用于 LevelSystem 访问）
func (s *LawnmowerSystem) GetStateEntityID() ecs.EntityID {
	return s.stateEntityID
}

// HasActiveLawnmowers 检查是否有活跃的（移动中的）除草车
// 用于胜利条件判断：除草车完全消失后才能显示胜利动画
// 返回:
//   - bool: 如果有任何除草车正在移动，返回 true
func (s *LawnmowerSystem) HasActiveLawnmowers() bool {
	lawnmowers := ecs.GetEntitiesWith1[*components.LawnmowerComponent](s.entityManager)

	for _, lawnmowerID := range lawnmowers {
		lawnmower, ok := ecs.GetComponent[*components.LawnmowerComponent](s.entityManager, lawnmowerID)
		if ok && lawnmower.IsMoving {
			return true
		}
	}

	return false
}

// triggerZombieDeath 触发僵尸死亡动画和粒子效果
// Story 10.3: 除草车碾压僵尸时调用此方法，不再直接删除
func (s *LawnmowerSystem) triggerZombieDeath(zombieID ecs.EntityID) {
	// 1. 切换行为类型为 BehaviorZombieDying
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
	if !ok {
		log.Printf("[LawnmowerSystem] 僵尸 %d 缺少 BehaviorComponent，无法触发死亡", zombieID)
		return
	}
	behavior.Type = components.BehaviorZombieDying
	log.Printf("[LawnmowerSystem] 僵尸 %d 行为切换为 BehaviorZombieDying", zombieID)

	// 2. 移除速度组件（僵尸停止移动）
	if ecs.HasComponent[*components.VelocityComponent](s.entityManager, zombieID) {
		ecs.RemoveComponent[*components.VelocityComponent](s.entityManager, zombieID)
		log.Printf("[LawnmowerSystem] 僵尸 %d 移除 VelocityComponent（停止移动）", zombieID)
	}

	// 3. 播放死亡动画（单次播放，不循环）
	// Story 13.8: 使用配置驱动的动画组合（自动隐藏装备轨道）
	if s.reanimSystem != nil {
		err := s.reanimSystem.PlayCombo(zombieID, "zombie", "death")
		if err != nil {
			log.Printf("[LawnmowerSystem] 僵尸 %d 播放死亡动画失败: %v", zombieID, err)
		} else {
			// 设置为不循环
			if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, zombieID); ok {
				reanim.IsLooping = false
			}
			log.Printf("[LawnmowerSystem] 僵尸 %d 开始播放死亡动画", zombieID)
		}
	}

	// 4. 触发粒子效果（手臂和头部掉落）
	// 这与 BehaviorSystem 的逻辑一致
	position, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
	if !ok {
		log.Printf("[LawnmowerSystem] 警告：僵尸 %d 缺少 PositionComponent，无法触发粒子效果", zombieID)
		return
	}

	// Story 7.6: 除草车碾压的僵尸通常向左走，需要翻转粒子方向
	angleOffset := 180.0 // 默认翻转（适合僵尸向左走）

	// 触发僵尸手臂掉落粒子效果
	_, err := entities.CreateParticleEffect(
		s.entityManager,
		s.resourceManager,
		"MoweredZombieArm",
		position.X, position.Y,
		angleOffset,
	)
	if err != nil {
		log.Printf("[LawnmowerSystem] 警告：创建僵尸手臂掉落粒子效果失败: %v", err)
	} else {
		log.Printf("[LawnmowerSystem] 僵尸 %d 触发手臂掉落粒子效果", zombieID)
	}

	// 触发僵尸头部掉落粒子效果
	_, err = entities.CreateParticleEffect(
		s.entityManager,
		s.resourceManager,
		"MoweredZombieHead",
		position.X, position.Y,
		angleOffset,
	)
	if err != nil {
		log.Printf("[LawnmowerSystem] 警告：创建僵尸头部掉落粒子效果失败: %v", err)
	} else {
		log.Printf("[LawnmowerSystem] 僵尸 %d 触发头部掉落粒子效果", zombieID)
	}
}
