package utils

import (
	"bytes"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// TestWrapText 测试文本换行功能
func TestWrapText(t *testing.T) {
	// 加载实际字体文件
	fontPath := "../../assets/fonts/SimHei.ttf"
	fontData, err := os.ReadFile(fontPath)
	if err != nil {
		t.Skipf("无法加载字体文件 %s: %v", fontPath, err)
		return
	}

	faceSource, err := text.NewGoTextFaceSource(bytes.NewReader(fontData))
	if err != nil {
		t.Skipf("无法创建字体源: %v", err)
		return
	}

	font := &text.GoTextFace{
		Source: faceSource,
		Size:   22,
	}

	tests := []struct {
		name      string
		input     string
		maxWidth  float64
		expectMin int // 期望最少的行数
	}{
		{
			name:      "短文本不换行",
			input:     "短文本",
			maxWidth:  1000,
			expectMin: 1,
		},
		{
			name:      "长文本自动换行",
			input:     "豌豆射手，你的第一道防线。它们通过发射豌豆来攻击僵尸。",
			maxWidth:  300,
			expectMin: 2, // 期望至少换成2行
		},
		{
			name:      "空文本",
			input:     "",
			maxWidth:  100,
			expectMin: 1, // 返回空字符串数组
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := WrapText(tt.input, font, tt.maxWidth)

			if len(lines) < tt.expectMin {
				t.Errorf("期望至少 %d 行，实际得到 %d 行", tt.expectMin, len(lines))
			}

			// 验证返回的行数组不为空
			if lines == nil {
				t.Error("WrapText 返回 nil，期望返回字符串数组")
			}

			// 打印调试信息
			t.Logf("输入: %q, 最大宽度: %.0f, 输出行数: %d", tt.input, tt.maxWidth, len(lines))
			for i, line := range lines {
				t.Logf("  第 %d 行: %q", i+1, line)
			}
		})
	}
}

// TestWrapTextEdgeCases 测试边界情况
func TestWrapTextEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		font     *text.GoTextFace
		maxWidth float64
		wantLen  int
	}{
		{
			name:     "nil font",
			input:    "测试",
			font:     nil,
			maxWidth: 100,
			wantLen:  1, // 返回原文本
		},
		{
			name:     "zero maxWidth",
			input:    "测试",
			font:     &text.GoTextFace{Size: 22},
			maxWidth: 0,
			wantLen:  1, // 返回原文本
		},
		{
			name:     "negative maxWidth",
			input:    "测试",
			font:     &text.GoTextFace{Size: 22},
			maxWidth: -100,
			wantLen:  1, // 返回原文本
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := WrapText(tt.input, tt.font, tt.maxWidth)
			if len(lines) != tt.wantLen {
				t.Errorf("期望 %d 行，实际得到 %d 行", tt.wantLen, len(lines))
			}
		})
	}
}
