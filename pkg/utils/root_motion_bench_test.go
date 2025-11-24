package utils

import (
	"testing"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
)

// BenchmarkCalculateRootMotionDelta 基准测试：根运动位移计算
// 模拟正常游戏场景，测试根运动法的性能
func BenchmarkCalculateRootMotionDelta(b *testing.B) {
	// 准备测试数据：模拟 _ground 轨道（100 帧）
	groundFrames := make([]reanim.Frame, 100)
	for i := 0; i < 100; i++ {
		x := float64(i * 5) // 每帧移动 5 像素
		y := 40.0
		groundFrames[i] = reanim.Frame{X: &x, Y: &y}
	}

	comp := &components.ReanimComponent{
		MergedTracks: map[string][]reanim.Frame{
			"_ground": groundFrames,
		},
		CurrentFrame: 0,
		LastGroundX:  0.0,
		LastGroundY:  40.0,
	}

	// 重置计时器
	b.ResetTimer()

	// 执行基准测试
	for i := 0; i < b.N; i++ {
		// 模拟帧推进
		comp.CurrentFrame = i % 99
		comp.LastGroundX = float64((i - 1) % 100 * 5)

		// 执行根运动计算
		CalculateRootMotionDelta(comp, "_ground")
	}
}

// BenchmarkFixedVelocityMethod 基准测试：固定速度法（对比基准）
// 模拟传统的固定速度法，与根运动法进行性能对比
func BenchmarkFixedVelocityMethod(b *testing.B) {
	// 固定速度法的计算非常简单
	velocityX := -150.0 // 僵尸向左移动
	velocityY := 0.0
	deltaTime := 1.0 / 60.0 // 假设 60 FPS
	positionX := 800.0
	positionY := 300.0

	// 重置计时器
	b.ResetTimer()

	// 执行基准测试
	for i := 0; i < b.N; i++ {
		// 模拟位置更新
		positionX += velocityX * deltaTime
		positionY += velocityY * deltaTime

		// 重置位置（防止溢出）
		if positionX < 0 {
			positionX = 800.0
		}
	}

	// 使用变量防止编译器优化
	_ = positionX
	_ = positionY
}

// BenchmarkCalculateRootMotionDelta_Parallel 基准测试：并行根运动计算
// 模拟多僵尸同屏场景
func BenchmarkCalculateRootMotionDelta_Parallel(b *testing.B) {
	// 准备测试数据
	groundFrames := make([]reanim.Frame, 100)
	for i := 0; i < 100; i++ {
		x := float64(i * 5)
		y := 40.0
		groundFrames[i] = reanim.Frame{X: &x, Y: &y}
	}

	// 创建 10 个僵尸的组件（模拟多僵尸同屏）
	zombies := make([]*components.ReanimComponent, 10)
	for j := 0; j < 10; j++ {
		zombies[j] = &components.ReanimComponent{
			MergedTracks: map[string][]reanim.Frame{
				"_ground": groundFrames,
			},
			CurrentFrame: j * 10 % 100,
			LastGroundX:  float64(j * 50),
			LastGroundY:  40.0,
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, zombie := range zombies {
			// 模拟帧推进
			zombie.CurrentFrame = (zombie.CurrentFrame + 1) % 99
			zombie.LastGroundX = float64(zombie.CurrentFrame * 5)

			CalculateRootMotionDelta(zombie, "_ground")
		}
	}
}
