package components

// RewardAnimationComponent 管理奖励动画的状态和数据。
// 用于控制卡片包从草坪右侧弹出、等待点击、展开并显示奖励面板的完整流程。
//
// 动画流程（5个阶段）：
// 1. appearing (0.3秒): 卡片从草坪右侧随机行弹出，微小上升 + 缩放动画 (0.8 → 1.0)
// 2. waiting: 卡片静止，显示 SeedPacket 粒子效果（光晕 + 向下箭头），等待玩家点击
// 3. expanding (2秒): 点击后触发 Award.xml 粒子特效，卡片放大并移动到屏幕中央上方
// 4. showing: 粒子特效完成后显示新植物介绍面板，等待玩家点击"下一关"或关闭
// 5. closing (0.5秒): 淡出动画，清理实体，返回主菜单或进入下一关
type RewardAnimationComponent struct {
	// Phase 表示当前动画阶段："appearing", "waiting", "expanding", "showing", "closing"
	Phase string

	// ElapsedTime 记录当前阶段已用时间（秒）
	ElapsedTime float64

	// StartX, StartY 起始位置坐标（草坪右侧随机行）
	StartX, StartY float64

	// TargetX, TargetY Phase 3 (expanding) 的目标位置（屏幕中央上方）
	TargetX, TargetY float64

	// Scale 缩放比例
	// - Phase 1 (appearing): 0.8 → 1.0
	// - Phase 3 (expanding): 1.0 → 2.0
	Scale float64

	// PlantID 解锁的植物ID（如 "sunflower"）
	PlantID string
}
