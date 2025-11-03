package systems

import (
	"testing"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
)

// ==================================================================
// Story 6.6: Playback Mode Detection Tests (播放模式检测测试)
// ==================================================================

// TestDetectPlaybackMode_Simple 测试简单动画模式检测
func TestDetectPlaybackMode_Simple(t *testing.T) {
	// 简单动画：1-2 个轨道，无 f 值
	reanimData := &reanim.ReanimXML{
		Tracks: []reanim.Track{
			{
				Name: "leaf",
				Frames: []reanim.Frame{
					{ImagePath: "image1.png"},
					{ImagePath: "image2.png"},
				},
			},
		},
	}

	mode := detectPlaybackMode(reanimData)

	if mode != ModeSimple {
		t.Errorf("Expected ModeSimple, got %v", mode)
	}
}

// TestDetectPlaybackMode_Skeleton 测试骨骼动画模式检测
func TestDetectPlaybackMode_Skeleton(t *testing.T) {
	// 骨骼动画：4-30 个轨道，无动画定义轨道
	reanimData := &reanim.ReanimXML{
		Tracks: []reanim.Track{
			{
				Name: "head",
				Frames: []reanim.Frame{
					{ImagePath: "head1.png"},
				},
			},
			{
				Name: "body",
				Frames: []reanim.Frame{
					{ImagePath: "body1.png"},
				},
			},
			{
				Name: "arm_left",
				Frames: []reanim.Frame{
					{ImagePath: "arm1.png"},
				},
			},
			{
				Name: "arm_right",
				Frames: []reanim.Frame{
					{ImagePath: "arm2.png"},
				},
			},
		},
	}

	mode := detectPlaybackMode(reanimData)

	if mode != ModeSkeleton {
		t.Errorf("Expected ModeSkeleton, got %v", mode)
	}
}

// TestDetectPlaybackMode_Sequence 测试序列动画模式检测
func TestDetectPlaybackMode_Sequence(t *testing.T) {
	// 序列动画：10-30 个轨道，有 f 值
	frameNumMinus1 := -1
	frameNum0 := 0

	reanimData := &reanim.ReanimXML{
		Tracks: []reanim.Track{},
	}

	// 创建 15 个轨道（满足序列动画的轨道数要求）
	for i := 0; i < 15; i++ {
		track := reanim.Track{
			Name: "track_" + string(rune('A'+i)),
			Frames: []reanim.Frame{
				{FrameNum: &frameNumMinus1, ImagePath: "image1.png"},
				{FrameNum: &frameNum0, ImagePath: "image2.png"},
			},
		}
		reanimData.Tracks = append(reanimData.Tracks, track)
	}

	mode := detectPlaybackMode(reanimData)

	if mode != ModeSequence {
		t.Errorf("Expected ModeSequence, got %v", mode)
	}
}

// TestDetectPlaybackMode_ComplexScene 测试复杂场景动画模式检测
func TestDetectPlaybackMode_ComplexScene(t *testing.T) {
	// 复杂场景动画：50+ 轨道
	reanimData := &reanim.ReanimXML{
		Tracks: []reanim.Track{},
	}

	// 创建 60 个轨道
	for i := 0; i < 60; i++ {
		track := reanim.Track{
			Name: "complex_track_" + string(rune('0'+i%10)),
			Frames: []reanim.Frame{
				{ImagePath: "image.png"},
			},
		}
		reanimData.Tracks = append(reanimData.Tracks, track)
	}

	mode := detectPlaybackMode(reanimData)

	if mode != ModeComplexScene {
		t.Errorf("Expected ModeComplexScene, got %v", mode)
	}
}

// TestDetectPlaybackMode_Blended 测试混合模式检测
func TestDetectPlaybackMode_Blended(t *testing.T) {
	// 混合模式：3+ 动画定义轨道 + 3+ 部件轨道
	frameNum0 := 0
	frameNumMinus1 := -1

	reanimData := &reanim.ReanimXML{
		Tracks: []reanim.Track{
			// 动画定义轨道（只有 f 值，无图片）
			{
				Name: "anim_idle",
				Frames: []reanim.Frame{
					{FrameNum: &frameNum0},
					{FrameNum: &frameNum0},
				},
			},
			{
				Name: "anim_shooting",
				Frames: []reanim.Frame{
					{FrameNum: &frameNumMinus1},
					{FrameNum: &frameNum0},
				},
			},
			{
				Name: "anim_head_idle",
				Frames: []reanim.Frame{
					{FrameNum: &frameNum0},
				},
			},
			// 部件轨道（有图片）
			{
				Name: "head",
				Frames: []reanim.Frame{
					{ImagePath: "head.png", FrameNum: &frameNumMinus1},
				},
			},
			{
				Name: "body",
				Frames: []reanim.Frame{
					{ImagePath: "body.png", FrameNum: &frameNumMinus1},
				},
			},
			{
				Name: "arm",
				Frames: []reanim.Frame{
					{ImagePath: "arm.png", FrameNum: &frameNumMinus1},
				},
			},
		},
	}

	mode := detectPlaybackMode(reanimData)

	if mode != ModeBlended {
		t.Errorf("Expected ModeBlended, got %v", mode)
	}
}

// TestDetectPlaybackMode_BlendedByAnimDefCount 测试通过动画定义轨道数量检测混合模式
func TestDetectPlaybackMode_BlendedByAnimDefCount(t *testing.T) {
	// 混合模式：2+ 动画定义轨道
	frameNum0 := 0

	reanimData := &reanim.ReanimXML{
		Tracks: []reanim.Track{
			// 动画定义轨道
			{
				Name: "anim_idle",
				Frames: []reanim.Frame{
					{FrameNum: &frameNum0},
				},
			},
			{
				Name: "anim_shooting",
				Frames: []reanim.Frame{
					{FrameNum: &frameNum0},
				},
			},
			// 部件轨道
			{
				Name: "head",
				Frames: []reanim.Frame{
					{ImagePath: "head.png"},
				},
			},
		},
	}

	mode := detectPlaybackMode(reanimData)

	if mode != ModeBlended {
		t.Errorf("Expected ModeBlended, got %v", mode)
	}
}

// ==================================================================
// Playback Strategy Tests (播放策略测试)
// ==================================================================

// TestSimplePlaybackStrategy_GetVisibleTracks 测试简单播放策略
func TestSimplePlaybackStrategy_GetVisibleTracks(t *testing.T) {
	strategy := &SimplePlaybackStrategy{}

	comp := &components.ReanimComponent{
		MergedTracks: map[string][]reanim.Frame{
			"track1": {
				{ImagePath: "image1.png"},
			},
			"track2": {
				{ImagePath: "image2.png"},
			},
			"empty_track": {
				{ImagePath: ""}, // 无图片
			},
		},
	}

	visible := strategy.GetVisibleTracks(comp, 0)

	// 应该显示有图片的轨道
	if !visible["track1"] {
		t.Error("track1 should be visible")
	}
	if !visible["track2"] {
		t.Error("track2 should be visible")
	}
	if visible["empty_track"] {
		t.Error("empty_track should not be visible")
	}
}

// TestSkeletonPlaybackStrategy_GetVisibleTracks 测试骨骼播放策略
func TestSkeletonPlaybackStrategy_GetVisibleTracks(t *testing.T) {
	strategy := &SkeletonPlaybackStrategy{}

	comp := &components.ReanimComponent{
		MergedTracks: map[string][]reanim.Frame{
			"head": {
				{ImagePath: "head.png"},
			},
			"body": {
				{ImagePath: "body.png"},
			},
		},
	}

	visible := strategy.GetVisibleTracks(comp, 0)

	// 所有部件都应该可见
	if !visible["head"] {
		t.Error("head should be visible")
	}
	if !visible["body"] {
		t.Error("body should be visible")
	}
}

// TestSequencePlaybackStrategy_GetVisibleTracks 测试序列播放策略
func TestSequencePlaybackStrategy_GetVisibleTracks(t *testing.T) {
	strategy := &SequencePlaybackStrategy{}

	frameNumMinus1 := -1
	frameNum0 := 0

	comp := &components.ReanimComponent{
		MergedTracks: map[string][]reanim.Frame{
			"text_ready": {
				{ImagePath: "ready.png", FrameNum: &frameNum0},
				{ImagePath: "ready.png", FrameNum: &frameNumMinus1},
			},
			"text_set": {
				{ImagePath: "set.png", FrameNum: &frameNumMinus1},
				{ImagePath: "set.png", FrameNum: &frameNum0},
			},
		},
	}

	// 帧 0：text_ready 显示（f=0），text_set 隐藏（f=-1）
	visible0 := strategy.GetVisibleTracks(comp, 0)
	if !visible0["text_ready"] {
		t.Error("text_ready should be visible at frame 0")
	}
	if visible0["text_set"] {
		t.Error("text_set should not be visible at frame 0")
	}

	// 帧 1：text_ready 隐藏（f=-1），text_set 显示（f=0）
	visible1 := strategy.GetVisibleTracks(comp, 1)
	if visible1["text_ready"] {
		t.Error("text_ready should not be visible at frame 1")
	}
	if !visible1["text_set"] {
		t.Error("text_set should be visible at frame 1")
	}
}

// TestComplexScenePlaybackStrategy_GetVisibleTracks 测试复杂场景播放策略
func TestComplexScenePlaybackStrategy_GetVisibleTracks(t *testing.T) {
	strategy := &ComplexScenePlaybackStrategy{}

	// 测试使用 VisibleTracks 白名单
	comp := &components.ReanimComponent{
		VisibleTracks: map[string]bool{
			"track1": true,
			"track2": true,
		},
		MergedTracks: map[string][]reanim.Frame{
			"track1": {
				{ImagePath: "image1.png"},
			},
			"track2": {
				{ImagePath: "image2.png"},
			},
			"track3": {
				{ImagePath: "image3.png"},
			},
		},
	}

	visible := strategy.GetVisibleTracks(comp, 0)

	// 只有白名单中的轨道可见
	if !visible["track1"] {
		t.Error("track1 should be visible (in whitelist)")
	}
	if !visible["track2"] {
		t.Error("track2 should be visible (in whitelist)")
	}
	if visible["track3"] {
		t.Error("track3 should not be visible (not in whitelist)")
	}
}

// TestBlendedPlaybackStrategy_GetVisibleTracks 测试混合播放策略
func TestBlendedPlaybackStrategy_GetVisibleTracks(t *testing.T) {
	strategy := &BlendedPlaybackStrategy{}

	frameNumMinus1 := -1
	frameNum0 := 0

	comp := &components.ReanimComponent{
		CurrentAnim: "anim_idle",
		AnimVisiblesMap: map[string][]int{
			"anim_idle": {0, 0, 0}, // 所有帧都在时间窗口内
		},
		MergedTracks: map[string][]reanim.Frame{
			"anim_idle": {
				{FrameNum: &frameNum0}, // 动画定义轨道，不渲染
			},
			"head": {
				{ImagePath: "head.png", FrameNum: &frameNumMinus1}, // f=-1，检查时间窗口
			},
			"body": {
				{ImagePath: "body.png", FrameNum: &frameNum0}, // f=0，显示
			},
		},
	}

	visible := strategy.GetVisibleTracks(comp, 0)

	// 动画定义轨道不应该显示
	if visible["anim_idle"] {
		t.Error("anim_idle (animation definition) should not be visible")
	}

	// head（f=-1）应该显示（时间窗口内）
	if !visible["head"] {
		t.Error("head should be visible (f=-1, within time window)")
	}

	// body（f=0）应该显示
	if !visible["body"] {
		t.Error("body should be visible (f=0)")
	}
}

// TestBlendedPlaybackStrategy_OutsideTimeWindow 测试混合策略的时间窗口外逻辑
func TestBlendedPlaybackStrategy_OutsideTimeWindow(t *testing.T) {
	strategy := &BlendedPlaybackStrategy{}

	frameNumMinus1 := -1

	comp := &components.ReanimComponent{
		CurrentAnim: "anim_shooting",
		AnimVisiblesMap: map[string][]int{
			"anim_shooting": {-1, 0, 0}, // 帧 0 在时间窗口外，帧 1-2 在时间窗口内
		},
		MergedTracks: map[string][]reanim.Frame{
			"head": {
				{ImagePath: "head.png", FrameNum: &frameNumMinus1},
				{ImagePath: "head.png", FrameNum: &frameNumMinus1},
				{ImagePath: "head.png", FrameNum: &frameNumMinus1},
			},
		},
	}

	// 帧 0：时间窗口外，所有部件隐藏
	visible0 := strategy.GetVisibleTracks(comp, 0)
	if len(visible0) > 0 {
		t.Errorf("Expected no visible tracks at frame 0 (outside time window), got %d", len(visible0))
	}

	// 帧 1：时间窗口内，部件显示（因为 f=-1 且时间窗口为 0）
	visible1 := strategy.GetVisibleTracks(comp, 1)
	if !visible1["head"] {
		t.Error("head should be visible at frame 1 (inside time window)")
	}
}

// ==================================================================
// PlaybackMode String Tests (播放模式字符串测试)
// ==================================================================

// TestPlaybackMode_String 测试播放模式的字符串表示
func TestPlaybackMode_String(t *testing.T) {
	tests := []struct {
		mode     PlaybackMode
		expected string
	}{
		{ModeSimple, "Simple"},
		{ModeSkeleton, "Skeleton"},
		{ModeSequence, "Sequence"},
		{ModeComplexScene, "ComplexScene"},
		{ModeBlended, "Blended"},
		{PlaybackMode(999), "Unknown"},
	}

	for _, tt := range tests {
		result := tt.mode.String()
		if result != tt.expected {
			t.Errorf("PlaybackMode(%d).String() = %s, expected %s", tt.mode, result, tt.expected)
		}
	}
}
