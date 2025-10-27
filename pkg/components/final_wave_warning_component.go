package components

import "github.com/decker502/pvz/pkg/ecs"

// FinalWaveWarningComponent 最后一波提示组件
//
// 用于标记和管理最后一波僵尸来袭时的提示动画实体。
// 该组件遵循 ECS 架构原则，只包含数据，不包含逻辑方法。
//
// Story 11.3: 最后一波僵尸提示动画
type FinalWaveWarningComponent struct {
	// 动画实体引用
	AnimEntity ecs.EntityID // FinalWave.reanim 动画实体

	// 显示控制
	DisplayTime float64 // 显示时长（秒，如 2.5）
	ElapsedTime float64 // 已显示时间（秒）

	// 状态标志
	IsPlaying bool // 是否正在播放
}
