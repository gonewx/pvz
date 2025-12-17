package systems

import (
	"log"
	"math/rand"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ============================================================================
// 奖励动画系统核心（RewardAnimationSystem）
// ============================================================================

const (
	// Phase 1 - Appearing (出现阶段)
	RewardAppearDuration   = 0.6  // 出现动画持续时间（秒）
	RewardAppearJumpHeight = 40.0 // 微小跳跃高度（像素）

	// Phase 3 - Expanding (展开阶段)
	RewardExpandDuration     = 2.0  // 展开动画持续时间（秒）
	RewardExpandScaleEnd     = 1.0  // 最终缩放（放大到原始大小）
	RewardExpandTargetYRatio = 0.45 // 目标Y位置（screenHeight * ratio）- 草坪上方中央

	// Phase 3.5 - Flashing (闪光展示阶段)
	RewardFlashingDuration = 2.5 // Award 粒子光芒展示时长（秒）- 让玩家充分欣赏视觉效果

	// Phase 4 - Showing (显示奖励面板)
	RewardCardScaleStart    = 0.5 // 卡片初始缩放
	RewardCardScaleEnd      = 1.5 // 卡片最终缩放
	RewardCardScaleDuration = 1.0 // 卡片缩放动画持续时间（秒）

	// Phase 5 - Closing (关闭阶段)
	RewardFadeOutDuration = 0.5 // 淡出持续时间（秒）

	// 卡片包位置配置（可手工调整）
	RewardCardPackOffsetFromRight = 80.0 // 距离草坪右边界的距离
)

// RewardAnimationSystem 管理关卡完成后的奖励动画流程。
// 负责卡片包弹出、等待点击、展开、显示奖励面板等阶段的状态管理和动画更新。
//
// 设计原则：完全封装奖励流程，调用者只需：
// 1. 创建系统：NewRewardAnimationSystem()
// 2. 触发奖励：TriggerReward(plantID)
// 3. 更新逻辑：Update(dt)
// 4. 渲染画面：Draw(screen)
//
// 内部自动管理：
// - 卡片包的创建和渲染
// - 粒子效果的创建和渲染
// - 奖励面板的创建和渲染
// - 所有阶段的转换和清理
//
// 动画流程（5个阶段）：
// 1. appearing (0.3秒): 卡片从草坪右侧随机行弹出，微小上升 + 缩放动画 (0.8 → 1.0)
// 2. waiting: 卡片静止，显示粒子背景框 + 橙色箭头指示器（闪烁），等待玩家点击
// 3. expanding (2秒): 点击后触发 Award.xml 粒子特效，卡片放大并移动到屏幕中央上方
// 4. showing: 粒子特效完成后显示新植物介绍面板，等待玩家点击"下一关"或关闭
// 5. closing (0.5秒): 淡出动画，清理实体，返回主菜单或进入下一关
type RewardAnimationSystem struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager
	sceneManager    *game.SceneManager // 场景管理器，用于切换到下一关
	reanimSystem    *ReanimSystem      // Reanim系统用于创建和管理动画
	particleSystem  *ParticleSystem    // 粒子系统用于检查粒子特效完成状态
	renderSystem    *RenderSystem      // 渲染系统用于绘制Reanim和粒子效果
	rewardEntity    ecs.EntityID       // 奖励动画实体ID（卡片包）
	panelEntity     ecs.EntityID       // 奖励面板实体ID
	glowEntity      ecs.EntityID       // 光晕粒子发射器实体ID
	isActive        bool               // 系统是否激活
	currentPhase    string             // 当前阶段（flashing/showing/closing 时可能没有 rewardEntity）
	currentPlantID  string             // 当前植物ID（保存用于面板创建）
	phaseElapsed    float64            // 当前阶段经过时间
	screenWidth     float64
	screenHeight    float64

	// Story 8.4重构：内部封装所有渲染系统，调用者无需关心
	panelRenderSystem *RewardPanelRenderSystem // 奖励面板渲染系统（内部使用）
	sunFont           *text.GoTextFaceSource   // 阳光数字字体源（用于渲染卡片）
	sunFontSize       float64                  // 阳光字体大小
}

// NewRewardAnimationSystem 创建新的奖励动画系统。
func NewRewardAnimationSystem(em *ecs.EntityManager, gs *game.GameState, rm *game.ResourceManager, sm *game.SceneManager, reanimSys *ReanimSystem, particleSys *ParticleSystem, renderSys *RenderSystem) *RewardAnimationSystem {
	// Story 8.4重构：内部创建所有渲染系统，调用者无需关心
	panelRenderSystem := NewRewardPanelRenderSystem(em, gs, rm, reanimSys)

	// 加载阳光字体（用于渲染卡片的阳光数字）
	sunFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.PlantCardSunCostFontSize)
	var fontSource *text.GoTextFaceSource
	var fontSize float64 = config.PlantCardSunCostFontSize
	if err != nil {
		log.Printf("[RewardAnimationSystem] Warning: Failed to load sun cost font: %v", err)
		fontSource = nil
	} else {
		fontSource = sunFont.Source
	}

	return &RewardAnimationSystem{
		entityManager:     em,
		gameState:         gs,
		resourceManager:   rm,
		sceneManager:      sm, // 保存场景管理器引用
		reanimSystem:      reanimSys,
		particleSystem:    particleSys,
		renderSystem:      renderSys, // 渲染系统用于绘制Reanim和粒子
		rewardEntity:      0,
		panelEntity:       0,
		glowEntity:        0,
		isActive:          false,
		screenWidth:       800,
		screenHeight:      600,
		panelRenderSystem: panelRenderSystem, // 内部封装
		sunFont:           fontSource,
		sunFontSize:       fontSize,
	}
}

// TriggerReward 触发奖励动画，开始卡片包弹出流程。
// rewardType: 奖励类型（"plant" 或 "tool"）
// rewardID: 奖励ID（植物ID如 "sunflower" 或工具ID如 "shovel"）
func (ras *RewardAnimationSystem) TriggerReward(rewardType string, rewardID string) {
	if ras.isActive {
		log.Printf("[RewardAnimationSystem] 奖励动画已在播放，忽略新触发")
		return
	}

	log.Printf("[RewardAnimationSystem] 触发奖励动画，类型: %s, ID: %s", rewardType, rewardID)

	// 根据类型选择粒子效果
	var particleEffect string
	if rewardType == "tool" {
		particleEffect = "AwardPickupArrow" // 工具使用向下箭头粒子效果
	} else {
		particleEffect = "Award" // 植物使用默认光芒粒子效果
	}

	// 创建奖励动画实体
	ras.rewardEntity = ras.entityManager.CreateEntity()
	ras.isActive = true

	// 随机选择草坪行（2-4行，偏中间位置）
	randomLane := 1 + rand.Intn(3) // 第2、3或4行

	// 计算起始位置（屏幕坐标，从屏幕右侧弹出）
	// 屏幕宽度：800，选择屏幕右半部分（500-700）
	startX := 500.0 + rand.Float64()*200.0 // 屏幕坐标 500-700
	startY := config.GridWorldStartY + float64(randomLane)*config.CellHeight + config.CellHeight/2.0

	// 根据奖励类型计算不同的目标位置和初始缩放
	var targetX, targetY float64
	var startScale, endScale float64

	if rewardType == "tool" {
		// 工具奖励：目标位置应与奖励面板中铲子的显示位置一致
		// 奖励面板使用卡片显示区域（RewardPanelCardBox）的中心
		bgWidth := config.RewardPanelBackgroundWidth
		bgHeight := config.RewardPanelBackgroundHeight
		offsetX := (ras.screenWidth - bgWidth) / 2
		offsetY := (ras.screenHeight - bgHeight) / 2

		// 卡片显示区域中心（与 drawToolIcon 中的计算一致）
		boxCenterX := offsetX + config.RewardPanelCardBoxLeft + config.RewardPanelCardBoxWidth/2
		boxCenterY := offsetY + config.RewardPanelCardBoxTop + config.RewardPanelCardBoxHeight/2

		// 工具图标使用中心锚点绘制，所以目标位置就是中心坐标
		targetX = boxCenterX
		targetY = boxCenterY

		// 工具奖励使用专用的缩放配置
		startScale = config.ShovelRewardStartScale // 约 0.56，与植物卡包视觉大小一致
		endScale = config.ShovelRewardEndScale     // 1.0，与奖励面板中的显示一致
	} else {
		// 植物奖励：目标位置是屏幕中央上方
		screenCenterX := ras.screenWidth / 2.0
		lawnTopY := config.GridWorldStartY

		// 卡片原始尺寸 100x140，目标缩放 1.0
		cardWidthAtEnd := 100.0
		targetX = screenCenterX - cardWidthAtEnd/2.0
		targetY = lawnTopY + 50.0

		// 植物奖励使用标准卡片缩放
		startScale = config.PlantCardScale // 0.50
		endScale = RewardExpandScaleEnd    // 1.0
	}

	log.Printf("[RewardAnimationSystem] ===== 位置计算调试信息 =====")
	log.Printf("[RewardAnimationSystem] 屏幕宽度: %.1f", ras.screenWidth)
	log.Printf("[RewardAnimationSystem] 起始屏幕坐标: (%.1f, %.1f)", startX, startY)
	log.Printf("[RewardAnimationSystem] 目标位置: (%.1f, %.1f)", targetX, targetY)
	log.Printf("[RewardAnimationSystem] 起始缩放: %.2f, 目标缩放: %.2f", startScale, endScale)
	log.Printf("[RewardAnimationSystem] 随机行: %d", randomLane)

	// 添加 RewardAnimationComponent
	ecs.AddComponent(ras.entityManager, ras.rewardEntity, &components.RewardAnimationComponent{
		Phase:          "appearing",
		ElapsedTime:    0,
		StartX:         startX,
		StartY:         startY,
		TargetX:        targetX,
		TargetY:        targetY,
		Scale:          startScale, // 使用计算的初始缩放
		RewardType:     rewardType,
		PlantID:        rewardID,
		ToolID:         rewardID,
		ParticleEffect: particleEffect,
	})

	// 添加 PositionComponent
	ecs.AddComponent(ras.entityManager, ras.rewardEntity, &components.PositionComponent{
		X: startX,
		Y: startY,
	})

	// 根据奖励类型创建不同的实体
	if rewardType == "plant" {
		// 使用 NewPlantCardEntity 创建卡片包
		// 这样可以自动获得：背景图、植物图标、阳光数字等完整渲染
		plantType := ras.plantIDToType(rewardID)

		if plantType == components.PlantUnknown {
			log.Printf("[RewardAnimationSystem] Unknown plant ID: %s, aborting", rewardID)
			ras.entityManager.DestroyEntity(ras.rewardEntity)
			ras.entityManager.RemoveMarkedEntities()
			ras.isActive = false
			return
		}

		// 创建植物卡片实体作为卡片包（使用统一工厂方法）
		// 注意：这里的位置会被 RewardAnimationComponent 覆盖更新
		cardEntity, err := entities.NewPlantCardEntity(
			ras.entityManager,
			ras.resourceManager,
			ras.reanimSystem,
			plantType,
			startX,
			startY,
			startScale, // 使用计算的初始缩放
		)
		if err != nil {
			log.Printf("[RewardAnimationSystem] Failed to create card pack entity: %v", err)
			ras.entityManager.DestroyEntity(ras.rewardEntity)
			ras.entityManager.RemoveMarkedEntities()
			ras.isActive = false
			return
		}

		// 添加 RewardCardComponent 标记，区分奖励卡片和选择栏卡片
		// 这样 RewardAnimationSystem 的 plantCardRenderSystem 只渲染奖励卡片
		// 而验证工具的 plantCardRenderSystem 只渲染选择栏卡片
		ecs.AddComponent(ras.entityManager, cardEntity, &components.RewardCardComponent{})

		// 重要：使用卡片实体作为奖励实体
		ras.entityManager.DestroyEntity(ras.rewardEntity)
		ras.rewardEntity = cardEntity

		// 重新添加 RewardAnimationComponent（控制动画状态）
		ecs.AddComponent(ras.entityManager, ras.rewardEntity, &components.RewardAnimationComponent{
			Phase:          "appearing",
			ElapsedTime:    0,
			StartX:         startX,
			StartY:         startY,
			TargetX:        targetX,
			TargetY:        targetY,
			Scale:          startScale,
			RewardType:     rewardType,
			PlantID:        rewardID,
			ToolID:         rewardID,
			ParticleEffect: particleEffect,
		})

		log.Printf("[RewardAnimationSystem] 植物卡片包已创建（使用 Story 8.4 统一工厂方法）")
	} else if rewardType == "tool" {
		// 工具奖励：添加 SpriteComponent 显示工具图标
		var toolImage *ebiten.Image
		if rewardID == "shovel" {
			toolImage = ras.resourceManager.GetImageByID("IMAGE_SHOVEL_HI_RES")
		}

		if toolImage != nil {
			ecs.AddComponent(ras.entityManager, ras.rewardEntity, &components.SpriteComponent{
				Image: toolImage,
			})
			// 添加 UIComponent 标记，表示这是 UI 实体（不需要相机偏移）
			ecs.AddComponent(ras.entityManager, ras.rewardEntity, &components.UIComponent{})
			log.Printf("[RewardAnimationSystem] 工具奖励实体已创建（类型: %s, ID: %s），使用 IMAGE_SHOVEL_HI_RES", rewardType, rewardID)
		} else {
			log.Printf("[RewardAnimationSystem] 警告：工具图片加载失败（ID: %s），只显示粒子效果", rewardID)
		}
	}

	// Phase 2: 粒子背景框效果（待实现）
}

// TriggerPlantReward 触发植物奖励动画（向后兼容方法）
func (ras *RewardAnimationSystem) TriggerPlantReward(plantID string) {
	ras.TriggerReward("plant", plantID)
}

// TriggerToolReward 触发工具奖励动画
func (ras *RewardAnimationSystem) TriggerToolReward(toolID string) {
	ras.TriggerReward("tool", toolID)
}

// Update 更新奖励动画系统，处理各阶段的动画逻辑。
func (ras *RewardAnimationSystem) Update(dt float64) {
	if !ras.isActive {
		return
	}

	// Phase 1-3: 使用 rewardEntity（卡片包）的 RewardAnimationComponent
	// Phase 4+: 使用系统内部的 currentPhase，因为卡片包已消失
	var rewardComp *components.RewardAnimationComponent
	var ok bool

	if ras.currentPhase == "showing" || ras.currentPhase == "closing" {
		// showing/closing 阶段：卡片包已消失，直接处理面板逻辑
		if ras.currentPhase == "showing" {
			ras.updateShowingPhaseInternal(dt)
		} else if ras.currentPhase == "closing" {
			ras.updateClosingPhaseInternal(dt)
		}
		return
	}

	// Phase 1-3: 查询奖励动画实体（卡片包）
	rewardComp, ok = ecs.GetComponent[*components.RewardAnimationComponent](ras.entityManager, ras.rewardEntity)
	if !ok {
		return
	}

	rewardComp.ElapsedTime += dt

	// 状态机处理
	switch rewardComp.Phase {
	case "appearing":
		ras.updateAppearingPhase(dt, rewardComp)
	case "waiting":
		ras.updateWaitingPhase(dt, rewardComp)
	case "expanding":
		ras.updateExpandingPhase(dt, rewardComp)
	case "pausing":
		ras.updatePausingPhase(dt, rewardComp)
	case "disappearing":
		ras.updateDisappearingPhase(dt, rewardComp)
	}

	// 同步当前阶段到系统（用于Draw方法判断）
	ras.currentPhase = rewardComp.Phase

	// 更新粒子发射器位置（跟随卡片包）
	//
	// 设计说明：
	//   - waiting 阶段：SeedPacket 粒子跟随卡片
	//   - expanding 阶段：Award 粒子跟随卡片移动
	//   - pausing 阶段：Award 粒子静止在目标位置
	//
	// 流程：
	//   1. 计算卡片中心坐标（视觉中心）
	//   2. 从配置获取粒子效果的锚点偏移（根据当前阶段）
	//   3. 锚点 = 视觉中心 + 锚点偏移
	//   4. 更新发射器位置（影响新生成的粒子）
	//   5. 更新所有已生成粒子位置（让它们跟随卡片移动）
	if ras.glowEntity != 0 {
		shouldUpdatePosition := false
		var anchorOffsetX, anchorOffsetY float64

		// waiting 阶段：更新 SeedPacket 粒子位置
		if rewardComp.Phase == "waiting" {
			shouldUpdatePosition = true
			anchorOffsetX, anchorOffsetY = config.GetParticleAnchorOffset("SeedPacket")
		}

		// expanding 阶段：更新 Award 粒子位置（跟随卡片移动）
		if rewardComp.Phase == "expanding" {
			shouldUpdatePosition = true
			anchorOffsetX, anchorOffsetY = config.GetParticleAnchorOffset("Award")
		}

		if shouldUpdatePosition {
			rewardPos, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.rewardEntity)
			if ok {
				glowPos, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.glowEntity)
				if ok {
					cardComp, _ := ecs.GetComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity)

					// 计算卡片视觉中心坐标（基于当前缩放）
					visualCenterX := rewardPos.X
					visualCenterY := rewardPos.Y
					if cardComp != nil && cardComp.BackgroundImage != nil {
						cardWidth := float64(cardComp.BackgroundImage.Bounds().Dx()) * cardComp.CardScale
						cardHeight := float64(cardComp.BackgroundImage.Bounds().Dy()) * cardComp.CardScale
						visualCenterX += cardWidth / 2.0
						visualCenterY += cardHeight / 2.0
					}

					// 应用锚点偏移
					anchorX := visualCenterX + anchorOffsetX
					anchorY := visualCenterY + anchorOffsetY

					// 计算位置偏移量（相对于发射器当前位置）
					deltaX := anchorX - glowPos.X
					deltaY := anchorY - glowPos.Y

					// 更新发射器位置（影响新生成的粒子）
					// 注意：Award.xml 有13个发射器共享同一个 PositionComponent
					glowPos.X = anchorX
					glowPos.Y = anchorY

					// 更新所有已生成粒子的位置（让它们跟随卡片移动）
					// 需要遍历所有共享同一个 PositionComponent 的发射器
					allEmitters := ecs.GetEntitiesWith2[
						*components.EmitterComponent,
						*components.PositionComponent,
					](ras.entityManager)

					for _, emitterID := range allEmitters {
						emitterPos, _ := ecs.GetComponent[*components.PositionComponent](ras.entityManager, emitterID)
						// 使用指针比较，共享同一个 PositionComponent 实例的发射器属于同一组
						if emitterPos == glowPos {
							emitter, ok := ecs.GetComponent[*components.EmitterComponent](ras.entityManager, emitterID)
							if ok && len(emitter.ActiveParticles) > 0 {
								for _, particleID := range emitter.ActiveParticles {
									particlePos, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, particleID)
									if ok {
										particlePos.X += deltaX
										particlePos.Y += deltaY
									}
								}
							}
						}
					}
				}
			}
		}
	}
}
