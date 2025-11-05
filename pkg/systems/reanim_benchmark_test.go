package systems

import (
	"testing"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
)

// ==================================================================
// Story 6.6: Performance Benchmark Tests (性能基准测试)
// ==================================================================
//
// 这些基准测试用于验证 AC5 的性能要求：
// - FPS ≥ 60（帧更新时间 < 16.67ms）
// - 渲染时间 < 5ms/frame
// - 内存使用无明显增加
//
// 使用方法：
//   go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof
//   go tool pprof cpu.prof
//   go tool pprof mem.prof
//
// ==================================================================

// BenchmarkReanimUpdate 测试 ReanimSystem.Update() 的性能
//
// 目标：验证帧推进逻辑的性能开销
// 期望：每次 Update 操作 < 1ms（1,000,000 ns）
func BenchmarkReanimUpdate(b *testing.B) {
	// 设置测试环境
	em := ecs.NewEntityManager()
	reanimSys := NewReanimSystem(em)

	// 创建 10 个测试实体（模拟游戏场景中的植物数量）
	entities := make([]ecs.EntityID, 10)
	for i := 0; i < 10; i++ {
		entity := em.CreateEntity()
		entities[i] = entity

		// 创建简单的 Reanim 数据（骨骼动画模式）
		reanimData := &reanim.ReanimXML{
			FPS: 12,
			Tracks: []reanim.Track{
				{
					Name: "part_1",
					Frames: []reanim.Frame{
						{ImagePath: "image1.png"},
						{ImagePath: "image2.png"},
						{ImagePath: "image3.png"},
					},
				},
				{
					Name: "part_2",
					Frames: []reanim.Frame{
						{ImagePath: "image1.png"},
						{ImagePath: "image2.png"},
						{ImagePath: "image3.png"},
					},
				},
			},
		}

		// 添加 ReanimComponent
		ecs.AddComponent(em, entity, &components.ReanimComponent{
			Reanim:            reanimData,
			CurrentAnim:       "anim_idle",
			IsLooping:         true,
			VisibleFrameCount: 3,
		})

		// 播放动画
		_ = reanimSys.PlayAnimation(entity, "anim_idle")
	}

	// 重置计时器（忽略设置时间）
	b.ResetTimer()

	// 基准测试：模拟游戏循环中的 Update 调用
	deltaTime := 1.0 / 60.0 // 60 FPS
	for i := 0; i < b.N; i++ {
		reanimSys.Update(deltaTime)
	}
}

// BenchmarkModeDetection 测试模式检测的性能
//
// 目标：验证 detectPlaybackMode() 不会成为瓶颈
// 期望：单次检测 < 10μs（10,000 ns）
func BenchmarkModeDetection(b *testing.B) {
	// 创建不同复杂度的 Reanim 数据进行测试
	testCases := []struct {
		name string
		data *reanim.ReanimXML
	}{
		{
			name: "Simple_1Track",
			data: &reanim.ReanimXML{
				Tracks: []reanim.Track{
					{Name: "leaf", Frames: []reanim.Frame{{ImagePath: "image.png"}}},
				},
			},
		},
		{
			name: "Skeleton_10Tracks",
			data: &reanim.ReanimXML{
				Tracks: make([]reanim.Track, 10),
			},
		},
		{
			name: "ComplexScene_500Tracks",
			data: &reanim.ReanimXML{
				Tracks: make([]reanim.Track, 500),
			},
		},
	}

	// 为每个测试用例添加图片数据
	for _, tc := range testCases {
		for i := range tc.data.Tracks {
			tc.data.Tracks[i].Name = "track_" + string(rune('A'+(i%26)))
			tc.data.Tracks[i].Frames = []reanim.Frame{{ImagePath: "image.png"}}
		}
	}

	// 运行基准测试
	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = detectPlaybackMode(tc.name, tc.data)
			}
		})
	}
}
