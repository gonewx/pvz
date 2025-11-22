package components

// MenuButtonComponent 菜单按钮标记组件
// 用于标识游戏场景中的菜单按钮（右上角的"菜单"按钮）
//
// 用途：
//   - 在游戏冻结期间隐藏菜单按钮（Story 8.8 - Task 6）
//   - 与 ButtonComponent 配合使用
//
// 设计原则：
//   - 纯标记组件，不包含任何数据
//   - 用于区分菜单按钮和其他类型的按钮（对话框按钮、暂停菜单按钮等）
type MenuButtonComponent struct {
	// 标记组件，无需字段
}
