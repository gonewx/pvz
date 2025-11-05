package systems

import (
	"testing"

	"github.com/decker502/pvz/internal/reanim"
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

	mode := detectPlaybackMode("TestSimple", reanimData)

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

	mode := detectPlaybackMode("TestSimple", reanimData)

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

	mode := detectPlaybackMode("TestSimple", reanimData)

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

	mode := detectPlaybackMode("TestSimple", reanimData)

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

	mode := detectPlaybackMode("TestSimple", reanimData)

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

	mode := detectPlaybackMode("TestSimple", reanimData)

	if mode != ModeBlended {
		t.Errorf("Expected ModeBlended, got %v", mode)
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
