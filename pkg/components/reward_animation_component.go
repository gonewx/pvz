package components

// RewardAnimationComponent 管理奖励动画的状态和数据。
// 用于控制卡片包从草坪右侧弹出、等待点击、展开并显示奖励面板的完整流程。
//
// 动画流程（植物/工具奖励 - 5个阶段）：
// 1. appearing (0.3秒): 卡片从草坪右侧随机行弹出，微小上升 + 缩放动画 (0.8 → 1.0)
// 2. waiting: 卡片静止，显示 SeedPacket 粒子效果（光晕 + 向下箭头），等待玩家点击
// 3. expanding (2秒): 点击后触发 Award.xml 粒子特效，卡片放大并移动到屏幕中央上方
// 4. showing: 粒子特效完成后显示新植物介绍面板，等待玩家点击"下一关"或关闭
// 5. closing (0.5秒): 淡出动画，清理实体，返回主菜单或进入下一关
//
// 动画流程（来信奖励 - Story 8.14 - 7个阶段）：
// 1. appearing: 卡包弹出（与工具奖励相同）
// 2. waiting: 等待玩家点击卡包
// 3. expanding: 点击后播放 Starburst 粒子效果
// 4. fadingOut (0.5秒): 画面渐暗到黑色
// 5. fadingIn (0.5秒): 来信面板渐显
// 6. showing: 显示来信面板
// 7. closing: 关闭面板
type RewardAnimationComponent struct {
	// Phase 表示当前动画阶段
	// 常规值："appearing", "waiting", "expanding", "showing", "closing"
	// 来信专属："fadingOut", "fadingIn"（Story 8.14）
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

	// RewardType 奖励类型："plant"、"tool" 或 "note"
	// - 空字符串：自动推断（如果 PlantID 非空则视为 "plant"）
	// - "plant"：植物奖励
	// - "tool"：工具奖励（如铲子）
	// - "note"：僵尸来信奖励（Story 8.14）
	RewardType string

	// PlantID 解锁的植物ID（如 "sunflower"）
	// 当 RewardType="plant" 时使用
	PlantID string

	// ToolID 解锁的工具ID（如 "shovel"）
	// 当 RewardType="tool" 时使用
	ToolID string

	// NoteID 来信ID（如 "zombienote1"）（Story 8.14）
	// 当 RewardType="note" 时使用，对应 ZombieNote{N}.png
	NoteID string

	// ParticleEffect 粒子效果名称（如 "Award" 或 "AwardPickupArrow"）
	// - 空字符串：自动选择（plant → "Award", tool → "AwardPickupArrow", note → "Starburst"）
	// - 非空：使用指定的粒子效果
	ParticleEffect string

	// FadeAlpha 淡入淡出透明度（Story 8.14）
	// - fadingOut 阶段：0.0 → 1.0（画面渐暗）
	// - fadingIn 阶段：0.0 → 1.0（面板渐显）
	FadeAlpha float32
}

