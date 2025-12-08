package systems

import (
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// CheckboxMouseInput 复选框系统鼠标输入接口
// 用于依赖注入，支持测试时 mock
type CheckboxMouseInput interface {
	CursorPosition() (int, int)
	IsMouseButtonJustReleased(button ebiten.MouseButton) bool
}

// ebitenCheckboxMouseInput Ebitengine 默认实现
type ebitenCheckboxMouseInput struct{}

func (e *ebitenCheckboxMouseInput) CursorPosition() (int, int) {
	return utils.GetPointerPosition()
}

func (e *ebitenCheckboxMouseInput) IsMouseButtonJustReleased(button ebiten.MouseButton) bool {
	// 使用支持触摸的释放检测
	released, _, _ := utils.IsPointerJustReleased()
	return released
}

// defaultCheckboxMouseInput 默认鼠标输入实例
var defaultCheckboxMouseInput CheckboxMouseInput = &ebitenCheckboxMouseInput{}

// CheckboxSystem 复选框交互系统
// 负责处理复选框的鼠标点击交互
//
// 职责：
//   - 检测鼠标是否在复选框图片区域内
//   - 检测鼠标左键 JustPressed 状态
//   - 切换 CheckboxComponent.IsChecked 状态
//   - 调用 OnToggle 回调
type CheckboxSystem struct {
	entityManager *ecs.EntityManager
	mouseInput    CheckboxMouseInput
}

// NewCheckboxSystem 创建复选框交互系统
func NewCheckboxSystem(em *ecs.EntityManager) *CheckboxSystem {
	return &CheckboxSystem{
		entityManager: em,
		mouseInput:    defaultCheckboxMouseInput,
	}
}

// NewCheckboxSystemWithInput 创建带自定义鼠标输入的复选框交互系统（用于测试）
func NewCheckboxSystemWithInput(em *ecs.EntityManager, input CheckboxMouseInput) *CheckboxSystem {
	return &CheckboxSystem{
		entityManager: em,
		mouseInput:    input,
	}
}

// Update 更新复选框交互状态
// 检测鼠标位置和点击，更新复选框状态
func (s *CheckboxSystem) Update(deltaTime float64) {
	// 获取鼠标位置
	mouseX, mouseY := s.mouseInput.CursorPosition()

	// 检测鼠标左键是否刚释放
	mouseJustReleased := s.mouseInput.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)

	// 查询所有复选框实体
	entities := ecs.GetEntitiesWith2[*components.CheckboxComponent, *components.PositionComponent](s.entityManager)

	for _, entityID := range entities {
		checkbox, _ := ecs.GetComponent[*components.CheckboxComponent](s.entityManager, entityID)
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)

		if checkbox == nil || pos == nil {
			continue
		}

		// 获取复选框尺寸
		var width, height float64
		if checkbox.IsChecked && checkbox.CheckedImage != nil {
			width = float64(checkbox.CheckedImage.Bounds().Dx())
			height = float64(checkbox.CheckedImage.Bounds().Dy())
		} else if checkbox.UncheckedImage != nil {
			width = float64(checkbox.UncheckedImage.Bounds().Dx())
			height = float64(checkbox.UncheckedImage.Bounds().Dy())
		} else {
			checkbox.IsHovered = false
			continue
		}

		// 检测鼠标是否在复选框区域内
		isInCheckbox := s.isMouseInCheckbox(float64(mouseX), float64(mouseY), pos.X, pos.Y, width, height)

		// 更新悬停状态
		checkbox.IsHovered = isInCheckbox

		// 释放时处理点击
		if mouseJustReleased && isInCheckbox {
			// 切换状态
			checkbox.IsChecked = !checkbox.IsChecked

			// 播放释放音效
			if checkbox.ClickSoundID != "" {
				if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
					audioManager.PlaySound(checkbox.ClickSoundID)
				}
			}

			// 触发回调
			if checkbox.OnToggle != nil {
				checkbox.OnToggle(checkbox.IsChecked)
			}
		}
	}
}

// isMouseInCheckbox 检测鼠标是否在复选框区域内
func (s *CheckboxSystem) isMouseInCheckbox(mouseX, mouseY, checkboxX, checkboxY, width, height float64) bool {
	return mouseX >= checkboxX &&
		mouseX <= checkboxX+width &&
		mouseY >= checkboxY &&
		mouseY <= checkboxY+height
}
