package behavior

import (
	"log"
	"math"
	"math/rand"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

func (s *BehaviorSystem) handleSunflowerBehavior(entityID ecs.EntityID, deltaTime float64) {
	// 获取计时器组件
	timer, ok := ecs.GetComponent[*components.TimerComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[BehaviorSystem] ⚠️ 向日葵 %d 缺少 TimerComponent!", entityID)
		return
	}

	// 记录更新前的时间（用于检测是否跨过预热阈值）
	prevTime := timer.CurrentTime

	// 更新计时器
	timer.CurrentTime += deltaTime

	// 检查是否需要提前触发发光效果（预热）
	// 在阳光生产前 SunflowerGlowPrewarmTime 秒开始发光
	prewarmThreshold := timer.TargetTime - config.SunflowerGlowPrewarmTime
	if prevTime < prewarmThreshold && timer.CurrentTime >= prewarmThreshold {
		// 刚刚跨过预热阈值，触发发光效果
		_, hasGlow := ecs.GetComponent[*components.SunflowerGlowComponent](s.entityManager, entityID)
		if !hasGlow {
			ecs.AddComponent(s.entityManager, entityID, &components.SunflowerGlowComponent{
				Intensity:    0.0,  // 从 0 开始，逐渐亮起
				MaxIntensity: 1.0,  // 最大强度
				IsRising:     true, // 开始亮起阶段
				RiseSpeed:    config.SunflowerGlowRiseSpeed,
				FadeSpeed:    config.SunflowerGlowFadeSpeed,
				ColorR:       config.SunflowerGlowColorR,
				ColorG:       config.SunflowerGlowColorG,
				ColorB:       config.SunflowerGlowColorB,
			})
			log.Printf("[BehaviorSystem] 向日葵 %d 预热发光效果（阳光即将生产）", entityID)
		}
	}

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

		// 根据配置决定是否生产阳光（调试开关）
		if config.SunflowerProduceSunEnabled {
			log.Printf("[BehaviorSystem] 创建阳光实体，起始位置: (%.0f, %.0f), 目标位置: (%.0f, %.0f), 随机偏移: (%.1f, %.1f)",
				sunStartX, sunStartY, sunTargetX, sunTargetY, randomOffsetX, randomOffsetY)

			// 创建向日葵生产的阳光实体
			sunID := entities.NewPlantSunEntity(s.entityManager, s.resourceManager, sunStartX, sunStartY, sunTargetX, sunTargetY)

			// 添加 AnimationCommand 组件来播放阳光动画（与自然生成的阳光一致）
			// Sun.reanim 只有轨道(Sun1, Sun2, Sun3)，使用配置的"idle"组合播放动画
			ecs.AddComponent(s.entityManager, sunID, &components.AnimationCommandComponent{
				UnitID:    "sun",
				ComboName: "idle",
				Processed: false,
			})

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
		} else {
			log.Printf("[BehaviorSystem] 向日葵阳光生产已禁用（调试模式）")
		}

		// 发光效果已在预热阶段触发，这里不需要再添加
		// 如果预热时发光组件已存在，保持其自然衰减即可

		// 重置计时器
		timer.CurrentTime = 0
		// 首次生产后，后续生产周期为 24 秒
		timer.TargetTime = 24.0
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

func (s *BehaviorSystem) handleWallnutBehavior(entityID ecs.EntityID, deltaTime float64) {
	// 获取生命值组件
	health, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 获取植物组件
	plantComp, hasPlant := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID)
	if !hasPlant {
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

	// 检测坚果墙是否正在被啃食（检查同格子是否有啃食状态的僵尸）
	isBeingEaten := s.isPlantBeingEaten(plantComp.GridRow, plantComp.GridCol)

	// 处理被啃食状态变化
	if isBeingEaten != plantComp.WallnutBeingEaten {
		plantComp.WallnutBeingEaten = isBeingEaten

		// 初始化暂停状态 map（如果为空）
		if reanim.AnimationPausedStates == nil {
			reanim.AnimationPausedStates = make(map[string]bool)
		}

		if isBeingEaten {
			// 刚开始被啃食：暂停身体动画使其保持静止
			// 不切换动画组合，只是暂停当前的 idle 动画
			reanim.AnimationPausedStates["anim_idle"] = true
			reanim.AnimationPausedStates["anim_face"] = true
			// 初始化眨眼计时器
			plantComp.WallnutBlinkTimer = config.WallnutBlinkIntervalMin +
				rand.Float64()*(config.WallnutBlinkIntervalMax-config.WallnutBlinkIntervalMin)
			log.Printf("[BehaviorSystem] 坚果墙 %d 开始被啃食，暂停身体动画", entityID)
		} else {
			// 停止被啃食，恢复身体动画
			reanim.AnimationPausedStates["anim_idle"] = false
			reanim.AnimationPausedStates["anim_face"] = false
			// 切换回 idle 动画（如果之前在播放眨眼动画）
			ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
				UnitID:    "wallnut",
				ComboName: "idle",
				Processed: false,
			})
			log.Printf("[BehaviorSystem] 坚果墙 %d 停止被啃食，恢复 idle 动画", entityID)
		}
	}

	// 被啃食时的眨眼逻辑（偶尔眨一次眼）
	if plantComp.WallnutBeingEaten {
		// 检测眨眼动画是否播放完成（使用计时器）
		if plantComp.WallnutBlinkDuration > 0 {
			plantComp.WallnutBlinkDuration -= deltaTime
			if plantComp.WallnutBlinkDuration <= 0 {
				// 眨眼动画播放完成，切换回 being_eaten 组合（只有身体，没有眨眼轨道）
				ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
					UnitID:    "wallnut",
					ComboName: "being_eaten",
					Processed: false,
				})
				// 确保身体动画保持暂停
				reanim.AnimationPausedStates["anim_idle"] = true
				reanim.AnimationPausedStates["anim_face"] = true
				plantComp.WallnutBlinkDuration = 0
				log.Printf("[BehaviorSystem] 坚果墙 %d 眨眼动画结束，恢复静止", entityID)
			}
		}

		plantComp.WallnutBlinkTimer -= deltaTime
		if plantComp.WallnutBlinkTimer <= 0 && plantComp.WallnutBlinkDuration <= 0 {
			// 随机选择眨眼动画
			blinkAnim := "blink_twice"
			blinkDuration := 0.5 // blink_twice 约 0.5 秒
			if rand.Float64() < 0.5 {
				blinkAnim = "blink_thrice"
				blinkDuration = 0.75 // blink_thrice 约 0.75 秒
			}
			// 触发眨眼动画（配置中已设置 loop: false，播放一次后停止）
			ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
				UnitID:    "wallnut",
				ComboName: blinkAnim,
				Processed: false,
			})
			// 确保身体动画保持暂停
			reanim.AnimationPausedStates["anim_idle"] = true
			reanim.AnimationPausedStates["anim_face"] = true
			// 设置眨眼动画持续时间
			plantComp.WallnutBlinkDuration = blinkDuration
			// 重置眨眼计时器
			plantComp.WallnutBlinkTimer = config.WallnutBlinkIntervalMin +
				rand.Float64()*(config.WallnutBlinkIntervalMax-config.WallnutBlinkIntervalMin)
			log.Printf("[BehaviorSystem] 坚果墙 %d 播放眨眼动画: %s, 持续 %.2f 秒", entityID, blinkAnim, blinkDuration)
		}
	}

	// 确定应显示的身体图片路径和当前状态
	// 文件名使用正确的大小写：Wallnut_xxx.png
	var targetBodyImagePath string
	var newDamageState int // 0=完好, 1=轻伤, 2=重伤
	if healthPercent > config.WallnutCracked1Threshold {
		// 完好状态 (> 66%)
		targetBodyImagePath = "assets/reanim/Wallnut_body.png"
		newDamageState = 0
	} else if healthPercent > config.WallnutCracked2Threshold {
		// 轻伤状态 (33% - 66%)
		targetBodyImagePath = "assets/reanim/Wallnut_cracked1.png"
		newDamageState = 1
	} else {
		// 重伤状态 (< 33%)
		targetBodyImagePath = "assets/reanim/Wallnut_cracked2.png"
		newDamageState = 2
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

	// 如果图片不同，则替换并触发大碎屑粒子效果
	if currentBodyImage != targetBodyImage {
		// 检查是否是从更好的状态变为更差的状态（受损状态变化）
		// 只有在状态变差时才触发 WallnutEatLarge 粒子效果
		if newDamageState > plantComp.WallnutDamageState {
			// 状态变差，触发大碎屑粒子效果
			if plantPos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID); ok {
				_, err := entities.CreateParticleEffect(
					s.entityManager,
					s.resourceManager,
					"WallnutEatLarge",
					plantPos.X,
					plantPos.Y,
				)
				if err != nil {
					log.Printf("[BehaviorSystem] 警告：创建坚果墙大碎屑粒子效果失败: %v", err)
				} else {
					log.Printf("[BehaviorSystem] 坚果墙 %d 受损状态变化 %d→%d，触发大碎屑粒子效果",
						entityID, plantComp.WallnutDamageState, newDamageState)
				}
			}
			// 更新受损状态
			plantComp.WallnutDamageState = newDamageState
		}

		reanim.PartImages["IMAGE_REANIM_WALLNUT_BODY"] = targetBodyImage
		log.Printf("[BehaviorSystem] 坚果墙 %d 切换外观: HP=%d/%d (%.1f%%), 图片=%s",
			entityID, health.CurrentHealth, health.MaxHealth, healthPercent*100, targetBodyImagePath)
	}
}

// isPlantBeingEaten 检查指定格子的植物是否正在被僵尸啃食
func (s *BehaviorSystem) isPlantBeingEaten(row, col int) bool {
	// 查询所有啃食状态的僵尸
	zombieEntities := ecs.GetEntitiesWith2[*components.BehaviorComponent, *components.PositionComponent](s.entityManager)

	for _, zombieID := range zombieEntities {
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
		if !ok || behavior.Type != components.BehaviorZombieEating {
			continue
		}

		// 获取僵尸位置，计算所在格子
		pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
		if !ok {
			continue
		}

		// 获取碰撞组件，用于计算碰撞盒中心
		collision, hasCollisionComp := ecs.GetComponent[*components.CollisionComponent](s.entityManager, zombieID)
		collisionOffsetX := 0.0
		if hasCollisionComp {
			collisionOffsetX = collision.OffsetX
		}

		// 计算僵尸碰撞盒中心所在格子
		zombieCol := int((pos.X + collisionOffsetX - config.GridWorldStartX) / config.CellWidth)
		zombieRow := int((pos.Y - config.GridWorldStartY - config.ZombieVerticalOffset - config.CellHeight/2.0) / config.CellHeight)

		if zombieRow == row && zombieCol == col {
			return true
		}
	}
	return false
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

// updateSunflowerGlowEffects 更新所有向日葵脸部发光效果
// 亮起阶段：每帧增加发光强度，直到达到最大值
// 衰减阶段：每帧降低发光强度，直到归零
// 当强度归零时，移除发光组件
func (s *BehaviorSystem) updateSunflowerGlowEffects(deltaTime float64) {
	// 查询所有拥有向日葵发光组件的实体
	entities := ecs.GetEntitiesWith1[*components.SunflowerGlowComponent](s.entityManager)

	for _, entityID := range entities {
		glowComp, ok := ecs.GetComponent[*components.SunflowerGlowComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		if glowComp.IsRising {
			// 亮起阶段：增加强度
			glowComp.Intensity += glowComp.RiseSpeed * deltaTime
			if glowComp.Intensity >= glowComp.MaxIntensity {
				glowComp.Intensity = glowComp.MaxIntensity
				glowComp.IsRising = false // 切换到衰减阶段
			}
		} else {
			// 衰减阶段：降低强度
			glowComp.Intensity -= glowComp.FadeSpeed * deltaTime

			// 如果强度归零，移除组件
			if glowComp.Intensity <= 0 {
				ecs.RemoveComponent[*components.SunflowerGlowComponent](s.entityManager, entityID)
			}
		}
	}
}

// updateWallnutHitGlowEffects 更新所有坚果墙被啃食发光效果
// 每帧降低发光强度，实现一闪一闪的效果
// 当强度归零时，移除发光组件
func (s *BehaviorSystem) updateWallnutHitGlowEffects(deltaTime float64) {
	// 查询所有拥有坚果墙发光组件的实体
	glowEntities := ecs.GetEntitiesWith1[*components.WallnutHitGlowComponent](s.entityManager)

	for _, entityID := range glowEntities {
		glowComp, ok := ecs.GetComponent[*components.WallnutHitGlowComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 降低发光强度
		glowComp.Intensity -= glowComp.FadeSpeed * deltaTime

		// 如果强度归零，移除组件
		if glowComp.Intensity <= 0 {
			ecs.RemoveComponent[*components.WallnutHitGlowComponent](s.entityManager, entityID)
		}
	}
}

// handlePotatoMineBehavior 处理土豆地雷行为
// Story 8.9: 土豆地雷需要武装时间，武装完成后等待僵尸触发爆炸
//
// 行为阶段（根据 PotatoMine.reanim 动画定义）:
//  1. 武装阶段: 播放 anim_idle（埋在地下，只显示泥土），计时
//  2. 升起阶段: 播放 anim_rise，从地下升起
//  3. 待机阶段: 播放 armed_with_light（武装就绪+警告灯闪烁），等待僵尸触发
//     - 警告灯闪烁频率随僵尸距离动态变化，越近越快
//  4. 爆炸阶段: 播放 anim_mashed，造成范围伤害后删除
func (s *BehaviorSystem) handlePotatoMineBehavior(entityID ecs.EntityID, deltaTime float64) {
	// 获取植物组件
	plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 防止重复爆炸
	if plant.IsExploding {
		return
	}

	// 阶段 1: 武装阶段（未武装状态）
	if !plant.IsArmed {
		// 更新武装计时器
		plant.ArmingTimer += deltaTime

		// 检查是否武装完成
		if plant.ArmingTimer >= config.PotatoMineArmingTime {
			plant.IsArmed = true
			log.Printf("[BehaviorSystem] 土豆地雷 %d: 武装完成！播放 anim_rise 动画", entityID)

			// 播放升起动画（anim_rise，非循环动画）
			ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
				UnitID:        "potatomine",
				AnimationName: "anim_rise",
				Processed:     false,
			})
		}
		return
	}

	// 阶段 2: 升起阶段 - 检测升起动画是否播放完成
	reanim, hasReanim := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if hasReanim && reanim.IsFinished && len(reanim.CurrentAnimations) > 0 && reanim.CurrentAnimations[0] == "anim_rise" {
		log.Printf("[BehaviorSystem] 土豆地雷 %d: 升起完成！切换到 idle 待机", entityID)

		// 切换到待机动画（idle 组合包含 anim_armed + anim_light + anim_glow）
		ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
			UnitID:    "potatomine",
			ComboName: "idle",
			Processed: false,
		})

		// 初始化闪烁状态
		plant.WarningLightOn = false
		plant.WarningLightTimer = config.PotatoMineBlinkIntervalMax
		plant.WarningLightInterval = config.PotatoMineBlinkIntervalMax
		return
	}

	// 如果正在播放升起动画，等待完成
	if hasReanim && len(reanim.CurrentAnimations) > 0 && reanim.CurrentAnimations[0] == "anim_rise" {
		return
	}

	// 阶段 3: 待机阶段 - 更新警告灯闪烁速度并检测僵尸触发爆炸
	// 获取土豆地雷位置
	plantPos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 查询所有僵尸实体
	zombieEntities := ecs.GetEntitiesWith2[*components.BehaviorComponent, *components.PositionComponent](s.entityManager)

	// 计算同行僵尸中最近的距离
	plantRow := plant.GridRow
	nearestDistance := config.PotatoMineWarningDistanceMax + 100 // 初始化为一个很大的值

	// 检测是否有僵尸在土豆地雷的格子内，同时计算最近距离
	for _, zombieID := range zombieEntities {
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
		if !ok {
			continue
		}

		// 只检测移动中或啃食中的僵尸（排除死亡中的僵尸）
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

		// 计算僵尸所在行
		zombieRow := int((zombiePos.Y - config.GridWorldStartY - config.ZombieVerticalOffset) / config.CellHeight)

		// 只检测同一行的僵尸
		if zombieRow != plantRow {
			continue
		}

		// 获取僵尸碰撞盒（考虑偏移）
		collision, hasCollision := ecs.GetComponent[*components.CollisionComponent](s.entityManager, zombieID)
		collisionOffsetX := 0.0
		if hasCollision {
			collisionOffsetX = collision.OffsetX
		}

		// 计算僵尸碰撞盒中心
		zombieCenterX := zombiePos.X + collisionOffsetX

		// 计算距离（只考虑X轴距离）
		distance := zombieCenterX - plantPos.X
		if distance > 0 && distance < nearestDistance {
			nearestDistance = distance
		}

		// 计算僵尸碰撞盒中心所在列
		zombieCol := int((zombieCenterX - config.GridWorldStartX) / config.CellWidth)

		// 如果僵尸在同一格，触发爆炸
		if zombieCol == plant.GridCol {
			log.Printf("[BehaviorSystem] 土豆地雷 %d: 检测到僵尸 %d 触发！位置: (%.1f, %.1f), 格子: (%d, %d)",
				entityID, zombieID, zombiePos.X, zombiePos.Y, zombieCol, zombieRow)

			// 触发爆炸
			s.triggerPotatoMineExplosion(entityID, plantPos.X, plantPos.Y)
			return
		}
	}

	// 更新警告灯闪烁
	// 根据最近僵尸距离计算闪烁间隔（线性插值）
	var newInterval float64
	if nearestDistance <= config.PotatoMineWarningDistanceMin {
		// 僵尸很近，最快闪烁
		newInterval = config.PotatoMineBlinkIntervalMin
	} else if nearestDistance >= config.PotatoMineWarningDistanceMax {
		// 僵尸很远或没有僵尸，最慢闪烁
		newInterval = config.PotatoMineBlinkIntervalMax
	} else {
		// 线性插值计算间隔
		t := (config.PotatoMineWarningDistanceMax - nearestDistance) /
			(config.PotatoMineWarningDistanceMax - config.PotatoMineWarningDistanceMin)
		newInterval = config.PotatoMineBlinkIntervalMax -
			t*(config.PotatoMineBlinkIntervalMax-config.PotatoMineBlinkIntervalMin)
	}

	// 更新闪烁间隔
	plant.WarningLightInterval = newInterval

	// 更新闪烁计时器
	plant.WarningLightTimer -= deltaTime
	if plant.WarningLightTimer <= 0 {
		// 切换灯的亮灭状态
		plant.WarningLightOn = !plant.WarningLightOn
		plant.WarningLightTimer = newInterval

		log.Printf("[BehaviorSystem] 土豆地雷 %d: 警告灯切换 → %v (间隔=%.2fs)", entityID, plant.WarningLightOn, newInterval)

		// 使用 ImageOverrides 切换警告灯图片
		// anim_light 轨道使用 IMAGE_REANIM_POTATOMINE_LIGHT1（灰灯）或 LIGHT2（红灯）
		if hasReanim {
			if reanim.ImageOverrides == nil {
				reanim.ImageOverrides = make(map[string]*ebiten.Image)
			}

			if plant.WarningLightOn {
				// 红灯亮：将 LIGHT1 替换为 LIGHT2
				light2Img, err := s.resourceManager.LoadImage("assets/reanim/PotatoMine_light2.png")
				if err == nil && light2Img != nil {
					reanim.ImageOverrides["IMAGE_REANIM_POTATOMINE_LIGHT1"] = light2Img
					log.Printf("[BehaviorSystem] 土豆地雷 %d: 设置红灯图片覆盖", entityID)
				} else {
					log.Printf("[BehaviorSystem] 土豆地雷 %d: 加载红灯图片失败: %v", entityID, err)
				}
			} else {
				// 红灯灭：恢复为 LIGHT1（移除覆盖）
				delete(reanim.ImageOverrides, "IMAGE_REANIM_POTATOMINE_LIGHT1")
				log.Printf("[BehaviorSystem] 土豆地雷 %d: 移除红灯图片覆盖", entityID)
			}

			// 强制缓存失效，确保下一帧重新渲染
			reanim.LastRenderFrame = -1
		}
	}
}

// triggerPotatoMineExplosion 触发土豆地雷爆炸
// Story 8.9: 土豆地雷爆炸造成 1800 伤害，1x1 格范围
func (s *BehaviorSystem) triggerPotatoMineExplosion(entityID ecs.EntityID, explosionX, explosionY float64) {
	log.Printf("[BehaviorSystem] 土豆地雷 %d: 开始爆炸！位置: (%.1f, %.1f)", entityID, explosionX, explosionY)

	// 获取植物组件并标记为正在爆炸（防止重复触发）
	plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID)
	if !ok {
		return
	}
	plant.IsExploding = true

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
