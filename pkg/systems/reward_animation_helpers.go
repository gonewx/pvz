package systems

import (
	"image/color"
	"log"

	"github.com/gonewx/pvz/internal/reanim"
	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// ============================================================================
// 奖励动画辅助方法和渲染
// ============================================================================

// IsActive 返回系统是否正在播放动画。
func (ras *RewardAnimationSystem) IsActive() bool {
	return ras.isActive
}

// IsCompleted 返回系统是否已完成。
func (ras *RewardAnimationSystem) IsCompleted() bool {
	return !ras.isActive && ras.rewardEntity == 0
}

// isCardPackClicked 检查鼠标点击是否在卡包范围内（Story 8.13）
// 用于 waiting 阶段的精确点击检测，只有点击卡包本身才触发展开动画
// 参数：mouseX, mouseY - 鼠标点击位置（屏幕坐标）
// 返回：true 如果点击在卡包范围内
func (ras *RewardAnimationSystem) isCardPackClicked(mouseX, mouseY float64) bool {
	if ras.rewardEntity == 0 {
		return false
	}

	posComp, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.rewardEntity)
	if !ok {
		return false
	}

	// 获取奖励组件，确定奖励类型
	rewardComp, ok := ecs.GetComponent[*components.RewardAnimationComponent](ras.entityManager, ras.rewardEntity)
	if !ok {
		return false
	}

	// 根据奖励类型计算边界框
	if rewardComp.RewardType == "tool" {
		// 工具图标：使用铲子图片尺寸 (116 x 125) * scale
		// 图标以中心点为锚点绘制
		width := 116.0 * rewardComp.Scale
		height := 125.0 * rewardComp.Scale
		// 计算边界框（以中心点为锚点）
		left := posComp.X - width/2
		top := posComp.Y - height/2
		right := posComp.X + width/2
		bottom := posComp.Y + height/2
		return mouseX >= left && mouseX <= right &&
			mouseY >= top && mouseY <= bottom
	}

	if rewardComp.RewardType == "note" {
		// 来信奖励：使用 ZombieNoteSmall.png 尺寸 (78 x 52) * scale
		// 图标以中心点为锚点绘制
		width := 78.0 * rewardComp.Scale
		height := 52.0 * rewardComp.Scale
		// 计算边界框（以中心点为锚点）
		left := posComp.X - width/2
		top := posComp.Y - height/2
		right := posComp.X + width/2
		bottom := posComp.Y + height/2
		return mouseX >= left && mouseX <= right &&
			mouseY >= top && mouseY <= bottom
	}

	// 植物卡片：使用卡片尺寸
	// 尝试从 PlantCardComponent 获取精确尺寸
	if cardComp, ok := ecs.GetComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity); ok && cardComp.BackgroundImage != nil {
		cardWidth := float64(cardComp.BackgroundImage.Bounds().Dx()) * cardComp.CardScale
		cardHeight := float64(cardComp.BackgroundImage.Bounds().Dy()) * cardComp.CardScale
		// 卡片以左上角为锚点绘制
		return mouseX >= posComp.X && mouseX <= posComp.X+cardWidth &&
			mouseY >= posComp.Y && mouseY <= posComp.Y+cardHeight
	}

	// 默认使用 RewardAnimationComponent 的缩放值
	width := 100.0 * rewardComp.Scale
	height := 140.0 * rewardComp.Scale
	return mouseX >= posComp.X && mouseX <= posComp.X+width &&
		mouseY >= posComp.Y && mouseY <= posComp.Y+height
}

// IsHoveringReward 检查鼠标是否悬停在奖励图标上（用于光标显示）
// 在 waiting 阶段悬停在卡片/工具图标上时返回 true
func (ras *RewardAnimationSystem) IsHoveringReward() bool {
	if !ras.isActive || ras.rewardEntity == 0 {
		return false
	}

	// 只在 waiting 阶段检测悬停
	rewardComp, ok := ecs.GetComponent[*components.RewardAnimationComponent](ras.entityManager, ras.rewardEntity)
	if !ok || rewardComp.Phase != "waiting" {
		return false
	}

	// 获取鼠标位置
	mouseX, mouseY := utils.GetPointerPosition()

	// 获取位置组件
	posComp, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.rewardEntity)
	if !ok {
		return false
	}

	// 根据奖励类型计算边界框
	var width, height float64
	if rewardComp.RewardType == "tool" {
		// 工具图标：使用铲子图片尺寸 (116 x 125) * scale
		// 图标以中心点为锚点绘制
		width = 116.0 * rewardComp.Scale
		height = 125.0 * rewardComp.Scale
		// 计算边界框（以中心点为锚点）
		left := posComp.X - width/2
		top := posComp.Y - height/2
		right := posComp.X + width/2
		bottom := posComp.Y + height/2
		return float64(mouseX) >= left && float64(mouseX) <= right &&
			float64(mouseY) >= top && float64(mouseY) <= bottom
	} else if rewardComp.RewardType == "note" {
		// 来信奖励：使用 ZombieNoteSmall.png 尺寸 (78 x 52) * scale
		// 图标以中心点为锚点绘制
		width = 78.0 * rewardComp.Scale
		height = 52.0 * rewardComp.Scale
		// 计算边界框（以中心点为锚点）
		left := posComp.X - width/2
		top := posComp.Y - height/2
		right := posComp.X + width/2
		bottom := posComp.Y + height/2
		return float64(mouseX) >= left && float64(mouseX) <= right &&
			float64(mouseY) >= top && float64(mouseY) <= bottom
	} else {
		// 植物卡片：使用卡片尺寸 (100 x 140) * scale
		// 卡片以左上角为锚点绘制
		width = 100.0 * rewardComp.Scale
		height = 140.0 * rewardComp.Scale
		left := posComp.X
		top := posComp.Y
		right := posComp.X + width
		bottom := posComp.Y + height
		return float64(mouseX) >= left && float64(mouseX) <= right &&
			float64(mouseY) >= top && float64(mouseY) <= bottom
	}
}

// IsHoveringNextButton 检查鼠标是否悬停在"下一关"按钮上
// 在 showing 阶段悬停在按钮上时返回 true
func (ras *RewardAnimationSystem) IsHoveringNextButton() bool {
	if !ras.isActive || ras.currentPhase != "showing" {
		return false
	}

	// 获取鼠标位置
	mouseX, mouseY := utils.GetPointerPosition()

	// 使用与 isNextLevelButtonClicked 相同的按钮位置计算
	return ras.isNextLevelButtonClicked(float64(mouseX), float64(mouseY))
}

// GetCursorShape 返回当前应显示的光标形状
// 调用者应在 Update 循环中调用此方法并设置光标
func (ras *RewardAnimationSystem) GetCursorShape() ebiten.CursorShapeType {
	if !ras.isActive {
		return ebiten.CursorShapeDefault
	}

	// waiting 阶段：悬停在卡片/图标上时显示手形
	if ras.IsHoveringReward() {
		return ebiten.CursorShapePointer
	}

	// showing 阶段：悬停在"下一关"按钮或"主菜单"按钮上时显示手形
	if ras.IsHoveringNextButton() {
		return ebiten.CursorShapePointer
	}

	// Story 8.13: 悬停在"主菜单"按钮上时显示手形
	if ras.panelRenderSystem != nil && ras.panelRenderSystem.IsHoveringMainMenuButton() {
		return ebiten.CursorShapePointer
	}

	return ebiten.CursorShapeDefault
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

// createAwardParticle 创建 Award 粒子特效（在玩家点击卡包后立即触发）
// 粒子效果会根据奖励类型自动选择：
//   - 植物奖励：Award（12个光芒发射器，360°星爆效果）
//   - 工具奖励：Award（与植物奖励一致）
//   - 来信奖励：Starburst（Story 8.14）
func (ras *RewardAnimationSystem) createAwardParticle(rewardComp *components.RewardAnimationComponent) {
	// 获取卡片位置
	posComp, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.rewardEntity)
	if !ok {
		log.Printf("[RewardAnimationSystem] Warning: Failed to get position for particle creation")
		return
	}

	// 获取卡片组件（用于计算视觉中心）
	cardComp, _ := ecs.GetComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity)

	// 计算卡片视觉中心坐标（基于当前缩放）
	particleX := posComp.X
	particleY := posComp.Y
	if cardComp != nil && cardComp.BackgroundImage != nil {
		cardWidth := float64(cardComp.BackgroundImage.Bounds().Dx()) * cardComp.CardScale
		cardHeight := float64(cardComp.BackgroundImage.Bounds().Dy()) * cardComp.CardScale
		particleX += cardWidth / 2.0  // X：卡片水平中心
		particleY += cardHeight / 2.0 // Y：卡片垂直中心
	} else {
		// 对于工具/来信奖励（使用 SpriteComponent），计算精灵中心
		if spriteComp, ok := ecs.GetComponent[*components.SpriteComponent](ras.entityManager, ras.rewardEntity); ok && spriteComp.Image != nil {
			spriteWidth := float64(spriteComp.Image.Bounds().Dx()) * rewardComp.Scale
			spriteHeight := float64(spriteComp.Image.Bounds().Dy()) * rewardComp.Scale
			particleX += spriteWidth / 2.0
			particleY += spriteHeight / 2.0
		}
	}

	// Story 8.14: 根据奖励类型选择粒子效果
	var particleEffectName string
	switch rewardComp.RewardType {
	case "note":
		particleEffectName = "Starburst" // 来信奖励使用星爆效果
	default:
		particleEffectName = "Award" // 植物/工具奖励使用12光芒效果
	}

	// 创建粒子特效
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
		log.Printf("[RewardAnimationSystem] 创建粒子特效成功: %s, ID=%d（点击后立即出现）", particleEffectName, awardID)
	}
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
// 使用统一的植物注册表，新增植物时只需在 config/plant_registry.go 中添加记录
func (ras *RewardAnimationSystem) plantIDToType(plantID string) components.PlantType {
	return config.PlantIDToType(plantID)
}

// getReanimName 根据 plantID 获取 Reanim 资源名称
// 使用统一的植物注册表，新增植物时只需在 config/plant_registry.go 中添加记录
func (ras *RewardAnimationSystem) getReanimName(plantID string) string {
	return config.GetPlantReanimNameByID(plantID)
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
		// CreateParticleEffect 创建的所有发射器共享同一个 PositionComponent
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
// - 粒子效果（SeedPacket 背景框 + Award 爆炸特效）
// - 植物卡片（Phase 1-3 的卡片包）
// - 奖励面板（Phase 4）
//
// 渲染顺序（从下到上）：
// 1. 粒子效果（装饰层）
// 2. 植物卡片 / 奖励面板（最上层，不被粒子遮挡）
//
// 注意：
// - 游戏世界实体（植物、僵尸、阴影）由 GameScene.Draw() 的 Layer 4 渲染
// - 奖励动画的粒子标记为 UIComponent（isUIParticle=true）
// - GameScene Layer 6 会过滤掉 UI 粒子，只渲染游戏世界粒子
func (ras *RewardAnimationSystem) Draw(screen *ebiten.Image) {
	if !ras.isActive {
		return
	}

	cameraOffsetX := ras.gameState.CameraX

	// 注意：不再调用 renderSystem.Draw()，因为：
	// 1. 游戏世界实体（植物、僵尸、阴影）已经在 GameScene.Draw() 的 Layer 4 渲染过了
	// 2. renderSystem.Draw() 会渲染所有实体，导致阴影等重复绘制
	// 奖励动画只需要渲染自己的粒子效果和卡片/面板

	// 1. 绘制粒子效果（SeedPacket 背景框 + Award 爆炸）
	//    只渲染 UI 粒子（奖励动画的粒子），不渲染游戏世界粒子
	ras.renderSystem.DrawParticles(screen, cameraOffsetX)

	// 2a. Phase 1-3: 渲染奖励卡片/工具图标（appearing/waiting/expanding/pausing/disappearing）在最上层
	// 直接渲染自己管理的卡片实体（符合 ECS 原则：系统负责自己的实体）
	if ras.currentPhase != "showing" && ras.currentPhase != "closing" && ras.rewardEntity != 0 {
		// 植物奖励：渲染植物卡片
		if card, ok := ecs.GetComponent[*components.PlantCardComponent](ras.entityManager, ras.rewardEntity); ok {
			if pos, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.rewardEntity); ok {
				// 使用统一的渲染函数
				entities.RenderPlantCard(screen, card, pos.X, pos.Y, ras.sunFont, ras.sunFontSize)
			}
		}

		// 工具奖励：渲染工具图标（应用缩放，与植物卡包逻辑一致）
		if sprite, ok := ecs.GetComponent[*components.SpriteComponent](ras.entityManager, ras.rewardEntity); ok {
			if pos, ok := ecs.GetComponent[*components.PositionComponent](ras.entityManager, ras.rewardEntity); ok {
				if sprite.Image != nil {
					op := &ebiten.DrawImageOptions{}

					// 获取当前缩放值
					rewardComp, _ := ecs.GetComponent[*components.RewardAnimationComponent](ras.entityManager, ras.rewardEntity)
					scale := config.ShovelRewardStartScale // 默认缩放（铲子初始大小）
					if rewardComp != nil {
						scale = rewardComp.Scale // 使用动画缩放值
					}

					// 居中图片
					bounds := sprite.Image.Bounds()
					op.GeoM.Translate(-float64(bounds.Dx())/2, -float64(bounds.Dy())/2)

					// 应用缩放变换（与植物卡包一致）
					op.GeoM.Scale(scale, scale)

					// 移动到位置（屏幕坐标）
					op.GeoM.Translate(pos.X, pos.Y)

					screen.DrawImage(sprite.Image, op)
				}
			}
		}
	}

	// 2b. Phase 4-5: 渲染奖励面板（showing/closing）在最上层
	// closing 阶段也需要渲染面板（带淡出效果），否则游戏场景的菜单按钮会闪烁
	// 来信奖励使用 notePanelModule，植物/工具奖励使用 panelRenderSystem
	if ras.currentPhase == "showing" || ras.currentPhase == "closing" {
		if ras.notePanelModule != nil {
			// 来信奖励：渲染来信面板模块
			ras.notePanelModule.Draw(screen)
		} else if ras.panelEntity != 0 {
			// 植物/工具奖励：渲染传统奖励面板
			ras.panelRenderSystem.Draw(screen)
		}
	}

	// 2c. fadingOut/fadingIn 阶段：渲染来信面板和黑色遮罩
	if ras.currentPhase == "fadingOut" || ras.currentPhase == "fadingIn" {
		fadeAlpha := ras.gameState.NoteFadeAlpha

		if ras.currentPhase == "fadingOut" {
			// fadingOut 阶段：渲染半透明黑色遮罩（背景渐暗）
			if fadeAlpha > 0 {
				blackOverlay := ebiten.NewImage(screen.Bounds().Dx(), screen.Bounds().Dy())
				blackOverlay.Fill(color.RGBA{0, 0, 0, uint8(fadeAlpha * 255)})
				screen.DrawImage(blackOverlay, nil)
			}
		} else {
			// fadingIn 阶段：先绘制完全不透明的黑色背景，然后淡入面板
			// 这样可以完全覆盖游戏背景，直接从黑屏过渡到面板
			blackBg := ebiten.NewImage(screen.Bounds().Dx(), screen.Bounds().Dy())
			blackBg.Fill(color.RGBA{0, 0, 0, 255})
			screen.DrawImage(blackBg, nil)

			// 在黑色背景上渲染来信面板（透明度由 notePanelModule.SetFadeAlpha 控制）
			if ras.notePanelModule != nil {
				ras.notePanelModule.Draw(screen)
			}
		}
	}
}
