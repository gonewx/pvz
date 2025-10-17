package systems

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// SoddingSystem 管理铺草皮动画系统
// Story 8.2 QA改进：完整的 SodRoll 草皮滚动动画
//
// 功能：
//   - 播放 SodRoll.reanim 草皮滚动动画（2.17秒，52帧 @ 24fps）
//   - 追踪草皮卷位置，同步显示草皮叠加层
type SoddingSystem struct {
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager
	reanimSystem    *ReanimSystem

	// 动画实体ID
	sodRollEntityID    ecs.EntityID // 主草皮卷实体
	isAnimationPlaying bool         // 是否正在播放动画
	animationTimer     float64      // 动画计时器
	animationDuration  float64      // 动画总时长（秒）
	animationStarted   bool         // 动画是否已经启动过（包括已完成的）

	// 草皮卷位置追踪（用于同步草皮显示）
	sodRollStartX  float64 // 草皮卷起始X坐标（来自 SodRoll.reanim 第1帧）
	sodRollStartY  float64 // 草皮卷起始Y坐标（来自 SodRoll.reanim 第1帧）
	sodRollEndX    float64 // 草皮卷结束X坐标
	sodRollWidth   float64 // 草皮宽度（用于计算显示进度）

	// 动画完成回调
	onAnimationComplete func() // 动画完成时调用
}

// NewSoddingSystem 创建铺草皮动画系统
func NewSoddingSystem(entityManager *ecs.EntityManager, rm *game.ResourceManager, reanimSystem *ReanimSystem) *SoddingSystem {
	return &SoddingSystem{
		entityManager:      entityManager,
		resourceManager:    rm,
		reanimSystem:       reanimSystem,
		isAnimationPlaying: false,
		animationTimer:     0,
		animationDuration:  52.0 / 24.0, // 52帧 @ 24fps = 2.17秒
	}
}

// StartAnimation 开始播放铺草皮动画
// 参数：
//   - onComplete: 动画完成时的回调函数
//   - enabledLanes: 启用的行列表，如 [3] 或 [2,3,4]（用于计算动画位置）
func (s *SoddingSystem) StartAnimation(onComplete func(), enabledLanes []int) {
	if s.isAnimationPlaying {
		log.Printf("[SoddingSystem] Animation already playing, ignoring")
		return
	}

	log.Printf("[SoddingSystem] Starting SodRoll animation for lanes %v", enabledLanes)
	s.onAnimationComplete = onComplete
	s.isAnimationPlaying = true
	s.animationStarted = true  // 标记动画已经启动
	s.animationTimer = 0

	// 计算 SodRoll 实体的Position（动态适配行配置）
	posX, posY := config.CalculateSodRollPosition(enabledLanes)
	log.Printf("[SoddingSystem] Calculated SodRoll entity position: (%.1f, %.1f)", posX, posY)

	// 初始化草皮卷位置参数（用于追踪草皮显示进度）
	// 注意：这里的坐标是"最终世界坐标"，用于草皮叠加图的显示控制
	// 草皮卷动画从左向右滚动，当草皮卷到达某个X位置时，草皮叠加图在该位置之前应该显示
	//
	// 草皮叠加图从 X≈222 开始，宽771px，到X≈993
	// 草皮卷应该从草皮叠加图的起点滚到终点
	sodOverlayStartX := config.GridWorldStartX - 30.0 // 与 CalculateSodOverlayPosition 保持一致
	s.sodRollStartX = sodOverlayStartX                  // 草皮卷从草皮叠加图起点开始
	s.sodRollEndX = sodOverlayStartX + config.SodRowWidth // 滚到草皮叠加图终点
	s.sodRollWidth = config.SodRowWidth
	s.sodRollStartY = posY + config.SodRollBaseY

	log.Printf("[SoddingSystem] Sod roll will animate from X=%.1f to X=%.1f", s.sodRollStartX, s.sodRollEndX)

	// 创建 SodRoll 草皮卷实体
	s.createSodRollEntity(posX, posY)
}

// createSodRollEntity 创建草皮卷动画实体
// 参数：
//   - posX: 实体的世界X坐标（通常为0）
//   - posY: 实体的世界Y坐标（动态调整，让动画对齐目标行）
func (s *SoddingSystem) createSodRollEntity(posX, posY float64) {
	// 加载 SodRoll Reanim 资源
	reanimXML := s.resourceManager.GetReanimXML("SodRoll")
	partImages := s.resourceManager.GetReanimPartImages("SodRoll")

	if reanimXML == nil || partImages == nil {
		log.Printf("[SoddingSystem] ERROR: Failed to load SodRoll reanim resources")
		log.Printf("[SoddingSystem] ReanimXML: %v, PartImages: %v", reanimXML != nil, partImages != nil)
		return
	}

	log.Printf("[SoddingSystem] Creating SodRoll entity with %d parts", len(partImages))

	// 创建实体
	s.sodRollEntityID = s.entityManager.CreateEntity()

	// 添加位置组件
	// Position + reanim坐标 = 最终世界坐标
	// 例如：第3行时，posY=78，reanim.Y=244，最终Y=322（第3行中心）
	ecs.AddComponent(s.entityManager, s.sodRollEntityID, &components.PositionComponent{
		X: posX, // 通常为0，让reanim的X直接等于世界坐标
		Y: posY, // 动态调整，让reanim的基准Y对齐目标行
	})

	// 添加 ReanimComponent
	ecs.AddComponent(s.entityManager, s.sodRollEntityID, &components.ReanimComponent{
		Reanim:       reanimXML,
		PartImages:   partImages,
		CurrentAnim:  "", // 初始为空，等待初始化
		CurrentFrame: 0,
		IsLooping:    false, // 不循环播放
		IsFinished:   false,
	})

	// 添加生命周期组件（动画持续约2.2秒）
	ecs.AddComponent(s.entityManager, s.sodRollEntityID, &components.LifetimeComponent{
		MaxLifetime:     s.animationDuration + 0.1, // 略长于动画时间
		CurrentLifetime: 0,
		IsExpired:       false,
	})

	// 初始化场景动画（使用 InitializeSceneAnimation 不计算 CenterOffset）
	// SodRoll 是场景动画，坐标在 reanim 文件中已经定义好，不需要自动居中
	if err := s.reanimSystem.InitializeSceneAnimation(s.sodRollEntityID); err != nil {
		log.Printf("[SoddingSystem] ERROR: Failed to initialize SodRoll scene animation: %v", err)
		log.Printf("[SoddingSystem] Animation may not display correctly")
	} else {
		log.Printf("[SoddingSystem] SodRoll scene animation initialized successfully")
	}
}

// Update 更新铺草皮动画
func (s *SoddingSystem) Update(deltaTime float64) {
	if !s.isAnimationPlaying {
		return
	}

	s.animationTimer += deltaTime

	// 注意：不需要手动更新实体Position.X
	// SodRoll.reanim 会自己播放动画（x 从 10.3 到 769.7）
	// 实体Position保持为(0, posY)，最终世界坐标 = Position + reanim坐标

	// 检查动画是否完成
	if s.animationTimer >= s.animationDuration {
		s.completeAnimation()
	}
}

// completeAnimation 完成动画并清理
func (s *SoddingSystem) completeAnimation() {
	log.Printf("[SoddingSystem] Animation complete, cleaning up entities")

	// 标记草皮卷实体为过期（LifetimeSystem 会自动清理）
	if s.sodRollEntityID != 0 {
		if lifetime, ok := ecs.GetComponent[*components.LifetimeComponent](s.entityManager, s.sodRollEntityID); ok {
			lifetime.IsExpired = true
		}
		s.sodRollEntityID = 0
	}

	// 标记动画已完成
	s.isAnimationPlaying = false

	// 调用完成回调
	if s.onAnimationComplete != nil {
		log.Printf("[SoddingSystem] Calling animation complete callback")
		s.onAnimationComplete()
	}
}

// IsPlaying 返回动画是否正在播放
func (s *SoddingSystem) IsPlaying() bool {
	return s.isAnimationPlaying
}

// GetProgress 返回动画播放进度（0-1）
func (s *SoddingSystem) GetProgress() float64 {
	if s.animationDuration == 0 {
		return 0
	}
	progress := s.animationTimer / s.animationDuration
	if progress > 1.0 {
		return 1.0
	}
	return progress
}

// GetSodRollPosition 返回草皮卷当前X坐标（世界坐标）
// 用于同步草皮叠加层的显示
// 注意：返回的是"逻辑上的草皮卷位置"，用于控制草皮叠加图的显示范围
func (s *SoddingSystem) GetSodRollPosition() float64 {
	// 动画未启动：返回起点（不显示草皮）
	if !s.animationStarted {
		return s.sodRollStartX
	}

	// 动画已完成：返回终点（完整显示草皮）
	if !s.isAnimationPlaying {
		return s.sodRollEndX
	}

	// 动画进行中：根据进度计算位置
	progress := s.GetProgress()
	return s.sodRollStartX + (s.sodRollEndX-s.sodRollStartX)*progress
}
