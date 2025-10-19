package systems

import (
	"log"
	"math"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	// 抛物线动画物理参数
	RewardGravity          = 800.0  // 重力加速度（像素/秒²）
	RewardInitialVelocityX = -200.0 // 初始水平速度（向左）
	RewardInitialVelocityY = -400.0 // 初始垂直速度（向上）

	// 弹跳动画参数
	RewardBounceFrequency      = 8.0  // 弹跳频率（Hz）
	RewardBounceAmplitude      = 30.0 // 初始弹跳振幅（像素）
	RewardBounceDecay          = 0.6  // 振幅衰减系数
	RewardMaxBounces           = 3    // 弹跳次数
	RewardBounceDuration       = 0.5  // 单次弹跳持续时间（秒）

	// 卡片展开动画参数
	RewardCardScaleStart    = 0.5  // 卡片初始缩放
	RewardCardScaleEnd      = 1.5  // 卡片最终缩放
	RewardCardScaleDuration = 1.0  // 卡片缩放动画持续时间（秒）

	// 淡出动画参数
	RewardFadeOutDuration = 0.5 // 淡出持续时间（秒）
)

// RewardAnimationSystem 管理关卡完成后的奖励动画流程。
// 负责卡片包掉落、弹跳、展开、显示奖励面板等阶段的状态管理和动画更新。
type RewardAnimationSystem struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager
	reanimSystem    *ReanimSystem    // Reanim系统用于创建和管理动画
	rewardEntity    ecs.EntityID // 奖励动画实体ID（卡片包）
	panelEntity     ecs.EntityID // 奖励面板实体ID
	isActive        bool         // 系统是否激活
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

// TriggerReward 触发奖励动画，开始卡片包掉落流程。
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

	// 计算起点和终点
	startX := ras.screenWidth + 100   // 屏幕右侧上方
	startY := 100.0
	targetX := ras.screenWidth / 2    // 草坪中央
	targetY := ras.screenHeight / 2 + 100

	// 添加 RewardAnimationComponent
	ecs.AddComponent(ras.entityManager, ras.rewardEntity, &components.RewardAnimationComponent{
		Phase:                  "dropping",
		ElapsedTime:            0,
		StartX:                 startX,
		StartY:                 startY,
		TargetX:                targetX,
		TargetY:                targetY,
		VelocityX:              RewardInitialVelocityX,
		VelocityY:              RewardInitialVelocityY,
		PlantID:                plantID,
		BounceCount:            0,
		InitialBounceAmplitude: RewardBounceAmplitude,
	})

	// 添加 PositionComponent
	ecs.AddComponent(ras.entityManager, ras.rewardEntity, &components.PositionComponent{
		X: startX,
		Y: startY,
	})

	// 添加 ReanimComponent（卡片包图片 + 植物图标合成）
	// Story 6.3: 所有游戏世界实体使用 ReanimComponent 渲染
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

		// 使用 ReanimSystem 播放动画（这会初始化 AnimTracks 和 MergedTracks）
		err := ras.reanimSystem.PlayAnimation(ras.rewardEntity, "idle")
		if err != nil {
			log.Printf("[RewardAnimationSystem] Warning: Failed to play animation: %v", err)
		}

		log.Printf("[RewardAnimationSystem] 卡片包 ReanimComponent 已创建并初始化")
	}

	// TODO: 触发 Award.xml 粒子特效
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
	case "dropping":
		ras.updateDroppingPhase(dt, rewardComp)
	case "bouncing":
		ras.updateBouncingPhase(dt, rewardComp)
	case "expanding":
		ras.updateExpandingPhase(dt, rewardComp)
	case "showing":
		ras.updateShowingPhase(dt, rewardComp)
	case "closing":
		ras.updateClosingPhase(dt, rewardComp)
	}
}

// updateDroppingPhase 处理抛物线掉落阶段。
func (ras *RewardAnimationSystem) updateDroppingPhase(dt float64, rewardComp *components.RewardAnimationComponent) {
	// 获取位置组件
	posComp, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.rewardEntity)
	if !ok {
		return
	}

	// 应用重力
	rewardComp.VelocityY += RewardGravity * dt

	// 更新位置
	posComp.X += rewardComp.VelocityX * dt
	posComp.Y += rewardComp.VelocityY * dt

	// 检测到达目标位置
	if posComp.Y >= rewardComp.TargetY {
		// 修正位置到目标点
		posComp.X = rewardComp.TargetX
		posComp.Y = rewardComp.TargetY

		// 切换到弹跳阶段
		rewardComp.Phase = "bouncing"
		rewardComp.ElapsedTime = 0
		rewardComp.BounceCount = 0

		log.Printf("[RewardAnimationSystem] 卡片包到达目标位置，开始弹跳动画")
	}
}

// updateBouncingPhase 处理弹跳动画阶段。
func (ras *RewardAnimationSystem) updateBouncingPhase(dt float64, rewardComp *components.RewardAnimationComponent) {
	// 获取位置组件
	posComp, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.rewardEntity)
	if !ok {
		return
	}

	// 计算弹跳偏移
	bounceTime := rewardComp.ElapsedTime
	amplitude := rewardComp.InitialBounceAmplitude * math.Pow(RewardBounceDecay, float64(rewardComp.BounceCount))
	offsetY := amplitude * math.Abs(math.Sin(bounceTime*RewardBounceFrequency*math.Pi))

	// 更新位置（基于目标位置）
	posComp.Y = rewardComp.TargetY - offsetY

	// 检查单次弹跳是否完成
	if bounceTime >= RewardBounceDuration {
		rewardComp.BounceCount++
		rewardComp.ElapsedTime = 0

		// 检查是否完成所有弹跳
		if rewardComp.BounceCount >= RewardMaxBounces {
			// 切换到展开阶段
			rewardComp.Phase = "expanding"
			rewardComp.ElapsedTime = 0

			// 重置位置到目标点
			posComp.Y = rewardComp.TargetY

			log.Printf("[RewardAnimationSystem] 弹跳动画完成，等待玩家点击")
		}
	}
}

// updateExpandingPhase 处理等待玩家点击展开阶段。
func (ras *RewardAnimationSystem) updateExpandingPhase(dt float64, rewardComp *components.RewardAnimationComponent) {
	// 检测玩家点击
	if ras.handleClick() {
		// 切换到显示奖励面板阶段
		rewardComp.Phase = "showing"
		rewardComp.ElapsedTime = 0

		// 创建奖励面板
		ras.createRewardPanel(rewardComp.PlantID)

		// TODO: 触发卡片展开粒子特效

		log.Printf("[RewardAnimationSystem] 玩家点击卡片包，显示奖励面板")
	}
}

// updateShowingPhase 处理显示奖励面板阶段。
func (ras *RewardAnimationSystem) updateShowingPhase(dt float64, rewardComp *components.RewardAnimationComponent) {
	// 更新面板动画
	panelComp, ok := ecs.GetComponent[*components.RewardPanelComponent](ras.entityManager, ras.panelEntity)
	if !ok {
		return
	}

	panelComp.AnimationTime += dt

	// 卡片缩放动画
	progress := math.Min(panelComp.AnimationTime/RewardCardScaleDuration, 1.0)
	panelComp.CardScale = RewardCardScaleStart + (RewardCardScaleEnd-RewardCardScaleStart)*ras.easeInOutQuad(progress)

	// 淡入动画
	panelComp.FadeAlpha = math.Min(panelComp.AnimationTime/0.5, 1.0)

	// 检测玩家点击关闭
	if panelComp.AnimationTime > 0.5 && ras.handleClick() {
		// 切换到关闭阶段
		rewardComp.Phase = "closing"
		rewardComp.ElapsedTime = 0

		log.Printf("[RewardAnimationSystem] 玩家点击关闭奖励面板")
	}
}

// updateClosingPhase 处理关闭奖励面板阶段。
func (ras *RewardAnimationSystem) updateClosingPhase(dt float64, rewardComp *components.RewardAnimationComponent) {
	// 更新淡出动画
	panelComp, ok := ecs.GetComponent[*components.RewardPanelComponent](ras.entityManager, ras.panelEntity)
	if ok {
		progress := rewardComp.ElapsedTime / RewardFadeOutDuration
		panelComp.FadeAlpha = 1.0 - progress

		if progress >= 1.0 {
			// 清理实体
			ras.entityManager.DestroyEntity(ras.rewardEntity)
			ras.entityManager.DestroyEntity(ras.panelEntity)
			ras.rewardEntity = 0
			ras.panelEntity = 0
			ras.isActive = false

			// TODO: 触发场景切换（返回主菜单）
			log.Printf("[RewardAnimationSystem] 奖励动画完成，准备返回主菜单")
		}
	}
}

// createRewardPanel 创建奖励面板实体。
func (ras *RewardAnimationSystem) createRewardPanel(plantID string) {
	ras.panelEntity = ras.entityManager.CreateEntity()

	// 从 PlantUnlockManager 和 LawnStrings 获取植物信息
	plantName, plantDesc := ras.getPlantInfo(plantID)

	// 渲染植物图标（使用 ReanimSystem 离屏渲染）
	plantIcon := ras.renderPlantIcon(plantID)
	if plantIcon != nil {
		log.Printf("[RewardAnimationSystem] 植物图标渲染成功：%dx%d", plantIcon.Bounds().Dx(), plantIcon.Bounds().Dy())
	} else {
		log.Printf("[RewardAnimationSystem] 警告：植物图标渲染失败")
	}

	// 计算卡片位置（使用配置中的位置比例）
	// 卡片位置基于800x600的奖励背景，需要计算背景在屏幕上的偏移
	bgWidth := config.RewardPanelBackgroundWidth
	bgHeight := config.RewardPanelBackgroundHeight
	offsetX := (ras.screenWidth - bgWidth) / 2
	offsetY := (ras.screenHeight - bgHeight) / 2

	cardX := offsetX + bgWidth*config.RewardPanelCardX   // 使用配置的X位置
	cardY := offsetY + bgHeight*config.RewardPanelCardY  // 使用配置的Y位置

	// 添加 RewardPanelComponent
	ecs.AddComponent(ras.entityManager, ras.panelEntity, &components.RewardPanelComponent{
		PlantID:          plantID,
		PlantName:        plantName,
		PlantDescription: plantDesc,
		SunCost:          game.GetPlantSunCost(plantID), // 从 config 获取阳光值
		PlantIconTexture: plantIcon,                     // 添加植物图标纹理
		CardScale:        RewardCardScaleStart,
		CardX:            cardX,
		CardY:            cardY,
		IsVisible:        true,
		FadeAlpha:        0,
		AnimationTime:    0,
	})

	log.Printf("[RewardAnimationSystem] 创建奖励面板：%s - %s (阳光: %d)", plantName, plantDesc, game.GetPlantSunCost(plantID))
}

// getPlantInfo 获取植物名称和描述。
func (ras *RewardAnimationSystem) getPlantInfo(plantID string) (name, desc string) {
	// 从 PlantUnlockManager 获取植物信息
	plantInfo := ras.gameState.GetPlantUnlockManager().GetPlantInfo(plantID)

	log.Printf("[RewardAnimationSystem] 植物信息 - NameKey: %s, DescriptionKey: %s", plantInfo.NameKey, plantInfo.DescriptionKey)

	// 从 LawnStrings 加载本地化文本
	if ras.gameState.LawnStrings != nil {
		name = ras.gameState.LawnStrings.GetString(plantInfo.NameKey)
		desc = ras.gameState.LawnStrings.GetString(plantInfo.DescriptionKey)
		log.Printf("[RewardAnimationSystem] LawnStrings 查询结果 - Name: '%s', Desc: '%s'", name, desc)
	} else {
		log.Printf("[RewardAnimationSystem] WARNING: LawnStrings is nil!")
	}

	// 如果加载失败，使用默认值
	if name == "" {
		name = plantInfo.NameKey
		log.Printf("[RewardAnimationSystem] Name 为空，使用 NameKey: %s", name)
	}
	if desc == "" {
		desc = plantInfo.DescriptionKey
		log.Printf("[RewardAnimationSystem] Desc 为空，使用 DescriptionKey: %s", desc)
	}

	return name, desc
}

// handleClick 检测玩家点击（鼠标或触摸）。
func (ras *RewardAnimationSystem) handleClick() bool {
	return inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
}

// easeInOutQuad 二次缓动函数。
func (ras *RewardAnimationSystem) easeInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}

// IsActive 返回系统是否激活。
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
// 参考 plant_card_factory.go 的 renderPlantIcon 实现。
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

	// 添加 PositionComponent (设置为纹理中心，考虑 Reanim 的 CenterOffset)
	// 由于 RenderSystem 会减去 CenterOffset，我们需要将位置设置为纹理中心
	// 这样 screenX = Position.X - CenterOffsetX 会得到正确的绘制原点
	iconCenterX := 50.0 // 纹理宽度的一半 (100/2)
	iconCenterY := 70.0 // 纹理高度的一半 (140/2)
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

	// 离屏渲染到纹理 (100x140 用于奖励卡片，比普通卡片大)
	iconTexture := ebiten.NewImage(100, 140)
	ras.reanimSystem.RenderToTexture(tempEntity, iconTexture)

	return iconTexture
}

// compositePlantCard 将植物图标合成到卡片包背景上。
// 参数:
//   - cardPack: 卡片包背景图片（SeedPacket_Larger.png）
//   - plantIcon: 植物图标纹理（100x140，Reanim 离屏渲染）
//
// 返回:
//   - 合成后的图片（卡片包 + 居中的植物图标）
func (ras *RewardAnimationSystem) compositePlantCard(cardPack, plantIcon *ebiten.Image) *ebiten.Image {
	if cardPack == nil {
		return plantIcon
	}
	if plantIcon == nil {
		return cardPack
	}

	// 创建合成图片（使用卡片包的尺寸）
	cardWidth := cardPack.Bounds().Dx()
	cardHeight := cardPack.Bounds().Dy()
	composite := ebiten.NewImage(cardWidth, cardHeight)

	// 1. 绘制卡片包背景
	composite.DrawImage(cardPack, &ebiten.DrawImageOptions{})

	// 2. 绘制植物图标（居中对齐）
	iconWidth := plantIcon.Bounds().Dx()
	iconHeight := plantIcon.Bounds().Dy()

	iconOp := &ebiten.DrawImageOptions{}
	// 居中对齐：卡片中心 - 图标中心
	iconX := float64(cardWidth-iconWidth) / 2
	iconY := float64(cardHeight-iconHeight) / 2
	iconOp.GeoM.Translate(iconX, iconY)

	composite.DrawImage(plantIcon, iconOp)

	log.Printf("[RewardAnimationSystem] 合成卡片包图片：卡片尺寸=%dx%d, 图标尺寸=%dx%d, 图标位置=(%.1f, %.1f)",
		cardWidth, cardHeight, iconWidth, iconHeight, iconX, iconY)

	return composite
}
