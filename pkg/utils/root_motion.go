// Package utils 提供通用工具函数
package utils

import (
	"fmt"
	"math"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
)

// RootMotionMaxDelta 是瞬移检测阈值
// 当帧间位移超过此值时，认为发生了动画循环重置，返回 0 避免瞬移
//
// 根据 Zombie.reanim 的 _ground 轨道分析：
// - 正常帧间位移: 0.1-2.5 像素（12 FPS 动画）
// - 动画循环重置位移: 49.8 像素（从 X=40 跳回 X=-9.8）
//
// 阈值设置为 20.0 可以有效区分正常移动和循环重置
const RootMotionMaxDelta = 20.0

// CalculateRootMotionDelta 计算根运动位移增量
//
// 从 Reanim 动画的 _ground 轨道读取当前帧与上一帧的位移差值，
// 用于实现僵尸脚步与地面的完美同步，消除"滑步"现象。
//
// 工作原理:
//  1. 获取当前帧的 _ground 轨道 X/Y 坐标
//  2. 与上一帧的坐标进行对比，计算增量
//  3. 检测动画循环重置（防止瞬移）
//  4. 更新 LastGroundX/Y 用于下一次计算
//
// 参数:
//   - reanimComp: Reanim 组件（包含动画数据和当前帧信息）
//   - groundTrackName: _ground 轨道名称（通常为 "_ground"）
//
// 返回:
//   - deltaX: X 轴位移增量（世界坐标单位）
//   - deltaY: Y 轴位移增量
//   - error: 如果轨道不存在或数据无效返回错误
//
// 注意:
//   - 当动画循环重置时（从最后一帧跳回第一帧），自动检测并返回 0（避免瞬移）
//   - 调用方需要在 ReanimComponent 初始化时设置 LastGroundX/Y = 0
//   - 返回的位移增量是负值（僵尸向左移动），需要直接加到 position.X 上
func CalculateRootMotionDelta(
	reanimComp *components.ReanimComponent,
	groundTrackName string,
) (deltaX, deltaY float64, err error) {
	// 参数验证
	if reanimComp == nil {
		return 0, 0, fmt.Errorf("reanimComp is nil")
	}

	if reanimComp.MergedTracks == nil {
		return 0, 0, fmt.Errorf("MergedTracks is nil")
	}

	// 获取 _ground 轨道数据
	groundFrames, ok := reanimComp.MergedTracks[groundTrackName]
	if !ok || len(groundFrames) == 0 {
		return 0, 0, fmt.Errorf("ground track '%s' not found or empty", groundTrackName)
	}

	// 获取当前动画的物理帧索引
	// 需要将逻辑帧（CurrentFrame）转换为物理帧索引

	// 1. 获取当前播放的动画名称
	// 如果没有 CurrentAnimations，回退到直接使用 CurrentFrame（兼容旧代码和测试）
	var frameIndex int

	if len(reanimComp.CurrentAnimations) == 0 {
		// 没有动画信息，直接使用 CurrentFrame 作为物理帧索引（向后兼容）
		frameIndex = reanimComp.CurrentFrame
	} else {
		currentAnimName := reanimComp.CurrentAnimations[0]

		// 2. 获取动画的可见性数组（用于逻辑帧到物理帧的映射）
		animVisibles, ok := reanimComp.AnimVisiblesMap[currentAnimName]

		if !ok || len(animVisibles) == 0 {
			// PlayAllFrames 模式：CurrentFrame 直接是物理帧
			frameIndex = reanimComp.CurrentFrame
		} else {
			// PlayAnimation 模式：需要映射逻辑帧到物理帧
			// 逻辑帧 N = 第 N 个可见段（animVisibles[i] == 0）的物理帧索引
			logicalFrame := reanimComp.CurrentFrame
			visibleCount := 0
			frameIndex = -1

			for i, val := range animVisibles {
				if val == 0 {
					// 这是一个可见帧的开始
					if visibleCount == logicalFrame {
						frameIndex = i
						break
					}
					visibleCount++
				}
			}

			if frameIndex < 0 {
				// 逻辑帧超出范围，使用最后一个可见帧
				for i := len(animVisibles) - 1; i >= 0; i-- {
					if animVisibles[i] == 0 {
						frameIndex = i
						break
					}
				}
				if frameIndex < 0 {
					return 0, 0, fmt.Errorf("no visible frames in animation %s", currentAnimName)
				}
			}
		}
	}

	// 边界检查
	if frameIndex < 0 || frameIndex >= len(groundFrames) {
		return 0, 0, fmt.Errorf("frame index %d out of range (0-%d)", frameIndex, len(groundFrames)-1)
	}

	// 获取当前帧的 _ground 坐标
	currentX, currentY := getGroundPosition(groundFrames, frameIndex)

	// ========== 位移插值逻辑（消除抖动） ==========
	//
	// 问题：游戏 60 FPS，动画 12 FPS，每 5 帧动画帧才变化一次
	// 如果直接在动画帧变化时返回全部位移，会导致"跳跃式"移动（抖动）
	//
	// 解决：
	// 1. 检测动画帧是否变化
	// 2. 如果变化：计算本次动画帧的总位移，存入累积变量
	// 3. 如果未变化：从累积变量中均匀分配位移
	//
	// 假设动画 12 FPS，游戏 60 FPS：
	// - 每个动画帧持续约 5 个游戏帧
	// - 将动画帧的总位移除以 5，每个游戏帧返回 1/5

	// 检测动画帧是否变化
	animFrameChanged := (frameIndex != reanimComp.LastAnimFrame)

	if animFrameChanged {
		// 动画帧发生变化，计算本次动画帧的位移增量
		// 注意：僵尸向左移动，_ground 轨道的 X 值通常是递增的
		// 所以位移增量 = 当前帧 - 上一帧 = 正值
		// 但僵尸需要向左移动，所以需要取负值
		rawDeltaX := -(currentX - reanimComp.LastGroundX)
		rawDeltaY := -(currentY - reanimComp.LastGroundY)

		// 检测瞬移（动画循环重置）
		if math.Abs(rawDeltaX) > RootMotionMaxDelta || math.Abs(rawDeltaY) > RootMotionMaxDelta {
			rawDeltaX, rawDeltaY = 0, 0
		}

		// 计算每帧固定位移（线性均匀分配，避免指数衰减）
		// 策略：假设每个动画帧持续 5 个游戏帧（60 FPS / 12 FPS = 5）
		// 一次性计算每个游戏帧应该分配的固定位移量
		const interpolationFrames = 5.0 // 60 FPS / 12 FPS
		reanimComp.AccumulatedDeltaX = rawDeltaX / interpolationFrames
		reanimComp.AccumulatedDeltaY = rawDeltaY / interpolationFrames

		// 更新 LastGroundX/Y 和 LastAnimFrame
		reanimComp.LastGroundX = currentX
		reanimComp.LastGroundY = currentY
		reanimComp.LastAnimFrame = frameIndex
	}

	// 直接返回预先计算好的每帧固定位移
	// 这样可以保证线性均匀分配，避免指数衰减
	deltaX = reanimComp.AccumulatedDeltaX
	deltaY = reanimComp.AccumulatedDeltaY

	return deltaX, deltaY, nil
}

// getGroundPosition 获取指定帧的 _ground 轨道坐标
//
// 处理 Reanim 的空帧继承机制：如果某帧的 X/Y 为 nil，
// 则继承最近的非空帧的坐标。
//
// 参数:
//   - frames: _ground 轨道的帧数组（已经过 MergedTracks 处理，包含累加值）
//   - frameIndex: 帧索引
//
// 返回:
//   - x: X 坐标
//   - y: Y 坐标
func getGroundPosition(frames []reanim.Frame, frameIndex int) (x, y float64) {
	// 边界检查
	if frameIndex < 0 || frameIndex >= len(frames) {
		return 0, 0
	}

	frame := frames[frameIndex]

	// MergedTracks 已经处理了帧继承，直接读取即可
	// 如果 X/Y 为 nil，说明没有设置值，使用默认值 0
	if frame.X != nil {
		x = *frame.X
	}
	if frame.Y != nil {
		y = *frame.Y
	}

	return x, y
}

// GetGroundTrackFrameCount 获取 _ground 轨道的帧数
//
// 用于调试和验证 _ground 轨道是否存在。
//
// 参数:
//   - reanimComp: Reanim 组件
//   - groundTrackName: _ground 轨道名称
//
// 返回:
//   - frameCount: 帧数，如果轨道不存在返回 0
func GetGroundTrackFrameCount(
	reanimComp *components.ReanimComponent,
	groundTrackName string,
) int {
	if reanimComp == nil || reanimComp.MergedTracks == nil {
		return 0
	}

	groundFrames, ok := reanimComp.MergedTracks[groundTrackName]
	if !ok {
		return 0
	}

	return len(groundFrames)
}
