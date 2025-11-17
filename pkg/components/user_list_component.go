package components

import "time"

// UserInfo 用户信息（避免循环引用 game 包）
//
// Story 12.4: 用户管理 UI
//
// 存储用户的基本信息
type UserInfo struct {
	Username    string    // 用户名
	CreatedAt   time.Time // 创建时间
	LastLoginAt time.Time // 最后登录时间
}

// UserListComponent 用户列表组件
//
// Story 12.4: 用户管理 UI
//
// 用于用户管理对话框中显示用户列表
type UserListComponent struct {
	// 用户列表
	Users []UserInfo

	// 当前选中的用户索引（-1 表示选中"建立一位新用户"）
	SelectedIndex int

	// 当前登录的用户名（用于高亮显示）
	CurrentUser string

	// 列表渲染配置
	ItemHeight    float64 // 每个列表项的高度（像素）
	VisibleItems  int     // 可见的项目数量（滚动前）
	ScrollOffset  int     // 滚动偏移（从第几个项开始显示）
	MaxScrollRows int     // 最大滚动行数
}

// GetSelectedUsername 获取当前选中的用户名
// 如果选中"建立一位新用户"（即 SelectedIndex == len(Users)），返回空字符串
func (c *UserListComponent) GetSelectedUsername() string {
	if c.SelectedIndex >= 0 && c.SelectedIndex < len(c.Users) {
		return c.Users[c.SelectedIndex].Username
	}
	return "" // "建立一位新用户" 或无效索引
}

// IsNewUserSelected 检查是否选中了"建立一位新用户"
func (c *UserListComponent) IsNewUserSelected() bool {
	return c.SelectedIndex == len(c.Users)
}
