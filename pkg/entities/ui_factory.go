package entities

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
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

	// 创建组件
	return &components.ReanimComponent{
		ReanimXML:  reanimXML,
		PartImages: partImages,
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

	// 创建实体
	entityID := em.CreateEntity()

	// 设置位置（屏幕中央）
	posComp := &components.PositionComponent{
		X: centerX,
		Y: centerY,
	}

	// 添加 UI 组件（标记为 UI 元素，最上层渲染）
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
