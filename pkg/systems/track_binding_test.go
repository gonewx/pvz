package systems

import (
	"testing"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
)

// ==================================================================
// Story 13.1: Track Binding Tests (轨道绑定测试)
// ==================================================================

// TestGetVisualTracks 测试获取视觉轨道功能
func TestGetVisualTracks(t *testing.T) {
	// Given: 一个包含多种轨道的 ReanimComponent
	comp := &components.ReanimComponent{
		MergedTracks: map[string][]reanim.Frame{
			// 视觉轨道（有图片）
			"head":         {{ImagePath: "IMAGE_HEAD"}},
			"stalk_bottom": {{ImagePath: "IMAGE_STALK"}},
			// 逻辑轨道（无图片，应被排除）
			"anim_stem": {{X: floatPtr(0.0), Y: floatPtr(0.0)}},
			"_ground":   {{X: floatPtr(0.0), Y: floatPtr(0.0)}},
			// 动画定义轨道（应被排除）
			"anim_idle":     {{FrameNum: intPtr(0)}},
			"anim_shooting": {{FrameNum: intPtr(0)}},
			// 空轨道（无图片，应被排除）
			"empty_track": {{}},
		},
	}

	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)

	// When: 调用 getVisualTracks
	visualTracks := rs.getVisualTracks(comp)

	// Then: 只返回有图片的轨道
	if len(visualTracks) != 2 {
		t.Fatalf("Expected 2 visual tracks, got %d", len(visualTracks))
	}

	// 验证返回的轨道是视觉轨道
	hasHead := false
	hasStalk := false
	for _, trackName := range visualTracks {
		if trackName == "head" {
			hasHead = true
		}
		if trackName == "stalk_bottom" {
			hasStalk = true
		}
	}

	if !hasHead {
		t.Error("Expected 'head' in visual tracks")
	}
	if !hasStalk {
		t.Error("Expected 'stalk_bottom' in visual tracks")
	}
}

// TestFindVisibleWindow 测试查找可见窗口功能
func TestFindVisibleWindow(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)

	tests := []struct {
		name         string
		animVisibles []int
		wantFirst    int
		wantLast     int
		description  string
	}{
		{
			name:         "Normal animation",
			animVisibles: []int{-1, -1, 0, 0, 0, 0, -1, -1},
			wantFirst:    2,
			wantLast:     5,
			description:  "正常动画（可见窗口在中间）",
		},
		{
			name:         "All visible",
			animVisibles: []int{0, 0, 0, 0},
			wantFirst:    0,
			wantLast:     3,
			description:  "全部可见",
		},
		{
			name:         "All hidden",
			animVisibles: []int{-1, -1, -1, -1},
			wantFirst:    -1,
			wantLast:     -1,
			description:  "全部隐藏",
		},
		{
			name:         "Single visible",
			animVisibles: []int{-1, 0, -1},
			wantFirst:    1,
			wantLast:     1,
			description:  "单个可见帧",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: 调用 findVisibleWindow
			first, last := rs.findVisibleWindow(tt.animVisibles)

			// Then: 验证结果
			if first != tt.wantFirst {
				t.Errorf("%s: Expected first=%d, got %d", tt.description, tt.wantFirst, first)
			}
			if last != tt.wantLast {
				t.Errorf("%s: Expected last=%d, got %d", tt.description, tt.wantLast, last)
			}
		})
	}
}

// TestCalculatePositionVariance 测试位置方差计算
func TestCalculatePositionVariance(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)

	tests := []struct {
		name        string
		frames      []reanim.Frame
		start       int
		end         int
		description string
		wantZero    bool // 是否期望方差为 0
	}{
		{
			name: "Static track",
			frames: []reanim.Frame{
				{X: floatPtr(10.0), Y: floatPtr(20.0)},
				{X: floatPtr(10.0), Y: floatPtr(20.0)},
				{X: floatPtr(10.0), Y: floatPtr(20.0)},
			},
			start:       0,
			end:         2,
			description: "静止轨道（方差应为 0）",
			wantZero:    true,
		},
		{
			name: "Moving track",
			frames: []reanim.Frame{
				{X: floatPtr(10.0), Y: floatPtr(20.0)},
				{X: floatPtr(20.0), Y: floatPtr(30.0)},
				{X: floatPtr(30.0), Y: floatPtr(40.0)},
			},
			start:       0,
			end:         2,
			description: "运动轨道（方差应 > 0）",
			wantZero:    false,
		},
		{
			name:        "Empty range",
			frames:      []reanim.Frame{{X: floatPtr(10.0), Y: floatPtr(20.0)}},
			start:       5,
			end:         10,
			description: "越界范围（方差应为 0）",
			wantZero:    true,
		},
		{
			name: "Nil positions",
			frames: []reanim.Frame{
				{X: nil, Y: nil},
				{X: nil, Y: nil},
			},
			start:       0,
			end:         1,
			description: "空位置（方差应为 0）",
			wantZero:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: 调用 calculatePositionVariance
			variance := rs.calculatePositionVariance(tt.frames, tt.start, tt.end)

			// Then: 验证结果
			if tt.wantZero {
				if variance != 0 {
					t.Errorf("%s: Expected variance=0, got %.2f", tt.description, variance)
				}
			} else {
				if variance == 0 {
					t.Errorf("%s: Expected variance>0, got 0", tt.description)
				}
			}
		})
	}
}

// TestAnalyzeTrackBinding_SimpleCase 测试简单场景的轨道绑定分析
func TestAnalyzeTrackBinding_SimpleCase(t *testing.T) {
	// Given: 一个简单的 ReanimComponent，有两个动画和两个轨道
	comp := &components.ReanimComponent{
		AnimVisiblesMap: map[string][]int{
			// anim_shooting: 帧 0-4 可见
			"anim_shooting": {0, 0, 0, 0, 0, -1, -1, -1},
			// anim_head_idle: 帧 5-7 可见
			"anim_head_idle": {-1, -1, -1, -1, -1, 0, 0, 0},
		},
		MergedTracks: map[string][]reanim.Frame{
			// stalk_bottom: 在 anim_shooting 中运动明显
			"stalk_bottom": {
				{ImagePath: "IMG", X: floatPtr(0.0), Y: floatPtr(0.0)},   // frame 0
				{ImagePath: "IMG", X: floatPtr(10.0), Y: floatPtr(10.0)}, // frame 1
				{ImagePath: "IMG", X: floatPtr(20.0), Y: floatPtr(20.0)}, // frame 2
				{ImagePath: "IMG", X: floatPtr(30.0), Y: floatPtr(30.0)}, // frame 3
				{ImagePath: "IMG", X: floatPtr(40.0), Y: floatPtr(40.0)}, // frame 4
				{ImagePath: "IMG", X: floatPtr(40.0), Y: floatPtr(40.0)}, // frame 5 (静止)
				{ImagePath: "IMG", X: floatPtr(40.0), Y: floatPtr(40.0)}, // frame 6
				{ImagePath: "IMG", X: floatPtr(40.0), Y: floatPtr(40.0)}, // frame 7
			},
			// anim_face: 在 anim_head_idle 中运动明显
			"anim_face": {
				{ImagePath: "IMG", X: floatPtr(100.0), Y: floatPtr(100.0)}, // frame 0 (静止)
				{ImagePath: "IMG", X: floatPtr(100.0), Y: floatPtr(100.0)}, // frame 1
				{ImagePath: "IMG", X: floatPtr(100.0), Y: floatPtr(100.0)}, // frame 2
				{ImagePath: "IMG", X: floatPtr(100.0), Y: floatPtr(100.0)}, // frame 3
				{ImagePath: "IMG", X: floatPtr(100.0), Y: floatPtr(100.0)}, // frame 4
				{ImagePath: "IMG", X: floatPtr(110.0), Y: floatPtr(110.0)}, // frame 5 (运动)
				{ImagePath: "IMG", X: floatPtr(120.0), Y: floatPtr(120.0)}, // frame 6
				{ImagePath: "IMG", X: floatPtr(130.0), Y: floatPtr(130.0)}, // frame 7
			},
		},
	}

	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)

	// When: 调用 AnalyzeTrackBinding
	animNames := []string{"anim_shooting", "anim_head_idle"}
	bindings := rs.AnalyzeTrackBinding(comp, animNames)

	// Then: 验证绑定结果
	if len(bindings) != 2 {
		t.Fatalf("Expected 2 bindings, got %d", len(bindings))
	}

	// stalk_bottom 应该绑定到 anim_shooting（在其中运动明显）
	if bindings["stalk_bottom"] != "anim_shooting" {
		t.Errorf("Expected stalk_bottom -> anim_shooting, got %s", bindings["stalk_bottom"])
	}

	// anim_face 应该绑定到 anim_head_idle（在其中运动明显）
	if bindings["anim_face"] != "anim_head_idle" {
		t.Errorf("Expected anim_face -> anim_head_idle, got %s", bindings["anim_face"])
	}
}

// TestSetTrackBindings_Validation 测试 SetTrackBindings 的验证逻辑
func TestSetTrackBindings_Validation(t *testing.T) {
	// Given: 一个有效的 ReanimComponent
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)
	entity := em.CreateEntity()

	comp := &components.ReanimComponent{
		AnimVisiblesMap: map[string][]int{
			"anim_idle": {0, 0, 0},
		},
		MergedTracks: map[string][]reanim.Frame{
			"head": {{ImagePath: "IMG"}},
		},
	}
	ecs.AddComponent(em, entity, comp)

	tests := []struct {
		name        string
		bindings    map[string]string
		expectError bool
		description string
	}{
		{
			name:        "Valid bindings",
			bindings:    map[string]string{"head": "anim_idle"},
			expectError: false,
			description: "有效的绑定",
		},
		{
			name:        "Invalid track name",
			bindings:    map[string]string{"nonexistent_track": "anim_idle"},
			expectError: true,
			description: "轨道名不存在",
		},
		{
			name:        "Invalid animation name",
			bindings:    map[string]string{"head": "nonexistent_anim"},
			expectError: true,
			description: "动画名不存在",
		},
		{
			name:        "Empty bindings",
			bindings:    map[string]string{},
			expectError: false,
			description: "空绑定（应该成功）",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: 调用 SetTrackBindings
			err := rs.SetTrackBindings(entity, tt.bindings)

			// Then: 验证结果
			if tt.expectError && err == nil {
				t.Errorf("%s: Expected error, got nil", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("%s: Expected no error, got %v", tt.description, err)
			}
		})
	}
}

// TestPlayAnimations_AutoBinding 测试 PlayAnimations 的自动绑定集成
func TestPlayAnimations_AutoBinding(t *testing.T) {
	// Given: 一个包含多个动画的 Reanim
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)
	entity := em.CreateEntity()

	reanimData := &reanim.ReanimXML{
		FPS: 12,
		Tracks: []reanim.Track{
			// 动画定义轨道
			{
				Name: "anim_shooting",
				Frames: []reanim.Frame{
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
				},
			},
			{
				Name: "anim_head_idle",
				Frames: []reanim.Frame{
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
				},
			},
			// 视觉轨道
			{
				Name: "stalk_bottom",
				Frames: []reanim.Frame{
					{ImagePath: "IMG_STALK", X: floatPtr(0.0), Y: floatPtr(0.0)},
					{ImagePath: "IMG_STALK", X: floatPtr(10.0), Y: floatPtr(10.0)},
					{ImagePath: "IMG_STALK", X: floatPtr(20.0), Y: floatPtr(20.0)},
				},
			},
			{
				Name: "anim_face",
				Frames: []reanim.Frame{
					{ImagePath: "IMG_FACE", X: floatPtr(100.0), Y: floatPtr(100.0)},
					{ImagePath: "IMG_FACE", X: floatPtr(100.0), Y: floatPtr(100.0)},
					{ImagePath: "IMG_FACE", X: floatPtr(100.0), Y: floatPtr(100.0)},
				},
			},
		},
	}

	comp := &components.ReanimComponent{
		Reanim: reanimData,
	}
	ecs.AddComponent(em, entity, comp)
	ecs.AddComponent(em, entity, &components.PositionComponent{X: 0, Y: 0})

	// When: 调用 PlayAnimations（多个动画）
	err := rs.PlayAnimations(entity, []string{"anim_shooting", "anim_head_idle"})

	// Then: 应该成功且 TrackBindings 不为空
	if err != nil {
		t.Fatalf("PlayAnimations failed: %v", err)
	}

	updatedComp, _ := ecs.GetComponent[*components.ReanimComponent](em, entity)
	if updatedComp.TrackBindings == nil {
		t.Error("Expected TrackBindings to be set, got nil")
	}

	// 验证单个动画时 TrackBindings 为 nil
	err = rs.PlayAnimations(entity, []string{"anim_shooting"})
	if err != nil {
		t.Fatalf("PlayAnimations (single) failed: %v", err)
	}

	updatedComp, _ = ecs.GetComponent[*components.ReanimComponent](em, entity)
	if updatedComp.TrackBindings != nil {
		t.Error("Expected TrackBindings to be nil for single animation, got non-nil")
	}
}

// ==================================================================
// Helper functions (辅助函数)
// ==================================================================
// 注意：floatPtr 和 intPtr 已在 reanim_system_test.go 中定义，此处不重复声明
