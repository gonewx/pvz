package behavior

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/systems"
)

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

	// 4. 切换 BehaviorComponent.Type 为 BehaviorZombieEating
	behavior.Type = components.BehaviorZombieEating
	// 初始化啃食动画帧跟踪（用于伤害和音效同步）
	// -1 表示尚未开始，首次进入会触发伤害
	behavior.LastEatAnimFrame = -1

	// 切换僵尸动画为啃食状态
	s.changeZombieAnimation(zombieID, components.ZombieAnimEating)
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
					// 粒子位置：使用僵尸下巴轨道位置（anim_head2）获取精确的嘴巴位置
					// 参考 zombie_death_handler.go 中头部掉落粒子效果的处理方式
					particleX, particleY := pos.X, pos.Y // 回退值
					if s.reanimSystem != nil {
						// 使用 AnchorCenter 获取下巴中心位置，作为啃食接触点
						if trackX, trackY, found := s.reanimSystem.GetTrackWorldPosition(entityID, "anim_head2", systems.AnchorCenter); found {
							particleX, particleY = trackX, trackY
							log.Printf("[BehaviorSystem] 僵尸 %d 下巴轨道位置: (%.1f, %.1f)", entityID, particleX, particleY)
						} else {
							log.Printf("[BehaviorSystem] 警告：僵尸 %d 无法获取下巴轨道位置，使用回退值", entityID)
						}
					}
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
