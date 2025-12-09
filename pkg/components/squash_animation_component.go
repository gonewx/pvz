package components

import "github.com/gonewx/pvz/pkg/ecs"

// SquashAnimationComponent 压扁动画组件
// 用于播放僵尸被除草车碾压时的压扁动画
//
// 设计说明：
// - 本组件采用简化实现，手动应用 LawnMoweredZombie.reanim 的 locator 轨道变换
// - 不实现真正的父子层级系统（Transform Hierarchy）
// - 僵尸在压扁过程中保持当前姿势（通常为 anim_idle），不播放其他动画
// - 动画结束后，僵尸切换为 BehaviorZombieDying 并触发粒子效果
//
// FIXME(future): 当前使用手动变换实现压扁效果，
// 未来可重构为通用的父子层级变换系统（Transform Hierarchy）
type SquashAnimationComponent struct {
	// ElapsedTime 已播放时间（秒）
	// 从 0 开始累积，每帧增加 deltaTime
	ElapsedTime float64

	// Duration 动画总时长（秒）
	// 基于 LawnMoweredZombie.reanim 的帧数（8帧）和 FPS（12）
	// 默认值：8 / 12 ≈ 0.667 秒
	Duration float64

	// LocatorFrames locator 轨道的帧数据（从 LawnMoweredZombie.reanim 加载）
	// 每帧包含：X, Y, SkewX, SkewY, ScaleX, ScaleY
	// 这些变换会逐帧应用到僵尸的渲染上
	LocatorFrames []LocatorFrame

	// CurrentFrameIndex 当前帧索引（0 到 len(LocatorFrames)-1）
	// 随着动画进度推进而增加
	CurrentFrameIndex int

	// OriginalPosX 僵尸原始 X 坐标（碾压开始时的世界坐标）
	// 用于计算 locator 变换的基准位置
	OriginalPosX float64

	// OriginalPosY 僵尸原始 Y 坐标
	OriginalPosY float64

	// LawnmowerEntityID 关联的除草车实体 ID
	// 用于跟随除草车移动（僵尸的 X 坐标会同步除草车的移动）
	LawnmowerEntityID ecs.EntityID

	// ParticlesTriggered 是否已触发粒子效果
	// 用于在动画播放中途（压扁开始时）触发头部/手臂掉落粒子，并隐藏对应身体部件
	ParticlesTriggered bool

	// IsCompleted 动画是否已完成
	// 当动画播放到最后一帧时设置为 true
	IsCompleted bool
}

// LocatorFrame locator 轨道的单帧数据
// 对应 LawnMoweredZombie.reanim 的 <track name="locator"> 中的 <t> 元素
type LocatorFrame struct {
	// X 相对位移 X（像素，相对于除草车位置）
	// 僵尸被铲起并向右移动
	X float64

	// Y 相对位移 Y（像素，垂直方向）
	// 僵尸被铲起（向上移动）
	Y float64

	// SkewX 倾斜角度 X（度）
	// 用于旋转僵尸（从站立 0° 到平躺 90°）
	SkewX float64

	// SkewY 倾斜角度 Y（度）
	// 通常与 SkewX 相同，用于实现旋转
	SkewY float64

	// ScaleX X 轴缩放比例
	// 压扁效果的关键：从 1.0 变为 0.263（约 26%）
	ScaleX float64

	// ScaleY Y 轴缩放比例
	// 压扁时 Y 轴略微拉伸（从 1.0 到 1.042）
	ScaleY float64
}

// GetProgress 获取动画进度（0.0 到 1.0）
// 返回:
//   - float64: 动画进度，0.0 表示开始，1.0 表示结束
func (s *SquashAnimationComponent) GetProgress() float64 {
	if s.ElapsedTime < 0 {
		return 0.0
	}
	progress := s.ElapsedTime / s.Duration
	if progress > 1.0 {
		return 1.0
	}
	return progress
}

// IsComplete 判断动画是否播放完成
// 返回:
//   - bool: 如果动画已完成返回 true
func (s *SquashAnimationComponent) IsComplete() bool {
	return s.IsCompleted || s.ElapsedTime >= s.Duration
}

// GetCurrentFrameIndex 根据当前已播放时间计算应该播放的帧索引
// 返回:
//   - int: 帧索引（0 到 len(LocatorFrames)-1），如果动画结束返回最后一帧
//
// Story 10.6 帧索引映射说明：
//   - 8帧动画（索引 0-7），每帧占据动画时长的 1/8
//   - 帧0: progress ∈ [0.000, 0.125)
//   - 帧1: progress ∈ [0.125, 0.250)
//   - ...
//   - 帧7: progress ∈ [0.875, 1.000]（注意：最后一帧包含1.0）
//   - 公式：frameIndex = min(int(progress * frameCount), frameCount - 1)
//     确保 progress=1.0 时不会越界
func (s *SquashAnimationComponent) GetCurrentFrameIndex() int {
	progress := s.GetProgress()
	frameCount := len(s.LocatorFrames)
	if frameCount == 0 {
		return 0
	}

	// 帧索引计算：progress × frameCount，然后限制到最后一帧
	frameIndex := int(progress * float64(frameCount))

	// 确保不越界：当 progress=1.0 时，frameIndex=frameCount，需要钳制到 frameCount-1
	if frameIndex >= frameCount {
		frameIndex = frameCount - 1
	}

	return frameIndex
}
