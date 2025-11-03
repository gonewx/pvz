package entities

import (
	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/hajimehoshi/ebiten/v2"
)

// createSimpleReanimComponent 为单图片实体创建简单的 ReanimComponent
// 这个辅助函数将单张图片包装成一个简单的单帧 Reanim 动画
// 所有简单实体（阳光、子弹、特效等）都使用这个函数创建 ReanimComponent
func createSimpleReanimComponent(image *ebiten.Image, imageName string) *components.ReanimComponent {
	// 处理 nil 图片的情况
	if image == nil {
		return &components.ReanimComponent{
			Reanim:            &reanim.ReanimXML{FPS: 12},
			PartImages:        map[string]*ebiten.Image{},
			CurrentAnim:       "idle",
			CurrentFrame:      0,
			FrameAccumulator:  0.0,
			VisibleFrameCount: 0,
			IsLooping:         true,
			IsFinished:        false,
			AnimVisiblesMap:   map[string][]int{"idle": {}},
			MergedTracks:      map[string][]reanim.Frame{},
			AnimTracks:        []reanim.Track{},
			CenterOffsetX:     0,
			CenterOffsetY:     0,
		}
	}

	bounds := image.Bounds()
	imageWidth := float64(bounds.Dx())
	imageHeight := float64(bounds.Dy())

	// 创建一个简单的单帧 Reanim
	// 使用中心对齐锚点
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
		FrameAccumulator:  0.0,
		VisibleFrameCount: 1,
		IsLooping:         true,
		IsFinished:        false,
		AnimVisiblesMap: map[string][]int{
			"idle": {0},
		},
		MergedTracks: map[string][]reanim.Frame{
			imageName: {frame},
		},
		AnimTracks:    []reanim.Track{track},
		CenterOffsetX: centerX,
		CenterOffsetY: centerY,
	}
}
