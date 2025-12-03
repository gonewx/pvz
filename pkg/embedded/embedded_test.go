package embedded

import (
	"embed"
	"testing"
)

// 测试用的 embed.FS
// 注意：由于 Go embed 指令只能嵌入当前包目录及其子目录的文件，
// 真正的资源嵌入在项目根目录的 embed.go 中。
// 这里我们测试 embedded 包的接口功能，需要在集成测试中验证完整功能。

// TestIsInitialized 测试初始化状态检测
func TestIsInitialized(t *testing.T) {
	// 重置状态
	initialized = false

	if IsInitialized() {
		t.Error("Expected IsInitialized() to return false before Init()")
	}

	// 用空的 embed.FS 初始化
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)

	if !IsInitialized() {
		t.Error("Expected IsInitialized() to return true after Init()")
	}

	// 重置状态以避免影响其他测试
	initialized = false
}

// TestOpenNotInitialized 测试未初始化时调用 Open
func TestOpenNotInitialized(t *testing.T) {
	// 重置状态
	initialized = false

	_, err := Open("assets/test.png")
	if err == nil {
		t.Error("Expected error when calling Open() before Init()")
	}
	if err.Error() != "embedded package not initialized, call Init() first" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestReadFileNotInitialized 测试未初始化时调用 ReadFile
func TestReadFileNotInitialized(t *testing.T) {
	// 重置状态
	initialized = false

	_, err := ReadFile("assets/test.txt")
	if err == nil {
		t.Error("Expected error when calling ReadFile() before Init()")
	}
	if err.Error() != "embedded package not initialized, call Init() first" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestExistsNotInitialized 测试未初始化时调用 Exists
func TestExistsNotInitialized(t *testing.T) {
	// 重置状态
	initialized = false

	// Exists 在未初始化时应返回 false（因为内部调用 Open 会出错）
	if Exists("assets/test.png") {
		t.Error("Expected Exists() to return false before Init()")
	}
}

// TestGlobNotInitialized 测试未初始化时调用 Glob
func TestGlobNotInitialized(t *testing.T) {
	// 重置状态
	initialized = false

	_, err := Glob("assets/*.png")
	if err == nil {
		t.Error("Expected error when calling Glob() before Init()")
	}
	if err.Error() != "embedded package not initialized, call Init() first" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestOpenInvalidPrefix 测试无效路径前缀
func TestOpenInvalidPrefix(t *testing.T) {
	// 用空的 embed.FS 初始化
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
	defer func() { initialized = false }()

	_, err := Open("invalid/path/test.png")
	if err == nil {
		t.Error("Expected error for invalid path prefix")
	}
	if err.Error() != "unknown resource path prefix: invalid/path/test.png (must start with 'assets/' or 'data/')" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestReadFileInvalidPrefix 测试无效路径前缀
func TestReadFileInvalidPrefix(t *testing.T) {
	// 用空的 embed.FS 初始化
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
	defer func() { initialized = false }()

	_, err := ReadFile("invalid/path/test.txt")
	if err == nil {
		t.Error("Expected error for invalid path prefix")
	}
	if err.Error() != "unknown resource path prefix: invalid/path/test.txt (must start with 'assets/' or 'data/')" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestGlobInvalidPrefix 测试无效路径前缀
func TestGlobInvalidPrefix(t *testing.T) {
	// 用空的 embed.FS 初始化
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
	defer func() { initialized = false }()

	_, err := Glob("invalid/*.txt")
	if err == nil {
		t.Error("Expected error for invalid path prefix")
	}
	if err.Error() != "unknown resource path prefix: invalid/*.txt (must start with 'assets/' or 'data/')" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestReadDirNotInitialized 测试未初始化时调用 ReadDir
func TestReadDirNotInitialized(t *testing.T) {
	// 重置状态
	initialized = false

	_, err := ReadDir("assets/images")
	if err == nil {
		t.Error("Expected error when calling ReadDir() before Init()")
	}
	if err.Error() != "embedded package not initialized, call Init() first" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestSubNotInitialized 测试未初始化时调用 Sub
func TestSubNotInitialized(t *testing.T) {
	// 重置状态
	initialized = false

	_, err := Sub("assets/images")
	if err == nil {
		t.Error("Expected error when calling Sub() before Init()")
	}
	if err.Error() != "embedded package not initialized, call Init() first" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestStatNotInitialized 测试未初始化时调用 Stat
func TestStatNotInitialized(t *testing.T) {
	// 重置状态
	initialized = false

	_, err := Stat("assets/test.png")
	if err == nil {
		t.Error("Expected error when calling Stat() before Init()")
	}
	// Stat 内部调用 Open，所以错误信息相同
	if err.Error() != "embedded package not initialized, call Init() first" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestPathNormalization 测试路径规范化
func TestPathNormalization(t *testing.T) {
	// 用空的 embed.FS 初始化
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
	defer func() { initialized = false }()

	// 测试 "./" 前缀被正确移除
	// 注意：由于使用空的 embed.FS，文件不存在会返回错误，
	// 但错误信息应该显示标准化后的路径
	_, err := Open("./assets/test.png")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	// 错误信息应该不包含 "./" 前缀
	errStr := err.Error()
	if errStr == "unknown resource path prefix: ./assets/test.png (must start with 'assets/' or 'data/')" {
		t.Error("Path normalization should remove './' prefix")
	}
}

// TestReadDirInvalidPrefix 测试 ReadDir 无效路径前缀
func TestReadDirInvalidPrefix(t *testing.T) {
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
	defer func() { initialized = false }()

	_, err := ReadDir("invalid/path")
	if err == nil {
		t.Error("Expected error for invalid path prefix")
	}
}

// TestSubInvalidPrefix 测试 Sub 无效路径前缀
func TestSubInvalidPrefix(t *testing.T) {
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
	defer func() { initialized = false }()

	_, err := Sub("invalid/path")
	if err == nil {
		t.Error("Expected error for invalid path prefix")
	}
}

// TestExistsWithValidPrefix 测试 Exists 带有效前缀但文件不存在
func TestExistsWithValidPrefix(t *testing.T) {
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
	defer func() { initialized = false }()

	// 有效前缀但文件不存在
	if Exists("assets/nonexistent.png") {
		t.Error("Expected Exists() to return false for non-existent file")
	}
	if Exists("data/nonexistent.yaml") {
		t.Error("Expected Exists() to return false for non-existent file")
	}
}

// TestOpenAssetsPath 测试 Open assets 路径
func TestOpenAssetsPath(t *testing.T) {
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
	defer func() { initialized = false }()

	// 由于空 FS，应该返回文件不存在错误（而不是前缀错误）
	_, err := Open("assets/test.png")
	if err == nil {
		t.Error("Expected error for non-existent file in empty FS")
	}
	// 确保错误不是前缀错误
	errStr := err.Error()
	if errStr == "unknown resource path prefix: assets/test.png (must start with 'assets/' or 'data/')" {
		t.Error("Should recognize 'assets/' as valid prefix")
	}
}

// TestOpenDataPath 测试 Open data 路径
func TestOpenDataPath(t *testing.T) {
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
	defer func() { initialized = false }()

	// 由于空 FS，应该返回文件不存在错误（而不是前缀错误）
	_, err := Open("data/test.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file in empty FS")
	}
	// 确保错误不是前缀错误
	errStr := err.Error()
	if errStr == "unknown resource path prefix: data/test.yaml (must start with 'assets/' or 'data/')" {
		t.Error("Should recognize 'data/' as valid prefix")
	}
}

// TestReadFileAssetsPath 测试 ReadFile assets 路径
func TestReadFileAssetsPath(t *testing.T) {
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
	defer func() { initialized = false }()

	_, err := ReadFile("assets/test.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	// 确保错误不是前缀错误
	errStr := err.Error()
	if errStr == "unknown resource path prefix: assets/test.txt (must start with 'assets/' or 'data/')" {
		t.Error("Should recognize 'assets/' as valid prefix")
	}
}

// TestReadFileDataPath 测试 ReadFile data 路径
func TestReadFileDataPath(t *testing.T) {
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
	defer func() { initialized = false }()

	_, err := ReadFile("data/test.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	// 确保错误不是前缀错误
	errStr := err.Error()
	if errStr == "unknown resource path prefix: data/test.yaml (must start with 'assets/' or 'data/')" {
		t.Error("Should recognize 'data/' as valid prefix")
	}
}

// TestGlobAssetsPath 测试 Glob assets 路径
func TestGlobAssetsPath(t *testing.T) {
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
	defer func() { initialized = false }()

	// 空 FS 应该返回空结果（而不是错误）
	results, err := Glob("assets/*.png")
	if err != nil {
		// 可能返回错误（取决于 fs.Glob 的实现）
		t.Logf("Glob returned error (expected for empty FS): %v", err)
	} else if len(results) != 0 {
		t.Errorf("Expected empty results for empty FS, got %v", results)
	}
}

// TestGlobDataPath 测试 Glob data 路径
func TestGlobDataPath(t *testing.T) {
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
	defer func() { initialized = false }()

	// 空 FS 应该返回空结果（而不是错误）
	results, err := Glob("data/*.yaml")
	if err != nil {
		// 可能返回错误（取决于 fs.Glob 的实现）
		t.Logf("Glob returned error (expected for empty FS): %v", err)
	} else if len(results) != 0 {
		t.Errorf("Expected empty results for empty FS, got %v", results)
	}
}

// TestReadDirAssetsPath 测试 ReadDir assets 路径
func TestReadDirAssetsPath(t *testing.T) {
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
	defer func() { initialized = false }()

	_, err := ReadDir("assets/images")
	if err == nil {
		// 空 FS 中没有目录，应该返回错误
		t.Log("ReadDir returned nil error for empty FS (might be valid)")
	}
	// 确保错误不是前缀错误
	if err != nil {
		errStr := err.Error()
		if errStr == "unknown resource path prefix: assets/images (must start with 'assets/' or 'data/')" {
			t.Error("Should recognize 'assets/' as valid prefix")
		}
	}
}

// TestReadDirDataPath 测试 ReadDir data 路径
func TestReadDirDataPath(t *testing.T) {
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
	defer func() { initialized = false }()

	_, err := ReadDir("data/levels")
	if err == nil {
		// 空 FS 中没有目录，应该返回错误
		t.Log("ReadDir returned nil error for empty FS (might be valid)")
	}
	// 确保错误不是前缀错误
	if err != nil {
		errStr := err.Error()
		if errStr == "unknown resource path prefix: data/levels (must start with 'assets/' or 'data/')" {
			t.Error("Should recognize 'data/' as valid prefix")
		}
	}
}

// TestSubAssetsPath 测试 Sub assets 路径
func TestSubAssetsPath(t *testing.T) {
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
	defer func() { initialized = false }()

	_, err := Sub("assets/images")
	if err == nil {
		// 空 FS 中没有目录，应该返回错误
		t.Log("Sub returned nil error for empty FS (might be valid)")
	}
	// 确保错误不是前缀错误
	if err != nil {
		errStr := err.Error()
		if errStr == "unknown resource path prefix: assets/images (must start with 'assets/' or 'data/')" {
			t.Error("Should recognize 'assets/' as valid prefix")
		}
	}
}

// TestSubDataPath 测试 Sub data 路径
func TestSubDataPath(t *testing.T) {
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
	defer func() { initialized = false }()

	_, err := Sub("data/levels")
	if err == nil {
		// 空 FS 中没有目录，应该返回错误
		t.Log("Sub returned nil error for empty FS (might be valid)")
	}
	// 确保错误不是前缀错误
	if err != nil {
		errStr := err.Error()
		if errStr == "unknown resource path prefix: data/levels (must start with 'assets/' or 'data/')" {
			t.Error("Should recognize 'data/' as valid prefix")
		}
	}
}

