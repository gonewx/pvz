package systems

import (
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// SliderMouseInput 滑块系统鼠标输入接口
// 用于依赖注入，支持测试时 mock
type SliderMouseInput interface {
	CursorPosition() (int, int)
	IsMouseButtonPressed(button ebiten.MouseButton) bool
}

// ebitenSliderMouseInput Ebitengine 默认实现
type ebitenSliderMouseInput struct{}

func (e *ebitenSliderMouseInput) CursorPosition() (int, int) {
	return utils.GetPointerPosition()
}

func (e *ebitenSliderMouseInput) IsMouseButtonPressed(button ebiten.MouseButton) bool {
	// 使用支持触摸的按下检测
	return utils.IsPointerPressed()
}

// defaultSliderMouseInput 默认鼠标输入实例
var defaultSliderMouseInput SliderMouseInput = &ebitenSliderMouseInput{}

// SliderSystem 滑块交互系统
// 负责处理滑块的鼠标拖拽交互
//
// 职责：
//   - 检测鼠标是否在滑槽区域内
//   - 检测鼠标左键按下/拖拽状态
//   - 计算点击位置并转换为 0.0~1.0 的 Value
//   - 更新 SliderComponent.Value 并调用 OnValueChange 回调
type SliderSystem struct {
	entityManager *ecs.EntityManager
	mouseInput    SliderMouseInput
}

// NewSliderSystem 创建滑块交互系统
func NewSliderSystem(em *ecs.EntityManager) *SliderSystem {
	return &SliderSystem{
		entityManager: em,
		mouseInput:    defaultSliderMouseInput,
	}
}

// NewSliderSystemWithInput 创建带自定义鼠标输入的滑块交互系统（用于测试）
func NewSliderSystemWithInput(em *ecs.EntityManager, input SliderMouseInput) *SliderSystem {
	return &SliderSystem{
		entityManager: em,
		mouseInput:    input,
	}
}

// Update 更新滑块交互状态
// 检测鼠标位置和按下状态，更新滑块值
func (s *SliderSystem) Update(deltaTime float64) {
	// 获取鼠标位置和按下状态（通过接口调用，支持 mock）
	mouseX, mouseY := s.mouseInput.CursorPosition()
	mousePressed := s.mouseInput.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	// 查询所有滑块实体
	entities := ecs.GetEntitiesWith2[*components.SliderComponent, *components.PositionComponent](s.entityManager)

	for _, entityID := range entities {
		slider, _ := ecs.GetComponent[*components.SliderComponent](s.entityManager, entityID)
		pos, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)

		if slider == nil || pos == nil {
			continue
		}

		// 计算滑槽边界
		slotX := pos.X
		slotY := pos.Y
		slotWidth := slider.SlotWidth
		slotHeight := slider.SlotHeight

		// 如果尺寸为0，尝试从图片获取
		if slotWidth == 0 && slider.SlotImage != nil {
			slotWidth = float64(slider.SlotImage.Bounds().Dx())
		}
		if slotHeight == 0 && slider.SlotImage != nil {
			slotHeight = float64(slider.SlotImage.Bounds().Dy())
		}

		// 检测鼠标是否在滑槽区域内
		isInSlot := s.isMouseInSlot(float64(mouseX), float64(mouseY), slotX, slotY, slotWidth, slotHeight)

		// 更新悬停状态
		slider.IsHovered = isInSlot

		// 记录拖拽前的状态，用于检测释放
		wasDragging := slider.IsDragging

		if mousePressed {
			// 如果鼠标按下且在滑槽内，或者正在拖拽
			if isInSlot || slider.IsDragging {
				slider.IsDragging = true

				// 计算新的值（0.0 ~ 1.0）
				newValue := s.calculateValue(float64(mouseX), slotX, slotWidth)

				// 限制在 0.0 ~ 1.0 范围内
				if newValue < 0.0 {
					newValue = 0.0
				}
				if newValue > 1.0 {
					newValue = 1.0
				}

				// 如果值发生变化，更新并触发回调
				if newValue != slider.Value {
					slider.Value = newValue
					if slider.OnValueChange != nil {
						slider.OnValueChange(newValue)
					}
				}
			}
		} else {
			// 鼠标释放，停止拖拽
			slider.IsDragging = false

			// 释放时播放音效（只在真正拖拽过后释放时播放）
			if wasDragging && slider.ClickSoundID != "" {
				if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
					audioManager.PlaySound(slider.ClickSoundID)
				}
			}
		}
	}
}

// isMouseInSlot 检测鼠标是否在滑槽区域内
func (s *SliderSystem) isMouseInSlot(mouseX, mouseY, slotX, slotY, slotWidth, slotHeight float64) bool {
	return mouseX >= slotX &&
		mouseX <= slotX+slotWidth &&
		mouseY >= slotY &&
		mouseY <= slotY+slotHeight
}

// calculateValue 根据鼠标X坐标计算滑块值
func (s *SliderSystem) calculateValue(mouseX, slotX, slotWidth float64) float64 {
	if slotWidth <= 0 {
		return 0.0
	}
	return (mouseX - slotX) / slotWidth
}
