package systems

import (
	"log"
	"reflect"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
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
// 假设碰撞盒中心对齐实体位置
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

	// 计算第一个碰撞盒的边界（中心对齐）
	left1 := pos1.X - col1.Width/2
	right1 := pos1.X + col1.Width/2
	top1 := pos1.Y - col1.Height/2
	bottom1 := pos1.Y + col1.Height/2

	// 计算第二个碰撞盒的边界（中心对齐）
	left2 := pos2.X - col2.Width/2
	right2 := pos2.X + col2.Width/2
	top2 := pos2.Y - col2.Height/2
	bottom2 := pos2.Y + col2.Height/2

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
	// 查询所有拥有必要组件的实体（子弹和僵尸都需要这些组件）
	allEntities := ps.em.GetEntitiesWith(
		reflect.TypeOf(&components.BehaviorComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.CollisionComponent{}),
	)

	// 分离子弹和僵尸
	bullets := make([]ecs.EntityID, 0)
	zombies := make([]ecs.EntityID, 0)

	for _, entityID := range allEntities {
		behaviorComp, ok := ps.em.GetComponent(entityID, reflect.TypeOf(&components.BehaviorComponent{}))
		if !ok {
			continue
		}
		behavior := behaviorComp.(*components.BehaviorComponent)

		if behavior.Type == components.BehaviorPeaProjectile {
			bullets = append(bullets, entityID)
		} else if behavior.Type == components.BehaviorZombieBasic ||
			behavior.Type == components.BehaviorZombieEating ||
			behavior.Type == components.BehaviorZombieConehead ||
			behavior.Type == components.BehaviorZombieBuckethead {
			// 包括移动中的僵尸和啃食中的僵尸（普通、路障、铁桶）
			zombies = append(zombies, entityID)
		}
	}

	// 嵌套遍历检测碰撞
	for _, bulletID := range bullets {
		// 获取子弹的位置和碰撞组件
		bulletPosComp, ok := ps.em.GetComponent(bulletID, reflect.TypeOf(&components.PositionComponent{}))
		if !ok {
			continue
		}
		bulletPos := bulletPosComp.(*components.PositionComponent)

		bulletColComp, ok := ps.em.GetComponent(bulletID, reflect.TypeOf(&components.CollisionComponent{}))
		if !ok {
			continue
		}
		bulletCol := bulletColComp.(*components.CollisionComponent)

		// 检查子弹与所有僵尸的碰撞
		for _, zombieID := range zombies {
			// 获取僵尸的位置和碰撞组件
			zombiePosComp, ok := ps.em.GetComponent(zombieID, reflect.TypeOf(&components.PositionComponent{}))
			if !ok {
				continue
			}
			zombiePos := zombiePosComp.(*components.PositionComponent)

			zombieColComp, ok := ps.em.GetComponent(zombieID, reflect.TypeOf(&components.CollisionComponent{}))
			if !ok {
				continue
			}
			zombieCol := zombieColComp.(*components.CollisionComponent)

			// 执行AABB碰撞检测
			if ps.checkAABBCollision(bulletPos, bulletCol, zombiePos, zombieCol) {
				// 碰撞发生！
				// 1. 创建击中效果实体（在子弹位置）
				_, err := entities.NewPeaBulletHitEffect(ps.em, ps.rm, bulletPos.X, bulletPos.Y)
				if err != nil {
					// 如果创建击中效果失败，记录错误但继续处理碰撞
					// 在实际项目中可以使用日志系统记录错误
					// 这里为了简化，忽略错误
				}

				// Story 7.4: 触发豌豆击中溅射粒子效果
				// 获取子弹的 BehaviorComponent 以确定子弹类型
				bulletBehaviorComp, ok := ps.em.GetComponent(bulletID, reflect.TypeOf(&components.BehaviorComponent{}))
				if ok {
					bulletBehavior := bulletBehaviorComp.(*components.BehaviorComponent)

					// 根据子弹类型选择粒子效果
					var particleEffectName string
					if bulletBehavior.Type == components.BehaviorPeaProjectile {
						particleEffectName = "PeaSplat" // 豌豆溅射效果
					}
					// TODO: 添加卷心菜子弹类型判断（当实现卷心菜投手后）
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

				// 2. 处理护甲伤害（Story 5.3: 优先扣除护甲值）
				armorComp, hasArmor := ps.em.GetComponent(zombieID, reflect.TypeOf(&components.ArmorComponent{}))
				if hasArmor {
					armor := armorComp.(*components.ArmorComponent)
					if armor.CurrentArmor > 0 {
						// 有护甲且护甲未破坏，优先扣除护甲
						armor.CurrentArmor -= config.PeaBulletDamage
						// 播放击中护甲音效
						ps.playArmorHitSound()
						// 注意：护甲可以降到负数，BehaviorSystem 会检查 <= 0 的情况并处理护甲破坏
					} else {
						// 护甲已破坏，扣除身体生命值
						zombieHealthComp, ok := ps.em.GetComponent(zombieID, reflect.TypeOf(&components.HealthComponent{}))
						if ok {
							zombieHealth := zombieHealthComp.(*components.HealthComponent)
							zombieHealth.CurrentHealth -= config.PeaBulletDamage
						}
						// 播放击中身体音效
						ps.playHitSound()
					}
				} else {
					// 3. 没有护甲，直接减少僵尸生命值（Story 4.4: 伤害计算）
					zombieHealthComp, ok := ps.em.GetComponent(zombieID, reflect.TypeOf(&components.HealthComponent{}))
					if ok {
						zombieHealth := zombieHealthComp.(*components.HealthComponent)
						zombieHealth.CurrentHealth -= config.PeaBulletDamage
						// 注意：生命值可以降到负数，BehaviorSystem 会检查 <= 0 的情况
					}
					// 播放击中身体音效
					ps.playHitSound()
				}

				// 4. 标记子弹实体待删除
				ps.em.DestroyEntity(bulletID)

				// 5. 一个子弹只能击中一个僵尸，跳出内层循环
				break
			}
		}
	}
}

// playHitSound 播放子弹击中僵尸的音效
// 使用配置文件中定义的音效（config.ZombieHitSoundPath）
// 注意：音效每次都会重新开始播放，不会等待上一次播放结束
func (ps *PhysicsSystem) playHitSound() {
	// 加载击中音效（如果已加载，会返回缓存的播放器）
	// 音效路径在 pkg/config/unit_config.go 中配置，可根据需要切换测试
	hitSound, err := ps.rm.LoadSoundEffect(config.ZombieHitSoundPath)
	if err != nil {
		// 音效加载失败时不阻止游戏继续运行
		// 在实际项目中可以使用日志系统记录错误
		return
	}

	// 重置播放器位置到开头（允许快速连续播放）
	hitSound.Rewind()

	// 播放音效
	hitSound.Play()
}

// playArmorHitSound 播放子弹击中护甲的音效
// 使用配置文件中定义的音效（config.ArmorBreakSoundPath）
// 用于路障僵尸和铁桶僵尸受到攻击时
func (ps *PhysicsSystem) playArmorHitSound() {
	// 加载击中护甲音效（如果已加载，会返回缓存的播放器）
	armorSound, err := ps.rm.LoadSoundEffect(config.ArmorBreakSoundPath)
	if err != nil {
		// 音效加载失败时不阻止游戏继续运行
		return
	}

	// 重置播放器位置到开头（允许快速连续播放）
	armorSound.Rewind()

	// 播放音效
	armorSound.Play()
}
