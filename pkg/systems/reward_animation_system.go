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
)

const (
	// Phase 1 - Appearing (出现阶段)
	RewardAppearDuration   = 0.6  // 出现动画持续时间（秒）
	RewardAppearJumpHeight = 40.0 // 微小跳跃高度（像素）

	// Phase 3 - Expanding (展开阶段)
	RewardExpandDuration     = 2.0  // 展开动画持续时间（秒）
	RewardExpandScaleEnd     = 1.0  // 最终缩放（放大到原始大小）
	RewardExpandTargetYRatio = 0.45 // 目标Y位置（screenHeight * ratio）- 草坪上方中央

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
	glowEntity      ecs.EntityID  // 光晕粒子发射器实体ID
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
		glowEntity:      0,
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

	// 随机选择草坪行（2-4行，偏中间位置）
	randomLane := 1 + rand.Intn(3) // 第2、3或4行

	// 计算起始位置（草坪偏右半部分随机位置，确保不超出草坪范围）
	// 草坪共9列（0-8），选择第4-6列（偏右但不超出草坪）
	randomCol := 3 + rand.Intn(3) // 第4、5或第6列
	startX := config.GridWorldStartX + float64(randomCol)*config.CellWidth + config.CellWidth/2.0
	startY := config.GridWorldStartY + float64(randomLane)*config.CellHeight + config.CellHeight/2.0

	// 计算 Phase 3 目标位置（屏幕可见区域中央上方）
	//
	// 设计说明：
	//   - 用户期望：卡片在屏幕可见区域的水平中央
	//   - 屏幕可见区域（世界坐标）：[CameraX, CameraX + ScreenWidth]
	//   - 屏幕中央（世界坐标）= CameraX + ScreenWidth/2 = 220 + 800/2 = 620
	//
	// 计算流程：
	//   1. 计算屏幕中央的世界坐标
	//   2. 减去半个卡片宽度，得到卡片左上角X坐标
	//   3. Y坐标：草坪顶部上方约50像素
	screenCenterX := ras.gameState.CameraX + ras.screenWidth/2.0 // 220 + 400 = 620
	lawnTopY := config.GridWorldStartY                           // 76

	// 注意：PositionComponent 是卡片左上角坐标，需要减去半个卡片宽度使卡片中心对齐
	// 卡片原始尺寸 100x140，在 expanding 结束时，CardScale = 1.0，所以卡片宽度 = 100
	cardWidthAtEnd := 100.0                          // RewardExpandScaleEnd = 1.0 时的卡片宽度
	targetX := screenCenterX - cardWidthAtEnd/2.0    // 卡片左上角X = 屏幕中心X - 半宽 = 620 - 50 = 570
	targetY := lawnTopY - 50.0                       // 草坪顶部上方50像素

	log.Printf("[RewardAnimationSystem] 卡片包起始位置（草坪格子）: (%.1f, %.1f), 目标位置: (%.1f, %.1f), 随机行: %d, 随机列: %d",
		startX, startY, targetX, targetY, randomLane, randomCol)

	// 添加 RewardAnimationComponent
	ecs.AddComponent(ras.entityManager, ras.rewardEntity, &components.RewardAnimationComponent{
		Phase:       "appearing",
		ElapsedTime: 0,
		StartX:      startX,
		StartY:      startY,
		TargetX:     targetX,
		TargetY:     targetY,
		Scale:       config.PlantCardScale, // 使用标准卡片缩放（0.50）
		PlantID:     plantID,
	})

	// 添加 PositionComponent
	ecs.AddComponent(ras.entityManager, ras.rewardEntity, &components.PositionComponent{
		X: startX,
		Y: startY,
	})

	// Story 8.4: 使用 NewPlantCardEntity 创建卡片包
	// 这样可以自动获得：背景图、植物图标、阳光数字等完整渲染
	plantType := ras.plantIDToType(plantID)
	if plantType == components.PlantUnknown {
		log.Printf("[RewardAnimationSystem] Unknown plant ID: %s, aborting", plantID)
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

	// 重要：使用卡片实体作为奖励实体
	ras.entityManager.DestroyEntity(ras.rewardEntity)
	ras.rewardEntity = cardEntity

	// 重新添加 RewardAnimationComponent（控制动画状态）
	ecs.AddComponent(ras.entityManager, ras.rewardEntity, &components.RewardAnimationComponent{
		Phase:       "appearing",
		ElapsedTime: 0,
		StartX:      startX,
		StartY:      startY,
		TargetX:     targetX,
		TargetY:     targetY,
		Scale:       config.PlantCardScale, // 使用标准卡片缩放（0.50）
		PlantID:     plantID,
	})

	log.Printf("[RewardAnimationSystem] 卡片包已创建（使用 Story 8.4 统一工厂方法）")

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

		// 创建光晕粒子效果（包含光晕 + 向下箭头）
		if ras.glowEntity == 0 {
			// 验证猜想：有Name的发射器是"主粒子"
			// SeedPacketGlow 有 EmitterOffsetY=62，如果它应该在卡片中心
			// 那么基准位置应该是：卡片中心 - 62
			cardComp, _ := ecs.GetComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity)
			particleX := posComp.X
			particleY := posComp.Y
			if cardComp != nil && cardComp.BackgroundImage != nil {
				cardWidth := float64(cardComp.BackgroundImage.Bounds().Dx()) * cardComp.CardScale
				cardHeight := float64(cardComp.BackgroundImage.Bounds().Dy()) * cardComp.CardScale
				particleX += cardWidth / 2.0   // X：卡片水平中心
				particleY += cardHeight / 2.0  // Y：卡片垂直中心
				particleY -= 62.0              // 减去主粒子的偏移量，使主粒子在中心
			}

			glowID, err := entities.CreateParticleEffect(
				ras.entityManager,
				ras.resourceManager,
				"SeedPacket",
				particleX,
				particleY,
				0.0,  // angleOffset = 0
				true, // isUIParticle = true
			)
			if err != nil {
				log.Printf("[RewardAnimationSystem] 创建光晕粒子失败: %v", err)
			} else {
				ras.glowEntity = glowID
				log.Printf("[RewardAnimationSystem] 创建光晕粒子成功: ID=%d（验证主粒子假设，基准位置=(%.1f, %.1f)）", glowID, particleX, particleY)
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

			// 清理发射器实体（SeedPacket有2个发射器，需要查找并清理所有相关发射器）
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

	// 检查完成
	if rewardComp.ElapsedTime >= RewardExpandDuration {
		log.Printf("[RewardAnimationSystem] Phase 3 (expanding) 移动完成，触发 Award 粒子特效")

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

			// 创建 Award 粒子特效（12个光芒发射器）
			awardID, err := entities.CreateParticleEffect(
				ras.entityManager,
				ras.resourceManager,
				"Award",
				particleX,
				particleY,
				0.0,  // angleOffset = 0
				true, // isUIParticle = true
			)
			if err != nil {
				log.Printf("[RewardAnimationSystem] 创建 Award 粒子特效失败: %v", err)
			} else {
				// 保存 Award 粒子ID
				ras.glowEntity = awardID
				log.Printf("[RewardAnimationSystem] 创建 Award 粒子特效成功: ID=%d（13个发射器：8光芒+1光晕+4闪光）", awardID)
			}
		}

		rewardComp.Phase = "showing"
		rewardComp.ElapsedTime = 0

		// 创建奖励面板
		ras.createRewardPanel(rewardComp.PlantID)
	}
}

// updateShowingPhase 处理显示奖励面板阶段。
// - 显示新植物介绍面板
// - 需要玩家手动点击才能关闭（无自动播放）
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

	// 检测玩家点击（鼠标或空格键）- 手动关闭
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
		if ras.glowEntity != 0 {
			ras.entityManager.DestroyEntity(ras.glowEntity)
		}

		ras.rewardEntity = 0
		ras.panelEntity = 0
		ras.glowEntity = 0
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
		return "PeaShooter"
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
