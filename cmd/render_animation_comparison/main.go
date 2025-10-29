// cmd/render_animation_comparison/main.go
// 渲染动画对比程序 - 生成实际的可视化对比图片
// 左侧: 严格遵守 f=-1
// 右侧: 忽略部件 f=-1
//
// 用法：
//   go run cmd/render_animation_comparison/main.go

package main

import (
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	canvasWidth  = 240
	canvasHeight = 240
	windowWidth  = canvasWidth*3 + 80
	windowHeight = canvasHeight + 150
	fps          = 12
)

type Game struct {
	reanimXML              *reanim.ReanimXML
	partImages             map[string]*ebiten.Image
	mergedTracks           map[string][]reanim.Frame
	visualTracks           []string
	visibleFrameIndices    []int // anim_shooting 时间窗口
	idleVisibleFrameIndices []int // anim_idle 时间窗口

	currentFrame int
	totalFrames  int
}

func NewGame() (*Game, error) {
	// 加载 Reanim 文件
	reanimXML, err := reanim.ParseReanimFile("assets/effect/reanim/PeaShooterSingle.reanim")
	if err != nil {
		return nil, fmt.Errorf("加载 Reanim 文件失败: %w", err)
	}

	log.Printf("✓ 加载 Reanim: FPS=%d, 轨道=%d", reanimXML.FPS, len(reanimXML.Tracks))

	// 加载部件图片 - 使用 reanim 文件中实际的键名
	partImages := make(map[string]*ebiten.Image)
	imageFiles := map[string]string{
		"IMAGE_REANIM_PEASHOOTER_BACKLEAF":            "assets/reanim/PeaShooter_backleaf.png",
		"IMAGE_REANIM_PEASHOOTER_BACKLEAF_LEFTTIP":    "assets/reanim/PeaShooter_backleaf_lefttip.png",
		"IMAGE_REANIM_PEASHOOTER_BACKLEAF_RIGHTTIP":   "assets/reanim/PeaShooter_backleaf_righttip.png",
		"IMAGE_REANIM_PEASHOOTER_FRONTLEAF":           "assets/reanim/PeaShooter_frontleaf.png",
		"IMAGE_REANIM_PEASHOOTER_FRONTLEAF_LEFTTIP":   "assets/reanim/PeaShooter_frontleaf_lefttip.png",
		"IMAGE_REANIM_PEASHOOTER_FRONTLEAF_RIGHTTIP":  "assets/reanim/PeaShooter_frontleaf_righttip.png",
		"IMAGE_REANIM_PEASHOOTER_STALK_BOTTOM":        "assets/reanim/PeaShooter_stalk_bottom.png",
		"IMAGE_REANIM_PEASHOOTER_STALK_TOP":           "assets/reanim/PeaShooter_stalk_top.png",
		"IMAGE_REANIM_PEASHOOTER_HEAD":                "assets/reanim/PeaShooter_Head.png",
		"IMAGE_REANIM_PEASHOOTER_MOUTH":               "assets/reanim/PeaShooter_mouth.png",
		"IMAGE_REANIM_PEASHOOTER_BLINK1":              "assets/reanim/PeaShooter_blink1.png",
		"IMAGE_REANIM_PEASHOOTER_BLINK2":              "assets/reanim/PeaShooter_blink2.png",
		"IMAGE_REANIM_ANIM_SPROUT":                    "assets/reanim/PeaShooter_sprout.png",
		"IMAGE_REANIM_PEASHOOTER_HEADLEAF_NEAREST":    "assets/reanim/PeaShooter_headleaf_nearest.png",
		"IMAGE_REANIM_PEASHOOTER_HEADLEAF_FARTHEST":   "assets/reanim/PeaShooter_headleaf_farthest.png",
		"IMAGE_REANIM_PEASHOOTER_HEADLEAF_2RDFARTHEST": "assets/reanim/PeaShooter_headleaf_2rdfarthest.png",
		"IMAGE_REANIM_PEASHOOTER_HEADLEAF_3RDFARTHEST": "assets/reanim/PeaShooter_headleaf_3rdfarthest.png",
	}

	for ref, path := range imageFiles {
		img, _, err := ebitenutil.NewImageFromFile(path)
		if err != nil {
			log.Printf("警告: 无法加载 %s: %v", path, err)
			continue
		}
		partImages[ref] = img
	}
	log.Printf("✓ 加载图片: %d 张", len(partImages))

	// 构建 MergedTracks
	standardFrameCount := 0
	for _, track := range reanimXML.Tracks {
		if len(track.Frames) > standardFrameCount {
			standardFrameCount = len(track.Frames)
		}
	}
	mergedTracks := buildMergedTracks(reanimXML, standardFrameCount)

	// 找出视觉轨道
	visualTracks := []string{}
	for _, track := range reanimXML.Tracks {
		hasImage := false
		for _, frame := range track.Frames {
			if frame.ImagePath != "" {
				hasImage = true
				break
			}
		}
		if hasImage {
			visualTracks = append(visualTracks, track.Name)
		}
	}

	// 构建 anim_shooting 时间窗口
	var animDefTrack *reanim.Track
	for i := range reanimXML.Tracks {
		if reanimXML.Tracks[i].Name == "anim_shooting" {
			animDefTrack = &reanimXML.Tracks[i]
			break
		}
	}

	animVisibles := make([]int, standardFrameCount)
	currentValue := 0
	for i := 0; i < standardFrameCount; i++ {
		if i < len(animDefTrack.Frames) {
			frame := animDefTrack.Frames[i]
			if frame.FrameNum != nil {
				currentValue = *frame.FrameNum
			}
		}
		animVisibles[i] = currentValue
	}

	visibleFrameIndices := []int{}
	for i, v := range animVisibles {
		if v == 0 {
			visibleFrameIndices = append(visibleFrameIndices, i)
		}
	}

	log.Printf("✓ 动画窗口: 物理帧 %d-%d (%d 帧)",
		visibleFrameIndices[0], visibleFrameIndices[len(visibleFrameIndices)-1], len(visibleFrameIndices))

	// 构建 anim_idle 时间窗口
	var idleDefTrack *reanim.Track
	for i := range reanimXML.Tracks {
		if reanimXML.Tracks[i].Name == "anim_idle" {
			idleDefTrack = &reanimXML.Tracks[i]
			break
		}
	}

	idleAnimVisibles := make([]int, standardFrameCount)
	idleCurrentValue := 0
	for i := 0; i < standardFrameCount; i++ {
		if i < len(idleDefTrack.Frames) {
			frame := idleDefTrack.Frames[i]
			if frame.FrameNum != nil {
				idleCurrentValue = *frame.FrameNum
			}
		}
		idleAnimVisibles[i] = idleCurrentValue
	}

	idleVisibleFrameIndices := []int{}
	for i, v := range idleAnimVisibles {
		if v == 0 {
			idleVisibleFrameIndices = append(idleVisibleFrameIndices, i)
		}
	}

	log.Printf("✓ anim_idle 窗口: 物理帧 %d-%d (%d 帧)",
		idleVisibleFrameIndices[0], idleVisibleFrameIndices[len(idleVisibleFrameIndices)-1], len(idleVisibleFrameIndices))

	return &Game{
		reanimXML:              reanimXML,
		partImages:             partImages,
		mergedTracks:           mergedTracks,
		visualTracks:           visualTracks,
		visibleFrameIndices:    visibleFrameIndices,
		idleVisibleFrameIndices: idleVisibleFrameIndices,
		currentFrame:           0,
		totalFrames:            len(visibleFrameIndices),
	}, nil
}

func (g *Game) Update() error {
	// 推进动画
	g.currentFrame++
	if g.currentFrame >= g.totalFrames {
		g.currentFrame = 0
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{50, 50, 50, 255})

	// 标题
	physicalFrame := g.visibleFrameIndices[g.currentFrame]
	title := fmt.Sprintf("anim_shooting - 逻辑帧 %d (物理帧 %d)", g.currentFrame, physicalFrame)
	ebitenutil.DebugPrintAt(screen, title, 10, 10)

	// 左侧标签
	ebitenutil.DebugPrintAt(screen, "左: 严格 f=-1", 20, 40)
	ebitenutil.DebugPrintAt(screen, "(f=-1隐藏)", 20, 55)

	// 中间标签
	ebitenutil.DebugPrintAt(screen, "中: 忽略 f=-1", 280, 40)
	ebitenutil.DebugPrintAt(screen, "(只看定义轨道)", 280, 55)

	// 右侧标签
	ebitenutil.DebugPrintAt(screen, "右: 双动画", 540, 40)
	ebitenutil.DebugPrintAt(screen, "(idle+shooting)", 540, 55)

	// 左侧: 严格规则
	leftCanvas := ebiten.NewImage(canvasWidth, canvasHeight)
	leftCanvas.Fill(color.RGBA{240, 240, 240, 255})
	strictCount := g.renderWithStrictRule(leftCanvas, physicalFrame)

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(20, 75)
	screen.DrawImage(leftCanvas, opts)

	// 中间: 忽略规则
	midCanvas := ebiten.NewImage(canvasWidth, canvasHeight)
	midCanvas.Fill(color.RGBA{240, 240, 240, 255})
	ignoreCount := g.renderWithIgnoreRule(midCanvas, physicalFrame)

	opts2 := &ebiten.DrawImageOptions{}
	opts2.GeoM.Translate(280, 75)
	screen.DrawImage(midCanvas, opts2)

	// 右侧: 双动画叠加 (anim_idle + anim_shooting)
	rightCanvas := ebiten.NewImage(canvasWidth, canvasHeight)
	rightCanvas.Fill(color.RGBA{240, 240, 240, 255})
	dualCount := g.renderWithDualAnimation(rightCanvas, physicalFrame)

	opts3 := &ebiten.DrawImageOptions{}
	opts3.GeoM.Translate(540, 75)
	screen.DrawImage(rightCanvas, opts3)

	// 显示部件数
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("可见: %d", strictCount), 20, canvasHeight+85)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("可见: %d", ignoreCount), 280, canvasHeight+85)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("可见: %d", dualCount), 540, canvasHeight+85)
}

func (g *Game) renderWithStrictRule(canvas *ebiten.Image, physicalIndex int) int {
	count := 0
	for _, trackName := range g.visualTracks {
		mergedFrames, ok := g.mergedTracks[trackName]
		if !ok || physicalIndex >= len(mergedFrames) {
			continue
		}

		frame := mergedFrames[physicalIndex]

		// 严格规则：f=-1 时跳过
		if frame.FrameNum != nil && *frame.FrameNum == -1 {
			continue
		}

		if frame.ImagePath == "" {
			continue
		}

		img, ok := g.partImages[frame.ImagePath]
		if !ok || img == nil {
			continue
		}

		g.drawPart(canvas, frame, img, canvasWidth/2, canvasHeight/2)
		count++
	}
	return count
}

func (g *Game) renderWithIgnoreRule(canvas *ebiten.Image, physicalIndex int) int {
	count := 0
	for _, trackName := range g.visualTracks {
		mergedFrames, ok := g.mergedTracks[trackName]
		if !ok || physicalIndex >= len(mergedFrames) {
			continue
		}

		frame := mergedFrames[physicalIndex]

		// 新规则：忽略 f 值
		if frame.ImagePath == "" {
			continue
		}

		img, ok := g.partImages[frame.ImagePath]
		if !ok || img == nil {
			continue
		}

		g.drawPart(canvas, frame, img, canvasWidth/2, canvasHeight/2)
		count++
	}
	return count
}

// renderWithDualAnimation 验证猜想：同时播放 anim_idle(身体) + anim_shooting(头部)
// 头部继承 anim_stem 的位置偏移，实现随身体摆动
func (g *Game) renderWithDualAnimation(canvas *ebiten.Image, physicalIndex int) int {
	count := 0

	// 定义哪些轨道属于"头部动画"（应该使用 anim_shooting）
	headTracks := map[string]bool{
		"anim_face":         true,
		"idle_mouth":        true,
		"anim_blink":        true, // 眨眼动画
		"idle_shoot_blink":  true, // 射击时眨眼
		"anim_sprout":       true,
	}

	// 计算 anim_idle 对应的物理帧（循环播放）
	var idlePhysicalFrame int
	if len(g.idleVisibleFrameIndices) > 0 {
		idleLogicalFrame := g.currentFrame % len(g.idleVisibleFrameIndices)
		idlePhysicalFrame = g.idleVisibleFrameIndices[idleLogicalFrame]
	} else {
		idlePhysicalFrame = physicalIndex
	}

	// 获取 anim_stem 的位置偏移（头部挂载点）
	// anim_stem 的初始位置（物理帧4）: (37.6, 48.7)
	const stemInitX = 37.6
	const stemInitY = 48.7

	var stemOffsetX, stemOffsetY float64
	if stemFrames, ok := g.mergedTracks["anim_stem"]; ok && idlePhysicalFrame < len(stemFrames) {
		stemFrame := stemFrames[idlePhysicalFrame]

		// 计算相对于初始位置的偏移
		currentStemX := stemInitX
		currentStemY := stemInitY

		if stemFrame.X != nil {
			currentStemX = *stemFrame.X
		}
		if stemFrame.Y != nil {
			currentStemY = *stemFrame.Y
		}

		stemOffsetX = currentStemX - stemInitX
		stemOffsetY = currentStemY - stemInitY
	}

	for _, trackName := range g.visualTracks {
		mergedFrames, ok := g.mergedTracks[trackName]
		if !ok {
			continue
		}

		var frame reanim.Frame
		isHeadTrack := headTracks[trackName]

		if isHeadTrack {
			// 头部轨道：使用 anim_shooting 的物理帧
			if physicalIndex >= len(mergedFrames) {
				continue
			}
			frame = mergedFrames[physicalIndex]

			// 严格检查：f=-1 时跳过
			if frame.FrameNum != nil && *frame.FrameNum == -1 {
				continue
			}

			// 关键：头部继承 anim_stem 的位置偏移
			// 头部最终位置 = 头部自身位置 + anim_stem偏移
			if frame.X != nil {
				newX := *frame.X + stemOffsetX
				frame.X = &newX
			}
			if frame.Y != nil {
				newY := *frame.Y + stemOffsetY
				frame.Y = &newY
			}
		} else {
			// 身体轨道：使用 anim_idle 的物理帧
			if idlePhysicalFrame >= len(mergedFrames) {
				continue
			}
			frame = mergedFrames[idlePhysicalFrame]

			// 忽略 f 值（anim_idle 中身体部分也可能有 f=-1）
		}

		if frame.ImagePath == "" {
			continue
		}

		img, ok := g.partImages[frame.ImagePath]
		if !ok || img == nil {
			continue
		}

		g.drawPart(canvas, frame, img, canvasWidth/2, canvasHeight/2)
		count++
	}
	return count
}

func (g *Game) drawPart(canvas *ebiten.Image, frame reanim.Frame, img *ebiten.Image, centerX, centerY float64) {
	x, y := 0.0, 0.0
	if frame.X != nil {
		x = *frame.X
	}
	if frame.Y != nil {
		y = *frame.Y
	}

	scaleX, scaleY := 1.0, 1.0
	if frame.ScaleX != nil {
		scaleX = *frame.ScaleX
	}
	if frame.ScaleY != nil {
		scaleY = *frame.ScaleY
	}

	skewX, skewY := 0.0, 0.0
	if frame.SkewX != nil {
		skewX = *frame.SkewX
	}
	if frame.SkewY != nil {
		skewY = *frame.SkewY
	}

	kxRad := skewX * math.Pi / 180.0
	kyRad := skewY * math.Pi / 180.0

	a := math.Cos(kxRad) * scaleX
	b := math.Sin(kxRad) * scaleX
	c := -math.Sin(kyRad) * scaleY
	d := math.Cos(kyRad) * scaleY

	tx := centerX + x
	ty := centerY + y

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.SetElement(0, 0, a)
	opts.GeoM.SetElement(0, 1, c)
	opts.GeoM.SetElement(1, 0, b)
	opts.GeoM.SetElement(1, 1, d)
	opts.GeoM.Translate(tx, ty)

	canvas.DrawImage(img, opts)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return windowWidth, windowHeight
}

func buildMergedTracks(reanimXML *reanim.ReanimXML, standardFrameCount int) map[string][]reanim.Frame {
	mergedTracks := make(map[string][]reanim.Frame)

	for _, track := range reanimXML.Tracks {
		accX := 0.0
		accY := 0.0
		accSX := 1.0
		accSY := 1.0
		accKX := 0.0
		accKY := 0.0
		accF := 0
		accImg := ""

		mergedFrames := make([]reanim.Frame, standardFrameCount)

		for i := 0; i < standardFrameCount; i++ {
			if i < len(track.Frames) {
				frame := track.Frames[i]

				if frame.X != nil {
					accX = *frame.X
				}
				if frame.Y != nil {
					accY = *frame.Y
				}
				if frame.ScaleX != nil {
					accSX = *frame.ScaleX
				}
				if frame.ScaleY != nil {
					accSY = *frame.ScaleY
				}
				if frame.SkewX != nil {
					accKX = *frame.SkewX
				}
				if frame.SkewY != nil {
					accKY = *frame.SkewY
				}
				if frame.FrameNum != nil {
					accF = *frame.FrameNum
				}
				if frame.ImagePath != "" {
					accImg = frame.ImagePath
				}
			}

			x := accX
			y := accY
			sx := accSX
			sy := accSY
			kx := accKX
			ky := accKY
			f := accF

			mergedFrames[i] = reanim.Frame{
				X:         &x,
				Y:         &y,
				ScaleX:    &sx,
				ScaleY:    &sy,
				SkewX:     &kx,
				SkewY:     &ky,
				FrameNum:  &f,
				ImagePath: accImg,
			}
		}

		mergedTracks[track.Name] = mergedFrames
	}

	return mergedTracks
}

func main() {
	log.Println("=" + string(make([]byte, 78)))
	log.Println("Reanim 动画渲染对比 - 生成截图")
	log.Println("=" + string(make([]byte, 78)))
	log.Println()

	game, err := NewGame()
	if err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	ebiten.SetWindowSize(windowWidth, windowHeight)
	ebiten.SetWindowTitle("Reanim 渲染对比 - anim_shooting")
	ebiten.SetTPS(fps)

	log.Println()
	log.Println("开始播放动画对比...")
	log.Println("  - 左侧: 严格遵守 f=-1（部件 f=-1 时隐藏）")
	log.Println("  - 右侧: 忽略部件 f=-1（只看动画定义轨道）")
	log.Println("  - 按 ESC 退出")
	log.Println()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
