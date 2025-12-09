package entities

import (
	"fmt"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
)

// NewUserDialogResult 新用户对话框的回调结果
type NewUserDialogResult struct {
	Confirmed bool   // 用户是否确认（true=确定，false=取消）
	Username  string // 输入的用户名
}

// NewUserDialogCallback 新用户对话框的回调函数类型
type NewUserDialogCallback func(result NewUserDialogResult)

// NewNewUserDialogEntity 创建"新用户"对话框实体（带文本输入框）
//
// 参数：
//   - em: 实体管理器
//   - rm: 资源管理器
//   - windowWidth, windowHeight: 游戏窗口大小（用于居中）
//   - callback: 用户点击"确定"或"取消"后的回调函数
//
// 返回：
//   - 对话框实体ID
//   - 文本输入框实体ID
//   - 错误信息
func NewNewUserDialogEntity(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	windowWidth, windowHeight int,
	callback NewUserDialogCallback,
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
	const dialogHeight = 280.0 // 增加高度以容纳描述文字和输入框

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
	inputBoxY := 140.0 // 在描述文字下方（标题60 + 消息区域约60 + 间距）

	// 创建输入框实体
	inputEntity := em.CreateEntity()

	// 添加位置组件（绝对位置 = 对话框位置 + 相对位置）
	ecs.AddComponent(em, inputEntity, &components.PositionComponent{
		X: dialogX + inputBoxX,
		Y: dialogY + inputBoxY,
	})

	// 添加输入框组件
	ecs.AddComponent(em, inputEntity, &components.TextInputComponent{
		Text:             "",
		BorderImage:      editboxBorderImg,
		BackgroundImage:  editboxBgImg,
		Width:            inputBoxWidth,
		Height:           inputBoxHeight,
		CursorVisible:    true,
		CursorBlinkTimer: 0,
		CursorPosition:   0,
		MaxLength:        20, // 最多20个字符
		Placeholder:      "",
		IsFocused:        true, // 默认获得焦点
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

	// 创建对话框按钮（确定 + 取消）
	btnLeftWidth := float64(btnLeftImg.Bounds().Dx())
	btnRightWidth := float64(btnRightImg.Bounds().Dx())
	btnMiddleWidth := 100.0
	btnTotalWidth := btnLeftWidth + btnMiddleWidth + btnRightWidth
	btnHeight := float64(btnLeftImg.Bounds().Dy())

	// 两个按钮的间距
	const btnSpacing = 20.0
	btn1X := dialogWidth/2 - btnTotalWidth - btnSpacing/2
	btn2X := dialogWidth/2 + btnSpacing/2
	btnY := dialogHeight - 65.0

	dialogButtons := []components.DialogButton{
		{
			Label:          "好",
			X:              btn1X,
			Y:              btnY,
			Width:          btnTotalWidth,
			Height:         btnHeight,
			LeftImage:      btnLeftImg,
			MiddleImage:    btnMiddleImg,
			RightImage:     btnRightImg,
			MiddleWidth:    btnMiddleWidth,
			ClickSoundID:   "SOUND_BUTTONCLICK", // Story 10.9: 释放时播放
			PressedSoundID: "SOUND_GRAVEBUTTON", // Story 10.9: 按下时播放
			OnClick: func() {
				// 确定回调（需要获取输入框的文本）
				inputComp, ok := ecs.GetComponent[*components.TextInputComponent](em, inputEntity)
				if ok && callback != nil {
					callback(NewUserDialogResult{
						Confirmed: true,
						Username:  inputComp.Text,
					})
				}
			},
		},
		{
			Label:          "取消",
			X:              btn2X,
			Y:              btnY,
			Width:          btnTotalWidth,
			Height:         btnHeight,
			LeftImage:      btnLeftImg,
			MiddleImage:    btnMiddleImg,
			RightImage:     btnRightImg,
			MiddleWidth:    btnMiddleWidth,
			ClickSoundID:   "SOUND_BUTTONCLICK", // Story 10.9: 释放时播放
			PressedSoundID: "SOUND_GRAVEBUTTON", // Story 10.9: 按下时播放
			OnClick: func() {
				// 取消回调
				if callback != nil {
					callback(NewUserDialogResult{
						Confirmed: false,
						Username:  "",
					})
				}
			},
		},
	}

	// 添加对话框组件
	ecs.AddComponent(em, dialogEntity, &components.DialogComponent{
		Title:                "新用户",
		Message:              "请输入你的名字：",
		Buttons:              dialogButtons,
		Parts:                parts,
		IsVisible:            true,
		Width:                dialogWidth,
		Height:               dialogHeight,
		ChildEntities:        []ecs.EntityID{inputEntity}, // 输入框是对话框的子实体
		AutoClose:            false,                       // 不自动关闭，由回调逻辑控制
		Modal:                true,                        // 模态对话框，点击遮罩不关闭
		HoveredButtonIdx:     -1,                          // 初始化为未悬停状态
		PressedButtonIdx:     -1,                          // 初始化为未按下状态
		LastPressedButtonIdx: -1,                          // Story 10.9: 初始化为未按下状态
	})

	// 添加 UI 组件标记
	ecs.AddComponent(em, dialogEntity, &components.UIComponent{
		State: components.UINormal,
	})

	return dialogEntity, inputEntity, nil
}
