package systems

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/utils"
)

// SoddingSystem 管理铺草皮动画系统
// Story 8.2 优化：基于草皮卷图片中心的简化定位逻辑
// Story 11.4 扩展：支持土粒飞溅粒子特效
// 重构简化：所有叠加层从 (0, 0) 开始，草皮卷中心自动对齐到可见边缘
//
// 功能：
//   - 播放 SodRoll.reanim 草皮滚动动画（2.17秒，52帧 @ 24fps）
//   - 草皮卷的 Reanim 包围盒中心 = 草皮可见宽度
//   - 可选：播放土粒飞溅粒子特效（SodRoll.xml 粒子配置）
//
// 定位原理：
//   - Position.X: 草皮卷实体位置（从 0 向右移动到草皮宽度）
//   - Position.Y: 行中心Y坐标
//   - Reanim 包围盒中心自动对齐到 Position
//   - 草皮可见宽度 = Position.X（自动从 Reanim 渲染计算）
type SoddingSystem struct {
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager

	// 动画实体ID
	sodRollEntityIDs   []ecs.EntityID // Story 8.6 QA修正: 每行一个草皮卷实体
	isAnimationPlaying bool           // 是否正在播放动画
	animationTimer     float64        // 动画计时器
	animationDuration  float64        // 动画总时长（秒）
	animationStarted   bool           // 动画是否已经启动过（包括已完成的）

	// 动画定位参数（基于网格坐标 + 可配置偏移）
	animStartX float64 // 动画起点X（世界坐标）- 从配置计算
	animLanes  []int   // Story 8.6 QA修正: 需要播放动画的行列表

	// 缓存最后一帧的中心位置（避免动画完成后跳跃）
	lastFrameCenterX float64 // 动画结束时的实际中心X

	// 缓存最后一帧的边缘位置（用于调试绘制）
	lastFrameLeftEdge  float64
	lastFrameRightEdge float64

	// 动画完成回调
	onAnimationComplete func() // 动画完成时调用

	// 粒子发射器相关
	sodRollEmitterIDs []ecs.EntityID // Story 8.6 QA修正: 每行一个粒子发射器
	particlesEnabled  bool           // 是否启用粒子特效
}

// NewSoddingSystem 创建铺草皮动画系统
func NewSoddingSystem(entityManager *ecs.EntityManager, rm *game.ResourceManager) *SoddingSystem {
	return &SoddingSystem{
		entityManager:      entityManager,
		resourceManager:    rm,
		isAnimationPlaying: false,
		animationTimer:     0,
		animationDuration:  0, // 将在 StartAnimation 时从 reanim 数据计算
	}
}

// StartAnimation 开始播放铺草皮动画
// 参数:
//   - onComplete: 动画完成时的回调函数
//   - enabledLanes: 启用的行列表(如 [2,3,4])
//   - animLanes: 播放动画的行列表(如 [2,4],空表示使用 enabledLanes)
//   - sodOverlayX: 草皮叠加图的世界X坐标（重构简化：现在固定为 0）
//   - sodImageHeight: 草皮图片的实际高度（重构简化：现在不使用此参数）
//   - enableParticles: 是否启用土粒飞溅粒子特效
func (s *SoddingSystem) StartAnimation(onComplete func(), enabledLanes, animLanes []int, sodOverlayX, sodImageHeight float64, enableParticles bool) {
	if s.isAnimationPlaying {
		log.Printf("[SoddingSystem] Animation already playing, ignoring")
		return
	}

	// Story 8.6 QA修正: 如果未指定动画行,使用所有启用的行
	if len(animLanes) == 0 {
		animLanes = enabledLanes
	}

	log.Printf("[SoddingSystem] Starting SodRoll animation for lanes %v (enabled lanes: %v)", animLanes, enabledLanes)
	s.onAnimationComplete = onComplete
	s.isAnimationPlaying = true
	s.animationStarted = true // 标记动画已经启动
	s.animationTimer = 0
	s.animLanes = animLanes // 保存动画行列表

	// Story 10.9: 播放铺草皮音效 (gravebusterchomp.ogg)
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_GRAVEBUSTERCHOMP")
		log.Printf("[SoddingSystem] 播放铺草皮音效: SOUND_GRAVEBUSTERCHOMP")
	}

	// 加载 SodRoll Reanim 资源
	reanimXML := s.resourceManager.GetReanimXML("SodRoll")
	if reanimXML == nil {
		log.Printf("[SoddingSystem] ERROR: Failed to load SodRoll reanim")
		return
	}

	// 从 reanim 数据计算动画时长
	maxFrames := 0
	for _, track := range reanimXML.Tracks {
		if len(track.Frames) > maxFrames {
			maxFrames = len(track.Frames)
		}
	}

	fps := reanimXML.FPS
	if fps == 0 {
		fps = 12 // PVZ 默认 FPS
	}

	s.animationDuration = float64(maxFrames) / float64(fps)

	// 草皮卷从网格起点开始滚动
	s.animStartX = config.GridWorldStartX + config.SodRollStartOffsetX
	log.Printf("[SoddingSystem] animStartX = %.1f (GridWorldStartX=%.1f + SodRollStartOffsetX=%.1f)",
		s.animStartX, config.GridWorldStartX, config.SodRollStartOffsetX)

	// Story 8.6 QA修正: 为每个动画行创建独立的草皮卷实体
	s.sodRollEntityIDs = make([]ecs.EntityID, 0, len(animLanes))

	for _, lane := range animLanes {
		// 验证行号合法性（1-5）
		if lane < 1 || lane > 5 {
			log.Printf("[SoddingSystem] Warning: 动画行 %d 超出范围 [1,5],跳过", lane)
			continue
		}

		// 验证行是否在启用列表中
		found := false
		for _, enabledLane := range enabledLanes {
			if enabledLane == lane {
				found = true
				break
			}
		}
		if !found {
			log.Printf("[SoddingSystem] Warning: 动画行 %d 不在启用行列表 %v 中,跳过", lane, enabledLanes)
			continue
		}

		// 重构简化: 计算此行的Y坐标（行中心）
		// 草皮卷的 Reanim 包围盒中心会自动对齐到 Position
		targetCenterY := config.GridWorldStartY + float64(lane-1)*config.CellHeight + config.CellHeight/2.0 + config.SodRollOffsetY

		log.Printf("[SoddingSystem] 创建第 %d 行草皮卷: Position=(0, %.1f)", lane, targetCenterY)

		// 创建草皮卷实体
		entityID := s.createSodRollEntity(s.animStartX, targetCenterY, lane)
		if entityID != 0 {
			s.sodRollEntityIDs = append(s.sodRollEntityIDs, entityID)
		}
	}

	log.Printf("[SoddingSystem] Created %d SodRoll entities for lanes %v", len(s.sodRollEntityIDs), animLanes)

	// 如果启用粒子特效,为每个草皮卷创建独立的粒子发射器
	// Story 8.6 QA修正: 每行一个粒子发射器
	if enableParticles {
		s.sodRollEmitterIDs = make([]ecs.EntityID, 0, len(s.sodRollEntityIDs))
		for i, entityID := range s.sodRollEntityIDs {
			emitterID := s.createSodRollParticleEmitterForEntity(entityID, animLanes[i])
			if emitterID != 0 {
				s.sodRollEmitterIDs = append(s.sodRollEmitterIDs, emitterID)
			}
		}
	}
}

// createSodRollEntity 创建草皮卷动画实体
// Story 8.6 QA修正: 为每个行创建独立的草皮卷
// 参数:
//   - posX: 实体的世界X坐标
//   - posY: 实体的世界Y坐标(对齐到行中心)
//   - lane: 行号(用于日志)
//
// 返回: 创建的实体ID,失败返回0
func (s *SoddingSystem) createSodRollEntity(posX, posY float64, lane int) ecs.EntityID {
	// 加载 SodRoll Reanim 资源
	reanimXML := s.resourceManager.GetReanimXML("SodRoll")
	partImages := s.resourceManager.GetReanimPartImages("SodRoll")

	if reanimXML == nil || partImages == nil {
		log.Printf("[SoddingSystem] ERROR: Failed to load SodRoll reanim resources for lane %d", lane)
		log.Printf("[SoddingSystem] ReanimXML: %v, PartImages: %v", reanimXML != nil, partImages != nil)
		return 0
	}

	log.Printf("[SoddingSystem] Creating SodRoll entity for lane %d at (%.1f, %.1f)", lane, posX, posY)

	// 创建实体
	entityID := s.entityManager.CreateEntity()

	// 添加位置组件
	ecs.AddComponent(s.entityManager, entityID, &components.PositionComponent{
		X: posX,
		Y: posY,
	})

	// 添加 ReanimComponent
	// 使用新的简化结构
	// 重要：MergedTracks 设置为 nil，让 PlayCombo 自动初始化
	ecs.AddComponent(s.entityManager, entityID, &components.ReanimComponent{
		ReanimXML:         reanimXML,
		PartImages:        partImages,
		MergedTracks:      nil,        // nil 让 PlayCombo 自动初始化
		CurrentAnimations: []string{}, // 初始为空，等待初始化
		IsLooping:         false,      // 不循环播放
		IsFinished:        false,
		AnimationFPS:      float64(reanimXML.FPS),
	})

	// 添加生命周期组件（动画持续约2.2秒）
	ecs.AddComponent(s.entityManager, entityID, &components.LifetimeComponent{
		MaxLifetime:     s.animationDuration + 0.1, // 略长于动画时间
		CurrentLifetime: 0,
		IsExpired:       false,
	})

	// 初始化 SodRoll 动画
	// 使用配置驱动的 PlayCombo API
	// 配置文件: data/reanim_config/sodroll.yaml
	// 使用 AnimationCommand 组件触发动画
	ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
		UnitID:    "sodroll",
		ComboName: "roll",
		Processed: false,
	})
	log.Printf("[SoddingSystem] SodRoll animation command added for lane %d", lane)

	return entityID
}

// Update 更新铺草皮动画
func (s *SoddingSystem) Update(deltaTime float64) {
	if !s.isAnimationPlaying {
		return
	}

	// Story 11.4 DEBUG: 跟踪动画进度
	if s.animationTimer < 0.5 { // 前0.5秒
		log.Printf("[SoddingSystem] Update: deltaTime=%.3f, animationTimer=%.3f, animationDuration=%.3f",
			deltaTime, s.animationTimer, s.animationDuration)
	}

	s.animationTimer += deltaTime

	// 更新粒子发射器位置(跟随草皮卷)
	// 注意：SodRoll.xml 中已配置 SystemPosition 字段，粒子系统会自动处理位置动画
	// 但我们仍然可以手动同步以确保精确跟随草皮卷实际位置（可选）
	// 目前依赖 XML 配置的 SystemPosition 自动插值

	// 检查动画是否完成
	if s.animationTimer >= s.animationDuration {
		log.Printf("[SoddingSystem] 动画完成条件触发: animationTimer=%.3f >= animationDuration=%.3f",
			s.animationTimer, s.animationDuration)
		s.completeAnimation()
	}
}

// completeAnimation 完成动画并清理
func (s *SoddingSystem) completeAnimation() {
	log.Printf("[SoddingSystem] Animation complete, cleaning up entities (animationTimer=%.3f)", s.animationTimer)

	// 保存最后一帧的中心位置和边缘位置（避免完成后跳跃）
	// 在标记实体过期之前读取位置
	if len(s.sodRollEntityIDs) > 0 {
		s.lastFrameCenterX = s.calculateCurrentCenterX()
		// 同时缓存边缘位置
		leftEdge, centerEdge, rightEdge := s.calculateCurrentEdges()
		s.lastFrameLeftEdge = leftEdge
		s.lastFrameRightEdge = rightEdge
		log.Printf("[SoddingSystem] 缓存最后一帧位置: 中心=%.1f, 左=%.1f, 中边=%.1f, 右=%.1f",
			s.lastFrameCenterX, s.lastFrameLeftEdge, centerEdge, s.lastFrameRightEdge)

		// Story 8.6 QA修正: 标记所有草皮卷实体为过期
		for _, entityID := range s.sodRollEntityIDs {
			if lifetime, ok := ecs.GetComponent[*components.LifetimeComponent](s.entityManager, entityID); ok {
				lifetime.IsExpired = true
			}
		}
		s.sodRollEntityIDs = nil
	}

	// 停止粒子发射器(但不立即销毁,等粒子自然消失)
	// Story 8.6 QA修正: 停止所有粒子发射器
	if s.particlesEnabled && len(s.sodRollEmitterIDs) > 0 {
		for _, emitterID := range s.sodRollEmitterIDs {
			if emitterComp, ok := ecs.GetComponent[*components.EmitterComponent](s.entityManager, emitterID); ok {
				emitterComp.Active = false
			}

			// 添加延迟清理(粒子最大生命周期 = 0.25秒)
			if lifetime, ok := ecs.GetComponent[*components.LifetimeComponent](s.entityManager, emitterID); ok {
				lifetime.MaxLifetime = 0.25 + 0.1 // 粒子最大生命周期 + 缓冲
				lifetime.CurrentLifetime = 0
			} else {
				// 如果没有 LifetimeComponent,添加一个
				ecs.AddComponent(s.entityManager, emitterID, &components.LifetimeComponent{
					MaxLifetime:     0.35,
					CurrentLifetime: 0,
					IsExpired:       false,
				})
			}
		}
		log.Printf("[SoddingSystem] Stopped %d particle emitters", len(s.sodRollEmitterIDs))

		s.sodRollEmitterIDs = nil
		s.particlesEnabled = false
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

// HasStarted 返回动画是否已经启动过(包括正在播放和已完成)
// Story 8.6 QA修正: 用于判断预渲染草皮是否应该显示
func (s *SoddingSystem) HasStarted() bool {
	return s.animationStarted
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

// GetSodRollCenterX 返回草皮可见区域的右边缘X坐标（世界坐标）
// 用于同步草皮叠加层的显示（叠加层从起点显示到这个位置）
//
// 定位原理：
//   - 追踪 SodRoll 轨道的X坐标和sx缩放值
//   - 计算草皮卷的左边缘 = SodRoll.X - (图片宽度 * sx) / 2
//   - 草皮可见右边缘 = 实体Position.X + 草皮卷左边缘
//   - 草皮叠加层应该裁剪显示：从世界坐标0到此返回值
func (s *SoddingSystem) GetSodRollCenterX() float64 {
	// 动画未启动：返回起点（草皮不可见）
	if !s.animationStarted {
		return s.animStartX
	}

	// 动画已完成：返回缓存的最后一帧中心位置（避免跳跃）
	if !s.isAnimationPlaying {
		return s.lastFrameCenterX
	}

	// 动画进行中：读取实际位置
	return s.calculateCurrentCenterX()
}

// GetSodRollEdges 返回草皮卷的左、中、右边缘X坐标（世界坐标）
// 用于调试绘制
// 返回值: leftEdge, centerX, rightEdge
func (s *SoddingSystem) GetSodRollEdges() (float64, float64, float64) {
	// 动画未启动：返回起点
	if !s.animationStarted {
		return s.animStartX, s.animStartX, s.animStartX
	}

	// 动画已完成：返回缓存的最后一帧位置
	if !s.isAnimationPlaying {
		return s.lastFrameLeftEdge, s.lastFrameCenterX, s.lastFrameRightEdge
	}

	// 动画进行中：计算实时位置
	return s.calculateCurrentEdges()
}

// calculateCurrentEdges 计算草皮卷当前帧的左、中、右边缘（辅助函数）
func (s *SoddingSystem) calculateCurrentEdges() (float64, float64, float64) {
	// 检查是否有草皮卷实体
	if len(s.sodRollEntityIDs) == 0 {
		return s.animStartX, s.animStartX, s.animStartX
	}

	// 使用第一个草皮卷实体
	entityID := s.sodRollEntityIDs[0]

	// 获取 ReanimComponent
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return s.animStartX, s.animStartX, s.animStartX
	}

	// 获取实体Position
	posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		return s.animStartX, s.animStartX, s.animStartX
	}

	// 查找 SodRoll 轨道
	var sodRollX *float64
	var sodRollSX *float64
	frameIndex := reanimComp.CurrentFrame

	for _, track := range reanimComp.ReanimXML.Tracks {
		if track.Name == "SodRoll" {
			if frameIndex >= len(track.Frames) {
				frameIndex = len(track.Frames) - 1
			}
			if frameIndex < 0 {
				break
			}

			frame := track.Frames[frameIndex]
			if frame.X != nil {
				sodRollX = frame.X
			}
			if frame.ScaleX != nil {
				sodRollSX = frame.ScaleX
			}
			break
		}
	}

	if sodRollX == nil {
		return s.animStartX, s.animStartX, s.animStartX
	}

	// 计算缩放后的半宽
	sodRollImageWidth := 68.0
	scaleX := 1.0
	if sodRollSX != nil {
		scaleX = *sodRollSX
	}
	scaledHalfWidth := (sodRollImageWidth * scaleX) / 2.0

	// 计算左、中、右边缘（相对坐标）
	// 注意：渲染系统直接把 SodRoll.X 当作图片左上角使用
	// 所以这里不需要转换，直接用 SodRoll.X
	leftEdge := *sodRollX                      // 图片左边缘 = SodRoll.X（渲染系统当作左上角）
	centerX := *sodRollX + scaledHalfWidth     // 图片中心
	rightEdge := *sodRollX + scaledHalfWidth*2 // 图片右边缘

	// 转换为世界坐标（使用坐标转换工具库）
	worldLeftEdgeX, _, err := utils.ReanimLocalToWorld(s.entityManager, entityID, posComp, leftEdge, 0)
	if err != nil {
		// 实体没有 ReanimComponent（理论上不会到这里）
		return 0, 0, 0
	}
	worldCenterX, _, err := utils.ReanimLocalToWorld(s.entityManager, entityID, posComp, centerX, 0)
	if err != nil {
		return 0, 0, 0
	}
	worldRightEdgeX, _, err := utils.ReanimLocalToWorld(s.entityManager, entityID, posComp, rightEdge, 0)
	if err != nil {
		return 0, 0, 0
	}

	return worldLeftEdgeX, worldCenterX, worldRightEdgeX
}

// calculateCurrentCenterX 计算草皮可见区域右边缘的X坐标（世界坐标）
// 通过追踪 SodRoll 轨道的X坐标和sx缩放，计算草皮卷左边缘作为草皮右边缘
// Story 8.6 QA修正: 使用第一个草皮卷实体(所有行X坐标相同)
func (s *SoddingSystem) calculateCurrentCenterX() float64 {
	// Story 8.6 QA修正: 检查是否有草皮卷实体
	if len(s.sodRollEntityIDs) == 0 {
		return s.animStartX
	}

	// 使用第一个草皮卷实体(所有行的X坐标相同)
	entityID := s.sodRollEntityIDs[0]

	// 获取 ReanimComponent
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		// 降级：返回起点
		return s.animStartX
	}

	// 获取实体Position
	posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	if !ok {
		// 降级：返回起点
		return s.animStartX
	}

	// 查找 SodRoll 轨道（草皮卷主体）
	var sodRollX *float64
	var sodRollSX *float64
	frameIndex := reanimComp.CurrentFrame

	for _, track := range reanimComp.ReanimXML.Tracks {
		// 找到名为 "SodRoll" 的轨道
		if track.Name == "SodRoll" {
			if frameIndex >= len(track.Frames) {
				frameIndex = len(track.Frames) - 1
			}
			if frameIndex < 0 {
				break
			}

			frame := track.Frames[frameIndex]
			if frame.X != nil {
				sodRollX = frame.X
			}
			if frame.ScaleX != nil {
				sodRollSX = frame.ScaleX
			}
			break
		}
	}

	// 如果没有找到 SodRoll 轨道的X坐标，返回起点
	if sodRollX == nil {
		return s.animStartX
	}

	// 获取 SodRoll 图片宽度
	sodRollImageWidth := 68.0 // SodRoll.png 的宽度

	// 计算缩放后的半宽
	scaleX := 1.0
	if sodRollSX != nil {
		scaleX = *sodRollSX
	}
	scaledHalfWidth := (sodRollImageWidth * scaleX) / 2.0

	// 草皮的右边缘应该对齐到草皮卷的中心
	// 渲染系统把 SodRoll.X 当作图片左上角，图片中心 = SodRoll.X + scaledHalfWidth
	sodRollCenterX := *sodRollX + scaledHalfWidth

	// 转换为世界坐标（使用坐标转换工具库）
	worldRightEdgeX, _, err := utils.ReanimLocalToWorld(s.entityManager, entityID, posComp, sodRollCenterX, 0)
	if err != nil {
		// 实体没有 ReanimComponent，返回起点
		return s.animStartX
	}

	return worldRightEdgeX
}

// createSodRollParticleEmitterForEntity 为指定草皮卷实体创建粒子发射器
// Story 8.6 QA修正: 每行一个粒子发射器
// 参数:
//   - entityID: 草皮卷实体ID
//   - lane: 行号（用于日志）
//
// 返回: 粒子发射器实体ID，失败返回0
func (s *SoddingSystem) createSodRollParticleEmitterForEntity(entityID ecs.EntityID, lane int) ecs.EntityID {
	// 从草皮卷实体读取位置
	particleX := s.animStartX
	particleY := 0.0

	if posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID); ok {
		particleX = posComp.X
		particleY = posComp.Y
	}

	// 应用配置的偏移量
	particleX += config.SodRollParticleOffsetX
	particleY += config.SodRollParticleOffsetY

	log.Printf("[SoddingSystem] 第 %d 行粒子发射器初始位置: X=%.1f, Y=%.1f", lane, particleX, particleY)

	// 使用粒子工厂创建发射器
	emitterID, err := entities.CreateParticleEffect(
		s.entityManager,
		s.resourceManager,
		"SodRoll", // 粒子配置名称
		particleX, // 起始位置X
		particleY, // 起始位置Y
	)

	if err != nil {
		log.Printf("[SoddingSystem] Failed to create SodRoll particle emitter for lane %d: %v", lane, err)
		return 0
	}

	s.particlesEnabled = true
	log.Printf("[SoddingSystem] SodRoll particle emitter created for lane %d at (%.1f, %.1f)", lane, particleX, particleY)
	return emitterID
}
