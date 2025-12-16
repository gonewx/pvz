package behavior

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/utils"
)

// handlePeashooterBehavior 处理豌豆射手/寒冰射手的攻击行为
// 简化设计：攻击动画决定攻击节奏，不需要额外的冷却计时器
// - 有僵尸 → 播放攻击动画（循环）
// - 动画到达关键帧 → 发射子弹
// - 没有僵尸 → 切回 idle
func (s *BehaviorSystem) handlePeashooterBehavior(entityID ecs.EntityID, deltaTime float64, zombieEntityList []ecs.EntityID) {
	// 获取植物组件（用于状态管理）
	plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[BehaviorSystem] ⚠️ 豌豆射手 %d 缺少 PlantComponent", entityID)
		return
	}

	// 根据植物类型确定动画 UnitID
	unitID := "peashootersingle"
	if plant.PlantType == components.PlantSnowPea {
		unitID = "snowpea"
	}

	// 检测豌豆射手是否正在被啃食（检查同格子是否有啃食状态的僵尸）
	isBeingEaten := s.isPlantBeingEaten(plant.GridRow, plant.GridCol)

	// 处理被啃食状态变化
	if isBeingEaten != plant.BeingEaten {
		plant.BeingEaten = isBeingEaten

		// 获取 ReanimComponent 用于暂停/恢复动画
		if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
			// 初始化暂停状态 map（如果为空）
			if reanim.AnimationPausedStates == nil {
				reanim.AnimationPausedStates = make(map[string]bool)
			}

			if isBeingEaten {
				// 刚开始被啃食：暂停身体摇晃动画使其保持静止
				reanim.AnimationPausedStates["anim_idle"] = true
				log.Printf("[BehaviorSystem] 豌豆射手 %d 开始被啃食，暂停摇晃动画", entityID)
			} else {
				// 停止被啃食，恢复身体摇晃动画
				reanim.AnimationPausedStates["anim_idle"] = false
				log.Printf("[BehaviorSystem] 豌豆射手 %d 停止被啃食，恢复摇晃动画", entityID)
			}
		}
	}

	// 获取豌豆射手的位置组件
	peashooterPos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 计算豌豆射手所在的行
	peashooterRow := utils.GetEntityRow(peashooterPos.Y, config.GridWorldStartY, config.CellHeight)

	// 扫描同行僵尸：查找在豌豆射手正前方（右侧）且在攻击范围内的僵尸
	hasZombieInLine := false
	screenRightBoundary := config.GridWorldEndX + 50.0

	for _, zombieID := range zombieEntityList {
		zombiePos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
		if !ok {
			continue
		}

		// 检查僵尸是否已死亡（过滤死亡状态的僵尸）
		zombieBehavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
		if !ok || zombieBehavior.Type == components.BehaviorZombieDying {
			continue // 跳过死亡中的僵尸
		}

		// 跳过正在跳跃的撑杆僵尸（跳跃期间免疫子弹，植物不应攻击）
		if poleVault, ok := ecs.GetComponent[*components.PoleVaultComponent](s.entityManager, zombieID); ok {
			if poleVault.IsJumping {
				continue
			}
		}

		// 计算僵尸所在的行
		zombieRow := utils.GetEntityRow(zombiePos.Y, config.GridWorldStartY, config.CellHeight)

		// 检查僵尸是否在同一行、在豌豆射手右侧、且已进入屏幕可见区域
		if zombieRow == peashooterRow &&
			zombiePos.X > peashooterPos.X &&
			zombiePos.X < screenRightBoundary {
			hasZombieInLine = true
			break
		}
	}

	// 状态切换逻辑
	if plant.AttackAnimState == components.AttackAnimAttacking {
		// 当前是攻击状态
		if !hasZombieInLine {
			// 没有僵尸了，切换回空闲状态
			log.Printf("[BehaviorSystem] 豌豆射手 %d 没有目标，切换回空闲状态", entityID)
			ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
				UnitID:           unitID,
				ComboName:        "idle",
				Processed:        false,
				PreserveProgress: true,
			})
			plant.AttackAnimState = components.AttackAnimIdle
			plant.LastFiredFrame = -1 // 重置，下次进入攻击状态时可以立即发射
		}
		// 有僵尸则继续保持攻击状态，updatePlantAttackAnimation 会处理子弹发射
	} else {
		// 当前是空闲状态
		if hasZombieInLine {
			// 有僵尸，切换到攻击动画
			log.Printf("[BehaviorSystem] 🎯 豌豆射手 %d 发现目标，切换到攻击动画", entityID)
			ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
				UnitID:           unitID,
				ComboName:        "attack_with_sway",
				Processed:        false,
				PreserveProgress: true,
			})
			plant.AttackAnimState = components.AttackAnimAttacking
			plant.LastFiredFrame = -1 // 重置，允许立即发射
		}
	}
}

// updatePlantAttackAnimation 处理射手植物的攻击动画帧事件
// 当动画到达关键帧时发射子弹，攻击频率由动画播放速度决定
func (s *BehaviorSystem) updatePlantAttackAnimation(entityID ecs.EntityID, deltaTime float64) {
	plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID)
	if !ok || plant.AttackAnimState != components.AttackAnimAttacking {
		return
	}

	// 获取 ReanimComponent 检查动画状态
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	currentFrame := reanim.CurrentFrame

	// 防止在同一个关键帧内重复发射（动画可能在同一帧停留多个 Update）
	if currentFrame == plant.LastFiredFrame {
		return
	}

	// 检测动画循环：如果当前帧小于上次发射帧，说明动画已循环
	// 此时需要重置 LastFiredFrame，允许下次到达发射帧时再次发射
	if currentFrame < plant.LastFiredFrame {
		plant.LastFiredFrame = -1
	}

	// 到达发射关键帧，发射子弹
	if currentFrame == config.PeashooterShootingFireFrame {
		log.Printf("[BehaviorSystem] 🔫 豌豆射手 %d 到达关键帧(%d)，发射子弹！",
			entityID, currentFrame)

		// 获取植物世界坐标
		pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		if !ok {
			return
		}

		// 子弹起始位置 = 植物位置 + 固定偏移
		bulletStartX := pos.X + config.PeaBulletOffsetX
		bulletStartY := pos.Y + config.PeaBulletOffsetY

		// 播放发射音效
		s.playShootSound()

		// 根据植物类型创建不同的子弹
		var bulletID ecs.EntityID
		var err error
		switch plant.PlantType {
		case components.PlantSnowPea:
			// 寒冰射手发射冰豌豆
			bulletID, err = entities.NewSnowPeaProjectile(s.entityManager, s.resourceManager, bulletStartX, bulletStartY, plant.GridRow)
			if err != nil {
				log.Printf("[BehaviorSystem] 创建冰豌豆子弹失败: %v", err)
			} else {
				log.Printf("[BehaviorSystem] 寒冰射手 %d 发射冰豌豆 %d", entityID, bulletID)
				// 为冰豌豆子弹添加雪花尾迹粒子效果（Y 偏移使粒子在子弹中心发射）
				if trailErr := entities.AttachTrailEffect(s.entityManager, s.resourceManager, bulletID, "SnowPeaTrail", config.PeaBulletTrailOffsetY); trailErr != nil {
					log.Printf("[BehaviorSystem] 添加冰豌豆尾迹效果失败: %v", trailErr)
				}
			}
		default:
			// 豌豆射手发射普通豌豆
			bulletID, err = entities.NewPeaProjectile(s.entityManager, s.resourceManager, bulletStartX, bulletStartY, plant.GridRow)
			if err != nil {
				log.Printf("[BehaviorSystem] 创建豌豆子弹失败: %v", err)
			} else {
				log.Printf("[BehaviorSystem] 豌豆射手 %d 发射子弹 %d", entityID, bulletID)
			}
		}

		// 记录本次发射的帧号，防止在同一帧内重复发射
		plant.LastFiredFrame = currentFrame
	}
}

// handleRepeaterBehavior 处理双发射手的攻击行为
// Story 8.12: 双发射手每次攻击发射2颗豌豆，与豌豆射手共用目标检测逻辑
func (s *BehaviorSystem) handleRepeaterBehavior(entityID ecs.EntityID, deltaTime float64, zombieEntityList []ecs.EntityID) {
	// 获取植物组件（用于状态管理）
	plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[BehaviorSystem] ⚠️ 双发射手 %d 缺少 PlantComponent", entityID)
		return
	}

	unitID := "repeater"

	// 检测双发射手是否正在被啃食
	isBeingEaten := s.isPlantBeingEaten(plant.GridRow, plant.GridCol)

	// 处理被啃食状态变化
	if isBeingEaten != plant.BeingEaten {
		plant.BeingEaten = isBeingEaten

		// 获取 ReanimComponent 用于暂停/恢复动画
		if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
			if reanim.AnimationPausedStates == nil {
				reanim.AnimationPausedStates = make(map[string]bool)
			}

			if isBeingEaten {
				reanim.AnimationPausedStates["anim_idle"] = true
				log.Printf("[BehaviorSystem] 双发射手 %d 开始被啃食，暂停摇晃动画", entityID)
			} else {
				reanim.AnimationPausedStates["anim_idle"] = false
				log.Printf("[BehaviorSystem] 双发射手 %d 停止被啃食，恢复摇晃动画", entityID)
			}
		}
	}

	// 处理连发计时器（在动画帧事件之外独立更新）
	if plant.BurstShotsRemaining > 0 && plant.BurstTimer > 0 {
		plant.BurstTimer -= deltaTime
		if plant.BurstTimer <= 0 {
			// 计时器到期，发射下一颗豌豆
			s.fireRepeaterBullet(entityID, plant)
			plant.BurstShotsRemaining--
			if plant.BurstShotsRemaining > 0 {
				plant.BurstTimer = config.RepeaterBurstInterval
			}
		}
	}

	// 获取双发射手的位置组件
	repeaterPos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 计算双发射手所在的行
	repeaterRow := utils.GetEntityRow(repeaterPos.Y, config.GridWorldStartY, config.CellHeight)

	// 扫描同行僵尸
	hasZombieInLine := false
	screenRightBoundary := config.GridWorldEndX + 50.0

	for _, zombieID := range zombieEntityList {
		zombiePos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
		if !ok {
			continue
		}

		zombieBehavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
		if !ok || zombieBehavior.Type == components.BehaviorZombieDying {
			continue
		}

		// 跳过正在跳跃的撑杆僵尸（跳跃期间免疫子弹，植物不应攻击）
		if poleVault, ok := ecs.GetComponent[*components.PoleVaultComponent](s.entityManager, zombieID); ok {
			if poleVault.IsJumping {
				continue
			}
		}

		zombieRow := utils.GetEntityRow(zombiePos.Y, config.GridWorldStartY, config.CellHeight)

		if zombieRow == repeaterRow &&
			zombiePos.X > repeaterPos.X &&
			zombiePos.X < screenRightBoundary {
			hasZombieInLine = true
			break
		}
	}

	// 状态切换逻辑
	if plant.AttackAnimState == components.AttackAnimAttacking {
		if !hasZombieInLine {
			log.Printf("[BehaviorSystem] 双发射手 %d 没有目标，切换回空闲状态", entityID)
			ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
				UnitID:           unitID,
				ComboName:        "idle",
				Processed:        false,
				PreserveProgress: true,
			})
			plant.AttackAnimState = components.AttackAnimIdle
			plant.LastFiredFrame = -1
			plant.BurstShotsRemaining = 0
		}
	} else {
		if hasZombieInLine {
			log.Printf("[BehaviorSystem] 🎯 双发射手 %d 发现目标，切换到攻击动画", entityID)
			ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
				UnitID:           unitID,
				ComboName:        "attack_with_sway",
				Processed:        false,
				PreserveProgress: true,
			})
			plant.AttackAnimState = components.AttackAnimAttacking
			plant.LastFiredFrame = -1
		}
	}
}

// updateRepeaterAttackAnimation 处理双发射手的攻击动画帧事件
// Story 8.12: 在动画关键帧时开始连发序列
func (s *BehaviorSystem) updateRepeaterAttackAnimation(entityID ecs.EntityID, deltaTime float64) {
	plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID)
	if !ok || plant.AttackAnimState != components.AttackAnimAttacking {
		return
	}

	reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	currentFrame := reanim.CurrentFrame

	if currentFrame == plant.LastFiredFrame {
		return
	}

	if currentFrame < plant.LastFiredFrame {
		plant.LastFiredFrame = -1
	}

	// 到达发射关键帧，开始连发序列
	if currentFrame == config.PeashooterShootingFireFrame && plant.BurstShotsRemaining == 0 {
		log.Printf("[BehaviorSystem] 🔫🔫 双发射手 %d 到达关键帧(%d)，开始连发！",
			entityID, currentFrame)

		// 发射第一颗豌豆
		s.fireRepeaterBullet(entityID, plant)

		// 设置连发状态（还需要发射 1 颗）
		plant.BurstShotsRemaining = config.RepeaterBurstCount - 1
		plant.BurstTimer = config.RepeaterBurstInterval

		plant.LastFiredFrame = currentFrame
	}
}

// fireRepeaterBullet 双发射手发射豌豆
func (s *BehaviorSystem) fireRepeaterBullet(entityID ecs.EntityID, plant *components.PlantComponent) {
	pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	bulletStartX := pos.X + config.PeaBulletOffsetX
	bulletStartY := pos.Y + config.PeaBulletOffsetY

	// 播放发射音效
	s.playShootSound()

	// 双发射手发射普通豌豆
	bulletID, err := entities.NewPeaProjectile(s.entityManager, s.resourceManager, bulletStartX, bulletStartY, plant.GridRow)
	if err != nil {
		log.Printf("[BehaviorSystem] 创建双发射手豌豆失败: %v", err)
	} else {
		log.Printf("[BehaviorSystem] 双发射手 %d 发射豌豆 %d", entityID, bulletID)
	}
}
