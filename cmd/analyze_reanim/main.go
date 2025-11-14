package main

import (
	"fmt"
	"log"
	"math"
	"os"

	"github.com/decker502/pvz/internal/reanim"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run cmd/analyze_reanim/main.go <reanim文件路径>")
		os.Exit(1)
	}

	reanimFile := os.Args[1]

	// 解析 reanim 文件
	reanimXML, err := reanim.ParseReanimFile(reanimFile)
	if err != nil {
		log.Fatalf("解析失败: %v", err)
	}

	fmt.Printf("动画文件: %s\n", reanimFile)
	fmt.Printf("FPS: %d\n", reanimXML.FPS)
	fmt.Printf("轨道数量: %d\n\n", len(reanimXML.Tracks))

	// 构建合并轨道
	mergedTracks := reanim.BuildMergedTracks(reanimXML)

	// 计算 BoundingBox（第一帧）
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64

	frameIndex := 0
	count := 0

	for trackName, frames := range mergedTracks {
		if frameIndex >= len(frames) {
			continue
		}

		frame := frames[frameIndex]

		if frame.ImagePath == "" {
			continue
		}

		x := 0.0
		if frame.X != nil {
			x = *frame.X
		}
		y := 0.0
		if frame.Y != nil {
			y = *frame.Y
		}

		// 假设图片是 100x100（这里简化，实际需要加载图片）
		// 但我们只关心坐标范围
		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}

		count++

		if count <= 10 {
			fmt.Printf("  轨道: %-40s  坐标: (%.1f, %.1f)\n", trackName, x, y)
		}
	}

	if count == 0 {
		fmt.Println("没有找到有效的帧数据")
		return
	}

	width := maxX - minX
	height := maxY - minY
	centerX := (minX + maxX) / 2.0
	centerY := (minY + maxY) / 2.0

	fmt.Printf("\n=== BoundingBox 分析（第一帧）===\n")
	fmt.Printf("X 范围: %.1f ~ %.1f (宽度: %.1f)\n", minX, maxX, width)
	fmt.Printf("Y 范围: %.1f ~ %.1f (高度: %.1f)\n", minY, maxY, height)
	fmt.Printf("CenterOffset: (%.1f, %.1f)\n\n", centerX, centerY)

	fmt.Printf("虚拟显示区域: 800x600\n")
	fmt.Printf("动画 BoundingBox: %.1fx%.1f\n\n", width, height)

	if height > 600 {
		fmt.Printf("⚠️  动画高度 (%.1f) 超过虚拟显示区域高度 (600)\n", height)
		fmt.Printf("   超出: %.1f 像素 (%.1f%%)\n\n", height-600, (height/600-1)*100)

		fmt.Printf("当使用中心锚点时:\n")
		fmt.Printf("  - 虚拟区域中心 Y = 300\n")
		fmt.Printf("  - CenterOffset Y = %.1f\n", centerY)
		fmt.Printf("  - 渲染原点 Y = 300 - %.1f = %.1f\n", centerY, 300-centerY)
		fmt.Printf("  - 动画顶部 Y = %.1f + %.1f = %.1f\n", 300-centerY, minY, 300-centerY+minY)
		fmt.Printf("  - 动画底部 Y = %.1f + %.1f = %.1f\n\n", 300-centerY, maxY, 300-centerY+maxY)

		topY := 300 - centerY + minY
		bottomY := 300 - centerY + maxY

		if topY < 0 {
			fmt.Printf("  ❌ 动画顶部超出虚拟区域上边界 %.1f 像素\n", -topY)
		}
		if bottomY > 600 {
			fmt.Printf("  ❌ 动画底部超出虚拟区域下边界 %.1f 像素\n", bottomY-600)
		}
	}

	if width > 800 {
		fmt.Printf("\n⚠️  动画宽度 (%.1f) 超过虚拟显示区域宽度 (800)\n", width)
		fmt.Printf("   超出: %.1f 像素\n", width-800)
	}
}
