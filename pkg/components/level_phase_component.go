package components

// LevelPhaseComponent 关卡阶段组件
// Story 19.4: 管理关卡多阶段流程（如铲子教学阶段 → 保龄球阶段）
//
// 此组件用于追踪关卡的当前阶段和转场状态，支持：
// - 多阶段关卡（如 Level 1-5 的铲子教学 + 保龄球）
// - 阶段间转场动画（传送带滑入、红线显示等）
// - 各阶段的 UI 状态控制
type LevelPhaseComponent struct {
	// CurrentPhase 当前阶段编号
	// 1 = 铲子教学阶段（Phase 1: Shovel Tutorial）
	// 2 = 保龄球阶段（Phase 2: Bowling）
	CurrentPhase int

	// PhaseState 阶段状态
	// "active" - 阶段正在进行
	// "transitioning" - 阶段转场中
	// "completed" - 阶段已完成
	PhaseState string

	// TransitionProgress 转场动画进度
	// 范围：0.0（开始）到 1.0（完成）
	TransitionProgress float64

	// TransitionStep 转场序列当前步骤
	// 0 = 未开始
	// 1 = 关闭强引导模��
	// 2 = Dave 对话进行中
	// 3 = 传送带滑入动画
	// 4 = 显示红线
	// 5 = 激活保龄球阶段
	TransitionStep int

	// ConveyorBeltY 传送带当前 Y 位置
	// 用于滑入动画：从屏幕上��（负值）滑入到目标位置
	ConveyorBeltY float64

	// ConveyorBeltVisible 传送带是否可见
	ConveyorBeltVisible bool

	// ShowRedLine 是否显示红线
	// 保龄球阶段的视觉提示，位于第 3 列和第 4 列之间
	ShowRedLine bool

	// DaveDialogueEntityID Dave 对话实体ID（转场对话）
	DaveDialogueEntityID int

	// OnPhaseTransitionComplete 阶段转场完成回调
	// 转场完成后调用，用于激活下一阶段的游戏逻辑
	OnPhaseTransitionComplete func()
}

// PhaseState 常量
const (
	// PhaseStateActive 阶段正在进行
	PhaseStateActive = "active"

	// PhaseStateTransitioning 阶段转场中
	PhaseStateTransitioning = "transitioning"

	// PhaseStateCompleted 阶段已完成
	PhaseStateCompleted = "completed"
)

// TransitionStep 常量
const (
	// TransitionStepNone 未开始
	TransitionStepNone = 0

	// TransitionStepDisableGuided 步骤1: 关闭强引导模式
	TransitionStepDisableGuided = 1

	// TransitionStepDaveDialogue 步骤2: Dave 对话进行中
	TransitionStepDaveDialogue = 2

	// TransitionStepConveyorSlide 步骤3: 传送带滑入动画
	TransitionStepConveyorSlide = 3

	// TransitionStepShowRedLine 步骤4: 显示红线
	TransitionStepShowRedLine = 4

	// TransitionStepActivateBowling 步骤5: 激活保龄球阶段
	TransitionStepActivateBowling = 5
)
