package entities

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// UserManagementAction 用户管理对话框的操作类型
type UserManagementAction int

const (
	UserActionNone UserManagementAction = iota
	UserActionRename
	UserActionDelete
	UserActionSwitch
	UserActionCreateNew
)

// UserManagementDialogResult 用户管理对话框的回调结果
type UserManagementDialogResult struct {
	Action         UserManagementAction // 用户执行的操作
	SelectedUser   string               // 选中的用户名
	NewUserClicked bool                 // 是否点击了"建立一位新用户"
}

// UserManagementDialogCallback 用户管理对话框的回调函数类型
type UserManagementDialogCallback func(result UserManagementDialogResult)

// NewUserManagementDialogEntity 创建用户管理对话框实体
//
// 这是一个简化实现：显示用户列表 + 操作按钮
//
// 参数：
//   - em: 实体管理器
//   - rm: 资源管理器
//   - users: 用户列表
//   - currentUser: 当前用户名
//   - windowWidth, windowHeight: 游戏窗口大小
//   - callback: 操作回调函数
//
// 返回：
//   - 对话框实体ID
//   - 错误信息
func NewUserManagementDialogEntity(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	users []game.UserMetadata,
	currentUser string,
	windowWidth, windowHeight int,
	callback UserManagementDialogCallback,
) (ecs.EntityID, error) {
	// 加载九宫格对话框资源
	parts, err := loadDialogParts(rm)
	if err != nil {
		return 0, fmt.Errorf("加载对话框资源失败: %w", err)
	}

	// 用户管理对话框需要更高的底部区域以容纳 2 排按钮
	// 将 BigBottom 图片覆盖到 Bottom 字段，由应用层决定使用哪种 UI 表现
	if parts.BigBottomLeft != nil {
		parts.BottomLeft = parts.BigBottomLeft
	}
	if parts.BigBottomMiddle != nil {
		parts.BottomMiddle = parts.BigBottomMiddle
	}
	if parts.BigBottomRight != nil {
		parts.BottomRight = parts.BigBottomRight
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
	const dialogWidth = 510.0
	const dialogHeight = 400.0

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

	// 构建用户列表显示文本（已移除，由 UserListComponent 渲染）
	// userListText := ""

	// 创建对话框按钮（4 个石板风格按钮）
	btnLeftWidth := float64(btnLeftImg.Bounds().Dx())
	btnRightWidth := float64(btnRightImg.Bounds().Dx())
	btnMiddleWidth := 150.0 // 缩小按钮宽度以适应4个按钮
	btnTotalWidth := btnLeftWidth + btnMiddleWidth + btnRightWidth
	btnHeight := float64(btnLeftImg.Bounds().Dy())

	// 按钮布局（底部4个按钮：重命名、删除、好、取消）
	const btnSpacing = 5.0
	const btnBottomMargin = -22.0 // 按钮区域距离底部的边距

	// 按钮横向布局（2x2网格）
	// 第一行（上排）：重命名（左上）、删除（右上）
	// 第二行（下排）：好（左下）、取消（右下）
	btnRow2Y := dialogHeight - btnBottomMargin - btnHeight // 下排按钮 Y 坐标
	btnRow1Y := btnRow2Y - btnHeight - btnSpacing          // 上排按钮 Y 坐标（给2排按钮留足空间）

	btnCol1X := dialogWidth/2 - btnTotalWidth - btnSpacing/2 // 左列按钮 X 坐标
	btnCol2X := dialogWidth/2 + btnSpacing/2                 // 右列按钮 X 坐标

	dialogButtons := []components.DialogButton{
		// 左上：重命名
		{
			Label:          "重命名",
			X:              btnCol1X,
			Y:              btnRow1Y,
			Width:          btnTotalWidth,
			Height:         btnHeight,
			LeftImage:      btnLeftImg,
			MiddleImage:    btnMiddleImg,
			RightImage:     btnRightImg,
			MiddleWidth:    btnMiddleWidth,
			ClickSoundID:   "SOUND_BUTTONCLICK",  // Story 10.9: 释放时播放
			PressedSoundID: "SOUND_GRAVEBUTTON",  // Story 10.9: 按下时播放
			OnClick: func() {
				if callback != nil {
					callback(UserManagementDialogResult{
						Action:       UserActionRename,
						SelectedUser: "", // 主菜单场景会从 UserListComponent 读取
					})
				}
			},
		},
		// 右上：删除
		{
			Label:          "删除",
			X:              btnCol2X,
			Y:              btnRow1Y,
			Width:          btnTotalWidth,
			Height:         btnHeight,
			LeftImage:      btnLeftImg,
			MiddleImage:    btnMiddleImg,
			RightImage:     btnRightImg,
			MiddleWidth:    btnMiddleWidth,
			ClickSoundID:   "SOUND_BUTTONCLICK",  // Story 10.9: 释放时播放
			PressedSoundID: "SOUND_GRAVEBUTTON",  // Story 10.9: 按下时播放
			OnClick: func() {
				if callback != nil {
					callback(UserManagementDialogResult{
						Action:       UserActionDelete,
						SelectedUser: "", // 主菜单场景会从 UserListComponent 读取
					})
				}
			},
		},
		// 左下：好
		{
			Label:          "好",
			X:              btnCol1X,
			Y:              btnRow2Y,
			Width:          btnTotalWidth,
			Height:         btnHeight,
			LeftImage:      btnLeftImg,
			MiddleImage:    btnMiddleImg,
			RightImage:     btnRightImg,
			MiddleWidth:    btnMiddleWidth,
			ClickSoundID:   "SOUND_BUTTONCLICK",  // Story 10.9: 释放时播放
			PressedSoundID: "SOUND_GRAVEBUTTON",  // Story 10.9: 按下时播放
			OnClick: func() {
				if callback != nil {
					callback(UserManagementDialogResult{
						Action:       UserActionSwitch,
						SelectedUser: "", // 主菜单场景会从 UserListComponent 读取
					})
				}
			},
		},
		// 右下：取消
		{
			Label:          "取消",
			X:              btnCol2X,
			Y:              btnRow2Y,
			Width:          btnTotalWidth,
			Height:         btnHeight,
			LeftImage:      btnLeftImg,
			MiddleImage:    btnMiddleImg,
			RightImage:     btnRightImg,
			MiddleWidth:    btnMiddleWidth,
			ClickSoundID:   "SOUND_BUTTONCLICK",  // Story 10.9: 释放时播放
			PressedSoundID: "SOUND_GRAVEBUTTON",  // Story 10.9: 按下时播放
			OnClick: func() {
				log.Printf("[UserManagementDialog] 取消按钮被点击")
				if callback != nil {
					callback(UserManagementDialogResult{
						Action: UserActionNone,
					})
				}
			},
		},
	}

	// 添加对话框组件
	ecs.AddComponent(em, dialogEntity, &components.DialogComponent{
		Title:            "你叫啥？",
		Message:          "", // 用户列表由 UserListComponent 渲染，不需要 Message
		Buttons:          dialogButtons,
		Parts:            parts,
		IsVisible:        true,
		Width:            dialogWidth,
		Height:           dialogHeight,
		AutoClose:            false, // 用户管理对话框不自动关闭（需要显式关闭）
		HoveredButtonIdx:     -1,    // 初始化为未悬停状态
		PressedButtonIdx:     -1,    // 初始化为未按下状态
		LastPressedButtonIdx: -1,    // Story 10.9: 初始化为未按下状态
	})

	// 添加 UI 组件标记
	ecs.AddComponent(em, dialogEntity, &components.UIComponent{
		State: components.UINormal,
	})

	// 添加用户列表组件
	userInfoList := make([]components.UserInfo, len(users))
	selectedIndex := 0 // 默认选中第一个用户
	for i, user := range users {
		userInfoList[i] = components.UserInfo{
			Username:    user.Username,
			CreatedAt:   user.CreatedAt,
			LastLoginAt: user.LastLoginAt,
		}
		// 找到当前用户的索引
		if user.Username == currentUser {
			selectedIndex = i
		}
	}

	ecs.AddComponent(em, dialogEntity, &components.UserListComponent{
		Users:         userInfoList,
		SelectedIndex: selectedIndex, // 初始化为当前用户
		HoveredIndex:  -1,            // 初始化为未悬停
		CurrentUser:   currentUser,
		ItemHeight:    30.0,
		VisibleItems:  5,
		ScrollOffset:  0,
	})

	return dialogEntity, nil
}
