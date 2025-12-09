package systems

import (
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// mockSliderMouseInput 用于测试的 mock 鼠标输入
type mockSliderMouseInput struct {
	mouseX       int
	mouseY       int
	mousePressed bool
}

func (m *mockSliderMouseInput) CursorPosition() (int, int) {
	return m.mouseX, m.mouseY
}

func (m *mockSliderMouseInput) IsMouseButtonPressed(button ebiten.MouseButton) bool {
	return m.mousePressed
}

// TestSliderSystem_calculateValue 测试滑块值计算
func TestSliderSystem_calculateValue(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewSliderSystem(em)

	tests := []struct {
		name      string
		mouseX    float64
		slotX     float64
		slotWidth float64
		expected  float64
	}{
		{
			name:      "左边界",
			mouseX:    100,
			slotX:     100,
			slotWidth: 200,
			expected:  0.0,
		},
		{
			name:      "右边界",
			mouseX:    300,
			slotX:     100,
			slotWidth: 200,
			expected:  1.0,
		},
		{
			name:      "中间位置",
			mouseX:    200,
			slotX:     100,
			slotWidth: 200,
			expected:  0.5,
		},
		{
			name:      "25%位置",
			mouseX:    150,
			slotX:     100,
			slotWidth: 200,
			expected:  0.25,
		},
		{
			name:      "75%位置",
			mouseX:    250,
			slotX:     100,
			slotWidth: 200,
			expected:  0.75,
		},
		{
			name:      "零宽度滑槽",
			mouseX:    100,
			slotX:     100,
			slotWidth: 0,
			expected:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := system.calculateValue(tt.mouseX, tt.slotX, tt.slotWidth)
			if result != tt.expected {
				t.Errorf("calculateValue() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestSliderSystem_isMouseInSlot 测试鼠标在滑槽内检测
func TestSliderSystem_isMouseInSlot(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewSliderSystem(em)

	tests := []struct {
		name       string
		mouseX     float64
		mouseY     float64
		slotX      float64
		slotY      float64
		slotWidth  float64
		slotHeight float64
		expected   bool
	}{
		{
			name:       "在滑槽内",
			mouseX:     150,
			mouseY:     25,
			slotX:      100,
			slotY:      20,
			slotWidth:  200,
			slotHeight: 20,
			expected:   true,
		},
		{
			name:       "左边界外",
			mouseX:     99,
			mouseY:     25,
			slotX:      100,
			slotY:      20,
			slotWidth:  200,
			slotHeight: 20,
			expected:   false,
		},
		{
			name:       "右边界外",
			mouseX:     301,
			mouseY:     25,
			slotX:      100,
			slotY:      20,
			slotWidth:  200,
			slotHeight: 20,
			expected:   false,
		},
		{
			name:       "上边界外",
			mouseX:     150,
			mouseY:     19,
			slotX:      100,
			slotY:      20,
			slotWidth:  200,
			slotHeight: 20,
			expected:   false,
		},
		{
			name:       "下边界外",
			mouseX:     150,
			mouseY:     41,
			slotX:      100,
			slotY:      20,
			slotWidth:  200,
			slotHeight: 20,
			expected:   false,
		},
		{
			name:       "左上角",
			mouseX:     100,
			mouseY:     20,
			slotX:      100,
			slotY:      20,
			slotWidth:  200,
			slotHeight: 20,
			expected:   true,
		},
		{
			name:       "右下角",
			mouseX:     300,
			mouseY:     40,
			slotX:      100,
			slotY:      20,
			slotWidth:  200,
			slotHeight: 20,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := system.isMouseInSlot(tt.mouseX, tt.mouseY, tt.slotX, tt.slotY, tt.slotWidth, tt.slotHeight)
			if result != tt.expected {
				t.Errorf("isMouseInSlot() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestNewSliderSystem 测试滑块系统创建
func TestNewSliderSystem(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewSliderSystem(em)

	if system == nil {
		t.Fatal("NewSliderSystem() returned nil")
	}
	if system.entityManager != em {
		t.Error("entityManager not set correctly")
	}
}

// TestSliderSystem_EntityWithSliderComponent 测试创建带滑块组件的实体
func TestSliderSystem_EntityWithSliderComponent(t *testing.T) {
	em := ecs.NewEntityManager()

	// 创建滑块实体
	entity := em.CreateEntity()
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: 100,
		Y: 50,
	})

	var callbackValue float64 = -1
	ecs.AddComponent(em, entity, &components.SliderComponent{
		SlotWidth:  200,
		SlotHeight: 20,
		Value:      0.5,
		Label:      "Test Slider",
		OnValueChange: func(value float64) {
			callbackValue = value
		},
	})

	// 验证组件正确添加
	slider, ok := ecs.GetComponent[*components.SliderComponent](em, entity)
	if !ok {
		t.Fatal("SliderComponent not found")
	}
	if slider.Value != 0.5 {
		t.Errorf("Initial value = %v, want 0.5", slider.Value)
	}
	if slider.Label != "Test Slider" {
		t.Errorf("Label = %v, want 'Test Slider'", slider.Label)
	}

	// 模拟值变化
	slider.Value = 0.8
	if slider.OnValueChange != nil {
		slider.OnValueChange(slider.Value)
	}
	if callbackValue != 0.8 {
		t.Errorf("Callback value = %v, want 0.8", callbackValue)
	}
}

// TestSliderSystem_Update_ClickInSlot 测试点击滑槽内更新值
func TestSliderSystem_Update_ClickInSlot(t *testing.T) {
	em := ecs.NewEntityManager()
	mockInput := &mockSliderMouseInput{
		mouseX:       150, // 中间位置
		mouseY:       55,  // 在滑槽内
		mousePressed: true,
	}
	system := NewSliderSystemWithInput(em, mockInput)

	// 创建滑块实体
	entity := em.CreateEntity()
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: 100,
		Y: 50,
	})

	var callbackValue float64 = -1
	ecs.AddComponent(em, entity, &components.SliderComponent{
		SlotWidth:  100,
		SlotHeight: 20,
		Value:      0.0,
		OnValueChange: func(value float64) {
			callbackValue = value
		},
	})

	// 执行 Update
	system.Update(0.016)

	// 验证值已更新
	slider, _ := ecs.GetComponent[*components.SliderComponent](em, entity)
	if slider.Value != 0.5 {
		t.Errorf("Slider value = %v, want 0.5", slider.Value)
	}
	if callbackValue != 0.5 {
		t.Errorf("Callback value = %v, want 0.5", callbackValue)
	}
}

// TestSliderSystem_Update_ClickOutsideSlot 测试点击滑槽外不更新值
func TestSliderSystem_Update_ClickOutsideSlot(t *testing.T) {
	em := ecs.NewEntityManager()
	mockInput := &mockSliderMouseInput{
		mouseX:       50, // 滑槽外
		mouseY:       55,
		mousePressed: true,
	}
	system := NewSliderSystemWithInput(em, mockInput)

	// 创建滑块实体
	entity := em.CreateEntity()
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: 100,
		Y: 50,
	})

	var callbackCalled bool
	ecs.AddComponent(em, entity, &components.SliderComponent{
		SlotWidth:  100,
		SlotHeight: 20,
		Value:      0.3,
		OnValueChange: func(value float64) {
			callbackCalled = true
		},
	})

	// 执行 Update
	system.Update(0.016)

	// 验证值未更新
	slider, _ := ecs.GetComponent[*components.SliderComponent](em, entity)
	if slider.Value != 0.3 {
		t.Errorf("Slider value = %v, want 0.3 (unchanged)", slider.Value)
	}
	if callbackCalled {
		t.Error("Callback should not be called when clicking outside slot")
	}
}

// TestSliderSystem_Update_MouseNotPressed 测试鼠标未按下时不更新值
func TestSliderSystem_Update_MouseNotPressed(t *testing.T) {
	em := ecs.NewEntityManager()
	mockInput := &mockSliderMouseInput{
		mouseX:       150, // 在滑槽内
		mouseY:       55,
		mousePressed: false, // 鼠标未按下
	}
	system := NewSliderSystemWithInput(em, mockInput)

	// 创建滑块实体
	entity := em.CreateEntity()
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: 100,
		Y: 50,
	})

	var callbackCalled bool
	ecs.AddComponent(em, entity, &components.SliderComponent{
		SlotWidth:  100,
		SlotHeight: 20,
		Value:      0.3,
		OnValueChange: func(value float64) {
			callbackCalled = true
		},
	})

	// 执行 Update
	system.Update(0.016)

	// 验证值未更新
	slider, _ := ecs.GetComponent[*components.SliderComponent](em, entity)
	if slider.Value != 0.3 {
		t.Errorf("Slider value = %v, want 0.3 (unchanged)", slider.Value)
	}
	if callbackCalled {
		t.Error("Callback should not be called when mouse not pressed")
	}
}

// TestSliderSystem_Update_Dragging 测试拖拽功能
func TestSliderSystem_Update_Dragging(t *testing.T) {
	em := ecs.NewEntityManager()
	mockInput := &mockSliderMouseInput{
		mouseX:       150, // 初始位置
		mouseY:       55,
		mousePressed: true,
	}
	system := NewSliderSystemWithInput(em, mockInput)

	// 创建滑块实体
	entity := em.CreateEntity()
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: 100,
		Y: 50,
	})

	ecs.AddComponent(em, entity, &components.SliderComponent{
		SlotWidth:     100,
		SlotHeight:    20,
		Value:         0.0,
		OnValueChange: func(value float64) {},
	})

	// 第一次点击
	system.Update(0.016)

	slider, _ := ecs.GetComponent[*components.SliderComponent](em, entity)
	if !slider.IsDragging {
		t.Error("IsDragging should be true after click")
	}

	// 模拟拖拽到不同位置（即使鼠标移出滑槽，仍应更新值）
	mockInput.mouseX = 180 // 超出滑槽右侧
	mockInput.mouseY = 100 // 超出滑槽下方
	system.Update(0.016)

	if slider.Value != 0.8 {
		t.Errorf("Slider value = %v, want 0.8 after drag", slider.Value)
	}

	// 释放鼠标
	mockInput.mousePressed = false
	system.Update(0.016)

	if slider.IsDragging {
		t.Error("IsDragging should be false after mouse release")
	}
}

// TestSliderSystem_Update_ValueClamp 测试值的边界限制
func TestSliderSystem_Update_ValueClamp(t *testing.T) {
	em := ecs.NewEntityManager()

	tests := []struct {
		name     string
		mouseX   int
		expected float64
	}{
		{"超出左边界", 50, 0.0},
		{"超出右边界", 250, 1.0},
		{"左边界", 100, 0.0},
		{"右边界", 200, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInput := &mockSliderMouseInput{
				mouseX:       tt.mouseX,
				mouseY:       55,
				mousePressed: true,
			}
			system := NewSliderSystemWithInput(em, mockInput)

			entity := em.CreateEntity()
			ecs.AddComponent(em, entity, &components.PositionComponent{
				X: 100,
				Y: 50,
			})
			ecs.AddComponent(em, entity, &components.SliderComponent{
				SlotWidth:     100,
				SlotHeight:    20,
				Value:         0.5,
				IsDragging:    true, // 设置为拖拽状态以测试边界限制
				OnValueChange: func(value float64) {},
			})

			system.Update(0.016)

			slider, _ := ecs.GetComponent[*components.SliderComponent](em, entity)
			if slider.Value != tt.expected {
				t.Errorf("Slider value = %v, want %v", slider.Value, tt.expected)
			}

			// 清理实体
			em.DestroyEntity(entity)
		})
	}
}

// TestSliderSystem_Update_NoCallback 测试没有回调时不崩溃
func TestSliderSystem_Update_NoCallback(t *testing.T) {
	em := ecs.NewEntityManager()
	mockInput := &mockSliderMouseInput{
		mouseX:       150,
		mouseY:       55,
		mousePressed: true,
	}
	system := NewSliderSystemWithInput(em, mockInput)

	entity := em.CreateEntity()
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: 100,
		Y: 50,
	})
	ecs.AddComponent(em, entity, &components.SliderComponent{
		SlotWidth:     100,
		SlotHeight:    20,
		Value:         0.0,
		OnValueChange: nil, // 没有回调
	})

	// 不应崩溃
	system.Update(0.016)

	slider, _ := ecs.GetComponent[*components.SliderComponent](em, entity)
	if slider.Value != 0.5 {
		t.Errorf("Slider value = %v, want 0.5", slider.Value)
	}
}

// TestSliderSystem_Update_MultipleSliders 测试多个滑块
func TestSliderSystem_Update_MultipleSliders(t *testing.T) {
	em := ecs.NewEntityManager()
	mockInput := &mockSliderMouseInput{
		mouseX:       150, // 在第一个滑块范围内
		mouseY:       55,
		mousePressed: true,
	}
	system := NewSliderSystemWithInput(em, mockInput)

	// 创建第一个滑块
	entity1 := em.CreateEntity()
	ecs.AddComponent(em, entity1, &components.PositionComponent{X: 100, Y: 50})
	var callback1Value float64 = -1
	ecs.AddComponent(em, entity1, &components.SliderComponent{
		SlotWidth:     100,
		SlotHeight:    20,
		Value:         0.0,
		OnValueChange: func(v float64) { callback1Value = v },
	})

	// 创建第二个滑块（不同位置）
	entity2 := em.CreateEntity()
	ecs.AddComponent(em, entity2, &components.PositionComponent{X: 100, Y: 100})
	var callback2Value float64 = -1
	ecs.AddComponent(em, entity2, &components.SliderComponent{
		SlotWidth:     100,
		SlotHeight:    20,
		Value:         0.0,
		OnValueChange: func(v float64) { callback2Value = v },
	})

	system.Update(0.016)

	// 只有第一个滑块应该被更新
	if callback1Value != 0.5 {
		t.Errorf("Slider 1 callback value = %v, want 0.5", callback1Value)
	}
	if callback2Value != -1 {
		t.Errorf("Slider 2 callback should not be called, got %v", callback2Value)
	}
}

// TestSliderSystem_Update_NoEntities 测试没有实体时不崩溃
func TestSliderSystem_Update_NoEntities(t *testing.T) {
	em := ecs.NewEntityManager()
	mockInput := &mockSliderMouseInput{
		mouseX:       150,
		mouseY:       55,
		mousePressed: true,
	}
	system := NewSliderSystemWithInput(em, mockInput)

	// 不应崩溃
	system.Update(0.016)
}
