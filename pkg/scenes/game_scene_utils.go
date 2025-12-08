package scenes

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// Config constants for intro animation
const (
	IntroAnimDuration = config.IntroAnimDuration
	GameCameraX       = config.GameCameraX
)

// updateIntroAnimation updates the intro camera animation that showcases the entire lawn.
// The animation has two phases:
//   - Phase 1 (0.0-0.5): Camera scrolls from left edge (0) to right edge (maxCameraX)
//   - Phase 2 (0.5-1.0): Camera scrolls back from right edge to gameplay position (GameCameraX)
//
// Both phases use an ease-out quadratic easing function for smooth motion.
func (s *GameScene) updateIntroAnimation(deltaTime float64) {
	s.introAnimTimer += deltaTime
	progress := s.introAnimTimer / IntroAnimDuration

	if progress >= 1.0 {
		// Animation complete, camera settled at gameplay position
		s.cameraX = GameCameraX
		s.isIntroAnimPlaying = false
		return
	}

	if progress < 0.5 {
		// Phase 1: Scroll from left (0) to right (maxCameraX)
		phaseProgress := progress / 0.5
		easedProgress := s.easeOutQuad(phaseProgress)
		s.cameraX = easedProgress * s.maxCameraX
	} else {
		// Phase 2: Scroll from right (maxCameraX) back to center (GameCameraX)
		phaseProgress := (progress - 0.5) / 0.5
		easedProgress := s.easeOutQuad(phaseProgress)
		s.cameraX = s.maxCameraX + easedProgress*(GameCameraX-s.maxCameraX)
	}
}

// easeOutQuad applies an ease-out quadratic easing function to the input value.
// Formula: 1 - (1-t)^2
// This creates a smooth deceleration effect.
//
// Parameters:
//   - t: Input value in range [0, 1]
//
// Returns:
//   - Eased value in range [0, 1]
func (s *GameScene) easeOutQuad(t float64) float64 {
	return 1 - (1-t)*(1-t)
}

// isDaveDialogueActive 检测 Dave 对话是否正在进行
// 对话期间（Entering/Talking/Leaving 状态）返回 true
// 用于在对话期间屏蔽其他交互（如铲子槽位点击）
func (s *GameScene) isDaveDialogueActive() bool {
	daveEntities := ecs.GetEntitiesWith1[*components.DaveDialogueComponent](s.entityManager)
	for _, entityID := range daveEntities {
		daveComp, ok := ecs.GetComponent[*components.DaveDialogueComponent](s.entityManager, entityID)
		if ok && (daveComp.State == components.DaveStateTalking ||
			daveComp.State == components.DaveStateEntering ||
			daveComp.State == components.DaveStateLeaving) {
			return true
		}
	}
	return false
}

// updateMouseCursor 根据组件状态更新鼠标光标形状
//
// ECS 架构原则:
//   - 只读取组件状态,不进行碰撞检测
//   - DialogInputSystem 负责更新 DialogComponent.HoveredButtonIdx
//   - ButtonSystem 负责更新 ButtonComponent.State
//   - InputSystem 负责更新 PlantCardComponent.UIComponent.State
//   - InputSystem 负责更新 SunComponent 的 ClickableComponent.IsHovered
//   - 这里只根据状态设置光标
//
// 检查优先级:
//  0. 奖励动画 (RewardAnimationSystem - 最高优先级，遮盖其他元素)
//  0.5. Dave 对话 (DaveDialogueComponent - 对话期间强制默认光标)
//  1. 面板按钮 (ButtonComponent)
//  2. 对话框按钮 (DialogComponent.HoveredButtonIdx)
//  3. 植物卡片 (PlantCardComponent + UIComponent)
//  4. 阳光 (SunComponent + ClickableComponent.IsHovered)
func (s *GameScene) updateMouseCursor() {
	// Default cursor shape
	cursorShape := ebiten.CursorShapeDefault

	// 0. 奖励动画系统悬停检测（最高优先级，因为奖励面板会遮盖其他元素）
	if s.rewardSystem != nil && s.rewardSystem.IsActive() {
		rewardCursor := s.rewardSystem.GetCursorShape()
		if rewardCursor != ebiten.CursorShapeDefault {
			cursorShape = rewardCursor
		}
	}

	// 0.5. Story 19.x QA: Dave 对话期间强制使用默认光标
	// Dave 对话时不检测其他元素的悬停状态，直接使用默认光标
	if s.isDaveDialogueActive() {
		// Dave 对话期间，跳过其他检测，直接设置默认光标
		if cursorShape != s.lastCursorShape {
			ebiten.SetCursorShape(ebiten.CursorShapeDefault)
			s.lastCursorShape = ebiten.CursorShapeDefault
		}
		return
	}

	// 1. Check if hovering over any panel button (pause menu, settings panel, menu button)
	if cursorShape == ebiten.CursorShapeDefault {
		panelButtons := ecs.GetEntitiesWith1[*components.ButtonComponent](s.entityManager)
		for _, entityID := range panelButtons {
			button, ok := ecs.GetComponent[*components.ButtonComponent](s.entityManager, entityID)
			if ok && button.State == components.UIHovered {
				cursorShape = ebiten.CursorShapePointer
				break
			}
		}
	}

	// 1.5. Check if hovering over any slider (settings panel)
	if cursorShape == ebiten.CursorShapeDefault {
		sliders := ecs.GetEntitiesWith1[*components.SliderComponent](s.entityManager)
		for _, entityID := range sliders {
			slider, ok := ecs.GetComponent[*components.SliderComponent](s.entityManager, entityID)
			if ok && slider.IsHovered {
				cursorShape = ebiten.CursorShapePointer
				break
			}
		}
	}

	// 1.6. Check if hovering over any checkbox (settings panel)
	if cursorShape == ebiten.CursorShapeDefault {
		checkboxes := ecs.GetEntitiesWith1[*components.CheckboxComponent](s.entityManager)
		for _, entityID := range checkboxes {
			checkbox, ok := ecs.GetComponent[*components.CheckboxComponent](s.entityManager, entityID)
			if ok && checkbox.IsHovered {
				cursorShape = ebiten.CursorShapePointer
				break
			}
		}
	}

	// 2. Check if hovering over any dialog button (if there are any dialogs)
	if cursorShape == ebiten.CursorShapeDefault {
		dialogEntities := ecs.GetEntitiesWith1[*components.DialogComponent](s.entityManager)
		for _, entityID := range dialogEntities {
			dialogComp, ok := ecs.GetComponent[*components.DialogComponent](s.entityManager, entityID)
			if ok && dialogComp.IsVisible {
				// 检查对话框按钮是否悬停（只读取状态）
				if dialogComp.HoveredButtonIdx >= 0 {
					cursorShape = ebiten.CursorShapePointer
					break
				}

				// 检查用户列表是否有悬停项（只读取状态）
				if userList, ok2 := ecs.GetComponent[*components.UserListComponent](s.entityManager, entityID); ok2 {
					if userList.HoveredIndex >= 0 {
						cursorShape = ebiten.CursorShapePointer
						break
					}
				}
			}
		}
	}

	// 3. Story 8.2.1: Check if hovering over any plant card
	if cursorShape == ebiten.CursorShapeDefault {
		plantCards := ecs.GetEntitiesWith2[
			*components.PlantCardComponent,
			*components.UIComponent,
		](s.entityManager)
		for _, entityID := range plantCards {
			// 跳过奖励卡片
			if _, isRewardCard := ecs.GetComponent[*components.RewardCardComponent](s.entityManager, entityID); isRewardCard {
				continue
			}

			ui, ok := ecs.GetComponent[*components.UIComponent](s.entityManager, entityID)
			if ok && ui.State == components.UIHovered {
				cursorShape = ebiten.CursorShapePointer
				break
			}
		}
	}

	// 4. Check if hovering over any clickable sun
	if cursorShape == ebiten.CursorShapeDefault {
		sunEntities := ecs.GetEntitiesWith2[
			*components.SunComponent,
			*components.ClickableComponent,
		](s.entityManager)
		for _, entityID := range sunEntities {
			clickable, ok := ecs.GetComponent[*components.ClickableComponent](s.entityManager, entityID)
			if ok && clickable.IsHovered && clickable.IsEnabled {
				cursorShape = ebiten.CursorShapePointer
				break
			}
		}
	}

	// 5. Story 19.x QA: Check if hovering over shovel slot
	// 铲子槽位悬停时显示手形光标
	if cursorShape == ebiten.CursorShapeDefault && !s.shovelSelected {
		mouseX, mouseY := utils.GetPointerPosition()
		bounds := s.GetShovelSlotBounds()
		if mouseX >= bounds.Min.X && mouseX <= bounds.Max.X &&
			mouseY >= bounds.Min.Y && mouseY <= bounds.Max.Y {
			// 检查铲子是否可用（已解锁或铲子教学关卡）
			isShovelTutorialLevel := s.gameState.CurrentLevel != nil && len(s.gameState.CurrentLevel.PresetPlants) > 0
			if s.gameState.IsToolUnlocked("shovel") || isShovelTutorialLevel {
				cursorShape = ebiten.CursorShapePointer
			}
		}
	}

	// 6. Check if hovering over conveyor belt cards
	// 传送带卡片悬停时显示手形光标
	if cursorShape == ebiten.CursorShapeDefault {
		if s.isMouseOverConveyorCard() {
			cursorShape = ebiten.CursorShapePointer
		}
	}

	// Only update cursor if shape changed (避免闪烁)
	if cursorShape != s.lastCursorShape {
		ebiten.SetCursorShape(cursorShape)
		s.lastCursorShape = cursorShape
	}
}

// retryLevel 重新尝试当前关卡
// Story 8.8: 重新加载关卡，重置所有状态
func (s *GameScene) retryLevel() {
	log.Printf("[GameScene] 重新尝试关卡")

	// 获取当前关卡ID
	currentLevelID := "1-1" // 默认值
	if s.gameState.CurrentLevel != nil {
		currentLevelID = s.gameState.CurrentLevel.ID
	}

	// 重新加载场景
	s.sceneManager.SwitchTo(NewGameScene(s.resourceManager, s.sceneManager, currentLevelID))
}
