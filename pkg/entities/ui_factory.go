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
