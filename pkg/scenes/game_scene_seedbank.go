package scenes

import (
	"log"

	"github.com/gonewx/pvz/pkg/config"
)

// ============================================================================
// SeedBank 动画系统相关方法
// Story 15.5: 从 game_scene.go 拆分出 SeedBank 滑入动画逻辑（约100行）
// ============================================================================

// easeOutQuad 二次缓动函数（快速开始，缓慢结束）
// 用于植物选择栏滑入动画
func easeOutQuad(t float64) float64 {
	return 1.0 - (1.0-t)*(1.0-t)
}

// updateSeedBankSlideIn 更新植物选择栏滑入动画
// 在铺草皮动画完成后立即启动，与除草车入场动画同时播放
func (s *GameScene) updateSeedBankSlideIn(deltaTime float64) {
	// 如果动画已完成，不需要更新
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

// getSeedBankCurrentY 获取植物选择栏当前 Y 位置
// 根据滑入动画进度计算
func (s *GameScene) getSeedBankCurrentY() float64 {
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
