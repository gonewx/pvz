package entities

import (
	"fmt"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// RenameUserDialogResult 重命名用户对话框的回调结果
type RenameUserDialogResult struct {
	Confirmed bool   // 用户是否确认
	OldName   string // 旧用户名
	NewName   string // 新用户名
}

// RenameUserDialogCallback 重命名用户对话框的回调函数类型
type RenameUserDialogCallback func(result RenameUserDialogResult)

// NewRenameUserDialogEntity 创建重命名用户对话框实体
//
// 参数：
//   - em: 实体管理器
//   - rm: 资源管理器
//   - oldUsername: 当前用户名（用于填充输入框）
//   - windowWidth, windowHeight: 游戏窗口大小
//   - callback: 操作回调函数
//
// 返回：
//   - 对话框实体ID
//   - 文本输入框实体ID
//   - 错误信息
func NewRenameUserDialogEntity(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	oldUsername string,
	windowWidth, windowHeight int,
	callback RenameUserDialogCallback,
) (dialogID ecs.EntityID, inputBoxID ecs.EntityID, err error) {
	// 加载九宫格对话框资源
	parts, err := loadDialogParts(rm)
	if err != nil {
		return 0, 0, fmt.Errorf("加载对话框资源失败: %w", err)
	}

	// 加载按钮图片资源
	btnLeftImg, err := rm.LoadImageByID("IMAGE_BUTTON_LEFT")
	if err != nil {
		return 0, 0, fmt.Errorf("加载按钮左边图片失败: %w", err)
	}

	btnMiddleImg, err := rm.LoadImageByID("IMAGE_BUTTON_MIDDLE")
	if err != nil {
		return 0, 0, fmt.Errorf("加载按钮中间图片失败: %w", err)
	}

	btnRightImg, err := rm.LoadImageByID("IMAGE_BUTTON_RIGHT")
	if err != nil {
		return 0, 0, fmt.Errorf("加载按钮右边图片失败: %w", err)
	}

	// 加载输入框图片资源
	editboxBorderImg, err := rm.LoadImageByID("IMAGE_EDITBOX")
	if err != nil {
		return 0, 0, fmt.Errorf("加载输入框边框失败: %w", err)
	}

	editboxBgImg, _ := rm.LoadImageByID("IMAGE_EDITBOX_BACKGROUND") // 可选

	// 对话框尺寸
	const dialogWidth = 500.0
	const dialogHeight = 250.0

	// 计算居中位置
	dialogX := float64(windowWidth)/2 - dialogWidth/2
	dialogY := float64(windowHeight)/2 - dialogHeight/2

	// 创建对话框实体
	dialogEntity := em.CreateEntity()

	// 添加位置组件
	ecs.AddComponent(em, dialogEntity, &components.PositionComponent{
		X: dialogX,
		Y: dialogY,
	})

	// 输入框尺寸和位置（相对对话框）
	const inputBoxWidth = 350.0
	const inputBoxHeight = 30.0
	inputBoxX := dialogWidth/2 - inputBoxWidth/2
	inputBoxY := 110.0 // 往下移动（原来是90.0）

	// 创建输入框实体
	inputEntity := em.CreateEntity()

	// 添加位置组件（绝对位置 = 对话框位置 + 相对位置）
	ecs.AddComponent(em, inputEntity, &components.PositionComponent{
		X: dialogX + inputBoxX,
		Y: dialogY + inputBoxY,
	})

	// 添加输入框组件（默认填充旧用户名）
	ecs.AddComponent(em, inputEntity, &components.TextInputComponent{
		Text:             oldUsername, // 填充旧用户名
		BorderImage:      editboxBorderImg,
		BackgroundImage:  editboxBgImg,
		Width:            inputBoxWidth,
		Height:           inputBoxHeight,
		CursorVisible:    true,
		CursorBlinkTimer: 0,
		CursorPosition:   len(oldUsername), // 光标在末尾
		MaxLength:        20,
		Placeholder:      "",
		IsFocused:        true,
		TextOffsetX:      0,
		PaddingLeft:      10.0,
		PaddingRight:     10.0,
		PaddingTop:       5.0,
		PaddingBottom:    5.0,
	})

	// 添加 UI 组件标记
	ecs.AddComponent(em, inputEntity, &components.UIComponent{
		State: components.UINormal,
	})

	// 创建对话框按钮
	btnLeftWidth := float64(btnLeftImg.Bounds().Dx())
	btnRightWidth := float64(btnRightImg.Bounds().Dx())
	btnMiddleWidth := 100.0
	btnTotalWidth := btnLeftWidth + btnMiddleWidth + btnRightWidth
	btnHeight := float64(btnLeftImg.Bounds().Dy())

	const btnSpacing = 20.0
	btn1X := dialogWidth/2 - btnTotalWidth - btnSpacing/2
	btn2X := dialogWidth/2 + btnSpacing/2
	btnY := dialogHeight - 65.0

	dialogButtons := []components.DialogButton{
		{
			Label:       "好",
			X:           btn1X,
			Y:           btnY,
			Width:       btnTotalWidth,
			Height:      btnHeight,
			LeftImage:   btnLeftImg,
			MiddleImage: btnMiddleImg,
			RightImage:  btnRightImg,
			MiddleWidth: btnMiddleWidth,
			OnClick: func() {
				inputComp, ok := ecs.GetComponent[*components.TextInputComponent](em, inputEntity)
				if ok && callback != nil {
					callback(RenameUserDialogResult{
						Confirmed: true,
						OldName:   oldUsername,
						NewName:   inputComp.Text,
					})
				}
			},
		},
		{
			Label:       "取消",
			X:           btn2X,
			Y:           btnY,
			Width:       btnTotalWidth,
			Height:      btnHeight,
			LeftImage:   btnLeftImg,
			MiddleImage: btnMiddleImg,
			RightImage:  btnRightImg,
			MiddleWidth: btnMiddleWidth,
			OnClick: func() {
				if callback != nil {
					callback(RenameUserDialogResult{
						Confirmed: false,
						OldName:   oldUsername,
						NewName:   "",
					})
				}
			},
		},
	}

	// 添加对话框组件
	ecs.AddComponent(em, dialogEntity, &components.DialogComponent{
		Title:            "重命名用户",
		Message:          "请输入新的用户名：",
		Buttons:          dialogButtons,
		Parts:            parts,
		IsVisible:        true,
		Width:            dialogWidth,
		Height:           dialogHeight,
		ChildEntities:    []ecs.EntityID{inputEntity}, // 输入框是对话框的子实体
		AutoClose:        false,                       // 需要验证后才关闭，由回调逻辑控制
		HoveredButtonIdx: -1,                          // 初始化为未悬停状态
	})

	// 添加 UI 组件标记
	ecs.AddComponent(em, dialogEntity, &components.UIComponent{
		State: components.UINormal,
	})

	return dialogEntity, inputEntity, nil
}
