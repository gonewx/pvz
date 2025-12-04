package systems

import (
	"log"
	"math"
	"math/rand"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

// BowlingNutSystem 保龄球坚果滚动系统
// Story 19.6: 处理保龄球坚果的滚动移动和边界销毁
// Story 19.7: 处理碰撞检测、伤害处理和弹射逻辑
//
// 职责：
// - 更新坚果的水平位置
// - 检测边界并销毁越界坚果
// - 管理滚动音效播放
// - 检测与僵尸的碰撞并造成伤害
// - 计算弹射方向并执行弹射移动
type BowlingNutSystem struct {
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager

	// 滚动音效播放器映射（entityID -> player）
	soundPlayers map[ecs.EntityID]*audio.Player
}

// NewBowlingNutSystem 创建保龄球坚果滚动系统
//
// 参数:
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载音效）
//
// 返回:
//   - *BowlingNutSystem: 系统实例
func NewBowlingNutSystem(em *ecs.EntityManager, rm *game.ResourceManager) *BowlingNutSystem {
	return &BowlingNutSystem{
		entityManager:   em,
		resourceManager: rm,
		soundPlayers:    make(map[ecs.EntityID]*audio.Player),
	}
}

// Update 更新所有保龄球坚果的位置和碰撞检测
//
// 参数:
//   - dt: 帧间隔时间（秒）
//
// 处理逻辑：
// 1. 查询所有 BowlingNutComponent 实体
// 2. 更新碰撞冷却时间
// 3. 如果正在滚动，更新 X 位置
// 4. 如果正在弹射，更新 Y 位置
// 5. 检测碰撞并处理伤害和弹射
// 6. 检查是否超出边界，超出则销毁
// 7. 管理滚动音效
func (s *BowlingNutSystem) Update(dt float64) {
	// 查询所有保龄球坚果实体
	entities := ecs.GetEntitiesWith2[*components.BowlingNutComponent, *components.PositionComponent](s.entityManager)

	for _, entityID := range entities {
		nutComp, nutOk := ecs.GetComponent[*components.BowlingNutComponent](s.entityManager, entityID)
		posComp, posOk := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)

		if !nutOk || !posOk {
			continue
		}

		// 更新碰撞冷却时间
		if nutComp.CollisionCooldown > 0 {
			nutComp.CollisionCooldown -= dt
		}

		// 如果正在滚动，更新水平位置
		if nutComp.IsRolling {
			posComp.X += nutComp.VelocityX * dt

			// 开始播放滚动音效（如果还没播放）
			if !nutComp.SoundPlaying {
				s.startRollingSound(entityID)
				nutComp.SoundPlaying = true
			}
		}

		// 如果正在弹射，更新垂直位置
		if nutComp.IsBouncing {
			posComp.Y += nutComp.VelocityY * dt

			// 检测是否到达目标行
			targetY := s.calculateRowCenterY(nutComp.TargetRow)
			if (nutComp.VelocityY > 0 && posComp.Y >= targetY) ||
				(nutComp.VelocityY < 0 && posComp.Y <= targetY) {
				// 到达目标行
				posComp.Y = targetY
				nutComp.Row = nutComp.TargetRow
				nutComp.IsBouncing = false
				nutComp.VelocityY = 0
				log.Printf("[BowlingNutSystem] 坚果到达目标行: entityID=%d, row=%d", entityID, nutComp.Row)
			}
		}

		// 不在弹射中且冷却结束时检测碰撞
		if !nutComp.IsBouncing && nutComp.CollisionCooldown <= 0 {
			collidedZombie := s.checkCollisionWithNearestZombie(entityID, posComp, nutComp)
			if collidedZombie != 0 {
				// 处理碰撞后的行为
				if nutComp.IsExplosive {
					// Story 19.8: 爆炸坚果触发 3x3 范围爆炸（直接爆炸，不走普通碰撞伤害）
					log.Printf("[BowlingNutSystem] 爆炸坚果碰撞，触发爆炸: entityID=%d", entityID)
					s.triggerExplosion(entityID, posComp)
					continue
				} else {
					// 普通坚果：对碰撞的僵尸造成伤害，然后弹射
					s.applyDamageToZombie(collidedZombie)
					s.playImpactSound()
					targetRow := s.calculateBounceDirection(nutComp.Row, posComp.X)
					s.startBounce(entityID, nutComp, posComp, targetRow)
				}
			} else if nutComp.BounceDirection != 0 {
				// 没有碰撞僵尸，但有弹射方向 -> 继续弹射直到边缘
				s.continueBounce(entityID, nutComp, posComp)
			}
		}

		// 检查边界：X 坐标超过背景宽度时销毁
		if posComp.X > config.BackgroundWidth {
			log.Printf("[BowlingNutSystem] 坚果越界销毁: entityID=%d, X=%.1f", entityID, posComp.X)

			// 停止音效
			s.stopRollingSound(entityID)

			// 标记实体销毁
			s.entityManager.DestroyEntity(entityID)
		}
	}

	// 清理已销毁实体的音效播放器
	s.cleanupSoundPlayers()
}

// checkCollisionWithNearestZombie 检测坚果与最近僵尸的碰撞
// 每次碰撞只返回一只僵尸（X轴最近的那只）
//
// 参数:
//   - nutEntityID: 坚果实体ID
//   - nutPos: 坚果位置组件
//   - nutComp: 坚果组件
//
// 返回:
//   - ecs.EntityID: 碰撞的最近僵尸实体ID，无碰撞返回0
func (s *BowlingNutSystem) checkCollisionWithNearestZombie(
	nutEntityID ecs.EntityID,
	nutPos *components.PositionComponent,
	nutComp *components.BowlingNutComponent,
) ecs.EntityID {
	var nearestZombie ecs.EntityID
	nearestDist := math.MaxFloat64

	// 计算坚果碰撞盒
	nutLeft := nutPos.X - config.BowlingNutCollisionWidth/2
	nutRight := nutPos.X + config.BowlingNutCollisionWidth/2
	nutTop := nutPos.Y - config.BowlingNutCollisionHeight/2
	nutBottom := nutPos.Y + config.BowlingNutCollisionHeight/2

	// 查询所有僵尸实体
	zombieEntities := ecs.GetEntitiesWith3[
		*components.BehaviorComponent,
		*components.PositionComponent,
		*components.CollisionComponent,
	](s.entityManager)

	for _, zombieID := range zombieEntities {
		behavior, _ := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)

		// 检查是否是僵尸类型
		if !s.isZombieType(behavior.Type) {
			continue
		}

		zombiePos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
		zombieCol, _ := ecs.GetComponent[*components.CollisionComponent](s.entityManager, zombieID)

		// 计算僵尸碰撞盒
		zombieLeft := zombiePos.X + zombieCol.OffsetX - zombieCol.Width/2
		zombieRight := zombiePos.X + zombieCol.OffsetX + zombieCol.Width/2
		zombieTop := zombiePos.Y + zombieCol.OffsetY - zombieCol.Height/2
		zombieBottom := zombiePos.Y + zombieCol.OffsetY + zombieCol.Height/2

		// AABB 碰撞检测
		if nutRight >= zombieLeft && nutLeft <= zombieRight &&
			nutBottom >= zombieTop && nutTop <= zombieBottom {
			// 计算与坚果的X轴距离，选择最近的
			dist := math.Abs(zombiePos.X - nutPos.X)
			if dist < nearestDist {
				nearestDist = dist
				nearestZombie = zombieID
			}
		}
	}

	if nearestZombie != 0 {
		log.Printf("[BowlingNutSystem] 坚果碰撞最近僵尸: nutID=%d, zombieID=%d", nutEntityID, nearestZombie)
	}

	return nearestZombie
}

// isZombieType 检查行为类型是否是活着的僵尸（排除死亡状态）
func (s *BowlingNutSystem) isZombieType(behaviorType components.BehaviorType) bool {
	switch behaviorType {
	case components.BehaviorZombieBasic,
		components.BehaviorZombieEating,
		components.BehaviorZombieConehead,
		components.BehaviorZombieBuckethead,
		components.BehaviorZombieFlag:
		return true
	default:
		// 排除 BehaviorZombieDying, BehaviorZombieSquashing, BehaviorZombieDyingExplosion 等死亡状态
		return false
	}
}

// applyDamageToZombie 对僵尸造成碰撞伤害
//
// 参数:
//   - zombieID: 僵尸实体ID
//
// 处理逻辑（与樱桃炸弹不同）：
// - 有护甲：移除护甲（打掉帽子/桶），不造成身体伤害
// - 无护甲：秒杀僵尸（直接将血量设为0）
// - 添加闪烁效果
func (s *BowlingNutSystem) applyDamageToZombie(zombieID ecs.EntityID) {
	// 检查是否有护甲
	armor, hasArmor := ecs.GetComponent[*components.ArmorComponent](s.entityManager, zombieID)
	health, hasHealth := ecs.GetComponent[*components.HealthComponent](s.entityManager, zombieID)

	if hasArmor && armor.CurrentArmor > 0 {
		// 有护甲且护甲未破坏：移除护甲，不造成身体伤害
		log.Printf("[BowlingNutSystem] 僵尸护甲破坏: zombieID=%d, 原护甲=%d", zombieID, armor.CurrentArmor)
		armor.CurrentArmor = 0
	} else if hasHealth {
		// 没有护甲或护甲已破坏：秒杀僵尸
		log.Printf("[BowlingNutSystem] 僵尸被秒杀: zombieID=%d, 原血量=%d", zombieID, health.CurrentHealth)
		health.CurrentHealth = 0
	}

	// 添加闪烁效果
	s.addFlashEffect(zombieID)
}

// addFlashEffect 为僵尸添加受击闪烁效果
func (s *BowlingNutSystem) addFlashEffect(zombieID ecs.EntityID) {
	flashComp, hasFlash := ecs.GetComponent[*components.FlashEffectComponent](s.entityManager, zombieID)

	if hasFlash {
		// 已有闪烁组件，重置时间
		flashComp.Elapsed = 0
		flashComp.IsActive = true
	} else {
		// 没有闪烁组件，创建新的
		ecs.AddComponent(s.entityManager, zombieID, &components.FlashEffectComponent{
			Duration:  0.1,
			Elapsed:   0,
			Intensity: 0.8,
			IsActive:  true,
		})
	}
}

// triggerExplosion 触发爆炸坚果的 3x3 范围爆炸
// Story 19.8: 爆炸坚果碰撞时调用
//
// 参数:
//   - entityID: 爆炸坚果实体ID
//   - posComp: 爆炸坚果位置组件
//
// 处理逻辑:
//  1. 计算爆炸中心位置
//  2. 查询范围内所有僵尸
//  3. 对范围内僵尸造成 1800 伤害
//  4. 播放爆炸粒子特效
//  5. 播放爆炸音效
//  6. 停止滚动音效并销毁坚果实体
func (s *BowlingNutSystem) triggerExplosion(entityID ecs.EntityID, posComp *components.PositionComponent) {
	// 计算爆炸范围半径（像素）
	explosionRadius := config.ExplosiveNutExplosionRadius * config.CellWidth
	explosionRadiusSq := explosionRadius * explosionRadius

	log.Printf("[BowlingNutSystem] 爆炸坚果爆炸: entityID=%d, 位置=(%.1f, %.1f), 半径=%.1f像素",
		entityID, posComp.X, posComp.Y, explosionRadius)

	// 查询所有僵尸实体
	zombieEntities := ecs.GetEntitiesWith2[
		*components.BehaviorComponent,
		*components.PositionComponent,
	](s.entityManager)

	damageCount := 0
	for _, zombieID := range zombieEntities {
		behavior, _ := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)

		// 检查是否是僵尸类型
		if !s.isZombieType(behavior.Type) {
			continue
		}

		zombiePos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)

		// 计算距离（使用距离平方优化性能）
		// 修正：僵尸的 PositionComponent.Y 包含了 ZombieVerticalOffset (-25.0)
		// 这导致上行僵尸距离变远，下行僵尸距离变近
		// 为了保证上下行对称判定，需要还原到格子中心进行距离计算
		zombieEffectiveY := zombiePos.Y - config.ZombieVerticalOffset

		dx := zombiePos.X - posComp.X
		dy := zombieEffectiveY - posComp.Y
		distSq := dx*dx + dy*dy

		// 检查是否在爆炸范围内
		if distSq <= explosionRadiusSq {
			s.applyExplosionDamageToZombie(zombieID)
			damageCount++
			log.Printf("[BowlingNutSystem] 僵尸在爆炸范围内: zombieID=%d, 距离=%.1f像素",
				zombieID, math.Sqrt(distSq))
		}
	}

	log.Printf("[BowlingNutSystem] 爆炸造成伤害: %d 个僵尸", damageCount)

	// 播放爆炸粒子特效
	s.playExplosionParticle(posComp.X, posComp.Y)

	// 播放爆炸音效
	s.playExplosionSound()

	// 停止滚动音效
	s.stopRollingSound(entityID)

	// 销毁坚果实体
	s.entityManager.DestroyEntity(entityID)
}

// applyExplosionDamageToZombie 对僵尸造成爆炸伤害
// Story 19.8: 爆炸伤害 1800，与樱桃炸弹相同
//
// 参数:
//   - zombieID: 僵尸实体ID
//
// 处理逻辑：
// - 有护甲：优先扣除护甲，溢出伤害扣身体
// - 无护甲：直接扣除身体生命值
// - 如果僵尸被杀死，标记为爆炸死亡（触发烧焦动画）
// - 添加闪烁效果
func (s *BowlingNutSystem) applyExplosionDamageToZombie(zombieID ecs.EntityID) {
	damage := config.ExplosiveNutDamage

	// 检查是否有护甲
	armor, hasArmor := ecs.GetComponent[*components.ArmorComponent](s.entityManager, zombieID)
	health, hasHealth := ecs.GetComponent[*components.HealthComponent](s.entityManager, zombieID)

	if hasArmor && armor.CurrentArmor > 0 {
		// 有护甲且护甲未破坏
		overflowDamage := damage - armor.CurrentArmor
		armor.CurrentArmor = 0 // 护甲完全破坏

		// 溢出伤害扣除身体生命值
		if overflowDamage > 0 && hasHealth {
			health.CurrentHealth -= overflowDamage
			// 如果被杀死，标记为爆炸死亡
			if health.CurrentHealth <= 0 {
				health.KilledByExplosion = true
			}
		}
		log.Printf("[BowlingNutSystem] 爆炸伤害破坏护甲: zombieID=%d, 溢出伤害=%d", zombieID, overflowDamage)
	} else if hasHealth {
		// 没有护甲或护甲已破坏，直接扣除身体生命值
		health.CurrentHealth -= damage
		// 如果被杀死，标记为爆炸死亡
		if health.CurrentHealth <= 0 {
			health.KilledByExplosion = true
		}
		log.Printf("[BowlingNutSystem] 爆炸伤害: zombieID=%d, 伤害=%d, 剩余血量=%d",
			zombieID, damage, health.CurrentHealth)
	}

	// 添加闪烁效果
	s.addFlashEffect(zombieID)
}

// playExplosionParticle 播放爆炸粒子特效
// Story 19.8: 使用 Powie.xml 粒子配置
//
// 参数:
//   - x, y: 爆炸位置（世界坐标）
func (s *BowlingNutSystem) playExplosionParticle(x, y float64) {
	if s.resourceManager == nil {
		return
	}

	_, err := entities.CreateParticleEffect(
		s.entityManager,
		s.resourceManager,
		config.ExplosiveNutParticleEffect, // "Powie"
		x, y,
	)
	if err != nil {
		log.Printf("[BowlingNutSystem] 创建爆炸粒子特效失败: %v", err)
	} else {
		log.Printf("[BowlingNutSystem] 爆炸粒子特效创建成功: 位置=(%.1f, %.1f)", x, y)
	}
}

// playExplosionSound 播放爆炸音效
// Story 19.8: 使用 explosion.ogg 音效
func (s *BowlingNutSystem) playExplosionSound() {
	if s.resourceManager == nil {
		return
	}

	player, err := s.resourceManager.LoadSoundEffect(config.ExplosiveNutExplosionSoundPath)
	if err != nil {
		log.Printf("[BowlingNutSystem] 加载爆炸音效失败: %v", err)
		return
	}
	player.Rewind()
	player.Play()
	log.Printf("[BowlingNutSystem] 爆炸音效播放")
}

// findNearestZombieDistance 查找指定行最近僵尸的 X 轴距离
//
// 参数:
//   - row: 要查找的行号
//   - nutX: 坚果的 X 坐标
//
// 返回:
//   - float64: 最近僵尸的 X 轴距离，无僵尸返回 math.MaxFloat64
func (s *BowlingNutSystem) findNearestZombieDistance(row int, nutX float64) float64 {
	minDist := math.MaxFloat64

	zombieEntities := ecs.GetEntitiesWith2[
		*components.BehaviorComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, zombieID := range zombieEntities {
		behavior, _ := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)

		if !s.isZombieType(behavior.Type) {
			continue
		}

		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
		zombieRow := s.calculateRowFromY(pos.Y)

		if zombieRow == row {
			dist := math.Abs(pos.X - nutX)
			if dist < minDist {
				minDist = dist
			}
		}
	}

	return minDist
}

// calculateRowFromY 从 Y 坐标计算行号
func (s *BowlingNutSystem) calculateRowFromY(y float64) int {
	row := int((y - config.GridWorldStartY) / config.CellHeight)
	if row < 0 {
		row = 0
	}
	if row > 4 {
		row = 4
	}
	return row
}

// calculateRowCenterY 计算行中心 Y 坐标
func (s *BowlingNutSystem) calculateRowCenterY(row int) float64 {
	return config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2
}

// calculateBounceDirection 计算弹射目标行
//
// 参数:
//   - currentRow: 当前行号
//   - nutX: 坚果的 X 坐标
//
// 返回:
//   - int: 目标行号 (0-4)
//
// 逻辑：
// 1. 边缘行处理：第 0 行只能向下，第 4 行只能向上
// 2. 计算上下行最近僵尸的 X 轴距离
// 3. 优先弹向 X 轴距离最近的僵尸所在行
// 4. 距离相等或都没有僵尸：随机选择
func (s *BowlingNutSystem) calculateBounceDirection(currentRow int, nutX float64) int {
	upRow := currentRow - 1
	downRow := currentRow + 1

	// 边缘行处理
	if currentRow == 0 {
		return downRow // 顶部边缘只能向下
	}
	if currentRow == 4 {
		return upRow // 底部边缘只能向上
	}

	// 计算上下行最近僵尸的 X 轴距离
	distUp := s.findNearestZombieDistance(upRow, nutX)
	distDown := s.findNearestZombieDistance(downRow, nutX)

	// 优先弹向 X 轴距离最近的僵尸所在行
	if distUp < distDown {
		return upRow
	}
	if distDown < distUp {
		return downRow
	}

	// 距离相等或都没有僵尸：随机选择
	if rand.Float32() < 0.5 {
		return upRow
	}
	return downRow
}

// startBounce 开始弹射
//
// 参数:
//   - entityID: 坚果实体ID
//   - nutComp: 坚果组件
//   - posComp: 位置组件
//   - targetRow: 目标行号
func (s *BowlingNutSystem) startBounce(
	entityID ecs.EntityID,
	nutComp *components.BowlingNutComponent,
	posComp *components.PositionComponent,
	targetRow int,
) {
	nutComp.IsBouncing = true
	nutComp.TargetRow = targetRow
	nutComp.BounceCount++
	nutComp.CollisionCooldown = config.BowlingNutCollisionCooldown

	// 计算垂直速度方向并记录弹射方向
	targetY := s.calculateRowCenterY(targetRow)
	if targetY > posComp.Y {
		nutComp.VelocityY = config.BowlingNutBounceSpeed // 向下
		nutComp.BounceDirection = 1                      // 记录向下弹射
	} else {
		nutComp.VelocityY = -config.BowlingNutBounceSpeed // 向上
		nutComp.BounceDirection = -1                      // 记录向上弹射
	}

	log.Printf("[BowlingNutSystem] 开始弹射: entityID=%d, fromRow=%d, toRow=%d, bounceCount=%d, direction=%d",
		entityID, nutComp.Row, targetRow, nutComp.BounceCount, nutComp.BounceDirection)
}

// continueBounce 持续弹射直到边缘
// 当坚果到达目标行后没有碰到僵尸时调用
//
// 逻辑：
// - 如果当前在边缘行（0或4），反转弹射方向并弹向相邻行
// - 如果在中间行（1-3），继续向同一方向弹射
//
// 参数:
//   - entityID: 坚果实体ID
//   - nutComp: 坚果组件
//   - posComp: 位置组件
func (s *BowlingNutSystem) continueBounce(
	entityID ecs.EntityID,
	nutComp *components.BowlingNutComponent,
	posComp *components.PositionComponent,
) {
	var targetRow int

	if nutComp.Row == 0 {
		// 到达顶边，反转方向向下弹射
		nutComp.BounceDirection = 1
		targetRow = 1
		log.Printf("[BowlingNutSystem] 到达顶边，反弹向下: entityID=%d", entityID)
	} else if nutComp.Row == 4 {
		// 到达底边，反转方向向上弹射
		nutComp.BounceDirection = -1
		targetRow = 3
		log.Printf("[BowlingNutSystem] 到达底边，反弹向上: entityID=%d", entityID)
	} else {
		// 中间行，继续向同一方向弹射
		targetRow = nutComp.Row + nutComp.BounceDirection
		log.Printf("[BowlingNutSystem] 继续弹射: entityID=%d, row=%d, direction=%d, targetRow=%d",
			entityID, nutComp.Row, nutComp.BounceDirection, targetRow)
	}

	// 开始弹射到目标行
	nutComp.IsBouncing = true
	nutComp.TargetRow = targetRow
	nutComp.CollisionCooldown = config.BowlingNutCollisionCooldown

	// 计算垂直速度
	targetY := s.calculateRowCenterY(targetRow)
	if targetY > posComp.Y {
		nutComp.VelocityY = config.BowlingNutBounceSpeed // 向下
	} else {
		nutComp.VelocityY = -config.BowlingNutBounceSpeed // 向上
	}
}

// playImpactSound 播放撞击音效
func (s *BowlingNutSystem) playImpactSound() {
	if s.resourceManager == nil {
		return
	}

	// 随机选择音效
	var soundPath string
	if rand.Float32() < 0.5 {
		soundPath = config.BowlingImpactSoundPath
	} else {
		soundPath = config.BowlingImpact2SoundPath
	}

	player, err := s.resourceManager.LoadSoundEffect(soundPath)
	if err != nil {
		log.Printf("[BowlingNutSystem] 加载撞击音效失败: %v", err)
		return
	}
	player.Rewind()
	player.Play()
}

// startRollingSound 开始播放滚动音效
func (s *BowlingNutSystem) startRollingSound(entityID ecs.EntityID) {
	if s.resourceManager == nil {
		return
	}

	// 检查是否已有播放器
	if _, exists := s.soundPlayers[entityID]; exists {
		return
	}

	// 加载音效
	soundPath := config.BowlingRollSoundPath
	player, err := s.resourceManager.LoadSoundEffect(soundPath)
	if err != nil {
		log.Printf("[BowlingNutSystem] 加载滚动音效失败: %v", err)
		return
	}

	// 设置循环播放
	// 注意：Ebitengine 的 audio.Player 不直接支持循环
	// 需要在音效结束时重新播放
	player.Rewind()
	player.Play()

	s.soundPlayers[entityID] = player
	log.Printf("[BowlingNutSystem] 开始播放滚动音效: entityID=%d", entityID)
}

// stopRollingSound 停止滚动音效
func (s *BowlingNutSystem) stopRollingSound(entityID ecs.EntityID) {
	if player, exists := s.soundPlayers[entityID]; exists {
		if player != nil {
			player.Pause()
		}
		delete(s.soundPlayers, entityID)
		log.Printf("[BowlingNutSystem] 停止滚动音效: entityID=%d", entityID)
	}
}

// cleanupSoundPlayers 清理已销毁实体的音效播放器
func (s *BowlingNutSystem) cleanupSoundPlayers() {
	for entityID, player := range s.soundPlayers {
		// 检查实体是否还存在
		if _, ok := ecs.GetComponent[*components.BowlingNutComponent](s.entityManager, entityID); !ok {
			if player != nil {
				player.Pause()
			}
			delete(s.soundPlayers, entityID)
		}
	}
}

// StopAllSounds 停止所有滚动音效
// 在场景切换或游戏结束时调用
func (s *BowlingNutSystem) StopAllSounds() {
	for entityID, player := range s.soundPlayers {
		if player != nil {
			player.Pause()
		}
		delete(s.soundPlayers, entityID)
	}
	log.Printf("[BowlingNutSystem] 停止所有滚动音效")
}
