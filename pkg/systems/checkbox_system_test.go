package systems

import (
	"image"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// mockCheckboxMouseInput 用于测试的 mock 鼠标输入
type mockCheckboxMouseInput struct {
	mouseX      int
	mouseY      int
	justPressed bool
}

func (m *mockCheckboxMouseInput) CursorPosition() (int, int) {
	return m.mouseX, m.mouseY
}

func (m *mockCheckboxMouseInput) IsMouseButtonJustPressed(button ebiten.MouseButton) bool {
	return m.justPressed
}

// createTestImage 创建测试用图片
func createTestImage(width, height int) *ebiten.Image {
	return ebiten.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, width, height)))
}

// TestCheckboxSystem_isMouseInCheckbox 测试鼠标在复选框内检测
func TestCheckboxSystem_isMouseInCheckbox(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewCheckboxSystem(em)

	tests := []struct {
		name     string
		mouseX   float64
		mouseY   float64
		boxX     float64
		boxY     float64
		width    float64
		height   float64
		expected bool
	}{
		{
			name:     "在复选框内",
			mouseX:   115,
			mouseY:   65,
			boxX:     100,
			boxY:     50,
			width:    30,
			height:   30,
			expected: true,
		},
		{
			name:     "左边界外",
			mouseX:   99,
			mouseY:   65,
			boxX:     100,
			boxY:     50,
			width:    30,
			height:   30,
			expected: false,
		},
		{
			name:     "右边界外",
			mouseX:   131,
			mouseY:   65,
			boxX:     100,
			boxY:     50,
			width:    30,
			height:   30,
			expected: false,
		},
		{
			name:     "上边界外",
			mouseX:   115,
			mouseY:   49,
			boxX:     100,
			boxY:     50,
			width:    30,
			height:   30,
			expected: false,
		},
		{
			name:     "下边界外",
			mouseX:   115,
			mouseY:   81,
			boxX:     100,
			boxY:     50,
			width:    30,
			height:   30,
			expected: false,
		},
		{
			name:     "左上角",
			mouseX:   100,
			mouseY:   50,
			boxX:     100,
			boxY:     50,
			width:    30,
			height:   30,
			expected: true,
		},
		{
			name:     "右下角",
			mouseX:   130,
			mouseY:   80,
			boxX:     100,
			boxY:     50,
			width:    30,
			height:   30,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := system.isMouseInCheckbox(tt.mouseX, tt.mouseY, tt.boxX, tt.boxY, tt.width, tt.height)
			if result != tt.expected {
				t.Errorf("isMouseInCheckbox() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestNewCheckboxSystem 测试复选框系统创建
func TestNewCheckboxSystem(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewCheckboxSystem(em)

	if system == nil {
		t.Fatal("NewCheckboxSystem() returned nil")
	}
	if system.entityManager != em {
		t.Error("entityManager not set correctly")
	}
}

// TestCheckboxSystem_EntityWithCheckboxComponent 测试创建带复选框组件的实体
func TestCheckboxSystem_EntityWithCheckboxComponent(t *testing.T) {
	em := ecs.NewEntityManager()

	// 创建复选框实体
	entity := em.CreateEntity()
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: 100,
		Y: 50,
	})

	var toggleCount int
	var lastChecked bool
	ecs.AddComponent(em, entity, &components.CheckboxComponent{
		IsChecked: false,
		Label:     "Test Checkbox",
		OnToggle: func(isChecked bool) {
			toggleCount++
			lastChecked = isChecked
		},
	})

	// 验证组件正确添加
	checkbox, ok := ecs.GetComponent[*components.CheckboxComponent](em, entity)
	if !ok {
		t.Fatal("CheckboxComponent not found")
	}
	if checkbox.IsChecked {
		t.Error("Initial IsChecked should be false")
	}
	if checkbox.Label != "Test Checkbox" {
		t.Errorf("Label = %v, want 'Test Checkbox'", checkbox.Label)
	}

	// 模拟切换
	checkbox.IsChecked = !checkbox.IsChecked
	if checkbox.OnToggle != nil {
		checkbox.OnToggle(checkbox.IsChecked)
	}
	if toggleCount != 1 {
		t.Errorf("Toggle count = %v, want 1", toggleCount)
	}
	if !lastChecked {
		t.Error("Last checked should be true after toggle")
	}

	// 再次切换
	checkbox.IsChecked = !checkbox.IsChecked
	if checkbox.OnToggle != nil {
		checkbox.OnToggle(checkbox.IsChecked)
	}
	if toggleCount != 2 {
		t.Errorf("Toggle count = %v, want 2", toggleCount)
	}
	if lastChecked {
		t.Error("Last checked should be false after second toggle")
	}
}

// TestCheckboxSystem_Update_ClickInCheckbox 测试点击复选框内切换状态
func TestCheckboxSystem_Update_ClickInCheckbox(t *testing.T) {
	em := ecs.NewEntityManager()
	mockInput := &mockCheckboxMouseInput{
		mouseX:      115, // 在复选框内
		mouseY:      65,
		justPressed: true,
	}
	system := NewCheckboxSystemWithInput(em, mockInput)

	// 创建复选框实体
	entity := em.CreateEntity()
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: 100,
		Y: 50,
	})

	var toggleCalled bool
	var toggleValue bool
	ecs.AddComponent(em, entity, &components.CheckboxComponent{
		IsChecked:      false,
		UncheckedImage: createTestImage(30, 30),
		CheckedImage:   createTestImage(30, 30),
		OnToggle: func(isChecked bool) {
			toggleCalled = true
			toggleValue = isChecked
		},
	})

	// 执行 Update
	system.Update(0.016)

	// 验证状态已切换
	checkbox, _ := ecs.GetComponent[*components.CheckboxComponent](em, entity)
	if !checkbox.IsChecked {
		t.Error("Checkbox should be checked after click")
	}
	if !toggleCalled {
		t.Error("OnToggle callback should be called")
	}
	if !toggleValue {
		t.Error("Toggle value should be true")
	}
}

// TestCheckboxSystem_Update_ClickOutsideCheckbox 测试点击复选框外不切换状态
func TestCheckboxSystem_Update_ClickOutsideCheckbox(t *testing.T) {
	em := ecs.NewEntityManager()
	mockInput := &mockCheckboxMouseInput{
		mouseX:      50, // 复选框外
		mouseY:      65,
		justPressed: true,
	}
	system := NewCheckboxSystemWithInput(em, mockInput)

	// 创建复选框实体
	entity := em.CreateEntity()
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: 100,
		Y: 50,
	})

	var toggleCalled bool
	ecs.AddComponent(em, entity, &components.CheckboxComponent{
		IsChecked:      false,
		UncheckedImage: createTestImage(30, 30),
		CheckedImage:   createTestImage(30, 30),
		OnToggle: func(isChecked bool) {
			toggleCalled = true
		},
	})

	// 执行 Update
	system.Update(0.016)

	// 验证状态未切换
	checkbox, _ := ecs.GetComponent[*components.CheckboxComponent](em, entity)
	if checkbox.IsChecked {
		t.Error("Checkbox should remain unchecked")
	}
	if toggleCalled {
		t.Error("OnToggle callback should not be called")
	}
}

// TestCheckboxSystem_Update_MouseNotJustPressed 测试鼠标未刚按下时不切换状态
func TestCheckboxSystem_Update_MouseNotJustPressed(t *testing.T) {
	em := ecs.NewEntityManager()
	mockInput := &mockCheckboxMouseInput{
		mouseX:      115, // 在复选框内
		mouseY:      65,
		justPressed: false, // 鼠标未刚按下
	}
	system := NewCheckboxSystemWithInput(em, mockInput)

	// 创建复选框实体
	entity := em.CreateEntity()
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: 100,
		Y: 50,
	})

	var toggleCalled bool
	ecs.AddComponent(em, entity, &components.CheckboxComponent{
		IsChecked:      false,
		UncheckedImage: createTestImage(30, 30),
		CheckedImage:   createTestImage(30, 30),
		OnToggle: func(isChecked bool) {
			toggleCalled = true
		},
	})

	// 执行 Update
	system.Update(0.016)

	// 验证状态未切换
	checkbox, _ := ecs.GetComponent[*components.CheckboxComponent](em, entity)
	if checkbox.IsChecked {
		t.Error("Checkbox should remain unchecked")
	}
	if toggleCalled {
		t.Error("OnToggle callback should not be called")
	}
}

// TestCheckboxSystem_Update_Toggle 测试多次点击切换
func TestCheckboxSystem_Update_Toggle(t *testing.T) {
	em := ecs.NewEntityManager()
	mockInput := &mockCheckboxMouseInput{
		mouseX:      115,
		mouseY:      65,
		justPressed: true,
	}
	system := NewCheckboxSystemWithInput(em, mockInput)

	// 创建复选框实体
	entity := em.CreateEntity()
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: 100,
		Y: 50,
	})

	var toggleCount int
	ecs.AddComponent(em, entity, &components.CheckboxComponent{
		IsChecked:      false,
		UncheckedImage: createTestImage(30, 30),
		CheckedImage:   createTestImage(30, 30),
		OnToggle: func(isChecked bool) {
			toggleCount++
		},
	})

	// 第一次点击
	system.Update(0.016)
	checkbox, _ := ecs.GetComponent[*components.CheckboxComponent](em, entity)
	if !checkbox.IsChecked {
		t.Error("Checkbox should be checked after first click")
	}

	// 第二次点击（切换回未选中）
	system.Update(0.016)
	if checkbox.IsChecked {
		t.Error("Checkbox should be unchecked after second click")
	}

	if toggleCount != 2 {
		t.Errorf("Toggle count = %v, want 2", toggleCount)
	}
}

// TestCheckboxSystem_Update_NoCallback 测试没有回调时不崩溃
func TestCheckboxSystem_Update_NoCallback(t *testing.T) {
	em := ecs.NewEntityManager()
	mockInput := &mockCheckboxMouseInput{
		mouseX:      115,
		mouseY:      65,
		justPressed: true,
	}
	system := NewCheckboxSystemWithInput(em, mockInput)

	entity := em.CreateEntity()
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: 100,
		Y: 50,
	})
	ecs.AddComponent(em, entity, &components.CheckboxComponent{
		IsChecked:      false,
		UncheckedImage: createTestImage(30, 30),
		CheckedImage:   createTestImage(30, 30),
		OnToggle:       nil, // 没有回调
	})

	// 不应崩溃
	system.Update(0.016)

	checkbox, _ := ecs.GetComponent[*components.CheckboxComponent](em, entity)
	if !checkbox.IsChecked {
		t.Error("Checkbox should be checked")
	}
}

// TestCheckboxSystem_Update_NoImage 测试没有图片时跳过
func TestCheckboxSystem_Update_NoImage(t *testing.T) {
	em := ecs.NewEntityManager()
	mockInput := &mockCheckboxMouseInput{
		mouseX:      115,
		mouseY:      65,
		justPressed: true,
	}
	system := NewCheckboxSystemWithInput(em, mockInput)

	entity := em.CreateEntity()
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: 100,
		Y: 50,
	})

	var toggleCalled bool
	ecs.AddComponent(em, entity, &components.CheckboxComponent{
		IsChecked:      false,
		UncheckedImage: nil, // 没有图片
		CheckedImage:   nil,
		OnToggle: func(isChecked bool) {
			toggleCalled = true
		},
	})

	// 不应崩溃，也不应触发回调
	system.Update(0.016)

	if toggleCalled {
		t.Error("OnToggle should not be called when no image")
	}
}

// TestCheckboxSystem_Update_MultipleCheckboxes 测试多个复选框
func TestCheckboxSystem_Update_MultipleCheckboxes(t *testing.T) {
	em := ecs.NewEntityManager()
	mockInput := &mockCheckboxMouseInput{
		mouseX:      115, // 在第一个复选框范围内
		mouseY:      65,
		justPressed: true,
	}
	system := NewCheckboxSystemWithInput(em, mockInput)

	// 创建第一个复选框
	entity1 := em.CreateEntity()
	ecs.AddComponent(em, entity1, &components.PositionComponent{X: 100, Y: 50})
	var toggle1Called bool
	ecs.AddComponent(em, entity1, &components.CheckboxComponent{
		IsChecked:      false,
		UncheckedImage: createTestImage(30, 30),
		CheckedImage:   createTestImage(30, 30),
		OnToggle:       func(isChecked bool) { toggle1Called = true },
	})

	// 创建第二个复选框（不同位置）
	entity2 := em.CreateEntity()
	ecs.AddComponent(em, entity2, &components.PositionComponent{X: 100, Y: 150})
	var toggle2Called bool
	ecs.AddComponent(em, entity2, &components.CheckboxComponent{
		IsChecked:      false,
		UncheckedImage: createTestImage(30, 30),
		CheckedImage:   createTestImage(30, 30),
		OnToggle:       func(isChecked bool) { toggle2Called = true },
	})

	system.Update(0.016)

	// 只有第一个复选框应该被切换
	if !toggle1Called {
		t.Error("Checkbox 1 toggle should be called")
	}
	if toggle2Called {
		t.Error("Checkbox 2 toggle should not be called")
	}
}

// TestCheckboxSystem_Update_NoEntities 测试没有实体时不崩溃
func TestCheckboxSystem_Update_NoEntities(t *testing.T) {
	em := ecs.NewEntityManager()
	mockInput := &mockCheckboxMouseInput{
		mouseX:      115,
		mouseY:      65,
		justPressed: true,
	}
	system := NewCheckboxSystemWithInput(em, mockInput)

	// 不应崩溃
	system.Update(0.016)
}

// TestCheckboxSystem_Update_UsesCorrectImageForSize 测试使用正确的图片确定尺寸
func TestCheckboxSystem_Update_UsesCorrectImageForSize(t *testing.T) {
	em := ecs.NewEntityManager()

	// 测试未选中状态使用 UncheckedImage 的尺寸
	t.Run("使用UncheckedImage尺寸", func(t *testing.T) {
		mockInput := &mockCheckboxMouseInput{
			mouseX:      115,
			mouseY:      65,
			justPressed: true,
		}
		system := NewCheckboxSystemWithInput(em, mockInput)

		entity := em.CreateEntity()
		ecs.AddComponent(em, entity, &components.PositionComponent{X: 100, Y: 50})

		var toggleCalled bool
		ecs.AddComponent(em, entity, &components.CheckboxComponent{
			IsChecked:      false, // 未选中，使用 UncheckedImage
			UncheckedImage: createTestImage(30, 30),
			CheckedImage:   createTestImage(50, 50), // 不同尺寸
			OnToggle:       func(isChecked bool) { toggleCalled = true },
		})

		system.Update(0.016)

		if !toggleCalled {
			t.Error("Should toggle when clicking within UncheckedImage bounds")
		}

		em.DestroyEntity(entity)
	})

	// 测试选中状态使用 CheckedImage 的尺寸
	t.Run("使用CheckedImage尺寸", func(t *testing.T) {
		mockInput := &mockCheckboxMouseInput{
			mouseX:      140, // 在 CheckedImage 范围内但超出 UncheckedImage
			mouseY:      90,
			justPressed: true,
		}
		system := NewCheckboxSystemWithInput(em, mockInput)

		entity := em.CreateEntity()
		ecs.AddComponent(em, entity, &components.PositionComponent{X: 100, Y: 50})

		var toggleCalled bool
		ecs.AddComponent(em, entity, &components.CheckboxComponent{
			IsChecked:      true, // 选中，使用 CheckedImage
			UncheckedImage: createTestImage(30, 30),
			CheckedImage:   createTestImage(50, 50),
			OnToggle:       func(isChecked bool) { toggleCalled = true },
		})

		system.Update(0.016)

		if !toggleCalled {
			t.Error("Should toggle when clicking within CheckedImage bounds")
		}

		em.DestroyEntity(entity)
	})
}
