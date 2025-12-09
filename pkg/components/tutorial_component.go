package components

import (
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
)

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

	// Story 8.2.1: 卡片闪烁效果支持（遮罩式闪烁）
	HighlightedCardEntity ecs.EntityID // 被高亮的卡片实体ID（0表示无）
	FlashTimer            float64      // 闪烁计时器（秒）
	FlashCycleDuration    float64      // 闪烁周期（秒）
}
