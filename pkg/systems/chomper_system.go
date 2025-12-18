package systems

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/utils"
)

// ChomperSystem 大嘴花系统
// Story 8.11: 处理大嘴花的目标检测、吞噬/撕咬攻击和消化逻辑
type ChomperSystem struct {
	entityManager *ecs.EntityManager
	gameState     *game.GameState
}

// NewChomperSystem 创建新的大嘴花系统
func NewChomperSystem(em *ecs.EntityManager, gs *game.GameState) *ChomperSystem {
	return &ChomperSystem{
		entityManager: em,
		gameState:     gs,
	}
}

// Update 更新所有大嘴花实体的状态
func (s *ChomperSystem) Update(deltaTime float64) {
	// 检查游戏是否结束
	if s.gameState != nil && s.gameState.IsGameOver {
		return
	}

	// 检查游戏是否冻结
	freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](s.entityManager)
	if len(freezeEntities) > 0 {
		return
	}

	// 分别查询 ChomperComponent 和 PositionComponent，用于调试
	chomperOnly := ecs.GetEntitiesWith1[*components.ChomperComponent](s.entityManager)
	posOnly := ecs.GetEntitiesWith1[*components.PositionComponent](s.entityManager)

	// 获取所有拥有 ChomperComponent 的植物
	chompers := ecs.GetEntitiesWith2[*components.ChomperComponent, *components.PositionComponent](s.entityManager)

	// 静默忽略未使用的调试变量
	_ = chomperOnly
	_ = posOnly

	for _, entityID := range chompers {
		chomper, ok := ecs.GetComponent[*components.ChomperComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 根据当前状态处理逻辑
		switch chomper.State {
		case components.ChomperStateIdle:
			s.handleIdleState(entityID, chomper, deltaTime)
		case components.ChomperStateAttacking:
			s.handleAttackingState(entityID, chomper, deltaTime)
		case components.ChomperStateSwallowing:
			s.handleSwallowingState(entityID, chomper, deltaTime)
		case components.ChomperStateBiting:
			s.handleBitingState(entityID, chomper, deltaTime)
		case components.ChomperStateChewing:
			s.handleChewingState(entityID, chomper, deltaTime)
		}
	}
}

// handleIdleState 处理待机状态：检测前方僵尸
func (s *ChomperSystem) handleIdleState(entityID ecs.EntityID, chomper *components.ChomperComponent, deltaTime float64) {
	// 获取大嘴花位置
	pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[ChomperSystem] 大嘴花 %d 没有 PositionComponent", entityID)
		return
	}

	plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[ChomperSystem] 大嘴花 %d 没有 PlantComponent", entityID)
		return
	}

	// 检测前方僵尸
	targetZombie := s.findNearestZombieInRange(pos, plant.GridRow)
	if targetZombie == 0 {
		return // 没有目标
	}

	// 检查目标是否可吞噬
	canSwallow := s.canSwallowZombie(targetZombie)

	chomper.TargetZombie = targetZombie
	chomper.DamageDealt = false

	if canSwallow {
		// 可吞噬：先进入攻击状态播放咬住动画
		chomper.State = components.ChomperStateAttacking
		chomper.AttackCooldown = 0
		s.playAnimation(entityID, "anim_bite")
	} else {
		// 不可吞噬：切换到撕咬状态
		chomper.State = components.ChomperStateBiting
		chomper.AttackCooldown = 0
		s.playAnimation(entityID, "anim_bite")
	}
}

// handleAttackingState 处理攻击状态：播放咬住动画
func (s *ChomperSystem) handleAttackingState(entityID ecs.EntityID, chomper *components.ChomperComponent, deltaTime float64) {
	// 检查动画组件
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 在咬住关键帧时删除僵尸
	if !chomper.DamageDealt && reanim.CurrentFrame >= config.ChomperBiteFrame && chomper.TargetZombie != 0 {
		s.dealSwallowDamage(chomper.TargetZombie)
		chomper.DamageDealt = true
		chomper.TargetZombie = 0
	}

	// anim_bite 动画完成后进入消化状态（咀嚼 42 秒）
	if reanim.IsFinished {
		chomper.State = components.ChomperStateChewing
		chomper.DigestTimer = 0
		chomper.DamageDealt = false
		s.playAnimation(entityID, "anim_chew")
	}
}

// handleSwallowingState 处理吞咽状态
func (s *ChomperSystem) handleSwallowingState(entityID ecs.EntityID, chomper *components.ChomperComponent, deltaTime float64) {
	// 检查动画是否完成
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 吞咽动画完成后回到待机状态
	if reanim.IsFinished {
		chomper.State = components.ChomperStateIdle
		s.playAnimation(entityID, "anim_idle")
	}
}

// handleBitingState 处理撕咬状态
func (s *ChomperSystem) handleBitingState(entityID ecs.EntityID, chomper *components.ChomperComponent, deltaTime float64) {
	// 检查动画是否完成
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 更新攻击计时器
	chomper.AttackCooldown += deltaTime

	// 在动画播放到一定程度时造成伤害（约 0.4 秒后）
	const damageDelay = 0.4
	if !chomper.DamageDealt && chomper.AttackCooldown >= damageDelay {
		s.dealBiteDamage(chomper.TargetZombie)
		chomper.DamageDealt = true
	}

	// 动画完成后检查目标是否还存在
	if reanim.IsFinished {
		// 检查目标是否还存在且在范围内
		if s.isZombieStillValid(chomper.TargetZombie, entityID) {
			// 继续撕咬
			chomper.DamageDealt = false
			chomper.AttackCooldown = 0
			s.playAnimation(entityID, "anim_bite")
		} else {
			// 目标消失，回到待机
			chomper.State = components.ChomperStateIdle
			chomper.TargetZombie = 0
			chomper.AttackCooldown = 0
			s.playAnimation(entityID, "anim_idle")
		}
	}
}

// handleChewingState 处理消化状态（咀嚼 42 秒）
func (s *ChomperSystem) handleChewingState(entityID ecs.EntityID, chomper *components.ChomperComponent, deltaTime float64) {
	// 更新消化计时器
	chomper.DigestTimer += deltaTime

	// 消化完成（42秒），播放吞咽动画
	if chomper.DigestTimer >= config.ChomperDigestTime {
		chomper.State = components.ChomperStateSwallowing
		chomper.DigestTimer = 0
		s.playAnimation(entityID, "anim_swallow")
	}
}

// findNearestZombieInRange 查找前方 1 格内最近的僵尸
func (s *ChomperSystem) findNearestZombieInRange(pos *components.PositionComponent, plantRow int) ecs.EntityID {
	// 获取所有僵尸
	zombies := ecs.GetEntitiesWith2[*components.ZombieTagComponent, *components.PositionComponent](s.entityManager)

	var nearestZombie ecs.EntityID = 0
	nearestDistance := float64(config.ChomperAttackRange*config.CellWidth) + config.CellWidth // 最大检测距离

	for _, zombieID := range zombies {
		zombiePos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
		if !ok {
			continue
		}

		// 检查僵尸是否在同一行
		zombieRow := utils.GetEntityRow(zombiePos.Y, config.GridWorldStartY, config.CellHeight)
		if zombieRow != plantRow {
			continue
		}

		// 检查僵尸是否在大嘴花前方（X 坐标更大）
		if zombiePos.X <= pos.X {
			continue
		}

		// 检查是否在攻击范围内（2 格）
		distance := zombiePos.X - pos.X
		maxRange := float64(config.ChomperAttackRange) * config.CellWidth

		if distance <= maxRange && distance < nearestDistance {
			// 检查僵尸是否处于活动状态（不是死亡中）
			behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
			if ok && (behavior.Type == components.BehaviorZombieDying || behavior.Type == components.BehaviorZombieDyingExplosion) {
				continue // 跳过死亡中的僵尸
			}

			nearestZombie = zombieID
			nearestDistance = distance
		}
	}

	return nearestZombie
}

// canSwallowZombie 检查是否可以吞噬目标僵尸
// 巨人僵尸、红眼巨人、僵王博士等无法吞噬
func (s *ChomperSystem) canSwallowZombie(zombieID ecs.EntityID) bool {
	// 通过 ReanimComponent.ReanimName 判断僵尸类型
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, zombieID)
	if !ok {
		// 如果没有动画组件，假设可以吞噬
		return true
	}

	// 检查僵尸类型是否为巨人类（通过 Reanim 名称判断）
	reanimName := reanim.ReanimName
	switch reanimName {
	case "Gargantuar", "GargantuarRedeye", "DrZomboss":
		return false // 巨人类无法吞噬
	default:
		return true
	}
}

// dealSwallowDamage 对目标造成吞噬伤害（僵尸直接消失）
func (s *ChomperSystem) dealSwallowDamage(zombieID ecs.EntityID) {
	// 播放吞噬音效
	if s.gameState != nil {
		if audioManager := s.gameState.GetAudioManager(); audioManager != nil {
			audioManager.PlaySound("SOUND_CHOMP")
		}
	}

	// 增加僵尸击杀计数（必须在删除实体之前调用）
	if s.gameState != nil {
		s.gameState.IncrementZombiesKilled()
	}

	// 直接删除僵尸实体（不播放死亡动画）
	s.entityManager.DestroyEntity(zombieID)
}

// dealBiteDamage 对目标造成撕咬伤害
func (s *ChomperSystem) dealBiteDamage(zombieID ecs.EntityID) {
	health, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, zombieID)
	if !ok {
		return
	}

	// 先对护甲造成伤害（如果有）
	armor, hasArmor := ecs.GetComponent[*components.ArmorComponent](s.entityManager, zombieID)
	damage := config.ChomperBiteDamage

	if hasArmor && armor.CurrentArmor > 0 {
		if armor.CurrentArmor >= damage {
			armor.CurrentArmor -= damage
			damage = 0
		} else {
			damage -= armor.CurrentArmor
			armor.CurrentArmor = 0
		}
	}

	// 剩余伤害对生命值造成伤害
	if damage > 0 {
		health.CurrentHealth -= damage
	}

	// 播放撕咬音效
	if s.gameState != nil {
		if audioManager := s.gameState.GetAudioManager(); audioManager != nil {
			audioManager.PlaySound("SOUND_CHOMP")
		}
	}
}

// isZombieStillValid 检查僵尸是否仍然有效（存在且在范围内）
func (s *ChomperSystem) isZombieStillValid(zombieID ecs.EntityID, chomperID ecs.EntityID) bool {
	// 检查僵尸是否存在且还有生命值
	// 如果实体不存在或没有 HealthComponent，GetComponent 会返回 ok=false
	health, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, zombieID)
	if !ok || health.CurrentHealth <= 0 {
		return false
	}

	// 检查僵尸是否在攻击范围内
	chomperPos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, chomperID)
	if !ok {
		return false
	}

	zombiePos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombieID)
	if !ok {
		return false
	}

	// 检查是否在前方
	if zombiePos.X <= chomperPos.X {
		return false
	}

	// 检查是否在攻击范围内
	distance := zombiePos.X - chomperPos.X
	maxRange := float64(config.ChomperAttackRange) * config.CellWidth

	return distance <= maxRange
}

// playAnimation 通过 AnimationCommandComponent 播放动画
func (s *ChomperSystem) playAnimation(entityID ecs.EntityID, animName string) {
	ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
		UnitID:        "chomper",
		AnimationName: animName,
		Processed:     false,
	})
}
