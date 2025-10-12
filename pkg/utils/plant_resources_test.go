package utils

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
)

// TestGetPlantPreviewImagePath 测试植物预览图像路径获取
func TestGetPlantPreviewImagePath(t *testing.T) {
	testCases := []struct {
		name       string
		plantType  components.PlantType
		expectPath string
	}{
		{
			name:       "向日葵",
			plantType:  components.PlantSunflower,
			expectPath: "assets/images/Plants/SunFlower/SunFlower_1.png",
		},
		{
			name:       "豌豆射手",
			plantType:  components.PlantPeashooter,
			expectPath: "assets/images/Plants/Peashooter/Peashooter_1.png",
		},
		{
			name:       "未知植物类型（默认为向日葵）",
			plantType:  components.PlantType(999), // 不存在的类型
			expectPath: "assets/images/Plants/SunFlower/SunFlower_1.png",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := GetPlantPreviewImagePath(tc.plantType)
			if path != tc.expectPath {
				t.Errorf("GetPlantPreviewImagePath(%v) = %q, expected %q",
					tc.plantType, path, tc.expectPath)
			}
		})
	}
}

// TestGetPlantPreviewImagePathConsistency 测试所有已知植物类型都有对应路径
func TestGetPlantPreviewImagePathConsistency(t *testing.T) {
	knownPlantTypes := []components.PlantType{
		components.PlantSunflower,
		components.PlantPeashooter,
	}

	for _, plantType := range knownPlantTypes {
		path := GetPlantPreviewImagePath(plantType)
		if path == "" {
			t.Errorf("GetPlantPreviewImagePath(%v) returned empty path", plantType)
		}
		// 验证路径格式正确
		if len(path) < 10 {
			t.Errorf("GetPlantPreviewImagePath(%v) returned suspiciously short path: %q",
				plantType, path)
		}
	}
}

// TestLoadWallnutAnimations 测试坚果墙动画加载函数
// 注意：此测试需要实际的图像文件存在，在单元测试中会失败
// 实际功能将在集成测试中验证（运行游戏时）
func TestLoadWallnutAnimations(t *testing.T) {
	t.Skip("此测试需要实际的图像资源文件，将在集成测试中验证")
	// 坚果墙动画加载功能的实际验证将在游戏运行时进行
	// 单元测试主要确保函数签名和编译正确性
}
