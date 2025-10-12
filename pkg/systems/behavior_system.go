package systems

import (
	"log"
	"math/rand"
	"reflect"

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
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager
	logFrameCounter int // 日志输出计数器（避免全局变量）
}

// 阳光生产位置偏移常量
const (
	SunOffsetCenterX       = 40.0 // 阳光图像居中偏移（阳光约80px宽）
	SunOffsetBaseY         = 80.0 // 阳光基础向上偏移（向日葵上方）
	SunRandomOffsetRangeX  = 40.0 // 随机水平偏移范围（-20 到 +20）
	SunRandomOffsetRangeY  = 20.0 // 随机垂直偏移范围（-10 到 +10）
	LogOutputFrameInterval = 100  // 日志输出间隔（每N帧输出一次）
)

// NewBehaviorSystem 创建一个新的行为系统
// 参数:
//   - em: EntityManager 实例
//   - rm: ResourceManager 实例
func NewBehaviorSystem(em *ecs.EntityManager, rm *game.ResourceManager) *BehaviorSystem {
	return &BehaviorSystem{
		entityManager:   em,
		resourceManager: rm,
	}
}

// Update 更新所有拥有行为组件的实体
func (s *BehaviorSystem) Update(deltaTime float64) {
	// 查询所有拥有 BehaviorComponent, TimerComponent, PlantComponent, PositionComponent 的实体（植物）
	plantEntityList := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.BehaviorComponent{}),
		reflect.TypeOf(&components.TimerComponent{}),
		reflect.TypeOf(&components.PlantComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
	)

	// 查询所有拥有 BehaviorComponent, PositionComponent, VelocityComponent 的实体（移动中的僵尸）
	zombieEntityList := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.BehaviorComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.VelocityComponent{}),
	)

	// 查询所有死亡中的僵尸实体（没有 VelocityComponent，只有 BehaviorComponent, PositionComponent, AnimationComponent）
	dyingZombieEntityList := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.BehaviorComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.AnimationComponent{}),
	)

	// 查询所有豌豆子弹实体（拥有 BehaviorComponent, PositionComponent, VelocityComponent）
	// 注意：子弹和僵尸的组件组合相同，需要通过 BehaviorType 区分
	projectileEntityList := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.BehaviorComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.VelocityComponent{}),
	)

	// 日志输出（避免每帧都打印）
	totalEntities := len(plantEntityList) + len(zombieEntityList) + len(projectileEntityList)
	if totalEntities > 0 {
		s.logFrameCounter++
		if s.logFrameCounter%LogOutputFrameInterval == 1 {
			log.Printf("[BehaviorSystem] 更新 %d 个行为实体 (植物: %d, 僵尸: %d, 子弹: %d)",
				totalEntities, len(plantEntityList), len(zombieEntityList), len(projectileEntityList))
		}
	}

	// 遍历所有植物实体，根据行为类型分发处理
	for _, entityID := range plantEntityList {
		behaviorComp, _ := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.BehaviorComponent{}))
		behavior := behaviorComp.(*components.BehaviorComponent)

		// 根据行为类型分发
		switch behavior.Type {
		case components.BehaviorSunflower:
			s.handleSunflowerBehavior(entityID, deltaTime)
		case components.BehaviorPeashooter:
			s.handlePeashooterBehavior(entityID, deltaTime, zombieEntityList)
		default:
			// 未知行为类型，忽略
		}
	}

	// 遍历所有僵尸实体，根据行为类型分发处理
	for _, entityID := range zombieEntityList {
		behaviorComp, _ := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.BehaviorComponent{}))
		behavior := behaviorComp.(*components.BehaviorComponent)

		// 根据行为类型分发
		switch behavior.Type {
		case components.BehaviorZombieBasic:
			s.handleZombieBasicBehavior(entityID, deltaTime)
		case components.BehaviorZombieEating:
			s.handleZombieEatingBehavior(entityID, deltaTime)
		default:
			// 未知僵尸类型，忽略
			// 注意：BehaviorZombieDying 在单独的 dyingZombieEntityList 中处理
		}
	}

	// 遍历所有子弹实体，根据行为类型分发处理
	for _, entityID := range projectileEntityList {
		behaviorComp, _ := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.BehaviorComponent{}))
		behavior := behaviorComp.(*components.BehaviorComponent)

		// 根据行为类型分发
		switch behavior.Type {
		case components.BehaviorPeaProjectile:
			s.handlePeaProjectileBehavior(entityID, deltaTime)
		default:
			// 忽略非子弹类型（如僵尸）
		}
	}

	// 遍历所有死亡中的僵尸实体（处理死亡动画完成后的删除）
	for _, entityID := range dyingZombieEntityList {
		behaviorComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.BehaviorComponent{}))
		if !ok {
			continue
		}
		behavior := behaviorComp.(*components.BehaviorComponent)

		// 只处理死亡中的僵尸
		if behavior.Type == components.BehaviorZombieDying {
			s.handleZombieDyingBehavior(entityID)
		}
	}

	// 查询所有击中效果实体（拥有 BehaviorComponent 和 TimerComponent）
	hitEffectEntityList := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.BehaviorComponent{}),
		reflect.TypeOf(&components.TimerComponent{}),
	)

	// 遍历所有击中效果实体，管理其生命周期
	for _, entityID := range hitEffectEntityList {
		behaviorComp, _ := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.BehaviorComponent{}))
		behavior := behaviorComp.(*components.BehaviorComponent)

		// 只处理击中效果类型
		if behavior.Type == components.BehaviorPeaBulletHit {
			s.handleHitEffectBehavior(entityID, deltaTime)
		}
	}
}

// handleSunflowerBehavior 处理向日葵的行为逻辑
// 向日葵会定期生产阳光
func (s *BehaviorSystem) handleSunflowerBehavior(entityID ecs.EntityID, deltaTime float64) {
	// 获取计时器组件
	timerComp, _ := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.TimerComponent{}))
	timer := timerComp.(*components.TimerComponent)

	// 更新计时器
	timer.CurrentTime += deltaTime

	// 检查计时器是否完成
	if timer.CurrentTime >= timer.TargetTime {
		log.Printf("[BehaviorSystem] 向日葵生产阳光！计时器: %.2f/%.2f 秒", timer.CurrentTime, timer.TargetTime)

		// 获取位置组件，计算阳光生成位置
		positionComp, _ := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.PositionComponent{}))
		position := positionComp.(*components.PositionComponent)

		// 阳光生成位置：向日葵位置附近随机偏移
		// 向日葵生产的阳光应该从向日葵中心附近弹出
		// 添加随机偏移使其更自然，不总是在正上方
		// 注意：阳光图像是左上角对齐，尺寸约80x80

		// 随机水平偏移：-20 到 +20 像素
		randomOffsetX := (rand.Float64() - 0.5) * SunRandomOffsetRangeX // -20 ~ +20
		// 随机垂直偏移：-10 到 +10 像素
		randomOffsetY := (rand.Float64() - 0.5) * SunRandomOffsetRangeY // -10 ~ +10

		sunStartX := position.X - SunOffsetCenterX + randomOffsetX // 向左偏移居中，加上随机偏移
		sunStartY := position.Y - SunOffsetBaseY + randomOffsetY   // 向日葵上方，加上随机偏移

		log.Printf("[BehaviorSystem] 创建阳光实体，位置: (%.0f, %.0f), 随机偏移: (%.1f, %.1f)",
			sunStartX, sunStartY, randomOffsetX, randomOffsetY)

		// 创建阳光实体
		// 注意：NewSunEntity 会将 Y 坐标重置为 -50（屏幕顶部），这是为天降阳光设计的
		// 向日葵生产的阳光需要特殊处理
		sunID := entities.NewSunEntity(s.entityManager, s.resourceManager, sunStartX, position.Y)

		// 修正阳光的起始位置为向日葵位置（覆盖 NewSunEntity 中的 Y=-50）
		sunPosComp, _ := s.entityManager.GetComponent(sunID, reflect.TypeOf(&components.PositionComponent{}))
		sunPos := sunPosComp.(*components.PositionComponent)
		sunPos.Y = sunStartY

		// 修正阳光的速度：向日葵生产的阳光应该是静止的
		sunVelComp, ok := s.entityManager.GetComponent(sunID, reflect.TypeOf(&components.VelocityComponent{}))
		if ok {
			sunVel := sunVelComp.(*components.VelocityComponent)
			sunVel.VX = 0
			sunVel.VY = 0 // 静止，不下落
		}

		// 修正阳光的状态：向日葵生产的阳光直接是已落地状态（可以点击）
		sunCompComp, ok := s.entityManager.GetComponent(sunID, reflect.TypeOf(&components.SunComponent{}))
		if ok {
			sunComp := sunCompComp.(*components.SunComponent)
			sunComp.State = components.SunLanded // 直接设置为已落地状态
		}

		log.Printf("[BehaviorSystem] 修正阳光位置为: (%.0f, %.0f), 状态: Landed", sunPos.X, sunPos.Y)

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
	// 检查生命值（Story 4.4: 僵尸死亡逻辑）
	healthComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.HealthComponent{}))
	if ok {
		health := healthComp.(*components.HealthComponent)
		if health.CurrentHealth <= 0 {
			// 生命值 <= 0，触发死亡状态转换
			log.Printf("[BehaviorSystem] 僵尸 %d 生命值 <= 0 (HP=%d)，触发死亡", entityID, health.CurrentHealth)
			s.triggerZombieDeath(entityID)
			return // 跳过正常移动逻辑
		}
	}

	// 获取位置组件
	positionComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.PositionComponent{}))
	if !ok {
		return
	}
	position := positionComp.(*components.PositionComponent)

	// Story 5.1: 检测植物碰撞（在移动之前）
	// 计算僵尸所在格子
	zombieCol := int((position.X - config.GridWorldStartX) / config.CellWidth)
	zombieRow := int((position.Y - config.GridWorldStartY) / config.CellHeight)

	// 检测是否与植物在同一格子
	plantID, hasCollision := s.detectPlantCollision(zombieRow, zombieCol)
	if hasCollision {
		// 进入啃食状态
		s.startEatingPlant(entityID, plantID)
		return // 跳过移动逻辑
	}

	// 获取速度组件
	velocityComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.VelocityComponent{}))
	if !ok {
		return
	}
	velocity := velocityComp.(*components.VelocityComponent)

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
func (s *BehaviorSystem) triggerZombieDeath(entityID ecs.EntityID) {
	// 1. 切换行为类型为 BehaviorZombieDying
	behaviorComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.BehaviorComponent{}))
	if !ok {
		log.Printf("[BehaviorSystem] 僵尸 %d 缺少 BehaviorComponent，无法触发死亡", entityID)
		return
	}
	behavior := behaviorComp.(*components.BehaviorComponent)
	behavior.Type = components.BehaviorZombieDying
	log.Printf("[BehaviorSystem] 僵尸 %d 行为切换为 BehaviorZombieDying", entityID)

	// 2. 移除 VelocityComponent（停止移动）
	s.entityManager.RemoveComponent(entityID, reflect.TypeOf(&components.VelocityComponent{}))
	log.Printf("[BehaviorSystem] 僵尸 %d 移除速度组件，停止移动", entityID)

	// 3. 加载死亡动画帧
	deathFrames, err := utils.LoadZombieDeathAnimation(s.resourceManager)
	if err != nil {
		log.Printf("[BehaviorSystem] 加载僵尸死亡动画失败: %v，使用占位动画", err)
		// 错误处理：如果死亡动画加载失败，直接删除僵尸
		s.entityManager.DestroyEntity(entityID)
		return
	}

	// 4. 替换 AnimationComponent 的动画帧
	animComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.AnimationComponent{}))
	if !ok {
		log.Printf("[BehaviorSystem] 僵尸 %d 缺少 AnimationComponent，无法播放死亡动画", entityID)
		s.entityManager.DestroyEntity(entityID)
		return
	}
	anim := animComp.(*components.AnimationComponent)
	anim.Frames = deathFrames
	anim.IsLooping = false // 死亡动画不循环
	anim.IsFinished = false
	anim.CurrentFrame = 0
	anim.FrameCounter = 0
	anim.FrameSpeed = config.ZombieDieFrameSpeed

	log.Printf("[BehaviorSystem] 僵尸 %d 死亡动画已设置 (帧数=%d, 帧速率=%.2f)", entityID, len(deathFrames), config.ZombieDieFrameSpeed)
}

// handlePeashooterBehavior 处理豌豆射手的行为逻辑
// 豌豆射手会周期性扫描同行僵尸并发射豌豆子弹
func (s *BehaviorSystem) handlePeashooterBehavior(entityID ecs.EntityID, deltaTime float64, zombieEntityList []ecs.EntityID) {
	// 获取计时器组件
	timerComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.TimerComponent{}))
	if !ok {
		return
	}
	timer := timerComp.(*components.TimerComponent)

	// 更新计时器
	timer.CurrentTime += deltaTime

	// 检查计时器是否就绪（达到攻击间隔）
	if timer.CurrentTime >= timer.TargetTime {
		// 获取豌豆射手的位置组件
		positionComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.PositionComponent{}))
		if !ok {
			return
		}
		peashooterPos := positionComp.(*components.PositionComponent)

		// 计算豌豆射手所在的行
		peashooterRow := utils.GetEntityRow(peashooterPos.Y, config.GridWorldStartY, config.CellHeight)

		// 扫描同行僵尸：查找在豌豆射手正前方（右侧）且在攻击范围内的僵尸
		hasZombieInLine := false
		for _, zombieID := range zombieEntityList {
			zombiePosComp, ok := s.entityManager.GetComponent(zombieID, reflect.TypeOf(&components.PositionComponent{}))
			if !ok {
				continue
			}
			zombiePos := zombiePosComp.(*components.PositionComponent)

			// 计算僵尸所在的行
			zombieRow := utils.GetEntityRow(zombiePos.Y, config.GridWorldStartY, config.CellHeight)

			// 检查僵尸是否在同一行、在豌豆射手右侧、且在攻击范围内（草坪内）
			// 使用 config.GridWorldEndX 判断僵尸是否在草坪右边界内
			if zombieRow == peashooterRow &&
				zombiePos.X > peashooterPos.X &&
				zombiePos.X <= config.GridWorldEndX {
				hasZombieInLine = true
				// DEBUG: 只在找到目标时输出
				log.Printf("[BehaviorSystem] 发现目标僵尸 %d: 位置=(%.1f, %.1f), 豌豆射手X=%.1f, 草坪边界=%.1f",
					zombieID, zombiePos.X, zombiePos.Y, peashooterPos.X, config.GridWorldEndX)
				break
			}
		}

		// 如果有僵尸在同一行，发射子弹
		if hasZombieInLine {
			// 计算子弹起始位置：豌豆射手口部位置（世界坐标）
			bulletStartX := peashooterPos.X + config.PeaBulletOffsetX
			bulletStartY := peashooterPos.Y + config.PeaBulletOffsetY

			// 播放发射音效（如果配置了音效路径）
			s.playShootSound()

			// 创建豌豆子弹实体
			bulletID, err := entities.NewPeaProjectile(s.entityManager, s.resourceManager, bulletStartX, bulletStartY)
			if err != nil {
				log.Printf("[BehaviorSystem] 创建豌豆子弹失败: %v", err)
			} else {
				log.Printf("[BehaviorSystem] 豌豆射手 %d 发射子弹 %d，位置: (%.1f, %.1f)",
					entityID, bulletID, bulletStartX, bulletStartY)
			}

			// 重置计时器
			timer.CurrentTime = 0
		}
		// 如果没有僵尸，不发射子弹，但计时器继续累加（下次检测时立即发射）
	}
}

// handlePeaProjectileBehavior 处理豌豆子弹的移动逻辑
// 豌豆子弹会以恒定速度向右移动，飞出屏幕后被删除
func (s *BehaviorSystem) handlePeaProjectileBehavior(entityID ecs.EntityID, deltaTime float64) {
	// 获取位置组件
	positionComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.PositionComponent{}))
	if !ok {
		return
	}
	position := positionComp.(*components.PositionComponent)

	// 获取速度组件
	velocityComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.VelocityComponent{}))
	if !ok {
		return
	}
	velocity := velocityComp.(*components.VelocityComponent)

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
	timerComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.TimerComponent{}))
	if !ok {
		return
	}
	timer := timerComp.(*components.TimerComponent)

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
	// 获取动画组件
	animComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.AnimationComponent{}))
	if !ok {
		// 如果没有动画组件，直接删除僵尸
		log.Printf("[BehaviorSystem] 死亡中的僵尸 %d 缺少 AnimationComponent，直接删除", entityID)
		s.entityManager.DestroyEntity(entityID)
		return
	}
	anim := animComp.(*components.AnimationComponent)

	// 检查死亡动画是否完成
	if anim.IsFinished {
		log.Printf("[BehaviorSystem] 僵尸 %d 死亡动画完成，删除实体", entityID)
		s.entityManager.DestroyEntity(entityID)
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
	plantEntityList := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.PlantComponent{}),
	)

	// 遍历所有植物，比对网格位置
	for _, plantID := range plantEntityList {
		plantComp, ok := s.entityManager.GetComponent(plantID, reflect.TypeOf(&components.PlantComponent{}))
		if !ok {
			continue
		}

		plant := plantComp.(*components.PlantComponent)

		// 检查是否在同一格子
		if plant.GridRow == zombieRow && plant.GridCol == zombieCol {
			return plantID, true
		}
	}

	// 没有找到植物
	return 0, false
}

// startEatingPlant 开始啃食植物
// 参数:
//   - zombieID: 僵尸实体ID
//   - plantID: 植物实体ID
func (s *BehaviorSystem) startEatingPlant(zombieID, plantID ecs.EntityID) {
	log.Printf("[BehaviorSystem] 僵尸 %d 开始啃食植物 %d", zombieID, plantID)

	// 1. 移除僵尸的 VelocityComponent（停止移动）
	s.entityManager.RemoveComponent(zombieID, reflect.TypeOf(&components.VelocityComponent{}))

	// 2. 切换 BehaviorComponent.Type 为 BehaviorZombieEating
	behaviorComp, ok := s.entityManager.GetComponent(zombieID, reflect.TypeOf(&components.BehaviorComponent{}))
	if ok {
		behavior := behaviorComp.(*components.BehaviorComponent)
		behavior.Type = components.BehaviorZombieEating
	}

	// 3. 添加 TimerComponent 用于伤害间隔
	s.entityManager.AddComponent(zombieID, &components.TimerComponent{
		Name:        "eating_damage",
		TargetTime:  config.ZombieEatingDamageInterval,
		CurrentTime: 0,
		IsReady:     false,
	})

	// 4. 加载僵尸啃食动画帧序列
	eatFrames := utils.LoadZombieEatAnimation(s.resourceManager)

	// 5. 替换 AnimationComponent 为啃食动画
	animComp, ok := s.entityManager.GetComponent(zombieID, reflect.TypeOf(&components.AnimationComponent{}))
	if ok {
		anim := animComp.(*components.AnimationComponent)
		anim.Frames = eatFrames
		anim.FrameSpeed = config.ZombieEatFrameSpeed
		anim.CurrentFrame = 0
		anim.FrameCounter = 0
		anim.IsLooping = true
		anim.IsFinished = false
	}
}

// stopEatingAndResume 停止啃食并恢复移动
// 参数:
//   - zombieID: 僵尸实体ID
func (s *BehaviorSystem) stopEatingAndResume(zombieID ecs.EntityID) {
	log.Printf("[BehaviorSystem] 僵尸 %d 结束啃食，恢复移动", zombieID)

	// 1. 移除 TimerComponent
	s.entityManager.RemoveComponent(zombieID, reflect.TypeOf(&components.TimerComponent{}))

	// 2. 切换 BehaviorComponent.Type 回 BehaviorZombieBasic
	behaviorComp, ok := s.entityManager.GetComponent(zombieID, reflect.TypeOf(&components.BehaviorComponent{}))
	if ok {
		behavior := behaviorComp.(*components.BehaviorComponent)
		behavior.Type = components.BehaviorZombieBasic
	}

	// 3. 恢复 VelocityComponent
	s.entityManager.AddComponent(zombieID, &components.VelocityComponent{
		VX: config.ZombieWalkSpeed,
		VY: 0,
	})

	// 4. 加载僵尸走路动画帧序列
	walkFrames := utils.LoadZombieWalkAnimation(s.resourceManager)

	// 5. 替换 AnimationComponent 为走路动画
	animComp, ok := s.entityManager.GetComponent(zombieID, reflect.TypeOf(&components.AnimationComponent{}))
	if ok {
		anim := animComp.(*components.AnimationComponent)
		anim.Frames = walkFrames
		anim.FrameSpeed = config.ZombieWalkFrameSpeed
		anim.CurrentFrame = 0
		anim.FrameCounter = 0
		anim.IsLooping = true
		anim.IsFinished = false
	}
}

// handleZombieEatingBehavior 处理僵尸啃食植物的行为
// 参数:
//   - entityID: 僵尸实体ID
//   - deltaTime: 帧间隔时间
func (s *BehaviorSystem) handleZombieEatingBehavior(entityID ecs.EntityID, deltaTime float64) {
	// 获取僵尸的 TimerComponent
	timerComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.TimerComponent{}))
	if !ok {
		// 没有计时器，恢复移动
		s.stopEatingAndResume(entityID)
		return
	}
	timer := timerComp.(*components.TimerComponent)

	// 更新计时器
	timer.CurrentTime += deltaTime

	log.Printf("[BehaviorSystem] 僵尸 %d 啃食计时器: %.2f/%.2f 秒, IsReady=%v",
		entityID, timer.CurrentTime, timer.TargetTime, timer.IsReady)

	// 检查计时器是否完成
	if timer.CurrentTime >= timer.TargetTime {
		timer.IsReady = true
	}

	// 如果计时器完成，造成伤害
	if timer.IsReady {
		// 获取僵尸当前网格位置
		posComp, ok := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.PositionComponent{}))
		if !ok {
			return
		}
		pos := posComp.(*components.PositionComponent)

		// 计算僵尸所在格子
		zombieCol := int((pos.X - config.GridWorldStartX) / config.CellWidth)
		zombieRow := int((pos.Y - config.GridWorldStartY) / config.CellHeight)

		// 检测植物
		plantID, hasPlant := s.detectPlantCollision(zombieRow, zombieCol)

		if hasPlant {
			// 植物存在，造成伤害
			plantHealthComp, ok := s.entityManager.GetComponent(plantID, reflect.TypeOf(&components.HealthComponent{}))
			if ok {
				plantHealth := plantHealthComp.(*components.HealthComponent)
				plantHealth.CurrentHealth -= config.ZombieEatingDamage

				log.Printf("[BehaviorSystem] 僵尸 %d 啃食植物 %d，造成 %d 伤害，剩余生命值 %d",
					entityID, plantID, config.ZombieEatingDamage, plantHealth.CurrentHealth)

				// 播放啃食音效
				s.playEatingSound()

				// 检查植物是否死亡
				if plantHealth.CurrentHealth <= 0 {
					log.Printf("[BehaviorSystem] 植物 %d 被吃掉，删除实体", plantID)
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
