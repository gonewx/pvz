package systems

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// 按键高亮持续时间（秒）
const keyPressHighlightDuration = 0.1

// VirtualKeyboardSystem 虚拟键盘系统
// 处理虚拟键盘的触摸输入和字符输入逻辑
type VirtualKeyboardSystem struct {
	entityManager *ecs.EntityManager
}

// NewVirtualKeyboardSystem 创建虚拟键盘系统
func NewVirtualKeyboardSystem(em *ecs.EntityManager) *VirtualKeyboardSystem {
	return &VirtualKeyboardSystem{
		entityManager: em,
	}
}

// Update 更新虚拟键盘系统
func (s *VirtualKeyboardSystem) Update(deltaTime float64) {
	// 获取所有虚拟键盘实体
	keyboards := ecs.GetEntitiesWith1[*components.VirtualKeyboardComponent](s.entityManager)

	for _, kbEntity := range keyboards {
		kb, ok := ecs.GetComponent[*components.VirtualKeyboardComponent](s.entityManager, kbEntity)
		if !ok {
			continue
		}

		// 每帧开始时重置输入消费状态
		kb.InputConsumedThisFrame = false

		// 更新按键高亮计时器
		if kb.PressedKey != "" {
			kb.PressedTimer -= deltaTime
			if kb.PressedTimer <= 0 {
				kb.PressedKey = ""
				kb.PressedTimer = 0
			}
		}

		// 如果键盘不可见，检测是否点击了输入框以重新显示键盘
		if !kb.IsVisible {
			s.checkInputBoxClick(kb)
			continue
		}

		// 检测触摸/点击输入
		s.handleInput(kb)
	}
}

// checkInputBoxClick 检测是否点击了输入框，以重新显示键盘
func (s *VirtualKeyboardSystem) checkInputBoxClick(kb *components.VirtualKeyboardComponent) {
	// 如果没有目标输入框，跳过
	if kb.TargetInputEntity == 0 {
		return
	}

	// 获取目标输入框组件和位置
	inputComp, ok := ecs.GetComponent[*components.TextInputComponent](s.entityManager, kb.TargetInputEntity)
	if !ok {
		return
	}

	inputPos, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, kb.TargetInputEntity)
	if !ok {
		return
	}

	// 检测点击
	touchIDs := inpututil.AppendJustPressedTouchIDs(nil)
	var x, y int
	hasInput := false

	if len(touchIDs) > 0 {
		x, y = ebiten.TouchPosition(touchIDs[0])
		hasInput = true
	} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y = ebiten.CursorPosition()
		hasInput = true
	}

	if !hasInput {
		return
	}

	// 检查点击是否在输入框区域内
	fx, fy := float64(x), float64(y)
	if fx >= inputPos.X && fx <= inputPos.X+inputComp.Width &&
		fy >= inputPos.Y && fy <= inputPos.Y+inputComp.Height {
		// 点击了输入框，重新显示键盘并恢复焦点
		kb.IsVisible = true
		kb.InputConsumedThisFrame = true
		inputComp.IsFocused = true
		log.Printf("[VirtualKeyboardSystem] Input box clicked, reopening keyboard")
	}
}

// handleInput 处理触摸/点击输入
func (s *VirtualKeyboardSystem) handleInput(kb *components.VirtualKeyboardComponent) {
	// 获取刚按下的触摸点
	touchIDs := inpututil.AppendJustPressedTouchIDs(nil)
	var touchX, touchY int
	hasTouchInput := false

	if len(touchIDs) > 0 {
		// 使用第一个触摸点
		touchX, touchY = ebiten.TouchPosition(touchIDs[0])
		hasTouchInput = true
	}

	// 检测鼠标点击（兼容桌面端测试）
	hasMouseInput := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	var mouseX, mouseY int
	if hasMouseInput {
		mouseX, mouseY = ebiten.CursorPosition()
	}

	// 如果没有输入，返回
	if !hasTouchInput && !hasMouseInput {
		return
	}

	// 使用触摸位置或鼠标位置
	var x, y int
	if hasTouchInput {
		x, y = touchX, touchY
	} else {
		x, y = mouseX, mouseY
	}

	// 当虚拟键盘可见时，阻断所有点击事件传递到下层
	// 无论点击在键盘区域内还是外，都消费输入事件
	kb.InputConsumedThisFrame = true
	log.Printf("[VirtualKeyboardSystem] Input consumed at (%d, %d), keyboard visible", x, y)

	// 检查点击是否在键盘区域内
	if s.isPointInKeyboardArea(kb, float64(x), float64(y)) {
		// 检测点击的按键
		key := s.hitTestKey(kb, float64(x), float64(y))
		if key != nil {
			s.handleKeyPress(kb, key.Action)
			// 设置按键高亮
			kb.PressedKey = key.Action
			kb.PressedTimer = keyPressHighlightDuration
		}
	} else {
		// 点击在键盘区域外，关闭键盘
		log.Printf("[VirtualKeyboardSystem] Click outside keyboard area, closing keyboard")
		kb.IsVisible = false
	}
}

// hitTestKey 检测点击位置对应的按键
func (s *VirtualKeyboardSystem) hitTestKey(kb *components.VirtualKeyboardComponent, x, y float64) *components.KeyInfo {
	allKeys := entities.GetAllKeys(kb)
	for i := range allKeys {
		key := &allKeys[i]
		if x >= key.X && x <= key.X+key.Width &&
			y >= key.Y && y <= key.Y+key.Height {
			return key
		}
	}
	return nil
}

// isPointInKeyboardArea 检查点是否在键盘区域内
func (s *VirtualKeyboardSystem) isPointInKeyboardArea(kb *components.VirtualKeyboardComponent, x, y float64) bool {
	// 计算键盘区域（包括背景）
	keyboardTop := kb.KeyboardY - 10 // 上边距
	keyboardBottom := kb.KeyboardY + float64(len(components.KeyboardLayoutLower))*(kb.KeyHeight+kb.KeySpacing) + 10

	return y >= keyboardTop && y <= keyboardBottom && x >= 0 && x <= kb.ScreenWidth
}

// handleKeyPress 处理按键按下事件
func (s *VirtualKeyboardSystem) handleKeyPress(kb *components.VirtualKeyboardComponent, action string) {
	// 获取目标输入框组件
	if kb.TargetInputEntity == 0 {
		log.Printf("[VirtualKeyboardSystem] No target input entity")
		return
	}

	targetInput, ok := ecs.GetComponent[*components.TextInputComponent](s.entityManager, kb.TargetInputEntity)
	if !ok {
		log.Printf("[VirtualKeyboardSystem] Target input entity has no TextInputComponent")
		return
	}

	switch action {
	case "SHIFT":
		// 切换大小写模式
		kb.ShiftActive = !kb.ShiftActive
		log.Printf("[VirtualKeyboardSystem] Shift toggled: %v", kb.ShiftActive)

	case "BACKSPACE":
		// 删除光标前的字符
		s.deleteCharBefore(targetInput)

	case "SPACE":
		// 输入空格
		s.insertChar(targetInput, ' ')

	case "DONE":
		// 关闭键盘并确认输入
		kb.IsVisible = false
		// 触发失去焦点（如果需要）
		targetInput.IsFocused = false
		log.Printf("[VirtualKeyboardSystem] Keyboard closed, input confirmed: %s", targetInput.Text)

	case "123":
		// 切换到数字模式
		kb.NumericMode = true
		log.Printf("[VirtualKeyboardSystem] Switched to numeric mode")

	case "ABC":
		// 切换回字母模式
		kb.NumericMode = false
		log.Printf("[VirtualKeyboardSystem] Switched to ABC mode")

	default:
		// 字母/数字键
		if len(action) == 1 {
			s.insertChar(targetInput, rune(action[0]))
		}
	}
}

// insertChar 在光标位置插入字符
func (s *VirtualKeyboardSystem) insertChar(input *components.TextInputComponent, char rune) {
	// 过滤不符合用户名规则的字符（与 TextInputSystem 保持一致）
	if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == ' ') {
		log.Printf("[VirtualKeyboardSystem] Character '%c' not allowed for username", char)
		return
	}

	// 检查最大长度限制
	runes := []rune(input.Text)
	if input.MaxLength > 0 && len(runes) >= input.MaxLength {
		log.Printf("[VirtualKeyboardSystem] Max length reached (%d chars)", input.MaxLength)
		return
	}

	// 在光标位置插入字符
	before := runes[:input.CursorPosition]
	after := runes[input.CursorPosition:]

	result := append(before, char)
	result = append(result, after...)

	input.Text = string(result)
	input.CursorPosition++

	// 重置光标闪烁状态
	input.CursorBlinkTimer = 0
	input.CursorVisible = true

	log.Printf("[VirtualKeyboardSystem] Inserted char '%c', text now: %s", char, input.Text)
}

// deleteCharBefore 删除光标前的字符
func (s *VirtualKeyboardSystem) deleteCharBefore(input *components.TextInputComponent) {
	if input.CursorPosition == 0 {
		return // 光标在开头，无法删除
	}

	runes := []rune(input.Text)
	before := runes[:input.CursorPosition-1]
	after := runes[input.CursorPosition:]

	input.Text = string(append(before, after...))
	input.CursorPosition--

	// 重置光标闪烁状态
	input.CursorBlinkTimer = 0
	input.CursorVisible = true

	log.Printf("[VirtualKeyboardSystem] Deleted char, text now: %s", input.Text)
}

// ShowKeyboard 显示虚拟键盘并绑定到目标输入框
func (s *VirtualKeyboardSystem) ShowKeyboard(targetEntity ecs.EntityID) {
	keyboards := ecs.GetEntitiesWith1[*components.VirtualKeyboardComponent](s.entityManager)
	for _, kbEntity := range keyboards {
		kb, ok := ecs.GetComponent[*components.VirtualKeyboardComponent](s.entityManager, kbEntity)
		if !ok {
			continue
		}
		kb.IsVisible = true
		kb.TargetInputEntity = targetEntity
		kb.ShiftActive = false
		kb.NumericMode = false
		log.Printf("[VirtualKeyboardSystem] Keyboard shown for entity %d", targetEntity)
	}
}

// HideKeyboard 隐藏虚拟键盘
func (s *VirtualKeyboardSystem) HideKeyboard() {
	keyboards := ecs.GetEntitiesWith1[*components.VirtualKeyboardComponent](s.entityManager)
	for _, kbEntity := range keyboards {
		kb, ok := ecs.GetComponent[*components.VirtualKeyboardComponent](s.entityManager, kbEntity)
		if !ok {
			continue
		}
		kb.IsVisible = false
		kb.TargetInputEntity = 0
		log.Printf("[VirtualKeyboardSystem] Keyboard hidden")
	}
}

// IsKeyboardVisible 检查虚拟键盘是否可见
func (s *VirtualKeyboardSystem) IsKeyboardVisible() bool {
	keyboards := ecs.GetEntitiesWith1[*components.VirtualKeyboardComponent](s.entityManager)
	for _, kbEntity := range keyboards {
		kb, ok := ecs.GetComponent[*components.VirtualKeyboardComponent](s.entityManager, kbEntity)
		if ok && kb.IsVisible {
			return true
		}
	}
	return false
}

// ConsumeInput 检查本帧输入是否被虚拟键盘消费
// 如果返回 true，其他系统应该跳过处理本帧的点击事件
func (s *VirtualKeyboardSystem) ConsumeInput() bool {
	keyboards := ecs.GetEntitiesWith1[*components.VirtualKeyboardComponent](s.entityManager)
	for _, kbEntity := range keyboards {
		kb, ok := ecs.GetComponent[*components.VirtualKeyboardComponent](s.entityManager, kbEntity)
		if !ok {
			continue
		}
		if kb.InputConsumedThisFrame {
			return true
		}
	}
	return false
}
