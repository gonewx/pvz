package systems

import (
	"log"
	"math"
	"math/rand"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	// Phase 1 - Appearing (出现阶段)
	RewardAppearDuration     = 0.3  // 出现动画持续时间（秒）
	RewardAppearScaleStart   = 0.5  // 初始缩放
	RewardAppearScaleEnd     = 0.8  // 最终缩放
	RewardAppearRiseDistance = 20.0 // 上升距离（像素）

	// Phase 2 - Waiting (等待点击阶段)
	RewardArrowBlinkFrequency = 3.0 // 箭头闪烁频率（Hz）

	// Phase 3 - Expanding (展开阶段)
	RewardExpandDuration     = 2.0 // 展开动画持续时间（秒）
	RewardExpandScaleStart   = 0.8 // 初始缩放
	RewardExpandScaleEnd     = 1.0 // 最终缩放
	RewardExpandTargetYRatio = 0.3 // 目标Y位置（screenHeight * ratio）

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
	reanimSystem    *ReanimSystem // Reanim系统用于创建和管理动画
	rewardEntity    ecs.EntityID  // 奖励动画实体ID（卡片包）
	panelEntity     ecs.EntityID  // 奖励面板实体ID
	isActive        bool          // 系统是否激活
	screenWidth     float64
	screenHeight    float64
}

// NewRewardAnimationSystem 创建新的奖励动画系统。
func NewRewardAnimationSystem(em *ecs.EntityManager, gs *game.GameState, rm *game.ResourceManager, reanimSys *ReanimSystem) *RewardAnimationSystem {
	return &RewardAnimationSystem{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
		reanimSystem:    reanimSys,
		rewardEntity:    0,
		panelEntity:     0,
		isActive:        false,
		screenWidth:     800, // TODO: 从配置获取
		screenHeight:    600,
	}
}

// TriggerReward 触发奖励动画，开始卡片包弹出流程。
// plantID: 解锁的植物ID（如 "sunflower"）
func (ras *RewardAnimationSystem) TriggerReward(plantID string) {
	if ras.isActive {
		log.Printf("[RewardAnimationSystem] 奖励动画已在播放，忽略新触发")
		return
	}

	log.Printf("[RewardAnimationSystem] 触发奖励动画，植物ID: %s", plantID)

	// 创建奖励动画实体
	ras.rewardEntity = ras.entityManager.CreateEntity()
	ras.isActive = true

	// 随机选择草坪行（1-5行）
	randomLane := rand.Intn(config.GridRows) + 1

	// 计算起始位置（草坪右侧边缘，随机行中央）
	startX := config.GridWorldEndX - RewardCardPackOffsetFromRight
	startY := config.GridWorldStartY + float64(randomLane-1)*config.CellHeight + config.CellHeight/2.0

	// 计算 Phase 3 目标位置（屏幕中央上方）
	targetX := ras.screenWidth / 2.0
	targetY := ras.screenHeight * RewardExpandTargetYRatio

	log.Printf("[RewardAnimationSystem] 卡片包起始位置: (%.1f, %.1f), 目标位置: (%.1f, %.1f), 随机行: %d",
		startX, startY, targetX, targetY, randomLane)

	// 添加 RewardAnimationComponent
	ecs.AddComponent(ras.entityManager, ras.rewardEntity, &components.RewardAnimationComponent{
		Phase:       "appearing",
		ElapsedTime: 0,
		StartX:      startX,
		StartY:      startY,
		TargetX:     targetX,
		TargetY:     targetY,
		Scale:       RewardAppearScaleStart,
		PlantID:     plantID,
		ShowArrow:   false,
		ArrowAlpha:  1.0,
	})

	// 添加 PositionComponent
	ecs.AddComponent(ras.entityManager, ras.rewardEntity, &components.PositionComponent{
		X: startX,
		Y: startY,
	})

	// 添加 ReanimComponent（卡片包图片 + 植物图标合成）
	cardPackImg := ras.resourceManager.GetImageByID("IMAGE_SEEDPACKET_LARGER")
	if cardPackImg == nil {
		// Fallback: 尝试直接加载
		var err error
		cardPackImg, err = ras.resourceManager.LoadImage("images/SeedPacket_Larger.png")
		if err != nil {
			log.Printf("[RewardAnimationSystem] Warning: Failed to load card pack image: %v", err)
		}
	}

	if cardPackImg != nil {
		// 渲染植物图标到卡片包上（预合成）
		plantIcon := ras.renderPlantIcon(plantID)
		cardPackWithIcon := ras.compositePlantCard(cardPackImg, plantIcon)

		// 创建 ReanimComponent 并使用 ReanimSystem 初始化
		reanimDef := ras.createCardPackReanim()
		reanimComp := &components.ReanimComponent{
			Reanim:            reanimDef,
			PartImages:        map[string]*ebiten.Image{"card_pack": cardPackWithIcon},
			CurrentAnim:       "",
			CurrentFrame:      0,
			FrameAccumulator:  0,
			VisibleFrameCount: 0,
			IsLooping:         true,
		}

		ecs.AddComponent(ras.entityManager, ras.rewardEntity, reanimComp)

		// 使用 ReanimSystem 播放动画
		err := ras.reanimSystem.PlayAnimation(ras.rewardEntity, "idle")
		if err != nil {
			log.Printf("[RewardAnimationSystem] Warning: Failed to play animation: %v", err)
		}

		log.Printf("[RewardAnimationSystem] 卡片包 ReanimComponent 已创建并初始化")
	}

	// TODO: Phase 2 粒子背景框效果（需要查找合适的粒子配置）
}

// Update 更新奖励动画系统，处理各阶段的动画逻辑。
func (ras *RewardAnimationSystem) Update(dt float64) {
	if !ras.isActive {
		return
	}

	// 查询奖励动画实体
	rewardComp, ok := ecs.GetComponent[*components.RewardAnimationComponent](ras.entityManager, ras.rewardEntity)
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
	case "showing":
		ras.updateShowingPhase(dt, rewardComp)
	case "closing":
		ras.updateClosingPhase(dt, rewardComp)
	}
}

// updateAppearingPhase 处理卡片包弹出阶段（0.3秒）。
// - 缩放动画：0.8 → 1.0
// - 微小上升：Y -= 20px
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

	// 应用缓动函数（easeOutQuad - 快速开始，缓慢结束）
	easedProgress := easeOutQuad(progress)

	// 更新缩放
	rewardComp.Scale = RewardAppearScaleStart + (RewardAppearScaleEnd-RewardAppearScaleStart)*easedProgress

	// 更新位置（微小上升）
	posComp.Y = rewardComp.StartY - RewardAppearRiseDistance*easedProgress

	// 检查完成
	if rewardComp.ElapsedTime >= RewardAppearDuration {
		log.Printf("[RewardAnimationSystem] Phase 1 (appearing) 完成，切换到 waiting")
		rewardComp.Phase = "waiting"
		rewardComp.ElapsedTime = 0
		rewardComp.ShowArrow = true // 显示箭头指示器
	}
}

// updateWaitingPhase 处理等待玩家点击阶段。
// - 卡片包静止
// - 箭头闪烁动画
// - TODO: 粒子背景框效果
// - 检测玩家点击
func (ras *RewardAnimationSystem) updateWaitingPhase(dt float64, rewardComp *components.RewardAnimationComponent) {
	// 更新箭头闪烁动画（sin 曲线）
	rewardComp.ArrowAlpha = 0.5 + 0.5*math.Sin(rewardComp.ElapsedTime*RewardArrowBlinkFrequency*2*math.Pi)

	// 检测玩家点击（鼠标或空格键）
	clicked := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace)

	if clicked {
		log.Printf("[RewardAnimationSystem] 玩家点击卡片包，切换到 expanding")
		rewardComp.Phase = "expanding"
		rewardComp.ElapsedTime = 0
		rewardComp.ShowArrow = false // 隐藏箭头

		// TODO: 触发 Award.xml 粒子特效
	}
}

// updateExpandingPhase 处理卡片展开阶段（2秒）。
// - TODO: Award.xml 粒子特效播放
// - 卡片放大：1.0 → 2.0
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

	// 更新缩放
	rewardComp.Scale = RewardExpandScaleStart + (RewardExpandScaleEnd-RewardExpandScaleStart)*easedProgress

	// 更新位置（从当前位置移动到目标位置）
	posComp.X = rewardComp.StartX + (rewardComp.TargetX-rewardComp.StartX)*easedProgress
	posComp.Y = (rewardComp.StartY - RewardAppearRiseDistance) +
		(rewardComp.TargetY-(rewardComp.StartY-RewardAppearRiseDistance))*easedProgress

	// 检查完成
	if rewardComp.ElapsedTime >= RewardExpandDuration {
		log.Printf("[RewardAnimationSystem] Phase 3 (expanding) 完成，切换到 showing")
		rewardComp.Phase = "showing"
		rewardComp.ElapsedTime = 0

		// 创建奖励面板
		ras.createRewardPanel(rewardComp.PlantID)
	}
}

// updateShowingPhase 处理显示奖励面板阶段。
// - 显示新植物介绍面板
// - 等待玩家点击"下一关"或关闭
func (ras *RewardAnimationSystem) updateShowingPhase(dt float64, rewardComp *components.RewardAnimationComponent) {
	// 更新面板动画（如果有）
	if panelComp, ok := ecs.GetComponent[*components.RewardPanelComponent](ras.entityManager, ras.panelEntity); ok {
		// 卡片缩放动画（0.5 → 1.5）
		progress := rewardComp.ElapsedTime / RewardCardScaleDuration
		if progress > 1.0 {
			progress = 1.0
		}
		easedProgress := easeOutQuad(progress)
		panelComp.CardScale = RewardCardScaleStart + (RewardCardScaleEnd-RewardCardScaleStart)*easedProgress
	}

	// 检测玩家点击（鼠标或空格键）
	clicked := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace)

	if clicked {
		log.Printf("[RewardAnimationSystem] 玩家关闭奖励面板，切换到 closing")
		rewardComp.Phase = "closing"
		rewardComp.ElapsedTime = 0
	}
}

// updateClosingPhase 处理关闭奖励面板阶段（0.5秒）。
// - 淡出动画
// - 清理实体
// - TODO: 触发场景切换
func (ras *RewardAnimationSystem) updateClosingPhase(dt float64, rewardComp *components.RewardAnimationComponent) {
	// 检查完成
	if rewardComp.ElapsedTime >= RewardFadeOutDuration {
		log.Printf("[RewardAnimationSystem] Phase 5 (closing) 完成，清理实体")

		// 清理奖励实体
		ras.entityManager.DestroyEntity(ras.rewardEntity)
		if ras.panelEntity != 0 {
			ras.entityManager.DestroyEntity(ras.panelEntity)
		}

		ras.rewardEntity = 0
		ras.panelEntity = 0
		ras.isActive = false

		// TODO: 触发场景切换（返回主菜单或进入下一关）
		log.Printf("[RewardAnimationSystem] 奖励动画完成")
	}
}

// createRewardPanel 创建奖励面板实体。
func (ras *RewardAnimationSystem) createRewardPanel(plantID string) {
	ras.panelEntity = ras.entityManager.CreateEntity()

	// 获取植物信息
	plantInfo := ras.gameState.GetPlantUnlockManager().GetPlantInfo(plantID)

	// 从 LawnStrings 加载实际文本
	plantName := ras.gameState.LawnStrings.GetString(plantInfo.NameKey)
	plantDesc := ras.gameState.LawnStrings.GetString(plantInfo.DescriptionKey)

	// 硬编码阳光值（TODO: 从配置文件读取）
	sunCostMap := map[string]int{
		"sunflower":  50,
		"peashooter": 100,
		"cherrybomb": 150,
		"wallnut":    50,
	}
	sunCost := sunCostMap[plantID]

	// 添加 RewardPanelComponent
	ecs.AddComponent(ras.entityManager, ras.panelEntity, &components.RewardPanelComponent{
		PlantID:          plantID,            // 设置 PlantID，让渲染系统自动加载图标
		PlantName:        plantName,
		PlantDescription: plantDesc,
		SunCost:          sunCost,            // 设置阳光值
		CardScale:        RewardCardScaleStart,
		// Story 8.4: 卡片位置由 RewardPanelRenderSystem 自动计算（水平居中）
		IsVisible:        true,
	})

	log.Printf("[RewardAnimationSystem] 奖励面板已创建：%s - %s", plantName, plantDesc)
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

// renderPlantIcon 使用 ReanimSystem 离屏渲染植物图标。
func (ras *RewardAnimationSystem) renderPlantIcon(plantID string) *ebiten.Image {
	// 根据 plantID 确定 Reanim 名称
	var reanimName string
	switch plantID {
	case "sunflower":
		reanimName = "SunFlower"
	case "peashooter":
		reanimName = "PeaShooter"
	case "cherrybomb":
		reanimName = "CherryBomb"
	case "wallnut":
		reanimName = "Wallnut"
	default:
		log.Printf("[RewardAnimationSystem] Unknown plant ID: %s", plantID)
		return nil
	}

	// 创建临时实体用于离屏渲染
	tempEntity := ras.entityManager.CreateEntity()
	defer func() {
		ras.entityManager.DestroyEntity(tempEntity)
		ras.entityManager.RemoveMarkedEntities()
	}()

	// 加载 Reanim 资源
	reanimXML := ras.resourceManager.GetReanimXML(reanimName)
	partImages := ras.resourceManager.GetReanimPartImages(reanimName)

	if reanimXML == nil || partImages == nil {
		log.Printf("[RewardAnimationSystem] Failed to load Reanim resources for %s", reanimName)
		return nil
	}

	// 添加 PositionComponent
	iconCenterX := 50.0
	iconCenterY := 70.0
	ecs.AddComponent(ras.entityManager, tempEntity, &components.PositionComponent{
		X: iconCenterX,
		Y: iconCenterY,
	})

	// 添加 ReanimComponent
	ecs.AddComponent(ras.entityManager, tempEntity, &components.ReanimComponent{
		Reanim:     reanimXML,
		PartImages: partImages,
	})

	// 播放 idle 动画
	if err := ras.reanimSystem.PlayAnimation(tempEntity, "anim_idle"); err != nil {
		log.Printf("[RewardAnimationSystem] Failed to play animation: %v", err)
		return nil
	}

	// 创建离屏画布
	canvas := ebiten.NewImage(100, 140)

	// 更新 Reanim 动画到第0帧
	ras.reanimSystem.Update(0.01)

	// 使用 RenderSystem 渲染（需要传入 entities 列表）
	// 注意：这里简化实现，实际需要调用完整的渲染流程
	// TODO: 完整实现离屏渲染

	return canvas
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
