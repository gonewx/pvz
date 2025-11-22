package components

// GameFreezeComponent 游戏冻结组件
//
// 标记游戏进入冻结状态（僵尸获胜流程期间）
//
// 系统行为：
// - BehaviorSystem: 检测到此组件时，停止更新植物攻击逻辑
// - PhysicsSystem: 停止子弹移动（可选：直接删除子弹实体）
// - UISystem: 隐藏植物选择栏、菜单按钮、进度条
// - RenderSystem: 检测此组件，隐藏 UI 元素
//
// 设计原则（ECS零耦合）：
// - 系统通过查询此组件决定是否更新
// - 避免全局标志位，符合 ECS 架构
type GameFreezeComponent struct {
	IsFrozen bool // 是否已冻结（防止重复冻结）
}
