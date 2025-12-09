package entities

import (
	"fmt"
	"log"

	"github.com/gonewx/pvz/internal/reanim"
	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
)

// NewLevelProgressBarEntity 创建关卡进度条实体
//
// 参数：
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载图片资源）
//
// 返回：
//   - ecs.EntityID: 进度条实体ID
//   - error: 如果资源加载失败返回错误
func NewLevelProgressBarEntity(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
) (ecs.EntityID, error) {
	// 加载资源
	log.Printf("[UI Factory] Attempting to load progress bar resources...")

	backgroundImage := rm.GetImageByID("IMAGE_FLAG_METER")
	if backgroundImage == nil {
		log.Printf("[UI Factory] ERROR: Failed to load IMAGE_FLAG_METER")
		return 0, fmt.Errorf("failed to load FlagMeter.png")
	}
	log.Printf("[UI Factory] Loaded IMAGE_FLAG_METER: %dx%d", backgroundImage.Bounds().Dx(), backgroundImage.Bounds().Dy())

	progressBarImage := rm.GetImageByID("IMAGE_FLAG_METER_LEVEL_PROGRESS")
	if progressBarImage == nil {
		log.Printf("[UI Factory] ERROR: Failed to load IMAGE_FLAG_METER_LEVEL_PROGRESS")
		return 0, fmt.Errorf("failed to load FlagMeterLevelProgress.png")
	}
	log.Printf("[UI Factory] Loaded IMAGE_FLAG_METER_LEVEL_PROGRESS: %dx%d", progressBarImage.Bounds().Dx(), progressBarImage.Bounds().Dy())

	partsImage := rm.GetImageByID("IMAGE_FLAG_METER_PARTS")
	if partsImage == nil {
		log.Printf("[UI Factory] ERROR: Failed to load IMAGE_FLAG_METER_PARTS")
		return 0, fmt.Errorf("failed to load FlagMeterParts.png")
	}
	log.Printf("[UI Factory] Loaded IMAGE_FLAG_METER_PARTS: %dx%d", partsImage.Bounds().Dx(), partsImage.Bounds().Dy())

	// 创建实体
	entityID := em.CreateEntity()

	// 添加组件（位置会在渲染时根据右对齐动态计算）
	ecs.AddComponent(em, entityID, &components.LevelProgressBarComponent{
		BackgroundImage:   backgroundImage,
		ProgressBarImage:  progressBarImage,
		PartsImage:        partsImage,
		TotalZombies:      0, // 将由 LevelSystem 初始化
		KilledZombies:     0,
		ProgressPercent:   0.0,
		FlagPositions:     []float64{},
		LevelText:         "",   // 将由 LevelSystem 初始化
		ShowLevelTextOnly: true, // 默认只显示文本
		X:                 0,    // 位置会在渲染时动态计算
		Y:                 0,    // 位置会在渲染时动态计算
	})

	log.Printf("[UI Factory] Created level progress bar entity (ID: %d, right-aligned)", entityID)

	return entityID, nil
}

// NewFinalWaveWarningEntity 创建最后一波提示动画实体
//
// Story 11.3: 最后一波僵尸提示动画
//
// 参数：
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载 FinalWave.reanim）
//   - centerX: 屏幕中央 X 坐标（通常是 ScreenWidth/2）
//   - centerY: 屏幕中央 Y 坐标（通常是 ScreenHeight/2）
//
// 返回：
//   - ecs.EntityID: 提示动画实体ID
//   - error: 如果资源加载失败返回错误
//
// 注意：
//   - 调用者需要在创建后添加 AnimationCommandComponent 来播放动画
//   - 推荐使用 combo 配置: UnitID="finalwave", ComboName="warning" (loop: false)
func NewFinalWaveWarningEntity(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	centerX, centerY float64,
) (ecs.EntityID, error) {
	// 加载 FinalWave.reanim 动画
	reanimComp, err := createReanimComponent(rm, "FinalWave")
	if err != nil {
		return 0, fmt.Errorf("failed to load FinalWave.reanim: %w", err)
	}

	// 创建实体
	entityID := em.CreateEntity()

	// 设置位置（屏幕中央）
	posComp := &components.PositionComponent{
		X: centerX,
		Y: centerY,
	}

	// 创建提示组件
	warningComp := &components.FinalWaveWarningComponent{
		AnimEntity:  entityID,
		DisplayTime: 2.5, // 显示 2.5 秒
		ElapsedTime: 0.0,
		IsPlaying:   true,
	}

	// 添加 UI 组件（标记为 UI 元素，最上层渲染）
	uiComp := &components.UIComponent{
		State: components.UINormal,
	}

	// 添加所有组件
	ecs.AddComponent(em, entityID, reanimComp)
	ecs.AddComponent(em, entityID, posComp)
	ecs.AddComponent(em, entityID, warningComp)
	ecs.AddComponent(em, entityID, uiComp)

	log.Printf("[UI Factory] Created final wave warning entity (ID: %d) at (%.2f, %.2f)", entityID, centerX, centerY)

	return entityID, nil
}

// createReanimComponent 创建 Reanim 组件（辅助函数）
//
// 参数：
//   - rm: 资源管理器
//   - unitName: 单位名称（如 "FinalWave"）
//
// 返回：
//   - *components.ReanimComponent: 创建的组件
//   - error: 如果资源加载失败
func createReanimComponent(rm *game.ResourceManager, unitName string) (*components.ReanimComponent, error) {
	// 获取 Reanim XML 定义
	reanimXML := rm.GetReanimXML(unitName)
	if reanimXML == nil {
		return nil, fmt.Errorf("reanim XML not found for %s", unitName)
	}

	// 获取 Reanim 图片资源
	partImages := rm.GetReanimPartImages(unitName)
	if len(partImages) == 0 {
		return nil, fmt.Errorf("reanim images not found for %s", unitName)
	}

	// 构建 MergedTracks（合并所有轨道数据）
	mergedTracks := reanim.BuildMergedTracks(reanimXML)

	// 提取视觉轨道列表（从 XML 中的 <track> 标签）
	visualTracks := make([]string, 0, len(reanimXML.Tracks))
	for _, track := range reanimXML.Tracks {
		visualTracks = append(visualTracks, track.Name)
	}

	// 检查是否为单动画文件（无命名动画）
	// 单动画文件的特征：没有以 "anim_" 开头的控制轨道（除了可能的 anim_screen）
	isSingleAnimFile := true
	for _, track := range reanimXML.Tracks {
		if track.Name != "anim_screen" && len(track.Name) >= 5 && track.Name[:5] == "anim_" {
			isSingleAnimFile = false
			break
		}
	}

	// 为单动画文件构建 visiblesArray
	// 所有物理帧都标记为可见（值为 0），让轨道自己的 FrameNum 控制显隐
	var currentAnimations []string
	var animVisiblesMap map[string][]int

	if isSingleAnimFile {
		// 单动画文件模式：使用 "_root" 合成动画名
		maxFrames := 0
		for _, frames := range mergedTracks {
			if len(frames) > maxFrames {
				maxFrames = len(frames)
			}
		}

		visiblesArray := make([]int, maxFrames)
		for i := 0; i < maxFrames; i++ {
			visiblesArray[i] = 0 // All frames are visible
		}

		currentAnimations = []string{"_root"}
		animVisiblesMap = map[string][]int{
			"_root": visiblesArray,
		}

		log.Printf("[UI Factory] Single-anim file '%s': maxFrames=%d, using _root animation", unitName, maxFrames)
	} else {
		// 命名动画文件：使用默认动画或留空（等待 AnimationCommandComponent）
		currentAnimations = []string{}
		animVisiblesMap = make(map[string][]int)
	}

	// 创建组件
	return &components.ReanimComponent{
		// 基础数据
		ReanimName:   unitName, // Story 8.8: 设置 ReanimName 用于调试和识别
		ReanimXML:    reanimXML,
		PartImages:   partImages,
		MergedTracks: mergedTracks,

		// 轨道分类
		VisualTracks:  visualTracks,
		LogicalTracks: []string{},

		// 播放状态
		CurrentFrame:      0,
		FrameAccumulator:  0.0,
		AnimationFPS:      float64(reanimXML.FPS),
		CurrentAnimations: currentAnimations,

		// 动画数据
		AnimVisiblesMap: animVisiblesMap,

		// 循环与暂停状态 (由 ReanimSystem.PlayCombo() 初始化)
		IsLooping: true, // 默认循环,会被 PlayCombo 覆盖

		// 配置字段
		ParentTracks: nil,
		HiddenTracks: nil,
	}, nil
}

// NewZombiesWonEntity 创建僵尸胜利动画实体
//
// 当僵尸到达左边界且所有除草车已用完时，显示僵尸胜利画面。
//
// 参数：
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载 ZombiesWon.reanim）
//   - centerX: 屏幕中央 X 坐标（通常是 ScreenWidth/2）
//   - centerY: 屏幕中央 Y 坐标（通常是 ScreenHeight/2）
//
// 返回：
//   - ecs.EntityID: 僵尸胜利动画实体ID
//   - error: 如果资源加载失败返回错误
//
// 注意：
//   - 调用者需要在创建后添加 AnimationCommandComponent 来播放动画
//   - 推荐使用单动画模式: AnimationName="anim_screen"
func NewZombiesWonEntity(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	centerX, centerY float64,
) (ecs.EntityID, error) {
	// 加载 ZombiesWon.reanim 动画
	reanimComp, err := createReanimComponent(rm, "ZombiesWon")
	if err != nil {
		return 0, fmt.Errorf("failed to load ZombiesWon.reanim: %w", err)
	}

	// ZombiesWon 是全屏动画，位置从(0, 0)开始，不需要居中偏移
	reanimComp.CenterOffsetX = 0
	reanimComp.CenterOffsetY = 0

	// ✅ 循环状态和速度由配置文件控制 (data/reanim_config/zombieswon.yaml)
	// 调用者需要添加 AnimationCommandComponent 来播放动画:
	//   UnitID: "zombieswon", ComboName: "appear"

	// 创建实体
	entityID := em.CreateEntity()

	// 设置位置（全屏动画从左上角开始）
	posComp := &components.PositionComponent{
		X: 0,
		Y: 0,
	}

	// 添加 UI 组件（标记为 UI 元素，不受摄像机影响）
	uiComp := &components.UIComponent{
		State: components.UINormal,
	}

	// 添加所有组件
	ecs.AddComponent(em, entityID, reanimComp)
	ecs.AddComponent(em, entityID, posComp)
	ecs.AddComponent(em, entityID, uiComp)

	log.Printf("[UI Factory] Created zombies won entity (ID: %d) at (%.2f, %.2f)", entityID, centerX, centerY)

	return entityID, nil
}
