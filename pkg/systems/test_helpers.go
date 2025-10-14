package systems

import (
	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/hajimehoshi/ebiten/v2"
)

// createTestReanimComponent 创建测试用的 ReanimComponent
// 这是一个测试辅助函数，被多个测试文件共享使用
func createTestReanimComponent(image *ebiten.Image, imageName string) *components.ReanimComponent {
	if image == nil {
		// Return a minimal ReanimComponent with no images
		return &components.ReanimComponent{
			Reanim:            &reanim.ReanimXML{FPS: 12},
			PartImages:        map[string]*ebiten.Image{},
			CurrentAnim:       "idle",
			CurrentFrame:      0,
			FrameCounter:      0,
			VisibleFrameCount: 0,
			IsLooping:         true,
			IsFinished:        false,
			AnimVisibles:      []int{},
			MergedTracks:      map[string][]reanim.Frame{},
			AnimTracks:        []reanim.Track{},
			CenterOffsetX:     0,
			CenterOffsetY:     0,
		}
	}

	bounds := image.Bounds()
	imageWidth := float64(bounds.Dx())
	imageHeight := float64(bounds.Dy())

	centerX := imageWidth / 2
	centerY := imageHeight / 2

	frame := reanim.Frame{
		FrameNum: new(int),
		X:        new(float64),
		Y:        new(float64),
		ScaleX:   new(float64),
		ScaleY:   new(float64),
	}
	*frame.FrameNum = 0
	*frame.X = 0
	*frame.Y = 0
	*frame.ScaleX = 1.0
	*frame.ScaleY = 1.0
	frame.ImagePath = imageName

	track := reanim.Track{
		Name:   imageName,
		Frames: []reanim.Frame{frame},
	}

	reanimXML := &reanim.ReanimXML{
		FPS:    12,
		Tracks: []reanim.Track{track},
	}

	partImages := map[string]*ebiten.Image{
		imageName: image,
	}

	return &components.ReanimComponent{
		Reanim:            reanimXML,
		PartImages:        partImages,
		CurrentAnim:       "idle",
		CurrentFrame:      0,
		FrameCounter:      0,
		VisibleFrameCount: 1,
		IsLooping:         true,
		IsFinished:        false,
		AnimVisibles:      []int{0},
		MergedTracks: map[string][]reanim.Frame{
			imageName: {frame},
		},
		AnimTracks:    []reanim.Track{track},
		CenterOffsetX: centerX,
		CenterOffsetY: centerY,
	}
}
