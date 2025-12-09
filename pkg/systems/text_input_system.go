package systems

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// TextInputSystem 文本输入系统
// 处理文本输入框的键盘输入、光标闪烁等逻辑
type TextInputSystem struct {
	entityManager *ecs.EntityManager
}

// NewTextInputSystem 创建文本输入系统
func NewTextInputSystem(em *ecs.EntityManager) *TextInputSystem {
	return &TextInputSystem{
		entityManager: em,
	}
}

// Update 更新文本输入系统
func (s *TextInputSystem) Update(deltaTime float64) {
	// 获取所有文本输入实体
	entities := ecs.GetEntitiesWith1[*components.TextInputComponent](s.entityManager)

	for _, entityID := range entities {
		input, ok := ecs.GetComponent[*components.TextInputComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 只处理获得焦点的输入框
		if !input.IsFocused {
			input.CursorVisible = false
			continue
		}

		// 更新光标闪烁
		s.updateCursorBlink(input, deltaTime)

		// 移动端：跳过物理键盘输入，由 VirtualKeyboardSystem 处理
		if utils.IsMobile() {
			continue
		}

		// 桌面端：处理键盘输入
		s.handleKeyboardInput(input)
	}
}

// updateCursorBlink 更新光标闪烁状态
func (s *TextInputSystem) updateCursorBlink(input *components.TextInputComponent, deltaTime float64) {
	const blinkInterval = 0.5 // 光标闪烁间隔（秒）

	input.CursorBlinkTimer += deltaTime
	if input.CursorBlinkTimer >= blinkInterval {
		input.CursorBlinkTimer = 0
		input.CursorVisible = !input.CursorVisible
	}
}

// handleKeyboardInput 处理键盘输入
func (s *TextInputSystem) handleKeyboardInput(input *components.TextInputComponent) {
	// 1. 处理文本字符输入（使用 AppendInputChars）
	runes := ebiten.AppendInputChars(nil)
	if len(runes) > 0 {
		s.insertText(input, string(runes))
		// 重置光标闪烁（输入时光标应该可见）
		input.CursorBlinkTimer = 0
		input.CursorVisible = true
	}

	// 2. 处理退格键（删除光标前的字符）
	// 使用 KeyPressDuration 支持按住连续删除
	backspaceDuration := inpututil.KeyPressDuration(ebiten.KeyBackspace)
	if backspaceDuration == 1 || (backspaceDuration >= 30 && backspaceDuration%3 == 0) {
		// 第1帧立即响应，之后每隔3帧响应一次（实现连续删除）
		s.deleteCharBefore(input)
		input.CursorBlinkTimer = 0
		input.CursorVisible = true
	}

	// 3. 处理删除键（删除光标后的字符）
	// 使用 KeyPressDuration 支持按住连续删除
	deleteDuration := inpututil.KeyPressDuration(ebiten.KeyDelete)
	if deleteDuration == 1 || (deleteDuration >= 30 && deleteDuration%3 == 0) {
		s.deleteCharAfter(input)
		input.CursorBlinkTimer = 0
		input.CursorVisible = true
	}

	// 4. 处理左箭头键（光标左移）
	// 使用 KeyPressDuration 支持按住连续移动
	leftDuration := inpututil.KeyPressDuration(ebiten.KeyArrowLeft)
	if leftDuration == 1 || (leftDuration >= 30 && leftDuration%3 == 0) {
		s.moveCursorLeft(input)
		input.CursorBlinkTimer = 0
		input.CursorVisible = true
	}

	// 5. 处理右箭头键（光标右移）
	// 使用 KeyPressDuration 支持按住连续移动
	rightDuration := inpututil.KeyPressDuration(ebiten.KeyArrowRight)
	if rightDuration == 1 || (rightDuration >= 30 && rightDuration%3 == 0) {
		s.moveCursorRight(input)
		input.CursorBlinkTimer = 0
		input.CursorVisible = true
	}

	// 6. 处理 Home 键（光标移到开头）
	if inpututil.IsKeyJustPressed(ebiten.KeyHome) {
		input.CursorPosition = 0
		input.CursorBlinkTimer = 0
		input.CursorVisible = true
	}

	// 7. 处理 End 键（光标移到结尾）
	if inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
		input.CursorPosition = len([]rune(input.Text))
		input.CursorBlinkTimer = 0
		input.CursorVisible = true
	}
}

// insertText 在光标位置插入文本
func (s *TextInputSystem) insertText(input *components.TextInputComponent, text string) {
	// ✅ Story 12.4: 过滤不符合用户名规则的字符
	// 用户名只能包含字母、数字、空格
	filteredText := ""
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' {
			filteredText += string(r)
		}
	}

	if filteredText == "" {
		return
	}

	// 检查最大长度限制
	runes := []rune(input.Text)
	if input.MaxLength > 0 && len(runes)+len([]rune(filteredText)) > input.MaxLength {
		log.Printf("[TextInputSystem] 达到最大长度限制 (%d 字符)", input.MaxLength)
		return
	}

	// 在光标位置插入文本
	textRunes := []rune(input.Text)
	newRunes := []rune(filteredText)

	// 分割为光标前和光标后
	before := textRunes[:input.CursorPosition]
	after := textRunes[input.CursorPosition:]

	// 合并
	result := append(before, newRunes...)
	result = append(result, after...)

	input.Text = string(result)
	input.CursorPosition += len(newRunes)
}

// deleteCharBefore 删除光标前的字符（退格）
func (s *TextInputSystem) deleteCharBefore(input *components.TextInputComponent) {
	if input.CursorPosition == 0 {
		return // 光标在开头，无法删除
	}

	runes := []rune(input.Text)
	before := runes[:input.CursorPosition-1]
	after := runes[input.CursorPosition:]

	input.Text = string(append(before, after...))
	input.CursorPosition--
}

// deleteCharAfter 删除光标后的字符（Delete键）
func (s *TextInputSystem) deleteCharAfter(input *components.TextInputComponent) {
	runes := []rune(input.Text)
	if input.CursorPosition >= len(runes) {
		return // 光标在结尾，无法删除
	}

	before := runes[:input.CursorPosition]
	after := runes[input.CursorPosition+1:]

	input.Text = string(append(before, after...))
	// 光标位置不变
}

// moveCursorLeft 光标左移
func (s *TextInputSystem) moveCursorLeft(input *components.TextInputComponent) {
	if input.CursorPosition > 0 {
		input.CursorPosition--
	}
}

// moveCursorRight 光标右移
func (s *TextInputSystem) moveCursorRight(input *components.TextInputComponent) {
	runes := []rune(input.Text)
	if input.CursorPosition < len(runes) {
		input.CursorPosition++
	}
}
