package behavior

import (
	"log"
	"math/rand"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
)

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
