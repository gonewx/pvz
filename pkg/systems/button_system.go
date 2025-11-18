package systems

import (
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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
	// 获取鼠标位置
	mouseX, mouseY := ebiten.CursorPosition()
	mousePressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	mouseReleased := inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)

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
			if mousePressed {
				// 按下状态（显示按下效果）
				button.State = components.UIClicked
			} else if mouseReleased {
				// ✅ 释放时执行：释放瞬间触发回调
				if button.OnClick != nil {
					button.OnClick()
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

// isMouseInButton 检测鼠标是否在按钮范围内
func (s *ButtonSystem) isMouseInButton(mouseX, mouseY, buttonX, buttonY, buttonWidth, buttonHeight float64) bool {
	return mouseX >= buttonX &&
		mouseX <= buttonX+buttonWidth &&
		mouseY >= buttonY &&
		mouseY <= buttonY+buttonHeight
}
