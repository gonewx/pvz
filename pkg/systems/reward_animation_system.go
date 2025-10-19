package systems

import (
	"log"
	"math"

	"github.com/decker502/pvz/pkg/components"
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
	rewardEntity    ecs.EntityID // 奖励动画实体ID（卡片包）
	panelEntity     ecs.EntityID // 奖励面板实体ID
	isActive        bool         // 系统是否激活
	screenWidth     float64
	screenHeight    float64
}

// NewRewardAnimationSystem 创建新的奖励动画系统。
func NewRewardAnimationSystem(em *ecs.EntityManager, gs *game.GameState, rm *game.ResourceManager) *RewardAnimationSystem {
	return &RewardAnimationSystem{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
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

	// TODO: 添加 SpriteComponent（卡片包图片）
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

	// 计算卡片位置（屏幕上方中央）
	cardX := ras.screenWidth / 2
	cardY := ras.screenHeight * 0.35

	// 添加 RewardPanelComponent
	ecs.AddComponent(ras.entityManager, ras.panelEntity, &components.RewardPanelComponent{
		PlantID:          plantID,
		PlantName:        plantName,
		PlantDescription: plantDesc,
		CardScale:        RewardCardScaleStart,
		CardX:            cardX,
		CardY:            cardY,
		IsVisible:        true,
		FadeAlpha:        0,
		AnimationTime:    0,
	})

	log.Printf("[RewardAnimationSystem] 创建奖励面板：%s - %s", plantName, plantDesc)
}

// getPlantInfo 获取植物名称和描述。
func (ras *RewardAnimationSystem) getPlantInfo(plantID string) (name, desc string) {
	// 从 PlantUnlockManager 获取植物信息
	plantInfo := ras.gameState.GetPlantUnlockManager().GetPlantInfo(plantID)

	// 从 LawnStrings 加载本地化文本
	if ras.gameState.LawnStrings != nil {
		name = ras.gameState.LawnStrings.GetString(plantInfo.NameKey)
		desc = ras.gameState.LawnStrings.GetString(plantInfo.DescriptionKey)
	}

	// 如果加载失败，使用默认值
	if name == "" {
		name = plantInfo.NameKey
	}
	if desc == "" {
		desc = plantInfo.DescriptionKey
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
