package behavior

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/utils"
)

func (s *BehaviorSystem) handlePeashooterBehavior(entityID ecs.EntityID, deltaTime float64, zombieEntityList []ecs.EntityID) {
	// 获取植物组件（用于状态管理）
	plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[BehaviorSystem] ⚠️ 豌豆射手 %d 缺少 PlantComponent", entityID)
		return
	}

	// 获取计时器组件
	timer, ok := ecs.GetComponent[*components.TimerComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[BehaviorSystem] ⚠️ 豌豆射手 %d 缺少 TimerComponent", entityID)
		return
	}

	// 更新计时器
	timer.CurrentTime += deltaTime

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

	// 如果正在攻击状态
	if plant.AttackAnimState == components.AttackAnimAttacking {
		// 检查是否还有僵尸
		if !hasZombieInLine {
			// 没有僵尸了，切换回空闲状态
			log.Printf("[BehaviorSystem] 豌豆射手 %d 没有目标，切换回空闲状态", entityID)
			ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
				UnitID:           "peashootersingle",
				ComboName:        "idle", // 使用配置驱动的 idle 组合（播放 anim_full_idle）
				Processed:        false,
				PreserveProgress: true, // 保留动画进度，避免抖动
			})
			plant.AttackAnimState = components.AttackAnimIdle
			plant.PendingProjectile = false
		} else {
			// 有僵尸且计时器就绪，准备下一次发射
			if timer.CurrentTime >= timer.TargetTime && !plant.PendingProjectile {
				// 获取当前动画帧号
				reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
				if ok && reanim.CurrentFrame == config.PeashooterShootingFireFrame {
					// 当前帧恰好是关键帧，延后一帧再设置 PendingProjectile
					// 避免在同一帧内立即发射
					log.Printf("[BehaviorSystem] ⏸️ 豌豆射手 %d 计时器就绪但当前在关键帧(%d)，延后1帧",
						entityID, config.PeashooterShootingFireFrame)
					return
				}

				plant.PendingProjectile = true
				plant.LastFiredFrame = -1 // 重置发射帧号，允许新的射击周期
				timer.CurrentTime = 0
				log.Printf("[BehaviorSystem] 🎯 豌豆射手 %d 计时器就绪(%.3f)，设置 PendingProjectile=true, 重置 LastFiredFrame=-1（攻击状态中）",
					entityID, timer.CurrentTime)
			}
		}
		// 继续在攻击状态，updatePlantAttackAnimation 会处理子弹发射
		return
	}

	// 空闲状态，检查是否有僵尸需要攻击
	if timer.CurrentTime >= timer.TargetTime && hasZombieInLine {
		// 获取当前动画帧号（如果有的话）
		reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
		if ok && reanim.CurrentFrame == config.PeashooterShootingFireFrame {
			// 当前帧恰好是关键帧（从空闲切换时不太可能，但还是检查一下）
			log.Printf("[BehaviorSystem] ⏸️ 豌豆射手 %d 空闲状态计时器就绪但当前在关键帧(%d)，延后1帧",
				entityID, config.PeashooterShootingFireFrame)
			return
		}

		// 切换到攻击动画
		ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
			UnitID:           "peashootersingle",
			ComboName:        "attack_with_sway",
			Processed:        false,
			PreserveProgress: true, // 保留动画进度，避免抖动
		})

		log.Printf("[BehaviorSystem] 🎯 豌豆射手 %d 切换到攻击动画（配置驱动），计时器=%.3f", entityID, timer.CurrentTime)
		plant.AttackAnimState = components.AttackAnimAttacking

		// 设置"等待发射"状态，但不立即创建子弹
		plant.PendingProjectile = true
		plant.LastFiredFrame = -1 // 重置发射帧号，允许新的射击周期
		log.Printf("[BehaviorSystem] 豌豆射手 %d 进入攻击状态，等待关键帧(%d)发射子弹，设置 PendingProjectile=true, LastFiredFrame=-1",
			entityID, config.PeashooterShootingFireFrame)

		// 重置计时器
		timer.CurrentTime = 0
	}
}

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

	// 关键帧事件监听 - 子弹发射时机同步
	if plant.PendingProjectile {
		// 直接使用 CurrentFrame
		currentFrame := reanim.CurrentFrame

		// 防止在同一个关键帧内重复发射（循环动画问题）
		if currentFrame == plant.LastFiredFrame {
			// 仍在上次发射的同一帧，跳过
			return
		}

		// 精确匹配发射帧（零延迟）
		if currentFrame == config.PeashooterShootingFireFrame {
			// 获取计时器信息用于调试
			timer, _ := ecs.GetComponent[*components.TimerComponent](s.entityManager, entityID)
			timerValue := 0.0
			if timer != nil {
				timerValue = timer.CurrentTime
			}
			log.Printf("[BehaviorSystem] 🔫 豌豆射手 %d 到达关键帧(%d)，发射子弹！计时器=%.3f, 动画帧索引=%v",
				entityID, currentFrame, timerValue, reanim.AnimationFrameIndices)

			// 使用固定偏移值计算子弹发射位置
			bulletOffsetX := config.PeaBulletOffsetX
			bulletOffsetY := config.PeaBulletOffsetY

			// 获取植物世界坐标
			pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
			if !ok {
				return
			}

			// 子弹起始位置 = 植物位置 + 固定偏移
			bulletStartX := pos.X + bulletOffsetX
			bulletStartY := pos.Y + bulletOffsetY

			log.Printf("[BehaviorSystem] 射手 %d 在关键帧发射子弹，位置: (%.1f, %.1f)",
				entityID, bulletStartX, bulletStartY)

			// 播放发射音效
			s.playShootSound()

			// Story 8.9: 根据植物类型创建不同的子弹
			var bulletID ecs.EntityID
			var err error
			switch plant.PlantType {
			case components.PlantSnowPea:
				// 寒冰射手发射冰豌豆
				bulletID, err = entities.NewSnowPeaProjectile(s.entityManager, s.resourceManager, bulletStartX, bulletStartY)
				if err != nil {
					log.Printf("[BehaviorSystem] 创建冰豌豆子弹失败: %v", err)
				} else {
					log.Printf("[BehaviorSystem] 寒冰射手 %d 发射冰豌豆 %d（零延迟帧同步）", entityID, bulletID)
				}
			default:
				// 豌豆射手发射普通豌豆
				bulletID, err = entities.NewPeaProjectile(s.entityManager, s.resourceManager, bulletStartX, bulletStartY)
				if err != nil {
					log.Printf("[BehaviorSystem] 创建豌豆子弹失败: %v", err)
				} else {
					log.Printf("[BehaviorSystem] 豌豆射手 %d 发射子弹 %d（零延迟帧同步）", entityID, bulletID)
				}
			}

			// 清除"等待发射"状态
			plant.PendingProjectile = false
			// 记录本次发射的帧号，防止在同一帧内重复发射
			plant.LastFiredFrame = currentFrame
			log.Printf("[BehaviorSystem] ✅ 射手 %d 清除 PendingProjectile=false, LastFiredFrame=%d", entityID, currentFrame)
		}
	}

	// 注意：攻击动画现在是循环的，不依赖 IsFinished 切换回空闲
	// 切换回空闲状态的逻辑在 handlePeashooterBehavior 中（检测没有僵尸时）
}
