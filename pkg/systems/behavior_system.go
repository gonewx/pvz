package systems

import (
	"log"
	"math/rand"
	"reflect"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
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
	// 查询所有拥有 BehaviorComponent, TimerComponent, PlantComponent, PositionComponent 的实体
	entityList := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.BehaviorComponent{}),
		reflect.TypeOf(&components.TimerComponent{}),
		reflect.TypeOf(&components.PlantComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
	)

	// 只在有实体时输出日志（避免每帧都打印）
	if len(entityList) > 0 {
		// 每N帧打印一次（约每1.67秒）
		s.logFrameCounter++
		if s.logFrameCounter%LogOutputFrameInterval == 1 {
			log.Printf("[BehaviorSystem] 更新 %d 个行为实体", len(entityList))
		}
	}

	// 遍历所有实体，根据行为类型分发处理
	for _, entityID := range entityList {
		behaviorComp, _ := s.entityManager.GetComponent(entityID, reflect.TypeOf(&components.BehaviorComponent{}))
		behavior := behaviorComp.(*components.BehaviorComponent)

		// 根据行为类型分发
		switch behavior.Type {
		case components.BehaviorSunflower:
			s.handleSunflowerBehavior(entityID, deltaTime)
		case components.BehaviorPeashooter:
			// 豌豆射手行为（未来实现）
		default:
			// 未知行为类型，忽略
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
