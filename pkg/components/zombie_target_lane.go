package components

// LaneTransitionMode 僵尸行转换模式
//
// Story 8.7: 定义僵尸从非有效行转移到目标有效行时的转换方式
type LaneTransitionMode int

const (
	// TransitionModeGradual 渐变模式 - 僵尸通过Y轴速度平滑移动到目标行（约3秒）
	//
	// 适用场景：
	//   - 需要视觉过渡效果的特殊关卡
	//   - 展示僵尸从一行移动到另一行的动画过程
	//
	// 实现方式：
	//   - 计算到达目标行所需的VY速度（deltaY / 3.0秒）
	//   - 每帧更新Y坐标：Y += VY * deltaTime
	//   - 检测是否到达目标行（容差5像素）
	TransitionModeGradual LaneTransitionMode = iota

	// TransitionModeInstant 瞬间模式 - 僵尸立即调整Y坐标到目标行（无动画）
	//
	// 适用场景：
	//   - 标准关卡（默认模式）
	//   - 僵尸进攻时直接出现在正确行上
	//   - 不需要行间移动过渡效果
	//
	// 实现方式：
	//   - 在僵尸激活时，直接设置 Y = targetY
	//   - 无过渡动画，瞬间完成
	//   - 启动X轴移动（VX = -23.0）
	TransitionModeInstant
)

// ZombieTargetLaneComponent 僵尸目标行组件
//
// 用于跟踪僵尸需要移动到的目标行（有效行）
// 当僵尸在非有效行生成时，它会先在屏幕右侧站位，
// 然后在进攻时（进入屏幕前）移动到目标行
type ZombieTargetLaneComponent struct {
	// TargetRow 目标行索引（0-4，0-based）
	TargetRow int

	// HasReachedTargetLane 是否已到达目标行
	// true 表示僵尸已经在目标行上，可以正常进攻
	// false 表示僵尸还在移动到目标行的过程中
	HasReachedTargetLane bool

	// TransitionMode 行转换模式（Story 8.7 新增）
	//
	// 控制僵尸如何从当前行移动到目标行：
	//   - TransitionModeGradual: 渐变动画（3秒平滑移动）
	//   - TransitionModeInstant: 瞬间调整（无动画）
	//
	// 该字段由 WaveSpawnSystem 在添加组件时设置，
	// 值来自关卡配置文件的 laneTransitionMode 字段
	TransitionMode LaneTransitionMode
}
