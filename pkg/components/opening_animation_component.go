package components

import "github.com/decker502/pvz/pkg/ecs"

// OpeningAnimationComponent 管理开场动画的状态机。
// 用于控制开场流程：镜头右移 → 展示僵尸预告 → 镜头返回 → 开始游戏。
type OpeningAnimationComponent struct {
	// State 当前状态：
	// - "idle": 待机（初始化）
	// - "cameraMoveRight": 镜头右移到僵尸预告位置
	// - "showZombies": 展示僵尸预告
	// - "cameraMoveLeft": 镜头返回草坪
	// - "gameStart": 游戏开始
	State string

	// ElapsedTime 当前状态已用时间（秒）
	ElapsedTime float64

	// ZombieEntities 预告僵尸实体ID列表（用于清理）
	ZombieEntities []ecs.EntityID

	// IsSkipped 是否被跳过
	IsSkipped bool

	// IsCompleted 是否已完成
	IsCompleted bool
}
