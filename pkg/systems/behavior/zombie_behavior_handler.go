package behavior

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
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
			log.Printf("[BehaviorSystem] 僵尸 %d 生命值 <= 0 (HP=%d)，触发死亡", entityID, health.CurrentHealth)
			s.triggerZombieDeath(entityID)
			return // 跳过正常移动逻辑
		}
	}

	// 获取位置组件
	position, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 检测植物碰撞（在移动之前）
	// 计算僵尸所在格子
	// 注意：需要减去 ZombieVerticalOffset，因为僵尸Y坐标包含了偏移
	zombieCol := int((position.X - config.GridWorldStartX) / config.CellWidth)
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

	// DEBUG: 记录僵尸速度
	log.Printf("[BehaviorSystem] Zombie %d moving: X=%.1f, VX=%.2f, VY=%.2f",
		entityID, position.X, velocity.VX, velocity.VY)

	// 更新位置：根据速度和时间增量移动僵尸
	position.X += velocity.VX * deltaTime
	position.Y += velocity.VY * deltaTime

	// 边界检查：如果僵尸移出屏幕左侧，标记删除
	// 使用 config.ZombieDeletionBoundary 提供容错空间，避免僵尸刚移出就被删除
	if position.X < config.ZombieDeletionBoundary {
		log.Printf("[BehaviorSystem] 僵尸 %d 移出屏幕左侧 (X=%.1f)，标记删除", entityID, position.X)
		s.entityManager.DestroyEntity(entityID)
	}
}

// triggerZombieDeath 触发僵尸死亡状态转换
// 当僵尸生命值 <= 0 时调用，将僵尸从正常行为状态切换到死亡动画播放状态
// 添加僵尸死亡粒子效果触发（头部掉落）
// 注意：手臂掉落粒子效果在 updateZombieDamageState 中触发（受伤时）

func (s *BehaviorSystem) triggerZombieDeath(entityID ecs.EntityID) {
	// 1. 切换行为类型为 BehaviorZombieDying
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[BehaviorSystem] 僵尸 %d 缺少 BehaviorComponent，无法触发死亡", entityID)
		return
	}
	behavior.Type = components.BehaviorZombieDying
	log.Printf("[BehaviorSystem] 僵尸 %d 行为切换为 BehaviorZombieDying", entityID)

	// 获取僵尸位置，用于触发粒子效果
	position, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[BehaviorSystem] 警告：僵尸 %d 缺少 PositionComponent，无法触发粒子效果", entityID)
	} else {
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
	}

	// 2. 隐藏头部轨道（头掉落效果）
	// 直接修改 HiddenTracks 字段而不调用废弃的 HideTrack API
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

	// 3. 移除 VelocityComponent（停止移动）
	ecs.RemoveComponent[*components.VelocityComponent](s.entityManager, entityID)
	log.Printf("[BehaviorSystem] 僵尸 %d 移除速度组件，停止移动", entityID)

	// 4. 使用 AnimationCommand 组件播放死亡动画（不循环）
	// 使用组件通信替代直接调用
	// 使用配置驱动的动画组合（自动隐藏装备轨道）
	ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
		UnitID:    "zombie",
		ComboName: "death",
		Processed: false,
	})
	log.Printf("[BehaviorSystem] 僵尸 %d 添加死亡动画命令", entityID)

	// 设置为不循环
	if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
		reanim.IsLooping = false
	}

	log.Printf("[BehaviorSystem] 僵尸 %d 死亡动画已开始播放 (anim_death, 不循环)", entityID)
}

// handlePeashooterBehavior 处理豌豆射手的行为逻辑
// 豌豆射手会周期性扫描同行僵尸并发射豌豆子弹

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

// playShootSound 播放豌豆射手发射子弹的音效
// 使用配置文件中定义的音效（config.PeashooterShootSoundPath）
// 如果配置为空字符串，则不播放音效（静音模式）

func (s *BehaviorSystem) detectPlantCollision(zombieRow, zombieCol int) (ecs.EntityID, bool) {
	// 查询所有植物实体（拥有 PlantComponent）
	plantEntityList := ecs.GetEntitiesWith1[*components.PlantComponent](s.entityManager)

	// 遍历所有植物，比对网格位置
	for _, plantID := range plantEntityList {
		plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, plantID)
		if !ok {
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

	// 根据状态确定组合名称
	// 使用配置驱动的动画播放
	var comboName string
	switch newState {
	case components.ZombieAnimIdle:
		comboName = "idle"
	case components.ZombieAnimWalking:
		comboName = "walk"
	case components.ZombieAnimEating:
		comboName = "eat"
	case components.ZombieAnimDying:
		// 根据僵尸类型使用不同的 unitID
		var unitID string
		switch behavior.Type {
		case components.BehaviorZombieConehead:
			unitID = "zombie_conehead"
		case components.BehaviorZombieBuckethead:
			unitID = "zombie_buckethead"
		default:
			unitID = "zombie"
		}

		// 使用组件通信替代直接调用
		ecs.AddComponent(s.entityManager, zombieID, &components.AnimationCommandComponent{
			UnitID:    unitID,
			ComboName: "death",
			Processed: false,
		})
		log.Printf("[BehaviorSystem] 僵尸 %d (%s) 添加死亡动画命令", zombieID, unitID)
		return
	default:
		return
	}

	// 根据僵尸类型选择正确的 unitID
	var unitID string
	switch behavior.Type {
	case components.BehaviorZombieConehead:
		unitID = "zombie_conehead"
	case components.BehaviorZombieBuckethead:
		unitID = "zombie_buckethead"
	default:
		unitID = "zombie"
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

	// 2. 在切换类型之前，先记住原始僵尸类型（用于选择正确的啃食动画）
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
	if !ok {
		return
	}
	originalZombieType := behavior.Type // 记住原始类型

	// 3. 切换 BehaviorComponent.Type 为 BehaviorZombieEating
	behavior.Type = components.BehaviorZombieEating

	// 切换僵尸动画为啃食状态
	s.changeZombieAnimation(zombieID, components.ZombieAnimEating)

	// 4. 添加 TimerComponent 用于伤害间隔
	ecs.AddComponent(s.entityManager, zombieID, &components.TimerComponent{
		Name:        "eating_damage",
		TargetTime:  config.ZombieEatingDamageInterval,
		CurrentTime: 0,
		IsReady:     false,
	})

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

	// 1. 移除 TimerComponent
	ecs.RemoveComponent[*components.TimerComponent](s.entityManager, zombieID)

	// 2. 切换 BehaviorComponent.Type 回 BehaviorZombieBasic
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
	if ok {
		behavior.Type = components.BehaviorZombieBasic
	}

	// 切换僵尸动画回行走状态
	s.changeZombieAnimation(zombieID, components.ZombieAnimWalking)

	// 3. 恢复 VelocityComponent
	ecs.AddComponent(s.entityManager, zombieID, &components.VelocityComponent{
		VX: config.ZombieWalkSpeed,
		VY: 0,
	})

	// 待迁移到 ReanimComponent
	// 4. 加载僵尸走路动画帧序列
	// walkFrames := utils.LoadZombieWalkAnimation(s.resourceManager)

	// 待迁移到 ReanimComponent
	// 5. 替换 AnimationComponent 为走路动画
	// animComp, ok := s.entityManager.GetComponent(zombieID, reflect.TypeOf(&components.AnimationComponent{}))
	// if ok {
	// 	anim := animComp.(*components.AnimationComponent)
	// 	anim.Frames = walkFrames
	// 	anim.FrameSpeed = config.ZombieWalkFrameSpeed
	// 	anim.CurrentFrame = 0
	// 	anim.FrameCounter = 0
	// 	anim.IsLooping = true
	// 	anim.IsFinished = false
	// }
}

// handleZombieEatingBehavior 处理僵尸啃食植物的行为
// 参数:
//   - entityID: 僵尸实体ID
//   - deltaTime: 帧间隔时间

func (s *BehaviorSystem) handleZombieEatingBehavior(entityID ecs.EntityID, deltaTime float64) {
	// DEBUG: 添加日志确认函数被调用
	log.Printf("[BehaviorSystem] 🍴 处理僵尸 %d 啃食行为", entityID)

	// 检查生命值并更新受伤状态（掉手臂、掉头）
	health, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, entityID)
	if ok {

		// 更新僵尸的受伤状态（掉手臂）
		s.updateZombieDamageState(entityID, health)

		// 检查生命值是否归零（即使在啃食状态也要检查）
		if health.CurrentHealth <= 0 {
			log.Printf("[BehaviorSystem] 啃食中的僵尸 %d 生命值 <= 0 (HP=%d)，触发死亡", entityID, health.CurrentHealth)
			s.triggerZombieDeath(entityID)
			return
		}
	}

	// 检查护甲状态（护甲僵尸即使在啃食也需要检测护甲破坏）
	armor, hasArmor := ecs.GetComponent[*components.ArmorComponent](s.entityManager, entityID)
	if hasArmor {
		// 待迁移到 ReanimComponent
		// 如果护甲已破坏，切换为普通僵尸动画
		// if armor.CurrentArmor <= 0 {
		// 	// 加载普通僵尸啃食动画
		// 	normalEatFrames := utils.LoadZombieEatAnimation(s.resourceManager)
		// 	animComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.AnimationComponent{}))
		// 	if ok {
		// 		anim := animComp.(*components.AnimationComponent)
		// 		// 检查是否已经是普通僵尸动画(避免重复切换)
		// 		if len(anim.Frames) != config.ZombieEatAnimationFrames {
		// 			anim.Frames = normalEatFrames
		// 			anim.CurrentFrame = 0
		// 			anim.FrameCounter = 0
		// 			log.Printf("[BehaviorSystem] 啃食中的护甲僵尸 %d 护甲耗尽，切换为普通僵尸啃食动画", entityID)
		// 		}
		// 	}
		// }
		_ = armor // 临时避免未使用警告
	}

	// 获取僵尸的 TimerComponent
	timer, ok := ecs.GetComponent[*components.TimerComponent](s.entityManager, entityID)
	if !ok {
		// 没有计时器，恢复移动
		s.stopEatingAndResume(entityID)
		return
	}

	// 更新计时器
	timer.CurrentTime += deltaTime

	// 检查计时器是否完成
	if timer.CurrentTime >= timer.TargetTime {
		timer.IsReady = true
	}

	// 如果计时器完成，造成伤害
	if timer.IsReady {
		// 获取僵尸当前网格位置
		pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		if !ok {
			return
		}

		// 计算僵尸所在格子
		// 注意：需要减去 ZombieVerticalOffset，因为僵尸Y坐标包含了偏移
		zombieCol := int((pos.X - config.GridWorldStartX) / config.CellWidth)
		zombieRow := int((pos.Y - config.GridWorldStartY - config.ZombieVerticalOffset - config.CellHeight/2.0) / config.CellHeight)

		// 检测植物
		plantID, hasPlant := s.detectPlantCollision(zombieRow, zombieCol)

		if hasPlant {
			// 植物存在，造成伤害
			plantHealth, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, plantID)
			if ok {
				plantHealth.CurrentHealth -= config.ZombieEatingDamage

				log.Printf("[BehaviorSystem] 僵尸 %d 啃食植物 %d，造成 %d 伤害，剩余生命值 %d",
					entityID, plantID, config.ZombieEatingDamage, plantHealth.CurrentHealth)

				// 播放啃食音效
				s.playEatingSound()

				// 检查植物是否死亡
				if plantHealth.CurrentHealth <= 0 {
					log.Printf("[BehaviorSystem] 植物 %d 被吃掉，删除实体", plantID)

					// 释放网格占用状态，允许重新种植
					if plantComp, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, plantID); ok {
						err := s.lawnGridSystem.ReleaseCell(s.lawnGridEntityID, plantComp.GridCol, plantComp.GridRow)
						if err != nil {
							log.Printf("[BehaviorSystem] 警告：释放网格占用失败: %v", err)
						} else {
							log.Printf("[BehaviorSystem] 网格 (%d, %d) 已释放", plantComp.GridCol, plantComp.GridRow)
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
				s.entityManager.DestroyEntity(plantID)
				s.stopEatingAndResume(entityID)
				return
			}
		} else {
			// 植物不存在（可能被其他僵尸吃掉），恢复移动
			s.stopEatingAndResume(entityID)
			return
		}

		// 重置计时器
		timer.CurrentTime = 0
		timer.IsReady = false
	}
}

// playEatingSound 播放僵尸啃食音效

func (s *BehaviorSystem) playEatingSound() {
	// 加载啃食音效
	eatingSound, err := s.resourceManager.LoadSoundEffect(config.ZombieEatingSoundPath)
	if err != nil {
		// 音效加载失败时不阻止游戏继续运行
		return
	}

	// 重置播放器位置到开头
	eatingSound.Rewind()

	// 播放音效
	eatingSound.Play()
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

				// 2. 隐藏路障轨道（使用 HiddenTracks 黑名单）
				reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
				if ok {
					if reanim.HiddenTracks == nil {
						reanim.HiddenTracks = make(map[string]bool)
					}
					reanim.HiddenTracks["anim_cone"] = true // 隐藏路障
					log.Printf("[BehaviorSystem] 路障僵尸 %d 隐藏 anim_cone 轨道", entityID)
				}

				// 3. 移除护甲组件（可选，但保留可能对调试有帮助）
				// ecs.RemoveComponent[*components.ArmorComponent](s.entityManager, entityID)
			}
		}

		// 护甲已破坏，继续以普通僵尸行为运作
		s.handleZombieBasicBehavior(entityID, deltaTime)
		return
	}

	// 护甲完好，执行普通僵尸的基本行为（移动、碰撞检测、啃食植物）
	s.handleZombieBasicBehavior(entityID, deltaTime)
}

// handleBucketheadZombieBehavior 处理铁桶僵尸的行为逻辑
// 铁桶僵尸拥有更高的护甲层，护甲耗尽后切换为普通僵尸外观和行为

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

				// 2. 隐藏铁桶轨道（使用 HiddenTracks 黑名单）
				reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
				if ok {
					if reanim.HiddenTracks == nil {
						reanim.HiddenTracks = make(map[string]bool)
					}
					reanim.HiddenTracks["anim_bucket"] = true // 隐藏铁桶
					log.Printf("[BehaviorSystem] 铁桶僵尸 %d 隐藏 anim_bucket 轨道", entityID)
				}

				// 3. 移除护甲组件（可选，但保留可能对调试有帮助）
				// ecs.RemoveComponent[*components.ArmorComponent](s.entityManager, entityID)
			}
		}

		// 护甲已破坏，继续以普通僵尸行为运作
		s.handleZombieBasicBehavior(entityID, deltaTime)
		return
	}

	// 护甲完好，执行普通僵尸的基本行为（移动、碰撞检测、啃食植物）
	s.handleZombieBasicBehavior(entityID, deltaTime)
}

// handleCherryBombBehavior 处理樱桃炸弹的行为逻辑
// 樱桃炸弹种植后开始引信倒计时（1.5秒），倒计时结束后触发爆炸
