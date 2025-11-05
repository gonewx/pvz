package systems

import (
	"testing"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// TestPrepareStaticPreview_NormalPath tests PrepareStaticPreview when a complete visible frame exists
func TestPrepareStaticPreview_NormalPath(t *testing.T) {
	// Given: 一个有完整可见帧的 Reanim 组件
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)
	entity := em.CreateEntity()

	reanimComp := createTestReanimWithCompleteFrames()
	ecs.AddComponent(em, entity, reanimComp)

	// When: 调用 PrepareStaticPreview
	err := rs.PrepareStaticPreview(entity, "TestPlant")

	// Then: 应该成功选择第一个完整帧
	if err != nil {
		t.Fatalf("PrepareStaticPreview failed: %v", err)
	}

	comp, exists := ecs.GetComponent[*components.ReanimComponent](em, entity)
	if !exists {
		t.Fatal("ReanimComponent not found")
	}

	if comp.BestPreviewFrame != 0 {
		t.Errorf("Expected BestPreviewFrame to be 0, got %d", comp.BestPreviewFrame)
	}

	if comp.CurrentAnim != "static_preview" {
		t.Errorf("Expected CurrentAnim to be 'static_preview', got '%s'", comp.CurrentAnim)
	}

	if comp.IsLooping {
		t.Error("Expected IsLooping to be false for static preview")
	}

	if !comp.IsFinished {
		t.Error("Expected IsFinished to be true for static preview")
	}
}

// TestPrepareStaticPreview_HeuristicFallback tests PrepareStaticPreview when no complete frame exists
func TestPrepareStaticPreview_HeuristicFallback(t *testing.T) {
	// Given: 一个没有完整可见帧的 Reanim 组件（所有帧都缺少部分部件）
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)
	entity := em.CreateEntity()

	totalFrames := 100
	reanimComp := createTestReanimWithIncompleteFrames(totalFrames)
	ecs.AddComponent(em, entity, reanimComp)

	// When: 调用 PrepareStaticPreview
	err := rs.PrepareStaticPreview(entity, "TestPlant")

	// Then: 应该使用启发式策略（40% 位置）
	if err != nil {
		t.Fatalf("PrepareStaticPreview failed: %v", err)
	}

	comp, _ := ecs.GetComponent[*components.ReanimComponent](em, entity)
	expectedFrame := int(float64(totalFrames) * 0.4) // 40

	if comp.BestPreviewFrame != expectedFrame {
		t.Errorf("Expected BestPreviewFrame to be %d (40%% of %d), got %d",
			expectedFrame, totalFrames, comp.BestPreviewFrame)
	}

	if comp.CurrentAnim != "static_preview" {
		t.Errorf("Expected CurrentAnim to be 'static_preview', got '%s'", comp.CurrentAnim)
	}
}

// TestPrepareStaticPreview_EmptyReanim tests PrepareStaticPreview with no tracks
func TestPrepareStaticPreview_EmptyReanim(t *testing.T) {
	// Given: 一个空的 Reanim 组件
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)
	entity := em.CreateEntity()

	reanimComp := &components.ReanimComponent{
		Reanim: &reanim.ReanimXML{
			FPS:    12,
			Tracks: []reanim.Track{}, // 空轨道
		},
		PartImages: make(map[string]*ebiten.Image),
	}
	ecs.AddComponent(em, entity, reanimComp)

	// When: 调用 PrepareStaticPreview
	err := rs.PrepareStaticPreview(entity, "EmptyPlant")

	// Then: 应该成功处理（使用默认值）
	if err != nil {
		t.Fatalf("PrepareStaticPreview should handle empty Reanim, got error: %v", err)
	}

	comp, _ := ecs.GetComponent[*components.ReanimComponent](em, entity)

	// 空 Reanim 应该使用帧 0
	if comp.BestPreviewFrame != 0 {
		t.Errorf("Expected BestPreviewFrame to be 0 for empty Reanim, got %d", comp.BestPreviewFrame)
	}
}

// TestFindFirstCompleteVisibleFrame tests the findFirstCompleteVisibleFrame method
func TestFindFirstCompleteVisibleFrame(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)

	testCases := []struct {
		name          string
		reanimComp    *components.ReanimComponent
		expectedFrame int
	}{
		{
			name:          "All frames complete",
			reanimComp:    createTestReanimWithCompleteFrames(),
			expectedFrame: 0, // 第一个完整帧
		},
		{
			name:          "No complete frames",
			reanimComp:    createTestReanimWithIncompleteFrames(50),
			expectedFrame: -1, // 没有完整帧
		},
		{
			name:          "First frame incomplete, second complete",
			reanimComp:    createTestReanimWithFirstFrameIncomplete(),
			expectedFrame: 1, // 第二个帧是第一个完整帧
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Build merged tracks
			tc.reanimComp.MergedTracks = reanim.BuildMergedTracks(tc.reanimComp.Reanim)

			// Call findFirstCompleteVisibleFrame
			result := rs.findFirstCompleteVisibleFrame(tc.reanimComp)

			if result != tc.expectedFrame {
				t.Errorf("Expected frame %d, got %d", tc.expectedFrame, result)
			}
		})
	}
}

// TestFindPreviewFrameHeuristic tests the findPreviewFrameHeuristic method
func TestFindPreviewFrameHeuristic(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)

	testCases := []struct {
		name          string
		totalFrames   int
		expectedFrame int
	}{
		{
			name:          "100 frames",
			totalFrames:   100,
			expectedFrame: 40, // 40% of 100
		},
		{
			name:          "50 frames",
			totalFrames:   50,
			expectedFrame: 20, // 40% of 50
		},
		{
			name:          "1 frame",
			totalFrames:   1,
			expectedFrame: 0, // 40% of 1 = 0
		},
		{
			name:          "0 frames",
			totalFrames:   0,
			expectedFrame: 0, // Edge case: 0 frames
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reanimComp := createTestReanimWithIncompleteFrames(tc.totalFrames)
			reanimComp.MergedTracks = reanim.BuildMergedTracks(reanimComp.Reanim)

			result := rs.findPreviewFrameHeuristic(reanimComp)

			if result != tc.expectedFrame {
				t.Errorf("Expected frame %d for %d total frames, got %d",
					tc.expectedFrame, tc.totalFrames, result)
			}
		})
	}
}

// createTestReanimWithCompleteFrames creates a ReanimComponent with complete visible frames
func createTestReanimWithCompleteFrames() *components.ReanimComponent {
	// Create a Reanim with 3 part tracks, all frames have images and are visible
	return &components.ReanimComponent{
		Reanim: &reanim.ReanimXML{
			FPS: 12,
			Tracks: []reanim.Track{
				{
					Name: "part1",
					Frames: []reanim.Frame{
						{X: floatPtr(0), Y: floatPtr(0), ScaleX: floatPtr(1), ScaleY: floatPtr(1),
							FrameNum: intPtr(0), ImagePath: "IMAGE_PART1"},
						{X: floatPtr(0), Y: floatPtr(0), ScaleX: floatPtr(1), ScaleY: floatPtr(1),
							FrameNum: intPtr(0), ImagePath: "IMAGE_PART1"},
					},
				},
				{
					Name: "part2",
					Frames: []reanim.Frame{
						{X: floatPtr(0), Y: floatPtr(0), ScaleX: floatPtr(1), ScaleY: floatPtr(1),
							FrameNum: intPtr(0), ImagePath: "IMAGE_PART2"},
						{X: floatPtr(0), Y: floatPtr(0), ScaleX: floatPtr(1), ScaleY: floatPtr(1),
							FrameNum: intPtr(0), ImagePath: "IMAGE_PART2"},
					},
				},
				{
					Name: "part3",
					Frames: []reanim.Frame{
						{X: floatPtr(0), Y: floatPtr(0), ScaleX: floatPtr(1), ScaleY: floatPtr(1),
							FrameNum: intPtr(0), ImagePath: "IMAGE_PART3"},
						{X: floatPtr(0), Y: floatPtr(0), ScaleX: floatPtr(1), ScaleY: floatPtr(1),
							FrameNum: intPtr(0), ImagePath: "IMAGE_PART3"},
					},
				},
			},
		},
		PartImages: make(map[string]*ebiten.Image),
	}
}

// createTestReanimWithIncompleteFrames creates a ReanimComponent where no frame has all parts visible
func createTestReanimWithIncompleteFrames(totalFrames int) *components.ReanimComponent {
	// Create frames for part1 and part2, but they never align on the same frame
	frames1 := make([]reanim.Frame, totalFrames)
	frames2 := make([]reanim.Frame, totalFrames)

	for i := 0; i < totalFrames; i++ {
		// part1 visible on even frames, hidden on odd frames
		if i%2 == 0 {
			frames1[i] = reanim.Frame{
				X: floatPtr(0), Y: floatPtr(0), ScaleX: floatPtr(1), ScaleY: floatPtr(1),
				FrameNum: intPtr(0), ImagePath: "IMAGE_PART1",
			}
		} else {
			frames1[i] = reanim.Frame{
				X: floatPtr(0), Y: floatPtr(0), ScaleX: floatPtr(1), ScaleY: floatPtr(1),
				FrameNum: intPtr(-1), ImagePath: "IMAGE_PART1", // Hidden
			}
		}

		// part2 visible on odd frames, hidden on even frames
		if i%2 == 1 {
			frames2[i] = reanim.Frame{
				X: floatPtr(0), Y: floatPtr(0), ScaleX: floatPtr(1), ScaleY: floatPtr(1),
				FrameNum: intPtr(0), ImagePath: "IMAGE_PART2",
			}
		} else {
			frames2[i] = reanim.Frame{
				X: floatPtr(0), Y: floatPtr(0), ScaleX: floatPtr(1), ScaleY: floatPtr(1),
				FrameNum: intPtr(-1), ImagePath: "IMAGE_PART2", // Hidden
			}
		}
	}

	return &components.ReanimComponent{
		Reanim: &reanim.ReanimXML{
			FPS: 12,
			Tracks: []reanim.Track{
				{Name: "part1", Frames: frames1},
				{Name: "part2", Frames: frames2},
			},
		},
		PartImages: make(map[string]*ebiten.Image),
	}
}

// createTestReanimWithFirstFrameIncomplete creates a ReanimComponent where first frame is incomplete
func createTestReanimWithFirstFrameIncomplete() *components.ReanimComponent {
	return &components.ReanimComponent{
		Reanim: &reanim.ReanimXML{
			FPS: 12,
			Tracks: []reanim.Track{
				{
					Name: "part1",
					Frames: []reanim.Frame{
						// Frame 0: hidden (f=-1)
						{X: floatPtr(0), Y: floatPtr(0), ScaleX: floatPtr(1), ScaleY: floatPtr(1),
							FrameNum: intPtr(-1), ImagePath: "IMAGE_PART1"},
						// Frame 1: visible
						{X: floatPtr(0), Y: floatPtr(0), ScaleX: floatPtr(1), ScaleY: floatPtr(1),
							FrameNum: intPtr(0), ImagePath: "IMAGE_PART1"},
					},
				},
				{
					Name: "part2",
					Frames: []reanim.Frame{
						// Frame 0: no image
						{X: floatPtr(0), Y: floatPtr(0), ScaleX: floatPtr(1), ScaleY: floatPtr(1),
							FrameNum: intPtr(0), ImagePath: ""},
						// Frame 1: visible
						{X: floatPtr(0), Y: floatPtr(0), ScaleX: floatPtr(1), ScaleY: floatPtr(1),
							FrameNum: intPtr(0), ImagePath: "IMAGE_PART2"},
					},
				},
			},
		},
		PartImages: make(map[string]*ebiten.Image),
	}
}

// Helper functions to create pointers
func intPtr(i int) *int {
	return &i
}

func floatPtr(f float64) *float64 {
	return &f
}

// TestGetTrackTransform tests the GetTrackTransform method
// Story 10.5: 测试获取轨道实时坐标的API
func TestGetTrackTransform(t *testing.T) {
	// 创建 EntityManager
	em := ecs.NewEntityManager()

	// 创建 ReanimSystem
	rs := NewReanimSystem(em)

	// 测试用例 1: 正常获取轨道坐标
	t.Run("Normal track transform", func(t *testing.T) {
		// 创建测试实体
		entityID := em.CreateEntity()

		// 创建测试用的 Reanim 组件
		reanimComp := &components.ReanimComponent{
			CurrentAnim:  "test_anim",
			CurrentFrame: 2, // 使用第3帧（0-based）
			MergedTracks: map[string][]reanim.Frame{
				"idle_mouth": {
					// Frame 0
					{X: floatPtr(10.0), Y: floatPtr(20.0)},
					// Frame 1
					{X: floatPtr(15.0), Y: floatPtr(25.0)},
					// Frame 2 (当前帧)
					{X: floatPtr(20.0), Y: floatPtr(30.0)},
					// Frame 3
					{X: floatPtr(25.0), Y: floatPtr(35.0)},
				},
			},
		}
		ecs.AddComponent(em, entityID, reanimComp)

		// 调用 GetTrackTransform
		x, y, err := rs.GetTrackTransform(entityID, "idle_mouth")

		// 验证结果
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if x != 20.0 {
			t.Errorf("Expected x=20.0, got %f", x)
		}
		if y != 30.0 {
			t.Errorf("Expected y=30.0, got %f", y)
		}
	})

	// 测试用例 2: 轨道不存在
	t.Run("Track not found", func(t *testing.T) {
		entityID := em.CreateEntity()
		reanimComp := &components.ReanimComponent{
			CurrentAnim:  "test_anim",
			CurrentFrame: 0,
			MergedTracks: map[string][]reanim.Frame{
				"other_track": {
					{X: floatPtr(10.0), Y: floatPtr(20.0)},
				},
			},
		}
		ecs.AddComponent(em, entityID, reanimComp)

		_, _, err := rs.GetTrackTransform(entityID, "nonexistent_track")

		if err == nil {
			t.Fatal("Expected error for nonexistent track, got nil")
		}
		expectedErrMsg := "track 'nonexistent_track' not found"
		if !containsString(err.Error(), expectedErrMsg) {
			t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrMsg, err.Error())
		}
	})

	// 测试用例 3: 实体无 ReanimComponent
	t.Run("Entity without ReanimComponent", func(t *testing.T) {
		entityID := em.CreateEntity()

		_, _, err := rs.GetTrackTransform(entityID, "idle_mouth")

		if err == nil {
			t.Fatal("Expected error for entity without ReanimComponent, got nil")
		}
		expectedErrMsg := "does not have ReanimComponent"
		if !containsString(err.Error(), expectedErrMsg) {
			t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrMsg, err.Error())
		}
	})

	// 测试用例 4: 帧号越界（使用最后一帧）
	t.Run("Frame index out of bounds", func(t *testing.T) {
		entityID := em.CreateEntity()
		reanimComp := &components.ReanimComponent{
			CurrentAnim:  "test_anim",
			CurrentFrame: 100, // 超出范围
			MergedTracks: map[string][]reanim.Frame{
				"idle_mouth": {
					{X: floatPtr(10.0), Y: floatPtr(20.0)},
					{X: floatPtr(15.0), Y: floatPtr(25.0)},
				},
			},
		}
		ecs.AddComponent(em, entityID, reanimComp)

		x, y, err := rs.GetTrackTransform(entityID, "idle_mouth")

		// 应该使用最后一帧（Frame 1）
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if x != 15.0 {
			t.Errorf("Expected x=15.0 (last frame), got %f", x)
		}
		if y != 25.0 {
			t.Errorf("Expected y=25.0 (last frame), got %f", y)
		}
	})

	// 测试用例 5: 坐标为 nil（默认 0, 0）
	t.Run("Nil coordinates", func(t *testing.T) {
		entityID := em.CreateEntity()
		reanimComp := &components.ReanimComponent{
			CurrentAnim:  "test_anim",
			CurrentFrame: 0,
			MergedTracks: map[string][]reanim.Frame{
				"idle_mouth": {
					{X: nil, Y: nil}, // 坐标为 nil
				},
			},
		}
		ecs.AddComponent(em, entityID, reanimComp)

		x, y, err := rs.GetTrackTransform(entityID, "idle_mouth")

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if x != 0.0 {
			t.Errorf("Expected x=0.0 (default), got %f", x)
		}
		if y != 0.0 {
			t.Errorf("Expected y=0.0 (default), got %f", y)
		}
	})

	// 测试用例 6: 轨道无帧数据
	t.Run("Track with no frames", func(t *testing.T) {
		entityID := em.CreateEntity()
		reanimComp := &components.ReanimComponent{
			CurrentAnim:  "test_anim",
			CurrentFrame: 0,
			MergedTracks: map[string][]reanim.Frame{
				"idle_mouth": {}, // 空轨道
			},
		}
		ecs.AddComponent(em, entityID, reanimComp)

		_, _, err := rs.GetTrackTransform(entityID, "idle_mouth")

		if err == nil {
			t.Fatal("Expected error for track with no frames, got nil")
		}
		expectedErrMsg := "has no frames"
		if !containsString(err.Error(), expectedErrMsg) {
			t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrMsg, err.Error())
		}
	})
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}()))
}
