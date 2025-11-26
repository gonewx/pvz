package main

import (
	"image/color"
	"log"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type TestGame struct {
	entityManager   *ecs.EntityManager
	resourceManager *game.ResourceManager
	reanimSystem    *systems.ReanimSystem
	renderSystem    *systems.RenderSystem
	entity          ecs.EntityID
	frameCount      int
}

func (g *TestGame) Update() error {
	g.frameCount++
	g.reanimSystem.Update(1.0 / 60.0)
	return nil
}

func (g *TestGame) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{50, 50, 50, 255})

	// 渲染 UI 元素
	g.renderSystem.DrawUIElements(screen)

	// 显示帧计数
	ebitenutil.DebugPrint(screen, "ReadySetPlant Animation Test")
	ebitenutil.DebugPrintAt(screen, "Frame: "+string(rune('0'+g.frameCount%10)), 10, 20)

	// 获取 ReanimComponent 的状态
	if reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](g.entityManager, g.entity); ok {
		log.Printf("[Test] Entity %d: CurrentFrame=%d, CachedRenderData=%d parts",
			g.entity, reanimComp.CurrentFrame, len(reanimComp.CachedRenderData))
	}
}

func (g *TestGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return int(config.ScreenWidth), int(config.ScreenHeight)
}

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)

	// 初始化 EntityManager
	em := ecs.NewEntityManager()

	// 初始化音频上下文
	audioContext := audio.NewContext(44100)

	// 初始化 ResourceManager
	rm := game.NewResourceManager(audioContext)

	// 加载资源配置
	if err := rm.LoadResourceConfig("assets/config/resources.yaml"); err != nil {
		log.Printf("Warning: Failed to load resource config: %v", err)
	}

	// 加载所有 reanim 资源
	if err := rm.LoadReanimResources(); err != nil {
		log.Fatalf("Failed to load reanim resources: %v", err)
	}

	// 初始化系统
	reanimSystem := systems.NewReanimSystem(em)
	renderSystem := systems.NewRenderSystem(em)
	renderSystem.SetReanimSystem(reanimSystem)
	renderSystem.SetResourceManager(rm)

	// 获取 StartReadySetPlant 动画资源
	reanimXML := rm.GetReanimXML("StartReadySetPlant")
	partImages := rm.GetReanimPartImages("StartReadySetPlant")
	if reanimXML == nil || partImages == nil {
		log.Fatalf("Failed to load StartReadySetPlant resources")
	}

	log.Printf("[Test] Loaded StartReadySetPlant: FPS=%d, Tracks=%d", reanimXML.FPS, len(reanimXML.Tracks))
	for _, track := range reanimXML.Tracks {
		log.Printf("[Test]   Track: %s, Frames=%d", track.Name, len(track.Frames))
	}

	// 创建动画实体
	entity := em.CreateEntity()

	// 添加位置组件（屏幕中心）
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: config.ScreenWidth / 2,
		Y: config.ScreenHeight / 2,
	})

	// 添加 UI 组件
	ecs.AddComponent(em, entity, &components.UIComponent{
		State: components.UINormal,
	})

	// 构建 MergedTracks
	mergedTracks := reanim.BuildMergedTracks(reanimXML)
	log.Printf("[Test] MergedTracks keys: %v", func() []string {
		keys := make([]string, 0, len(mergedTracks))
		for k := range mergedTracks {
			keys = append(keys, k)
		}
		return keys
	}())

	// 计算总帧数
	totalFrames := 0
	for _, track := range reanimXML.Tracks {
		if len(track.Frames) > totalFrames {
			totalFrames = len(track.Frames)
		}
	}
	log.Printf("[Test] Total frames: %d", totalFrames)

	// 设置 VisualTracks
	var visualTracks []string
	for _, track := range reanimXML.Tracks {
		visualTracks = append(visualTracks, track.Name)
	}
	log.Printf("[Test] VisualTracks: %v", visualTracks)

	// 设置 AnimVisiblesMap
	animVisiblesMap := make(map[string][]int)
	visibles := make([]int, totalFrames)
	for i := range visibles {
		visibles[i] = 0 // 所有帧可见
	}
	animVisiblesMap["_root"] = visibles
	log.Printf("[Test] AnimVisiblesMap[_root] length: %d", len(animVisiblesMap["_root"]))

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
		IsLooping:         true, // 循环播放以便观察
		IsFinished:        false,
		LastRenderFrame:   -1,
	}
	ecs.AddComponent(em, entity, reanimComp)

	log.Printf("[Test] Created entity %d with ReanimComponent", entity)

	// 创建游戏
	g := &TestGame{
		entityManager:   em,
		resourceManager: rm,
		reanimSystem:    reanimSystem,
		renderSystem:    renderSystem,
		entity:          entity,
	}

	// 运行游戏
	ebiten.SetWindowSize(int(config.ScreenWidth), int(config.ScreenHeight))
	ebiten.SetWindowTitle("ReadySetPlant Animation Test")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
