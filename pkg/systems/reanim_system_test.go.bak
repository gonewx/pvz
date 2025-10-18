package systems

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
)

// TestReanimSystem_Update_FrameAdvance tests that the Update method correctly advances frames
func TestReanimSystem_Update_FrameAdvance(t *testing.T) {
	tests := []struct {
		name              string
		fps               int
		visibleFrameCount int
		updateCalls       int
		expectedFrame     int
		expectedCounter   int
	}{
		{
			name:              "FPS 12, advance after 5 updates",
			fps:               12,
			visibleFrameCount: 10,
			updateCalls:       5,
			expectedFrame:     1,
			expectedCounter:   0,
		},
		{
			name:              "FPS 12, no advance after 4 updates",
			fps:               12,
			visibleFrameCount: 10,
			updateCalls:       4,
			expectedFrame:     0,
			expectedCounter:   4,
		},
		{
			name:              "FPS 24, advance after 2.5 updates (use 2)",
			fps:               24,
			visibleFrameCount: 10,
			updateCalls:       3,
			expectedFrame:     1,
			expectedCounter:   1, // 3 updates: 0→1→0(frame++)→1
		},
		{
			name:              "FPS 0 (default 12), advance after 5 updates",
			fps:               0,
			visibleFrameCount: 10,
			updateCalls:       5,
			expectedFrame:     1,
			expectedCounter:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create entity manager and system
			em := ecs.NewEntityManager()
			system := NewReanimSystem(em)

			// Create entity with ReanimComponent
			entity := em.CreateEntity()
			reanimComp := &components.ReanimComponent{
				Reanim: &reanim.ReanimXML{
					FPS: tt.fps,
				},
				CurrentAnim:       "anim_idle",
				CurrentFrame:      0,
				FrameCounter:      0,
				VisibleFrameCount: tt.visibleFrameCount,
			}
			em.AddComponent(entity, reanimComp)

			// Call Update multiple times
			for i := 0; i < tt.updateCalls; i++ {
				system.Update(0.016) // ~60 FPS game loop
			}

			// Verify frame and counter
			comp, _ := em.GetComponent(entity, reflect.TypeOf(&components.ReanimComponent{}))
			result := comp.(*components.ReanimComponent)

			if result.CurrentFrame != tt.expectedFrame {
				t.Errorf("CurrentFrame = %d, want %d", result.CurrentFrame, tt.expectedFrame)
			}
			if result.FrameCounter != tt.expectedCounter {
				t.Errorf("FrameCounter = %d, want %d", result.FrameCounter, tt.expectedCounter)
			}
		})
	}
}

// TestReanimSystem_Update_Loop tests that the Update method correctly loops animations
func TestReanimSystem_Update_Loop(t *testing.T) {
	// Create entity manager and system
	em := ecs.NewEntityManager()
	system := NewReanimSystem(em)

	// Create entity with ReanimComponent
	entity := em.CreateEntity()
	reanimComp := &components.ReanimComponent{
		Reanim: &reanim.ReanimXML{
			FPS: 12,
		},
		CurrentAnim:       "anim_idle",
		CurrentFrame:      9,    // One frame before the end
		FrameCounter:      4,    // Ready to advance on next update (frameSkip=5)
		VisibleFrameCount: 10,   // Total 10 frames (0-9)
		IsLooping:         true, // Enable looping for this test
	}
	em.AddComponent(entity, reanimComp)

	// Call Update once to advance to the end and loop
	system.Update(0.016)

	// Verify frame loops back to 0
	comp, _ := em.GetComponent(entity, reflect.TypeOf(&components.ReanimComponent{}))
	result := comp.(*components.ReanimComponent)

	if result.CurrentFrame != 0 {
		t.Errorf("CurrentFrame = %d, want 0 (should loop)", result.CurrentFrame)
	}
	if result.FrameCounter != 0 {
		t.Errorf("FrameCounter = %d, want 0 (should reset)", result.FrameCounter)
	}
}

// TestReanimSystem_Update_FPSControl tests FPS control logic
func TestReanimSystem_Update_FPSControl(t *testing.T) {
	tests := []struct {
		name             string
		fps              int
		expectedFrameRun int // Number of updates needed to advance one frame
	}{
		{
			name:             "FPS 12 -> frameSkip 5",
			fps:              12,
			expectedFrameRun: 5,
		},
		{
			name:             "FPS 24 -> frameSkip 2.5 (use 2)",
			fps:              24,
			expectedFrameRun: 2,
		},
		{
			name:             "FPS 60 -> frameSkip 1",
			fps:              60,
			expectedFrameRun: 1,
		},
		{
			name:             "FPS 0 -> default to 12 -> frameSkip 5",
			fps:              0,
			expectedFrameRun: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create entity manager and system
			em := ecs.NewEntityManager()
			system := NewReanimSystem(em)

			// Create entity with ReanimComponent
			entity := em.CreateEntity()
			reanimComp := &components.ReanimComponent{
				Reanim: &reanim.ReanimXML{
					FPS: tt.fps,
				},
				CurrentAnim:       "anim_idle",
				CurrentFrame:      0,
				FrameCounter:      0,
				VisibleFrameCount: 10,
			}
			em.AddComponent(entity, reanimComp)

			// Call Update the expected number of times
			for i := 0; i < tt.expectedFrameRun; i++ {
				system.Update(0.016)
			}

			// Verify frame advanced by 1
			comp, _ := em.GetComponent(entity, reflect.TypeOf(&components.ReanimComponent{}))
			result := comp.(*components.ReanimComponent)

			if result.CurrentFrame != 1 {
				t.Errorf("After %d updates, CurrentFrame = %d, want 1", tt.expectedFrameRun, result.CurrentFrame)
			}
		})
	}
}

// TestReanimSystem_Update_SkipNoReanimData tests that Update skips entities without Reanim data
func TestReanimSystem_Update_SkipNoReanimData(t *testing.T) {
	// Create entity manager and system
	em := ecs.NewEntityManager()
	system := NewReanimSystem(em)

	// Create entity with ReanimComponent but no Reanim data
	entity := em.CreateEntity()
	reanimComp := &components.ReanimComponent{
		Reanim:      nil, // No Reanim data
		CurrentAnim: "",
	}
	em.AddComponent(entity, reanimComp)

	// Call Update (should not panic)
	system.Update(0.016)

	// Verify nothing changed
	comp, _ := em.GetComponent(entity, reflect.TypeOf(&components.ReanimComponent{}))
	result := comp.(*components.ReanimComponent)

	if result.CurrentFrame != 0 || result.FrameCounter != 0 {
		t.Errorf("Entity without Reanim data should not be updated")
	}
}

// TestReanimSystem_Update_SkipNoAnimation tests that Update skips entities without current animation
func TestReanimSystem_Update_SkipNoAnimation(t *testing.T) {
	// Create entity manager and system
	em := ecs.NewEntityManager()
	system := NewReanimSystem(em)

	// Create entity with ReanimComponent but no current animation
	entity := em.CreateEntity()
	reanimComp := &components.ReanimComponent{
		Reanim: &reanim.ReanimXML{
			FPS: 12,
		},
		CurrentAnim: "", // No animation set
	}
	em.AddComponent(entity, reanimComp)

	// Call Update (should not panic)
	system.Update(0.016)

	// Verify nothing changed
	comp, _ := em.GetComponent(entity, reflect.TypeOf(&components.ReanimComponent{}))
	result := comp.(*components.ReanimComponent)

	if result.CurrentFrame != 0 || result.FrameCounter != 0 {
		t.Errorf("Entity without current animation should not be updated")
	}
}

// TestReanimSystem_PlayAnimation_Success tests successful animation playback
func TestReanimSystem_PlayAnimation_Success(t *testing.T) {
	// Create test Reanim data with anim_idle
	f0 := 0
	f1 := -1
	x10 := 10.0
	y20 := 20.0

	reanimData := &reanim.ReanimXML{
		FPS: 12,
		Tracks: []reanim.Track{
			{
				Name: "anim_idle",
				Frames: []reanim.Frame{
					{FrameNum: &f0}, // Frame 0: visible
					{FrameNum: &f0}, // Frame 1: visible
					{FrameNum: &f1}, // Frame 2: hidden
				},
			},
			{
				Name: "head",
				Frames: []reanim.Frame{
					{X: &x10, Y: &y20},
					{}, // Frame 1: inherit from frame 0
					{X: &x10},
				},
			},
		},
	}

	// Create entity manager and system
	em := ecs.NewEntityManager()
	system := NewReanimSystem(em)

	// Create entity with ReanimComponent
	entity := em.CreateEntity()
	reanimComp := &components.ReanimComponent{
		Reanim: reanimData,
	}
	em.AddComponent(entity, reanimComp)

	// Play animation
	err := system.PlayAnimation(entity, "anim_idle")
	if err != nil {
		t.Fatalf("PlayAnimation failed: %v", err)
	}

	// Verify animation state
	comp, _ := em.GetComponent(entity, reflect.TypeOf(&components.ReanimComponent{}))
	result := comp.(*components.ReanimComponent)

	if result.CurrentAnim != "anim_idle" {
		t.Errorf("CurrentAnim = %s, want anim_idle", result.CurrentAnim)
	}
	if result.CurrentFrame != 0 {
		t.Errorf("CurrentFrame = %d, want 0", result.CurrentFrame)
	}
	if result.FrameCounter != 0 {
		t.Errorf("FrameCounter = %d, want 0", result.FrameCounter)
	}
	if result.VisibleFrameCount != 2 {
		t.Errorf("VisibleFrameCount = %d, want 2 (frames with f=0)", result.VisibleFrameCount)
	}

	// Verify AnimVisibles array
	expectedVisibles := []int{0, 0, -1}
	if len(result.AnimVisibles) != len(expectedVisibles) {
		t.Errorf("AnimVisibles length = %d, want %d", len(result.AnimVisibles), len(expectedVisibles))
	} else {
		for i, v := range expectedVisibles {
			if result.AnimVisibles[i] != v {
				t.Errorf("AnimVisibles[%d] = %d, want %d", i, result.AnimVisibles[i], v)
			}
		}
	}

	// Verify MergedTracks exist
	if len(result.MergedTracks) == 0 {
		t.Errorf("MergedTracks is empty, expected at least one track")
	}
	if _, ok := result.MergedTracks["head"]; !ok {
		t.Errorf("MergedTracks missing 'head' track")
	}
}

// TestReanimSystem_PlayAnimation_VisiblesArray tests visibility array construction
func TestReanimSystem_PlayAnimation_VisiblesArray(t *testing.T) {
	tests := []struct {
		name             string
		animFrames       []reanim.Frame
		expectedVisibles []int
	}{
		{
			name: "All frames visible",
			animFrames: []reanim.Frame{
				{FrameNum: intPtr(0)},
				{FrameNum: intPtr(0)},
				{FrameNum: intPtr(0)},
			},
			expectedVisibles: []int{0, 0, 0},
		},
		{
			name: "Frame inheritance - nil inherits previous",
			animFrames: []reanim.Frame{
				{FrameNum: intPtr(0)},  // Frame 0: visible
				{},                     // Frame 1: inherit 0
				{FrameNum: intPtr(-1)}, // Frame 2: hidden
				{},                     // Frame 3: inherit -1
			},
			expectedVisibles: []int{0, 0, -1, -1},
		},
		{
			name: "First frame nil defaults to 0",
			animFrames: []reanim.Frame{
				{}, // Frame 0: defaults to 0
				{}, // Frame 1: inherit 0
			},
			expectedVisibles: []int{0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test Reanim data
			reanimData := &reanim.ReanimXML{
				FPS: 12,
				Tracks: []reanim.Track{
					{
						Name:   "anim_test",
						Frames: tt.animFrames,
					},
				},
			}

			// Create entity manager and system
			em := ecs.NewEntityManager()
			system := NewReanimSystem(em)

			// Create entity
			entity := em.CreateEntity()
			reanimComp := &components.ReanimComponent{
				Reanim: reanimData,
			}
			em.AddComponent(entity, reanimComp)

			// Play animation
			err := system.PlayAnimation(entity, "anim_test")
			if err != nil {
				t.Fatalf("PlayAnimation failed: %v", err)
			}

			// Verify AnimVisibles
			comp, _ := em.GetComponent(entity, reflect.TypeOf(&components.ReanimComponent{}))
			result := comp.(*components.ReanimComponent)

			if len(result.AnimVisibles) != len(tt.expectedVisibles) {
				t.Errorf("AnimVisibles length = %d, want %d", len(result.AnimVisibles), len(tt.expectedVisibles))
			} else {
				for i, expected := range tt.expectedVisibles {
					if result.AnimVisibles[i] != expected {
						t.Errorf("AnimVisibles[%d] = %d, want %d", i, result.AnimVisibles[i], expected)
					}
				}
			}
		})
	}
}

// TestReanimSystem_PlayAnimation_Errors tests error scenarios
func TestReanimSystem_PlayAnimation_Errors(t *testing.T) {
	tests := []struct {
		name          string
		setupEntity   func(*ecs.EntityManager) ecs.EntityID
		animName      string
		expectedError string
	}{
		{
			name: "Entity without ReanimComponent",
			setupEntity: func(em *ecs.EntityManager) ecs.EntityID {
				return em.CreateEntity() // No component added
			},
			animName:      "anim_idle",
			expectedError: "does not have a ReanimComponent",
		},
		{
			name: "Entity with ReanimComponent but no Reanim data",
			setupEntity: func(em *ecs.EntityManager) ecs.EntityID {
				entity := em.CreateEntity()
				em.AddComponent(entity, &components.ReanimComponent{
					Reanim: nil,
				})
				return entity
			},
			animName:      "anim_idle",
			expectedError: "no Reanim data",
		},
		{
			name: "Animation not found",
			setupEntity: func(em *ecs.EntityManager) ecs.EntityID {
				entity := em.CreateEntity()
				em.AddComponent(entity, &components.ReanimComponent{
					Reanim: &reanim.ReanimXML{
						FPS: 12,
						Tracks: []reanim.Track{
							{Name: "anim_idle"},
						},
					},
				})
				return entity
			},
			animName:      "anim_nonexistent",
			expectedError: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create entity manager and system
			em := ecs.NewEntityManager()
			system := NewReanimSystem(em)

			// Setup entity
			entity := tt.setupEntity(em)

			// Try to play animation
			err := system.PlayAnimation(entity, tt.animName)

			// Verify error
			if err == nil {
				t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
			} else if !contains(err.Error(), tt.expectedError) {
				t.Errorf("Error = '%s', want error containing '%s'", err.Error(), tt.expectedError)
			}
		})
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestReanimSystem_FrameInheritance tests frame inheritance logic
func TestReanimSystem_FrameInheritance(t *testing.T) {
	x10 := 10.0
	y20 := 20.0
	x30 := 30.0

	// Create test Reanim data with null fields
	reanimData := &reanim.ReanimXML{
		FPS: 12,
		Tracks: []reanim.Track{
			{
				Name: "anim_idle",
				Frames: []reanim.Frame{
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
				},
			},
			{
				Name: "head",
				Frames: []reanim.Frame{
					{X: &x10, Y: &y20}, // Frame 0: X=10, Y=20
					{Y: &y20},          // Frame 1: X=nil (inherit 10), Y=20
					{X: &x30},          // Frame 2: X=30, Y=nil (inherit 20)
				},
			},
		},
	}

	// Create entity manager and system
	em := ecs.NewEntityManager()
	system := NewReanimSystem(em)

	// Create entity
	entity := em.CreateEntity()
	reanimComp := &components.ReanimComponent{
		Reanim: reanimData,
	}
	em.AddComponent(entity, reanimComp)

	// Play animation to trigger buildMergedTracks
	err := system.PlayAnimation(entity, "anim_idle")
	if err != nil {
		t.Fatalf("PlayAnimation failed: %v", err)
	}

	// Get the component
	comp, _ := em.GetComponent(entity, reflect.TypeOf(&components.ReanimComponent{}))
	result := comp.(*components.ReanimComponent)

	// Verify MergedTracks exist
	mergedHead, ok := result.MergedTracks["head"]
	if !ok {
		t.Fatalf("MergedTracks missing 'head' track")
	}

	if len(mergedHead) != 3 {
		t.Fatalf("MergedTracks['head'] length = %d, want 3", len(mergedHead))
	}

	// Verify frame 0: X=10, Y=20
	if mergedHead[0].X == nil || *mergedHead[0].X != 10.0 {
		t.Errorf("Frame 0: X = %v, want 10.0", mergedHead[0].X)
	}
	if mergedHead[0].Y == nil || *mergedHead[0].Y != 20.0 {
		t.Errorf("Frame 0: Y = %v, want 20.0", mergedHead[0].Y)
	}

	// Verify frame 1: X=10 (inherited), Y=20
	if mergedHead[1].X == nil || *mergedHead[1].X != 10.0 {
		t.Errorf("Frame 1: X = %v, want 10.0 (inherited from frame 0)", mergedHead[1].X)
	}
	if mergedHead[1].Y == nil || *mergedHead[1].Y != 20.0 {
		t.Errorf("Frame 1: Y = %v, want 20.0", mergedHead[1].Y)
	}

	// Verify frame 2: X=30, Y=20 (inherited)
	if mergedHead[2].X == nil || *mergedHead[2].X != 30.0 {
		t.Errorf("Frame 2: X = %v, want 30.0", mergedHead[2].X)
	}
	if mergedHead[2].Y == nil || *mergedHead[2].Y != 20.0 {
		t.Errorf("Frame 2: Y = %v, want 20.0 (inherited from frame 1)", mergedHead[2].Y)
	}
}

// TestReanimSystem_FrameInheritance_IndependentPointers tests that each frame has independent pointers
func TestReanimSystem_FrameInheritance_IndependentPointers(t *testing.T) {
	x10 := 10.0

	// Create test Reanim data
	reanimData := &reanim.ReanimXML{
		FPS: 12,
		Tracks: []reanim.Track{
			{
				Name: "anim_idle",
				Frames: []reanim.Frame{
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
				},
			},
			{
				Name: "body",
				Frames: []reanim.Frame{
					{X: &x10}, // Frame 0: X=10
					{},        // Frame 1: X=nil (inherit 10)
				},
			},
		},
	}

	// Create entity manager and system
	em := ecs.NewEntityManager()
	system := NewReanimSystem(em)

	// Create entity
	entity := em.CreateEntity()
	reanimComp := &components.ReanimComponent{
		Reanim: reanimData,
	}
	em.AddComponent(entity, reanimComp)

	// Play animation
	err := system.PlayAnimation(entity, "anim_idle")
	if err != nil {
		t.Fatalf("PlayAnimation failed: %v", err)
	}

	// Get the component
	comp, _ := em.GetComponent(entity, reflect.TypeOf(&components.ReanimComponent{}))
	result := comp.(*components.ReanimComponent)

	// Get merged frames
	mergedBody, ok := result.MergedTracks["body"]
	if !ok {
		t.Fatalf("MergedTracks missing 'body' track")
	}

	if len(mergedBody) != 2 {
		t.Fatalf("MergedTracks['body'] length = %d, want 2", len(mergedBody))
	}

	// Verify both frames have X=10
	if mergedBody[0].X == nil || *mergedBody[0].X != 10.0 {
		t.Errorf("Frame 0: X = %v, want 10.0", mergedBody[0].X)
	}
	if mergedBody[1].X == nil || *mergedBody[1].X != 10.0 {
		t.Errorf("Frame 1: X = %v, want 10.0 (inherited)", mergedBody[1].X)
	}

	// IMPORTANT: Verify pointers are different (not shared)
	if mergedBody[0].X == mergedBody[1].X {
		t.Errorf("Frame 0 and Frame 1 X pointers are the same (shared address), want independent pointers")
	}
}

// TestReanimSystem_FrameInheritance_AccumulateState tests cumulative state updates
func TestReanimSystem_FrameInheritance_AccumulateState(t *testing.T) {
	x10 := 10.0
	x20 := 20.0
	y5 := 5.0
	y15 := 15.0

	// Create test Reanim data with multiple updates
	reanimData := &reanim.ReanimXML{
		FPS: 12,
		Tracks: []reanim.Track{
			{
				Name: "anim_idle",
				Frames: []reanim.Frame{
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
					{FrameNum: intPtr(0)},
				},
			},
			{
				Name: "arm",
				Frames: []reanim.Frame{
					{X: &x10, Y: &y5},  // Frame 0: X=10, Y=5
					{},                 // Frame 1: X=10 (inherit), Y=5 (inherit)
					{X: &x20, Y: &y15}, // Frame 2: X=20, Y=15
					{},                 // Frame 3: X=20 (inherit), Y=15 (inherit)
				},
			},
		},
	}

	// Create entity manager and system
	em := ecs.NewEntityManager()
	system := NewReanimSystem(em)

	// Create entity
	entity := em.CreateEntity()
	reanimComp := &components.ReanimComponent{
		Reanim: reanimData,
	}
	em.AddComponent(entity, reanimComp)

	// Play animation
	err := system.PlayAnimation(entity, "anim_idle")
	if err != nil {
		t.Fatalf("PlayAnimation failed: %v", err)
	}

	// Get the component
	comp, _ := em.GetComponent(entity, reflect.TypeOf(&components.ReanimComponent{}))
	result := comp.(*components.ReanimComponent)

	// Get merged frames
	mergedArm, ok := result.MergedTracks["arm"]
	if !ok {
		t.Fatalf("MergedTracks missing 'arm' track")
	}

	if len(mergedArm) != 4 {
		t.Fatalf("MergedTracks['arm'] length = %d, want 4", len(mergedArm))
	}

	// Verify accumulation: Frame 0: X=10, Y=5
	if mergedArm[0].X == nil || *mergedArm[0].X != 10.0 {
		t.Errorf("Frame 0: X = %v, want 10.0", mergedArm[0].X)
	}
	if mergedArm[0].Y == nil || *mergedArm[0].Y != 5.0 {
		t.Errorf("Frame 0: Y = %v, want 5.0", mergedArm[0].Y)
	}

	// Verify accumulation: Frame 1: X=10 (inherited), Y=5 (inherited)
	if mergedArm[1].X == nil || *mergedArm[1].X != 10.0 {
		t.Errorf("Frame 1: X = %v, want 10.0 (inherited)", mergedArm[1].X)
	}
	if mergedArm[1].Y == nil || *mergedArm[1].Y != 5.0 {
		t.Errorf("Frame 1: Y = %v, want 5.0 (inherited)", mergedArm[1].Y)
	}

	// Verify accumulation: Frame 2: X=20, Y=15
	if mergedArm[2].X == nil || *mergedArm[2].X != 20.0 {
		t.Errorf("Frame 2: X = %v, want 20.0", mergedArm[2].X)
	}
	if mergedArm[2].Y == nil || *mergedArm[2].Y != 15.0 {
		t.Errorf("Frame 2: Y = %v, want 15.0", mergedArm[2].Y)
	}

	// Verify accumulation: Frame 3: X=20 (inherited), Y=15 (inherited)
	if mergedArm[3].X == nil || *mergedArm[3].X != 20.0 {
		t.Errorf("Frame 3: X = %v, want 20.0 (inherited)", mergedArm[3].X)
	}
	if mergedArm[3].Y == nil || *mergedArm[3].Y != 15.0 {
		t.Errorf("Frame 3: Y = %v, want 15.0 (inherited)", mergedArm[3].Y)
	}
}
