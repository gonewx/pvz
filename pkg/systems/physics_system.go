package systems

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
)

// PhysicsSystem 处理游戏物理逻辑
// 主要负责碰撞检测（子弹与僵尸的碰撞）
type PhysicsSystem struct {
	em *ecs.EntityManager
	rm *game.ResourceManager
}

// NewPhysicsSystem 创建物理系统
//
// 参数:
//   - em: 实体管理器，用于查询和操作实体组件
//   - rm: 资源管理器，用于创建击中效果时加载资源
//
// 返回:
//   - *PhysicsSystem: 物理系统实例
func NewPhysicsSystem(em *ecs.EntityManager, rm *game.ResourceManager) *PhysicsSystem {
	return &PhysicsSystem{
		em: em,
		rm: rm,
	}
}

// checkAABBCollision 检查两个实体的AABB（轴对齐边界框）是否发生碰撞
// 碰撞盒中心 = 实体位置 + 碰撞组件偏移量
//
// 参数:
//   - pos1: 第一个实体的位置组件
//   - col1: 第一个实体的碰撞组件
//   - pos2: 第二个实体的位置组件
//   - col2: 第二个实体的碰撞组件
//
// 返回:
//   - bool: 如果两个碰撞盒重叠返回 true，否则返回 false
func (ps *PhysicsSystem) checkAABBCollision(
	pos1 *components.PositionComponent, col1 *components.CollisionComponent,
	pos2 *components.PositionComponent, col2 *components.CollisionComponent) bool {

	// 计算第一个碰撞盒的中心（应用偏移量）
	center1X := pos1.X + col1.OffsetX
	center1Y := pos1.Y + col1.OffsetY

	// 计算第一个碰撞盒的边界
	left1 := center1X - col1.Width/2
	right1 := center1X + col1.Width/2
	top1 := center1Y - col1.Height/2
	bottom1 := center1Y + col1.Height/2

	// 计算第二个碰撞盒的中心（应用偏移量）
	center2X := pos2.X + col2.OffsetX
	center2Y := pos2.Y + col2.OffsetY

	// 计算第二个碰撞盒的边界
	left2 := center2X - col2.Width/2
	right2 := center2X + col2.Width/2
	top2 := center2Y - col2.Height/2
	bottom2 := center2Y + col2.Height/2

	// AABB碰撞检测：检查两个矩形是否重叠
	// 如果任一轴上没有重叠，则没有碰撞
	return right1 >= left2 &&
		left1 <= right2 &&
		bottom1 >= top2 &&
		top1 <= bottom2
}

// Update 更新物理系统，处理碰撞检测
// 检测所有豌豆子弹与僵尸的碰撞
//
// 参数:
//   - deltaTime: 自上一帧以来经过的时间（秒），本系统暂不使用
func (ps *PhysicsSystem) Update(deltaTime float64) {
	// 检查游戏是否冻结（僵尸获胜流程期间）
	// Story 8.8: 游戏冻结时，删除所有子弹实体
	freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](ps.em)
	if len(freezeEntities) > 0 {
		// 删除所有子弹实体
		bulletEntities := ecs.GetEntitiesWith1[*components.VelocityComponent](ps.em)
		for _, bulletID := range bulletEntities {
			behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](ps.em, bulletID)
			if ok && behaviorComp.Type == components.BehaviorPeaProjectile {
				ps.em.DestroyEntity(bulletID)
			}
		}
		return
	}

	// 查询所有拥有必要组件的实体（子弹和僵尸都需要这些组件）
	allEntities := ecs.GetEntitiesWith3[
		*components.BehaviorComponent,
		*components.PositionComponent,
		*components.CollisionComponent,
	](ps.em)

	// 分离子弹和僵尸
	bullets := make([]ecs.EntityID, 0)
	zombies := make([]ecs.EntityID, 0)

	for _, entityID := range allEntities {
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](ps.em, entityID)
		if !ok {
			continue
		}
		// 泛型 API 已提供类型安全

		if behavior.Type == components.BehaviorPeaProjectile {
			bullets = append(bullets, entityID)
		} else if behavior.Type == components.BehaviorZombieBasic ||
			behavior.Type == components.BehaviorZombieEating ||
			behavior.Type == components.BehaviorZombieDying || // 死亡动画期间仍然检测碰撞
			behavior.Type == components.BehaviorZombieConehead ||
			behavior.Type == components.BehaviorZombieBuckethead ||
			behavior.Type == components.BehaviorZombieFlag {
			// 包括移动中的僵尸、啃食中的僵尸、死亡中的僵尸（普通、路障、铁桶、旗帜）
			// 死亡中的僵尸仍然需要碰撞检测，以便子弹不会穿透尸体
			zombies = append(zombies, entityID)
		}
	}

	// 嵌套遍历检测碰撞
	for _, bulletID := range bullets {
		// 获取子弹的位置和碰撞组件
		bulletPos, ok := ecs.GetComponent[*components.PositionComponent](ps.em, bulletID)
		if !ok {
			continue
		}
		// 泛型 API 已提供类型安全

		bulletCol, ok := ecs.GetComponent[*components.CollisionComponent](ps.em, bulletID)
		if !ok {
			continue
		}
		// 泛型 API 已提供类型安全

		// 找出与子弹碰撞的所有僵尸中 X 坐标最小（最靠前）的那个
		// 这样确保在同一行多个僵尸位置接近时，只有最前面的僵尸被击中
		var hitZombieID ecs.EntityID
		var hitZombieX float64 = 1e9 // 初始化为一个很大的值

		// 检查子弹与所有僵尸的碰撞，找出 X 最小的碰撞目标
		for _, zombieID := range zombies {
			// 获取僵尸的位置和碰撞组件
			zombiePos, ok := ecs.GetComponent[*components.PositionComponent](ps.em, zombieID)
			if !ok {
				continue
			}

			zombieCol, ok := ecs.GetComponent[*components.CollisionComponent](ps.em, zombieID)
			if !ok {
				continue
			}

			// 执行AABB碰撞检测
			if ps.checkAABBCollision(bulletPos, bulletCol, zombiePos, zombieCol) {
				// 碰撞发生！记录这个僵尸，但只选择 X 最小的
				if zombiePos.X < hitZombieX {
					hitZombieID = zombieID
					hitZombieX = zombiePos.X
				}
			}
		}

		// 如果找到了碰撞的僵尸，处理碰撞
		if hitZombieID != 0 {
			zombieID := hitZombieID

			// 1. 创建击中效果实体（在子弹位置）
			_, err := entities.NewPeaBulletHitEffect(ps.em, ps.rm, bulletPos.X, bulletPos.Y)
			if err != nil {
				// 如果创建击中效果失败，记录错误但继续处理碰撞
				// 在实际项目中可以使用日志系统记录错误
				// 这里为了简化，忽略错误
			}

			// 触发豌豆击中溅射粒子效果
			// 获取子弹的 BehaviorComponent 以确定子弹类型
			bulletBehavior, ok := ecs.GetComponent[*components.BehaviorComponent](ps.em, bulletID)
			if ok {

				// 根据子弹类型选择粒子效果
				var particleEffectName string
				if bulletBehavior.Type == components.BehaviorPeaProjectile {
					particleEffectName = "PeaSplat" // 豌豆溅射效果
				}
				// 未来扩展: 卷心菜子弹类型
				// else if bulletBehavior.Type == components.BehaviorCabbageProjectile {
				//     particleEffectName = "CabbageSplat"
				// }

				// 触发粒子效果
				if particleEffectName != "" {
					_, err := entities.CreateParticleEffect(
						ps.em,
						ps.rm,
						particleEffectName,
						bulletPos.X, bulletPos.Y,
					)
					if err != nil {
						log.Printf("[PhysicsSystem] 警告：创建击中粒子效果失败: %v", err)
						// 不阻塞游戏逻辑，游戏继续运行
					} else {
						log.Printf("[PhysicsSystem] 子弹 %d 击中僵尸 %d，触发粒子效果 '%s'，位置: (%.1f, %.1f)",
							bulletID, zombieID, particleEffectName, bulletPos.X, bulletPos.Y)
					}
				}
			}

			// 2. 处理护甲伤害（优先扣除护甲值）
			armor, hasArmor := ecs.GetComponent[*components.ArmorComponent](ps.em, zombieID)
			if hasArmor {
				if armor.CurrentArmor > 0 {
					// 有护甲且护甲未破坏，优先扣除护甲
					armor.CurrentArmor -= config.PeaBulletDamage
					// 播放击中护甲音效（根据僵尸类型选择不同音效）
					ps.playArmorHitSound(zombieID)
					// 方案A+：护甲受击也添加闪烁效果
					ps.addFlashEffect(zombieID)
					// 注意：护甲可以降到负数，BehaviorSystem 会检查 <= 0 的情况并处理护甲破坏
				} else {
					// 护甲已破坏，扣除身体生命值
					zombieHealth, ok := ecs.GetComponent[*components.HealthComponent](ps.em, zombieID)
					if ok {
						zombieHealth.CurrentHealth -= config.PeaBulletDamage

						// 方案A+：添加受击闪烁效果
						ps.addFlashEffect(zombieID)
					}
					// 播放击中身体音效
					ps.playHitSound()
				}
			} else {
				// 3. 没有护甲，直接减少僵尸生命值（伤害计算）
				zombieHealth, ok := ecs.GetComponent[*components.HealthComponent](ps.em, zombieID)
				if ok {
					zombieHealth.CurrentHealth -= config.PeaBulletDamage
					// 注意：生命值可以降到负数，BehaviorSystem 会检查 <= 0 的情况

					// 方案A+：添加受击闪烁效果
					ps.addFlashEffect(zombieID)
				}
				// 播放击中身体音效
				ps.playHitSound()
			}

			// 4. 标记子弹实体待删除
			ps.em.DestroyEntity(bulletID)
		}
	}
}

// playHitSound 播放子弹击中僵尸的音效
// 使用 AudioManager 统一管理音效（Story 10.9）
func (ps *PhysicsSystem) playHitSound() {
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_SPLAT")
	}
}

// playArmorHitSound 播放子弹击中护甲的音效
// 根据护甲材质类型选择不同的音效：
// - 塑料护甲（路障）：使用 SOUND_PLASTICHIT - 塑料路障音效
// - 金属护甲（铁桶）：使用 SOUND_SHIELDHIT - 金属铁桶音效
// Story 10.9: 护甲击中音效差异化
func (ps *PhysicsSystem) playArmorHitSound(zombieID ecs.EntityID) {
	// 获取僵尸的护甲组件，确定护甲材质类型
	armor, ok := ecs.GetComponent[*components.ArmorComponent](ps.em, zombieID)
	if !ok {
		return
	}

	// 使用 AudioManager 统一管理音效（Story 10.9）
	audioManager := game.GetGameState().GetAudioManager()
	if audioManager == nil {
		return
	}

	// 根据护甲材质类型选择音效
	switch armor.Type {
	case components.ArmorTypePlastic:
		// 塑料护甲（路障）使用塑料音效
		audioManager.PlaySound("SOUND_PLASTICHIT")
	case components.ArmorTypeMetal:
		// 金属护甲（铁桶）使用金属音效
		audioManager.PlaySound("SOUND_SHIELDHIT")
	default:
		// 默认使用塑料音效（兜底）
		audioManager.PlaySound("SOUND_PLASTICHIT")
	}
}

// addFlashEffect 为僵尸添加受击闪烁效果（方案A+）
// 参数：
//   - zombieID: 僵尸实体ID
func (ps *PhysicsSystem) addFlashEffect(zombieID ecs.EntityID) {
	// 检查是否已有闪烁组件
	flashComp, hasFlash := ecs.GetComponent[*components.FlashEffectComponent](ps.em, zombieID)

	if hasFlash {
		// 已有闪烁组件，重置时间（连续受击时延长闪烁）
		flashComp.Elapsed = 0
		flashComp.IsActive = true
	} else {
		// 没有闪烁组件，创建新的
		ecs.AddComponent(ps.em, zombieID, &components.FlashEffectComponent{
			Duration:  0.1,  // 闪烁持续0.1秒（原版默认值）
			Elapsed:   0,    // 从0开始计时
			Intensity: 0.8,  // 闪烁强度80%（白色叠加）
			IsActive:  true, // 激活状态
		})
	}
}
