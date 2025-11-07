package systems

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestRenderSystemQuery(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)

	// 创建测试图像
	testImage := ebiten.NewImage(10, 10)

	// 创建拥有位置和 Reanim 组件的实体
	id1 := em.CreateEntity()
	em.AddComponent(id1, &components.PositionComponent{X: 100, Y: 200})
	em.AddComponent(id1, createTestReanimComponent(testImage, "test"))

	// 创建只有位置组件的实体(不应被渲染)
	id2 := em.CreateEntity()
	em.AddComponent(id2, &components.PositionComponent{X: 50, Y: 50})

	// 验证系统能正确查询
	entities := em.GetEntitiesWith(
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.ReanimComponent{}),
	)

	if len(entities) != 1 {
		t.Errorf("Expected 1 renderable entity, got %d", len(entities))
	}

	if len(entities) > 0 && entities[0] != id1 {
		t.Error("Should find id1 as renderable entity")
	}

	// Draw 应该不会崩溃
	screen := ebiten.NewImage(800, 600)
	system.Draw(screen, 0.0) // cameraX = 0
}

func TestRenderSystemWithNilImage(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)

	// 创建有位置但图片为nil的实体 (使用 ReanimComponent 但 PartImages 为 nil)
	id := em.CreateEntity()
	em.AddComponent(id, &components.PositionComponent{X: 100, Y: 200})
	em.AddComponent(id, createTestReanimComponent(nil, "test"))

	// Draw 应该跳过nil图片而不崩溃
	screen := ebiten.NewImage(800, 600)
	system.Draw(screen, 0.0) // cameraX = 0

	// 如果没有panic,测试通过
}

func TestRenderSystemMultipleEntities(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)

	// 创建测试图像
	testImage := ebiten.NewImage(10, 10)

	// 创建多个可渲染实体
	for i := 0; i < 5; i++ {
		id := em.CreateEntity()
		em.AddComponent(id, &components.PositionComponent{X: float64(i * 100), Y: 100})
		em.AddComponent(id, createTestReanimComponent(testImage, "test"))
	}

	// 验证查询结果
	entities := em.GetEntitiesWith(
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.ReanimComponent{}),
	)

	if len(entities) != 5 {
		t.Errorf("Expected 5 renderable entities, got %d", len(entities))
	}

	// Draw 应该不会崩溃
	screen := ebiten.NewImage(800, 600)
	system.Draw(screen, 0.0) // cameraX = 0
}

func TestRenderSystemEmptyScene(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)

	// 没有实体
	screen := ebiten.NewImage(800, 600)

	// Draw 应该不会崩溃
	system.Draw(screen, 0.0) // cameraX = 0
}

// ============================================================================
// Tests for renderReanimEntity() method
// ============================================================================

// TestRenderReanimEntity_MissingComponents tests that renderReanimEntity handles missing components gracefully
func TestRenderReanimEntity_MissingComponents(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)
	screen := ebiten.NewImage(800, 600)

	tests := []struct {
		name          string
		hasPosition   bool
		hasReanim     bool
		expectNoPanic bool
	}{
		{
			name:          "Missing PositionComponent",
			hasPosition:   false,
			hasReanim:     true,
			expectNoPanic: true,
		},
		{
			name:          "Missing ReanimComponent",
			hasPosition:   true,
			hasReanim:     false,
			expectNoPanic: true,
		},
		{
			name:          "Missing both components",
			hasPosition:   false,
			hasReanim:     false,
			expectNoPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && tt.expectNoPanic {
					t.Errorf("renderReanimEntity panicked when it should not: %v", r)
				}
			}()

			entity := em.CreateEntity()

			if tt.hasPosition {
				em.AddComponent(entity, &components.PositionComponent{X: 100, Y: 200})
			}

			if tt.hasReanim {
				em.AddComponent(entity, &components.ReanimComponent{})
			}

			// Should not panic
			system.Draw(screen, 0.0)
		})
	}
}

// TestRenderReanimEntity_EmptyAnimation tests handling of empty or invalid animation data
func TestRenderReanimEntity_EmptyAnimation(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)
	screen := ebiten.NewImage(800, 600)

	tests := []struct {
		name        string
		currentAnim string
		animTracks  []string
	}{
		{
			name:        "Empty CurrentAnim",
			currentAnim: "",
			animTracks:  []string{"track1"},
		},
		{
			name:        "Empty AnimTracks",
			currentAnim: "anim_idle",
			animTracks:  []string{},
		},
		{
			name:        "Both empty",
			currentAnim: "",
			animTracks:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity := em.CreateEntity()
			em.AddComponent(entity, &components.PositionComponent{X: 100, Y: 200})

			reanimComp := &components.ReanimComponent{
				CurrentAnim: tt.currentAnim,
				AnimTracks:  make([]reanim.Track, len(tt.animTracks)),
			}

			for i, track := range tt.animTracks {
				reanimComp.AnimTracks[i] = reanim.Track{Name: track}
			}

			em.AddComponent(entity, reanimComp)

			// Should not panic, just skip rendering
			system.Draw(screen, 0.0)
		})
	}
}

// TestRenderReanimEntity_InvalidPhysicalFrame tests handling of invalid physical frame index
func TestRenderReanimEntity_InvalidPhysicalFrame(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)
	screen := ebiten.NewImage(800, 600)

	entity := em.CreateEntity()
	em.AddComponent(entity, &components.PositionComponent{X: 100, Y: 200})

	// Story 13.2: ReanimComponent 使用 AnimStates 替代 CurrentFrame
	reanimComp := &components.ReanimComponent{
		CurrentAnim:     "anim_idle",
		AnimTracks:      []reanim.Track{{Name: "track1"}},
		AnimVisiblesMap: map[string][]int{"anim_idle": {0, 0, -1}}, // Only 3 frames
		AnimStates: map[string]*components.AnimState{
			"anim_idle": {
				Name:         "anim_idle",
				LogicalFrame: 999, // Invalid frame number
				IsActive:     true,
			},
		},
	}

	em.AddComponent(entity, reanimComp)

	// Should not panic, just skip rendering
	system.Draw(screen, 0.0)
}

// TestRenderReanimEntity_VisibleTracksWhitelist tests VisibleTracks filtering
func TestRenderReanimEntity_VisibleTracksWhitelist(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)
	screen := ebiten.NewImage(800, 600)

	entity := em.CreateEntity()
	em.AddComponent(entity, &components.PositionComponent{X: 100, Y: 200})

	// Create a small test image
	testImage := ebiten.NewImage(10, 10)

	// Setup ReanimComponent with VisibleTracks whitelist
	x := 10.0
	y := 20.0
	frameNum := 0
	reanimComp := &components.ReanimComponent{
		CurrentAnim: "anim_idle",
		AnimTracks: []reanim.Track{
			{Name: "track_visible"},
			{Name: "track_hidden"},
		},
		AnimVisiblesMap: map[string][]int{"anim_idle": {0}}, // One visible frame
		MergedTracks: map[string][]reanim.Frame{
			"track_visible": {
				{X: &x, Y: &y, FrameNum: &frameNum, ImagePath: "test_img"},
			},
			"track_hidden": {
				{X: &x, Y: &y, FrameNum: &frameNum, ImagePath: "test_img"},
			},
		},
		VisibleTracks: map[string]bool{
			"track_visible": true,
			// track_hidden is not in whitelist, should be skipped
		},
		PartImages: map[string]*ebiten.Image{
			"test_img": testImage,
		},
	}

	em.AddComponent(entity, reanimComp)

	// Should not panic, only renders track_visible
	system.Draw(screen, 0.0)
}

// TestRenderReanimEntity_HiddenTracksBlacklist tests HiddenTracks filtering
func TestRenderReanimEntity_HiddenTracksBlacklist(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)
	screen := ebiten.NewImage(800, 600)

	entity := em.CreateEntity()
	em.AddComponent(entity, &components.PositionComponent{X: 100, Y: 200})

	testImage := ebiten.NewImage(10, 10)

	x := 10.0
	y := 20.0
	frameNum := 0
	reanimComp := &components.ReanimComponent{
		CurrentAnim: "anim_idle",
		AnimTracks: []reanim.Track{
			{Name: "track_normal"},
			{Name: "track_hidden"},
		},
		AnimVisiblesMap: map[string][]int{"anim_idle": {0}},
		MergedTracks: map[string][]reanim.Frame{
			"track_normal": {
				{X: &x, Y: &y, FrameNum: &frameNum, ImagePath: "test_img"},
			},
			"track_hidden": {
				{X: &x, Y: &y, FrameNum: &frameNum, ImagePath: "test_img"},
			},
		},
		VisibleTracks: map[string]bool{
			"track_normal": true,
			// track_hidden 不在白名单中，因此不会渲染
		},
		PartImages: map[string]*ebiten.Image{
			"test_img": testImage,
		},
	}

	em.AddComponent(entity, reanimComp)

	// Should not panic, skips track_hidden (not in VisibleTracks)
	system.Draw(screen, 0.0)
}

// TestRenderReanimEntity_MissingMergedFrames tests handling of missing merged track data
func TestRenderReanimEntity_MissingMergedFrames(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)
	screen := ebiten.NewImage(800, 600)

	entity := em.CreateEntity()
	em.AddComponent(entity, &components.PositionComponent{X: 100, Y: 200})

	reanimComp := &components.ReanimComponent{
		CurrentAnim: "anim_idle",
		AnimTracks: []reanim.Track{
			{Name: "track1"},
		},
		AnimVisiblesMap: map[string][]int{"anim_idle": {0}},
		MergedTracks:    map[string][]reanim.Frame{},
		// MergedTracks does not contain "track1"
	}

	em.AddComponent(entity, reanimComp)

	// Should not panic, just skip rendering
	system.Draw(screen, 0.0)
}

// TestRenderReanimEntity_FrameOutOfBounds tests handling of physical index out of bounds
func TestRenderReanimEntity_FrameOutOfBounds(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)
	screen := ebiten.NewImage(800, 600)

	entity := em.CreateEntity()
	em.AddComponent(entity, &components.PositionComponent{X: 100, Y: 200})

	x := 10.0
	y := 20.0
	frameNum := 0
	reanimComp := &components.ReanimComponent{
		CurrentAnim: "anim_idle",
		AnimTracks: []reanim.Track{
			{Name: "track1"},
		},
		AnimVisiblesMap: map[string][]int{"anim_idle": {0}},
		MergedTracks: map[string][]reanim.Frame{
			"track1": {
				{X: &x, Y: &y, FrameNum: &frameNum, ImagePath: "test_img"},
			},
		},
		PartImages: map[string]*ebiten.Image{
			"test_img": ebiten.NewImage(10, 10),
		},
	}

	em.AddComponent(entity, reanimComp)

	// Story 13.2: 手动设置 AnimState.LogicalFrame 超出可用帧数
	if state, ok := reanimComp.AnimStates["anim_idle"]; ok {
		state.LogicalFrame = 10
	} else {
		reanimComp.AnimStates = map[string]*components.AnimState{
			"anim_idle": {
				Name:         "anim_idle",
				LogicalFrame: 10,
				IsActive:     true,
			},
		}
	}
	reanimComp.AnimVisiblesMap["anim_idle"] = []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0} // 11 frames

	// Should not panic, skip rendering when physical index >= len(mergedFrames)
	system.Draw(screen, 0.0)
}

// TestRenderReanimEntity_HiddenFrame tests handling of frames marked as hidden (f == -1)
func TestRenderReanimEntity_HiddenFrame(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)
	screen := ebiten.NewImage(800, 600)

	entity := em.CreateEntity()
	em.AddComponent(entity, &components.PositionComponent{X: 100, Y: 200})

	x := 10.0
	y := 20.0
	frameNumHidden := -1
	reanimComp := &components.ReanimComponent{
		CurrentAnim: "anim_idle",
		AnimTracks: []reanim.Track{
			{Name: "track1"},
		},
		AnimVisiblesMap: map[string][]int{"anim_idle": {0}},
		MergedTracks: map[string][]reanim.Frame{
			"track1": {
				{X: &x, Y: &y, FrameNum: &frameNumHidden, ImagePath: "test_img"},
			},
		},
		PartImages: map[string]*ebiten.Image{
			"test_img": ebiten.NewImage(10, 10),
		},
	}

	em.AddComponent(entity, reanimComp)

	// Should not panic, skip rendering hidden frame
	system.Draw(screen, 0.0)
}

// TestRenderReanimEntity_EmptyImagePath tests handling of frames without image reference
func TestRenderReanimEntity_EmptyImagePath(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)
	screen := ebiten.NewImage(800, 600)

	entity := em.CreateEntity()
	em.AddComponent(entity, &components.PositionComponent{X: 100, Y: 200})

	x := 10.0
	y := 20.0
	frameNum := 0
	reanimComp := &components.ReanimComponent{
		CurrentAnim: "anim_idle",
		AnimTracks: []reanim.Track{
			{Name: "track1"},
		},
		AnimVisiblesMap: map[string][]int{"anim_idle": {0}},
		MergedTracks: map[string][]reanim.Frame{
			"track1": {
				{X: &x, Y: &y, FrameNum: &frameNum, ImagePath: ""}, // Empty image path
			},
		},
	}

	em.AddComponent(entity, reanimComp)

	// Should not panic, skip rendering frame without image
	system.Draw(screen, 0.0)
}

// TestRenderReanimEntity_MissingPartImage tests handling of missing image in PartImages map
func TestRenderReanimEntity_MissingPartImage(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)
	screen := ebiten.NewImage(800, 600)

	entity := em.CreateEntity()
	em.AddComponent(entity, &components.PositionComponent{X: 100, Y: 200})

	x := 10.0
	y := 20.0
	frameNum := 0
	reanimComp := &components.ReanimComponent{
		CurrentAnim: "anim_idle",
		AnimTracks: []reanim.Track{
			{Name: "track1"},
		},
		AnimVisiblesMap: map[string][]int{"anim_idle": {0}},
		MergedTracks: map[string][]reanim.Frame{
			"track1": {
				{X: &x, Y: &y, FrameNum: &frameNum, ImagePath: "missing_img"},
			},
		},
		PartImages: map[string]*ebiten.Image{
			// "missing_img" not in map
		},
	}

	em.AddComponent(entity, reanimComp)

	// Should not panic, skip rendering when image not found
	system.Draw(screen, 0.0)
}

// TestRenderReanimEntity_NilPartImage tests handling of nil image in PartImages map
func TestRenderReanimEntity_NilPartImage(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)
	screen := ebiten.NewImage(800, 600)

	entity := em.CreateEntity()
	em.AddComponent(entity, &components.PositionComponent{X: 100, Y: 200})

	x := 10.0
	y := 20.0
	frameNum := 0
	reanimComp := &components.ReanimComponent{
		CurrentAnim: "anim_idle",
		AnimTracks: []reanim.Track{
			{Name: "track1"},
		},
		AnimVisiblesMap: map[string][]int{"anim_idle": {0}},
		MergedTracks: map[string][]reanim.Frame{
			"track1": {
				{X: &x, Y: &y, FrameNum: &frameNum, ImagePath: "nil_img"},
			},
		},
		PartImages: map[string]*ebiten.Image{
			"nil_img": nil, // Nil image
		},
	}

	em.AddComponent(entity, reanimComp)

	// Should not panic, skip rendering when image is nil
	system.Draw(screen, 0.0)
}

// TestRenderReanimEntity_TransformCalculation tests affine transformation matrix calculation
func TestRenderReanimEntity_TransformCalculation(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)
	screen := ebiten.NewImage(800, 600)

	entity := em.CreateEntity()
	em.AddComponent(entity, &components.PositionComponent{X: 400, Y: 300})

	testImage := ebiten.NewImage(100, 100)

	// Test different transformation scenarios
	tests := []struct {
		name   string
		scaleX float64
		scaleY float64
		skewX  float64
		skewY  float64
		x      float64
		y      float64
	}{
		{
			name:   "Identity transform",
			scaleX: 1.0,
			scaleY: 1.0,
			skewX:  0.0,
			skewY:  0.0,
			x:      0.0,
			y:      0.0,
		},
		{
			name:   "Scaled 2x",
			scaleX: 2.0,
			scaleY: 2.0,
			skewX:  0.0,
			skewY:  0.0,
			x:      0.0,
			y:      0.0,
		},
		{
			name:   "Scaled 0.5x",
			scaleX: 0.5,
			scaleY: 0.5,
			skewX:  0.0,
			skewY:  0.0,
			x:      0.0,
			y:      0.0,
		},
		{
			name:   "With translation",
			scaleX: 1.0,
			scaleY: 1.0,
			skewX:  0.0,
			skewY:  0.0,
			x:      50.0,
			y:      50.0,
		},
		{
			name:   "With skew",
			scaleX: 1.0,
			scaleY: 1.0,
			skewX:  15.0,
			skewY:  15.0,
			x:      0.0,
			y:      0.0,
		},
		{
			name:   "Combined transform",
			scaleX: 1.5,
			scaleY: 1.2,
			skewX:  10.0,
			skewY:  5.0,
			x:      30.0,
			y:      40.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frameNum := 0
			reanimComp := &components.ReanimComponent{
				CurrentAnim: "anim_idle",
				AnimTracks: []reanim.Track{
					{Name: "track1"},
				},
				AnimVisiblesMap: map[string][]int{"anim_idle": {0}},
				MergedTracks: map[string][]reanim.Frame{
					"track1": {
						{
							X:         &tt.x,
							Y:         &tt.y,
							ScaleX:    &tt.scaleX,
							ScaleY:    &tt.scaleY,
							SkewX:     &tt.skewX,
							SkewY:     &tt.skewY,
							FrameNum:  &frameNum,
							ImagePath: "test_img",
						},
					},
				},
				PartImages: map[string]*ebiten.Image{
					"test_img": testImage,
				},
			}

			em.AddComponent(entity, reanimComp)

			// Should not panic, renders with transformation
			system.Draw(screen, 0.0)
		})
	}
}

// TestRenderReanimEntity_CameraOffset tests screen coordinate calculation with camera
func TestRenderReanimEntity_CameraOffset(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)
	screen := ebiten.NewImage(800, 600)

	entity := em.CreateEntity()
	em.AddComponent(entity, &components.PositionComponent{X: 500, Y: 300})

	testImage := ebiten.NewImage(50, 50)

	x := 0.0
	y := 0.0
	frameNum := 0
	reanimComp := &components.ReanimComponent{
		CurrentAnim:   "anim_idle",
		CenterOffsetX: 25.0, // Center offset
		CenterOffsetY: 25.0,
		AnimTracks: []reanim.Track{
			{Name: "track1"},
		},
		AnimVisiblesMap: map[string][]int{"anim_idle": {0}},
		MergedTracks: map[string][]reanim.Frame{
			"track1": {
				{X: &x, Y: &y, FrameNum: &frameNum, ImagePath: "test_img"},
			},
		},
		PartImages: map[string]*ebiten.Image{
			"test_img": testImage,
		},
	}

	em.AddComponent(entity, reanimComp)

	// Test different camera positions
	cameraPositions := []float64{0, 100, 215, 500}

	for _, cameraX := range cameraPositions {
		// Should not panic with different camera positions
		system.Draw(screen, cameraX)
	}
}

// TestRenderReanimEntity_MultipleEntities tests rendering multiple reanim entities
func TestRenderReanimEntity_MultipleEntities(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)
	screen := ebiten.NewImage(800, 600)

	testImage := ebiten.NewImage(32, 32)

	// Create 5 entities with ReanimComponent
	for i := 0; i < 5; i++ {
		entity := em.CreateEntity()
		em.AddComponent(entity, &components.PositionComponent{
			X: float64(100 + i*100),
			Y: 300,
		})

		x := 0.0
		y := 0.0
		frameNum := 0
		reanimComp := &components.ReanimComponent{
			CurrentAnim: "anim_idle",
			AnimTracks: []reanim.Track{
				{Name: "body"},
			},
			AnimVisiblesMap: map[string][]int{"anim_idle": {0}},
			MergedTracks: map[string][]reanim.Frame{
				"body": {
					{X: &x, Y: &y, FrameNum: &frameNum, ImagePath: "body_img"},
				},
			},
			PartImages: map[string]*ebiten.Image{
				"body_img": testImage,
			},
		}

		em.AddComponent(entity, reanimComp)
	}

	// Should not panic rendering multiple entities
	system.Draw(screen, 0.0)
}

// TestFindPhysicalFrameIndex tests the findPhysicalFrameIndex helper method
func TestFindPhysicalFrameIndex(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewRenderSystem(em)

	tests := []struct {
		name             string
		animVisiblesMap  map[string][]int
		currentAnim      string
		logicalFrameNum  int
		expectedPhysical int
	}{
		{
			name:             "First frame",
			animVisiblesMap:  map[string][]int{"anim_idle": {0, 0, -1, 0}},
			currentAnim:      "anim_idle",
			logicalFrameNum:  0,
			expectedPhysical: 0,
		},
		{
			name:             "Second frame",
			animVisiblesMap:  map[string][]int{"anim_idle": {0, 0, -1, 0}},
			currentAnim:      "anim_idle",
			logicalFrameNum:  1,
			expectedPhysical: 1,
		},
		{
			name:             "Third frame (skip hidden)",
			animVisiblesMap:  map[string][]int{"anim_idle": {0, 0, -1, 0}},
			currentAnim:      "anim_idle",
			logicalFrameNum:  2,
			expectedPhysical: 3,
		},
		{
			name:             "Out of bounds",
			animVisiblesMap:  map[string][]int{"anim_idle": {0, 0, -1}},
			currentAnim:      "anim_idle",
			logicalFrameNum:  5,
			expectedPhysical: -1,
		},
		{
			name:             "Empty visibles",
			animVisiblesMap:  map[string][]int{},
			currentAnim:      "anim_idle",
			logicalFrameNum:  0,
			expectedPhysical: 0, // PlayAllFrames mode - direct passthrough
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reanim := &components.ReanimComponent{
				CurrentAnim:     tt.currentAnim,
				AnimVisiblesMap: tt.animVisiblesMap,
			}

			result := system.findPhysicalFrameIndex(reanim, tt.logicalFrameNum)

			if result != tt.expectedPhysical {
				t.Errorf("findPhysicalFrameIndex(%d) = %d, want %d",
					tt.logicalFrameNum, result, tt.expectedPhysical)
			}
		})
	}
}
