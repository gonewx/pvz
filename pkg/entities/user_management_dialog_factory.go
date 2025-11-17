package entities

import (
	"fmt"

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
	const dialogWidth = 600.0
	const dialogHeight = 450.0

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

	// 构建用户列表显示文本
	userListText := "当前用户列表:\n\n"
	for i, user := range users {
		prefix := "  "
		if user.Username == currentUser {
			prefix = "► " // 当前用户标记
		}
		userListText += fmt.Sprintf("%s%d. %s\n", prefix, i+1, user.Username)
	}
	userListText += "\n[点击下方按钮进行操作]"

	// 创建对话框按钮
	btnLeftWidth := float64(btnLeftImg.Bounds().Dx())
	btnRightWidth := float64(btnRightImg.Bounds().Dx())
	btnMiddleWidth := 120.0
	btnTotalWidth := btnLeftWidth + btnMiddleWidth + btnRightWidth
	btnHeight := float64(btnLeftImg.Bounds().Dy())

	// 按钮布局（底部4个按钮：重命名、删除、确定、取消）
	const btnSpacing = 15.0
	btnY := dialogHeight - 65.0

	// 简化版本：只实现"新建用户"和"取消"按钮
	// 完整版本需要更复杂的UI布局
	btn1X := dialogWidth/2 - btnTotalWidth - btnSpacing/2
	btn2X := dialogWidth/2 + btnSpacing/2

	dialogButtons := []components.DialogButton{
		{
			Label:       "建立一位新用户",
			X:           btn1X,
			Y:           btnY,
			Width:       btnTotalWidth,
			Height:      btnHeight,
			LeftImage:   btnLeftImg,
			MiddleImage: btnMiddleImg,
			RightImage:  btnRightImg,
			MiddleWidth: btnMiddleWidth,
			OnClick: func() {
				if callback != nil {
					callback(UserManagementDialogResult{
						Action:         UserActionCreateNew,
						NewUserClicked: true,
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
					callback(UserManagementDialogResult{
						Action: UserActionNone,
					})
				}
			},
		},
	}

	// 添加对话框组件
	ecs.AddComponent(em, dialogEntity, &components.DialogComponent{
		Title:     "你叫啥？",
		Message:   userListText,
		Buttons:   dialogButtons,
		Parts:     parts,
		IsVisible: true,
		Width:     dialogWidth,
		Height:    dialogHeight,
		AutoClose: true, // 用户管理对话框点击后自动关闭
	})

	// 添加 UI 组件标记
	ecs.AddComponent(em, dialogEntity, &components.UIComponent{
		State: components.UINormal,
	})

	// 添加用户列表组件（用于后续扩展）
	userInfoList := make([]components.UserInfo, len(users))
	for i, user := range users {
		userInfoList[i] = components.UserInfo{
			Username:    user.Username,
			CreatedAt:   user.CreatedAt,
			LastLoginAt: user.LastLoginAt,
		}
	}

	ecs.AddComponent(em, dialogEntity, &components.UserListComponent{
		Users:         userInfoList,
		SelectedIndex: 0,
		CurrentUser:   currentUser,
		ItemHeight:    30.0,
		VisibleItems:  5,
		ScrollOffset:  0,
	})

	return dialogEntity, nil
}
