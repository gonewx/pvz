package utils

import (
	"testing"
)

// TestLoadBitmapFont_HouseofTerror28 测试加载 HouseofTerror28 字体
func TestLoadBitmapFont_HouseofTerror28(t *testing.T) {
	// 加载字体
	font, err := LoadBitmapFont(
		"../../assets/data/HouseofTerror28.png",
		"../../assets/data/HouseofTerror28.txt",
	)
	if err != nil {
		t.Fatalf("Failed to load HouseofTerror28: %v", err)
	}

	// 验证字体已加载
	if font == nil {
		t.Fatal("Expected non-nil font")
	}

	if font.Image == nil {
		t.Error("Expected non-nil Image")
	}

	if len(font.CharMap) == 0 {
		t.Error("Expected non-empty CharMap")
	}

	// 验证行高
	if font.LineHeight <= 0 {
		t.Errorf("Expected positive LineHeight, got %d", font.LineHeight)
	}

	t.Logf("Loaded font with %d characters, line height: %d", len(font.CharMap), font.LineHeight)
}

// TestBitmapFont_MeasureText 测试文本宽度测量
func TestBitmapFont_MeasureText(t *testing.T) {
	font, err := LoadBitmapFont(
		"../../assets/data/HouseofTerror28.png",
		"../../assets/data/HouseofTerror28.txt",
	)
	if err != nil {
		t.Skipf("Skipping test: font not available: %v", err)
	}

	tests := []struct {
		text string
		desc string
	}{
		{"Hello", "英文单词"},
		{"123", "数字"},
		{"Hello World!", "含空格和标点"},
		{"点击收集掉落的阳光！", "中文字符（可能不支持）"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			width := font.MeasureText(tt.text)
			t.Logf("Text: %q, Width: %d px", tt.text, width)

			// 验证宽度为非负数
			if width < 0 {
				t.Errorf("Expected non-negative width, got %d", width)
			}
		})
	}
}

// TestBitmapFont_CharacterSupport 测试字符支持
func TestBitmapFont_CharacterSupport(t *testing.T) {
	font, err := LoadBitmapFont(
		"../../assets/data/HouseofTerror28.png",
		"../../assets/data/HouseofTerror28.txt",
	)
	if err != nil {
		t.Skipf("Skipping test: font not available: %v", err)
	}

	// 测试常用字符（字母和数字是必须支持的）
	// 注意：HouseofTerror28 字体的空字符 '' 被解析为空格，但它实际上是一个占位符
	// 某些标点符号可能不在字体中，我们只测试确定存在的字符
	commonChars := []rune{
		'A', 'Z', 'a', 'z', '0', '9',
		'!', // 感叹号在字符列表中
	}

	for _, char := range commonChars {
		if _, ok := font.CharMap[char]; !ok {
			t.Errorf("Expected font to support character: %q", char)
		}
	}

	// 记录字体支持的标点符号情况（不作为失败条件）
	optionalChars := []rune{' ', '?', '.', ','}
	for _, char := range optionalChars {
		if _, ok := font.CharMap[char]; !ok {
			t.Logf("Note: Font does not support optional character: %q", char)
		}
	}
}
