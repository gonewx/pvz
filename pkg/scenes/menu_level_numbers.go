package scenes

import (
	"image"
	"log"
	"strconv"

	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
)

// renderLevelNumbers 在指定位置渲染关卡进度数字（如 "1-4"）
// 参数：
//   - screen: 渲染目标
//   - rm: 资源管理器
//   - levelText: 关卡文本（如 "1-4"）
//   - x, y: 渲染中心位置
//   - angle: 旋转角度（弧度，负值表示逆时针旋转）
func renderLevelNumbers(screen *ebiten.Image, rm *game.ResourceManager, levelText string, x, y, angle float64) {
	if levelText == "" {
		return
	}

	// 加载数字精灵图
	spriteSheet, err := rm.LoadImageByID("IMAGE_SELECTORSCREEN_LEVELNUMBERS")
	if err != nil || spriteSheet == nil {
		log.Printf("[LevelNumbers] ❌ 加载精灵图失败: %v", err)
		return
	}

	// 精灵图尺寸：120x17，10 列（数字 0-9）
	const (
		spriteWidth  = 120
		spriteHeight = 17
		numDigits    = 10
	)

	// 连字符的宽度（缩小以避免过宽）
	const dashWidth = 8.0

	// 数字实际渲染宽度（去掉左右边距，使数字更紧凑）
	const digitRenderWidth = 8.0 // 原本 12，缩小到 8

	// 计算总宽度（用于居中）
	totalWidth := 0.0
	for _, char := range levelText {
		if char == '-' {
			totalWidth += dashWidth
		} else {
			totalWidth += digitRenderWidth // 使用实际渲染宽度
		}
	}

	// 当前渲染位置（从左边缘开始）
	currentX := -totalWidth / 2

	// 逐字符渲染
	for _, char := range levelText {
		if char == '-' {
			// 渲染连字符（留空间）
			currentX += dashWidth
			continue
		}

		// 解析数字
		digit, err := strconv.Atoi(string(char))
		if err != nil {
			// 不是数字，跳过
			continue
		}

		// 从精灵图中裁剪对应数字
		srcX := digit * (spriteWidth / numDigits)
		srcRect := image.Rect(srcX, 0, srcX+(spriteWidth/numDigits), spriteHeight)
		digitImg := spriteSheet.SubImage(srcRect).(*ebiten.Image)

		// 应用变换：旋转 + 平移
		opts := &ebiten.DrawImageOptions{}

		// 1. 先平移到相对位置（相对于中心）
		opts.GeoM.Translate(currentX, -float64(spriteHeight)/2)

		// 2. 应用旋转（绕原点旋转）
		opts.GeoM.Rotate(angle)

		// 3. 平移到最终位置
		opts.GeoM.Translate(x, y)

		// 渲染到屏幕
		screen.DrawImage(digitImg, opts)

		// 移动到下一个位置（使用实际渲染宽度）
		currentX += digitRenderWidth
	}
}
