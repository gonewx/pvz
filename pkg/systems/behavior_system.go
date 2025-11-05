package systems

import (
	"log"
	"math/rand"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/utils"
)

// BehaviorSystem 处理实体的行为逻辑
// 根据实体的 BehaviorComponent 类型执行相应的行为（如向日葵生产阳光、豌豆射手攻击等）
type BehaviorSystem struct {
	entityManager    *ecs.EntityManager
	resourceManager  *game.ResourceManager
	reanimSystem     *ReanimSystem   // Story 6.3: 用于切换僵尸动画状态
	gameState        *game.GameState // Story 5.5: 用于僵尸死亡计数
	logFrameCounter  int             // 日志输出计数器（避免全局变量）
	lawnGridSystem   *LawnGridSystem // Bug Fix: 用于植物死亡时释放网格占用
	lawnGridEntityID ecs.EntityID    // Bug Fix: 草坪网格实体ID
}

// 日志输出间隔常量
const LogOutputFrameInterval = 100 // 日志输出间隔（每N帧输出一次）

// NewBehaviorSystem 创建一个新的行为系统
// 参数:
//   - em: EntityManager 实例
//   - rm: ResourceManager 实例
//   - rs: ReanimSystem 实例 (Story 6.3: 用于切换僵尸动画)
//   - gs: GameState 实例 (Story 5.5: 用于僵尸死亡计数)
//   - lgs: LawnGridSystem 实例 (Bug Fix: 用于植物死亡时释放网格占用)
//   - lawnGridID: 草坪网格实体ID (Bug Fix)
func NewBehaviorSystem(em *ecs.EntityManager, rm *game.ResourceManager, rs *ReanimSystem, gs *game.GameState, lgs *LawnGridSystem, lawnGridID ecs.EntityID) *BehaviorSystem {
	return &BehaviorSystem{
		entityManager:    em,
		resourceManager:  rm,
		reanimSystem:     rs,
		gameState:        gs,
		lawnGridSystem:   lgs,
		lawnGridEntityID: lawnGridID,
	}
}

// Update 更新所有拥有行为组件的实体
func (s *BehaviorSystem) Update(deltaTime float64) {
	// 查询所有植物实体
	plantEntityList := s.queryPlants()

	// 查询所有移动中的僵尸实体
	zombieEntityList := s.queryMovingZombies()

	// DEBUG: 记录僵尸数量
	if len(zombieEntityList) > 0 {
		log.Printf("[BehaviorSystem] Update called, found %d moving zombies", len(zombieEntityList))
	}

	// 查询所有啃食中的僵尸实体
	eatingZombieEntityList := s.queryEatingZombies()

	// 查询所有死亡中的僵尸实体
	dyingZombieEntityList := s.queryDyingZombies()

	// 合并所有活动僵尸列表（移动中 + 啃食中），用于豌豆射手检测目标
	allZombieEntityList := append([]ecs.EntityID{}, zombieEntityList...)
	allZombieEntityList = append(allZombieEntityList, eatingZombieEntityList...)

	// 查询所有豌豆子弹实体
	projectileEntityList := s.queryProjectiles()

	// 日志输出（避免每帧都打印）
	totalZombies := len(zombieEntityList) + len(eatingZombieEntityList)
	totalEntities := len(plantEntityList) + totalZombies + len(projectileEntityList)
	if totalEntities > 0 {
		s.logFrameCounter++
		if s.logFrameCounter%LogOutputFrameInterval == 1 {
			log.Printf("[BehaviorSystem] 更新 %d 个行为实体 (植物: %d, 僵尸: %d [移动:%d 啃食:%d], 子弹: %d)",
				totalEntities, len(plantEntityList), totalZombies, len(zombieEntityList), len(eatingZombieEntityList), len(projectileEntityList))
		}
	}

	// 遍历所有植物实体，根据行为类型分发处理
	for _, entityID := range plantEntityList {
		behaviorComp, _ := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)

		// 根据行为类型分发
		switch behaviorComp.Type {
		case components.BehaviorSunflower:
			s.handleSunflowerBehavior(entityID, deltaTime)
		case components.BehaviorPeashooter:
			s.handlePeashooterBehavior(entityID, deltaTime, allZombieEntityList)
		case components.BehaviorWallnut:
			s.handleWallnutBehavior(entityID)
		case components.BehaviorCherryBomb:
			s.handleCherryBombBehavior(entityID, deltaTime)
		default:
			// 未知行为类型，忽略
		}
	}

	// Story 10.3: 更新植物攻击动画状态（在所有行为处理之后）
	for _, entityID := range plantEntityList {
		s.updatePlantAttackAnimation(entityID, deltaTime)
	}

	// 遍历所有移动中的僵尸实体，根据行为类型分发处理
	for _, entityID := range zombieEntityList {
		behaviorComp, _ := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)

		// 根据行为类型分发
		switch behaviorComp.Type {
		case components.BehaviorZombieBasic:
			s.handleZombieBasicBehavior(entityID, deltaTime)
		case components.BehaviorZombieConehead:
			s.handleConeheadZombieBehavior(entityID, deltaTime)
		case components.BehaviorZombieBuckethead:
			s.handleBucketheadZombieBehavior(entityID, deltaTime)
		default:
			// 未知僵尸类型，忽略
		}
	}

	// 遍历所有啃食中的僵尸实体
	for _, entityID := range eatingZombieEntityList {
		behaviorComp, _ := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)

		// 只处理啃食状态的僵尸
		if behaviorComp.Type == components.BehaviorZombieEating {
			s.handleZombieEatingBehavior(entityID, deltaTime)
		}
	}

	// 遍历所有子弹实体，根据行为类型分发处理
	for _, entityID := range projectileEntityList {
		behaviorComp, _ := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)

		// 根据行为类型分发
		switch behaviorComp.Type {
		case components.BehaviorPeaProjectile:
			s.handlePeaProjectileBehavior(entityID, deltaTime)
		default:
			// 忽略非子弹类型（如僵尸）
		}
	}

	// 遍历所有死亡中的僵尸实体（处理死亡动画完成后的删除）
	for _, entityID := range dyingZombieEntityList {
		behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 只处理死亡中的僵尸
		if behaviorComp.Type == components.BehaviorZombieDying {
			s.handleZombieDyingBehavior(entityID)
		}
	}

	// 查询所有击中效果实体（拥有 BehaviorComponent 和 TimerComponent）
	hitEffectEntityList := ecs.GetEntitiesWith2[
		*components.BehaviorComponent,
		*components.TimerComponent,
	](s.entityManager)

	// 遍历所有击中效果实体，管理其生命周期
	for _, entityID := range hitEffectEntityList {
		behaviorComp, _ := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)

		// 只处理击中效果类型
		if behaviorComp.Type == components.BehaviorPeaBulletHit {
			s.handleHitEffectBehavior(entityID, deltaTime)
		}
	}

}

// handleSunflowerBehavior 处理向日葵的行为逻辑
// 向日葵会定期生产阳光
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

		// 初始化阳光动画（Sun.reanim 是效果动画，使用场景动画模式，不计算 CenterOffset）
		if err := s.reanimSystem.InitializeSceneAnimation(sunID); err != nil {
			log.Printf("[BehaviorSystem] WARNING: 初始化阳光动画失败: %v", err)
		}

		// 重置计时器
		timer.CurrentTime = 0
		// 首次生产后，后续生产周期为 24 秒
		timer.TargetTime = 24.0

		// 注意：向日葵的待机动画一直循环播放，生产阳光时不需要特殊动画
	}
}

// handleZombieBasicBehavior 处理普通僵尸的行为逻辑
// 普通僵尸会以恒定速度从右向左移动
func (s *BehaviorSystem) handleZombieBasicBehavior(entityID ecs.EntityID, deltaTime float64) {
	// Story 8.3: 检查僵尸是否已激活（开场动画期间僵尸未激活，不应移动）
	if waveState, ok := ecs.GetComponent[*components.ZombieWaveStateComponent](s.entityManager, entityID); ok {
		if !waveState.IsActivated {
			// DEBUG: 记录未激活的僵尸被跳过
			log.Printf("[BehaviorSystem] Zombie %d NOT activated (wave %d), skipping behavior", entityID, waveState.WaveIndex)
			// 僵尸未激活，跳过所有行为逻辑（保持静止展示）
			return
		}
	}

	// 检查生命值（Story 4.4: 僵尸死亡逻辑）
	health, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, entityID)
	if ok {
		// 更新僵尸的受伤状态（掉手臂、掉头）
		s.updateZombieDamageState(entityID, health)

		if health.CurrentHealth <= 0 {
			// 生命值 <= 0，触发死亡状态转换
			log.Printf("[BehaviorSystem] 僵尸 %d 生命值 <= 0 (HP=%d)，触发死亡", entityID, health.CurrentHealth)
			s.triggerZombieDeath(entityID)
			return // 跳过正常移动逻辑
		}
	}

	// 获取位置组件
	position, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// Story 5.1: 检测植物碰撞（在移动之前）
	// 计算僵尸所在格子
	// 注意：需要减去 ZombieVerticalOffset，因为僵尸Y坐标包含了偏移
	zombieCol := int((position.X - config.GridWorldStartX) / config.CellWidth)
	zombieRow := int((position.Y - config.GridWorldStartY - config.ZombieVerticalOffset - config.CellHeight/2.0) / config.CellHeight)

	// 检测是否与植物在同一格子
	plantID, hasCollision := s.detectPlantCollision(zombieRow, zombieCol)
	if hasCollision {
		log.Printf("[BehaviorSystem] ✅ 僵尸 %d 检测到植物 %d，位置(%d,%d)，开始啃食！", entityID, plantID, zombieRow, zombieCol)
		// 进入啃食状态
		s.startEatingPlant(entityID, plantID)
		return // 跳过移动逻辑
	}

	// 获取速度组件
	velocity, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[BehaviorSystem] ⚠️ 僵尸 %d 缺少 VelocityComponent（可能已进入啃食状态）", entityID)
		return
	}

	// DEBUG: 记录僵尸速度
	log.Printf("[BehaviorSystem] Zombie %d moving: X=%.1f, VX=%.2f, VY=%.2f",
		entityID, position.X, velocity.VX, velocity.VY)

	// 更新位置：根据速度和时间增量移动僵尸
	position.X += velocity.VX * deltaTime
	position.Y += velocity.VY * deltaTime

	// 边界检查：如果僵尸移出屏幕左侧，标记删除
	// 使用 config.ZombieDeletionBoundary 提供容错空间，避免僵尸刚移出就被删除
	if position.X < config.ZombieDeletionBoundary {
		log.Printf("[BehaviorSystem] 僵尸 %d 移出屏幕左侧 (X=%.1f)，标记删除", entityID, position.X)
		s.entityManager.DestroyEntity(entityID)
	}
}

// triggerZombieDeath 触发僵尸死亡状态转换
// 当僵尸生命值 <= 0 时调用，将僵尸从正常行为状态切换到死亡动画播放状态
// Story 7.4: 添加僵尸死亡粒子效果触发（头部掉落）
// 注意：手臂掉落粒子效果在 updateZombieDamageState 中触发（受伤时）
func (s *BehaviorSystem) triggerZombieDeath(entityID ecs.EntityID) {
	// 1. 切换行为类型为 BehaviorZombieDying
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[BehaviorSystem] 僵尸 %d 缺少 BehaviorComponent，无法触发死亡", entityID)
		return
	}
	behavior.Type = components.BehaviorZombieDying
	log.Printf("[BehaviorSystem] 僵尸 %d 行为切换为 BehaviorZombieDying", entityID)

	// Story 7.4: 获取僵尸位置，用于触发粒子效果
	position, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		log.Printf("[BehaviorSystem] 警告：僵尸 %d 缺少 PositionComponent，无法触发粒子效果", entityID)
	} else {
		// Story 7.6: 检测僵尸行进方向，计算粒子角度偏移
		// 粒子效果应该在僵尸行进的反方向飞出
		//
		// 角度系统：标准屏幕坐标系（0°=右，90°=下，180°=左，270°=上）
		// ZombieHead 配置：LaunchAngle [150-185°] ≈ 向左下
		// 该配置是为**向右走的僵尸**设计的（头向左后方飞）
		//
		// 我们游戏中僵尸通常向左走，需要翻转方向
		angleOffset := 180.0 // 默认翻转（适合僵尸向左走）
		velocity, hasVelocity := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID)
		if hasVelocity {
			if velocity.VX > 0 {
				// 僵尸向右走 → 配置已经正确 → 不翻转
				angleOffset = 0.0
			} else {
				// 僵尸向左走 → 配置方向相反 → 翻转 180°
				// [150-185°] + 180° = [330-365°] = [-30 to 5°] → 向右后方
				angleOffset = 180.0
			}
			log.Printf("[BehaviorSystem] 僵尸 %d 方向: VX=%.1f → 粒子角度偏移=%.0f°", entityID, velocity.VX, angleOffset)
		}

		// Story 7.4: 触发僵尸头部掉落粒子效果
		_, err := entities.CreateParticleEffect(
			s.entityManager,
			s.resourceManager,
			"ZombieHead", // 粒子效果名称（不带.xml后缀）
			position.X, position.Y,
			angleOffset, // Story 7.6: 传递角度偏移
		)
		if err != nil {
			log.Printf("[BehaviorSystem] 警告：创建僵尸头部掉落粒子效果失败: %v", err)
			// 不阻塞游戏逻辑，游戏继续运行
		} else {
			log.Printf("[BehaviorSystem] 僵尸 %d 触发头部掉落粒子效果，位置: (%.1f, %.1f)", entityID, position.X, position.Y)
		}
	}

	// 2. 使用 ReanimSystem 通用接口隐藏 "head" 部件组（头掉落效果）
	// 部件组映射在实体创建时配置（zombie_factory.go），BehaviorSystem 不需要知道具体轨道名
	if err := s.reanimSystem.HidePartGroup(entityID, "head"); err != nil {
		log.Printf("[BehaviorSystem] 警告：僵尸 %d 隐藏头部失败: %v", entityID, err)
	} else {
		log.Printf("[BehaviorSystem] 僵尸 %d 头部掉落", entityID)
	}

	// 3. 移除 VelocityComponent（停止移动）
	ecs.RemoveComponent[*components.VelocityComponent](s.entityManager, entityID)
	log.Printf("[BehaviorSystem] 僵尸 %d 移除速度组件，停止移动", entityID)

	// 4. 使用 ReanimSystem 播放死亡动画（不循环）
	// 尝试播放 anim_death 动画（从Zombie.reanim）
	if err := s.reanimSystem.PlayAnimationNoLoop(entityID, "anim_death"); err != nil {
		log.Printf("[BehaviorSystem] 僵尸 %d 播放死亡动画失败: %v，直接删除", entityID, err)
		// 错误处理：如果死亡动画播放失败，直接删除僵尸
		s.entityManager.DestroyEntity(entityID)
		return
	}

	log.Printf("[BehaviorSystem] 僵尸 %d 死亡动画已开始播放 (anim_death, 不循环)", entityID)
}

// handlePeashooterBehavior 处理豌豆射手的行为逻辑
// 豌豆射手会周期性扫描同行僵尸并发射豌豆子弹
func (s *BehaviorSystem) handlePeashooterBehavior(entityID ecs.EntityID, deltaTime float64, zombieEntityList []ecs.EntityID) {
	// 获取植物组件（用于状态管理）
	plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 获取计时器组件
	timer, ok := ecs.GetComponent[*components.TimerComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 更新计时器
	timer.CurrentTime += deltaTime

	// Story 10.3: 只有在空闲状态时才能触发新的攻击
	// 确保攻击动画播放完毕后，才能进行下一次攻击
	if plant.AttackAnimState != components.AttackAnimIdle {
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
			// Story 6.9: 使用多动画叠加实现攻击动画
			// 同时播放身体攻击动画（anim_shooting）和头部动画（anim_head_idle）
			// 这样可以确保头部在攻击时不会消失
			err := s.reanimSystem.PlayAnimations(entityID, []string{"anim_shooting", "anim_head_idle"})
			if err != nil {
				log.Printf("[BehaviorSystem] 切换到攻击动画失败: %v", err)
			} else {
				// 设置为非循环模式（单次播放）
				if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
					reanim.IsLooping = false
				}

				log.Printf("[BehaviorSystem] 豌豆射手 %d 切换到攻击动画（anim_shooting + anim_head_idle，单次播放）", entityID)
				// 设置攻击动画状态，用于动画完成后切换回 idle
				plant.AttackAnimState = components.AttackAnimAttacking
			}

			// Story 10.5: 设置"等待发射"状态，但不立即创建子弹
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

// handlePeaProjectileBehavior 处理豌豆子弹的移动逻辑
// 豌豆子弹会以恒定速度向右移动，飞出屏幕后被删除
func (s *BehaviorSystem) handlePeaProjectileBehavior(entityID ecs.EntityID, deltaTime float64) {
	// 获取位置组件
	position, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 获取速度组件
	velocity, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 更新位置：根据速度和时间增量移动子弹
	position.X += velocity.VX * deltaTime
	position.Y += velocity.VY * deltaTime

	// 边界检查：如果子弹飞出屏幕右侧，标记删除
	if position.X > config.PeaBulletDeletionBoundary {
		log.Printf("[BehaviorSystem] 豌豆子弹 %d 飞出屏幕右侧 (X=%.1f)，标记删除", entityID, position.X)
		s.entityManager.DestroyEntity(entityID)
	}
}

// handleHitEffectBehavior 处理击中效果的生命周期
// 击中效果会在显示一段时间后自动消失
func (s *BehaviorSystem) handleHitEffectBehavior(entityID ecs.EntityID, deltaTime float64) {
	// 获取计时器组件
	timer, ok := ecs.GetComponent[*components.TimerComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 更新计时器
	timer.CurrentTime += deltaTime

	// 检查计时器是否完成（超时）
	if timer.CurrentTime >= timer.TargetTime {
		// 击中效果生命周期结束，标记删除
		s.entityManager.DestroyEntity(entityID)
	}
}

// handleZombieDyingBehavior 处理僵尸死亡动画播放
// 当死亡动画完成后，删除僵尸实体
func (s *BehaviorSystem) handleZombieDyingBehavior(entityID ecs.EntityID) {
	// 获取 ReanimComponent
	reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		// 如果没有 ReanimComponent，直接删除僵尸
		log.Printf("[BehaviorSystem] 死亡中的僵尸 %d 缺少 ReanimComponent，直接删除", entityID)
		// Story 5.5: 僵尸死亡，增加计数
		s.gameState.IncrementZombiesKilled()
		s.entityManager.DestroyEntity(entityID)
		return
	}

	// 检查死亡动画是否完成
	// 使用 IsFinished 标志来判断非循环动画是否已完成
	if reanim.IsFinished {
		log.Printf("[BehaviorSystem] 僵尸 %d 死亡动画完成 (frame %d/%d)，删除实体",
			entityID, reanim.CurrentFrame, reanim.VisibleFrameCount)
		// Story 5.5: 僵尸死亡，增加计数
		s.gameState.IncrementZombiesKilled()
		s.entityManager.DestroyEntity(entityID)
	} else {
		// 调试日志：定期输出动画状态（每10帧输出一次）
		// if reanim.CurrentFrame%10 == 0 {
		// 	log.Printf("[BehaviorSystem] 僵尸 %d 死亡动画进行中: Frame=%d/%d, IsLooping=%v, IsFinished=%v",
		// 		entityID, reanim.CurrentFrame, reanim.VisibleFrameCount, reanim.IsLooping, reanim.IsFinished)
		// }
	}
}

// updateZombieDamageState 根据生命值更新僵尸的受伤状态
// 僵尸有三个受伤阶段：
// 1. 健康（HP > 90）：完整外观
// 2. 掉手臂（HP <= 90 且 HP > 0）：隐藏外侧手臂
// 3. 掉头（HP <= 0）：无头状态（在 triggerZombieDeath 中处理）
func (s *BehaviorSystem) updateZombieDamageState(entityID ecs.EntityID, health *components.HealthComponent) {
	// 生命值阈值：90（33%，根据原版游戏数据）
	const armLostThreshold = 90

	// 检查是否应该掉手臂（生命值 <= 90 且手臂尚未掉落）
	if health.CurrentHealth <= armLostThreshold && !health.ArmLost {
		// 标记手臂已掉落，防止重复触发
		health.ArmLost = true

		// 使用 ReanimSystem 通用接口隐藏 "arm" 部件组
		// 部件组映射在实体创建时配置（zombie_factory.go），BehaviorSystem 不需要知道具体轨道名
		if err := s.reanimSystem.HidePartGroup(entityID, "arm"); err != nil {
			// 如果实体没有配置 PartGroups（非僵尸实体），静默忽略
			return
		}

		log.Printf("[BehaviorSystem] 僵尸 %d 手臂掉落 (HP=%d/%d)",
			entityID, health.CurrentHealth, health.MaxHealth)

		// 获取僵尸位置，用于触发粒子效果
		position, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		if !ok {
			log.Printf("[BehaviorSystem] 警告：僵尸 %d 缺少 PositionComponent，无法触发手臂掉落粒子", entityID)
			return
		}

		// 检测僵尸行进方向，计算粒子角度偏移
		angleOffset := 180.0 // 默认翻转（适合僵尸向左走）
		velocity, hasVelocity := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID)
		if hasVelocity {
			if velocity.VX > 0 {
				angleOffset = 0.0 // 僵尸向右走
			} else {
				angleOffset = 180.0 // 僵尸向左走
			}
		}

		// 触发僵尸手臂掉落粒子效果
		_, err := entities.CreateParticleEffect(
			s.entityManager,
			s.resourceManager,
			"ZombieArm", // 粒子效果名称（不带.xml后缀）
			position.X, position.Y,
			angleOffset, // 角度偏移
		)
		if err != nil {
			log.Printf("[BehaviorSystem] 警告：创建僵尸手臂掉落粒子效果失败: %v", err)
		} else {
			log.Printf("[BehaviorSystem] 僵尸 %d 触发手臂掉落粒子效果，位置: (%.1f, %.1f)", entityID, position.X, position.Y)
		}
	}
}

// playShootSound 播放豌豆射手发射子弹的音效
// 使用配置文件中定义的音效（config.PeashooterShootSoundPath）
// 如果配置为空字符串，则不播放音效（静音模式）
func (s *BehaviorSystem) playShootSound() {
	// 如果配置为空字符串，不播放音效（保持原版静音风格）
	if config.PeashooterShootSoundPath == "" {
		return
	}

	// 加载发射音效（如果已加载，会返回缓存的播放器）
	// 音效路径在 pkg/config/unit_config.go 中配置，可根据需要切换测试
	shootSound, err := s.resourceManager.LoadSoundEffect(config.PeashooterShootSoundPath)
	if err != nil {
		// 音效加载失败时不阻止游戏继续运行
		// 在实际项目中可以使用日志系统记录错误
		return
	}

	// 重置播放器位置到开头（允许快速连续播放）
	shootSound.Rewind()

	// 播放音效
	shootSound.Play()
}

// detectPlantCollision 检测僵尸是否与植物发生网格碰撞
// 参数:
//   - zombieRow: 僵尸所在行
//   - zombieCol: 僵尸所在列
//
// 返回:
//   - ecs.EntityID: 植物实体ID（如果碰撞）
//   - bool: 是否发生碰撞
func (s *BehaviorSystem) detectPlantCollision(zombieRow, zombieCol int) (ecs.EntityID, bool) {
	// 查询所有植物实体（拥有 PlantComponent）
	plantEntityList := ecs.GetEntitiesWith1[*components.PlantComponent](s.entityManager)

	// 遍历所有植物，比对网格位置
	for _, plantID := range plantEntityList {
		plant, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, plantID)
		if !ok {
			continue
		}

		// 检查是否在同一格子
		if plant.GridRow == zombieRow && plant.GridCol == zombieCol {
			return plantID, true
		}
	}

	// 没有找到植物
	return 0, false
}

// changeZombieAnimation 切换僵尸动画状态
// 参数:
//   - zombieID: 僵尸实体ID
//   - newState: 新的动画状态
func (s *BehaviorSystem) changeZombieAnimation(zombieID ecs.EntityID, newState components.ZombieAnimState) {
	// 获取行为组件
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
	if !ok {
		return
	}

	// 如果状态没有变化，不需要切换动画
	if behavior.ZombieAnimState == newState {
		return
	}

	// 更新状态
	behavior.ZombieAnimState = newState

	// 根据状态切换动画
	var animName string
	switch newState {
	case components.ZombieAnimIdle:
		animName = "anim_idle"
	case components.ZombieAnimWalking:
		animName = "anim_walk"
	case components.ZombieAnimEating:
		animName = "anim_eat"
	case components.ZombieAnimDying:
		animName = "anim_death"
	default:
		return
	}

	// 使用 ReanimSystem 播放新动画
	if s.reanimSystem != nil {
		err := s.reanimSystem.PlayAnimation(zombieID, animName)
		if err != nil {
			log.Printf("[BehaviorSystem] 僵尸 %d 切换动画失败: %v", zombieID, err)
		} else {
			log.Printf("[BehaviorSystem] 僵尸 %d 切换动画: %s", zombieID, animName)
		}
	}
}

// startEatingPlant 开始啃食植物
// 参数:
//   - zombieID: 僵尸实体ID
//   - plantID: 植物实体ID
func (s *BehaviorSystem) startEatingPlant(zombieID, plantID ecs.EntityID) {
	log.Printf("[BehaviorSystem] 僵尸 %d 开始啃食植物 %d", zombieID, plantID)

	// 1. 移除僵尸的 VelocityComponent（停止移动）
	ecs.RemoveComponent[*components.VelocityComponent](s.entityManager, zombieID)

	// 2. Story 5.3: 在切换类型之前，先记住原始僵尸类型（用于选择正确的啃食动画）
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
	if !ok {
		return
	}
	originalZombieType := behavior.Type // 记住原始类型

	// 3. 切换 BehaviorComponent.Type 为 BehaviorZombieEating
	behavior.Type = components.BehaviorZombieEating

	// Story 6.3: 切换僵尸动画为啃食状态
	s.changeZombieAnimation(zombieID, components.ZombieAnimEating)

	// 4. 添加 TimerComponent 用于伤害间隔
	ecs.AddComponent(s.entityManager, zombieID, &components.TimerComponent{
		Name:        "eating_damage",
		TargetTime:  config.ZombieEatingDamageInterval,
		CurrentTime: 0,
		IsReady:     false,
	})

	// TODO(Story 6.3): 迁移到 ReanimComponent
	// 5. Story 5.3: 根据原始僵尸类型加载对应的啃食动画
	// var eatFrames []*ebiten.Image

	_ = originalZombieType // 临时避免未使用警告
	/*
		switch originalZombieType {
		case components.BehaviorZombieConehead:
			// 路障僵尸啃食动画
			eatFrames, _ = utils.LoadConeheadZombieEatAnimation(s.resourceManager)
			log.Printf("[BehaviorSystem] 路障僵尸 %d 开始啃食，使用路障僵尸啃食动画", zombieID)
		case components.BehaviorZombieBuckethead:
			// 铁桶僵尸啃食动画
			eatFrames, _ = utils.LoadBucketheadZombieEatAnimation(s.resourceManager)
			log.Printf("[BehaviorSystem] 铁桶僵尸 %d 开始啃食，使用铁桶僵尸啃食动画", zombieID)
		default:
			// 普通僵尸或其他类型
			eatFrames = utils.LoadZombieEatAnimation(s.resourceManager)
		}

		// TODO(Story 6.3): 迁移到 ReanimComponent
		// 6. 替换 AnimationComponent 为啃食动画
		// animComp, ok := s.entityManager.GetComponent(zombieID, reflect.TypeOf(&components.AnimationComponent{}))
		// if ok {
		// 	anim := animComp.(*components.AnimationComponent)
		// 	anim.Frames = eatFrames
		// 	anim.FrameSpeed = config.ZombieEatFrameSpeed
		// 	anim.CurrentFrame = 0
		// 	anim.FrameCounter = 0
		// 	anim.IsLooping = true
		// 	anim.IsFinished = false
		// }
	*/
}

// stopEatingAndResume 停止啃食并恢复移动
// 参数:
//   - zombieID: 僵尸实体ID
func (s *BehaviorSystem) stopEatingAndResume(zombieID ecs.EntityID) {
	log.Printf("[BehaviorSystem] 僵尸 %d 结束啃食，恢复移动", zombieID)

	// 1. 移除 TimerComponent
	ecs.RemoveComponent[*components.TimerComponent](s.entityManager, zombieID)

	// 2. 切换 BehaviorComponent.Type 回 BehaviorZombieBasic
	behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, zombieID)
	if ok {
		behavior.Type = components.BehaviorZombieBasic
	}

	// Story 6.3: 切换僵尸动画回行走状态
	s.changeZombieAnimation(zombieID, components.ZombieAnimWalking)

	// 3. 恢复 VelocityComponent
	ecs.AddComponent(s.entityManager, zombieID, &components.VelocityComponent{
		VX: config.ZombieWalkSpeed,
		VY: 0,
	})

	// TODO(Story 6.3): 迁移到 ReanimComponent
	// 4. 加载僵尸走路动画帧序列
	// walkFrames := utils.LoadZombieWalkAnimation(s.resourceManager)

	// TODO(Story 6.3): 迁移到 ReanimComponent
	// 5. 替换 AnimationComponent 为走路动画
	// animComp, ok := s.entityManager.GetComponent(zombieID, reflect.TypeOf(&components.AnimationComponent{}))
	// if ok {
	// 	anim := animComp.(*components.AnimationComponent)
	// 	anim.Frames = walkFrames
	// 	anim.FrameSpeed = config.ZombieWalkFrameSpeed
	// 	anim.CurrentFrame = 0
	// 	anim.FrameCounter = 0
	// 	anim.IsLooping = true
	// 	anim.IsFinished = false
	// }
}

// handleZombieEatingBehavior 处理僵尸啃食植物的行为
// 参数:
//   - entityID: 僵尸实体ID
//   - deltaTime: 帧间隔时间
func (s *BehaviorSystem) handleZombieEatingBehavior(entityID ecs.EntityID, deltaTime float64) {
	// DEBUG: 添加日志确认函数被调用
	log.Printf("[BehaviorSystem] 🍴 处理僵尸 %d 啃食行为", entityID)

	// 检查生命值并更新受伤状态（掉手臂、掉头）
	health, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, entityID)
	if ok {

		// 更新僵尸的受伤状态（掉手臂）
		s.updateZombieDamageState(entityID, health)

		// 检查生命值是否归零（即使在啃食状态也要检查）
		if health.CurrentHealth <= 0 {
			log.Printf("[BehaviorSystem] 啃食中的僵尸 %d 生命值 <= 0 (HP=%d)，触发死亡", entityID, health.CurrentHealth)
			s.triggerZombieDeath(entityID)
			return
		}
	}

	// Story 5.3: 检查护甲状态（护甲僵尸即使在啃食也需要检测护甲破坏）
	armor, hasArmor := ecs.GetComponent[*components.ArmorComponent](s.entityManager, entityID)
	if hasArmor {
		// TODO(Story 6.3): 迁移到 ReanimComponent
		// 如果护甲已破坏，切换为普通僵尸动画
		// if armor.CurrentArmor <= 0 {
		// 	// 加载普通僵尸啃食动画
		// 	normalEatFrames := utils.LoadZombieEatAnimation(s.resourceManager)
		// 	animComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.AnimationComponent{}))
		// 	if ok {
		// 		anim := animComp.(*components.AnimationComponent)
		// 		// 检查是否已经是普通僵尸动画(避免重复切换)
		// 		if len(anim.Frames) != config.ZombieEatAnimationFrames {
		// 			anim.Frames = normalEatFrames
		// 			anim.CurrentFrame = 0
		// 			anim.FrameCounter = 0
		// 			log.Printf("[BehaviorSystem] 啃食中的护甲僵尸 %d 护甲耗尽，切换为普通僵尸啃食动画", entityID)
		// 		}
		// 	}
		// }
		_ = armor // 临时避免未使用警告
	}

	// 获取僵尸的 TimerComponent
	timer, ok := ecs.GetComponent[*components.TimerComponent](s.entityManager, entityID)
	if !ok {
		// 没有计时器，恢复移动
		s.stopEatingAndResume(entityID)
		return
	}

	// 更新计时器
	timer.CurrentTime += deltaTime

	// 检查计时器是否完成
	if timer.CurrentTime >= timer.TargetTime {
		timer.IsReady = true
	}

	// 如果计时器完成，造成伤害
	if timer.IsReady {
		// 获取僵尸当前网格位置
		pos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		if !ok {
			return
		}

		// 计算僵尸所在格子
		// 注意：需要减去 ZombieVerticalOffset，因为僵尸Y坐标包含了偏移
		zombieCol := int((pos.X - config.GridWorldStartX) / config.CellWidth)
		zombieRow := int((pos.Y - config.GridWorldStartY - config.ZombieVerticalOffset - config.CellHeight/2.0) / config.CellHeight)

		// 检测植物
		plantID, hasPlant := s.detectPlantCollision(zombieRow, zombieCol)

		if hasPlant {
			// 植物存在，造成伤害
			plantHealth, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, plantID)
			if ok {
				plantHealth.CurrentHealth -= config.ZombieEatingDamage

				log.Printf("[BehaviorSystem] 僵尸 %d 啃食植物 %d，造成 %d 伤害，剩余生命值 %d",
					entityID, plantID, config.ZombieEatingDamage, plantHealth.CurrentHealth)

				// 播放啃食音效
				s.playEatingSound()

				// 检查植物是否死亡
				if plantHealth.CurrentHealth <= 0 {
					log.Printf("[BehaviorSystem] 植物 %d 被吃掉，删除实体", plantID)

					// Bug Fix: 释放网格占用状态，允许重新种植
					if plantComp, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, plantID); ok {
						err := s.lawnGridSystem.ReleaseCell(s.lawnGridEntityID, plantComp.GridCol, plantComp.GridRow)
						if err != nil {
							log.Printf("[BehaviorSystem] 警告：释放网格占用失败: %v", err)
						} else {
							log.Printf("[BehaviorSystem] 网格 (%d, %d) 已释放", plantComp.GridCol, plantComp.GridRow)
						}
					}

					s.entityManager.DestroyEntity(plantID)
					// 恢复僵尸移动
					s.stopEatingAndResume(entityID)
					return
				}
			} else {
				// 植物没有 HealthComponent（不应该发生，但作为保护措施）
				log.Printf("[BehaviorSystem] 警告：植物 %d 没有 HealthComponent，直接删除", plantID)
				s.entityManager.DestroyEntity(plantID)
				s.stopEatingAndResume(entityID)
				return
			}
		} else {
			// 植物不存在（可能被其他僵尸吃掉），恢复移动
			s.stopEatingAndResume(entityID)
			return
		}

		// 重置计时器
		timer.CurrentTime = 0
		timer.IsReady = false
	}
}

// playEatingSound 播放僵尸啃食音效
func (s *BehaviorSystem) playEatingSound() {
	// 加载啃食音效
	eatingSound, err := s.resourceManager.LoadSoundEffect(config.ZombieEatingSoundPath)
	if err != nil {
		// 音效加载失败时不阻止游戏继续运行
		return
	}

	// 重置播放器位置到开头
	eatingSound.Rewind()

	// 播放音效
	eatingSound.Play()
}

// handleWallnutBehavior 处理坚果墙的行为逻辑
// 坚果墙没有主动行为（不生产阳光，不攻击），但会根据生命值百分比切换外观状态
// 外观状态：完好(>66%) → 轻伤(33-66%) → 重伤(<33%)
func (s *BehaviorSystem) handleWallnutBehavior(entityID ecs.EntityID) {
	// 获取生命值组件
	health, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 计算生命值百分比
	healthPercent := float64(health.CurrentHealth) / float64(health.MaxHealth)

	// Story 6.3: 使用 ReanimComponent 实现外观状态切换
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

// handleConeheadZombieBehavior 处理路障僵尸的行为逻辑
// 路障僵尸拥有护甲层，护甲耗尽后切换为普通僵尸外观和行为
func (s *BehaviorSystem) handleConeheadZombieBehavior(entityID ecs.EntityID, deltaTime float64) {
	// Story 8.3: 检查僵尸是否已激活（开场动画期间僵尸未激活，不应移动）
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

				// 2. Story 6.3: 从可见轨道列表中移除路障
				reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
				if ok {
					if reanim.VisibleTracks != nil {
						delete(reanim.VisibleTracks, "anim_cone") // 移除路障
						log.Printf("[BehaviorSystem] 路障僵尸 %d 移除 anim_cone 轨道", entityID)
					}
				}

				// 3. 移除护甲组件（可选，但保留可能对调试有帮助）
				// ecs.RemoveComponent[*components.ArmorComponent](s.entityManager, entityID)
			}
		}

		// 护甲已破坏，继续以普通僵尸行为运作
		s.handleZombieBasicBehavior(entityID, deltaTime)
		return
	}

	// 护甲完好，执行普通僵尸的基本行为（移动、碰撞检测、啃食植物）
	s.handleZombieBasicBehavior(entityID, deltaTime)
}

// handleBucketheadZombieBehavior 处理铁桶僵尸的行为逻辑
// 铁桶僵尸拥有更高的护甲层，护甲耗尽后切换为普通僵尸外观和行为
func (s *BehaviorSystem) handleBucketheadZombieBehavior(entityID ecs.EntityID, deltaTime float64) {
	// Story 8.3: 检查僵尸是否已激活（开场动画期间僵尸未激活，不应移动）
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

				// 2. Story 6.3: 从可见轨道列表中移除铁桶
				reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
				if ok {
					if reanim.VisibleTracks != nil {
						delete(reanim.VisibleTracks, "anim_bucket") // 移除铁桶
						log.Printf("[BehaviorSystem] 铁桶僵尸 %d 移除 anim_bucket 轨道", entityID)
					}
				}

				// 3. 移除护甲组件（可选，但保留可能对调试有帮助）
				// ecs.RemoveComponent[*components.ArmorComponent](s.entityManager, entityID)
			}
		}

		// 护甲已破坏，继续以普通僵尸行为运作
		s.handleZombieBasicBehavior(entityID, deltaTime)
		return
	}

	// 护甲完好，执行普通僵尸的基本行为（移动、碰撞检测、啃食植物）
	s.handleZombieBasicBehavior(entityID, deltaTime)
}

// handleCherryBombBehavior 处理樱桃炸弹的行为逻辑
// 樱桃炸弹种植后开始引信倒计时（1.5秒），倒计时结束后触发爆炸
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

// triggerCherryBombExplosion 触发樱桃炸弹爆炸
// 对以自身为中心的3x3范围内的所有僵尸造成1800点伤害，播放音效，删除樱桃炸弹实体
// 修复: 使用世界坐标进行爆炸范围检测，确保边缘网格的爆炸范围可以覆盖网格外的僵尸
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

	// Story 7.4: 创建爆炸粒子效果
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

	// Bug Fix: 释放樱桃炸弹占用的网格，允许重新种植
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

// ============================================================================
// 实体查询辅助函数（封装复杂的查询逻辑）
// ============================================================================

// queryPlants 查询所有植物实体
//
// 返回所有拥有 BehaviorComponent, PlantComponent, PositionComponent 的实体
func (s *BehaviorSystem) queryPlants() []ecs.EntityID {
	return ecs.GetEntitiesWith3[
		*components.BehaviorComponent,
		*components.PlantComponent,
		*components.PositionComponent,
	](s.entityManager)
}

// queryMovingZombies 查询所有移动中的僵尸实体
//
// 返回所有拥有 VelocityComponent 且 BehaviorType 为僵尸类型的实体
// 注意：排除子弹（BehaviorPeaProjectile）
func (s *BehaviorSystem) queryMovingZombies() []ecs.EntityID {
	// 查询所有拥有 BehaviorComponent, PositionComponent, VelocityComponent 的实体
	candidates := ecs.GetEntitiesWith3[
		*components.BehaviorComponent,
		*components.PositionComponent,
		*components.VelocityComponent,
	](s.entityManager)

	// 过滤出真正的僵尸（排除子弹和其他实体）
	var zombies []ecs.EntityID
	for _, entityID := range candidates {
		behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 只保留僵尸类型的实体
		if s.isZombieBehaviorType(behaviorComp.Type) {
			zombies = append(zombies, entityID)
		}
	}

	return zombies
}

// queryEatingZombies 查询所有啃食中的僵尸实体
//
// 返回所有处于啃食状态的僵尸（BehaviorType == BehaviorZombieEating）
func (s *BehaviorSystem) queryEatingZombies() []ecs.EntityID {
	// 查询所有拥有 BehaviorComponent, PositionComponent, TimerComponent 的实体
	// 注意：这个查询会同时匹配啃食僵尸和豌豆射手植物（植物也有 TimerComponent）
	candidates := ecs.GetEntitiesWith3[
		*components.BehaviorComponent,
		*components.PositionComponent,
		*components.TimerComponent,
	](s.entityManager)

	// 过滤出真正处于啃食状态的僵尸
	var eatingZombies []ecs.EntityID
	for _, entityID := range candidates {
		behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		if behaviorComp.Type == components.BehaviorZombieEating {
			eatingZombies = append(eatingZombies, entityID)
		}
	}

	return eatingZombies
}

// queryDyingZombies 查询所有死亡中的僵尸实体
//
// 返回所有处于死亡状态的僵尸（BehaviorType == BehaviorZombieDying）
// 死亡状态的僵尸已移除 VelocityComponent，但保留 ReanimComponent（播放死亡动画）
func (s *BehaviorSystem) queryDyingZombies() []ecs.EntityID {
	// 查询所有拥有 BehaviorComponent, PositionComponent, ReanimComponent 的实体
	candidates := ecs.GetEntitiesWith3[
		*components.BehaviorComponent,
		*components.PositionComponent,
		*components.ReanimComponent,
	](s.entityManager)

	// 过滤出真正处于死亡状态的僵尸
	var dyingZombies []ecs.EntityID
	for _, entityID := range candidates {
		behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		if behaviorComp.Type == components.BehaviorZombieDying {
			dyingZombies = append(dyingZombies, entityID)
		}
	}

	return dyingZombies
}

// queryProjectiles 查询所有豌豆子弹实体
//
// 返回所有 BehaviorType 为 BehaviorPeaProjectile 的实体
func (s *BehaviorSystem) queryProjectiles() []ecs.EntityID {
	// 查询所有拥有 BehaviorComponent, PositionComponent, VelocityComponent 的实体
	// 注意：子弹和移动中的僵尸组件组合相同，需要通过 BehaviorType 区分
	candidates := ecs.GetEntitiesWith3[
		*components.BehaviorComponent,
		*components.PositionComponent,
		*components.VelocityComponent,
	](s.entityManager)

	// 过滤出子弹
	var projectiles []ecs.EntityID
	for _, entityID := range candidates {
		behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		if behaviorComp.Type == components.BehaviorPeaProjectile {
			projectiles = append(projectiles, entityID)
		}
	}

	return projectiles
}

// isZombieBehaviorType 判断行为类型是否为僵尸类型
//
// 参数:
//   - behaviorType: 行为类型
//
// 返回:
//   - true: 是僵尸类型
//   - false: 不是僵尸类型
func (s *BehaviorSystem) isZombieBehaviorType(behaviorType components.BehaviorType) bool {
	switch behaviorType {
	case components.BehaviorZombieBasic,
		components.BehaviorZombieConehead,
		components.BehaviorZombieBuckethead,
		components.BehaviorZombieEating,
		components.BehaviorZombieDying:
		return true
	default:
		return false
	}
}

// ============================================================================
// Story 10.3: 植物攻击动画系统（重新激活 - 2025-10-24）
// ============================================================================
//
// 正确实现：使用简单的 PlayAnimation() 切换，依赖 VisibleTracks 机制显示完整身体
//
// 核心逻辑：
// - ✅ 发射子弹时切换到 anim_shooting
// - ✅ 攻击动画完成后自动切换回 anim_idle
// - ✅ 与僵尸动画实现保持一致（所有实体使用简单切换）
//

// updatePlantAttackAnimation 检测攻击动画是否完成，自动切换回 idle
// Story 10.3: 实现攻击动画状态机（Idle ↔ Attacking）
// Story 10.5: 添加关键帧事件监听，在精确时刻发射子弹
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

	// Story 10.5: 关键帧事件监听 - 子弹发射时机同步
	if plant.PendingProjectile {
		// 精确匹配发射帧（零延迟）
		if reanim.CurrentFrame == config.PeashooterShootingFireFrame {
			log.Printf("[BehaviorSystem] 豌豆射手 %d 到达关键帧(%d)，发射子弹！",
				entityID, reanim.CurrentFrame)

			// Story 10.5: 使用固定偏移值计算子弹发射位置
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

	// Story 10.3: 检查攻击动画是否播放完毕，切换回 idle
	if reanim.IsFinished {
		// 切换回空闲动画
		// 根据植物类型选择正确的空闲动画
		idleAnimName := "anim_idle"
		if plant.PlantType == components.PlantPeashooter {
			// 豌豆射手使用 anim_full_idle（包含头部）
			idleAnimName = "anim_full_idle"
		}

		err := s.reanimSystem.PlayAnimation(entityID, idleAnimName)
		if err != nil {
			log.Printf("[BehaviorSystem] 切换回空闲动画失败: %v", err)
		} else {
			plant.AttackAnimState = components.AttackAnimIdle
			log.Printf("[BehaviorSystem] 植物 %d 攻击动画完成，切换回空闲动画 '%s'", entityID, idleAnimName)
		}
	}
}
