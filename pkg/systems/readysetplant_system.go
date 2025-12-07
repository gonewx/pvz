package systems

import (
	"log"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

const (
	// ReadySetPlant 动画常量
	readySetPlantFPS      = 12   // 动画帧率
	readySetPlantDuration = 2.25 // 动画总时长（25帧 / 12FPS ≈ 2.08秒，加缓冲）
)

// ReadySetPlantSystem 管理 "Ready Set Plant" 动画的播放。
// 该动画在铺草皮完成、UI 显示后播放，告知玩家游戏即将开始。
type ReadySetPlantSystem struct {
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager

	entity      ecs.EntityID // 动画实体
	elapsedTime float64      // 已播放时间
	isPlaying   bool         // 是否正在播放
	isCompleted bool         // 是否已完成
}

// NewReadySetPlantSystem 创建 ReadySetPlant 动画系统。
func NewReadySetPlantSystem(em *ecs.EntityManager, rm *game.ResourceManager) *ReadySetPlantSystem {
	return &ReadySetPlantSystem{
		entityManager:   em,
		resourceManager: rm,
		entity:          0,
		elapsedTime:     0,
		isPlaying:       false,
		isCompleted:     false,
	}
}

// Start 开始播放 ReadySetPlant 动画。
// 在铺草皮完成、UI 显示后调用此方法。
func (s *ReadySetPlantSystem) Start() {
	if s.isPlaying || s.isCompleted {
		return
	}

	// 获取 Reanim 资源
	reanimXML := s.resourceManager.GetReanimXML("StartReadySetPlant")
	partImages := s.resourceManager.GetReanimPartImages("StartReadySetPlant")
	if reanimXML == nil || partImages == nil {
		log.Println("[ReadySetPlantSystem] ⚠️ Failed to load StartReadySetPlant reanim resources")
		s.isCompleted = true // 标记为完成，不阻塞游戏
		return
	}

	// 创建动画实体
	s.entity = s.entityManager.CreateEntity()
	s.isPlaying = true
	s.elapsedTime = 0

	// 添加位置组件（屏幕中心）
	ecs.AddComponent(s.entityManager, s.entity, &components.PositionComponent{
		X: config.ScreenWidth / 2,
		Y: config.ScreenHeight / 2,
	})

	// 添加 UI 组件（标记为 UI 元素，不受摄像机影响）
	ecs.AddComponent(s.entityManager, s.entity, &components.UIComponent{
		State: components.UINormal,
	})

	// 构建 MergedTracks
	mergedTracks := reanim.BuildMergedTracks(reanimXML)

	// 计算总帧数
	totalFrames := 0
	for _, track := range reanimXML.Tracks {
		if len(track.Frames) > totalFrames {
			totalFrames = len(track.Frames)
		}
	}

	// 初始化 AnimVisiblesMap：使用合成动画名 "_root"
	animVisiblesMap := make(map[string][]int)
	visibles := make([]int, totalFrames)
	for i := range visibles {
		visibles[i] = 0 // 所有帧可见
	}
	animVisiblesMap["_root"] = visibles

	// 分析轨道类型（所有轨道都是可视轨道）
	var visualTracks []string
	for _, track := range reanimXML.Tracks {
		visualTracks = append(visualTracks, track.Name)
	}

	// 添加 ReanimComponent
	reanimComp := &components.ReanimComponent{
		ReanimName:        "StartReadySetPlant",
		ReanimXML:         reanimXML,
		PartImages:        partImages,
		MergedTracks:      mergedTracks,
		VisualTracks:      visualTracks,
		LogicalTracks:     nil,
		CurrentFrame:      0,
		FrameAccumulator:  0,
		AnimationFPS:      float64(reanimXML.FPS),
		CurrentAnimations: []string{"_root"},
		AnimVisiblesMap:   animVisiblesMap,
		IsLooping:         false, // 单次播放
		IsFinished:        false,
		LastRenderFrame:   -1,
	}
	ecs.AddComponent(s.entityManager, s.entity, reanimComp)

	// 播放音效
	s.playSound()

	log.Printf("[ReadySetPlantSystem] Started animation (FPS=%d, Frames=%d)", reanimXML.FPS, totalFrames)
}

// Update 更新 ReadySetPlant 动画系统。
func (s *ReadySetPlantSystem) Update(deltaTime float64) {
	if !s.isPlaying || s.isCompleted {
		return
	}

	s.elapsedTime += deltaTime

	// 检查动画是否完成
	if s.elapsedTime >= readySetPlantDuration {
		s.stop()
	}
}

// stop 停止动画并清理实体。
func (s *ReadySetPlantSystem) stop() {
	if s.entity != 0 {
		s.entityManager.DestroyEntity(s.entity)
		s.entity = 0
	}
	s.isPlaying = false
	s.isCompleted = true

	log.Println("[ReadySetPlantSystem] Animation completed")
}

// playSound 播放 ReadySetPlant 音效。
func (s *ReadySetPlantSystem) playSound() {
	// 使用 AudioManager 统一管理音效（Story 10.9）
	if audioManager := game.GetGameState().GetAudioManager(); audioManager != nil {
		audioManager.PlaySound("SOUND_READYSETPLANT")
		log.Println("[ReadySetPlantSystem] Playing sound effect")
	}
}

// IsPlaying 返回动画是否正在播放。
func (s *ReadySetPlantSystem) IsPlaying() bool {
	return s.isPlaying
}

// IsCompleted 返回动画是否已完成。
func (s *ReadySetPlantSystem) IsCompleted() bool {
	return s.isCompleted
}
