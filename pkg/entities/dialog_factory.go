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
