package entities

import (
	"fmt"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// DeleteUserDialogResult 删除用户对话框的回调结果
type DeleteUserDialogResult struct {
	Confirmed bool   // 用户是否确认删除
	Username  string // 待删除的用户名
}

// DeleteUserDialogCallback 删除用户对话框的回调函数类型
type DeleteUserDialogCallback func(result DeleteUserDialogResult)

// NewDeleteUserDialogEntity 创建删除用户确认对话框实体
//
// 参数：
//   - em: 实体管理器
//   - rm: 资源管理器
//   - username: 待删除的用户名
//   - windowWidth, windowHeight: 游戏窗口大小
//   - callback: 操作回调函数
//
// 返回：
//   - 对话框实体ID
//   - 错误信息
func NewDeleteUserDialogEntity(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	username string,
	windowWidth, windowHeight int,
	callback DeleteUserDialogCallback,
) (ecs.EntityID, error) {
	// 加载九宫格对话框资源
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

	// 对话框尺寸
	const dialogWidth = 450.0
	const dialogHeight = 200.0

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

	// 创建对话框按钮
	btnLeftWidth := float64(btnLeftImg.Bounds().Dx())
	btnRightWidth := float64(btnRightImg.Bounds().Dx())
	btnMiddleWidth := 80.0
	btnTotalWidth := btnLeftWidth + btnMiddleWidth + btnRightWidth
	btnHeight := float64(btnLeftImg.Bounds().Dy())

	const btnSpacing = 20.0
	btn1X := dialogWidth/2 - btnTotalWidth - btnSpacing/2
	btn2X := dialogWidth/2 + btnSpacing/2
	btnY := dialogHeight - 65.0

	dialogButtons := []components.DialogButton{
		{
			Label:          "否",
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
				if callback != nil {
					callback(DeleteUserDialogResult{
						Confirmed: false,
						Username:  username,
					})
				}
			},
		},
		{
			Label:          "是",
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
				if callback != nil {
					callback(DeleteUserDialogResult{
						Confirmed: true,
						Username:  username,
					})
				}
			},
		},
	}

	// 构建确认消息
	message := fmt.Sprintf("从玩家簿中永久删除 '%s'!", username)

	// 添加对话框组件
	ecs.AddComponent(em, dialogEntity, &components.DialogComponent{
		Title:                "你确定吗？",
		Message:              message,
		Buttons:              dialogButtons,
		Parts:                parts,
		IsVisible:            true,
		Width:                dialogWidth,
		Height:               dialogHeight,
		AutoClose:            true, // 删除确认对话框点击后自动关闭
		Modal:                true, // 模态对话框，点击遮罩不关闭
		HoveredButtonIdx:     -1,   // 初始化为未悬停状态
		PressedButtonIdx:     -1,   // 初始化为未按下状态
		LastPressedButtonIdx: -1,   // Story 10.9: 初始化为未按下状态
	})

	// 添加 UI 组件标记
	ecs.AddComponent(em, dialogEntity, &components.UIComponent{
		State: components.UINormal,
	})

	return dialogEntity, nil
}
