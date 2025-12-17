package scenes

import (
	"log"

	"github.com/gonewx/pvz/pkg/config"
)

// ============================================================================
// SeedBank 动画系统相关方法
// Story 15.5: 从 game_scene.go 拆分出 SeedBank 滑入动画逻辑（约100行）
// Story 8.12: 添加选卡关卡水平滑动动画支持
// ============================================================================

// easeOutQuad 二次缓动函数（快速开始，缓慢结束）
// 用于植物选择栏滑入动画
func easeOutQuad(t float64) float64 {
	return 1.0 - (1.0-t)*(1.0-t)
}

// updateSeedBankSlideIn 更新植物选择栏滑入动画
// 在铺草皮动画完成后立即启动，与除草车入场动画同时播放
// Story 8.12: 选卡关卡使用水平滑动，非选卡关卡使用垂直滑入
func (s *GameScene) updateSeedBankSlideIn(deltaTime float64) {
	// Story 8.12: 选卡关卡特殊处理
	// 选卡关卡使用水平滑动动画，而不是垂直滑入动画
	isSeedSelectionLevel := s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.EnableSeedSelection
	seedSelectionConfirmed := isSeedSelectionLevel && s.gameState.IsSeedSelectionConfirmed()

	if seedSelectionConfirmed {
		// 选卡关卡：跳过垂直滑入，直接标记完成
		// 选卡关卡使用水平滑动（由 updateSeedBankHorizontalSlide 处理）
		if !s.seedBankSlideInCompleted {
			s.seedBankSlideInStarted = true
			s.seedBankSlideInProgress = 1.0
			s.seedBankSlideInCompleted = true
			log.Printf("[GameScene] 选卡关卡: 跳过垂直滑入，使用水平滑动")
		}
		return
	}

	// 非选卡关卡：原有的垂直滑入逻辑
	if s.seedBankSlideInCompleted {
		return
	}

	// 检查铺草皮动画是否完成（与除草车创建时机一致）
	isOpeningComplete := s.openingSystem == nil || s.openingSystem.IsCompleted()
	isSoddingComplete := s.soddingSystem == nil || !s.soddingSystem.IsPlaying()

	// 必须等待铺草皮动画启动后才能判断（避免过早启动）
	if !s.soddingAnimStarted {
		return
	}

	// 铺草皮完成后，与除草车入场动画同时启动滑入动画
	if isOpeningComplete && isSoddingComplete {
		if !s.seedBankSlideInStarted {
			s.seedBankSlideInStarted = true
			s.seedBankSlideInProgress = 0
			log.Printf("[GameScene] 植物选择栏滑入动画启动（与除草车入场同时）")
		}

		// 更新动画进度
		duration := config.SeedBankSlideInDuration
		s.seedBankSlideInProgress += deltaTime / duration

		if s.seedBankSlideInProgress >= 1.0 {
			s.seedBankSlideInProgress = 1.0
			s.seedBankSlideInCompleted = true
			log.Printf("[GameScene] 植物选择栏滑入动画完成")
		}
	}
}

// updateSeedBankHorizontalSlide 更新选卡关卡的水平滑动动画
// Story 8.12: 选卡确认后，植物选择栏从选卡位置（X=0）水平滑动到正常游戏位置（X=SeedBankX）
func (s *GameScene) updateSeedBankHorizontalSlide(deltaTime float64) {
	// 只有选卡关卡才需要水平滑动
	isSeedSelectionLevel := s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.EnableSeedSelection
	if !isSeedSelectionLevel {
		return
	}

	// 水平滑动已完成
	if s.seedBankHorizontalSlideCompleted {
		return
	}

	// 等待选卡确认
	seedSelectionConfirmed := s.gameState.IsSeedSelectionConfirmed()
	if !seedSelectionConfirmed {
		return
	}

	// 启动水平滑动动画
	if !s.seedBankHorizontalSlideStarted {
		s.seedBankHorizontalSlideStarted = true
		s.seedBankHorizontalSlideProgress = 0
		log.Printf("[GameScene] 植物选择栏水平滑动动画启动（选卡确认后）")
	}

	// 更新动画进度
	duration := config.SeedBankHorizontalSlideDuration
	s.seedBankHorizontalSlideProgress += deltaTime / duration

	if s.seedBankHorizontalSlideProgress >= 1.0 {
		s.seedBankHorizontalSlideProgress = 1.0
		s.seedBankHorizontalSlideCompleted = true
		log.Printf("[GameScene] 植物选择栏水平滑动动画完成")
	}
}

// getSeedBankCurrentY 获取植物选择栏当前 Y 位置
// 根据滑入动画进度计算
// Story 8.12: 选卡关卡 Y 位置固定为 SeedBankY
func (s *GameScene) getSeedBankCurrentY() float64 {
	// Story 8.12: 选卡关卡使用水平滑动，Y 位置固定
	// 修复：检查选卡确认或开场动画完成（因为 CompleteSeedSelection 会重置 Phase 为 None）
	isSeedSelectionLevel := s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.EnableSeedSelection
	openingCompleted := s.openingSystem == nil || s.openingSystem.IsCompleted()
	seedSelectionConfirmed := s.gameState.IsSeedSelectionConfirmed()

	// 选卡关卡一旦选卡确认或开场动画完成，Y 位置固定
	if isSeedSelectionLevel && (seedSelectionConfirmed || openingCompleted) {
		return float64(config.SeedBankY)
	}

	// 非选卡关卡：原有的垂直滑入逻辑
	// 如果动画已完成，返回最终位置
	if s.seedBankSlideInCompleted {
		return float64(config.SeedBankY)
	}

	// 如果动画未启动，返回起始位置（屏幕外，不显示）
	if !s.seedBankSlideInStarted {
		return config.SeedBankSlideInStartY
	}

	// 使用 ease-out 缓动计算 Y 位置
	progress := easeOutQuad(s.seedBankSlideInProgress)
	startY := config.SeedBankSlideInStartY
	targetY := float64(config.SeedBankY)
	return startY + (targetY-startY)*progress
}

// getSeedBankCurrentX 获取植物选择栏当前 X 位置
// Story 8.12: 选卡关卡使用水平滑动动画，非选卡关卡固定为 SeedBankX
func (s *GameScene) getSeedBankCurrentX() float64 {
	// 非选卡关卡：X 位置固定
	isSeedSelectionLevel := s.gameState.CurrentLevel != nil && s.gameState.CurrentLevel.EnableSeedSelection
	if !isSeedSelectionLevel {
		return float64(config.SeedBankX)
	}

	// Story 8.12 修复：检查开场动画是否完成
	// 当开场动画完成时，CompleteSeedSelection() 会将 Phase 重置为 None
	// 此时应该使用最终位置或水平滑动动画位置
	openingCompleted := s.openingSystem == nil || s.openingSystem.IsCompleted()
	seedSelectionConfirmed := s.gameState.IsSeedSelectionConfirmed()

	// 选卡阶段（开场动画未完成且选卡未确认）：X 位置固定在选卡面板位置
	if !seedSelectionConfirmed && !openingCompleted {
		return float64(config.SeedChooserPanelX)
	}

	// 选卡确认后或开场动画完成后：使用水平滑动动画位置
	if s.seedBankHorizontalSlideCompleted {
		return float64(config.SeedBankX)
	}

	if !s.seedBankHorizontalSlideStarted {
		return config.SeedBankHorizontalSlideStartX
	}

	// 使用 ease-out 缓动计算 X 位置
	progress := easeOutQuad(s.seedBankHorizontalSlideProgress)
	startX := config.SeedBankHorizontalSlideStartX
	targetX := float64(config.SeedBankX)
	return startX + (targetX-startX)*progress
}
