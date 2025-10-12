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

	// 查询所有拥有 BehaviorComponent, PositionComponent, VelocityComponent 的实体（僵尸）
	zombieEntityList := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.BehaviorComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.VelocityComponent{}),
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
		default:
			// 未知僵尸类型，忽略
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
