package components

import "github.com/decker502/pvz/pkg/config"

// TutorialComponent 教学系统组件
// 存储教学引导的运行时状态，跟踪当前步骤和完成进度
// 用于关卡 1-1 的分步教学引导系统
type TutorialComponent struct {
	// CurrentStepIndex 当前教学步骤索引（从0开始）
	CurrentStepIndex int

	// CompletedSteps 已完成的步骤触发器ID映射
	// 例如：{"gameStart": true, "sunClicked": true}
	CompletedSteps map[string]bool

	// IsActive 教学系统是否激活
	// false 表示教学已完成或被禁用
	IsActive bool

	// TutorialSteps 教学步骤配置（从 LevelConfig 复制）
	// 包含触发条件、文本键和动作定义
	TutorialSteps []config.TutorialStep
}
