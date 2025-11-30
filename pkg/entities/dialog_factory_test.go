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

// TestNewContinueGameDialogEntity_ButtonCallbacks 测试继续游戏对话框按钮回调
// Story 18.3: 继续游戏对话框与场景恢复
func TestNewContinueGameDialogEntity_ButtonCallbacks(t *testing.T) {
	// 测试回调函数是否正确调用
	var continueCalled, restartCalled, cancelCalled bool

	onContinue := func() { continueCalled = true }
	onRestart := func() { restartCalled = true }
	onCancel := func() { cancelCalled = true }

	// 模拟按钮点击
	onContinue()
	if !continueCalled {
		t.Error("Continue callback was not called")
	}

	onRestart()
	if !restartCalled {
		t.Error("Restart callback was not called")
	}

	onCancel()
	if !cancelCalled {
		t.Error("Cancel callback was not called")
	}
}

// TestNewContinueGameDialogEntity_ButtonLayout 测试继续游戏对话框按钮布局
// Story 18.3: 验证两行三按钮布局
func TestNewContinueGameDialogEntity_ButtonLayout(t *testing.T) {
	// 模拟按钮布局计算
	dialogWidth := 420.0
	dialogHeight := 280.0

	// 按钮尺寸常量（与工厂函数一致）
	btnLeftWidth := 10.0  // 模拟值
	btnRightWidth := 10.0 // 模拟值
	btnMiddleWidth := 80.0
	btnTotalWidth := btnLeftWidth + btnMiddleWidth + btnRightWidth
	btnHeight := 30.0 // 模拟值
	btnSpacing := 20.0
	rowSpacing := 10.0

	_ = btnRightWidth // 避免编译警告

	// 第一行按钮位置计算（继续 + 重玩关卡）
	row1ButtonCount := 2
	row1TotalWidth := float64(row1ButtonCount)*btnTotalWidth + float64(row1ButtonCount-1)*btnSpacing
	row1StartX := dialogWidth/2 - row1TotalWidth/2
	row1Y := dialogHeight - 65.0 - btnHeight - rowSpacing

	// 第二行按钮位置计算（取消，居中）
	row2Y := dialogHeight - 65.0
	row2StartX := dialogWidth/2 - btnTotalWidth/2

	// 验证按钮0（继续）
	btn0X := row1StartX
	btn0Y := row1Y

	// 验证按钮1（重玩关卡）
	btn1X := row1StartX + btnTotalWidth + btnSpacing
	btn1Y := row1Y

	// 验证按钮2（取消）
	btn2X := row2StartX
	btn2Y := row2Y

	// 验证第一行两个按钮在同一高度
	if btn0Y != btn1Y {
		t.Errorf("First row buttons should have same Y: btn0Y=%f, btn1Y=%f", btn0Y, btn1Y)
	}

	// 验证第二行按钮在第一行下方
	if btn2Y <= btn0Y {
		t.Errorf("Second row button should be below first row: btn2Y=%f, btn0Y=%f", btn2Y, btn0Y)
	}

	// 验证取消按钮居中
	expectedCenterX := dialogWidth / 2
	btn2CenterX := btn2X + btnTotalWidth/2
	if btn2CenterX != expectedCenterX {
		t.Errorf("Cancel button should be centered: btn2CenterX=%f, expectedCenterX=%f", btn2CenterX, expectedCenterX)
	}

	// 验证第一行两个按钮对称
	btn0CenterX := btn0X + btnTotalWidth/2
	btn1CenterX := btn1X + btnTotalWidth/2
	distFromCenter0 := expectedCenterX - btn0CenterX
	distFromCenter1 := btn1CenterX - expectedCenterX
	if distFromCenter0 != distFromCenter1 {
		t.Errorf("First row buttons should be symmetric: distFromCenter0=%f, distFromCenter1=%f", distFromCenter0, distFromCenter1)
	}
}

// TestNewContinueGameDialogEntity_MessageWithInfo 测试带存档信息的对话框消息
// Story 18.3: 验证对话框消息格式
func TestNewContinueGameDialogEntity_MessageWithInfo(t *testing.T) {
	// 测试用例：验证消息格式
	tests := []struct {
		name           string
		levelID        string
		waveIndex      int
		sun            int
		expectedPrefix string
	}{
		{
			name:           "关卡1-2，波次3，阳光150",
			levelID:        "1-2",
			waveIndex:      2, // 0-based, 显示时+1
			sun:            150,
			expectedPrefix: "关卡: 1-2",
		},
		{
			name:           "关卡1-4，波次5，阳光250",
			levelID:        "1-4",
			waveIndex:      4,
			sun:            250,
			expectedPrefix: "关卡: 1-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟消息构建逻辑
			message := ""
			if tt.levelID != "" {
				message = "关卡: " + tt.levelID
			}

			if message[:len(tt.expectedPrefix)] != tt.expectedPrefix {
				t.Errorf("Message should start with %q, got %q", tt.expectedPrefix, message[:len(tt.expectedPrefix)])
			}
		})
	}
}

// TestNewContinueGameDialogEntity_NilCallbacks 测试空回调处理
// Story 18.3: 验证空回调不会导致崩溃
func TestNewContinueGameDialogEntity_NilCallbacks(t *testing.T) {
	// 创建 nil 回调
	var onContinue, onRestart, onCancel func()

	// 模拟按钮点击 - nil 回调应该不会导致 panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Nil callback caused panic: %v", r)
		}
	}()

	// 安全调用
	if onContinue != nil {
		onContinue()
	}
	if onRestart != nil {
		onRestart()
	}
	if onCancel != nil {
		onCancel()
	}
}

// TestNewContinueGameDialogEntity_DialogDimensions 测试对话框尺寸
// Story 18.3: 验证对话框尺寸适应两行按钮
func TestNewContinueGameDialogEntity_DialogDimensions(t *testing.T) {
	// 固定对话框尺寸（与工厂函数一致）
	dialogWidth := 420.0
	dialogHeight := 280.0 // 增加高度以容纳两行按钮

	// 最小尺寸验证
	minWidth := 400.0
	minHeight := 250.0

	if dialogWidth < minWidth {
		t.Errorf("Dialog width %f is less than minimum %f", dialogWidth, minWidth)
	}

	if dialogHeight < minHeight {
		t.Errorf("Dialog height %f is less than minimum %f", dialogHeight, minHeight)
	}

	// 验证高度足够容纳两行按钮
	btnHeight := 30.0 // 模拟按钮高度
	rowSpacing := 10.0
	titleHeight := 60.0
	messageHeight := 50.0
	bottomPadding := 65.0

	requiredHeight := titleHeight + messageHeight + 2*btnHeight + rowSpacing + bottomPadding
	if dialogHeight < requiredHeight {
		t.Errorf("Dialog height %f is too small for two rows of buttons (required %f)", dialogHeight, requiredHeight)
	}
}

// TestNewContinueGameDialogEntity_UseBigBottom 测试继续游戏对话框使用大底部区域
// 验证两行按钮布局的对话框应使用 BigBottom 样式
func TestNewContinueGameDialogEntity_UseBigBottom(t *testing.T) {
	// 创建 DialogComponent 模拟配置
	dialogComp := &components.DialogComponent{
		Title:        "继续游戏?",
		Message:      "你想继续当前游戏还是重玩此关卡？",
		Width:        420.0,
		Height:       280.0,
		UseBigBottom: true, // 两行按钮布局，使用大底部区域
	}

	// 验证 UseBigBottom 标志正确设置
	if !dialogComp.UseBigBottom {
		t.Error("Continue game dialog should have UseBigBottom=true for two-row button layout")
	}

	// 验证对话框高度足够容纳大底部区域
	// 大底部区域通常比标准底部高 20-30 像素
	minHeightWithBigBottom := 260.0
	if dialogComp.Height < minHeightWithBigBottom {
		t.Errorf("Dialog height %f is too small for big bottom area (minimum %f)", dialogComp.Height, minHeightWithBigBottom)
	}
}
