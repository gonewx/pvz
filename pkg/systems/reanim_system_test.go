package systems

import (
	"testing"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// TestPlayAnimationOverlay tests the PlayAnimationOverlay method
func TestPlayAnimationOverlay(t *testing.T) {
	// Setup
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)
	entity := em.CreateEntity()

	// Create a test ReanimComponent with mock data
	reanimComp := createTestReanimComponentForOverlay()
	ecs.AddComponent(em, entity, reanimComp)

	// Play base animation
	err := rs.PlayAnimation(entity, "anim_idle")
	if err != nil {
		t.Fatalf("Failed to play base animation: %v", err)
	}

	// Verify base animation is set
	comp, _ := ecs.GetComponent[*components.ReanimComponent](em, entity)
	if comp.BaseAnimName != "anim_idle" {
		t.Errorf("Expected BaseAnimName to be 'anim_idle', got '%s'", comp.BaseAnimName)
	}

	// Play overlay animation
	err = rs.PlayAnimationOverlay(entity, "anim_blink", true)
	if err != nil {
		t.Fatalf("PlayAnimationOverlay failed: %v", err)
	}

	// Verify overlay animation was added
	comp, _ = ecs.GetComponent[*components.ReanimComponent](em, entity)
	if len(comp.OverlayAnims) != 1 {
		t.Errorf("Expected 1 overlay animation, got %d", len(comp.OverlayAnims))
	}

	// Verify base animation was not changed
	if comp.BaseAnimName != "anim_idle" {
		t.Errorf("Base animation should not be changed, got '%s'", comp.BaseAnimName)
	}

	// Verify overlay animation properties
	layer := comp.OverlayAnims[0]
	if layer.AnimName != "anim_blink" {
		t.Errorf("Expected overlay AnimName to be 'anim_blink', got '%s'", layer.AnimName)
	}
	if !layer.IsOneShot {
		t.Error("Expected overlay to be one-shot")
	}
	if layer.IsFinished {
		t.Error("Expected overlay to not be finished initially")
	}
	if layer.CurrentFrame != 0 {
		t.Errorf("Expected overlay CurrentFrame to be 0, got %d", layer.CurrentFrame)
	}
}

// TestOverlayAnimationLifecycle tests the lifecycle of a one-shot overlay animation
func TestOverlayAnimationLifecycle(t *testing.T) {
	// Setup
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)
	entity := em.CreateEntity()

	// Create a test ReanimComponent
	reanimComp := createTestReanimComponentForOverlay()
	ecs.AddComponent(em, entity, reanimComp)

	// Play base animation
	err := rs.PlayAnimation(entity, "anim_idle")
	if err != nil {
		t.Fatalf("Failed to play base animation: %v", err)
	}

	// Play overlay animation (one-shot)
	err = rs.PlayAnimationOverlay(entity, "anim_blink", true)
	if err != nil {
		t.Fatalf("PlayAnimationOverlay failed: %v", err)
	}

	// Get FPS from component
	comp, _ := ecs.GetComponent[*components.ReanimComponent](em, entity)
	fps := float64(comp.Reanim.FPS)
	if fps == 0 {
		fps = 12.0
	}
	timePerFrame := 1.0 / fps

	// Simulate updates until the overlay animation finishes
	// The overlay has 3 visible frames (FrameNum=0,0,0,-1)
	// It should be removed after 3 frame advances + 1 more update to cleanup
	maxUpdates := 10
	removedAt := -1
	for i := 0; i < maxUpdates; i++ {
		rs.Update(timePerFrame)
		comp, _ = ecs.GetComponent[*components.ReanimComponent](em, entity)

		// Check if overlay animation is still present
		if len(comp.OverlayAnims) == 0 {
			// Animation has been removed (completed)
			removedAt = i + 1 // i is 0-based, so add 1 for actual update count
			break
		}
	}

	// Verify animation was removed
	if removedAt < 0 {
		t.Error("Overlay animation did not finish after maximum updates")
	} else if removedAt < 3 {
		t.Errorf("Overlay animation was removed too early (after %d updates, expected >= 3)", removedAt)
	} else {
		t.Logf("Overlay animation completed and removed after %d updates (expected 3-4)", removedAt)
	}
}

// TestMultipleOverlayAnimations tests multiple overlay animations playing simultaneously
func TestMultipleOverlayAnimations(t *testing.T) {
	// Setup
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)
	entity := em.CreateEntity()

	// Create a test ReanimComponent
	reanimComp := createTestReanimComponentForOverlay()
	ecs.AddComponent(em, entity, reanimComp)

	// Play base animation
	err := rs.PlayAnimation(entity, "anim_idle")
	if err != nil {
		t.Fatalf("Failed to play base animation: %v", err)
	}

	// Play first overlay animation
	err = rs.PlayAnimationOverlay(entity, "anim_blink", true)
	if err != nil {
		t.Fatalf("PlayAnimationOverlay failed for anim_blink: %v", err)
	}

	// Play second overlay animation
	err = rs.PlayAnimationOverlay(entity, "anim_full_idle", false) // Looping overlay
	if err != nil {
		t.Fatalf("PlayAnimationOverlay failed for anim_full_idle: %v", err)
	}

	// Verify both overlays were added
	comp, _ := ecs.GetComponent[*components.ReanimComponent](em, entity)
	if len(comp.OverlayAnims) != 2 {
		t.Errorf("Expected 2 overlay animations, got %d", len(comp.OverlayAnims))
	}

	// Verify their properties
	if comp.OverlayAnims[0].AnimName != "anim_blink" {
		t.Errorf("First overlay should be 'anim_blink', got '%s'", comp.OverlayAnims[0].AnimName)
	}
	if comp.OverlayAnims[1].AnimName != "anim_full_idle" {
		t.Errorf("Second overlay should be 'anim_full_idle', got '%s'", comp.OverlayAnims[1].AnimName)
	}

	// Verify one-shot flags
	if !comp.OverlayAnims[0].IsOneShot {
		t.Error("First overlay should be one-shot")
	}
	if comp.OverlayAnims[1].IsOneShot {
		t.Error("Second overlay should not be one-shot")
	}
}

// TestPlayAnimationClearsOverlays tests that PlayAnimation clears all overlay animations
func TestPlayAnimationClearsOverlays(t *testing.T) {
	// Setup
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)
	entity := em.CreateEntity()

	// Create a test ReanimComponent
	reanimComp := createTestReanimComponentForOverlay()
	ecs.AddComponent(em, entity, reanimComp)

	// Play base animation
	err := rs.PlayAnimation(entity, "anim_idle")
	if err != nil {
		t.Fatalf("Failed to play base animation: %v", err)
	}

	// Add overlay animation
	err = rs.PlayAnimationOverlay(entity, "anim_blink", true)
	if err != nil {
		t.Fatalf("PlayAnimationOverlay failed: %v", err)
	}

	// Verify overlay was added
	comp, _ := ecs.GetComponent[*components.ReanimComponent](em, entity)
	if len(comp.OverlayAnims) != 1 {
		t.Fatalf("Expected 1 overlay animation, got %d", len(comp.OverlayAnims))
	}

	// Switch to a new base animation
	err = rs.PlayAnimation(entity, "anim_shooting")
	if err != nil {
		t.Fatalf("Failed to play new base animation: %v", err)
	}

	// Verify overlays were cleared
	comp, _ = ecs.GetComponent[*components.ReanimComponent](em, entity)
	if len(comp.OverlayAnims) != 0 {
		t.Errorf("Expected overlay animations to be cleared, got %d", len(comp.OverlayAnims))
	}

	// Verify new base animation is set
	if comp.BaseAnimName != "anim_shooting" {
		t.Errorf("Expected BaseAnimName to be 'anim_shooting', got '%s'", comp.BaseAnimName)
	}
}

// TestOverlayAnimationLooping tests that non-one-shot overlays loop correctly
func TestOverlayAnimationLooping(t *testing.T) {
	// Setup
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)
	entity := em.CreateEntity()

	// Create a test ReanimComponent
	reanimComp := createTestReanimComponentForOverlay()
	ecs.AddComponent(em, entity, reanimComp)

	// Play base animation
	err := rs.PlayAnimation(entity, "anim_idle")
	if err != nil {
		t.Fatalf("Failed to play base animation: %v", err)
	}

	// Play looping overlay animation
	err = rs.PlayAnimationOverlay(entity, "anim_full_idle", false)
	if err != nil {
		t.Fatalf("PlayAnimationOverlay failed: %v", err)
	}

	// Get FPS
	comp, _ := ecs.GetComponent[*components.ReanimComponent](em, entity)
	fps := float64(comp.Reanim.FPS)
	if fps == 0 {
		fps = 12.0
	}
	timePerFrame := 1.0 / fps

	// Simulate many updates (more than the animation length)
	for i := 0; i < 20; i++ {
		rs.Update(timePerFrame)
	}

	// Verify overlay animation is still present (not removed)
	comp, _ = ecs.GetComponent[*components.ReanimComponent](em, entity)
	if len(comp.OverlayAnims) != 1 {
		t.Errorf("Expected looping overlay to still be present, got %d overlays", len(comp.OverlayAnims))
	}

	// Verify it's not marked as finished
	if comp.OverlayAnims[0].IsFinished {
		t.Error("Looping overlay should not be marked as finished")
	}
}

// createTestReanimComponentForOverlay creates a ReanimComponent for overlay animation testing
// This function creates mock animation data with multiple animation tracks for testing overlay functionality
func createTestReanimComponentForOverlay() *components.ReanimComponent {
	// Create mock Reanim data with test animations
	reanimXML := &reanim.ReanimXML{
		FPS: 12,
		Tracks: []reanim.Track{
			// anim_idle track (animation definition)
			{
				Name: "anim_idle",
				Frames: []reanim.Frame{
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(-1)}, // End marker
				},
			},
			// anim_blink track (overlay animation)
			{
				Name: "anim_blink",
				Frames: []reanim.Frame{
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(-1)}, // End marker
				},
			},
			// anim_shooting track
			{
				Name: "anim_shooting",
				Frames: []reanim.Frame{
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(-1)},
				},
			},
			// anim_full_idle track (longer animation for looping test)
			{
				Name: "anim_full_idle",
				Frames: []reanim.Frame{
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(-1)},
				},
			},
			// Test part track (with actual image)
			{
				Name: "test_part",
				Frames: []reanim.Frame{
					{
						X:         floatPtr(0),
						Y:         floatPtr(0),
						ScaleX:    floatPtr(1),
						ScaleY:    floatPtr(1),
						SkewX:     floatPtr(0),
						SkewY:     floatPtr(0),
						FrameNum:  intPtr(0),
						ImagePath: "IMAGE_TEST",
					},
				},
			},
		},
	}

	return &components.ReanimComponent{
		Reanim:     reanimXML,
		PartImages: make(map[string]*ebiten.Image), // Empty image map for testing
	}
}

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
			tc.reanimComp.MergedTracks = rs.buildMergedTracks(tc.reanimComp)

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
			reanimComp.MergedTracks = rs.buildMergedTracks(reanimComp)

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
