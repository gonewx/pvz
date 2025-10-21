package components

// RewardCardComponent 标记组件，用于区分奖励动画中的植物卡片
// 与选择栏的植物卡片分别渲染，避免渲染冲突
//
// 用途：
//   - RewardAnimationSystem 创建的奖励卡片带有此标记
//   - PlantCardRenderSystem 可以根据此标记过滤不同类型的卡片
//
// 设计原因：
//   - 验证工具中同时存在"选择栏卡片"和"奖励卡片"
//   - 它们需要在不同的渲染层级绘制
//   - 使用此标记可以让不同的 PlantCardRenderSystem 实例只渲染自己负责的卡片
type RewardCardComponent struct {
	// 空结构体，仅用作标记
}
