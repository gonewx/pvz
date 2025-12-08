package systems

import (
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/utils"
)

// ButtonSystem 按钮交互系统
// 负责处理按钮的鼠标悬停、点击等交互逻辑
//
// 职责：
//   - 检测鼠标悬停（更新按钮状态为 UIHovered）
//   - 检测鼠标点击（触发 OnClick 回调）
//   - 根据 Enabled 状态决定是否响应交互
//
// 注意：光标形状由调用者（如 MainMenuScene）统一管理
type ButtonSystem struct {
	entityManager *ecs.EntityManager
}

// NewButtonSystem 创建按钮交互系统
func NewButtonSystem(em *ecs.EntityManager) *ButtonSystem {
	return &ButtonSystem{
		entityManager: em,
	}
}

// Update 更新按钮交互状态
// 检测鼠标位置和释放，更新按钮状态并触发回调
func (s *ButtonSystem) Update(deltaTime float64) {
	// ✅ 检查虚拟键盘是否消费了本帧输入（阻止事件穿透）
	if s.isInputConsumedByKeyboard() {
		// 重置所有按钮状态为正常
		entities := ecs.GetEntitiesWith2[*components.ButtonComponent, *components.PositionComponent](s.entityManager)
		for _, entityID := range entities {
			button, _ := ecs.GetComponent[*components.ButtonComponent](s.entityManager, entityID)
			if button.Enabled {
				button.State = components.UINormal
			}
		}
		return
	}

	// 更新最后触摸位置
	utils.UpdateLastTouchPosition()

	// 获取鼠标位置
	mouseX, mouseY := utils.GetPointerPosition()
	mousePressed := utils.IsPointerPressed()
	// 使用支持触摸的按下/释放检测
	justPressed, pressX, pressY := utils.IsPointerJustPressed()
	justReleased, releaseX, releaseY := utils.IsPointerJustReleased()

	// 查询所有按钮实体
	entities := ecs.GetEntitiesWith2[*components.ButtonComponent, *components.PositionComponent](s.entityManager)

	for _, entityID := range entities {
		button, _ := ecs.GetComponent[*components.ButtonComponent](s.entityManager, entityID)
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)

		// 禁用状态不响应交互
		if !button.Enabled {
			button.State = components.UIDisabled
			continue
		}

		// 检测鼠标是否在按钮范围内
		isHovered := s.isMouseInButton(float64(mouseX), float64(mouseY), pos.X, pos.Y, button.Width, button.Height)

		if isHovered {
			// 鼠标在按钮内
			if justPressed {
				// 检查按下位置是否也在按钮内
				isPressInButton := s.isMouseInButton(float64(pressX), float64(pressY), pos.X, pos.Y, button.Width, button.Height)
				if isPressInButton {
					// 刚按下时播放按下音效（墓碑按钮专用）
					if button.PressedSoundID != "" {
						if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
							audioManager.PlaySound(button.PressedSoundID)
						}
					}
					button.State = components.UIClicked
				}
			} else if mousePressed {
				// 持续按下状态（显示按下效果）
				button.State = components.UIClicked
			} else if justReleased {
				// 检查释放位置是否也在按钮内
				isReleaseInButton := s.isMouseInButton(float64(releaseX), float64(releaseY), pos.X, pos.Y, button.Width, button.Height)
				if isReleaseInButton {
					// ✅ 释放时执行：释放瞬间触发回调和音效
					// 播放按钮释放音效
					if button.ClickSoundID != "" {
						if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
							audioManager.PlaySound(button.ClickSoundID)
						}
					}
					if button.OnClick != nil {
						button.OnClick()
					}
				}
				// 释放后恢复悬停状态
				button.State = components.UIHovered
			} else {
				// 悬停状态
				button.State = components.UIHovered
			}
		} else {
			// 鼠标不在按钮内，恢复正常状态
			button.State = components.UINormal
		}
	}

	// 注意：光标形状由调用者（如 MainMenuScene）统一管理，此处不再设置
}

// isInputConsumedByKeyboard 检查虚拟键盘是否可见或消费了本帧输入
// 当虚拟键盘可见时，阻断所有下层输入事件
func (s *ButtonSystem) isInputConsumedByKeyboard() bool {
	keyboards := ecs.GetEntitiesWith1[*components.VirtualKeyboardComponent](s.entityManager)
	for _, kbEntity := range keyboards {
		kb, ok := ecs.GetComponent[*components.VirtualKeyboardComponent](s.entityManager, kbEntity)
		if ok && (kb.IsVisible || kb.InputConsumedThisFrame) {
			return true
		}
	}
	return false
}

// isMouseInButton 检测鼠标是否在按钮范围内
func (s *ButtonSystem) isMouseInButton(mouseX, mouseY, buttonX, buttonY, buttonWidth, buttonHeight float64) bool {
	return mouseX >= buttonX &&
		mouseX <= buttonX+buttonWidth &&
		mouseY >= buttonY &&
		mouseY <= buttonY+buttonHeight
}
