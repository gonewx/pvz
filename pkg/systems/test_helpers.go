package systems

import (
	"sync"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

// testAudioContext 是测试用的共享音频上下文
// 所有测试文件共享此上下文以避免重复创建
// 使用延迟初始化避免与 main.go 冲突
var (
	testAudioContext     *audio.Context
	testAudioContextOnce sync.Once
)

// getTestAudioContext 获取测试音频上下文（延迟创建）
func getTestAudioContext() *audio.Context {
	testAudioContextOnce.Do(func() {
		testAudioContext = audio.NewContext(48000)
	})
	return testAudioContext
}

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
			FrameAccumulator:  0.0,
			VisibleFrameCount: 0,
			IsLooping:         true,
			IsFinished:        false,
			AnimVisiblesMap:   map[string][]int{},
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
		FrameAccumulator:  0.0,
		VisibleFrameCount: 1,
		IsLooping:         true,
		IsFinished:        false,
		AnimVisiblesMap:   map[string][]int{"anim_idle": {0}},
		MergedTracks: map[string][]reanim.Frame{
			imageName: {frame},
		},
		AnimTracks:    []reanim.Track{track},
		CenterOffsetX: centerX,
		CenterOffsetY: centerY,
	}
}
