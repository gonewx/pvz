package behavior

import (
	"log"
	"math/rand"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/utils"
)

func (s *BehaviorSystem) handleZombieBasicBehavior(entityID ecs.EntityID, deltaTime float64) {
	// 检查僵尸是否已激活（开场动画期间僵尸未激活，不应移动）
	if waveState, ok := ecs.GetComponent[*components.ZombieWaveStateComponent](s.entityManager, entityID); ok {
		if !waveState.IsActivated {
			// DEBUG: 记录未激活的僵尸被跳过
			log.Printf("[BehaviorSystem] Zombie %d NOT activated (wave %d), skipping behavior", entityID, waveState.WaveIndex)
			// 僵尸未激活，跳过所有行为逻辑（保持静止展示）
			return
		}
	}

	// 检查生命值（僵尸死亡逻辑）
	health, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, entityID)
	if ok {
		// 更新僵尸的受伤状态（掉手臂、掉头）
		s.updateZombieDamageState(entityID, health)

		if health.CurrentHealth <= 0 {
			// 生命值 <= 0，触发死亡状态转换
			// 根据死亡效果类型选择不同的死亡动画
			switch health.DeathEffectType {
			case components.DeathEffectExplosion:
				log.Printf("[BehaviorSystem] 僵尸 %d 被爆炸杀死 (HP=%d)，触发烧焦死亡", entityID, health.CurrentHealth)
				s.triggerZombieExplosionDeath(entityID)
			default:
				log.Printf("[BehaviorSystem] 僵尸 %d 生命值 <= 0 (HP=%d)，触发死亡", entityID, health.CurrentHealth)
				s.triggerZombieDeath(entityID)
			}
			return // 跳过正常移动逻辑
		}
	}

	// 获取位置组件
	position, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 检测植物碰撞（在移动之前）
	// 计算僵尸实际位置所在格子
	// 注意：不使用 CollisionComponent.OffsetX，该偏移量仅用于子弹碰撞检测
	// 啃食检测应基于僵尸脚底位置，确保视觉上僵尸与植物保持合理距离
	zombieCol := int((position.X - config.GridWorldStartX) / config.CellWidth)
	zombieRow := int((position.Y - config.GridWorldStartY - config.ZombieVerticalOffset - config.CellHeight/2.0) / config.CellHeight)

	// Story 8.9: 撑杆僵尸跳跃和持杆状态处理
	// 撑杆僵尸持杆时遇到植物应该跳跃而不是啃食
	shouldCheckPlantCollision := true
	if poleVault, ok := ecs.GetComponent[*components.PoleVaultComponent](s.entityManager, entityID); ok {
		if poleVault.IsJumping || poleVault.HasPole {
			// 跳跃中或持杆状态，跳过植物碰撞检测
			// 但继续执行根运动来控制位移（让动画的 _ground 数据生效）
			shouldCheckPlantCollision = false
			log.Printf("[BehaviorSystem] 撑杆僵尸 %d 跳跃/持杆状态，跳过植物碰撞检测", entityID)
		}
	}

	// 检测是否与植物在同一格子
	if shouldCheckPlantCollision {
		plantID, hasCollision := s.detectPlantCollision(zombieRow, zombieCol)
		if hasCollision {
			log.Printf("[BehaviorSystem] ✅ 僵尸 %d 检测到植物 %d，位置(%d,%d)，开始啃食！", entityID, plantID, zombieRow, zombieCol)
			// 进入啃食状态
			s.startEatingPlant(entityID, plantID)
			return // 跳过移动逻辑
		}
	}

	// 获取速度组件
	velocity, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[BehaviorSystem] ⚠️ 僵尸 %d 缺少 VelocityComponent（可能已进入啃食状态）", entityID)
		return
	}

	// 尝试使用根运动法计算位移
	// 根运动法：从 Reanim 动画的 _ground 轨道读取帧间位移增量，实现脚步与地面同步
	reanim, hasReanim := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	useRootMotion := false

	if hasReanim {
		// 尝试使用根运动法
		deltaX, deltaY, err := utils.CalculateRootMotionDelta(reanim, "_ground")

		if err == nil {
			// 成功：应用根运动位移
			position.X += deltaX
			position.Y += deltaY
			useRootMotion = true

			// DEBUG 日志（通过 verbose 标志控制）
			log.Printf("[BehaviorSystem] Zombie %d root motion: X=%.1f, deltaX=%.2f, deltaY=%.2f",
				entityID, position.X, deltaX, deltaY)
		} else {
			// 失败：记录警告并回退到固定速度法
			log.Printf("[BehaviorSystem] ⚠️ Root motion failed for zombie %d: %v, falling back to fixed velocity",
				entityID, err)
		}
	}

	// 后备方案：如果根运动失败或没有 ReanimComponent，使用固定速度法
	if !useRootMotion {
		// DEBUG: 记录僵尸速度
		log.Printf("[BehaviorSystem] Zombie %d moving: X=%.1f, VX=%.2f, VY=%.2f",
			entityID, position.X, velocity.VX, velocity.VY)

		// 更新位置：根据速度和时间增量移动僵尸
		position.X += velocity.VX * deltaTime
		position.Y += velocity.VY * deltaTime
	}

	// 边界检查：如果僵尸移出屏幕左侧，标记删除
	// 使用 config.ZombieDeletionBoundary 提供容错空间，避免僵尸刚移出就被删除
	if position.X < config.ZombieDeletionBoundary {
		log.Printf("[BehaviorSystem] 僵尸 %d 移出屏幕左侧 (X=%.1f)，标记删除", entityID, position.X)
		s.entityManager.DestroyEntity(entityID)
	}
}

func (s *BehaviorSystem) detectPlantCollision(zombieRow, zombieCol int) (ecs.EntityID, bool) {
	// 查询所有植物实体（拥有 PlantComponent）
	plantEntityList := ecs.GetEntitiesWith1[*components.PlantComponent](s.entityManager)

	// 遍历所有植物，比对网格位置
	for _, plantID := range plantEntityList {
		plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, plantID)
		if !ok {
			continue
		}

		// 跳过一次性爆炸植物（樱桃炸弹）
		// 僵尸不应该吃这类植物，而是让它们自然爆炸
		if plant.PlantType == components.PlantCherryBomb {
			continue
		}

		// 检查是否在同一格子
		if plant.GridRow == zombieRow && plant.GridCol == zombieCol {
			return plantID, true
		}
	}

	// 没有找到植物
	return 0, false
}

// changeZombieAnimation 切换僵尸动画状态
// 参数:
//   - zombieID: 僵尸实体ID
//   - newState: 新的动画状态

func (s *BehaviorSystem) changeZombieAnimation(zombieID ecs.EntityID, newState components.ZombieAnimState) {
	// 获取行为组件
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
	if !ok {
		return
	}

	// 如果状态没有变化，不需要切换动画
	if behavior.ZombieAnimState == newState {
		return
	}

	// 更新状态
	behavior.ZombieAnimState = newState

	// 检查是否为旗帜僵尸且旗帜已受损（用于选择受损动画）
	isFlagZombieDamaged := false
	if behavior.UnitID == "zombie_flag" {
		if health, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, zombieID); ok {
			isFlagZombieDamaged = health.ArmLost
		}
	}

	// 根据状态确定组合名称
	// 使用配置驱动的动画播放
	var comboName string
	switch newState {
	case components.ZombieAnimIdle:
		comboName = "idle"
	case components.ZombieAnimWalking:
		// 随机选择 walk 或 walk2 动画
		baseWalk := "walk"
		if rand.Float32() < 0.5 {
			baseWalk = "walk2"
		}
		// 旗帜僵尸受损时使用 walk_damaged 或 walk2_damaged 动画
		if isFlagZombieDamaged {
			comboName = baseWalk + "_damaged"
		} else {
			comboName = baseWalk
		}
	case components.ZombieAnimEating:
		// 旗帜僵尸受损时使用 eat_damaged 动画
		if isFlagZombieDamaged {
			comboName = "eat_damaged"
		} else {
			comboName = "eat"
		}
	case components.ZombieAnimDying:
		// 使用 BehaviorComponent 中存储的 UnitID
		unitID := behavior.UnitID
		if unitID == "" {
			unitID = "zombie" // 后备默认值
		}

		// 随机选择 death 或 death2 动画
		deathCombo := "death"
		if rand.Float32() < 0.5 {
			deathCombo = "death2"
		}

		// 使用组件通信替代直接调用
		ecs.AddComponent(s.entityManager, zombieID, &components.AnimationCommandComponent{
			UnitID:    unitID,
			ComboName: deathCombo,
			Processed: false,
		})
		log.Printf("[BehaviorSystem] 僵尸 %d (%s) 添加死亡动画命令 (%s)", zombieID, unitID, deathCombo)
		return
	default:
		return
	}

	// 使用 BehaviorComponent 中存储的 UnitID
	unitID := behavior.UnitID
	if unitID == "" {
		unitID = "zombie" // 后备默认值
	}

	// 使用组件通信替代直接调用
	// 使用 AnimationCommand 组件播放新动画组合
	ecs.AddComponent(s.entityManager, zombieID, &components.AnimationCommandComponent{
		UnitID:    unitID,
		ComboName: comboName,
		Processed: false,
	})
	log.Printf("[BehaviorSystem] 僵尸 %d (%s) 添加动画命令: %s（配置驱动）", zombieID, unitID, comboName)
}

// handleZombieFlagBehavior 处理旗帜僵尸的行为逻辑
// 旗帜僵尸与普通僵尸行为完全相同，只是外观不同（显示旗帜手）
func (s *BehaviorSystem) handleZombieFlagBehavior(entityID ecs.EntityID, deltaTime float64) {
	// 旗帜僵尸的行为与普通僵尸相同
	s.handleZombieBasicBehavior(entityID, deltaTime)
}

// updateTriggerZombieMovement 更新触发僵尸的移动（游戏冻结期间）
// Story 8.8: 简化的移动逻辑，只更新位置，不检测碰撞和啃食
// 用于 Phase 2 期间让触发僵尸继续走出屏幕
func (s *BehaviorSystem) updateTriggerZombieMovement(entityID ecs.EntityID, deltaTime float64) {
	// 获取位置组件
	position, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 获取速度组件
	velocity, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[BehaviorSystem] ⚠️ 触发僵尸 %d 缺少 VelocityComponent", entityID)
		return
	}

	// 更新位置：根据速度和时间增量移动僵尸
	position.X += velocity.VX * deltaTime
	position.Y += velocity.VY * deltaTime

	// DEBUG: 记录僵尸移动
	log.Printf("[BehaviorSystem] Trigger zombie %d moving: X=%.1f, Y=%.1f, VX=%.2f, VY=%.2f", entityID, position.X, position.Y, velocity.VX, velocity.VY)

	// 注意：不检测边界删除，由 ZombiesWonPhaseSystem 处理
}
