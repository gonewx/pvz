package behavior

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
)

// BehaviorSystem 处理实体的行为逻辑
// 根据实体的 BehaviorComponent 类型执行相应的行为（如向日葵生产阳光、豌豆射手攻击等）
type BehaviorSystem struct {
	entityManager    *ecs.EntityManager
	resourceManager  *game.ResourceManager
	gameState        *game.GameState         // 用于僵尸死亡计数
	logFrameCounter  int                     // 日志输出计数器（避免全局变量）
	lawnGridSystem   *systems.LawnGridSystem // 用于植物死亡时释放网格占用
	lawnGridEntityID ecs.EntityID            // 草坪网格实体ID
}

// 日志输出间隔常量
const LogOutputFrameInterval = 100 // 日志输出间隔（每N帧输出一次）

// NewBehaviorSystem 创建一个新的行为系统
// 参数:
//   - em: EntityManager 实例
//   - rm: ResourceManager 实例
//   - gs: GameState 实例 (用于僵尸死亡计数)
//   - lgs: LawnGridSystem 实例 (用于植物死亡时释放网格占用)
//   - lawnGridID: 草坪网格实体ID
func NewBehaviorSystem(em *ecs.EntityManager, rm *game.ResourceManager, gs *game.GameState, lgs *systems.LawnGridSystem, lawnGridID ecs.EntityID) *BehaviorSystem {
	return &BehaviorSystem{
		entityManager:    em,
		resourceManager:  rm,
		gameState:        gs,
		lawnGridSystem:   lgs,
		lawnGridEntityID: lawnGridID,
	}
}

// Update 更新所有拥有行为组件的实体
func (s *BehaviorSystem) Update(deltaTime float64) {
	// 检查游戏是否冻结（僵尸获胜流程期间）
	// Story 8.8: 游戏冻结时，所有植物停止攻击，但触发僵尸继续移动
	freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](s.entityManager)
	if len(freezeEntities) > 0 {
		// 游戏冻结期间，只允许触发僵尸继续移动
		// 查询 ZombiesWonPhaseComponent 获取触发僵尸ID
		phaseEntities := ecs.GetEntitiesWith1[*components.ZombiesWonPhaseComponent](s.entityManager)
		for _, phaseEntityID := range phaseEntities {
			phaseComp, ok := ecs.GetComponent[*components.ZombiesWonPhaseComponent](s.entityManager, phaseEntityID)
			if !ok {
				continue
			}
			// 只更新触发僵尸的移动（Phase 2 期间僵尸需要继续走出屏幕）
			triggerZombieID := phaseComp.TriggerZombieID
			// Story 8.8: Phase 1 (冻结) 期间僵尸不应移动，只有 Phase 2+ 才允许移动
			if triggerZombieID != 0 && phaseComp.CurrentPhase >= 2 {
				// 简化的移动逻辑：只更新位置，不检测碰撞
				s.updateTriggerZombieMovement(triggerZombieID, deltaTime)
			}
		}
		return
	}

	// 查询所有植物实体
	plantEntityList := s.queryPlants()

	// 查询所有移动中的僵尸实体
	zombieEntityList := s.queryMovingZombies()

	// DEBUG: 记录僵尸数量
	if len(zombieEntityList) > 0 {
		log.Printf("[BehaviorSystem] Update called, found %d moving zombies", len(zombieEntityList))
	}

	// 查询所有啃食中的僵尸实体
	eatingZombieEntityList := s.queryEatingZombies()

	// 查询所有死亡中的僵尸实体
	dyingZombieEntityList := s.queryDyingZombies()

	// 合并所有活动僵尸列表（移动中 + 啃食中），用于豌豆射手检测目标
	allZombieEntityList := append([]ecs.EntityID{}, zombieEntityList...)
	allZombieEntityList = append(allZombieEntityList, eatingZombieEntityList...)

	// 查询所有豌豆子弹实体
	projectileEntityList := s.queryProjectiles()

	// 日志输出（避免每帧都打印）
	totalZombies := len(zombieEntityList) + len(eatingZombieEntityList)
	totalEntities := len(plantEntityList) + totalZombies + len(projectileEntityList)
	if totalEntities > 0 {
		s.logFrameCounter++
		if s.logFrameCounter%LogOutputFrameInterval == 1 {
			log.Printf("[BehaviorSystem] 更新 %d 个行为实体 (植物: %d, 僵尸: %d [移动:%d 啃食:%d], 子弹: %d)",
				totalEntities, len(plantEntityList), totalZombies, len(zombieEntityList), len(eatingZombieEntityList), len(projectileEntityList))
		}
	}

	// 遍历所有植物实体，根据行为类型分发处理
	for _, entityID := range plantEntityList {
		behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			log.Printf("[BehaviorSystem] ⚠️ 植物实体 %d 缺少 BehaviorComponent", entityID)
			continue
		}

		// 根据行为类型分发
		switch behaviorComp.Type {
		case components.BehaviorSunflower:
			s.handleSunflowerBehavior(entityID, deltaTime)
		case components.BehaviorPeashooter:
			s.handlePeashooterBehavior(entityID, deltaTime, allZombieEntityList)
		case components.BehaviorWallnut:
			s.handleWallnutBehavior(entityID)
		case components.BehaviorCherryBomb:
			s.handleCherryBombBehavior(entityID, deltaTime)
		default:
			// 未知行为类型，记录警告
			if s.logFrameCounter%LogOutputFrameInterval == 1 {
				log.Printf("[BehaviorSystem] ⚠️ 植物实体 %d 有未知行为类型: %v", entityID, behaviorComp.Type)
			}
		}
	}

	// 更新植物攻击动画状态（在所有行为处理之后）
	for _, entityID := range plantEntityList {
		s.updatePlantAttackAnimation(entityID, deltaTime)
	}

	// 遍历所有移动中的僵尸实体，根据行为类型分发处理
	for _, entityID := range zombieEntityList {
		behaviorComp, _ := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)

		// 根据行为类型分发
		switch behaviorComp.Type {
		case components.BehaviorZombieBasic:
			s.handleZombieBasicBehavior(entityID, deltaTime)
		case components.BehaviorZombieConehead:
			s.handleConeheadZombieBehavior(entityID, deltaTime)
		case components.BehaviorZombieBuckethead:
			s.handleBucketheadZombieBehavior(entityID, deltaTime)
		default:
			// 未知僵尸类型，忽略
		}
	}

	// 遍历所有啃食中的僵尸实体
	for _, entityID := range eatingZombieEntityList {
		behaviorComp, _ := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)

		// 只处理啃食状态的僵尸
		if behaviorComp.Type == components.BehaviorZombieEating {
			s.handleZombieEatingBehavior(entityID, deltaTime)
		}
	}

	// 遍历所有子弹实体，根据行为类型分发处理
	for _, entityID := range projectileEntityList {
		behaviorComp, _ := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)

		// 根据行为类型分发
		switch behaviorComp.Type {
		case components.BehaviorPeaProjectile:
			s.handlePeaProjectileBehavior(entityID, deltaTime)
		default:
			// 忽略非子弹类型（如僵尸）
		}
	}

	// 遍历所有死亡中的僵尸实体（处理死亡动画完成后的删除）
	for _, entityID := range dyingZombieEntityList {
		behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 只处理死亡中的僵尸
		if behaviorComp.Type == components.BehaviorZombieDying {
			s.handleZombieDyingBehavior(entityID)
		}
	}

	// Story 5.4.1: 处理爆炸烧焦死亡中的僵尸实体
	explosionDyingZombieEntityList := s.queryExplosionDyingZombies()
	for _, entityID := range explosionDyingZombieEntityList {
		s.handleZombieDyingExplosionBehavior(entityID)
	}

	// 查询所有击中效果实体（拥有 BehaviorComponent 和 TimerComponent）
	hitEffectEntityList := ecs.GetEntitiesWith2[
		*components.BehaviorComponent,
		*components.TimerComponent,
	](s.entityManager)

	// 遍历所有击中效果实体，管理其生命周期
	for _, entityID := range hitEffectEntityList {
		behaviorComp, _ := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)

		// 只处理击中效果类型
		if behaviorComp.Type == components.BehaviorPeaBulletHit {
			s.handleHitEffectBehavior(entityID, deltaTime)
		}
	}

}

// ============================================================================
// 实体查询辅助函数（封装复杂的查询逻辑）
// ============================================================================

// queryPlants 查询所有植物实体
//
// 返回所有拥有 BehaviorComponent, PlantComponent, PositionComponent 的实体
func (s *BehaviorSystem) queryPlants() []ecs.EntityID {
	return ecs.GetEntitiesWith3[
		*components.BehaviorComponent,
		*components.PlantComponent,
		*components.PositionComponent,
	](s.entityManager)
}

// queryMovingZombies 查询所有移动中的僵尸实体
//
// 返回所有拥有 VelocityComponent 且 BehaviorType 为僵尸类型的实体
// 注意：排除子弹（BehaviorPeaProjectile）
func (s *BehaviorSystem) queryMovingZombies() []ecs.EntityID {
	// 查询所有拥有 BehaviorComponent, PositionComponent, VelocityComponent 的实体
	candidates := ecs.GetEntitiesWith3[
		*components.BehaviorComponent,
		*components.PositionComponent,
		*components.VelocityComponent,
	](s.entityManager)

	// 过滤出真正的僵尸（排除子弹和其他实体）
	var zombies []ecs.EntityID
	for _, entityID := range candidates {
		behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 只保留僵尸类型的实体
		if s.isZombieBehaviorType(behaviorComp.Type) {
			zombies = append(zombies, entityID)
		}
	}

	return zombies
}

// queryEatingZombies 查询所有啃食中的僵尸实体
//
// 返回所有处于啃食状态的僵尸（BehaviorType == BehaviorZombieEating）
func (s *BehaviorSystem) queryEatingZombies() []ecs.EntityID {
	// 查询所有拥有 BehaviorComponent, PositionComponent, TimerComponent 的实体
	// 注意：这个查询会同时匹配啃食僵尸和豌豆射手植物（植物也有 TimerComponent）
	candidates := ecs.GetEntitiesWith3[
		*components.BehaviorComponent,
		*components.PositionComponent,
		*components.TimerComponent,
	](s.entityManager)

	// 过滤出真正处于啃食状态的僵尸
	var eatingZombies []ecs.EntityID
	for _, entityID := range candidates {
		behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		if behaviorComp.Type == components.BehaviorZombieEating {
			eatingZombies = append(eatingZombies, entityID)
		}
	}

	return eatingZombies
}

// queryDyingZombies 查询所有死亡中的僵尸实体
//
// 返回所有处于死亡状态的僵尸（BehaviorType == BehaviorZombieDying）
// 死亡状态的僵尸已移除 VelocityComponent，但保留 ReanimComponent（播放死亡动画）
func (s *BehaviorSystem) queryDyingZombies() []ecs.EntityID {
	// 查询所有拥有 BehaviorComponent, PositionComponent, ReanimComponent 的实体
	candidates := ecs.GetEntitiesWith3[
		*components.BehaviorComponent,
		*components.PositionComponent,
		*components.ReanimComponent,
	](s.entityManager)

	// 过滤出真正处于死亡状态的僵尸
	var dyingZombies []ecs.EntityID
	for _, entityID := range candidates {
		behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		if behaviorComp.Type == components.BehaviorZombieDying {
			dyingZombies = append(dyingZombies, entityID)
		}
	}

	return dyingZombies
}

// queryExplosionDyingZombies 查询所有爆炸烧焦死亡中的僵尸实体
//
// 返回所有处于爆炸烧焦死亡状态的僵尸（BehaviorType == BehaviorZombieDyingExplosion）
// Story 5.4.1: 实现爆炸类攻击（樱桃炸弹、土豆雷等）的专用烧焦死亡动画
func (s *BehaviorSystem) queryExplosionDyingZombies() []ecs.EntityID {
	// 查询所有拥有 BehaviorComponent, PositionComponent, ReanimComponent 的实体
	candidates := ecs.GetEntitiesWith3[
		*components.BehaviorComponent,
		*components.PositionComponent,
		*components.ReanimComponent,
	](s.entityManager)

	// 过滤出真正处于爆炸烧焦死亡状态的僵尸
	var explosionDyingZombies []ecs.EntityID
	for _, entityID := range candidates {
		behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		if behaviorComp.Type == components.BehaviorZombieDyingExplosion {
			explosionDyingZombies = append(explosionDyingZombies, entityID)
		}
	}

	return explosionDyingZombies
}

// queryProjectiles 查询所有豌豆子弹实体
//
// 返回所有 BehaviorType 为 BehaviorPeaProjectile 的实体
func (s *BehaviorSystem) queryProjectiles() []ecs.EntityID {
	// 查询所有拥有 BehaviorComponent, PositionComponent, VelocityComponent 的实体
	// 注意：子弹和移动中的僵尸组件组合相同，需要通过 BehaviorType 区分
	candidates := ecs.GetEntitiesWith3[
		*components.BehaviorComponent,
		*components.PositionComponent,
		*components.VelocityComponent,
	](s.entityManager)

	// DEBUG: 记录候选实体数量
	if len(candidates) > 0 {
		log.Printf("[BehaviorSystem] queryProjectiles: 找到 %d 个候选实体（有 Behavior+Position+Velocity）", len(candidates))
	}

	// 过滤出子弹
	var projectiles []ecs.EntityID
	for _, entityID := range candidates {
		behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			log.Printf("[BehaviorSystem] queryProjectiles: 实体 %d 没有 BehaviorComponent", entityID)
			continue
		}

		// DEBUG: 记录每个候选实体的行为类型
		log.Printf("[BehaviorSystem] queryProjectiles: 实体 %d 的行为类型 = %v（是子弹: %v）",
			entityID, behaviorComp.Type, behaviorComp.Type == components.BehaviorPeaProjectile)

		if behaviorComp.Type == components.BehaviorPeaProjectile {
			projectiles = append(projectiles, entityID)
		}
	}

	// DEBUG: 记录找到的子弹数量
	if len(projectiles) > 0 {
		log.Printf("[BehaviorSystem] queryProjectiles: 找到 %d 个子弹实体", len(projectiles))
	}

	return projectiles
}

// isZombieBehaviorType 判断行为类型是否为僵尸类型
//
// 参数:
//   - behaviorType: 行为类型
//
// 返回:
//   - true: 是僵尸类型
//   - false: 不是僵尸类型
func (s *BehaviorSystem) isZombieBehaviorType(behaviorType components.BehaviorType) bool {
	switch behaviorType {
	case components.BehaviorZombieBasic,
		components.BehaviorZombieConehead,
		components.BehaviorZombieBuckethead,
		components.BehaviorZombieEating,
		components.BehaviorZombieDying:
		return true
	default:
		return false
	}
}

// ============================================================================
// 植物攻击动画系统（重新激活 - 2025-10-24）
// ============================================================================
//
// 正确实现：使用简单的 PlayAnimation() 切换，依赖 VisibleTracks 机制显示完整身体
//
// 核心逻辑：
// - ✅ 发射子弹时切换到 anim_shooting
// - ✅ 攻击动画完成后自动切换回 anim_idle
// - ✅ 与僵尸动画实现保持一致（所有实体使用简单切换）
//

// updatePlantAttackAnimation 检测攻击动画是否完成，自动切换回 idle
// 实现攻击动画状态机（Idle ↔ Attacking）
// 添加关键帧事件监听，在精确时刻发射子弹
