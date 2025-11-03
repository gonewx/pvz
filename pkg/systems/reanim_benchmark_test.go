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

// BenchmarkComplexSceneRendering 测试复杂场景（500+ 轨道）的渲染性能
//
// 目标：验证 ComplexScenePlaybackStrategy 在 SelectorScreen 等场景的效率
// 期望：GetVisibleTracks 操作 < 100μs（100,000 ns）
func BenchmarkComplexSceneRendering(b *testing.B) {
	// 创建复杂场景 Reanim 数据（模拟 SelectorScreen）
	reanimData := &reanim.ReanimXML{
		FPS:    12,
		Tracks: []reanim.Track{},
	}

	// 创建 500 个轨道（模拟 SelectorScreen 的复杂度）
	for i := 0; i < 500; i++ {
		track := reanim.Track{
			Name: "track_" + string(rune('A'+(i%26))) + string(rune('0'+(i/26))),
			Frames: []reanim.Frame{
				{ImagePath: "element.png"},
				{ImagePath: "element.png"},
				{ImagePath: "element.png"},
			},
		}
		reanimData.Tracks = append(reanimData.Tracks, track)
	}

	// 设置实体
	em := ecs.NewEntityManager()
	reanimSys := NewReanimSystem(em)
	entity := em.CreateEntity()

	ecs.AddComponent(em, entity, &components.ReanimComponent{
		Reanim:            reanimData,
		CurrentAnim:       "anim_idle",
		IsLooping:         true,
		VisibleFrameCount: 3,
	})

	// 播放动画
	_ = reanimSys.PlayAnimation(entity, "anim_idle")

	// 获取组件
	reanimComp, _ := ecs.GetComponent[*components.ReanimComponent](em, entity)

	// 创建策略实例
	strategy := &ComplexScenePlaybackStrategy{}

	// 重置计时器
	b.ResetTimer()

	// 基准测试：模拟每帧调用 GetVisibleTracks
	for i := 0; i < b.N; i++ {
		_ = strategy.GetVisibleTracks(reanimComp, 0)
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
				_ = detectPlaybackMode(tc.data)
			}
		})
	}
}

// BenchmarkBlendedPlaybackStrategy 测试混合模式策略的性能
//
// 目标：验证 BlendedPlaybackStrategy（60% 的植物）的效率
// 期望：GetVisibleTracks 操作 < 50μs（50,000 ns）
func BenchmarkBlendedPlaybackStrategy(b *testing.B) {
	// 创建混合模式 Reanim 数据（模拟豌豆射手）
	frameNum0 := 0
	frameNumMinus1 := -1

	reanimData := &reanim.ReanimXML{
		FPS: 12,
		Tracks: []reanim.Track{
			// 动画定义轨道
			{Name: "anim_idle", Frames: []reanim.Frame{{FrameNum: &frameNum0}}},
			{Name: "anim_shooting", Frames: []reanim.Frame{{FrameNum: &frameNum0}}},
			{Name: "anim_head_idle", Frames: []reanim.Frame{{FrameNum: &frameNum0}}},
			// 部件轨道
			{Name: "head", Frames: []reanim.Frame{{ImagePath: "head.png", FrameNum: &frameNumMinus1}}},
			{Name: "body", Frames: []reanim.Frame{{ImagePath: "body.png", FrameNum: &frameNumMinus1}}},
			{Name: "stalk", Frames: []reanim.Frame{{ImagePath: "stalk.png", FrameNum: &frameNumMinus1}}},
		},
	}

	// 设置实体
	em := ecs.NewEntityManager()
	reanimSys := NewReanimSystem(em)
	entity := em.CreateEntity()

	ecs.AddComponent(em, entity, &components.ReanimComponent{
		Reanim:            reanimData,
		CurrentAnim:       "anim_idle",
		IsLooping:         true,
		VisibleFrameCount: 1,
	})

	// 播放动画
	_ = reanimSys.PlayAnimation(entity, "anim_idle")

	// 获取组件
	reanimComp, _ := ecs.GetComponent[*components.ReanimComponent](em, entity)

	// 创建策略实例
	strategy := &BlendedPlaybackStrategy{}

	// 重置计时器
	b.ResetTimer()

	// 基准测试：模拟每帧调用 GetVisibleTracks
	for i := 0; i < b.N; i++ {
		_ = strategy.GetVisibleTracks(reanimComp, 0)
	}
}

// BenchmarkAllStrategies 测试所有播放策略的性能对比
//
// 目标：对比 5 种播放模式的性能差异
func BenchmarkAllStrategies(b *testing.B) {
	strategies := []struct {
		name     string
		strategy PlaybackStrategy
		comp     *components.ReanimComponent
	}{
		{
			name:     "Simple",
			strategy: &SimplePlaybackStrategy{},
			comp: &components.ReanimComponent{
				MergedTracks: map[string][]reanim.Frame{
					"track1": {{ImagePath: "image.png"}},
				},
			},
		},
		{
			name:     "Skeleton",
			strategy: &SkeletonPlaybackStrategy{},
			comp: &components.ReanimComponent{
				MergedTracks: map[string][]reanim.Frame{
					"head": {{ImagePath: "head.png"}},
					"body": {{ImagePath: "body.png"}},
				},
			},
		},
		{
			name:     "Sequence",
			strategy: &SequencePlaybackStrategy{},
			comp: &components.ReanimComponent{
				MergedTracks: map[string][]reanim.Frame{
					"text": {{ImagePath: "text.png", FrameNum: &[]int{0}[0]}},
				},
			},
		},
		{
			name:     "ComplexScene",
			strategy: &ComplexScenePlaybackStrategy{},
			comp: &components.ReanimComponent{
				VisibleTracks: map[string]bool{
					"track1": true,
					"track2": true,
				},
				MergedTracks: map[string][]reanim.Frame{
					"track1": {{ImagePath: "image1.png"}},
					"track2": {{ImagePath: "image2.png"}},
					"track3": {{ImagePath: "image3.png"}},
				},
			},
		},
		{
			name:     "Blended",
			strategy: &BlendedPlaybackStrategy{},
			comp: &components.ReanimComponent{
				CurrentAnim: "anim_idle",
				AnimVisiblesMap: map[string][]int{
					"anim_idle": {0, 0, 0},
				},
				MergedTracks: map[string][]reanim.Frame{
					"head": {{ImagePath: "head.png", FrameNum: &[]int{-1}[0]}},
					"body": {{ImagePath: "body.png", FrameNum: &[]int{0}[0]}},
				},
			},
		},
	}

	for _, s := range strategies {
		b.Run(s.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = s.strategy.GetVisibleTracks(s.comp, 0)
			}
		})
	}
}
