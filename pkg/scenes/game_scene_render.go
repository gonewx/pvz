package scenes

import (
	"github.com/gonewx/pvz/pkg/config"
	"github.com/hajimehoshi/ebiten/v2"
)

// game_scene_render.go
// 渲染层级管理模块 - 将 Draw() 的复杂逻辑拆分为清晰的渲染层级函数
//
// 架构说明:
//   - 本文件包含所有渲染层级的绘制逻辑
//   - 按渲染顺序组织: 背景 → UI基础 → 游戏世界 → UI覆盖 → 顶层UI
//   - 每个层级函数负责一组相关的渲染内容
//   - Draw() 调用这些函数以保持清晰的渲染顺序

// RenderContext 渲染上下文，包含渲染时需要的状态信息
type RenderContext struct {
	hideUI              bool // 是否隐藏 UI（开场动画/铺草皮期间）
	hideMenuButton      bool // 是否隐藏菜单按钮
	hideShovel          bool // 是否隐藏铲子
	isSeedSelectionMode bool // 是否处于选卡模式
}

// calculateRenderContext 计算当前帧的渲染上下文
func (s *GameScene) calculateRenderContext() RenderContext {
	ctx := RenderContext{}

	// Story 8.3.1: 开场动画或铺草皮动画期间隐藏 UI 元素
	// 注意：需要检查 soddingAnimStarted 来避免开场动画完成和铺草皮动画开始之间的闪现
	isOpeningPlaying := s.openingSystem != nil && !s.openingSystem.IsCompleted()
	isSoddingPlaying := s.soddingSystem != nil && s.soddingSystem.IsPlaying()
	// 如果有开场动画系统且铺草皮动画还未启动，也要隐藏 UI（过渡期间）
	isWaitingForSodding := s.openingSystem != nil && s.openingSystem.IsCompleted() && !s.soddingAnimStarted
	ctx.hideUI = isOpeningPlaying || isSoddingPlaying || isWaitingForSodding

	// 暂停菜单按钮需要在所有开场动画完成后才显示
	// 包括：开场动画、铺草皮动画、除草车入场动画、ReadySetPlant 动画
	isLawnmowerEntering := s.lawnmowerSystem != nil && !s.lawnmowerSystem.AreAllEntered()
	isReadySetPlantPlaying := s.readySetPlantSystem != nil && s.readySetPlantSystem.IsPlaying()
	ctx.hideMenuButton = ctx.hideUI || isLawnmowerEntering || isReadySetPlantPlaying

	// Level 1-5 特殊处理：菜单按钮在传送带滑入完成后才显示
	// 不影响其他关卡的菜单显示逻辑
	isShovelTutorialLevel := s.gameState.CurrentLevel != nil && len(s.gameState.CurrentLevel.PresetPlants) > 0
	if isShovelTutorialLevel && !ctx.hideMenuButton {
		// 检查传送带滑入是否完成
		if s.levelPhaseSystem == nil || !s.levelPhaseSystem.IsConveyorBeltSlideComplete() {
			ctx.hideMenuButton = true
		}
	}

	// Story 19.x: 铲子槽的显示逻辑与菜单按钮分离
	// 对于有预设植物的关卡（Level 1-5），铲子槽在 Phase 1 已经显示
	// 传送带滑入后创建除草车时，铲子槽不应因除草车入场动画而消失
	// 只有菜单按钮需要等待除草车入场完成
	ctx.hideShovel = ctx.hideUI || isReadySetPlantPlaying
	// 非铲子教学关卡的普通流程：铲子槽与菜单按钮同时显示
	if !isShovelTutorialLevel {
		ctx.hideShovel = ctx.hideMenuButton
	}

	// 检查是否处于选卡模式
	ctx.isSeedSelectionMode = s.gameState.IsSeedSelectionActive()

	return ctx
}

// drawBackgroundLayer 绘制背景层（Layer 1）
func (s *GameScene) drawBackgroundLayer(screen *ebiten.Image) {
	// Layer 1: Draw lawn background
	s.drawBackground(screen)

	// Story 19.4: Draw bowling red line (保龄球红线)
	// 红线在背景之上、植物之下渲染
	s.drawBowlingRedLine(screen)
}

// drawUIBaseLayer 绘制 UI 基础层（Layer 2-3）
// 包括：SeedBank、Shovel、PlantCards
func (s *GameScene) drawUIBaseLayer(screen *ebiten.Image, ctx RenderContext) {
	if ctx.hideUI {
		return // 开场动画期间不渲染 UI
	}

	// Layer 2: Draw UI base elements (seed bank, shovel, plant cards)
	// 按照原版PVZ设计，UI元素在游戏世界实体下方渲染
	s.drawSeedBank(screen)

	// Story 19.5: 绘制传送带（在铲子和植物卡片之间）
	s.drawConveyorBelt(screen)

	// 铲子槽：对于铲子教学关卡，在除草车入场期间保持显示
	if !ctx.hideShovel {
		s.drawShovel(screen)
	}

	// 菜单按钮：需要等待所有开场动画完成后才显示（包括除草车入场和 ReadySetPlant）
	if !ctx.hideMenuButton {
		if s.buttonRenderSystem != nil {
			s.buttonRenderSystem.Draw(screen)
		}
	}

	// Layer 3: Draw plant cards (Story 3.1 架构优化)
	// 在植物和僵尸下方渲染，符合原版PVZ设计
	// Story 19.5: 保龄球模式不显示植物选择模块（使用传送带）
	// 滑入动画：计算 Y 偏移量，与植物选择栏同步滑入
	if s.plantSelectionModule != nil {
		if s.gameState.CurrentLevel == nil || s.gameState.CurrentLevel.InitialSun != 0 {
			// 计算滑入动画 Y 偏移
			yOffset := s.getSeedBankCurrentY() - float64(config.SeedBankY)
			s.plantSelectionModule.DrawWithOffset(screen, yOffset)
		}
	}
}

// drawGameWorldLayer 绘制游戏世界层（Layer 4-4.5）
// 包括：Plants、Zombies、Projectiles、LawnFlash、CardFlash
func (s *GameScene) drawGameWorldLayer(screen *ebiten.Image) {
	// Layer 4: Draw game world entities (plants, zombies, projectiles) - 不包括阳光
	// 游戏实体在UI卡片上方，这样植物和僵尸可以被看清
	// 传递 cameraX 以正确转换世界坐标到屏幕坐标
	s.renderSystem.DrawGameWorld(screen, s.cameraX)

	// 开场动画用户名显示（在游戏世界之后、其他UI之前）
	// 显示 "{username}的房子" 文本，白色带黑色阴影
	if s.openingSystem != nil {
		s.openingSystem.Draw(screen)
	}

	// Layer 4.5: Draw lawn flash effect (Story 8.2 教学)
	// 草坪闪烁效果，用于教学提示玩家可以种植
	s.drawLawnFlash(screen)

	// Story 8.2.1: Draw card flash effect (教学)
	// 卡片闪烁效果，用于提示玩家点击卡片
	s.drawCardFlash(screen)
}

// drawSeedChooserLayer 绘制选卡界面层
// Story 8.12: 选卡界面特殊渲染（在开场动画期间显示）
func (s *GameScene) drawSeedChooserLayer(screen *ebiten.Image, ctx RenderContext) {
	if !ctx.isSeedSelectionMode || s.seedChooserRenderSystem == nil {
		return
	}

	// 选卡阶段需要先渲染 SeedBank 背景和阳光计数器（与游戏中相同的顶部UI）
	// 注意：选卡阶段时滑入动画还没启动，所以需要直接渲染在正常位置，不能使用 drawSeedBank()

	// 直接渲染 SeedBank 背景（选卡界面中没有偏移，与面板 X 对齐）
	if s.seedBank != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(config.SeedChooserPanelX), float64(config.SeedBankY))
		screen.DrawImage(s.seedBank, op)
	}

	// 直接渲染阳光计数器（使用选卡界面的 X 偏移）
	s.drawSunCounterAtPosition(screen, float64(config.SeedChooserPanelX), float64(config.SeedBankY))

	// 渲染选卡界面（面板、仓库卡片、顶部卡槽、按钮）
	s.seedChooserRenderSystem.Draw(screen)

	// Story 8.12 QA: 选卡阶段也渲染菜单按钮
	if s.buttonRenderSystem != nil {
		s.buttonRenderSystem.Draw(screen)
	}
}

// drawUIOverlayLayer 绘制 UI 覆盖层（Layer 5-7.6）
// 包括：SunCounter、Particles、PlantPreview、Suns
func (s *GameScene) drawUIOverlayLayer(screen *ebiten.Image, ctx RenderContext) {
	// Layer 5: Draw UI overlays (sun counter text)
	// 文字始终在最上层以确保可读性
	// Story 8.3.1: 开场动画或铺草皮动画期间隐藏阳光计数器
	if !ctx.hideUI {
		s.drawSunCounter(screen)
	}

	// Layer 6: Draw particle effects (Story 7.3)
	// 只渲染游戏世界的粒子（爆炸、溅射等），过滤掉 UI 粒子
	// UI 粒子（如奖励动画）由各自的系统在更高层级渲染
	s.renderSystem.DrawGameWorldParticles(screen, s.cameraX)

	// Layer 7: Draw plant preview (Story 3.2)
	// 拖拽预览在所有内容上方
	s.plantPreviewRenderSystem.Draw(screen, s.cameraX)

	// Layer 7.5: Draw shovel interaction (Story 19.2)
	// 铲子光标和植物高亮效果
	if s.shovelInteractionSystem != nil && s.shovelSelected {
		s.shovelInteractionSystem.Draw(screen, s.cameraX)
	}

	// Layer 7.6: Draw conveyor card preview (Story 19.5)
	// 传送带卡片拖拽预览
	s.drawConveyorCardPreview(screen)

	// Layer 8: Draw suns (阳光) - 最顶层
	// 阳光在最顶层以确保始终可点击
	s.renderSystem.DrawSuns(screen, s.cameraX)
}

// drawTutorialAndWarningLayer 绘制教学和警告层（Layer 8.5-10.1）
// 包括：TutorialText、UIParticles、ProgressBar、HugeWaveWarning
func (s *GameScene) drawTutorialAndWarningLayer(screen *ebiten.Image, ctx RenderContext) {
	// Layer 8.5: Draw tutorial text (Story 8.2 + Story 19.x QA)
	// 教学文本在阳光之下、UI之上
	// 渲染所有 TutorialTextComponent 实体（包括临时提示文本）
	// 优先使用 bowlingTutorialFont，其次 tutorialFont，最后 sunCounterFont
	if s.bowlingTutorialFont != nil {
		s.renderSystem.DrawTutorialText(screen, s.bowlingTutorialFont, s.bowlingTutorialFont)
	} else if s.tutorialFont != nil {
		s.renderSystem.DrawTutorialText(screen, s.tutorialFont, s.bowlingTutorialFont)
	} else if s.sunCounterFont != nil {
		s.renderSystem.DrawTutorialText(screen, s.sunCounterFont, nil)
	}

	// Layer 8.6: Draw UI particles (教学箭头、奖励粒子等)
	// UI 粒子不受摄像机影响，渲染在教学文本之后
	s.renderSystem.DrawParticles(screen, 0)

	// Layer 9: Draw level progress bar (Story 5.5 - 正式版本)
	// 右下角图形化进度条
	// Story 8.3.1: 开场动画或铺草皮动画期间隐藏进度条
	if !ctx.hideUI {
		s.drawProgressBar(screen)
	}

	// Layer 10: Draw last wave warning (Story 5.5) - DISABLED for production
	// 最后一波提示（如果需要显示）（开发调试用，已禁用）
	// s.drawLastWaveWarning(screen) // 已禁用：改为使用 FinalWave.reanim 动画

	// Layer 10.1: Draw huge wave warning (Story 17.7)
	// 红字警告 "A Huge Wave of Zombies is Approaching!"
	s.drawHugeWaveWarning(screen)
}

// drawTopMostUILayer 绘制最顶层 UI（Layer 10.2-最后）
// 包括：RewardPanel、UIElements、Dialogs、Tooltip、PauseMenu
func (s *GameScene) drawTopMostUILayer(screen *ebiten.Image) {
	// Layer 10.5: Draw reward panel (Story 8.3 + 8.4)
	// Story 8.4重构：RewardAnimationSystem完全封装奖励面板渲染
	// 内部自动管理面板和植物卡片的渲染，调用者只需调用Draw方法
	s.rewardSystem.Draw(screen)

	// Story 8.3: 移除 "You Win" 覆盖层逻辑
	// 奖励流程完成后通过"下一关"按钮进入下一关，不再显示 You Win
	// Layer 11: Draw game result overlay (Story 5.5) - DISABLED for Story 8.3
	// s.drawGameResultOverlay(screen) // 已禁用：改为通过奖励面板的"下一关"按钮进入下一关

	// DEBUG: Draw particle test instructions (Story 7.4 debugging) - DISABLED
	// s.drawParticleTestInstructions(screen)

	// DEBUG: Draw grid boundaries and SodRoll debug lines (Story 3.3 debugging)
	s.drawGridDebug(screen)

	// Story 8.8: Draw UI elements (ZombiesWon animation, dialogs, etc.)
	// 渲染所有标记为 UIComponent 的实体（包括 Dave Reanim）
	// 手持物品（坚果）通过 InterlayerDrawRequests 在 Reanim 渲染过程中绘制
	s.renderSystem.DrawUIElements(screen)

	// Layer 10.2: Draw Dave dialogue bubble (Story 19.1)
	// 对话气泡在 Dave Reanim 之后绘制
	if s.daveDialogueSystem != nil {
		s.daveDialogueSystem.Draw(screen)
	}

	// Story 8.8: Draw dialog boxes (game over dialog, etc.)
	// 对话框在 UI 元素之后渲染，确保显示在最上层
	if s.dialogRenderSystem != nil {
		s.dialogRenderSystem.Draw(screen)
	}

	// Story 10.8: Draw Tooltip (植物卡片提示框)
	// Tooltip 在对话框之后、暂停菜单之前渲染
	s.drawTooltip(screen)

	// Story 10.1: Draw pause menu (最顶层 - 在所有其他元素之上)
	if s.pauseMenuModule != nil {
		s.pauseMenuModule.Draw(screen)
	}
}
