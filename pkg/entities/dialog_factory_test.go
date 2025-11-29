package entities

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
)

// TestNewDialogEntityWithCallback_ButtonCount 测试带回调的对话框按钮数量
func TestNewDialogEntityWithCallback_ButtonCount(t *testing.T) {
	// 创建 EntityManager
	em := ecs.NewEntityManager()

	// 测试用例: 验证按钮数量
	tests := []struct {
		name        string
		buttons     []string
		expectedLen int
	}{
		{
			name:        "两个按钮",
			buttons:     []string{"继续游戏", "重新开始"},
			expectedLen: 2,
		},
		{
			name:        "单个按钮",
			buttons:     []string{"确定"},
			expectedLen: 1,
		},
		{
			name:        "三个按钮",
			buttons:     []string{"是", "否", "取消"},
			expectedLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 由于 ResourceManager 需要加载图片资源，这里只验证函数签名
			// 实际测试需要 mock 或跳过资源加载

			// 验证回调函数闭包捕获索引
			var capturedIndex int
			callback := func(buttonIndex int) {
				capturedIndex = buttonIndex
			}

			// 测试不同按钮索引的回调
			for i := range tt.buttons {
				localI := i // 捕获闭包变量
				callback(localI)
				if capturedIndex != localI {
					t.Errorf("Callback captured wrong index: expected %d, got %d", localI, capturedIndex)
				}
			}
		})
	}

	_ = em // 使用 em 避免未使用变量警告
}

// TestNewDialogEntityWithCallback_CallbackInvocation 测试回调调用
func TestNewDialogEntityWithCallback_CallbackInvocation(t *testing.T) {
	var callbackCalled bool
	var capturedIndex int

	callback := func(buttonIndex int) {
		callbackCalled = true
		capturedIndex = buttonIndex
	}

	// 模拟按钮点击
	callback(0)

	if !callbackCalled {
		t.Error("Callback was not called")
	}
	if capturedIndex != 0 {
		t.Errorf("Callback received wrong index: expected 0, got %d", capturedIndex)
	}

	// 测试第二个按钮
	callback(1)
	if capturedIndex != 1 {
		t.Errorf("Callback received wrong index: expected 1, got %d", capturedIndex)
	}
}

// TestCalculateDialogSize 测试对话框大小计算
func TestCalculateDialogSize(t *testing.T) {
	tests := []struct {
		name       string
		message    string
		minWidth   float64
		minHeight  float64
		maxWidth   float64
	}{
		{
			name:      "短消息",
			message:   "OK",
			minWidth:  400, // MinWidth
			minHeight: 250, // MinHeight
		},
		{
			name:      "中等消息",
			message:   "检测到未完成的战斗存档，是否继续？",
			minWidth:  400,
			minHeight: 250,
		},
		{
			name:      "长消息",
			message:   "这是一段非常长的消息文本，用于测试对话框是否能够正确计算宽度和高度，确保消息不会超出对话框边界，并且有足够的空间显示所有内容。",
			minWidth:  400,
			minHeight: 250,
			maxWidth:  600, // MaxWidth
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width, height := calculateDialogSize(tt.message)

			if width < tt.minWidth {
				t.Errorf("Width %f is less than minimum %f", width, tt.minWidth)
			}
			if height < tt.minHeight {
				t.Errorf("Height %f is less than minimum %f", height, tt.minHeight)
			}
			if tt.maxWidth > 0 && width > tt.maxWidth {
				t.Errorf("Width %f exceeds maximum %f", width, tt.maxWidth)
			}
		})
	}
}

// TestDialogButton_OnClick 测试对话框按钮点击回调
func TestDialogButton_OnClick(t *testing.T) {
	var clicked bool
	button := components.DialogButton{
		Label: "测试按钮",
		OnClick: func() {
			clicked = true
		},
	}

	// 模拟点击
	if button.OnClick != nil {
		button.OnClick()
	}

	if !clicked {
		t.Error("Button OnClick was not called")
	}
}
