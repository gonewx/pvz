package behavior

import (
	"log"
	"math"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
)

func (s *BehaviorSystem) handleCherryBombBehavior(entityID ecs.EntityID, deltaTime float64) {
	// 获取计时器组件
	timer, ok := ecs.GetComponent[*components.TimerComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 检查引信计时器状态
	if !timer.IsReady {
		// 继续计时
		timer.CurrentTime += deltaTime
		if timer.CurrentTime >= timer.TargetTime {
			timer.IsReady = true
			log.Printf("[BehaviorSystem] 樱桃炸弹 %d: 引信计时完成，准备爆炸", entityID)
		}
		return
	}

	// 计时器已完成，触发爆炸
	s.triggerCherryBombExplosion(entityID)
}

func (s *BehaviorSystem) triggerCherryBombExplosion(entityID ecs.EntityID) {
	log.Printf("[BehaviorSystem] 樱桃炸弹 %d: 开始爆炸！", entityID)

	// 获取樱桃炸弹的世界坐标位置
	position, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[BehaviorSystem] 警告：樱桃炸弹 %d 缺少 PositionComponent，无法确定爆炸位置", entityID)
		return
	}

	// 获取樱桃炸弹的网格位置用于调试
	plantComp, _ := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID)
	if plantComp != nil {
		log.Printf("[BehaviorSystem] 樱桃炸弹 %d 网格位置: col=%d, row=%d", entityID, plantComp.GridCol, plantComp.GridRow)
	}

	// 计算爆炸圆心：植物位置 + 偏移量
	// 修正：PositionComponent 已经是网格中心，偏移量已在配置中归零
	explosionCenterX := position.X + config.CherryBombExplosionCenterOffsetX
	explosionCenterY := position.Y + config.CherryBombExplosionCenterOffsetY
	explosionRadius := config.CherryBombExplosionRadius
	explosionRadiusSq := explosionRadius * explosionRadius // 预计算半径平方，避免开方运算

	log.Printf("[BehaviorSystem] 樱桃炸弹爆炸范围 (圆形): 圆心(%.1f, %.1f), 半径%.1f, position.Y=%.1f",
		explosionCenterX, explosionCenterY, explosionRadius, position.Y)

	// 查询所有僵尸实体（移动中和啃食中的僵尸）
	allZombies := ecs.GetEntitiesWith2[*components.BehaviorComponent, *components.PositionComponent](s.entityManager)

	// 统计受影响的僵尸数量
	affectedZombies := 0

	// 对每个僵尸检查是否在爆炸范围内
	for _, zombieID := range allZombies {
		// 获取僵尸的行为组件，确认是僵尸类型
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
		if !ok {
			continue
		}

		// 只处理僵尸类型的实体
		if behavior.Type != components.BehaviorZombieBasic &&
			behavior.Type != components.BehaviorZombieEating &&
			behavior.Type != components.BehaviorZombieConehead &&
			behavior.Type != components.BehaviorZombieBuckethead &&
			behavior.Type != components.BehaviorZombieFlag &&
			behavior.Type != components.BehaviorZombieDying {
			continue
		}

		// 获取僵尸的位置组件
		zombiePos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
		if !ok {
			continue
		}

		// 计算爆炸圆心到僵尸碰撞盒的最近距离
		// 僵尸碰撞盒：以 zombiePos 为中心，宽 ZombieCollisionWidth，高 ZombieCollisionHeight
		// 僵尸的 PositionComponent.Y 已包含 ZombieVerticalOffset，需要还原到格子中心进行计算
		zombieCenterY := zombiePos.Y - config.ZombieVerticalOffset

		// 计算僵尸碰撞盒边界
		zombieLeft := zombiePos.X - config.ZombieCollisionWidth/2
		zombieRight := zombiePos.X + config.ZombieCollisionWidth/2
		zombieTop := zombieCenterY - config.ZombieCollisionHeight/2
		zombieBottom := zombieCenterY + config.ZombieCollisionHeight/2

		// 计算爆炸圆心到碰撞盒的最近点
		// 如果圆心在盒子内部或边缘，最近距离为0
		closestX := math.Max(zombieLeft, math.Min(explosionCenterX, zombieRight))
		closestY := math.Max(zombieTop, math.Min(explosionCenterY, zombieBottom))

		dx := closestX - explosionCenterX
		dy := closestY - explosionCenterY
		distanceSq := dx*dx + dy*dy

		// 计算僵尸所在行（用于调试）
		zombieRow := int((zombiePos.Y - config.GridWorldStartY - config.ZombieVerticalOffset) / config.CellHeight)
		log.Printf("[BehaviorSystem] 检测僵尸 %d: pos=(%.1f, %.1f), 碰撞盒中心Y=%.1f, row=%d, 到碰撞盒距离=%.1f, 半径=%.1f",
			zombieID, zombiePos.X, zombiePos.Y, zombieCenterY, zombieRow, math.Sqrt(distanceSq), explosionRadius)

		// 如果距离平方 <= 半径平方，则在爆炸范围内
		if distanceSq <= explosionRadiusSq {
			affectedZombies++
			log.Printf("[BehaviorSystem] 僵尸 %d 在爆炸范围内（世界坐标: %.1f, %.1f），应用伤害", zombieID, zombiePos.X, zombiePos.Y)

			// 应用伤害：先扣护甲，护甲不足或无护甲则扣生命值
			damage := config.CherryBombDamage

			// 检查是否有护甲组件
			armor, hasArmor := ecs.GetComponent[*components.ArmorComponent](s.entityManager, zombieID)
			if hasArmor {
				if armor.CurrentArmor > 0 {
					// 护甲优先扣除
					armorDamage := damage
					if armorDamage > armor.CurrentArmor {
						armorDamage = armor.CurrentArmor
					}
					armor.CurrentArmor -= armorDamage
					damage -= armorDamage
					log.Printf("[BehaviorSystem] 僵尸 %d 护甲受损：-%d，剩余护甲：%d，剩余伤害：%d",
						zombieID, armorDamage, armor.CurrentArmor, damage)
				}
			}

			// 如果还有剩余伤害，扣除生命值
			if damage > 0 {
				health, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, zombieID)
				if ok {
					originalHealth := health.CurrentHealth
					health.CurrentHealth -= damage
					if health.CurrentHealth < 0 {
						health.CurrentHealth = 0
					}
					log.Printf("[BehaviorSystem] 僵尸 %d 生命值受损：%d -> %d（伤害：%d）",
						zombieID, originalHealth, health.CurrentHealth, damage)

					// Story 5.4.1: 如果僵尸被爆炸杀死（生命值归零），立即触发烧焦死亡动画
					if health.CurrentHealth <= 0 {
						log.Printf("[CherryBomb] 僵尸 %d 被爆炸杀死，触发烧焦死亡", zombieID)
						s.triggerZombieExplosionDeath(zombieID)
					}
				}
			}
		}
	}

	log.Printf("[BehaviorSystem] 樱桃炸弹爆炸影响了 %d 个僵尸", affectedZombies)

	// 播放爆炸音效（使用 AudioManager 统一管理 - Story 10.9）
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_CHERRYBOMB")
		log.Printf("[BehaviorSystem] 播放樱桃炸弹爆炸音效")
	}

	// 创建爆炸粒子效果
	// 触发爆炸粒子效果（使用已获取的position组件）
	_, err := entities.CreateParticleEffect(
		s.entityManager,
		s.resourceManager,
		config.ExplosiveNutParticleEffect, // 使用与爆炸坚果相同的 Powie 粒子效果
		position.X, position.Y,
	)
	if err != nil {
		log.Printf("[BehaviorSystem] 警告：创建樱桃炸弹爆炸粒子效果失败: %v", err)
		// 不阻塞游戏逻辑，游戏继续运行
	} else {
		log.Printf("[BehaviorSystem] 樱桃炸弹 %d 触发爆炸粒子效果，位置: (%.1f, %.1f)", entityID, position.X, position.Y)
	}

	// 释放樱桃炸弹占用的网格，允许重新种植
	if plantComp, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID); ok {
		if s.lawnGridSystem != nil && s.lawnGridEntityID != 0 {
			err := s.lawnGridSystem.ReleaseCell(s.lawnGridEntityID, plantComp.GridCol, plantComp.GridRow)
			if err != nil {
				log.Printf("[BehaviorSystem] 警告：释放樱桃炸弹网格占用失败: %v", err)
			} else {
				log.Printf("[BehaviorSystem] 樱桃炸弹网格 (%d, %d) 已释放", plantComp.GridCol, plantComp.GridRow)
			}
		} else {
			log.Printf("[BehaviorSystem] 警告：无法释放网格，lawnGridSystem=%v, lawnGridEntityID=%d",
				s.lawnGridSystem != nil, s.lawnGridEntityID)
		}
	}

	// 删除樱桃炸弹实体
	s.entityManager.DestroyEntity(entityID)
	log.Printf("[BehaviorSystem] 樱桃炸弹 %d 已删除", entityID)
}

// handlePotatoMineBehavior 处理土豆地雷行为
// Story 8.9: 土豆地雷需要武装时间，武装完成后等待僵尸触发爆炸
//
// 行为阶段（使用 PotatoMinePhase 状态枚举）:
//  1. PotatoMineArming: 播放 anim_idle（埋在地下），计时等待武装完成
//  2. PotatoMineRising: 播放 anim_rise，从地下升起
//  3. PotatoMineArmed: 播放 idle combo（anim_armed + anim_light + anim_glow），等待僵尸触发
//     - 警告灯闪烁频率随僵尸距离动态变化，越近越快
//     - 使用 HiddenTracks 控制 anim_glow 的显隐实现闪烁
//  4. PotatoMineExploding: 播放 anim_mashed，造成范围伤害后删除
func (s *BehaviorSystem) handlePotatoMineBehavior(entityID ecs.EntityID, deltaTime float64) {
	// 获取植物组件
	plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 获取 Reanim 组件（后续阶段需要）
	reanim, hasReanim := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)

	// 根据阶段状态处理
	switch plant.PotatoMinePhase {
	case components.PotatoMineArming:
		// 阶段 1: 武装阶段
		plant.ArmingTimer += deltaTime
		if plant.ArmingTimer >= config.PotatoMineArmingTime {
			// 武装完成，切换到升起阶段
			plant.PotatoMinePhase = components.PotatoMineRising
			plant.IsArmed = true // 兼容旧字段
			log.Printf("[BehaviorSystem] 土豆地雷 %d: 武装完成！切换到升起阶段", entityID)

			// 播放升起动画（anim_rise，非循环动画）
			ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
				UnitID:        "potatomine",
				AnimationName: "anim_rise",
				Processed:     false,
			})
		}

	case components.PotatoMineRising:
		// 阶段 2: 升起阶段 - 等待 anim_rise 动画播放完成
		if hasReanim && reanim.IsFinished {
			// 升起完成，切换到待机阶段
			plant.PotatoMinePhase = components.PotatoMineArmed
			log.Printf("[BehaviorSystem] 土豆地雷 %d: 升起完成！切换到待机阶段", entityID)

			// 播放 idle combo（只包含 anim_armed，呼吸动画）
			ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
				UnitID:    "potatomine",
				ComboName: "idle",
				Processed: false,
			})

			// 初始化闪烁状态
			plant.WarningLightOn = false
			plant.WarningLightTimer = config.PotatoMineBlinkOffMax
			plant.WarningLightInterval = config.PotatoMineBlinkOffMax
			plant.WarningLightInitialized = false // 等待 PlayCombo 完成后再初始化
		}

	case components.PotatoMineArmed:
		// 阶段 3: 待机阶段 - 警告灯闪烁 + 检测僵尸触发
		s.handlePotatoMineArmedPhase(entityID, plant, reanim, hasReanim, deltaTime)

	case components.PotatoMineExploding:
		// 阶段 4: 爆炸阶段 - 不做任何处理，等待动画播放完成后由其他系统删除
		return
	}
}

// handlePotatoMineArmedPhase 处理土豆地雷待机阶段
// 包括警告灯闪烁和僵尸触发检测
func (s *BehaviorSystem) handlePotatoMineArmedPhase(entityID ecs.EntityID, plant *components.PlantComponent, reanim *components.ReanimComponent, hasReanim bool, deltaTime float64) {
	// DEBUG: 确认 Armed 阶段被调用
	log.Printf("[PotatoMine] handleArmedPhase: entityID=%d, initialized=%v, hasReanim=%v",
		entityID, plant.WarningLightInitialized, hasReanim)

	// 确保 anim_glow 轨道被正确设置（在 PlayCombo 完成后执行一次）
	// 重要：必须等待 AnimationCommandComponent 处理完成后再初始化
	// 否则 PlayCombo 会覆盖我们设置的 CurrentAnimations 和 AnimVisiblesMap
	if hasReanim && !plant.WarningLightInitialized {
		// 检查是否有待处理的 AnimationCommandComponent
		animCmd, hasAnimCmd := ecs.GetComponent[*components.AnimationCommandComponent](s.entityManager, entityID)
		if hasAnimCmd && !animCmd.Processed {
			// AnimationCommandComponent 尚未处理，等待下一帧（但不跳过僵尸检测）
			log.Printf("[BehaviorSystem] 土豆地雷: 等待 AnimationCommandComponent 处理完成...")
			// 不 return，继续执行僵尸检测（但跳过红灯闪烁）
		} else {
			// AnimationCommandComponent 已处理（或不存在），可以初始化红灯轨道
			s.initPotatoMineWarningLight(reanim)
			plant.WarningLightInitialized = true
			log.Printf("[BehaviorSystem] 土豆地雷: 初始化红灯轨道完成")
		}
	}

	// 获取土豆地雷位置
	plantPos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 查询所有僵尸实体
	zombieEntities := ecs.GetEntitiesWith2[*components.BehaviorComponent, *components.PositionComponent](s.entityManager)

	// 计算同行僵尸中最近的距离
	plantRow := plant.GridRow
	nearestDistance := config.PotatoMineWarningDistanceMax + 100

	// 检测僵尸
	for _, zombieID := range zombieEntities {
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
		if !ok {
			continue
		}

		// 只检测活着的僵尸
		if !s.isZombieBehaviorType(behavior.Type) {
			continue
		}
		if behavior.Type == components.BehaviorZombieDying || behavior.Type == components.BehaviorZombieDyingExplosion {
			continue
		}

		zombiePos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
		if !ok {
			continue
		}

		// 计算僵尸所在行
		zombieRow := int((zombiePos.Y - config.GridWorldStartY - config.ZombieVerticalOffset) / config.CellHeight)
		if zombieRow != plantRow {
			continue
		}

		// 获取僵尸碰撞盒偏移
		collision, hasCollision := ecs.GetComponent[*components.CollisionComponent](s.entityManager, zombieID)
		collisionOffsetX := 0.0
		if hasCollision {
			collisionOffsetX = collision.OffsetX
		}

		zombieCenterX := zombiePos.X + collisionOffsetX
		distance := zombieCenterX - plantPos.X

		if distance > 0 && distance < nearestDistance {
			nearestDistance = distance
		}

		// 检测触发爆炸
		zombieCol := int((zombieCenterX - config.GridWorldStartX) / config.CellWidth)
		if zombieCol == plant.GridCol {
			log.Printf("[BehaviorSystem] 土豆地雷 %d: 检测到僵尸 %d 触发！", entityID, zombieID)
			s.triggerPotatoMineExplosion(entityID, plantPos.X, plantPos.Y)
			return
		}
	}

	// 更新警告灯闪烁（只有在初始化完成后才更新）
	if plant.WarningLightInitialized {
		// 调试日志：输出最近僵尸距离和闪烁间隔
		if nearestDistance < config.PotatoMineWarningDistanceMax+50 {
			log.Printf("[PotatoMine] 土豆地雷 %d: 最近僵尸距离=%.1f, 当前间隔=%.3f", entityID, nearestDistance, plant.WarningLightInterval)
		}
		s.updatePotatoMineWarningLight(entityID, plant, reanim, hasReanim, nearestDistance, deltaTime)
	}
}

// updatePotatoMineWarningLight 更新土豆地雷警告灯闪烁
// 使用 HiddenTracks 控制 anim_glow 轨道（红灯）的显隐
// 灰灯（anim_light）始终显示，红灯（anim_glow）叠加显示实现闪烁效果
// 红灯亮的时间固定，灭的时间根据僵尸距离变化
func (s *BehaviorSystem) updatePotatoMineWarningLight(entityID ecs.EntityID, plant *components.PlantComponent, reanim *components.ReanimComponent, hasReanim bool, nearestDistance float64, deltaTime float64) {
	// DEBUG: 确认函数被调用
	log.Printf("[PotatoMine] updateWarningLight: entityID=%d, timer=%.3f, deltaTime=%.3f, lightOn=%v",
		entityID, plant.WarningLightTimer, deltaTime, plant.WarningLightOn)

	// 计算红灯灭的时间（线性插值）
	var offDuration float64
	if nearestDistance <= config.PotatoMineWarningDistanceMin {
		offDuration = config.PotatoMineBlinkOffMin
	} else if nearestDistance >= config.PotatoMineWarningDistanceMax {
		offDuration = config.PotatoMineBlinkOffMax
	} else {
		t := (config.PotatoMineWarningDistanceMax - nearestDistance) /
			(config.PotatoMineWarningDistanceMax - config.PotatoMineWarningDistanceMin)
		offDuration = config.PotatoMineBlinkOffMax -
			t*(config.PotatoMineBlinkOffMax-config.PotatoMineBlinkOffMin)
	}

	plant.WarningLightTimer -= deltaTime

	if plant.WarningLightTimer <= 0 {
		// 切换灯的亮灭状态
		plant.WarningLightOn = !plant.WarningLightOn

		// 根据新状态设置下一次切换的时间
		if plant.WarningLightOn {
			// 刚切换到亮：下次切换时间 = 固定的亮时间
			plant.WarningLightTimer = config.PotatoMineBlinkOnDuration
			log.Printf("[PotatoMine] 🔴 红灯亮！下次切换时间=%.3f秒", plant.WarningLightTimer)
		} else {
			// 刚切换到灭：下次切换时间 = 根据距离计算的灭时间
			plant.WarningLightTimer = offDuration
			log.Printf("[PotatoMine] ⚪ 红灯灭！下次切换时间=%.3f秒（offDuration）", plant.WarningLightTimer)
		}
		plant.WarningLightInterval = offDuration // 用于调试日志

		if hasReanim {
			// 使用 HiddenTracks 只控制红灯（anim_glow）的显隐
			// 灰灯（anim_light）始终显示，红灯叠加在灰灯之上实现闪烁
			if reanim.HiddenTracks == nil {
				reanim.HiddenTracks = make(map[string]bool)
			}

			// 灰灯始终显示（确保不被隐藏）
			delete(reanim.HiddenTracks, "anim_light")

			if plant.WarningLightOn {
				// 红灯亮：显示 anim_glow（叠加在灰灯上）
				delete(reanim.HiddenTracks, "anim_glow")
				log.Printf("[PotatoMine] HiddenTracks: 显示红灯（叠加在灰灯上）")
			} else {
				// 红灯灭：隐藏 anim_glow，灰灯继续显示
				reanim.HiddenTracks["anim_glow"] = true
				log.Printf("[PotatoMine] HiddenTracks: 隐藏红灯，灰灯继续显示")
			}

			// 强制缓存失效
			reanim.LastRenderFrame = -1
		}
	}
}

// triggerPotatoMineExplosion 触发土豆地雷爆炸
// Story 8.9: 土豆地雷爆炸造成 1800 伤害，1x1 格范围
func (s *BehaviorSystem) triggerPotatoMineExplosion(entityID ecs.EntityID, explosionX, explosionY float64) {
	log.Printf("[BehaviorSystem] 土豆地雷 %d: 开始爆炸！位置: (%.1f, %.1f)", entityID, explosionX, explosionY)

	// 获取植物组件并设置为爆炸阶段
	plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID)
	if !ok {
		return
	}
	plant.PotatoMinePhase = components.PotatoMineExploding
	plant.IsExploding = true // 兼容旧字段

	// 播放爆炸动画（anim_mashed）
	ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
		UnitID:        "potatomine",
		AnimationName: "anim_mashed",
		Processed:     false,
	})

	// 计算爆炸范围（1x1 格）
	// 土豆地雷的爆炸范围比樱桃炸弹小，只影响同一格内的僵尸
	explosionRadius := config.CellWidth * config.PotatoMineExplosionRadius / 2
	explosionRadiusSq := explosionRadius * explosionRadius

	log.Printf("[BehaviorSystem] 土豆地雷爆炸范围: 圆心(%.1f, %.1f), 半径%.1f",
		explosionX, explosionY, explosionRadius)

	// 查询所有僵尸实体
	allZombies := ecs.GetEntitiesWith2[*components.BehaviorComponent, *components.PositionComponent](s.entityManager)

	// 统计受影响的僵尸数量
	affectedZombies := 0

	// 对每个僵尸检查是否在爆炸范围内
	for _, zombieID := range allZombies {
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
		if !ok {
			continue
		}

		// 只处理活着的僵尸
		if !s.isZombieBehaviorType(behavior.Type) {
			continue
		}
		if behavior.Type == components.BehaviorZombieDying || behavior.Type == components.BehaviorZombieDyingExplosion {
			continue
		}

		// 获取僵尸位置
		zombiePos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
		if !ok {
			continue
		}

		// 计算僵尸到爆炸中心的距离
		dx := zombiePos.X - explosionX
		dy := (zombiePos.Y - config.ZombieVerticalOffset) - explosionY
		distanceSq := dx*dx + dy*dy

		// 如果在爆炸范围内
		if distanceSq <= explosionRadiusSq {
			affectedZombies++
			log.Printf("[BehaviorSystem] 僵尸 %d 在土豆地雷爆炸范围内，应用 %d 伤害",
				zombieID, config.PotatoMineExplosionDamage)

			// 应用伤害
			damage := config.PotatoMineExplosionDamage

			// 检查是否有护甲组件
			armor, hasArmor := ecs.GetComponent[*components.ArmorComponent](s.entityManager, zombieID)
			if hasArmor && armor.CurrentArmor > 0 {
				armorDamage := damage
				if armorDamage > armor.CurrentArmor {
					armorDamage = armor.CurrentArmor
				}
				armor.CurrentArmor -= armorDamage
				damage -= armorDamage
				log.Printf("[BehaviorSystem] 僵尸 %d 护甲受损：-%d，剩余护甲：%d",
					zombieID, armorDamage, armor.CurrentArmor)
			}

			// 扣除生命值
			if damage > 0 {
				health, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, zombieID)
				if ok {
					health.CurrentHealth -= damage
					if health.CurrentHealth < 0 {
						health.CurrentHealth = 0
					}
					log.Printf("[BehaviorSystem] 僵尸 %d 生命值受损：剩余 %d",
						zombieID, health.CurrentHealth)

					// 如果僵尸被杀死，直接删除（不播放变焦动画）
					if health.CurrentHealth <= 0 {
						log.Printf("[PotatoMine] 僵尸 %d 被爆炸杀死，直接删除", zombieID)
						s.triggerZombieInstantDeath(zombieID, zombiePos.X, zombiePos.Y)
					}
				}
			}
		}
	}

	log.Printf("[BehaviorSystem] 土豆地雷爆炸影响了 %d 个僵尸", affectedZombies)

	// 播放爆炸音效
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_POTATO_MINE")
		log.Printf("[BehaviorSystem] 播放土豆地雷爆炸音效")
	}

	// 创建爆炸粒子效果
	_, err := entities.CreateParticleEffect(
		s.entityManager,
		s.resourceManager,
		"PotatoMine", // 使用土豆地雷专用粒子效果
		explosionX, explosionY,
	)
	if err != nil {
		log.Printf("[BehaviorSystem] 警告：创建土豆地雷爆炸粒子效果失败: %v", err)
	}

	// 释放土豆地雷占用的网格
	if s.lawnGridSystem != nil && s.lawnGridEntityID != 0 {
		err := s.lawnGridSystem.ReleaseCell(s.lawnGridEntityID, plant.GridCol, plant.GridRow)
		if err != nil {
			log.Printf("[BehaviorSystem] 警告：释放土豆地雷网格占用失败: %v", err)
		} else {
			log.Printf("[BehaviorSystem] 土豆地雷网格 (%d, %d) 已释放", plant.GridCol, plant.GridRow)
		}
	}

	// 设置延迟删除土豆地雷实体（等待爆炸动画播放完成）
	// anim_mashed 动画大约 0.5 秒
	ecs.AddComponent(s.entityManager, entityID, &components.LifetimeComponent{
		MaxLifetime:     0.5,
		CurrentLifetime: 0,
		IsExpired:       false,
	})
}

// initPotatoMineWarningLight 初始化土豆地雷的红灯闪烁
// 使用 HiddenTracks 控制 anim_glow（红灯）和 anim_light（灰灯）的互斥显示
// 注意：anim_glow 和 anim_light 已在 idle combo 配置中，此函数只需设置额外参数
func (s *BehaviorSystem) initPotatoMineWarningLight(reanim *components.ReanimComponent) {
	if reanim.TrackFrameOverrides == nil {
		reanim.TrackFrameOverrides = make(map[string]int)
	}

	// 1. 设置 TrackFrameOverrides 强制 anim_glow 使用 frame 20
	// anim_glow 轨道只有 frame 20 有图像数据，其他帧都是 f=-1 或空
	reanim.TrackFrameOverrides["anim_glow"] = 20

	// 2. 设置 TrackFrameOverrides 强制 anim_light 使用 frame 0
	// anim_light 轨道在 frame 0 有图像数据（IMAGE_REANIM_POTATOMINE_LIGHT1）
	// 注意：anim_light 在 frame 20-22 是空帧，frame 9-10 也是空帧
	// 如果让 anim_light 跟随 anim_armed 的帧索引（20-30），会在帧 20-22 时没有图像
	// 所以固定使用 frame 0，确保灰灯始终显示
	reanim.TrackFrameOverrides["anim_light"] = 0

	// 3. 设置 anim_glow 和 anim_light 的动画速度为 0，阻止帧索引被自动更新
	if reanim.AnimationSpeedOverrides == nil {
		reanim.AnimationSpeedOverrides = make(map[string]float64)
	}
	reanim.AnimationSpeedOverrides["anim_glow"] = 0
	reanim.AnimationSpeedOverrides["anim_light"] = 0

	// 4. 重置帧索引为 0（防止 PlayCombo 后已有更新）
	if reanim.AnimationFrameIndices == nil {
		reanim.AnimationFrameIndices = make(map[string]float64)
	}
	reanim.AnimationFrameIndices["anim_glow"] = 0
	reanim.AnimationFrameIndices["anim_light"] = 0

	// 5. 确保 HiddenTracks 已初始化（combo 配置会设置 anim_glow 为隐藏）
	if reanim.HiddenTracks == nil {
		reanim.HiddenTracks = make(map[string]bool)
	}

	// 6. 强制缓存失效
	reanim.LastRenderFrame = -1

	log.Printf("[BehaviorSystem] initPotatoMineWarningLight: 初始化完成，TrackFrameOverrides[anim_glow]=20, TrackFrameOverrides[anim_light]=0")
}
