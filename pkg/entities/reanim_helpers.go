package entities

import (
	"github.com/gonewx/pvz/internal/reanim"
	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// GetTrackWorldPosition 获取指定轨道在世界坐标系中的位置
//
// 用于从 Reanim 动画的特定轨道（如头部、手臂）获取世界坐标，
// 以便在正确的位置生成粒子效果（如僵尸死亡时头部/手臂掉落）。
//
// 参数：
//   - em: 实体管理器
//   - entityID: 实体 ID
//   - trackName: 轨道名称（如 "anim_head1", "Zombie_outerarm_hand"）
//
// 返回：
//   - x, y: 轨道在世界坐标系中的位置
//   - found: 是否找到该轨道
//
// 计算方法：
//  1. 从 CachedRenderData 中查找指定轨道的渲染数据
//  2. 获取轨道的局部坐标（Frame.X, Frame.Y）和父子偏移（OffsetX, OffsetY）
//  3. 加上实体位置（PositionComponent）得到世界坐标
//  4. 减去 CenterOffset（因为渲染时会减去 CenterOffset）
func GetTrackWorldPosition(em *ecs.EntityManager, entityID ecs.EntityID, trackName string) (x, y float64, found bool) {
	// 获取 ReanimComponent
	comp, ok := ecs.GetComponent[*components.ReanimComponent](em, entityID)
	if !ok {
		return 0, 0, false
	}

	// 获取 PositionComponent
	pos, ok := ecs.GetComponent[*components.PositionComponent](em, entityID)
	if !ok {
		return 0, 0, false
	}

	// 在 CachedRenderData 中查找指定轨道
	for _, data := range comp.CachedRenderData {
		if data.TrackName == trackName {
			// 获取轨道的局部坐标
			localX := getFloat(data.Frame.X)
			localY := getFloat(data.Frame.Y)

			// 计算世界坐标：
			// worldPos = entityPos + (localPos + parentOffset) - centerOffset
			// 注意：渲染时使用 screenX = pos.X - comp.CenterOffsetX + localX + offsetX
			worldX := pos.X + localX + data.OffsetX - comp.CenterOffsetX
			worldY := pos.Y + localY + data.OffsetY - comp.CenterOffsetY

			return worldX, worldY, true
		}
	}

	// 未找到轨道（可能是隐藏的轨道或不存在）
	return 0, 0, false
}

// getFloat 安全获取 float64 指针的值，nil 返回 0
func getFloat(ptr *float64) float64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

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
