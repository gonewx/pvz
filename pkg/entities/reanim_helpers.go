package entities

import (
	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/hajimehoshi/ebiten/v2"
)

// createSimpleReanimComponent 为单图片实体创建简单的 ReanimComponent
// 这个辅助函数将单张图片包装成一个简单的单帧 Reanim 动画
// 所有简单实体（阳光、子弹、特效等）都使用这个函数创建 ReanimComponent
// Story 13.8: 重写以适配新的 ReanimComponent 结构
func createSimpleReanimComponent(image *ebiten.Image, imageName string) *components.ReanimComponent {
	// 处理 nil 图片的情况
	if image == nil {
		return &components.ReanimComponent{
			ReanimName:        "simple_nil",
			ReanimXML:         &reanim.ReanimXML{FPS: 12},
			PartImages:        map[string]*ebiten.Image{},
			MergedTracks:      map[string][]reanim.Frame{},
			VisualTracks:      []string{},
			LogicalTracks:     []string{},
			CurrentFrame:      0,
			FrameAccumulator:  0.0,
			AnimationFPS:      12,
			CurrentAnimations: []string{"idle"},
			AnimVisiblesMap:   map[string][]int{"idle": {}},
			// ✅ Story 13.10: TrackAnimationBinding 已删除
			IsLooping:  true,
			IsFinished: false,
		}
	}

	// 创建一个简单的单帧 Reanim
	// ✅ 修复：使用 "idle" 作为统一的轨道名称和动画名称，确保 buildVisiblesArray 能找到轨道
	animName := "idle" // 统一使用 "idle" 作为简单实体的动画名称

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
		Name:   animName, // 使用 "idle" 作为轨道名称，与动画名称一致
		Frames: []reanim.Frame{frame},
	}

	reanimXML := &reanim.ReanimXML{
		FPS:    12,
		Tracks: []reanim.Track{track},
	}

	partImages := map[string]*ebiten.Image{
		imageName: image,
	}

	mergedTracks := map[string][]reanim.Frame{
		animName: {frame}, // 使用 "idle" 作为轨道名称，与动画名称一致
	}

	// Story 13.8: 新的 ReanimComponent 结构
	// ✅ 修复：确保轨道名称和动画名称一致，避免 buildVisiblesArray 找不到轨道
	return &components.ReanimComponent{
		// 基础数据
		ReanimName:   "simple_" + imageName,
		ReanimXML:    reanimXML,
		PartImages:   partImages,
		MergedTracks: mergedTracks,

		// 轨道分类
		VisualTracks:  []string{animName}, // 使用 "idle" 作为视觉轨道名称
		LogicalTracks: []string{},         // 简单实体没有逻辑轨道

		// 播放状态
		CurrentFrame:      0,
		FrameAccumulator:  0.0,
		AnimationFPS:      12,
		CurrentAnimations: []string{animName}, // 使用统一的动画名称

		// 动画数据
		AnimVisiblesMap: map[string][]int{
			animName: {0}, // 单帧动画，使用统一的动画名称
		},
		// ✅ Story 13.10: TrackAnimationBinding 已删除

		// 配置字段（简单实体不需要）
		ParentTracks: nil,
		HiddenTracks: nil,

		// 渲染缓存
		CachedRenderData: []components.RenderPartData{},
		LastRenderFrame:  -1,

		// 控制标志
		IsPaused:   false,
		IsLooping:  true,
		IsFinished: false,
	}
}
