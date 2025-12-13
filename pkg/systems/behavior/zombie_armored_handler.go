package behavior

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/hajimehoshi/ebiten/v2"
)

// handleConeheadZombieBehavior 处理路障僵尸的行为逻辑
// 路障僵尸在护甲未破坏时具有额外的护甲，护甲破坏后退化为普通僵尸
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

// handleBucketheadZombieBehavior 处理铁桶僵尸的行为逻辑
// 铁桶僵尸在护甲未破坏时具有额外的护甲，护甲破坏后退化为普通僵尸
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
