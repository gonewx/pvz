package components

import "github.com/gonewx/pvz/pkg/ecs"

// ChomperState 大嘴花状态枚举
// Story 8.11: 定义大嘴花的各种行为状态
type ChomperState int

const (
	// ChomperStateIdle 待机状态
	// 播放 anim_idle 动画，检测前方僵尸
	ChomperStateIdle ChomperState = iota

	// ChomperStateAttacking 攻击准备状态
	// 检测到僵尸，准备执行攻击
	ChomperStateAttacking

	// ChomperStateSwallowing 吞噬状态
	// 播放 anim_swallow 动画，对目标造成 1800 点伤害
	ChomperStateSwallowing

	// ChomperStateBiting 撕咬状态
	// 播放 anim_bite 动画，对无法吞噬的目标造成 40 点伤害
	ChomperStateBiting

	// ChomperStateChewing 消化状态
	// 播放 anim_chew 动画，持续 42 秒，期间无法攻击
	ChomperStateChewing
)

// ChomperComponent 大嘴花特有组件
// Story 8.11: 定义大嘴花的吞噬、消化等行为状态
//
// 大嘴花的特殊机制：
// - 检测前方 2 格内的僵尸
// - 对普通僵尸执行吞噬攻击（1800 伤害，秒杀）
// - 对巨人僵尸等无法吞噬的目标执行撕咬攻击（40 伤害/次）
// - 吞噬后进入 42 秒消化期，期间无法攻击
// - 消化完成后恢复待机状态
type ChomperComponent struct {
	// State 当前状态
	// Idle: 待机，检测僵尸
	// Attacking: 攻击准备
	// Swallowing: 吞噬中
	// Biting: 撕咬中
	// Chewing: 消化中
	State ChomperState

	// TargetZombie 当前目标僵尸实体
	// 用于追踪吞噬/撕咬的目标
	TargetZombie ecs.EntityID

	// DigestTimer 消化计时器（秒）
	// 吞噬后开始计时，达到 42 秒后恢复待机状态
	DigestTimer float64

	// AttackCooldown 攻击冷却计时器（秒）
	// 用于控制撕咬攻击的间隔
	AttackCooldown float64

	// DamageDealt 本次攻击是否已造成伤害
	// 用于确保每次攻击动画只造成一次伤害
	DamageDealt bool
}
