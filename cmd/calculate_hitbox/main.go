package main

import (
	"fmt"
	"log"
	"math"

	"github.com/gonewx/pvz/internal/reanim"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// ButtonTrackInfo 按钮轨道信息
type ButtonTrackInfo struct {
	TrackName    string
	ImageRefName string // IMAGE_REANIM_xxx
	ImagePath    string
}

var buttonTracks = []ButtonTrackInfo{
	{
		TrackName:    "SelectorScreen_Adventure_button",
		ImageRefName: "IMAGE_REANIM_SELECTORSCREEN_ADVENTURE_BUTTON",
		ImagePath:    "assets/reanim/SelectorScreen_Adventure_button.png",
	},
	{
		TrackName:    "SelectorScreen_StartAdventure_button",
		ImageRefName: "IMAGE_REANIM_SELECTORSCREEN_STARTADVENTURE_BUTTON",
		ImagePath:    "assets/reanim/SelectorScreen_Adventure_button.png", // 使用 Adventure 的图片
	},
	{
		TrackName:    "SelectorScreen_Survival_button",
		ImageRefName: "IMAGE_REANIM_SELECTORSCREEN_SURVIVAL_BUTTON",
		ImagePath:    "assets/reanim/SelectorScreen_Survival_button.png",
	},
	{
		TrackName:    "SelectorScreen_Challenges_button",
		ImageRefName: "IMAGE_REANIM_SELECTORSCREEN_CHALLENGES_BUTTON",
		ImagePath:    "assets/reanim/SelectorScreen_Challenges_button.png",
	},
	{
		TrackName:    "SelectorScreen_ZenGarden_button",
		ImageRefName: "IMAGE_REANIM_SELECTORSCREEN_VASEBREAKER_BUTTON",
		ImagePath:    "assets/reanim/SelectorScreen_Vasebreaker_button.png",
	},
}

func main() {
	// 加载 SelectorScreen.reanim 文件
	reanimPath := "data/reanim/SelectorScreen.reanim"
	reanimXML, err := reanim.ParseReanimFile(reanimPath)
	if err != nil {
		log.Fatalf("加载 Reanim 文件失败: %v", err)
	}

	fmt.Println("==========================================================")
	fmt.Println("SelectorScreen 按钮 Hitbox 自动计算工具")
	fmt.Println("==========================================================")
	fmt.Println()

	// 遍历每个按钮
	for _, btnInfo := range buttonTracks {
		fmt.Printf("按钮: %s\n", btnInfo.TrackName)
		fmt.Println("----------------------------------------------------------")

		// 1. 获取轨道数据
		track := findTrack(reanimXML, btnInfo.TrackName)
		if track == nil {
			fmt.Printf("  ❌ 未找到轨道: %s\n\n", btnInfo.TrackName)
			continue
		}

		// 2. 获取最终帧位置（合并所有帧的累加继承）
		finalFrame := getMergedFrame(track)
		if finalFrame == nil || finalFrame.X == nil || finalFrame.Y == nil {
			fmt.Printf("  ❌ 未找到有效帧（X 或 Y 坐标为空）\n\n")
			continue
		}

		fmt.Printf("  📍 Reanim 位置: X=%.1f, Y=%.1f\n", *finalFrame.X, *finalFrame.Y)

		// 3. 加载图片获取尺寸
		img, err := loadImage(btnInfo.ImagePath)
		if err != nil {
			fmt.Printf("  ❌ 加载图片失败: %v\n\n", err)
			continue
		}

		bounds := img.Bounds()
		width := float64(bounds.Dx())
		height := float64(bounds.Dy())

		fmt.Printf("  🖼️  图片尺寸: %.0f x %.0f\n", width, height)

		// 4. 输出变换参数
		if finalFrame.ScaleX != nil || finalFrame.ScaleY != nil {
			scaleX := 1.0
			scaleY := 1.0
			if finalFrame.ScaleX != nil {
				scaleX = *finalFrame.ScaleX
			}
			if finalFrame.ScaleY != nil {
				scaleY = *finalFrame.ScaleY
			}
			fmt.Printf("  🔍 缩放: ScaleX=%.2f, ScaleY=%.2f\n", scaleX, scaleY)
		}

		if finalFrame.SkewX != nil || finalFrame.SkewY != nil {
			skewX := 0.0
			skewY := 0.0
			if finalFrame.SkewX != nil {
				skewX = *finalFrame.SkewX
			}
			if finalFrame.SkewY != nil {
				skewY = *finalFrame.SkewY
			}
			fmt.Printf("  🔍 倾斜: SkewX=%.2f°, SkewY=%.2f°\n", skewX, skewY)
		}

		// 5. 计算四边形四个角坐标
		quadCorners := calculateQuadCorners(finalFrame, width, height)

		fmt.Printf("  ✅ 计算的四边形 Hitbox:\n")
		fmt.Printf("     左上角: (%.1f, %.1f)\n", quadCorners.TopLeft.X, quadCorners.TopLeft.Y)
		fmt.Printf("     右上角: (%.1f, %.1f)\n", quadCorners.TopRight.X, quadCorners.TopRight.Y)
		fmt.Printf("     右下角: (%.1f, %.1f)\n", quadCorners.BottomRight.X, quadCorners.BottomRight.Y)
		fmt.Printf("     左下角: (%.1f, %.1f)\n", quadCorners.BottomLeft.X, quadCorners.BottomLeft.Y)
		fmt.Println()
	}

	fmt.Println("==========================================================")
	fmt.Println("配置代码生成（可直接复制到 menu_config.go）:")
	fmt.Println("==========================================================")
	fmt.Println()

	generateConfigCode(reanimXML)
}

// findTrack 查找轨道
func findTrack(reanimXML *reanim.ReanimXML, trackName string) *reanim.Track {
	for i := range reanimXML.Tracks {
		if reanimXML.Tracks[i].Name == trackName {
			return &reanimXML.Tracks[i]
		}
	}
	return nil
}

// getMergedFrame 获取合并后的最终帧（考虑累加继承）
func getMergedFrame(track *reanim.Track) *reanim.Frame {
	if len(track.Frames) == 0 {
		return nil
	}

	// 创建一个累加帧
	merged := &reanim.Frame{}
	var x, y, scaleX, scaleY, skewX, skewY float64
	hasX, hasY := false, false
	hasScaleX, hasScaleY := false, false
	hasSkewX, hasSkewY := false, false

	// 遍历所有帧，累加继承值
	for i := range track.Frames {
		frame := &track.Frames[i]

		// X 坐标
		if frame.X != nil {
			x = *frame.X
			hasX = true
		}

		// Y 坐标
		if frame.Y != nil {
			y = *frame.Y
			hasY = true
		}

		// 缩放
		if frame.ScaleX != nil {
			scaleX = *frame.ScaleX
			hasScaleX = true
		}
		if frame.ScaleY != nil {
			scaleY = *frame.ScaleY
			hasScaleY = true
		}

		// 倾斜
		if frame.SkewX != nil {
			skewX = *frame.SkewX
			hasSkewX = true
		}
		if frame.SkewY != nil {
			skewY = *frame.SkewY
			hasSkewY = true
		}

		// 图片路径
		if frame.ImagePath != "" {
			merged.ImagePath = frame.ImagePath
		}
	}

	if hasX {
		merged.X = &x
	}
	if hasY {
		merged.Y = &y
	}
	if hasScaleX {
		merged.ScaleX = &scaleX
	}
	if hasScaleY {
		merged.ScaleY = &scaleY
	}
	if hasSkewX {
		merged.SkewX = &skewX
	}
	if hasSkewY {
		merged.SkewY = &skewY
	}

	return merged
}

// QuadCorners 表示四边形的四个角
type QuadCorners struct {
	TopLeft     Point
	TopRight    Point
	BottomRight Point
	BottomLeft  Point
}

// Point 表示2D坐标点
type Point struct {
	X float64
	Y float64
}

// calculateQuadCorners 计算旋转矩形的四个角坐标
// 应用 Reanim 的变换矩阵（缩放 + 倾斜）
func calculateQuadCorners(frame *reanim.Frame, width, height float64) QuadCorners {
	// 默认值
	scaleX := 1.0
	scaleY := 1.0
	skewX := 0.0
	skewY := 0.0
	originX := 0.0
	originY := 0.0

	if frame.ScaleX != nil {
		scaleX = *frame.ScaleX
	}
	if frame.ScaleY != nil {
		scaleY = *frame.ScaleY
	}
	if frame.SkewX != nil {
		skewX = *frame.SkewX
	}
	if frame.SkewY != nil {
		skewY = *frame.SkewY
	}
	if frame.X != nil {
		originX = *frame.X
	}
	if frame.Y != nil {
		originY = *frame.Y
	}

	// Reanim 坐标是图片左上角
	// 四个本地角坐标（相对于图片左上角）
	corners := []Point{
		{0, 0},          // 左上
		{width, 0},      // 右上
		{width, height}, // 右下
		{0, height},     // 左下
	}

	// 应用变换矩阵
	// Reanim 的变换顺序：缩放 → 倾斜 → 平移
	transformed := make([]Point, 4)
	for i, corner := range corners {
		// 1. 应用缩放
		x := corner.X * scaleX
		y := corner.Y * scaleY

		// 2. 应用倾斜（Flash 变换矩阵）
		// SkewX 和 SkewY 是角度（度），需要转换为弧度
		// 变换矩阵：
		//   a = cos(ky) * scaleX
		//   b = sin(ky) * scaleX
		//   c = -sin(kx) * scaleY
		//   d = cos(kx) * scaleY
		//
		// 简化版（因为我们已经应用了缩放）：
		//   newX = x + tan(ky) * y
		//   newY = tan(kx) * x + y

		if skewX != 0 || skewY != 0 {
			skewXRad := skewX * math.Pi / 180.0
			skewYRad := skewY * math.Pi / 180.0

			tanKX := math.Tan(skewXRad)
			tanKY := math.Tan(skewYRad)

			newX := x + tanKY*y
			newY := tanKX*x + y

			x = newX
			y = newY
		}

		// 3. 应用平移（世界坐标）
		transformed[i] = Point{
			X: originX + x,
			Y: originY + y,
		}
	}

	return QuadCorners{
		TopLeft:     transformed[0],
		TopRight:    transformed[1],
		BottomRight: transformed[2],
		BottomLeft:  transformed[3],
	}
}

// loadImage 加载图片
func loadImage(path string) (*ebiten.Image, error) {
	img, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("加载图片 %s 失败: %w", path, err)
	}
	return img, nil
}

// generateConfigCode 生成配置代码
func generateConfigCode(reanimXML *reanim.ReanimXML) {
	buttonTypeMap := map[string]string{
		"SelectorScreen_Adventure_button":      "MenuButtonAdventure",
		"SelectorScreen_StartAdventure_button": "MenuButtonAdventure",
		"SelectorScreen_Survival_button":       "MenuButtonChallenges",
		"SelectorScreen_Challenges_button":     "MenuButtonVasebreaker",
		"SelectorScreen_ZenGarden_button":      "MenuButtonSurvival",
	}

	commentMap := map[string]string{
		"SelectorScreen_StartAdventure_button": "新用户版本的冒险按钮",
		"SelectorScreen_Survival_button":       "注意：轨道名称是 Survival，但实际对应玩玩小游戏",
		"SelectorScreen_Challenges_button":     "注意：轨道名称是 Challenges，但实际对应解谜模式",
		"SelectorScreen_ZenGarden_button":      "注意：轨道名称是 ZenGarden，但实际对应生存模式",
	}

	fmt.Println("var MenuButtonHitboxes = []config.MenuButtonHitbox{")

	for _, btnInfo := range buttonTracks {
		track := findTrack(reanimXML, btnInfo.TrackName)
		if track == nil {
			continue
		}

		finalFrame := getMergedFrame(track)
		if finalFrame == nil || finalFrame.X == nil || finalFrame.Y == nil {
			continue
		}

		img, err := loadImage(btnInfo.ImagePath)
		if err != nil {
			continue
		}

		bounds := img.Bounds()
		width := float64(bounds.Dx())
		height := float64(bounds.Dy())

		// 计算四边形四个角
		quadCorners := calculateQuadCorners(finalFrame, width, height)

		fmt.Println("\t{")
		fmt.Printf("\t\tTrackName:  %q,\n", btnInfo.TrackName)

		buttonType := buttonTypeMap[btnInfo.TrackName]
		comment := commentMap[btnInfo.TrackName]
		if comment != "" {
			fmt.Printf("\t\tButtonType: config.%s, // %s\n", buttonType, comment)
		} else {
			fmt.Printf("\t\tButtonType: config.%s,\n", buttonType)
		}

		fmt.Printf("\t\tTopLeft:     config.Point{X: %.1f, Y: %.1f},\n",
			quadCorners.TopLeft.X, quadCorners.TopLeft.Y)
		fmt.Printf("\t\tTopRight:    config.Point{X: %.1f, Y: %.1f},\n",
			quadCorners.TopRight.X, quadCorners.TopRight.Y)
		fmt.Printf("\t\tBottomRight: config.Point{X: %.1f, Y: %.1f},\n",
			quadCorners.BottomRight.X, quadCorners.BottomRight.Y)
		fmt.Printf("\t\tBottomLeft:  config.Point{X: %.1f, Y: %.1f},\n",
			quadCorners.BottomLeft.X, quadCorners.BottomLeft.Y)
		fmt.Println("\t},")
	}

	fmt.Println("}")
}
