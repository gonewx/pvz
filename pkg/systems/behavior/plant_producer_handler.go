package behavior

import (
	"log"
	"math/rand"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
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
