package game

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLawnStrings_Load 验证 LawnStrings.txt 文件加载
func TestLawnStrings_Load(t *testing.T) {
	// 测试实际的 LawnStrings.txt 文件
	filePath := "../../assets/properties/LawnStrings.txt"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Skipf("Skipping test: LawnStrings.txt not found at %s", filePath)
	}

	ls, err := NewLawnStrings(filePath)
	if err != nil {
		t.Fatalf("Failed to load LawnStrings.txt: %v", err)
	}

	if ls == nil {
		t.Fatal("Expected non-nil LawnStrings")
	}

	if ls.strings == nil {
		t.Fatal("Expected non-nil strings map")
	}

	// 验证至少加载了一些字符串
	if len(ls.strings) == 0 {
		t.Error("Expected at least some strings to be loaded")
	}

	t.Logf("Loaded %d strings from LawnStrings.txt", len(ls.strings))
}

// TestLawnStrings_GetString 验证文本获取
func TestLawnStrings_GetString(t *testing.T) {
	// 创建临时测试文件
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_strings.txt")

	content := `[ADVICE_CLICK_ON_SUN]
点击收集掉落的阳光！

[ADVICE_CLICKED_ON_SUN]
继续收集阳光！你需要他们来种下更多的植物！

[ADVICE_CLICK_SEED_PACKET]
点击拾取种子包！
`

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 加载测试文件
	ls, err := NewLawnStrings(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load test strings: %v", err)
	}

	// 测试用例
	tests := []struct {
		key      string
		expected string
	}{
		{
			key:      "ADVICE_CLICK_ON_SUN",
			expected: "点击收集掉落的阳光！",
		},
		{
			key:      "ADVICE_CLICKED_ON_SUN",
			expected: "继续收集阳光！你需要他们来种下更多的植物！",
		},
		{
			key:      "ADVICE_CLICK_SEED_PACKET",
			expected: "点击拾取种子包！",
		},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := ls.GetString(tt.key)
			if result != tt.expected {
				t.Errorf("GetString(%q) = %q, want %q", tt.key, result, tt.expected)
			}
		})
	}
}

// TestLawnStrings_MissingKey 验证缺失键处理
func TestLawnStrings_MissingKey(t *testing.T) {
	// 创建临时测试文件
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_strings.txt")

	content := `[EXISTING_KEY]
存在的文本
`

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 加载测试文件
	ls, err := NewLawnStrings(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load test strings: %v", err)
	}

	// 测试不存在的键
	missingKey := "NON_EXISTENT_KEY"
	result := ls.GetString(missingKey)
	expected := "[NON_EXISTENT_KEY]"

	if result != expected {
		t.Errorf("GetString(%q) = %q, want %q (debug format)", missingKey, result, expected)
	}
}

// TestLawnStrings_RealFile 验证真实文件中的教学文本
func TestLawnStrings_RealFile(t *testing.T) {
	// 测试实际的 LawnStrings.txt 文件
	filePath := "../../assets/properties/LawnStrings.txt"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Skipf("Skipping test: LawnStrings.txt not found at %s", filePath)
	}

	ls, err := NewLawnStrings(filePath)
	if err != nil {
		t.Fatalf("Failed to load LawnStrings.txt: %v", err)
	}

	// 验证所有教学文本键都存在
	requiredKeys := []string{
		"ADVICE_CLICK_ON_SUN",
		"ADVICE_CLICKED_ON_SUN",
		"ADVICE_CLICK_SEED_PACKET",
		"ADVICE_CLICK_ON_GRASS",
		"ADVICE_PLANTED_PEASHOOTER",
		"ADVICE_ZOMBIE_ONSLAUGHT",
	}

	for _, key := range requiredKeys {
		text := ls.GetString(key)
		// 如果返回 [KEY] 格式，说明键不存在
		if text == "["+key+"]" {
			t.Errorf("Required tutorial text key %q not found in LawnStrings.txt", key)
		} else {
			t.Logf("%s = %q", key, text)
		}
	}
}
