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

// TestLawnStrings_KeysWithTags 验证带尾部标签的键能正确解析
// 例如 [CRAZY_DAVE_2412] {SHOW_WALLNUT} 应该被解析为键 "CRAZY_DAVE_2412"
func TestLawnStrings_KeysWithTags(t *testing.T) {
	// 创建临时测试文件，包含带标签的键
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_strings.txt")

	content := `[CRAZY_DAVE_2412] {SHOW_WALLNUT}
嘿，拿好这个坚果墙！

[CRAZY_DAVE_2414] {SHAKE}
因为我发~~~疯了！！！！！{MOUTH_BIG_SMILE}

[NORMAL_KEY]
普通文本
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
			key:      "CRAZY_DAVE_2412",
			expected: "嘿，拿好这个坚果墙！",
		},
		{
			key:      "CRAZY_DAVE_2414",
			expected: "因为我发~~~疯了！！！！！{MOUTH_BIG_SMILE}",
		},
		{
			key:      "NORMAL_KEY",
			expected: "普通文本",
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

	// 验证 key 行标签
	keyTagTests := []struct {
		key          string
		expectedTags []string
	}{
		{
			key:          "CRAZY_DAVE_2412",
			expectedTags: []string{"SHOW_WALLNUT"},
		},
		{
			key:          "CRAZY_DAVE_2414",
			expectedTags: []string{"SHAKE"},
		},
		{
			key:          "NORMAL_KEY",
			expectedTags: nil, // 无标签
		},
	}

	for _, tt := range keyTagTests {
		t.Run(tt.key+"_tags", func(t *testing.T) {
			tags := ls.GetKeyTags(tt.key)
			if len(tags) != len(tt.expectedTags) {
				t.Errorf("GetKeyTags(%q) = %v, want %v", tt.key, tags, tt.expectedTags)
				return
			}
			for i, tag := range tags {
				if tag != tt.expectedTags[i] {
					t.Errorf("GetKeyTags(%q)[%d] = %q, want %q", tt.key, i, tag, tt.expectedTags[i])
				}
			}
		})
	}
}

// TestLawnStrings_CrazyDaveDialogue 验证疯狂戴夫对话文本
func TestLawnStrings_CrazyDaveDialogue(t *testing.T) {
	// 测试实际的 LawnStrings.txt 文件
	filePath := "../../assets/properties/LawnStrings.txt"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Skipf("Skipping test: LawnStrings.txt not found at %s", filePath)
	}

	ls, err := NewLawnStrings(filePath)
	if err != nil {
		t.Fatalf("Failed to load LawnStrings.txt: %v", err)
	}

	// 验证 Level 1-5 转场对话文本
	daveKeys := []string{
		"CRAZY_DAVE_2410",
		"CRAZY_DAVE_2411",
		"CRAZY_DAVE_2412", // 带 {SHOW_WALLNUT} 标签
		"CRAZY_DAVE_2413",
		"CRAZY_DAVE_2414", // 带 {SHAKE} 标签
		"CRAZY_DAVE_2415",
	}

	for _, key := range daveKeys {
		text := ls.GetString(key)
		// 如果返回 [KEY] 格式，说明键不存在
		if text == "["+key+"]" {
			t.Errorf("Dave dialogue key %q not found in LawnStrings.txt", key)
		} else {
			t.Logf("%s = %q", key, text)
		}
	}
}

// TestLawnStrings_ShovelTutorial 验证铲子教学文本
func TestLawnStrings_ShovelTutorial(t *testing.T) {
	// 测试实际的 LawnStrings.txt 文件
	filePath := "../../assets/properties/LawnStrings.txt"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Skipf("Skipping test: LawnStrings.txt not found at %s", filePath)
	}

	ls, err := NewLawnStrings(filePath)
	if err != nil {
		t.Fatalf("Failed to load LawnStrings.txt: %v", err)
	}

	// 验证铲子教学文本
	shovelKeys := []string{
		"ADVICE_CLICK_SHOVEL",
		"ADVICE_CLICK_PLANT",
		"ADVICE_KEEP_DIGGING",
	}

	for _, key := range shovelKeys {
		text := ls.GetString(key)
		// 如果返回 [KEY] 格式，说明键不存在
		if text == "["+key+"]" {
			t.Errorf("Shovel tutorial key %q not found in LawnStrings.txt", key)
		} else {
			t.Logf("%s = %q", key, text)
		}
	}
}
