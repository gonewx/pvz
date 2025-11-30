package entities

import (
	"fmt"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// NewDialogEntity 创建通用对话框实体
//
// 参数：
//   - em: 实体管理器
//   - rm: 资源管理器
//   - title: 对话框标题
//   - message: 对话框消息
//   - buttons: 按钮文字列表
//   - windowWidth, windowHeight: 游戏窗口大小（用于居中）
//
// 返回：
//   - 对话框实体ID
//   - 错误信息
func NewDialogEntity(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	title string,
	message string,
	buttons []string,
	windowWidth, windowHeight int,
) (ecs.EntityID, error) {
	// 加载九宫格资源
	parts, err := loadDialogParts(rm)
	if err != nil {
		return 0, fmt.Errorf("加载对话框资源失败: %w", err)
	}

	// 加载按钮图片资源
	btnLeftImg, err := rm.LoadImageByID("IMAGE_BUTTON_LEFT")
	if err != nil {
		return 0, fmt.Errorf("加载按钮左边图片失败: %w", err)
	}

	btnMiddleImg, err := rm.LoadImageByID("IMAGE_BUTTON_MIDDLE")
	if err != nil {
		return 0, fmt.Errorf("加载按钮中间图片失败: %w", err)
	}

	btnRightImg, err := rm.LoadImageByID("IMAGE_BUTTON_RIGHT")
	if err != nil {
		return 0, fmt.Errorf("加载按钮右边图片失败: %w", err)
	}

	// 计算对话框大小
	dialogWidth, dialogHeight := calculateDialogSize(message)

	// 计算居中位置
	x := float64(windowWidth)/2 - dialogWidth/2
	y := float64(windowHeight)/2 - dialogHeight/2

	// 创建对话框实体
	entity := em.CreateEntity()

	// 添加位置组件
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// 创建对话框按钮
	dialogButtons := make([]components.DialogButton, 0, len(buttons))
	for i, btnText := range buttons {
		// 计算按钮尺寸
		btnLeftWidth := float64(btnLeftImg.Bounds().Dx())
		btnRightWidth := float64(btnRightImg.Bounds().Dx())
		btnMiddleWidth := 220.0 // 增加中间可拉伸部分宽度（原来是60）
		btnTotalWidth := btnLeftWidth + btnMiddleWidth + btnRightWidth
		btnHeight := float64(btnLeftImg.Bounds().Dy())

		// 按钮位置计算（底部居中）
		btnX := dialogWidth/2 - btnTotalWidth/2
		btnY := dialogHeight - 65.0

		dialogButtons = append(dialogButtons, components.DialogButton{
			Label:       btnText,
			X:           btnX,
			Y:           btnY,
			Width:       btnTotalWidth,
			Height:      btnHeight,
			LeftImage:   btnLeftImg,
			MiddleImage: btnMiddleImg,
			RightImage:  btnRightImg,
			MiddleWidth: btnMiddleWidth,
			OnClick: func() {
				// 点击回调将在 DialogInputSystem 中设置
				// 这里设置为 nil，后续由系统负责关闭逻辑
			},
		})

		_ = i // 暂时不处理多个按钮的情况
	}

	// 添加对话框组件
	ecs.AddComponent(em, entity, &components.DialogComponent{
		Title:            title,
		Message:          message,
		Buttons:          dialogButtons,
		Parts:            parts,
		IsVisible:        true,
		Width:            dialogWidth,
		Height:           dialogHeight,
		AutoClose:        true, // 错误对话框点击后自动关闭
		HoveredButtonIdx: -1,   // 初始化为未悬停状态
		PressedButtonIdx: -1,   // 初始化为未按下状态
	})

	// 添加 UI 组件标记
	ecs.AddComponent(em, entity, &components.UIComponent{
		State: components.UINormal,
	})

	return entity, nil
}

// loadDialogParts 加载九宫格对话框资源
func loadDialogParts(rm *game.ResourceManager) (*components.DialogParts, error) {
	parts := &components.DialogParts{}

	var err error

	// 加载四个边角
	parts.TopLeft, err = rm.LoadImageByID("IMAGE_DIALOG_TOPLEFT")
	if err != nil {
		return nil, fmt.Errorf("加载左上角失败: %w", err)
	}

	parts.TopRight, err = rm.LoadImageByID("IMAGE_DIALOG_TOPRIGHT")
	if err != nil {
		return nil, fmt.Errorf("加载右上角失败: %w", err)
	}

	parts.BottomLeft, err = rm.LoadImageByID("IMAGE_DIALOG_BOTTOMLEFT")
	if err != nil {
		return nil, fmt.Errorf("加载左下角失败: %w", err)
	}

	parts.BottomRight, err = rm.LoadImageByID("IMAGE_DIALOG_BOTTOMRIGHT")
	if err != nil {
		return nil, fmt.Errorf("加载右下角失败: %w", err)
	}

	// 加载四个边缘
	parts.TopMiddle, err = rm.LoadImageByID("IMAGE_DIALOG_TOPMIDDLE")
	if err != nil {
		return nil, fmt.Errorf("加载上边缘失败: %w", err)
	}

	parts.BottomMiddle, err = rm.LoadImageByID("IMAGE_DIALOG_BOTTOMMIDDLE")
	if err != nil {
		return nil, fmt.Errorf("加载下边缘失败: %w", err)
	}

	parts.CenterLeft, err = rm.LoadImageByID("IMAGE_DIALOG_CENTERLEFT")
	if err != nil {
		return nil, fmt.Errorf("加载左边缘失败: %w", err)
	}

	parts.CenterRight, err = rm.LoadImageByID("IMAGE_DIALOG_CENTERRIGHT")
	if err != nil {
		return nil, fmt.Errorf("加载右边缘失败: %w", err)
	}

	// 加载中心区域
	parts.CenterMiddle, err = rm.LoadImageByID("IMAGE_DIALOG_CENTERMIDDLE")
	if err != nil {
		return nil, fmt.Errorf("加载中心区域失败: %w", err)
	}

	// 加载骷髅头装饰（可选）
	parts.Header, _ = rm.LoadImageByID("IMAGE_DIALOG_HEADER")

	// 加载大对话框部分（可选）
	parts.BigBottomLeft, _ = rm.LoadImageByID("IMAGE_DIALOG_BIGBOTTOMLEFT")
	parts.BigBottomMiddle, _ = rm.LoadImageByID("IMAGE_DIALOG_BIGBOTTOMMIDDLE")
	parts.BigBottomRight, _ = rm.LoadImageByID("IMAGE_DIALOG_BIGBOTTOMRIGHT")

	return parts, nil
}

// calculateDialogSize 计算对话框大小
// message: 对话框消息文字
// 返回：宽度和高度
func calculateDialogSize(message string) (width, height float64) {
	// 常量定义
	const (
		MinWidth        = 400.0 // 最小宽度
		MaxWidth        = 600.0 // 最大宽度
		MinHeight       = 250.0 // 最小高度
		PaddingX        = 40.0  // 水平内边距（左右各20）
		PaddingY        = 40.0  // 垂直内边距（上下各20）
		TitleHeight     = 60.0  // 标题区域高度（包含骷髅头装饰空间）
		ButtonHeight    = 70.0  // 按钮区域高度
		MessageFontSize = 18.0  // 消息字体大小
		CharsPerLine    = 28    // 每行最大字符数（基于 MaxWidth 和字体大小估算）
		LineHeight      = 25.0  // 行高（字体大小 + 行间距）
	)

	// 计算宽度：根据文字长度估算
	messageLen := len([]rune(message)) // 使用 rune 计数以正确处理中文字符

	// 估算文字所需宽度（中文字符约等于字号宽度）
	estimatedTextWidth := float64(messageLen) * MessageFontSize * 0.7 // 0.7 是平均宽度系数
	requiredWidth := estimatedTextWidth + PaddingX

	// 限制在 MinWidth 和 MaxWidth 之间
	width = requiredWidth
	if width < MinWidth {
		width = MinWidth
	}
	if width > MaxWidth {
		width = MaxWidth
	}

	// 计算高度：根据文字行数估算
	// 计算需要多少行来显示消息
	lines := (messageLen + CharsPerLine - 1) / CharsPerLine // 向上取整
	if lines < 1 {
		lines = 1
	}

	// 总高度 = 标题区域 + 消息区域 + 按钮区域 + 垂直内边距
	messageAreaHeight := float64(lines) * LineHeight
	totalHeight := TitleHeight + messageAreaHeight + ButtonHeight + PaddingY

	// 确保不小于最小高度
	height = totalHeight
	if height < MinHeight {
		height = MinHeight
	}

	return width, height
}

// NewDialogEntityWithCallback 创建带回调的对话框实体
//
// Story 18.2: 战斗存档对话框需要自定义回调
//
// 参数：
//   - em: 实体管理器
//   - rm: 资源管理器
//   - title: 对话框标题
//   - message: 对话框消息
//   - buttons: 按钮文字列表
//   - windowWidth, windowHeight: 游戏窗口大小（用于居中）
//   - callback: 点击回调函数，参数为按钮索引
//
// 返回：
//   - 对话框实体ID
//   - 错误信息
func NewDialogEntityWithCallback(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	title string,
	message string,
	buttons []string,
	windowWidth, windowHeight int,
	callback func(buttonIndex int),
) (ecs.EntityID, error) {
	// 加载九宫格资源
	parts, err := loadDialogParts(rm)
	if err != nil {
		return 0, fmt.Errorf("加载对话框资源失败: %w", err)
	}

	// 加载按钮图片资源
	btnLeftImg, err := rm.LoadImageByID("IMAGE_BUTTON_LEFT")
	if err != nil {
		return 0, fmt.Errorf("加载按钮左边图片失败: %w", err)
	}

	btnMiddleImg, err := rm.LoadImageByID("IMAGE_BUTTON_MIDDLE")
	if err != nil {
		return 0, fmt.Errorf("加载按钮中间图片失败: %w", err)
	}

	btnRightImg, err := rm.LoadImageByID("IMAGE_BUTTON_RIGHT")
	if err != nil {
		return 0, fmt.Errorf("加载按钮右边图片失败: %w", err)
	}

	// 计算对话框大小
	dialogWidth, dialogHeight := calculateDialogSize(message)

	// 计算居中位置
	x := float64(windowWidth)/2 - dialogWidth/2
	y := float64(windowHeight)/2 - dialogHeight/2

	// 创建对话框实体
	entity := em.CreateEntity()

	// 添加位置组件
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// 创建对话框按钮
	dialogButtons := make([]components.DialogButton, 0, len(buttons))
	btnLeftWidth := float64(btnLeftImg.Bounds().Dx())
	btnRightWidth := float64(btnRightImg.Bounds().Dx())
	btnMiddleWidth := 100.0 // 按钮中间宽度
	btnTotalWidth := btnLeftWidth + btnMiddleWidth + btnRightWidth
	btnHeight := float64(btnLeftImg.Bounds().Dy())
	btnSpacing := 20.0 // 按钮间距

	// 计算按钮组总宽度
	totalButtonsWidth := float64(len(buttons))*btnTotalWidth + float64(len(buttons)-1)*btnSpacing
	startX := dialogWidth/2 - totalButtonsWidth/2

	for i, btnText := range buttons {
		btnIdx := i // 捕获闭包变量
		btnX := startX + float64(i)*(btnTotalWidth+btnSpacing)
		btnY := dialogHeight - 65.0

		dialogButtons = append(dialogButtons, components.DialogButton{
			Label:       btnText,
			X:           btnX,
			Y:           btnY,
			Width:       btnTotalWidth,
			Height:      btnHeight,
			LeftImage:   btnLeftImg,
			MiddleImage: btnMiddleImg,
			RightImage:  btnRightImg,
			MiddleWidth: btnMiddleWidth,
			OnClick: func() {
				if callback != nil {
					callback(btnIdx)
				}
			},
		})
	}

	// 添加对话框组件
	ecs.AddComponent(em, entity, &components.DialogComponent{
		Title:            title,
		Message:          message,
		Buttons:          dialogButtons,
		Parts:            parts,
		IsVisible:        true,
		Width:            dialogWidth,
		Height:           dialogHeight,
		AutoClose:        true, // 点击后自动关闭
		HoveredButtonIdx: -1,
		PressedButtonIdx: -1,
	})

	// 添加 UI 组件标记
	ecs.AddComponent(em, entity, &components.UIComponent{
		State: components.UINormal,
	})

	return entity, nil
}

// NewContinueGameDialogEntity 创建继续游戏对话框实体
//
// Story 18.3: 继续游戏对话框与场景恢复
//
// 对话框布局：
//
//	┌─────────────────────────────────────────┐
//	│              继续游戏?                   │
//	│                                         │
//	│   你想继续当前游戏还是重玩此关卡？         │
//	│                                         │
//	│      [继续]          [重玩关卡]          │ ← 第一行
//	│               [取消]                    │ ← 第二行
//	│                                         │
//	└─────────────────────────────────────────┘
//
// 参数：
//   - em: 实体管理器
//   - rm: 资源管理器
//   - info: 战斗存档信息（可选，用于显示详情）
//   - windowWidth, windowHeight: 游戏窗口大小（用于居中）
//   - onContinue: "继续"按钮回调
//   - onRestart: "重玩关卡"按钮回调
//   - onCancel: "取消"按钮回调
//
// 返回：
//   - 对话框实体ID
//   - 错误信息
func NewContinueGameDialogEntity(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	info *game.BattleSaveInfo,
	windowWidth, windowHeight int,
	onContinue, onRestart, onCancel func(),
) (ecs.EntityID, error) {
	// 加载九宫格资源
	parts, err := loadDialogParts(rm)
	if err != nil {
		return 0, fmt.Errorf("加载对话框资源失败: %w", err)
	}

	// 加载按钮图片资源
	btnLeftImg, err := rm.LoadImageByID("IMAGE_BUTTON_LEFT")
	if err != nil {
		return 0, fmt.Errorf("加载按钮左边图片失败: %w", err)
	}

	btnMiddleImg, err := rm.LoadImageByID("IMAGE_BUTTON_MIDDLE")
	if err != nil {
		return 0, fmt.Errorf("加载按钮中间图片失败: %w", err)
	}

	btnRightImg, err := rm.LoadImageByID("IMAGE_BUTTON_RIGHT")
	if err != nil {
		return 0, fmt.Errorf("加载按钮右边图片失败: %w", err)
	}

	// 构建对话框消息
	message := "你想继续当前游戏还是重玩此关卡？"

	// 固定对话框大小（适应两行按钮）
	dialogWidth := 420.0
	dialogHeight := 280.0 // 增加高度以容纳两行按钮

	// 计算居中位置
	x := float64(windowWidth)/2 - dialogWidth/2
	y := float64(windowHeight)/2 - dialogHeight/2

	// 创建对话框实体
	entity := em.CreateEntity()

	// 添加位置组件
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// 按钮尺寸常量
	btnLeftWidth := float64(btnLeftImg.Bounds().Dx())
	btnRightWidth := float64(btnRightImg.Bounds().Dx())
	btnMiddleWidth := 80.0 // 按钮中间宽度（较小）
	btnTotalWidth := btnLeftWidth + btnMiddleWidth + btnRightWidth
	btnHeight := float64(btnLeftImg.Bounds().Dy())
	btnSpacing := 20.0 // 按钮水平间距
	rowSpacing := 10.0 // 行间距

	// 避免编译警告
	_ = btnRightWidth

	// 第一行按钮位置计算（继续 + 重玩关卡）
	row1ButtonCount := 2
	row1TotalWidth := float64(row1ButtonCount)*btnTotalWidth + float64(row1ButtonCount-1)*btnSpacing
	row1StartX := dialogWidth/2 - row1TotalWidth/2
	row1Y := dialogHeight - 65.0 - btnHeight - rowSpacing // 第一行在第二行上方

	// 第二行按钮位置计算（取消，居中）
	row2Y := dialogHeight - 65.0 // 底部
	row2StartX := dialogWidth/2 - btnTotalWidth/2

	// 创建三个按钮
	dialogButtons := []components.DialogButton{
		// 按钮0: 继续（第一行左边）
		{
			Label:       "继续",
			X:           row1StartX,
			Y:           row1Y,
			Width:       btnTotalWidth,
			Height:      btnHeight,
			LeftImage:   btnLeftImg,
			MiddleImage: btnMiddleImg,
			RightImage:  btnRightImg,
			MiddleWidth: btnMiddleWidth,
			OnClick:     onContinue,
		},
		// 按钮1: 重玩关卡（第一行右边）
		{
			Label:       "重玩关卡",
			X:           row1StartX + btnTotalWidth + btnSpacing,
			Y:           row1Y,
			Width:       btnTotalWidth,
			Height:      btnHeight,
			LeftImage:   btnLeftImg,
			MiddleImage: btnMiddleImg,
			RightImage:  btnRightImg,
			MiddleWidth: btnMiddleWidth,
			OnClick:     onRestart,
		},
		// 按钮2: 取消（第二行居中）
		{
			Label:       "取消",
			X:           row2StartX,
			Y:           row2Y,
			Width:       btnTotalWidth,
			Height:      btnHeight,
			LeftImage:   btnLeftImg,
			MiddleImage: btnMiddleImg,
			RightImage:  btnRightImg,
			MiddleWidth: btnMiddleWidth,
			OnClick:     onCancel,
		},
	}

	// 添加对话框组件
	ecs.AddComponent(em, entity, &components.DialogComponent{
		Title:            "继续游戏?",
		Message:          message,
		Buttons:          dialogButtons,
		Parts:            parts,
		IsVisible:        true,
		Width:            dialogWidth,
		Height:           dialogHeight,
		AutoClose:        true, // 点击后自动关闭
		HoveredButtonIdx: -1,
		PressedButtonIdx: -1,
		UseBigBottom:     true, // 两行按钮布局，使用大底部区域
	})

	// 添加 UI 组件标记
	ecs.AddComponent(em, entity, &components.UIComponent{
		State: components.UINormal,
	})

	return entity, nil
}
