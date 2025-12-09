package behavior

import (
	"log"
	"math/rand"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/types"
	"github.com/decker502/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
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

	// 获取碰撞组件，用于计算碰撞盒中心
	collision, hasCollision := ecs.GetComponent[*components.CollisionComponent](s.entityManager, entityID)
	collisionOffsetX := 0.0
	if hasCollision {
		collisionOffsetX = collision.OffsetX
	}

	// 检测植物碰撞（在移动之前）
	// 计算僵尸碰撞盒中心所在格子
	// 使用碰撞盒中心而非实体位置，确保旗帜僵尸等有偏移的僵尸正确检测
	zombieCol := int((position.X + collisionOffsetX - config.GridWorldStartX) / config.CellWidth)
	zombieRow := int((position.Y - config.GridWorldStartY - config.ZombieVerticalOffset - config.CellHeight/2.0) / config.CellHeight)

	// 检测是否与植物在同一格子
	plantID, hasCollision := s.detectPlantCollision(zombieRow, zombieCol)
	if hasCollision {
		log.Printf("[BehaviorSystem] ✅ 僵尸 %d 检测到植物 %d，位置(%d,%d)，开始啃食！", entityID, plantID, zombieRow, zombieCol)
		// 进入啃食状态
		s.startEatingPlant(entityID, plantID)
		return // 跳过移动逻辑
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

// triggerZombieDeath 触发僵尸死亡状态转换
// 当僵尸生命值 <= 0 时调用，将僵尸从正常行为状态切换到死亡动画播放状态
// 根据 DeathEffectType 决定是否触发肢体掉落粒子效果：
// - DeathEffectNormal: 触发头部、旗帜掉落粒子效果
// - DeathEffectInstant: 不触发肢体掉落效果（如坚果保龄球撞击）
// 注意：手臂掉落粒子效果在 updateZombieDamageState 中触发（受伤时）

func (s *BehaviorSystem) triggerZombieDeath(entityID ecs.EntityID) {
	// 1. 切换行为类型为 BehaviorZombieDying
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[BehaviorSystem] 僵尸 %d 缺少 BehaviorComponent，无法触发死亡", entityID)
		return
	}
	behavior.Type = components.BehaviorZombieDying
	log.Printf("[BehaviorSystem] ��尸 %d 行为切换为 BehaviorZombieDying", entityID)

	// 获取生命组件，检查死亡效果类型
	health, _ := ecs.GetComponent[*components.HealthComponent](s.entityManager, entityID)
	showLimbEffects := health == nil || health.DeathEffectType == components.DeathEffectNormal

	// 获取僵尸位置，用于触发粒子效果
	position, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[BehaviorSystem] 警告：僵尸 %d 缺少 PositionComponent，无法触发粒子效果", entityID)
	} else if showLimbEffects {
		// 只有普通死亡才触发肢体掉落效果
		// 检测僵尸行进方向，计算粒子角度偏移
		// 粒子效果应该在僵尸行进的反方向飞出
		//
		// 角度系统：标准屏幕坐标系（0°=右，90°=下，180°=左，270°=上）
		// ZombieHead 配置：LaunchAngle [150-185°] ≈ 向左下
		// 该配置是为**向右走的僵尸**设计的（头向左后方飞）
		//
		// 我们游戏中僵尸通常向左走，需要翻转方向
		angleOffset := 180.0 // 默认翻转（适合僵尸向左走）
		velocity, hasVelocity := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID)
		if hasVelocity {
			if velocity.VX > 0 {
				// 僵尸向右走 → 配置已经正确 → 不翻转
				angleOffset = 0.0
			} else {
				// 僵尸向左走 → 配置方向相反 → 翻转 180°
				// [150-185°] + 180° = [330-365°] = [-30 to 5°] → 向右后方
				angleOffset = 180.0
			}
			log.Printf("[BehaviorSystem] 僵尸 %d 方向: VX=%.1f → 粒子角度偏移=%.0f°", entityID, velocity.VX, angleOffset)
		}

		// 播放头部掉落音效
		if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
			audioManager.PlaySound("SOUND_LIMBS_POP")
		}

		// 触发僵尸头部掉落粒子效果
		_, err := entities.CreateParticleEffect(
			s.entityManager,
			s.resourceManager,
			"ZombieHead", // 粒子效果名称（不带.xml后缀）
			position.X, position.Y,
			angleOffset, // 传递角度偏移
		)
		if err != nil {
			log.Printf("[BehaviorSystem] 警告：创建僵尸头部掉落粒子效果失败: %v", err)
			// 不阻塞游戏逻辑，游戏继续运行
		} else {
			log.Printf("[BehaviorSystem] 僵尸 %d 触发头部掉落粒子效果，位置: (%.1f, %.1f)", entityID, position.X, position.Y)
		}

		// 旗帜僵尸特殊处理：触发旗帜掉落粒子效果
		if behavior.UnitID == "zombie_flag" {
			_, err := entities.CreateParticleEffect(
				s.entityManager,
				s.resourceManager,
				"ZombieFlag", // ���帜掉落粒子效果
				position.X, position.Y,
				angleOffset, // 与头部掉落方向一致
			)
			if err != nil {
				log.Printf("[BehaviorSystem] 警告：创建旗帜掉落粒子效果失败: %v", err)
			} else {
				log.Printf("[BehaviorSystem] 旗帜僵尸 %d 触发旗帜掉落粒子效果，位置: (%.1f, %.1f)", entityID, position.X, position.Y)
			}
		}
	} else {
		log.Printf("[BehaviorSystem] 僵尸 %d 瞬间死亡，跳过肢体掉落效果", entityID)
	}

	// 2. 隐藏头部轨道（头掉落效果）
	// 直接修改 HiddenTracks 字段而不调用废弃的 HideTrack API
	// 注意：旗帜僵尸的旗帜隐藏在 zombie_flag.yaml 的 death/death_damaged 配置中处理
	// 只有普通死亡才隐藏头部轨道
	if showLimbEffects {
		if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
			if reanim.HiddenTracks == nil {
				reanim.HiddenTracks = make(map[string]bool)
			}
			// 隐藏所有头部相关轨道
			headTracks := []string{"anim_head1", "anim_head2"}
			for _, trackName := range headTracks {
				reanim.HiddenTracks[trackName] = true
			}
			log.Printf("[BehaviorSystem] 僵尸 %d 头部掉落，隐藏轨道: %v", entityID, headTracks)
		}
	}

	// 3. 移除 VelocityComponent（停止移动）
	ecs.RemoveComponent[*components.VelocityComponent](s.entityManager, entityID)
	log.Printf("[BehaviorSystem] 僵尸 %d 移除速度组件，停止移动", entityID)

	// 4. 使用 AnimationCommand 组件播放死亡动画（不循环）
	// 使用组件通信替代直接调用
	// 使用配置驱动的动画组合（自动隐藏装备轨道）
	// 旗帜僵尸特殊处理：根据 ArmLost 选择死亡动画
	// 随机选择 death 或 death2 动画
	deathComboName := "death"
	if rand.Float32() < 0.5 {
		deathComboName = "death2"
	}
	unitID := behavior.UnitID
	if unitID == "" {
		unitID = types.UnitIDZombie // 后备默认值
	}
	if unitID == "zombie_flag" {
		if health, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, entityID); ok {
			if health.ArmLost {
				deathComboName = deathComboName + "_damaged"
				log.Printf("[BehaviorSystem] 旗帜僵尸 %d 使用受损死亡动画 (%s)", entityID, deathComboName)
			}
		}
	}
	ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
		UnitID:    unitID,
		ComboName: deathComboName,
		Processed: false,
	})
	log.Printf("[BehaviorSystem] 僵尸 %d 添加死亡动画命令 (%s/%s)", entityID, unitID, deathComboName)

	// 设置为不循环
	if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
		reanim.IsLooping = false
	}

	log.Printf("[BehaviorSystem] 僵尸 %d 死亡动画已开始播放 (anim_death, 不循环)", entityID)
}

// handleZombieDyingBehavior 处理僵尸死亡动画播放
// 当死亡动画完成后，删除僵尸实体并增加击杀计数

func (s *BehaviorSystem) handleZombieDyingBehavior(entityID ecs.EntityID) {
	// 获取 ReanimComponent
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		// 如果没有 ReanimComponent，直接删除僵尸
		log.Printf("[BehaviorSystem] 死亡中的僵尸 %d 缺少 ReanimComponent，直接删除", entityID)
		// 僵尸死亡，增加计数
		s.gameState.IncrementZombiesKilled()
		s.entityManager.DestroyEntity(entityID)
		return
	}

	// 检查死亡动画是否完成
	// 使用 IsFinished 标志来判断非循环动画是否已完成
	if reanim.IsFinished {
		// 使用 CurrentFrame 替代 AnimStates
		log.Printf("[BehaviorSystem] 僵尸 %d 死亡动画完成 (frame %d)，删除实体",
			entityID, reanim.CurrentFrame)
		// 僵尸死亡，增加计数
		s.gameState.IncrementZombiesKilled()
		s.entityManager.DestroyEntity(entityID)
	}
}

// updateZombieDamageState 根据生命值更新僵尸的受伤状态
// 僵尸有三个受伤阶段：
// 1. 健康（HP > 90）：完整外观
// 2. 掉手臂（HP <= 90 且 HP > 0）：隐藏外侧手臂
// 3. 掉头（HP <= 0）：无头状态（在 triggerZombieDeath 中处理）

func (s *BehaviorSystem) updateZombieDamageState(entityID ecs.EntityID, health *components.HealthComponent) {
	// 生命值阈值：90（33%，根据原版游戏数据）
	const armLostThreshold = 90

	// 检查是否应该掉手臂（生命值 <= 90 且手臂尚未掉落）
	if health.CurrentHealth <= armLostThreshold && !health.ArmLost {
		// 标记手臂已掉落，防止重复触发
		health.ArmLost = true

		// 播放手臂掉落音效
		if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
			audioManager.PlaySound("SOUND_LIMBS_POP")
		}

		// 获取行为组件，检查僵尸类型
		behavior, hasBehavior := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)

		// 隐藏手臂轨道（手臂掉落效果）
		// 直接修改 HiddenTracks 字段而不调用废弃的 HideTrack API
		if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
			if reanim.HiddenTracks == nil {
				reanim.HiddenTracks = make(map[string]bool)
			}
			armTracks := []string{"Zombie_outerarm_hand", "Zombie_outerarm_upper", "Zombie_outerarm_lower"}
			for _, trackName := range armTracks {
				reanim.HiddenTracks[trackName] = true
			}
			log.Printf("[BehaviorSystem] 僵尸 %d 手臂掉落，隐藏轨道: %v", entityID, armTracks)
		}

		log.Printf("[BehaviorSystem] 僵尸 %d 手臂掉落 (HP=%d/%d)",
			entityID, health.CurrentHealth, health.MaxHealth)

		// 旗帜僵尸特殊处理：根据当前状态切换到对应的受损动画
		if hasBehavior && behavior.UnitID == "zombie_flag" {
			var damagedComboName string
			switch behavior.Type {
			case components.BehaviorZombieEating:
				damagedComboName = "eat_damaged"
			case components.BehaviorZombieBasic, components.BehaviorZombieFlag:
				damagedComboName = "walk_damaged"
			}
			if damagedComboName != "" {
				ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
					UnitID:    "zombie_flag",
					ComboName: damagedComboName,
					Processed: false,
				})
				log.Printf("[BehaviorSystem] 旗帜僵尸 %d 受损，切换到 %s 动画", entityID, damagedComboName)
			}
		}

		// 获取僵尸位置，用于触发粒子效果
		position, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		if !ok {
			log.Printf("[BehaviorSystem] 警告：僵尸 %d 缺少 PositionComponent，无法触发手臂掉落粒子", entityID)
			return
		}

		// 检测僵尸行进方向，计算粒子角度偏移
		angleOffset := 180.0 // 默认翻转（适合僵尸向左走）
		velocity, hasVelocity := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID)
		if hasVelocity {
			if velocity.VX > 0 {
				angleOffset = 0.0 // 僵尸向右走
			} else {
				angleOffset = 180.0 // 僵尸向左走
			}
		}

		// 触发僵尸手臂掉落粒子效果
		_, err := entities.CreateParticleEffect(
			s.entityManager,
			s.resourceManager,
			"ZombieArm", // 粒子效果名称（不带.xml后缀）
			position.X, position.Y,
			angleOffset, // 角度偏移
		)
		if err != nil {
			log.Printf("[BehaviorSystem] 警告：创建僵尸手臂掉落粒子效果失败: %v", err)
		} else {
			log.Printf("[BehaviorSystem] 僵尸 %d 触发手臂掉落粒子效果，位置: (%.1f, %.1f)", entityID, position.X, position.Y)
		}
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

// startEatingPlant 开始啃食植物
// 参数:
//   - zombieID: 僵尸实体ID
//   - plantID: 植物实体ID

func (s *BehaviorSystem) startEatingPlant(zombieID, plantID ecs.EntityID) {
	log.Printf("[BehaviorSystem] 僵尸 %d 开始啃食植物 %d", zombieID, plantID)

	// 1. 移除僵尸的 VelocityComponent（停止移动）
	ecs.RemoveComponent[*components.VelocityComponent](s.entityManager, zombieID)

	// 2. 重置根运动状态（防止动画切换导致位移跳变）
	// 虽然啃食时僵尸不移动，但重置状态确保恢复移动时不会发生问题
	if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, zombieID); ok {
		reanim.LastGroundX = 0
		reanim.LastGroundY = 0
		reanim.LastAnimFrame = -1
		reanim.AccumulatedDeltaX = 0
		reanim.AccumulatedDeltaY = 0
	}

	// 3. 在切换类型之前，先记住原始僵尸类型（用于选择正确的啃食动画）
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
	if !ok {
		return
	}
	originalZombieType := behavior.Type // 记住原始类型

	// 4. 切换 BehaviorComponent.Type 为 BehaviorZombieEating
	behavior.Type = components.BehaviorZombieEating
	// 初始化啃食动画帧跟踪（用于伤害和音效同步）
	// -1 表示尚未开始，首次进入会触发伤害
	behavior.LastEatAnimFrame = -1

	// 切换僵尸动画为啃食状态
	s.changeZombieAnimation(zombieID, components.ZombieAnimEating)

	// 待迁移到 ReanimComponent
	// 5. 根据原始僵尸类型加载对应的啃食动画
	// var eatFrames []*ebiten.Image

	_ = originalZombieType // 临时避免未使用警告
	/*
		switch originalZombieType {
		case components.BehaviorZombieConehead:
			// 路障僵尸啃食动画
			eatFrames, _ = utils.LoadConeheadZombieEatAnimation(s.resourceManager)
			log.Printf("[BehaviorSystem] 路障僵尸 %d 开始啃食，使用路障僵尸啃食动画", zombieID)
		case components.BehaviorZombieBuckethead:
			// 铁桶僵尸啃食动画
			eatFrames, _ = utils.LoadBucketheadZombieEatAnimation(s.resourceManager)
			log.Printf("[BehaviorSystem] 铁桶僵尸 %d 开始啃食，使用铁桶僵尸啃食动画", zombieID)
		default:
			// 普通僵尸或其他类型
			eatFrames = utils.LoadZombieEatAnimation(s.resourceManager)
		}

		// 待迁移到 ReanimComponent
		// 6. 替换 AnimationComponent 为啃食动画
		// animComp, ok := s.entityManager.GetComponent(zombieID, reflect.TypeOf(&components.AnimationComponent{}))
		// if ok {
		// 	anim := animComp.(*components.AnimationComponent)
		// 	anim.Frames = eatFrames
		// 	anim.FrameSpeed = config.ZombieEatFrameSpeed
		// 	anim.CurrentFrame = 0
		// 	anim.FrameCounter = 0
		// 	anim.IsLooping = true
		// 	anim.IsFinished = false
		// }
	*/
}

// stopEatingAndResume 停止啃食并恢复移动
// 参数:
//   - zombieID: 僵尸实体ID

func (s *BehaviorSystem) stopEatingAndResume(zombieID ecs.EntityID) {
	log.Printf("[BehaviorSystem] 僵尸 %d 结束啃食，恢复移动", zombieID)

	// 1. 切换 BehaviorComponent.Type 回 BehaviorZombieBasic
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
	if ok {
		behavior.Type = components.BehaviorZombieBasic
		// 重置啃食动画帧跟踪
		behavior.LastEatAnimFrame = -1
	}

	// 2. 切换僵尸动画回行走状态
	s.changeZombieAnimation(zombieID, components.ZombieAnimWalking)

	// 3. 重置根运动状态（防止动画切换导致位移跳变）
	// 当从啃食动画切换回行走动画时，_ground 轨道的坐标会发生跳变
	// 如果不重置这些状态，会导致僵尸瞬间后退
	if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, zombieID); ok {
		reanim.LastGroundX = 0
		reanim.LastGroundY = 0
		reanim.LastAnimFrame = -1
		reanim.AccumulatedDeltaX = 0
		reanim.AccumulatedDeltaY = 0
		log.Printf("[BehaviorSystem] 僵尸 %d 重置根运动状态", zombieID)
	}

	// 4. 恢复 VelocityComponent
	ecs.AddComponent(s.entityManager, zombieID, &components.VelocityComponent{
		VX: config.ZombieWalkSpeed,
		VY: 0,
	})
}

// handleZombieEatingBehavior 处理僵尸啃食植物的行为
// 参数:
//   - entityID: 僵尸实体ID
//   - deltaTime: 帧间隔时间

func (s *BehaviorSystem) handleZombieEatingBehavior(entityID ecs.EntityID, deltaTime float64) {
	// DEBUG: 添加日志确认函数被调用
	log.Printf("[BehaviorSystem] 🍴 处理僵尸 %d 啃食行为", entityID)

	// 获取行为组件和动画组件，用于伤害和音效同步
	behavior, hasBehavior := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
	reanim, hasReanim := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)

	// 基于动画帧触发伤害和音效（完全同步）
	// 普通僵尸（双手啃食）：在动画开始和中间点各触发一次
	// 旗帜僵尸（单手啃食）或掉了手臂的僵尸：只在动画开始时触发一次
	shouldDealDamage := false
	if hasBehavior && hasReanim {
		currentFrame := reanim.CurrentFrame
		lastFrame := behavior.LastEatAnimFrame

		// 判断是否是单手僵尸：
		// 1. 旗帜僵尸天生单手（拿旗的手不用于啃食）
		// 2. 任何僵尸掉了手臂后都变成单手
		isSingleHand := behavior.UnitID == "zombie_flag"
		if health, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, entityID); ok {
			if health.ArmLost {
				isSingleHand = true
			}
		}

		// 获取动画总帧数（用于计算中间点）
		totalFrames := 0
		if animVisibles, ok := reanim.AnimVisiblesMap["anim_eat"]; ok {
			for _, v := range animVisibles {
				if v == 0 {
					totalFrames++
				}
			}
		}
		midFrame := totalFrames / 2

		// 检测动画循环：当前帧小于上一帧，说明动画循环了
		// 或者第一次进入啃食状态（lastFrame == -1）
		if lastFrame == -1 || currentFrame < lastFrame {
			// 动画循环开始，触发伤害和音效
			shouldDealDamage = true
			s.playEatingSound()
			log.Printf("[BehaviorSystem] 🔊 僵尸 %d 啃食动画循环，触发伤害+音效（帧 %d → %d）",
				entityID, lastFrame, currentFrame)
		} else if !isSingleHand && totalFrames > 0 {
			// 双手僵尸：检测是否跨过中间点，触发第二次伤害和音效
			if lastFrame < midFrame && currentFrame >= midFrame {
				shouldDealDamage = true
				s.playEatingSound()
				log.Printf("[BehaviorSystem] 🔊 僵尸 %d 双手啃食中间点，触发伤害+音效（帧 %d → %d，mid=%d）",
					entityID, lastFrame, currentFrame, midFrame)
			}
		}

		// 更新上一帧记录
		behavior.LastEatAnimFrame = currentFrame
	}

	// 检查生命值并更新受伤状态（掉手臂、掉头）
	health, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, entityID)
	if ok {

		// 更新僵尸的受伤状态（掉手臂）
		s.updateZombieDamageState(entityID, health)

		// 检查生命值是否归零（即使在啃食状态也要检查）
		if health.CurrentHealth <= 0 {
			// 根据死亡效果类型选择不同的死亡动画
			switch health.DeathEffectType {
			case components.DeathEffectExplosion:
				log.Printf("[BehaviorSystem] 啃食中的僵尸 %d 被爆炸杀死 (HP=%d)，触发烧焦死亡", entityID, health.CurrentHealth)
				s.triggerZombieExplosionDeath(entityID)
			default:
				log.Printf("[BehaviorSystem] 啃食中的僵尸 %d 生命值 <= 0 (HP=%d)，触发死亡", entityID, health.CurrentHealth)
				s.triggerZombieDeath(entityID)
			}
			return
		}
	}

	// 检查护甲状态（护甲僵尸即使在啃食也需要检测护甲破坏）
	// 当护甲被打掉时，需要立即隐藏护甲轨道并更新 UnitID，
	// 防止恢复移动时使用错误的动画配置导致护甲重新显示
	armor, hasArmor := ecs.GetComponent[*components.ArmorComponent](s.entityManager, entityID)
	if hasArmor && armor.CurrentArmor <= 0 {
		s.handleArmorDestroyedWhileEating(entityID, behavior)
	}

	// 获取僵尸当前网格位置
	pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 获取碰撞组件，用于计算碰撞盒中心
	collision, hasCollisionComp := ecs.GetComponent[*components.CollisionComponent](s.entityManager, entityID)
	collisionOffsetX := 0.0
	if hasCollisionComp {
		collisionOffsetX = collision.OffsetX
	}

	// 计算僵尸碰撞盒中心所在格子
	// 使用碰撞盒中心而非实体位置，确保旗帜僵尸等有偏移的僵尸正确检测
	zombieCol := int((pos.X + collisionOffsetX - config.GridWorldStartX) / config.CellWidth)
	zombieRow := int((pos.Y - config.GridWorldStartY - config.ZombieVerticalOffset - config.CellHeight/2.0) / config.CellHeight)

	// 检测植物
	plantID, hasPlant := s.detectPlantCollision(zombieRow, zombieCol)

	if !hasPlant {
		// 植物不存在（可能被其他僵尸吃掉），恢复移动
		s.stopEatingAndResume(entityID)
		return
	}

	// 基于动画帧触发伤害（与音效同步）
	if shouldDealDamage {
		// 植物存在，造成伤害
		plantHealth, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, plantID)
		if ok {
			plantHealth.CurrentHealth -= config.ZombieEatingDamage

			// 坚果墙被啃食时触发小碎屑粒子效果和发光效果
			// WallnutEatSmall: 每次啃食伤害时触发
			// WallnutEatLarge: 在受损状态变化时触发（在 handleWallnutBehavior 中）
			if plantComp, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, plantID); ok {
				if plantComp.PlantType == components.PlantWallnut {
					// 粒子位置：僵尸嘴巴位置（啃食接触点）
					particleX := pos.X + config.ZombieEatParticleOffsetX
					particleY := pos.Y + config.ZombieEatParticleOffsetY
					_, err := entities.CreateParticleEffect(
						s.entityManager,
						s.resourceManager,
						"WallnutEatSmall",
						particleX,
						particleY,
					)
					if err != nil {
						log.Printf("[BehaviorSystem] 警告：创建坚果墙小碎屑粒子效果失败: %v", err)
					}

					// 添加发光效果（一闪一闪）
					ecs.AddComponent(s.entityManager, plantID, &components.WallnutHitGlowComponent{
						Intensity: 1.0,
						FadeSpeed: config.WallnutHitGlowFadeSpeed,
						ColorR:    config.WallnutHitGlowColorR,
						ColorG:    config.WallnutHitGlowColorG,
						ColorB:    config.WallnutHitGlowColorB,
					})
				}
			}

			log.Printf("[BehaviorSystem] 僵尸 %d 啃食植物 %d，造成 %d 伤害，剩余生命值 %d",
				entityID, plantID, config.ZombieEatingDamage, plantHealth.CurrentHealth)

			// 检查植物是否死亡
			if plantHealth.CurrentHealth <= 0 {
				log.Printf("[BehaviorSystem] 植物 %d 被吃掉，删除实体", plantID)

				// 释放网格占用状态，允许重新种植
				if plantComp, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, plantID); ok {
					if s.lawnGridSystem != nil && s.lawnGridEntityID != 0 {
						err := s.lawnGridSystem.ReleaseCell(s.lawnGridEntityID, plantComp.GridCol, plantComp.GridRow)
						if err != nil {
							log.Printf("[BehaviorSystem] 警告：释放网格占用失败: %v", err)
						} else {
							log.Printf("[BehaviorSystem] 网格 (%d, %d) 已释放", plantComp.GridCol, plantComp.GridRow)
						}
					} else {
						log.Printf("[BehaviorSystem] 警告：无法释放网格，lawnGridSystem=%v, lawnGridEntityID=%d",
							s.lawnGridSystem != nil, s.lawnGridEntityID)
					}
				}

				s.entityManager.DestroyEntity(plantID)
				// 恢复僵尸移动
				s.stopEatingAndResume(entityID)
				return
			}
		} else {
			// 植物没有 HealthComponent（不应该发生，但作为保护措施）
			log.Printf("[BehaviorSystem] 警告：植物 %d 没有 HealthComponent，直接删除", plantID)

			// Bug Fix: 释放网格占用状态，允许重新种植
			if plantComp, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, plantID); ok {
				if s.lawnGridSystem != nil && s.lawnGridEntityID != 0 {
					err := s.lawnGridSystem.ReleaseCell(s.lawnGridEntityID, plantComp.GridCol, plantComp.GridRow)
					if err != nil {
						log.Printf("[BehaviorSystem] 警告：释放网格占用失败: %v", err)
					} else {
						log.Printf("[BehaviorSystem] 网格 (%d, %d) 已释放", plantComp.GridCol, plantComp.GridRow)
					}
				} else {
					log.Printf("[BehaviorSystem] 警告：无法释放网格，lawnGridSystem=%v, lawnGridEntityID=%d",
						s.lawnGridSystem != nil, s.lawnGridEntityID)
				}
			}

			s.entityManager.DestroyEntity(plantID)
			s.stopEatingAndResume(entityID)
			return
		}
	}
}

// playEatingSound 播放僵尸啃食音效
func (s *BehaviorSystem) playEatingSound() {
	// 使用 AudioManager 统一管理音效（Story 10.9）
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_CHOMP")
	}
}

// handleWallnutBehavior 处理坚果墙的行为逻辑
// 坚果墙没有主动行为（不生产阳光，不攻击），但会根据生命值百分比切换外观状态
// 外观状态：完好(>66%) → 轻伤(33-66%) → 重伤(<33%)

func (s *BehaviorSystem) handleConeheadZombieBehavior(entityID ecs.EntityID, deltaTime float64) {
	// 检查僵尸是否已激活（开场动画期间僵尸未激活，不应移动）
	if waveState, ok := ecs.GetComponent[*components.ZombieWaveStateComponent](s.entityManager, entityID); ok {
		if !waveState.IsActivated {
			// 僵尸未激活，跳过所有行为逻辑（保持静止展示）
			return
		}
	}

	// 首先检查护甲状态
	armor, ok := ecs.GetComponent[*components.ArmorComponent](s.entityManager, entityID)
	if !ok {
		// 没有护甲组件（不应该发生），退化为普通僵尸行为
		log.Printf("[BehaviorSystem] 警告：路障僵尸 %d 缺少 ArmorComponent，转为普通僵尸", entityID)
		s.handleZombieBasicBehavior(entityID, deltaTime)
		return
	}

	// 如果护甲已破坏，切换为普通僵尸
	if armor.CurrentArmor <= 0 {
		// 检查是否已经切换过（避免每帧都触发）
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if ok {
			if behavior.Type == components.BehaviorZombieConehead {
				// 首次护甲破坏，执行切换
				log.Printf("[BehaviorSystem] 路障僵尸 %d 护甲破坏，切换为普通僵尸", entityID)

				// 1. 改变行为类型为普通僵尸
				behavior.Type = components.BehaviorZombieBasic

				// 2. 更新 UnitID 为普通僵尸，防止后续动画切换使用错误配置
				behavior.UnitID = "zombie"

				// 3. 隐藏路障轨道（使用 HiddenTracks 黑名单）
				reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
				if ok {
					if reanim.HiddenTracks == nil {
						reanim.HiddenTracks = make(map[string]bool)
					}
					reanim.HiddenTracks["anim_cone"] = true // 隐藏路障
					log.Printf("[BehaviorSystem] 路障僵尸 %d 隐藏 anim_cone 轨道", entityID)
				}

				// 4. 触发路障掉落粒子效果
				position, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
				velocity, hasVel := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID)
				if hasPos {
					// 粒子发射角度调整
					angleOffset := 180.0
					if hasVel && velocity.VX > 0 {
						angleOffset = 0.0
					}

					// 创建掉落粒子
					_, err := entities.CreateParticleEffect(
						s.entityManager,
						s.resourceManager,
						"ZombieTrafficCone", // 掉落粒子配置文件名
						position.X, position.Y,
						angleOffset,
					)
					if err != nil {
						log.Printf("[BehaviorSystem] 警告：创建路障掉落粒子失败: %v", err)
					} else {
						log.Printf("[BehaviorSystem] 路障僵尸 %d 触发路障掉落效果", entityID)
					}
				}
			}
		}

		// 护甲已破坏，继续以普通僵尸行为运作
		s.handleZombieBasicBehavior(entityID, deltaTime)
		return
	}

	// 护甲完好，更新外观状态（根据受损程度切换图片）
	s.updateArmorVisualState(entityID, armor, "cone")

	// 执行普通僵尸的基本行为（移动、碰撞检测、啃食植物）
	s.handleZombieBasicBehavior(entityID, deltaTime)
}

func (s *BehaviorSystem) handleBucketheadZombieBehavior(entityID ecs.EntityID, deltaTime float64) {
	// 检查僵尸是否已激活（开场动画期间僵尸未激活，不应移动）
	if waveState, ok := ecs.GetComponent[*components.ZombieWaveStateComponent](s.entityManager, entityID); ok {
		if !waveState.IsActivated {
			// 僵尸未激活，跳过所有行为逻辑（保持静止展示）
			return
		}
	}

	// 首先检查护甲状态
	armor, ok := ecs.GetComponent[*components.ArmorComponent](s.entityManager, entityID)
	if !ok {
		// 没有护甲组件（不应该发生），退化为普通僵尸行为
		log.Printf("[BehaviorSystem] 警告：铁桶僵尸 %d 缺少 ArmorComponent，转为普通僵尸", entityID)
		s.handleZombieBasicBehavior(entityID, deltaTime)
		return
	}

	// 如果护甲已破坏，切换为普通僵尸
	if armor.CurrentArmor <= 0 {
		// 检查是否已经切换过（避免每帧都触发）
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if ok {
			if behavior.Type == components.BehaviorZombieBuckethead {
				// 首次护甲破坏，执行切换
				log.Printf("[BehaviorSystem] 铁桶僵尸 %d 护甲破坏，切换为普通僵尸", entityID)

				// 1. 改变行为类型为普通僵尸
				behavior.Type = components.BehaviorZombieBasic

				// 2. 更新 UnitID 为普通僵尸，防止后续动画切换使用错误配置
				behavior.UnitID = "zombie"

				// 3. 隐藏铁桶轨道（使用 HiddenTracks 黑名单）
				reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
				if ok {
					if reanim.HiddenTracks == nil {
						reanim.HiddenTracks = make(map[string]bool)
					}
					reanim.HiddenTracks["anim_bucket"] = true // 隐藏铁桶
					log.Printf("[BehaviorSystem] 铁桶僵尸 %d 隐藏 anim_bucket 轨道", entityID)
				}

				// 4. 触发铁桶掉落粒子效果
				position, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
				velocity, hasVel := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID)
				if hasPos {
					// 粒子发射角度调整
					angleOffset := 180.0
					if hasVel && velocity.VX > 0 {
						angleOffset = 0.0
					}

					// 创建掉落粒子
					_, err := entities.CreateParticleEffect(
						s.entityManager,
						s.resourceManager,
						"ZombiePail", // 掉落粒子配置文件名
						position.X, position.Y,
						angleOffset,
					)
					if err != nil {
						log.Printf("[BehaviorSystem] 警告：创建铁桶掉落粒子失败: %v", err)
					} else {
						log.Printf("[BehaviorSystem] 铁桶僵尸 %d 触发铁桶掉落效果", entityID)
					}
				}
			}
		}

		// 护甲已破坏，继续以普通僵尸行为运作
		s.handleZombieBasicBehavior(entityID, deltaTime)
		return
	}

	// 护甲完好，更新外观状态（根据受损程度切换图片）
	s.updateArmorVisualState(entityID, armor, "bucket")

	// 执行普通僵尸的基本行为（移动、碰撞检测、啃食植物）
	s.handleZombieBasicBehavior(entityID, deltaTime)
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

// triggerZombieExplosionDeath 触发僵尸爆炸烧焦死亡动画
//
// 当僵尸被爆炸类攻击（樱桃炸弹、土豆雷、辣椒等）杀死时调用此方法
// 切换为烧焦死亡行为，播放 Zombie_charred.reanim 动画
//
// 参数:
//   - entityID: 僵尸实体ID
//
// 使用场景:
//   - 樱桃炸弹爆炸杀死僵尸 (Story 5.4)
//   - 土豆雷爆炸杀死僵尸（未来实现）
//
// 技术说明:
//   - 使用 AnimationCommand 组件触发动画切换
//   - ReanimSystem 的 PlayCombo 负责处理单位切换（Story 5.4.1 重构）
//   - 不隐藏头部轨道（烧焦动画中僵尸整体烧焦，头不掉落）
//   - 不触发粒子效果（爆炸效果已在爆炸时播放）
//   - 参考实现: triggerZombieDeath() (普通死亡)
func (s *BehaviorSystem) triggerZombieExplosionDeath(entityID ecs.EntityID) {
	// 1. 切换行为类型为 BehaviorZombieDyingExplosion
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[BehaviorSystem] 僵尸 %d 缺少 BehaviorComponent，无法触发爆炸死亡", entityID)
		return
	}
	behavior.Type = components.BehaviorZombieDyingExplosion
	log.Printf("[BehaviorSystem] 僵尸 %d 行为切换为 BehaviorZombieDyingExplosion", entityID)

	// 2. 移除 VelocityComponent（停止移动）
	ecs.RemoveComponent[*components.VelocityComponent](s.entityManager, entityID)
	log.Printf("[BehaviorSystem] 僵尸 %d 移除速度组件，停止移动", entityID)

	// 3. 使用 AnimationCommand 触发烧焦死亡动画
	//    Story 5.4.1: ReanimSystem.PlayCombo 现在支持单位切换
	//    当 UnitID 与当前 ReanimName 不同时，自动重新加载 Reanim 数据
	ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
		UnitID:    types.UnitIDZombieCharred, // 指向 zombie_charred 配置
		ComboName: "death",                   // 配置中的 death 组合
		Processed: false,
	})
	log.Printf("[BehaviorSystem] 僵尸 %d 添加烧焦死亡动画命令 (zombie_charred/death)", entityID)

	log.Printf("[BehaviorSystem] 僵尸 %d 烧焦死亡动画已开始播放 (zombie_charred/death, 不循环)", entityID)
}

// handleZombieDyingExplosionBehavior 处理僵尸爆炸烧焦死亡动画播放
//
// 当僵尸被爆炸类攻击杀死时，播放专用的烧焦黑化动画
// 动画播放完成后删除僵尸实体并增加消灭计数
//
// 参数:
//   - entityID: 僵尸实体ID
//
// 技术说明:
//   - 烧焦动画为非循环动画，ReanimSystem 会自动推进帧
//   - 当 reanim.IsFinished = true 时，动画完成
//   - 必须在删除实体前增加计数，否则计数丢失
//   - 参考实现: handleZombieDyingBehavior() (普通死亡)
func (s *BehaviorSystem) handleZombieDyingExplosionBehavior(entityID ecs.EntityID) {
	// 获取 ReanimComponent
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		// 如果没有 ReanimComponent，直接删除僵尸
		log.Printf("[BehaviorSystem] 爆炸死亡中的僵尸 %d 缺少 ReanimComponent，直接删除", entityID)
		s.gameState.IncrementZombiesKilled()
		s.entityManager.DestroyEntity(entityID)
		return
	}

	// 检查烧焦死亡动画是否完成
	if reanim.IsFinished {
		log.Printf("[BehaviorSystem] 僵尸 %d 烧焦死亡动画完成，删除实体", entityID)

		// 增加僵尸消灭计数
		s.gameState.IncrementZombiesKilled()

		// 删除僵尸实体
		s.entityManager.DestroyEntity(entityID)
	}
}

// updateArmorVisualState 更新护甲僵尸的外观状态
// 根据护甲的受损程度（剩余百分比）切换不同的护甲图片
// 支持路障僵尸（cone）和铁桶僵尸（bucket）
func (s *BehaviorSystem) updateArmorVisualState(entityID ecs.EntityID, armor *components.ArmorComponent, armorType string) {
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok || reanim.PartImages == nil {
		return
	}

	var targetImageName string
	var imageKey string
	maxArmor := float64(armor.MaxArmor)
	currentArmor := float64(armor.CurrentArmor)
	ratio := currentArmor / maxArmor

	if armorType == "cone" {
		imageKey = "IMAGE_REANIM_ZOMBIE_CONE1"
		// 阶段1: 完整 (66% - 100%)
		// 阶段2: 轻微受损 (33% - 66%)
		// 阶段3: 严重受损 (0% - 33%)
		if ratio > 0.66 {
			targetImageName = "assets/reanim/Zombie_cone1.png"
		} else if ratio > 0.33 {
			targetImageName = "assets/reanim/Zombie_cone2.png"
		} else {
			targetImageName = "assets/reanim/Zombie_cone3.png"
		}
	} else if armorType == "bucket" {
		imageKey = "IMAGE_REANIM_ZOMBIE_BUCKET1"
		if ratio > 0.66 {
			targetImageName = "assets/reanim/Zombie_bucket1.png"
		} else if ratio > 0.33 {
			targetImageName = "assets/reanim/Zombie_bucket2.png"
		} else {
			targetImageName = "assets/reanim/Zombie_bucket3.png"
		}
	} else {
		return
	}

	// 加载目标图片
	targetImage, err := s.resourceManager.LoadImage(targetImageName)
	if err != nil {
		// 降低日志频率，避免每帧刷屏
		if s.logFrameCounter%100 == 0 {
			log.Printf("[BehaviorSystem] 警告：无法加载受损护甲图片 %s: %v", targetImageName, err)
		}
		return
	}

	// 检查当前显示的图片是否已经是目标图片
	if reanim.PartImages[imageKey] != targetImage {
		// 确保 PartImages 是独立的副本
		// 我们无法简单判断是否已经是独立副本，所以如果需要修改，就总是创建一个新的 map
		// 这是一个浅拷贝，开销很小
		newPartImages := make(map[string]*ebiten.Image)
		for k, v := range reanim.PartImages {
			newPartImages[k] = v
		}
		// 更新目标图片的映射
		newPartImages[imageKey] = targetImage
		// 替换组件中的 map
		reanim.PartImages = newPartImages

		log.Printf("[BehaviorSystem] 僵尸 %d 护甲外观更新: %s -> %s (HP ratio: %.2f)", entityID, imageKey, targetImageName, ratio)
	}
}

// handleArmorDestroyedWhileEating 处理僵尸在啃食状态下护甲被打掉的情况
// 这个函数确保护甲被破坏时，即使僵尸正在啃食，也能正确隐藏护甲轨道并更新 UnitID
// 防止恢复移动时使用错误的动画配置（如 zombie_conehead）导致护甲重新显示
//
// 参数:
//   - entityID: 僵尸实体ID
//   - behavior: 僵尸的行为组件（已获取，避免重复查询）
func (s *BehaviorSystem) handleArmorDestroyedWhileEating(entityID ecs.EntityID, behavior *components.BehaviorComponent) {
	// 检查 UnitID 判断是哪种护甲僵尸
	var armorTrackName string
	var particleEffectName string

	switch behavior.UnitID {
	case "zombie_conehead":
		armorTrackName = "anim_cone"
		particleEffectName = "ZombieTrafficCone"
	case "zombie_buckethead":
		armorTrackName = "anim_bucket"
		particleEffectName = "ZombiePail"
	default:
		// 不是护甲僵尸，不需要处理
		return
	}

	// 检查是否已经处理过（通过检查轨道是否已隐藏）
	reanim, hasReanim := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if hasReanim {
		if reanim.HiddenTracks != nil && reanim.HiddenTracks[armorTrackName] {
			// 轨道已隐藏，不需要重复处理
			return
		}

		// 隐藏护甲轨道
		if reanim.HiddenTracks == nil {
			reanim.HiddenTracks = make(map[string]bool)
		}
		reanim.HiddenTracks[armorTrackName] = true
		log.Printf("[BehaviorSystem] 啃食中的僵尸 %d 护甲破坏，隐藏 %s 轨道", entityID, armorTrackName)
	}

	// 更新 UnitID 为普通僵尸，这样恢复移动时会使用正确的动画配置
	oldUnitID := behavior.UnitID
	behavior.UnitID = "zombie"
	log.Printf("[BehaviorSystem] 啃食中的僵尸 %d UnitID 更新: %s -> zombie", entityID, oldUnitID)

	// 触发护甲掉落粒子效果
	position, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if hasPos {
		// 啃食状态没有 VelocityComponent，默认角度偏移 180°（僵尸向左走）
		angleOffset := 180.0

		_, err := entities.CreateParticleEffect(
			s.entityManager,
			s.resourceManager,
			particleEffectName,
			position.X, position.Y,
			angleOffset,
		)
		if err != nil {
			log.Printf("[BehaviorSystem] 警告：创建护甲掉落粒子失败: %v", err)
		} else {
			log.Printf("[BehaviorSystem] 啃食中的僵尸 %d 触发护甲掉落效果 (%s)", entityID, particleEffectName)
		}
	}
}

// handleCherryBombBehavior 处理樱桃炸弹的行为逻辑
// 樱桃炸弹种植后开始引信倒计时（1.5秒），倒计时结束后触发爆炸
