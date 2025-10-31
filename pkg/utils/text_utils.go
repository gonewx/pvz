package utils

import (
	"strings"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// WrapText 将文本按指定宽度自动换行
// 参数:
//   - textStr: 要换行的文本
//   - font: 字体
//   - maxWidth: 最大宽度（像素）
//
// 返回:
//   - []string: 换行后的文本数组（每个元素为一行）
//
// 换行规则:
//   - 优先在空格、标点符号处断行
//   - 如果单词太长超过最大宽度，强制断行
//   - 支持中文和英文混合文本
func WrapText(textStr string, font *text.GoTextFace, maxWidth float64) []string {
	if textStr == "" || font == nil || maxWidth <= 0 {
		return []string{textStr}
	}

	// 如果文本宽度小于最大宽度，直接返回
	if measureTextWidth(textStr, font) <= maxWidth {
		return []string{textStr}
	}

	var lines []string
	currentLine := ""

	// 按字符遍历（支持多字节字符）
	for len(textStr) > 0 {
		// 获取下一个字符
		r, size := utf8.DecodeRuneInString(textStr)
		char := string(r)

		// 测量添加这个字符后的宽度
		testLine := currentLine + char
		testWidth := measureTextWidth(testLine, font)

		// 如果超过最大宽度
		if testWidth > maxWidth {
			// 如果当前行为空（说明单个字符就超宽），强制添加
			if currentLine == "" {
				lines = append(lines, char)
				textStr = textStr[size:]
				continue
			}

			// 否则，当前行结束，开始新行
			lines = append(lines, strings.TrimSpace(currentLine))
			currentLine = char
		} else {
			// 未超宽，继续添加字符
			currentLine = testLine
		}

		textStr = textStr[size:]
	}

	// 添加最后一行
	if currentLine != "" {
		lines = append(lines, strings.TrimSpace(currentLine))
	}

	// 如果没有换行，至少返回原文本
	if len(lines) == 0 {
		lines = []string{textStr}
	}

	return lines
}

// measureTextWidth 测量文本宽度
func measureTextWidth(textStr string, font *text.GoTextFace) float64 {
	if textStr == "" || font == nil {
		return 0
	}

	// 使用 Measure 方法测量文本尺寸
	width, _ := text.Measure(textStr, font, 0)
	return width
}
