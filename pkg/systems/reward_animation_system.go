package systems

import (
	"log"
	"math"
	"math/rand"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

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
		screenWidth:       800, // TODO: 从配置获取
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

	// 计算 Phase 3 目标位置（屏幕坐标中央上方）
	//
	// 设计说明：
	//   - 用户期望：卡片在屏幕的水平中央上方
	//   - 屏幕宽度：800
	//   - 屏幕中央X：800 / 2 = 400
	//
	// 计算流程：
	//   1. 计算屏幕中央坐标
	//   2. 减去半个卡片宽度，得到卡片左上角X坐标
	//   3. Y坐标：草坪顶部上方约50像素（屏幕坐标）
	screenCenterX := ras.screenWidth / 2.0 // 400
	lawnTopY := config.GridWorldStartY     // 76 (世界坐标和屏幕坐标Y相同)

	// 注意：PositionComponent 是卡片左上角坐标，需要减去半个卡片宽度使卡片中心对齐
	// 卡片原始尺寸 100x140，在 expanding 结束时，CardScale = 1.0，所以卡片宽度 = 100
	cardWidthAtEnd := 100.0                       // RewardExpandScaleEnd = 1.0 时的卡片宽度
	targetX := screenCenterX - cardWidthAtEnd/2.0 // 卡片左上角X = 屏幕中心X - 半宽 = 400 - 50 = 350
	targetY := lawnTopY + 50.0                    // 草坪顶部下方50像素

	log.Printf("[RewardAnimationSystem] ===== 位置计算调试信息 =====")
	log.Printf("[RewardAnimationSystem] 屏幕宽度: %.1f, 屏幕中心X: %.1f", ras.screenWidth, screenCenterX)
	log.Printf("[RewardAnimationSystem] 起始屏幕坐标: (%.1f, %.1f)", startX, startY)
	log.Printf("[RewardAnimationSystem] 卡片宽度(scale=1.0): %.1f", cardWidthAtEnd)
	log.Printf("[RewardAnimationSystem] 目标X计算: %.1f - %.1f/2 = %.1f", screenCenterX, cardWidthAtEnd, targetX)
	log.Printf("[RewardAnimationSystem] 卡片包起始位置: (%.1f, %.1f), 目标位置: (%.1f, %.1f)", startX, startY, targetX, targetY)
	log.Printf("[RewardAnimationSystem] 随机行: %d", randomLane)

	// 添加 RewardAnimationComponent
	ecs.AddComponent(ras.entityManager, ras.rewardEntity, &components.RewardAnimationComponent{
		Phase:          "appearing",
		ElapsedTime:    0,
		StartX:         startX,
		StartY:         startY,
		TargetX:        targetX,
		TargetY:        targetY,
		Scale:          config.PlantCardScale, // 使用标准卡片缩放（0.50）
		RewardType:     rewardType,            // 新增：奖励类型
		PlantID:        rewardID,              // 兼容性：植物ID或工具ID都存这里
		ToolID:         rewardID,              // 新增：工具ID（如果是工具奖励）
		ParticleEffect: particleEffect,        // 新增：粒子效果名称
	})

	// 添加 PositionComponent
	ecs.AddComponent(ras.entityManager, ras.rewardEntity, &components.PositionComponent{
		X: startX,
		Y: startY,
	})

	// 根据奖励类型创建不同的实体
	if rewardType == "plant" {
		// Story 8.4: 使用 NewPlantCardEntity 创建卡片包
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
			config.PlantCardScale, // 使用标准卡片缩放（0.50）
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
			Scale:          config.PlantCardScale, // 使用标准卡片缩放（0.50）
			RewardType:     rewardType,            // 新增：奖励类型
			PlantID:        rewardID,              // 兼容性：植物ID或工具ID都存这里
			ToolID:         rewardID,              // 新增：工具ID
			ParticleEffect: particleEffect,        // 新增：粒子效果名称
		})

		log.Printf("[RewardAnimationSystem] 植物卡片包已创建（使用 Story 8.4 统一工厂方法）")
	} else if rewardType == "tool" {
		// 工具奖励：添加 SpriteComponent 显示工具图标
		var toolImage *ebiten.Image
		if rewardID == "shovel" {
			toolImage = ras.resourceManager.GetImageByID("IMAGE_SHOVEL")
		}

		if toolImage != nil {
			ecs.AddComponent(ras.entityManager, ras.rewardEntity, &components.SpriteComponent{
				Image: toolImage,
			})
			// 添加 UIComponent 标记，表示这是 UI 实体（不需要相机偏移）
			ecs.AddComponent(ras.entityManager, ras.rewardEntity, &components.UIComponent{})
			log.Printf("[RewardAnimationSystem] 工具奖励实体已创建（类型: %s, ID: %s），使用 IMAGE_SHOVEL", rewardType, rewardID)
		} else {
			log.Printf("[RewardAnimationSystem] 警告：工具图片加载失败（ID: %s），只显示粒子效果", rewardID)
		}
	}

	// TODO: Phase 2 粒子背景框效果（需要查找合适的粒子配置）
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
	//   - expanding 阶段：无粒子（SeedPacket已清理，Award未创建）
	//   - showing 阶段：Award 粒子静止在目标位置
	//
	// 流程：
	//   1. 计算卡片中心坐标（视觉中心）
	//   2. 从配置获取粒子效果的锚点偏移（根据当前阶段）
	//   3. 锚点 = 视觉中心 + 锚点偏移
	//   4. 粒子系统根据 XML 中的 EmitterOffset 自动计算各发射器位置
	if ras.glowEntity != 0 && rewardComp.Phase == "waiting" {
		// 只在 waiting 阶段更新 SeedPacket 粒子位置
		rewardPos, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.rewardEntity)
		if ok {
			glowPos, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.glowEntity)
			if ok {
				cardComp, _ := ecs.GetComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity)

				// 计算卡片中心坐标（期望的视觉中心）
				visualCenterX := rewardPos.X
				visualCenterY := rewardPos.Y
				if cardComp != nil && cardComp.BackgroundImage != nil {
					cardWidth := float64(cardComp.BackgroundImage.Bounds().Dx()) * cardComp.CardScale
					cardHeight := float64(cardComp.BackgroundImage.Bounds().Dy()) * cardComp.CardScale
					visualCenterX += cardWidth / 2.0
					visualCenterY += cardHeight / 2.0
				}

				// SeedPacket 粒子有特殊的锚点偏移
				offsetX, offsetY := config.GetParticleAnchorOffset("SeedPacket")
				anchorX := visualCenterX + offsetX
				anchorY := visualCenterY + offsetY

				glowPos.X = anchorX
				glowPos.Y = anchorY
			}
		}
	}
}

// updateAppearingPhase 处理卡片包弹出阶段（0.4秒）。
// - 缩放动画：0.5 → 0.8
// - 微小抛物线：从草坪弹起，轻微上升后落回原位，无弹跳
func (ras *RewardAnimationSystem) updateAppearingPhase(dt float64, rewardComp *components.RewardAnimationComponent) {
	// 获取位置组件
	posComp, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.rewardEntity)
	if !ok {
		return
	}

	// 计算进度（0.0 - 1.0）
	progress := rewardComp.ElapsedTime / RewardAppearDuration
	if progress > 1.0 {
		progress = 1.0
	}

	// Phase 1 不需要缩放动画，保持标准卡片大小（config.PlantCardScale = 0.50）
	// Scale 已在 TriggerReward 中设置，这里无需修改

	// 计算Y轴位置（微小抛物线）
	// 使用 sin 曲线：从起点向上跳跃，到达最高点，然后平滑落回起点
	// sin(0) = 0（起点），sin(PI/2) = 1（最高点），sin(PI) = 0（落回起点）
	// 向上为负方向（Y轴向下为正）
	yOffset := -RewardAppearJumpHeight * math.Sin(progress*math.Pi)

	posComp.Y = rewardComp.StartY + yOffset

	// 检查完成
	if rewardComp.ElapsedTime >= RewardAppearDuration {
		log.Printf("[RewardAnimationSystem] Phase 1 (appearing) 完成，切换到 waiting")
		rewardComp.Phase = "waiting"
		rewardComp.ElapsedTime = 0

		// 根据奖励类型创建不同的 waiting 阶段粒子效果
		// - 植物奖励：SeedPacket（光晕 + 向下箭头）
		// - 工具奖励：AwardPickupArrow（向下箭头）
		if ras.glowEntity == 0 {
			var particleEffectName string
			if rewardComp.RewardType == "tool" {
				particleEffectName = "AwardPickupArrow" // 工具使用向下箭头
			} else {
				particleEffectName = "SeedPacket" // 植物使用光晕+箭头
			}

			// 计算粒子位置（卡片中心）
			cardComp, _ := ecs.GetComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity)
			particleX := posComp.X
			particleY := posComp.Y
			if cardComp != nil && cardComp.BackgroundImage != nil {
				cardWidth := float64(cardComp.BackgroundImage.Bounds().Dx()) * cardComp.CardScale
				cardHeight := float64(cardComp.BackgroundImage.Bounds().Dy()) * cardComp.CardScale
				particleX += cardWidth / 2.0  // X：卡片水平中心
				particleY += cardHeight / 2.0 // Y：卡片垂直中心
				if particleEffectName == "SeedPacket" {
					particleY -= 62.0 // SeedPacket 需要减去主粒子偏移量
				}
			}

			glowID, err := entities.CreateParticleEffect(
				ras.entityManager,
				ras.resourceManager,
				particleEffectName,
				particleX,
				particleY,
				0.0,  // angleOffset = 0
				true, // isUIParticle = true
			)
			if err != nil {
				log.Printf("[RewardAnimationSystem] 创建 waiting 粒子失败（%s）: %v", particleEffectName, err)
			} else {
				ras.glowEntity = glowID
				log.Printf("[RewardAnimationSystem] 创建 waiting 粒子成功: %s, ID=%d", particleEffectName, glowID)
			}
		}
	}
}

// updateWaitingPhase 处理等待阶段。
// - 卡片包静止，显示光晕和箭头粒子效果
// - 等待玩家点击后进入展开阶段
func (ras *RewardAnimationSystem) updateWaitingPhase(dt float64, rewardComp *components.RewardAnimationComponent) {
	// 检测玩家点击（鼠标或空格键）
	clicked := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace)

	if clicked {
		log.Printf("[RewardAnimationSystem] 玩家点击卡片包，切换到 expanding")
		rewardComp.Phase = "expanding"
		rewardComp.ElapsedTime = 0

		// 清理 Phase 2 的光晕粒子（SeedPacket）
		ras.cleanupGlowParticles()

		// 注意：Award 粒子特效在 expanding 阶段结束时才触发
		// 这样卡片先移动到目标位置，然后才显示光芒效果
	}
}

// updateExpandingPhase 处理卡片展开阶段（2秒）。
// - Award.xml 粒子特效播放（12个光芒发射器跟随卡片移动）
// - 卡片放大：config.PlantCardScale (0.50) → 1.0
// - 移动到屏幕中央上方
func (ras *RewardAnimationSystem) updateExpandingPhase(dt float64, rewardComp *components.RewardAnimationComponent) {
	// 获取位置组件
	posComp, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.rewardEntity)
	if !ok {
		return
	}

	// 计算进度（0.0 - 1.0）
	progress := rewardComp.ElapsedTime / RewardExpandDuration
	if progress > 1.0 {
		progress = 1.0
	}

	// 应用缓动函数（easeOutQuad）
	easedProgress := easeOutQuad(progress)

	// 更新缩放（从标准卡片大小 0.50 放大到原始大小 1.0）
	rewardComp.Scale = config.PlantCardScale + (RewardExpandScaleEnd-config.PlantCardScale)*easedProgress

	// 同步缩放到 PlantCardComponent（Story 8.4）
	if cardComp, ok := ecs.GetComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity); ok {
		cardComp.CardScale = rewardComp.Scale
	}

	// 更新位置（从当前位置移动到目标位置）
	// 注意：起始Y坐标已经包含了抛物线结束时的偏移
	posComp.X = rewardComp.StartX + (rewardComp.TargetX-rewardComp.StartX)*easedProgress
	posComp.Y = rewardComp.StartY + (rewardComp.TargetY-rewardComp.StartY)*easedProgress

	// 调试：每0.5秒输出一次当前位置
	if int(rewardComp.ElapsedTime*2)%1 == 0 && rewardComp.ElapsedTime > 0.01 {
		log.Printf("[Expanding] 进度: %.2f%%, 当前位置: (%.1f, %.1f), 当前缩放: %.2f",
			progress*100, posComp.X, posComp.Y, rewardComp.Scale)
	}

	// 检查完成
	if rewardComp.ElapsedTime >= RewardExpandDuration {
		log.Printf("[RewardAnimationSystem] Phase 3 (expanding) 移动完成")
		log.Printf("[RewardAnimationSystem] 最终位置: (%.1f, %.1f), 最终缩放: %.2f", posComp.X, posComp.Y, rewardComp.Scale)
		log.Printf("[RewardAnimationSystem] 目标位置: (%.1f, %.1f)", rewardComp.TargetX, rewardComp.TargetY)
		log.Printf("[RewardAnimationSystem] 触发 Award 粒子特效")

		// 在移动完成后触发 Award.xml 粒子特效（12个光芒发射器）
		posComp, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.rewardEntity)
		if ok {
			cardComp, _ := ecs.GetComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity)

			// 计算卡片中心坐标（粒子效果的锚点）
			particleX := posComp.X
			particleY := posComp.Y
			if cardComp != nil && cardComp.BackgroundImage != nil {
				cardWidth := float64(cardComp.BackgroundImage.Bounds().Dx()) * cardComp.CardScale
				cardHeight := float64(cardComp.BackgroundImage.Bounds().Dy()) * cardComp.CardScale
				particleX += cardWidth / 2.0  // X：卡片水平中心
				particleY += cardHeight / 2.0 // Y：卡片垂直中心
			}

			// 确定使用的粒子效果名称
		// expanding 结束时统一使用 Award（12个光芒），无论植物还是工具
		particleEffectName := "Award"

		// 创建粒子特效（Award）
			awardID, err := entities.CreateParticleEffect(
				ras.entityManager,
				ras.resourceManager,
				particleEffectName,
				particleX,
				particleY,
				0.0,  // angleOffset = 0
				true, // isUIParticle = true
			)
			if err != nil {
				log.Printf("[RewardAnimationSystem] 创建粒子特效失败（%s）: %v", particleEffectName, err)
			} else {
				// 保存粒子ID
				ras.glowEntity = awardID
				log.Printf("[RewardAnimationSystem] 创建粒子特效成功: %s, ID=%d", particleEffectName, awardID)
			}
		}

		// Phase 3 完成，进入短暂停顿阶段（让玩家看清楚卡片包到达目标位置）
		rewardComp.Phase = "pausing"
		rewardComp.ElapsedTime = 0
		log.Printf("[RewardAnimationSystem] Phase 3 完成，进入 pausing 阶段（短暂停顿）")
	}
}

// updatePausingPhase 处理短暂停顿阶段。
// - 卡片包静止在目标位置，展示 Award 粒子特效
// - 持续时间由 config.RewardPausingDuration 配置
// - 然后自动进入消失阶段
func (ras *RewardAnimationSystem) updatePausingPhase(dt float64, rewardComp *components.RewardAnimationComponent) {
	// 检查完成
	if rewardComp.ElapsedTime >= config.RewardPausingDuration {
		log.Printf("[RewardAnimationSystem] 停顿完成，进入 disappearing 阶段")
		rewardComp.Phase = "disappearing"
		rewardComp.ElapsedTime = 0
	}
}

// updateDisappearingPhase 处理卡片包消失阶段。
// - 卡片包淡出（透明度 1.0 → 0.0）
// - 持续时间由 config.RewardDisappearDuration 配置
// - 消失完成后，立即进入 Phase 4 显示面板
func (ras *RewardAnimationSystem) updateDisappearingPhase(dt float64, rewardComp *components.RewardAnimationComponent) {
	// 计算透明度（从 1.0 淡出到 0.0）
	progress := rewardComp.ElapsedTime / config.RewardDisappearDuration
	if progress > 1.0 {
		progress = 1.0
	}
	alpha := 1.0 - progress

	// 更新卡片透明度（通过 PlantCardComponent.Alpha）
	if cardComp, ok := ecs.GetComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity); ok {
		cardComp.Alpha = alpha
	}

	// 检查完成
	if rewardComp.ElapsedTime >= config.RewardDisappearDuration {
		log.Printf("[RewardAnimationSystem] 卡片包消失完成，进入面板显示阶段")

		// 保存奖励信息（在删除组件前）
		rewardType := rewardComp.RewardType
		plantID := rewardComp.PlantID
		toolID := rewardComp.ToolID

		// 重要：先更新组件状态，避免下一帧重复进入此阶段
		rewardComp.Phase = "showing"
		rewardComp.ElapsedTime = 0

		// 隐藏卡片包（移除 PlantCardComponent，使其不被渲染）
		if ras.rewardEntity != 0 {
			if _, hasCard := ecs.GetComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity); hasCard {
				ecs.RemoveComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity)
			}
		}

		// 清理 Award 粒子特效（避免与面板视觉冲突）
		if ras.glowEntity != 0 {
			ras.cleanupGlowParticles()
		}

		// 立即移除所有标记的实体
		ras.entityManager.RemoveMarkedEntities()

		// 进入 Phase 4 (showing)
		// 注意：不再使用rewardComp，因为rewardEntity的PlantCardComponent已被移除
		// 直接设置系统内部状态
		ras.currentPhase = "showing"
		ras.phaseElapsed = 0
		ras.currentPlantID = plantID // 保存植物ID（向后兼容）

		// 创建奖励面板（传递完整奖励信息）
		ras.createRewardPanel(rewardType, plantID, toolID)
		log.Printf("[RewardAnimationSystem] 进入 Phase 4 (showing)，显示奖励面板")
	}
}

// updateShowingPhaseInternal 处理显示奖励面板阶段（内部版本，不依赖rewardComp）。
// - 面板淡入动画（透明度 0.0 → 1.0，持续 config.RewardPanelFadeInDuration）
// - 淡入完成后，等待玩家点击"下一关"按钮
func (ras *RewardAnimationSystem) updateShowingPhaseInternal(dt float64) {
	ras.phaseElapsed += dt

	// 更新面板透明度（淡入动画）
	if ras.panelEntity != 0 {
		panelComp, ok := ecs.GetComponent[*components.RewardPanelComponent](ras.entityManager, ras.panelEntity)
		if ok {
			// 更新动画时间
			panelComp.AnimationTime += dt

			// 计算淡入进度（0.0 → 1.0）
			progress := panelComp.AnimationTime / config.RewardPanelFadeInDuration
			if progress > 1.0 {
				progress = 1.0
			}

			// 应用淡入效果
			panelComp.FadeAlpha = progress
		}
	}

	// 检测鼠标点击（只响应左键）
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// 获取鼠标点击位置
		mouseX, mouseY := ebiten.CursorPosition()

		// 检查是否点击了"下一关"按钮
		if ras.isNextLevelButtonClicked(float64(mouseX), float64(mouseY)) {
			log.Printf("[RewardAnimationSystem] 玩家点击\"下一关\"按钮，准备切换场景")
			ras.currentPhase = "closing"
			ras.phaseElapsed = 0
		}
		// 注意：空格键不再触发关闭，只有点击按钮才能继续
	}
}

// isNextLevelButtonClicked 检查鼠标点击是否在"下一关"按钮范围内
func (ras *RewardAnimationSystem) isNextLevelButtonClicked(mouseX, mouseY float64) bool {
	// 计算按钮位置（与 reward_panel_render_system.go 中的 drawNextLevelButton 逻辑一致）
	bgWidth := config.RewardPanelBackgroundWidth
	bgHeight := config.RewardPanelBackgroundHeight
	offsetX := (ras.screenWidth - bgWidth) / 2
	offsetY := (ras.screenHeight - bgHeight) / 2

	// 按钮中心位置
	buttonX := offsetX + bgWidth/2
	buttonY := offsetY + bgHeight*config.RewardPanelButtonY

	// 按钮尺寸（根据 IMAGE_SEEDCHOOSER_BUTTON 的尺寸）
	// SeedChooser_Button.png 尺寸约为 155x38（需要验证）
	// 为了更容易点击，使用稍大的点击区域
	buttonWidth := 155.0
	buttonHeight := 50.0 // 增大高度以提高可点击性

	// AABB 碰撞检测
	halfWidth := buttonWidth / 2
	halfHeight := buttonHeight / 2

	if mouseX >= buttonX-halfWidth && mouseX <= buttonX+halfWidth &&
		mouseY >= buttonY-halfHeight && mouseY <= buttonY+halfHeight {
		log.Printf("[RewardAnimationSystem] 按钮点击命中! 鼠标=(%.1f, %.1f), 按钮中心=(%.1f, %.1f)", mouseX, mouseY, buttonX, buttonY)
		return true
	}

	return false
}

// updateClosingPhaseInternal 处理关闭奖励面板阶段（0.5秒，内部版本）。
// - 淡出动画
// - 清理实体
// - 切换到下一关（或返回主菜单）
func (ras *RewardAnimationSystem) updateClosingPhaseInternal(dt float64) {
	ras.phaseElapsed += dt

	// 检查完成
	if ras.phaseElapsed >= RewardFadeOutDuration {
		log.Printf("[RewardAnimationSystem] Phase 5 (closing) 完成，清理实体")

		// 清理奖励实体
		if ras.rewardEntity != 0 {
			ras.entityManager.DestroyEntity(ras.rewardEntity)
		}
		if ras.panelEntity != 0 {
			ras.entityManager.DestroyEntity(ras.panelEntity)
		}
		if ras.glowEntity != 0 {
			ras.entityManager.DestroyEntity(ras.glowEntity)
		}

		ras.rewardEntity = 0
		ras.panelEntity = 0
		ras.glowEntity = 0
		ras.isActive = false
		ras.currentPhase = ""

		// 切换到下一关
		nextLevelID := ras.gameState.GetNextLevelID()
		if nextLevelID != "" {
			log.Printf("[RewardAnimationSystem] 切换到下一关: %s", nextLevelID)
			// 检查 sceneManager 是否存在（测试程序可能传入 nil）
			if ras.sceneManager != nil {
				ras.sceneManager.LoadLevel(nextLevelID)
			} else {
				log.Printf("[RewardAnimationSystem] SceneManager 为 nil，跳过场景切换（可能在测试环境）")
			}
		} else {
			log.Printf("[RewardAnimationSystem] 没有下一关，返回主菜单")
			// TODO: 切换到主菜单场景
			// ras.sceneManager.SwitchToMainMenu()
		}

		log.Printf("[RewardAnimationSystem] 奖励动画完成")
	}
}

// createRewardPanel 创建奖励面板实体。
func (ras *RewardAnimationSystem) createRewardPanel(rewardType string, plantID string, toolID string) {
	ras.panelEntity = ras.entityManager.CreateEntity()

	var rewardName, rewardDesc string
	var sunCost int

	// 根据奖励类型加载不同信息
	if rewardType == "tool" {
		// 工具奖励信息 - 从 LawnStrings 加载
		if toolID == "shovel" {
			// 从 LawnStrings 加载铲子文本
			rewardName = "铁铲" // 默认值
			rewardDesc = "让你挖出一株植物，腾出空间给其他植物" // 默认值

			if ras.gameState.LawnStrings != nil {
				// 加载铲子名称
				if name := ras.gameState.LawnStrings.GetString("SHOVEL"); name != "" {
					rewardName = name
				}
				// 加载铲子描述
				if desc := ras.gameState.LawnStrings.GetString("SHOVEL_DESCRIPTION"); desc != "" {
					rewardDesc = desc
				}
			}
		} else {
			// 未知工具，使用默认文本
			rewardName = "新工具"
			rewardDesc = "一个实用的工具！"
		}
		sunCost = 0 // 工具无阳光消耗
	} else {
		// 植物奖励信息
		plantInfo := ras.gameState.GetPlantUnlockManager().GetPlantInfo(plantID)

		// 从 LawnStrings 加载实际文本
		rewardName = ras.gameState.LawnStrings.GetString(plantInfo.NameKey)
		rewardDesc = ras.gameState.LawnStrings.GetString(plantInfo.DescriptionKey)

		// 硬编码阳光值（TODO: 从配置文件读取）
		sunCostMap := map[string]int{
			"sunflower":  50,
			"peashooter": 100,
			"cherrybomb": 150,
			"wallnut":    50,
		}
		sunCost = sunCostMap[plantID]
	}

	// 添加 RewardPanelComponent
	ecs.AddComponent(ras.entityManager, ras.panelEntity, &components.RewardPanelComponent{
		RewardType:       rewardType, // 新增：奖励类型
		PlantID:          plantID,    // 设置 PlantID，让渲染系统自动加载图标
		ToolID:           toolID,     // 新增：工具ID
		PlantName:        rewardName, // 名称（植物或工具）
		PlantDescription: rewardDesc, // 描述（植物或工具）
		SunCost:          sunCost,    // 设置阳光值（工具为0）
		CardScale:        1.0,        // 卡片固定大小，不做动画
		FadeAlpha:        0.0,        // Story 8.4: 初始完全透明，用于淡入动画
		// Story 8.4: 卡片位置由 RewardPanelRenderSystem 自动计算（水平居中）
		IsVisible:     true,
		AnimationTime: 0.0, // Story 8.4: 动画时间计数器，用于淡入效果
	})

	log.Printf("[RewardAnimationSystem] 奖励面板已创建：%s - %s", rewardName, rewardDesc)
}

// IsActive 返回系统是否正在播放动画。
func (ras *RewardAnimationSystem) IsActive() bool {
	return ras.isActive
}

// IsCompleted 返回系统是否已完成。
func (ras *RewardAnimationSystem) IsCompleted() bool {
	return !ras.isActive && ras.rewardEntity == 0
}

// GetEntity 返回当前奖励动画实体ID（用于调试和验证）。
func (ras *RewardAnimationSystem) GetEntity() ecs.EntityID {
	return ras.rewardEntity
}

// ========== 辅助方法 ==========

// easeOutQuad 二次缓动函数（快速开始，缓慢结束）。
func easeOutQuad(t float64) float64 {
	return 1.0 - (1.0-t)*(1.0-t)
}

// createCardPackReanim 创建卡片包的最小化 Reanim 定义。
// 返回一个单帧、单 track 的静态图片 Reanim。
func (ras *RewardAnimationSystem) createCardPackReanim() *reanim.ReanimXML {
	frameNum := 0
	x := 0.0
	y := 0.0
	scaleX := 1.0
	scaleY := 1.0
	skewX := 0.0
	skewY := 0.0

	return &reanim.ReanimXML{
		FPS: 12,
		Tracks: []reanim.Track{
			{
				Name: "idle",
				Frames: []reanim.Frame{
					{
						FrameNum:  &frameNum,
						X:         &x,
						Y:         &y,
						ScaleX:    &scaleX,
						ScaleY:    &scaleY,
						SkewX:     &skewX,
						SkewY:     &skewY,
						ImagePath: "card_pack",
					},
				},
			},
		},
	}
}

// plantIDToType 将 plantID 字符串转换为 PlantType 枚举
func (ras *RewardAnimationSystem) plantIDToType(plantID string) components.PlantType {
	switch plantID {
	case "sunflower":
		return components.PlantSunflower
	case "peashooter":
		return components.PlantPeashooter
	case "cherrybomb":
		return components.PlantCherryBomb
	case "wallnut":
		return components.PlantWallnut
	default:
		return components.PlantUnknown
	}
}

// getReanimName 根据 plantID 获取 Reanim 资源名称
func (ras *RewardAnimationSystem) getReanimName(plantID string) string {
	switch plantID {
	case "sunflower":
		return "SunFlower"
	case "peashooter":
		return "PeaShooterSingle" // Story 10.3: 修正为普通豌豆射手资源
	case "cherrybomb":
		return "CherryBomb"
	case "wallnut":
		return "Wallnut"
	default:
		return ""
	}
}

// compositePlantCard 将植物图标合成到卡片包图片上。
func (ras *RewardAnimationSystem) compositePlantCard(cardPackImg, plantIcon *ebiten.Image) *ebiten.Image {
	if plantIcon == nil {
		return cardPackImg
	}

	// 创建新画布
	bounds := cardPackImg.Bounds()
	compositeImg := ebiten.NewImage(bounds.Dx(), bounds.Dy())

	// 绘制卡片包背景
	compositeImg.DrawImage(cardPackImg, nil)

	// 绘制植物图标（居中对齐）
	op := &ebiten.DrawImageOptions{}
	iconBounds := plantIcon.Bounds()
	offsetX := float64(bounds.Dx()-iconBounds.Dx()) / 2.0
	offsetY := float64(bounds.Dy()-iconBounds.Dy()) / 2.0
	op.GeoM.Translate(offsetX, offsetY)
	compositeImg.DrawImage(plantIcon, op)

	return compositeImg
}

// cleanupGlowParticles 清理光晕粒子和发射器
// 用于清理 SeedPacket（Phase 2）或 Award（Phase 3）粒子特效
func (ras *RewardAnimationSystem) cleanupGlowParticles() {
	// 注意：需要立即清理发射器及所有已生成的粒子，让它们立即消失
	if ras.glowEntity != 0 {
		// 获取发射器的所有活跃粒子并立即销毁
		emitter, ok := ecs.GetComponent[*components.EmitterComponent](ras.entityManager, ras.glowEntity)
		if ok && len(emitter.ActiveParticles) > 0 {
			// 立即销毁所有已生成的粒子实体
			for _, particleID := range emitter.ActiveParticles {
				// 方法1：直接销毁粒子实体
				ras.entityManager.DestroyEntity(particleID)

				// 方法2：如果方法1不够快，设置粒子生命周期让它立即过期
				if particle, ok := ecs.GetComponent[*components.ParticleComponent](ras.entityManager, particleID); ok {
					particle.Age = particle.Lifetime + 1.0 // 让粒子立即过期
				}
			}
			log.Printf("[RewardAnimationSystem] 清理 %d 个光晕粒子", len(emitter.ActiveParticles))
		}

		// 清理发射器实体（SeedPacket有2个发射器，Award有13个发射器，需要查找并清理所有相关发射器）
		// Story 7.4: CreateParticleEffect 创建的所有发射器共享同一个 PositionComponent
		// 我们需要找到所有共享同一位置的发射器实体
		glowPos, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.glowEntity)
		if ok {
			// 查询所有发射器实体
			allEmitters := ecs.GetEntitiesWith2[
				*components.EmitterComponent,
				*components.PositionComponent,
			](ras.entityManager)

			// 清理所有与光晕粒子共享位置的发射器
			cleanedCount := 0
			for _, emitterID := range allEmitters {
				emitterPos, _ := ecs.GetComponent[*components.PositionComponent](ras.entityManager, emitterID)
				// 使用指针比较，共享同一个 PositionComponent 实例的发射器属于同一组
				if emitterPos == glowPos {
					// 清理该发射器的所有粒子
					if emitterComp, ok := ecs.GetComponent[*components.EmitterComponent](ras.entityManager, emitterID); ok {
						for _, particleID := range emitterComp.ActiveParticles {
							ras.entityManager.DestroyEntity(particleID)
							// 让粒子立即过期
							if particle, ok := ecs.GetComponent[*components.ParticleComponent](ras.entityManager, particleID); ok {
								particle.Age = particle.Lifetime + 1.0
							}
						}
					}
					ras.entityManager.DestroyEntity(emitterID)
					cleanedCount++
				}
			}
			log.Printf("[RewardAnimationSystem] 清理了 %d 个发射器实体", cleanedCount)
		}

		// 立即移除标记的实体，让清理生效
		ras.entityManager.RemoveMarkedEntities()

		ras.glowEntity = 0
		log.Printf("[RewardAnimationSystem] 清理光晕粒子完成")
	}
}

// isParticleEffectCompleted 检查粒子特效是否播放完成
// 委托给 ParticleSystem 来判断（遵循模块职责分离原则）
func (ras *RewardAnimationSystem) isParticleEffectCompleted() bool {
	if ras.particleSystem == nil || ras.glowEntity == 0 {
		return false
	}

	// 使用 ParticleSystem 提供的公开接口
	return ras.particleSystem.IsParticleEffectCompleted(ras.glowEntity)
}

// Draw 渲染奖励动画的所有元素
// Story 8.4重构：完全封装渲染逻辑，调用者只需调用此方法
//
// 内部自动处理：
// - Reanim 实体（卡片包的 Reanim 动画）
// - 粒子效果（SeedPacket 背景框 + Award 爆炸特效）
// - 植物卡片（Phase 1-3 的卡片包）
// - 奖励面板（Phase 4）
//
// 渲染顺序（从下到上）：
// 1. Reanim 实体
// 2. 粒子效果（装饰层）
// 3. 植物卡片 / 奖励面板（最上层，不被粒子遮挡）
//
// 注意：奖励动画的粒子标记为 UIComponent（isUIParticle=true）
// GameScene Layer 6 会过滤掉 UI 粒子，只渲染游戏世界粒子
func (ras *RewardAnimationSystem) Draw(screen *ebiten.Image) {
	if !ras.isActive {
		return
	}

	cameraOffsetX := ras.gameState.CameraX

	// 1. 绘制 Reanim 实体（如果有的话）
	ras.renderSystem.Draw(screen, cameraOffsetX)

	// 2. 绘制粒子效果（SeedPacket 背景框 + Award 爆炸）
	//    只渲染 UI 粒子（奖励动画的粒子），不渲染游戏世界粒子
	ras.renderSystem.DrawParticles(screen, cameraOffsetX)

	// 3a. Phase 1-3: 渲染奖励卡片/工具图标（appearing/waiting/expanding/pausing/disappearing）在最上层
	// 直接渲染自己管理的卡片实体（符合 ECS 原则：系统负责自己的实体）
	if ras.currentPhase != "showing" && ras.currentPhase != "closing" && ras.rewardEntity != 0 {
		// 植物奖励：渲染植物卡片
		if card, ok := ecs.GetComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity); ok {
			if pos, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.rewardEntity); ok {
				// 使用统一的渲染函数
				entities.RenderPlantCard(screen, card, pos.X, pos.Y, ras.sunFont, ras.sunFontSize)
			}
		}

		// 工具奖励：渲染工具图标
		if sprite, ok := ecs.GetComponent[*components.SpriteComponent](ras.entityManager, ras.rewardEntity); ok {
			if pos, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.rewardEntity); ok {
				if rewardComp, ok := ecs.GetComponent[*components.RewardAnimationComponent](ras.entityManager, ras.rewardEntity); ok {
					if sprite.Image != nil {
						op := &ebiten.DrawImageOptions{}

						// 居中图片
						bounds := sprite.Image.Bounds()
						op.GeoM.Translate(-float64(bounds.Dx())/2, -float64(bounds.Dy())/2)

						// 应用缩放（与植物卡片相同的缩放逻辑）
						op.GeoM.Scale(rewardComp.Scale, rewardComp.Scale)

						// 移动到位置（屏幕坐标）
						op.GeoM.Translate(pos.X, pos.Y)

						screen.DrawImage(sprite.Image, op)
					}
				}
			}
		}
	}

	// 3b. Phase 4: 渲染奖励面板（showing）在最上层
	if ras.currentPhase == "showing" || ras.panelEntity != 0 {
		ras.panelRenderSystem.Draw(screen)
	}
}
