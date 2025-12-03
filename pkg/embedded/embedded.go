// Package embedded 提供嵌入资源的统一访问接口
//
// 由于 Go embed 指令只能嵌入当前包目录及其子目录的文件，
// embed.FS 变量必须声明在项目根目录（embed.go）。
// 本包提供包装函数，让其他包可以访问嵌入的资源。
//
// 使用前必须调用 Init() 初始化。
package embedded

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

var (
	assetsFS    embed.FS
	dataFS      embed.FS
	initialized bool
)

// Init 初始化 embed.FS 变量
// 必须在 main() 开始时、任何资源加载之前调用
func Init(assets, data embed.FS) {
	assetsFS = assets
	dataFS = data
	initialized = true
}

// IsInitialized 返回 embedded 包是否已初始化
func IsInitialized() bool {
	return initialized
}

// Open 根据路径前缀选择正确的 embed.FS 并打开文件
// 路径必须以 "assets/" 或 "data/" 开头
func Open(path string) (fs.File, error) {
	if !initialized {
		return nil, fmt.Errorf("embedded package not initialized, call Init() first")
	}

	// 标准化路径分隔符为正斜杠（embed.FS 使用正斜杠）
	path = filepath.ToSlash(path)

	// 移除可能的 "./" 前缀
	path = strings.TrimPrefix(path, "./")

	if strings.HasPrefix(path, "assets/") {
		return assetsFS.Open(path)
	} else if strings.HasPrefix(path, "data/") {
		return dataFS.Open(path)
	}
	return nil, fmt.Errorf("unknown resource path prefix: %s (must start with 'assets/' or 'data/')", path)
}

// ReadFile 根据路径前缀选择正确的 embed.FS 并读取文件内容
// 路径必须以 "assets/" 或 "data/" 开头
func ReadFile(path string) ([]byte, error) {
	if !initialized {
		return nil, fmt.Errorf("embedded package not initialized, call Init() first")
	}

	// 标准化路径分隔符为正斜杠
	path = filepath.ToSlash(path)

	// 移除可能的 "./" 前缀
	path = strings.TrimPrefix(path, "./")

	if strings.HasPrefix(path, "assets/") {
		return fs.ReadFile(assetsFS, path)
	} else if strings.HasPrefix(path, "data/") {
		return fs.ReadFile(dataFS, path)
	}
	return nil, fmt.Errorf("unknown resource path prefix: %s (must start with 'assets/' or 'data/')", path)
}

// Exists 检查文件是否存在于 embed.FS 中
func Exists(path string) bool {
	file, err := Open(path)
	if err != nil {
		return false
	}
	file.Close()
	return true
}

// Glob 在 embed.FS 中匹配文件
// 路径模式必须以 "assets/" 或 "data/" 开头
func Glob(pattern string) ([]string, error) {
	if !initialized {
		return nil, fmt.Errorf("embedded package not initialized, call Init() first")
	}

	// 标准化路径分隔符为正斜杠
	pattern = filepath.ToSlash(pattern)

	// 移除可能的 "./" 前缀
	pattern = strings.TrimPrefix(pattern, "./")

	if strings.HasPrefix(pattern, "assets/") {
		return fs.Glob(assetsFS, pattern)
	} else if strings.HasPrefix(pattern, "data/") {
		return fs.Glob(dataFS, pattern)
	}
	return nil, fmt.Errorf("unknown resource path prefix: %s (must start with 'assets/' or 'data/')", pattern)
}

// ReadDir 读取目录内容
// 路径必须以 "assets/" 或 "data/" 开头
func ReadDir(path string) ([]fs.DirEntry, error) {
	if !initialized {
		return nil, fmt.Errorf("embedded package not initialized, call Init() first")
	}

	// 标准化路径分隔符为正斜杠
	path = filepath.ToSlash(path)

	// 移除可能的 "./" 前缀
	path = strings.TrimPrefix(path, "./")

	if strings.HasPrefix(path, "assets/") {
		return fs.ReadDir(assetsFS, path)
	} else if strings.HasPrefix(path, "data/") {
		return fs.ReadDir(dataFS, path)
	}
	return nil, fmt.Errorf("unknown resource path prefix: %s (must start with 'assets/' or 'data/')", path)
}

// Sub 返回指定目录的子文件系统
// 路径必须以 "assets/" 或 "data/" 开头
func Sub(dir string) (fs.FS, error) {
	if !initialized {
		return nil, fmt.Errorf("embedded package not initialized, call Init() first")
	}

	// 标准化路径分隔符为正斜杠
	dir = filepath.ToSlash(dir)

	// 移除可能的 "./" 前缀
	dir = strings.TrimPrefix(dir, "./")

	if strings.HasPrefix(dir, "assets/") {
		return fs.Sub(assetsFS, dir)
	} else if strings.HasPrefix(dir, "data/") {
		return fs.Sub(dataFS, dir)
	}
	return nil, fmt.Errorf("unknown resource path prefix: %s (must start with 'assets/' or 'data/')", dir)
}

// Stat 获取文件信息
// 路径必须以 "assets/" 或 "data/" 开头
func Stat(path string) (fs.FileInfo, error) {
	file, err := Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return file.Stat()
}

