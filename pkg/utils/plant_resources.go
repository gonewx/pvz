package utils

import (
	"fmt"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
)

// GetPlantPreviewImagePath 根据植物类型返回对应的预览图像路径
// 预览图像使用植物动画的第一帧（_1.png）
//
// 参数:
//   - plantType: 植物类型
//
// 返回:
//   - string: 图像文件路径，如果植物类型未知则返回向日葵的默认路径
func GetPlantPreviewImagePath(plantType components.PlantType) string {
	switch plantType {
	case components.PlantSunflower:
		return "assets/images/Plants/SunFlower/SunFlower_1.png"
	case components.PlantPeashooter:
		return "assets/images/Plants/Peashooter/Peashooter_1.png"
	case components.PlantWallnut:
		return "assets/images/Plants/WallNut/WallNut_1.png"
	default:
		// 未知植物类型，返回默认向日葵图像
		return "assets/images/Plants/SunFlower/SunFlower_1.png"
	}
}

// GetPlantImagePath 根据植物类型返回对应的植物实体图像路径
// 植物实体图像使用植物动画的第1帧（_1.png）
//
// 参数:
//   - plantType: 植物类型
//
// 返回:
//   - string: 图像文件路径，如果植物类型未知则返回向日葵的默认路径
func GetPlantImagePath(plantType components.PlantType) string {
	switch plantType {
	case components.PlantSunflower:
		return "assets/images/Plants/SunFlower/SunFlower_1.png"
	case components.PlantPeashooter:
		return "assets/images/Plants/Peashooter/Peashooter_1.png"
	case components.PlantWallnut:
		return "assets/images/Plants/WallNut/WallNut_1.png"
	default:
		// 未知植物类型，返回默认向日葵图像
		return "assets/images/Plants/SunFlower/SunFlower_1.png"
	}
}

// LoadWallnutFullHealthAnimation 加载坚果墙完好状态动画帧
// 加载 WallNut_1.png 到 WallNut_16.png 共16帧
//
// 参数:
//   - rm: ResourceManager 实例，用于加载图像
//
// 返回:
//   - []*ebiten.Image: 动画帧切片（16帧）
//   - error: 如果加载失败则返回错误
func LoadWallnutFullHealthAnimation(rm *game.ResourceManager) ([]*ebiten.Image, error) {
	frames := make([]*ebiten.Image, config.WallnutAnimationFrames)
	for i := 0; i < config.WallnutAnimationFrames; i++ {
		framePath := fmt.Sprintf("assets/images/Plants/WallNut/WallNut_%d.png", i+1)
		frameImage, err := rm.LoadImage(framePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load wallnut full health frame %d: %w", i+1, err)
		}
		frames[i] = frameImage
	}
	return frames, nil
}

// LoadWallnutCracked1Animation 加载坚果墙轻伤状态动画帧
// 加载 Wallnut_cracked1_1.png 到 Wallnut_cracked1_11.png（实际文件只有11帧）
//
// 参数:
//   - rm: ResourceManager 实例，用于加载图像
//
// 返回:
//   - []*ebiten.Image: 动画帧切片
//   - error: 如果加载失败则返回错误
func LoadWallnutCracked1Animation(rm *game.ResourceManager) ([]*ebiten.Image, error) {
	// 注意：根据实际文件，轻伤状态只有11帧（_1 到 _11）
	const cracked1Frames = 11
	frames := make([]*ebiten.Image, cracked1Frames)
	for i := 0; i < cracked1Frames; i++ {
		framePath := fmt.Sprintf("assets/images/Plants/WallNut/Wallnut_cracked1_%d.png", i+1)
		frameImage, err := rm.LoadImage(framePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load wallnut cracked1 frame %d: %w", i+1, err)
		}
		frames[i] = frameImage
	}
	return frames, nil
}

// LoadWallnutCracked2Animation 加载坚果墙重伤状态动画帧
// 加载 Wallnut_cracked2_1.png 到 Wallnut_cracked2_15.png（实际文件有15帧）
//
// 参数:
//   - rm: ResourceManager 实例，用于加载图像
//
// 返回:
//   - []*ebiten.Image: 动画帧切片
//   - error: 如果加载失败则返回错误
func LoadWallnutCracked2Animation(rm *game.ResourceManager) ([]*ebiten.Image, error) {
	// 注意：根据实际文件，重伤状态有15帧（_1 到 _15）
	const cracked2Frames = 15
	frames := make([]*ebiten.Image, cracked2Frames)
	for i := 0; i < cracked2Frames; i++ {
		framePath := fmt.Sprintf("assets/images/Plants/WallNut/Wallnut_cracked2_%d.png", i+1)
		frameImage, err := rm.LoadImage(framePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load wallnut cracked2 frame %d: %w", i+1, err)
		}
		frames[i] = frameImage
	}
	return frames, nil
}
