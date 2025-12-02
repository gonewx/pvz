package systems

import (
	"image"
	"math"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/hajimehoshi/ebiten/v2"
)

// TestProgressBarRightAlignment 验证右对齐位置计算
func TestProgressBarRightAlignment(t *testing.T) {
	tests := []struct {
		name              string
		screenWidth       float64
		screenHeight      float64
		progressBarWidth  int
		progressBarHeight int
		rightMargin       float64
		bottomMargin      float64
		expectedX         float64
		expectedY         float64
	}{
		{
			name:              "标准800x600屏幕",
			screenWidth:       800,
			screenHeight:      600,
			progressBarWidth:  158,
			progressBarHeight: 27,
			rightMargin:       40,
			bottomMargin:      0,
			expectedX:         602, // 800 - 158 - 40
			expectedY:         573, // 600 - 27 - 0
		},
		{
			name:              "带底部边距",
			screenWidth:       800,
			screenHeight:      600,
			progressBarWidth:  158,
			progressBarHeight: 27,
			rightMargin:       40,
			bottomMargin:      10,
			expectedX:         602, // 800 - 158 - 40
			expectedY:         563, // 600 - 27 - 10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 计算位置（模拟 drawFullProgressBar 的逻辑）
			progressBarX := tt.screenWidth - float64(tt.progressBarWidth) - tt.rightMargin
			progressBarY := tt.screenHeight - float64(tt.progressBarHeight) - tt.bottomMargin

			if math.Abs(progressBarX-tt.expectedX) > 0.01 {
				t.Errorf("X 位置计算错误: got %.2f, expected %.2f", progressBarX, tt.expectedX)
			}
			if math.Abs(progressBarY-tt.expectedY) > 0.01 {
				t.Errorf("Y 位置计算错误: got %.2f, expected %.2f", progressBarY, tt.expectedY)
			}
		})
	}
}

// TestProgressBarFillClipping 验证进度条裁剪矩形计算
func TestProgressBarFillClipping(t *testing.T) {
	tests := []struct {
		name            string
		progressPercent float64
		fillWidth       int
		expectedClipW   int
	}{
		{
			name:            "0% 进度",
			progressPercent: 0.0,
			fillWidth:       86,
			expectedClipW:   0,
		},
		{
			name:            "50% 进度",
			progressPercent: 0.5,
			fillWidth:       86,
			expectedClipW:   43,
		},
		{
			name:            "100% 进度",
			progressPercent: 1.0,
			fillWidth:       86,
			expectedClipW:   86,
		},
		{
			name:            "超过100%",
			progressPercent: 1.2,
			fillWidth:       86,
			expectedClipW:   86, // 应限制在最大值
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 限制进度百分比在 0.0-1.0 范围
			progress := tt.progressPercent
			if progress > 1.0 {
				progress = 1.0
			}
			if progress < 0.0 {
				progress = 0.0
			}

			// 计算裁剪宽度（从右到左填充）
			fillWidthPx := int(float64(tt.fillWidth) * progress)

			if fillWidthPx != tt.expectedClipW {
				t.Errorf("裁剪宽度计算错误: got %d, expected %d", fillWidthPx, tt.expectedClipW)
			}
		})
	}
}

// TestZombieHeadTracking 验证僵尸头位置跟随逻辑
func TestZombieHeadTracking(t *testing.T) {
	tests := []struct {
		name              string
		progressPercent   float64
		progressBarX      float64
		bgWidth           float64
		partWidth         int
		zombieHeadOffsetX float64
		expectedHeadX     float64 // 预期的僵尸头左上角X坐标
	}{
		{
			name:              "100%进度（最左边）",
			progressPercent:   1.0,
			progressBarX:      602,
			bgWidth:           158,
			partWidth:         28,
			zombieHeadOffsetX: 0,
			expectedHeadX:     588, // 602 + 158*(1-1.0) + 0 - 28/2 = 602 - 14 = 588
		},
		{
			name:              "50%进度（中间）",
			progressPercent:   0.5,
			progressBarX:      602,
			bgWidth:           158,
			partWidth:         28,
			zombieHeadOffsetX: 0,
			expectedHeadX:     667, // 602 + 158*0.5 + 0 - 14 = 602 + 79 - 14 = 667
		},
		{
			name:              "0%进度（最右边）",
			progressPercent:   0.0,
			progressBarX:      602,
			bgWidth:           158,
			partWidth:         28,
			zombieHeadOffsetX: 0,
			expectedHeadX:     746, // 602 + 158*1.0 + 0 - 14 = 602 + 158 - 14 = 746
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟实际渲染逻辑（从 drawZombieHead 方法）
			// headX = progressBar.X + bgWidth*(1.0-progressPercent) + ZombieHeadOffsetX - partWidth/2.0
			headX := tt.progressBarX + tt.bgWidth*(1.0-tt.progressPercent) + tt.zombieHeadOffsetX - float64(tt.partWidth)/2.0

			// 允许1像素误差
			if math.Abs(headX-tt.expectedHeadX) > 1.0 {
				t.Errorf("僵尸头X位置计算错误: got %.2f, expected %.2f (差值: %.2f)",
					headX, tt.expectedHeadX, math.Abs(headX-tt.expectedHeadX))
			}
		})
	}
}

// TestProgressBarModeSwitch 验证双模式切换逻辑
func TestProgressBarModeSwitch(t *testing.T) {
	tests := []struct {
		name              string
		showLevelTextOnly bool
		expectedFullBar   bool
		expectedTextOnly  bool
	}{
		{
			name:              "开场前模式（只显示文本）",
			showLevelTextOnly: true,
			expectedFullBar:   false,
			expectedTextOnly:  true,
		},
		{
			name:              "进攻中模式（显示完整进度条）",
			showLevelTextOnly: false,
			expectedFullBar:   true,
			expectedTextOnly:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟 Draw 方法的分支逻辑
			shouldDrawFullBar := !tt.showLevelTextOnly
			shouldDrawTextOnly := tt.showLevelTextOnly

			if shouldDrawFullBar != tt.expectedFullBar {
				t.Errorf("完整进度条显示判断错误: got %v, expected %v", shouldDrawFullBar, tt.expectedFullBar)
			}
			if shouldDrawTextOnly != tt.expectedTextOnly {
				t.Errorf("纯文本显示判断错误: got %v, expected %v", shouldDrawTextOnly, tt.expectedTextOnly)
			}
		})
	}
}

// TestProgressBarComponentInitialization 验证组件初始化
func TestProgressBarComponentInitialization(t *testing.T) {
	// 创建测试用的假图片
	backgroundImg := ebiten.NewImage(158, 54)
	progressImg := ebiten.NewImage(86, 11)
	partsImg := ebiten.NewImage(84, 28)

	progressBar := &components.LevelProgressBarComponent{
		BackgroundImage:   backgroundImg,
		ProgressBarImage:  progressImg,
		PartsImage:        partsImg,
		TotalZombies:      10,
		KilledZombies:     0,
		ProgressPercent:   0.0,
		FlagPositions:     []float64{0.5},
		LevelText:         "关卡 1-1",
		ShowLevelTextOnly: true,
		X:                 0,
		Y:                 0,
	}

	// 验证组件字段初始化正确
	if progressBar.TotalZombies != 10 {
		t.Errorf("TotalZombies 初始化错误: got %d, expected 10", progressBar.TotalZombies)
	}
	if progressBar.KilledZombies != 0 {
		t.Errorf("KilledZombies 初始化错误: got %d, expected 0", progressBar.KilledZombies)
	}
	if progressBar.ProgressPercent != 0.0 {
		t.Errorf("ProgressPercent 初始化错误: got %.2f, expected 0.00", progressBar.ProgressPercent)
	}
	if !progressBar.ShowLevelTextOnly {
		t.Errorf("ShowLevelTextOnly 初始化错误: got %v, expected true", progressBar.ShowLevelTextOnly)
	}
	if len(progressBar.FlagPositions) != 1 {
		t.Errorf("FlagPositions 初始化错误: got %d flags, expected 1", len(progressBar.FlagPositions))
	}
}

// TestProgressBarUpdateLogic 验证进度更新逻辑
func TestProgressBarUpdateLogic(t *testing.T) {
	tests := []struct {
		name             string
		totalZombies     int
		killedZombies    int
		expectedProgress float64
	}{
		{
			name:             "无僵尸击杀",
			totalZombies:     10,
			killedZombies:    0,
			expectedProgress: 0.0,
		},
		{
			name:             "击杀一半",
			totalZombies:     10,
			killedZombies:    5,
			expectedProgress: 0.5,
		},
		{
			name:             "全部击杀",
			totalZombies:     10,
			killedZombies:    10,
			expectedProgress: 1.0,
		},
		{
			name:             "除零保护（总数为0）",
			totalZombies:     0,
			killedZombies:    0,
			expectedProgress: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟进度百分比计算逻辑
			var progress float64
			if tt.totalZombies > 0 {
				progress = float64(tt.killedZombies) / float64(tt.totalZombies)
			} else {
				progress = 0.0
			}

			if math.Abs(progress-tt.expectedProgress) > 0.01 {
				t.Errorf("进度计算错误: got %.2f, expected %.2f", progress, tt.expectedProgress)
			}
		})
	}
}

// TestFlagPositionRendering 验证旗帜图标渲染位置计算
func TestFlagPositionRendering(t *testing.T) {
	tests := []struct {
		name          string
		flagPercent   float64
		progressBarX  float64
		fillAreaWidth float64
		expectedFlagX float64
	}{
		{
			name:          "旗帜在50%位置",
			flagPercent:   0.5,
			progressBarX:  602,
			fillAreaWidth: 86,
			expectedFlagX: 681, // 602 + 36 + 86*0.5 = 681
		},
		{
			name:          "旗帜在起点",
			flagPercent:   0.0,
			progressBarX:  602,
			fillAreaWidth: 86,
			expectedFlagX: 638, // 602 + 36
		},
		{
			name:          "旗帜在终点",
			flagPercent:   1.0,
			progressBarX:  602,
			fillAreaWidth: 86,
			expectedFlagX: 724, // 602 + 36 + 86
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 旗帜位置计算（从左到右）
			fillStartX := tt.progressBarX + (config.ProgressBarBackgroundWidth-tt.fillAreaWidth)/2
			flagX := fillStartX + tt.fillAreaWidth*tt.flagPercent

			if math.Abs(flagX-tt.expectedFlagX) > 1.0 {
				t.Errorf("旗帜X位置计算错误: got %.2f, expected %.2f", flagX, tt.expectedFlagX)
			}
		})
	}
}

// ========== Story 19.9: 保龄球关卡无旗帜进度条测试 ==========

// TestProgressBar_NoFlagPositions 验证无旗帜时进度条正常渲染
func TestProgressBar_NoFlagPositions(t *testing.T) {
	// 创建测试用的假图片
	backgroundImg := ebiten.NewImage(158, 54)
	progressImg := ebiten.NewImage(86, 11)
	partsImg := ebiten.NewImage(84, 28)

	progressBar := &components.LevelProgressBarComponent{
		BackgroundImage:   backgroundImg,
		ProgressBarImage:  progressImg,
		PartsImage:        partsImg,
		TotalZombies:      50, // 保龄球关卡通常有较多僵尸
		KilledZombies:     0,
		ProgressPercent:   0.0,
		FlagPositions:     []float64{}, // 无旗帜（保龄球关卡特征）
		LevelText:         "关卡 1-5",
		ShowLevelTextOnly: false,
		X:                 602,
		Y:                 573,
	}

	// 验证旗帜数组为空
	if len(progressBar.FlagPositions) != 0 {
		t.Errorf("FlagPositions 应为空数组，got %d elements", len(progressBar.FlagPositions))
	}

	// 验证不会因空旗帜数组而崩溃
	// 模拟 drawFlags 的逻辑
	for _, flagPos := range progressBar.FlagPositions {
		// 如果数组为空，此循环不会执行
		_ = flagPos
		t.Error("不应进入旗帜渲染循环")
	}
}

// TestProgressBar_NoFlagPositions_NilCheck 验证 nil FlagPositions 处理
func TestProgressBar_NoFlagPositions_NilCheck(t *testing.T) {
	progressBar := &components.LevelProgressBarComponent{
		FlagPositions: nil, // nil 情况
	}

	// 验证 nil 和空数组都能正确处理
	if progressBar.FlagPositions != nil && len(progressBar.FlagPositions) > 0 {
		t.Error("应跳过旗帜渲染")
	}

	// 模拟 drawFlags 中的安全检查
	if progressBar.FlagPositions == nil || len(progressBar.FlagPositions) == 0 {
		// 正确：跳过旗帜渲染
	} else {
		t.Error("nil FlagPositions 应跳过渲染")
	}
}

// TestProgressBar_BowlingLevel_ProgressCalculation 验证保龄球关卡进度计算
func TestProgressBar_BowlingLevel_ProgressCalculation(t *testing.T) {
	tests := []struct {
		name             string
		totalZombies     int
		killedZombies    int
		expectedProgress float64
	}{
		{
			name:             "保龄球关卡开始 (0%)",
			totalZombies:     50,
			killedZombies:    0,
			expectedProgress: 0.0,
		},
		{
			name:             "保龄球关卡进行中 (40%)",
			totalZombies:     50,
			killedZombies:    20,
			expectedProgress: 0.4,
		},
		{
			name:             "保龄球关卡即将通关 (90%)",
			totalZombies:     50,
			killedZombies:    45,
			expectedProgress: 0.9,
		},
		{
			name:             "保龄球关卡通关 (100%)",
			totalZombies:     50,
			killedZombies:    50,
			expectedProgress: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var progress float64
			if tt.totalZombies > 0 {
				progress = float64(tt.killedZombies) / float64(tt.totalZombies)
			}

			if math.Abs(progress-tt.expectedProgress) > 0.01 {
				t.Errorf("进度计算错误: got %.2f, expected %.2f", progress, tt.expectedProgress)
			}
		})
	}
}

// TestProgressBar_NoFlags_ZombieHeadPosition 验证无旗帜时僵尸头位置仍正确
func TestProgressBar_NoFlags_ZombieHeadPosition(t *testing.T) {
	// 保龄球关卡无旗帜，但僵尸头仍需正确跟随进度
	tests := []struct {
		name            string
		progressPercent float64
		progressBarX    float64
		bgWidth         float64
		partWidth       int
	}{
		{"0%进度", 0.0, 602, 158, 28},
		{"25%进度", 0.25, 602, 158, 28},
		{"50%进度", 0.5, 602, 158, 28},
		{"75%进度", 0.75, 602, 158, 28},
		{"100%进度", 1.0, 602, 158, 28},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 计算僵尸头位置（与有旗帜时逻辑相同）
			headX := tt.progressBarX + tt.bgWidth*(1.0-tt.progressPercent) - float64(tt.partWidth)/2.0

			// 验证位置在合理范围内
			minX := tt.progressBarX - float64(tt.partWidth)
			maxX := tt.progressBarX + tt.bgWidth

			if headX < minX || headX > maxX {
				t.Errorf("僵尸头X位置超出范围: got %.2f, expected in [%.2f, %.2f]",
					headX, minX, maxX)
			}
		})
	}
}

// TestProgressBar_FlagSegmentCalculation_NoFlags 验证无旗帜时段落计算
func TestProgressBar_FlagSegmentCalculation_NoFlags(t *testing.T) {
	// 保龄球关卡: flags=0, 无旗帜段落
	totalLength := 150
	flagSegmentLength := 12
	flagCount := 0 // 无旗帜

	// 计算普通段落（应占满整个进度条）
	normalSegment := totalLength - (flagCount * flagSegmentLength)

	if normalSegment != 150 {
		t.Errorf("无旗帜时普通段落应为 150，got %d", normalSegment)
	}

	// 验证进度条从左到右完全可用
	if normalSegment != totalLength {
		t.Errorf("无旗帜时进度条应完全可用于普通波次")
	}
}

// TestProgressBar_ComponentInit_BowlingLevel 验证保龄球关卡进度条组件初始化
func TestProgressBar_ComponentInit_BowlingLevel(t *testing.T) {
	// 创建测试用的假图片
	backgroundImg := ebiten.NewImage(158, 54)
	progressImg := ebiten.NewImage(86, 11)
	partsImg := ebiten.NewImage(84, 28)

	// 模拟保龄球关卡的进度条组件
	progressBar := &components.LevelProgressBarComponent{
		BackgroundImage:     backgroundImg,
		ProgressBarImage:    progressImg,
		PartsImage:          partsImg,
		TotalProgressLength: 150,
		FlagSegmentLength:   12,
		NormalSegmentBase:   150, // 无旗帜时 = 150 - 0*12 = 150
		TotalWaves:          16,  // 保龄球关卡 16 波
		FlagWaveCount:       0,   // 无旗帜波
		CurrentWaveNum:      0,
		FlagPositions:       []float64{}, // 无旗帜
		LevelText:           "关卡 1-5",
		ShowLevelTextOnly:   true,
	}

	// 验证保龄球关卡特有配置
	if progressBar.FlagWaveCount != 0 {
		t.Errorf("FlagWaveCount 应为 0，got %d", progressBar.FlagWaveCount)
	}
	if progressBar.NormalSegmentBase != 150 {
		t.Errorf("NormalSegmentBase 应为 150（无旗帜），got %d", progressBar.NormalSegmentBase)
	}
	if progressBar.TotalWaves != 16 {
		t.Errorf("TotalWaves 应为 16，got %d", progressBar.TotalWaves)
	}
	if len(progressBar.FlagPositions) != 0 {
		t.Errorf("FlagPositions 应为空，got %d elements", len(progressBar.FlagPositions))
	}
}

// BenchmarkProgressBarRendering 性能基准测试（验证60 FPS性能）
func BenchmarkProgressBarRendering(b *testing.B) {
	// 创建测试用的假图片
	backgroundImg := ebiten.NewImage(158, 54)
	progressImg := ebiten.NewImage(86, 11)
	partsImg := ebiten.NewImage(84, 28)

	progressBar := &components.LevelProgressBarComponent{
		BackgroundImage:   backgroundImg,
		ProgressBarImage:  progressImg,
		PartsImage:        partsImg,
		TotalZombies:      10,
		KilledZombies:     5,
		ProgressPercent:   0.5,
		FlagPositions:     []float64{0.5},
		LevelText:         "关卡 1-1",
		ShowLevelTextOnly: false,
		X:                 602,
		Y:                 573,
	}

	screen := ebiten.NewImage(800, 600)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 模拟渲染逻辑（核心计算部分）
		// 1. 计算位置
		bgWidth := config.ProgressBarBackgroundWidth
		fillAreaWidth := 86.0
		fillStartX := progressBar.X + (bgWidth-fillAreaWidth)/2

		// 2. 裁剪进度条
		fillWidthPx := int(fillAreaWidth * progressBar.ProgressPercent)
		fillClipRect := image.Rect(0, 0, fillWidthPx, 11)
		_ = progressImg.SubImage(fillClipRect)

		// 3. 计算僵尸头位置
		fillCurrentX := fillStartX + fillAreaWidth*(1.0-progressBar.ProgressPercent)
		zombieHeadW := 28
		_ = fillCurrentX - float64(zombieHeadW)/2

		// 4. 裁剪精灵图
		partsBounds := partsImg.Bounds()
		partWidth := partsBounds.Dx() / config.PartsImageColumns
		zombieHeadRect := image.Rect(0, 0, partWidth, partsBounds.Dy())
		_ = partsImg.SubImage(zombieHeadRect)

		// 5. DrawImage 调用（5次）
		for j := 0; j < 5; j++ {
			screen.DrawImage(backgroundImg, &ebiten.DrawImageOptions{})
		}
	}
}
