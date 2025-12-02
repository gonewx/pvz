package components

// DaveState 疯狂戴夫的状态枚举
type DaveState int

const (
	// DaveStateHidden Dave 不可见（初始状态或对话结束后）
	DaveStateHidden DaveState = iota

	// DaveStateEntering Dave 从屏幕左侧滑入
	DaveStateEntering

	// DaveStateTalking Dave 正在对话（显示对话气泡）
	DaveStateTalking

	// DaveStateLeaving Dave 向屏幕左侧滑出
	DaveStateLeaving
)

// String 返回 DaveState 的字符串表示
func (s DaveState) String() string {
	switch s {
	case DaveStateHidden:
		return "Hidden"
	case DaveStateEntering:
		return "Entering"
	case DaveStateTalking:
		return "Talking"
	case DaveStateLeaving:
		return "Leaving"
	default:
		return "Unknown"
	}
}

// DaveDialogueComponent 疯狂戴夫对话组件（纯数据，无方法）
//
// 设计目的:
//
//	存储疯狂戴夫对话系统的所有状态数据，包括对话内容、当前状态、表情等。
//	该组件配合 DaveDialogueSystem 实现完整的对话功能。
//
// 使用场景:
//  1. Level 1-5 铲子教学阶段的 Dave 对话
//  2. 其他需要 Dave 出场解说的关卡
//
// 生命周期:
//  1. NewCrazyDaveEntity 创建实体时添加此组件
//  2. DaveDialogueSystem 更新状态和处理交互
//  3. 对话完成后触发回调，实体被销毁
//
// 注意事项:
//   - 组件只包含数据，不包含方法（符合 ECS 数据纯净性原则）
//   - 所有状态转换逻辑在 DaveDialogueSystem 中实现
type DaveDialogueComponent struct {
	// ==========================================================================
	// 对话内容 (Dialogue Content)
	// ==========================================================================

	// DialogueKeys 对话文本 key 列表（从 LawnStrings.txt 加载）
	// 示例: ["CRAZY_DAVE_2400", "CRAZY_DAVE_2401", "CRAZY_DAVE_2402"]
	// 按顺序显示每条对话
	DialogueKeys []string

	// CurrentLineIndex 当前对话行索引（从 0 开始）
	// 每次点击推进到下一条对话时递增
	CurrentLineIndex int

	// CurrentText 当前显示的文本内容（已解析，去除表情指令）
	// 由 DaveDialogueSystem 从 LawnStrings.txt 加载并解析
	CurrentText string

	// CurrentExpressions 当前对话的表情指令列表
	// 示例: ["MOUTH_SMALL_OH", "SCREAM"]
	// 由 DaveDialogueSystem 解析文本时提取
	CurrentExpressions []string

	// ==========================================================================
	// 显示状态 (Display State)
	// ==========================================================================

	// IsVisible 对话气泡是否可见
	// true: 显示对话气泡和文本
	// false: 隐藏对话气泡（如入场动画期间）
	IsVisible bool

	// State Dave 当前状态（Hidden/Entering/Talking/Leaving）
	// 控制 Dave 的行为和动画播放
	State DaveState

	// ==========================================================================
	// 表情与动画 (Expression & Animation)
	// ==========================================================================

	// Expression 当前表情状态
	// 用于切换嘴型轨道图片
	// 可选值: "MOUTH_SMALL_OH", "MOUTH_SMALL_SMILE", "MOUTH_BIG_SMILE", "SCREAM" 等
	// 空字符串表示默认表情
	Expression string

	// ==========================================================================
	// 位置配置 (Position Configuration)
	// ==========================================================================

	// BubbleOffsetX 对话气泡相对于 Dave 位置的 X 偏移（像素）
	// 正值向右偏移
	BubbleOffsetX float64

	// BubbleOffsetY 对话气泡相对于 Dave 位置的 Y 偏移（像素）
	// 负值向上偏移（气泡在 Dave 头顶上方）
	BubbleOffsetY float64

	// ==========================================================================
	// 回调 (Callbacks)
	// ==========================================================================

	// OnCompleteCallback 对话完成回调
	// 当所有对话结束且 Dave 离场动画完成后调用
	// 用于通知其他系统（如 TutorialSystem）对话已完成
	OnCompleteCallback func()
}
