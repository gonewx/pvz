package entities

import (
	"fmt"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// GameOverDialogCallback 游戏结束对话框的回调函数类型
type GameOverDialogCallback func()

// NewGameOverDialogEntity 创建游戏结束对话框实体
//
// 参数：
//   - em: 实体管理器
//   - rm: 资源管理器
//   - windowWidth, windowHeight: 游戏窗口大小
//   - onRetry: "再次尝试"按钮回调
//   - onMenu: "返回主菜单"按钮回调（可选，为 nil 时只显示"再次尝试"按钮）
//
// 返回：
//   - 对话框实体ID
//   - 错误信息
func NewGameOverDialogEntity(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	windowWidth, windowHeight int,
	onRetry GameOverDialogCallback,
	onMenu GameOverDialogCallback,
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
	const dialogWidth = 300.0
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
	btnMiddleWidth := 150.0 // 按钮中间可拉伸宽度
	btnTotalWidth := btnLeftWidth + btnMiddleWidth + btnRightWidth
	btnHeight := float64(btnLeftImg.Bounds().Dy())

	// 根据是否有"返回主菜单"按钮决定按钮布局
	var dialogButtons []components.DialogButton

	if onMenu != nil {
		// 双按钮布局（水平排列）
		const btnSpacing = 20.0
		btn1X := dialogWidth/2 - btnTotalWidth - btnSpacing/2
		btn2X := dialogWidth/2 + btnSpacing/2
		btnY := dialogHeight - 65.0

		dialogButtons = []components.DialogButton{
			{
				Label:          "再次尝试",
				X:              btn1X,
				Y:              btnY,
				Width:          btnTotalWidth,
				Height:         btnHeight,
				LeftImage:      btnLeftImg,
				MiddleImage:    btnMiddleImg,
				RightImage:     btnRightImg,
				MiddleWidth:    btnMiddleWidth,
				ClickSoundID:   "SOUND_BUTTONCLICK",  // Story 10.9: 释放时播放
				PressedSoundID: "SOUND_GRAVEBUTTON",  // Story 10.9: 按下时播放
				OnClick: func() {
					if onRetry != nil {
						onRetry()
					}
				},
			},
			{
				Label:          "返回主菜单",
				X:              btn2X,
				Y:              btnY,
				Width:          btnTotalWidth,
				Height:         btnHeight,
				LeftImage:      btnLeftImg,
				MiddleImage:    btnMiddleImg,
				RightImage:     btnRightImg,
				MiddleWidth:    btnMiddleWidth,
				ClickSoundID:   "SOUND_BUTTONCLICK",  // Story 10.9: 释放时播放
				PressedSoundID: "SOUND_GRAVEBUTTON",  // Story 10.9: 按下时播放
				OnClick: func() {
					if onMenu != nil {
						onMenu()
					}
				},
			},
		}
	} else {
		// 单按钮布局（居中）
		btnX := (dialogWidth - btnTotalWidth) / 2
		btnY := dialogHeight - 65.0

		dialogButtons = []components.DialogButton{
			{
				Label:          "再次尝试",
				X:              btnX,
				Y:              btnY,
				Width:          btnTotalWidth,
				Height:         btnHeight,
				LeftImage:      btnLeftImg,
				MiddleImage:    btnMiddleImg,
				RightImage:     btnRightImg,
				MiddleWidth:    btnMiddleWidth,
				ClickSoundID:   "SOUND_BUTTONCLICK",  // Story 10.9: 释放时播放
				PressedSoundID: "SOUND_GRAVEBUTTON",  // Story 10.9: 按下时播放
				OnClick: func() {
					if onRetry != nil {
						onRetry()
					}
				},
			},
		}
	}

	// 添加对话框组件
	ecs.AddComponent(em, dialogEntity, &components.DialogComponent{
		Title:            "游戏结束",
		Message:          "", // 无描述文字
		Buttons:          dialogButtons,
		Parts:            parts,
		IsVisible:        true,
		Width:            dialogWidth,
		Height:           dialogHeight,
		AutoClose:            true, // 点击按钮后自动关闭
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
