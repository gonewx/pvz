package behavior

import (
	"log"
	"math/rand"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/utils"
)

func (s *BehaviorSystem) handleSunflowerBehavior(entityID ecs.EntityID, deltaTime float64) {
	// 获取计时器组件
	timer, _ := ecs.GetComponent[*components.TimerComponent](s.entityManager, entityID)

	// 更新计时器
	timer.CurrentTime += deltaTime

	// 检查计时器是否完成
	if timer.CurrentTime >= timer.TargetTime {
		log.Printf("[BehaviorSystem] 向日葵生产阳光！计时器: %.2f/%.2f 秒", timer.CurrentTime, timer.TargetTime)

		// 获取位置组件，计算阳光生成位置
		position, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		plant, _ := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID)

		log.Printf("[BehaviorSystem] 向日葵位置: (%.0f, %.0f), 网格: (col=%d, row=%d)",
			position.X, position.Y, plant.GridCol, plant.GridRow)

		// 阳光生成位置：向日葵位置附近随机偏移
		// 向日葵生产的阳光应该从向日葵中心弹出，然后落到附近随机位置
		// position.X, position.Y 是向日葵中心的世界坐标

		// 阳光生成逻辑：
		// position.X, position.Y 是向日葵的中心位置（Reanim 的 CenterOffset 已经处理了对齐）
		// 阳光的 PositionComponent 也表示阳光的中心位置（阳光的 CenterOffset 会自动处理渲染）

		// 随机目标偏移：决定阳光落地位置相对于向日葵的偏移
		randomOffsetX := (rand.Float64() - 0.5) * config.SunRandomOffsetRangeX // -30 ~ +30
		randomOffsetY := (rand.Float64() - 0.5) * config.SunRandomOffsetRangeY // -20 ~ +20

		// 阳光起始位置（中心）：从向日葵中心开始
		sunStartX := position.X
		sunStartY := position.Y

		// 阳光目标位置（中心）：向日葵下方 + 随机偏移
		// config.SunDropBelowPlantOffset: 阳光落在向日葵下方约50像素的位置（视觉上自然）
		sunTargetX := position.X + randomOffsetX
		sunTargetY := position.Y + config.SunDropBelowPlantOffset + randomOffsetY

		log.Printf("[BehaviorSystem] 向日葵中心: (%.1f, %.1f), 阳光起始中心: (%.1f, %.1f)",
			position.X, position.Y, sunStartX, sunStartY)

		// 边界检查（AC10）：确保阳光目标位置在屏幕内
		// 屏幕尺寸800x600，阳光尺寸80x80（半径40）
		// 中心坐标有效范围：[40, 760] x [40, 560]
		sunRadius := config.SunOffsetCenterX // 40
		if sunTargetX < sunRadius {
			sunTargetX = sunRadius
		}
		if sunTargetX > 800-sunRadius {
			sunTargetX = 800 - sunRadius
		}
		if sunTargetY < sunRadius {
			sunTargetY = sunRadius
		}
		if sunTargetY > 600-sunRadius {
			sunTargetY = 600 - sunRadius
		}

		log.Printf("[BehaviorSystem] 创建阳光实体，起始位置: (%.0f, %.0f), 目标位置: (%.0f, %.0f), 随机偏移: (%.1f, %.1f)",
			sunStartX, sunStartY, sunTargetX, sunTargetY, randomOffsetX, randomOffsetY)

		// 创建向日葵生产的阳光实体
		sunID := entities.NewPlantSunEntity(s.entityManager, s.resourceManager, sunStartX, sunStartY, sunTargetX, sunTargetY)

		// 设置阳光的速度：抛物线运动
		// 阳光先向上弹起，然后在重力作用下落到目标位置
		sunVel, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, sunID)
		if ok {
			// 使用固定的向上初速度，让阳光弹起
			initialUpwardSpeed := -100.0 // 向上初速度（负值表示向上）

			// 水平速度：匀速运动到目标X位置
			duration := 1.5 // 预计运动时间（秒）
			sunVel.VX = (sunTargetX - sunStartX) / duration

			// 垂直初速度：固定向上弹起
			// 重力会自然地将阳光拉向目标位置
			sunVel.VY = initialUpwardSpeed
		}

		log.Printf("[BehaviorSystem] 阳光实体创建完成，ID=%d, 状态: Rising, 速度: (%.1f, %.1f)",
			sunID, sunVel.VX, sunVel.VY)

		// 简单实体的动画已在 createSimpleReanimComponent 中初始化，无需额外初始化

		// 重置计时器
		timer.CurrentTime = 0
		// 首次生产后，后续生产周期为 24 秒
		timer.TargetTime = 24.0

		// 注意：向日葵的待机动画一直循环播放，生产阳光时不需要特殊动画
	}
}

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

	// DEBUG: 每100帧输出一次状态
	s.logFrameCounter++
	if s.logFrameCounter%100 == 0 {
		log.Printf("[BehaviorSystem] 🔫 豌豆射手 %d 状态: AttackState=%d, Timer=%.2f/%.2f, 僵尸数=%d",
			entityID, plant.AttackAnimState, timer.CurrentTime, timer.TargetTime, len(zombieEntityList))
	}

	// 只有在空闲状态时才能触发新的攻击
	// 确保攻击动画播放完毕后，才能进行下一次攻击
	if plant.AttackAnimState != components.AttackAnimIdle {
		// DEBUG: 记录非空闲状态
		if s.logFrameCounter%100 == 0 {
			log.Printf("[BehaviorSystem] 豌豆射手 %d 不在空闲状态（AttackState=%d），跳过攻击检测", entityID, plant.AttackAnimState)
		}
		return // 攻击动画正在播放，跳过攻击逻辑
	}

	// 检查计时器是否就绪（达到攻击间隔）
	if timer.CurrentTime >= timer.TargetTime {
		// 获取豌豆射手的位置组件
		peashooterPos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		if !ok {
			return
		}

		// 计算豌豆射手所在的行
		peashooterRow := utils.GetEntityRow(peashooterPos.Y, config.GridWorldStartY, config.CellHeight)

		// 扫描同行僵尸：查找在豌豆射手正前方（右侧）且在攻击范围内的僵尸
		hasZombieInLine := false

		// DEBUG: 输出僵尸列表信息（每秒一次）
		s.logFrameCounter++
		if s.logFrameCounter >= LogOutputFrameInterval && len(zombieEntityList) > 0 {
			log.Printf("[BehaviorSystem] 扫描僵尸: 总数=%d, 豌豆射手行=%d", len(zombieEntityList), peashooterRow)
			s.logFrameCounter = 0
		}

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
			// 只攻击屏幕内的僵尸（X坐标 < 屏幕右边界，约800）
			// 使用 config.GridWorldEndX (971) 作为攻击范围右边界，确保僵尸进入草坪后才被攻击
			screenRightBoundary := config.GridWorldEndX + 50.0 // 草坪边界右侧50像素内可攻击
			if zombieRow == peashooterRow &&
				zombiePos.X > peashooterPos.X &&
				zombiePos.X < screenRightBoundary {
				hasZombieInLine = true
				// DEBUG: 只在找到目标时输出
				log.Printf("[BehaviorSystem] 发现目标僵尸 %d: 位置=(%.1f, %.1f), 豌豆射手X=%.1f, 攻击边界=%.1f",
					zombieID, zombiePos.X, zombiePos.Y, peashooterPos.X, screenRightBoundary)
				break
			}
		}

		// 如果有僵尸在同一行，发射子弹
		if hasZombieInLine {
			// 使用组件通信替代直接调用
			// 使用配置驱动的动画播放
			// 播放配置文件中定义的攻击动画组合
			ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
				UnitID:    "peashootersingle",
				ComboName: "attack_with_sway",
				Processed: false,
			})

			// 设置为非循环模式（单次播放）
			if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
				reanim.IsLooping = false
			}

			log.Printf("[BehaviorSystem] 豌豆射手 %d 切换到攻击动画（配置驱动）", entityID)
			// 设置攻击动画状态，用于动画完成后切换回 idle
			plant.AttackAnimState = components.AttackAnimAttacking

			// 设置"等待发射"状态，但不立即创建子弹
			// 子弹将在攻击动画的关键帧（Frame 5）创建
			plant.PendingProjectile = true
			log.Printf("[BehaviorSystem] 豌豆射手 %d 进入攻击状态，等待关键帧(%d)发射子弹",
				entityID, config.PeashooterShootingFireFrame)

			// 重置计时器
			timer.CurrentTime = 0
		}
		// 如果没有僵尸，不发射子弹，计时器也不重置（保持就绪状态）
	}
}

func (s *BehaviorSystem) handleWallnutBehavior(entityID ecs.EntityID) {
	// 获取生命值组件
	health, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 计算生命值百分比
	healthPercent := float64(health.CurrentHealth) / float64(health.MaxHealth)

	// 使用 ReanimComponent 实现外观状态切换
	// 根据生命值百分比动态替换 PartImages 中的身体图片
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 确定应显示的身体图片路径
	var targetBodyImagePath string
	if healthPercent > config.WallnutCracked1Threshold {
		// 完好状态 (> 66%)
		targetBodyImagePath = "assets/reanim/wallnut_body.png"
	} else if healthPercent > config.WallnutCracked2Threshold {
		// 轻伤状态 (33% - 66%)
		targetBodyImagePath = "assets/reanim/wallnut_cracked1.png"
	} else {
		// 重伤状态 (< 33%)
		targetBodyImagePath = "assets/reanim/wallnut_cracked2.png"
	}

	// 检查是否需要切换图片（避免每帧重复加载）
	currentBodyImage, exists := reanim.PartImages["IMAGE_REANIM_WALLNUT_BODY"]
	if !exists {
		return
	}

	// 加载目标图片
	targetBodyImage, err := s.resourceManager.LoadImage(targetBodyImagePath)
	if err != nil {
		log.Printf("[BehaviorSystem] 警告：无法加载坚果墙图片 %s: %v", targetBodyImagePath, err)
		return
	}

	// 如果图片不同，则替换
	if currentBodyImage != targetBodyImage {
		reanim.PartImages["IMAGE_REANIM_WALLNUT_BODY"] = targetBodyImage
		log.Printf("[BehaviorSystem] 坚果墙 %d 切换外观: HP=%d/%d (%.1f%%), 图片=%s",
			entityID, health.CurrentHealth, health.MaxHealth, healthPercent*100, targetBodyImagePath)
	}
}

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

	// 计算基于世界坐标的3x3爆炸范围
	// 3x3格子 = 横向和纵向各覆盖1.5个格子的距离
	// 这样可以确保即使在边缘网格，爆炸范围也能扩展到网格外
	explosionRadiusX := (float64(config.CherryBombRangeRadius) + 0.5) * config.CellWidth  // 1.5 * 80 = 120
	explosionRadiusY := (float64(config.CherryBombRangeRadius) + 0.5) * config.CellHeight // 1.5 * 100 = 150

	minX := position.X - explosionRadiusX
	maxX := position.X + explosionRadiusX
	minY := position.Y - explosionRadiusY
	maxY := position.Y + explosionRadiusY

	log.Printf("[BehaviorSystem] 樱桃炸弹爆炸范围 (世界坐标): X[%.1f-%.1f], Y[%.1f-%.1f]", minX, maxX, minY, maxY)

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
			behavior.Type != components.BehaviorZombieDying {
			continue
		}

		// 获取僵尸的位置组件
		zombiePos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
		if !ok {
			continue
		}

		// 使用世界坐标检查僵尸是否在爆炸范围内
		// 这样可以覆盖网格边界外的僵尸
		if zombiePos.X >= minX && zombiePos.X <= maxX &&
			zombiePos.Y >= minY && zombiePos.Y <= maxY {
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
				}
			}
		}
	}

	log.Printf("[BehaviorSystem] 樱桃炸弹爆炸影响了 %d 个僵尸", affectedZombies)

	// 播放爆炸音效
	if config.CherryBombExplodeSoundPath != "" {
		soundPlayer, err := s.resourceManager.LoadSoundEffect(config.CherryBombExplodeSoundPath)
		if err != nil {
			log.Printf("[BehaviorSystem] 警告：加载樱桃炸弹爆炸音效失败: %v", err)
		} else {
			soundPlayer.Rewind()
			soundPlayer.Play()
			log.Printf("[BehaviorSystem] 播放樱桃炸弹爆炸音效")
		}
	}

	// 创建爆炸粒子效果
	// 触发爆炸粒子效果（使用已获取的position组件）
	_, err := entities.CreateParticleEffect(
		s.entityManager,
		s.resourceManager,
		"BossExplosion", // 粒子效果名称（不带.xml后缀）
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
		err := s.lawnGridSystem.ReleaseCell(s.lawnGridEntityID, plantComp.GridCol, plantComp.GridRow)
		if err != nil {
			log.Printf("[BehaviorSystem] 警告：释放樱桃炸弹网格占用失败: %v", err)
		} else {
			log.Printf("[BehaviorSystem] 樱桃炸弹网格 (%d, %d) 已释放", plantComp.GridCol, plantComp.GridRow)
		}
	}

	// 删除樱桃炸弹实体
	s.entityManager.DestroyEntity(entityID)
	log.Printf("[BehaviorSystem] 樱桃炸弹 %d 已删除", entityID)
}

func (s *BehaviorSystem) updatePlantAttackAnimation(entityID ecs.EntityID, deltaTime float64) {
	plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID)
	if !ok || plant.AttackAnimState != components.AttackAnimAttacking {
		return
	}

	// 获取 ReanimComponent 检查动画是否完成
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 关键帧事件监听 - 子弹发射时机同步
	if plant.PendingProjectile {
		// 直接使用 CurrentFrame
		currentFrame := reanim.CurrentFrame
		// 精确匹配发射帧（零延迟）
		if currentFrame == config.PeashooterShootingFireFrame {
			log.Printf("[BehaviorSystem] 豌豆射手 %d 到达关键帧(%d)，发射子弹！",
				entityID, currentFrame)

			// 使用固定偏移值计算子弹发射位置
			// 注意：经过测试，Reanim 轨道坐标（如 idle_mouth, anim_stem）不直接提供嘴部位置
			// - idle_mouth 轨道坐标为 (0, 0)（无运动数据）
			// - anim_stem 轨道坐标为茎部中心，不是嘴部
			// 因此使用固定偏移值（相对于格子中心）来计算子弹起始位置
			bulletOffsetX := config.PeaBulletOffsetX
			bulletOffsetY := config.PeaBulletOffsetY

			// 获取植物世界坐标
			pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
			if !ok {
				return
			}

			// 子弹起始位置 = 植物位置 + 固定偏移
			// 这个偏移是经过调优的，相对于格子中心的偏移
			bulletStartX := pos.X + bulletOffsetX
			bulletStartY := pos.Y + bulletOffsetY

			log.Printf("[BehaviorSystem] 豌豆射手 %d 在关键帧发射子弹，位置: (%.1f, %.1f)",
				entityID, bulletStartX, bulletStartY)

			// 播放发射音效
			s.playShootSound()

			// 创建豌豆子弹实体
			bulletID, err := entities.NewPeaProjectile(s.entityManager, s.resourceManager, bulletStartX, bulletStartY)
			if err != nil {
				log.Printf("[BehaviorSystem] 创建豌豆子弹失败: %v", err)
			} else {
				log.Printf("[BehaviorSystem] 豌豆射手 %d 发射子弹 %d（零延迟帧同步）", entityID, bulletID)
			}

			// 清除"等待发射"状态
			plant.PendingProjectile = false
		}
	}

	// 检查攻击动画是否播放完毕，切换回 idle
	if reanim.IsFinished {
		// 使用配置驱动的动画播放
		// 根据植物类型确定 unitID
		var unitID string
		switch plant.PlantType {
		case components.PlantPeashooter:
			unitID = "peashooter"
		case components.PlantSunflower:
			unitID = "sunflower"
		case components.PlantWallnut:
			unitID = "wallnut"
		case components.PlantCherryBomb:
			unitID = "cherrybomb"
		default:
			log.Printf("[BehaviorSystem] 未知的植物类型: %v", plant.PlantType)
			return
		}

		// 使用组件通信替代直接调用
		// 使用 AnimationCommand 组件播放默认动画组合
		ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
			UnitID:    unitID,
			ComboName: "",
			Processed: false,
		})
		plant.AttackAnimState = components.AttackAnimIdle
		log.Printf("[BehaviorSystem] 植物 %d 攻击动画完成，添加空闲动画命令（配置驱动）", entityID)
	}
}
