package systems

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
)

// SoddingSystem 管理铺草皮动画系统
// Story 8.2 优化：基于草皮卷图片中心的简化定位逻辑
// Story 11.4 扩展：支持土粒飞溅粒子特效
//
// 功能：
//   - 播放 SodRoll.reanim 草皮滚动动画（2.17秒，52帧 @ 24fps）
//   - 根据动画进度线性插值计算草皮卷中心位置，同步显示草皮叠加层
//   - 可选：播放土粒飞溅粒子特效（SodRoll.xml 粒子配置）
//
// 定位原理：
//   - X轴：草皮卷图片中心从草皮左边缘线性移动到草皮右边缘
//   - Y轴：草皮卷图片中心与草皮叠加图Y中心对齐（都对齐到目标行中心）
//   - 草皮可见宽度 = 草皮卷中心X - 草皮左边缘X
type SoddingSystem struct {
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager
	reanimSystem    *ReanimSystem

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

	// 动画完成回调
	onAnimationComplete func() // 动画完成时调用

	// Story 11.4: 粒子发射器相关
	sodRollEmitterIDs []ecs.EntityID // Story 8.6 QA修正: 每行一个粒子发射器
	particlesEnabled  bool            // 是否启用粒子特效
}

// NewSoddingSystem 创建铺草皮动画系统
func NewSoddingSystem(entityManager *ecs.EntityManager, rm *game.ResourceManager, reanimSystem *ReanimSystem) *SoddingSystem {
	return &SoddingSystem{
		entityManager:      entityManager,
		resourceManager:    rm,
		reanimSystem:       reanimSystem,
		isAnimationPlaying: false,
		animationTimer:     0,
		animationDuration:  0, // 将在 StartAnimation 时从 reanim 数据计算
	}
}

// StartAnimation 开始播放铺草皮动画
// 参数:
//   - onComplete: 动画完成时的回调函数
//   - enabledLanes: 启用的行列表(如 [2,3,4])
//   - animLanes: 播放动画的行列表(如 [2,4],空表示使用 enabledLanes) - Story 8.6 QA修正
//   - sodOverlayX: 草皮叠加图的世界X坐标(左边缘)
//   - sodImageHeight: 草皮图片的实际高度
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
	// log.Printf("[SoddingSystem] 从 reanim 读取: 帧数=%d, FPS=%d, 时长=%.2f秒",
	// 	maxFrames, fps, s.animationDuration)

	// Story 8.6 QA修正: 计算动画起点X坐标(所有行共用)
	s.animStartX = config.GridWorldStartX + config.SodRollStartOffsetX

	// Story 8.6 QA修正: 为每个动画行创建独立的草皮卷实体
	s.sodRollEntityIDs = make([]ecs.EntityID, 0, len(animLanes))

	// 计算 reanim 动画的包围盒（用于Y坐标对齐）
	// 从 reanim 数据计算包围盒中心
	var minY, maxY *float64
	for _, track := range reanimXML.Tracks {
		for _, frame := range track.Frames {
			if frame.Y != nil {
				y := *frame.Y
				if minY == nil || y < *minY {
					minY = &y
				}
				if maxY == nil || y > *maxY {
					maxY = &y
				}
			}
		}
	}

	var animCenterY float64
	if minY != nil && maxY != nil {
		animCenterY = (*minY + *maxY) / 2.0
		log.Printf("[SoddingSystem] Reanim包围盒: minY=%.1f, maxY=%.1f, centerY=%.1f", *minY, *maxY, animCenterY)
	} else {
		animCenterY = 0
		log.Printf("[SoddingSystem] Warning: 无法从reanim计算包围盒,使用默认centerY=0")
	}

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

		// Story 8.6 QA修正: 计算此行的Y坐标（使用与CalculateSodRollPosition相同的逻辑）
		// 1. 计算目标行的中心Y坐标（绝对行号）
		targetCenterY := config.GridWorldStartY + float64(lane-1)*config.CellHeight + config.CellHeight/2.0

		// 2. 对齐 reanim 动画包围盒中心到目标行中心
		laneY := targetCenterY - animCenterY + config.SodRollOffsetY

		log.Printf("[SoddingSystem] 计算第 %d 行草皮卷位置: targetCenterY=%.1f, animCenterY=%.1f, offsetY=%.1f → laneY=%.1f",
			lane, targetCenterY, animCenterY, config.SodRollOffsetY, laneY)

		// 创建草皮卷实体
		entityID := s.createSodRollEntity(s.animStartX, laneY, lane)
		if entityID != 0 {
			s.sodRollEntityIDs = append(s.sodRollEntityIDs, entityID)
		}
	}

	log.Printf("[SoddingSystem] Created %d SodRoll entities for lanes %v", len(s.sodRollEntityIDs), animLanes)

	// Story 11.4: 如果启用粒子特效,为每个草皮卷创建独立的粒子发射器
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
	ecs.AddComponent(s.entityManager, entityID, &components.ReanimComponent{
		Reanim:       reanimXML,
		PartImages:   partImages,
		CurrentAnim:  "", // 初始为空，等待初始化
		CurrentFrame: 0,
		IsLooping:    false, // 不循环播放
		IsFinished:   false,
	})

	// 添加生命周期组件（动画持续约2.2秒）
	ecs.AddComponent(s.entityManager, entityID, &components.LifetimeComponent{
		MaxLifetime:     s.animationDuration + 0.1, // 略长于动画时间
		CurrentLifetime: 0,
		IsExpired:       false,
	})

	// 初始化场景动画（使用 InitializeSceneAnimation 不计算 CenterOffset）
	// SodRoll 是场景动画，坐标在 reanim 文件中已经定义好，不需要自动居中
	if err := s.reanimSystem.InitializeSceneAnimation(entityID); err != nil {
		log.Printf("[SoddingSystem] ERROR: Failed to initialize SodRoll scene animation for lane %d: %v", lane, err)
		log.Printf("[SoddingSystem] Animation may not display correctly")
	} else {
		log.Printf("[SoddingSystem] SodRoll scene animation initialized successfully for lane %d", lane)
	}

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

	// Story 11.4: 更新粒子发射器位置(跟随草皮卷)
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

	// 保存最后一帧的中心位置（避免完成后跳跃）
	// 在标记实体过期之前读取位置
	if len(s.sodRollEntityIDs) > 0 {
		s.lastFrameCenterX = s.calculateCurrentCenterX()
		log.Printf("[SoddingSystem] 缓存最后一帧中心位置: %.1f", s.lastFrameCenterX)

		// Story 8.6 QA修正: 标记所有草皮卷实体为过期
		for _, entityID := range s.sodRollEntityIDs {
			if lifetime, ok := ecs.GetComponent[*components.LifetimeComponent](s.entityManager, entityID); ok {
				lifetime.IsExpired = true
			}
		}
		s.sodRollEntityIDs = nil
	}

	// Story 11.4: 停止粒子发射器(但不立即销毁,等粒子自然消失)
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

// GetSodRollCenterX 返回草皮卷图片中心当前X坐标（世界坐标）
// 用于同步草皮叠加层的显示
//
// 定位原理：
//   - 读取 SodRoll 动画当前帧的实际位置
//   - 计算整体包围盒的中心X坐标
//   - 草皮可见宽度 = 中心X - 动画起点X
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

// GetAnimStartX 返回动画起点X坐标（世界坐标）
// 用于计算草皮可见宽度：visibleWidth = GetSodRollCenterX() - GetAnimStartX()
func (s *SoddingSystem) GetAnimStartX() float64 {
	return s.animStartX
}

// calculateCurrentCenterX 计算草皮卷当前帧的中心X坐标（世界坐标）
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

	// 计算整体包围盒的中心X坐标
	// 策略：遍历所有轨道，找到当前帧所有部件的X范围，取中心
	var minX, maxX *float64

	for _, track := range reanimComp.Reanim.Tracks {
		// 计算当前帧索引
		frameIndex := int(reanimComp.CurrentFrame)
		if frameIndex >= len(track.Frames) {
			frameIndex = len(track.Frames) - 1
		}
		if frameIndex < 0 {
			continue
		}

		frame := track.Frames[frameIndex]
		if frame.X != nil {
			x := *frame.X
			if minX == nil || x < *minX {
				minX = &x
			}
			if maxX == nil || x > *maxX {
				maxX = &x
			}
		}
	}

	// 如果没有找到任何X坐标，返回起点
	if minX == nil || maxX == nil {
		return s.animStartX
	}

	// 计算包围盒中心X（相对于实体Position）
	// 草皮卷图片右边有透明边，所以追踪中心而不是右边缘
	centerX := (*minX + *maxX) / 2.0

	// 转换为世界坐标
	worldCenterX := posComp.X + centerX

	// 调试日志（每10帧输出一次）
	// frameIndex := int(reanimComp.CurrentFrame)
	// if frameIndex%10 == 0 || frameIndex == 0 || frameIndex == len(reanimComp.Reanim.Tracks[0].Frames)-1 {
	// 	progress := s.GetProgress()
	// 	log.Printf("[SoddingSystem] 帧:%d, 进度:%.1f%%, 包围盒:[%.1f,%.1f], 中心:%.1f",
	// 		frameIndex, progress*100, *minX, *maxX, worldCenterX)
	// }

	return worldCenterX
}

// Story 11.4: createSodRollParticleEmitterForEntity 为指定草皮卷实体创建粒子发射器
// Story 8.6 QA修正: 每行一个粒子发射器
// 参数:
//   - entityID: 草皮卷实体ID
//   - lane: 行号（用于日志）
// 返回: 粒子发射器实体ID，失败返回0
func (s *SoddingSystem) createSodRollParticleEmitterForEntity(entityID ecs.EntityID, lane int) ecs.EntityID {
	// 计算粒子发射器位置
	// 注意：粒子发射器需要使用草皮卷的视觉中心Y坐标，而不是动画实体的锚点Y坐标
	particleX := s.animStartX
	particleY := 0.0

	// 从草皮卷实体读取当前视觉中心位置
	if posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID); ok {
		particleX = posComp.X
		particleY = posComp.Y

		// 读取 reanim 的包围盒中心偏移
		if reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
			if reanimComp.Reanim != nil {
				// 计算 reanim 的 Y 包围盒中心
				var minY, maxY *float64
				for _, track := range reanimComp.Reanim.Tracks {
					for _, frame := range track.Frames {
						if frame.Y != nil {
							y := *frame.Y
							if minY == nil || y < *minY {
								minY = &y
							}
							if maxY == nil || y > *maxY {
								maxY = &y
							}
						}
					}
				}

				if minY != nil && maxY != nil {
					// 粒子发射器的Y坐标 = 实体锚点Y + 包围盒中心偏移
					animCenterOffsetY := (*minY + *maxY) / 2.0
					particleY = posComp.Y + animCenterOffsetY
				}
			}
		}
	}

	// 应用配置的偏移量
	particleX += config.SodRollParticleOffsetX
	particleY += config.SodRollParticleOffsetY

	log.Printf("[SoddingSystem] 第 %d 行粒子发射器初始位置: X=%.1f, Y=%.1f", lane, particleX, particleY)

	// 使用粒子工厂创建发射器
	emitterID, err := entities.CreateParticleEffect(
		s.entityManager,
		s.resourceManager,
		"SodRoll",  // 粒子配置名称
		particleX,  // 起始位置X
		particleY,  // 起始位置Y（草皮卷视觉中心）
	)

	if err != nil {
		log.Printf("[SoddingSystem] Failed to create SodRoll particle emitter for lane %d: %v", lane, err)
		return 0
	}

	s.particlesEnabled = true
	log.Printf("[SoddingSystem] SodRoll particle emitter created for lane %d at (%.1f, %.1f)", lane, particleX, particleY)
	return emitterID
}
