package utils

import "github.com/decker502/pvz/pkg/components"

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
	default:
		// 未知植物类型，返回默认向日葵图像
		return "assets/images/Plants/SunFlower/SunFlower_1.png"
	}
}
